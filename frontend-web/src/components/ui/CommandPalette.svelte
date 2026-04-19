<!--
  OBLIVRA — CommandPalette (Svelte 5)
  Global command palette (Cmd+K) for navigation, host discovery, and actions.
  Covers all 60+ routes. Fixes prop interface: uses `open` (bindable) not onClose.
-->
<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { appStore } from '@lib/stores/app.svelte.ts';
  import { push } from '@lib/router.svelte';

  import type { PaletteAction } from '@lib/stores/app.svelte.ts';

  interface Props {
    open: boolean;
    onClose?: () => void;
    actions?: PaletteAction[];
  }

  let { open = $bindable(), actions = [] }: Props = $props();

  let query = $state('');
  let selectedIndex = $state(0);
  let inputRef = $state<HTMLInputElement>();

  // ── Full route index (mirrors CommandRail routeMap) ────────────────
  const ALL_ROUTES: { id: string; label: string; sublabel: string; path: string; section: string }[] = [
    // Observe
    { id: 'dashboard',            label: 'Dashboard',             sublabel: 'Command hub & platform status',          path: '/dashboard',            section: 'OBS' },
    { id: 'siem',                 label: 'SIEM',                  sublabel: 'Security information & event mgmt',      path: '/siem',                 section: 'OBS' },
    { id: 'siem-search',          label: 'SIEM Search',           sublabel: 'Query log streams',                      path: '/siem-search',          section: 'OBS' },
    { id: 'alerts',               label: 'Alerts',                sublabel: 'Real-time incident detection',           path: '/alerts',               section: 'OBS' },
    { id: 'alert-management',     label: 'Alert Management',      sublabel: 'Triage and bulk-resolve alerts',         path: '/alert-management',     section: 'OBS' },
    { id: 'topology',             label: 'Topology',              sublabel: 'Network topology view',                  path: '/topology',             section: 'OBS' },
    { id: 'mitre-heatmap',        label: 'MITRE Heatmap',         sublabel: 'ATT&CK technique coverage',              path: '/mitre-heatmap',        section: 'OBS' },
    { id: 'threat-map',           label: 'Threat Map',            sublabel: 'Global threat geo-visualization',        path: '/threat-map',           section: 'OBS' },
    { id: 'health',               label: 'Health Monitor',        sublabel: 'Platform health & metrics',              path: '/monitoring',           section: 'OBS' },
    { id: 'recordings',           label: 'Recordings',            sublabel: 'Session recording library',              path: '/recordings',           section: 'OBS' },
    // Operate
    { id: 'terminal',             label: 'Terminal',              sublabel: 'Active shell sessions',                  path: '/terminal',             section: 'OPS' },
    { id: 'ssh',                  label: 'SSH Bookmarks',         sublabel: 'Saved SSH connections',                  path: '/ssh',                  section: 'OPS' },
    { id: 'tunnels',              label: 'Tunnels',               sublabel: 'Port-forward & reverse tunnels',         path: '/tunnels',              section: 'OPS' },
    { id: 'hosts',                label: 'Hosts',                 sublabel: 'Infrastructure management',              path: '/hosts',                section: 'OPS' },
    { id: 'fleet-management',     label: 'Fleet',                 sublabel: 'Fleet dashboard & agent status',         path: '/fleet-management',     section: 'OPS' },
    { id: 'agents',               label: 'Agents',                sublabel: 'Agent console',                          path: '/agents',               section: 'OPS' },
    { id: 'ops',                  label: 'Ops Center',            sublabel: 'Operational command board',              path: '/ops',                  section: 'OPS' },
    { id: 'soc',                  label: 'SOC',                   sublabel: 'Security operations center',             path: '/soc',                  section: 'OPS' },
    { id: 'escalation',           label: 'Escalation Center',     sublabel: 'Manage incident escalations',            path: '/escalation',           section: 'OPS' },
    { id: 'playbook-builder',     label: 'Playbook Builder',      sublabel: 'Automate response playbooks',            path: '/playbook-builder',     section: 'OPS' },
    { id: 'snippets',             label: 'Snippets',              sublabel: 'Command snippet library',                path: '/snippets',             section: 'OPS' },
    { id: 'notes',                label: 'Notes',                 sublabel: 'Analyst notes & scratch pad',            path: '/notes',                section: 'OPS' },
    { id: 'ai-assistant',         label: 'AI Assistant',          sublabel: 'AI-powered shell & query assist',        path: '/ai-assistant',         section: 'OPS' },
    { id: 'cases',                label: 'Case Management',       sublabel: 'Investigation case tracking',            path: '/cases',                section: 'OPS' },
    { id: 'ledger',               label: 'Evidence Ledger',       sublabel: 'Immutable evidence log',                 path: '/ledger',               section: 'OPS' },
    { id: 'response',             label: 'SOAR / Response',       sublabel: 'Security orchestration & automation',    path: '/response',             section: 'OPS' },
    { id: 'session-playback',     label: 'Session Playback',      sublabel: 'Replay recorded terminal sessions',      path: '/session-playback',     section: 'OPS' },
    // Intel
    { id: 'ueba',                 label: 'UEBA',                  sublabel: 'User & entity behaviour analytics',      path: '/ueba',                 section: 'INTEL' },
    { id: 'threat-hunter',        label: 'Threat Hunter',         sublabel: 'Proactive threat hunting',               path: '/threat-hunter',        section: 'INTEL' },
    { id: 'threat-intel',         label: 'Threat Intel',          sublabel: 'Threat intelligence dashboard',          path: '/threat-intel',         section: 'INTEL' },
    { id: 'enrichment',           label: 'Enrichment Viewer',     sublabel: 'IOC enrichment & context',               path: '/enrichment',           section: 'INTEL' },
    { id: 'ndr',                  label: 'NDR',                   sublabel: 'Network detection & response',           path: '/ndr',                  section: 'INTEL' },
    { id: 'purple-team',          label: 'Purple Team',           sublabel: 'Adversary simulation & red/blue',        path: '/purple-team',          section: 'INTEL' },
    { id: 'threat-graph',         label: 'Threat Graph',          sublabel: 'Entity relationship graph',              path: '/threat-graph',         section: 'INTEL' },
    { id: 'credentials',          label: 'Credential Intel',      sublabel: 'Exposed credential monitoring',          path: '/credentials',          section: 'INTEL' },
    { id: 'global-topology',      label: 'Global Topology',       sublabel: 'Cross-site network map',                 path: '/global-topology',      section: 'INTEL' },
    { id: 'network-map',          label: 'Network Map',           sublabel: 'Internal network layout',                path: '/network-map',          section: 'INTEL' },
    // Govern
    { id: 'compliance',           label: 'Compliance',            sublabel: 'Policy & regulatory posture',            path: '/compliance',           section: 'GOV' },
    { id: 'vault',                label: 'Vault',                 sublabel: 'Credential & secret management',         path: '/vault',                section: 'GOV' },
    { id: 'identity',             label: 'Identity Admin',        sublabel: 'User identity & access control',         path: '/identity',             section: 'GOV' },
    { id: 'security',             label: 'Runtime Trust',         sublabel: 'Runtime integrity & trust state',        path: '/trust',                section: 'GOV' },
    { id: 'forensics',            label: 'Forensics',             sublabel: 'Host forensic analysis',                 path: '/forensics',            section: 'GOV' },
    { id: 'terminal-forensics',   label: 'Terminal Forensics',    sublabel: 'Terminal session forensics',             path: '/terminal-forensics',   section: 'GOV' },
    { id: 'ransomware',           label: 'Ransomware Response',   sublabel: 'Ransomware detection & recovery',        path: '/ransomware',           section: 'GOV' },
    { id: 'war-mode',             label: 'War Mode',              sublabel: 'Emergency platform lockdown',            path: '/war-mode',             section: 'GOV' },
    { id: 'data-destruction',     label: 'Data Destruction',      sublabel: 'Secure wipe orchestration',              path: '/data-destruction',     section: 'GOV' },
    // Audit
    { id: 'temporal',             label: 'Temporal Integrity',    sublabel: 'Timestamp & log integrity',              path: '/temporal-integrity',   section: 'AUDIT' },
    { id: 'lineage',              label: 'Data Lineage',          sublabel: 'Data provenance explorer',               path: '/lineage',              section: 'AUDIT' },
    { id: 'decisions',            label: 'Decision Inspector',    sublabel: 'AI decision audit log',                  path: '/decisions',            section: 'AUDIT' },
    { id: 'replay',               label: 'Response Replay',       sublabel: 'Replay incident response',               path: '/response-replay',      section: 'AUDIT' },
    { id: 'chain-of-custody',     label: 'Chain of Custody',      sublabel: 'Evidence chain of custody',              path: '/chain-of-custody',     section: 'AUDIT' },
    { id: 'oql',                  label: 'OQL Dashboard',         sublabel: 'OBLIVRA query language',                 path: '/oql',                  section: 'AUDIT' },
    { id: 'soar-panel',           label: 'SOAR Panel',            sublabel: 'Automated response panel',               path: '/soar',                 section: 'AUDIT' },
    { id: 'simulation',           label: 'Simulation',            sublabel: 'Attack simulation & testing',            path: '/simulation',           section: 'AUDIT' },
    // System
    { id: 'executive',            label: 'Executive Dashboard',   sublabel: 'C-suite risk & metrics view',            path: '/executive',            section: 'SYS' },
    { id: 'team',                 label: 'Team Dashboard',        sublabel: 'Team collaboration & ops',               path: '/team',                 section: 'SYS' },
    { id: 'plugins',              label: 'Plugin Manager',        sublabel: 'Manage OBLIVRA extensions',              path: '/plugins',              section: 'SYS' },
    { id: 'sync',                 label: 'Sync',                  sublabel: 'Config & state synchronisation',         path: '/sync',                 section: 'SYS' },
    { id: 'offline-update',       label: 'Offline Update',        sublabel: 'Air-gap software update',                path: '/offline-update',       section: 'SYS' },
    { id: 'settings',             label: 'Settings',              sublabel: 'Platform configuration',                 path: '/settings',             section: 'SYS' },
    { id: 'license',              label: 'License',               sublabel: 'License & entitlement management',       path: '/license',              section: 'SYS' },
    { id: 'risk',                 label: 'Config Risk',           sublabel: 'Configuration risk posture',             path: '/risk',                 section: 'SYS' },
    { id: 'entity',               label: 'Entity View',           sublabel: 'Entity context & timeline',              path: '/entity',               section: 'SYS' },
    { id: 'fusion',               label: 'Fusion Dashboard',      sublabel: 'Cross-source intelligence fusion',       path: '/fusion',               section: 'SYS' },
  ];

  type ResultItem =
    | { kind: 'route'; id: string; label: string; sublabel: string; path: string; section: string }
    | { kind: 'host';  id: string; label: string; sublabel: string }
    | { kind: 'action'; id: string; label: string; sublabel: string; action: () => void; icon?: string; shortcut?: string };

  const results = $derived.by((): ResultItem[] => {
    const q = query.toLowerCase().trim();
    if (!q) return [];
    const out: ResultItem[] = [];

    // Custom actions first
    for (const a of actions) {
      if (a.label.toLowerCase().includes(q) || a.description?.toLowerCase().includes(q)) {
        out.push({ kind: 'action', id: a.id, label: a.label, sublabel: a.description ?? '', action: a.action, icon: a.icon, shortcut: a.shortcut });
      }
    }

    for (const r of ALL_ROUTES) {
      if (
        r.label.toLowerCase().includes(q) ||
        r.sublabel.toLowerCase().includes(q) ||
        r.section.toLowerCase().includes(q)
      ) {
        out.push({ kind: 'route', ...r });
      }
    }

    for (const h of appStore.hosts) {
      if (
        h.label.toLowerCase().includes(q) ||
        (h as any).hostname?.toLowerCase().includes(q)
      ) {
        out.push({ kind: 'host', id: h.id, label: h.label, sublabel: (h as any).hostname ?? '' });
      }
    }

    return out.slice(0, 12);
  });

  $effect(() => {
    if (open) {
      query = '';
      selectedIndex = 0;
      setTimeout(() => inputRef?.focus(), 10);
    }
  });

  $effect(() => {
    // Keep selectedIndex in bounds when results change
    if (selectedIndex >= results.length) selectedIndex = 0;
  });

  function close() { open = false; }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') { close(); }
    else if (e.key === 'ArrowDown') { e.preventDefault(); selectedIndex = (selectedIndex + 1) % Math.max(1, results.length); }
    else if (e.key === 'ArrowUp')   { e.preventDefault(); selectedIndex = (selectedIndex - 1 + Math.max(1, results.length)) % Math.max(1, results.length); }
    else if (e.key === 'Enter')     { e.preventDefault(); if (results[selectedIndex]) execute(results[selectedIndex]); }
  }

  function execute(item: ResultItem) {
    if (item.kind === 'route') {
      appStore.setActiveNavTab(item.id as any);
      push(item.path);
    } else if (item.kind === 'host') {
      appStore.connectToHost(item.id);
    } else if (item.kind === 'action') {
      item.action();
    }
    close();
    query = '';
  }

  // Quick-nav items shown when no query typed
  const quickNav = [
    { label: 'Dashboard',     path: '/dashboard',    id: 'dashboard',    section: 'OBS'   },
    { label: 'Alerts',        path: '/alerts',       id: 'alerts',       section: 'OBS'   },
    { label: 'Terminal',      path: '/terminal',     id: 'terminal',     section: 'OPS'   },
    { label: 'SIEM',          path: '/siem',         id: 'siem',         section: 'OBS'   },
    { label: 'Threat Hunter', path: '/threat-hunter',id: 'threat-hunter',section: 'INTEL' },
    { label: 'War Mode',      path: '/war-mode',     id: 'war-mode',     section: 'GOV'   },
  ];
