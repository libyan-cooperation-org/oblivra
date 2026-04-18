/**
 * Bridge between Svelte frontend and Go backend via Wails.
 *
 * In DESKTOP / HYBRID mode: uses the Wails v3 runtime module (@wailsio/runtime).
 * In BROWSER mode: gracefully degrades — subscribe() is a no-op,
 * but the app still renders. Components that need live events should
 * check IS_DESKTOP before subscribing.
 *
 * Also provides rpcGuard() — a per-method debounce that prevents
 * accidental double-fires and rate-limits destructive Wails RPC calls.
 */
import { APP_CONTEXT, IS_BROWSER } from './context';
export { APP_CONTEXT };

let ws: WebSocket | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECTS = 5;


// Wails v3 uses module-based event API, not window globals.
// We lazy-import so browser mode doesn't crash on missing @wailsio/runtime.
let WailsRuntime: typeof import('@wailsio/runtime') | null = null;
let loadPromise: Promise<typeof import('@wailsio/runtime') | null> | null = null;

async function loadWailsRuntime() {
    if (WailsRuntime) return WailsRuntime;
    if (IS_BROWSER) {
        console.debug('[bridge] Skipping runtime load — in browser mode');
        return null;
    }
    
    if (loadPromise) return loadPromise;

    loadPromise = (async () => {
        try {
            console.log('[bridge] Dynamic loading @wailsio/runtime...');
            WailsRuntime = await import('@wailsio/runtime');
            console.info('[bridge] @wailsio/runtime loaded successfully');
            return WailsRuntime;
        } catch (e) {
            console.error('[bridge] Failed to load @wailsio/runtime:', e);
            return null;
        }
    })();

    return loadPromise;
}

type EventCallback<T = any> = (data: T) => void;
const eventListeners: Map<string, EventCallback[]> = new Map();
// Track unsubscribe functions returned by Wails v3 On()
const wailsUnsubs: Map<string, Map<EventCallback, () => void>> = new Map();

/**
 * Initialise the bridge.
 *
 * Desktop: eagerly loads the Wails v3 runtime event module.
 * Browser: resolves immediately — no runtime needed.
 */
export async function initBridge(): Promise<void> {
    if (typeof window === 'undefined') throw new Error('Window not available');

    if (IS_BROWSER) {
        console.info('[bridge] Running in browser mode — starting WebSocket event stream');
        initBrowserBridge();
        return;
    }

    // Pre-load the runtime so subscribe() calls are synchronous if they happen after init
    await loadWailsRuntime();
}

/**
 * Browser Bridge: Event streaming via WebSocket
 */
function initBrowserBridge(): void {
    if (ws) return;

    // Detect protocol and host
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/api/v1/events?token=oblivra-dev-key`;

    console.log(`[bridge] Connecting to event stream: ${url}`);
    
    ws = new WebSocket(url);

    ws.onopen = () => {
        console.info('%c[bridge] WebSocket connected to event stream', 'color: #00ff00; font-weight: bold; background: #000; padding: 2px 5px;');
        reconnectAttempts = 0;
    };

    ws.onmessage = (event) => {
        console.debug('[bridge] Raw WebSocket message:', event.data);
        try {
            const raw = JSON.parse(event.data);
            const eventType = raw.Type || raw.type;
            const eventData = raw.Data || raw.data;

            if (!eventType) return;

            // Diagnostic: verbose logging for SIEM events
            if (eventType === 'siem.event_indexed') {
                console.log('[bridge] Received SIEM indexing event via WS');
                // Map to siem-stream for UI components
                emitLocal('siem-stream', eventData);
            }

            // Also emit globally for any subscriber using the internal backend name
            emitLocal(eventType, eventData);
        } catch (e) {
            console.error('[bridge] Failed to parse WebSocket message:', e);
        }
    };

    ws.onerror = (err) => {
        console.error('[bridge] WebSocket error:', err);
    };

    ws.onclose = () => {
        console.warn('[bridge] WebSocket disconnected');
        ws = null;
        if (reconnectAttempts < MAX_RECONNECTS) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            console.log(`[bridge] Reconnecting in ${delay}ms...`);
            setTimeout(initBrowserBridge, delay);
        } else {
            console.error('[bridge] Maximum reconnect attempts reached. Live feed suspended.');
        }
    };
}


/**
 * Subscribe to a backend event.
 *
 * Desktop / Hybrid: delegates to the Wails v3 Events.On() module API.
 * Browser: no-op (events arrive via WebSocket from the server, handled separately).
 *
 * Returns an unsubscribe function.
 */
export function subscribe<T = any>(event: string, callback: EventCallback<T>): () => void {
    console.debug(`[bridge] Subscribing to: ${event}`);
    
    let isUnsubscribed = false;

    // Async registration for Wails v3 runtime
    loadWailsRuntime().then(runtime => {
        if (isUnsubscribed) return;

        if (runtime && runtime.Events) {
            console.log(`[bridge] Registering Wails listener for: ${event}`);
            const wrappedCb = (wailsEvent: any) => {
                const payload = wailsEvent?.data ?? wailsEvent;
                console.debug(`[bridge] Event received: ${event}`, payload);
                (callback as EventCallback)(payload);
            };
            const unsub = runtime.Events.On(event, wrappedCb);
            
            if (!wailsUnsubs.has(event)) wailsUnsubs.set(event, new Map());
            wailsUnsubs.get(event)!.set(callback as EventCallback, unsub);
        } else if (!IS_BROWSER) {
            console.warn(`[bridge] Wails runtime unavailable for subscription: ${event}`);
        }
    });

    // Also track locally for emitLocal support
    if (!eventListeners.has(event)) eventListeners.set(event, []);
    eventListeners.get(event)!.push(callback as EventCallback);

    return () => {
        isUnsubscribed = true;
        // Unsubscribe from Wails v3
        const eventUnsubs = wailsUnsubs.get(event);
        if (eventUnsubs) {
            const unsub = eventUnsubs.get(callback as EventCallback);
            if (unsub) {
                unsub();
                eventUnsubs.delete(callback as EventCallback);
            }
        }

        // Remove from local listeners
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
    console.info('[bridge] Cleaning up all listeners');
    // Clean up all Wails v3 subscriptions
    for (const [, cbMap] of wailsUnsubs) {
        for (const [, unsub] of cbMap) {
            unsub();
        }
    }
    wailsUnsubs.clear();
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
            '@wailsjs/github.com/kingknull/oblivrashell/internal/services/vaultservice'
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
            '@wailsjs/github.com/kingknull/oblivrashell/internal/services/hostservice'
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
            '@wailsjs/github.com/kingknull/oblivrashell/internal/services/settingsservice'
        );
        return ClearDatabase();
    });
}
