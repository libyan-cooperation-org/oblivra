<!--
  OBLIVRA — Notes (Svelte 5)
  Live from NotesService — real vault-backed notes and runbooks.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Button, EmptyState, SearchBar, Badge } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  interface Note {
    ID: string;
    HostID: string;
    Title: string;
    Content: string;
    Category: string;
    UpdatedAt: string;
  }

  let notes   = $state<Note[]>([]);
  let loading = $state(false);
  let searchQ = $state('');
  let showAdd = $state(false);
  let editNote: Note | null = $state(null);

  let fTitle    = $state('');
  let fContent  = $state('');
  let fCategory = $state('General');
  let saving    = $state(false);

  const filtered = $derived(
    searchQ.trim()
      ? notes.filter(n =>
          n.Title?.toLowerCase().includes(searchQ.toLowerCase()) ||
          n.Content?.toLowerCase().includes(searchQ.toLowerCase()) ||
          n.Category?.toLowerCase().includes(searchQ.toLowerCase())
        )
      : notes
  );

  async function load() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      // GetNotes('') returns all notes across all hosts
      const { GetNotes } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/notesservice');
      notes = ((await GetNotes('')) || []) as Note[];
    } catch (e: any) {
      appStore.notify('Failed to load notes', 'error', e?.message);
    } finally { loading = false; }
  }

  async function save() {
    if (!fTitle.trim()) return;
    saving = true;
    try {
      const { SaveNote } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/notesservice');
      await SaveNote({
        ID:       editNote?.ID ?? '',
        HostID:   '',
        Title:    fTitle,
        Content:  fContent,
        Category: fCategory,
        UpdatedAt: new Date().toISOString(),
      } as any);
      fTitle = ''; fContent = ''; fCategory = 'General';
      showAdd = false; editNote = null;
      await load();
      appStore.notify('Note saved', 'success');
    } catch (e: any) {
      appStore.notify('Save failed', 'error', e?.message);
    } finally { saving = false; }
  }

  async function remove(id: string) {
    if (!confirm('Delete note?')) return;
    try {
      const { DeleteNote } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/notesservice');
      await DeleteNote(id);
      await load();
    } catch (e: any) {
      appStore.notify('Delete failed', 'error', e?.message);
    }
  }

  function openEdit(n: Note) {
    editNote = n;
    fTitle    = n.Title;
    fContent  = n.Content;
    fCategory = n.Category || 'General';
    showAdd   = true;
  }

  function timeAgo(iso: string): string {
    if (!iso) return '—';
    const diff = Date.now() - new Date(iso).getTime();
    const m = Math.floor(diff / 60000);
    if (m < 2)    return 'just now';
    if (m < 60)   return `${m}m ago`;
    const h = Math.floor(m / 60);
    if (h < 24)   return `${h}h ago`;
    return `${Math.floor(h / 24)}d ago`;
  }

  onMount(load);
</script>

<PageLayout title="Knowledge Base & Notes" subtitle="Secure investigation notes and runbooks">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQ} placeholder="Search notes…" compact />
    <Button variant="secondary" size="sm" onclick={load}>↺ Refresh</Button>
    <Button variant="primary"   size="sm" onclick={() => { showAdd = !showAdd; editNote = null; fTitle = ''; fContent = ''; fCategory = 'General'; }}>
      + New Note
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Total Notes" value={notes.length}  trend="stable" />
      <KPI label="Categories"  value={[...new Set(notes.map(n => n.Category).filter(Boolean))].length} variant="accent" />
      <KPI label="Mode"        value={IS_BROWSER ? 'Browser (read-only)' : 'Desktop'} variant={IS_BROWSER ? 'muted' : 'success'} />
    </div>

    {#if showAdd}
      <div class="shrink-0 bg-surface-2 border border-border-secondary rounded-md p-4 flex flex-col gap-3">
        <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">
          {editNote ? 'Edit Note' : 'New Note'}
        </span>
        <div class="grid grid-cols-3 gap-2">
          <input class="col-span-2 bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Title" bind:value={fTitle} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Category" bind:value={fCategory} />
          <textarea class="col-span-3 bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary font-mono outline-none focus:border-accent resize-none h-24" placeholder="Content (markdown supported)" bind:value={fContent}></textarea>
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="secondary" size="sm" onclick={() => { showAdd = false; editNote = null; }}>Cancel</Button>
          <Button variant="primary"   size="sm" onclick={save} disabled={saving || !fTitle}>
            {saving ? 'Saving…' : editNote ? 'Update' : 'Save'}
          </Button>
        </div>
      </div>
    {/if}

    {#if IS_BROWSER}
      <EmptyState title="Desktop feature" description="Notes require the OBLIVRA desktop binary." icon="🔒" />
    {:else if loading && notes.length === 0}
      <div class="text-[11px] text-text-muted font-mono p-4 animate-pulse">Loading notes…</div>
    {:else if filtered.length === 0}
      <EmptyState
        title={searchQ ? 'No matches' : 'No notes yet'}
        description={searchQ ? 'Try a different search.' : 'Create your first investigation note above.'}
        icon="🗒️"
      />
    {:else}
      <div class="flex-1 min-h-0 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 overflow-auto content-start">
        {#each filtered as n (n.ID)}
          <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2 group hover:border-accent transition-colors cursor-pointer"
               onclick={() => openEdit(n)} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && openEdit(n)}>
            <div class="flex justify-between items-center">
              <Badge variant="info">{n.Category || 'General'}</Badge>
              <span class="text-[9px] text-text-muted">{timeAgo(n.UpdatedAt)}</span>
            </div>
            <h3 class="text-sm font-bold text-text-heading group-hover:text-accent transition-colors leading-tight">{n.Title}</h3>
            <p class="text-[11px] text-text-muted leading-relaxed line-clamp-3">
              {n.Content || 'No content yet.'}
            </p>
            <div class="flex justify-end mt-auto pt-2 opacity-0 group-hover:opacity-100 transition-opacity">
              <Button variant="danger" size="xs" onclick={(e: Event) => { e.stopPropagation(); remove(n.ID); }}>✕</Button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</PageLayout>
