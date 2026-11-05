import { describe, expect, it } from 'vitest';
import { HEIGHTS, FRAGMENT_CONCURRENCY, isDownloadOptions } from './download-options';

describe('download option boundary', () => {
  it('accepts subtitle presets and rejects injected selectors and media-only settings', () => {
    expect(isDownloadOptions({format:'subtitles'})).toBe(true);
    for (const subtitleLanguage of ['zh-en','zh','en','ja','all']) for (const subtitleFormat of ['srt','vtt']) {
      expect(isDownloadOptions({format:'subtitles',subtitleLanguage,subtitleFormat,subtitleAuto:true})).toBe(true);
    }
    for (const bad of [{subtitleLanguage:'.*'},{subtitleFormat:'../../file'},{subtitleAuto:'true'},{maxHeight:1080},{audioFormat:'mp3'},{gifStart:1}]) {
      expect(isDownloadOptions({format:'subtitles',...bad})).toBe(false);
    }
    expect(isDownloadOptions({format:'best',subtitleLanguage:'en'})).toBe(false);
    expect(isDownloadOptions({format:'gif',subtitleAuto:true})).toBe(false);
  });
  it('accepts bounded GIF segments and rejects contradictory settings', () => {
    expect(isDownloadOptions({format:'gif'})).toBe(true);
    expect(isDownloadOptions({format:'gif',gifStart:2,gifDuration:5,gifWidth:320,gifFps:8})).toBe(true);
    for (const bad of [{gifStart:-1},{gifStart:NaN},{gifStart:0.5},{gifDuration:61},{gifWidth:1920},{gifFps:60},{audioFormat:'mp3'},{maxHeight:720}]) expect(isDownloadOptions({format:'gif',...bad})).toBe(false);
    expect(isDownloadOptions({format:'best',gifDuration:15})).toBe(false);
  });
  it('supports bounded fragment concurrency for video and audio, including older settings', () => {
    for (const concurrentFragments of FRAGMENT_CONCURRENCY) {
      expect(isDownloadOptions({ format: 'best', concurrentFragments })).toBe(true);
      expect(isDownloadOptions({ format: 'audio', audioFormat: 'original', concurrentFragments })).toBe(true);
    }
    expect(isDownloadOptions({ format: 'best', maxHeight: 2160 })).toBe(true);
  });
  it('accepts all supported resolution caps and unlimited best', () => {
    expect(isDownloadOptions({ format: 'best' })).toBe(true);
    for (const maxHeight of HEIGHTS) expect(isDownloadOptions({ format: 'best', maxHeight })).toBe(true);
  });
  it('accepts original and lossless formats without bitrate and lossy formats with quality', () => {
    for (const audioFormat of ['original', 'm4a', 'flac', 'wav']) expect(isDownloadOptions({ format: 'audio', audioFormat })).toBe(true);
    for (const audioFormat of ['mp3']) for (const audioQuality of ['best', '128', '192', '256', '320']) {
      expect(isDownloadOptions({ format: 'audio', audioFormat, audioQuality })).toBe(true);
    }
    expect(isDownloadOptions({ format: 'audio' })).toBe(true);
  });
  it.each([
    { format: 'best', maxHeight: 999 }, { format: 'best', maxHeight: '1080' },
    { format: 'best', audioFormat: 'mp3' }, { format: 'audio', maxHeight: 1080 },
    { format: 'audio', audioFormat: 'original', audioQuality: '320' },
    { format: 'audio', audioFormat: 'flac', audioQuality: 'best' },
    { format: 'audio', audioFormat: '--exec' }, { format: 'audio', audioQuality: '999' },
    { format: 'audio', audioQuality: 128 }, { format: 'best', outputDir: 'C:/' }, null,
    ...[0, -1, 2, 32, 1.5, '4', null].map(concurrentFragments => ({format: 'best', concurrentFragments})),
  ])('rejects invalid and contradictory settings: %j', options => expect(isDownloadOptions(options)).toBe(false));
});
