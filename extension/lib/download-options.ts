import type { Options } from './types';

export const HEIGHTS = [144, 240, 360, 480, 720, 1080, 1440, 2160, 4320];
export const FRAGMENT_CONCURRENCY = [1, 4, 8, 16];
export function isDownloadOptions(value: unknown): value is Options {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const o = value as Record<string, unknown>;
  if (Object.keys(o).some(key => !['format', 'maxHeight', 'audioFormat', 'audioQuality', 'concurrentFragments', 'gifStart', 'gifDuration', 'gifWidth', 'gifFps', 'subtitleLanguage', 'subtitleFormat', 'subtitleAuto'].includes(key))) return false;
  if (o.concurrentFragments !== undefined && (typeof o.concurrentFragments !== 'number' || !FRAGMENT_CONCURRENCY.includes(o.concurrentFragments))) return false;
  if (o.format === 'subtitles') {
    return ['maxHeight', 'audioFormat', 'audioQuality', 'gifStart', 'gifDuration', 'gifWidth', 'gifFps'].every(key => o[key] === undefined) &&
      (o.subtitleLanguage === undefined || (typeof o.subtitleLanguage === 'string' && ['zh-en', 'zh', 'en', 'ja', 'all'].includes(o.subtitleLanguage))) &&
      (o.subtitleFormat === undefined || o.subtitleFormat === 'srt' || o.subtitleFormat === 'vtt') &&
      (o.subtitleAuto === undefined || typeof o.subtitleAuto === 'boolean');
  }
  if (['subtitleLanguage', 'subtitleFormat', 'subtitleAuto'].some(key => o[key] !== undefined)) return false;
  if (o.format === 'gif') {
    return o.audioFormat === undefined && o.audioQuality === undefined && o.maxHeight === undefined &&
      (o.gifStart === undefined || (typeof o.gifStart === 'number' && Number.isInteger(o.gifStart) && o.gifStart >= 0 && o.gifStart <= 86400)) &&
      (o.gifDuration === undefined || (typeof o.gifDuration === 'number' && [5, 10, 15, 30, 60].includes(o.gifDuration))) &&
      (o.gifWidth === undefined || (typeof o.gifWidth === 'number' && [320, 480, 640].includes(o.gifWidth))) &&
      (o.gifFps === undefined || (typeof o.gifFps === 'number' && [8, 12, 15, 20].includes(o.gifFps)));
  }
  if (['gifStart', 'gifDuration', 'gifWidth', 'gifFps'].some(key => o[key] !== undefined)) return false;
  if (o.format === 'best') {
    return o.audioFormat === undefined && o.audioQuality === undefined &&
      (o.maxHeight === undefined || (typeof o.maxHeight === 'number' && HEIGHTS.includes(o.maxHeight)));
  }
  if (o.format !== 'audio' || o.maxHeight !== undefined) return false;
  const format = o.audioFormat ?? 'mp3'; // Compatibility with the first release.
  if (typeof format !== 'string' || !['original', 'mp3', 'm4a', 'flac', 'wav'].includes(format)) return false;
  if (o.audioQuality === undefined) return true;
  return format === 'mp3' && typeof o.audioQuality === 'string' && ['best', '128', '192', '256', '320'].includes(o.audioQuality);
}
