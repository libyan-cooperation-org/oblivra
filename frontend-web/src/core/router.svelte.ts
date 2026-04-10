/**
 * OBLIVRA Web — Minimal client-side router (Svelte 5 runes)
 * Replaces @solidjs/router with a lightweight hash/history router.
 */

class Router {
  path = $state(window.location.pathname || '/');

  constructor() {
    window.addEventListener('popstate', () => {
      this.path = window.location.pathname || '/';
    });
  }

  push(path: string) {
    window.history.pushState(null, '', path);
    this.path = path;
  }

  replace(path: string) {
    window.history.replaceState(null, '', path);
    this.path = path;
  }
}

export const router = new Router();
export const push    = (path: string) => router.push(path);
export const replace = (path: string) => router.replace(path);
