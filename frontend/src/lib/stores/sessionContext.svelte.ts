// OBLIVRA — Session Context Store (Phase 32, UIUX P0/P1).
//
// Tracks the operator's pivot path across the app so the chrome can
// render a breadcrumb-like trail that's clickable (jumps back to any
// previous step with the original filter / time-range / tenant
// context preserved).
//
// Why not just use browser history?
//
//  • History records URL navigations, but our pivots carry FILTERS
//    that aren't in the URL (selected severities, search query,
//    selected entity).
//  • History entries can't be expressed as semantic "alert →
//    host → process" — they're just URLs.
//  • We want to clear the trail on idle (>30 min) so a returning
//    operator after lunch starts fresh.
//
// Persistence is sessionStorage (NOT localStorage) — pivot trails are
// session-scoped, not cross-launch. A new browser session starts with
// a clean trail.

const STORAGE_KEY = 'oblivra:pivotTrail';
const TTL_MS = 30 * 60 * 1000; // 30 min — see header.

export type PivotKind =
  | 'alert'
  | 'host'
  | 'user'
  | 'process'
  | 'ip'
  | 'session'
  | 'case'
  | 'rule'
  | 'tenant'
  | 'page';

export interface PivotCrumb {
  /** Stable id — used for click-to-restore and de-dup. */
  id: string;
  /** Discriminator for the icon + colour in the chrome. */
  kind: PivotKind;
  /** Short label shown in the crumb chip, e.g. "DB-PROD-01" or
   *  alert id. Keep <= 24 chars for chrome density. */
  label: string;
  /** Hash route to navigate back to — restores the page that produced
   *  this crumb in its original filter state. */
  route: string;
  /** Optional extra params the destination page can read to restore
   *  filters / selections (alert id, time range, severity multiselect). */
  params?: Record<string, string>;
  /** When the operator first visited this crumb. */
  ts: number;
}

interface PersistedTrail {
  trail: PivotCrumb[];
  expires: number;
}

class SessionContextStore {
  trail = $state<PivotCrumb[]>([]);
  /** Caps the trail length so the chrome doesn't run off the edge. */
  readonly MAX_DEPTH = 8;

  constructor() {
    this.hydrate();
  }

  /**
   * Push a new crumb. If the same `id` already exists in the trail we
   * truncate everything after it (operator pivoted back, then forward
   * to a new branch — old branch shouldn't linger).
   */
  push(c: Omit<PivotCrumb, 'ts'>) {
    const stamped: PivotCrumb = { ...c, ts: Date.now() };
    const idx = this.trail.findIndex((p) => p.id === c.id);
    if (idx >= 0) {
      // Truncate forward path, then reuse the existing crumb so the
      // timestamp stays meaningful (still tracks first visit).
      this.trail = this.trail.slice(0, idx + 1);
    } else {
      this.trail = [...this.trail, stamped];
      if (this.trail.length > this.MAX_DEPTH) {
        this.trail = this.trail.slice(this.trail.length - this.MAX_DEPTH);
      }
    }
    this.persist();
  }

  /** Replace the entire trail — used when the operator clicks an
   *  earlier crumb (we trim forward) or starts a new task. */
  jumpTo(id: string) {
    const idx = this.trail.findIndex((p) => p.id === id);
    if (idx >= 0) {
      this.trail = this.trail.slice(0, idx + 1);
      this.persist();
    }
  }

  clear() {
    this.trail = [];
    try { sessionStorage.removeItem(STORAGE_KEY); } catch { /* private mode */ }
  }

  /** Pop the last crumb — useful for an explicit "back" affordance. */
  pop() {
    if (this.trail.length === 0) return null;
    const popped = this.trail[this.trail.length - 1];
    this.trail = this.trail.slice(0, -1);
    this.persist();
    return popped;
  }

  // ── persistence ─────────────────────────────────────────────
  private hydrate() {
    if (typeof sessionStorage === 'undefined') return;
    try {
      const raw = sessionStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw) as PersistedTrail;
      if (!parsed || typeof parsed.expires !== 'number') return;
      if (Date.now() > parsed.expires) {
        sessionStorage.removeItem(STORAGE_KEY);
        return;
      }
      this.trail = parsed.trail ?? [];
    } catch { /* corrupt — ignore */ }
  }

  private persist() {
    if (typeof sessionStorage === 'undefined') return;
    try {
      const payload: PersistedTrail = {
        trail: this.trail,
        expires: Date.now() + TTL_MS,
      };
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    } catch { /* quota / private mode */ }
  }
}

export const sessionContext = new SessionContextStore();
