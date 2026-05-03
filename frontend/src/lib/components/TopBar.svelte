<script lang="ts">
  let { title, hint }: { title: string; hint?: string } = $props();

  let now = $state(new Date());
  let interval: ReturnType<typeof setInterval>;

  $effect(() => {
    interval = setInterval(() => { now = new Date(); }, 1000);
    return () => clearInterval(interval);
  });

  function pad(n: number) { return String(n).padStart(2, '0'); }
  const timeStr = $derived(`${pad(now.getUTCHours())}:${pad(now.getUTCMinutes())}:${pad(now.getUTCSeconds())} UTC`);
  const dateStr = $derived(now.toISOString().slice(0, 10));

  function openPalette(e: Event) {
    e.preventDefault();
    const ev = new KeyboardEvent('keydown', {
      key: 'k', ctrlKey: true, metaKey: true, bubbles: true,
    });
    window.dispatchEvent(ev);
  }
</script>

<header
  class="flex h-12 items-center justify-between border-b border-base-700 px-5"
  style="background: linear-gradient(90deg, rgba(11,16,23,0.98) 0%, rgba(11,16,23,0.92) 100%); backdrop-filter: blur(8px); flex-shrink: 0;"
>
  <!-- Left: breadcrumb + page title -->
  <div class="flex items-center gap-3" style="min-width:0;">
    <!-- Breadcrumb marker -->
    <span style="font-family:'Share Tech Mono',monospace; font-size:10px; color:var(--color-cyan-500); letter-spacing:1px; opacity:0.7; flex-shrink:0;">OBV://</span>

    <div class="flex items-baseline gap-2" style="min-width:0; overflow:hidden;">
      <h1 style="
        font-family: 'Rajdhani', sans-serif;
        font-weight: 700;
        font-size: 16px;
        letter-spacing: 2px;
        text-transform: uppercase;
        color: #e8f4f8;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      ">{title}</h1>
      {#if hint}
        <span style="font-family:'Share Tech Mono',monospace; font-size:10px; color:var(--color-base-300); letter-spacing:0.5px; white-space:nowrap;">{hint}</span>
      {/if}
    </div>
  </div>

  <!-- Right: search + UTC clock -->
  <div class="flex items-center gap-4" style="flex-shrink:0;">
    <!-- UTC Clock -->
    <div class="hidden sm:flex flex-col items-end" style="line-height:1.1;">
      <span style="font-family:'Share Tech Mono',monospace; font-size:12px; color:var(--color-cyan-500); letter-spacing:1px;">{timeStr}</span>
      <span style="font-family:'Share Tech Mono',monospace; font-size:9px; color:var(--color-base-300); letter-spacing:1px;">{dateStr}</span>
    </div>

    <!-- Divider -->
    <div class="hidden sm:block h-8 w-px" style="background:var(--color-base-600);"></div>

    <!-- Search / Command palette trigger -->
    <button
      type="button"
      onclick={openPalette}
      class="flex items-center gap-2 transition-all duration-150"
      style="
        padding: 5px 12px;
        border-radius: 2px;
        border: 1px solid var(--color-base-600);
        background: rgba(11,16,23,0.8);
        color: var(--color-base-300);
        font-family: 'Share Tech Mono', monospace;
        font-size: 11px;
        letter-spacing: 0.5px;
        width: 260px;
      "
      aria-label="Open command palette"
      onmouseenter={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-cyan-500)';
        (e.currentTarget as HTMLElement).style.color = '#e8f4f8';
        (e.currentTarget as HTMLElement).style.boxShadow = '0 0 10px rgba(0,188,216,0.15)';
      }}
      onmouseleave={(e) => {
        (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-base-600)';
        (e.currentTarget as HTMLElement).style.color = 'var(--color-base-300)';
        (e.currentTarget as HTMLElement).style.boxShadow = 'none';
      }}
    >
      <span style="font-size:12px;">⌕</span>
      <span style="flex:1; text-align:left;">Search events, hosts, rules…</span>
      <kbd style="
        padding: 1px 5px;
        border-radius: 2px;
        border: 1px solid var(--color-base-600);
        background: var(--color-base-800);
        font-family: 'Share Tech Mono', monospace;
        font-size: 9px;
        color: var(--color-base-300);
        letter-spacing: 1px;
      ">^K</kbd>
    </button>
  </div>
</header>
