import { Component, createSignal, onMount, Show, For } from 'solid-js';
import { GetTeamName, ListMembers, ListSecrets } from '../../../wailsjs/go/services/TeamService';
import {
    IsUnlocked, Unlock, ListCredentials, AddCredential,
    DeleteCredential, GenerateEd25519Key, GetDecryptedCredential, UnlockWithHardware
} from '../../../wailsjs/go/services/VaultService';
import { YubiKeyDetect, YubiKeyDeriveVaultKey } from '../../../wailsjs/go/services/SecurityService';
import { team, database } from '../../../wailsjs/go/models';
import '../../styles/vault.css';

export const VaultManager: Component = () => {
    const [unlocked, setUnlocked] = createSignal(false);
    const [password, setPassword] = createSignal("");
    const [teamName, setTeamName] = createSignal("Personal");
    const [members, setMembers] = createSignal<team.TeamMember[]>([]);
    const [secrets, setSecrets] = createSignal<team.VaultEntry[]>([]);
    const [credentials, setCredentials] = createSignal<database.Credential[]>([]);
    const [activeTab, setActiveTab] = createSignal<"secrets" | "members" | "credentials">("credentials");

    // UI State
    const [showAddCred, setShowAddCred] = createSignal(false);
    const [viewCred, setViewCred] = createSignal<{ label: string, data: string, type: string } | null>(null);
    const [newCredLabel, setNewCredLabel] = createSignal("");
    const [newCredData, setNewCredData] = createSignal("");
    const [newCredType, setNewCredType] = createSignal("ssh_key");
    const [generatedPubKey, setGeneratedPubKey] = createSignal("");
    const [isGenerating, setIsGenerating] = createSignal(false);

    const loadData = async () => {
        try {
            const isUn = await IsUnlocked();
            setUnlocked(isUn);
            if (!isUn) return;

            const name = await GetTeamName();
            setTeamName(name);
            const mems = await ListMembers();
            setMembers(mems || []);
            const secs = await ListSecrets();
            setSecrets(secs || []);
            const creds = await ListCredentials("");
            setCredentials(creds || []);
        } catch (e) {
            console.error("Failed to load vault data", e);
        }
    };

    const handleAddCredential = async () => {
        if (!newCredLabel() || !newCredData()) return;
        try {
            await AddCredential(newCredLabel(), newCredType(), newCredData());
            resetForm();
            await loadData();
        } catch (e) {
            alert("Failed to add credential: " + e);
        }
    };

    const handleGenerateKey = async () => {
        if (!newCredLabel()) {
            alert("Please provide a label first");
            return;
        }
        setIsGenerating(true);
        try {
            const pubKey = await GenerateEd25519Key(newCredLabel());
            setGeneratedPubKey(pubKey);
            await loadData();
        } catch (e) {
            alert("Failed to generate key: " + e);
        } finally {
            setIsGenerating(false);
        }
    };

    const handleViewCred = async (id: string) => {
        try {
            const cred = credentials().find(c => c.id === id);
            if (!cred) return;
            const data = await GetDecryptedCredential(id);
            setViewCred({ label: cred.label, data, type: cred.type });
        } catch (e) {
            alert("Failed to decrypt: " + e);
        }
    };

    const resetForm = () => {
        setNewCredLabel("");
        setNewCredData("");
        setGeneratedPubKey("");
        setShowAddCred(false);
    };

    const handleDeleteCredential = async (id: string) => {
        if (!confirm("Are you sure you want to delete this credential? This action is irreversible.")) return;
        try {
            await DeleteCredential(id);
            await loadData();
        } catch (e) {
            alert("Failed to delete: " + e);
        }
    };

    const handleFileUpload = (e: Event) => {
        const file = (e.target as HTMLInputElement).files?.[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = (ev) => {
            setNewCredData(ev.target?.result as string);
            if (!newCredLabel()) setNewCredLabel(file.name);
        };
        reader.readAsText(file);
    };

    const handleUnlock = async () => {
        try {
            await Unlock(password(), [], false);
            // SECURITY: Clear the password signal immediately after use so it
            // doesn't linger in JS heap memory once the vault is unlocked.
            setPassword("");
            setUnlocked(true);
            await loadData();
        } catch (e) {
            alert("Failed to unlock: " + e);
        }
    };

    const handleHardwareUnlock = async () => {
        if (!password()) {
            alert("Please enter your master password first to perform hardware challenge derivation.");
            return;
        }
        try {
            const keys = await YubiKeyDetect();
            if (!keys || keys.length === 0) {
                alert("No YubiKey detected. Please insert your hardware key.");
                return;
            }
            const hardwareKey = await YubiKeyDeriveVaultKey(keys[0].serial, password());
            // hardwareKey is a byte array (Uint8Array/number[])
            await UnlockWithHardware(password(), Array.from(hardwareKey), false);
            // SECURITY: Clear password signal immediately after hardware unlock
            setPassword("");
            setUnlocked(true);
            await loadData();
        } catch (e) {
            alert("Hardware unlock failed: " + e);
        }
    };

    const copyToClipboard = (text: string) => {
        navigator.clipboard.writeText(text);
        // Simple visual feedback could go here
    };

    onMount(loadData);

    return (
        <div class="vault-manager">
            <header class="vault-header">
                <h2>
                    <span>🗄️</span>
                    {unlocked() ? `Vault: ${teamName()}` : "Secure Vault Access"}
                </h2>
                <Show when={unlocked()}>
                    <button class="vault-tab" onClick={() => { /* Lock logic */ }}>Lock Vault</button>
                </Show>
            </header>

            <Show
                when={unlocked()}
                fallback={
                    <div class="vault-locked-container">
                        <div class="vault-locked-card">
                            <div class="vault-locked-icon">🔒</div>
                            <h3>Vault is Encrypted</h3>
                            <p style="color: var(--text-muted); margin-bottom: 24px;">
                                Enter your master password to decrypt your credentials and enterprise keys.
                            </p>
                            <div class="form-group">
                                <input
                                    type="password"
                                    placeholder="••••••••••••"
                                    value={password()}
                                    onInput={(e) => setPassword(e.currentTarget.value)}
                                    class="input-primary"
                                    style="text-align: center; font-size: 1.1rem;"
                                />
                            </div>
                            <button onClick={handleUnlock} class="action-btn primary" style="width: 100%; margin-top: 8px;">
                                Decrypt & Unlock
                            </button>
                            <div style="display: flex; align-items: center; gap: 8px; margin-top: 12px;">
                                <div style="height: 1px; background: var(--border-primary); flex: 1;"></div>
                                <span style="font-size: 10px; color: var(--text-muted); text-transform: uppercase;">Or use Hardware</span>
                                <div style="height: 1px; background: var(--border-primary); flex: 1;"></div>
                            </div>
                            <button onClick={handleHardwareUnlock} class="action-btn" style="width: 100%; margin-top: 12px; background: var(--bg-tertiary); border: 1px solid var(--accent-primary);">
                                🔑 Hardware Key Unlock (YubiKey)
                            </button>
                        </div>
                    </div>
                }
            >
                <div class="vault-tabs">
                    <button
                        class={`vault-tab ${activeTab() === 'credentials' ? 'active' : ''}`}
                        onClick={() => setActiveTab('credentials')}
                    >
                        Keys & Credentials
                    </button>
                    <button
                        class={`vault-tab ${activeTab() === 'secrets' ? 'active' : ''}`}
                        onClick={() => setActiveTab('secrets')}
                    >
                        Secrets
                    </button>
                    <button
                        class={`vault-tab ${activeTab() === 'members' ? 'active' : ''}`}
                        onClick={() => setActiveTab('members')}
                    >
                        Team Members
                    </button>
                </div>

                <div class="vault-content">
                    <Show when={activeTab() === 'credentials'}>
                        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                            <h3 style="margin: 0; font-size: 1.1rem;">Managed Credentials</h3>
                            <button class="action-btn primary" onClick={() => setShowAddCred(true)}>
                                ➕ Add Credential
                            </button>
                        </div>

                        <div class="credentials-grid">
                            <For each={credentials()}>
                                {(cred) => (
                                    <div class="vault-card">
                                        <div class="vault-card-info">
                                            <div class="vault-card-label">{cred.label}</div>
                                            <div class="vault-card-meta">
                                                <span>{cred.type === 'ssh_key' ? '🔑 SSH Key' : '🔒 Password'}</span>
                                                {cred.fingerprint && <span>• {cred.fingerprint}</span>}
                                            </div>
                                        </div>
                                        <div class="vault-card-actions">
                                            <button class="btn-icon" title="View/Edit" onClick={() => handleViewCred(cred.id)}>👁️</button>
                                            <button class="btn-icon delete" title="Delete" onClick={() => handleDeleteCredential(cred.id)}>🗑️</button>
                                        </div>
                                    </div>
                                )}
                            </For>
                        </div>
                        {credentials().length === 0 && (
                            <div style="text-align: center; padding: 48px; color: var(--text-muted);">
                                No credentials found. Add a key or password to get started.
                            </div>
                        )}
                    </Show>

                    {/* Other tabs remain largely the same but with classes */}
                    <Show when={activeTab() === 'secrets'}>
                        <div class="credentials-grid">
                            <For each={secrets()}>
                                {(secret) => (
                                    <div class="vault-card">
                                        <div class="vault-card-info">
                                            <div class="vault-card-label">{secret.title}</div>
                                            <div class="vault-card-meta">{secret.entry_type} • Linked</div>
                                        </div>
                                    </div>
                                )}
                            </For>
                        </div>
                    </Show>

                    <Show when={activeTab() === 'members'}>
                        <div class="credentials-grid">
                            <For each={members()}>
                                {(member) => (
                                    <div class="vault-card">
                                        <div class="vault-card-info">
                                            <div class="vault-card-label">{member.name}</div>
                                            <div class="vault-card-meta">{member.email}</div>
                                        </div>
                                        <div class="vault-card-meta" style="font-weight: bold; background: var(--bg-tertiary); padding: 4px 8px; border-radius: 4px;">
                                            {member.role}
                                        </div>
                                    </div>
                                )}
                            </For>
                        </div>
                    </Show>
                </div>

                {/* Modals */}
                <Show when={showAddCred()}>
                    <div class="modal-overlay" onClick={resetForm}>
                        <div class="modal-container" onClick={e => e.stopPropagation()}>
                            <div class="modal-header">
                                <h3 class="modal-title">Secure a New Credential</h3>
                                <button class="btn-icon" onClick={resetForm}>✕</button>
                            </div>

                            <div class="form-group">
                                <label>Identifier Label</label>
                                <input
                                    type="text"
                                    placeholder="e.g., Production Bastion Key"
                                    value={newCredLabel()}
                                    onInput={(e) => setNewCredLabel(e.currentTarget.value)}
                                    class="input-primary"
                                />
                            </div>

                            <div class="form-group">
                                <label>Credential Type</label>
                                <select
                                    value={newCredType()}
                                    onChange={(e) => setNewCredType(e.currentTarget.value)}
                                    class="input-primary"
                                >
                                    <option value="ssh_key">SSH Private Key (PEM)</option>
                                    <option value="password">Static Password / Token</option>
                                </select>
                            </div>

                            <div class="form-group">
                                <label>{newCredType() === 'ssh_key' ? "Private Key Content" : "Password Content"}</label>
                                <textarea
                                    placeholder={newCredType() === 'ssh_key' ? "-----BEGIN RSA PRIVATE KEY-----" : "Enter password..."}
                                    value={newCredData()}
                                    onInput={(e) => setNewCredData(e.currentTarget.value)}
                                    rows="6"
                                    class="input-primary"
                                    style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem;"
                                />
                            </div>

                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                <div style="font-size: 0.8rem; color: var(--text-muted);">
                                    <label style="cursor: pointer; color: var(--accent-primary);">
                                        📁 Upload File
                                        <input type="file" onChange={handleFileUpload} style="display: none;" />
                                    </label>
                                </div>
                                <div style="display: flex; gap: 8px;">
                                    <Show when={newCredType() === 'ssh_key'}>
                                        <button
                                            class="action-btn"
                                            onClick={handleGenerateKey}
                                            disabled={isGenerating()}
                                        >
                                            {isGenerating() ? "Generating..." : "⚡ Generate Ed25519"}
                                        </button>
                                    </Show>
                                    <button onClick={handleAddCredential} class="action-btn success">Secure Store</button>
                                </div>
                            </div>

                            <Show when={generatedPubKey()}>
                                <div class="pubkey-display">
                                    <label class="form-group" style="margin-bottom: 8px;">Your Public Key (SSH Authorized Key Format)</label>
                                    <code class="pubkey-text" onClick={() => copyToClipboard(generatedPubKey())}>
                                        {generatedPubKey()}
                                    </code>
                                    <span class="copy-hint">Click to copy to clipboard</span>
                                    <div style="margin-top: 12px; font-size: 0.75rem; color: #10b981;">
                                        ✅ Key generated and secured in vault.
                                    </div>
                                </div>
                            </Show>
                        </div>
                    </div>
                </Show>

                <Show when={viewCred()}>
                    <div class="modal-overlay" onClick={() => setViewCred(null)}>
                        <div class="modal-container" onClick={e => e.stopPropagation()}>
                            <div class="modal-header">
                                <h3 class="modal-title">Decrypted Credential: {viewCred()?.label}</h3>
                                <button class="btn-icon" onClick={() => setViewCred(null)}>✕</button>
                            </div>

                            <div class="form-group">
                                <label>{viewCred()?.type === 'ssh_key' ? "Private Key (Keep Secure)" : "Secret Password"}</label>
                                <textarea
                                    readOnly
                                    value={viewCred()?.data}
                                    rows="10"
                                    class="input-primary"
                                    style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem; background: var(--bg-tertiary);"
                                />
                            </div>

                            <div class="form-actions">
                                <button class="action-btn" onClick={() => copyToClipboard(viewCred()?.data || "")}>
                                    📋 Copy to Clipboard
                                </button>
                                <button class="action-btn primary" onClick={() => setViewCred(null)}>Close</button>
                            </div>
                        </div>
                    </div>
                </Show>
            </Show>
        </div>
    );
};
