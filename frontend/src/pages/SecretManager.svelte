<script lang="ts">
  import { onMount } from 'svelte';
  import PageLayout from '@components/ui/PageLayout.svelte';
  import * as VaultService from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/vaultservice';
  import * as RotationService from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/rotationservice';
  import * as HostService from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/hostservice';
  import { toastStore } from '@lib/stores/toast.svelte';
  import Badge from '@components/ui/Badge.svelte';
  import Button from '@components/ui/Button.svelte';
  import { Shield, RefreshCw, Calendar, Clock, AlertTriangle, Search, Settings2, Lock, Key, ShieldAlert } from 'lucide-svelte';

  let credentials = $state<any[]>([]);
  let policies = $state<any[]>([]);
  let hosts = $state<any[]>([]);
  let loading = $state(true);
  let searchQuery = $state('');
  let filterType = $state('all');
  let selectedCredID = $state<string | null>(null);

  // Rotation setup state
  let showSetupModal = $state(false);
  let setupFreq = $state(90);
  let setupNotifyOnly = $state(false);

  const stats = $derived({
    total: credentials.length,
    activePolicies: policies.length,
    dueSoon: policies.filter(p => {
      const next = new Date(p.next_rotation);
      const soon = new Date();
      soon.setDate(soon.getDate() + 7);
      return next < soon;
    }).length,
    critical: credentials.filter(c => c.type === 'ssh_key').length // Keys are critical
  });

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    loading = true;
    try {
      const [credsList, policiesList, hostsList] = await Promise.all([
        VaultService.ListCredentials(''),
        RotationService.ListPolicies(),
        HostService.ListHosts()
      ]);
      credentials = credsList || [];
      policies = policiesList || [];
      hosts = hostsList || [];
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Load Failed', message: e.message });
    } finally {
      loading = false;
    }
  }

  function getPolicy(credId: string) {
    return policies.find(p => p.credential_id === credId);
  }

  function getLinkedHosts(credId: string) {
    return hosts.filter(h => h.credential_id === credId);
  }

  async function handleRotate(credId: string) {
    try {
      toastStore.add({ type: 'info', title: 'Rotation Started', message: 'Orchestrating secret lifecycle...' });
      await RotationService.RotateCredential(credId);
      toastStore.add({ type: 'success', title: 'Success', message: 'Credential rotated and deployed.' });
      await loadData();
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Rotation Failed', message: e.message });
    }
  }

  async function savePolicy() {
    if (!selectedCredID) return;
    try {
      await RotationService.RegisterPolicy(selectedCredID, setupFreq, setupNotifyOnly);
      toastStore.add({ type: 'success', title: 'Policy Saved', message: 'Rotation schedule active.' });
      showSetupModal = false;
      await loadData();
    } catch (e: any) {
      toastStore.add({ type: 'error', title: 'Save Failed', message: e.message });
    }
  }

  const filteredCreds = $derived(
    credentials.filter(c => {
      const matchesSearch = c.label.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesType = filterType === 'all' || c.type === filterType;
      return matchesSearch && matchesType;
    })
  );

  function formatDate(dateStr: string) {
    if (!dateStr) return 'Never';
    return new Date(dateStr).toLocaleDateString();
  }
</script>

