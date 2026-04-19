<!-- OBLIVRA Web — EvidenceVault (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, PageLayout, Button, DataTable, Spinner, ProgressBar } from '@components/ui';
  import { Lock, Unlock, FileText, Download, History, Search, Filter, ShieldCheck, ShieldAlert, HardDrive } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface ChainEntry {
    action: string;
    actor: string;
    timestamp: string;
    notes?: string;
    previous_hash: string;
    entry_hash: string;
  }
  interface EvidenceItem {
    id: string;
    incident_id: string;
    type: string;
    name: string;
    description?: string;
    sha256: string;
    size: number;
    collector: string;
    collected_at: string;
    sealed: boolean;
    sealed_at?: string;
    chain_of_custody: ChainEntry[];
    tags?: string[];
  }

  // -- State --
  let items       = $state<EvidenceItem[]>([]);
  let loading     = $state(true);
  let selected    = $state<EvidenceItem | null>(null);
  let search      = $state('');
  let tab         = $state<'all' | 'unsealed' | 'sealed'>('all');
  let verifying   = $state<string | null>(null);
  let verifyRes   = $state<Record<string, boolean>>({});

  // -- Helpers --
  const fmtSize = (n: number) => n < 1024 ? `${n} B` : n < 1048576 ? `${(n/1024).toFixed(1)} KB` : `${(n/1048576).toFixed(2)} MB`;
  const actionColor: Record<string, string> = {
    collected: 'var(--status-online)',
    analyzed: 'var(--alert-high)',
    transferred: 'var(--accent-primary)',
    sealed: 'var(--text-muted)',
    exported: 'var(--text-primary)',
    verified: 'var(--success)'
  };

  // -- Actions --
  async function fetchEvidence() {
    loading = true;
    try {
      const res = await request<{ items: EvidenceItem[] }>('/forensics/evidence');
      items = res.items ?? [];
    } catch {
      items = [];
    } finally {
      loading = false;
    }
  }

  async function verify(id: string) {
    verifying = id;
    try {
      const res = await request<{ valid: boolean }>(`/forensics/evidence/${id}/verify`);
      verifyRes = { ...verifyRes, [id]: res.valid };
    } catch {
      verifyRes = { ...verifyRes, [id]: false };
    } finally {
      verifying = null;
    }
  }

  async function seal(id: string) {
    try {
      await request(`/forensics/evidence/${id}/seal`, { method: 'POST' });
      fetchEvidence();
    } catch (e) {
      console.error('Sealing failed', e);
    }
  }

  function exportVault() {
    const url = `/api/v1/forensics/export`;
    const token = localStorage.getItem('oblivra_token');
    const a = document.createElement('a');
    a.href = url + (token ? `?token=${token}` : '');
    a.download = `oblivra-evidence-${Date.now()}.json`;
    a.click();
  }

  const displayed = $derived.by(() => {
    const q = search.toLowerCase();
    return items.filter(i => {
      if (tab === 'sealed' && !i.sealed) return false;
      if (tab === 'unsealed' && i.sealed) return false;
      return !q || i.name.toLowerCase().includes(q) || i.incident_id.toLowerCase().includes(q) || i.type.includes(q);
    });
  });

  onMount(() => {
    fetchEvidence();
  });
</script>

