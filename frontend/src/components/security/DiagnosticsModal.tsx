import { Component, Show } from 'solid-js';
import '../../styles/modal.css';

interface DiagnosticResult {
    step: 'dns' | 'tcp' | 'banner' | 'auth';
    status: 'success' | 'failed' | 'pending';
    message: string;
    latencyMs?: number;
}

interface DiagnosticsModalProps {
    isOpen: boolean;
    onClose: () => void;
    hostName: string;
    hostAddress: string;
    port: number;
    results: DiagnosticResult[];
    probableCause?: string;
    suggestedActions?: { label: string; action: () => void }[];
}

export const DiagnosticsModal: Component<DiagnosticsModalProps> = (props) => {
    return (
        <Show when={props.isOpen}>
            <div class="modal-overlay" onClick={props.onClose}>
                <div class="modal-content diagnostics-modal" onClick={e => e.stopPropagation()}>
                    <div class="modal-header">
                        <h2>CONNECTION FAILED</h2>
                        <button class="icon-btn" onClick={props.onClose}>×</button>
                    </div>

                    <div class="modal-body">
                        <div class="diag-target">
                            <span class="label">Host:</span> {props.hostName} ({props.hostAddress}:{props.port})
                        </div>

                        <div class="diag-section">
                            <h3>DIAGNOSTIC RESULTS:</h3>
                            <div class="diag-steps">
                                <Show when={props.results}>
                                    {props.results.map(res => (
                                        <div class={`diag-step ${res.status}`}>
                                            <span class="diag-icon">
                                                {res.status === 'success' ? '✓' : res.status === 'failed' ? '✗' : '⋯'}
                                            </span>
                                            <span class="diag-msg">{res.message}</span>
                                            <Show when={res.latencyMs}>
                                                <span class="diag-latency">({res.latencyMs}ms)</span>
                                            </Show>
                                        </div>
                                    ))}
                                </Show>
                            </div>
                        </div>

                        <Show when={props.probableCause}>
                            <div class="diag-section error-cause">
                                <h3>PROBABLE CAUSE:</h3>
                                <div class="cause-text">{props.probableCause}</div>
                            </div>
                        </Show>

                        <Show when={props.suggestedActions && props.suggestedActions.length > 0}>
                            <div class="diag-section">
                                <h3>SUGGESTED ACTIONS:</h3>
                                <div class="diag-actions">
                                    {props.suggestedActions?.map(action => (
                                        <button class="text-btn action-link" onClick={action.action}>
                                            → {action.label}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        </Show>
                    </div>

                    <div class="modal-footer">
                        <button class="action-btn" onClick={props.onClose}>Close</button>
                        <button class="action-btn primary" onClick={() => {
                            // Copy debug trace
                        }}>Copy Debug Trace</button>
                    </div>
                </div>
            </div>
            <style>{`
                .diagnostics-modal {
                    max-width: 500px;
                    border: 1px solid var(--error);
                    box-shadow: 0 0 20px rgba(239, 68, 68, 0.1);
                }
                .diagnostics-modal .modal-header h2 {
                    color: var(--error);
                    display: flex;
                    align-items: center;
                    gap: 8px;
                }
                .diagnostics-modal .modal-header h2::before {
                    content: '✗';
                    font-weight: bold;
                }
                .diag-target {
                    padding: 12px;
                    background: var(--bg-tertiary);
                    border-radius: 6px;
                    font-family: var(--font-mono);
                    font-size: 13px;
                    margin-bottom: 20px;
                }
                .diag-target .label {
                    color: var(--text-muted);
                }
                .diag-section {
                    margin-bottom: 20px;
                }
                .diag-section h3 {
                    font-size: 11px;
                    color: var(--text-muted);
                    text-transform: uppercase;
                    letter-spacing: 1px;
                    margin: 0 0 12px 0;
                }
                .diag-steps {
                    display: flex;
                    flex-direction: column;
                    gap: 8px;
                    font-family: var(--font-mono);
                    font-size: 13px;
                }
                .diag-step {
                    display: flex;
                    align-items: center;
                    gap: 12px;
                }
                .diag-step.success { color: var(--success); }
                .diag-step.failed { color: var(--error); }
                .diag-step.pending { color: var(--text-muted); }
                .diag-msg { color: var(--text-primary); }
                .diag-latency { color: var(--text-muted); }
                
                .error-cause {
                    padding-left: 12px;
                    border-left: 2px solid var(--error);
                }
                .cause-text {
                    color: var(--error);
                    font-family: var(--font-mono);
                    font-size: 13px;
                    line-height: 1.5;
                }
                .diag-actions {
                    display: flex;
                    flex-direction: column;
                    align-items: flex-start;
                    gap: 8px;
                }
                .action-link {
                    padding: 4px 8px;
                    margin-left: -8px;
                    color: var(--accent-primary);
                }
                .action-link:hover {
                    background: var(--bg-tertiary);
                }
            `}</style>
        </Show>
    );
};
