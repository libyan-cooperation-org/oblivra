import { Component, createSignal, For, Show } from 'solid-js';
import { RunOQL } from '../../wailsjs/go/services/AnalyticsService';
import { Card, Button, Badge } from '../components/ui/TacticalComponents';
import { QueryEditor } from '../components/oql/QueryEditor';
import '../styles/splunk.css';

// Wails bindings sometimes fail to export deeply nested sub-package models
// We define the OQL result types here locally.
export namespace oql {
    export interface Row {
        [key: string]: any;
    }
    export interface QueryMeta {
        ExecTime: number;
        OutputRows: number;
        ScannedRows: number;
        MemoryBytes: number;
        ScanBytes: number;
    }
    export interface QueryResult {
        Rows: Row[];
        Meta: QueryMeta;
        Warnings: string[];
    }
}

export const OQLDashboard: Component = () => {
    const [query, setQuery] = createSignal<string>("* | stats count() by host | sort -count");
    const [results, setResults] = createSignal<oql.Row[]>([]);
    const [meta, setMeta] = createSignal<oql.QueryMeta | null>(null);
    const [loading, setLoading] = createSignal<boolean>(false);
    const [error, setError] = createSignal<string | null>(null);
    const [showExplain, setShowExplain] = createSignal<boolean>(false);

    const executeQuery = async () => {
        if (!query().trim()) return;
        setLoading(true);
        setError(null);
        try {
            const res = await RunOQL(query());
            if (res) {
                setResults((res.Rows as oql.Row[]) || []);
                setMeta(res.Meta as any || null);
                if (res.Warnings && res.Warnings.length > 0) {
                    console.warn("OQL Warnings:", res.Warnings);
                }
            } else {
                setResults([]);
                setMeta(null);
            }
        } catch (err: any) {
            setError(err.message || String(err));
            setResults([]);
            setMeta(null);
        } finally {
            setLoading(false);
        }
    };

    // Extract columns from the first row of results for the table header
    const columns = () => {
        if (results().length === 0) return [];
        return Object.keys(results()[0]);
    };

    return (
        <div class="splunk-ui" style="display: flex; flex-direction: column; gap: 1rem; padding: 1rem; height: 100%; overflow: hidden;">
            {/* Header / Query Bar */}
            <Card variant="raised" padding="16px" style="display: flex; flex-direction: column; gap: 12px; background: rgba(10, 14, 23, 0.85); backdrop-filter: blur(12px);">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <h1 class="dashboard-name" style="font-family: var(--font-ui); font-weight: 800; letter-spacing: 1px; color: var(--text-primary); margin: 0; display: flex; align-items: center; gap: 8px;">
                        <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                        </svg>
                        OQL EXPLORER
                    </h1>
                    <div style="display: flex; gap: 8px;">
                        <Badge severity="info">OQL Engine v2</Badge>
                        <Show when={meta()}>
                            <Badge severity="success">Rows: {meta()!.OutputRows}</Badge>
                            <Badge severity="neutral">{Math.round(meta()!.ExecTime / 1000000)}ms</Badge>
                        </Show>
                    </div>
                </div>

                <div style="display: flex; flex-direction: column; gap: 8px;">
                    <QueryEditor 
                        value={query()} 
                        onInput={setQuery} 
                        onRun={executeQuery} 
                    />
                    <div style="display: flex; justify-content: flex-end; gap: 8px; margin-top: 8px;">
                        <Button 
                            variant="secondary" 
                            size="md" 
                            onClick={() => setShowExplain(!showExplain())}
                        >
                            Explain Plan
                        </Button>
                        <Button 
                            variant="primary" 
                            size="md" 
                            onClick={executeQuery} 
                            disabled={loading()}
                            style="min-width: 120px;"
                        >
                            {loading() ? 'Executing...' : 'Run Query ⚡'}
                        </Button>
                    </div>
                </div>
            </Card>

            <Show when={showExplain()}>
                <Card variant="raised" padding="16px" style="background: rgba(10, 14, 23, 0.9); border: 1px solid var(--accent-primary);">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                        <h4 style="margin: 0; color: var(--accent-primary); text-transform: uppercase; font-size: 12px;">Logical Query Plan</h4>
                        <Button variant="ghost" size="sm" onClick={() => setShowExplain(false)}>×</Button>
                    </div>
                    <pre style="font-family: var(--font-mono); font-size: 11px; color: var(--text-secondary); white-space: pre-wrap;">
                        {/* Plan would be fetched from backend Optimizer.ExplainPlan in a real scenario */}
                        STÁGE 0: SCAN partition=all scan_type=PARALLEL_PARTITION
                        STÁGE 1: BLOOM_FILTER matches=maybe
                        STÁGE 2: PREDICATE_PUSHDOWN (optimized)
                        STÁGE 3: PIPELINE_EXECUTE
                    </pre>
                </Card>
            </Show>

            <Show when={error()}>
                <Card variant="raised" padding="16px" style="border-left: 4px solid var(--status-error); background: rgba(220, 38, 38, 0.1);">
                    <div style="color: var(--status-error); font-family: var(--font-mono); font-size: 13px;">
                        <strong>Query Error:</strong> {error()}
                    </div>
                </Card>
            </Show>

            {/* Results Table */}
            <Card variant="raised" padding="0" style="flex: 1; display: flex; flex-direction: column; overflow: hidden; background: rgba(10, 14, 23, 0.6); backdrop-filter: blur(8px);">
                <div style="padding: 12px 16px; border-bottom: 1px solid var(--border-primary); background: rgba(0,0,0,0.2);">
                    <h3 style="margin: 0; font-family: var(--font-ui); font-size: 12px; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 1px;">Results</h3>
                </div>
                
                <div style="flex: 1; overflow: auto; padding: 0;">
                    <Show when={results().length > 0} fallback={
                        <div style="display: flex; justify-content: center; align-items: center; height: 100%; color: var(--text-muted); font-family: var(--font-mono); font-size: 13px;">
                            {loading() ? 'Processing query...' : 'No results. Run a query to see data.'}
                        </div>
                    }>
                        <table style="width: 100%; border-collapse: collapse; text-align: left; font-family: var(--font-mono); font-size: 12px;">
                            <thead style="position: sticky; top: 0; background: var(--bg-secondary); box-shadow: 0 1px 0 var(--border-primary);">
                                <tr>
                                    <For each={columns()}>
                                        {(col) => (
                                            <th style="padding: 8px 12px; font-weight: 600; color: var(--text-secondary); white-space: nowrap; border-right: 1px solid var(--border-primary);">&nbsp;{col}&nbsp;</th>
                                        )}
                                    </For>
                                </tr>
                            </thead>
                            <tbody>
                                <For each={results()}>
                                    {(row, idx) => (
                                        <tr style={{ "background": idx() % 2 === 0 ? 'transparent' : 'rgba(255,255,255,0.02)', "border-bottom": '1px solid var(--border-primary)' }}>
                                            <For each={columns()}>
                                                {(col) => {
                                                    const val = row[col];
                                                    const displayVal = typeof val === 'object' ? JSON.stringify(val) : String(val);
                                                    return (
                                                        <td style="padding: 6px 12px; color: var(--text-primary); max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; border-right: 1px solid var(--border-primary);">
                                                            {displayVal}
                                                        </td>
                                                    );
                                                }}
                                            </For>
                                        </tr>
                                    )}
                                </For>
                            </tbody>
                        </table>
                    </Show>
                </div>
            </Card>
        </div>
    );
};

