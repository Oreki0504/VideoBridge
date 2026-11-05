package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
	"videobridge/host/internal/bridge"
)

func main() {
	if macCommand() {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}
	configPath := filepath.Join(filepath.Dir(exe), "config.json")
	cfg, err := bridge.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "VideoBridge: local configuration unavailable.")
		os.Exit(1)
	}
	if len(os.Args) == 2 && os.Args[1] == "--setup" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		cfg, err = bridge.EnsureTools(ctx, configPath, func(message string) { fmt.Fprintln(os.Stderr, message) })
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if len(os.Args) == 2 && (os.Args[1] == "--check" || os.Args[1] == "--setup") {
		data, _ := json.MarshalIndent(cfg.Health(), "", "  ")
		fmt.Println(string(data))
		return
	}
	if len(os.Args) < 2 || !regexp.MustCompile(`^chrome-extension://[a-p]{32}/$`).MatchString(os.Args[1]) {
		fmt.Fprintln(os.Stderr, "VideoBridge: start via registered browser extension or use --check.")
		os.Exit(1)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		os.Exit(1)
	}
	root := filepath.Join(cache, "VideoBridge", "cookie-tasks")
	if err := bridge.PrepareCookieRoot(root); err != nil {
		fmt.Fprintln(os.Stderr, "VideoBridge: protected cookie directory unavailable.")
		os.Exit(1)
	}
	server := bridge.Server{Config: cfg, ConfigPath: configPath, CookieRoot: root}
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "VideoBridge: protocol connection closed.")
		os.Exit(1)
	}
}
