import { Component, Switch, Match, Show, createSignal } from 'solid-js';
import { useApp } from '@core/store';
import { IS_DESKTOP, IS_HYBRID, isRouteAvailable } from '@core/context';
import { EmptyState } from '../ui/EmptyState';
import { HostTree } from '../sidebar/HostTree';
import { SidebarHostList } from '../discovery/SidebarHostList';
import { SnippetVault } from '../sidebar/SnippetVault';
import { TunnelManager } from '../tunnels/TunnelManager';
import { VaultManager } from '../vault/VaultManager';
import { SessionHistory } from '../sessions/SessionHistory';
import { RecordingPanel } from '../recordings/RecordingPanel';
import { NotesPanel } from '../notes/NotesPanel';
import { TeamPanel } from '../team/TeamPanel';
import { SyncPanel } from '../sync/SyncPanel';
import { PluginPanel } from '../plugins/PluginPanel';
import { HealthPanel } from '../monitoring/HealthPanel';
import { MetricsPanel } from '../monitoring/MetricsPanel';
import { UpdaterPanel } from '../updater/UpdaterPanel';
import { WorkspacePanel } from '../workspace/WorkspacePanel';
import { AddHostModal } from '../sidebar/AddHostModal';
import { database } from '../../../wailsjs/go/models';
import { IS_BROWSER } from '@core/context';

interface DrawerPanelProps {
    onAddHost?: () => void;
}

// ── Context-restricted drawer placeholder ─────────────────────────────────────
const ContextLockedDrawer: Component<{ feature: string; route: string }> = (props) => {
    const isDesktopOnly = !isRouteAvailable(props.route) && (IS_DESKTOP || IS_HYBRID) === false;
    return (
        <div style={{
            display: 'flex', 'flex-direction': 'column', 'align-items': 'center',
            'justify-content': 'center', padding: '32px 16px', gap: '12px',
            'text-align': 'center',
        }}>
            <div style={{ 'font-size': '24px', opacity: '0.3' }}>⊘</div>
            <div style={{
                'font-family': 'var(--font-ui)', 'font-size': '12px',
                'font-weight': '600', color: 'var(--text-secondary)',
            }}>{props.feature}</div>
            <div style={{
                'font-family': 'var(--font-ui)', 'font-size': '11px',
                color: 'var(--text-muted)', 'line-height': '1.5',
            }}>
                {isDesktopOnly
                    ? 'Requires the desktop binary'
                    : 'Requires server mode'}
            </div>
        </div>
    );
};

export const DrawerPanel: Component<DrawerPanelProps> = (props) => {
    const [state, actions] = useApp();
    const [searchQuery, setSearchQuery] = createSignal('');
    const [editHost, setEditHost] = createSignal<database.Host | null>(null);

    const showSearch = () => ['hosts'].includes(state.activeNavTab);

    const handleEditHost = (hostId: string) => {
        const host = (state.hosts as unknown as database.Host[]).find(h => h.id === hostId);
        if (host) setEditHost(host);
    };

    const handleDeleteHost = async (hostId: string) => {
        if (IS_BROWSER || !confirm('Delete this host? This cannot be undone.')) return;
        try {
            const { Delete } = await import('../../../wailsjs/go/services/HostService');
            await Delete(hostId);
            actions.removeHost(hostId);
        } catch (err) {
            console.error('Failed to delete host:', err);
        }
    };

    return (
        <aside class="drawer-panel">
            {showSearch() && (
                <div class="drawer-search">
                    <div class="drawer-search-wrapper">
                        <span class="drawer-search-icon">🔍</span>
                        <input
                            type="text"
                            placeholder="Search hosts, commands..."
                            value={searchQuery()}
                            onInput={(e) => setSearchQuery(e.currentTarget.value)}
                        />
                    </div>
                </div>
            )}

            <div class="drawer-content">
                <Switch fallback={<EmptyState icon="🧭" title="Select a Tool"
                    description="Choose a tool from the dock to get started." compact />}>

                    {/* ── Both contexts ────────────────────────────────────── */}

                    <Match when={state.activeNavTab === 'hosts'}>
                        <HostTree searchQuery={searchQuery()} onAddHost={props.onAddHost}
                            onEditHost={handleEditHost} onDeleteHost={handleDeleteHost} />
                        <div style="border-top: 1px solid var(--border-primary); margin-top: 8px; padding-top: 4px;">
                            <div class="section-label">Discovery & Cloud</div>
                            <SidebarHostList />
                        </div>
                    </Match>

                    <Match when={state.activeNavTab === 'security'}>
                        <VaultManager />
                    </Match>

                    <Match when={state.activeNavTab === 'health'}>
                        <HealthPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'metrics'}>
                        <MetricsPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'updater'}>
                        <UpdaterPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'workspace'}>
                        <WorkspacePanel />
                    </Match>

                    <Match when={state.activeNavTab === 'team'}>
                        <TeamPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'plugins'}>
                        <PluginPanel />
                    </Match>

                    {/* ── Desktop-only drawers ─────────────────────────────── */}

                    <Match when={state.activeNavTab === 'terminal'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Session History" route="/terminal" />}
                        >
                            <SessionHistory />
                        </Show>
                    </Match>

                    <Match when={state.activeNavTab === 'snippets'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Command Snippets" route="/snippets" />}
                        >
                            <SnippetVault />
                        </Show>
                    </Match>

                    <Match when={state.activeNavTab === 'tunnels'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Port Forwarding" route="/tunnels" />}
                        >
                            <div class="drawer-header"><span class="drawer-title">Port Forwarding</span></div>
                            <Show
                                when={state.activeSessionId}
                                fallback={<EmptyState icon="🔗" title="Port Forwarding"
                                    description="Connect to a host to create SSH tunnels and forward ports." compact />}
                            >
                                <TunnelManager sessionId={state.activeSessionId!} />
                            </Show>
                        </Show>
                    </Match>

                    <Match when={state.activeNavTab === 'recordings'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Session Recordings" route="/recordings" />}
                        >
                            <RecordingPanel />
                        </Show>
                    </Match>

                    <Match when={state.activeNavTab === 'notes'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Notes" route="/notes" />}
                        >
                            <NotesPanel sessionId={state.activeSessionId || undefined} />
                        </Show>
                    </Match>

                    <Match when={state.activeNavTab === 'sync'}>
                        <Show
                            when={IS_DESKTOP || IS_HYBRID}
                            fallback={<ContextLockedDrawer feature="Node Sync" route="/sync" />}
                        >
                            <SyncPanel />
                        </Show>
                    </Match>

                    {/* ── Browser/server-only drawers ──────────────────────── */}

                    <Match when={state.activeNavTab === 'soc' || state.activeNavTab === 'agents'}>
                        <Show
                            when={!IS_DESKTOP || IS_HYBRID}
                            fallback={
                                <EmptyState icon="🌐" title="Server Mode Required"
                                    description="Connect to a remote OBLIVRA server in Settings → Server Connection to access fleet features."
                                    compact />
                            }
                        >
                            {/* SOC/Agents content rendered by the main page, not drawer */}
                            <EmptyState icon="📡" title="Use the main panel"
                                description="SOC workspace and agent console are full-screen views." compact />
                        </Show>
                    </Match>

                </Switch>
            </div>

            <Show when={editHost()}>
                <AddHostModal
                    host={editHost()!}
                    onClose={() => setEditHost(null)}
                    onHostAdded={() => { setEditHost(null); actions.refreshHosts(); }}
                />
            </Show>
        </aside>
    );
};
