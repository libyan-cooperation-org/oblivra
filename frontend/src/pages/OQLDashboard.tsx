import { Component, createSignal, For, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { RunOQL } from '../../wailsjs/go/services/AnalyticsService';
import { 
    PageLayout, 
    Panel, 
    Badge, 
    Button, 
    Table, 
    Column,
    Notice,
    SectionHeader,
    CodeBlock,
    LoadingState 
} from '@components/ui';
import { QueryEditor } from '../components/oql/QueryEditor';

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
    const navigate = useNavigate();
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

    const columns = () => {
        if (results().length === 0) return [];
        const keys = Object.keys(results()[0]);
        
        return keys.map(key => {
            const col: Column<oql.Row> = {
                key,
                label: key.toUpperCase(),
                mono: true,
                render: (row) => {
                    const val = row[key];
                    const displayVal = typeof val === 'object' ? JSON.stringify(val) : String(val);
                    
                    // Entity Linking
                    const isIP = (key.toLowerCase().includes('ip') || key.toLowerCase().includes('addr')) && /^(?:\d{1,3}\.){3}\d{1,3}$/.test(displayVal);
                    const isHost = (key.toLowerCase().includes('host') || key.toLowerCase().includes('device')) && displayVal.length > 3;

                    if (isIP || isHost) {
                        return (
                            <a 
                                href="#"
                                style="color: var(--accent-primary); text-decoration: none; border-bottom: 1px dashed rgba(0, 153, 224, 0.3);"
                                onClick={(ev) => {
                                    ev.preventDefault();
                                    navigate(`/entity/${isIP ? 'ip' : 'host'}/${displayVal}`);
                                }}
                            >
                                {displayVal}
                            </a>
                        );
                    }
                    return displayVal;
                }
            };
            return col;
        });
    };

    return (
        <PageLayout 
            title="OQL Explorer" 
            subtitle="ANALYTICS_QUERY_ENGINE_V2"
            actions={
                <div style="display: flex; gap: 8px;">
                    <Badge severity="info">ENGINE:OPTIMIZED</Badge>
                    <Show when={meta()}>
                        <Badge severity="success">ROWS:{meta()!.OutputRows}</Badge>
                        <Badge severity="neutral">{Math.round(meta()!.ExecTime / 1000000)}ms</Badge>
                    </Show>
                </div>
            }
        >
            <div style="display: flex; flex-direction: column; gap: var(--gap-lg); height: 100%; overflow: hidden;">
                {/* Editor Panel */}
                <Panel title="QUERY_INPUT" subtitle="OQL_PIPELINE">
                    <div style="display: flex; flex-direction: column; gap: var(--gap-md);">
                        <QueryEditor 
                            value={query()} 
                            onInput={setQuery} 
                            onRun={executeQuery} 
                        />
                        <div style="display: flex; justify-content: flex-end; gap: var(--gap-md);">
                            <Button 
                                variant="default" 
                                onClick={() => setShowExplain(!showExplain())}
                            >
                                {showExplain() ? 'HIDE_PLAN' : 'EXPLAIN_PLAN'}
                            </Button>
                            <Button 
                                variant="primary" 
                                onClick={executeQuery} 
                                disabled={loading()}
                                style="min-width: 140px;"
                            >
                                {loading() ? 'EXECUTING...' : 'RUN_QUERY_PIPELINE'}
                            </Button>
                        </div>
                    </div>
                </Panel>

                {/* Explain Plan */}
                <Show when={showExplain()}>
                    <Panel title="EXECUTION_PLAN" subtitle="QUERY_OPTIMIZER_LOG">
                        <CodeBlock>
                            STÁGE 0: SCAN partition=all scan_type=PARALLEL_PARTITION
                            STÁGE 1: BLOOM_FILTER matches=maybe
                            STÁGE 2: PREDICATE_PUSHDOWN (optimized)
                            STÁGE 3: PIPELINE_EXECUTE
                        </CodeBlock>
                    </Panel>
                </Show>

                {/* Error Notice */}
                <Show when={error()}>
                    <Notice level="error">
                        <div style="font-family: var(--font-mono); font-size: 13px;">
                            <strong>QUERY_EXCEPTION:</strong> {error()}
                        </div>
                    </Notice>
                </Show>

                {/* Results Table Panel */}
                <Panel title="QUERY_RESULTS" subtitle="FETCH_STATE" noPadding class="flex-1 overflow-hidden">
                    <div style="height: 100%; display: flex; flex-direction: column;">
                        <Show when={loading() && results().length === 0}>
                            <div style="flex: 1; display: flex; align-items: center; justify-content: center;">
                                <LoadingState message="PARSING_OQL_AST..." />
                            </div>
                        </Show>
                        <Show when={!loading() || results().length > 0}>
                            <div style="flex: 1; overflow: auto; min-height: 200px;">
                                <Table 
                                    columns={columns()} 
                                    data={results()} 
                                    emptyText="NO_QUERY_RESULTS_FETCHED"
                                    striped
                                />
                            </div>
                        </Show>
                    </div>
                </Panel>
            </div>
        </PageLayout>
    );
};
