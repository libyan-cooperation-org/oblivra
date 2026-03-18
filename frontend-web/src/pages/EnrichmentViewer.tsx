/**
 * EnrichmentViewer.tsx — Enrichment Pipeline Inspector (Hybrid — Phase 3.2)
 *
 * Live enrichment results: GeoIP, DNS/ASN, asset mapping, and risk scoring for
 * IP addresses or host IDs. Connects to /api/v1/enrich/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

// ── Types ─────────────────────────────────────────────────────────────────────
interface GeoResult {
  ip: string;
  country_code: string;
  country_name: string;
  city: string;
  latitude: number;
  longitude: number;
  asn: string;
  org: string;
  isp: string;
}
interface DNSResult {
  hostname: string;
  ptr: string;
  asn: string;
  abuse_contact?: string;
  is_tor?: boolean;
  is_vpn?: boolean;
}
interface AssetResult {
  ip: string;
  host_id?: string;
  hostname?: string;
  os?: string;
  tags?: string[];
  last_seen?: string;
  risk_score?: number;
}
interface EnrichmentResult {
  query: string;
  geo?: GeoResult;
  dns?: DNSResult;
  asset?: AssetResult;
  ioc_match?: { matched: boolean; severity?: string; source?: string; description?: string };
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function riskColor(score?: number): string {
  if (!score) return '#607070';
  if (score >= 80) return '#ff3355';
  if (score >= 60) return '#ff6600';
  if (score >= 40) return '#ffaa00';
  return '#00ff88';
}

// ── Component ─────────────────────────────────────────────────────────────────
export default function EnrichmentViewer() {
  const [query, setQuery] = createSignal('');
  const [submitted, setSubmitted] = createSignal('');

  const [result] = createResource(
    () => submitted(),
    async (q) => {
      if (!q) return null;
      try {
        return await request<EnrichmentResult>(`/enrich?q=${encodeURIComponent(q)}`);
      } catch { return null; }
    }
  );

  const [recent, { refetch: refetchRecent }] = createResource<EnrichmentResult[]>(async () => {
    try { return await request<EnrichmentResult[]>('/enrich/recent'); }
    catch { return []; }
  });

  function run() {
    const q = query().trim();
    if (q) setSubmitted(q);
  }

  return (
    <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
      {/* Header */}
      <div style="margin-bottom:1.5rem;">
        <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ ENRICHMENT PIPELINE</h1>
        <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
          GeoIP · DNS/ASN · Asset mapping · IOC correlation — enter an IP or hostname
        </p>
      </div>

      {/* Query */}
      <div style="display:flex; gap:0.75rem; margin-bottom:1.5rem;">
        <input
          id="enrich-query"
          type="text" value={query()}
          onInput={e => setQuery(e.currentTarget.value)}
          onKeyDown={e => e.key === 'Enter' && run()}
          placeholder="1.2.3.4 / suspicious.com / hostname-01"
          style="flex:1; background:#0d1a1f; border:1px solid #1e3040; color:#c8d8d8; padding:0.6rem 1rem; border-radius:4px; font-size:0.82rem;"
        />
        <button onClick={run}
          style="background:#00ffe7; color:#080f12; border:none; padding:0.6rem 1.5rem; border-radius:4px; cursor:pointer; font-weight:700; font-size:0.82rem; letter-spacing:0.1em;">
          ENRICH
        </button>
      </div>

      {/* Result */}
      <Show when={submitted()}>
        <Show when={result.loading}>
          <div style="color:#607070; margin-bottom:1.5rem; font-size:0.78rem;">Enriching {submitted()}…</div>
        </Show>
        <Show when={!result.loading && result()}>
          {(r) => (
            <div style="display:grid; grid-template-columns:1fr 1fr; gap:1rem; margin-bottom:1.5rem;">
              {/* GeoIP Card */}
              <Show when={r().geo}>
                {(g) => (
                  <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #00ffe7; border-radius:6px; padding:1.25rem;">
                    <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">GEOIP</div>
                    {[
                      ['Country', `${g().country_name} (${g().country_code})`],
                      ['City', g().city || '—'],
                      ['ASN', g().asn || '—'],
                      ['ISP / Org', g().org || g().isp || '—'],
                      ['Coords', g().latitude ? `${g().latitude.toFixed(2)}, ${g().longitude.toFixed(2)}` : '—'],
                    ].map(([k, v]) => (
                      <div style="display:flex; justify-content:space-between; margin-bottom:0.4rem; font-size:0.76rem;">
                        <span style="color:#607070;">{k}</span>
                        <span style="color:#c8d8d8;">{v}</span>
                      </div>
                    ))}
                  </div>
                )}
              </Show>

              {/* DNS Card */}
              <Show when={r().dns}>
                {(d) => (
                  <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #00ff88; border-radius:6px; padding:1.25rem;">
                    <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">DNS / ASN</div>
                    {[
                      ['PTR Record', d().ptr || '—'],
                      ['ASN', d().asn || '—'],
                      ['Abuse Contact', d().abuse_contact || '—'],
                      ['Tor Exit', d().is_tor ? '⚠ YES' : 'No'],
                      ['VPN / Proxy', d().is_vpn ? '⚠ YES' : 'No'],
                    ].map(([k, v]) => (
                      <div style="display:flex; justify-content:space-between; margin-bottom:0.4rem; font-size:0.76rem;">
                        <span style="color:#607070;">{k}</span>
                        <span style={`color:${v.startsWith('⚠') ? '#ffaa00' : '#c8d8d8'};`}>{v}</span>
                      </div>
                    ))}
                  </div>
                )}
              </Show>

              {/* Asset Card */}
              <Show when={r().asset}>
                {(a) => (
                  <div style="background:#0d1a1f; border:1px solid #1e3040; border-top:2px solid #ffaa00; border-radius:6px; padding:1.25rem;">
                    <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">ASSET RECORD</div>
                    {[
                      ['Host ID', a().host_id || '—'],
                      ['Hostname', a().hostname || '—'],
                      ['OS', a().os || '—'],
                      ['Last Seen', a().last_seen ? new Date(a().last_seen!).toLocaleString() : '—'],
                    ].map(([k, v]) => (
                      <div style="display:flex; justify-content:space-between; margin-bottom:0.4rem; font-size:0.76rem;">
                        <span style="color:#607070;">{k}</span>
                        <span style="color:#c8d8d8;">{v}</span>
                      </div>
                    ))}
                    <Show when={a().risk_score !== undefined}>
                      <div style="margin-top:0.75rem;">
                        <div style="display:flex; justify-content:space-between; font-size:0.72rem; margin-bottom:0.3rem;">
                          <span style="color:#607070;">RISK SCORE</span>
                          <span style={`color:${riskColor(a().risk_score)}; font-weight:700;`}>{a().risk_score}/100</span>
                        </div>
                        <div style="height:6px; background:#1e3040; border-radius:3px;">
                          <div style={`height:100%; border-radius:3px; background:${riskColor(a().risk_score)}; width:${a().risk_score}%;`}></div>
                        </div>
                      </div>
                    </Show>
                    <Show when={a().tags?.length}>
                      <div style="margin-top:0.75rem; display:flex; flex-wrap:wrap; gap:0.3rem;">
                        <For each={a().tags}>{(tag) => (
                          <span style="background:#1e3040; color:#607070; padding:0.1rem 0.4rem; border-radius:2px; font-size:0.65rem;">{tag}</span>
                        )}</For>
                      </div>
                    </Show>
                  </div>
                )}
              </Show>

              {/* IOC Match Card */}
              <Show when={r().ioc_match}>
                {(ioc) => (
                  <div style={`background:#0d1a1f; border:1px solid ${ioc().matched ? '#ff3355' : '#1e3040'}; border-top:2px solid ${ioc().matched ? '#ff3355' : '#607070'}; border-radius:6px; padding:1.25rem;`}>
                    <div style="font-size:0.7rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">IOC CORRELATION</div>
                    <div style={`font-size:1rem; font-weight:700; color:${ioc().matched ? '#ff3355' : '#00ff88'}; margin-bottom:0.5rem;`}>
                      {ioc().matched ? '⚠ MATCH FOUND' : '✓ CLEAN'}
                    </div>
                    <Show when={ioc().matched}>
                      {[
                        ['Severity', ioc().severity ?? '—'],
                        ['Source', ioc().source ?? '—'],
                        ['Description', ioc().description ?? '—'],
                      ].map(([k, v]) => (
                        <div style="font-size:0.74rem; color:#607070; margin-bottom:0.25rem;">
                          <span style="color:#c8d8d8;">{k}:</span> {v}
                        </div>
                      ))}
                    </Show>
                  </div>
                )}
              </Show>
            </div>
          )}
        </Show>
      </Show>

      {/* Recent queries */}
      <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; overflow:hidden;">
        <div style="padding:0.7rem 1rem; background:#0a1318; border-bottom:1px solid #1e3040; display:flex; justify-content:space-between;">
          <span style="color:#607070; font-size:0.72rem; letter-spacing:0.1em;">RECENT QUERIES</span>
          <button onClick={() => refetchRecent()} style="background:none; border:none; color:#607070; cursor:pointer; font-size:0.72rem;">↻</button>
        </div>
        <Show when={!recent.loading} fallback={<div style="padding:1rem; color:#607070; font-size:0.76rem;">Loading…</div>}>
          <For each={recent() ?? []} fallback={<div style="padding:1rem; color:#607070; font-size:0.76rem;">No recent queries.</div>}>
            {(re) => (
              <div style="padding:0.6rem 1rem; border-bottom:1px solid #0a1318; display:flex; align-items:center; gap:1rem; font-size:0.76rem; cursor:pointer;"
                onClick={() => { setQuery(re.query); setSubmitted(re.query); }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}
              >
                <span style="color:#00ffe7;">{re.query}</span>
                <Show when={re.geo}><span style="color:#607070;">{re.geo!.country_code} — {re.geo!.org || re.geo!.asn}</span></Show>
                <Show when={re.ioc_match?.matched}><span style="color:#ff3355; font-size:0.68rem; letter-spacing:0.1em;">⚠ IOC</span></Show>
              </div>
            )}
          </For>
        </Show>
      </div>
    </div>
  );
}
