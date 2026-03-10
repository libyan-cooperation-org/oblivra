import { Component, Show } from 'solid-js';
import '../../styles/modal.css';

interface HostKeyAlertModalProps {
    isOpen: boolean;
    onClose: () => void;
    hostName: string;
    hostAddress: string;
    port: number;
    keyType: string;
    fingerprint: string;
    isNew: boolean; // true = first time seeing host, false = key changed (DANGEROUS)
    previousFingerprint?: string;
    onAccept: () => void;
    onReject: () => void;
}

export const HostKeyAlertModal: Component<HostKeyAlertModalProps> = (props) => {
    return (
        <Show when={props.isOpen}>
            <div class="modal-overlay" onClick={props.onClose}>
                <div class="modal-content key-alert-modal" classList={{ 'danger-mode': !props.isNew }} onClick={e => e.stopPropagation()}>
                    <div class="modal-header">
                        <h2>{props.isNew ? 'NEW HOST KEY' : '⚠️ REMOTE HOST IDENTIFICATION HAS CHANGED!'}</h2>
                    </div>

                    <div class="modal-body">
                        <Show when={!props.isNew}>
                            <div class="alert-banner">
                                IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!<br />
                                Someone could be eavesdropping on you right now (man-in-the-middle attack)!<br />
                                It is also possible that a host key has just been changed.
                            </div>
                        </Show>

                        <p>
                            The authenticity of host <strong>{props.hostName} ({props.hostAddress}:{props.port})</strong> can't be established.
                        </p>

                        <div class="key-details">
                            <div class="key-row">
                                <span class="label">Key Type:</span> {props.keyType}
                            </div>
                            <div class="key-row fingerprint-row highlight">
                                <span class="label">{props.isNew ? 'Fingerprint:' : 'New Fingerprint:'}</span>
                                <code>{props.fingerprint}</code>
                            </div>

                            <Show when={!props.isNew && props.previousFingerprint}>
                                <div class="key-row fingerprint-row old-key">
                                    <span class="label">Old Fingerprint:</span>
                                    <code>{props.previousFingerprint}</code>
                                </div>
                            </Show>
                        </div>

                        <p class="question">
                            Are you sure you want to continue connecting?
                        </p>
                    </div>

                    <div class="modal-footer">
                        <button class="action-btn" onClick={props.onReject}>
                            Reject and Cancel
                        </button>
                        <button class={`action-btn ${props.isNew ? 'primary' : 'danger'}`} onClick={props.onAccept}>
                            {props.isNew ? 'Accept and Save' : 'Accept Changed Key'}
                        </button>
                    </div>
                </div>
            </div>

            <style>{`
                .key-alert-modal {
                    max-width: 550px;
                }
                .key-alert-modal.danger-mode {
                    border: 2px solid var(--error);
                    box-shadow: 0 0 30px rgba(239, 68, 68, 0.2);
                }
                .key-alert-modal.danger-mode .modal-header h2 {
                    color: var(--error);
                }
                
                .alert-banner {
                    background: rgba(239, 68, 68, 0.1);
                    border-left: 4px solid var(--error);
                    padding: 12px;
                    color: var(--error);
                    font-family: var(--font-mono);
                    font-size: 13px;
                    line-height: 1.5;
                    margin-bottom: 20px;
                }
                
                .key-details {
                    background: var(--bg-tertiary);
                    border: 1px solid var(--border-primary);
                    border-radius: 6px;
                    padding: 16px;
                    margin: 20px 0;
                    font-family: var(--font-mono);
                    font-size: 13px;
                    display: flex;
                    flex-direction: column;
                    gap: 12px;
                }
                
                .key-row {
                    display: flex;
                    align-items: flex-start;
                    gap: 12px;
                }
                
                .key-row .label {
                    color: var(--text-muted);
                    width: 120px;
                    flex-shrink: 0;
                }
                
                .fingerprint-row code {
                    word-break: break-all;
                    color: var(--text-primary);
                }
                
                .fingerprint-row.highlight code {
                    color: var(--accent-primary);
                    font-weight: bold;
                }
                
                .fingerprint-row.old-key code {
                    color: var(--error);
                    text-decoration: line-through;
                }
                
                .question {
                    font-weight: 500;
                    margin-top: 20px;
                }
                
                .action-btn.danger {
                    background: var(--error);
                    color: #fff;
                    border: none;
                }
                .action-btn.danger:hover {
                    background: #dc2626; /* darker red */
                }
            `}</style>
        </Show>
    );
};
