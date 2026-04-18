<!--
  OBLIVRA — Configuration Risk (Svelte 5)
  System compliance and configuration drift: Assessing the security posture of platform-wide settings.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield, Zap, Activity, ShieldCheck, Database, Settings, AlertTriangle } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const risks = [
    { id: 'CR-01', name: 'Open SSH Root Auth', severity: 'High', status: 'vulnerable', domain: 'Operations' },
    { id: 'CR-02', name: 'Weak Vault Entropy', severity: 'Critical', status: 'warning', domain: 'Security' },
    { id: 'CR-03', name: 'Unsigned Script Exec', severity: 'Medium', status: 'baseline', domain: 'Orchestration' },
  ];
</script>

<PageLayout title="Configuration Risk" subtitle="Platform posture assessment: Identifying configuration drift and regulatory non-compliance across the fleet">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Baseline All Nodes</Button>
     <Button variant="primary" size="sm" icon="🛡️">Remediate Critical</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Risk Factor" value="0.22" trend="-0.04" variant="success" />
      <KPI title="Active Vulnerabilities" value="3" trend="Critical" variant="error" />
      <KPI title="Compliance Score" value="94%" trend="Optimized" variant="success" />
      <KPI title="OBLIVRA Posture" value="HARDENED" trend="NOMINAL" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Risk Inventory -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Configuration Posture Registry
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={risks} columns={[
              { key: 'name', label: 'Security Metric' },
              { key: 'severity', label: 'Gravity', width: '100px' },
              { key: 'domain', label: 'Domain', width: '120px' },
              { key: 'status', label: 'Compliance', width: '120px' },
              { key: 'action', label: '', width: '60px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'status'}
                   <div class="flex items-center gap-2">
                      <div class="w-2 h-2 rounded-full {row.status === 'vulnerable' ? 'bg-error animate-pulse' : row.status === 'warning' ? 'bg-warning' : 'bg-success'}"></div>
                      <span class="text-[10px] font-bold uppercase">{row.status}</span>
                   </div>
                {:else if column.key === 'severity'}
                   <Badge variant={row.severity === 'Critical' ? 'error' : row.severity === 'High' ? 'warning' : 'info'}>{row.severity.toUpperCase()}</Badge>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Settings size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                   </div>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs"><Shield size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Posture Visuals -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 border-dashed shadow-sm">
            <AlertTriangle size={48} class="text-warning opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Configuration Drift</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">OBLIVRA monitors disk and memory configuration in real-time to prevent unauthorized persistence mechanisms.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Platform Resilience
            </div>
            <div class="space-y-4">
                {#each Array(3) as _, i}
                   <div class="flex gap-3 items-start opacity-70">
                      <div class="w-1 h-1 rounded-full bg-accent mt-2"></div>
                      <div class="flex flex-col">
                         <span class="text-[10px] font-bold text-text-heading">Compliance Block {i+1} verified</span>
                         <span class="text-[8px] text-text-muted font-mono">0.0ms drift detected</span>
                      </div>
                   </div>
                {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
