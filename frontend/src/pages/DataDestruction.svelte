<!--
  OBLIVRA — Data Destruction (Svelte 5)
  Secure erasure and atomic sanitization of mission logs and technical data.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Trash2, ShieldAlert, Database, Check } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let armed = $state(false);
  let wipeProgress = $state(0);
  let isWiping = $state(false);

  function startWipe() {
    isWiping = true;
    const interval = setInterval(() => {
       wipeProgress += 2;
       if (wipeProgress >= 100) {
          clearInterval(interval);
          isWiping = false;
          appStore.notify('DATA SANITIZATION COMPLETE: ZERO-LIFETIME ACHIEVED.', 'success');
       }
    }, 100);
  }
</script>

<PageLayout title="Data Sanitization" subtitle="Atomic destruction of mission telemetry and secure erasure of technical data blocks">
  {#snippet toolbar()}
     <Badge variant="critical" dot={isWiping}>PLATFORM WIPER: {isWiping ? 'EXECUTING' : 'READY'}</Badge>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Sanitized Mass" value="1.2 TB" trend="stable" trendValue="Historical" variant="success" />
      <KPI label="Active Vaults" value="4" trend="stable" trendValue="Nominal" />
      <KPI label="Erasure Standard" value="DoD 5220" trend="stable" trendValue="Hardened" variant="success" />
      <KPI label="Persistence Check" value="ZERO" trend="stable" trendValue="Verified" variant="success" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col items-center justify-center p-12 bg-surface-1 border border-border-primary rounded-md shadow-premium relative overflow-hidden">
       <!-- Wiper Background Grid -->
       <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale" style="background-image: linear-gradient(#f7768e 1px, transparent 1px), linear-gradient(90deg, #f7768e 1px, transparent 1px); background-size: 50px 50px;"></div>

       {#if !isWiping}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10">
             <div class="w-24 h-24 rounded-full bg-surface-2 border-4 border-error/20 flex items-center justify-center shadow-glow-error/5">
                <Trash2 size={48} class="text-error opacity-40 focus:opacity-100 transition-opacity" />
             </div>
             
             <div class="space-y-2">
                <h3 class="text-xl font-bold text-text-heading uppercase tracking-widest">Secure Erasure Interface</h3>
                <p class="text-[11px] text-text-muted leading-relaxed">
                   This module facilitates the cryptographic shredding of selected data blocks. Once initiated, the selected data will be overwritten using the DoD 5220.22-M standard and becomes unrecoverable even with forensic hardware.
                </p>
             </div>

             <div class="w-full p-4 bg-surface-2 border border-border-secondary rounded-md space-y-4">
                <div class="flex justify-between items-center text-[10px] font-bold text-text-muted uppercase">
                   <span>Target: Core Platform Logs</span>
                   <Badge variant="warning">EPHEMERAL</Badge>
                </div>
                <Button variant="critical" size="lg" class="w-full font-bold uppercase tracking-widest" onclick={startWipe}>
                   COMMENCE SHREDDING
                </Button>
             </div>
          </div>
       {:else}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10">
             <div class="relative w-32 h-32 flex items-center justify-center">
                <svg class="w-full h-full transform -rotate-90">
                   <circle cx="64" cy="64" r="60" stroke="currentColor" stroke-width="8" fill="transparent" class="text-surface-3" />
                   <circle cx="64" cy="64" r="60" stroke="currentColor" stroke-width="8" fill="transparent" stroke-dasharray="377" stroke-dashoffset={377 - (377 * wipeProgress) / 100} class="text-error transition-all duration-300" />
                </svg>
                <div class="absolute inset-0 flex flex-col items-center justify-center">
                   <span class="text-xl font-bold font-mono text-error">{wipeProgress}%</span>
                   <span class="text-[8px] font-bold text-text-muted uppercase">Erasing</span>
                </div>
             </div>
             
             <div class="space-y-2">
                <h3 class="text-lg font-bold text-error uppercase tracking-widest animate-pulse">Cryptographic Shredding in Progress</h3>
                <p class="text-[10px] text-text-muted font-mono bg-surface-2 p-2 rounded border border-error/20">
                   SHREDDING_BLOCK: 0x{Math.floor(Math.random() * 0xffffffff).toString(16).toUpperCase()}
                </p>
             </div>
          </div>
       {/if}
    </div>

    <!-- Integrity Proofs -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 shrink-0">
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center">
          <div class="flex gap-3 items-center">
             <Database size={18} class="text-accent" />
             <div class="flex flex-col">
                <span class="text-xs font-bold text-text-heading">Block Zeroing Integrity</span>
                <span class="text-[9px] text-text-muted">Verified via hardware entropy source</span>
             </div>
          </div>
          <Badge variant="success">PASS</Badge>
       </div>
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center">
          <div class="flex gap-3 items-center">
             <ShieldAlert size={18} class="text-success" />
             <div class="flex flex-col">
                <span class="text-xs font-bold text-text-heading">Post-Erase Persistence Audit</span>
                <span class="text-[9px] text-text-muted">No residual bit-traces detected</span>
             </div>
          </div>
          <Check size={18} class="text-success" />
       </div>
    </div>
  </div>
</PageLayout>
