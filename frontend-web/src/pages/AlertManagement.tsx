/**
 * AlertManagement.tsx — Alerting & Notification Management (Hybrid — Phase 0.3)
 *
 * Real-time alert dashboard with acknowledge, status tracking, and live
 * WebSocket feed from /api/v1/events. Connects to /api/v1/alerts for
 * the initial list and extends with live event-bus updates.
 */

import { createSignal, createResource, For, Show, onMount, onCleanup } from 'solid-js';
import { request } from '../services/api';
import { isDesktop } from '../context';

// ── Types ─────────────────────────────────────────────────────────────────────
interface Alert {
  id: number;
  tenant_id: string;
  host_id: string;
  timestamp: string;
  event_type: string;
  source_ip: string;
  user: string;
  raw_log: string;
  // Locally tracked (not from API)
  status?: 'new' | 'investigating' | 'acknowledged' | 'closed';
}

interface AlertsResponse {
  active_alerts: number;
  alerts: Alert[];
}

// ── Helpers ───────────────────────────────────────────────────────────────────
const SEV: Record<string, { label: string; color: string; bg: string }> = {
  security_alert:   { label: 'CRITICAL', color: '#ff3355', bg: '#2a0d15' },
  failed_login:     { label: 'HIGH',     color: '#ff6600', bg: '#2a1500' },
  sudo_exec:        { label: 'MEDIUM',   color: '#ffaa00', bg: '#2a2000' },
  successful_login: { label: 'INFO',     color: '#00ff88', bg: '#002a1a' },
};
function sev(t: string) { return SEV[t] ?? { label: 'LOW', color: '#607070', bg: '#0d1a1f' }; }

