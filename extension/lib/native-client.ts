import { BridgeError, HOST_NAME, type HostEvent } from './types';

export class NativeClient {
  private port?: chrome.runtime.Port;
  private pending = new Map<string, { resolve: (e: HostEvent) => void; reject: (e: Error) => void; timer: ReturnType<typeof setTimeout> }>();
  constructor(private onEvent: (e: HostEvent) => void, private onDisconnect: () => void) {}
  get connected() { return !!this.port; }

  request(type: string, payload: unknown = {}, timeoutMs = 20_000): Promise<HostEvent> {
    if (!this.port) {
      this.port = chrome.runtime.connectNative(HOST_NAME);
      this.port.onMessage.addListener((e: HostEvent) => {
        if (!e || e.version !== 1 || typeof e.id !== 'string' || typeof e.type !== 'string' || !e.payload) return;
        const pending = this.pending.get(e.id);
        if (pending && e.type !== 'tools.progress') {
          clearTimeout(pending.timer);
          this.pending.delete(e.id);
          if (e.type === 'error') pending.reject(new BridgeError(String(e.payload.code), String(e.payload.message)));
          else pending.resolve(e);
        }
        this.onEvent(e);
      });
      this.port.onDisconnect.addListener(() => {
        // Consume lastError without forwarding browser diagnostics or request data.
        void chrome.runtime.lastError;
        this.port = undefined;
        for (const p of this.pending.values()) {
          clearTimeout(p.timer);
          p.reject(new BridgeError('HOST_DISCONNECTED', '本地桥接未连接。请检查 Native Host 安装、扩展 ID 和配置文件。'));
        }
        this.pending.clear();
        this.onDisconnect();
      });
    }
    const id = crypto.randomUUID();
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new BridgeError('HOST_TIMEOUT', '本地桥接响应超时，请检查本地程序。'));
      }, timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      try { this.port!.postMessage({ version: 1, id, type, payload }); }
      catch { clearTimeout(timer); this.pending.delete(id); reject(new BridgeError('HOST_DISCONNECTED', '本地桥接连接失败。')); }
    });
  }
}
