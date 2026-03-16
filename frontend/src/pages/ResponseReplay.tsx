import { Component, createSignal, onMount, For, Show } from 'solid-js';

interface ExecutionSignature {
    id: string;
    timestamp: string;
    event_batch_id: string;
    policy_hash: string;
    action_taken: string;
    input_hash: string;
    final_hash: string;
}

const svc = () => (window as any).go?.services?.DeterministicResponseService;

const FieldRow: Component<{ label: string; value: string; accent?: string }> = (p) => (
    <div style={{ display: 'contents' }}>
        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 700, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '0.8px', 'padding': '4px 0', 'align-self': 'start', 'margin-top': '2px' }}>{p.label}</div>
        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: p.accent ?? 'var(--text-secondary)', 'word-break': 'break-all', padding: '4px 0', 'line-height': 1.5 }}>{p.value}</div>
    </div>
);

export const ResponseReplay: Component = () => {
    const [signatures, setSignatures]   = createSignal<ExecutionSignature[]>([]);
    const [inputAction, setInputAction] = createSignal('IsolateHost');
    const [inputEvent, setInputEvent]   = createSignal('{"event_id": "EVT-123", "type": "LateralMovement"}');
    const [inputPolicy, setInputPolicy] = createSignal('8e3a2b4cd9');
    const [replayData, setReplayData]   = createSignal<any>(null);
    const [running, setRunning]         = createSignal(false);
    const [runErr, setRunErr]           = createSignal('');
    const [loading, setLoading]         = createSignal(true);

    const loadSigs = async () => {
        setLoading(true);
        try {
            const data = await svc()?.GetSignatures();
            setSignatures(data || []);
        } catch { /* service may not be ready */ }
        finally { setLoading(false); }
    };

    onMount(loadSigs);

    const runSimulation = async () => {
        setRunning(true);
        setRunErr('');
        setReplayData(null);
        try {
            const s = svc();
            if (!s) throw new Error('DeterministicResponseService not available');
            const newSig = await s.MapResponse(inputAction(), inputEvent(), inputPolicy());
            const verify = await s.Replay(newSig.input_hash, inputPolicy(), inputAction());
            setReplayData({ signature: newSig, verified: verify });
            loadSigs();
        } catch (err: any) {
            setRunErr(err?.message ?? String(err));
        } finally {
            setRunning(false);
        }
    };

    const sorted = () => [...signatures()].sort((a, b) =>
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            overflow: 'hidden',
            background: 'var(--surface-0)',
            padding: '28px 32px 0 32px',
        }}>
            {/* ── Header ── */}
            <div style={{
                'margin-bottom': '20px',
                'padding-bottom': '16px',
                'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': 0,
            }}>
                <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 700, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '2px', 'margin-bottom': '4px' }}>
                    AUDIT TRAIL
                </div>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800, color: 'var(--text-primary)', margin: 0, 'letter-spacing': '-0.5px' }}>
                    Deterministic Response Engine
                </h1>
                <p style={{ 'font-size': '11px', color: 'var(--text-muted)', margin: '6px 0 0 0', 'font-family': 'var(--font-ui)' }}>
                    Prove response executions via cryptographic signatures. Every action is deterministically reproducible.
                </p>
            </div>

            {/* ── Two-column layout ── */}
            <div style={{ display: 'grid', 'grid-template-columns': '1fr 1fr', gap: '16px', flex: 1, 'min-height': 0, 'padding-bottom': '28px' }}>

                {/* Simulation panel */}
                <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border-primary)', display: 'flex', 'flex-direction': 'column', overflow: 'hidden' }}>
                    <div style={{ padding: '12px 16px', 'border-bottom': '1px solid var(--border-primary)', 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1.5px' }}>
                        EXECUTION SIMULATOR
                    </div>
                    <div style={{ flex: 1, 'overflow-y': 'auto', padding: '16px', display: 'flex', 'flex-direction': 'column', gap: '14px' }}>
                        {/* Action select */}
                        <div>
                            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'margin-bottom': '6px' }}>Response Action</div>
                            <select
                                value={inputAction()}
                                onChange={(e) => setInputAction(e.currentTarget.value)}
                                style={{ width: '100%', background: 'var(--surface-0)', border: '1px solid var(--border-primary)', color: 'var(--text-primary)', 'font-family': 'var(--font-mono)', 'font-size': '11px', padding: '7px 10px', outline: 'none' }}
                            >
                                <option value="IsolateHost">IsolateHost</option>
                                <option value="BlockIP">BlockIP</option>
                                <option value="KillProcess">KillProcess</option>
                                <option value="DisableUser">DisableUser</option>
                            </select>
                        </div>

                        {/* Policy hash */}
                        <div>
                            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'margin-bottom': '6px' }}>State Policy Hash</div>
                            <input
                                type="text"
                                value={inputPolicy()}
                                onInput={(e) => setInputPolicy(e.currentTarget.value)}
                                style={{ width: '100%', background: 'var(--surface-0)', border: '1px solid var(--border-primary)', color: 'var(--text-primary)', 'font-family': 'var(--font-mono)', 'font-size': '11px', padding: '7px 10px', outline: 'none', 'box-sizing': 'border-box' }}
                                placeholder="e.g. 8e3a2b4cd9"
                            />
                        </div>

                        {/* Event JSON */}
                        <div>
                            <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'margin-bottom': '6px' }}>Trigger Event (JSON)</div>
                            <textarea
                                value={inputEvent()}
                                onInput={(e) => setInputEvent(e.currentTarget.value)}
                                rows={5}
                                style={{ width: '100%', background: 'var(--surface-0)', border: '1px solid var(--border-primary)', color: 'var(--text-primary)', 'font-family': 'var(--font-mono)', 'font-size': '11px', padding: '8px 10px', outline: 'none', resize: 'vertical', 'box-sizing': 'border-box' }}
                            />
                        </div>

                        {/* Run button */}
                        <button
                            onClick={runSimulation}
                            disabled={running()}
                            style={{
                                background: running() ? 'var(--surface-2)' : 'var(--accent-primary)',
                                border: 'none',
                                color: running() ? 'var(--text-muted)' : 'var(--surface-0)',
                                'font-family': 'var(--font-mono)',
                                'font-size': '11px',
                                'font-weight': 800,
                                'text-transform': 'uppercase',
                                'letter-spacing': '1px',
                                padding: '10px',
                                cursor: running() ? 'wait' : 'pointer',
                                width: '100%',
                            }}
                        >
                            {running() ? 'EXECUTING...' : 'EXECUTE & GENERATE SIGNATURE'}
                        </button>

                        {/* Run error */}
                        <Show when={runErr()}>
                            <div style={{ padding: '8px 12px', border: '1px solid var(--alert-critical)', 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--alert-critical)' }}>
                                {runErr()}
                            </div>
                        </Show>

                        {/* Result */}
                        <Show when={replayData()}>
                            <div style={{ 'border-top': '1px solid var(--border-primary)', 'padding-top': '14px' }}>
                                <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'margin-bottom': '10px', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center' }}>
                                    <span>EXECUTION SIGNATURE</span>
                                    <Show when={replayData().verified?.matched_past}>
                                        <span style={{ color: 'var(--alert-low)', 'font-size': '9px', border: '1px solid var(--alert-low)', padding: '2px 6px' }}>HISTORICAL MATCH</span>
                                    </Show>
                                </div>
                                <div style={{ background: 'var(--surface-0)', border: '1px solid var(--border-primary)', 'border-left': '3px solid var(--accent-primary)', padding: '12px', display: 'grid', 'grid-template-columns': '90px 1fr', gap: '6px 12px' }}>
                                    <FieldRow label="ID" value={replayData().signature.id} accent="var(--accent-primary)" />
                                    <FieldRow label="Input Hash" value={replayData().signature.input_hash} />
                                    <FieldRow label="Policy Hash" value={replayData().signature.policy_hash} />
                                    <FieldRow label="Final Proof" value={replayData().signature.final_hash} accent="var(--alert-medium)" />
                                </div>
                            </div>
                        </Show>
                    </div>
                </div>

                {/* History panel */}
                <div style={{ background: 'var(--surface-1)', border: '1px solid var(--border-primary)', display: 'flex', 'flex-direction': 'column', overflow: 'hidden' }}>
                    <div style={{ padding: '12px 16px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1.5px' }}>
                        <span>IMMUTABLE HISTORY</span>
                        <span>{signatures().length} EXECUTIONS</span>
                    </div>
                    <div style={{ flex: 1, 'overflow-y': 'auto', padding: '8px' }}>
                        <Show when={loading()}>
                            <div style={{ padding: '24px', 'text-align': 'center', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)' }}>LOADING...</div>
                        </Show>
                        <Show when={!loading() && signatures().length === 0}>
                            <div style={{ padding: '32px', 'text-align': 'center', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>
                                NO EXECUTIONS RECORDED<br />
                                <span style={{ 'font-size': '10px', opacity: 0.6 }}>Run the simulator to generate a proof.</span>
                            </div>
                        </Show>
                        <For each={sorted()}>
                            {(sig) => (
                                <div style={{
                                    padding: '10px 12px',
                                    'margin-bottom': '4px',
                                    background: 'var(--surface-0)',
                                    border: '1px solid var(--border-primary)',
                                    'border-left': '3px solid var(--accent-primary)',
                                }}>
                                    <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '6px', 'padding-bottom': '6px', 'border-bottom': '1px solid var(--border-primary)' }}>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 800, color: 'var(--accent-primary)', 'text-transform': 'uppercase' }}>{sig.action_taken}</span>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-muted)' }}>{new Date(sig.timestamp).toLocaleTimeString()}</span>
                                    </div>
                                    <div style={{ display: 'grid', 'grid-template-columns': '60px 1fr', gap: '3px 8px' }}>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>INPUT</span>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-secondary)', overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{sig.input_hash.substring(0, 28)}…</span>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>PROOF</span>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--alert-medium)', overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }}>{sig.final_hash.substring(0, 28)}…</span>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

            </div>
        </div>
    );
};
