<!--
  OBLIVRA — SOAR Panel (Svelte 5)
  Security Orchestration, Automation, and Response dashboard.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button, DataTable } from '@components/ui';
  import { Zap, Play, Settings, Activity, Clock } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const playbooks = [
    { name: 'Brute Force Auto-Contain', status: 'active', triggers: 142, success: '98%' },
    { name: 'Phishing URI Enrichment', status: 'active', triggers: 890, success: '100%' },
    { name: 'Crypto-Mining Kill Switch', status: 'paused', triggers: 5, success: '80%' },
    { name: 'Identity Drift Correction', status: 'testing', triggers: 0, success: 'N/A' },
  ];
</script>

<PageLayout title="SOAR Orchestrator" subtitle="Automated signal response and playbook execution engine">
  {#snippet toolbar()}
    <Button variant="primary" size="sm">New Playbook</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Automation Rate" value="84%" trend="+5%" variant="success" />
      <KPI title="Time Saved (MoMD)" value="122h" trend="Productive" variant="accent" />
      <KPI title="Active Playbooks" value={playbooks.filter(p => p.status === 'active').length} trend="Nominal" />
      <KPI title="Failure Rate" value="0.2%" trend="Low" variant="success" />
    </div>

    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
       <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
          Active Playbook Library
       </div>
       <div class="flex-1 overflow-auto">
          <DataTable data={playbooks} columns={[
            { key: 'name', label: 'Playbook Logic' },
            { key: 'status', label: 'Status', width: '100px' },
            { key: 'triggers', label: 'Executions', width: '100px' },
            { key: 'success', label: 'Efficiency', width: '100px' },
            { key: 'action', label: '', width: '100px' }
          ]} density="compact">
            {#snippet cell({ column, row })}
              {#if column.key === 'status'}
                 <Badge variant={row.status === 'active' ? 'success' : row.status === 'paused' ? 'warning' : 'info'}>{row.status}</Badge>
              {:else if column.key === 'name'}
                 <div class="flex items-center gap-2">
                    <Zap size={14} class={row.status === 'active' ? 'text-accent' : 'text-text-muted'} />
                    <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                 </div>
              {:else if column.key === 'triggers'}
                 <span class="text-[11px] font-mono text-text-secondary">{row.triggers}</span>
              {:else if column.key === 'action'}
                 <div class="flex gap-2">
                    <Button variant="ghost" size="xs">Edit</Button>
                    <Button variant="ghost" size="xs" class="text-accent"><Play size={12} /></Button>
                 </div>
              {:else}
                <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
              {/if}
            {/snippet}
          </DataTable>
       </div>
    </div>
  </div>
</PageLayout>
