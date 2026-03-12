import { Component, For, Show, createResource, onMount, onCleanup } from 'solid-js';
import { GetSubGraph } from '../../../../wailsjs/go/app/GraphService';

const nodeColor = (type: string) => {
    if (type === 'PROCESS') return 'var(--accent-secondary)';
    if (type === 'USER')    return 'var(--accent-primary)';
    return 'var(--alert-critical)';
};

export const ThreatGraphPanel: Component = () => {
    const [graphData, { refetch }] = createResource(async () => {
        try {
            return await GetSubGraph('root', 2);
        } catch (e) {
            console.error('Failed to fetch graph data:', e);
            return { Nodes: [], Edges: [] };
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 20000);
        onCleanup(() => clearInterval(interval));
    });

    const nodes = () => graphData()?.Nodes ?? [];
    const edges = () => graphData()?.Edges ?? [];

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Header */}
            <div style={{ padding: '8px 12px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', background: 'var(--surface-1)', 'flex-shrink': 0 }}>
                <span style={{ color: 'var(--text-muted)', 'font-weight': 800, 'letter-spacing': '2px', 'text-transform': 'uppercase', 'font-size': '10px' }}>Multi-Hop Threat Graph</span>
                <span style={{ 'font-size': '9px', color: 'var(--accent-primary)', 'font-weight': 800 }}>G-QUANT ENGINE</span>
            </div>

            {/* Graph body */}
            <div style={{
                flex: 1, 'overflow-y': 'auto', padding: '12px',
                background: 'var(--surface-0)',
                'background-image': 'radial-gradient(rgba(255,255,255,0.018) 1px, transparent 1px)',
                'background-size': '20px 20px',
                position: 'relative',
            }}>
                <Show when={nodes().length > 0} fallback={
                    <div style={{ height: '100%', display: 'flex', 'flex-direction': 'column', 'align-items': 'center', 'justify-content': 'center', opacity: '0.22', gap: '10px', 'padding-top': '40px' }}>
                        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2">
                            <circle cx="12" cy="12" r="3" /><circle cx="19" cy="5" r="2" /><circle cx="5" cy="19" r="2" />
                            <line x1="12" y1="12" x2="19" y2="5" /><line x1="12" y1="12" x2="5" y2="19" />
                        </svg>
                        <span style={{ 'font-size': '10px', 'text-transform': 'uppercase', 'letter-spacing': '1px' }}>No active attack paths</span>
                    </div>
                }>
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '14px' }}>
                        <For each={nodes()}>
                            {(node: any) => (
                                <div style={{ display: 'flex', 'flex-direction': 'column', gap: '6px' }}>
                                    {/* Node */}
                                    <div style={{ display: 'flex', 'align-items': 'center', gap: '8px' }}>
                                        <div style={{ width: '8px', height: '8px', 'border-radius': '50%', background: nodeColor(node.Type), 'box-shadow': `0 0 6px ${nodeColor(node.Type)}`, 'flex-shrink': 0 }} />
                                        <span style={{ color: 'var(--text-primary)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '11px' }}>
                                            {node.Type}: {node.ID}
                                        </span>
                                    </div>

                                    {/* Outbound edges */}
                                    <div style={{ 'padding-left': '16px', 'border-left': '1px solid var(--border-primary)', display: 'flex', 'flex-direction': 'column', gap: '4px' }}>
                                        <For each={edges().filter((e: any) => e.Source === node.ID)}>
                                            {(edge: any) => (
                                                <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', 'font-size': '10px' }}>
                                                    <span style={{ color: 'var(--text-muted)', opacity: '0.6' }}>──[{edge.Type}]──›</span>
                                                    <span style={{ color: 'var(--text-secondary)', 'font-weight': 700 }}>{edge.Target}</span>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>
            </div>

            {/* Footer */}
            <div style={{ padding: '6px 12px', 'border-top': '1px solid var(--border-primary)', background: 'var(--surface-1)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'flex-shrink': 0 }}>
                <div style={{ display: 'flex', gap: '16px', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                    <span>ENTITIES: <span style={{ color: 'var(--accent-primary)' }}>{nodes().length}</span></span>
                    <span>EDGES: <span style={{ color: 'var(--accent-primary)' }}>{edges().length}</span></span>
                </div>
                <button style={{ 'font-size': '9px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', 'font-weight': 800, 'font-family': 'var(--font-mono)', 'text-transform': 'uppercase' }}>
                    EXPAND FULL GRAPH →
                </button>
            </div>
        </div>
    );
};
