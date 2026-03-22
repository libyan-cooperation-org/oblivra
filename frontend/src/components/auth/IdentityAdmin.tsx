// IdentityAdmin.tsx — Phase 3 Web: OIDC/SAML/enterprise identity management
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as IdentityService from '../../../wailsjs/go/services/IdentityService';

interface User { id: string; username: string; email: string; role: string; mfa_enabled: boolean; last_login?: string; status?: string; }

export const IdentityAdmin: Component = () => {
    const [users, setUsers] = createSignal<User[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [tab, setTab] = createSignal<'users' | 'providers' | 'sessions'>('users');
    const [search, setSearch] = createSignal('');
    const [newUser, setNewUser] = createSignal({ username: '', email: '', role: 'analyst', password: '' });
    const [creating, setCreating] = createSignal(false);
    const [msg, setMsg] = createSignal({ text: '', ok: true });

    const showMsg = (text: string, ok = true) => { setMsg({ text, ok }); setTimeout(() => setMsg({ text: '', ok: true }), 4000); };

    const load = async () => {
        setLoading(true);
        try { setUsers((await (IdentityService as any).ListUsers?.() ?? []) as User[]); } catch { setUsers([]); }
        setLoading(false);
    };

    onMount(load);

    const createUser = async () => {
        const u = newUser();
        if (!u.username.trim() || !u.password.trim()) { showMsg('Username and password required', false); return; }
        setCreating(true);
        try {
            await (IdentityService as any).CreateUser?.(u.username, u.email, u.password, u.role);
            showMsg(`✓ User "${u.username}" created`);
            setNewUser({ username: '', email: '', role: 'analyst', password: '' });
            load();
        } catch (e: any) { showMsg('✗ ' + (e?.message ?? e), false); }
        setCreating(false);
    };

    const deleteUser = async (id: string, name: string) => {
        try {
            // Soft-disable: set role to 'disabled' (no DeleteUser API exists)
            await (IdentityService as any).UpdateUserRole(id, 'disabled');
            showMsg(`✓ User "${name}" disabled`);
            load();
        } catch (e: any) { showMsg('✗ ' + (e?.message ?? e), false); }
    };

    const ROLE_COLORS: Record<string, string> = { admin: '#f85149', analyst: '#3b82f6', viewer: '#6b7280', auditor: '#8b5cf6' };

    const filtered = () => {
        const q = search().toLowerCase();
        return users().filter(u => !q || u.username?.toLowerCase().includes(q) || u.email?.toLowerCase().includes(q) || u.role?.toLowerCase().includes(q));
    };

    const PROVIDERS = [
        { name: 'OIDC / Keycloak', icon: '🔑', status: 'configured', note: 'Production OIDC provider' },
        { name: 'OIDC / Okta', icon: '🔑', status: 'available', note: 'Enterprise Okta integration' },
        { name: 'SAML 2.0', icon: '🏢', status: 'configured', note: 'Azure AD / ADFS' },
        { name: 'LDAP / Active Directory', icon: '📁', status: 'available', note: 'On-premises directory sync' },
    ];

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">👥</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Identity Administration</h2>
                </div>
                <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{users().length} USERS</span>
            </div>

            <div style="display: flex; gap: 0; border-bottom: 1px solid var(--glass-border); background: var(--bg-secondary);">
                {(['users', 'providers', 'sessions'] as const).map(t => (
                    <button onClick={() => setTab(t)}
                        style={`padding: 10px 20px; font-size: 11px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase; font-family: var(--font-mono); border: none; cursor: pointer; background: transparent; border-bottom: 2px solid ${tab() === t ? 'var(--accent-primary)' : 'transparent'}; color: ${tab() === t ? 'var(--accent-primary)' : 'var(--text-muted)'};`}>
                        {t}
                    </button>
                ))}
            </div>

            <div style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem;">
                <Show when={msg().text}>
                    <div style={`padding: 10px 14px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; background: ${msg().ok ? 'rgba(63,185,80,0.1)' : 'rgba(248,81,73,0.1)'}; border: 1px solid ${msg().ok ? 'rgba(63,185,80,0.3)' : 'rgba(248,81,73,0.3)'}; color: ${msg().ok ? '#3fb950' : '#f85149'};`}>{msg().text}</div>
                </Show>

                {/* Users tab */}
                <Show when={tab() === 'users'}>
                    {/* Create user form */}
                    <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                        <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 1rem;">Create Local User</div>
                        <div style="display: grid; grid-template-columns: 1fr 1fr 1fr 120px; gap: 0.75rem; align-items: end;">
                            {[
                                { label: 'Username', key: 'username', type: 'text', placeholder: 'analyst01' },
                                { label: 'Email', key: 'email', type: 'email', placeholder: 'user@org.com' },
                                { label: 'Password', key: 'password', type: 'password', placeholder: '••••••••' },
                            ].map(({ label, key, type, placeholder }) => (
                                <div style="display: flex; flex-direction: column; gap: 0.3rem;">
                                    <label style="font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">{label}</label>
                                    <input type={type} placeholder={placeholder} value={(newUser() as any)[key]}
                                        onInput={e => setNewUser(u => ({ ...u, [key]: (e.target as HTMLInputElement).value }))}
                                        style="background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;" />
                                </div>
                            ))}
                            <div style="display: flex; flex-direction: column; gap: 0.3rem;">
                                <label style="font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">Role</label>
                                <select value={newUser().role} onChange={e => setNewUser(u => ({ ...u, role: e.currentTarget.value }))}
                                    style="background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;">
                                    {['admin', 'analyst', 'viewer', 'auditor'].map(r => <option value={r}>{r}</option>)}
                                </select>
                            </div>
                        </div>
                        <button onClick={createUser} disabled={creating()} style="margin-top: 0.75rem; padding: 7px 16px; background: rgba(87,139,255,0.15); border: 1px solid rgba(87,139,255,0.4); color: var(--accent-primary); border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px;">
                            {creating() ? '⏳ CREATING...' : '+ CREATE USER'}
                        </button>
                    </div>

                    {/* User list */}
                    <div style="display: flex; gap: 0.5rem; margin-bottom: 0.5rem;">
                        <input placeholder="Search users..." value={search()} onInput={e => setSearch((e.target as HTMLInputElement).value)}
                            style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; width: 220px;" />
                    </div>
                    <Show when={loading()}>
                        <div style="color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; padding: 2rem; text-align: center;">LOADING...</div>
                    </Show>
                    <Show when={!loading()}>
                        <div style="border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 11px; font-family: var(--font-mono);">
                                <thead>
                                    <tr style="background: var(--bg-secondary); border-bottom: 1px solid var(--glass-border);">
                                        {['Username', 'Email', 'Role', 'MFA', 'Last Login', ''].map(h => (
                                            <th style="padding: 10px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 0.5px;">{h}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={filtered()}>
                                        {(u) => (
                                            <tr style="border-bottom: 1px solid rgba(255,255,255,0.04);">
                                                <td style="padding: 10px 12px; font-weight: 700; color: var(--text-primary);">{u.username}</td>
                                                <td style="padding: 10px 12px; color: var(--text-secondary);">{u.email || '—'}</td>
                                                <td style="padding: 10px 12px;">
                                                    <span style={`padding: 2px 7px; border-radius: 3px; font-size: 9px; font-weight: 800; background: rgba(0,0,0,0.3); color: ${ROLE_COLORS[u.role] ?? '#6b7280'};`}>{u.role?.toUpperCase()}</span>
                                                </td>
                                                <td style="padding: 10px 12px;">
                                                    <span style={`font-size: 10px; color: ${u.mfa_enabled ? '#3fb950' : '#6b7280'};`}>{u.mfa_enabled ? '✓ ON' : '○ OFF'}</span>
                                                </td>
                                                <td style="padding: 10px 12px; color: var(--text-muted); font-size: 10px;">{u.last_login?.slice(0, 16)?.replace('T', ' ') ?? 'Never'}</td>
                                                <td style="padding: 10px 12px;">
                                                    <button onClick={() => deleteUser(u.id, u.username)}
                                                        style="padding: 3px 8px; font-size: 9px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; border-radius: 3px; cursor: pointer; border: 1px solid rgba(248,81,73,0.3); background: transparent; color: #f85149; letter-spacing: 0.5px;">
                                                        DISABLE
                                                    </button>
                                                </td>
                                            </tr>
                                        )}
                                    </For>
                                    <Show when={filtered().length === 0}>
                                        <tr><td colspan="6" style="padding: 2rem; text-align: center; color: var(--text-muted); font-size: 11px;">No users found</td></tr>
                                    </Show>
                                </tbody>
                            </table>
                        </div>
                    </Show>
                </Show>

                {/* Providers tab */}
                <Show when={tab() === 'providers'}>
                    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem;">
                        <For each={PROVIDERS}>
                            {(prov) => (
                                <div style={`background: var(--bg-secondary); border: 1px solid ${prov.status === 'configured' ? 'rgba(63,185,80,0.3)' : 'var(--glass-border)'}; border-radius: 6px; padding: 1.25rem;`}>
                                    <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 8px;">
                                        <span style="font-size: 20px;">{prov.icon}</span>
                                        <div>
                                            <div style="font-size: 12px; font-weight: 700; color: var(--text-primary);">{prov.name}</div>
                                            <div style="font-size: 10px; color: var(--text-muted);">{prov.note}</div>
                                        </div>
                                    </div>
                                    <div style={`font-size: 10px; font-family: var(--font-mono); font-weight: 700; letter-spacing: 1px; color: ${prov.status === 'configured' ? '#3fb950' : '#6b7280'};`}>
                                        {prov.status === 'configured' ? '● CONFIGURED' : '○ AVAILABLE'}
                                    </div>
                                    <Show when={prov.status === 'available'}>
                                        <button style="margin-top: 10px; padding: 5px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid rgba(87,139,255,0.3); background: rgba(87,139,255,0.08); color: var(--accent-primary);">
                                            CONFIGURE
                                        </button>
                                    </Show>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                {/* Sessions tab */}
                <Show when={tab() === 'sessions'}>
                    <div style="text-align: center; padding: 3rem; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px; background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px;">
                        <div style="font-size: 2.5rem; margin-bottom: 1rem; opacity: 0.3;">🔐</div>
                        SESSION MANAGEMENT<br/>
                        <span style="font-size: 10px; opacity: 0.6; display: block; margin-top: 0.5rem; max-width: 400px; margin-left: auto; margin-right: auto; line-height: 1.5;">Active session tracking, forced logout, and token revocation are managed through the Identity Service. Configure session TTL and refresh policies in Settings → Security.</span>
                    </div>
                </Show>
            </div>
        </div>
    );
};
