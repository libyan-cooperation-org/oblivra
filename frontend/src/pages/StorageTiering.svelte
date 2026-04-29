<!--
  OBLIVRA — Storage Tiering (Phase 22.3)

  Visualises the Hot / Warm / Cold tier state and the most recent
  migration cycle. Read-only for analyst+; the "Promote now" button
  is admin-only and triggers an out-of-band migration cycle (useful
  after changing retention policy).

  Data path:
    GET  /api/v1/storage/tiering/stats   — per-tier sizes + last cycle
    POST /api/v1/storage/tiering/promote — manual trigger (admin)

  Both go through `apiFetch` so desktop mode retargets `/api/*` to
  the in-process REST listener.

  Honest empty state:
    503 from the backend means tiering wasn't configured (e.g. HotStore
    failed to open). We render an explicit "tiering unavailable" panel
    instead of zeros, so the operator never mistakes "not running" for
    "running and empty".
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, KPI, Badge, Button, PopOutButton } from '@components/ui';
  import { Database, HardDrive, Snowflake, RefreshCw, Zap } from 'lucide-svelte';
  import { apiFetch } from '@lib/apiClient';
  import { appStore } from '@lib/stores/app.svelte';

  type TierEntry = { id: 'hot' | 'warm' | 'cold'; size_bytes: number };
  type CycleStats = {
    started_at: string;
    finished_at: string;
    hot_to_warm: number;
    warm_to_cold: number;
    errors?: string[];
  };
  type Resp = { ok: boolean; tiers: TierEntry[]; last_cycle: CycleStats | null };

  let tiers = $state<TierEntry[]>([]);
  let lastCycle = $state<CycleStats | null>(null);
  let loading = $state(false);
  let unavailable = $state(false);
  let promoting = $state(false);
  let lastError = $state<string | null>(null);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  function fmtBytes(n: number): string {
    if (n < 0) return '—';
    if (n === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    let i = 0;
    let v = n;
    while (v >= 1024 && i < units.length - 1) { v /= 1024; i += 1; }
    return v < 10 ? `${v.toFixed(2)} ${units[i]}` : v < 100 ? `${v.toFixed(1)} ${units[i]}` : `${Math.round(v)} ${units[i]}`;
  }

  function tierIcon(id: string) {
    if (id === 'hot') return Database;
    if (id === 'warm') return HardDrive;
    return Snowflake;
  }
  function tierVariant(id: string): 'critical' | 'warning' | 'accent' {
    if (id === 'hot') return 'critical';
    if (id === 'warm') return 'warning';
    return 'accent';
  }
  function tierDesc(id: string): string {
    if (id === 'hot') return 'BadgerDB, 0–30 d, fast SSD — active query path';
    if (id === 'warm') return 'Parquet, 30–180 d, columnar — multi-second query';
    return 'JSONL local, 180+ d, cheap — compliance / forensic only';
  }

  async function refresh() {
    loading = true;
    lastError = null;
    try {
      const res = await apiFetch('/api/v1/storage/tiering/stats');
      if (res.status === 503) {
        unavailable = true;
        return;
      }
      if (!res.ok) {
        lastError = `HTTP ${res.status}`;
        return;
      }
      const body = (await res.json()) as Resp;
      unavailable = false;
      tiers = body.tiers ?? [];
      lastCycle = body.last_cycle ?? null;
    } catch (err: any) {
      lastError = String(err?.message ?? err);
    } finally {
      loading = false;
    }
  }

  async function promoteNow() {
    if (promoting) return;
    if (!confirm('Run a Hot→Warm→Cold migration cycle now? Events older than the configured thresholds will be moved between tiers. This may take several minutes on large datasets.')) {
      return;
    }
    promoting = true;
    try {
      const res = await apiFetch('/api/v1/storage/tiering/promote', { method: 'POST' });
      if (!res.ok) {
        const txt = await res.text();
        appStore.notify('Promotion failed', 'error', txt || `HTTP ${res.status}`);
        return;
      }
      const body = await res.json();
      const c: CycleStats = body.cycle;
      lastCycle = c;
      const moved = (c.hot_to_warm ?? 0) + (c.warm_to_cold ?? 0);
      appStore.notify(
        'Migration complete',
        'success',
        `${c.hot_to_warm ?? 0} hot→warm · ${c.warm_to_cold ?? 0} warm→cold (${moved} total)`,
      );
      // Sizes likely changed — refresh.
      void refresh();
    } catch (err: any) {
      appStore.notify('Promotion failed', 'error', String(err?.message ?? err));
    } finally {
      promoting = false;
    }
  }

  onMount(() => {
    refresh();
    // Poll every 30 s — tier sizes don't move at sub-minute timescales.
    pollTimer = setInterval(refresh, 30_000);
  });

  onDestroy(() => {
    if (pollTimer) clearInterval(pollTimer);
  });

  const totalBytes = $derived(
    tiers.reduce((acc, t) => acc + (t.size_bytes > 0 ? t.size_bytes : 0), 0),
  );
</script>

<PageLayout
  title="Storage Tiering"
  subtitle="Hot · Warm · Cold storage lifecycle and migration observability"
>
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh} loading={loading}>
      Refresh
    </Button>
    <Button variant="primary" size="sm" icon={Zap} onclick={promoteNow} loading={promoting} disabled={unavailable}>
      Promote now
    </Button>
    <PopOutButton route="/storage-tiering" title="Storage Tiering" />
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    {#if unavailable}
      <div class="flex flex-col items-center justify-center flex-1 gap-3 opacity-70 text-center max-w-2xl mx-auto px-6">
        <HardDrive size={48} class="text-text-muted" />
        <div class="text-sm font-bold text-text-heading">Tiering not configured on this deployment</div>
        <p class="text-[11px] text-text-muted leading-relaxed">
          The Hot/Warm/Cold tier migrator could not initialise. Most likely the BadgerDB hot store
          failed to open at boot — check the server log for <code class="font-mono text-text-secondary">[STORAGE]</code> warnings.
          Until the migrator is online, all events accumulate in the hot tier indefinitely (no rotation).
        </p>
      </div>
    {:else}
      <!-- Tier strip -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
        {#each tiers as t (t.id)}
          {@const Icon = tierIcon(t.id)}
          <div class="bg-surface-1 border border-border-primary rounded-md p-5 flex flex-col gap-3">
            <div class="flex items-center justify-between">
              <Icon size={18} class={t.id === 'hot' ? 'text-error' : t.id === 'warm' ? 'text-warning' : 'text-accent'} />
              <Badge variant={tierVariant(t.id)} size="xs" class="uppercase">{t.id}</Badge>
            </div>
            <div class="text-2xl font-mono font-bold text-text-heading tabular-nums">{fmtBytes(t.size_bytes)}</div>
            <div class="text-[10px] text-text-muted leading-relaxed">{tierDesc(t.id)}</div>
            {#if totalBytes > 0 && t.size_bytes > 0}
              <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                <div
                  class="h-full {t.id === 'hot' ? 'bg-error' : t.id === 'warm' ? 'bg-warning' : 'bg-accent'}"
                  style="width: {Math.round((t.size_bytes / totalBytes) * 100)}%"
                ></div>
              </div>
              <div class="text-[9px] text-text-muted font-mono">
                {Math.round((t.size_bytes / totalBytes) * 100)}% of total
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <!-- Last migration cycle -->
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col">
        <div class="p-3 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
          Most recent migration cycle
        </div>
        <div class="p-5">
          {#if !lastCycle}
            <div class="text-[11px] text-text-muted">
              No migration cycle has run yet since startup. The migrator runs hourly; click <strong>Promote now</strong> to fire one immediately.
            </div>
          {:else}
            <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
              <KPI
                label="Hot → Warm"
                value={(lastCycle.hot_to_warm ?? 0).toLocaleString()}
                sublabel="Events moved"
                variant={lastCycle.hot_to_warm > 0 ? 'success' : 'muted'}
              />
              <KPI
                label="Warm → Cold"
                value={(lastCycle.warm_to_cold ?? 0).toLocaleString()}
                sublabel="Events moved"
                variant={lastCycle.warm_to_cold > 0 ? 'success' : 'muted'}
              />
              <KPI
                label="Started"
                value={lastCycle.started_at?.slice(11, 19) ?? '—'}
                sublabel={lastCycle.started_at?.slice(0, 10) ?? ''}
                variant="accent"
              />
              <KPI
                label="Errors"
                value={(lastCycle.errors?.length ?? 0).toString()}
                sublabel={lastCycle.errors?.length ? 'See log' : 'Clean cycle'}
                variant={(lastCycle.errors?.length ?? 0) > 0 ? 'critical' : 'success'}
              />
            </div>
            {#if (lastCycle.errors?.length ?? 0) > 0}
              <div class="mt-4 p-3 bg-error/10 border border-error/30 rounded text-[10px] font-mono">
                {#each lastCycle.errors ?? [] as err}
                  <div class="text-error mb-1">• {err}</div>
                {/each}
              </div>
            {/if}
          {/if}
        </div>
      </div>

      {#if lastError}
        <div class="text-[10px] font-mono text-error">load error: {lastError}</div>
      {/if}
    {/if}
  </div>
</PageLayout>
