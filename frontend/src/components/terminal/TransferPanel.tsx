import { Component, createSignal, onCleanup, For, Show } from 'solid-js';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { app } from '../../../wailsjs/go/models';
import { CancelTransfer, ClearTransfers } from '../../../wailsjs/go/app/SSHService';
import '../../styles/filebrowser.css'; // Reuse some existing styles

export const TransferPanel: Component = () => {
    const [transfers, setTransfers] = createSignal<app.TransferJob[]>([]);
    const [isOpen, setIsOpen] = createSignal(false);

    // Listen for real-time progress events
    EventsOn('sftp_transfer_update', (job: app.TransferJob) => {
        setTransfers(prev => {
            const idx = prev.findIndex(t => t.id === job.id);
            if (idx >= 0) {
                const newArr = [...prev];
                newArr[idx] = job;
                return newArr;
            }
            return [...prev, job];
        });

        // Auto-open panel on new transfer
        if (job.status === 'in_progress' || job.status === 'queued') {
            setIsOpen(true);
        }
    });

    EventsOn('sftp_transfers_list', (jobs: app.TransferJob[]) => {
        setTransfers(jobs);
    });

    onCleanup(() => {
        EventsOff('sftp_transfer_update');
        EventsOff('sftp_transfers_list');
    });

    const activeCount = () => transfers().filter(t => t.status === 'in_progress' || t.status === 'queued').length;

    const formatSize = (bytes: number) => {
        if (!bytes || bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const handleCancel = async (id: string) => {
        try {
            await CancelTransfer(id);
        } catch (err) {
            console.error("Failed to cancel transfer", err);
        }
    };

    const handleClear = async () => {
        try {
            await ClearTransfers();
            setTransfers(prev => prev.filter(t => t.status === 'in_progress' || t.status === 'queued'));
        } catch (err) {
            console.error("Failed to clear transfers", err);
        }
    };

    return (
        <Show when={transfers().length > 0}>
            <div class={`transfer-panel-wrapper ${isOpen() ? 'open' : 'closed'}`} style={{
                position: 'absolute',
                bottom: '16px',
                right: '16px',
                width: '350px',
                background: 'var(--bg-surface)',
                border: '1px solid var(--border-primary)',
                'border-radius': '8px',
                'box-shadow': '0 8px 16px rgba(0,0,0,0.4)',
                'z-index': 1000,
                display: 'flex',
                'flex-direction': 'column',
                transition: 'transform 0.3s ease, opacity 0.3s ease',
                transform: isOpen() ? 'translateY(0)' : 'translateY(calc(100% - 40px))' // Show only header when closed
            }}>
                {/* Header Toggle */}
                <div
                    class="transfer-header"
                    onClick={() => setIsOpen(!isOpen())}
                    style={{
                        padding: '10px 16px',
                        background: 'var(--bg-primary)',
                        'border-bottom': '1px solid var(--border-primary)',
                        'border-top-left-radius': '8px',
                        'border-top-right-radius': '8px',
                        display: 'flex',
                        'justify-content': 'space-between',
                        'align-items': 'center',
                        cursor: 'pointer',
                        'user-select': 'none'
                    }}
                >
                    <span style={{ 'font-size': '13px', 'font-weight': 'bold' }}>
                        File Transfers {activeCount() > 0 ? `(${activeCount()} Active)` : ''}
                    </span>
                    <div style={{ display: 'flex', gap: '8px', 'align-items': 'center' }}>
                        <Show when={isOpen() && transfers().length > 0}>
                            <button
                                class="action-btn"
                                style={{ padding: '2px 6px', 'font-size': '11px' }}
                                onClick={(e) => { e.stopPropagation(); handleClear(); }}
                            >
                                Clear Done
                            </button>
                        </Show>
                        <span style={{ 'font-size': '14px', transition: 'transform 0.3s' }}>
                            {isOpen() ? '▼' : '▲'}
                        </span>
                    </div>
                </div>

                {/* Content Body */}
                <div style={{
                    'max-height': '300px',
                    'overflow-y': 'auto',
                    padding: '8px',
                    opacity: isOpen() ? 1 : 0,
                    'pointer-events': isOpen() ? 'auto' : 'none'
                }}>
                    <For each={transfers()}>
                        {(job) => {
                            const pct = job.total_bytes > 0 ? Math.min(100, (job.bytes_copied / job.total_bytes) * 100) : 0;
                            const isFinished = job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled';

                            let statusColor = 'var(--text-accent)';
                            let icon = '⏳';
                            if (job.status === 'in_progress') { statusColor = 'var(--primary-color)'; icon = job.type === 'download' ? '↓' : '↑'; }
                            if (job.status === 'completed') { statusColor = 'var(--success-color)'; icon = '✓'; }
                            if (job.status === 'failed') { statusColor = 'var(--error-color)'; icon = '✕'; }
                            if (job.status === 'cancelled') { statusColor = 'var(--text-secondary)'; icon = '⏹'; }

                            return (
                                <div style={{
                                    background: 'var(--bg-primary)',
                                    border: '1px solid var(--border-primary)',
                                    'border-radius': '6px',
                                    padding: '10px',
                                    'margin-bottom': '8px'
                                }}>
                                    <div style={{ display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'margin-bottom': '6px' }}>
                                        <div style={{ display: 'flex', 'align-items': 'center', gap: '6px', overflow: 'hidden' }}>
                                            <span style={{ 'font-size': '12px', color: statusColor }}>{icon}</span>
                                            <span style={{ 'font-size': '12px', 'font-weight': '500', 'white-space': 'nowrap', 'text-overflow': 'ellipsis', overflow: 'hidden', 'max-width': '200px' }} title={job.filename}>
                                                {job.filename}
                                            </span>
                                        </div>
                                        <Show when={!isFinished}>
                                            <button
                                                class="action-btn danger"
                                                style={{ padding: '0px 4px', 'font-size': '10px' }}
                                                onClick={() => handleCancel(job.id)}
                                                title="Cancel Transfer"
                                            >✕</button>
                                        </Show>
                                    </div>

                                    <div style={{ width: '100%', height: '4px', background: 'var(--bg-surface)', 'border-radius': '2px', overflow: 'hidden', 'margin-bottom': '6px' }}>
                                        <div style={{
                                            width: `${job.status === 'completed' ? 100 : pct}%`,
                                            height: '100%',
                                            background: statusColor,
                                            transition: 'width 0.3s ease'
                                        }}></div>
                                    </div>

                                    <div style={{ display: 'flex', 'justify-content': 'space-between', 'font-size': '10px', color: 'var(--text-secondary)' }}>
                                        <Show
                                            when={!isFinished}
                                            fallback={<span>{job.status === 'completed' ? formatSize(job.total_bytes) : job.status}</span>}
                                        >
                                            <span>{formatSize(job.bytes_copied)} / {formatSize(job.total_bytes)}</span>
                                            <Show when={job.speed_bytes_s > 0}>
                                                <span>{formatSize(job.speed_bytes_s)}/s</span>
                                            </Show>
                                        </Show>
                                    </div>
                                    <Show when={job.error}>
                                        <div style={{ 'font-size': '10px', color: 'var(--error-color)', 'margin-top': '4px', 'white-space': 'nowrap', 'text-overflow': 'ellipsis', overflow: 'hidden' }} title={job.error}>
                                            {job.error}
                                        </div>
                                    </Show>
                                </div>
                            );
                        }}
                    </For>
                </div>
            </div>
        </Show>
    );
};
