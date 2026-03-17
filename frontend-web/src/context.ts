/**
 * context.ts — Dual-Context Substrate (Phase 0.5)
 *
 * Provides a single authoritative source for detecting whether OBLIVRA is
 * running inside the Wails Desktop runtime or as a standalone Browser web app.
 *
 * Usage:
 *   import { getAppContext, isDesktop, isWeb } from './context';
 *   if (isWeb()) { ... }
 */

/** The two execution contexts OBLIVRA supports. */
export type AppContext = 'desktop' | 'web';

/**
 * Detects the current execution context.
 *
 * - 'desktop' if running inside the Wails runtime (window.__WAILS__ is present)
 *   or if the app was built with VITE_APP_CONTEXT=desktop.
 * - 'web'     in all other cases (standard browser, Electron-less, CI, etc.)
 */
export function getAppContext(): AppContext {
  // 1. Build-time override from Vite env variable
  //    Set VITE_APP_CONTEXT=desktop when building the Wails bundle.
  if (import.meta.env.VITE_APP_CONTEXT === 'desktop') return 'desktop';

  // 2. Runtime detection — Wails injects a global marker on load.
  //    We use `window.__WAILS__` as the canonical sentinel.
  if (typeof window !== 'undefined' && '__WAILS__' in window) return 'desktop';

  return 'web';
}

/** Convenience helpers — avoids inline comparisons throughout the code. */
export const isDesktop = (): boolean => getAppContext() === 'desktop';
export const isWeb = (): boolean => getAppContext() === 'web';
