import { Component, createSignal, createEffect, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { SearchLogs } from '../../../wailsjs/go/services/AnalyticsService';
import { 
    SearchHostEvents, 
    GetHostEvents, 
    AnalyzeEvent, 
    AggregateHostEvents, 
    CreateSavedSearch, 
    GetSavedSearches 
} from '../../../wailsjs/go/services/SIEMService';
import { SaveRunbook } from '../../../wailsjs/go/services/NotesService';
import { 
    SearchBar, 
    Select, 
    Button, 
    Badge, 
    EmptyState, 
    formatTimestamp,
    normalizeSeverity,
    Panel,
    Toolbar,
    ToolbarSpacer,
    showModal,
    ModalOptions
} from '@components/ui';
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
                const modalOpts: ModalOptions = {
                    title: 'AI Forensic Analysis',
                    message: res.text,
                    confirmText: 'Got it',
                    onConfirm: () => {},
                    onCancel: () => {},
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
                };
                showModal(modalOpts);
            }
        } catch (err) {
            console.error("AI analysis failed:", err);
        } finally {
            setAnalyzingId(null);
        }
    };

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

    return (
        <div style="height: 100%; display: flex; flex-direction: column; overflow: hidden; background: var(--surface-0);">
            <Toolbar>
                <div style="display: flex; gap: 8px;">
                    <Select
                        value={selectedHost()}
                        onChange={setSelectedHost}
                        options={[
                            { label: 'GLOBAL_HOSTS', value: 'global' },
                            ...state.hosts.map(h => ({ label: (h.label || h.hostname).toUpperCase(), value: h.id }))
                        ]}
                    />
                    <Select
                        value={searchMode()}
                        onChange={(v: string) => setSearchMode(v as any)}
                        options={[
                            { label: 'TERMINAL_OUTPUT', value: 'terminal' },
                            { label: 'SECURITY_EVENTS', value: 'security' }
                        ]}
                    />
                </div>
                <ToolbarSpacer />
                <div style="display: flex; gap: 8px;">
                    <Select
                        value=""
                        placeholder="SAVED_SEARCHES"
                        onChange={(v: string) => {
                            if (v) { setSearchQuery(v); runSearch(); }
                        }}
                        options={savedSearches().map(ss => ({ label: ss.name.toUpperCase(), value: ss.query }))}
                    />
                    <Button variant="ghost" onClick={handleSaveSearch} title="Save Search">💾</Button>
                </div>
            </Toolbar>

            <div style="padding: var(--gap-md); border-bottom: 1px solid var(--border-primary);">
                <SearchBar
                    value={searchQuery()}
                    onInput={setSearchQuery}
                    onSubmit={runSearch}
                    placeholder={searchMode() === 'terminal' ? "Search output text... (e.g. 'docker')" : "Search anomalies... (IP, User)"}
                    buttonText={isSearching() ? 'SEARCHING...' : 'RUN_QUERY'}
                />
            </div>

            <div style="flex: 1; overflow-y: auto; padding: var(--gap-md);">
                <Show when={searchMode() === 'security' && Object.keys(aggregations()).length > 0}>
                    <div style="display: flex; align-items: center; gap: 8px; margin-bottom: var(--gap-md); overflow-x: auto;">
                        <span style="font-size: 10px; font-weight: 800; color: var(--text-muted); text-transform: uppercase;">Top Events:</span>
                        <For each={Object.entries(aggregations())}>
                            {([type, count]) => (
                                <Badge severity="info">{type} ({count})</Badge>
                            )}
                        </For>
                    </div>
                </Show>

                <Show when={results().length === 0 && !isSearching()}>
                    <div style="padding: 100px 0;">
                        <EmptyState
                            icon="🔍"
                            title="NO_EVENTS_DISCOVERED"
                            description="Modifier search constraints or host filter to expand results."
                            action="CLEAR_FILTERS"
                            onAction={() => {
                                setSearchQuery('');
                                setSelectedHost('global');
                                runSearch();
                            }}
                        />
                    </div>
                </Show>

                <div style="display: flex; flex-direction: column; gap: 8px;">
                    <For each={results()}>
                        {(row) => (
                            <Panel noPadding>
                                <div style="display: flex; flex-direction: column; padding: 12px 16px;">
                                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                                        <div style="display: flex; align-items: center; gap: 12px;">
                                            <span style="font-family: var(--font-mono); font-size: 11px; color: var(--text-muted);">
                                                {formatTimestamp(row.timestamp || row.CreatedAt)}
                                            </span>
                                            <Badge severity="neutral" size="sm">HOST:{row.host_id || row.HostID}</Badge>
                                            <Show when={row.session_id}>
                                                <Badge severity="info" size="sm">SESSION:{row.session_id.substring(0, 8)}</Badge>
                                            </Show>
                                            <Show when={row.EventType || row.Category}>
                                                <Badge severity={normalizeSeverity(row.Severity || 'info')} size="sm">
                                                    {row.EventType || row.Category}
                                                </Badge>
                                            </Show>
                                        </div>
                                        <div style="display: flex; gap: 6px;">
                                            <Button 
                                                variant="ghost" 
                                                size="sm"
                                                onClick={() => handleAnalyze((row.output_clean || row.output || row.RawLog || JSON.stringify(row)), (row.id || row.ID))}
                                                disabled={analyzingId() === (row.id || row.ID)}
                                            >
                                                {analyzingId() === (row.id || row.ID) ? '⏳ ANALYZING...' : '🤖 AI_ANALYZE'}
                                            </Button>
                                            <Show when={row.session_id}>
                                                <Button
                                                    variant="primary"
                                                    size="sm"
                                                    onClick={() => setPlaybackSessionId(row.session_id)}
                                                    style="background: var(--surface-3); border-color: var(--border-primary); color: var(--accent-primary);"
                                                >
                                                    🎬 REPLAY
                                                </Button>
                                            </Show>
                                        </div>
                                    </div>
                                    <div style="font-family: var(--font-mono); font-size: 12px; color: var(--text-primary); white-space: pre-wrap; word-break: break-all; line-height: 1.5; background: rgba(0,0,0,0.15); padding: 8px; border-radius: var(--radius-sm); border: 1px solid var(--border-subtle);">
                                        {row.output_clean || row.output || row.RawLog || JSON.stringify(row)}
                                    </div>
                                </div>
                            </Panel>
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
        </div>
    );
};
