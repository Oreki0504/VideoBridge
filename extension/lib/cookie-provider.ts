import { BridgeError, type CookieSnapshot, type Site } from './types';
import { siteDomains, siteName, sitePermission } from './site-resolver';

const AUTH_COOKIE = /^(?:SID|HSID|SSID|SAPISID|APISID|__Secure-[13]P(?:SID|APISID))$/;

export function snapshotCookies(cookies: chrome.cookies.Cookie[], now = Date.now() / 1000, site: Site = 'youtube'): CookieSnapshot[] {
  const domains = siteDomains(site);
  const relevant = cookies.filter(c => domains.some(domain => c.domain.replace(/^\./, '') === domain || c.domain.endsWith(`.${domain}`)));
  if (relevant.some(c => c.partitionKey)) {
    throw new BridgeError('PARTITIONED_COOKIES_UNSUPPORTED', '检测到分区 Cookie；Netscape 格式无法保留分区信息，当前版本不能安全转换。');
  }
  const live = relevant.filter(c => c.session || (c.expirationDate !== undefined && c.expirationDate > now));
  // yt-dlp selects Niconico's authenticated API only when user_session is present.
  if (site === 'nicovideo.jp' && !live.some(c => c.name === 'user_session' && c.value)) {
    throw new BridgeError('COOKIE_UNAVAILABLE', '未读取到 Niconico 登录凭据。请重新授权该站点，并确认当前浏览器已登录后重试。');
  }
  if (!live.some(c => (site === 'youtube' ? AUTH_COOKIE.test(c.name) : site === 'x' ? c.name === 'auth_token' : true) && c.value)) {
    throw new BridgeError('COOKIE_UNAVAILABLE', `未发现 ${siteName(site)} 可用会话 Cookie。请先在当前 Profile 登录，或关闭会话模式尝试公开视频。`);
  }
  return live.map(c => ({
    name: c.name, value: c.value, domain: c.domain, path: c.path,
    secure: c.secure, httpOnly: c.httpOnly, hostOnly: c.hostOnly, session: c.session,
    ...(!c.session ? { expirationDate: c.expirationDate } : {}),
  }));
}

export async function readCookies(tab: chrome.tabs.Tab, site: Site = 'youtube'): Promise<{ storeId: string; cookies: CookieSnapshot[] }> {
  if (tab.incognito) throw new BridgeError('INCOGNITO_UNSUPPORTED', '当前版本不支持隐身窗口。');
  if (!await chrome.permissions.contains(sitePermission(site))) {
    throw new BridgeError('COOKIE_PERMISSION_REQUIRED', '请先授权当前站点 Cookie 访问。');
  }
  const stores = (await chrome.cookies.getAllCookieStores()).filter(s => tab.id !== undefined && s.tabIds.includes(tab.id));
  if (stores.length !== 1) throw new BridgeError('COOKIE_STORE_UNAVAILABLE', '无法唯一确定当前标签页的 Cookie Store，请重新打开视频页面。');
  const storeId = stores[0]!.id;
  try {
    const domains = siteDomains(site);
    const cookies = (await Promise.all(domains.map(domain => chrome.cookies.getAll({ domain, storeId })))).flat();
    return { storeId, cookies: snapshotCookies(cookies, Date.now() / 1000, site) };
  } catch (e) {
    if (e instanceof BridgeError) throw e;
    throw new BridgeError('COOKIE_READ_FAILED', '读取站点 Cookie 失败，请检查权限后重试。');
  }
}
