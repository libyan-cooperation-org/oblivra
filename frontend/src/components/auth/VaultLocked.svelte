<!--
  OBLIVRA — VaultLocked (Svelte 5)
  High-security barrier screen shown when the local vault is locked.
-->
<script lang="ts">
  import { guardedUnlock } from '@lib/bridge';
  import { appStore } from '@lib/stores/app.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import Button from '@components/ui/Button.svelte';
  import Input from '@components/ui/Input.svelte';

  let passphrase = $state('');
  let loading = $state(false);
  let remember = $state(false);
  let error = $state<string | null>(null);

  async function handleUnlock() {
    if (!passphrase) return;
    loading = true;
    error = null;

    try {
      await guardedUnlock(passphrase, remember);
      // If successful, 'vault:unlocked' event will fire and AppStore will update.
    } catch (err: any) {
      error = err.message || 'Access Denied';
      toastStore.add({
        type: 'error',
        title: 'Vault Unlock Failed',
        message: error
      });
      passphrase = '';
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleUnlock();
  }
</script>

<div class="fixed inset-0 z-[1000] flex items-center justify-center bg-black font-mono">
  <!-- Scanline effect -->
  <div class="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(18,16,16,0)_50%,rgba(0,0,0,0.25)_50%),linear-gradient(90deg,rgba(255,0,0,0.06),rgba(0,255,0,0.02),rgba(0,0,255,0.06))] bg-[length:100%_2px,3px_100%]"></div>

  <div class="relative w-full max-w-md p-8 bg-zinc-900 border border-zinc-700 shadow-[0_0_50px_rgba(0,0,0,0.8)] animate-in fade-in zoom-in duration-300">
    <div class="mb-10 text-center">
      <h1 class="text-3xl font-black text-white tracking-tighter uppercase italic">
        OBLIVRA <span class="bg-red-600 px-1 not-italic text-black">VAULT</span>
      </h1>
      <div class="mt-2 flex items-center justify-center gap-2">
        <div class="h-1.5 w-1.5 rounded-full bg-red-600 animate-pulse"></div>
        <p class="text-zinc-500 text-[10px] uppercase tracking-[0.3em] font-bold">Encrypted Volume Terminal</p>
      </div>
    </div>

    <div class="space-y-6">
      <div class="space-y-2">
        <label class="block text-zinc-500 text-[10px] uppercase tracking-widest font-bold ml-1">Master Passphrase</label>
        <div class="relative">
          <input
            type="password"
            bind:value={passphrase}
            onkeydown={handleKeydown}
            placeholder="••••••••••••"
            class="w-full bg-black border border-zinc-700 p-4 text-white focus:outline-none focus:border-red-600 focus:shadow-[0_0_15px_rgba(220,38,38,0.2)] transition-all duration-300 placeholder:text-zinc-800"
            disabled={loading}
          />
          {#if loading}
            <div class="absolute right-4 top-1/2 -translate-y-1/2">
              <div class="h-4 w-4 border-2 border-red-600 border-t-transparent rounded-full animate-spin"></div>
            </div>
          {/if}
        </div>
      </div>

      <div class="flex items-center gap-2 px-1">
        <input 
          type="checkbox" 
          id="remember" 
          bind:checked={remember} 
          class="w-4 h-4 bg-black border-zinc-700 rounded-sm text-red-600 focus:ring-0 focus:ring-offset-0"
        />
        <label for="remember" class="text-[10px] text-zinc-500 uppercase tracking-widest cursor-pointer hover:text-zinc-300 transition-colors">Remember for session</label>
      </div>

      <button
        onclick={handleUnlock}
        disabled={loading || !passphrase}
        class="w-full h-14 bg-white text-black font-black uppercase tracking-widest hover:bg-red-600 hover:text-white transition-all duration-300 disabled:opacity-20 disabled:grayscale relative overflow-hidden group"
      >
        <span class="relative z-10">Decrypt Volume</span>
        <div class="absolute inset-0 bg-red-600 translate-y-full group-hover:translate-y-0 transition-transform duration-300"></div>
      </button>

      {#if error}
        <div class="bg-red-900/20 border border-red-900/50 p-3 text-red-500 text-[10px] font-bold uppercase text-center animate-shake">
          Access Denied: {error}
        </div>
      {/if}
    </div>

    <div class="mt-10 pt-6 border-t border-zinc-800 flex flex-col gap-3">
      <div class="flex justify-between items-center text-[9px] text-zinc-600 uppercase tracking-widest">
        <span>FIDO2 / YubiKey</span>
        <span class="text-zinc-800">Ready</span>
      </div>
      <div class="flex justify-between items-center text-[9px] text-zinc-600 uppercase tracking-widest">
        <span>AES-256-GCM / Argon2</span>
        <span class="text-green-900/50">Active</span>
      </div>
    </div>
  </div>

  <div class="fixed bottom-8 text-[10px] text-zinc-700 uppercase tracking-[0.5em] font-medium opacity-50">
    Sovereign Intelligence Node — Session Isolated
  </div>
</div>

<style>
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-4px); }
    75% { transform: translateX(4px); }
  }
  .animate-shake {
    animation: shake 0.5s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
