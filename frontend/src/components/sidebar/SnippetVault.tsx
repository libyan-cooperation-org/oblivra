import { Component, createSignal, createEffect, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';

export const SnippetVault: Component = () => {
    const [state] = useApp();
    const [snippets, setSnippets] = createSignal<any[]>([]);
    const [isCreating, setIsCreating] = createSignal(false);

    // Form fields
    const [title, setTitle] = createSignal('');
    const [command, setCommand] = createSignal('');
    const [description, setDescription] = createSignal('');

    // Execution state
    const [runningSnippetId, setRunningSnippetId] = createSignal<string | null>(null);
    const [inputVariables, setInputVariables] = createSignal<Record<string, string>>({});
    const [requiredVariables, setRequiredVariables] = createSignal<string[]>([]);
    const [autoSudo, setAutoSudo] = createSignal(false);

    const loadSnippets = async () => {
        if (IS_BROWSER) return;
        try {
            const { List } = await import('../../../wailsjs/go/services/SnippetService');
            setSnippets(await List() || []);
        } catch (err) { console.error('Failed to load snippets', err); }
    };

    createEffect(() => { loadSnippets(); });

    const handleSave = async () => {
        if (!title() || !command() || IS_BROWSER) return;
        try {
            const { ExtractVariables, Create } = await import('../../../wailsjs/go/services/SnippetService');
            const extractedVars = await ExtractVariables(command());
            await Create(title(), command(), description(), [], extractedVars || []);
            setIsCreating(false); setTitle(''); setCommand(''); setDescription('');
            loadSnippets();
        } catch (err) { console.error('Snippet save failed', err); }
    };

    const handleDelete = async (id: string) => {
        if (IS_BROWSER || !confirm('Delete this snippet?')) return;
        try {
            const { Delete } = await import('../../../wailsjs/go/services/SnippetService');
            await Delete(id); loadSnippets();
        } catch (err) { console.error('Delete failed', err); }
    };

    const promptExecution = async (snippet: any) => {
        if (!state.activeSessionId) {
            alert("Please open an active terminal session first.");
            return;
        }

        if (snippet.variables && snippet.variables.length > 0) {
            setRunningSnippetId(snippet.id);
            setRequiredVariables(snippet.variables);
            const initialMap: Record<string, string> = {};
            snippet.variables.forEach((v: string) => initialMap[v] = '');
            setInputVariables(initialMap);
        } else {
            // Run instantly if no variables required
            await runConfirmedSnippet(snippet.id, {});
        }
    };

    const runConfirmedSnippet = async (id: string, vars: Record<string, string>) => {
        if (IS_BROWSER) return;
        try {
            const { ExecuteSnippet } = await import('../../../wailsjs/go/services/SnippetService');
            await ExecuteSnippet(id, state.activeSessionId!, vars, autoSudo());
            setRunningSnippetId(null);
        } catch (err) {
            console.error('Snippet execution failed', err);
            alert('Execution failed. See console.');
        }
    };

    return (
        <div class="snippet-vault" style="height: 100%; display: flex; flex-direction: column; background: var(--bg-surface); color: var(--text-primary); padding: 16px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;">
                <h2 style="margin: 0; font-size: 16px; font-weight: 500;">Snippet Vault</h2>
                <button class="action-btn" onClick={() => setIsCreating(!isCreating())}>
                    {isCreating() ? 'Cancel' : '+ New'}
                </button>
            </div>

            <Show when={isCreating()}>
                <div style="background: var(--bg-primary); border: 1px solid var(--border-primary); padding: 16px; border-radius: 6px; margin-bottom: 16px;">
                    <div class="form-group" style="margin-bottom: 12px;">
                        <label style="display: block; font-size: 12px; margin-bottom: 4px; color: var(--text-secondary);">Title</label>
                        <input type="text" value={title()} onInput={(e) => setTitle(e.currentTarget.value)} class="form-control" placeholder="e.g. Restart Docker" style="width: 100%; box-sizing: border-box;" />
                    </div>
                    <div class="form-group" style="margin-bottom: 12px;">
                        <label style="display: block; font-size: 12px; margin-bottom: 4px; color: var(--text-secondary);">Command (Playbook Mode Enabled)</label>
                        <p style="font-size: 10px; color: var(--text-accent); margin: 0 0 6px 0;">Use <code>{'{{var}}'}</code> for basic injection, or Go Templates for logic: <code>{'{{if eq .os "ubuntu"}}apt{{else}}yum{{end}}'}</code></p>
                        <textarea value={command()} onInput={(e) => setCommand(e.currentTarget.value)} class="form-control" style="width: 100%; min-height: 80px; box-sizing: border-box; font-family: monospace;" placeholder={'{{if eq .os "ubuntu"}}\n  apt-get update\n{{else}}\n  yum update\n{{end}}'} />
                    </div>
                    <div class="form-group" style="margin-bottom: 12px;">
                        <label style="display: block; font-size: 12px; margin-bottom: 4px; color: var(--text-secondary);">Description</label>
                        <input type="text" value={description()} onInput={(e) => setDescription(e.currentTarget.value)} class="form-control" placeholder="Optional details..." style="width: 100%; box-sizing: border-box;" />
                    </div>
                    <button class="btn btn-primary" style="width: 100%; padding: 8px;" onClick={handleSave}>Save Snippet</button>
                </div>
            </Show>

            <div style="flex: 1; overflow-y: auto;">
                <Show when={snippets().length === 0 && !isCreating()}>
                    <div style="text-align: center; color: var(--text-secondary); margin-top: 32px; font-size: 13px;">
                        No snippets found. Create one to quickly inject commands into active terminals.
                    </div>
                </Show>

                <For each={snippets()}>
                    {(snippet) => (
                        <div style="background: var(--bg-primary); border: 1px solid var(--border-primary); padding: 12px; border-radius: 6px; margin-bottom: 12px;">
                            <Show when={runningSnippetId() === snippet.id} fallback={
                                <>
                                    <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                        <b style="font-size: 14px;">{snippet.title}</b>
                                        <button class="action-btn danger" style="padding: 2px 6px; font-size: 11px;" onClick={() => handleDelete(snippet.id)}>✕</button>
                                    </div>
                                    <p style="color: var(--text-secondary); font-size: 12px; margin: 4px 0 8px;">{snippet.description}</p>
                                    <code style="display: block; background: #0d1117; padding: 8px; border-radius: 4px; font-size: 11px; margin-bottom: 12px; white-space: pre-wrap; word-break: break-all;">
                                        {snippet.command}
                                    </code>
                                    <button class="btn btn-primary" style="width: 100%; padding: 6px; font-size: 12px;" onClick={() => promptExecution(snippet)}>
                                        Inject to Terminal
                                    </button>
                                </>
                            }>
                                {/* Variable Input Form */}
                                <div style="background: var(--bg-surface); border: 1px solid var(--border-primary); padding: 12px; border-radius: 6px;">
                                    <b style="font-size: 13px; margin-bottom: 8px; display: block; color: var(--primary-color);">Execute Playbook: {snippet.title}</b>
                                    <p style="font-size: 11px; color: var(--text-secondary); margin-bottom: 12px;">Provide values for playbook variables to evaluate conditions and inject this snippet.</p>
                                    <For each={requiredVariables()}>
                                        {(v) => (
                                            <div style="margin-bottom: 8px;">
                                                <label style="font-size: 11px; color: var(--text-secondary);">{v}</label>
                                                <input
                                                    type="text"
                                                    class="form-control"
                                                    style="width: 100%; padding: 4px;"
                                                    value={inputVariables()[v]}
                                                    onInput={(e) => setInputVariables({ ...inputVariables(), [v]: e.currentTarget.value })}
                                                />
                                            </div>
                                        )}
                                    </For>

                                    <label style="display: flex; align-items: center; gap: 8px; font-size: 11px; margin-bottom: 12px; margin-top: 12px;">
                                        <input type="checkbox" checked={autoSudo()} onChange={(e) => setAutoSudo(e.currentTarget.checked)} />
                                        Auto-prefix with 'sudo'
                                    </label>

                                    <div style="display: flex; gap: 8px;">
                                        <button class="btn btn-primary" style="flex: 1; padding: 6px; font-size: 12px;" onClick={() => runConfirmedSnippet(snippet.id, inputVariables())}>Run</button>
                                        <button class="action-btn" style="flex: 1; padding: 6px; font-size: 12px;" onClick={() => setRunningSnippetId(null)}>Cancel</button>
                                    </div>
                                </div>
                            </Show>
                        </div>
                    )}
                </For>
            </div>
        </div>
    );
};
