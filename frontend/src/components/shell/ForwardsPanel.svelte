<!--
  ForwardsPanel — list active SSH port forwards. Stop any of them.

  Bound to OBLIVRA's TunnelService. CreateTunnel takes an *ssh.Client
  pointer that can't be marshalled from the frontend, so creating new
  forwards is deferred to the existing /tunnels page (link button at top).
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Network, Trash2, RefreshCw, Plus, ExternalLink } from 'lucide-svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { push } from '@lib/router.svelte';

  type Tunnel = {
    id?: string;
    type?: string;       // 'local' | 'remote' | 'dynamic'
    local_host?: string;
    local_port?: number;
    remote_host?: string;
    remote_port?: number;
    session_id?: string;
    status?: string;
    created_at?: string;
  };

  let items = $state<Tunnel[]>([]);
  let loading = $state(false);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function load() {
    loading = true;
    try {
      const { GetAll } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/tunnelservice'
      );
      const list = ((await GetAll()) ?? []) as Tunnel[];
      items = list;
    } catch (e: any) {
      toastStore.add({ type: 'warning', title: 'Forwards load failed', message: e?.message ?? String(e) });
    } finally {
      loading = false;
    }
  }

  async function stopOne(id: string) {
    try {
      const { StopTunnel } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/tunnelservice'
      );
      await StopTunnel(id);
      items = items.filter((i) => i.id !== id);
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Stop failed', message: e?.message ?? String(e) });
    }
  }

  async function stopAll() {
    if (!confirm('Stop ALL active forwards?')) return;
    try {
      const { CloseAll } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/tunnelservice'
      );
      await CloseAll();
      items = [];
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Close-all failed', message: e?.message ?? String(e) });
    }
  }

  onMount(() => {
    void load();
    // Light polling so newly-created tunnels show up without manual refresh.
    timer = setInterval(load, 5000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Network size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Port Forwards</span>
    <span class="text-[10px] text-[var(--tx3)]">· {items.length} active</span>

    <div class="ml-auto flex items-center gap-1">
      <button
        class="flex items-center gap-1 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-2 py-1 text-[11px] text-cyan-200 hover:bg-cyan-400/20"
        onclick={() => push('/tunnels')}
        title="Open the full Tunnels page to create new forwards"
      >
        <Plus size={11} />
        New forward
        <ExternalLink size={9} />
      </button>
      <button
        class="flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={load}
      >
        <RefreshCw size={11} class={loading ? 'animate-spin' : ''} />
        Refresh
      </button>
      {#if items.length > 0}
        <button
          class="flex items-center gap-1 rounded-md border border-rose-400/30 bg-rose-400/10 px-2 py-1 text-[11px] text-rose-300 hover:bg-rose-400/20"
          onclick={stopAll}
        >
          <Trash2 size={11} />
          Stop all
        </button>
      {/if}
    </div>
  </header>

  <div class="min-h-0 flex-1 overflow-y-auto">
    {#if items.length === 0 && !loading}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        No active forwards.
        <div class="mt-2 text-[11px]">
          Open a forward from the
          <button class="text-cyan-300 underline" onclick={() => push('/tunnels')}>Tunnels page</button>.
        </div>
      </div>
    {/if}
    <table class="w-full text-xs">
      <thead class="sticky top-0 bg-[var(--s1)] text-[10px] uppercase tracking-wider text-[var(--tx3)]">
        <tr>
          <th class="px-3 py-2 text-left">Type</th>
          <th class="px-3 py-2 text-left">Local</th>
          <th class="px-3 py-2 text-left">Remote</th>
          <th class="px-3 py-2 text-left">Session</th>
          <th class="px-3 py-2 text-left">Status</th>
          <th class="px-3 py-2"></th>
        </tr>
      </thead>
      <tbody>
        {#each items as t (t.id)}
          <tr class="group border-t border-[var(--b1)] hover:bg-[var(--s1)]">
            <td class="px-3 py-2">
              <span class="rounded bg-[var(--s3)] px-1.5 py-0.5 text-[9px] font-mono uppercase tracking-wider text-[var(--tx2)]">
                {t.type ?? 'local'}
              </span>
            </td>
            <td class="px-3 py-2 font-mono">{t.local_host ?? '127.0.0.1'}:{t.local_port ?? '?'}</td>
            <td class="px-3 py-2 font-mono">{t.remote_host ?? '?'}:{t.remote_port ?? '?'}</td>
            <td class="px-3 py-2 font-mono text-[10px] text-[var(--tx3)]">{(t.session_id ?? '').slice(0, 8) || '—'}</td>
            <td class="px-3 py-2">
              <span class="rounded-sm border border-emerald-400/30 bg-emerald-400/10 px-1.5 py-0.5 text-[9px] uppercase tracking-wider text-emerald-300">
                {t.status ?? 'active'}
              </span>
            </td>
            <td class="px-3 py-2">
              {#if t.id}
                <button
                  class="rounded p-1 text-rose-300 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-rose-400/10"
                  onclick={() => stopOne(t.id!)}
                  title="Stop"
                >
                  <Trash2 size={12} />
                </button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
