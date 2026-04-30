<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { storageStats, storagePromote, type TierStats } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let stats = $state<TierStats | null>(null);
  let busy = $state(false);
  let error = $state<string | null>(null);
  let lastResult = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      stats = await storageStats();
      error = null;
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function promote() {
    busy = true;
    try {
      const res = await storagePromote();
      lastResult = `migrated ${res.moved} events to warm tier`;
      await refresh();
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 5000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<div class="mx-auto max-w-5xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Admin</p>
    <h2 class="text-2xl font-semibold tracking-tight">Storage tiering · Tenants · Health</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Warm files" value={stats?.warmFiles ?? '—'} hint="parquet" />
    <Tile label="Warm events" value={stats?.warmEvents ?? '—'} />
    <Tile label="Hot age max" value={stats?.hotAgeMax ?? '—'} hint="migration cutoff" />
    <Tile
      label="Last migration"
      value={stats?.lastRunAt ? new Date(stats.lastRunAt).toLocaleTimeString() : '—'}
      hint={stats ? `${stats.lastRunMoved} moved` : ''}
    />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-3">
    <h3 class="text-sm font-semibold tracking-wide text-slate-100">Promote hot → warm</h3>
    <p class="text-xs text-night-300">
      Walks the hot store, exports events older than {stats?.hotAgeMax ?? '30 days'} to a Parquet
      file in <code class="text-night-200">{stats?.warmDir ?? 'warm.parquet/'}</code>. Events stay
      in the hot store until the next pass — this is intentionally a copy first.
    </p>
    <div class="flex items-center gap-3">
      <button
        class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-accent-600 disabled:opacity-50"
        disabled={busy}
        onclick={promote}
      >
        Run migration now
      </button>
      {#if lastResult}
        <span class="text-xs text-signal-success">{lastResult}</span>
      {/if}
      {#if error}
        <span class="text-xs text-signal-error">{error}</span>
      {/if}
    </div>
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-2">
    <h3 class="text-sm font-semibold tracking-wide text-slate-100">RBAC roles</h3>
    <p class="text-xs text-night-300">
      Active roles: <code>admin</code>, <code>analyst</code>, <code>readonly</code>,
      <code>agent</code>. Set <code class="text-night-200">OBLIVRA_API_KEYS=k1,k2,...</code>
      to require <code>Authorization: Bearer &lt;key&gt;</code> on every <code>/api/*</code> route.
      Per-key role assignment lands in Phase 12.x.
    </p>
  </section>
</div>
