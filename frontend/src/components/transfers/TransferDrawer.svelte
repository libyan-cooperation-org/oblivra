<!--
  OBLIVRA — Transfer Drawer (Svelte 5)
  Real-time file movement tracking and orchestration.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { Shield, Check, X, ArrowUpRight, ArrowDownLeft } from 'lucide-svelte';
  import { Button } from '@components/ui';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  let { open, onClose }: Props = $props();

  function clearCompleted() {
    // Logic to clear completed transfers from store
    appStore.transfers = appStore.transfers.filter(t => t.status === 'active' || t.status === 'pending');
  }
</script>

{#if open}
  <!-- Backdrop -->
  <div 
    class="fixed inset-0 z-[100] bg-black/40 backdrop-blur-[2px]"
    onclick={onClose}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
    role="button"
    tabindex="-1"
  ></div>

  <!-- Content -->
  <aside 
    class="fixed right-0 top-6 bottom-6 w-96 z-[101] bg-surface-1 border-l border-border-primary shadow-2xl flex flex-col transition-transform duration-300"
    style="transform: translateX(0)"
  >
    <div class="px-4 py-3 border-b border-border-primary flex items-center justify-between bg-surface-2">
      <div class="flex items-center gap-2">
        <span class="text-[11px] font-bold tracking-widest uppercase text-text-muted opacity-60">Operations</span>
        <span class="text-xs font-bold text-text-primary">Data Movement</span>
      </div>
      <button 
        class="text-text-muted hover:text-text-primary transition-colors cursor-pointer"
        onclick={onClose}
      >✕</button>
    </div>

    <div class="flex-1 overflow-y-auto p-4 space-y-4">
      {#if appStore.transfers.length === 0}
        <div class="h-full flex flex-col items-center justify-center text-center opacity-30 space-y-3">
          <div class="w-12 h-12 rounded-full border border-dashed border-text-muted flex items-center justify-center text-2xl">⇅</div>
          <div class="text-xs font-medium">No active data payloads</div>
        </div>
      {:else}
        {#each appStore.transfers as transfer (transfer.id)}
          <div class="bg-surface-0 border border-border-secondary p-3 rounded-sm space-y-2 group">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 overflow-hidden">
                <span class={transfer.type === 'upload' ? 'text-accent' : 'text-success'}>
                  {#if transfer.type === 'upload'}
                    ↑
                  {:else}
                    ↓
                  {/if}
                </span>
                <span class="text-xs font-bold truncate text-text-secondary">{transfer.name}</span>
              </div>
              <Badge variant={transfer.status === 'completed' ? 'success' : transfer.status === 'failed' ? 'error' : 'info'}>
                {transfer.status}
              </Badge>
            </div>

            {#if transfer.status === 'active' || transfer.status === 'pending'}
              <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                <div 
                  class="bg-accent h-full transition-all duration-300" 
                  style="width: {transfer.progress || 0}%"
                ></div>
              </div>
              <div class="flex items-center justify-between text-[10px] text-text-muted font-mono">
                <span>{transfer.progress || 0}%</span>
                <span>{transfer.speed || '0 KB/s'}</span>
              </div>
            {:else if transfer.status === 'completed'}
              <div class="text-[10px] text-success font-mono">
                Payload secured in {transfer.duration || '0.2s'}
              </div>
            {:else if transfer.status === 'failed'}
              <div class="text-[10px] text-error font-medium truncate">
                {transfer.error || 'Endpoint refused connection'}
              </div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>

    {#if appStore.transfers.length > 0}
      <div class="p-3 border-t border-border-primary bg-surface-2 flex gap-2">
        <Button variant="secondary" size="sm" class="flex-1" onclick={clearCompleted}>Purge History</Button>
        <Button variant="ghost" size="sm" onclick={onClose}>Dismiss</Button>
      </div>
    {/if}
  </aside>
{/if}

<style>
  aside {
    box-shadow: -10px 0 30px rgba(0, 0, 0, 0.5);
  }
</style>
