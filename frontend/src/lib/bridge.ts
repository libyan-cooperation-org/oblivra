// REST client used by every Svelte component.
//
// The frontend talks to the headless server's REST API. Both the Wails
// desktop shell (when wired up) and the headless cmd/server expose the
// same /api/v1 surface, so a single fetch-based client covers both.

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

async function rest<T>(path: string): Promise<T> {
  const res = await fetch(path, { credentials: 'same-origin' });
  if (!res.ok) throw new Error(`${path}: ${res.status} ${res.statusText}`);
  return (await res.json()) as T;
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

export async function getSystemInfo(): Promise<SystemInfo> {
  return rest<SystemInfo>('/api/v1/system/info');
}

export async function ping(): Promise<Health> {
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

export async function siemIngest(ev: Partial<OblivraEvent>): Promise<OblivraEvent> {
  return restPost<OblivraEvent, Partial<OblivraEvent>>('/api/v1/siem/ingest', ev);
}

export async function siemSearch(opts: SearchOptions = {}): Promise<SearchResponse> {
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
  return rest<IngestStats>('/api/v1/siem/stats');
}

// ---- Live tail (WebSocket) ----

export interface LiveTailHandle {
  close: () => void;
}

export function liveTail(
  onEvent: (ev: OblivraEvent) => void,
  onError?: (err: Error) => void,
): LiveTailHandle {
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

export type AlertState = 'open' | 'ack' | 'assigned' | 'resolved';
export type AlertVerdict = 'true-positive' | 'false-positive' | 'benign-true-positive' | 'duplicate' | '';

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
  state: AlertState | string;
  // Lifecycle metadata.
  acknowledgedBy?: string;
  acknowledgedAt?: string;
  assignedTo?: string;
  assignedAt?: string;
  resolvedBy?: string;
  resolvedAt?: string;
  verdict?: AlertVerdict;
  notes?: string;
}

export async function listAlerts(limit = 100): Promise<Alert[]> {
  return rest<Alert[]>(`/api/v1/alerts?limit=${limit}`);
}

export const alertGet = (id: string) =>
  rest<Alert>(`/api/v1/alerts/${encodeURIComponent(id)}`);

export const alertAck = (id: string) =>
  restPost<Alert, {}>(`/api/v1/alerts/${encodeURIComponent(id)}/ack`, {});

export const alertAssign = (id: string, assignee: string) =>
  restPost<Alert, { assignee: string }>(`/api/v1/alerts/${encodeURIComponent(id)}/assign`, { assignee });

export const alertResolve = (id: string, verdict: AlertVerdict, notes?: string) =>
  restPost<Alert, { verdict: AlertVerdict; notes?: string }>(
    `/api/v1/alerts/${encodeURIComponent(id)}/resolve`,
    { verdict, notes },
  );

export const alertReopen = (id: string) =>
  restPost<Alert, {}>(`/api/v1/alerts/${encodeURIComponent(id)}/reopen`, {});

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
  // Rich state from heartbeat (populated by agent v0.1+).
  pubkeyB64?: string;
  pubkeyFingerprint?: string;
  inputCount?: number;
  spillFiles?: number;
  spillBytes?: number;
  queueDepth?: number;
  droppedEvents?: number;
  batchSize?: number;
}

export async function fleetList(): Promise<Agent[]> {
  return rest('/api/v1/agent/fleet');
}

export async function agentGet(id: string): Promise<Agent> {
  return rest<Agent>(`/api/v1/agent/fleet/${encodeURIComponent(id)}`);
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

// ---- Reconstruction / cases / trust / quality ----

export interface Session {
  id: string;
  hostId: string;
  user: string;
  sourceIp?: string;
  method?: string;
  state: 'open' | 'closed' | 'failed' | 'unknown';
  startedAt: string;
  endedAt?: string;
  failedAttempts?: number;
}

export interface ProcessSnapshot {
  hostId: string;
  at: string;
  running: { pid: number; ppid?: number; image?: string; command?: string; started: string }[];
  exited: { pid: number; ppid?: number; image?: string; started: string; ended: string }[];
}

export interface CmdLine {
  hostId: string;
  user?: string;
  pid?: number;
  image?: string;
  command: string;
  timestamp: string;
  eventId: string;
  suspicious?: boolean;
}

export interface AuthChain {
  user: string;
  day: string;
  events: { protocol: string; result: string; hostId: string; sourceIp?: string; timestamp: string }[];
  hosts: string[];
  ips: string[];
  protocols: string[];
  successes: number;
  failures: number;
}

export interface TrustSummary {
  verified: number;
  consistent: number;
  suspicious: number;
  untrusted: number;
}

export interface SourceProfile {
  host: string;
  source: string;
  total: number;
  parsed: number;
  unparsedRate: number;
  gapsObserved: number;
  avgDelayMs: number;
  lastSeen: string;
  firstSeen: string;
}

export interface TamperFinding {
  hostId: string;
  kind: string;
  detail: string;
  eventId?: string;
  timestamp: string;
}

export interface CaseSummary {
  id: string;
  title: string;
  openedBy: string;
  openedAt: string;
  state: string;
  scope: { hostId?: string; from: string; to: string; auditRootAtOpen: string };
  hypotheses?: { id: string; statement: string; status: string }[];
  notes?: { author: string; body: string; timestamp: string }[];
  sealedAt?: string;
  sealedBy?: string;
}

export interface CaseConfidence {
  score: number;
  eventCount: number;
  alertCount: number;
  sourceCount: number;
  gapCount: number;
  explanation: string;
  contributions?: string[];
}

export const reconSessions = (host?: string) =>
  rest<Session[]>(`/api/v1/reconstruction/sessions${host ? '?host=' + host : ''}`);

export const reconStateAt = (host: string, atUnix?: number) =>
  rest<ProcessSnapshot>(`/api/v1/reconstruction/state?host=${encodeURIComponent(host)}${atUnix ? '&at=' + atUnix : ''}`);

export const reconCmdSus = () =>
  rest<CmdLine[]>('/api/v1/reconstruction/cmdline/suspicious');

export const reconAuthMulti = () =>
  rest<AuthChain[]>('/api/v1/reconstruction/auth/multi-protocol');

export const trustSummary = () => rest<TrustSummary>('/api/v1/trust/summary');

export const qualitySources = () => rest<SourceProfile[]>('/api/v1/quality/sources');
export const qualityCoverage = () =>
  rest<{ host: string; lastSeen: string; eventsLastHour: number; eventsLastDay: number; sources: string[] }[]>('/api/v1/quality/coverage');

export const tamperFindings = () => rest<TamperFinding[]>('/api/v1/forensics/tamper');

export const casesList = () => rest<CaseSummary[]>('/api/v1/cases');
export const caseGet = (id: string) => rest<CaseSummary>(`/api/v1/cases/${id}`);
export const caseTimeline = (id: string) =>
  rest<{ kind: string; timestamp: string; severity?: string; title: string; detail?: string }[]>(`/api/v1/cases/${id}/timeline`);
export const caseConfidence = (id: string) => rest<CaseConfidence>(`/api/v1/cases/${id}/confidence`);
export const caseOpen = (body: { title: string; hostId?: string; fromUnix?: number; toUnix?: number }) =>
  restPost<CaseSummary, typeof body>('/api/v1/cases', body);
export const caseSeal = (id: string) => restPost<CaseSummary, {}>(`/api/v1/cases/${id}/seal`, {});
export const caseLegalSubmit = (id: string) => restPost<CaseSummary, {}>(`/api/v1/cases/${id}/legal/submit`, {});
export const caseLegalApprove = (id: string, reason: string) =>
  restPost<CaseSummary, { reason: string }>(`/api/v1/cases/${id}/legal/approve`, { reason });
export const caseLegalReject = (id: string, reason: string) =>
  restPost<CaseSummary, { reason: string }>(`/api/v1/cases/${id}/legal/reject`, { reason });

// ---- Graph / Vault / Webhooks / Pivot ----

export interface GraphEdge {
  from: { kind: string; id: string };
  to: { kind: string; id: string };
  kind: string;
  evidence?: string;
  timestamp: string;
}
export interface GraphStats {
  edges: number;
  nodes: number;
  byKind: Record<string, number>;
}
export const graphStats = () => rest<GraphStats>('/api/v1/graph/stats');
export const graphSubgraph = (kind: string, id: string, depth = 2) =>
  rest<GraphEdge[]>(`/api/v1/graph/subgraph?kind=${encodeURIComponent(kind)}&id=${encodeURIComponent(id)}&depth=${depth}`);

export interface VaultStatus {
  exists: boolean;
  unlocked: boolean;
  names?: string[];
}
export const vaultStatus = () => rest<VaultStatus>('/api/v1/vault/status');
export const vaultInit = (passphrase: string) =>
  restPost<VaultStatus, { passphrase: string }>('/api/v1/vault/init', { passphrase });
export const vaultUnlock = (passphrase: string) =>
  restPost<VaultStatus, { passphrase: string }>('/api/v1/vault/unlock', { passphrase });
export const vaultLock = () => restPost<VaultStatus, {}>('/api/v1/vault/lock', {});
export const vaultSet = (name: string, value: string) =>
  restPost<{}, { name: string; value: string }>('/api/v1/vault/secret', { name, value });
export async function vaultDelete(name: string): Promise<void> {
  const res = await fetch('/api/v1/vault/secret?name=' + encodeURIComponent(name), {
    method: 'DELETE',
    credentials: 'same-origin',
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}

export interface Webhook {
  id: string;
  url: string;
  minSeverity?: string;
  includeRules?: string[];
  excludeRules?: string[];
  createdAt: string;
  // Server elides this field entirely when the webhook has never fired,
  // so undefined === "never delivered" and the UI must guard accordingly.
  lastDelivered?: string;
  disabled?: boolean;
}
export interface WebhookDelivery {
  webhookId: string;
  alertId: string;
  status: number;
  error?: string;
  deliveredAt: string;
}
export const webhooksList = () => rest<Webhook[]>('/api/v1/webhooks');
export const webhookRegister = (w: { url: string; secret?: string; minSeverity?: string }) =>
  restPost<Webhook, typeof w>('/api/v1/webhooks', w);
export async function webhookDelete(id: string): Promise<void> {
  const res = await fetch('/api/v1/webhooks/' + id, { method: 'DELETE', credentials: 'same-origin' });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}
export const webhookDeliveries = () =>
  rest<WebhookDelivery[]>('/api/v1/webhooks/deliveries?limit=50');

export interface PivotEntry {
  kind: string;
  timestamp: string;
  severity?: string;
  title: string;
  detail?: string;
  refId?: string;
}
// ---- Saved searches ----

export interface SavedSearch {
  id: string;
  name: string;
  query: string;
  queryKind?: 'bleve' | 'oql' | string;
  tenantId?: string;
  createdAt: string;
  createdBy?: string;
  intervalMinutes?: number;
  alertOnAtLeast?: number;
  severity?: 'low' | 'medium' | 'high' | 'critical';
  lastRunAt?: string;
  lastHitCount?: number;
  lastError?: string;
}

export const savedSearchesList = () => rest<SavedSearch[]>('/api/v1/saved-searches');
export const savedSearchesSave = (q: Partial<SavedSearch> & { name: string; query: string }) =>
  restPost<SavedSearch, typeof q>('/api/v1/saved-searches', q);
export const savedSearchesRun = (id: string) =>
  restPost<{ id: string; hits: number }, {}>(`/api/v1/saved-searches/${encodeURIComponent(id)}/run`, {});
export async function savedSearchesDelete(id: string): Promise<void> {
  const res = await fetch('/api/v1/saved-searches/' + encodeURIComponent(id), {
    method: 'DELETE',
    credentials: 'same-origin',
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}

// ---- Event detail ----

export interface EventDetail {
  event: OblivraEvent;
  related: OblivraEvent[];
}

export const eventGet = (id: string) =>
  rest<EventDetail>(`/api/v1/siem/events/${encodeURIComponent(id)}`);

// Build a download URL for the current search query in the requested format.
export function siemSearchExportUrl(opts: SearchOptions, format: 'csv' | 'ndjson'): string {
  const params = new URLSearchParams();
  if (opts.query) params.set('q', opts.query);
  if (opts.fromUnix) params.set('from', String(opts.fromUnix));
  if (opts.toUnix) params.set('to', String(opts.toUnix));
  if (opts.limit) params.set('limit', String(opts.limit));
  if (opts.newestFirst) params.set('newestFirst', 'true');
  if (opts.tenantId) params.set('tenant', opts.tenantId);
  params.set('format', format);
  return `/api/v1/siem/search?${params}`;
}

// ---- Categories (sourceType breakdown) ----

export interface CategoryStat {
  sourceType: string;
  count: number;
  lastSeen: string;
  topHosts: { host: string; count: number }[];
}

export const categoriesList = () =>
  rest<CategoryStat[]>('/api/v1/categories');

// ---- Email / notification channels ----

export interface NotificationChannel {
  id: string;
  kind: 'email' | 'webhook';
  name: string;
  // email-only
  smtpHost?: string;
  smtpPort?: number;
  smtpFrom?: string;
  smtpTo?: string;
  smtpUsername?: string;
  // webhook fields are already covered by Webhook type above
  minSeverity?: string;
  createdAt: string;
  lastDelivered?: string;
  lastError?: string;
  disabled?: boolean;
}

export const notificationsList = () => rest<NotificationChannel[]>('/api/v1/notifications');
export const notificationsAdd = (n: Partial<NotificationChannel> & { kind: string; name: string }) =>
  restPost<NotificationChannel, typeof n>('/api/v1/notifications', n);
export const notificationsTest = (id: string) =>
  restPost<{ delivered: boolean; error?: string }, {}>(`/api/v1/notifications/${id}/test`, {});
export async function notificationsDelete(id: string): Promise<void> {
  const res = await fetch('/api/v1/notifications/' + id, {
    method: 'DELETE',
    credentials: 'same-origin',
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}

export const investigationsPivot = (host: string, atUnix?: number, deltaSec = 900) => {
  const q = new URLSearchParams();
  q.set('host', host);
  if (atUnix) q.set('at', String(atUnix));
  q.set('delta', String(deltaSec));
  return rest<PivotEntry[]>(`/api/v1/investigations/pivot?${q}`);
};
