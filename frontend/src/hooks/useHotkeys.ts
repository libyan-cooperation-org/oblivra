import { onMount, onCleanup } from 'solid-js';

type HotkeyCallback = (e: KeyboardEvent) => void;

interface HotkeyMap {
    [keyCombo: string]: HotkeyCallback;
}

/**
 * Parses a key combination string like "Cmd+P" or "Ctrl+Shift+F"
 */
function parseCombo(combo: string) {
    const parts = combo.toLowerCase().split('+');
    const key = parts[parts.length - 1];

    return {
        key,
        ctrlCmd: parts.includes('cmd') || parts.includes('ctrl') || parts.includes('mod'),
        shift: parts.includes('shift'),
        alt: parts.includes('alt')
    };
}

/**
 * A global hook to register keyboard shortcuts across the app.
 * Pass a map of string combos to their respective callbacks.
 * Example hook usage: 
 * useHotkeys({
 *   'Cmd+P': (e) => { e.preventDefault(); openPalette(); },
 *   'Alt+1': () => switchTab(1)
 * })
 */
export function useHotkeys(hotkeys: HotkeyMap) {
    onMount(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // Don't trigger hotkeys if user is typing in an input/textarea
            const target = e.target as HTMLElement;
            if (['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable) {
                // Exception: Escape should generally still work to blur/close
                if (e.key !== 'Escape') {
                    return;
                }
            }

            for (const [combo, callback] of Object.entries(hotkeys)) {
                const specs = parseCombo(combo);
                const isCtrlCmd = e.ctrlKey || e.metaKey;

                if (
                    e.key.toLowerCase() === specs.key &&
                    isCtrlCmd === specs.ctrlCmd &&
                    e.shiftKey === specs.shift &&
                    e.altKey === specs.alt
                ) {
                    callback(e);
                }
            }
        };

        window.addEventListener('keydown', handleKeyDown);

        onCleanup(() => {
            window.removeEventListener('keydown', handleKeyDown);
        });
    });
}
