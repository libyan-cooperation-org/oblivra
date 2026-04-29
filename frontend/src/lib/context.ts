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

/**
 * Detects if we are running inside a Wails v3 WebView.
 *
 * Wails v3 sets window._wails AFTER the page loads (WindowLoadFinished hook),
 * so it cannot be relied upon for synchronous detection.
 *
 * However, WebKit2GTK (Linux/macOS) ALWAYS injects
 * `window.webkit.messageHandlers.external` synchronously before any JS runs.
 * Windows WebView2 injects `window.chrome.webview`.
 * These are the actual IPC channels Wails uses — if they exist, we are inside
 * a Wails (or other native) WebView.
 *
 * Fallbacks for older Wails v2 globals are also kept.
 */
function detectContext(): AppContext {
    if (typeof window === 'undefined') return 'browser';

    const w = window as any;

    const isWails =
        // Wails v3 Linux/macOS: WebKit2GTK IPC (synchronous, always present
        // before any JS runs).
        !!w.webkit?.messageHandlers?.external
        // Wails v3 Windows: WebView2 IPC
        || !!w.chrome?.webview
        // Wails v3 globals — `_wails` (single underscore) is the v3
        // namespace; v2 used double-underscore. We check both.
        || !!w._wails
        || !!w.__WAILS__
        // Wails v2 legacy globals
        || !!w.runtime
        || !!w.wails;

    if (!isWails) return 'browser';
    const remoteServer = localStorage.getItem('oblivra:remote_server');
    if (remoteServer && remoteServer.trim() !== '') return 'hybrid';
    return 'desktop';
}

export const APP_CONTEXT = detectContext() as AppContext;

export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER  = APP_CONTEXT === 'browser';
export const IS_HYBRID   = APP_CONTEXT === 'hybrid';

/** No-op shim — kept for compatibility with initBridge() call. */
export async function initContext(): Promise<void> { /* detection is synchronous */ }

/** Returns the current context. */
export function getContext(): AppContext { return APP_CONTEXT; }


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
    // Phase 36.x: /compliance and /governance entries removed (compliance packs deleted).
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
    // Phase 36: /playbook-builder removed (SOAR scope cut).
    { path: '/ransomware',             availability: 'both' },
    { path: '/ransomware-ui',          availability: 'both' },
    { path: '/trust',                  availability: 'both' },
    { path: '/ops',                    availability: 'both' },
    { path: '/topology',               availability: 'both' },
    // Phase 36: /ai-assistant removed (AI Assistant scope cut).
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
    // Phase 36: /plugins removed (plugin framework deleted).
    { path: '/shortcuts',              availability: 'both' },
    { path: '/admin',                  availability: 'both' },

    // ── Desktop-only ──────────────────────────────────────────────────────
    {
        path: '/shell',
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
    {
        path: '/operator',
        availability: 'desktop',
        unavailableReason: 'Operator Mode requires a direct PTY connection to the host via the desktop binary.',
    },

    // ── Browser/Server-only ───────────────────────────────────────────────
    {
        path: '/soc',
        availability: 'both',
        unavailableReason: 'The SOC workspace requires multi-analyst server mode.',
    },
    {
        path: '/agents',
        availability: 'both',
        unavailableReason: 'Agent fleet management requires the server backend to receive agent registrations.',
    },
    {
        path: '/fleet',
        availability: 'both',
        unavailableReason: 'Fleet management requires the OBLIVRA server backend.',
    },
    {
        path: '/fleet-management',
        availability: 'both',
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
