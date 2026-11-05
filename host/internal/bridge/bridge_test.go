package bridge

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func validStart() Start {
	return Start{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=discard", StoreID: "1", Mode: "always", Options: Options{Format: "best"}, Cookies: []Cookie{{Name: "SID", Value: "synthetic-secret-never-log", Domain: ".youtube.com", Path: "/", Secure: true, HTTPOnly: true, Session: true}}}
}

func TestNativeFrames(t *testing.T) {
	var buffer bytes.Buffer
	w := Writer{Out: &buffer}
	if err := w.Send("test", "host.status", map[string]string{"message": "中文"}); err != nil {
		t.Fatal(err)
	}
	data, err := ReadFrame(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	var event Event
	if json.Unmarshal(data, &event) != nil || event.ID != "test" {
		t.Fatal("frame did not round trip")
	}
	for _, size := range []uint32{0, MaxFrame + 1} {
		var h [4]byte
		binary.NativeEndian.PutUint32(h[:], size)
		if _, err := ReadFrame(bytes.NewReader(h[:])); err == nil {
			t.Fatal("unbounded frame accepted")
		}
	}
	if _, err := ReadFrame(bytes.NewReader([]byte{2, 0, 0, 0, 1})); err == nil {
		t.Fatal("truncated frame accepted")
	}
	var r Request
	if Decode([]byte(`{"version":1,"unknown":true}`), &r) == nil {
		t.Fatal("unknown fields accepted")
	}
	if Decode([]byte(`{} {}`), &r) == nil {
		t.Fatal("trailing JSON accepted")
	}
}
func TestURLAndRequestValidation(t *testing.T) {
	for _, raw := range []string{"https://youtu.be/dQw4w9WgXcQ", "https://m.youtube.com/shorts/dQw4w9WgXcQ", "https://youtube.com/live/dQw4w9WgXcQ"} {
		if _, err := NormalizeURL(raw); err != nil {
			t.Fatal(raw, err)
		}
	}
	for _, raw := range []string{"http://youtube.com/watch?v=dQw4w9WgXcQ", "https://user@youtube.com/watch?v=dQw4w9WgXcQ", "https://youtube.com:443/watch?v=dQw4w9WgXcQ", "https://youtube.com/playlist?list=x", "file:///C:/secret", "https://youtu.be/invalid"} {
		if _, err := NormalizeURL(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
	cases := map[string]func(*Start){
		"foreign cookie":     func(s *Start) { s.Cookies[0].Domain = ".google.com" },
		"cookie injection":   func(s *Start) { s.Cookies[0].Value = "abc\n.evil.com\tTRUE" },
		"host-only mismatch": func(s *Start) { s.Cookies[0].HostOnly = true },
		"bad options":        func(s *Start) { s.Options.Format = "--exec" },
		"missing store":      func(s *Start) { s.StoreID = "" },
		"expired auth":       func(s *Start) { s.Cookies[0].Session = false; s.Cookies[0].ExpirationDate = 1 },
		"anonymous cookies":  func(s *Start) { s.Mode = "off" },
		"duplicate":          func(s *Start) { s.Cookies = append(s.Cookies, s.Cookies[0]) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validStart()
			mutate(&s)
			if Validate(&s) == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
	s := validStart()
	if err := Validate(&s); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s.URL, "list") {
		t.Fatal("playlist not stripped")
	}
	s.Mode = "off"
	s.Cookies = nil
	if err := Validate(&s); err != nil {
		t.Fatal(err)
	}
}
func TestNetscapeAndCookieLifecycle(t *testing.T) {
	root := t.TempDir()
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	cookies := validStart().Cookies
	cookies = append(cookies, Cookie{Name: "PREF", Value: "test", Domain: "www.youtube.com", Path: "/watch", HostOnly: true, ExpirationDate: 2000000000})
	file, err := CreateCookieFile(root, cookies)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "#HttpOnly_.youtube.com\tTRUE\t/\tTRUE\t0\tSID\t") || !strings.Contains(string(data), "www.youtube.com\tFALSE\t/watch\tFALSE\t2000000000\tPREF\ttest") {
		t.Fatal("incorrect Netscape attributes")
	}
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file.Path); err != nil {
		t.Fatal("active task removed by startup cleanup")
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file.Path); !os.IsNotExist(err) {
		t.Fatal("cookie file retained")
	}
	stale, err := CreateCookieFile(root, cookies)
	if err != nil {
		t.Fatal(err)
	}
	stale.lock.Close()
	stale.lock = nil
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale.Path); !os.IsNotExist(err) {
		t.Fatal("stale cookie file retained")
	}
}
func TestArgsAndFailureClassification(t *testing.T) {
	s := validStart()
	s.Options.MaxHeight = 1080
	args := BuildArgs(s, t.TempDir(), "cookie-path", filepath.Join(t.TempDir(), "ffmpeg.exe"))
	all := strings.Join(args, "|")
	for _, required := range []string{"--ignore-config", "--no-playlist", "height<=1080", "--cookies|cookie-path", "--output-na-placeholder|null"} {
		if !strings.Contains(all, required) {
			t.Fatal("missing argument", required)
		}
	}
	if strings.Contains(all, s.Cookies[0].Value) || args[len(args)-2] != "--" {
		t.Fatal("secret or unsafe URL in args")
	}
	for input, code := range map[string]string{"ERROR: PO Token required": "EXTRACTOR_TOKEN_REQUIRED", "ERROR: cookies are no longer valid": "COOKIE_REJECTED", "Sign in to confirm you're not a bot": "AUTH_REQUIRED", "Connection timed out": "NETWORK_ERROR", "ffmpeg failed": "MEDIA_PROCESSING_FAILED"} {
		failure := Classify(input)
		if failure == nil || failure.Code != code {
			t.Fatalf("misclassified %s", input)
		}
	}
}

func buildFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-tool.exe")
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", path, "./testdata/fake-tool")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture build failed: %v %s", err, data)
	}
	return path
}
func fixtureConfig(t *testing.T, tool string) Config {
	t.Helper()
	dir := t.TempDir()
	data, err := os.ReadFile(tool)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"yt-dlp.exe", "ffmpeg.exe", "ffprobe.exe"} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return Config{OutputDir: t.TempDir(), YTDLPPath: filepath.Join(dir, "yt-dlp.exe"), FFmpegPath: filepath.Join(dir, "ffmpeg.exe")}
}
func TestDownloadProcessAndNativeSession(t *testing.T) {
	tool := buildFixture(t)
	for _, mode := range []string{"success", "failure", "cancel", "niconico-success", "niconico-failure", "niconico-anonymous-failure"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("VB_FIXTURE_MODE", mode)
			cfg := fixtureConfig(t, tool)
			root := t.TempDir()
			if err := PrepareCookieRoot(root); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			progressed := false
			start := validStart()
			if strings.HasPrefix(mode, "niconico-") {
				start.URL = "https://www.nicovideo.jp/watch/so46647398"
				start.Cookies[0].Domain = ".nicovideo.jp"
				start.Cookies[0].Name = "user_session"
				start.Cookies[0].Secure = false
			}
			if mode == "niconico-anonymous-failure" {
				start.Mode = "off"
				start.Cookies = nil
			}
			files, err := Download(ctx, cfg, root, start, func(p map[string]any) {
				progressed = true
				if mode == "cancel" {
					cancel()
				}
			})
			if (mode == "success" || mode == "niconico-success") && (err != nil || len(files) != 1 || !progressed) {
				t.Fatalf("success: files=%v progress=%v error=%v", files, progressed, err)
			}
			if (mode == "success" || mode == "niconico-success") && filepath.Base(files[0]) != "fixture.mp4" {
				t.Fatal("completed event kept the ID filename", files)
			}
			if mode == "failure" && (err == nil || asFailure(err).Code != "AUTH_REQUIRED" || strings.Contains(err.Error(), "synthetic-secret")) {
				t.Fatalf("failure not sanitized: %v", err)
			}
			if mode == "niconico-failure" && (err == nil || asFailure(err).Code != "MEMBERSHIP_REQUIRED" || strings.Contains(err.Error(), "synthetic-secret")) {
				t.Fatalf("Niconico failure not classified and sanitized: %v", err)
			}
			if mode == "niconico-anonymous-failure" && (err == nil || asFailure(err).Code != "MEMBERSHIP_REQUIRED" || !strings.Contains(err.Error(), "本次为匿名下载") || strings.Contains(err.Error(), "synthetic-secret")) {
				t.Fatalf("Anonymous membership failure lacks actionable guidance: %v", err)
			}
			if mode == "cancel" && err != context.Canceled {
				t.Fatalf("cancel failed: %v", err)
			}
			entries, _ := os.ReadDir(root)
			if len(entries) != 0 {
				t.Fatal("task secret directory retained")
			}
		})
	}
	t.Run("native session", func(t *testing.T) {
		t.Setenv("VB_FIXTURE_MODE", "success")
		cfg := fixtureConfig(t, tool)
		root := t.TempDir()
		if err := PrepareCookieRoot(root); err != nil {
			t.Fatal(err)
		}
		input, writer := io.Pipe()
		reader, output := io.Pipe()
		done := make(chan error, 1)
		server := &Server{Config: cfg, CookieRoot: root}
		go func() { done <- server.Serve(input, output); output.Close() }()
		defer writer.Close()
		defer reader.Close()
		request := func(id, kind string, p any) {
			t.Helper()
			data, _ := json.Marshal(map[string]any{"version": 1, "id": id, "type": kind, "payload": p})
			var h [4]byte
			binary.NativeEndian.PutUint32(h[:], uint32(len(data)))
			if _, err := writer.Write(append(h[:], data...)); err != nil {
				t.Fatal(err)
			}
		}
		request("start-1", "download.start", validStart())
		kinds := []string{}
		for {
			data, err := ReadFrame(reader)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "synthetic-secret") {
				t.Fatal("secret leaked in native response")
			}
			var event Event
			if json.Unmarshal(data, &event) != nil {
				t.Fatal("invalid event")
			}
			kinds = append(kinds, event.Type)
			if event.Type == "download.completed" {
				break
			}
			if event.Type == "download.failed" {
				t.Fatalf("unexpected failure: %s", data)
			}
		}
		if kinds[0] != "download.accepted" || len(kinds) < 3 {
			t.Fatalf("missing native events: %v", kinds)
		}
		writer.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("host did not exit on EOF")
		}
	})
}
