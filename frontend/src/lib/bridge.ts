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

// ---- SIEM ----

export type Severity = 'debug' | 'info' | 'notice' | 'warning' | 'error' | 'critical' | 'alert';

export interface OblivraEvent {
  id: string;
  tenantId: string;
  timestamp: string;
  receivedAt: string;
  source: string;
  hostId?: string;
  eventType?: string;
  severity?: Severity;
  message: string;
  raw?: string;
  fields?: Record<string, string>;
}

export interface IngestStats {
  total: number;
  hotCount: number;
  wal: { path: string; bytes: number; count: number };
  eps: number;
  generatedAt: string;
}

export interface SearchOptions {
  query?: string;
  fromUnix?: number;
  toUnix?: number;
  limit?: number;
  newestFirst?: boolean;
  tenantId?: string;
}

export interface SearchResponse {
  events: OblivraEvent[];
  total: number;
  took: string;
  mode: 'chrono' | 'fulltext';
}

async function restPost<T, B>(path: string, body: B): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    credentials: 'same-origin',
  });
  if (!res.ok) throw new Error(`${path}: ${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

export async function siemIngest(ev: Partial<OblivraEvent>): Promise<OblivraEvent> {
  if (inWails()) {
    return (await window.wails!.Call('SiemService', 'Ingest', ev)) as OblivraEvent;
  }
  return restPost<OblivraEvent, Partial<OblivraEvent>>('/api/v1/siem/ingest', ev);
}

export async function siemSearch(opts: SearchOptions = {}): Promise<SearchResponse> {
  if (inWails()) {
    return (await window.wails!.Call('SiemService', 'Search', {
      tenantId: opts.tenantId ?? '',
      query: opts.query ?? '',
      fromUnix: opts.fromUnix ?? 0,
      toUnix: opts.toUnix ?? 0,
      limit: opts.limit ?? 0,
      newestFirst: opts.newestFirst ?? true,
    })) as SearchResponse;
  }
  const params = new URLSearchParams();
  if (opts.query) params.set('q', opts.query);
  if (opts.fromUnix) params.set('from', String(opts.fromUnix));
  if (opts.toUnix) params.set('to', String(opts.toUnix));
  if (opts.limit) params.set('limit', String(opts.limit));
  if (opts.newestFirst) params.set('newestFirst', 'true');
  if (opts.tenantId) params.set('tenant', opts.tenantId);
  return rest<SearchResponse>(`/api/v1/siem/search?${params}`);
}

export async function siemStats(): Promise<IngestStats> {
  if (inWails()) {
    return (await window.wails!.Call('SiemService', 'Stats')) as IngestStats;
  }
  return rest<IngestStats>('/api/v1/siem/stats');
}
