package bridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Config     Config
	ConfigPath string
	CookieRoot string
	mu         sync.Mutex
	activeID   string
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func (s *Server) currentConfig() (Config, error) {
	if s.ConfigPath != "" {
		cfg, err := LoadConfig(s.ConfigPath)
		if err != nil {
			return cfg, err
		}
		s.mu.Lock()
		s.Config = cfg
		s.mu.Unlock()
		return cfg, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Config, nil
}

func (s *Server) Serve(in io.Reader, out io.Writer) error {
	ctx, stop := context.WithCancel(context.Background())
	defer func() {
		stop()
		s.mu.Lock()
		if s.cancel != nil {
			s.cancel()
		}
		s.mu.Unlock()
		s.wg.Wait()
	}()
	w := &Writer{Out: out}
	send := func(id, kind string, payload any) {
		if w.Send(id, kind, payload) != nil {
			stop()
		}
	}
	for {
		data, err := ReadFrame(in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		var req Request
		if Decode(data, &req) != nil || req.Version != 1 || !safeID.MatchString(req.ID) {
			send("", "error", &Failure{"INVALID_REQUEST", "请求协议、版本或 ID 无效。"})
			continue
		}
		switch req.Type {
		case "host.status":
			var empty struct{}
			if Decode(req.Payload, &empty) != nil {
				send(req.ID, "error", &Failure{"INVALID_REQUEST", "状态请求字段无效。"})
				continue
			}
			cfg, err := s.currentConfig()
			if err != nil {
				send(req.ID, "error", asFailure(err))
				continue
			}
			send(req.ID, "host.status", cfg.Health())
		case "host.ytdlp.check", "host.ytdlp.update", "host.ytdlp.rollback":
			var p struct {
				Channel string `json:"channel"`
			}
			if Decode(req.Payload, &p) != nil || updateRepo(p.Channel) == "" || s.ConfigPath == "" {
				send(req.ID, "error", &Failure{"INVALID_REQUEST", "更新通道或配置无效。"})
				continue
			}
			s.mu.Lock()
			if s.activeID != "" {
				s.mu.Unlock()
				send(req.ID, "error", &Failure{"BUSY", "请等待当前下载或工具操作完成。"})
				continue
			}
			duration := 15 * time.Minute
			if req.Type == "host.ytdlp.check" {
				duration = 45 * time.Second
			}
			updateCtx, cancel := context.WithTimeout(ctx, duration)
			s.activeID = "update-" + req.ID
			s.cancel = cancel
			s.mu.Unlock()
			s.wg.Add(1)
			go func(id, channel, action string) {
				defer s.wg.Done()
				defer cancel()
				installer := newToolInstaller(func(message string) { send(id, "tools.progress", map[string]string{"message": message}) })
				cfg, info, err := installer.updateYTDLP(updateCtx, s.ConfigPath, channel, action)
				s.mu.Lock()
				if err == nil {
					s.Config = cfg
				}
				s.activeID = ""
				s.cancel = nil
				s.mu.Unlock()
				if err != nil {
					send(id, "error", asFailure(err))
				} else {
					send(id, "ytdlp.status", YTDLPResult{Update: info, Health: cfg.Health()})
				}
			}(req.ID, p.Channel, strings.TrimPrefix(req.Type, "host.ytdlp."))
		case "host.tools.ensure":
			var empty struct{}
			if Decode(req.Payload, &empty) != nil || s.ConfigPath == "" {
				send(req.ID, "error", &Failure{"INVALID_REQUEST", "依赖安装请求无效。"})
				continue
			}
			s.mu.Lock()
			if s.activeID != "" {
				s.mu.Unlock()
				send(req.ID, "error", &Failure{"BUSY", "已有任务正在执行。"})
				continue
			}
			setupCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			s.activeID = "setup-" + req.ID
			s.cancel = cancel
			s.mu.Unlock()
			s.wg.Add(1)
			go func(id string) {
				defer s.wg.Done()
				defer cancel()
				cfg, err := EnsureTools(setupCtx, s.ConfigPath, func(message string) { send(id, "tools.progress", map[string]string{"message": message}) })
				s.mu.Lock()
				if err == nil {
					s.Config = cfg
				}
				s.activeID = ""
				s.cancel = nil
				s.mu.Unlock()
				if err != nil {
					send(id, "error", asFailure(err))
				} else {
					send(id, "host.status", cfg.Health())
				}
			}(req.ID)
		case "download.start":
			var start Start
			if Decode(req.Payload, &start) != nil {
				send(req.ID, "error", &Failure{"INVALID_REQUEST", "下载请求字段无效。"})
				continue
			}
			if err := Validate(&start); err != nil {
				send(req.ID, "error", asFailure(err))
				continue
			}
			cfg, err := s.currentConfig()
			if err != nil {
				send(req.ID, "error", asFailure(err))
				continue
			}
			s.mu.Lock()
			if s.activeID != "" {
				s.mu.Unlock()
				send(req.ID, "error", &Failure{"BUSY", "已有任务正在执行。"})
				continue
			}
			var entropy [16]byte
			if _, err := rand.Read(entropy[:]); err != nil {
				s.mu.Unlock()
				send(req.ID, "error", &Failure{"INTERNAL_ERROR", "无法创建任务 ID。"})
				continue
			}
			jobID := hex.EncodeToString(entropy[:])
			jobCtx, cancel := context.WithCancel(ctx)
			s.activeID = jobID
			s.cancel = cancel
			s.mu.Unlock()
			send(req.ID, "download.accepted", map[string]any{"jobId": jobID, "url": start.URL})
			s.wg.Add(1)
			go func(id, jobID string, start Start) {
				defer s.wg.Done()
				defer cancel()
				files, err := Download(jobCtx, cfg, s.CookieRoot, start, func(p map[string]any) { p["jobId"] = jobID; send(id, "download.progress", p) })
				s.mu.Lock()
				s.activeID = ""
				s.cancel = nil
				s.mu.Unlock()
				switch {
				case errors.Is(err, context.Canceled):
					send(id, "download.cancelled", map[string]any{"jobId": jobID})
				case err != nil:
					f := asFailure(err)
					send(id, "download.failed", map[string]any{"jobId": jobID, "code": f.Code, "message": f.Message})
				default:
					send(id, "download.completed", map[string]any{"jobId": jobID, "files": files})
				}
			}(req.ID, jobID, start)
		case "download.cancel":
			var p struct {
				JobID string `json:"jobId"`
			}
			if Decode(req.Payload, &p) != nil || !safeID.MatchString(p.JobID) {
				send(req.ID, "error", &Failure{"INVALID_REQUEST", "取消请求无效。"})
				continue
			}
			s.mu.Lock()
			found := s.activeID == p.JobID && s.cancel != nil
			if found {
				s.cancel()
			}
			s.mu.Unlock()
			if !found {
				send(req.ID, "error", &Failure{"JOB_NOT_FOUND", "任务不存在或已结束。"})
			} else {
				send(req.ID, "download.cancelling", map[string]any{"jobId": p.JobID})
			}
		default:
			send(req.ID, "error", &Failure{"INVALID_REQUEST", "未知消息类型。"})
		}
	}
}
