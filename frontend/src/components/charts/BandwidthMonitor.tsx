import { Component, createSignal, onMount, onCleanup } from 'solid-js';
import * as echarts from 'echarts';
import { ChartBlock } from './ChartBlock';
import { GetActiveSessions } from '../../../wailsjs/go/services/SSHService';

export const BandwidthMonitor: Component = () => {
    // Array of { time: string, in: number, out: number }
    const [data, setData] = createSignal<any[]>([]);

    // Store previous totals to calculate delta per second
    let lastBytesIn = 0;
    let lastBytesOut = 0;
    let isFirstTick = true;

    const tick = async () => {
        try {
            const sessions = await GetActiveSessions();
            let totalIn = 0;
            let totalOut = 0;

            sessions.forEach((s: any) => {
                totalIn += (s.bytesIn || 0);
                totalOut += (s.bytesOut || 0);
            });

            const now = new Date();
            const timeStr = now.toLocaleTimeString([], { hour12: false, second: '2-digit' });

            if (isFirstTick) {
                lastBytesIn = totalIn;
                lastBytesOut = totalOut;
                isFirstTick = false;
                // Add an initial 0 point
                setData([{ time: timeStr, in: 0, out: 0 }]);
                return;
            }

            const deltaIn = Math.max(0, totalIn - lastBytesIn);
            const deltaOut = Math.max(0, totalOut - lastBytesOut);

            // Convert raw bytes to KB/s for display
            const inKB = parseFloat((deltaIn / 1024).toFixed(1));
            const outKB = parseFloat((deltaOut / 1024).toFixed(1));

            lastBytesIn = totalIn;
            lastBytesOut = totalOut;

            setData(prev => {
                const next = [...prev, { time: timeStr, in: inKB, out: outKB }];
                // Keep the last 60 seconds of data (rolling window to prevent memory leaks)
                if (next.length > 60) next.shift();
                return next;
            });

        } catch (err) {
            console.error("Bandwidth monitor tick error:", err);
        }
    };

    onMount(() => {
        tick();
        const interval = setInterval(tick, 1000);
        onCleanup(() => clearInterval(interval));
    });

    const chartOptions = () => {
        const currentData = data();
        const times = currentData.map((d: any) => d.time);
        const inValues = currentData.map((d: any) => d.in);
        const outValues = currentData.map((d: any) => d.out);

        return {
            tooltip: {
                trigger: 'axis',
                backgroundColor: 'rgba(22, 27, 34, 0.9)',
                borderColor: '#30363d',
                textStyle: { color: '#c9d1d9' }
            },
            legend: {
                data: ['Inbound (KB/s)', 'Outbound (KB/s)'],
                textStyle: { color: '#8b949e' },
                top: 0
            },
            grid: {
                left: '2%',
                right: '4%',
                bottom: '3%',
                containLabel: true
            },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: times,
                axisLine: { lineStyle: { color: '#30363d' } },
                axisLabel: { color: '#8b949e' }
            },
            yAxis: {
                type: 'value',
                axisLine: { lineStyle: { color: '#30363d' } },
                splitLine: { lineStyle: { color: '#21262d', type: 'dashed' } },
                axisLabel: { color: '#8b949e' }
            },
            series: [
                {
                    name: 'Inbound (KB/s)',
                    type: 'line',
                    smooth: true,
                    showSymbol: false,
                    areaStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: 'rgba(46, 160, 67, 0.4)' },
                            { offset: 1, color: 'rgba(46, 160, 67, 0.05)' }
                        ])
                    },
                    itemStyle: { color: '#2ea043' },
                    data: inValues
                },
                {
                    name: 'Outbound (KB/s)',
                    type: 'line',
                    smooth: true,
                    showSymbol: false,
                    areaStyle: {
                        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                            { offset: 0, color: 'rgba(88, 166, 255, 0.4)' },
                            { offset: 1, color: 'rgba(88, 166, 255, 0.05)' }
                        ])
                    },
                    itemStyle: { color: '#58a6ff' },
                    data: outValues
                }
            ]
        };
    };

    return <ChartBlock options={chartOptions()} theme="dark" />;
};
