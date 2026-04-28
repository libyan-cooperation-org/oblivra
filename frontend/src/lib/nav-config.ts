/**
 * nav-config.ts — central definition of the 5 OBLIVRA navigation
 * groups (Phase 32, UIUX_IMPROVEMENTS.md P3 sidebar consolidation).
 *
 * Working memory holds 5±2 chunks. We had 8. We now have 5:
 *
 *   SIEM        → "what's the system reporting?"
 *                 alerts, search, hunt, threat intel, MITRE, suppression,
 *                 ransomware, response (SOAR)
 *   OPERATIONS  → "what am I doing on a host?"
 *                 shell, ssh, recordings, fleet, agents, operator,
 *                 topology, NDR, tunnels
 *   INVESTIGATE → "what happened?"
 *                 cases, timeline, forensics, evidence, lineage, replay,
 *                 threat-graph, anomalies, purple-team
 *   GOVERN      → "is the platform in order?"
 *                 compliance, DSR, identity, identity-admin, SSO, vault
 *   ADMIN       → "configure the platform"
 *                 settings, plugins, sync, license, AI assistant, shortcuts
 *
 * Switching a group sets the active CONTEXT (navigationStore.activeGroup)
 * and lands on the group's primary route. The BottomDock surfaces the
 * group's tools as cards.
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

// ── 5 OBLIVRA Groups ─────────────────────────────────────────────────

export const NAV_GROUPS: NavGroup[] = [
  {
    id: 'siem',
    label: 'SIEM',
    icon: 'Shield',
    subtitle: 'Alerts, search, hunt',
    items: [
      // Alert queue + dashboards
      { id: 'overview',                route: '/overview',                label: 'Mission Control',     icon: 'LayoutDashboard', description: 'Risk + incidents + live activity' },
      { id: 'dashboard',               route: '/dashboard',               label: 'Tactical Hub',        icon: 'Activity',        description: 'Operator overview' },
      { id: 'executive',               route: '/executive',               label: 'Executive View',      icon: 'TrendingUp',      description: 'KPIs for leadership' },
      { id: 'alerts',                  route: '/alerts',                  label: 'Alerts',              icon: 'Bell',            description: 'Active triggers', badgeKey: 'alerts' },
      { id: 'alert-management',        route: '/alert-management',        label: 'Alert Management',    icon: 'BellRing',        description: 'Triage queue', context: 'browser' },
      // Search + hunt
      { id: 'siem-search',             route: '/siem-search',             label: 'Search',              icon: 'Search',          description: 'OQL + Lucene', context: 'browser' },
      { id: 'siem',                    route: '/siem',                    label: 'Live Stream',         icon: 'Activity',        description: 'Tail -f style' },
      { id: 'threat-hunter',           route: '/threat-hunter',           label: 'Threat Hunter',       icon: 'Telescope',       description: 'Hypothesis-driven hunts' },
      { id: 'threat-intel-dashboard',  route: '/threat-intel-dashboard',  label: 'Threat Intel',        icon: 'Globe',           description: 'TI feeds & matches', context: 'browser' },
      { id: 'mitre-heatmap',           route: '/mitre-heatmap',           label: 'MITRE Heatmap',       icon: 'Grid3x3',         description: 'ATT&CK coverage' },
      // Response
      { id: 'response',                route: '/response',                label: 'SOAR / Response',     icon: 'Zap',             description: 'Run playbooks' },
      { id: 'ransomware',              route: '/ransomware',              label: 'Ransomware Defense',  icon: 'ShieldAlert',     description: 'Live ransomware shield' },
      { id: 'suppression',             route: '/suppression',             label: 'Suppression Rules',   icon: 'EyeOff',          description: 'False-positive rules' },
      // Newly surfaced
      { id: 'playbook-builder',        route: '/playbook-builder',        label: 'Playbook Builder',    icon: 'Workflow',        description: 'Author SOAR playbooks' },
      { id: 'credentials',             route: '/credentials',             label: 'Credential Intel',    icon: 'KeyRound',        description: 'Leaked-credential monitoring' },
      { id: 'simulation',              route: '/simulation',              label: 'Simulation',          icon: 'FlaskConical',    description: 'Attack-path replay' },
    ],
  },
  {
    id: 'operations',
    // 'OPERATIONS' (10 chars) overflows the 64 px sidebar at 9 px font
    // size. 'OPS' is the canonical SOC abbreviation. Tooltip surfaces
    // the full word for new operators.
    label: 'OPS',
    icon: 'TerminalSquare',
    subtitle: 'Shell, hosts, network · Operations',
    items: [
      // Shell + SSH (desktop-only)
      { id: 'shell',                   route: '/shell',                   label: 'Shell Workspace',     icon: 'TerminalSquare',  description: 'Multi-pane local + SSH terminals', context: 'desktop' },
      { id: 'operator',                route: '/operator',                label: 'Operator Mode',       icon: 'Crosshair',       description: 'SIEM-aware shell context', context: 'desktop' },
      { id: 'ssh',                     route: '/ssh',                     label: 'SSH Bookmarks',       icon: 'Server',          description: 'Vaulted hosts', context: 'desktop' },
      { id: 'recordings',              route: '/recordings',              label: 'Recordings',          icon: 'Video',           description: 'Replay past shell sessions', context: 'desktop' },
      { id: 'snippets',                route: '/snippets',                label: 'Snippets',            icon: 'Code',            description: 'Saved shell commands', context: 'desktop' },
      // Fleet
      { id: 'fleet-management',        route: '/fleet-management',        label: 'All Hosts',           icon: 'Boxes',           description: 'Fleet inventory', context: 'browser' },
      { id: 'fleet',                   route: '/fleet',                   label: 'Fleet Overview',      icon: 'Server',          description: 'Status grid' },
      { id: 'agents',                  route: '/agents',                  label: 'Agent Console',       icon: 'Cpu',             description: 'Per-agent control', context: 'browser', badgeKey: 'agents' },
      // Network
      { id: 'topology',                route: '/topology',                label: 'Connections',         icon: 'Network',         description: 'Live connection graph' },
      { id: 'ndr',                     route: '/ndr',                     label: 'Traffic',             icon: 'Wifi',            description: 'NDR flow analytics' },
      { id: 'tunnels',                 route: '/tunnels',                 label: 'Tunnels',             icon: 'Cable',           description: 'Port forwards', context: 'desktop' },
      { id: 'fleet-map',               route: '/fleet-map',               label: 'Fleet Map',           icon: 'Map',             description: 'Asset geo distribution' },
      { id: 'global-topology',         route: '/global-topology',         label: 'Global Topology',     icon: 'Globe',           description: 'Cross-region asset graph' },
      { id: 'network-map',             route: '/network-map',             label: 'Network Map',         icon: 'Map',             description: 'Subnet / VLAN map' },
      { id: 'data-destruction',        route: '/data-destruction',        label: 'Data Destruction',    icon: 'Trash2',          description: 'Authorised wipe of host data', context: 'desktop' },
    ],
  },
  {
    id: 'investigate',
    // 'INVESTIGATE' (11 chars) overflows the 64 px sidebar; 'CASES'
    // is the operator-facing umbrella term anyway (cases include
    // timeline + evidence + lineage).
    label: 'CASES',
    icon: 'Folder',
    subtitle: 'Cases, timeline, evidence · Investigate',
    items: [
      { id: 'cases',                   route: '/cases',                   label: 'Incidents',           icon: 'FolderOpen',      description: 'Multi-alert cases' },
      { id: 'timeline',                route: '/timeline',                label: 'Timeline',            icon: 'Clock4',          description: 'Cross-source incident timeline' },
      { id: 'graph',                   route: '/graph',                   label: 'Campaigns',           icon: 'GitBranch',       description: 'Multi-step attack chains' },
      { id: 'forensics',               route: '/forensics',               label: 'Forensics',           icon: 'Microscope',      description: 'Evidence acquisition', context: 'desktop' },
      { id: 'evidence',                route: '/evidence',                label: 'Evidence Ledger',     icon: 'BookLock',        description: 'Tamper-evident chain-of-custody' },
      { id: 'lineage',                 route: '/lineage',                 label: 'Data Lineage',        icon: 'Workflow',        description: 'Where did this come from?' },
      { id: 'replay',                  route: '/response-replay',         label: 'Response Replay',     icon: 'Rewind',          description: 'Re-run incident' },
      { id: 'temporal',                route: '/temporal-integrity',      label: 'Temporal Integrity',  icon: 'Clock4',          description: 'Late-event audit' },
      { id: 'ueba',                    route: '/ueba',                    label: 'Anomalies',           icon: 'Users',           description: 'UEBA deviations' },
      { id: 'ueba-overview',           route: '/ueba-overview',           label: 'Risky Users',         icon: 'AlertTriangle',   description: 'Highest-risk identities' },
      { id: 'purple-team',             route: '/purple-team',             label: 'Purple Team',         icon: 'Swords',          description: 'Sim & validate' },
      { id: 'enrichment',              route: '/enrichment',              label: 'Enrichment',          icon: 'Sparkles',        description: 'GeoIP, ASN, TI', context: 'browser' },
      { id: 'threat-map',              route: '/threat-map',              label: 'Geo Map',             icon: 'Map',             description: 'Worldwide threat overlay' },
    ],
  },
  {
    id: 'govern',
    label: 'Govern',
    icon: 'Scale',
    subtitle: 'Compliance, identity, DSR',
    items: [
      { id: 'compliance',              route: '/compliance',              label: 'Compliance',          icon: 'ClipboardCheck',  description: 'SOC2, HIPAA, ISO' },
      { id: 'dsr',                     route: '/dsr',                     label: 'DSR',                 icon: 'Scale',           description: 'GDPR Art. 15 / 17 + CCPA' },
      { id: 'identity',                route: '/identity',                label: 'Users',               icon: 'Users',           description: 'Identity directory', context: 'browser' },
      { id: 'identity-admin',          route: '/identity-admin',          label: 'Identity Admin',      icon: 'UsersRound',      description: 'Per-tenant identity', context: 'browser' },
      { id: 'identity-connectors',     route: '/identity-connectors',     label: 'SSO Connectors',      icon: 'KeyRound',        description: 'OIDC + SAML federation' },
      { id: 'team',                    route: '/team',                    label: 'Team',                icon: 'Users',           description: 'Operator team' },
      { id: 'escalation',              route: '/escalation',              label: 'On-call',             icon: 'Flame',           description: 'On-call routing', context: 'browser' },
      { id: 'vault',                   route: '/vault',                   label: 'Vault',               icon: 'KeyRound',        description: 'Encrypted secrets' },
      { id: 'secrets',                 route: '/secrets',                 label: 'Secrets',             icon: 'Lock',            description: 'Rotation policies' },
      { id: 'secret-manager',          route: '/secret-manager',          label: 'Secret Manager',      icon: 'Lock',            description: 'Cross-tenant secret governance' },
      { id: 'audit',                   route: '/audit',                   label: 'Audit Log',           icon: 'ScrollText',      description: 'Sealed operator timeline' },
      { id: 'chain-of-custody',        route: '/chain-of-custody',        label: 'Chain-of-Custody',    icon: 'Link',            description: 'Evidence custody chain' },
      { id: 'trust',                   route: '/trust',                   label: 'Runtime Trust',       icon: 'ShieldCheck',     description: 'Trust scores' },
      { id: 'admin',                   route: '/admin',                   label: 'Super Admin',         icon: 'ShieldHalf',      description: 'Tenant lifecycle' },
    ],
  },
  {
    id: 'admin',
    label: 'Admin',
    icon: 'Settings',
    subtitle: 'Settings, plugins, sync',
    items: [
      { id: 'settings',                route: '/workspace',               label: 'Settings',            icon: 'Settings',        description: 'App preferences + Operator Profile' },
      { id: 'plugins',                 route: '/plugins',                 label: 'Integrations',        icon: 'Puzzle',          description: 'Plugin marketplace' },
      { id: 'sync',                    route: '/sync',                    label: 'Sync',                icon: 'RefreshCw',       description: 'Cross-instance sync', context: 'desktop' },
      { id: 'license',                 route: '/license',                 label: 'License',             icon: 'Award',           description: 'Feature flags' },
      { id: 'health',                  route: '/monitoring',              label: 'Pipeline Health',     icon: 'HeartPulse',      description: 'Ingest metrics' },
      { id: 'ai-assistant',            route: '/ai-assistant',            label: 'AI Shell',            icon: 'Bot',             description: 'Ask the assistant' },
      { id: 'shortcuts',               route: '/shortcuts',               label: 'Keyboard Shortcuts',  icon: 'Keyboard',        description: 'Hotkey reference' },
      { id: 'notes',                   route: '/notes',                   label: 'Notes',               icon: 'StickyNote',      description: 'Per-host notes', context: 'desktop' },
      // Newly surfaced
      { id: 'setup-wizard',            route: '/setup-wizard',            label: 'Setup Wizard',        icon: 'Wand2',           description: 'First-run + reconfigure' },
      { id: 'dashboard-studio',        route: '/dashboard-studio',        label: 'Dashboard Studio',    icon: 'LayoutDashboard', description: 'Author custom dashboards' },
      { id: 'features',                route: '/features',                label: 'Feature Flags',       icon: 'Flag',            description: 'Per-deployment flags' },
      { id: 'risk',                    route: '/risk',                    label: 'Config Risk',         icon: 'AlertTriangle',   description: 'Misconfiguration risk' },
      { id: 'offline-update',          route: '/offline-update',          label: 'Offline Update',      icon: 'Download',        description: 'Air-gap update bundle', context: 'desktop' },
      { id: 'development',             route: '/development',             label: 'Development',         icon: 'Code',            description: 'Internal dev tools' },
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
