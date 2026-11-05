package bridge

import (
	"errors"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
var videoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var cookieDomain = regexp.MustCompile(`^\.?([a-z0-9-]+\.)*youtube\.com$`)
var xCookieDomain = regexp.MustCompile(`^\.?([a-z0-9-]+\.)*(x\.com|twitter\.com)$`)
var xStatusPath = regexp.MustCompile(`^/(?:[A-Za-z0-9_]{1,15}|i/web)/status/([0-9]{1,20})(?:/video/([1-4]))?/?$`)
var authCookie = regexp.MustCompile(`^(SID|HSID|SSID|SAPISID|APISID|__Secure-[13]P(SID|APISID))$`)

type Options struct {
	Format              string `json:"format"`
	MaxHeight           int    `json:"maxHeight,omitempty"`
	AudioFormat         string `json:"audioFormat,omitempty"`
	AudioQuality        string `json:"audioQuality,omitempty"`
	ConcurrentFragments int    `json:"concurrentFragments,omitempty"`
	GIFStart            int    `json:"gifStart,omitempty"`
	GIFDuration         int    `json:"gifDuration,omitempty"`
	GIFWidth            int    `json:"gifWidth,omitempty"`
	GIFFPS              int    `json:"gifFps,omitempty"`
	SubtitleLanguage    string `json:"subtitleLanguage,omitempty"`
	SubtitleFormat      string `json:"subtitleFormat,omitempty"`
	SubtitleAuto        bool   `json:"subtitleAuto,omitempty"`
}

func (o Options) gifDuration() int {
	if o.GIFDuration == 0 {
		return 15
	}
	return o.GIFDuration
}
func (o Options) gifWidth() int {
	if o.GIFWidth == 0 {
		return 480
	}
	return o.GIFWidth
}
func (o Options) gifFPS() int {
	if o.GIFFPS == 0 {
		return 12
	}
	return o.GIFFPS
}
func oneOf(value int, allowed ...int) bool {
	for _, n := range allowed {
		if value == n {
			return true
		}
	}
	return false
}

func (o Options) effectiveConcurrency() int {
	if o.ConcurrentFragments == 0 {
		return 1
	} // Omitted by older extensions.
	return o.ConcurrentFragments
}

func (o Options) effectiveAudioFormat() string {
	if o.AudioFormat == "" {
		return "mp3"
	} // Older extensions sent only format=audio.
	return o.AudioFormat
}
func (o Options) effectiveAudioQuality() string {
	if o.AudioQuality == "" {
		return "best"
	}
	return o.AudioQuality
}
func (o Options) Valid() bool {
	switch o.effectiveConcurrency() {
	case 1, 4, 8, 16:
	default:
		return false
	}
	if o.Format == "subtitles" {
		return o.AudioFormat == "" && o.AudioQuality == "" && o.MaxHeight == 0 &&
			o.GIFStart == 0 && o.GIFDuration == 0 && o.GIFWidth == 0 && o.GIFFPS == 0 &&
			o.subtitleLanguages() != "" && (o.SubtitleFormat == "" || o.SubtitleFormat == "srt" || o.SubtitleFormat == "vtt")
	}
	if o.SubtitleLanguage != "" || o.SubtitleFormat != "" || o.SubtitleAuto {
		return false
	}
	if o.Format == "gif" {
		return o.AudioFormat == "" && o.AudioQuality == "" && o.MaxHeight == 0 && o.GIFStart >= 0 && o.GIFStart <= 86400 && oneOf(o.gifDuration(), 5, 10, 15, 30, 60) && oneOf(o.gifWidth(), 320, 480, 640) && oneOf(o.gifFPS(), 8, 12, 15, 20)
	}
	if o.GIFStart != 0 || o.GIFDuration != 0 || o.GIFWidth != 0 || o.GIFFPS != 0 {
		return false
	}
	if o.Format == "best" {
		if o.AudioFormat != "" || o.AudioQuality != "" {
			return false
		}
		switch o.MaxHeight {
		case 0, 144, 240, 360, 480, 720, 1080, 1440, 2160, 4320:
			return true
		}
		return false
	}
	if o.Format != "audio" || o.MaxHeight != 0 {
		return false
	}
	switch o.effectiveAudioFormat() {
	case "original", "m4a", "flac", "wav":
		return o.AudioQuality == ""
	case "mp3":
		switch o.effectiveAudioQuality() {
		case "best", "128", "192", "256", "320":
			return true
		}
	}
	return false
}

type Cookie struct {
	Name           string  `json:"name"`
	Value          string  `json:"value"`
	Domain         string  `json:"domain"`
	Path           string  `json:"path"`
	Secure         bool    `json:"secure"`
	HTTPOnly       bool    `json:"httpOnly"`
	HostOnly       bool    `json:"hostOnly"`
	Session        bool    `json:"session"`
	ExpirationDate float64 `json:"expirationDate,omitempty"`
}
type Start struct {
	URL     string   `json:"url"`
	StoreID string   `json:"profileStoreId"`
	Mode    string   `json:"cookieMode"`
	Options Options  `json:"options"`
	Cookies []Cookie `json:"cookies"`
}
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (f *Failure) Error() string      { return f.Message }
func fail(code, message string) error { return &Failure{code, message} }

func NormalizeURL(raw string) (string, error) {
	if len(raw) > 4096 {
		return "", errors.New("invalid URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Port() != "" {
		return "", errors.New("invalid URL")
	}
	u.Host = strings.ToLower(u.Host)
	if !publicWebHost(u.Host) {
		return "", errors.New("not a public web hostname")
	}
	if u.Scheme != "https" && (u.Host == "youtu.be" || u.Host == "youtube.com" || strings.HasSuffix(u.Host, ".youtube.com") || u.Host == "x.com" || strings.HasSuffix(u.Host, ".x.com") || u.Host == "twitter.com" || strings.HasSuffix(u.Host, ".twitter.com")) {
		return "", errors.New("HTTPS required")
	}
	var id string
	switch u.Host {
	case "x.com", "www.x.com", "twitter.com", "www.twitter.com", "mobile.twitter.com":
		match := xStatusPath.FindStringSubmatch(u.Path)
		if match == nil {
			return "", errors.New("not a status")
		}
		result := "https://x.com/i/web/status/" + match[1]
		if match[2] != "" {
			result += "/video/" + match[2]
		}
		return result, nil
	case "youtu.be":
		id = strings.TrimPrefix(u.Path, "/")
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else {
			parts := strings.Split(strings.TrimSuffix(u.Path, "/"), "/")
			if len(parts) == 3 && (parts[1] == "shorts" || parts[1] == "live" || parts[1] == "embed") {
				id = parts[2]
			}
		}
	default:
		return u.String(), nil
	}
	if !videoID.MatchString(id) {
		return "", errors.New("not a video")
	}
	return "https://www.youtube.com/watch?v=" + id, nil
}

func Validate(s *Start) error {
	normalized, err := NormalizeURL(s.URL)
	if err != nil {
		return fail("INVALID_URL", "请使用公共网站的 HTTP / HTTPS 媒体链接；不支持本地地址、特殊协议或嵌入用户名密码。")
	}
	s.URL = normalized
	if !s.Options.Valid() {
		return fail("INVALID_OPTIONS", "不支持的下载选项。")
	}
	if s.Mode != "always" && s.Mode != "off" {
		return fail("INVALID_REQUEST", "Cookie 模式无效。")
	}
	if len(s.Cookies) > 512 || len(s.StoreID) > 80 {
		return fail("INVALID_COOKIES", "Cookie 快照过大。")
	}
	if s.Mode == "off" {
		if len(s.Cookies) != 0 {
			return fail("INVALID_COOKIES", "关闭 Cookie 时不能附带会话。")
		}
		return nil
	}
	if s.StoreID == "" || len(s.Cookies) == 0 {
		return fail("COOKIE_UNAVAILABLE", "缺少当前浏览器会话 Cookie。")
	}
	if !strings.HasPrefix(s.URL, "https://") {
		return fail("HTTPS_REQUIRED", "HTTP 页面仅支持匿名下载；请打开 HTTPS 页面后使用会话。")
	}
	auth := false
	isX := strings.HasPrefix(s.URL, "https://x.com/")
	isYouTube := strings.HasPrefix(s.URL, "https://www.youtube.com/")
	roots := cookieRoots(s.URL)
	seen := map[string]bool{}
	for _, c := range s.Cookies {
		domainAllowed := allowedCookieDomain(c.Domain, roots)
		if !domainAllowed || c.Name == "" || len(c.Name) > 1024 || len(c.Value) > 16384 || !strings.HasPrefix(c.Path, "/") || len(c.Path) > 4096 || strings.ContainsAny(c.Name+c.Value+c.Domain+c.Path, "\x00\t\r\n") || (c.HostOnly && strings.HasPrefix(c.Domain, ".")) || math.IsNaN(c.ExpirationDate) || math.IsInf(c.ExpirationDate, 0) || c.ExpirationDate < 0 || c.ExpirationDate > 253402300799 {
			return fail("INVALID_COOKIES", "Cookie 字段无法安全转换。")
		}
		key := c.Domain + "\t" + c.Path + "\t" + c.Name
		if seen[key] {
			return fail("INVALID_COOKIES", "Cookie 快照包含重复记录。")
		}
		seen[key] = true
		isAuth := !isYouTube || authCookie.MatchString(c.Name)
		if isX {
			isAuth = c.Name == "auth_token"
		}
		if isAuth && c.Value != "" && (c.Session || c.ExpirationDate > float64(time.Now().Unix())) {
			auth = true
		}
	}
	if !auth {
		return fail("COOKIE_UNAVAILABLE", "没有可用的当前站点登录 Cookie。")
	}
	return nil
}
