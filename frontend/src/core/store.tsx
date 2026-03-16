import {
    createContext,
    useContext,
    ParentComponent,
    onMount,
} from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { createStore } from 'solid-js/store';
import { Connect } from '../../wailsjs/go/services/SSHService';
import { StartLocalSession } from '../../wailsjs/go/services/LocalService';
import { ListHosts } from '../../wailsjs/go/services/HostService';
import { IsUnlocked } from '../../wailsjs/go/services/VaultService';
import { subscribe } from './bridge';
import {
    Host,
    Session,
    PluginPanel,
    PluginStatusIcon,
    Notification,
    Transfer
} from './types';

export interface AppState {
    hosts: Host[];
    sessions: Session[];
    activeSessionId: string | null;
    vaultUnlocked: boolean;
    theme: string;
    sidebarOpen: boolean;
    loading: boolean;
    activeNavTab: 'dashboard' | 'hosts' | 'snippets' | 'tunnels' | 'security' | 'terminal' | 'recordings' | 'notes' | 'compliance' | 'ops' | 'team' | 'sync' | 'plugins' | 'health' | 'metrics' | 'updater' | 'workspace' | 'alerts' | 'siem' | 'settings' | 'topology' | 'vault' | 'soc' | 'temporal' | 'lineage' | 'decisions' | 'ledger' | 'replay' | 'ai-assistant' | 'mitre-heatmap';
    workspace: 'Personal' | 'Work' | 'Team';
    pluginPanels: PluginPanel[];
    pluginStatusIcons: PluginStatusIcon[];
    focusMode: boolean;
    notifications: Notification[];
    transfers: Transfer[];
}

export interface AppActions {
    setHosts: (hosts: Host[]) => void;
    addHost: (host: Host) => void;
    removeHost: (id: string) => void;
    setActiveSession: (id: string | null) => void;
    addSession: (session: Session) => void;
    removeSession: (id: string) => void;
    setVaultUnlocked: (unlocked: boolean) => void;
    toggleSidebar: () => void;
    setTheme: (theme: string) => void;
    setLoading: (loading: boolean) => void;
    setActiveNavTab: (tab: AppState['activeNavTab']) => void;
    setWorkspace: (workspace: AppState['workspace']) => void;
    connectToHost: (hostId: string) => void;
    connectToLocal: () => void;
    refreshHosts: () => Promise<void>;
    toggleFocusMode: () => void;
    notify: (message: string, type?: Notification['type'], details?: string, duration?: number) => void;
    dismissNotification: (id: string) => void;
    updateTransfer: (transfer: Partial<Transfer> & { id: string }) => void;
}

type AppContextType = [AppState, AppActions];
const AppContext = createContext<AppContextType>();

