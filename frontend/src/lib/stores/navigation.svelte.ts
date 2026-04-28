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
// Schema bumped from v1 → v2 with the SOC redesign (Overview/Security/
// Network/Identity/Hosts/Logs/System). v1 stored 'operate'/'detect'
// etc. which no longer exist; rather than migrate the IDs we drop the
// stored prefs on first load — the operator just lands on the new
// 'overview' default and re-pins anything they care about.
const SCHEMA_VERSION = 4;  // Phase 33: 5 groups → 6 groups + subsections

// SOC investigation-first taxonomy.
// Phase 33 (Path A): 6 sidebar groups, each with 2-3 BottomDock
// subsections. Each top-level group answers one operator question:
//
//   siem    → "what is the system reporting?"  — triage + hunt + dashboards
//   invest  → "what happened?"                 — cases, timeline, lineage
//   respond → "what should I do?"              — SOAR, forensics, evidence
//   fleet   → "what's running?"                — assets, network, shell
//   govern  → "is the platform in order?"      — identity, compliance, secrets
//   admin   → "configure the platform"         — settings, dashboards, help
//
// Old IDs (Phase 32 5-group set + the legacy 8-group set before that)
// are migrated below so persisted state survives the schema bump.
export type NavGroupId =
  | 'siem'
  | 'invest'
  | 'respond'
  | 'fleet'
  | 'govern'
  | 'admin';

/**
 * Migrate legacy NavGroupId values into the current 6-group schema.
 *
 * Mapping table (locked — do not change without bumping SCHEMA_VERSION):
 *
 *   legacy id     → current id
 *   ───────────   ─────────────
 *   Phase 32 set:
 *   operations    → fleet      (shell + SSH + hosts + network all moved here)
 *   investigate   → invest     (renamed; cases/timeline stay)
 *
 *   Pre-Phase-32 (8-group) set:
 *   overview      → siem       (Mission Control lives at the top of SIEM)
 *   security      → siem       (alerts, hunt, intel — SIEM-native)
 *   logs          → siem       (search, live tail, saved queries)
 *   shell         → fleet      (terminal lives under FLEET → Shell)
 *   hosts         → fleet      (assets, agents)
 *   network       → fleet      (topology, NDR, tunnels)
 *   identity      → govern     (users, SSO, DSR)
 *   system        → admin      (settings, plugins, license, sync)
 *
 *   already-current id → returned unchanged
 *   unknown id         → 'siem' (most-trafficked default)
 *
 * Exported so we can unit-test the mapping table without spinning up
 * the Svelte rune runtime.
 */
export function migrateLegacyGroup(id: string): NavGroupId {
  switch (id) {
    // Phase 32 → Phase 33
    case 'operations':
      return 'fleet';
    case 'investigate':
      return 'invest';
    // Pre-Phase-32 (8-group)
    case 'overview':
    case 'logs':
    case 'security':
      return 'siem';
    case 'shell':
    case 'hosts':
    case 'network':
      return 'fleet';
    case 'identity':
      return 'govern';
    case 'system':
      return 'admin';
  }
  // Already a current id, or unknown — fall back to siem so the operator
  // lands on the most-trafficked group.
  if (id === 'siem' || id === 'invest' || id === 'respond' ||
      id === 'fleet' || id === 'govern' || id === 'admin') {
    return id;
  }
  return 'siem';
}

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
    // Schema-aware migration: v < SCHEMA_VERSION rewrites the active
    // group through migrateLegacyGroup so a 7-group preference survives
    // the collapse to 5 groups instead of dropping to the default.
    if (typeof parsed.activeGroup !== 'string') return null;
    if (!Array.isArray(parsed.pinned)) return null;
    return {
      ...parsed,
      v: SCHEMA_VERSION,
      activeGroup: migrateLegacyGroup(parsed.activeGroup),
    };
  } catch {
    return null;
  }
}

class NavigationStore {
  // ── State ────────────────────────────────────────────────────────
  activeGroup = $state<NavGroupId>('siem');
  pinned = $state<string[]>([]);
  dockExpanded = $state<boolean>(true);

  // Last route opened inside each group, so re-clicking a group restores
  // the operator's last view rather than always landing on the first item.
  lastRouteByGroup = $state<Record<NavGroupId, string | null>>({
    siem:    null,
    invest:  null,
    respond: null,
    fleet:   null,
    govern:  null,
    admin:   null,
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
