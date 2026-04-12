<!--
  OBLIVRA — Forensics Engine (Svelte 5)
  Deep artifact analysis, root cause forensics, and disk imaging orchestration.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Tabs, Spinner } from '@components/ui';
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';

  const forensicsTabs = [
    { id: 'artifacts', label: 'Artifact Collection', icon: '🔍' },
    { id: 'timeline', label: 'Forensic Timeline', icon: '🕒' },
    { id: 'imaging', label: 'Disk Imaging', icon: '💿' },
    { id: 'volatile', label: 'Memory Analysis', icon: '⚡' },
  ];

  let activeTab = $state('artifacts');
  let artifacts = $state<any[]>([]);
  let loading = $state(false);

  async function loadForensics() {
    if (IS_BROWSER) {
        artifacts = [
            { id: 'a1', Type: 'Prefetch', Name: 'MALICIOUS.EXE-A1B2C3D4.pf', Source: 'C:\\Windows\\Prefetch', RiskScore: 85, CollectedAt: '10m ago' },
            { id: 'a2', Type: 'Shimcache', Name: 'N/A', Source: 'Registry', RiskScore: 45, CollectedAt: '15m ago' },
            { id: 'a3', Type: 'MFT', Name: '$MFT', Source: 'C:\\$MFT', RiskScore: 5, CollectedAt: '1h ago' },
            { id: 'a4', Type: 'Amcache', Name: 'Amcache.hve', Source: 'C:\\Windows\\AppCompat\\Programs', RiskScore: 98, CollectedAt: '5m ago' },
        ];
        return;
    }
    loading = true;
    try {
        const { ListEvidence } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
        const items = await ListEvidence(""); // All evidence
        artifacts = items || [];
    } catch (err) {
        console.error('Forensics load failed', err);
    } finally {
        loading = false;
    }
  }

  onMount(() => {
    loadForensics();
  });

  const columns: any[] = [
    { key: 'collected', label: 'Collected', width: '100px' },
    { key: 'type', label: 'Type', width: '100px' },
    { key: 'name', label: 'Artifact Name', sortable: true },
    { key: 'risk', label: 'Risk Score', width: '120px' },
  ];
</script>

<PageLayout title="Digital Forensics" subtitle="Post-incident root cause analysis and evidence preservation">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Download Evidence Container</Button>
    <Button variant="primary" size="sm" icon="⚔">Acquire All Artifacts</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Artifacts Found" value={mockArtifacts.length} variant="default" />
      <KPI label="Suspicious Files" value="2" variant="critical" />
      <KPI label="Imaging State" value="Idle" variant="accent" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden relative">
      {#if loading}
        <div class="absolute inset-0 bg-surface-1/40 backdrop-blur-xs z-10 flex items-center justify-center">
            <Spinner />
        </div>
      {/if}
      <Tabs tabs={forensicsTabs} bind:active={activeTab} />

      <div class="p-4 flex-1">
        {#if activeTab === 'artifacts'}
          <DataTable data={artifacts} {columns} compact striped>
            {#snippet render({ value, col, row }: any)}
              {#if col.key === 'risk'}
                <Badge variant={row.RiskScore >= 90 ? 'critical' : row.RiskScore >= 70 ? 'critical' : row.RiskScore >= 40 ? 'warning' : 'success'}>
                  {row.RiskScore >= 90 ? 'CRITICAL' : row.RiskScore >= 70 ? 'HIGH' : row.RiskScore >= 40 ? 'MEDIUM' : 'LOW'}
                </Badge>
              {:else if col.key === 'type'}
                <span class="text-[10px] font-bold text-accent px-1.5 py-0.5 bg-accent/5 rounded border border-accent/10">{row.Type}</span>
              {:else if col.key === 'name'}
                <span class="font-mono text-[11px] text-text-heading">{row.Name}</span>
              {:else if col.key === 'collected'}
                <span class="text-[10px] text-text-muted">{row.CollectedAt}</span>
              {:else}
                {value}
              {/if}
            {/snippet}
          </DataTable>
        {:else}
          <div class="flex flex-col items-center justify-center h-full opacity-40 py-12">
            <span class="text-4xl mb-4">🧪</span>
            <div class="text-sm font-bold text-text-heading">{activeTab} module is loading...</div>
          </div>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>
