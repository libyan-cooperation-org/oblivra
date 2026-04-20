<!--
  OBLIVRA — Compliance Hub (Svelte 5)
  Real-time regulatory posture and continuous control monitoring.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, ProgressBar } from '@components/ui';
  import { Shield, CheckCircle, AlertTriangle, Clock, RefreshCw, Filter, Download } from 'lucide-svelte';
  import { complianceStore } from '@lib/stores/compliance.svelte';
  import { onMount } from 'svelte';

  const controls = $derived(complianceStore.controls);
  const stats = $derived(complianceStore.stats);

  onMount(() => {
    complianceStore.refresh();
  });
</script>

<PageLayout title="Compliance Hub" subtitle="Continuous regulatory monitoring and autonomous control validation">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={RefreshCw} onclick={() => complianceStore.validateAll()} loading={complianceStore.loading}>VALIDATE ALL</Button>
      <Button variant="primary" size="sm" icon={Download}>GENERATE REPORT</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- POSTURE STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Global Compliance</div>
            <div class="text-xl font-mono font-bold text-success">{stats.global_score}%</div>
            <div class="text-[9px] text-success mt-1">▲ Real-time posture</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Breaches</div>
            <div class="text-xl font-mono font-bold text-error">{stats.active_breaches.toString().padStart(2, '0')}</div>
            <div class="text-[9px] text-error mt-1 {stats.active_breaches > 0 ? 'animate-pulse' : ''}">Critical breaches detected</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Controls Monitored</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.controls_monitored}</div>
            <div class="text-[9px] text-text-muted mt-1">Continuous validation active</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Audit Readiness</div>
            <div class="text-xl font-mono font-bold text-accent">{stats.audit_readiness}</div>
            <div class="text-[9px] text-accent mt-1">Real-time evidence locked</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- CONTROL LEDGER -->
        <div class="flex-1 flex flex-col min-w-0">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <Shield size={14} class="text-accent" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Regulatory Control Ledger</span>
                </div>
                <div class="flex gap-2">
                    <Button variant="ghost" size="xs" icon={Filter}>FRAMEWORKS</Button>
                </div>
            </div>

            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={controls} 
                    columns={[
                        { key: 'id', label: 'ID', width: '80px' },
                        { key: 'framework', label: 'FRAMEWORK', width: '120px' },
                        { key: 'control', label: 'CONTROL_DESCRIPTION' },
                        { key: 'coverage', label: 'COVERAGE', width: '140px' },
                        { key: 'status', label: 'STATUS', width: '100px' },
                        { key: 'last_audit', label: 'VALIDATED', width: '100px' }
                    ]} 
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'id'}
                            <span class="text-[10px] font-mono font-bold text-text-heading">{row.id}</span>
                        {:else if col.key === 'framework'}
                            <Badge variant="info" size="xs">{row.framework}</Badge>
                        {:else if col.key === 'control'}
                            <span class="text-[10px] font-bold text-text-secondary leading-tight">{row.control}</span>
                        {:else if col.key === 'coverage'}
                            <div class="flex items-center gap-2 w-full">
                                <ProgressBar value={Number(row.coverage)} variant={Number(row.coverage) > 90 ? 'success' : Number(row.coverage) > 70 ? 'warning' : 'error'} size="xs" />
                                <span class="text-[9px] font-mono text-text-muted w-8">{row.coverage}%</span>
                            </div>
                        {:else if col.key === 'status'}
                            <Badge variant={row.status === 'compliant' ? 'success' : row.status === 'warning' ? 'warning' : 'critical'} size="xs" dot>
                                {row.status.toUpperCase()}
                            </Badge>
                        {:else if col.key === 'last_audit'}
                            <div class="flex items-center gap-1">
                                <Clock size={10} class="text-text-muted" />
                                <span class="text-[9px] font-mono text-text-muted">{row.last_audit}</span>
                            </div>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- RIGHT: AUDIT TRENDS -->
        <div class="w-80 bg-surface-2 border-l border-border-primary flex flex-col shrink-0">
            <div class="p-4 border-b border-border-primary space-y-4">
                <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Compliance by Framework</span>
                <div class="space-y-3">
                    {#each [['NIST 800-53', 98], ['SOC2', 82], ['ISO 27001', 100], ['GDPR', 45]] as [fw, val]}
                        <div class="space-y-1.5">
                            <div class="flex justify-between text-[8px] font-mono uppercase">
                                <span class="text-text-muted">{fw}</span>
                                <span class="text-text-heading font-bold">{val}%</span>
                            </div>
                            <div class="h-1 bg-surface-1 rounded-full overflow-hidden">
                                <div class="h-full {Number(val) > 90 ? 'bg-success' : Number(val) > 70 ? 'bg-warning' : 'bg-error'}" style="width: {val}%"></div>
                            </div>
                        </div>
                    {/each}
                </div>
            </div>

            <div class="p-4 space-y-4 flex-1">
                <div class="flex items-center justify-between">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Evidence Lock Status</span>
                    <CheckCircle size={14} class="text-success" />
                </div>
                <div class="p-3 bg-surface-3 border border-border-primary rounded-sm space-y-2">
                    <div class="text-[9px] font-mono text-text-muted leading-relaxed italic">
                        All compliance evidence is automatically hashed and sealed in the Evidence Vault using Root Key #14A.
                    </div>
                    <div class="flex items-center gap-2 text-[8px] font-mono text-success">
                        <Shield size={10} />
                        <span>INTEGRITY VERIFIED</span>
                    </div>
                </div>
            </div>

            <div class="mt-auto p-4 bg-surface-3 border-t border-border-primary">
                 <Button variant="secondary" size="sm" class="w-full font-bold uppercase tracking-widest">
                    <AlertTriangle size={14} class="mr-2 text-warning" /> VIEW DRIFT REPORT
                 </Button>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>POSTURE:</span>
            <span class="text-success font-bold uppercase">STABLE</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>AUDIT_MODE:</span>
            <span class="text-accent font-bold uppercase">Continuous</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">COMPLY_CORE v1.2.1</div>
    </div>
  </div>
</PageLayout>
