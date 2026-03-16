import { Component, For, Show, createResource } from 'solid-js';
import { GetCapabilitiesMatrix } from '../../../../wailsjs/go/services/PolicyService';

const tactics = [
    { name: 'Initial Access',         technique: 'T1190 — Exploit Public-Facing App' },
    { name: 'Execution',              technique: 'T1059 — Command & Scripting Interpreter' },
    { name: 'Persistence',            technique: 'T1547 — Boot or Logon Autostart' },
    { name: 'Privilege Escalation',   technique: 'T1068 — Exploitation for Priv Esc' },
    { name: 'Defense Evasion',        technique: 'T1562 — Impair Defenses' },
    { name: 'Credential Access',      technique: 'T1003 — OS Credential Dumping' },
    { name: 'Discovery',              technique: 'T1046 — Network Service Discovery' },
    { name: 'Lateral Movement',       technique: 'T1021 — Remote Services' },
];

export const MitreAttackPanel: Component = () => {
    const [capabilities] = createResource(async () => {
        try { return await GetCapabilitiesMatrix(); }
        catch (e) { console.error('Failed to fetch capabilities:', e); return {}; }
    });

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Header */}
            <div style={{ padding: '8px 12px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', background: 'var(--surface-1)', 'flex-shrink': 0 }}>
                <span style={{ color: 'var(--text-muted)', 'font-weight': 800, 'letter-spacing': '2px', 'text-transform': 'uppercase', 'font-size': '10px' }}>ATT&CK Coverage Matrix</span>
                <span style={{ 'font-size': '9px', color: 'var(--accent-primary)', 'font-weight': 800 }}>V12.1</span>
            </div>

            {/* Grid */}
            <div style={{ flex: 1, 'overflow-y': 'auto', padding: '10px', display: 'grid', 'grid-template-columns': '1fr 1fr', gap: '8px' }}>
                <For each={tactics}>
                    {(tactic) => {
                        const isActive = () => capabilities()?.[tactic.name] || false;
                        return (
                            <div style={{
                                padding: '10px',
                                border: `1px solid ${isActive() ? 'rgba(87,139,255,0.4)' : 'var(--border-primary)'}`,
                                background: isActive() ? 'rgba(87,139,255,0.07)' : 'var(--surface-1)',
                                'border-radius': '3px',
                                transition: 'all 0.3s',
                                opacity: isActive() ? '1' : '0.55',
                                'box-shadow': isActive() ? 'inset 0 0 12px rgba(87,139,255,0.08)' : 'none',
                            }}>
                                <div style={{ 'font-size': '9px', 'text-transform': 'uppercase', color: 'var(--text-muted)', 'font-weight': 800, 'margin-bottom': '4px', 'letter-spacing': '0.5px' }}>
                                    {tactic.name}
                                </div>
                                <div style={{ 'font-size': '10px', 'line-height': '1.4', color: isActive() ? 'var(--accent-primary)' : 'var(--text-muted)', 'font-weight': isActive() ? 700 : 400, 'font-style': isActive() ? 'normal' : 'italic' }}>
                                    {tactic.technique}
                                </div>
                                <Show when={isActive()}>
                                    <div style={{ 'margin-top': '6px', display: 'flex', 'align-items': 'center', gap: '4px' }}>
                                        <div style={{ width: '6px', height: '6px', 'border-radius': '50%', background: 'var(--accent-primary)' }} />
                                        <span style={{ 'font-size': '8px', color: 'var(--accent-primary)', 'font-weight': 800, 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>Active Shield</span>
                                    </div>
                                </Show>
                            </div>
                        );
                    }}
                </For>
            </div>

            {/* Footer */}
            <div style={{ padding: '6px 12px', 'border-top': '1px solid var(--border-primary)', background: 'var(--surface-1)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'flex-shrink': 0 }}>
                <div style={{ display: 'flex', gap: '16px', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                    <span>COVERAGE: <span style={{ color: 'var(--accent-primary)' }}>84.2%</span></span>
                    <span>DETECTIONS: <span style={{ color: 'var(--accent-primary)' }}>142</span></span>
                </div>
                <button style={{ 'font-size': '9px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', 'font-weight': 800, 'font-family': 'var(--font-mono)', 'text-transform': 'uppercase' }}>
                    FULL MATRIX →
                </button>
            </div>
        </div>
    );
};
