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

// ---- Live tail (WebSocket) ----

export interface LiveTailHandle {
  close: () => void;
}

/** Subscribe to /api/v1/events. Browser surface only — desktop Wails uses
 *  the polling path until we wire a Wails event channel. */
export function liveTail(
  onEvent: (ev: OblivraEvent) => void,
  onError?: (err: Error) => void,
): LiveTailHandle {
  if (inWails()) {
    // No WS in Wails yet — fall back to a no-op handle. Siem view will keep polling.
    return { close: () => {} };
  }
  const url =
    (location.protocol === 'https:' ? 'wss://' : 'ws://') +
    location.host +
    '/api/v1/events';
  let closed = false;
  let ws: WebSocket | null = null;
  let retry: ReturnType<typeof setTimeout> | null = null;

  const connect = () => {
    if (closed) return;
    ws = new WebSocket(url);
    ws.onmessage = (m) => {
      try {
        const msg = JSON.parse(m.data);
        if (msg.type === 'event' && msg.event) onEvent(msg.event as OblivraEvent);
      } catch (e) {
        onError?.(e as Error);
      }
    };
    ws.onerror = () => {
      onError?.(new Error('websocket error'));
    };
    ws.onclose = () => {
      if (closed) return;
      retry = setTimeout(connect, 1500);
    };
  };
  connect();

  return {
    close() {
      closed = true;
      if (retry) clearTimeout(retry);
      if (ws) ws.close();
    },
  };
}

// ---- Alerts ----

export interface Alert {
  id: string;
  tenantId: string;
  ruleId: string;
  ruleName: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  hostId?: string;
  message: string;
  mitre?: string[];
  triggered: string;
  eventIds: string[];
  state: string;
}

export async function listAlerts(limit = 100): Promise<Alert[]> {
  return rest<Alert[]>(`/api/v1/alerts?limit=${limit}`);
}

// ---- Threat intel ----

export interface Indicator {
  value: string;
  type: 'ip' | 'domain' | 'hash' | 'url';
  source?: string;
  tags?: string[];
  severity?: string;
  added: string;
}

export async function intelList(): Promise<Indicator[]> {
  return rest<Indicator[]>('/api/v1/threatintel/indicators');
}

export async function intelLookup(value: string): Promise<{ match: boolean; indicator?: Indicator }> {
  return rest(`/api/v1/threatintel/lookup?value=${encodeURIComponent(value)}`);
}

// ---- Rules ----

export interface Rule {
  id: string;
  name: string;
  severity: string;
  fields: string[];
  anyContain?: string[];
  allContain?: string[];
  eventType?: string;
  mitre?: string[];
  source?: string;
  disabled?: boolean;
}

export async function listRules(): Promise<Rule[]> {
  return rest<Rule[]>('/api/v1/detection/rules');
}

export async function reloadRules(): Promise<{ loaded: number }> {
  return restPost('/api/v1/detection/rules/reload', {});
}

export async function mitreHeatmap(): Promise<{ technique: string; count: number }[]> {
  return rest('/api/v1/mitre/heatmap');
}

// ---- Audit ----

export interface AuditEntry {
  seq: number;
  timestamp: string;
  actor: string;
  action: string;
  tenantId: string;
  detail?: Record<string, string>;
  parentHash: string;
  hash: string;
  signature?: string;
}

export interface VerifyResult {
  ok: boolean;
  entries: number;
  brokenAt?: number;
  rootHash: string;
  generatedAt: string;
}

export async function auditLog(limit = 200): Promise<AuditEntry[]> {
  return rest<AuditEntry[]>(`/api/v1/audit/log?limit=${limit}`);
}

export async function auditVerify(): Promise<VerifyResult> {
  return rest('/api/v1/audit/verify');
}

export async function auditPackage(): Promise<unknown> {
  return restPost('/api/v1/audit/packages/generate', {});
}

// ---- Fleet ----

export interface Agent {
  id: string;
  token: string;
  hostname: string;
  os?: string;
  arch?: string;
  version?: string;
  tags?: string[];
  registered: string;
  lastSeen: string;
  events: number;
}

export async function fleetList(): Promise<Agent[]> {
  return rest('/api/v1/agent/fleet');
}

// ---- UEBA ----

export interface UebaProfile {
  entity: string;
  tenant: string;
  windowsSeen: number;
  mean: number;
  stdDev: number;
  lastEpm: number;
  lastSpike: number;
  updatedAt: string;
}

export async function uebaProfiles(): Promise<UebaProfile[]> {
  return rest('/api/v1/ueba/profiles');
}

// ---- Forensics ----

export interface LogGap {
  hostId: string;
  startedAt: string;
  endedAt: string;
  duration: string;
}

export interface EvidenceItem {
  id: string;
  tenantId: string;
  hostId: string;
  title: string;
  from: string;
  to: string;
  eventIds: string[];
  hash: string;
  sealedAt: string;
  sealed: boolean;
}

export async function forensicsGaps(): Promise<LogGap[]> {
  return rest('/api/v1/forensics/gaps');
}

export async function forensicsList(): Promise<EvidenceItem[]> {
  return rest('/api/v1/forensics/evidence');
}

// ---- Storage tiering ----

export interface TierStats {
  warmFiles: number;
  warmEvents: number;
  lastRunAt: string;
  lastRunMoved: number;
  warmDir: string;
  hotAgeMax: string;
}

export async function storageStats(): Promise<TierStats> {
  return rest('/api/v1/storage/stats');
}

export async function storagePromote(): Promise<{ moved: number }> {
  return restPost('/api/v1/storage/promote', {});
}
