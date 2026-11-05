package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Opt-in integration test with real yt-dlp and FFmpeg. The local media is generated
// by the test; no YouTube account is involved. Download is called directly so the
// production Server's HTTPS/YouTube allowlist remains unchanged and fully tested.
func TestRealToolsLocalMedia(t *testing.T) {
	ytdlp, ffmpeg := os.Getenv("VB_INTEGRATION_YTDLP"), os.Getenv("VB_INTEGRATION_FFMPEG")
	if ytdlp == "" || ffmpeg == "" {
		t.Skip("set VB_INTEGRATION_YTDLP and VB_INTEGRATION_FFMPEG to run real tools")
	}
	mediaDir := t.TempDir()
	media := filepath.Join(mediaDir, "fixture.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=24:d=2", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", media)
	hideWindow(cmd)
	data, err := cmd.CombinedOutput()
	cancel()
	if err != nil {
		t.Fatalf("generate media: %v %s", err, data)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/video" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><title>VideoBridge local media</title><video src="/fixture.mp4" controls></video></html>`))
			return
		}
		if r.URL.Path == "/fixture.mp4" {
			http.ServeFile(w, r, media)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	cases := []struct {
		name       string
		options    Options
		ext, codec string
	}{
		{"video", Options{Format: "best"}, ".mp4", "aac"},
		{"legacy-mp3", Options{Format: "audio"}, ".mp3", "mp3"},
		{"original", Options{Format: "audio", AudioFormat: "original"}, ".m4a", "aac"},
		{"mp3-128", Options{Format: "audio", AudioFormat: "mp3", AudioQuality: "128"}, ".mp3", "mp3"},
		{"mp3-320", Options{Format: "audio", AudioFormat: "mp3", AudioQuality: "320"}, ".mp3", "mp3"},
		{"m4a", Options{Format: "audio", AudioFormat: "m4a"}, ".m4a", "aac"},
		{"flac", Options{Format: "audio", AudioFormat: "flac"}, ".flac", "flac"},
		{"wav", Options{Format: "audio", AudioFormat: "wav"}, ".wav", "pcm_s16le"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := PrepareCookieRoot(root); err != nil {
				t.Fatal(err)
			}
			cfg := Config{OutputDir: t.TempDir(), YTDLPPath: ytdlp, FFmpegPath: ffmpeg}
			s := validStart()
			s.URL = server.URL + "/video"
			s.Options = tc.options
			var progress atomic.Int32
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			files, err := Download(ctx, cfg, root, s, func(_ map[string]any) { progress.Add(1) })
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 || progress.Load() == 0 {
				t.Fatalf("files=%v progress=%d", files, progress.Load())
			}
			if strings.Contains(filepath.Base(files[0]), "[video") {
				t.Fatalf("extractor ID retained: %s", files[0])
			}
			if tc.name == "video" {
				again := validStart()
				again.URL = s.URL
				again.Options = tc.options
				repeated, err := Download(ctx, cfg, root, again, func(map[string]any) {})
				want := strings.TrimSuffix(files[0], tc.ext) + " (2)" + tc.ext
				if err != nil || len(repeated) != 1 || repeated[0] != want {
					t.Fatalf("repeat download did not preserve both files: %v %v", repeated, err)
				}
			}
			probePath := filepath.Join(filepath.Dir(ffmpeg), "ffprobe"+filepath.Ext(ffmpeg))
			probe := exec.CommandContext(ctx, probePath, "-v", "error", "-show_entries", "stream=codec_type,codec_name,bit_rate:format=duration", "-of", "json", files[0])
			hideWindow(probe)
			data, err := probe.Output()
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Streams []struct {
					Type    string `json:"codec_type"`
					Codec   string `json:"codec_name"`
					BitRate string `json:"bit_rate"`
				} `json:"streams"`
				Format struct {
					Duration string `json:"duration"`
				} `json:"format"`
			}
			if json.Unmarshal(data, &result) != nil {
				t.Fatal("invalid ffprobe result")
			}
			audio, video := false, false
			for _, s := range result.Streams {
				audio = audio || s.Type == "audio"
				video = video || s.Type == "video"
				if s.Type == "audio" && s.Codec != tc.codec {
					t.Fatalf("expected codec %s: %s", tc.codec, data)
				}
				if s.Type == "audio" && tc.options.AudioQuality != "" && s.BitRate != tc.options.AudioQuality+"000" {
					t.Fatalf("target bitrate not applied: %s", data)
				}
			}
			if !audio || (tc.options.Format == "best" && !video) || (tc.options.Format == "audio" && video) || filepath.Ext(files[0]) != tc.ext {
				t.Fatalf("unexpected media streams: %s", data)
			}
			if tc.name == "original" {
				hashes := func(file string) string {
					t.Helper()
					cmd := exec.CommandContext(ctx, probePath, "-v", "error", "-select_streams", "a:0", "-show_packets", "-show_entries", "packet=data_hash", "-show_data_hash", "sha256", "-of", "csv=p=0", file)
					hideWindow(cmd)
					data, err := cmd.Output()
					if err != nil {
						t.Fatal(err)
					}
					return string(data)
				}
				if hashes(media) != hashes(files[0]) {
					t.Fatal("original audio packet data changed")
				}
			}
			remaining, _ := os.ReadDir(root)
			if len(remaining) != 0 {
				t.Fatal("cookie file not cleaned")
			}
			t.Logf("real yt-dlp + FFmpeg: case=%s duration=%s progressEvents=%d; Cookie cleanup verified", tc.name, result.Format.Duration, progress.Load())
		})
	}
}
