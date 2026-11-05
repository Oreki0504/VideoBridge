package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var subtitleSuffix = regexp.MustCompile(`^\.[A-Za-z0-9_-]+\.(srt|vtt)$`)

func (o Options) subtitleLanguages() string {
	languages := ""
	switch o.SubtitleLanguage {
	case "", "zh-en":
		languages = "zh.*,en.*,ai-zh.*,ai-en.*"
	case "zh":
		languages = "zh.*,ai-zh.*"
	case "en":
		languages = "en.*,ai-en.*"
	case "ja":
		languages = "ja.*,ai-ja.*"
	case "all":
		languages = "all"
	default:
		return ""
	}
	if !o.SubtitleAuto {
		languages += ",-ai-.*"
	}
	return languages + ",-live_chat,-danmaku"
}

func (o Options) subtitleFormat() string {
	if o.SubtitleFormat == "" {
		return "srt"
	}
	return o.SubtitleFormat
}

func (p *outputParser) subtitleFiles(raw string) {
	var report struct {
		ID    string   `json:"id"`
		Paths []string `json:"paths"`
	}
	if json.Unmarshal([]byte(raw), &report) != nil || len(report.Paths) > 512 {
		p.failure = &Failure{"SUBTITLE_DOWNLOAD_FAILED", "无法验证字幕文件，请更新 yt-dlp 后重试。"}
		return
	}
	for _, path := range report.Paths {
		info, err := os.Lstat(path)
		ext := strings.ToLower(filepath.Ext(path))
		if !filepath.IsAbs(path) || filepath.Clean(filepath.Dir(path)) != filepath.Clean(p.outputDir) || err != nil || !info.Mode().IsRegular() || info.Size() == 0 || (ext != ".srt" && ext != ".vtt") {
			p.failure = &Failure{"SUBTITLE_DOWNLOAD_FAILED", "部分字幕未成功下载或转换，请重试或更换字幕格式。"}
			return
		}
		duplicate := false
		for _, existing := range p.files {
			duplicate = duplicate || existing.Path == path
		}
		if !duplicate {
			p.files = append(p.files, downloadedFile{Path: path, ID: report.ID})
		}
	}
}

func finishSubtitles(files []downloadedFile, outputDir string, options Options) ([]string, error) {
	if len(files) == 0 {
		return nil, fail("SUBTITLES_UNAVAILABLE", "此视频没有所选语言的可用字幕；可尝试其他语言或开启自动字幕。Bilibili 弹幕不属于字幕。")
	}
	result := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.Ext(file.Path) != "."+options.subtitleFormat() {
			return nil, fail("SUBTITLE_DOWNLOAD_FAILED", "字幕未转换为所选格式，请检查 FFmpeg 后重试。")
		}
		name := filepath.Base(file.Path)
		marker := " [" + file.ID + "]"
		if position := strings.LastIndex(name, marker); file.ID != "" && position >= 0 && subtitleSuffix.MatchString(name[position+len(marker):]) {
			title := strings.TrimRight(name[:position], " .")
			if title == "" {
				title = "subtitles"
			}
			name = title + name[position+len(marker):]
		}
		if windowsDeviceName(name) {
			name = "_" + name
		}
		path, err := publishUnique(file.Path, filepath.Join(outputDir, name))
		if err != nil {
			return nil, fail("OUTPUT_UNAVAILABLE", "无法保存字幕，请检查下载目录和磁盘空间。")
		}
		result = append(result, path)
	}
	return result, nil
}
