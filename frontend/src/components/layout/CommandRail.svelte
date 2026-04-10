<!--
  OBLIVRA — CommandRail (Svelte 5)
  Vertical navigation rail with all 60+ route items, organized into sections.
  Context-aware: hides desktop-only items in browser mode and vice versa.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';
  import { IS_DESKTOP, IS_BROWSER, IS_HYBRID, isRouteAvailable } from '@lib/context';
  import type { NavTab } from '@lib/types';

  // ── Route map ──────────────────────────────────────────────────────────
  const routeMap: Record<string, string> = {
    dashboard: '/dashboard', siem: '/siem', alerts: '/alerts',
    cases: '/cases', evidence: '/evidence', 'chain-of-custody': '/chain-of-custody',
    'alert-management': '/alert-management', 'siem-search': '/siem-search',
    topology: '/topology', health: '/monitoring', ops: '/ops',
    response: '/response', escalation: '/escalation',
    'playbook-builder': '/playbook-builder', ueba: '/ueba',
    'ueba-overview': '/ueba-overview', 'threat-hunter': '/threat-hunter',
    'threat-intel-dashboard': '/threat-intel-dashboard', enrichment: '/enrichment',
    'threat-map': '/threat-map',
    ndr: '/ndr', 'ndr-overview': '/ndr-overview', 'purple-team': '/purple-team',
    graph: '/graph', ransomware: '/ransomware', 'ransomware-ui': '/ransomware-ui',
    simulation: '/simulation', compliance: '/compliance', vault: '/vault',
    'war-mode': '/war-mode', forensics: '/forensics',
    'remote-forensics': '/remote-forensics', security: '/trust',
    temporal: '/temporal-integrity', lineage: '/lineage',
    decisions: '/decisions', ledger: '/ledger', replay: '/response-replay',
    plugins: '/plugins', executive: '/executive', settings: '/workspace',
    'ai-assistant': '/ai-assistant', 'mitre-heatmap': '/mitre-heatmap',
    license: '/license', team: '/team', hosts: '/hosts',
    terminal: '/terminal', tunnels: '/tunnels', recordings: '/recordings',
    snippets: '/snippets', notes: '/notes', sync: '/sync',
    agents: '/agents', 'fleet-management': '/fleet-management',
    identity: '/identity', 'identity-admin': '/identity-admin', soc: '/soc',
    ssh: '/ssh',
  };

  type NavContext = 'both' | 'desktop' | 'browser';

  interface NavItem {
    id: string;
    icon: string;
    label: string;
    context?: NavContext;
    urgent?: boolean;
  }

  function isItemVisible(item: NavItem): boolean {
    const ctx = item.context ?? 'both';
    if (ctx === 'both') return true;
    if (ctx === 'desktop') return IS_DESKTOP || IS_HYBRID;
    if (ctx === 'browser') return IS_BROWSER || IS_HYBRID;
    return true;
  }

  function go(id: string) {
    appStore.setActiveNavTab(id as NavTab);
    const route = routeMap[id];
    if (route) push(route);
  }

  // ── Icon map (SVG path data) ──────────────────────────────────────────
  const icons: Record<string, string> = {
    dashboard: 'M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z',
    siem: 'M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2zM22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z',
    alerts: 'M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0',
    terminal: 'M2 4h20v16H2zM8 10l3 2-3 2M13 15h3',
    hosts: 'M4 4h16v6H4zM4 14h16v6H4z',
    topology: 'M12 12m-3 0a3 3 0 106 0 3 3 0 10-6 0M5 18m-2 0a2 2 0 104 0 2 2 0 10-4 0M19 18m-2 0a2 2 0 104 0 2 2 0 10-4 0',
    health: 'M22 12h-4l-3 9L9 3l-3 9H2',
    ops: 'M18 20V10M12 20V4M6 20v-4',
    tunnels: 'M12 2a10 10 0 100 20 10 10 0 000-20M2 12h20',
    snippets: 'M16 18l6-6-6-6M8 6l-6 6 6 6',
    recordings: 'M12 2a10 10 0 100 20 10 10 0 000-20M12 8a4 4 0 100 8 4 4 0 000-8',
    notes: 'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M16 13H8M16 17H8',
    sync: 'M23 4v6h-6M1 20v-6h6',
    plugins: 'M21 16V8l-9-5-9 5v8l9 5 9-5z',
    compliance: 'M9 11l3 3L22 4M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11',
    vault: 'M3 11h18v11H3zM7 11V7a5 5 0 0110 0v4',
    settings: 'M12 15a3 3 0 100-6 3 3 0 000 6z',
    security: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
    forensics: 'M11 19a8 8 0 100-16 8 8 0 000 16zM21 21l-4.35-4.35',
    agents: 'M4 4h16v16H4zM9 9h.01M15 9h.01M9 15h6',
    soc: 'M3 3h18v18H3zM3 9h18M9 21V9',
    ueba: 'M16 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M8.5 11a4 4 0 100-8 4 4 0 000 8M20 8v6M23 11h-6',
    hunter: 'M11 19a8 8 0 100-16 8 8 0 000 16zM21 21l-4.35-4.35M8 11h6M11 8v6',
    ndr: 'M6 9a3 3 0 100-6 3 3 0 000 6zM18 9a3 3 0 100-6 3 3 0 000 6zM6 21a3 3 0 100-6 3 3 0 000 6zM18 21a3 3 0 100-6 3 3 0 000 6z',
    purple: 'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
    graph: 'M12 8a3 3 0 100-6 3 3 0 000 6zM5 22a3 3 0 100-6 3 3 0 000 6zM19 22a3 3 0 100-6 3 3 0 000 6z',
    war: 'M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z',
    response: 'M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1zM4 22v-7',
    executive: 'M12 20V10M18 20V4M6 20v-4',
    ai: 'M12 2a10 10 0 100 20h10M12 12L2.69 7.11M12 12l4.89 8.69',
    mitre: 'M3 3h18v18H3zM3 9h18M3 15h18M9 3v18M15 3v18',
    identity: 'M3 4h18v16H3zM9 12a2 2 0 100-4 2 2 0 000 4M15 8h2M15 12h2M7 16h10',
    audit: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10zM9 12l2 2 4-4',
  };

  // ── Nav sections ──────────────────────────────────────────────────────
  const observe: NavItem[] = [
    { id: 'dashboard', icon: 'dashboard', label: 'Dash' },
    { id: 'siem', icon: 'siem', label: 'SIEM' },
    { id: 'siem-search', icon: 'hunter', label: 'Search', context: 'browser' },
    { id: 'alerts', icon: 'alerts', label: 'Alerts' },
    { id: 'alert-management', icon: 'alerts', label: 'AlertMgr', context: 'browser' },
    { id: 'recordings', icon: 'recordings', label: 'Recs', context: 'desktop' },
    { id: 'topology', icon: 'topology', label: 'Net' },
    { id: 'mitre-heatmap', icon: 'mitre', label: 'Mitre' },
    { id: 'threat-map', icon: 'topology', label: 'Map' },
    { id: 'health', icon: 'health', label: 'Health' },
  ];

  const operate: NavItem[] = [
    { id: 'terminal', icon: 'terminal', label: 'Shell', context: 'desktop' },
    { id: 'ssh', icon: 'vault', label: 'SSH', context: 'desktop' },
    { id: 'tunnels', icon: 'tunnels', label: 'Tunnels', context: 'desktop' },
    { id: 'hosts', icon: 'hosts', label: 'Hosts' },
    { id: 'agents', icon: 'agents', label: 'Agents', context: 'browser' },
    { id: 'fleet-management', icon: 'agents', label: 'Fleet', context: 'browser' },
    { id: 'ops', icon: 'ops', label: 'Ops' },
    { id: 'soc', icon: 'soc', label: 'SOC', context: 'browser' },
    { id: 'escalation', icon: 'response', label: 'Escalate', context: 'browser' },
    { id: 'playbook-builder', icon: 'purple', label: 'Playbooks', context: 'browser' },
    { id: 'snippets', icon: 'snippets', label: 'Snips', context: 'desktop' },
    { id: 'notes', icon: 'notes', label: 'Notes', context: 'desktop' },
    { id: 'ai-assistant', icon: 'ai', label: 'AI Shell' },
    { id: 'cases', icon: 'forensics', label: 'Cases' },
    { id: 'ledger', icon: 'security', label: 'Ledger' },
    { id: 'response', icon: 'response', label: 'SOAR' },
  ];

  const intel: NavItem[] = [
    { id: 'ueba', icon: 'ueba', label: 'UEBA' },
    { id: 'ueba-overview', icon: 'ueba', label: 'UEBA+', context: 'browser' },
    { id: 'threat-hunter', icon: 'hunter', label: 'Hunt' },
    { id: 'threat-intel-dashboard', icon: 'security', label: 'TI', context: 'browser' },
    { id: 'enrichment', icon: 'forensics', label: 'Enrich', context: 'browser' },
    { id: 'ndr', icon: 'ndr', label: 'NDR' },
    { id: 'ndr-overview', icon: 'ndr', label: 'NDR+', context: 'browser' },
    { id: 'purple-team', icon: 'purple', label: 'Purple' },
    { id: 'graph', icon: 'graph', label: 'Graph' },
  ];

  const govern: NavItem[] = [
    { id: 'compliance', icon: 'compliance', label: 'Comply' },
    { id: 'vault', icon: 'vault', label: 'Vault' },
    { id: 'identity', icon: 'identity', label: 'Users', context: 'browser' },
    { id: 'identity-admin', icon: 'identity', label: 'IdAdmin', context: 'browser' },
    { id: 'security', icon: 'security', label: 'Trust' },
    { id: 'forensics', icon: 'forensics', label: 'Forensics' },
    { id: 'remote-forensics', icon: 'forensics', label: 'RemForen', context: 'browser' },
    { id: 'ransomware', icon: 'war', label: 'Ransom' },
    { id: 'ransomware-ui', icon: 'war', label: 'RansomW', context: 'browser' },
    { id: 'war-mode', icon: 'war', label: 'WarMode' },
  ];

  const system: NavItem[] = [
    { id: 'executive', icon: 'executive', label: 'Exec' },
    { id: 'plugins', icon: 'plugins', label: 'Plugins' },
    { id: 'sync', icon: 'sync', label: 'Sync', context: 'desktop' },
    { id: 'license', icon: 'vault', label: 'License' },
    { id: 'settings', icon: 'settings', label: 'Config' },
  ];

  const auditItems = [
    { id: 'temporal', label: 'Temporal Integrity' },
    { id: 'lineage', label: 'Data Lineage' },
    { id: 'decisions', label: 'Decision Log' },
    { id: 'ledger', label: 'Evidence Ledger' },
    { id: 'replay', label: 'Response Replay' },
  ];

  let auditOpen = $state(false);
  let flyoutTop = $state(0);

  const auditActive = $derived(auditItems.some(i => i.id === appStore.activeNavTab));

  function visibleItems(items: NavItem[]) {
    return items.filter(isItemVisible);
  }
