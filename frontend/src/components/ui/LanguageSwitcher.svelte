<!--
  LanguageSwitcher — segmented locale picker. Drop into the Settings
  page (or anywhere) to let the operator change the UI language.

  Reads i18n.locale and writes via i18n.setLocale, which persists to
  localStorage and updates <html dir="rtl"> for Arabic (Phase 24.2).
-->
<script lang="ts">
  import { Languages } from 'lucide-svelte';
  import { i18n, t, type LocaleCode } from '@lib/i18n';

  const locales = Object.entries(i18n.availableLocales) as Array<[LocaleCode, string]>;

  function pick(code: LocaleCode) {
    i18n.setLocale(code);
  }
</script>

<div class="language-switcher">
  <div class="header">
    <Languages class="w-3.5 h-3.5 text-text-muted" />
    <span class="title">{t('settings.language')}</span>
  </div>
  <p class="description">{t('settings.language.description')}</p>

  <div class="picker" role="radiogroup" aria-label={t('settings.language')}>
    {#each locales as [code, label] (code)}
      <button
        type="button"
        role="radio"
        aria-checked={i18n.locale === code}
        class="opt"
        class:active={i18n.locale === code}
        onclick={() => pick(code)}
      >
        <span class="code">{code.toUpperCase()}</span>
        <span class="label">{label}</span>
      </button>
    {/each}
  </div>
</div>

<style>
  .language-switcher {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 1rem;
    border: 1px solid var(--color-border-primary, #2b2d31);
    background: var(--color-surface-1, #1a1c20);
    border-radius: 4px;
    max-width: 480px;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .title {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--color-text-heading, #d4d4d4);
  }

  .description {
    font-size: 0.75rem;
    color: var(--color-text-muted, #888);
    margin: 0 0 0.5rem;
    line-height: 1.5;
  }

  .picker {
    display: flex;
    gap: 0.25rem;
  }

  .opt {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.125rem;
    padding: 0.625rem 0.875rem;
    background: var(--color-surface-2, #2b2d31);
    border: 1px solid var(--color-border-primary, #2b2d31);
    color: var(--color-text-primary, #d4d4d4);
    cursor: pointer;
    font-family: inherit;
    transition: all 150ms;
  }

  .opt:hover {
    border-color: var(--color-accent, #0099e0);
  }

  .opt.active {
    background: var(--color-surface-3, #353740);
    border-color: var(--color-accent, #0099e0);
    box-shadow: inset 0 0 0 1px var(--color-accent, #0099e0);
  }

  .code {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    color: var(--color-accent, #0099e0);
  }

  .label {
    font-size: 0.875rem;
  }
</style>
