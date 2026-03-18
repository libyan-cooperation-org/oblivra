/* @refresh reload */
import { render } from 'solid-js/web';
import { HashRouter, Route, Navigate } from '@solidjs/router';
import App from './App';
import './styles/global.css';

import { Dashboard } from './components/dashboard/Dashboard';
import { ComplianceCenter } from './components/compliance/ComplianceCenter';
import { FleetDashboard } from './components/fleet/FleetDashboard';
import { SIEMPanel } from './components/siem/SIEMPanel';
import { AlertDashboard } from './components/siem/AlertDashboard';
import { TeamDashboard } from './components/team/TeamDashboard';
import { OpsCenter } from './pages/OpsCenter';
import { GlobalTopology } from './pages/GlobalTopology';
import { PluginManager } from './pages/PluginManager';
import { ConfigRisk } from './pages/ConfigRisk';
import { TerminalLayout } from './components/terminal/TerminalLayout';
import { EvidenceLocker } from './components/forensics/EvidenceLocker';
import { GovernanceDashboard } from './components/governance/GovernanceDashboard';
import { FeatureGovernance } from './components/governance/FeatureGovernance';
import { AgentConsole } from './components/fleet/AgentConsole';
import { CommandCenter } from './components/incident/CommandCenter';
import { RansomwareDashboard } from './components/security/RansomwareDashboard';
import SimulationPanel from './pages/SimulationPanel';
import { PurpleTeam } from './pages/PurpleTeam';
import { ResponseReplay } from './pages/ResponseReplay';
import { UEBAPanel } from './components/intelligence/UEBAPanel';
import { SelfMonitor } from './components/monitoring/SelfMonitor';
import { ThreatHunter } from './components/security/ThreatHunter';
import { ThreatGraph } from './components/intelligence/ThreatGraph';
import { NetworkMap } from './components/intelligence/NetworkMap';
import { RuntimeTrust } from './pages/RuntimeTrust';
import { CredentialIntel } from './pages/CredentialIntel';
import { TemporalIntegrity } from './pages/TemporalIntegrity';
import { DataDestruction } from './pages/DataDestruction';
import { EvidenceLedger } from './pages/EvidenceLedger';
import { UsersPanel } from './components/settings/UsersPanel';
import { ExecutiveDashboard } from './pages/ExecutiveDashboard';
import { PasswordVault } from './pages/PasswordVault';
import { WarMode } from './pages/WarMode';
import { SettingsManager } from './components/settings/SettingsManager';
import { SOCWorkspace } from './components/soc/SOCWorkspace';
import { LineageExplorer } from './pages/LineageExplorer';
import { DecisionInspector } from './pages/DecisionInspector';
import { OfflineUpdate } from './pages/OfflineUpdate';
import { OQLDashboard } from './pages/OQLDashboard';
import { AIAssistantPage } from './pages/AIAssistantPage';
import { TunnelsPage } from './pages/TunnelsPage';
import { RecordingsPage } from './pages/RecordingsPage';
import { SnippetsPage } from './pages/SnippetsPage';
import { NotesPage } from './pages/NotesPage';
import { SyncPage } from './pages/SyncPage';
import { MitreHeatmap } from './components/siem/MitreHeatmap';
import { EntityView } from './pages/EntityView';
import { FusionDashboard } from './pages/FusionDashboard';
import './styles/incident.css';
import { RouteGuard } from './core/RouteGuard';

const root = document.getElementById('root');
if (!root) throw new Error('Root element not found');

render(() => (
    <HashRouter root={App}>
        <Route path="/" component={() => <Navigate href="/dashboard" />} />
        <Route path="/dashboard" component={Dashboard} />
        <Route path="/compliance" component={ComplianceCenter} />
        <Route path="/hosts/*" component={FleetDashboard} />
        <Route path="/siem/*" component={SIEMPanel} />
        <Route path="/alerts" component={AlertDashboard} />
        <Route path="/team" component={TeamDashboard} />
        <Route path="/ops/*" component={OpsCenter} />
        <Route path="/topology" component={GlobalTopology} />
        <Route path="/plugins" component={PluginManager} />
        <Route path="/risk" component={ConfigRisk} />
        <Route path="/workspace" component={SettingsManager} />
        <Route path="/workspaces" component={SOCWorkspace} />
        <Route path="/terminal" component={() => <RouteGuard path="/terminal"><TerminalLayout /></RouteGuard>} />
        <Route path="/forensics" component={EvidenceLocker} />
        <Route path="/governance" component={GovernanceDashboard} />
        <Route path="/features" component={FeatureGovernance} />
        <Route path="/agents" component={() => <RouteGuard path="/agents"><AgentConsole /></RouteGuard>} />
        <Route path="/response" component={CommandCenter} />
        <Route path="/ransomware" component={RansomwareDashboard} />
        <Route path="/simulation" component={SimulationPanel} />
        <Route path="/purple-team" component={PurpleTeam} />
        <Route path="/response-replay" component={ResponseReplay} />
        <Route path="/ueba" component={UEBAPanel} />
        <Route path="/graph" component={ThreatGraph} />
        <Route path="/ndr" component={NetworkMap} />
        <Route path="/monitoring" component={SelfMonitor} />
        <Route path="/trust" component={RuntimeTrust} />
        <Route path="/credentials" component={CredentialIntel} />
        <Route path="/threat-hunter" component={ThreatHunter} />
        <Route path="/war-mode" component={WarMode} />
        <Route path="/temporal-integrity" component={TemporalIntegrity} />
        <Route path="/data-destruction" component={DataDestruction} />
        <Route path="/ledger" component={EvidenceLedger} />
        <Route path="/identity" component={UsersPanel} />
        <Route path="/executive" component={ExecutiveDashboard} />
        <Route path="/vault" component={PasswordVault} />
        <Route path="/soc" component={SOCWorkspace} />
        <Route path="/lineage" component={LineageExplorer} />
        <Route path="/decisions" component={DecisionInspector} />
        <Route path="/offline-update" component={OfflineUpdate} />
        <Route path="/analytics" component={OQLDashboard} />
        <Route path="/ai-assistant" component={AIAssistantPage} />
        <Route path="/tunnels" component={() => <RouteGuard path="/tunnels"><TunnelsPage /></RouteGuard>} />
        <Route path="/recordings" component={() => <RouteGuard path="/recordings"><RecordingsPage /></RouteGuard>} />
        <Route path="/snippets" component={() => <RouteGuard path="/snippets"><SnippetsPage /></RouteGuard>} />
        <Route path="/notes" component={() => <RouteGuard path="/notes"><NotesPage /></RouteGuard>} />
        <Route path="/sync" component={() => <RouteGuard path="/sync"><SyncPage /></RouteGuard>} />
        <Route path="/mitre-heatmap" component={MitreHeatmap} />
        <Route path="/fusion" component={FusionDashboard} />
        <Route path="/entity/:type/:id" component={EntityView} />
    </HashRouter>
), root);
