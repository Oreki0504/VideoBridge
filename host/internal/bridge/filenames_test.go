package bridge

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestFinishedNameMatchesOnlyReportedIDSuffix(t *testing.T) {
	for _, tc := range []struct {
		name, id, want string
		options        Options
	}{
		{"标题 [现场] [dQw4w9WgXcQ].mp4", "dQw4w9WgXcQ", "标题 [现场].mp4", Options{Format: "best"}},
		{"标题 [same] [same] [max-1080p].mp4", "same", "标题 [same] [max-1080p].mp4", Options{Format: "best", MaxHeight: 1080}},
		{"声音 [so123] [audio-mp3-320].mp3", "so123", "声音 [audio-mp3-320].mp3", Options{Format: "audio", AudioFormat: "mp3", AudioQuality: "320"}},
		{"标题 [BV123].mp4", "BV123", "标题.mp4", Options{Format: "gif"}},
		{"标题 [other].mp4", "wrong-id", "标题 [other].mp4", Options{Format: "best"}},
		{"标题 [id].mp4", "", "标题 [id].mp4", Options{Format: "best"}},
		{"标题 [id] [max-720p].mp4", "id", "标题 [id] [max-720p].mp4", Options{Format: "best", MaxHeight: 1080}},
		{"尾部. [id].mp4", "id", "尾部.mp4", Options{Format: "best"}},
	} {
		if got := finishedName(downloadedFile{Path: filepath.Join(t.TempDir(), tc.name), ID: tc.id}, tc.options); got != tc.want {
			t.Fatalf("%q: got %q, want %q", tc.name, got, tc.want)
		}
	}
	if runtime.GOOS == "windows" {
		if got := finishedName(downloadedFile{Path: `C:\out\CON [id].mp4`, ID: "id"}, Options{Format: "best"}); got != "_CON.mp4" {
			t.Fatal(got)
		}
	}
}

func TestFinishFilenamePreservesCollisionsAndContents(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	original := write("同名.mp4", "existing user media")
	for n, id := range []string{"youtube-id", "so123"} {
		source := write("同名 ["+id+"].mp4", id)
		got, err := finishFilename(downloadedFile{Path: source, ID: id}, Options{Format: "best"})
		want := []string{"同名 (2).mp4", "同名 (3).mp4"}[n]
		if err != nil || filepath.Base(got) != want {
			t.Fatalf("%s %v", got, err)
		}
		if data, err := os.ReadFile(got); err != nil || string(data) != id {
			t.Fatal("new media changed")
		}
		if _, err := os.Stat(source); !os.IsNotExist(err) {
			t.Fatal("ID filename retained")
		}
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "existing user media" {
		t.Fatal("existing media replaced")
	}
	missing := filepath.Join(dir, "missing [id].mp4")
	if _, err := finishFilename(downloadedFile{Path: missing, ID: "id"}, Options{Format: "best"}); err == nil {
		t.Fatal("failed rename reported success")
	}
}

func TestConcurrentFinishedNamesNeverReplace(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	results := make(chan string, 8)
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		source := filepath.Join(dir, "shared ["+id+"].mp4")
		if err := os.WriteFile(source, []byte(id), 0600); err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := finishFilename(downloadedFile{Path: source, ID: id}, Options{Format: "best"})
			if err != nil {
				t.Error(err)
				return
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != id {
				t.Error("concurrent media replaced")
			}
			results <- out
		}()
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{}
	for p := range results {
		if seen[p] {
			t.Fatal("duplicate result")
		}
		seen[p] = true
	}
	if len(seen) != 8 {
		t.Fatal("missing outputs")
	}
}
