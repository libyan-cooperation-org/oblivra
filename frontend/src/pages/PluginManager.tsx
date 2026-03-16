import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetPlugins, Activate, Deactivate, Refresh } from '../../wailsjs/go/services/PluginService';
import { plugin } from '../../wailsjs/go/models';
import { EmptyState } from '../components/ui/EmptyState';

export const PluginManager: Component = () => {
    const pluginsState = createSignal<plugin.Plugin[]>([]);
    const [plugins, setPlugins] = pluginsState;
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    const loadPlugins = async () => {
        setLoading(true);
        setError(null);
        try {
            await Refresh();
            const list = await GetPlugins();
            setPlugins(list || []);
        } catch (err) {
            setError((err as Error).message || String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadPlugins();
    });

    const togglePlugin = async (p: plugin.Plugin) => {
        if (!p.manifest) return;
        try {
            if (p.state === 'active') {
                await Deactivate(p.manifest.id);
            } else {
                await Activate(p.manifest.id);
            }
            await loadPlugins();
        } catch (err) {
            setError(`Failed to toggle ${p.manifest.name}: ${(err as Error).message || String(err)}`);
        }
    };

    return (
        <div class="page-container">
            <div class="page-header">
                <div>
                    <h2>Plugin Management</h2>
                    <p class="description">
                        Extend OblivraShell capabilities using embedded Lua scripts.
                    </p>
                </div>
                <button
                    class="action-btn primary"
                    onClick={loadPlugins}
                    disabled={loading()}
                >
                    {loading() ? 'Refreshing...' : 'Refresh Registry'}
                </button>
            </div>

            <Show when={error()}>
                <div class="alert error">
                    {error()}
                </div>
            </Show>

            <Show when={plugins().length === 0 && !loading()}>
                <EmptyState
                    icon="🧩"
                    title="No Plugins Found"
                    description="Place Lua manifest folders in ~/.oblivrashell/plugins/ to extend functionality."
                />
            </Show>

            <div class="plugin-grid grid-350">
                <For each={plugins()}>
                    {(p) => (
                        <div class="plugin-card card-bg">
                            <div class="card-header">
                                <div>
                                    <h3>
                                        {p.manifest?.name || 'Unknown Plugin'}
                                        <Show when={p.manifest?.version}>
                                            <span class="version"> v{p.manifest!.version}</span>
                                        </Show>
                                    </h3>
                                    <span class="author">{p.manifest?.author || 'Unknown Author'}</span>
                                </div>
                                <span class={`status-badge ${p.state}`}>
                                    {p.state}
                                </span>
                            </div>

                            <p class="description-text">
                                {p.manifest?.description || 'No description available.'}
                            </p>

                            <Show when={p.error}>
                                <div class="error-msg">
                                    {p.error}
                                </div>
                            </Show>

                            <div class="plugin-actions border-top">
                                <button
                                    class={`action-btn ${p.state === 'active' ? 'danger' : 'primary'}`}
                                    onClick={() => togglePlugin(p)}
                                    disabled={p.state === 'error'}
                                >
                                    {p.state === 'active' ? 'Deactivate' : 'Activate'}
                                </button>
                            </div>
                        </div>
                    )}
                </For>
            </div>
        </div>
    );
};
