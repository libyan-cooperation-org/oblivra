<!-- Data Destruction — bound to DisasterService kill-switch + air-gap. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, Badge, PopOutButton } from '@components/ui';
  import { Skull, Power, ShieldOff, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let mode = $state<string>('—');
  let killSwitchActive = $state(false);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/disasterservice');
      const m = await svc.GetMode();
      mode = (m as any)?.mode ?? (typeof m === 'string' ? m : '—');
      killSwitchActive = (m as any)?.kill_switch ?? false;
    } finally { loading = false; }
  }

  async function activateKill() {
    const reason = prompt('Reason for kill-switch activation?'); if (!reason) return;
    if (!confirm('FINAL CONFIRM: Activate platform kill-switch?')) return;
    try {
      const { ActivateKillSwitch } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/disasterservice');
      await ActivateKillSwitch(reason);
      appStore.notify('Kill switch ACTIVE', 'error');
      void refresh();
    } catch (e: any) { appStore.notify(`Failed: ${e?.message ?? e}`, 'error'); }
  }
  async function deactivateKill() {
    if (!confirm('Release kill-switch?')) return;
    try {
      const { DeactivateKillSwitch } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/disasterservice');
      await DeactivateKillSwitch();
      appStore.notify('Kill switch released', 'success');
      void refresh();
    } catch (e: any) { appStore.notify(`Failed: ${e?.message ?? e}`, 'error'); }
  }
  async function airGap() {
    if (!confirm('Activate air-gap mode? All outbound network calls will be severed.')) return;
    try {
      const { ActivateAirGapMode } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/disasterservice');
      await ActivateAirGapMode();
      appStore.notify('Air-gap mode active', 'warning');
      void refresh();
    } catch (e: any) { appStore.notify(`Failed: ${e?.message ?? e}`, 'error'); }
  }
  async function nuke() {
    if (!confirm('NUCLEAR DESTRUCTION wipes ALL platform data. Type YES below to confirm.')) return;
    const t = prompt('Type EXACTLY "NUKE" to confirm:'); if (t !== 'NUKE') { appStore.notify('Cancelled', 'info'); return; }
    try {
      const { TriggerNuclearDestruction } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/disasterservice');
      await TriggerNuclearDestruction();
      appStore.notify('Nuclear destruction triggered', 'error');
    } catch (e: any) { appStore.notify(`Failed: ${e?.message ?? e}`, 'error'); }
  }
  onMount(refresh);
</script>

<PageLayout title="Data Destruction & Disaster" subtitle="Kill-switch, air-gap, and final-action controls">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/data-destruction" title="Data Destruction" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Mode" value={mode} variant={mode === 'normal' ? 'success' : 'warning'} />
      <KPI label="Kill Switch" value={killSwitchActive ? 'ACTIVE' : 'idle'} variant={killSwitchActive ? 'critical' : 'muted'} />
      <KPI label="Air-gap" value={mode === 'air-gap' || mode === 'airgap' ? 'YES' : 'no'} variant={mode === 'air-gap' || mode === 'airgap' ? 'warning' : 'muted'} />
    </div>

    <div class="bg-surface-1 border border-border-primary rounded-md p-6">
      <h3 class="text-xs uppercase tracking-widest font-bold mb-4">Emergency Controls</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div class="border border-warning/30 bg-warning/5 rounded p-4">
          <div class="text-sm font-bold mb-2">Air-gap Mode</div>
          <div class="text-[11px] text-text-muted mb-3">Cuts all outbound network traffic except agent traffic. Reversible.</div>
          <Button variant="warning" size="sm" icon={ShieldOff} onclick={airGap}>Activate Air-gap</Button>
        </div>
        <div class="border border-error/30 bg-error/5 rounded p-4">
          <div class="text-sm font-bold mb-2">Kill Switch</div>
          <div class="text-[11px] text-text-muted mb-3">Halts ingestion + correlation. Operator-only release.</div>
          {#if killSwitchActive}
            <Button variant="primary" size="sm" onclick={deactivateKill}>Release Kill Switch</Button>
          {:else}
            <Button variant="cta" size="sm" icon={Power} onclick={activateKill}>Activate Kill Switch</Button>
          {/if}
        </div>
        <div class="border border-error/40 bg-error/10 rounded p-4 md:col-span-2">
          <div class="text-sm font-bold text-error mb-2 flex items-center gap-2"><Skull size={14} />Nuclear Destruction</div>
          <div class="text-[11px] text-text-muted mb-3">
            Cryptographically wipes ALL tenants, ALL stored events, ALL evidence.
            <Badge variant="critical" size="xs">UNRECOVERABLE</Badge>
          </div>
          <Button variant="cta" size="sm" onclick={nuke}>Trigger Nuclear Destruction</Button>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
