/**
 * FleetManagement.tsx — Multi-Tenant Agent Fleet (Web Only — Phase 0.5)
 *
 * Provides a full-screen management console for remotely deployed OBLIVRA
 * agents. Connects to /api/v1/agents and /api/v1/agents/{id}/status.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface AgentInfo {
  id: string;
  hostname: string;
  version: string;
  last_seen: string;
  remote_address: string;
  tenant_id?: string;
  status?: 'online' | 'offline' | 'degraded';
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
      a.hostname.toLowerCase().includes(search().toLowerCase()) ||
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
            Multi-tenant agent status — Web Context Only
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
        {[
          { label: 'TOTAL AGENTS',  val: agents()?.length ?? 0,   color: '#00ffe7' },
          { label: 'ONLINE',        val: (agents() ?? []).filter(a => a.status === 'online').length, color: '#00ff88' },
          { label: 'OFFLINE',       val: (agents() ?? []).filter(a => a.status !== 'online').length, color: '#ff3355' },
        ].map(stat => (
          <div style={`background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid ${stat.color}; padding:1rem; border-radius:4px;`}>
            <div style={`font-size:1.6rem; font-weight:700; color:${stat.color};`}>{stat.val}</div>
            <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em;">{stat.label}</div>
          </div>
        ))}
      </div>

      {/* Agent Table */}
      <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
        <table style="width:100%; border-collapse:collapse; font-size:0.78rem;">
          <thead>
            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
              {['STATUS', 'HOSTNAME', 'TENANT', 'VERSION', 'LAST SEEN', 'REMOTE ADDR'].map(h => (
                <th style="padding:0.75rem 1rem; text-align:left; color:#607070; letter-spacing:0.12em; font-weight:400;">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            <Show when={!agents.loading} fallback={
              <tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">Loading agents…</td></tr>
            }>
              <For each={filtered()} fallback={
                <tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">No agents registered. Deploy agents using the Onboarding wizard.</td></tr>
              }>
                {(agent) => (
                  <tr style="border-bottom:1px solid #0d1a1f; transition:background 0.15s;"
                      onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = '#111f28'}
                      onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = ''}>
                    <td style="padding:0.7rem 1rem;">
                      <span style={`display:inline-block; width:8px; height:8px; border-radius:50%; background:${statusColor(agent.status)}; margin-right:0.5rem;`}></span>
                      <span style={`color:${statusColor(agent.status)}; font-size:0.7rem; letter-spacing:0.1em;`}>
                        {(agent.status ?? 'unknown').toUpperCase()}
                      </span>
                    </td>
                    <td style="padding:0.7rem 1rem; color:#c8d8d8;">{agent.hostname}</td>
                    <td style="padding:0.7rem 1rem; color:#00ffe7;">{agent.tenant_id ?? 'GLOBAL'}</td>
                    <td style="padding:0.7rem 1rem; color:#607070;">{agent.version}</td>
                    <td style="padding:0.7rem 1rem; color:#607070;">{timeSince(agent.last_seen)}</td>
                    <td style="padding:0.7rem 1rem; color:#607070; font-size:0.72rem;">{agent.remote_address}</td>
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
