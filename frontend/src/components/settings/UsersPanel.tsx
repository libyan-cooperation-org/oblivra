import { Component, createSignal, For, Show, onMount } from 'solid-js';
import '../../styles/identity.css';

// Stub types matching the Go backend models
interface User {
    id: string;
    email: string;
    name: string;
    auth_provider: string;
    is_mfa_enabled: boolean;
    role_id: string;
    created_at: string;
    last_login_at: string;
}

interface Role {
    id: string;
    name: string;
    description: string;
    permissions: string[];
    is_system: boolean;
}

type Tab = 'users' | 'roles' | 'sso';

export const UsersPanel: Component = () => {
    const [activeTab, setActiveTab] = createSignal<Tab>('users');
    const [users, setUsers] = createSignal<User[]>([]);
    const [roles, setRoles] = createSignal<Role[]>([]);
    const [showCreateUser, setShowCreateUser] = createSignal(false);
    const [showCreateRole, setShowCreateRole] = createSignal(false);

    // Create user form
    const [newEmail, setNewEmail] = createSignal('');
    const [newName, setNewName] = createSignal('');
    const [newPassword, setNewPassword] = createSignal('');
    const [newRoleId, setNewRoleId] = createSignal('role_analyst');

    // Create role form
    const [roleName, setRoleName] = createSignal('');
    const [roleDesc, setRoleDesc] = createSignal('');

    onMount(async () => {
        try {
            // @ts-ignore - Wails runtime bindings
            if ((window as any).go?.app?.IdentityService) {
                const u = await (window as any).go.app.IdentityService.ListUsers();
                if (u) setUsers(u);
                const r = await (window as any).go.app.IdentityService.ListRoles();
                if (r) setRoles(r);
            } else {
                // Demo data for development
                setRoles([
                    { id: 'role_admin', name: 'Administrator', description: 'Full access to all features', permissions: ['*'], is_system: true },
                    { id: 'role_analyst', name: 'Analyst', description: 'Investigation and read access', permissions: ['hosts:read', 'sessions:read', 'siem:read', 'siem:write', 'incidents:read', 'incidents:write'], is_system: true },
                    { id: 'role_readonly', name: 'Read-Only', description: 'View-only access to dashboards', permissions: ['hosts:read', 'sessions:read', 'siem:read', 'incidents:read'], is_system: true },
                ]);
                setUsers([
                    { id: '1', email: 'admin@sovereign.local', name: 'System Admin', auth_provider: 'local', is_mfa_enabled: true, role_id: 'role_admin', created_at: '2026-01-01T00:00:00Z', last_login_at: '2026-03-03T22:00:00Z' },
                ]);
            }
        } catch (e) {
            console.error('[IAM] Failed to load identity data:', e);
        }
    });

    const handleCreateUser = async () => {
        try {
            // @ts-ignore
            if ((window as any).go?.app?.IdentityService) {
                await (window as any).go.app.IdentityService.CreateUser(newEmail(), newName(), newPassword(), newRoleId());
                const u = await (window as any).go.app.IdentityService.ListUsers();
                if (u) setUsers(u);
            }
        } catch (e) {
            console.error('[IAM] Failed to create user:', e);
        }
        setShowCreateUser(false);
        setNewEmail(''); setNewName(''); setNewPassword('');
    };

    const handleCreateRole = async () => {
        try {
            // @ts-ignore
            if ((window as any).go?.app?.IdentityService) {
                await (window as any).go.app.IdentityService.CreateRole(roleName(), roleDesc(), []);
                const r = await (window as any).go.app.IdentityService.ListRoles();
                if (r) setRoles(r);
            }
        } catch (e) {
            console.error('[IAM] Failed to create role:', e);
        }
        setShowCreateRole(false);
        setRoleName(''); setRoleDesc('');
    };

    const getRoleName = (roleId: string): string => {
        const role = roles().find(r => r.id === roleId);
        return role ? role.name : roleId;
    };

    const getRoleBadgeClass = (roleId: string): string => {
        if (roleId === 'role_admin') return 'admin';
        if (roleId === 'role_analyst') return 'analyst';
        if (roleId === 'role_readonly') return 'readonly';
        return 'custom';
    };

    return (
        <div class="identity-page">
            {/* Sidebar */}
            <div class="identity-sidebar">
                <div class="sidebar-title">Identity & Access</div>
                <button class={activeTab() === 'users' ? 'active' : ''} onClick={() => setActiveTab('users')}>
                    ⊡ Users
                </button>
                <button class={activeTab() === 'roles' ? 'active' : ''} onClick={() => setActiveTab('roles')}>
                    ◈ Roles & Permissions
                </button>
                <button class={activeTab() === 'sso' ? 'active' : ''} onClick={() => setActiveTab('sso')}>
                    ⊞ SSO Providers
                </button>
            </div>

            {/* Content */}
            <div class="identity-content">
                <Show when={activeTab() === 'users'}>
                    <h2>User Management</h2>

                    {/* Stats */}
                    <div class="identity-stats">
                        <div class="stat-card">
                            <div class="stat-value">{users().length}</div>
                            <div class="stat-label">Total Users</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">{users().filter(u => u.is_mfa_enabled).length}</div>
                            <div class="stat-label">MFA Enabled</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">{users().filter(u => u.auth_provider !== 'local').length}</div>
                            <div class="stat-label">SSO Users</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-value">{roles().length}</div>
                            <div class="stat-label">Active Roles</div>
                        </div>
                    </div>

                    <div class="identity-actions">
                        <button class="btn-primary" onClick={() => setShowCreateUser(true)}>+ Create User</button>
                    </div>

                    <Show when={users().length > 0} fallback={
                        <div class="empty-state">
                            <div class="empty-icon">⊡</div>
                            <p>No users provisioned yet.</p>
                            <p>Create the first user to enable identity-based access control.</p>
                        </div>
                    }>
                        <table class="users-table">
                            <thead>
                                <tr>
                                    <th>Name</th>
                                    <th>Email</th>
                                    <th>Role</th>
                                    <th>Auth</th>
                                    <th>MFA</th>
                                    <th>Last Login</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <For each={users()}>
                                    {(user) => (
                                        <tr>
                                            <td>{user.name}</td>
                                            <td><span class="user-email">{user.email}</span></td>
                                            <td><span class={`role-badge ${getRoleBadgeClass(user.role_id)}`}>{getRoleName(user.role_id)}</span></td>
                                            <td><span class="auth-badge">{user.auth_provider.toUpperCase()}</span></td>
                                            <td>
                                                <span class={`mfa-indicator ${user.is_mfa_enabled ? 'enabled' : 'disabled'}`}>
                                                    {user.is_mfa_enabled ? '● TOTP' : '○ Off'}
                                                </span>
                                            </td>
                                            <td style={{ 'font-size': '12px', color: 'var(--text-muted)' }}>
                                                {user.last_login_at ? new Date(user.last_login_at).toLocaleDateString() : 'Never'}
                                            </td>
                                            <td>
                                                <button class="btn-secondary" style={{ padding: '4px 10px', 'font-size': '11px' }}>Edit</button>
                                            </td>
                                        </tr>
                                    )}
                                </For>
                            </tbody>
                        </table>
                    </Show>
                </Show>

                <Show when={activeTab() === 'roles'}>
                    <h2>Roles & Permissions</h2>
                    <div class="identity-actions">
                        <button class="btn-primary" onClick={() => setShowCreateRole(true)}>+ Create Role</button>
                    </div>

                    <div class="roles-grid">
                        <For each={roles()}>
                            {(role) => (
                                <div class="role-card">
                                    <div class="role-card-header">
                                        <h4>{role.name}</h4>
                                        <Show when={role.is_system}>
                                            <span class="system-tag">SYSTEM</span>
                                        </Show>
                                    </div>
                                    <p>{role.description}</p>
                                    <div class="permissions-list">
                                        <For each={role.permissions}>
                                            {(perm) => <span class="permission-tag">{perm}</span>}
                                        </For>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                <Show when={activeTab() === 'sso'}>
                    <h2>SSO Providers</h2>
                    <div class="identity-actions">
                        <button class="btn-primary">+ Add OIDC Provider</button>
                        <button class="btn-secondary">+ Add SAML IdP</button>
                    </div>

                    <div class="empty-state">
                        <div class="empty-icon">⊞</div>
                        <p>No SSO providers configured.</p>
                        <p>Connect Okta, Entra ID, Keycloak, or any SAML/OIDC identity provider.</p>
                    </div>
                </Show>

                {/* Create User Modal */}
                <Show when={showCreateUser()}>
                    <div class="modal-overlay" onClick={() => setShowCreateUser(false)}>
                        <div class="modal-container" onClick={(e) => e.stopPropagation()}>
                            <h3>Create User</h3>
                            <div class="form-field">
                                <label>Full Name</label>
                                <input type="text" value={newName()} onInput={(e) => setNewName(e.currentTarget.value)} placeholder="Jane Analyst" />
                            </div>
                            <div class="form-field">
                                <label>Email</label>
                                <input type="email" value={newEmail()} onInput={(e) => setNewEmail(e.currentTarget.value)} placeholder="jane@sovereign.local" />
                            </div>
                            <div class="form-field">
                                <label>Password</label>
                                <input type="password" value={newPassword()} onInput={(e) => setNewPassword(e.currentTarget.value)} placeholder="••••••••" />
                            </div>
                            <div class="form-field">
                                <label>Role</label>
                                <select value={newRoleId()} onChange={(e) => setNewRoleId(e.currentTarget.value)}>
                                    <For each={roles()}>
                                        {(role) => <option value={role.id}>{role.name}</option>}
                                    </For>
                                </select>
                            </div>
                            <div class="modal-actions">
                                <button class="btn-secondary" onClick={() => setShowCreateUser(false)}>Cancel</button>
                                <button class="btn-primary" onClick={handleCreateUser}>Create User</button>
                            </div>
                        </div>
                    </div>
                </Show>

                {/* Create Role Modal */}
                <Show when={showCreateRole()}>
                    <div class="modal-overlay" onClick={() => setShowCreateRole(false)}>
                        <div class="modal-container" onClick={(e) => e.stopPropagation()}>
                            <h3>Create Role</h3>
                            <div class="form-field">
                                <label>Role Name</label>
                                <input type="text" value={roleName()} onInput={(e) => setRoleName(e.currentTarget.value)} placeholder="SOC Observer" />
                            </div>
                            <div class="form-field">
                                <label>Description</label>
                                <input type="text" value={roleDesc()} onInput={(e) => setRoleDesc(e.currentTarget.value)} placeholder="Limited read access for SOC tier-1" />
                            </div>
                            <div class="modal-actions">
                                <button class="btn-secondary" onClick={() => setShowCreateRole(false)}>Cancel</button>
                                <button class="btn-primary" onClick={handleCreateRole}>Create Role</button>
                            </div>
                        </div>
                    </div>
                </Show>
            </div>
        </div>
    );
};
