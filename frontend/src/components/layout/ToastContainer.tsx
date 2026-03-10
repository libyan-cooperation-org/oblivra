import { Component, For } from 'solid-js';
import { useToast, ToastType } from '../../core/toast';
import '../../styles/toast.css';

const IconForType = (type: ToastType) => {
    switch (type) {
        case 'success': return '✓';
        case 'error': return '✕';
        case 'warning': return '⚠';
        case 'info':
        default: return 'i';
    }
};

export const ToastContainer: Component = () => {
    const { toasts, removeToast } = useToast();

    return (
        <div class="toast-container">
            <For each={toasts()}>
                {(toast) => (
                    <div class={`toast toast-${toast.type}`}>
                        <div class="toast-icon">
                            {IconForType(toast.type)}
                        </div>
                        <div class="toast-content">
                            <span class="toast-title">{toast.title}</span>
                            {toast.message && <span class="toast-message">{toast.message}</span>}
                        </div>
                        <button class="toast-close" onClick={() => removeToast(toast.id)}>
                            ✕
                        </button>
                    </div>
                )}
            </For>
        </div>
    );
};
