import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const WorkspacePanel: Component = () => {
    const [workspaces, setWorkspaces] = createSignal<any[]>([]);
    const [active, setActive] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(true);
    const [newName, setNewName] = createSignal('');

    const reload = async () => {
        try {
            const { GetAll, GetActive } = await import('../../../wailsjs/go/app/WorkspaceService');
            const [all, act] = await Promise.all([GetAll(), GetActive()]);
            setWorkspaces(all || []);
            setActive(act);
        } catch (e) { console.error('Workspace load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const createWorkspace = async () => {
        if (!newName()) return;
        try {
            const { Create } = await import('../../../wailsjs/go/app/WorkspaceService');
            await Create(newName(), '', '');
            setNewName('');
            await reload();
        } catch (e) { console.error('Create workspace:', e); }
    };

    const activateWorkspace = async (id: string) => {
        try {
            const { Activate } = await import('../../../wailsjs/go/app/WorkspaceService');
            await Activate(id);
            await reload();
        } catch (e) { console.error('Activate workspace:', e); }
    };

    const deleteWorkspace = async (id: string) => {
        try {
            const { Delete } = await import('../../../wailsjs/go/app/WorkspaceService');
            await Delete(id);
            await reload();
        } catch (e) { console.error('Delete workspace:', e); }
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div class="drawer-header"><span class="drawer-title">Workspaces</span></div>
            <div style="padding: 8px; display: flex; gap: 4px; border-bottom: 1px solid var(--border-subtle);">
                <input type="text" placeholder="New workspace name" value={newName()} onInput={(e) => setNewName(e.currentTarget.value)}
                    onKeyDown={(e) => e.key === 'Enter' && createWorkspace()}
                    style="flex: 1; background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-xs); color: var(--text-primary); padding: 4px 8px; font-size: 11px; outline: none;" />
                <button class="action-btn primary" style="padding: 2px 8px; font-size: 10px;" onClick={createWorkspace}>+</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading()}>
                    <For each={workspaces()} fallback={<div class="placeholder">No workspaces. Create one to save your terminal layout.</div>}>
                        {(ws) => (
                            <div
                                class={`host-item ${active()?.id === ws.id ? 'selected' : ''}`}
                                onClick={() => activateWorkspace(ws.id)}
                            >
                                <span style="font-size: 14px;">📐</span>
                                <div class="host-info">
                                    <span class="host-label">{ws.name || ws.id}</span>
                                    <span class="host-address">{ws.connections?.length || 0} connections • {ws.created_at ? new Date(ws.created_at).toLocaleDateString() : ''}</span>
                                </div>
                                <Show when={active()?.id === ws.id}>
                                    <span style="color: var(--accent-primary); font-size: 10px;">Active</span>
                                </Show>
                                <button onClick={(e) => { e.stopPropagation(); deleteWorkspace(ws.id); }} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 12px;" title="Delete">✕</button>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
