/**
 * OBLIVRA — Phase 0.5: Desktop vs Browser Context
 *
 * This module is the single source of truth for which deployment
 * context the frontend is running in:
 *
 *   DESKTOP  → Wails native binary (direct PTY, OS keychain, local FS)
 *   BROWSER  → Served from OBLIVRA server (SOC collaboration, fleet mgmt)
 *   HYBRID   → Desktop binary connected to a remote OBLIVRA server
 *
 * Every routing decision, feature gate, and service registration
 * flows from this one constant.
 */

// ── Context detection ──────────────────────────────────────────────────────

/**
 * Wails injects `window.__WAILS__` into every Wails-hosted WebView.
 * A plain browser (Chrome, Firefox, Safari) never has this property.
 */
export type AppContext = 'desktop' | 'browser' | 'hybrid';

function detectContext(): AppContext {
    const isWails = !!(window as any).__WAILS__ || !!(window as any).runtime;

    if (!isWails) return 'browser';

    // Hybrid: desktop binary but a remote server URL has been configured.
    // The server URL is written to localStorage by the Settings page when
    // the operator enters a remote OBLIVRA server address.
    const remoteServer = localStorage.getItem('oblivra:remote_server');
    if (remoteServer && remoteServer.trim() !== '') return 'hybrid';

    return 'desktop';
}

/**
 * Stable at module load time — does not change during the session.
 * Import this instead of calling detectContext() repeatedly.
 */
export const APP_CONTEXT: AppContext = detectContext();

export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER  = APP_CONTEXT === 'browser';
export const IS_HYBRID   = APP_CONTEXT === 'hybrid';

// ── Route availability matrix ─────────────────────────────────────────────

/**
 * Defines which deployment contexts each route is available in.
 *
 *   desktop  → Wails binary only (requires local PTY / OS keychain)
 *   browser  → Remote server only (requires multi-user backend)
 *   both     → Available everywhere
 */
export type RouteAvailability = 'desktop' | 'browser' | 'both';

export interface RouteConfig {
    path: string;
    availability: RouteAvailability;
    /** Human-readable reason shown if user tries to access unavailable route */
    unavailableReason?: string;
}

export const ROUTE_AVAILABILITY: RouteConfig[] = [
    // ── Available everywhere ──────────────────────────────────────────────
    { path: '/dashboard',         availability: 'both' },
    { path: '/siem',              availability: 'both' },
    { path: '/alerts',            availability: 'both' },
    { path: '/compliance',        availability: 'both' },
    { path: '/governance',        availability: 'both' },
    { path: '/vault',             availability: 'both' },
    { path: '/executive',         availability: 'both' },
    { path: '/monitoring',        availability: 'both' },
    { path: '/forensics',         availability: 'both' },
    { path: '/ledger',            availability: 'both' },
    { path: '/mitre-heatmap',     availability: 'both' },
    { path: '/purple-team',       availability: 'both' },
    { path: '/threat-hunter',     availability: 'both' },
    { path: '/ueba',              availability: 'both' },
    { path: '/ndr',               availability: 'both' },
    { path: '/graph',             availability: 'both' },
    { path: '/response',          availability: 'both' },
    { path: '/ransomware',        availability: 'both' },
    { path: '/trust',             availability: 'both' },
    { path: '/ops',               availability: 'both' },
    { path: '/topology',          availability: 'both' },
    { path: '/ai-assistant',      availability: 'both' },
    { path: '/settings',          availability: 'both' },
    { path: '/workspace',         availability: 'both' },
    { path: '/analytics',         availability: 'both' },
    { path: '/war-mode',          availability: 'both' },
    { path: '/data-destruction',  availability: 'both' },
    { path: '/temporal-integrity',availability: 'both' },
    { path: '/lineage',           availability: 'both' },
    { path: '/decisions',         availability: 'both' },
    { path: '/simulation',        availability: 'both' },
    { path: '/response-replay',   availability: 'both' },

    // ── Desktop-only ──────────────────────────────────────────────────────
    {
        path: '/terminal',
        availability: 'desktop',
        unavailableReason: 'Local PTY terminal requires the desktop binary. In browser mode, connect to hosts via the Fleet page.',
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
        path: '/fleet',
        availability: 'browser',
        unavailableReason: 'Fleet management requires the OBLIVRA server backend. Launch the server and connect via browser.',
    },
    {
        path: '/identity',
        availability: 'browser',
        unavailableReason: 'User & role administration requires the server backend with OIDC/SAML configured.',
    },
    {
        path: '/agents',
        availability: 'browser',
        unavailableReason: 'Agent fleet management requires the server backend to receive agent registrations.',
    },
    {
        path: '/soc',
        availability: 'browser',
        unavailableReason: 'The SOC workspace requires multi-analyst server mode.',
    },
    {
        path: '/credentials',
        availability: 'browser',
        unavailableReason: 'Credential intelligence requires the server-side threat intel feeds.',
    },
];

