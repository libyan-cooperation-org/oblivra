import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { subscribe } from '@core/bridge';
import '@styles/multiexec.css';

interface Host {
    id: string;
    label: string;
    hostname: string;
}

interface MultiExecResult {
    host_id: string;
    exit_code: number;
    output: string;
    duration_ms: number;
    error?: string;
}

const hostSvc = (window as any).go?.app?.HostService;
const multiExecSvc = (window as any).go?.app?.MultiExecService;

export const MultiExec: Component = () => {
    const [hosts, setHosts] = createSignal<Host[]>([]);
    const [selectedHosts, setSelectedHosts] = createSignal<Set<string>>(new Set());
    const [command, setCommand] = createSignal('');
    const [isRunning, setIsRunning] = createSignal(false);
    const [results, setResults] = createSignal<Record<string, MultiExecResult>>({});
    const [viewMode, setViewMode] = createSignal<'split' | 'unified'>('split');
    const [activeJobId, setActiveJobId] = createSignal<string | null>(null);

    let unsubProgress: (() => void) | undefined;
    let unsubCompleted: (() => void) | undefined;

    onMount(async () => {
        if (hostSvc) {
            try {
                const list = await hostSvc.ListHosts();
                // If it's empty, provide some dummy data for testing
                if (!list || list.length === 0) {
                    setHosts([
                        { id: 'localhost', label: 'Local Dev', hostname: 'localhost' },
                        { id: '127.0.0.1', label: 'Loopback', hostname: '127.0.0.1' },
                        { id: 'server1.local', label: 'Staging Server', hostname: 'server1.local' }
                    ]);
                } else {
                    setHosts(list);
                }
            } catch (err) {
                console.error("Failed loading hosts", err);
            }
        }

        unsubProgress = subscribe('multiexec.progress', (data: { job_id: string, result: MultiExecResult }) => {
            if (data.job_id === activeJobId()) {
                const res = data.result;
                setResults(prev => ({ ...prev, [res.host_id]: res }));
            }
        });

        unsubCompleted = subscribe('multiexec.completed', (jobId: string) => {
            if (jobId === activeJobId()) {
                setIsRunning(false);
            }
        });
    });

    onCleanup(() => {
        if (unsubProgress) unsubProgress();
        if (unsubCompleted) unsubCompleted();
    });

    const toggleHost = (id: string) => {
        const newSet = new Set(selectedHosts());
        if (newSet.has(id)) newSet.delete(id);
        else newSet.add(id);
        setSelectedHosts(newSet);
    };

    const runCommand = async () => {
        if (selectedHosts().size === 0 || !command() || !multiExecSvc) return;

        setIsRunning(true);
        setResults({});

        try {
            const targets = Array.from(selectedHosts());
            const id = await multiExecSvc.Execute(command(), targets, 60);
            setActiveJobId(id);
        } catch (err) {
            alert("Error starting multi-exec: " + err);
            setIsRunning(false);
        }
    };

    const hostMap = () => {
        const map: Record<string, Host> = {};
        hosts().forEach(h => map[h.id] = h);
        return map;
    };

    return (
        <div class="multiexec-container">
            <div class="multiexec-sidebar">
                <h3>Target Hosts</h3>
                <div class="hosts-list">
                    <For each={hosts()}>
                        {h => (
                            <label class="host-checkbox">
                                <input
                                    type="checkbox"
                                    checked={selectedHosts().has(h.id)}
                                    onChange={() => toggleHost(h.id)}
                                    disabled={isRunning()}
                                />
                                <span>{h.label} ({h.hostname})</span>
                            </label>
                        )}
                    </For>
                </div>
                <div class="run-section">
                    <button
                        class="btn-primary"
                        disabled={selectedHosts().size === 0 || !command() || isRunning()}
                        onClick={runCommand}
                    >
                        {isRunning() ? 'Running...' : `Run on ${selectedHosts().size} Hosts`}
                    </button>
                </div>
            </div>

            <div class="multiexec-main">
                <div class="cmd-input-area">
                    <textarea
                        class="cmd-input"
                        placeholder="Enter command to run concurrently across selected hosts..."
                        value={command()}
                        onInput={e => setCommand(e.currentTarget.value)}
                        disabled={isRunning()}
                    />
                </div>

                <div class="results-toolbar">
                    <div class="view-toggles">
                        <button class={`btn-toggle ${viewMode() === 'split' ? 'active' : ''}`} onClick={() => setViewMode('split')}>Split View</button>
                        <button class={`btn-toggle ${viewMode() === 'unified' ? 'active' : ''}`} onClick={() => setViewMode('unified')}>Unified View</button>
                    </div>
                </div>

                <div class="results-area">
                    <Show when={viewMode() === 'split'}>
                        <div class="split-view">
                            <For each={Array.from(selectedHosts())}>
                                {hostId => {
                                    const res = results()[hostId];
                                    const hostName = hostMap()[hostId]?.label || hostId;
                                    return (
                                        <div class={`host-result-pane ${res ? (res.error || res.exit_code !== 0 ? 'error' : 'success') : 'pending'}`}>
                                            <div class="host-result-header">
                                                <span>{hostName}</span>
                                                <Show when={res}>
                                                    <span class="host-result-status">
                                                        {res.error ? 'Failed' : `Exited: ${res.exit_code} (${(res.duration_ms / 1000000).toFixed(2)}ms)`}
                                                    </span>
                                                </Show>
                                                <Show when={!res && isRunning()}>
                                                    <span class="host-result-status pending">Running...</span>
                                                </Show>
                                            </div>
                                            <div class="host-result-output">
                                                {res ? (res.error ? res.error : res.output) : 'Waiting for results...'}
                                            </div>
                                        </div>
                                    );
                                }}
                            </For>
                        </div>
                    </Show>

                    <Show when={viewMode() === 'unified'}>
                        <div class="unified-view">
                            <For each={Object.entries(results())}>
                                {([hostId, res]) => (
                                    <div class="unified-output-block">
                                        <div class="unified-header">
                                            <strong>[{hostMap()[hostId]?.label || hostId}]</strong>
                                            <span class={`status-badge ${res.error || res.exit_code !== 0 ? 'err' : 'ok'}`}>
                                                {res.error ? 'ERR' : 'OK'}
                                            </span>
                                        </div>
                                        <pre class="unified-output">{res.error || res.output}</pre>
                                    </div>
                                )}
                            </For>
                            <Show when={isRunning()}>
                                <div class="unified-pending">Running command... waiting for remaining hosts.</div>
                            </Show>
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
