import { Component, createSignal, createEffect, onMount } from 'solid-js';
import * as echarts from 'echarts';
import { GetFailedLoginsByHost, GetRiskScoreByHost } from '../../../wailsjs/go/app/SIEMService';

export const ThreatMap: Component<{ hostId: string }> = (props) => {
    let chartRef!: HTMLDivElement;
    let gaugeRef!: HTMLDivElement;
    const [stats, setStats] = createSignal<any[]>([]);
    const [riskScore, setRiskScore] = createSignal(0);
    const [loading, setLoading] = createSignal(true);

    const loadData = async () => {
        setLoading(true);
        try {
            const [data, score] = await Promise.all([
                GetFailedLoginsByHost(props.hostId),
                GetRiskScoreByHost(props.hostId)
            ]);
            setStats(data || []);
            setRiskScore(score || 0);
            renderCharts(data || [], score || 0);
        } catch (err) {
            console.error("Failed to load generic threat data", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        if (props.hostId) {
            loadData();
        }
    });

    createEffect(() => {
        if (props.hostId) {
            loadData();
        }
    });

    const renderCharts = (data: any[], score: number) => {
        renderBarChart(data);
        renderGaugeChart(score);
    };

    const renderGaugeChart = (score: number) => {
        if (!gaugeRef) return;
        const myChart = echarts.init(gaugeRef);

        const option = {
            backgroundColor: 'transparent',
            series: [
                {
                    type: 'gauge',
                    startAngle: 180,
                    endAngle: 0,
                    center: ['50%', '85%'],
                    radius: '100%',
                    min: 0,
                    max: 100,
                    splitNumber: 4,
                    axisLine: {
                        lineStyle: {
                            width: 2,
                            color: [
                                [1, 'var(--glass-border)']
                            ]
                        }
                    },
                    progress: {
                        show: true,
                        width: 8,
                        itemStyle: {
                            color: score > 70 ? 'var(--status-offline)' : (score > 40 ? 'var(--status-degraded)' : 'var(--status-online)')
                        }
                    },
                    pointer: { show: false },
                    axisTick: { show: false },
                    splitLine: { show: false },
                    axisLabel: { show: true, distance: -40, color: 'var(--text-muted)', fontSize: 10, fontFamily: 'var(--font-mono)' },
                    title: { show: false },
                    detail: {
                        fontSize: 32,
                        offsetCenter: [0, '-10%'],
                        valueAnimation: true,
                        formatter: '{value}',
                        color: 'var(--text-primary)',
                        fontFamily: 'var(--font-mono)'
                    },
                    data: [{ value: score }]
                }
            ]
        };

        myChart.setOption(option);
    };

    const renderBarChart = (data: any[]) => {
        if (!chartRef) return;
        const myChart = echarts.init(chartRef);

        const ips = data.map(d => d.source_ip);
        const attempts = data.map(d => d.attempts);

        const option = {
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'shadow' }
            },
            grid: {
                left: '0',
                right: '40',
                bottom: '0',
                top: '0',
                containLabel: true
            },
            xAxis: {
                type: 'value',
                splitLine: { lineStyle: { color: 'var(--glass-border)' } },
                axisLabel: { color: 'var(--text-muted)', fontSize: 10, fontFamily: 'var(--font-mono)' }
            },
            yAxis: {
                type: 'category',
                data: ips,
                axisLabel: { color: 'var(--text-primary)', fontSize: 11, fontFamily: 'var(--font-mono)' },
                axisLine: { lineStyle: { color: 'var(--glass-border)' } }
            },
            series: [
                {
                    name: 'ATTEMPTS',
                    type: 'bar',
                    data: attempts,
                    itemStyle: {
                        color: 'var(--status-offline)',
                    },
                    barWidth: '50%',
                    label: {
                        show: true,
                        position: 'right',
                        color: 'var(--text-primary)',
                        fontFamily: 'var(--font-mono)',
                        fontSize: 11
                    }
                }
            ]
        };

        myChart.setOption(option);

        window.addEventListener('resize', () => {
            myChart.resize();
        });
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%; overflow: hidden; padding: 0; gap: 24px;">
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                <div>
                    <h2 style="font-size: 14px; font-weight: 800; color: var(--status-offline); margin: 0; font-family: var(--font-ui); text-transform: uppercase; letter-spacing: 1px;">
                        SECURITY_AUDIT_INTELLIGENCE: {props.hostId}
                    </h2>
                    <p style="font-size: 11px; color: var(--text-muted); margin: 4px 0 0 0; font-family: var(--font-mono); text-transform: uppercase;">
                        HEURISTIC_THREAT_ANALYSIS_ENGINE // STATUS: ACTIVE
                    </p>
                </div>
                <button class="ob-btn ob-btn-secondary ob-btn-sm" onClick={loadData} disabled={loading()}>
                    ↻ REFRESH_BUFFER
                </button>
            </div>

            <div style="display: grid; grid-template-columns: 1fr 2fr; gap: 16px; flex: 1; min-height: 0;">
                <div class="ob-card" style="display: flex; flex-direction: column; padding: 20px;">
                    <div style={`font-size: 10px; font-weight: 800; font-family: var(--font-mono); color: ${riskScore() > 70 ? 'var(--status-offline)' : 'var(--text-muted)'}; margin-bottom: 16px;`}>
                        CALCULATED_ANOMALY_SCORE
                    </div>
                    {loading() ? (
                        <div style="flex:1; display:flex; align-items:center; justify-content:center;">
                            <div class="ob-skeleton" style="width:160px; height:160px; border-radius:50%;" />
                        </div>
                    ) : (
                        <div ref={gaugeRef} style="flex: 1; width: 100%; min-height: 200px;"></div>
                    )}
                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); padding: 8px 0 0 0; text-align: center; border-top: 1px solid var(--border-primary); margin-top: 16px;">
                        VECTORS: ATTEMPT_DENSITY + ROOT_TARGETING
                    </div>
                </div>

                <div class="ob-card" style="display: flex; flex-direction: column; padding: 20px;">
                    <div style="font-size: 10px; font-weight: 800; font-family: var(--font-mono); color: var(--text-muted); margin-bottom: 16px;">
                        FAILED_LOGINS_BY_SOURCE_IP
                    </div>
                    <div style="flex: 1; min-height: 0;">
                        {loading() ? (
                            <div style="display: flex; flex-direction: column; gap: 12px; height: 100%; justify-content: center;">
                                <div class="ob-skeleton" style="width: 100%; height: 24px;" />
                                <div class="ob-skeleton" style="width: 80%; height: 24px;" />
                                <div class="ob-skeleton" style="width: 90%; height: 24px;" />
                                <div class="ob-skeleton" style="width: 60%; height: 24px;" />
                            </div>
                        ) : stats().length === 0 ? (
                            <div style="height: 100%; display: flex; align-items: center; justify-content: center; font-size: 11px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">
                                NO_THREATS_DETECTED_IN_AUTH_BUFFER
                            </div>
                        ) : (
                            <div ref={chartRef} style="width: 100%; height: 100%;"></div>
                        )}
                    </div>
                </div>
            </div>

            {(!loading() && stats().length > 0) && (
                <div class="ob-card" style="max-height: 240px; overflow-y: auto; padding: 0;">
                    <table class="ob-table-zebra">
                        <thead>
                            <tr>
                                <th>SOURCE_IP</th>
                                <th>TARGET_USER</th>
                                <th>ATTEMPTS</th>
                                <th>LAST_VECTOR_SEEN</th>
                            </tr>
                        </thead>
                        <tbody>
                            {stats().map(s => (
                                <tr>
                                    <td class="font-mono" style="color: var(--status-offline);">{s.source_ip}</td>
                                    <td>{s.user}</td>
                                    <td class="font-mono" style="font-weight: 800;">{s.attempts}</td>
                                    <td class="font-mono" style="color: var(--text-muted);">{new Date(s.last_attempt).toISOString().replace('T', ' ').slice(0, 19)}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
};
