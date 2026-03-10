// Bridge between SolidJS frontend and Go backend via Wails
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

type EventCallback<T = any> = (data: T) => void;
const eventListeners: Map<string, EventCallback[]> = new Map();

export async function initBridge(): Promise<void> {
    if (typeof window === 'undefined') throw new Error('Window not available');
    return new Promise((resolve) => {
        let attempts = 0;
        const check = () => {
            if ((window as any)['runtime']) {
                resolve();
            } else if (attempts > 40) {
                console.warn("Wails runtime not found. Are you running outside of Wails?");
                resolve();
            } else {
                attempts++;
                setTimeout(check, 50);
            }
        };
        check();
    });
}

export function subscribe<T = any>(event: string, callback: EventCallback<T>): () => void {
    if ((window as any)['runtime']) {
        EventsOn(event, callback as any);
    } else {
        console.warn(`[Mock] subscribe(${event}) called but Wails runtime is missing`);
    }

    if (!eventListeners.has(event)) eventListeners.set(event, []);
    eventListeners.get(event)!.push(callback as EventCallback);
    return () => {
        if ((window as any)['runtime']) {
            EventsOff(event);
        }
        const listeners = eventListeners.get(event);
        if (listeners) {
            const idx = listeners.indexOf(callback as EventCallback);
            if (idx > -1) listeners.splice(idx, 1);
        }
    };
}

export function cleanupAllListeners(): void {
    if ((window as any)['runtime']) {
        for (const [event] of eventListeners) EventsOff(event);
    }
    eventListeners.clear();
}
