import { Component, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { TerminalView } from './Terminal';
import { Disconnect, SendInput as SendSSHInput, Resize as ResizeSSH } from '../../../wailsjs/go/services/SSHService';
import { SendInput as SendLocalInput, Resize as ResizeLocal, CloseSession } from '../../../wailsjs/go/services/LocalService';

export const TerminalLayout: Component = () => {
    const [state, actions] = useApp();

    const activeSession = () => state.sessions.find(s => s.id === state.activeSessionId);

    return (
        <div class="terminal-pane" style={{ height: '100%', width: '100%', display: 'flex', 'flex-direction': 'column' }}>
            {/* Session tabs */}
            <div style={{
                display: 'flex',
                background: 'var(--surface-1)',
                'border-bottom': '1px solid var(--border-primary)',
                'min-height': '36px',
                'align-items': 'center',
                gap: '0',
                padding: '0',
                'overflow-x': 'auto',
            }}>
                <For each={state.sessions.filter(s => s.status === 'active')}>
                    {(session) => (
                        <div
                            style={{
                                display: 'flex',
                                'align-items': 'center',
                                gap: '6px',
                                padding: '6px 16px',
                                cursor: 'pointer',
                                'font-family': 'var(--font-mono)',
                                'font-size': '11px',
                                'font-weight': 700,
                                'text-transform': 'uppercase',
                                'border-radius': '0',
                                'border-right': '1px solid var(--border-primary)',
                                background: state.activeSessionId === session.id
                                    ? 'var(--surface-0)' : 'transparent',
                                color: state.activeSessionId === session.id
                                    ? 'var(--text-primary)' : 'var(--text-muted)',
                                'border-bottom': state.activeSessionId === session.id
                                    ? '2px solid var(--accent-primary)' : '2px solid transparent',
                            }}
                            onClick={() => actions.setActiveSession(session.id)}
                        >
                            <span>{session.hostId === 'local' ? '💻' : '🖥️'}</span>
                            <span>{session.hostLabel || session.hostId}</span>
                            <button
                                style={{
                                    background: 'none', border: 'none', color: 'var(--text-muted)',
                                    cursor: 'pointer', padding: '0 4px', 'font-size': '14px',
                                }}
                                onClick={async (e) => {
                                    e.stopPropagation();
                                    if (session.hostId === 'local') {
                                        await CloseSession(session.id);
                                    } else {
                                        await Disconnect(session.id);
                                    }
                                    actions.removeSession(session.id);
                                }}
                            >×</button>
                        </div>
                    )}
                </For>

                {/* New terminal + button */}
                <button
                    style={{
                        background: 'none',
                        border: 'none',
                        color: 'var(--text-muted)',
                        cursor: 'pointer',
                        padding: '6px 12px',
                        'font-size': '16px',
                        'font-family': 'var(--font-mono)',
                        'font-weight': 700,
                        transition: 'color 0.15s',
                    }}
                    onMouseOver={(e) => e.currentTarget.style.color = 'var(--text-primary)'}
                    onMouseOut={(e) => e.currentTarget.style.color = 'var(--text-muted)'}
                    onClick={() => actions.connectToLocal()}
                    title="New Local Terminal"
                >+</button>
            </div>

            {/* Terminal content */}
            <div style={{ flex: 1, position: 'relative' }}>
                <Show when={activeSession()} fallback={
                    <div style={{
                        display: 'flex', 'align-items': 'center', 'justify-content': 'center',
                        height: '100%', color: 'var(--text-muted)',
                        'flex-direction': 'column', gap: '16px',
                    }}>
                        <div style={{ 'text-align': 'center' }}>
                            <div style={{ 'font-size': '48px', 'margin-bottom': '16px' }}>🖥️</div>
                            <div style={{ 'font-size': '14px', 'margin-bottom': '24px' }}>No active sessions</div>
                            <button
                                style={{
                                    background: 'var(--accent-primary)',
                                    color: '#000',
                                    border: 'none',
                                    padding: '10px 24px',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '12px',
                                    'font-weight': 700,
                                    'text-transform': 'uppercase',
                                    'letter-spacing': '1px',
                                    cursor: 'pointer',
                                }}
                                onClick={() => actions.connectToLocal()}
                            >
                                ▶ NEW LOCAL TERMINAL
                            </button>
                        </div>
                    </div>
                }>
                    <TerminalView
                        sessionId={state.activeSessionId!}
                        onData={(data) => {
                            const session = activeSession();
                            if (!session) return;
                            const encoded = btoa(data);
                            if (session.hostId === 'local') {
                                SendLocalInput(session.id, encoded);
                            } else {
                                SendSSHInput(session.id, encoded);
                            }
                        }}
                        onResize={(cols, rows) => {
                            const session = activeSession();
                            if (!session) return;
                            if (session.hostId === 'local') {
                                ResizeLocal(session.id, cols, rows);
                            } else {
                                ResizeSSH(session.id, cols, rows);
                            }
                        }}
                    />
                </Show>
            </div>
        </div>
    );
};
