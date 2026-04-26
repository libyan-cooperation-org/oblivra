/**
 * savedQueries.svelte.ts — operator-saved SIEM/log queries.
 *
 * Phase 30.4c: closes the "saved queries / share with team" gap from
 * the operator UX audit.
 *
 * Each saved query has:
 *   - id (slug derived from name + timestamp)
 *   - name (operator-chosen label)
 *   - query (OQL string)
 *   - pinned (renders as a chip at the top of SIEMSearch)
 *   - lastUsed (so the recent-searches list orders by recency)
 *   - usageCount (for analytics — most-used queries float to top)
 *   - createdAt
 *
 * Persistence: localStorage. The same shape would round-trip to a
 * server-side endpoint when 'share with team' is wired (out of scope
 * for this phase — left as 30.4c-followup).
 *
 * Lives in `.svelte.ts` (not plain `.ts`) so we can use $state — see
 * Phase 29 for why this matters.
 */

const STORAGE_KEY = 'oblivra:savedQueries';
const SCHEMA_VERSION = 1;
const MAX_RECENT = 50;

export interface SavedQuery {
  id: string;
  name: string;
  query: string;
  pinned: boolean;
  lastUsed: string;     // ISO
  usageCount: number;
  createdAt: string;    // ISO
}

interface PersistedShape {
  v: number;
  items: SavedQuery[];
}

function readPersisted(): SavedQuery[] {
  if (typeof localStorage === 'undefined') return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as PersistedShape;
    if (parsed.v !== SCHEMA_VERSION) return [];
    if (!Array.isArray(parsed.items)) return [];
    return parsed.items;
  } catch {
    return [];
  }
}

function genId(name: string): string {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 32);
  return `${slug || 'query'}-${Date.now().toString(36)}`;
}

class SavedQueriesStore {
  items = $state<SavedQuery[]>([]);

  private _initialized = false;
  private _persistTimer: ReturnType<typeof setTimeout> | null = null;

  init() {
    if (this._initialized) return;
    this._initialized = true;
    this.items = readPersisted();
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => this.flush());
    }
  }

  // ── Derived helpers (callers can use these or compute their own) ──
  get pinned(): SavedQuery[] {
    return this.items.filter((q) => q.pinned);
  }

  get recent(): SavedQuery[] {
    return [...this.items]
      .sort((a, b) => new Date(b.lastUsed).getTime() - new Date(a.lastUsed).getTime())
      .slice(0, MAX_RECENT);
  }

  // ── Mutations ─────────────────────────────────────────────────────
  save(name: string, query: string): SavedQuery {
    const trimmedName = name.trim();
    const trimmedQuery = query.trim();
    if (!trimmedName || !trimmedQuery) {
      throw new Error('Name and query are both required');
    }
    // Dedupe on identical query string — bump usageCount instead of
    // creating a duplicate entry. Preserves operator's name choice.
    const existing = this.items.find((q) => q.query === trimmedQuery);
    if (existing) {
      existing.lastUsed = new Date().toISOString();
      existing.usageCount += 1;
      this.items = [...this.items];
      this.persistDebounced();
      return existing;
    }
    const item: SavedQuery = {
      id: genId(trimmedName),
      name: trimmedName,
      query: trimmedQuery,
      pinned: false,
      lastUsed: new Date().toISOString(),
      usageCount: 1,
      createdAt: new Date().toISOString(),
    };
    this.items = [item, ...this.items];
    this.persistDebounced();
    return item;
  }

  /** Record that a saved query was just executed. */
  bumpUsage(id: string) {
    const idx = this.items.findIndex((q) => q.id === id);
    if (idx === -1) return;
    this.items[idx].usageCount += 1;
    this.items[idx].lastUsed = new Date().toISOString();
    this.items = [...this.items];
    this.persistDebounced();
  }

  togglePin(id: string) {
    const idx = this.items.findIndex((q) => q.id === id);
    if (idx === -1) return;
    this.items[idx].pinned = !this.items[idx].pinned;
    this.items = [...this.items];
    this.persistDebounced();
  }

  rename(id: string, newName: string) {
    const trimmed = newName.trim();
    if (!trimmed) return;
    const idx = this.items.findIndex((q) => q.id === id);
    if (idx === -1) return;
    this.items[idx].name = trimmed;
    this.items = [...this.items];
    this.persistDebounced();
  }

  remove(id: string) {
    this.items = this.items.filter((q) => q.id !== id);
    this.persistDebounced();
  }

  /** Replace the entire collection — used by import/export flows. */
  replaceAll(items: SavedQuery[]) {
    this.items = items;
    this.persistDebounced();
  }

  // ── Persistence ───────────────────────────────────────────────────
  private persistDebounced() {
    if (this._persistTimer) clearTimeout(this._persistTimer);
    this._persistTimer = setTimeout(() => this.flush(), 200);
  }

  flush() {
    if (typeof localStorage === 'undefined') return;
    const payload: PersistedShape = { v: SCHEMA_VERSION, items: this.items };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    } catch {
      // Quota exhausted — drop the oldest half and retry once.
      try {
        const trimmed = [...this.items]
          .sort((a, b) => new Date(b.lastUsed).getTime() - new Date(a.lastUsed).getTime())
          .slice(0, Math.floor(this.items.length / 2));
        localStorage.setItem(
          STORAGE_KEY,
          JSON.stringify({ v: SCHEMA_VERSION, items: trimmed }),
        );
        this.items = trimmed;
      } catch {
        // Give up silently.
      }
    }
  }
}

export const savedQueriesStore = new SavedQueriesStore();
