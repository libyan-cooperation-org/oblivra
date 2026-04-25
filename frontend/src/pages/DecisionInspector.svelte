<!--
  OBLIVRA — Decision Inspector (Svelte 5)
  Inspecting autonomous platform decisions: Transparency and audit trails for SOAR and AI orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, DataTable, PopOutButton} from '@components/ui';
  import { Activity, Zap, Globe, Eye } from 'lucide-svelte';

  const decisions = [
    { id: 'D-101', action: 'Terminate Process 1422', reason: 'High Entropy Exit', confidence: 0.98, time: '2m ago' },
    { id: 'D-102', action: 'BGP Route Sever', reason: 'Anomalous Egress', confidence: 0.94, time: '14m ago' },
    { id: 'D-103', action: 'Rotate Vault Keys', reason: 'Policy Trigger', confidence: 1.0, time: '1h ago' },
  ];

  const columns = [
    { key: 'action', label: 'Orchestrated Action' },
    { key: 'reason', label: 'Logic Reason', width: '200px' },
    { key: 'confidence', label: 'Confidence', width: '120px' },
    { key: 'time', label: 'Delta', width: '80px' },
    { key: 'action_btn', label: '', width: '60px' },
  ];
</script>

<PageLayout title="Decision Inspector" subtitle="Autonomous transparency: Auditing cognitive decisions and automated containment actions">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Logic Re-Evaluation</Button>
     <Button variant="primary" size="sm" icon="🧠">Decision Audit</Button>
      <PopOutButton route="/decisions" title="Decision Inspector" />
    {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Total Decisions" value="1.2k" trend="stable" trendValue="24H" />
      <KPI label="Auto-Containment" value="48" trend="up" trendValue="Critical" variant="critical" />
      <KPI label="Logic Accuracy" value="99.9%" trend="stable" trendValue="Stable" variant="success" />
      <KPI label="Human Override" value="0.2%" trend="down" trendValue="Low" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Cognitive Decision Ledger
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={decisions} {columns} compact>
              {#snippet render({ col, row, value })}
                {#if col.key === 'confidence'}
                   <div class="flex items-center gap-2">
                      <div class="flex-1 h-1 bg-surface-3 rounded-full overflow-hidden w-12">
                         <div class="h-full bg-accent" style="width: {row.confidence * 100}%"></div>
                      </div>
                      <span class="text-[10px] font-mono opacity-60">{(row.confidence * 100).toFixed(0)}%</span>
                   </div>
                {:else if col.key === 'action'}
                   <div class="flex items-center gap-2">
                      <Zap size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{value}</span>
                   </div>
                {:else if col.key === 'action_btn'}
                   <Button variant="ghost" size="xs"><Eye size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary border-dashed rounded-md p-6 flex flex-col items-center justify-center text-center gap-3 shadow-card">
            <Globe size={32} class="text-accent opacity-40" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Global Correlation</h4>
            <p class="text-[10px] text-text-muted max-w-[150px]">Decisions are synchronized across regional clusters for uniform containment logic.</p>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Decision Engine Entropy
            </div>
            <div class="h-32 flex items-end justify-between px-2 gap-1">
               {#each Array(10) as _, i}
                  <div class="flex-1 bg-accent/20 rounded-t-sm" style="height: {20 + Math.sin(i * 0.7) * 35 + 40}%"></div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
