import { Component, createSignal, For, onMount, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface Message {
    role: 'user' | 'assistant' | 'system' | 'error';
    content: string;
    timestamp: string;
}

type Mode = 'chat' | 'explain' | 'generate';

export const AIAssistantPage: Component = () => {
    const [messages, setMessages] = createSignal<Message[]>([]);
    const [input, setInput] = createSignal('');
    const [loading, setLoading] = createSignal(false);
    const [ollamaStatus, setOllamaStatus] = createSignal<'unknown' | 'online' | 'offline'>('unknown');
    const [mode, setMode] = createSignal<Mode>('chat');
    let bottomRef: HTMLDivElement | undefined;

    const scrollToBottom = () => {
        bottomRef?.scrollIntoView({ behavior: 'smooth' });
    };

    const checkOllamaStatus = async () => {
        if (IS_BROWSER) { setOllamaStatus('offline'); return; }
        try {
            const { SendMessage } = await import('../../wailsjs/go/services/AIService');
            await SendMessage('ping');
            setOllamaStatus('online');
        } catch {
            setOllamaStatus('offline');
        }
    };

    onMount(async () => {
        if (!IS_BROWSER) {
            try {
                const { GetChatHistory } = await import('../../wailsjs/go/services/AIService');
                const history = await GetChatHistory();
                const displayHistory = (history || []).filter((m: Message) => m.role !== 'system');
                setMessages(displayHistory);
            } catch (e) {
                console.error('Failed to load chat history:', e);
            }
        }
        await checkOllamaStatus();
        scrollToBottom();
    });

    const pushMessage = (msg: Message) => {
        setMessages(prev => [...prev, msg]);
        setTimeout(scrollToBottom, 50);
    };

    const handleSend = async () => {
        const text = input().trim();
        if (!text || loading()) return;

        pushMessage({ role: 'user', content: text, timestamp: new Date().toISOString() });
        setInput('');
        setLoading(true);

        try {
            if (IS_BROWSER) throw new Error('AI assistant requires the desktop binary with Ollama.');
            let responseText = '';
            const AI = await import('../../wailsjs/go/services/AIService');
            if (mode() === 'explain') {
                const r = await AI.ExplainError(text);
                responseText = r.Text;
            } else if (mode() === 'generate') {
                const r = await AI.GenerateCommand(text);
                responseText = r.Text;
            } else {
                responseText = await AI.SendMessage(text);
            }

            setOllamaStatus('online');
            pushMessage({ role: 'assistant', content: responseText, timestamp: new Date().toISOString() });
        } catch (e: any) {
            setOllamaStatus('offline');
            const errMsg = e?.toString().includes('ollama')
                ? 'Ollama is not running. Start it with: ollama serve\nThen pull a model: ollama pull llama3'
                : `Error: ${e}`;
            pushMessage({ role: 'error', content: errMsg, timestamp: new Date().toISOString() });
        } finally {
            setLoading(false);
        }
    };

    const modeLabels: Record<Mode, string> = {
        chat: 'CHAT',
        explain: 'EXPLAIN ERROR',
        generate: 'GEN COMMAND',
    };

    const modePlaceholders: Record<Mode, string> = {
        chat: 'Ask anything about security, commands, or your infrastructure...',
        explain: 'Paste an error message to explain...',
        generate: 'Describe what you want to do in plain English...',
    };

    const statusColor = () => {
        switch (ollamaStatus()) {
            case 'online': return '#3fb950';
            case 'offline': return '#f85149';
            default: return '#8b949e';
        }
    };

    const statusLabel = () => {
        switch (ollamaStatus()) {
            case 'online': return 'Ollama online';
            case 'offline': return 'Ollama offline';
            default: return 'Checking...';
        }
    };

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            background: 'var(--surface-0)',
            'font-family': 'var(--font-ui)',
        }}>
            {/* ── Header ─────────────────────────────────────────── */}
            <div style={{
                display: 'flex',
                'align-items': 'center',
                'justify-content': 'space-between',
                padding: '16px 24px',
                'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': '0',
            }}>
                <div>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '10px' }}>
                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '16px', 'font-weight': 800, color: 'var(--text-primary)' }}>
                            AI SHELL ASSISTANT
                        </span>
                        {/* Ollama status badge */}
                        <span style={{
                            display: 'flex',
                            'align-items': 'center',
                            gap: '5px',
                            background: 'var(--surface-2)',
                            border: `1px solid ${statusColor()}`,
                            'border-radius': '4px',
                            padding: '2px 8px',
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            color: statusColor(),
                            'font-weight': 700,
                            'letter-spacing': '0.5px',
                        }}>
                            <span style={{ width: '6px', height: '6px', 'border-radius': '50%', background: statusColor(), display: 'inline-block' }} />
                            {statusLabel()}
                        </span>
                    </div>
                    <p style={{ color: 'var(--text-muted)', 'font-size': '11px', margin: '4px 0 0 0' }}>
                        Powered by local Ollama (llama3). No data leaves your machine.
                    </p>
                </div>

                {/* Mode selector */}
                <div style={{ display: 'flex', gap: '4px' }}>
                    <For each={(['chat', 'explain', 'generate'] as Mode[])}>
                        {(m) => (
                            <button
                                onClick={() => setMode(m)}
                                style={{
                                    background: mode() === m ? 'var(--accent-primary)' : 'var(--surface-2)',
                                    border: `1px solid ${mode() === m ? 'var(--accent-primary)' : 'var(--border-primary)'}`,
                                    color: mode() === m ? '#fff' : 'var(--text-muted)',
                                    padding: '5px 12px',
                                    'border-radius': '4px',
                                    cursor: 'pointer',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '10px',
                                    'font-weight': 700,
                                    'letter-spacing': '0.5px',
                                    transition: 'all 0.15s',
                                }}
                            >
                                {modeLabels[m]}
                            </button>
                        )}
                    </For>
                </div>
            </div>

            {/* ── Offline banner ─────────────────────────────────── */}
            <Show when={ollamaStatus() === 'offline'}>
                <div style={{
                    background: 'rgba(248, 81, 73, 0.1)',
                    border: '1px solid rgba(248, 81, 73, 0.3)',
                    'border-radius': '6px',
                    margin: '12px 24px 0',
                    padding: '12px 16px',
                    'flex-shrink': '0',
                }}>
                    <div style={{ color: '#f85149', 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 700, 'margin-bottom': '6px' }}>
                        ⚠ OLLAMA NOT RUNNING
                    </div>
                    <div style={{ color: 'var(--text-secondary)', 'font-size': '12px', 'line-height': '1.6' }}>
                        The AI assistant requires a local Ollama instance. To start it:
                    </div>
                    <div style={{
                        'margin-top': '8px',
                        background: 'var(--surface-1)',
                        'border-radius': '4px',
                        padding: '8px 12px',
                        'font-family': 'var(--font-mono)',
                        'font-size': '12px',
                        color: '#3fb950',
                    }}>
                        <div>$ ollama serve</div>
                        <div>$ ollama pull llama3</div>
                    </div>
                    <div style={{ color: 'var(--text-muted)', 'font-size': '11px', 'margin-top': '6px' }}>
                        Download Ollama at <span style={{ color: 'var(--accent-primary)' }}>https://ollama.com</span>
                    </div>
                </div>
            </Show>

            {/* ── Message feed ───────────────────────────────────── */}
            <div style={{
                flex: '1',
                overflow: 'auto',
                padding: '20px 24px',
                display: 'flex',
                'flex-direction': 'column',
                gap: '14px',
            }}>
                <Show when={messages().length === 0}>
                    <div style={{
                        margin: 'auto',
                        'text-align': 'center',
                        color: 'var(--text-muted)',
                        'max-width': '420px',
                    }}>
                        <div style={{ 'font-size': '32px', 'margin-bottom': '12px' }}>⚡</div>
                        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '13px', 'font-weight': 700, 'margin-bottom': '8px', color: 'var(--text-secondary)' }}>
                            SOVEREIGN AI SHELL
                        </div>
                        <div style={{ 'font-size': '12px', 'line-height': '1.7' }}>
                            Ask anything: generate shell commands, explain errors, analyse security events, or write automation scripts. All inference runs locally.
                        </div>
                    </div>
                </Show>

                <For each={messages()}>
                    {(msg) => (
                        <div style={{
                            'align-self': msg.role === 'user' ? 'flex-end' : 'flex-start',
                            'max-width': '72%',
                        }}>
                            {/* Role label */}
                            <div style={{
                                'font-family': 'var(--font-mono)',
                                'font-size': '9px',
                                'font-weight': 700,
                                'letter-spacing': '1px',
                                color: msg.role === 'user' ? 'var(--accent-primary)'
                                    : msg.role === 'error' ? '#f85149'
                                    : 'var(--text-muted)',
                                'margin-bottom': '4px',
                                'text-align': msg.role === 'user' ? 'right' : 'left',
                                'padding': '0 4px',
                            }}>
                                {msg.role === 'user' ? 'YOU'
                                    : msg.role === 'error' ? '⚠ ERROR'
                                    : 'AI ASSISTANT'}
                            </div>
                            {/* Bubble */}
                            <div style={{
                                padding: '11px 15px',
                                background: msg.role === 'user' ? 'rgba(87, 139, 255, 0.15)'
                                    : msg.role === 'error' ? 'rgba(248, 81, 73, 0.1)'
                                    : 'var(--surface-1)',
                                border: `1px solid ${msg.role === 'user' ? 'rgba(87,139,255,0.3)'
                                    : msg.role === 'error' ? 'rgba(248,81,73,0.3)'
                                    : 'var(--border-primary)'}`,
                                'border-radius': msg.role === 'user' ? '12px 12px 2px 12px' : '12px 12px 12px 2px',
                                'font-family': msg.role === 'assistant' ? 'var(--font-mono)' : 'var(--font-ui)',
                                'font-size': '13px',
                                'line-height': '1.6',
                                color: msg.role === 'error' ? '#f85149' : 'var(--text-primary)',
                                'white-space': 'pre-wrap',
                                'word-break': 'break-word',
                            }}>
                                {msg.content}
                            </div>
                        </div>
                    )}
                </For>

                <Show when={loading()}>
                    <div style={{ 'align-self': 'flex-start', display: 'flex', 'align-items': 'center', gap: '8px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
                        <span style={{ animation: 'pulse 1.5s infinite' }}>▌</span>
                        AI is thinking...
                    </div>
                </Show>

                <div ref={bottomRef} />
            </div>

            {/* ── Input bar ──────────────────────────────────────── */}
            <div style={{
                'flex-shrink': '0',
                padding: '16px 24px',
                'border-top': '1px solid var(--border-primary)',
                display: 'flex',
                gap: '10px',
                'align-items': 'flex-end',
                background: 'var(--surface-0)',
            }}>
                <div style={{
                    flex: '1',
                    background: 'var(--surface-1)',
                    border: '1px solid var(--border-primary)',
                    'border-radius': '8px',
                    padding: '10px 14px',
                    display: 'flex',
                    'align-items': 'center',
                    gap: '10px',
                }}>
                    <span style={{ color: 'var(--accent-primary)', 'font-family': 'var(--font-mono)', 'font-size': '13px', 'font-weight': 700, 'flex-shrink': '0' }}>
                        {mode() === 'chat' ? '>' : mode() === 'explain' ? '!?' : '$'}
                    </span>
                    <textarea
                        rows={1}
                        value={input()}
                        onInput={(e) => {
                            setInput(e.currentTarget.value);
                            // Auto-expand up to 4 rows
                            e.currentTarget.style.height = 'auto';
                            e.currentTarget.style.height = Math.min(e.currentTarget.scrollHeight, 96) + 'px';
                        }}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter' && !e.shiftKey) {
                                e.preventDefault();
                                handleSend();
                            }
                        }}
                        placeholder={modePlaceholders[mode()]}
                        style={{
                            flex: '1',
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--text-primary)',
                            outline: 'none',
                            'font-size': '13px',
                            resize: 'none',
                            'line-height': '1.5',
                            'font-family': 'var(--font-ui)',
                            overflow: 'hidden',
                        }}
                        disabled={loading()}
                    />
                </div>
                <button
                    onClick={handleSend}
                    disabled={loading() || !input().trim()}
                    style={{
                        background: loading() || !input().trim() ? 'var(--surface-2)' : 'var(--accent-primary)',
                        border: 'none',
                        color: loading() || !input().trim() ? 'var(--text-muted)' : '#fff',
                        padding: '10px 18px',
                        'border-radius': '8px',
                        cursor: loading() || !input().trim() ? 'not-allowed' : 'pointer',
                        'font-family': 'var(--font-mono)',
                        'font-size': '11px',
                        'font-weight': 800,
                        'letter-spacing': '1px',
                        'white-space': 'nowrap',
                        transition: 'all 0.15s',
                    }}
                >
                    {loading() ? '...' : 'SEND'}
                </button>
            </div>
        </div>
    );
};
