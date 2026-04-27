<!--
  SettingsPanel — shell-workspace specific preferences. Most app-wide
  settings live on /workspace; this panel surfaces the bits that affect
  the shell session experience directly.
-->
<script lang="ts">
  import { Settings as SettingsIcon, MoonStar, Sun, ExternalLink } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { push } from '@lib/router.svelte';

  function setTheme(t: 'dark' | 'light') {
    shellStore.theme = t;
  }
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <SettingsIcon size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Shell Settings</span>
  </header>

  <div class="space-y-4 p-4">
    <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Terminal theme</div>
      <div class="mt-2 flex items-center gap-2">
        <button
          class="flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs {shellStore.theme === 'dark'
            ? 'border-cyan-400/40 bg-cyan-400/10 text-cyan-200'
            : 'border-[var(--b1)] bg-[var(--s2)] text-[var(--tx2)] hover:bg-[var(--s3)]'}"
          onclick={() => setTheme('dark')}
        >
          <MoonStar size={12} />
          Dark
        </button>
        <button
          class="flex items-center gap-2 rounded-md border px-3 py-1.5 text-xs {shellStore.theme === 'light'
            ? 'border-cyan-400/40 bg-cyan-400/10 text-cyan-200'
            : 'border-[var(--b1)] bg-[var(--s2)] text-[var(--tx2)] hover:bg-[var(--s3)]'}"
          onclick={() => setTheme('light')}
        >
          <Sun size={12} />
          Light
        </button>
      </div>
      <div class="mt-2 text-[10px] text-[var(--tx3)]">
        Applies to newly-spawned panes. Existing panes keep the theme they spawned with.
      </div>
    </section>

    <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Recording</div>
      <div class="mt-2 flex items-center justify-between">
        <div class="flex flex-col">
          <span class="text-sm">Record new sessions</span>
          <span class="text-[10px] text-[var(--tx3)]">{shellStore.recordingsEnabled ? 'On — every PTY/SSH session is captured' : 'Off — sessions are NOT recorded'}</span>
        </div>
        <button
          class="rounded-md border px-3 py-1.5 text-xs {shellStore.recordingsEnabled
            ? 'border-emerald-400/40 bg-emerald-400/10 text-emerald-200'
            : 'border-[var(--b1)] bg-[var(--s2)] text-[var(--tx2)] hover:bg-[var(--s3)]'}"
          onclick={() => (shellStore.recordingsEnabled = !shellStore.recordingsEnabled)}
        >
          {shellStore.recordingsEnabled ? 'On' : 'Off'}
        </button>
      </div>
      <div class="mt-2 text-[10px] text-[var(--tx3)]">
        Status mirrored from <code class="rounded bg-[var(--s2)] px-1">RecordingService</code>.
        Persistence wires up in Stage 4.
      </div>
    </section>

    <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">App-wide settings</div>
          <div class="mt-1 text-sm">Tenant, fleet auth, OIDC, integrations, MFA…</div>
        </div>
        <button
          class="flex items-center gap-1.5 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2.5 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
          onclick={() => push('/workspace')}
        >
          Open Settings
          <ExternalLink size={10} />
        </button>
      </div>
    </section>
  </div>
</div>
