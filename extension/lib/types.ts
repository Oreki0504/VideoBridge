export const HOST_NAME = 'com.videobridge.host';
export const COOKIE_PERMISSION: chrome.permissions.Permissions = { permissions: ['cookies'], origins: ['http://*.youtube.com/*', 'https://*.youtube.com/*'] };
export type Site = string;
export const SITE_PERMISSIONS: Record<'youtube' | 'x', chrome.permissions.Permissions> = {
  youtube: COOKIE_PERMISSION,
  x: { permissions: ['cookies'], origins: ['http://*.x.com/*', 'https://*.x.com/*', 'http://*.twitter.com/*', 'https://*.twitter.com/*'] },
};

export type Format = 'best' | 'audio' | 'gif' | 'subtitles';
export type SubtitleLanguage = 'zh-en' | 'zh' | 'en' | 'ja' | 'all';
export type SubtitleFormat = 'srt' | 'vtt';
export type AudioFormat = 'original' | 'mp3' | 'm4a' | 'flac' | 'wav';
export type AudioQuality = 'best' | '128' | '192' | '256' | '320';
export type FragmentConcurrency = 1 | 4 | 8 | 16;
export interface Options { format: Format; maxHeight?: number; audioFormat?: AudioFormat; audioQuality?: AudioQuality; concurrentFragments?: FragmentConcurrency; gifStart?: number; gifDuration?: number; gifWidth?: number; gifFps?: number; subtitleLanguage?: SubtitleLanguage; subtitleFormat?: SubtitleFormat; subtitleAuto?: boolean }
export interface CookieSnapshot {
  name: string; value: string; domain: string; path: string;
  secure: boolean; httpOnly: boolean; hostOnly: boolean; session: boolean;
  expirationDate?: number;
}
export interface StartPayload {
  url: string; profileStoreId: string; cookieMode: 'always' | 'off';
  options: Options; cookies: CookieSnapshot[];
}
export interface HostEvent {
  version: 1; id: string; type: string;
  payload: Record<string, unknown>;
}
export interface Job {
  jobId: string; url: string; status: 'running' | 'completed' | 'failed' | 'cancelled';
  phase: string; percent?: number; speed?: number; eta?: number;
  error?: string; code?: string; files?: string[];
}
export interface Health {
  ytdlpCanRollback?: boolean;
  version?: string;
  capabilities?: string[];
  outputDir: string;
  tools: { name: string; available: boolean; path?: string; version?: string }[];
}
export type UpdateChannel = 'stable' | 'nightly';
export interface YTDLPUpdate { channel: UpdateChannel; currentVersion: string; latestVersion: string; updateAvailable: boolean; canRollback: boolean; supported: boolean; message: string }
export interface State { connected: boolean; jobs: Job[]; health?: Health; setupMessage?: string; installing?: boolean; ytdlpUpdate?: YTDLPUpdate }

export class BridgeError extends Error {
  constructor(public code: string, message: string) { super(message); }
}
