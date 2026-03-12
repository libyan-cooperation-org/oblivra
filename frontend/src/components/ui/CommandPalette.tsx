import {
    Component,
    createSignal,
    createMemo,
    For,
    Show,
    onMount,
    onCleanup,
    createEffect,
} from 'solid-js';
import { useApp } from '../../core/store';
import { GenerateCommand } from '../../../wailsjs/go/app/AIService';
import { List as ListSnippets } from '../../../wailsjs/go/app/SnippetService';
import { JSX } from 'solid-js';
import '../../styles/palette.css';

// Action types for the palette
type ActionCategory =
    | 'connection'
    | 'terminal'
    | 'navigation'
    | 'tools'
    | 'security'
    | 'appearance'
    | 'hosts'
    | 'snippets'
    | 'recent'
    | 'ai';

interface Snippet {
    id: string;
    title: string;
    description?: string;
    command: string;
    tags?: string[];
}

interface PaletteAction {
    id: string;
    label: string;
    description?: string;
    icon: string;
    category: ActionCategory;
    shortcut?: string;
    keywords?: string[];
    action: () => void;
    dangerous?: boolean;
}

interface CommandPaletteProps {
    open: boolean;
    onClose: () => void;
    onConnect: (hostId: string) => void;
    onNavigate: (page: string) => void;
}

