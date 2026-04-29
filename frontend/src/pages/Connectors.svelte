<!--
  OBLIVRA — I/O Connectors (Slice 5).

  Operator-facing page for managing the agent/server input/output
  pipeline. Backed by:
    GET  /api/v1/io/config  → current YAML
    PUT  /api/v1/io/config  → replace YAML (validates + hot-reloads)
    POST /api/v1/io/test    → validate without committing
    GET  /api/v1/io/stats   → events_in / _out / _drop counters

  Design choices:
   • YAML editor, not a form-builder. The full config surface (5
     input types × 5 output types × per-type knobs × pipeline filters)
     would be 50+ form fields. A YAML pane with syntax-aware tips +
     a "validate" button is faster for the kind of operator who sets
     up SIEM connectors.
   • "Apply" writes to disk; the file watcher hot-reloads. No restart.
   • Stats panel auto-refreshes every 5s so operators see throughput
     change as they tune.
   • Templates dropdown — pre-fills common scenarios (ship /var/log/auth,
     accept Cisco syslog, forward critical to Splunk) so operators
     don't start from a blank page.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, Badge, Button, LastRefreshed } from '@components/ui';
  import { Cable, RefreshCw, Save, FlaskConical, Plug, ArrowLeftRight } from 'lucide-svelte';
  import { apiFetch } from '@lib/apiClient';
  import { appStore } from '@lib/stores/app.svelte';

  let yamlText = $state('');
  let originalYaml = $state('');
  let dirty = $derived(yamlText !== originalYaml);

  let stats = $state<{ events_in: number; events_out: number; events_drop: number } | null>(null);
  // Trust signal — pipeline stats poll every 5s; surface freshness
  // so operators can tell if the connector telemetry has stalled.
  let statsLastSync = $state<Date | null>(null);
  let testResult = $state<{ ok: boolean; error?: string; inputs?: number; outputs?: number } | null>(null);
  let busy = $state(false);
  let loadError = $state<string | null>(null);

  let statsTimer: ReturnType<typeof setInterval> | null = null;

  // Templates surface common scenarios. Operators copy/edit; not
  // intended to be exhaustive, just a "I don't have to start from
  // scratch" lift.
  const TEMPLATES: Array<{ name: string; description: string; yaml: string }> = [
    {
      name: 'Tail /var/log/auth.log → OBLIVRA server',
      description: 'Ship Linux auth events to the central server.',
      yaml: `tls:
  mode: on

inputs:
  - id: auth-log
    type: file
    paths: ["/var/log/auth.log", "/var/log/auth.log.*"]
    sourcetype: linux:auth

outputs:
  - id: primary
    type: oblivra
    server: "https://oblivra.internal:8443"
    fleet_secret: "\${OBLIVRA_FLEET_SECRET}"
`,
    },
    {
      name: 'Cisco / network syslog → server',
      description: 'Listen on UDP 514 for syslog from network gear; ship upstream.',
      yaml: `tls:
  mode: on

inputs:
  - id: net-syslog
    type: syslog
    listen_udp: "0.0.0.0:514"
    sourcetype: cisco:asa

outputs:
  - id: primary
    type: oblivra
    server: "https://oblivra.internal:8443"
    fleet_secret: "\${OBLIVRA_FLEET_SECRET}"
`,
    },
    {
      name: 'HEC endpoint (curl-friendly)',
      description: 'Accept HTTP-Event-Collector events from any script.',
      yaml: `tls:
  mode: on

inputs:
  - id: hec
    type: hec
    listen: "0.0.0.0:8088"
    token: "\${OBLIVRA_HEC_TOKEN}"
    sourcetype: hec:default

outputs:
  - id: primary
    type: oblivra
    server: "https://oblivra.internal:8443"
    fleet_secret: "\${OBLIVRA_FLEET_SECRET}"
`,
    },
    {
      name: 'Parallel-run with Splunk',
      description: 'Forward critical events to existing Splunk during migration.',
      yaml: `tls:
  mode: on

inputs:
  - id: auth-log
    type: file
    paths: ["/var/log/auth.log"]
    sourcetype: linux:auth

outputs:
  - id: primary
    type: oblivra
    server: "https://oblivra.internal:8443"
    fleet_secret: "\${OBLIVRA_FLEET_SECRET}"

  - id: legacy-splunk
    type: syslog
    target: "tcp://splunk-fwd.internal:514"
`,
    },
    {
      name: 'Cold archive → S3',
      description: 'Compliance retention to S3-compatible bucket.',
      yaml: `tls:
  mode: on

inputs:
  - id: auth-log
    type: file
    paths: ["/var/log/auth.log"]
    sourcetype: linux:auth

outputs:
  - id: primary
    type: oblivra
    server: "https://oblivra.internal:8443"
    fleet_secret: "\${OBLIVRA_FLEET_SECRET}"

  - id: cold
    type: s3
    endpoint: "https://s3.us-east-1.amazonaws.com"
    bucket: "oblivra-cold"
    prefix: "year=%Y/month=%m/day=%d/"
    region: "us-east-1"
    access_key: "\${AWS_ACCESS_KEY_ID}"
    secret_key: "\${AWS_SECRET_ACCESS_KEY}"
    rotate_after: "5m"
`,
    },
  ];

  async function loadConfig() {
    busy = true;
    loadError = null;
    try {
      const res = await apiFetch('/api/v1/io/config');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const body = await res.json();
      yamlText = body.yaml ?? '';
      originalYaml = yamlText;
    } catch (e: any) {
      loadError = e?.message ?? String(e);
    } finally {
      busy = false;
    }
  }

  async function refreshStats() {
    try {
      const res = await apiFetch('/api/v1/io/stats');
      if (res.ok) {
        stats = await res.json();
        statsLastSync = new Date();
      }
    } catch { /* network blip — keep last value */ }
  }

  async function validateYaml() {
    busy = true;
    testResult = null;
    try {
      const res = await apiFetch('/api/v1/io/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ yaml: yamlText }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      testResult = await res.json();
    } catch (e: any) {
      testResult = { ok: false, error: e?.message ?? String(e) };
    } finally {
      busy = false;
    }
  }

  async function applyConfig() {
    busy = true;
    try {
      const res = await apiFetch('/api/v1/io/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ yaml: yamlText }),
      });
      if (!res.ok) {
        const txt = await res.text().catch(() => '');
        throw new Error(`HTTP ${res.status}: ${txt}`);
      }
      const body = await res.json();
      originalYaml = yamlText;
      appStore.notify(
        `Applied ${body.inputs} input(s) + ${body.outputs} output(s)`,
        'success',
        'Pipeline hot-reloaded — no restart needed.',
      );
    } catch (e: any) {
      appStore.notify('Apply failed', 'error', e?.message ?? String(e));
    } finally {
      busy = false;
    }
  }

  function loadTemplate(t: typeof TEMPLATES[0]) {
    if (dirty && !confirm('Discard your unsaved changes and load this template?')) return;
    yamlText = t.yaml;
  }

  onMount(() => {
    void loadConfig();
    void refreshStats();
    statsTimer = setInterval(refreshStats, 5_000);
  });

  onDestroy(() => {
    if (statsTimer) clearInterval(statsTimer);
  });
