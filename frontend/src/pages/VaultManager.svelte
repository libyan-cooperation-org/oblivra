<!--
  OBLIVRA — Vault Manager (Svelte 5)
  Secure credential storage and key management.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { DataTable, PageLayout, Button, EmptyState, Tabs, Input } from '@components/ui';

  const vaultTabs = [
    { id: 'keys', label: 'SSH Keys', icon: '🔑' },
    { id: 'passwords', label: 'Passwords', icon: '🔒' },
    { id: 'tokens', label: 'Cloud Tokens', icon: '☁️' },
  ];

  let activeTab = $state('keys');
  let unlockPassword = $state('');
  let unlocking = $state(false);

  // Mock data
  const mockKeys = [
    { id: 'k1', name: 'Work RSA', fingerprint: 'SHA256:abc...123', type: 'id_rsa', created: '2025-12-01' },
    { id: 'k2', name: 'Cloud Provider', fingerprint: 'SHA256:xyz...789', type: 'id_ed25519', created: '2026-01-15' },
  ];

  async function handleUnlock() {
    if (!unlockPassword.trim()) return;
    unlocking = true;
    try {
      const { guardedUnlock } = await import('@lib/bridge');
      await guardedUnlock(unlockPassword, false);
      // vault:unlocked event will flip appStore.vaultUnlocked via the bridge subscriber
      unlockPassword = '';
    } catch (e: any) {
      appStore.notify('Unlock failed', 'error', e?.message ?? String(e));
    } finally {
      unlocking = false;
    }
  }

  const columns = [
    { key: 'name', label: 'Key Name', sortable: true },
    { key: 'type', label: 'Type', width: '120px' },
    { key: 'fingerprint', label: 'Fingerprint' },
    { key: 'created', label: 'Added On', width: '120px' },
  ];
</script>

<PageLayout title="Credential Vault" subtitle="Encrypted storage for infrastructure access">
  {#snippet toolbar()}
    {#if appStore.vaultUnlocked}
      <Button variant="danger" size="sm" onclick={() => appStore.vaultUnlocked = false}>Lock Vault</Button>
    {/if}
  {/snippet}

  {#if !appStore.vaultUnlocked}
    <div class="flex flex-col items-center justify-center h-full max-w-md mx-auto text-center gap-6">
      <div class="text-4xl">🔐</div>
      <div>
        <h2 class="text-lg font-bold text-text-heading">Vault is Locked</h2>
        <p class="text-xs text-text-muted mt-1">Enter your master passphrase to access secure credentials.</p>
      </div>
      <div class="w-full flex flex-col gap-2">
        <Input 
          type="password" 
          placeholder="Master Passphrase" 
          bind:value={unlockPassword} 
          onkeydown={(e) => e.key === 'Enter' && handleUnlock()}
        />
        <Button variant="primary" onclick={handleUnlock} loading={unlocking} class="w-full">
          Unlock Vault
        </Button>
      </div>
      <div class="text-[10px] text-text-muted italic">
        Credentials are encrypted locally with AES-GCM-256 and never leave your machine in plain text.
      </div>
    </div>
  {:else}
    <div class="flex flex-col h-full gap-5">
      <Tabs tabs={vaultTabs} bind:active={activeTab} />

      {#if activeTab === 'keys'}
        <div class="flex flex-col gap-4">
          <div class="flex justify-between items-center">
            <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Managed SSH Keys</h3>
            <Button variant="secondary" size="sm" icon="+">Add Key</Button>
          </div>
          <DataTable data={mockKeys} {columns}>
            {#snippet render({ value, col })}
              {#if col.key === 'type'}
                <code class="text-[10px] bg-surface-2 px-1.5 py-0.5 rounded border border-border-primary text-accent">{value}</code>
              {:else if col.key === 'fingerprint'}
                <span class="text-[10px] font-mono opacity-50">{value}</span>
              {:else}
                {value}
              {/if}
            {/snippet}
          </DataTable>
        </div>
      {:else}
        <EmptyState title="Feature coming soon" description="Password and token management are currently under migration." icon="🛠" />
      {/if}
    </div>
  {/if}
</PageLayout>
