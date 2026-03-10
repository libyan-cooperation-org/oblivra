import { Component, onMount, onCleanup, createEffect } from 'solid-js';
import * as echarts from 'echarts';
import '../../styles/fleet.css';

interface DashboardCardProps {
    hostId: string;
    hostLabel: string;
    telemetry: {
        cpu_usage: number;
        mem_used_mb: number;
        mem_total_mb: number;
        disk_used_gb: number;
        disk_total_gb: number;
        load_avg: number;
    };
}

export const DashboardCard: Component<DashboardCardProps> = (props) => {
    let chartDom: HTMLDivElement | undefined;
    let chart: echarts.ECharts | undefined;
    const history: number[] = [];
    const maxHistory = 30;

    onMount(() => {
        if (chartDom) {
            chart = echarts.init(chartDom);
            chart.setOption({
                grid: { top: 0, bottom: 0, left: 0, right: 0 },
                xAxis: { type: 'category', show: false },
                yAxis: { type: 'value', min: 0, max: 100, show: false },
                series: [{
                    type: 'line',
                    smooth: false,
                    symbol: 'none',
                    lineStyle: { color: 'var(--status-online)', width: 1 },
                    areaStyle: {
                        color: 'rgba(16, 185, 129, 0.1)'
                    },
                    data: []
                }]
            });
        }
    });

    onCleanup(() => {
        chart?.dispose();
    });

    createEffect(() => {
        if (chart && props.telemetry) {
            history.push(props.telemetry.cpu_usage);
            if (history.length > maxHistory) history.shift();

            const isHigh = props.telemetry.cpu_usage > 70;
            chart.setOption({
                series: [{
                    data: history,
                    lineStyle: { color: isHigh ? 'var(--status-offline)' : 'var(--status-online)' },
                    areaStyle: { color: isHigh ? 'rgba(239, 68, 68, 0.1)' : 'rgba(16, 185, 129, 0.1)' }
                }]
            });
        }
    });

    const isCritical = () => props.telemetry.cpu_usage > 90 || (props.telemetry.mem_used_mb / props.telemetry.mem_total_mb) > 0.9;
    const isWarning = () => props.telemetry.cpu_usage > 70 || (props.telemetry.mem_used_mb / props.telemetry.mem_total_mb) > 0.7;

    const statusColor = () => {
        if (isCritical()) return 'var(--status-offline)';
        if (isWarning()) return 'var(--status-degraded)';
        return 'var(--status-online)';
    };

    const memPercent = () => (props.telemetry.mem_used_mb / props.telemetry.mem_total_mb) * 100;
    const diskPercent = () => (props.telemetry.disk_used_gb / props.telemetry.disk_total_gb) * 100;

    const MetricBox = (p: { label: string, value: string, percent?: number, color?: string }) => (
        <div style="display: flex; flex-direction: column; gap: 4px;">
            <div style="font-size: 9px; font-weight: 800; color: var(--text-muted); font-family: var(--font-ui); letter-spacing: 0.5px; text-transform: uppercase;">{p.label}</div>
            <div style="font-size: 14px; font-weight: 700; color: var(--text-primary); font-family: var(--font-mono);">{p.value}</div>
            {p.percent !== undefined && (
                <div style="width: 100%; height: 2px; background: rgba(255,255,255,0.1); margin-top: 4px;">
                    <div style={`height: 100%; width: ${p.percent}%; background: ${p.color || 'var(--accent-primary)'};`} />
                </div>
            )}
        </div>
    );

    return (
        <div class="ob-card" style="padding: 20px; display: flex; flex-direction: column; gap: 16px;">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                <div>
                    <div style="font-size: 14px; font-weight: 800; color: var(--text-primary); font-family: var(--font-ui);">{props.hostLabel}</div>
                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); margin-top: 2px;">NODE_ID: {props.hostId.slice(0, 8)}</div>
                </div>
                <div style={`width: 8px; height: 8px; background: ${statusColor()}; box-shadow: 0 0 8px ${statusColor()};`} title="OPERATIONAL_STATUS" />
            </div>

            <div ref={chartDom} style="width: 100%; height: 40px; margin: 8px 0;" />

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                <MetricBox label="CPU_UTILIZATION" value={`${props.telemetry.cpu_usage.toFixed(1)}%`} percent={props.telemetry.cpu_usage} color={statusColor()} />
                <MetricBox label="LOAD_AVG_1M" value={props.telemetry.load_avg.toFixed(2)} />
                <MetricBox label="MEMORY_COMMIT" value={`${Math.round(props.telemetry.mem_used_mb)}MB`} percent={memPercent()} />
                <MetricBox label="STORAGE_UTIL" value={`${props.telemetry.disk_used_gb.toFixed(1)}GB`} percent={diskPercent()} color="var(--text-muted)" />
            </div>
        </div>
    );
};
