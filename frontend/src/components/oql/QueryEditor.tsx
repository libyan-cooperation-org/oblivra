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

    const handleInput = (e: InputEvent) => {
        const val = (e.target as HTMLTextAreaElement).value;
        const pos = (e.target as HTMLTextAreaElement).selectionStart;
        setCursorPos(pos);
        props.onInput(val);

        // Simple autocomplete logic
        const lastWord = val.slice(0, pos).split(/[\s|]+/).pop() || '';
        if (lastWord.length > 1) {
            setSuggestions(COMMANDS.filter(c => c.startsWith(lastWord.toLowerCase())));
        } else {
            setSuggestions([]);
        }
    };

    const insertSuggestion = (s: string) => {
        const val = props.value;
        const pos = cursorPos();
        const words = val.slice(0, pos).split(/([\s|]+)/);
        words.pop(); // remove partial word
        const newVal = words.join('') + s + val.slice(pos);
        props.onInput(newVal);
        setSuggestions([]);
    };

    return (
        <div class="relative font-mono text-sm bg-black/40 border border-white/10 rounded-lg overflow-hidden focus-within:border-blue-500/50 transition-colors">
            <textarea
                class="w-full h-32 p-4 bg-transparent text-gray-100 outline-none resize-none placeholder-gray-600"
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
                <div class="absolute z-10 bg-gray-900 border border-white/10 shadow-xl rounded-md w-48 overflow-hidden" 
                     style={{ left: '1rem', bottom: '100%', 'margin-bottom': '4px' }}>
                    <For each={suggestions()}>
                        {s => (
                            <div 
                                class="px-3 py-1.5 hover:bg-blue-600/30 cursor-pointer text-blue-400 border-b border-white/5 last:border-0"
                                onClick={() => insertSuggestion(s)}
                            >
                                {s}
                            </div>
                        )}
                    </For>
                </div>
            </Show>

            <div class="px-4 py-2 border-t border-white/5 bg-white/5 flex justify-between items-center text-[10px] uppercase tracking-widest text-gray-500 font-bold">
                <div>OQL v2.0 · Sovereign Query Language</div>
                <div>CTRL+ENTER TO RUN</div>
            </div>
        </div>
    );
};
