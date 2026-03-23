import { Component, Show, createSignal, onCleanup, onMount, For } from 'solid-js';
import { SplitPane } from './SplitPane';
import { TerminalView } from './Terminal';
import { FileBrowser } from './FileBrowser';
import { TerminalToolbar } from './TerminalToolbar';
import { SessionShareModal } from './SessionShareModal';
import { subscribe } from '@core/bridge';
import { IS_BROWSER } from '@core/context';
import '../../styles/splitpane.css';

// Tree structure for pane layout
export interface PaneNode {
    id: string;
    type: 'terminal' | 'split-h' | 'split-v';
    sessionId?: string;
    hostLabel?: string;
    children?: [PaneNode, PaneNode];
    splitRatio?: number;
}

interface TerminalGridProps {
    layout: PaneNode;
    activePane: string;
    onPaneSelect: (paneId: string) => void;
    onPaneSplit: (paneId: string, direction: 'horizontal' | 'vertical') => void;
    onPaneClose: (paneId: string) => void;
    onData: (sessionId: string, data: string) => void;
    onResize: (sessionId: string, cols: number, rows: number) => void;
    onDisconnect: (sessionId: string) => void;
    broadcastMode: boolean;
}

export const TerminalGrid: Component<TerminalGridProps> = (props) => {
    // Keep track of which panes have their SFTP browser open or share modal
    const [browserOpen, setBrowserOpen] = createSignal<Record<string, boolean>>({});
    const [shareOpen, setShareOpen] = createSignal<Record<string, boolean>>({});
    const [viewerCounts, setViewerCounts] = createSignal<Record<string, number>>({});
    const [systemMessages, setSystemMessages] = createSignal<Record<string, { id: string; content: string }[]>>({});

    onMount(() => {
        // Listen for system command outputs
        const unsubscribe = subscribe('session.system_output', (data: any) => {
            const { sessionId, content } = data;
            setSystemMessages(prev => {
                const msgs = prev[sessionId] || [];
                return {
                    ...prev,
                    [sessionId]: [...msgs, { id: Math.random().toString(36), content }]
                };
            });

            // Auto-clear after 10 seconds
            setTimeout(() => {
                setSystemMessages(prev => {
                    const msgs = prev[sessionId] || [];
                    return {
                        ...prev,
                        [sessionId]: msgs.filter(m => m.content !== content)
                    };
                });
            }, 10000);
        });

        onCleanup(() => unsubscribe());
    });

    // Poll for viewer counts
    const pollViewers = async () => {
        if (IS_BROWSER) return;
        const counts: Record<string, number> = {};
        const findSessions = (node: PaneNode) => {
            if (node.type === 'terminal' && node.sessionId) counts[node.sessionId] = 0;
            if (node.children) { findSessions(node.children[0]); findSessions(node.children[1]); }
        };
        findSessions(props.layout);
        try {
            const { GetTotalViewers } = await import('../../../wailsjs/go/services/ShareService');
            for (const sid of Object.keys(counts)) {
                counts[sid] = await GetTotalViewers(sid).catch(() => 0);
            }
        } catch { /* ignore */ }
        setViewerCounts(counts);
    };

    const pollInterval = setInterval(pollViewers, 5000);
    onCleanup(() => clearInterval(pollInterval));

    const toggleBrowser = (paneId: string) => {
        setBrowserOpen(prev => ({ ...prev, [paneId]: !prev[paneId] }));
    };

    const toggleShare = (paneId: string) => {
        setShareOpen(prev => ({ ...prev, [paneId]: !prev[paneId] }));
    };

    const toggleSearch = () => {
        // Search logic handled by TerminalToolbar directly now
    };

    const renderPane = (node: PaneNode): any => {
        if (node.type === 'terminal') {
            return (
                <div
                    class={`grid-pane ${props.activePane === node.id ? 'active' : ''}`}
                    onClick={() => props.onPaneSelect(node.id)}
                >
                    {/* Unified Terminal Toolbar */}
                    <Show when={node.sessionId}>
                        <TerminalToolbar
                            sessionId={node.sessionId!}
                            hostLabel={node.hostLabel || 'Terminal'}
                            onDisconnect={() => props.onDisconnect(node.sessionId!)}
                            onToggleSearch={() => toggleSearch()}
                            onClear={() => { /* TODO: expose terminal clear */ }}
                            onSplit={(dir) => props.onPaneSplit(node.id, dir)}
                            onToggleSftp={() => toggleBrowser(node.id)}
                            onShare={() => toggleShare(node.id)}
                        />
                    </Show>

                    {/* Terminal content */}
                    <Show
                        when={node.sessionId}
                        fallback={
                            <div class="grid-pane-empty">
                                <p>No active session</p>
                                <button onClick={() => { /* open connection picker */ }}>
                                    Connect
                                </button>
                            </div>
                        }
                    >
                        <div style="display: flex; flex-direction: row; height: 100%; width: 100%;">
                            <div style="flex: 1; position: relative;">
                                <TerminalView
                                    sessionId={node.sessionId!}
                                    onData={(data) => props.onData(node.sessionId!, data)}
                                    onResize={(cols, rows) => props.onResize(node.sessionId!, cols, rows)}
                                    {...{ broadcastStatus: props.broadcastMode ? (props.activePane === node.id ? 'leader' : 'follower') : null }}
                                />
                            </div>
                            <Show when={browserOpen()[node.id]}>
                                <div style="border-left: 1px solid var(--border-primary); max-width: 400px; min-width: 250px;">
                                    <FileBrowser sessionId={node.sessionId!} onClose={() => toggleBrowser(node.id)} />
                                </div>
                            </Show>

                            {/* Viewer Badge */}
                            <Show when={(viewerCounts()[node.sessionId!] || 0) > 0}>
                                <div class="collab-badge">
                                    <div class="collab-pulse"></div>
                                    <span>{(viewerCounts()[node.sessionId!] || 0)} Viewers</span>
                                </div>
                            </Show>

                            {/* System Output Overlay */}
                            <div class="system-overlay-container" style={{
                                position: 'absolute',
                                top: '40px',
                                right: '10px',
                                display: 'flex',
                                'flex-direction': 'column',
                                gap: '8px',
                                'z-index': 1000,
                                'max-width': '80%'
                            }}>
                                <For each={systemMessages()[node.sessionId!] || []}>
                                    {(msg) => (
                                        <div class="system-msg-box app-entry-animation" style={{
                                            background: 'rgba(17, 24, 39, 0.95)',
                                            border: '1px solid #3b82f6',
                                            'border-left': '4px solid #3b82f6',
                                            padding: '12px',
                                            'font-family': 'monospace',
                                            'font-size': '12px',
                                            color: '#e5e7eb',
                                            'white-space': 'pre-wrap',
                                            'box-shadow': '0 10px 15px -3px rgba(0, 0, 0, 0.5)',
                                            'border-radius': '2px'
                                        }}>
                                            {msg.content}
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>
                        <Show when={shareOpen()[node.id]}>
                            <SessionShareModal
                                sessionId={node.sessionId!}
                                hostLabel={node.hostLabel || 'Terminal'}
                                onClose={() => toggleShare(node.id)}
                            />
                        </Show>
                    </Show>
                </div>
            );
        }

        if ((node.type === 'split-h' || node.type === 'split-v') && node.children) {
            const direction = node.type === 'split-h' ? 'horizontal' : 'vertical';
            return (
                <SplitPane
                    direction={direction}
                    initialSplit={node.splitRatio || 0.5}
                >
                    {renderPane(node.children[0])}
                    {renderPane(node.children[1])}
                </SplitPane>
            );
        }

        return null;
    };

    return (
        <div class="terminal-grid">
            {renderPane(props.layout)}
        </div>
    );
};
