<script module>
  export interface Node {
    id: string;
    type: 'user' | 'host' | 'process' | 'file' | 'ip';
    label?: string;
    meta?: Record<string, string>;
    x?: number;
    y?: number;
    vx?: number;
    vy?: number;
  }

  export interface Edge {
    from: string;
    to: string;
    type: string;
    timestamp?: string;
  }
</script>

<script lang="ts">
  import { 
    User, 
    Server, 
    Cpu, 
    FileText, 
    Globe, 
    Maximize2, 
    Search,
    AlertTriangle
  } from 'lucide-svelte';

  let { nodes = [], edges = [], onNodeClick = (_n: Node) => {} } = $props();

  let canvasWidth = $state(0);
  let canvasHeight = $state(0);
  let simulation: any;
  let d3: any;

  // Internal state for the force simulation
  let internalNodes = $state<Node[]>([]);
  let internalEdges = $state<Edge[]>([]);

  $effect(() => {
    // Sync external props with internal simulation state
    if (nodes.length > 0) {
      updateSimulation();
    }
  });

  async function updateSimulation() {
    if (!d3) {
      try {
        d3 = await import('d3');
      } catch (err) {
        console.warn("D3 not found, falling back to static layout", err);
        return;
      }
    }

    // Preserve positions of existing nodes
    const nodeMap = new Map(internalNodes.map(n => [n.id, n]));
    internalNodes = nodes.map(n => ({
      ...n,
      x: nodeMap.get(n.id)?.x ?? canvasWidth / 2 + (Math.random() - 0.5) * 100,
      y: nodeMap.get(n.id)?.y ?? canvasHeight / 2 + (Math.random() - 0.5) * 100
    }));

    internalEdges = [...edges];

    if (simulation) simulation.stop();

    simulation = d3.forceSimulation(internalNodes)
      .force("link", d3.forceLink(internalEdges).id((d: any) => d.id).distance(120))
      .force("charge", d3.forceManyBody().strength(-300))
      .force("center", d3.forceCenter(canvasWidth / 2, canvasHeight / 2))
      .force("collision", d3.forceCollide().radius(40))
      .on("tick", () => {
        internalNodes = [...internalNodes];
        internalEdges = [...internalEdges];
      });
  }

  function getNodeIcon(type: string) {
    switch (type) {
      case 'user': return User;
      case 'host': return Server;
      case 'process': return Cpu;
      case 'file': return FileText;
      case 'ip': return Globe;
      default: return AlertTriangle;
    }
  }

  function getNodeColor(type: string) {
    switch (type) {
      case 'user': return 'var(--color-accent)';
      case 'host': return 'var(--color-primary)';
      case 'process': return 'var(--color-warning)';
      case 'file': return 'var(--color-success)';
      case 'ip': return 'var(--color-critical)';
      default: return 'var(--color-text-muted)';
    }
  }

  function handleDragStart(_event: any, node: Node) {
    if (!simulation) return;
    node.vx = 0;
    node.vy = 0;
    simulation.alphaTarget(0.3).restart();
    
    function dragged(e: any) {
      node.x = e.clientX;
      node.y = e.clientY;
    }

    function dragEnded() {
      simulation.alphaTarget(0);
      window.removeEventListener('mousemove', dragged);
      window.removeEventListener('mouseup', dragEnded);
    }

    window.addEventListener('mousemove', dragged);
    window.addEventListener('mouseup', dragEnded);
  }
</script>

<div 
  class="relative w-full h-full bg-surface-2 overflow-hidden rounded-lg border border-border-primary select-none cursor-grab active:cursor-grabbing"
  bind:clientWidth={canvasWidth}
  bind:clientHeight={canvasHeight}
