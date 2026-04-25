<!--
  SetupWizard.svelte — Phase 22.5 first-run flow.

  Six-step claim in task.md:
    1. admin account
    2. TLS cert
    3. first log source
    4. alert channel
    5. detection pack selection
    6. first search tutorial

  This MVP ships steps 1, 4, 5, 6 — the ones that need real input from the
  operator. TLS cert and log source ingestion are deferred: TLS cert is
  best handled by the existing certgen CLI and is operator infrastructure
  rather than a UI flow; first log source has its own pages/Onboarding.svelte
  for fleet agent deployment which subsumes "first log source" once an agent
  is online.

  After completion this calls POST /api/v1/setup/initialize (already routed
  in rest.go) so a follow-up backend hook can wire admin creation + alert
  channel persistence. Today the endpoint is a stub; the wizard sends the
  payload it expects.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';
  import { push } from '../core/router.svelte';

  let step = $state(1);
  let submitting = $state(false);
  let submitError = $state<string | null>(null);

  // Step 1 — admin account
  let adminEmail = $state('');
  let adminPassword = $state('');
  let adminPasswordConfirm = $state('');

  // Step 4 — alert channel
  let channelType = $state<'none' | 'webhook' | 'email' | 'slack'>('none');
  let channelTarget = $state('');

  // Step 5 — detection pack
  let detectionPack = $state<'essential' | 'extended' | 'paranoid'>('extended');

  const TOTAL_STEPS = 4;

  // Frontend validation only — backend re-validates everything.
  const adminValid = $derived(
    adminEmail.includes('@') &&
    adminPassword.length >= 12 &&
    adminPassword === adminPasswordConfirm
  );

  const channelValid = $derived(
    channelType === 'none' ||
    (channelType === 'email' && channelTarget.includes('@')) ||
    (channelType === 'webhook' && channelTarget.startsWith('https://')) ||
    (channelType === 'slack' && channelTarget.startsWith('https://hooks.slack.com/'))
  );

  function next() {
    submitError = null;
    if (step < TOTAL_STEPS) step++;
  }

  function back() {
    submitError = null;
    if (step > 1) step--;
  }

  async function complete() {
    submitting = true;
    submitError = null;
    try {
      await request('/setup/initialize', {
        method: 'POST',
        body: JSON.stringify({
          admin: {
            email: adminEmail,
            password: adminPassword,
          },
          alert_channel: channelType === 'none' ? null : {
            type: channelType,
            target: channelTarget,
          },
          detection_pack: detectionPack,
        }),
      });
      push('/');
    } catch (err) {
      submitError = err instanceof Error ? err.message : 'Setup failed';
    } finally {
      submitting = false;
    }
  }

  onMount(() => {
    // If already configured (a token is present), skip setup.
    if (localStorage.getItem('oblivra_token')) {
      push('/');
    }
  });
</script>

