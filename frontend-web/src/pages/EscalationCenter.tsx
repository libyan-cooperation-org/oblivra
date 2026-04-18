/**
 * EscalationCenter.tsx — Escalation Policy Management (Web Only — Phase 2.1.5)
 *
 * Admin console for creating multi-level escalation chains, managing on-call
 * rotation schedules, monitoring active escalations, and reviewing SLA breaches.
 * Connects to /api/v1/escalation/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface EscalationLevel {
  level: number;
  name: string;
  users: string[];
  channel: string;
  wait_mins: number;
}
interface EscalationPolicy {
  id: string;
  name: string;
  alert_types: string[];
  levels: EscalationLevel[];
  sla_mins: number;
  active: boolean;
}
interface ActiveEscalation {
  alert_id: string;
  policy_id: string;
  current_level: number;
  created_at: string;
  last_escalated_at: string;
  acked_by?: string;
  acked_at?: string;
  sla_breached: boolean;
  closed: boolean;
}
interface OnCallEntry {
  user_id: string;
  name: string;
  weekday_start: number;
  weekday_end: number;
  hour_start: number;
  hour_end: number;
}

// ── Helpers ───────────────────────────────────────────────────────────────────
const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const CH_COLOR: Record<string, string> = {
  slack: '#4A154B', email: '#00ffe7', webhook: '#ffaa00', sms: '#00ff88', teams: '#6264A7',
};

function msAgo(iso: string): string {
  const d = Math.floor((Date.now() - new Date(iso).getTime()) / 60000);
  if (d < 60) return `${d}m ago`;
  return `${Math.floor(d/60)}h ${d%60}m ago`;
}

// ── Component ─────────────────────────────────────────────────────────────────
type Tab = 'policies' | 'active' | 'oncall' | 'history';

export default function EscalationCenter() {
  const [tab, setTab] = createSignal<Tab>('policies');

  const [policies, { refetch: refetchPolicies }] = createResource<EscalationPolicy[]>(async () => {
    const r = await request<{ policies: EscalationPolicy[] }>('/escalation/policies');
    return r.policies ?? [];
  });

  const [active, { refetch: refetchActive }] = createResource<ActiveEscalation[]>(async () => {
    const r = await request<{ escalations: ActiveEscalation[] }>('/escalation/active');
    return r.escalations ?? [];
  });

  const [history] = createResource<ActiveEscalation[]>(async () => {
    const r = await request<{ escalations: ActiveEscalation[] }>('/escalation/history?limit=50');
    return r.escalations ?? [];
  });

  const [onCall] = createResource<{entries: OnCallEntry[]; current?: OnCallEntry}>(async () => {
    return await request('/escalation/oncall');
  });

  // Policy form state
  const [policyName, setPolicyName]   = createSignal('');
  const [slaMins, setSlaMins]         = createSignal(30);
  const [alertTypes, setAlertTypes]   = createSignal('security_alert,failed_login');
  const [saveMsg, setSaveMsg]         = createSignal('');

  async function savePolicy() {
    if (!policyName()) return;
    const policy: Partial<EscalationPolicy> = {
      id:          policyName().toLowerCase().replace(/\s+/g, '_'),
      name:        policyName(),
      sla_mins:    slaMins(),
      alert_types: alertTypes().split(',').map(s => s.trim()),
      active:      true,
      levels: [
        { level: 1, name: 'Analyst',  users: ['analyst@oblivra.io'],  channel: 'slack', wait_mins: 10 },
        { level: 2, name: 'Team Lead', users: ['lead@oblivra.io'],    channel: 'email', wait_mins: 15 },
        { level: 3, name: 'Manager',  users: ['manager@oblivra.io'],  channel: 'email', wait_mins: 20 },
        { level: 4, name: 'CISO',     users: ['ciso@oblivra.io'],     channel: 'sms',   wait_mins: 999 },
      ],
    };
    try {
      await request('/escalation/policies', { method: 'POST', body: JSON.stringify(policy) });
      setSaveMsg('✓ Policy saved.');
      refetchPolicies();
    } catch(e: any) {
      setSaveMsg(`✗ ${e.message}`);
    }
  }

  async function ackAlert(alertId: string) {
    const user = JSON.parse(localStorage.getItem('oblivra_user') ?? '{}');
    await request('/escalation/ack', {
      method: 'POST',
      body: JSON.stringify({ alert_id: alertId, user_id: user.id ?? 'unknown', comment: 'Acknowledged via web console' }),
    });
    refetchActive();
  }

  const TAB_STYLE = (active: boolean) =>
    `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${active ? '#ff6600' : 'transparent'}; background:none; color:${active ? '#ff6600' : '#607070'}; transition:color 0.15s;`;

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff6600;">⬡ ESCALATION CENTER</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
          Policy management · On-call scheduling · SLA tracking · Web Only
        </p>
      </div>

      {/* Stat strip */}
      <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
        {[
          { label: 'POLICIES',    val: policies()?.length ?? 0,                          color: '#ff6600' },
          { label: 'ACTIVE',      val: active()?.filter(a => !a.closed).length ?? 0,     color: '#ff3355' },
          { label: 'SLA BREACHED',val: active()?.filter(a => a.sla_breached).length ?? 0,color: '#ffaa00' },
          { label: 'ACKED',       val: history()?.length ?? 0,                            color: '#00ff88' },
        ].map(s => (
          <div style={`background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid ${s.color}; padding:1rem; border-radius:4px;`}>
            <div style={`font-size:1.6rem; font-weight:700; color:${s.color};`}>{s.val}</div>
            <div style="font-size:0.68rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Tabs */}
      <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.5rem;">
        {(['policies', 'active', 'oncall', 'history'] as Tab[]).map(t => (
          <button style={TAB_STYLE(tab() === t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
        ))}
      </div>

      {/* ── Policies Tab ── */}
      <Show when={tab() === 'policies'}>
        <div style="display:grid; grid-template-columns:1fr 340px; gap:1.5rem; align-items:start;">
          {/* Policy list */}
          <div style="display:flex; flex-direction:column; gap:1rem;">
            <Show when={!policies.loading} fallback={<div style="color:#607070;">Loading…</div>}>
              <For each={policies()} fallback={<div style="color:#607070;">No policies defined.</div>}>
                {(p) => (
                  <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
                    <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:0.75rem;">
                      <div>
                        <div style="color:#ff6600; font-size:0.88rem; letter-spacing:0.1em;">{p.name}</div>
                        <div style="color:#607070; font-size:0.7rem; margin-top:0.15rem;">
                          SLA: {p.sla_mins}min · Types: {p.alert_types?.join(', ')}
                        </div>
                      </div>
                      <span style={`font-size:0.65rem; color:${p.active ? '#00ff88' : '#ff3355'}; letter-spacing:0.1em;`}>
                        ● {p.active ? 'ACTIVE' : 'INACTIVE'}
                      </span>
                    </div>
                    <div style="display:flex; flex-direction:column; gap:0.35rem;">
                      <For each={p.levels ?? []}>
                        {(lvl) => (
                          <div style="display:flex; align-items:center; gap:0.75rem; padding:0.4rem 0.6rem; background:#0a1318; border-radius:3px; font-size:0.72rem;">
                            <span style="color:#607070; min-width:20px;">L{lvl.level}</span>
                            <span style="color:#c8d8d8; min-width:80px;">{lvl.name}</span>
                            <span style={`background:${CH_COLOR[lvl.channel] ?? '#1e3040'}22; border:1px solid ${CH_COLOR[lvl.channel] ?? '#1e3040'}; color:${CH_COLOR[lvl.channel] ?? '#607070'}; padding:0.1rem 0.4rem; border-radius:2px; font-size:0.65rem; letter-spacing:0.08em;`}>{lvl.channel.toUpperCase()}</span>
                            <span style="color:#607070; margin-left:auto;">{lvl.wait_mins < 999 ? `→ escalate after ${lvl.wait_mins}m` : 'terminal level'}</span>
                          </div>
                        )}
                      </For>
                    </div>
                  </div>
                )}
              </For>
            </Show>
          </div>

          {/* Create policy form */}
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem; position:sticky; top:1rem;">
            <div style="font-size:0.72rem; color:#607070; letter-spacing:0.12em; margin-bottom:1rem;">NEW POLICY</div>
            {[
              { label: 'POLICY NAME',  val: policyName(),   set: setPolicyName,  type: 'text',   placeholder: 'Critical Security' },
              { label: 'ALERT TYPES',  val: alertTypes(),   set: setAlertTypes,  type: 'text',   placeholder: 'security_alert,failed_login' },
            ].map(f => (
              <div style="margin-bottom:0.75rem;">
                <div style="font-size:0.65rem; color:#607070; margin-bottom:0.25rem;">{f.label}</div>
                <input type={f.type} value={f.val} onInput={e => f.set(e.currentTarget.value)}
                  placeholder={f.placeholder}
                  style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem; box-sizing:border-box;" />
              </div>
            ))}
            <div style="margin-bottom:0.75rem;">
              <div style="font-size:0.65rem; color:#607070; margin-bottom:0.25rem;">SLA (MINUTES)</div>
              <input type="number" value={slaMins()} onInput={e => setSlaMins(Number(e.currentTarget.value))} min="1" max="1440"
                style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem; box-sizing:border-box;" />
            </div>
            <div style="color:#607070; font-size:0.7rem; margin-bottom:0.75rem;">
              Levels auto-seeded: Analyst (10m) → Lead (15m) → Manager (20m) → CISO
            </div>
            <button onClick={savePolicy}
              style="width:100%; background:#ff6600; color:#080f12; border:none; padding:0.5rem; border-radius:3px; cursor:pointer; font-weight:700; font-size:0.78rem; letter-spacing:0.1em;">
              SAVE POLICY
            </button>
            <Show when={saveMsg()}>
              <div style={`margin-top:0.5rem; font-size:0.72rem; color:${saveMsg().startsWith('✓') ? '#00ff88' : '#ff3355'};`}>{saveMsg()}</div>
            </Show>
          </div>
        </div>
      </Show>

      {/* ── Active Tab ── */}
      <Show when={tab() === 'active'}>
        <Show when={!active.loading} fallback={<div style="color:#607070;">Loading…</div>}>
          <For each={active()?.filter(a => !a.closed)} fallback={
            <div style="padding:2rem; text-align:center; color:#607070;">No active escalations. All clear.</div>
          }>
            {(esc) => (
              <div style={`background:#0d1a1f; border:1px solid ${esc.sla_breached ? '#ffaa00' : '#1e3040'}; border-left:4px solid ${esc.sla_breached ? '#ffaa00' : '#ff3355'}; border-radius:6px; padding:1.25rem; margin-bottom:1rem;`}>
                <div style="display:flex; justify-content:space-between; align-items:flex-start;">
                  <div>
                    <div style="color:#c8d8d8; font-size:0.88rem; margin-bottom:0.25rem;">{esc.alert_id}</div>
                    <div style="color:#607070; font-size:0.72rem;">
                      Policy: <span style="color:#ff6600;">{esc.policy_id}</span>
                      &nbsp;·&nbsp;Level: <span style="color:#ff3355;">L{esc.current_level}</span>
                      &nbsp;·&nbsp;Started: {msAgo(esc.created_at)}
                    </div>
                    <Show when={esc.sla_breached}>
                      <div style="color:#ffaa00; font-size:0.7rem; margin-top:0.3rem; letter-spacing:0.1em;">⚠ SLA BREACHED</div>
                    </Show>
                  </div>
                  <button onClick={() => ackAlert(esc.alert_id)}
                    style="background:#00ff88; color:#080f12; border:none; padding:0.4rem 1rem; border-radius:4px; cursor:pointer; font-size:0.72rem; font-weight:700; letter-spacing:0.1em;">
                    ACKNOWLEDGE
                  </button>
                </div>
              </div>
            )}
          </For>
        </Show>
      </Show>

      {/* ── On-Call Tab ── */}
      <Show when={tab() === 'oncall'}>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; margin-bottom:1.5rem;">
          <div style="padding:0.75rem 1rem; background:#0a1318; border-bottom:1px solid #1e3040; display:flex; justify-content:space-between;">
            <span style="color:#00ffe7; font-size:0.8rem;">PRIMARY ON-CALL ROTATION</span>
            <Show when={onCall()?.current}>
              <span style="color:#00ff88; font-size:0.72rem;">
                ● NOW ON-CALL: {onCall()!.current!.name}
              </span>
            </Show>
          </div>
          <table style="width:100%; border-collapse:collapse; font-size:0.76rem;">
            <thead>
              <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                {['ENGINEER', 'DAYS', 'HOURS (UTC)'].map(h => (
                  <th style="padding:0.65rem 1rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <Show when={!onCall.loading} fallback={<tr><td colspan="3" style="padding:1rem; color:#607070;">Loading…</td></tr>}>
                <For each={onCall()?.entries ?? []}>
                  {(e) => (
                    <tr style="border-bottom:1px solid #0a1318;" onMouseEnter={el => (el.currentTarget as HTMLElement).style.background='#111f28'} onMouseLeave={el => (el.currentTarget as HTMLElement).style.background=''}>
                      <td style="padding:0.65rem 1rem; color:#c8d8d8;">{e.name}</td>
                      <td style="padding:0.65rem 1rem; color:#607070;">{DAYS[e.weekday_start]}–{DAYS[e.weekday_end]}</td>
                      <td style="padding:0.65rem 1rem; color:#607070;">{String(e.hour_start).padStart(2,'0')}:00 – {String(e.hour_end).padStart(2,'0')}:00</td>
                    </tr>
                  )}
                </For>
              </Show>
            </tbody>
          </table>
        </div>
      </Show>

      {/* ── History Tab ── */}
      <Show when={tab() === 'history'}>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
          <table style="width:100%; border-collapse:collapse; font-size:0.76rem;">
            <thead>
              <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                {['ALERT ID', 'POLICY', 'FINAL LEVEL', 'ACKED BY', 'ACKED AT', 'SLA'].map(h => (
                  <th style="padding:0.65rem 1rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <Show when={!history.loading} fallback={<tr><td colspan="6" style="padding:1rem; color:#607070;">Loading…</td></tr>}>
                <For each={history()} fallback={<tr><td colspan="6" style="padding:1rem; text-align:center; color:#607070;">No history yet.</td></tr>}>
                  {(h) => (
                    <tr style="border-bottom:1px solid #0a1318;" onMouseEnter={el => (el.currentTarget as HTMLElement).style.background='#111f28'} onMouseLeave={el => (el.currentTarget as HTMLElement).style.background=''}>
                      <td style="padding:0.65rem 1rem; color:#c8d8d8;">{h.alert_id}</td>
                      <td style="padding:0.65rem 1rem; color:#ff6600;">{h.policy_id}</td>
                      <td style="padding:0.65rem 1rem; color:#ff3355;">L{h.current_level}</td>
                      <td style="padding:0.65rem 1rem; color:#607070;">{h.acked_by || '—'}</td>
                      <td style="padding:0.65rem 1rem; color:#607070;">{h.acked_at ? new Date(h.acked_at).toLocaleString() : '—'}</td>
                      <td style="padding:0.65rem 1rem;">
                        <span style={`font-size:0.68rem; color:${h.sla_breached ? '#ffaa00' : '#00ff88'};`}>
                          {h.sla_breached ? '⚠ BREACHED' : '✓ OK'}
                        </span>
                      </td>
                    </tr>
                  )}
                </For>
              </Show>
            </tbody>
          </table>
        </div>
      </Show>
    </div>
  );
}
