import './style.css';
import { resolveVideo, sitePermission, cookieSettingKey, usesCookies } from '../../lib/site-resolver';
import { isDownloadOptions } from '../../lib/download-options';
import {
  type Locale,
  UI_TEXT,
  formatCookieLabel,
  formatEta,
  formatPermissionState,
  formatSiteLabel,
  formatToolsHealth,
  formatUpdateStatus,
  resolveLocale,
  translateDynamicText,
} from '../../lib/i18n';
import { type Site, type AudioFormat, type AudioQuality, type FragmentConcurrency, type Options, type State, type SubtitleLanguage, type SubtitleFormat } from '../../lib/types';

const el = <T extends HTMLElement = HTMLElement>(id: string) => document.getElementById(id) as T;
const download = el<HTMLButtonElement>('download');
const cookies = el<HTMLInputElement>('use-cookies');
let validVideo = false;
let busy = false;
let submitting = false;
let permitted = false;
let site: Site = 'youtube';
let securePage = true;
let lastState: State | undefined;
let toolSubmitting = false;
let locale: Locale = 'zh';
let rawMessage = '';

function setSelectOptionLabels(selectId: string, labels: Record<string, string>) {
  const select = el<HTMLSelectElement>(selectId);
  for (const option of Array.from(select.options)) {
    if (labels[option.value]) option.textContent = labels[option.value]!;
  }
}

function selectedOptions(): Options {
  const concurrentFragments = Number(el<HTMLSelectElement>('concurrent-fragments').value) as FragmentConcurrency;
  if (el<HTMLSelectElement>('download-type').value === 'subtitles') {
    return { format: 'subtitles', concurrentFragments,
      subtitleLanguage: el<HTMLSelectElement>('subtitle-language').value as SubtitleLanguage,
      subtitleFormat: el<HTMLSelectElement>('subtitle-format').value as SubtitleFormat,
      subtitleAuto: el<HTMLInputElement>('subtitle-auto').checked };
  }
  if (el<HTMLSelectElement>('download-type').value === 'gif') {
    return { format: 'gif', concurrentFragments, gifStart: el<HTMLInputElement>('gif-start').value === '' ? NaN : Number(el<HTMLInputElement>('gif-start').value), gifDuration: Number(el<HTMLSelectElement>('gif-duration').value), gifWidth: Number(el<HTMLSelectElement>('gif-width').value), gifFps: Number(el<HTMLSelectElement>('gif-fps').value) };
  }
  if (el<HTMLSelectElement>('download-type').value === 'audio') {
    const audioFormat = el<HTMLSelectElement>('audio-format').value as AudioFormat;
    return { format: 'audio', audioFormat, concurrentFragments, ...(audioFormat === 'mp3' ? { audioQuality: el<HTMLSelectElement>('audio-quality').value as AudioQuality } : {}) };
  }
  const quality = el<HTMLSelectElement>('quality').value;
  return { format: 'best', concurrentFragments, ...(quality !== 'best' ? { maxHeight: Number(quality) } : {}) };
}

function updateOptions() {
  const t = UI_TEXT[locale];
  const audio = el<HTMLSelectElement>('download-type').value === 'audio';
  const gif = el<HTMLSelectElement>('download-type').value === 'gif';
  const subtitles = el<HTMLSelectElement>('download-type').value === 'subtitles';
  const format = el<HTMLSelectElement>('audio-format').value;
  el('video-options').hidden = audio || gif || subtitles;
  el('subtitle-options').hidden = !subtitles;
  el('download-label').textContent = subtitles ? t.subtitleDownload : t.downloadButton;
  el('gif-options').hidden = !gif;
  el('audio-options').hidden = !audio;
  el('audio-quality-group').hidden = format !== 'mp3';
  el('audio-hint').textContent = format === 'original'
    ? t.audioHints.original
    : ['flac', 'wav'].includes(format)
      ? t.audioHints.lossless
      : format === 'm4a' ? t.audioHints.m4a
        : t.audioHints.mp3;
}

