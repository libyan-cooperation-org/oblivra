<!-- OBLIVRA Web — MitreHeatmap (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface TechniqueCell { id:string; name:string; hits:number; }
  interface TacticRow     { id:string; name:string; techniques:TechniqueCell[]; }
  interface HeatmapData   { tactics:TacticRow[]; total_hits:number; last_updated:string; }

  const TACTIC_ORDER = ['TA0001','TA0002','TA0003','TA0004','TA0005','TA0006','TA0007','TA0008','TA0009','TA0011','TA0010','TA0040'];

  let data     = $state<HeatmapData|null>(null);
  let loading  = $state(true);
  let showZero = $state(false);
  let selected = $state<TechniqueCell|null>(null);

  async function fetchData() {
    loading = true;
    try {
      const r = await request<HeatmapData>('/mitre/heatmap');
      r.tactics = [...r.tactics].sort((a,b) => TACTIC_ORDER.indexOf(a.id) - TACTIC_ORDER.indexOf(b.id));
      data = r;
    } catch { data = null; }
    loading = false;
  }
  onMount(fetchData);

  const maxHits = $derived.by(() => {
    let m = 0;
    (data?.tactics??[]).forEach(t => t.techniques.forEach(tc => { if(tc.hits>m) m=tc.hits; }));
    return m;
  });

  function heatColor(hits:number, max:number){
    if(!hits) return {bg:'#0d1a1f',border:'#1e3040',text:'#607070'};
    const r = Math.min(hits/Math.max(max,1),1);
    if(r>.75) return {bg:'#2a0d15',border:'#ff3355',text:'#ff3355'};
    if(r>.5)  return {bg:'#2a1500',border:'#ff6600',text:'#ff6600'};
    if(r>.25) return {bg:'#2a2000',border:'#ffaa00',text:'#ffaa00'};
    return {bg:'#001a0a',border:'#00ff88',text:'#00ff88'};
  }
</script>

<div class="mh-page">
  <div class="mh-header">
    <div>
      <h1 class="mh-title">⬡ MITRE ATT&amp;CK HEATMAP</h1>
      <p class="mh-sub">Technique coverage · Alert frequency · Tactic grouping</p>
    </div>
    <div class="mh-controls">
      <button class="mh-toggle" class:mh-toggle--on={showZero} onclick={() => showZero = !showZero}>
        {showZero ? '○ ALL' : '● HIT ONLY'}
      </button>
      <button class="mh-refresh" onclick={fetchData}>↻ REFRESH</button>
    </div>
  </div>

  <div class="mh-legend">
    <span class="mh-legend-label">FREQUENCY:</span>
    {#each [{l:'None',bg:'#0d1a1f',b:'#1e3040',t:'#607070'},{l:'Low',bg:'#001a0a',b:'#00ff88',t:'#00ff88'},{l:'Med',bg:'#2a2000',b:'#ffaa00',t:'#ffaa00'},{l:'High',bg:'#2a1500',b:'#ff6600',t:'#ff6600'},{l:'Crit',bg:'#2a0d15',b:'#ff3355',t:'#ff3355'}] as item}
      <div class="mh-legend-item">
        <div class="mh-legend-swatch" style="background:{item.bg}; border-color:{item.b}"></div>
        <span style="color:{item.t}">{item.l}</span>
      </div>
    {/each}
    {#if data}
      <span class="mh-total">{data.total_hits.toLocaleString()} total hits · Last: {data.last_updated ? new Date(data.last_updated).toLocaleTimeString() : '—'}</span>
    {/if}
  </div>

  {#if loading}
    <div class="mh-loading">Loading ATT&amp;CK matrix…</div>
  {:else if data}
    <div class="mh-grid">
      {#each data.tactics as tactic (tactic.id)}
        <div class="mh-tactic">
          <div class="mh-tactic-header">
            <div class="mh-tactic-id">{tactic.id}</div>
            <div class="mh-tactic-name">{tactic.name}</div>
            <div class="mh-tactic-count">{tactic.techniques.filter(t=>t.hits>0).length}/{tactic.techniques.length} triggered</div>
          </div>
          <div class="mh-cells">
            {#each (showZero ? tactic.techniques : tactic.techniques) as tech (tech.id)}
              {@const c = heatColor(tech.hits, maxHits)}
              <div
                class="mh-cell"
                class:mh-cell--selected={selected?.id === tech.id}
                style="background:{c.bg}; border-color:{selected?.id===tech.id ? '#c8d8d8' : c.border}"
                title="{tech.id} — {tech.name}: {tech.hits} hits"
                onclick={() => selected = selected?.id===tech.id ? null : tech}
                role="button"
                tabindex="0"
                onkeydown={(e) => e.key==='Enter' && (selected = selected?.id===tech.id ? null : tech)}
              >
                <div class="mh-tech-id" style="color:{c.text}">{tech.id}</div>
                <div class="mh-tech-name" title={tech.name}>{tech.name}</div>
                {#if tech.hits > 0}<span class="mh-tech-hits" style="color:{c.text}">{tech.hits}</span>{/if}
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if selected}
    {@const c = heatColor(selected.hits, maxHits)}
    <div class="mh-detail" style="border-color:{c.border}">
      <div class="mh-detail-header">
        <div>
          <div class="mh-detail-id" style="color:{c.text}">{selected.id}</div>
          <div class="mh-detail-name">{selected.name}</div>
        </div>
        <button class="mh-detail-close" onclick={() => selected = null}>✕</button>
      </div>
      <div class="mh-detail-row">
        <span>Alert Hits</span>
        <span class="mh-detail-count" style="color:{c.text}">{selected.hits.toLocaleString()}</span>
      </div>
      <div class="mh-detail-bar-bg">
        <div class="mh-detail-bar" style="width:{Math.round(selected.hits/Math.max(maxHits,1)*100)}%; background:{c.border}"></div>
      </div>
      <div class="mh-detail-link">ATT&amp;CK Navigator ·
        <a href="https://attack.mitre.org/techniques/{selected.id}/" target="_blank" style="color:{c.text}">View on MITRE ↗</a>
      </div>
    </div>
  {/if}
</div>

<style>
  .mh-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .mh-header { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:20px; }
  .mh-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#ff3355; }
  .mh-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .mh-controls { display:flex; gap:10px; }
  .mh-toggle  { background:#0d1a1f; border:1px solid #1e3040; color:#607070; padding:5px 12px; border-radius:3px; cursor:pointer; font-size:11px; letter-spacing:.08em; font-family:inherit; }
  .mh-toggle--on { background:#1e3040; }
  .mh-refresh { background:#0d1a1f; border:1px solid #1e3040; color:#607070; padding:5px 12px; border-radius:3px; cursor:pointer; font-size:11px; font-family:inherit; }
  .mh-legend { display:flex; gap:14px; margin-bottom:20px; align-items:center; flex-wrap:wrap; }
  .mh-legend-label { font-size:10px; color:#607070; letter-spacing:.1em; }
  .mh-legend-item  { display:flex; align-items:center; gap:5px; font-size:10px; }
  .mh-legend-swatch { width:12px; height:12px; border-radius:2px; border:1px solid; }
  .mh-total { margin-left:auto; font-size:11px; color:#607070; }
  .mh-loading { color:#607070; font-size:12px; text-align:center; padding:40px; }
  .mh-grid { display:grid; grid-template-columns:repeat(6,1fr); gap:10px; }
  .mh-tactic { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; }
  .mh-tactic-header { padding:8px 10px; background:#0a1318; border-bottom:1px solid #1e3040; }
  .mh-tactic-id    { font-size:10px; color:#607070; letter-spacing:.1em; }
  .mh-tactic-name  { font-size:11px; color:#c8d8d8; font-weight:600; margin-top:1px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
  .mh-tactic-count { font-size:9px; color:#607070; margin-top:2px; }
  .mh-cells { padding:5px; display:flex; flex-direction:column; gap:4px; }
  .mh-cell  { background:#0d1a1f; border:1px solid; border-radius:3px; padding:5px 7px; cursor:pointer; transition:opacity 100ms; display:flex; justify-content:space-between; align-items:flex-start; }
  .mh-cell:hover { opacity:0.8; }
  .mh-cell--selected { border-color:#c8d8d8 !important; }
  .mh-tech-id   { font-size:10px; font-weight:700; letter-spacing:.04em; }
  .mh-tech-name { font-size:9px; color:#607070; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; max-width:90px; }
  .mh-tech-hits { font-size:10px; font-weight:700; flex-shrink:0; margin-left:4px; }
  .mh-detail {
    position:fixed; right:20px; bottom:20px; background:#0d1a1f; border:1px solid; border-radius:8px;
    padding:18px; width:270px; box-shadow:0 8px 32px rgba(0,0,0,0.6); z-index:100; font-family:var(--font-mono);
  }
  .mh-detail-header { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:14px; }
  .mh-detail-id     { font-size:11px; font-weight:700; letter-spacing:.1em; }
  .mh-detail-name   { font-size:13px; color:#c8d8d8; margin-top:3px; }
  .mh-detail-close  { background:none; border:none; color:#607070; cursor:pointer; font-size:15px; padding:0; }
  .mh-detail-row    { display:flex; justify-content:space-between; font-size:11px; margin-bottom:7px; }
  .mh-detail-count  { font-weight:700; font-size:16px; }
  .mh-detail-bar-bg { height:6px; background:#1e3040; border-radius:3px; margin-bottom:10px; }
  .mh-detail-bar    { height:100%; border-radius:3px; }
  .mh-detail-link   { font-size:10px; color:#607070; }
  .mh-detail-link a { text-decoration:none; }
</style>
