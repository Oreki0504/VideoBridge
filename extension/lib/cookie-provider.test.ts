import { describe, it, expect, vi, afterEach } from 'vitest';
import { readCookies, snapshotCookies } from './cookie-provider';

const cookie = (overrides: Partial<chrome.cookies.Cookie> = {}): chrome.cookies.Cookie => ({
  name: 'SID', value: 'synthetic-test-only', domain: '.youtube.com', path: '/',
  secure: true, httpOnly: true, hostOnly: false, session: true, storeId: '1', sameSite: 'lax', ...overrides,
});
afterEach(() => vi.unstubAllGlobals());
describe('cookie snapshots', () => {
  it('requires Niconico login credentials and retains their non-Secure HttpOnly flags', () => {
    const session = cookie({name:'user_session',domain:'.nicovideo.jp',secure:false});
    const marker = cookie({name:'user_session_secure',domain:'.nicovideo.jp'});
    expect(() => snapshotCookies([marker],20,'nicovideo.jp')).toThrow('Niconico 登录凭据');
    expect(() => snapshotCookies([marker,{...session,session:false,expirationDate:10}],20,'nicovideo.jp')).toThrow('Niconico 登录凭据');
    expect(snapshotCookies([marker,session],20,'nicovideo.jp')).toContainEqual(expect.objectContaining({name:'user_session',secure:false,httpOnly:true}));
  });
  it('keeps generic sites and private tenants isolated', () => {
    const bili=cookie({name:'SESSDATA',domain:'.bilibili.com'});
    const nico=cookie({name:'user_session',domain:'.nicovideo.jp'});
    expect(snapshotCookies([bili,nico,cookie()],20,'bilibili.com')).toEqual([expect.objectContaining({domain:'.bilibili.com'})]);
    expect(snapshotCookies([bili,nico,cookie()],20,'nicovideo.jp')).toEqual([expect.objectContaining({domain:'.nicovideo.jp'})]);
    expect(snapshotCookies([cookie({domain:'.foo.github.io'}),cookie({domain:'.bar.github.io'}),cookie({domain:'.github.io'})],20,'foo.github.io')).toHaveLength(1);
  });
  it('isolates X cookies and requires X authentication when enabled', () => {
    const x = cookie({ name: 'auth_token', domain: '.x.com' });
    expect(snapshotCookies([cookie(), x], 20, 'x')).toEqual([expect.objectContaining({ domain: '.x.com' })]);
    expect(snapshotCookies([cookie(), x], 20, 'youtube')).toEqual([expect.objectContaining({ domain: '.youtube.com' })]);
    expect(() => snapshotCookies([cookie()], 20, 'x')).toThrow('登录');
    expect(() => snapshotCookies([x], 20, 'youtube')).toThrow('登录');
  });
  it('retains Netscape attributes but excludes browser metadata and other domains', () => {
    const snapshot = snapshotCookies([cookie(), cookie({ domain: '.example.com' })]);
    expect(snapshot).toHaveLength(1);
    expect(snapshot[0]).toMatchObject({ httpOnly: true, hostOnly: false, session: true });
    expect(snapshot[0]).not.toHaveProperty('storeId');
    expect(snapshot[0]).not.toHaveProperty('sameSite');
  });
  it('does not silently flatten partitioned cookies', () => {
    expect(() => snapshotCookies([cookie({ partitionKey: { topLevelSite: 'https://youtube.com' } })])).toThrow('分区');
  });
  it('rejects missing or expired login cookies', () => {
    expect(() => snapshotCookies([cookie({ name: 'PREF' })])).toThrow('登录');
    expect(() => snapshotCookies([cookie({ session: false, expirationDate: 10 })], 20)).toThrow('登录');
  });
  it('selects the active tab store and fetches a new snapshot for every request', async () => {
    const getAll = vi.fn().mockResolvedValue([cookie()]);
    vi.stubGlobal('chrome', {
      permissions: { contains: vi.fn().mockResolvedValue(true) },
      cookies: { getAllCookieStores: vi.fn().mockResolvedValue([{ id: '0', tabIds: [1] }, { id: '1', tabIds: [42] }]), getAll },
    });
    await readCookies({ id: 42 } as chrome.tabs.Tab);
    await readCookies({ id: 42 } as chrome.tabs.Tab);
    expect(getAll).toHaveBeenCalledTimes(2);
    expect(getAll).toHaveBeenLastCalledWith({ domain: 'youtube.com', storeId: '1' });
  });
  it('does not read cookies after permission revocation', async () => {
    const getAll = vi.fn();
    vi.stubGlobal('chrome', { permissions: { contains: vi.fn().mockResolvedValue(false) }, cookies: { getAll } });
    await expect(readCookies({ id: 42 } as chrome.tabs.Tab)).rejects.toMatchObject({ code: 'COOKIE_PERMISSION_REQUIRED' });
    expect(getAll).not.toHaveBeenCalled();
  });
});
