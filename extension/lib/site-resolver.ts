import { parse } from 'tldts';
import { BridgeError, SITE_PERMISSIONS, type Site } from './types';

export function siteDomains(site: Site): string[] {
  return site === 'youtube' ? ['youtube.com'] : site === 'x' ? ['x.com', 'twitter.com'] : [site];
}
export function siteName(site: Site): string {
  return ({youtube:'YouTube',x:'X / Twitter','bilibili.com':'Bilibili','b23.tv':'Bilibili 短链','nicovideo.jp':'Niconico','nico.ms':'Niconico 短链','vimeo.com':'Vimeo','twitch.tv':'Twitch','dailymotion.com':'Dailymotion'} as Record<string,string>)[site] ?? site;
}
export function sitePermission(site: Site): chrome.permissions.Permissions {
  if (site === 'youtube' || site === 'x') return SITE_PERMISSIONS[site];
  // Chrome checks non-Secure cookies against HTTP host permissions even on HTTPS pages.
  // Request both schemes for this site only; session downloads still require HTTPS.
  // Keep schemes separate to match the optional manifest patterns; Chrome rejects
  // a wildcard-scheme request against separate HTTP and HTTPS declarations.
  return {permissions:['cookies'],origins:siteDomains(site).flatMap(domain=>[`http://*.${domain}/*`,`https://*.${domain}/*`])};
}
export function cookieSettingKey(site: Site): string { return site === 'youtube' ? 'useCookies' : site === 'x' ? 'useCookiesX' : `useCookies:${site}`; }
export function usesCookies(settings: Record<string,unknown>,site:Site): boolean {
  return site === 'youtube' ? settings.useCookies !== false : settings[cookieSettingKey(site)] === true;
}
export function resolveVideo(input: string): { url: string; videoId: string; site: Site } {
  let u: URL;
  try { u = new URL(input); } catch { throw new BridgeError('INVALID_URL', '请打开包含视频或音频的网页。'); }
  const host = u.hostname;
  const parsed = parse(host,{allowPrivateDomains:true});
  if (input.length > 4096 || !['http:','https:'].includes(u.protocol) || u.username || u.password || u.port || parsed.isIp || !parsed.domain || /(^|\.)(localhost|local|internal|lan|home|onion)$/.test(host) || host.endsWith('.')) {
    throw new BridgeError('INVALID_URL', '请使用公共网站的 HTTP / HTTPS 媒体页面；不支持本地地址或特殊协议。');
  }
  if (['x.com', 'www.x.com', 'twitter.com', 'www.twitter.com', 'mobile.twitter.com'].includes(host)) {
    if (u.protocol !== 'https:') throw new BridgeError('INVALID_URL','请使用站点的 HTTPS 页面。');
    const match = /^\/(?:[A-Za-z0-9_]{1,15}|i\/web)\/status\/([0-9]{1,20})(?:\/video\/([1-4]))?\/?$/.exec(u.pathname);
    if (match) return { site: 'x', videoId: match[1]!, url: `https://x.com/i/web/status/${match[1]}${match[2] ? `/video/${match[2]}` : ''}` };
    throw new BridgeError('UNSUPPORTED_URL', '请打开具体的 X 视频帖子，而不是首页或个人资料页。');
  }
  if (['youtu.be','youtube.com','www.youtube.com','m.youtube.com','music.youtube.com'].includes(host)) {
    if (u.protocol !== 'https:') throw new BridgeError('INVALID_URL','请使用站点的 HTTPS 页面。');
    let id: string | null = null;
    if (host === 'youtu.be') id = u.pathname.slice(1);
    else if (u.pathname === '/watch') id = u.searchParams.get('v');
    else id = /^\/(?:shorts|live|embed)\/([A-Za-z0-9_-]{11})\/?$/.exec(u.pathname)?.[1] ?? null;
    if (!id || !/^[A-Za-z0-9_-]{11}$/.test(id)) throw new BridgeError('UNSUPPORTED_URL', '请打开单个 YouTube 视频、Shorts 或直播页面。');
    return { site: 'youtube', videoId: id, url: `https://www.youtube.com/watch?v=${id}` };
  }
  // Preserve part selectors, fragments and signed media query strings.
  return {site: parsed.domain,videoId:host,url:u.href};
}
