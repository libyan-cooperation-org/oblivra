import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const SecurityPanel: Component = () => {
    const [tab, setTab] = createSignal<'certs' | 'fido2' | 'yubikey'>('certs');
    const [certs, setCerts] = createSignal<any[]>([]);
    const [fido2Creds, setFido2Creds] = createSignal<any[]>([]);
    const [yubikeys, setYubikeys] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    onMount(async () => {
        try {
            const { SSHListCertificates, FIDO2ListCredentials, YubiKeyDetect } = await import('../../../wailsjs/go/services/SecurityService');
            const [c, f, y] = await Promise.allSettled([SSHListCertificates(), FIDO2ListCredentials(), YubiKeyDetect()]);
            setCerts(c.status === 'fulfilled' ? (c.value || []) : []);
            setFido2Creds(f.status === 'fulfilled' ? (f.value || []) : []);
            setYubikeys(y.status === 'fulfilled' ? (y.value || []) : []);
        } catch (e) { console.error('Security load:', e); }
        setLoading(false);
    });

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div style="display: flex; gap: 2px; padding: 8px 12px; border-bottom: 1px solid var(--border-primary);">
                <button class={`header-tab ${tab() === 'certs' ? 'active' : ''}`} onClick={() => setTab('certs')} style="font-size: 11px; padding: 4px 8px;">🔑 SSH Certs</button>
                <button class={`header-tab ${tab() === 'fido2' ? 'active' : ''}`} onClick={() => setTab('fido2')} style="font-size: 11px; padding: 4px 8px;">🛡 FIDO2</button>
                <button class={`header-tab ${tab() === 'yubikey' ? 'active' : ''}`} onClick={() => setTab('yubikey')} style="font-size: 11px; padding: 4px 8px;">🔐 YubiKey</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading() && tab() === 'certs'}>
                    <For each={certs()} fallback={<div class="placeholder">No SSH certificates found</div>}>
                        {(cert) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 8px; margin-bottom: 4px;">
                                <div style="font-size: 12px; color: var(--text-primary);">🔑 {cert.key_id || cert.serial || 'Certificate'}</div>
                                <div style="font-size: 10px; color: var(--text-muted); margin-top: 2px;">{cert.type || ''} • Expires: {cert.valid_before ? new Date(cert.valid_before).toLocaleDateString() : 'Unknown'}</div>
                            </div>
                        )}
                    </For>
                </Show>
                <Show when={!loading() && tab() === 'fido2'}>
                    <For each={fido2Creds()} fallback={<div class="placeholder">No FIDO2 credentials registered</div>}>
                        {(cred) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 8px; margin-bottom: 4px;">
                                <div style="font-size: 12px; color: var(--text-primary);">🛡 {cred.name || cred.id}</div>
                                <div style="font-size: 10px; color: var(--text-muted);">{cred.created_at ? new Date(cred.created_at).toLocaleDateString() : ''}</div>
                            </div>
                        )}
                    </For>
                </Show>
                <Show when={!loading() && tab() === 'yubikey'}>
                    <For each={yubikeys()} fallback={<div class="placeholder">No YubiKey devices detected</div>}>
                        {(yk) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 8px; margin-bottom: 4px;">
                                <div style="font-size: 12px; color: var(--text-primary);">🔐 YubiKey {yk.serial || ''}</div>
                                <div style="font-size: 10px; color: var(--text-muted);">Version: {yk.version || 'Unknown'} • Slots: {yk.slots_used || 0}</div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
