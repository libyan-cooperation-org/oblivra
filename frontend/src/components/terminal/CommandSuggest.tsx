import { Component, For, Show, createSignal, onCleanup } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface CommandSuggestProps {
    hostId: string;
    currentInput: string;
    visible: boolean;
    anchorX: number;
    anchorY: number;
    onSelect: (command: string) => void;
    onDismiss: () => void;
}

/**
 * CommandSuggest is a floating autocomplete overlay for the terminal.
 * It shows command suggestions from:
 * 1. Per-host command history
 * 2. Common shell commands
 * 3. (Future) AI-powered suggestions from AIService
 */
export const CommandSuggest: Component<CommandSuggestProps> = (props) => {
    const [suggestions, setSuggestions] = createSignal<string[]>([]);
    const [selectedIdx, setSelectedIdx] = createSignal(0);

    // Fetch suggestions when input changes
    const fetchSuggestions = async () => {
        if (IS_BROWSER || !props.hostId || !props.currentInput) {
            setSuggestions([]);
            return;
        }
        try {
            const { GetSuggestions } = await import('../../../wailsjs/go/services/CommandHistoryService');
            const results = await GetSuggestions(props.hostId, props.currentInput);
            setSuggestions(results || []);
            setSelectedIdx(0);
        } catch {
            setSuggestions([]);
        }
    };

    // Watch input changes (debounced)
    let debounceTimer: number | undefined;
    const debouncedFetch = () => {
        clearTimeout(debounceTimer);
        debounceTimer = window.setTimeout(fetchSuggestions, 150);
    };

    // Trigger on input change
    (() => {
        if (props.visible && props.currentInput) debouncedFetch();
    })();

    onCleanup(() => clearTimeout(debounceTimer));

    const handleKeyDown = (e: KeyboardEvent) => {
        if (!props.visible || suggestions().length === 0) return;

        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setSelectedIdx(i => Math.min(i + 1, suggestions().length - 1));
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setSelectedIdx(i => Math.max(i - 1, 0));
        } else if (e.key === 'Tab' || e.key === 'Enter') {
            e.preventDefault();
            props.onSelect(suggestions()[selectedIdx()]);
        } else if (e.key === 'Escape') {
            props.onDismiss();
        }
    };

    return (
        <Show when={props.visible && suggestions().length > 0}>
            <div
                style={{
                    position: 'fixed',
                    left: `${props.anchorX}px`,
                    top: `${props.anchorY}px`,
                    'min-width': '280px',
                    'max-width': '400px',
                    'max-height': '200px',
                    overflow: 'auto',
                    background: '#1a1b1f',
                    border: '1px solid rgba(0,153,224,0.3)',
                    'border-radius': '4px',
                    'box-shadow': '0 8px 24px rgba(0,0,0,0.6)',
                    'z-index': '9999',
                    'font-family': "'JetBrains Mono', monospace",
                    'font-size': '12px',
                }}
                onKeyDown={handleKeyDown}
            >
                {/* Header */}
                <div style={{
                    padding: '4px 10px',
                    'font-size': '9px',
                    color: 'var(--text-muted)',
                    'text-transform': 'uppercase',
                    'letter-spacing': '0.5px',
                    'border-bottom': '1px solid rgba(255,255,255,0.06)',
                }}>
                    Suggestions · ↑↓ navigate · Tab accept
                </div>

                {/* Suggestion list */}
                <For each={suggestions()}>
                    {(cmd, idx) => (
                        <div
                            onClick={() => props.onSelect(cmd)}
                            onMouseEnter={() => setSelectedIdx(idx())}
                            style={{
                                padding: '5px 10px',
                                cursor: 'pointer',
                                background: selectedIdx() === idx() ? 'rgba(0,153,224,0.15)' : 'transparent',
                                color: selectedIdx() === idx() ? '#33b8ff' : 'var(--text-secondary)',
                                'border-left': selectedIdx() === idx() ? '2px solid #0099e0' : '2px solid transparent',
                                transition: 'background 0.08s',
                            }}
                        >
                            {cmd}
                        </div>
                    )}
                </For>
            </div>
        </Show>
    );
};
