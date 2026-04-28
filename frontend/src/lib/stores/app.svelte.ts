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

// ── Operator Profile types (Phase 32) ─────────────────────────────
//
// A profile is a bundled rule-set. Picking a profile flips ~9 settings
// at once instead of forcing the operator to tune each individually.
// See docs/ux/operator-profiles.md for the design rationale.

export type OperatorProfileId =
    | 'soc-analyst'
    | 'threat-hunter'
    | 'incident-commander'
    | 'msp-admin'
    | 'custom';

export interface OperatorProfileRules {
    /** Where pressing Home / clicking the OBLIVRA wordmark takes you. */
    homeRoute: string;
    /** UI density. */
    defaultDensity: 'comfortable' | 'compact';
    /** Which metric the dashboards prioritise visually. */
    primaryMetric: 'mttr' | 'fp-rate' | 'evidence-latency' | 'hunt-yield';
    /** Single-screen vs war-room (multi-monitor) bias. */
    layoutMode: 'single' | 'war-room';
    /** When true, ⌘K palette is pre-focused on app idle so typing
     *  immediately starts a search. */
    paletteFront: boolean;
    /** Vim-style g+letter navigation leader keys. */
    vimLeader: boolean;
    /** How prominent the active-tenant indicator is. */
    tenantChrome: 'breadcrumb' | 'badge' | 'switcher-bar';
    /** Crisis Mode behaviour. */
    crisisAffordance: 'banner' | 'fullscreen-takeover';
    /** Default alert-list filter floor. Profile cannot prevent the user
     *  from temporarily showing more — it's a default, not a wall. */
    alertNoiseFloor: 'critical-only' | 'high+' | 'medium+' | 'all';
}

/**
 * Shipped presets. Treat as immutable — selecting one COPIES it into
 * `profileRules`, then the operator can fork via 'custom'.
 */
export const PROFILE_PRESETS: Record<Exclude<OperatorProfileId, 'custom'>, OperatorProfileRules> = {
    'soc-analyst': {
        homeRoute: '/alert-management',
        defaultDensity: 'comfortable',
        primaryMetric: 'mttr',
        layoutMode: 'single',
        paletteFront: false,
        vimLeader: false,
        tenantChrome: 'breadcrumb',
        crisisAffordance: 'banner',
        alertNoiseFloor: 'medium+',
    },
    'threat-hunter': {
        homeRoute: '/siem-search',
        defaultDensity: 'compact',
        primaryMetric: 'hunt-yield',
        layoutMode: 'single',
        paletteFront: true,
        vimLeader: true,
        tenantChrome: 'breadcrumb',
        crisisAffordance: 'banner',
        alertNoiseFloor: 'all',
    },
    'incident-commander': {
        homeRoute: '/war-mode',
        defaultDensity: 'comfortable',
        primaryMetric: 'mttr',
        layoutMode: 'war-room',
        paletteFront: false,
        vimLeader: false,
        tenantChrome: 'badge',
        crisisAffordance: 'fullscreen-takeover',
        alertNoiseFloor: 'critical-only',
    },
    'msp-admin': {
        homeRoute: '/admin',
        defaultDensity: 'compact',
        primaryMetric: 'fp-rate',
        layoutMode: 'single',
        paletteFront: true,
        vimLeader: true,
        tenantChrome: 'switcher-bar',
        crisisAffordance: 'banner',
        alertNoiseFloor: 'high+',
    },
};

