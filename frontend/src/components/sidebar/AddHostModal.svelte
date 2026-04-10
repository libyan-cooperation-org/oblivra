<!--
  OBLIVRA — AddHostModal (Svelte 5)
  Modal for adding new SSH hosts to the infrastructure.
-->
<script lang="ts">
  import { Modal, Input, Button } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  let { open, onClose }: Props = $props();

  let label = $state('');
  let address = $state('');
  let username = $state('root');
  let port = $state(22);
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function handleAdd() {
    if (!label || !address) {
      error = 'Label and Address are required';
      return;
    }

    loading = true;
    error = null;

    try {
      // Logic would call Wails backend here
      // For now, we simulate a delay and add to local store if possible
      // (In real app, this would refresh from backend)
      await new Promise(r => setTimeout(r, 800));
      
      onClose();
      appStore.notify('Host added successfully', 'success');
    } catch (err: any) {
      error = err.message || 'Failed to add host';
    } finally {
      loading = false;
    }
  }
</script>

<Modal {open} {onClose} title="Add New Host" size="md">
  <div class="flex flex-col gap-4">
    <div class="grid grid-cols-2 gap-4">
      <Input
        label="Label"
        placeholder="e.g. Production Web-01"
        bind:value={label}
        autofocus
      />
      <Input
        label="Address"
        placeholder="10.0.0.1 or domain.com"
        bind:value={address}
      />
    </div>

    <div class="grid grid-cols-2 gap-4">
      <Input
        label="Username"
        placeholder="root"
        bind:value={username}
      />
      <Input
        label="Port"
        type="number"
        placeholder="22"
        bind:value={port}
      />
    </div>

    {#if error}
      <div class="text-[10px] text-error font-bold uppercase bg-error/10 p-2 rounded border border-error/20">
        Error: {error}
      </div>
    {/if}

    <div class="bg-surface-1 p-3 rounded-sm border border-border-primary text-[10px] text-text-muted leading-relaxed font-mono">
      <span class="text-accent font-bold">PRO-TIP:</span> You can also import hosts from your ~/.ssh/config or cloud providers in the
      <span class="text-text-secondary">Fleet Management</span> tab.
    </div>
  </div>

  {#snippet footer()}
    <Button variant="ghost" onclick={onClose}>Cancel</Button>
    <Button variant="primary" onclick={handleAdd} {loading}>
      Create Host
    </Button>
  {/snippet}
</Modal>
