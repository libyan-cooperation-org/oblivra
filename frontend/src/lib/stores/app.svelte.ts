/**
 * OBLIVRA — App Store (Svelte 5 runes)
 *
 * Central application state. Replaces the SolidJS createStore + Context pattern.
 * Uses Svelte 5 $state runes for fine-grained reactivity.
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
    showCommandPalette = $state(false);
    currentUser = $state<any>(null);

    private _initialized = false;

    async init() {
        if (this._initialized) return;
        this._initialized = true;

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

        if (IS_BROWSER) {
            fetch('/api/v1/auth/me', { credentials: 'include' })
                .then(res => res.json())
                .then(user => { this.currentUser = user; })
                .catch(err => console.error('Failed to fetch user:', err));
        } else {
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
            if (data && typeof data.status === 'string') {
                const status = data.status.toLowerCase();
                if (status.includes('degraded')) {
                    this.systemHealth = {
                        status: 'degraded',
                        message: data.message || 'System performance is degraded (High Load)',
                    };
                } else if (status.includes('critical')) {
                    this.systemHealth = {
                        status: 'critical',
                        message: data.message || 'System is in a critical state (Saturation)',
                    };
                } else if (status === 'healthy' || status === 'online') {
                    this.systemHealth = { status: 'healthy' };
                }
            }
        });
    }

    // ── Actions ──────────────────────────────────────────────────────

    setHosts(hosts: Host[]) { this.hosts = hosts; }
    addHost(host: Host) { this.hosts = [...this.hosts, host]; }
    removeHost(id: string) { this.hosts = this.hosts.filter((h) => h.id !== id); }
    setActiveSession(id: string | null) { this.activeSessionId = id; }
    addSession(session: Session) { this.sessions = [...this.sessions, session]; }
    removeSession(id: string) { this.sessions = this.sessions.filter((s) => s.id !== id); }
    setVaultUnlocked(unlocked: boolean) { this.vaultUnlocked = unlocked; }
    toggleSidebar() { this.sidebarOpen = !this.sidebarOpen; }
    setTheme(theme: string) { this.theme = theme; }
    setLoading(loading: boolean) { this.loading = loading; }
    setActiveNavTab(tab: NavTab) { this.activeNavTab = tab; }
    setWorkspace(ws: Workspace) { this.workspace = ws; }
    toggleFocusMode() { this.focusMode = !this.focusMode; }
    toggleCommandPalette() { this.showCommandPalette = !this.showCommandPalette; }

    /**
     * navigate — programmatic navigation helper used by page components.
     * Navigates to a named route, optionally storing context in sessionStorage
     * for the target page to read (e.g. pre-selecting an agent by ID).
     *
     * Usage:
     *   appStore.navigate('agent-console', { id: row.id })
     *   appStore.navigate('/fleet')
     */
    navigate(routeOrTab: string, params?: Record<string, string>) {
        // Store params for the destination page to pick up
        if (params && Object.keys(params).length > 0) {
            sessionStorage.setItem('oblivra:nav_params', JSON.stringify(params));
        }
        // Resolve route: if it starts with '/' use directly, else prepend '/'
        const route = routeOrTab.startsWith('/') ? routeOrTab : '/' + routeOrTab;
        push(route);
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
            this.notify('Local shell requires the desktop binary.', 'error',
                'Download the OBLIVRA desktop app to use the local PTY terminal.');
            return;
        }
        try {
            const { StartLocalSession } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
            const sessionId = await StartLocalSession();
            if (sessionId) {
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
            this.notify('SSH connections require the desktop binary.', 'error',
                'Download the OBLIVRA desktop app to open SSH sessions.');
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

    /**
     * popOut — Spawns the current or specified route into a new native window.
     * Perfect for multi-monitor setups where an operator wants to keep the 
     * SIEM dashboard on one screen and the terminal on another.
     */
    async popOut(routeOrTab?: string, title?: string) {
        if (IS_BROWSER) {
            this.notify('Pop-out windows require the desktop app.', 'info');
            return;
        }

        const path = routeOrTab || window.location.hash.replace('#', '') || '/';
        const displayTitle = title || 'OBLIVRA ' + (routeOrTab || 'View');

        try {
            const { PopOut } = await import(
                '@wailsjs/github.com/kingknull/oblivrashell/internal/services/windowservice'
            );
            await PopOut(path, displayTitle);
        } catch (e) {
            console.error('Failed to pop out window:', e);
            this.notify('Failed to open new window', 'error');
        }
    }

    /**
     * launchSOCExperience — Orhcestrates a multi-window layout.
     * Automatically spawns 3 specialized windows:
     * 1. SIEM Operations (Full dashboard)
     * 2. Live Terminal (For active response)
     * 3. Threat Graph (For entity relationship hunting)
     */
    async launchSOCExperience() {
        if (IS_BROWSER) return;
        this.notify('Initializing SOC Multi-Monitor Experience...', 'success');
        
        // 1. SIEM Dashboard
        await this.popOut('/siem', 'OBLIVRA | SIEM Operations');
        
        // 2. Terminal Console
        setTimeout(() => this.popOut('/terminal', 'OBLIVRA | Command Console'), 500);
        
        // 3. Threat Graph
        setTimeout(() => this.popOut('/graph', 'OBLIVRA | Threat Graph'), 1000);
    }
}

export const appStore = new AppStore();
