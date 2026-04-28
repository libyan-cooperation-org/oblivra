<!--
  Identity Admin — list users + roles + SSO connectors. Bound to IdentityService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Users, UserPlus, RefreshCw, Trash2 } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { apiFetch } from '@lib/apiClient';

  type User = { id?: string; email?: string; name?: string; role_id?: string; role?: string; mfa_enabled?: boolean; created_at?: string };
  type Role = { id?: string; name?: string; description?: string; permissions?: string[] };

  let users = $state<User[]>([]);
  let roles = $state<Role[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) {
        // Audit fix — browser mode now consumes the real
        // /api/v1/users + /api/v1/roles endpoints (wired in
        // commit 641907f). Previously this branch silently
        // returned empty arrays and the page showed "No users yet"
        // even when 50 users existed in the DB.
        const [uRes, rRes] = await Promise.all([
          apiFetch('/api/v1/users'),
          apiFetch('/api/v1/roles'),
        ]);
        if (uRes.ok) {
          const body = await uRes.json();
          users = (body.users ?? body.identities ?? []) as User[];
        }
        if (rRes.ok) {
          const body = await rRes.json();
          roles = (body.roles ?? []) as Role[];
        }
        return;
      }
      const svc = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      // ListUsers and a roles RPC. Some bindings may differ; defensive.
      const u = await svc.ListUsers();
      users = (u ?? []) as User[];
      const r = await (svc as any).ListRoles?.();
      if (Array.isArray(r)) roles = r as Role[];
    } catch (e: any) {
      appStore.notify(`Identity load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function createUser() {
    const email = prompt('Email:'); if (!email) return;
    const name  = prompt('Display name:', email) ?? email;
    const password = prompt('Initial password:'); if (!password) return;
    const roleID = roles[0]?.id ?? '';
    try {
      const { CreateUser } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      await CreateUser(email, name, password, roleID);
      appStore.notify(`User ${email} created`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Create failed: ${e?.message ?? e}`, 'error');
    }
  }

  async function deleteUser(id: string, email: string) {
    if (!confirm(`Delete user ${email}?`)) return;
    try {
      const { DeleteUser } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      await DeleteUser(id);
      users = users.filter((u) => u.id !== id);
    } catch (e: any) {
      appStore.notify(`Delete failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let stats = $derived({
    users: users.length,
    mfa: users.filter((u) => u.mfa_enabled).length,
    roles: roles.length,
  });
</script>

<PageLayout title="Identity Administration" subtitle="Users, roles, MFA, SSO connectors">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" icon={UserPlus} onclick={createUser}>New user</Button>
    <PopOutButton route="/identity-admin" title="Identity Admin" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Users" value={stats.users.toString()} variant="accent" />
      <KPI label="MFA Enabled" value={`${stats.mfa}/${stats.users || 1}`} variant={stats.users > 0 && stats.mfa === stats.users ? 'success' : 'warning'} />
      <KPI label="Roles" value={stats.roles.toString()} variant="muted" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Users size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Users</span>
      </div>
      {#if users.length === 0}
        <div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No users yet.'}</div>
      {:else}
        <DataTable data={users} columns={[
          { key: 'email', label: 'Email' },
          { key: 'name',  label: 'Name' },
          { key: 'role',  label: 'Role',  width: '120px' },
          { key: 'mfa',   label: 'MFA',   width: '70px' },
          { key: 'created_at', label: 'Created', width: '140px' },
          { key: 'del',   label: '',      width: '50px' },
        ]} compact>
          {#snippet render({ col, row })}
            {#if col.key === 'mfa'}
              <Badge variant={row.mfa_enabled ? 'success' : 'muted'} size="xs">{row.mfa_enabled ? 'On' : 'Off'}</Badge>
            {:else if col.key === 'role'}
              <span class="font-mono text-[10px]">{row.role ?? row.role_id ?? '—'}</span>
            {:else if col.key === 'created_at'}
              <span class="font-mono text-[10px] text-text-muted">{row.created_at?.slice(0, 10) ?? '—'}</span>
            {:else if col.key === 'del'}
              <button class="p-1 text-error hover:bg-error/10 rounded" onclick={() => deleteUser(row.id, row.email)}><Trash2 size={11} /></button>
            {:else}
              <span class="text-[11px]">{row[col.key] ?? '—'}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>
  </div>
</PageLayout>
