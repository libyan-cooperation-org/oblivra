<!-- OBLIVRA Web — FusionDashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface FusionCampaign { id:string; entities:string[]; alert_count:number; tactic_stages:string[]; stage_count:number; confidence:number; first_seen:string; last_seen:string; status:'active'|'closed'|'investigating'; kill_chain_progress:number; }
  interface KillChainStage  { tactic_id:string; tactic_name:string; hit_count:number; techniques:string[]; first_seen?:string; }

  const TACTIC_ORDER = ['Initial Access','Execution','Persistence','Privilege Escalation','Defense Evasion','Credential Access','Discovery','Lateral Movement','Collection','Command & Control','Exfiltration','Impact'];

  let campaigns  = $state<FusionCampaign[]>([]);
  let selected   = $state<FusionCampaign|null>(null);
  let killChain  = $state<KillChainStage[]>([]);
  let loading    = $state(true);
  let loadingKC  = $state(false);

  async function fetchCampaigns() {
    loading = true;
    try { const r = await request<{campaigns:FusionCampaign[]}>('/fusion/campaigns'); campaigns = r.campaigns ?? []; }
    catch { campaigns = []; }
    loading = false;
  }

  async function selectCampaign(c: FusionCampaign) {
    selected = selected?.id === c.id ? null : c;
    if (selected) {
      loadingKC = true;
      try { const r = await request<{stages:KillChainStage[]}>(`/fusion/campaigns/${c.id}/kill-chain`); killChain = r.stages ?? []; }
      catch { killChain = []; }
      loadingKC = false;
    }
  }

  onMount(fetchCampaigns);

  const activeCampaigns = $derived(campaigns.filter(c => c.status==='active').length);
  const highConf        = $derived(campaigns.filter(c => c.confidence>=0.7).length);
  const maxStages       = $derived(Math.max(0, ...campaigns.map(c=>c.stage_count)));

  function confColor(c:number){ return c>=0.8?'#ff3355':c>=0.6?'#ff6600':c>=0.4?'#ffaa00':'#00ff88'; }
  function statusColor(s:string){ return s==='active'?'#ff3355':s==='investigating'?'#ffaa00':'#607070'; }

  function tacticActive(campaign:FusionCampaign, tactic:string){
    return campaign.tactic_stages.some(s => s.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]));
  }
</script>

