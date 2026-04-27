<!--
  HostList — left sidebar for the Shell Workspace.

  Adapted from Blacknode (MIT) but bound to OBLIVRA's HostService via
  shellStore. Differences vs upstream:
    - Click a host → spawn a NEW tab pre-connected to that host (via
      shellStore.spawnTabForHost). Blacknode's original was passive
      selection-only; the spawn-on-click variant is what most operators
      actually want — it matches Termius / iTerm "double-click to open".
    - Edit / Delete / Import buttons are stubs that link to OBLIVRA's
      existing /ssh bookmarks page rather than re-implementing
      HostEditor.svelte (~260 lines) and SSHConfigImport.svelte (~230
      lines). Stage 3 polish can in-line them.
    - Grouping uses OBLIVRA's `category` field (the closest analogue to
      Blacknode's `group`).
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Search, Plus, Server, KeyRound, Lock, FileText, ExternalLink } from 'lucide-svelte';
  import { shellStore, type ShellHostSummary } from '@lib/stores/shell.svelte';
  import { push } from '@lib/router.svelte';

  let filter = $state('');

  let visible = $derived.by<ShellHostSummary[]>(() =>
    shellStore.hosts.filter((h) => {
      if (!filter) return true;
      const f = filter.toLowerCase();
      return (
        h.name.toLowerCase().includes(f) ||
        h.host.toLowerCase().includes(f) ||
        (h.environment ?? '').toLowerCase().includes(f) ||
        (h.username ?? '').toLowerCase().includes(f)
      );
    }),
  );

  // Group by environment/category. "Ungrouped" bucket for hosts with none.
  let groups = $derived.by(() =>
    visible.reduce<Record<string, ShellHostSummary[]>>((acc, h) => {
      const g = (h.environment ?? '').trim() || 'Ungrouped';
      (acc[g] ??= []).push(h);
      return acc;
    }, {}),
  );

  function envBadge(env: string | undefined) {
    const e = (env ?? '').toLowerCase();
    if (e === 'production' || e === 'prod') return { label: 'PROD', color: '#fca5a5', bg: 'rgba(239,68,68,0.15)', border: 'rgba(239,68,68,0.40)' };
    if (e === 'staging' || e === 'stage') return { label: 'STG', color: '#fcd34d', bg: 'rgba(245,158,11,0.15)', border: 'rgba(245,158,11,0.40)' };
    if (e === 'development' || e === 'dev') return { label: 'DEV', color: '#86efac', bg: 'rgba(34,197,94,0.15)', border: 'rgba(34,197,94,0.40)' };
    return { label: '', color: '', bg: '', border: '' };
  }

  function authIcon(method: string | undefined) {
    return method === 'key' ? KeyRound : Lock;
  }

  function connect(h: ShellHostSummary) {
    shellStore.selectedHostID = h.id;
    shellStore.spawnTabForHost(h.id);
  }

  onMount(() => {
    void shellStore.refreshHosts();
  });
</script>

<div class="flex h-full w-full flex-col bg-[var(--s1)] text-[var(--tx)]">
  <!-- Header -->
  <div class="flex items-center gap-2 border-b border-[var(--b1)] px-3 py-2.5">
    <span class="text-[10px] font-medium uppercase tracking-[0.14em] text-[var(--tx3)]">Hosts</span>
    <button
      class="ml-auto flex h-6 w-6 items-center justify-center rounded-md text-[var(--tx3)] hover:bg-[var(--s3)] hover:text-cyan-400"
      onclick={() => push('/ssh')}
      title="Manage hosts (opens SSH Bookmarks page)"
    >
      <ExternalLink size={11} />
    </button>
    <button
      class="flex h-6 w-6 items-center justify-center rounded-md text-[var(--tx3)] hover:bg-[var(--s3)] hover:text-cyan-400"
      onclick={() => push('/ssh')}
      title="Add host"
    >
      <Plus size={12} />
    </button>
  </div>

  <!-- Search -->
  <div class="px-3 py-2">
    <div class="relative flex items-center rounded-md border border-[var(--b1)] bg-[var(--s2)] focus-within:border-cyan-400/40">
      <Search size={12} class="absolute left-2.5 text-[var(--tx3)]" />
      <input
        class="w-full bg-transparent py-1.5 pl-7 pr-2 text-xs outline-none placeholder:text-[var(--tx3)]"
        placeholder="Search hosts…"
        bind:value={filter}
      />
    </div>
  </div>

  <!-- Body -->
  <div class="flex-1 overflow-y-auto pb-2">
    {#each Object.entries(groups) as [name, list] (name)}
      <div class="px-3 pt-3 pb-1 text-[9px] font-medium uppercase tracking-[0.14em] text-[var(--tx3)]">
        {name}
      </div>
      {#each list as h (h.id)}
        {@const Icon = authIcon(h.authMethod)}
        {@const env = envBadge(h.environment)}
        {@const isSel = shellStore.selectedHostID === h.id}
        <button
          class="group relative mx-2 my-0.5 flex w-[calc(100%-1rem)] items-center gap-2 overflow-hidden rounded-md border px-2 py-1.5 text-left transition-colors {isSel
            ? 'border-cyan-400/30 bg-cyan-400/10 text-[var(--tx)]'
            : 'border-transparent text-[var(--tx2)] hover:bg-[var(--s2)]'}"
          onclick={() => connect(h)}
          title={`Connect — opens a new tab via SSH to ${h.username}@${h.host}`}
        >
          {#if env.label}
            <span class="absolute inset-y-0 left-0 w-0.5" style:background={env.color}></span>
          {/if}
          <Server size={13} class={isSel ? 'text-cyan-400' : 'text-[var(--tx3)]'} />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5 truncate text-xs">
              <span class="truncate">{h.name}</span>
              {#if env.label}
                <span
                  class="shrink-0 rounded-sm border px-1 text-[8px] font-mono font-semibold"
                  style:color={env.color}
                  style:background={env.bg}
                  style:border-color={env.border}
                >
                  {env.label}
                </span>
              {/if}
              <Icon size={9} class="shrink-0 text-[var(--tx3)]" />
            </div>
            <div class="truncate text-[10px] text-[var(--tx3)]">
              {h.username}@{h.host}:{h.port}
            </div>
          </div>
        </button>
      {/each}
    {/each}
    {#if shellStore.hosts.length === 0}
      <div class="px-4 py-8 text-center">
        <Server size={20} class="mx-auto text-[var(--tx3)]" />
        <p class="mt-2 text-[11px] text-[var(--tx3)]">No saved hosts yet</p>
        <button
          class="mt-3 rounded-md border border-[var(--b1)] px-2.5 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
          onclick={() => push('/ssh')}
        >
          + Add your first host
        </button>
      </div>
    {/if}
  </div>

  <!-- Footer hint -->
  <div class="border-t border-[var(--b1)] px-3 py-2 text-[9px] uppercase tracking-[0.14em] text-[var(--tx3)]">
    <FileText size={9} class="-mt-0.5 mr-1 inline" />
    Click a host → opens new tab pre-connected
  </div>
</div>
