import { Component, createSignal, onMount } from 'solid-js';
import { AppLogo } from '../ui/AppLogo';
import { WindowMinimise, WindowToggleMaximise, Quit, WindowIsMaximised } from '../../../wailsjs/runtime/runtime';
import { useApp } from '@core/store';

export const TitleBar: Component = () => {
    const [, actions] = useApp();
    const [quickVal, setQuickVal] = createSignal('');
    const [isMaximized, setIsMaximized] = createSignal(false);

    const handleQuickConnect = () => {
        const val = quickVal().trim();
        if (!val) return;
        setQuickVal('');
        actions.connectToHost(val);
    };

    onMount(async () => {
        // Poll for maximized state or use an event if available
        const checkMax = async () => {
            const max = await WindowIsMaximised();
            setIsMaximized(max);
        };
        checkMax();
        const interval = setInterval(checkMax, 1000);
        return () => clearInterval(interval);
    });

    const [focused, setFocused] = createSignal(false);

    return (
        <header class="title-bar" style="height: var(--header-height); background: var(--surface-0); border-bottom: 1px solid var(--border-primary);">
            {/* Left: window controls only — logo is in CommandRail */}
            <div class="title-bar-left" style="min-width: 80px;">
                <div class="window-controls">
                    <div class="control-dot close" onClick={() => Quit()} title="Close" />
                    <div class="control-dot minimize" onClick={() => WindowMinimise()} title="Minimize" />
                    <div class="control-dot maximize" onClick={() => WindowToggleMaximise()} title={isMaximized() ? 'Restore' : 'Maximize'} />
                </div>
                <span style="
                    font-family: var(--font-mono);
                    font-size: 10px;
                    font-weight: 700;
                    color: var(--text-muted);
                    letter-spacing: 2px;
                    text-transform: uppercase;
                    opacity: 0.5;
                    margin-left: 8px;
                ">SOVEREIGN</span>
            </div>

            {/* Center: Quick Connect */}
            <div class="title-bar-center">
                <div style={`
                    display: flex;
                    align-items: center;
                    background: ${focused() ? 'var(--surface-2)' : 'transparent'};
                    border: 1px solid ${focused() ? 'var(--accent-primary)' : 'var(--border-primary)'};
                    padding: 0 10px;
                    height: 26px;
                    gap: 6px;
                    width: 260px;
                    transition: all var(--transition-fast);
                `}>
                    <span style="color: var(--accent-primary); font-size: 9px; font-weight: 700; font-family: var(--font-mono); letter-spacing: 1px; white-space: nowrap; opacity: 0.7;">SSH</span>
                    <span style="width: 1px; height: 12px; background: var(--border-primary);" />
                    <input
                        type="text"
                        placeholder="user@host or ip:port"
                        value={quickVal()}
                        onFocus={() => setFocused(true)}
                        onBlur={() => setFocused(false)}
                        onInput={(e) => setQuickVal(e.currentTarget.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') handleQuickConnect(); }}
                        style="
                            flex: 1;
                            background: transparent;
                            border: none;
                            outline: none;
                            color: var(--text-primary);
                            font-family: var(--font-mono);
                            font-size: 11px;
                        "
                    />
                    <span style="color: var(--text-muted); font-size: 9px; font-family: var(--font-mono); opacity: 0.5;">&#9166;</span>
                </div>
            </div>

            {/* Right: Status + Avatar */}
            <div class="title-bar-right" style="min-width: 80px; gap: 8px;">
                <span style="
                    font-family: var(--font-mono);
                    font-size: 9px;
                    color: var(--accent-primary);
                    letter-spacing: 1px;
                    opacity: 0.6;
                    text-transform: uppercase;
                ">v0.1</span>
                <div style="
                    width: 26px;
                    height: 26px;
                    background: var(--accent-dim);
                    border: 1px solid var(--accent-primary);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    font-size: 10px;
                    font-weight: 800;
                    font-family: var(--font-mono);
                    color: var(--accent-primary);
                    cursor: pointer;
                    border-radius: 2px;
                " title="Profile">K</div>
            </div>
        </header>
    );
};
