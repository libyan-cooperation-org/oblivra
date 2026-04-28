// OBLIVRA — Sovereignty store (Phase 32).
//
// Hot-loads the sovereignty score from the backend so the chrome badge
// and the executive dash tile both read from a single source of truth.
// Refreshed every 5 minutes — the four signals (on-prem, TPM, air-gap,
// KMS) don't move at sub-minute timescales, so we conserve CPU.

import { apiFetch } from '@lib/apiClient';

export interface SovereigntyComponent {
  name: string;
  ok: boolean;
  reason: string;
  weight: number;
  earned: number;
}

export interface SovereigntyScore {
  score: number;
  tier: 'gold' | 'silver' | 'bronze' | 'unverified';
  components: SovereigntyComponent[];
}

class SovereigntyStore {
  score = $state<SovereigntyScore | null>(null);
  loaded = $state(false);
  loading = $state(false);

  private timer: ReturnType<typeof setInterval> | null = null;

  async refresh() {
    this.loading = true;
    try {
      const res = await apiFetch('/api/v1/sovereignty/score');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const body = (await res.json()) as SovereigntyScore;
      this.score = body;
      this.loaded = true;
    } catch (e) {
      // Network down or backend wedged — leave the previous value in
      // place so the badge doesn't flicker. The chrome badge defaults
      // to "—" if score is null.
      console.warn('[sovereignty] refresh failed:', e);
    } finally {
      this.loading = false;
    }
  }

  start(intervalMs = 5 * 60 * 1000) {
    if (this.timer) return;
    void this.refresh();
    this.timer = setInterval(() => { void this.refresh(); }, intervalMs);
  }

  stop() {
    if (this.timer) clearInterval(this.timer);
    this.timer = null;
  }
}

export const sovereigntyStore = new SovereigntyStore();
