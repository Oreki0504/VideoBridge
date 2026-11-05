package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type limitedOutput struct {
	mu    sync.Mutex
	data  []byte
	limit int
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	left := b.limit - len(b.data)
	if left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}
func (b *limitedOutput) String() string { b.mu.Lock(); defer b.mu.Unlock(); return string(b.data) }

func outputSuffix(o Options) string {
	suffix := ""
	if o.Format == "audio" {
		suffix = " [audio-" + o.effectiveAudioFormat()
		if o.effectiveAudioFormat() == "mp3" {
			suffix += "-" + o.effectiveAudioQuality()
		}
		suffix += "]"
	} else if o.MaxHeight > 0 {
		suffix = " [max-" + strconv.Itoa(o.MaxHeight) + "p]"
	}
	return suffix
}

func BuildArgs(s Start, outputDir, cookiePath, ffmpeg string) []string {
	args := []string{"--ignore-config", "--no-playlist", "--playlist-items", "1", "--no-cache-dir", "--newline", "--progress", "--progress-delta", "1", "--encoding", "utf-8", "--windows-filenames", "--trim-filenames", "180", "--no-simulate", "--no-overwrites",
		// yt-dlp uses bare NA for missing template values, which is invalid JSON.
		"--output-na-placeholder", "null",
		"--concurrent-fragments", strconv.Itoa(s.Options.effectiveConcurrency()),
		"--paths", outputDir, "--output", "%(title).160B [%(id)s]" + outputSuffix(s.Options) + ".%(ext)s", "--ffmpeg-location", filepath.Dir(ffmpeg),
		"--progress-template", `download:VB_PROGRESS {"status":%(progress.status)j,"downloaded_bytes":%(progress.downloaded_bytes)j,"total_bytes":%(progress.total_bytes)j,"total_bytes_estimate":%(progress.total_bytes_estimate)j,"speed":%(progress.speed)j,"eta":%(progress.eta)j}`,
		"--print", `after_move:VB_FILE {"path":%(filepath)j,"id":%(id)j}`}
	if cookiePath != "" {
		args = append(args, "--cookies", cookiePath)
	}
	if s.Options.Format == "subtitles" {
		args = append(args, "--skip-download", "--ignore-no-formats-error", "--write-subs", "--sub-langs", s.Options.subtitleLanguages(),
			"--sub-format", s.Options.subtitleFormat()+"/best", "--convert-subs", s.Options.subtitleFormat(),
			"--print", `after_video:VB_SUBS {"id":%(id)j,"paths":%(requested_subtitles.:.filepath)j}`)
		if s.Options.SubtitleAuto {
			args = append(args, "--write-auto-subs")
		}
	} else if s.Options.Format == "gif" {
		args = append(args, "-f", "bv*[height<=720]/b[height<=720]/bv*/b")
	} else if s.Options.Format == "audio" {
		format := s.Options.effectiveAudioFormat()
		if format == "original" {
			format = "best"
		}
		args = append(args, "-f", "ba/b", "--extract-audio", "--audio-format", format)
		if format == "mp3" {
			quality := s.Options.effectiveAudioQuality()
			if quality == "best" {
				quality = "0"
			} else {
				quality += "K"
			}
			args = append(args, "--audio-quality", quality)
		}
	} else {
		format := "bv*+ba/b"
		if s.Options.MaxHeight > 0 {
			height := strconv.Itoa(s.Options.MaxHeight)
			format = "bv*[height<=" + height + "]+ba/b[height<=" + height + "]"
		}
		args = append(args, "-f", format, "--merge-output-format", "mp4")
	}
	return append(args, "--", s.URL)
}

