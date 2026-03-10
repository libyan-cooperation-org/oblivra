import { Component, createEffect, onCleanup, onMount } from 'solid-js';
import * as echarts from 'echarts';
import { useApp } from '@core/store';

export const GlobalTopology: Component = () => {
    const [state] = useApp();
    let chartDom!: HTMLDivElement;
    let chartInstance: echarts.ECharts | undefined;

    onMount(() => {
        chartInstance = echarts.init(chartDom, 'dark');

        const handleResize = () => {
            chartInstance?.resize();
        };
        window.addEventListener('resize', handleResize);

        onCleanup(() => {
            window.removeEventListener('resize', handleResize);
            chartInstance?.dispose();
        });
    });

    createEffect(() => {
        if (!chartInstance) return;

        const hosts = state.hosts || [];

        interface GraphNode {
            id: string;
            name: string;
            symbolSize: number;
            category: number;
            value?: number;
            label?: any;
        }

        interface GraphLink {
            source: string;
            target: string;
            lineStyle?: any;
        }

        const nodes: GraphNode[] = [];
        const links: GraphLink[] = [];
        const categories = [
            { name: 'Core' },
            { name: 'Servers' },
            { name: 'Stacks' }
        ];

        // 1. Core Node
        nodes.push({
            id: 'root',
            name: 'OblivraShell',
            symbolSize: 60,
            category: 0,
            label: { show: true, fontSize: 16, fontWeight: 'bold' }
        });

        const uniqueTags = new Set<string>();

        // 2. Iterate Hosts
        hosts.forEach(host => {
            const hostNodeId = `host-${host.id}`;
            const hostLabel = host.label || host.hostname;

            // Server Node
            nodes.push({
                id: hostNodeId,
                name: hostLabel,
                symbolSize: 40,
                category: 1,
                label: { show: true }
            });

            // Link Core -> Server
            links.push({
                source: 'root',
                target: hostNodeId
            });

            // 3. Iterate Tags
            if (host.tags && host.tags.length > 0) {
                host.tags.forEach(tag => {
                    const formatTag = tag.charAt(0).toUpperCase() + tag.slice(1);
                    const tagNodeId = `tag-${formatTag}`;

                    // Add unique Tag node if not present
                    if (!uniqueTags.has(formatTag)) {
                        uniqueTags.add(formatTag);
                        nodes.push({
                            id: tagNodeId,
                            name: formatTag,
                            symbolSize: 30,
                            category: 2,
                            label: { show: true, fontStyle: 'italic', color: '#a8bbd9' }
                        });
                    }

                    // Link Server -> Tag
                    links.push({
                        source: hostNodeId,
                        target: tagNodeId,
                        lineStyle: { type: 'dashed', opacity: 0.5 }
                    });
                });
            }
        });

        const option: echarts.EChartsOption = {
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'item',
                formatter: '{b}'
            },
            legend: [{
                data: categories.map(a => a.name),
                textStyle: { color: '#8b9bb4' },
                bottom: 20
            }],
            animationDuration: 1500,
            animationEasingUpdate: 'quinticInOut',
            series: [
                {
                    type: 'graph',
                    layout: 'force',
                    force: {
                        repulsion: 800,
                        edgeLength: [50, 200],
                        gravity: 0.1
                    },
                    data: nodes,
                    links: links,
                    categories: categories,
                    roam: true,
                    label: {
                        position: 'right',
                        formatter: '{b}'
                    },
                    lineStyle: {
                        color: 'source',
                        curveness: 0.3
                    },
                    emphasis: {
                        focus: 'adjacency',
                        lineStyle: {
                            width: 3
                        }
                    }
                }
            ]
        };

        chartInstance.setOption(option);
    });

    return (
        <div style={{ width: '100%', height: '100%', padding: '24px', display: 'flex', 'flex-direction': 'column' }}>
            <div style={{ "margin-bottom": '16px' }}>
                <h2 style={{ margin: 0, "font-size": '22px', "font-weight": 600, color: 'var(--text-primary)' }}>Global Topology</h2>
                <p style={{ margin: '4px 0 0 0', "font-size": '13px', color: 'var(--text-secondary)' }}>Live force-directed graph illustrating active connections and discovered software stacks.</p>
            </div>
            <div
                ref={chartDom}
                style={{ flex: 1, "background-color": 'var(--bg-panel)', "border-radius": '12px', border: '1px solid var(--border-color)', overflow: 'hidden' }}
            />
        </div>
    );
};
