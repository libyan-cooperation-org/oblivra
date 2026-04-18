<!-- OBLIVRA Web — ThreatIntelDashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface Indicator { type:string; value:string; source:string; severity:'low'|'medium'|'high'|'critical'; description:string; campaign_id?:string; expires_at:string; }
  interface Campaign  { id:string; name:string; actor?:string; ttps?:string[]; description?:string; }
  type IOCStats = Record<string,number>;
  type Tab = 'indicators'|'campaigns'|'stats';

  const SEV: Record<string,{color:string;bg:string}> = {
    critical:{color:'#ff3355',bg:'#2a0d15'}, high:{color:'#ff6600',bg:'#2a1500'},
    medium:{color:'#ffaa00',bg:'#2a2000'},   low:{color:'#00ff88',bg:'#002a1a'},
  };
  const sev = (s:string) => SEV[s]??{color:'#607070',bg:'#0d1a1f'};
  const IOC_ICONS: Record<string,string> = {'ipv4-addr':'🌐','ipv6-addr':'🌐','domain-name':'🔗','md5':'#','sha256':'#','url':'↗'};

  let tab        = $state<Tab>('indicators');
  let indicators = $state<Indicator[]>([]);
  let campaigns  = $state<Campaign[]>([]);
  let stats      = $state<IOCStats>({});
  let loading    = $state(true);
  let search     = $state('');
  let typeFilter = $state('all');
  let sevFilter  = $state('all');
  let queryVal   = $state('');
  let queryResult = $state<Indicator|null|'none'>(null);

  onMount(async () => {
    try { const r = await request<{indicators:Indicator[]}>('/threatintel/indicators?limit=500'); indicators = r.indicators??[]; } catch {}
    try { const r = await request<{campaigns:Campaign[]}>('/threatintel/campaigns'); campaigns = r.campaigns??[]; } catch {}
    try { const r = await request<{stats:IOCStats}>('/threatintel/stats'); stats = r.stats??{}; } catch {}
    loading = false;
  });

  const types = $derived(['all', ...new Set(indicators.map(i => i.type))]);
  const filtered = $derived(indicators.filter(i => {
    const ms = !search || i.value.includes(search) || i.source.includes(search);
    const mt = typeFilter==='all' || i.type===typeFilter;
    const ms2 = sevFilter==='all' || i.severity===sevFilter;
    return ms && mt && ms2;
  }));
  const totalIOCs = $derived(Object.values(stats).reduce((a,b)=>a+b,0));

  async function lookupIOC() {
    if (!queryVal.trim()) return;
    try {
      const r = await request<{match:boolean;indicator?:Indicator}>(`/threatintel/lookup?value=${encodeURIComponent(queryVal)}`);
      queryResult = r.match && r.indicator ? r.indicator : 'none';
    } catch { queryResult = 'none'; }
  }
</script>

