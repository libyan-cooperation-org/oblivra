import { Component, createSignal, For, onMount, Show } from 'solid-js';
import { GetChatHistory, SendMessage } from '../../wailsjs/go/app/AIService';

interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string;
    timestamp: string;
}

export const AIAssistantPage: Component = () => {
    const [messages, setMessages] = createSignal<Message[]>([]);
    const [input, setInput] = createSignal('');
    const [loading, setLoading] = createSignal(false);

    onMount(async () => {
        try {
            const history = await GetChatHistory();
            setMessages(history || []);
        } catch (e) {
            console.error('Failed to load chat history:', e);
        }
    });

    const handleSend = async () => {
        if (!input().trim() || loading()) return;
        
        const userMsg: Message = {
            role: 'user',
            content: input(),
            timestamp: new Date().toISOString()
        };
        
        setMessages([...messages(), userMsg]);
        const currentInput = input();
        setInput('');
        setLoading(true);

        try {
            const response = await SendMessage(currentInput);
            const assistantMsg: Message = {
                role: 'assistant',
                content: response,
                timestamp: new Date().toISOString()
            };
            setMessages([...messages(), assistantMsg]);
        } catch (e) {
            console.error('AI Error:', e);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div class="ai-page" style={{ display: 'flex', 'flex-direction': 'column', height: '100%', padding: '24px', background: 'var(--surface-0)' }}>
            <div class="ai-header" style={{ 'margin-bottom': '24px' }}>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800 }}>AI Command Assistant</h1>
                <p style={{ color: 'var(--text-muted)', 'font-size': '12px' }}>Intelligent shell assistance and autonomous security analysis.</p>
            </div>

            <div class="chat-container" style={{ flex: 1, overflow: 'auto', display: 'flex', 'flex-direction': 'column', gap: '16px', 'padding-bottom': '20px' }}>
                    <For each={messages()}>
                        {(msg) => (
                            <div style={{
                                'align-self': msg.role === 'user' ? 'flex-end' : 'flex-start',
                                'max-width': '70%',
                                padding: '12px 16px',
                            background: msg.role === 'user' ? 'var(--accent-primary)' : 'var(--surface-1)',
                            color: msg.role === 'user' ? '#fff' : 'var(--text-primary)',
                            'border-radius': '12px',
                            'font-family': msg.role === 'assistant' ? 'var(--font-mono)' : 'inherit',
                            'font-size': '13px',
                            'line-height': '1.5',
                            border: msg.role === 'assistant' ? '1px solid var(--border-primary)' : 'none'
                        }}>
                            {msg.content}
                        </div>
                    )}
                </For>
                <Show when={loading()}>
                    <div style={{ color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>AI is thinking...</div>
                </Show>
            </div>

            <div class="input-container" style={{ display: 'flex', gap: '12px', padding: '16px', background: 'var(--surface-1)', 'border-radius': '8px', border: '1px solid var(--border-primary)' }}>
                <input
                    type="text"
                    value={input()}
                    onInput={(e) => setInput(e.currentTarget.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSend()}
                    placeholder="Ask for command help, security analysis, or automation..."
                    style={{ flex: 1, background: 'transparent', border: 'none', color: 'var(--text-primary)', outline: 'none', 'font-size': '14px' }}
                />
                <button 
                    onClick={handleSend}
                    style={{ background: 'var(--accent-primary)', border: 'none', color: '#fff', padding: '6px 16px', 'border-radius': '4px', cursor: 'pointer', 'font-weight': 600 }}
                >
                    SEND
                </button>
            </div>
        </div>
    );
};
