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
//   2. Register it in LOCALES (in `store.svelte.ts`)
//   3. If the locale is RTL, list it in RTL_LOCALES (in `store.svelte.ts`)
//
// Adding a translation key:
//   1. Add the English string to `en.ts`
//   2. Add the Arabic translation to `ar.ts` (or other locales)
//   3. Use `t('your.key')` in any Svelte component
//
// File layout:
//   - `store.svelte.ts` — runes-aware I18nStore (uses `$state`)
//   - `index.ts` (this file) — barrel + the rune-free `t()` helper
//
// The split exists because Svelte 5 forbids `$state` in regular `.ts`
// files; `t()` is a plain function so it stays here.

import { i18n, LOCALES, type LocaleCode, type Translations } from './store.svelte';

export { i18n };
export type { LocaleCode, Translations };

/**
 * t — translate a key. Optional positional `{0}`, `{1}` ... interpolations.
 *
 * Reactivity: this function reads `i18n.locale` — a state rune defined
 * in `store.svelte.ts`. Any component that calls `t(...)` inside a
 * derived or template re-renders when the locale changes. Reading a
 * rune from a non-`.svelte.ts` file is allowed; only DECLARING one
 * (the rune-invocation form) is restricted.
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
      // Vite/ImportMeta typing isn't picked up by `tsc --noEmit` because
      // we don't include `vite/client` in tsconfig types. Cast through
      // `any` to read the dev-mode flag at runtime; the JSDoc above
      // documents intent so this is not a silent any.
      const isDev = (import.meta as any)?.env?.DEV === true;
      if (typeof console !== 'undefined' && isDev) {
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
