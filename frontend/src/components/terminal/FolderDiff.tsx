import { Component, createSignal, For, Show } from 'solid-js';
import { CompareDirectories } from '../../../wailsjs/go/services/SSHService';
import { Download, Upload } from '../../../wailsjs/go/services/FileService';
import { services } from '../../../wailsjs/go/models';
import { useApp } from '../../core/store';
import '../../styles/filebrowser.css';

interface FolderDiffProps {
    sessionId: string;
    onClose: () => void;
}

export const FolderDiff: Component<FolderDiffProps> = (props) => {
    const [localPath, setLocalPath] = createSignal<string>('C:\\'); // Default starting path
    const [remotePath, setRemotePath] = createSignal<string>('/etc/nginx'); // Example default
    const [_state, actions] = useApp();

    const [diff, setDiff] = createSignal<services.DirectoryDiff | null>(null);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal('');

    const runCompare = async () => {
        if (!localPath() || !remotePath()) return;
        setLoading(true);
        setError('');
        try {
            const result = await CompareDirectories(props.sessionId, localPath(), remotePath());
            setDiff(result);
        } catch (err) {
            setError((err as Error).message || 'Comparison failed');
        } finally {
            setLoading(false);
        }
    };

    const handleSync = async (item: services.DiffItem, direction: 'to_remote' | 'to_local') => {
        try {
            const local = `${localPath()}\\${item.name}`; // Naive join, could use path logic
            const remote = `${remotePath()}/${item.name}`;

            if (direction === 'to_remote') {
                await Upload(props.sessionId, local, remote);
            } else {
                await Download(props.sessionId, remote, local, item.remote_size || 0);
            }
            // We dont auto-refresh here to avoid layout jump during active sync
        } catch (e) {
            actions.notify(`Sync Failed: ${(e as Error).message}`, 'error');
        }
    };

    const formatSize = (bytes: number) => {
        if (!bytes) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'identical': return 'var(--text-secondary)';
            case 'missing_local': return 'var(--primary-color)';   // exists only on remote
            case 'missing_remote': return 'var(--success-color)';  // exists only on local
            case 'modified': return 'var(--warning)';              // exists on both but different
            default: return 'var(--text-primary)';
        }
    };

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'identical': return '✓';
            case 'missing_local': return 'Cloud Only';
            case 'missing_remote': return 'Local Only';
            case 'modified': return '≠ Modified';
            default: return '?';
        }
    };

    return (
        <div class="file-browser" style="display: flex; flex-direction: column;">
            <div class="fb-header" style="flex-direction: column; align-items: stretch; gap: 8px;">
                <div style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
                    <h3 style="margin: 0; font-size: 14px;">🔍 Visual Folder Diff</h3>
                    <button onClick={props.onClose} class="fb-icon-btn" title="Close">✕</button>
                </div>

                <div style="display: flex; gap: 8px; align-items: center;">
                    <div style="flex: 1; display: flex; flex-direction: column; gap: 4px;">
                        <span style="font-size: 10px; color: var(--text-secondary)">Local Base Path</span>
                        <input
                            class="ops-input"
                            style="padding: 4px 8px; font-size: 12px; height: 28px;"
                            value={localPath()}
                            onInput={(e) => setLocalPath(e.currentTarget.value)}
                        />
                    </div>
                    <div style="flex: 1; display: flex; flex-direction: column; gap: 4px;">
                        <span style="font-size: 10px; color: var(--text-secondary)">Remote Base Path</span>
                        <input
                            class="ops-input"
                            style="padding: 4px 8px; font-size: 12px; height: 28px;"
                            value={remotePath()}
                            onInput={(e) => setRemotePath(e.currentTarget.value)}
                        />
                    </div>
                    <button
                        class="action-btn"
                        style="height: 28px; margin-top: 18px;"
                        onClick={runCompare}
                        disabled={loading()}
                    >
                        {loading() ? '...' : 'Compare'}
                    </button>
                </div>
            </div>

            <Show when={error()}>
                <div class="fb-error">{error()}</div>
            </Show>

            <div class="fb-content" style="flex: 1; overflow-y: auto; background: var(--bg-surface);">
                <Show when={loading()}>
                    <div class="fb-loading">
                        <div class="skeleton-row"></div>
                        <div class="skeleton-row"></div>
                        <div class="skeleton-row"></div>
                    </div>
                </Show>

                <Show when={!loading() && diff()}>
                    <div style="padding: 12px;">
                        <table style="width: 100%; border-collapse: collapse; text-align: left; font-size: 13px;">
                            <thead>
                                <tr style="border-bottom: 1px solid var(--border-subtle); color: var(--text-secondary);">
                                    <th style="padding: 8px; font-weight: 500;">Status</th>
                                    <th style="padding: 8px; font-weight: 500;">File Name</th>
                                    <th style="padding: 8px; font-weight: 500;">Local Size</th>
                                    <th style="padding: 8px; font-weight: 500;">Remote Size</th>
                                    <th style="padding: 8px; font-weight: 500;">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <For each={diff()?.items}>
                                    {(item) => (
                                        <tr style={`border-bottom: 1px solid var(--border-subtle); background: ${item.status === 'identical' ? 'transparent' : 'rgba(255,255,255,0.02)'};`}>
                                            <td style={`padding: 8px; color: ${getStatusColor(item.status)}; font-size: 11px; white-space: nowrap;`}>
                                                {getStatusIcon(item.status)}
                                            </td>
                                            <td style="padding: 8px; display: flex; align-items: center; gap: 6px;">
                                                <span>{item.is_dir ? '📁' : '📄'}</span>
                                                <span style={item.status === 'identical' ? 'color: var(--text-secondary)' : ''}>{item.name}</span>
                                            </td>
                                            <td style="padding: 8px; color: var(--text-secondary);">{item.status !== 'missing_local' && !item.is_dir ? formatSize(item.local_size) : '-'}</td>
                                            <td style="padding: 8px; color: var(--text-secondary);">{item.status !== 'missing_remote' && !item.is_dir ? formatSize(item.remote_size) : '-'}</td>
                                            <td style="padding: 8px;">
                                                <div style="display: flex; gap: 4px;">
                                                    <Show when={item.status === 'missing_remote' || item.status === 'modified' && !item.is_dir}>
                                                        <button class="action-btn" style="padding: 2px 6px; font-size: 10px;" onClick={() => handleSync(item, 'to_remote')} title="Upload to Remote">Upload ↑</button>
                                                    </Show>
                                                    <Show when={item.status === 'missing_local' || item.status === 'modified' && !item.is_dir}>
                                                        <button class="action-btn" style="padding: 2px 6px; font-size: 10px;" onClick={() => handleSync(item, 'to_local')} title="Download from Remote">Pull ↓</button>
                                                    </Show>
                                                </div>
                                            </td>
                                        </tr>
                                    )}
                                </For>
                            </tbody>
                        </table>

                        <Show when={diff()?.items.length === 0}>
                            <div class="fb-empty" style="text-align: center; padding: 40px; color: var(--text-secondary);">
                                No files found in either directory.
                            </div>
                        </Show>
                    </div>
                </Show>
            </div>
        </div>
    );
};
