<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    vaultStatus, vaultInit, vaultUnlock, vaultLock,
    vaultSet, vaultDelete,
    type VaultStatus,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let status = $state<VaultStatus | null>(null);
  let passphrase = $state('');
  let secretName = $state('');
  let secretValue = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try { status = await vaultStatus(); error = null; }
    catch (e) { error = (e as Error).message; }
  }

  async function action(fn: () => Promise<unknown>) {
    busy = true; error = null;
    try { await fn(); await refresh(); passphrase = ''; }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  onMount(() => { void refresh(); timer = setInterval(refresh, 10000); });
  onDestroy(() => { if (timer) clearInterval(timer); });
</script>

<div class="mx-auto max-w-4xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Vault</p>
    <h2 class="text-2xl font-semibold tracking-tight">AES-256-GCM secret store</h2>
  </header>

  <section class="grid grid-cols-2 gap-4">
    <Tile label="Vault file" value={status?.exists ? 'present' : 'not initialised'}
      hint={status?.exists ? 'oblivra.vault on disk' : 'Initialise with a passphrase'} />
    <Tile label="State" value={status?.unlocked ? 'unlocked' : 'locked'}
      hint={status?.unlocked ? 'AES key in process memory' : 'no in-memory key'} />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-3">
    <h3 class="text-sm font-semibold tracking-wide text-slate-100">
      {status?.exists ? (status.unlocked ? 'Lock' : 'Unlock') : 'Initialise vault'}
    </h3>

    {#if !status?.exists}
      <p class="text-xs text-night-300">
        Pick a strong passphrase. There is <em>no</em> recovery — losing it loses every secret.
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <input type="password" bind:value={passphrase} placeholder="passphrase"
          class="flex-1 min-w-[24ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
          disabled={busy || !passphrase}
          onclick={() => action(() => vaultInit(passphrase))}>
          Initialise
        </button>
      </div>
    {:else if !status.unlocked}
      <div class="flex flex-wrap items-center gap-2">
        <input type="password" bind:value={passphrase} placeholder="passphrase"
          class="flex-1 min-w-[24ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
          disabled={busy || !passphrase}
          onclick={() => action(() => vaultUnlock(passphrase))}>
          Unlock
        </button>
      </div>
    {:else}
      <div class="flex items-center gap-2">
        <span class="text-xs text-signal-success">Vault is unlocked.</span>
        <button class="ml-auto rounded-md border border-night-600 px-3 py-1 text-xs text-slate-100 hover:bg-night-700"
          disabled={busy} onclick={() => action(() => vaultLock())}>
          Lock now
        </button>
      </div>
    {/if}
    {#if error}<p class="text-xs text-signal-error">{error}</p>{/if}
  </section>

  {#if status?.unlocked}
    <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-3">
      <h3 class="text-sm font-semibold tracking-wide text-slate-100">Add secret</h3>
      <div class="flex flex-wrap items-center gap-2">
        <input bind:value={secretName} placeholder="name (e.g. agent.token)"
          class="flex-1 min-w-[20ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        <input type="password" bind:value={secretValue} placeholder="value"
          class="flex-1 min-w-[20ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
          disabled={busy || !secretName || !secretValue}
          onclick={() => action(async () => {
            await vaultSet(secretName, secretValue);
            secretName = ''; secretValue = '';
          })}>
          Save
        </button>
      </div>
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
        Secrets ({status.names?.length ?? 0})
      </div>
      {#if !status.names || status.names.length === 0}
        <div class="px-4 py-8 text-center text-sm text-night-300">No secrets stored.</div>
      {:else}
        <ul class="divide-y divide-night-700/70 font-mono text-xs">
          {#each status.names as n}
            <li class="flex items-center gap-3 px-4 py-2">
              <span class="text-slate-100">{n}</span>
              <button class="ml-auto rounded-md border border-signal-error/50 px-2 py-0.5 text-[10px] text-signal-error hover:bg-night-700"
                onclick={() => action(() => vaultDelete(n))}>
                delete
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
</div>
