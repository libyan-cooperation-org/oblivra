<!--
  License page — bound to LicensingService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, PopOutButton } from '@components/ui';
  import { Award, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let status = $state<any>(null);
  let features = $state<Record<string, boolean>>({});
  let agents = $state(0);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/licensingservice');
      const s = await svc.GetLicenseStatus();
      status = s;
      const fm = await svc.GetFeatureMap();
      features = (fm ?? {}) as Record<string, boolean>;
      const c = await svc.ActiveAgentCount();
      agents = (c as number) ?? 0;
    } catch (e: any) {
      appStore.notify(`License load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function activate() {
    const token = prompt('Paste license token:'); if (!token) return;
    try {
      const { ActivateLicense } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/licensingservice');
      await ActivateLicense(token);
      appStore.notify('License activated', 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Activation failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let featureList = $derived(Object.entries(features));
  let enabledCount = $derived(featureList.filter(([_, v]) => v).length);
</script>

<PageLayout title="License & Entitlements" subtitle="Active capabilities and seat usage">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={activate}>Activate License</Button>
    <PopOutButton route="/license" title="License" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
      <KPI label="Tier"          value={status?.tier ?? status?.plan ?? '—'} variant="accent" />
      <KPI label="Status"        value={status?.status ?? (status?.active ? 'Active' : 'Pending')} variant={status?.active ? 'success' : 'warning'} />
      <KPI label="Agents in Use" value={agents.toString()} sublabel={status?.max_agents ? `${status.max_agents} cap` : ''} />
      <KPI label="Features On"   value={`${enabledCount}/${featureList.length}`} variant="muted" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Award size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Feature Map</span>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 p-4 overflow-auto">
        {#each featureList as [name, enabled] (name)}
          <div class="flex items-center justify-between bg-surface-2 border border-border-primary rounded p-2">
            <span class="text-[11px] font-mono">{name}</span>
            <Badge variant={enabled ? 'success' : 'muted'} size="xs">{enabled ? 'on' : 'off'}</Badge>
          </div>
        {:else}
          <div class="md:col-span-3 text-center text-sm text-text-muted py-8">{loading ? 'Loading…' : 'No feature flags reported.'}</div>
        {/each}
      </div>
    </div>
  </div>
</PageLayout>
