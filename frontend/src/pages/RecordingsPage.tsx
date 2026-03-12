import { Component } from 'solid-js';
import { RecordingPanel } from '../components/recordings/RecordingPanel';

export const RecordingsPage: Component = () => {
    return (
        <div class="page-container" style={{ padding: '24px', height: '100%', display: 'flex', 'flex-direction': 'column' }}>
            <div class="page-header" style={{ 'margin-bottom': '24px' }}>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800 }}>Session Recordings</h1>
                <p style={{ color: 'var(--text-muted)', 'font-size': '12px' }}>Immutable audit logs of all interactive sessions.</p>
            </div>
            
            <div class="page-content" style={{ flex: 1, overflow: 'auto' }}>
                <RecordingPanel />
            </div>
        </div>
    );
};
