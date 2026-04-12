/**
 * OBLIVRA — App Store (Svelte 5 runes)
 *
 * Central application state. Replaces the SolidJS createStore + Context pattern.
 * Uses Svelte 5 $state runes for fine-grained reactivity.
 *
 * Usage:
 *   import { appStore } from '@lib/stores/app.svelte';
 *   appStore.hosts; // reactive read
 *   appStore.setHosts(newHosts); // mutation
 */
import { subscribe } from '../bridge';
import { IS_BROWSER } from '../context';
import { push } from '../router.svelte';
import type {
    Host,
    Session,
    PluginPanel,
    PluginStatusIcon,
    Notification,
    Transfer,
    NavTab,
    Workspace,
    SystemHealth,
} from '../types';

class AppStore {
    // ── State ────────────────────────────────────────────────────────
    hosts = $state<Host[]>([]);
    sessions = $state<Session[]>([]);
    activeSessionId = $state<string | null>(null);
    vaultUnlocked = $state(false);
    theme = $state('dark');
    sidebarOpen = $state(true);
    loading = $state(false);
    activeNavTab = $state<NavTab>('dashboard');
    workspace = $state<Workspace>('Personal');
    pluginPanels = $state<PluginPanel[]>([]);
    pluginStatusIcons = $state<PluginStatusIcon[]>([]);
    focusMode = $state(false);
    notifications = $state<Notification[]>([]);
    transfers = $state<Transfer[]>([]);
    isMaximized = $state(false);
    systemHealth = $state<SystemHealth>({ status: 'healthy' });

    // ── Initialization ───────────────────────────────────────────────

    private _initialized = false;

    /**
     * Initialize event subscriptions and load initial state.
     * Must be called once from the root App.svelte onMount.
     */
    async init() {
        if (this._initialized) return;
        this._initialized = true;

        // Register ALL event listeners FIRST — before any async calls — so we
        // never miss events that fire during startup race conditions.
        subscribe('vault:unlocked', () => {
            this.vaultUnlocked = true;
            this.refreshHosts();
        });

        subscribe('vault:locked', () => {
            this.vaultUnlocked = false;
        });

        subscribe('security:fido2', (data: { message: string }) => {
            this.notify(data.message || 'Security Key Required', 'info', 'Touch your security key to authorize.');
        });

        subscribe('host:list_updated', () => {
            this.refreshHosts();
        });

        // Initial vault/host state — desktop only.
        if (!IS_BROWSER) {
            try {
                const { IsUnlocked } = await import(
                    '@wailsjs/github.com/kingknull/oblivrashell/internal/services/vaultservice'
                );
                const unlocked = await IsUnlocked();
                this.vaultUnlocked = unlocked;
                if (unlocked) {
                    await this.refreshHosts();
                }
            } catch (_) {
                // Vault may not be configured yet
            }
        }

        subscribe('session:started', (data: { id: string; hostId: string; label: string }) => {
            if (this.sessions.find((s) => s.id === data.id)) return;
            const newSession: Session = {
                id: data.id,
                hostId: data.hostId,
                hostLabel: data.label,
                status: 'active',
                startedAt: new Date().toISOString(),
            };
            this.sessions = [...this.sessions, newSession];
            this.activeSessionId = data.id;
        });

        subscribe('session:closed', (sessionId: string) => {
            this.sessions = this.sessions.filter((s) => s.id !== sessionId);
            if (this.activeSessionId === sessionId) {
                const remaining = this.sessions;
                this.activeSessionId = remaining.length > 0 ? remaining[0].id : null;
            }
        });

        subscribe('recording:stopped', (sessionId: string) => {
            this.sessions = this.sessions.map((s) =>
                s.id === sessionId ? { ...s, isRecording: false } : s
            );
        });

        subscribe('ui.register_panel', (data: PluginPanel) => {
            if (this.pluginPanels.find((p) => p.plugin_id === data.plugin_id && p.panel_id === data.panel_id)) return;
            this.pluginPanels = [...this.pluginPanels, data];
        });

        subscribe('ui.add_status_icon', (data: PluginStatusIcon) => {
            if (this.pluginStatusIcons.find((i) => i.plugin_id === data.plugin_id && i.icon_id === data.icon_id)) return;
            this.pluginStatusIcons = [...this.pluginStatusIcons, data];
        });

        subscribe('transfer:updated', (data: Partial<Transfer> & { id: string }) => {
            const idx = this.transfers.findIndex((t) => t.id === data.id);
            if (idx === -1) {
                this.transfers = [...this.transfers, data as Transfer];
            } else {
                this.transfers = this.transfers.map((t) =>
                    t.id === data.id ? { ...t, ...data } : t
                );
            }
        });

        subscribe('transfer:completed', (data: { id: string; name: string }) => {
            this.transfers = this.transfers.map((t) =>
                t.id === data.id ? { ...t, status: 'completed' as const } : t
            );
            this.notify(`Transfer ${data.name} completed`, 'success');
        });

        subscribe('transfer:failed', (data: { id: string; name: string; error: string }) => {
            this.transfers = this.transfers.map((t) =>
                t.id === data.id ? { ...t, status: 'failed' as const, error: data.error } : t
            );
            this.notify(`Transfer ${data.name} failed: ${data.error}`, 'error');
        });

        subscribe('health_status_changed', (data: any) => {
            // Data can be a host's health or global service health report.
            // If it's the ingest service reporting an error, we mark as degraded.
            if (data && typeof data.status === 'string') {
                const status = data.status.toLowerCase();
                if (status.includes('degraded')) {
                    this.systemHealth = { 
                        status: 'degraded', 
                        message: data.message || 'System performance is degraded (High Load)' 
                    };
                } else if (status.includes('critical')) {
                    this.systemHealth = { 
                        status: 'critical', 
                        message: data.message || 'System is in a critical state (Saturation)' 
                    };
                } else if (status === 'healthy' || status === 'online') {
                    this.systemHealth = { status: 'healthy' };
                }
            }
        });
    }

