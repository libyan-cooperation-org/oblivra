import { Component, Show, For, createSignal, onMount } from 'solid-js';
import { useApp } from '@core/store';
import { TerminalView } from './Terminal';
import {
    Disconnect,
    SendInput as SendSSHInput,
    Resize as ResizeSSH,
} from '../../../wailsjs/go/services/SSHService';
import {
    SendInput as SendLocalInput,
    Resize as ResizeLocal,
    CloseSession,
} from '../../../wailsjs/go/services/LocalService';

export const TerminalLayout: Component = () => {
    const [state, actions] = useApp();
    const [hovered, setHovered] = createSignal<string | null>(null);

    // Auto-open a local shell on first mount if no sessions exist
    onMount(() => {
        if (state.sessions.filter(s => s.status === 'active').length === 0) {
            actions.connectToLocal();
        }
    });

    const sessions = () => state.sessions.filter(s => s.status === 'active');
    const isLocal = (s: any) =>
        s.hostId === 'local' || !s.hostId || s.hostLabel === 'Local Shell';

    const closeTab = async (session: any, e: MouseEvent) => {
        e.stopPropagation();
        try {
            if (isLocal(session)) await CloseSession(session.id);
            else await Disconnect(session.id);
        } catch (_) {}
        actions.removeSession(session.id);
    };

    const sendData = (session: any, data: string) => {
        // UTF-8 safe base64
        const encoded = btoa(unescape(encodeURIComponent(data)));
        if (isLocal(session)) {
            SendLocalInput(session.id, encoded).catch(() => {});
        } else {
            SendSSHInput(session.id, encoded).catch(() => {});
        }
    };

    const sendResize = (session: any, cols: number, rows: number) => {
        if (isLocal(session)) ResizeLocal(session.id, cols, rows).catch(() => {});
        else ResizeSSH(session.id, cols, rows).catch(() => {});
    };

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            width: '100%',
            background: '#0d0e10',
            overflow: 'hidden',
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
            </div>

            {/* ── Terminal pane ───────────────────────────────────── */}
            <div style={{ flex: '1', position: 'relative', overflow: 'hidden' }}>

                {/* Empty state — shown when no sessions exist */}
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
        </div>
    );
};
