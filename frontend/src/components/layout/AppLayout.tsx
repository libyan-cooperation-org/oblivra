import { Component, Show, createSignal, onMount, onCleanup } from 'solid-js';
import { useApp } from '@core/store';
import { IS_DESKTOP, IS_HYBRID } from '@core/context';
import { TitleBar } from './TitleBar';
import { StatusBar } from './StatusBar';
import { QuickSwitcher } from '../ui/QuickSwitcher';
import { SecurityKeyModal } from '../auth/SecurityKeyModal';
import { CommandPalette } from '../ui/CommandPalette';
import { AddHostModal } from '../sidebar/AddHostModal';
import { CommandRail } from './CommandRail';
import { DrawerPanel } from './DrawerPanel';
import { ModalSystem } from '../ui/ModalSystem';
import { useHotkeys } from '../../hooks/useHotkeys';
import { TransferPanel } from '../terminal/TransferPanel';
import { TransferDrawer } from '../terminal/TransferDrawer';
import { IS_BROWSER } from '@core/context';
import '../../styles/team.css';

import '../../styles/dashboard.css';
import '../../styles/panel.css';
import { IncidentSuggestion } from '../security/IncidentSuggestion';
import '../../styles/suggestion.css';
import { AlertToastContainer } from '../notifications/AlertSystem';

import { JSX } from 'solid-js/jsx-runtime';

import { usePanelShortcuts } from '../../hooks/usePanelShortcuts';

// Tabs that show a drawer panel — filtered by context
const DRAWER_TABS_BOTH    = ['hosts', 'team', 'plugins', 'health', 'metrics', 'updater', 'workspace', 'security', 'vault'];
const DRAWER_TABS_DESKTOP = ['snippets', 'tunnels', 'terminal', 'recordings', 'notes', 'sync'];
const DRAWER_TABS_BROWSER = ['soc', 'agents'];

function isDrawerVisible(tab: string): boolean {
    if (DRAWER_TABS_BOTH.includes(tab)) return true;
    if (DRAWER_TABS_DESKTOP.includes(tab)) return IS_DESKTOP || IS_HYBRID;
    if (DRAWER_TABS_BROWSER.includes(tab)) return !IS_DESKTOP || IS_HYBRID;
    return false;
}

export const AppLayout: Component<{ children?: JSX.Element }> = (props) => {
    const [state, actions] = useApp();
    const [showSecurityKeys, setShowSecurityKeys] = createSignal(false);
    const [showAddHost, setShowAddHost] = createSignal(false);
    const [showCommandPalette, setShowCommandPalette] = createSignal(false);
    const [showTransferDrawer, setShowTransferDrawer] = createSignal(false);

    onMount(async () => {
        if (IS_BROWSER) return; // WorkspaceService is desktop-only
        try {
            const { GetActive } = await import('../../../wailsjs/go/services/WorkspaceService');
            const ws = await GetActive();
            if (ws && ws.layout) {
                if (ws.active_tab) actions.setActiveNavTab(ws.active_tab as any);
            }
        } catch (err) {
            console.error('Failed to load workspace:', err);
        }
    });

    // Panel keyboard shortcuts (Ctrl+Shift+T = terminal, Ctrl+Shift+D = dashboard, …)
    usePanelShortcuts();

    // Global Hotkeys
    useHotkeys({
        'Cmd+K': (e) => { e.preventDefault(); setShowCommandPalette(p => !p); },
        'Cmd+P': (e) => { e.preventDefault(); setShowCommandPalette(true); },
        'Cmd+N': (e) => { e.preventDefault(); setShowAddHost(true); },
        'Cmd+B': (e) => { e.preventDefault(); actions.toggleSidebar(); },
        'Cmd+,': (e) => { e.preventDefault(); actions.setActiveNavTab('settings'); }
    });

    onMount(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key.toLowerCase() === 'f') {
                e.preventDefault();
                actions.toggleFocusMode();
            }
        };
        window.addEventListener('keydown', handleKeyDown);
        onCleanup(() => window.removeEventListener('keydown', handleKeyDown));
    });

    return (
        <div class={`app-layout ${state.focusMode ? 'focus-mode' : ''}`}>
            <QuickSwitcher />

            <Show when={!state.focusMode}>
                <TitleBar />
            </Show>

            <div class="app-body">
                <Show when={!state.focusMode}>
                    <CommandRail />
                </Show>

                <Show when={!state.focusMode && state.sidebarOpen && isDrawerVisible(state.activeNavTab)}>
                    <DrawerPanel onAddHost={() => setShowAddHost(true)} />
                </Show>

                <main class="main-content">
                    {props.children}
                </main>
            </div>

            <Show when={!state.focusMode}>
                <StatusBar onToggleTransfers={() => setShowTransferDrawer(p => !p)} />
            </Show>

            <CommandPalette
                open={showCommandPalette()}
                onClose={() => setShowCommandPalette(false)}
                onConnect={actions.connectToHost}
                onNavigate={(page) => actions.setActiveNavTab(page === 'sessions' ? 'terminal' : page as any)}
            />

            <Show when={showAddHost()}>
                <AddHostModal onClose={() => setShowAddHost(false)} />
            </Show>

            <Show when={showSecurityKeys()}>
                <SecurityKeyModal onClose={() => setShowSecurityKeys(false)} />
            </Show>

            <TransferPanel />
            <TransferDrawer open={showTransferDrawer()} onClose={() => setShowTransferDrawer(false)} />
            <ModalSystem />
            <IncidentSuggestion />
            <AlertToastContainer />
        </div>
    );
};
