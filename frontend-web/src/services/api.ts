import { isDesktop } from '../context';

/**
 * BASE_URL resolves the API endpoint based on execution context (Phase 0.5).
 *
 * - Desktop (Wails): Wails injects a local HTTP proxy at localhost:8080. Using
 *   a relative path would fail because Wails serves the frontend via custom
 *   protocol, so we always point to the local backend.
 * - Browser: Rely on VITE_API_URL if configured (staging/prod) or fall back
 *   to the same origin so the web server can reverse-proxy /api/v1 correctly.
 */
function resolveBaseURL(): string {
  if (isDesktop()) {
    // Wails desktop: backend is always on localhost:8080
    return 'http://localhost:8080/api/v1';
  }
  // Browser: use env override or same-origin (reverse-proxied API)
  return import.meta.env.VITE_API_URL ?? '/api/v1';
}

const BASE_URL = resolveBaseURL();

export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('oblivra_token');
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { 'X-API-Key': token } : {}),
    ...Object.fromEntries(new Headers(options.headers || {}).entries()),
  };

  const response = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(error || response.statusText);
  }

  if (response.status === 204) return {} as T;

  return response.json();
}
