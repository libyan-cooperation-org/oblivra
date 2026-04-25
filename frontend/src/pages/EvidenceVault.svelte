<!--
  OBLIVRA — Evidence Vault (Svelte 5)
  Immutable forensic storage and chain-of-custody management.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, PopOutButton} from '@components/ui';
  import { Lock, FileText, Download, History, Search, Filter, HardDrive } from 'lucide-svelte';
  import { forensicsStore } from '@lib/stores/forensics.svelte.ts';
  import { onMount } from 'svelte';

  const evidenceItems = $derived(forensicsStore.items);

  onMount(() => {
    forensicsStore.loadIncidentEvidence('GLOBAL'); // Load all for the vault view
  });
</script>

<PageLayout title="Evidence Vault" subtitle="Immutable forensic storage shards with hardware-backed chain of custody">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={HardDrive}>MOUNT SHARD</Button>
      <Button variant="primary" size="sm" icon={Download}>EXPORT BUNDLE</Button>
    </div>
      <PopOutButton route="/evidence-vault" title="Evidence Vault" />
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
                    { key: 'name', label: 'ARTIFACT_NAME', width: '200px' },
                    { key: 'type', label: 'TYPE', width: '120px' },
                    { key: 'hash', label: 'INTEGRITY_HASH' },
                    { key: 'sealed', label: 'STATUS', width: '100px' },
                    { key: 'timestamp', label: 'INGESTED', width: '100px' }
                ]} 
                compact
                onRowClick={(row) => forensicsStore.loadChain(row.id)}
            >
                {#snippet render({ col, row })}
                    {#if col.key === 'id'}
                        <div class="flex items-center gap-2 py-0.5">
                            <FileText size={12} class="text-accent opacity-60" />
                            <span class="text-[10px] font-mono font-bold text-text-heading">{row.id}</span>
                        </div>
                    {:else if col.key === 'name'}
                        <span class="text-[9px] font-mono text-accent">{row.name}</span>
                    {:else if col.key === 'type'}
                        <span class="text-[9px] font-mono text-text-muted uppercase">{row.type}</span>
                    {:else if col.key === 'hash'}
                        <code class="text-[9px] font-mono text-text-muted opacity-60">{row.hash?.substring(0, 16)}...</code>
                    {:else if col.key === 'sealed'}
                        <Badge variant={row.sealed ? 'success' : 'info'} size="xs" dot>{row.sealed ? 'SEALED' : 'OPEN'}</Badge>
                    {:else if col.key === 'timestamp'}
                        <span class="text-[9px] font-mono text-text-muted">{new Date(row.timestamp).toLocaleTimeString()}</span>
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
            {#each forensicsStore.activeChain as entry}
                <div class="flex gap-2 text-text-muted">
                    <span class="opacity-40">[{new Date(entry.timestamp).toLocaleTimeString()}]</span>
                    <span class="text-accent font-bold uppercase">[{entry.action}]</span>
                    <span>{entry.actor}: {entry.notes}</span>
                </div>
            {:else}
                <div class="h-full flex items-center justify-center text-[9px] text-text-muted uppercase tracking-[0.2em] opacity-30">
                    Select an item to view chain of custody
                </div>
            {/each}
        </div>
    </div>
  </div>
</PageLayout>
