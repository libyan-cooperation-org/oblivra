import { subscribe } from './bridge';
import { graphStore, type GraphNode, type GraphEdge } from './stores/graph.svelte';

/**
 * Initializes real-time synchronization between the backend GraphEngine
 * and the frontend AttackGraph store.
 */
export function initGraphSync() {
    console.info('[graph-sync] Initializing real-time graph synchronization');

    // 1. Listen for node updates
    const unsubNodes = subscribe<GraphNode>('graph.node_upserted', (node) => {
        graphStore.upsertNode(node);
    });

    // 2. Listen for new edges
    const unsubEdges = subscribe<GraphEdge>('graph.edge_created', (edge) => {
        graphStore.addEdge(edge);
    });

    // 3. Listen for full graph refreshes (if needed)
    const unsubStats = subscribe<{node_count: number, edge_count: number}>('graph.stats', (stats) => {
        // We could use this to trigger a fetch if the counts drift significantly
        console.debug('[graph-sync] Stats update:', stats);
    });

    return () => {
        unsubNodes();
        unsubEdges();
        unsubStats();
    };
}

/**
 * Loads a subgraph centered on a specific entity.
 * Uses the Wails bridge to call GraphService.GetSubGraph.
 */
export async function loadSubGraph(nodeID: string, hops: number = 2) {
    try {
        const { GetSubGraph } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice');
        const result = await GetSubGraph(nodeID, hops);
        
        if (result && result.nodes && result.edges) {
            graphStore.setSubGraph(result.nodes, result.edges);
        }
    } catch (err) {
        console.error('[graph-sync] Failed to load subgraph:', err);
        throw err;
    }
}
