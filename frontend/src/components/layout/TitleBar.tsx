import { Component, createSignal, onMount } from 'solid-js';
import { WindowMinimise, WindowToggleMaximise, Quit, WindowIsMaximised } from '../../../wailsjs/runtime/runtime';
import { useApp } from '@core/store';
import { PanelLauncher } from './PanelLauncher';

export const TitleBar: Component = () => {
    const [, actions] = useApp();
    const [quickVal, setQuickVal] = createSignal('');
    const [isMaximized, setIsMaximized] = createSignal(false);
    const [focused, setFocused] = createSignal(false);

    const handleQuickConnect = () => {
        const val = quickVal().trim();
        if (!val) return;
        setQuickVal('');
        actions.connectToHost(val);
    };

    onMount(async () => {
        const checkMax = async () => {
            const max = await WindowIsMaximised();
            setIsMaximized(max);
        };
        checkMax();
        const interval = setInterval(checkMax, 1000);
        return () => clearInterval(interval);
    });

    return (
        <>
            <style>{`
                .title-bar-v2 {
                    display: flex;
                    align-items: center;
                    height: var(--header-height);
                    background: var(--surface-1);
                    border-bottom: 1px solid var(--border-primary);
                    padding: 0 12px 0 0;
                    -webkit-app-region: drag;
                    user-select: none;
                    z-index: 100;
                    gap: 0;
                }

                .tb-controls {
                    display: flex;
                    align-items: center;
                    gap: 6px;
                    padding: 0 14px 0 14px;
                    -webkit-app-region: no-drag;
                    flex-shrink: 0;
                }

                .tb-dot {
                    width: 12px;
                    height: 12px;
                    border-radius: 50%;
                    cursor: pointer;
                    border: none;
                    padding: 0;
                    transition: filter var(--transition-fast), transform var(--transition-fast);
                    flex-shrink: 0;
                }
                .tb-dot:hover { filter: brightness(1.2); transform: scale(1.1); }
                .tb-dot.close    { background: #ff5f57; }
                .tb-dot.minimize { background: #ffbd2e; }
                .tb-dot.maximize { background: #28c840; }

                .tb-brand {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    padding: 0 16px 0 4px;
                    border-right: 1px solid var(--border-primary);
                    height: 100%;
                    flex-shrink: 0;
                }

                .tb-brand-name {
                    font-family: var(--font-mono);
                    font-size: 11px;
                    font-weight: 700;
                    color: var(--text-secondary);
                    letter-spacing: 2px;
                    text-transform: uppercase;
                    opacity: 0.7;
                }

                .tb-center {
                    flex: 1;
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    padding: 0 12px;
                    -webkit-app-region: no-drag;
                }

                .tb-ssh-bar {
                    display: flex;
                    align-items: center;
                    background: var(--surface-2);
                    border: 1px solid var(--border-primary);
                    border-radius: var(--radius-md);
                    padding: 0 10px;
                    height: 30px;
                    gap: 8px;
                    width: 280px;
                    transition: all var(--transition-fast);
                    cursor: text;
                }
                .tb-ssh-bar:focus-within {
                    border-color: var(--accent-primary);
                    background: var(--surface-3);
                    box-shadow: 0 0 0 2px rgba(87,139,255,0.15);
                }

                .tb-ssh-icon {
                    color: var(--accent-primary);
                    font-size: 9px;
                    font-weight: 700;
                    font-family: var(--font-mono);
                    letter-spacing: 1px;
                    white-space: nowrap;
                    opacity: 0.8;
                    flex-shrink: 0;
                }

                .tb-sep {
                    width: 1px;
                    height: 14px;
                    background: var(--border-primary);
                    flex-shrink: 0;
                }

                .tb-ssh-input {
                    flex: 1;
                    background: transparent;
                    border: none;
                    outline: none;
                    color: var(--text-primary);
                    font-family: var(--font-mono);
                    font-size: 11px;
                }
                .tb-ssh-input::placeholder {
                    color: var(--text-muted);
                }

                .tb-ssh-enter {
                    color: var(--text-muted);
                    font-size: 10px;
                    font-family: var(--font-mono);
                    opacity: 0.5;
                    flex-shrink: 0;
                }

                .tb-right {
                    display: flex;
                    align-items: center;
                    gap: 10px;
                    flex-shrink: 0;
                    -webkit-app-region: no-drag;
                }

                .tb-version {
                    font-family: var(--font-mono);
                    font-size: 9px;
                    color: var(--text-muted);
                    letter-spacing: 0.5px;
                    opacity: 0.5;
                }

                .tb-avatar {
                    width: 28px;
                    height: 28px;
                    background: linear-gradient(135deg, var(--accent-primary) 0%, var(--accent-secondary) 100%);
                    border-radius: var(--radius-sm);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    font-size: 11px;
                    font-weight: 700;
                    font-family: var(--font-ui);
                    color: #fff;
                    cursor: pointer;
                    transition: filter var(--transition-fast), transform var(--transition-fast);
                    flex-shrink: 0;
                }
                .tb-avatar:hover {
                    filter: brightness(1.1);
                    transform: scale(1.05);
                }
            `}</style>

            <header class="title-bar-v2">
                {/* macOS traffic lights */}
                <div class="tb-controls">
                    <button class="tb-dot close"    onClick={() => Quit()}                   title="Close"    />
                    <button class="tb-dot minimize" onClick={() => WindowMinimise()}          title="Minimize" />
                    <button class="tb-dot maximize" onClick={() => WindowToggleMaximise()}    title={isMaximized() ? 'Restore' : 'Maximize'} />
                </div>

                {/* Brand */}
                <div class="tb-brand">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                        <path d="M12 2L4 6.5v11L12 22l8-4.5v-11L12 2z" stroke="var(--accent-primary)" stroke-width="1.5" fill="none"/>
                        <circle cx="12" cy="12" r="2.5" fill="var(--accent-primary)"/>
                    </svg>
                    <span class="tb-brand-name">Sovereign</span>
                </div>

                {/* Quick Connect */}
                <div class="tb-center">
                    <div class="tb-ssh-bar">
                        <span class="tb-ssh-icon">SSH</span>
                        <div class="tb-sep" />
                        <input
                            class="tb-ssh-input"
                            type="text"
                            placeholder="user@host or ip:port"
                            value={quickVal()}
                            onFocus={() => setFocused(true)}
                            onBlur={() => setFocused(false)}
                            onInput={(e) => setQuickVal(e.currentTarget.value)}
                            onKeyDown={(e) => { if (e.key === 'Enter') handleQuickConnect(); }}
                        />
                        <span class="tb-ssh-enter">↵</span>
                    </div>
                </div>

                {/* Panel Launcher */}
                <div style="-webkit-app-region: no-drag; padding-right: 10px;">
                    <PanelLauncher />
                </div>

                {/* Right controls */}
                <div class="tb-right">
                    <span class="tb-version">v0.1</span>
                    <div class="tb-avatar" title="Profile">K</div>
                </div>
            </header>
        </>
    );
};
