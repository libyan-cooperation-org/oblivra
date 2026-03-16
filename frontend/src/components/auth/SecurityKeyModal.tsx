import { Component, createSignal, onMount, Show, For } from 'solid-js';
import {
    FIDO2ListCredentials,
    YubiKeyDetect,
    YubiKeyGenerateSSHKey,
    FIDO2RemoveCredential
} from '../../../wailsjs/go/services/SecurityService';
import { security } from '../../../wailsjs/go/models';

export const SecurityKeyModal: Component<{ onClose: () => void }> = (props) => {
    const [yubikeys, setYubikeys] = createSignal<security.YubiKeyInfo[]>([]);
    const [fidoHws, setFidoHws] = createSignal<security.FIDO2Credential[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    const loadDevices = async () => {
        setLoading(true);
        setError(null);
        try {
            const yk = await YubiKeyDetect();
            setYubikeys(yk || []);
            const fido = await FIDO2ListCredentials();
            setFidoHws(fido || []);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadDevices();
    });

    const handleGenerateKey = async (serial: string) => {
        setLoading(true);
        try {
            const pubKey = await YubiKeyGenerateSSHKey(serial, "9a", "000000"); // default pin for demo
            alert("Generated SSH Key: " + pubKey);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleRemoveFido = async (id: string) => {
        setLoading(true);
        try {
            await FIDO2RemoveCredential(id);
            await loadDevices();
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    return (
        <div class="modal-backdrop" style="position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000;">
            <div class="modal-content" style="background: var(--bg-primary); padding: 20px; border-radius: 8px; width: 500px; max-width: 90vw; border: 1px solid var(--border-primary);">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h2 style="margin: 0; font-size: 18px;">Hardware Security Keys</h2>
                    <button onClick={props.onClose} class="action-btn" style="border: none; background: transparent; cursor: pointer;">✕</button>
                </div>

                <Show when={error()}>
                    <div style="color: var(--error); margin-bottom: 10px; font-size: 13px;">{error()}</div>
                </Show>

                <div class="device-section" style="margin-bottom: 20px;">
                    <h3 style="font-size: 14px; margin-bottom: 10px; border-bottom: 1px solid var(--border-primary); padding-bottom: 5px;">YubiKeys (PIV)</h3>
                    <Show when={loading()} fallback={
                        <For each={yubikeys()}>
                            {(yk) => (
                                <div style="display: flex; justify-content: space-between; align-items: center; padding: 10px; border: 1px solid var(--border-primary); margin-bottom: 8px; border-radius: 4px;">
                                    <div>
                                        <div style="font-weight: bold;">Serial: {yk.serial}</div>
                                        <div style="font-size: 12px; color: var(--text-muted);">Version: {yk.version}</div>
                                    </div>
                                    <button onClick={() => handleGenerateKey(yk.serial)} class="action-btn primary" disabled={loading()}>
                                        Generate SSH Key
                                    </button>
                                </div>
                            )}
                        </For>
                    }>
                        <div style="font-size: 13px; color: var(--text-muted);">Scanning for devices...</div>
                    </Show>
                    {yubikeys().length === 0 && !loading() && <div style="font-size: 13px; color: var(--text-muted);">No YubiKeys detected.</div>}
                </div>

                <div class="device-section">
                    <h3 style="font-size: 14px; margin-bottom: 10px; border-bottom: 1px solid var(--border-primary); padding-bottom: 5px;">FIDO2 WebAuthn Credentials</h3>
                    <Show when={!loading()}>
                        <For each={fidoHws()}>
                            {(cred) => (
                                <div style="display: flex; justify-content: space-between; align-items: center; padding: 10px; border: 1px solid var(--border-primary); margin-bottom: 8px; border-radius: 4px;">
                                    <div>
                                        <div style="font-weight: bold;">{cred.device_name || 'FIDO2 Key'}</div>
                                        <div style="font-size: 12px; color: var(--text-muted);">Uses: {cred.sign_count} • Created: {new Date(cred.created_at.toString()).toLocaleDateString()}</div>
                                    </div>
                                    <button onClick={() => handleRemoveFido(cred.id)} class="action-btn" disabled={loading()} style="color: var(--error);">
                                        Remove
                                    </button>
                                </div>
                            )}
                        </For>
                    </Show>
                    {fidoHws().length === 0 && !loading() && <div style="font-size: 13px; color: var(--text-muted);">No FIDO2 credentials registered.</div>}
                </div>

                <div style="margin-top: 20px; text-align: right;">
                    <button onClick={loadDevices} class="action-btn" disabled={loading()} style="margin-right: 10px;">Refresh Devices</button>
                    <button onClick={props.onClose} class="action-btn primary">Done</button>
                </div>
            </div>
        </div>
    );
};
