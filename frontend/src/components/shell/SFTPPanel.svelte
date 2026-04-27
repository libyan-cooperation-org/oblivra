<!--
  SFTPPanel — remote file browser bound to OBLIVRA's SSHService SFTP API.
  No new backend needed: ListDirectory / ReadFile / WriteFile / Mkdir /
  Remove are already exposed.

  Single-pane variant for first ship. Two-pane local-vs-remote with
  drag-drop transfers can come in Stage 4.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Folder, File as FileIcon, ArrowUp, RefreshCw, FolderPlus, Trash2, Download, ChevronRight } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { ensureSessionForSelectedHost } from './useShellSession.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';

  type FileInfo = {
    name: string;
    path?: string;
    is_dir?: boolean;
    size?: number;
    mode?: string;
    modified?: string;
  };

  let path = $state('/');
  let entries = $state<FileInfo[]>([]);
  let loading = $state(false);
  let sessionID = $state<string | null>(null);

  async function refresh() {
    loading = true;
    try {
      const sid = await ensureSessionForSelectedHost();
      sessionID = sid;
      if (!sid) {
        entries = [];
        return;
      }
      const { ListDirectory } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
      );
      const list = ((await ListDirectory(sid, path)) ?? []) as FileInfo[];
      // Sort: dirs first, then by name.
      entries = list.sort((a, b) => {
        if (!!a.is_dir !== !!b.is_dir) return a.is_dir ? -1 : 1;
        return (a.name ?? '').localeCompare(b.name ?? '');
      });
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'List directory failed', message: e?.message ?? String(e) });
      entries = [];
    } finally {
      loading = false;
    }
  }

  function go(p: string) {
    path = p.endsWith('/') || p === '/' ? p : p + '/';
    void refresh();
  }
  function open(entry: FileInfo) {
    if (entry.is_dir) go(joinPath(path, entry.name));
  }
  function up() {
    if (path === '/' || !path) return;
    const trimmed = path.replace(/\/$/, '');
    const parent = trimmed.substring(0, trimmed.lastIndexOf('/')) || '/';
    go(parent);
  }
  function joinPath(a: string, b: string) {
    if (!a.endsWith('/')) a += '/';
    return a + b;
  }

  async function mkdir() {
    const name = prompt('New directory name:');
    if (!name) return;
    try {
      const sid = sessionID ?? (await ensureSessionForSelectedHost());
      if (!sid) return;
      const { Mkdir } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
      );
      await Mkdir(sid, joinPath(path, name));
      void refresh();
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Mkdir failed', message: e?.message ?? String(e) });
    }
  }

  async function remove(entry: FileInfo) {
    if (!confirm(`Delete ${entry.is_dir ? 'directory' : 'file'} "${entry.name}"?`)) return;
    try {
      const sid = sessionID ?? (await ensureSessionForSelectedHost());
      if (!sid) return;
      const { Remove } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
      );
      await Remove(sid, joinPath(path, entry.name));
      void refresh();
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Remove failed', message: e?.message ?? String(e) });
    }
  }

  async function download(entry: FileInfo) {
    if (entry.is_dir) return;
    try {
      const sid = sessionID ?? (await ensureSessionForSelectedHost());
      if (!sid) return;
      const remotePath = joinPath(path, entry.name);
      const { SftpDownloadAsync } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
      );
      const local = prompt('Local download path:', `./${entry.name}`);
      if (!local) return;
      await SftpDownloadAsync(sid, remotePath, local, entry.size ?? 0);
      toastStore.add({ type: 'success', title: 'Download started', message: `${remotePath} → ${local}` });
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Download failed', message: e?.message ?? String(e) });
    }
  }

  function fmtSize(b?: number) {
    if (b === undefined) return '—';
    if (b < 1024) return `${b}B`;
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)}KB`;
    if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)}MB`;
    return `${(b / 1024 / 1024 / 1024).toFixed(1)}GB`;
  }

  // Pretty breadcrumb.
  let crumbs = $derived.by(() => {
    if (path === '/') return [{ label: '/', target: '/' }];
    const parts = path.split('/').filter(Boolean);
    const out: { label: string; target: string }[] = [{ label: '/', target: '/' }];
    let acc = '';
    for (const p of parts) {
      acc += '/' + p;
      out.push({ label: p, target: acc });
    }
    return out;
  });

  $effect(() => {
    // Re-list when the operator picks a different host.
    if (shellStore.selectedHostID) {
      path = '/';
      void refresh();
    }
  });

  onMount(() => {
    if (shellStore.selectedHostID) void refresh();
  });
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Folder size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Files</span>
    {#if shellStore.selectedHostID}
      <span class="text-[10px] text-[var(--tx3)]">·
        {shellStore.hosts.find((h) => h.id === shellStore.selectedHostID)?.name ?? shellStore.selectedHostID}
      </span>
    {/if}

    <div class="ml-auto flex items-center gap-1">
      <button
        class="rounded p-1 text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
        title="Up"
        onclick={up}
        disabled={path === '/'}
      >
        <ArrowUp size={12} />
      </button>
      <button
        class="rounded p-1 text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
        title="New folder"
        onclick={mkdir}
        disabled={!shellStore.selectedHostID}
      >
        <FolderPlus size={12} />
      </button>
      <button
        class="rounded p-1 text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
        title="Refresh"
        onclick={refresh}
      >
        <RefreshCw size={12} class={loading ? 'animate-spin' : ''} />
      </button>
    </div>
  </header>

  <!-- Breadcrumb -->
  <nav class="flex items-center gap-1 overflow-x-auto border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-1.5 text-[11px]">
    {#each crumbs as c, i (c.target)}
      {#if i > 0}<ChevronRight size={10} class="text-[var(--tx3)]" />{/if}
      <button
        class="rounded px-1 py-0.5 font-mono hover:bg-[var(--s2)] hover:text-[var(--tx)] {c.target === path ? 'text-[var(--tx)]' : 'text-[var(--tx2)]'}"
        onclick={() => go(c.target)}
      >{c.label}</button>
    {/each}
  </nav>

  <div class="min-h-0 flex-1 overflow-auto">
    {#if !shellStore.selectedHostID}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        Pick a host on the left to browse its files.
      </div>
    {:else if entries.length === 0 && !loading}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        Empty directory.
      </div>
    {:else}
      <table class="w-full text-xs">
        <thead class="sticky top-0 bg-[var(--s1)] text-[10px] uppercase tracking-wider text-[var(--tx3)]">
          <tr>
            <th class="px-3 py-2 text-left">Name</th>
            <th class="px-3 py-2 text-right">Size</th>
            <th class="px-3 py-2 text-left">Modified</th>
            <th class="px-3 py-2 text-left">Mode</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          {#each entries as e (e.name)}
            <tr class="group border-t border-[var(--b1)] hover:bg-[var(--s1)]">
              <td class="px-3 py-1.5">
                <button
                  class="flex items-center gap-2 text-left {e.is_dir ? 'font-medium text-[var(--tx)]' : 'text-[var(--tx2)]'}"
                  onclick={() => open(e)}
                  ondblclick={() => open(e)}
                >
                  {#if e.is_dir}<Folder size={12} class="text-cyan-400" />{:else}<FileIcon size={12} class="text-[var(--tx3)]" />{/if}
                  <span class="truncate">{e.name}</span>
                </button>
              </td>
              <td class="px-3 py-1.5 text-right font-mono text-[10px] text-[var(--tx2)]">{e.is_dir ? '—' : fmtSize(e.size)}</td>
              <td class="px-3 py-1.5 font-mono text-[10px] text-[var(--tx3)]">{e.modified ?? ''}</td>
              <td class="px-3 py-1.5 font-mono text-[10px] text-[var(--tx3)]">{e.mode ?? ''}</td>
              <td class="px-3 py-1.5">
                <div class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                  {#if !e.is_dir}
                    <button class="rounded p-1 text-cyan-300 hover:bg-cyan-400/10" onclick={() => download(e)} title="Download"><Download size={11} /></button>
                  {/if}
                  <button class="rounded p-1 text-rose-300 hover:bg-rose-400/10" onclick={() => remove(e)} title="Delete"><Trash2 size={11} /></button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
