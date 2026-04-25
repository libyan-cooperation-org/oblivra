<!--
  LoadingSkeleton — placeholder rows that match the shape of a list /
  table while data is in flight. Gives the operator visual continuity
  instead of a flash of nothing followed by content jumping into place.

  Usage:
    <LoadingSkeleton rows={5} />            // default row layout
    <LoadingSkeleton rows={3} columns={4} />// table-shaped grid
    <LoadingSkeleton variant="card" />      // card-block placeholder
-->
<script lang="ts">
  interface Props {
    /** Number of rows to render. Default 4. */
    rows?: number;
    /** Number of columns when variant="row". Each column animates separately. */
    columns?: number;
    /** Layout variant. */
    variant?: 'row' | 'card' | 'block';
    /** Optional CSS class for the outer wrapper. */
    class?: string;
  }

  let {
    rows = 4,
    columns = 1,
    variant = 'row',
    class: cls = '',
  }: Props = $props();
</script>

<div class="skel {cls}" aria-busy="true" aria-live="polite">
  {#if variant === 'row'}
    {#each Array(rows) as _, r (r)}
      <div class="skel-row" role="presentation">
        {#each Array(columns) as _, c (c)}
          <div class="skel-bar" style:--w="{c === columns - 1 ? '40%' : '100%'}"></div>
        {/each}
      </div>
    {/each}
  {:else if variant === 'card'}
    {#each Array(rows) as _, r (r)}
      <div class="skel-card" role="presentation">
        <div class="skel-bar" style:--w="60%"></div>
        <div class="skel-bar" style:--w="90%"></div>
        <div class="skel-bar" style:--w="40%"></div>
      </div>
    {/each}
  {:else}
    {#each Array(rows) as _, r (r)}
      <div class="skel-bar" style:--w="{(60 + ((r * 13) % 30)).toFixed(0)}%"></div>
    {/each}
  {/if}
</div>

<style>
  .skel {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.5rem 0;
  }

  .skel-row {
    display: flex;
    gap: 0.75rem;
    align-items: center;
  }

  .skel-card {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem;
    border: 1px solid var(--border-primary, #2b2d31);
    background: var(--surface-1, #1a1c20);
    border-radius: 4px;
  }

  .skel-bar {
    flex: 1;
    height: 0.625rem;
    width: var(--w, 100%);
    background: linear-gradient(
      90deg,
      var(--surface-2, #2b2d31) 0%,
      var(--surface-3, #353740) 50%,
      var(--surface-2, #2b2d31) 100%
    );
    background-size: 200% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
    border-radius: 2px;
  }

  @keyframes shimmer {
    0%   { background-position: 100% 0; }
    100% { background-position: -100% 0; }
  }

  /* Honour reduced-motion preferences — degrade to a static dim bar. */
  @media (prefers-reduced-motion: reduce) {
    .skel-bar {
      animation: none;
      opacity: 0.5;
    }
  }
</style>
