import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { useApp } from '@core/store';

export const QuickSwitcher: Component = () => {
    const [state, actions] = useApp();
    const [isOpen, setIsOpen] = createSignal(false);
    const [query, setQuery] = createSignal('');
    const [selectedIndex, setSelectedIndex] = createSignal(0);

    const filteredHosts = () => {
        const q = query().toLowerCase();
        if (!q) return state.hosts.slice(0, 8);
        return state.hosts.filter(h =>
            h.label.toLowerCase().includes(q) ||
            h.hostname.toLowerCase().includes(q)
        ).slice(0, 8);
    };

    onMount(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Cmd+T or Ctrl+T for Quick Switcher
            if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 't') {
                e.preventDefault();
                setIsOpen(true);
                setQuery('');
                setSelectedIndex(0);
            }
            if (isOpen()) {
                if (e.key === 'Escape') setIsOpen(false);
                if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    setSelectedIndex((s) => Math.min(s + 1, filteredHosts().length - 1));
                }
                if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    setSelectedIndex((s) => Math.max(s - 1, 0));
                }
                if (e.key === 'Enter') {
                    const host = filteredHosts()[selectedIndex()];
                    if (host) {
                        actions.connectToHost(host.id);
                        setIsOpen(false);
                    }
                }
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    });

    return (
        <Show when={isOpen()}>
            <div
                class="quick-switcher-overlay"
                onClick={() => setIsOpen(false)}
            >
                <div
                    class="quick-switcher-modal glass-surface"
                    onClick={(e) => e.stopPropagation()}
                >
                    <div class="qs-search">
                        <span class="qs-icon">🔍</span>
                        <input
                            type="text"
                            placeholder="Jump to host..."
                            value={query()}
                            onInput={(e) => {
                                setQuery(e.currentTarget.value);
                                setSelectedIndex(0);
                            }}
                            autofocus
                        />
                        <kbd class="qs-kbd">ESC</kbd>
                    </div>

                    <div class="qs-results">
                        <For each={filteredHosts()}>
                            {(host, i) => (
                                <div
                                    class={`qs-item ${selectedIndex() === i() ? 'selected' : ''}`}
                                    onMouseEnter={() => setSelectedIndex(i())}
                                    onClick={() => {
                                        actions.connectToHost(host.id);
                                        setIsOpen(false);
                                    }}
                                >
                                    <span class="qs-host-label">{host.label}</span>
                                    <span class="qs-host-addr">{host.hostname}</span>
                                </div>
                            )}
                        </For>
                        <Show when={filteredHosts().length === 0}>
                            <div class="qs-no-results">No hosts matching "{query()}"</div>
                        </Show>
                    </div>
                </div>
            </div>
        </Show>
    );
};
