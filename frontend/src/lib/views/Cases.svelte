<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    casesList, caseTimeline, caseConfidence,
    caseOpen, caseSeal, caseLegalSubmit, caseLegalApprove, caseLegalReject,
    type CaseSummary, type CaseConfidence,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let cases = $state<CaseSummary[]>([]);
  let selected = $state<string | null>(null);
  let detail = $state<CaseSummary | null>(null);
  let timeline = $state<{ kind: string; timestamp: string; title: string; severity?: string; detail?: string }[]>([]);
  let confidence = $state<CaseConfidence | null>(null);
  let title = $state('');
  let host = $state('');
  let busy = $state(false);
  let error = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refreshList() {
    try { cases = await casesList(); } catch (e) { error = (e as Error).message; }
  }

  async function loadDetail(id: string) {
    selected = id;
    try {
      const [tl, conf] = await Promise.all([caseTimeline(id), caseConfidence(id)]);
      detail = cases.find((c) => c.id === id) ?? null;
      timeline = tl;
      confidence = conf;
    } catch (e) { error = (e as Error).message; }
  }

  async function open() {
    if (!title) { error = 'title required'; return; }
    busy = true; error = null;
    try {
      const c = await caseOpen({ title, hostId: host });
      title = ''; host = '';
      await refreshList();
      void loadDetail(c.id);
    } catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  async function transition(fn: (id: string) => Promise<unknown>) {
    if (!selected) return;
    busy = true; error = null;
    try { await fn(selected); await refreshList(); await loadDetail(selected); }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  onMount(() => { void refreshList(); timer = setInterval(refreshList, 5000); });
  onDestroy(() => { if (timer) clearInterval(timer); });

  const reportLink = $derived(selected ? `/api/v1/cases/${selected}/report.html` : '');
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Cases</p>
    <h2 class="text-2xl font-semibold tracking-tight">Frozen-snapshot investigations · legal-review workflow · evidence packages</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Cases" value={cases.length} />
    <Tile label="In legal review" value={cases.filter((c) => c.state === 'legal-review').length} />
    <Tile label="Sealed" value={cases.filter((c) => c.state === 'sealed').length} />
    <Tile label="Confidence (selected)" value={confidence ? `${confidence.score}%` : '—'} hint={confidence?.explanation ?? ''} />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-2">
    <h3 class="text-sm font-semibold tracking-wide text-slate-100">Open new case</h3>
    <div class="flex flex-wrap items-center gap-2">
      <input bind:value={title} placeholder="title" class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      <input bind:value={host} placeholder="hostId (optional)" class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100" />
      <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600 disabled:opacity-50" disabled={busy} onclick={open}>Open</button>
      {#if error}<span class="text-xs text-signal-error">{error}</span>{/if}
    </div>
  </section>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">All cases</div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each cases as c}
          <li>
            <button type="button"
              class="flex w-full items-center justify-between px-4 py-2 text-left hover:bg-night-700/40"
              class:bg-night-700={selected === c.id}
              onclick={() => loadDetail(c.id)}>
              <span class="truncate text-slate-100">{c.title}</span>
              <span class="ml-2 rounded bg-night-800 px-1.5 py-0.5 text-[10px] text-night-200">{c.state}</span>
            </button>
          </li>
        {/each}
      </ul>
    </section>

    <section class="rounded-xl border border-night-700 bg-night-900/70 lg:col-span-2">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">
        {detail ? detail.title : 'Pick a case'}
      </div>
      {#if detail}
        <div class="grid grid-cols-2 gap-3 px-4 py-3 font-mono text-xs">
          <div><span class="text-night-300">opened by:</span> {detail.openedBy}</div>
          <div><span class="text-night-300">opened at:</span> {new Date(detail.openedAt).toLocaleString()}</div>
          <div><span class="text-night-300">scope host:</span> {detail.scope.hostId ?? '—'}</div>
          <div><span class="text-night-300">state:</span> {detail.state}</div>
          <div class="col-span-2"><span class="text-night-300">audit root @ open:</span>
            <span class="break-all text-slate-100">{detail.scope.auditRootAtOpen}</span>
          </div>
        </div>

        <div class="border-t border-night-700 px-4 py-3 flex flex-wrap gap-2 text-xs">
          <button class="rounded-md border border-night-600 px-3 py-1 text-slate-100 hover:bg-night-700 disabled:opacity-50"
            disabled={busy || detail.state !== 'open'}
            onclick={() => transition(caseLegalSubmit)}>Submit for legal review</button>
          <button class="rounded-md border border-signal-success/50 px-3 py-1 text-signal-success hover:bg-night-700 disabled:opacity-50"
            disabled={busy || detail.state !== 'legal-review'}
            onclick={() => transition((id) => caseLegalApprove(id, 'approved via UI'))}>Legal approve</button>
          <button class="rounded-md border border-signal-error/50 px-3 py-1 text-signal-error hover:bg-night-700 disabled:opacity-50"
            disabled={busy || detail.state !== 'legal-review'}
            onclick={() => transition((id) => caseLegalReject(id, 'rejected via UI'))}>Legal reject</button>
          <button class="rounded-md border border-night-600 bg-accent-500 px-3 py-1 text-white hover:bg-accent-600 disabled:opacity-50"
            disabled={busy || (detail.state !== 'legal-approved' && detail.state !== 'open')}
            onclick={() => transition(caseSeal)}>Seal</button>
          <a class="ml-auto rounded-md border border-night-600 px-3 py-1 text-slate-100 hover:bg-night-700"
            href={reportLink} target="_blank" rel="noopener">Open evidence package (HTML)</a>
        </div>

        <div class="border-t border-night-700 px-4 py-3">
          <h4 class="mb-2 text-sm font-semibold text-slate-100">Timeline ({timeline.length})</h4>
          {#if timeline.length === 0}
            <div class="text-xs text-night-300">No entries — frozen snapshot.</div>
          {:else}
            <ul class="divide-y divide-night-700/70 font-mono text-xs max-h-64 overflow-y-auto scrollbar-thin">
              {#each timeline.slice(0, 50) as t}
                <li class="flex items-start gap-3 py-1">
                  <span class="text-night-300">{new Date(t.timestamp).toLocaleTimeString()}</span>
                  <span class="rounded bg-night-800 px-1.5 py-0.5 text-[10px] text-night-200">{t.kind}</span>
                  <span class="text-slate-100">{t.title}</span>
                  <span class="ml-auto truncate text-night-300 max-w-[28ch]">{t.detail ?? ''}</span>
                </li>
              {/each}
            </ul>
          {/if}
        </div>

        {#if confidence?.contributions}
          <div class="border-t border-night-700 px-4 py-3">
            <h4 class="mb-2 text-sm font-semibold text-slate-100">Confidence breakdown</h4>
            <ul class="space-y-1 font-mono text-xs text-night-200">
              {#each confidence.contributions as c}<li>{c}</li>{/each}
            </ul>
          </div>
        {/if}
      {:else}
        <div class="px-4 py-12 text-center text-sm text-night-300">No case selected.</div>
      {/if}
    </section>
  </div>
</div>
