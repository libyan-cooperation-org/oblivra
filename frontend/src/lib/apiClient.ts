/**
 * apiClient.ts — tenant-aware fetch wrapper.
 *
 * Phase 30.4d follow-up: every outbound HTTP request from the SPA
 * needs to carry the operator's currently-selected tenant context so
 * the backend can scope its query / mutation to the right tenant.
 *
 * The selected tenant lives in `appStore.currentTenantId`. This
 * wrapper reads it at call time (not capture time, so a tenant
 * switch immediately affects subsequent requests) and attaches it
 * as the `X-Tenant-Id` header.
 *
 * Why a separate file rather than extending `bridge.ts`:
 *   - bridge.ts is the Wails IPC layer; this is the REST/HTTP layer.
 *     Different transport, different header concerns.
 *   - Plain .ts file (no runes) — we just READ from `appStore` which
 *     is fine. See Phase 29 for why writing $state in a plain .ts
 *     file is forbidden.
 *
 * Usage:
 *   import { apiFetch } from '@lib/apiClient';
 *   const res = await apiFetch('/api/v1/alerts');
 *
 * Migration: existing call-sites use `fetch(...)` directly. Migrate
 * them as needed — the wrapper accepts the same arguments as the
 * native fetch API.
 */
import { appStore } from './stores/app.svelte';
import { IS_DESKTOP, IS_HYBRID } from './context';

/**
 * Desktop-mode REST base URL.
 *
 * In Wails desktop mode the frontend is served by the embedded asset
 * handler at `http://wails.localhost/...`, but the REST API listens
 * on a separate port (default 8080, set in
 * `internal/services/api_service.go::NewAPIService`). The asset
 * handler does NOT proxy `/api/*` to the REST mux — relative-URL
 * fetches just 404 against the asset FS.
 *
 * To keep relative-URL `apiFetch('/api/v1/...')` calls working from
 * either context, we rewrite the URL in desktop mode to point at
 * `http://127.0.0.1:8080` (the REST listener). Browser mode is
 * unaffected because the page is already served from the same origin
 * as the REST API.
 *
 * Hybrid mode follows the operator's configured remote-server URL
 * stored in `oblivra:remote_server` localStorage (see context.ts).
 */
function getApiBase(): string {
  if (IS_DESKTOP) return 'http://127.0.0.1:8080';
  if (IS_HYBRID) {
    try {
      const remote = localStorage.getItem('oblivra:remote_server');
      if (remote && remote.trim() !== '') return remote.trim().replace(/\/$/, '');
    } catch { /* private mode */ }
  }
  return ''; // browser — same-origin
}

/**
 * Resolve `/api/...`-prefixed URLs to the REST base. Absolute URLs
 * (http(s)://...) and non-API relative URLs pass through unchanged.
 */
function resolveURL(input: RequestInfo | URL): RequestInfo | URL {
  if (typeof input !== 'string') return input;
  if (!input.startsWith('/api/')) return input;
  const base = getApiBase();
  if (!base) return input;
  return base + input;
}

/**
 * apiFetch — drop-in replacement for window.fetch that:
 *   1. retargets `/api/*` URLs to the REST listener in desktop / hybrid mode
 *   2. attaches the operator's currently-selected `X-Tenant-Id`
 *   3. auto-handles 401/403 by bouncing to /login (audit fix High-7)
 *
 * The tenant header name matches the Go REST middleware in
 * `internal/api/rest.go` — the existing resolver reads from this
 * header before falling back to URL or session derivation.
 */
