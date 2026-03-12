import { Component } from 'solid-js';
import { useApp } from '@core/store';
import { TunnelManager } from '../components/tunnels/TunnelManager';

export const TunnelsPage: Component = () => {
    const [state] = useApp();

    return (
        <div class="page-container" style={{ padding: '24px', height: '100%', display: 'flex', 'flex-direction': 'column' }}>
            <div class="page-header" style={{ 'margin-bottom': '24px' }}>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800 }}>Tunnel Management</h1>
                <p style={{ color: 'var(--text-muted)', 'font-size': '12px' }}>Active SSH tunnels and port forwarding rules across the fleet.</p>
            </div>
            
            <div class="page-content" style={{ flex: 1, overflow: 'auto' }}>
                <TunnelManager sessionId={state.activeSessionId || ""} />
            </div>
        </div>
    );
};
