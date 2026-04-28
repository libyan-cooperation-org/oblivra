<!--
  OBLIVRA — Session Playback (Svelte 5)
  Forensic replay of recorded terminal telemetry.

  Audit fix — the page used to render a hardcoded eventLog and fake
  metadata block (Origin Host: 10.0.4.15 (prod-gateway), Operator:
  maverick (UID: 1000), and a 4-line fictional /root/secrets.gpg
  story). Operators reading this couldn't tell whether they were
  looking at a real session or a demo. Worse: the static title bar
  said "TS-9921" — a session ID that probably doesn't exist.

  We now:
    • Read `?id=...` from the hash URL (e.g. #/session-playback?id=rec-xxx)
    • Call RecordingService.GetRecordingMeta(id) + GetRecordingFrames(id)
    • Render a real frame timeline + real metadata
    • Show an honest "select a recording" banner if no id was given
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Button, Badge } from '@components/ui';
  import { Play, Pause, SkipBack, SkipForward, Clock, Shield } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { getCurrentPath, push } from '@lib/router.svelte';

  let playing = $state(false);
  let progress = $state(0);
  let speed = $state(1);

  let recordingId = $state<string | null>(null);
  let meta = $state<Record<string, any> | null>(null);
  let frames = $state<Array<Record<string, any>>>([]);
  let loadError = $state<string | null>(null);
  let loading = $state(false);

  // Parse `?id=...` out of the hash. The router uses hash-based URLs
  // (#/session-playback?id=rec-abc) so a simple URLSearchParams over
  // the hash's query portion works.
  function parseRecordingIdFromHash(): string | null {
    const path = getCurrentPath();
    const q = path.indexOf('?');
    if (q === -1) return null;
    return new URLSearchParams(path.slice(q + 1)).get('id');
  }

  async function loadRecording(id: string) {
    if (IS_BROWSER) {
      // Recording playback is desktop-only — the frames live on local
      // disk and the WebView2 binary owns them. Be honest.
      loadError = 'Session playback is desktop-only. Open OBLIVRA on the desktop to replay this recording.';
      return;
    }
    loading = true;
    loadError = null;
    try {
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/recordingservice');
      const [m, fr] = await Promise.all([
        svc.GetRecordingMeta(id),
        svc.GetRecordingFrames(id),
      ]);
      meta = (m ?? null) as Record<string, any> | null;
      frames = (fr ?? []) as Array<Record<string, any>>;
    } catch (err: any) {
      console.error('Recording load failed:', err);
      loadError = String(err?.message ?? err);
    } finally {
      loading = false;
    }
  }

  // Derive display fields from the metadata. RecordingMetadata
  // (sharing.RecordingMetadata) carries host_label, session_id,
  // operator, started_at, ended_at, frame_count — we surface what's
  // available with "—" fallbacks so we never invent.
  const display = $derived({
    sessionId: meta?.session_id ?? meta?.id ?? recordingId ?? '—',
    host:      meta?.host_label ?? meta?.host ?? '—',
    operator:  meta?.operator ?? meta?.user ?? '—',
    started:   meta?.started_at ?? meta?.start_time ?? '—',
    duration:  meta?.duration_ms ? `${Math.round(Number(meta.duration_ms) / 1000)}s` : (meta?.duration ?? '—'),
    frameCount: frames.length,
  });

  // Render frames as a timeline. RecordingFrame is { ts, type, data, ... }
  // so we degrade gracefully: any frame without a `type` becomes "frame".
  const eventLog = $derived.by(() => {
    return frames.slice(0, 200).map((f, i) => ({
      time: typeof f.ts === 'number' ? `${(f.ts / 1000).toFixed(2)}s` : (f.ts ?? `#${i}`),
      type: String(f.type ?? 'frame'),
      content: typeof f.data === 'string' ? f.data : JSON.stringify(f.data ?? f),
    }));
  });

  onMount(() => {
    recordingId = parseRecordingIdFromHash();
    if (recordingId) {
      loadRecording(recordingId);
    }
  });
</script>

