// Clipboard helper — wraps navigator.clipboard.writeText with a toast
// so analysts get instant confirmation when they copy a hash, event
// id, OQL string, or vault secret. Falls back gracefully on browsers
// that don't expose the clipboard API (rare; mostly older Safari and
// secure-context-disabled iframes).

import { toast } from './stores/toast.svelte';

export async function copy(text: string, label = 'Copied'): Promise<void> {
  if (!text) {
    toast.warn('Nothing to copy');
    return;
  }
  try {
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
    } else {
      // Fallback: temporary textarea + execCommand. Older path; works
      // on file:// and pre-secure-context environments.
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
    }
    toast.success(label, truncate(text, 60));
  } catch (err) {
    toast.error('Copy failed', String(err));
  }
}

function truncate(s: string, n: number): string {
  return s.length <= n ? s : s.slice(0, n - 1) + '…';
}
