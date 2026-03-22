/* @refresh reload */
import { render } from 'solid-js/web';
import { HashRouter, Route, Navigate } from '@solidjs/router';
import { Show } from 'solid-js';
import App from './App';
import './styles/global.css';

// ── Context ───────────────────────────────────────────────────────────────────
import { RouteGuard } from './core/RouteGuard';
import { IS_DESKTOP, IS_HYBRID } from './core/context';

// ── Shared / both contexts ────────────────────────────────────────────────────
import { Dashboard }           from './components/dashboard/Dashboard';
import { ComplianceCenter }    from './components/compliance/ComplianceCenter';
import { FleetDashboard }      from './components/fleet/FleetDashboard';
import { SIEMPanel }           from './components/siem/SIEMPanel';
import { AlertDashboard }      from './components/siem/AlertDashboard';
import { AlertManagement }     from './components/siem/AlertManagement';
import { SIEMSearch }          from './components/siem/SIEMSearch';
import { MitreHeatmap }        from './components/siem/MitreHeatmap';
import { TeamDashboard }       from './components/team/TeamDashboard';
import { OpsCenter }           from './pages/OpsCenter';
import { GlobalTopology }      from './pages/GlobalTopology';
import { PluginManager }       from './pages/PluginManager';
import { ConfigRisk }          from './pages/ConfigRisk';
import { GovernanceDashboard } from './components/governance/GovernanceDashboard';
import { FeatureGovernance }   from './components/governance/FeatureGovernance';
import { CommandCenter }       from './components/incident/CommandCenter';
import { EscalationCenter }    from './components/incident/EscalationCenter';
import { PlaybookEngineUI }    from './components/incident/PlaybookEngineUI';
import SimulationPanel         from './pages/SimulationPanel';
import { PurpleTeam }          from './pages/PurpleTeam';
import { ResponseReplay }      from './pages/ResponseReplay';
import { ThreatGraph }         from './components/intelligence/ThreatGraph';
import { ThreatIntelDashboard }from './components/intelligence/ThreatIntelDashboard';
import { EnrichmentViewer }    from './components/intelligence/EnrichmentViewer';
import { SelfMonitor }         from './components/monitoring/SelfMonitor';
import { ThreatHunter }        from './components/security/ThreatHunter';
import { RuntimeTrust }        from './pages/RuntimeTrust';
import { CredentialIntel }     from './pages/CredentialIntel';
import { TemporalIntegrity }   from './pages/TemporalIntegrity';
import { DataDestruction }     from './pages/DataDestruction';
import { EvidenceLedger }      from './pages/EvidenceLedger';
import { ExecutiveDashboard }  from './pages/ExecutiveDashboard';
import { PasswordVault }       from './pages/PasswordVault';
import { WarMode }             from './pages/WarMode';
import { SettingsManager }     from './components/settings/SettingsManager';
import { SOCWorkspace }        from './components/soc/SOCWorkspace';
import { LineageExplorer }     from './pages/LineageExplorer';
import { DecisionInspector }   from './pages/DecisionInspector';
import { OQLDashboard }        from './pages/OQLDashboard';
import { AIAssistantPage }     from './pages/AIAssistantPage';
import { FusionDashboard }     from './pages/FusionDashboard';
import { EntityView }          from './pages/EntityView';
import { LicensePage }         from './pages/LicensePage';
import { UEBAOverview }        from './components/analytics/UEBAOverview';
import { NDROverview }         from './components/intelligence/NDROverview';

// ── Desktop-only ──────────────────────────────────────────────────────────────
import { TerminalLayout }      from './components/terminal/TerminalLayout';
import { TunnelsPage }         from './pages/TunnelsPage';
import { RecordingsPage }      from './pages/RecordingsPage';
import { SnippetsPage }        from './pages/SnippetsPage';
import { NotesPage }           from './pages/NotesPage';
import { SyncPage }            from './pages/SyncPage';
import { OfflineUpdate }       from './pages/OfflineUpdate';

// ── Desktop primary / Browser alternate ──────────────────────────────────────
// These features exist in both contexts but with different UI components.
// One URL → context-switched rendering. No duplicate routes.
import { UEBAPanel }           from './components/intelligence/UEBAPanel';
import { NetworkMap }          from './components/intelligence/NetworkMap';
import { EvidenceLocker }      from './components/forensics/EvidenceLocker';
import { RemoteForensics }     from './components/forensics/RemoteForensics';
import { RansomwareDashboard } from './components/security/RansomwareDashboard';
import { RansomwareUI }        from './components/security/RansomwareUI';

// ── Browser-only ──────────────────────────────────────────────────────────────
import { AgentConsole }        from './components/fleet/AgentConsole';
import { FleetManagement }     from './components/fleet/FleetManagement';
import { UsersPanel }          from './components/settings/UsersPanel';
import { IdentityAdmin }       from './components/auth/IdentityAdmin';

import './styles/incident.css';

const root = document.getElementById('root');
if (!root) throw new Error('Root element not found');

// ── Context-switched route wrapper ────────────────────────────────────────────
// One URL renders different components for desktop vs browser. Hybrid shows desktop variant.
const DesktopOrBrowser = (deskComp: any, browserComp: any) => () => (
    <Show when={IS_DESKTOP || IS_HYBRID} fallback={browserComp}>
        {deskComp}
    </Show>
);

