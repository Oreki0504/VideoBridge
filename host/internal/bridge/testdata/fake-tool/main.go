// A local fixture, never shipped or used for real downloads.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var fixtureVersion = "fixture 1.0"

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--child" {
		_ = os.WriteFile(os.Args[2], []byte(fmt.Sprint(os.Getpid())), 0600)
		time.Sleep(30 * time.Second)
		return
	}
	if len(os.Args) >= 2 && (os.Args[len(os.Args)-1] == "--version" || os.Args[len(os.Args)-1] == "-version") {
		fmt.Println(fixtureVersion)
		return
	}
	var output, cookie string
	for i, arg := range os.Args {
		if i+1 < len(os.Args) {
			if arg == "--paths" {
				output = os.Args[i+1]
			}
			if arg == "--cookies" {
				cookie = os.Args[i+1]
			}
		}
	}
	if cookie != "" {
		data, err := os.ReadFile(cookie)
		expected := "#HttpOnly_.youtube.com\tTRUE"
		if strings.HasPrefix(os.Getenv("VB_FIXTURE_MODE"), "niconico-") {
			expected = "#HttpOnly_.nicovideo.jp\tTRUE\t/\tFALSE\t0\tuser_session\tsynthetic-secret-never-log"
		}
		if err != nil || !strings.Contains(string(data), expected) {
			os.Exit(3)
		}
	}
	if os.Getenv("VB_FIXTURE_MODE") == "niconico-failure" || os.Getenv("VB_FIXTURE_MODE") == "niconico-anonymous-failure" {
		fmt.Fprintln(os.Stderr, "WARNING: Connection timed out")
		fmt.Fprintln(os.Stderr, "ERROR: [niconico] so46647398: Channel members only. Use --cookies, --cookies-from-browser, --username and --password, --netrc-cmd, or --netrc (niconico) to provide account credentials. synthetic-secret-never-log")
		os.Exit(1)
	}
	if os.Getenv("VB_FIXTURE_MODE") == "failure" {
		fmt.Fprintln(os.Stderr, "ERROR: Sign in to confirm your identity. synthetic-secret-never-log")
		os.Exit(1)
	}
	if os.Getenv("VB_FIXTURE_MODE") == "cancel-tree" {
		exe, _ := os.Executable()
		child := exec.Command(exe, "--child", filepath.Join(output, "child.pid"))
		if child.Start() != nil {
			os.Exit(5)
		}
		for i := 0; i < 100; i++ {
			if _, err := os.Stat(filepath.Join(output, "child.pid")); err == nil {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	fmt.Println(`VB_PROGRESS {"status":"downloading","downloaded_bytes":50,"total_bytes":100,"speed":1024,"eta":1}`)
	if os.Getenv("VB_FIXTURE_MODE") == "slow" {
		time.Sleep(4 * time.Second)
	}
	if os.Getenv("VB_FIXTURE_MODE") == "cancel" || os.Getenv("VB_FIXTURE_MODE") == "cancel-tree" {
		time.Sleep(30 * time.Second)
		return
	}
	path := filepath.Join(output, "fixture [dQw4w9WgXcQ].mp4")
	if err := os.WriteFile(path, []byte("synthetic test media"), 0600); err != nil {
		os.Exit(4)
	}
	encoded, _ := json.Marshal(map[string]string{"path": path, "id": "dQw4w9WgXcQ"})
	fmt.Printf("VB_FILE %s\n", encoded)
}
