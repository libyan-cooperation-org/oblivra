<!--
  OBLIVRA — SSH Bookmarks (Svelte 5)
  Live data from BookmarkService — real vault-backed SSH host management.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, PageLayout, Button, SearchBar, EmptyState } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  interface BookmarkEntry {
    ID: string;
    Label: string;
    Hostname: string;
    Port: number;
    Username: string;
    AuthMethod: string;
    Tags: string[];
    IsFavorite: boolean;
    LastConnected?: string;
    ConnectionCount: number;
  }

  let bookmarks = $state<BookmarkEntry[]>([]);
  let loading    = $state(false);
  let searchQuery = $state('');
  let showAdd    = $state(false);

  // add-form fields
  let fLabel = $state('');
  let fHost  = $state('');
  let fPort  = $state(22);
  let fUser  = $state('');
  let fPass  = $state('');
  let saving = $state(false);

  const filtered = $derived(
    searchQuery.trim()
      ? bookmarks.filter(b =>
          b.Label?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          b.Hostname?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          (b.Tags || []).some(t => t.toLowerCase().includes(searchQuery.toLowerCase()))
        )
      : bookmarks
  );

  const favoriteCount = $derived(bookmarks.filter(b => b.IsFavorite).length);

  // ── Backend calls ──────────────────────────────────────────────────────────

  async function load() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      const { ListAll } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/bookmarkservice');
      bookmarks = ((await ListAll()) || []) as BookmarkEntry[];
    } catch (e: any) {
      appStore.notify('Failed to load bookmarks', 'error', e?.message);
    } finally {
      loading = false;
    }
  }

  async function save() {
    if (!fLabel.trim() || !fHost.trim() || !fUser.trim()) return;
    if (IS_BROWSER) { appStore.notify('Requires desktop app', 'error'); return; }
    saving = true;
    try {
      const { Create } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/bookmarkservice');
      await Create({
        Label: fLabel, Hostname: fHost, Port: fPort,
        Username: fUser, Password: fPass,
        AuthMethod: fPass ? 'password' : 'key',
        Tags: [], IsFavorite: false,
      });
      fLabel = ''; fHost = ''; fUser = ''; fPass = ''; fPort = 22;
      showAdd = false;
      await load();
      appStore.notify('Bookmark saved', 'success');
    } catch (e: any) {
      appStore.notify('Save failed', 'error', e?.message);
    } finally {
      saving = false;
    }
  }

  async function remove(id: string) {
    if (IS_BROWSER || !confirm('Delete this bookmark?')) return;
    try {
      const { Delete } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/bookmarkservice');
      await Delete(id);
      await load();
      appStore.notify('Removed', 'info');
    } catch (e: any) {
      appStore.notify('Delete failed', 'error', e?.message);
    }
  }

  async function toggleFav(id: string) {
    if (IS_BROWSER) return;
    try {
      const { ToggleFavorite } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/bookmarkservice');
      await ToggleFavorite(id);
      await load();
    } catch {}
  }

  function connect(id: string) { appStore.connectToHost(id); }

  // Debounced search
  let _debounce: ReturnType<typeof setTimeout>;
  $effect(() => {
    const q = searchQuery;
    clearTimeout(_debounce);
    if (!q.trim()) { load(); return; }
    _debounce = setTimeout(async () => {
      if (IS_BROWSER) return;
      try {
        const { Search } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/bookmarkservice');
        bookmarks = ((await Search(q)) || []) as BookmarkEntry[];
      } catch { load(); }
    }, 280);
  });

  onMount(load);
</script>