</script>

<style>
  .cr-rail {
    width: 64px;
    min-width: 64px;
    height: 100%;
    background: var(--surface-1);
    border-right: 1px solid var(--border-primary);
    display: flex;
    flex-direction: column;
    z-index: 1000;
    overflow-y: auto;
    overflow-x: hidden;
  }
  .cr-rail::-webkit-scrollbar { width: 0; }

  .cr-logo {
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-bottom: 1px solid var(--border-primary);
    flex-shrink: 0;
  }

  .cr-nav-btn {
    position: relative;
    background: transparent;
    border: none;
    color: var(--text-muted);
    width: 100%;
    padding: 8px 0 6px 0;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
    transition: color 100ms ease, background 100ms ease;
    border-radius: 0;
  }
  .cr-nav-btn:hover:not(.locked) {
    color: var(--text-secondary);
    background: rgba(87, 139, 255, 0.06);
  }
  .cr-nav-btn.active {
    color: var(--accent-primary) !important;
    background: rgba(87, 139, 255, 0.12) !important;
  }
  .cr-nav-btn.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 20%;
    bottom: 20%;
    width: 3px;
    background: var(--accent-primary);
    border-radius: 0 3px 3px 0;
  }
  .cr-nav-btn.locked {
    opacity: 0.28;
    cursor: not-allowed !important;
    pointer-events: none;
  }
  .cr-nav-icon {
    width: 18px;
    height: 18px;
    opacity: 0.5;
    transition: opacity 100ms ease;
  }
  .cr-nav-btn.active .cr-nav-icon {
    opacity: 1;
    filter: drop-shadow(0 0 6px rgba(87, 139, 255, 0.4));
  }
  .cr-nav-label {
    font-family: var(--font-ui);
    font-size: 9px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.3px;
    line-height: 1;
  }
  .cr-nav-btn.active .cr-nav-label { font-weight: 700; }

  .cr-divider {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 8px 4px 3px 4px;
    margin-top: 2px;
  }
  .cr-divider-label {
    font-family: var(--font-mono);
    font-size: 6.5px;
    font-weight: 700;
    color: var(--text-muted);
    opacity: 0.35;
    letter-spacing: 1px;
    text-transform: uppercase;
  }

  .cr-flyout {
    position: fixed;
    left: 65px;
    background: var(--surface-2);
    border: 1px solid var(--border-secondary);
    border-radius: var(--radius-md);
    z-index: 2000;
    min-width: 192px;
    padding: 4px;
    box-shadow: var(--shadow-lg);
  }
  .cr-flyout-header {
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    color: var(--text-muted);
    padding: 8px 10px 6px 10px;
    letter-spacing: 1px;
    text-transform: uppercase;
    border-bottom: 1px solid var(--border-primary);
    margin-bottom: 4px;
  }
  .cr-flyout-item {
    display: block;
    width: 100%;
    background: transparent;
    border: none;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    padding: 7px 10px;
    text-align: left;
    font-family: var(--font-ui);
    font-size: 12px;
    cursor: pointer;
    transition: all 100ms ease;
  }
  .cr-flyout-item:hover {
    background: var(--surface-3);
    color: var(--text-primary);
  }
  .cr-flyout-item.active {
    background: rgba(87, 139, 255, 0.12);
    color: var(--accent-primary);
    font-weight: 600;
  }