<div class="wizard-shell">
  <header>
    <h1>Initial Setup</h1>
    <p class="subtitle">Step {step} of {TOTAL_STEPS}</p>
    <div class="progress" role="progressbar" aria-valuemin="0" aria-valuemax={TOTAL_STEPS} aria-valuenow={step}>
      {#each Array(TOTAL_STEPS) as _, i}
        <div class="seg" class:active={i < step}></div>
      {/each}
    </div>
  </header>

  <main>
    {#if step === 1}
      <h2>Administrator Account</h2>
      <p class="hint">This account has full platform access. Pick a passphrase you can remember without a password manager — this is the recovery seat.</p>

      <label>
        Email
        <input type="email" bind:value={adminEmail} autocomplete="email" placeholder="admin@company.local" />
      </label>
      <label>
        Passphrase (12+ characters)
        <input type="password" bind:value={adminPassword} autocomplete="new-password" />
      </label>
      <label>
        Confirm passphrase
        <input type="password" bind:value={adminPasswordConfirm} autocomplete="new-password" />
      </label>
      {#if adminPassword.length > 0 && adminPassword.length < 12}
        <p class="warn">Minimum 12 characters.</p>
      {/if}
      {#if adminPasswordConfirm.length > 0 && adminPassword !== adminPasswordConfirm}
        <p class="warn">Passphrases do not match.</p>
      {/if}
    {:else if step === 2}
      <h2>Alert Channel</h2>
      <p class="hint">Where should the platform send detection alerts? You can add more channels later.</p>

      <fieldset class="channel-options">
        <label class="opt">
          <input type="radio" bind:group={channelType} value="none" />
          <span class="opt-title">Skip for now</span>
          <span class="opt-desc">Alerts will be visible in the dashboard only.</span>
        </label>
        <label class="opt">
          <input type="radio" bind:group={channelType} value="email" />
          <span class="opt-title">Email</span>
          <span class="opt-desc">Send to a single distribution list. Requires SMTP config later.</span>
        </label>
        <label class="opt">
          <input type="radio" bind:group={channelType} value="webhook" />
          <span class="opt-title">Generic webhook</span>
          <span class="opt-desc">POST JSON to your own URL. HTTPS only.</span>
        </label>
        <label class="opt">
          <input type="radio" bind:group={channelType} value="slack" />
          <span class="opt-title">Slack incoming-webhook</span>
          <span class="opt-desc">Paste the hooks.slack.com URL from your Slack app.</span>
        </label>
      </fieldset>

      {#if channelType !== 'none'}
        <label>
          {channelType === 'email' ? 'Email address' : 'Webhook URL'}
          <input
            type={channelType === 'email' ? 'email' : 'url'}
            bind:value={channelTarget}
            placeholder={channelType === 'email' ? 'soc@company.local' : 'https://hooks.example.com/...'}
          />
        </label>
      {/if}
    {:else if step === 3}
      <h2>Detection Pack</h2>
      <p class="hint">Pick the rule pack you want enabled at boot. You can add or remove packs from the rule manager later.</p>

      <fieldset class="channel-options">
        <label class="opt">
          <input type="radio" bind:group={detectionPack} value="essential" />
          <span class="opt-title">Essential</span>
          <span class="opt-desc">~25 high-confidence rules covering the most common attacks. Lowest false-positive rate.</span>
        </label>
        <label class="opt">
          <input type="radio" bind:group={detectionPack} value="extended" />
          <span class="opt-title">Extended (recommended)</span>
          <span class="opt-desc">All 80+ built-in rules across the MITRE ATT&CK matrix. Balanced precision/recall.</span>
        </label>
        <label class="opt">
          <input type="radio" bind:group={detectionPack} value="paranoid" />
          <span class="opt-title">Paranoid</span>
          <span class="opt-desc">All built-in rules plus the SigmaHQ community pack. Higher false-positive rate, more coverage.</span>
        </label>
      </fieldset>
    {:else if step === 4}
      <h2>You're done</h2>
      <p class="hint">A quick orientation before you start:</p>

      <ul class="tutorial">
        <li>The <strong>Search</strong> page (left rail) accepts Lucene-style queries against ingested events. Try <code>EventType:auth_fail</code>.</li>
        <li>The <strong>Alerts</strong> page shows live detections. Acknowledge or assign with the keyboard shortcuts shown on hover.</li>
        <li>The <strong>Fleet</strong> page is empty until you deploy at least one agent — see the Onboarding page for one-line install commands.</li>
        <li>If the platform comes under load you'll see a banner at the top of every page (degraded / critical). Operators can dismiss it; it returns when state escalates.</li>
      </ul>

      {#if submitError}
        <p class="error">Setup failed: {submitError}. Please retry or check the server logs.</p>
      {/if}
    {/if}
  </main>

  <footer>
    {#if step > 1}
      <button class="secondary" onclick={back} disabled={submitting}>Back</button>
    {/if}
    {#if step < TOTAL_STEPS}
      <button
        class="primary"
        onclick={next}
        disabled={(step === 1 && !adminValid) || (step === 2 && !channelValid)}
      >
        Continue
      </button>
    {:else}
      <button class="primary" onclick={complete} disabled={submitting}>
        {submitting ? 'Initialising…' : 'Finish setup'}
      </button>
    {/if}
  </footer>
</div>

<style>
  .wizard-shell {
    max-width: 640px;
    margin: 4rem auto;
    padding: 2.5rem;
    background: var(--surface-1, #1a1c20);
    border: 1px solid var(--border-primary, #2b2d31);
    color: var(--text-primary, #d4d4d4);
    font-family: 'JetBrains Mono', monospace;
  }

  header h1 {
    font-size: 1.5rem;
    margin: 0 0 0.25rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .subtitle {
    color: var(--text-muted, #888);
    margin: 0 0 1.5rem;
    font-size: 0.75rem;
  }

  .progress {
    display: flex;
    gap: 4px;
    margin-bottom: 2rem;
  }

  .seg {
    flex: 1;
    height: 3px;
    background: var(--surface-2, #2b2d31);
    transition: background 200ms;
  }

  .seg.active {
    background: var(--accent-primary, #0099e0);
  }

  main h2 {
    font-size: 1.125rem;
    margin: 0 0 0.5rem;
  }

  .hint {
    color: var(--text-muted, #888);
    margin: 0 0 1.5rem;
    font-size: 0.8125rem;
    line-height: 1.5;
  }

  label {
    display: block;
    margin-bottom: 1rem;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  input[type='email'],
  input[type='password'],
  input[type='url'] {
    display: block;
    width: 100%;
    margin-top: 0.375rem;
    padding: 0.625rem 0.75rem;
    background: var(--surface-0, #0d0e10);
    border: 1px solid var(--border-primary, #2b2d31);
    color: var(--text-primary, #d4d4d4);
    font-family: inherit;
    font-size: 0.875rem;
    text-transform: none;
    letter-spacing: normal;
  }

  input:focus {
    outline: none;
    border-color: var(--accent-primary, #0099e0);
  }

  fieldset.channel-options {
    border: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.5rem;
  }

  .opt {
    display: grid;
    grid-template-columns: auto 1fr;
    grid-template-areas: 'radio title' 'radio desc';
    column-gap: 0.75rem;
    padding: 0.875rem;
    border: 1px solid var(--border-primary, #2b2d31);
    cursor: pointer;
    text-transform: none;
    letter-spacing: normal;
  }

  .opt:hover {
    border-color: var(--accent-primary, #0099e0);
  }

  .opt input {
    grid-area: radio;
    align-self: start;
    margin-top: 0.25rem;
  }

  .opt-title {
    grid-area: title;
    font-weight: 700;
    font-size: 0.875rem;
  }

  .opt-desc {
    grid-area: desc;
    font-size: 0.75rem;
    color: var(--text-muted, #888);
    margin-top: 0.25rem;
  }

  .tutorial {
    margin: 0 0 1rem;
    padding-left: 1.25rem;
    font-size: 0.8125rem;
    line-height: 1.6;
  }

  .tutorial code {
    padding: 0.125rem 0.375rem;
    background: var(--surface-0, #0d0e10);
    border: 1px solid var(--border-primary, #2b2d31);
    font-size: 0.75rem;
  }

  .warn {
    color: #f58b00;
    font-size: 0.75rem;
    margin: -0.5rem 0 1rem;
  }

  .error {
    color: #e04040;
    font-size: 0.8125rem;
    margin-top: 1rem;
  }

  footer {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
    margin-top: 2rem;
    padding-top: 1.5rem;
    border-top: 1px solid var(--border-primary, #2b2d31);
  }

  button {
    padding: 0.625rem 1.25rem;
    border: 1px solid var(--border-primary, #2b2d31);
    background: var(--surface-2, #2b2d31);
    color: var(--text-primary, #d4d4d4);
    font-family: inherit;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    cursor: pointer;
  }

  button.primary {
    background: var(--accent-primary, #f58b00);
    border-color: var(--accent-primary, #f58b00);
    color: #000;
  }

  button:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  button:hover:not(:disabled) {
    filter: brightness(1.15);
  }
</style>
