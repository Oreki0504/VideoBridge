package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubtitleOptions(t *testing.T) {
	for _, language := range []string{"", "zh-en", "zh", "en", "ja", "all"} {
		for _, format := range []string{"", "srt", "vtt"} {
			start := validStart()
			start.Options = Options{Format: "subtitles", SubtitleLanguage: language, SubtitleFormat: format, SubtitleAuto: true}
			if err := Validate(&start); err != nil {
				t.Fatal(err)
			}
			args := BuildArgs(start, t.TempDir(), "session.txt", "ffmpeg.exe")
			if !slices.Contains(args, "--skip-download") || !slices.Contains(args, "--write-subs") || !slices.Contains(args, "--write-auto-subs") || slices.Contains(args, "-f") || argument(args, "--cookies") != "session.txt" {
				t.Fatal("subtitle-only arguments lost", args)
			}
			if argument(args, "--convert-subs") != start.Options.subtitleFormat() || !strings.Contains(argument(args, "--sub-langs"), "-live_chat") {
				t.Fatal(args)
			}
		}
	}
	start := validStart()
	start.Options = Options{Format: "subtitles"}
	if slices.Contains(BuildArgs(start, t.TempDir(), "", "ffmpeg.exe"), "--write-auto-subs") {
		t.Fatal("unrequested auto captions enabled")
	}
	if !strings.Contains(start.Options.subtitleLanguages(), "-ai-.*") || !strings.Contains(start.Options.subtitleLanguages(), "-danmaku") {
		t.Fatal("Bilibili AI captions or danmaku not excluded")
	}
	for _, invalid := range []Options{
		{Format: "subtitles", SubtitleLanguage: ".*"}, {Format: "subtitles", SubtitleFormat: "../../file"},
		{Format: "subtitles", AudioFormat: "mp3"}, {Format: "subtitles", MaxHeight: 1080}, {Format: "subtitles", GIFStart: 1},
		{Format: "best", SubtitleLanguage: "en"}, {Format: "gif", SubtitleAuto: true},
	} {
		if invalid.Valid() {
			t.Fatalf("accepted %+v", invalid)
		}
	}
}

func TestSubtitleOutputValidation(t *testing.T) {
	for _, invalid := range []string{"outside", "missing", "empty", "media"} {
		t.Run(invalid, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "captions.en.srt")
			content := "1\n00:00:00,000 --> 00:00:01,000\nTest\n"
			switch invalid {
			case "outside":
				path = filepath.Join(t.TempDir(), "captions.en.srt")
			case "empty":
				content = ""
			case "media":
				path = filepath.Join(root, "video.mp4")
			}
			if invalid != "missing" {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			raw, _ := json.Marshal(map[string]any{"id": "id", "paths": []string{path}})
			parser := outputParser{outputDir: root}
			parser.line("VB_SUBS " + string(raw))
			if parser.failure == nil || len(parser.files) != 0 {
				t.Fatal("accepted invalid output")
			}
		})
	}
}

func TestSubtitleFailureAndCancellationCleanup(t *testing.T) {
	tool := buildFixture(t)
	for _, mode := range []string{"failure", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("VB_FIXTURE_MODE", mode)
			cfg := fixtureConfig(t, tool)
			root := t.TempDir()
			if err := PrepareCookieRoot(root); err != nil {
				t.Fatal(err)
			}
			start := validStart()
			start.Options = Options{Format: "subtitles"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := Download(ctx, cfg, root, start, func(update map[string]any) {
				if mode == "cancel" && update["phase"] == "下载中" {
					cancel()
				}
			})
			if err == nil {
				t.Fatal("failed subtitle task reported success")
			}
			for _, directory := range []string{root, cfg.OutputDir} {
				entries, err := os.ReadDir(directory)
				if err != nil || len(entries) != 0 {
					t.Fatalf("temporary files remain: %v %v", entries, err)
				}
			}
		})
	}
}

func TestRealSubtitleDownload(t *testing.T) {
	ytdlp, ffmpeg := os.Getenv("VB_INTEGRATION_YTDLP"), os.Getenv("VB_INTEGRATION_FFMPEG")
	if ytdlp == "" || ffmpeg == "" {
		t.Skip("real tools not configured")
	}
	var mediaRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/watch":
			writer.Header().Set("Content-Type", "text/html")
			_, _ = writer.Write([]byte(`<html><title>Subtitle fixture</title><video src="/media.mp4"><track kind="subtitles" src="/english.vtt" srclang="en"><track kind="subtitles" src="/chinese.vtt" srclang="zh-Hans"></video></html>`))
		case "/english.vtt", "/chinese.vtt":
			writer.Header().Set("Content-Type", "text/vtt")
			_, _ = writer.Write([]byte("WEBVTT\n\n00:00:00.000 --> 00:00:01.000\nHello 字幕\n"))
		case "/media.mp4":
			mediaRequests.Add(1)
			http.Error(writer, "video body must not be downloaded", http.StatusForbidden)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	if err := PrepareCookieRoot(root); err != nil {
		t.Fatal(err)
	}
	cfg := Config{OutputDir: t.TempDir(), YTDLPPath: ytdlp, FFmpegPath: ffmpeg}
	for _, format := range []string{"srt", "vtt"} {
		for attempt := 0; attempt < 2; attempt++ {
			start := validStart()
			start.URL = server.URL + "/watch"
			start.Options = Options{Format: "subtitles", SubtitleLanguage: "zh-en", SubtitleFormat: format, SubtitleAuto: true}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			files, err := Download(ctx, cfg, root, start, func(map[string]any) {})
			cancel()
			if err != nil || len(files) != 2 {
				t.Fatalf("subtitle output: %v %v", files, err)
			}
			for _, file := range files {
				content, err := os.ReadFile(file)
				if err != nil || !strings.Contains(string(content), "Hello 字幕") || filepath.Ext(file) != "."+format || strings.Contains(filepath.Base(file), "[") {
					t.Fatalf("wrong subtitle: %s %v", file, err)
				}
				if attempt == 1 && !strings.HasSuffix(file, " (2)."+format) {
					t.Fatal("duplicate replaced", file)
				}
			}
		}
	}
	start := validStart()
	start.URL = server.URL + "/watch"
	start.Options = Options{Format: "subtitles", SubtitleLanguage: "ja"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := Download(ctx, cfg, root, start, func(map[string]any) {})
	if err == nil || asFailure(err).Code != "SUBTITLES_UNAVAILABLE" {
		t.Fatalf("missing language: %v", err)
	}
	if mediaRequests.Load() != 0 {
		t.Fatal("downloaded video during subtitle-only task")
	}
	remaining, err := os.ReadDir(root)
	if err != nil || len(remaining) != 0 {
		t.Fatal("cookie cleanup failed")
	}
	outputs, err := os.ReadDir(cfg.OutputDir)
	if err != nil || len(outputs) != 8 {
		t.Fatalf("unexpected output or temporary leftovers: %v %v", outputs, err)
	}
}
