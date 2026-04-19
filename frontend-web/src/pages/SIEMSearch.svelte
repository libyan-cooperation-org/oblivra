<!-- OBLIVRA Web — SIEMSearch (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface HostEvent { id:number; tenant_id:string; host_id:string; timestamp:string; event_type:string; source_ip:string; user:string; raw_log:string; }
  const SMAP: Record<string,{label:string;color:string}> = {
    failed_login:     {label:'CRIT',color:'#ff3355'},
    security_alert:   {label:'HIGH',color:'#ff6600'},
    sudo_exec:        {label:'MED', color:'#ffaa00'},
    successful_login: {label:'INFO',color:'#00ff88'},
  };
  function getSev(t:string){return SMAP[t]??{label:'LOW',color:'#607070'};}

  let query     = $state('');
  let limit     = $state(100);
  let lastQuery = $state('');
  let results   = $state<HostEvent[]>([]);
  let loading   = $state(false);
  let expanded  = $state<Set<number>>(new Set());

  async function runSearch() {
    if (!query.trim()) return;
    lastQuery = query;
    loading = true;
    expanded = new Set();
    try {
      const p = new URLSearchParams({q:query, limit:String(limit)});
      const r = await request<{events:HostEvent[];total:number}>(`/siem/search?${p}`);
      results = r.events ?? [];
    } catch { results = []; }
    loading = false;
  }

  function handleKey(e: KeyboardEvent){ if(e.key==='Enter') runSearch(); }
  function toggleRow(id:number){ const s = new Set(expanded); s.has(id)?s.delete(id):s.add(id); expanded=s; }
  function appendFilter(field:string, val:string){
    const cur = query.trim();
    query = cur ? `${cur} AND ${field}:"${val}"` : `${field}:"${val}"`;
    runSearch();
  }

  // Histogram — 40 bins
  const histogram = $derived.by(() => {
    if (!results.length) return [];
    const times = results.map(e => new Date(e.timestamp).getTime());
    const minT = Math.min(...times), maxT = Math.max(...times);
    const bins = Array.from({length:40},()=>0);
    const span = maxT - minT || 1;
    results.forEach(e => { let i=Math.floor(((new Date(e.timestamp).getTime()-minT)/span)*40); if(i>=40)i=39; bins[i]++; });
    const mx = Math.max(...bins,1);
    return bins.map((count,i)=>({ count, pct:(count/mx)*100, label:new Date(minT+(i/40)*span).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'}) }));
  });

  // Interesting fields
  const fields = $derived.by(() => {
    const fc: Record<string,Record<string,number>> = {};
    results.forEach(e => {
      (['host_id','event_type','source_ip','user'] as const).forEach(k => {
        const v = String((e as any)[k]||'');
        if(v&&v!=='undefined'){ if(!fc[k]) fc[k]={}; fc[k][v]=(fc[k][v]||0)+1; }
      });
    });
    return Object.entries(fc).map(([field,vm])=>({field, vals:Object.entries(vm).sort((a,b)=>b[1]-a[1]).slice(0,5)}));
  });
</script>

<div class="ss-page">
  <!-- Search bar -->
  <div class="ss-topbar">
    <div class="ss-title-row">
      <div>
        <h1 class="ss-title">⬡ OBLIVRA SEARCH</h1>
        <p class="ss-sub">High-performance Splunk-style event analysis and field extraction</p>
      </div>
      {#if lastQuery}<div class="ss-result-count"><span class="ss-count-num">{results.length}</span> Events matched</div>{/if}
    </div>
    <div class="ss-bar">
      <span class="ss-prompt">&gt;</span>
      <input
        class="ss-input"
        type="text"
        bind:value={query}
        onkeydown={handleKey}
        placeholder='e.g. event_type:failed_login OR source_ip:192.168.*'
      />
      <div class="ss-divider"></div>
      <select class="ss-limit" bind:value={limit}>
        {#each [50,100,250,500,1000] as n}<option value={n}>{n} results</option>{/each}
      </select>
      <button class="ss-search-btn" onclick={runSearch}>{loading ? 'SEARCHING…' : 'SEARCH'}</button>
    </div>
  </div>

  <!-- Body -->
  <div class="ss-body">
    <!-- Sidebar -->
    <div class="ss-sidebar">
      <div class="ss-sidebar-title">INTERESTING FIELDS</div>
      {#if fields.length === 0}
        <div class="ss-sidebar-empty">No events to extract.</div>
      {:else}
        {#each fields as {field, vals}}
          <div class="ss-field-group">
            <div class="ss-field-name">{field}</div>
            {#each vals as [val, count]}
              <div 
                role="button"
                tabindex="0"
                class="ss-field-val" 
                onclick={() => appendFilter(field, val as string)} 
                onkeydown={(e) => e.key === 'Enter' && appendFilter(field, val as string)}
                title='Filter: {field}="{val}"'
              >
                <span class="ss-field-text">{val}</span>
                <span class="ss-field-count">{count}</span>
              </div>
            {/each}
          </div>
        {/each}
      {/if}
    </div>

    <!-- Results -->
    <div class="ss-results-area">
      <!-- Histogram -->
      <div class="ss-histogram">
        {#if results.length === 0 && !loading}
          <div class="ss-histogram-empty">No timeline data</div>
        {:else}
          {#each histogram as bin}
            <div
              class="ss-bin"
              style="height:{bin.pct}%; background:{bin.count > 0 ? '#00ffe7' : 'transparent'}; min-height:{bin.count > 0 ? '4px' : '0'}"
              title="{bin.label} | Count: {bin.count}"
            ></div>
          {/each}
        {/if}
      </div>

      <!-- Table -->
      <div class="ss-table-wrap">
        {#if loading}
          <div class="ss-table-overlay"><span class="ss-searching">SEARCHING…</span></div>
        {/if}
        <table class="ss-table">
          <colgroup>
            <col style="width:2rem"/>
            <col style="width:11rem"/>
            <col style="width:8rem"/>
            <col style="width:8rem"/>
            <col/>
          </colgroup>
          <thead>
            <tr>
              <th></th>
              <th>TIME</th><th>HOST</th><th>EVENT TYPE</th><th>RAW EVENT</th>
            </tr>
          </thead>
          <tbody>
            {#if !loading && results.length === 0}
              <tr><td colspan="5" class="ss-empty-row">
                <div class="ss-empty-icon">⬡</div>
                {lastQuery ? '0 EVENTS MATCHED THIS QUERY' : 'READY FOR SEARCH'}
              </td></tr>
            {:else}
              {#each results as evt (evt.id)}
                {@const isExpanded = expanded.has(evt.id)}
                {@const sev = getSev(evt.event_type)}
                <tr class="ss-row" onclick={() => toggleRow(evt.id)}>
                  <td class="ss-expand-col">{isExpanded ? '▼' : '▶'}</td>
                  <td class="ss-ts">{new Date(evt.timestamp).toISOString().replace('T',' ').slice(0,19)}</td>
                  <td class="ss-host">{evt.host_id}</td>
                  <td class="ss-evtype" style="color:{sev.color}">{evt.event_type}</td>
                  <td class="ss-raw">{evt.raw_log}</td>
                </tr>
                {#if isExpanded}
                  {@const parsed = (() => { try { return JSON.parse(evt.raw_log); } catch { return null; } })()}
                  <tr class="ss-detail-row">
                    <td colspan="5">
                      <div class="ss-detail">
                        <div class="ss-detail-col">
                          <div class="ss-detail-heading">EVENT DETAIL</div>
                          <table class="ss-detail-table">
                            <tbody>
                              {#each [['ID',evt.id],['Tenant',evt.tenant_id||'GLOBAL'],['Source IP',evt.source_ip||'—'],['User',evt.user||'—']] as [k,v]}
                                <tr><td class="ss-dk">{k}</td><td class="ss-dv">{v}</td></tr>
                              {/each}
                            </tbody>
                          </table>
                        </div>
                        <div class="ss-detail-col ss-detail-col--wide">
                          <div class="ss-detail-heading">PARSED RAW LOG</div>
                          {#if parsed}
                            <table class="ss-detail-table">
                              <tbody>
                                {#each Object.entries(parsed) as [k,v]}
                                  <tr><td class="ss-pk">{k}</td><td class="ss-pv">{typeof v==='object' ? JSON.stringify(v) : String(v)}</td></tr>
                                {/each}
                              </tbody>
                            </table>
                          {:else}
                            <pre class="ss-raw-block">{evt.raw_log}</pre>
                          {/if}
                        </div>
                      </div>
                    </td>
                  </tr>
                {/if}
              {/each}
            {/if}
          </tbody>
        </table>
      </div>
    </div>
  </div>
</div>

<style>
  .ss-page { padding:18px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; display:flex; flex-direction:column; gap:14px; }
  .ss-topbar {}
  .ss-title-row { display:flex; justify-content:space-between; align-items:center; margin-bottom:12px; }
  .ss-title { font-size:20px; letter-spacing:.14em; margin:0; color:#00ffe7; }
  .ss-sub   { margin:3px 0 0; font-size:11px; color:#607070; }
  .ss-result-count { font-size:12px; color:#607070; }
  .ss-count-num { font-size:18px; color:#00ffe7; font-weight:700; margin-right:6px; }
  .ss-bar { display:flex; align-items:center; gap:8px; background:#0d1a1f; padding:7px 10px; border-radius:5px; border:1px solid #1e3040; }
  .ss-prompt { color:#00ffe7; font-weight:700; font-size:14px; }
  .ss-input  { flex:1; background:transparent; border:none; color:#c8d8d8; padding:4px; font-size:13px; font-family:inherit; outline:none; }
  .ss-divider { width:1px; background:#1e3040; margin:0 6px; height:18px; }
  .ss-limit { background:transparent; border:none; color:#00ffe7; font-family:inherit; font-size:12px; outline:none; cursor:pointer; }
  .ss-search-btn { background:#00ffe7; color:#080f12; border:none; padding:5px 20px; border-radius:3px; font-weight:800; cursor:pointer; font-size:12px; letter-spacing:.1em; font-family:inherit; transition:opacity 150ms; }
  .ss-search-btn:hover { opacity:0.8; }
  .ss-body { display:flex; gap:18px; flex:1; overflow:hidden; }
  .ss-sidebar { width:240px; flex-shrink:0; overflow-y:auto; }
  .ss-sidebar-title { font-size:11px; color:#607070; font-weight:700; letter-spacing:.1em; margin-bottom:12px; border-bottom:1px solid #1e3040; padding-bottom:7px; }
  .ss-sidebar-empty { font-size:11px; color:#607070; font-style:italic; }
  .ss-field-group { margin-bottom:16px; }
  .ss-field-name  { font-size:11px; color:#00ffe7; margin-bottom:5px; font-weight:700; }
  .ss-field-val   { display:flex; justify-content:space-between; align-items:center; font-size:11px; padding:3px 5px; cursor:pointer; border-radius:3px; transition:background 80ms; }
  .ss-field-val:hover { background:#111f28; color:#fff; }
  .ss-field-text  { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; max-width:160px; }
  .ss-field-count { color:#607070; font-size:10px; background:#0d1a1f; padding:1px 5px; border-radius:8px; flex-shrink:0; }
  .ss-results-area { flex:1; display:flex; flex-direction:column; min-width:0; gap:14px; overflow:hidden; }
  .ss-histogram { height:80px; display:flex; align-items:flex-end; gap:2px; border-bottom:1px solid #1e3040; padding-bottom:7px; }
  .ss-histogram-empty { color:#607070; font-size:11px; align-self:center; margin:auto; }
  .ss-bin { flex:1; opacity:0.8; transition:opacity 100ms; cursor:pointer; }
  .ss-bin:hover { opacity:1; }
  .ss-table-wrap { flex:1; background:#0d1a1f; border:1px solid #1e3040; border-radius:5px; overflow:auto; position:relative; }
  .ss-table-overlay { position:absolute; inset:0; background:rgba(8,15,18,0.75); display:flex; align-items:center; justify-content:center; z-index:10; }
  .ss-searching { color:#00ffe7; font-weight:700; font-size:18px; letter-spacing:.2em; animation:pulse 1.4s ease-in-out infinite; }
  .ss-table { width:100%; border-collapse:collapse; font-size:11px; table-layout:fixed; }
  .ss-table thead tr { border-bottom:1px solid #1e3040; background:#0a1318; position:sticky; top:0; z-index:5; }
  .ss-table th { padding:8px 13px; text-align:left; color:#607070; font-weight:400; letter-spacing:.1em; font-size:10px; }
  .ss-row { border-bottom:1px solid #111f28; cursor:pointer; transition:background 60ms; }
  .ss-row:hover { background:#111f28; }
  .ss-expand-col { padding:8px; text-align:center; color:#00ffe7; font-size:10px; width:28px; }
  .ss-ts   { padding:8px 13px; color:#607070; white-space:nowrap; }
  .ss-host { padding:8px 13px; color:#c8d8d8; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .ss-evtype { padding:8px 13px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .ss-raw  { padding:8px 13px; color:#607070; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
  .ss-empty-row { padding:48px; text-align:center; color:#607070; }
  .ss-empty-icon { font-size:36px; opacity:0.2; margin-bottom:12px; }
  .ss-detail-row { background:#0a1318; border-bottom:1px solid #1e3040; }
  .ss-detail { display:flex; gap:28px; padding:18px; }
  .ss-detail-col { flex:1; }
  .ss-detail-col--wide { flex:2; }
  .ss-detail-heading { color:#00ffe7; font-size:10px; font-weight:700; letter-spacing:.1em; margin-bottom:8px; }
  .ss-detail-table   { width:100%; border-collapse:collapse; font-size:11px; }
  .ss-dk { padding:3px 0; color:#607070; width:110px; vertical-align:top; }
  .ss-dv { padding:3px 0; color:#c8d8d8; }
  .ss-pk { padding:2px 0; color:#ffaa00; width:30%; vertical-align:top; }
  .ss-pv { padding:2px 0; color:#00ffe7; word-break:break-word; }
  .ss-raw-block { font-size:10px; color:#607070; background:#080f12; padding:10px; border:1px solid #1e3040; border-radius:3px; overflow-x:auto; white-space:pre-wrap; word-break:break-all; margin:0; }
  @keyframes pulse { 0%,100%{opacity:0.5} 50%{opacity:1} }
</style>
