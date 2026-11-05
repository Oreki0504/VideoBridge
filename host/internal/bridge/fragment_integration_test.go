package bridge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Verifies actual simultaneous HTTP fragment requests, not just argument assembly.
// The adapter is called directly with a local generated source; production URL
// validation remains YouTube-only.
func TestRealFragmentConcurrency(t *testing.T) {
	ytdlp, ffmpeg := os.Getenv("VB_INTEGRATION_YTDLP"), os.Getenv("VB_INTEGRATION_FFMPEG")
	if ytdlp == "" || ffmpeg == "" {
		t.Skip("set real tool paths to test fragment concurrency")
	}
	mediaDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	generate := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=24:d=4", "-f", "lavfi", "-i", "sine=frequency=440:duration=4", "-c:v", "libx264", "-g", "12", "-sc_threshold", "0", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest", "-f", "hls", "-hls_time", "0.5", "-hls_list_size", "0", "-hls_segment_filename", filepath.Join(mediaDir, "segment-%03d.ts"), filepath.Join(mediaDir, "index.m3u8"))
	hideWindow(generate)
	data, err := generate.CombinedOutput()
	cancel()
	if err != nil {
		t.Fatalf("HLS generation: %v %s", err, data)
	}
	for _, n := range []int{1, 4, 8, 16} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			var active, peak, requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, ".ts") {
					current := active.Add(1)
					defer active.Add(-1)
					requests.Add(1)
					for {
						previous := peak.Load()
						if current <= previous || peak.CompareAndSwap(previous, current) {
							break
						}
					}
					// A short deterministic transport delay makes overlapping requests observable.
					time.Sleep(150 * time.Millisecond)
				}
				http.FileServer(http.Dir(mediaDir)).ServeHTTP(w, r)
			}))
			defer server.Close()
			cfg := Config{OutputDir: t.TempDir(), YTDLPPath: ytdlp, FFmpegPath: ffmpeg}
			root := t.TempDir()
			if err := PrepareCookieRoot(root); err != nil {
				t.Fatal(err)
			}
			s := validStart()
			s.URL = server.URL + "/index.m3u8"
			s.Options = Options{Format: "best", ConcurrentFragments: n}
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			files, err := Download(ctx, cfg, root, s, func(map[string]any) {})
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 || requests.Load() < 4 {
				t.Fatalf("missing HLS download: files=%v requests=%d", files, requests.Load())
			}
			if peak.Load() > int32(n) || (n == 1 && peak.Load() != 1) || (n > 1 && peak.Load() < 2) {
				t.Fatalf("concurrency %d produced peak %d", n, peak.Load())
			}
			decode := exec.CommandContext(ctx, ffmpeg, "-v", "error", "-i", files[0], "-f", "null", "-")
			hideWindow(decode)
			if data, err := decode.CombinedOutput(); err != nil {
				t.Fatalf("output decoding: %v %s", err, data)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 0 {
				t.Fatal("Cookie task not cleaned")
			}
			t.Logf("configured=%d observed simultaneous fragment requests=%d total=%d; full decode and Cookie cleanup passed", n, peak.Load(), requests.Load())
		})
	}
}
