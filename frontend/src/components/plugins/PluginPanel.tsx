import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const PluginPanel: Component = () => {
    const [plugins, setPlugins] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { GetPlugins } = await import('../../../wailsjs/go/app/PluginService');
            setPlugins(await GetPlugins() || []);
        } catch (e) { console.error('Plugins load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const toggle = async (id: string, active: boolean) => {
        try {
            const { Activate, Deactivate } = await import('../../../wailsjs/go/app/PluginService');
            if (active) await Deactivate(id); else await Activate(id);
            await reload();
        } catch (e) { console.error('Plugin toggle:', e); }
    };

    return (
        <div class="panel-container flex-column h-full">
            <div class="panel-header drawer-header">
                <span class="panel-title drawer-title">Plugins</span>
                <button class="action-btn sm" onClick={async () => { const { Refresh } = await import('../../../wailsjs/go/app/PluginService'); await Refresh(); await reload(); }}>↻ Refresh</button>
            </div>
            <div class="panel-content flex-1 overflow-y p-8">
                <Show when={loading()}><div class="placeholder-text">Loading...</div></Show>
                <Show when={!loading()}>
                    <For each={plugins()} fallback={<div class="placeholder-text">No plugins installed.</div>}>
                        {(p) => (
                            <div class="plugin-item-compact card-bg-tertiary border-primary mb-6 p-10">
                                <div class="flex-between align-center">
                                    <div>
                                        <div class="plugin-name-sm font-500 text-primary">🧩 {p.name || p.id}</div>
                                        <div class="plugin-meta-xs text-muted mt-2">{p.description || ''}</div>
                                        <Show when={p.active}>
                                            <div class="resource-stats-xs mt-4 flex gap-8">
                                                <span class="stat cpu">CPU: {p.cpu_usage || '0'}%</span>
                                                <span class="stat mem">MEM: {p.mem_usage || '0'}MB</span>
                                            </div>
                                        </Show>
                                    </div>
                                    <button
                                        onClick={() => toggle(p.id, p.active)}
                                        class={`status-pill-sm ${p.active ? 'active' : 'inactive'}`}
                                    >
                                        {p.active ? 'Active' : 'Inactive'}
                                    </button>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