</style>

<nav class="cr-rail">
  <!-- Logo -->
  <div class="cr-logo">
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
      <path d="M12 2L4 6.5v11L12 22l8-4.5v-11L12 2z" stroke="var(--accent-primary)" stroke-width="1.5" fill="none"/>
      <path d="M12 6.5L7 9.25v5.5L12 17.5l5-2.75v-5.5L12 6.5z" fill="var(--accent-primary)" opacity="0.2"/>
      <circle cx="12" cy="12" r="2" fill="var(--accent-primary)"/>
    </svg>
  </div>

  <!-- OBS section -->
  <div class="cr-divider"><span class="cr-divider-label">OBS</span></div>
  {#each visibleItems(observe) as item}
    {@const route = routeMap[item.id]}
    {@const available = route ? isRouteAvailable(route) : true}
    <button
      class="cr-nav-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — not available in this mode' : ''}"
      onclick={() => available && go(item.id)}
    >
      <div class="cr-nav-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
          <path d={icons[item.icon] ?? icons.dashboard}/>
        </svg>
      </div>
      <span class="cr-nav-label">{item.label}</span>
    </button>
  {/each}

  <!-- OPS section -->
  <div class="cr-divider"><span class="cr-divider-label">OPS</span></div>
  {#each visibleItems(operate) as item}
    {@const route = routeMap[item.id]}
    {@const available = route ? isRouteAvailable(route) : true}
    <button
      class="cr-nav-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — not available in this mode' : ''}"
      onclick={() => available && go(item.id)}
    >
      <div class="cr-nav-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
          <path d={icons[item.icon] ?? icons.dashboard}/>
        </svg>
      </div>
      <span class="cr-nav-label">{item.label}</span>
    </button>
  {/each}

  <!-- INTEL section -->
  <div class="cr-divider"><span class="cr-divider-label">INTEL</span></div>
  {#each visibleItems(intel) as item}
    {@const route = routeMap[item.id]}
    {@const available = route ? isRouteAvailable(route) : true}
    <button
      class="cr-nav-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — not available in this mode' : ''}"
      onclick={() => available && go(item.id)}
    >
      <div class="cr-nav-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
          <path d={icons[item.icon] ?? icons.dashboard}/>
        </svg>
      </div>
      <span class="cr-nav-label">{item.label}</span>
    </button>
  {/each}

  <!-- GOV section -->
  <div class="cr-divider"><span class="cr-divider-label">GOV</span></div>
  {#each visibleItems(govern) as item}
    {@const route = routeMap[item.id]}
    {@const available = route ? isRouteAvailable(route) : true}
    <button
      class="cr-nav-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — not available in this mode' : ''}"
      onclick={() => available && go(item.id)}
    >
      <div class="cr-nav-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
          <path d={icons[item.icon] ?? icons.dashboard}/>
        </svg>
      </div>
      <span class="cr-nav-label">{item.label}</span>
    </button>
  {/each}

  <!-- Audit flyout trigger -->
  <button
    class="cr-nav-btn"
    class:active={auditActive}
    title="Audit Trail"
    onmouseenter={(e) => { flyoutTop = e.currentTarget.getBoundingClientRect().top; auditOpen = true; }}
  >
    <div class="cr-nav-icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
        <path d={icons.audit}/>
      </svg>
    </div>
    <span class="cr-nav-label">Audit</span>
  </button>

  {#if auditOpen}
    <div class="cr-flyout" style="top: {flyoutTop}px;" role="menu" tabindex="-1" onmouseleave={() => auditOpen = false}>
      <div class="cr-flyout-header">AUDIT TRAIL</div>
      {#each auditItems as item}
        <button
          class="cr-flyout-item"
          class:active={appStore.activeNavTab === item.id}
          onclick={() => { go(item.id); auditOpen = false; }}
        >
          {item.label}
        </button>
      {/each}
    </div>
  {/if}

  <!-- Spacer -->
  <div class="flex-1"></div>

  <!-- SYS section -->
  <div class="cr-divider"><span class="cr-divider-label">SYS</span></div>
  {#each visibleItems(system) as item}
    {@const route = routeMap[item.id]}
    {@const available = route ? isRouteAvailable(route) : true}
    <button
      class="cr-nav-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — not available in this mode' : ''}"
      onclick={() => available && go(item.id)}
    >
      <div class="cr-nav-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6">
          <path d={icons[item.icon] ?? icons.dashboard}/>
        </svg>
      </div>
      <span class="cr-nav-label">{item.label}</span>
    </button>
  {/each}

  <div class="h-2"></div>
</nav>
