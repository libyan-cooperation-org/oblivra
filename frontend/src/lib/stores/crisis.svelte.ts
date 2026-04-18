/**
 * OBLIVRA — Crisis Mode Store (Svelte 5 runes)
 *
 * Tracks whether Crisis Mode is active and exposes trigger/stand-down logic.
 * Crisis Mode auto-activates when:
 *   - alert spike rate exceeds threshold (calls checkAlertSpike from outside)
 *   - confirmed breach pattern event arrives via subscribe()
 *   - manual arm via arm()
 *
 * When active, the App shell applies `.crisis-mode` to <main>.
 * UI components can read crisisStore.active to simplify their layout.
 */
import { subscribe } from '../bridge';

// Thresholds
const SPIKE_WINDOW_MS   = 60_000;  // 1-minute rolling window
const SPIKE_THRESHOLD   = 5;       // critical alerts in window → auto-arm
const BREACH_SEVERITIES = ['P0', 'critical', 'breach_confirmed'];

class CrisisStore {
  active   = $state(false);
  reason   = $state<string | null>(null);
  armedAt  = $state<string | null>(null);

  // Rolling window: timestamps of recent critical alerts
  private _spikeBucket: number[] = [];
  private _initialized = false;

  init() {
    if (this._initialized) return;
    this._initialized = true;

    // Auto-arm on confirmed breach events from backend
    subscribe('alert:critical', (data: any) => {
      this._recordSpike();
      const severity = data?.severity ?? data?.level ?? '';
      if (BREACH_SEVERITIES.some(s => severity?.toLowerCase().includes(s))) {
        this.arm(`Confirmed breach pattern: ${data?.title ?? severity}`);
      }
    });

    subscribe('crisis:arm', (data: any) => {
      this.arm(data?.reason ?? 'Remote crisis signal received');
    });

    subscribe('crisis:stand_down', () => {
      this.standDown();
    });
  }

  /** Record an inbound critical alert and check spike threshold. */
  checkAlertSpike(count: number = 1) {
    for (let i = 0; i < count; i++) this._recordSpike();
  }

  private _recordSpike() {
    const now = Date.now();
    this._spikeBucket = this._spikeBucket.filter(t => now - t < SPIKE_WINDOW_MS);
    this._spikeBucket.push(now);
    if (!this.active && this._spikeBucket.length >= SPIKE_THRESHOLD) {
      this.arm(`Alert spike: ${this._spikeBucket.length} critical alerts in 60s`);
    }
  }

  arm(reason: string = 'Manual activation') {
    if (this.active) return;
    this.active  = true;
    this.reason  = reason;
    this.armedAt = new Date().toISOString();
    console.warn('[OBLIVRA] Crisis Mode ARMED:', reason);
  }

  standDown() {
    this.active  = false;
    this.reason  = null;
    this.armedAt = null;
    this._spikeBucket = [];
  }
}

export const crisisStore = new CrisisStore();
