import { Component, createSignal, For, Show, createMemo, onMount } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface Snippet {
  id: string;
  title: string;
  command: string;
  description: string;
  tags: string[];
  variables: string[];
  use_count: number;
}

interface SnippetManagerProps {
  onExecute: (command: string) => void;
  onClose?: () => void;
}

export const SnippetManager: Component<SnippetManagerProps> = (props) => {
  const [snippets, setSnippets] = createSignal<Snippet[]>([]);
  const [searchQuery, setSearchQuery] = createSignal('');
  const [editingSnippet, setEditingSnippet] = createSignal<Snippet | null>(null);
  const [showForm, setShowForm] = createSignal(false);
  const [variableValues, setVariableValues] = createSignal<Record<string, string>>({});
  const [expandedId, setExpandedId] = createSignal<string | null>(null);
  const [autoSudo, setAutoSudo] = createSignal(false);

  // Form state
  const [formTitle, setFormTitle] = createSignal('');
  const [formCommand, setFormCommand] = createSignal('');
  const [formDescription, setFormDescription] = createSignal('');
  const [formTags, setFormTags] = createSignal('');

  onMount(async () => {
    if (IS_BROWSER) return;
    try {
      const { List } = await import('../../../wailsjs/go/services/SnippetService');
      setSnippets(await List() || []);
    } catch (e) { console.error('SnippetManager load:', e); }
  });

  const filtered = createMemo(() => {
    const q = searchQuery().toLowerCase();
    if (!q) return snippets();
    return snippets().filter(
      s => s.title.toLowerCase().includes(q) ||
        s.command.toLowerCase().includes(q) ||
        s.tags.some(t => t.toLowerCase().includes(q))
    );
  });

  const detectedVariables = createMemo(() => {
    const re = /\{\{(\w+)\}\}/g;
    const vars: string[] = [];
    let match;
    while ((match = re.exec(formCommand())) !== null) {
      if (!vars.includes(match[1])) vars.push(match[1]);
    }
    return vars;
  });

  const executeSnippet = (snippet: Snippet) => {
    let command = snippet.command;
    const vars = variableValues();
    for (const [key, value] of Object.entries(vars)) {
      command = command.replaceAll(`{{${key}}}`, value);
    }
    const unresolved = command.match(/\{\{(\w+)\}\}/g);
    if (unresolved) { setExpandedId(snippet.id); return; }
    props.onExecute(command);
    setVariableValues({});
    setExpandedId(null);
  };

  const handleExport = async () => {
    if (IS_BROWSER) return;
    try {
      const { ExportJSON } = await import('../../../wailsjs/go/services/SnippetService');
      const data = await ExportJSON();
      const blob = new Blob([new Uint8Array(data)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = 'snippets.json'; a.click();
    } catch (e) { console.error('Export failed:', e); }
  };

  const handleImport = async () => {
    if (IS_BROWSER) return;
    const input = document.createElement('input');
    input.type = 'file'; input.accept = '.json';
    input.onchange = async (e: any) => {
      const file = e.target.files[0];
      const reader = new FileReader();
      reader.onload = async (re: any) => {
        try {
          const { ImportJSON, List } = await import('../../../wailsjs/go/services/SnippetService');
          const data = Array.from(new Uint8Array(re.target.result));
          await ImportJSON(data);
          setSnippets(await List() || []);
        } catch (e) { console.error('Import failed:', e); }
      };
      reader.readAsArrayBuffer(file);
    };
    input.click();
  };

  const saveSnippet = async () => {
    resetForm();
    setShowForm(false);
  };

  const resetForm = () => {
    setFormTitle('');
    setFormCommand('');
    setFormDescription('');
    setFormTags('');
    setEditingSnippet(null);
  };

  const startEdit = (snippet: Snippet) => {
    setFormTitle(snippet.title);
    setFormCommand(snippet.command);
    setFormDescription(snippet.description);
    setFormTags(snippet.tags.join(', '));
    setEditingSnippet(snippet);
    setShowForm(true);
  };

  return (
    <div class="snippet-manager">
      <div class="sm-header">
        <h3>Command Snippets</h3>
        <div class="sm-actions">
          <button class="sm-btn" onClick={handleExport}>Export</button>
          <button class="sm-btn" onClick={handleImport}>Import</button>
          <button class="sm-btn primary" onClick={() => { resetForm(); setShowForm(true); }}>+ New</button>
        </div>
      </div>

      <div class="sm-search">
        <input type="text" placeholder="Search snippets..." value={searchQuery()}
          onInput={(e) => setSearchQuery(e.currentTarget.value)} />
      </div>

      <Show when={showForm()}>
        <div class="sm-form">
          <div class="form-group">
            <label>Title</label>
            <input type="text" value={formTitle()} onInput={(e) => setFormTitle(e.currentTarget.value)} placeholder="e.g., Docker Cleanup" />
          </div>
          <div class="form-group">
            <label>Command</label>
            <textarea value={formCommand()} onInput={(e) => setFormCommand(e.currentTarget.value)}
              placeholder="docker system prune -af --volumes" rows={3} />
            <Show when={detectedVariables().length > 0}>
              <div class="detected-vars">Variables: {detectedVariables().map((v: string) => `{{${v}}}`).join(', ')}</div>
            </Show>
          </div>
          <div class="form-group">
            <label>Description</label>
            <input type="text" value={formDescription()} onInput={(e) => setFormDescription(e.currentTarget.value)} placeholder="Optional description" />
          </div>
          <div class="form-group">
            <label>Tags (comma separated)</label>
            <input type="text" value={formTags()} onInput={(e) => setFormTags(e.currentTarget.value)} placeholder="docker, cleanup, maintenance" />
          </div>
          <div class="form-actions">
            <button class="sm-btn" onClick={() => { setShowForm(false); resetForm(); }}>Cancel</button>
            <button class="sm-btn primary" onClick={saveSnippet}>{editingSnippet() ? 'Update' : 'Save'}</button>
          </div>
        </div>
      </Show>

      <div class="sm-list">
        <For each={filtered()}>
          {(snippet) => (
            <div class="snippet-card">
              <div class="snippet-header" onClick={() => setExpandedId(expandedId() === snippet.id ? null : snippet.id)}>
                <div class="snippet-title-row">
                  <span class="snippet-title">{snippet.title}</span>
                  <span class="snippet-uses">{snippet.use_count} uses</span>
                </div>
                <code class="snippet-command">{snippet.command}</code>
                <Show when={snippet.tags.length > 0}>
                  <div class="snippet-tags">
                    <For each={snippet.tags}>{(tag) => <span class="tag">{tag}</span>}</For>
                  </div>
                </Show>
              </div>
              <Show when={expandedId() === snippet.id}>
                <div class="snippet-expanded">
                  <Show when={snippet.description}><p class="snippet-desc">{snippet.description}</p></Show>
                  <Show when={snippet.variables.length > 0}>
                    <div class="snippet-variables">
                      <For each={snippet.variables}>
                        {(varName) => (
                          <div class="var-input">
                            <label>{`{{${varName}}}`}</label>
                            <input type="text" placeholder={varName} value={variableValues()[varName] || ''}
                              onInput={(e) => setVariableValues(prev => ({ ...prev, [varName]: e.currentTarget.value }))} />
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                  <div class="snippet-options">
                    <label class="checkbox-item">
                      <input type="checkbox" checked={autoSudo()} onChange={(e) => setAutoSudo(e.currentTarget.checked)} />
                      <span>Auto-Sudo</span>
                    </label>
                  </div>
                  <div class="snippet-actions">
                    <button class="sm-btn primary" onClick={() => executeSnippet(snippet)}>▶ Execute</button>
                    <button class="sm-btn" onClick={() => navigator.clipboard.writeText(snippet.command)}>📋 Copy</button>
                    <button class="sm-btn" onClick={() => startEdit(snippet)}>✏️ Edit</button>
                    <button class="sm-btn danger" onClick={() => { /* delete */ }}>🗑️</button>
                  </div>
                </div>
              </Show>
            </div>
          )}
        </For>
        <Show when={filtered().length === 0}>
          <div class="sm-empty">
            <p>{searchQuery() ? 'No matching snippets' : 'No snippets yet'}</p>
            <p class="sm-hint">Create reusable commands with {'{{variables}}'} for quick execution</p>
          </div>
        </Show>
      </div>
    </div>
  );
};
