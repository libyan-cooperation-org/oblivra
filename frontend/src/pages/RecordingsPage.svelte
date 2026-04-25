<!--
  OBLIVRA — Session Recordings (Svelte 5)
  Live from RecordingService — real audit-trail session replay library.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  interface RecordingMeta {
    ID: string;
    SessionID: string;
    HostLabel: string;
    StartedAt: string;
    EndedAt: string;
    DurationSecs: number;
    SizeBytes: number;
    FrameCount: number;
  }

  let recordings = $state<RecordingMeta[]>([]);
  let loading    = $state(false);
  let searchQ    = $state('');

  const filtered = $derived(
    searchQ.trim()
      ? recordings.filter(r =>
          r.HostLabel?.toLowerCase().includes(searchQ.toLowerCase()) ||
          r.ID?.toLowerCase().includes(searchQ.toLowerCase())
        )
      : recordings
  );

  const totalSizeMB = $derived(
    recordings.reduce((acc, r) => acc + (r.SizeBytes || 0), 0) / (1024 * 1024)
  );

  function fmtDuration(secs: number): string {
    if (!secs) return '—';
    const m = Math.floor(secs / 60), s = secs % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
  }

  function fmtSize(bytes: number): string {
    if (!bytes) return '—';
    if (bytes < 1024)       return `${bytes} B`;
    if (bytes < 1048576)    return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }

  function fmtDate(iso: string): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString();
  }

  async function load() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      const { ListRecordings } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice');
      recordings = ((await ListRecordings()) || []) as RecordingMeta[];
    } catch (e: any) {
      appStore.notify('Failed to load recordings', 'error', e?.message);
    } finally { loading = false; }
  }

  async function remove(id: string) {
    if (!confirm('Delete recording?')) return;
    try {
      const { DeleteRecording } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice');
      await DeleteRecording(id);
      await load();
      appStore.notify('Recording deleted', 'info');
    } catch (e: any) {
      appStore.notify('Delete failed', 'error', e?.message);
    }
  }

  function replay(id: string) {
    sessionStorage.setItem('oblivra:nav_params', JSON.stringify({ id }));
    push('/session-playback');
  }

  onMount(load);
</script>

<PageLayout title="Session Recordings" subtitle="Immutable terminal audit replays">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQ} placeholder="Search recordings…" compact />
    <Button variant="secondary" size="sm" onclick={load}>
      {loading ? '⟳' : '↺'} Refresh
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Total Recordings" value={recordings.length}             trend="stable"  />
      <KPI label="Storage Used"     value={`${totalSizeMB.toFixed(1)} MB`} trend="stable"  />
      <KPI label="Mode"             value={IS_BROWSER ? 'Browser (read-only)' : 'Desktop'} variant="accent" />
    </div>

    {#if IS_BROWSER}
      <EmptyState title="Desktop feature" description="Session recordings are stored by the desktop binary." icon="🔒" />
    {:else if loading && recordings.length === 0}
      <div class="text-[11px] text-text-muted font-mono p-4 animate-pulse">Loading recordings…</div>
    {:else if filtered.length === 0}
      <EmptyState
        title={searchQ ? 'No matches' : 'No recordings yet'}
        description={searchQ ? 'Try a different search.' : 'Enable session recording in Settings to start capturing.'}
        icon="🎥"
      />
    {:else}
      <div class="flex-1 min-h-0 overflow-auto bg-surface-1 border border-border-primary rounded-sm">
        <table class="w-full text-left min-w-[600px]">
          <thead class="sticky top-0">
            <tr class="bg-surface-2 border-b border-border-primary text-[9px] font-bold uppercase tracking-widest text-text-muted">
              <th class="px-3 py-2">Date</th>
              <th class="px-3 py-2">Host</th>
              <th class="px-3 py-2 w-24">Duration</th>
              <th class="px-3 py-2 w-24">Size</th>
              <th class="px-3 py-2 w-24">Frames</th>
              <th class="px-3 py-2 text-right w-28">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filtered as r (r.ID)}
              <tr class="border-b border-border-primary hover:bg-surface-2/50 transition-colors group">
                <td class="px-3 py-2 font-mono text-[11px] text-text-muted tabular-nums">{fmtDate(r.StartedAt)}</td>
                <td class="px-3 py-2">
                  <div class="flex items-center gap-2">
                    <span class="text-accent opacity-60">▶</span>
                    <div class="flex flex-col">
                      <span class="text-[11px] font-bold text-text-heading">{r.HostLabel || 'Unknown host'}</span>
                      <span class="text-[9px] text-text-muted font-mono">{r.ID?.slice(0, 8)}</span>
                    </div>
                  </div>
                </td>
                <td class="px-3 py-2 font-mono text-[11px] text-text-muted">{fmtDuration(r.DurationSecs)}</td>
                <td class="px-3 py-2 font-mono text-[11px] text-text-muted">{fmtSize(r.SizeBytes)}</td>
                <td class="px-3 py-2 text-[11px] text-text-muted tabular-nums">{r.FrameCount ?? '—'}</td>
                <td class="px-3 py-2">
                  <div class="flex justify-end gap-1">
                    <Button variant="primary" size="xs" onclick={() => replay(r.ID)}>Replay</Button>
                    <Button variant="danger"  size="xs" onclick={() => remove(r.ID)}>✕</Button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</PageLayout>
