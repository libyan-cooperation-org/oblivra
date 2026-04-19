<!-- OBLIVRA Web — Playbook Builder (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, Button, PageLayout } from '@components/ui';
  import { Play, Save, Plus, Trash2, ChevronUp, ChevronDown, Terminal, Shield, Activity, Lock, User, Scissors, Clipboard, Database, Globe } from 'lucide-svelte';
  import { History as HistoryIcon } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface PlaybookStep {
    id: string;
    action: string;
    params: Record<string, string>;
    enabled: boolean;
  }
  interface SavedPlaybook {
    id: string;
    name: string;
    steps: PlaybookStep[];
    created_at: string;
    last_run?: string;
  }

  // -- Constants --
  const STEP_ICONS: Record<string, any> = {
    isolate_host: Lock,
    lock_account: User,
    kill_process: Scissors,
    collect_logs: Clipboard,
    snapshot_memory: Database,
    notify_team: Activity,
    run_scan: Shield,
    block_ip: XCircle,
    webhook: Globe,
    default: Terminal,
  };

  import { XCircle } from 'lucide-svelte';

  // -- State --
  let availableActions = $state<string[]>([]);
  let savedPlaybooks   = $state<SavedPlaybook[]>([]);
  let steps            = $state<PlaybookStep[]>([]);
  let name             = $state('');
  let targetIncident   = $state('');
  let running          = $state(false);
  let saving           = $state(false);
  let result           = $state('');

  // -- Actions --
  async function fetchData() {
    try {
      const [a, p] = await Promise.all([
        request<{ actions: string[] }>('/playbooks/actions'),
        request<{ playbooks: SavedPlaybook[] }>('/playbooks')
      ]);
      availableActions = a.actions ?? [];
      savedPlaybooks = p.playbooks ?? [];
    } catch (e) {
      console.error('Playbook fetch failed', e);
      availableActions = ['isolate_host', 'lock_account', 'kill_process', 'collect_logs', 'notify_team', 'block_ip', 'webhook'];
    } finally {
      // Done
    }
  }

  const addStep = (action: string) => {
    steps = [...steps, {
      id: `step-${Date.now()}`,
      action, params: {}, enabled: true,
    }];
  };

  const removeStep = (id: string) => {
    steps = steps.filter(x => x.id !== id);
  };

  const toggleStep = (id: string) => {
    steps = steps.map(x => x.id === id ? { ...x, enabled: !x.enabled } : x);
  };

  const moveUp = (idx: number) => {
    if (idx === 0) return;
    const a = [...steps];
    [a[idx-1], a[idx]] = [a[idx], a[idx-1]];
    steps = a;
  };

  const moveDown = (idx: number) => {
    if (idx >= steps.length - 1) return;
    const a = [...steps];
    [a[idx], a[idx+1]] = [a[idx+1], a[idx]];
    steps = a;
  };

  async function savePlaybook() {
    if (!name.trim() || steps.length === 0) {
      result = '✗ Name and at least one step required.';
      return;
    }
    saving = true;
    try {
      await request('/playbooks', { method: 'POST', body: JSON.stringify({ name, steps }) });
      result = '✓ Playbook saved successfully.';
      fetchData();
    } catch (e: any) {
      result = '✗ ' + (e?.message ?? e);
    } finally {
      saving = false;
      setTimeout(() => result = '', 4000);
    }
  }

  async function executePlaybook() {
    if (!targetIncident.trim()) {
      result = '✗ Enter a target incident ID.';
      return;
    }
    if (steps.length === 0) {
      result = '✗ Add at least one step.';
      return;
    }
    running = true;
    try {
      await request('/playbooks/run', {
        method: 'POST',
        body: JSON.stringify({
          name: name || 'adhoc-playbook',
          steps: steps.filter(s => s.enabled),
          incident_id: targetIncident,
        })
      });
      result = `✓ Playbook executed against ${targetIncident}`;
    } catch (e: any) {
      result = '✗ ' + (e?.message ?? e);
    } finally {
      running = false;
      setTimeout(() => result = '', 5000);
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Playbook Builder" subtitle="Tactical SOAR automation: Assemble, verify, and orchestrate mission-critical response sequences">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <HistoryIcon size={14} class="mr-2" />
        RE-SYNC
      </Button>
      <Button variant="primary" size="sm" icon={Plus}>NEW TEMPLATE</Button>
    </div>
  {/snippet}

  <div class="flex h-full gap-4 -m-6 p-6 overflow-hidden bg-surface-0">
    <!-- LEFT: PALETTE -->
    <div class="w-72 flex flex-col gap-4 shrink-0 overflow-hidden">
       <!-- Action Palette -->
       <div class="bg-surface-1 border border-border-primary rounded-sm flex flex-col min-h-0 shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex items-center justify-between">
             <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Available Shards</span>
             <Badge variant="accent" size="xs">L7_ATOMS</Badge>
          </div>
          <div class="flex-1 overflow-auto p-2 space-y-1.5">
             {#each availableActions as action}
                {@const Icon = STEP_ICONS[action] || STEP_ICONS.default}
                <button 
                  class="w-full text-left p-2.5 rounded-sm border border-border-subtle bg-surface-2 hover:border-accent-primary group transition-all flex items-center gap-3"
                  onclick={() => addStep(action)}
                >
                   <div class="p-1.5 bg-surface-1 border border-border-subtle rounded-xs group-hover:border-accent-primary/40 transition-colors">
                      <Icon size={14} class="text-text-secondary group-hover:text-accent-primary" />
                   </div>
                   <span class="text-[10px] font-bold text-text-secondary uppercase group-hover:text-text-heading transition-colors">{action.replace(/_/g, ' ')}</span>
                   <Plus size={12} class="ml-auto text-accent-primary opacity-0 group-hover:opacity-100 transition-opacity" />
                </button>
             {/each}
          </div>
       </div>

       <!-- Saved Playbooks -->
       <div class="bg-surface-1 border border-border-primary rounded-sm flex-1 flex flex-col min-h-0 shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex items-center gap-2">
             <Save size={14} class="text-text-muted" />
             <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Saved Shards</span>
          </div>
          <div class="flex-1 overflow-auto p-2 space-y-1.5">
             {#each savedPlaybooks as pb}
                <button 
                  class="w-full text-left p-3 rounded-sm border border-border-subtle bg-surface-2 hover:border-accent-primary group transition-all"
                  onclick={() => { name = pb.name; steps = [...pb.steps]; }}
                >
                   <div class="text-[11px] font-bold text-text-heading uppercase tracking-tighter mb-1">{pb.name}</div>
                   <div class="flex justify-between items-center text-[8px] font-mono text-text-muted uppercase tracking-widest">
                      <span>{pb.steps.length} Steps</span>
                      <span>{new Date(pb.created_at).toLocaleDateString()}</span>
                   </div>
                </button>
             {:else}
                <div class="py-10 text-center opacity-40 text-[9px] font-mono uppercase">No saved sequences</div>
             {/each}
          </div>
       </div>
    </div>

    <!-- CENTER: CANVAS -->
    <div class="flex-1 flex flex-col min-w-0 bg-surface-1 border border-border-primary rounded-sm shadow-premium overflow-hidden">
       <div class="bg-surface-2 border-b border-border-primary p-4 flex items-center gap-4">
          <div class="p-2 bg-surface-1 border border-border-subtle rounded-xs text-accent-primary">
             <Terminal size={18} />
          </div>
          <input 
             bind:value={name}
             class="flex-1 bg-transparent text-xl font-black text-text-heading uppercase italic tracking-tighter focus:outline-none placeholder:opacity-20" 
             placeholder="UNTITLED_PLAYBOOK_SEQUENCE..."
          />
          <div class="flex items-center gap-2">
             <Badge variant="secondary" size="sm">{steps.length} STEPS</Badge>
          </div>
       </div>

       <div class="flex-1 overflow-auto p-8 relative">
          {#if steps.length === 0}
             <div class="h-full flex flex-col items-center justify-center gap-6 opacity-20 py-20">
                <div class="w-32 h-32 border-2 border-dashed border-text-muted rounded-full flex items-center justify-center">
                   <Plus size={48} />
                </div>
                <div class="text-center space-y-2">
                   <p class="text-sm font-black uppercase tracking-widest">Assemble Logic Bridge</p>
                   <p class="text-[10px] font-mono">Inject atomic actions from the left shard palette</p>
                </div>
             </div>
          {:else}
             <div class="max-w-3xl mx-auto space-y-4">
                {#each steps as step, idx}
                   {@const Icon = STEP_ICONS[step.action] || STEP_ICONS.default}
                   <div class="relative group animate-in slide-in-from-top-2 duration-300">
                      <!-- Connector Line -->
                      {#if idx < steps.length - 1}
                         <div class="absolute left-6 top-full h-4 w-px bg-border-primary z-0"></div>
                      {/if}
                      
                      <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex items-center gap-4 relative z-10 transition-all
                        {step.enabled ? 'border-l-4 border-l-accent-primary' : 'opacity-40 grayscale'}">
                         
                         <div class="flex flex-col gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button class="text-text-muted hover:text-text-heading" onclick={() => moveUp(idx)}><ChevronUp size={14} /></button>
                            <button class="text-text-muted hover:text-text-heading" onclick={() => moveDown(idx)}><ChevronDown size={14} /></button>
                         </div>

                         <div class="w-8 h-8 rounded-full bg-surface-1 border border-border-subtle flex items-center justify-center text-[10px] font-black text-text-muted shrink-0">
                            {idx + 1}
                         </div>

                         <div class="p-2 bg-surface-3 border border-border-subtle rounded-xs shrink-0">
                            <Icon size={16} class="text-accent-primary" />
                         </div>

                         <div class="flex-1 min-w-0">
                            <div class="text-[12px] font-black text-text-heading uppercase tracking-tighter">{step.action.replace(/_/g, ' ')}</div>
                            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Shard_ID: {step.id}</div>
                         </div>

                         <div class="flex items-center gap-3">
                            <button 
                               class="text-[9px] font-black uppercase tracking-widest px-3 py-1 rounded-xs border transition-colors
                                 {step.enabled ? 'border-status-online text-status-online hover:bg-status-online/10' : 'border-text-muted text-text-muted hover:text-text-heading'}"
                               onclick={() => toggleStep(step.id)}
                            >
                               {step.enabled ? 'ACTIVE' : 'MUTED'}
                            </button>
                            <button class="text-text-muted hover:text-alert-critical transition-colors" onclick={() => removeStep(step.id)}>
                               <Trash2 size={16} />
                            </button>
                         </div>
                      </div>
                   </div>
                {/each}
             </div>
          {/if}
       </div>
    </div>

    <!-- RIGHT: EXECUTE -->
    <div class="w-72 flex flex-col gap-4 shrink-0 overflow-hidden">
       <div class="bg-surface-1 border border-border-primary rounded-sm p-5 space-y-6 shadow-premium h-full flex flex-col">
          <div class="space-y-4">
             <div class="flex items-center gap-2">
                <Play size={14} class="text-alert-critical" />
                <span class="text-[10px] font-black text-text-heading uppercase tracking-widest">Execute Sequence</span>
             </div>
             
             <div class="space-y-1.5">
                <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Target Incident ID</span>
                <input 
                   bind:value={targetIncident}
                   class="w-full bg-surface-2 border border-border-subtle rounded-sm px-3 py-2 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none" 
                   placeholder="INCIDENT-2026-X" 
                />
             </div>

             <Button 
                variant="danger" 
                class="w-full py-4 font-black italic tracking-tighter group overflow-hidden relative" 
                onclick={executePlaybook} 
                loading={running}
             >
                <div class="absolute inset-0 bg-white opacity-0 group-hover:opacity-10 transition-opacity"></div>
                RUN_PLAYBOOK_NOW
             </Button>

             <Button 
                variant="secondary" 
                class="w-full font-black italic tracking-tighter" 
                onclick={savePlaybook} 
                loading={saving}
             >
                SAVE_TO_VAULT
             </Button>

             {#if result}
                <div class="p-3 rounded-sm border animate-in fade-in duration-300 text-[10px] font-bold leading-tight
                   {result.startsWith('✓') ? 'bg-status-online/5 border-status-online/40 text-status-online' : 'bg-alert-critical/5 border-alert-critical/40 text-alert-critical'}"
                >
                   {result}
                </div>
             {/if}
          </div>

          <div class="mt-auto space-y-4 pt-6 border-t border-border-primary">
             <div class="flex items-center gap-2">
                <Activity size={14} class="text-accent-primary" />
                <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Sequence Metrics</span>
             </div>
             <div class="space-y-2">
                <div class="flex justify-between text-[10px] font-mono">
                   <span class="text-text-muted uppercase">Total Atoms</span>
                   <span class="font-bold text-text-heading">{steps.length}</span>
                </div>
                <div class="flex justify-between text-[10px] font-mono">
                   <span class="text-text-muted uppercase">Active Path</span>
                   <span class="font-bold text-status-online">{steps.filter(s => s.enabled).length}</span>
                </div>
                <div class="flex justify-between text-[10px] font-mono">
                   <span class="text-text-muted uppercase">Est. Latency</span>
                   <span class="font-bold text-accent-primary">&lt; 400ms</span>
                </div>
             </div>
             <div class="p-3 bg-surface-2 border border-border-subtle rounded-xs text-[8px] font-mono text-text-muted italic leading-relaxed">
                Sequence execution is atomic. Failures in intermediate steps will trigger rollback protocols unless 'CONTINUE_ON_FAIL' is defined.
             </div>
          </div>
       </div>
    </div>

    <!-- STATUS BAR -->
    <div class="fixed bottom-0 left-0 right-0 bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted z-50 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>SOAR_ENGINE:</span>
            <span class="text-status-online font-bold italic">CONNECTED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>SEQUENCE_INTEGRITY:</span>
            <span class="text-status-online font-bold italic">VALIDATED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>ORCHESTRATION_L8:</span>
            <span class="text-accent-primary font-bold italic">ARMED</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_SOAR_BUILDER v8.2.0</div>
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
