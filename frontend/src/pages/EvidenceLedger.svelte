<!--
  OBLIVRA — Evidence Ledger (Svelte 5)
  Immutable hash-linked record of all SOC actions and forensic evidence.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const mockLedger: Record<string, any>[] = [
    { id: 'tx_01', type: 'Artifact Hash', action: 'SHA-256 Registered', identity: 'maverick', timestamp: '2026-04-10 01:22:05', hash: 'e3b0c442...8fc1' },
    { id: 'tx_02', type: 'Decision', action: 'Kill Process - ID 4122', identity: 'system_autopilot', timestamp: '2026-04-10 01:21:55', hash: 'f8a1c92...a321' },
    { id: 'tx_03', type: 'Note', action: 'Entry Modified', identity: 'iceman', timestamp: '2026-04-10 01:15:00', hash: '928ca11...12cc' },
    { id: 'tx_04', type: 'Access', action: 'Vault Unlocked', identity: 'maverick', timestamp: '2026-04-10 01:05:12', hash: '412bb22...99ee' },
  ];
</script>

<PageLayout title="Temporal Integrity Ledger" subtitle="Non-repudiatory record of all operations and forensic acquisitions">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-2 px-3 py-1 bg-success/10 border border-success/20 rounded-full">
        <div class="w-1.5 h-1.5 rounded-full bg-success animate-pulse"></div>
        <span class="text-[9px] font-bold text-success uppercase tracking-widest">State Verified</span>
      </div>
      <Button variant="secondary" size="sm">Export Signed Bundle</Button>
      <Button variant="primary" size="sm" onclick={() => appStore.notify('Re-indexing blockchain segments...', 'info')}>Verify Integrity</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Ledger Metrics -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Total Blocks" value="14,211" trend="stable" trendValue="Synced" />
      <KPI label="Latest Commit" value="41s ago" trend="stable" trendValue="Chain Height: 882,109" variant="accent" />
      <KPI label="Validator Nodes" value="8" trend="stable" trendValue="Healthy" variant="success" />
      <KPI label="Ledger Stability" value="100.00%" trend="stable" trendValue="Zero Delta" variant="success" />
    </div>

    <!-- Ledger Table -->
    <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-premium">
      <DataTable data={mockLedger} columns={[
        { key: 'timestamp', label: 'Commit Time', width: '150px' },
        { key: 'type', label: 'Primitive', width: '120px' },
        { key: 'action', label: 'Mutation' },
        { key: 'identity', label: 'Signatory', width: '120px' },
        { key: 'hash', label: 'Merkle Proof', width: '150px' }
      ]} compact>
        {#snippet render({ col: column, row })}
          {#if column.key === 'hash'}
            <div class="flex items-center gap-2">
              <code class="text-[9px] font-mono text-accent opacity-70 truncate max-w-[100px]">{row.hash}</code>
              <Badge variant="success" size="xs">OK</Badge>
            </div>
          {:else if column.key === 'type'}
            <span class="text-[10px] uppercase font-bold text-text-muted tracking-tight">{row.type}</span>
          {:else if column.key === 'timestamp'}
             <span class="text-[10px] font-mono text-text-muted tabular-nums">{row.timestamp.split(' ')[1]}</span>
          {:else if column.key === 'identity'}
             <div class="flex items-center gap-1.5">
                <div class="w-4 h-4 rounded-full bg-surface-3 flex items-center justify-center text-[8px] font-bold text-text-muted">
                  {row.identity[0].toUpperCase()}
                </div>
                <span class="text-[11px] font-bold text-text-secondary">{row.identity}</span>
             </div>
          {:else if column.key === 'action'}
             <span class="text-[11px] font-bold text-text-heading">{row.action}</span>
          {:else}
            <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
          {/if}
        {/snippet}
      </DataTable>
    </div>

    <!-- Verification Banner -->
    <div class="p-6 bg-surface-2 border border-border-primary rounded-md flex flex-col items-center justify-center text-center gap-2 border-dashed">
        <div class="flex items-center gap-2 text-success">
          <Shield size={16} />
          <h4 class="text-[11px] font-bold uppercase tracking-widest">Cryptographic Consensus Achieved</h4>
        </div>
        <p class="text-[10px] text-text-muted max-w-sm">
          All records are hash-linked and synchronized across the distributed ledger cluster. 
          The current state block <span class="text-accent font-mono">0x4F12...EE03</span> has been validated by 8 independent agents.
        </p>
    </div>
  </div>
</PageLayout>