func Classify(line string) *Failure {
	s := strings.ToLower(line)
	switch {
	case strings.Contains(s, "drm protected"), strings.Contains(s, "drm-protected"), strings.Contains(s, "has drm"):
		return &Failure{"DRM_PROTECTED", "媒体受 DRM 保护，当前下载器不能下载。"}
	case strings.Contains(s, "unsupported url"), strings.Contains(s, "no video formats found"), strings.Contains(s, "no video found"), strings.Contains(s, "does not contain any video"), strings.Contains(s, "no media found"):
		return &Failure{"UNSUPPORTED_MEDIA", "yt-dlp 未识别到可下载的媒体。请打开具体播放页面，或更新 yt-dlp 后重试。"}
	case strings.Contains(s, "po token"), strings.Contains(s, "po_token"), strings.Contains(s, "pot provider"), strings.Contains(s, "signature extraction"), strings.Contains(s, "nsig"), strings.Contains(s, "challenge solving"), strings.Contains(s, "javascript runtime"):
		return &Failure{"EXTRACTOR_TOKEN_REQUIRED", "YouTube 提取或 PO Token / JavaScript 运行时要求未满足；请更新 yt-dlp 并检查其官方运行时与 Token 支持。"}
	case strings.Contains(s, "cookie") && (strings.Contains(s, "invalid") || strings.Contains(s, "expired") || strings.Contains(s, "no longer valid") || strings.Contains(s, "could not") || strings.Contains(s, "failed")):
		return &Failure{"COOKIE_REJECTED", "yt-dlp 报告 Cookie 不可用或已过期，请重新登录当前站点后重试。"}
	case strings.Contains(s, "members only"), strings.Contains(s, "members-only"), strings.Contains(s, "payment information required"):
		return &Failure{"MEMBERSHIP_REQUIRED", "站点未向下载器开放此会员或付费视频。若当前浏览器能完整播放，请重新授权该站点会话后重试；否则需具有该视频的访问权限。"}
	case strings.Contains(s, "sign in"), strings.Contains(s, "login required"), strings.Contains(s, "login is required"), strings.Contains(s, "private video"), strings.Contains(s, "confirm your age"), strings.Contains(s, "authentication"), strings.Contains(s, "not authorized"), strings.Contains(s, "no permission"), strings.Contains(s, "protected tweet"):
		return &Failure{"AUTH_REQUIRED", "站点要求登录、年龄验证或访问权限；Cookie 导出成功不等于具有该视频的权限。"}
	case strings.Contains(s, "ffmpeg"), strings.Contains(s, "ffprobe"):
		return &Failure{"MEDIA_PROCESSING_FAILED", "FFmpeg / FFprobe 媒体处理失败，请检查本地版本与视频格式。"}
	case strings.Contains(s, "timed out"), strings.Contains(s, "connection"), strings.Contains(s, "http error"), strings.Contains(s, "unable to download"), strings.Contains(s, "proxy"), strings.Contains(s, "certificate"), strings.Contains(s, "name resolution"):
		return &Failure{"NETWORK_ERROR", "网络请求失败，请检查网络、代理及站点限制。"}
	case strings.Contains(s, "unavailable"), strings.Contains(s, "removed"), strings.Contains(s, "not available"):
		return &Failure{"VIDEO_UNAVAILABLE", "视频不可用，可能已移除或受到地区限制。"}
	default:
		return nil
	}
}
func rank(code string) int {
	switch code {
	case "DRM_PROTECTED", "UNSUPPORTED_MEDIA":
		return 7
	case "EXTRACTOR_TOKEN_REQUIRED":
		return 6
	case "COOKIE_REJECTED":
		return 5
	case "AUTH_REQUIRED", "MEMBERSHIP_REQUIRED":
		return 4
	case "MEDIA_PROCESSING_FAILED":
		return 3
	case "VIDEO_UNAVAILABLE":
		return 2
	default:
		return 1
	}
}

type outputParser struct {
	mu        sync.Mutex
	pending   []byte
	dropping  bool
	failure   *Failure
	files     []downloadedFile
	outputDir string
	progress  func(map[string]any)
}

func (p *outputParser) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(data)
	for _, b := range data {
		if b == '\n' || b == '\r' {
			if !p.dropping && len(p.pending) > 0 {
				p.line(string(p.pending))
			}
			p.pending = nil
			p.dropping = false
		} else if !p.dropping {
			if len(p.pending) >= 65536 {
				p.pending = nil
				p.dropping = true
			} else {
				p.pending = append(p.pending, b)
			}
		}
	}
	return n, nil
}
func (p *outputParser) finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) > 0 && !p.dropping {
		p.line(string(p.pending))
	}
	p.pending = nil
}
func number(v any) float64 {
	if f, ok := v.(float64); ok && f >= 0 {
		return f
	}
	return 0
}
func (p *outputParser) line(line string) {
	if strings.HasPrefix(line, "VB_SUBS ") {
		p.subtitleFiles(strings.TrimPrefix(line, "VB_SUBS "))
		return
	}
	if strings.HasPrefix(line, "VB_PROGRESS ") {
		var data map[string]any
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "VB_PROGRESS ")), &data) != nil {
			return
		}
		total := number(data["total_bytes"])
		if total == 0 {
			total = number(data["total_bytes_estimate"])
		}
		update := map[string]any{"phase": "下载中", "speed": number(data["speed"]), "eta": number(data["eta"])}
		if total > 0 {
			update["percent"] = min(100, number(data["downloaded_bytes"])/total*100)
		}
		if data["status"] == "finished" {
			update["phase"] = "合并 / 后处理"
		}
		p.progress(update)
		return
	}
	if strings.HasPrefix(line, "VB_FILE ") {
		var file downloadedFile
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "VB_FILE ")), &file) == nil && filepath.IsAbs(file.Path) {
			relative, err := filepath.Rel(p.outputDir, filepath.Clean(file.Path))
			if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative) && len(p.files) < 10 {
				if info, err := os.Lstat(file.Path); err == nil && info.Mode().IsRegular() {
					p.files = append(p.files, file)
				}
			}
		}
		return
	}
	if f := Classify(line); f != nil && (p.failure == nil || rank(f.Code) > rank(p.failure.Code)) {
		p.failure = f
	}
}

