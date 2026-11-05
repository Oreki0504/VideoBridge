package bridge

import "runtime"

func ytdlpAsset(goos, arch string) string {
	if goos == "windows" && arch == "amd64" {
		return "yt-dlp.exe"
	}
	if goos == "darwin" && (arch == "arm64" || arch == "amd64") {
		return "yt-dlp_macos"
	}
	return ""
}

func toolName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func toolsSupported() bool { return ytdlpAsset(runtime.GOOS, runtime.GOARCH) != "" }