<PageLayout title="SSH Bookmarks" subtitle="Vault-backed remote host management">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={load}>
        {#if loading}<span class="inline-block animate-spin">⟳</span>{:else}↺{/if} Refresh
      </Button>
      <Button variant="primary" size="sm" onclick={() => showAdd = !showAdd}>+ New Bookmark</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4">

    <!-- Search -->
    <div class="w-full max-w-sm shrink-0">
      <SearchBar bind:value={searchQuery} placeholder="Search hosts, labels, tags…" />
    </div>

    <!-- Add form (inline) -->
    {#if showAdd}
      <div class="shrink-0 bg-surface-2 border border-border-secondary rounded-md p-4 flex flex-col gap-3">
        <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">New Bookmark</span>
        <div class="grid grid-cols-2 gap-2">
          <input class="col-span-2 bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Label  e.g. prod-web-01" bind:value={fLabel} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary font-mono outline-none focus:border-accent" placeholder="Hostname / IP" bind:value={fHost} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" type="number" placeholder="Port" bind:value={fPort} min="1" max="65535" />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" placeholder="Username" bind:value={fUser} />
          <input class="bg-surface-3 border border-border-primary rounded-sm px-2 py-1.5 text-xs text-text-primary outline-none focus:border-accent" type="password" placeholder="Password (blank = key auth)" bind:value={fPass} />
        </div>
        <div class="flex justify-end gap-2">
          <Button variant="secondary" size="sm" onclick={() => showAdd = false}>Cancel</Button>
          <Button variant="primary" size="sm" onclick={save} disabled={saving || !fLabel || !fHost || !fUser}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </div>
      </div>
    {/if}

    <!-- Table / empty states -->
    {#if IS_BROWSER}
      <EmptyState title="Desktop feature" description="SSH bookmarks require the OBLIVRA desktop binary." icon="🔒" />
    {:else if loading && bookmarks.length === 0}
      <div class="text-[11px] text-text-muted font-mono p-4 animate-pulse">Loading bookmarks…</div>
    {:else if filtered.length === 0}
      <EmptyState
        title={searchQuery ? 'No matches' : 'No bookmarks yet'}
        description={searchQuery ? 'Try a different search term.' : 'Add your first SSH host using the button above.'}
        icon="🖧"
      />
    {:else}
      <div class="flex-1 min-h-0 overflow-auto bg-surface-1 border border-border-primary rounded-sm">
        <table class="w-full text-left min-w-[600px]">
          <thead class="sticky top-0 z-10">
            <tr class="bg-surface-2 border-b border-border-primary text-[9px] font-bold uppercase tracking-widest text-text-muted">
              <th class="px-2 py-2 w-6"></th>
              <th class="px-3 py-2">Identity</th>
              <th class="px-3 py-2">Endpoint</th>
              <th class="px-3 py-2">Principal</th>
              <th class="px-3 py-2">Auth</th>
              <th class="px-3 py-2">Tags</th>
              <th class="px-3 py-2 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filtered as b (b.ID)}
              <tr class="border-b border-border-primary hover:bg-surface-2/50 transition-colors group">
                <!-- Favourite star -->
                <td class="px-2 py-2 text-center">
                  <button
                    class="opacity-0 group-hover:opacity-100 transition-opacity leading-none"
                    onclick={() => toggleFav(b.ID)}
                    title="Toggle favourite"
                  >
                    <span class={b.IsFavorite ? 'text-yellow-400' : 'text-text-muted'}>★</span>
                  </button>
                </td>
                <!-- Label + ID -->
                <td class="px-3 py-2">
                  <div class="flex flex-col">
                    <span class="text-[11px] font-bold text-text-heading">{b.Label}</span>
                    <span class="text-[9px] text-text-muted font-mono">{b.ID?.slice(0, 8) ?? '—'}</span>
                  </div>
                </td>
                <!-- Endpoint -->
                <td class="px-3 py-2 font-mono text-[11px] text-text-secondary">{b.Hostname}:{b.Port || 22}</td>
                <!-- Principal -->
                <td class="px-3 py-2 text-[11px] text-text-secondary">{b.Username}</td>
                <!-- Auth method -->
                <td class="px-3 py-2">
                  <Badge variant={b.AuthMethod === 'key' ? 'accent' : 'muted'}>{b.AuthMethod || 'key'}</Badge>
                </td>
                <!-- Tags -->
                <td class="px-3 py-2">
                  <div class="flex gap-1 flex-wrap">
                    {#each (b.Tags || []) as tag}
                      <Badge variant="info">{tag}</Badge>
                    {/each}
                  </div>
                </td>
                <!-- Actions -->
                <td class="px-3 py-2">
                  <div class="flex justify-end gap-1">
                    <Button variant="primary" size="xs" onclick={() => connect(b.ID)}>Connect</Button>
                    <Button variant="danger"  size="xs" onclick={() => remove(b.ID)}>✕</Button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    <!-- KPI strip -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Bookmarks"    value={bookmarks.length}  trend="stable" />
      <KPI label="Favourites"   value={favoriteCount}     variant="accent" trend="stable" />
      <KPI label="Vault Status" value={appStore.vaultUnlocked ? 'Unlocked' : 'Locked'}
           variant={appStore.vaultUnlocked ? 'success' : 'critical'} />
    </div>

  </div>
</PageLayout>
