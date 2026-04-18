<!--
  OBLIVRA — Login Page (Svelte 5)
  Used in BROWSER mode for standard web authentication.
-->
<script lang="ts">
  import { login } from '@lib/services/auth'; // I will create this service next
  import { toastStore } from '@lib/stores/toast.svelte';

  let email = $state('');
  let password = $state('');
  let loading = $state(false);
  let error = $state<string | null>(null);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    loading = true;
    error = null;

    try {
      await login(email, password);
      // For now, redirect to root or update appStore
      window.location.href = '#/';
      window.location.reload(); // Hard reload to clear state for now
    } catch (err: any) {
      error = err.message || 'Access Denied';
      toastStore.add({
        type: 'error',
        title: 'Authentication Failed',
        message: error || undefined
      });
    } finally {
      loading = false;
    }
  }
</script>

<div class="min-h-screen bg-black flex items-center justify-center p-4 font-mono">
  <!-- Grid background -->
  <div class="fixed inset-0 bg-[linear-gradient(rgba(18,16,16,0.1)_1px,transparent_1px),linear-gradient(90deg,rgba(18,16,16,0.1)_1px,transparent_1px)] bg-[size:40px_40px] pointer-events-none opacity-20"></div>

  <div class="relative w-full max-w-md bg-zinc-900 border border-zinc-700 p-10 shadow-[0_0_100px_rgba(0,0,0,1)] animate-fade-in">
    <div class="mb-10">
      <h1 class="text-3xl font-black text-white tracking-tighter uppercase italic">
        OBLIVRA <span class="bg-red-600 px-1 not-italic text-black font-black">ENTERPRISE</span>
      </h1>
      <p class="text-zinc-500 text-[10px] uppercase tracking-[0.4em] font-bold mt-2">Headless Access Portal v1.0.0</p>
    </div>

    <form onsubmit={handleSubmit} class="space-y-6">
      <div class="space-y-2">
        <label class="block text-zinc-400 text-[10px] uppercase tracking-widest font-bold ml-1">Identity (Email)</label>
        <input
          type="email"
          bind:value={email}
          placeholder="operator@oblivra.org"
          required
          class="w-full bg-black border border-zinc-700 p-4 text-white focus:outline-none focus:border-red-600 transition-all duration-300 placeholder:text-zinc-800"
          disabled={loading}
        />
      </div>

      <div class="space-y-2">
        <label class="block text-zinc-400 text-[10px] uppercase tracking-widest font-bold ml-1">Passphrase</label>
        <input
          type="password"
          bind:value={password}
          placeholder="••••••••••••"
          required
          class="w-full bg-black border border-zinc-700 p-4 text-white focus:outline-none focus:border-red-600 transition-all duration-300 placeholder:text-zinc-800"
          disabled={loading}
        />
      </div>

      <button
        type="submit"
        disabled={loading || !email || !password}
        class="w-full h-14 bg-white text-black font-black uppercase tracking-widest hover:bg-red-600 hover:text-white transition-all duration-300 disabled:opacity-20 disabled:grayscale relative overflow-hidden group"
      >
        <span class="relative z-10">{loading ? 'Verifying...' : 'Authorize Session'}</span>
        {#if !loading}
          <div class="absolute inset-0 bg-red-600 translate-y-full group-hover:translate-y-0 transition-transform duration-300"></div>
        {/if}
      </button>

      {#if error}
        <div class="bg-red-900/20 border border-red-900/50 p-4 text-red-500 text-[10px] font-bold uppercase text-center">
          Access Denied: {error}
        </div>
      {/if}
    </form>

    <div class="mt-12 flex flex-col gap-4">
      <button class="w-full border border-zinc-800 text-zinc-600 text-[10px] uppercase tracking-widest font-bold py-3 hover:border-zinc-500 hover:text-zinc-300 transition-all">
        Single Sign-On (OIDC)
      </button>
      <button class="w-full border border-zinc-800 text-zinc-600 text-[10px] uppercase tracking-widest font-bold py-3 hover:border-zinc-500 hover:text-zinc-300 transition-all">
        Federated Identity (SAML)
      </button>
    </div>

    <div class="mt-10 text-[9px] text-zinc-700 text-center uppercase tracking-[0.3em] font-medium leading-loose">
      Sovereign-Grade Encryption Active<br />
      Attestation Level: <span class="text-green-900/50">PLATINUM-ROOT</span>
    </div>
  </div>
</div>
