package main

import (
	"testing"
	"videobridge/host/internal/bridge"
)

func TestMacLocaleResolution(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"default_en", map[string]string{}, "en"},
		{"c_locale_fallback", map[string]string{"LANG": "C.UTF-8"}, "en"},
		{"zh_cn_lang", map[string]string{"LANG": "zh_CN.UTF-8"}, "zh"},
		{"zh_hans_override", map[string]string{"VIDEOBRIDGE_LANG": "zh-Hans", "LANG": "en_US.UTF-8"}, "zh"},
		{"en_override_over_zh_lang", map[string]string{"VIDEOBRIDGE_LANG": "en", "LANG": "zh_CN.UTF-8"}, "en"},
		{"ja_fallback_en", map[string]string{"LC_ALL": "ja_JP.UTF-8"}, "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := macLocale(func(k string) string { return tc.env[k] })
			if got != tc.want {
				t.Fatalf("macLocale(%v) = %q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

func TestLocalizeMacMessage(t *testing.T) {
	if got := localizeMacMessage("查询 yt-dlp 官方版本…", "en"); got != "Checking official yt-dlp releases…" {
		t.Fatalf("unexpected en message: %q", got)
	}
	if got := localizeMacMessage("下载并校验 yt-dlp 2026.10.26…", "en"); got != "Downloading & verifying yt-dlp 2026.10.26…" {
		t.Fatalf("unexpected ytdlp message: %q", got)
	}
	if got := localizeMacMessage("下载并校验 macOS ffmpeg（OSXExperts 构建）…", "en"); got != "Downloading & verifying macOS ffmpeg (OSXExperts build)…" {
		t.Fatalf("unexpected mac tool message: %q", got)
	}
	if got := localizeMacMessage("查询 yt-dlp 官方版本…", "zh"); got != "查询 yt-dlp 官方版本…" {
		t.Fatalf("unexpected zh message: %q", got)
	}
}

func TestLocalizeMacFailure(t *testing.T) {
	failure := &bridge.Failure{Code: "SETUP_BUSY", Message: "另一个安装进程正在运行，请稍后重试。"}
	if got := localizeMacMessage(failure.Error(), "en"); got != "Another installation process is running. Please retry shortly. (SETUP_BUSY)" {
		t.Fatalf("real setup error was not translated: %q", got)
	}
}