<div class="ti-page">
  <div class="ti-header">
    <h1 class="ti-title">⬡ THREAT INTELLIGENCE</h1>
    <p class="ti-sub">IOC browser · Campaign correlation · STIX/TAXII feed status</p>
  </div>

  <div class="ti-stats">
    <div class="ti-stat" style="border-top-color:#ff6600">
      <div class="ti-stat-val" style="color:#ff6600">{totalIOCs.toLocaleString()}</div>
      <div class="ti-stat-label">TOTAL IOCs</div>
    </div>
    {#each Object.entries(stats).slice(0,4) as [type, count]}
      <div class="ti-stat"><div class="ti-stat-val">{count}</div><div class="ti-stat-label">{IOC_ICONS[type]??''} {type}</div></div>
    {/each}
  </div>

  <div class="ti-lookup">
    <div>
      <div class="ti-lookup-label">INSTANT IOC LOOKUP</div>
      <div class="ti-lookup-row">
        <input type="text" placeholder="Enter IP, domain, hash, URL…" bind:value={queryVal} onkeydown={(e)=>e.key==='Enter'&&lookupIOC()} class="ti-lookup-input" />
        <button class="ti-lookup-btn" onclick={lookupIOC}>LOOKUP</button>
      </div>
    </div>
    {#if queryResult !== null}
      {@const r = queryResult}
      <div class="ti-lookup-result" style="{r==='none' ? '' : `background:${sev((r as Indicator).severity).bg}; border-color:${sev((r as Indicator).severity).color}`}">
        {#if r === 'none'}
          <div class="ti-muted">○ No match — indicator not found</div>
        {:else}
          <div class="ti-match-sev" style="color:{sev((r as Indicator).severity).color}">● {(r as Indicator).severity.toUpperCase()} MATCH</div>
          <div class="ti-match-val">{(r as Indicator).type}: {(r as Indicator).value}</div>
          <div class="ti-muted">{(r as Indicator).source} — {(r as Indicator).description}</div>
        {/if}
      </div>
    {/if}
  </div>

  <div class="ti-tabs">
    {#each (['indicators','campaigns','stats'] as Tab[]) as t}
      <button class="ti-tab {tab===t ? 'ti-tab--active' : ''}" onclick={() => tab=t}>{t.toUpperCase()}</button>
    {/each}
  </div>

  {#if tab === 'indicators'}
    <div class="ti-filters">
      <input type="text" placeholder="Filter by value or source…" bind:value={search} class="ti-filter-input" />
      <select bind:value={typeFilter} class="ti-select">
        {#each types as t}<option value={t}>{t==='all' ? 'All Types' : t}</option>{/each}
      </select>
      <select bind:value={sevFilter} class="ti-select">
        {#each ['all','critical','high','medium','low'] as s}<option value={s}>{s==='all' ? 'All Severities' : s}</option>{/each}
      </select>
      <span class="ti-count">{filtered.length} indicators</span>
    </div>
    <div class="ti-table-wrap">
      <table class="ti-table">
        <thead><tr>{#each ['SEV','TYPE','INDICATOR','SOURCE','DESCRIPTION','EXPIRES'] as h}<th>{h}</th>{/each}</tr></thead>
        <tbody>
          {#if loading}
            <tr><td colspan="6" class="ti-center">Loading indicators…</td></tr>
          {:else if filtered.length === 0}
            <tr><td colspan="6" class="ti-center">No IOCs loaded.</td></tr>
          {:else}
            {#each filtered.slice(0,500) as ind}
              {@const s = sev(ind.severity)}
              <tr class="ti-row">
                <td><span style="color:{s.color}; font-size:10px; font-weight:700">{ind.severity.toUpperCase()}</span></td>
                <td class="ti-muted">{IOC_ICONS[ind.type]??''} {ind.type}</td>
                <td style="color:{s.color}; font-size:11px">{ind.value}</td>
                <td class="ti-muted">{ind.source}</td>
                <td class="ti-muted ti-truncate" title={ind.description}>{ind.description}</td>
                <td class="ti-muted">{ind.expires_at ? new Date(ind.expires_at).toLocaleDateString() : '—'}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

  {:else if tab === 'campaigns'}
    <div class="ti-campaigns">
      {#each campaigns as c (c.id)}
        <div class="ti-campaign-card">
          <div class="ti-campaign-top">
            <div>
              <div class="ti-campaign-name">{c.name}</div>
              <div class="ti-muted">ID: {c.id}{c.actor ? ` · Actor: ${c.actor}` : ''}</div>
            </div>
            {#if c.ttps?.length}
              <div class="ti-ttps">{#each c.ttps.slice(0,6) as t}<span class="ti-ttp">{t}</span>{/each}</div>
            {/if}
          </div>
          {#if c.description}<div class="ti-campaign-desc">{c.description}</div>{/if}
        </div>
      {:else}
        <div class="ti-muted">No campaigns registered.</div>
      {/each}
    </div>

  {:else}
    <div class="ti-stats-grid">
      {#each Object.entries(stats) as [type, count]}
        <div class="ti-stat-card">
          <div class="ti-stat-big">{count.toLocaleString()}</div>
          <div class="ti-stat-type">{IOC_ICONS[type]??''} {type}</div>
          <div class="ti-stat-bar-bg"><div class="ti-stat-bar" style="width:{Math.min(100,Math.round(count/totalIOCs*100))}%"></div></div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .ti-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .ti-header { margin-bottom:20px; }
  .ti-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#ff6600; }
  .ti-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .ti-stats  { display:grid; grid-template-columns:repeat(5,1fr); gap:14px; margin-bottom:20px; }
  .ti-stat   { background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:12px; border-radius:4px; }
  .ti-stat-val   { font-size:22px; font-weight:700; color:#c8d8d8; }
  .ti-stat-label { font-size:10px; color:#607070; letter-spacing:.1em; margin-top:2px; }
  .ti-lookup { display:flex; gap:12px; margin-bottom:20px; background:#0d1a1f; border:1px solid #1e3040; padding:14px; border-radius:6px; align-items:flex-start; }
  .ti-lookup-label { font-size:10px; color:#607070; letter-spacing:.1em; margin-bottom:6px; }
  .ti-lookup-row   { display:flex; gap:8px; }
  .ti-lookup-input { flex:1; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:6px 10px; border-radius:3px; font-size:12px; font-family:inherit; outline:none; min-width:280px; }
  .ti-lookup-btn   { background:#ff6600; color:#080f12; border:none; padding:6px 18px; border-radius:3px; cursor:pointer; font-weight:700; font-size:12px; font-family:inherit; }
  .ti-lookup-result { padding:10px 14px; border-radius:4px; min-width:200px; background:#0a1318; border:1px solid #1e3040; }
  .ti-match-sev    { font-size:10px; font-weight:700; letter-spacing:.1em; margin-bottom:4px; }
  .ti-match-val    { font-size:12px; color:#c8d8d8; margin-bottom:2px; }
  .ti-tabs { display:flex; border-bottom:1px solid #1e3040; margin-bottom:16px; }
  .ti-tab  { padding:8px 18px; cursor:pointer; font-size:11px; letter-spacing:.12em; border:none; border-bottom:2px solid transparent; background:none; color:#607070; font-family:inherit; }
  .ti-tab--active { border-bottom-color:#ff6600; color:#ff6600; }
  .ti-filters { display:flex; gap:10px; margin-bottom:12px; align-items:center; }
  .ti-filter-input { background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:6px 10px; border-radius:3px; font-size:12px; flex:1; font-family:inherit; outline:none; }
  .ti-select { background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:6px; border-radius:3px; font-size:12px; font-family:inherit; outline:none; }
  .ti-count  { color:#607070; font-size:11px; white-space:nowrap; }
  .ti-table-wrap { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; }
  .ti-table  { width:100%; border-collapse:collapse; font-size:11px; }
  .ti-table thead tr { border-bottom:1px solid #1e3040; background:#0a1318; }
  .ti-table th { padding:8px 13px; text-align:left; color:#607070; letter-spacing:.1em; font-weight:400; font-size:10px; }
  .ti-row { border-bottom:1px solid #0a1318; transition:background 80ms; }
  .ti-row:hover { background:#111f28; }
  .ti-row td    { padding:8px 13px; }
  .ti-muted     { color:#607070; }
  .ti-truncate  { max-width:200px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .ti-center    { padding:28px; text-align:center; color:#607070; }
  .ti-campaigns { display:flex; flex-direction:column; gap:14px; }
  .ti-campaign-card { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:16px; }
  .ti-campaign-top  { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:8px; }
  .ti-campaign-name { color:#ff6600; font-size:13px; letter-spacing:.08em; margin-bottom:3px; }
  .ti-campaign-desc { color:#607070; font-size:11px; line-height:1.5; }
  .ti-ttps { display:flex; flex-wrap:wrap; gap:4px; max-width:280px; justify-content:flex-end; }
  .ti-ttp  { background:#111f28; border:1px solid #1e3040; color:#607070; padding:2px 6px; border-radius:2px; font-size:10px; }
  .ti-stats-grid { display:grid; grid-template-columns:repeat(auto-fill,minmax(180px,1fr)); gap:14px; }
  .ti-stat-card { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:16px; }
  .ti-stat-big  { font-size:24px; font-weight:700; color:#ff6600; }
  .ti-stat-type { font-size:11px; color:#c8d8d8; margin-top:3px; }
  .ti-stat-bar-bg { height:4px; background:#1e3040; border-radius:2px; margin-top:10px; }
  .ti-stat-bar    { height:100%; border-radius:2px; background:#ff6600; }
</style>
