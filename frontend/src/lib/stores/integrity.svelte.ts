// OBLIVRA — Integrity store (Tamper Path 1, Layer 5).
//
// Polls /api/v1/integrity every 30s and exposes the per-status
// counts. Drives:
//   • Tactical Hub KPI tile ("Tamper Indicators")
//   • Crisis Decision Panel auto-arm (≥2 tampered hosts → arm)
//   • Sidebar badge on the GOVERN → Compliance → Integrity entry
//     (future enhancement; present count exposed already)

import { apiFetch } from '@lib/apiClient';

export interface IntegrityCounts {
  healthy: number;
  stale: number;
  dark: number;
  tampered: number;
}

class IntegrityStore {
  counts = $state<IntegrityCounts>({ healthy: 0, stale: 0, dark: 0, tampered: 0 });
  loaded = $state(false);
  lastRefresh = $state<string | null>(null);

  private timer: ReturnType<typeof setInterval> | null = null;

  async refresh() {
    try {
      const res = await apiFetch('/api/v1/integrity');
      if (!res.ok) return;
      const body = await res.json();
      this.counts = {
        healthy: body.healthy_count ?? 0,
        stale: body.stale_count ?? 0,
        dark: body.dark_count ?? 0,
        tampered: body.tampered_count ?? 0,
      };
      this.loaded = true;
      this.lastRefresh = new Date().toISOString();
    } catch {
      // Network blip — keep last value.
    }
  }

  start(intervalMs = 30_000) {
    if (this.timer) return;
    void this.refresh();
    this.timer = setInterval(() => { void this.refresh(); }, intervalMs);
  }

  stop() {
    if (this.timer) clearInterval(this.timer);
    this.timer = null;
  }
}

export const integrityStore = new IntegrityStore();
