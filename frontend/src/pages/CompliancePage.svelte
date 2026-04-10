<!--
  OBLIVRA — Compliance Console (Svelte 5)
  Real-time tracking of SOC2, HIPAA, and GDPR posture.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Tabs, Spinner } from '@components/ui';
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';

  const complianceTabs = [
    { id: 'soc2', label: 'SOC2 Type II', icon: '📋' },
    { id: 'hipaa', label: 'HIPAA', icon: '🏥' },
    { id: 'gdpr', label: 'GDPR', icon: '🇪🇺' },
    { id: 'custom', label: 'Internal Audit', icon: '🛡️' },
  ];

  let activeTab = $state('soc2');
  let compliancePacks = $state<any[]>([]);
  let loading = $state(false);

  async function loadCompliance() {
    if (IS_BROWSER) {
        compliancePacks = [
            { PackID: 'soc2', ControlID: 'cc1.1', Name: 'Logical Access Control', Description: 'Restricts access to system components...', Status: 'compliant', EvidenceSource: 'Automatic' },
            { PackID: 'soc2', ControlID: 'cc3.2', Name: 'Risk Assessment', Description: 'Periodic assessment of internal and external risks...', Status: 'needs_review', EvidenceSource: 'Manual' },
            { PackID: 'soc2', ControlID: 'cc6.1', Name: 'Boundary Protection', Description: 'Monitoring and control of communications at boundaries...', Status: 'compliant', EvidenceSource: 'Automatic' },
            { PackID: 'soc2', ControlID: 'cc7.3', Name: 'Incident Response', Description: 'Response to identified security incidents...', Status: 'failed', EvidenceSource: 'Missing' },
        ];
        return;
    }
    loading = true;
    try {
        const { ListCompliancePacks } = await import('@wailsjs/go/services/ComplianceService.js');
        const packs = await ListCompliancePacks();
        // Flatten or filter based on active tab if the service returns nested data
        // For now, we'll assume a list of controls for the mock-up
        compliancePacks = packs || [];
    } catch (err) {
        console.error('Compliance load failed', err);
    } finally {
        loading = false;
    }
  }

  const filteredControls = $derived(
    compliancePacks.filter(p => !p.PackID || p.PackID.toLowerCase().includes(activeTab))
  );

  onMount(() => {
    loadCompliance();
  });

  const columns = [
    { key: 'id', label: 'ID', width: '80px' },
    { key: 'control', label: 'Control Name', sortable: true },
    { key: 'status', label: 'Status', width: '120px' },
    { key: 'evidence', label: 'Evidence', width: '100px' },
  ];
</script>

<PageLayout title="Compliance Hub" subtitle="Evidence collection and regulatory posture monitoring">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Download PDF Report</Button>
    <Button variant="cta" size="sm">Kickoff Internal Audit</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Overall Posture" value="92%" variant="success" />
      <KPI label="Open Findings" value="4" variant="warning" />
      <KPI label="Evidence Sync" value="Live" variant="accent" />
      <KPI label="Next Renewal" value="142 Days" variant="default" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden relative">
      {#if loading}
        <div class="absolute inset-0 bg-surface-1/40 backdrop-blur-xs z-10 flex items-center justify-center">
            <Spinner />
        </div>
      {/if}
      <Tabs tabs={complianceTabs} bind:active={activeTab} />

      <div class="flex-1 overflow-hidden p-0">
        <DataTable data={filteredControls} {columns} striped>
          {#snippet render({ value, col, row })}
            {#if col.key === 'status'}
              <Badge variant={value === 'compliant' ? 'success' : value === 'needs_review' ? 'warning' : 'critical'} dot>
                {value.replace('_', ' ')}
              </Badge>
            {:else if col.key === 'control'}
              <div class="flex flex-col">
                <span class="font-bold text-text-heading">{row.Name}</span>
                <span class="text-[9px] text-text-muted">{row.Description}</span>
              </div>
            {:else if col.key === 'evidence'}
              <span class="text-[10px] font-mono {row.EvidenceSource === 'Automatic' ? 'text-accent' : 'text-text-muted'}">{row.EvidenceSource || 'Manual'}</span>
            {:else if col.key === 'id'}
              <span class="text-[10px] font-mono text-text-muted">{row.ControlID}</span>
            {:else}
              {value}
            {/if}
          {/snippet}
        </DataTable>
      </div>
    </div>
  </div>
</PageLayout>