    // ── Actions ──────────────────────────────────────────────────────

    setHosts(hosts: Host[]) {
        this.hosts = hosts;
    }

    addHost(host: Host) {
        this.hosts = [...this.hosts, host];
    }

    removeHost(id: string) {
        this.hosts = this.hosts.filter((h) => h.id !== id);
    }

    setActiveSession(id: string | null) {
        this.activeSessionId = id;
    }

    addSession(session: Session) {
        this.sessions = [...this.sessions, session];
    }

    removeSession(id: string) {
        this.sessions = this.sessions.filter((s) => s.id !== id);
    }

    setVaultUnlocked(unlocked: boolean) {
        this.vaultUnlocked = unlocked;
    }

    toggleSidebar() {
        this.sidebarOpen = !this.sidebarOpen;
    }

    setTheme(theme: string) {
        this.theme = theme;
    }

    setLoading(loading: boolean) {
        this.loading = loading;
    }

    setActiveNavTab(tab: NavTab) {
        this.activeNavTab = tab;
    }

    setWorkspace(ws: Workspace) {
        this.workspace = ws;
    }

    toggleFocusMode() {
        this.focusMode = !this.focusMode;
    }

    async refreshHosts() {
        if (IS_BROWSER) return;
        try {
            const { ListHosts } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/hostservice');
            const hosts = await ListHosts();
            this.hosts = hosts || [];
        } catch (e) {
            console.error('Failed to refresh hosts', e);
        }
    }

    async connectToLocal() {
        if (IS_BROWSER) {
            this.notify('Local shell requires the desktop binary.', 'error');
            return;
        }
        try {
            const { StartLocalSession } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
            const sessionId = await StartLocalSession();
            if (sessionId) {
                // The session:started event handler in init() will add it to the list
                this.setActiveSession(sessionId);
                this.setActiveNavTab('terminal');
                push('/terminal');
            }
        } catch (e: any) {
            this.notify('Failed to start local shell', 'error', e.message);
        }
    }

    async connectToHost(hostId: string) {
        if (IS_BROWSER) {
            this.notify('SSH connections require the desktop binary.', 'error');
            return;
        }
        let host = this.hosts.find((h) => h.id === hostId);
        if (!host) host = this.hosts.find((h) => h.hostname === hostId || h.label === hostId);
        if (!host) {
            this.notify(`Host not found: ${hostId}. Please add it first.`, 'error');
            return;
        }
        
        try {
            const { Connect } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice');
            const sessionId = await Connect(host.id);
            if (sessionId) {
                // The session:started event handler in init() will add it to the list
                this.setActiveSession(sessionId);
                this.setActiveNavTab('terminal');
                push('/terminal');
            }
        } catch (e: any) {
            this.notify(`Connection failed to ${host.label}`, 'error', e.message);
        }
    }


    notify(message: string, type: Notification['type'] = 'info', details?: string, duration = 5000) {
        const id = Math.random().toString(36).substring(2, 9);
        const notification: Notification = { id, message, type, details, duration };
        this.notifications = [...this.notifications, notification];

        if (duration > 0) {
            setTimeout(() => this.dismissNotification(id), duration);
        }
    }

    dismissNotification(id: string) {
        this.notifications = this.notifications.filter((n) => n.id !== id);
    }

    updateTransfer(transfer: Partial<Transfer> & { id: string }) {
        const idx = this.transfers.findIndex((t) => t.id === transfer.id);
        if (idx === -1) {
            this.transfers = [...this.transfers, transfer as Transfer];
        } else {
            this.transfers = this.transfers.map((t) =>
                t.id === transfer.id ? { ...t, ...transfer } : t
            );
        }
    }
}

export const appStore = new AppStore();
