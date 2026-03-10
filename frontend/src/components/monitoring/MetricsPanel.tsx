import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const MetricsPanel: Component = () => {
    const [metrics, setMetrics] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { GetAllMetrics } = await import('../../../wailsjs/go/app/MetricsService');
            setMetrics(await GetAllMetrics() || []);
        } catch (e) { console.error('Metrics load:', e); }
        setLoading(false);
    };

    onMount(reload);

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div class="drawer-header">
                <span class="drawer-title">Metrics & Analytics</span>
                <button class="action-btn" style="padding: 2px 8px; font-size: 10px;" onClick={reload}>↻</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading()}>
                    <For each={metrics()} fallback={<div class="placeholder">No metrics collected yet. Metrics are recorded as you use the app.</div>}>
                        {(m) => (
                            <div style="display: flex; justify-content: space-between; align-items: center; padding: 6px 8px; border-bottom: 1px solid var(--border-subtle); font-size: 11px;">
                                <div>
                                    <div style="color: var(--text-primary);">{m.name}</div>
                                    <div style="font-size: 10px; color: var(--text-muted);">{m.type || 'counter'}{m.labels ? ` • ${JSON.stringify(m.labels)}` : ''}</div>
                                </div>
                                <span style="font-family: var(--font-mono); color: var(--accent-primary); font-size: 12px; font-weight: 600;">
                                    {typeof m.value === 'number' ? m.value.toLocaleString() : m.value || '0'}
                                </span>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
