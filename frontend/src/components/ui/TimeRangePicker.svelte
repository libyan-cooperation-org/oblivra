<!--
  TimeRangePicker — preset + custom-range time selector for SIEM,
  HostDetail, ActivityFeed, etc.

  Phase 30.4d: closes the "no time presets" gap from the logging audit.

  Presets:
    - LIVE   (no time bound — updates as events arrive)
    - 5M     (last 5 minutes)
    - 1H     (last hour)
    - 24H    (last 24 hours)
    - 7D     (last 7 days)
    - 30D    (last 30 days)
    - INSTALL (since the agent for this tenant first checked in — the
       "Since agent install" preset from the audit. Resolved at click
       time by the parent page which knows its own scope.)
    - CUSTOM (operator picks start + end via two datetime inputs)

  Output: emits `onChange({ start, end, preset })` where `start`/`end`
  are ISO strings (or `null` for LIVE) and `preset` is the preset id.
  Parent decides what to do with the bounds (push into URL, build OQL
  query clause, refilter local data, etc.) — this component is dumb.
-->
<script lang="ts">
  import { Clock, ChevronDown } from 'lucide-svelte';

  export type TimePreset =
    | 'live'
    | '5m'
    | '1h'
    | '24h'
    | '7d'
    | '30d'
    | 'install'
    | 'custom';

  interface Range {
    start: string | null;
    end: string | null;
    preset: TimePreset;
  }

  interface Props {
    value?: Range;
    /** Resolver for the "Since agent install" preset — caller-supplied
     *  because only the parent page knows the relevant install time. */
    installStart?: string | null;
    onChange?: (range: Range) => void;
  }

  let {
    value = { start: null, end: null, preset: 'live' },
    installStart = null,
    onChange,
  }: Props = $props();

  let active = $state<TimePreset>(value.preset);
  let customOpen = $state(false);
  let customStart = $state<string>('');
  let customEnd = $state<string>('');

  const PRESETS: Array<{ id: TimePreset; label: string; mins?: number }> = [
    { id: 'live',    label: 'LIVE' },
    { id: '5m',      label: '5M',     mins: 5 },
    { id: '1h',      label: '1H',     mins: 60 },
    { id: '24h',     label: '24H',    mins: 60 * 24 },
    { id: '7d',      label: '7D',     mins: 60 * 24 * 7 },
    { id: '30d',     label: '30D',    mins: 60 * 24 * 30 },
    { id: 'install', label: 'INSTALL' },
  ];

  function pick(preset: TimePreset) {
    active = preset;
    if (preset === 'live') {
      onChange?.({ start: null, end: null, preset });
      return;
    }
    if (preset === 'custom') {
      customOpen = true;
      return;
    }
    if (preset === 'install') {
      onChange?.({ start: installStart, end: null, preset });
      return;
    }
    const def = PRESETS.find((p) => p.id === preset);
    if (!def?.mins) return;
    const end = new Date();
    const start = new Date(end.getTime() - def.mins * 60_000);
    onChange?.({ start: start.toISOString(), end: end.toISOString(), preset });
  }

  function applyCustom() {
    if (!customStart) return;
    const range: Range = {
      start: new Date(customStart).toISOString(),
      end: customEnd ? new Date(customEnd).toISOString() : null,
      preset: 'custom',
    };
    customOpen = false;
    onChange?.(range);
  }
</script>

<div class="trp" role="toolbar" aria-label="Time range">
  <Clock class="trp-leading-icon" size={11} strokeWidth={1.6} />
  {#each PRESETS as p}
    <button
      type="button"
      class="trp-btn"
      class:active={active === p.id}
      onclick={() => pick(p.id)}
      title={
        p.id === 'live'    ? 'Live tail — no time filter' :
        p.id === 'install' ? 'Since this tenant\'s first agent check-in' :
        `Last ${p.label}`
      }
    >
      {p.label}
    </button>
  {/each}

  <button
    type="button"
    class="trp-btn"
    class:active={active === 'custom'}
    onclick={() => pick('custom')}
    title="Custom range"
  >
    CUSTOM
    <ChevronDown size={9} strokeWidth={1.8} />
  </button>

  {#if customOpen}
    <div class="trp-popover" role="dialog" aria-label="Custom time range">
      <label class="trp-field">
        <span>Start</span>
        <input type="datetime-local" bind:value={customStart} />
      </label>
      <label class="trp-field">
        <span>End</span>
        <input type="datetime-local" bind:value={customEnd} placeholder="now" />
      </label>
      <div class="trp-actions">
        <button type="button" class="trp-cancel" onclick={() => (customOpen = false)}>Cancel</button>
        <button type="button" class="trp-apply" onclick={applyCustom} disabled={!customStart}>Apply</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .trp {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 2px;
    padding: 3px 4px;
    background: var(--color-surface-2);
    border: 1px solid var(--color-border-primary);
    border-radius: 4px;
  }

  :global(.trp-leading-icon) {
    color: var(--color-text-muted);
    margin-right: 4px;
  }

  .trp-btn {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 3px 8px;
    background: transparent;
    border: none;
    border-radius: 3px;
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 700;
    color: var(--color-text-muted);
    cursor: pointer;
    letter-spacing: 0.05em;
    transition: color 100ms, background 100ms;
  }
  .trp-btn:hover { color: var(--color-text-heading); background: var(--color-surface-3); }
  .trp-btn.active {
    color: var(--color-accent-hover);
    background: var(--color-sev-info-bg);
  }

  .trp-popover {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    z-index: 50;
    background: var(--color-surface-2);
    border: 1px solid var(--color-border-primary);
    border-radius: 6px;
    padding: 10px;
    min-width: 220px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .trp-field {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .trp-field input {
    background: var(--color-surface-1);
    border: 1px solid var(--color-border-primary);
    border-radius: 3px;
    color: var(--color-text-heading);
    padding: 4px 6px;
    font-family: var(--font-mono);
    font-size: 11px;
  }
  .trp-actions {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
    margin-top: 4px;
  }
  .trp-cancel,
  .trp-apply {
    padding: 4px 10px;
    background: transparent;
    border: 1px solid var(--color-border-primary);
    border-radius: 3px;
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 700;
    cursor: pointer;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
  }
  .trp-cancel:hover { color: var(--color-text-heading); }
  .trp-apply {
    background: var(--color-accent);
    border-color: var(--color-accent);
    color: white;
  }
  .trp-apply:disabled { opacity: 0.4; cursor: not-allowed; }
  .trp-apply:not(:disabled):hover { background: var(--color-accent-hover); }
</style>