function applyLocale() {
  const t = UI_TEXT[locale];
  document.documentElement.lang = t.htmlLang;
  const toggle = el<HTMLButtonElement>('lang-toggle');
  toggle.textContent = t.langToggleLabel;
  toggle.title = t.langToggleTitle;
  toggle.setAttribute('aria-label', t.langToggleTitle);
  el('brand-subtitle').textContent = t.brandSubtitle;
  el('site-label').textContent = formatSiteLabel(site, locale);
  if (!validVideo) {
    el('video-title').textContent = t.invalidPageTitle;
    el('video-url').textContent = t.invalidPageUrl;
  }
  el('x-hint').textContent = t.xHint;
  el('download-type-label').textContent = t.downloadTypeLabel;
  setSelectOptionLabels('download-type', t.downloadTypes);
  el('subtitle-language-label').textContent = t.subtitleLanguageLabel;
  setSelectOptionLabels('subtitle-language', t.subtitleLanguages);
  el('subtitle-format-label').textContent = t.subtitleFormatLabel;
  el('subtitle-auto-label').textContent = t.subtitleAutoLabel;
  el('subtitle-hint').textContent = t.subtitleHint;
  el('quality-label').textContent = t.qualityLabel;
  setSelectOptionLabels('quality', t.qualities);
  el('quality-hint').textContent = t.qualityHint;
  el('audio-format-label').textContent = t.audioFormatLabel;
  setSelectOptionLabels('audio-format', t.audioFormats);
  el('audio-quality-label').textContent = t.audioQualityLabel;
  setSelectOptionLabels('audio-quality', t.audioQualities);
  el('gif-start-label').textContent = t.gifStartLabel;
  el('gif-duration-label').textContent = t.gifDurationLabel;
  setSelectOptionLabels('gif-duration', t.gifDurations);
  el('gif-width-label').textContent = t.gifWidthLabel;
  el('gif-fps-label').textContent = t.gifFpsLabel;
  el('gif-hint').textContent = t.gifHint;
  el('concurrency-label').textContent = t.concurrencyLabel;
  setSelectOptionLabels('concurrent-fragments', t.concurrencies);
  el('concurrency-hint').textContent = t.concurrencyHint;
  el('cookie-label').textContent = formatCookieLabel(site, locale);
  el('cookie-hint').textContent = t.cookieHint;
  el('permission-state').textContent = formatPermissionState(site, permitted, locale);
  el('grant').textContent = t.grant;
  el('revoke').textContent = t.revoke;
  el('download-label').textContent = t.downloadButton;
  el('check').textContent = t.checkDeps;
  if (!lastState?.health) el('tools').textContent = t.toolsDefault;
  el('update-channel-label').textContent = t.updateChannelLabel;
  setSelectOptionLabels('update-channel', t.updateChannels);
  el('update-check').textContent = t.updateCheck;
  el('update-install').textContent = t.updateInstall;
  el('update-rollback').textContent = t.updateRollback;
  el('update-note').textContent = t.updateNote;
  el('background-note').textContent = t.backgroundNote;
  el('scope-note').textContent = t.scopeNote;
  el('footer-left').textContent = t.footerLeft;
  updateOptions();
  if (rawMessage) message(rawMessage);
  if (lastState) render(lastState);
}

for (const id of ['download-type', 'quality', 'audio-format', 'audio-quality', 'concurrent-fragments', 'gif-start', 'gif-duration', 'gif-width', 'gif-fps', 'subtitle-language', 'subtitle-format', 'subtitle-auto']) {
  el<HTMLSelectElement>(id).onchange = () => {
    updateOptions();
    updateButton();
    const options = selectedOptions();
    if (isDownloadOptions(options)) void chrome.storage.local.set({ downloadOptions: options }).catch(() => message(UI_TEXT[locale].optionSaveFailed));
  };
}

