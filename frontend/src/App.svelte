<!--
  OBLIVRA — Root App Component (Svelte 5)

  Replaces App.tsx + index.tsx router setup.
  Uses a minimal hash-based router for Wails compatibility.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { initBridge } from '@lib/bridge';
  import { appStore } from '@lib/stores/app.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { APP_CONTEXT } from '@lib/context';
  import Router from '@components/layout/Router.svelte';
  import type { RouteDefinition } from '@components/layout/Router.svelte';

  // ── Pages
  import Dashboard from '@pages/Dashboard.svelte';
  import LoadingScreen from '@components/ui/LoadingScreen.svelte';
  import ErrorScreen from '@components/ui/ErrorScreen.svelte';
  import ToastContainer from '@components/layout/ToastContainer.svelte';

  // ── App state
  let ready = $state(false);
  let error = $state<string | null>(null);

  // ── Route definitions
  // As components are migrated from SolidJS, add them here.
  const routes: RouteDefinition[] = [
    { path: '/', component: Dashboard },
    { path: '/dashboard', component: Dashboard },

    // Catch-all fallback during migration
    { path: '*', component: Dashboard },
  ];

  onMount(async () => {
    try {
      await initBridge();

      // Hook Global Toasts into Wails Events (desktop only)
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
    <Router {routes} />
    <ToastContainer />
  </div>
{:else if error}
  <ErrorScreen message={error} />
{:else}
  <LoadingScreen />
{/if}
