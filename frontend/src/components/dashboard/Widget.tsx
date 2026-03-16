import { Component, createSignal, onMount, onCleanup, createEffect, Switch, Match, Show, For } from 'solid-js';
import * as echarts from 'echarts';
import { RunWidgetQuery, RunOsquery } from '../../../wailsjs/go/services/AnalyticsService';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { Badge } from '../ui/TacticalComponents';
import type { Widget as WidgetConfig } from '../../types/dashboard';

interface WidgetProps {
    config: WidgetConfig;
    sessionId?: string;
    timeRange: string;
}

export const Widget: Component<WidgetProps> = (props) => {
    const [data, setData] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);
    let chartRef: HTMLDivElement | undefined;
    let chartInstance: echarts.ECharts | undefined;
    let timer: ReturnType<typeof setInterval> | undefined;

    const fetchData = async () => {
        setLoading(true);
        setError(null);
        try {
            let result: Record<string, unknown>[];
            if (props.config.source === 'osquery') {
                result = await RunOsquery(props.config.query);
            } else {
                result = await RunWidgetQuery(props.config.query, 100);
            }
            setData(result || []);
            renderChart(result || []);
        } catch (e) {
            const err = e as Error;
            setError((err as Error).message || String(e));
        } finally {
            setLoading(false);
        }
    };

    const renderChart = (dataset: Record<string, unknown>[]) => {
        if (!chartRef) return;
        if (props.config.type === 'metric' || props.config.type === 'table' || props.config.type === 'log-stream') return;
        if (!dataset || dataset.length === 0) return;

        if (!chartInstance) {
            chartInstance = echarts.init(chartRef, 'dark');
        }

        const keys = Object.keys(dataset[0] || {});
        const xKey = keys[0];
        const yKey = keys[1];

        let option: echarts.EChartsOption;

        if (props.config.type === 'pie') {
            option = {
                backgroundColor: 'transparent',
                tooltip: { trigger: 'item' },
                series: [{
                    type: 'pie',
                    radius: ['40%', '70%'],
                    avoidLabelOverlap: false,
                    itemStyle: { borderRadius: 8, borderColor: '#1a1e2e', borderWidth: 2 },
                    label: { color: '#e1e6eb' },
                    data: dataset.map(d => ({ name: String(d[xKey]), value: Number(d[yKey]) }))
                }]
            };
        } else {
            option = {
                backgroundColor: 'transparent',
                tooltip: { trigger: 'axis' },
                grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
                xAxis: {
                    type: 'category',
                    data: dataset.map(d => String(d[xKey])),
                    axisLabel: { color: '#8b95a5' },
                    axisLine: { lineStyle: { color: '#3c444d' } }
                },
                yAxis: {
                    type: 'value',
                    axisLabel: { color: '#8b95a5' },
                    splitLine: { lineStyle: { color: '#2a2e3e' } }
                },
                series: [{
                    data: dataset.map(d => Number(d[yKey])),
                    type: props.config.type as 'line' | 'bar',
                    smooth: true,
                    areaStyle: props.config.type === 'line' ? { opacity: 0.15 } : undefined,
                    itemStyle: {
                        color: props.config.type === 'bar'
                            ? new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                                { offset: 0, color: '#00d4ff' },
                                { offset: 1, color: '#0071ce' }
                            ])
                            : '#00d4ff'
                    }
                }]
            };
        }

        chartInstance.setOption(option);
    };

    onMount(() => {
        fetchData();
        if (props.config.refreshInterval > 0) {
            timer = setInterval(fetchData, props.config.refreshInterval * 1000);
        }

        if (props.config.type === 'log-stream') {
            EventsOn("siem-stream", (evt: Record<string, unknown>) => {
                const newRow = {
                    timestamp: (evt.Timestamp || evt.timestamp || new Date().toISOString()) as string,
                    host: (evt.HostID || evt.host || 'unknown') as string,
                    output: (evt.Output || evt.output || JSON.stringify(evt)) as string
                };
                setData(prev => [newRow, ...prev].slice(0, 50));
            });
        }

        const ro = new ResizeObserver(() => chartInstance?.resize());
        if (chartRef) ro.observe(chartRef);
    });

    // Re-fetch automatically if the timeRange prop changes
    createEffect(() => {
        if (props.timeRange) fetchData();
    });

    onCleanup(() => {
        if (timer) clearInterval(timer);
        if (props.config.type === 'log-stream') {
            EventsOff("siem-stream");
        }
        chartInstance?.dispose();
    });

    return (
        <div class="widget-container">
            <div class="widget-header" style="display: flex; justify-content: space-between; align-items: center; padding: 10px 14px; border-bottom: 1px solid var(--border-primary); background: rgba(0,0,0,0.1);">
                <span class="widget-title" style="font-family: var(--font-ui); font-weight: 700; font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">{props.config.title}</span>
                <div class="widget-controls" style="display: flex; align-items: center; gap: 8px;">
                    <Show when={loading()}><span class="widget-spinner" style="font-size: 10px; opacity: 0.6;">↻</span></Show>
                    <Show when={props.config.refreshInterval > 0}>
                        <Badge severity="neutral" style="font-size: 8px; padding: 1px 4px;">{props.config.refreshInterval}S</Badge>
                    </Show>
                    <button class="widget-reload" onClick={fetchData} title="Refresh" style="background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 12px; padding: 0;">⟳</button>
                </div>
            </div>

            <Show when={error()}>
                <div class="widget-error">{error()}</div>
            </Show>

            <div class="widget-content">
                <Switch>
                    {/* Charts (line, bar, pie) */}
                    <Match when={['line', 'bar', 'pie'].includes(props.config.type)}>
                        <div ref={chartRef} style={{ width: '100%', height: '100%' }} />
                    </Match>

                    {/* Single Metric (Big Number) */}
                    <Match when={props.config.type === 'metric'}>
                        <div class="metric-display" style="height: 100%; display: flex; align-items: center; justify-content: center; padding: 20px;">
                            <span class="metric-value" style="font-size: 32px; font-weight: 800; color: var(--text-primary); font-family: var(--font-mono); letter-spacing: -1px;">
                                {data().length > 0 ? Object.values(data()[0])[0] as string : '—'}
                            </span>
                        </div>
                    </Match>

                    {/* Data Table */}
                    <Match when={props.config.type === 'table'}>
                        <div class="widget-table-scroll">
                            <table class="widget-table">
                                <thead>
                                    <tr>
                                        <Show when={data().length > 0}>
                                            <For each={Object.keys(data()[0])}>
                                                {k => <th>{k}</th>}
                                            </For>
                                        </Show>
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={data()}>
                                        {row => (
                                            <tr>
                                                <For each={Object.values(row)}>
                                                    {(v: unknown) => <td>{String(v)}</td>}
                                                </For>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </Match>

                    {/* Log Stream */}
                    <Match when={props.config.type === 'log-stream'}>
                        <div class="widget-log-stream">
                            <For each={data()}>
                                {(row: Record<string, unknown>) => (
                                    <div class="log-line">
                                        <span class="log-ts">{(row.timestamp as string) || ''}</span>
                                        <span class="log-host">{(row.host as string) || ''}</span>
                                        <span class="log-msg">{(row.output as string) || JSON.stringify(row)}</span>
                                    </div>
                                )}
                            </For>
                        </div>
                    </Match>
                </Switch>

                <Show when={data().length === 0 && !loading() && !error()}>
                    <div class="widget-empty">No data</div>
                </Show>
            </div>
        </div>
    );
};
