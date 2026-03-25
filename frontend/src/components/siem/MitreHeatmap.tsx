import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { 
    Button, 
    normalizeSeverity 
} from '@components/ui';

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
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { GetDetectionRules, GetAlertHistory } = await import('../../../wailsjs/go/services/AlertingService');
            setRules(await GetDetectionRules() || []);
            setAlerts(await GetAlertHistory() || []);
        } catch (e) {
            console.error('Failed to load MITRE data:', e);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        fetchData();
        const interval = setInterval(fetchData, 30000);
        return () => clearInterval(interval);
    });

    const getAlertsForRule = (ruleId: string) => {
        return alerts().filter(a => a.rule_id === ruleId);
    };

    const getRulesForTactic = (tacticId: string) => {
        return rules().filter(r => r.MitreTactics && r.MitreTactics.includes(tacticId));
    };

    return (
        <div style="height: 100%; display: flex; flex-direction: column; overflow: hidden; background: var(--surface-0);">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--gap-lg); padding: 0 var(--gap-md);">
                <div>
                    <h2 style="margin: 0; font-size: 1.25rem; color: var(--text-heading); font-family: var(--font-ui); font-weight: 800; letter-spacing: -0.5px;">MITRE ATT&CK® Matrix</h2>
                    <p style="color: var(--text-muted); margin: 4px 0 0 0; font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px;">
                        VISUALIZING_THREAT_VECTORS_ACROSS_KILL_CHAIN
                    </p>
                </div>
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={fetchData}
                    loading={loading()}
                >
                    REFRESH_HEURISTICS
                </Button>
            </div>

            <div style="flex: 1; overflow-x: auto; overflow-y: hidden; border: 1px solid var(--border-primary); border-radius: var(--radius-sm); background: var(--surface-1); margin: 0 var(--gap-md) var(--gap-md) var(--gap-md);">
                <div style="display: flex; min-width: max-content; height: 100%;">
                    <For each={TACTICS}>
                        {(tactic) => (
                            <div style="width: 240px; min-width: 240px; border-right: 1px solid var(--border-primary); display: flex; flex-direction: column; height: 100%;">
                                <div style="background: var(--surface-2); padding: 12px 16px; border-bottom: 2px solid var(--border-secondary); position: sticky; top: 0; z-index: 10;">
                                    <div style="font-weight: 800; color: var(--accent-primary); font-size: 11px; text-transform: uppercase; letter-spacing: 1px;">{tactic.name}</div>
                                    <div style="color: var(--text-muted); font-size: 9px; font-family: var(--font-mono); margin-top: 4px; font-weight: 700;">{tactic.id}</div>
                                </div>
                                <div style="padding: 12px; display: flex; flex-direction: column; gap: 8px; flex: 1; overflow-y: auto; background: var(--surface-0);">
                                    <For each={getRulesForTactic(tactic.id)}>
                                        {(rule) => {
                                            const triggeredAlerts = getAlertsForRule(rule.ID || rule.id);
                                            const isTriggered = triggeredAlerts.length > 0;
                                            const severity = normalizeSeverity(rule.Severity || rule.severity);
                                            
                                            return (
                                                <div
                                                    title={rule.Description || rule.description}
                                                    style={{
                                                        'background': isTriggered ? `rgba(var(--alert-${severity}-rgb), 0.15)` : 'var(--surface-1)',
                                                        'border': `1px solid ${isTriggered ? `var(--alert-${severity})` : 'var(--border-primary)'}`,
                                                        'border-radius': 'var(--radius-sm)',
                                                        'padding': '10px',
                                                        'font-size': '11px',
                                                        'transition': 'all 0.2s',
                                                        'color': isTriggered ? 'var(--text-primary)' : 'var(--text-secondary)',
                                                        'position': 'relative',
                                                        'display': 'flex',
                                                        'flex-direction': 'column',
                                                        'gap': '6px',
                                                        'cursor': 'default'
                                                    }}
                                                >
                                                    <div style={{ 'font-weight': '700', 'font-family': 'var(--font-ui)', 'line-height': '1.3' }}>
                                                        {rule.Name || rule.name}
                                                    </div>
                                                    <div style={{ 'display': 'flex', 'flex-wrap': 'wrap', 'gap': '4px' }}>
                                                        <For each={rule.MitreTechniques || rule.mitre_techniques}>
                                                            {(tech) => (
                                                                <span style={{
                                                                    'background': 'rgba(0,0,0,0.2)',
                                                                    'padding': '1px 5px',
                                                                    'border-radius': '3px',
                                                                    'font-size': '9px',
                                                                    'font-family': 'var(--font-mono)',
                                                                    'color': 'var(--text-muted)',
                                                                    'border': '1px solid var(--border-subtle)'
                                                                }}>{tech}</span>
                                                            )}
                                                        </For>
                                                    </div>
                                                    <Show when={isTriggered}>
                                                        <div style={{ 
                                                            'position': 'absolute', 'top': '-6px', 'right': '-6px', 
                                                            'background': `var(--alert-${severity})`, 
                                                            'color': '#000', 'font-size': '9px', 'font-weight': '900', 
                                                            'padding': '1px 6px', 'border-radius': '10px', 
                                                            'box-shadow': '0 0 8px rgba(0,0,0,0.5)',
                                                            'border': '1px solid rgba(0,0,0,0.2)'
                                                        }}>
                                                            {triggeredAlerts.length}
                                                        </div>
                                                    </Show>
                                                </div>
                                            )
                                        }}
                                    </For>
                                    <Show when={getRulesForTactic(tactic.id).length === 0}>
                                        <div style="color: var(--text-muted); font-size: 10px; text-align: center; padding: 24px 0; font-family: var(--font-mono); opacity: 0.5;">
                                            NO_TRACKING_RULES
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
