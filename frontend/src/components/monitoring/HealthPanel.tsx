import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { EmptyState } from '../ui/EmptyState';

export const HealthPanel: Component = () => {
    type HealthData = { status?: string, cpu_percent?: number, memory_percent?: number, disk_percent?: number, load_avg?: number, uptime?: string };
    const [health, setHealth] = createSignal<Record<string, HealthData>>({});
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { GetAllHealth } = await import('../../../wailsjs/go/app/HealthService');
            setHealth((await GetAllHealth() || {}) as Record<string, HealthData>);
        } catch (e) { console.error('Health load:', e); }
        setLoading(false);
    };

    onMount(reload);

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div class="drawer-header">
                <span class="drawer-title">Host Health</span>
                <button class="action-btn" style="padding: 2px 8px; font-size: 10px;" onClick={reload}>↻</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading()}>
                    <For each={Object.entries(health())} fallback={<EmptyState icon="📊" title="No Hosts Monitored" description="Connect to a host to start health monitoring. CPU, memory, disk, and network metrics will appear here." compact />}>
                        {([hostId, h]) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 10px; margin-bottom: 6px;">
                                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px;">
                                    <span style="font-size: 12px; font-weight: 500; color: var(--text-primary);">{hostId}</span>
                                    <span class={`status-dot ${h.status === 'healthy' ? 'online' : 'offline'}`} style="width: 8px; height: 8px; border-radius: 50%;" />
                                </div>
                                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 4px;">
                                    <div style="font-size: 10px; color: var(--text-muted);">CPU: <span style="color: var(--text-primary);">{h.cpu_percent?.toFixed(1) || '—'}%</span></div>
                                    <div style="font-size: 10px; color: var(--text-muted);">Mem: <span style="color: var(--text-primary);">{h.memory_percent?.toFixed(1) || '—'}%</span></div>
                                    <div style="font-size: 10px; color: var(--text-muted);">Disk: <span style="color: var(--text-primary);">{h.disk_percent?.toFixed(1) || '—'}%</span></div>
                                    <div style="font-size: 10px; color: var(--text-muted);">Load: <span style="color: var(--text-primary);">{h.load_avg || '—'}</span></div>
                                </div>
                                <Show when={h.uptime}><div style="font-size: 10px; color: var(--text-muted); margin-top: 4px;">Uptime: {h.uptime}</div></Show>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
