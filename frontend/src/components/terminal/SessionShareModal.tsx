import { Component, createSignal, Show, onMount, onCleanup, For } from 'solid-js';
import { CreateShare, GetSharesBySession, RevokeShare, GetViewers } from '../../../wailsjs/go/services/ShareService';

export const SessionShareModal: Component<{ sessionId: string; hostLabel: string; onClose: () => void }> = (props) => {
    const [mode, setMode] = createSignal<string>('observe');
    const [maxViewers, setMaxViewers] = createSignal(5);
    const [expires, setExpires] = createSignal(60);
    const [shareUrl, setShareUrl] = createSignal<string | null>(null);
    const [activeShareId, setActiveShareId] = createSignal<string | null>(null);
    const [viewers, setViewers] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);

    const checkExisting = async () => {
        try {
            const shares = await GetSharesBySession(props.sessionId);
            if (shares && shares.length > 0) {
                const active = shares[0]; // Take first active for now
                setActiveShareId(active.id);
                // We reconstruct the URL for display if we had it, or just show ID
                setShareUrl(`sovereign://${active.id}`);
            }
        } catch (err) {
            console.error("Failed to check existing shares:", err);
        }
    };

    const pollViewers = async () => {
        if (!activeShareId()) return;
        try {
            const v = await GetViewers(activeShareId()!);
            setViewers(v || []);
        } catch (err) {
            console.error("Failed to poll viewers:", err);
        }
    };

    let pollInterval: any;

    onMount(() => {
        checkExisting();
        pollInterval = setInterval(pollViewers, 3000);
    });

    onCleanup(() => {
        clearInterval(pollInterval);
    });

    const handleCreate = async () => {
        setLoading(true);
        setError(null);
        try {
            const backendMode = mode() === 'collaborate' ? 'read_write' : 'observe';
            const url = await CreateShare(
                props.sessionId,
                props.hostLabel,
                backendMode,
                "admin", // TODO: real user ID
                expires(),
                maxViewers()
            );
            setShareUrl(url);
            // Extract ID from URL sovereign://<id>?...
            const id = url.split('://')[1]?.split('?')[0];
            if (id) setActiveShareId(id);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleRevoke = async () => {
        if (!activeShareId()) return;
        setLoading(true);
        try {
            await RevokeShare(activeShareId()!);
            setShareUrl(null);
            setActiveShareId(null);
            setViewers([]);
            alert("Session share revoked.");
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    const handleCopy = () => {
        if (shareUrl()) {
            navigator.clipboard.writeText(shareUrl()!);
            alert("Share link copied to clipboard!");
        }
    };

    return (
        <div class="modal-backdrop" style="position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 2000;">
            <div class="modal-content glass-surface" style="padding: 24px; width: 440px; border-radius: 12px;">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                    <h2 style="margin: 0; font-size: 18px; color: var(--text-primary);">Share Session</h2>
                    <button onClick={props.onClose} class="close-btn" style="border: none; background: transparent; color: var(--text-muted); cursor: pointer; font-size: 18px;">✕</button>
                </div>

                <Show when={error()}>
                    <div class="error-badge" style="background: rgba(255, 69, 58, 0.1); color: #FF453A; padding: 10px; border-radius: 6px; margin-bottom: 15px; font-size: 13px; border: 1px solid rgba(255, 69, 58, 0.2);">
                        {error()}
                    </div>
                </Show>

                <Show
                    when={!shareUrl()}
                    fallback={
                        <div style="display: flex; flex-direction: column; gap: 20px;">
                            <div class="share-url-box" style="padding: 16px; background: rgba(255,255,255,0.05); border: 1px dashed var(--glass-border); border-radius: 8px; position: relative;">
                                <div style="font-size: 11px; color: var(--text-muted); margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.05em;">Share Link</div>
                                <div style="word-break: break-all; font-family: var(--font-mono); font-size: 12px; color: var(--accent-primary); line-height: 1.4;">
                                    {shareUrl()}
                                </div>
                            </div>

                            <div class="viewers-section">
                                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                                    <h3 style="font-size: 13px; margin: 0; color: var(--text-secondary);">Active Viewers ({viewers().length})</h3>
                                    <Show when={viewers().length > 0}>
                                        <div class="pulse-dot online"></div>
                                    </Show>
                                </div>

                                <div class="viewers-list" style="max-height: 120px; overflow-y: auto; background: rgba(0,0,0,0.2); border-radius: 8px; padding: 4px;">
                                    <Show when={viewers().length === 0}>
                                        <div style="padding: 20px; text-align: center; color: var(--text-muted); font-size: 12px;">
                                            No active viewers yet
                                        </div>
                                    </Show>
                                    <For each={viewers()}>
                                        {(viewer) => (
                                            <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 12px; border-bottom: 1px solid rgba(255,255,255,0.05);">
                                                <div style="display: flex; align-items: center; gap: 8px;">
                                                    <div style="width: 24px; height: 24px; border-radius: 50%; background: var(--accent-primary); display: flex; align-items: center; justify-content: center; font-size: 10px; color: black; font-weight: bold;">
                                                        {viewer.name.charAt(0).toUpperCase()}
                                                    </div>
                                                    <span style="font-size: 13px;">{viewer.name}</span>
                                                </div>
                                                <span style="font-size: 11px; color: var(--text-muted);">
                                                    Joined {new Date(viewer.joined_at).toLocaleTimeString()}
                                                </span>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>

                            <div style="display: flex; gap: 12px;">
                                <button onClick={handleCopy} class="action-btn primary" style="flex: 1; height: 38px;">Copy Link</button>
                                <button onClick={handleRevoke} class="action-btn" style="flex: 1; height: 38px; border-color: rgba(255,69,58,0.3); color: #FF453A;">Revoke All</button>
                            </div>
                        </div>
                    }
                >
                    <div style="display: flex; flex-direction: column; gap: 18px;">
                        <div class="input-group">
                            <label style="display: block; margin-bottom: 8px; font-size: 12px; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em;">Access Privilege</label>
                            <select value={mode()} onChange={(e) => setMode(e.currentTarget.value)} class="input-primary" style="width: 100%; height: 40px; background: rgba(255,255,255,0.05); border: 1px solid var(--glass-border); border-radius: 8px; color: white; padding: 0 12px;">
                                <option value="observe">Observe Only (Read-Only)</option>
                                <option value="collaborate">Collaborate (Read/Write)</option>
                            </select>
                            <p style="margin: 6px 0 0; font-size: 11px; color: var(--text-muted);">
                                {mode() === 'collaborate' ? 'Viewers can type and send commands to your terminal.' : 'Viewers can only see your terminal output.'}
                            </p>
                        </div>

                        <div style="display: flex; gap: 16px;">
                            <div style="flex: 1;">
                                <label style="display: block; margin-bottom: 8px; font-size: 12px; color: var(--text-secondary);">MAX VIEWERS</label>
                                <input type="number" value={maxViewers()} onInput={(e) => setMaxViewers(parseInt(e.currentTarget.value) || 1)} min="1" max="10" class="input-primary" style="width: 100%; height: 40px; background: rgba(255,255,255,0.05); border: 1px solid var(--glass-border); border-radius: 8px; color: white; padding: 0 12px;" />
                            </div>
                            <div style="flex: 1;">
                                <label style="display: block; margin-bottom: 8px; font-size: 12px; color: var(--text-secondary);">EXPIRES (MINS)</label>
                                <input type="number" value={expires()} onInput={(e) => setExpires(parseInt(e.currentTarget.value) || 60)} min="5" max="1440" class="input-primary" style="width: 100%; height: 40px; background: rgba(255,255,255,0.05); border: 1px solid var(--glass-border); border-radius: 8px; color: white; padding: 0 12px;" />
                            </div>
                        </div>

                        <div style="display: flex; justify-content: flex-end; gap: 12px; margin-top: 10px;">
                            <button onClick={props.onClose} class="action-btn" style="height: 40px; padding: 0 20px;">Cancel</button>
                            <button onClick={handleCreate} class="action-btn primary" disabled={loading()} style="height: 40px; padding: 0 24px; font-weight: 600;">
                                {loading() ? 'Generating...' : 'Start Sharing'}
                            </button>
                        </div>
                    </div>
                </Show>
            </div>
        </div>
    );
};
