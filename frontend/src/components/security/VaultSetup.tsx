import { Component, createSignal, For } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface VaultSetupProps {
    onComplete: () => void;
}

export const VaultSetup: Component<VaultSetupProps> = (props) => {
    const [passphrase, setPassphrase] = createSignal("");
    const [confirm, setConfirm] = createSignal("");
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);
    const [useTPM, setUseTPM] = createSignal(false);
    const [pcr, setPCR] = createSignal(7);

    const getStrength = () => {
        const p = passphrase();
        if (!p) return 0;
        let score = 0;
        if (p.length > 8) score++;
        if (p.length > 12) score++;
        if (/[A-Z]/.test(p)) score++;
        if (/[0-9]/.test(p)) score++;
        if (/[^A-Za-z0-9]/.test(p)) score++;
        return score;
    };

    const strengthText = () => {
        const s = getStrength();
        return s < 2 ? 'Weak' : s < 4 ? 'Fair' : 'Strong';
    };

    const strengthColor = () => {
        const s = getStrength();
        return s < 2 ? '#f85149' : s < 4 ? '#d29922' : '#3fb950';
    };

    const handleSetup = async (e: Event) => {
        e.preventDefault();
        if (passphrase() !== confirm()) { setError('Passphrases do not match'); return; }
        if (passphrase().length < 8) { setError('Passphrase must be at least 8 characters'); return; }
        if (IS_BROWSER) return;
        setLoading(true); setError(null);
        try {
            const { SetupVault, SetupVaultWithTPM } = await import('../../../wailsjs/go/services/VaultService') as any;
            if (useTPM()) {
                await SetupVaultWithTPM(passphrase(), pcr());
            } else {
                await SetupVault(passphrase());
            }
            props.onComplete();
        } catch (err) { setError(String(err)); } finally { setLoading(false); }
    };

    return (
        <div class="vault-gate">
            <div class="vault-gate-bg" />
            <div class="vault-gate-card">
                <div class="vault-gate-logo">
                    <div class="vault-gate-shield">🛡</div>
                    <div class="vault-gate-title">OBLIVRA</div>
                    <div class="vault-gate-subtitle">INITIALIZE VAULT</div>
                </div>

                <div class="vault-gate-divider" />

                <form class="vault-gate-form" onSubmit={handleSetup}>
                    <label class="vault-gate-label">MASTER PASSPHRASE</label>
                    <input
                        type="password"
                        class="vault-gate-input"
                        value={passphrase()}
                        onInput={(e) => setPassphrase(e.currentTarget.value)}
                        placeholder="Choose a strong passphrase"
                        required autofocus
                    />

                    {/* Strength meter */}
                    <div class="vault-gate-strength">
                        <div class="vault-gate-strength-bar" style={{ width: `${(getStrength() / 5) * 100}%`, background: strengthColor() }} />
                    </div>
                    <div class="vault-gate-strength-label" style={{ color: strengthColor() }}>{strengthText()}</div>

                    <label class="vault-gate-label">CONFIRM PASSPHRASE</label>
                    <input
                        type="password"
                        class="vault-gate-input"
                        value={confirm()}
                        onInput={(e) => setConfirm(e.currentTarget.value)}
                        placeholder="Type it again"
                        required
                    />

                    <div class="vault-gate-options" style={{ margin: '12px 0', border: '1px solid rgba(255,255,255,0.1)', padding: '12px', 'border-radius': '4px' }}>
                        <label style={{ display: 'flex', 'align-items': 'center', gap: '8px', cursor: 'pointer', 'font-size': '12px', 'font-weight': '600', color: '#8b949e' }}>
                            <input type="checkbox" checked={useTPM()} onChange={(e) => setUseTPM(e.currentTarget.checked)} />
                            BIND TO HARDWARE (TPM 2.0)
                        </label>
                        {useTPM() && (
                            <div style={{ 'margin-top': '8px', display: 'flex', 'align-items': 'center', gap: '8px' }}>
                                <label style={{ 'font-size': '11px', color: '#8b949e' }}>PCR INDEX:</label>
                                <select
                                    class="ob-select"
                                    style={{ padding: '2px 4px', 'font-size': '11px', background: 'transparent', border: '1px solid rgba(255,255,255,0.2)', color: 'white' }}
                                    value={pcr()}
                                    onChange={(e) => setPCR(parseInt(e.currentTarget.value))}
                                >
                                    <For each={[0, 1, 7, 11, 23]}>{(i: number) => <option value={i}>PCR {i}</option>}</For>
                                </select>
                                <span style={{ 'font-size': '10px', color: '#d29922' }}>PCR 7 = Secure Boot State</span>
                            </div>
                        )}
                    </div>

                    {error() && <div class="vault-gate-error">{error()}</div>}

                    <button type="submit" class="vault-gate-btn" disabled={loading()}>
                        {loading() ? <span class="vault-gate-spinner" /> : <>🔐 Initialize Vault</>}
                    </button>
                </form>

                <div class="vault-gate-footer">
                    ⚠ If you lose this passphrase, encrypted data is unrecoverable.
                </div>
            </div>
        </div>
    );
};
