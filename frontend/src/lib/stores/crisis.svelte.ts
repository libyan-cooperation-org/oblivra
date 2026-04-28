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
 *
 * Phase 32: arming also auto-lifts the alert noise floor to
 * 'critical-only' so the commander isn't drowning in medium-severity
 * chatter while the building's on fire. The pre-arm floor is saved
 * and restored on stand-down.
 */
import { subscribe } from '../bridge';
import { appStore } from './app.svelte';

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

  // Saved pre-arm noise floor — restored on stand-down so the
  // operator's profile preference isn't permanently overwritten.
  private _preArmNoiseFloor: 'critical-only' | 'high+' | 'medium+' | 'all' | null = null;

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

    // Auto-lift the noise floor so the commander only sees critical
    // signals while running the incident. Saved here so stand-down
    // can put it back exactly where the operator had it.
    try {
      this._preArmNoiseFloor = appStore.profileRules.alertNoiseFloor;
      if (this._preArmNoiseFloor !== 'critical-only') {
        appStore.setProfileRule('alertNoiseFloor', 'critical-only');
      }
    } catch (e) {
      console.warn('[crisis] noise-floor lift failed:', e);
    }
  }

  standDown() {
    this.active  = false;
    this.reason  = null;
    this.armedAt = null;
    this._spikeBucket = [];

    // Restore the operator's pre-arm noise floor so their profile
    // preference isn't permanently overwritten by an incident.
    if (this._preArmNoiseFloor) {
      try {
        appStore.setProfileRule('alertNoiseFloor', this._preArmNoiseFloor);
      } catch { /* store may be torn down */ }
      this._preArmNoiseFloor = null;
    }
  }
}

export const crisisStore = new CrisisStore();
