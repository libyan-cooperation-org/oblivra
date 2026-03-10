import { Component, createEffect, onCleanup, onMount } from 'solid-js';
import * as echarts from 'echarts';

interface ChartBlockProps {
    options: echarts.EChartsCoreOption;
    class?: string;
    style?: string;
    theme?: 'light' | 'dark';
}

export const ChartBlock: Component<ChartBlockProps> = (props) => {
    let chartRef: HTMLDivElement | undefined;
    let chartInstance: echarts.ECharts | null = null;
    let resizeObserver: ResizeObserver | null = null;

    onMount(() => {
        if (!chartRef) return;

        // Initialize ECharts instance
        chartInstance = echarts.init(chartRef, props.theme || 'dark', { renderer: 'canvas' });

        // Initial setup
        chartInstance.setOption(props.options);

        // Handle resizing automatically to fit parent flex/grid containers perfectly
        resizeObserver = new ResizeObserver(() => {
            chartInstance?.resize();
        });
        resizeObserver.observe(chartRef);

        onCleanup(() => {
            resizeObserver?.disconnect();
            chartInstance?.dispose();
        });
    });

    // Reactively update chart when props.options changes reference
    createEffect(() => {
        if (chartInstance && props.options) {
            chartInstance.setOption(props.options, { notMerge: false, lazyUpdate: true });
        }
    });

    return (
        <div
            ref={chartRef}
            class={props.class}
            style={`width: 100%; height: 100%; min-height: 200px; ${props.style || ''}`}
        />
    );
};
