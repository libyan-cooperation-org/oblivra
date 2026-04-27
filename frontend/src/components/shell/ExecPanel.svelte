<!--
  ExecPanel — run one command in parallel across N selected hosts and
  show per-host stdout side by side. Backed by OBLIVRA's SSHService.Exec
  and (for multi-host) Connect to spin up sessions on demand.
-->
<script lang="ts">
  import { Zap, Play, X, CheckCircle2, AlertTriangle, Loader2 } from 'lucide-svelte';
  import { shellStore, type ShellHostSummary } from '@lib/stores/shell.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';

  type RunStatus = 'pending' | 'connecting' | 'running' | 'ok' | 'error';
  type Run = {
    host: ShellHostSummary;
    status: RunStatus;
    output: string;
    durationMs?: number;
    error?: string;
  };

  let selected = $state<Set<string>>(new Set());
  let cmd = $state('uname -a');
  let runs = $state<Run[]>([]);
  let running = $state(false);

  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selected = next;
  }

  function selectAll() {
    selected = new Set(shellStore.hosts.map((h) => h.id));
  }
  function clearAll() {
    selected = new Set();
  }

  async function execute() {
    if (selected.size === 0) {
      toastStore.add({ type: 'warning', title: 'Select at least one host' });
      return;
    }
    if (!cmd.trim()) {
      toastStore.add({ type: 'warning', title: 'Enter a command' });
      return;
    }
    running = true;
    const hosts = shellStore.hosts.filter((h) => selected.has(h.id));
    runs = hosts.map((h) => ({ host: h, status: 'pending' as RunStatus, output: '' }));

    const ssh = await import(
      '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
    );

    // Fire each host in parallel; per-row state updates as they complete.
    await Promise.all(
      hosts.map(async (h, idx) => {
        const start = performance.now();
        try {
          runs[idx] = { ...runs[idx], status: 'connecting' };
          let sid = (await ssh.GetActiveSessionForHost(h.id)) as string;
          if (!sid) sid = (await ssh.Connect(h.id)) as string;
          if (!sid) throw new Error('failed to obtain SSH session');
          runs[idx] = { ...runs[idx], status: 'running' };
          const out = (await ssh.Exec(sid, cmd)) as string;
          runs[idx] = {
            ...runs[idx],
            status: 'ok',
            output: out ?? '',
            durationMs: Math.round(performance.now() - start),
          };
        } catch (e: any) {
          runs[idx] = {
            ...runs[idx],
            status: 'error',
            error: e?.message ?? String(e),
            durationMs: Math.round(performance.now() - start),
          };
        }
      }),
    );
    running = false;
  }
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Zap size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Multi-host exec</span>
    <span class="text-[10px] text-[var(--tx3)]">· {selected.size} / {shellStore.hosts.length} hosts</span>

    <div class="ml-auto flex items-center gap-1">
      <button class="rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]" onclick={selectAll}>Select all</button>
      <button class="rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]" onclick={clearAll}>Clear</button>
    </div>
  </header>

  <!-- Command bar -->
  <div class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <input
      class="flex-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 font-mono text-xs outline-none focus:border-cyan-400/40"
      placeholder="Command (e.g. uname -a)"
      bind:value={cmd}
      onkeydown={(e) => e.key === 'Enter' && execute()}
    />
    <button
      class="flex items-center gap-1.5 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-3 py-1.5 text-xs text-cyan-200 hover:bg-cyan-400/20 disabled:opacity-50"
      onclick={execute}
      disabled={running || selected.size === 0}
    >
      {#if running}<Loader2 size={12} class="animate-spin" />{:else}<Play size={12} />{/if}
      Run on {selected.size} host{selected.size === 1 ? '' : 's'}
    </button>
  </div>

  <div class="grid min-h-0 flex-1 grid-cols-[260px_1fr] overflow-hidden">
    <!-- Host picker -->
    <aside class="overflow-y-auto border-r border-[var(--b1)] bg-[var(--s1)]">
      {#each shellStore.hosts as h (h.id)}
        <label class="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-xs hover:bg-[var(--s2)]">
          <input type="checkbox" class="accent-cyan-400" checked={selected.has(h.id)} onchange={() => toggle(h.id)} />
          <div class="min-w-0 flex-1">
            <div class="truncate">{h.name}</div>
            <div class="truncate text-[10px] text-[var(--tx3)]">{h.username}@{h.host}</div>
          </div>
        </label>
      {/each}
      {#if shellStore.hosts.length === 0}
        <div class="px-4 py-6 text-center text-[11px] text-[var(--tx3)]">
          No saved hosts. Add some on the SSH page.
        </div>
      {/if}
    </aside>

    <!-- Per-host result panes -->
    <div class="overflow-y-auto">
      {#if runs.length === 0}
        <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
          Pick hosts on the left, type a command above, hit Run.
        </div>
      {:else}
        <div class="divide-y divide-[var(--b1)]">
          {#each runs as r (r.host.id)}
            <div class="p-3">
              <div class="mb-1 flex items-center gap-2">
                {#if r.status === 'ok'}
                  <CheckCircle2 size={13} class="text-emerald-400" />
                {:else if r.status === 'error'}
                  <AlertTriangle size={13} class="text-rose-400" />
                {:else if r.status === 'running' || r.status === 'connecting'}
                  <Loader2 size={13} class="animate-spin text-cyan-400" />
                {:else}
                  <span class="h-2 w-2 rounded-full bg-[var(--tx3)]"></span>
                {/if}
                <span class="text-xs font-semibold">{r.host.name}</span>
                <span class="font-mono text-[10px] text-[var(--tx3)]">{r.host.username}@{r.host.host}</span>
                <span class="ml-auto font-mono text-[10px] text-[var(--tx3)]">
                  {r.status}{r.durationMs !== undefined ? ` · ${r.durationMs}ms` : ''}
                </span>
              </div>
              {#if r.error}
                <pre class="rounded-md border border-rose-400/30 bg-rose-400/5 p-2 font-mono text-[11px] text-rose-200">{r.error}</pre>
              {:else if r.output}
                <pre class="rounded-md border border-[var(--b1)] bg-[var(--s0)] p-2 font-mono text-[11px] text-[var(--tx2)]">{r.output}</pre>
              {:else if r.status === 'pending' || r.status === 'connecting' || r.status === 'running'}
                <div class="text-[11px] text-[var(--tx3)]">…</div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
