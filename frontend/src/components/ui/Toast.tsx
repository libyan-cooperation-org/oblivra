import { Component, For, Show } from 'solid-js';
import { useApp } from '../../core/store';
import '../../styles/toast.css';

const typeConfig: Record<string, { icon: string; className: string }> = {
    success: { icon: '✓', className: 'toast-success' },
    error: { icon: '✕', className: 'toast-error' },
    info: { icon: 'ℹ', className: 'toast-info' },
    warning: { icon: '⚠', className: 'toast-warning' },
};

export const ToastContainer: Component = () => {
    const [state, actions] = useApp();

    return (
        <div class="toast-container">
            <For each={state.notifications}>
                {(notification) => {
                    const config = typeConfig[notification.type] || typeConfig.info;
                    return (
                        <div class={`toast ${config.className} toast-enter`}>
                            <span class="toast-icon">{config.icon}</span>
                            <div class="toast-content">
                                <span class="toast-message">{notification.message}</span>
                                <Show when={notification.details}>
                                    <span class="toast-details">{notification.details}</span>
                                </Show>
                            </div>
                            <button class="toast-close" onClick={() => actions.dismissNotification(notification.id)}>✕</button>
                        </div>
                    );
                }}
            </For>
        </div>
    );
};
