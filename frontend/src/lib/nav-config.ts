/**
 * nav-config.ts — central definition of navigation groups and their items.
 *
 * Single source of truth shared by:
 *   - `AppSidebar.svelte`   (renders the GROUP buttons on the left rail)
 *   - `BottomDock.svelte`   (renders the ITEMS belonging to the active group)
 *   - `CommandPalette`      (flat search across every item — same labels)
 *
 * The `id` field matches `NavTab` ids in `lib/types.ts` so that
 * `appStore.setActiveNavTab(id)` continues to highlight the legacy
 * `CommandRail` while the new chrome is opt-in. Routes resolve via the
 * existing `routeMap` defined at the call sites.
 */

import type { NavGroupId } from './stores/navigation.svelte';

export type NavContext = 'both' | 'desktop' | 'browser';

export interface NavItem {
  /** NavTab id — must match `lib/types.ts` NavTab union. */
  id: string;
  /** Route path. Authoritative — call sites use this directly with `push`. */
  route: string;
  /** Human-readable label. */
  label: string;
  /** Lucide icon name (used by AppSidebar/BottomDock to look up `lucide-svelte` imports). */
  icon: string;
  /** Where this item is allowed to render (browser-only, desktop-only, or both). */
  context?: NavContext;
  /** Optional badge count source — pulled live from a store at render time. */
  badgeKey?: 'alerts' | 'incidents' | 'agents';
  /** Optional one-line description shown beneath the label in the dock. */
  description?: string;
}

export interface NavGroup {
  id: NavGroupId;
  label: string;
  /** Lucide icon name for the sidebar. */
  icon: string;
  /** One-line subtitle — used in tooltips + dock header. */
  subtitle: string;
  items: NavItem[];
}

// ── Groups ───────────────────────────────────────────────────────────
//
// Carved up by operator workflow, NOT by feature taxonomy. The bottom
// dock is the operator's "what can I do RIGHT NOW in this context"
// surface, so co-locate items the operator reaches for in the same
// breath. Cross-references:
//
//   Operate     → live actions: terminal, dashboards, ongoing incidents
//   Detect      → things that watch: SIEM, alerts, intel, MITRE
//   Investigate → things that explain: graphs, timelines, evidence, lineage
//   Defend      → things that block: SOAR, ransomware, NDR, simulations
//   Govern      → things that prove: compliance, identity, audit, vault
//   System      → things that configure: fleet, plugins, settings
//
// IDs MUST match the NavTab union in lib/types.ts. routes MUST exist in
// the Router's path map.

