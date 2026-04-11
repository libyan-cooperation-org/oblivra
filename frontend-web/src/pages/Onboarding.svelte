<!-- OBLIVRA Web — Onboarding (Svelte 5) -->
<script lang="ts">
  import { push } from '../core/router.svelte';

  let step      = $state(1);
  let platform  = $state('linux');
  let collectors = $state<string[]>(['process', 'network']);

  const collectorOptions = [
    { id:'process',  label:'Process Execution',    desc:'Monitor process creation, termination, and suspicious activity.' },
    { id:'network',  label:'Network Connections',  desc:'Track inbound/outbound traffic and suspicious DNS queries.' },
    { id:'file',     label:'File Integrity (FIM)', desc:'Audit modifications to critical system files and configuration.' },
    { id:'registry', label:'Registry Audit',       desc:'Monitor persistence mechanisms in Windows registry.' },
    { id:'syslog',   label:'Syslog Ingest',         desc:'Forward local system logs to the OBLIVRA pipeline.' },
  ];

  function toggleCollector(id: string) {
    collectors = collectors.includes(id)
      ? collectors.filter(c => c !== id)
      : [...collectors, id];
  }

  const script = $derived.by(() => {
    const csv = collectors.join(',');
    const base = 'https://oblivra.enterprise.local:8443';
    if (platform === 'linux')   return `curl -sSL ${base}/scripts/install.sh | sudo bash -s -- --collectors ${csv}`;
    if (platform === 'windows') return `iex (iwr -UseBasicParsing ${base}/scripts/install.ps1).Content; Install-Oblivra -Collectors "${csv}"`;
    return `curl -sSL ${base}/scripts/install-mac.sh | bash -s -- --collectors ${csv}`;
  });

  function copyScript() {
    navigator.clipboard.writeText(script);
  }
</script>

