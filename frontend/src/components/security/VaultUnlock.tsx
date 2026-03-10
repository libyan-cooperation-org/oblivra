import { Component, createSignal } from 'solid-js';
import { Unlock } from '../../../wailsjs/go/app/VaultService';

interface VaultUnlockProps {
    onUnlock: () => void;
}

export const VaultUnlock: Component<VaultUnlockProps> = (props) => {
    const [passphrase, setPassphrase] = createSignal("");
    const [remember, setRemember] = createSignal(true);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    const handleUnlock = async (e: Event) => {
        e.preventDefault();
        setLoading(true);
        setError(null);
        try {
            await Unlock(passphrase(), [], remember());
            props.onUnlock();
        } catch (err: unknown) {
            setError("Incorrect passphrase or vault error");
            console.error(err);
        } finally {
            setLoading(false);
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

                {/* Separator */}
                <div class="vault-gate-divider" />

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
                        {loading() ? (
                            <span class="vault-gate-spinner" />
                        ) : (
                            <>🔓 Access Vault</>
                        )}
                    </button>
                </form>

                <div class="vault-gate-footer">
                    AES-256-GCM · Ed25519 · Zero-Knowledge
                </div>
            </div>
        </div>
    );
};
