import { Component, createSignal, onMount, Show, For } from 'solid-js';
import { SSHListCertificates } from '../../../wailsjs/go/services/SecurityService';
import { ssh } from '../../../wailsjs/go/models';

export const CertificateManager: Component = () => {
    const [certs, setCerts] = createSignal<ssh.CertificateInfo[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    const loadCerts = async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await SSHListCertificates();
            setCerts(data || []);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadCerts();
    });

    const isExpired = (cert: ssh.CertificateInfo) => {
        return new Date(cert.valid_before.toString()) < new Date();
    };

    return (
        <div class="certificate-manager" style="padding: 15px;">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px;">
                <h3 style="margin: 0;">SSH Certificates</h3>
                <button onClick={loadCerts} class="action-btn" disabled={loading()}>
                    Refresh
                </button>
            </div>

            <Show when={error()}>
                <div style="color: var(--error); margin-bottom: 15px; font-size: 13px;">{error()}</div>
            </Show>

            <Show when={loading()} fallback={
                <div class="cert-list">
                    <For each={certs()}>
                        {(cert) => (
                            <div style={`padding: 15px; border: 1px solid ${isExpired(cert) ? 'var(--error)' : 'var(--border-primary)'}; margin-bottom: 10px; border-radius: 6px; background: var(--bg-secondary);`}>
                                <div style="display: flex; justify-content: space-between; margin-bottom: 8px;">
                                    <span style="font-weight: bold; font-size: 14px;">{cert.key_id}</span>
                                    <span style={`font-size: 12px; font-weight: bold; padding: 2px 8px; border-radius: 12px; background: ${isExpired(cert) ? 'var(--error)' : 'var(--success)'}; color: #fff;`}>
                                        {isExpired(cert) ? 'EXPIRED' : 'VALID'}
                                    </span>
                                </div>
                                <div style="font-size: 13px; color: var(--text-muted); display: grid; grid-template-columns: 100px 1fr; gap: 4px;">
                                    <strong>Principals:</strong> <span>{cert.valid_principals.join(', ') || '*'}</span>
                                    <strong>Valid After:</strong> <span>{new Date(cert.valid_after.toString()).toLocaleString()}</span>
                                    <strong>Valid Until:</strong> <span>{new Date(cert.valid_before.toString()).toLocaleString()}</span>
                                    <strong>Extensions:</strong> <span>{Object.keys(cert.extensions || {}).join(', ') || 'None'}</span>
                                </div>
                            </div>
                        )}
                    </For>
                    {certs().length === 0 && <div style="color: var(--text-muted); font-size: 13px; text-align: center; padding: 20px;">No SSH certificates found in identity agent or files.</div>}
                </div>
            }>
                <div style="color: var(--text-muted); font-size: 13px;">Loading certificates...</div>
            </Show>
        </div>
    );
};
