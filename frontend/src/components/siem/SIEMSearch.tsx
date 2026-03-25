import { Component, createSignal, createResource, createMemo, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';
import { 
    PageLayout, 
    SearchBar, 
    Select, 
    Badge, 
    Histogram, 
    SectionHeader, 
    LoadingState, 
    EmptyState,
    formatTimestamp 
} from '@components/ui';
import '../../styles/siem-search.css';

const SEVERITY_MAP: Record<string, string> = {
  failed_login:     'critical',
  security_alert:   'high',
  sudo_exec:        'medium',
  successful_login: 'low',
};

function getSev(eventType: string) {
  return SEVERITY_MAP[eventType] || 'info';
}

export const SIEMSearch: Component = () => {
    const [query, setQuery] = createSignal('');
    const [limit, setLimit] = createSignal(100);
    const [trigger, setTrigger] = createSignal(0);
    const [lastQuery, setLastQuery] = createSignal('');
    const [expandedRows, setExpandedRows] = createSignal<Set<number>>(new Set());

    const [events] = createResource(
        () => ({ q: lastQuery(), l: limit(), t: trigger() }),
        async ({ q, l }) => {
            setExpandedRows(new Set<number>());
            if (!q.trim()) return [];
            try {
                const r = await (SIEMService as any).SearchHostEvents(q, l);
                return r || [];
            } catch (e) {
                console.error("SIEM Search Error:", e);
                return [];
            }
        }
    );

    const runSearch = () => {
        setLastQuery(query());
        setTrigger(t => t + 1);
    }

    const toggleRow = (id: number) => {
        const next = new Set(expandedRows());
        if (next.has(id)) next.delete(id);
        else next.add(id);
        setExpandedRows(next);
    }

    const appendFilter = (field: string, val: string) => {
        const current = query().trim();
        const filter = `${field}:"${val}"`;
        const newQuery = current ? `${current} AND ${filter}` : filter;
        setQuery(newQuery);
        runSearch();
    }

    const evts = () => events() ?? [];
    const getField = (e: any, key: string, upperKey: string) => e[key] !== undefined ? e[key] : e[upperKey];

    const histogramData = createMemo(() => {
        const es = evts();
        if (es.length === 0) return [];
        
        const times = es.map((e: any) => {
           const ts = getField(e, 'timestamp', 'Timestamp');
           return ts ? new Date(ts).getTime() : 0;
        }).filter((t: number) => t > 0);
        
        if (times.length === 0) return [];

        const minT = Math.min(...times);
        const maxT = Math.max(...times);
        const binCount = 40;
        const bins = Array.from({ length: binCount }, () => 0);
        const timeSpan = maxT - minT || 1; 
        
        es.forEach((e: any) => {
            const ts = getField(e, 'timestamp', 'Timestamp');
            if (!ts) return;
            const t = new Date(ts).getTime();
            let idx = Math.floor(((t - minT) / timeSpan) * binCount);
            if (idx >= binCount) idx = binCount - 1;
            bins[idx]++;
        });
        
        const maxCount = Math.max(...bins, 1);
        
        return bins.map((count, i) => {
           const binTime = new Date(minT + (i / binCount) * timeSpan);
           return {
             count,
             heightPct: (count / maxCount) * 100,
             timeLabel: binTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
           };
        });
    });

    const fields = createMemo(() => {
        const es = evts();
        const fieldCounts: Record<string, Record<string, number>> = {};
        const track = [
            { key: 'host_id', l: 'HOST' },
            { key: 'event_type', l: 'TYPE' },
            { key: 'source_ip', l: 'SRC_IP' },
            { key: 'user', l: 'USER' }
        ];

        es.forEach((e: any) => {
            track.forEach(f => {
                const val = String(getField(e, f.key, '') || '');
                if (val && val !== 'undefined') {
                    if (!fieldCounts[f.l]) fieldCounts[f.l] = {};
                    fieldCounts[f.l][val] = (fieldCounts[f.l][val] || 0) + 1;
                }
            });
        });
        
        return Object.entries(fieldCounts).map(([label, valMap]) => {
            const vals = Object.entries(valMap).sort((a,b) => b[1]-a[1]).slice(0, 5);
            return { label, vals };
        });
    });

    return (
        <PageLayout 
            title="Sovereign SIEM" 
            subtitle="GLOBAL_EVENT_EXPLORER"
            actions={
                <Show when={lastQuery()}>
                   <Badge severity="info">{evts().length} MATCHES</Badge> 
                </Show>
            }
        >
            <div style="margin-bottom: var(--gap-lg);">
                <SearchBar
                    value={query()}
                    onInput={setQuery}
                    onSubmit={runSearch}
                    placeholder="Search logs: host_id:node01 OR user:admin | stats..."
                    buttonText={events.loading ? 'SEARCHING...' : 'SEARCH_RECORDS'}
                    suffix={
                        <Select 
                            variant="ghost" 
                            value={limit()} 
                            onChange={(v: string) => setLimit(Number(v))}
                            options={[50, 100, 250, 500, 1000]}
                        />
                    }
                />
            </div>

            <div style="display: flex; gap: var(--gap-lg); height: 100%; overflow: hidden;">
                {/* Interesting Fields Sidebar */}
                <aside class="siem-sidebar">
                    <SectionHeader>INTERESTING_FIELDS</SectionHeader>
                    <Show when={evts().length === 0 && !events.loading}>
                        <div style="font-size: 11px; color: var(--text-muted); font-style: italic; padding: 12px;">No fields extracted.</div>
                    </Show>
                    <For each={fields()}>
                        {({ label, vals }) => (
                            <div class="siem-field-group">
                                <div class="siem-field-name">{label}</div>
                                <For each={vals}>
                                    {([val, count]) => (
                                        <div class="siem-field-val" onClick={() => appendFilter(label.toLowerCase(), val)}>
                                            <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 140px;">{val}</span>
                                            <span class="siem-field-count">{count}</span>
                                        </div>
                                    )}
                                </For>
                            </div>
                        )}
                    </For>
                </aside>

                <main style="flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden;">
                    {/* Histogram */}
                    <Histogram data={histogramData()} height={80} class="siem-histogram-wrap" />

                    {/* Results Table */}
                    <div style="flex: 1; overflow: auto; border: 1px solid var(--border-primary); border-radius: var(--radius-sm); margin-top: var(--gap-md);">
                        <table class="ob-table striped" style="width: 100%; border-collapse: collapse;">
                            <thead>
                                <tr>
                                    <th style="width: 32px; padding-left: 12px;"></th>
                                    <th style="width: 160px; text-align: left;">TIME</th>
                                    <th style="width: 120px; text-align: left;">HOST</th>
                                    <th style="width: 140px; text-align: left;">TYPE</th>
                                    <th style="text-align: left;">RAW_LOG</th>
                                </tr>
                            </thead>
                            <tbody>
                                <Show when={!events.loading && evts().length === 0}>
                                    <tr>
                                        <td colspan="5" style="padding: 100px 0;">
                                            <EmptyState 
                                                icon="🔎"
                                                title={lastQuery() ? "NO_EVENTS_MATCH_SPECIFIED_QUERY" : "READY_FOR_QUERY_EXECUTION"} 
                                                description="Enter a query string above to begin forensic review."
                                            />
                                        </td>
                                    </tr>
                                </Show>

                                <For each={evts()}>
                                    {(evt: any) => {
                                        const id = getField(evt, 'id', 'ID');
                                        const expanded = expandedRows().has(id);
                                        const type = getField(evt, 'event_type', 'EventType');
                                        const raw = getField(evt, 'raw_log', 'RawLog');
                                        
                                        return (
                                            <>
                                                <tr 
                                                    onClick={() => toggleRow(id)} 
                                                    style={{ 
                                                        cursor: 'pointer',
                                                        background: expanded ? 'var(--surface-1)' : '' 
                                                    }}
                                                >
                                                    <td style="text-align: center; font-size: 8px; color: var(--accent-primary); padding-left: 12px;">{expanded ? '▼' : '▶'}</td>
                                                    <td class="mono" style="color: var(--text-muted); font-size: 11px;">{formatTimestamp(getField(evt, 'timestamp', ''))}</td>
                                                    <td class="mono">{getField(evt, 'host_id', 'HostID')}</td>
                                                    <td class={`mono siem-sev-${getSev(type)}`}>{type}</td>
                                                    <td class="mono" style="max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-muted);">
                                                        {raw}
                                                    </td>
                                                </tr>
                                                <Show when={expanded}>
                                                    <tr>
                                                        <td colspan="5" style="padding: 0;">
                                                            <div class="siem-detail-grid">
                                                                <div>
                                                                    <div class="siem-detail-section-title">METADATA</div>
                                                                    <table class="siem-kv-table">
                                                                        <tr><td class="siem-kv-key">SYSTEM_ID</td><td class="siem-kv-val">{id}</td></tr>
                                                                        <tr><td class="siem-kv-key">TENANT_ID</td><td class="siem-kv-val">{getField(evt, 'tenant_id', 'GLOBAL')}</td></tr>
                                                                        <tr><td class="siem-kv-key">SOURCE_IP</td><td class="siem-kv-val">{getField(evt, 'source_ip', '—')}</td></tr>
                                                                        <tr><td class="siem-kv-key">AUDIT_USER</td><td class="siem-kv-val highlight">{getField(evt, 'user', '—')}</td></tr>
                                                                    </table>
                                                                </div>
                                                                <div>
                                                                    <div class="siem-detail-section-title">RAW_RECONSTRUCTION</div>
                                                                    <div class="siem-raw-log-well">{raw}</div>
                                                                </div>
                                                            </div>
                                                        </td>
                                                    </tr>
                                                </Show>
                                            </>
                                        )
                                    }}
                                </For>
                            </tbody>
                        </table>
                        <Show when={events.loading}>
                            <div style="padding: 100px 0;">
                                <LoadingState message="RECONSTRUCTING_SEGMENT_INDEX..." />
                            </div>
                        </Show>
                    </div>
                </main>
            </div>
        </PageLayout>
    );
};
