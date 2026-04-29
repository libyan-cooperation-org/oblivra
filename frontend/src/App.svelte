<script lang="ts">
  console.log("App.svelte <script> is executing!");
  import { onMount, onDestroy } from 'svelte';
  import RouterView from '@components/RouterView.svelte';
  import { initBridge, APP_CONTEXT } from '@lib/bridge';
  import { appStore } from '@lib/stores/app.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { crisisStore } from '@lib/stores/crisis.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
import Sidebar from '@components/layout/CommandRail.svelte';
  import AppSidebar from '@components/layout/AppSidebar.svelte';
  import BottomDock from '@components/layout/BottomDock.svelte';
  import InvestigationPanel from '@components/layout/InvestigationPanel.svelte';
  import { navigationStore } from '@lib/stores/navigation.svelte';
  import TopBar from '@components/layout/TitleBar.svelte';
  import PivotCrumb from '@components/layout/PivotCrumb.svelte';
  import TLSBanner from '@components/layout/TLSBanner.svelte';
  import CrisisDecisionPanel from '@components/layout/CrisisDecisionPanel.svelte';
  import CommandPalette from '@components/ui/CommandPalette.svelte';
  import TenantFastSwitcher from '@components/ui/TenantFastSwitcher.svelte';
  import ToastContainer from '@components/layout/ToastContainer.svelte';
  import NotificationDrawer from '@components/layout/NotificationDrawer.svelte';
  import BottomNav from '@components/layout/BottomNav.svelte';
  import { notificationStore } from '@lib/stores/notifications.svelte';
  import LoadingScreen from '@components/ui/LoadingScreen.svelte';
  import ErrorScreen from '@components/ui/ErrorScreen.svelte';
  import { siemStore } from '@lib/stores/siem.svelte';

  // ── Pages
  import Dashboard from '@pages/Dashboard.svelte';
  // ShellPage import removed — shell subsystem deleted in Phase 33,
  // pending replacement. The /shell route now falls through to the
  // wildcard DevelopmentPage placeholder.
  import RecordingsPage from '@pages/RecordingsPage.svelte';
  import SIEMPanel from '@pages/SIEMPanel.svelte';
  import AlertDashboard from '@pages/AlertDashboard.svelte';
  import AlertManagement from '@pages/AlertManagement.svelte';
  import OfflineUpdate from '@pages/OfflineUpdate.svelte';
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
  import HostDetail from '@pages/HostDetail.svelte';
  import Overview from '@pages/Overview.svelte';
  import UEBAOverview from '@pages/UEBAOverview.svelte';
  import NDROverview from '@pages/NDROverview.svelte';
  import EnrichmentViewer from '@pages/EnrichmentViewer.svelte';
  import AgentConsole from '@pages/AgentConsole.svelte';
  import RuntimeTrust from '@pages/RuntimeTrust.svelte';
  import DataDestruction from '@pages/DataDestruction.svelte';
  import LineageExplorer from '@pages/LineageExplorer.svelte';
  import DecisionInspector from '@pages/DecisionInspector.svelte';
  import OQLDashboard from '@pages/OQLDashboard.svelte';
  import EvidenceLedger from '@pages/EvidenceLedger.svelte';
  import ChainOfCustody from '@pages/ChainOfCustody.svelte';
  import FusionDashboard from '@pages/FusionDashboard.svelte';
  import Settings from '@pages/Settings.svelte';
  import TeamDashboard from '@pages/TeamDashboard.svelte';
  import SyncPage from '@pages/SyncPage.svelte';
  import ConfigRisk from '@pages/ConfigRisk.svelte';
  import EntityView from '@pages/EntityView.svelte';
  import WarMode from '@pages/WarMode.svelte';
  import FleetMap from '@pages/FleetMap.svelte';
  import InvestigationDashboard from '@pages/InvestigationDashboard.svelte';
  import SecretManager from '@pages/SecretManager.svelte';
  import SuppressionManager from '@pages/SuppressionManager.svelte';
  import DevelopmentPage from '@pages/DevelopmentPage.svelte';
  import SetupWizard from '@pages/SetupWizard.svelte';
  import EvidenceVault from '@pages/EvidenceVault.svelte';
  import OperatorMode from '@pages/OperatorMode.svelte';
  import MultiTenantAdmin from '@pages/MultiTenantAdmin.svelte';
  import KeyboardMap from '@pages/KeyboardMap.svelte';
  import DashboardStudio from '@pages/DashboardStudio.svelte';
  import DSRConsole from '@pages/DSRConsole.svelte';
  import IdentityConnectors from '@pages/IdentityConnectors.svelte';
  import AuditLog from '@pages/AuditLog.svelte';
  import Connectors from '@pages/Connectors.svelte';
  import Integrity from '@pages/Integrity.svelte';
  import StorageTiering from '@pages/StorageTiering.svelte';
  import ProfileWizard from '@components/onboarding/ProfileWizard.svelte';

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

  // UIUX_IMPROVEMENTS.md P2 #16 — three viewport bands instead of two:
  //   < 768  → mobile  (BottomNav, no sidebar, single-column KPIs)
  //   < 1024 → tablet  (icon-only sidebar, BottomDock labels hidden)
  //   ≥ 1024 → desktop (full chrome)
  // 768 was 640 originally, but real-world phones in landscape (e.g.
  // iPhone 14 Pro at 852 px) stayed in mobile mode and broke the
  // 12-col grid. Bumping the floor and adding a tablet step covers
  // the whole device space.
  const isMobile = $derived(screenWidth < 768);
  const isTablet = $derived(screenWidth >= 768 && screenWidth < 1024);

  function handleResize() {
    screenWidth = window.innerWidth;
  }

  // Push the breakpoint band onto <body data-vp> so CSS / pages can
  // react without prop-drilling.
  $effect(() => {
    if (typeof document === 'undefined') return;
    const band = isMobile ? 'mobile' : isTablet ? 'tablet' : 'desktop';
    document.body.setAttribute('data-vp', band);
  });

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
    // /shell removed — shell subsystem deleted Phase 33; pending replacement.
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
    { path: '/overview',         component: Overview },
    { path: '/host/:id',         component: HostDetail },
    { path: '/fleet',            component: FleetDashboard },
    { path: '/fleet-management', component: FleetDashboard },
    { path: '/hosts',            component: FleetDashboard },
    { path: '/soc',              component: FleetDashboard },
    { path: '/agents',           component: FleetDashboard },
    { path: '/fleet-map',        component: FleetMap },
    { path: '/workspace',        component: Dashboard },
    { path: '/fusion',           component: FusionDashboard },

    // Security & Incident Response
    { path: '/escalation',       component: EscalationCenter },
    { path: '/purple-team',      component: PurpleTeam },
    { path: '/war-mode',         component: WarMode },
    { path: '/data-destruction', component: DataDestruction },
    { path: '/simulation',       component: SimulationPanel },

    // Forensics & Audit
    { path: '/terminal-forensics', component: TerminalForensics },
    { path: '/lineage',            component: LineageExplorer },
    { path: '/decisions',          component: DecisionInspector },
    { path: '/oql',                component: OQLDashboard },
    { path: '/dashboard-studio',   component: DashboardStudio },
    { path: '/evidence',           component: EvidenceLedger },
    { path: '/ledger',             component: EvidenceLedger },
    { path: '/chain-of-custody',   component: ChainOfCustody },
    { path: '/temporal-integrity', component: TemporalIntegrity },
    { path: '/response-replay',    component: ResponseReplay },

    // Topology
    { path: '/topology',        component: TopologyPage },
    { path: '/network-map',     component: NetworkMap },
    { path: '/global-topology', component: GlobalTopology },
    { path: '/mitre-heatmap',   component: MitreHeatmap },

    // Governance, Trust & Identity
    { path: '/vault',          component: EvidenceVault },
    { path: '/evidence-vault', component: EvidenceVault },
    { path: '/setup',          component: SetupWizard },
    { path: '/setup-wizard',   component: SetupWizard },
    { path: '/trust',          component: RuntimeTrust },
    { path: '/runtime-trust',  component: RuntimeTrust },
    { path: '/identity-admin', component: IdentityAdmin },
    { path: '/identity-connectors', component: IdentityConnectors },
    { path: '/dsr', component: DSRConsole },
    { path: '/audit',          component: AuditLog },
    { path: '/admin',          component: MultiTenantAdmin },
    { path: '/shortcuts',      component: KeyboardMap },
    { path: '/secrets',        component: SecretManager },
    { path: '/suppression',    component: SuppressionManager },
    { path: '/secret-manager', component: SecretManager },

    // Management
    { path: '/settings',       component: Settings },
    { path: '/team',           component: TeamDashboard },
    { path: '/sync',           component: SyncPage },
    { path: '/offline-update', component: OfflineUpdate },
    { path: '/license',        component: LicensePage },
    { path: '/features',       component: FeaturesPage },
    { path: '/risk',           component: ConfigRisk },
    { path: '/entity',         component: EntityView },
    { path: '/development',    component: DevelopmentPage },
    { path: '/connectors',     component: Connectors },
    { path: '/integrity',      component: Integrity },
    { path: '/storage-tiering', component: StorageTiering },

    // Fallback
    { path: '*', component: DevelopmentPage },
  ];

  // Stash the runtime cleanup callbacks so onDestroy can release them.
  // Wails v3's EventsOn returns an unsubscribe function; we collect them
  // here and run them all on teardown so HMR / component remounts don't
  // accumulate duplicate listeners (audit finding HIGH-1).
  let runtimeUnsubs: Array<() => void> = [];

  // Lazy single-shot loader for the WindowService binding. Multiple menu
  // handlers used to dynamic-import this in parallel; cache and reuse.
  let windowSvc: any = null;
  async function getWindowService() {
    if (!windowSvc) {
      windowSvc = await import(
        '../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js'
      ).catch((err) => {
        console.error('[App] WindowService binding load failed:', err);
        return null;
      });
    }
    return windowSvc;
  }

  function on(rt: any, name: string, fn: (...args: any[]) => void) {
    // Wails v3's EventsOn returns an unsubscribe handle. v2 returned void
    // and required EventsOff. Support both.
    const result = rt.EventsOn(name, fn);
    if (typeof result === 'function') {
      runtimeUnsubs.push(result);
    } else if (typeof rt.EventsOff === 'function') {
      runtimeUnsubs.push(() => rt.EventsOff(name));
    }
  }

  /**
   * Audit fix High-7: when apiFetch detects 401/403 it dispatches
   * `oblivra:auth-failure` so this single listener can route the
   * operator to /login with a friendly toast. Idempotent: the
   * window event itself is single-flight on the apiClient side.
   */
  function onAuthFailure(e: Event) {
    const detail = (e as CustomEvent<{ status: number; url: string }>).detail;
    toastStore.add({
      type: 'warning',
      title: 'Session expired',
      message: `Please sign in again (${detail?.status ?? 401} on ${detail?.url ?? 'request'}).`,
    });
    appStore.navigate('/login');
  }

  onMount(async () => {
    try {
      await initBridge();
      window.addEventListener('oblivra:auth-failure', onAuthFailure);

      // UIUX_IMPROVEMENTS.md P0 #3 — only the desktop (Wails) build
      // traps document scroll inside an inner pane. The web build needs
      // natural document scroll for browser pull-to-refresh, middle-
      // click drag, and Cmd+Home/End. The class scopes that behaviour.
      if (typeof document !== 'undefined' && APP_CONTEXT !== 'browser') {
        document.body.classList.add('app-shell');
      }

      const rt = (window as any).runtime;
      if (rt && APP_CONTEXT !== 'browser') {
        console.debug('[App] Diagnostics initialized:', diagnosticsStore.healthGrade);
        on(rt, 'system.error', (msg: string) => {
          toastStore.add({ type: 'error', title: 'System Error', message: msg });
        });
        on(rt, 'system.toast', (toast: any) => {
          toastStore.add(toast);
        });

        // ── Menu / tray event handlers ────────────────────────────────
        // The native application menu and system tray (internal/app/menu.go,
        // internal/app/tray.go) emit menu:* / tray:* events instead of
        // directly mutating frontend state — this keeps the menu Go code
        // ignorant of router/UI internals.
        on(rt, 'menu:goto', (route: string) => {
          appStore.navigate(route);
        });
        on(rt, 'menu:new-terminal', () => {
          // /shell route is offline pending shell-subsystem replacement
          // (Phase 33). Surface a notice instead of navigating to a 404.
          appStore.notify(
            'Shell workspace is offline',
            'info',
            'The shell subsystem is being rebuilt. Use /operator for SIEM-aware host context until the new shell ships.',
          );
        });
        on(rt, 'menu:command-palette', () => {
          appStore.showCommandPalette = true;
        });
        on(rt, 'menu:toggle-sidebar', () => {
          // appStore exposes `sidebarOpen`, not `sidebarVisible`.
          appStore.toggleSidebar();
        });
        on(rt, 'menu:popout-current', async () => {
          const mod = await getWindowService();
          if (!mod) return;
          try {
            // Phase 35 audit fix #12 — same hash-routing bug as
            // PopOutButton. The app uses hash routing (#/foo) so
            // window.location.pathname is always '/' and Ctrl+Shift+O
            // always popped out the dashboard regardless of view.
            // Use the router's getCurrentPath() instead.
            const { getCurrentPath } = await import('@lib/router.svelte');
            const route = getCurrentPath() || '/';
            await mod.PopOut(route, document.title || '');
          } catch (e) {
            toastStore.add({ type: 'error', title: 'Pop-out failed', message: String(e) });
          }
        });
        on(rt, 'menu:close-popouts', async () => {
          const mod = await getWindowService();
          if (!mod) return;
          try {
            const closed = await mod.CloseAllPopouts();
            toastStore.add({ type: 'info', title: 'Pop-outs closed', message: `${closed} window(s)` });
          } catch (e) {
            console.warn('[App] CloseAllPopouts failed:', e);
          }
        });
        on(rt, 'menu:save-workspace', async () => {
          const mod = await getWindowService();
          if (!mod) return;
          try {
            const count = await mod.SaveWorkspace();
            toastStore.add({ type: 'info', title: 'Workspace saved', message: `${count} pop-out(s)` });
          } catch (e) {
            toastStore.add({ type: 'error', title: 'Workspace save failed', message: String(e) });
          }
        });
        on(rt, 'menu:restore-workspace', async () => {
          const mod = await getWindowService();
          if (!mod) return;
          try {
            const count = await mod.RestoreWorkspace(true);
            toastStore.add({ type: 'info', title: 'Workspace restored', message: `${count} pop-out(s)` });
          } catch (e) {
            toastStore.add({ type: 'error', title: 'Workspace restore failed', message: String(e) });
          }
        });
        on(rt, 'menu:diagnostics', () => {
          // Phase 35 cleanup — `appStore.showDiagnostics()` was a
          // dangling reference (no such method exists on appStore).
          // We always fell through to the navigate path; just call
          // it directly. The /monitoring page is the diagnostics
          // surface (live EPS, goroutines, heap, GC, health grade).
          appStore.navigate('/monitoring');
        });
        on(rt, 'menu:open-url', (url: string) => {
          window.open(url, '_blank', 'noopener,noreferrer');
        });
        on(rt, 'tray:popout', async (route: string) => {
          const mod = await getWindowService();
          if (!mod) return;
          try {
            await mod.PopOut(route, '');
          } catch (e) {
            console.warn('[App] tray:popout failed:', e);
          }
        });
      }

      await appStore.init();
      crisisStore.init();
      navigationStore.init();
      siemStore.refreshStats();

      // Hot-load IOC tables for inline terminal underlining (Phase 32).
      // Refreshes every 60 s; safe in browser mode (uses REST) and
      // desktop (still hits the embedded REST server on localhost).
      const { threatIntelStore } = await import('@lib/stores/threatIntel.svelte');
      threatIntelStore.start();

      // Tamper-evidence integrity counts (Tamper Path 1). Drives the
      // Tactical Hub KPI tile + Crisis auto-arm threshold.
      const { integrityStore } = await import('@lib/stores/integrity.svelte');
      integrityStore.start();

      // In pop-out mode the spawned window starts at "/?popout=1&route=X"
      // — navigate to the requested route now that the router store is up.
      if (popoutMode && popoutRoute) {
        appStore.navigate(popoutRoute);
      } else if (appStore.profileChosen) {
        // Operator-Profile-driven home routing (Phase 32). When the
        // operator lands on the bare "/", redirect to whatever route
        // their profile considers home (alerts for SOC, search for
        // hunters, war-mode for incident commanders, admin for MSP).
        // Skip if the URL hash already has a target — the operator
        // intentionally deep-linked.
        const hash = window.location.hash.replace('#', '');
        if (!hash || hash === '/' || hash === '/dashboard') {
          const home = appStore.profileRules.homeRoute;
          if (home && home !== '/' && home !== '/dashboard') {
            appStore.navigate(home);
          }
        }
      }

      ready = true;
    } catch (e: any) {
      console.error('App init failed:', e);
      error = e.message || 'Failed to initialize OBLIVRA core.';
    }
  });

  onDestroy(() => {
    // Drop every Wails event listener we registered in onMount. Without
    // this, a parent component remount (HMR or programmatic) accumulates
    // duplicate listeners and the same event fires N times.
    for (const off of runtimeUnsubs) {
      try { off(); } catch { /* already torn down */ }
    }
    window.removeEventListener('oblivra:auth-failure', onAuthFailure);
    runtimeUnsubs = [];
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

    // ── Global Navigation (G + key) — only when the profile enables
    // vim leader keys. SOC Analyst profile keeps this off so a stray
    // 'g' in a search box doesn't get hijacked. Threat Hunter / MSP
    // profiles ship it on by default.
    // We also bail when focus is in an editable element so typing 'g'
    // into Input / textarea / xterm doesn't trigger navigation.
    if (e.key.toLowerCase() === 'g' && appStore.profileRules.vimLeader) {
        const target = e.target as HTMLElement | null;
        const editable =
          target?.tagName === 'INPUT' ||
          target?.tagName === 'TEXTAREA' ||
          target?.isContentEditable === true ||
          target?.closest('.xterm') !== null;
        if (editable) return;
        const nextKeyHandler = (ne: KeyboardEvent) => {
            const key = ne.key.toLowerCase();
            if (key === 'd') appStore.navigate('/dashboard');
            if (key === 'a') appStore.navigate('/alerts');
            if (key === 's') appStore.navigate('/siem');
            if (key === 'f') appStore.navigate('/fleet');
            if (key === 'v') appStore.navigate('/evidence');
            if (key === 'h') appStore.navigate('/siem-search');  // 'h' for hunt
            if (key === 'o') appStore.navigate('/operator');
            if (key === 't') appStore.navigate('/timeline');
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
        {#if appStore.useGroupedNav}
          <AppSidebar />
        {:else}
          <Sidebar />
        {/if}
      {/if}

      <div class="relative flex flex-1 flex-col overflow-hidden">
        <TopBar />

        <!-- TLS plaintext banner — auto-hides when TLS is on. Sits
             above the pivot trail so operators see the security
             posture before the navigation crumb. -->
        <TLSBanner />

        <!-- Pivot trail — auto-hides when the trail is empty. -->
        <PivotCrumb />

        <!-- Crisis Mode Banner — height fixed to --banner-h (28px) so
             switching between routes that use this banner vs SystemBanner
             vs neither doesn't shift the content area. See
             UIUX_IMPROVEMENTS.md P0 #4. -->
        {#if crisisStore.active}
          <div
            class="crisis-banner flex items-center justify-between px-4 bg-error/10 border-b border-error/40 font-mono shrink-0"
            style="height: var(--banner-h, 28px); font-size: var(--fs-micro, 10px);"
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

        <!-- New v1.5.0 chrome: contextual bottom dock for the active
             nav group. Hidden on mobile (BottomNav takes its place)
             and when the operator opted into the legacy CommandRail. -->
        {#if !isMobile && !popoutMode && appStore.useGroupedNav}
          <BottomDock />
        {/if}

        {#if isMobile}
          <BottomNav />
        {/if}
      </div>

      {#if appStore.showCommandPalette}
        <CommandPalette bind:open={appStore.showCommandPalette} />
      {/if}

      <ToastContainer />
      <NotificationDrawer />

      <!-- Global investigation drawer — slides in whenever any
           EntityLink (host/user/IP/process/...) is clicked anywhere
           in the app. Phase 31 SOC redesign. -->
      <InvestigationPanel />

      <!-- First-run Operator Profile picker. Self-gates on
           appStore.profileChosen so it disappears after one decision. -->
      <ProfileWizard />

      <!-- Crisis Mode Decision Support — slides in when crisisStore.active
           and the active profile picks the 'banner' affordance. The
           'fullscreen-takeover' affordance still has /war-mode for the
           commander persona. -->
      <CrisisDecisionPanel />

      <!-- Tenant fast switcher — Cmd+T (or Ctrl+Alt+T on Win/Linux)
           when the profile sets tenantChrome=switcher-bar. Self-gates
           on profile so non-MSP users don't see surprise modals. -->
      <TenantFastSwitcher />
    </div>
  {:else if error}
    <ErrorScreen message={error} />
  {:else}
    <LoadingScreen />
  {/if}
</main>

<style>
  /* Crisis Mode — red-tinted surfaces. The previous version targeted
     `--surface-1` (without the `--color-` prefix) which doesn't match
     what app.css `@theme` defines, so the colour shift never landed.
     UIUX_IMPROVEMENTS.md P1 #10. */
  :global(main.crisis-mode-active) {
    --color-bg-primary: #1a0a0a;
    --color-surface-0: #1a0a0a;
    --color-surface-1: #1f0e0e;
    --color-surface-2: #2a1111;
    --color-surface-3: #3a1818;
    --bg: #1a0a0a;
    --s1: #1f0e0e;
    --s2: #2a1111;
    --s3: #3a1818;
    border-top: 1px solid rgba(224, 64, 64, 0.4);
  }
  :global(main.crisis-mode-active .cr-rail) {
    border-right-color: rgba(224, 64, 64, 0.25) !important;
  }
  /* Focus ring contrast — the global blue ring is invisible on red
     surfaces. Switch to a soft red for crisis mode. P2 #12. */
  :global(main.crisis-mode-active *:focus-visible) {
    outline-color: #ffb0b0 !important;
  }
</style>
