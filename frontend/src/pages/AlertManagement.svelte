<!--
  OBLIVRA — Alert Management (Svelte 5)
  Detection rule orchestration and signal filtering.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, Spinner, Input } from '@components/ui';
  import { Zap, ShieldAlert, MoreHorizontal } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let searchQuery = $state('');
  let activeTab = $state('OPEN');
  let selectedSeverities = $state(['CRITICAL', 'HIGH']);
  let selectedAlert = $state<any>(null);

  const stats = $derived({
    total: alertStore.alerts.length,
    critical: alertStore.alerts.filter(a => a.severity === 'critical').length,
    unassigned: alertStore.alerts.filter(a => !a.action).length,
    mttr: '14.2m',
    fpRate: '8.4%'
  });

  const filteredAlerts = $derived(alertStore.alerts.filter(a => {
    const matchesSearch = a.title.toLowerCase().includes(searchQuery.toLowerCase()) || a.host.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesSeverity = selectedSeverities.includes(a.severity.toUpperCase());
    const matchesTab = activeTab === 'OPEN' ? a.status === 'open' : true; 
    return matchesSearch && matchesSeverity && matchesTab;
  }));

  function toggleSeverity(sev: string) {
    if (selectedSeverities.includes(sev)) {
        selectedSeverities = selectedSeverities.filter(s => s !== sev);
    } else {
        selectedSeverities = [...selectedSeverities, sev];
    }
  }

  function handleRowClick(row: any) {
    if (selectedAlert?.id === row.id) {
        selectedAlert = null;
    } else {
        selectedAlert = row;
    }
  }
</script>

