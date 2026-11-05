package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"videobridge/host/internal/bridge"
	"videobridge/host/internal/macinstall"
)

func macLocale(getenv func(string) string) string {
	for _, key := range []string{"VIDEOBRIDGE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(getenv(key))
		switch strings.ToUpper(v) {
		case "", "C", "C.UTF-8", "POSIX", "UTF-8":
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "zh") {
			return "zh"
		}
		return "en"
	}
	return "en"
}

var macExactEn = map[string]string{
	"查询 yt-dlp 官方版本…":                                      "Checking official yt-dlp releases…",
	"获取 FFmpeg 校验值（FFmpeg.org 推荐的 Gyan 构建）…":               "Fetching FFmpeg checksums (Gyan build)…",
	"下载并校验 FFmpeg / FFprobe（约 115 MB）…":                    "Downloading & verifying FFmpeg / FFprobe (~115 MB)…",
	"解压 FFmpeg / FFprobe…":                                 "Extracting FFmpeg / FFprobe…",
	"检查已下载的工具…":                                            "Checking downloaded tools…",
	"本地依赖已就绪。":                                             "Local dependencies are ready.",
	"另一个安装进程正在运行，请稍后重试。":                                   "Another installation process is running. Please retry shortly. (SETUP_BUSY)",
	"自动安装目前支持 Windows x64 和 macOS（Apple Silicon / Intel）。": "Automatic installation currently supports Windows x64 and macOS (Apple Silicon / Intel). (SETUP_UNSUPPORTED)",
	"无法创建工具目录。":                                            "Could not create the tools directory. (SETUP_FAILED)",
	"无法创建安装临时目录。":                                          "Could not create a temporary setup directory. (SETUP_FAILED)",
	"依赖下载或 SHA256 校验失败，原配置已保留。请检查网络后点“检测并补齐依赖”重试，或手动配置工具。": "Dependency download or SHA256 verification failed; existing configuration was kept. Check your network and retry via \"Check & Install Dependencies\" in the extension, or configure tools manually. (SETUP_FAILED)",
	"已下载的工具无法运行，原配置已保留。":    "Downloaded tools could not run; existing configuration was kept. (SETUP_FAILED)",
	"无法保存工具目录。":             "Could not save the tools directory. (SETUP_FAILED)",
	"无法保存工具配置，原配置已保留。":      "Could not save tool configuration; existing configuration was kept. (SETUP_FAILED)",
	"无法读取 host 配置，请运行安装脚本。": "Could not read Host configuration. Run the installer. (CONFIG_ERROR)",
	"本地下载目录必须是绝对路径。":        "The download folder must be an absolute path. (CONFIG_ERROR)",
	"无法创建下载目录。":             "Could not create the download folder. (OUTPUT_UNAVAILABLE)",
	"下载目录不可用。":              "The download folder is unavailable. (OUTPUT_UNAVAILABLE)",
}

func localizeMacMessage(msg, locale string) string {
	if locale == "zh" || msg == "" {
		return msg
	}
	if en, ok := macExactEn[msg]; ok {
		return en
	}
	if strings.HasPrefix(msg, "下载并校验 yt-dlp ") && strings.HasSuffix(msg, "…") {
		ver := strings.TrimSuffix(strings.TrimPrefix(msg, "下载并校验 yt-dlp "), "…")
		return "Downloading & verifying yt-dlp " + ver + "…"
	}
	if strings.HasPrefix(msg, "下载并校验 macOS ") && strings.HasSuffix(msg, "（OSXExperts 构建）…") {
		name := strings.TrimSuffix(strings.TrimPrefix(msg, "下载并校验 macOS "), "（OSXExperts 构建）…")
		return "Downloading & verifying macOS " + name + " (OSXExperts build)…"
	}
	return msg
}

func macCommand() bool {
	if len(os.Args) < 2 || (os.Args[1] != "--install-macos" && os.Args[1] != "--unregister-macos") {
		return false
	}
	locale := macLocale(os.Getenv)
	if runtime.GOOS != "darwin" {
		if locale == "zh" {
			fmt.Fprintln(os.Stderr, "此命令仅适用于 macOS。")
		} else {
			fmt.Fprintln(os.Stderr, "This command requires macOS.")
		}
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
		if locale == "zh" {
			fmt.Println("已断开 Chrome / Edge 本地连接。下载文件、配置和工具保留。请在浏览器中移除扩展。")
		} else {
			fmt.Println("Disconnected Chrome / Edge native messaging. Downloaded files, settings, and tools were preserved. Remove the extension in your browser.")
		}
		return true
	}
	if len(os.Args) != 3 {
		if locale == "zh" {
			fmt.Fprintln(os.Stderr, "缺少安装包目录参数。")
		} else {
			fmt.Fprintln(os.Stderr, "Expected package directory.")
		}
		os.Exit(1)
	}
	exe, err := macinstall.Install(os.Args[2], home, runtime.GOARCH)
	if err != nil {
		if locale == "zh" {
			fmt.Fprintln(os.Stderr, "安装未完成：", err)
		} else {
			fmt.Fprintln(os.Stderr, "Installation incomplete:", err)
		}
		os.Exit(1)
	}
	if locale == "zh" {
		fmt.Println("扩展目录：", filepath.Join(macinstall.Root(home), "extension"))
		fmt.Println("本地程序已安装，正在检查并补齐依赖…")
	} else {
		fmt.Println("Extension folder:", filepath.Join(macinstall.Root(home), "extension"))
		fmt.Println("Native Host installed. Checking and installing dependencies…")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	_, err = bridge.EnsureTools(ctx, filepath.Join(filepath.Dir(exe), "config.json"), func(s string) {
		fmt.Println(localizeMacMessage(s, locale))
	})
	if err != nil {
		if locale == "zh" {
			fmt.Fprintln(os.Stderr, "依赖未就绪，可在插件内重试：", err)
		} else {
			fmt.Fprintln(os.Stderr, "Dependencies are not ready (you can retry inside the extension):", localizeMacMessage(err.Error(), locale))
		}
		os.Exit(1)
	}
	if locale == "zh" {
		fmt.Println("安装完成。请在 Chrome / Edge 中加载上述扩展目录；升级用户请重新加载扩展。")
	} else {
		fmt.Println("Installation complete. Load the extension folder above in Chrome / Edge, or reload the extension if upgrading.")
	}
	return true
}