render(() => (
    <HashRouter root={App}>
        <Route path="/" component={() => <Navigate href="/dashboard" />} />

        {/* ── Available in both contexts ────────────────────────────────── */}
        <Route path="/dashboard"          component={Dashboard} />
        <Route path="/compliance"         component={ComplianceCenter} />
        <Route path="/hosts/*"            component={FleetDashboard} />
        <Route path="/siem/*"             component={SIEMPanel} />
        <Route path="/alerts"             component={AlertDashboard} />
        <Route path="/alert-management"   component={AlertManagement} />
        <Route path="/siem-search"        component={SIEMSearch} />
        <Route path="/mitre-heatmap"      component={MitreHeatmap} />
        <Route path="/team"               component={TeamDashboard} />
        <Route path="/ops/*"              component={OpsCenter} />
        <Route path="/topology"           component={GlobalTopology} />
        <Route path="/plugins"            component={PluginManager} />
        <Route path="/risk"               component={ConfigRisk} />
        <Route path="/workspace"          component={SettingsManager} />
        <Route path="/workspaces"         component={SOCWorkspace} />
        <Route path="/governance"         component={GovernanceDashboard} />
        <Route path="/features"           component={FeatureGovernance} />
        <Route path="/response"           component={CommandCenter} />
        <Route path="/escalation"         component={EscalationCenter} />
        <Route path="/playbook-builder"   component={PlaybookEngineUI} />
        <Route path="/simulation"         component={SimulationPanel} />
        <Route path="/purple-team"        component={PurpleTeam} />
        <Route path="/response-replay"    component={ResponseReplay} />
        <Route path="/graph"              component={ThreatGraph} />
        <Route path="/threat-intel"       component={ThreatIntelDashboard} />
        <Route path="/enrichment"         component={EnrichmentViewer} />
        <Route path="/monitoring"         component={SelfMonitor} />
        <Route path="/trust"              component={RuntimeTrust} />
        <Route path="/credentials"        component={CredentialIntel} />
        <Route path="/threat-hunter"      component={ThreatHunter} />
        <Route path="/war-mode"           component={WarMode} />
        <Route path="/temporal-integrity" component={TemporalIntegrity} />
        <Route path="/data-destruction"   component={DataDestruction} />
        <Route path="/ledger"             component={EvidenceLedger} />
        <Route path="/executive"          component={ExecutiveDashboard} />
        <Route path="/vault"              component={PasswordVault} />
        <Route path="/soc"                component={SOCWorkspace} />
        <Route path="/lineage"            component={LineageExplorer} />
        <Route path="/decisions"          component={DecisionInspector} />
        <Route path="/analytics"          component={OQLDashboard} />
        <Route path="/fusion"             component={FusionDashboard} />
        <Route path="/ai-assistant"       component={AIAssistantPage} />
        <Route path="/entity/:type/:id"   component={EntityView} />
        <Route path="/license"            component={LicensePage} />

        {/* ── Context-switched: one URL, desktop vs browser component ──── */}
        {/*  /ueba      → UEBAPanel (desktop)   | UEBAOverview (browser)   */}
        {/*  /ndr       → NetworkMap (desktop)  | NDROverview (browser)    */}
        {/*  /forensics → EvidenceLocker (desk) | RemoteForensics (browser)*/}
        {/*  /ransomware→ Dashboard (desktop)   | RansomwareUI (browser)   */}
        <Route path="/ueba"       component={DesktopOrBrowser(<UEBAPanel />,           <UEBAOverview />)} />
        <Route path="/ndr"        component={DesktopOrBrowser(<NetworkMap />,           <NDROverview />)} />
        <Route path="/forensics"  component={DesktopOrBrowser(<EvidenceLocker />,       <RemoteForensics />)} />
        <Route path="/ransomware" component={DesktopOrBrowser(<RansomwareDashboard />,  <RansomwareUI />)} />

        {/* Aliases kept for deep-linking / legacy bookmarks */}
        <Route path="/ueba-overview"       component={UEBAOverview} />
        <Route path="/ndr-overview"        component={NDROverview} />
        <Route path="/remote-forensics"    component={RemoteForensics} />
        <Route path="/ransomware-ui"       component={RansomwareUI} />
        <Route path="/threat-intel-dashboard" component={ThreatIntelDashboard} />
        <Route path="/fleet-management"    component={() =>
            <RouteGuard path="/fleet-management"><FleetManagement /></RouteGuard>} />
        <Route path="/identity-admin"      component={() =>
            <RouteGuard path="/identity-admin"><IdentityAdmin /></RouteGuard>} />

        {/* ── Desktop-only ─────────────────────────────────────────────── */}
        <Route path="/terminal"      component={() => <RouteGuard path="/terminal"><TerminalLayout /></RouteGuard>} />
        <Route path="/tunnels"       component={() => <RouteGuard path="/tunnels"><TunnelsPage /></RouteGuard>} />
        <Route path="/recordings"    component={() => <RouteGuard path="/recordings"><RecordingsPage /></RouteGuard>} />
        <Route path="/snippets"      component={() => <RouteGuard path="/snippets"><SnippetsPage /></RouteGuard>} />
        <Route path="/notes"         component={() => <RouteGuard path="/notes"><NotesPage /></RouteGuard>} />
        <Route path="/sync"          component={() => <RouteGuard path="/sync"><SyncPage /></RouteGuard>} />
        <Route path="/offline-update"component={() => <RouteGuard path="/offline-update"><OfflineUpdate /></RouteGuard>} />

        {/* ── Browser-only ─────────────────────────────────────────────── */}
        <Route path="/agents"     component={() => <RouteGuard path="/agents"><AgentConsole /></RouteGuard>} />
        <Route path="/identity"   component={() => <RouteGuard path="/identity"><UsersPanel /></RouteGuard>} />
    </HashRouter>
), root);
