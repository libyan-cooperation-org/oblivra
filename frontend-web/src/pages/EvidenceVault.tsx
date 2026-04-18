/**
 * EvidenceVault.tsx — Legal-Grade Digital Evidence Vault (Phase 6.5)
 *
 * Chain-of-custody browser, evidence submission, integrity verification,
 * and forensic export. Connects to /api/v1/forensics/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface ChainEntry {
  action: string;
  actor: string;
  timestamp: string;
  notes?: string;
  previous_hash: string;
  entry_hash: string;
}
interface EvidenceItem {
  id: string;
  incident_id: string;
  type: string;
  name: string;
  description?: string;
  sha256: string;
  size: number;
  collector: string;
  collected_at: string;
  sealed: boolean;
  sealed_at?: string;
  chain_of_custody: ChainEntry[];
  tags?: string[];
}

// ── Helpers ───────────────────────────────────────────────────────────────────
const TYPE_ICONS: Record<string, string> = {
  file: '📄', log: '📋', screenshot: '🖼', pcap: '🌐',
  memory_dump: '💾', artifact: '🔍',
};
const ACTION_COLORS: Record<string, string> = {
  collected: '#00ff88', analyzed: '#ffaa00', transferred: '#0099ff',
  sealed: '#607070', exported: '#c8d8d8', verified: '#5cc05c',
};
const fmt = (n: number) => n < 1024 ? `${n} B`
  : n < 1048576 ? `${(n/1024).toFixed(1)} KB`
  : `${(n/1048576).toFixed(2)} MB`;

type Tab = 'all' | 'unsealed' | 'sealed';

// ── Component ─────────────────────────────────────────────────────────────────
export default function EvidenceVault() {
  const [tab, setTab] = createSignal<Tab>('all');
  const [selected, setSelected] = createSignal<EvidenceItem | null>(null);
  const [verifying, setVerifying] = createSignal<string | null>(null);
  const [verifyResult, setVerifyResult] = createSignal<Record<string, boolean>>({});
  const [search, setSearch] = createSignal('');

  const [items, { refetch }] = createResource<EvidenceItem[]>(async () => {
    const r = await request<{ items: EvidenceItem[] }>('/forensics/evidence');
    return r.items ?? [];
  });

  const filtered = () => {
    const q = search().toLowerCase();
    return (items() ?? []).filter(i => {
      if (tab() === 'sealed' && !i.sealed) return false;
      if (tab() === 'unsealed' && i.sealed) return false;
      return !q || i.name.toLowerCase().includes(q) || i.incident_id.toLowerCase().includes(q) || i.type.includes(q);
    });
  };

  async function verify(id: string) {
    setVerifying(id);
    try {
      const r = await request<{ valid: boolean }>(`/forensics/evidence/${id}/verify`);
      setVerifyResult(prev => ({ ...prev, [id]: r.valid }));
    } catch { setVerifyResult(prev => ({ ...prev, [id]: false })); }
    setVerifying(null);
  }

  async function seal(id: string) {
    await request(`/forensics/evidence/${id}/seal`, { method: 'POST' });
    refetch();
  }

  async function exportVault() {
    const url = `/api/v1/forensics/export`;
    const token = localStorage.getItem('oblivra_token');
    const a = document.createElement('a');
    a.href = url + (token ? `?token=${token}` : '');
    a.download = `oblivra-evidence-${Date.now()}.json`;
    a.click();
  }

  const TAB = (t: Tab) =>
    `padding:0.5rem 1.2rem; cursor:pointer; font-size:0.78rem; letter-spacing:0.12em; border:none; border-bottom:2px solid ${tab()===t ? '#ff3355' : 'transparent'}; background:none; color:${tab()===t ? '#ff3355' : '#607070'};`;

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="display:flex; justify-content:space-between; align-items:flex-start; margin-bottom:1.5rem;">
        <div>
          <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff3355;">⬡ EVIDENCE VAULT</h1>
          <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
            Chain-of-custody · Integrity verification · Legal-grade forensics
          </p>
        </div>
        <div style="display:flex; gap:0.75rem;">
          <button onClick={exportVault}
            style="background:#ff3355; color:#080f12; border:none; padding:0.45rem 1rem; border-radius:3px; cursor:pointer; font-weight:700; font-size:0.75rem; letter-spacing:0.08em;">
            ↓ EXPORT VAULT
          </button>
          <button onClick={() => refetch()}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#607070; padding:0.45rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.75rem;">
            ↻
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <Show when={!items.loading}>
        <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:1rem; margin-bottom:1.5rem;">
          {[
            { label: 'TOTAL', value: (items()??[]).length, color: '#c8d8d8' },
            { label: 'SEALED', value: (items()??[]).filter(i=>i.sealed).length, color: '#607070' },
            { label: 'ACTIVE', value: (items()??[]).filter(i=>!i.sealed).length, color: '#00ff88' },
            { label: 'INCIDENTS', value: new Set((items()??[]).map(i=>i.incident_id)).size, color: '#ffaa00' },
          ].map(s => (
            <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #1e3040; padding:0.9rem; border-radius:4px;">
              <div style={`font-size:1.4rem; font-weight:700; color:${s.color};`}>{s.value}</div>
              <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">{s.label}</div>
            </div>
          ))}
        </div>
      </Show>

      {/* Filter + tabs */}
      <div style="display:flex; align-items:center; justify-content:space-between; border-bottom:1px solid #1e3040; margin-bottom:1.25rem;">
        <div>
          {(['all','unsealed','sealed'] as Tab[]).map(t => (
            <button style={TAB(t)} onClick={() => setTab(t)}>{t.toUpperCase()}</button>
          ))}
        </div>
        <input type="text" value={search()} onInput={e => setSearch(e.currentTarget.value)}
          placeholder="Filter by name, incident, type…"
          style="background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.35rem 0.75rem; border-radius:3px; font-size:0.75rem; width:260px; margin-bottom:0.5rem;" />
      </div>

      {/* Evidence table */}
      <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; margin-bottom:1.5rem;">
        <table style="width:100%; border-collapse:collapse; font-size:0.74rem;">
          <thead>
            <tr style="border-bottom:1px solid #1e3040; background:#0a1318;">
              {['TYPE','NAME','INCIDENT','SIZE','COLLECTOR','STATUS','ACTIONS'].map(h => (
                <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            <Show when={!items.loading} fallback={
              <tr><td colspan="7" style="padding:1.5rem; text-align:center; color:#607070;">Loading evidence…</td></tr>
            }>
              <For each={filtered()} fallback={
                <tr><td colspan="7" style="padding:1.5rem; text-align:center; color:#607070;">No evidence items found.</td></tr>
              }>
                {(item) => (
                  <tr style="border-bottom:1px solid #0a1318; cursor:pointer;"
                    onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = '#111f28'}
                    onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = ''}
                    onClick={() => setSelected(selected()?.id === item.id ? null : item)}>
                    <td style="padding:0.55rem 0.9rem; color:#607070;">{TYPE_ICONS[item.type] ?? '?'} {item.type}</td>
                    <td style="padding:0.55rem 0.9rem; color:#c8d8d8; max-width:180px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={item.name}>{item.name}</td>
                    <td style="padding:0.55rem 0.9rem; color:#607070; font-size:0.68rem;">{item.incident_id.slice(0,12)}…</td>
                    <td style="padding:0.55rem 0.9rem; color:#607070;">{fmt(item.size)}</td>
                    <td style="padding:0.55rem 0.9rem; color:#607070;">{item.collector}</td>
                    <td style="padding:0.55rem 0.9rem;">
                      <span style={`font-size:0.65rem; font-weight:700; letter-spacing:0.1em; color:${item.sealed ? '#607070' : '#00ff88'};`}>
                        {item.sealed ? '🔒 SEALED' : '● ACTIVE'}
                      </span>
                    </td>
                    <td style="padding:0.55rem 0.9rem;">
                      <div style="display:flex; gap:0.4rem;">
                        <button onClick={e => { e.stopPropagation(); verify(item.id); }}
                          disabled={verifying() === item.id}
                          style={`background:${verifyResult()[item.id] === true ? '#001a0a' : verifyResult()[item.id] === false ? '#2a0d15' : '#0a1318'}; border:1px solid ${verifyResult()[item.id] === true ? '#00ff88' : verifyResult()[item.id] === false ? '#ff3355' : '#1e3040'}; color:${verifyResult()[item.id] === true ? '#00ff88' : verifyResult()[item.id] === false ? '#ff3355' : '#607070'}; padding:0.25rem 0.5rem; border-radius:2px; cursor:pointer; font-size:0.65rem;`}>
                          {verifying() === item.id ? '…' : verifyResult()[item.id] === true ? '✓ OK' : verifyResult()[item.id] === false ? '✗ FAIL' : 'VERIFY'}
                        </button>
                        <Show when={!item.sealed}>
                          <button onClick={e => { e.stopPropagation(); seal(item.id); }}
                            style="background:#0a1318; border:1px solid #1e3040; color:#607070; padding:0.25rem 0.5rem; border-radius:2px; cursor:pointer; font-size:0.65rem;">
                            SEAL
                          </button>
                        </Show>
                      </div>
                    </td>
                  </tr>
                )}
              </For>
            </Show>
          </tbody>
        </table>
      </div>

      {/* Detail panel — Chain of Custody */}
      <Show when={selected()}>
        {(item) => (
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ff3355; border-radius:6px; padding:1.25rem;">
            <div style="display:flex; justify-content:space-between; margin-bottom:1rem;">
              <div>
                <div style="font-size:0.88rem; color:#ff3355; letter-spacing:0.08em;">{item().name}</div>
                <div style="font-size:0.7rem; color:#607070; margin-top:0.25rem;">SHA-256: {item().sha256}</div>
              </div>
              <button onClick={() => setSelected(null)} style="background:none; border:none; color:#607070; cursor:pointer; font-size:1rem;">✕</button>
            </div>

            <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">CHAIN OF CUSTODY</div>
            <div style="display:flex; flex-direction:column; gap:0;">
              <For each={item().chain_of_custody}>
                {(entry, idx) => (
                  <div style={`display:flex; gap:1rem; padding:0.6rem 0; ${idx() < item().chain_of_custody.length-1 ? 'border-bottom:1px solid #0a1318' : ''}`}>
                    <div style="display:flex; flex-direction:column; align-items:center; gap:0;">
                      <div style={`width:8px; height:8px; border-radius:50%; background:${ACTION_COLORS[entry.action] ?? '#607070'}; flex-shrink:0;`}></div>
                      <Show when={idx() < item().chain_of_custody.length-1}>
                        <div style="width:1px; background:#1e3040; flex:1; min-height:12px;"></div>
                      </Show>
                    </div>
                    <div style="flex:1;">
                      <div style="display:flex; gap:0.75rem; align-items:baseline;">
                        <span style={`font-size:0.7rem; font-weight:700; color:${ACTION_COLORS[entry.action] ?? '#607070'}; letter-spacing:0.1em;`}>{entry.action.toUpperCase()}</span>
                        <span style="font-size:0.68rem; color:#607070;">{entry.actor}</span>
                        <span style="font-size:0.65rem; color:#3a5060; margin-left:auto;">{new Date(entry.timestamp).toLocaleString()}</span>
                      </div>
                      <Show when={entry.notes}>
                        <div style="font-size:0.68rem; color:#607070; margin-top:0.15rem;">{entry.notes}</div>
                      </Show>
                      <div style="font-size:0.6rem; color:#1e3040; margin-top:0.15rem; font-family:monospace;" title="HMAC entry hash">
                        #{entry.entry_hash.slice(0,16)}…
                      </div>
                    </div>
                  </div>
                )}
              </For>
            </div>
          </div>
        )}
      </Show>
    </div>
  );
}
