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
const AlertManagement = lazy(() => import('./pages/AlertManagement'));
const LookupManager   = lazy(() => import('./pages/LookupManager'));
const EscalationCenter    = lazy(() => import('./pages/EscalationCenter'));
const ThreatIntelDashboard = lazy(() => import('./pages/ThreatIntelDashboard'));
const EnrichmentViewer    = lazy(() => import('./pages/EnrichmentViewer'));
const MitreHeatmap        = lazy(() => import('./pages/MitreHeatmap'));
const PlaybookBuilder     = lazy(() => import('./pages/PlaybookBuilder'));
const UEBADashboard       = lazy(() => import('./pages/UEBADashboard'));
const NDRDashboard        = lazy(() => import('./pages/NDRDashboard'));
const RansomwareCenter    = lazy(() => import('./pages/RansomwareCenter'));
const RegulatorPortal     = lazy(() => import('./pages/RegulatorPortal'));
const EvidenceVault       = lazy(() => import('./pages/EvidenceVault'));
const PlaybookMetrics     = lazy(() => import('./pages/PlaybookMetrics'));
const PeerAnalytics       = lazy(() => import('./pages/PeerAnalytics'));
const FusionDashboard     = lazy(() => import('./pages/FusionDashboard'));

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
    <Route path="/"             component={guard('any', Dashboard)} />
    <Route path="/siem/search"  component={guard('any', SIEMSearch)} />
    <Route path="/alerts"       component={guard('any', AlertManagement)} />
    <Route path="/lookups"      component={guard('any', LookupManager)} />
    <Route path="/threatintel"      component={guard('any', ThreatIntelDashboard)} />
    <Route path="/enrich"           component={guard('any', EnrichmentViewer)} />
    <Route path="/mitre-heatmap"    component={guard('any', MitreHeatmap)} />

    {/* Web-only */}
    <Route path="/fleet"      component={guard('web', FleetManagement)} />
    <Route path="/identity"   component={guard('web', IdentityAdmin)} />
    <Route path="/escalation"       component={guard('web', EscalationCenter)} />
    <Route path="/playbooks"         component={guard('any', PlaybookBuilder)} />
    <Route path="/ueba"              component={guard('any', UEBADashboard)} />
    <Route path="/ndr"               component={guard('any', NDRDashboard)} />
    <Route path="/ransomware"        component={guard('any', RansomwareCenter)} />
    <Route path="/regulator"          component={guard('web', RegulatorPortal)} />
    <Route path="/evidence"           component={guard('any', EvidenceVault)} />
    <Route path="/playbook-metrics"   component={guard('any', PlaybookMetrics)} />
    <Route path="/peer-analytics"     component={guard('any', PeerAnalytics)} />
    <Route path="/fusion"             component={guard('any', FusionDashboard)} />
  </Router>
), root!);
