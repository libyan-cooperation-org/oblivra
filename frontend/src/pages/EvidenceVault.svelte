<!--
  OBLIVRA — Evidence Vault (Svelte 5)
  Immutable forensic storage and chain-of-custody management.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Lock, FileText, Download, History, Search, Filter, HardDrive } from 'lucide-svelte';

  const evidenceItems = [
    { id: 'EV-2026-001', case: 'INC-4421', type: 'Memory Dump', hash: 'sha256:a4c5..89e1', status: 'sealed', time: '10:42:15' },
    { id: 'EV-2026-002', case: 'INC-4421', type: 'Disk Image', hash: 'sha256:b12d..42c1', status: 'sealed', time: '10:45:10' },
    { id: 'EV-2026-003', case: 'INC-4418', type: 'Network PCAP', hash: 'sha256:f42a..11e9', status: 'accessed', time: '09:12:00' },
    { id: 'EV-2026-004', case: 'INC-4402', type: 'Process Tree', hash: 'sha256:e901..bc42', status: 'sealed', time: '1 day ago' }
  ];
</script>

<PageLayout title="Evidence Vault" subtitle="Immutable forensic storage shards with hardware-backed chain of custody">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={HardDrive}>MOUNT SHARD</Button>
      <Button variant="primary" size="sm" icon={Download}>EXPORT BUNDLE</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- VAULT STATS -->
    <div class="grid grid-cols-4 gap-4 shrink-0">
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-1">
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Vault Capacity</span>
            <div class="flex justify-between items-end">
                <span class="text-xl font-mono font-bold text-text-heading">1.4 TB</span>
                <span class="text-[9px] text-text-muted">12% used</span>
            </div>
            <div class="h-1 bg-surface-1 rounded-full overflow-hidden mt-1">
                <div class="h-full bg-accent" style="width: 12%"></div>
            </div>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-1">
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Immutable Shards</span>
            <div class="text-xl font-mono font-bold text-text-heading">142</div>
            <span class="text-[9px] text-success mt-1">▲ All shards verified</span>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-1">
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Active Cases</span>
            <div class="text-xl font-mono font-bold text-accent">12</div>
            <span class="text-[9px] text-text-muted mt-1">Managed forensic contexts</span>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-1">
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Root Trust</span>
            <div class="text-xl font-mono font-bold text-success">VERIFIED</div>
            <span class="text-[9px] text-success mt-1">TPM v2.0 Bound</span>
        </div>
    </div>

    <!-- MAIN VAULT LEDGER -->
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-sm flex flex-col min-h-0">
        <div class="flex items-center justify-between p-3 border-b border-border-primary bg-surface-2 shrink-0">
            <div class="flex items-center gap-2">
                <Lock size={14} class="text-accent" />
                <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">Sovereign Evidence Ledger</span>
            </div>
            <div class="flex gap-2">
                <Button variant="ghost" size="xs" icon={Search}>SEARCH</Button>
                <Button variant="ghost" size="xs" icon={Filter}>FILTER</Button>
            </div>
        </div>
        <div class="flex-1 overflow-auto mask-fade-bottom">
            <DataTable 
                data={evidenceItems} 
                columns={[
                    { key: 'id', label: 'EVIDENCE_ID', width: '120px' },
                    { key: 'case', label: 'CASE_REF', width: '100px' },
                    { key: 'type', label: 'ARTIFACT_TYPE', width: '120px' },
                    { key: 'hash', label: 'INTEGRITY_HASH' },
                    { key: 'status', label: 'STATUS', width: '100px' },
                    { key: 'time', label: 'INGESTED', width: '100px' }
                ]} 
                compact
            >
                {#snippet render({ col, row })}
                    {#if col.key === 'id'}
                        <div class="flex items-center gap-2 py-0.5">
                            <FileText size={12} class="text-accent opacity-60" />
                            <span class="text-[10px] font-mono font-bold text-text-heading">{row.id}</span>
                        </div>
                    {:else if col.key === 'case'}
                        <span class="text-[9px] font-mono text-accent">{row.case}</span>
                    {:else if col.key === 'type'}
                        <span class="text-[9px] font-mono text-text-muted uppercase">{row.type}</span>
                    {:else if col.key === 'hash'}
                        <code class="text-[9px] font-mono text-text-muted opacity-60">{row.hash}</code>
                    {:else if col.key === 'status'}
                        <Badge variant={row.status === 'sealed' ? 'success' : 'info'} size="xs" dot>{row.status}</Badge>
                    {:else if col.key === 'time'}
                        <span class="text-[9px] font-mono text-text-muted">{row.time}</span>
                    {/if}
                {/snippet}
            </DataTable>
        </div>
    </div>

    <!-- AUDIT TRAIL FOOTER -->
    <div class="h-32 bg-surface-2 border border-border-primary rounded-sm p-3 flex flex-col gap-2 shrink-0">
        <div class="flex items-center gap-2 border-b border-border-primary pb-1.5">
            <History size={12} class="text-text-muted" />
            <span class="text-[8px] font-mono font-bold text-text-muted uppercase tracking-widest">Vault Access Audit</span>
        </div>
        <div class="flex-1 overflow-auto space-y-1 font-mono text-[9px]">
            <div class="flex gap-2 text-text-muted">
                <span class="opacity-40">[10:45:10]</span>
                <span class="text-accent font-bold">[ACCESS]</span>
                <span>Operator K. MAVERICK accessed EV-2026-003 for Case INC-4418</span>
            </div>
            <div class="flex gap-2 text-text-muted">
                <span class="opacity-40">[10:42:15]</span>
                <span class="text-success font-bold">[SEAL]</span>
                <span>Evidence EV-2026-001 finalized and sealed with Root Key #14A</span>
            </div>
            <div class="flex gap-2 text-text-muted">
                <span class="opacity-40">[09:12:00]</span>
                <span class="text-info font-bold">[SYNC]</span>
                <span>Shard #04 synchronized with secondary vault node</span>
            </div>
        </div>
    </div>
  </div>
</PageLayout>