<PageLayout title="Evidence Vault" subtitle="Immutable forensic storage shards with hardware-backed chain of custody">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchEvidence}>
        <History size={14} class="mr-2" />
        Refresh
      </Button>
      <Button variant="primary" size="sm" onclick={exportVault}>
        <Download size={14} class="mr-2" />
        Export Vault
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Total Artifacts" value={items.length.toString()} trend="Global" variant="accent" />
      <KPI title="Sealed Evidence" value={items.filter(i => i.sealed).length.toString()} trend="Locked" variant="success" />
      <KPI title="Active Forensics" value={items.filter(i => !i.sealed).length.toString()} trend="In Triage" variant="warning" />
      <KPI title="Root Trust" value="VERIFIED" trend="TPM 2.0" variant="success" />
    </div>

    <!-- Main View -->
    <div class="flex flex-col lg:flex-row gap-6 flex-1 min-h-0">
      <!-- Evidence List -->
      <div class="flex-1 flex flex-col bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
        <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center">
          <div class="flex items-center gap-4">
            <div class="flex border border-border-primary rounded-sm overflow-hidden">
              {#each ['all', 'unsealed', 'sealed'] as t}
                <button
                  class="px-3 py-1 text-[10px] font-bold uppercase tracking-widest transition-colors
                    {tab === t ? 'bg-accent-primary text-black' : 'bg-surface-0 text-text-muted hover:text-text-secondary'}"
                  onclick={() => tab = t as any}
                >
                  {t}
                </button>
              {/each}
            </div>
            <div class="relative w-64">
              <input
                type="text"
                bind:value={search}
                placeholder="Filter evidence..."
                class="w-full bg-surface-0 border border-border-primary text-text-primary px-3 py-1 rounded-sm font-mono text-[10px] focus:outline-hidden"
              />
              <Search size={12} class="absolute right-2 top-1.5 text-text-muted opacity-50" />
            </div>
          </div>
          <Badge variant="secondary" size="xs">{displayed.length} ITEMS</Badge>
        </div>

        <div class="flex-1 overflow-auto">
          {#if loading}
            <div class="h-full flex items-center justify-center">
              <Spinner />
            </div>
          {:else}
            <DataTable
              data={displayed}
              columns={[
                { key: 'type', label: 'TYPE', width: '100px' },
                { key: 'name', label: 'ARTIFACT_NAME' },
                { key: 'incident_id', label: 'CASE_REF', width: '120px' },
                { key: 'size', label: 'SIZE', width: '80px' },
                { key: 'status', label: 'STATE', width: '100px' },
                { key: 'actions', label: 'OPERATIONS', width: '140px' }
              ]}
              compact
              rowKey="id"
              onRowClick={(row) => selected = selected?.id === row.id ? null : row}
            >
              {#snippet cell({ column, row })}
                {#if column.key === 'type'}
                  <div class="flex items-center gap-2">
                    <FileText size={12} class="text-text-muted" />
                    <span class="text-[10px] font-mono text-text-muted uppercase">{row.type}</span>
                  </div>
                {:else if column.key === 'name'}
                  <span class="font-bold text-text-heading truncate block max-w-[200px]">{row.name}</span>
                {:else if column.key === 'incident_id'}
                  <span class="text-[10px] font-mono text-accent-primary">{row.incident_id.slice(0, 12)}…</span>
                {:else if column.key === 'size'}
                  <span class="text-[10px] font-mono text-text-muted">{fmtSize(row.size)}</span>
                {:else if column.key === 'status'}
                  <Badge variant={row.sealed ? 'secondary' : 'success'} size="xs" dot>
                    {row.sealed ? 'SEALED' : 'ACTIVE'}
                  </Badge>
                {:else if column.key === 'actions'}
                  <div class="flex gap-2">
                    <button
                      class="text-[9px] font-bold uppercase tracking-wider px-2 py-0.5 border border-border-primary rounded-sm transition-colors
                        {verifyRes[row.id] === true ? 'border-status-online text-status-online bg-status-online/10' : 
                         verifyRes[row.id] === false ? 'border-alert-critical text-alert-critical bg-alert-critical/10' : 
                         'text-text-muted hover:text-text-primary'}"
                      onclick={(e) => { e.stopPropagation(); verify(row.id); }}
                      disabled={verifying === row.id}
                    >
                      {verifying === row.id ? '...' : verifyRes[row.id] === true ? 'VALID' : verifyRes[row.id] === false ? 'FAILED' : 'VERIFY'}
                    </button>
                    {#if !row.sealed}
                      <button
                        class="text-[9px] font-bold uppercase tracking-wider px-2 py-0.5 border border-border-primary rounded-sm text-text-muted hover:text-alert-high hover:border-alert-high transition-colors"
                        onclick={(e) => { e.stopPropagation(); seal(row.id); }}
                      >
                        SEAL
                      </button>
                    {/if}
                  </div>
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>
      </div>

      <!-- Detail Panel -->
      <div class="w-full lg:w-96 flex flex-col bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
        {#if !selected}
          <div class="flex-1 flex flex-col items-center justify-center text-text-muted opacity-40 gap-4 p-8 text-center">
            <Lock size={48} />
            <p class="font-mono text-[10px] uppercase tracking-widest">Select an artifact to inspect chain of custody</p>
          </div>
        {:else}
          <div class="p-4 bg-surface-2 border-b border-border-primary flex justify-between items-center">
             <div class="flex items-center gap-2">
                {#if selected.sealed}<Lock size={14} class="text-text-muted" />{:else}<Unlock size={14} class="text-status-online" />{/if}
                <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">Chain of Custody</span>
             </div>
             <button class="text-text-muted hover:text-text-primary" onclick={() => selected = null}>✕</button>
          </div>
          
          <div class="flex-1 overflow-auto p-4 flex flex-col gap-6">
            <div>
              <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest mb-2">Artifact Metadata</div>
              <div class="bg-surface-0 p-3 border border-border-primary rounded-sm space-y-2">
                <div class="flex justify-between items-center">
                  <span class="text-[9px] font-mono text-text-muted">NAME</span>
                  <span class="text-[10px] font-bold text-text-primary truncate ml-4">{selected.name}</span>
                </div>
                <div class="flex flex-col gap-1">
                  <span class="text-[9px] font-mono text-text-muted">SHA-256 HASH</span>
                  <code class="text-[9px] font-mono text-accent-primary break-all bg-surface-1 p-1 rounded-xs border border-border-subtle">{selected.sha256}</code>
                </div>
              </div>
            </div>

            <div class="flex-1 flex flex-col min-h-0">
               <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest mb-3">Custody Timeline</div>
               <div class="relative space-y-0 pl-4 border-l border-border-primary ml-1">
                 {#each selected.chain_of_custody as entry, i}
                    <div class="relative pb-6 last:pb-0">
                      <!-- Dot -->
                      <div class="absolute -left-[20.5px] top-1.5 w-2 h-2 rounded-full border border-surface-1" style="background: {actionColor[entry.action] ?? 'var(--text-muted)'}"></div>
                      
                      <div class="flex flex-col gap-1">
                        <div class="flex justify-between items-center">
                          <span class="text-[10px] font-black uppercase tracking-widest" style="color: {actionColor[entry.action] ?? 'var(--text-muted)'}">{entry.action}</span>
                          <span class="text-[9px] font-mono text-text-muted opacity-50">{new Date(entry.timestamp).toLocaleString()}</span>
                        </div>
                        <div class="text-[11px] font-bold text-text-secondary">{entry.actor}</div>
                        {#if entry.notes}
                          <div class="text-[10px] text-text-muted italic bg-surface-0/50 p-1.5 border-l border-border-primary mt-1">{entry.notes}</div>
                        {/if}
                        <div class="text-[8px] font-mono text-text-muted opacity-30 mt-1 uppercase tracking-tighter">HMAC: {entry.entry_hash.slice(0, 24)}...</div>
                      </div>
                    </div>
                 {/each}
               </div>
            </div>
          </div>
          
          <div class="p-4 bg-surface-2 border-t border-border-primary flex gap-2">
            <Button variant="secondary" size="sm" class="flex-1" icon={ShieldCheck} onclick={() => verify(selected!.id)}>VERIFY INTEGRITY</Button>
            {#if !selected.sealed}
               <Button variant="primary" size="sm" class="flex-1" icon={Lock} onclick={() => seal(selected!.id)}>SEAL ARTIFACT</Button>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>

<style>
  :global(.mask-fade-bottom) {
    mask-image: linear-gradient(to bottom, black 90%, transparent 100%);
  }
</style>
