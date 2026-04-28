<!--
  Identity Connectors — admin page for federated identity providers.
  Operators configure OIDC + SAML connectors here; the Login page's
  SSO buttons + the IdentityService backend flows pick them up.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import {
    PageLayout, KPI, Badge, Button, DataTable, Input, Tabs, EmptyState, PopOutButton,
  } from '@components/ui';
  import { KeyRound, Plus, Trash2, RefreshCw, Power, PowerOff, Edit3 } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Connector = {
    id: string;
    tenant_id?: string;
    name: string;
    type: 'oidc' | 'saml' | string;
    enabled: boolean;
    config_json: string;
    sync_interval_mins?: number;
    last_sync?: string;
    status?: string;
    error_message?: string;
    created_at?: string;
    updated_at?: string;
  };

  let connectors = $state<Connector[]>([]);
  let loading = $state(false);
  let activeTab = $state<'oidc' | 'saml'>('oidc');

  // Form state — shared by both kinds; only the inner shape differs.
  let editing = $state<Connector | null>(null);
  let showAdd = $state(false);
  let formName = $state('');
  let formType = $state<'oidc' | 'saml'>('oidc');
  let formEnabled = $state(true);

  // OIDC fields
  let oIssuer = $state('');
  let oClientID = $state('');
  let oClientSecret = $state('');
  let oRedirect = $state('');

  // SAML fields
  let sIdpMeta = $state('');
  let sSPEntity = $state('');
  let sSPACS = $state('');
  let sSPCertPEM = $state('');
  let sSPKeyPEM = $state('');

  function resetForm() {
    formName = '';
    formEnabled = true;
    oIssuer = ''; oClientID = ''; oClientSecret = ''; oRedirect = '';
    sIdpMeta = ''; sSPEntity = ''; sSPACS = ''; sSPCertPEM = ''; sSPKeyPEM = '';
    editing = null;
  }

  function openNew(kind: 'oidc' | 'saml') {
    resetForm();
    formType = kind;
    activeTab = kind;
    showAdd = true;
  }

  function openEdit(c: Connector) {
    resetForm();
    editing = c;
    formName = c.name;
    formType = (c.type === 'saml' ? 'saml' : 'oidc');
    formEnabled = c.enabled;
    try {
      const cfg = JSON.parse(c.config_json || '{}');
      if (formType === 'oidc') {
        oIssuer = cfg.issuer ?? '';
        oClientID = cfg.client_id ?? '';
        oClientSecret = cfg.client_secret ?? '';
        oRedirect = cfg.redirect_url ?? '';
      } else {
        sIdpMeta = cfg.idp_metadata_url ?? '';
        sSPEntity = cfg.sp_entity_id ?? '';
        sSPACS = cfg.sp_acs_url ?? '';
        sSPCertPEM = cfg.sp_cert_pem ?? '';
        sSPKeyPEM = cfg.sp_key_pem ?? '';
      }
    } catch {
      // bad JSON in stored config — let the operator re-enter from blank.
    }
    showAdd = true;
  }

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) {
        const { apiFetch } = await import('@lib/apiClient');
        const res = await apiFetch('/api/v1/identity/connectors');
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data = await res.json();
        connectors = (data.connectors ?? []) as Connector[];
        return;
      }
      const { ListConnectors } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      connectors = ((await ListConnectors()) ?? []) as Connector[];
    } catch (e: any) {
      appStore.notify('Connector list failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  function buildConfigJSON(): string {
    if (formType === 'oidc') {
      if (!oIssuer || !oClientID || !oClientSecret || !oRedirect) {
        throw new Error('OIDC: issuer, client_id, client_secret, redirect_url are all required');
      }
      return JSON.stringify({
        issuer: oIssuer,
        client_id: oClientID,
        client_secret: oClientSecret,
        redirect_url: oRedirect,
      });
    }
    if (!sIdpMeta || !sSPEntity || !sSPACS || !sSPCertPEM || !sSPKeyPEM) {
      throw new Error('SAML: idp_metadata_url, sp_entity_id, sp_acs_url, sp_cert_pem, sp_key_pem are all required');
    }
    return JSON.stringify({
      idp_metadata_url: sIdpMeta,
      sp_entity_id: sSPEntity,
      sp_acs_url: sSPACS,
      sp_cert_pem: sSPCertPEM,
      sp_key_pem: sSPKeyPEM,
    });
  }

  async function save() {
    if (!formName.trim()) {
      appStore.notify('Connector name required', 'warning');
      return;
    }
    let configJSON: string;
    try {
      configJSON = buildConfigJSON();
    } catch (e: any) {
      appStore.notify('Validation failed', 'error', e?.message ?? String(e));
      return;
    }

    const payload = {
      id: editing?.id ?? '',
      tenant_id: editing?.tenant_id ?? '',
      name: formName,
      type: formType,
      enabled: formEnabled,
      config_json: configJSON,
      sync_interval_mins: editing?.sync_interval_mins ?? 0,
    };
    try {
      if (IS_BROWSER) {
        const { apiPostJSON, apiFetch } = await import('@lib/apiClient');
        let res: Response;
        if (editing) {
          res = await apiFetch(`/api/v1/identity/connectors/${encodeURIComponent(editing.id)}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          });
        } else {
          res = await apiPostJSON('/api/v1/identity/connectors', payload);
        }
        if (!res.ok) {
          const txt = await res.text().catch(() => '');
          throw new Error(`HTTP ${res.status}: ${txt}`);
        }
      } else {
        const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice');
        if (editing) await svc.UpdateConnector(payload as any);
        else await svc.CreateConnector(payload as any);
      }
      appStore.notify(`${formType.toUpperCase()} connector ${editing ? 'updated' : 'created'}`, 'success');
      showAdd = false;
      resetForm();
      void refresh();
    } catch (e: any) {
      appStore.notify('Save failed', 'error', e?.message ?? String(e));
    }
  }

  async function toggleEnabled(c: Connector) {
    try {
      const { UpdateConnector } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      await UpdateConnector({ ...c, enabled: !c.enabled } as any);
      appStore.notify(`${c.name} ${c.enabled ? 'disabled' : 'enabled'}`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify('Toggle failed', 'error', e?.message ?? String(e));
    }
  }

  async function deleteConnector(c: Connector) {
    if (!confirm(`Delete connector "${c.name}"? Operators relying on this for SSO will lose access.`)) return;
    try {
      const { DeleteConnector } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/identityservice'
      );
      await DeleteConnector(c.id);
      appStore.notify('Connector deleted', 'warning');
      void refresh();
    } catch (e: any) {
      appStore.notify('Delete failed', 'error', e?.message ?? String(e));
    }
  }

  let oidcs = $derived(connectors.filter((c) => c.type === 'oidc'));
  let samls = $derived(connectors.filter((c) => c.type === 'saml'));
  let visible = $derived(activeTab === 'oidc' ? oidcs : samls);

  onMount(refresh);
</script>

<PageLayout title="Identity Connectors" subtitle="OIDC + SAML federated identity providers">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
    <Button variant="primary" size="sm" icon={Plus} onclick={() => openNew(activeTab)}>
      Add {activeTab.toUpperCase()} connector
    </Button>
    <PopOutButton route="/identity-connectors" title="Identity Connectors" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Total" value={connectors.length.toString()} variant="accent" />
      <KPI label="OIDC" value={oidcs.length.toString()} variant={oidcs.some((c) => c.enabled) ? 'success' : 'muted'} />
      <KPI label="SAML" value={samls.length.toString()} variant={samls.some((c) => c.enabled) ? 'success' : 'muted'} />
    </div>

    <Tabs tabs={[{ id: 'oidc', label: 'OIDC' }, { id: 'saml', label: 'SAML' }]} bind:active={activeTab} />

    {#if showAdd}
      <div class="border border-border-primary rounded-md bg-surface-1 p-4 space-y-3">
        <h3 class="text-xs uppercase tracking-widest font-bold flex items-center gap-2">
          <KeyRound size={14} />
          {editing ? `Edit ${formType.toUpperCase()} connector` : `New ${formType.toUpperCase()} connector`}
        </h3>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Name</div>
            <Input bind:value={formName} placeholder="e.g. Okta corp" />
          </div>
          <div class="flex items-end">
            <label class="flex items-center gap-2 text-xs">
              <input type="checkbox" bind:checked={formEnabled} class="accent-cyan-400" />
              Enabled
            </label>
          </div>
        </div>

        {#if formType === 'oidc'}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Issuer URL</div>
              <Input bind:value={oIssuer} placeholder="https://accounts.example.com" />
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Client ID</div>
              <Input bind:value={oClientID} placeholder="oblivra-prod" />
            </div>
            <div class="md:col-span-2">
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Client Secret</div>
              <Input type="password" bind:value={oClientSecret} placeholder="••••••••" />
            </div>
            <div class="md:col-span-2">
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">Redirect URL</div>
              <Input bind:value={oRedirect} placeholder="https://soc.example.com/api/v1/auth/oidc/callback" />
            </div>
          </div>
        {:else}
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div class="md:col-span-2">
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">IdP Metadata URL</div>
              <Input bind:value={sIdpMeta} placeholder="https://idp.example.com/metadata.xml" />
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">SP Entity ID</div>
              <Input bind:value={sSPEntity} placeholder="https://soc.example.com" />
            </div>
            <div>
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">SP ACS URL</div>
              <Input bind:value={sSPACS} placeholder="https://soc.example.com/api/v1/auth/saml/callback" />
            </div>
            <div class="md:col-span-2">
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">SP Certificate (PEM)</div>
              <textarea
                class="w-full h-24 bg-surface-2 border border-border-primary rounded px-2 py-1.5 font-mono text-[11px]"
                bind:value={sSPCertPEM}
                placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
              ></textarea>
            </div>
            <div class="md:col-span-2">
              <div class="text-[10px] uppercase tracking-widest text-text-muted mb-1">SP Private Key (PEM)</div>
              <textarea
                class="w-full h-24 bg-surface-2 border border-border-primary rounded px-2 py-1.5 font-mono text-[11px]"
                bind:value={sSPKeyPEM}
                placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"
              ></textarea>
            </div>
          </div>
        {/if}

        <div class="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onclick={() => { showAdd = false; resetForm(); }}>Cancel</Button>
          <Button variant="primary" onclick={save}>{editing ? 'Save changes' : 'Create connector'}</Button>
        </div>
      </div>
    {/if}

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      {#if visible.length === 0}
        <EmptyState
          title={loading ? 'Loading…' : `No ${activeTab.toUpperCase()} connectors`}
          description={`Add an ${activeTab.toUpperCase()} connector to enable federated sign-in. Operators see the SSO buttons on the Login page only after at least one connector is enabled.`}
          icon="🔐"
        >
          {#snippet action()}
            <Button variant="primary" onclick={() => openNew(activeTab)}>Add {activeTab.toUpperCase()} connector</Button>
          {/snippet}
        </EmptyState>
      {:else}
        <DataTable
          data={visible}
          columns={[
            { key: 'name',         label: 'Name',     sortable: true },
            { key: 'enabled',      label: 'Status',   width: '100px' },
            { key: 'status',       label: 'Last Sync',width: '160px' },
            { key: 'updated_at',   label: 'Updated',  width: '160px' },
            { key: 'actions',      label: '',         width: '180px' },
          ]}
          compact
        >
          {#snippet render({ col, row })}
            {#if col.key === 'enabled'}
              <Badge variant={row.enabled ? 'success' : 'muted'} size="xs">
                {row.enabled ? 'enabled' : 'disabled'}
              </Badge>
            {:else if col.key === 'status'}
              <span class="text-[10px] font-mono text-text-muted">{row.last_sync ?? row.status ?? '—'}</span>
            {:else if col.key === 'updated_at'}
              <span class="font-mono text-[10px] text-text-muted">{(row.updated_at ?? '').slice(0, 19)}</span>
            {:else if col.key === 'actions'}
              <div class="flex gap-1 justify-end">
                <Button variant="ghost" size="xs" onclick={() => openEdit(row)}>
                  <Edit3 size={11} />
                </Button>
                <Button variant="ghost" size="xs" onclick={() => toggleEnabled(row)}>
                  {#if row.enabled}<PowerOff size={11} />{:else}<Power size={11} />{/if}
                </Button>
                <Button variant="ghost" size="xs" onclick={() => deleteConnector(row)}>
                  <Trash2 size={11} class="text-rose-400" />
                </Button>
              </div>
            {:else}
              <span class="text-[11px]">{row[col.key] ?? '—'}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>
  </div>
</PageLayout>