<PageLayout title="Forensic Replay" subtitle={recordingId ? `Auditing recording ID: ${display.sessionId}` : 'Select a recording from the Recordings page'}>
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Badge variant={recordingId ? 'warning' : 'muted'}>{recordingId ? 'HI-RES TELEMETRY' : 'NO SESSION'}</Badge>
      <Button variant="secondary" size="sm" onclick={() => push('/recordings')}>Open Recordings</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    {#if !recordingId}
      <div class="flex-1 bg-surface-1 border border-border-primary border-dashed rounded-md p-12 flex flex-col items-center justify-center gap-3 text-center">
        <Shield size={48} class="text-text-muted opacity-60" />
        <div class="text-sm font-bold text-text-heading">No recording selected</div>
        <p class="text-[11px] text-text-muted max-w-[420px]">
          Pass a recording id via the URL (e.g.&nbsp;<code class="font-mono">#/session-playback?id=rec-xyz</code>),
          or open the Recordings page and click a row to navigate here.
        </p>
        <Button variant="primary" size="sm" onclick={() => push('/recordings')}>Browse recordings</Button>
      </div>
    {:else if loadError}
      <div class="flex-1 bg-surface-1 border border-error/40 rounded-md p-8 flex flex-col items-center justify-center gap-2 text-center">
        <div class="text-sm font-bold text-error">Recording load failed</div>
        <p class="text-[11px] text-text-muted font-mono">{loadError}</p>
      </div>
    {:else}
      <!-- Playback Engine -->
      <div class="flex-1 bg-black border border-border-primary rounded-md relative flex flex-col overflow-hidden shadow-2xl">
        <div class="flex-1 p-6 font-mono text-[12px] text-success/90 leading-relaxed overflow-y-auto">
          <div class="opacity-40 mb-4"># OBLIVRA Forensic Playback Engine — recording {display.sessionId}</div>
          {#if loading}
            <div class="opacity-60">Loading frames…</div>
          {:else if frames.length === 0}
            <div class="opacity-60">No frames available for this recording.</div>
          {:else}
            {#each frames.slice(0, 50) as f, i}
              <div class="mb-1 whitespace-pre-wrap break-words">{typeof f.data === 'string' ? f.data : JSON.stringify(f.data ?? f)}</div>
            {/each}
            {#if frames.length > 50}
              <div class="mt-3 opacity-50 italic">… {frames.length - 50} more frames; scroll the timeline panel below for the full event list.</div>
            {/if}
          {/if}
        </div>

        <!-- Controls -->
        <div class="p-4 bg-surface-2 border-t border-border-primary">
          <div class="flex flex-col gap-4">
            <!-- Seek bar -->
            <div class="relative w-full h-1.5 bg-surface-3 rounded-full cursor-pointer">
              <div class="absolute h-full bg-accent rounded-full transition-all" style="width: {progress}%"></div>
              <div class="absolute w-3 h-3 bg-white border-2 border-accent rounded-full -top-0.5 shadow-md" style="left: {progress}%"></div>
            </div>

            <div class="flex items-center justify-between">
              <div class="flex items-center gap-4">
                <button class="text-text-muted hover:text-text-primary"><SkipBack size={18} /></button>
                <button
                  class="w-10 h-10 rounded-full bg-accent text-white flex items-center justify-center hover:bg-accent/80 transition-colors"
                  onclick={() => playing = !playing}
                >
                  {#if playing}<Pause size={20} />{:else}<Play size={20} />{/if}
                </button>
                <button class="text-text-muted hover:text-text-primary"><SkipForward size={18} /></button>
              </div>

              <div class="flex items-center gap-6">
                <div class="flex items-center gap-2 text-[11px] font-mono text-text-muted">
                  <Clock size={12} />
                  <span>{display.duration}</span>
                </div>
                <div class="flex bg-surface-3 rounded-sm p-0.5">
                  {#each [1, 2, 4] as s}
                    <button
                      class="px-2 py-1 text-[9px] font-bold rounded-sm {speed === s ? 'bg-accent text-white' : 'text-text-muted hover:bg-surface-0'}"
                      onclick={() => speed = s}
                    >{s}x</button>
                  {/each}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Side Analysis -->
      <div class="h-48 flex gap-5">
        <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3">
          <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Event Timeline ({display.frameCount} frames)</div>
          <div class="flex-1 overflow-y-auto space-y-2">
            {#if eventLog.length === 0}
              <div class="text-[11px] text-text-muted">No frames available.</div>
            {:else}
              {#each eventLog as event, i (i)}
                <div class="flex items-center gap-3 text-[10px]">
                  <span class="text-text-muted font-mono w-12 shrink-0">{event.time}</span>
                  <Badge variant={event.type === 'warning' ? 'error' : 'info'} size="xs">{event.type}</Badge>
                  <span class="text-text-secondary truncate flex-1">{event.content}</span>
                </div>
              {/each}
            {/if}
          </div>
        </div>

        <div class="w-72 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
          <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Metadata</div>
          <div class="space-y-3">
            <div class="flex flex-col">
              <span class="text-[9px] text-text-muted uppercase">Origin Host</span>
              <span class="text-xs font-bold text-text-heading break-all">{display.host}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-[9px] text-text-muted uppercase">Operator</span>
              <span class="text-xs font-bold text-accent break-all">{display.operator}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-[9px] text-text-muted uppercase">Started</span>
              <span class="text-xs text-text-secondary font-mono break-all">{display.started}</span>
            </div>
          </div>
        </div>
      </div>
    {/if}
  </div>
</PageLayout>
