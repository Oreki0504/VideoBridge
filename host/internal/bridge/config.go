package bridge

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	OutputDir         string `json:"outputDir"`
	YTDLPPath         string `json:"ytDlpPath"`
	FFmpegPath        string `json:"ffmpegPath"`
	PreviousYTDLPPath string `json:"previousYtDlpPath,omitempty"`
}
type Tool struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}
type Health struct {
	YTDLPCanRollback bool     `json:"ytdlpCanRollback"`
	Version          string   `json:"version"`
	Capabilities     []string `json:"capabilities"`
	OutputDir        string   `json:"outputDir"`
	Tools            []Tool   `json:"tools"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	f, err := os.ReadFile(path)
	if err != nil {
		return cfg, fail("CONFIG_ERROR", "无法读取 host 配置，请运行安装脚本。")
	}
	if len(f) > 16384 || Decode(f, &cfg) != nil || !filepath.IsAbs(cfg.OutputDir) {
		return cfg, fail("CONFIG_ERROR", "本地下载目录必须是绝对路径。")
	}
	cfg.OutputDir = filepath.Clean(cfg.OutputDir)
	if err := os.MkdirAll(cfg.OutputDir, 0700); err != nil {
		return cfg, fail("OUTPUT_UNAVAILABLE", "无法创建下载目录。")
	}
	info, err := os.Stat(cfg.OutputDir)
	if err != nil || !info.IsDir() {
		return cfg, fail("OUTPUT_UNAVAILABLE", "下载目录不可用。")
	}
	return cfg, nil
}

func saveConfig(configPath string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(configPath), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), configPath)
	}
	return err
}

func findTool(configured, fallback string) string {
	if configured != "" {
		if !filepath.IsAbs(configured) {
			return ""
		}
		info, err := os.Stat(configured)
		if err != nil || !info.Mode().IsRegular() {
			return ""
		}
		return configured
	}
	path, err := exec.LookPath(fallback)
	if err != nil {
		// GUI-launched Chrome does not inherit a user's shell PATH.
		if runtime.GOOS == "darwin" && fallback != "" {
			for _, prefix := range []string{"/opt/homebrew/bin", "/usr/local/bin"} {
				candidate := filepath.Join(prefix, fallback)
				if info, e := os.Stat(candidate); e == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
					return candidate
				}
			}
		}
		return ""
	}
	return path
}
func (c Config) ToolPaths() (string, string, string) {
	y := findTool(c.YTDLPPath, "yt-dlp")
	f := findTool(c.FFmpegPath, "ffmpeg")
	p := ""
	if f != "" {
		suffix := filepath.Ext(f)
		p = findTool(filepath.Join(filepath.Dir(f), "ffprobe"+suffix), "")
	}
	return y, f, p
}
func (c Config) Health() Health {
	y, f, p := c.ToolPaths()
	h := Health{Version: "0.8.0", Capabilities: []string{"x", "gif", "subtitles", "tools.ensure", "generic-sites", "ytdlp.update"}, OutputDir: c.OutputDir, Tools: []Tool{}}
	h.YTDLPCanRollback = findTool(c.PreviousYTDLPPath, "") != ""
	for i, path := range []string{y, f, p} {
		tool := Tool{Name: []string{"yt-dlp", "ffmpeg", "ffprobe"}[i], Path: path}
		if path != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			arg := "-version"
			if i == 0 {
				arg = "--version"
			}
			cmd := exec.CommandContext(ctx, path, arg)
			hideWindow(cmd)
			output := &limitedOutput{limit: 4096}
			cmd.Stdout = output
			cmd.Stderr = output
			if cmd.Run() == nil {
				tool.Available = true
				tool.Version = strings.Split(strings.TrimSpace(output.String()), "\n")[0]
			}
			cancel()
		}
		h.Tools = append(h.Tools, tool)
	}
	return h
}
