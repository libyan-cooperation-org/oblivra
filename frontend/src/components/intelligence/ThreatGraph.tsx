import { Component, createSignal, onMount, For } from 'solid-js';
import { GetSubGraph } from '../../../wailsjs/go/app/GraphService';

interface GraphNode {
    id: string;
    type: string;
    meta?: Record<string, string>;
    x?: number;
    y?: number;
}

interface GraphEdge {
    from: string;
    to: string;
    type: string;
}

export const ThreatGraph: Component = () => {
    const [nodes, setNodes] = createSignal<GraphNode[]>([]);
    const [edges, setEdges] = createSignal<GraphEdge[]>([]);
    const [loading, setLoading] = createSignal(true);

    onMount(async () => {
        try {
            // Placeholder: Initial discovery of the graph
            // In a real scenario, this would follow a specific incident's principal entity
            const data = await GetSubGraph("system-root", 2);

            // Simple random layout for initial render
            const initializedNodes = data.nodes.map((n: any) => ({
                ...n,
                x: 100 + Math.random() * 600,
                y: 100 + Math.random() * 400
            }));

            setNodes(initializedNodes);
            setEdges(data.edges);
        } catch (err) {
            console.error("Failed to load threat graph:", err);
        } finally {
            setLoading(false);
        }
    });

    const getNodeColor = (type: string) => {
        switch (type) {
            case 'user': return '#3b82f6'; // Blue
            case 'host': return '#10b981'; // Green
            case 'process': return '#f59e0b'; // Amber
            case 'ip': return '#ef4444'; // Red
            default: return '#6b7280'; // Gray
        }
    };

    return (
        <div class="threat-graph-container" style={{ height: 'calc(100vh - 120px)', background: '#0a0a0c', position: 'relative', overflow: 'hidden' }}>
            <div class="graph-header" style={{ padding: '1rem', 'border-bottom': '1px solid #1f2937' }}>
                <h2 style={{ color: '#e5e7eb', margin: 0, "font-size": '1.25rem' }}>Security Graph Intelligence</h2>
                <p style={{ color: '#9ca3af', "font-size": '0.875rem' }}>Cross-entity relationship analysis & attack path discovery</p>
            </div>

            {loading() ? (
                <div style={{ display: 'flex', 'align-items': 'center', 'justify-content': 'center', height: '100%' }}>
                    <div class="loader-cyan"></div>
                </div>
            ) : (
                <svg width="100%" height="100%" viewBox="0 0 800 600">
                    {/* Edges */}
                    <For each={edges()}>
                        {(edge) => {
                            const source = nodes().find(n => n.id === edge.from);
                            const target = nodes().find(n => n.id === edge.to);
                            if (!source || !target) return null;
                            return (
                                <line
                                    x1={source.x} y1={source.y}
                                    x2={target.x} y2={target.y}
                                    stroke="#374151"
                                    stroke-width="1"
                                    stroke-dasharray="4"
                                />
                            );
                        }}
                    </For>

                    {/* Nodes */}
                    <For each={nodes()}>
                        {(node) => (
                            <g transform={`translate(${node.x}, ${node.y})`}>
                                <circle r="12" fill={getNodeColor(node.type)} stroke="#ffffff" stroke-width="1.5" />
                                <text
                                    y="24"
                                    text-anchor="middle"
                                    fill="#9ca3af"
                                    style={{ "font-size": '10px', "font-family": 'monospace' }}
                                >
                                    {node.id}
                                </text>
                            </g>
                        )}
                    </For>
                </svg>
            )}

            <div class="graph-legend" style={{ position: 'absolute', bottom: '1rem', left: '1rem', background: 'rgba(0,0,0,0.8)', padding: '0.75rem', 'border-radius': '4px', border: '1px solid #1f2937' }}>
                <div style={{ "font-size": '0.75rem', color: '#9ca3af', "margin-bottom": '0.5rem' }}>LEGEND</div>
                <div style={{ display: 'flex', gap: '1rem' }}>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '0.25rem' }}>
                        <div style={{ width: '8px', height: '8px', background: '#3b82f6', 'border-radius': '50%' }}></div>
                        <span style={{ "font-size": '0.75rem', color: '#e5e7eb' }}>User</span>
                    </div>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '0.25rem' }}>
                        <div style={{ width: '8px', height: '8px', background: '#10b981', 'border-radius': '50%' }}></div>
                        <span style={{ "font-size": '0.75rem', color: '#e5e7eb' }}>Host</span>
                    </div>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '0.25rem' }}>
                        <div style={{ width: '8px', height: '8px', background: '#ef4444', 'border-radius': '50%' }}></div>
                        <span style={{ "font-size": '0.75rem', color: '#e5e7eb' }}>IP</span>
                    </div>
                </div>
            </div>
        </div>
    );
};
