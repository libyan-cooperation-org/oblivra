<!--
  KeysPanel — view of stored credentials / SSH keys.
  OBLIVRA's vault doesn't expose a clean Wails-bound key listing yet, so
  this panel is intentionally minimal: shows configured-host count + a
  "Open Vault" link. Stage-3 polish can wire CredentialIntel for a fuller view.
-->
<script lang="ts">
  import { KeyRound, ExternalLink, ShieldCheck } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { push } from '@lib/router.svelte';
</script>

<div class="flex h-full flex-col bg-[var(--s0)] text-[var(--tx)]">
  <header class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
    <KeyRound size={14} class="text-[var(--tx3)]" />
    <span class="text-xs font-semibold uppercase tracking-wider">Keys & Credentials</span>
  </header>

  <div class="grid grid-cols-1 gap-3 p-4 md:grid-cols-3">
    <div class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Saved hosts</div>
      <div class="mt-1 text-2xl font-semibold">{shellStore.hosts.length}</div>
      <div class="mt-1 text-[10px] text-[var(--tx3)]">
        {shellStore.hosts.filter((h) => h.authMethod === 'key').length} with key auth
        ·
        {shellStore.hosts.filter((h) => h.authMethod === 'password').length} password
      </div>
    </div>

    <div class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Vault</div>
      <div class="mt-1 flex items-center gap-2">
        <ShieldCheck size={16} class="text-emerald-400" />
        <span class="text-sm">Encrypted at rest</span>
      </div>
      <button
        class="mt-3 flex items-center gap-1.5 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-2.5 py-1 text-[11px] text-cyan-200 hover:bg-cyan-400/20"
        onclick={() => push('/vault')}
      >
        Open vault
        <ExternalLink size={10} />
      </button>
    </div>

    <div class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-4">
      <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Per-credential intel</div>
      <div class="mt-1 text-sm">Reuse, age, last seen, exposure</div>
      <button
        class="mt-3 flex items-center gap-1.5 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2.5 py-1 text-[11px] text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={() => push('/credentials')}
      >
        Credential intel
        <ExternalLink size={10} />
      </button>
    </div>
  </div>

  <div class="px-4 pb-4 text-[11px] text-[var(--tx3)]">
    <p class="mb-1">
      Stage 4 will inline a key generator and per-key deploy actions
      (<code class="rounded bg-[var(--s2)] px-1 py-0.5">SSHService.DeployKey</code> already exists on the backend).
    </p>
  </div>
</div>
