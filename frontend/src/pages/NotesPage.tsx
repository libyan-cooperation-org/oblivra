import { Component } from 'solid-js';
import { useApp } from '@core/store';
import { NotesPanel } from '../components/notes/NotesPanel';

export const NotesPage: Component = () => {
    const [state] = useApp();

    return (
        <div class="page-container" style={{ padding: '24px', height: '100%', display: 'flex', 'flex-direction': 'column' }}>
            <div class="page-header" style={{ 'margin-bottom': '24px' }}>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800 }}>Operation Notes</h1>
                <p style={{ color: 'var(--text-muted)', 'font-size': '12px' }}>Collaborative notes and incident documentation.</p>
            </div>
            
            <div class="page-content" style={{ flex: 1, overflow: 'auto' }}>
                <NotesPanel sessionId={state.activeSessionId || undefined} />
            </div>
        </div>
    );
};
