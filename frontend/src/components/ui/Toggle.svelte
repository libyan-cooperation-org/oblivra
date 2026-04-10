<!--
  OBLIVRA — Toggle (Svelte 5)
  Standard switch/toggle component.
-->
<script lang="ts">
  interface Props {
    checked: boolean;
    disabled?: boolean;
    label?: string;
  }

  let { checked = $bindable(), disabled = false, label }: Props = $props();

  function toggle() {
    if (!disabled) checked = !checked;
  }
</script>

<div class="flex items-center gap-2 select-none {disabled ? 'opacity-50 pointer-events-none' : ''}">
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-label={label}
    class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2
      {checked ? 'bg-accent' : 'bg-surface-3'}"
    onclick={toggle}
  >
    <span
      aria-hidden="true"
      class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out
        {checked ? 'translate-x-4' : 'translate-x-0'}"
    ></span>
  </button>
  {#if label}
    <span class="text-[11px] font-semibold text-text-secondary uppercase tracking-tight">{label}</span>
  {/if}
</div>
