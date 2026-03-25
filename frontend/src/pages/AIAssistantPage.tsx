import { Component, createSignal, For, onMount, Show, createMemo } from 'solid-js';
import { 
    PageLayout, 
    TabBar, 
    StatusDot, 
    Notice, 
    Button,
    CodeBlock,
    LoadingState
} from '@components/ui';
import { IS_BROWSER } from '@core/context';
import '../styles/ai-assistant.css';

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
        bottomRef?.scrollIntoView({ behavior: 'smooth' });
    });

    const pushMessage = (msg: Message) => {
        setMessages(prev => [...prev, msg]);
        setTimeout(() => bottomRef?.scrollIntoView({ behavior: 'smooth' }), 50);
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
                ? 'Ollama is not running. Start it with:\n$ ollama serve\n$ ollama pull llama3'
                : `Error: ${e}`;
            pushMessage({ role: 'error', content: errMsg, timestamp: new Date().toISOString() });
        } finally {
            setLoading(false);
        }
    };

    const tabs = [
        { id: 'chat',     label: 'CHAT' },
        { id: 'explain',  label: 'EXPLAIN ERROR' },
        { id: 'generate', label: 'GEN COMMAND' }
    ];

    const placeholders: Record<Mode, string> = {
        chat: 'Ask anything about security, commands, or your infrastructure...',
        explain: 'Paste an error message to explain...',
        generate: 'Describe what you want to do in plain English...',
    };

    const statusProps = createMemo(() => ({
        color: ollamaStatus() === 'online' ? 'var(--status-online)' : 'var(--alert-critical)',
        label: ollamaStatus() === 'online' ? 'OLLAMA:ONLINE' : 'OLLAMA:OFFLINE'
    }));

    return (
        <PageLayout
            title="AI Shell Assistant"
            subtitle="LOCAL_INFERENCE_OLLAMA_LLAMA3"
            noPadding
            actions={
                <div style="display: flex; align-items: center; gap: 8px;">
                    <div style={{
                        display: 'flex', 'align-items': 'center', gap: '8px', 
                        background: 'var(--surface-3)', padding: '4px 10px', 'border-radius': 'var(--radius-full)',
                        border: '1px solid var(--border-primary)'
                    }}>
                        <StatusDot status={ollamaStatus() === 'online' ? 'online' : 'offline'} />
                        <span style={{ 
                            'font-family': 'var(--font-mono)', 'font-size': '10px', 
                            color: statusProps().color, 'font-weight': 800 
                        }}>{statusProps().label}</span>
                    </div>
                    <TabBar 
                        tabs={tabs} 
                        active={mode()} 
                        onSelect={(id) => setMode(id as Mode)} 
                        class="ai-mode-selector"
                    />
                </div>
            }
        >
            {/* Offline Alert */}
            <Show when={ollamaStatus() === 'offline'}>
                <div style="padding: var(--gap-md) var(--gap-xl); background: var(--surface-1); border-bottom: 1px solid var(--border-primary);">
                    <Notice level="error">
                        <div>
                            <div style="font-weight: 800; font-size: 11px; margin-bottom: 4px; font-family: var(--font-mono);">⚠ OLLAMA_NOT_RUNNING</div>
                            <div style="font-size: 13px; color: var(--text-secondary); line-height: 1.5;">
                                AI capabilities disabled. Run <code style="color: var(--status-online)">ollama serve</code> to activate.
                            </div>
                        </div>
                    </Notice>
                </div>
            </Show>

            {/* Chat Feed */}
            <div class="ai-chat-feed">
                <Show when={messages().length === 0}>
                    <div class="ai-empty-state">
                        <div style="font-size: 32px; margin-bottom: 12px; opacity: 0.5;">⚡</div>
                        <div style="font-family: var(--font-mono); font-size: 14px; font-weight: 800; color: var(--text-heading); margin-bottom: 8px;">
                            SOVEREIGN AI SHELL v1.0
                        </div>
                        <div style="font-size: 13px; line-height: 1.6; color: var(--text-muted); padding: 0 var(--gap-xl);">
                            Generate commands, explain log errors, or write automation scripts. 
                            All inference is local-first. No telemetrics.
                        </div>
                    </div>
                </Show>

                <For each={messages()}>
                    {(msg) => (
                        <div class={`msg-bubble-wrap ${msg.role}`}>
                            <div class={`msg-role ${msg.role}`}>
                                {msg.role === 'user' ? 'USER_PROMPT' : msg.role === 'assistant' ? 'AI_ASSISTANT' : 'SYSTEM_EXCEPTION'}
                            </div>
                            <div class="msg-bubble">
                                {msg.content}
                            </div>
                        </div>
                    )}
                </For>

                <Show when={loading()}>
                    <div class="ai-thinking">
                        <span class="ai-cursor" />
                        AI_IS_CONSULTING_LOCAL_GRAPHS...
                    </div>
                </Show>

                <div ref={bottomRef} style="height: 1px;" />
            </div>

            {/* Input Bar */}
            <div class="ai-input-wrap">
                <div class="ai-input-inner">
                    <span class="ai-input-prompt">
                        {mode() === 'chat' ? '>' : mode() === 'explain' ? '!?' : '$'}
                    </span>
                    <textarea
                        rows={1}
                        value={input()}
                        onInput={(e) => {
                            setInput(e.currentTarget.value);
                            e.currentTarget.style.height = 'auto';
                            e.currentTarget.style.height = Math.min(e.currentTarget.scrollHeight, 120) + 'px';
                        }}
                        onKeyDown={(e) => {
                            if (e.key === 'Enter' && !e.shiftKey) {
                                e.preventDefault();
                                handleSend();
                            }
                        }}
                        placeholder={placeholders[mode()]}
                        disabled={loading()}
                    />
                </div>
                <Button 
                    variant="primary" 
                    onClick={handleSend} 
                    disabled={loading() || !input().trim()}
                    loading={loading()}
                >
                    SEND_REQ
                </Button>
            </div>
        </PageLayout>
    );
};
