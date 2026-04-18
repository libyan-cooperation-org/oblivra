/**
 * FleetManagement.tsx — Multi-Tenant Agent Fleet (Web Only — Phase 0.5)
 *
 * Provides a full-screen management console for remotely deployed OBLIVRA
 * agents. Connects to /api/v1/agents and shows deep telemetry details.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface AgentInfo {
  id: string;
  hostname: string;
  os: string;
  arch: string;
  version: string;
  collectors: string[];
  last_seen: string;
  status: string;
  tenant_id?: string;
}

async function fetchAgents(): Promise<AgentInfo[]> {
  try {
    return await request<AgentInfo[]>('/agents');
  } catch {
    return [];
  }
}

function statusColor(status?: string): string {
  switch (status) {
    case 'online':   return '#00ff88';
    case 'degraded': return '#ffaa00';
    case 'offline':  return '#ff3355';
    case 'isolated': return '#f85149';
    default:         return '#607070';
  }
}

function timeSince(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1)  return 'just now';
  if (mins < 60) return `${mins}m ago`;
  return `${Math.floor(mins / 60)}h ago`;
}

export default function FleetManagement() {
  const [agents, { refetch }] = createResource(fetchAgents);
  const [search, setSearch] = createSignal('');

  const filtered = () =>
    (agents() ?? []).filter(a =>
      a.hostname?.toLowerCase().includes(search().toLowerCase()) ||
      (a.tenant_id ?? '').toLowerCase().includes(search().toLowerCase())
    );

  return (
    <div class="fleet-page" style="padding: 2rem; color: #c8d8d8; font-family: 'JetBrains Mono', monospace;">
      {/* Header */}
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:1.5rem;">
        <div>
          <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">
            ⬡ FLEET MANAGEMENT
          </h1>
          <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
            Multi-tenant agent status & telemetry — Web Context Only
          </p>
        </div>
        <div style="display:flex; gap:0.75rem; align-items:center;">
          <input
            type="text"
            placeholder="Search by host or tenant…"
            value={search()}
            onInput={e => setSearch(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.5rem 0.75rem; border-radius:4px; font-size:0.8rem; width:220px;"
          />
          <button
            onClick={() => refetch()}
            style="background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:0.5rem 1rem; border-radius:4px; cursor:pointer; font-size:0.8rem; letter-spacing:0.1em;"
          >
            ↻ REFRESH
          </button>
        </div>
      </div>

      {/* Stats Strip */}
      <div style="display:grid; grid-template-columns:repeat(3,1fr); gap:1rem; margin-bottom:1.5rem;">
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #00ffe7; padding:1rem; border-radius:4px;">
           <div style="font-size:1.6rem; font-weight:700; color:#00ffe7;">{agents()?.length ?? 0}</div>
           <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em;">TOTAL AGENTS</div>
        </div>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #00ff88; padding:1rem; border-radius:4px;">
           <div style="font-size:1.6rem; font-weight:700; color:#00ff88;">{(agents() ?? []).filter(a => a.status === 'online').length}</div>
           <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em;">ONLINE</div>
        </div>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ff3355; padding:1rem; border-radius:4px;">
           <div style="font-size:1.6rem; font-weight:700; color:#ff3355;">{(agents() ?? []).filter(a => a.status !== 'online').length}</div>
           <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em;">OFFLINE / DEGRADED</div>
        </div>
      </div>

      {/* Agent Table */}
      <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
        <table style="width:100%; border-collapse:collapse; font-size:0.78rem;">
          <thead>
            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
              {['STATUS', 'HOSTNAME', 'TENANT', 'PLATFORM', 'COLLECTORS', 'VERSION', 'LAST SEEN'].map(h => (
                <th style="padding:0.75rem 1rem; text-align:left; color:#607070; letter-spacing:0.12em; font-weight:400; white-space:nowrap;">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            <Show when={!agents.loading} fallback={
              <tr><td colspan="7" style="padding:2rem; text-align:center; color:#607070;">Loading agents…</td></tr>
            }>
              <For each={filtered()} fallback={
                <tr><td colspan="7" style="padding:2rem; text-align:center; color:#607070;">No agents registered. Deploy agents using the Onboarding wizard.</td></tr>
              }>
                {(agent) => (
                  <tr style="border-bottom:1px solid #0d1a1f; transition:background 0.15s;"
                      onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = '#111f28'}
                      onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = ''}>
                    <td style="padding:0.7rem 1rem; white-space:nowrap;">
                      <span style={`display:inline-block; width:8px; height:8px; border-radius:50%; background:${statusColor(agent.status)}; margin-right:0.5rem;`}></span>
                      <span style={`color:${statusColor(agent.status)}; font-size:0.7rem; letter-spacing:0.1em;`}>
                        {(agent.status ?? 'unknown').toUpperCase()}
                      </span>
                    </td>
                    <td style="padding:0.7rem 1rem; color:#c8d8d8; white-space:nowrap;">
                       {agent.hostname}
                       <div style="font-size:0.6rem; color:#607070; margin-top:2px;">{agent.id}</div>
                    </td>
                    <td style="padding:0.7rem 1rem; color:#00ffe7;">{agent.tenant_id ?? 'GLOBAL'}</td>
                    <td style="padding:0.7rem 1rem; color:#c8d8d8; white-space:nowrap;">
                       <span style="background:#1e3040; padding:2px 6px; border-radius:3px; font-size:0.7rem; margin-right:4px;">{agent.os || 'unknown'}</span>
                       <span style="background:#1e3040; padding:2px 6px; border-radius:3px; font-size:0.7rem;">{agent.arch || 'unknown'}</span>
                    </td>
                    <td style="padding:0.7rem 1rem; color:#607070;">
                       <div style="display:flex; gap:4px; flex-wrap:wrap;">
                          <For each={agent.collectors || []}>
                             {c => <span style="background:#0a1318; border:1px solid #1e3040; padding:2px 6px; border-radius:12px; font-size:0.65rem; color:#00ffe7;">{c}</span>}
                          </For>
                       </div>
                    </td>
                    <td style="padding:0.7rem 1rem; color:#607070; white-space:nowrap;">v{agent.version}</td>
                    <td style="padding:0.7rem 1rem; color:#607070; white-space:nowrap;">{timeSince(agent.last_seen)}</td>
                  </tr>
                )}
              </For>
            </Show>
          </tbody>
        </table>
      </div>
    </div>
  );
}
