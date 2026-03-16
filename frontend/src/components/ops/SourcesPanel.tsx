import { Component, createSignal, createMemo, For, Show, onMount, onCleanup } from 'solid-js';
import { EmptyState } from '../ui/EmptyState';
import type { LogSource, LogResult, ConnectionResult } from '../../types/ops';
import { SEVERITY_ORDER, SEVERITY_COLORS } from '../../types/ops';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import {
    GetSources, AddSource, RemoveSource, QuerySource, SearchAllSources, GetHealthStatus,
    SaveSearch, GetSavedSearches, DeleteSavedSearch,
    ExportCSV, ExportJSON,
    TestConnection,
    StartLokiStream, StopLokiStream
} from '../../../wailsjs/go/services/LogSourceService';

const emptySource: LogSource = {
    id: '', name: '', type: 'elasticsearch', url: '', enabled: true,
    api_key: '', username: '', password: '', index: '', org_id: '', tls_skip_verify: false, tags: []
};

const PrettyJson: Component<{ data: string }> = (props) => {
    const coloredJson = createMemo(() => {
        try {
            const parsed = JSON.parse(props.data);
            const formatted = JSON.stringify(parsed, null, 2);
            return formatted.replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, function (match) {
                let cls = 'color: #79c0ff;';
                if (/^"/.test(match)) {
                    if (/:$/.test(match)) {
                        cls = 'color: #ff7b72; font-weight: bold;';
                    } else {
                        cls = 'color: #a5d6ff;';
                    }
                } else if (/true|false/.test(match)) {
                    cls = 'color: #d2a8ff;';
                } else if (/null/.test(match)) {
                    cls = 'color: #ff7b72;';
                } else {
                    cls = 'color: #79c0ff;';
                }
                return '<span style="' + cls + '">' + match + '</span>';
            });
        } catch {
            return null;
        }
    });

    return (
        <Show when={coloredJson()} fallback={<>{props.data}</>}>
            <div innerHTML={coloredJson()!} style="margin: 0;" />
        </Show>
    );
};

