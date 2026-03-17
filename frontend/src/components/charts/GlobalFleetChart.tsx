import { Component, createSignal, onMount, onCleanup } from 'solid-js';
import * as echarts from 'echarts';
import { ChartBlock } from './ChartBlock';

export const GlobalFleetChart: Component = () => {
    const [options, setOptions] = createSignal<any>({});

    const regions = ['North America', 'EMEA', 'APAC', 'LATAM', 'Sovereign Node'];
    
    const generateData = () => {
        return regions.map(() => Math.floor(Math.random() * 1000) + 200);
    };

    const updateChart = () => {
        const data = generateData();
        
        setOptions({
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'shadow' },
                backgroundColor: 'rgba(22, 27, 34, 0.9)',
                borderColor: '#30363d',
                textStyle: { color: '#c9d1d9', fontSize: 12 }
            },
            grid: {
                top: '5%',
                left: '2%',
                right: '4%',
                bottom: '5%',
                containLabel: true
            },
            xAxis: {
                type: 'value',
                splitLine: { lineStyle: { color: '#21262d', type: 'dashed' } },
                axisLabel: { color: '#8b949e', fontSize: 10 }
            },
            yAxis: {
                type: 'category',
                data: regions,
                axisLine: { lineStyle: { color: '#30363d' } },
                axisLabel: { color: '#c9d1d9', fontSize: 11 }
            },
            series: [
                {
                    name: 'Active Endpoints',
                    type: 'bar',
                    barWidth: '50%',
                    itemStyle: {
                        color: new echarts.graphic.LinearGradient(1, 0, 0, 0, [
                            { offset: 0, color: '#58a6ff' },
                            { offset: 1, color: 'rgba(88, 166, 255, 0.1)' }
                        ]),
                        borderRadius: [0, 2, 2, 0]
                    },
                    emphasis: {
                        itemStyle: {
                            color: '#79c0ff'
                        }
                    },
                    data: data,
                    animationDurationUpdate: 500
                }
            ]
        });
    };

    onMount(() => {
        updateChart();
        const interval = setInterval(updateChart, 3000);
        onCleanup(() => clearInterval(interval));
    });

    return (
        <div class="global-fleet-chart-container">
            <div style={{ padding: '8px 12px', "border-bottom": '1px solid #30363d', display: 'flex', "justify-content": 'space-between', "align-items": 'center' }}>
                <span style={{ "font-size": '10px', "text-transform": 'uppercase', "letter-spacing": '0.1em', color: '#8b949e', "font-weight": 'bold' }}>
                    Global Fleet Identity Distribution 🌐 [Web Only]
                </span>
                <span style={{ "font-size": '9px', color: '#238636', "font-family": 'monospace' }}>
                    LIVE // ENCRYPTED
                </span>
            </div>
            <ChartBlock options={options()} theme="dark" />
        </div>
    );
};
