// Tiny toast notification store — stack-based, auto-dismissing.
// Designed for terse "copied!" / "saved!" feedback rather than long
// error reports (which belong inline in the relevant view).
//
// File ends in .svelte.ts so the Svelte 5 compiler treats `$state`
// as a rune. Any component can `import { toastState, toast }` and
// read/mutate. Toasts auto-expire 3.5s after they're added; the
// ToastStack component renders them.

export type ToastKind = 'info' | 'success' | 'warn' | 'error';

export interface Toast {
  id: number;
  kind: ToastKind;
  text: string;
  detail?: string;
  expiresAt: number;
}

let nextId = 1;

export const toastState = $state<{ items: Toast[] }>({ items: [] });

function push(kind: ToastKind, text: string, detail?: string, ttlMs = 3500) {
  const t: Toast = {
    id: nextId++,
    kind,
    text,
    detail,
    expiresAt: Date.now() + ttlMs,
  };
  toastState.items = [...toastState.items, t];
  setTimeout(() => dismiss(t.id), ttlMs);
}

export function dismiss(id: number) {
  toastState.items = toastState.items.filter((t) => t.id !== id);
}

export const toast = {
  info: (text: string, detail?: string) => push('info', text, detail),
  success: (text: string, detail?: string) => push('success', text, detail),
  warn: (text: string, detail?: string) => push('warn', text, detail, 5000),
  error: (text: string, detail?: string) => push('error', text, detail, 7000),
};
