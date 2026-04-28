<!--
  OBLIVRA — Shell Alert Rail (Phase 32, the moat-widener).

  A right-rail strip mounted inside the Shell Workspace. Auto-scopes
  to whichever host the active terminal session is connected to. On
  Ctrl+Click of any alert, injects a comment block into the active
  terminal with a suggested investigative command — completing the
  bidirectional fusion that no incumbent tool ships:

    ┌── alert ──────────────┐
    │ HIGH  02:14           │   ← Ctrl+Click
    │  Process injection    │
    │  pid 4242             │
    └───────────────────────┘
                ▼
    # ALERT 02:14 — process injection — try: lsof -p 4242
    └── injected as a comment in the active xterm

  The operator presses Enter and the suggested command runs. If a
  recommendation isn't available we still inject the alert summary
  so the operator has the context inline with their other commands.
-->
<script lang="ts">
  import { alertStore, type Alert } from '@lib/stores/alerts.svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { Bell, Filter } from 'lucide-svelte';
  import { Badge } from '@components/ui';

  /**
   * Map common alert categories to a one-liner Linux/Windows command
   * the operator usually wants to run as the first investigative step.
   * The keys are case-insensitive substrings of `category` or `title`.
   */
  const SUGGESTIONS: Array<{ match: RegExp; cmd: string; rationale: string }> = [
    { match: /process injection|cmd injection/i, cmd: 'lsof -p {PID}',                    rationale: 'list open files for the suspect pid' },
    { match: /outbound|c2|beacon/i,              cmd: 'ss -tnp | grep ESTAB',             rationale: 'show established outbound connections' },
    { match: /persistence|cron|scheduled/i,      cmd: 'systemctl list-timers --all',      rationale: 'enumerate scheduled tasks / cron-equivalents' },
    { match: /privilege|sudo|setuid/i,           cmd: 'find / -perm -4000 2>/dev/null',   rationale: 'find SUID binaries' },
    { match: /credential|password|hash dump/i,   cmd: 'last -50 | head -20',              rationale: 'recent logins' },
    { match: /file write|file change/i,          cmd: 'find / -mtime -1 -type f 2>/dev/null | head -50', rationale: 'recently-modified files' },
    { match: /exfil|transfer/i,                  cmd: 'ss -tnp | awk \'$5 !~ /127\\.0\\.0\\.1/\'', rationale: 'non-local connections' },
    { match: /memory|inject/i,                   cmd: 'cat /proc/{PID}/maps | head',      rationale: 'inspect the process memory map' },
  ];

  /**
   * Build the comment block injected into the terminal. Conservative —
   * the actual command is COMMENTED, never executed. Operator presses
   * Enter to run if they want.
   */
  function buildInjection(a: Alert): string {
    const t = (a.timestamp ?? '').slice(11, 19) || '??:??:??';
    const sev = (a.severity ?? '?').toUpperCase();

    // Try to extract a pid from the description (very common in alert payloads).
    const pidMatch = (a.description ?? '').match(/\bpid[=:\s]+(\d{2,7})\b/i);
    const pid = pidMatch?.[1] ?? '?';

    // Find the first matching suggestion. Alert type doesn't carry a
    // formal `category` field so we fall back to scanning title +
    // description; the detection engine puts MITRE / category hints in
    // the description blob anyway.
    const haystack = `${a.title ?? ''} ${(a as any).category ?? ''} ${a.description ?? ''}`;
    let suggestion: { cmd: string; rationale: string } | null = null;
    for (const s of SUGGESTIONS) {
      if (s.match.test(haystack)) {
        suggestion = { cmd: s.cmd.replace('{PID}', pid), rationale: s.rationale };
        break;
      }
    }

    const lines: string[] = [];
    lines.push(`# ──────── OBLIVRA alert ${a.id} (${sev} ${t}) ────────`);
    lines.push(`# ${a.title}`);
    if (a.host) lines.push(`# host: ${a.host}`);
    if (suggestion) {
      lines.push(`# suggest: ${suggestion.cmd}    # ${suggestion.rationale}`);
      lines.push(suggestion.cmd);
    } else {
      lines.push(`# (no automatic command suggestion — investigate manually)`);
    }
    return lines.join('\r\n') + '\r\n';
  }

  function inject(a: Alert) {
    const sessionID = shellStore.activeSessionID;
    if (!sessionID) {
      // Nothing to inject into — no active terminal. Bail silently;
      // the rail still expands details on a normal click below.
      return;
    }
    // Use the public helper — it stamps a uuid so concurrent injects
    // are distinguishable on the Terminal.svelte $effect side.
    shellStore.insertIntoTerminal(sessionID, buildInjection(a));
  }

  function onAlertClick(a: Alert, e: MouseEvent) {
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      inject(a);
    }
    // A regular click is a no-op here; operator can use the broader
    // /alerts page if they want full triage. The rail is intentionally
    // a passive "what's happening on this host" pane.
  }

  // Identify the host the active terminal is bound to so we can scope
  // alerts. shellStore.leafMeta holds 'remote' entries with title
  // 'username@host' — we extract the host portion. Local PTY sessions
  // have no host correspondence, so the rail shows nothing.
  let activeHost = $derived.by(() => {
    const tab = shellStore.tabs.find((t) => t.id === shellStore.activeTabID);
    if (!tab || !tab.activeLeafID) return null;
    const meta = tab.leafMeta[tab.activeLeafID];
    if (!meta || meta.kind !== 'remote') return null;
    const at = meta.title.indexOf('@');
    return at > 0 ? meta.title.slice(at + 1) : meta.title;
  });

  let scopedAlerts = $derived.by(() => {
    if (!activeHost) return [] as Alert[];
    return alertStore.alerts
      .filter((a) => {
        if (a.status === 'closed' || a.status === 'suppressed') return false;
        return a.host === activeHost ||
               a.host?.toLowerCase() === activeHost.toLowerCase();
      })
      .sort((a, b) => String(b.timestamp).localeCompare(String(a.timestamp)))
      .slice(0, 30);
  });

  let collapsed = $state(false);
