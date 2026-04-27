<!--
  OBLIVRA — Settings (Svelte 5)
  Application configuration and system management.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { Badge, PageLayout, Button, Tabs, Input } from '@components/ui';

  const settingsTabs = [
    { id: 'general', label: 'General', icon: '⚙️' },
    { id: 'server', label: 'Server Connection', icon: '🌐' },
    { id: 'security', label: 'Security', icon: '🛡️' },
    { id: 'advanced', label: 'Advanced', icon: '⚡' },
  ];

  let activeTab = $state('general');
  
  // Settings state
  let workspaceName = $state(appStore.workspace);
  let theme = $state(appStore.theme);
  let serverUrl = $state('https://api.oblivra.io');
  let apiToken = $state('');

  function saveGeneral() {
    appStore.setWorkspace(workspaceName as any);
    appStore.setTheme(theme);
    appStore.notify('General settings saved', 'success');
  }
</script>

<PageLayout title="System Settings" subtitle="Configure platform behavior and environment">
  <div class="flex flex-col h-full gap-5">
    <Tabs tabs={settingsTabs} bind:active={activeTab} />

    <div class="max-w-2xl">
      {#if activeTab === 'general'}
        <div class="flex flex-col gap-6">
          <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-4">
            <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Appearance & Workspace</h3>
            <Input label="Workspace Name" bind:value={workspaceName} />
            
            <div class="flex flex-col gap-1.5">
              <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted font-[var(--font-ui)]">Theme</div>
              <div class="flex gap-2">
                <Button variant={theme === 'dark' ? 'primary' : 'secondary'} size="sm" onclick={() => theme = 'dark'}>Tokyo Night (Dark)</Button>
                <Button variant={theme === 'light' ? 'primary' : 'secondary'} size="sm" onclick={() => theme = 'light'}>Paper (Light)</Button>
              </div>
            </div>

            <div class="pt-2 flex justify-end">
              <Button variant="cta" size="sm" onclick={saveGeneral}>Apply Changes</Button>
            </div>
          </div>

          <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-4">
            <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Language & Locale</h3>
            <div class="text-[11px] text-text-muted">Interface language: <span class="text-text-primary font-bold">English (US)</span></div>
          </div>
        </div>
      {:else if activeTab === 'server'}
        <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-4">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Remote Connection</h3>
          <p class="text-[11px] text-text-muted leading-relaxed">
            Configure a remote OBLIVRA server to enable <Badge variant="warning">Hybrid Mode</Badge>. 
            This allows viewing server-side logs and managing cloud agents from your desktop.
          </p>
          <Input label="Server URL" placeholder="https://..." bind:value={serverUrl} />
          <Input label="API Key / Token" type="password" placeholder="••••••••••••••••" bind:value={apiToken} />
          
          <div class="pt-2 flex justify-end gap-2">
            <Button variant="secondary" size="sm" onclick={async () => {
              try {
                const res = await fetch(serverUrl + '/healthz', { headers: apiToken ? { 'X-API-Key': apiToken } : {} });
                appStore.notify(res.ok ? `Connected (HTTP ${res.status})` : `Server returned ${res.status}`, res.ok ? 'success' : 'warning');
              } catch (e: any) {
                appStore.notify(`Connection failed: ${e?.message ?? e}`, 'error');
              }
            }}>Test Connection</Button>
            <Button variant="cta" size="sm" onclick={() => {
              try { localStorage.setItem('oblivra:serverUrl', serverUrl); if (apiToken) localStorage.setItem('oblivra:apiKey', apiToken); appStore.notify('Server config saved', 'success'); }
              catch { appStore.notify('Could not persist (localStorage unavailable)', 'warning'); }
            }}>Save Server Config</Button>
          </div>
        </div>
      {:else if activeTab === 'security'}
        <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-4">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Security</h3>
          <p class="text-[11px] text-text-muted">
            Vault, MFA, and identity controls live in their dedicated pages. Use the buttons below.
          </p>
          <div class="flex gap-2">
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/vault')}>Open Vault</Button>
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/identity-admin')}>Identity Admin</Button>
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/secrets')}>Secret Manager</Button>
          </div>
        </div>
      {:else}
        <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-4">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Advanced</h3>
          <p class="text-[11px] text-text-muted">
            Diagnostics, telemetry retention, kill-switch and air-gap mode controls live behind dedicated pages.
          </p>
          <div class="flex gap-2 flex-wrap">
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/monitoring')}>Pipeline Health</Button>
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/sync')}>Sync</Button>
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/license')}>License</Button>
            <Button variant="secondary" size="sm" onclick={() => appStore.navigate('/data-destruction')}>Disaster Controls</Button>
          </div>
        </div>
      {/if}
    </div>
  </div>
</PageLayout>
