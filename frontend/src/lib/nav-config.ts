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

/**
 * NavSubsection — a labelled bucket of items inside a NavGroup.
 *
 * The BottomDock renders each subsection as a sub-header + a card row
 * beneath it, so a single group can hold dozens of items without
 * losing scannability. Subsection labels are sentence-case (`Triage`,
 * `Hunt`, `Forensics`) — they're informational, not menu items.
 */
export interface NavSubsection {
  label: string;
  /** Optional one-line subtitle for the subsection header. */
  subtitle?: string;
  items: NavItem[];
}

export interface NavGroup {
  id: NavGroupId;
  /** Short label shown in the 64-px sidebar — keep ≤ 7 chars. */
  label: string;
  /** Lucide icon name for the sidebar. */
  icon: string;
  /** One-line subtitle — used in tooltips + dock header. */
  subtitle: string;
  /** Subsections rendered in the BottomDock. Every group has at least
   *  one subsection; flat groups use a single unnamed subsection. */
  subsections: NavSubsection[];
}

// ── 6 OBLIVRA Groups (Path A) ─────────────────────────────────────────
//
// Each top-level group holds 2–3 subsections so the BottomDock can
// render dense menus without cognitive overload at the sidebar level.
// 70+ pages reachable across 6 sidebar buttons; subsections do the
// scoping work inside the dock.

