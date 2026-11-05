package bridge

import (
	"context"
	"image/gif"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestXRequestAndGIFOptions(t *testing.T) {
	for _, u := range []string{"https://x.com/name/status/123?s=20", "https://twitter.com/i/web/status/123", "https://mobile.twitter.com/name/status/123/video/2"} {
		if _, e := NormalizeURL(u); e != nil {
			t.Fatal(e)
		}
	}
	for _, u := range []string{"https://x.com/home", "https://x.com/name/status/123/video/5", "https://x.com/name/status/abc"} {
		if _, e := NormalizeURL(u); e == nil {
			t.Fatal("accepted", u)
		}
	}
	s := validStart()
	s.URL = "https://x.com/name/status/123/video/2"
	if Validate(&s) == nil {
		t.Fatal("YouTube cookies sent to X")
	}
	s.Cookies[0].Domain = ".x.com"
	s.Cookies[0].Name = "auth_token"
	if e := Validate(&s); e != nil {
		t.Fatal(e)
	}
	if s.URL != "https://x.com/i/web/status/123/video/2" {
		t.Fatal(s.URL)
	}
	s.URL = "https://youtube.com/watch?v=dQw4w9WgXcQ"
	if Validate(&s) == nil {
		t.Fatal("X cookies sent to YouTube")
	}
	for _, o := range []Options{{Format: "gif"}, {Format: "gif", GIFStart: 86400, GIFDuration: 60, GIFWidth: 640, GIFFPS: 20}} {
		if !o.Valid() {
			t.Fatal(o)
		}
	}
	for _, o := range []Options{{Format: "gif", GIFStart: -1}, {Format: "gif", GIFDuration: 61}, {Format: "gif", GIFWidth: 1920}, {Format: "gif", GIFFPS: 60}, {Format: "gif", AudioFormat: "mp3"}, {Format: "best", GIFDuration: 15}} {
		if o.Valid() {
			t.Fatal("accepted", o)
		}
	}
}
func TestRealGIFDownload(t *testing.T) {
	y, f := os.Getenv("VB_INTEGRATION_YTDLP"), os.Getenv("VB_INTEGRATION_FFMPEG")
	if y == "" || f == "" {
		t.Skip("real tools not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	source := filepath.Join(t.TempDir(), "fixture.mp4")
	cmd := exec.CommandContext(ctx, f, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=s=640x360:r=24:d=7", "-c:v", "libx264", "-pix_fmt", "yuv420p", source)
	hideWindow(cmd)
	if b, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("generate %v %s", e, b)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, source) }))
	defer server.Close()
	cfg := Config{OutputDir: t.TempDir(), YTDLPPath: y, FFmpegPath: f}
	root := t.TempDir()
	PrepareCookieRoot(root)
	s := validStart()
	s.URL = server.URL + "/fixture.mp4"
	s.Options = Options{Format: "gif", GIFStart: 1, GIFDuration: 5, GIFWidth: 320, GIFFPS: 12}
	for n := 0; n < 2; n++ {
		files, e := Download(ctx, cfg, root, s, func(map[string]any) {})
		if e != nil {
			t.Fatal(e)
		}
		if len(files) != 1 {
			t.Fatal(files)
		}
		if strings.Contains(filepath.Base(files[0]), "[fixture]") || (n == 1 && !strings.HasSuffix(files[0], " (2).gif")) {
			t.Fatalf("unexpected GIF filename: %s", files[0])
		}
		file, _ := os.Open(files[0])
		g, e := gif.DecodeAll(file)
		file.Close()
		if e != nil {
			t.Fatal(e)
		}
		duration := 0
		for _, d := range g.Delay {
			duration += d
		}
		if g.Config.Width != 320 || g.Config.Height != 180 || len(g.Image) != 60 || duration < 490 || duration > 510 || g.LoopCount != 0 {
			t.Fatalf("GIF width=%d height=%d frames=%d duration=%d loop=%d", g.Config.Width, g.Config.Height, len(g.Image), duration, g.LoopCount)
		}
	}
	entries, _ := os.ReadDir(cfg.OutputDir)
	if len(entries) != 2 {
		t.Fatal("unexpected temporary outputs", entries)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".gif" {
			t.Fatal("source retained")
		}
	}
	cookies, _ := os.ReadDir(root)
	if len(cookies) != 0 {
		t.Fatal("cookies retained")
	}
	cancelCtx, stop := context.WithCancel(context.Background())
	stop()
	if _, e := Download(cancelCtx, cfg, root, s, func(map[string]any) {}); e == nil {
		t.Fatal("cancel ignored")
	}
	entries, _ = os.ReadDir(cfg.OutputDir)
	if len(entries) != 2 {
		t.Fatal("cancel retained source")
	}
	s.Options.GIFStart = 60
	if _, e := Download(ctx, cfg, root, s, func(map[string]any) {}); e == nil {
		t.Fatal("empty GIF accepted")
	}
	entries, _ = os.ReadDir(cfg.OutputDir)
	if len(entries) != 2 {
		t.Fatal("failure retained source")
	}
	t.Log("real GIF: 320x180, 60 frames, 5 seconds; duplicate preservation, cancellation, invalid range and cleanup passed")
}