const VirtualLogList: Component<{ logs: LogResult[], onRowClick: (log: LogResult) => void, severityColor: (level: string) => string, loading: boolean }> = (props) => {
    const [scrollTop, setScrollTop] = createSignal(0);
    const itemHeight = 32;

    const visibleRange = createMemo(() => {
        const start = Math.max(0, Math.floor(scrollTop() / itemHeight) - 5);
        const count = Math.ceil(600 / itemHeight) + 10;
        return { start, end: Math.min(props.logs.length, start + count) };
    });

    const visibleLogs = createMemo(() => {
        const { start, end } = visibleRange();
        return props.logs.slice(start, end).map((log, i) => ({ log, index: start + i }));
    });

    return (
        <div style={{ height: '600px', 'overflow-y': 'auto', 'min-height': '200px', 'position': 'relative', 'border-radius': '6px', 'background': 'rgba(0,0,0,0.2)', 'border': '1px solid var(--border-color)' }} onScroll={e => setScrollTop(e.currentTarget.scrollTop)}>
            <div style={{ height: `${props.logs.length * itemHeight} px`, position: 'relative' }}>
                <For each={visibleLogs()}>
                    {({ log, index }) => (
                        <div class="source-log-row" onClick={() => props.onRowClick(log)}
                            style={{ position: 'absolute', top: `${index * itemHeight} px`, left: 0, right: 0, height: `${itemHeight} px`, cursor: 'pointer', 'align-items': 'center', 'display': 'grid', 'box-sizing': 'border-box' }}>
                            <span class="source-log-ts" style={{ overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{log.timestamp ? new Date(log.timestamp).toLocaleString() : '—'}</span>
                            <span class="source-log-level" style={{ color: props.severityColor(log.level), 'border-color': props.severityColor(log.level) }}>
                                {(log.level || 'unknown').toUpperCase()}
                            </span>
                            <span class="source-log-host" style={{ overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{log.host || '—'}</span>
                            <span class="source-log-src" style={{ overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{log.source || '—'}</span>
                            <span class="source-log-msg" style={{ overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{log.message || '—'}</span>
                        </div>
                    )}
                </For>
            </div>
            <Show when={props.logs.length === 0 && !props.loading}>
                <EmptyState icon="📊" title="No Results" description="Run a query to search logs from this source." compact />
            </Show>
        </div>
    );
};

export const SourcesPanel: Component = () => {
    const [sources, setSources] = createSignal<LogSource[]>([]);
    const [sourceResults, setSourceResults] = createSignal<LogResult[]>([]);
    const [sourceQuery, setSourceQuery] = createSignal('');
    const [activeSource, setActiveSource] = createSignal('');
    const [sourceFilter, setSourceFilter] = createSignal('');
    const [sortBySeverity, setSortBySeverity] = createSignal(true);
    const [sourceTestMsg, setSourceTestMsg] = createSignal('');
    const [newSource, setNewSource] = createSignal<LogSource>({ ...emptySource });
    const [loading, setLoading] = createSignal(false);
    const [saving, setSaving] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);
    const [isTailing, setIsTailing] = createSignal(false);    // Health monitoring
    type HealthStatus = { source_id: string, healthy: boolean, message: string };
    const [health, setHealth] = createSignal<HealthStatus[]>([]);
    const [offset, setOffset] = createSignal(0);
    // Saved searches
    type SavedSearchType = { id: string, name: string, query: string, source_id: string };
    const [savedSearches, setSavedSearches] = createSignal<SavedSearchType[]>([]);
    const [saveName, setSaveName] = createSignal('');

    // Log correlation
    const [correlationLog, setCorrelationLog] = createSignal<LogResult | null>(null);

    // Multi-source mode
    const [multiSourceMode, setMultiSourceMode] = createSignal(false);

    // Live Tailing
    // Refs
    let searchInputRef: HTMLInputElement | undefined;

    const loadSources = async () => {
        try { setSources((await GetSources()) || []); } catch { /* ignore */ }
    };

    const loadHealth = async () => {
        try { setHealth((await GetHealthStatus()) || []); } catch { /* ignore */ }
    };

    const loadSavedSearches = async () => {
        try { setSavedSearches((await GetSavedSearches()) || []); } catch { /* ignore */ }
    };

    // Load on mount and add keyboard shortcuts
    onMount(() => {
        loadSources();
        loadHealth();
        loadSavedSearches();
        const iv = setInterval(loadHealth, 60000);

        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') {
                setCorrelationLog(null);
            } else if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                handleQuerySource();
            } else if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                searchInputRef?.focus();
            }
        };
        window.addEventListener('keydown', handleKeyDown);

        onCleanup(() => {
            clearInterval(iv);
            window.removeEventListener('keydown', handleKeyDown);
        });
    });

    const getHealthForSource = (id: string) => {
        return health().find(h => h.source_id === id);
    };

    const handleAddSource = async () => {
        const s = newSource();
        if (!s.name || !s.url) { setError('Name and URL are required'); return; }
        setSaving(true);
        setError(null);
        try {
            s.id = 'src-' + Date.now();
            await AddSource(s as any);
            setNewSource({ ...emptySource });
            await loadSources();
        } catch (e) { setError((e as Error).message || String(e)); }
        finally { setSaving(false); }
    };

    const handleTestSource = async (id: string) => {
        setSourceTestMsg('⏳ Testing...');
        try {
            const result = await TestConnection(id) as ConnectionResult;
            setSourceTestMsg(`${result.ok ? '✅' : '❌'} ${result.message} `);
            await loadHealth();
        } catch (e) { setSourceTestMsg('❌ ' + ((e as Error).message || String(e))); }
    };

    const handleRemoveSource = async (id: string) => {
        try {
            await RemoveSource(id);
            if (activeSource() === id) setActiveSource('');
            await loadSources();
        } catch (e) { setError((e as Error).message || String(e)); }
    };

    const toggleTail = async () => {
        if (!activeSource() || !sourceQuery()) return;
        const src = sources().find(s => s.id === activeSource());
        if (src?.type !== 'loki') return;

        const eventName = `loki - stream - ${activeSource()} `;
        const errorEventName = `loki - stream - error - ${activeSource()} `;

        if (isTailing()) {
            await StopLokiStream(activeSource());
            EventsOff(eventName);
            EventsOff(errorEventName);
            setIsTailing(false);
        } else {
            setSourceResults([]);
            setIsTailing(true);

            // Register event listener for the stream
            EventsOn(eventName, (log: LogResult) => {
                setSourceResults(prev => {
                    const combined = [log, ...prev];
                    return combined.slice(0, 1000); // keep last 1000 in memory
                });

                // Desktop Notification for critical logs
                const level = (log.level || '').toLowerCase();
                if (['error', 'critical', 'fatal'].includes(level)) {
                    if (window.Notification && Notification.permission === 'granted') {
                        new Notification(`🚨 ${level.toUpperCase()} Log Detected`, {
                            body: log.message ? (log.message.length > 100 ? log.message.substring(0, 100) + '...' : log.message) : '',
                        });
                    } else if (window.Notification && Notification.permission !== 'denied') {
                        Notification.requestPermission();
                    }
                }
            });

            EventsOn(errorEventName, (errStr: string) => {
                setError(errStr);
                setIsTailing(false);
                EventsOff(eventName);
                EventsOff(errorEventName);
            });

            try {
                await StartLokiStream(activeSource(), sourceQuery());
            } catch (e) {
                setError((e as Error).message || String(e));
                setIsTailing(false);
                EventsOff(eventName);
                EventsOff(errorEventName);
            }
        }
    };

    const handleQuerySource = async (append = false) => {
        if (!sourceQuery()) return;
        if (isTailing()) {
            await StopLokiStream(activeSource());
            EventsOff(`loki - stream - ${activeSource()} `);
            EventsOff(`loki - stream - error - ${activeSource()} `);
            setIsTailing(false);
        }

        if (!append) {
            setOffset(0);
            setSourceResults([]);
        }

        setLoading(true);
        setError(null);
        try {
            const currentOffset = append ? offset() + 300 : 0;
            if (append) setOffset(currentOffset);

            let res: LogResult[];
            // Currently ignoring timeRange in the global panel for simplicity, defaulting to 24h
            if (multiSourceMode()) {
                res = (await SearchAllSources(sourceQuery(), '24h', 300, currentOffset)) as LogResult[] || [];
            } else {
                if (!activeSource()) return;
                res = (await QuerySource(activeSource(), sourceQuery(), '24h', 300, currentOffset)) as LogResult[] || [];
            }

            setSourceResults(prev => append ? [...prev, ...res] : res);
        } catch (e) { setError((e as Error).message || String(e)); }
        finally { setLoading(false); }
    };

    const handleSaveSearch = async () => {
        if (!saveName() || !sourceQuery()) return;
        try {
            await SaveSearch(activeSource() || 'all', saveName(), sourceQuery());
            setSaveName('');
            await loadSavedSearches();
        } catch (e) { setError((e as Error).message || String(e)); }
    };

    const handleDeleteSavedSearch = async (id: string) => {
        try {
            await DeleteSavedSearch(id);
            await loadSavedSearches();
        } catch { /* ignore */ }
    };

    const handleExport = async (format: 'csv' | 'json') => {
        if (!activeSource() || !sourceQuery()) return;
        try {
            const data = format === 'csv'
                ? await ExportCSV(activeSource(), sourceQuery(), '24h', 500)
                : await ExportJSON(activeSource(), sourceQuery(), '24h', 500);
            // Download via blob
            const blob = new Blob([data], { type: format === 'csv' ? 'text/csv' : 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `logs_export_${Date.now()}.${format} `;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) { setError((e as Error).message || String(e)); }
    };

    const filteredResults = createMemo(() => {
        let r = sourceResults();
        const f = sourceFilter().toLowerCase();
        if (f) r = r.filter(l => (l.message || '').toLowerCase().includes(f) || (l.host || '').toLowerCase().includes(f) || (l.level || '').toLowerCase().includes(f));
        if (sortBySeverity()) {
            r = [...r].sort((a, b) => (SEVERITY_ORDER[a.level?.toLowerCase()] ?? 99) - (SEVERITY_ORDER[b.level?.toLowerCase()] ?? 99));
        }
        return r;
    });

    // Theme-aware severity color (CSS custom properties fallback)
    const severityColor = (level: string): string => {
        return SEVERITY_COLORS[level?.toLowerCase()] || 'var(--text-secondary, #8b949e)';
    };

    const highlightQuery = (query: string) => {
        // Simple regex-based syntax highlighter for SQL and LogQL
        let highlighted = query
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/\b(SELECT|FROM|WHERE|AND|OR|GROUP BY|ORDER BY|LIMIT|count|DISTINCT|LIKE)\b/gi, '<span style="color: #ff7b72; font-weight: bold;">$&</span>')
            .replace(/({|})/g, '<span style="color: #79c0ff; font-weight: bold;">$&</span>')
            .replace(/(=|!=|=~|!~|\|=)/g, '<span style="color: #d2a8ff;">$&</span>')
            .replace(/("[^"]*")/g, '<span style="color: #a5d6ff;">$&</span>');
        return highlighted;
    };

    return (
        <div class="sources-panel">
            <Show when={error()}><div class="ops-error">{error()}</div></Show>

            {/* Add Source Form */}
            <div class="alerts-section">
                <div class="section-header"><h3>🌐 External Log Sources</h3></div>
                <p class="config-hint">Connect to Elasticsearch, Grafana Loki, or Splunk. Credentials are obfuscated and persist across restarts.</p>

                <div class="source-form">
                    <div class="form-grid">
                        <input class="ops-input" placeholder="Name (e.g. Prod ES)" value={newSource().name}
                            onInput={e => setNewSource({ ...newSource(), name: e.currentTarget.value })} />
                        <select class="ops-select" value={newSource().type}
                            onChange={e => setNewSource({ ...newSource(), type: e.currentTarget.value as any })}>
                            <option value="elasticsearch">Elasticsearch</option>
                            <option value="loki">Grafana Loki</option>
                            <option value="splunk">Splunk</option>
                        </select>
                        <input class="ops-input" placeholder="URL (https://es.example.com:9200)" value={newSource().url}
                            onInput={e => setNewSource({ ...newSource(), url: e.currentTarget.value })} />
                        <input class="ops-input" placeholder="Index pattern (e.g. logs-*)" value={newSource().index}
                            onInput={e => setNewSource({ ...newSource(), index: e.currentTarget.value })} />
                        <input class="ops-input" placeholder="API Key / Bearer Token" type="password" value={newSource().api_key}
                            onInput={e => setNewSource({ ...newSource(), api_key: e.currentTarget.value })} />
                        <input class="ops-input" placeholder="Username (or leave blank)" value={newSource().username}
                            onInput={e => setNewSource({ ...newSource(), username: e.currentTarget.value })} />
                        <input class="ops-input" placeholder="Password" type="password" value={newSource().password}
                            onInput={e => setNewSource({ ...newSource(), password: e.currentTarget.value })} />
                        <Show when={newSource().type === 'loki'}>
                            <input class="ops-input" placeholder="Org ID (X-Scope-OrgID)" value={newSource().org_id}
                                onInput={e => setNewSource({ ...newSource(), org_id: e.currentTarget.value })} />
                        </Show>
                        <input class="ops-input" placeholder="Tags (comma-separated, e.g. prod, frontend)" value={newSource().tags?.join(', ') || ''}
                            onInput={e => setNewSource({ ...newSource(), tags: e.currentTarget.value.split(',').map(s => s.trim()).filter(Boolean) })} title="Source Tags" />
                        <label class="toggle-label">
                            <input type="checkbox" checked={newSource().tls_skip_verify}
                                onChange={e => setNewSource({ ...newSource(), tls_skip_verify: e.currentTarget.checked })} />
                            Skip TLS verify (self-signed certs)
                        </label>
                    </div>
                    <button class="ops-btn" onClick={handleAddSource} disabled={saving()}>
                        {saving() ? '⏳ Adding...' : '+ Add Source'}
                    </button>
                </div>
            </div>

            {/* Existing Sources with Health Badges */}
            <div class="alerts-section">
                <div class="section-header"><h3>⚡ Configured Sources</h3></div>
                <Show when={sourceTestMsg()}><div class="source-test-msg">{sourceTestMsg()}</div></Show>
                <div class="rules-list">
                    <For each={sources()}>
                        {(s) => {
                            const h = () => getHealthForSource(s.id);
                            return (
                                <div class="rule-card">
                                    <div class="rule-severity" style={{ position: 'relative' }}>
                                        {s.type === 'elasticsearch' ? '🔍' : s.type === 'loki' ? '📊' : '⚡'}
                                        <span style={{
                                            position: 'absolute', bottom: '-2px', right: '-2px',
                                            width: '10px', height: '10px', 'border-radius': '50%',
                                            background: h()?.healthy ? '#3fb950' : h() ? '#f85149' : '#484f58',
                                            border: '2px solid var(--bg-primary, #0d1117)'
                                        }} title={h()?.message || 'Unknown'} />
                                    </div>
                                    <div class="rule-info">
                                        <span class="rule-name">{s.name}</span>
                                        <span class="rule-pattern">{s.type} — <code>{s.url}</code>{s.tls_skip_verify ? ' 🔓' : ''}</span>
                                        <Show when={s.tags && s.tags.length > 0}>
                                            <div style={{ 'margin-top': '4px', display: 'flex', gap: '4px' }}>
                                                <For each={s.tags}>{tag => <span style={{ background: 'rgba(255,255,255,0.1)', padding: '2px 6px', 'border-radius': '4px', 'font-size': '10px' }}>{tag}</span>}</For>
                                            </div>
                                        </Show>
                                    </div>
                                    <button class="ops-btn-sm" onClick={() => handleTestSource(s.id)}>🧪 Test</button>
                                    <button class="ops-btn-sm" onClick={() => { setActiveSource(s.id); setMultiSourceMode(false); setSourceQuery(s.type === 'loki' ? '{job="varlogs"}' : s.type === 'splunk' ? 'index=main' : '*'); }}>📝 Query</button>
                                    <button class="ops-btn-danger" onClick={() => handleRemoveSource(s.id)}>✕</button>
                                </div>
                            );
                        }}
                    </For>
                    <Show when={sources().length === 0}>
                        <EmptyState icon="🌐" title="No Sources Configured" description="Connect to Elasticsearch, Grafana Loki, or Splunk to start querying external logs." compact />
                    </Show>
                </div>
            </div>

            {/* Saved Searches */}
            <Show when={savedSearches().length > 0}>
                <div class="alerts-section">
                    <div class="section-header"><h3>💾 Saved Searches</h3></div>
                    <div class="rules-list">
                        <For each={savedSearches()}>
                            {(ss) => (
                                <div class="rule-card">
                                    <div class="rule-info">
                                        <span class="rule-name">{ss.name}</span>
                                        <span class="rule-pattern"><code>{ss.query}</code></span>
                                    </div>
                                    <button class="ops-btn-sm" onClick={() => { setActiveSource(ss.source_id); setSourceQuery(ss.query); setMultiSourceMode(ss.source_id === 'all'); }}>▶</button>
                                    <button class="ops-btn-danger" onClick={() => handleDeleteSavedSearch(ss.id)}>✕</button>
                                </div>
                            )}
                        </For>
                    </div>
                </div>
            </Show>

            {/* Multi-Source Search */}
            <div class="alerts-section">
                <div class="section-header"><h3>🔎 {multiSourceMode() ? 'Multi-Source Search (All)' : activeSource() ? `Query: ${sources().find(s => s.id === activeSource())?.name || activeSource()} ` : 'Select a source or use Multi-Source'}</h3></div>

                <div class="ops-search-bar" style={{ gap: '8px', 'align-items': 'center' }}>
                    <label class="toggle-label" style={{ 'white-space': 'nowrap' }}>
                        <input type="checkbox" checked={multiSourceMode()} onChange={e => setMultiSourceMode(e.currentTarget.checked)} />
                        Search All
                    </label>
                    <div style={{ position: 'relative', flex: '1', display: 'flex' }}>
                        <div class="query-highlighter" innerHTML={highlightQuery(sourceQuery())}
                            style={{ position: 'absolute', top: '0', left: '0', right: '0', bottom: '0', padding: '6px 12px', 'pointer-events': 'none', 'white-space': 'pre', 'overflow': 'hidden', 'text-wrap': 'nowrap', 'font-family': 'monospace', color: 'transparent', "z-index": 1 }} />
                        <input ref={searchInputRef} class="ops-input query-input" placeholder="Enter query... (Ctrl+K to focus, Ctrl+Enter to run)" value={sourceQuery()}
                            onInput={e => setSourceQuery(e.currentTarget.value)}
                            style={{ position: 'relative', "z-index": 2, background: 'transparent', 'font-family': 'monospace', color: 'var(--text-primary)', width: '100%' }} />
                    </div>

                    <Show when={sources().find(s => s.id === activeSource())?.type === 'loki' && !multiSourceMode()}>
                        <button class="ops-btn-sm" onClick={toggleTail} style={{ background: isTailing() ? '#ff2d55' : 'var(--accent)', color: '#fff' }}>
                            {isTailing() ? '⏹ Stop Tail' : '🌊 Live Tail'}
                        </button>
                    </Show>

                    <button class="ops-btn run-btn" onClick={() => handleQuerySource(false)} disabled={loading() || isTailing()}>
                        {loading() ? '⏳ Querying...' : '▶ Run'}
                    </button>
                </div>

                {/* Save search + Export */}
                <div class="source-results-toolbar" style={{ 'margin-top': '8px' }}>
                    <input class="ops-input filter-input" placeholder="🔍 Filter results..." value={sourceFilter()}
                        onInput={e => setSourceFilter(e.currentTarget.value)} />
                    <label class="toggle-label">
                        <input type="checkbox" checked={sortBySeverity()} onChange={e => setSortBySeverity(e.currentTarget.checked)} />
                        Sort by severity
                    </label>
                    <span class="result-count">{filteredResults().length} results</span>

                    {/* Save search */}
                    <div style={{ display: 'flex', gap: '4px', 'margin-left': 'auto' }}>
                        <input class="ops-input" placeholder="Save as..." value={saveName()} onInput={e => setSaveName(e.currentTarget.value)}
                            style={{ width: '120px', 'font-size': '12px' }} />
                        <button class="ops-btn-sm" onClick={handleSaveSearch} disabled={!saveName() || !sourceQuery()}>💾</button>
                    </div>

                    {/* Export */}
                    <button class="ops-btn-sm" onClick={() => handleExport('csv')} disabled={filteredResults().length === 0}>📄 CSV</button>
                    <button class="ops-btn-sm" onClick={() => handleExport('json')} disabled={filteredResults().length === 0}>📋 JSON</button>
                </div>

                {/* Results with severity colors + correlation */}
                <div class="source-results-list" style={{ padding: 0, background: 'transparent', border: 'none' }}>
                    <VirtualLogList
                        logs={filteredResults()}
                        onRowClick={(log) => setCorrelationLog(correlationLog() === log ? null : log)}
                        severityColor={severityColor}
                        loading={loading()}
                    />
                </div>
            </div>

            {/* Log Correlation Detail */}
            <Show when={correlationLog()}>
                <div class="alerts-section" style={{ 'border-left': `3px solid ${severityColor(correlationLog()?.level || '')} ` }}>
                    <div class="section-header"><h3>🔗 Log Correlation — {correlationLog()?.host || 'Unknown'}</h3></div>
                    <div style={{ padding: '12px', display: 'flex', 'flex-direction': 'column', gap: '8px' }}>
                        <div style={{ display: 'grid', 'grid-template-columns': 'auto 1fr', gap: '4px 12px', 'font-size': '13px' }}>
                            <span style={{ color: '#8b949e' }}>Timestamp:</span><span>{correlationLog()?.timestamp}</span>
                            <span style={{ color: '#8b949e' }}>Host:</span><span>{correlationLog()?.host}</span>
                            <span style={{ color: '#8b949e' }}>Level:</span><span style={{ color: severityColor(correlationLog()?.level || '') }}>{correlationLog()?.level?.toUpperCase()}</span>
                            <span style={{ color: '#8b949e' }}>Source:</span><span>{correlationLog()?.source}</span>
                        </div>
                        <div style={{ background: 'rgba(0,0,0,0.3)', padding: '10px', 'border-radius': '6px', 'font-family': 'monospace', 'font-size': '12px', 'word-break': 'break-all', 'white-space': 'pre-wrap', color: 'var(--text-primary, #e6edf3)' }}>
                            <PrettyJson data={correlationLog()?.message || ''} />
                        </div>
                        <Show when={correlationLog()?.fields && Object.keys(correlationLog()!.fields!).length > 0}>
                            <div style={{ 'font-size': '12px', color: '#8b949e' }}>
                                <strong>Fields:</strong>
                                <For each={Object.entries(correlationLog()!.fields!)}>
                                    {([k, v]) => <span style={{ 'margin-left': '8px' }}><code>{k}</code>={v}</span>}
                                </For>
                            </div>
                        </Show>
                        <button class="ops-btn-sm" onClick={() => setCorrelationLog(null)} style={{ 'align-self': 'flex-end' }}>Close</button>
                    </div>
                </div>
            </Show>
        </div>
    );
};
