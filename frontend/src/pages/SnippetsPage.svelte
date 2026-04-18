<!--
  OBLIVRA — Snippet Vault (Svelte 5)
  Live from SnippetService — real vault-backed command library.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  interface Snippet {
    ID: string;
    Title: string;
    Command: string;
    Description: string;
    Tags: string[];
    Variables: string[];
    UseCount: number;
    CreatedAt: string;
  }

  let snippets  = $state<Snippet[]>([]);
  let loading   = $state(false);
  let searchQ   = $state('');
  let showAdd   = $state(false);

  // Add form
  let fTitle = $state('');
  let fCmd   = $state('');
  let fDesc  = $state('');
  let fTags  = $state('');
  let saving = $state(false);

  const filtered = $derived(
    searchQ.trim()
      ? snippets.filter(s =>
          s.Title?.toLowerCase().includes(searchQ.toLowerCase()) ||
          s.Command?.toLowerCase().includes(searchQ.toLowerCase()) ||
          (s.Tags || []).some(t => t.toLowerCase().includes(searchQ.toLowerCase()))
        )
      : snippets
  );

  async function load() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      const { List } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice');
      snippets = ((await List()) || []) as Snippet[];
    } catch (e: any) {
      appStore.notify('Failed to load snippets', 'error', e?.message);
    } finally { loading = false; }
  }

  async function save() {
    if (!fTitle || !fCmd) return;
    saving = true;
    try {
      const { Create } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice');
      const tags = fTags.split(',').map(t => t.trim()).filter(Boolean);
      await Create(fTitle, fCmd, fDesc, tags, []);
      fTitle = ''; fCmd = ''; fDesc = ''; fTags = '';
      showAdd = false;
      await load();
      appStore.notify('Snippet saved', 'success');
    } catch (e: any) {
      appStore.notify('Save failed', 'error', e?.message);
    } finally { saving = false; }
  }

  async function remove(id: string) {
    if (!confirm('Delete snippet?')) return;
    try {
      const { Delete } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice');
      await Delete(id);
      await load();
    } catch (e: any) {
      appStore.notify('Delete failed', 'error', e?.message);
    }
  }

  async function run(id: string) {
    const sessionId = appStore.activeSessionId;
    if (!sessionId) { appStore.notify('No active terminal session', 'warning', 'Open a shell first.'); return; }
    try {
      const { ExecuteSnippet } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/snippetservice');
      await ExecuteSnippet(id, sessionId, {}, false);
      appStore.notify('Snippet executed', 'success');
    } catch (e: any) {
      appStore.notify('Execute failed', 'error', e?.message);
    }
  }

  onMount(load);
</script>

<PageLayout title="Snippet Vault" subtitle="Execute atomic operations across your fleet with pre-validated commands">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQ} placeholder="Filter snippets…" compact />
    <Button variant="secondary" size="sm" onclick={load}>↺ Refresh</Button>
    <Button variant="primary"   size="sm" onclick={() => showAdd = !showAdd}>+ Create</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Saved Snippets"  value={snippets.length}  trend="stable" />
      <KPI label="Active Session"  value={appStore.activeSessionId ? 'Ready' : 'None'} variant={appStore.activeSessionId ? 'success' : 'muted'} />
      <KPI label="Mode"            value={IS_BROWSER ? 'Browser (read-only)' : 'Desktop'} variant="accent" />
    </div>

    {#if showAdd}
      <div class="shrink-0 bg-surface-2 border border-border-secondary rounded-md p-4 flex flex-col gap-3">
        <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">New Snippet</span>
        <div class="grid grid-cols-2 gap-2">
          <input class="col-span-2 bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Title" bind:value={fTitle} />
          <input class="col-span-2 bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary font-mono outline-none focus:border-accent" placeholder="Command  e.g. docker ps -a" bind:value={fCmd} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Description (optional)" bind:value={fDesc} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Tags (comma-separated)" bind:value={fTags} />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="secondary" size="sm" onclick={() => showAdd = false}>Cancel</Button>
          <Button variant="primary"   size="sm" onclick={save} disabled={saving || !fTitle || !fCmd}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>
    {/if}

    {#if IS_BROWSER}
      <EmptyState title="Desktop feature" description="Snippet vault requires the OBLIVRA desktop binary." icon="🔒" />
    {:else if loading && snippets.length === 0}
      <div class="text-[11px] text-text-muted font-mono p-4 animate-pulse">Loading snippets…</div>
    {:else if filtered.length === 0}
      <EmptyState
        title={searchQ ? 'No matches' : 'No snippets yet'}
        description={searchQ ? 'Try a different search term.' : 'Create your first reusable command above.'}
        icon="📝"
      />
    {:else}
      <div class="flex-1 min-h-0 overflow-auto bg-surface-1 border border-border-primary rounded-sm">
        <table class="w-full text-left">
          <thead class="sticky top-0">
            <tr class="bg-surface-2 border-b border-border-primary text-[9px] font-bold uppercase tracking-widest text-text-muted">
              <th class="px-3 py-2">Title</th>
              <th class="px-3 py-2">Command</th>
              <th class="px-3 py-2 w-32">Tags</th>
              <th class="px-3 py-2 w-24 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filtered as s (s.ID)}
              <tr class="border-b border-border-primary hover:bg-surface-2/50 transition-colors group">
                <td class="px-3 py-2">
                  <div class="flex flex-col">
                    <span class="text-[11px] font-bold text-text-heading">{s.Title}</span>
                    {#if s.Description}<span class="text-[9px] text-text-muted">{s.Description}</span>{/if}
                  </div>
                </td>
                <td class="px-3 py-2">
                  <code class="text-[10px] bg-surface-2 px-2 py-0.5 rounded border border-border-primary font-mono text-accent truncate max-w-xs block">
                    {s.Command}
                  </code>
                </td>
                <td class="px-3 py-2">
                  <div class="flex gap-1 flex-wrap">
                    {#each (s.Tags || []) as tag}
                      <Badge variant="muted">{tag}</Badge>
                    {/each}
                  </div>
                </td>
                <td class="px-3 py-2">
                  <div class="flex justify-end gap-1">
                    <Button variant="primary" size="xs" onclick={() => run(s.ID)}   title="Run in active session">▶</Button>
                    <Button variant="danger"  size="xs" onclick={() => remove(s.ID)} title="Delete">✕</Button>
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
