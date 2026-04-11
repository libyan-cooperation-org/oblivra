<!-- OBLIVRA Web — IdentityAdmin (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { request } from '../services/api';

  interface User  { id:string; email:string; name:string; role_id:string; role_name:string; tenant_id:string; created_at:string; last_login?:string; mfa_enabled?:boolean; }
  interface Role  { id:string; name:string; permissions:string[]; }

  type Tab = 'users'|'roles'|'providers';
  let tab       = $state<Tab>('users');
  let users     = $state<User[]>([]);
  let roles     = $state<Role[]>([]);
  let loading   = $state(true);
  let userSearch = $state('');

  onMount(async () => {
    try { users = await request<User[]>('/users'); } catch { users = []; }
    try { roles = await request<Role[]>('/roles'); } catch { roles = []; }
    loading = false;
  });

  const filteredUsers = $derived(users.filter(u =>
    u.email.toLowerCase().includes(userSearch.toLowerCase()) ||
    u.name.toLowerCase().includes(userSearch.toLowerCase())
  ));

  const providers = [
    { name:'OIDC / OAuth2', desc:'Google Workspace, Okta, Azure AD', endpoint:'/api/v1/auth/oidc/login', status:'configured' },
    { name:'SAML 2.0',      desc:'Okta SAML, ADFS, PingIdentity',   endpoint:'/api/v1/auth/saml/login',  status:'configured' },
  ];
</script>

<div class="ia-page">
  <div class="ia-header">
    <h1 class="ia-title">⬡ IDENTITY ADMINISTRATION</h1>
    <p class="ia-sub">User management, roles &amp; federated identity configuration</p>
  </div>

  <div class="ia-tabs">
    {#each (['users','roles','providers'] as Tab[]) as t}
      <button class="ia-tab {tab===t ? 'ia-tab--active' : ''}" onclick={() => tab = t}>{t.toUpperCase()}</button>
    {/each}
  </div>

  {#if tab === 'users'}
    <div class="ia-toolbar">
      <input type="text" placeholder="Search users…" bind:value={userSearch} class="ia-search" />
      <button class="ia-invite-btn">+ INVITE USER</button>
    </div>
    <div class="ia-table-wrap">
      <table class="ia-table">
        <thead><tr>{#each ['NAME','EMAIL','ROLE','TENANT','MFA','LAST LOGIN'] as h}<th>{h}</th>{/each}</tr></thead>
        <tbody>
          {#if loading}
            <tr><td colspan="6" class="ia-center">Loading…</td></tr>
          {:else if filteredUsers.length === 0}
            <tr><td colspan="6" class="ia-center">No users found.</td></tr>
          {:else}
            {#each filteredUsers as u (u.id)}
              <tr class="ia-row">
                <td>{u.name}</td>
                <td class="ia-muted">{u.email}</td>
                <td><span class="ia-role-badge">{(u.role_name??u.role_id).toUpperCase()}</span></td>
                <td class="ia-teal">{u.tenant_id||'GLOBAL'}</td>
                <td><span style="color:{u.mfa_enabled ? '#00ff88' : '#ff3355'}">{u.mfa_enabled ? '✓ ON' : '✗ OFF'}</span></td>
                <td class="ia-muted">{u.last_login ? new Date(u.last_login).toLocaleDateString() : '—'}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

  {:else if tab === 'roles'}
    <div class="ia-grid">
      {#each roles as role (role.id)}
        <div class="ia-role-card">
          <div class="ia-role-header">
            <span class="ia-role-name">{role.name.toUpperCase()}</span>
            <button class="ia-edit-btn">EDIT</button>
          </div>
          <div class="ia-perms">
            {#each role.permissions as p}<span class="ia-perm">{p}</span>{/each}
          </div>
        </div>
      {:else}
        <div class="ia-muted">No roles defined.</div>
      {/each}
    </div>

  {:else}
    <div class="ia-grid">
      {#each providers as p}
        <div class="ia-provider-card">
          <div>
            <div class="ia-provider-name">{p.name}</div>
            <div class="ia-muted">{p.desc}</div>
          </div>
          <div class="ia-provider-right">
            <span class="ia-provider-status">● {p.status.toUpperCase()}</span>
            <button class="ia-test-btn" onclick={() => window.location.href = p.endpoint}>TEST LOGIN</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .ia-page { padding:28px; color:#c8d8d8; font-family:var(--font-mono); min-height:100vh; background:#080f12; }
  .ia-header { margin-bottom:20px; }
  .ia-title  { font-size:20px; letter-spacing:.14em; margin:0; color:#00ffe7; }
  .ia-sub    { margin:3px 0 0; font-size:11px; color:#607070; }
  .ia-tabs   { display:flex; border-bottom:1px solid #1e3040; margin-bottom:20px; }
  .ia-tab    { padding:8px 18px; cursor:pointer; font-size:11px; letter-spacing:.12em; border:none; border-bottom:2px solid transparent; background:none; color:#607070; font-family:inherit; transition:color 100ms; }
  .ia-tab--active { border-bottom-color:#00ffe7; color:#00ffe7; }
  .ia-toolbar { display:flex; justify-content:space-between; margin-bottom:14px; }
  .ia-search  { background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:7px 12px; border-radius:4px; font-size:12px; width:240px; font-family:inherit; outline:none; }
  .ia-invite-btn { background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:7px 14px; border-radius:4px; cursor:pointer; font-size:11px; letter-spacing:.1em; font-family:inherit; }
  .ia-table-wrap { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; }
  .ia-table { width:100%; border-collapse:collapse; font-size:11px; }
  .ia-table thead tr { border-bottom:1px solid #1e3040; background:#0a1318; }
  .ia-table th { padding:9px 14px; text-align:left; color:#607070; letter-spacing:.1em; font-weight:400; font-size:10px; }
  .ia-row { border-bottom:1px solid #0a1318; transition:background 80ms; }
  .ia-row:hover { background:#111f28; }
  .ia-row td { padding:9px 14px; }
  .ia-muted { color:#607070; }
  .ia-teal  { color:#00ffe7; }
  .ia-role-badge { background:#1e3040; color:#00ffe7; padding:2px 7px; border-radius:3px; font-size:10px; letter-spacing:.1em; }
  .ia-center { padding:28px; text-align:center; color:#607070; }
  .ia-grid   { display:flex; flex-direction:column; gap:14px; }
  .ia-role-card { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:16px; }
  .ia-role-header { display:flex; justify-content:space-between; align-items:center; margin-bottom:10px; }
  .ia-role-name   { color:#00ffe7; font-size:13px; letter-spacing:.12em; }
  .ia-edit-btn { background:none; border:1px solid #1e3040; color:#607070; padding:3px 10px; border-radius:3px; cursor:pointer; font-size:11px; font-family:inherit; }
  .ia-perms { display:flex; flex-wrap:wrap; gap:5px; }
  .ia-perm  { background:#111f28; border:1px solid #1e3040; color:#607070; padding:2px 7px; border-radius:3px; font-size:10px; }
  .ia-provider-card  { background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:18px; display:flex; justify-content:space-between; align-items:center; }
  .ia-provider-name  { color:#00ffe7; font-size:13px; letter-spacing:.12em; margin-bottom:3px; }
  .ia-provider-right { display:flex; align-items:center; gap:14px; }
  .ia-provider-status { color:#00ff88; font-size:11px; letter-spacing:.1em; }
  .ia-test-btn { background:#1e3040; border:1px solid #00ffe7; color:#00ffe7; padding:6px 14px; border-radius:4px; cursor:pointer; font-size:11px; font-family:inherit; }
</style>
