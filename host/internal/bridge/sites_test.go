package bridge

import "testing"

func TestGenericMediaSites(t *testing.T) {
	for _, raw := range []string{"https://www.bilibili.com/video/BV1xx411c7mD?p=2", "https://www.nicovideo.jp/watch/sm9", "https://vimeo.com/123", "https://media.example.org/video.m3u8?token=abc#video", "http://media.example.org/video.mp4"} {
		actual, err := NormalizeURL(raw)
		if err != nil || actual != raw {
			t.Fatalf("%s: %s %v", raw, actual, err)
		}
	}
	for _, raw := range []string{"http://127.0.0.1/video", "https://[::1]/v", "https://localhost/v", "https://a.internal/v", "https://co.jp/v", "file:///C:/v.mp4", "https://2130706433/v"} {
		if _, err := NormalizeURL(raw); err == nil {
			t.Fatal("accepted", raw)
		}
	}
	for _, tc := range []struct{ url, domain string }{{"https://www.bilibili.com/video/123", ".bilibili.com"}, {"https://www.nicovideo.jp/watch/sm9", ".nicovideo.jp"}, {"https://video.example.co.jp/v", ".example.co.jp"}, {"https://foo.github.io/v", ".foo.github.io"}} {
		s := validStart()
		s.URL = tc.url
		s.Cookies[0].Domain = tc.domain
		if err := Validate(&s); err != nil {
			t.Fatal(tc, err)
		}
		for _, bad := range []string{".youtube.com", ".co.jp", ".github.io", ".bar.github.io", ".bilibili.com.evil.org"} {
			s.Cookies[0].Domain = bad
			if Validate(&s) == nil {
				t.Fatal("cross site cookie accepted", tc, bad)
			}
		}
	}
	s := validStart()
	s.URL = "http://media.example.org/video.mp4"
	s.Cookies[0].Domain = ".example.org"
	if Validate(&s) == nil {
		t.Fatal("HTTP credentials accepted")
	}
	for _, line := range []string{"ERROR: Unsupported URL: https://example.org", "ERROR: No video formats found!"} {
		if f := Classify(line); f == nil || f.Code != "UNSUPPORTED_MEDIA" {
			t.Fatal(line, f)
		}
	}
	if f := Classify("ERROR: This video is DRM protected"); f == nil || f.Code != "DRM_PROTECTED" {
		t.Fatal(f)
	}
}
