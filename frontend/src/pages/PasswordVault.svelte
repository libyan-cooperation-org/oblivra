<!--
  OBLIVRA — Password Vault (Svelte 5)
  Secure, mission-critical credential storage and rotation orchestration.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Input } from '@components/ui';
  import { Lock, Eye, EyeOff, Database } from 'lucide-svelte';

  const credentials = [
    { id: 'C-01', label: 'prod-db-master', username: 'admin', type: 'SSH Key', safety: 'hardened', rot: '4d' },
    { id: 'C-02', label: 'aws-root-key', username: 'iam_root', type: 'IAM Secret', safety: 'warning', rot: '2h' },
    { id: 'C-03', label: 'edge-gateway-01', username: 'operator', type: 'Password', safety: 'hardened', rot: '12d' },
  ];

  let searchQuery = $state('');
  const filteredCreds = $derived(credentials.filter(c => c.label.toLowerCase().includes(searchQuery.toLowerCase())));

  let showSecrets = $state(false);
</script>

<PageLayout title="Password Vault" subtitle="Encrypted credential store verified by hardware-locked temporal audit">
  {#snippet toolbar()}
     <div class="flex items-center gap-2">
        <Input variant="search" placeholder="Filter vault..." bind:value={searchQuery} class="w-64" />
        <Button variant="secondary" size="sm" onclick={() => showSecrets = !showSecrets}>
           {#if showSecrets}<EyeOff size={14} />{:else}<Eye size={14} />{/if}
        </Button>
        <Button variant="primary" size="sm" icon="🛡️">New Secret</Button> 
     </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Managed Secrets" value={credentials.length} trend="stable" trendValue="Encrypted" />
      <KPI label="Rotation Health" value="94%" trend="stable" trendValue="Optimal" variant="success" />
      <KPI label="Hardware Status" value="LOCKED" trend="stable" trendValue="Yubikey Sync" variant="accent" />
      <KPI label="Access Density" value="High" trend="up" trendValue="24 Active" variant="warning" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Credential List -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Secure Credential Registry
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={filteredCreds} columns={[
              { key: 'label', label: 'Secret Descriptor' },
              { key: 'type', label: 'Primitive', width: '120px' },
              { key: 'safety', label: 'Hardening', width: '100px' },
              { key: 'rot', label: 'Rotation', width: '80px' },
              { key: 'action', label: '', width: '80px' }
            ]} density="compact">
               {#snippet cell({ column, row }: { column: any, row: any })}
                {#if column.key === 'safety'}
                   <Badge variant={row.safety === 'hardened' ? 'success' : 'warning'}>{row.safety}</Badge>
                {:else if column.key === 'label'}
                   <div class="flex items-center gap-2">
                      <Lock size={14} class="text-accent opacity-70" />
                      <div class="flex flex-col">
                         <span class="text-[11px] font-bold text-text-heading">{row.label}</span>
                         <span class="text-[9px] text-text-muted font-mono">{row.username}</span>
                      </div>
                   </div>
                {:else if column.key === 'action'}
                    <div class="flex gap-2">
                       <Button variant="ghost" size="sm">Copy</Button>
                       <Button variant="ghost" size="sm">Use</Button>
                    </div>
                {:else if column.key === 'rot'}
                   <span class="text-[10px] font-mono text-text-muted opacity-80">{row.rot}</span>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Vault Metrics & Controls -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group border-dashed">
            <Database size={48} class="text-accent opacity-40" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-muted uppercase tracking-widest">Temporal Rotation Engine</h4>
               <p class="text-[10px] text-text-muted mt-2">Credentials for <span class="text-accent font-bold">aws-root-key</span> will auto-rotate in 2 hours to maintain Zero-Trust posture.</p>
            </div>
            <Button variant="secondary" size="sm" class="w-full">Force Global Rotation</Button>
         </div>

         <div class="bg-surface-1 border border-border-primary rounded-md p-4 space-y-3">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Vault Audit Status</div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Entropy Score</span>
               <Badge variant="success">98.4</Badge>
            </div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Leaked Detection</span>
               <Badge variant="success">CLEAN</Badge>
            </div>
            <div class="flex items-center justify-between">
               <span class="text-[11px] text-text-secondary">Last Sync</span>
               <span class="text-[10px] font-mono text-text-muted">41s ago</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
