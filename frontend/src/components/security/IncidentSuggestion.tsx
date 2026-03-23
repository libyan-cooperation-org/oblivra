import { Component, Show, createSignal, onMount, onCleanup } from 'solid-js';
import { subscribe } from '@core/bridge';
import { IS_BROWSER } from '@core/context';

interface Suggestion {
    host_id: string;
    session_id: string;
    snippet_id: string;
    title: string;
    command: string;
    reason: string;
}

export const IncidentSuggestion: Component = () => {
    const [suggestion, setSuggestion] = createSignal<Suggestion | null>(null);
    const [executing, setExecuting] = createSignal(false);

    let unsubSuggestion: (() => void) | undefined;

    onMount(() => {
        // subscribe() is a no-op in browser mode — suggestions only fire from desktop backend
        unsubSuggestion = subscribe('security.suggestion', (data: Suggestion) => {
            setSuggestion(data);
            setTimeout(() => setSuggestion(null), 15000);
        });
    });

    onCleanup(() => unsubSuggestion?.());

    const handleExecute = async () => {
        const s = suggestion();
        if (!s || IS_BROWSER) return;

        setExecuting(true);
        try {
            const { ExecuteSnippet } = await import('../../../wailsjs/go/services/SnippetService');
            await ExecuteSnippet(s.snippet_id, s.session_id, {}, true);
            setSuggestion(null);
        } catch (err) {
            console.error('Failed to execute suggestion:', err);
            alert('Execution failed: ' + err);
        } finally {
            setExecuting(false);
        }
    };

    return (
        <Show when={suggestion()}>
            <div class="incident-suggestion fade-in">
                <div class="suggestion-header">
                    <span class="suggestion-icon">🛡️</span>
                    <div class="suggestion-meta">
                        <h4>Security Playbook Suggestion</h4>
                        <p>{suggestion()?.reason}</p>
                    </div>
                </div>

                <div class="suggestion-content">
                    <div class="suggestion-snippet">
                        <span class="snippet-label">Recommended:</span>
                        <span class="snippet-title">{suggestion()?.title}</span>
                    </div>
                    <code class="suggestion-cmd">{suggestion()?.command}</code>
                </div>

                <div class="suggestion-actions">
                    <button class="sm-btn" onClick={() => setSuggestion(null)}>Ignore</button>
                    <button
                        class="sm-btn primary"
                        onClick={handleExecute}
                        disabled={executing()}
                    >
                        {executing() ? 'Executing...' : 'Run Playbook'}
                    </button>
                </div>
            </div>
        </Show>
    );
};
