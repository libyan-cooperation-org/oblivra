<!--
  LogsPanel — system log tail on a selected host.
  Polls `tail -n 200 <path>` every 5s. Path is a small dropdown of common
  log files; future Stage-5 work will switch to streaming `tail -F`
  via a long-lived SSH session.
-->
<script lang="ts">
  import { ScrollText } from 'lucide-svelte';
  import RemoteExecPanel from './RemoteExecPanel.svelte';

  const LOG_PATHS = [
    { value: '/var/log/syslog',          label: 'syslog' },
    { value: '/var/log/messages',        label: 'messages' },
    { value: '/var/log/auth.log',        label: 'auth.log' },
    { value: '/var/log/secure',          label: 'secure (RHEL)' },
    { value: '/var/log/kern.log',        label: 'kern.log' },
    { value: '/var/log/nginx/access.log', label: 'nginx access' },
    { value: '/var/log/nginx/error.log',  label: 'nginx error' },
  ];
  let path = $state(LOG_PATHS[0].value);
  let lines = $state(200);

  // `journalctl -n` covers systemd boxes; we wrap in a fallback chain
  // so the panel works on both file-log and journald-only systems.
  let cmd = $derived(`(test -r '${path}' && tail -n ${lines} '${path}') || (command -v journalctl >/dev/null && journalctl -n ${lines} --no-pager) || echo "no readable log at ${path} and journalctl unavailable"`);
</script>

<RemoteExecPanel title="Logs" icon={ScrollText} command={cmd} pollIntervalMs={5000}>
  {#snippet controls({ refresh: _refresh })}
    <select
      class="rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] outline-none"
      bind:value={path}
    >
      {#each LOG_PATHS as p}
        <option value={p.value}>{p.label}</option>
      {/each}
    </select>
    <input
      type="number"
      min="20"
      max="2000"
      step="20"
      class="w-16 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-2 py-1 text-[11px] outline-none"
      bind:value={lines}
      title="Lines"
    />
  {/snippet}
  {#snippet children({ output })}
    <pre class="whitespace-pre-wrap p-3 font-mono text-[10px] leading-relaxed text-[var(--tx2)]">{output || '(no output)'}</pre>
  {/snippet}
</RemoteExecPanel>
