import { writable, derived } from 'svelte/store';

export interface GraphNode {
    id: string;
    type: 'user' | 'host' | 'process' | 'file' | 'ip';
    meta?: Record<string, string>;
    last_seen: string;
}

export interface GraphEdge {
    from: string;
    to: string;
    type: string;
    timestamp: string;
}

export interface GraphState {
    nodes: Map<string, GraphNode>;
    edges: GraphEdge[];
    loading: boolean;
    error: string | null;
}

function createGraphStore() {
    const { subscribe, set, update } = writable<GraphState>({
        nodes: new Map(),
        edges: [],
        loading: false,
        error: null
    });

    return {
        subscribe,
        upsertNode: (node: GraphNode) => update(s => {
            s.nodes.set(node.id, node);
            return s;
        }),
        addEdge: (edge: GraphEdge) => update(s => {
            // Check for duplicates
            const exists = s.edges.some(e => 
                e.from === edge.from && e.to === edge.to && e.type === edge.type
            );
            if (!exists) {
                s.edges = [...s.edges, edge];
            }
            return s;
        }),
        setSubGraph: (nodes: GraphNode[], edges: GraphEdge[]) => update(s => {
            s.nodes = new Map(nodes.map(n => [n.id, n]));
            s.edges = edges;
            return s;
        }),
        clear: () => set({
            nodes: new Map(),
            edges: [],
            loading: false,
            error: null
        })
    };
}

export const graphStore = createGraphStore();

// UI-friendly arrays
export const nodesList = derived(graphStore, $s => Array.from($s.nodes.values()));
export const edgesList = derived(graphStore, $s => $s.edges);