<PageLayout title="Secret Lifecycle" subtitle="Automated Rotation & Governance">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <div class="relative group">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted group-focus-within:text-primary transition-colors" />
        <input 
          type="text" 
          placeholder="Search credentials..." 
          bind:value={searchQuery}
          class="bg-surface-2 border border-border-primary rounded-md pl-10 pr-4 py-1.5 text-sm focus:outline-none focus:border-primary/50 w-64 transition-all"
        />
      </div>
      <select 
        bind:value={filterType}
        class="bg-surface-2 border border-border-primary rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-primary/50"
      >
        <option value="all">All Types</option>
        <option value="ssh_key">SSH Keys</option>
        <option value="password">Passwords</option>
        <option value="token">API Tokens</option>
      </select>
      <Button variant="primary" icon={RefreshCw} onclick={loadData}>Refresh</Button>
    </div>
  {/snippet}

  <div class="space-y-6">
    <!-- Stats Grid -->
    <div class="grid grid-cols-4 gap-4">
      <div class="bg-surface-2 border border-border-primary rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center">
          <Shield class="w-6 h-6 text-primary" />
        </div>
        <div>
          <div class="text-2xl font-bold font-mono">{stats.total}</div>
          <div class="text-[10px] uppercase tracking-wider text-text-muted">Total Secrets</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-info/10 flex items-center justify-center">
          <RefreshCw class="w-6 h-6 text-info" />
        </div>
        <div>
          <div class="text-2xl font-bold font-mono">{stats.activePolicies}</div>
          <div class="text-[10px] uppercase tracking-wider text-text-muted">Active Policies</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-warning/10 flex items-center justify-center">
          <Calendar class="w-6 h-6 text-warning" />
        </div>
        <div>
          <div class="text-2xl font-bold font-mono">{stats.dueSoon}</div>
          <div class="text-[10px] uppercase tracking-wider text-text-muted">Due Soon</div>
        </div>
      </div>
      <div class="bg-surface-2 border border-border-primary rounded-xl p-4 flex items-center gap-4">
        <div class="w-12 h-12 rounded-lg bg-error/10 flex items-center justify-center">
          <ShieldAlert class="w-6 h-6 text-error" />
        </div>
        <div>
          <div class="text-2xl font-bold font-mono">{stats.critical}</div>
          <div class="text-[10px] uppercase tracking-wider text-text-muted">Critical Assets</div>
        </div>
      </div>
    </div>

    <!-- Main Content -->
    <div class="bg-surface-1 border border-border-primary rounded-xl overflow-hidden">
      <table class="w-full text-left border-collapse">
        <thead>
          <tr class="bg-surface-2/50 border-b border-border-primary">
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Credential</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Type</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Linked Hosts</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Policy</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border-primary/50">
          {#each filteredCreds as cred (cred.id)}
            {@const policy = getPolicy(cred.id)}
            {@const linkedHosts = getLinkedHosts(cred.id)}
            <tr class="hover:bg-primary/5 transition-colors group">
              <td class="px-6 py-4">
                <div class="flex items-center gap-3">
                  <div class="p-2 rounded bg-surface-3 border border-border-primary group-hover:border-primary/30 transition-colors">
                    {#if cred.type === 'ssh_key'}
                      <Key class="w-4 h-4 text-primary" />
                    {:else}
                      <Lock class="w-4 h-4 text-info" />
                    {/if}
                  </div>
                  <div>
                    <div class="font-medium text-sm">{cred.label}</div>
                    <div class="text-[10px] font-mono text-text-muted uppercase">ID: {cred.id.slice(0, 8)}</div>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4">
                <Badge variant={cred.type === 'ssh_key' ? 'info' : 'muted'} uppercase>{cred.type.replace('_', ' ')}</Badge>
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-1">
                  {#if linkedHosts.length > 0}
                    <Badge variant="muted">{linkedHosts.length} Hosts</Badge>
                  {:else}
                    <span class="text-[10px] text-text-muted italic">Unlinked</span>
                  {/if}
                </div>
              </td>
              <td class="px-6 py-4">
                {#if policy}
                  <div class="flex flex-col gap-1">
                    <div class="flex items-center gap-2 text-xs">
                      <Clock class="w-3 h-3 text-info" />
                      <span>Next: {formatDate(policy.next_rotation)}</span>
                    </div>
                    <div class="flex items-center gap-2 text-[10px] text-text-muted">
                      <RefreshCw class="w-3 h-3" />
                      Every {policy.frequency_days} days
                    </div>
                  </div>
                {:else}
                  <button 
                    class="text-[10px] text-primary hover:underline flex items-center gap-1"
                    onclick={() => { selectedCredID = cred.id; showSetupModal = true; }}
                  >
                    <Settings2 class="w-3 h-3" />
                    Setup Policy
                  </button>
                {/if}
              </td>
              <td class="px-6 py-4 text-right">
                <div class="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Button 
                    variant="muted" 
                    size="sm" 
                    icon={Settings2} 
                    onclick={() => { selectedCredID = cred.id; showSetupModal = true; }}
                  />
                  <Button 
                    variant="primary" 
                    size="sm" 
                    icon={RefreshCw} 
                    onclick={() => handleRotate(cred.id)}
                  >Rotate Now</Button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Setup Modal -->
  {#if showSetupModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-background/80 backdrop-blur-sm">
      <div class="bg-surface-1 border border-border-primary rounded-2xl shadow-2xl w-full max-w-md overflow-hidden">
        <div class="px-6 py-4 border-b border-border-primary flex items-center justify-between">
          <div class="flex items-center gap-2 font-bold">
            <Shield class="w-5 h-5 text-primary" />
            Rotation Policy Configuration
          </div>
          <button onclick={() => showSetupModal = false} class="text-text-muted hover:text-white">&times;</button>
        </div>
        <div class="p-6 space-y-4">
          <div class="space-y-2">
            <label class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Rotation Frequency (Days)</label>
            <div class="grid grid-cols-4 gap-2">
              {#each [30, 60, 90, 180] as days}
                <button 
                  class="py-2 rounded-md border transition-all text-xs font-mono {setupFreq === days ? 'bg-primary/20 border-primary text-primary' : 'bg-surface-2 border-border-primary text-text-muted'}"
                  onclick={() => setupFreq = days}
                >{days}d</button>
              {/each}
            </div>
            <input 
              type="number" 
              bind:value={setupFreq}
              class="w-full bg-surface-2 border border-border-primary rounded-md px-3 py-2 text-sm focus:outline-none focus:border-primary/50"
              placeholder="Custom days..."
            />
          </div>

          <div class="flex items-center gap-3 bg-surface-2 p-3 rounded-xl border border-border-primary">
            <input type="checkbox" bind:checked={setupNotifyOnly} class="accent-primary" />
            <div class="flex-1">
              <div class="text-xs font-bold">Notify Only</div>
              <div class="text-[10px] text-text-muted">Do not auto-deploy. Just warn me when rotation is due.</div>
            </div>
          </div>

          <div class="bg-warning/10 border border-warning/30 p-3 rounded-lg flex gap-3">
            <AlertTriangle class="w-4 h-4 text-warning shrink-0" />
            <p class="text-[10px] text-warning/80">
              For SSH keys, OBLIVRA will automatically push the new public key to all linked hosts before rotating the vault entry.
            </p>
          </div>
        </div>
        <div class="px-6 py-4 bg-surface-2 border-t border-border-primary flex justify-end gap-3">
          <Button variant="muted" onclick={() => showSetupModal = false}>Cancel</Button>
          <Button variant="primary" onclick={savePolicy}>Activate Policy</Button>
        </div>
      </div>
    </div>
  {/if}
</PageLayout>

<style>
  table thead th {
    position: sticky;
    top: 0;
    z-index: 10;
  }
</style>
