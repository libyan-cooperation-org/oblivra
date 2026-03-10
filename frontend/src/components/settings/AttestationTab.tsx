import { Component } from 'solid-js';

export const AttestationTab: Component = () => {
    return (
        <div class="brutalist-panel">
            <style>{`
                .brutalist-panel {
                    border: 3px solid var(--text-primary);
                    padding: 24px;
                    background: var(--bg-surface);
                    font-family: var(--font-mono);
                }
                .brutalist-header {
                    text-transform: uppercase;
                    font-size: 20px;
                    font-weight: 900;
                    letter-spacing: -1px;
                    border-bottom: 3px solid var(--text-primary);
                    padding-bottom: 12px;
                    margin-bottom: 24px;
                    margin-top: 0;
                    color: var(--text-primary);
                }
                .status-block {
                    border: 2px solid var(--text-primary);
                    padding: 16px;
                    margin-bottom: 16px;
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    background: var(--bg-primary);
                }
                .status-badge {
                    padding: 6px 14px;
                    font-weight: 800;
                    text-transform: uppercase;
                    border: 2px solid var(--text-primary);
                    font-size: 13px;
                }
                .status-badge.trusted {
                    background: var(--success);
                    color: #000;
                    border-color: var(--success);
                }
                .status-badge.verifying {
                    background: var(--warning);
                    color: #000;
                    border-color: var(--warning);
                }
                .brutalist-btn {
                    background: transparent;
                    border: 3px solid var(--text-primary);
                    color: var(--text-primary);
                    padding: 10px 20px;
                    font-family: inherit;
                    font-weight: 800;
                    text-transform: uppercase;
                    cursor: pointer;
                    transition: all 0.1s;
                    box-shadow: 4px 4px 0 var(--text-primary);
                }
                .brutalist-btn:hover {
                    transform: translate(2px, 2px);
                    box-shadow: 2px 2px 0 var(--text-primary);
                }
                .brutalist-btn:active {
                    transform: translate(4px, 4px);
                    box-shadow: none;
                }
            `}</style>

            <h2 class="brutalist-header">Runtime Attestation</h2>

            <div class="status-block">
                <div>
                    <div style="font-weight: 800; font-size: 16px; margin-bottom: 4px;">Primary Bootloader (TPM PCR 0)</div>
                    <div style="font-size: 12px; color: var(--text-muted); font-weight: 600;">Last verified: 2 minutes ago</div>
                </div>
                <div class="status-badge trusted">Trusted</div>
            </div>

            <div class="status-block">
                <div>
                    <div style="font-weight: 800; font-size: 16px; margin-bottom: 4px;">Kernel Integrity (TPM PCR 4)</div>
                    <div style="font-size: 12px; color: var(--text-muted); font-weight: 600;">Last verified: 2 minutes ago</div>
                </div>
                <div class="status-badge trusted">Trusted</div>
            </div>

            <div class="status-block">
                <div>
                    <div style="font-weight: 800; font-size: 16px; margin-bottom: 4px;">Sovereign Enclave (PCR 12)</div>
                    <div style="font-size: 12px; color: var(--text-muted); font-weight: 600;">Validation pending baseline anchor</div>
                </div>
                <div class="status-badge verifying">Verifying</div>
            </div>

            <div style="margin-top: 32px;">
                <button class="brutalist-btn" onClick={() => alert('Initiating remote attestation challenge...')}>
                    Force Attestation Challenge
                </button>
            </div>
        </div>
    );
};
