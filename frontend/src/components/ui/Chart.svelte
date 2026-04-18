<!--
  OBLIVRA — ECharts Wrapper (Svelte 5)
  Reactive chart component with automatic resizing.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as echarts from 'echarts';

  interface Props {
    option: echarts.EChartsOption;
    class?: string;
    loading?: boolean;
    onReady?: (instance: echarts.ECharts) => void;
  }

  let { option, class: className = '', loading = false, onReady }: Props = $props();

  let chartElement = $state<HTMLDivElement>();
  let chartInstance: echarts.ECharts | null = null;

  onMount(() => {
    if (!chartElement) return;

    chartInstance = echarts.init(chartElement, 'dark', {
      renderer: 'canvas'
    });

    if (onReady) onReady(chartInstance);

    window.addEventListener('resize', handleResize);
    
    // Initial draw
    chartInstance.setOption(option);
  });

  function handleResize() {
    chartInstance?.resize();
  }

  // Reactive update when option changes
  $effect(() => {
    if (chartInstance && option) {
      chartInstance.setOption(option, true);
    }
  });

  // Handle loading state
  $effect(() => {
    if (chartInstance) {
      if (loading) chartInstance.showLoading('default', {
        maskColor: 'rgba(0, 0, 0, 0.2)',
        textColor: '#a9b1d6',
        color: '#7aa2f7'
      });
      else chartInstance.hideLoading();
    }
  });

  onDestroy(() => {
    window.removeEventListener('resize', handleResize);
    chartInstance?.dispose();
  });
</script>

<div bind:this={chartElement} class="w-full h-full min-h-[100px] {className}"></div>
