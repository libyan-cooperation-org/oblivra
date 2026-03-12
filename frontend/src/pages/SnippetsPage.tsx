import { Component } from 'solid-js';
import { SnippetVault } from '../components/sidebar/SnippetVault';

export const SnippetsPage: Component = () => {
    return (
        <div class="page-container" style={{ padding: '24px', height: '100%', display: 'flex', 'flex-direction': 'column' }}>
            <div class="page-header" style={{ 'margin-bottom': '24px' }}>
                <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800 }}>Command Snippets</h1>
                <p style={{ color: 'var(--text-muted)', 'font-size': '12px' }}>Reusable command sequences and shell scripts.</p>
            </div>
            
            <div class="page-content" style={{ flex: 1, overflow: 'auto' }}>
                <SnippetVault />
            </div>
        </div>
    );
};
