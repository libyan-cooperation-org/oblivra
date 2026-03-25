import { Component, Show, For, createSignal, onMount } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';
import { TerminalView } from './Terminal';
import { SSHBookmarks } from './SSHBookmarks';
import { OperatorBanner } from './OperatorBanner';

export const TerminalLayout: Component = () => {
    const [state, actions] = useApp();
    const [hovered, setHovered] = createSignal<string | null>(null);
    const [showBookmarks, setShowBookmarks] = createSignal(true);
    const [restorableSessions, setRestorableSessions] = createSignal<any[]>([]);

    // Fetch restorable sessions on mount, or auto-open local shell
    onMount(async () => {
        if (!IS_BROWSER) {
            try {
                const { GetRestorable } = await import('../../../wailsjs/go/services/SessionPersistence');
                const restorable = await GetRestorable();
                if (restorable && restorable.length > 0) {
                    setRestorableSessions(restorable);
                    return; // Wait for user to accept/decline
                }
            } catch (e) { console.error('Failed to fetch restorable sessions', e); }
        }

        if (state.sessions.filter(s => s.status === 'active').length === 0) {
            actions.connectToLocal();
        }
    });

    const handleRestore = async () => {
        const sessions = restorableSessions();
        setRestorableSessions([]); // clear prompt
        
        if (!IS_BROWSER) {
            try {
                const { ClearState } = await import('../../../wailsjs/go/services/SessionPersistence');
                await ClearState();
            } catch {}
        }
        
        for (const s of sessions) {
            if (s.is_local || s.host_id === 'local') {
                actions.connectToLocal();
            } else {
                actions.connectToHost(s.host_id);
            }
        }
    };

    const handleDeclineRestore = async () => {
        setRestorableSessions([]);
        if (!IS_BROWSER) {
            try {
                const { ClearState } = await import('../../../wailsjs/go/services/SessionPersistence');
                await ClearState();
            } catch {}
        }
        if (state.sessions.filter(s => s.status === 'active').length === 0) {
            actions.connectToLocal();
        }
    };

    const sessions = () => state.sessions.filter(s => s.status === 'active');
    const activeSession = () => sessions().find(s => s.id === state.activeSessionId);
    
    const isLocal = (s: any) =>
        s.hostId === 'local' || !s.hostId || s.hostLabel === 'Local Shell';

    const closeTab = async (session: any, e: MouseEvent) => {
        e.stopPropagation();
        if (!IS_BROWSER) {
            try {
                if (isLocal(session)) {
                    const { CloseSession } = await import('../../../wailsjs/go/services/LocalService');
                    await CloseSession(session.id);
                } else {
                    const { Disconnect } = await import('../../../wailsjs/go/services/SSHService');
                    await Disconnect(session.id);
                }
            } catch (_) {}
        }
        actions.removeSession(session.id);
    };

    const sendData = (session: any, data: string) => {
        if (IS_BROWSER) return;
        const encoded = btoa(unescape(encodeURIComponent(data)));
        if (isLocal(session)) {
            import('../../../wailsjs/go/services/LocalService')
                .then(m => m.SendInput(session.id, encoded)).catch(() => {});
        } else {
            import('../../../wailsjs/go/services/SSHService')
                .then(m => m.SendInput(session.id, encoded)).catch(() => {});
        }
    };

    const sendResize = (session: any, cols: number, rows: number) => {
        if (IS_BROWSER) return;
        if (isLocal(session)) {
            import('../../../wailsjs/go/services/LocalService')
                .then(m => m.Resize(session.id, cols, rows)).catch(() => {});
        } else {
            import('../../../wailsjs/go/services/SSHService')
                .then(m => m.Resize(session.id, cols, rows)).catch(() => {});
        }
    };

    return (
        <div style={{ display: 'flex', height: '100%', width: '100%', background: '#0d0e10', overflow: 'hidden' }}>
            
            {/* Sidebar Overlay/Pane */}
            <Show when={showBookmarks()}>
                <SSHBookmarks 
                    onConnect={(id) => actions.connectToHost(id)}
                    onClose={() => setShowBookmarks(false)} 
                />
            </Show>

            {/* Main Content Area */}
            <div style={{
                display: 'flex',
                'flex-direction': 'column',
                flex: '1',
                height: '100%',
                'min-width': '0',
            }}>
                {/* ── Tab Bar ─────────────────────────────────────────── */}
                <div
                    class="term-tabbar"
                style={{
                    display: 'flex',
                    'align-items': 'stretch',
                    height: '36px',
                    'flex-shrink': '0',
                    background: 'var(--surface-1)',
                    'border-bottom': '1px solid var(--border-primary)',
                    'overflow-x': 'auto',
                    'overflow-y': 'hidden',
                }}
            >
                <For each={sessions()}>
                    {(session) => {
                        const active = () => state.activeSessionId === session.id;
                        const local = isLocal(session);
                        const accent = local ? '#5cc05c' : '#f58b00';

                        return (
                            <div
                                onClick={() => actions.setActiveSession(session.id)}
                                onMouseEnter={() => setHovered(session.id)}
                                onMouseLeave={() => setHovered(null)}
                                style={{
                                    display: 'flex',
                                    'align-items': 'center',
                                    gap: '7px',
                                    padding: '0 10px 0 12px',
                                    cursor: 'pointer',
                                    'border-right': '1px solid var(--border-primary)',
                                    'border-top': `2px solid ${active() ? accent : 'transparent'}`,
                                    background: active() ? '#0d0e10' : 'transparent',
                                    'min-width': '130px',
                                    'max-width': '200px',
                                    'flex-shrink': '0',
                                    'user-select': 'none',
                                    transition: 'background 0.1s',
                                }}
                            >
                                {/* type pill */}
                                <span style={{
                                    'font-size': '9px',
                                    'font-weight': 700,
                                    'text-transform': 'uppercase',
                                    'letter-spacing': '0.4px',
                                    padding: '1px 5px',
                                    'border-radius': '2px',
                                    background: local
                                        ? 'rgba(92,192,92,0.15)'
                                        : 'rgba(245,139,0,0.15)',
                                    color: accent,
                                    border: `1px solid ${local ? 'rgba(92,192,92,0.3)' : 'rgba(245,139,0,0.3)'}`,
                                    'flex-shrink': '0',
                                }}>
                                    {local ? 'LOCAL' : 'SSH'}
                                </span>

                                {/* label */}
                                <span style={{
                                    flex: '1',
                                    overflow: 'hidden',
                                    'text-overflow': 'ellipsis',
                                    'white-space': 'nowrap',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '11px',
                                    'font-weight': active() ? '600' : '400',
                                    color: active() ? accent : 'var(--text-muted)',
                                }}>
                                    {local ? 'shell' : (session.hostLabel || session.hostId)}
                                </span>

                                {/* close */}
                                <button
                                    onClick={(e) => closeTab(session, e)}
                                    style={{
                                        background: 'none',
                                        border: 'none',
                                        padding: '0 2px',
                                        cursor: 'pointer',
                                        color: hovered() === session.id
                                            ? 'var(--text-secondary)'
                                            : 'transparent',
                                        'font-size': '14px',
                                        'line-height': '1',
                                        'flex-shrink': '0',
                                        transition: 'color 0.1s',
                                    }}
                                    onMouseEnter={e => (e.currentTarget.style.color = '#e04040')}
                                    onMouseLeave={e => (e.currentTarget.style.color =
                                        hovered() === session.id
                                            ? 'var(--text-secondary)'
                                            : 'transparent'
                                    )}
                                >×</button>
                            </div>
                        );
                    }}
                </For>

                {/* + new local shell */}
                <button
                    onClick={() => actions.connectToLocal()}
                    title="New Local Shell  (Ctrl+T)"
                    style={{
                        background: 'none',
                        border: 'none',
                        'border-right': '1px solid var(--border-primary)',
                        color: 'var(--text-muted)',
                        cursor: 'pointer',
                        padding: '0 14px',
                        'font-size': '20px',
                        'font-weight': '300',
                        'flex-shrink': '0',
                        'line-height': '1',
                        transition: 'color 0.1s, background 0.1s',
                    }}
                    onMouseEnter={e => {
                        e.currentTarget.style.background = 'var(--surface-3)';
                        e.currentTarget.style.color = '#5cc05c';
                    }}
                    onMouseLeave={e => {
                        e.currentTarget.style.background = 'none';
                        e.currentTarget.style.color = 'var(--text-muted)';
                    }}
                >+</button>

                {/* spacer + count */}
                <div style={{ flex: '1' }} />
                <Show when={sessions().length > 0}>
                    <div style={{
                        display: 'flex',
                        'align-items': 'center',
                        padding: '0 14px',
                        gap: '6px',
                        'font-family': 'var(--font-mono)',
                        'font-size': '10px',
                        color: 'var(--text-muted)',
                        'flex-shrink': '0',
                    }}>
                        <span style={{ color: '#5cc05c' }}>
                            {sessions().filter(s => isLocal(s)).length} local
                        </span>
                        <span>·</span>
                        <span style={{ color: '#f58b00' }}>
                            {sessions().filter(s => !isLocal(s)).length} ssh
                        </span>
                    </div>
                </Show>

                {/* Bookmarks Toggle button */}
                <button
                    onClick={() => setShowBookmarks(!showBookmarks())}
                    style={{
                        background: 'transparent',
                        border: 'none',
                        'border-left': '1px solid var(--border-primary)',
                        color: showBookmarks() ? '#0099e0' : 'var(--text-muted)',
                        cursor: 'pointer',
                        padding: '0 12px',
                        'font-size': '16px',
                        transition: 'color 0.1s',
                    }}
                    title="Toggle Bookmarks"
                >
                    📑
                </button>
            </div>

            {/* ── Operator Security Context Banner ──────────────────── */}
            <Show when={activeSession()}>
                <OperatorBanner 
                    hostId={activeSession()?.hostId || ''} 
                    hostLabel={activeSession()?.hostLabel || activeSession()?.hostId || 'local'} 
                />
            </Show>

            {/* ── Terminal pane ───────────────────────────────────── */}
            <div style={{ flex: '1', position: 'relative', overflow: 'hidden' }}>

                {/* Empty & Restore state — shown when no sessions exist */}
                <Show when={sessions().length === 0}>
                    <div style={{
                        position: 'absolute',
                        inset: '0',
                        display: 'flex',
                        'flex-direction': 'column',
                        'align-items': 'center',
                        'justify-content': 'center',
                        background: '#0d0e10',
                        gap: '20px',
                        color: 'var(--text-muted)',
                    }}>
                        {/* Restore Prompt */}
                        <Show when={restorableSessions().length > 0}>
                            <div style={{
                                background: 'rgba(0,153,224,0.05)',
                                border: '1px solid rgba(0,153,224,0.3)',
                                padding: '20px 30px',
                                'border-radius': '6px',
                                display: 'flex',
                                'flex-direction': 'column',
                                'align-items': 'center',
                                gap: '16px',
                                'margin-bottom': '20px'
                            }}>
                                <span style={{ color: '#0099e0', 'font-size': '24px' }}>📡</span>
                                <div style={{ 'text-align': 'center' }}>
                                    <div style={{ color: 'var(--text-primary)', 'font-weight': '600', 'margin-bottom': '4px' }}>
                                        Restore previous sessions?
                                    </div>
                                    <div style={{ 'font-size': '12px', opacity: 0.7 }}>
                                        You had {restorableSessions().length} active connections before shutting down.
                                    </div>
                                </div>
                                <div style={{ display: 'flex', gap: '12px' }}>
                                    <button onClick={handleRestore} style={{
                                        background: '#0099e0', color: '#fff', border: 'none', padding: '6px 16px',
                                        'border-radius': '4px', cursor: 'pointer', 'font-weight': '600'
                                    }}>Restore All</button>
                                    <button onClick={handleDeclineRestore} style={{
                                        background: 'transparent', color: 'var(--text-muted)', border: '1px solid var(--border-primary)', 
                                        padding: '6px 16px', 'border-radius': '4px', cursor: 'pointer'
                                    }}>Dismiss</button>
                                </div>
                            </div>
                        </Show>

                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '11px',
                            'text-transform': 'uppercase',
                            'letter-spacing': '2px',
                            opacity: '0.4',
                        }}>No active sessions</div>
                        <div style={{ display: 'flex', gap: '12px' }}>
                            <button
                                onClick={() => actions.connectToLocal()}
                                style={{
                                    background: 'rgba(92,192,92,0.1)',
                                    border: '1px solid rgba(92,192,92,0.35)',
                                    color: '#5cc05c',
                                    padding: '9px 22px',
                                    'border-radius': '3px',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '11px',
                                    'font-weight': 700,
                                    'letter-spacing': '1px',
                                    'text-transform': 'uppercase',
                                    cursor: 'pointer',
                                }}
                                onMouseEnter={e => e.currentTarget.style.background = 'rgba(92,192,92,0.2)'}
                                onMouseLeave={e => e.currentTarget.style.background = 'rgba(92,192,92,0.1)'}
                            >+ New Local Shell</button>
                            <span style={{
                                display: 'flex',
                                'align-items': 'center',
                                'font-family': 'var(--font-ui)',
                                'font-size': '11px',
                                color: 'var(--text-muted)',
                                opacity: '0.6',
                            }}>
                                or select a host from the sidebar →
                            </span>
                            <button
                                onClick={() => setShowBookmarks(true)}
                                style={{
                                    background: 'transparent',
                                    border: '1px solid rgba(0,153,224,0.3)',
                                    color: '#0099e0',
                                    padding: '9px 22px',
                                    'border-radius': '3px',
                                    'font-size': '11px',
                                    cursor: 'pointer',
                                }}
                            >
                                Open Bookmarks
                            </button>
                        </div>
                    </div>
                </Show>

                {/* All sessions rendered at once — only the active one is visible.
                    This avoids xterm re-init on tab switch. */}
                <For each={sessions()}>
                    {(session) => {
                        const isActive = () => state.activeSessionId === session.id;
                        return (
                            <div
                                style={{
                                    position: 'absolute',
                                    inset: '0',
                                    // Use visibility + pointer-events instead of display:none
                                    // so xterm can still measure its container dimensions
                                    visibility: isActive() ? 'visible' : 'hidden',
                                    'pointer-events': isActive() ? 'auto' : 'none',
                                    'z-index': isActive() ? '1' : '0',
                                }}
                            >
                                <TerminalView
                                    sessionId={session.id}
                                    active={isActive()}
                                    onData={(data) => sendData(session, data)}
                                    onResize={(cols, rows) => sendResize(session, cols, rows)}
                                />
                            </div>
                        );
                    }}
                </For>
            </div>
            
            </div> {/* End Main Content Area */}
        </div>
    );
};
