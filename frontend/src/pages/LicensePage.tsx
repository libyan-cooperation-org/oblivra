import { Component, createSignal, onMount, Show, For } from 'solid-js';
import { IS_BROWSER } from '@core/context';

// ── Types ─────────────────────────────────────────────────────────────────────

interface LicenseStatus {
    tier: string;
    tier_int: number;
    is_licensed: boolean;
    is_expired: boolean;
    summary: string;
    licensee?: string;
    license_id?: string;
    max_seats?: number;
    max_agents?: number;
    expires_at?: number | null;
    active_agents?: number;
    agents_at_limit?: boolean;
}

const TIER_COLORS: Record<string, string> = {
    Community:    '#6b7280',
    Professional: '#3b82f6',
    Enterprise:   '#8b5cf6',
    Sovereign:    '#10b981',
};

const TIER_DESCRIPTIONS: Record<string, string> = {
    Community:    'Core SSH terminal, vault basics, and snippets — free forever.',
    Professional: 'Full SIEM, agents, AI assistant, recordings, and dashboard analytics.',
    Enterprise:   'UEBA, NDR, SOAR, compliance packs, identity federation, and cluster HA.',
    Sovereign:    'Every capability — war mode, ransomware defense, formal verification, and offline ops.',
};

// ── Feature matrix ─────────────────────────────────────────────────────────────

const FEATURE_GROUPS = [
    {
        label: 'Core Terminal',
        tier: 'Community',
        features: ['ssh', 'terminal', 'vault_basic', 'snippets', 'notes'],
    },
    {
        label: 'Professional',
        tier: 'Professional',
        features: ['siem', 'alerts', 'agents', 'vault_full', 'recordings',
                   'transfers', 'tunnels', 'multi_exec', 'ai_assistant',
                   'plugins', 'dashboard', 'health', 'metrics', 'topology'],
    },
    {
        label: 'Enterprise',
        tier: 'Enterprise',
        features: ['ueba', 'ndr', 'soar', 'purple_team', 'compliance',
                   'forensics', 'identity', 'graph', 'team', 'sync',
                   'threat_hunt', 'soc', 'risk', 'governance', 'executive', 'cluster'],
    },
    {
        label: 'Sovereign',
        tier: 'Sovereign',
        features: ['war_mode', 'ransomware_defense', 'simulation', 'temporal_integrity',
                   'data_lineage', 'decision_log', 'evidence_ledger', 'response_replay',
                   'counterfactual', 'deterministic_exec', 'memory_security',
                   'disaster_recovery', 'offline_updates'],
    },
];

const FEATURE_LABELS: Record<string, string> = {
    ssh: 'SSH Client', terminal: 'Local Terminal', vault_basic: 'Vault (Basic)',
    snippets: 'Snippet Library', notes: 'Notes & Runbooks',
    siem: 'SIEM Engine', alerts: 'Alerting & Detection', agents: 'Remote Agents',
    vault_full: 'Vault (Full)', recordings: 'Session Recordings',
    transfers: 'File Transfers', tunnels: 'SSH Tunnels', multi_exec: 'Multi-Exec',
    ai_assistant: 'AI Assistant', plugins: 'Plugin Framework',
    dashboard: 'Analytics Dashboard', health: 'Health Monitoring',
    metrics: 'Prometheus Metrics', topology: 'Network Topology',
    ueba: 'UEBA / Behavioral AI', ndr: 'NDR / Network Detection',
    soar: 'SOAR Playbooks', purple_team: 'Purple Team',
    compliance: 'Compliance Packs', forensics: 'Digital Forensics',
    identity: 'Identity & RBAC', graph: 'Threat Graph',
    team: 'Team Collaboration', sync: 'Multi-Node Sync',
    threat_hunt: 'Threat Hunter', soc: 'SOC Workspace',
    risk: 'Risk Scoring', governance: 'Governance Engine',
    executive: 'Executive Dashboard', cluster: 'HA Cluster (Raft)',
    war_mode: 'War Mode', ransomware_defense: 'Ransomware Defense',
    simulation: 'Attack Simulation', temporal_integrity: 'Temporal Integrity',
    data_lineage: 'Data Lineage', decision_log: 'Decision Log',
    evidence_ledger: 'Evidence Ledger', response_replay: 'Response Replay',
    counterfactual: 'Counterfactual Analysis', deterministic_exec: 'Deterministic Exec',
    memory_security: 'Memory Security', disaster_recovery: 'Disaster Recovery',
    offline_updates: 'Offline Updates',
};