export const PROFILE_LABELS: Record<OperatorProfileId, { name: string; subtitle: string }> = {
    'soc-analyst':        { name: 'SOC Analyst',        subtitle: 'Alert queue · triage · MTTR · single screen' },
    'threat-hunter':      { name: 'Threat Hunter',      subtitle: 'Search-first · palette · vim leader · all alerts' },
    'incident-commander': { name: 'Incident Commander', subtitle: 'War-room · multi-monitor · crisis-led · critical-only' },
    'msp-admin':          { name: 'MSP / Platform Admin', subtitle: 'Multi-tenant · fast switcher · FP-rate · compact' },
    'custom':             { name: 'Custom',             subtitle: 'Per-rule overrides' },
};

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

    // ── Multi-tenant context (Phase 30.4d) ──────────────────────
    // The tenant currently being viewed in the UI. `null` = "all
    // tenants" (platform-admin perspective). Persisted to localStorage
    // so the operator returns to the same tenant scope after reload.
    // Backend requests can read this via window.__currentTenantId or
    // the new bridge helper getCurrentTenantId() once a tenant is
    // chosen — until that wiring lands the value drives UI scoping
    // only (banner, breadcrumbs, ActivityFeed filter).
    currentTenantId = $state<string | null>(null);

    // ── Layout chrome toggle ─────────────────────────────────────
    // `useGroupedNav = true` swaps the legacy `CommandRail.svelte` for
    // the new `AppSidebar.svelte` + `BottomDock.svelte` pair (v1.5.0
    // UX redesign). Persisted to localStorage so the operator's choice
    // survives reloads. Defaults to true on fresh installs but a user
    // who flipped it off keeps it off. Initial value is hydrated in
    // `init()` so we never reach into `localStorage` from a class field
    // initializer (those run pre-window-ready in some bundler configs).
    useGroupedNav = $state<boolean>(true);

    // ── Density toggle (Phase 31, UIUX_IMPROVEMENTS.md P0 #2) ────
    // 'comfortable' is the new default — 12px body, 10px micro labels.
    // 'compact' is the legacy SOC-dense layout — 11px body, 9px micro.
    // Drives `body[data-density=…]` which scopes the `--fs-*` ramp in
    // app.css. Persisted to localStorage so the choice sticks.
    density = $state<'comfortable' | 'compact'>('comfortable');

    // ── Operator Profile (Phase 32) ──────────────────────────────
    // Bundled rule-set that drives home route, default chrome, alert
    // noise floor, palette front-door behaviour, vim leader, tenant
    // chrome, and crisis affordance. The whole UI re-aligns when the
    // profile changes. See docs/ux/operator-profiles.md.
    //
    // Profiles:
    //   • soc-analyst        — Maya: alert queue, MTTR, 2-click parity
    //   • threat-hunter      — Daniel: search-first, palette-front, vim
    //   • incident-commander — Rita: war-room, full-screen crisis
    //   • msp-admin          — multi-tenant operator, fast switcher
    //   • custom             — user-edited overrides
    profile = $state<OperatorProfileId>('soc-analyst');
    profileRules = $state<OperatorProfileRules>(PROFILE_PRESETS['soc-analyst']);
    /** Set true once the user has explicitly chosen a profile (or
     *  dismissed the wizard). Drives the first-run modal. */
    profileChosen = $state<boolean>(false);

    private _initialized = false;

    async init() {
        if (this._initialized) return;
        this._initialized = true;

        // Hydrate the layout-chrome preference now that `window` is ready.
        this.useGroupedNav = this._loadGroupedNavPref();

        // Hydrate density preference + push to <body> immediately so the
        // first paint uses the correct ramp.
        this.density = this._loadDensityPref();
        this._applyDensityToDom();

        // Hydrate Operator Profile. If unset, profileChosen stays false
        // and the first-run wizard fires from App.svelte.
        this._loadProfile();
        this._applyProfileToDom();

        // Hydrate tenant scope. Falls back to null ("all tenants") if
        // no preference is stored.
        try {
            const tid = localStorage.getItem('oblivra:currentTenantId');
            if (tid) this.currentTenantId = tid;
        } catch { /* private mode */ }

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

    /** Switch the active tenant context. Pass null for "all tenants". */
    setCurrentTenant(id: string | null) {
        this.currentTenantId = id;
        try {
            if (id) localStorage.setItem('oblivra:currentTenantId', id);
            else    localStorage.removeItem('oblivra:currentTenantId');
        } catch { /* private mode */ }
    }

    /** Toggle the new grouped nav (sidebar + bottom dock) vs legacy CommandRail. */
    toggleGroupedNav() {
        this.useGroupedNav = !this.useGroupedNav;
        try {
            localStorage.setItem('oblivra:useGroupedNav', this.useGroupedNav ? '1' : '0');
        } catch { /* quota / private mode */ }
    }

    /** Load grouped-nav preference from localStorage. Defaults to true (new UX). */
    private _loadGroupedNavPref(): boolean {
        if (typeof localStorage === 'undefined') return true;
        try {
            const v = localStorage.getItem('oblivra:useGroupedNav');
            if (v === null) return true;          // no preference → new UX
            return v === '1';
        } catch {
            return true;
        }
    }

    /** Switch density between comfortable (12/10) and compact (11/9). */
    setDensity(d: 'comfortable' | 'compact') {
        this.density = d;
        try {
            localStorage.setItem('oblivra:density', d);
        } catch { /* private mode */ }
        this._applyDensityToDom();
    }

    toggleDensity() {
        this.setDensity(this.density === 'comfortable' ? 'compact' : 'comfortable');
    }

    private _loadDensityPref(): 'comfortable' | 'compact' {
        if (typeof localStorage === 'undefined') return 'comfortable';
        try {
            const v = localStorage.getItem('oblivra:density');
            return v === 'compact' ? 'compact' : 'comfortable';
        } catch {
            return 'comfortable';
        }
    }

    private _applyDensityToDom() {
        if (typeof document === 'undefined') return;
        document.body.setAttribute('data-density', this.density);
    }

    /**
     * Apply a shipped preset profile. Copies the rule-set into local
     * state, persists to localStorage, and updates body data-* hooks.
     * Also adjusts density to match the profile's preferred default.
     */
    setProfile(id: OperatorProfileId) {
        this.profile = id;
        if (id !== 'custom') {
            this.profileRules = { ...PROFILE_PRESETS[id] };
            // Snap density to the profile's default — operator can still
            // override afterwards via Settings.
            this.setDensity(this.profileRules.defaultDensity);
        }
        this.profileChosen = true;
        this._persistProfile();
        this._applyProfileToDom();
    }

    /**
     * Override a single rule (auto-promotes the profile to 'custom' so
     * the user's tweak isn't overwritten by the next preset reload).
     */
    setProfileRule<K extends keyof OperatorProfileRules>(key: K, value: OperatorProfileRules[K]) {
        this.profileRules = { ...this.profileRules, [key]: value };
        if (this.profile !== 'custom') {
            this.profile = 'custom';
        }
        if (key === 'defaultDensity') {
            this.setDensity(value as 'comfortable' | 'compact');
        }
        this.profileChosen = true;
        this._persistProfile();
        this._applyProfileToDom();
    }

    /** Hide the first-run wizard without picking a profile. */
    dismissProfileWizard() {
        this.profileChosen = true;
        this._persistProfile();
    }

    private _loadProfile() {
        if (typeof localStorage === 'undefined') return;
        try {
            const id = localStorage.getItem('oblivra:profile') as OperatorProfileId | null;
            const rulesRaw = localStorage.getItem('oblivra:profileRules');
            const chosen = localStorage.getItem('oblivra:profileChosen') === '1';
            if (id && (id in PROFILE_PRESETS || id === 'custom')) {
                this.profile = id;
            }
            if (rulesRaw) {
                try {
                    const parsed = JSON.parse(rulesRaw) as Partial<OperatorProfileRules>;
                    // Merge over the current preset so a missing key gets
                    // a sensible default after a schema bump.
                    this.profileRules = { ...this.profileRules, ...parsed };
                } catch { /* corrupt — fall through to preset */ }
            } else if (id && id !== 'custom') {
                this.profileRules = { ...PROFILE_PRESETS[id] };
            }
            this.profileChosen = chosen;
        } catch { /* private mode */ }
    }

    private _persistProfile() {
        if (typeof localStorage === 'undefined') return;
        try {
            localStorage.setItem('oblivra:profile', this.profile);
            localStorage.setItem('oblivra:profileRules', JSON.stringify(this.profileRules));
            localStorage.setItem('oblivra:profileChosen', this.profileChosen ? '1' : '0');
        } catch { /* private mode */ }
    }

    /**
     * Push profile rules to the DOM as data attributes so CSS / global
     * keybinding handlers can react without importing the store.
     */
    private _applyProfileToDom() {
        if (typeof document === 'undefined') return;
        document.body.setAttribute('data-profile', this.profile);
        document.body.setAttribute('data-layout-mode', this.profileRules.layoutMode);
        document.body.setAttribute('data-tenant-chrome', this.profileRules.tenantChrome);
        document.body.setAttribute('data-crisis-affordance', this.profileRules.crisisAffordance);
        document.body.setAttribute('data-noise-floor', this.profileRules.alertNoiseFloor);
    }

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
                // Optimistic add to ensure UI updates immediately even if event is slightly delayed
                if (!this.sessions.find(s => s.id === sessionId)) {
                    this.sessions = [...this.sessions, {
                        id: sessionId,
                        hostId: 'local',
                        status: 'active',
                        hostLabel: 'Local Terminal',
                        startedAt: new Date().toISOString()
                    }];
                }
                this.setActiveSession(sessionId);
                this.setActiveNavTab('terminal');
                push('/shell');
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
                push('/shell');
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
        setTimeout(() => this.popOut('/shell', 'OBLIVRA | Command Console'), 500);
        
        // 3. Threat Graph
        setTimeout(() => this.popOut('/graph', 'OBLIVRA | Threat Graph'), 1000);
    }
}

export const appStore = new AppStore();
