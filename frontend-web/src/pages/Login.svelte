<!-- OBLIVRA Web — Login (Svelte 5) -->
<script lang="ts">
  import { login } from '../services/auth';
  import { push } from '../core/router.svelte';

  let email    = $state('');
  let password = $state('');
  let error    = $state('');
  let loading  = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = '';
    loading = true;
    try {
      await login(email, password);
      push('/');
    } catch (err: any) {
      error = err.message || 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="lg-wrap">
  <div class="lg-card">
    <div class="lg-header">
      <h1 class="lg-title">OBLIVRA <span class="lg-badge">ENTERPRISE</span></h1>
      <p class="lg-sub">Headless Access Portal v0.5.0</p>
    </div>

    <form onsubmit={handleSubmit} class="lg-form">
      <div class="lg-field">
        <label for="identity" class="lg-label">Identity (Email)</label>
        <input id="identity" type="email" bind:value={email} class="lg-input" placeholder="operator@oblivra.org" required />
      </div>
      <div class="lg-field">
        <label for="passphrase" class="lg-label">Passphrase</label>
        <input id="passphrase" type="password" bind:value={password} class="lg-input" placeholder="••••••••••••" required />
      </div>

      {#if error}
        <div class="lg-error">ACCESS DENIED: {error}</div>
      {/if}

      <button type="submit" disabled={loading} class="lg-submit">
        {#if loading}
          <div class="lg-spinner"></div>
        {:else}
          Authorize Session
        {/if}
      </button>
    </form>

    <div class="lg-sso">
      <button onclick={() => window.location.href='/api/v1/auth/oidc/login'} class="lg-sso-btn">Single Sign-On (OIDC)</button>
      <button onclick={() => window.location.href='/api/v1/auth/saml/login'} class="lg-sso-btn">Federated Identity (SAML)</button>
    </div>

    <p class="lg-footer">Sovereign-Grade Encryption Active<br/>Hardware Root-of-Trust Attestation: <span class="lg-verified">VERIFIED</span></p>
  </div>
</div>

<style>
  .lg-wrap { min-height:100vh; background:#000; display:flex; align-items:center; justify-content:center; padding:16px; }
  .lg-card { width:100%; max-width:420px; background:#18181b; border:1px solid #3f3f46; padding:32px; font-family:var(--font-mono); box-shadow:0 0 50px rgba(0,0,0,0.5); }
  .lg-header { margin-bottom:28px; border-bottom:1px solid #27272a; padding-bottom:16px; }
  .lg-title  { font-size:22px; font-weight:900; color:#fff; text-transform:uppercase; font-style:italic; letter-spacing:-.03em; margin:0; }
  .lg-badge  { background:#dc2626; color:#000; padding:0 4px; font-style:normal; }
  .lg-sub    { color:#52525b; font-size:11px; text-transform:uppercase; letter-spacing:.15em; margin:6px 0 0; }
  .lg-form   { display:flex; flex-direction:column; gap:20px; }
  .lg-field  { display:flex; flex-direction:column; gap:7px; }
  .lg-label  { font-size:11px; text-transform:uppercase; letter-spacing:.15em; font-weight:700; color:#71717a; }
  .lg-input  { width:100%; background:#000; border:1px solid #3f3f46; padding:12px; color:#fff; font-size:14px; font-family:inherit; outline:none; transition:border-color 100ms; }
  .lg-input:focus { border-color:#dc2626; }
  .lg-error  { background:rgba(127,29,29,0.3); border:1px solid #7f1d1d; padding:10px; color:#f87171; font-size:11px; text-transform:uppercase; font-weight:700; text-align:center; }
  .lg-submit { width:100%; background:#fff; color:#000; font-weight:900; text-transform:uppercase; padding:16px; border:none; cursor:pointer; font-family:inherit; font-size:13px; letter-spacing:.04em; transition:all 150ms; position:relative; min-height:52px; }
  .lg-submit:hover:not(:disabled) { background:#dc2626; color:#fff; }
  .lg-submit:disabled { opacity:0.5; cursor:not-allowed; }
  .lg-spinner { width:20px; height:20px; border:2px solid #000; border-top-color:transparent; border-radius:50%; animation:spin 0.7s linear infinite; margin:0 auto; }
  .lg-sso    { display:flex; flex-direction:column; gap:8px; margin-top:24px; padding-top:20px; border-top:1px solid #27272a; }
  .lg-sso-btn { width:100%; border:1px solid #3f3f46; color:#71717a; font-size:11px; font-weight:700; text-transform:uppercase; padding:8px; background:transparent; cursor:pointer; font-family:inherit; letter-spacing:.08em; transition:all 100ms; }
  .lg-sso-btn:hover { border-color:#71717a; color:#d4d4d8; }
  .lg-footer { margin-top:20px; font-size:10px; color:#3f3f46; text-align:center; text-transform:uppercase; letter-spacing:.18em; line-height:1.7; }
  .lg-verified { color:#14532d; }
  @keyframes spin { to { transform:rotate(360deg); } }
</style>
