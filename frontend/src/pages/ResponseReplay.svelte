<!--
  OBLIVRA — Response Replay (Svelte 5)
  Forensic replay and audit of historical containment and response missions.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield, Zap, Play, RotateCcw, Activity, Search, Clock } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const replayHistory = [
    { id: 'RR-101', mission: 'Auto-Isolation - Node Beta', trigger: 'Malicious Egress', time: '2026-04-09 14:22', result: 'Success' },
    { id: 'RR-102', mission: 'Vault Force-Lock', trigger: 'Lateral Movement', time: '2026-04-09 12:10', result: 'Partial' },
    { id: 'RR-103', mission: 'Binary Quarantine', trigger: 'Entropy Alert', time: '2026-04-08 22:45', result: 'Success' },
  ];
</script>

<PageLayout title="Response Replay" subtitle="Forensic reconstruction of historical containment missions and automated decisions">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Download Audit Log</Button>
    <Button variant="primary" size="sm" icon="🔄">New Reconstruction</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Replayed Missions" value={replayHistory.length} trend="Verified" />
      <KPI title="Success Rate" value="94.2%" trend="Optimal" variant="success" />
      <KPI title="Avg Replay Depth" value="L7" trend="Packet Level" variant="accent" />
      <KPI title="Audit Stability" value="100%" trend="Signed" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Replay Inventory -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Mission Replay Archive
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={replayHistory} columns={[
              { key: 'mission', label: 'Containment Event' },
              { key: 'trigger', label: 'Detection Trigger', width: '150px' },
              { key: 'time', label: 'Execution Time', width: '150px' },
              { key: 'result', label: 'Outcome', width: '100px' },
              { key: 'action', label: '', width: '80px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'result'}
                   <Badge variant={row.result === 'Success' ? 'success' : 'warning'}>{row.result}</Badge>
                {:else if column.key === 'mission'}
                   <div class="flex items-center gap-2">
                      <Shield size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.mission}</span>
                   </div>
                {:else if column.key === 'time'}
                   <span class="text-[10px] font-mono text-text-muted">{row.time}</span>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs"><Play size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Replay Controls & Context -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 border-dashed relative group">
            <RotateCcw size={48} class="text-accent opacity-40 group-hover:rotate-[-90deg] transition-transform duration-500" />
            <div class="relative z-10 font-bold uppercase tracking-widest text-[10px] text-text-muted">Mission Time-Machine</div>
            <p class="text-[10px] text-text-muted max-w-[180px]">Reconstruct any SOAR action with millisecond precision from the hash-linked ledger.</p>
            <Button variant="secondary" size="sm" class="w-full">Initialize Replay Session</Button>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-3">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Forensic Integrity
            </div>
            <div class="space-y-4">
               {#each Array(3) as _, i}
                  <div class="flex gap-3 items-start opacity-70">
                     <div class="w-1 h-1 rounded-full bg-accent mt-2"></div>
                     <div class="flex flex-col">
                        <span class="text-[10px] font-bold text-text-heading">Block {882100 + i} validated</span>
                        <span class="text-[8px] text-text-muted font-mono uppercase">Merkle Hash: 0x4f12...EE03</span>
                     </div>
                  </div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
