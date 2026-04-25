<script lang="ts">
  import { PageLayout, Button, KPI, DataTable, Input } from '@components/ui';
  import { Layout, Plus, Save, Play, Settings, Trash2, Move, BarChart, Terminal, Eye } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { onMount } from 'svelte';

  interface Widget {
    id: string;
    title: string;
    type: 'kpi' | 'table' | 'chart';
    query: string;
    width: number; // 1-4 columns
    data?: any;
    loading?: boolean;
  }

  let dashboardTitle = $state('New Tactical Dashboard');
  let widgets = $state<Widget[]>([
    { id: 'w1', title: 'Total Events', type: 'kpi', query: 'events | count', width: 1 },
    { id: 'w2', title: 'Top Attacking IPs', type: 'table', query: 'events | where severity == "critical" | group count() by source_ip | sort -count | limit 5', width: 2 },
  ]);

  let isPreview = $state(false);

  async function refreshWidget(w: Widget) {
    w.loading = true;
    try {
      const result = await siemStore.executeQuery(w.query);
      w.data = result;
    } catch (e) {
      console.error('Widget query failed:', e);
    } finally {
      w.loading = false;
    }
  }

  function addWidget() {
    widgets.push({
      id: Math.random().toString(36).substr(2, 9),
      title: 'New Metric',
      type: 'kpi',
      query: 'events | count',
      width: 1
    });
  }

  function removeWidget(id: string) {
    widgets = widgets.filter(w => w.id !== id);
  }

  onMount(() => {
    widgets.forEach(refreshWidget);
  });
</script>

