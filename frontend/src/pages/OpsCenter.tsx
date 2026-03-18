import { Component, createSignal, For, Show } from 'solid-js';
import { LogDetail } from '../components/analytics/LogDetail';
import { OQLDashboard } from './OQLDashboard';
import { FusionDashboard } from './FusionDashboard';
import { SourcesPanel } from '../components/ops/SourcesPanel';
import { SearchLogs, RunOsquery } from '../../wailsjs/go/services/AnalyticsService';
import {
    GetOsqueryTemplates, GetTriggers, AddTrigger, RemoveTrigger,
    GetAlertHistory, UpdateNotificationConfig, GetNotificationConfig, TestNotification
} from '../../wailsjs/go/services/AlertingService';
import { FleetDashboard } from '../components/fleet/FleetDashboard';
import { AgentConsole } from '../components/fleet/AgentConsole';
import { analytics, notifications, osquery } from '../../wailsjs/go/models';
import { EmptyState } from '../components/ui/EmptyState';
import { Card, Badge, Button } from '../components/ui/TacticalComponents';
import { LiveTailPanel } from '../components/ops/LiveTailPanel';
import { SyntheticMonitor } from '../components/ops/SyntheticMonitor';
import '../styles/ops_center.css';

type TabView = 'dashboard' | 'agents' | 'fusion' | 'search' | 'sources' | 'tail' | 'synthetic' | 'alerts' | 'history';

interface AlertHistoryEvent {
    timestamp: string;
    severity: 'critical' | 'high' | 'medium' | 'low';
    name: string;
    host: string;
    sent: boolean;
}


