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

/**
 * apiFetch — drop-in replacement for window.fetch that attaches
 * X-Tenant-Id automatically when a tenant is selected. Falls back
 * to a plain fetch when no tenant scope is set ("All Tenants" admin
 * view).
 *
 * The header name `X-Tenant-Id` matches the convention used by the
 * Go REST middleware in `internal/api/rest.go` — the existing
 * tenant resolver reads from this header before falling back to
 * URL or session derivation.
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

  return fetch(input, { ...init, headers, credentials });
}

/** Convenience helper: GET + JSON parse. Throws on non-2xx. */
export async function apiGetJSON<T = unknown>(url: string, init?: RequestInit): Promise<T> {
  const res = await apiFetch(url, { ...init, method: 'GET' });
  if (!res.ok) {
    throw new Error(`GET ${url} failed: ${res.status} ${res.statusText}`);
  }
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
  if (!res.ok) {
    throw new Error(`POST ${url} failed: ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}
