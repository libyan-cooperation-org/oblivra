<!-- OBLIVRA Web — FleetManagement (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface Agent { id:string; hostname:string; os:string; arch:string; version:string; collectors:string[]; last_seen:string; status:string; tenant_id?:string; }

  let agents  = $state<Agent[]>([]);
  let loading = $state(true);
  let search  = $state('');

  async function fetchAgents() {
    loading = true;
    try { agents = await request<Agent[]>('/agents'); }
    catch { agents = []; }
    loading = false;
  }

  onMount(fetchAgents);

  const filtered = $derived(agents.filter(a =>
    a.hostname?.toLowerCase().includes(search.toLowerCase()) ||
    (a.tenant_id ?? '').toLowerCase().includes(search.toLowerCase())
  ));

  const online   = $derived(agents.filter(a => a.status === 'online').length);
  const offline  = $derived(agents.filter(a => a.status !== 'online').length);

  function statusColor(s?: string): string {
    return s==='online' ? '#00ff88' : s==='degraded' ? '#ffaa00' : s==='offline' ? '#ff3355' : '#607070';
  }
  function timeSince(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const m = Math.floor(diff / 60000);
    return m < 1 ? 'just now' : m < 60 ? `${m}m ago` : `${Math.floor(m/60)}h ago`;
  }

  const headers = ['STATUS','HOSTNAME','TENANT','PLATFORM','COLLECTORS','VERSION','LAST SEEN'];
</script>

<div class="fm-page">
  <div class="fm-header">
    <div>
      <h1 class="fm-title">⬡ FLEET MANAGEMENT</h1>
      <p class="fm-sub">Multi-tenant agent status &amp; telemetry</p>
    </div>
    <div class="fm-controls">
      <input type="text" placeholder="Search by host or tenant…" bind:value={search} class="fm-search" />
      <button class="fm-refresh" onclick={fetchAgents}>↻ REFRESH</button>
    </div>
  </div>

  <div class="fm-stats">
    <div class="fm-stat fm-stat--teal">
      <div class="fm-stat-val">{agents.length}</div>
      <div class="fm-stat-label">TOTAL AGENTS</div>
    </div>
    <div class="fm-stat fm-stat--green">
      <div class="fm-stat-val">{online}</div>
      <div class="fm-stat-label">ONLINE</div>
    </div>
    <div class="fm-stat fm-stat--red">
      <div class="fm-stat-val">{offline}</div>
      <div class="fm-stat-label">OFFLINE / DEGRADED</div>
    </div>
  </div>

  <div class="fm-table-wrap">
    <table class="fm-table">
      <thead>
        <tr>{#each headers as h}<th>{h}</th>{/each}</tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="7" class="fm-cell-center">Loading agents…</td></tr>
        {:else if filtered.length === 0}
          <tr><td colspan="7" class="fm-cell-center">No agents registered. Deploy via the Onboarding wizard.</td></tr>
        {:else}
          {#each filtered as agent (agent.id)}
            <tr class="fm-row">
              <td>
                <span class="fm-dot" style="background:{statusColor(agent.status)}"></span>
                <span class="fm-status-text" style="color:{statusColor(agent.status)}">{(agent.status??'unknown').toUpperCase()}</span>
              </td>
              <td class="fm-hostname">
                {agent.hostname}
                <div class="fm-id">{agent.id}</div>
              </td>
              <td class="fm-tenant">{agent.tenant_id ?? 'GLOBAL'}</td>
              <td>
                <span class="fm-badge">{agent.os || 'unknown'}</span>
                <span class="fm-badge">{agent.arch || 'unknown'}</span>
              </td>
              <td>
                <div class="fm-tags">
                  {#each (agent.collectors || []) as c}
                    <span class="fm-tag">{c}</span>
                  {/each}
                </div>
              </td>
              <td class="fm-muted">v{agent.version}</td>
              <td class="fm-muted">{timeSince(agent.last_seen)}</td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>

<style>
  .fm-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .fm-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:20px; }
  .fm-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#00ffe7; }
  .fm-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .fm-controls { display:flex; gap:10px; align-items:center; }
  .fm-search { background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:8px 12px; border-radius:4px; font-size:12px; width:220px; font-family:inherit; outline:none; }
  .fm-refresh { background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:8px 14px; border-radius:4px; cursor:pointer; font-size:12px; letter-spacing:.1em; font-family:inherit; }
  .fm-stats { display:grid; grid-template-columns:repeat(3,1fr); gap:14px; margin-bottom:20px; }
  .fm-stat  { background:#0d1a1f; border:1px solid #1e3040; border-top-width:2px; padding:14px; border-radius:4px; }
  .fm-stat--teal  { border-top-color:#00ffe7; }
  .fm-stat--green { border-top-color:#00ff88; }
  .fm-stat--red   { border-top-color:#ff3355; }
  .fm-stat-val    { font-size:28px; font-weight:700; }
  .fm-stat--teal .fm-stat-val  { color:#00ffe7; }
  .fm-stat--green .fm-stat-val { color:#00ff88; }
  .fm-stat--red   .fm-stat-val { color:#ff3355; }
  .fm-stat-label  { font-size:10px; color:#607070; letter-spacing:.12em; margin-top:2px; }
  .fm-table-wrap { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; }
  .fm-table { width:100%; border-collapse:collapse; font-size:12px; }
  .fm-table thead tr { border-bottom:1px solid #1e3040; background:#0a1318; }
  .fm-table th { padding:10px 14px; text-align:left; color:#607070; letter-spacing:.12em; font-weight:400; white-space:nowrap; font-size:10px; }
  .fm-row { border-bottom:1px solid #0d1a1f; transition:background 80ms; }
  .fm-row:hover { background:#111f28; }
  .fm-row td { padding:10px 14px; }
  .fm-dot { display:inline-block; width:8px; height:8px; border-radius:50%; margin-right:7px; }
  .fm-status-text { font-size:10px; letter-spacing:.1em; }
  .fm-hostname { color:#c8d8d8; font-weight:600; }
  .fm-id      { font-size:9px; color:#607070; margin-top:2px; }
  .fm-tenant  { color:#00ffe7; }
  .fm-muted   { color:#607070; white-space:nowrap; }
  .fm-badge   { background:#1e3040; padding:2px 7px; border-radius:3px; font-size:10px; margin-right:4px; }
  .fm-tags    { display:flex; gap:4px; flex-wrap:wrap; }
  .fm-tag     { background:#0a1318; border:1px solid #1e3040; padding:2px 7px; border-radius:12px; font-size:10px; color:#00ffe7; }
  .fm-cell-center { padding:28px; text-align:center; color:#607070; }
</style>
