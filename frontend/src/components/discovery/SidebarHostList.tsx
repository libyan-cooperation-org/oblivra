import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { DiscoverAll as DiscoverNetwork } from '../../../wailsjs/go/app/DiscoveryService';
import { discovery } from '../../../wailsjs/go/models';
import { useApp } from '@core/store';

export const SidebarHostList: Component = () => {
    const [state, actions] = useApp();
    const [networkHosts, setNetworkHosts] = createSignal<discovery.DiscoveredHost[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [searchQuery, setSearchQuery] = createSignal('');

    const filteredHosts = () => {
        const query = searchQuery().toLowerCase();
        if (!query) return networkHosts();
        return networkHosts().filter(h =>
            (h.name && h.name.toLowerCase().includes(query)) ||
            (h.address && h.address.toLowerCase().includes(query)) ||
            (h.username && h.username.toLowerCase().includes(query))
        );
    };

    const loadData = async () => {
        setLoading(true);
        try {
            const hosts = await DiscoverNetwork();
            setNetworkHosts(hosts || []);
        } catch (err) {
            console.error('Failed to load discovery data:', err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadData();
    });

    const handleConnect = (host: discovery.DiscoveredHost) => {
        const id = host.address;
        const exists = state.hosts.find(h => h.id === id || h.hostname === id || h.label === id);
        if (exists) {
            actions.connectToHost(exists.id);
            actions.setActiveNavTab('terminal');
        } else {
            actions.notify(`Discovered ${host.name || id}. Open 'Add Host' (Cmd+N) to configure credentials first.`, 'info');
        }
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%; background: var(--bg-secondary);">
            <div style="padding: 12px 14px; border-bottom: 1px solid var(--border-primary); background: var(--bg-tertiary);">
                <div class="search-box">
                    <span class="search-icon">🔍</span>
                    <input
                        type="text"
                        placeholder="Search network hosts..."
                        value={searchQuery()}
                        onInput={(e) => setSearchQuery(e.currentTarget.value)}
                    />
                </div>
            </div>

            <style>{`
                .search-box {
                    position: relative;
                    display: flex;
                    align-items: center;
                }
                .search-icon {
                    position: absolute;
                    left: 10px;
                    font-size: 12px;
                    opacity: 0.5;
                    pointer-events: none;
                }
                .search-box input {
                    width: 100%;
                    background: var(--bg-surface);
                    border: 1px solid var(--border-subtle);
                    border-radius: var(--radius-md);
                    padding: 8px 10px 8px 30px;
                    color: var(--text-primary);
                    font-size: var(--font-size-sm);
                    outline: none;
                    transition: all var(--transition-fast);
                }
                .search-box input:focus {
                    border-color: var(--accent-primary);
                    box-shadow: 0 0 0 2px var(--accent-glow);
                }
                .search-box input::placeholder {
                    color: var(--text-muted);
                }
                .host-item {
                    padding: 12px 14px;
                    margin: 4px 8px;
                    border-radius: var(--radius-sm);
                    border: 1px solid transparent;
                    background: transparent;
                    cursor: pointer;
                    transition: all var(--transition-smooth);
                    display: flex;
                    align-items: center;
                    gap: 12px;
                }
                .host-item:hover {
                    background: var(--bg-hover);
                    border-color: var(--border-hover);
                    transform: translateY(-1px);
                }
                .host-item .status-pulse {
                    width: 8px;
                    height: 8px;
                    border-radius: 50%;
                    background: var(--success);
                    box-shadow: 0 0 8px rgba(0, 240, 192, 0.4);
                    flex-shrink: 0;
                }
                .host-item-info {
                    flex: 1;
                    min-width: 0;
                    display: flex;
                    flex-direction: column;
                    gap: 2px;
                }
                .host-item-title {
                    font-weight: 600;
                    font-size: 13px;
                    color: var(--text-primary);
                    white-space: nowrap;
                    overflow: hidden;
                    text-overflow: ellipsis;
                }
                .host-item-meta {
                    font-size: 11px;
                    color: var(--text-muted);
                    font-family: var(--font-mono);
                }
            `}</style>

            <div style="flex: 1; overflow-y: auto; padding-top: 8px;">
                <Show when={loading()}>
                    <div style="text-align: center; color: var(--text-muted); font-size: 12px; padding: 20px; animation: pulse 1.5s infinite;">
                        Scanning local network...
                    </div>
                </Show>

                <Show when={!loading()}>
                    <For each={filteredHosts()} fallback={<div style="color: var(--text-muted); font-size: 12px; text-align: center; padding: 20px;">No matching hosts found.</div>}>
                        {(host) => (
                            <div class="host-item" onClick={() => handleConnect(host)}>
                                <div class="status-pulse" />
                                <div class="host-item-info">
                                    <span class="host-item-title">{host.name || host.address}</span>
                                    <span class="host-item-meta">{host.source} • {host.username}</span>
                                </div>
                                <span style="color: var(--text-muted); font-size: 10px;">→</span>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
