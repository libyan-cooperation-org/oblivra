<!-- OBLIVRA Web — Regulator Portal (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, Input } from '@components/ui';
  import { Shield, FileText, Download, Lock, CheckCircle, AlertTriangle, XCircle, Filter, Calendar, History, FileCheck, Database, Fingerprint } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface AuditEntry {
    id: string;
    timestamp: string;
    actor: string;
    action: string;
    resource: string;
    outcome: 'success' | 'failure' | 'blocked';
    ip?: string;
    tenant_id?: string;
    entry_hash: string;
    prev_hash: string;
  }
  interface CompliancePackage {
    id: string;
    framework: string;
    generated_at: string;
    records: number;
    integrity_proof: string;
    download_url: string;
  }

  // -- Constants --
  const FRAMEWORKS = ['ALL', 'SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'GDPR'];
  const OUTCOME_STYLE = {
    success: { color: 'var(--status-online)',  bg: 'rgba(0,200,100,0.1)', icon: CheckCircle },
    failure: { color: 'var(--alert-critical)', bg: 'rgba(200,44,44,0.1)', icon: XCircle },
    blocked: { color: 'var(--alert-medium)',   bg: 'rgba(200,140,0,0.1)', icon: AlertTriangle },
  };

  // -- State --
  let tab          = $state<'audit' | 'packages' | 'export'>('audit');
  let framework    = $state('ALL');
  let dateFrom     = $state('');
  let dateTo       = $state('');
  let selectedFw   = $state('SOC2');
  let loading      = $state(true);
  let generating   = $state(false);

  let auditEntries = $state<AuditEntry[]>([]);
  let packages     = $state<CompliancePackage[]>([]);

  // -- Helpers --
  const totalStats = $derived({
    total: auditEntries.length,
    success: auditEntries.filter(e => e.outcome === 'success').length,
    failure: auditEntries.filter(e => e.outcome === 'failure').length,
    blocked: auditEntries.filter(e => e.outcome === 'blocked').length,
  });

  // -- Actions --
  async function fetchAudit() {
    loading = true;
    try {
      const params = new URLSearchParams({ limit: '200' });
      if (framework !== 'ALL') params.set('framework', framework);
      if (dateFrom) params.set('from', dateFrom);
      if (dateTo) params.set('to', dateTo);
      const res = await request<{ entries: AuditEntry[] }>(`/audit/log?${params}`);
      auditEntries = res.entries ?? [];
    } catch (e) {
      console.error('Audit fetch failed', e);
    } finally {
      loading = false;
    }
  }

  async function fetchPackages() {
    try {
      const res = await request<{ packages: CompliancePackage[] }>('/audit/packages');
      packages = res.packages ?? [];
    } catch (e) {
      console.error('Packages fetch failed', e);
    }
  }

  async function generatePackage() {
    generating = true;
    try {
      await request('/audit/packages/generate', {
        method: 'POST',
        body: JSON.stringify({ framework: selectedFw, from: dateFrom, to: dateTo }),
      });
      fetchPackages();
      tab = 'packages';
    } catch (e) {
      console.error('Generation failed', e);
    } finally {
      generating = false;
    }
  }

  onMount(() => {
    fetchAudit();
    fetchPackages();
  });
</script>

<PageLayout title="Regulator Portal" subtitle="Scoped audit exposure and cryptographically-verified compliance artifacts">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <div class="px-3 py-1 bg-surface-2 border border-accent-primary/30 rounded-xs flex items-center gap-2">
        <div class="w-1.5 h-1.5 rounded-full bg-accent-primary animate-pulse"></div>
        <span class="text-[9px] font-mono font-bold text-accent-primary uppercase tracking-widest">Read-Only Auditor Access</span>
      </div>
      <Button variant="secondary" size="sm" onclick={fetchAudit}>
        <History size={14} class="mr-2" />
        RE-SYNC
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Audit Stream Depth</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{totalStats.total.toLocaleString()}</div>
            <div class="text-[9px] text-text-muted mt-1 italic">Verified L7 entries</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Validated Outcomes</div>
            <div class="text-xl font-mono font-bold text-status-online">{totalStats.success.toLocaleString()}</div>
            <div class="text-[9px] text-status-online mt-1 uppercase tracking-tighter">✓ Integrity confirmed</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Audit Failures</div>
            <div class="text-xl font-mono font-bold text-alert-critical">{totalStats.failure.toLocaleString()}</div>
            <div class="text-[9px] text-alert-critical mt-1 uppercase tracking-tighter">⚠ System warnings</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Policy Blocked</div>
            <div class="text-xl font-mono font-bold text-alert-medium">{totalStats.blocked.toLocaleString()}</div>
            <div class="text-[9px] text-alert-medium mt-1 uppercase tracking-tighter">Proactive containment</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex flex-col min-h-0 bg-surface-0 overflow-hidden">
       <!-- TAB BAR -->
       <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
          <div class="flex border border-border-primary rounded-sm overflow-hidden">
            {#each ['audit', 'packages', 'export'] as t}
              <button
                class="px-4 py-1.5 text-[10px] font-black uppercase tracking-widest transition-colors
                  {tab === t ? 'bg-accent-primary text-black' : 'bg-surface-0 text-text-muted hover:text-text-secondary'}"
                onclick={() => tab = t as any}
              >
                {t}
              </button>
            {/each}
          </div>
          
          {#if tab === 'audit'}
             <div class="flex items-center gap-3">
                <select bind:value={framework} class="bg-surface-2 border border-border-subtle rounded-sm px-2 py-1 text-[10px] font-mono text-text-secondary focus:border-accent-primary focus:outline-none">
                   {#each FRAMEWORKS as fw}
                      <option value={fw}>{fw}</option>
                   {/each}
                </select>
                <div class="flex items-center gap-2">
                   <Calendar size={12} class="text-text-muted" />
                   <input type="date" bind:value={dateFrom} class="bg-surface-2 border border-border-subtle rounded-sm px-2 py-1 text-[10px] font-mono text-text-secondary" />
                   <span class="text-text-muted text-[10px]">→</span>
                   <input type="date" bind:value={dateTo} class="bg-surface-2 border border-border-subtle rounded-sm px-2 py-1 text-[10px] font-mono text-text-secondary" />
                </div>
                <Button variant="secondary" size="xs" onclick={fetchAudit}>APPLY FILTERS</Button>
             </div>
          {/if}
       </div>

       <div class="flex-1 overflow-auto">
          {#if loading}
             <div class="h-full flex items-center justify-center"><Spinner /></div>
          {:else if tab === 'audit'}
             <DataTable 
               data={auditEntries} 
               columns={[
                 { key: 'timestamp', label: 'EVENT_TIMESTAMP', width: '180px' },
                 { key: 'actor', label: 'ACTOR_IDENTITY', width: '200px' },
                 { key: 'action', label: 'ACTION_VERB', width: '150px' },
                 { key: 'resource', label: 'RESOURCE_TARGET' },
                 { key: 'outcome', label: 'OUTCOME', width: '120px' },
                 { key: 'entry_hash', label: 'INTEGRITY_HASH', width: '140px' }
               ]} 
               compact
               rowKey="id"
             >
                {#snippet cell({ column, row })}
                   {@const style = OUTCOME_STYLE[row.outcome]}
                   {#if column.key === 'timestamp'}
                      <span class="text-[10px] font-mono text-text-muted uppercase">{new Date(row.timestamp).toLocaleString()}</span>
                   {:else if column.key === 'actor'}
                      <div class="flex items-center gap-2">
                         <div class="w-1.5 h-1.5 rounded-full bg-accent-primary/40"></div>
                         <span class="text-[11px] font-bold text-text-heading uppercase">{row.actor}</span>
                      </div>
                   {:else if column.key === 'outcome'}
                      <div class="flex items-center gap-2">
                         <style.icon size={12} style="color: {style.color}" />
                         <span class="text-[9px] font-black uppercase tracking-widest italic" style="color: {style.color}">{row.outcome}</span>
                      </div>
                   {:else if column.key === 'entry_hash'}
                      <span class="text-[9px] font-mono text-text-muted opacity-40 uppercase tracking-tighter" title={row.entry_hash}>#{row.entry_hash.slice(0, 12)}...</span>
                   {:else}
                      <span class="text-[10px] text-text-secondary truncate block max-w-sm" title={row[column.key]}>{row[column.key]}</span>
                   {/if}
                {/snippet}
             </DataTable>
          {:else if tab === 'packages'}
             <div class="p-6 space-y-4 max-w-5xl mx-auto">
                {#each packages as pkg}
                   <div class="bg-surface-1 border border-border-primary p-5 rounded-sm flex justify-between items-center group hover:border-accent-primary transition-colors shadow-premium relative overflow-hidden">
                      <div class="absolute -left-2 -top-2 opacity-[0.03] grayscale">
                         <Lock size={120} />
                      </div>
                      <div class="flex flex-col gap-2 relative z-10">
                         <div class="flex items-center gap-3">
                            <Badge variant="accent" size="xs" class="font-bold">{pkg.framework}</Badge>
                            <span class="text-[13px] font-black text-text-heading uppercase tracking-tighter italic">{pkg.id}</span>
                         </div>
                         <div class="flex gap-4 text-[10px] font-mono text-text-muted uppercase tracking-widest opacity-60">
                            <span class="flex items-center gap-1.5"><Database size={12} /> {pkg.records.toLocaleString()} RECORDS</span>
                            <span class="flex items-center gap-1.5"><Calendar size={12} /> GENERATED {new Date(pkg.generated_at).toLocaleDateString()}</span>
                         </div>
                         <div class="flex items-center gap-2 text-[9px] font-mono text-text-muted uppercase tracking-tighter opacity-40">
                            <Fingerprint size={10} />
                            <span>Proof: {pkg.integrity_proof.slice(0, 32)}...</span>
                         </div>
                      </div>
                      <a href={pkg.download_url} download class="relative z-10 bg-accent-primary hover:bg-accent-secondary text-black px-6 py-2 rounded-xs font-black italic tracking-tighter text-[11px] transition-colors flex items-center gap-2">
                         <Download size={14} />
                         DOWNLOAD_PACKAGE
                      </a>
                   </div>
                {:else}
                   <div class="py-20 text-center opacity-40 flex flex-col items-center gap-4">
                      <FileText size={48} />
                      <p class="text-[10px] font-mono uppercase tracking-widest">No compliance packages archived</p>
                   </div>
                {/each}
             </div>
          {:else if tab === 'export'}
             <div class="p-12 flex items-center justify-center">
                <div class="bg-surface-1 border border-border-primary p-8 rounded-sm shadow-premium max-w-lg w-full space-y-8 relative overflow-hidden">
                   <div class="absolute right-0 top-0 w-24 h-24 bg-accent-primary/5 rounded-bl-full"></div>
                   <div class="space-y-2">
                      <h3 class="text-xl font-black text-text-heading uppercase italic tracking-tighter">Generate Evidence Shard</h3>
                      <p class="text-[11px] text-text-muted leading-relaxed">Select a regulatory framework to encapsulate and sign an immutable audit package for external review.</p>
                   </div>

                   <div class="space-y-6">
                      <div class="space-y-3">
                         <span class="text-[9px] font-black text-text-muted uppercase tracking-widest">Compliance Framework</span>
                         <div class="grid grid-cols-3 gap-2">
                            {#each FRAMEWORKS.filter(f => f !== 'ALL') as fw}
                               <button 
                                 class="p-2.5 rounded-xs border text-[10px] font-bold uppercase tracking-widest transition-all
                                   {selectedFw === fw ? 'bg-accent-primary/10 border-accent-primary text-accent-primary' : 'bg-surface-2 border-border-subtle text-text-muted hover:border-border-primary'}"
                                 onclick={() => selectedFw = fw}
                               >
                                  {fw}
                               </button>
                            {/each}
                         </div>
                      </div>

                      <div class="grid grid-cols-2 gap-4">
                         <div class="space-y-2">
                            <span class="text-[9px] font-black text-text-muted uppercase tracking-widest">Timeline Start</span>
                            <input type="date" bind:value={dateFrom} class="w-full bg-surface-2 border border-border-subtle rounded-sm px-3 py-2 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none" />
                         </div>
                         <div class="space-y-2">
                            <span class="text-[9px] font-black text-text-muted uppercase tracking-widest">Timeline End</span>
                            <input type="date" bind:value={dateTo} class="w-full bg-surface-2 border border-border-subtle rounded-sm px-3 py-2 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none" />
                         </div>
                      </div>

                      <div class="bg-surface-2 border border-border-subtle p-4 rounded-sm space-y-2">
                         <div class="flex items-center gap-2 text-[10px] font-bold text-text-heading uppercase">
                            <FileCheck size={14} class="text-status-online" />
                            Security Attestation
                         </div>
                         <p class="text-[9px] text-text-muted font-mono leading-relaxed opacity-60">
                            Package will include cryptographically-chained logs, Merkle proof, and Evidence Vault manifest signed by OBLIVRA_ORBIT_CA.
                         </p>
                      </div>

                      <Button variant="primary" class="w-full py-3 font-black italic tracking-tighter" onclick={generatePackage} loading={generating}>
                         GENERATE {selectedFw} AUDIT SHARD
                      </Button>
                   </div>
                </div>
             </div>
          {/if}
       </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>AUDIT_ENCLAVE:</span>
            <span class="text-status-online font-bold italic">LOCKED_IMMUTABLE</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>CHAIN_INTEGRITY:</span>
            <span class="text-status-online font-bold italic">VERIFIED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>REGULATOR_SIG:</span>
            <span class="text-accent-primary font-bold italic">RSA_4096_ACTIVE</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_REGULATOR_v6.6.2</div>
    </div>
  </div>
</PageLayout>

<style>
  :global(.flex-1::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  :global(.flex-1::-webkit-scrollbar-track) {
    background: var(--surface-0);
  }
  :global(.flex-1::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
</style>