>
  <!-- Background Grid -->
  <svg class="absolute inset-0 w-full h-full pointer-events-none opacity-[0.03]">
    <defs>
      <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
        <path d="M 40 0 L 0 0 0 40" fill="none" stroke="currentColor" stroke-width="1"/>
      </pattern>
    </defs>
    <rect width="100%" height="100%" fill="url(#grid)" />
  </svg>

  <svg class="w-full h-full">
    <!-- Edges -->
    <g class="edges">
      {#each internalEdges as edge}
        {@const source = typeof edge.from === 'object' ? edge.from : internalNodes.find(n => n.id === edge.from)}
        {@const target = typeof edge.to === 'object' ? edge.to : internalNodes.find(n => n.id === edge.to)}
        {#if source && target}
          <line
            x1={source.x}
            y1={source.y}
            x2={target.x}
            y2={target.y}
            stroke="var(--color-border-primary)"
            stroke-width="1"
            stroke-dasharray={edge.type === 'connected_to' ? '4 2' : 'none'}
            opacity="0.4"
          />
        {/if}
      {/each}
    </g>

    <!-- Nodes -->
    <g class="nodes">
      {#each internalNodes as node (node.id)}
        {@const Icon = getNodeIcon(node.type)}
        <g 
          class="node-group cursor-pointer"
          transform="translate({node.x},{node.y})"
          onclick={() => onNodeClick(node)}
          onmousedown={(e) => handleDragStart(e, node)}
          role="button"
          tabindex="0"
          onkeydown={(e) => e.key === 'Enter' && onNodeClick(node)}
        >
          <!-- Glow effect for nodes -->
          <circle 
            r="18" 
            fill={getNodeColor(node.type)} 
            opacity="0.1"
          />
          
          <!-- Main node circle -->
          <circle 
            r="14" 
            fill="var(--color-surface-1)" 
            stroke={getNodeColor(node.type)} 
            stroke-width="2"
            class="transition-all hover:scale-110"
          />
          
          <!-- Icon -->
          <g transform="translate(-7, -7)">
            <Icon 
              size={14} 
              color={getNodeColor(node.type)}
            />
          </g>

          <!-- Label -->
          <text
            y="28"
            text-anchor="middle"
            class="text-[9px] font-bold uppercase tracking-wider fill-text-heading pointer-events-none"
          >
            {node.label || node.id.split(':').pop()}
          </text>
          
          {#if node.meta?.criticality === 'high'}
            <circle r="4" cx="12" cy="-12" fill="var(--color-critical)" />
          {/if}
        </g>
      {/each}
    </g>
  </svg>

  <!-- Overlay Controls -->
  <div class="absolute bottom-4 left-4 flex flex-col gap-2">
    <div class="flex items-center gap-2 p-1 bg-surface-1/80 backdrop-blur-sm border border-border-primary rounded shadow-lg">
      <div class="flex items-center gap-1.5 px-2 py-1 border-r border-border-primary">
        <div class="w-2 h-2 rounded-full bg-accent"></div>
        <span class="text-[9px] font-bold uppercase tracking-tight">User</span>
      </div>
      <div class="flex items-center gap-1.5 px-2 py-1 border-r border-border-primary">
        <div class="w-2 h-2 rounded-full bg-primary"></div>
        <span class="text-[9px] font-bold uppercase tracking-tight">Host</span>
      </div>
      <div class="flex items-center gap-1.5 px-2 py-1 border-r border-border-primary">
        <div class="w-2 h-2 rounded-full bg-warning"></div>
        <span class="text-[9px] font-bold uppercase tracking-tight">Process</span>
      </div>
    </div>
  </div>

  <div class="absolute top-4 right-4 flex flex-col gap-2">
    <button class="p-2 bg-surface-1 hover:bg-surface-2 border border-border-primary rounded shadow-md text-text-muted transition-colors">
      <Maximize2 size={16} />
    </button>
    <button class="p-2 bg-surface-1 hover:bg-surface-2 border border-border-primary rounded shadow-md text-text-muted transition-colors">
      <Search size={16} />
    </button>
  </div>
</div>

<style>
  .node-group {
    transition: transform 0.05s linear;
  }
  
  .node-group:hover circle:first-child {
    opacity: 0.3;
    r: 22;
  }

  svg {
    user-select: none;
  }
</style>
