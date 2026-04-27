<!--
  RemoteExecPanel — generic remote-command runner used by ProcessesPanel,
  NetworkPanel, MetricsPanel, LogsPanel, and ContainersPanel.

  Responsibilities:
    - Resolve an SSH session for the operator-selected host (reusing
      `useShellSession`'s helper).
    - Periodically run a snapshot command (configurable interval + manual
      refresh) and pass the raw output to a slot for the wrapper to parse
      / render however it likes.
    - Show a sane empty state when no host is selected, and a friendly
      error overlay when Exec returns a non-empty stderr-shaped message.

  Each wrapper supplies: title, icon, command, optional pollIntervalMs,
  and a render snippet that consumes the captured stdout text.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { RefreshCw, AlertTriangle, Server } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { execOnHost } from './useShellSession.svelte';
  import type { Snippet } from 'svelte';

  type Props = {
    title: string;
    icon: any;
    command: string;
    /** ms; if >0 the panel re-runs the command every interval. */
    pollIntervalMs?: number;
    /** Render slot — receives the latest stdout string. */
    children: Snippet<[{ output: string; running: boolean }]>;
    /** Optional rightside controls in the header. */
    controls?: Snippet<[{ refresh: () => Promise<void> }]>;
  };

  let { title, icon: Icon, command, pollIntervalMs = 0, children, controls }: Props = $props();

  let output = $state('');
  let running = $state(false);
  let errorMsg = $state('');
  let lastRun = $state<Date | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function runOnce() {
    if (!shellStore.selectedHostID) {
      output = '';
      errorMsg = '';
      return;
    }
    running = true;
    errorMsg = '';
    try {
      output = await execOnHost(command);
      lastRun = new Date();
    } catch (e: any) {
      errorMsg = e?.message ?? String(e);
    } finally {
      running = false;
    }
  }

  // Re-run when the host changes or the command changes.
  $effect(() => {
    void shellStore.selectedHostID;
    void command;
    void runOnce();
  });

  onMount(() => {
    if (pollIntervalMs > 0) {
      timer = setInterval(() => {
        if (shellStore.selectedHostID) void runOnce();
      }, pollIntervalMs);
    }
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  let host = $derived(shellStore.hosts.find((h) => h.id === shellStore.selectedHostID) ?? null);
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Icon size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">{title}</span>
    {#if host}
      <span class="text-[10px] text-[var(--tx3)]">· {host.name}</span>
    {/if}
    {#if lastRun}
      <span class="text-[9px] text-[var(--tx3)]">· last run {lastRun.toLocaleTimeString()}</span>
    {/if}

    {#if controls}
      <div class="ml-auto flex items-center gap-1">
        {@render controls({ refresh: runOnce })}
        <button
          class="flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
          onclick={runOnce}
        >
          <RefreshCw size={11} class={running ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>
    {:else}
      <button
        class="ml-auto flex items-center gap-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={runOnce}
      >
        <RefreshCw size={11} class={running ? 'animate-spin' : ''} />
        Refresh
      </button>
    {/if}
  </header>

  <div class="min-h-0 flex-1 overflow-auto">
    {#if !shellStore.selectedHostID}
      <div class="flex h-full flex-col items-center justify-center gap-3 px-6 py-12 text-center">
        <Server size={28} class="text-[var(--tx3)]" />
        <div class="text-sm text-[var(--tx2)]">Pick a host on the left to inspect.</div>
        <div class="text-[11px] text-[var(--tx3)]">
          This view runs <code class="rounded bg-[var(--s2)] px-1 py-0.5 font-mono">{command}</code> on the selected host.
        </div>
      </div>
    {:else if errorMsg}
      <div class="flex h-full flex-col items-center justify-center gap-3 px-6 py-12 text-center">
        <AlertTriangle size={28} class="text-amber-400" />
        <div class="text-sm text-[var(--tx2)]">Command failed</div>
        <pre class="max-w-2xl whitespace-pre-wrap rounded-md border border-rose-400/30 bg-rose-400/5 p-3 text-left font-mono text-[11px] text-rose-200">{errorMsg}</pre>
      </div>
    {:else}
      {@render children({ output, running })}
    {/if}
  </div>
</div>
