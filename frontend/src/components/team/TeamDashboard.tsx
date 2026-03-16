import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { ListMembers, ListSecrets, GetTeamName, AddMember } from '../../../wailsjs/go/services/TeamService';
import { team } from '../../../wailsjs/go/models';
import { Card, Button, Badge } from '../ui/TacticalComponents';
import { showModal } from '../ui/ModalSystem';
import '../../styles/team.css';

export const TeamDashboard: Component = () => {
    const [, actions] = useApp();
    const [members, setMembers] = createSignal<team.TeamMember[]>([]);
    const [secrets, setSecrets] = createSignal<team.VaultEntry[]>([]);
    const [teamName, setTeamName] = createSignal('Personal');
    const [, setLoading] = createSignal(true);

    const loadData = async () => {
        setLoading(true);
        try {
            const [m, s, name] = await Promise.all([
                ListMembers(),
                ListSecrets(),
                GetTeamName()
            ]);
            setMembers(m || []);
            setSecrets(s || []);
            setTeamName(name || 'Personal');
        } catch (err) {
            console.error("Failed to load team data:", err);
            actions.notify("Failed to load team data", "error");
        } finally {
            setLoading(false);
        }
    };

    onMount(loadData);

    const handleInvite = async () => {
        showModal({
            title: "INVITE_OPERATOR",
            message: "Enter the email of the operator to authorize:",
            showInput: true,
            inputPlaceholder: "operator@oblivra.sovereign",
            confirmText: "CONTINUE_TO_NAME",
            onCancel: () => { },
            onConfirm: (email) => {
                if (!email) return;
                showModal({
                    title: "AUTHORIZE_OPERATOR",
                    message: `Provide a tactical callsign for ${email}:`,
                    showInput: true,
                    inputPlaceholder: "Callsign...",
                    confirmText: "INVITE",
                    onCancel: () => { },
                    onConfirm: async (name) => {
                        if (!name) return;
                        try {
                            await AddMember(email, name, "member");
                            actions.notify(`Invited ${name} to team`, "success");
                            loadData();
                        } catch (err) {
                            actions.notify(`Invite failed: ${(err as Error).message}`, "error");
                        }
                    }
                });
            }
        });
    };

    return (
        <div class="team-dashboard animate-fade-in">
            <header class="team-header" style="border-bottom: 1px solid var(--border-primary); padding: 20px;">
                <div class="team-title-group">
                    <h1 class="host-name" style="font-family: var(--font-ui); font-weight: 800; letter-spacing: 1px;">{teamName().toUpperCase()}</h1>
                    <p class="team-subtitle">OPERATOR_ORCHESTRATION & SHARED_GOVERNANCE_PROTOCOL</p>
                </div>
                <div class="team-actions">
                    <Button variant="primary" size="sm" onClick={handleInvite}>
                        INVITE_OPERATOR
                    </Button>
                </div>
            </header>

            <div class="team-grid" style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px; padding: 20px;">
                <Card variant="raised">
                    <div class="team-card-header" style="margin-bottom: 20px; border-bottom: 1px solid var(--border-primary); padding-bottom: 10px;">
                        <h2 class="team-card-title" style="font-size: 14px; color: var(--text-muted);">ACTIVE OPERATORS ({members().length})</h2>
                    </div>
                    <div class="member-list" style="display: flex; flex-direction: column; gap: 12px;">
                        <For each={members()}>
                            {(member) => (
                                <div class="member-item" style="display: flex; align-items: center; gap: 12px; padding: 8px; background: rgba(255,255,255,0.02); border-radius: var(--radius-sm);">
                                    <div class="member-avatar" style="width: 32px; height: 32px; background: var(--accent-secondary); border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: 700;">
                                        {member.name.charAt(0).toUpperCase()}
                                    </div>
                                    <div class="member-info" style="flex: 1;">
                                        <div class="member-name" style="font-weight: 700; font-size: 12px;">{member.name.toUpperCase()} {member.id === "local-user" ? "(ROOT)" : ""}</div>
                                        <div class="member-email" style="font-size: 11px; color: var(--text-muted);">{member.email}</div>
                                    </div>
                                    <Badge severity={member.role === 'admin' ? 'error' : 'info'}>
                                        {member.role}
                                    </Badge>
                                </div>
                            )}
                        </For>
                    </div>
                </Card>

                <Card variant="raised">
                    <div class="team-card-header" style="margin-bottom: 20px; border-bottom: 1px solid var(--border-primary); padding-bottom: 10px;">
                        <h2 class="team-card-title" style="font-size: 14px; color: var(--text-muted);">SHARED TACTICAL VAULT</h2>
                    </div>
                    <div class="vault-list" style="display: flex; flex-direction: column; gap: 8px;">
                        <For each={secrets()}>
                            {(secret) => (
                                <div class="vault-item" style="display: flex; align-items: center; gap: 12px; padding: 8px; border-bottom: 1px dashed var(--border-primary);">
                                    <Badge severity="info" style="font-family: var(--font-mono); font-size: 9px;">
                                        {secret.entry_type.replace('_', ' ')}
                                    </Badge>
                                    <div class="vault-details" style="flex: 1;">
                                        <div class="vault-title" style="font-weight: 700; font-size: 12px;">{secret.title.toUpperCase()}</div>
                                        <div class="vault-meta" style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                                            INITIALIZED: {new Date(secret.created_at.toString()).toISOString().replace('T', ' ').slice(0, 10)} // OP: {secret.created_by.toUpperCase()}
                                        </div>
                                    </div>
                                </div>
                            )}
                        </For>
                        <Show when={secrets().length === 0}>
                            <div style="padding: 40px; text-align: center; color: var(--text-muted);">
                                NO_SHARED_SECRETS_IN_BUFFER
                            </div>
                        </Show>
                    </div>
                </Card>

                <Card variant="raised" style="grid-column: span 2;">
                    <div class="team-card-header" style="margin-bottom: 20px; border-bottom: 1px solid var(--border-primary); padding-bottom: 10px;">
                        <h2 class="team-card-title" style="font-size: 14px; color: var(--text-muted);">ESCALATION ACTIVITY AUDIT LOG</h2>
                    </div>
                    <div class="activity-feed" style="display: flex; flex-direction: column; gap: 16px;">
                        <div class="activity-item" style="display: flex; gap: 12px;">
                            <div class="activity-dot" style="width: 8px; height: 8px; background: var(--accent-secondary); border-radius: 50%; margin-top: 4px;"></div>
                            <div class="activity-content">
                                <span style="font-weight: 700; color: var(--accent-secondary); font-family: var(--font-mono);">SYSTEM</span> INITIALIZED_GOVERNANCE_VAULT: <span style="font-weight: 700;">{teamName().toUpperCase()}</span>
                                <div class="activity-time" style="font-size: 10px; color: var(--text-muted); margin-top: 4px;">2026-02-28 10:24:00</div>
                            </div>
                        </div>
                        <Show when={members().length > 1}>
                            <div class="activity-item" style="display: flex; gap: 12px;">
                                <div class="activity-dot" style="width: 8px; height: 8px; background: var(--status-online); border-radius: 50%; margin-top: 4px;"></div>
                                <div class="activity-content">
                                    <span style="font-weight: 700;">{members()[1]?.name.toUpperCase()}</span> JOINED_ORCHESTRATION_LAYER
                                    <div class="activity-time" style="font-size: 10px; color: var(--text-muted); margin-top: 4px;">POLLING_INTERVAL_DELTA: 0S</div>
                                </div>
                            </div>
                        </Show>
                    </div>
                </Card>
            </div>
        </div>
    );
};

