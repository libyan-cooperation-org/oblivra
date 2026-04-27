<!--
  SnippetsPanel — list / create / inject command snippets.
  Bound to OBLIVRA's SnippetService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Bookmark, Trash2, RefreshCw, Plus, Search, Send, ExternalLink } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { push } from '@lib/router.svelte';

  type Snippet = {
    id: string;
    title: string;
    command: string;
    description?: string;
    tags?: string[];
    variables?: string[];
  };

  let items = $state<Snippet[]>([]);
  let filter = $state('');
  let loading = $state(false);

  // Inline create form state.
  let showAdd = $state(false);
  let fTitle = $state('');
  let fCmd = $state('');
  let fDesc = $state('');
  let fTags = $state('');

  let visible = $derived(
    !filter
      ? items
      : items.filter(
          (s) =>
            s.title.toLowerCase().includes(filter.toLowerCase()) ||
            s.command.toLowerCase().includes(filter.toLowerCase()) ||
            (s.tags ?? []).some((t) => t.toLowerCase().includes(filter.toLowerCase())),
        ),
  );

  async function load() {
    loading = true;
    try {
      const { List } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice'
      );
      items = ((await List()) ?? []) as Snippet[];
    } catch (e: any) {
      toastStore.add({ type: 'warning', title: 'Snippets load failed', message: e?.message ?? String(e) });
    } finally {
      loading = false;
    }
  }

  async function save() {
    if (!fTitle.trim() || !fCmd.trim()) {
      toastStore.add({ type: 'warning', title: 'Title and command required' });
      return;
    }
    try {
      const { Create } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice'
      );
      const tags = fTags.split(',').map((t) => t.trim()).filter(Boolean);
      await Create(fTitle, fCmd, fDesc, tags, []);
      fTitle = '';
      fCmd = '';
      fDesc = '';
      fTags = '';
      showAdd = false;
      await load();
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Save failed', message: e?.message ?? String(e) });
    }
  }

  async function del(id: string) {
    if (!confirm('Delete snippet?')) return;
    try {
      const { Delete } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice'
      );
      await Delete(id);
      items = items.filter((s) => s.id !== id);
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Delete failed', message: e?.message ?? String(e) });
    }
  }

  function inject(s: Snippet) {
    const sid = shellStore.activeSessionID;
    if (!sid) {
      toastStore.add({ type: 'warning', title: 'No active shell', message: 'Open a shell tab first.' });
      return;
    }
    shellStore.insertIntoTerminal(sid, s.command);
    toastStore.add({ type: 'success', title: 'Snippet sent', message: s.title });
  }

  onMount(load);
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Bookmark size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Snippets</span>
    <span class="text-[10px] text-[var(--tx3)]">· {visible.length} / {items.length}</span>

    <div class="relative ml-3 flex-1 max-w-md">
      <Search size={12} class="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--tx3)]" />
      <input
        class="w-full rounded-md border border-[var(--b1)] bg-[var(--s2)] py-1 pl-7 pr-2 text-xs outline-none placeholder:text-[var(--tx3)] focus:border-cyan-400/40"
        placeholder="Filter…"
        bind:value={filter}
      />
    </div>

    <div class="ml-auto flex items-center gap-1">
      <button
        class="flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={() => push('/snippets')}
      >
        <ExternalLink size={11} />
        Full editor
      </button>
      <button
        class="flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={load}
      >
        <RefreshCw size={11} class={loading ? 'animate-spin' : ''} />
        Refresh
      </button>
      <button
        class="flex items-center gap-1 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-2 py-1 text-[11px] text-cyan-200 hover:bg-cyan-400/20"
        onclick={() => (showAdd = !showAdd)}
      >
        <Plus size={11} />
        New
      </button>
    </div>
  </header>

  {#if showAdd}
    <div class="border-b border-[var(--b1)] bg-[var(--s1)] p-3">
      <div class="grid grid-cols-2 gap-2">
        <input class="col-span-2 rounded border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 text-xs outline-none" placeholder="Title" bind:value={fTitle} />
        <input class="col-span-2 rounded border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 font-mono text-xs outline-none" placeholder="Command  e.g. docker ps -a" bind:value={fCmd} />
        <input class="rounded border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 text-xs outline-none" placeholder="Description (optional)" bind:value={fDesc} />
        <input class="rounded border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 text-xs outline-none" placeholder="Tags (comma-separated)" bind:value={fTags} />
      </div>
      <div class="mt-2 flex justify-end gap-2">
        <button
          class="rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)]"
          onclick={() => (showAdd = false)}
        >Cancel</button>
        <button
          class="rounded-md border border-cyan-400/40 bg-cyan-400/10 px-2 py-1 text-[11px] text-cyan-200 hover:bg-cyan-400/20"
          onclick={save}
        >Save</button>
      </div>
    </div>
  {/if}

  <div class="min-h-0 flex-1 overflow-y-auto p-3">
    {#if visible.length === 0 && !loading}
      <div class="py-12 text-center text-sm text-[var(--tx3)]">
        {items.length === 0 ? 'No snippets yet — click "+ New" to add one.' : 'No matches for the current filter.'}
      </div>
    {/if}
    <div class="grid grid-cols-1 gap-2 lg:grid-cols-2 xl:grid-cols-3">
      {#each visible as s (s.id)}
        <div class="group flex flex-col gap-1 rounded-md border border-[var(--b1)] bg-[var(--s1)] p-3 hover:border-cyan-400/30">
          <div class="flex items-start gap-2">
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-semibold">{s.title}</div>
              {#if s.description}
                <div class="truncate text-[10px] text-[var(--tx3)]">{s.description}</div>
              {/if}
            </div>
            <div class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
              <button
                class="rounded border border-cyan-400/30 bg-cyan-400/10 px-1.5 py-0.5 text-[10px] text-cyan-200 hover:bg-cyan-400/20"
                onclick={() => inject(s)}
                title="Inject into active shell"
              >
                <Send size={10} class="-mt-0.5 mr-0.5 inline" />inject
              </button>
              <button
                class="rounded p-1 text-rose-300 hover:bg-rose-400/10"
                onclick={() => del(s.id)}
                title="Delete"
              >
                <Trash2 size={11} />
              </button>
            </div>
          </div>
          <code class="block overflow-x-auto rounded bg-[var(--s0)] px-2 py-1 font-mono text-[11px] text-cyan-100">{s.command}</code>
          {#if s.tags && s.tags.length}
            <div class="flex flex-wrap gap-1">
              {#each s.tags as t}
                <span class="rounded-sm bg-[var(--s3)] px-1.5 py-0.5 text-[9px] uppercase tracking-wider text-[var(--tx2)]">{t}</span>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  </div>
</div>
