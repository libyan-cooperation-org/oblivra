<!-- OBLIVRA Web — EscalationCenter (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface EscalationLevel { level:number; name:string; users:string[]; channel:string; wait_mins:number; }
  interface EscalationPolicy { id:string; name:string; alert_types:string[]; levels:EscalationLevel[]; sla_mins:number; active:boolean; }
  interface ActiveEscalation { alert_id:string; policy_id:string; current_level:number; created_at:string; last_escalated_at:string; acked_by?:string; acked_at?:string; sla_breached:boolean; closed:boolean; }
  interface OnCallEntry { user_id:string; name:string; weekday_start:number; weekday_end:number; hour_start:number; hour_end:number; }

  const DAYS = ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'];
  const CH_COLOR: Record<string,string> = { slack:'#4A154B', email:'#00ffe7', webhook:'#ffaa00', sms:'#00ff88', teams:'#6264A7' };
  function msAgo(iso:string){ const d=Math.floor((Date.now()-new Date(iso).getTime())/60000); return d<60 ? `${d}m ago` : `${Math.floor(d/60)}h ${d%60}m ago`; }

  type Tab = 'policies'|'active'|'oncall'|'history';
  let tab         = $state<Tab>('policies');
  let policies    = $state<EscalationPolicy[]>([]);
  let active      = $state<ActiveEscalation[]>([]);
  let history     = $state<ActiveEscalation[]>([]);
  let onCall      = $state<{entries:OnCallEntry[];current?:OnCallEntry}|null>(null);
  let loading     = $state(true);

  let policyName  = $state('');
  let slaMins     = $state(30);
  let alertTypes  = $state('security_alert,failed_login');
  let saveMsg     = $state('');

  onMount(async () => {
    try { const r = await request<{policies:EscalationPolicy[]}>('/escalation/policies'); policies = r.policies??[]; } catch {}
    try { const r = await request<{escalations:ActiveEscalation[]}>('/escalation/active'); active = r.escalations??[]; } catch {}
    try { const r = await request<{escalations:ActiveEscalation[]}>('/escalation/history?limit=50'); history = r.escalations??[]; } catch {}
    try { onCall = await request('/escalation/oncall'); } catch {}
    loading = false;
  });

  async function savePolicy() {
    if (!policyName.trim()) return;
    const policy = { id:policyName.toLowerCase().replace(/\s+/g,'_'), name:policyName, sla_mins:slaMins, alert_types:alertTypes.split(',').map(s=>s.trim()), active:true,
      levels:[{level:1,name:'Analyst',users:['analyst@oblivra.io'],channel:'slack',wait_mins:10},{level:2,name:'Team Lead',users:['lead@oblivra.io'],channel:'email',wait_mins:15},{level:3,name:'Manager',users:['manager@oblivra.io'],channel:'email',wait_mins:20},{level:4,name:'CISO',users:['ciso@oblivra.io'],channel:'sms',wait_mins:999}]
    };
    try {
      await request('/escalation/policies', {method:'POST', body:JSON.stringify(policy)});
      saveMsg = '✓ Policy saved.';
      const r = await request<{policies:EscalationPolicy[]}>('/escalation/policies'); policies = r.policies??[];
    } catch(e:any) { saveMsg = `✗ ${e.message}`; }
  }

  async function ackAlert(alertId:string) {
    const user = JSON.parse(localStorage.getItem('oblivra_user')??'{}');
    await request('/escalation/ack', {method:'POST', body:JSON.stringify({alert_id:alertId, user_id:user.id??'unknown', comment:'Acknowledged via web console'})});
    const r = await request<{escalations:ActiveEscalation[]}>('/escalation/active'); active = r.escalations??[];
  }

  const openEscalations = $derived(active.filter(a => !a.closed));
  const slaBreached     = $derived(active.filter(a => a.sla_breached).length);
</script>

<div class="ec-page">
  <div class="ec-header">
    <h1 class="ec-title">⬡ ESCALATION CENTER</h1>
    <p class="ec-sub">Policy management · On-call scheduling · SLA tracking</p>
  </div>

  <div class="ec-stats">
    {#each [{l:'POLICIES',v:policies.length,c:'#ff6600'},{l:'ACTIVE',v:openEscalations.length,c:'#ff3355'},{l:'SLA BREACHED',v:slaBreached,c:'#ffaa00'},{l:'HISTORY',v:history.length,c:'#00ff88'}] as s}
      <div class="ec-stat" style="border-top-color:{s.c}"><div class="ec-stat-val" style="color:{s.c}">{s.v}</div><div class="ec-stat-label">{s.l}</div></div>
    {/each}
  </div>

  <div class="ec-tabs">
    {#each (['policies','active','oncall','history'] as Tab[]) as t}
      <button class="ec-tab {tab===t ? 'ec-tab--active' : ''}" onclick={() => tab=t}>{t.toUpperCase()}</button>
    {/each}
  </div>

  {#if tab === 'policies'}
    <div class="ec-two-col">
      <div class="ec-policy-list">
        {#each policies as p (p.id)}
          <div class="ec-policy-card">
            <div class="ec-policy-top">
              <div>
                <div class="ec-policy-name">{p.name}</div>
                <div class="ec-muted">SLA: {p.sla_mins}min · {p.alert_types?.join(', ')}</div>
              </div>
              <span class="ec-status-dot" style="color:{p.active ? '#00ff88' : '#ff3355'}">● {p.active ? 'ACTIVE' : 'INACTIVE'}</span>
            </div>
            {#each (p.levels??[]) as lvl}
              <div class="ec-level-row">
                <span class="ec-level-num">L{lvl.level}</span>
                <span class="ec-level-name">{lvl.name}</span>
                <span class="ec-channel" style="background:{CH_COLOR[lvl.channel]??'#1e3040'}22; border-color:{CH_COLOR[lvl.channel]??'#1e3040'}; color:{CH_COLOR[lvl.channel]??'#607070'}">{lvl.channel.toUpperCase()}</span>
                <span class="ec-muted">{lvl.wait_mins<999 ? `→ ${lvl.wait_mins}m` : 'terminal'}</span>
              </div>
            {/each}
          </div>
        {:else}
          <div class="ec-muted">No policies defined.</div>
        {/each}
      </div>
      <div class="ec-new-policy">
        <div class="ec-form-title">NEW POLICY</div>
        {#each [{label:'POLICY NAME',bind:policyName,placeholder:'Critical Security'},{label:'ALERT TYPES',bind:alertTypes,placeholder:'security_alert,failed_login'}] as f}
          <div class="ec-field">
            <div class="ec-field-label">{f.label}</div>
            <input type="text" value={f.label==='POLICY NAME' ? policyName : alertTypes} oninput={(e)=>{if(f.label==='POLICY NAME') policyName=(e.target as HTMLInputElement).value; else alertTypes=(e.target as HTMLInputElement).value;}} placeholder={f.placeholder} class="ec-input" />
          </div>
        {/each}
        <div class="ec-field">
          <div class="ec-field-label">SLA (MINUTES)</div>
          <input type="number" bind:value={slaMins} min="1" max="1440" class="ec-input" />
        </div>
        <div class="ec-muted ec-note">Levels auto-seeded: Analyst → Lead → Manager → CISO</div>
        <button class="ec-save-btn" onclick={savePolicy}>SAVE POLICY</button>
        {#if saveMsg}<div class="ec-save-msg" style="color:{saveMsg.startsWith('✓') ? '#00ff88' : '#ff3355'}">{saveMsg}</div>{/if}
      </div>
    </div>

  {:else if tab === 'active'}
    {#each openEscalations as esc (esc.alert_id)}
      <div class="ec-active-card" style="border-left-color:{esc.sla_breached ? '#ffaa00' : '#ff3355'}; border-color:{esc.sla_breached ? '#ffaa00' : '#1e3040'}">
        <div class="ec-active-top">
          <div>
            <div class="ec-active-id">{esc.alert_id}</div>
            <div class="ec-muted">Policy: <span style="color:#ff6600">{esc.policy_id}</span> · Level: <span style="color:#ff3355">L{esc.current_level}</span> · {msAgo(esc.created_at)}</div>
            {#if esc.sla_breached}<div class="ec-sla-warn">⚠ SLA BREACHED</div>{/if}
          </div>
          <button class="ec-ack-btn" onclick={() => ackAlert(esc.alert_id)}>ACKNOWLEDGE</button>
        </div>
      </div>
    {:else}
      <div class="ec-empty">No active escalations. All clear.</div>
    {/each}

  {:else if tab === 'oncall'}
    <div class="ec-table-wrap">
      {#if onCall?.current}
        <div class="ec-oncall-now">● NOW ON-CALL: {onCall.current.name}</div>
      {/if}
      <table class="ec-table">
        <thead><tr>{#each ['ENGINEER','DAYS','HOURS (UTC)'] as h}<th>{h}</th>{/each}</tr></thead>
        <tbody>
          {#each (onCall?.entries??[]) as e}
            <tr class="ec-row"><td>{e.name}</td><td class="ec-muted">{DAYS[e.weekday_start]}–{DAYS[e.weekday_end]}</td><td class="ec-muted">{String(e.hour_start).padStart(2,'0')}:00 – {String(e.hour_end).padStart(2,'0')}:00</td></tr>
          {:else}
            <tr><td colspan="3" class="ec-empty">No on-call data.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>

  {:else}
    <div class="ec-table-wrap">
      <table class="ec-table">
        <thead><tr>{#each ['ALERT ID','POLICY','FINAL LEVEL','ACKED BY','ACKED AT','SLA'] as h}<th>{h}</th>{/each}</tr></thead>
        <tbody>
          {#each history as h (h.alert_id)}
            <tr class="ec-row">
              <td>{h.alert_id}</td><td style="color:#ff6600">{h.policy_id}</td><td style="color:#ff3355">L{h.current_level}</td>
              <td class="ec-muted">{h.acked_by||'—'}</td>
              <td class="ec-muted">{h.acked_at ? new Date(h.acked_at).toLocaleString() : '—'}</td>
              <td><span style="color:{h.sla_breached ? '#ffaa00' : '#00ff88'}">{h.sla_breached ? '⚠ BREACHED' : '✓ OK'}</span></td>
            </tr>
          {:else}
            <tr><td colspan="6" class="ec-empty">No history yet.</td></tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .ec-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .ec-header { margin-bottom:20px; }
  .ec-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#ff6600; }
  .ec-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .ec-stats  { display:grid; grid-template-columns:repeat(4,1fr); gap:14px; margin-bottom:20px; }
  .ec-stat   { background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid; padding:14px; border-radius:4px; }
  .ec-stat-val   { font-size:26px; font-weight:700; }
  .ec-stat-label { font-size:10px; color:#607070; letter-spacing:.12em; margin-top:2px; }
  .ec-tabs { display:flex; border-bottom:1px solid #1e3040; margin-bottom:20px; }
  .ec-tab  { padding:8px 18px; cursor:pointer; font-size:11px; letter-spacing:.12em; border:none; border-bottom:2px solid transparent; background:none; color:#607070; font-family:inherit; }
  .ec-tab--active { border-bottom-color:#ff6600; color:#ff6600; }
  .ec-two-col { display:grid; grid-template-columns:1fr 300px; gap:20px; align-items:start; }
  .ec-policy-list { display:flex; flex-direction:column; gap:14px; }
  .ec-policy-card { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:16px; }
  .ec-policy-top  { display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:10px; }
  .ec-policy-name { color:#ff6600; font-size:13px; letter-spacing:.1em; }
  .ec-status-dot  { font-size:10px; letter-spacing:.1em; }
  .ec-level-row   { display:flex; align-items:center; gap:10px; padding:5px 8px; background:#0a1318; border-radius:3px; margin-bottom:4px; font-size:11px; }
  .ec-level-num   { color:#607070; min-width:20px; }
  .ec-level-name  { color:#c8d8d8; min-width:80px; }
  .ec-channel     { padding:1px 6px; border:1px solid; border-radius:2px; font-size:10px; letter-spacing:.08em; }
  .ec-new-policy  { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:16px; position:sticky; top:14px; }
  .ec-form-title  { font-size:10px; color:#607070; letter-spacing:.12em; margin-bottom:12px; }
  .ec-field       { margin-bottom:10px; }
  .ec-field-label { font-size:10px; color:#607070; margin-bottom:3px; }
  .ec-input       { width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:6px 8px; border-radius:3px; font-size:11px; font-family:inherit; box-sizing:border-box; outline:none; }
  .ec-note        { font-size:10px; margin-bottom:10px; }
  .ec-save-btn    { width:100%; background:#ff6600; color:#080f12; border:none; padding:8px; border-radius:3px; cursor:pointer; font-weight:700; font-size:12px; letter-spacing:.1em; font-family:inherit; }
  .ec-save-msg    { margin-top:7px; font-size:11px; }
  .ec-active-card { background:#0d1a1f; border:1px solid; border-left:4px solid; border-radius:6px; padding:16px; margin-bottom:12px; }
  .ec-active-top  { display:flex; justify-content:space-between; align-items:flex-start; }
  .ec-active-id   { color:#c8d8d8; font-size:13px; margin-bottom:3px; }
  .ec-sla-warn    { color:#ffaa00; font-size:10px; letter-spacing:.1em; margin-top:4px; }
  .ec-ack-btn     { background:#00ff88; color:#080f12; border:none; padding:6px 14px; border-radius:4px; cursor:pointer; font-size:11px; font-weight:700; font-family:inherit; white-space:nowrap; }
  .ec-table-wrap  { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; }
  .ec-oncall-now  { padding:8px 14px; background:#0a1318; border-bottom:1px solid #1e3040; color:#00ff88; font-size:11px; }
  .ec-table       { width:100%; border-collapse:collapse; font-size:11px; }
  .ec-table thead tr { border-bottom:1px solid #1e3040; background:#0a1318; }
  .ec-table th    { padding:9px 14px; text-align:left; color:#607070; letter-spacing:.1em; font-weight:400; font-size:10px; }
  .ec-row { border-bottom:1px solid #0a1318; transition:background 80ms; }
  .ec-row:hover { background:#111f28; }
  .ec-row td      { padding:9px 14px; color:#c8d8d8; }
  .ec-muted       { color:#607070; font-size:11px; }
  .ec-empty       { padding:28px; text-align:center; color:#607070; font-size:12px; }
</style>