<div class="ob-page">
  <div class="ob-card">
    <header class="ob-header">
      <div>
        <h1 class="ob-title">Fleet Onboarding</h1>
        <p class="ob-sub">Deployment Wizard v0.5.0</p>
      </div>
      <div class="ob-step-indicator">
        <div class="ob-step-label">Step</div>
        <div class="ob-step-val">{step}/3</div>
      </div>
    </header>

    {#if step === 1}
      <div class="ob-section">
        <h2 class="ob-section-title">01. Select Target Platform</h2>
        <div class="ob-platform-grid">
          {#each ['linux','windows','darwin'] as p}
            <button class="ob-platform-btn {platform === p ? 'ob-platform-btn--active' : ''}" onclick={() => platform = p}>
              <span class="ob-platform-icon">{p==='linux' ? '🐧' : p==='windows' ? '🪟' : '🍎'}</span>
              {p}
            </button>
          {/each}
        </div>
        <div class="ob-footer-row">
          <button class="ob-btn-primary" onclick={() => step = 2}>Continue to Collectors</button>
        </div>
      </div>

    {:else if step === 2}
      <div class="ob-section">
        <h2 class="ob-section-title">02. Configure Collectors</h2>
        <div class="ob-collectors">
          {#each collectorOptions as c}
            <div class="ob-collector {collectors.includes(c.id) ? 'ob-collector--active' : ''}" onclick={() => toggleCollector(c.id)} role="checkbox" aria-checked={collectors.includes(c.id)} tabindex="0" onkeydown={(e) => e.key===' ' && toggleCollector(c.id)}>
              <div>
                <div class="ob-collector-label {collectors.includes(c.id) ? 'ob-collector-label--active' : ''}">{c.label}</div>
                <div class="ob-collector-desc">{c.desc}</div>
              </div>
              <div class="ob-checkbox {collectors.includes(c.id) ? 'ob-checkbox--active' : ''}">✓</div>
            </div>
          {/each}
        </div>
        <div class="ob-footer-row ob-footer-row--between">
          <button class="ob-btn-back" onclick={() => step = 1}>Go Back</button>
          <button class="ob-btn-primary" onclick={() => step = 3}>Generate Deployment Script</button>
        </div>
      </div>

    {:else}
      <div class="ob-section">
        <h2 class="ob-section-title">03. Finalize Deployment</h2>
        <div class="ob-script-block">
          <div class="ob-script-meta">
            <span>Deployment One-Liner</span>
            <span class="ob-priv-warning">Privileged Execution Required</span>
          </div>
          <code class="ob-script-code">{script}</code>
          <button class="ob-copy-btn" onclick={copyScript}>Copy to Clipboard</button>
        </div>
        <div class="ob-checklist">
          <span class="ob-checklist-title">Pre-Flight Checklist:</span><br/>
          1. Ensure port 8443 (Ingest) and 8080 (REST) are accessible from the target host.<br/>
          2. Validate minimum hardware requirements (2 vCPU, 4GB RAM).<br/>
          3. Run the script with root/administrator privileges.
        </div>
        <div class="ob-footer-row ob-footer-row--between">
          <button class="ob-btn-back" onclick={() => step = 2}>Adjust Config</button>
          <button class="ob-btn-outline" onclick={() => push('/')}>Complete Onboarding</button>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .ob-page  { min-height:100vh; background:#000; color:#fff; font-family:var(--font-mono); padding:32px 16px; }
  .ob-card  { max-width:860px; margin:0 auto; background:#09090b; border:1px solid #27272a; padding:48px; box-shadow:0 24px 60px rgba(0,0,0,0.5); }
  .ob-header { display:flex; justify-content:space-between; align-items:flex-end; margin-bottom:48px; border-bottom:1px solid #27272a; padding-bottom:28px; }
  .ob-title { font-size:36px; font-weight:900; text-transform:uppercase; font-style:italic; letter-spacing:-.04em; margin:0; }
  .ob-sub   { color:#52525b; text-transform:uppercase; letter-spacing:.15em; font-size:13px; margin:6px 0 0; }
  .ob-step-indicator { text-align:right; }
  .ob-step-label { font-size:10px; color:#52525b; text-transform:uppercase; letter-spacing:.12em; }
  .ob-step-val   { font-size:24px; font-weight:900; color:#dc2626; }
  .ob-section { display:flex; flex-direction:column; gap:28px; }
  .ob-section-title { font-size:18px; font-weight:700; text-transform:uppercase; letter-spacing:.04em; border-left:4px solid #dc2626; padding-left:14px; margin:0; }
  .ob-platform-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:20px; }
  .ob-platform-btn { padding:28px 16px; border:2px solid #27272a; background:#000; color:#52525b; font-weight:900; text-transform:uppercase; font-size:14px; cursor:pointer; display:flex; flex-direction:column; align-items:center; gap:12px; transition:all 150ms; font-family:inherit; }
  .ob-platform-btn--active { border-color:#dc2626; background:rgba(220,38,38,0.08); color:#fff; }
  .ob-platform-btn:hover:not(.ob-platform-btn--active) { border-color:#52525b; color:#d4d4d8; }
  .ob-platform-icon { font-size:36px; }
  .ob-collectors { display:flex; flex-direction:column; gap:10px; }
  .ob-collector { display:flex; justify-content:space-between; align-items:center; padding:20px 22px; border:1px solid #27272a; background:#000; cursor:pointer; transition:border-color 100ms; }
  .ob-collector--active { border-color:#dc2626; background:rgba(220,38,38,0.04); }
  .ob-collector:hover:not(.ob-collector--active) { border-color:#3f3f46; }
  .ob-collector-label { font-weight:900; text-transform:uppercase; font-size:13px; color:#52525b; }
  .ob-collector-label--active { color:#fff; }
  .ob-collector-desc { font-size:11px; color:#3f3f46; text-transform:uppercase; letter-spacing:.04em; margin-top:4px; }
  .ob-checkbox { width:22px; height:22px; border:2px solid #27272a; display:flex; align-items:center; justify-content:center; font-size:13px; color:transparent; flex-shrink:0; }
  .ob-checkbox--active { border-color:#dc2626; background:#dc2626; color:#000; }
  .ob-script-block { background:#000; border:1px solid #27272a; padding:24px; position:relative; }
  .ob-script-meta  { display:flex; justify-content:space-between; font-size:10px; color:#52525b; text-transform:uppercase; letter-spacing:.12em; margin-bottom:14px; }
  .ob-priv-warning { color:#7f1d1d; font-weight:700; }
  .ob-script-code  { font-size:13px; color:#f87171; word-break:break-all; display:block; line-height:1.65; }
  .ob-copy-btn     { position:absolute; right:12px; bottom:10px; background:#27272a; border:1px solid #3f3f46; color:#71717a; font-size:10px; font-weight:700; text-transform:uppercase; padding:4px 12px; cursor:pointer; font-family:inherit; transition:all 100ms; }
  .ob-copy-btn:hover { background:#3f3f46; color:#fff; }
  .ob-checklist { background:rgba(39,39,42,0.5); border-left:4px solid #3f3f46; padding:18px 20px; font-size:11px; color:#71717a; line-height:1.8; text-transform:uppercase; }
  .ob-checklist-title { color:#fff; font-weight:700; text-decoration:underline; letter-spacing:.15em; }
  .ob-footer-row { display:flex; justify-content:flex-end; padding-top:24px; }
  .ob-footer-row--between { justify-content:space-between; }
  .ob-btn-primary { background:#fff; color:#000; font-weight:900; text-transform:uppercase; padding:16px 40px; border:none; cursor:pointer; font-family:inherit; font-size:13px; transition:all 150ms; }
  .ob-btn-primary:hover { background:#dc2626; color:#fff; }
  .ob-btn-back    { background:transparent; border:none; color:#52525b; font-weight:700; text-transform:uppercase; font-size:12px; text-decoration:underline; text-underline-offset:6px; cursor:pointer; font-family:inherit; padding:0; }
  .ob-btn-back:hover { color:#fff; }
  .ob-btn-outline { border:2px solid #fff; background:transparent; color:#fff; font-weight:900; text-transform:uppercase; padding:16px 40px; cursor:pointer; font-family:inherit; font-size:13px; transition:all 150ms; }
  .ob-btn-outline:hover { background:#fff; color:#000; }
</style>
