/**
 * RegulatorPortal.tsx — Regulator-Ready Audit Export (Phase 6.6)
 *
 * Scoped, read-only audit viewer for external auditors.
 * One-click compliance package generation and export.
 * Connects to /api/v1/audit/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface AuditEntry {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  resource: string;
  outcome: 'success' | 'failure' | 'blocked';
  ip?: string;
  tenant_id?: string;
  entry_hash: string;
  prev_hash: string;
}
interface CompliancePackage {
  id: string;
  framework: string;
  generated_at: string;
  records: number;
  integrity_proof: string;
  download_url: string;
}

// ── Helpers ───────────────────────────────────────────────────────────────────
const OUTCOME_STYLE: Record<string, { color: string; bg: string }> = {
  success: { color: '#00ff88', bg: '#001a0a' },
  failure: { color: '#ff3355', bg: '#2a0d15' },
  blocked: { color: '#ffaa00', bg: '#2a2000' },
};

const FRAMEWORKS = ['ALL', 'SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'GDPR'];

// ── Component ─────────────────────────────────────────────────────────────────
type Tab = 'audit' | 'packages' | 'export';

export default function RegulatorPortal() {
  const [tab, setTab] = createSignal<Tab>('audit');
  const [framework, setFramework] = createSignal('ALL');
  const [dateFrom, setDateFrom] = createSignal('');
  const [dateTo, setDateTo] = createSignal('');
  const [generating, setGenerating] = createSignal(false);
  const [selectedFw, setSelectedFw] = createSignal('SOC2');

  const [auditLog, { refetch: refetchAudit }] = createResource<AuditEntry[]>(async () => {
    const params = new URLSearchParams({ limit: '200' });
    if (framework() !== 'ALL') params.set('framework', framework());
    if (dateFrom()) params.set('from', dateFrom());
    if (dateTo()) params.set('to', dateTo());
    const r = await request<{ entries: AuditEntry[] }>(`/audit/log?${params}`);
    return r.entries ?? [];
  });

  const [packages, { refetch: refetchPkgs }] = createResource<CompliancePackage[]>(async () => {
    const r = await request<{ packages: CompliancePackage[] }>('/audit/packages');
    return r.packages ?? [];
  });

  async function generatePackage() {
    setGenerating(true);
    try {
      await request('/audit/packages/generate', {
        method: 'POST',
        body: JSON.stringify({ framework: selectedFw(), from: dateFrom(), to: dateTo() }),
      });
      refetchPkgs();
    } finally { setGenerating(false); }
  }

  const TAB = (t: Tab) =>
    `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#0099ff' : 'transparent'}; background:none; color:${tab()===t ? '#0099ff' : '#607070'};`;

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
        <div>
          <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#0099ff;">⬡ REGULATOR PORTAL</h1>
          <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
            Read-only audit log · Compliance packages · Cryptographic integrity proofs
          </p>
        </div>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:4px; padding:0.5rem 1rem; font-size:0.7rem; color:#607070; text-align:center;">
          <div style="color:#0099ff; font-weight:700; font-size:0.65rem; letter-spacing:0.1em;">READ-ONLY MODE</div>
          <div style="margin-top:0.1rem;">Auditor View</div>
        </div>
      </div>

      {/* Stats */}
      <Show when={!auditLog.loading}>
        <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
          {[
            { label: 'AUDIT ENTRIES', value: (auditLog()??[]).length, color: '#0099ff' },
            { label: 'SUCCESS', value: (auditLog()??[]).filter(e=>e.outcome==='success').length, color: '#00ff88' },
            { label: 'FAILURES', value: (auditLog()??[]).filter(e=>e.outcome==='failure').length, color: '#ff3355' },
            { label: 'BLOCKED', value: (auditLog()??[]).filter(e=>e.outcome==='blocked').length, color: '#ffaa00' },
          ].map(s => (
            <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:0.9rem; border-radius:4px;">
              <div style={`font-size:1.4rem; font-weight:700; color:${s.color};`}>{s.value}</div>
              <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
            </div>
          ))}
        </div>
      </Show>

      {/* Tabs */}
      <div style="display:flex; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
        {(['audit','packages','export'] as Tab[]).map(t => (
          <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
        ))}
      </div>

      {/* ── Audit Log tab ── */}
      <Show when={tab() === 'audit'}>
        {/* Filters */}
        <div style="display:flex; gap:0.75rem; margin-bottom:1rem; flex-wrap:wrap;">
          <select value={framework()} onChange={e => setFramework(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.75rem;">
            <For each={FRAMEWORKS}>{f => <option value={f}>{f}</option>}</For>
          </select>
          <input type="date" value={dateFrom()} onChange={e => setDateFrom(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.6rem; border-radius:3px; font-size:0.75rem;" />
          <input type="date" value={dateTo()} onChange={e => setDateTo(e.currentTarget.value)}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.6rem; border-radius:3px; font-size:0.75rem;" />
          <button onClick={() => refetchAudit()}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#607070; padding:0.4rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.75rem;">
            ↻ FILTER
          </button>
          <span style="color:#607070; font-size:0.72rem; align-self:center; margin-left:auto;">
            {(auditLog()??[]).length} entries
          </span>
        </div>

        <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
          <table style="width:100%; border-collapse:collapse; font-size:0.72rem;">
            <thead>
              <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
                {['TIMESTAMP','ACTOR','ACTION','RESOURCE','OUTCOME','HASH'].map(h => (
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.08em; font-weight:400;">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              <Show when={!auditLog.loading} fallback={
                <tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">Loading audit log…</td></tr>
              }>
                <For each={auditLog()} fallback={
                  <tr><td colspan="6" style="padding:2rem; text-align:center; color:#607070;">No audit entries for selected filters.</td></tr>
                }>
                  {(entry) => {
                    const os = OUTCOME_STYLE[entry.outcome] ?? { color: '#607070', bg: '#0d1a1f' };
                    return (
                      <tr style="border-bottom:1px solid #0a1318;">
                        <td style="padding:0.5rem 0.9rem; color:#607070;">{new Date(entry.timestamp).toLocaleString()}</td>
                        <td style="padding:0.5rem 0.9rem; color:#c8d8d8;">{entry.actor}</td>
                        <td style="padding:0.5rem 0.9rem; color:#c8d8d8;">{entry.action}</td>
                        <td style="padding:0.5rem 0.9rem; color:#607070; max-width:160px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={entry.resource}>{entry.resource}</td>
                        <td style="padding:0.5rem 0.9rem;">
                          <span style={`font-size:0.65rem; font-weight:700; background:${os.bg}; color:${os.color}; border:1px solid ${os.color}33; padding:0.15rem 0.4rem; border-radius:2px; letter-spacing:0.08em;`}>
                            {entry.outcome.toUpperCase()}
                          </span>
                        </td>
                        <td style="padding:0.5rem 0.9rem; color:#1e3040; font-size:0.65rem;" title={entry.entry_hash}>#{entry.entry_hash.slice(0,8)}…</td>
                      </tr>
                    );
                  }}
                </For>
              </Show>
            </tbody>
          </table>
        </div>
      </Show>

      {/* ── Packages tab ── */}
      <Show when={tab() === 'packages'}>
        <Show when={!packages.loading} fallback={<div style="color:#607070; padding:1rem;">Loading packages…</div>}>
          <div style="display:grid; gap:1rem;">
            <For each={packages()} fallback={
              <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:2rem; text-align:center; color:#607070;">
                No compliance packages generated yet. Use the Export tab to create one.
              </div>
            }>
              {(pkg) => (
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem; display:flex; justify-content:space-between; align-items:center;">
                  <div>
                    <div style="display:flex; align-items:center; gap:0.75rem;">
                      <span style="background:#0a1318; border:1px solid #0099ff; color:#0099ff; padding:0.2rem 0.5rem; border-radius:3px; font-size:0.68rem; font-weight:700; letter-spacing:0.1em;">{pkg.framework}</span>
                      <span style="color:#c8d8d8; font-size:0.8rem;">{pkg.id}</span>
                    </div>
                    <div style="font-size:0.7rem; color:#607070; margin-top:0.4rem;">
                      {pkg.records.toLocaleString()} records · Generated {new Date(pkg.generated_at).toLocaleString()}
                    </div>
                    <div style="font-size:0.65rem; color:#1e3040; margin-top:0.2rem; font-family:monospace;" title="Merkle root / RFC3161 integrity proof">
                      Proof: {pkg.integrity_proof.slice(0,24)}…
                    </div>
                  </div>
                  <a href={pkg.download_url} download=""
                    style="background:#0099ff; color:#080f12; border:none; padding:0.45rem 1rem; border-radius:3px; cursor:pointer; font-weight:700; font-size:0.75rem; letter-spacing:0.08em; text-decoration:none;">
                    ↓ DOWNLOAD
                  </a>
                </div>
              )}
            </For>
          </div>
        </Show>
      </Show>

      {/* ── Export tab ── */}
      <Show when={tab() === 'export'}>
        <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #0099ff; border-radius:6px; padding:1.5rem; max-width:520px;">
          <div style="font-size:0.85rem; color:#c8d8d8; font-weight:600; margin-bottom:1rem;">Generate Compliance Package</div>

          <div style="display:flex; flex-direction:column; gap:1rem;">
            <div>
              <div style="font-size:0.65rem; color:#607070; letter-spacing:0.1em; margin-bottom:0.4rem;">FRAMEWORK</div>
              <div style="display:flex; flex-wrap:wrap; gap:0.4rem;">
                <For each={FRAMEWORKS.filter(f=>f!=='ALL')}>
                  {(f) => (
                    <button onClick={() => setSelectedFw(f)}
                      style={`background:${selectedFw()===f ? '#001833' : '#0a1318'}; border:1px solid ${selectedFw()===f ? '#0099ff' : '#1e3040'}; color:${selectedFw()===f ? '#0099ff' : '#607070'}; padding:0.35rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.72rem; letter-spacing:0.08em;`}>
                      {f}
                    </button>
                  )}
                </For>
              </div>
            </div>

            <div style="display:grid; grid-template-columns:1fr 1fr; gap:0.75rem;">
              <div>
                <div style="font-size:0.65rem; color:#607070; letter-spacing:0.1em; margin-bottom:0.35rem;">FROM DATE</div>
                <input type="date" value={dateFrom()} onChange={e => setDateFrom(e.currentTarget.value)}
                  style="background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.6rem; border-radius:3px; font-size:0.75rem; width:100%;" />
              </div>
              <div>
                <div style="font-size:0.65rem; color:#607070; letter-spacing:0.1em; margin-bottom:0.35rem;">TO DATE</div>
                <input type="date" value={dateTo()} onChange={e => setDateTo(e.currentTarget.value)}
                  style="background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.6rem; border-radius:3px; font-size:0.75rem; width:100%;" />
              </div>
            </div>

            <div style="background:#001833; border:1px solid #1e3040; border-radius:4px; padding:0.75rem; font-size:0.7rem; color:#607070; line-height:1.6;">
              Package will include:<br/>
              • Cryptographically chained audit log (JSON Lines)<br/>
              • Merkle tree integrity proof<br/>
              • Evidence locker manifest<br/>
              • Compliance posture snapshot for <strong style="color:#0099ff;">{selectedFw()}</strong>
            </div>

            <button onClick={generatePackage} disabled={generating()}
              style={`background:#0099ff; color:#080f12; border:none; padding:0.6rem 1.25rem; border-radius:3px; cursor:${generating() ? 'wait' : 'pointer'}; font-weight:700; font-size:0.78rem; letter-spacing:0.1em; opacity:${generating() ? '0.6' : '1'};`}>
              {generating() ? 'GENERATING…' : `↓ GENERATE ${selectedFw()} PACKAGE`}
            </button>
          </div>
        </div>
      </Show>
    </div>
  );
}
