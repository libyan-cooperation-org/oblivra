/**
 * timeRange.svelte.ts — global time-range scope.
 *
 * Phase 31 SOC redesign — there is one platform-wide "what time
 * window am I looking at" selector, mounted in TitleBar so it's
 * reachable from every page. Pages that care (SIEMSearch,
 * HostDetail timeline, ActivityFeed) read this store and re-filter
 * their content; pages that don't care simply ignore it.
 *
 * Persisted to localStorage so a reload drops the operator back
 * into the same window.
 */

const STORAGE_KEY = 'oblivra:timeRange';

export type TimePreset =
  | 'live'
  | '5m'
  | '1h'
  | '24h'
  | '7d'
  | '30d'
  | 'install'
  | 'custom';

export interface TimeRange {
  start: string | null;   // ISO; null when LIVE
  end: string | null;     // ISO; null = "now"
  preset: TimePreset;
}

const DEFAULT: TimeRange = { start: null, end: null, preset: 'live' };

function readPersisted(): TimeRange {
  if (typeof localStorage === 'undefined') return DEFAULT;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT;
    const parsed = JSON.parse(raw) as TimeRange;
    if (!parsed || typeof parsed.preset !== 'string') return DEFAULT;
    return parsed;
  } catch {
    return DEFAULT;
  }
}

class TimeRangeStore {
  range = $state<TimeRange>(readPersisted());

  set(next: TimeRange) {
    this.range = { ...next };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.range));
    } catch { /* private mode */ }
  }

  /**
   * Resolve the current range to absolute (start, end) timestamps,
   * filling 'now' for null end and recomputing relative presets at
   * call time so a long-running session doesn't drift.
   */
  resolve(): { start: Date | null; end: Date } {
    const r = this.range;
    const end = r.end ? new Date(r.end) : new Date();
    if (r.preset === 'live' || !r.start) return { start: null, end };
    return { start: new Date(r.start), end };
  }
}

export const timeRangeStore = new TimeRangeStore();