</script>

<PageLayout title="Connectors" subtitle="Inputs / outputs / pipeline · YAML edit · hot-reloads on apply">
  {#snippet toolbar()}
    <LastRefreshed time={statsLastSync} staleThresholdSec={15} />
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={loadConfig} disabled={busy}>
      {busy ? 'Loading…' : 'Reload from disk'}
    </Button>
    <Button variant="secondary" size="sm" icon={FlaskConical} onclick={validateYaml} disabled={busy || !yamlText}>
      Validate
    </Button>
    <Button variant="primary" size="sm" icon={Save} onclick={applyConfig} disabled={busy || !dirty}>
      Apply
    </Button>
  {/snippet}

  <div class="flex h-full gap-3">
    <!-- LEFT: Editor -->
    <div class="flex-1 flex flex-col gap-2 min-w-0">
      <div class="flex items-center gap-2 text-[var(--fs-micro)] text-text-muted">
        <Cable size={11} class="text-accent" />
        <span class="font-mono uppercase tracking-widest">io config (yaml)</span>
        {#if dirty}
          <Badge variant="warning" size="xs">UNSAVED</Badge>
        {/if}
        <span class="ml-auto font-mono">{yamlText.split('\n').length} lines</span>
      </div>

      {#if loadError}
        <div class="bg-error/10 border border-error/30 text-error rounded p-3 text-[var(--fs-label)]">
          Failed to load config: {loadError}
        </div>
      {/if}

      <!-- Plain textarea — operators editing IO config want a focused
           text surface, not a heavy-weight editor. Tabs + 2-space
           indent enforced via CSS. -->
      <textarea
        class="flex-1 min-h-0 w-full bg-surface-0 border border-border-primary rounded font-mono text-[12px] p-3 leading-relaxed text-text-secondary placeholder:text-text-muted focus:border-accent focus:outline-none resize-none"
        bind:value={yamlText}
        spellcheck="false"
        placeholder="# Paste or type a YAML config..."
      ></textarea>

      {#if testResult}
        {#if testResult.ok}
          <div class="bg-success/5 border border-success/30 text-success rounded p-2 text-[var(--fs-label)] flex items-center gap-2">
            <Plug size={11} />
            Validated: {testResult.inputs ?? 0} input(s), {testResult.outputs ?? 0} output(s).
          </div>
        {:else}
          <div class="bg-error/10 border border-error/30 text-error rounded p-2 text-[var(--fs-label)] font-mono whitespace-pre-wrap">
            ✗ {testResult.error ?? 'invalid'}
          </div>
        {/if}
      {/if}
    </div>

    <!-- RIGHT: Stats + templates -->
    <aside class="w-[300px] flex flex-col gap-3 shrink-0">
      <div class="bg-surface-1 border border-border-primary rounded-md p-3">
        <div class="flex items-center gap-2 mb-2">
          <ArrowLeftRight size={11} class="text-accent" />
          <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">
            Pipeline throughput
          </span>
        </div>
        {#if !stats}
          <div class="text-[var(--fs-label)] text-text-muted italic">Loading…</div>
        {:else}
          <ul class="grid grid-cols-3 gap-2">
            <li class="flex flex-col gap-0.5 items-center">
              <span class="font-mono text-[var(--fs-heading)] text-text-heading">{stats.events_in.toLocaleString()}</span>
              <span class="text-[var(--fs-micro)] text-text-muted uppercase">in</span>
            </li>
            <li class="flex flex-col gap-0.5 items-center">
              <span class="font-mono text-[var(--fs-heading)] text-success">{stats.events_out.toLocaleString()}</span>
              <span class="text-[var(--fs-micro)] text-text-muted uppercase">out</span>
            </li>
            <li class="flex flex-col gap-0.5 items-center">
              <span class="font-mono text-[var(--fs-heading)] {stats.events_drop > 0 ? 'text-warning' : 'text-text-muted'}">{stats.events_drop.toLocaleString()}</span>
              <span class="text-[var(--fs-micro)] text-text-muted uppercase">drop</span>
            </li>
          </ul>
          <p class="text-[var(--fs-micro)] text-text-muted mt-2 leading-relaxed">
            Drop indicates an output queue is wedged. Tune that output's batch_size or fix its connectivity.
          </p>
        {/if}
      </div>

      <div class="bg-surface-1 border border-border-primary rounded-md p-3 flex flex-col gap-2 flex-1 min-h-0">
        <div class="flex items-center gap-2">
          <Plug size={11} class="text-accent" />
          <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">
            Templates
          </span>
        </div>
        <div class="flex flex-col gap-1.5 overflow-auto">
          {#each TEMPLATES as t}
            <button
              class="text-start px-2 py-2 bg-surface-2 hover:bg-surface-3 border border-border-primary rounded transition-colors duration-fast"
              onclick={() => loadTemplate(t)}
            >
              <div class="text-[var(--fs-label)] font-bold text-text-secondary">{t.name}</div>
              <div class="text-[var(--fs-micro)] text-text-muted leading-relaxed mt-0.5">{t.description}</div>
            </button>
          {/each}
        </div>
        <p class="text-[var(--fs-micro)] text-text-muted leading-relaxed pt-1 border-t border-border-primary">
          Templates load into the editor. Edit, validate, then Apply. Existing config is replaced — copy the inputs/outputs you want to keep.
        </p>
      </div>
    </aside>
  </div>
</PageLayout>
