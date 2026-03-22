// SIEMSearch.tsx — Phase 2 Web: web-based SIEM query interface
import { Component, createSignal, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';

const EXAMPLE_QUERIES = [
    'failed_login source=linux | stats count by user',
    'event_type=aws_cloudtrail | where severity=critical',
    'source_ip=10.0.0.0/8 | head 50',
    'event_type=windows_login | stats count by host_id | sort -count',
];

export const SIEMSearch: Component = () => {
    const [query, setQuery] = createSignal('');
    const [results, setResults] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal('');
    const [searchTime, setSearchTime] = createSignal(0);
    const [limit, setLimit] = createSignal(100);
    const [savedSearches, setSavedSearches] = createSignal<any[]>([]);
    const [saveName, setSaveName] = createSignal('');
    const [columns, setColumns] = createSignal<string[]>([]);

    const run = async (q?: string) => {
        const qry = q ?? query().trim();
        if (!qry) return;
        setQuery(qry);
        setLoading(true);
        setError('');
        const start = Date.now();
        try {
            const r = await (SIEMService as any).SearchHostEvents(qry, limit());
            setResults(r ?? []);
            setSearchTime(Date.now() - start);
            if (r?.length > 0) {
                const cols = Object.keys(r[0]).filter(k => !['id', 'raw_line'].includes(k)).slice(0, 10);
                setColumns(cols);
            }
        } catch (e: any) {
            setError(e?.message ?? String(e));
            setResults([]);
        }
        setLoading(false);
    };

    const saveSearch = async () => {
        if (!saveName().trim() || !query().trim()) return;
        try {
            await (SIEMService as any).CreateSavedSearch?.({ name: saveName(), query: query() });
            setSaveName('');
            loadSaved();
        } catch { }
    };

    const loadSaved = async () => {
        try { setSavedSearches(await (SIEMService as any).GetSavedSearches?.() ?? []); } catch { }
    };

    const formatCellValue = (v: any): string => {
        if (v === null || v === undefined) return '—';
        if (typeof v === 'object') return JSON.stringify(v);
        const s = String(v);
        return s.length > 80 ? s.slice(0, 77) + '...' : s;
    };

    return (
        <div style="padding: 0; height: 100%; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui); display: flex; flex-direction: column; overflow: hidden;">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; align-items: center; gap: 0.75rem; padding: 0 1.5rem; background: var(--bg-secondary); flex-shrink: 0;">
                <span style="font-size: 16px;">🔎</span>
                <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">SIEM Search</h2>
            </div>

            {/* Search bar */}
            <div style="padding: 0.75rem 1.5rem; border-bottom: 1px solid var(--glass-border); background: var(--bg-secondary); flex-shrink: 0;">
                <div style="display: flex; gap: 0.5rem; align-items: stretch;">
                    <div style="flex: 1; position: relative;">
                        <input
                            placeholder='Search... e.g. "failed_login source=linux | stats count by user"'
                            value={query()}
                            onInput={e => setQuery((e.target as HTMLInputElement).value)}
                            onKeyDown={e => e.key === 'Enter' && !e.shiftKey && run()}
                            style="width: 100%; background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 9px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 12px; box-sizing: border-box;"
                        />
                    </div>
                    <select value={limit()} onChange={e => setLimit(parseInt(e.currentTarget.value))}
                        style="background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-muted); padding: 0 8px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; cursor: pointer;">
                        {[50, 100, 500, 1000].map(n => <option value={n}>{n} rows</option>)}
                    </select>
                    <button onClick={() => run()}
                        style="padding: 9px 20px; background: rgba(87,139,255,0.15); border: 1px solid rgba(87,139,255,0.4); color: var(--accent-primary); border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px; white-space: nowrap;">
                        {loading() ? '⏳' : '▶ RUN'}
                    </button>
                </div>

                {/* Example queries */}
                <div style="display: flex; gap: 0.5rem; margin-top: 0.5rem; flex-wrap: wrap;">
                    <span style="font-size: 9px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase; letter-spacing: 1px; line-height: 22px;">Examples:</span>
                    <For each={EXAMPLE_QUERIES}>
                        {(q) => (
                            <button onClick={() => run(q)}
                                style="padding: 2px 8px; font-size: 10px; font-family: var(--font-mono); border: 1px solid rgba(255,255,255,0.08); background: transparent; color: var(--text-muted); border-radius: 3px; cursor: pointer; white-space: nowrap;">
                                {q.slice(0, 40)}{q.length > 40 ? '...' : ''}
                            </button>
                        )}
                    </For>
                </div>
            </div>

            {/* Main area: results + sidebar */}
            <div style="flex: 1; display: grid; grid-template-columns: 1fr 220px; overflow: hidden;">
                {/* Results */}
                <div style="overflow: auto; display: flex; flex-direction: column;">
                    <Show when={error()}>
                        <div style="padding: 12px 1.5rem; background: rgba(248,81,73,0.06); border-bottom: 1px solid rgba(248,81,73,0.2); font-family: var(--font-mono); font-size: 11px; color: #f85149;">{error()}</div>
                    </Show>

                    <Show when={!loading() && results().length > 0}>
                        <div style="padding: 6px 1.5rem; border-bottom: 1px solid var(--glass-border); font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); display: flex; gap: 1.5rem; background: var(--bg-secondary); flex-shrink: 0;">
                            <span>{results().length} results</span>
                            <span>{searchTime()}ms</span>
                            <Show when={saveName() !== undefined}>
                                <div style="display: flex; gap: 0.5rem; margin-left: auto;">
                                    <input placeholder="Save as..." value={saveName()} onInput={e => setSaveName((e.target as HTMLInputElement).value)}
                                        style="background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 2px 8px; border-radius: 3px; font-family: var(--font-mono); font-size: 10px; width: 130px;" />
                                    <button onClick={saveSearch} style="padding: 2px 8px; font-size: 10px; font-family: var(--font-mono); border: 1px solid rgba(87,139,255,0.3); background: rgba(87,139,255,0.08); color: var(--accent-primary); border-radius: 3px; cursor: pointer; font-weight: 700;">SAVE</button>
                                </div>
                            </Show>
                        </div>
                        <div style="flex: 1; overflow: auto;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 10px; font-family: var(--font-mono);">
                                <thead style="position: sticky; top: 0; z-index: 1;">
                                    <tr style="background: var(--bg-secondary); border-bottom: 1px solid var(--glass-border);">
                                        <For each={columns()}>
                                            {(col) => <th style="padding: 8px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 0.5px; white-space: nowrap;">{col}</th>}
                                        </For>
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={results()}>
                                        {(row) => (
                                            <tr style="border-bottom: 1px solid rgba(255,255,255,0.03);"
                                                onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = 'rgba(255,255,255,0.025)'}
                                                onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = 'transparent'}>
                                                <For each={columns()}>
                                                    {(col) => <td style={`padding: 7px 12px; color: ${col === 'severity' && row[col] === 'critical' ? '#f85149' : col === 'event_type' ? 'var(--accent-primary)' : col === 'host_id' ? 'var(--text-primary)' : 'var(--text-secondary)'}; white-space: nowrap;`}>{formatCellValue(row[col])}</td>}
                                                </For>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </Show>

                    <Show when={!loading() && results().length === 0 && !error() && query()}>
                        <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                            <div style="font-size: 2.5rem; opacity: 0.2; margin-bottom: 1rem;">🔎</div>
                            NO RESULTS FOR QUERY
                        </div>
                    </Show>

                    <Show when={!query()}>
                        <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                            <div style="font-size: 2.5rem; opacity: 0.2; margin-bottom: 1rem;">🔎</div>
                            ENTER A SEARCH QUERY ABOVE<br/>
                            <span style="font-size: 10px; opacity: 0.6; display: block; margin-top: 0.5rem;">Supports OQL pipe syntax, field filters, stats aggregations</span>
                        </div>
                    </Show>

                    <Show when={loading()}>
                        <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                            SEARCHING...
                        </div>
                    </Show>
                </div>

                {/* Saved searches sidebar */}
                <div style="border-left: 1px solid var(--glass-border); padding: 1rem; overflow-y: auto; background: var(--bg-secondary);">
                    <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1.5px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem; display: flex; justify-content: space-between; align-items: center;">
                        Saved Searches
                        <button onClick={loadSaved} style="background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 12px;">↻</button>
                    </div>
                    <Show when={savedSearches().length === 0}>
                        <div style="color: var(--text-muted); font-size: 10px; font-family: var(--font-mono); opacity: 0.6;">None saved yet.<br/>Run a query and click Save.</div>
                    </Show>
                    <For each={savedSearches()}>
                        {(ss: any) => (
                            <div onClick={() => run(ss.query)}
                                style="padding: 8px; border: 1px solid var(--glass-border); border-radius: 4px; margin-bottom: 6px; cursor: pointer; font-family: var(--font-mono);"
                                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'rgba(87,139,255,0.4)'}
                                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--glass-border)'}>
                                <div style="font-size: 10px; font-weight: 700; color: var(--text-primary); margin-bottom: 3px;">{ss.name}</div>
                                <div style="font-size: 9px; color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{ss.query}</div>
                            </div>
                        )}
                    </For>
                </div>
            </div>
        </div>
    );
};
