package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type YTDLPUpdate struct {
	Channel         string `json:"channel"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	CanRollback     bool   `json:"canRollback"`
	Supported       bool   `json:"supported"`
	Message         string `json:"message"`
}

type YTDLPResult struct {
	Update YTDLPUpdate `json:"update"`
	Health Health      `json:"health"`
}

type ytdlpRelease struct{ Version, Binary, Checksums string }

var releaseVersion = regexp.MustCompile(`^[0-9]{4}\.[0-9]{2}\.[0-9]{2}(?:\.[0-9]{6})?$`)

func updateRepo(channel string) string {
	switch channel {
	case "stable":
		return "yt-dlp/yt-dlp"
	case "nightly":
		return "yt-dlp/yt-dlp-nightly-builds"
	}
	return ""
}

func versionKey(version string) string {
	if !releaseVersion.MatchString(version) {
		return ""
	}
	if len(version) == 10 {
		return version + ".000000"
	}
	return version
}

func (i toolInstaller) latestYTDLP(ctx context.Context, channel string) (ytdlpRelease, error) {
	var result ytdlpRelease
	repo := updateRepo(channel)
	if repo == "" {
		return result, fmt.Errorf("invalid channel")
	}
	data, err := i.small(ctx, "https://api.github.com/repos/"+repo+"/releases/latest")
	if err != nil {
		return result, err
	}
	var release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if json.Unmarshal(data, &release) != nil || versionKey(release.Tag) == "" || release.Draft || (channel == "stable" && release.Prerelease) {
		return result, fmt.Errorf("invalid release metadata")
	}
	result.Version = release.Tag
	base := "https://github.com/" + repo + "/releases/download/" + release.Tag + "/"
	for _, asset := range release.Assets {
		if asset.URL != base+asset.Name {
			continue
		}
		switch asset.Name {
		case ytdlpAsset(runtime.GOOS, runtime.GOARCH):
			result.Binary = asset.URL
		case "SHA2-256SUMS":
			result.Checksums = asset.URL
		}
	}
	if result.Binary == "" || result.Checksums == "" {
		return result, fmt.Errorf("official assets missing")
	}
	return result, nil
}

func ytdlpVersion(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--ignore-config", "--version")
	hideWindow(cmd)
	output := &limitedOutput{limit: 1024}
	cmd.Stdout = output
	cmd.Stderr = &limitedOutput{limit: 1024}
	if cmd.Run() != nil {
		return ""
	}
	return strings.TrimSpace(output.String())
}

func localUpdateInfo(ctx context.Context, cfg Config, channel string) YTDLPUpdate {
	y, _, _ := cfg.ToolPaths()
	return YTDLPUpdate{Channel: channel, CurrentVersion: ytdlpVersion(ctx, y), CanRollback: findTool(cfg.PreviousYTDLPPath, "") != "", Supported: toolsSupported()}
}

func (i toolInstaller) checkYTDLP(ctx context.Context, cfg Config, channel string) (YTDLPUpdate, ytdlpRelease, error) {
	info := localUpdateInfo(ctx, cfg, channel)
	release, err := i.latestYTDLP(ctx, channel)
	if err != nil {
		return info, release, fail("UPDATE_CHECK_FAILED", "无法查询 yt-dlp 官方版本。请检查网络或稍后重试；当前工具未改动。")
	}
	info.LatestVersion = release.Version
	current := versionKey(info.CurrentVersion)
	switch {
	case !info.Supported:
		info.Message = "自动更新目前支持 Windows x64 和 macOS（Apple Silicon / Intel）。"
	case info.CurrentVersion != "" && current == "":
		info.Message = "无法比较当前自定义版本，请通过原安装方式更新。"
	case current > versionKey(release.Version):
		info.Message = "当前版本比此通道的最新版更新，保持当前版本。"
	case current == versionKey(release.Version):
		info.Message = "已是此通道最新版本。"
	default:
		info.UpdateAvailable = true
		info.Message = "发现新版本，可更新。"
	}
	return info, release, nil
}

// Versioned managed copies preserve package-manager installations and running
// download processes. Only config.json switches after verification succeeds.
func (i toolInstaller) updateYTDLP(ctx context.Context, configPath, channel, action string) (Config, YTDLPUpdate, error) {
	var cfg Config
	var info YTDLPUpdate
	if updateRepo(channel) == "" || (action != "check" && action != "update" && action != "rollback") {
		return cfg, info, fail("INVALID_REQUEST", "更新操作或通道无效。")
	}
	lock, err := lockFile(filepath.Join(filepath.Dir(configPath), ".tools-install.lock"))
	if err != nil {
		return cfg, info, fail("SETUP_BUSY", "另一个工具安装或更新正在进行，请稍后重试。")
	}
	defer lock.Close()
	cfg, err = LoadConfig(configPath)
	if err != nil {
		return cfg, info, err
	}
	if action == "rollback" {
		info = localUpdateInfo(ctx, cfg, channel)
		previous := findTool(cfg.PreviousYTDLPPath, "")
		if previous == "" || ytdlpVersion(ctx, previous) == "" {
			return cfg, info, fail("ROLLBACK_UNAVAILABLE", "上一版本不存在或无法运行，当前配置已保留。")
		}
		next := cfg
		next.YTDLPPath = previous
		next.PreviousYTDLPPath = ""
		if ctx.Err() != nil {
			return cfg, info, ctx.Err()
		}
		if err = saveConfig(configPath, next); err != nil {
			return cfg, info, fail("UPDATE_FAILED", "恢复上一版本失败，当前配置已保留。")
		}
		info = localUpdateInfo(ctx, next, channel)
		info.Message = "已恢复上一版本。"
		return next, info, nil
	}
	info, release, err := i.checkYTDLP(ctx, cfg, channel)
	if err != nil || action == "check" || !info.UpdateAvailable {
		return cfg, info, err
	}
	root := filepath.Join(filepath.Dir(configPath), "tools")
	if err = os.MkdirAll(root, 0700); err != nil {
		return cfg, info, fail("UPDATE_FAILED", "无法创建工具目录，当前配置已保留。")
	}
	stage, err := os.MkdirTemp(root, ".update-")
	if err != nil {
		return cfg, info, fail("UPDATE_FAILED", "无法创建更新临时目录，当前配置已保留。")
	}
	defer os.RemoveAll(stage) // Only this generated directory is owned by this operation.
	candidate, err := i.downloadYTDLP(ctx, stage, release)
	if err != nil {
		return cfg, info, fail("UPDATE_FAILED", "yt-dlp 下载或 SHA256 校验失败，当前版本已保留。")
	}
	i.progress("验证新版本能否运行…")
	if ytdlpVersion(ctx, candidate) != release.Version {
		return cfg, info, fail("UPDATE_FAILED", "新工具无法运行或版本不符，当前版本已保留。")
	}
	if ctx.Err() != nil {
		return cfg, info, ctx.Err()
	}
	final := filepath.Join(root, "ytdlp-"+release.Version+"-"+strings.TrimPrefix(filepath.Base(stage), ".update-"))
	if err = os.Rename(stage, final); err != nil {
		return cfg, info, fail("UPDATE_FAILED", "无法保存新工具，当前版本已保留。")
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(final)
		}
	}()
	next := cfg
	next.PreviousYTDLPPath = findTool(cfg.YTDLPPath, "yt-dlp")
	next.YTDLPPath = filepath.Join(final, toolName("yt-dlp"))
	if err = saveConfig(configPath, next); err != nil {
		return cfg, info, fail("UPDATE_FAILED", "无法切换到新版本，当前配置已保留。")
	}
	committed = true
	info = localUpdateInfo(ctx, next, channel)
	info.LatestVersion = release.Version
	info.Message = "yt-dlp 已更新；如遇兼容问题，可恢复上一版本。"
	return next, info, nil
}