</script>

<aside
  class="flex flex-col bg-surface-1 border-l border-border-primary overflow-hidden"
  style="width: {collapsed ? '32px' : '280px'}; transition: width 120ms ease;"
  aria-label="Shell alert rail"
>
  <header class="flex items-center justify-between px-3 py-2 border-b border-border-primary shrink-0">
    {#if !collapsed}
      <div class="flex items-center gap-1.5 min-w-0">
        <Bell size={11} class="text-accent shrink-0" />
        <span class="text-[var(--fs-micro)] uppercase tracking-widest font-bold text-text-muted">
          Alerts
        </span>
        {#if activeHost}
          <span class="font-mono text-[var(--fs-micro)] text-text-secondary truncate" title={activeHost}>· {activeHost}</span>
        {/if}
      </div>
    {/if}
    <button
      class="text-text-muted hover:text-text-primary p-1 rounded-sm hover:bg-surface-2 shrink-0"
      onclick={() => (collapsed = !collapsed)}
      aria-label={collapsed ? 'Expand alert rail' : 'Collapse alert rail'}
      title={collapsed ? 'Expand' : 'Collapse'}
    >{collapsed ? '◂' : '▸'}</button>
  </header>

  {#if !collapsed}
    {#if !activeHost}
      <div class="flex-1 flex flex-col items-center justify-center gap-2 p-4 text-center">
        <Filter size={20} class="text-text-muted opacity-40" />
        <p class="text-[var(--fs-label)] text-text-muted leading-relaxed">
          Connect to a remote host — alerts on that host appear here. Local PTY sessions don't have a host context.
        </p>
      </div>
    {:else if scopedAlerts.length === 0}
      <div class="flex-1 flex flex-col items-center justify-center gap-2 p-4 text-center">
        <Bell size={18} class="text-text-muted opacity-40" />
        <p class="text-[var(--fs-label)] text-text-muted">No open alerts on <span class="font-mono">{activeHost}</span>.</p>
      </div>
    {:else}
      <ul class="flex-1 overflow-auto py-1">
        {#each scopedAlerts as a (a.id)}
          <li>
            <button
              class="w-full text-start px-3 py-2 hover:bg-surface-2 border-b border-border-primary/40 transition-colors duration-fast group"
              onclick={(e) => onAlertClick(a, e)}
              title="Ctrl+Click to inject investigation command into the active terminal"
            >
              <div class="flex items-center gap-1.5 mb-0.5">
                <Badge
                  variant={a.severity === 'critical' ? 'critical' : a.severity === 'high' ? 'warning' : 'info'}
                  size="xs"
                >{a.severity.toUpperCase()}</Badge>
                <span class="font-mono text-[var(--fs-micro)] text-text-muted">{(a.timestamp ?? '').slice(11, 19)}</span>
              </div>
              <div class="text-[var(--fs-label)] text-text-secondary leading-snug line-clamp-2">{a.title}</div>
              <div class="text-[var(--fs-micro)] font-mono text-accent opacity-0 group-hover:opacity-100 transition-opacity duration-fast mt-1">
                Ctrl+Click → inject
              </div>
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    <footer class="px-3 py-1.5 border-t border-border-primary shrink-0 text-[var(--fs-micro)] font-mono text-text-muted">
      {scopedAlerts.length} open · scoped to active session
    </footer>
  {/if}
</aside>