func Download(ctx context.Context, cfg Config, root string, s Start, progress func(map[string]any)) ([]string, error) {
	y, f, p := cfg.ToolPaths()
	if y == "" {
		return nil, fail("YTDLP_MISSING", "找不到 yt-dlp，请安装并配置可执行文件路径。")
	}
	if f == "" || p == "" {
		return nil, fail("FFMPEG_MISSING", "找不到 FFmpeg 或同目录的 FFprobe，请安装并配置。")
	}
	downloadDir := cfg.OutputDir
	if s.Options.Format == "gif" || s.Options.Format == "subtitles" {
		var err error
		downloadDir, err = os.MkdirTemp(cfg.OutputDir, ".videobridge-"+s.Options.Format+"-")
		if err != nil {
			return nil, fail("OUTPUT_UNAVAILABLE", "无法创建下载临时目录。")
		}
		defer os.RemoveAll(downloadDir) // Only this generated, task-owned directory.
	}
	var cf *CookieFile
	cookiePath := ""
	if s.Mode == "always" {
		var err error
		cf, err = CreateCookieFile(root, s.Cookies)
		if err != nil {
			return nil, fail("COOKIE_FILE_ERROR", "无法创建受保护的临时 Cookie 文件。")
		}
		cookiePath = cf.Path
	}
	// Never retain the snapshot in task history or diagnostics.
	for i := range s.Cookies {
		s.Cookies[i] = Cookie{}
	}
	s.Cookies = nil
	if s.Options.Format == "subtitles" {
		progress(map[string]any{"phase": "下载字幕"})
	}
	cmd := exec.CommandContext(ctx, y, BuildArgs(s, downloadDir, cookiePath, f)...)
	cmd.WaitDelay = 3 * time.Second
	parser := &outputParser{outputDir: downloadDir, progress: progress}
	cmd.Stdout = parser
	cmd.Stderr = parser
	cmd.Stdin = bytes.NewReader(nil)
	err := runCommand(cmd)
	parser.finish()
	if cf != nil {
		if cleanup := cf.Close(); cleanup != nil {
			return nil, fail("COOKIE_CLEANUP_FAILED", "临时 Cookie 清理失败；下次启动会重试，请检查文件占用。")
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		if parser.failure != nil {
			if parser.failure.Code == "MEMBERSHIP_REQUIRED" && s.Mode == "off" {
				return nil, fail("MEMBERSHIP_REQUIRED", "本次为匿名下载，站点要求会员或付费访问权限。请开启“使用当前站点会话”后重试；仅授予 Cookie 权限不会改变已关闭的会话开关。")
			}
			return nil, parser.failure
		}
		return nil, fail("DOWNLOAD_FAILED", "yt-dlp 执行失败。请检查版本、视频支持情况及本地工具配置。")
	}
	if s.Options.Format == "subtitles" {
		if parser.failure != nil && parser.failure.Code == "SUBTITLE_DOWNLOAD_FAILED" {
			return nil, parser.failure
		}
		return finishSubtitles(parser.files, cfg.OutputDir, s.Options)
	}
	if len(parser.files) == 0 {
		return nil, fail("OUTPUT_NOT_FOUND", "yt-dlp 未返回可验证的输出文件。")
	}
	files := make([]string, 0, len(parser.files))
	for _, file := range parser.files {
		cleaned, err := finishFilename(file, s.Options)
		if err != nil {
			return nil, err
		}
		files = append(files, cleaned)
	}
	if s.Options.Format == "gif" {
		progress(map[string]any{"phase": "转换 GIF（无声音）"})
		file, err := ConvertGIF(ctx, f, files[0], cfg.OutputDir, s.Options)
		if err != nil {
			return nil, err
		}
		return []string{file}, nil
	}
	return files, nil
}

func asFailure(err error) *Failure {
	if f, ok := err.(*Failure); ok {
		return f
	}
	return &Failure{"INTERNAL_ERROR", "本地桥接处理失败。"}
}
