<!--
  OBLIVRA — War Mode (Svelte 5)
  Emergency containment and platform-wide lockdown orchestration.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Input } from '@components/ui';
  import { ShieldAlert, Zap, Lock, Skull, Activity, Globe, WifiOff } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let armed = $state(false);
  let authCode = $state('');

  function armWarMode() {
    if (authCode === 'SIGMA') {
       armed = true;
       appStore.notify('WAR MODE ARMED: Platform lockdown initiated.', 'critical');
    } else {
       appStore.notify('INVALID AUTHENTICATION CODE', 'error');
    }
  }
</script>

<PageLayout title="War Mode" subtitle="Emergency platform-wide containment and high-gravity orchestration">
  {#snippet toolbar()}
    <Badge variant="critical" dot={armed}>OS-LEVEL LOCKDOWN: {armed ? 'ACTIVE' : 'READY'}</Badge>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Containment State" value={armed ? 'LOCKDOWN' : 'MONITOR'} trend="up" trendValue="Critical" variant={armed ? 'critical' : 'success'} />
      <KPI label="Active Blockers" value={armed ? 'ALL' : 'ZERO'} trend="stable" trendValue="Nominal" />
      <KPI label="Egress Shield" value="100%" trend="stable" trendValue="Verified" variant="success" />
      <KPI label="Destroy Readiness" value="ARMED" trend="up" trendValue="Hot" variant="warning" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col items-center justify-center p-12 bg-surface-1 border-2 {armed ? 'border-critical' : 'border-border-primary'} rounded-md shadow-premium relative overflow-hidden">
       <!-- Background Terror Grid -->
       <div class="absolute inset-0 opacity-[0.05] pointer-events-none grayscale" style="background-image: radial-gradient({armed ? '#f7768e' : '#7aa2f7'} 1px, transparent 1px); background-size: 30px 30px;"></div>

       {#if !armed}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10">
             <div class="w-24 h-24 rounded-full bg-surface-2 border-4 border-error/20 flex items-center justify-center shadow-glow-error/10">
                <Skull size={48} class="text-error opacity-40" />
             </div>
             
             <div class="space-y-2">
                <h3 class="text-xl font-bold text-text-heading uppercase tracking-widest">Arm War Mode</h3>
                <p class="text-[11px] text-text-muted leading-relaxed">
                   Initiating War Mode will immediately terminate all active sessions, lock all vaults, and rotate platform root keys. This action cannot be reversed without hardware intervention.
                </p>
             </div>

             <div class="w-full space-y-4">
                <div class="flex flex-col gap-1 text-left">
                   <label class="text-[9px] font-bold text-text-muted uppercase tracking-widest ml-1">Authentication Oracle</label>
                   <Input type="password" placeholder="Enter emergency authorization..." bind:value={authCode} class="font-mono text-center tracking-[4px]" />
                </div>
                <Button variant="critical" size="lg" class="w-full py-6 text-sm font-bold uppercase tracking-widest shadow-glow-error/20" onclick={armWarMode}>
                   ENGAGE LOCKDOWN
                </Button>
             </div>
          </div>
       {:else}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10">
             <div class="w-24 h-24 rounded-full bg-error flex items-center justify-center shadow-glow-error/40 animate-ping absolute opacity-20"></div>
             <div class="w-24 h-24 rounded-full bg-error flex items-center justify-center shadow-glow-error/40">
                <ShieldAlert size={48} class="text-white" />
             </div>
             
             <div class="space-y-4">
                <h3 class="text-2xl font-black text-error uppercase tracking-tighter">PLATFORM ISOLATED</h3>
                <div class="grid grid-cols-2 gap-4">
                   <Badge variant="critical" size="lg">EGRESS: KILLED</Badge>
                   <Badge variant="critical" size="lg">DEVICES: LOCKED</Badge>
                </div>
                <p class="text-xs text-text-heading font-mono bg-surface-2 p-3 border border-error/50 rounded">
                   CLEAN_SLATE_PROTOCOL: RUNNING
                </p>
             </div>

             <Button variant="secondary" size="sm" onclick={() => armed = false}>STAND DOWN (FORCE RE-AUTH)</Button>
          </div>
       {/if}
    </div>

    <!-- Secondary Emergency Controls -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 shrink-0">
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center group cursor-pointer hover:border-error transition-colors">
          <div class="flex gap-3 items-center">
             <WifiOff size={18} class="text-error" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">Sever External Uplinks</span>
          </div>
          <Badge variant="critical">ARMED</Badge>
       </div>
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center group cursor-pointer hover:border-error transition-colors">
          <div class="flex gap-3 items-center">
             <Skull size={18} class="text-error" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">Atomic Data Wipe</span>
          </div>
          <Badge variant="critical">UNARMED</Badge>
       </div>
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center group cursor-pointer hover:border-accent transition-colors">
          <div class="flex gap-3 items-center">
             <Globe size={18} class="text-accent" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">Propagate Lockdown</span>
          </div>
          <Badge variant="accent">READY</Badge>
       </div>
    </div>
  </div>
</PageLayout>