function message(text = '') {
  rawMessage = text;
  const translated = translateDynamicText(text, locale);
  el('message').textContent = translated;
  el('message').hidden = !translated;
}
function updateButton() { download.disabled = !validVideo || busy || submitting || !isDownloadOptions(selectedOptions()) || (cookies.checked && !permitted); }
async function send(type: string, payload: Record<string, unknown> = {}): Promise<State> {
  const reply = await chrome.runtime.sendMessage({ type, ...payload });
  if (!reply?.ok) throw new Error(`${translateDynamicText(reply?.message ?? UI_TEXT[locale].bridgeNoResponse, locale)}${reply?.code ? ` (${reply.code})` : ''}`);
  return reply.data;
}
async function permissionState() {
  permitted = await chrome.permissions.contains(sitePermission(site));
  el('permission-state').textContent = formatPermissionState(site, permitted, locale);
  el('grant').hidden = permitted || !securePage; el('revoke').hidden = !permitted;
  updateButton();
}
function render(state: State) {
  lastState = state;
  const t = UI_TEXT[locale];
  busy = toolSubmitting || !!state.installing || state.jobs.some(j => j.status === 'running'); updateButton();
  const setupText = translateDynamicText(state.setupMessage ?? '', locale);
  el('setup-message').textContent = setupText;
  el('setup-message').hidden = !setupText;
  el<HTMLButtonElement>('check').disabled = busy;
  el<HTMLButtonElement>('update-check').disabled = busy;
  el<HTMLSelectElement>('update-channel').disabled = busy;
  const update = state.ytdlpUpdate;
  const matching = update?.channel === el<HTMLSelectElement>('update-channel').value;
  el<HTMLButtonElement>('update-install').disabled = busy || !matching || !update?.updateAvailable;
  el<HTMLButtonElement>('update-rollback').disabled = busy || !(update?.canRollback ?? state.health?.ytdlpCanRollback);
  el('update-status').textContent = update && matching
    ? formatUpdateStatus(update, locale)
    : t.updateStatusDefault;
  el('host-status').textContent = state.connected ? t.hostConnected : t.hostDisconnected;
  if (state.health) {
    el('tools').textContent = formatToolsHealth(state.health, locale);
  }
  const container = el('tasks'); container.replaceChildren();
  for (const job of state.jobs.slice(-3).reverse()) {
    const article = document.createElement('article'); article.className = 'task';
    const head = document.createElement('div'); head.className = 'task-head';
    const title = document.createElement('span'); title.textContent = job.status === 'failed' ? t.downloadFailed : translateDynamicText(job.phase, locale); head.append(title);
    if (job.status === 'running') {
      const cancel = document.createElement('button'); cancel.className = 'text-button'; cancel.textContent = t.cancel;
      cancel.onclick = async () => { cancel.disabled = true; try { render(await send('download.cancel', { jobId: job.jobId })); } catch (e) { message((e as Error).message); cancel.disabled = false; } };
      head.append(cancel);
    }
    article.append(head);
    if (job.status === 'running' || job.status === 'completed') {
      const progress = document.createElement('progress'); progress.max = 100;
      if (job.percent !== undefined) progress.value = job.percent;
      progress.setAttribute('aria-label', t.downloadProgressAria); article.append(progress);
    }
    const detail = document.createElement('p');
    detail.textContent = job.error ? `${translateDynamicText(job.error, locale)} (${job.code})` : job.files?.join('\n') ?? `${job.percent?.toFixed(1) ?? '…'}%${job.speed ? ` · ${(job.speed / 1048576).toFixed(1)} MB/s` : ''}${job.eta ? formatEta(job.eta, locale) : ''}`;
    if (job.status === 'cancelled') detail.textContent = t.taskCancelledDetail;
    article.append(detail); container.append(article);
  }
}
el('lang-toggle').onclick = () => {
  locale = locale === 'zh' ? 'en' : 'zh';
  void chrome.storage.local.set({ uiLanguage: locale });
  applyLocale();
};
el('grant').onclick = () => {
  // Must be invoked directly from the click to preserve Chrome's user gesture.
  void chrome.permissions.request(sitePermission(site)).then(async granted => {
    if (granted) {
      await chrome.storage.local.set({ [cookieSettingKey(site)]: true });
      cookies.checked = true;
      message(UI_TEXT[locale].grantedMessage);
    } else message(UI_TEXT[locale].grantIncompleteMessage);
    await permissionState();
  }).catch(() => message(UI_TEXT[locale].grantErrorMessage));
};
el('revoke').onclick = async () => { await chrome.permissions.remove({ origins: sitePermission(site).origins }); await permissionState(); };
cookies.onchange = async () => {
  await chrome.storage.local.set({ [cookieSettingKey(site)]: cookies.checked }); updateButton();
  message(cookies.checked ? '' : UI_TEXT[locale].cookieDisabledMessage);
};
download.onclick = async () => {
  submitting = true; updateButton(); message();
  const options = selectedOptions();
  try { render(await send('download.start', { options })); } catch (e) { message((e as Error).message); }
  finally { submitting = false; updateButton(); }
};
el('check').onclick = async () => {
  const button = el<HTMLButtonElement>('check'); button.disabled = true; message();
  try { render(await send('host.check')); } catch (e) { message((e as Error).message); }
  finally { button.disabled = false; }
};
el<HTMLSelectElement>('update-channel').onchange = () => {
  void chrome.storage.local.set({updateChannel:el<HTMLSelectElement>('update-channel').value});
  if (lastState) render(lastState);
};
for (const [id,type] of [['update-check','ytdlp.check'],['update-install','ytdlp.update'],['update-rollback','ytdlp.rollback']]) {
  el(id!).onclick = async () => {
    toolSubmitting = true; message(); if (lastState) render(lastState);
    try { render(await send(type!,{channel:el<HTMLSelectElement>('update-channel').value})); }
    catch (e) { message((e as Error).message); }
    finally { toolSubmitting = false; if (lastState) render(lastState); }
  };
}
chrome.runtime.onMessage.addListener(event => { if (event?.type === 'state.changed') render(event.state); });
async function init() {
  const settings = await chrome.storage.local.get(['useCookies', 'useCookiesX', 'downloadOptions', 'updateChannel', 'uiLanguage']);
  locale = resolveLocale(settings.uiLanguage);
  if (settings.updateChannel === 'nightly') el<HTMLSelectElement>('update-channel').value = 'nightly';
  if (isDownloadOptions(settings.downloadOptions)) {
    const saved = settings.downloadOptions;
    el<HTMLSelectElement>('download-type').value = saved.format;
    el<HTMLSelectElement>('quality').value = saved.maxHeight ? String(saved.maxHeight) : 'best';
    el<HTMLSelectElement>('audio-format').value = saved.audioFormat ?? (saved.format === 'audio' ? 'mp3' : 'original');
    el<HTMLSelectElement>('audio-quality').value = saved.audioQuality ?? 'best';
    el<HTMLSelectElement>('concurrent-fragments').value = String(saved.concurrentFragments ?? 4);
    el<HTMLInputElement>('gif-start').value = String(saved.gifStart ?? 0);
    el<HTMLSelectElement>('gif-duration').value = String(saved.gifDuration ?? 15);
    el<HTMLSelectElement>('gif-width').value = String(saved.gifWidth ?? 480);
    el<HTMLSelectElement>('gif-fps').value = String(saved.gifFps ?? 12);
    el<HTMLSelectElement>('subtitle-language').value = saved.subtitleLanguage ?? 'zh-en';
    el<HTMLSelectElement>('subtitle-format').value = saved.subtitleFormat ?? 'srt';
    el<HTMLInputElement>('subtitle-auto').checked = saved.subtitleAuto ?? true;
  }
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  try {
    const video = resolveVideo(tab?.url ?? ''); validVideo = true;
    site = video.site;
    securePage = video.url.startsWith('https:');
    el('video-title').textContent = tab?.title?.replace(/ - YouTube$/, '') || video.videoId;
    el('video-url').textContent = video.url;
  } catch { validVideo = false; }
  const siteSettings = await chrome.storage.local.get(cookieSettingKey(site));
  cookies.checked = securePage && usesCookies(siteSettings,site);
  cookies.disabled = !securePage;
  el('x-hint').hidden = site !== 'x';
  applyLocale();
  if (!securePage) message(UI_TEXT[locale].httpAnonymousMessage);
  await permissionState();
  render(await send('state.get')); updateButton();
}
void init().catch(() => message(UI_TEXT[locale].initFailedMessage));
