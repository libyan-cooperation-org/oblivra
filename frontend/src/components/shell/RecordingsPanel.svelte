<!--
  RecordingsPanel — list / play / delete past terminal session recordings.
  Bound to OBLIVRA's RecordingService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Film, Trash2, RefreshCw, Play, Search } from 'lucide-svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { push } from '@lib/router.svelte';

  type Recording = {
    id: string;
    title?: string;
    host_label?: string;
    started_at?: string;
    ended_at?: string;
    duration_seconds?: number;
    size_bytes?: number;
  };

  let items = $state<Recording[]>([]);
  let filter = $state('');
  let loading = $state(false);

  let visible = $derived(
    !filter
      ? items
      : items.filter((r) =>
          (r.title ?? '').toLowerCase().includes(filter.toLowerCase()) ||
          (r.host_label ?? '').toLowerCase().includes(filter.toLowerCase()),
        ),
  );

  async function load() {
    loading = true;
    try {
      const { ListRecordings } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice'
      );
      const list = ((await ListRecordings()) ?? []) as Recording[];
      items = list;
    } catch (e: any) {
      toastStore.add({ type: 'warning', title: 'Recordings load failed', message: e?.message ?? String(e) });
    } finally {
      loading = false;
    }
  }

  async function del(id: string) {
    if (!confirm('Delete this recording? Cannot be undone.')) return;
    try {
      const { DeleteRecording } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice'
      );
      await DeleteRecording(id);
      items = items.filter((i) => i.id !== id);
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Delete failed', message: e?.message ?? String(e) });
    }
  }

  function play(id: string) {
    push(`/session-playback?id=${encodeURIComponent(id)}`);
  }

  function fmtDur(secs?: number) {
    if (!secs) return '—';
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}m${s}s`;
  }
  function fmtSize(b?: number) {
    if (!b) return '—';
    if (b < 1024) return `${b}B`;
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)}KB`;
    return `${(b / 1024 / 1024).toFixed(1)}MB`;
  }

  onMount(load);
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Film size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Recordings</span>
    <span class="text-[10px] text-[var(--tx3)]">· {visible.length} / {items.length}</span>

    <div class="relative ml-3 flex-1 max-w-md">
      <Search size={12} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--tx3)]" />
      <input
        class="w-full rounded-md border border-[var(--b1)] bg-[var(--s2)] py-1 pl-7 pr-2 text-xs outline-none placeholder:text-[var(--tx3)] focus:border-cyan-400/40"
        placeholder="Filter…"
        bind:value={filter}
      />
    </div>

    <button
      class="ml-auto flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
      onclick={load}
    >
      <RefreshCw size={11} class={loading ? 'animate-spin' : ''} />
      Refresh
    </button>
  </header>

  <div class="min-h-0 flex-1 overflow-y-auto">
    {#if visible.length === 0 && !loading}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        {items.length === 0 ? 'No recordings yet. Enable session recording in Settings.' : 'No matches.'}
      </div>
    {/if}
    <table class="w-full text-xs">
      <thead class="sticky top-0 bg-[var(--s1)] text-[10px] uppercase tracking-wider text-[var(--tx3)]">
        <tr>
          <th class="px-3 py-2 text-left">Title</th>
          <th class="px-3 py-2 text-left">Host</th>
          <th class="px-3 py-2 text-left">Started</th>
          <th class="px-3 py-2 text-right">Duration</th>
          <th class="px-3 py-2 text-right">Size</th>
          <th class="px-3 py-2"></th>
        </tr>
      </thead>
      <tbody>
        {#each visible as r (r.id)}
          <tr class="group border-t border-[var(--b1)] hover:bg-[var(--s1)]">
            <td class="px-3 py-2">{r.title ?? r.id.slice(0, 8)}</td>
            <td class="px-3 py-2 font-mono text-[10px] text-[var(--tx2)]">{r.host_label ?? '—'}</td>
            <td class="px-3 py-2 text-[10px] text-[var(--tx3)]">
              {r.started_at ? new Date(r.started_at).toLocaleString() : '—'}
            </td>
            <td class="px-3 py-2 text-right font-mono text-[10px] text-[var(--tx2)]">{fmtDur(r.duration_seconds)}</td>
            <td class="px-3 py-2 text-right font-mono text-[10px] text-[var(--tx2)]">{fmtSize(r.size_bytes)}</td>
            <td class="px-3 py-2">
              <div class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                <button
                  class="rounded p-1 text-cyan-300 hover:bg-cyan-400/10"
                  onclick={() => play(r.id)}
                  title="Play"
                >
                  <Play size={12} />
                </button>
                <button
                  class="rounded p-1 text-rose-300 hover:bg-rose-400/10"
                  onclick={() => del(r.id)}
                  title="Delete"
                >
                  <Trash2 size={12} />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
