<script lang="ts">
  console.log("App.svelte <script> is executing!");
  import { onMount } from 'svelte';
  import RouterView from '@components/RouterView.svelte';
  import { initBridge, APP_CONTEXT } from '@lib/bridge';
  import { appStore } from '@lib/stores/app.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { crisisStore } from '@lib/stores/crisis.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
import Sidebar from '@components/layout/CommandRail.svelte';
  import TopBar from '@components/layout/TitleBar.svelte';
  import CommandPalette from '@components/ui/CommandPalette.svelte';
  import ToastContainer from '@components/layout/ToastContainer.svelte';
  import BottomNav from '@components/layout/BottomNav.svelte';
  import LoadingScreen from '@components/ui/LoadingScreen.svelte';
  import ErrorScreen from '@components/ui/ErrorScreen.svelte';
  import { siemStore } from '@lib/stores/siem.svelte';

  // ── Pages
  import Dashboard from '@pages/Dashboard.svelte';
  import TerminalPage from '@pages/TerminalPage.svelte';
  import RecordingsPage from '@pages/RecordingsPage.svelte';
  import SIEMPanel from '@pages/SIEMPanel.svelte';
  import AlertDashboard from '@pages/AlertDashboard.svelte';
  import AlertManagement from '@pages/AlertManagement.svelte';
  import OfflineUpdate from '@pages/OfflineUpdate.svelte';
  import PlaybookBuilder from '@pages/PlaybookBuilder.svelte';
  import TasksPage from '@pages/TasksPage.svelte';
  import FeaturesPage from '@pages/FeaturesPage.svelte';
  import LicensePage from '@pages/LicensePage.svelte';
  import SimulationPanel from '@pages/SimulationPanel.svelte';
  import ThreatHunter from '@pages/ThreatHunter.svelte';
  import ThreatIntelPanel from '@pages/ThreatIntelPanel.svelte';
  import CredentialIntel from '@pages/CredentialIntel.svelte';
  import OpsCenter from '@pages/OpsCenter.svelte';
  import TunnelsPage from '@pages/TunnelsPage.svelte';
  import SSHBookmarks from '@pages/SSHBookmarks.svelte';
  import SIEMSearch from '@pages/SIEMSearch.svelte';
  import CompliancePage from '@pages/CompliancePage.svelte';
  import IdentityAdmin from '@pages/IdentityAdmin.svelte';
  import EscalationCenter from '@pages/EscalationCenter.svelte';
  import PurpleTeam from '@pages/PurpleTeam.svelte';
  import ExecutiveDash from '@pages/ExecutiveDash.svelte';
  import ResponseReplay from '@pages/ResponseReplay.svelte';
  import TemporalIntegrity from '@pages/TemporalIntegrity.svelte';
  import TopologyPage from '@pages/TopologyPage.svelte';
  import NetworkMap from '@pages/NetworkMap.svelte';
  import ThreatGraph from '@pages/ThreatGraph.svelte';
  import ThreatMap from '@pages/ThreatMap.svelte';
  import GlobalTopology from '@pages/GlobalTopology.svelte';
  import MitreHeatmap from '@pages/MitreHeatmap.svelte';
  import SessionPlayback from '@pages/SessionPlayback.svelte';
  import TerminalForensics from '@pages/TerminalForensics.svelte';
  import SnippetsPage from '@pages/SnippetsPage.svelte';
  import NotesPage from '@pages/NotesPage.svelte';
  import FleetDashboard from '@pages/FleetDashboard.svelte';
  import UEBAOverview from '@pages/UEBAOverview.svelte';
  import NDROverview from '@pages/NDROverview.svelte';
  import EnrichmentViewer from '@pages/EnrichmentViewer.svelte';
  import AgentConsole from '@pages/AgentConsole.svelte';
  import IncidentResponse from '@pages/IncidentResponse.svelte';
  import RuntimeTrust from '@pages/RuntimeTrust.svelte';
  import ForensicsPage from '@pages/ForensicsPage.svelte';
  import RansomwareUI from '@pages/RansomwareUI.svelte';
  import DataDestruction from '@pages/DataDestruction.svelte';
  import LineageExplorer from '@pages/LineageExplorer.svelte';
  import DecisionInspector from '@pages/DecisionInspector.svelte';
  import OQLDashboard from '@pages/OQLDashboard.svelte';
  import SOARPanel from '@pages/SOARPanel.svelte';
  import EvidenceLedger from '@pages/EvidenceLedger.svelte';
  import ChainOfCustody from '@pages/ChainOfCustody.svelte';
  import FusionDashboard from '@pages/FusionDashboard.svelte';
  import Settings from '@pages/Settings.svelte';
  import PluginManager from '@pages/PluginManager.svelte';
  import TeamDashboard from '@pages/TeamDashboard.svelte';
  import SyncPage from '@pages/SyncPage.svelte';
  import ConfigRisk from '@pages/ConfigRisk.svelte';
  import EntityView from '@pages/EntityView.svelte';
  import AIAssistantPage from '@pages/AIAssistantPage.svelte';
  import WarMode from '@pages/WarMode.svelte';
  import CaseManagement from '@pages/CaseManagement.svelte';
  import FleetMap from '@pages/FleetMap.svelte';
  import InvestigationDashboard from '@pages/InvestigationDashboard.svelte';
  import IncidentTimeline from '@pages/IncidentTimeline.svelte';
  import SecretManager from '@pages/SecretManager.svelte';
  import SuppressionManager from '@pages/SuppressionManager.svelte';
  import DevelopmentPage from '@pages/DevelopmentPage.svelte';
  import SetupWizard from '@pages/SetupWizard.svelte';
  import EvidenceVault from '@pages/EvidenceVault.svelte';
  import OperatorMode from '@pages/OperatorMode.svelte';
  import MultiTenantAdmin from '@pages/MultiTenantAdmin.svelte';
  import KeyboardMap from '@pages/KeyboardMap.svelte';
  import DashboardStudio from '@pages/DashboardStudio.svelte';

  // ── Types
  interface RouteDefinition {
    path: string;
    component: any;
  }

  let ready = $state(false);
  let error = $state<string | null>(null);
  let mainEl = $state<HTMLElement | null>(null);
  let screenWidth = $state(typeof window !== 'undefined' ? window.innerWidth : 1200);

  // Pop-out mode: when WindowService.PopOut spawns a new window with
  // ?popout=1&route=<r>, that window renders ONLY the requested route
  // without the sidebar / global chrome. The operator gets a clean
  // single-panel view they can drag onto a second monitor.
  const params = typeof window !== 'undefined' ? new URLSearchParams(window.location.search) : null;
  const popoutMode = params?.get('popout') === '1';
  const popoutRoute = params?.get('route') ?? '/';

  const isMobile = $derived(screenWidth < 640);

  function handleResize() {
    screenWidth = window.innerWidth;
  }

  $effect(() => {
    if (!mainEl) return;
    if (crisisStore.active) {
      mainEl.classList.add('crisis-mode-active');
    } else {
      mainEl.classList.remove('crisis-mode-active');
    }
  });

  const routes: RouteDefinition[] = [
    // Root & General
    { path: '/',            component: Dashboard },
    { path: '/dashboard',   component: Dashboard },
    { path: '/monitoring',  component: Dashboard },
    { path: '/analytics',   component: ExecutiveDash },
    { path: '/executive',   component: ExecutiveDash },

    // SIEM & Intelligence
    { path: '/siem',                   component: SIEMPanel },
    { path: '/siem-search',            component: SIEMSearch },
    { path: '/alerts',                 component: AlertDashboard },
    { path: '/alert-management',       component: AlertManagement },
    { path: '/threat-intel',           component: ThreatIntelPanel },
    { path: '/threat-intel-dashboard', component: ThreatIntelPanel },
    { path: '/threat-hunter',          component: ThreatHunter },
    { path: '/threat-graph',           component: ThreatGraph },
    { path: '/graph',                  component: ThreatGraph },   // CommandRail 'graph' → here
    { path: '/threat-map',             component: ThreatMap },
    { path: '/ueba',                   component: UEBAOverview },
    { path: '/ueba-overview',          component: UEBAOverview },
    { path: '/ndr',                    component: NDROverview },
    { path: '/ndr-overview',           component: NDROverview },
    { path: '/enrichment',             component: EnrichmentViewer },
    { path: '/credentials',            component: CredentialIntel },

    // Operations & Terminal
    { path: '/ops',              component: OpsCenter },
    { path: '/terminal',         component: TerminalPage },
    { path: '/ssh',              component: SSHBookmarks },
    { path: '/tunnels',          component: TunnelsPage },
    { path: '/recordings',       component: RecordingsPage },
    { path: '/session-playback', component: SessionPlayback },
    { path: '/tasks',            component: TasksPage },
    { path: '/snippets',         component: SnippetsPage },
    { path: '/notes',            component: NotesPage },
    { path: '/agent-console',    component: AgentConsole },
    { path: '/operator',         component: OperatorMode },

    // Fleet & Workspace
    { path: '/investigation',   component: InvestigationDashboard },
    { path: '/fleet',            component: FleetDashboard },
    { path: '/fleet-management', component: FleetDashboard },
    { path: '/hosts',            component: FleetDashboard },
    { path: '/soc',              component: FleetDashboard },
    { path: '/agents',           component: FleetDashboard },
    { path: '/fleet-map',        component: FleetMap },
    { path: '/workspace',        component: Dashboard },
    { path: '/fusion',           component: FusionDashboard },

    // Security & Incident Response
    { path: '/response',         component: IncidentResponse },
    { path: '/timeline',         component: IncidentTimeline },
    { path: '/timeline/:principalID/:principalType/:targetTime', component: IncidentTimeline },
    { path: '/cases',            component: CaseManagement },  // was missing
    { path: '/escalation',       component: EscalationCenter },
    { path: '/playbook-builder', component: PlaybookBuilder },
    { path: '/purple-team',      component: PurpleTeam },
    { path: '/war-mode',         component: WarMode },
    { path: '/data-destruction', component: DataDestruction },
    { path: '/ransomware',       component: RansomwareUI },
    { path: '/ransomware-ui',    component: RansomwareUI },
    { path: '/simulation',       component: SimulationPanel },

    // Forensics & Audit
    { path: '/forensics',          component: ForensicsPage },
    { path: '/remote-forensics',   component: ForensicsPage },
    { path: '/terminal-forensics', component: TerminalForensics },
    { path: '/lineage',            component: LineageExplorer },
    { path: '/decisions',          component: DecisionInspector },
    { path: '/oql',                component: OQLDashboard },
    { path: '/dashboard-studio',   component: DashboardStudio },
    { path: '/evidence',           component: EvidenceLedger },
    { path: '/ledger',             component: EvidenceLedger },
    { path: '/chain-of-custody',   component: ChainOfCustody },
    { path: '/soar',               component: SOARPanel },
    { path: '/temporal-integrity', component: TemporalIntegrity },
    { path: '/response-replay',    component: ResponseReplay },

    // Topology
    { path: '/topology',        component: TopologyPage },
    { path: '/network-map',     component: NetworkMap },
    { path: '/global-topology', component: GlobalTopology },
    { path: '/mitre-heatmap',   component: MitreHeatmap },

    // Governance, Trust & Identity
    { path: '/compliance',     component: CompliancePage },
    { path: '/governance',     component: CompliancePage },
    { path: '/vault',          component: EvidenceVault },
    { path: '/evidence-vault', component: EvidenceVault },
    { path: '/setup',          component: SetupWizard },
    { path: '/setup-wizard',   component: SetupWizard },
    { path: '/trust',          component: RuntimeTrust },
    { path: '/runtime-trust',  component: RuntimeTrust },
    { path: '/identity-admin', component: IdentityAdmin },
    { path: '/admin',          component: MultiTenantAdmin },
    { path: '/shortcuts',      component: KeyboardMap },
    { path: '/secrets',        component: SecretManager },
    { path: '/suppression',    component: SuppressionManager },
    { path: '/secret-manager', component: SecretManager },

    // Management
    { path: '/settings',       component: Settings },
    { path: '/plugins',        component: PluginManager },
    { path: '/team',           component: TeamDashboard },
    { path: '/sync',           component: SyncPage },
    { path: '/offline-update', component: OfflineUpdate },
    { path: '/license',        component: LicensePage },
    { path: '/features',       component: FeaturesPage },
    { path: '/risk',           component: ConfigRisk },
    { path: '/entity',         component: EntityView },
    { path: '/ai-assistant',   component: AIAssistantPage },

    // Fallback
    { path: '*', component: DevelopmentPage },
  ];

  onMount(async () => {
    try {
      await initBridge();

      const rt = (window as any).runtime;
      if (rt && APP_CONTEXT !== 'browser') {
        console.debug('[App] Diagnostics initialized:', diagnosticsStore.healthGrade);
        rt.EventsOn('system.error', (msg: string) => {
          toastStore.add({ type: 'error', title: 'System Error', message: msg });
        });
        rt.EventsOn('system.toast', (toast: any) => {
          toastStore.add(toast);
        });
      }

      await appStore.init();
      crisisStore.init();
      siemStore.refreshStats();

      // In pop-out mode the spawned window starts at "/?popout=1&route=X"
      // — navigate to the requested route now that the router store is up.
      if (popoutMode && popoutRoute) {
        appStore.navigate(popoutRoute);
      }

      ready = true;
    } catch (e: any) {
      console.error('App init failed:', e);
      error = e.message || 'Failed to initialize OBLIVRA core.';
    }
  });

  function onKeyDown(e: KeyboardEvent) {
    // ── Command Palette (⌘K)
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      appStore.toggleCommandPalette();
    }

    // ── Global Search (⌘P)
    if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
      e.preventDefault();
      appStore.showCommandPalette = true;
    }

    // ── Help / Shortcuts (⌘/)
    if ((e.ctrlKey || e.metaKey) && e.key === '/') {
      e.preventDefault();
      appStore.navigate('/shortcuts');
    }

    // ── Escape to clear
    if (e.key === 'Escape') {
      appStore.showCommandPalette = false;
    }

    // ── Global Navigation (G + key)
    if (e.key.toLowerCase() === 'g') {
        const nextKeyHandler = (ne: KeyboardEvent) => {
            const key = ne.key.toLowerCase();
            if (key === 'd') appStore.navigate('/dashboard');
            if (key === 'a') appStore.navigate('/alerts');
            if (key === 's') appStore.navigate('/siem');
            if (key === 'f') appStore.navigate('/fleet');
            if (key === 'v') appStore.navigate('/evidence');
            window.removeEventListener('keydown', nextKeyHandler);
        };
        window.addEventListener('keydown', nextKeyHandler, { once: true });
    }

    // ── Operator Shortcuts (⌃⇧ + key)
    if (e.ctrlKey && e.shiftKey) {
        // Ctrl+Shift+I — Host isolation. Dispatches a window event so the
        // OperatorMode page can pick it up and call agentStore.toggleQuarantine
        // for the currently-active host. If the operator isn't on /operator
        // yet, navigate there so they have a host context to isolate.
        if (e.key === 'I' || e.key === 'i') {
            e.preventDefault();
            const onOperator = window.location.pathname.startsWith('/operator');
            if (!onOperator) {
                appStore.notify('Open Operator Mode and select a host before isolating', 'warning');
                appStore.navigate('/operator');
                return;
            }
            window.dispatchEvent(new CustomEvent('oblivra:isolate-host'));
        }
        // Ctrl+Shift+E — Evidence capture (same pattern).
        if (e.key === 'E' || e.key === 'e') {
            e.preventDefault();
            const onOperator = window.location.pathname.startsWith('/operator');
            if (!onOperator) {
                appStore.notify('Open Operator Mode and select a host before capturing evidence', 'warning');
                appStore.navigate('/operator');
                return;
            }
            window.dispatchEvent(new CustomEvent('oblivra:capture-evidence'));
        }
    }
  }
