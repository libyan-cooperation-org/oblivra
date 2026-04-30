export type NavId =
  | 'overview'
  | 'siem'
  | 'detection'
  | 'investigations'
  | 'fleet'
  | 'evidence'
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
  { id: 'investigations', label: 'Investigations', group: 'respond', icon: '⌕', hint: 'Cases & timelines' },
  { id: 'evidence',       label: 'Evidence',       group: 'respond', icon: '⎙', hint: 'Chain of custody' },
  { id: 'fleet',          label: 'Fleet',          group: 'manage',  icon: '⌬', hint: 'Agents & collectors' },
  { id: 'admin',          label: 'Admin',          group: 'manage',  icon: '⚙', hint: 'Tenants & RBAC' },
];

export const GROUP_LABEL: Record<NavItem['group'], string> = {
  siem:    'Observe',
  respond: 'Respond',
  manage:  'Manage',
};
