<!-- Features Page — bound to LicensingService.GetFeatureMap. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, PopOutButton } from '@components/ui';
  import { Award, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';

  let features = $state<Record<string, boolean>>({});
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { GetFeatureMap } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/licensingservice');
      features = ((await GetFeatureMap()) ?? {}) as Record<string, boolean>;
    } finally { loading = false; }
  }
  onMount(refresh);

  let entries = $derived(Object.entries(features).sort());
  let enabled = $derived(entries.filter(([_, v]) => v).length);
</script>

<PageLayout title="Features & Capabilities" subtitle="Per-tenant feature flag map">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/features" title="Features" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Total Features" value={entries.length.toString()} variant="accent" />
      <KPI label="Enabled" value={enabled.toString()} variant="success" />
      <KPI label="Disabled" value={(entries.length - enabled).toString()} variant="muted" />
    </div>
    <div class="flex-1 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 overflow-auto">
      {#each entries as [name, on]}
        <div class="bg-surface-1 border border-border-primary rounded-md p-3 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Award size={11} class="text-accent" />
            <span class="font-mono text-[11px]">{name}</span>
          </div>
          <Badge variant={on ? 'success' : 'muted'} size="xs">{on ? 'on' : 'off'}</Badge>
        </div>
      {:else}
        <div class="md:col-span-3 text-center text-sm text-text-muted py-8">{loading ? 'Loading…' : 'No feature flags yet.'}</div>
      {/each}
    </div>
  </div>
</PageLayout>
