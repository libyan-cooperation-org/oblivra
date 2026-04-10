<!--
  OBLIVRA — UEBA Panel (Svelte 5)
  User and Entity Behavior Analytics: Deviation detection and identity risk scoring.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Chart } from '@components/ui';
  import { User, Shield, AlertTriangle, Activity, Database } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const riskEntities = [
    { id: 'E-12', name: 'maverick', type: 'user', score: 88, deviation: 'Critical', status: 'monitored' },
    { id: 'E-14', name: 'svc_jenkins', type: 'service', score: 42, deviation: 'Moderate', status: 'nominal' },
    { id: 'E-15', name: 'operator_k', type: 'user', score: 94, deviation: 'Extreme', status: 'isolated' },
  ];
</script>

<PageLayout title="Behavioral Analytics" subtitle="UEBA: Identity risk scoring and behavioral deviation detection">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Baseline Profiles</Button>
    <Button variant="primary" size="sm">Logic Re-calibrate</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Avg Risk Score" value="12.4" trend="-0.4" variant="success" />
      <KPI title="Extreme Deviations" value={riskEntities.filter(e => e.score > 80).length} trend="Alerting" variant="error" />
      <KPI title="Observed Entities" value="1,422" trend="+12" />
      <KPI title="AI Confidence" value="94.2%" trend="Stable" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- High Risk Entities -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Prioritized Risk Entities (Calculated real-time)
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={riskEntities} columns={[
              { key: 'name', label: 'Entity Identity' },
              { key: 'score', label: 'Risk Score', width: '100px' },
              { key: 'deviation', label: 'Anomaly', width: '120px' },
              { key: 'status', label: 'State', width: '100px' },
              { key: 'action', label: '', width: '60px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'score'}
                   <div class="flex items-center gap-2">
                      <div class="flex-1 bg-surface-3 h-1 rounded-full overflow-hidden min-w-[40px]">
                         <div class="h-full {row.score > 80 ? 'bg-error' : 'bg-accent'}" style="width: {row.score}%"></div>
                      </div>
                      <span class="text-[11px] font-mono font-bold {row.score > 80 ? 'text-error' : 'text-text-primary'}">{row.score}</span>
                   </div>
                {:else if column.key === 'status'}
                   <Badge variant={row.status === 'isolated' ? 'error' : row.status === 'monitored' ? 'warning' : 'success'}>{row.status}</Badge>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <User size={12} class="text-text-muted" />
                      <div class="flex flex-col">
                         <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                         <span class="text-[9px] text-text-muted uppercase tracking-tight">{row.type}</span>
                      </div>
                   </div>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs">Profile</Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Anomaly Distribution -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Top Anomaly Sources</div>
            <div class="space-y-4">
               <div>
                  <div class="flex justify-between text-[10px] mb-1">
                     <span class="text-text-secondary">Unusual Hours</span>
                     <span class="font-bold">42%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                     <div class="bg-accent h-full" style="width: 42%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1">
                     <span class="text-text-secondary">Geo-Drift</span>
                     <span class="font-bold text-error">12%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                     <div class="bg-error h-full" style="width: 12%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1">
                     <span class="text-text-secondary">Process Lineage</span>
                     <span class="font-bold">24%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                     <div class="bg-accent h-full" style="width: 24%"></div>
                  </div>
               </div>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-2">
            <Activity size={32} class="text-accent opacity-40 animate-pulse" />
            <span class="text-xs font-bold text-text-heading mt-2">AI ENGINE ENGAGED</span>
            <p class="text-[9px] text-text-muted max-w-[180px]">Our behavioral model is currently absorbing 12k events/sec to maintain identity baselines.</p>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