/**
 * Returns true if the given path is accessible in the current context.
 * Checks exact match and prefix match (for nested routes like /siem/search).
 */
export function isRouteAvailable(path: string): boolean {
    const config = findRouteConfig(path);
    if (!config) return true; // unknown routes are allowed by default

    switch (config.availability) {
        case 'both':    return true;
        case 'desktop': return IS_DESKTOP || IS_HYBRID;
        case 'browser': return IS_BROWSER || IS_HYBRID;
        default:        return true;
    }
}

/**
 * Returns the human-readable reason a route is unavailable, or null if available.
 */
export function routeUnavailableReason(path: string): string | null {
    if (isRouteAvailable(path)) return null;
    const config = findRouteConfig(path);
    return config?.unavailableReason ?? 'This feature is not available in the current deployment mode.';
}

function findRouteConfig(path: string): RouteConfig | undefined {
    // Exact match first
    const exact = ROUTE_AVAILABILITY.find(r => r.path === path);
    if (exact) return exact;
    // Prefix match for nested routes (/siem/search → /siem)
    return ROUTE_AVAILABILITY.find(r => path.startsWith(r.path + '/'));
}

// ── Service registration modes ─────────────────────────────────────────────

/**
 * Describes which backend capabilities are available in this context.
 * Used by UI components to decide whether to show/hide features.
 */
export interface ServiceCapabilities {
    /** Local PTY terminal — requires Wails + OS pty */
    localTerminal: boolean;
    /** OS keychain (Windows Credential Manager / macOS Keychain) */
    osKeychain: boolean;
    /** Direct SFTP to/from local filesystem */
    localSftp: boolean;
    /** Multi-user authentication (OIDC, SAML, MFA) */
    enterpriseAuth: boolean;
    /** Fleet agent management (mass push, registration) */
    agentFleet: boolean;
    /** Raft cluster coordination */
    clustering: boolean;
    /** Multi-analyst SOC collaboration */
    socCollaboration: boolean;
    /** Remote OBLIVRA server connected (hybrid mode) */
    remoteServer: boolean;
    remoteServerUrl: string | null;
}

export function getServiceCapabilities(): ServiceCapabilities {
    const remoteUrl = localStorage.getItem('oblivra:remote_server') ?? null;

    return {
        localTerminal:   IS_DESKTOP || IS_HYBRID,
        osKeychain:      IS_DESKTOP,
        localSftp:       IS_DESKTOP || IS_HYBRID,
        enterpriseAuth:  IS_BROWSER || IS_HYBRID,
        agentFleet:      IS_BROWSER || IS_HYBRID,
        clustering:      IS_BROWSER || IS_HYBRID,
        socCollaboration:IS_BROWSER || IS_HYBRID,
        remoteServer:    IS_HYBRID,
        remoteServerUrl: IS_HYBRID ? remoteUrl : null,
    };
}

// ── Hybrid mode helpers ────────────────────────────────────────────────────

/**
 * Configure hybrid mode by pointing the desktop binary at a remote server.
 * Call this from the Settings page when the operator enters a server URL.
 * Requires a page reload to take effect (context is detected at module load).
 */
export function configureHybridMode(serverUrl: string): void {
    if (!IS_DESKTOP) {
        console.warn('[context] configureHybridMode() called in non-desktop context, ignoring');
        return;
    }
    localStorage.setItem('oblivra:remote_server', serverUrl.trim());
    // Reload so APP_CONTEXT is re-detected
    window.location.reload();
}

/**
 * Disconnect from the remote server and return to pure desktop mode.
 */
export function disconnectHybridMode(): void {
    localStorage.removeItem('oblivra:remote_server');
    window.location.reload();
}

/**
 * Returns the remote server base URL for hybrid API calls, or null.
 */
export function getRemoteServerUrl(): string | null {
    if (!IS_HYBRID) return null;
    return localStorage.getItem('oblivra:remote_server');
}
