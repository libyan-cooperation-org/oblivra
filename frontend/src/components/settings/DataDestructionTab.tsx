import { Component, createSignal } from 'solid-js';
import { ClearDatabase } from '../../../wailsjs/go/services/SettingsService';

export const DataDestructionTab: Component = () => {
    const [confirmText, setConfirmText] = createSignal('');

    const handleWipe = async () => {
        if (confirmText() !== 'DESTROY') {
            alert('You must type DESTROY to confirm.');
            return;
        }
        if (confirm('CRITICAL WARNING: This completely wipes all local configurations, hosts, and data. Proceed?')) {
            try {
                await ClearDatabase();
                alert('Data destruction complete. Reloading...');
                window.location.reload();
            } catch (err) {
                alert('Destruction failed: ' + err);
            }
        }
    };

    return (
        <div class="brutalist-panel" style="border-color: var(--error);">
            <h2 class="brutalist-header" style="color: var(--error); border-color: var(--error);">Data Destruction</h2>

            <div class="status-block" style="border-color: var(--error); background: rgba(255,0,0,0.05);">
                <div>
                    <div style="font-weight: 900; font-size: 18px; margin-bottom: 8px; color: var(--error);">SCORCHED EARTH POLICY</div>
                    <div style="font-size: 13px; color: var(--text-primary); font-weight: 600; max-width: 500px; line-height: 1.5;">
                        Executing this routine will cryptographically wipe the local SQLite database container, zero out the vault key material in memory, and purge all settings.
                    </div>
                </div>
                <div class="status-badge" style="background: var(--error); color: #fff; border-color: var(--error);">DANGER</div>
            </div>

            <div style="margin-top: 32px; display: flex; flex-direction: column; gap: 16px; max-width: 400px;">
                <label style="font-size: 12px; font-weight: 800; text-transform: uppercase;">Type "DESTROY" to unlock</label>
                <input
                    type="text"
                    value={confirmText()}
                    onInput={(e) => setConfirmText(e.currentTarget.value)}
                    placeholder="DESTROY"
                    style="background: var(--bg-primary); border: 2px solid var(--error); color: var(--text-primary); padding: 12px; font-family: var(--font-mono); font-weight: bold; font-size: 16px; outline: none;"
                />

                <button
                    class="brutalist-btn"
                    style={`border-color: var(--error); color: ${confirmText() === 'DESTROY' ? '#fff' : 'var(--error)'}; background: ${confirmText() === 'DESTROY' ? 'var(--error)' : 'transparent'}; box-shadow: 4px 4px 0 ${confirmText() === 'DESTROY' ? 'rgba(255,0,0,0.3)' : 'var(--error)'}; opacity: ${confirmText() === 'DESTROY' ? '1' : '0.5'};`}
                    onClick={handleWipe}
                    disabled={confirmText() !== 'DESTROY'}
                >
                    Execute Protocol
                </button>
            </div>
        </div>
    );
};
