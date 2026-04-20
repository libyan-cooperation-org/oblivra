<!--
  OBLIVRA — Credential Intelligence (Svelte 5)
  Monitoring credential leaks, risk scores, and identity vulnerabilities.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Key, Activity, Globe, RefreshCcw } from 'lucide-svelte';

  const credentials: Record<string, any>[] = [
    { id: 'C-901', identity: 'maverick@oblivra.sh', leakType: 'Public Pastebin', risk: 'High', status: 'monitored' },
    { id: 'C-902', identity: 'svc-bridge-alpha', leakType: 'None', risk: 'Low', status: 'verified' },
    { id: 'C-903', identity: 'iceman@oblivra.sh', leakType: 'Known Breach', risk: 'Critical', status: 'revoked' },
  ];
</script>

<PageLayout title="Credential Intelligence" subtitle="Identity risk orchestration: Monitoring credential leaks, exposed secrets and identity-based vulnerabilities across the global surface">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Rotate All Critical</Button>
     <Button variant="primary" size="sm" icon="🔑">Surface Scan</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Tracked Identities" value="142" trend="stable" trendValue="Nominal" />
      <KPI label="Exposed Secrets" value="3" trend="stable" trendValue="Critical" variant="critical" />
      <KPI label="Rotation Health" value="98%" trend="stable" trendValue="Optimal" variant="success" />
      <KPI label="Darknet Reach" value="DEEP" trend="stable" trendValue="Active" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Credential Ledger -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Global Identity Integrity Ledger
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={credentials} columns={[
              { key: 'identity', label: 'Identity / Principal' },
              { key: 'leakType', label: 'Potential Exposure', width: '180px' },
              { key: 'risk', label: 'Gravity', width: '100px' },
              { key: 'status', label: 'State', width: '100px' },
              { key: 'action', label: '', width: '60px' }
            ]} compact>
              {#snippet render({ col: column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.status === 'verified' ? 'success' : row.status === 'revoked' ? 'critical' : 'muted'}>{row.status.toUpperCase()}</Badge>
                {:else if column.key === 'risk'}
                   <Badge variant={row.risk === 'Critical' ? 'critical' : row.risk === 'High' ? 'warning' : 'info'}>{row.risk.toUpperCase()}</Badge>
                {:else if column.key === 'identity'}
                   <div class="flex items-center gap-2">
                      <Key size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.identity}</span>
                   </div>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs"><RefreshCcw size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Intel Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-3 border-dashed shadow-sm">
            <Globe size={32} class="text-accent opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Global Leak Monitoring</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">OBLIVRA correlates local identities against 14B+ known exposed records in real-time.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Exposure Velocity
            </div>
            <div class="space-y-4">
                {#each Array(3) as _, i}
                   <div class="flex justify-between items-center text-[10px]">
                      <span class="text-text-secondary">Market Source {i+1}</span>
                      <span class="font-bold text-success">CLEAN</span>
                   </div>
                {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
