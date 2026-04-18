/**
 * ThreatIntelDashboard.tsx — Threat Intelligence Browser (Hybrid — Phase 3.1)
 *
 * Live IOC browsing, indicator search, campaign correlation, and TAXII feed
 * status. Connects to /api/v1/threatintel/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface Indicator {
  type: string;
  value: string;
  source: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  campaign_id?: string;
  expires_at: string;
}
interface Campaign {
  id: string;
  name: string;
  actor?: string;
  ttps?: string[];
  description?: string;
}
interface IOCStats { [type: string]: number; }

// ── Helpers ───────────────────────────────────────────────────────────────────
const SEV: Record<string, { color: string; bg: string }> = {
  critical: { color: '#ff3355', bg: '#2a0d15' },
  high:     { color: '#ff6600', bg: '#2a1500' },
  medium:   { color: '#ffaa00', bg: '#2a2000' },
  low:      { color: '#00ff88', bg: '#002a1a' },
};
const sev = (s: string) => SEV[s] ?? { color: '#607070', bg: '#0d1a1f' };

const IOC_ICONS: Record<string, string> = {
  'ipv4-addr':   '🌐',
  'ipv6-addr':   '🌐',
  'domain-name': '🔗',
  'md5':         '#',
  'sha256':      '#',
  'url':         '↗',
};

// ── Component ─────────────────────────────────────────────────────────────────
type Tab = 'indicators' | 'campaigns' | 'stats';

export default function ThreatIntelDashboard() {
  const [tab, setTab] = createSignal<Tab>('indicators');
  const [search, setSearch] = createSignal('');
  const [typeFilter, setTypeFilter] = createSignal('all');
  const [sevFilter, setSevFilter] = createSignal('all');
  const [queryValue, setQueryValue] = createSignal('');
  const [queryResult, setQueryResult] = createSignal<Indicator | null | 'none'>(null);

  const [stats] = createResource<IOCStats>(async () => {
    const r = await request<{ stats: IOCStats }>('/threatintel/stats');
    return r.stats ?? {};
  });

  const [indicators, { refetch }] = createResource<Indicator[]>(async () => {
    const params = new URLSearchParams({ limit: '500' });
    const r = await request<{ indicators: Indicator[] }>(`/threatintel/indicators?${params}`);
    return r.indicators ?? [];
  });

  const [campaigns] = createResource<Campaign[]>(async () => {
    const r = await request<{ campaigns: Campaign[] }>('/threatintel/campaigns');
    return r.campaigns ?? [];
  });

  const filtered = () => (indicators() ?? []).filter(i => {
    const matchSearch = !search() || i.value.includes(search()) || i.source.includes(search());
    const matchType = typeFilter() === 'all' || i.type === typeFilter();
    const matchSev  = sevFilter() === 'all' || i.severity === sevFilter();
    return matchSearch && matchType && matchSev;
  });

  const types = () => ['all', ...new Set((indicators() ?? []).map(i => i.type))];

  async function lookupIOC() {
    const v = queryValue().trim();
    if (!v) return;
    try {
      const r = await request<{ match: boolean; indicator?: Indicator }>(
        `/threatintel/lookup?value=${encodeURIComponent(v)}`
      );
      setQueryResult(r.match && r.indicator ? r.indicator : 'none');
    } catch { setQueryResult('none'); }
  }

  const TAB = (t: Tab) =>
    `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#ff6600' : 'transparent'}; background:none; color:${tab()===t ? '#ff6600' : '#607070'}; transition:color 0.15s;`;

  const totalIOCs = () => Object.values(stats() ?? {}).reduce((a, b) => a + b, 0);

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff6600;">⬡ THREAT INTELLIGENCE</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
          IOC browser · Campaign correlation · STIX/TAXII feed status
        </p>
      </div>

      {/* Stat strip */}
      <div style="display:grid; grid-template-columns:repeat(5,1fr); gap:1rem; margin-bottom:1.5rem;">
        <Show when={!stats.loading} fallback={null}>
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ff6600; padding:0.9rem; border-radius:4px;">
            <div style="font-size:1.4rem; font-weight:700; color:#ff6600;">{totalIOCs().toLocaleString()}</div>
            <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">TOTAL IOCs</div>
          </div>
          <For each={Object.entries(stats() ?? {}).slice(0, 4)}>
            {([type, count]) => (
              <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:0.9rem; border-radius:4px;">
                <div style="font-size:1.4rem; font-weight:700; color:#c8d8d8;">{count}</div>
                <div style="font-size:0.65rem; color:#607070; letter-spacing:0.1em;">{(IOC_ICONS[type] ?? '') + ' ' + type}</div>
              </div>
            )}
          </For>
        </Show>
      </div>

      {/* IOC Lookup Bar */}
      <div style="display:flex; gap:0.75rem; margin-bottom:1.5rem; background:#0d1a1f; border:1px solid #1e3040; padding:1rem; border-radius:6px;">
        <div style="flex:1;">
          <div style="font-size:0.65rem; color:#607070; margin-bottom:0.4rem; letter-spacing:0.1em;">INSTANT IOC LOOKUP</div>
          <div style="display:flex; gap:0.5rem;">
            <input
              type="text" placeholder="Enter IP, domain, hash, URL…"
              value={queryValue()} onInput={e => setQueryValue(e.currentTarget.value)}
              onKeyDown={e => e.key === 'Enter' && lookupIOC()}
              style="flex:1; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.45rem 0.75rem; border-radius:3px; font-size:0.8rem;"
            />
            <button onClick={lookupIOC}
              style="background:#ff6600; color:#080f12; border:none; padding:0.45rem 1.25rem; border-radius:3px; cursor:pointer; font-weight:700; font-size:0.78rem; letter-spacing:0.1em;">
              LOOKUP
            </button>
          </div>
        </div>
        <Show when={queryResult() !== null}>
          <div style={`padding:0.65rem 1rem; border-radius:4px; min-width:220px; background:${queryResult() === 'none' ? '#0a1318' : sev((queryResult() as Indicator).severity).bg}; border:1px solid ${queryResult() === 'none' ? '#1e3040' : sev((queryResult() as Indicator).severity).color};`}>
            <Show when={queryResult() === 'none'}>
              <div style="color:#607070; font-size:0.76rem;">○ No match — indicator not found</div>
            </Show>
            <Show when={queryResult() !== 'none' && queryResult() !== null}>
              <div>
                <div style={`font-size:0.7rem; font-weight:700; color:${sev((queryResult() as Indicator).severity).color}; letter-spacing:0.1em; margin-bottom:0.35rem;`}>● {(queryResult() as Indicator).severity.toUpperCase()} MATCH</div>
                <div style="font-size:0.75rem; color:#c8d8d8;">{(queryResult() as Indicator).type}: {(queryResult() as Indicator).value}</div>
                <div style="font-size:0.68rem; color:#607070; margin-top:0.2rem;">{(queryResult() as Indicator).source} — {(queryResult() as Indicator).description}</div>
              </div>
            </Show>
          </div>
        </Show>
      </div>

      {/* Tabs */}
      <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
        {(['indicators', 'campaigns', 'stats'] as Tab[]).map(t => (
          <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
        ))}
        <div style="margin-left:auto; display:flex; align-items:center; gap:0.5rem; padding-bottom:0.5rem;">
          <button onClick={() => refetch()}
            style="background:#1e3040; border:1px solid #607070; color:#607070; padding:0.3rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.72rem;">
            ↻ REFRESH
          </button>
        </div>
      </div>

      {/* ── Indicators ── */}
      <Show when={tab() === 'indicators'}>
        {/* Filter bar */}
        <div style="display:flex; gap:0.75rem; margin-bottom:1rem;">
          <input type="text" value={search()} onInput={e => setSearch(e.currentTarget.value)}
            placeholder="Filter by value or source…"
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.75rem; border-radius:3px; font-size:0.78rem; flex:1;" />
          <select value={typeFilter()} onChange={e => setTypeFilter(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.78rem;">
            <For each={types()}>{t => <option value={t}>{t === 'all' ? 'All Types' : t}</option>}</For>
          </select>
          <select value={sevFilter()} onChange={e => setSevFilter(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.78rem;">
            {['all', 'critical', 'high', 'medium', 'low'].map(s => (
              <option value={s}>{s === 'all' ? 'All Severities' : s}</option>
            ))}
          </select>
          <span style="color:#607070; font-size:0.75rem; align-self:center;">{filtered().length} indicators</span>
        </div>

        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
          <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
            <thead>
              <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                {['SEV', 'TYPE', 'INDICATOR', 'SOURCE', 'DESCRIPTION', 'EXPIRES'].map(h => (
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <Show when={!indicators.loading} fallback={<tr><td colspan="6" style="padding:1.5rem; text-align:center; color:#607070;">Loading indicators…</td></tr>}>
                <For each={filtered().slice(0, 500)} fallback={<tr><td colspan="6" style="padding:1.5rem; text-align:center; color:#607070;">No IOCs loaded. Upload a feed or connect a TAXII source.</td></tr>}>
                  {(ind) => {
                    const s = sev(ind.severity);
                    return (
                      <tr style="border-bottom:1px solid #0a1318;" onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'} onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}>
                        <td style="padding:0.55rem 0.9rem;">
                          <span style={`font-size:0.65rem; font-weight:700; color:${s.color}; letter-spacing:0.1em;`}>{ind.severity.toUpperCase()}</span>
                        </td>
                        <td style="padding:0.55rem 0.9rem; color:#607070;">{IOC_ICONS[ind.type] ?? ''} {ind.type}</td>
                        <td style={`padding:0.55rem 0.9rem; color:${s.color}; font-size:0.74rem;`}>{ind.value}</td>
                        <td style="padding:0.55rem 0.9rem; color:#607070;">{ind.source}</td>
                        <td style="padding:0.55rem 0.9rem; color:#607070; max-width:220px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={ind.description}>{ind.description}</td>
                        <td style="padding:0.55rem 0.9rem; color:#607070;">{ind.expires_at ? new Date(ind.expires_at).toLocaleDateString() : '—'}</td>
                      </tr>
                    );
                  }}
                </For>
              </Show>
            </tbody>
          </table>
        </div>
      </Show>

      {/* ── Campaigns ── */}
      <Show when={tab() === 'campaigns'}>
        <Show when={!campaigns.loading} fallback={<div style="color:#607070; padding:1rem;">Loading…</div>}>
          <div style="display:grid; gap:1rem;">
            <For each={campaigns()} fallback={<div style="color:#607070; padding:1rem;">No campaigns registered.</div>}>
              {(c) => (
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                  <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.75rem;">
                    <div>
                      <div style="color:#ff6600; font-size:0.88rem; letter-spacing:0.08em;">{c.name}</div>
                      <div style="color:#607070; font-size:0.72rem; margin-top:0.15rem;">
                        ID: {c.id}
                        <Show when={c.actor}><span style="color:#c8d8d8; margin-left:0.75rem;">Actor: {c.actor}</span></Show>
                      </div>
                    </div>
                    <Show when={c.ttps?.length}>
                      <div style="display:flex; flex-wrap:wrap; gap:0.3rem; max-width:300px; justify-content:flex-end;">
                        <For each={c.ttps!.slice(0,6)}>
                          {(ttp) => (
                            <span style="background:#111f28; border:1px solid #1e3040; color:#607070; padding:0.1rem 0.4rem; border-radius:2px; font-size:0.65rem;">{ttp}</span>
                          )}
                        </For>
                      </div>
                    </Show>
                  </div>
                  <Show when={c.description}>
                    <div style="color:#607070; font-size:0.74rem; line-height:1.5;">{c.description}</div>
                  </Show>
                </div>
              )}
            </For>
          </div>
        </Show>
      </Show>

      {/* ── Stats ── */}
      <Show when={tab() === 'stats'}>
        <div style="display:grid; grid-template-columns:repeat(auto-fill,minmax(200px,1fr)); gap:1rem;">
          <Show when={!stats.loading} fallback={<div style="color:#607070;">Loading…</div>}>
            <For each={Object.entries(stats() ?? {})}>
              {([type, count]) => (
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                  <div style="font-size:1.6rem; font-weight:700; color:#ff6600;">{count.toLocaleString()}</div>
                  <div style="font-size:0.72rem; color:#c8d8d8; margin-top:0.25rem;">{IOC_ICONS[type] ?? ''} {type}</div>
                  <div style="height:4px; background:#1e3040; border-radius:2px; margin-top:0.75rem;">
                    <div style={`height:100%; border-radius:2px; background:#ff6600; width:${Math.min(100, Math.round(count / totalIOCs() * 100))}%;`}></div>
                  </div>
                </div>
              )}
            </For>
          </Show>
        </div>
      </Show>
    </div>
  );
}
