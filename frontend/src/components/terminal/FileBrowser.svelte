<!--
  OBLIVRA — File Browser (Svelte 5)
  SFTP interface for remote host file management.
-->
<script lang="ts">
  import { KPI, Badge, Button, DataTable } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';

  let currentPath = $state('/var/log');
  let files = $state([
    { name: 'auth.log', size: '124 KB', type: 'file', modified: '2m ago' },
    { name: 'syslog', size: '1.2 MB', type: 'file', modified: 'now' },
    { name: 'nginx/', size: '--', type: 'dir', modified: '1h ago' },
    { name: 'secure', size: '89 KB', type: 'file', modified: '1d ago' },
  ]);

  const columns = [
    { key: 'name', label: 'Object' },
    { key: 'size', label: 'Mass' },
    { key: 'modified', label: 'Last Pulse' },
    { key: 'actions', label: '' },
  ];

  function upload() {
    // Logic for SFTP upload
    appStore.notify('Preparing upload...', 'info');
  }

  function download(name: string) {
    appStore.notify(`Downloading ${name}`, 'info');
  }
</script>

<div class="flex flex-col h-full bg-surface-1 border-l border-border-primary w-80">
  <div class="p-3 border-b border-border-primary bg-surface-2 flex items-center justify-between">
    <span class="text-[10px] font-bold uppercase tracking-wider text-text-secondary">SFTP Browser</span>
    <Button variant="ghost" size="xs" onclick={upload}>↑ Upload</Button>
  </div>

  <div class="px-3 py-2 bg-surface-0 border-b border-border-primary">
    <div class="flex items-center gap-2 text-[10px] font-mono text-text-muted">
      <span class="opacity-50">PATH:</span>
      <span class="text-accent truncate">{currentPath}</span>
    </div>
  </div>

  <div class="flex-1 overflow-auto">
    <DataTable {columns} data={files} density="compact">
      {#snippet cell({ column, row })}
        {#if column.key === 'name'}
          <div class="flex items-center gap-2">
            <span class="text-[10px]">{row.type === 'dir' ? '📁' : '📄'}</span>
            <span class="text-[11px] truncate {row.type === 'dir' ? 'text-text-primary font-bold' : 'text-text-secondary'}">
              {row.name}
            </span>
          </div>
        {:else if column.key === 'actions'}
          <button 
            class="text-[10px] text-text-muted hover:text-accent transition-colors"
            onclick={() => download(row.name)}
          >
            ꜜ
          </button>
        {:else}
          <span class="text-[10px] text-text-muted">{row[column.key]}</span>
        {/if}
      {/snippet}
    </DataTable>
  </div>

  <div class="p-2 border-t border-border-primary bg-surface-2 text-[9px] text-text-muted text-center">
    4 nodes discovered • 1.4 MB total
  </div>
</div>
