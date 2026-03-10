import { Component } from 'solid-js';

export const ErrorScreen: Component<{ title?: string; message: string; onRetry?: () => void }> = (props) => {
    return (
        <div class="error-screen-premium">
            <div class="mesh-background-error" />
            <div class="glass-card">
                <div class="error-icon">
                    <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 9v4m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 17c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                </div>
                <h1>{props.title || 'CRITICAL FAILURE'}</h1>
                <p class="error-message">{props.message}</p>
                <div class="actions">
                    <button class="retry-btn" onClick={() => props.onRetry?.() || window.location.reload()}>
                        REBOOT SYSTEM
                    </button>
                    <p class="hint">Check terminal logs for trace details</p>
                </div>
            </div>

            <style>{`
                .error-screen-premium {
                    position: fixed;
                    inset: 0;
                    background: #0a0505;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 10000;
                }

                .mesh-background-error {
                    position: absolute;
                    inset: 0;
                    background: radial-gradient(circle at 50% 50%, rgba(255, 77, 77, 0.1) 0%, transparent 70%);
                    animation: pulse-error 4s ease-in-out infinite;
                }

                .glass-card {
                    background: rgba(20, 10, 10, 0.6);
                    backdrop-filter: blur(20px) saturate(180%);
                    border: 1px solid rgba(255, 77, 77, 0.2);
                    padding: 48px;
                    border-radius: 24px;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 24px;
                    max-width: 480px;
                    text-align: center;
                    box-shadow: 0 24px 48px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(255, 77, 77, 0.1);
                    animation: scale-up 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
                }

                .error-icon {
                    color: var(--error);
                    filter: drop-shadow(0 0 10px var(--error-bg));
                }

                h1 {
                    font-size: 20px;
                    font-weight: 800;
                    letter-spacing: 2px;
                    color: #fff;
                    margin: 0;
                }

                .error-message {
                    color: var(--text-secondary);
                    font-family: var(--font-mono);
                    font-size: 14px;
                    line-height: 1.6;
                    background: rgba(0, 0, 0, 0.2);
                    padding: 16px;
                    border-radius: 8px;
                    border: 1px solid rgba(255, 255, 255, 0.05);
                }

                .retry-btn {
                    background: var(--error);
                    color: #fff;
                    border: none;
                    padding: 12px 32px;
                    border-radius: 8px;
                    font-weight: 700;
                    font-size: 13px;
                    letter-spacing: 1px;
                    cursor: pointer;
                    transition: all 0.2s;
                    box-shadow: 0 4px 12px rgba(255, 77, 77, 0.3);
                }

                .retry-btn:hover {
                    background: #ff6666;
                    transform: translateY(-2px);
                    box-shadow: 0 8px 20px rgba(255, 77, 77, 0.4);
                }

                .hint {
                    margin-top: 12px;
                    font-size: 11px;
                    color: var(--text-muted);
                    text-transform: uppercase;
                    letter-spacing: 1px;
                }

                @keyframes pulse-error {
                    0%, 100% { opacity: 0.5; }
                    50% { opacity: 1; }
                }

                @keyframes scale-up {
                    from { opacity: 0; transform: scale(0.9) translateY(20px); }
                    to { opacity: 1; transform: scale(1) translateY(0); }
                }
            `}</style>
        </div>
    );
};
