<!--
  OBLIVRA — Temporal Integrity (Svelte 5)
  Cryptographic verification of the system's temporal state and Merkle-tree consistency.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Database, Lock } from 'lucide-svelte';
  

  const integrityBlocks = [
    { epoch: '1,422,100', hash: '0x8f22...11ac', verified: true, signers: 4, drift: '0ms' },
    { epoch: '1,422,101', hash: '0x9a01...ee42', verified: true, signers: 4, drift: '0ms' },
    { epoch: '1,422,102', hash: '0x44cd...bc01', verified: false, signers: 2, drift: '+12ms' },
  ];
</script>

<PageLayout title="Temporal Integrity" subtitle="Cryptographic state verification: OBLIVRA Merkle-tree and hash-linked audit consistency">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Verify All Epochs</Button>
    <Button variant="primary" size="sm" icon="🔐">Resign State</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Verified Epochs" value="1.4M" trend="up" trendValue="Deep Scan" variant="success" />
      <KPI label="Current Trust" value="High" trend="stable" trendValue="4/4 Signers" variant="success" />
      <KPI label="Atomic Drift" value="1.4ms" trend="stable" trendValue="Nominal" />
      <KPI label="Integrity Alerts" value="1" trend="up" trendValue="Action Req." variant="critical" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Epoch Registry -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Merkle Epoch Registry (Temporal State)
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={integrityBlocks} columns={[
              { key: 'epoch', label: 'Epoch ID', width: '120px' },
              { key: 'hash', label: 'State Hash' },
              { key: 'signers', label: 'Signers', width: '80px' },
              { key: 'drift', label: 'Drift', width: '100px' },
              { key: 'status', label: 'Integrity', width: '100px' }
            ]} compact>
              {#snippet render({ value, col, row })}
                {#if col.key === 'status'}
                   <Badge variant={row.verified ? 'success' : 'critical'}>{row.verified ? 'VERIFIED' : 'FAILED'}</Badge>
                {:else if col.key === 'epoch'}
                   <span class="text-[11px] font-mono font-bold text-text-heading">{row.epoch}</span>
                {:else if col.key === 'hash'}
                   <code class="text-[10px] font-mono text-text-secondary opacity-60">{row.hash}</code>
                {:else if col.key === 'drift'}
                   <span class="text-[10px] font-mono {row.drift !== '0ms' ? 'text-error' : 'text-text-muted'}">{row.drift}</span>
                {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Trust Visuals -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group">
            <Lock size={48} class="text-success opacity-40" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-muted uppercase tracking-widest">Hardware Trust Root</h4>
               <p class="text-[10px] text-text-muted mt-2">The current state is linked to the local TPM and 4 distributed hardware signers.</p>
            </div>
            <div class="flex gap-2 relative z-10">
               {#each Array(4) as _, i}
                  <div class="w-3 h-3 rounded-full bg-success opacity-40 animate-pulse" style="animation-delay: {i * 200}ms"></div>
               {/each}
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Database size={12} />
               Storage Proof Continuity
            </div>
            <div class="space-y-4">
               <div>
                   <div class="flex justify-between text-[10px] mb-1">
                      <span class="text-text-secondary">Merkle Proof Consistency</span>
                      <span class="font-bold">100%</span>
                   </div>
                   <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                      <div class="bg-success h-full" style="width: 100%"></div>
                   </div>
               </div>
               <div>
                   <div class="flex justify-between text-[10px] mb-1">
                      <span class="text-text-secondary">Distributed Witness Sync</span>
                      <span class="font-bold text-warning">82%</span>
                   </div>
                   <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                      <div class="bg-warning h-full" style="width: 82%"></div>
                   </div>
               </div>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