function fmt(iso: string) {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

// ── Component ─────────────────────────────────────────────────────────────────
export default function AlertManagement() {
  const [alerts, { refetch }] = createResource<Alert[]>(async () => {
    const res = await request<AlertsResponse>('/alerts');
    return (res.alerts ?? []).map(a => ({ ...a, status: 'new' as const }));
  });

  const [localStatuses, setLocalStatuses] = createSignal<Record<number, Alert['status']>>({});
  const [filter, setFilter]     = createSignal<string>('all');
  const [selected, setSelected] = createSignal<Alert | null>(null);
  const [liveCount, setLiveCount] = createSignal(0);

  // ── Live WebSocket feed ───────────────────────────────────────────────────
  let ws: WebSocket | null = null;

  onMount(() => {
    const wsBase = isDesktop() ? 'ws://localhost:8080' : window.location.origin.replace('http', 'ws');
    const token  = localStorage.getItem('oblivra_token') ?? '';
    ws = new WebSocket(`${wsBase}/api/v1/events?token=${token}`);

    ws.onmessage = (evt) => {
      try {
        const ev = JSON.parse(evt.data);
        if (ev.topic?.includes('alert') || ev.topic?.includes('security')) {
          setLiveCount(c => c + 1);
          // Trigger refetch every 10 live events to debounce
          if (liveCount() % 10 === 0) refetch();
        }
      } catch { /* ignore */ }
    };
    ws.onerror = () => { /* silent — WS may not be available in all environments */ };
  });

  onCleanup(() => ws?.close());

  // ── Derived list ─────────────────────────────────────────────────────────
  const displayed = () => {
    const f = filter();
    const base = (alerts() ?? []).map(a => ({
      ...a,
      status: localStatuses()[a.id] ?? a.status ?? 'new',
    }));
    if (f === 'all') return base;
    return base.filter(a => a.status === f);
  };

  function setStatus(id: number, s: Alert['status']) {
    setLocalStatuses(prev => ({ ...prev, [id]: s }));
    if (selected()?.id === id) setSelected(prev => prev ? { ...prev, status: s } : null);
  }

  const statusColor: Record<string, string> = {
    new: '#ff3355', investigating: '#ffaa00', acknowledged: '#00ffe7', closed: '#607070',
  };

  return (
    <div style="display:flex; height:100vh; background:#080f12; color:#c8d8d8; font-family:'JetBrains Mono',monospace; overflow:hidden;">
      {/* Left panel — Alert list */}
      <div style="flex:1; display:flex; flex-direction:column; border-right:1px solid #1e3040; overflow:hidden;">
        {/* Header */}
        <div style="padding:1.5rem 1.5rem 1rem; border-bottom:1px solid #1e3040; flex-shrink:0;">
          <div style="display:flex; justify-content:space-between; align-items:flex-start;">
            <div>
              <h1 style="font-size:1.2rem; letter-spacing:0.15em; margin:0; color:#ff3355;">⚠ ALERT MANAGEMENT</h1>
              <p style="margin:0.2rem 0 0; font-size:0.72rem; color:#607070;">
                Live event feed · {alerts()?.length ?? 0} alerts
                <Show when={liveCount() > 0}>
                  <span style="color:#00ffe7; margin-left:0.75rem;">+{liveCount()} live events</span>
                </Show>
              </p>
            </div>
            <button onClick={() => refetch()} style="background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.74rem; letter-spacing:0.1em;">↻ REFRESH</button>
          </div>

          {/* Filter tabs */}
          <div style="display:flex; gap:0; margin-top:1rem; border:1px solid #1e3040; border-radius:4px; overflow:hidden;">
            {(['all', 'new', 'investigating', 'acknowledged', 'closed'] as const).map(f => (
              <button
                onClick={() => setFilter(f)}
                style={`flex:1; padding:0.4rem; border:none; cursor:pointer; font-size:0.68rem; letter-spacing:0.1em; transition:all 0.15s; background:${filter()===f ? '#1e3040' : '#0a1318'}; color:${filter()===f ? '#00ffe7' : '#607070'};`}
              >
                {f.toUpperCase()}
              </button>
            ))}
          </div>
        </div>

        {/* Alert rows */}
        <div style="flex:1; overflow-y:auto;" role="list" aria-label="Alert list">
          <Show when={!alerts.loading} fallback={<div style="padding:2rem; text-align:center; color:#607070;">Loading alerts…</div>}>
            <For each={displayed()} fallback={<div style="padding:2rem; text-align:center; color:#607070;">No alerts match the current filter.</div>}>
              {(a) => {
                const s = sev(a.event_type);
                const status = localStatuses()[a.id] ?? a.status ?? 'new';
                const isSelected = () => selected()?.id === a.id;
                return (
                  <div
                    role="listitem"
                    tabIndex={0}
                    onClick={() => setSelected(a)}
                    onKeyDown={e => e.key === 'Enter' && setSelected(a)}
                    style={`padding:0.9rem 1.2rem; border-bottom:1px solid #0a1318; cursor:pointer; border-left:3px solid ${isSelected() ? '#00ffe7' : s.color}; background:${isSelected() ? '#111f28' : 'transparent'}; transition:background 0.12s;`}
                    aria-selected={isSelected()}
                  >
                    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.3rem;">
                      <span style={`font-size:0.68rem; font-weight:700; letter-spacing:0.12em; color:${s.color}; background:${s.bg}; padding:0.1rem 0.4rem; border-radius:2px;`}>{s.label}</span>
                      <span style={`font-size:0.67rem; color:${statusColor[status] ?? '#607070'};`}>● {status.toUpperCase()}</span>
                    </div>
                    <div style="font-size:0.78rem; color:#c8d8d8;">{a.event_type.replace(/_/g,' ')}</div>
                    <div style="display:flex; gap:1rem; margin-top:0.25rem; font-size:0.68rem; color:#607070;">
                      <span>{a.host_id || 'unknown'}</span>
                      <span>{fmt(a.timestamp)}</span>
                    </div>
                  </div>
                );
              }}
            </For>
          </Show>
        </div>
      </div>

      {/* Right panel — Detail + Actions */}
      <div style="width:400px; flex-shrink:0; display:flex; flex-direction:column; overflow:hidden;">
        <Show when={selected()} fallback={
          <div style="flex:1; display:flex; align-items:center; justify-content:center; color:#607070; font-size:0.8rem; text-align:center; padding:2rem;">
            Select an alert<br/>to view details and take action
          </div>
        }>
          {(a) => {
            const s = sev(a().event_type);
            const status = () => localStatuses()[a().id] ?? a().status ?? 'new';
            return (
              <>
                {/* Detail header */}
                <div style={`padding:1.5rem; border-bottom:1px solid #1e3040; background:${s.bg};`}>
                  <div style={`font-size:0.72rem; font-weight:700; letter-spacing:0.15em; color:${s.color}; margin-bottom:0.5rem;`}>{s.label}</div>
                  <div style="font-size:1rem; color:#c8d8d8; margin-bottom:0.25rem;">{a().event_type.replace(/_/g,' ')}</div>
                  <div style="font-size:0.72rem; color:#607070;">{fmt(a().timestamp)}</div>
                </div>

                {/* Fields */}
                <div style="flex:1; overflow-y:auto; padding:1.5rem;">
                  {[
                    ['Tenant',     a().tenant_id || 'GLOBAL'],
                    ['Host',       a().host_id || '—'],
                    ['Source IP',  a().source_ip || '—'],
                    ['User',       a().user || '—'],
                    ['Status',     status().toUpperCase()],
                  ].map(([k, v]) => (
                    <div style="margin-bottom:0.9rem;">
                      <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.2rem;">{k}</div>
                      <div style="font-size:0.82rem; color:#c8d8d8;">{v}</div>
                    </div>
                  ))}
                  <div style="margin-bottom:0.9rem;">
                    <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.4rem;">RAW LOG</div>
                    <pre style="font-size:0.7rem; color:#607070; background:#0a1318; padding:0.75rem; border-radius:4px; overflow-x:auto; white-space:pre-wrap; word-break:break-all; border:1px solid #1e3040;">{a().raw_log || '(no raw data)'}</pre>
                  </div>
                </div>

                {/* Action buttons */}
                <div style="padding:1rem 1.5rem; border-top:1px solid #1e3040; display:grid; grid-template-columns:1fr 1fr; gap:0.5rem;">
                  {([
                    ['INVESTIGATE', 'investigating', '#ffaa00'],
                    ['ACKNOWLEDGE', 'acknowledged',  '#00ffe7'],
                    ['CLOSE',       'closed',        '#607070'],
                    ['REOPEN',      'new',           '#ff6600'],
                  ] as [string, Alert['status'], string][]).map(([label, st, color]) => (
                    <button
                      onClick={() => setStatus(a().id, st)}
                      disabled={status() === st}
                      style={`border:1px solid ${color}; background:none; color:${status()===st ? '#1e3040' : color}; padding:0.5rem; border-radius:4px; cursor:${status()===st ? 'default' : 'pointer'}; font-size:0.72rem; letter-spacing:0.1em; transition:all 0.15s; opacity:${status()===st ? '0.4' : '1'};`}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </>
            );
          }}
        </Show>
      </div>
    </div>
  );
}
