<!--
  OBLIVRA — Hash Router Component (Svelte 5)

  Renders the component matching the current hash route.
  Uses the minimal router from lib/router.svelte.ts.
-->
<script lang="ts" module>
  import type { Component } from 'svelte';

  export interface RouteDefinition {
    path: string;
    component: Component<any>;
    /** 'desktop' | 'browser' | 'both' — defaults to 'both' */
    availability?: 'desktop' | 'browser' | 'both';
  }
</script>

<script lang="ts">
  import { onMount, onDestroy, type Snippet } from 'svelte';
  import { matchRoute, getCurrentPath, type RouteParams } from '@lib/router.svelte';

  interface Props {
    routes: RouteDefinition[];
    fallback?: Component<any>;
  }

  let { routes, fallback }: Props = $props();

  let currentPath = $state(getCurrentPath());
  let matchedRoute = $state<RouteDefinition | null>(null);
  let routeParams = $state<RouteParams>({});

  function updateRoute() {
    currentPath = getCurrentPath();
    for (const route of routes) {
      const match = matchRoute(route.path, currentPath);
      if (match) {
        matchedRoute = route;
        routeParams = match.params;
        return;
      }
    }
    // No match — try wildcard '*'
    const wildcardRoute = routes.find((r) => r.path === '*');
    if (wildcardRoute) {
      matchedRoute = wildcardRoute;
      routeParams = {};
    } else {
      matchedRoute = null;
      routeParams = {};
    }
  }

  // Initial route
  updateRoute();

  function onHashChange() {
    updateRoute();
  }

  onMount(() => {
    window.addEventListener('hashchange', onHashChange);
  });

  onDestroy(() => {
    window.removeEventListener('hashchange', onHashChange);
  });
</script>

{#if matchedRoute}
  {@const Comp = matchedRoute.component}
  <Comp params={routeParams} />
{:else if fallback}
  {@const Fallback = fallback}
  <Fallback />
{:else}
  <div class="flex items-center justify-center h-full text-text-muted text-sm">
    Route not found: {currentPath}
  </div>
{/if}
