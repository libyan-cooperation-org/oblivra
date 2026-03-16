import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { CertificateManager } from '../auth/CertificateManager';
// Wails bindings
import { GetAllHealth } from '../../../wailsjs/go/services/HealthService';
import { CheckForUpdate, ApplyUpdate } from '../../../wailsjs/go/services/UpdaterService';
import { AttestationTab } from './AttestationTab';
import { TemporalTab } from './TemporalTab';
import { DataDestructionTab } from './DataDestructionTab';
import '@styles/settings.css';

const settingsSvc = (window as any).go?.services?.SettingsService;
// (Theme service was removed)

export const SettingsManager: Component = () => {
    const [activeTab, setActiveTab] = createSignal<'general' | 'auth' | 'observability' | 'updates' | 'attestation' | 'temporal' | 'destruction' | 'notifications'>('general');

    const [fontFamily, setFontFamily] = createSignal<string>("'JetBrains Mono', monospace");
    const [fontSize, setFontSize] = createSignal<number>(14);
    const [savedMessage, setSavedMessage] = createSignal<string>('');

    // Observability State
    type HealthInfo = { host_name?: string, host_id?: string, latency_ms?: number, status?: string };
    const [health, setHealth] = createSignal<Record<string, HealthInfo>>({});

    // Notifications State
    const [slackEnabled, setSlackEnabled] = createSignal<boolean>(false);
    const [slackUrl, setSlackUrl] = createSignal<string>('');
    const [slackChannel, setSlackChannel] = createSignal<string>('');
    const [teamsEnabled, setTeamsEnabled] = createSignal<boolean>(false);
    const [teamsUrl, setTeamsUrl] = createSignal<string>('');

    // Updates State
    const [updateInfo, setUpdateInfo] = createSignal<any>(null);
    const [checkingUpdate, setCheckingUpdate] = createSignal(false);

    const loadObservability = async () => {
        try {
            const h = await GetAllHealth();
            setHealth((h || {}) as Record<string, HealthInfo>);
        } catch (err) {
            console.error('Failed to load observability data:', err);
        }
    };

    const handleCheckUpdate = async () => {
        setCheckingUpdate(true);
        try {
            const info = await CheckForUpdate();
            setUpdateInfo(info);
        } catch (err) {
            console.error('Failed update check', err);
        } finally {
            setCheckingUpdate(false);
        }
    };

    const handleApplyUpdate = async () => {
        try {
            await ApplyUpdate();
        } catch (err) {
            console.error('Failed to apply update', err);
        }
    };


    const loadSettings = async () => {
        if (settingsSvc) {
            try {
                const ff = await settingsSvc.Get("terminal.fontFamily");
                if (ff) setFontFamily(ff);

                const fs = await settingsSvc.Get("terminal.fontSize");
                if (fs) setFontSize(parseInt(fs, 10));

                const se = await settingsSvc.Get("notifications.slackEnabled");
                if (se) setSlackEnabled(se === 'true');
                const su = await settingsSvc.Get("notifications.slackUrl");
                if (su) setSlackUrl(su);
                const sc = await settingsSvc.Get("notifications.slackChannel");
                if (sc) setSlackChannel(sc);

                const te = await settingsSvc.Get("notifications.teamsEnabled");
                if (te) setTeamsEnabled(te === 'true');
                const tu = await settingsSvc.Get("notifications.teamsUrl");
                if (tu) setTeamsUrl(tu);
            } catch (e) { console.warn("No custom settings yet"); }
        }
    };

    onMount(() => {
        loadSettings();
    });

    const handleSave = async () => {
        try {
            if (settingsSvc) {
                await settingsSvc.Set("terminal.fontFamily", fontFamily());
                await settingsSvc.Set("terminal.fontSize", fontSize().toString());

                await settingsSvc.Set("notifications.slackEnabled", slackEnabled() ? 'true' : 'false');
                await settingsSvc.Set("notifications.slackUrl", slackUrl());
                await settingsSvc.Set("notifications.slackChannel", slackChannel());

                await settingsSvc.Set("notifications.teamsEnabled", teamsEnabled() ? 'true' : 'false');
                await settingsSvc.Set("notifications.teamsUrl", teamsUrl());
            }
            setSavedMessage('Settings saved successfully!');
            setTimeout(() => setSavedMessage(''), 3000);

            // Note: Since terminal re-renders on mount, you may need to dispatch a specific 'settings.changed' event
            // to dynamically update Terminal without a reload. The backend 'Set' method actually triggers `settings.changed` event!
        } catch (err) {
            alert(`Failed to save settings: ${err}`);
        }
    };

    return (
        <div class="settings-container">
            <div class="settings-header">
                <h2>Settings</h2>
                {savedMessage() && <span class="saved-message">{savedMessage()}</span>}
            </div>

            <div class="settings-tabs">
                <button class={`tab-btn ${activeTab() === 'general' ? 'active' : ''}`} onClick={() => setActiveTab('general')}>General</button>
                <button class={`tab-btn ${activeTab() === 'auth' ? 'active' : ''}`} onClick={() => setActiveTab('auth')}>Auth</button>
                <button class={`tab-btn ${activeTab() === 'notifications' ? 'active' : ''}`} onClick={() => setActiveTab('notifications')}>Notifications</button>
                <button class={`tab-btn ${activeTab() === 'observability' ? 'active' : ''}`} onClick={() => { setActiveTab('observability'); loadObservability(); }}>Observability</button>
                <button class={`tab-btn ${activeTab() === 'updates' ? 'active' : ''}`} onClick={() => setActiveTab('updates')}>Updates</button>
                <div style="height: 20px;" />
                <div style="font-size: 11px; text-transform: uppercase; font-weight: bold; color: var(--text-muted); margin-bottom: 8px; margin-left: 12px;">Sovereign Capabilities</div>
                <button class={`tab-btn ${activeTab() === 'attestation' ? 'active' : ''}`} onClick={() => setActiveTab('attestation')}>Runtime Attestation</button>
                <button class={`tab-btn ${activeTab() === 'temporal' ? 'active' : ''}`} onClick={() => setActiveTab('temporal')}>Temporal Integrity</button>
                <button class={`tab-btn ${activeTab() === 'destruction' ? 'active' : ''}`} onClick={() => setActiveTab('destruction')} style="color: var(--error);">Data Destruction</button>
            </div>

            <div class="settings-content" style="flex: 1; overflow-y: auto;">
                <Show when={activeTab() === 'general'}>


                    <section class="settings-section">
                        <h3>Terminal Preferences</h3>

                        <div class="form-group row">
                            <label>Font Family</label>
                            <input
                                type="text"
                                value={fontFamily()}
                                onInput={e => setFontFamily(e.currentTarget.value)}
                            />
                        </div>

                        <div class="form-group row">
                            <label>Font Size</label>
                            <input
                                type="number"
                                value={fontSize()}
                                onInput={e => setFontSize(parseInt(e.currentTarget.value) || 14)}
                            />
                        </div>
                    </section>
                </Show>

                <Show when={activeTab() === 'auth'}>
                    <section class="settings-section">
                        <CertificateManager />
                    </section>
                </Show>

                <Show when={activeTab() === 'notifications'}>
                    <section class="settings-section">
                        <h3>SOAR Integrations</h3>
                        <p class="settings-hint">Configure external channels for security alerts.</p>

                        <div style="margin-top: 16px; margin-bottom: 24px;">
                            <h4 style="color: var(--text-primary); margin-bottom: 12px; display: flex; align-items: center; gap: 8px;">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"></path></svg>
                                Slack
                            </h4>
                            <div class="form-group row" style="align-items: center;">
                                <label style="flex: 1;">Enable Slack Notifications</label>
                                <input type="checkbox" checked={slackEnabled()} onChange={(e) => setSlackEnabled(e.target.checked)} />
                            </div>
                            <div class="form-group row">
                                <label>Webhook URL</label>
                                <input type="password" placeholder="https://hooks.slack.com/services/..." value={slackUrl()} onInput={e => setSlackUrl(e.currentTarget.value)} />
                            </div>
                            <div class="form-group row">
                                <label>Default Channel</label>
                                <input type="text" placeholder="#security-alerts" value={slackChannel()} onInput={e => setSlackChannel(e.currentTarget.value)} />
                            </div>
                        </div>

                        <div style="border-top: 1px solid var(--border-primary); margin-top: 24px; padding-top: 24px;">
                            <h4 style="color: var(--text-primary); margin-bottom: 12px; display: flex; align-items: center; gap: 8px;">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path><circle cx="9" cy="7" r="4"></circle><path d="M23 21v-2a4 4 0 0 0-3-3.87"></path><path d="M16 3.13a4 4 0 0 1 0 7.75"></path></svg>
                                Microsoft Teams
                            </h4>
                            <div class="form-group row" style="align-items: center;">
                                <label style="flex: 1;">Enable Teams Notifications</label>
                                <input type="checkbox" checked={teamsEnabled()} onChange={(e) => setTeamsEnabled(e.target.checked)} />
                            </div>
                            <div class="form-group row">
                                <label>Webhook URL (Workflows)</label>
                                <input type="password" placeholder="https://prod-1...logic.azure.com/..." value={teamsUrl()} onInput={e => setTeamsUrl(e.currentTarget.value)} />
                            </div>
                        </div>
                    </section>
                </Show>

                <Show when={activeTab() === 'observability'}>
                    <section class="settings-section">
                        <h3>Host Health</h3>
                        <For each={Object.values(health())} fallback={<div class="empty-hint">No hosts monitored.</div>}>
                            {(h) => (
                                <div class="health-card">
                                    <div>
                                        <div class="health-name">{h.host_name || h.host_id}</div>
                                        <div class="health-latency">Latency: {h.latency_ms}ms</div>
                                    </div>
                                    <span class={`status-dot ${h.status === 'healthy' ? 'online' : 'offline'}`}></span>
                                </div>
                            )}
                        </For>
                    </section>
                </Show>

                <Show when={activeTab() === 'updates'}>
                    <section class="settings-section updates-section">
                        <h3>Application Updates</h3>
                        <p class="settings-hint">Checking for latest features and security patches.</p>

                        <button class="btn-primary" onClick={handleCheckUpdate} disabled={checkingUpdate()}>
                            {checkingUpdate() ? 'Checking...' : 'Check now'}
                        </button>

                        <Show when={updateInfo() && updateInfo().available}>
                            <div class="update-banner">
                                <div class="update-title">Version {updateInfo().version} Available</div>
                                <button class="btn-success" onClick={handleApplyUpdate}>Install Update & Restart</button>
                            </div>
                        </Show>
                        <Show when={updateInfo() && !updateInfo().available}>
                            <div class="update-status">You are up to date.</div>
                        </Show>
                    </section>
                </Show>

                <Show when={activeTab() === 'attestation'}>
                    <section class="settings-section" style="padding: 0;">
                        <AttestationTab />
                    </section>
                </Show>

                <Show when={activeTab() === 'temporal'}>
                    <section class="settings-section" style="padding: 0;">
                        <TemporalTab />
                    </section>
                </Show>

                <Show when={activeTab() === 'destruction'}>
                    <section class="settings-section" style="padding: 0;">
                        <DataDestructionTab />
                    </section>
                </Show>
            </div>

            <div class="settings-footer">
                <button class="btn-success" onClick={handleSave}>Save Settings</button>
            </div>
        </div>
    );
};
