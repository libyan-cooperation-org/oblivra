import { Component, createSignal, createEffect, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { SearchLogs } from '../../../wailsjs/go/app/AnalyticsService';
import { SearchHostEvents, GetHostEvents, AnalyzeEvent, AggregateHostEvents, CreateSavedSearch, GetSavedSearches } from '../../../wailsjs/go/app/SIEMService';
import { SaveRunbook } from '../../../wailsjs/go/app/NotesService';
import { showModal } from '../ui/ModalSystem';
import { EmptyState } from '../ui/EmptyState';
import { SessionPlayback } from './SessionPlayback';

export const TerminalForensics: Component = () => {
    const [state] = useApp();
    const [selectedHost, setSelectedHost] = createSignal<string>('global');
    const [searchQuery, setSearchQuery] = createSignal('');
    const [results, setResults] = createSignal<any[]>([]);
    const [isSearching, setIsSearching] = createSignal(false);
    const [searchMode, setSearchMode] = createSignal<'terminal' | 'security'>('terminal');
    const [analyzingId, setAnalyzingId] = createSignal<string | null>(null);
    const [savedSearches, setSavedSearches] = createSignal<any[]>([]);
    const [aggregations, setAggregations] = createSignal<Record<string, number>>({});
    const [playbackSessionId, setPlaybackSessionId] = createSignal<string | null>(null);

    const handleAnalyze = async (rawLog: string, id: string) => {
        setAnalyzingId(id);
        try {
            const res = await AnalyzeEvent(rawLog);
            if (res && res.text) {
                showModal({
                    title: 'AI Forensic Analysis',
                    message: res.text,
                    confirmText: 'Got it',
                    onConfirm: () => { },
                    onCancel: () => { },
                    actions: [
                        {
                            label: '📕 Save as Playbook',
                            onClick: async () => {
                                try {
                                    await SaveRunbook({
                                        title: `Playbook: Forensic Analysis of ${id}`,
                                        description: `AI generated remediation steps for event ${id}:\n\n${res.text}`,
                                        steps: [],
                                        tags: ['siem', 'incident', 'ai-generated']
                                    } as any);
                                } catch (err) {
                                    console.error("Failed to save playbook:", err);
                                }
                            }
                        },
                        {
                            label: 'Got it',
                            onClick: () => { },
                            type: 'secondary'
                        }
                    ]
                });
            }
        } catch (err) {
            console.error("AI analysis failed:", err);
        } finally {
            setAnalyzingId(null);
        }
    };

    // Run a default 24h search on mount for logs
    createEffect(() => {
        if (state.activeNavTab === 'siem' && results().length === 0) {
            runSearch();
            loadSavedSearches();
        }
    });

    const loadSavedSearches = async () => {
        try {
            const ss = await GetSavedSearches();
            setSavedSearches(ss || []);
        } catch (e) { console.error(e); }
    };

    const runSearch = async () => {
        setIsSearching(true);
        try {
            if (searchMode() === 'terminal') {
                const hostFilter = selectedHost() === 'global' ? '' : ` WHERE host_id = '${selectedHost()}'`;
                const queryText = searchQuery().trim()
                    ? (hostFilter ? `${hostFilter} AND output LIKE '%${searchQuery()}%'` : ` WHERE output LIKE '%${searchQuery()}%'`)
                    : hostFilter;

                const q = `SELECT * FROM terminal_logs${queryText} ORDER BY timestamp DESC LIMIT 200`;
                const res = await SearchLogs(q, "sql", 200, 0);
                setResults(res || []);
            } else {
                if (selectedHost() === 'global') {
                    const res = await SearchHostEvents(searchQuery(), 100);
                    setResults(res || []);
                } else {
                    const res = await GetHostEvents(selectedHost(), 100);
                    setResults(res || []);
                }

                try {
                    const aggs = await AggregateHostEvents(searchQuery(), "event_type");
                    setAggregations(aggs || {});
                } catch (e) {
                    console.error("Aggregation failed:", e);
                    setAggregations({});
                }
            }
        } catch (err) {
            console.error("SIEM search failed:", err);
        } finally {
            setIsSearching(false);
        }
    };

    const handleSaveSearch = async () => {
        if (!searchQuery().trim()) return;
        const name = prompt("Enter a name for this saved search:");
        if (name) {
            await CreateSavedSearch({ name, query: searchQuery() } as any);
            await loadSavedSearches();
        }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Enter') {
            runSearch();
        }
    };

    return (
        <>
            <div class="siem-controls">
                <select
                    class="siem-select"
                    value={selectedHost()}
                    onChange={(e) => setSelectedHost(e.currentTarget.value)}
                >
                    <option value="global">Global (All Hosts)</option>
                    <For each={state.hosts}>
                        {(host) => (
                            <option value={host.id}>{host.label || host.hostname}</option>
                        )}
                    </For>
                </select>

                <select
                    class="siem-select"
                    value={searchMode()}
                    onChange={(e) => setSearchMode(e.currentTarget.value as any)}
                >
                    <option value="terminal">Terminal Output</option>
                    <option value="security">Security Events</option>
                </select>

                <input
                    type="text"
                    class="siem-input"
                    placeholder={searchMode() === 'terminal' ? "Search output text... (e.g. 'docker')" : "Search anomalies... (IP, User)"}
                    value={searchQuery()}
                    onInput={(e) => setSearchQuery(e.currentTarget.value)}
                    onKeyDown={handleKeyDown}
                />

                <select
                    class="siem-select"
                    style="max-width: 150px;"
                    onChange={(e) => {
                        if (e.currentTarget.value) {
                            setSearchQuery(e.currentTarget.value);
                            runSearch();
                        }
                    }}
                >
                    <option value="">Saved Searches</option>
                    <For each={savedSearches()}>
                        {(ss: any) => <option value={ss.query}>{ss.name}</option>}
                    </For>
                </select>

                <button class="btn btn-secondary" style="height: 44px; padding: 0 12px;" onClick={handleSaveSearch} title="Save Search">
                    💾
                </button>

                <button
                    onClick={runSearch}
                    disabled={isSearching()}
                    class="btn btn-primary"
                    style="height: 44px; padding: 0 24px;"
                >
                    {isSearching() ? 'Searching...' : 'Search'}
                </button>
            </div>

            <div class="siem-results-container">
                <Show when={searchMode() === 'security' && Object.keys(aggregations()).length > 0}>
                    <div class="siem-aggregations" style="display: flex; gap: 8px; margin-bottom: 12px; padding: 12px; background: rgba(0,0,0,0.2); border-radius: 6px; overflow-x: auto;">
                        <span style="font-size: 13px; color: var(--text-secondary); align-self: center; white-space: nowrap;">Top Events:</span>
                        <For each={Object.entries(aggregations())}>
                            {([type, count]) => (
                                <span class="siem-tag" style="background: rgba(var(--accent-primary-rgb), 0.1); color: var(--accent-primary); border: none; white-space: nowrap;">
                                    {type} ({count})
                                </span>
                            )}
                        </For>
                    </div>
                </Show>

                <Show when={results().length === 0 && !isSearching()}>
                    <div style="display: flex; flex-direction: column; align-items: center; gap: 16px;">
                        <EmptyState
                            icon="🔍"
                            title="No events found"
                            description="Try modifying your search constraints or host filter."
                        />
                        <button class="btn btn-secondary" onClick={() => {
                            setSearchQuery('');
                            setSelectedHost('global');
                            runSearch();
                        }}>Clear Filters</button>
                    </div>
                </Show>
                <div class="siem-results">
                    <For each={results()}>
                        {(row) => (
                            <div class="siem-event-card">
                                <div class="siem-event-header">
                                    <span class="siem-event-time">
                                        {new Date(row.timestamp || row.CreatedAt).toLocaleString()}
                                    </span>
                                    <span class="siem-host-badge">HOST: {row.host_id || row.HostID}</span>
                                    <Show when={row.session_id}>
                                        <span class="siem-tag">Session: {row.session_id.substring(0, 8)}</span>
                                    </Show>
                                    <Show when={row.EventType || row.Category}>
                                        <span class={`siem-tag severity-${row.Severity || 'info'}`}>{row.EventType || row.Category}</span>
                                    </Show>

                                    <div style="flex-grow: 1;"></div>
                                    <button
                                        class="siem-btn-icon"
                                        onClick={() => handleAnalyze((row.output_clean || row.output || row.RawLog || JSON.stringify(row)), (row.id || row.ID))}
                                        disabled={analyzingId() === (row.id || row.ID)}
                                        title="Send to AI Analyst"
                                    >
                                        {analyzingId() === (row.id || row.ID) ? '⏳' : '🤖 Analyze'}
                                    </button>

                                    <Show when={row.session_id}>
                                        <button
                                            class="siem-btn-icon"
                                            onClick={() => setPlaybackSessionId(row.session_id)}
                                            title="Replay Session"
                                            style="background: rgba(var(--accent-primary-rgb), 0.1); color: var(--accent-primary);"
                                        >
                                            🎬 REPLAY
                                        </button>
                                    </Show>
                                </div>
                                <div class="siem-event-body">
                                    {row.output_clean || row.output || row.RawLog || JSON.stringify(row)}
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>

            <Show when={playbackSessionId()}>
                <SessionPlayback 
                    sessionId={playbackSessionId()!} 
                    onClose={() => setPlaybackSessionId(null)} 
                />
            </Show>
        </>
    );
};
