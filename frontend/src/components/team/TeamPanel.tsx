import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const TeamPanel: Component = () => {
    const [members, setMembers] = createSignal<any[]>([]);
    const [secrets, setSecrets] = createSignal<any[]>([]);
    const [teamName, setTeamName] = createSignal('');
    const [tab, setTab] = createSignal<'members' | 'secrets'>('members');
    const [loading, setLoading] = createSignal(true);
    const [newEmail, setNewEmail] = createSignal('');
    const [newRole, setNewRole] = createSignal('viewer');

    onMount(async () => {
        try {
            const { GetTeamName, ListMembers, ListSecrets } = await import('../../../wailsjs/go/app/TeamService');
            const [tn, m, s] = await Promise.allSettled([GetTeamName(), ListMembers(), ListSecrets()]);
            setTeamName(tn.status === 'fulfilled' ? tn.value : '');
            setMembers(m.status === 'fulfilled' ? (m.value || []) : []);
            setSecrets(s.status === 'fulfilled' ? (s.value || []) : []);
        } catch (e) { console.error('Team load:', e); }
        setLoading(false);
    });

    const addMember = async () => {
        if (!newEmail()) return;
        try {
            const { AddMember } = await import('../../../wailsjs/go/app/TeamService');
            const member = await AddMember(newEmail(), newRole(), newEmail().split('@')[0]);
            setMembers(prev => [...prev, member]);
            setNewEmail('');
        } catch (e) { console.error('Add member:', e); }
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div class="drawer-header">
                <span class="drawer-title">Team{teamName() ? `: ${teamName()}` : ''}</span>
            </div>
            <div style="display: flex; gap: 2px; padding: 4px 8px; border-bottom: 1px solid var(--border-primary);">
                <button class={`header-tab ${tab() === 'members' ? 'active' : ''}`} onClick={() => setTab('members')} style="font-size: 11px; padding: 4px 8px;">👥 Members</button>
                <button class={`header-tab ${tab() === 'secrets' ? 'active' : ''}`} onClick={() => setTab('secrets')} style="font-size: 11px; padding: 4px 8px;">🔑 Shared Secrets</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading() && tab() === 'members'}>
                    <div style="display: flex; gap: 4px; margin-bottom: 8px;">
                        <input type="text" placeholder="Email" value={newEmail()} onInput={(e) => setNewEmail(e.currentTarget.value)}
                            style="flex: 1; background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-xs); color: var(--text-primary); padding: 4px 8px; font-size: 11px; outline: none;" />
                        <select value={newRole()} onChange={(e) => setNewRole(e.currentTarget.value)}
                            style="background: var(--bg-tertiary); color: var(--text-secondary); border: 1px solid var(--border-primary); border-radius: var(--radius-xs); padding: 2px 4px; font-size: 10px;">
                            <option value="admin">Admin</option><option value="editor">Editor</option><option value="viewer">Viewer</option>
                        </select>
                        <button class="action-btn primary" style="padding: 2px 8px; font-size: 10px;" onClick={addMember}>+</button>
                    </div>
                    <For each={members()} fallback={<div class="placeholder">No team members</div>}>
                        {(m) => (
                            <div style="display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: var(--radius-xs); margin-bottom: 2px;">
                                <div class="user-avatar" style="width: 24px; height: 24px; font-size: 9px;">{(m.name || m.email || '?')[0].toUpperCase()}</div>
                                <div style="flex: 1;">
                                    <div style="font-size: 12px; color: var(--text-primary);">{m.name || m.email}</div>
                                    <div style="font-size: 10px; color: var(--text-muted);">{m.role || 'member'}</div>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
                <Show when={!loading() && tab() === 'secrets'}>
                    <For each={secrets()} fallback={<div class="placeholder">No shared secrets</div>}>
                        {(s) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 6px 8px; margin-bottom: 4px;">
                                <div style="font-size: 12px; color: var(--text-primary);">🔑 {s.name || s.key}</div>
                                <div style="font-size: 10px; color: var(--text-muted);">{s.type || 'credential'} • {s.created_at ? new Date(s.created_at).toLocaleDateString() : ''}</div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
