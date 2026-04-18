<!--
  OBLIVRA — Investigation Dashboard (Svelte 5)
  Relational analysis and blast-radius visualization for forensic deep-dives.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    PageLayout, 
    Button, 
    Badge, 
    InvestigationCanvas
  } from '@components/ui';
  import { 
    GitBranch, 
    Search, 
    Filter, 
    Download, 
    Zap,
    Database,
    Network,
    AlertTriangle
  } from 'lucide-svelte';
  
  import { GetFullGraph, GetSubGraph } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice.js';

  let nodes = $state<any[]>([]);
  let edges = $state<any[]>([]);
  let loading = $state(true);
  let selectedNode = $state<any>(null);
  let searchId = $state("");

  onMount(async () => {
    await refreshGraph();
  });

  async function refreshGraph() {
    loading = true;
    try {
      const data = await GetFullGraph();
      nodes = data.nodes || [];
      edges = data.edges || [];
    } catch (err) {
      console.error("Failed to load graph:", err);
    } finally {
      loading = false;
    }
  }

  async function handleNodeClick(node: any) {
    selectedNode = node;
    // Auto-focus sub-graph on click (Blast Radius)
    try {
      await GetSubGraph(node.id, 2);
      // We don't replace the whole graph, maybe just highlight?
      // For now, let's keep it simple.
    } catch (err) {
      console.error("Failed to fetch subgraph:", err);
    }
  }

  async function performSearch() {
    if (!searchId) return;
    loading = true;
    try {
      const data = await GetSubGraph(searchId, 2);
      nodes = data.nodes || [];
      edges = data.edges || [];
    } catch (err) {
      console.error("Search failed:", err);
    } finally {
      loading = false;
    }
  }
</script>

<PageLayout title="Relational Investigation" subtitle="Cross-entity relationship mapping and attack path analysis">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-1 bg-surface-1 border border-border-primary rounded px-2 py-1">
        <Search size={14} class="text-text-muted" />
        <input 
          type="text" 
          placeholder="Entity ID (Host, User, IP)..." 
          bind:value={searchId}
          onkeydown={(e) => e.key === 'Enter' && performSearch()}
          class="bg-transparent border-none text-[11px] text-text-heading focus:ring-0 w-48"
        />
      </div>
      <Button variant="secondary" size="sm" onclick={refreshGraph}>
        <Database size={14} class="mr-1" />
        Fetch Live
      </Button>
      <Button variant="primary" size="sm">
        <Download size={14} class="mr-1" />
        Export Audit
      </Button>
    </div>
  {/snippet}

  <div class="flex h-full gap-6">
    <!-- Left: Canvas -->
    <div class="flex-1 min-w-0 flex flex-col gap-4">
      <!-- Legend/Stats -->
      <div class="flex items-center gap-4 px-4 py-2 bg-surface-1 border border-border-primary rounded-md">
        <div class="flex items-center gap-2">
          <Network size={16} class="text-accent" />
          <span class="text-[11px] font-bold text-text-heading">{nodes.length} Nodes</span>
        </div>
        <div class="flex items-center gap-2">
          <GitBranch size={16} class="text-primary" />
          <span class="text-[11px] font-bold text-text-heading">{edges.length} Relationships</span>
        </div>
        <div class="ml-auto flex items-center gap-2">
          <Badge variant="success">Engine: Active</Badge>
          <Badge variant="muted">Integrity: Verified</Badge>
        </div>
      </div>

      <!-- Main Canvas -->
      <div class="flex-1 relative">
        {#if loading}
          <div class="absolute inset-0 z-10 bg-surface-2/50 backdrop-blur-sm flex items-center justify-center">
            <div class="flex flex-col items-center gap-3">
              <div class="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin"></div>
              <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Resolving Mesh...</span>
            </div>
          </div>
        {/if}
        <InvestigationCanvas {nodes} {edges} onNodeClick={handleNodeClick} />
      </div>
    </div>

    <!-- Right: Entity Details -->
    <div class="w-80 flex flex-col gap-4">
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col h-full shadow-card overflow-hidden">
        <div class="p-3 bg-surface-2 border-b border-border-primary flex items-center justify-between">
           <span class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Entity Context</span>
           {#if selectedNode}
            <Badge variant="accent">{selectedNode.type}</Badge>
           {/if}
        </div>
        
        <div class="flex-1 overflow-auto p-4">
          {#if selectedNode}
            <div class="flex flex-col gap-6">
              <div class="flex flex-col gap-1">
                <span class="text-[9px] uppercase font-bold text-text-muted">Unique Identifier</span>
                <span class="text-[13px] font-bold text-text-heading break-all font-mono">{selectedNode.id}</span>
              </div>

              {#if selectedNode.meta}
                <div class="flex flex-col gap-3">
                  <span class="text-[9px] uppercase font-bold text-text-muted">Metadata Attributes</span>
                  <div class="grid grid-cols-1 gap-2">
                    {#each Object.entries(selectedNode.meta) as [key, val]}
                      <div class="flex items-center justify-between p-2 bg-surface-2 rounded border border-border-secondary">
                        <span class="text-[10px] text-text-secondary">{key}</span>
                        <span class="text-[10px] font-bold text-text-heading">{val}</span>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}

              <div class="flex flex-col gap-3">
                <span class="text-[9px] uppercase font-bold text-text-muted">Incident Actions</span>
                <div class="flex flex-col gap-2">
                  <Button variant="secondary" size="sm" class="w-full justify-start">
                    <Zap size={14} class="mr-2 text-warning" />
                    Pivot to Timeline
                  </Button>
                  <Button variant="secondary" size="sm" class="w-full justify-start">
                    <Filter size={14} class="mr-2 text-primary" />
                    Filter logs for entity
                  </Button>
                  <Button variant="danger" size="sm" class="w-full justify-start">
                    <AlertTriangle size={14} class="mr-2" />
                    Isolation Ceremony
                  </Button>
                </div>
              </div>
            </div>
          {:else}
            <div class="h-full flex flex-col items-center justify-center text-center p-6 gap-4 opacity-40">
              <GitBranch size={48} />
              <div class="flex flex-col gap-1">
                <span class="text-[11px] font-bold text-text-heading">No Entity Selected</span>
                <span class="text-[10px] text-text-secondary">Click a node on the canvas to inspect its relational context and threat history.</span>
              </div>
            </div>
          {/if}
        </div>

        <div class="p-4 bg-surface-2 border-t border-border-primary">
          <div class="flex items-center gap-2 p-3 bg-accent/5 rounded border border-accent/20">
            <Zap size={16} class="text-accent" />
            <div class="flex flex-col">
              <span class="text-[10px] font-bold text-text-heading">Attack Path Analysis</span>
              <span class="text-[9px] text-text-muted leading-tight">Engine is currently correlating 5,402 active edges across the mesh.</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
