import { Component, createSignal, onMount } from 'solid-js';
import { Unlock, ResetVault, IsUnlocked } from '../../../wailsjs/go/app/VaultService';

interface VaultUnlockProps {
    onUnlock: () => void;
}

export const VaultUnlock: Component<VaultUnlockProps> = (props) => {
    const [passphrase, setPassphrase] = createSignal('');
    const [remember, setRemember] = createSignal(true);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    // If vault is already unlocked (e.g. auto-unlock completed before we mounted),
    // skip the login screen entirely.
    onMount(async () => {
        const already = await IsUnlocked();
        if (already) props.onUnlock();
    });
    const [showReset, setShowReset] = createSignal(false);
    const [resetConfirm, setResetConfirm] = createSignal('');
    const [resetting, setResetting] = createSignal(false);

    const handleUnlock = async (e: Event) => {
        e.preventDefault();
        if (loading()) return;
        setLoading(true);
        setError(null);
        try {
            await Unlock(passphrase(), [], remember());
            props.onUnlock();
        } catch (err: unknown) {
            // Show the actual backend error message so the user knows what failed
            const msg = err instanceof Error ? err.message : String(err);
            setError(msg || 'Incorrect passphrase or vault error');
        } finally {
            setLoading(false);
        }
    };

    const handleReset = async () => {
        if (resetConfirm() !== 'RESET') return;
        setResetting(true);
        try {
            await ResetVault();
            // Reload the app — VaultGuard will now show VaultSetup
            window.location.reload();
        } catch (err) {
            setError(`Reset failed: ${err}`);
            setShowReset(false);
        } finally {
            setResetting(false);
        }
    };

    return (
        <div class="vault-gate">
            <div class="vault-gate-bg" />
            <div class="vault-gate-card">
                {/* Logo */}
                <div class="vault-gate-logo">
                    <div class="vault-gate-shield">🛡</div>
                    <div class="vault-gate-title">OBLIVRA</div>
                    <div class="vault-gate-subtitle">SOVEREIGN TERMINAL</div>
                </div>

                <div class="vault-gate-divider" />

                {!showReset() ? (
                    /* ── Normal unlock form ── */
                    <form class="vault-gate-form" onSubmit={handleUnlock}>
                        <label class="vault-gate-label">MASTER PASSPHRASE</label>
                        <input
                            type="password"
                            class="vault-gate-input"
                            value={passphrase()}
                            onInput={(e) => setPassphrase(e.currentTarget.value)}
                            placeholder="Enter your passphrase"
                            required
                            autofocus
                        />

                        {error() && <div class="vault-gate-error">{error()}</div>}

                        <div class="vault-gate-remember">
                            <label>
                                <input
                                    type="checkbox"
                                    checked={remember()}
                                    onChange={(e) => setRemember(e.currentTarget.checked)}
                                />
                                Remember for this session
                            </label>
                        </div>

                        <button type="submit" class="vault-gate-btn" disabled={loading()}>
                            {loading() ? <span class="vault-gate-spinner" /> : <>🔓 Access Vault</>}
                        </button>

                        {/* Escape hatch — visible but subtle */}
                        <button
                            type="button"
                            onClick={() => { setShowReset(true); setError(null); }}
                            style={{
                                background: 'none', border: 'none', cursor: 'pointer',
                                color: 'rgba(255,255,255,0.2)', 'font-size': '11px',
                                'font-family': 'var(--font-mono)', 'letter-spacing': '0.5px',
                                'text-align': 'center', 'margin-top': '4px',
                                transition: 'color 0.2s',
                            }}
                            onMouseOver={(e) => (e.currentTarget.style.color = 'rgba(248,81,73,0.6)')}
                            onMouseOut={(e) => (e.currentTarget.style.color = 'rgba(255,255,255,0.2)')}
                        >
                            Forgot passphrase?
                        </button>
                    </form>
                ) : (
                    /* ── Reset confirmation ── */
                    <div class="vault-gate-form">
                        <div style={{
                            background: 'rgba(248,81,73,0.08)', border: '1px solid rgba(248,81,73,0.3)',
                            'border-radius': '6px', padding: '14px', 'margin-bottom': '4px',
                            'font-size': '12px', color: 'rgba(248,81,73,0.9)', 'line-height': '1.6',
                        }}>
                            ⚠ <strong>This will permanently destroy your vault.</strong><br />
                            All stored credentials and SSH keys will be <strong>unrecoverable</strong>.<br />
                            Type <code style={{ background: 'rgba(255,255,255,0.08)', padding: '1px 4px', 'border-radius': '2px' }}>RESET</code> to confirm.
                        </div>

                        <input
                            type="text"
                            class="vault-gate-input"
                            placeholder="Type RESET to confirm"
                            value={resetConfirm()}
                            onInput={(e) => setResetConfirm(e.currentTarget.value)}
                            style={{ 'border-color': resetConfirm() === 'RESET' ? 'rgba(248,81,73,0.6)' : undefined }}
                            autofocus
                        />

                        <button
                            type="button"
                            class="vault-gate-btn"
                            disabled={resetConfirm() !== 'RESET' || resetting()}
                            onClick={handleReset}
                            style={{
                                background: resetConfirm() === 'RESET'
                                    ? 'linear-gradient(135deg, #dc2626, #991b1b)'
                                    : undefined,
                            }}
                        >
                            {resetting() ? <span class="vault-gate-spinner" /> : <>💥 Destroy & Reset Vault</>}
                        </button>

                        <button
                            type="button"
                            onClick={() => { setShowReset(false); setResetConfirm(''); }}
                            style={{
                                background: 'none', border: '1px solid rgba(255,255,255,0.1)',
                                'border-radius': '6px', padding: '10px', cursor: 'pointer',
                                color: 'rgba(255,255,255,0.4)', 'font-size': '13px',
                                'font-family': 'var(--font-ui)', transition: 'all 0.2s',
                            }}
                            onMouseOver={(e) => {
                                (e.currentTarget as HTMLElement).style.borderColor = 'rgba(255,255,255,0.3)';
                                (e.currentTarget as HTMLElement).style.color = 'rgba(255,255,255,0.7)';
                            }}
                            onMouseOut={(e) => {
                                (e.currentTarget as HTMLElement).style.borderColor = 'rgba(255,255,255,0.1)';
                                (e.currentTarget as HTMLElement).style.color = 'rgba(255,255,255,0.4)';
                            }}
                        >
                            ← Cancel
                        </button>
                    </div>
                )}

                <div class="vault-gate-footer">
                    AES-256-GCM · Ed25519 · Zero-Knowledge
                </div>
            </div>
        </div>
    );
};
