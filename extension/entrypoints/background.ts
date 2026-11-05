import { readCookies } from '../lib/cookie-provider';
import { isDownloadOptions } from '../lib/download-options';
import { NativeClient } from '../lib/native-client';
import { resolveVideo, cookieSettingKey, usesCookies } from '../lib/site-resolver';
import { BridgeError, type Health, type HostEvent, type Job, type State, type YTDLPUpdate } from '../lib/types';

export default defineBackground(() => {
  const jobs = new Map<string, Job>();
  let health: Health | undefined;
  let starting = false;
  let setup: Promise<void> | undefined;
  let setupMessage = '';
  let ytdlpUpdate: YTDLPUpdate | undefined;
  const client = new NativeClient(onEvent, () => {
    for (const job of jobs.values()) if (job.status === 'running') {
      job.status = 'failed'; job.code = 'HOST_DISCONNECTED'; job.error = '本地连接中断，任务已停止。';
    }
    broadcast();
  });
  function state(): State { return { connected: client.connected, jobs: [...jobs.values()], health, installing: !!setup, setupMessage, ytdlpUpdate }; }
  function broadcast() { void chrome.runtime.sendMessage({ type: 'state.changed', state: state() }).catch(() => {}); }
  function onEvent(e: HostEvent) {
    if (e.type === 'tools.progress') setupMessage = String(e.payload.message ?? '正在安装依赖…');
    if (e.type === 'host.status') health = e.payload as unknown as Health;
    if (e.type === 'ytdlp.status') {
      health = e.payload.health as Health;
      ytdlpUpdate = e.payload.update as YTDLPUpdate;
    }
    const id = typeof e.payload.jobId === 'string' ? e.payload.jobId : '';
    if (e.type === 'download.accepted') {
      jobs.set(id, { jobId: id, url: String(e.payload.url), status: 'running', phase: '准备下载' });
      if (jobs.size > 30) jobs.delete(jobs.keys().next().value!);
    }
    const job = jobs.get(id);
    if (job) {
      if (e.type === 'download.progress') {
        job.phase = String(e.payload.phase ?? '下载中');
        if (typeof e.payload.percent === 'number') job.percent = e.payload.percent;
        if (typeof e.payload.speed === 'number') job.speed = e.payload.speed;
        if (typeof e.payload.eta === 'number') job.eta = e.payload.eta;
      }
      if (e.type === 'download.completed') { job.status = 'completed'; job.percent = 100; job.phase = '已完成'; job.files = e.payload.files as string[]; }
      if (e.type === 'download.failed') { job.status = 'failed'; job.code = String(e.payload.code); job.error = String(e.payload.message); }
      if (e.type === 'download.cancelled') { job.status = 'cancelled'; job.phase = '已取消'; }
    }
    broadcast();
  }
  async function ensureTools() {
    if (setup) return setup;
    setup = (async () => {
      await client.request('host.status');
      if (health?.tools.some(t => !t.available)) {
        if (!health.capabilities?.includes('tools.ensure')) throw new BridgeError('HOST_UPDATE_REQUIRED', '本地 Host 版本过旧，请安装新版 Host 后重新加载扩展。');
        setupMessage = '正在自动补齐 yt-dlp / FFmpeg…'; broadcast();
        await client.request('host.tools.ensure', {}, 16 * 60_000);
      }
    })();
    broadcast();
    try { await setup; setupMessage = ''; }
    catch (e) { setupMessage = e instanceof Error ? e.message : '依赖安装失败，请重试。'; throw e; }
    finally { setup = undefined; broadcast(); }
  }
  async function handle(message: Record<string, unknown>) {
    switch (message.type) {
      case 'state.get': return state();
      case 'host.check': await ensureTools(); return state();
      case 'ytdlp.check':
      case 'ytdlp.update':
      case 'ytdlp.rollback': {
        if (message.channel !== 'stable' && message.channel !== 'nightly') throw new BridgeError('INVALID_REQUEST','请选择有效的更新通道。');
        if (starting || setup || [...jobs.values()].some(j => j.status === 'running')) throw new BridgeError('BUSY','请等待当前下载或工具操作完成。');
        setupMessage = message.type === 'ytdlp.check' ? '正在查询 yt-dlp 官方版本…' : message.type === 'ytdlp.rollback' ? '正在恢复上一版本…' : '正在更新 yt-dlp…';
        ytdlpUpdate = undefined;
        setup = (async () => {
          await client.request('host.status');
          if (!health?.capabilities?.includes('ytdlp.update')) throw new BridgeError('HOST_UPDATE_REQUIRED','yt-dlp 更新管理需要 v0.6 或更新的 Host，请安装新版并重新加载扩展。');
          await client.request(`host.${message.type}`, {channel:message.channel}, message.type === 'ytdlp.check' ? 60_000 : 16 * 60_000);
        })();
        broadcast();
        try { await setup; setupMessage = ''; }
        catch (e) { setupMessage = e instanceof Error ? e.message : '工具操作失败，当前版本已保留。'; throw e; }
        finally { setup = undefined; broadcast(); }
        return state();
      }
      case 'download.cancel':
        if (typeof message.jobId !== 'string') throw new BridgeError('INVALID_REQUEST', '任务 ID 无效。');
        await client.request('download.cancel', { jobId: message.jobId }); return state();
      case 'download.start': {
        if (starting || setup || [...jobs.values()].some(j => j.status === 'running')) throw new BridgeError('BUSY', '请等待当前任务完成或先取消。');
        starting = true;
        try {
          const options = message.options;
          if (!isDownloadOptions(options)) {
            throw new BridgeError('INVALID_OPTIONS', '下载选项无效。');
          }
          const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
          if (!tab?.url) throw new BridgeError('NO_ACTIVE_TAB', '无法读取当前标签页。');
          const { url, site } = resolveVideo(tab.url);
          const settings = await chrome.storage.local.get(cookieSettingKey(site));
          const useCookies = usesCookies(settings,site);
          if (useCookies && !url.startsWith('https:')) throw new BridgeError('HTTPS_REQUIRED','HTTP 页面仅支持匿名下载；请打开 HTTPS 页面后使用会话。');
          await ensureTools(); // Capture the intended URL first; read credentials only after setup.
          if (options.format === 'subtitles' && !health?.capabilities?.includes('subtitles')) {
            throw new BridgeError('HOST_UPDATE_REQUIRED', '字幕下载需要 v0.8 或更新的 Host，请安装新版后重新加载扩展。');
          }
          if ((site === 'x' && !health?.capabilities?.includes('x')) || (options.format === 'gif' && !health?.capabilities?.includes('gif'))) {
            throw new BridgeError('HOST_UPDATE_REQUIRED', 'X / GIF 需要 v0.4 或更新的 Host，请完成本地更新后重新加载扩展。');
          }
          if (!['youtube','x'].includes(site) && !health?.capabilities?.includes('generic-sites')) throw new BridgeError('HOST_UPDATE_REQUIRED','其他媒体网站需要 v0.5 或更新的 Host，请安装新版后重新加载扩展。');
          const snapshot = useCookies ? await readCookies(tab, site) : { storeId: '', cookies: [] };
          try {
            await client.request('download.start', { url, options, profileStoreId: snapshot.storeId, cookieMode: useCookies ? 'always' : 'off', cookies: snapshot.cookies });
          } finally { snapshot.cookies.length = 0; }
          return state();
        } finally { starting = false; }
      }
      default: throw new BridgeError('INVALID_REQUEST', '不支持的请求。');
    }
  }
  chrome.runtime.onMessage.addListener((message, sender, reply) => {
    // Only extension pages may invoke the bridge; never accept content-script requests.
    if (sender.id !== chrome.runtime.id || sender.tab || !sender.url?.startsWith(chrome.runtime.getURL(''))) return false;
    if (!message || typeof message !== 'object' || message.type === 'state.changed') return false;
    void handle(message).then(data => reply({ ok: true, data })).catch(e => reply({
      ok: false, code: e instanceof BridgeError ? e.code : 'BRIDGE_ERROR',
      message: e instanceof BridgeError ? e.message : '操作失败，请检查权限与本地安装。',
    }));
    return true;
  });
});
