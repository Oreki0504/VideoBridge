package bridge

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
const ffmpegURL = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"

type toolInstaller struct {
	client   *http.Client
	progress func(string)
}

func trustedDownloadURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	switch u.Host {
	case "api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "www.gyan.dev", "www.osxexperts.net":
		return true
	}
	return false
}

func (i toolInstaller) fetch(ctx context.Context, raw string, out io.Writer, limit int64) error {
	if !trustedDownloadURL(raw) {
		return fmt.Errorf("untrusted download URL")
	}
	req, e := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if e != nil {
		return e
	}
	req.Header.Set("User-Agent", "VideoBridge dependency manager")
	resp, e := i.client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > limit {
		return fmt.Errorf("download status/size rejected")
	}
	n, e := io.Copy(out, io.LimitReader(resp.Body, limit+1))
	if e != nil {
		return e
	}
	if n > limit {
		return fmt.Errorf("download too large")
	}
	return nil
}
func (i toolInstaller) small(ctx context.Context, raw string) ([]byte, error) {
	var b strings.Builder
	e := i.fetch(ctx, raw, &b, 2<<20)
	return []byte(b.String()), e
}
func checksum(manifest []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 1 && filename == "" || len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == filename {
			b, e := hex.DecodeString(fields[0])
			if e == nil && len(b) == 32 {
				return strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", fmt.Errorf("missing SHA256")
}
func (i toolInstaller) verifiedFile(ctx context.Context, raw, dest, digest string, limit int64) error {
	f, e := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	h := sha256.New()
	e = i.fetch(ctx, raw, io.MultiWriter(f, h), limit)
	closeErr := f.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return fmt.Errorf("SHA256 mismatch")
	}
	return nil
}
func (i toolInstaller) ytdlp(ctx context.Context, stage string) (string, error) {
	i.progress("查询 yt-dlp 官方版本…")
	release, e := i.latestYTDLP(ctx, "stable")
	if e != nil {
		return "", e
	}
	return i.downloadYTDLP(ctx, stage, release)
}

func (i toolInstaller) downloadYTDLP(ctx context.Context, stage string, release ytdlpRelease) (string, error) {
	data, e := i.small(ctx, release.Checksums)
	if e != nil {
		return "", e
	}
	digest, e := checksum(data, ytdlpAsset(runtime.GOOS, runtime.GOARCH))
	if e != nil {
		return "", e
	}
	i.progress("下载并校验 yt-dlp " + release.Version + "…")
	dest := filepath.Join(stage, toolName("yt-dlp"))
	e = i.verifiedFile(ctx, release.Binary, dest, digest, 64<<20)
	if e == nil {
		e = os.Chmod(dest, 0700)
	}
	return dest, e
}

func extractFFmpeg(archive, dest string) error {
	r, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer r.Close()
	found := map[string]bool{}
	for _, f := range r.File {
		name := path.Base(f.Name)
		if name != "ffmpeg.exe" && name != "ffprobe.exe" && name != "LICENSE" {
			continue
		}
		// Never use an archive path as a destination. Also reject malformed entries.
		if strings.Contains(f.Name, "\\") || strings.HasPrefix(f.Name, "/") || strings.Contains("/"+f.Name+"/", "/../") || f.Mode()&os.ModeType != 0 || found[name] || f.UncompressedSize64 > 300<<20 {
			return fmt.Errorf("unsafe archive entry")
		}
		found[name] = true
		src, e := f.Open()
		if e != nil {
			return e
		}
		out, e := os.OpenFile(filepath.Join(dest, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
		if e != nil {
			src.Close()
			return e
		}
		n, e := io.Copy(out, io.LimitReader(src, (300<<20)+1))
		src.Close()
		ce := out.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		if n > 300<<20 {
			return fmt.Errorf("archive entry too large")
		}
	}
	if !found["ffmpeg.exe"] || !found["ffprobe.exe"] {
		return fmt.Errorf("FFmpeg or FFprobe missing from archive")
	}
	return nil
}
func (i toolInstaller) ffmpeg(ctx context.Context, stage string) (string, error) {
	if runtime.GOOS == "darwin" {
		return i.macFFmpeg(ctx, stage, runtime.GOARCH)
	}
	i.progress("获取 FFmpeg 校验值（FFmpeg.org 推荐的 Gyan 构建）…")
	data, e := i.small(ctx, ffmpegURL+".sha256")
	if e != nil {
		return "", e
	}
	digest, e := checksum(data, "")
	if e != nil {
		digest, e = checksum(data, "ffmpeg-release-essentials.zip")
	}
	if e != nil {
		return "", e
	}
	i.progress("下载并校验 FFmpeg / FFprobe（约 115 MB）…")
	archive := filepath.Join(stage, "ffmpeg.zip")
	if e = i.verifiedFile(ctx, ffmpegURL, archive, digest, 350<<20); e != nil {
		return "", e
	}
	i.progress("解压 FFmpeg / FFprobe…")
	e = extractFFmpeg(archive, stage)
	os.Remove(archive)
	return filepath.Join(stage, "ffmpeg.exe"), e
}

// EnsureTools only installs missing/broken dependencies. It never accepts a
// download URL from the browser, updates PATH, or replaces an existing tool.
func EnsureTools(ctx context.Context, configPath string, progress func(string)) (Config, error) {
	return newToolInstaller(progress).ensure(ctx, configPath)
}

func newToolInstaller(progress func(string)) toolInstaller {
	if progress == nil {
		progress = func(string) {}
	}
	client := &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 8 || !trustedDownloadURL(req.URL.String()) {
			return fmt.Errorf("redirect rejected")
		}
		return nil
	}}
	return toolInstaller{client, progress}
}
func (i toolInstaller) ensure(ctx context.Context, configPath string) (Config, error) {
	if i.progress == nil {
		i.progress = func(string) {}
	}
	var cfg Config
	lock, e := lockFile(filepath.Join(filepath.Dir(configPath), ".tools-install.lock"))
	if e != nil {
		return cfg, fail("SETUP_BUSY", "另一个安装进程正在运行，请稍后重试。")
	}
	defer lock.Close()
	cfg, e = LoadConfig(configPath)
	if e != nil {
		return cfg, e
	}
	health := cfg.Health()
	missingY := !health.Tools[0].Available
	missingF := !health.Tools[1].Available || !health.Tools[2].Available
	if !missingY && !missingF {
		return cfg, nil
	}
	if !toolsSupported() {
		return cfg, fail("SETUP_UNSUPPORTED", "自动安装目前支持 Windows x64 和 macOS（Apple Silicon / Intel）。")
	}
	root := filepath.Join(filepath.Dir(configPath), "tools")
	if e = os.MkdirAll(root, 0700); e != nil {
		return cfg, fail("SETUP_FAILED", "无法创建工具目录。")
	}
	stage, e := os.MkdirTemp(root, ".setup-")
	if e != nil {
		return cfg, fail("SETUP_FAILED", "无法创建安装临时目录。")
	}
	defer os.RemoveAll(stage) // This operation owns only this generated directory.
	next := cfg
	if missingY {
		next.YTDLPPath, e = i.ytdlp(ctx, stage)
	}
	if e == nil && missingF {
		next.FFmpegPath, e = i.ffmpeg(ctx, stage)
	}
	if e != nil {
		return cfg, fail("SETUP_FAILED", "依赖下载或 SHA256 校验失败，原配置已保留。请检查网络后点“检测并补齐依赖”重试，或手动配置工具。")
	}
	i.progress("检查已下载的工具…")
	for _, t := range next.Health().Tools {
		if !t.Available {
			return cfg, fail("SETUP_FAILED", "已下载的工具无法运行，原配置已保留。")
		}
	}
	if ctx.Err() != nil {
		return cfg, ctx.Err()
	}
	final := filepath.Join(root, "bundle-"+strings.TrimPrefix(filepath.Base(stage), ".setup-"))
	if e = os.Rename(stage, final); e != nil {
		return cfg, fail("SETUP_FAILED", "无法保存工具目录。")
	}
	committed := false
	defer func() {
		if !committed {
			os.RemoveAll(final)
		}
	}()
	if missingY {
		next.YTDLPPath = filepath.Join(final, toolName("yt-dlp"))
	}
	if missingF {
		next.FFmpegPath = filepath.Join(final, toolName("ffmpeg"))
	}
	if e = saveConfig(configPath, next); e != nil {
		return cfg, fail("SETUP_FAILED", "无法保存工具配置，原配置已保留。")
	}
	committed = true
	i.progress("本地依赖已就绪。")
	return next, nil
}
