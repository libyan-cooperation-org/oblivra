/**
 * LookupManager.tsx — Lookup Table Management (Hybrid — Phase 1.3)
 *
 * Manage CSV/JSON lookup tables used by the OQL enrichment pipeline.
 * Supports Exact, CIDR, Wildcard, and Regex match strategies.
 * Connects to /api/v1/lookups/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface LookupTable {
  name: string;
  match_type: 'exact' | 'cidr' | 'wildcard' | 'regex';
  fields: string[];
  rows?: Record<string, string>[];
}

interface LookupResponse { tables: LookupTable[]; }

type MatchType = 'exact' | 'cidr' | 'wildcard' | 'regex';

// ── Helpers ───────────────────────────────────────────────────────────────────
const MT_COLOR: Record<MatchType, string> = {
  exact:    '#00ffe7',
  cidr:     '#00ff88',
  wildcard: '#ffaa00',
  regex:    '#ff6600',
};

// ── Component ─────────────────────────────────────────────────────────────────
export default function LookupManager() {
  const [tables, { refetch }] = createResource<LookupTable[]>(async () => {
    const res = await request<LookupResponse>('/lookups');
    return res.tables ?? [];
  });

  const [selected, setSelected] = createSignal<LookupTable | null>(null);
  const [queryTable, setQueryTable] = createSignal('');
  const [queryKey, setQueryKey]     = createSignal('');
  const [queryResult, setQueryResult] = createSignal<{match: boolean; data: Record<string,string>|null}|null>(null);

  // Upload state
  const [uploadName,  setUploadName]  = createSignal('');
  const [uploadMT,    setUploadMT]    = createSignal<MatchType>('exact');
  const [uploadFmt,   setUploadFmt]   = createSignal<'csv'|'json'>('csv');
  const [uploadFile,  setUploadFile]  = createSignal<File|null>(null);
  const [uploadMsg,   setUploadMsg]   = createSignal('');

  async function loadTable(name: string) {
    try {
      const t = await request<LookupTable>(`/lookups/${name}`);
      setSelected(t);
    } catch { /* ignore */ }
  }

  async function deleteTable(name: string) {
    try {
      await request(`/lookups/${name}`, { method: 'DELETE' });
      refetch();
      if (selected()?.name === name) setSelected(null);
    } catch(e: any) {
      alert(e.message);
    }
  }

  async function uploadTable() {
    const file = uploadFile();
    if (!file || !uploadName()) return;
    const form = new FormData();
    form.append('name', uploadName());
    form.append('match_type', uploadMT());
    form.append('format', uploadFmt());
    form.append('file', file);
    try {
      await fetch('/api/v1/lookups/upload', {
        method: 'POST',
        headers: { 'X-API-Key': localStorage.getItem('oblivra_token') ?? '' },
        body: form,
      });
      setUploadMsg(`✓ Table "${uploadName()}" uploaded successfully.`);
      refetch();
    } catch(e: any) {
      setUploadMsg(`✗ Upload failed: ${e.message}`);
    }
  }

  async function runQuery() {
    const t = queryTable(), k = queryKey();
    if (!t || !k) return;
    try {
      const res = await request<{match:boolean; data: Record<string,string>|null}>(
        `/lookups/query?table=${encodeURIComponent(t)}&key=${encodeURIComponent(k)}`
      );
      setQueryResult(res);
    } catch { setQueryResult({ match: false, data: null }); }
  }

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ LOOKUP TABLES</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
          Enrichment lookup tables — Exact · CIDR · Wildcard · Regex
        </p>
      </div>

      <div style="display:grid; grid-template-columns:300px 1fr; gap:1.5rem; align-items:start;">
        {/* Left — Table list */}
        <div>
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden; margin-bottom:1rem;">
            <div style="padding:0.65rem 1rem; background:#0a1318; border-bottom:1px solid #1e3040; display:flex; justify-content:space-between; align-items:center;">
              <span style="font-size:0.72rem; color:#607070; letter-spacing:0.12em;">TABLES ({tables()?.length ?? 0})</span>
              <button onClick={() => refetch()} style="background:none; border:none; color:#607070; cursor:pointer; font-size:0.75rem;">↻</button>
            </div>
            <Show when={!tables.loading} fallback={<div style="padding:1rem; color:#607070; font-size:0.76rem;">Loading…</div>}>
              <For each={tables()} fallback={<div style="padding:1rem; color:#607070; font-size:0.76rem;">No tables found.</div>}>
                {(t) => (
                  <div
                    onClick={() => loadTable(t.name)}
                    style={`padding:0.7rem 1rem; cursor:pointer; border-bottom:1px solid #0a1318; border-left:3px solid ${selected()?.name===t.name ? '#00ffe7' : 'transparent'}; background:${selected()?.name===t.name ? '#111f28' : 'transparent'};`}
                  >
                    <div style="display:flex; justify-content:space-between; align-items:center;">
                      <span style="font-size:0.8rem; color:#c8d8d8;">{t.name}</span>
                      <button
                        onClick={e => { e.stopPropagation(); deleteTable(t.name); }}
                        style="background:none; border:none; color:#607070; cursor:pointer; font-size:0.72rem;"
                        title="Delete table"
                      >✕</button>
                    </div>
                    <span style={`font-size:0.65rem; color:${MT_COLOR[t.match_type]}; letter-spacing:0.1em;`}>{t.match_type.toUpperCase()}</span>
                  </div>
                )}
              </For>
            </Show>
          </div>

          {/* Quick Query */}
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem;">
            <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">QUICK QUERY</div>
            <select
              value={queryTable()} onChange={e => setQueryTable(e.currentTarget.value)}
              style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem; margin-bottom:0.5rem;"
            >
              <option value="">Select table…</option>
              <For each={tables()}>
                {(t) => <option value={t.name}>{t.name}</option>}
              </For>
            </select>
            <input
              type="text" placeholder="Key to look up…"
              value={queryKey()} onInput={e => setQueryKey(e.currentTarget.value)}
              onKeyDown={e => e.key === 'Enter' && runQuery()}
              style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem 0.5rem; border-radius:3px; font-size:0.76rem; margin-bottom:0.5rem; box-sizing:border-box;"
            />
            <button
              onClick={runQuery}
              style="width:100%; background:#00ffe7; color:#080f12; border:none; padding:0.45rem; border-radius:3px; cursor:pointer; font-size:0.75rem; font-weight:700; letter-spacing:0.1em;"
            >LOOKUP</button>
            <Show when={queryResult() !== null}>
              <div style={`margin-top:0.75rem; padding:0.65rem; border-radius:3px; background:${queryResult()?.match ? '#002a1a' : '#2a0d15'}; border:1px solid ${queryResult()?.match ? '#00ff88' : '#ff3355'};`}>
                <div style={`font-size:0.68rem; font-weight:700; color:${queryResult()?.match ? '#00ff88' : '#ff3355'}; margin-bottom:${queryResult()?.match ? '0.5rem' : '0'};`}>
                  {queryResult()?.match ? '● MATCH FOUND' : '○ NO MATCH'}
                </div>
                <Show when={queryResult()?.data}>
                  <For each={Object.entries(queryResult()!.data!)}>
                    {([k, v]) => (
                      <div style="font-size:0.7rem; color:#607070;">
                        <span style="color:#c8d8d8;">{k}</span>: {v}
                      </div>
                    )}
                  </For>
                </Show>
              </div>
            </Show>
          </div>
        </div>

        {/* Right — Table detail or Upload form */}
        <div style="display:flex; flex-direction:column; gap:1.5rem;">
          {/* Upload form */}
          <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1.25rem;">
            <div style="font-size:0.72rem; color:#607070; letter-spacing:0.12em; margin-bottom:1rem;">UPLOAD NEW TABLE</div>
            <div style="display:grid; grid-template-columns:1fr 1fr 1fr; gap:0.75rem; margin-bottom:0.75rem;">
              <div>
                <div style="font-size:0.65rem; color:#607070; margin-bottom:0.25rem;">TABLE NAME</div>
                <input type="text" value={uploadName()} onInput={e => setUploadName(e.currentTarget.value)}
                  placeholder="my_blocklist"
                  style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem; box-sizing:border-box;" />
              </div>
              <div>
                <div style="font-size:0.65rem; color:#607070; margin-bottom:0.25rem;">MATCH TYPE</div>
                <select value={uploadMT()} onChange={e => setUploadMT(e.currentTarget.value as MatchType)}
                  style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem;">
                  <option value="exact">Exact</option>
                  <option value="cidr">CIDR</option>
                  <option value="wildcard">Wildcard</option>
                  <option value="regex">Regex</option>
                </select>
              </div>
              <div>
                <div style="font-size:0.65rem; color:#607070; margin-bottom:0.25rem;">FORMAT</div>
                <select value={uploadFmt()} onChange={e => setUploadFmt(e.currentTarget.value as 'csv'|'json')}
                  style="width:100%; background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:0.4rem; border-radius:3px; font-size:0.76rem;">
                  <option value="csv">CSV</option>
                  <option value="json">JSON</option>
                </select>
              </div>
            </div>
            <div style="display:flex; gap:0.75rem; align-items:center;">
              <input type="file" accept=".csv,.json"
                onChange={e => setUploadFile(e.currentTarget.files?.[0] ?? null)}
                style="flex:1; background:#0a1318; border:1px solid #1e3040; color:#607070; padding:0.4rem; border-radius:3px; font-size:0.76rem;" />
              <button onClick={uploadTable}
                style="background:#00ffe7; color:#080f12; border:none; padding:0.45rem 1.25rem; border-radius:3px; cursor:pointer; font-size:0.76rem; font-weight:700; letter-spacing:0.1em; white-space:nowrap;">
                UPLOAD
              </button>
            </div>
            <Show when={uploadMsg()}>
              <div style={`margin-top:0.6rem; font-size:0.73rem; color:${uploadMsg().startsWith('✓') ? '#00ff88' : '#ff3355'};`}>{uploadMsg()}</div>
            </Show>
          </div>

          {/* Table viewer */}
          <Show when={selected()}>
            {(t) => (
              <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
                <div style="padding:0.75rem 1rem; background:#0a1318; border-bottom:1px solid #1e3040; display:flex; justify-content:space-between;">
                  <span style="color:#00ffe7; font-size:0.85rem;">{t().name}</span>
                  <span style={`font-size:0.7rem; color:${MT_COLOR[t().match_type]};`}>{t().match_type.toUpperCase()} · {t().rows?.length ?? 0} rows</span>
                </div>
                <div style="overflow-x:auto; max-height:400px; overflow-y:auto;">
                  <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                    <thead>
                      <tr style="border-bottom:1px solid #1e3040;">
                        <For each={t().fields}>
                          {(f) => <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; letter-spacing:0.1em; font-weight:400;">{f.toUpperCase()}</th>}
                        </For>
                      </tr>
                    </thead>
                    <tbody>
                      <For each={t().rows ?? []}>
                        {(row) => (
                          <tr style="border-bottom:1px solid #0a1318;">
                            <For each={t().fields}>
                              {(f) => <td style="padding:0.55rem 0.9rem; color:#c8d8d8;">{row[f] ?? '—'}</td>}
                            </For>
                          </tr>
                        )}
                      </For>
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </Show>
        </div>
      </div>
    </div>
  );
}