<PageLayout title="Dashboard Studio" subtitle="Design sovereign tactical visualizations using OQL-powered logic components">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={isPreview ? Eye : Layout} onclick={() => isPreview = !isPreview}>
        {isPreview ? 'EDITOR MODE' : 'PREVIEW'}
      </Button>
      <Button variant="primary" size="sm" icon={Save}>SAVE DASHBOARD</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Studio Header -->
    <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex items-center justify-between">
      <div class="flex items-center gap-4 flex-1">
        <div class="w-10 h-10 rounded-sm bg-accent/10 flex items-center justify-center border border-accent/20">
          <Layout size={20} class="text-accent" />
        </div>
        <div class="flex-1">
          <input 
            bind:value={dashboardTitle}
            class="bg-transparent border-none text-text-heading font-black text-lg focus:ring-0 w-full p-0"
            placeholder="Untitled Dashboard"
          />
          <p class="text-[10px] text-text-muted font-mono uppercase tracking-widest mt-1">ID: DASH-{Math.random().toString(36).substr(2, 6).toUpperCase()}</p>
        </div>
      </div>
      <div class="flex items-center gap-4">
        <div class="text-right">
          <p class="text-[9px] font-mono text-text-muted uppercase">Refresh Rate</p>
          <select class="bg-surface-3 border-none text-[10px] font-mono text-accent outline-none cursor-pointer">
            <option>1 MINUTE</option>
            <option>5 MINUTES</option>
            <option>REAL-TIME</option>
          </select>
        </div>
        <div class="h-8 w-px bg-border-primary"></div>
        <Button variant="ghost" size="sm" icon={Plus} onclick={addWidget}>ADD WIDGET</Button>
      </div>
    </div>

    <!-- Dashboard Canvas -->
    <div class="flex-1 overflow-auto">
      <div class="grid grid-cols-4 gap-6">
        {#each widgets as widget (widget.id)}
          <div 
            class="bg-surface-1 border border-border-primary rounded-sm flex flex-col shadow-premium transition-all relative group"
            style="grid-column: span {widget.width}"
          >
            <!-- Widget Header -->
            <div class="px-4 py-2 bg-surface-2 border-b border-border-primary flex items-center justify-between">
              {#if !isPreview}
                <div class="flex items-center gap-2">
                  <Move size={12} class="text-text-muted cursor-move opacity-0 group-hover:opacity-100 transition-opacity" />
                  <input 
                    bind:value={widget.title}
                    class="bg-transparent border-none text-[10px] font-bold text-text-heading uppercase tracking-widest focus:ring-0 p-0"
                  />
                </div>
              {:else}
                <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">{widget.title}</span>
              {/if}
              
              <div class="flex items-center gap-2">
                {#if !isPreview}
                  <select bind:value={widget.width} class="bg-surface-3 border-none text-[9px] font-mono text-text-muted outline-none px-1 rounded-xs">
                    <option value={1}>1X</option>
                    <option value={2}>2X</option>
                    <option value={3}>3X</option>
                    <option value={4}>4X</option>
                  </select>
                  <Button variant="ghost" size="xs" onclick={() => removeWidget(widget.id)}><Trash2 size={12} /></Button>
                {/if}
                <Button variant="ghost" size="xs" onclick={() => refreshWidget(widget)} loading={widget.loading}><Play size={10} /></Button>
              </div>
            </div>

            <!-- Widget Body -->
            <div class="p-6 flex-1 flex flex-col min-h-[160px]">
              {#if !isPreview}
                <div class="flex-1 flex flex-col gap-4">
                   <div class="flex-1 bg-black/40 rounded-sm p-3 font-mono text-[11px] relative">
                      <Terminal size={14} class="absolute top-2 right-2 text-accent opacity-30" />
                      <textarea 
                        bind:value={widget.query}
                        class="w-full h-full bg-transparent border-none outline-none text-text-secondary resize-none"
                      ></textarea>
                   </div>
                   <div class="flex items-center justify-between">
                      <div class="flex gap-2">
                        <Button variant={widget.type === 'kpi' ? 'primary' : 'secondary'} size="xs" onclick={() => widget.type = 'kpi'}>KPI</Button>
                        <Button variant={widget.type === 'table' ? 'primary' : 'secondary'} size="xs" onclick={() => widget.type = 'table'}>TABLE</Button>
                        <Button variant={widget.type === 'chart' ? 'primary' : 'secondary'} size="xs" onclick={() => widget.type = 'chart'}>CHART</Button>
                      </div>
                      <span class="text-[9px] font-mono text-text-muted italic">Press RUN to preview data</span>
                   </div>
                </div>
              {:else}
                {#if widget.type === 'kpi'}
                  <div class="flex-1 flex flex-col items-center justify-center">
                    <span class="text-4xl font-black text-text-heading tracking-tighter">
                      {widget.data ? (Array.isArray(widget.data) ? widget.data.length : widget.data.count || '0') : '—'}
                    </span>
                    <span class="text-[10px] font-mono text-text-muted uppercase mt-2">{widget.title}</span>
                  </div>
                {:else if widget.type === 'table'}
                  <div class="flex-1 overflow-auto border border-border-primary/50 rounded-xs">
                    <DataTable 
                      data={widget.data || []} 
                      columns={widget.data && widget.data.length > 0 ? Object.keys(widget.data[0]).map(k => ({ key: k, label: k.toUpperCase() })) : []}
                      compact
                    />
                  </div>
                {:else}
                  <div class="flex-1 flex flex-col items-center justify-center gap-2 opacity-40">
                    <BarChart size={32} />
                    <span class="text-[10px] font-mono uppercase tracking-widest">ECharts Renderer Ready</span>
                  </div>
                {/if}
              {/if}
            </div>
          </div>
        {/each}

        {#if !isPreview}
          <button 
            onclick={addWidget}
            class="bg-surface-1 border-2 border-dashed border-border-primary/50 rounded-sm flex flex-col items-center justify-center gap-3 py-12 hover:border-accent hover:bg-accent/5 transition-all group"
          >
            <div class="w-10 h-10 rounded-full bg-surface-2 flex items-center justify-center border border-border-primary group-hover:border-accent transition-colors">
              <Plus size={20} class="text-text-muted group-hover:text-accent transition-colors" />
            </div>
            <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest group-hover:text-accent transition-colors">Append Widget</span>
          </button>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>

<style>
  textarea {
    scrollbar-width: thin;
    scrollbar-color: var(--border-primary) transparent;
  }
</style>
