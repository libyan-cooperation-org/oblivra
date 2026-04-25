<!--
  OBLIVRA — CommandRail v2
  44px icon-only rail — matches design spec exactly.
  Labels removed from rail buttons (tooltip on hover).
  Flyout pattern preserved for Audit.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';
  import { IS_DESKTOP, IS_BROWSER, IS_HYBRID, isRouteAvailable } from '@lib/context';
  import type { NavTab } from '@lib/types';

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
    ssh: '/ssh', investigation: '/investigation', timeline: '/timeline',
    secrets: '/secrets', suppression: '/suppression', admin: '/admin',
    operator: '/operator', shortcuts: '/shortcuts',
  };

  type NavContext = 'both' | 'desktop' | 'browser';
  interface NavItem { id: string; icon: string; label: string; context?: NavContext; }

  function visible(items: NavItem[]) {
    return items.filter(item => {
      const ctx = item.context ?? 'both';
      if (ctx === 'both')    return true;
      if (ctx === 'desktop') return IS_DESKTOP || IS_HYBRID;
      if (ctx === 'browser') return IS_BROWSER || IS_HYBRID;
      return true;
    });
  }

  function go(id: string) {
    appStore.setActiveNavTab(id as NavTab);
    const route = routeMap[id];
    if (route) push(route);
  }

  // SVG path data — 24×24 viewBox, stroke icons
  const icons: Record<string, string> = {
    dashboard: 'M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z',
    siem:      'M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2zM22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z',
    alerts:    'M18 8A6 6 0 006 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 01-3.46 0',
    terminal:  'M2 4h20v16H2zM8 10l3 2-3 2M13 15h3',
    hosts:     'M4 4h16v6H4zM4 14h16v6H4z',
    topology:  'M12 12m-3 0a3 3 0 106 0 3 3 0 10-6 0M5 18m-2 0a2 2 0 104 0 2 2 0 10-4 0M19 18m-2 0a2 2 0 104 0 2 2 0 10-4 0',
    health:    'M22 12h-4l-3 9L9 3l-3 9H2',
    ops:       'M18 20V10M12 20V4M6 20v-4',
    tunnels:   'M12 2a10 10 0 100 20 10 10 0 000-20M2 12h20',
    snippets:  'M16 18l6-6-6-6M8 6l-6 6 6 6',
    recordings:'M12 2a10 10 0 100 20 10 10 0 000-20M12 8a4 4 0 100 8 4 4 0 000-8',
    notes:     'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6M16 13H8M16 17H8',
    sync:      'M23 4v6h-6M1 20v-6h6',
    plugins:   'M21 16V8l-9-5-9 5v8l9 5 9-5z',
    compliance:'M9 11l3 3L22 4M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11',
    vault:     'M3 11h18v11H3zM7 11V7a5 5 0 0110 0v4',
    settings:  'M12 15a3 3 0 100-6 3 3 0 000 6zM19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-2 2 2 2 0 01-2-2v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83 0 2 2 0 010-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 01-2-2 2 2 0 012-2h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 010-2.83 2 2 0 012.83 0l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 012-2 2 2 0 012 2v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 0 2 2 0 010 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 012 2 2 2 0 01-2 2h-.09a1.65 1.65 0 00-1.51 1z',
    security:  'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
    forensics: 'M11 19a8 8 0 100-16 8 8 0 000 16zM21 21l-4.35-4.35',
    agents:    'M4 4h16v16H4zM9 9h.01M15 9h.01M9 15h6',
    soc:       'M3 3h18v18H3zM3 9h18M9 21V9',
    ueba:      'M16 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M8.5 11a4 4 0 100-8 4 4 0 000 8M20 8v6M23 11h-6',
    hunter:    'M11 19a8 8 0 100-16 8 8 0 000 16zM21 21l-4.35-4.35M8 11h6M11 8v6',
    ndr:       'M6 9a3 3 0 100-6 3 3 0 000 6zM18 9a3 3 0 100-6 3 3 0 000 6zM6 21a3 3 0 100-6 3 3 0 000 6zM18 21a3 3 0 100-6 3 3 0 000 6z',
    purple:    'M13 2L3 14h9l-1 8 10-12h-9l1-8z',
    graph:     'M12 8a3 3 0 100-6 3 3 0 000 6zM5 22a3 3 0 100-6 3 3 0 000 6zM19 22a3 3 0 100-6 3 3 0 000 6z',
    war:       'M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z',
    response:  'M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1zM4 22v-7',
    executive: 'M12 20V10M18 20V4M6 20v-4',
    ai:        'M12 2a10 10 0 100 20h10M12 12L2.69 7.11M12 12l4.89 8.69',
    mitre:     'M3 3h18v18H3zM3 9h18M3 15h18M9 3v18M15 3v18',
    identity:  'M3 4h18v16H3zM9 12a2 2 0 100-4 2 2 0 000 4M15 8h2M15 12h2M7 16h10',
    audit:     'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10zM9 12l2 2 4-4',
    cases:     'M20 7H4a2 2 0 00-2 2v10a2 2 0 002 2h16a2 2 0 002-2V9a2 2 0 00-2-2zM16 7V5a2 2 0 00-2-2h-4a2 2 0 00-2 2v2',
    ledger:    'M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8zM14 2v6h6',
    investigation: 'M11 19a8 8 0 100-16 8 8 0 000 16zM21 21l-4.35-4.35',
    timeline:  'M12 22V2M2 12h20M4.93 4.93l14.14 14.14M19.07 4.93L4.93 19.07',
    admin:     'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75',
    operator:  'M2 4h20v16H2zM8 10l3 2-3 2M13 15h3',
    shortcuts: 'M3 3h18v18H3zM9 9h.01M15 9h.01M9 15h.01M15 15h.01',
    ssh:       'M3 11h18v11H3zM7 11V7a5 5 0 0110 0v4',
    secrets:   'M21 2l-2 2m-7.61 7.61a5.5 5.5 0 11-7.778 7.778 5.5 5.5 0 017.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4',
    suppression:'M18 6L6 18M6 6l12 12',
  };

  // ── Nav sections ───────────────────────────────────────────────────
  const observe: NavItem[] = [
    { id: 'dashboard',          icon: 'dashboard',  label: 'Dashboard' },
    { id: 'siem',               icon: 'siem',       label: 'SIEM' },
    { id: 'siem-search',        icon: 'hunter',     label: 'SIEM Search',     context: 'browser' },
    { id: 'alerts',             icon: 'alerts',     label: 'Alerts' },
    { id: 'alert-management',   icon: 'alerts',     label: 'Alert Management',context: 'browser' },
    { id: 'recordings',         icon: 'recordings', label: 'Recordings',      context: 'desktop' },
    { id: 'topology',           icon: 'topology',   label: 'Topology' },
    { id: 'mitre-heatmap',      icon: 'mitre',      label: 'MITRE Heatmap' },
    { id: 'threat-map',         icon: 'topology',   label: 'Threat Map' },
    { id: 'health',             icon: 'health',     label: 'Health Monitor' },
  ];

  const operate: NavItem[] = [
    { id: 'terminal',           icon: 'terminal',   label: 'Terminal' },
    { id: 'ssh',                icon: 'ssh',        label: 'SSH Bookmarks',   context: 'desktop' },
    { id: 'tunnels',            icon: 'tunnels',    label: 'Tunnels',         context: 'desktop' },
    { id: 'hosts',              icon: 'hosts',      label: 'Hosts' },
    { id: 'agents',             icon: 'agents',     label: 'Agent Console',   context: 'browser' },
    { id: 'fleet-management',   icon: 'agents',     label: 'Fleet',           context: 'browser' },
    { id: 'ops',                icon: 'ops',        label: 'Ops Center' },
    { id: 'soc',                icon: 'soc',        label: 'SOC',             context: 'browser' },
    { id: 'escalation',         icon: 'response',   label: 'Escalation',      context: 'browser' },
    { id: 'playbook-builder',   icon: 'purple',     label: 'Playbook Builder',context: 'browser' },
    { id: 'snippets',           icon: 'snippets',   label: 'Snippets',        context: 'desktop' },
    { id: 'notes',              icon: 'notes',      label: 'Notes',           context: 'desktop' },
    { id: 'ai-assistant',       icon: 'ai',         label: 'AI Shell' },
    { id: 'cases',              icon: 'cases',      label: 'Cases' },
    { id: 'ledger',             icon: 'ledger',     label: 'Evidence Ledger' },
    { id: 'timeline',           icon: 'timeline',   label: 'Timeline' },
    { id: 'response',           icon: 'response',   label: 'SOAR / Response' },
    { id: 'operator',           icon: 'operator',   label: 'Operator Mode',   context: 'desktop' },
    { id: 'investigation',      icon: 'investigation', label: 'Investigation' },
  ];

  const intel: NavItem[] = [
    { id: 'ueba',               icon: 'ueba',       label: 'UEBA' },
    { id: 'threat-hunter',      icon: 'hunter',     label: 'Threat Hunter' },
    { id: 'threat-intel-dashboard', icon: 'security', label: 'Threat Intel',  context: 'browser' },
    { id: 'enrichment',         icon: 'forensics',  label: 'Enrichment',      context: 'browser' },
    { id: 'ndr',                icon: 'ndr',        label: 'NDR' },
    { id: 'purple-team',        icon: 'purple',     label: 'Purple Team' },
    { id: 'graph',              icon: 'graph',      label: 'Threat Graph' },
  ];

  const govern: NavItem[] = [
    { id: 'compliance',         icon: 'compliance', label: 'Compliance' },
    { id: 'vault',              icon: 'vault',      label: 'Vault' },
    { id: 'identity',           icon: 'identity',   label: 'Identity Admin',  context: 'browser' },
    { id: 'security',           icon: 'security',   label: 'Runtime Trust' },
    { id: 'forensics',          icon: 'forensics',  label: 'Forensics' },
    { id: 'ransomware',         icon: 'war',        label: 'Ransomware' },
    { id: 'war-mode',           icon: 'war',        label: 'War Mode' },
    { id: 'secrets',            icon: 'secrets',    label: 'Secrets' },
    { id: 'suppression',        icon: 'suppression',label: 'Suppression' },
    { id: 'admin',              icon: 'admin',      label: 'Super Admin' },
  ];

  const systemItems: NavItem[] = [
    { id: 'executive',          icon: 'executive',  label: 'Executive Dashboard' },
    { id: 'plugins',            icon: 'plugins',    label: 'Plugins' },
    { id: 'sync',               icon: 'sync',       label: 'Sync',            context: 'desktop' },
    { id: 'license',            icon: 'vault',      label: 'License' },
    { id: 'settings',           icon: 'settings',   label: 'Settings' },
    { id: 'shortcuts',          icon: 'shortcuts',  label: 'Keyboard Shortcuts' },
  ];

  const auditItems = [
    { id: 'temporal',          label: 'Temporal Integrity' },
    { id: 'lineage',           label: 'Data Lineage' },
    { id: 'decisions',         label: 'Decision Log' },
    { id: 'ledger',            label: 'Evidence Ledger' },
    { id: 'replay',            label: 'Response Replay' },
    { id: 'chain-of-custody',  label: 'Chain of Custody' },
  ];

  let auditOpen = $state(false);
  let flyoutTop = $state(0);
  const auditActive = $derived(auditItems.some(i => appStore.activeNavTab === i.id));

  function handleAuditEnter(e: MouseEvent) {
    flyoutTop = (e.currentTarget as HTMLElement).getBoundingClientRect().top;
    auditOpen = true;
  }

  function iconPath(name: string) {
    return icons[name] ?? icons.dashboard;
  }
