<!--
  OBLIVRA — Settings (Svelte 5)
  Application configuration and system management.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { appStore, PROFILE_LABELS, type OperatorProfileId } from '@lib/stores/app.svelte';
  import { Badge, PageLayout, Button, Tabs, Input } from '@components/ui';
  import { apiFetch } from '@lib/apiClient';

  const settingsTabs = [
    { id: 'general', label: 'General', icon: '⚙️' },
    { id: 'profile', label: 'Operator Profile', icon: '👤' },
    { id: 'server', label: 'Server Connection', icon: '🌐' },
    { id: 'security', label: 'Security', icon: '🛡️' },
    { id: 'advanced', label: 'Advanced', icon: '⚡' },
  ];

  const profileIds: OperatorProfileId[] = [
    'soc-analyst',
    'threat-hunter',
    'incident-commander',
    'msp-admin',
    'custom',
  ];

  let activeTab = $state('general');

  // Settings state — hydrated from /api/v1/settings/<key> on mount
  // (server is the source of truth) and falls back to localStorage if
  // the server is offline so the UI still feels alive.
  let workspaceName = $state(appStore.workspace);
  let theme = $state(appStore.theme);
  let serverUrl = $state('https://api.oblivra.io');
  let apiToken = $state('');

  /** Pull a setting value from the server, falling back to localStorage. */
  async function loadSetting(key: string, fallback = ''): Promise<string> {
    try {
      const res = await apiFetch(`/api/v1/settings/${encodeURIComponent(key)}`);
      if (res.ok) {
        const body = await res.json();
        return body.value ?? fallback;
      }
    } catch { /* offline — use fallback */ }
    try {
      return localStorage.getItem(`oblivra:setting:${key}`) ?? fallback;
    } catch { return fallback; }
  }

  /** Persist a setting via REST + mirror to localStorage as a fallback. */
  async function saveSetting(key: string, value: string): Promise<boolean> {
    try { localStorage.setItem(`oblivra:setting:${key}`, value); } catch { /* quota */ }
    try {
      const res = await apiFetch(`/api/v1/settings/${encodeURIComponent(key)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value }),
      });
      return res.ok;
    } catch {
      return false;
    }
  }

  onMount(async () => {
    // Hydrate from server. Errors fall through to the existing
    // appStore / localStorage values so the UI stays responsive.
    const [ws, th, sUrl] = await Promise.all([
      loadSetting('workspace_name', String(appStore.workspace ?? '')),
      loadSetting('theme', appStore.theme ?? 'dark'),
      loadSetting('server_url', 'https://api.oblivra.io'),
    ]);
    if (ws) workspaceName = ws as any;
    if (th) theme = th;
    if (sUrl) serverUrl = sUrl;
  });

  async function saveGeneral() {
    appStore.setWorkspace(workspaceName as any);
    appStore.setTheme(theme);
    // Persist server-side too so cross-device + tenant consistency.
    const [a, b] = await Promise.all([
      saveSetting('workspace_name', String(workspaceName)),
      saveSetting('theme', String(theme)),
    ]);
    if (a && b) {
      appStore.notify('General settings saved', 'success');
    } else {
      appStore.notify(
        'Saved locally only',
        'warning',
        'Server unreachable — settings persisted to localStorage and will sync on next reconnect.',
      );
    }
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
              <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted font-sans">Theme</div>
              <div class="flex gap-2 items-center">
                <Button variant={theme === 'dark' ? 'primary' : 'secondary'} size="sm" onclick={() => theme = 'dark'}>Tokyo Night (Dark)</Button>
                <!-- Phase 35 honesty fix — the light-mode button used to
                     accept the click and save 'light' to settings, but
                     no light-mode CSS exists in app.css so nothing
                     changed visually. That's a UI lie: operator clicks,
                     saves, nothing happens, trust drops. Disabled until
                     a real light-mode token pass lands; tooltip
                     explains why. -->
                <Button
                  variant="secondary"
                  size="sm"
                  disabled
                  title="Light mode pending — operator-grade dark theme is the only fully-styled option today. Tracked as a future design pass."
                >Paper (Light) — coming soon</Button>
              </div>
              <div class="text-[9px] text-text-muted">SIEM operators run dark by default; light-mode tokens are a separate design pass.</div>
            </div>

            <!-- Density toggle (UIUX_IMPROVEMENTS.md P0 #2). 'Comfortable'
                 is the new default at 12px body / 10px micro labels;
                 'Compact' returns to the legacy 11/9px ramp for power
                 users who want maximum information density. Persisted
                 to localStorage by appStore.setDensity. -->
            <div class="flex flex-col gap-1.5">
              <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted font-sans">Density</div>
              <div class="flex gap-2">
                <Button
                  variant={appStore.density === 'comfortable' ? 'primary' : 'secondary'}
                  size="sm"
                  onclick={() => appStore.setDensity('comfortable')}
                >Comfortable (12 px)</Button>
                <Button
                  variant={appStore.density === 'compact' ? 'primary' : 'secondary'}
                  size="sm"
                  onclick={() => appStore.setDensity('compact')}
                >Compact (11 px)</Button>
              </div>
              <p class="text-[10px] text-text-muted leading-relaxed">
                Compact restores the legacy SOC-dense layout. Comfortable adds ~1 px to every label and improves long-shift readability without disrupting layouts.
              </p>
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
      {:else if activeTab === 'profile'}
        <div class="flex flex-col gap-4 max-w-3xl">
          <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Active profile</h3>
                <p class="text-[11px] text-text-muted leading-relaxed mt-1 max-w-xl">
                  Profiles bundle 9 settings together (home route, density, palette behaviour, vim leader keys, tenant chrome, crisis affordance, alert noise floor, layout mode, primary metric). Picking a preset re-aligns the whole UI. Editing a single rule below auto-promotes you to <span class="font-mono text-accent">Custom</span>.
                </p>
              </div>
              <Badge variant={appStore.profile === 'custom' ? 'warning' : 'success'}>{PROFILE_LABELS[appStore.profile].name}</Badge>
            </div>
          </div>

          <!-- Preset cards -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            {#each profileIds as id}
              {@const meta = PROFILE_LABELS[id]}
              {@const isActive = appStore.profile === id}
              <button
                class="text-start flex flex-col gap-2 p-4 bg-surface-1 border rounded-md transition-colors {isActive ? 'border-accent bg-accent/5' : 'border-border-primary hover:border-border-hover'}"
                onclick={() => id !== 'custom' && appStore.setProfile(id)}
                disabled={id === 'custom'}
                title={id === 'custom' ? 'Auto-selected when you override individual rules below.' : `Switch to ${meta.name}`}
              >
                <div class="flex items-center justify-between">
                  <span class="text-[12px] font-bold text-text-heading">{meta.name}</span>
                  {#if isActive}<Badge variant="success" size="xs">ACTIVE</Badge>{/if}
                </div>
                <span class="text-[var(--fs-micro)] font-mono text-text-muted uppercase tracking-widest">{id}</span>
                <p class="text-[var(--fs-label)] text-text-secondary leading-relaxed">{meta.subtitle}</p>
              </button>
            {/each}
          </div>

          <!-- Per-rule override grid -->
          <div class="p-4 bg-surface-1 border border-border-primary rounded-md flex flex-col gap-3">
            <h3 class="text-xs font-bold uppercase tracking-widest text-text-heading">Rule overrides</h3>
            <p class="text-[11px] text-text-muted">Editing any of these promotes the profile to <span class="font-mono text-accent">Custom</span>.</p>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-3 mt-1">
              <!-- Home route -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Home route</span>
                <input
                  type="text"
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs font-mono"
                  value={appStore.profileRules.homeRoute}
                  oninput={(e) => appStore.setProfileRule('homeRoute', (e.currentTarget as HTMLInputElement).value)}
                  placeholder="/dashboard"
                />
              </label>

              <!-- Density -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Density</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.defaultDensity}
                  onchange={(e) => appStore.setProfileRule('defaultDensity', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="comfortable">Comfortable (12 px)</option>
                  <option value="compact">Compact (11 px)</option>
                </select>
              </label>

              <!-- Primary metric -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Primary metric</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.primaryMetric}
                  onchange={(e) => appStore.setProfileRule('primaryMetric', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="mttr">MTTR (response speed)</option>
                  <option value="fp-rate">False-positive rate</option>
                  <option value="evidence-latency">Evidence-export latency</option>
                  <option value="hunt-yield">Hunt yield</option>
                </select>
              </label>

              <!-- Layout mode -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Layout mode</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.layoutMode}
                  onchange={(e) => appStore.setProfileRule('layoutMode', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="single">Single screen</option>
                  <option value="war-room">War-room (multi-monitor)</option>
                </select>
              </label>

              <!-- Tenant chrome -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Tenant indicator</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.tenantChrome}
                  onchange={(e) => appStore.setProfileRule('tenantChrome', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="breadcrumb">Breadcrumb (8 px)</option>
                  <option value="badge">Badge (always visible)</option>
                  <option value="switcher-bar">Switcher bar (Cmd+T fast-switch)</option>
                </select>
              </label>

              <!-- Crisis affordance -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Crisis Mode behaviour</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.crisisAffordance}
                  onchange={(e) => appStore.setProfileRule('crisisAffordance', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="banner">Banner (compact)</option>
                  <option value="fullscreen-takeover">Fullscreen takeover</option>
                </select>
              </label>

              <!-- Noise floor -->
              <label class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Alert noise floor</span>
                <select
                  class="bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs"
                  value={appStore.profileRules.alertNoiseFloor}
                  onchange={(e) => appStore.setProfileRule('alertNoiseFloor', (e.currentTarget as HTMLSelectElement).value as any)}
                >
                  <option value="critical-only">Critical only</option>
                  <option value="high+">High and above</option>
                  <option value="medium+">Medium and above</option>
                  <option value="all">All severities</option>
                </select>
              </label>

              <!-- Boolean toggles -->
              <div class="flex flex-col gap-1">
                <span class="text-[10px] font-bold uppercase tracking-wider text-text-muted">Behaviours</span>
                <div class="flex flex-col gap-1.5 pt-1">
                  <label class="flex items-center gap-2 text-[11px]">
                    <input
                      type="checkbox"
                      checked={appStore.profileRules.paletteFront}
                      onchange={(e) => appStore.setProfileRule('paletteFront', (e.currentTarget as HTMLInputElement).checked)}
                    />
                    <span class="text-text-secondary">Palette as front door (⌘K-first)</span>
                  </label>
                  <label class="flex items-center gap-2 text-[11px]">
                    <input
                      type="checkbox"
                      checked={appStore.profileRules.vimLeader}
                      onchange={(e) => appStore.setProfileRule('vimLeader', (e.currentTarget as HTMLInputElement).checked)}
                    />
                    <span class="text-text-secondary">Vim-style g+letter navigation leader</span>
                  </label>
                </div>
              </div>
            </div>

            <div class="pt-2 flex items-center justify-between">
              <button
                class="text-[10px] font-mono text-text-muted hover:text-text-secondary uppercase tracking-widest"
                onclick={() => { localStorage.removeItem('oblivra:profileChosen'); appStore.profileChosen = false; }}
                title="Show the first-run wizard again"
              >▸ Re-run wizard</button>
              {#if appStore.profile === 'custom'}
                <Button variant="secondary" size="sm" onclick={() => appStore.setProfile('soc-analyst')}>Reset to SOC Analyst preset</Button>
              {/if}
            </div>
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
            <Button variant="cta" size="sm" onclick={async () => {
              // Persist via the settings endpoint so cross-device sync
              // works. The api_key value is auto-detected as sensitive
              // by the SettingsService isSensitiveKey gate and gets
              // vault-encrypted at rest.
              const [a, b] = await Promise.all([
                saveSetting('server_url', serverUrl),
                apiToken ? saveSetting('api_key', apiToken) : Promise.resolve(true),
              ]);
              // Local mirror so connection-test paths that read from
              // localStorage continue to work even when the server
              // round-trip lags.
              try { localStorage.setItem('oblivra:serverUrl', serverUrl); if (apiToken) localStorage.setItem('oblivra:apiKey', apiToken); } catch { /* quota */ }
              if (a && b) {
                appStore.notify('Server config saved', 'success');
              } else {
                appStore.notify('Server config saved locally only', 'warning', 'Backend unreachable — will sync on next reconnect.');
              }
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
