/**
 * OBLIVRA — Phase 0.5: Desktop vs Browser Context
 *
 * Single source of truth for deployment context detection.
 * Pure TypeScript — no framework dependency.
 *
 *   DESKTOP → Wails native binary (direct PTY, OS keychain, local FS)
 *   BROWSER → Served from OBLIVRA server (SOC, fleet management)
 *   HYBRID  → Desktop binary connected to a remote OBLIVRA server
 */

// ── Context detection ──────────────────────────────────────────────────────

export type AppContext = 'desktop' | 'browser' | 'hybrid';

function detectContext(): AppContext {
    const isWails = !!(window as any).__WAILS__ || !!(window as any).runtime;
    if (!isWails) return 'browser';
    const remoteServer = localStorage.getItem('oblivra:remote_server');
    if (remoteServer && remoteServer.trim() !== '') return 'hybrid';
    return 'desktop';
}

export const APP_CONTEXT: AppContext = detectContext();

export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER  = APP_CONTEXT === 'browser';
export const IS_HYBRID   = APP_CONTEXT === 'hybrid';

// ── Route availability matrix ─────────────────────────────────────────────

export type RouteAvailability = 'desktop' | 'browser' | 'both';

export interface RouteConfig {
    path: string;
    availability: RouteAvailability;
    unavailableReason?: string;
}

export const ROUTE_AVAILABILITY: RouteConfig[] = [

    // ── Available in all contexts ─────────────────────────────────────────
    { path: '/dashboard',              availability: 'both' },
    { path: '/siem',                   availability: 'both' },
    { path: '/alerts',                 availability: 'both' },
    { path: '/alert-management',       availability: 'both' },
    { path: '/siem-search',            availability: 'both' },
    { path: '/mitre-heatmap',          availability: 'both' },
    { path: '/compliance',             availability: 'both' },
    { path: '/governance',             availability: 'both' },
    { path: '/features',               availability: 'both' },
    { path: '/risk',                   availability: 'both' },
    { path: '/vault',                  availability: 'both' },
    { path: '/executive',              availability: 'both' },
    { path: '/monitoring',             availability: 'both' },
    { path: '/forensics',              availability: 'both' },
    { path: '/remote-forensics',       availability: 'both' },
    { path: '/ledger',                 availability: 'both' },
    { path: '/ueba',                   availability: 'both' },
    { path: '/ueba-overview',          availability: 'both' },
    { path: '/ndr',                    availability: 'both' },
    { path: '/ndr-overview',           availability: 'both' },
    { path: '/graph',                  availability: 'both' },
    { path: '/threat-hunter',          availability: 'both' },
    { path: '/threat-intel',           availability: 'both' },
    { path: '/threat-intel-dashboard', availability: 'both' },
    { path: '/enrichment',             availability: 'both' },
    { path: '/credentials',            availability: 'both' },
    { path: '/purple-team',            availability: 'both' },
    { path: '/response',               availability: 'both' },
    { path: '/escalation',             availability: 'both' },
    { path: '/playbook-builder',       availability: 'both' },
    { path: '/ransomware',             availability: 'both' },
    { path: '/ransomware-ui',          availability: 'both' },
    { path: '/trust',                  availability: 'both' },
    { path: '/ops',                    availability: 'both' },
    { path: '/topology',               availability: 'both' },
    { path: '/ai-assistant',           availability: 'both' },
    { path: '/workspace',              availability: 'both' },
    { path: '/settings',               availability: 'both' },
    { path: '/analytics',              availability: 'both' },
    { path: '/fusion',                 availability: 'both' },
    { path: '/war-mode',               availability: 'both' },
    { path: '/data-destruction',       availability: 'both' },
    { path: '/temporal-integrity',     availability: 'both' },
    { path: '/lineage',                availability: 'both' },
    { path: '/decisions',              availability: 'both' },
    { path: '/simulation',             availability: 'both' },
    { path: '/response-replay',        availability: 'both' },
    { path: '/entity',                 availability: 'both' },
    { path: '/license',                availability: 'both' },
    { path: '/team',                   availability: 'both' },
    { path: '/hosts',                  availability: 'both' },
    { path: '/plugins',                availability: 'both' },

    // ── Desktop-only ──────────────────────────────────────────────────────
    {
        path: '/terminal',
        availability: 'desktop',
        unavailableReason: 'Local PTY terminal requires the desktop binary. In browser mode, use the Fleet page to connect to hosts.',
    },
    {
        path: '/tunnels',
        availability: 'desktop',
        unavailableReason: 'Port-forwarding tunnels require a direct SSH connection from the desktop binary.',
    },
    {
        path: '/recordings',
        availability: 'desktop',
        unavailableReason: 'Session recordings are stored locally by the desktop binary.',
    },
    {
        path: '/snippets',
        availability: 'desktop',
        unavailableReason: 'Command snippets are stored in the local vault.',
    },
    {
        path: '/notes',
        availability: 'desktop',
        unavailableReason: 'Notes are persisted in the local encrypted store.',
    },
    {
        path: '/sync',
        availability: 'desktop',
        unavailableReason: 'Sync configures replication from this desktop node to a remote server.',
    },
    {
        path: '/offline-update',
        availability: 'desktop',
        unavailableReason: 'Offline update bundles are applied to the local binary only.',
    },

    // ── Browser/Server-only ───────────────────────────────────────────────
    {
        path: '/soc',
        availability: 'browser',
        unavailableReason: 'The SOC workspace requires multi-analyst server mode.',
    },
    {
        path: '/agents',
        availability: 'browser',
        unavailableReason: 'Agent fleet management requires the server backend to receive agent registrations.',
    },
    {
        path: '/fleet',
        availability: 'browser',
        unavailableReason: 'Fleet management requires the OBLIVRA server backend.',
    },
    {
        path: '/fleet-management',
        availability: 'browser',
        unavailableReason: 'Centralized fleet management requires the server backend.',
    },
    {
        path: '/identity',
        availability: 'browser',
        unavailableReason: 'User & role administration requires the server backend with OIDC/SAML configured.',
    },
    {
        path: '/identity-admin',
        availability: 'browser',
        unavailableReason: 'Enterprise identity administration (OIDC/SAML/LDAP) requires the server backend.',
    },
];