// ── Component ─────────────────────────────────────────────────────────────────

export const LicensePage: Component = () => {
    const [status, setStatus] = createSignal<LicenseStatus | null>(null);
    const [featureMap, setFeatureMap] = createSignal<Record<string, boolean>>({});
    const [token, setToken] = createSignal('');
    const [activating, setActivating] = createSignal(false);
    const [error, setError] = createSignal('');
    const [success, setSuccess] = createSignal('');

    const refresh = async () => {
        if (IS_BROWSER) return;
        try {
            const LS = await import('../../wailsjs/go/services/LicensingService');
            const [s, fm] = await Promise.all([(LS as any).GetLicenseStatus(), (LS as any).GetFeatureMap()]);
            setStatus(s);
            setFeatureMap(fm || {});
        } catch (e) {
            console.error('License status fetch failed:', e);
        }
    };

    onMount(refresh);

    const activate = async () => {
        if (!token().trim()) { setError('Please paste your license token.'); return; }
        if (IS_BROWSER) return;
        setActivating(true); setError(''); setSuccess('');
        try {
            const LS = await import('../../wailsjs/go/services/LicensingService');
            await (LS as any).ActivateLicense(token().trim());
            setSuccess('License activated successfully.');
            setToken('');
            await refresh();
        } catch (e: any) {
            setError(e?.message ?? String(e));
        } finally {
            setActivating(false);
        }
    };

    const deactivate = async () => {
        if (IS_BROWSER) return;
        setError(''); setSuccess('');
        try {
            const LS = await import('../../wailsjs/go/services/LicensingService');
            await (LS as any).DeactivateLicense();
            setSuccess('License removed — reverted to Community tier.');
            await refresh();
        } catch (e: any) {
            setError(e?.message ?? String(e));
        }
    };

    const tierColor = () => TIER_COLORS[status()?.tier ?? 'Community'] ?? '#6b7280';
    const expiresLabel = () => {
        const exp = status()?.expires_at;
        if (!exp) return 'Perpetual';
        return new Date(exp * 1000).toLocaleDateString();
    };

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">

            {/* ── Header ── */}
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 18px;">🔑</span>
                    <h2 style="font-size: 14px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">License & Entitlements</h2>
                </div>
                <Show when={status()}>
                    <div style={`font-family: var(--font-mono); font-size: 11px; font-weight: 800; letter-spacing: 1px; color: ${tierColor()};`}>
                        {status()!.tier.toUpperCase()}
                    </div>
                </Show>
            </div>

            <div style="padding: 1.5rem; display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; max-width: 1200px;">

                {/* ── Current License Card ── */}
                <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem;">
                    <div style="font-size: 10px; letter-spacing: 2px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Current License</div>

                    <Show when={status()}>
                        <div style={`font-size: 2rem; font-weight: 900; font-family: var(--font-mono); color: ${tierColor()};`}>
                            {status()!.tier}
                        </div>
                        <div style="font-size: 12px; color: var(--text-secondary); line-height: 1.5;">
                            {TIER_DESCRIPTIONS[status()!.tier ?? 'Community']}
                        </div>

                        <Show when={status()!.is_licensed}>
                            <div style="display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.5rem;">
                                <div style="display: flex; justify-content: space-between; font-size: 11px; font-family: var(--font-mono);">
                                    <span style="color: var(--text-muted);">LICENSEE</span>
                                    <span>{status()!.licensee}</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 11px; font-family: var(--font-mono);">
                                    <span style="color: var(--text-muted);">LICENSE ID</span>
                                    <span style="color: var(--text-muted); font-size: 10px;">{status()!.license_id}</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 11px; font-family: var(--font-mono);">
                                    <span style="color: var(--text-muted);">EXPIRES</span>
                                    <span style={status()!.is_expired ? 'color: #f85149;' : ''}>{expiresLabel()}</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 11px; font-family: var(--font-mono);">
                                    <span style="color: var(--text-muted);">SEATS</span>
                                    <span>{status()!.max_seats === 0 ? 'Unlimited' : status()!.max_seats}</span>
                                </div>
                                <div style="display: flex; justify-content: space-between; font-size: 11px; font-family: var(--font-mono);">
                                    <span style="color: var(--text-muted);">AGENTS</span>
                                    <span style={status()!.agents_at_limit ? 'color: #f0883e;' : ''}>
                                        {status()!.active_agents} / {status()!.max_agents === 0 ? '∞' : status()!.max_agents}
                                        {status()!.agents_at_limit ? ' ⚠ AT LIMIT' : ''}
                                    </span>
                                </div>
                            </div>
                        </Show>

                        <Show when={status()!.is_expired}>
                            <div style="background: rgba(248,81,73,0.1); border: 1px solid rgba(248,81,73,0.4); border-radius: 4px; padding: 8px 12px; font-size: 11px; color: #f85149; font-family: var(--font-mono);">
                                ⚠ LICENSE EXPIRED — features locked to Community tier
                            </div>
                        </Show>

                        <Show when={status()!.is_licensed}>
                            <button
                                onClick={deactivate}
                                style="margin-top: auto; background: transparent; border: 1px solid rgba(248,81,73,0.4); color: #f85149; padding: 7px 14px; border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px; text-transform: uppercase; width: fit-content;"
                            >
                                Remove License
                            </button>
                        </Show>
                    </Show>
                </div>

                {/* ── Activate License Card ── */}
                <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem;">
                    <div style="font-size: 10px; letter-spacing: 2px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Activate License</div>

                    <div style="font-size: 12px; color: var(--text-secondary); line-height: 1.5;">
                        Paste your license token below. Verification is fully offline — no network request is made.
                        Tokens are Ed25519-signed and hardware-bound to prevent redistribution.
                    </div>

                    <textarea
                        placeholder="oblivra.v1.eyJ....<paste license token here>"
                        value={token()}
                        onInput={(e) => setToken((e.target as HTMLTextAreaElement).value)}
                        style="width: 100%; height: 120px; background: var(--bg-primary); border: 1px solid var(--glass-border); border-radius: 4px; color: var(--text-primary); font-family: var(--font-mono); font-size: 11px; padding: 10px; resize: vertical; box-sizing: border-box;"
                    />

                    <Show when={error()}>
                        <div style="background: rgba(248,81,73,0.1); border: 1px solid rgba(248,81,73,0.4); border-radius: 4px; padding: 8px 12px; font-size: 11px; color: #f85149; font-family: var(--font-mono);">
                            {error()}
                        </div>
                    </Show>

                    <Show when={success()}>
                        <div style="background: rgba(63,185,80,0.1); border: 1px solid rgba(63,185,80,0.4); border-radius: 4px; padding: 8px 12px; font-size: 11px; color: #3fb950; font-family: var(--font-mono);">
                            ✓ {success()}
                        </div>
                    </Show>

                    <button
                        onClick={activate}
                        disabled={activating()}
                        style={`background: ${activating() ? 'rgba(87,139,255,0.1)' : 'rgba(87,139,255,0.2)'}; border: 1px solid rgba(87,139,255,0.5); color: var(--accent-primary); padding: 8px 20px; border-radius: 4px; cursor: ${activating() ? 'not-allowed' : 'pointer'}; font-family: var(--font-mono); font-size: 12px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase; width: 100%;`}
                    >
                        {activating() ? '⏳ VERIFYING...' : '🔑 ACTIVATE LICENSE'}
                    </button>
                </div>

                {/* ── Feature Matrix ── */}
                <div style="grid-column: 1 / -1; background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.5rem;">
                    <div style="font-size: 10px; letter-spacing: 2px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 1.25rem;">Feature Entitlements</div>

                    <div style="display: grid; grid-template-columns: repeat(4, 1fr); gap: 1.5rem;">
                        <For each={FEATURE_GROUPS}>
                            {(group) => (
                                <div>
                                    <div style={`font-size: 10px; font-weight: 800; letter-spacing: 1.5px; text-transform: uppercase; margin-bottom: 0.75rem; color: ${TIER_COLORS[group.tier] ?? '#6b7280'}; font-family: var(--font-mono);`}>
                                        {group.label}
                                    </div>
                                    <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                                        <For each={group.features}>
                                            {(feat) => {
                                                const enabled = () => featureMap()[feat] === true;
                                                return (
                                                    <div style={`display: flex; align-items: center; gap: 8px; font-size: 11px; font-family: var(--font-mono); opacity: ${enabled() ? '1' : '0.35'};`}>
                                                        <span style={`color: ${enabled() ? '#3fb950' : '#6b7280'}; font-size: 10px;`}>
                                                            {enabled() ? '●' : '○'}
                                                        </span>
                                                        <span style={enabled() ? 'color: var(--text-primary);' : 'color: var(--text-muted);'}>
                                                            {FEATURE_LABELS[feat] ?? feat}
                                                        </span>
                                                    </div>
                                                );
                                            }}
                                        </For>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

            </div>
        </div>
    );
};
