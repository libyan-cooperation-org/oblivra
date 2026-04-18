<script lang="ts">
  import { getCurrentPath, matchRoute } from '@lib/router.svelte';

  interface RouteDefinition {
    path: string;
    component: any;
  }

  interface Props {
    routes: RouteDefinition[];
  }

  let { routes }: Props = $props();

  // Find the matching route reactively
  let currentMatchedRoute = $derived.by(() => {
    const path = getCurrentPath();
    
    for (const route of routes) {
      const match = matchRoute(route.path, path);
      if (match) {
        return { component: route.component, params: match.params };
      }
    }
    
    const fallback = routes.find(r => r.path === '*');
    return fallback ? { component: fallback.component, params: {} } : null;
  });
</script>

{#if currentMatchedRoute}
  <currentMatchedRoute.component {...currentMatchedRoute.params} />
{/if}
