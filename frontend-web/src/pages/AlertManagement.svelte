<!-- OBLIVRA Web — AlertManagement (Svelte 5) -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { request } from '../services/api';
  import { isDesktop } from '../context';

  interface Alert { id:number; tenant_id:string; host_id:string; timestamp:string; event_type:string; source_ip:string; user:string; raw_log:string; status?:Status; }
  type Status = 'new'|'investigating'|'acknowledged'|'closed';

  const SEV: Record<string,{label:string;color:string;bg:string}> = {
    security_alert:   {label:'CRITICAL',color:'#ff3355',bg:'#2a0d15'},
    failed_login:     {label:'HIGH',    color:'#ff6600',bg:'#2a1500'},
    sudo_exec:        {label:'MEDIUM',  color:'#ffaa00',bg:'#2a2000'},
    successful_login: {label:'INFO',    color:'#00ff88',bg:'#002a1a'},
  };
  function sev(t:string){return SEV[t]??{label:'LOW',color:'#607070',bg:'#0d1a1f'};}
  function fmt(iso:string){const d=new Date(iso);const p=(n:number)=>String(n).padStart(2,'0');return `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;}

  let alerts     = $state<Alert[]>([]);
  let loading    = $state(true);
  let localSt    = $state<Record<number,Status>>({});
  let filter     = $state<string>('all');
  let selected   = $state<Alert|null>(null);
  let liveCount  = $state(0);
  let ws: WebSocket|null = null;

  async function fetchAlerts() {
    loading = true;
    try {
      const res = await request<{active_alerts:number;alerts:Alert[]}>('/alerts');
      alerts = (res.alerts ?? []).map(a => ({...a, status:'new' as Status}));
    } catch { alerts = []; }
    loading = false;
  }

  onMount(() => {
    fetchAlerts();
    const wsBase = isDesktop() ? 'ws://localhost:8080' : window.location.origin.replace('http','ws');
    const token  = localStorage.getItem('oblivra_token') ?? '';
    ws = new WebSocket(`${wsBase}/api/v1/events?token=${token}`);
    ws.onmessage = (evt) => {
      try {
        const ev = JSON.parse(evt.data);
        if (ev.topic?.includes('alert')||ev.topic?.includes('security')) {
          liveCount++;
          if (liveCount % 10 === 0) fetchAlerts();
        }
      } catch {}
    };
  });
  onDestroy(() => ws?.close());

  const displayed = $derived.by(() => {
    const base = alerts.map(a=>({...a, status:localSt[a.id]??a.status??'new' as Status}));
    return filter==='all' ? base : base.filter(a=>a.status===filter);
  });

  function setStatus(id:number, s:Status) {
    localSt = {...localSt, [id]:s};
    if (selected?.id===id) selected = selected ? {...selected, status:s} : null;
  }

  const statusColor: Record<string,string> = {new:'#ff3355',investigating:'#ffaa00',acknowledged:'#00ffe7',closed:'#607070'};
  const filters = ['all','new','investigating','acknowledged','closed'] as const;
  const actions: [string, Status, string][] = [
    ['INVESTIGATE','investigating','#ffaa00'],
    ['ACKNOWLEDGE','acknowledged','#00ffe7'],
    ['CLOSE','closed','#607070'],
    ['REOPEN','new','#ff6600'],
  ];
</script>

<div class="am-wrap">
  <!-- Left panel -->
  <div class="am-list-panel">
    <div class="am-list-header">
      <div>
        <h1 class="am-title">⚠ ALERT MANAGEMENT</h1>
        <p class="am-sub">Live event feed · {alerts.length} alerts{liveCount > 0 ? ` · +${liveCount} live` : ''}</p>
      </div>
      <button class="am-refresh-btn" onclick={fetchAlerts}>↻ REFRESH</button>
    </div>
    <div class="am-filters">
      {#each filters as f}
        <button class="am-filter-btn {filter===f ? 'am-filter-btn--active' : ''}" onclick={() => filter = f}>{f.toUpperCase()}</button>
      {/each}
    </div>

    <div class="am-rows" role="list" aria-label="Alert list">
      {#if loading}
        <div class="am-loading">Loading alerts…</div>
      {:else if displayed.length === 0}
        <div class="am-loading">No alerts match the current filter.</div>
      {:else}
        {#each displayed as a (a.id)}
          {@const s = sev(a.event_type)}
          {@const status = localSt[a.id] ?? a.status ?? 'new'}
          <div
            class="am-row"
            class:am-row--selected={selected?.id === a.id}
            style="border-left-color:{selected?.id===a.id ? '#00ffe7' : s.color}"
            role="listitem"
            tabindex="0"
            onclick={() => selected = a}
            onkeydown={(e) => e.key==='Enter' && (selected = a)}
            aria-selected={selected?.id === a.id}
          >
            <div class="am-row-top">
              <span class="am-sev-badge" style="color:{s.color}; background:{s.bg}">{s.label}</span>
              <span class="am-status-dot" style="color:{statusColor[status]??'#607070'}">● {status.toUpperCase()}</span>
            </div>
            <div class="am-row-event">{a.event_type.replace(/_/g,' ')}</div>
            <div class="am-row-meta">
              <span>{a.host_id || 'unknown'}</span>
              <span>{fmt(a.timestamp)}</span>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  </div>

  <!-- Right panel -->
  <div class="am-detail-panel">
    {#if !selected}
      <div class="am-detail-empty">Select an alert<br/>to view details and take action</div>
    {:else}
      {@const s = sev(selected.event_type)}
      {@const status = localSt[selected.id] ?? selected.status ?? 'new'}
      <div class="am-detail-header" style="background:{s.bg}">
        <div class="am-detail-sev" style="color:{s.color}">{s.label}</div>
        <div class="am-detail-event">{selected.event_type.replace(/_/g,' ')}</div>
        <div class="am-detail-ts">{fmt(selected.timestamp)}</div>
      </div>
      <div class="am-detail-fields">
        {#each [['Tenant',selected.tenant_id||'GLOBAL'],['Host',selected.host_id||'—'],['Source IP',selected.source_ip||'—'],['User',selected.user||'—'],['Status',status.toUpperCase()]] as [k,v]}
          <div class="am-field"><div class="am-field-key">{k}</div><div class="am-field-val">{v}</div></div>
        {/each}
        <div class="am-field">
          <div class="am-field-key">RAW LOG</div>
          <pre class="am-raw-log">{selected.raw_log||'(no raw data)'}</pre>
        </div>
      </div>
      <div class="am-action-grid">
        {#each actions as [label, st, color]}
          <button
            class="am-action-btn"
            style="border-color:{color}; color:{status===st ? '#1e3040' : color}; opacity:{status===st ? 0.4 : 1};"
            disabled={status === st}
            onclick={() => setStatus(selected!.id, st)}
          >{label}</button>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .am-wrap { display:flex; height:100vh; background:#080f12; color:#c8d8d8; font-family:var(--font-mono); overflow:hidden; }
  .am-list-panel { flex:1; display:flex; flex-direction:column; border-right:1px solid #1e3040; overflow:hidden; }
  .am-list-header { padding:22px 22px 14px; border-bottom:1px solid #1e3040; flex-shrink:0; display:flex; justify-content:space-between; align-items:flex-start; }
  .am-title { font-size:18px; letter-spacing:.14em; margin:0; color:#ff3355; }
  .am-sub   { margin:4px 0 0; font-size:11px; color:#607070; }
  .am-refresh-btn { background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:6px 14px; border-radius:4px; cursor:pointer; font-size:11px; letter-spacing:.1em; font-family:inherit; }
  .am-filters { display:flex; gap:0; border-bottom:1px solid #1e3040; flex-shrink:0; }
  .am-filter-btn { flex:1; padding:8px 4px; border:none; cursor:pointer; font-size:10px; letter-spacing:.1em; background:#0a1318; color:#607070; font-family:inherit; transition:all 80ms; }
  .am-filter-btn--active { background:#1e3040; color:#00ffe7; }
  .am-rows { flex:1; overflow-y:auto; }
  .am-loading { padding:28px; text-align:center; color:#607070; font-size:12px; }
  .am-row { padding:13px 18px; border-bottom:1px solid #0a1318; cursor:pointer; border-left:3px solid transparent; background:transparent; transition:background 80ms; }
  .am-row--selected { background:#111f28; }
  .am-row:not(.am-row--selected):hover { background:rgba(255,255,255,0.02); }
  .am-row-top  { display:flex; justify-content:space-between; align-items:center; margin-bottom:4px; }
  .am-sev-badge { font-size:10px; font-weight:700; letter-spacing:.12em; padding:1px 6px; border-radius:2px; }
  .am-status-dot { font-size:10px; }
  .am-row-event { font-size:12px; color:#c8d8d8; }
  .am-row-meta  { display:flex; gap:14px; margin-top:3px; font-size:10px; color:#607070; }
  .am-detail-panel { width:400px; flex-shrink:0; display:flex; flex-direction:column; overflow:hidden; }
  .am-detail-empty { flex:1; display:flex; align-items:center; justify-content:center; color:#607070; font-size:12px; text-align:center; padding:28px; line-height:1.8; }
  .am-detail-header { padding:22px; border-bottom:1px solid #1e3040; flex-shrink:0; }
  .am-detail-sev   { font-size:11px; font-weight:700; letter-spacing:.14em; margin-bottom:7px; }
  .am-detail-event { font-size:15px; color:#c8d8d8; margin-bottom:3px; }
  .am-detail-ts    { font-size:11px; color:#607070; }
  .am-detail-fields { flex:1; overflow-y:auto; padding:22px; display:flex; flex-direction:column; gap:14px; }
  .am-field {}
  .am-field-key { font-size:10px; color:#607070; letter-spacing:.12em; margin-bottom:3px; }
  .am-field-val { font-size:13px; color:#c8d8d8; }
  .am-raw-log   { font-size:10px; color:#607070; background:#0a1318; padding:10px; border:1px solid #1e3040; border-radius:3px; overflow-x:auto; white-space:pre-wrap; word-break:break-all; margin:0; }
  .am-action-grid { padding:14px 22px; border-top:1px solid #1e3040; display:grid; grid-template-columns:1fr 1fr; gap:8px; flex-shrink:0; }
  .am-action-btn { border:1px solid; background:none; padding:8px; border-radius:3px; cursor:pointer; font-size:11px; letter-spacing:.1em; font-family:inherit; transition:all 100ms; }
  .am-action-btn:disabled { cursor:default; }
</style>
