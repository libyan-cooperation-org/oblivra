<script lang="ts">
  let { name }: { name: string } = $props();
</script>

<div class="mx-auto flex max-w-3xl flex-col items-center justify-center p-16 text-center">
  <div class="mb-4 grid h-16 w-16 place-items-center rounded-2xl bg-night-800 text-3xl text-night-300">
    ◌
  </div>
  <h2 class="text-xl font-semibold text-slate-100">{name}</h2>
  <p class="mt-2 max-w-md text-sm text-night-300">
    This surface is part of the OBLIVRA roadmap but isn't wired up yet. The skeleton is live —
    real services will plug in during the next phases.
  </p>
</div>
