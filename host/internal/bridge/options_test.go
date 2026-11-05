package bridge

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func argument(args []string, key string) string {
	for i, v := range args {
		if v == key && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
func TestExpandedDownloadOptions(t *testing.T) {
	s := validStart()
	out := t.TempDir()
	ff := filepath.Join(out, "ffmpeg.exe")
	names := map[string]bool{}
	for _, height := range []int{0, 144, 240, 360, 480, 720, 1080, 1440, 2160, 4320} {
		s.Options = Options{Format: "best", MaxHeight: height}
		if err := Validate(&s); err != nil {
			t.Fatal(height, err)
		}
		args := BuildArgs(s, out, "", ff)
		if height > 0 && !strings.Contains(argument(args, "-f"), "height<="+strconv.Itoa(height)) {
			t.Fatal("cap not applied", height)
		}
		name := argument(args, "--output")
		if names[name] {
			t.Fatal("different resolutions collide")
		}
		names[name] = true
	}
	for _, format := range []string{"original", "mp3", "m4a", "flac", "wav"} {
		qualities := []string{""}
		if format == "mp3" {
			qualities = []string{"best", "128", "192", "256", "320"}
		}
		for _, quality := range qualities {
			s.Options = Options{Format: "audio", AudioFormat: format, AudioQuality: quality}
			if err := Validate(&s); err != nil {
				t.Fatal(format, quality, err)
			}
			args := BuildArgs(s, out, "", ff)
			expected := format
			if format == "original" {
				expected = "best"
			}
			if argument(args, "--audio-format") != expected {
				t.Fatal("wrong audio codec option")
			}
			wantQuality := ""
			if quality == "best" {
				wantQuality = "0"
			} else if quality != "" {
				wantQuality = quality + "K"
			}
			if argument(args, "--audio-quality") != wantQuality {
				t.Fatal("wrong audio quality")
			}
			name := argument(args, "--output")
			if names[name] {
				t.Fatal("different formats/bitrates collide")
			}
			names[name] = true
		}
	}
	for _, o := range []Options{
		{Format: "best", MaxHeight: 999}, {Format: "best", AudioFormat: "mp3"},
		{Format: "audio", MaxHeight: 720}, {Format: "audio", AudioFormat: "--exec"},
		{Format: "audio", AudioFormat: "original", AudioQuality: "320"}, {Format: "audio", AudioFormat: "wav", AudioQuality: "best"},
		{Format: "audio", AudioFormat: "mp3", AudioQuality: "999"},
	} {
		s.Options = o
		if Validate(&s) == nil {
			t.Fatalf("accepted invalid options: %+v", o)
		}
	}
}

func TestFragmentConcurrency(t *testing.T) {
	for _, format := range []string{"best", "audio"} {
		base := validStart()
		base.Options = Options{Format: format}
		baseArgs := BuildArgs(base, t.TempDir(), "", "ffmpeg.exe")
		if argument(baseArgs, "--concurrent-fragments") != "1" {
			t.Fatal("legacy default changed")
		}
		for _, n := range []int{1, 4, 8, 16} {
			s := validStart()
			s.Options = Options{Format: format, ConcurrentFragments: n}
			if err := Validate(&s); err != nil {
				t.Fatal(err)
			}
			args := BuildArgs(s, t.TempDir(), "", "ffmpeg.exe")
			if argument(args, "--concurrent-fragments") != strconv.Itoa(n) {
				t.Fatal("wrong concurrency")
			}
			if argument(args, "--output") != argument(baseArgs, "--output") {
				t.Fatal("concurrency changed media filename")
			}
		}
	}
	for _, n := range []int{-1, 2, 3, 32, 1000} {
		s := validStart()
		s.Options.ConcurrentFragments = n
		if Validate(&s) == nil {
			t.Fatal("accepted unsupported concurrency", n)
		}
	}
	for _, raw := range []string{`{"format":"best","concurrentFragments":4.5}`, `{"format":"best","concurrentFragments":"4"}`} {
		var options Options
		if Decode([]byte(raw), &options) == nil {
			t.Fatal("accepted wrong concurrency type")
		}
	}
}
