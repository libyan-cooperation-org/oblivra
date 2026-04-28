// OBLIVRA — Crown-Jewel asset tagging (Phase 32 follow-up).
//
// Operator-side tag store: a Set of host ids the operator has marked
// as "crown-jewel" (tier-1 / business-critical). Drives the NBA
// recommender's `IsFromCrownJewel` fact-flag and the UI badge in
// HostDetail / FleetDashboard.
//
// Persistence: localStorage for v1. Cross-device + tenant-aware
// persistence will land in Phase 33 once we wire HostRepository.Update
// through to a REST mutation endpoint (see internal/database/hosts.go
// — the schema already has a Tags []string field; we just don't have
// a REST surface for it yet).

const STORAGE_KEY = 'oblivra:crownJewels';

class CrownJewelsStore {
  ids = $state<Set<string>>(new Set());

  constructor() {
    this.hydrate();
  }

  has(hostId: string): boolean {
    if (!hostId) return false;
    return this.ids.has(hostId);
  }

  add(hostId: string) {
    if (!hostId || this.ids.has(hostId)) return;
    const next = new Set(this.ids);
    next.add(hostId);
    this.ids = next;
    this.persist();
  }

  remove(hostId: string) {
    if (!this.ids.has(hostId)) return;
    const next = new Set(this.ids);
    next.delete(hostId);
    this.ids = next;
    this.persist();
  }

  toggle(hostId: string) {
    if (this.has(hostId)) this.remove(hostId);
    else this.add(hostId);
  }

  list(): string[] {
    return Array.from(this.ids);
  }

  private hydrate() {
    if (typeof localStorage === 'undefined') return;
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const arr = JSON.parse(raw) as string[];
      if (Array.isArray(arr)) this.ids = new Set(arr);
    } catch { /* private mode / corrupt */ }
  }

  private persist() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(Array.from(this.ids)));
    } catch { /* quota */ }
  }
}

export const crownJewels = new CrownJewelsStore();
