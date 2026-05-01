<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    webhooksList, webhookRegister, webhookDelete, webhookDeliveries,
    type Webhook, type WebhookDelivery,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let hooks = $state<Webhook[]>([]);
  let recent = $state<WebhookDelivery[]>([]);
  let url = $state('');
  let secret = $state('');
  let minSeverity = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const [h, d] = await Promise.all([webhooksList(), webhookDeliveries()]);
      hooks = h; recent = d; error = null;
    } catch (e) { error = (e as Error).message; }
  }

  async function register() {
    busy = true; error = null;
    try {
      await webhookRegister({ url, secret, minSeverity });
      url = ''; secret = ''; minSeverity = '';
      await refresh();
    } catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  async function remove(id: string) {
    busy = true;
    try { await webhookDelete(id); await refresh(); }
    finally { busy = false; }
  }

  onMount(() => { void refresh(); timer = setInterval(refresh, 5000); });
  onDestroy(() => { if (timer) clearInterval(timer); });
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Webhooks</p>
    <h2 class="text-2xl font-semibold tracking-tight">Outbound alert delivery — informational, not response</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Webhooks" value={hooks.length} />
    <Tile label="Active" value={hooks.filter((h) => !h.disabled).length} />
    <Tile label="Recent deliveries" value={recent.length} hint="last 50" />
    <Tile label="Failed" value={recent.filter((d) => d.error).length} />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-3">
    <h3 class="text-sm font-semibold tracking-wide text-slate-100">Register webhook</h3>
    <p class="text-xs text-night-300">
      OBLIVRA POSTs the alert as JSON. If a secret is set, the body is HMAC-SHA256 signed
      under <code>X-OBLIVRA-Signature: sha256=&lt;hex&gt;</code>. Slack-compatible incoming
      webhooks accept the JSON shape directly.
    </p>
    <div class="grid grid-cols-1 gap-2 lg:grid-cols-3">
      <input bind:value={url} placeholder="https://hooks.example.com/..." class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      <input bind:value={secret} placeholder="HMAC secret (optional)" class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      <select bind:value={minSeverity} class="rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
        <option value="">deliver all severities</option>
        <option value="low">low and above</option>
        <option value="medium">medium and above</option>
        <option value="high">high and above</option>
        <option value="critical">critical only</option>
      </select>
    </div>
    <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50"
      disabled={busy || !url} onclick={register}>
      Register
    </button>
    {#if error}<p class="text-xs text-signal-error">{error}</p>{/if}
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">Registered webhooks</div>
    {#if hooks.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">None registered.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each hooks as h}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="rounded bg-night-800 px-1.5 py-0.5 text-[10px] text-night-200">{h.id}</span>
            <span class="flex-1 truncate text-slate-100">{h.url}</span>
            {#if h.minSeverity}<span class="text-night-300">≥{h.minSeverity}</span>{/if}
            {#if h.lastDelivered}<span class="text-night-300">last: {new Date(h.lastDelivered).toLocaleTimeString()}</span>{/if}
            <button class="rounded-md border border-signal-error/50 px-2 py-0.5 text-[10px] text-signal-error hover:bg-night-700"
              disabled={busy} onclick={() => remove(h.id)}>delete</button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">Recent deliveries</div>
    {#if recent.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No deliveries yet.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each recent as d}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="rounded bg-night-800 px-1.5 py-0.5 text-[10px] text-night-200">{d.webhookId}</span>
            <span class="text-night-300">→ alert {d.alertId}</span>
            {#if d.error}
              <span class="text-signal-error">{d.error}</span>
            {:else}
              <span class="text-signal-success">HTTP {d.status}</span>
            {/if}
            <span class="ml-auto text-night-300">{new Date(d.deliveredAt).toLocaleTimeString()}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