</script>

<svelte:window onkeydown={onKeyDown} onresize={handleResize} />

<main
  bind:this={mainEl}
  class="h-screen w-screen overflow-hidden bg-background text-foreground font-sans transition-colors duration-slow"
>
  {#if ready}
    <div class="flex h-full w-full">
      {#if !isMobile && !popoutMode}
        <Sidebar />
      {/if}

      <div class="relative flex flex-1 flex-col overflow-hidden">
        <TopBar />

        <!-- Crisis Mode Banner -->
        {#if crisisStore.active}
          <div
            class="crisis-banner flex items-center justify-between px-4 py-1.5 bg-error/10 border-b border-error/40 text-[10px] font-mono shrink-0"
            role="alert"
            aria-live="assertive"
          >
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full bg-error animate-pulse shrink-0"></span>
              <span class="font-extrabold uppercase tracking-widest text-error">CRISIS MODE ACTIVE</span>
              {#if crisisStore.reason}
                <span class="text-text-muted opacity-60">— {crisisStore.reason}</span>
              {/if}
            </div>
            <button
              class="text-text-muted hover:text-text-secondary transition-colors border border-border-primary rounded-sm px-2 py-0.5 hover:border-border-hover"
              onclick={() => crisisStore.standDown()}
            >Stand Down</button>
          </div>
        {/if}

        <div class="flex-1 overflow-auto relative">
          <RouterView {routes} />
        </div>
        
        {#if isMobile}
          <BottomNav />
        {/if}
      </div>

      {#if appStore.showCommandPalette}
        <CommandPalette bind:open={appStore.showCommandPalette} />
      {/if}

      <ToastContainer />
    </div>
  {:else if error}
    <ErrorScreen message={error} />
  {:else}
    <LoadingScreen />
  {/if}
</main>

<style>
  :global(main.crisis-mode-active) {
    --surface-1: #1f0e0e;
    --surface-2: #2a1111;
    --surface-3: #3a1818;
    --bg-primary: #1a0a0a;
  }
  :global(main.crisis-mode-active .cr-rail) {
    border-right-color: rgba(224, 64, 64, 0.25) !important;
  }
</style>
