import { afterEach, expect, it, vi } from 'vitest';
import type { HostEvent } from './types';
const fixture = vi.hoisted(() => ({
  requests: [] as {type:string;payload:unknown}[],
  missing: false,
  old: false,
  url: 'https://x.com/name/status/123',
  setup: undefined as undefined | (() => Promise<void>),
  update: undefined as undefined | (() => Promise<void>),
}));
vi.mock('./native-client', () => ({ NativeClient: class {
  connected=true;
  constructor(private event: (e: HostEvent) => void) {}
  async request(type: string,payload: unknown={}) {
    fixture.requests.push({type,payload});
    if (type === 'host.tools.ensure') { await fixture.setup?.(); fixture.missing=false; }
    if (type === 'host.status' || type === 'host.tools.ensure') this.event({version:1,id:'fixture',type:'host.status',payload:{capabilities:fixture.old?[]:['x','gif','subtitles','tools.ensure','generic-sites','ytdlp.update'],tools:[{name:'yt-dlp',available:!fixture.missing}]}});
    if (type.startsWith('host.ytdlp.')) {
      await fixture.update?.();
      this.event({version:1,id:'fixture',type:'ytdlp.status',payload:{health:{capabilities:['ytdlp.update'],tools:[]},update:{channel:(payload as any).channel,currentVersion:'2026.10.01',latestVersion:'2026.11.04',updateAvailable:true,canRollback:false,supported:true,message:'发现新版本'}}});
    }
    return {};
  }
}}));
afterEach(() => {vi.unstubAllGlobals();vi.resetModules();fixture.requests=[];fixture.missing=false;fixture.old=false;fixture.setup=undefined;fixture.update=undefined;fixture.url='https://x.com/name/status/123';});
async function startBackground(settings: Record<string,unknown> = {}) {
  let handler: (message: unknown, sender: unknown, reply: (value: any) => void) => void = () => {};
  const getAll=vi.fn(async ({domain}: {domain:string}) => domain === 'x.com' ? [{name:'auth_token',value:'synthetic-only',domain:'.x.com',path:'/',session:true,hostOnly:false,secure:true,httpOnly:true}] : []);
  vi.stubGlobal('defineBackground',(cb:()=>void) => cb());
  vi.stubGlobal('chrome', {
    runtime: {id:'test',getURL:()=> 'chrome-extension://test/',onMessage:{addListener:(cb:typeof handler)=>{handler=cb;}},sendMessage:vi.fn().mockResolvedValue({})},
    storage: {local:{get:vi.fn().mockResolvedValue(settings)}},
    tabs:{query:vi.fn(async()=>[{id:42,url:fixture.url}])},
    permissions:{contains:vi.fn().mockResolvedValue(true)},
    cookies:{getAll,getAllCookieStores:vi.fn().mockResolvedValue([{id:'1',tabIds:[42]}])},
  });
  await import('../entrypoints/background');
  return { getAll, send: (message: unknown) => new Promise<any>(resolve => handler(message,{id:'test',url:'chrome-extension://test/popup.html'},resolve)) };
}
it('starts X anonymously even when the YouTube session is enabled',async()=>{
  const {send,getAll}=await startBackground({useCookies:true});
  expect((await send({type:'download.start',options:{format:'gif'}})).ok).toBe(true);
  expect(getAll).not.toHaveBeenCalled();
  expect(fixture.requests.at(-1)).toEqual({type:'download.start',payload:expect.objectContaining({url:'https://x.com/i/web/status/123',cookieMode:'off',cookies:[]})});
});
it('downloads YouTube and Bilibili subtitles and rejects an old Host before reading cookies',async()=>{
  const {send,getAll}=await startBackground({useCookies:false});
  for (const url of ['https://www.youtube.com/watch?v=dQw4w9WgXcQ','https://www.bilibili.com/video/BV13x41117TL?p=2']) {
    fixture.url=url;
    const options={format:'subtitles',subtitleLanguage:'zh-en',subtitleFormat:'srt',subtitleAuto:true};
    expect((await send({type:'download.start',options})).ok).toBe(true);
    expect(fixture.requests.at(-1)?.payload).toMatchObject({url,options,cookieMode:'off'});
  }
  fixture.old=true;
  expect((await send({type:'download.start',options:{format:'subtitles'}})).code).toBe('HOST_UPDATE_REQUIRED');
  expect(getAll).not.toHaveBeenCalled();
});
it('submits Bilibili parts anonymously and requires generic-site Host support',async()=>{
  fixture.url='https://www.bilibili.com/video/BV1xx411c7mD?p=2';
  const {send,getAll}=await startBackground({useCookies:true,useCookiesX:true});
  expect((await send({type:'download.start',options:{format:'best'}})).ok).toBe(true);
  expect(fixture.requests.at(-1)?.payload).toMatchObject({url:fixture.url,cookieMode:'off'});
  expect(getAll).not.toHaveBeenCalled();
  fixture.old=true;
  expect((await send({type:'download.start',options:{format:'best'}})).code).toBe('HOST_UPDATE_REQUIRED');
});
it('finishes dependency setup before reading an enabled X session',async()=>{
  fixture.missing=true;
  const {send,getAll}=await startBackground({useCookiesX:true});
  fixture.setup=async()=>{expect(getAll).not.toHaveBeenCalled();};
  expect((await send({type:'download.start',options:{format:'best'}})).ok).toBe(true);
  expect(fixture.requests.map(r=>r.type)).toEqual(['host.status','host.tools.ensure','download.start']);
  expect(getAll.mock.calls.map(([p])=>p.domain)).toEqual(['x.com','twitter.com']);
  expect(fixture.requests.at(-1)?.payload).toMatchObject({cookieMode:'always',profileStoreId:'1'});
});
it('stops before reading cookies when setup fails or the Host lacks X support',async()=>{
  fixture.missing=true;
  const {send,getAll}=await startBackground({useCookiesX:true});
  fixture.setup=async()=>{throw new Error('download failed');};
  expect((await send({type:'download.start',options:{format:'gif'}})).ok).toBe(false);
  expect(getAll).not.toHaveBeenCalled();
  fixture.missing=false;fixture.old=true;
  expect((await send({type:'download.start',options:{format:'gif'}})).code).toBe('HOST_UPDATE_REQUIRED');
  expect(getAll).not.toHaveBeenCalled();
  expect(fixture.requests.some(r=>r.type==='download.start')).toBe(false);
});

