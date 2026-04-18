/**
 * MitreHeatmap.tsx — MITRE ATT&CK Heatmap (Hybrid — Phase 4)
 *
 * Visual heatmap of triggered ATT&CK techniques grouped by tactic.
 * Fetches alert counts from /api/v1/mitre/heatmap and colors each
 * technique cell by hit frequency.
 */

import { createResource, For, Show, createSignal } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface TacticRow {
  id: string;
  name: string;
  techniques: TechniqueCell[];
}
interface TechniqueCell {
  id: string;
  name: string;
  hits: number;
}
interface HeatmapData {
  tactics: TacticRow[];
  total_hits: number;
  last_updated: string;
}

// ── Constants ─────────────────────────────────────────────────────────────────
const TACTIC_ORDER = [
  'TA0001','TA0002','TA0003','TA0004','TA0005',
  'TA0006','TA0007','TA0008','TA0009','TA0011','TA0010','TA0040',
];

// ── Helpers ───────────────────────────────────────────────────────────────────
function heatColor(hits: number, max: number): { bg: string; border: string; text: string } {
  if (hits === 0) return { bg: '#0d1a1f', border: '#1e3040', text: '#607070' };
  const ratio = Math.min(hits / Math.max(max, 1), 1);
  if (ratio > 0.75) return { bg: '#2a0d15', border: '#ff3355', text: '#ff3355' };
  if (ratio > 0.5)  return { bg: '#2a1500', border: '#ff6600', text: '#ff6600' };
  if (ratio > 0.25) return { bg: '#2a2000', border: '#ffaa00', text: '#ffaa00' };
  return { bg: '#001a0a', border: '#00ff88', text: '#00ff88' };
}