export const NAV_GROUPS: NavGroup[] = [
  {
    id: 'operate',
    label: 'Operate',
    icon: 'Activity',
    subtitle: 'Live operations & response',
    items: [
      { id: 'dashboard',    route: '/dashboard',    label: 'Dashboard',         icon: 'LayoutDashboard', description: 'Operator overview' },
      { id: 'terminal',     route: '/terminal',     label: 'Terminal',          icon: 'Terminal',        description: 'PTY sessions', context: 'desktop' },
      { id: 'operator',     route: '/operator',     label: 'Operator Mode',     icon: 'Crosshair',       description: 'SIEM-aware shell context', context: 'desktop' },
      { id: 'response',     route: '/response',     label: 'SOAR / Response',   icon: 'Zap',             description: 'Run playbooks' },
      { id: 'cases',        route: '/cases',        label: 'Cases',             icon: 'FolderOpen',      description: 'Incident folders' },
      { id: 'ops',          route: '/ops',          label: 'Ops Center',        icon: 'Monitor',         description: 'War-room view' },
      { id: 'soc',          route: '/soc',          label: 'SOC View',          icon: 'Eye',             description: 'Multi-tenant SOC', context: 'browser' },
      { id: 'ssh',          route: '/ssh',          label: 'SSH Bookmarks',     icon: 'Server',          description: 'Vaulted hosts', context: 'desktop' },
      { id: 'tunnels',      route: '/tunnels',      label: 'Tunnels',           icon: 'Cable',           description: 'Port forwards', context: 'desktop' },
      { id: 'snippets',     route: '/snippets',     label: 'Snippets',          icon: 'Code',            description: 'Saved commands', context: 'desktop' },
      { id: 'notes',        route: '/notes',        label: 'Notes',             icon: 'StickyNote',      description: 'Per-host notes', context: 'desktop' },
      { id: 'recordings',   route: '/recordings',   label: 'Recordings',        icon: 'Video',           description: 'Session replay', context: 'desktop' },
    ],
  },
  {
    id: 'detect',
    label: 'Detect',
    icon: 'Radar',
    subtitle: 'Visibility & detection',
    items: [
      { id: 'siem',                    route: '/siem',                    label: 'SIEM',               icon: 'Database',     description: 'Event store' },
      { id: 'siem-search',             route: '/siem-search',             label: 'SIEM Search',        icon: 'Search',       description: 'Lucene + OQL', context: 'browser' },
      { id: 'alerts',                  route: '/alerts',                  label: 'Alerts',             icon: 'Bell',         description: 'Active triggers', badgeKey: 'alerts' },
      { id: 'alert-management',        route: '/alert-management',        label: 'Alert Management',   icon: 'BellRing',     description: 'Triage queue', context: 'browser' },
      { id: 'threat-hunter',           route: '/threat-hunter',           label: 'Threat Hunter',      icon: 'Telescope',    description: 'Hypothesis-driven hunts' },
      { id: 'threat-intel-dashboard',  route: '/threat-intel-dashboard',  label: 'Threat Intel',       icon: 'Globe',        description: 'TI feeds & matches', context: 'browser' },
      { id: 'enrichment',              route: '/enrichment',              label: 'Enrichment',         icon: 'Sparkles',      description: 'GeoIP, ASN, TI', context: 'browser' },
      { id: 'mitre-heatmap',           route: '/mitre-heatmap',           label: 'MITRE Heatmap',      icon: 'Grid3x3',      description: 'ATT&CK coverage' },
      { id: 'topology',                route: '/topology',                label: 'Topology',           icon: 'Network',      description: 'Asset map' },
      { id: 'threat-map',              route: '/threat-map',              label: 'Threat Map',         icon: 'Map',          description: 'Geo overlay' },
      { id: 'health',                  route: '/monitoring',              label: 'Health Monitor',     icon: 'HeartPulse',   description: 'Pipeline health' },
      { id: 'ueba',                    route: '/ueba',                    label: 'UEBA',               icon: 'Users',        description: 'Behavioural anomalies' },
    ],
  },
  {
    id: 'investigate',
    label: 'Investigate',
    icon: 'Search',
    subtitle: 'Reconstruct & explain',
    items: [
      { id: 'graph',         route: '/graph',           label: 'Threat Graph',       icon: 'GitBranch',  description: 'User → Host → Process' },
      { id: 'timeline',      route: '/timeline',        label: 'Timeline',           icon: 'History',    description: 'Causal reconstruction' },
      { id: 'investigation', route: '/investigation',   label: 'Investigations',     icon: 'FileSearch', description: 'Active probes' },
      { id: 'forensics',     route: '/forensics',       label: 'Forensics',          icon: 'Microscope', description: 'Evidence analysis' },
      { id: 'remote-forensics', route: '/remote-forensics', label: 'Remote Forensics', icon: 'HardDrive', description: 'Live agent capture' },
      { id: 'ledger',        route: '/ledger',          label: 'Evidence Ledger',    icon: 'BookLock',   description: 'WORM evidence chain' },
      { id: 'chain-of-custody', route: '/chain-of-custody', label: 'Chain of Custody', icon: 'Link',     description: 'Custody handoffs' },
      { id: 'lineage',       route: '/lineage',         label: 'Data Lineage',       icon: 'Workflow',   description: 'Where did this come from?' },
      { id: 'temporal',      route: '/temporal-integrity', label: 'Temporal Integrity', icon: 'Clock4',  description: 'Late-event audit' },
      { id: 'replay',        route: '/response-replay', label: 'Response Replay',    icon: 'Rewind',     description: 'Re-run incident' },
      { id: 'decisions',     route: '/decisions',       label: 'Decision Log',       icon: 'GitCompare', description: 'Why did SOAR do that?' },
    ],
  },
  {
    id: 'defend',
    label: 'Defend',
    icon: 'Shield',
    subtitle: 'Block & contain',
    items: [
      { id: 'playbook-builder', route: '/playbook-builder', label: 'Playbook Builder', icon: 'BookOpen', description: 'Edit SOAR flows', context: 'browser' },
      { id: 'ransomware',    route: '/ransomware',     label: 'Ransomware Defense', icon: 'ShieldAlert', description: 'Live ransomware shield' },
      { id: 'ransomware-ui', route: '/ransomware-ui',  label: 'Ransomware Console', icon: 'Skull',     description: 'Decoys & triggers' },
      { id: 'ndr',           route: '/ndr',            label: 'NDR',                icon: 'Wifi',       description: 'Network anomalies' },
      { id: 'ndr-overview',  route: '/ndr-overview',   label: 'NDR Overview',       icon: 'Wifi',       description: 'NDR rollup' },
      { id: 'purple-team',   route: '/purple-team',    label: 'Purple Team',        icon: 'Swords',     description: 'Sim & validate' },
      { id: 'simulation',    route: '/simulation',     label: 'Simulation',         icon: 'Play',       description: 'Adversary emulation' },
      { id: 'war-mode',      route: '/war-mode',       label: 'War Mode',           icon: 'Siren',      description: 'Active-incident lockdown' },
      { id: 'escalation',    route: '/escalation',     label: 'Escalation',         icon: 'Flame',      description: 'On-call routing', context: 'browser' },
    ],
  },
  {
    id: 'govern',
    label: 'Govern',
    icon: 'Scale',
    subtitle: 'Trust & compliance',
    items: [
      { id: 'compliance',   route: '/compliance',     label: 'Compliance',         icon: 'ClipboardCheck', description: 'SOC2, HIPAA, ISO' },
      { id: 'vault',        route: '/vault',          label: 'Vault',              icon: 'KeyRound',     description: 'Encrypted secrets' },
      { id: 'secrets',      route: '/secrets',        label: 'Secrets',            icon: 'Lock',         description: 'Rotation policies' },
      { id: 'identity',     route: '/identity',       label: 'Identity Admin',     icon: 'UserCog',      description: 'Users & roles', context: 'browser' },
      { id: 'identity-admin', route: '/identity-admin', label: 'Identity (Tenant)', icon: 'UsersRound', description: 'Per-tenant identity', context: 'browser' },
      { id: 'security',     route: '/trust',          label: 'Runtime Trust',      icon: 'ShieldCheck',  description: 'Trust scores' },
      { id: 'suppression',  route: '/suppression',    label: 'Suppression',        icon: 'EyeOff',       description: 'False-positive rules' },
      { id: 'admin',        route: '/admin',          label: 'Super Admin',        icon: 'ShieldHalf',   description: 'Tenant lifecycle' },
      { id: 'license',      route: '/license',        label: 'License',            icon: 'Award',        description: 'Feature flags' },
    ],
  },
  {
    id: 'system',
    label: 'System',
    icon: 'Settings',
    subtitle: 'Configure & extend',
    items: [
      { id: 'hosts',             route: '/hosts',             label: 'Hosts',             icon: 'Server',       description: 'Asset inventory' },
      { id: 'agents',            route: '/agents',            label: 'Agents',            icon: 'Cpu',          description: 'Agent fleet', context: 'browser', badgeKey: 'agents' },
      { id: 'fleet-management',  route: '/fleet-management',  label: 'Fleet',             icon: 'Boxes',        description: 'Multi-tenant fleet', context: 'browser' },
      { id: 'plugins',           route: '/plugins',           label: 'Plugins',           icon: 'Puzzle',       description: 'Loaded extensions' },
      { id: 'sync',              route: '/sync',              label: 'Sync',              icon: 'RefreshCw',    description: 'Cross-instance sync', context: 'desktop' },
      { id: 'executive',         route: '/executive',         label: 'Executive Dashboard', icon: 'TrendingUp', description: 'KPIs for leadership' },
      { id: 'ai-assistant',      route: '/ai-assistant',      label: 'AI Shell',          icon: 'Bot',          description: 'Ask the assistant' },
      { id: 'settings',          route: '/workspace',         label: 'Settings',          icon: 'Settings',     description: 'App preferences' },
      { id: 'shortcuts',         route: '/shortcuts',         label: 'Keyboard Shortcuts', icon: 'Keyboard',    description: 'Hotkey reference' },
    ],
  },
];

/** Lookup the group definition for a NavGroupId. */
export function getGroup(id: NavGroupId): NavGroup | undefined {
  return NAV_GROUPS.find((g) => g.id === id);
}

/** Find which group a route id belongs to (first match). */
export function findGroupForRoute(routeId: string): NavGroupId | null {
  for (const g of NAV_GROUPS) {
    if (g.items.some((it) => it.id === routeId)) return g.id;
  }
  return null;
}

/** Flat list of every item — used by the command palette + the pinned strip. */
export function allItems(): NavItem[] {
  return NAV_GROUPS.flatMap((g) => g.items);
}
