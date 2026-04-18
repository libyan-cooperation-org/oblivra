<!-- OBLIVRA Web — LoadingScreen (Svelte 5) -->
<script lang="ts">
  import AppLogo from './AppLogo.svelte';
  interface Props { message?: string; }
  let { message = 'Initializing sovereign core…' }: Props = $props();
</script>

<div class="ls-wrap">
  <div class="ls-content">
    <div class="ls-logo-wrap">
      <AppLogo size={72} class="ls-logo" />
    </div>
    <div class="ls-info">
      <h1 class="ls-title">OBLIVRA</h1>
      <div class="ls-bar-wrap"><div class="ls-bar"></div></div>
      <p class="ls-msg">{message}</p>
    </div>
  </div>
</div>

<style>
  .ls-wrap {
    position: fixed; inset: 0; background: #080f12;
    display: flex; align-items: center; justify-content: center; z-index: 9999;
  }
  .ls-content { display: flex; flex-direction: column; align-items: center; gap: 32px; }
  .ls-logo-wrap { position: relative; }
  .ls-logo { filter: drop-shadow(0 0 16px rgba(0,153,224,0.4)); }
  .ls-info { display: flex; flex-direction: column; align-items: center; gap: 12px; }
  .ls-title {
    font-size: 22px; font-weight: 800; letter-spacing: 6px;
    color: #fff; text-transform: uppercase; font-family: var(--font-ui);
  }
  .ls-bar-wrap {
    width: 200px; height: 2px; background: rgba(255,255,255,0.06);
    border-radius: 1px; overflow: hidden;
  }
  .ls-bar {
    height: 100%; width: 40%; background: var(--accent-primary);
    border-radius: 1px; animation: ls-slide 1.4s ease-in-out infinite;
  }
  .ls-msg { font-size: 12px; color: #607070; font-family: monospace; }
  @keyframes ls-slide {
    0%   { transform: translateX(-100%); }
    100% { transform: translateX(600%); }
  }
</style>
