import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetAlertHistory, GetDetectionRules } from '../../../wailsjs/go/services/AlertingService';

const TACTICS = [
    { id: "TA0001", name: "Initial Access" },
    { id: "TA0002", name: "Execution" },
    { id: "TA0003", name: "Persistence" },
    { id: "TA0004", name: "Privilege Escalation" },
    { id: "TA0005", name: "Defense Evasion" },
    { id: "TA0006", name: "Credential Access" },
    { id: "TA0007", name: "Discovery" },
    { id: "TA0008", name: "Lateral Movement" },
    { id: "TA0009", name: "Collection" },
    { id: "TA0011", name: "Command and Control" },
    { id: "TA0010", name: "Exfiltration" },
    { id: "TA0040", name: "Impact" }
];

export const MitreHeatmap: Component = () => {
    const [rules, setRules] = createSignal<any[]>([]);
    const [alerts, setAlerts] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    const fetchData = async () => {
        setLoading(true);
        try {
            const fetchedRules = await GetDetectionRules();
            const fetchedAlerts = await GetAlertHistory();
            setRules(fetchedRules || []);
            setAlerts(fetchedAlerts || []);
        } catch (e) {
            console.error('Failed to load MITRE data:', e);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        fetchData();
        // optionally refresh every 30s
        const interval = setInterval(fetchData, 30000);
        return () => clearInterval(interval);
    });

    const getAlertsForRule = (ruleId: string) => {
        return alerts().filter(a => a.rule_id === ruleId);
    };

    const getRulesForTactic = (tacticId: string) => {
        return rules().filter(r => r.MitreTactics && r.MitreTactics.includes(tacticId));
    };

    const getSeverityColor = (severity: string, isTriggered: boolean) => {
        if (!isTriggered) return 'rgba(255, 255, 255, 0.05)';
        switch (severity.toLowerCase()) {
            case 'critical': return 'rgba(248, 81, 73, 0.8)';
            case 'high': return 'rgba(210, 153, 34, 0.8)';
            case 'medium': return 'rgba(210, 153, 34, 0.4)';
            default: return 'rgba(47, 129, 247, 0.8)';
        }
    };

    const getSeverityBorderColor = (severity: string, isTriggered: boolean) => {
        if (!isTriggered) return 'rgba(255, 255, 255, 0.1)';
        switch (severity.toLowerCase()) {
            case 'critical': return '#f85149';
            case 'high': return '#d29922';
            case 'medium': return '#d29922';
            default: return '#2f81f7';
        }
    };

    return (
        <div class="mitre-heatmap" style={{ 'padding': '20px', 'color': '#c9d1d9', 'height': '100%', 'overflow': 'hidden', 'display': 'flex', 'flex-direction': 'column' }}>
            <div style={{ 'display': 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'margin-bottom': '20px' }}>
                <div>
                    <h2 style={{ 'margin': '0', 'font-size': '1.5rem', 'color': '#fff' }}>MITRE ATT&CK Matrix</h2>
                    <p style={{ 'color': '#8b949e', 'margin': '4px 0 0 0', 'font-size': '0.9rem' }}>
                        Visualizing active detection rules and observed heuristics across the attack kill chain.
                    </p>
                </div>
                <button
                    onClick={fetchData}
                    disabled={loading()}
                    style={{ 'background': '#21262d', 'border': '1px solid rgba(240,246,252,0.1)', 'color': '#c9d1d9', 'padding': '6px 12px', 'border-radius': '6px', 'cursor': 'pointer' }}
                >
                    {loading() ? '↻ Reloading...' : '↻ Refresh Data'}
                </button>
            </div>

            <div class="matrix-scroll-container" style={{ 'flex': '1', 'overflow-x': 'auto', 'overflow-y': 'auto', 'border': '1px solid #30363d', 'border-radius': '8px', 'background': '#0d1117' }}>
                <div style={{ 'display': 'flex', 'min-width': 'max-content' }}>
                    <For each={TACTICS}>
                        {(tactic) => (
                            <div class="tactic-column" style={{ 'width': '220px', 'min-width': '220px', 'border-right': '1px solid #30363d', 'display': 'flex', 'flex-direction': 'column' }}>
                                <div style={{ 'background': '#161b22', 'padding': '12px 16px', 'border-bottom': '2px solid #30363d', 'position': 'sticky', 'top': '0', 'z-index': '10' }}>
                                    <div style={{ 'font-weight': '600', 'color': '#e6edf3', 'font-size': '0.95rem' }}>{tactic.name}</div>
                                    <div style={{ 'color': '#8b949e', 'font-size': '0.75rem', 'margin-top': '4px' }}>{window.btoa(tactic.id).replace(/=/g, '').substring(0, 6).toUpperCase()} / {tactic.id}</div>
                                </div>
                                <div style={{ 'padding': '12px', 'display': 'flex', 'flex-direction': 'column', 'gap': '12px', 'flex': '1' }}>
                                    <For each={getRulesForTactic(tactic.id)}>
                                        {(rule) => {
                                            const triggeredAlerts = getAlertsForRule(rule.ID || rule.id);
                                            const isTriggered = triggeredAlerts.length > 0;
                                            return (
                                                <div
                                                    title={rule.Description || rule.description}
                                                    style={{
                                                        'background': getSeverityColor(rule.Severity || rule.severity, isTriggered),
                                                        'border': `1px solid ${getSeverityBorderColor(rule.Severity || rule.severity, isTriggered)}`,
                                                        'border-radius': '6px',
                                                        'padding': '10px',
                                                        'font-size': '0.85rem',
                                                        'transition': 'all 0.2s',
                                                        'color': isTriggered ? '#fff' : '#c9d1d9',
                                                        'cursor': 'default',
                                                        'position': 'relative',
                                                        'display': 'flex',
                                                        'flex-direction': 'column',
                                                        'gap': '6px'
                                                    }}
                                                >
                                                    <div style={{ 'font-weight': '600', 'line-height': '1.3' }}>
                                                        {rule.Name || rule.name}
                                                    </div>
                                                    <div style={{ 'display': 'flex', 'flex-wrap': 'wrap', 'gap': '4px' }}>
                                                        <For each={rule.MitreTechniques || rule.mitre_techniques}>
                                                            {(tech) => (
                                                                <span style={{
                                                                    'background': isTriggered ? 'rgba(255,255,255,0.2)' : 'rgba(255,255,255,0.1)',
                                                                    'padding': '2px 4px',
                                                                    'border-radius': '3px',
                                                                    'font-size': '0.7rem',
                                                                    'color': isTriggered ? '#fff' : '#8b949e'
                                                                }}>{tech}</span>
                                                            )}
                                                        </For>
                                                    </div>
                                                    <Show when={isTriggered}>
                                                        <div style={{ 'position': 'absolute', 'top': '-6px', 'right': '-6px', 'background': '#f85149', 'color': 'white', 'font-size': '0.7rem', 'font-weight': 'bold', 'padding': '2px 6px', 'border-radius': '10px', 'box-shadow': '0 0 4px rgba(0,0,0,0.5)' }}>
                                                            {triggeredAlerts.length}
                                                        </div>
                                                    </Show>
                                                </div>
                                            )
                                        }}
                                    </For>
                                    <Show when={getRulesForTactic(tactic.id).length === 0}>
                                        <div style={{ 'color': 'rgba(139, 148, 158, 0.4)', 'font-size': '0.8rem', 'text-align': 'center', 'padding': '20px 0', 'font-style': 'italic' }}>
                                            No tracking rules
                                        </div>
                                    </Show>
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>
        </div>
    );
};
