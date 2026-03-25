import { Component, Show, createSignal, createResource, onMount, onCleanup } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface OperatorContext {
    host_id: string;
    host_label: string;
    risk_score: number;
    risk_level: string;
    alert_count: number;
    failed_logins: number;
    threat_summary: string;
}

interface OperatorBannerProps {
    hostId: string;
    hostLabel: string;
}

/**
 * OperatorBanner displays a SIEM-powered security context overlay
 * on the terminal tab bar. Shows risk score, alert count, and threat summary
 * for the currently active SSH host.
 */
export const OperatorBanner: Component<OperatorBannerProps> = (props) => {
    const [expanded, setExpanded] = createSignal(false);
    let pollInterval: number | undefined;

    const fetchContext = async () => {
        if (IS_BROWSER || !props.hostId || props.hostId === 'local') return null;
        try {
            const { GetContext } = await import('../../../wailsjs/go/services/OperatorService');
            return await GetContext(props.hostId) as OperatorContext;
        } catch { return null; }
    };

    const [ctx, { refetch }] = createResource(() => props.hostId, fetchContext);

    // Poll every 30 seconds for live updates
    onMount(() => {
        pollInterval = window.setInterval(() => refetch(), 30000);
    });
    onCleanup(() => {
        if (pollInterval) clearInterval(pollInterval);
    });

    const riskColor = () => {
        const level = ctx()?.risk_level;
        switch (level) {
            case 'critical': return '#e04040';
            case 'high': return '#f58b00';
            case 'medium': return '#f5c518';
            case 'low': return '#5cc05c';
            default: return 'var(--text-muted)';
        }
    };

    const riskBg = () => {
        const level = ctx()?.risk_level;
        switch (level) {
            case 'critical': return 'rgba(224,64,64,0.1)';
            case 'high': return 'rgba(245,139,0,0.1)';
            case 'medium': return 'rgba(245,197,24,0.08)';
            default: return 'transparent';
        }
    };

    return (
        <Show when={ctx() && ctx()!.risk_level !== 'none' && ctx()!.alert_count > 0}>
            <div
                onClick={() => setExpanded(!expanded())}
                style={{
                    display: 'flex',
                    'align-items': 'center',
                    gap: '8px',
                    padding: '4px 12px',
                    background: riskBg(),
                    'border-bottom': `1px solid ${riskColor()}33`,
                    cursor: 'pointer',
                    'flex-shrink': '0',
                    transition: 'background 0.2s',
                    'font-family': 'var(--font-ui)',
                }}
            >
                {/* Risk indicator */}
                <div style={{
                    width: '8px', height: '8px', 'border-radius': '50%',
                    background: riskColor(),
                    'box-shadow': `0 0 6px ${riskColor()}80`,
                    animation: ctx()?.risk_level === 'critical' ? 'pulse 1.5s infinite' : 'none',
                }} />

                {/* Summary text */}
                <span style={{
                    'font-size': '10px',
                    'font-weight': '600',
                    color: riskColor(),
                    'letter-spacing': '0.3px',
                    flex: '1',
                }}>
                    {ctx()?.threat_summary}
                </span>

                {/* Badges */}
                <Show when={ctx()!.alert_count > 0}>
                    <span style={{
                        'font-size': '9px',
                        'font-weight': '700',
                        padding: '1px 6px',
                        'border-radius': '8px',
                        background: `${riskColor()}20`,
                        color: riskColor(),
                        border: `1px solid ${riskColor()}40`,
                    }}>
                        {ctx()!.alert_count} ALERT{ctx()!.alert_count > 1 ? 'S' : ''}
                    </span>
                </Show>

                <Show when={ctx()!.failed_logins > 0}>
                    <span style={{
                        'font-size': '9px',
                        padding: '1px 6px',
                        'border-radius': '8px',
                        background: 'rgba(224,64,64,0.1)',
                        color: '#e04040',
                        border: '1px solid rgba(224,64,64,0.3)',
                    }}>
                        {ctx()!.failed_logins} FAILED
                    </span>
                </Show>

                {/* Expand icon */}
                <span style={{
                    'font-size': '10px', color: 'var(--text-muted)',
                    transform: expanded() ? 'rotate(180deg)' : 'rotate(0)',
                    transition: 'transform 0.2s',
                }}>▾</span>
            </div>

            {/* Expanded detail panel */}
            <Show when={expanded()}>
                <div style={{
                    padding: '8px 14px',
                    background: 'var(--surface-2)',
                    'border-bottom': '1px solid var(--border-primary)',
                    'font-size': '11px',
                    display: 'flex',
                    gap: '16px',
                    'flex-shrink': '0',
                }}>
                    <div>
                        <div style={{ color: 'var(--text-muted)', 'font-size': '9px', 'text-transform': 'uppercase' }}>Risk</div>
                        <div style={{ color: riskColor(), 'font-weight': '700', 'font-size': '18px' }}>{ctx()?.risk_score}</div>
                    </div>
                    <div>
                        <div style={{ color: 'var(--text-muted)', 'font-size': '9px', 'text-transform': 'uppercase' }}>Alerts</div>
                        <div style={{ color: 'var(--text-primary)', 'font-weight': '600' }}>{ctx()?.alert_count}</div>
                    </div>
                    <div>
                        <div style={{ color: 'var(--text-muted)', 'font-size': '9px', 'text-transform': 'uppercase' }}>Failed Logins</div>
                        <div style={{ color: '#e04040', 'font-weight': '600' }}>{ctx()?.failed_logins}</div>
                    </div>
                    <div style={{ flex: '1' }}>
                        <div style={{ color: 'var(--text-muted)', 'font-size': '9px', 'text-transform': 'uppercase' }}>Host</div>
                        <div style={{ color: 'var(--text-primary)', 'font-family': 'var(--font-mono)' }}>
                            {ctx()?.host_label}
                        </div>
                    </div>
                    <button
                        onClick={(e) => { e.stopPropagation(); /* TODO: trigger isolation */ }}
                        style={{
                            background: 'rgba(224,64,64,0.1)',
                            border: '1px solid rgba(224,64,64,0.4)',
                            color: '#e04040',
                            padding: '4px 10px',
                            'border-radius': '3px',
                            'font-size': '9px',
                            'font-weight': '700',
                            cursor: 'pointer',
                            'text-transform': 'uppercase',
                            'align-self': 'center',
                        }}
                        title="Isolate Host (Ctrl+Shift+I)"
                    >
                        🔒 ISOLATE
                    </button>
                </div>
            </Show>
        </Show>
    );
};
