import { describe, expect, it } from 'vitest';
import {
  formatCookieLabel,
  formatEta,
  formatPermissionState,
  formatSiteLabel,
  formatToolsHealth,
  formatUpdateStatus,
  localizedSiteName,
  resolveLocale,
  resolveTheme,
  translateDynamicText,
} from './i18n';

describe('i18n helpers', () => {
  it('follows browser language on first use and preserves an explicit choice', () => {
    expect(resolveLocale(undefined, 'en-US')).toBe('en');
    expect(resolveLocale(undefined, 'zh-CN')).toBe('zh');
    expect(resolveLocale(undefined, 'zh-TW')).toBe('zh');
    expect(resolveLocale('invalid', 'ja-JP')).toBe('en');
    expect(resolveLocale('zh', 'en-US')).toBe('zh');
    expect(resolveLocale('en', 'zh-CN')).toBe('en');
    expect(resolveTheme(undefined)).toBe('dark');
    expect(resolveTheme('light')).toBe('light');
    expect(resolveTheme('dark')).toBe('dark');
  });

  it('formats site, permission, and cookie labels in both locales', () => {
    expect(localizedSiteName('b23.tv', 'zh')).toBe('Bilibili 短链');
    expect(localizedSiteName('b23.tv', 'en')).toBe('Bilibili Short Link');
    expect(formatSiteLabel('youtube', 'zh')).toBe('YouTube · 当前页面');
    expect(formatSiteLabel('youtube', 'en')).toBe('YouTube · Current Page');
    expect(formatCookieLabel('nicovideo.jp', 'zh')).toBe('使用当前 Niconico 会话');
    expect(formatCookieLabel('nicovideo.jp', 'en')).toBe('Use current Niconico session');
    expect(formatPermissionState('youtube', false, 'zh')).toBe('尚未授权 YouTube Cookie');
    expect(formatPermissionState('youtube', true, 'zh')).toBe('已授权 YouTube Cookie');
    expect(formatPermissionState('youtube', false, 'en')).toBe('YouTube Cookie not granted');
    expect(formatPermissionState('youtube', true, 'en')).toBe('YouTube Cookie granted');
  });

  it('translates dynamic phases, setup messages, update status, and errors', () => {
    expect(translateDynamicText('下载中', 'en')).toBe('Downloading');
    expect(translateDynamicText('Downloading', 'zh')).toBe('下载中');
    expect(translateDynamicText('下载并校验 yt-dlp 2026.10.26…', 'en')).toBe('Downloading & verifying yt-dlp 2026.10.26…');
    expect(
      translateDynamicText('未发现 YouTube 可用会话 Cookie。请先在当前 Profile 登录，或关闭会话模式尝试公开视频。 (COOKIE_UNAVAILABLE)', 'en'),
    ).toBe('No usable YouTube session cookies found. Sign in to this profile first, or disable session mode for public videos. (COOKIE_UNAVAILABLE)');
    expect(formatEta(15.2, 'zh')).toBe(' · 剩余 16 秒');
    expect(formatEta(15.2, 'en')).toBe(' · 16s left');
    expect(
      formatUpdateStatus({ currentVersion: '2026.09.27', latestVersion: '2026.10.26', message: '发现新版本，可更新。' }, 'en'),
    ).toBe('Current: 2026.09.27 · Latest: 2026.10.26\nNew version available.');
    expect(
      formatToolsHealth({ version: '0.7.0', outputDir: 'D:\\Videos', tools: [{ name: 'yt-dlp', available: true, version: '2026.10.26' }] }, 'en'),
    ).toContain('Download folder: D:\\Videos');
  });
});