export const NAV_GROUPS: NavGroup[] = [
  // ─────────────────────────────────────────────────────────────────
  // SIEM — most-trafficked. Triage + Hunt + Overview dashboards.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'siem',
    label: 'SIEM',
    icon: 'Shield',
    subtitle: 'Triage, hunt, dashboards',
    subsections: [
      {
        label: 'Overview',
        subtitle: 'Tactical command surfaces',
        items: [
          { id: 'dashboard',          route: '/dashboard',          label: 'Tactical Hub',        icon: 'Activity',        description: 'Operator overview' },
          { id: 'overview',           route: '/overview',           label: 'Mission Control',     icon: 'LayoutDashboard', description: 'Risk + incidents + live activity' },
          { id: 'ops',                route: '/ops',                label: 'Ops Center',          icon: 'Monitor',         description: 'War-room layout' },
          { id: 'soc',                route: '/soc',                label: 'SOC Workspace',       icon: 'Eye',             description: 'Multi-analyst SOC', context: 'browser' },
          { id: 'war-mode',           route: '/war-mode',           label: 'War Mode',            icon: 'Siren',           description: 'Crisis-led full-screen takeover' },
        ],
      },
      {
        label: 'Triage',
        subtitle: 'Inbound signal, queue management',
        items: [
          { id: 'alerts',             route: '/alerts',             label: 'Alerts Feed',         icon: 'Bell',            description: 'Active triggers', badgeKey: 'alerts' },
          { id: 'alert-management',   route: '/alert-management',   label: 'Alert Management',    icon: 'BellRing',        description: 'Triage queue + drawer', context: 'browser' },
          { id: 'suppression',        route: '/suppression',        label: 'Suppression Rules',   icon: 'EyeOff',          description: 'False-positive rules' },
          // Phase 36.9: 'escalation' removed (EscalationCenter depended on incidentservice).
        ],
      },
      {
        label: 'Hunt',
        subtitle: 'Search, intel, behavioural analytics',
        items: [
          { id: 'siem-search',        route: '/siem-search',        label: 'Search',              icon: 'Search',          description: 'OQL + Lucene', context: 'browser' },
          { id: 'siem',               route: '/siem',               label: 'Live Stream',         icon: 'Activity',        description: 'Tail -f style' },
          { id: 'threat-hunter',      route: '/threat-hunter',      label: 'Threat Hunter',       icon: 'Telescope',       description: 'Hypothesis-driven hunts' },
          { id: 'mitre-heatmap',      route: '/mitre-heatmap',      label: 'MITRE Heatmap',       icon: 'Grid3x3',         description: 'ATT&CK coverage' },
          { id: 'ueba',               route: '/ueba',               label: 'UEBA / Anomalies',    icon: 'Users',           description: 'UEBA deviations' },
          { id: 'ueba-overview',      route: '/ueba-overview',      label: 'Risky Users',         icon: 'AlertTriangle',   description: 'Highest-risk identities' },
          { id: 'ndr',                route: '/ndr',                label: 'NDR / Traffic',       icon: 'Wifi',            description: 'Network-detection flow' },
          { id: 'threat-intel',       route: '/threat-intel',       label: 'Threat Intel Feed',   icon: 'Globe',           description: 'TI feed list' },
          { id: 'threat-intel-dashboard', route: '/threat-intel-dashboard', label: 'TI Dashboard', icon: 'Globe',          description: 'TI feed posture', context: 'browser' },
          { id: 'credentials',        route: '/credentials',        label: 'Credential Intel',    icon: 'KeyRound',        description: 'Leaked-credential monitoring' },
          { id: 'enrichment',         route: '/enrichment',         label: 'Enrichment Tools',    icon: 'Sparkles',        description: 'GeoIP, ASN, TI', context: 'browser' },
          // Phase 36.9: 'purple-team' removed (PurpleTeam depended on playbookservice).
        ],
      },
    ],
  },

  // ─────────────────────────────────────────────────────────────────
  // INVEST — investigation + entity drilldowns + lineage.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'invest',
    label: 'INVEST',
    icon: 'Folder',
    subtitle: 'Cases, timeline, lineage',
    subsections: [
      {
        label: 'Cases',
        subtitle: 'Incident workspace',
        items: [
          { id: 'investigation',      route: '/investigation',      label: 'Investigation',       icon: 'SearchCheck',     description: 'Drill-down workspace' },
          { id: 'entity',             route: '/entity',             label: 'Entity View',         icon: 'Network',         description: 'Per-entity drill-down' },
        ],
      },
      {
        label: 'Graph & Lineage',
        subtitle: 'Relationships and data provenance',
        items: [
          { id: 'graph',              route: '/graph',              label: 'Threat Graph',        icon: 'GitBranch',       description: 'Multi-step attack chains' },
          { id: 'lineage',            route: '/lineage',            label: 'Data Lineage',        icon: 'Workflow',        description: 'Where did this come from?' },
          { id: 'decisions',          route: '/decisions',          label: 'Decision Inspector',  icon: 'Brain',           description: 'Why-the-engine-decided' },
        ],
      },
      {
        label: 'Replay & Audit',
        subtitle: 'Reconstruct, verify, query',
        items: [
          // Phase 36.9: 'replay' removed (ResponseReplay depended on incidentservice;
          // slot reserved for Phase 38 evidence-package replay viewer).
          { id: 'temporal',           route: '/temporal-integrity', label: 'Temporal Integrity',  icon: 'Clock',           description: 'Late-event audit' },
          { id: 'oql',                route: '/oql',                label: 'OQL Dashboard',       icon: 'Code',            description: 'Saved OQL queries' },
        ],
      },
    ],
  },

  // ─────────────────────────────────────────────────────────────────
  // RESPOND — kill-chain right side. SOAR + forensics + evidence.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'respond',
    label: 'RESPOND',
    icon: 'Zap',
    subtitle: 'SOAR, forensics, evidence',
    subsections: [
      {
        label: 'Automate',
        subtitle: 'Run playbooks, contain, simulate',
        items: [
          { id: 'simulation',         route: '/simulation',         label: 'Simulation',          icon: 'FlaskConical',    description: 'Attack-path replay' },
          { id: 'data-destruction',   route: '/data-destruction',   label: 'Data Destruction',    icon: 'Trash2',          description: 'DoD-compliant wipe', context: 'desktop' },
        ],
      },
      {
        label: 'Forensics',
        subtitle: 'Evidence acquisition',
        items: [
          { id: 'terminal-forensics', route: '/terminal-forensics', label: 'Terminal Forensics',  icon: 'TerminalSquare',  description: 'Per-session forensic tooling', context: 'desktop' },
        ],
      },
      {
        label: 'Evidence',
        subtitle: 'Sealed chain-of-custody',
        items: [
          { id: 'evidence',           route: '/evidence',           label: 'Evidence Ledger',     icon: 'BookLock',        description: 'Tamper-evident ledger' },
          { id: 'chain-of-custody',   route: '/chain-of-custody',   label: 'Chain of Custody',    icon: 'Link',            description: 'Custody chain detail' },
          { id: 'evidence-vault',     route: '/evidence-vault',     label: 'Evidence Vault',      icon: 'Vault',           description: 'Sealed evidence storage' },
        ],
      },
    ],
  },

  // ─────────────────────────────────────────────────────────────────
  // FLEET — assets + network + shell. Everything host-related.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'fleet',
    label: 'FLEET',
    icon: 'Server',
    subtitle: 'Assets, network, shell',
    subsections: [
      {
        label: 'Assets',
        subtitle: 'Hosts, agents, trust',
        items: [
          { id: 'hosts',              route: '/hosts',              label: 'Asset Inventory',     icon: 'HardDrive',       description: 'Discovered assets' },
          { id: 'fleet',              route: '/fleet',              label: 'Fleet Overview',      icon: 'Server',          description: 'Status grid' },
          { id: 'fleet-management',   route: '/fleet-management',   label: 'All Hosts',           icon: 'Boxes',           description: 'Full fleet inventory', context: 'browser' },
          { id: 'agents',             route: '/agents',             label: 'Agent Console',       icon: 'Cpu',             description: 'Per-agent control', context: 'browser', badgeKey: 'agents' },
          { id: 'operator',           route: '/operator',           label: 'Operator Mode',       icon: 'Crosshair',       description: 'SIEM-aware shell context', context: 'desktop' },
          { id: 'trust',              route: '/runtime-trust',      label: 'Runtime Trust',       icon: 'ShieldCheck',     description: 'Trust scores' },
        ],
      },
      {
        label: 'Network',
        subtitle: 'Topology and geo views',
        items: [
          { id: 'topology',           route: '/topology',           label: 'Network Topology',    icon: 'Network',         description: 'Live connection graph' },
          { id: 'network-map',        route: '/network-map',        label: 'Network Map',         icon: 'Map',             description: 'Subnet / VLAN map' },
          { id: 'global-topology',    route: '/global-topology',    label: 'Global Topology',     icon: 'Globe',           description: 'Cross-region asset graph' },
          { id: 'threat-map',         route: '/threat-map',         label: 'Geo / Threat Map',    icon: 'Map',             description: 'Worldwide threat overlay' },
          { id: 'fleet-map',          route: '/fleet-map',          label: 'Fleet Map',           icon: 'MapPin',          description: 'Asset geo distribution' },
        ],
      },
      // Phase 33: Shell subsection removed. The shell subsystem is
      // being rebuilt; /shell, /ssh, /tunnels, /recordings,
      // /session-playback are offline pending replacement. /snippets
      // and /notes still work and have moved to ADMIN → Help → ...
      // (or surface them where the new shell lands).
    ],
  },

  // ─────────────────────────────────────────────────────────────────
  // GOVERN — identity + compliance + audit + secrets/vault.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'govern',
    label: 'GOVERN',
    icon: 'Scale',
    subtitle: 'Identity, compliance, secrets',
    subsections: [
      {
        label: 'Identity & Access',
        subtitle: 'RBAC, SSO, federation',
        items: [
          // No bare /identity page exists — Identity Admin is the
          // canonical user-management surface. Pointing the "Users"
          // entry at /identity-admin avoids the previous 404. The
          // duplicate Identity Admin entry below is kept for the
          // explicit "admin" mental model — both lead to the same
          // page; pick whichever label resonates.
          { id: 'identity',           route: '/identity-admin',     label: 'Users & Identity',    icon: 'Users',           description: 'Identity directory', context: 'browser' },
          { id: 'identity-admin',     route: '/identity-admin',     label: 'Identity Admin',      icon: 'UsersRound',      description: 'Per-tenant identity', context: 'browser' },
          { id: 'identity-connectors',route: '/identity-connectors',label: 'SSO Connectors',      icon: 'KeyRound',        description: 'OIDC + SAML federation' },
          { id: 'team',               route: '/team',               label: 'Team',                icon: 'Users',           description: 'Operator team' },
          { id: 'admin',              route: '/admin',              label: 'Super Admin',         icon: 'ShieldHalf',      description: 'Tenant lifecycle' },
        ],
      },
      {
        label: 'Compliance',
        subtitle: 'Frameworks, privacy, audit',
        items: [
          { id: 'dsr',                route: '/dsr',                label: 'DSR (GDPR / CCPA)',   icon: 'Scale',           description: 'Data subject requests' },
          { id: 'audit',              route: '/audit',              label: 'Audit Log',           icon: 'ScrollText',      description: 'Sealed operator timeline' },
          { id: 'integrity',          route: '/integrity',          label: 'Agent Integrity',     icon: 'ShieldCheck',     description: 'Tamper-evidence · heartbeat · log truncation' },
          { id: 'storage-tiering',    route: '/storage-tiering',    label: 'Storage Tiering',     icon: 'Database',        description: 'Hot · Warm · Cold lifecycle + migration cycles' },
        ],
      },
      {
        label: 'Vault & Secrets',
        subtitle: 'Encrypted storage, rotation',
        items: [
          { id: 'vault',              route: '/vault',              label: 'Vault',               icon: 'KeyRound',        description: 'Encrypted secrets' },
          { id: 'secrets',            route: '/secrets',            label: 'Secrets',             icon: 'Lock',            description: 'Rotation policies' },
          { id: 'secret-manager',     route: '/secret-manager',     label: 'Secret Manager',      icon: 'Lock',            description: 'Cross-tenant governance' },
        ],
      },
    ],
  },

  // ─────────────────────────────────────────────────────────────────
  // ADMIN — settings + platform + executive dashboards + help.
  // ─────────────────────────────────────────────────────────────────
  {
    id: 'admin',
    label: 'ADMIN',
    icon: 'Settings',
    subtitle: 'Settings, dashboards, help',
    subsections: [
      {
        label: 'Platform',
        subtitle: 'Configuration and lifecycle',
        items: [
          { id: 'settings',           route: '/settings',           label: 'Settings',            icon: 'Settings',        description: 'App preferences + Operator Profile' },
          { id: 'connectors',         route: '/connectors',         label: 'Connectors (I/O)',    icon: 'Cable',           description: 'Inputs / outputs YAML + hot-reload' },
          { id: 'setup-wizard',       route: '/setup-wizard',       label: 'Setup Wizard',        icon: 'Wand2',           description: 'First-run + reconfigure' },
          { id: 'license',            route: '/license',            label: 'License',             icon: 'Award',           description: 'License key' },
          { id: 'features',           route: '/features',           label: 'Feature Flags',       icon: 'Flag',            description: 'Per-deployment flags' },
          { id: 'health',             route: '/monitoring',         label: 'Pipeline Health',     icon: 'HeartPulse',      description: 'Ingest metrics' },
          // Phase 36.9: 'risk' removed (ConfigRisk depended on complianceservice).
          { id: 'sync',               route: '/sync',                label: 'Sync',                icon: 'RefreshCw',       description: 'Cross-instance sync', context: 'desktop' },
          { id: 'offline-update',     route: '/offline-update',     label: 'Offline Update',      icon: 'Download',        description: 'Air-gap update bundle', context: 'desktop' },
          { id: 'development',        route: '/development',        label: 'Development Tools',   icon: 'Code',            description: 'Internal dev tools' },
        ],
      },
      {
        label: 'Dashboards',
        subtitle: 'Executive + analytics surfaces',
        items: [
          { id: 'executive',          route: '/executive',          label: 'Executive View',      icon: 'TrendingUp',      description: 'Leadership KPIs' },
          { id: 'analytics',          route: '/analytics',          label: 'Analytics',           icon: 'BarChart3',       description: 'Cross-source analytics' },
          { id: 'fusion',             route: '/fusion',             label: 'Fusion Dashboard',    icon: 'Sparkle',         description: 'Cross-source attack fusion' },
          { id: 'dashboard-studio',   route: '/dashboard-studio',   label: 'Dashboard Studio',    icon: 'LayoutDashboard', description: 'Author custom dashboards' },
        ],
      },
      {
        label: 'Help & Productivity',
        subtitle: 'Assistance, reference, personal',
        items: [
          // AI Assistant removed in Phase 36 (broad scope cut — log-driven SIEM).
          { id: 'shortcuts',          route: '/shortcuts',          label: 'Keyboard Shortcuts',  icon: 'Keyboard',        description: 'Hotkey reference' },
          // Snippets + notes still ship — moved here from FLEET → Shell
          // since they're personal productivity, not shell-bound.
          { id: 'snippets',           route: '/snippets',           label: 'Snippets',            icon: 'Code',            description: 'Saved commands (shell-exec offline)' },
          { id: 'notes',              route: '/notes',              label: 'Notes',               icon: 'StickyNote',      description: 'Per-host notes', context: 'desktop' },
        ],
      },
    ],
  },
];

/** Lookup the group definition for a NavGroupId. */
export function getGroup(id: NavGroupId): NavGroup | undefined {
  return NAV_GROUPS.find((g) => g.id === id);
}

/** Find which group a route id belongs to (first match). Walks every
 *  subsection of every group. */
export function findGroupForRoute(routeId: string): NavGroupId | null {
  for (const g of NAV_GROUPS) {
    for (const sec of g.subsections) {
      if (sec.items.some((it) => it.id === routeId)) return g.id;
    }
  }
  return null;
}

/** Flat list of every item — used by the command palette + the pinned
 *  strip. Walks every subsection of every group. */
export function allItems(): NavItem[] {
  return NAV_GROUPS.flatMap((g) => g.subsections.flatMap((sec) => sec.items));
}
