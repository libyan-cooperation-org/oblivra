import { createSignal } from 'solid-js';

export type ToastType = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
    id: string;
    type: ToastType;
    title: string;
    message?: string;
    duration?: number; // ms
}

const [toasts, setToasts] = createSignal<Toast[]>([]);

export const useToast = () => {
    const addToast = (toast: Omit<Toast, 'id'>) => {
        const id = Math.random().toString(36).substring(2, 9);
        const newToast = { ...toast, id };
        setToasts((prev) => [...prev, newToast]);

        if (newToast.duration !== 0) {
            setTimeout(() => removeToast(id), newToast.duration || 5000);
        }
    };

    const removeToast = (id: string) => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
    };

    return {
        toasts,
        addToast,
        removeToast,
    };
};
