<!--
  OBLIVRA — Agent Integrity (Tamper Path 1, Layer 5).

  Per-agent integrity dashboard. Shows:
    • Last-seen heartbeat time
    • Time skew vs server clock
    • Log file size delta (truncation = critical)
    • Status: OK / STALE / DARK / TAMPERED

  Backed by GET /api/v1/integrity. Auto-refreshes every 30s.

  When tamper indicators fire, alerts also flow into the regular
  /alerts queue via the bus event `tamper:detected` — operators see
  the tamper from either entry point.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, Badge, Button, DataTable, EmptyState, LastRefreshed } from '@components/ui';
  import { ShieldCheck, ShieldAlert, ShieldQuestion, RefreshCw, Activity } from 'lucide-svelte';
  import { apiFetch } from '@lib/apiClient';
  import { appStore } from '@lib/stores/app.svelte';

  type IntegrityRow = {
    agent_id: string;
    received_at: string;
    uptime_s: number;
    log_file_size: number;
    skew_seconds: number;
    seconds_since_heartbeat: number;
    status: 'OK' | 'STALE' | 'DARK' | 'TAMPERED';
    last_hash?: string;
  };

  let rows = $state<IntegrityRow[]>([]);
  let counts = $state({ healthy: 0, stale: 0, dark: 0, tampered: 0 });
  let loading = $state(false);
  // Trust signal — operators auditing tamper indicators need to know
  // whether the table is fresh or a stale 30s-old poll.
  let lastSync = $state<Date | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    loading = true;
    try {
      const res = await apiFetch('/api/v1/integrity');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const body = await res.json();
      rows = (body.agents ?? []) as IntegrityRow[];
      counts = {
        healthy: body.healthy_count ?? 0,
        stale: body.stale_count ?? 0,
        dark: body.dark_count ?? 0,
        tampered: body.tampered_count ?? 0,
      };
      lastSync = new Date();
    } catch (e: any) {
      appStore.notify('Integrity load failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(refresh, 30_000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  function statusBadge(s: IntegrityRow['status']) {
    switch (s) {
      case 'OK': return { variant: 'success' as const, label: 'OK' };
      case 'STALE': return { variant: 'warning' as const, label: 'STALE' };
      case 'DARK': return { variant: 'muted' as const, label: 'DARK' };
      case 'TAMPERED': return { variant: 'critical' as const, label: 'TAMPERED' };
    }
  }

  function fmtAgo(seconds: number): string {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
    return `${Math.floor(seconds / 86400)}d`;
  }
</script>

<PageLayout
  title="Agent Integrity"
  subtitle="Tamper-evidence · heartbeat · log-truncation detection"
>
  {#snippet toolbar()}
    <LastRefreshed time={lastSync} staleThresholdSec={45} />
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-3">
    <!-- KPI strip -->
    <div class="grid grid-cols-4 gap-3 shrink-0">
      <div class="bg-surface-2 border border-border-primary rounded-md p-3 flex items-center gap-3">
        <ShieldCheck size={18} class="text-success" />
        <div>
          <div class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted">Healthy</div>
          <div class="text-[20px] font-mono font-bold text-success">{counts.healthy}</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-md p-3 flex items-center gap-3">
        <Activity size={18} class={counts.stale > 0 ? 'text-warning' : 'text-text-muted'} />
        <div>
          <div class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted">Stale</div>
          <div class="text-[20px] font-mono font-bold {counts.stale > 0 ? 'text-warning' : 'text-text-muted'}">{counts.stale}</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-md p-3 flex items-center gap-3">
        <ShieldQuestion size={18} class={counts.dark > 0 ? 'text-warning' : 'text-text-muted'} />
        <div>
          <div class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted">Dark</div>
          <div class="text-[20px] font-mono font-bold {counts.dark > 0 ? 'text-warning' : 'text-text-muted'}">{counts.dark}</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-md p-3 flex items-center gap-3">
        <ShieldAlert size={18} class={counts.tampered > 0 ? 'text-error animate-pulse' : 'text-text-muted'} />
        <div>
          <div class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted">Tampered</div>
          <div class="text-[20px] font-mono font-bold {counts.tampered > 0 ? 'text-error' : 'text-text-muted'}">{counts.tampered}</div>
        </div>
      </div>
    </div>

    <!-- Per-agent table -->
    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      {#if rows.length === 0}
        <EmptyState
          type="list"
          title={loading ? 'Loading…' : 'No agent heartbeats yet'}
          description="Agents start sending heartbeats every 30s once they connect. If you've installed the agent and waited >2 minutes without a row appearing, check the agent's `oplog` shipping config."
        />
      {:else}
        <DataTable
          data={rows}
          columns={[
            { key: 'agent_id',                 label: 'Agent ID',        width: '220px' },
            { key: 'status',                   label: 'Status',          width: '120px' },
            { key: 'seconds_since_heartbeat',  label: 'Last seen',       width: '120px' },
            { key: 'log_file_size',            label: 'Log size',        width: '120px' },
            { key: 'skew_seconds',             label: 'Clock skew',      width: '120px' },
            { key: 'uptime_s',                 label: 'Uptime',          width: '120px' },
            { key: 'last_hash',                label: 'Last hash' },
          ]}
          compact
        >
          {#snippet render({ col, row })}
            {#if col.key === 'agent_id'}
              <span class="font-mono text-[var(--fs-micro)] text-text-secondary truncate">{row.agent_id}</span>
            {:else if col.key === 'status'}
              {@const b = statusBadge(row.status)}
              <Badge variant={b.variant} size="xs">{b.label}</Badge>
            {:else if col.key === 'seconds_since_heartbeat'}
              <span class="font-mono text-[var(--fs-micro)] {row.seconds_since_heartbeat > 90 ? 'text-warning' : 'text-text-muted'}">{fmtAgo(row.seconds_since_heartbeat)} ago</span>
            {:else if col.key === 'log_file_size'}
              <span class="font-mono text-[var(--fs-micro)] text-text-muted">{row.log_file_size.toLocaleString()} B</span>
            {:else if col.key === 'skew_seconds'}
              <span class="font-mono text-[var(--fs-micro)] {Math.abs(row.skew_seconds) > 60 ? 'text-warning' : 'text-text-muted'}">{row.skew_seconds.toFixed(1)}s</span>
            {:else if col.key === 'uptime_s'}
              <span class="font-mono text-[var(--fs-micro)] text-text-muted">{fmtAgo(row.uptime_s)}</span>
            {:else if col.key === 'last_hash'}
              <span class="font-mono text-[var(--fs-micro)] text-text-muted truncate" title={row.last_hash}>{row.last_hash?.slice(0, 16) ?? '—'}{row.last_hash ? '…' : ''}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>

    <p class="text-[var(--fs-micro)] text-text-muted leading-relaxed">
      <strong class="text-text-secondary">Status rubric:</strong> OK = heartbeat &lt;90s · STALE = 90s–2h · DARK = &gt;2h silent · TAMPERED = clock skew &gt;5min OR log truncation detected. Tamper events also fire <span class="font-mono text-accent">tamper:*</span> bus events visible in <span class="font-mono text-accent">/alerts</span>.
    </p>
  </div>
</PageLayout>
