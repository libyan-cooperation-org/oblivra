/**
 * Bridge between SolidJS frontend and Go backend via Wails.
 *
 * In DESKTOP / HYBRID mode: uses the Wails runtime (window.runtime).
 * In BROWSER mode: gracefully degrades — subscribe() is a no-op,
 * but the app still renders. Components that need live events should
 * check IS_DESKTOP before subscribing.
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
