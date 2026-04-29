<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import * as echarts from 'echarts';
    import { nodesList, edgesList, type GraphNode, type GraphEdge } from '@lib/stores/graph.svelte';
    import { Activity, User, Monitor, Network, Info, Trash2 } from 'lucide-svelte';
    import { initGraphSync, refreshFullGraph } from '@lib/graph-sync';
    import { campaignStore } from '@lib/stores/campaigns.svelte';
    // Phase 36.9: KillProcess + IsolateHost imports removed (response-action chain
    // deleted in Phase 36; networkisolatorservice no longer exists). The graph
    // visualization remains; forensic-action buttons removed below.

    let chartDom: HTMLElement;
    let myChart: echarts.ECharts;
    let selectedNode = $state<any>(null);
    let unsubSync: () => void;

    // Mapping entity types to icons/colors
    const typeConfigs: Record<string, { color: string, icon: any }> = {
        'host': { color: '#3b82f6', icon: Monitor },
        'user': { color: '#ec4899', icon: User },
        'process': { color: '#f59e0b', icon: Activity },
        'ip': { color: '#8b5cf6', icon: Network },
        'file': { color: '#10b981', icon: Info }
    };

    function updateChart() {
        if (!myChart) return;

        const data = $nodesList.map((n: GraphNode) => {
            const inCluster = $campaignStore.some(c => c.entities.includes(n.id));
            const baseColor = typeConfigs[n.type]?.color || '#94a3b8';

            return {
                id: n.id,
                name: (n.id || "").split(':').pop(), // Show short name
                value: n.type,
                category: n.type,
                itemStyle: {
                    color: baseColor,
                    shadowBlur: inCluster ? 20 : 0,
                    shadowColor: inCluster ? '#ef4444' : 'transparent',
                    borderColor: inCluster ? '#ffffff' : 'transparent',
                    borderWidth: inCluster ? 2 : 0
                },
                label: {
                    show: true,
                    position: 'right',
                    formatter: '{b}'
                },
                meta: n.meta,
                symbolSize: inCluster ? 20 : 15
            };
        });

        const links = $edgesList.map((e: GraphEdge) => ({
            source: e.from,
            target: e.to,
            label: {
                show: false,
                formatter: e.type
            },
            lineStyle: {
                opacity: 0.6,
                curveness: 0.1
            }
        }));

        const categories = Object.keys(typeConfigs).map(t => ({ name: t }));

        myChart.setOption({
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'item',
                formatter: (params: any) => {
                    if (params.dataType === 'node') {
                        return `<div class="p-2 font-mono text-xs">
                                    <div class="font-bold border-b border-white/20 pb-1 mb-1">${params.data.id}</div>
                                    <div class="opacity-70">Type: ${params.data.category}</div>
                                </div>`;
                    }
                    return `Relation: ${params.data.label?.formatter}`;
                }
            },
            legend: [{
                data: categories.map(c => c.name),
                textStyle: { color: '#94a3b8' },
                bottom: 10
            }],
            series: [{
                type: 'graph',
                layout: 'force',
                data: data,
                links: links,
                categories: categories,
                roam: true,
                label: {
                    position: 'right'
                },
                force: {
                    repulsion: 200,
                    edgeLength: 100
                },
                emphasis: {
                    focus: 'adjacency',
                    lineStyle: { width: 4 }
                }
            }]
        });
    }

    onMount(() => {
        myChart = echarts.init(chartDom, 'dark');
        unsubSync = initGraphSync();
        refreshFullGraph();

        myChart.on('click', (params: any) => {
            if (params.dataType === 'node') {
                selectedNode = params.data;
            } else {
                selectedNode = null;
            }
        });

        window.addEventListener('resize', () => myChart?.resize());
    });

    onDestroy(() => {
        unsubSync?.();
        myChart?.dispose();
    });

    // Reactive update when store changes
    $effect(() => {
        if ($nodesList && $edgesList && $campaignStore) {
            updateChart();
        }
    });

    // Phase 36.9: handleIsolate / handleKillProcess removed — response actions
    // deleted in Phase 36 broad scope cut. The graph remains a visualization
    // surface; pivot to forensic profile / SIEM search for downstream work.
</script>