<PageLayout title="Alert Management" subtitle="Detection pipeline orchestration and logic control">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Search alerts..." bind:value={searchQuery} class="w-64" />
      <Button variant="primary" size="sm" onclick={() => appStore.notify('Manual alert creation disabled', 'warning')}>Create Alert</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-5 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Open</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.total}</div>
            <div class="text-[9px] text-text-muted mt-1">+31 vs prev 4h</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Critical</div>
            <div class="text-xl font-mono font-bold text-error">{stats.critical}</div>
            <div class="text-[9px] text-error/80 mt-1 animate-pulse">▲ SLA BREACH RISK</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Unassigned</div>
            <div class="text-xl font-mono font-bold text-warning">{stats.unassigned}</div>
            <div class="text-[9px] text-text-muted mt-1">Needs triage</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg MTTR</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.mttr}</div>
            <div class="text-[9px] text-success mt-1">▼ -2.1m vs SLA</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">False Positive</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.fpRate}</div>
            <div class="text-[9px] text-text-muted mt-1">Last 7 days</div>
        </div>
    </div>

    <!-- TOOLBAR & TABS -->
    <div class="bg-surface-1 border-b border-border-primary p-2 flex items-center gap-4 shrink-0">
        <div class="flex bg-surface-2 border border-border-primary rounded-sm overflow-hidden">
            {#each ['OPEN', 'ACK', 'INVESTIGATING', 'CLOSED'] as tab}
                <button 
                    class="px-3 py-1 text-[9px] font-mono font-bold transition-colors {activeTab === tab ? 'bg-surface-3 text-accent border-b-2 border-accent' : 'text-text-muted hover:text-text-secondary'}"
                    onclick={() => activeTab = tab}
                >
                    {tab} ({tab === 'OPEN' ? stats.total : 0})
                </button>
            {/each}
        </div>

        <div class="flex items-center gap-1">
            {#each ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'] as sev}
                <button 
                    class="px-2 py-1 text-[8px] font-mono font-bold rounded-sm border transition-all {selectedSeverities.includes(sev) ? 'border-accent bg-accent/10 text-accent' : 'border-border-primary text-text-muted hover:border-border-hover'}"
                    onclick={() => toggleSeverity(sev)}
                >
                    {sev}
                </button>
            {/each}
        </div>

        <div class="ml-auto flex gap-2">
            <Button variant="secondary" size="xs" class="text-[9px] font-mono" onclick={() => appStore.notify('Bulk ACK triggered', 'info')}>ACK SELECTED</Button>
            <Button variant="secondary" size="xs" class="text-[9px] font-mono">ASSIGN</Button>
            <Button variant="danger" size="xs" class="text-[9px] font-mono" onclick={() => appStore.notify('Marked as False Positive', 'error')}>FALSE POS.</Button>
        </div>
    </div>

    <!-- ALERT TABLE -->
    <div class="flex-1 overflow-hidden relative">
        {#if alertStore.isLoading}
            <div class="absolute inset-0 bg-surface-1/50 backdrop-blur-xs z-20 flex items-center justify-center">
                <Spinner />
            </div>
        {/if}

        <div class="h-full overflow-auto">
            <DataTable 
                data={filteredAlerts} 
                onRowClick={handleRowClick}
                columns={[
                    { key: 'severity', label: 'SEV', width: '60px' },
                    { key: 'title', label: 'ALERT' },
                    { key: 'host', label: 'HOST', width: '120px' },
                    { key: 'id', label: 'RISK', width: '60px' },
                    { key: 'status', label: 'SLA', width: '80px' },
                    { key: 'status', label: 'STATUS', width: '80px' },
                    { key: 'id', label: '', width: '100px' }
                ]} 
                compact
            >
                {#snippet render({ col, row })}
                    {#if col.label === 'SEV'}
                        <div class="flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full {row.severity === 'critical' ? 'bg-error shadow-[0_0_4px_rgba(200,44,44,1)]' : row.severity === 'high' ? 'bg-warning' : 'bg-info'}"></div>
                            <span class="text-[8px] font-mono font-bold {row.severity === 'critical' ? 'text-error' : row.severity === 'high' ? 'text-warning' : 'text-info'} uppercase">
                                {row.severity.slice(0, 4)}
                            </span>
                        </div>
                    {:else if col.label === 'ALERT'}
                        <div class="flex flex-col py-0.5">
                            <span class="text-[11px] font-bold text-text-secondary leading-tight">{row.title}</span>
                            <span class="text-[9px] font-mono text-text-muted opacity-60">{row.id} · EDR-3847</span>
                        </div>
                    {:else if col.label === 'HOST'}
                        <span class="text-[10px] font-mono text-accent">{row.host}</span>
                    {:else if col.label === 'RISK'}
                        <div class="px-1.5 py-0.5 rounded-sm font-mono text-[9px] font-bold text-center {row.severity === 'critical' ? 'bg-error/10 text-error' : 'bg-warning/10 text-warning'}">
                            {row.severity === 'critical' ? '97' : '72'}
                        </div>
                    {:else if col.label === 'SLA'}
                        <div class="flex flex-col gap-0.5">
                            <div class="w-full h-0.5 bg-border-primary rounded-full overflow-hidden">
                                <div class="h-full bg-error" style="width: 85%"></div>
                            </div>
                            <span class="text-[8px] font-mono text-error animate-pulse">08:12</span>
                        </div>
                    {:else if col.label === 'STATUS'}
                        <Badge variant={row.status === 'open' ? 'accent' : 'warning'} size="xs" class="text-[8px] px-1.5">
                            {row.status.toUpperCase()}
                        </Badge>
                    {:else if col.label === ''}
                        <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-error border border-error/20" title="Isolate Host"><ShieldAlert size={12} /></button>
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-accent border border-accent/20" title="Run Playbook"><Zap size={12} /></button>
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-text-muted border border-border-primary" title="More"><MoreHorizontal size={12} /></button>
                        </div>
                    {/if}
                {/snippet}
            </DataTable>
        </div>
    </div>

    <!-- DETAIL DRAWER -->
    {#if selectedAlert}
        <div class="bg-surface-2 border-t border-border-primary h-64 shrink-0 grid grid-cols-3 gap-px bg-border-primary p-px shadow-2xl">
            <div class="bg-surface-2 p-4 overflow-auto">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Alert Details</div>
                <div class="space-y-2">
                    {#each Object.entries(selectedAlert).filter(([k]) => k !== 'action') as [key, val]}
                        <div class="flex items-start gap-2">
                            <span class="w-24 text-[9px] font-mono text-text-muted uppercase">{key}</span>
                            <span class="text-[10px] font-mono text-text-secondary">{val}</span>
                        </div>
                    {/each}
                </div>
            </div>
            <div class="bg-surface-2 p-4 overflow-auto">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Related Context</div>
                <div class="space-y-2">
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">Campaign</span>
                        <span class="text-[10px] font-mono text-error font-bold">FIN-THREAT-22</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">Incident</span>
                        <span class="text-[10px] font-mono text-accent">INC-2026-0419-007</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">UEBA Score</span>
                        <span class="text-[10px] font-mono text-warning">+4.2σ deviation</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">IOC Matches</span>
                        <span class="text-[10px] font-mono text-error font-bold">3 (TI feed)</span>
                    </div>
                </div>
            </div>
            <div class="bg-surface-2 p-4">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Quick Actions</div>
                <div class="flex flex-col gap-2">
                    <Button variant="danger" size="sm" class="justify-start text-[9px] h-8">
                        <ShieldAlert size={14} class="mr-2" /> ISOLATE HOST ⌃⇧I
                    </Button>
                    <Button variant="secondary" size="sm" class="justify-start text-[9px] h-8">
                        <MoreHorizontal size={14} class="mr-2" /> CAPTURE EVIDENCE ⌃⇧E
                    </Button>
                    <Button variant="cta" size="sm" class="justify-start text-[9px] h-8">
                        <Zap size={14} class="mr-2" /> RUN PLAYBOOK PB-CRED-01
                    </Button>
                    <Button variant="secondary" size="sm" class="justify-start text-[9px] h-8">
                        <MoreHorizontal size={14} class="mr-2" /> PIVOT IN SIEM
                    </Button>
                </div>
            </div>
        </div>
    {/if}

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>QUEUE:</span>
            <span class="text-success font-bold">LIVE</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>AUTO-TRIAGE:</span>
            <span>ON</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="ml-auto">ALERT ENGINE v2.14.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
