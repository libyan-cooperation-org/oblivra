/**
 * nav-config.ts — central definition of the 7 SOC navigation domains.
 *
 * Phase 31 SOC redesign: investigation-first information architecture.
 * The 7 domains are organised around *what the operator is currently
 * looking at*, not feature taxonomy:
 *
 *   Overview   → mission control: risk, incidents, live activity
 *   Security   → alerts, incidents, campaigns, anomalies
 *   Network    → connections, traffic, DNS, geo
 *   Identity   → users, sessions, auth events, risky users
 *   Hosts      → fleet, activity, processes, host alerts
 *   Logs       → search, live tail, saved queries
 *   System     → agents, health, rules, integrations
 *
 * Switching a domain sets the active CONTEXT (active group in
 * navigationStore) and lands on the domain's primary route. The
 * BottomDock surfaces the domain's tools as cards.
 *
 * Single source of truth — both `AppSidebar.svelte` and
 * `BottomDock.svelte` read from this module.
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
  /** Lucide icon name (looked up by string in BottomDock's static map). */
  icon: string;
  /** Where this item is allowed to render. */
  context?: NavContext;
  /** Optional badge count source — pulled from a store at render time. */
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

// ── 7 SOC Domains ────────────────────────────────────────────────────

export const NAV_GROUPS: NavGroup[] = [
  {
    id: 'overview',
    label: 'Overview',
    icon: 'LayoutDashboard',
    subtitle: 'Mission control',
    items: [
      { id: 'overview',     route: '/overview',     label: 'Mission Control', icon: 'LayoutDashboard', description: 'Risk + incidents + live activity' },
      { id: 'dashboard',    route: '/dashboard',    label: 'Tactical Hub',    icon: 'Activity',        description: 'Operator overview' },
      { id: 'executive',    route: '/executive',    label: 'Executive View',  icon: 'TrendingUp',      description: 'KPIs for leadership' },
      { id: 'ops',          route: '/ops',          label: 'Ops Center',      icon: 'Monitor',         description: 'War-room layout' },
      { id: 'soc',          route: '/soc',          label: 'SOC Workspace',   icon: 'Eye',             description: 'Multi-analyst SOC', context: 'browser' },
    ],
  },
  {
    id: 'security',
    label: 'Security',
    icon: 'Shield',
    subtitle: 'Alerts, incidents, campaigns',
    items: [
      { id: 'alerts',                  route: '/alerts',                  label: 'Alerts',             icon: 'Bell',         description: 'Active triggers', badgeKey: 'alerts' },
      { id: 'alert-management',        route: '/alert-management',        label: 'Alert Management',   icon: 'BellRing',     description: 'Triage queue', context: 'browser' },
      { id: 'cases',                   route: '/cases',                   label: 'Incidents',          icon: 'FolderOpen',   description: 'Multi-alert cases' },
      { id: 'graph',                   route: '/graph',                   label: 'Campaigns',          icon: 'GitBranch',    description: 'Multi-step attack chains' },
      { id: 'ueba',                    route: '/ueba',                    label: 'Anomalies',          icon: 'Users',        description: 'UEBA deviations' },
      { id: 'threat-hunter',           route: '/threat-hunter',           label: 'Threat Hunter',      icon: 'Telescope',    description: 'Hypothesis-driven hunts' },
      { id: 'threat-intel-dashboard',  route: '/threat-intel-dashboard',  label: 'Threat Intel',       icon: 'Globe',        description: 'TI feeds & matches', context: 'browser' },
      { id: 'mitre-heatmap',           route: '/mitre-heatmap',           label: 'MITRE Heatmap',      icon: 'Grid3x3',      description: 'ATT&CK coverage' },
      { id: 'response',                route: '/response',                label: 'SOAR / Response',    icon: 'Zap',          description: 'Run playbooks' },
      { id: 'ransomware',              route: '/ransomware',              label: 'Ransomware Defense', icon: 'ShieldAlert',  description: 'Live ransomware shield' },
      { id: 'purple-team',             route: '/purple-team',             label: 'Purple Team',        icon: 'Swords',       description: 'Sim & validate' },
    ],
  },
  {
    id: 'network',
    label: 'Network',
    icon: 'Network',
    subtitle: 'Connections, traffic, DNS, geo',
    items: [
      { id: 'topology',     route: '/topology',     label: 'Connections', icon: 'Network',  description: 'Live connection graph' },
      { id: 'ndr',          route: '/ndr',          label: 'Traffic',     icon: 'Wifi',     description: 'NDR flow analytics' },
      { id: 'enrichment',   route: '/enrichment',   label: 'DNS / Enrich', icon: 'Sparkles', description: 'GeoIP, ASN, TI', context: 'browser' },
      { id: 'threat-map',   route: '/threat-map',   label: 'Geo Map',     icon: 'Map',      description: 'Worldwide threat overlay' },
      { id: 'fleet-map',    route: '/fleet-map',    label: 'Fleet Map',   icon: 'Map',      description: 'Asset geo distribution' },
      { id: 'tunnels',      route: '/tunnels',      label: 'Tunnels',     icon: 'Cable',    description: 'Port forwards', context: 'desktop' },
    ],
  },
  {
    id: 'identity',
    label: 'Identity',
    icon: 'UserCog',
    subtitle: 'Users, sessions, auth, risky users',
    items: [
      { id: 'identity',         route: '/identity',         label: 'Users',         icon: 'Users',     description: 'Identity directory', context: 'browser' },
      { id: 'identity-admin',   route: '/identity-admin',   label: 'Identity Admin', icon: 'UsersRound', description: 'Per-tenant identity', context: 'browser' },
      { id: 'recordings',       route: '/recordings',       label: 'Sessions',      icon: 'Video',     description: 'Session replay', context: 'desktop' },
      { id: 'escalation',       route: '/escalation',       label: 'Auth Events',   icon: 'Flame',     description: 'On-call routing', context: 'browser' },
      { id: 'ueba-overview',    route: '/ueba-overview',    label: 'Risky Users',   icon: 'AlertTriangle', description: 'Highest-risk identities' },
      { id: 'team',             route: '/team',             label: 'Team',          icon: 'Users',     description: 'Operator team' },
    ],
  },
  {
    id: 'hosts',
    label: 'Hosts',
    icon: 'Server',
    subtitle: 'Fleet, activity, processes',
    items: [
      { id: 'fleet-management', route: '/fleet-management', label: 'All Hosts',     icon: 'Boxes',         description: 'Fleet inventory', context: 'browser' },
      { id: 'fleet',            route: '/fleet',            label: 'Fleet Overview', icon: 'Server',       description: 'Status grid' },
      { id: 'hosts',            route: '/hosts',            label: 'Asset Inventory', icon: 'HardDrive',   description: 'Discovered assets' },
      { id: 'agents',           route: '/agents',           label: 'Agent Console',  icon: 'Cpu',          description: 'Per-agent control', context: 'browser', badgeKey: 'agents' },
      { id: 'operator',         route: '/operator',         label: 'Operator Mode',  icon: 'Crosshair',    description: 'SIEM-aware shell context', context: 'desktop' },
      { id: 'ssh',              route: '/ssh',              label: 'SSH Bookmarks',  icon: 'Server',       description: 'Vaulted hosts', context: 'desktop' },
    ],
  },
  {
    id: 'shell',
    label: 'Shell',
    icon: 'TerminalSquare',
    subtitle: 'Local & remote terminals',
    items: [
      { id: 'shell',            route: '/shell',            label: 'Shell Workspace', icon: 'TerminalSquare', description: 'Multi-pane local + SSH terminals', context: 'desktop' },
      { id: 'shell-recordings', route: '/recordings',       label: 'Recordings',      icon: 'Video',          description: 'Replay past shell sessions', context: 'desktop' },
      { id: 'shell-snippets',   route: '/snippets',         label: 'Snippets',        icon: 'Code',           description: 'Saved shell commands', context: 'desktop' },
      { id: 'shell-tunnels',    route: '/tunnels',          label: 'Port Forwards',   icon: 'Cable',          description: 'SSH tunnels', context: 'desktop' },
    ],
  },
  {
    id: 'logs',
    label: 'Logs',
    icon: 'FileText',
    subtitle: 'Search, live, saved',
    items: [
      { id: 'siem-search',  route: '/siem-search',  label: 'Search',        icon: 'Search',      description: 'OQL + Lucene', context: 'browser' },
      { id: 'siem',         route: '/siem',         label: 'Live Stream',   icon: 'Activity',    description: 'Tail -f style' },
      { id: 'health',       route: '/monitoring',   label: 'Pipeline Health', icon: 'HeartPulse', description: 'Ingest metrics' },
      { id: 'lineage',      route: '/lineage',      label: 'Data Lineage',  icon: 'Workflow',    description: 'Where did this come from?' },
      { id: 'temporal',     route: '/temporal-integrity', label: 'Temporal Integrity', icon: 'Clock4', description: 'Late-event audit' },
      { id: 'replay',       route: '/response-replay', label: 'Response Replay', icon: 'Rewind', description: 'Re-run incident' },
      { id: 'snippets',     route: '/snippets',     label: 'Saved Queries', icon: 'Code',        description: 'Pinned OQL', context: 'desktop' },
      { id: 'notes',        route: '/notes',        label: 'Notes',         icon: 'StickyNote',  description: 'Per-host notes', context: 'desktop' },
    ],
  },
  {
    id: 'system',
    label: 'System',
    icon: 'Settings',
    subtitle: 'Agents, health, rules, integrations',
    items: [
      { id: 'compliance',  route: '/compliance',  label: 'Compliance',         icon: 'ClipboardCheck', description: 'SOC2, HIPAA, ISO' },
      { id: 'vault',       route: '/vault',       label: 'Vault',              icon: 'KeyRound',       description: 'Encrypted secrets' },
      { id: 'secrets',     route: '/secrets',     label: 'Secrets',            icon: 'Lock',           description: 'Rotation policies' },
      { id: 'security',    route: '/trust',       label: 'Runtime Trust',      icon: 'ShieldCheck',    description: 'Trust scores' },
      { id: 'suppression', route: '/suppression', label: 'Suppression Rules',  icon: 'EyeOff',         description: 'False-positive rules' },
      { id: 'admin',       route: '/admin',       label: 'Super Admin',        icon: 'ShieldHalf',     description: 'Tenant lifecycle' },
      { id: 'plugins',     route: '/plugins',     label: 'Integrations',       icon: 'Puzzle',         description: 'Plugin marketplace' },
      { id: 'license',     route: '/license',     label: 'License',            icon: 'Award',          description: 'Feature flags' },
      { id: 'sync',        route: '/sync',        label: 'Sync',               icon: 'RefreshCw',      description: 'Cross-instance sync', context: 'desktop' },
      { id: 'ai-assistant', route: '/ai-assistant', label: 'AI Shell',         icon: 'Bot',            description: 'Ask the assistant' },
      { id: 'settings',    route: '/workspace',   label: 'Settings',           icon: 'Settings',       description: 'App preferences' },
      { id: 'shortcuts',   route: '/shortcuts',   label: 'Keyboard Shortcuts', icon: 'Keyboard',       description: 'Hotkey reference' },
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