export const CommandPalette: Component<CommandPaletteProps> = (props) => {
    const [state, actions] = useApp();
    const [query, setQuery] = createSignal('');
    const [selectedIndex, setSelectedIndex] = createSignal(0);
    const [mode, setMode] = createSignal<'commands' | 'hosts' | 'snippets' | 'files' | 'ai'>('commands');
    const [snippets, setSnippets] = createSignal<Snippet[]>([]);
    const [aiSuggesting, setAiSuggesting] = createSignal(false);
    const [aiResult, setAiResult] = createSignal<string | null>(null);
    let inputRef: HTMLInputElement | undefined;
    let listRef: HTMLDivElement | undefined;

    const loadSnippets = async () => {
        try {
            const res = await ListSnippets();
            setSnippets(res || []);
        } catch (err) {
            console.error("Failed to load snippets for palette:", err);
        }
    };

    onMount(() => {
        loadSnippets();
    });

    // Build all available actions
    const allActions = createMemo((): PaletteAction[] => {
        const items: PaletteAction[] = [
            // Connection actions
            {
                id: 'new-connection',
                label: 'New Connection',
                description: 'Open SSH connection dialog',
                icon: '🔗',
                category: 'connection',
                shortcut: 'Ctrl+N',
                keywords: ['connect', 'ssh', 'new', 'open'],
                action: () => props.onNavigate('new-connection'),
            },
            {
                id: 'new-local-terminal',
                label: 'New Local Terminal',
                description: 'Open a local PowerShell session',
                icon: '💻',
                category: 'terminal',
                shortcut: 'Ctrl+Shift+L',
                keywords: ['local', 'shell', 'terminal', 'powershell', 'cmd'],
                action: () => {
                    props.onClose();
                    actions.connectToLocal();
                },
            },
            {
                id: 'quick-connect',
                label: 'Quick Connect',
                description: 'Connect with user@host:port',
                icon: '⚡',
                category: 'connection',
                shortcut: 'Ctrl+Shift+N',
                keywords: ['quick', 'fast', 'connect'],
                action: () => {
                    props.onClose();
                    // Focus quick connect input
                    document.querySelector<HTMLInputElement>('.quick-connect-input input')?.focus();
                },
            },
            {
                id: 'disconnect-current',
                label: 'Disconnect Current Session',
                icon: '🔌',
                category: 'connection',
                shortcut: 'Ctrl+W',
                keywords: ['disconnect', 'close', 'end'],
                action: () => {
                    if (state.activeSessionId) {
                        // SSHService.Disconnect(state.activeSessionId);
                    }
                },
            },
            {
                id: 'disconnect-all',
                label: 'Disconnect All Sessions',
                icon: '⛔',
                category: 'connection',
                keywords: ['disconnect', 'close', 'all'],
                dangerous: true,
                action: () => {
                    // SSHService.CloseAll();
                },
            },
            {
                id: 'import-ssh-config',
                label: 'Import from SSH Config',
                description: 'Import hosts from ~/.ssh/config',
                icon: '📥',
                category: 'connection',
                keywords: ['import', 'ssh', 'config'],
                action: () => {
                    // HostService.ImportSSHConfig();
                },
            },
            {
                id: 'discover-hosts',
                label: 'Discover Hosts',
                description: 'Scan for hosts from multiple sources',
                icon: '🔍',
                category: 'connection',
                keywords: ['discover', 'scan', 'find', 'terraform', 'ansible'],
                action: () => props.onNavigate('discover'),
            },

            // Navigation
            {
                id: 'toggle-sidebar',
                label: 'Toggle Sidebar',
                icon: '📋',
                category: 'navigation',
                shortcut: 'Ctrl+B',
                keywords: ['sidebar', 'toggle', 'hide', 'show'],
                action: () => actions.toggleSidebar(),
            },
            {
                id: 'go-settings',
                label: 'Open Settings',
                icon: '⚙️',
                category: 'navigation',
                shortcut: 'Ctrl+,',
                keywords: ['settings', 'preferences', 'config'],
                action: () => props.onNavigate('settings'),
            },
            {
                id: 'go-dashboard',
                label: 'Dashboard',
                icon: '🏠',
                category: 'navigation',
                keywords: ['home', 'dashboard', 'main', 'analytics'],
                action: () => props.onNavigate('dashboard'),
            },
            {
                id: 'go-siem',
                label: 'SIEM Panel',
                icon: '🛡️',
                category: 'navigation',
                keywords: ['siem', 'security', 'logs', 'audit'],
                action: () => props.onNavigate('siem'),
            },
            {
                id: 'go-compliance',
                label: 'Compliance Center',
                icon: '⚖️',
                category: 'navigation',
                keywords: ['compliance', 'soc2', 'pci', 'audit', 'governance'],
                action: () => props.onNavigate('compliance'),
            },
            {
                id: 'go-hosts',
                label: 'Host Manager',
                icon: '🖥️',
                category: 'navigation',
                keywords: ['hosts', 'servers', 'manage'],
                action: () => props.onNavigate('hosts'),
            },
            {
                id: 'go-vault',
                label: 'Credential Vault',
                icon: '🔐',
                category: 'navigation',
                keywords: ['vault', 'credentials', 'passwords', 'keys'],
                action: () => props.onNavigate('vault'),
            },
            {
                id: 'go-sessions',
                label: 'Session History',
                icon: '📜',
                category: 'navigation',
                keywords: ['sessions', 'history', 'audit', 'logs'],
                action: () => props.onNavigate('sessions'),
            },
            {
                id: 'go-health',
                label: 'Health Dashboard',
                icon: '💚',
                category: 'navigation',
                keywords: ['health', 'monitoring', 'status'],
                action: () => props.onNavigate('health'),
            },

            // Tools
            {
                id: 'open-sftp',
                label: 'Open SFTP Browser',
                description: 'File browser for current session',
                icon: '📁',
                category: 'tools',
                shortcut: 'Ctrl+Shift+E',
                keywords: ['sftp', 'files', 'browser', 'upload', 'download'],
                action: () => { /* open sftp */ },
            },
            {
                id: 'open-tunnels',
                label: 'Tunnel Manager',
                description: 'Manage port forwarding tunnels',
                icon: '🚇',
                category: 'tools',
                keywords: ['tunnel', 'port', 'forward', 'socks', 'proxy'],
                action: () => props.onNavigate('tunnels'),
            },
            {
                id: 'open-snippets',
                label: 'Command Snippets',
                icon: '📝',
                category: 'tools',
                shortcut: 'Ctrl+Shift+S',
                keywords: ['snippets', 'commands', 'saved', 'macros'],
                action: () => props.onNavigate('snippets'),
            },
            {
                id: 'multi-exec',
                label: 'Multi-Host Execution',
                description: 'Run command on multiple hosts',
                icon: '🖥️',
                category: 'tools',
                keywords: ['multi', 'parallel', 'batch', 'all'],
                action: () => props.onNavigate('multi-exec'),
            },

            // Appearance
            {
                id: 'fullscreen',
                label: 'Toggle Fullscreen',
                icon: '⛶',
                category: 'appearance',
                shortcut: 'F11',
                keywords: ['fullscreen', 'maximize'],
                action: () => {
                    if (document.fullscreenElement) {
                        document.exitFullscreen();
                    } else {
                        document.documentElement.requestFullscreen();
                    }
                },
            }
        ];

        // Add dynamic host entries
        state.hosts.forEach((host) => {
            items.push({
                id: `connect-${host.id}`,
                label: `Connect: ${host.label}`,
                description: `${host.username ? host.username + '@' : ''}${host.hostname}:${host.port || 22}`,
                icon: host.is_favorite ? '⭐' : '🖥️',
                category: 'hosts',
                keywords: [host.label, host.hostname, ...(host.tags || [])],
                action: () => props.onConnect(host.id),
            });
        });

        // Add snippet entries
        snippets().forEach((snippet) => {
            items.push({
                id: `snippet-${snippet.id}`,
                label: `Snippet: ${snippet.title}`,
                description: snippet.description || snippet.command,
                icon: '📝',
                category: 'snippets',
                keywords: [snippet.title, ...(snippet.tags || []), 'run', 'execute'],
                action: () => {
                    // Navigate to library or prompt execution
                    props.onNavigate('snippets');
                },
            });
        });

        return items;
    });

    // Fuzzy search with scoring
    const filtered = createMemo(() => {
        const q = query().toLowerCase().trim();

        // Mode prefixes
        if (q.startsWith('>')) {
            setMode('commands');
            const cmdQuery = q.slice(1).trim();

            if (!cmdQuery) return allActions().filter(a => a.category !== 'hosts').slice(0, 20);

            // Trigger AI Suggestion
            if (cmdQuery.length > 3) {
                setMode('ai');
                return [{
                    id: 'ai-generate',
                    label: `Generate command: "${cmdQuery}"`,
                    description: aiSuggesting() ? '🧠 AI is thinking...' : aiResult() || 'Press Enter to ask AI',
                    icon: '✨',
                    category: 'ai' as ActionCategory,
                    action: async () => {
                        if (aiSuggesting()) return;
                        setAiSuggesting(true);
                        try {
                            const res = await GenerateCommand(cmdQuery);
                            setAiResult(res.text);
                            // We don't close here, we want the user to see/copy the command
                        } catch (err: unknown) {
                            setAiResult("Error: " + (err instanceof Error ? (err as Error).message : String(err)));
                        } finally {
                            setAiSuggesting(false);
                        }
                    }
                }];
            }

            return scoreAndFilter(allActions().filter(a => a.category !== 'hosts'), cmdQuery);
        }

        if (q.startsWith('@')) {
            setMode('hosts');
            const hostQuery = q.slice(1).trim();
            const hostActions = allActions().filter(a => a.category === 'hosts');
            if (!hostQuery) return hostActions.slice(0, 20);
            return scoreAndFilter(hostActions, hostQuery);
        }

        if (q.startsWith('/')) {
            setMode('snippets');
            const snippetQuery = q.slice(1).trim();
            const snippetActions = allActions().filter(a => a.category === 'snippets');
            if (!snippetQuery) return snippetActions.slice(0, 20);
            return scoreAndFilter(snippetActions, snippetQuery);
        }

        if (!q) return allActions().slice(0, 15);

        setMode('commands');
        return scoreAndFilter(allActions(), q);
    });

    function scoreAndFilter(items: PaletteAction[], query: string): PaletteAction[] {
        const scored = items
            .map((item) => {
                let score = 0;
                const label = item.label.toLowerCase();
                const desc = (item.description || '').toLowerCase();
                const keywords = (item.keywords || []).map(k => k.toLowerCase());

                // Exact match in label
                if (label === query) score += 100;
                // Starts with
                else if (label.startsWith(query)) score += 80;
                // Contains
                else if (label.includes(query)) score += 60;
                // Description contains
                if (desc.includes(query)) score += 40;
                // Keyword match
                for (const kw of keywords) {
                    if (kw.includes(query) || query.includes(kw)) score += 30;
                }

                // Fuzzy character match
                if (score === 0) {
                    let qi = 0;
                    for (const char of label) {
                        if (qi < query.length && char === query[qi]) {
                            qi++;
                            score += 2;
                        }
                    }
                    if (qi < query.length) score = 0; // Not all chars matched
                }

                return { item, score };
            })
            .filter(({ score }) => score > 0)
            .sort((a, b) => b.score - a.score)
            .slice(0, 15)
            .map(({ item }) => item);

        return scored;
    }

    // Reset selection when results change
    createEffect(() => {
        filtered();
        setSelectedIndex(0);
    });

    // Scroll selected item into view
    createEffect(() => {
        const idx = selectedIndex();
        if (listRef) {
            const items = listRef.querySelectorAll('.palette-item');
            if (items[idx]) {
                items[idx].scrollIntoView({ block: 'nearest' });
            }
        }
    });

    const executeSelected = () => {
        const items = filtered();
        const item = items[selectedIndex()];
        if (item) {
            if (item.category === 'ai' && !aiResult()) {
                item.action(); // Just trigger, don't close
                return;
            }
            props.onClose();
            setQuery('');
            // Delay to allow palette to close
            requestAnimationFrame(() => item.action());
        }
    };


    const handlePaletteKeyDown = (e: KeyboardEvent) => {
        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                setSelectedIndex((i) => Math.min(i + 1, filtered().length - 1));
                break;
            case 'ArrowUp':
                e.preventDefault();
                setSelectedIndex((i) => Math.max(i - 1, 0));
                break;
            case 'Enter':
                e.preventDefault();
                executeSelected();
                break;
            case 'Escape':
                e.preventDefault();
                props.onClose();
                setQuery('');
                break;
            case 'Tab':
                e.preventDefault();
                // Cycle through modes
                if (mode() === 'commands') setQuery('@');
                else if (mode() === 'hosts') setQuery('/');
                else setQuery('>');
                break;
        }
    };

    onMount(() => {
        // document.addEventListener('keydown', handleGlobalKeyDown);
    });

    onCleanup(() => {
        // document.removeEventListener('keydown', handleGlobalKeyDown);
    });

    // Auto-focus input when palette opens
    createEffect(() => {
        if (props.open && inputRef) {
            setTimeout(() => inputRef?.focus(), 50);
        }
    });

    const categoryLabel = (cat: ActionCategory): string => {
        const labels: Record<ActionCategory, string> = {
            connection: 'Connections',
            terminal: 'Terminal',
            navigation: 'Navigation',
            tools: 'Tools',
            security: 'Security',
            appearance: 'Appearance',
            hosts: 'Hosts',
            snippets: 'Snippets',
            recent: 'Recent',
            ai: 'AI Assistance',
        };
        return labels[cat] || cat;
    };

    const modeHint = () => {
        switch (mode()) {
            case 'hosts': return 'Search hosts...';
            case 'snippets': return 'Search snippets...';
            default: return 'Type a command, host name, or action...';
        }
    };

    return (
        <Show when={props.open}>
            <div class="palette-overlay" onClick={() => { props.onClose(); setQuery(''); }}>
                <div class="palette omni-search" onClick={(e) => e.stopPropagation()}>
                    {/* Input */}
                    <div class="palette-input-wrapper">
                        <span class="palette-icon">
                            {mode() === 'hosts' ? '🖥️' : mode() === 'snippets' ? '📝' : '🔍'}
                        </span>
                        <input
                            ref={inputRef}
                            class="palette-input"
                            type="text"
                            placeholder={modeHint()}
                            value={query()}
                            onInput={(e) => {
                                setQuery(e.currentTarget.value);
                                setSelectedIndex(0);
                            }}
                            onKeyDown={handlePaletteKeyDown}
                        />
                        <Show when={query()}>
                            <button
                                class="palette-clear"
                                onClick={() => { setQuery(''); inputRef?.focus(); }}
                            >
                                ✕
                            </button>
                        </Show>
                        <kbd class="palette-kbd">ESC</kbd>
                    </div>

                    {/* Mode tabs */}
                    <div class="palette-modes">
                        <button
                            class={`palette-mode ${mode() === 'commands' ? 'active' : ''}`}
                            onClick={() => setQuery('>')}
                        >
                            Commands
                        </button>
                        <button
                            class={`palette-mode ${mode() === 'hosts' ? 'active' : ''}`}
                            onClick={() => setQuery('@')}
                        >
                            Hosts ({state.hosts.length})
                        </button>
                        <button
                            class={`palette-mode ${mode() === 'snippets' ? 'active' : ''}`}
                            onClick={() => setQuery('/')}
                        >
                            Snippets
                        </button>
                    </div>

                    {/* Results */}
                    <div class="palette-results" ref={listRef}>
                        <For each={filtered()}>
                            {(item, index) => {
                                const showCategory = () =>
                                    index() === 0 || item.category !== filtered()[index() - 1]?.category;

                                return (
                                    <>
                                        <Show when={showCategory()}>
                                            <div class="palette-category">{categoryLabel(item.category)}</div>
                                        </Show>
                                        <div
                                            class={`palette-item ${selectedIndex() === index() ? 'selected' : ''} ${item.dangerous ? 'dangerous' : ''}`}
                                            onClick={() => {
                                                setSelectedIndex(index());
                                                executeSelected();
                                            }}
                                            onMouseEnter={() => setSelectedIndex(index())}
                                        >
                                            <span class="palette-item-icon">{item.icon}</span>
                                            <div class="palette-item-text">
                                                <span class="palette-item-label">
                                                    {highlightMatch(item.label, query().replace(/^[>@\/]/, ''))}
                                                </span>
                                                <Show when={item.description}>
                                                    <span class="palette-item-desc">{item.description}</span>
                                                </Show>
                                            </div>
                                            <Show when={item.shortcut}>
                                                <kbd class="palette-item-kbd">{item.shortcut}</kbd>
                                            </Show>
                                            <Show when={item.dangerous}>
                                                <span class="palette-item-danger" title="Potentially dangerous">⚠️</span>
                                            </Show>
                                        </div>
                                    </>
                                );
                            }}
                        </For>

                        <Show when={filtered().length === 0}>
                            <div class="palette-empty">
                                <p>No results for "{query().replace(/^[>@\/]/, '')}"</p>
                                <p class="palette-empty-hint">
                                    Try: <code>&gt;</code> for commands, <code>@</code> for hosts, <code>/</code> for snippets
                                </p>
                            </div>
                        </Show>
                    </div>

                    {/* Footer */}
                    <div class="palette-footer">
                        <div class="palette-footer-left">
                            <span><kbd>↑↓</kbd> Navigate</span>
                            <span><kbd>↵</kbd> Select</span>
                            <span><kbd>Tab</kbd> Switch mode</span>
                        </div>
                        <div class="palette-footer-right">
                            <span>{filtered().length} results</span>
                        </div>
                    </div>
                </div>
            </div>
        </Show>
    );
};

// Highlight matching characters in label
function highlightMatch(text: string, query: string): JSX.Element | string {
    if (!query) return text;

    const lowerText = text.toLowerCase();
    const lowerQuery = query.toLowerCase();
    const idx = lowerText.indexOf(lowerQuery);

    if (idx === -1) return text;

    return (
        <>
            {text.slice(0, idx)}
            <mark class="palette-highlight">{text.slice(idx, idx + query.length)}</mark>
            {text.slice(idx + query.length)}
        </>
    );
}