it('checks and updates tools without reading cookies, and rejects unknown channels',async()=>{
  const {send,getAll}=await startBackground({useCookiesX:true});
  expect((await send({type:'ytdlp.check',channel:'nightly'})).data.ytdlpUpdate.channel).toBe('nightly');
  expect(fixture.requests.at(-1)).toEqual({type:'host.ytdlp.check',payload:{channel:'nightly'}});
  expect((await send({type:'ytdlp.update',channel:'nightly'})).ok).toBe(true);
  expect((await send({type:'ytdlp.rollback',channel:'stable'})).ok).toBe(true);
  const count=fixture.requests.length;
  expect((await send({type:'ytdlp.update',channel:'https://evil.example'})).code).toBe('INVALID_REQUEST');
  expect(fixture.requests).toHaveLength(count);
  expect(getAll).not.toHaveBeenCalled();
});

it('blocks a download or second update while updating, and recovers after failure',async()=>{
  const {send,getAll}=await startBackground({useCookiesX:true});
  let rejectUpdate!: (error:Error)=>void;
  fixture.update=()=>new Promise<void>((_,reject)=>{rejectUpdate=reject;});
  const pending=send({type:'ytdlp.update',channel:'stable'});
  await vi.waitFor(()=>expect(rejectUpdate).toBeTypeOf('function'));
  expect((await send({type:'state.get'})).data.installing).toBe(true);
  expect((await send({type:'download.start',options:{format:'best'}})).code).toBe('BUSY');
  expect((await send({type:'ytdlp.check',channel:'stable'})).code).toBe('BUSY');
  expect(getAll).not.toHaveBeenCalled();
  rejectUpdate(new Error('校验失败，旧版本保留'));
  expect((await pending).ok).toBe(false);
  const state=(await send({type:'state.get'})).data;
  expect(state.installing).toBe(false);
  expect(state.setupMessage).toContain('旧版本保留');
  expect(state.ytdlpUpdate).toBeUndefined();
});

it('requires an updated Host for update management',async()=>{
  fixture.old=true;
  const {send}=await startBackground();
  expect((await send({type:'ytdlp.check',channel:'stable'})).code).toBe('HOST_UPDATE_REQUIRED');
  expect(fixture.requests.map(r=>r.type)).toEqual(['host.status']);
});
