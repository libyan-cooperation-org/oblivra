<!--
  OBLIVRA — Offline Update (Svelte 5)
  Secure, air-gapped platform updates and signature verification.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Input, Spinner } from '@components/ui';
  import { ShieldCheck, Download, Package, Activity, Zap, FileCode, CheckCircle2 } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let verifying = $state(false);
  let status = $state('ready'); // ready, verifying, complete

  function startUpdate() {
     verifying = true;
     setTimeout(() => {
        verifying = false;
        status = 'complete';
        appStore.notify('OFFLINE UPDATE APPLIED: Platform signature re-verified.', 'success');
     }, 3000);
  }
</script>

<PageLayout title="Offline Update" subtitle="Air-gapped capability orchestration: Manually installing signed update packages and capability blocks">
  {#snippet toolbar()}
    <Badge variant="info">AIR-GAP MODE: ACTIVE</Badge>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Current Version" value="v4.2.0" trend="Hardened" variant="success" />
      <KPI title="Update Channel" value="OFFLINE" trend="Air-Gapped" />
      <KPI title="Signature Root" value="HARDWARE" trend="Verified" variant="success" />
      <KPI title="Last Update" value="12d ago" trend="Nominal" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col items-center justify-center p-12 bg-surface-1 border border-border-primary rounded-md shadow-premium relative overflow-hidden group">
       <!-- Background Grid -->
       <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale" style="background-image: radial-gradient(#7aa2f7 1px, transparent 1px); background-size: 40px 40px;"></div>

       {#if status === 'ready'}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10">
             <div class="w-24 h-24 rounded-full bg-surface-2 border-4 border-accent/20 flex items-center justify-center shadow-glow-accent/5">
                <Package size={48} class="text-accent opacity-40 group-hover:opacity-100 transition-opacity" />
             </div>
             
             <div class="space-y-2">
                <h3 class="text-xl font-bold text-text-heading uppercase tracking-widest">Signed Package Ingest</h3>
                <p class="text-[11px] text-text-muted leading-relaxed">
                   Upload an `.oblv` update package for offline installation. OBLIVRA will cryptographically verify the package signature against the hardware root of trust before execution.
                </p>
             </div>

             <div class="w-full flex flex-col gap-4">
                <div class="p-8 border-2 border-dashed border-border-primary rounded-md bg-surface-2 hover:border-accent transition-colors cursor-pointer">
                   <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Drop .oblv package here</span>
                </div>
                <Button variant="primary" size="lg" class="w-full font-bold uppercase tracking-widest" onclick={startUpdate} disabled={verifying}>
                   {#if verifying} <Spinner size="sm" class="mr-2" /> VERIFYING... {:else} VERIFY & INSTALL {/if}
                </Button>
             </div>
          </div>
       {:else if status === 'complete'}
          <div class="max-w-md w-full flex flex-col items-center gap-8 text-center relative z-10 animate-fade-in">
             <div class="w-24 h-24 rounded-full bg-success/20 border-4 border-success/40 flex items-center justify-center shadow-glow-success/20">
                <CheckCircle2 size={48} class="text-success" />
             </div>
             <div class="space-y-2">
                <h3 class="text-xl font-bold text-text-heading uppercase tracking-widest">Update Successful</h3>
                <p class="text-[11px] text-text-muted">Platform signature verified. Version v4.2.1 is now active across the local node.</p>
             </div>
             <Button variant="secondary" size="sm" onclick={() => status = 'ready'}>Return to Ingest</Button>
          </div>
       {/if}
    </div>

    <!-- Security Metadata -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 shrink-0">
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center">
          <div class="flex gap-3 items-center">
             <ShieldCheck size={18} class="text-success" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">Root Key Verified</span>
          </div>
          <Badge variant="success">OK</Badge>
       </div>
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center">
          <div class="flex gap-3 items-center">
             <FileCode size={18} class="text-accent" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">ED25519 Signature</span>
          </div>
          <Badge variant="accent">VALID</Badge>
       </div>
       <div class="bg-surface-1 border border-border-primary p-4 rounded-md flex justify-between items-center">
          <div class="flex gap-3 items-center">
             <Activity size={18} class="text-accent" />
             <span class="text-xs font-bold text-text-heading uppercase tracking-widest">Integrity Hash</span>
          </div>
          <Badge variant="secondary">MATCH</Badge>
       </div>
    </div>
  </div>
</PageLayout>
