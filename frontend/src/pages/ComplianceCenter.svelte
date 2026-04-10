<!--
  OBLIVRA — Compliance Center (Svelte 5)
  Governance and Regulatory alignment: SOC2, HIPAA, NIST monitoring.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';
  import { Shield, CheckCircle, AlertTriangle, FileText, ClipboardList } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const frameworks = [
    { id: 'F-SOC2', name: 'SOC2 Type II', coverage: '94%', controls: 42, status: 'compliant' },
    { id: 'F-NIST', name: 'NIST 800-53', coverage: '82%', controls: 120, status: 'drift_detected' },
    { id: 'F-HIPAA', name: 'HIPAA Security', coverage: '100%', controls: 28, status: 'compliant' },
  ];
</script>

<PageLayout title="Governance & Compliance" subtitle="Framework alignment and regulatory continuous monitoring">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Evidence Locker</Button>
    <Button variant="primary" size="sm">Standard Audit</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Overall Compliance" value="92%" trend="+4%" variant="success" />
      <KPI title="Failed Controls" value="3" trend="Manual Action Req." variant="error" />
      <KPI title="Next Audit" value="12d" trend="Scheduled" />
      <KPI title="Evidence Coverage" value="98%" trend="Optimal" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Frameworks Table -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Active Compliance Frameworks
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={frameworks} columns={[
              { key: 'name', label: 'Framework' },
              { key: 'coverage', label: 'Coverage', width: '100px' },
              { key: 'controls', label: 'Controls', width: '100px' },
              { key: 'status', label: 'State', width: '120px' },
              { key: 'action', label: '', width: '60px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.status === 'compliant' ? 'success' : 'warning'}>{row.status.replace('_', ' ')}</Badge>
                {:else if column.key === 'coverage'}
                   <div class="flex items-center gap-2">
                      <div class="flex-1 bg-surface-3 h-1 rounded-full overflow-hidden min-w-[40px]">
                         <div class="bg-success h-full" style="width: {row.coverage}"></div>
                      </div>
                      <span class="text-[11px] font-mono font-bold">{row.coverage}</span>
                   </div>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2 font-bold text-text-heading text-[11px]">
                      <FileText size={14} class="text-accent" />
                      {row.name}
                   </div>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs">Audit</Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Compliance Health -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4 shadow-sm">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Top Drift Factors</div>
            <div class="space-y-4">
               <div>
                  <div class="flex justify-between text-[10px] mb-1">
                     <span class="text-text-secondary">Untracked Shadow Identity</span>
                     <span class="font-bold text-error">Critical</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                     <div class="bg-error h-full" style="width: 80%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1">
                     <span class="text-text-secondary">Manual Access Review Outdated</span>
                     <span class="font-bold">Warning</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                     <div class="bg-warning h-full" style="width: 40%"></div>
                  </div>
               </div>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-3">
            <CheckCircle size={32} class="text-success opacity-40" />
            <span class="text-xs font-bold text-text-heading mt-2">CONTINUOUS ATTESTATION</span>
            <p class="text-[9px] text-text-muted max-w-[180px]">Autonomous evidence collection is active. All SOC actions are currently hash-linked to regulatory requirements.</p>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
