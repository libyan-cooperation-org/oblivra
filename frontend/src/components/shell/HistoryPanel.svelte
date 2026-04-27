<!--
  HistoryPanel — searchable command history backed by OBLIVRA's
  CommandHistoryService. Adapted from Blacknode's HistoryPanel concept;
  bindings adjusted for OBLIVRA's API:
    Blacknode HistoryService.List(hostID, source, limit) → CommandHistoryService.GetGlobalHistory(limit)
    Blacknode HistoryService.Search(query) → SearchHistory(hostID, prefix) (per-host)
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Search, Trash2, Clipboard, RefreshCw, History } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';

  type Entry = {
    id?: string;
    host_id?: string;
    command: string;
    timestamp?: string;
  };

  let entries = $state<Entry[]>([]);
  let filter = $state('');
  let loading = $state(false);

  let visible = $derived(
    !filter
      ? entries
      : entries.filter((e) => e.command.toLowerCase().includes(filter.toLowerCase())),
  );

  async function load() {
    loading = true;
    try {
      const { GetGlobalHistory } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/commandhistoryservice'
      );
      const list = ((await GetGlobalHistory(500)) ?? []) as Entry[];
      entries = list;
    } catch (e: any) {
      toastStore.add({ type: 'warning', title: 'History load failed', message: e?.message ?? String(e) });
    } finally {
      loading = false;
    }
  }

  async function clearAll() {
    if (!confirm('Clear ALL command history? This cannot be undone.')) return;
    try {
      const { ClearHistory } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/commandhistoryservice'
      );
      await ClearHistory(''); // empty hostID = all hosts
      entries = [];
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Clear failed', message: e?.message ?? String(e) });
    }
  }

  function copy(cmd: string) {
    if (navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(cmd);
      toastStore.add({ type: 'success', title: 'Copied to clipboard', message: cmd });
    }
  }

  function injectIntoTerminal(cmd: string) {
    const sid = shellStore.activeSessionID;
    if (!sid) {
      toastStore.add({ type: 'warning', title: 'No active shell', message: 'Open a shell tab first.' });
      return;
    }
    shellStore.insertIntoTerminal(sid, cmd);
  }

  onMount(load);
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <History size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">History</span>
    <span class="text-[10px] text-[var(--tx3)]">· {visible.length} / {entries.length}</span>

    <div class="relative ml-3 flex-1 max-w-md">
      <Search size={12} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--tx3)]" />
      <input
        class="w-full rounded-md border border-[var(--b1)] bg-[var(--s2)] py-1 pl-7 pr-2 text-xs outline-none placeholder:text-[var(--tx3)] focus:border-cyan-400/40"
        placeholder="Filter commands…"
        bind:value={filter}
      />
    </div>

    <div class="ml-auto flex items-center gap-1">
      <button
        class="flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={load}
      >
        <RefreshCw size={11} class={loading ? 'animate-spin' : ''} />
        Refresh
      </button>
      <button
        class="flex items-center gap-1 rounded-md border border-rose-400/30 bg-rose-400/10 px-2 py-1 text-[11px] text-rose-300 hover:bg-rose-400/20"
        onclick={clearAll}
      >
        <Trash2 size={11} />
        Clear all
      </button>
    </div>
  </header>

  <div class="min-h-0 flex-1 overflow-y-auto">
    {#if visible.length === 0 && !loading}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        {entries.length === 0 ? 'No commands recorded yet.' : 'No matches for the current filter.'}
      </div>
    {/if}
    <ul class="divide-y divide-[var(--b1)]">
      {#each visible as e, i (e.id ?? `${e.timestamp}-${i}`)}
        <li class="group flex items-center gap-3 px-3 py-1.5 hover:bg-[var(--s1)]">
          <span class="font-mono text-[10px] text-[var(--tx3)]">
            {e.timestamp ? new Date(e.timestamp).toLocaleString() : ''}
          </span>
          {#if e.host_id}
            <span class="rounded-sm bg-[var(--s3)] px-1.5 py-0.5 font-mono text-[9px] uppercase tracking-wider text-[var(--tx2)]">
              {e.host_id.slice(0, 6)}
            </span>
          {/if}
          <code class="min-w-0 flex-1 truncate font-mono text-xs text-[var(--tx)]">{e.command}</code>
          <div class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
            <button
              class="rounded p-1 text-[var(--tx3)] hover:bg-[var(--s3)] hover:text-cyan-300"
              title="Copy"
              onclick={() => copy(e.command)}
            >
              <Clipboard size={11} />
            </button>
            <button
              class="rounded border border-cyan-400/30 bg-cyan-400/10 px-1.5 py-0.5 text-[10px] text-cyan-200 hover:bg-cyan-400/20"
              onclick={() => injectIntoTerminal(e.command)}
              title="Insert into active shell"
            >
              ↳ inject
            </button>
          </div>
        </li>
      {/each}
    </ul>
  </div>
</div>
