import { Component, Switch, Match, Show, createSignal } from 'solid-js';
import { useApp } from '@core/store';
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
import { Delete } from '../../../wailsjs/go/app/HostService';
import { database } from '../../../wailsjs/go/models';

interface DrawerPanelProps {
    onAddHost?: () => void;
}

export const DrawerPanel: Component<DrawerPanelProps> = (props) => {
    const [state, actions] = useApp();
    const [searchQuery, setSearchQuery] = createSignal('');
    const [editHost, setEditHost] = createSignal<database.Host | null>(null);

    // Only show search for host-related tabs
    const showSearch = () => ['hosts'].includes(state.activeNavTab);

    const handleEditHost = (hostId: string) => {
        const host = (state.hosts as unknown as database.Host[]).find(h => h.id === hostId);
        if (host) setEditHost(host);
    };

    const handleDeleteHost = async (hostId: string) => {
        if (!confirm('Delete this host? This cannot be undone.')) return;
        try {
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
                <Switch fallback={<EmptyState icon="🧭" title="Select a Tool" description="Choose a tool from the dock below to get started." compact />}>
                    <Match when={state.activeNavTab === 'hosts'}>
                        <HostTree searchQuery={searchQuery()} onAddHost={props.onAddHost} onEditHost={handleEditHost} onDeleteHost={handleDeleteHost} />
                        <div style="border-top: 1px solid var(--border-primary); margin-top: 8px; padding-top: 4px;">
                            <div class="section-label">Discovery & Cloud</div>
                            <SidebarHostList />
                        </div>
                    </Match>

                    <Match when={state.activeNavTab === 'snippets'}>
                        <SnippetVault />
                    </Match>

                    <Match when={state.activeNavTab === 'tunnels'}>
                        <div class="drawer-header"><span class="drawer-title">Port Forwarding</span></div>
                        {state.activeSessionId ? (
                            <TunnelManager sessionId={state.activeSessionId} />
                        ) : (
                            <EmptyState icon="🔗" title="Port Forwarding" description="Connect to a host to create SSH tunnels and forward ports." compact />
                        )}
                    </Match>

                    <Match when={state.activeNavTab === 'security'}>
                        <VaultManager />
                    </Match>

                    {/* ── New panels ── */}

                    <Match when={state.activeNavTab === 'terminal'}>
                        <SessionHistory />
                    </Match>

                    <Match when={state.activeNavTab === 'recordings'}>
                        <RecordingPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'notes'}>
                        <NotesPanel sessionId={state.activeSessionId || undefined} />
                    </Match>

                    <Match when={state.activeNavTab === 'team'}>
                        <TeamPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'sync'}>
                        <SyncPanel />
                    </Match>

                    <Match when={state.activeNavTab === 'plugins'}>
                        <PluginPanel />
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
