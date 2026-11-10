// Render the built popup in a fresh headless browser with synthetic extension APIs.
// This does not connect to the user's profile or validate real Cookie/Native APIs.
import assert from 'node:assert/strict';
import http from 'node:http';
import { readFile, mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { chromium } from 'playwright-core';

const base = fileURLToPath(new URL('../.output/chrome-mv3/', import.meta.url));
const server = http.createServer(async (req, res) => {
  const pathname = new URL(req.url, 'http://localhost').pathname;
  const target = path.resolve(base, '.' + pathname);
  if (!target.startsWith(base)) { res.writeHead(403).end(); return; }
  try {
    const mime = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css' }[path.extname(target)] ?? 'application/octet-stream';
    const data = await readFile(target);
    res.writeHead(200, { 'Content-Type': mime }); res.end(data);
  } catch { res.writeHead(404).end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const url = `http://127.0.0.1:${server.address().port}/popup.html`;

try {
  for (const channel of ['chrome', 'msedge']) {
    const browser = await chromium.launch({ channel, headless: true });
    try {
      const context = await browser.newContext({ viewport: { width: 400, height: 760 }, deviceScaleFactor: 1, locale: 'zh-CN' });
      const page = await context.newPage();
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      await page.addInitScript(() => {
        const allowed = JSON.parse(localStorage.getItem('fixture-allowed-sites') ?? '{}');
        const permissionKey = p => p.origins.some(o => o.includes('x.com')) ? 'x' : p.origins.some(o => o.includes('youtube.com')) ? 'youtube' : p.origins[0];
        const listeners = [];
        const storage = JSON.parse(localStorage.getItem('fixture-settings') ?? '{}');
        const state = { connected: false, jobs: [] };
        globalThis.fixture = { requests: [], publish: () => listeners.forEach(cb => cb({ type: 'state.changed', state })), state };
        globalThis.chrome = {
          permissions: { contains: async p => !!allowed[permissionKey(p)], request: async p => { allowed[permissionKey(p)] = true; localStorage.setItem('fixture-allowed-sites', JSON.stringify(allowed)); return true; }, remove: async p => { allowed[permissionKey(p)] = false; localStorage.setItem('fixture-allowed-sites', JSON.stringify(allowed)); globalThis.fixture.revoked = p; return true; } },
          storage: { local: { get: async () => storage, set: async value => {Object.assign(storage, value);localStorage.setItem('fixture-settings', JSON.stringify(storage));} } },
          tabs: { query: async () => [{ id: 42, title: '山间的晨光 · A quiet morning - YouTube', url: localStorage.getItem('fixture-tab') || 'https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=test' }] },
          runtime: {
            onMessage: { addListener: callback => listeners.push(callback) },
            sendMessage: async request => {
              globalThis.fixture.requests.push(request);
              if (request.type === 'host.check') {
                state.connected = true;
                state.health = { outputDir: 'D:\\Downloads\\VideoBridge', tools: ['yt-dlp', 'ffmpeg', 'ffprobe'].map(name => ({ name, available: true, version: 'test fixture' })) };
              }
              if (request.type.startsWith('ytdlp.')) {
                if (globalThis.fixture.updateFail) return {ok:false,code:'UPDATE_FAILED',message:'下载校验失败，当前版本已保留。'};
                state.ytdlpUpdate={channel:request.channel,currentVersion:request.type==='ytdlp.update'?'2026.10.26':'2026.09.27',latestVersion:request.type==='ytdlp.rollback'?'':'2026.10.26',updateAvailable:request.type==='ytdlp.check',canRollback:request.type==='ytdlp.update',supported:true,message:request.type==='ytdlp.update'?'yt-dlp 已更新':request.type==='ytdlp.rollback'?'已恢复上一版本':'发现新版本'};
              }
              if (request.type === 'download.start') {
                state.connected = true;
                state.jobs = [{ jobId: 'fixture-1', url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ', status: 'running', phase: '下载中', percent: 42.6, speed: 8388608, eta: 16 }];
              }
              if (request.type === 'download.cancel') state.jobs[0].status = 'cancelled';
              return { ok: true, data: state };
            },
          },
        };
      });
      await page.goto(url);
      await page.getByText('尚未授权 YouTube Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#download').isDisabled(), true);
      await page.locator('#grant').click();
      await page.getByText('已授权 YouTube Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#download').isEnabled(), true);
      await page.locator('#quality').selectOption('1080');
      await page.locator('#download').click();
      await page.locator('.task').waitFor();
      assert.equal(await page.locator('progress').getAttribute('value'), '42.6');
      const options = await page.evaluate(() => fixture.requests.find(r => r.type === 'download.start').options);
      assert.deepEqual(options, { format: 'best', maxHeight: 1080, concurrentFragments: 4 });
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await mkdir(new URL('../../docs/screenshots/', import.meta.url), { recursive: true });
      await page.screenshot({ path: fileURLToPath(new URL(`../../docs/screenshots/popup-${channel}.png`, import.meta.url)), fullPage: true });
      await page.getByRole('button', { name: '取消', exact: true }).click();
      await page.getByText('任务已停止，临时 Cookie 已清理。', { exact: true }).waitFor();
      const submitAndCancel = async expected => {
        await page.locator('#download').click();
        await page.getByRole('button', { name: '取消', exact: true }).waitFor();
        assert.deepEqual(await page.evaluate(() => fixture.requests.filter(r => r.type === 'download.start').at(-1).options), { concurrentFragments: 4, ...expected });
        await page.getByRole('button', { name: '取消', exact: true }).click();
      };
      assert.equal(await page.locator('#quality option').count(), 10);
      for (const maxHeight of [144, 240, 360, 480, 720, 1440, 2160, 4320]) {
        await page.locator('#quality').selectOption(String(maxHeight));
        await submitAndCancel({ format: 'best', maxHeight });
      }
      await page.locator('#quality').selectOption('best');
      await submitAndCancel({ format: 'best' });
      for (const concurrentFragments of [1,4,8,16]) {
        await page.locator('#concurrent-fragments').selectOption(String(concurrentFragments));
        await submitAndCancel({ format: 'best', concurrentFragments });
      }
      await page.locator('#concurrent-fragments').selectOption('4');
      await page.locator('#download-type').selectOption('audio');
      assert.equal(await page.locator('#video-options').isHidden(), true);
      assert.equal(await page.locator('#audio-quality-group').isHidden(), true);
      await submitAndCancel({ format: 'audio', audioFormat: 'original' });
      for (const concurrentFragments of [1,4,8,16]) {
        await page.locator('#concurrent-fragments').selectOption(String(concurrentFragments));
        await submitAndCancel({ format: 'audio', audioFormat: 'original', concurrentFragments });
      }
      await page.locator('#concurrent-fragments').selectOption('4');
      for (const audioFormat of ['mp3', 'm4a', 'flac', 'wav']) {
        await page.locator('#audio-format').selectOption(audioFormat);
        if (audioFormat === 'mp3') {
          for (const audioQuality of ['best', '128', '192', '256', '320']) {
            await page.locator('#audio-quality').selectOption(audioQuality);
            await submitAndCancel({ format: 'audio', audioFormat, audioQuality });
          }
        } else {
          assert.equal(await page.locator('#audio-quality-group').isHidden(), true);
          await submitAndCancel({ format: 'audio', audioFormat });
        }
      }
      await page.locator('#audio-format').selectOption('mp3');
      await page.locator('#audio-quality').selectOption('192');
      await page.locator('#concurrent-fragments').selectOption('8');
      await page.reload();
      await page.getByText('已授权 YouTube Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#download-type').inputValue(), 'audio');
      assert.equal(await page.locator('#audio-format').inputValue(), 'mp3');
      assert.equal(await page.locator('#audio-quality').inputValue(), '192');
      assert.equal(await page.locator('#concurrent-fragments').inputValue(), '8');
      await submitAndCancel({ format: 'audio', audioFormat: 'mp3', audioQuality: '192', concurrentFragments: 8 });
      await page.screenshot({ path: fileURLToPath(new URL(`../../docs/screenshots/audio-${channel}.png`, import.meta.url)), fullPage: true });
      await page.evaluate(() => { const settings=JSON.parse(localStorage.getItem('fixture-settings')); delete settings.downloadOptions.concurrentFragments; localStorage.setItem('fixture-settings',JSON.stringify(settings)); });
      await page.reload();
      await page.getByText('已授权 YouTube Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#concurrent-fragments').inputValue(), '4');
      assert.equal(await page.locator('#audio-quality').inputValue(), '192');
      await page.locator('#revoke').click();
      await page.getByText('尚未授权 YouTube Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#download').isDisabled(), true);
      await page.locator('#use-cookies').uncheck();
      assert.equal(await page.locator('#download').isEnabled(), true);
      await page.locator('summary').click();
      await page.locator('#check').click();
      await page.getByText('本地桥接 · 已连接', { exact: true }).waitFor();
      assert.match(await page.locator('#tools').textContent(), /ffprobe/);
      assert.equal(await page.locator('#update-install').isDisabled(),true);
      await page.locator('#update-check').click();
      await page.getByText(/当前：2026.09.27 · 最新：2026.10.26/).waitFor();
      assert.equal(await page.locator('#update-install').isEnabled(),true);
      await page.locator('#update-install').click();
      await page.getByText(/yt-dlp 已更新/).waitFor();
      assert.equal(await page.locator('#update-rollback').isEnabled(),true);
      assert.equal(await page.locator('#update-install').isDisabled(),true);
      await page.locator('#update-rollback').click();
      await page.getByText(/已恢复上一版本/).waitFor();
      assert.equal(await page.locator('#update-rollback').isDisabled(),true);
      await page.locator('#update-channel').selectOption('nightly');
      assert.equal(await page.locator('#update-install').isDisabled(),true);
      await page.locator('#update-check').click();
      assert.deepEqual(await page.evaluate(()=>fixture.requests.filter(r=>r.type==='ytdlp.check').at(-1)),{type:'ytdlp.check',channel:'nightly'});
      await page.evaluate(()=>{fixture.state.installing=true;fixture.state.setupMessage='下载并校验 yt-dlp…';fixture.publish();});
      assert.equal(await page.locator('#download').isDisabled(),true);
      assert.equal(await page.locator('#update-check').isDisabled(),true);
      assert.equal(await page.locator('#update-channel').isDisabled(),true);
      await page.evaluate(()=>{fixture.state.installing=false;fixture.state.setupMessage='';fixture.publish();});
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
      await page.screenshot({path:fileURLToPath(new URL(`../../docs/screenshots/updater-${channel}.png`,import.meta.url)),fullPage:true});
      await page.evaluate(()=>{fixture.updateFail=true;});
      await page.locator('#update-install').click();
      await page.getByText('下载校验失败，当前版本已保留。 (UPDATE_FAILED)',{exact:true}).waitFor();
      assert.equal(await page.locator('#update-check').isEnabled(),true);
      // After reconnecting offline, local health alone must allow rollback.
      await page.evaluate(()=>{fixture.state.ytdlpUpdate=undefined;fixture.state.health.ytdlpCanRollback=true;fixture.publish();});
      assert.equal(await page.locator('#update-rollback').isEnabled(),true);
      assert.equal(await page.locator('.version').textContent(),'0.9.0');
      await page.reload();
      assert.equal(await page.locator('#update-channel').inputValue(),'nightly');
      await page.locator('#grant').click();
      await page.evaluate(() => localStorage.setItem('fixture-tab','https://x.com/someone/status/123/video/2?s=20'));
      await page.reload();
      await page.getByText('尚未授权 X / Twitter Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(), false);
      assert.equal(await page.locator('#download').isEnabled(), true);
      assert.equal(await page.locator('#video-url').textContent(), 'https://x.com/i/web/status/123/video/2');
      await page.locator('#download-type').selectOption('gif');
      await page.locator('#concurrent-fragments').selectOption('4');
      assert.equal(await page.locator('#video-options').isHidden(), true);
      assert.equal(await page.locator('#audio-options').isHidden(), true);
      await submitAndCancel({format:'gif',gifStart:0,gifDuration:15,gifWidth:480,gifFps:12});
      await page.locator('#gif-start').fill('3');
      await page.locator('#gif-duration').selectOption('5');
      await page.locator('#gif-width').selectOption('320');
      await page.locator('#gif-fps').selectOption('8');
      await submitAndCancel({format:'gif',gifStart:3,gifDuration:5,gifWidth:320,gifFps:8});
      await page.reload();
      await page.getByText('尚未授权 X / Twitter Cookie', { exact: true }).waitFor();
      assert.equal(await page.locator('#gif-start').inputValue(),'3');
      await page.locator('#use-cookies').check();
      assert.equal(await page.locator('#download').isDisabled(),true);
      await page.locator('#grant').click();
      await page.getByText('已授权 X / Twitter Cookie', { exact: true }).waitFor();
      await page.locator('#revoke').click();
      await page.getByText('尚未授权 X / Twitter Cookie', { exact: true }).waitFor();
      assert.equal(await page.evaluate(() => 'permissions' in fixture.revoked),false);
      assert.equal(await page.evaluate(() => JSON.parse(localStorage.getItem('fixture-allowed-sites')).youtube),true);
      await page.locator('#use-cookies').uncheck();
      await page.evaluate(() => {fixture.state.installing=true;fixture.state.setupMessage='下载并校验 FFmpeg…';fixture.publish();});
      await page.getByText('下载并校验 FFmpeg…', {exact:true}).waitFor();
      assert.equal(await page.locator('#download').isDisabled(),true);
      await page.evaluate(() => {fixture.state.installing=false;fixture.state.setupMessage='';fixture.publish();});
      await page.screenshot({ path: fileURLToPath(new URL(`../../docs/screenshots/x-gif-${channel}.png`, import.meta.url)), fullPage: true });
      await page.evaluate(() => localStorage.setItem('fixture-tab','https://www.bilibili.com/video/BV13x41117TL?p=2'));
      await page.reload();
      await page.getByText('尚未授权 Bilibili Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),false);
      assert.equal(await page.locator('#download').isEnabled(),true);
      assert.match(await page.locator('#video-url').textContent(),/\?p=2$/);
      await page.locator('#grant').click();
      await page.getByText('已授权 Bilibili Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),true);
      assert.equal(await page.evaluate(()=>JSON.parse(localStorage.getItem('fixture-settings'))['useCookies:bilibili.com']),true);
      await page.evaluate(() => localStorage.setItem('fixture-tab','https://www.nicovideo.jp/watch/sm9'));
      await page.reload();
      await page.getByText('尚未授权 Niconico Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),false);
      await page.locator('#grant').click();
      await page.getByText('已授权 Niconico Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),true);
      await page.locator('#use-cookies').uncheck();
      await page.reload();
      await page.getByText('已授权 Niconico Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),false);
      await page.locator('#download-type').selectOption('best');
      await page.locator('#quality').selectOption('best');
      await submitAndCancel({format:'best'});
      await page.screenshot({path:fileURLToPath(new URL(`../../docs/screenshots/multisite-${channel}.png`,import.meta.url)),fullPage:true});
      await page.evaluate(() => localStorage.setItem('fixture-tab','https://www.bilibili.com/video/BV13x41117TL?p=2'));
      await page.reload();await page.getByText('已授权 Bilibili Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#use-cookies').isChecked(),true);
      await page.locator('#lang-toggle').click();
      await page.getByText('Bilibili Cookie granted',{exact:true}).waitFor();
      assert.equal(await page.locator('#download-label').textContent(),'Download Current Media');
      assert.equal(await page.locator('#cookie-label').textContent(),'Use current Bilibili session');
      await page.reload();
      await page.getByText('Bilibili Cookie granted',{exact:true}).waitFor();
      assert.equal(await page.locator('#lang-toggle').textContent(),'中文');
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      await page.locator('#download').click();
      await page.getByRole('button', { name: 'Cancel', exact: true }).click();
      await page.getByText('Task stopped and temporary cookies cleaned up.', { exact: true }).waitFor();
      await page.locator('#download-type').selectOption('subtitles');
      assert.equal(await page.locator('#video-options').isHidden(),true);
      assert.equal(await page.locator('#audio-options').isHidden(),true);
      assert.equal(await page.locator('#gif-options').isHidden(),true);
      assert.equal(await page.locator('#subtitle-options').isVisible(),true);
      assert.equal(await page.locator('#download-label').textContent(),'Download Subtitles');
      for (const subtitleLanguage of ['zh-en','zh','en','ja','all']) {
        await page.locator('#subtitle-language').selectOption(subtitleLanguage);
        await page.locator('#download').click();
        await page.getByRole('button',{name:'Cancel',exact:true}).waitFor();
        assert.deepEqual(await page.evaluate(()=>fixture.requests.filter(request=>request.type==='download.start').at(-1).options),
          {format:'subtitles',concurrentFragments:4,subtitleLanguage,subtitleFormat:'srt',subtitleAuto:true});
        await page.getByRole('button',{name:'Cancel',exact:true}).click();
      }
      await page.locator('#subtitle-format').selectOption('vtt');
      await page.locator('#subtitle-auto').uncheck();
      await page.reload();
      await page.getByText('Bilibili Cookie granted',{exact:true}).waitFor();
      assert.equal(await page.locator('#download-type').inputValue(),'subtitles');
      assert.equal(await page.locator('#subtitle-language').inputValue(),'all');
      assert.equal(await page.locator('#subtitle-format').inputValue(),'vtt');
      assert.equal(await page.locator('#subtitle-auto').isChecked(),false);
      await page.evaluate(()=>{
        fixture.state.jobs=[{jobId:'subs',status:'completed',phase:'已完成',percent:100,files:['D:\\Downloads\\VideoBridge\\Example.zh-Hans.srt']}];fixture.publish();
      });
      await page.getByText('Completed',{exact:true}).waitFor();
      await page.getByText('D:\\Downloads\\VideoBridge\\Example.zh-Hans.srt',{exact:true}).waitFor();
      await page.evaluate(()=>{
        fixture.state.jobs=[{jobId:'subs',status:'failed',code:'SUBTITLES_UNAVAILABLE',error:'此视频没有所选语言的可用字幕；可尝试其他语言或开启自动字幕。Bilibili 弹幕不属于字幕。'}];fixture.publish();
      });
      await page.getByText('Download failed',{exact:true}).waitFor();
      assert.match(await page.locator('.task p').textContent(),/^No subtitles.*SUBTITLES_UNAVAILABLE/);
      assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),true);
      await page.screenshot({path:fileURLToPath(new URL(`../../docs/screenshots/subtitles-en-${channel}.png`,import.meta.url)),fullPage:true});
      await page.locator('#lang-toggle').click();
      await page.getByText('已授权 Bilibili Cookie',{exact:true}).waitFor();
      assert.equal(await page.locator('#download-label').textContent(),'下载字幕');
      await page.screenshot({path:fileURLToPath(new URL(`../../docs/screenshots/subtitles-zh-${channel}.png`,import.meta.url)),fullPage:true});
      await page.locator('#theme-toggle').click();
      assert.equal(await page.locator('html').getAttribute('data-theme'),'light');
      assert.equal(await page.locator('#theme-toggle').textContent(),'☾ 深色');
      await page.reload();
      assert.equal(await page.locator('html').getAttribute('data-theme'),'light');
      await page.locator('.mode-tab[data-mode="best"]').click();
      await page.evaluate(()=>{
        fixture.state.jobs=[{jobId:'light-1',status:'running',phase:'下载中',percent:64.2,speed:10485760,eta:9}];fixture.publish();
      });
      await page.screenshot({path:fileURLToPath(new URL(`../../docs/screenshots/popup-light-${channel}.png`,import.meta.url)),fullPage:true});
      await page.locator('#theme-toggle').click();
      assert.equal(await page.locator('html').getAttribute('data-theme'),'dark');
      assert.deepEqual(errors, []);
      console.log(`PASS ${channel}: video/audio/GIF options, X URL and anonymous default, site permissions, concurrency, persistence, setup progress, cancel and layout; browser ${browser.version()}`);
    } finally { await browser.close(); }
  }
} finally { server.close(); }
