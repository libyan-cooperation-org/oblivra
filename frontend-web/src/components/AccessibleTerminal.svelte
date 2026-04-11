<!-- OBLIVRA Web — AccessibleTerminal (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';

  interface TerminalLine { id: string; content: string; type: 'command' | 'output' | 'error'; }

  let lines = $state<TerminalLine[]>([]);
  let lastAnnouncement = $state('');

  onMount(() => {
    lines = [
      { id: '1', content: 'OBLIVRA-OS v0.5.0 // SECURE_SHELL_INITIALIZED', type: 'output' },
      { id: '2', content: 'substrate login --tenant GLOBAL_CORP', type: 'command' },
      { id: '3', content: 'Authentication successful. Access granted to 42 nodes.', type: 'output' },
    ];
  });

  function addLine(content: string, type: 'command' | 'output' | 'error') {
    lines = [...lines, { id: Math.random().toString(36).slice(2, 9), content, type }];
    lastAnnouncement = `${type.toUpperCase()}: ${content}`;
  }
</script>

<section class="at-wrap">
  <div class="at-header">
    <span class="at-title">Sovereign Control Substrate // ARIA_AUDIT_MODE</span>
    <span class="at-sub">TTY: /dev/pts/0 // SHELL: oblivra-sh</span>
  </div>

  <div class="at-log" role="log" aria-label="Accessible Terminal Output" aria-live="polite">
    {#each lines as line (line.id)}
      <div class="at-line at-line--{line.type}">
        {#if line.type === 'command'}<span class="at-prompt">❯ </span>{/if}{line.content}
      </div>
    {/each}
    <div class="at-cursor-row"><span class="at-prompt">❯ </span><span class="at-cursor"></span></div>
  </div>

  <div class="sr-only" role="status" aria-live="polite">{lastAnnouncement}</div>

  <div class="at-actions">
    <button onclick={() => addLine('oblivra-scan --target intranet', 'command')}>Run Scan</button>
    <button onclick={() => addLine('Scan completed. No threats found in /intranet/', 'output')}>Mock Output</button>
  </div>
</section>

<style>
  .at-wrap { font-family: var(--font-mono); display: flex; flex-direction: column; gap: 12px; }
  .at-header { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--border-bold, #1e3040); padding-bottom: 8px; }
  .at-title  { font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: .15em; color: var(--accent-primary); }
  .at-sub    { font-size: 9px; text-transform: uppercase; letter-spacing: .1em; color: var(--text-muted); }
  .at-log {
    background: rgba(0,0,0,0.8); border: 1px solid #1e3040; padding: 20px;
    height: 200px; overflow-y: auto; display: flex; flex-direction: column;
    justify-content: flex-end; gap: 6px;
  }
  .at-line { font-size: 11px; line-height: 1.5; }
  .at-line--command { color: var(--accent-primary); }
  .at-line--output  { color: #c8d8d8; }
  .at-line--error   { color: #ff3355; font-weight: 700; }
  .at-prompt { color: #607070; margin-right: 4px; }
  .at-cursor-row { display: flex; align-items: center; gap: 6px; font-size: 11px; }
  .at-cursor { width: 8px; height: 14px; background: var(--accent-primary); animation: blink 1s step-end infinite; }
  .at-actions { display: flex; gap: 12px; }
  .at-actions button {
    padding: 4px 12px; background: #1e3040; border: 1px solid #3a5060;
    color: #9b9ea4; font-size: 10px; font-weight: 700; text-transform: uppercase;
    letter-spacing: .08em; cursor: pointer; transition: color 100ms ease;
    font-family: var(--font-mono);
  }
  .at-actions button:hover { color: var(--accent-primary); }
  @keyframes blink { 0%,100% { opacity:1; } 50% { opacity:0; } }
</style>
