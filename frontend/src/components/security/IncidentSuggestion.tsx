import { Component, Show, createSignal, onMount, onCleanup } from 'solid-js';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { ExecuteSnippet } from '../../../wailsjs/go/services/SnippetService';

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

    onMount(() => {
        EventsOn('security.suggestion', (data: Suggestion) => {
            setSuggestion(data);
            // Auto-hide after 15 seconds if ignored
            setTimeout(() => setSuggestion(null), 15000);
        });
    });

    onCleanup(() => {
        EventsOff('security.suggestion');
    });

    const handleExecute = async () => {
        const s = suggestion();
        if (!s) return;

        setExecuting(true);
        try {
            await ExecuteSnippet(s.snippet_id, s.session_id, {}, true);
            setSuggestion(null);
        } catch (err) {
            console.error("Failed to execute suggestion:", err);
            alert("Execution failed: " + err);
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
