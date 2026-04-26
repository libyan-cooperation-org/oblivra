// i18n/store.svelte.ts — runes-aware locale store.
//
// Lives in a `.svelte.ts` file so it can use `$state`. The companion
// `index.ts` re-exports `i18n` from here plus the (rune-free) `t()`
// helper, so callers continue to write `import { t, i18n } from '@lib/i18n'`.
//
// Why split? Svelte 5 throws `rune_outside_svelte` at runtime when a
// regular `.ts` file uses `$state`. The original `index.ts` got away
// with it in dev mode (the dev runtime is permissive about file
// extensions for back-compat) but production strictly enforces the
// `.svelte.ts` requirement and threw a Svelte runtime error during
// the very first import — breaking the entire app boot.

import { en } from './en';
import { ar } from './ar';

export type Translations = Record<string, string>;
export type LocaleCode = 'en' | 'ar';

export const LOCALES: Record<LocaleCode, Translations> = { en, ar };
export const RTL_LOCALES: ReadonlySet<LocaleCode> = new Set<LocaleCode>(['ar']);

const STORAGE_KEY = 'oblivra:locale';

// Resolve the operator's preferred locale at module load. Order:
//   1. localStorage override (operator chose explicitly via Settings)
//   2. navigator.language prefix
//   3. fallback to English
function resolveInitialLocale(): LocaleCode {
  if (typeof localStorage !== 'undefined') {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored === 'en' || stored === 'ar') return stored;
    } catch { /* private browsing */ }
  }
  if (typeof navigator !== 'undefined' && navigator.language) {
    const prefix = navigator.language.toLowerCase().slice(0, 2);
    if (prefix === 'ar') return 'ar';
  }
  return 'en';
}

class I18nStore {
  locale = $state<LocaleCode>(resolveInitialLocale());

  /** True when the active locale is right-to-left. Drives <html dir="rtl">. */
  get isRTL(): boolean {
    return RTL_LOCALES.has(this.locale);
  }

  /** Map of available locale codes → human-readable name (in their own script). */
  readonly availableLocales: Record<LocaleCode, string> = {
    en: 'English',
    ar: 'العربية',
  };

  setLocale(next: LocaleCode) {
    if (this.locale === next) return;
    this.locale = next;
    if (typeof localStorage !== 'undefined') {
      try { localStorage.setItem(STORAGE_KEY, next); } catch { /* quota / private mode */ }
    }
    this.applyDocumentDirection();
  }

  /** Force the <html dir> attribute to match the active locale. Idempotent. */
  applyDocumentDirection() {
    if (typeof document === 'undefined') return;
    const dir = this.isRTL ? 'rtl' : 'ltr';
    document.documentElement.setAttribute('dir', dir);
    document.documentElement.setAttribute('lang', this.locale);
  }
}

export const i18n = new I18nStore();

// Apply on first load so the page renders in the correct direction
// before any component mounts.
if (typeof document !== 'undefined') {
  i18n.applyDocumentDirection();
}
