import { Component, createSignal, For, Show } from 'solid-js';

interface QueryEditorProps {
    value: string;
    onInput: (val: string) => void;
    onRun: () => void;
}

export const QueryEditor: Component<QueryEditorProps> = (props) => {
    const [suggestions, setSuggestions] = createSignal<string[]>([]);
    const [cursorPos, setCursorPos] = createSignal(0);

    const COMMANDS = [
        'where', 'stats', 'eval', 'table', 'sort', 'head', 'tail', 
        'dedup', 'rename', 'fields', 'fillnull', 'top', 'rare', 
        'rex', 'lookup', 'join', 'append', 'timechart', 'chart', 'mvexpand'
    ];

    const handleInput = (e: any) => {
        const val = e.currentTarget.value;
        const pos = e.currentTarget.selectionStart;
        setCursorPos(pos);
        props.onInput(val);

        // Simple autocomplete logic
        const rawLastWord = val.slice(0, pos).split(/[\s|]+/).pop() || '';
        if (rawLastWord.length > 1) {
            setSuggestions(COMMANDS.filter(c => c.startsWith(rawLastWord.toLowerCase())));
        } else {
            setSuggestions([]);
        }
    };

    const insertSuggestion = (s: string) => {
        const val = props.value;
        const pos = cursorPos();
        const parts = val.slice(0, pos).split(/([\s|]+)/);
        parts.pop(); // remove partial word
        const newVal = parts.join('') + s + val.slice(pos);
        props.onInput(newVal);
        setSuggestions([]);
    };

    return (
        <div style={{
            position: 'relative',
            'font-family': 'var(--font-mono)',
            'font-size': '13px',
            background: 'var(--surface-0)',
            border: '1px solid var(--border-primary)',
            'border-radius': 'var(--radius-md)',
            overflow: 'hidden',
            transition: 'border-color var(--transition-fast)'
        }}>
            <textarea
                style={{
                    width: '100%',
                    height: '120px',
                    padding: 'var(--gap-md)',
                    background: 'transparent',
                    color: 'var(--text-primary)',
                    border: 'none',
                    outline: 'none',
                    resize: 'none',
                    'font-family': 'inherit',
                    'font-size': '13px',
                    'line-height': '1.6'
                }}
                spellcheck={false}
                placeholder="source=terminal | where severity=critical | stats count by host"
                value={props.value}
                onInput={handleInput}
                onKeyDown={e => {
                    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                        e.preventDefault();
                        props.onRun();
                    }
                }}
            />
            
            <Show when={suggestions().length > 0}>
                <div style={{
                    position: 'absolute',
                    'z-index': 100,
                    bottom: '100%',
                    left: 'var(--gap-md)',
                    'margin-bottom': '4px',
                    width: '200px',
                    background: 'var(--surface-2)',
                    border: '1px solid var(--border-primary)',
                    'border-radius': 'var(--radius-sm)',
                    'box-shadow': 'var(--shadow-lg)',
                    overflow: 'hidden'
                }}>
                    <div style="background: var(--surface-3); padding: 4px 10px; font-size: 9px; font-weight: 800; color: var(--text-muted); border-bottom: 1px solid var(--border-primary);">SUGGESTIONS</div>
                    <For each={suggestions()}>
                        {s => (
                            <div 
                                style={{
                                    padding: '6px 12px',
                                    cursor: 'pointer',
                                    color: 'var(--accent-primary)',
                                    'border-bottom': '1px solid var(--border-subtle)',
                                    'font-weight': '600',
                                    transition: 'background 0.1s'
                                }}
                                onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(0,153,224,0.1)'}
                                onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                                onClick={() => insertSuggestion(s)}
                            >
                                {s}
                            </div>
                        )}
                    </For>
                </div>
            </Show>

            <div style={{
                padding: '4px var(--gap-md)',
                background: 'var(--surface-1)',
                'border-top': '1px solid var(--border-primary)',
                display: 'flex',
                'justify-content': 'space-between',
                'align-items': 'center',
                'font-size': '10px',
                'font-weight': '800',
                color: 'var(--text-muted)'
            }}>
                <div style="letter-spacing: 0.5px;">OQL v2.0 · SOVEREIGN QUERY LANGUAGE</div>
                <div style="letter-spacing: 0.5px;">CTRL+ENTER TO RUN</div>
            </div>
        </div>
    );
};
