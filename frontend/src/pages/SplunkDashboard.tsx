import { Component, createSignal, For, Show, onMount } from 'solid-js';
import { Widget } from '../components/dashboard/Widget';
import type { Dashboard, Widget as DashboardWidget } from '../types/dashboard';
import { SaveDashboard, LoadDashboard } from '../../wailsjs/go/app/AnalyticsService';
import { GetSources } from '../../wailsjs/go/app/LogSourceService';
import { Card, Button } from '../components/ui/TacticalComponents';
import '../styles/splunk.css';

// Default dashboard with pre-configured widgets
const defaultDashboard: Dashboard = {
    id: 'main',
    name: 'Infrastructure Overview',
    widgets: [
        {
            id: 'w1', title: 'Total Log Entries', type: 'metric', source: 'analytics',
            query: "SELECT count(*) as count FROM terminal_logs",
            refreshInterval: 10,
            layout: { x: 0, y: 0, w: 1, h: 1 }
        },
        {
            id: 'w2', title: 'Error Events (Last Hour)', type: 'metric', source: 'analytics',
            query: "SELECT count(*) as count FROM terminal_logs WHERE output LIKE '%error%' AND timestamp > datetime('now', '-1 hour')",
            refreshInterval: 10,
            layout: { x: 1, y: 0, w: 1, h: 1 }
        },
        {
            id: 'w3', title: 'Unique Hosts', type: 'metric', source: 'analytics',
            query: "SELECT count(DISTINCT host) as hosts FROM terminal_logs",
            refreshInterval: 30,
            layout: { x: 2, y: 0, w: 1, h: 1 }
        },
        {
            id: 'w4', title: 'Active Sessions', type: 'metric', source: 'analytics',
            query: "SELECT count(DISTINCT session_id) as sessions FROM terminal_logs",
            refreshInterval: 15,
            layout: { x: 3, y: 0, w: 1, h: 1 }
        },
        {
            id: 'w5', title: 'Log Volume by Host', type: 'bar', source: 'analytics',
            query: "SELECT host, count(*) as count FROM terminal_logs GROUP BY host ORDER BY count DESC LIMIT 10",
            refreshInterval: 30,
            layout: { x: 0, y: 1, w: 2, h: 2 }
        },
        {
            id: 'w6', title: 'Recent Errors', type: 'table', source: 'analytics',
            query: "SELECT timestamp, host, output FROM terminal_logs WHERE output LIKE '%error%' ORDER BY timestamp DESC LIMIT 20",
            refreshInterval: 15,
            layout: { x: 2, y: 1, w: 2, h: 2 }
        },
        {
            id: 'w7', title: 'Log Stream (Latest)', type: 'log-stream', source: 'analytics',
            query: "SELECT timestamp, host, output FROM terminal_logs ORDER BY timestamp DESC LIMIT 50",
            refreshInterval: 5,
            layout: { x: 0, y: 3, w: 4, h: 2 }
        },
    ]
};

const widgetTypes = [
    { value: 'metric', label: 'Metric (KPI)' },
    { value: 'bar', label: 'Bar Chart' },
    { value: 'line', label: 'Line Chart' },
    { value: 'pie', label: 'Pie Chart' },
    { value: 'table', label: 'Data Table' },
    { value: 'log-stream', label: 'Log Stream' },
];

