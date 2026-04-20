<script lang="ts">
  import { onMount } from 'svelte';
  import { isAuthenticated } from './services/auth';
  import { IS_DESKTOP } from './context';
  import { router, push } from './core/router.svelte';
  import LoadingScreen from './components/ui/LoadingScreen.svelte';

  // ── Pages (lazy via dynamic import map)
  // ── Pages (Dynamic Imports)
  const Login             = () => import('./pages/Login.svelte');
  const Onboarding        = () => import('./pages/Onboarding.svelte');
  const Dashboard         = () => import('./core/Dashboard.svelte');
  const FleetManagement   = () => import('./pages/FleetManagement.svelte');
  const SIEMSearch        = () => import('./pages/SIEMSearch.svelte');
  const IdentityAdmin     = () => import('./pages/IdentityAdmin.svelte');
  const AlertManagement   = () => import('./pages/AlertManagement.svelte');
  const LookupManager     = () => import('./pages/LookupManager.svelte');
  const EscalationCenter  = () => import('./pages/EscalationCenter.svelte');
  const ThreatIntelDash   = () => import('./pages/ThreatIntelDashboard.svelte');
  const EnrichmentViewer  = () => import('./pages/EnrichmentViewer.svelte');
  const MitreHeatmap      = () => import('./pages/MitreHeatmap.svelte');
  const PlaybookBuilder   = () => import('./pages/PlaybookBuilder.svelte');
  const UEBADashboard     = () => import('./pages/UEBADashboard.svelte');
  const NDRDashboard      = () => import('./pages/NDRDashboard.svelte');
  const RansomwareCenter  = () => import('./pages/RansomwareCenter.svelte');
  const RegulatorPortal   = () => import('./pages/RegulatorPortal.svelte');
  const EvidenceVault     = () => import('./pages/EvidenceVault.svelte');
  const PlaybookMetrics   = () => import('./pages/PlaybookMetrics.svelte');
  const PeerAnalytics     = () => import('./pages/PeerAnalytics.svelte');
  const FusionDashboard   = () => import('./pages/FusionDashboard.svelte');
  const Investigation     = () => import('./pages/InvestigationCanvas.svelte');

  const PUBLIC_PATHS = ['/login', '/onboarding'];

  const ROUTES: Record<string, any> = {
    '/':                 Dashboard,
    '/login':            Login,
    '/onboarding':       Onboarding,
    '/siem/search':      SIEMSearch,
    '/alerts':           AlertManagement,
    '/lookups':          LookupManager,
    '/threatintel':      ThreatIntelDash,
    '/enrich':           EnrichmentViewer,
    '/mitre-heatmap':    MitreHeatmap,
    '/fleet':            FleetManagement,
    '/identity':         IdentityAdmin,
    '/escalation':       EscalationCenter,
    '/playbooks':        PlaybookBuilder,
    '/ueba':             UEBADashboard,
    '/ndr':              NDRDashboard,
    '/ransomware':       RansomwareCenter,
    '/regulator':        RegulatorPortal,
    '/evidence':         EvidenceVault,
    '/playbook-metrics': PlaybookMetrics,
    '/peer-analytics':   PeerAnalytics,
    '/fusion':           FusionDashboard,
    '/investigation':    Investigation,
  };

  let ready = $state(false);

  onMount(() => {
    if (!IS_DESKTOP) {
      const path = window.location.pathname;
      if (!isAuthenticated() && !PUBLIC_PATHS.includes(path)) {
        push('/login');
      }
    }
    ready = true;
  });

  import { fade } from 'svelte/transition';
  import TacticalSidebar from './components/ui/TacticalSidebar.svelte';

  const loader = $derived(ROUTES[router.path] ?? Dashboard);
  const showShell = $derived(!PUBLIC_PATHS.includes(router.path));
</script>

{#if ready}
  <div class="flex h-screen w-screen overflow-hidden bg-surface-0">
    {#if showShell}
      <TacticalSidebar />
    {/if}

    <main class="flex-1 relative overflow-hidden">
      {#await loader()}
        <div class="absolute inset-0 z-50 bg-surface-0 flex items-center justify-center" transition:fade={{ duration: 150 }}>
          <LoadingScreen />
        </div>
      {:then module}
        <div class="h-full w-full" transition:fade={{ duration: 200 }}>
          <module.default />
        </div>
      {:catch error}
        <div class="h-full w-full flex items-center justify-center bg-surface-0 text-error font-mono" transition:fade>
          TACTICAL_LOAD_FAILURE: {error.message}
        </div>
      {/await}
    </main>
  </div>
{:else}
  <LoadingScreen />
{/if}
