import { afterEach, expect, it, vi } from 'vitest';
import { NativeClient } from './native-client';
afterEach(() => { vi.unstubAllGlobals(); vi.useRealTimers(); });
it('keeps installation pending through progress and resolves on final status', async () => {
  vi.useFakeTimers();
  let receive: (e: unknown) => void = () => {};
  let request: {id: string};
  vi.stubGlobal('chrome', {runtime: {connectNative: () => ({
    onMessage: {addListener: (cb: typeof receive) => {receive = cb;}},
    onDisconnect: {addListener: () => {}},
    postMessage: (e: typeof request) => {request=e;},
  })}});
  const client = new NativeClient(vi.fn(), vi.fn());
  let completed=false;
  const pending=client.request('host.tools.ensure',{},16*60_000).then(() => {completed=true;});
  receive({version:1,id:request!.id,type:'tools.progress',payload:{message:'downloading'}});
  await vi.advanceTimersByTimeAsync(30_000);
  expect(completed).toBe(false);
  receive({version:1,id:request!.id,type:'host.status',payload:{tools:[]}});
  await pending;
  expect(completed).toBe(true);
});
