<script lang="ts">
  // Toast renderer — pinned bottom-right, stacks newest on top.
  // Each toast has a kind (info/success/warn/error) controlling its
  // accent color; clicking dismisses early. The store auto-expires
  // entries on its own timer so this component is purely view.

  import { toastState, dismiss, type ToastKind } from '../stores/toast.svelte';

  const kindStyle: Record<ToastKind, string> = {
    info:    'border-night-600 bg-night-800/90 text-slate-100',
    success: 'border-emerald-700 bg-emerald-900/85 text-emerald-100',
    warn:    'border-amber-700 bg-amber-900/85 text-amber-100',
    error:   'border-red-700 bg-red-900/85 text-red-100',
  };

  const kindIcon: Record<ToastKind, string> = {
    info: '◔', success: '✓', warn: '⚠', error: '✕',
  };
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-end gap-2 px-4">
  {#each toastState.items as t (t.id)}
    <button
      type="button"
      onclick={() => dismiss(t.id)}
      class="pointer-events-auto group flex max-w-md items-start gap-3 rounded-lg border px-4 py-3 text-left text-sm shadow-xl shadow-black/30 backdrop-blur transition hover:translate-y-[-2px] {kindStyle[t.kind]}"
      aria-label="Dismiss notification"
    >
      <span class="mt-0.5 text-base leading-none">{kindIcon[t.kind]}</span>
      <span class="flex flex-col">
        <span class="font-medium">{t.text}</span>
        {#if t.detail}
          <span class="mt-0.5 text-[11px] opacity-80">{t.detail}</span>
        {/if}
      </span>
      <span class="ml-2 text-xs opacity-50 group-hover:opacity-100">esc</span>
    </button>
  {/each}
</div>
