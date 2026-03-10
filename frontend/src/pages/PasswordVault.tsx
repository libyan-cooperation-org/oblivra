import { Component, createSignal, onMount, For, Show } from 'solid-js';

interface Credential {
    id: string;
    label: string;
    type: string;
    created_at: string;
    updated_at: string;
}

export const PasswordVault: Component = () => {
    const [credentials, setCredentials] = createSignal<Credential[]>([]);
    const [filter, setFilter] = createSignal('');
    const [typeFilter, setTypeFilter] = createSignal('');
    const [showAdd, setShowAdd] = createSignal(false);
    const [newLabel, setNewLabel] = createSignal('');
    const [newType, setNewType] = createSignal('password');
    const [newData, setNewData] = createSignal('');
    const [generatedPass, setGeneratedPass] = createSignal('');
    const [passLength, setPassLength] = createSignal(20);
    const [includeSymbols, setIncludeSymbols] = createSignal(true);
    const [revealedId, setRevealedId] = createSignal<string | null>(null);
    const [revealedData, setRevealedData] = createSignal('');
    const [copied, setCopied] = createSignal<string | null>(null);

    const loadCredentials = async () => {
        try {
            // @ts-ignore
            const creds = await window.go?.app?.VaultService?.ListCredentials(typeFilter());
            if (creds) setCredentials(creds);
        } catch (e) {
            console.error('[VAULT] Failed to load credentials:', e);
            setCredentials([
                { id: '1', label: 'Production DB (Postgres)', type: 'password', created_at: '2026-02-15', updated_at: '2026-03-01' },
                { id: '2', label: 'AWS IAM Root Key', type: 'api_key', created_at: '2026-01-20', updated_at: '2026-02-28' },
                { id: '3', label: 'Deploy SSH Key', type: 'key', created_at: '2026-01-05', updated_at: '2026-01-05' },
                { id: '4', label: 'Slack Webhook Token', type: 'token', created_at: '2026-02-10', updated_at: '2026-02-25' },
                { id: '5', label: 'SAML SP Certificate', type: 'certificate', created_at: '2026-03-01', updated_at: '2026-03-01' },
            ]);
        }
    };

    onMount(loadCredentials);

    const handleGenerate = async () => {
        try {
            // @ts-ignore
            const pass = await window.go?.app?.VaultService?.GeneratePassword(passLength(), includeSymbols());
            if (pass) { setGeneratedPass(pass); setNewData(pass); }
        } catch {
            const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789' + (includeSymbols() ? '!@#$%^&*()-_=+[]{}|;:,.<>?' : '');
            const arr = new Uint8Array(passLength());
            crypto.getRandomValues(arr);
            const pass = Array.from(arr, b => chars[b % chars.length]).join('');
            setGeneratedPass(pass); setNewData(pass);
        }
    };

    const handleAdd = async () => {
        if (!newLabel() || !newData()) return;
        try {
            // @ts-ignore
            await window.go?.app?.VaultService?.AddCredential(newLabel(), newType(), newData());
            setShowAdd(false); setNewLabel(''); setNewType('password'); setNewData(''); setGeneratedPass('');
            await loadCredentials();
        } catch (e) { console.error('[VAULT] Add failed:', e); }
    };

    const handleReveal = async (id: string) => {
        if (revealedId() === id) { setRevealedId(null); setRevealedData(''); return; }
        try {
            // @ts-ignore
            const data = await window.go?.app?.VaultService?.GetDecryptedCredential(id);
            if (data) { setRevealedId(id); setRevealedData(data); }
        } catch (e) { console.error('[VAULT] Decrypt failed:', e); }
    };

    const handleCopy = async (id: string) => {
        try {
            // @ts-ignore
            const data = await window.go?.app?.VaultService?.GetDecryptedCredential(id);
            if (data) { await navigator.clipboard.writeText(data); setCopied(id); setTimeout(() => setCopied(null), 2000); }
        } catch (e) { console.error('[VAULT] Copy failed:', e); }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Permanently delete this credential?')) return;
        try {
            // @ts-ignore
            await window.go?.app?.VaultService?.DeleteCredential(id);
            await loadCredentials();
        } catch (e) { console.error('[VAULT] Delete failed:', e); }
    };

    const typeIcon = (t: string) => {
        switch (t) {
            case 'password': return '🔑';
            case 'key': return '🔐';
            case 'api_key': return '🗝️';
            case 'token': return '🎫';
            case 'certificate': return '📜';
            default: return '🔒';
        }
    };

    const filteredCreds = () => {
        let creds = credentials();
        const q = filter().toLowerCase();
        if (q) creds = creds.filter(c => c.label.toLowerCase().includes(q) || c.type.toLowerCase().includes(q));
        return creds;
    };

    const strength = (pass: string) => {
        let score = 0;
        if (pass.length >= 12) score++;
        if (pass.length >= 20) score++;
        if (/[A-Z]/.test(pass)) score++;
        if (/[a-z]/.test(pass)) score++;
        if (/[0-9]/.test(pass)) score++;
        if (/[^A-Za-z0-9]/.test(pass)) score++;
        return score >= 5 ? 'EXCELLENT' : score >= 3 ? 'GOOD' : 'WEAK';
    };

    const strengthColor = (s: string) => s === 'EXCELLENT' ? '#3fb950' : s === 'GOOD' ? '#d29922' : '#f85149';

    return (
        <div class="ob-page">
            {/* Header */}
            <div class="ob-page-header">
                <div>
                    <h1 class="ob-page-title">Credential Vault</h1>
                    <div class="ob-page-subtitle">AES-256-GCM Encrypted · Zero-Knowledge · Air-Gap Safe</div>
                </div>
                <button class="ob-btn ob-btn-primary" onClick={() => setShowAdd(true)}>+ Add Credential</button>
            </div>

            {/* Search & Filter */}
            <div class="ob-toolbar">
                <input type="text" placeholder="Search credentials..." value={filter()} onInput={e => setFilter(e.currentTarget.value)} class="ob-input" style={{ flex: '1' }} />
                <select value={typeFilter()} onChange={e => { setTypeFilter(e.currentTarget.value); loadCredentials(); }} class="ob-select" style={{ width: '160px' }}>
                    <option value="">All Types</option>
                    <option value="password">Passwords</option>
                    <option value="key">SSH Keys</option>
                    <option value="api_key">API Keys</option>
                    <option value="token">Tokens</option>
                    <option value="certificate">Certificates</option>
                </select>
            </div>

            {/* Credential Table */}
            <div class="ob-table-wrap">
                <table class="ob-table">
                    <thead>
                        <tr>
                            <th style={{ width: '40px' }}></th>
                            <th>Label</th>
                            <th>Type</th>
                            <th>Created</th>
                            <th>Updated</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={filteredCreds()}>
                            {(cred) => (
                                <>
                                    <tr>
                                        <td style={{ 'font-size': '18px', 'text-align': 'center' }}>{typeIcon(cred.type)}</td>
                                        <td style={{ 'font-weight': '600' }}>{cred.label}</td>
                                        <td><span class="ob-badge">{cred.type.replace(/_/g, ' ')}</span></td>
                                        <td class="ob-text-muted">{cred.created_at?.substring(0, 10)}</td>
                                        <td class="ob-text-muted">{cred.updated_at?.substring(0, 10)}</td>
                                        <td>
                                            <div class="ob-toolbar" style={{ 'margin-bottom': '0', gap: '4px' }}>
                                                <button class="ob-btn ob-btn-sm ob-btn-ghost" onClick={() => handleCopy(cred.id)} title="Copy to clipboard">
                                                    {copied() === cred.id ? '✓' : '📋'}
                                                </button>
                                                <button class="ob-btn ob-btn-sm ob-btn-ghost" onClick={() => handleReveal(cred.id)} title="Reveal/Hide">
                                                    {revealedId() === cred.id ? '🙈' : '👁️'}
                                                </button>
                                                <button class="ob-btn ob-btn-sm ob-btn-danger" onClick={() => handleDelete(cred.id)} title="Delete">🗑</button>
                                            </div>
                                        </td>
                                    </tr>
                                    <Show when={revealedId() === cred.id}>
                                        <tr>
                                            <td colSpan={6}>
                                                <pre class="ob-code" style={{ color: '#d29922', border: '1px solid rgba(210,153,34,0.3)' }}>
                                                    {revealedData()}
                                                </pre>
                                            </td>
                                        </tr>
                                    </Show>
                                </>
                            )}
                        </For>
                    </tbody>
                </table>

                <Show when={filteredCreds().length === 0}>
                    <div class="ob-empty">
                        <div class="ob-empty-icon">🔒</div>
                        <div class="ob-empty-title">No Credentials Found</div>
                        <div class="ob-empty-desc">Click <strong>+ Add Credential</strong> to store your first secret.</div>
                    </div>
                </Show>
            </div>

            {/* Stats */}
            <div class="ob-stat-grid ob-stat-grid-5" style={{ 'margin-top': '16px' }}>
                {[
                    { label: 'Total', value: credentials().length },
                    { label: 'Passwords', value: credentials().filter(c => c.type === 'password').length },
                    { label: 'Keys', value: credentials().filter(c => c.type === 'key').length },
                    { label: 'API Keys', value: credentials().filter(c => c.type === 'api_key').length },
                    { label: 'Tokens', value: credentials().filter(c => c.type === 'token').length },
                ].map(s => (
                    <div class="ob-kpi">
                        <div class="ob-kpi-value">{s.value}</div>
                        <div class="ob-kpi-label">{s.label}</div>
                    </div>
                ))}
            </div>

            {/* Add Modal */}
            <Show when={showAdd()}>
                <div class="ob-modal-overlay" onClick={() => setShowAdd(false)}>
                    <div class="ob-modal" onClick={(e) => e.stopPropagation()}>
                        <h2>Add Credential</h2>
                        <div class="ob-form">
                            <label>Label</label>
                            <input class="ob-input" placeholder="e.g. Production Database" value={newLabel()} onInput={e => setNewLabel(e.currentTarget.value)} />

                            <label>Type</label>
                            <select class="ob-select" value={newType()} onChange={e => setNewType(e.currentTarget.value)}>
                                <option value="password">Password</option>
                                <option value="key">SSH Key</option>
                                <option value="api_key">API Key</option>
                                <option value="token">Token</option>
                                <option value="certificate">Certificate</option>
                            </select>

                            <label>Secret Data</label>
                            <textarea class="ob-input ob-textarea" rows={4} placeholder="Enter the secret value..." value={newData()} onInput={e => setNewData(e.currentTarget.value)} />

                            <Show when={newType() === 'password'}>
                                <div class="ob-card">
                                    <div class="ob-section-header">Password Generator</div>
                                    <div class="ob-toolbar" style={{ 'margin-bottom': '8px' }}>
                                        <label style={{ 'font-size': '12px', 'white-space': 'nowrap' }}>Length: {passLength()}</label>
                                        <input type="range" min="8" max="64" value={passLength()} onInput={e => setPassLength(parseInt(e.currentTarget.value))} style={{ flex: '1' }} />
                                        <label style={{ 'font-size': '12px', display: 'flex', gap: '4px', 'align-items': 'center' }}>
                                            <input type="checkbox" checked={includeSymbols()} onChange={e => setIncludeSymbols(e.currentTarget.checked)} /> Symbols
                                        </label>
                                        <button class="ob-btn ob-btn-sm" onClick={handleGenerate}>⚡ Generate</button>
                                    </div>
                                    <Show when={generatedPass()}>
                                        <pre class="ob-code">{generatedPass()}</pre>
                                        <div style={{ 'font-size': '11px', 'margin-top': '6px' }}>
                                            Strength: <span style={{ color: strengthColor(strength(generatedPass())), 'font-weight': '700' }}>{strength(generatedPass())}</span>
                                        </div>
                                    </Show>
                                </div>
                            </Show>

                            <div class="ob-modal-actions">
                                <button class="ob-btn" onClick={() => setShowAdd(false)}>Cancel</button>
                                <button class="ob-btn ob-btn-primary" onClick={handleAdd} disabled={!newLabel() || !newData()}>🔒 Encrypt & Save</button>
                            </div>
                        </div>
                    </div>
                </div>
            </Show>
        </div>
    );
};
