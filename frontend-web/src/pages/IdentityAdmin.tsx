/**
 * IdentityAdmin.tsx — User & Role Administration (Web Only — Phase 0.5)
 *
 * Provides identity management: view/invite users, manage roles,
 * and configure OIDC/SAML identity provider settings.
 * Connects to /api/v1/users and /api/v1/roles.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface User {
  id: string;
  email: string;
  name: string;
  role_id: string;
  role_name: string;
  tenant_id: string;
  created_at: string;
  last_login?: string;
  mfa_enabled?: boolean;
}

interface Role {
  id: string;
  name: string;
  permissions: string[];
}

async function fetchUsers(): Promise<User[]> {
  try { return await request<User[]>('/users'); } catch { return []; }
}
async function fetchRoles(): Promise<Role[]> {
  try { return await request<Role[]>('/roles'); } catch { return []; }
}

type Tab = 'users' | 'roles' | 'providers';

export default function IdentityAdmin() {
  const [tab, setTab] = createSignal<Tab>('users');
  const [users] = createResource(fetchUsers);
  const [roles] = createResource(fetchRoles);
  const [userSearch, setUserSearch] = createSignal('');

  const filteredUsers = () =>
    (users() ?? []).filter(u =>
      u.email.toLowerCase().includes(userSearch().toLowerCase()) ||
      u.name.toLowerCase().includes(userSearch().toLowerCase())
    );

  const TAB_STYLE = (active: boolean) =>
    `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${active ? '#00ffe7' : 'transparent'}; background:none; color:${active ? '#00ffe7' : '#607070'}; transition:color 0.15s;`;

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ IDENTITY ADMINISTRATION</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">User management, roles & federated identity configuration</p>
      </div>

      {/* Tabs */}
      <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.5rem;">
        {(['users', 'roles', 'providers'] as Tab[]).map(t => (
          <button style={TAB_STYLE(tab() === t)} onClick={() => setTab(t)}>
            {t.toUpperCase()}
          </button>
        ))}
      </div>

      {/* Users Tab */}
      <Show when={tab() === 'users'}>
        <div style="display:flex; justify-content:space-between; margin-bottom:1rem;">
          <input
            type="text" placeholder="Search users…"
            value={userSearch()} onInput={e => setUserSearch(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.5rem 0.75rem; border-radius:4px; font-size:0.8rem; width:240px;"
          />
          <button style="background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:0.5rem 1rem; border-radius:4px; cursor:pointer; font-size:0.78rem; letter-spacing:0.1em;">
            + INVITE USER
          </button>
        </div>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
          <table style="width:100%; border-collapse:collapse; font-size:0.77rem;">
            <thead>
              <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                {['NAME', 'EMAIL', 'ROLE', 'TENANT', 'MFA', 'LAST LOGIN'].map(h => (
                  <th style="padding:0.65rem 1rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <Show when={!users.loading} fallback={<tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">Loading…</td></tr>}>
                <For each={filteredUsers()} fallback={<tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">No users found.</td></tr>}>
                  {(u) => (
                    <tr style="border-bottom:1px solid #0a1318;" onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'} onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}>
                      <td style="padding:0.65rem 1rem; color:#c8d8d8;">{u.name}</td>
                      <td style="padding:0.65rem 1rem; color:#607070;">{u.email}</td>
                      <td style="padding:0.65rem 1rem;">
                        <span style="background:#1e3040; color:#00ffe7; padding:0.15rem 0.5rem; border-radius:3px; font-size:0.7rem; letter-spacing:0.1em;">
                          {(u.role_name ?? u.role_id).toUpperCase()}
                        </span>
                      </td>
                      <td style="padding:0.65rem 1rem; color:#00ffe7;">{u.tenant_id || 'GLOBAL'}</td>
                      <td style="padding:0.65rem 1rem;">
                        <span style={`color:${u.mfa_enabled ? '#00ff88' : '#ff3355'}; font-size:0.7rem;`}>
                          {u.mfa_enabled ? '✓ ON' : '✗ OFF'}
                        </span>
                      </td>
                      <td style="padding:0.65rem 1rem; color:#607070;">
                        {u.last_login ? new Date(u.last_login).toLocaleDateString() : '—'}
                      </td>
                    </tr>
                  )}
                </For>
              </Show>
            </tbody>
          </table>
        </div>
      </Show>

      {/* Roles Tab */}
      <Show when={tab() === 'roles'}>
        <div style="display:grid; gap:1rem;">
          <Show when={!roles.loading} fallback={<div style="color:#607070;">Loading roles…</div>}>
            <For each={roles()} fallback={<div style="color:#607070;">No roles defined.</div>}>
              {(role) => (
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem;">
                  <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:0.75rem;">
                    <span style="color:#00ffe7; font-size:0.9rem; letter-spacing:0.12em;">{role.name.toUpperCase()}</span>
                    <button style="background:none; border:1px solid #1e3040; color:#607070; padding:0.25rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.72rem;">EDIT</button>
                  </div>
                  <div style="display:flex; flex-wrap:wrap; gap:0.4rem;">
                    <For each={role.permissions}>
                      {(p) => (
                        <span style="background:#111f28; border:1px solid #1e3040; color:#607070; padding:0.15rem 0.5rem; border-radius:3px; font-size:0.7rem;">{p}</span>
                      )}
                    </For>
                  </div>
                </div>
              )}
            </For>
          </Show>
        </div>
      </Show>

      {/* Identity Providers Tab */}
      <Show when={tab() === 'providers'}>
        <div style="display:grid; gap:1rem;">
          {[
            { name: 'OIDC / OAuth2', desc: 'Google Workspace, Okta, Azure AD', endpoint: '/api/v1/auth/oidc/login', status: 'configured' },
            { name: 'SAML 2.0',      desc: 'Okta SAML, ADFS, PingIdentity',   endpoint: '/api/v1/auth/saml/login',  status: 'configured' },
          ].map(p => (
            <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem; display:flex; justify-content:space-between; align-items:center;">
              <div>
                <div style="color:#00ffe7; font-size:0.9rem; letter-spacing:0.12em; margin-bottom:0.25rem;">{p.name}</div>
                <div style="color:#607070; font-size:0.76rem;">{p.desc}</div>
                <div style="color:#1e3040; font-size:0.7rem; margin-top:0.25rem;">{p.endpoint}</div>
              </div>
              <div style="display:flex; align-items:center; gap:1rem;">
                <span style="color:#00ff88; font-size:0.75rem; letter-spacing:0.1em;">● {p.status.toUpperCase()}</span>
                <button
                  onClick={() => window.location.href = p.endpoint}
                  style="background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:0.4rem 0.9rem; border-radius:4px; cursor:pointer; font-size:0.75rem; letter-spacing:0.1em;"
                >
                  TEST LOGIN
                </button>
              </div>
            </div>
          ))}
        </div>
      </Show>
    </div>
  );
}
