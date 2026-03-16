import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const NotesPanel: Component<{ sessionId?: string }> = (props) => {
    const [notes, setNotes] = createSignal<any[]>([]);
    const [runbooks, setRunbooks] = createSignal<any[]>([]);
    const [tab, setTab] = createSignal<'notes' | 'runbooks'>('notes');
    const [editNote, setEditNote] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { GetNotes, GetRunbooks } = await import('../../../wailsjs/go/services/NotesService');
            const [n, r] = await Promise.all([GetNotes(props.sessionId || ''), GetRunbooks(props.sessionId || '')]);
            setNotes(n || []);
            setRunbooks(r || []);
        } catch (e) { console.error('Notes load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const saveNote = async () => {
        const note = editNote();
        if (!note) return;
        try {
            const { SaveNote } = await import('../../../wailsjs/go/services/NotesService');
            await SaveNote(note);
            setEditNote(null);
            await reload();
        } catch (e) { console.error('Save note:', e); }
    };

    const deleteNote = async (id: string) => {
        try {
            const { DeleteNote } = await import('../../../wailsjs/go/services/NotesService');
            await DeleteNote(id);
            await reload();
        } catch (e) { console.error('Delete note:', e); }
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div style="display: flex; gap: 2px; padding: 8px 12px; border-bottom: 1px solid var(--border-primary); align-items: center;">
                <button class={`header-tab ${tab() === 'notes' ? 'active' : ''}`} onClick={() => setTab('notes')} style="font-size: 11px; padding: 4px 10px;">📝 Notes</button>
                <button class={`header-tab ${tab() === 'runbooks' ? 'active' : ''}`} onClick={() => setTab('runbooks')} style="font-size: 11px; padding: 4px 10px;">📖 Runbooks</button>
                <div style="flex: 1;" />
                <button class="action-btn primary" style="padding: 3px 8px; font-size: 10px;" onClick={() => setEditNote({ id: '', title: '', content: '', session_id: props.sessionId || '', tags: [] })}>+ New</button>
            </div>

            <Show when={editNote()}>
                <div style="padding: 8px; border-bottom: 1px solid var(--border-primary);">
                    <input type="text" placeholder="Title" value={editNote()?.title || ''} onInput={(e) => setEditNote((p: any) => ({ ...p, title: e.currentTarget.value }))}
                        style="width: 100%; background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-xs); color: var(--text-primary); padding: 4px 8px; font-size: 12px; outline: none; margin-bottom: 4px;" />
                    <textarea placeholder="Content..." value={editNote()?.content || ''} onInput={(e) => setEditNote((p: any) => ({ ...p, content: e.currentTarget.value }))}
                        style="width: 100%; background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-xs); color: var(--text-primary); padding: 4px 8px; font-size: 12px; outline: none; height: 80px; resize: vertical; font-family: var(--font-mono);" />
                    <div style="display: flex; gap: 4px; margin-top: 4px;">
                        <button class="action-btn primary" style="padding: 3px 10px; font-size: 11px;" onClick={saveNote}>Save</button>
                        <button class="action-btn" style="padding: 3px 10px; font-size: 11px;" onClick={() => setEditNote(null)}>Cancel</button>
                    </div>
                </div>
            </Show>

            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading() && tab() === 'notes'}>
                    <For each={notes()} fallback={<div class="placeholder">No notes yet. Create one to document your sessions.</div>}>
                        {(n) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 8px; margin-bottom: 4px; cursor: pointer;" onClick={() => setEditNote(n)}>
                                <div style="display: flex; justify-content: space-between;">
                                    <span style="font-size: 12px; font-weight: 500; color: var(--text-primary);">{n.title || 'Untitled'}</span>
                                    <button onClick={(e) => { e.stopPropagation(); deleteNote(n.id); }} style="background: none; border: none; color: var(--error); cursor: pointer; font-size: 10px;">✕</button>
                                </div>
                                <div style="font-size: 11px; color: var(--text-muted); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{n.content?.slice(0, 80) || ''}</div>
                            </div>
                        )}
                    </For>
                </Show>
                <Show when={!loading() && tab() === 'runbooks'}>
                    <For each={runbooks()} fallback={<div class="placeholder">No runbooks yet.</div>}>
                        {(r) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 8px; margin-bottom: 4px;">
                                <div style="font-size: 12px; font-weight: 500; color: var(--text-primary);">📖 {r.name || 'Untitled'}</div>
                                <div style="font-size: 10px; color: var(--text-muted); margin-top: 2px;">{r.steps?.length || 0} steps</div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
