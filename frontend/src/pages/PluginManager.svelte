<!--
  Plugin Manager — bound to PluginService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, PopOutButton } from '@components/ui';
  import { Puzzle, RefreshCw, Power, PowerOff } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Plugin = { id?: string; name?: string; description?: string; version?: string; active?: boolean; kind?: string };
  let plugins = $state<Plugin[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { GetPlugins } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/pluginservice');
      plugins = ((await GetPlugins()) ?? []) as Plugin[];
    } catch (e: any) {
      appStore.notify(`Plugin load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function toggle(p: Plugin) {
    if (!p.id) return;
    try {
      const { Activate, Deactivate } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/pluginservice');
      if (p.active) await Deactivate(p.id);
      else await Activate(p.id);
      appStore.notify(`${p.name} ${p.active ? 'deactivated' : 'activated'}`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Toggle failed: ${e?.message ?? e}`, 'error');
    }
  }

  async function reload() {
    try {
      const { Refresh } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/pluginservice');
      await Refresh();
      appStore.notify('Plugins reloaded', 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Reload failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let stats = $derived({
    total: plugins.length,
    active: plugins.filter((p) => p.active).length,
  });
</script>

<PageLayout title="Plugin Manager" subtitle="Third-party integrations and modules">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={reload}>Reload</Button>
    <Button variant="primary" size="sm" onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/plugins" title="Plugins" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Installed" value={stats.total.toString()} variant="accent" />
      <KPI label="Active" value={stats.active.toString()} variant={stats.active > 0 ? 'success' : 'muted'} />
      <KPI label="Inactive" value={(stats.total - stats.active).toString()} variant="muted" />
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 overflow-auto flex-1 min-h-0">
      {#each plugins as p (p.id ?? p.name)}
        <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
          <div class="flex items-start gap-2">
            <Puzzle size={14} class="text-accent shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="font-bold text-[12px] truncate">{p.name ?? p.id}</div>
              {#if p.version}<div class="text-[9px] font-mono text-text-muted">v{p.version}</div>{/if}
            </div>
            <Badge variant={p.active ? 'success' : 'muted'} size="xs">{p.active ? 'on' : 'off'}</Badge>
          </div>
          {#if p.description}<div class="text-[10px] text-text-muted line-clamp-3">{p.description}</div>{/if}
          <div class="mt-auto pt-2">
            <Button variant={p.active ? 'ghost' : 'cta'} size="sm" onclick={() => toggle(p)}>
              {#if p.active}<PowerOff size={11} class="mr-1" />Deactivate{:else}<Power size={11} class="mr-1" />Activate{/if}
            </Button>
          </div>
        </div>
      {:else}
        <div class="md:col-span-3 text-center text-sm text-text-muted py-12">{loading ? 'Loading…' : 'No plugins installed.'}</div>
      {/each}
    </div>
  </div>
</PageLayout>
