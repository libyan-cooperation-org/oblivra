<!--
  OBLIVRA — License & Entitlements (Svelte 5)
  Managing platform licensing, capabilities and sovereign-grade entitlements.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, Input } from '@components/ui';
  import { ShieldCheck, Lock } from 'lucide-svelte';
  

  let licenseKey = $state('OBL-7721-MESH-SIGN');
</script>

<PageLayout title="License & Entitlements" subtitle="Managing sovereign hardware-bound licenses and platform capability entitlements">
  <div class="flex flex-col h-full gap-8 max-w-4xl">
    
    <!-- License Hero -->
    <div class="bg-surface-1 border border-border-primary rounded-lg p-8 flex flex-col items-center text-center gap-6 relative overflow-hidden group shadow-premium">
       <!-- Verification Pulse -->
       <div class="absolute inset-0 bg-success/5 animate-pulse"></div>
       
       <div class="relative w-24 h-24 rounded-full border-4 border-success/20 flex items-center justify-center bg-surface-2 shadow-glow-success/10">
          <ShieldCheck size={48} class="text-success" />
       </div>

       <div class="relative">
          <h2 class="text-2xl font-bold text-text-heading">Sovereign Enterprise</h2>
          <p class="text-xs text-text-muted mt-1 uppercase tracking-widest font-bold">Hardware-Bound Identity Verified</p>
       </div>

       <div class="relative grid grid-cols-2 gap-4 w-full max-w-sm">
          <div class="p-4 bg-surface-2 border border-border-secondary rounded-md">
             <span class="text-[9px] text-text-muted uppercase font-bold tracking-widest">Valid Until</span>
             <span class="block text-lg font-bold font-mono text-text-heading">2029-12-31</span>
          </div>
          <div class="p-4 bg-surface-2 border border-border-secondary rounded-md">
             <span class="text-[9px] text-text-muted uppercase font-bold tracking-widest">Max Nodes</span>
             <span class="block text-lg font-bold font-mono text-text-heading">Unlimited</span>
          </div>
       </div>

       <div class="relative flex gap-4">
          <Button variant="cta" size="sm">Download Signature</Button>
          <Button variant="secondary" size="sm">Transfer License</Button>
       </div>
    </div>

    <!-- Activation Logic -->
    <div class="bg-surface-1 border border-border-primary rounded-md p-6 space-y-4">
       <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
          <Lock size={12} />
          License Transformation
       </div>
       <div class="flex gap-4">
          <Input placeholder="Enter new license key..." bind:value={licenseKey} class="flex-1 font-mono uppercase" />
          <Button variant="primary">Apply Key</Button>
       </div>
       <p class="text-[9px] text-text-muted italic">Applying a new key will immediately re-calibrate platform capability blocks and propagate signatures to all mesh nodes.</p>
    </div>

    <!-- Capabilities Registry -->
    <div class="flex flex-col gap-4">
       <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Entitled Capability Blocks</h3>
       <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          {#each ['Autonomous SOAR', 'Distributed SIEM', 'Cognitive AI Shell', 'Merkle-Tree Audit', 'BGP Global Graph', 'WASM Sandbox'] as cap}
             <div class="bg-surface-1 border border-border-primary rounded px-4 py-3 flex justify-between items-center group hover:border-success/40 transition-colors cursor-default">
                <div class="flex items-center gap-3">
                   <div class="w-2 h-2 rounded-full bg-success"></div>
                   <span class="text-xs font-bold text-text-heading">{cap}</span>
                </div>
                <Badge variant="success" size="xs">ENABLED</Badge>
             </div>
          {/each}
       </div>
    </div>
  </div>
</PageLayout>