</script>

<style>
  /* 44px wide — icon only, tooltip via title attr */
  .cr-rail {
    width: 44px;
    min-width: 44px;
    height: 100%;
    background: var(--s1);
    border-right: 1px solid var(--b1);
    display: flex;
    flex-direction: column;
    z-index: 1000;
    overflow-y: auto;
    overflow-x: hidden;
    flex-shrink: 0;
  }
  .cr-rail::-webkit-scrollbar { width: 0; }

  .cr-logo {
    height: 44px;
    min-height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-bottom: 1px solid var(--b1);
    flex-shrink: 0;
  }
  .cr-logo-text {
    font-family: var(--mn);
    font-size: 9px;
    font-weight: 700;
    color: #d0e8f8;
    letter-spacing: 0.12em;
  }
  .cr-logo-em { color: var(--cr2); }

  .cr-btn {
    position: relative;
    width: 44px;
    height: 36px;
    min-height: 36px;
    background: transparent;
    border: none;
    color: var(--tx3);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 100ms, background 100ms;
    flex-shrink: 0;
  }
  .cr-btn:hover:not(.locked) {
    color: var(--tx);
    background: rgba(24,120,200,0.04);
  }
  .cr-btn.active {
    color: var(--ac2) !important;
    background: rgba(24,120,200,0.1) !important;
  }
  .cr-btn.active::before {
    content: '';
    position: absolute;
    left: 0; top: 20%; bottom: 20%;
    width: 2px;
    background: var(--ac);
    border-radius: 0 1px 1px 0;
    box-shadow: 0 0 5px var(--ac);
  }
  .cr-btn.locked { opacity: 0.22; cursor: not-allowed; pointer-events: none; }

  .cr-icon {
    width: 16px; height: 16px;
    transition: opacity 100ms;
  }
  .cr-btn:not(.active) .cr-icon { opacity: 0.5; }
  .cr-btn.active .cr-icon        { opacity: 1; }

  .cr-divider {
    display: flex; align-items: center; justify-content: center;
    padding: 6px 0 2px;
  }
  .cr-div-line {
    width: 20px; height: 1px; background: var(--b1);
  }

  .cr-flyout {
    position: fixed;
    left: 46px;
    background: var(--s2);
    border: 1px solid var(--b2);
    border-radius: 2px;
    z-index: 9000;
    min-width: 192px;
    padding: 4px;
    box-shadow: 0 4px 16px rgba(0,0,0,0.45);
  }
  .cr-flyout-hd {
    font-family: var(--mn);
    font-size: 8px; font-weight: 800;
    color: var(--tx3);
    padding: 7px 10px 5px;
    letter-spacing: 0.15em;
    text-transform: uppercase;
    border-bottom: 1px solid var(--b1);
    margin-bottom: 3px;
  }
  .cr-flyout-item {
    display: block; width: 100%;
    background: transparent; border: none; border-radius: 1px;
    color: var(--tx2);
    padding: 6px 10px;
    text-align: left;
    font-family: var(--sn); font-size: 11px;
    cursor: pointer;
    transition: all 100ms;
    border-left: 2px solid transparent;
  }
  .cr-flyout-item:hover { background: var(--s3); color: var(--tx); }
  .cr-flyout-item.active {
    background: rgba(24,120,200,0.1);
    color: var(--ac2);
    border-left-color: var(--ac);
    font-weight: 600;
  }