<div class="flex flex-col h-full w-full bg-slate-950/50 backdrop-blur-xl border border-white/5 overflow-hidden">
    <div class="flex items-center justify-between p-4 border-b border-white/5">
        <div class="flex items-center gap-3">
            <div class="p-2 bg-blue-500/10 rounded-lg">
                <Network class="w-5 h-5 text-blue-400" />
            </div>
            <div>
                <h2 class="text-sm font-bold text-white uppercase tracking-widest">Sovereign Threat Graph</h2>
                <p class="text-[10px] text-slate-400 font-mono">LIVE ENTITY CORRELATION ENGINE</p>
            </div>
        </div>
        
        <div class="flex items-center gap-2">
            <div class="flex items-center gap-2 px-3 py-1 bg-green-500/10 border border-green-500/20 rounded-full">
                <div class="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse"></div>
                <span class="text-[10px] font-mono text-green-400 uppercase">Live Streaming</span>
            </div>
        </div>
    </div>

    <div class="flex h-full min-h-0">
        <!-- Interactive Graph Area -->
        <div bind:this={chartDom} class="flex-1 min-w-0 h-full"></div>

        <!-- Forensic Detail Sidebar -->
        {#if selectedNode}
            <div class="w-80 border-l border-white/5 bg-slate-900/40 p-4 flex flex-col gap-4 animate-in slide-in-from-right fade-in duration-300">
                <div class="flex items-center justify-between">
                    <span class="text-[10px] font-mono text-slate-400 uppercase">Entity Analysis</span>
                    <button onclick={() => selectedNode = null}>
                        <Trash2 class="w-3.5 h-3.5 text-slate-500 hover:text-white transition-colors" />
                    </button>
                </div>

                <div class="flex flex-col items-center gap-3 py-4">
                    <div class="p-4 rounded-full bg-black/40 border border-white/10 shadow-2xl relative">
                        {#if selectedNode.category === 'host'}<Monitor class="w-8 h-8 text-blue-400" />
                        {:else if selectedNode.category === 'user'}<User class="w-8 h-8 text-pink-400" />
                        {:else if selectedNode.category === 'process'}<Activity class="w-8 h-8 text-amber-400" />
                        {:else}<Info class="w-8 h-8 text-slate-400" />
                        {/if}
                        
                        <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full border-2 border-slate-900"></div>
                    </div>
                    <div class="text-center">
                        <div class="text-sm font-bold text-white truncate max-w-[240px]">{selectedNode.id}</div>
                        <div class="text-[10px] font-mono text-slate-500 uppercase">{selectedNode.category} ENTITY</div>
                    </div>
                </div>

                <div class="flex-1 space-y-3 overflow-y-auto pr-1">
                    <div class="p-3 bg-white/5 rounded-lg border border-white/5">
                        <div class="text-[10px] font-bold text-slate-500 uppercase mb-2">Properties</div>
                        {#each Object.entries(selectedNode.meta || {}) as [key, val]}
                            <div class="flex justify-between items-start gap-4 mb-2">
                                <span class="text-[10px] font-mono text-slate-400">{key}</span>
                                <span class="text-[10px] font-mono text-white text-right break-all">{val}</span>
                            </div>
                        {/each}
                    </div>

                    <!-- Phase 36.9: Forensic-action panel (QUARANTINE / TERMINATE)
                         removed with response-action chain. Pair detection signals
                         with external SOAR for active response. -->

                </div>
            </div>
        {:else}
             <div class="w-80 border-l border-white/5 bg-slate-900/20 p-8 flex flex-col items-center justify-center text-center opacity-40">
                <Network class="w-12 h-12 text-slate-600 mb-4" />
                <div class="text-xs font-bold text-slate-500 uppercase tracking-widest">No Entity Selected</div>
                <div class="text-[10px] font-mono text-slate-600 mt-2">SELECT A NODE TO REVEAL FORENSIC INTELLIGENCE AND ACTIVE RESPONSE CONTROLS</div>
            </div>
        {/if}
    </div>
</div>

<style>
    :global(.echarts-tooltip) {
        background: rgba(15, 23, 42, 0.9) !important;
        backdrop-filter: blur(8px) !important;
        border: 1px solid rgba(255, 255, 255, 0.1) !important;
        box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.5) !important;
    }
</style>
