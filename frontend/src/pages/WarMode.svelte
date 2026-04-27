<!-- War Mode — high-intensity SOC dashboard mixing real alert + agent + diag stores. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, PopOutButton } from '@components/ui';
  import { Siren, ShieldAlert, Eye, Zap } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
  import { crisisStore } from '@lib/stores/crisis.svelte';
  import { push } from '@lib/router.svelte';

  onMount(() => {
    if (typeof alertStore.init === 'function') alertStore.init();
    if (typeof agentStore.init === 'function') agentStore.init();
    diagnosticsStore.init();
  });

  let crit = $derived((alertStore.alerts ?? []).filter((a) => a.severity === 'critical'));
  let high = $derived((alertStore.alerts ?? []).filter((a) => a.severity === 'high'));
  let isolated = $derived((agentStore.agents ?? []).filter((a: any) => a.quarantined || a.status === 'quarantined'));

  function activateCrisis() {
    if (typeof (crisisStore as any).activate === 'function') {
      (crisisStore as any).activate('Manual war-mode activation');
    } else {
      (crisisStore as any).active = true;
    }
  }
</script>

<PageLayout title="War Mode" subtitle="High-intensity tactical response">
  {#snippet toolbar()}
    <Button variant="cta" size="sm" icon={Siren} onclick={activateCrisis}>ENGAGE CRISIS</Button>
    <PopOutButton route="/war-mode" title="War Mode" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <KPI label="Critical Alerts" value={crit.length.toString()} variant={crit.length > 0 ? 'critical' : 'muted'} />
      <KPI label="High Alerts" value={high.length.toString()} variant={high.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Isolated Hosts" value={isolated.length.toString()} variant={isolated.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Health" value={diagnosticsStore.healthGrade} variant={diagnosticsStore.healthGrade.startsWith('A') ? 'success' : 'warning'} />
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-error/30 rounded-md p-4 flex flex-col gap-3">
        <div class="flex items-center gap-2"><ShieldAlert size={14} class="text-error" /><span class="text-[10px] uppercase tracking-widest font-bold">Active Critical</span></div>
        <div class="flex-1 overflow-auto space-y-1">
          {#each crit.slice(0, 30) as a (a.id)}
            <div class="bg-surface-2 border border-error/20 rounded p-2 text-[11px]">
              <div class="font-bold">{a.title}</div>
              <div class="text-[10px] text-text-muted font-mono">{a.host}</div>
            </div>
          {:else}
            <div class="text-center text-sm text-text-muted py-6">No critical alerts.</div>
          {/each}
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
        <div class="flex items-center gap-2"><Eye size={14} class="text-accent" /><span class="text-[10px] uppercase tracking-widest font-bold">Quick Pivots</span></div>
        <Button variant="secondary" size="sm" onclick={() => push('/cases')}>Incident Cases</Button>
        <Button variant="secondary" size="sm" onclick={() => push('/operator')}>Operator Mode</Button>
        <Button variant="secondary" size="sm" onclick={() => push('/ransomware')}>Ransomware Shield</Button>
        <Button variant="secondary" size="sm" onclick={() => push('/response')}>SOAR / Response</Button>
        <Button variant="secondary" size="sm" onclick={() => push('/forensics')}>Forensics</Button>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
        <div class="flex items-center gap-2"><Zap size={14} class="text-accent" /><span class="text-[10px] uppercase tracking-widest font-bold">Crisis State</span></div>
        <div class="text-[11px]">{crisisStore.active ? 'Crisis declared' : 'Calm'}</div>
        <Badge variant={crisisStore.active ? 'critical' : 'muted'}>{crisisStore.active ? 'CRISIS' : 'NORMAL'}</Badge>
      </div>
    </div>
  </div>
</PageLayout>
