/**
 * OBLIVRA — Toast Store (Svelte 5 runes)
 *
 * Replaces the SolidJS createSignal-based toast system.
 */

export type ToastType = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
    id: string;
    type: ToastType;
    title: string;
    message?: string;
    duration?: number; // ms
}

class ToastStore {
    items = $state<Toast[]>([]);

    add(toast: Omit<Toast, 'id'>) {
        const id = Math.random().toString(36).substring(2, 9);
        const newToast = { ...toast, id };
        this.items = [...this.items, newToast];

        if (newToast.duration !== 0) {
            setTimeout(() => this.remove(id), newToast.duration || 5000);
        }
    }

    remove(id: string) {
        this.items = this.items.filter((t) => t.id !== id);
    }
}

export const toastStore = new ToastStore();
