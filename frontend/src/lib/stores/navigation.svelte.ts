/**
 * navigation.svelte.ts — Svelte 5 store backing the new sidebar+dock UX.
 *
 * Concept (per v1.4.x UI spec):
 *   - Left sidebar shows GROUPS (Operate / Detect / Investigate / Defend /
 *     Govern / System). One group is "active" at a time.
 *   - Bottom dock slides up showing the items belonging to the active group.
 *   - Operators can pin frequently-used items so they're always reachable
 *     regardless of the active group (rendered in a separate "Pinned" strip
 *     at the top of the dock).
 *   - The last-active group + the pinned set + dock-expanded state all
 *     persist to localStorage so a restart drops the operator back into
 *     the same context.
 *
 * Coexists with `CommandRail.svelte` — the legacy 44px icon rail can be
 * rendered alongside (or replaced) via `appStore.useGroupedNav` in the
 * AppLayout shell. This keeps the G+letter muscle-memory keybinds intact
 * during the transition.
 */

const STORAGE_KEY = 'oblivra:nav';
const SCHEMA_VERSION = 1;

export type NavGroupId =
  | 'operate'
  | 'detect'
  | 'investigate'
  | 'defend'
  | 'govern'
  | 'system';

interface PersistedShape {
  v: number;                  // schema version (forward-compat)
  activeGroup: NavGroupId;
  pinned: string[];           // route ids (matches CommandRail's NavTab ids)
  dockExpanded: boolean;
}

function readPersisted(): PersistedShape | null {
  if (typeof localStorage === 'undefined') return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as PersistedShape;
    // Refuse stale schema; future migrations bump SCHEMA_VERSION.
    if (parsed.v !== SCHEMA_VERSION) return null;
    if (typeof parsed.activeGroup !== 'string') return null;
    if (!Array.isArray(parsed.pinned)) return null;
    return parsed;
  } catch {
    return null;
  }
}

class NavigationStore {
  // ── State ────────────────────────────────────────────────────────
  activeGroup = $state<NavGroupId>('operate');
  pinned = $state<string[]>([]);
  dockExpanded = $state<boolean>(true);

  // Last route opened inside each group, so re-clicking a group restores
  // the operator's last view rather than always landing on the first item.
  lastRouteByGroup = $state<Record<NavGroupId, string | null>>({
    operate: null,
    detect: null,
    investigate: null,
    defend: null,
    govern: null,
    system: null,
  });

  private _initialized = false;
  private _persistTimer: ReturnType<typeof setTimeout> | null = null;

  init() {
    if (this._initialized) return;
    this._initialized = true;
    const persisted = readPersisted();
    if (persisted) {
      this.activeGroup = persisted.activeGroup;
      this.pinned = persisted.pinned;
      this.dockExpanded = persisted.dockExpanded ?? true;
    }
    // Persist on tab close so we don't lose the last second of changes.
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => this.flush());
    }
  }

  /** Switch the active group. No-op if already active. */
  setActiveGroup(g: NavGroupId) {
    if (this.activeGroup === g) {
      // Same-group click: toggle the dock open/closed for a snappier UX.
      this.dockExpanded = !this.dockExpanded;
    } else {
      this.activeGroup = g;
      this.dockExpanded = true;
    }
    this.persistDebounced();
  }

  /** Record that the operator just navigated to a route inside a group. */
  rememberRoute(group: NavGroupId, routeId: string) {
    this.lastRouteByGroup[group] = routeId;
    this.persistDebounced();
  }

  /** Toggle pinned state for a route. */
  togglePin(routeId: string) {
    if (this.pinned.includes(routeId)) {
      this.pinned = this.pinned.filter((id) => id !== routeId);
    } else {
      this.pinned = [...this.pinned, routeId];
    }
    this.persistDebounced();
  }

  isPinned(routeId: string): boolean {
    return this.pinned.includes(routeId);
  }

  toggleDock() {
    this.dockExpanded = !this.dockExpanded;
    this.persistDebounced();
  }

  // ── Persistence ──────────────────────────────────────────────────
  private persistDebounced() {
    if (this._persistTimer) clearTimeout(this._persistTimer);
    this._persistTimer = setTimeout(() => this.flush(), 200);
  }

  flush() {
    if (typeof localStorage === 'undefined') return;
    const payload: PersistedShape = {
      v: SCHEMA_VERSION,
      activeGroup: this.activeGroup,
      pinned: this.pinned,
      dockExpanded: this.dockExpanded,
    };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    } catch {
      // Quota exhausted or private browsing — drop silently.
    }
  }
}

export const navigationStore = new NavigationStore();
