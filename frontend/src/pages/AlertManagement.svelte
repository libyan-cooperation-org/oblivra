<!--
  OBLIVRA — Alert Management (Svelte 5)
  Detection rule orchestration and signal filtering.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button, DataTable, Spinner, Input, Toggle } from '@components/ui';
  import { Shield, Bell, AlertTriangle, Search, Plus } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  let rules = $state<any[]>([]);
  let loading = $state(false);
  let searchQuery = $state('');

  const filteredRules = $derived(rules.filter(r => 
    r.Name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    r.ID?.toLowerCase().includes(searchQuery.toLowerCase())
  ));

  async function refreshRules() {
    if (IS_BROWSER) {
        rules = [
            { id: 'R-001', Name: 'Brute Force SSH', Type: 'correlation', Severity: 'high', Active: true, Hits: 142 },
            { id: 'R-002', Name: 'Log4j RCE Pattern', Type: 'signature', Severity: 'critical', Active: true, Hits: 12 },
            { id: 'R-003', Name: 'Reverse Shell Detect', Type: 'behavior', Severity: 'critical', Active: true, Hits: 3 },
            { id: 'R-004', Name: 'Abnormal Traffic Peak', Type: 'anomaly', Severity: 'medium', Active: false, Hits: 0 },
            { id: 'R-005', Name: 'Local Privilege Esc.', Type: 'behavior', Severity: 'high', Active: true, Hits: 5 },
        ];
        return;
    }
    loading = true;
    try {
        const { GetDetectionRules } = await import('@wailsjs/go/services/AlertingService.js');
        const data = await GetDetectionRules();
        rules = data || [];
    } catch (err) {
        appStore.notify('Failed to load rules', 'error', (err as Error).message);
    } finally {
        loading = false;
    }
  }

  function toggleRule(id: string) {
    rules = rules.map(r => r.ID === id || r.id === id ? { ...r, Active: !r.Active, active: !r.active } : r);
    const rule = rules.find(r => r.ID === id || r.id === id);
    appStore.notify(`Rule ${rule?.Name || rule?.name} ${rule?.Active || rule?.active ? 'engaged' : 'paused'}`, rule?.Active || rule?.active ? 'success' : 'warning');
  }

  onMount(() => {
    refreshRules();
  });
</script>

<PageLayout title="Rule Management" subtitle="Detection pipeline orchestration and logic control">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Filter rules..." bind:value={searchQuery} class="w-64" />
      <Button variant="primary" size="sm" icon="plus" onclick={() => appStore.notify('Rule Editor coming soon', 'info')}>Create Rule</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="System Vigilance" value="99.2%" trend="stable" trendValue="engaged" variant="success" />
      <KPI label="Active Threats" value="14" trend="up" trendValue="+8%" variant="warning" />
      <KPI label="Response Latency" value="2.4ms" trend="stable" trendValue="nominal" />
    </div>

    <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-premium relative min-h-[400px]">
      {#if loading}
          <div class="absolute inset-0 bg-surface-1/50 backdrop-blur-xs z-10 flex items-center justify-center">
            <Spinner />
          </div>
      {/if}

      <DataTable data={filteredRules} columns={[
        { key: 'Name', label: 'Detection Logic' },
        { key: 'Type', label: 'Mechanism', width: '120px' },
        { key: 'Severity', label: 'Impact', width: '100px' },
        { key: 'Hits', label: 'Signals', width: '100px' },
        { key: 'Active', label: 'Status', width: '120px' },
        { key: 'actions', label: '', width: '60px' }
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'Active'}
             <div class="flex items-center gap-2">
                <Toggle 
                  checked={row.Active ?? row.active} 
                  onchange={() => toggleRule(row.ID || row.id)}
                />
             </div>
          {:else if col.key === 'Severity'}
            <Badge variant={(row.Severity || row.severity) === 'critical' ? 'error' : (row.Severity || row.severity) === 'high' ? 'warning' : 'info'}>
              {row.Severity || row.severity}
            </Badge>
          {:else if col.key === 'Type'}
            <span class="text-[9px] font-bold text-text-muted uppercase tracking-widest">{row.Type || row.type}</span>
          {:else if col.key === 'Name'}
            <div class="flex flex-col">
              <span class="text-[11px] font-bold text-text-secondary">{row.Name || row.name}</span>
              <span class="text-[9px] text-text-muted font-mono">{row.ID || row.id}</span>
            </div>
          {:else if col.key === 'Hits'}
            <span class="font-mono text-[11px] {(row.Hits || row.hits) > 50 ? 'text-error font-bold' : 'text-text-muted'}">{row.Hits || row.hits || 0}</span>
          {:else if col.key === 'actions'}
            <button class="text-text-muted hover:text-accent transition-colors">
              <Shield size={14} />
            </button>
          {:else}
            {row[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</PageLayout>
