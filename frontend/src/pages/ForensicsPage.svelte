<!--
  OBLIVRA — Forensics Engine (Svelte 5)
  Deep artifact analysis, root cause forensics, and disk imaging orchestration.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Tabs, Spinner } from '@components/ui';
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { apiFetch } from '@lib/apiClient';

  const forensicsTabs = [
    { id: 'artifacts', label: 'Artifact Collection', icon: '🔍' },
    { id: 'timeline', label: 'Forensic Timeline', icon: '🕒' },
    { id: 'imaging', label: 'Disk Imaging', icon: '💿' },
    { id: 'volatile', label: 'Memory Analysis', icon: '⚡' },
  ];

  let activeTab = $state('artifacts');
  let artifacts = $state<any[]>([]);
  let loading = $state(false);
  let loadError = $state<string | null>(null);

  // Audit fix — browser mode used to populate `artifacts` with four
  // FAKE items (`MALICIOUS.EXE-A1B2C3D4.pf` RiskScore=85, etc.) and
  // the Suspicious-Files KPI was hardcoded "2". Forensic evidence is
  // a chain-of-custody artefact; presenting fictitious entries to a
  // responder could push them to act on phantom IOCs. We now hit the
  // real `/api/v1/forensics/evidence` endpoint and surface the actual
  // (possibly empty) locker contents — schemas accommodate either the
  // REST shape (snake_case) or Wails shape (PascalCase).
  async function loadForensics() {
    loading = true;
    loadError = null;
    try {
      if (IS_BROWSER) {
        const res = await apiFetch('/api/v1/forensics/evidence');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const body = await res.json();
        artifacts = (body.evidence ?? []) as any[];
        return;
      }
      const { ListEvidence } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      const items = await ListEvidence(''); // All evidence
      artifacts = (items ?? []) as any[];
    } catch (err: any) {
      console.error('Forensics load failed', err);
      loadError = String(err?.message ?? err);
    } finally {
      loading = false;
    }
  }

  // Suspicious-files count is derived, not invented. Threshold matches
  // the table's "HIGH"/"CRITICAL" split (≥70).
  const suspiciousCount = $derived.by(() => {
    return artifacts.filter((a: any) => {
      const r = Number(a.RiskScore ?? a.risk_score ?? 0);
      return r >= 70;
    }).length;
  });

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
      <KPI label="Artifacts Found" value={artifacts.length} variant="default" />
      <KPI label="Suspicious Files" value={suspiciousCount.toString()} variant={suspiciousCount > 0 ? 'critical' : 'muted'} sublabel={suspiciousCount > 0 ? 'risk ≥ 70' : 'No high-risk items'} />
      <KPI label="Imaging State" value="Idle" sublabel="No imaging job in flight" variant="muted" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden relative">
      {#if loading}
        <div class="absolute inset-0 bg-surface-1/40 backdrop-blur-xs z-10 flex items-center justify-center">
            <Spinner />
        </div>
      {/if}
      <Tabs tabs={forensicsTabs} bind:active={activeTab} />

      <div class="p-4 flex-1">
        {#if loadError}
          <div class="mb-3 px-3 py-2 text-[11px] font-mono text-error bg-error/10 border border-error/30 rounded">
            Evidence load failed: {loadError}
          </div>
        {/if}
        {#if activeTab === 'artifacts'}
          {#if !loading && artifacts.length === 0}
            <div class="flex flex-col items-center justify-center h-full opacity-50 py-12">
              <span class="text-4xl mb-4">🗃️</span>
              <div class="text-sm font-bold text-text-heading">No evidence collected yet</div>
              <p class="text-[10px] text-text-muted mt-1 max-w-[260px] text-center">
                Trigger an artefact acquisition or wait for an automated capture to populate this view.
              </p>
            </div>
          {:else}
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
          {/if}
        {:else}
          <div class="flex flex-col items-center justify-center h-full opacity-40 py-12">
            <span class="text-4xl mb-4">🧪</span>
            <div class="text-sm font-bold text-text-heading">{activeTab} module not yet available</div>
            <p class="text-[10px] text-text-muted mt-1">This tab will activate once the corresponding service is wired.</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>
