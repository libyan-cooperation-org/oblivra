import { Component, createSignal, For } from 'solid-js';
import { database } from '../../../wailsjs/go/models';
import { Create, Update, ImportSSHConfig } from '../../../wailsjs/go/app/HostService';
import { useApp } from '@core/store';
import { z } from 'zod';

const hostSchema = z.object({
    label: z.string().optional(),
    category: z.string().optional(),
    hostname: z.string().min(1, "Hostname or IP is required").refine(val => !val.includes(' '), "Hostname cannot contain spaces"),
    username: z.string().min(1, "Username is required"),
    port: z.number().int().min(1, "Port must be at least 1").max(65535, "Port cannot exceed 65535"),
    password: z.string().optional()
});

export const AddHostModal: Component<{ host?: database.Host, onClose: () => void; onHostAdded?: (host: database.Host) => void }> = (props) => {
    const [state, actions] = useApp();
    const isEdit = !!props.host;

    // Seed initial values from props.host if modifying an existing one
    const [label, setLabel] = createSignal(props.host?.label || '');
    const [category, setCategory] = createSignal(props.host?.category || '');
    const [hostname, setHostname] = createSignal(props.host?.hostname || '');
    const [username, setUsername] = createSignal(props.host?.username || '');
    const [password, setPassword] = createSignal(props.host?.password || '');
    const [port, setPort] = createSignal(props.host?.port?.toString() || '22');
    const [jumpHostId, setJumpHostId] = createSignal(props.host?.jump_host_id || '');
    const [saving, setSaving] = createSignal(false);
    const [fieldErrors, setFieldErrors] = createSignal<Record<string, string>>({});
    const [generalError, setGeneralError] = createSignal('');

    const handleSave = async () => {
        setFieldErrors({});
        setGeneralError('');

        const portNum = parseInt(port(), 10);
        const parseResult = hostSchema.safeParse({
            label: label(),
            category: category(),
            hostname: hostname(),
            username: username(),
            port: isNaN(portNum) ? 0 : portNum,
            password: password()
        });

        if (!parseResult.success) {
            const errors: Record<string, string> = {};
            parseResult.error.issues.forEach(issue => {
                if (issue.path[0]) {
                    errors[issue.path[0].toString()] = issue.message;
                }
            });
            setFieldErrors(errors);
            return;
        }

        setSaving(true);

        try {
            const hostData = database.Host.createFrom({
                id: isEdit ? props.host!.id : '',
                label: parseResult.data.label || parseResult.data.hostname,
                hostname: parseResult.data.hostname,
                username: parseResult.data.username,
                password: parseResult.data.password || '',
                port: parseResult.data.port,
                auth_method: 'password', // Default
                tags: isEdit ? props.host!.tags : [],
                category: category(),
                color: isEdit ? props.host!.color : '#58a6ff',
                notes: isEdit ? props.host!.notes : '',
                is_favorite: isEdit ? props.host!.is_favorite : false,
                connection_count: isEdit ? props.host!.connection_count : 0,
                jump_host_id: jumpHostId()
            });

            const savedHost = isEdit ? await Update(hostData) : await Create(hostData);

            if (isEdit) {
                // To trigger UI re-render, we need to completely replace the array
                const updatedHosts = state.hosts.map((h: any) => h.id === savedHost.id ? {
                    ...savedHost,
                    auth_method: savedHost.auth_method || 'password',
                    is_favorite: savedHost.is_favorite || false,
                    connection_count: savedHost.connection_count || 0
                } : h) as database.Host[];
                actions.setHosts(updatedHosts);
            } else {
                actions.addHost({
                    ...savedHost,
                    auth_method: savedHost.auth_method || 'password',
                    is_favorite: savedHost.is_favorite || false,
                } as any);
            }

            if (props.onHostAdded) {
                props.onHostAdded(savedHost);
            }
            props.onClose();
        } catch (err) {
            setGeneralError((err as Error).message || String(err));
        } finally {
            setSaving(false);
        }
    };

    return (
        <div class="modal-backdrop" style="position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 2000;">
            <div class="modal-content" style="background: var(--bg-primary); padding: 20px; border-radius: 8px; width: 400px; max-width: 90vw; border: 1px solid var(--border-primary);">
                <h3 style="margin-top: 0; margin-bottom: 20px;">{isEdit ? 'Edit Connection' : 'New Connection'}</h3>

                <div style="display: flex; flex-direction: column; gap: 15px;">
                    <div style="display: flex; gap: 10px;">
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Label (Optional)</label>
                            <input
                                type="text"
                                value={label()}
                                onInput={(e) => setLabel(e.currentTarget.value)}
                                placeholder="My Server"
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px;"
                            />
                        </div>
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Category (Folder)</label>
                            <input
                                type="text"
                                value={category()}
                                onInput={(e) => setCategory(e.currentTarget.value)}
                                placeholder="infrastructure"
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px;"
                            />
                        </div>
                    </div>

                    <div>
                        <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Hostname / IP *</label>
                        <input
                            type="text"
                            value={hostname()}
                            onInput={(e) => setHostname(e.currentTarget.value)}
                            placeholder="example.com or 192.168.1.100"
                            style={`width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid ${fieldErrors().hostname ? 'var(--error)' : 'var(--border-primary)'}; color: var(--text-primary); padding: 8px; border-radius: 4px;`}
                        />
                        {fieldErrors().hostname && <div style="color: var(--error); font-size: 11px; margin-top: 4px;">{fieldErrors().hostname}</div>}
                    </div>

                    <div style="display: flex; gap: 10px;">
                        <div style="flex: 2;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Identity Profile</label>
                            <select
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px; appearance: none;"
                            >
                                <option value="default">Default Profile</option>
                                <option value="personal">Personal (id_rsa)</option>
                                <option value="work">Work (ed25519) + JumpHost</option>
                                <option value="custom">-- Custom Connection --</option>
                            </select>
                        </div>
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Port *</label>
                            <input
                                type="number"
                                value={port()}
                                onInput={(e) => setPort(e.currentTarget.value)}
                                placeholder="22"
                                style={`width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid ${fieldErrors().port ? 'var(--error)' : 'var(--border-primary)'}; color: var(--text-primary); padding: 8px; border-radius: 4px;`}
                            />
                            {fieldErrors().port && <div style="color: var(--error); font-size: 11px; margin-top: 4px;">{fieldErrors().port}</div>}
                        </div>
                    </div>

                    <div style="display: flex; gap: 10px; margin-top: 5px;">
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Username *</label>
                            <input
                                type="text"
                                value={username()}
                                onInput={(e) => setUsername(e.currentTarget.value)}
                                placeholder="root"
                                style={`width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid ${fieldErrors().username ? 'var(--error)' : 'var(--border-primary)'}; color: var(--text-primary); padding: 8px; border-radius: 4px;`}
                            />
                            {fieldErrors().username && <div style="color: var(--error); font-size: 11px; margin-top: 4px;">{fieldErrors().username}</div>}
                        </div>
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Auth Method</label>
                            <select
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px; appearance: none;"
                            >
                                <option value="password">Password</option>
                                <option value="key">SSH Key</option>
                                <option value="agent">SSH Agent</option>
                            </select>
                        </div>
                    </div>

                    <div style="display: flex; gap: 10px; margin-top: 5px;">
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Password</label>
                            <input
                                type="password"
                                value={password()}
                                onInput={(e) => setPassword(e.currentTarget.value)}
                                placeholder="Leave blank if using Key/Agent"
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px;"
                            />
                        </div>
                        <div style="flex: 1;">
                            <label style="display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-secondary);">Jump Host (Optional)</label>
                            <select
                                value={jumpHostId()}
                                onChange={(e) => setJumpHostId(e.currentTarget.value)}
                                style="width: 100%; box-sizing: border-box; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px; border-radius: 4px; appearance: none;"
                            >
                                <option value="">Direct Connection</option>
                                <For each={state.hosts.filter((h: database.Host) => h.id !== props.host?.id)}>
                                    {(h) => <option value={h.id}>{h.label || h.hostname}</option>}
                                </For>
                            </select>
                        </div>
                    </div>

                    {generalError() && (
                        <div style="color: var(--error); font-size: 12px; padding: 10px; background: rgba(248, 81, 73, 0.1); border-radius: 4px; border: 1px solid var(--error);">
                            {generalError()}
                        </div>
                    )}

                    <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 10px;">
                        <button
                            class="action-btn"
                            onClick={async () => {
                                try {
                                    setSaving(true);
                                    const count = await ImportSSHConfig();
                                    alert(`Successfully imported ${count} hosts from ~/.ssh/config!`);
                                    props.onClose();
                                } catch (err) {
                                    setGeneralError((err as Error).message || String(err));
                                } finally {
                                    setSaving(false);
                                }
                            }}
                            title="Auto-import hosts from your local SSH config"
                            disabled={saving()}
                        >
                            📦 Bulk Import ~/.ssh/config
                        </button>
                        <div style="display: flex; gap: 10px;">
                            <button
                                onClick={props.onClose}
                                style="background: transparent; border: 1px solid var(--border-primary); color: var(--text-primary); padding: 8px 16px; border-radius: 4px; cursor: pointer;"
                                disabled={saving()}
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleSave}
                                style="background: var(--accent-primary); border: none; color: white; padding: 8px 16px; border-radius: 4px; cursor: pointer;"
                                disabled={saving() || !hostname()}
                            >
                                {saving() ? 'Saving...' : 'Save Connection'}
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};
