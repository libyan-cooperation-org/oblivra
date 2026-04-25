/**
 * OBLIVRA — Toast Store (Svelte 5 runes)
 *
 * Replaces the SolidJS createSignal-based toast system.
 *
 * Toasts are ephemeral. We also push every toast into the persistent
 * notificationStore so the operator can scroll back through their
 * morning's notifications via the bell-icon drawer.
 */

import { notificationStore } from './notifications.svelte';

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

        // Mirror into the persistent notification log so it survives
        // toast auto-dismiss. Errors / warnings get their full message;
        // info / success entries are kept terse.
        try {
            notificationStore.push(toast.type, toast.title, toast.message);
        } catch {
            // Don't let logging failures block toast rendering.
        }

        if (newToast.duration !== 0) {
            setTimeout(() => this.remove(id), newToast.duration || 5000);
        }
    }

    remove(id: string) {
        this.items = this.items.filter((t) => t.id !== id);
    }

    success(title: string, message?: string) {
        this.add({ type: 'success', title, message });
    }

    error(title: string, message?: string) {
        this.add({ type: 'error', title, message, duration: 8000 });
    }

    warn(title: string, message?: string) {
        this.add({ type: 'warning', title, message });
    }

    info(title: string, message?: string) {
        this.add({ type: 'info', title, message });
    }
}

export const toastStore = new ToastStore();