<div class="fd-page">
  <div class="fd-header">
    <div>
      <h1 class="fd-title">⬡ FUSION ENGINE</h1>
      <p class="fd-sub">Kill chain correlation · Campaign clustering · Probabilistic attack scoring</p>
    </div>
    <button class="fd-refresh" onclick={fetchCampaigns}>↻ REFRESH</button>
  </div>

  <div class="fd-stats">
    {#each [{label:'TOTAL CAMPAIGNS',val:campaigns.length,c:'#c8d8d8'},{label:'ACTIVE',val:activeCampaigns,c:'#ff3355'},{label:'HIGH CONFIDENCE',val:highConf,c:'#ff6600'},{label:'MAX STAGE COVERAGE',val:`${maxStages}/12`,c:'#ffaa00'}] as s}
      <div class="fd-stat">
        <div class="fd-stat-val" style="color:{s.c}">{s.val}</div>
        <div class="fd-stat-label">{s.label}</div>
      </div>
    {/each}
  </div>

  {#if loading}
    <div class="fd-loading">Loading fusion data…</div>
  {:else if campaigns.length === 0}
    <div class="fd-empty">
      <div class="fd-empty-icon">🔗</div>
      <div>NO CAMPAIGNS DETECTED</div>
      <p>The Fusion Engine correlates alerts sharing entities across tactic stages. Campaigns appear when 3+ tactic stages are observed on the same entity cluster.</p>
    </div>
  {:else}
    <div class="fd-content">
      <!-- Campaign list -->
      <div class="fd-list">
        {#each campaigns as c (c.id)}
          <div
            class="fd-campaign"
            class:fd-campaign--selected={selected?.id === c.id}
            style="border-left-color:{statusColor(c.status)}"
            onclick={() => selectCampaign(c)}
            role="button"
            tabindex="0"
            onkeydown={(e) => e.key==='Enter' && selectCampaign(c)}
          >
            <div class="fd-campaign-top">
              <span class="fd-campaign-id">{c.id.slice(0,12)}…</span>
              <span class="fd-campaign-status" style="color:{statusColor(c.status)}">{c.status.toUpperCase()}</span>
            </div>
            <div class="fd-kc-mini">
              {#each TACTIC_ORDER as tactic}
                <div class="fd-kc-slot" style="background:{tacticActive(c,tactic) ? confColor(c.confidence) : '#1e3040'}" title={tactic}></div>
              {/each}
            </div>
            <div class="fd-campaign-meta">
              <span>{c.stage_count}/12 stages</span>
              <span>{c.alert_count} alerts</span>
              <span style="color:{confColor(c.confidence)}">{Math.round(c.confidence*100)}% confidence</span>
            </div>
            <div class="fd-campaign-entities">{c.entities.slice(0,3).join(', ')}{c.entities.length>3 ? ` +${c.entities.length-3}` : ''}</div>
          </div>
        {/each}
      </div>

      <!-- Kill chain detail -->
      <div class="fd-detail">
        {#if !selected}
          <div class="fd-detail-empty"><div class="fd-empty-icon">🔗</div>SELECT A CAMPAIGN TO VIEW KILL CHAIN</div>
        {:else}
          <div class="fd-detail-header">
            <div>
              <div class="fd-detail-id">{selected.id}</div>
              <div class="fd-detail-dates">
                First seen: {new Date(selected.first_seen).toLocaleString()} ·
                Last activity: {new Date(selected.last_seen).toLocaleString()}
              </div>
            </div>
            <div class="fd-detail-conf">
              <div class="fd-detail-conf-val" style="color:{confColor(selected.confidence)}">{Math.round(selected.confidence*100)}%</div>
              <div class="fd-detail-conf-label">confidence</div>
            </div>
          </div>
          <div class="fd-entities">
            {#each selected.entities as e}<span class="fd-entity">{e}</span>{/each}
          </div>

          <div class="fd-kc-panel">
            <div class="fd-kc-label">KILL CHAIN PROGRESSION</div>
            {#if loadingKC}
              <div class="fd-loading">Loading kill chain…</div>
            {:else}
              <div class="fd-kc-track">
                {#each TACTIC_ORDER as tactic, i}
                  {@const stage = killChain.find(s => s.tactic_name.toLowerCase().includes(tactic.toLowerCase().split(' ')[0]))}
                  {@const active = stage || tacticActive(selected, tactic)}
                  <div
                    class="fd-kc-node"
                    class:fd-kc-node--active={active}
                    style={active ? `background:#2a0d15; border-color:#ff3355; color:#ff3355` : ''}
                    title={stage ? `${stage.hit_count} hits: ${stage.techniques.join(', ')}` : 'Not observed'}
                  >
                    <div class="fd-kc-node-dot">{active ? (stage ? stage.hit_count : '●') : '○'}</div>
                    <div class="fd-kc-node-label">{tactic}</div>
                  </div>
                  {#if i < 11}<div class="fd-kc-arrow" style="color:{active ? '#ff3355' : '#1e3040'}">→</div>{/if}
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .fd-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .fd-header { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:20px; }
  .fd-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#ff3355; }
  .fd-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .fd-refresh { background:#1e3040; border:1px solid #607070; color:#607070; padding:6px 14px; border-radius:4px; cursor:pointer; font-size:11px; font-family:inherit; }
  .fd-stats { display:grid; grid-template-columns:repeat(4,1fr); gap:14px; margin-bottom:20px; }
  .fd-stat { background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:14px; border-radius:4px; }
  .fd-stat-val   { font-size:26px; font-weight:700; }
  .fd-stat-label { font-size:10px; color:#607070; letter-spacing:.12em; margin-top:2px; }
  .fd-loading { color:#607070; padding:28px; text-align:center; font-size:12px; }
  .fd-empty { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:56px; text-align:center; color:#607070; }
  .fd-empty-icon { font-size:36px; opacity:0.3; margin-bottom:12px; }
  .fd-empty p { font-size:11px; max-width:440px; margin:8px auto 0; line-height:1.6; }
  .fd-content { display:grid; grid-template-columns:340px 1fr; gap:20px; align-items:start; }
  .fd-list { display:flex; flex-direction:column; gap:10px; }
  .fd-campaign { background:#0d1a1f; border:1px solid #1e3040; border-left:3px solid transparent; border-radius:4px; padding:14px; cursor:pointer; transition:border-color 100ms; }
  .fd-campaign--selected { border-color:#ff3355 !important; }
  .fd-campaign:hover:not(.fd-campaign--selected) { border-color:#2a3a48; }
  .fd-campaign-top { display:flex; justify-content:space-between; margin-bottom:6px; }
  .fd-campaign-id     { font-size:12px; font-weight:700; color:#c8d8d8; }
  .fd-campaign-status { font-size:10px; font-weight:700; letter-spacing:.1em; }
  .fd-kc-mini { display:flex; gap:2px; margin-bottom:7px; }
  .fd-kc-slot { flex:1; height:6px; border-radius:1px; }
  .fd-campaign-meta   { display:flex; gap:14px; font-size:10px; color:#607070; }
  .fd-campaign-entities { font-size:10px; color:#607070; margin-top:3px; }
  .fd-detail { }
  .fd-detail-empty { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:40px; text-align:center; color:#607070; font-size:12px; letter-spacing:.1em; }
  .fd-detail-header { background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ff3355; border-radius:6px; padding:18px; margin-bottom:14px; display:flex; justify-content:space-between; align-items:flex-start; }
  .fd-detail-id    { font-size:13px; color:#ff3355; letter-spacing:.08em; margin-bottom:5px; }
  .fd-detail-dates { font-size:11px; color:#607070; }
  .fd-detail-conf  { text-align:right; }
  .fd-detail-conf-val   { font-size:24px; font-weight:700; }
  .fd-detail-conf-label { font-size:10px; color:#607070; }
  .fd-entities { display:flex; flex-wrap:wrap; gap:5px; margin-bottom:14px; }
  .fd-entity   { background:#1e3040; color:#c8d8d8; padding:2px 8px; border-radius:2px; font-size:10px; }
  .fd-kc-panel { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:18px; }
  .fd-kc-label { font-size:10px; color:#607070; letter-spacing:.12em; margin-bottom:14px; }
  .fd-kc-track { display:flex; align-items:center; gap:3px; overflow-x:auto; padding-bottom:7px; }
  .fd-kc-node  { flex-shrink:0; padding:8px 10px; border-radius:3px; font-size:10px; text-align:center; min-width:80px; background:#0a1318; border:1px solid #1e3040; color:#3a5060; }
  .fd-kc-node--active {}
  .fd-kc-node-dot   { font-weight:700; margin-bottom:2px; font-size:12px; }
  .fd-kc-node-label { font-size:9px; letter-spacing:.04em; }
  .fd-kc-arrow { font-size:13px; flex-shrink:0; }
</style>