</style>

<nav class="cr-rail" aria-label="Main navigation">

  <!-- Brand mark -->
  <div class="cr-logo">
    <span class="cr-logo-text">O<em class="cr-logo-em">I</em></span>
  </div>

  <!-- OBS -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  {#each visible(observe) as item}
    {@const available = routeMap[item.id] ? isRouteAvailable(routeMap[item.id]) : true}
    <button
      class="cr-btn"
      class:active={appStore.activeNavTab === item.id}
      class:locked={!available}
      title="{item.label}{!available ? ' — unavailable' : ''}"
      onclick={() => available && go(item.id)}
    >
      <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d={iconPath(item.icon)} />
      </svg>
    </button>
  {/each}

  <!-- OPS -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  {#each visible(operate) as item}
    {@const available = routeMap[item.id] ? isRouteAvailable(routeMap[item.id]) : true}
    <button class="cr-btn" class:active={appStore.activeNavTab === item.id} class:locked={!available}
      title="{item.label}{!available ? ' — unavailable' : ''}" onclick={() => available && go(item.id)}>
      <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d={iconPath(item.icon)} />
      </svg>
    </button>
  {/each}

  <!-- INTEL -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  {#each visible(intel) as item}
    {@const available = routeMap[item.id] ? isRouteAvailable(routeMap[item.id]) : true}
    <button class="cr-btn" class:active={appStore.activeNavTab === item.id} class:locked={!available}
      title="{item.label}{!available ? ' — unavailable' : ''}" onclick={() => available && go(item.id)}>
      <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d={iconPath(item.icon)} />
      </svg>
    </button>
  {/each}

  <!-- GOV -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  {#each visible(govern) as item}
    {@const available = routeMap[item.id] ? isRouteAvailable(routeMap[item.id]) : true}
    <button class="cr-btn" class:active={appStore.activeNavTab === item.id} class:locked={!available}
      title="{item.label}{!available ? ' — unavailable' : ''}" onclick={() => available && go(item.id)}>
      <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d={iconPath(item.icon)} />
      </svg>
    </button>
  {/each}

  <!-- AUDIT flyout -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  <button
    class="cr-btn"
    class:active={auditActive}
    title="Audit Trail"
    onmouseenter={handleAuditEnter}
    onmouseleave={() => auditOpen = false}
  >
    <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
      <path d={iconPath('audit')} />
    </svg>
  </button>

  {#if auditOpen}
    <div
      class="cr-flyout"
      style="top:{flyoutTop}px;"
      role="menu"
      tabindex="-1"
      onmouseenter={() => auditOpen = true}
      onmouseleave={() => auditOpen = false}
    >
      <div class="cr-flyout-hd">AUDIT TRAIL</div>
      {#each auditItems as item}
        <button
          class="cr-flyout-item"
          class:active={appStore.activeNavTab === item.id}
          onclick={() => { go(item.id); auditOpen = false; }}
        >{item.label}</button>
      {/each}
    </div>
  {/if}

  <!-- Spacer pushes SYS to bottom -->
  <div class="flex-1 min-h-2"></div>

  <!-- SYS -->
  <div class="cr-divider"><div class="cr-div-line"></div></div>
  {#each visible(systemItems) as item}
    {@const available = routeMap[item.id] ? isRouteAvailable(routeMap[item.id]) : true}
    <button class="cr-btn" class:active={appStore.activeNavTab === item.id} class:locked={!available}
      title="{item.label}{!available ? ' — unavailable' : ''}" onclick={() => available && go(item.id)}>
      <svg class="cr-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d={iconPath(item.icon)} />
      </svg>
    </button>
  {/each}

  <div class="h-3"></div>
</nav>
