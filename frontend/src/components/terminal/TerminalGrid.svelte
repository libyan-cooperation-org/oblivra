<!--
  OBLIVRA — Terminal Grid (Svelte 5)
  Advanced split-pane orchestration using Golden Layout.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { VirtualLayout, LayoutConfig, ComponentItemConfig } from 'golden-layout';
  import { appStore } from '@lib/stores/app.svelte';
  import XTerm from './XTerm.svelte';
  import 'golden-layout/dist/css/goldenlayout-base.css';
  import 'golden-layout/dist/css/themes/goldenlayout-dark-theme.css';

  let layoutElement = $state<HTMLDivElement>();
  let layout: VirtualLayout;

  const config: LayoutConfig = {
    root: {
      type: 'row',
      content: []
    }
  };

  onMount(() => {
    if (!layoutElement) return;

    layout = new VirtualLayout(layoutElement);
    
    // Register XTerm component for GoldenLayout
    layout.registerComponentFactoryFunction('terminal', (container, itemConfig) => {
      const sessionId = (itemConfig as any).sessionId;
      // In GoldenLayout v2, we can't easily mount Svelte components INSIDE the container 
      // without complex portals or manual mounting. 
      // For now, I'll use a simpler approach or a dedicated wrapper.
      const el = document.createElement('div');
      el.className = 'w-full h-full';
      container.element.appendChild(el);
      
      // We'll need a way to mount XTerm here.
      // Svelte 5 mount() can be used.
    });

    layout.loadLayout(config);

    window.addEventListener('resize', handleResize);
  });

  function handleResize() {
    layout?.updateSize();
  }

  onDestroy(() => {
    window.removeEventListener('resize', handleResize);
    layout?.destroy();
  });
</script>

<div bind:this={layoutElement} class="w-full h-full"></div>

<style>
  :global(.lm_header) {
    background: #111 !important;
    height: 24px !important;
  }
  :global(.lm_tab) {
    background: #1a1a1a !important;
    border: none !important;
    color: #666 !important;
    font-family: ui-monospace, monospace !important;
    font-size: 10px !important;
    text-transform: uppercase !important;
    padding: 2px 10px !important;
  }
  :global(.lm_tab.lm_active) {
    background: #222 !important;
    color: #fff !important;
    border-top: 1px solid #ff0000 !important;
  }
</style>
