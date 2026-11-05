package bridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUpdateVersionsAndOfficialAssets(t *testing.T) {
	if versionKey("2026.10.01.232000") <= versionKey("2026.10.01") || versionKey("2026.10.02") <= versionKey("2026.10.01.232000") {
		t.Fatal("version order")
	}
	for _, v := range []string{"latest", "v2026.10.01", "2026.1.1", "../../x", "2026.10.01?x"} {
		if versionKey(v) != "" {
			t.Fatal(v)
		}
	}
	for _, channel := range []string{"stable", "nightly"} {
		i := releaseFixture(t, channel, "2026.11.04", []byte("binary"), false)
		r, err := i.latestYTDLP(context.Background(), channel)
		if err != nil || !strings.Contains(r.Binary, "/"+updateRepo(channel)+"/") {
			t.Fatal(r, err)
		}
	}
	for _, body := range []string{`{"tag_name":"../../bad"}`, `{"tag_name":"2026.11.04","draft":true}`, `{"tag_name":"2026.11.04","assets":[{"name":"yt-dlp.exe","browser_download_url":"https://github.com/other/repo/releases/download/2026.11.04/yt-dlp.exe"}]}`} {
		if _, err := fakeInstaller(body).latestYTDLP(context.Background(), "stable"); err == nil {
			t.Fatal("unsafe metadata accepted")
		}
	}
}

func releaseFixture(t *testing.T, channel, version string, binary []byte, badHash bool) toolInstaller {
	t.Helper()
	base := "https://github.com/" + updateRepo(channel) + "/releases/download/" + version + "/"
	asset := ytdlpAsset(runtime.GOOS, runtime.GOARCH)
	hash := sha256.Sum256(binary)
	digest := hex.EncodeToString(hash[:])
	if badHash {
		digest = strings.Repeat("0", 64)
	}
	metadata, _ := json.Marshal(map[string]any{"tag_name": version, "assets": []map[string]string{{"name": asset, "browser_download_url": base + asset}, {"name": "SHA2-256SUMS", "browser_download_url": base + "SHA2-256SUMS"}}})
	return toolInstaller{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var data []byte
		switch r.URL.String() {
		case "https://api.github.com/repos/" + updateRepo(channel) + "/releases/latest":
			data = metadata
		case base + "SHA2-256SUMS":
			data = []byte(digest + "  " + asset + "\n")
		case base + asset:
			data = binary
		default:
			t.Errorf("unexpected request %s", r.URL)
			return nil, fmt.Errorf("unexpected request")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header)}, nil
	})}, progress: func(string) {}}
}

func buildVersionFixture(t *testing.T, version string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "yt-dlp.exe")
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-ldflags", "-X main.fixtureVersion="+version, "-o", p, "./testdata/fake-tool")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture build %v %s", err, data)
	}
	return p
}

