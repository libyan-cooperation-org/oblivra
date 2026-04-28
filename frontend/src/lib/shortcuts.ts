// OBLIVRA — Keyboard shortcuts registry (Phase 32, UIUX_IMPROVEMENTS.md P2 #14).
//
// The single source of truth for every operator-visible keyboard
// shortcut. App.svelte and TriageDrawer.svelte read from this list
// when wiring handlers; KeyboardMap.svelte reads it to render the
// help page. Adding a new shortcut means editing ONE file.
//
// Why a registry instead of inline keydown handlers per page?
//
//  1. Discoverability — operators can't memorise ten shortcuts they've
//     never seen. The shortcuts page is auto-generated from this list,
//     so it's never stale.
//  2. Conflict detection — two pages binding the same key fight at
//     runtime. With the registry, a future PR can lint for duplicates.
//  3. Per-profile gating — the `requires` field lets the registry
//     declare "this only fires when profile.vimLeader is true," which
//     keeps SOC Analyst's `g`-in-search-box from being hijacked.

export type ShortcutScope = 'global' | 'operator' | 'shell' | 'triage';

export interface ShortcutDef {
  /** Human-readable key sequence, in Mac convention (⌘ for cmd). */
  keys: string;
  /** What the shortcut does. Shown verbatim on the shortcuts page. */
  label: string;
  /** Where the shortcut is active. `global` = any page. */
  scope: ShortcutScope;
  /** Optional profile-rule predicate: when present, the shortcut only
   *  fires if `appStore.profileRules[<name>]` matches. */
  requires?: 'vimLeader' | 'paletteFront' | 'tenantSwitcherBar';
  /** Optional grouping for the shortcuts page (e.g. "Navigation"). */
  group?: string;
}

export const SHORTCUTS: ShortcutDef[] = [
  // ── Global ─────────────────────────────────────────────────────
  { keys: '⌘K',           label: 'Open command palette',                scope: 'global', group: 'Global' },
  { keys: '⌘P',           label: 'Quick-find (alias for ⌘K)',           scope: 'global', group: 'Global' },
  { keys: '⌘/',           label: 'Show keyboard shortcuts',             scope: 'global', group: 'Global' },
  { keys: 'Esc',          label: 'Close palette / drawer / modal',      scope: 'global', group: 'Global' },

  // ── Vim leader navigation (profile-gated) ──────────────────────
  { keys: 'g, d',         label: 'Go to Dashboard',                     scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, a',         label: 'Go to Alerts',                        scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, s',         label: 'Go to SIEM',                          scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, h',         label: 'Go to Hunt (SIEM Search)',            scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, f',         label: 'Go to Fleet',                         scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, v',         label: 'Go to Evidence Ledger',               scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, o',         label: 'Go to Operator Mode',                 scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },
  { keys: 'g, t',         label: 'Go to Timeline',                      scope: 'global', requires: 'vimLeader', group: 'Navigation (g+letter)' },

  // ── Tenant fast-switcher (profile-gated) ───────────────────────
  { keys: '⌘T',           label: 'Switch tenant (Mac)',                 scope: 'global', requires: 'tenantSwitcherBar', group: 'Multi-tenant' },
  { keys: 'Ctrl+Alt+T',   label: 'Switch tenant (Win/Linux)',           scope: 'global', requires: 'tenantSwitcherBar', group: 'Multi-tenant' },

  // ── Operator (active host context, /operator) ──────────────────
  { keys: 'Ctrl+Shift+I', label: 'Isolate active host',                 scope: 'operator', group: 'Operator' },
  { keys: 'Ctrl+Shift+E', label: 'Capture evidence on active host',     scope: 'operator', group: 'Operator' },

  // ── Triage drawer (when an alert is selected) ──────────────────
  { keys: 'Enter',        label: 'Open selected alert in drawer',       scope: 'triage', group: 'Triage' },
  { keys: '⌘+Enter',      label: 'Execute recommended next-best-action',scope: 'triage', group: 'Triage' },
  { keys: 't',            label: 'Pivot to timeline',                   scope: 'triage', group: 'Triage' },
  { keys: 's',            label: 'Pivot to shell on host',              scope: 'triage', group: 'Triage' },
  { keys: 'g',            label: 'Pivot to threat graph (drawer-local)',scope: 'triage', group: 'Triage' },
  { keys: 'e',            label: 'Pivot to evidence',                   scope: 'triage', group: 'Triage' },
  { keys: 'x',            label: 'Suppress alert (5s undo)',            scope: 'triage', group: 'Triage' },

  // ── Shell workspace ────────────────────────────────────────────
  { keys: 'Ctrl+Click',   label: 'Inject investigation command (alert rail)', scope: 'shell', group: 'Shell' },
  { keys: 'Ctrl+.',       label: 'Expand IOC threat-intel popover',     scope: 'shell', group: 'Shell' },
];

/** Group shortcuts by their `group` field for the help page. */
export function groupedShortcuts(): Record<string, ShortcutDef[]> {
  const out: Record<string, ShortcutDef[]> = {};
  for (const s of SHORTCUTS) {
    const k = s.group ?? 'Other';
    (out[k] = out[k] ?? []).push(s);
  }
  return out;
}
