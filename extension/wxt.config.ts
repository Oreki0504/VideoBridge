import { defineConfig } from 'wxt';

export default defineConfig({
  manifest: {
    name: 'VideoBridge',
    description: 'Download media and subtitles from YouTube, Bilibili and other yt-dlp sites, or convert videos to GIF.',
    version: '0.9.0',
    minimum_chrome_version: '120',
    permissions: ['activeTab', 'storage', 'nativeMessaging'],
    optional_permissions: ['cookies'],
    optional_host_permissions: ['http://*/*', 'https://*/*'],
    incognito: 'not_allowed',
    action: { default_title: 'VideoBridge · 下载当前视频' },
  },
});