export const OpsCenter: Component = () => {
    const [activeTab, setActiveTab] = createSignal<TabView>('dashboard');
    const [mode, setMode] = createSignal<'logql' | 'lucene' | 'sql' | 'osquery'>('logql');
    const [query, setQuery] = createSignal('{host="*"}');
    const [results, setResults] = createSignal<Record<string, unknown>[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);
    const [limit] = createSignal(100);
    const [offset, setOffset] = createSignal(0);

    // Osquery templates
    const [templates, setTemplates] = createSignal<osquery.QueryTemplate[]>([]);

    // Alert state
    const [triggers, setTriggers] = createSignal<analytics.Trigger[]>([]);
    const [alertHistory, setAlertHistory] = createSignal<AlertHistoryEvent[]>([]);

    // New trigger form
    const [newTriggerName, setNewTriggerName] = createSignal('');
    const [newTriggerPattern, setNewTriggerPattern] = createSignal('');
    const [newTriggerSeverity, setNewTriggerSeverity] = createSignal('critical');

    // New metric trigger form (removed unused signal)

    // Notification config
    const [notifConfig, setNotifConfig] = createSignal<Partial<notifications.NotificationConfig>>({});
    const [configDirty, setConfigDirty] = createSignal(false);

    // Dashboard sub-tab
    const [dashboardMode, setDashboardMode] = createSignal<'telemetry' | 'analytics'>('telemetry');

    const loadTemplates = async () => {
        try {
            const t = await GetOsqueryTemplates();
            setTemplates(t || []);
        } catch { /* ignore */ }
    };

    const loadAlerts = async () => {
        try {
            const [trigs, history, config] = await Promise.all([
                GetTriggers(),
                GetAlertHistory(),
                GetNotificationConfig()
            ]);
            setTriggers(trigs || []);
            setAlertHistory((history as unknown as AlertHistoryEvent[]) || []);
            setNotifConfig(config || {});
        } catch { /* ignore */ }
    };

    const handleModeChange = (newMode: string) => {
        setMode(newMode as any);
        if (newMode === 'osquery' && templates().length === 0) {
            loadTemplates();
        }
    };

    const runSearch = async (isLoadMore = false) => {
        if (!isLoadMore) {
            setOffset(0);
        }
        setLoading(true);
        setError(null);
        try {
            if (mode() === 'osquery') {
                const res = await RunOsquery(query());
                setResults(res || []);
            } else {
                const res = await SearchLogs(query(), mode(), limit(), offset());
                if (isLoadMore) {
                    setResults(prev => [...prev, ...(res || [])]);
                } else {
                    setResults(res || []);
                }
            }
        } catch (err: unknown) {
            setError(err instanceof Error ? (err as Error).message : String(err));
            if (!isLoadMore) setResults([]);
        } finally {
            setLoading(false);
        }
    };

    const addFilter = (key: string, value: string, operator: '=' | '!=') => {
        let currentQ = query();
        if (mode() === 'logql') {
            const filterStr = operator === '=' ? `|= "${key}=${value}"` : `|!= "${key}=${value}"`;
            setQuery(currentQ + " " + filterStr);
            runSearch(false);
        } else if (mode() === 'lucene') {
            const filterStr = operator === '=' ? `AND ${key}:${value}` : `AND NOT ${key}:${value}`;
            setQuery(currentQ + " " + filterStr);
            runSearch(false);
        }
    };

    const handleAddTrigger = async () => {
        if (!newTriggerName() || !newTriggerPattern()) return;
        const id = 'custom-' + Date.now();
        try {
            await AddTrigger(id, newTriggerName(), newTriggerPattern(), newTriggerSeverity());
            setNewTriggerName('');
            setNewTriggerPattern('');
            loadAlerts();
        } catch (e: unknown) {
            setError(e instanceof Error ? (e as Error).message : String(e));
        }
    };

    const handleRemoveTrigger = async (id: string) => {
        try {
            await RemoveTrigger(id);
            loadAlerts();
        } catch (e: unknown) {
            setError(e instanceof Error ? (e as Error).message : String(e));
        }
    };

    const updateCfg = (key: keyof notifications.NotificationConfig, value: string | number | boolean) => {
        setNotifConfig((prev) => ({ ...prev, [key]: value }));
        setConfigDirty(true);
    };

    const saveNotifConfig = async () => {
        try {
            await UpdateNotificationConfig(notifConfig() as notifications.NotificationConfig);
            setConfigDirty(false);
        } catch (e: unknown) {
            setError(e instanceof Error ? (e as Error).message : String(e));
        }
    };

    const testNotif = async () => {
        try {
            await TestNotification();
        } catch (e: unknown) {
            setError(e instanceof Error ? (e as Error).message : String(e));
        }
    };

    return (
        <div class="ops-center-layout">
            {/* Header */}
            <div class="ob-tabs" style={{ padding: '0 16px', 'flex-shrink': '0', background: 'var(--surface-1)', 'border-bottom': '1px solid var(--border-primary)' }}>
                <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', 'margin-right': '20px', 'padding': '0 0 0 0' }}>
                    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="var(--accent-primary)" stroke-width="2">
                        <line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" />
                    </svg>
                    <span style={{ 'font-size': '11px', 'font-weight': '700', 'text-transform': 'uppercase', 'letter-spacing': '0.5px', color: 'var(--text-secondary)' }}>Ops Center</span>
                </div>
                {(['dashboard','agents','fusion','search','sources','tail','synthetic','alerts','history'] as TabView[]).map(t => (
                    <button
                        class={`ob-tab${activeTab() === t ? ' active' : ''}`}
                        onClick={() => { setActiveTab(t); if (t === 'alerts' || t === 'history') loadAlerts(); }}
                    >{t.replace('-', ' ').toUpperCase()}</button>
                ))}
            </div>

            <Show when={error()}>
                <div class="ops-error">{error()}</div>
            </Show>

            {/* ===== DASHBOARD TAB ===== */}
            <Show when={activeTab() === 'dashboard'}>
                <div class="dashboard-container">
                    <div class="dashboard-sub-toggle">
                        <Button variant={dashboardMode() === 'telemetry' ? 'primary' : 'secondary'} size="sm" onClick={() => setDashboardMode('telemetry')}>
                            📡 Live Telemetry
                        </Button>
                        <Button variant={dashboardMode() === 'analytics' ? 'primary' : 'secondary'} size="sm" onClick={() => setDashboardMode('analytics')}>
                            📈 Infrastructure Analytics
                        </Button>
                    </div>

                    <Show when={dashboardMode() === 'telemetry'}>
                        <FleetDashboard />
                    </Show>
                    <Show when={dashboardMode() === 'analytics'}>
                        <OQLDashboard />
                    </Show>
                </div>
            </Show>

            {/* ===== AGENTS TAB ===== */}
            <Show when={activeTab() === 'agents'}>
                <AgentConsole />
            </Show>

            {/* ===== FUSION TAB ===== */}
            <Show when={activeTab() === 'fusion'}>
                <FusionDashboard />
            </Show>

            {/* ===== SOURCES TAB ===== */}
            <Show when={activeTab() === 'sources'}>
                <SourcesPanel />
            </Show>

            {/* ===== LIVE TAIL TAB ===== */}
            <Show when={activeTab() === 'tail'}>
                <LiveTailPanel />
            </Show>

            {/* ===== SYNTHETIC TAB ===== */}
            <Show when={activeTab() === 'synthetic'}>
                <SyntheticMonitor />
            </Show>

            {/* ===== SEARCH TAB ===== */}
            <Show when={activeTab() === 'search'}>
                <div class="search-panel">
                    <div class="ops-search-bar">
                        <select class="ops-select" aria-label="Select query language mode" value={mode()} onChange={e => handleModeChange(e.currentTarget.value)}>
                            <option value="logql">LogQL</option>
                            <option value="lucene">Lucene</option>
                            <option value="sql">Raw SQL</option>
                            <option value="osquery">Osquery (Live)</option>
                        </select>

                        <Show when={mode() === 'osquery'}>
                            <select class="ops-select template-select" aria-label="Load Osquery Template" onChange={e => { if (e.currentTarget.value) setQuery(e.currentTarget.value); }}>
                                <option value="">Load Template...</option>
                                <For each={templates()}>
                                    {(t) => <option value={t.sql}>{t.category} — {t.name}</option>}
                                </For>
                            </select>
                        </Show>

                        <input class="ops-input query-input" type="text"
                            aria-label="Query Search Input"
                            placeholder={
                                mode() === 'logql' ? '{host="web"} |= "error"' :
                                    mode() === 'lucene' ? 'host:web AND message:error' :
                                        mode() === 'osquery' ? 'SELECT * FROM processes LIMIT 10;' :
                                            'SELECT * FROM terminal_logs WHERE...'
                            }
                            value={query()} onInput={e => setQuery(e.currentTarget.value)}
                            onKeyDown={e => e.key === 'Enter' && runSearch(false)} />

                        <Button variant="primary" size="sm" onClick={() => runSearch(false)} disabled={loading()}>
                            {loading() ? '⏳' : '▶'} {loading() ? 'Searching...' : 'Run'}
                        </Button>
                    </div>

                    <div class="ops-results">
                        <div class="results-header">
                            <span class="result-count">{results().length} results</span>
                            <span class="result-mode">{mode().toUpperCase()}</span>
                        </div>
                        <div class="results-stream">
                            <For each={results()}>
                                {(res) => (
                                    mode() === 'osquery' ? (
                                        <pre class="osquery-result">{JSON.stringify(res, null, 2)}</pre>
                                    ) : (
                                        <LogDetail log={res} onAddFilter={addFilter} />
                                    )
                                )}
                            </For>
                            <Show when={results().length === 0 && !loading() && !error()}>
                                <EmptyState
                                    icon="📊"
                                    title="No Results"
                                    description="Run a query to search terminal logs, security events, or live forensics."
                                    compact
                                />
                            </Show>
                            <Show when={results().length > 0 && mode() !== 'osquery' && results().length % limit() === 0}>
                                <div class="load-more-container" style="text-align: center; margin-top: 10px; padding-bottom: 20px;">
                                    <button class="ops-btn" onClick={() => { setOffset(offset() + limit()); runSearch(true); }} disabled={loading()}>
                                        {loading() ? '⏳ Loading...' : 'Load More'}
                                    </button>
                                </div>
                            </Show>
                        </div>
                    </div>
                </div>
            </Show>

            {/* ===== ALERTS TAB ===== */}
            <Show when={activeTab() === 'alerts'}>
                <div class="alerts-panel">
                    {/* Notification Channels Config */}
                    <div class="alerts-section">
                        <div class="section-header">
                            <h3 style="font-family: var(--font-ui); text-transform: uppercase; letter-spacing: 1px;">📡 Notification Channels</h3>
                            <div style="display: flex; gap: 8px;">
                                <Button variant="secondary" size="sm" onClick={testNotif}>TEST CONNECTION</Button>
                                <Show when={configDirty()}>
                                    <Button variant="primary" size="sm" onClick={saveNotifConfig}>SAVE CONFIGURATION</Button>
                                </Show>
                            </div>
                        </div>
                        <p class="config-hint">Credentials are stored locally. You connect directly to providers — sovereign, no middleman.</p>

                        {/* Email */}
                        <div class="channel-group">
                            <div class="channel-header-row">
                                <span>📧 Email (SMTP)</span>
                                <label class="toggle-label">
                                    <input type="checkbox" checked={notifConfig().enable_email || false}
                                        onChange={e => updateCfg('enable_email', e.currentTarget.checked)} />
                                    {notifConfig().enable_email ? 'Enabled' : 'Disabled'}
                                </label>
                            </div>
                            <Show when={notifConfig().enable_email}>
                                <div class="form-grid">
                                    <input class="ops-input" placeholder="SMTP Host (smtp.gmail.com)" value={notifConfig().smtp_host || ''}
                                        onInput={e => updateCfg('smtp_host', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Port (587)" type="number" value={notifConfig().smtp_port || ''}
                                        onInput={e => updateCfg('smtp_port', parseInt(e.currentTarget.value) || 0)} />
                                    <input class="ops-input" placeholder="Username / Email" value={notifConfig().smtp_username || ''}
                                        onInput={e => updateCfg('smtp_username', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Password / App Password" type="password" value={notifConfig().smtp_password || ''}
                                        onInput={e => updateCfg('smtp_password', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Send To Email" value={notifConfig().to_email || ''}
                                        onInput={e => updateCfg('to_email', e.currentTarget.value)} />
                                </div>
                            </Show>
                        </div>

                        {/* Telegram */}
                        <div class="channel-group">
                            <div class="channel-header-row">
                                <span>✈️ Telegram</span>
                                <label class="toggle-label">
                                    <input type="checkbox" checked={notifConfig().enable_telegram || false}
                                        onChange={e => updateCfg('enable_telegram', e.currentTarget.checked)} />
                                    {notifConfig().enable_telegram ? 'Enabled' : 'Disabled'}
                                </label>
                            </div>
                            <Show when={notifConfig().enable_telegram}>
                                <div class="form-grid">
                                    <input class="ops-input" placeholder="Bot Token (from @BotFather)" value={notifConfig().telegram_token || ''}
                                        onInput={e => updateCfg('telegram_token', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Chat ID (from @userinfobot)" value={notifConfig().telegram_chat_id || ''}
                                        onInput={e => updateCfg('telegram_chat_id', e.currentTarget.value)} />
                                </div>
                                <p class="config-hint">Start a chat with your bot first!</p>
                            </Show>
                        </div>

                        {/* Twilio SMS + WhatsApp */}
                        <div class="channel-group">
                            <div class="channel-header-row">
                                <span>💬 SMS & WhatsApp (Twilio)</span>
                                <label class="toggle-label">
                                    <input type="checkbox" checked={(notifConfig().enable_sms || notifConfig().enable_whatsapp) || false}
                                        onChange={e => { updateCfg('enable_sms', e.currentTarget.checked); updateCfg('enable_whatsapp', e.currentTarget.checked); }} />
                                    {(notifConfig().enable_sms || notifConfig().enable_whatsapp) ? 'Enabled' : 'Disabled'}
                                </label>
                            </div>
                            <Show when={notifConfig().enable_sms || notifConfig().enable_whatsapp}>
                                <div class="form-grid">
                                    <input class="ops-input" placeholder="Account SID" value={notifConfig().twilio_sid || ''}
                                        onInput={e => updateCfg('twilio_sid', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Auth Token" type="password" value={notifConfig().twilio_token || ''}
                                        onInput={e => updateCfg('twilio_token', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="From Number (+1234567890)" value={notifConfig().twilio_from || ''}
                                        onInput={e => updateCfg('twilio_from', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="To Number" value={notifConfig().to_phone || ''}
                                        onInput={e => updateCfg('to_phone', e.currentTarget.value)} />
                                </div>
                                <div class="checkbox-row">
                                    <label><input type="checkbox" checked={notifConfig().enable_sms || false}
                                        onChange={e => updateCfg('enable_sms', e.currentTarget.checked)} /> SMS</label>
                                    <label><input type="checkbox" checked={notifConfig().enable_whatsapp || false}
                                        onChange={e => updateCfg('enable_whatsapp', e.currentTarget.checked)} /> WhatsApp</label>
                                </div>
                            </Show>
                        </div>

                        {/* Webhook */}
                        <div class="channel-group">
                            <div class="channel-header-row">
                                <span>🔗 Webhook (Slack / Discord / Teams)</span>
                                <label class="toggle-label">
                                    <input type="checkbox" checked={notifConfig().enable_webhook || false}
                                        onChange={e => updateCfg('enable_webhook', e.currentTarget.checked)} />
                                    {notifConfig().enable_webhook ? 'Enabled' : 'Disabled'}
                                </label>
                            </div>
                            <Show when={notifConfig().enable_webhook}>
                                <div class="form-grid" style="grid-template-columns: 2fr 1fr;">
                                    <input class="ops-input" placeholder="Webhook URL" value={notifConfig().webhook_url || ''}
                                        onInput={e => updateCfg('webhook_url', e.currentTarget.value)} />
                                    <input class="ops-input" placeholder="Secret (optional)" value={notifConfig().webhook_secret || ''}
                                        onInput={e => updateCfg('webhook_secret', e.currentTarget.value)} />
                                </div>
                                <p class="config-hint">Supports rich formatting for Slack, Discord, and MS Teams natively.</p>
                            </Show>
                        </div>
                    </div>

                    {/* Alert Triggers */}
                    <div class="alerts-section">
                        <div class="section-header">
                            <h3>⚡ Alert Triggers (Regex)</h3>
                        </div>

                        <div class="rule-form">
                            <div class="form-row">
                                <input class="ops-input" placeholder="Name (e.g. SSH Brute Force)" value={newTriggerName()} onInput={e => setNewTriggerName(e.currentTarget.value)} />
                                <input class="ops-input" placeholder="Regex pattern (e.g. Failed password)" value={newTriggerPattern()} onInput={e => setNewTriggerPattern(e.currentTarget.value)} />
                                <select class="ops-select" value={newTriggerSeverity()} onChange={e => setNewTriggerSeverity(e.currentTarget.value)}>
                                    <option value="critical">🚨 Critical</option>
                                    <option value="high">🔴 High</option>
                                    <option value="medium">🟡 Medium</option>
                                    <option value="low">🔵 Low</option>
                                </select>
                                <button class="ops-btn" onClick={handleAddTrigger}>+ Add</button>
                            </div>
                        </div>

                        <div class="rules-list">
                            <For each={triggers()}>
                                {(t) => (
                                    <Card variant="raised" padding="10px 16px" style="display: flex; align-items: center; gap: 1rem;">
                                        <Badge severity={t.severity === 'critical' ? 'error' : t.severity === 'high' ? 'error' : t.severity === 'medium' ? 'warning' : 'info'}>
                                            {t.severity}
                                        </Badge>
                                        <div class="rule-info" style="flex: 1; display: flex; flex-direction: column;">
                                            <span class="rule-name" style="font-weight: 700; font-size: 13px;">{t.name}</span>
                                            <span class="rule-pattern" style="font-size: 11px; color: var(--text-muted);">Regex: <code style="color: var(--accent-secondary); background: rgba(0,0,0,0.2); padding: 2px 4px; border-radius: 2px;">{t.pattern}</code></span>
                                        </div>
                                        <span class="rule-cooldown" style="font-size: 10px; color: var(--text-muted); opacity: 0.6;">
                                            {t.id.startsWith('builtin') ? 'SYSTEM' : 'CUSTOM'}
                                        </span>
                                        <Show when={!t.id.startsWith('builtin')}>
                                            <Button variant="danger" size="sm" onClick={() => handleRemoveTrigger(t.id)} style="padding: 4px 8px;">✕</Button>
                                        </Show>
                                    </Card>
                                )}
                            </For>
                        </div>
                    </div>
                </div>
            </Show>

            {/* ===== HISTORY TAB ===== */}
            <Show when={activeTab() === 'history'}>
                <div class="history-panel">
                    <div class="section-header">
                        <h3>📜 Alert History</h3>
                        <button class="ops-btn-sm" onClick={loadAlerts}>🔄 Refresh</button>
                    </div>
                    <div class="history-list">
                        <For each={alertHistory()}>
                            {(event) => (
                                <Card variant="raised" padding="12px" style="display: flex; align-items: center; gap: 1rem; border-left: 3px solid var(--glass-border);">
                                    <div class="history-time" style="font-family: var(--font-mono); font-size: 11px; color: var(--text-muted); min-width: 160px;">
                                        {new Date(event.timestamp).toLocaleString()}
                                    </div>
                                    <Badge severity={event.severity === 'critical' ? 'error' : event.severity === 'high' ? 'error' : event.severity === 'medium' ? 'warning' : 'info'}>
                                        {event.severity}
                                    </Badge>
                                    <div class="history-info" style="flex: 1; display: flex; flex-direction: column;">
                                        <span class="history-rule" style="font-weight: 700; font-size: 13px;">{event.name}</span>
                                        <span class="history-host" style="font-size: 11px; color: var(--text-muted);">Node: {event.host}</span>
                                    </div>
                                    <Badge severity={event.sent ? 'success' : 'neutral'}>
                                        {event.sent ? 'NOTIFIED' : 'LOGGED'}
                                    </Badge>
                                </Card>
                            )}
                        </For>
                        <Show when={alertHistory().length === 0}>
                            <EmptyState
                                icon="🔔"
                                title="No Alerts Yet"
                                description="Alert triggers will log events here when they fire. Configure triggers in the Alerts tab."
                                compact
                            />
                        </Show>
                    </div>
                </div>
            </Show>
        </div>
    );
};
