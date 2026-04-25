// i18n — minimal Svelte 5 translation store + locale switcher.
//
// Why custom instead of svelte-i18n / i18next: those libraries weigh
// 30-80 KB and pull in ICU MessageFormat etc. The OBLIVRA UI has
// ~500 user-facing strings, no plurals beyond simple count formatting,
// and ships in an air-gap binary where every kilobyte matters. A
// hand-rolled store covers our needs in ~80 LOC.
//
// Phase 24.2 — Arabic / RTL support.
//
// Adding a locale:
//   1. Drop `frontend/src/lib/i18n/<code>.ts` exporting a Translations object
//   2. Register it in LOCALES below
//   3. If the locale is RTL, list it in RTL_LOCALES
//
// Adding a translation key:
//   1. Add the English string to `en.ts`
//   2. Add the Arabic translation to `ar.ts` (or other locales)
//   3. Use `t('your.key')` in any Svelte component
//
// Missing-key behaviour: returns the key itself with a console warning
// in dev mode so untranslated strings are visible in the UI without
// breaking the page.

import { en } from './en';
import { ar } from './ar';

export type Translations = Record<string, string>;
export type LocaleCode = 'en' | 'ar';

const LOCALES: Record<LocaleCode, Translations> = { en, ar };
const RTL_LOCALES: ReadonlySet<LocaleCode> = new Set<LocaleCode>(['ar']);

const STORAGE_KEY = 'oblivra:locale';

// Resolve the operator's preferred locale at module load. Order:
//   1. localStorage override (operator chose explicitly via Settings)
//   2. navigator.language prefix
//   3. fallback to English
function resolveInitialLocale(): LocaleCode {
  if (typeof localStorage !== 'undefined') {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'en' || stored === 'ar') return stored;
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

/**
 * t — translate a key. Optional positional `{0}`, `{1}` ... interpolations.
 *
 * Reactivity: this function reads `i18n.locale` (a $state rune) so any
 * component that calls `t(...)` inside a `$derived` or template will
 * re-render when the locale changes.
 *
 * Missing keys: returns the key itself plus a console.warn in dev so the
 * untranslated string is visible without crashing the page.
 */
export function t(key: string, ...args: Array<string | number>): string {
  const table = LOCALES[i18n.locale] ?? LOCALES.en;
  let raw = table[key];
  if (raw == null) {
    // Fall back to English; warn once in dev for visibility.
    raw = LOCALES.en[key];
    if (raw == null) {
      if (typeof console !== 'undefined' && import.meta?.env?.DEV) {
        console.warn(`[i18n] missing translation: ${key} (locale=${i18n.locale})`);
      }
      return key;
    }
  }
  if (args.length === 0) return raw;
  return raw.replace(/\{(\d+)\}/g, (_match, idx) => {
    const v = args[Number(idx)];
    return v == null ? '' : String(v);
  });
}
