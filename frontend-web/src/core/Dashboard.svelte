<!-- OBLIVRA Web — Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { logout } from '../services/auth';
  import { APP_CONTEXT } from '../context';
  import type { Severity } from '../components/SeverityIcon.svelte';
  import SeverityIcon from '../components/SeverityIcon.svelte';
  import CommandPalette, { type PaletteAction } from '../components/CommandPalette.svelte';
  import WarRoomGrid from '../components/WarRoomGrid.svelte';
  import FleetOverview from '../components/FleetOverview.svelte';
  import AccessibleTerminal from '../components/AccessibleTerminal.svelte';
  import { push } from './router.svelte';

  const nav = (path: string) => () => push(path);

  let status = $state<'INITIALIZING'|'READY'>('INITIALIZING');
  let heatmapData = $state<Severity[]>([]);
  let isPaletteOpen = $state(false);
  let isWarRoomMode = $state(false);

  const paletteActions: PaletteAction[] = [
    { id:'war-room',    label:'Toggle War Room Mode',      description:'High-density tactical view',           icon:'⚔️',  shortcut:'W',     action:() => { isWarRoomMode = !isWarRoomMode; } },
    { id:'onboard',     label:'Fleet Onboarding',          description:'Deploy new agents',                    icon:'🖥️',  shortcut:'G O',   action:nav('/onboarding') },
    { id:'fleet',       label:'Fleet Management',          description:'Agent status and bulk config',         icon:'🛰️',  action:nav('/fleet') },
    { id:'siem',        label:'SIEM Search',               description:'Search the event substrate',          icon:'🛡️',  shortcut:'G S',   action:nav('/siem/search') },
    { id:'alerts',      label:'Alert Management',          description:'Triage and manage alerts',             icon:'🚨',  action:nav('/alerts') },
    { id:'escalation',  label:'Escalation Center',         description:'Escalation policies and on-call',     icon:'⚡',  action:nav('/escalation') },
    { id:'playbooks',   label:'Playbook Builder',          description:'Build and execute SOAR playbooks',    icon:'🔧',  action:nav('/playbooks') },
    { id:'identity',    label:'Identity Administration',   description:'Users, roles, and federated SSO',     icon:'👥',  action:nav('/identity') },
    { id:'threatintel', label:'Threat Intelligence',       description:'IOC browser and campaign correlation', icon:'🔴',  action:nav('/threatintel') },
    { id:'enrich',      label:'Enrichment Viewer',         description:'GeoIP, DNS, asset mapping',            icon:'🔬',  action:nav('/enrich') },
    { id:'ueba',        label:'UEBA Dashboard',            description:'Entity behavior and risk scoring',    icon:'🧠',  action:nav('/ueba') },
    { id:'ndr',         label:'NDR Dashboard',             description:'Network flows and lateral movement',  icon:'🌐',  action:nav('/ndr') },
    { id:'ransomware',  label:'Ransomware Defense',        description:'Fleet-wide ransomware protection',    icon:'🦠',  action:nav('/ransomware') },
    { id:'mitre',       label:'MITRE ATT&CK Heatmap',     description:'Technique coverage visualization',    icon:'📊',  action:nav('/mitre-heatmap') },
    { id:'evidence',    label:'Evidence Vault',            description:'Legal-grade chain of custody',        icon:'🔍',  action:nav('/evidence') },
    { id:'regulator',   label:'Regulator Portal',          description:'Audit export and compliance packages',icon:'📄',  action:nav('/regulator') },
    { id:'lookups',     label:'Lookup Tables',             description:'CIDR/Exact/Regex enrichment tables',  icon:'🗒️',  action:nav('/lookups') },
    { id:'pb-metrics',  label:'Playbook Metrics',          description:'MTTR, success rates, bottlenecks',   icon:'📊',  action:nav('/playbook-metrics') },
    { id:'peer',        label:'Peer Analytics',            description:'Behavioral peer group outliers',      icon:'🧠',  action:nav('/peer-analytics') },
    { id:'fusion',      label:'Fusion Engine',             description:'Kill chain and campaign clustering',  icon:'🔗',  action:nav('/fusion') },
    { id:'logout',      label:'Terminate Session',         description:'Securely exit OBLIVRA',               icon:'🔌',  shortcut:'ALT+X', action:async () => { await logout(); push('/login'); } },
  ];

  function handleGlobalKey(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'p')) {
      e.preventDefault();
      isPaletteOpen = true;
    }
  }

  const severities: Severity[] = ['info','low','medium','high','critical'];

  onMount(() => {
    window.addEventListener('keydown', handleGlobalKey);
    setTimeout(() => { status = 'READY'; }, 800);
    heatmapData = Array.from({ length: 48 }, () => severities[Math.floor(Math.random() * severities.length)]);
  });

  onDestroy(() => window.removeEventListener('keydown', handleGlobalKey));
