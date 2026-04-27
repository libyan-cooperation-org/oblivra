<script lang="ts">
  import { onMount } from 'svelte';
  import PageLayout from '@components/ui/PageLayout.svelte';
  import * as SuppressionService from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/suppressionservice.js';
  import * as AlertingService from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/alertingservice.js';
  import { toastStore } from '@lib/stores/toast.svelte';
  import Badge from '@components/ui/Badge.svelte';
  import Button from '@components/ui/Button.svelte';
  import Modal from '@components/ui/Modal.svelte';
  import Input from '@components/ui/Input.svelte';
  import { 
    ShieldOff, 
    Plus, 
    Trash2, 
    Power, 
    Search, 
    Filter,
    Clock,
    AlertTriangle,
    CheckCircle2,
    Settings2
  } from 'lucide-svelte';

  let rules = $state<any[]>([]);
  let detectionRules = $state<any[]>([]);
  let isLoading = $state(true);
  let showCreateModal = $state(false);
  let searchQuery = $state('');

  let newRule = $state({
    label: '',
    description: '',
    rule_id: '',
    field: 'host_id',
    value: '',
    is_regex: false,
    expires_at: '',
    is_active: true
  });

  const fields = [
    { id: 'host_id', label: 'Host ID' },
    { id: 'user_name', label: 'User Name' },
    { id: 'src_ip', label: 'Source IP' },
    { id: 'process_name', label: 'Process Name' },
    { id: 'event_type', label: 'Event Type' },
    { id: 'message', label: 'Message Content' }
  ];

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    isLoading = true;
    try {
      const [rulesList, dr] = await Promise.all([
        SuppressionService.ListRules(),
        AlertingService.GetDetectionRules()
      ]);
      rules = rulesList || [];
      detectionRules = dr || [];
    } catch (error) {
      toastStore.error('Failed to load suppression data');
    } finally {
      isLoading = false;
    }
  }

  async function handleCreateRule() {
    if (!newRule.label || !newRule.value) {
      toastStore.warn('Please provide a label and value');
      return;
    }

    try {
      await SuppressionService.CreateRule(newRule);
      toastStore.success('Suppression rule created');
      showCreateModal = false;
      await loadData();
      resetForm();
    } catch (error) {
      toastStore.error('Failed to create rule');
    }
  }

  async function handleToggleRule(rule: any) {
    try {
      await SuppressionService.ToggleRule(rule.id, !rule.is_active);
      await loadData();
      toastStore.success(`Rule ${!rule.is_active ? 'enabled' : 'disabled'}`);
    } catch (error) {
      toastStore.error('Failed to toggle rule');
    }
  }

  async function handleDeleteRule(id: string) {
    if (!confirm('Are you sure you want to delete this rule?')) return;
    try {
      await SuppressionService.DeleteRule(id);
      await loadData();
      toastStore.success('Rule deleted');
    } catch (error) {
      toastStore.error('Failed to delete rule');
    }
  }

  function resetForm() {
    newRule = {
      label: '',
      description: '',
      rule_id: '',
      field: 'host_id',
      value: '',
      is_regex: false,
      expires_at: '',
      is_active: true
    };
  }

  const filteredRules = $derived(
    rules.filter((r: any) => 
      r.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
      r.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      r.value.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  const stats = $derived({
    total: rules.length,
    active: rules.filter((r: any) => r.is_active).length,
    hits: rules.reduce((acc: number, r: any) => acc + (r.last_matched_at ? 1 : 0), 0)
  });

  function getRuleName(id: string) {
    if (!id) return 'Global (All Rules)';
    const rule = detectionRules.find(r => r.id === id);
    return rule ? rule.name : id;
  }
</script>

<PageLayout title="Suppression Manager">
  <div slot="actions" class="flex items-center gap-3">
    <div class="relative">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" size={16} />
      <input 
        type="text" 
        bind:value={searchQuery}
        placeholder="Filter rules..." 
        class="bg-surface-light border border-white/5 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:border-primary/50 transition-all w-64"
      />
    </div>
    <Button variant="primary" onclick={() => showCreateModal = true}>
      <Plus size={16} class="mr-2" />
      Create Rule
    </Button>
  </div>

  <div class="space-y-6">
    <!-- Stats Overview -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div class="bg-surface-light border border-white/5 rounded-xl p-6 relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
          <ShieldOff size={48} />
        </div>
        <div class="text-text-muted text-xs uppercase tracking-widest font-bold mb-1">Active Silencers</div>
        <div class="text-3xl font-mono font-bold text-primary">{stats.active}</div>
        <div class="text-[10px] text-text-muted mt-2 flex items-center gap-1">
          <CheckCircle2 size={10} class="text-success" />
          Reducing alert fatigue globally
        </div>
      </div>

      <div class="bg-surface-light border border-white/5 rounded-xl p-6 relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
          <Filter size={48} />
        </div>
        <div class="text-text-muted text-xs uppercase tracking-widest font-bold mb-1">Total Definitions</div>
        <div class="text-3xl font-mono font-bold">{stats.total}</div>
        <div class="text-[10px] text-text-muted mt-2 flex items-center gap-1">
          <Settings2 size={10} />
          Configured suppression logic
        </div>
      </div>

      <div class="bg-surface-light border border-white/5 rounded-xl p-6 relative overflow-hidden group">
        <div class="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
          <AlertTriangle size={48} />
        </div>
        <div class="text-text-muted text-xs uppercase tracking-widest font-bold mb-1">Match Activity</div>
        <div class="text-3xl font-mono font-bold text-accent">{stats.hits}</div>
        <div class="text-[10px] text-text-muted mt-2 flex items-center gap-1">
          <Clock size={10} />
          Rules triggered recently
        </div>
      </div>
    </div>

    <!-- Rules Table -->
    <div class="bg-surface-light border border-white/5 rounded-xl overflow-hidden">
      <table class="w-full text-left">
        <thead>
          <tr class="border-b border-white/5 bg-white/5">
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Rule Label</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Target Context</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Matching Criteria</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Last Hit</th>
            <th class="px-6 py-4 text-[10px] uppercase tracking-widest text-text-muted font-bold">Status</th>
            <th class="px-6 py-4 text-right"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-white/5">
          {#if isLoading}
            {#each Array(3) as _}
              <tr class="animate-pulse">
                <td colspan="6" class="px-6 py-8 h-16 bg-white/[0.02]"></td>
              </tr>
            {/each}
          {:else if filteredRules.length === 0}
            <tr>
              <td colspan="6" class="px-6 py-12 text-center">
                <div class="flex flex-col items-center gap-2 text-text-muted">
                  <ShieldOff size={32} class="opacity-20" />
                  <p class="text-sm">No suppression rules found</p>
                  <Button variant="ghost" size="sm" onclick={() => showCreateModal = true}>Create your first rule</Button>
                </div>
              </td>
            </tr>
          {:else}
            {#each filteredRules as rule}
              <tr class="hover:bg-white/[0.02] transition-colors group">
                <td class="px-6 py-4">
                  <div class="font-bold text-sm">{rule.label}</div>
                  <div class="text-[10px] text-text-muted truncate max-w-[200px]">{rule.description || 'No description'}</div>
                </td>
                <td class="px-6 py-4">
                  <Badge variant={rule.rule_id ? 'info' : 'warning'}>
                    {getRuleName(rule.rule_id)}
                  </Badge>
                </td>
                <td class="px-6 py-4">
                  <div class="flex items-center gap-2">
                    <span class="text-[10px] text-text-muted font-mono">{rule.field}:</span>
                    <span class="text-xs font-mono bg-black/20 px-1.5 py-0.5 rounded border border-white/5">
                      {rule.value}
                    </span>
                    {#if rule.is_regex}
                      <Badge variant="accent" class="text-[8px] px-1 py-0 uppercase">Regex</Badge>
                    {/if}
                  </div>
                </td>
                <td class="px-6 py-4 text-xs font-mono text-text-muted">
                  {rule.last_matched_at ? new Date(rule.last_matched_at).toLocaleString() : 'Never'}
                </td>
                <td class="px-6 py-4">
                  <Badge variant={rule.is_active ? 'success' : 'muted'}>
                    {rule.is_active ? 'Active' : 'Disabled'}
                  </Badge>
                </td>
                <td class="px-6 py-4 text-right">
                  <div class="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button variant="ghost" size="icon" onclick={() => handleToggleRule(rule)} title={rule.is_active ? 'Disable' : 'Enable'}>
                      <Power size={14} class={rule.is_active ? 'text-text-muted' : 'text-success'} />
                    </Button>
                    <Button variant="ghost" size="icon" onclick={() => handleDeleteRule(rule.id)}>
                      <Trash2 size={14} class="text-critical" />
                    </Button>
                  </div>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>

  <!-- Create Rule Modal -->
  <Modal open={showCreateModal} onClose={() => showCreateModal = false} title="New Suppression Rule">
    <div class="space-y-4">
      <div class="space-y-2">
        <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Rule Label</div>
        <Input bind:value={newRule.label} placeholder="e.g., Silence Dev Host Noise" />
      </div>

      <div class="space-y-2">
        <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Description</div>
        <textarea 
          bind:value={newRule.description} 
          class="w-full bg-surface-light border border-white/10 rounded-lg p-3 text-sm focus:outline-none focus:border-primary/50 min-h-[80px]"
          placeholder="Why are we suppressing this?"
        ></textarea>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-2">
          <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Target Detection</div>
          <select 
            bind:value={newRule.rule_id}
            class="w-full bg-surface-light border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary/50"
          >
            <option value="">Global (All Alerts)</option>
            {#each detectionRules as dr}
              <option value={dr.id}>{dr.name}</option>
            {/each}
          </select>
        </div>
        <div class="space-y-2">
          <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Field to Match</div>
          <select 
            bind:value={newRule.field}
            class="w-full bg-surface-light border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-primary/50"
          >
            {#each fields as f}
              <option value={f.id}>{f.label}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="space-y-2">
        <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Value to Silence</div>
        <div class="flex gap-2">
          <div class="flex-1">
            <Input bind:value={newRule.value} placeholder={newRule.is_regex ? "Regex pattern..." : "Exact value..."} />
          </div>
          <button 
            onclick={() => newRule.is_regex = !newRule.is_regex}
            class="px-3 rounded-lg border transition-all text-[10px] font-bold uppercase tracking-wider {newRule.is_regex ? 'bg-accent/20 border-accent text-accent' : 'bg-white/5 border-white/10 text-text-muted'}"
          >
            Regex
          </button>
        </div>
      </div>

      <div class="space-y-2">
        <div class="text-[10px] uppercase tracking-widest text-text-muted font-bold">Expiration (Optional)</div>
        <Input type="datetime-local" bind:value={newRule.expires_at} />
      </div>

      <div class="flex justify-end gap-3 pt-4">
        <Button variant="ghost" onclick={() => showCreateModal = false}>Cancel</Button>
        <Button variant="primary" onclick={handleCreateRule}>
          Create Rule
        </Button>
      </div>
    </div>
  </Modal>
</PageLayout>