// ── Helpers ───────────────────────────────────────────────────────────────

export function isRouteAvailable(path: string): boolean {
    const config = findRouteConfig(path);
    if (!config) return true;
    switch (config.availability) {
        case 'both':    return true;
        case 'desktop': return IS_DESKTOP || IS_HYBRID;
        case 'browser': return IS_BROWSER || IS_HYBRID;
        default:        return true;
    }
}

export function routeUnavailableReason(path: string): string | null {
    if (isRouteAvailable(path)) return null;
    const config = findRouteConfig(path);
    return config?.unavailableReason ?? 'This feature is not available in the current deployment mode.';
}

function findRouteConfig(path: string): RouteConfig | undefined {
    const exact = ROUTE_AVAILABILITY.find(r => r.path === path);
    if (exact) return exact;
    return ROUTE_AVAILABILITY.find(r => path.startsWith(r.path + '/'));
}

// ── Service capabilities ──────────────────────────────────────────────────

export interface ServiceCapabilities {
    localTerminal:    boolean;
    osKeychain:       boolean;
    localSftp:        boolean;
    enterpriseAuth:   boolean;
    agentFleet:       boolean;
    clustering:       boolean;
    socCollaboration: boolean;
    remoteServer:     boolean;
    remoteServerUrl:  string | null;
}

export function getServiceCapabilities(): ServiceCapabilities {
    const remoteUrl = localStorage.getItem('oblivra:remote_server') ?? null;
    return {
        localTerminal:    IS_DESKTOP || IS_HYBRID,
        osKeychain:       IS_DESKTOP,
        localSftp:        IS_DESKTOP || IS_HYBRID,
        enterpriseAuth:   IS_BROWSER || IS_HYBRID,
        agentFleet:       IS_BROWSER || IS_HYBRID,
        clustering:       IS_BROWSER || IS_HYBRID,
        socCollaboration: IS_BROWSER || IS_HYBRID,
        remoteServer:     IS_HYBRID,
        remoteServerUrl:  IS_HYBRID ? remoteUrl : null,
    };
}

// ── Hybrid mode ───────────────────────────────────────────────────────────

export function configureHybridMode(serverUrl: string): void {
    if (!IS_DESKTOP) {
        console.warn('[context] configureHybridMode() called in non-desktop context');
        return;
    }
    localStorage.setItem('oblivra:remote_server', serverUrl.trim());
    window.location.reload();
}

export function disconnectHybridMode(): void {
    localStorage.removeItem('oblivra:remote_server');
    window.location.reload();
}

export function getRemoteServerUrl(): string | null {
    if (!IS_HYBRID) return null;
    return localStorage.getItem('oblivra:remote_server');
}
