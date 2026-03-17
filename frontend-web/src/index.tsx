/* @refresh reload */
/// <reference types="vite/client" />
import { render } from 'solid-js/web';
import { Router, Route } from '@solidjs/router';
import { lazy, type Component } from 'solid-js';
import './styles/index.css';
import App from './App.tsx';
import ContextRoute from './components/ContextRoute.tsx';
import type { AppContext } from './context.ts';

// ── Pages (lazy) ──────────────────────────────────────────────────────────────
const Login           = lazy(() => import('./pages/Login'));
const Onboarding      = lazy(() => import('./pages/Onboarding'));
const Dashboard       = lazy(() => import('./core/Dashboard'));
// Phase 0.5 — New pages
const FleetManagement = lazy(() => import('./pages/FleetManagement'));
const SIEMSearch      = lazy(() => import('./pages/SIEMSearch'));
const IdentityAdmin   = lazy(() => import('./pages/IdentityAdmin'));

// ── ContextRoute wrapper factory ──────────────────────────────────────────────
// Returns a component that renders the target page only in the correct context.
function guard(ctx: AppContext | 'any', page: Component): Component {
  return () => <ContextRoute context={ctx} component={page} />;
}

// ── Entry point ───────────────────────────────────────────────────────────────
const root = document.getElementById('root');

if (import.meta.env.DEV && !(root instanceof HTMLElement)) {
  throw new Error(
    'Root element not found. Did you forget to add it to your index.html? Or is the id misspelled?',
  );
}

render(() => (
  <Router root={App}>
    {/* Public */}
    <Route path="/login"      component={Login} />
    <Route path="/onboarding" component={Onboarding} />

    {/* Hybrid (any context) */}
    <Route path="/"            component={guard('any', Dashboard)} />
    <Route path="/siem/search" component={guard('any', SIEMSearch)} />

    {/* Web-only */}
    <Route path="/fleet"    component={guard('web', FleetManagement)} />
    <Route path="/identity" component={guard('web', IdentityAdmin)} />
  </Router>
), root!);
