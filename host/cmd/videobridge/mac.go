package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"videobridge/host/internal/bridge"
	"videobridge/host/internal/macinstall"
)

func macCommand() bool {
	if len(os.Args) < 2 || (os.Args[1] != "--install-macos" && os.Args[1] != "--unregister-macos") {
		return false
	}
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(os.Stderr, "This command requires macOS.")
		os.Exit(1)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if os.Args[1] == "--unregister-macos" {
		if err = macinstall.Unregister(home); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("已断开 Chrome / Edge 本地连接。下载文件、配置和工具保留。请在浏览器中移除扩展。")
		return true
	}
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Expected package directory.")
		os.Exit(1)
	}
	exe, err := macinstall.Install(os.Args[2], home, runtime.GOARCH)
	if err != nil {
		fmt.Fprintln(os.Stderr, "安装未完成：", err)
		os.Exit(1)
	}
	fmt.Println("扩展目录：", filepath.Join(macinstall.Root(home), "extension"))
	fmt.Println("本地程序已安装，正在检查并补齐依赖…")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	_, err = bridge.EnsureTools(ctx, filepath.Join(filepath.Dir(exe), "config.json"), func(s string) { fmt.Println(s) })
	if err != nil {
		fmt.Fprintln(os.Stderr, "依赖未就绪，可在插件内重试：", err)
		os.Exit(1)
	}
	fmt.Println("安装完成。请在 Chrome / Edge 中加载上述扩展目录；升级用户请重新加载扩展。")
	return true
}