export function apiFetch(
  input: RequestInfo | URL,
  init: RequestInit = {},
): Promise<Response> {
  const tenantId = appStore.currentTenantId;

  // Don't mutate the caller's headers — clone first.
  const headers = new Headers(init.headers ?? undefined);
  if (tenantId) {
    headers.set('X-Tenant-Id', tenantId);
  }

  // Default to credentials:'include' so the browser sends the auth
  // cookie. Callers can override by passing `credentials` explicitly.
  const credentials = init.credentials ?? 'include';

  const target = resolveURL(input);

  return fetch(target, { ...init, headers, credentials }).then((res) => {
    // Audit fix High-7: auto-handle 401/403 by clearing local
    // session and bouncing to /login. Without this, an expired
    // session leaves the UI showing stale data + getting silent
    // 401s on every action.
    if (res.status === 401 || res.status === 403) {
      handleAuthFailure(res.status, urlAsString(input));
    }
    return res;
  });
}

/** Cast RequestInfo to a string label for logging. */
function urlAsString(input: RequestInfo | URL): string {
  if (typeof input === 'string') return input;
  if (input instanceof URL) return input.href;
  return (input as Request).url ?? '<unknown>';
}

// Single-flight guard so a burst of 401s during a refresh storm
// only fires one redirect.
let authFailureFiring = false;

/**
 * handleAuthFailure runs when the server rejects the session as
 * unauthenticated/unauthorised. We:
 *   - clear local user metadata so isAuthenticated() returns false
 *   - dispatch a global window event that the App-level layout
 *     listens for and pushes the user to /login (Svelte router-level
 *     redirects can't happen from outside a component context)
 *
 * Auth endpoints (/api/v1/auth/*) are exempt — login itself returning
 * 401 is the normal "wrong password" path, not a session expiry.
 */
function handleAuthFailure(status: number, url: string) {
  if (url.includes('/api/v1/auth/')) return; // wrong-password path, not expiry
  if (authFailureFiring) return;
  authFailureFiring = true;
  // Reset the guard after a tick so genuine new failures still fire.
  setTimeout(() => { authFailureFiring = false; }, 1000);

  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem('oblivra_user');
    }
  } catch { /* private mode */ }

  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('oblivra:auth-failure', {
      detail: { status, url },
    }));
  }
}

/**
 * apiError represents a failed API call with context the caller
 * needs to react. The `featurePending` flag makes the audit's
 * Critical-4 stub-honesty fix surface cleanly: a UI button can
 * disable itself or show "coming soon" instead of pretending the
 * action succeeded.
 */
export class APIError extends Error {
  status: number;
  code?: string;
  featurePending: boolean;
  constructor(opts: { status: number; message: string; code?: string; featurePending?: boolean }) {
    super(opts.message);
    this.name = 'APIError';
    this.status = opts.status;
    this.code = opts.code;
    this.featurePending = opts.featurePending ?? false;
  }
}

/**
 * extractError parses the new `{ok, status, error, code}` envelope
 * shape from the server (see internal/api/error_envelope.go) and
 * surfaces a typed APIError with the public-safe message.
 */
async function extractError(res: Response, defaultPrefix: string): Promise<APIError> {
  const featurePending = res.headers.get('X-Feature-Status') === 'pending';
  let message = `${defaultPrefix} failed: ${res.status} ${res.statusText}`;
  let code: string | undefined;
  try {
    const body = await res.clone().json() as { error?: string; code?: string };
    if (body.error) message = body.error;
    if (body.code) code = body.code;
  } catch {
    // body wasn't JSON envelope — keep the default message
  }
  return new APIError({ status: res.status, message, code, featurePending });
}

/** Convenience helper: GET + JSON parse. Throws APIError on non-2xx. */
export async function apiGetJSON<T = unknown>(url: string, init?: RequestInit): Promise<T> {
  const res = await apiFetch(url, { ...init, method: 'GET' });
  if (!res.ok) throw await extractError(res, `GET ${url}`);
  return res.json() as Promise<T>;
}

/** Convenience helper: POST JSON body + JSON parse response. */
export async function apiPostJSON<T = unknown>(
  url: string,
  body: unknown,
  init?: RequestInit,
): Promise<T> {
  const headers = new Headers(init?.headers ?? undefined);
  if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  const res = await apiFetch(url, {
    ...init,
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  if (!res.ok) throw await extractError(res, `POST ${url}`);
  return res.json() as Promise<T>;
}
