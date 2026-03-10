import { Component, createSignal, onMount, onCleanup, Show, For, createEffect } from 'solid-js';
import { useApp } from '@core/store';

// Represents a single executable action inside the command palette
interface CommandAction {
    id: string;
    title: string;
    description: string;
    icon: string;
    group: string;
    action: () => void;
}

export const CommandPalette: Component = () => {
    const [state, actions] = useApp();
    const [isOpen, setIsOpen] = createSignal(false);
    const [query, setQuery] = createSignal('');
    const [selectedIndex, setSelectedIndex] = createSignal(0);
    const [results, setResults] = createSignal<CommandAction[]>([]);

    let inputRef!: HTMLInputElement;

    // Build the dynamic list of available commands based on app state
    const buildCommands = (): CommandAction[] => {
        const commands: CommandAction[] = [];

        // App Navigation
        commands.push({
            id: 'nav-dashboard', title: 'Dashboard', description: 'Go to Fleet Overview',
            icon: '📊', group: 'Navigation',
            action: () => actions.setActiveNavTab('dashboard' as any)
        });
        commands.push({
            id: 'nav-siem', title: 'SIEM Investigation', description: 'Analyze logs and security events',
            icon: '🛡️', group: 'Navigation',
            action: () => actions.setActiveNavTab('siem' as any)
        });
        commands.push({
            id: 'nav-threathunter', title: 'Threat Hunter', description: 'Proactive UEBA profiling workspace',
            icon: '🎯', group: 'Navigation',
            action: () => actions.setActiveNavTab('security' as any)
        });
        commands.push({
            id: 'nav-sessions', title: 'Terminal Workspace', description: 'MultiExec Terminal Panel',
            icon: '💻', group: 'Navigation',
            action: () => actions.setActiveNavTab('sessions' as any)
        });

        // Fast Host Connections
        state.hosts.forEach((h: any) => {
            commands.push({
                id: `connect-${h.id}`,
                title: `Connect to ${h.label || h.hostname}`,
                description: `${h.username}@${h.hostname}`,
                icon: '⚡', group: 'Hosts',
                action: () => {
                    actions.connectToHost(h.id);
                    actions.setActiveNavTab('sessions' as any);
                }
            });
        });

        // System Actions
        commands.push({
            id: 'sys-metrics', title: 'System Metrics', description: 'View system health and analytics',
            icon: '📈', group: 'System',
            action: () => actions.setActiveNavTab('metrics' as any)
        });

        return commands;
    };

    // Filter logic
    createEffect(() => {
        const q = query().toLowerCase();
        let filtered = buildCommands();
        if (q) {
            filtered = filtered.filter(cmd =>
                cmd.title.toLowerCase().includes(q) ||
                cmd.description.toLowerCase().includes(q) ||
                cmd.group.toLowerCase().includes(q)
            );
        }
        setResults(filtered);
        setSelectedIndex(0); // Reset index on new search
    });

    const handleKeyDown = (e: KeyboardEvent) => {
        if (!isOpen()) {
            // Toggle on Ctrl+K or Cmd+K
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                setIsOpen(true);
                setTimeout(() => inputRef?.focus(), 50);
            }
            return;
        }

        switch (e.key) {
            case 'Escape':
                e.preventDefault();
                setIsOpen(false);
                break;
            case 'ArrowDown':
                e.preventDefault();
                setSelectedIndex(i => Math.min(i + 1, results().length - 1));
                break;
            case 'ArrowUp':
                e.preventDefault();
                setSelectedIndex(i => Math.max(i - 1, 0));
                break;
            case 'Enter':
                e.preventDefault();
                const activeCmd = results()[selectedIndex()];
                if (activeCmd) {
                    activeCmd.action();
                    setIsOpen(false);
                    setQuery('');
                }
                break;
        }
    };

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
    });

    onCleanup(() => {
        window.removeEventListener('keydown', handleKeyDown);
    });

    // Group results for rendering
    const groupedResults = () => {
        const groups: Record<string, CommandAction[]> = {};
        results().forEach(cmd => {
            if (!groups[cmd.group]) groups[cmd.group] = [];
            groups[cmd.group].push(cmd);
        });
        return groups;
    };

    return (
        <Show when={isOpen()}>
            <div class="cmd-palette-overlay" onClick={(e) => {
                if (e.target.className === 'cmd-palette-overlay') setIsOpen(false);
            }}>
                <div class="cmd-palette-container" onClick={e => e.stopPropagation()}>
                    <div class="cmd-palette-header">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
                        <input
                            ref={inputRef}
                            type="text"
                            class="cmd-palette-input"
                            placeholder="Type a command or search for a host..."
                            value={query()}
                            onInput={e => setQuery(e.currentTarget.value)}
                        />
                        <span class="cmd-palette-badge">ESC</span>
                    </div>

                    <div class="cmd-palette-content">
                        <Show when={results().length === 0}>
                            <div class="cmd-palette-empty">
                                No commands found matching "{query()}"
                            </div>
                        </Show>

                        <For each={Object.entries(groupedResults())}>
                            {([groupName, groupCommands]) => {
                                // We need to calculate global index for selection highlighting
                                const getGlobalIndex = (cmdId: string) => results().findIndex(c => c.id === cmdId);

                                return (
                                    <div class="cmd-group">
                                        <div class="cmd-group-label">{groupName}</div>
                                        <For each={groupCommands}>
                                            {(cmd) => {
                                                const isActive = () => getGlobalIndex(cmd.id) === selectedIndex();
                                                return (
                                                    <div
                                                        class={`cmd-item ${isActive() ? 'active' : ''}`}
                                                        onClick={() => {
                                                            cmd.action();
                                                            setIsOpen(false);
                                                            setQuery('');
                                                        }}
                                                        onMouseEnter={() => setSelectedIndex(getGlobalIndex(cmd.id))}
                                                    >
                                                        <div class="cmd-item-icon">
                                                            {cmd.icon}
                                                        </div>
                                                        <div class="cmd-item-body">
                                                            <div class="cmd-item-title">{cmd.title}</div>
                                                            <div class="cmd-item-desc">{cmd.description}</div>
                                                        </div>
                                                        <Show when={isActive()}>
                                                            <div class="cmd-item-hint">↵</div>
                                                        </Show>
                                                    </div>
                                                );
                                            }}
                                        </For>
                                    </div>
                                );
                            }}
                        </For>
                    </div>

                    <div class="cmd-palette-footer">
                        <div class="cmd-kbd-group">
                            <span class="cmd-kbd">↑</span>
                            <span class="cmd-kbd">↓</span>
                            <span>to navigate</span>
                        </div>
                        <div class="cmd-kbd-group">
                            <span class="cmd-kbd">↵</span>
                            <span>to execute</span>
                        </div>
                    </div>
                </div>
            </div>
        </Show>
    );
};
