<!--
  HTTPPanel — Postman-style request runner that tunnels via the selected
  host (curl-via-SSH). Useful for "what does this internal HTTP endpoint
  see when the request comes from inside the prod box?".
-->
<script lang="ts">
  import { Globe2, Send, Loader2 } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { execOnHost } from './useShellSession.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';

  type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD';
  let method = $state<Method>('GET');
  let url = $state('https://example.com');
  let headersText = $state('');
  let bodyText = $state('');

  let running = $state(false);
  let respText = $state('');
  let respMeta = $state<{ statusLine: string; durationMs: number } | null>(null);
  let errorMsg = $state('');

  // Build a curl invocation that prints separator + status code + body so
  // we can parse a clean response on the JS side. -s silences progress; -i
  // includes headers; -w '\n%{http_code}\n' appends the status code last.
  function buildCmd(): string {
    const headerArgs = headersText
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean)
      .map((h) => `-H ${shellEscape(h)}`)
      .join(' ');
    const bodyArg = bodyText.trim() && method !== 'GET' && method !== 'HEAD' ? `--data ${shellEscape(bodyText)}` : '';
    return `curl -s -i -X ${method} ${headerArgs} ${bodyArg} -w '\\n###STATUS###%{http_code}###TIME###%{time_total}\\n' ${shellEscape(url)}`;
  }

  function shellEscape(s: string): string {
    // Single-quote escaping for POSIX shells.
    return `'${s.replace(/'/g, `'\\''`)}'`;
  }

  async function send() {
    if (!shellStore.selectedHostID) {
      toastStore.add({ type: 'warning', title: 'Pick a host first' });
      return;
    }
    if (!url.trim()) {
      toastStore.add({ type: 'warning', title: 'URL required' });
      return;
    }
    running = true;
    errorMsg = '';
    respText = '';
    respMeta = null;
    const start = performance.now();
    try {
      const out = await execOnHost(buildCmd());
      const m = out.match(/###STATUS###(\d+)###TIME###([\d.]+)/);
      const status = m ? m[1] : '?';
      const time = m ? parseFloat(m[2]!) : 0;
      respText = out.replace(/###STATUS###\d+###TIME###[\d.]+\n?$/, '').trim();
      respMeta = {
        statusLine: `HTTP ${status}`,
        durationMs: Math.round(time * 1000) || Math.round(performance.now() - start),
      };
    } catch (e: any) {
      errorMsg = e?.message ?? String(e);
    } finally {
      running = false;
    }
  }
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <Globe2 size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">HTTP (via SSH)</span>
    {#if shellStore.selectedHostID}
      <span class="text-[10px] text-[var(--tx3)]">·
        {shellStore.hosts.find((h) => h.id === shellStore.selectedHostID)?.name ?? shellStore.selectedHostID}
      </span>
    {/if}
  </header>

  <div class="grid min-h-0 flex-1 grid-cols-1 gap-3 overflow-y-auto p-3 lg:grid-cols-2">
    <!-- Request -->
    <section class="space-y-2">
      <div class="flex items-center gap-2">
        <select class="rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 text-xs outline-none" bind:value={method}>
          {#each ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD'] as m}
            <option value={m}>{m}</option>
          {/each}
        </select>
        <input
          class="flex-1 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 font-mono text-xs outline-none focus:border-cyan-400/40"
          placeholder="https://example.com/api/path"
          bind:value={url}
          onkeydown={(e) => e.key === 'Enter' && send()}
        />
        <button
          class="flex items-center gap-1.5 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-3 py-1.5 text-xs text-cyan-200 hover:bg-cyan-400/20 disabled:opacity-50"
          onclick={send}
          disabled={running}
        >
          {#if running}<Loader2 size={12} class="animate-spin" />{:else}<Send size={12} />{/if}
          Send
        </button>
      </div>
      <div>
        <div class="mb-1 text-[10px] uppercase tracking-wider text-[var(--tx3)]">Headers (one per line, "Key: Value")</div>
        <textarea
          class="h-24 w-full resize-none rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 font-mono text-xs outline-none focus:border-cyan-400/40"
          placeholder="Authorization: Bearer abc123"
          bind:value={headersText}
        ></textarea>
      </div>
      <div>
        <div class="mb-1 text-[10px] uppercase tracking-wider text-[var(--tx3)]">Body</div>
        <textarea
          class="h-40 w-full resize-none rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1.5 font-mono text-xs outline-none focus:border-cyan-400/40"
          placeholder="(JSON, form, or raw)"
          bind:value={bodyText}
        ></textarea>
      </div>
    </section>

    <!-- Response -->
    <section class="space-y-2">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Response</div>
      {#if errorMsg}
        <pre class="rounded-md border border-rose-400/30 bg-rose-400/5 p-2 font-mono text-[11px] text-rose-200">{errorMsg}</pre>
      {:else if running}
        <div class="text-sm text-[var(--tx3)]">Awaiting reply…</div>
      {:else if !respText && !respMeta}
        <div class="text-sm text-[var(--tx3)]">Hit Send to make the request.</div>
      {:else}
        {#if respMeta}
          <div class="flex items-center gap-2 text-[11px]">
            <span class="rounded-md border border-cyan-400/30 bg-cyan-400/10 px-2 py-0.5 font-mono text-cyan-200">{respMeta.statusLine}</span>
            <span class="font-mono text-[10px] text-[var(--tx3)]">{respMeta.durationMs}ms</span>
          </div>
        {/if}
        <pre class="max-h-[60vh] overflow-auto rounded-md border border-[var(--b1)] bg-[var(--s0)] p-2 font-mono text-[11px] text-[var(--tx2)]">{respText}</pre>
      {/if}
    </section>
  </div>
</div>
