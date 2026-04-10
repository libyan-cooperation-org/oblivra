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
    
    // Find first matching route
    for (const route of routes) {
      if (matchRoute(route.path, path)) {
        return route;
      }
    }
    
    // Fallback if no match (already handled by routes list usually)
    return routes.find(r => r.path === '*') || null;
  });
</script>

{#if currentMatchedRoute}
  {@const Component = currentMatchedRoute.component}
  <Component />
{/if}
