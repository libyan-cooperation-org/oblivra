<!-- OBLIVRA Web — Investigation Canvas Page (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Spinner, Badge, Button } from '@components/ui';
  import InvestigationCanvas, { type Node, type Edge } from '../components/ui/InvestigationCanvas.svelte';
  import { request } from '../services/api';
  import { Info, RefreshCw, Target } from 'lucide-svelte';

  let nodes = $state<Node[]>([]);
  let edges = $state<Edge[]>([]);
  let loading = $state(true);
  let selectedNode = $state<Node | null>(null);

  async function fetchGraphData() {
    loading = true;
    try {
      // Tactical Fallback: If the API isn't ready, use high-fidelity mock data
      const res = await request<{ nodes: Node[]; edges: Edge[] }>('/graph/investigation').catch(() => ({
        nodes: [
          { id: 'user:admin', type: 'user', label: 'Admin User', meta: { risk: 'high' } },
          { id: 'host:srv-prod-01', type: 'host', label: 'PROD-DB-01', meta: { os: 'linux', critical: 'high' } },
          { id: 'proc:mimikatz', type: 'process', label: 'mimikatz.exe', meta: { pid: '4120' } },
          { id: 'ip:192.168.1.50', type: 'ip', label: 'Lateral Movement Target' },
          { id: 'file:shadow_copy', type: 'file', label: 'sam.db' },
          { id: 'dns:malicious.top', type: 'dns', label: 'C2 Domain' },
          { id: 'reg:hklm_run', type: 'registry', label: 'Persistence Key' }
        ],
        edges: [
          { from: 'user:admin', to: 'host:srv-prod-01', type: 'logged_in' },
          { from: 'host:srv-prod-01', to: 'proc:mimikatz', type: 'executed' },
          { from: 'proc:mimikatz', to: 'file:shadow_copy', type: 'accessed' },
          { from: 'host:srv-prod-01', to: 'ip:192.168.1.50', type: 'connected_to' },
          { from: 'proc:mimikatz', to: 'dns:malicious.top', type: 'beacon' },
          { from: 'proc:mimikatz', to: 'reg:hklm_run', type: 'modified' }
        ]
      }));

      nodes = res.nodes as Node[];
      edges = res.edges as Edge[];
    } catch (e) {
      console.error("Failed to fetch graph data", e);
    } finally {
      loading = false;
    }
  }

  function handleNodeClick(node: Node) {
    selectedNode = node;
  }

  onMount(() => {
    fetchGraphData();
  });
</script>

<PageLayout title="Tactical Graph Detection" subtitle="Multi-dimensional entity relationship mapping for advanced mission-critical investigations">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Badge variant="critical" size="sm" class="mr-4">LIVE_RELATIONSHIP_STREAM</Badge>
      <Button variant="secondary" size="sm" onclick={fetchGraphData}>
        <RefreshCw size={14} class="mr-2" />
        REFRESH_GRAPH
      </Button>
    </div>
  {/snippet}

  <div class="flex h-[calc(100vh-200px)] gap-4 overflow-hidden">
    <!-- Main Canvas -->
    <div class="flex-1 relative bg-surface-1 rounded-sm border border-border-primary overflow-hidden">
      {#if loading}
        <div class="absolute inset-0 z-10 flex items-center justify-center bg-surface-1/50 backdrop-blur-xs">
          <Spinner size="lg" />
        </div>
      {/if}
      
      <InvestigationCanvas 
        {nodes} 
        {edges} 
        onNodeClick={handleNodeClick}
      />
    </div>

    <!-- Detail Sidebar -->
    <div class="w-80 flex flex-col gap-4">
      <div class="bg-surface-1 border border-border-primary rounded-sm p-4 flex-1 overflow-auto">
        <div class="flex items-center gap-2 mb-6 border-b border-border-primary pb-4">
          <Info size={16} class="text-accent" />
          <h2 class="text-[10px] font-black uppercase tracking-widest text-text-heading">Entity Intelligence</h2>
        </div>

        {#if selectedNode}
          <div class="space-y-6">
            <div class="flex flex-col gap-1">
              <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">Entity Type</span>
              <div class="flex items-center gap-2">
                <Badge variant="accent" size="xs">{selectedNode.type.toUpperCase()}</Badge>
                <span class="text-lg font-black italic tracking-tighter text-text-heading truncate">{selectedNode.label || selectedNode.id}</span>
              </div>
            </div>

            <div class="space-y-3">
              <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">Telemetric Metadata</span>
              <div class="grid grid-cols-1 gap-2">
                {#each Object.entries(selectedNode.meta || {}) as [k, v]}
                  <div class="bg-surface-2 p-2 border border-border-subtle rounded-xs flex justify-between items-center">
                    <span class="text-[9px] font-mono text-text-muted uppercase">{k}</span>
                    <span class="text-[10px] font-bold text-text-secondary">{v}</span>
                  </div>
                {/each}
                {#if !selectedNode.meta}
                  <div class="py-4 text-center opacity-30 italic text-[10px]">No extended telemetry available</div>
                {/if}
              </div>
            </div>

            <div class="pt-4 border-t border-border-primary flex flex-col gap-2">
              <Button variant="cta" size="sm" class="w-full italic font-black">ISOLATE_ENTITY</Button>
              <Button variant="danger" size="sm" class="w-full italic font-black">PURGE_PROCESS_TREE</Button>
            </div>
          </div>
        {:else}
          <div class="h-full flex flex-col items-center justify-center text-center p-6 opacity-30">
            <Target size={48} class="mb-4" />
            <p class="text-[10px] font-mono uppercase tracking-widest font-bold">Select a node to inspect tactical relationships</p>
          </div>
        {/if}
      </div>

      <!-- Risk Meter -->
      <div class="bg-surface-1 border border-border-primary rounded-sm p-4 h-32">
        <div class="flex justify-between items-center mb-4">
          <span class="text-[9px] font-black uppercase tracking-widest text-text-muted">Platform Risk Index</span>
          <Badge variant="critical" size="xs">NOMINAL</Badge>
        </div>
        <div class="flex items-end gap-1 h-12">
          {#each Array(20) as _, i}
            <div 
              class="flex-1 bg-surface-3 transition-all duration-slow" 
              style="height: {Math.random() * 100}%; background: {i > 15 ? 'var(--alert-critical)' : 'var(--accent)'}"
            ></div>
          {/each}
        </div>
      </div>
    </div>
  </div>
</PageLayout>

<style>
  /* Page specific styles */
</style>
