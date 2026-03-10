import { Component, For, createResource, onMount } from 'solid-js';
import { GetSubGraph } from '../../../../wailsjs/go/app/GraphService';

export const ThreatGraphPanel: Component = () => {
    // ... (rest of the component)
    // We'll fetch a subgraph starting from a root entity (e.g., the last alerted entity)
    const [graphData, { refetch }] = createResource(async () => {
        try {
            // Fetching a level-2 subgraph of "root" as a placeholder
            // In production, this would be tied to the current investigation context
            return await GetSubGraph("root", 2);
        } catch (e) {
            console.error("Failed to fetch graph data:", e);
            return { Nodes: [], Edges: [] };
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 20000);
        return () => clearInterval(interval);
    });

    return (
        <div class="h-full flex flex-col bg-surface-0 font-mono text-[11px] overflow-hidden">
            <div class="p-3 border-b border-border-primary flex justify-between items-center bg-surface-1">
                <span class="text-text-muted font-bold tracking-widest uppercase">Multi-Hop Threat Graph</span>
                <span class="text-accent-primary font-bold text-[9px]">G-QUANT ENGINE</span>
            </div>

            <div class="flex-1 overflow-auto p-4 relative bg-[#050505] bg-[radial-gradient(#111_1px,transparent_1px)] [background-size:20px_20px]">
                {/* Simplified Graph Representation (List of edges/paths) */}
                <div class="space-y-4">
                    <For each={graphData()?.Nodes || []}>
                        {(node: any) => (
                            <div class="flex flex-col gap-2">
                                <div class="flex items-center gap-2">
                                    <div class={`w-2 h-2 rounded-full ${node.Type === 'PROCESS' ? 'bg-purple-500' :
                                        node.Type === 'USER' ? 'bg-blue-500' : 'bg-red-500'} shadow-[0_0_5px_currentColor]`}></div>
                                    <span class="text-white font-bold uppercase">{node.Type}: {node.ID}</span>
                                </div>
                                <div class="pl-4 border-l border-gray-800 space-y-2">
                                    <For each={graphData()?.Edges.filter((e: any) => e.Source === node.ID) || []}>
                                        {(edge: any) => (
                                            <div class="flex items-center gap-3 text-[10px]">
                                                <span class="text-gray-600">--[{edge.Type}]--&gt;</span>
                                                <span class="text-gray-400 font-bold">{edge.Target}</span>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>
                        )}
                    </For>

                    {(!graphData()?.Nodes || graphData()?.Nodes.length === 0) && (
                        <div class="h-full flex flex-col items-center justify-center opacity-20 py-20">
                            <svg class="w-12 h-12 mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                                <circle cx="12" cy="12" r="3" />
                                <circle cx="19" cy="5" r="2" />
                                <circle cx="5" cy="19" r="2" />
                                <line x1="12" y1="12" x2="19" y2="5" />
                                <line x1="12" y1="12" x2="5" y2="19" />
                            </svg>
                            <span class="text-[10px] uppercase font-bold tracking-tighter">No Active Attack Paths Correlated</span>
                        </div>
                    )}
                </div>
            </div>

            <div class="p-2 border-t border-border-primary bg-surface-1 flex justify-between items-center px-4">
                <div class="flex gap-4">
                    <span class="text-[9px] text-text-muted">ENTITIES: <span class="text-accent-primary">{graphData()?.Nodes?.length || 0}</span></span>
                    <span class="text-[9px] text-text-muted">EDGES: <span class="text-accent-primary">{graphData()?.Edges?.length || 0}</span></span>
                </div>
                <button class="text-[9px] text-accent-primary font-bold hover:text-white transition-colors uppercase">Expand Full Graph</button>
            </div>
        </div>
    );
};
