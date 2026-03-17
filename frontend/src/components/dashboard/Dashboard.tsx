import { Component, createSignal, onMount, For, Show, onCleanup } from 'solid-js';
import * as echarts from 'echarts';
import { GetGlobalThreatStats, GetEventTrend } from '../../../wailsjs/go/services/SIEMService';
import { GetFleetTelemetry } from '../../../wailsjs/go/services/TelemetryService';
import { useApp } from '@core/store';
import { Sparkline } from '../ui/Sparkline';
import { InceptionView } from './InceptionView';
import '../../styles/dashboard.css';

export const Dashboard: Component = () => {
    const [state] = useApp();
    const [stats, setStats] = createSignal<any>({});
    const [fleetHealth, setFleetHealth] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    let chartRef!: HTMLDivElement;

    const isInception = () => state.hosts.length === 0;

    const loadData = async () => {
        setLoading(true);
        try {
            const [s, t, f] = await Promise.all([
                GetGlobalThreatStats(),
                GetEventTrend(7),
                GetFleetTelemetry()
            ]);
            setStats(s || {});
            setFleetHealth(f || []);
            renderTrendChart(t || []);
        } catch (err) {
            console.error("Dashboard data load failed:", err);
        } finally {
            setLoading(false);
        }
    };

    const renderTrendChart = (data: any[]) => {
        if (!chartRef) return;
        const myChart = echarts.init(chartRef);
        myChart.setOption({
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'axis',
                backgroundColor: '#2b2d31',
                borderColor: '#4a4d55',
                textStyle: { color: '#d4d5d8', fontSize: 11 },
                formatter: (params: any[]) => {
                    const p = params[0];
                    return `<div style="font-family:monospace">${p.name}<br/><span style="color:#0099e0">●</span> Events: <b>${p.value}</b></div>`;
                }
            },
            grid: { left: '3%', right: '3%', bottom: '3%', top: '8%', containLabel: true },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: data.map(d => d.date),
                axisLabel: { color: '#6b6e76', fontSize: 10, fontFamily: 'monospace' },
                axisLine: { lineStyle: { color: '#3a3d44' } },
                splitLine: { show: false }
            },
            yAxis: {
                type: 'value',
                splitLine: { lineStyle: { color: '#2b2d31', type: 'dashed' } },
                axisLabel: { color: '#6b6e76', fontSize: 10, fontFamily: 'monospace' }
            },
            series: [{
                name: 'Security Events',
                type: 'line',
                smooth: false,
                data: data.map(d => d.count),
                itemStyle: { color: '#0099e0' },
                lineStyle: { width: 1.5, color: '#0099e0' },
                symbol: 'circle',
                symbolSize: 4,
                areaStyle: {
                    color: new (echarts as any).graphic.LinearGradient(0, 0, 0, 1, [
                        { offset: 0, color: 'rgba(0,153,224,0.15)' },
                        { offset: 1, color: 'rgba(0,153,224,0)' }
                    ])
                }
            }]
        });

        const resizeHandler = () => myChart.resize();
        window.addEventListener('resize', resizeHandler);
        onCleanup(() => window.removeEventListener('resize', resizeHandler));
    };

    onMount(() => loadData());

    const kpis = () => [
        { label: 'Total Threats', value: stats().total_failed_logins || 0, trend: '+12.4%', trendBad: true, sub: 'Weekly delta' },
        { label: 'Attacker IPs', value: stats().unique_attacker_ips || 0, sub: 'Unique source vectors' },
        { label: 'Critical Nodes', value: stats().high_risk_hosts || 0, alert: (stats().high_risk_hosts || 0) > 0, sub: 'Anomaly threshold breach' },
        { label: 'Fleet Size', value: state.hosts.length, sub: 'Active agents' },
    ];

    // Deterministic sparkline — seeded by value so it's stable across re-renders.
    // Uses a simple LCG so the shape reflects the magnitude without random thrash.
    const generateSparklineData = (baseValue: number, volatility: number = 0.2) => {
        const data: number[] = [];
        let seed = (baseValue || 10) * 2654435761;
        const rand = () => {
            seed = (seed * 1664525 + 1013904223) & 0xffffffff;
            return ((seed >>> 16) & 0xffff) / 0xffff;
        };
        let current = baseValue || 10;
        for (let i = 0; i < 20; i++) {
            data.push(current);
            current = Math.max(0, current + (rand() - 0.5) * current * volatility);
        }
        return data;
    };

    return (
        <Show when={!isInception()} fallback={<InceptionView />}>
            <div class="ob-page page-enter">
                {/* Header */}
                <div class="dash-header">
                    <div>
                        <h1 class="dash-title">Fleet Intelligence</h1>
                        <p class="dash-subtitle">Global security orchestration & proactive threat telemetry</p>
                    </div>
                    <button class="ob-btn ob-btn-sm ob-btn-ghost" onClick={loadData}>↻ Refresh</button>
                </div>

                {/* KPI Cards */}
                <Show when={!loading()} fallback={
                    <div class="dash-kpi-grid">
                        <For each={[1, 2, 3, 4]}>{() =>
                            <div class="ob-card dash-kpi">
                                <div class="ob-skeleton" style="width:80px;height:10px;margin-bottom:12px" />
                                <div class="ob-skeleton" style="width:60px;height:28px;margin-bottom:8px" />
                                <div class="ob-skeleton" style="width:100px;height:8px" />
                            </div>
                        }</For>
                    </div>
                }>
                    <div class="dash-kpi-grid">
                        <For each={kpis()}>
                            {(kpi) => (
                                <div class={`ob-card dash-kpi ${kpi.alert ? 'dash-kpi-alert' : ''}`}>
                                    <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                        <div>
                                            <div class="dash-kpi-label">{kpi.label}</div>
                                            <div class="dash-kpi-value">{kpi.value}</div>
                                        </div>
                                        {kpi.trend && <span class={kpi.trendBad ? 'dash-trend-bad' : 'dash-trend-good'}>{kpi.trend}</span>}
                                    </div>
                                    <Sparkline data={generateSparklineData(kpi.value as number)} color={kpi.alert ? '#e04040' : kpi.trendBad ? '#f58b00' : '#0099e0'} />
                                    <div class="dash-kpi-sub" style="margin-top: 8px;">
                                        {kpi.sub}
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                {/* Main Grid — Chart + Heatmap */}
                <div class="dash-main-grid">
                    <div class="ob-card dash-chart-card">
                        <div class="dash-section-header">Anomaly Trend (7D)</div>
                        <Show when={!loading()} fallback={
                            <div class="ob-skeleton" style="width:100%;height:280px;border-radius:4px" />
                        }>
                            <div ref={chartRef} class="dash-chart" />
                        </Show>
                    </div>

                    <div class="ob-card dash-heatmap-card">
                        <div class="dash-section-header">Fleet Health</div>
                        <Show when={!loading()} fallback={
                            <div class="ob-skeleton" style="width:100%;height:200px;border-radius:4px" />
                        }>
                            <div class="dash-heatmap-grid">
                                <For each={state.hosts}>
                                    {(host) => {
                                        const telemetry = fleetHealth().find(t => t.host_id === host.id);
                                        const load = telemetry && telemetry.mem_total_mb > 0
                                            ? (telemetry.cpu_usage + (telemetry.mem_used_mb / telemetry.mem_total_mb * 100)) / 2 : 0;
                                        const color = load > 80 ? 'var(--status-offline)' : load > 50 ? 'var(--status-degraded)' : 'var(--status-online)';
                                        return (
                                            <div
                                                class="dash-heatmap-cell"
                                                style={`background:${color};opacity:${Math.max(0.2, load / 100)}`}
                                                title={`${host.hostname}: ${load.toFixed(1)}%`}
                                            />
                                        );
                                    }}
                                </For>
                                <Show when={state.hosts.length === 0}>
                                    <div class="ob-empty-sm">No hosts detected</div>
                                </Show>
                            </div>
                            <div class="dash-legend">
                                <span class="dash-legend-item"><span class="dash-dot" style="background:var(--status-online)" /> Healthy</span>
                                <span class="dash-legend-item"><span class="dash-dot" style="background:var(--status-degraded)" /> Busy</span>
                                <span class="dash-legend-item"><span class="dash-dot" style="background:var(--status-offline)" /> Overload</span>
                            </div>
                        </Show>
                    </div>
                </div>
            </div>
        </Show>
    );
};
