<!-- Runtime Trust — bound to RuntimeTrustService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { ShieldCheck, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let trustIndex = $state<number | null>(null);
  let pillars = $state<Record<string, number>>({});
  let drift = $state<any>(null);
  let status = $state<any>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/runtimetrustservice');
      const [idx, p, d, s] = await Promise.all([
        svc.CalculateTrustIndex(), svc.GetPillarScores(), svc.GetTrustDriftMetrics(), svc.GetAggregatedStatus(),
      ]);
      trustIndex = (idx as number) ?? null;
      pillars = (p ?? {}) as Record<string, number>;
      drift = d; status = s;
    } catch (e: any) {
      appStore.notify(`Trust load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function verify() {
    try {
      const { VerifyIntegrity } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/runtimetrustservice');
      const r = await VerifyIntegrity();
      appStore.notify(`Integrity: ${(r as any)?.valid ? 'OK' : 'FAIL'}`, (r as any)?.valid ? 'success' : 'error');
    } catch (e: any) {
      appStore.notify(`Verify failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let pillarList = $derived(Object.entries(pillars));
</script>

<PageLayout title="Runtime Trust" subtitle="Process verification and execution trust">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={verify}>Verify</Button>
    <PopOutButton route="/trust" title="Runtime Trust" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Trust Index" value={trustIndex !== null ? trustIndex.toFixed(2) : '—'} variant={(trustIndex ?? 0) >= 0.8 ? 'success' : (trustIndex ?? 0) >= 0.5 ? 'warning' : 'critical'} />
      <KPI label="Status" value={status?.state ?? status?.status ?? '—'} variant={status?.healthy ? 'success' : 'muted'} />
      <KPI label="Drift" value={(drift?.delta ?? '—').toString()} variant="muted" />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex-1">
      <div class="flex items-center gap-2 mb-4">
        <ShieldCheck size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Pillar Scores</span>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {#each pillarList as [name, score]}
          <div class="bg-surface-2 border border-border-primary rounded-md p-3">
            <div class="text-[10px] uppercase tracking-wider text-text-muted">{name}</div>
            <div class="font-mono text-lg {score >= 0.8 ? 'text-success' : score >= 0.5 ? 'text-warning' : 'text-error'}">{(score * 100).toFixed(0)}</div>
            <div class="h-1 bg-surface-3 rounded mt-2 overflow-hidden">
              <div class="h-full bg-accent" style="width: {Math.min(100, score * 100)}%"></div>
            </div>
          </div>
        {:else}
          <div class="md:col-span-3 text-center text-sm text-text-muted py-8">{loading ? 'Loading…' : 'No pillar data.'}</div>
        {/each}
      </div>
    </div>
  </div>
</PageLayout>
