<!-- OBLIVRA Web — Login (Svelte 5) -->
<script lang="ts">
  import { login } from '../services/auth';
  import { push } from '../core/router.svelte';
  import { Button, Spinner } from '@components/ui';
  import { Shield, Lock, Key, Globe, ShieldCheck } from 'lucide-svelte';

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

<div class="min-h-screen bg-surface-0 flex items-center justify-center p-6 font-mono selection:bg-accent-primary selection:text-black relative overflow-hidden">
  <!-- Dynamic Background Shards -->
  <div class="absolute inset-0 overflow-hidden pointer-events-none opacity-20">
    <div class="absolute -top-1/4 -left-1/4 w-full h-full bg-accent-primary/10 rounded-full blur-[120px]"></div>
    <div class="absolute -bottom-1/4 -right-1/4 w-full h-full bg-accent-primary/5 rounded-full blur-[120px]"></div>
  </div>

  <div class="w-full max-w-md bg-surface-1 border border-border-primary p-10 shadow-premium relative z-10 backdrop-blur-sm">
    <div class="text-center mb-10">
      <div class="inline-flex p-4 bg-surface-2 border border-border-primary rounded-sm mb-6 relative group cursor-pointer overflow-hidden">
        <div class="absolute inset-0 bg-accent-primary/5 group-hover:bg-accent-primary/10 transition-colors"></div>
        <Shield size={32} class="text-accent-primary group-hover:scale-110 transition-transform duration-500 relative z-10" />
        <div class="absolute -inset-2 bg-accent-primary/20 blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-700 animate-pulse"></div>
        
        <!-- Scanning line effect -->
        <div class="absolute top-0 left-0 w-full h-[1px] bg-accent-primary/40 -translate-y-4 group-hover:animate-scan"></div>
      </div>
      <h1 class="text-3xl font-black italic uppercase tracking-tighter text-text-heading">
        OBL<em>IV</em>RA <span class="bg-accent-primary text-black px-2 not-italic">ORBIT</span>
      </h1>
      <div class="flex items-center justify-center gap-4 mt-3">
         <div class="flex items-center gap-1.5">
            <div class="w-1.5 h-1.5 rounded-full bg-status-online animate-pulse"></div>
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Auth_Mesh: LIVE</span>
         </div>
         <div class="flex items-center gap-1.5">
            <div class="w-1.5 h-1.5 rounded-full bg-status-online"></div>
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Substrate: NOMINAL</span>
         </div>
      </div>
    </div>

    <form onsubmit={handleSubmit} class="space-y-6">
      <div class="space-y-2">
        <label for="identity" class="text-[10px] font-black text-text-muted uppercase tracking-widest flex items-center gap-2">
          <Globe size={12} class="text-accent-primary/60" />
          Operator Identity
        </label>
        <div class="relative">
          <input 
            id="identity" 
            type="email" 
            bind:value={email} 
            class="w-full bg-surface-2 border border-border-primary p-4 text-sm text-text-secondary outline-none focus:border-accent-primary transition-colors pl-12" 
            placeholder="operator@oblivra.org" 
            required 
          />
          <div class="absolute left-4 top-1/2 -translate-y-1/2 text-text-muted">
            @
          </div>
        </div>
      </div>

      <div class="space-y-2">
        <label for="passphrase" class="text-[10px] font-black text-text-muted uppercase tracking-widest flex items-center gap-2">
          <Lock size={12} class="text-accent-primary/60" />
          Neural Passphrase
        </label>
        <div class="relative">
          <input 
            id="passphrase" 
            type="password" 
            bind:value={password} 
            class="w-full bg-surface-2 border border-border-primary p-4 text-sm text-text-secondary outline-none focus:border-accent-primary transition-colors pl-12" 
            placeholder="••••••••••••" 
            required 
          />
          <div class="absolute left-4 top-1/2 -translate-y-1/2 text-text-muted">
            <Key size={14} />
          </div>
        </div>
      </div>

      {#if error}
        <div class="p-4 bg-alert-critical/10 border border-alert-critical text-alert-critical text-[10px] font-black uppercase tracking-widest text-center animate-pulse">
          ACCESS_DENIED: {error}
        </div>
      {/if}

      <Button type="submit" disabled={loading} variant="primary" class="w-full py-6 font-black italic tracking-tighter text-lg relative overflow-hidden group">
        {#if loading}
          <Spinner />
        {:else}
          <span class="relative z-10">INITIALIZE_SESSION</span>
          <div class="absolute inset-0 bg-white/10 translate-y-full group-hover:translate-y-0 transition-transform"></div>
        {/if}
      </Button>
    </form>

    <div class="mt-10 pt-8 border-t border-border-primary space-y-3">
      <button 
        onclick={() => window.location.href='/api/v1/auth/oidc/login'} 
        class="w-full border border-border-subtle p-3 text-[10px] font-bold text-text-muted uppercase tracking-widest hover:border-text-muted hover:text-text-secondary transition-all flex items-center justify-center gap-2"
      >
        SINGLE SIGN-ON (OIDC)
      </button>
      <button 
        onclick={() => window.location.href='/api/v1/auth/saml/login'} 
        class="w-full border border-border-subtle p-3 text-[10px] font-bold text-text-muted uppercase tracking-widest hover:border-text-muted hover:text-text-secondary transition-all flex items-center justify-center gap-2"
      >
        FEDERATED IDENTITY (SAML)
      </button>
    </div>

    <div class="mt-10 text-center space-y-2">
      <div class="flex items-center justify-center gap-2 text-[9px] font-mono text-text-muted uppercase tracking-[0.2em]">
        <ShieldCheck size={12} class="text-status-online" />
        Root-of-Trust Attestation: <span class="text-status-online font-black">VERIFIED</span>
      </div>
      <p class="text-[8px] font-mono text-text-muted/40 uppercase tracking-widest">
        End-to-End Encryption Protocol: RSA_4096_L7_AWARE
      </p>
    </div>
  </div>
</div>

<style>
  :global(.animate-scan) {
    animation: scan 2s linear infinite;
  }
  @keyframes scan {
    0% { transform: translateY(-10px); }
    100% { transform: translateY(80px); }
  }
</style>
