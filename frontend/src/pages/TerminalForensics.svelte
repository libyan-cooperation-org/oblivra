<!-- Terminal Forensics — search session recordings + jump to playback. Uses RecordingService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, DataTable, PopOutButton } from '@components/ui';
  import { Microscope, RefreshCw, Search, Play } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let recordings = $state<any[]>([]);
  let query = $state('');
  let hits = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListRecordings } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice');
      recordings = ((await ListRecordings()) ?? []) as any[];
    } finally { loading = false; }
  }

  async function search() {
    if (!query.trim()) { hits = []; return; }
    try {
      const { SearchRecordings } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice');
      hits = ((await SearchRecordings(query)) ?? []) as any[];
    } catch (e: any) {
      appStore.notify(`Search failed: ${e?.message ?? e}`, 'error');
    }
  }
  onMount(refresh);
</script>

<PageLayout title="Terminal Forensics" subtitle="Search and replay recorded shell sessions">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/terminal-forensics" title="Terminal Forensics" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Recorded Sessions" value={recordings.length.toString()} variant="accent" />
      <KPI label="Search Hits" value={hits.length.toString()} variant={hits.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser' : 'Desktop'} variant="muted" />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-3 flex items-center gap-2">
      <Search size={14} class="text-text-muted" />
      <input class="flex-1 bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs outline-none focus:border-accent" placeholder="Search across all session content (e.g. 'sudo rm')" bind:value={query} onkeydown={(e) => e.key === 'Enter' && search()} />
      <Button variant="cta" size="sm" onclick={search}>Search</Button>
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Microscope size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">{hits.length > 0 ? 'Search Hits' : 'All Recordings'}</span>
      </div>
      <DataTable data={hits.length > 0 ? hits : recordings} columns={[
        { key: 'title',    label: 'Title' },
        { key: 'host_label', label: 'Host', width: '140px' },
        { key: 'started_at', label: 'When', width: '160px' },
        { key: 'play',     label: '',     width: '60px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'play'}<Button variant="ghost" size="xs" onclick={() => push(`/session-playback?id=${row.id}`)}><Play size={11} /></Button>
          {:else if col.key === 'started_at'}<span class="font-mono text-[10px] text-text-muted">{(row.started_at ?? '').slice(0, 19)}</span>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</PageLayout>
