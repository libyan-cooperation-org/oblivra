<!--
  EntityLink — turns any host / user / IP / process / hash / domain
  string into a click-to-investigate primitive.

  Phase 31 SOC redesign — implements the "everything is clickable"
  rule from the audit spec. Use everywhere an entity name appears
  in a table, timeline, alert detail, log row, etc.

  Usage:
    <EntityLink type="host" id={row.host} />
    <EntityLink type="user" id={alert.user} label={alert.user_display} />
    <EntityLink type="ip"   id={evt.src_ip} />

  Renders inline (span-friendly). Stops click propagation so it
  doesn't accidentally activate row-level click handlers.
-->
<script lang="ts">
  import { Server, User as UserIcon, Globe, Cpu, Hash as HashIcon, Network as NetIcon, AlertTriangle, type Icon as IconType } from 'lucide-svelte';
  import { investigationStore, type EntityType } from '@lib/stores/investigation.svelte';

  interface Props {
    type: EntityType;
    id: string;
    /** Optional display string. Falls back to `id`. */
    label?: string;
    /** Optional metadata to attach to the entity ref (severity, host, etc.). */
    context?: Record<string, any>;
    /** When true, render an icon next to the label. */
    showIcon?: boolean;
    /** Optional CSS hook. */
    class?: string;
  }
  let { type, id, label, context, showIcon = false, class: cls = '' }: Props = $props();

  // Static icon map. Phase 29 lesson: never reflective lookup.
  const ICONS: Record<EntityType, typeof IconType> = {
    host:    Server,
    user:    UserIcon,
    ip:      Globe,
    process: Cpu,
    hash:    HashIcon,
    domain:  NetIcon,
    alert:   AlertTriangle,
  };
  const Icon = $derived(ICONS[type] ?? Globe);

  function open(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    if (!id) return;
    investigationStore.openEntity({ type, id, label, context });
  }
</script>

{#if id}
  <button
    type="button"
    class="entity-link entity-{type} {cls}"
    onclick={open}
    title="Investigate {type}: {id}"
  >
    {#if showIcon}
      <Icon size={10} strokeWidth={1.7} class="entity-icon" />
    {/if}
    <span class="entity-label">{label ?? id}</span>
  </button>
{:else}
  <span class="entity-empty">—</span>
{/if}

<style>
  .entity-link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 0;
    margin: 0;
    background: transparent;
    border: none;
    border-bottom: 1px dashed transparent;
    color: inherit;
    cursor: pointer;
    font-family: inherit;
    font-size: inherit;
    line-height: inherit;
    text-decoration: none;
    transition: color 100ms, border-color 100ms, background 100ms;
    border-radius: 2px;
  }
  .entity-link:hover {
    color: var(--color-accent-hover);
    border-bottom-color: var(--color-accent);
  }
  .entity-link:focus-visible {
    outline: none;
    box-shadow: 0 0 0 1px var(--color-accent);
  }

  /* Subtle type-specific hint colour on hover so the operator gets
     a glanceable signal that this is e.g. a host vs an IP. */
  .entity-host:hover    { color: var(--color-accent-hover); }
  .entity-user:hover    { color: var(--color-sev-info); }
  .entity-ip:hover      { color: var(--color-sev-warn); }
  .entity-process:hover { color: var(--color-sev-info); }
  .entity-hash:hover    { color: var(--color-text-secondary); }
  .entity-domain:hover  { color: var(--color-sev-warn); }
  .entity-alert:hover   { color: var(--color-sev-error); }

  :global(.entity-icon) { opacity: 0.7; flex-shrink: 0; }
  .entity-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .entity-empty {
    color: var(--color-text-muted);
    opacity: 0.5;
  }
</style>
