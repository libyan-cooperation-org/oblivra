<!-- Multi-Tenant Admin — bound to MultiTenantStore. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Building2, RefreshCw } from 'lucide-svelte';
  import { tenantStore } from '@lib/stores/tenant.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (typeof (tenantStore as any).init === 'function') await (tenantStore as any).init();
      else if (typeof (tenantStore as any).load === 'function') await (tenantStore as any).load();
    } catch (e: any) {
      appStore.notify(`Tenant load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }
  onMount(refresh);

  let metrics = $derived((tenantStore as any).platformMetrics ?? {});
  let tenants = $derived((tenantStore as any).tenants ?? []);
</script>

<PageLayout title="Multi-Tenant Admin" subtitle="Cross-tenant governance and platform metrics">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/multi-tenant-admin" title="Multi-Tenant Admin" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <KPI label="Tenants" value={tenants.length.toString()} variant="accent" />
      <KPI label="Total Agents" value={(metrics?.totalAgents ?? 0).toString()} variant="muted" />
      <KPI label="Active Incidents" value={(metrics?.totalIncidents ?? 0).toString()} variant={(metrics?.totalIncidents ?? 0) > 0 ? 'warning' : 'muted'} />
      <KPI label="Storage" value={(metrics?.totalStorage ?? '—').toString()} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Building2 size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Tenant Roster</span>
      </div>
      <DataTable data={tenants} columns={[
        { key: 'name',   label: 'Tenant' },
        { key: 'mode',   label: 'Mode',  width: '120px' },
        { key: 'tier',   label: 'Tier',  width: '100px' },
        { key: 'agents', label: 'Agents', width: '90px' },
        { key: 'eps',    label: 'EPS',    width: '90px' },
        { key: 'health', label: 'Health', width: '90px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'mode'}<Badge variant={row.mode === 'SOVEREIGN' ? 'success' : 'info'} size="xs">{row.mode}</Badge>
          {:else if col.key === 'health'}<span class="font-mono text-[10px] {(row.health ?? 0) >= 90 ? 'text-success' : (row.health ?? 0) >= 70 ? 'text-warning' : 'text-error'}">{row.health ?? '—'}%</span>
          {:else}<span class="text-[11px] font-mono">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if tenants.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No tenants registered.'}</div>{/if}
    </div>
  </div>
</PageLayout>
