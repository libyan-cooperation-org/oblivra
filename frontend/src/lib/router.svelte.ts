/**
 * OBLIVRA — Minimal Hash Router (Svelte 5 runes)
 *
 * Zero-dependency hash-based router designed for Wails desktop apps.
 * Supports:
 *   - Hash-based routing (#/path)
 *   - Path parameters (#/entity/:type/:id)
 *   - Wildcard routes (#/siem/*)
 *   - Programmatic navigation via push()
 *   - Reactive current route state
 */

export type RouteParams = Record<string, string>;

export interface RouteMatch {
    path: string;
    params: RouteParams;
}

/** Reactive current route */
let _currentPath = $state(getHashPath());
let _currentParams = $state<RouteParams>({});

function getHashPath(): string {
    const hash = window.location.hash;
    if (!hash || hash === '#') return '/';
    return hash.startsWith('#/') ? hash.slice(1) : hash.slice(1);
}

// Listen for hash changes
if (typeof window !== 'undefined') {
    window.addEventListener('hashchange', () => {
        _currentPath = getHashPath();
    });
}

export function getCurrentPath(): string {
    return _currentPath;
}

export function getCurrentParams(): RouteParams {
    return _currentParams;
}

/**
 * Navigate programmatically.
 */
export function push(path: string): void {
    window.location.hash = '#' + path;
}

export function replace(path: string): void {
    window.location.replace('#' + path);
}

/**
 * Match a route pattern against the current path.
 * Supports:
 *   - Exact: '/dashboard'
 *   - Params: '/entity/:type/:id' → { type: 'host', id: '123' }
 *   - Wildcard: '/siem/*' → matches /siem/anything
 */
export function matchRoute(pattern: string, path: string): RouteMatch | null {
    // Exact match
    if (pattern === path) {
        return { path, params: {} };
    }

    // Wildcard
    if (pattern.endsWith('/*')) {
        const base = pattern.slice(0, -2);
        if (path === base || path.startsWith(base + '/')) {
            return { path, params: { '*': path.slice(base.length + 1) } };
        }
        return null;
    }

    // Parameter matching
    const patternParts = pattern.split('/');
    const pathParts = path.split('/');

    if (patternParts.length !== pathParts.length) return null;

    const params: RouteParams = {};
    for (let i = 0; i < patternParts.length; i++) {
        if (patternParts[i].startsWith(':')) {
            params[patternParts[i].slice(1)] = pathParts[i];
        } else if (patternParts[i] !== pathParts[i]) {
            return null;
        }
    }

    return { path, params };
}
