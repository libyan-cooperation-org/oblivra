import { createEffect, type JSX } from 'solid-js';
import { isAuthenticated } from './services/auth';
import { isDesktop } from './context';

export default function App(props: { children?: JSX.Element }) {
  createEffect(() => {
    // Phase 0.5: In Desktop (Wails) context, auth is managed by the native
    // app layer — skip the browser-based JWT guard to avoid redirect loops.
    if (isDesktop()) return;

    // Browser context: enforce JWT/API-key authentication.
    const path = window.location.pathname;
    const publicPaths = ['/login', '/onboarding'];
    if (!isAuthenticated() && !publicPaths.includes(path)) {
      window.location.href = '/login';
    }
  });

  return <>{props.children}</>;
}
