export type NavId =
  | 'overview'
  | 'siem'
  | 'detection'
  | 'investigations'
  | 'cases'
  | 'reconstruction'
  | 'trust'
  | 'graph'
  | 'fleet'
  | 'evidence'
  | 'vault'
  | 'webhooks'
  | 'admin';

export interface NavItem {
  id: NavId;
  label: string;
  group: 'siem' | 'respond' | 'manage';
  icon: string; // emoji placeholder; replace with svg pack later
  hint?: string;
}

export const NAV: NavItem[] = [
  { id: 'overview',       label: 'Overview',       group: 'siem',    icon: '◎', hint: 'Platform health' },
  { id: 'siem',           label: 'SIEM',           group: 'siem',    icon: '⌗', hint: 'Search & live tail' },
  { id: 'detection',      label: 'Detection',      group: 'siem',    icon: '◈', hint: 'Sigma + native rules' },
  { id: 'investigations', label: 'Investigations', group: 'respond', icon: '⌕', hint: 'Per-host triage' },
  { id: 'cases',          label: 'Cases',          group: 'respond', icon: '⎙', hint: 'Frozen-snapshot investigations' },
  { id: 'reconstruction', label: 'Reconstruction', group: 'respond', icon: '⏮', hint: 'Sessions, state, cmdline, auth' },
  { id: 'trust',          label: 'Trust & Quality',group: 'respond', icon: '✓', hint: 'Provenance & tamper signals' },
  { id: 'evidence',       label: 'Evidence',       group: 'respond', icon: '⎘', hint: 'Audit chain & sealed packages' },
  { id: 'graph',          label: 'Evidence Graph', group: 'respond', icon: '☍', hint: 'Cross-references' },
  { id: 'fleet',          label: 'Fleet',          group: 'manage',  icon: '⌬', hint: 'Agents & collectors' },
  { id: 'vault',          label: 'Vault',          group: 'manage',  icon: '🔒', hint: 'Encrypted secrets' },
  { id: 'webhooks',       label: 'Webhooks',       group: 'manage',  icon: '↗', hint: 'Outbound alert delivery' },
  { id: 'admin',          label: 'Admin',          group: 'manage',  icon: '⚙', hint: 'Tenants & storage' },
];

export const GROUP_LABEL: Record<NavItem['group'], string> = {
  siem:    'Observe',
  respond: 'Respond',
  manage:  'Manage',
};