</script>

{#if open}
  <div
    class="fixed inset-0 z-[10000] flex items-start justify-center pt-[15vh] px-4"
    transition:fade={{ duration: 120 }}
    role="button"
    tabindex="-1"
    onclick={close}
    onkeydown={(e) => e.key === 'Escape' && close()}
  >
    <div class="fixed inset-0 bg-black/60 backdrop-blur-sm"></div>

    <div
      class="relative w-full max-w-2xl bg-surface-2 border border-border-secondary rounded-xl shadow-2xl overflow-hidden"
      transition:fly={{ y: -16, duration: 180 }}
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
    >
      <!-- Search input -->
      <div
        class="flex items-center gap-3 px-4 py-3 border-b border-border-primary bg-surface-1"
        role="combobox"
        aria-haspopup="listbox"
        aria-expanded={results.length > 0}
        aria-controls="palette-results"
      >
        <span class="text-accent text-lg font-bold select-none" aria-hidden="true">⌕</span>
        <input
          bind:this={inputRef}
          type="text"
          bind:value={query}
          onkeydown={handleKeydown}
          placeholder="Search pages, hosts, or actions..."
          class="flex-1 bg-transparent border-none outline-none text-text-primary text-sm font-[var(--font-ui)] placeholder:text-text-muted"
          role="searchbox"
          aria-autocomplete="list"
          aria-controls="palette-results"
          aria-activedescendant={results.length > 0 ? `palette-item-${selectedIndex}` : undefined}
        />
        <div class="flex items-center gap-1.5" aria-hidden="true">
          <span class="text-[9px] font-mono text-text-muted bg-surface-3 px-1.5 py-0.5 rounded border border-border-primary">ESC</span>
        </div>
      </div>

      <!-- Results -->
      <div class="max-h-[420px] overflow-auto py-1" id="palette-results" role="listbox">
        {#if results.length > 0}
          {#each results as item, i}
            {@const isSelected = i === selectedIndex}
            <button
              id="palette-item-{i}"
              role="option"
              tabindex="-1"
              aria-selected={isSelected}
              class="w-full flex items-center justify-between px-4 py-2.5 text-left transition-colors duration-fast outline-hidden border-l-2
                {isSelected ? 'bg-accent/10 border-accent' : 'hover:bg-surface-3 border-transparent'}"
              onclick={() => execute(item)}
              onmouseenter={() => selectedIndex = i}
            >
              <div class="flex flex-col min-w-0">
                <span class="text-[12px] font-bold truncate {isSelected ? 'text-accent' : 'text-text-heading'}">
                  {item.label}
                </span>
                <span class="text-[10px] text-text-muted font-mono truncate">{item.sublabel}</span>
              </div>
              <div class="flex items-center gap-2 shrink-0 ml-3">
                {#if item.kind === 'route'}
                  <span class="text-[8px] font-mono font-bold uppercase tracking-wider text-text-muted opacity-40 px-1.5 py-0.5 border border-border-primary rounded bg-surface-2">
                    {item.section}
                  </span>
                {:else if item.kind === 'host'}
                  <span class="text-[8px] font-mono font-bold uppercase tracking-wider text-accent opacity-70 px-1.5 py-0.5 border border-accent/30 rounded bg-accent/5">
                    HOST
                  </span>
                {:else if item.kind === 'action'}
                   {#if item.shortcut}
                     <span class="text-[8px] font-mono font-bold text-text-muted opacity-40 px-1.5 py-0.5 border border-border-primary rounded bg-surface-2">
                       {item.shortcut}
                     </span>
                   {/if}
                   <span class="text-[12px] opacity-70">{item.icon || '⚡'}</span>
                {/if}
              </div>
            </button>
          {/each}
        {:else if query}
          <div class="px-6 py-8 text-center">
            <div class="text-text-muted text-xs">No matches for <span class="text-text-secondary font-mono">"{query}"</span></div>
          </div>
        {:else}
          <div class="px-4 py-3">
            <div class="text-[9px] font-bold uppercase tracking-widest text-text-muted opacity-40 mb-2 px-1">Quick Navigation</div>
            <div class="grid grid-cols-2 gap-0.5">
              {#each quickNav as nav}
                <button
                  class="flex items-center gap-2 px-3 py-2 rounded-sm hover:bg-surface-3 transition-colors text-left"
                  onclick={() => { appStore.setActiveNavTab(nav.id as any); push(nav.path); close(); }}
                >
                  <span class="text-accent text-[10px] shrink-0">→</span>
                  <span class="text-[11px] text-text-secondary truncate">{nav.label}</span>
                  <span class="text-[8px] font-mono text-text-muted opacity-40 ml-auto shrink-0">{nav.section}</span>
                </button>
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-4 py-2 border-t border-border-primary bg-surface-1 flex items-center justify-between text-[9px] font-mono text-text-muted opacity-50">
        <div class="flex items-center gap-3">
          <span><span class="text-text-secondary">↑↓</span> navigate</span>
          <span><span class="text-text-secondary">↵</span> select</span>
          <span><span class="text-text-secondary">ESC</span> close</span>
        </div>
        <span>{results.length > 0 ? `${results.length} result${results.length > 1 ? 's' : ''}` : `${ALL_ROUTES.length} routes indexed`}</span>
      </div>
    </div>
  </div>
{/if}