</script>

<main class="db-main" style:--war-header={isWarRoomMode ? 'rgba(127,29,29,0.2)' : 'transparent'}>
  <a href="#main-content" class="skip-link">Skip to main content</a>
  <div class="sr-only" role="status" aria-live="polite">
    {status === 'READY' ? 'Sovereign Substrate is READY.' : 'Initializing OBLIVRA Enterprise…'}
  </div>

  <!-- Header -->
  <header class="db-header" class:db-header--war={isWarRoomMode}>
    <div class="db-header-left">
      <div class="db-logo" class:db-logo--war={isWarRoomMode}>{isWarRoomMode ? '!' : 'O'}</div>
      <div>
        <h1 class="db-brand">OBLIVRA <span class:text-red-400={isWarRoomMode} class:text-accent={!isWarRoomMode}>{isWarRoomMode ? 'WAR ROOM' : 'ENTERPRISE'}</span></h1>
        <p class="db-context">{isWarRoomMode ? 'HIGH_DENSITY_TELEMETRY' : 'Sovereign SOC Platform'} // Context: {APP_CONTEXT}</p>
      </div>
    </div>
    <div class="db-header-right">
      <button class="db-war-btn" class:db-war-btn--active={isWarRoomMode} onclick={() => isWarRoomMode = !isWarRoomMode}>
        {isWarRoomMode ? 'Disable War Mode' : 'Enter War Room'}
      </button>
      <div class="db-status-block">
        <span class="db-status-label">System Status</span>
        <div class="db-status-row">
          <span class="db-dot" class:db-dot--ready={status==='READY'} class:db-dot--war={isWarRoomMode && status==='READY'}></span>
          <span class="db-status-val">{status}</span>
        </div>
      </div>
      <button class="db-logout" onclick={async () => { await logout(); push('/login'); }}>Terminate Session</button>
    </div>
  </header>

  <!-- Main -->
  <div id="main-content" class="db-content" tabindex="-1">
    {#if !isWarRoomMode}
      <section class="db-hero">
        <div class="db-phase-badge">Deployment Phase 0.5.0 // Svelte Migration Complete</div>
        <h2 class="db-headline">Multi-Tenant <br /><span class="db-headline-accent">Orchestration.</span></h2>
        <p class="db-subtext">Managing secure telemetry across distributed infrastructure. The OBLIVRA <strong>War Room</strong> mode provides zero-latency visibility into multi-tenant threat patterns and fleet health.</p>
      </section>

      <div class="db-stats-grid">
        {#each [
          { label:'Active Nodes',  value:'001', sub:'Primary Headless Server' },
          { label:'Event Velocity',value:'12.4',sub:'EPS (Ingest Active)' },
          { label:'Active Alerts', value:'003', sub:'Requires Intervention', severity:'high' as Severity },
          { label:'Threat Index',  value:'ELEVATED', sub:'Pattern Delta Detected', severity:'medium' as Severity },
        ] as stat}
          <div class="db-stat">
            <div class="db-stat-top">
              <span class="db-stat-label">{stat.label}</span>
              {#if stat.severity}<SeverityIcon severity={stat.severity} size={14} />{/if}
            </div>
            <div class="db-stat-value">{stat.value}</div>
            <p class="db-stat-sub">{stat.sub}</p>
          </div>
        {/each}
      </div>

      <FleetOverview />

      <section class="db-heatmap-section">
        <div class="db-heatmap-header">
          <h3 class="db-section-label">Inbound Event Density (Pattern-Fills)</h3>
          <div class="db-legend">
            {#each [['low','Low'],['medium','Med'],['high','High'],['critical','Crit']] as [s,l]}
              <div class="db-legend-item">
                <div class="db-legend-swatch" style="background:var(--alert-{s}); background-image:var(--pattern-{s});"></div>
                <span>{l}</span>
              </div>
            {/each}
          </div>
        </div>
        <div class="db-heatmap-grid">
          {#each heatmapData as s, i}
            <div
              class="db-heatmap-cell"
              style="background-color:var(--alert-{s}); background-image:var(--pattern-{s});"
              title="Severity: {s.toUpperCase()}"
              role="img"
              aria-label="{s} severity event block"
            ></div>
          {/each}
        </div>
      </section>

      <div class="db-actions-grid">
        {#each [
          { title:'Fleet Onboarding',    desc:'Deploy sovereign agents to your infrastructure.', shortcut:'F1', onClick:() => push('/onboarding') },
          { title:'Vault Orchestration', desc:'Access shared enterprise secrets and manage multi-tenant compliance.', shortcut:'F2', onClick:() => {} },
        ] as card}
          <div class="db-action-card" onclick={card.onClick} role="button" tabindex="0" onkeydown={(e) => e.key==='Enter' && card.onClick()}>
            <span class="db-action-shortcut">CMD + {card.shortcut}</span>
            <h3 class="db-action-title">{card.title}</h3>
            <p class="db-action-desc">{card.desc}</p>
            <div class="db-action-cta">Launch Operation →</div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="db-warroom">
        <div class="db-warroom-left"><WarRoomGrid /></div>
        <div class="db-warroom-right">
          <section>
            <h3 class="db-warroom-section-title" style="color:#f87171">Node Criticality</h3>
            <div class="db-criticality-card">
              <div class="db-crit-row">
                <span class="db-crit-host">LNUX-PROD-01</span>
                <span class="db-crit-status">CRITICAL</span>
              </div>
              <div class="db-crit-bar-bg"><div class="db-crit-bar"></div></div>
            </div>
          </section>
          <section>
            <h3 class="db-warroom-section-title">Global Ingest Load</h3>
            <div class="db-ingest-chart"></div>
          </section>
          <AccessibleTerminal />
        </div>
      </div>
    {/if}
  </div>

  <footer class="db-footer" aria-hidden="true">
    <div class="db-marquee">SECURE_SUBSTRATE // RAFT_CONSENSUS_STABLE // SIEM_PIPELINE_ACTIVE // NO_THREATS_DETECTED // OBLIVRA_ENTERPRISE_READY</div>
  </footer>

  <CommandPalette open={isPaletteOpen} onClose={() => isPaletteOpen = false} actions={paletteActions} />
</main>

<style>
  .db-main { min-height: 100vh; background: var(--surface-0); color: var(--text-primary); font-family: var(--font-mono); display: flex; flex-direction: column; }
  .skip-link { position: absolute; left: -9999px; }
  .skip-link:focus { position: fixed; top: 0; left: 0; z-index: 100; background: var(--accent-primary); color: #000; padding: 12px 16px; font-weight: 700; }
  .sr-only { position: absolute; width:1px; height:1px; overflow:hidden; clip:rect(0,0,0,0); }

  /* Header */
  .db-header { display:flex; justify-content:space-between; align-items:center; padding:16px 24px; border-bottom:1px solid var(--border-primary); background:var(--surface-1); position:sticky; top:0; z-index:50; transition:background 300ms ease; }
  .db-header--war { background:rgba(127,29,29,0.15); border-bottom-color:rgba(220,38,38,0.3); }
  .db-header-left { display:flex; align-items:center; gap:14px; }
  .db-logo { width:32px; height:32px; background:var(--accent-primary); display:grid; place-items:center; font-weight:900; color:#000; font-size:15px; transition:background 300ms; }
  .db-logo--war { background:#dc2626; box-shadow:0 0 15px rgba(224,64,64,0.4); }
  .db-brand { font-size:18px; font-weight:900; text-transform:uppercase; letter-spacing:-.03em; color:var(--text-heading); margin:0; }
  .text-accent { color:var(--accent-primary); }
  .text-red-400 { color:#f87171; }
  .db-context { font-size:10px; text-transform:uppercase; letter-spacing:.2em; color:var(--text-muted); margin:2px 0 0; opacity:0.7; }
  .db-header-right { display:flex; align-items:center; gap:20px; }
  .db-war-btn { font-size:10px; font-weight:700; text-transform:uppercase; letter-spacing:.1em; padding:4px 12px; border:1px solid #3a3d44; color:#9b9ea4; background:transparent; cursor:pointer; transition:all 100ms; font-family:var(--font-mono); }
  .db-war-btn--active { border-color:#dc2626; color:#f87171; background:rgba(127,29,29,0.15); }
  .db-status-block { display:flex; flex-direction:column; align-items:flex-end; }
  .db-status-label { font-size:10px; text-transform:uppercase; letter-spacing:.15em; color:var(--text-muted); }
  .db-status-row { display:flex; align-items:center; gap:7px; }
  .db-dot { width:8px; height:8px; border-radius:50%; background:var(--status-offline); animation:pulse 2s infinite; }
  .db-dot--ready { background:var(--status-online); box-shadow:0 0 8px var(--status-online); }
  .db-dot--war   { background:#f87171; box-shadow:0 0 8px #f87171; }
  .db-status-val { font-size:12px; font-weight:700; }
  .db-logout { height:40px; padding:0 20px; border:1px solid var(--border-primary); color:var(--text-muted); font-weight:700; text-transform:uppercase; letter-spacing:.06em; background:transparent; cursor:pointer; font-family:var(--font-mono); transition:all 150ms; font-size:11px; }
  .db-logout:hover { background:#dc2626; color:#fff; border-color:#dc2626; }

  /* Content */
  .db-content { flex:1; padding:32px; max-width:1280px; margin:0 auto; width:100%; display:flex; flex-direction:column; gap:40px; }

  /* Hero */
  .db-hero { display:flex; flex-direction:column; gap:16px; }
  .db-phase-badge { display:inline-block; padding:4px 12px; background:var(--accent-secondary,#0077b3); font-size:10px; font-weight:700; text-transform:uppercase; letter-spacing:.2em; color:#fff; }
  .db-headline { font-size:clamp(40px,6vw,72px); font-weight:900; font-style:italic; letter-spacing:-.04em; text-transform:uppercase; color:var(--text-heading); margin:0; line-height:0.92; }
  .db-headline-accent { color:var(--accent-primary); }
  .db-subtext { font-size:16px; color:var(--text-muted); max-width:560px; line-height:1.65; font-family:var(--font-ui); }

  /* Stats */
  .db-stats-grid { display:grid; grid-template-columns:repeat(4,1fr); gap:1px; border:1px solid rgba(255,255,255,0.04); background:rgba(255,255,255,0.04); }
  .db-stat { background:var(--surface-0); padding:24px; display:flex; flex-direction:column; gap:4px; position:relative; overflow:hidden; }
  .db-stat-top { display:flex; justify-content:space-between; align-items:flex-start; }
  .db-stat-label { font-size:10px; text-transform:uppercase; letter-spacing:.15em; color:var(--text-muted); }
  .db-stat-value { font-size:36px; font-weight:900; font-style:italic; letter-spacing:-.04em; color:var(--text-heading); }
  .db-stat-sub { font-size:12px; color:var(--text-muted); opacity:0.6; letter-spacing:.04em; margin:0; }

  /* Heatmap */
  .db-heatmap-section { display:flex; flex-direction:column; gap:12px; }
  .db-heatmap-header { display:flex; justify-content:space-between; align-items:flex-end; }
  .db-section-label { font-size:10px; font-weight:900; text-transform:uppercase; letter-spacing:.2em; color:var(--text-muted); margin:0; }
  .db-legend { display:flex; gap:16px; }
  .db-legend-item { display:flex; align-items:center; gap:5px; font-size:10px; text-transform:uppercase; font-weight:700; color:var(--text-muted); }
  .db-legend-swatch { width:12px; height:12px; background-size:4px 4px; }
  .db-heatmap-grid { display:grid; grid-template-columns:repeat(24,1fr); gap:3px; }
  .db-heatmap-cell { aspect-ratio:1; cursor:help; border:1px solid rgba(0,0,0,0.2); background-size:4px 4px; transition:transform 100ms ease; }
  .db-heatmap-cell:hover { transform:scale(1.1); z-index:1; }

  /* Action cards */
  .db-actions-grid { display:grid; grid-template-columns:1fr 1fr; gap:24px; padding-bottom:40px; }
  .db-action-card { border:1px solid var(--border-primary); background:var(--surface-1); padding:32px; cursor:pointer; position:relative; overflow:hidden; transition:border-color 150ms; }
  .db-action-card:hover { border-color:var(--accent-primary); }
  .db-action-card:hover .db-action-cta { opacity:1; transform:translateX(0); }
  .db-action-shortcut { position:absolute; top:8px; right:8px; font-size:8px; font-weight:700; color:var(--text-muted); opacity:0.2; transition:opacity 150ms; }
  .db-action-card:hover .db-action-shortcut { opacity:1; }
  .db-action-title { font-size:28px; font-weight:900; text-transform:uppercase; font-style:italic; letter-spacing:-.04em; color:var(--text-heading); margin:0 0 12px; transition:color 150ms; }
  .db-action-card:hover .db-action-title { color:var(--accent-primary); }
  .db-action-desc  { color:var(--text-muted); font-size:14px; font-family:var(--font-ui); line-height:1.5; margin:0 0 16px; }
  .db-action-cta   { font-size:11px; font-weight:700; text-transform:uppercase; letter-spacing:.15em; color:var(--accent-primary); opacity:0; transform:translateX(-10px); transition:all 200ms ease; }

  /* War room */
  .db-warroom { display:grid; grid-template-columns:1fr 380px; gap:24px; }
  .db-warroom-left {}
  .db-warroom-right { display:flex; flex-direction:column; gap:24px; }
  .db-warroom-section-title { font-size:10px; font-weight:900; text-transform:uppercase; letter-spacing:.2em; color:var(--text-muted); margin:0 0 10px; }
  .db-criticality-card { border:1px solid rgba(127,29,29,0.3); background:rgba(127,29,29,0.08); padding:14px; }
  .db-crit-row { display:flex; justify-content:space-between; margin-bottom:10px; }
  .db-crit-host   { font-size:10px; color:#9b9ea4; }
  .db-crit-status { font-size:10px; color:#f87171; font-weight:700; }
  .db-crit-bar-bg { width:100%; background:#1e2937; height:4px; }
  .db-crit-bar    { width:85%; height:100%; background:#dc2626; box-shadow:0 0 10px rgba(224,64,64,0.5); }
  .db-ingest-chart { height:100px; border:1px solid #1e2937; background:rgba(0,0,0,0.4); position:relative; overflow:hidden; }

  /* Footer marquee */
  .db-footer { border-top:1px solid var(--border-primary); overflow:hidden; padding:6px 0; background:var(--surface-0); }
  .db-marquee { white-space:nowrap; font-size:10px; text-transform:uppercase; letter-spacing:.3em; color:var(--text-muted); opacity:0.3; animation:marquee 30s linear infinite; display:inline-block; }
  @keyframes marquee { from { transform:translateX(100vw); } to { transform:translateX(-100%); } }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.5} }
</style>
