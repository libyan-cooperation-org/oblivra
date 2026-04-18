<!--
  OBLIVRA — Chain of Custody (Svelte 5)
  Interactive tracking of digital evidence artifacts and handling history.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';
  import { Shield, FileArchive, ArrowRight, User, Hash } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const evidence = [
    { id: 'E-401', name: 'mem_dump_prod_01.bin', collector: 'maverick', size: '4.2 GB', protocol: 'SFTP-SEC', integrity: 'verified' },
    { id: 'E-402', name: 'bash_history_operator.log', collector: 'iceman', size: '12 KB', protocol: 'LOCAL', integrity: 'verified' },
    { id: 'E-403', name: 'secrets.gpg.bak', collector: 'system', size: '840 Bytes', protocol: 'SHADOW-COPY', integrity: 'warning' },
  ];
</script>

<PageLayout title="Chain of Custody" subtitle="Formal tracking of forensic evidence acquisition and transfer">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Download Bundle</Button>
    <Button variant="cta" size="sm">Acquire Artifact</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <KPI title="Managed Artifacts" value={evidence.length} trend="Active" />
      <KPI title="Total Forensic Mass" value="4.20 GB" trend="High Density" variant="warning" />
      <KPI title="Verification Score" value="99.2%" trend="Optimal" variant="success" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
      <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">Evidence Registry</div>
      <div class="flex-1 overflow-auto">
        <DataTable data={evidence} columns={[
          { key: 'name', label: 'Artifact Name' },
          { key: 'size', label: 'Volume', width: '100px' },
          { key: 'collector', label: 'Acquired By', width: '120px' },
          { key: 'protocol', label: 'Transport', width: '120px' },
          { key: 'integrity', label: 'State', width: '100px' }
        ]} density="compact">
          {#snippet cell({ column, row })}
            {#if column.key === 'integrity'}
               <Badge variant={row.integrity === 'verified' ? 'success' : 'warning'}>{row.integrity}</Badge>
            {:else if column.key === 'name'}
               <div class="flex items-center gap-2">
                 <FileArchive size={14} class="text-accent" />
                 <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
               </div>
            {:else if column.key === 'collector'}
               <div class="flex items-center gap-1.5 font-bold text-[10px] text-text-secondary">
                  <User size={12} class="opacity-40" />
                  {row.collector}
               </div>
            {:else}
              <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
            {/if}
          {/snippet}
        </DataTable>
      </div>
    </div>

    <!-- Handover History -->
    <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
      <div class="flex justify-between items-center border-b border-border-primary pb-2">
        <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Digital Handover Log</div>
        <Badge variant="accent" size="xs">SECURE STREAM</Badge>
      </div>
      
      <div class="flex items-center gap-6 overflow-x-auto py-2">
         {#each Array(4) as _, i}
            <div class="flex items-center gap-3 shrink-0">
               <div class="flex flex-col items-center">
                  <div class="w-8 h-8 rounded-full bg-surface-3 flex items-center justify-center border border-border-primary shadow-sm">
                    <User size={14} class="text-text-muted" />
                  </div>
                  <span class="text-[9px] font-bold mt-1 text-text-muted">Node {i+1}</span>
               </div>
               {#if i < 3}
                 <div class="flex flex-col items-center gap-1">
                    <ArrowRight size={14} class="text-accent opacity-50" />
                    <span class="text-[8px] font-mono text-success">SIGNED</span>
                 </div>
               {/if}
            </div>
         {/each}
      </div>
    </div>
  </div>
</PageLayout>
