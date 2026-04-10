<script lang="ts">
  import { onMount } from 'svelte';
  import { isAuthenticated } from './services/auth';
  import { IS_DESKTOP } from './context';
  import { router, push } from './core/router.svelte';
  import LoadingScreen from './components/ui/LoadingScreen.svelte';

  // ── Pages (lazy via dynamic import map)
  import Login             from './pages/Login.svelte';
  import Onboarding        from './pages/Onboarding.svelte';
  import Dashboard         from './core/Dashboard.svelte';
  import FleetManagement   from './pages/FleetManagement.svelte';
  import SIEMSearch        from './pages/SIEMSearch.svelte';
  import IdentityAdmin     from './pages/IdentityAdmin.svelte';
  import AlertManagement   from './pages/AlertManagement.svelte';
  import LookupManager     from './pages/LookupManager.svelte';
  import EscalationCenter  from './pages/EscalationCenter.svelte';
  import ThreatIntelDash   from './pages/ThreatIntelDashboard.svelte';
  import EnrichmentViewer  from './pages/EnrichmentViewer.svelte';
  import MitreHeatmap      from './pages/MitreHeatmap.svelte';
  import PlaybookBuilder   from './pages/PlaybookBuilder.svelte';
  import UEBADashboard     from './pages/UEBADashboard.svelte';
  import NDRDashboard      from './pages/NDRDashboard.svelte';
  import RansomwareCenter  from './pages/RansomwareCenter.svelte';
  import RegulatorPortal   from './pages/RegulatorPortal.svelte';
  import EvidenceVault     from './pages/EvidenceVault.svelte';
  import PlaybookMetrics   from './pages/PlaybookMetrics.svelte';
  import PeerAnalytics     from './pages/PeerAnalytics.svelte';
  import FusionDashboard   from './pages/FusionDashboard.svelte';

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

  const currentComponent = $derived(ROUTES[router.path] ?? Dashboard);
</script>

{#if ready}
  <svelte:component this={currentComponent} />
{:else}
  <LoadingScreen />
{/if}
