package bridge

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeInstaller(body string) toolInstaller {
	return toolInstaller{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}, progress: func(string) {}}
}
func TestSetupVerificationBoundaries(t *testing.T) {
	for _, u := range []string{"http://github.com/x", "https://github.com.evil.test/x", "https://user@github.com/x", "https://github.com:444/x", "file:///x"} {
		if trustedDownloadURL(u) {
			t.Fatal("untrusted URL accepted", u)
		}
	}
	i := fakeInstaller("binary")
	hash := sha256.Sum256([]byte("binary"))
	digest := hex.EncodeToString(hash[:])
	ctx := context.Background()
	if e := i.verifiedFile(ctx, ffmpegURL, filepath.Join(t.TempDir(), "good"), digest, 100); e != nil {
		t.Fatal(e)
	}
	if e := i.verifiedFile(ctx, ffmpegURL, filepath.Join(t.TempDir(), "bad"), strings.Repeat("0", 64), 100); e == nil {
		t.Fatal("hash mismatch accepted")
	}
	if e := i.fetch(ctx, ffmpegURL, io.Discard, 3); e == nil {
		t.Fatal("oversized body accepted")
	}
	if _, e := checksum([]byte(digest+"  other.exe"), "yt-dlp.exe"); e == nil {
		t.Fatal("wrong asset checksum accepted")
	}
	if _, e := checksum([]byte("xx  yt-dlp.exe"), "yt-dlp.exe"); e == nil {
		t.Fatal("invalid checksum accepted")
	}
}
func TestSetupArchiveBoundary(t *testing.T) {
	for _, tc := range []struct {
		names []string
		valid bool
	}{
		{[]string{"ffmpeg/bin/ffmpeg.exe", "ffmpeg/bin/ffprobe.exe", "ffmpeg/LICENSE"}, true},
		{[]string{"../ffmpeg.exe", "bin/ffprobe.exe"}, false},
		{[]string{"bin/ffmpeg.exe", "other/ffmpeg.exe", "bin/ffprobe.exe"}, false},
		{[]string{"bin/ffmpeg.exe"}, false},
	} {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for _, name := range tc.names {
			f, _ := w.Create(name)
			f.Write([]byte("fixture"))
		}
		w.Close()
		archive := filepath.Join(t.TempDir(), "tools.zip")
		os.WriteFile(archive, b.Bytes(), 0600)
		if e := extractFFmpeg(archive, t.TempDir()); (e == nil) != tc.valid {
			t.Fatalf("names=%v err=%v", tc.names, e)
		}
	}
}
func TestSetupFailurePreservesConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{OutputDir: filepath.Join(dir, "downloads"), YTDLPPath: filepath.Join(dir, "missing.exe"), FFmpegPath: filepath.Join(dir, "also-missing.exe")}
	data, _ := json.Marshal(cfg)
	file := filepath.Join(dir, "config.json")
	os.WriteFile(file, data, 0600)
	if _, e := fakeInstaller(`{"tag_name":"bad","assets":[]}`).ensure(context.Background(), file); e == nil {
		t.Fatal("invalid release accepted")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(data, after) {
		t.Fatal("failure changed config")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "tools"))
	if len(entries) != 0 {
		t.Fatal("staging not cleaned")
	}
	lock, e := lockFile(filepath.Join(dir, ".tools-install.lock"))
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Close()
	if _, e = fakeInstaller("").ensure(context.Background(), file); e == nil {
		t.Fatal("concurrent installation accepted")
	}
}

// Opt-in network test of the installer in an isolated, test-owned directory.
// It never registers or starts a Native Host or changes the user's configuration.
func TestRealDependencyBootstrap(t *testing.T) {
	if os.Getenv("VB_INTEGRATION_SETUP") != "1" {
		t.Skip("set VB_INTEGRATION_SETUP=1 for official downloads")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cfg := Config{OutputDir: filepath.Join(dir, "downloads"), YTDLPPath: filepath.Join(dir, "missing-ytdlp.exe"), FFmpegPath: filepath.Join(dir, "missing-ffmpeg.exe")}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	installed, err := EnsureTools(ctx, configPath, func(message string) { t.Log(message) })
	if err != nil {
		t.Fatal(err)
	}
	if installed.OutputDir != cfg.OutputDir {
		t.Fatal("download directory changed")
	}
	for _, tool := range installed.Health().Tools {
		if !tool.Available {
			t.Fatal("tool unavailable", tool.Name)
		}
		t.Log(tool.Name, tool.Version)
	}
	before, _ := os.ReadFile(configPath)
	// Reuse must be offline and preserve the exact configuration.
	noNetwork := toolInstaller{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("existing tools triggered a download")
		return nil, nil
	})}}
	if _, err := noNetwork.ensure(ctx, configPath); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(configPath)
	if !bytes.Equal(before, after) {
		t.Fatal("reuse rewrote configuration")
	}
}
