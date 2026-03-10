import { Component, For, Show } from 'solid-js';
import { useApp } from '../../core/store';
import '../../styles/transfer-drawer.css';

export const TransferDrawer: Component<{ open: boolean, onClose: () => void }> = (props) => {
    const [state] = useApp();

    const formatSize = (bytes: number) => {
        if (!bytes) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const activeTransfers = () => state.transfers.filter(t => t.status === 'active' || t.status === 'pending');
    const completedTransfers = () => state.transfers.filter(t => t.status === 'completed' || t.status === 'failed');

    return (
        <Show when={props.open}>
            <div class="transfer-drawer-overlay" onClick={props.onClose}>
                <div class="transfer-drawer glass-surface" onClick={(e) => e.stopPropagation()}>
                    <div class="td-header">
                        <h3>Transfer Manager</h3>
                        <button class="fb-icon-btn" onClick={props.onClose}>✕</button>
                    </div>

                    <div class="td-content">
                        <section class="td-section">
                            <h4 class="td-section-title">Active ({activeTransfers().length})</h4>
                            <div class="td-list">
                                <For each={activeTransfers()}>
                                    {(transfer) => (
                                        <div class="td-item">
                                            <div class="td-item-info">
                                                <span class="td-item-icon">{transfer.type === 'upload' ? '↑' : '↓'}</span>
                                                <div class="td-item-details">
                                                    <span class="td-item-name">{transfer.name}</span>
                                                    <span class="td-item-meta">
                                                        {formatSize(transfer.size)} • {transfer.speed || 'Calculating...'}
                                                    </span>
                                                </div>
                                            </div>
                                            <div class="td-progress-container">
                                                <div class="td-progress-bar">
                                                    <div class="td-progress-fill" style={{ width: `${transfer.progress}%` }}></div>
                                                </div>
                                                <span class="td-progress-text">{transfer.progress}%</span>
                                            </div>
                                        </div>
                                    )}
                                </For>
                                <Show when={activeTransfers().length === 0}>
                                    <div class="td-empty">No active transfers</div>
                                </Show>
                            </div>
                        </section>

                        <Show when={completedTransfers().length > 0}>
                            <section class="td-section">
                                <h4 class="td-section-title">Completed</h4>
                                <div class="td-list">
                                    <For each={completedTransfers()}>
                                        {(transfer) => (
                                            <div class={`td-item completed ${transfer.status}`}>
                                                <div class="td-item-info">
                                                    <span class="td-item-icon">{transfer.status === 'completed' ? '✓' : '✕'}</span>
                                                    <div class="td-item-details">
                                                        <span class="td-item-name">{transfer.name}</span>
                                                        <Show when={transfer.error}>
                                                            <span class="td-item-error">{transfer.error}</span>
                                                        </Show>
                                                    </div>
                                                </div>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </section>
                        </Show>
                    </div>
                </div>
            </div>
        </Show>
    );
};
