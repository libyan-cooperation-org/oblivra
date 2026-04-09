/**
 * Bridge between Svelte frontend and Go backend via Wails.
 *
 * In DESKTOP / HYBRID mode: uses the Wails runtime (window.runtime).
 * In BROWSER mode: gracefully degrades — subscribe() is a no-op,
 * but the app still renders. Components that need live events should
 * check IS_DESKTOP before subscribing.
 *
 * Also provides rpcGuard() — a per-method debounce that prevents
 * accidental double-fires and rate-limits destructive Wails RPC calls.
 */
import { APP_CONTEXT } from './context';

// Conditionally import Wails runtime — only safe when running inside Wails.
// In browser mode window.runtime doesn't exist, so these are no-ops.
function getRuntime() {
    return (window as any)['runtime'] ?? null;
}

type EventCallback<T = any> = (data: T) => void;
const eventListeners: Map<string, EventCallback[]> = new Map();

/**
 * Initialise the bridge.
 *
 * Desktop: waits up to 2s for the Wails runtime to inject itself.
 * Browser: resolves immediately — no runtime needed.
 */
export async function initBridge(): Promise<void> {
    if (typeof window === 'undefined') throw new Error('Window not available');

    // In browser mode we don't need the Wails runtime — resolve immediately.
    if (APP_CONTEXT === 'browser') {
        console.info('[bridge] Running in browser mode — Wails runtime not required');
        return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
        let attempts = 0;
        const check = () => {
            if (getRuntime()) {
                resolve();
            } else if (attempts > 40) {
                // Desktop build but runtime missing — something went wrong.
                reject(new Error(
                    "Wails runtime missing. Run via 'wails dev' or the native binary. " +
                    "If you meant to run in browser mode, the server should serve the app directly."
                ));
            } else {
                attempts++;
                setTimeout(check, 50);
            }
        };
        check();
    });
}

/**
 * Subscribe to a backend event.
 *
 * Desktop / Hybrid: delegates to the Wails EventsOn mechanism.
 * Browser: no-op (events arrive via WebSocket from the server, handled separately).
 *
 * Returns an unsubscribe function.
 */
export function subscribe<T = any>(event: string, callback: EventCallback<T>): () => void {
    const rt = getRuntime();

    if (rt) {
        rt.EventsOn(event, callback as any);
    }

    if (!eventListeners.has(event)) eventListeners.set(event, []);
    eventListeners.get(event)!.push(callback as EventCallback);

    return () => {
        if (rt) {
            rt.EventsOff(event);
        }
        const listeners = eventListeners.get(event);
        if (listeners) {
            const idx = listeners.indexOf(callback as EventCallback);
            if (idx > -1) listeners.splice(idx, 1);
        }
    };
}

/**
 * Unsubscribe all listeners. Called on app teardown.
 */
export function cleanupAllListeners(): void {
    const rt = getRuntime();
    if (rt) {
        for (const [event] of eventListeners) rt.EventsOff(event);
    }
    eventListeners.clear();
}

/**
 * Emit a synthetic event locally (for testing / browser mock mode).
 */
export function emitLocal<T = any>(event: string, data: T): void {
    const listeners = eventListeners.get(event);
    if (listeners) {
        for (const cb of listeners) cb(data);
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// RPC Rate-Limiter
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Tracks the last invocation timestamp for each guard key.
 * Keys are arbitrary strings, typically `ServiceName.MethodName`.
 */
const _rpcLastCall = new Map<string, number>();

/**
 * Tracks in-flight promises to prevent concurrent duplicate calls.
 */
const _rpcInflight = new Map<string, Promise<any>>();

/**
 * rpcGuard — wraps a Wails RPC thunk with:
 *   1. Cooldown enforcement   — rejects if called again within `cooldownMs`.
 *   2. In-flight deduplication — returns the existing promise if one is already
 *      running for the same key (no concurrent duplicate calls).
 *
 * Usage:
 *   const result = await rpcGuard('HostService.Delete', 2000, () =>
 *       import('...HostService').then(m => m.Delete(id))
 *   );
 *
 * @param key        Unique identifier for this call site (e.g. 'VaultService.Unlock').
 * @param cooldownMs Minimum milliseconds between successive calls. Default 1000ms.
 * @param thunk      Async factory that performs the actual RPC.
 * @returns          Resolves with the RPC result, or rejects with a RateLimitError.
 */
export async function rpcGuard<T>(
    key: string,
    cooldownMs: number,
    thunk: () => Promise<T>
): Promise<T> {
    // Return in-flight promise for the same key (deduplication)
    const inflight = _rpcInflight.get(key);
    if (inflight) return inflight as Promise<T>;

    const now = Date.now();
    const last = _rpcLastCall.get(key) ?? 0;
    const elapsed = now - last;

    if (elapsed < cooldownMs) {
        const remaining = Math.ceil((cooldownMs - elapsed) / 1000);
        throw new RateLimitError(
            key,
            `Too fast — please wait ${remaining}s before retrying.`,
            cooldownMs - elapsed
        );
    }

    _rpcLastCall.set(key, now);
    const promise = thunk().finally(() => _rpcInflight.delete(key));
    _rpcInflight.set(key, promise);
    return promise;
}

/**
 * Thrown by rpcGuard when a call is rejected due to cooldown.
 */
export class RateLimitError extends Error {
    readonly key: string;
    readonly retryAfterMs: number;
    constructor(key: string, message: string, retryAfterMs: number) {
        super(message);
        this.name = 'RateLimitError';
        this.key = key;
        this.retryAfterMs = retryAfterMs;
    }
}

/**
 * isRateLimitError — type guard for RateLimitError.
 */
export function isRateLimitError(err: unknown): err is RateLimitError {
    return err instanceof RateLimitError;
}

// ─────────────────────────────────────────────────────────────────────────────
// Pre-wired guards for the three destructive desktop methods
// ─────────────────────────────────────────────────────────────────────────────

/**
 * guardedUnlock — rate-limited VaultService.UnlockWithPassword.
 * Cooldown: 3 seconds (prevents brute-force hammering from the UI).
 */
export async function guardedUnlock(
    passphrase: string,
    remember: boolean
): Promise<void> {
    return rpcGuard('VaultService.Unlock', 3_000, async () => {
        const { UnlockWithPassword } = await import(
            '../../wailsjs/go/services/VaultService'
        );
        return UnlockWithPassword(passphrase, remember);
    });
}

/**
 * guardedDeleteHost — rate-limited HostService.Delete.
 * Cooldown: 2 seconds per host ID.
 */
export async function guardedDeleteHost(hostId: string): Promise<void> {
    return rpcGuard(`HostService.Delete:${hostId}`, 2_000, async () => {
        const { Delete } = await import(
            '../../wailsjs/go/services/HostService'
        );
        return Delete(hostId);
    });
}

/**
 * guardedNuclearDestruction — rate-limited SettingsService.ClearDatabase.
 * Cooldown: 30 seconds — gives the user time to reconsider if they
 * somehow trigger it twice in quick succession.
 */
export async function guardedNuclearDestruction(): Promise<void> {
    return rpcGuard('SettingsService.ClearDatabase', 30_000, async () => {
        const { ClearDatabase } = await import(
            '../../wailsjs/go/services/SettingsService'
        );
        return ClearDatabase();
    });
}
