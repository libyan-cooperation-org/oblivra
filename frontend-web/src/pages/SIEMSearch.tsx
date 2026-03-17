/**
 * SIEMSearch.tsx — Dedicated SIEM Search & Analytics Page (Hybrid — Phase 0.5)
 *
 * A full-featured query interface connecting to /api/v1/siem/search.
 * Supports Lucene-style queries with real-time result streaming.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface HostEvent {
  id: number;
  tenant_id: string;
  host_id: string;
  timestamp: string;
  event_type: string;
  source_ip: string;
  user: string;
  raw_log: string;
}

interface SearchResponse {
  events: HostEvent[];
  total: number;
}

const SEVERITY_MAP: Record<string, { label: string; color: string }> = {
  failed_login:     { label: 'CRIT', color: '#ff3355' },
  security_alert:   { label: 'HIGH', color: '#ff6600' },
  sudo_exec:        { label: 'MED',  color: '#ffaa00' },
  successful_login: { label: 'INFO', color: '#00ff88' },
};

function getSeverity(eventType: string) {
  return SEVERITY_MAP[eventType] ?? { label: 'LOW', color: '#607070' };
}

export default function SIEMSearch() {
  const [query, setQuery]     = createSignal('');
  const [limit, setLimit]     = createSignal(100);
  const [trigger, setTrigger] = createSignal(0);
  const [lastQuery, setLastQuery] = createSignal('');

  const [results] = createResource(
    () => ({ q: lastQuery(), l: limit(), t: trigger() }),
    async ({ q, l }) => {
      if (!q.trim()) return { events: [], total: 0 };
      try {
        const params = new URLSearchParams({ q, limit: String(l) });
        return await request<SearchResponse>(`/siem/search?${params}`);
      } catch {
        return { events: [], total: 0 };
      }
    }
  );

  function runSearch() {
    setLastQuery(query());
    setTrigger(t => t + 1);
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === 'Enter') runSearch();
  }

  const events = () => results()?.events ?? [];

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ SIEM SEARCH</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
          Lucene-style full-text query across the OBLIVRA event substrate
        </p>
      </div>

      {/* Query Bar */}
      <div style="display:flex; gap:0.75rem; margin-bottom:1.5rem;">
        <input
          id="siem-query-input"
          type="text"
          value={query()}
          onInput={e => setQuery(e.currentTarget.value)}
          onKeyDown={handleKey}
          placeholder='e.g.  event_type:failed_login  OR  source_ip:192.168.*'
          style="flex:1; background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.65rem 1rem; border-radius:4px; font-size:0.83rem; font-family:inherit;"
        />
        <select
          value={limit()}
          onChange={e => setLimit(Number(e.currentTarget.value))}
          style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.65rem; border-radius:4px; font-family:inherit; font-size:0.8rem;"
        >
          {[50, 100, 250, 500].map(n => <option value={n}>{n} results</option>)}
        </select>
        <button
          id="siem-search-btn"
          onClick={runSearch}
          style="background:#00ffe7; color:#080f12; border:none; padding:0.65rem 1.5rem; border-radius:4px; font-weight:700; cursor:pointer; font-size:0.83rem; letter-spacing:0.1em;"
        >
          SEARCH
        </button>
      </div>

      {/* Results Info */}
      <Show when={lastQuery()}>
        <div style="margin-bottom:1rem; font-size:0.75rem; color:#607070;">
          <span style="color:#00ffe7;">{events().length}</span> results for{' '}
          <span style="color:#c8d8d8;">"{lastQuery()}"</span>
          <Show when={results.loading}> — <span style="color:#ffaa00;">searching…</span></Show>
        </div>
      </Show>

      {/* Results Table */}
      <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
        <table style="width:100%; border-collapse:collapse; font-size:0.76rem;">
          <thead>
            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
              {['SEV', 'TIMESTAMP', 'TENANT', 'HOST', 'TYPE', 'SOURCE IP', 'USER', 'RAW LOG'].map(h => (
                <th style="padding:0.65rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400; white-space:nowrap;">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            <Show when={!results.loading} fallback={
              <tr><td colspan="8" style="padding:2rem; text-align:center; color:#607070;">Searching…</td></tr>
            }>
              <Show when={!lastQuery()}>
                <tr><td colspan="8" style="padding:2.5rem; text-align:center; color:#607070;">
                  Enter a query above and press <kbd style="background:#1e3040; padding:0.1rem 0.4rem; border-radius:3px; color:#00ffe7;">Enter</kbd> to search.
                </td></tr>
              </Show>
              <For each={events()} fallback={
                <Show when={lastQuery()}>
                  <tr><td colspan="8" style="padding:2rem; text-align:center; color:#607070;">No results found for "{lastQuery()}"</td></tr>
                </Show>
              }>
                {(evt) => {
                  const sev = getSeverity(evt.event_type);
                  return (
                    <tr style="border-bottom:1px solid #0a1318; transition:background 0.1s;"
                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}>
                      <td style="padding:0.6rem 0.9rem;">
                        <span style={`font-size:0.65rem; letter-spacing:0.1em; color:${sev.color}; font-weight:700;`}>{sev.label}</span>
                      </td>
                      <td style="padding:0.6rem 0.9rem; color:#607070; white-space:nowrap;">
                        {new Date(evt.timestamp).toISOString().replace('T',' ').slice(0,19)}
                      </td>
                      <td style="padding:0.6rem 0.9rem; color:#00ffe7;">{evt.tenant_id || 'GLOBAL'}</td>
                      <td style="padding:0.6rem 0.9rem; color:#c8d8d8;">{evt.host_id}</td>
                      <td style={`padding:0.6rem 0.9rem; color:${sev.color};`}>{evt.event_type}</td>
                      <td style="padding:0.6rem 0.9rem; color:#607070;">{evt.source_ip || '—'}</td>
                      <td style="padding:0.6rem 0.9rem; color:#c8d8d8;">{evt.user || '—'}</td>
                      <td style="padding:0.6rem 0.9rem; color:#607070; max-width:280px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={evt.raw_log}>
                        {evt.raw_log}
                      </td>
                    </tr>
                  );
                }}
              </For>
            </Show>
          </tbody>
        </table>
      </div>
    </div>
  );
}
