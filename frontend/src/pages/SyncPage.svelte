<!--
  Sync — bound to SyncService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let lastSync = $state<Date | null>(null);
  let syncing = $state(false);

  async function sync() {
    syncing = true;
    try {
      if (IS_BROWSER) { appStore.notify('Sync only in desktop', 'warning'); return; }
      const { Sync } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/syncservice');
      await Sync();
      lastSync = new Date();
      appStore.notify('Sync complete', 'success');
    } catch (e: any) {
      appStore.notify(`Sync failed: ${e?.message ?? e}`, 'error');
    } finally { syncing = false; }
  }

  onMount(() => { void sync(); });
</script>

<PageLayout title="Sync" subtitle="Cross-instance synchronization">
  {#snippet toolbar()}
    <Button variant="primary" size="sm" icon={RefreshCw} onclick={sync} disabled={syncing}>{syncing ? 'Syncing…' : 'Sync now'}</Button>
    <PopOutButton route="/sync" title="Sync" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Last Sync" value={lastSync ? lastSync.toLocaleTimeString() : '—'} variant={lastSync ? 'success' : 'muted'} />
      <KPI label="Status" value={syncing ? 'In progress' : (lastSync ? 'Idle' : 'Never run')} variant={syncing ? 'accent' : 'muted'} />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser (no sync)' : 'Desktop'} variant="muted" />
    </div>

    <div class="bg-surface-1 border border-border-primary rounded-md p-6 text-sm text-text-muted">
      <p class="mb-2">SyncService coordinates encrypted state replication between this OBLIVRA instance and its peers (e.g., a backup HQ + an on-call laptop).</p>
      <p>Conflicts are surfaced via <code class="bg-surface-2 px-1 rounded">QueueUpdate</code> / <code class="bg-surface-2 px-1 rounded">ResolveConflict</code> RPCs; the conflict-resolution UI is part of the next phase. For now this page exposes the manual trigger.</p>
    </div>
  </div>
</PageLayout>
