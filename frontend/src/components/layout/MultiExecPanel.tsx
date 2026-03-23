import { Component, For, Show, createSignal, onMount } from 'solid-js';
import { useApp } from '@core/store';
import { subscribe } from '@core/bridge';
import { IS_BROWSER } from '@core/context';
import { services } from '../../../wailsjs/go/models';

export const MultiExecPanel: Component = () => {
    const [state] = useApp();
    const [command, setCommand] = createSignal('');
    const [selectedHosts, setSelectedHosts] = createSignal<Set<string>>(new Set());
    const [executing, setExecuting] = createSignal(false);
    type JobResult = { host_id: string, host_label: string, status: string, output?: string, error?: string };
    type ActiveJob = { id: string, command: string, status: string, host_ids?: string[], results: JobResult[] };
    const [activeJob, setActiveJob] = createSignal<ActiveJob | null>(null);
    const [recentJobs, setRecentJobs] = createSignal<ActiveJob[]>([]);
    const [safetyResult, setSafetyResult] = createSignal<services.DestructiveCheckResult | null>(null);
    const [showSafetyModal, setShowSafetyModal] = createSignal(false);
    const [concurrency, setConcurrency] = createSignal(5);

    onMount(async () => {
        if (IS_BROWSER) return;
        const { GetRecentJobs } = await import('../../../wailsjs/go/services/MultiExecService');
        setRecentJobs(await GetRecentJobs(5) || []);

        subscribe('multiexec.progress', (data: { job_id: string, index: number, status: string, output?: string }) => {
            const current = activeJob();
            if (current && data.job_id === current.id && current.results) {
                const next = { ...current, results: [...current.results] };
                next.results[data.index] = { ...next.results[data.index], status: data.status };
                if (data.output) next.results[data.index].output = data.output;
                setActiveJob(next);
            }
        });

        subscribe('multiexec.completed', (data: ActiveJob) => {
            if (activeJob() && data.id === activeJob()!.id) {
                setActiveJob(data);
                setExecuting(false);
                import('../../../wailsjs/go/services/MultiExecService')
                    .then(m => m.GetRecentJobs(5))
                    .then(setRecentJobs);
            }
        });
    });

    const toggleHost = (id: string) => {
        const next = new Set(selectedHosts());
        if (next.has(id)) next.delete(id);
        else next.add(id);
        setSelectedHosts(next);
    };

    const handleConfirmRun = async () => {
        setShowSafetyModal(false);
        await runExecution();
    };

    const handleRun = async () => {
        if (!command() || selectedHosts().size === 0 || IS_BROWSER) return;
        const { CheckSafety } = await import('../../../wailsjs/go/services/MultiExecService');
        const res = await CheckSafety(command());
        setSafetyResult(res);
        if (res.is_destructive) setShowSafetyModal(true);
        else await runExecution();
    };

    const runExecution = async () => {
        setExecuting(true);
        try {
            const { Execute } = await import('../../../wailsjs/go/services/MultiExecService');
            const jobId = await Execute(command(), Array.from(selectedHosts()), 60);
            setActiveJob({
                id: jobId, command: command(), status: 'running',
                results: Array.from(selectedHosts()).map(id => ({
                    host_id: id,
                    host_label: state.hosts.find(h => h.id === id)?.label || 'Unknown',
                    status: 'pending'
                }))
            });
        } catch (err) { console.error(err); setExecuting(false); }
    };

    const handleConcurrencyChange = (val: number) => {
        setConcurrency(val);
        if (!IS_BROWSER) import('../../../wailsjs/go/services/MultiExecService')
            .then(m => m.SetMaxConcurrency(val));
    };

    return (
        <div class="ops-panel">
            <div class="panel-header">
                <h2>Batch Execution</h2>
                <div class="panel-actions">
                    <button class="btn btn-primary" onClick={handleRun} disabled={executing()}>
                        {executing() ? 'Running...' : 'Run on Selected'}
                    </button>
                </div>
            </div>

            <div class="ops-layout">
                <div class="ops-sidebar">
                    <h3>Select Hosts</h3>
                    <div class="host-selection-list">
                        <For each={state.hosts}>
                            {(host) => (
                                <label class="checkbox-item">
                                    <input
                                        type="checkbox"
                                        checked={selectedHosts().has(host.id)}
                                        onChange={() => toggleHost(host.id)}
                                    />
                                    <span>{host.label || host.hostname}</span>
                                </label>
                            )}
                        </For>
                        <Show when={state.hosts.length === 0}>
                            <div class="dash-empty" style="margin-top: 16px;">
                                <div style="font-size: 20px; margin-bottom: 8px;">📡</div>
                                <p style="margin: 0; color: var(--text-primary); font-weight: 500;">No hosts available</p>
                                <p style="margin: 4px 0 0; font-size: 11px;">Add hosts in the sidebar to run batch commands</p>
                            </div>
                        </Show>
                    </div>

                    <div class="concurrency-control" style="margin-top: 24px; padding: 12px; background: rgba(0,0,0,0.2); border-radius: 8px;">
                        <h4 style="margin: 0 0 12px; font-size: 13px;">Batch Concurrency</h4>
                        <div style="display: flex; align-items: center; gap: 12px;">
                            <input
                                type="range"
                                min="1"
                                max="20"
                                value={concurrency()}
                                onInput={(e) => handleConcurrencyChange(parseInt(e.currentTarget.value))}
                                style="flex: 1;"
                            />
                            <span style="font-family: monospace; font-size: 12px; min-width: 2ch;">{concurrency()}</span>
                        </div>
                        <p style="margin: 8px 0 0; font-size: 10px; color: var(--text-muted);">
                            Limits how many hosts are processed at the same time.
                        </p>
                    </div>
                </div>

                <div class="ops-main">
                    <div class="command-input-area">
                        <textarea
                            placeholder="Enter command to run on all selected hosts..."
                            value={command()}
                            onInput={(e) => setCommand(e.currentTarget.value)}
                            class="terminal-textarea"
                        />
                    </div>

                    <Show when={activeJob()}>
                        <div class="execution-results">
                            <h3>Results: {activeJob()?.id.slice(0, 8)}</h3>
                            <div class="results-grid">
                                <For each={activeJob()?.results || []}>
                                    {(result: JobResult) => (
                                        <div class={`result-card ${result.status}`}>
                                            <div class="result-header">
                                                <span class="host-label">{result.host_label}</span>
                                                <span class={`status-badge ${result.status}`}>{result.status}</span>
                                            </div>
                                            <Show when={result.output}>
                                                <pre class="result-output">{result.output}</pre>
                                            </Show>
                                            <Show when={result.error}>
                                                <div class="result-error">{result.error}</div>
                                            </Show>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>
                    </Show>

                    <Show when={!activeJob() && recentJobs().length > 0}>
                        <div class="recent-jobs">
                            <h3>Recent Jobs</h3>
                            <For each={recentJobs()}>
                                {(job) => (
                                    <div class="job-summary" onClick={() => setActiveJob(job)}>
                                        <code>{job.command}</code>
                                        <span class="job-meta">{(job.host_ids?.length) || 0} hosts • {job.status}</span>
                                    </div>
                                )}
                            </For>
                        </div>
                    </Show>
                </div>
            </div>
            {/* Safety Warning Modal */}
            <Show when={showSafetyModal()}>
                <div class="modal-overlay">
                    <div class="modal-content danger" style="max-width: 500px;">
                        <div class="modal-header">
                            <h3 style="color: var(--error-color);">⚠️ DESTRUCTIVE ACTION DETECTED</h3>
                        </div>
                        <div class="modal-body">
                            <p>The command you are about to execute has been flagged as potentially destructive:</p>
                            <div class="destructive-command-box">
                                <code>{command()}</code>
                            </div>
                            <h4 style="margin: 16px 0 8px;">Matched Threats:</h4>
                            <ul class="threat-list">
                                <For each={safetyResult()?.threats}>
                                    {(threat) => <li>{threat}</li>}
                                </For>
                            </ul>
                            <p style="margin-top: 16px; font-weight: bold; color: var(--error-color);">
                                This will execute on {selectedHosts().size} hosts simultaneously.
                            </p>
                        </div>
                        <div class="modal-footer" style="display: flex; justify-content: flex-end; gap: 12px;">
                            <button class="btn" onClick={() => setShowSafetyModal(false)}>Cancel</button>
                            <button class="btn btn-danger" onClick={handleConfirmRun}>
                                I understand, execute anyway
                            </button>
                        </div>
                    </div>
                </div>
            </Show>
        </div>
    );
};
