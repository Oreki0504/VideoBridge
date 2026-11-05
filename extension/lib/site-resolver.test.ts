import { describe, it, expect } from 'vitest';
import { resolveVideo, sitePermission, cookieSettingKey, usesCookies } from './site-resolver';

describe('video URL boundary', () => {
  it('normalizes X posts and preserves the selected video index', () => {
    expect(resolveVideo('https://twitter.com/someone/status/123/video/2?s=20')).toEqual({url:'https://x.com/i/web/status/123/video/2',videoId:'123',site:'x'});
    expect(resolveVideo('https://x.com/i/web/status/123').site).toBe('x');
    for (const url of ['https://x.com/home', 'https://x.com/a/status/abc', 'https://x.com/a/status/123/video/5']) expect(() => resolveVideo(url)).toThrow();
  });
  it.each([
    'https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123&index=2',
    'https://youtu.be/dQw4w9WgXcQ?t=12',
    'https://m.youtube.com/shorts/dQw4w9WgXcQ',
    'https://music.youtube.com/watch?v=dQw4w9WgXcQ',
    'https://www.youtube.com/live/dQw4w9WgXcQ',
  ])('canonicalizes only the video: %s', raw => {
    expect(resolveVideo(raw).url).toBe('https://www.youtube.com/watch?v=dQw4w9WgXcQ');
  });
  it.each([
    'http://www.youtube.com/watch?v=dQw4w9WgXcQ',
    'https://youtube.com@evil.example/watch?v=dQw4w9WgXcQ',
    'https://user@youtube.com/watch?v=dQw4w9WgXcQ',
    'https://youtube.com:444/watch?v=dQw4w9WgXcQ',
    'https://youtube.com/playlist?list=123',
    'https://youtu.be/dQw4w9WgXcQ/extra',
    'file:///etc/passwd',
  ])('rejects %s', raw => expect(() => resolveVideo(raw)).toThrow());
  it('delegates media sites without losing query or fragment selectors', () => {
    for (const raw of ['https://www.bilibili.com/video/BV1xx411c7mD?p=2','https://www.nicovideo.jp/watch/sm9','https://vimeo.com/123','https://media.example.org/movie.m3u8?token=abc#video','http://media.example.org/movie.mp4']) expect(resolveVideo(raw).url).toBe(raw);
    expect(resolveVideo('https://youtube.com.evil.example/watch?v=dQw4w9WgXcQ').site).toBe('evil.example');
    expect(resolveVideo('https://x.com.evil.test/a/status/123').site).toBe('evil.test');
  });
  it('uses the registrable domain including private suffix boundaries', () => {
    expect(resolveVideo('https://www.nicovideo.jp/watch/sm9').site).toBe('nicovideo.jp');
    expect(resolveVideo('https://video.example.co.jp/watch/1').site).toBe('example.co.jp');
    expect(resolveVideo('https://foo.github.io/watch/1').site).toBe('foo.github.io');
    expect(sitePermission('bilibili.com').origins).toEqual(['http://*.bilibili.com/*','https://*.bilibili.com/*']);
    expect(sitePermission('nicovideo.jp').origins).toEqual(['http://*.nicovideo.jp/*','https://*.nicovideo.jp/*']);
    expect(sitePermission('youtube').origins).toEqual(['http://*.youtube.com/*','https://*.youtube.com/*']);
    expect(sitePermission('x').origins).toEqual(['http://*.x.com/*','https://*.x.com/*','http://*.twitter.com/*','https://*.twitter.com/*']);
    expect(usesCookies({useCookies:true,useCookiesX:true},'bilibili.com')).toBe(false);
    expect(usesCookies({[cookieSettingKey('bilibili.com')]:true},'nicovideo.jp')).toBe(false);
  });
  it.each(['https://127.0.0.1/video','http://[::1]/v','https://localhost/v','https://a.local/v','https://co.jp/v','file:///C:/video.mp4','ftp://media.example.org/video.mp4','https://2130706433/v'])('rejects local or special-protocol source %s',raw=>expect(()=>resolveVideo(raw)).toThrow());
});
