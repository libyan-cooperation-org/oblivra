<!--
  OBLIVRA — Ransomware Response UI (Svelte 5)
  Emergency dashboard for high-intensity encryption events and fleet isolation.
-->
<script lang="ts">
  import { Badge, PageLayout, Button } from '@components/ui';

</script>

<PageLayout title="Ransomware Defusal" subtitle="Active countermeasures and data recovery orchestration">
  {#snippet toolbar()}
    <Button variant="danger" size="sm" icon="🚫">Global Fleet Isolation</Button>
    <Button variant="cta" size="sm" icon="🔥">War Mode</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Active Threats -->
    <div class="p-12 border-2 border-dashed border-border-primary rounded-lg flex flex-col items-center justify-center text-center bg-error/5">
      <div class="text-6xl mb-4 animate-pulse">📡</div>
      <h2 class="text-xl font-bold text-text-heading">Scanning Fleet for Entropy Abnormality</h2>
      <p class="text-sm text-text-muted mt-2 max-w-lg">
        OBLIVRA Ransomware Defense is actively monitoring file-write entropy across all 142 nodes. 
        No widespread encryption events detected in the last 24 hours.
      </p>
      <div class="mt-8 flex gap-4">
        <Button variant="secondary" size="sm">Run Entropy Sweep</Button>
        <Button variant="ghost" size="sm">View Honeypot Status</Button>
      </div>
    </div>

    <!-- Analytics -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
      <div class="bg-surface-1 border border-border-primary rounded-md p-6">
        <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted mb-4">Shadow Copy Status</h3>
        <div class="flex flex-col gap-4">
          <div class="flex justify-between items-center bg-surface-2 p-3 rounded">
            <span class="text-xs">Prod-Web-Cluster</span>
            <Badge variant="success">Protected (14 snapshots)</Badge>
          </div>
          <div class="flex justify-between items-center bg-surface-2 p-3 rounded">
            <span class="text-xs">DB-Master-ReadOnly</span>
            <Badge variant="success">Protected (4 snapshots)</Badge>
          </div>
          <div class="flex justify-between items-center bg-surface-2 p-3 rounded opacity-50">
            <span class="text-xs">Legacy-Fileserver</span>
            <Badge variant="warning">Unsupported OS</Badge>
          </div>
        </div>
      </div>

      <div class="bg-surface-1 border border-border-primary rounded-md p-6">
         <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted mb-4">Isolation Readiness</h3>
         <div class="flex flex-col gap-4">
           <div class="flex justify-between items-center">
             <span class="text-xs text-text-muted">Avg. Isolation Latency</span>
             <span class="text-sm font-bold text-text-primary">142ms</span>
           </div>
           <div class="flex justify-between items-center">
             <span class="text-xs text-text-muted">Nodes pre-staged</span>
             <span class="text-sm font-bold text-accent">142 / 142</span>
           </div>
           <Button variant="danger" class="mt-2">Dry Run Isolation Sequence</Button>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
