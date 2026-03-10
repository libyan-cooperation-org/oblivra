import { Component, createSignal, createEffect, For, Show, onCleanup } from 'solid-js';
import {
    ListDirectory,
    Mkdir,
    Rename,
    Remove,
    Download,
    Upload,
    ReadFile,
    WriteFile,
} from '../../../wailsjs/go/app/FileService';
import { app } from '../../../wailsjs/go/models';
import { useApp } from '../../core/store';
import { FolderDiff } from './FolderDiff';
import '../../styles/filebrowser.css';

interface FileBrowserProps {
    sessionId: string;
    onClose: () => void;
    initialPath?: string;
}

export const FileBrowser: Component<FileBrowserProps> = (props) => {
    const [state, actions] = useApp();
    const [currentPath, setCurrentPath] = createSignal(props.initialPath || '/');
    const [files, setFiles] = createSignal<app.FileInfo[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [selectedItems, setSelectedItems] = createSignal<string[]>([]);
    const [contextMenu, setContextMenu] = createSignal<{ x: number, y: number, item: app.FileInfo | null } | null>(null);
    const [showDiffMode, setShowDiffMode] = createSignal(false);
    const [isOver, setIsOver] = createSignal(false);
    const [panelWidth, setPanelWidth] = createSignal(380);

    const isLocal = () => {
        const session = state.sessions.find(s => s.id === props.sessionId);
        return session?.hostId === 'local';
    };

    const provider = {
        list: (path: string) => ListDirectory(props.sessionId, path),
        mkdir: (path: string) => Mkdir(props.sessionId, path),
        rename: (old: string, next: string) => Rename(props.sessionId, old, next),
        remove: (path: string) => Remove(props.sessionId, path),
        download: (sessionId: string, path: string, dest: string, size: number) => Download(sessionId, path, dest, size),
        upload: (sessionId: string, local: string, dest: string) => Upload(sessionId, local, dest),
        readFile: (path: string) => ReadFile(props.sessionId, path),
        writeFile: (path: string, content: string) => WriteFile(props.sessionId, path, content),
    };

    const loadDirectory = async (path: string) => {
        setLoading(true);
        try {
            const result = await provider.list(path);
            const sorted = result.sort((a: app.FileInfo, b: app.FileInfo) => {
                if (a.is_dir && !b.is_dir) return -1;
                if (!a.is_dir && b.is_dir) return 1;
                return a.name.localeCompare(b.name);
            });
            setFiles(sorted);
            setCurrentPath(path);
            setSelectedItems([]);
        } catch (err: unknown) {
            actions.notify(`Failed to list directory: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
        } finally {
            setLoading(false);
        }
    };

    createEffect(() => {
        if (props.sessionId) {
            loadDirectory(currentPath());
        }
    });

    const toggleSelection = (itemName: string) => {
        setSelectedItems(prev => {
            if (prev.includes(itemName)) {
                return prev.filter(name => name !== itemName);
            } else {
                return [...prev, itemName];
            }
        });
    };

    const handleDownload = async (item: app.FileInfo) => {
        if (item.is_dir) return;
        try {
            // @ts-ignore
            const localPath = await window.runtime.SaveFileDialog({
                Title: `Save ${item.name}`,
                DefaultFilename: item.name
            });
            if (!localPath) return;

            const sep = currentPath().endsWith('/') ? '' : '/';
            const remotePath = currentPath() === '~' ? item.name : `${currentPath()}${sep}${item.name}`;

            setLoading(true);
            const jobID = await provider.download(props.sessionId, remotePath, localPath, item.size);
            actions.notify(`Starting download: ${item.name}`, 'info');
            actions.updateTransfer({
                id: jobID,
                name: item.name,
                type: 'download',
                status: 'active',
                progress: 0,
                size: item.size
            });
        } catch (err: unknown) {
            actions.notify(`Download failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
        } finally {
            setLoading(false);
        }
    };

    const handleUpload = async () => {
        try {
            // @ts-ignore
            const filePaths = await window.runtime.OpenFileDialog({
                Title: 'Select File to Upload'
            });
            if (!filePaths || (Array.isArray(filePaths) && filePaths.length === 0)) return;

            const localPath = Array.isArray(filePaths) ? filePaths[0] : filePaths;
            const sep = currentPath().endsWith('/') ? '' : '/';

            const parts = localPath.split(/[/\\]/);
            const fileName = parts[parts.length - 1];
            const remoteDestPath = currentPath() === '~' ? fileName : `${currentPath()}${sep}${fileName}`;

            setLoading(true);
            const jobID = await provider.upload(props.sessionId, localPath, remoteDestPath);
            actions.notify(`Starting upload: ${fileName}`, 'info');
            actions.updateTransfer({
                id: jobID,
                name: fileName,
                type: 'upload',
                status: 'active',
                progress: 0,
                size: 0
            });
        } catch (err: unknown) {
            actions.notify(`Upload failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
        } finally {
            setLoading(false);
        }
    };

    const handleItemDblClick = (item: app.FileInfo) => {
        if (item.is_dir) {
            const separator = currentPath().endsWith('/') ? '' : '/';
            const nextPath = currentPath() === '~' ? item.name : `${currentPath()}${separator}${item.name}`;
            loadDirectory(nextPath);
        } else {
            handleDownload(item);
        }
    };

    const handleContextMenu = (e: MouseEvent, item: app.FileInfo | null) => {
        e.preventDefault();
        if (item) toggleSelection(item.name);
        setContextMenu({ x: e.clientX, y: e.clientY, item });
    };

    const closeContextMenu = () => setContextMenu(null);

    const handleRename = async () => {
        const item = contextMenu()?.item;
        if (!item) return;
        const newName = prompt(`Rename ${item.name} to:`, item.name);
        if (newName && newName !== item.name) {
            try {
                const sep = currentPath().endsWith('/') ? '' : '/';
                const oldPath = currentPath() === '~' ? item.name : `${currentPath()}${sep}${item.name}`;
                const newPath = currentPath() === '~' ? newName : `${currentPath()}${sep}${newName}`;
                await provider.rename(oldPath, newPath);
                actions.notify(`Renamed ${item.name} to ${newName}`, 'success');
                loadDirectory(currentPath());
            } catch (err: unknown) {
                actions.notify(`Rename failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
            }
        }
        closeContextMenu();
    };

    const handleDelete = async () => {
        const item = contextMenu()?.item;
        if (!item) return;
        if (confirm(`Are you sure you want to delete ${item.name}?`)) {
            try {
                const sep = currentPath().endsWith('/') ? '' : '/';
                const path = currentPath() === '~' ? item.name : `${currentPath()}${sep}${item.name}`;
                await provider.remove(path);
                actions.notify(`Deleted ${item.name}`, 'success');
                loadDirectory(currentPath());
            } catch (err: unknown) {
                actions.notify(`Delete failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
            }
        }
        closeContextMenu();
    };

    const handleMkdir = async () => {
        const name = prompt("New Folder Name:");
        if (name) {
            try {
                const sep = currentPath().endsWith('/') ? '' : '/';
                const path = currentPath() === '~' ? name : `${currentPath()}${sep}${name}`;
                await provider.mkdir(path);
                actions.notify(`Created folder: ${name}`, 'success');
                loadDirectory(currentPath());
            } catch (err: unknown) {
                actions.notify(`Mkdir failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
            }
        }
        closeContextMenu();
    };

    const onDragStart = (e: DragEvent, item: app.FileInfo) => {
        if (!e.dataTransfer) return;
        const sep = currentPath().endsWith('/') ? '' : '/';
        const fullPath = currentPath() === '~' ? item.name : `${currentPath()}${sep}${item.name}`;

        const data = {
            sessionId: props.sessionId,
            path: fullPath,
            name: item.name,
            size: item.size,
            isDir: item.is_dir,
            isLocal: isLocal()
        };

        e.dataTransfer.setData('application/oblivra-file', JSON.stringify(data));
        e.dataTransfer.effectAllowed = 'copyMove';
    };

    const onDragOver = (e: DragEvent) => {
        e.preventDefault();
        setIsOver(true);
        if (e.dataTransfer) {
            e.dataTransfer.dropEffect = 'copy';
        }
    };

    const onDrop = async (e: DragEvent) => {
        e.preventDefault();
        setIsOver(false);
        const rawData = e.dataTransfer?.getData('application/oblivra-file');
        if (!rawData) return;

        try {
            const source = JSON.parse(rawData);
            if (source.sessionId === props.sessionId) return;

            const targetSep = currentPath().endsWith('/') ? '' : '/';
            const destPath = currentPath() === '~' ? source.name : `${currentPath()}${targetSep}${source.name}`;

            setLoading(true);

            if (source.isLocal && !isLocal()) {
                const jobID = await provider.upload(props.sessionId, source.path, destPath);
                actions.notify(`Starting upload: ${source.name}`, 'info');
                actions.updateTransfer({
                    id: jobID,
                    name: source.name,
                    type: 'upload',
                    status: 'active',
                    progress: 0,
                    size: source.size || 0
                });
            } else if (!source.isLocal && isLocal()) {
                const jobID = await Download(source.sessionId, source.path, destPath, source.size || 0);
                actions.notify(`Starting download: ${source.name}`, 'info');
                actions.updateTransfer({
                    id: jobID,
                    name: source.name,
                    type: 'download',
                    status: 'active',
                    progress: 0,
                    size: source.size || 0
                });
            } else if (!source.isLocal && !isLocal()) {
                actions.notify("Remote-to-remote transfer via Drag & Drop is not yet supported.", "warning");
            }
        } catch (err: unknown) {
            actions.notify(`Transfer failed: ${err instanceof Error ? (err as Error).message : String(err)}`, 'error');
        } finally {
            setLoading(false);
        }
    };

    const formatSize = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    const getPathParts = () => {
        const path = currentPath();
        if (path === '~') return [{ name: '~', path: '~' }];
        const parts = path.split('/').filter(Boolean);
        const result = [{ name: '/', path: '/' }];
        let current = '';
        for (const p of parts) {
            current += '/' + p;
            result.push({ name: p, path: current });
        }
        return result;
    };

    const onGlobalClick = () => closeContextMenu();

    // Resizing logic
    let isResizing = false;
    const startResizing = () => {
        isResizing = true;
        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', stopResizing);
    };
    const handleMouseMove = (e: MouseEvent) => {
        if (!isResizing) return;
        const newWidth = e.clientX; // Assuming it's on the left side
        if (newWidth > 200 && newWidth < 800) {
            setPanelWidth(newWidth);
        }
    };
    const stopResizing = () => {
        isResizing = false;
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', stopResizing);
    };

    window.addEventListener('click', onGlobalClick);
    onCleanup(() => window.removeEventListener('click', onGlobalClick));

    return (
        <div
            class="file-browser"
            style={{ width: `${panelWidth()}px` }}
            onContextMenu={(e) => { e.preventDefault(); if (!loading() && !showDiffMode()) setContextMenu({ x: e.clientX, y: e.clientY, item: null }); }}
        >
            <div class="fb-resizer" onMouseDown={startResizing}></div>
            <Show when={showDiffMode()}>
                <FolderDiff sessionId={props.sessionId} onClose={() => setShowDiffMode(false)} />
            </Show>

            <Show when={!showDiffMode()}>
                <div class="fb-header">
                    <div class="fb-toolbar">
                        <button onClick={() => loadDirectory('..')} class="fb-icon-btn" title="Back">←</button>
                        <button onClick={() => loadDirectory(currentPath())} class="fb-icon-btn" title="Refresh">↻</button>
                        <button onClick={handleMkdir} class="fb-icon-btn" title="New Folder">+</button>
                        <button onClick={handleUpload} class="fb-icon-btn" title="Upload File">↑</button>
                        <div style="width: 1px; height: 16px; background: var(--border-subtle); margin: 0 4px;" />
                        <button
                            onClick={() => setShowDiffMode(true)}
                            class="fb-icon-btn"
                            title="Compare with Local Folder"
                            style="color: var(--primary-color);"
                        >≂</button>
                        <div style="flex: 1"></div>
                        <button onClick={props.onClose} class="fb-icon-btn" title="Close">✕</button>
                    </div>
                    <div class="fb-breadcrumbs">
                        <For each={getPathParts()}>
                            {(part, i) => (
                                <>
                                    <span class="fb-breadcrumb-item" onClick={() => loadDirectory(part.path)}>{part.name}</span>
                                    <Show when={i() < getPathParts().length - 1}>
                                        <span class="fb-separator">/</span>
                                    </Show>
                                </>
                            )}
                        </For>
                    </div>
                </div>

                <div
                    class={`fb-content ${isOver() ? 'drag-over' : ''}`}
                    onDragOver={onDragOver}
                    onDragLeave={() => setIsOver(false)}
                    onDrop={onDrop}
                >
                    <Show when={loading()}>
                        <div class="fb-loading">
                            <For each={[1, 2, 3, 4, 5, 6, 7, 8]}>
                                {() => <div class="skeleton-row" style={{ width: `${Math.random() * 40 + 60}%` }}></div>}
                            </For>
                        </div>
                    </Show>

                    <Show when={!loading()}>
                        <div class="fb-list">
                            <For each={files()}>
                                {(item) => (
                                    <div
                                        class={`fb-item ${selectedItems().includes(item.name) ? 'selected' : ''}`}
                                        draggable={!item.is_dir}
                                        onDragStart={(e) => onDragStart(e, item)}
                                        onClick={() => toggleSelection(item.name)}
                                        onDblClick={() => handleItemDblClick(item)}
                                        onContextMenu={(e) => handleContextMenu(e, item)}
                                    >
                                        <span class="fb-icon">{item.is_dir ? '📁' : '📄'}</span>
                                        <span class="fb-name">{item.name}</span>
                                        <Show when={!item.is_dir}>
                                            <span class="fb-size">{formatSize(item.size)}</span>
                                        </Show>
                                    </div>
                                )}
                            </For>
                            <Show when={files().length === 0}>
                                <div class="fb-empty">No files found</div>
                            </Show>
                        </div>
                    </Show>
                </div>

                <Show when={contextMenu()}>
                    <div
                        class="fb-context-menu"
                        style={{ left: `${contextMenu()!.x}px`, top: `${contextMenu()!.y}px` }}
                        onClick={(e) => e.stopPropagation()}
                    >
                        <Show when={contextMenu()?.item}>
                            <Show when={!contextMenu()!.item!.is_dir}>
                                <div class="cm-item" onClick={() => handleDownload(contextMenu()!.item!)}>Download</div>
                            </Show>
                            <div class="cm-item" onClick={handleRename}>Rename</div>
                            <div class="cm-item danger" onClick={handleDelete}>Delete</div>
                        </Show>
                        <Show when={!contextMenu()?.item}>
                            <div class="cm-item" onClick={handleMkdir}>New Folder</div>
                            <div class="cm-item" onClick={() => loadDirectory(currentPath())}>Refresh</div>
                        </Show>
                    </div>
                </Show>
            </Show>
        </div>
    );
};
