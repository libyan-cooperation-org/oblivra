<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    notificationsList, notificationsAdd, notificationsTest, notificationsDelete,
    type NotificationChannel,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let channels = $state<NotificationChannel[]>([]);
  let loadError = $state<string | null>(null);
  let busy = $state(false);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  // Add-channel form state.
  let formKind = $state<'email' | 'webhook'>('email');
  let formName = $state('');
  let formMinSev = $state<'low' | 'medium' | 'high' | 'critical' | ''>('high');
  let smtpHost = $state('');
  let smtpPort = $state(587);
  let smtpFrom = $state('');
  let smtpTo = $state('');
  let smtpUser = $state('');
  let smtpPass = $state('');
  let webhookUrl = $state('');
  let formError = $state<string | null>(null);
  let testResult = $state<Record<string, string>>({});

  async function refresh() {
    const seq = ++inFlight;
    try {
      const list = await notificationsList();
      if (seq !== inFlight) return;
      channels = list;
      loadError = null;
    } catch (err) {
      if (seq !== inFlight) return;
      loadError = (err as Error).message;
    }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(() => void refresh(), 4000);
  });
  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  async function add() {
    busy = true;
    formError = null;
    try {
      const payload: any = {
        kind: formKind,
        name: formName,
        minSeverity: formMinSev || undefined,
      };
      if (formKind === 'email') {
        payload.smtpHost = smtpHost;
        payload.smtpPort = smtpPort;
        payload.smtpFrom = smtpFrom;
        payload.smtpTo = smtpTo;
        if (smtpUser) payload.smtpUsername = smtpUser;
        if (smtpPass) payload.smtpPassword = smtpPass;
      } else {
        payload.webhookUrl = webhookUrl;
      }
      await notificationsAdd(payload);
      // Reset form.
      formName = ''; smtpHost = ''; smtpFrom = ''; smtpTo = '';
      smtpUser = ''; smtpPass = ''; webhookUrl = '';
      await refresh();
    } catch (err) {
      formError = (err as Error).message;
    } finally {
      busy = false;
    }
  }

  async function test(id: string) {
    testResult = { ...testResult, [id]: 'sending…' };
    try {
      const r = await notificationsTest(id);
      testResult = { ...testResult, [id]: r.delivered ? '✓ delivered' : `✗ ${r.error ?? 'failed'}` };
    } catch (err) {
      testResult = { ...testResult, [id]: '✗ ' + (err as Error).message };
    }
    setTimeout(() => {
      testResult = Object.fromEntries(Object.entries(testResult).filter(([k]) => k !== id));
    }, 5000);
  }

  async function remove(id: string) {
    if (!confirm('Delete this channel?')) return;
    busy = true;
    try {
      await notificationsDelete(id);
      await refresh();
    } catch (err) {
      loadError = (err as Error).message;
    } finally {
      busy = false;
    }
  }

  function relTime(ts?: string): string {
    if (!ts) return 'never';
    const sec = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Notifications</p>
    <h2 class="text-2xl font-semibold tracking-tight">Email · Webhook delivery for alerts</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Channels" value={channels.length} />
    <Tile label="Email" value={channels.filter((c) => c.kind === 'email').length} />
    <Tile label="Webhook" value={channels.filter((c) => c.kind === 'webhook').length} />
    <Tile
      label="Errors"
      value={channels.filter((c) => c.lastError).length}
      hint="last delivery failed"
    />
  </section>

  {#if loadError}
    <p class="text-xs text-signal-error">Failed to load: {loadError}</p>
  {/if}

  <!-- Add-channel form -->
  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4">
    <h3 class="mb-3 text-sm font-semibold tracking-wide text-slate-100">Add channel</h3>
    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <label class="text-xs">
        <div class="text-night-300 mb-1">Kind</div>
        <select bind:value={formKind} class="w-full rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
          <option value="email">email</option>
          <option value="webhook">webhook</option>
        </select>
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Name</div>
        <input bind:value={formName} placeholder="e.g. soc-pager" class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      </label>
      <label class="text-xs">
        <div class="text-night-300 mb-1">Min severity</div>
        <select bind:value={formMinSev} class="w-full rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
          <option value="">any</option>
          <option value="low">low</option>
          <option value="medium">medium</option>
          <option value="high">high</option>
          <option value="critical">critical</option>
        </select>
      </label>

      {#if formKind === 'email'}
        <label class="text-xs">
          <div class="text-night-300 mb-1">SMTP host</div>
          <input bind:value={smtpHost} placeholder="smtp.example.com" class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
        <label class="text-xs">
          <div class="text-night-300 mb-1">SMTP port</div>
          <input type="number" bind:value={smtpPort} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
        <label class="text-xs">
          <div class="text-night-300 mb-1">From</div>
          <input bind:value={smtpFrom} placeholder="oblivra@example.com" class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
        <label class="text-xs lg:col-span-2">
          <div class="text-night-300 mb-1">To (comma-separated)</div>
          <input bind:value={smtpTo} placeholder="soc@example.com,alerts@example.com" class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
        <label class="text-xs">
          <div class="text-night-300 mb-1">SMTP username (optional)</div>
          <input bind:value={smtpUser} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
        <label class="text-xs">
          <div class="text-night-300 mb-1">SMTP password</div>
          <input type="password" bind:value={smtpPass} class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
      {:else}
        <label class="text-xs lg:col-span-2">
          <div class="text-night-300 mb-1">Webhook URL</div>
          <input bind:value={webhookUrl} placeholder="https://hooks.example.com/path" class="w-full rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
        </label>
      {/if}
    </div>

    {#if formError}
      <p class="mt-3 text-xs text-signal-error">{formError}</p>
    {/if}

    <div class="mt-4 flex items-center gap-2">
      <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white shadow-sm hover:bg-accent-600 disabled:opacity-50"
              disabled={busy || !formName} onclick={add}>
        Save channel
      </button>
      <span class="text-[11px] text-night-300">Throttle: 1 alert per (channel, rule) per 5 min</span>
    </div>
  </section>

  <!-- Channels list -->
  <section class="rounded-xl border border-night-700 bg-night-900/70">
    <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide text-slate-100">
      Channels
    </div>
    {#if channels.length === 0}
      <div class="px-4 py-8 text-center text-sm text-night-300">No channels yet.</div>
    {:else}
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each channels as c (c.id)}
          <li class="flex flex-col gap-1.5 px-4 py-3">
            <div class="flex items-center gap-2">
              <span class="rounded border border-night-600 bg-night-800/70 px-1.5 py-0.5 text-[10px] text-night-200 uppercase">{c.kind}</span>
              <span class="text-slate-100">{c.name}</span>
              {#if c.minSeverity}
                <span class="text-night-300">· ≥{c.minSeverity}</span>
              {/if}
              <span class="ml-auto text-night-300">created {relTime(c.createdAt)}</span>
              <button class="rounded-md border border-night-600 bg-night-800/70 px-2 py-0.5 text-[10px] text-slate-100 hover:bg-night-700 disabled:opacity-50"
                      disabled={busy} onclick={() => test(c.id)}>
                test
              </button>
              <button class="rounded-md border border-signal-error/50 px-2 py-0.5 text-[10px] text-signal-error hover:bg-night-700 disabled:opacity-50"
                      disabled={busy} onclick={() => remove(c.id)}>
                delete
              </button>
            </div>
            {#if c.kind === 'email'}
              <div class="text-night-300">
                {c.smtpUsername ? `auth as ${c.smtpUsername} → ` : ''}{c.smtpHost}:{c.smtpPort}
                · from {c.smtpFrom} → {c.smtpTo}
              </div>
            {:else}
              <div class="text-night-300">→ {(c as any).webhookUrl ?? '—'}</div>
            {/if}
            <div class="text-night-300">
              last delivered: {c.lastDelivered ? relTime(c.lastDelivered) : 'never'}
              {#if c.lastError}
                <span class="ml-2 text-signal-error">· {c.lastError}</span>
              {/if}
            </div>
            {#if testResult[c.id]}
              <div class="rounded bg-night-800/60 px-2 py-1 text-[11px] text-slate-100">test: {testResult[c.id]}</div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
