<!-- Configuration Risk — bound to ComplianceService + RuntimeTrustService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { AlertTriangle, ShieldCheck, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  let trustIdx = $state<number | null>(null);
  let pillars = $state<Record<string, number>>({});
  let packs = $state<any[]>([]);

  async function refresh() {
    if (IS_BROWSER) return;
    try {
      const trust = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/runtimetrustservice');
      trustIdx = (await trust.CalculateTrustIndex()) as number;
      pillars = ((await trust.GetPillarScores()) ?? {}) as Record<string, number>;
      const cmp = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/complianceservice');
      packs = ((await cmp.ListCompliancePacks()) ?? []) as any[];
    } catch {}
  }
  onMount(refresh);

  let pillarsLow = $derived(Object.entries(pillars).filter(([_, v]) => v < 0.5));
  let cmpFail = $derived(packs.filter((p) => p.controls && p.passing < p.controls).length);
</script>

<PageLayout title="Configuration Risk" subtitle="Drift, posture, and trust attestation">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>Refresh</Button>
    <PopOutButton route="/config-risk" title="Config Risk" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Trust Index" value={trustIdx !== null ? trustIdx.toFixed(2) : '—'} variant={(trustIdx ?? 0) >= 0.8 ? 'success' : 'warning'} />
      <KPI label="Failing Pillars" value={pillarsLow.length.toString()} variant={pillarsLow.length > 0 ? 'critical' : 'success'} />
      <KPI label="Failing Compliance" value={cmpFail.toString()} variant={cmpFail > 0 ? 'warning' : 'muted'} />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="flex items-center gap-2 mb-3"><AlertTriangle size={14} class="text-warning" /><span class="text-[10px] uppercase tracking-widest font-bold">Pillars Below Threshold</span></div>
        {#if pillarsLow.length === 0}
          <div class="text-center text-sm text-text-muted py-6">All pillars OK.</div>
        {:else}
          {#each pillarsLow as [name, v]}
            <div class="flex justify-between py-1 border-b border-border-primary text-[11px]">
              <span class="font-mono">{name}</span><span class="font-mono text-error">{(v * 100).toFixed(0)}%</span>
            </div>
          {/each}
        {/if}
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="flex items-center gap-2 mb-3"><ShieldCheck size={14} class="text-accent" /><span class="text-[10px] uppercase tracking-widest font-bold">Quick Pivots</span></div>
        <div class="grid grid-cols-1 gap-2">
          <Button variant="secondary" size="sm" onclick={() => push('/trust')}>Runtime Trust →</Button>
          <Button variant="secondary" size="sm" onclick={() => push('/compliance')}>Compliance Center →</Button>
          <Button variant="secondary" size="sm" onclick={() => push('/temporal-integrity')}>Temporal Integrity →</Button>
          <Button variant="secondary" size="sm" onclick={() => push('/governance')}>Governance →</Button>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
