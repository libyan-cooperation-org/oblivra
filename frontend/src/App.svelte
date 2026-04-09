<!--
  OBLIVRA — Root App Component (Svelte 5)

  Initializes bridge, stores, and events, then renders the layout shell
  with all routes wired through the hash router.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { initBridge } from '@lib/bridge';
  import { appStore } from '@lib/stores/app.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { APP_CONTEXT } from '@lib/context';

  // Layout
  import AppLayout from '@components/layout/AppLayout.svelte';
  import RouterView from '@components/layout/Router.svelte';
  import type { RouteDefinition } from '@components/layout/Router.svelte';
  import ToastContainer from '@components/layout/ToastContainer.svelte';

  // UI
  import LoadingScreen from '@components/ui/LoadingScreen.svelte';
  import ErrorScreen from '@components/ui/ErrorScreen.svelte';

  // Pages — migrated
  import Dashboard from '@pages/Dashboard.svelte';

  // Pages — placeholder for unmigrated
  import Placeholder from '@pages/Placeholder.svelte';

  // ── App state
  let ready = $state(false);
  let error = $state<string | null>(null);

  // ── Route definitions — ALL 60+ routes
  const routes: RouteDefinition[] = [
    // Root
    { path: '/', component: Dashboard },

    // Dashboard (migrated)
    { path: '/dashboard', component: Dashboard },

    // SIEM & Alerts
    { path: '/siem', component: Placeholder },
    { path: '/siem-search', component: Placeholder },
    { path: '/alerts', component: Placeholder },
    { path: '/alert-management', component: Placeholder },

    // Monitoring & Topology
    { path: '/monitoring', component: Placeholder },
    { path: '/topology', component: Placeholder },
    { path: '/mitre-heatmap', component: Placeholder },

    // Operations
    { path: '/ops', component: Placeholder },
    { path: '/terminal', component: Placeholder },
    { path: '/tunnels', component: Placeholder },
    { path: '/hosts', component: Placeholder },
    { path: '/recordings', component: Placeholder },
    { path: '/snippets', component: Placeholder },
    { path: '/notes', component: Placeholder },

    // SOC / Fleet (browser)
    { path: '/soc', component: Placeholder },
    { path: '/agents', component: Placeholder },
    { path: '/fleet', component: Placeholder },
    { path: '/fleet-management', component: Placeholder },

    // Incident Response
    { path: '/response', component: Placeholder },
    { path: '/escalation', component: Placeholder },
    { path: '/playbook-builder', component: Placeholder },

    // Intelligence
    { path: '/ueba', component: Placeholder },
    { path: '/ueba-overview', component: Placeholder },
    { path: '/threat-hunter', component: Placeholder },
    { path: '/threat-intel', component: Placeholder },
    { path: '/threat-intel-dashboard', component: Placeholder },
    { path: '/enrichment', component: Placeholder },
    { path: '/ndr', component: Placeholder },
    { path: '/ndr-overview', component: Placeholder },
    { path: '/purple-team', component: Placeholder },
    { path: '/graph', component: Placeholder },
    { path: '/credentials', component: Placeholder },

    // Governance & Compliance
    { path: '/compliance', component: Placeholder },
    { path: '/governance', component: Placeholder },
    { path: '/vault', component: Placeholder },
    { path: '/trust', component: Placeholder },
    { path: '/forensics', component: Placeholder },
    { path: '/remote-forensics', component: Placeholder },
    { path: '/ransomware', component: Placeholder },
    { path: '/ransomware-ui', component: Placeholder },

    // Identity (browser)
    { path: '/identity', component: Placeholder },
    { path: '/identity-admin', component: Placeholder },

    // War Mode & Security
    { path: '/war-mode', component: Placeholder },
    { path: '/data-destruction', component: Placeholder },

    // Audit Trail
    { path: '/temporal-integrity', component: Placeholder },
    { path: '/lineage', component: Placeholder },
    { path: '/decisions', component: Placeholder },
    { path: '/ledger', component: Placeholder },
    { path: '/response-replay', component: Placeholder },

    // Executive & Analytics
    { path: '/executive', component: Placeholder },
    { path: '/analytics', component: Placeholder },
    { path: '/simulation', component: Placeholder },

    // AI & Workspace
    { path: '/ai-assistant', component: Placeholder },
    { path: '/workspace', component: Placeholder },
    { path: '/fusion', component: Placeholder },

    // System
    { path: '/settings', component: Placeholder },
    { path: '/plugins', component: Placeholder },
    { path: '/team', component: Placeholder },
    { path: '/sync', component: Placeholder },
    { path: '/offline-update', component: Placeholder },
    { path: '/license', component: Placeholder },
    { path: '/features', component: Placeholder },
    { path: '/risk', component: Placeholder },
    { path: '/entity', component: Placeholder },

    // Catch-all fallback
    { path: '*', component: Placeholder },
  ];

  onMount(async () => {
    try {
      await initBridge();

      // Hook global system events to toast notifications
      const rt = (window as any).runtime;
      if (rt && APP_CONTEXT !== 'browser') {
        rt.EventsOn('system.error', (msg: string) => {
          toastStore.add({ type: 'error', title: 'System Error', message: msg });
        });
        rt.EventsOn('system.toast', (toast: any) => {
          toastStore.add(toast);
        });
      }

      // Initialize the app store (event subscriptions, initial state)
      await appStore.init();

      ready = true;
    } catch (err) {
      error = `${err}`;
    }
  });
</script>

{#if ready}
  <div class="h-full w-full animate-fade-in">
    <AppLayout>
      {#snippet children()}
        <RouterView {routes} />
      {/snippet}
    </AppLayout>
    <ToastContainer />
  </div>
{:else if error}
  <ErrorScreen message={error} />
{:else}
  <LoadingScreen />
{/if}