func TestYTDLPUpdateAndRollback(t *testing.T) {
	if !toolsSupported() {
		t.Skip("unsupported updater platform")
	}
	old := buildVersionFixture(t, "2026.10.01")
	newTool := buildVersionFixture(t, "2026.11.04")
	binary, err := os.ReadFile(newTool)
	if err != nil {
		t.Fatal(err)
	}
	oldBytes, _ := os.ReadFile(old)
	for _, mode := range []string{"success", "checksum", "version-mismatch", "network", "cancel", "newer-current", "unknown-current", "locked"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.json")
			cfg := Config{OutputDir: filepath.Join(dir, "downloads"), YTDLPPath: old, FFmpegPath: newTool}
			if mode == "newer-current" {
				cfg.YTDLPPath = newTool
			}
			if mode == "unknown-current" {
				cfg.YTDLPPath = buildFixture(t)
			}
			if err := saveConfig(configPath, cfg); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(configPath)
			version := "2026.11.04"
			if mode == "version-mismatch" {
				version = "2026.11.05"
			}
			if mode == "newer-current" {
				version = "2026.10.01"
			}
			i := releaseFixture(t, "stable", version, binary, mode == "checksum")
			if mode == "network" {
				i = fakeInstaller("invalid response")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if mode == "cancel" {
				cancel()
			}
			if mode == "locked" {
				lock, err := lockFile(filepath.Join(dir, ".tools-install.lock"))
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Close()
			}
			next, info, err := i.updateYTDLP(ctx, configPath, "stable", "update")
			if mode != "success" {
				if mode == "newer-current" || mode == "unknown-current" {
					if err != nil || info.UpdateAvailable {
						t.Fatal(info, err)
					}
				} else if err == nil {
					t.Fatal("failure accepted")
				}
				after, _ := os.ReadFile(configPath)
				if !bytes.Equal(before, after) {
					t.Fatal("failed/skipped update changed config")
				}
				entries, _ := os.ReadDir(filepath.Join(dir, "tools"))
				if len(entries) != 0 {
					t.Fatal("failed update left files")
				}
				return
			}
			if err != nil || info.CurrentVersion != "2026.11.04" || info.UpdateAvailable || !info.CanRollback {
				t.Fatal(info, err)
			}
			if next.OutputDir != cfg.OutputDir || next.FFmpegPath != cfg.FFmpegPath || next.PreviousYTDLPPath != old || !strings.HasPrefix(next.YTDLPPath, filepath.Join(dir, "tools")+string(os.PathSeparator)) {
				t.Fatal("bad config transition")
			}
			after, _ := os.ReadFile(old)
			if !bytes.Equal(oldBytes, after) {
				t.Fatal("external installation modified")
			}
			checkBefore, _ := os.ReadFile(configPath)
			_, check, err := i.updateYTDLP(ctx, configPath, "stable", "check")
			if err != nil || check.UpdateAvailable {
				t.Fatal(check, err)
			}
			checkAfter, _ := os.ReadFile(configPath)
			if !bytes.Equal(checkBefore, checkAfter) {
				t.Fatal("check mutated config")
			}
			// Rollback must work offline and must not remove either binary.
			if !next.Health().YTDLPCanRollback {
				t.Fatal("offline health did not expose rollback")
			}
			i.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("rollback used network"); return nil, nil })}
			restored, info, err := i.updateYTDLP(ctx, configPath, "stable", "rollback")
			if err != nil || restored.YTDLPPath != old || info.CurrentVersion != "2026.10.01" || info.CanRollback {
				t.Fatal(restored, info, err)
			}
			if _, err := os.Stat(next.YTDLPPath); err != nil {
				t.Fatal("rollback deleted new binary")
			}
			if _, _, err := i.updateYTDLP(ctx, configPath, "stable", "rollback"); err == nil {
				t.Fatal("rollback without previous accepted")
			}
		})
	}
}

func TestRealYTDLPUpdate(t *testing.T) {
	if os.Getenv("VB_INTEGRATION_UPDATE") != "1" {
		t.Skip("opt-in official yt-dlp download")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "config.json")
	cfg := Config{OutputDir: filepath.Join(dir, "downloads"), YTDLPPath: buildVersionFixture(t, "2000.01.01"), FFmpegPath: os.Getenv("VB_INTEGRATION_FFMPEG")}
	if err := saveConfig(file, cfg); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	i := newToolInstaller(func(message string) { t.Log(message) })
	next, info, err := i.updateYTDLP(ctx, file, "stable", "update")
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentVersion != info.LatestVersion || !info.CanRollback || next.YTDLPPath == cfg.YTDLPPath {
		t.Fatal(info)
	}
	t.Logf("Official update verified: %s; old executable retained", info.CurrentVersion)
	_, restored, err := i.updateYTDLP(ctx, file, "stable", "rollback")
	if err != nil || restored.CurrentVersion != "2000.01.01" {
		t.Fatal(restored, err)
	}
}

func TestRealYTDLPReleaseCheck(t *testing.T) {
	if os.Getenv("VB_INTEGRATION_UPDATE_CHECK") != "1" {
		t.Skip("opt-in official release checks")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	i := newToolInstaller(nil)
	for _, channel := range []string{"stable", "nightly"} {
		release, err := i.latestYTDLP(ctx, channel)
		if err != nil {
			t.Fatalf("%s official release check: %v", channel, err)
		}
		t.Logf("%s: %s", channel, release.Version)
	}
}
