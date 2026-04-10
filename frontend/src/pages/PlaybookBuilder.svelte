<!--
  OBLIVRA — Playbook Builder (Svelte 5)
  Designing tactical response playbooks: Automating containment and investigation logic.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield, BookOpen, Send, Sparkles, Activity, FileText, Bot, Zap, Play, Save, Plus } from 'lucide-svelte';

  const playbooks = [
    { id: 'PB-01', name: 'Ransomware Containment v4', type: 'Hardened', status: 'verified', coverage: '98%', steps: 4, triggers: 'File Entropy > 0.8' },
    { id: 'PB-02', name: 'Zero-Day Exfiltration Logic', type: 'Advanced', status: 'draft', coverage: '42%', steps: 6, triggers: 'WMI Query Anomaly' },
    { id: 'PB-03', name: 'Lateral Movement Neutralizer', type: 'Optimal', status: 'verified', coverage: '89%', steps: 3, triggers: 'Peer AS Drift' },
    { id: 'PB-04', name: 'Identity Spoofing Counter', type: 'Signed', status: 'verified', coverage: '92%', steps: 5, triggers: 'WMI Query Anomaly' },
  ];
</script>

<PageLayout title="Playbook Orchestration" subtitle="Mission-critical response logic: Designing, validating and deploying automated tactical playbooks">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Logic Audit</Button>
     <Button variant="primary" size="sm">Synthesize Playbook</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Response Logic" value="41" trend="stable" trendValue="Hardened" variant="success" />
      <KPI label="Auto-Synthesis" value="ON" trend="stable" trendValue="Advanced" variant="accent" />
      <KPI label="Model Delta" value="0.04" trend="stable" trendValue="Optimal" />
      <KPI label="Logic Integrity" value="VAL-OK" trend="stable" trendValue="Signed" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Tactical Playbook Repository
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={playbooks} columns={[
              { key: 'name', label: 'Playbook Designation' },
              { key: 'type', label: 'Architecture', width: '120px' },
              { key: 'coverage', label: 'TTP Coverage', width: '100px' },
              { key: 'status', label: 'Validation', width: '120px' }
            ]} compact striped>
              {#snippet render({ value, col, row })}
                {#if col.key === 'status'}
                   <Badge variant={value === 'verified' ? 'success' : 'muted'} dot={value === 'verified'}>{value.toUpperCase()}</Badge>
                {:else if col.key === 'name'}
                   <div class="flex flex-col gap-1 py-1">
                       <div class="flex items-center gap-2">
                          <BookOpen size={14} class="text-accent opacity-70" />
                          <span class="text-[11px] font-bold text-text-heading">{value}</span>
                       </div>
                       <div class="text-[9px] text-text-muted font-mono flex items-center gap-1">
                          <Zap size={10} /> {row.triggers}
                       </div>
                   </div>
                {:else if col.key === 'type'}
                   <span class="text-[10px] font-mono text-text-muted">{value}</span>
                {:else if col.key === 'coverage'}
                   <div class="flex items-center gap-2">
                      <div class="w-12 h-1 bg-surface-3 rounded-full overflow-hidden">
                         <div class="h-full bg-accent" style="width: {value}"></div>
                      </div>
                      <span class="text-[10px] font-mono">{value}</span>
                   </div>
                {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
            
            <div class="p-6 border-t border-border-primary/50 flex flex-col items-center justify-center text-center opacity-40 hover:opacity-100 transition-opacity cursor-pointer bg-surface-0/30">
               <Plus size={24} class="text-text-muted" />
               <span class="text-[9px] font-bold uppercase tracking-widest mt-2">Draft New containment Strategy</span>
            </div>
         </div>
      </div>

      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-premium">
            <div class="absolute inset-0 bg-gradient-to-br from-accent/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
            <Sparkles size={32} class="text-accent animate-pulse" />
            <Bot size={24} class="absolute top-2 right-2 text-text-muted opacity-20" />
            <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Synthetic Builder</h4>
            <p class="text-[10px] text-text-muted max-w-[180px]">Synthesize complex multi-stage response logic from forensic sightings.</p>
            <Button variant="cta" size="sm" class="w-full">Initialize Design</Button>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Logic Simulation Flow
            </div>
            <div class="space-y-3">
               {#each Array(3) as _}
                  <div class="p-2 bg-surface-2 border border-border-primary rounded flex items-center gap-3">
                     <div class="w-1 h-8 bg-accent rounded-full"></div>
                     <div class="flex-1">
                        <div class="text-[9px] font-bold text-text-heading uppercase">Isolation Event</div>
                        <div class="text-[8px] text-text-muted">Network shunting triggered via forensic pivot.</div>
                     </div>
                     <Button variant="ghost" size="sm"><Send size={10} /></Button>
                  </div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
