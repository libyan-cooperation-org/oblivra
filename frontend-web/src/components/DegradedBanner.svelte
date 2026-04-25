<script lang="ts">
  // DegradedBanner — operator-visible notice when the ingestion pipeline is
  // running hot. Backed by /api/v1/health/load (Phase 22.1 graceful degradation).
  //
  // Polls every 10s. On `degraded` shows an amber banner; on `critical` shows
  // a red banner. Hides itself on `healthy` or `unknown` (cold boot, pipeline
  // not initialised) so it never flashes during normal operation.

  import { onMount, onDestroy } from 'svelte';
  import { request } from '../services/api';

  type LoadStatus = 'healthy' | 'degraded' | 'critical' | 'unknown';

  interface HealthLoad {
    status: LoadStatus;
    queue_fill_pct?: number;
    events_per_second?: number;
    dropped_events?: number;
    collected_at?: string;
  }

  let snapshot: HealthLoad = $state({ status: 'unknown' });
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let dismissed = $state(false);

  // Reset the dismiss flag whenever the status changes — operators who
  // dismissed a "degraded" banner should still see a "critical" banner.
  let lastStatus: LoadStatus = 'unknown';

  async function poll() {
    try {
      const next = await request<HealthLoad>('/health/load');
      if (next.status !== lastStatus) {
        dismissed = false;
        lastStatus = next.status;
      }
      snapshot = next;
    } catch {
      // Network errors leave the previous snapshot in place. We'd rather
      // under-warn than render a misleading banner during a transient blip.
    }
  }

  onMount(() => {
    poll();
    pollTimer = setInterval(poll, 10_000);
  });

  onDestroy(() => {
    if (pollTimer) clearInterval(pollTimer);
  });

  const visible = $derived(
    !dismissed && (snapshot.status === 'degraded' || snapshot.status === 'critical')
  );

  const tone = $derived(snapshot.status === 'critical' ? 'critical' : 'degraded');
</script>

{#if visible}
  <div
    class="degraded-banner"
    class:critical={tone === 'critical'}
    role="alert"
    aria-live="assertive"
  >
    <span class="badge">{snapshot.status.toUpperCase()}</span>
    <span class="message">
      {#if tone === 'critical'}
        Pipeline stalled — buffer near capacity, events may be dropped.
      {:else}
        Pipeline under load — degraded performance, ingestion still nominal.
      {/if}
    </span>
    <span class="metrics">
      EPS: {snapshot.events_per_second ?? 0}
      · Queue: {(snapshot.queue_fill_pct ?? 0).toFixed(1)}%
      {#if (snapshot.dropped_events ?? 0) > 0}
        · Dropped: {snapshot.dropped_events}
      {/if}
    </span>
    <button class="dismiss" onclick={() => (dismissed = true)} aria-label="Dismiss banner">
      ×
    </button>
  </div>
{/if}

<style>
  .degraded-banner {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem 1rem;
    background: rgba(245, 139, 0, 0.12);
    border-bottom: 1px solid rgba(245, 139, 0, 0.4);
    color: #f58b00;
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    z-index: 100;
  }

  .degraded-banner.critical {
    background: rgba(224, 64, 64, 0.15);
    border-bottom-color: rgba(224, 64, 64, 0.5);
    color: #e04040;
  }

  .badge {
    font-weight: 700;
    letter-spacing: 0.05em;
    padding: 0.125rem 0.5rem;
    border: 1px solid currentColor;
    border-radius: 2px;
  }

  .message {
    flex: 1;
  }

  .metrics {
    opacity: 0.8;
    white-space: nowrap;
  }

  .dismiss {
    background: transparent;
    border: none;
    color: currentColor;
    cursor: pointer;
    font-size: 1.25rem;
    line-height: 1;
    padding: 0 0.25rem;
    opacity: 0.7;
  }

  .dismiss:hover {
    opacity: 1;
  }
</style>