// ── Component ─────────────────────────────────────────────────────────────────
export default function MitreHeatmap() {
  const [selected, setSelected] = createSignal<TechniqueCell | null>(null);
  const [showZero, setShowZero] = createSignal(false);

  const [data, { refetch }] = createResource<HeatmapData>(async () => {
    const r = await request<HeatmapData>('/mitre/heatmap');
    // Sort tactics into ATT&CK canonical order
    r.tactics = [...r.tactics].sort(
      (a, b) => TACTIC_ORDER.indexOf(a.id) - TACTIC_ORDER.indexOf(b.id)
    );
    return r;
  });

  const maxHits = () => {
    let m = 0;
    (data()?.tactics ?? []).forEach(t =>
      t.techniques.forEach(tc => { if (tc.hits > m) m = tc.hits; })
    );
    return m;
  };

  const visibleTechniques = (tactic: TacticRow) =>
    showZero() ? tactic.techniques : tactic.techniques.filter(t => t.hits > 0 || true);

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="display:flex; align-items:flex-start; justify-content:space-between; margin-bottom:1.5rem;">
        <div>
          <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff3355;">⬡ MITRE ATT&amp;CK HEATMAP</h1>
          <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
            Technique coverage · Alert frequency · Tactic grouping
          </p>
        </div>
        <div style="display:flex; gap:0.75rem; align-items:center;">
          <button onClick={() => setShowZero(v => !v)}
            style={`background:${showZero() ? '#1e3040' : '#0d1a1f'}; border:1px solid #1e3040; color:#607070; padding:0.35rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.72rem; letter-spacing:0.08em;`}>
            {showZero() ? '○ ALL' : '● HIT ONLY'}
          </button>
          <button onClick={() => refetch()}
            style="background:#0d1a1f; border:1px solid #1e3040; color:#607070; padding:0.35rem 0.75rem; border-radius:3px; cursor:pointer; font-size:0.72rem;">
            ↻ REFRESH
          </button>
        </div>
      </div>

      {/* Legend */}
      <div style="display:flex; gap:1rem; margin-bottom:1.5rem; align-items:center;">
        <span style="font-size:0.65rem; color:#607070; letter-spacing:0.1em;">FREQUENCY:</span>
        {[
          { label: 'None', bg: '#0d1a1f', border: '#1e3040', text: '#607070' },
          { label: 'Low',  bg: '#001a0a', border: '#00ff88', text: '#00ff88' },
          { label: 'Med',  bg: '#2a2000', border: '#ffaa00', text: '#ffaa00' },
          { label: 'High', bg: '#2a1500', border: '#ff6600', text: '#ff6600' },
          { label: 'Crit', bg: '#2a0d15', border: '#ff3355', text: '#ff3355' },
        ].map(l => (
          <div style="display:flex; align-items:center; gap:0.35rem;">
            <div style={`width:12px; height:12px; border-radius:2px; background:${l.bg}; border:1px solid ${l.border};`}></div>
            <span style={`font-size:0.65rem; color:${l.text};`}>{l.label}</span>
          </div>
        ))}
        <Show when={data()}>
          <span style="margin-left:auto; font-size:0.7rem; color:#607070;">
            {data()!.total_hits.toLocaleString()} total hits ·
            Last updated: {data()!.last_updated ? new Date(data()!.last_updated).toLocaleTimeString() : '—'}
          </span>
        </Show>
      </div>

      {/* Loading */}
      <Show when={data.loading}>
        <div style="color:#607070; font-size:0.78rem; text-align:center; padding:3rem;">
          Loading ATT&amp;CK matrix…
        </div>
      </Show>

      {/* Heatmap Grid */}
      <Show when={!data.loading && data()}>
        <div style="display:grid; grid-template-columns:repeat(6, 1fr); gap:0.75rem;">
          <For each={data()!.tactics}>
            {(tactic) => (
              <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
                {/* Tactic header */}
                <div style="padding:0.6rem 0.75rem; background:#0a1318; border-bottom:1px solid #1e3040;">
                  <div style="font-size:0.62rem; color:#607070; letter-spacing:0.1em;">{tactic.id}</div>
                  <div style="font-size:0.72rem; color:#c8d8d8; font-weight:600; margin-top:0.1rem; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">{tactic.name}</div>
                  <div style="font-size:0.6rem; color:#607070; margin-top:0.15rem;">
                    {tactic.techniques.filter(t => t.hits > 0).length}/{tactic.techniques.length} triggered
                  </div>
                </div>

                {/* Technique cells */}
                <div style="padding:0.4rem; display:flex; flex-direction:column; gap:0.3rem;">
                  <For each={showZero() ? tactic.techniques : visibleTechniques(tactic)}>
                    {(tech) => {
                      const c = heatColor(tech.hits, maxHits());
                      return (
                        <div
                          onClick={() => setSelected(selected()?.id === tech.id ? null : tech)}
                          title={`${tech.id} — ${tech.name}: ${tech.hits} hits`}
                          style={`background:${c.bg}; border:1px solid ${selected()?.id === tech.id ? '#c8d8d8' : c.border}; border-radius:3px; padding:0.4rem 0.5rem; cursor:pointer; transition:all 0.1s; display:flex; justify-content:space-between; align-items:center;`}
                          onMouseEnter={e => (e.currentTarget as HTMLElement).style.opacity = '0.8'}
                          onMouseLeave={e => (e.currentTarget as HTMLElement).style.opacity = '1'}
                        >
                          <div>
                            <div style={`font-size:0.62rem; font-weight:700; color:${c.text}; letter-spacing:0.05em;`}>{tech.id}</div>
                            <div style="font-size:0.62rem; color:#607070; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; max-width:90px;" title={tech.name}>{tech.name}</div>
                          </div>
                          <Show when={tech.hits > 0}>
                            <span style={`font-size:0.65rem; font-weight:700; color:${c.text};`}>{tech.hits}</span>
                          </Show>
                        </div>
                      );
                    }}
                  </For>
                </div>
              </div>
            )}
          </For>
        </div>
      </Show>

      {/* Detail panel */}
      <Show when={selected()}>
        {(s) => {
          const c = heatColor(s().hits, maxHits());
          return (
            <div style={`position:fixed; right:1.5rem; bottom:1.5rem; background:#0d1a1f; border:1px solid ${c.border}; border-radius:8px; padding:1.25rem; width:280px; box-shadow:0 8px 32px rgba(0,0,0,0.6); z-index:100;`}>
              <div style="display:flex; justify-content:space-between; align-items:flex-start;">
                <div>
                  <div style={`font-size:0.72rem; font-weight:700; color:${c.text}; letter-spacing:0.1em;`}>{s().id}</div>
                  <div style="font-size:0.88rem; color:#c8d8d8; margin-top:0.25rem;">{s().name}</div>
                </div>
                <button onClick={() => setSelected(null)}
                  style="background:none; border:none; color:#607070; cursor:pointer; font-size:1rem; padding:0;">✕</button>
              </div>
              <div style="margin-top:1rem; display:flex; justify-content:space-between; font-size:0.74rem;">
                <span style="color:#607070;">Alert Hits</span>
                <span style={`color:${c.text}; font-weight:700; font-size:1rem;`}>{s().hits.toLocaleString()}</span>
              </div>
              <div style="height:6px; background:#1e3040; border-radius:3px; margin-top:0.5rem;">
                <div style={`height:100%; border-radius:3px; background:${c.border}; width:${Math.round(s().hits / Math.max(maxHits(), 1) * 100)}%;`}></div>
              </div>
              <div style="margin-top:0.75rem; font-size:0.68rem; color:#607070;">
                ATT&amp;CK Navigator · <a href={`https://attack.mitre.org/techniques/${s().id}/`} target="_blank" style={`color:${c.text}; text-decoration:none;`}>View on MITRE ↗</a>
              </div>
            </div>
          );
        }}
      </Show>
    </div>
  );
}
