<!--
  Compliance Center — list compliance packs, evaluate, view reports.
  Bound to ComplianceService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { ClipboardCheck, Play, FileText, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Pack = { id?: string; name?: string; framework?: string; controls?: number; passing?: number };
  type Report = { id?: string; report_type?: string; generated_at?: string; status?: string; size_bytes?: number };

  let packs = $state<Pack[]>([]);
  let reports = $state<Report[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) { packs = []; reports = []; return; }
      const svc = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/complianceservice'
      );
      const [p, r] = await Promise.all([svc.ListCompliancePacks(), svc.ListReports()]);
      packs = (p ?? []) as Pack[];
      reports = (r ?? []) as Report[];
    } catch (e: any) {
      appStore.notify(`Compliance load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function evaluate(packID: string) {
    appStore.notify(`Evaluating ${packID}…`, 'info');
    try {
      const { EvaluatePack } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/complianceservice'
      );
      const result = await EvaluatePack(packID);
      appStore.notify(`Evaluated ${packID}`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Evaluate failed: ${e?.message ?? e}`, 'error');
    }
  }

  async function generateReport() {
    const reportType = prompt('Report type (soc2 | hipaa | iso27001):', 'soc2');
    if (!reportType) return;
    const start = Math.floor((Date.now() - 30 * 86400_000) / 1000);
    const end = Math.floor(Date.now() / 1000);
    try {
      const { GenerateReport } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/complianceservice'
      );
      await GenerateReport(reportType, start, end);
      appStore.notify(`Report ${reportType} generated`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Report failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let stats = $derived({
    packs: packs.length,
    reports: reports.length,
    overallPct: packs.length === 0 ? 0
      : Math.round(packs.reduce((s, p) => s + (p.controls ? (p.passing ?? 0) / p.controls : 0), 0) / packs.length * 100),
  });
</script>

<PageLayout title="Compliance Center" subtitle="SOC2 / HIPAA / NIST evaluation and audit reports">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" icon={FileText} onclick={generateReport}>Generate Report</Button>
    <PopOutButton route="/compliance" title="Compliance Center" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Active Packs"     value={stats.packs.toString()}      variant="accent" />
      <KPI label="Compliance Score" value={`${stats.overallPct}%`}      variant={stats.overallPct >= 90 ? 'success' : stats.overallPct >= 70 ? 'warning' : 'critical'} />
      <KPI label="Stored Reports"   value={stats.reports.toString()}    variant="muted" />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 flex-1 min-h-0">
      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <ClipboardCheck size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Frameworks</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each packs as p (p.id)}
            {@const ratio = p.controls && p.controls > 0 ? Math.round(((p.passing ?? 0) / p.controls) * 100) : 0}
            <div class="border-b border-border-primary px-3 py-2 flex items-center gap-3">
              <div class="flex-1">
                <div class="text-[11px] font-bold">{p.name ?? p.id ?? '—'}</div>
                <div class="text-[10px] text-text-muted">{p.framework ?? '—'} · {p.passing ?? 0}/{p.controls ?? 0}</div>
              </div>
              <span class="font-mono text-[10px] {ratio >= 90 ? 'text-success' : ratio >= 70 ? 'text-warning' : 'text-error'}">{ratio}%</span>
              <Button variant="ghost" size="xs" onclick={() => evaluate(p.id ?? '')}><Play size={10} /></Button>
            </div>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No compliance packs registered.'}</div>
          {/each}
        </div>
      </section>

      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <FileText size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Recent Reports</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each reports as r (r.id)}
            <div class="border-b border-border-primary px-3 py-2">
              <div class="text-[11px] font-bold">{r.report_type ?? r.id}</div>
              <div class="text-[10px] text-text-muted font-mono">{r.generated_at?.slice(0, 19) ?? '—'}</div>
              {#if r.status}<Badge variant={r.status === 'sealed' ? 'success' : 'info'} size="xs">{r.status}</Badge>{/if}
            </div>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">No reports generated yet.</div>
          {/each}
        </div>
      </section>
    </div>
  </div>
</PageLayout>