export const AppProvider: ParentComponent = (props) => {
    const [state, setState] = createStore<AppState>({
        hosts: [],
        sessions: [],
        activeSessionId: null,
        vaultUnlocked: false,
        theme: 'dark',
        sidebarOpen: true,
        loading: false,
        activeNavTab: 'dashboard',
        workspace: 'Personal',
        pluginPanels: [],
        pluginStatusIcons: [],
        focusMode: false,
        notifications: [],
        transfers: [],
    });

    const navigate = useNavigate();

    const refreshHosts = async () => {
        try {
            const hosts = await ListHosts();
            setState('hosts', hosts || []);
        } catch (e) {
            console.error("Failed to refresh hosts", e);
        }
    };

    onMount(async () => {
        // Register ALL event listeners FIRST — before any async calls — so we
        // never miss events that fire during startup race conditions.
        subscribe('vault:unlocked', () => {
            setState('vaultUnlocked', true);
            refreshHosts();
        });

        subscribe('vault:locked', () => {
            setState('vaultUnlocked', false);
        });

        subscribe('host:list_updated', () => {
            refreshHosts();
        });

        // Initial state check — now safe because all listeners are already registered
        const unlocked = await IsUnlocked();
        setState('vaultUnlocked', unlocked);
        if (unlocked) {
            await refreshHosts();
        } else {
            // Poll every 300ms for up to 15s to catch auto-unlock (startup race)
            let polls = 0;
            const poll = setInterval(async () => {
                polls++;
                // SECURITY: Stop polling after limit or if vault was manually locked in between
                if (polls > 50 || !state.sidebarOpen) { 
                    clearInterval(poll); 
                    return; 
                }
                
                const isNowUnlocked = await IsUnlocked();
                if (isNowUnlocked) {
                    clearInterval(poll);
                    setState('vaultUnlocked', true);
                    await refreshHosts();
                }
            }, 300);

            // Ensure poll is cleared if we lock manually during the polling window
            subscribe('vault:locked', () => clearInterval(poll));
        }

        subscribe('session:started', (data: { id: string, hostId: string, label: string }) => {
            // Backend started a session (e.g. from terminal or click)
            const newSession: Session = {
                id: data.id,
                hostId: data.hostId,
                hostLabel: data.label,
                status: 'active',
                startedAt: new Date().toISOString(),
            };
            setState('sessions', (prev) => {
                if (prev.find(s => s.id === data.id)) return prev;
                return [...prev, newSession];
            });
            setState('activeSessionId', data.id);
        });

        subscribe('session:closed', (sessionId: string) => {
            setState('sessions', (prev) => prev.filter(s => s.id !== sessionId));
            if (state.activeSessionId === sessionId) {
                const remaining = state.sessions.filter(s => s.id !== sessionId);
                setState('activeSessionId', remaining.length > 0 ? remaining[0].id : null);
            }
        });

        subscribe('recording:stopped', (sessionId: string) => {
            setState('sessions', (s) => s.id === sessionId, 'isRecording', false);
        });

        subscribe('ui.register_panel', (data: PluginPanel) => {
            setState('pluginPanels', (prev) => {
                if (prev.find(p => p.plugin_id === data.plugin_id && p.panel_id === data.panel_id)) return prev;
                return [...prev, data];
            });
        });

        subscribe('ui.add_status_icon', (data: PluginStatusIcon) => {
            setState('pluginStatusIcons', (prev) => {
                if (prev.find(i => i.plugin_id === data.plugin_id && i.icon_id === data.icon_id)) return prev;
                return [...prev, data];
            });
        });

        // Listen for transfer updates from backend
        subscribe('transfer:updated', (data: Partial<Transfer> & { id: string }) => {
            // data matches Transfer interface
            setState('transfers', (t) => t.id === data.id, (prev) => {
                if (!prev) {
                    // New transfer
                    setState('transfers', (curr) => [...curr, data as Transfer]);
                    return prev;
                }
                return { ...prev, ...data };
            });
        });

        subscribe('transfer:completed', (data: { id: string, name: string }) => {
            setState('transfers', (t) => t.id === data.id, 'status', 'completed');
            actions.notify(`Transfer ${data.name} completed`, 'success');
        });

        subscribe('transfer:failed', (data: { id: string, name: string, error: string }) => {
            setState('transfers', (t) => t.id === data.id, 'status', 'failed');
            setState('transfers', (t) => t.id === data.id, 'error', data.error);
            actions.notify(`Transfer ${data.name} failed: ${data.error}`, 'error');
        });
    });

    const actions: AppActions = {
        setHosts: (hosts) => setState('hosts', hosts),
        addHost: (host) => setState('hosts', (prev) => [...prev, host]),
        removeHost: (id) => setState('hosts', (prev) => prev.filter((h) => h.id !== id)),
        setActiveSession: (id) => setState('activeSessionId', id),
        addSession: (session) => setState('sessions', (prev) => [...prev, session]),
        removeSession: (id) => setState('sessions', (prev) => prev.filter((s) => s.id !== id)),
        setVaultUnlocked: (unlocked) => setState('vaultUnlocked', unlocked),
        toggleSidebar: () => setState('sidebarOpen', (prev) => !prev),
        setTheme: (theme) => setState('theme', theme),
        setLoading: (loading) => setState('loading', loading),
        setActiveNavTab: (tab) => setState('activeNavTab', tab),
        setWorkspace: (ws) => setState('workspace', ws),
        refreshHosts,

        connectToHost: (hostId: string) => {
            // First try direct ID match
            let host = state.hosts.find(h => h.id === hostId);

            // Fallback: If not found by ID, try finding by hostname or label (useful for Discovery sidebar)
            if (!host) {
                host = state.hosts.find(h => h.hostname === hostId || h.label === hostId);
            }

            if (!host) {
                actions.notify(`Host not found in Fleet: ${hostId}. Please add it first.`, 'error');
                return;
            }

            const targetId = host.id;
            setState('activeNavTab', 'terminal');
            navigate('/terminal');

            const existing = state.sessions.find(s => s.hostId === targetId && s.status === 'active');
            if (existing) {
                setState('activeSessionId', existing.id);
                return;
            }

            Connect(targetId).catch((err: unknown) => {
                console.error('SSH connect failed:', err);
                const errorMsg = err instanceof Error ? err.message : String(err);
                actions.notify(`Failed to connect to ${host?.label || targetId}`, 'error', errorMsg);
            });
        },

        connectToLocal: () => {
            setState('activeNavTab', 'terminal');
            navigate('/terminal');
            StartLocalSession().catch((err: unknown) => {
                console.error('Local PTY start failed:', err);
                actions.notify('Local shell failed to start', 'error', err instanceof Error ? (err as Error).message : String(err));
            });
        },
        toggleFocusMode: () => setState('focusMode', (prev) => !prev),

        notify: (message, type = 'info', details, duration = 5000) => {
            const id = Math.random().toString(36).substring(2, 9);
            const notification: Notification = { id, message, type, details, duration };
            setState('notifications', (prev) => [...prev, notification]);

            if (duration > 0) {
                setTimeout(() => actions.dismissNotification(id), duration);
            }
        },

        dismissNotification: (id) => {
            setState('notifications', (prev) => prev.filter(n => n.id !== id));
        },

        updateTransfer: (transfer) => {
            const exists = state.transfers.find(t => t.id === transfer.id);
            if (!exists) {
                setState('transfers', (curr) => [...curr, transfer as Transfer]);
            } else {
                setState('transfers', (t) => t.id === transfer.id, (prev) => ({ ...prev, ...transfer }));
            }
        }
    };

    return (
        <AppContext.Provider value={[state, actions]}>
            {props.children}
        </AppContext.Provider>
    );
};

export function useApp(): AppContextType {
    const context = useContext(AppContext);
    if (!context) throw new Error('useApp must be used within AppProvider');
    return context;
}
