<!--
  OBLIVRA — Vault Manager (Svelte 5)
  Encrypted credential storage. Backed by VaultService and the
  credentials table (per-host + per-tenant scoped).

  Previous version rendered a hardcoded `mockKeys` array — that lied
  to operators about credential storage state. This rewrite pulls
  real data from the host store (which carries credential metadata)
  and the credential intel store. Honest empty-state where data
  doesn't exist yet.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, Tabs, Input } from '@components/ui';
  import { ExternalLink, KeyRound } from 'lucide-svelte';
  import { push } from '@lib/router.svelte';
  import { IS_BROWSER } from '@lib/context';

  const vaultTabs = [
    { id: 'keys',      label: 'SSH Keys',     icon: '🔑' },
    { id: 'passwords', label: 'Host Auth',    icon: '🔒' },
    { id: 'tokens',    label: 'Cloud Tokens', icon: '☁️' },
  ];

  let activeTab = $state('keys');
  let unlockPassword = $state('');
  let unlocking = $state(false);

  // Real key inventory derived from the host catalog.
  type KeyRow = {
    id: string;
    name: string;
    fingerprint: string;
    type: string;
    created: string;
    host?: string;
  };
  let keys = $state<KeyRow[]>([]);
  let passwordHosts = $state<{ id: string; name: string; host: string; created?: string }[]>([]);
  let cloudTokens = $state<{ id: string; provider: string; name: string; created?: string }[]>([]);
  let loading = $state(false);

  async function refresh() {
    if (!appStore.vaultUnlocked) return;
    loading = true;
    try {
      if (IS_BROWSER) {
        // REST endpoints aren't yet exposed for vault enumeration over
        // the wire (intentional — vault contents stay desktop-only).
        keys = [];
        passwordHosts = [];
        cloudTokens = [];
        return;
      }
      const { ListHosts } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/hostservice'
      );
      const hosts = ((await ListHosts()) ?? []) as any[];

      // SSH key auth → keys panel.
      keys = hosts
        .filter((h) => h.auth_method === 'key')
        .map((h) => ({
          id: h.id,
          name: h.label || h.hostname,
          host: h.hostname,
          fingerprint: h.credential_fingerprint || h.public_key_sha256 || '—',
          type: h.key_type || 'ssh',
          created: (h.created_at || '').slice(0, 10),
        }));

      // Password auth → host auth panel.
      passwordHosts = hosts
        .filter((h) => h.auth_method === 'password' && h.has_password)
        .map((h) => ({
          id: h.id,
          name: h.label || h.hostname,
          host: h.hostname,
          created: (h.created_at || '').slice(0, 10),
        }));

      // Cloud tokens — bookmark-tagged hosts with a `token` auth or
      // explicit category=cloud get surfaced here.
      cloudTokens = hosts
        .filter((h) => h.category === 'cloud' || h.auth_method === 'token')
        .map((h) => ({
          id: h.id,
          provider: h.category || 'unknown',
          name: h.label || h.hostname,
          created: (h.created_at || '').slice(0, 10),
        }));
    } catch (e: any) {
      appStore.notify('Vault refresh failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  async function handleUnlock() {
    if (!unlockPassword.trim()) return;
    unlocking = true;
    try {
      const { guardedUnlock } = await import('@lib/bridge');
      await guardedUnlock(unlockPassword, false);
      unlockPassword = '';
      // vault:unlocked event flips appStore.vaultUnlocked via the bridge.
      // Wait one tick for the store update, then refresh.
      setTimeout(refresh, 100);
    } catch (e: any) {
      appStore.notify('Unlock failed', 'error', e?.message ?? String(e));
    } finally {
      unlocking = false;
    }
  }

  $effect(() => {
    if (appStore.vaultUnlocked) void refresh();
  });

  onMount(refresh);

  const keyColumns = [
    { key: 'name',        label: 'Key Name',   sortable: true },
    { key: 'host',        label: 'Host',       width: '160px' },
    { key: 'type',        label: 'Type',       width: '100px' },
    { key: 'fingerprint', label: 'Fingerprint' },
    { key: 'created',     label: 'Added',      width: '110px' },
  ];
</script>

<PageLayout title="Credential Vault" subtitle="Encrypted storage for infrastructure access">
  {#snippet toolbar()}
    {#if appStore.vaultUnlocked}
      <Button variant="secondary" size="sm" onclick={refresh} disabled={loading}>
        {loading ? 'Loading…' : 'Refresh'}
      </Button>
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
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
        <KPI label="Stored SSH Keys" value={keys.length.toString()} variant={keys.length > 0 ? 'accent' : 'muted'} />
        <KPI label="Password Hosts" value={passwordHosts.length.toString()} variant="muted" />
        <KPI label="Cloud Tokens"   value={cloudTokens.length.toString()} variant="muted" />
      </div>

      <Tabs tabs={vaultTabs} bind:active={activeTab} />

      {#if activeTab === 'keys'}
        <div class="flex flex-col gap-4 flex-1 min-h-0">
          <div class="flex justify-between items-center">
            <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Managed SSH Keys</h3>
            <Button variant="secondary" size="sm" onclick={() => push('/ssh')}>
              <KeyRound size={11} class="mr-1" />Add Key
              <ExternalLink size={9} class="ml-1" />
            </Button>
          </div>
          {#if keys.length === 0}
            <EmptyState
              title={loading ? 'Loading…' : 'No SSH keys stored yet'}
              description="Add a host with key-based auth from the SSH Bookmarks page; its key fingerprint will appear here."
              icon="🔑"
            >
              {#snippet action()}
                <Button variant="secondary" onclick={() => push('/ssh')}>Open SSH Bookmarks</Button>
              {/snippet}
            </EmptyState>
          {:else}
            <DataTable data={keys} columns={keyColumns}>
              {#snippet render({ value, col })}
                {#if col.key === 'type'}
                  <code class="text-[10px] bg-surface-2 px-1.5 py-0.5 rounded border border-border-primary text-accent">{value}</code>
                {:else if col.key === 'fingerprint'}
                  <span class="text-[10px] font-mono opacity-70">{value}</span>
                {:else}
                  {value}
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>

      {:else if activeTab === 'passwords'}
        <div class="flex flex-col gap-4 flex-1 min-h-0">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Hosts with Stored Passwords</h3>
          {#if passwordHosts.length === 0}
            <EmptyState
              title={loading ? 'Loading…' : 'No password-auth hosts'}
              description="Hosts you connect to with password auth (rather than key) appear here. Passwords are AES-GCM encrypted at rest."
              icon="🔒"
            >
              {#snippet action()}
                <Button variant="secondary" onclick={() => push('/ssh')}>Open SSH Bookmarks</Button>
              {/snippet}
            </EmptyState>
          {:else}
            <DataTable data={passwordHosts} columns={[
              { key: 'name',    label: 'Host',    sortable: true },
              { key: 'host',    label: 'Address', width: '200px' },
              { key: 'created', label: 'Added',   width: '120px' },
            ]} />
          {/if}
        </div>

      {:else}
        <div class="flex flex-col gap-4 flex-1 min-h-0">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Cloud Provider Tokens</h3>
          {#if cloudTokens.length === 0}
            <EmptyState
              title={loading ? 'Loading…' : 'No cloud tokens stored'}
              description="Tag a host with category=cloud or auth_method=token to surface it here. Manage cloud-provider API keys via the Secrets page."
              icon="☁️"
            >
              {#snippet action()}
                <Button variant="secondary" onclick={() => push('/secrets')}>Open Secret Manager</Button>
              {/snippet}
            </EmptyState>
          {:else}
            <DataTable data={cloudTokens} columns={[
              { key: 'name',     label: 'Name',     sortable: true },
              { key: 'provider', label: 'Provider', width: '140px' },
              { key: 'created',  label: 'Added',    width: '120px' },
            ]} />
          {/if}
        </div>
      {/if}
    </div>
  {/if}
</PageLayout>
