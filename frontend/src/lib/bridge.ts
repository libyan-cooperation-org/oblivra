// Single bridge used by every Svelte component.
// Routes calls to Wails bindings when running inside the desktop shell, or to
// the REST API when running in a browser (served by cmd/server).

declare global {
  interface Window {
    wails?: {
      Call: (service: string, method: string, ...args: unknown[]) => Promise<unknown>;
    };
  }
}

export interface SystemInfo {
  version: string;
  goVersion: string;
  os: string;
  arch: string;
  numCpu: number;
  startedAt: string;
  goroutines: number;
}

export interface Health {
  status: string;
  timestamp: string;
}

const inWails = () => typeof window !== 'undefined' && !!window.wails;

async function rest<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: 'same-origin' });
  if (!res.ok) throw new Error(`${path}: ${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

export async function getSystemInfo(): Promise<SystemInfo> {
  if (inWails()) {
    return (await window.wails!.Call('SystemService', 'Info')) as SystemInfo;
  }
  return rest<SystemInfo>('/api/v1/system/info');
}

export async function ping(): Promise<Health> {
  if (inWails()) {
    return (await window.wails!.Call('SystemService', 'Ping')) as Health;
  }
  return rest<Health>('/api/v1/system/ping');
}
