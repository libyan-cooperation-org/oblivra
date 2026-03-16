/**
 * ContextBadge — shows the current deployment context in the status bar.
 * Clicking it opens the hybrid mode configuration panel.
 */
import { Component, createSignal, Show } from 'solid-js';
import { APP_CONTEXT, IS_DESKTOP, IS_HYBRID, configureHybridMode, disconnectHybridMode, getRemoteServerUrl } from '../core/context';

export const ContextBadge: Component = () => {
    const [showPanel, setShowPanel] = createSignal(false);
    const [serverInput, setServerInput] = createSignal(getRemoteServerUrl() ?? '');
    const [saving, setSaving] = createSignal(false);

    const label = APP_CONTEXT === 'desktop' ? 'DESKTOP'
                : APP_CONTEXT === 'hybrid'  ? 'HYBRID'
                : 'BROWSER';

    const color = APP_CONTEXT === 'desktop' ? '#5cc05c'
                : APP_CONTEXT === 'hybrid'  ? '#f58b00'
                : '#0099e0';

    const handleSave = () => {
        if (!serverInput().trim()) return;
        setSaving(true);
        configureHybridMode(serverInput()); // reloads page
    };

    const handleDisconnect = () => {
        disconnectHybridMode(); // reloads page
    };

    return (
        <>
            <span
                title={`Deployment mode: ${label} — click to configure`}
                onClick={() => IS_DESKTOP ? setShowPanel(p => !p) : undefined}
                style={{
                    display: 'inline-flex',
                    'align-items': 'center',
                    gap: '4px',
                    'font-family': 'var(--font-mono)',
                    'font-size': '9px',
                    'font-weight': '700',
                    'text-transform': 'uppercase',
                    'letter-spacing': '0.8px',
                    color: color,
                    cursor: IS_DESKTOP ? 'pointer' : 'default',
                    opacity: '0.8',
                    transition: 'opacity 0.15s',
                }}
                onMouseEnter={e => IS_DESKTOP && ((e.currentTarget as HTMLElement).style.opacity = '1')}
                onMouseLeave={e => IS_DESKTOP && ((e.currentTarget as HTMLElement).style.opacity = '0.8')}
            >
                <span style={{
                    width: '5px', height: '5px',
                    background: color,
                    'border-radius': '50%',
                    display: 'inline-block',
                    'flex-shrink': '0',
                }} />
                {label}
            </span>

            {/* Hybrid mode config panel */}
            <Show when={showPanel()}>
                <div
                    onClick={() => setShowPanel(false)}
                    style={{
                        position: 'fixed',
                        inset: '0',
                        'z-index': '8500',
                    }}
                />
                <div style={{
                    position: 'fixed',
                    bottom: '32px',
                    left: '210px',
                    'z-index': '8600',
                    background: 'var(--surface-2)',
                    border: '1px solid var(--border-secondary)',
                    'border-radius': 'var(--radius-md)',
                    'box-shadow': 'var(--shadow-xl)',
                    padding: '16px',
                    width: '340px',
                    display: 'flex',
                    'flex-direction': 'column',
                    gap: '12px',
                }}>
                    <div style={{
                        'font-family': 'var(--font-ui)',
                        'font-size': '12px',
                        'font-weight': '700',
                        color: 'var(--text-heading)',
                        'border-bottom': '1px solid var(--border-primary)',
                        'padding-bottom': '8px',
                    }}>
                        Server Connection
                    </div>

                    <div style={{ 'font-size': '11px', color: 'var(--text-muted)', 'line-height': '1.5' }}>
                        Connect this desktop binary to a remote OBLIVRA server to enable{' '}
                        <strong style={{ color: '#f58b00' }}>Hybrid mode</strong>:{' '}
                        local terminal + enterprise fleet management.
                    </div>

                    <Show when={IS_HYBRID}>
                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            color: '#5cc05c',
                            background: 'rgba(92,192,92,0.08)',
                            border: '1px solid rgba(92,192,92,0.2)',
                            padding: '6px 10px',
                            'border-radius': 'var(--radius-sm)',
                        }}>
                            ● Connected: {getRemoteServerUrl()}
                        </div>
                    </Show>

                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '6px' }}>
                        <label style={{
                            'font-size': '10px',
                            'font-weight': '600',
                            color: 'var(--text-secondary)',
                            'text-transform': 'uppercase',
                            'letter-spacing': '0.5px',
                        }}>
                            Server URL
                        </label>
                        <input
                            type="text"
                            placeholder="https://siem.yourorg.com:8080"
                            value={serverInput()}
                            onInput={e => setServerInput(e.currentTarget.value)}
                            style={{
                                background: 'var(--surface-0)',
                                border: '1px solid var(--border-primary)',
                                'border-radius': 'var(--radius-sm)',
                                color: 'var(--text-primary)',
                                'font-family': 'var(--font-mono)',
                                'font-size': '12px',
                                padding: '7px 10px',
                                outline: 'none',
                                width: '100%',
                            }}
                            onFocus={e => (e.currentTarget.style.borderColor = 'var(--accent-primary)')}
                            onBlur={e => (e.currentTarget.style.borderColor = 'var(--border-primary)')}
                        />
                    </div>

                    <div style={{ display: 'flex', gap: '8px' }}>
                        <button
                            onClick={handleSave}
                            disabled={saving() || !serverInput().trim()}
                            style={{
                                flex: '1',
                                background: '#f58b00',
                                border: 'none',
                                'border-radius': 'var(--radius-sm)',
                                color: '#fff',
                                'font-family': 'var(--font-ui)',
                                'font-size': '12px',
                                'font-weight': '600',
                                padding: '8px',
                                cursor: saving() ? 'wait' : 'pointer',
                                opacity: !serverInput().trim() ? '0.4' : '1',
                            }}
                        >
                            {saving() ? 'Connecting…' : 'Connect'}
                        </button>

                        <Show when={IS_HYBRID}>
                            <button
                                onClick={handleDisconnect}
                                style={{
                                    background: 'var(--surface-3)',
                                    border: '1px solid var(--border-primary)',
                                    'border-radius': 'var(--radius-sm)',
                                    color: 'var(--text-secondary)',
                                    'font-family': 'var(--font-ui)',
                                    'font-size': '12px',
                                    padding: '8px 12px',
                                    cursor: 'pointer',
                                }}
                                onMouseEnter={e => (e.currentTarget.style.borderColor = 'var(--alert-critical)')}
                                onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border-primary)')}
                            >
                                Disconnect
                            </button>
                        </Show>

                        <button
                            onClick={() => setShowPanel(false)}
                            style={{
                                background: 'transparent',
                                border: '1px solid var(--border-primary)',
                                'border-radius': 'var(--radius-sm)',
                                color: 'var(--text-muted)',
                                'font-family': 'var(--font-ui)',
                                'font-size': '12px',
                                padding: '8px 12px',
                                cursor: 'pointer',
                            }}
                        >
                            Cancel
                        </button>
                    </div>
                </div>
            </Show>
        </>
    );
};
