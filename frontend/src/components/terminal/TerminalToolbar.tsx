import { Component, createSignal, Show, For, onCleanup } from 'solid-js';
import { ListCredentials } from '../../../wailsjs/go/app/VaultService';
// @ts-ignore
import { PushCredential, SendInput } from '../../../wailsjs/go/app/SSHService';

interface TerminalToolbarProps {
    sessionId: string;
    hostLabel: string;
    onDisconnect: () => void;
    onToggleSearch: () => void;
    onClear: () => void;
    onSplit: (direction: 'horizontal' | 'vertical') => void;
    onToggleSftp: () => void;
    onShare: () => void;
    onCopyOutput?: () => void; // [FIXED] Explicit copy-output handler
    onError?: (msg: string) => void; 
}

export const TerminalToolbar: Component<TerminalToolbarProps> = (props) => {
    // UI State
    const [activeMenu, setActiveMenu] = createSignal<'none' | 'vault' | 'menu'>('none');
    
    // Vault State
    const [credentials, setCredentials] = createSignal<any[]>([]);
    const [loadingVault, setLoadingVault] = createSignal(false);

    // [New] Close menus when clicking outside
    const handleOutsideClick = (e: MouseEvent) => {
        const target = e.target as HTMLElement;
        if (!target.closest('.toolbar-dropdown')) {
            setActiveMenu('none');
        }
    };

    document.addEventListener('click', handleOutsideClick);
    onCleanup(() => document.removeEventListener('click', handleOutsideClick));

    const toggleMenu = (menu: 'vault' | 'menu') => {
        setActiveMenu(prev => prev === menu ? 'none' : menu);
        if (menu === 'vault' && activeMenu() === 'vault') {
            loadCredentials();
        }
    };

    const loadCredentials = async () => {
        if (loadingVault()) return; // Prevent race conditions on multiple clicks
        
        setLoadingVault(true);
        try {
            const creds = await ListCredentials("");
            const filtered = creds.filter((c: any) =>
                ['password', 'token', 'api_key', 'key'].includes(c.type)
            );
            setCredentials(filtered);
        } catch (e) {
            console.error("[TOOLBAR] Failed to load credentials:", e);
            props.onError?.("Failed to load vault credentials.");
        } finally {
            setLoadingVault(false);
        }
    };

    const handleInject = async (credId: string) => {
        try {
            await PushCredential(props.sessionId, credId);
            setActiveMenu('none');
        } catch (e) {
            console.error("[TOOLBAR] Injection failed:", e);
            props.onError?.("Failed to inject credential.");
        }
    };

    const sendSignal = async (signal: string) => {
        try {
            await SendInput(props.sessionId, signal);
        } catch (e) {
            console.error("[TOOLBAR] Signal failed:", e);
            props.onError?.("Failed to send terminal signal.");
        }
    };

    return (
        <div class="terminal-toolbar">
            <div class="toolbar-left">
                <span class="toolbar-label">{props.hostLabel}</span>
                <div class="mobile-signals">
                    <button class="signal-btn" onClick={() => sendSignal('\x03')} aria-label="Send Ctrl+C" title="Ctrl+C">^C</button>
                    <button class="signal-btn" onClick={() => sendSignal('\x1a')} aria-label="Send Ctrl+Z" title="Ctrl+Z">^Z</button>
                    <button class="signal-btn" onClick={() => sendSignal('\x04')} aria-label="Send Ctrl+D" title="Ctrl+D">^D</button>
                </div>
            </div>

            <div class="toolbar-right">
                <button id={`btn-search-${props.sessionId}`} class="toolbar-btn" onClick={props.onToggleSearch} aria-label="Search" title="Search (Ctrl+Shift+F)">🔍</button>
                <button id={`btn-sftp-${props.sessionId}`} class="toolbar-btn" onClick={props.onToggleSftp} aria-label="SFTP File Browser" title="SFTP File Browser">📂</button>
                <button id={`btn-split-h-${props.sessionId}`} class="toolbar-btn" onClick={() => props.onSplit('horizontal')} aria-label="Split horizontal" title="Split horizontal">⬜⬜</button>
                <button id={`btn-split-v-${props.sessionId}`} class="toolbar-btn" onClick={() => props.onSplit('vertical')} aria-label="Split vertical" title="Split vertical">⬛<br />⬛</button>
                <button id={`btn-share-${props.sessionId}`} class="toolbar-btn" onClick={() => props.onShare()} aria-label="Share Session" title="Share Session">🔗</button>

                <div class="toolbar-dropdown">
                    <button
                        id={`btn-vault-${props.sessionId}`}
                        class="toolbar-btn"
                        onClick={(e) => { e.stopPropagation(); toggleMenu('vault'); }}
                        aria-label="Inject Credential from Vault"
                        title="Inject Credential from Vault"
                    >
                        🔑
                    </button>
                    
                    <Show when={activeMenu() === 'vault'}>
                        <div class="dropdown-menu vault-dropdown" style={{ right: '0', 'min-width': '220px' }}>
                            <div class="dropdown-header" style={{ padding: '8px 12px', 'font-size': '10px', 'font-weight': 'bold', color: 'var(--text-muted)', 'border-bottom': '1px solid var(--border-primary)' }}>SELECT CREDENTIAL TO INJECT</div>
                            <div class="vault-credential-list" style={{ 'max-height': '300px', 'overflow-y': 'auto' }}>
                                <Show when={credentials().length > 0} fallback={
                                    <div class="vault-empty-state" style={{ padding: '12px', 'font-size': '11px', color: 'var(--text-muted)' }}>
                                        {loadingVault() ? 'Scanning Vault...' : 'No credentials found.'}
                                    </div>
                                }>
                                    <For each={credentials()}>
                                        {(cred) => (
                                            <button onClick={() => handleInject(cred.id)} class="vault-cred-btn" style={{ display: 'flex', 'flex-direction': 'column', 'align-items': 'flex-start', gap: '2px', padding: '8px 12px' }}>
                                                <div class="vault-cred-header" style={{ display: 'flex', 'align-items': 'center', gap: '6px' }}>
                                                    <span style={{ 'font-size': '14px' }}>{cred.type === 'key' ? '🔐' : '🔑'}</span>
                                                    <span class="vault-cred-label" style={{ 'font-weight': '600' }}>{cred.label}</span>
                                                </div>
                                                <span class="vault-cred-meta" style={{ 'font-size': '9px', opacity: 0.6 }}>{cred.type.toUpperCase()} • {cred.id.substring(0, 8)}</span>
                                            </button>
                                        )}
                                    </For>
                                </Show>
                            </div>
                        </div>
                    </Show>
                </div>

                <button id={`btn-clear-${props.sessionId}`} class="toolbar-btn" onClick={props.onClear} aria-label="Clear terminal" title="Clear terminal">🗑️</button>

                <div class="toolbar-dropdown">
                    <button
                        id={`btn-menu-${props.sessionId}`}
                        class="toolbar-btn"
                        onClick={(e) => { e.stopPropagation(); toggleMenu('menu'); }}
                        aria-label="More options"
                    >
                        ⋮
                    </button>

                    <Show when={activeMenu() === 'menu'}>
                        <div class="dropdown-menu">
                            <button onClick={() => { 
                                if (props.onCopyOutput) {
                                    props.onCopyOutput();
                                } else {
                                    // Fallback if not specifically wired in parent
                                    navigator.clipboard?.writeText('Clipboard unsupported without proper parent wiring');
                                }
                                setActiveMenu('none'); 
                            }}>
                                Copy Output
                            </button>
                            <button onClick={() => setActiveMenu('none')}>
                                Start Recording
                            </button>
                            <button onClick={() => setActiveMenu('none')}>
                                Session Info
                            </button>
                            <hr />
                            <button class="danger" onClick={() => { setActiveMenu('none'); props.onDisconnect(); }}>
                                Disconnect
                            </button>
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