export const SplunkDashboard: Component = () => {
    const [dashboard, setDashboard] = createSignal(defaultDashboard);
    const [timeRange, setTimeRange] = createSignal('1h');
    const [showAddWidget, setShowAddWidget] = createSignal(false);
    const [dirty, setDirty] = createSignal(false);

    // Widget form
    const [widgetTitle, setWidgetTitle] = createSignal('');
    const [widgetType, setWidgetType] = createSignal('metric');
    const [widgetSource, setWidgetSource] = createSignal('analytics');
    const [widgetQuery, setWidgetQuery] = createSignal('');
    const [widgetW, setWidgetW] = createSignal(1);
    const [widgetH, setWidgetH] = createSignal(1);

    const [sources, setSources] = createSignal<any[]>([]);

    // Load saved dashboard on mount
    onMount(async () => {
        try { setSources((await GetSources()) || []); } catch { /* ignore */ }
        try {
            const saved = await LoadDashboard('main');
            if (saved) {
                const parsed = JSON.parse(saved);
                setDashboard(parsed);
            }
        } catch { /* Use default */ }
    });

    const saveDashboard = async () => {
        try {
            await SaveDashboard('main', JSON.stringify(dashboard()));
            setDirty(false);
        } catch { /* ignore */ }
    };

    const addWidget = () => {
        if (!widgetTitle() || !widgetQuery()) return;
        const d = dashboard();
        const maxY = d.widgets.reduce((m, w) => Math.max(m, w.layout.y + w.layout.h), 0);
        const newWidget: DashboardWidget = {
            id: 'w-' + Date.now(),
            title: widgetTitle(),
            type: widgetType() as any,
            source: widgetSource(),
            query: widgetQuery(),
            refreshInterval: 15,
            layout: { x: 0, y: maxY, w: widgetW(), h: widgetH() }
        };
        setDashboard({
            ...d,
            widgets: [...d.widgets, newWidget]
        });
        setWidgetTitle('');
        setWidgetQuery('');
        setShowAddWidget(false);
        setDirty(true);
    };

    const removeWidget = (id: string) => {
        const d = dashboard();
        setDashboard({
            ...d,
            widgets: d.widgets.filter(w => w.id !== id)
        });
        setDirty(true);
    };

    return (
        <div class="splunk-ui">
            {/* Toolbar */}
            <div class="dashboard-toolbar">
                <div class="toolbar-left">
                    <h1 class="dashboard-name" style="font-family: var(--font-ui); font-weight: 800; letter-spacing: 1px;">
                        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" />
                        </svg>
                        {dashboard().name.toUpperCase()}
                    </h1>
                </div>
                <div class="toolbar-actions" style="display: flex; gap: 12px; align-items: center;">
                    <Button variant="primary" size="sm" onClick={() => setShowAddWidget(!showAddWidget())}>
                        + ADD WIDGET
                    </Button>
                    <Show when={dirty()}>
                        <Button variant="secondary" size="sm" onClick={saveDashboard} style="background: var(--status-online); border-color: var(--status-online);">
                            💾 SAVE LAYOUT
                        </Button>
                    </Show>
                    <select class="time-picker" value={timeRange()} onChange={e => setTimeRange(e.currentTarget.value)}
                        style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 5px 10px; border-radius: var(--radius-sm); font-family: var(--font-ui); font-size: 12px;">
                        <option value="15m">Last 15m</option>
                        <option value="1h">Last 1h</option>
                        <option value="24h">Last 24h</option>
                        <option value="7d">7 Days</option>
                    </select>
                </div>
            </div>

            {/* Add Widget Form */}
            <Show when={showAddWidget()}>
                <Card variant="raised" padding="16px" style="margin: 0 16px 20px;">
                    <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center;">
                        <input class="ops-input" placeholder="Widget title" value={widgetTitle()} onInput={e => setWidgetTitle(e.currentTarget.value)} style="flex: 1;" />
                        <select class="ops-select" value={widgetType()} onChange={e => setWidgetType(e.currentTarget.value)}>
                            <For each={widgetTypes}>{(wt) => <option value={wt.value}>{wt.label}</option>}</For>
                        </select>
                        <select class="ops-select" value={widgetSource()} onChange={e => setWidgetSource(e.currentTarget.value)}>
                            <option value="analytics">Local (Analytics SQL)</option>
                            <option value="osquery">Local (SSH osquery)</option>
                            <For each={sources()}>{(src) => <option value={src.id}>{src.name} ({src.type})</option>}</For>
                        </select>
                        <input class="ops-input" placeholder="Query: SELECT * FROM..." value={widgetQuery()} onInput={e => setWidgetQuery(e.currentTarget.value)} style="flex: 2;" />
                        <div style="display: flex; gap: 4px; align-items: center;">
                            <span style="font-size: 10px; color: var(--text-muted);">SIZE:</span>
                            <input class="ops-input" type="number" min="1" max="4" value={widgetW()} onInput={e => setWidgetW(parseInt(e.currentTarget.value) || 1)} style="width: 45px; padding: 4px;" />
                            <span style="font-size: 10px; color: var(--text-muted);">×</span>
                            <input class="ops-input" type="number" min="1" max="4" value={widgetH()} onInput={e => setWidgetH(parseInt(e.currentTarget.value) || 1)} style="width: 45px; padding: 4px;" />
                        </div>
                        <Button variant="primary" size="sm" onClick={addWidget}>ADD_WIDGET</Button>
                        <Button variant="ghost" size="sm" onClick={() => setShowAddWidget(false)}>CANCEL</Button>
                    </div>
                </Card>
            </Show>

            {/* The Grid */}
            <div class="dashboard-grid">
                <For each={dashboard().widgets}>
                    {(widget) => (
                        <Card
                            variant="raised"
                            padding="0"
                            style={{
                                'grid-column': `span ${widget.layout.w}`,
                                'grid-row': `span ${widget.layout.h}`,
                                position: 'relative',
                                overflow: 'hidden'
                            }}
                        >
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => removeWidget(widget.id)}
                                style="position: absolute; top: 8px; right: 8px; z-index: 10; padding: 4px; width: 24px; height: 24px; border: none; opacity: 0.5;"
                                title="Remove widget"
                            >✕</Button>
                            <Widget config={widget} timeRange={timeRange()} />
                        </Card>
                    )}
                </For>
            </div>
        </div>
    );
};
