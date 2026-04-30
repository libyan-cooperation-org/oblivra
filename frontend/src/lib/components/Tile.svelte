<script lang="ts">
  let {
    label,
    value,
    hint,
  }: { label: string; value: string | number; hint?: string } = $props();
</script>

<div class="rounded-xl border border-night-700 bg-night-900/70 p-4 shadow-sm">
  <div class="text-[11px] uppercase tracking-widest text-night-300">{label}</div>
  <div class="mt-1 font-mono text-2xl text-slate-100">{value}</div>
  {#if hint}
    <div class="mt-2 text-xs text-night-300">{hint}</div>
  {/if}
</div>
