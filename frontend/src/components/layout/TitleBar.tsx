import { Component, createSignal, onMount } from 'solid-js';
import { AppLogo } from '../ui/AppLogo';
import { WindowMinimise, WindowToggleMaximise, Quit, WindowIsMaximised } from '../../../wailsjs/runtime/runtime';

export const TitleBar: Component = () => {
    const [quickVal, setQuickVal] = createSignal('');
    const [isMaximized, setIsMaximized] = createSignal(false);

    const handleQuickConnect = () => {
        const val = quickVal().trim();
        if (val) {
            console.log('Quick connect to:', val);
            setQuickVal('');
        }
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

    return (
        <header class="title-bar">
            <div class="title-bar-left">
                <div class="window-controls">
                    <div class="control-dot close" onClick={() => Quit()} title="Close" />
                    <div class="control-dot minimize" onClick={() => WindowMinimise()} title="Minimize" />
                    <div class="control-dot maximize" onClick={() => WindowToggleMaximise()} title={isMaximized() ? "Restore" : "Maximize"} />
                </div>

                <span class="app-logo" style="margin-left: 8px;">
                    <AppLogo size={16} />
                    SOVEREIGN
                </span>
            </div>

            <div class="title-bar-center">
                {/* Reserved for global breadcrumbs or active context */}
            </div>

            <div class="title-bar-right">
                <div class="quick-connect-pill" style="border-radius: 0; border: 1px solid var(--border-primary); padding: 0 8px;">
                    <span style="color: var(--text-muted); font-size: 10px; font-weight: 700; font-family: var(--font-mono); margin-right: 4px;">CONN:</span>
                    <input
                        type="text"
                        placeholder="target..."
                        value={quickVal()}
                        onInput={(e) => setQuickVal(e.currentTarget.value)}
                        onKeyDown={(e) => { if (e.key === 'Enter') handleQuickConnect(); }}
                        style="border-radius: 0; outline: none;"
                    />
                    <button class="connect-btn" onClick={handleQuickConnect} style="border-radius: 0;">&gt;</button>
                </div>
                <div class="user-avatar" title="Profile">K</div>
            </div>
        </header>
    );
};
