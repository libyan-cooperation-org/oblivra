<!--
  OBLIVRA — Audit Log (Phase 33).

  Lists the most recent ~200 audit_logs rows. Powered by the existing
  GET /api/v1/audit/log endpoint (see rest.go handleAuditLog), which
  is fed by:
    - destructive-action bus subscriber (api_service.go) — logs
      `disaster:*`, `tenant:deleted`, `agent:quarantined`, `crisis:*`,
      `evidence:bulk_sealed`, `licensing:bypass_attempt`
    - REST handlers' explicit `appendAuditEntry()` calls (settings,
      suppression, evidence-seal, NBA recommendations, etc.)

  Design notes:
   • This is the read-only operator-facing surface. Compliance officers
     export via /api/v1/audit/packages/* (separate page lives at
     /compliance).
   • Filter by event-type prefix (settings.* / agent.* / evidence.*)
     because operators typically need "show me every settings change
     this week" not a 200-row scroll.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, DataTable, EmptyState, Input } from '@components/ui';
  import { ScrollText, RefreshCw } from 'lucide-svelte';
  import { apiFetch } from '@lib/apiClient';
  import { appStore } from '@lib/stores/app.svelte';

  interface AuditEntry {
    id?: string;
    actor?: string;
    event_type?: string;
    target?: string;
    details?: string | Record<string, any>;
    timestamp?: string;
    created_at?: string;
    ip_address?: string;
    user_agent?: string;
  }

  let entries = $state<AuditEntry[]>([]);
  let loading = $state(false);
  let filter = $state('');

  async function refresh() {
    loading = true;
    try {
      const res = await apiFetch('/api/v1/audit/log');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const body = await res.json();
      entries = (body.entries ?? body ?? []) as AuditEntry[];
    } catch (e: any) {
      appStore.notify('Audit log fetch failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  onMount(refresh);

  const filtered = $derived.by(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter((e) => {
      const hay = `${e.event_type ?? ''} ${e.actor ?? ''} ${e.target ?? ''} ${typeof e.details === 'string' ? e.details : ''}`.toLowerCase();
      return hay.includes(q);
    });
  });

  function severityForEvent(eventType?: string): 'critical' | 'warning' | 'info' | 'muted' {
    if (!eventType) return 'muted';
    if (eventType.startsWith('disaster') || eventType.includes('crisis') ||
        eventType.includes('quarantine') || eventType.includes('isolate')) return 'critical';
    if (eventType.includes('seal') || eventType.includes('delete') ||
        eventType.includes('destroy') || eventType.includes('wipe')) return 'warning';
    return 'info';
  }
</script>

<PageLayout
  title="Audit Log"
  subtitle="Sealed operator timeline · last 200 events"
>
  {#snippet toolbar()}
    <Input variant="search" bind:value={filter} placeholder="Filter event-type, actor, target…" class="w-64" />
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-3">
    <div class="text-[var(--fs-micro)] text-text-muted leading-relaxed flex items-center gap-2">
      <ScrollText size={11} class="text-accent" />
      Every row in this table is also written to the SQLite <span class="font-mono text-text-secondary">audit_logs</span> table; it survives restart and is included in compliance exports under
      <span class="font-mono text-accent">/compliance</span>.
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      {#if filtered.length === 0}
        <EmptyState
          type="list"
          title={loading ? 'Loading…' : (filter ? `No events match "${filter}"` : 'No audit events recorded')}
          description={filter
            ? 'Clear the filter to see all 200 most-recent events.'
            : 'Audit entries are written when settings change, when alerts are suppressed, when crisis mode arms, when hosts are quarantined, and when evidence is sealed.'}
        />
      {:else}
        <DataTable
          data={filtered}
          columns={[
            { key: 'timestamp',  label: 'Timestamp',  width: '180px' },
            { key: 'event_type', label: 'Event',      width: '220px' },
            { key: 'actor',      label: 'Actor',      width: '180px' },
            { key: 'target',     label: 'Target',     width: '200px' },
            { key: 'details',    label: 'Details' },
          ]}
          compact
        >
          {#snippet render({ col, row })}
            {#if col.key === 'timestamp'}
              <span class="font-mono text-[var(--fs-micro)] text-text-muted">
                {(row.timestamp ?? row.created_at ?? '').slice(0, 19).replace('T', ' ')}
              </span>
            {:else if col.key === 'event_type'}
              <Badge variant={severityForEvent(row.event_type)} size="xs">
                {row.event_type ?? '—'}
              </Badge>
            {:else if col.key === 'actor'}
              <span class="font-mono text-[var(--fs-label)] text-text-secondary">{row.actor ?? '—'}</span>
            {:else if col.key === 'target'}
              <span class="font-mono text-[var(--fs-micro)] text-text-muted truncate" title={row.target}>{row.target ?? '—'}</span>
            {:else if col.key === 'details'}
              <span class="text-[var(--fs-micro)] text-text-secondary truncate" title={typeof row.details === 'string' ? row.details : JSON.stringify(row.details)}>
                {typeof row.details === 'string' ? row.details : JSON.stringify(row.details ?? {})}
              </span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>
  </div>
</PageLayout>
