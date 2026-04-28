<!--
  DSR Console — operator view + admin queue for GDPR / CCPA data
  subject requests (see docs/security/dpia.md §5.2). Backed by the
  REST endpoints we ship at /api/v1/dsr/requests:
    POST /api/v1/dsr/requests           file a request
    GET  /api/v1/dsr/requests           list per tenant
    POST /api/v1/dsr/requests/:id/fulfill   admin: execute access/deletion
    POST /api/v1/dsr/requests/:id/reject    admin: reject
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import {
    PageLayout,
    KPI,
    Badge,
    Button,
    DataTable,
    Input,
    Tabs,
    PopOutButton,
    EmptyState,
  } from '@components/ui';
  import { Scale, Shield, RefreshCw, Plus, Check, X } from 'lucide-svelte';
  import { apiFetch, apiPostJSON } from '@lib/apiClient';
  import { appStore } from '@lib/stores/app.svelte';

  type DSR = {
    id: string;
    tenant_id: string;
    subject_id: string;
    request_type: 'access' | 'deletion';
    reason?: string;
    requester?: string;
    verification?: string;
    status: 'pending' | 'fulfilled' | 'rejected';
    created_at: string;
    resolved_at?: string;
    resolved_by?: string;
    resolution_notes?: string;
  };

  let requests = $state<DSR[]>([]);
  let loading = $state(false);
  let activeTab = $state<'pending' | 'fulfilled' | 'rejected' | 'all'>('pending');

  // New-request form state.
  let showNew = $state(false);
  let nSubject = $state('');
  let nType = $state<'access' | 'deletion'>('access');
  let nReason = $state('');
  let nVerify = $state('');

  // Fulfillment-result state for the access path (returns inline records).
  let lastExport = $state<{ id: string; records: any[] } | null>(null);

  async function refresh() {
    loading = true;
    try {
      const res = await apiFetch('/api/v1/dsr/requests');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      requests = (data.requests ?? []) as DSR[];
    } catch (e: any) {
      appStore.notify('DSR list failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  async function fileRequest() {
    if (!nSubject.trim()) {
      appStore.notify('Subject id required', 'warning');
      return;
    }
    try {
      const res = await apiPostJSON('/api/v1/dsr/requests', {
        subject_id: nSubject,
        request_type: nType,
        reason: nReason,
        verification: nVerify,
      });
      if (!res.ok) {
        const txt = await res.text().catch(() => '');
        throw new Error(`HTTP ${res.status}: ${txt}`);
      }
      appStore.notify('DSR request filed', 'success');
      showNew = false;
      nSubject = '';
      nReason = '';
      nVerify = '';
      void refresh();
    } catch (e: any) {
      appStore.notify('File DSR failed', 'error', e?.message ?? String(e));
    }
  }

  async function fulfill(d: DSR) {
    if (d.request_type === 'deletion') {
      if (!confirm(`Crypto-wipe ALL records for ${d.subject_id}?\n\nThis is irreversible. Audit-log entries are pseudonymised (legal retention).`)) return;
    }
    try {
      const res = await apiPostJSON(`/api/v1/dsr/requests/${encodeURIComponent(d.id)}/fulfill`, {});
      if (!res.ok) {
        const txt = await res.text().catch(() => '');
        throw new Error(`HTTP ${res.status}: ${txt}`);
      }
      const body = await res.json();
      if (d.request_type === 'access') {
        lastExport = { id: d.id, records: body.records ?? [] };
      }
      appStore.notify(`DSR ${d.id} fulfilled`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify('Fulfill failed', 'error', e?.message ?? String(e));
    }
  }

  async function reject(d: DSR) {
    const reason = prompt(`Reject DSR ${d.id}. Reason?`);
    if (!reason) return;
    try {
      const res = await apiPostJSON(`/api/v1/dsr/requests/${encodeURIComponent(d.id)}/reject`, { reason });
      if (!res.ok) {
        const txt = await res.text().catch(() => '');
        throw new Error(`HTTP ${res.status}: ${txt}`);
      }
      appStore.notify(`DSR ${d.id} rejected`, 'warning');
      void refresh();
    } catch (e: any) {
      appStore.notify('Reject failed', 'error', e?.message ?? String(e));
    }
  }

  function downloadExport() {
    if (!lastExport) return;
    const blob = new Blob([JSON.stringify(lastExport, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `dsr-${lastExport.id}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }

  let visible = $derived(activeTab === 'all' ? requests : requests.filter((r) => r.status === activeTab));

  let stats = $derived.by(() => ({
    total: requests.length,
    pending: requests.filter((r) => r.status === 'pending').length,
    deletions: requests.filter((r) => r.request_type === 'deletion').length,
    overdue: requests.filter((r) => {
      if (r.status !== 'pending') return false;
      const t = Date.parse(r.created_at);
      // GDPR 30-day response window.
      return Number.isFinite(t) && Date.now() - t > 30 * 86_400_000;
    }).length,
  }));

  onMount(refresh);
</script>

<PageLayout title="Data Subject Requests" subtitle="GDPR Art. 15 / 17 + CCPA §1798.105 / §1798.110 workflow">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
    <Button variant="primary" size="sm" icon={Plus} onclick={() => (showNew = !showNew)}>
      File request
    </Button>
    <PopOutButton route="/dsr" title="DSR Console" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
      <KPI label="Total Requests" value={stats.total.toString()} variant="accent" />
      <KPI label="Pending" value={stats.pending.toString()} variant={stats.pending > 0 ? 'warning' : 'muted'} />
      <KPI label="Deletion Requests" value={stats.deletions.toString()} variant="muted" />
      <KPI label="Past 30-day SLA" value={stats.overdue.toString()} variant={stats.overdue > 0 ? 'critical' : 'muted'} />
    </div>

    {#if showNew}
      <div class="border border-border-primary rounded-md bg-surface-1 p-4 space-y-3">
        <h3 class="text-xs uppercase tracking-widest font-bold flex items-center gap-2">
          <Scale size={14} /> File new DSR
        </h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Subject ID</div>
            <Input bind:value={nSubject} placeholder="email or stable user id" />
          </div>
          <div>
            <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Type</div>
            <select class="w-full bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs" bind:value={nType}>
              <option value="access">Access (Art. 15)</option>
              <option value="deletion">Deletion (Art. 17)</option>
            </select>
          </div>
          <div class="md:col-span-2">
            <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Reason / context</div>
            <Input bind:value={nReason} placeholder="Optional — what prompted the request" />
          </div>
          <div class="md:col-span-2">
            <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Verification reference</div>
            <Input bind:value={nVerify} placeholder="Internal ticket id, email-verification proof, etc." />
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onclick={() => (showNew = false)}>Cancel</Button>
          <Button variant="primary" onclick={fileRequest}>File request</Button>
        </div>
      </div>
    {/if}

    {#if lastExport}
      <div class="border border-cyan-400/30 bg-cyan-400/5 rounded-md p-4 space-y-2">
        <div class="flex items-center justify-between">
          <h3 class="text-xs uppercase tracking-widest font-bold">Access export ready — DSR {lastExport.id}</h3>
          <Button variant="primary" size="sm" onclick={downloadExport}>Download JSON</Button>
        </div>
        <p class="text-[11px] text-text-muted">{lastExport.records.length} record(s) bundled. The audit log records this export.</p>
      </div>
    {/if}

    <Tabs
      tabs={[
        { id: 'pending',   label: `Pending (${stats.pending})` },
        { id: 'fulfilled', label: 'Fulfilled' },
        { id: 'rejected',  label: 'Rejected' },
        { id: 'all',       label: 'All' },
      ]}
      bind:active={activeTab}
    />

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      {#if visible.length === 0}
        <EmptyState
          title={loading ? 'Loading…' : `No ${activeTab === 'all' ? '' : activeTab} requests`}
          description="Subjects file DSRs via the operator. Use the &quot;File request&quot; button above to record one on their behalf."
          icon="📜"
        />
      {:else}
        <DataTable
          data={visible}
          columns={[
            { key: 'id',           label: 'ID',         width: '200px' },
            { key: 'subject_id',   label: 'Subject',    width: '220px' },
            { key: 'request_type', label: 'Type',       width: '110px' },
            { key: 'created_at',   label: 'Filed',      width: '160px' },
            { key: 'status',       label: 'Status',     width: '110px' },
            { key: 'requester',    label: 'Filed by',   width: '160px' },
            { key: 'actions',      label: '',           width: '160px' },
          ]}
          compact
        >
          {#snippet render({ col, row })}
            {#if col.key === 'id'}
              <span class="font-mono text-[10px] text-accent">{row.id}</span>
            {:else if col.key === 'subject_id'}
              <span class="font-mono text-[11px]">{row.subject_id}</span>
            {:else if col.key === 'request_type'}
              <Badge variant={row.request_type === 'deletion' ? 'critical' : 'info'} size="xs">
                {row.request_type}
              </Badge>
            {:else if col.key === 'created_at'}
              <span class="font-mono text-[10px] text-text-muted">{(row.created_at ?? '').slice(0, 19)}</span>
            {:else if col.key === 'status'}
              <Badge
                variant={row.status === 'fulfilled' ? 'success' : row.status === 'rejected' ? 'muted' : 'warning'}
                size="xs"
              >{row.status}</Badge>
            {:else if col.key === 'actions'}
              {#if row.status === 'pending'}
                <div class="flex gap-1 justify-end">
                  <Button variant="ghost" size="xs" onclick={() => fulfill(row)}>
                    <Check size={11} class="mr-0.5" />Fulfill
                  </Button>
                  <Button variant="ghost" size="xs" onclick={() => reject(row)}>
                    <X size={11} class="mr-0.5" />Reject
                  </Button>
                </div>
              {:else if row.resolution_notes}
                <span class="text-[10px] text-text-muted truncate" title={row.resolution_notes}>
                  {row.resolution_notes}
                </span>
              {/if}
            {:else}
              <span class="text-[11px]">{row[col.key] ?? '—'}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>

    <div class="text-[10px] text-text-muted italic flex items-center gap-1.5 shrink-0">
      <Shield size={10} />
      Every state transition is sealed in audit_logs (event_type = "dsr.*"). See docs/security/dpia.md.
    </div>
  </div>
</PageLayout>
