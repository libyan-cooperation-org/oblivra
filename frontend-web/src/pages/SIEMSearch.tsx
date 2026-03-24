/**
 * SIEMSearch.tsx — Dedicated SIEM Search & Analytics Page (Hybrid — Phase 0.5)
 *
 * A full-featured Splunk-style query interface connecting to /api/v1/siem/search.
 * Supports Lucene-style queries, time-series histogram, field extraction, and raw event viewer.
 */

import { createSignal, createResource, createMemo, For, Show } from 'solid-js';
import { request } from '../services/api';

interface HostEvent {
  id: number;
  tenant_id: string;
  host_id: string;
  timestamp: string;
  event_type: string;
  source_ip: string;
  user: string;
  raw_log: string;
}

interface SearchResponse {
  events: HostEvent[];
  total: number;
}

const SEVERITY_MAP: Record<string, { label: string; color: string }> = {
  failed_login:     { label: 'CRIT', color: '#ff3355' },
  security_alert:   { label: 'HIGH', color: '#ff6600' },
  sudo_exec:        { label: 'MED',  color: '#ffaa00' },
  successful_login: { label: 'INFO', color: '#00ff88' },
};

function getSeverity(eventType: string) {
  return SEVERITY_MAP[eventType] ?? { label: 'LOW', color: '#607070' };
}

export default function SIEMSearch() {
  const [query, setQuery]     = createSignal('');
  const [limit, setLimit]     = createSignal(100);
  const [trigger, setTrigger] = createSignal(0);
  const [lastQuery, setLastQuery] = createSignal('');
  const [expandedRows, setExpandedRows] = createSignal<Set<number>>(new Set());

  const [results] = createResource(
    () => ({ q: lastQuery(), l: limit(), t: trigger() }),
    async ({ q, l }) => {
      setExpandedRows(new Set<number>()); // Reset expansions on new search
      if (!q.trim()) return { events: [], total: 0 };
      try {
        const params = new URLSearchParams({ q, limit: String(l) });
        return await request<SearchResponse>(`/siem/search?${params}`);
      } catch {
        return { events: [], total: 0 };
      }
    }
  );

  function runSearch() {
    setLastQuery(query());
    setTrigger(t => t + 1);
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === 'Enter') runSearch();
  }

  function toggleRow(id: number) {
    const next = new Set(expandedRows());
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setExpandedRows(next);
  }

  function appendFilter(field: string, val: string) {
    const current = query().trim();
    const filter = `${field}:"${val}"`;
    const newQuery = current ? `${current} AND ${filter}` : filter;
    setQuery(newQuery);
    runSearch();
  }

  const events = () => results()?.events ?? [];

  // ==========================================
  // 1. Time-Series Histogram
  // ==========================================
  const histogram = createMemo(() => {
    const evts = events();
    if (evts.length === 0) return [];
    
    const times = evts.map(e => new Date(e.timestamp).getTime());
    const minT = Math.min(...times);
    const maxT = Math.max(...times);
    
    // Create 40 bins
    const binCount = 40;
    const bins = Array.from({ length: binCount }, () => 0);
    const timeSpan = maxT - minT || 1; // avoid division by zero
    
    evts.forEach(e => {
      const t = new Date(e.timestamp).getTime();
      let idx = Math.floor(((t - minT) / timeSpan) * binCount);
      if (idx >= binCount) idx = binCount - 1;
      bins[idx]++;
    });
    
    const maxCount = Math.max(...bins, 1);
    
    return bins.map((count, i) => {
       const binTime = new Date(minT + (i / binCount) * timeSpan);
       return {
         count,
         heightPct: (count / maxCount) * 100,
         timeLabel: binTime.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
       };
    });
  });

  // ==========================================
  // 2. Interesting Fields Extraction
  // ==========================================
  const extractedFields = createMemo(() => {
    const evts = events();
    const fieldCounts: Record<string, Record<string, number>> = {};
    
    evts.forEach(e => {
       const keys: (keyof HostEvent)[] = ['host_id', 'event_type', 'source_ip', 'user'];
       keys.forEach(k => {
          const val = String(e[k] || '');
          if (val && val !== 'undefined') {
             if (!fieldCounts[k]) fieldCounts[k] = {};
             fieldCounts[k][val] = (fieldCounts[k][val] || 0) + 1;
          }
       });
    });
    
    // Convert to sorted array
    return Object.entries(fieldCounts).map(([field, valMap]) => {
      const vals = Object.entries(valMap)
        .sort((a, b) => b[1] - a[1]) // sort by count desc
        .slice(0, 5); // top 5
      return { field, vals };
    });
  });

  return (
    <div style="padding:1.5rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12; display:flex; flex-direction:column;">
      {/* Header & Query Bar */}
      <div style="margin-bottom:1rem;">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:1rem;">
          <div>
            <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#00ffe7;">⬡ OBLIVRA SEARCH</h1>
            <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">
              High-performance Splunk-style event analysis and field extraction
            </p>
          </div>
          <div style="text-align:right;">
             <Show when={lastQuery()}>
                <span style="font-size:1.2rem; color:#00ffe7; font-weight:700;">{events().length}</span> 
                <span style="font-size:0.8rem; color:#607070; margin-left:0.5rem;">Events matched</span>
             </Show>
          </div>
        </div>

        <div style="display:flex; gap:0.5rem; background:#0d1a1f; padding:0.5rem; border-radius:6px; border:1px solid #1e3040;">
          <div style="display:flex; align-items:center; padding:0 0.5rem; color:#00ffe7; font-weight:700;">&gt;</div>
          <input
            id="siem-query-input"
            type="text"
            value={query()}
            onInput={e => setQuery(e.currentTarget.value)}
            onKeyDown={handleKey}
            placeholder='e.g. event_type:failed_login OR source_ip:192.168.* | stats count by user'
            style="flex:1; background:transparent; border:none; color:#c8d8d8; padding:0.4rem; font-size:0.85rem; font-family:inherit; outline:none;"
          />
          <div style="width:1px; background:#1e3040; margin:0 0.5rem;"></div>
          <select
            value={limit()}
            onChange={e => setLimit(Number(e.currentTarget.value))}
            style="background:transparent; border:none; color:#00ffe7; font-family:inherit; font-size:0.8rem; outline:none; cursor:pointer;"
          >
            {[50, 100, 250, 500, 1000].map(n => <option style="background:#0d1a1f" value={n}>{n} results</option>)}
          </select>
          <button
            onClick={runSearch}
            style="background:#00ffe7; color:#080f12; border:none; padding:0.4rem 1.5rem; border-radius:3px; font-weight:800; cursor:pointer; font-size:0.8rem; letter-spacing:0.1em; transition:0.2s;"
            onMouseEnter={e => (e.currentTarget as HTMLElement).style.opacity='0.8'}
            onMouseLeave={e => (e.currentTarget as HTMLElement).style.opacity='1'}
          >
            {results.loading ? 'SEARCHING...' : 'SEARCH'}
          </button>
        </div>
      </div>

      {/* Main Content Area (Sidebar + Results) */}
      <div style="display:flex; gap:1.5rem; flex:1; overflow:hidden;">
        
        {/* Left Sidebar: Extracted Fields */}
        <div style="width:260px; flex-shrink:0; overflow-y:auto; padding-right:0.5rem;">
          <div style="font-size:0.75rem; color:#607070; font-weight:700; letter-spacing:0.1em; margin-bottom:1rem; border-bottom:1px solid #1e3040; padding-bottom:0.5rem;">
            INTERESTING FIELDS
          </div>
          <Show when={events().length === 0 && !results.loading}>
            <div style="font-size:0.75rem; color:#607070; font-style:italic;">No events to extract.</div>
          </Show>
          <For each={extractedFields()}>
             {({ field, vals }) => (
               <div style="margin-bottom:1.2rem;">
                 <div style="font-size:0.75rem; color:#00ffe7; margin-bottom:0.4rem; font-weight:700;">{field}</div>
                 <For each={vals}>
                   {([val, count]) => (
                     <div 
                       onClick={() => appendFilter(field, val)}
                       style="display:flex; justify-content:space-between; align-items:center; font-size:0.75rem; padding:0.2rem 0.4rem; cursor:pointer; border-radius:3px; transition:0.1s;"
                       onMouseEnter={e => {
                         const el = e.currentTarget as HTMLElement;
                         el.style.background = '#111f28';
                         el.style.color = '#fff';
                       }}
                       onMouseLeave={e => {
                         const el = e.currentTarget as HTMLElement;
                         el.style.background = 'transparent';
                         el.style.color = '#c8d8d8';
                       }}
                       title={`Filter: ${field}="${val}"`}
                     >
                       <span style="color:inherit; max-width:180px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
                         {val}
                       </span>
                       <span style="color:#607070; font-size:0.7rem; background:#0d1a1f; padding:0.1rem 0.4rem; border-radius:8px;">
                         {count}
                       </span>
                     </div>
                   )}
                 </For>
               </div>
             )}
          </For>
        </div>

        {/* Right Area: Timeline & Event List */}
        <div style="flex:1; display:flex; flex-direction:column; min-width:0;">
          
          {/* Time-Series Histogram */}
          <div style="height:100px; display:flex; align-items:flex-end; gap:2px; margin-bottom:1rem; padding-bottom:0.5rem; border-bottom:1px solid #1e3040;">
            <Show when={events().length === 0 && !results.loading}>
              <div style="width:100%; height:100%; display:flex; align-items:center; justify-content:center; color:#607070; font-size:0.8rem;">
                No timeline data available
              </div>
            </Show>
            <For each={histogram()}>
              {(bin) => (
                <div 
                   title={`Time: ${bin.timeLabel} | Count: ${bin.count}`}
                   style={`flex:1; background:${bin.count > 0 ? '#00ffe7' : 'transparent'}; opacity:0.8; transition:0.2s; height:${bin.heightPct}%; min-height:${bin.count > 0 ? '4px' : '0'}; cursor:pointer;`}
                   onMouseEnter={e => (e.currentTarget as HTMLElement).style.opacity='1'}
                   onMouseLeave={e => (e.currentTarget as HTMLElement).style.opacity='0.8'}
                ></div>
              )}
            </For>
          </div>

          {/* Event List */}
          <div style="flex:1; overflow-y:auto; background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; position:relative;">
            
            <Show when={results.loading}>
              <div style="position:absolute; inset:0; background:rgba(8,15,18,0.7); display:flex; align-items:center; justify-content:center; z-index:10; backdrop-filter:blur(2px);">
                 <span style="color:#00ffe7; font-weight:700; font-size:1.2rem; letter-spacing:0.2em; animation: pulse 1.5s infinite;">SEARCHING...</span>
              </div>
            </Show>

            <table style="width:100%; border-collapse:collapse; font-size:0.76rem; table-layout:fixed;">
              <colgroup>
                 <col style="width:2.5rem;" />
                 <col style="width:11rem;" />
                 <col style="width:8rem;" />
                 <col style="width:8rem;" />
                 <col />
              </colgroup>
              <thead>
                <tr style="border-bottom:1px solid #1e3040; background:#0a1318; position:sticky; top:0; z-index:5;">
                  <th style="padding:0.6rem;"></th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; font-weight:400; letter-spacing:0.1em;">TIME</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; font-weight:400; letter-spacing:0.1em;">HOST</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; font-weight:400; letter-spacing:0.1em;">EVENT TYPE</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:#607070; font-weight:400; letter-spacing:0.1em;">RAW EVENT</th>
                </tr>
              </thead>
              <tbody>
                <Show when={!results.loading && events().length === 0}>
                  <tr>
                    <td colspan="5" style="padding:4rem; text-align:center; color:#607070;">
                      <div style="font-size:3rem; opacity:0.2; margin-bottom:1rem;">⬡</div>
                      {lastQuery() ? '0 EVENTS MATCHED THIS QUERY' : 'READY FOR SEARCH'}
                    </td>
                  </tr>
                </Show>
                
                <For each={events()}>
                  {(evt) => {
                    const isExpanded = expandedRows().has(evt.id);
                    const sev = getSeverity(evt.event_type);
                    
                    // Attempt to parse raw_log for the expanded view
                    let parsedLog: any = null;
                    if (isExpanded) {
                       try {
                          parsedLog = JSON.parse(evt.raw_log);
                       } catch {
                          // Not JSON, just display raw
                       }
                    }

                    return (
                      <>
                      <tr 
                        onClick={() => toggleRow(evt.id)}
                        style="border-bottom:1px solid #111f28; cursor:pointer;"
                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='#111f28'}
                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background=''}
                      >
                        <td style="padding:0.6rem; text-align:center; color:#00ffe7; font-size:0.7rem;">
                           {isExpanded ? '▼' : '▶'}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:#607070; white-space:nowrap;">
                          {new Date(evt.timestamp).toISOString().replace('T',' ').slice(0,19)}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:#c8d8d8; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">
                          {evt.host_id}
                        </td>
                        <td style={`padding:0.6rem 0.9rem; color:${sev.color}; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;`}>
                          {evt.event_type}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:#607070; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={evt.raw_log}>
                          <span style="color:#c8d8d8;">{evt.raw_log}</span>
                        </td>
                      </tr>
                      {/* Expanded Row Content */}
                      <Show when={isExpanded}>
                         <tr style="background:#0a1318; border-bottom:1px solid #1e3040;">
                            <td colspan="5" style="padding:1.5rem;">
                               <div style="display:flex; gap:2rem;">
                                  <div style="flex:1;">
                                     <div style="color:#00ffe7; font-size:0.7rem; font-weight:700; margin-bottom:0.5rem; letter-spacing:0.1em;">EVENT DETAIL</div>
                                     <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                                        <tbody>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:#607070;">ID</td><td style="color:#c8d8d8;">{evt.id}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:#607070;">Tenant</td><td style="color:#00ffe7;">{evt.tenant_id || 'GLOBAL'}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:#607070;">Source IP</td><td style="color:#c8d8d8;">{evt.source_ip || '—'}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:#607070;">User</td><td style="color:#c8d8d8;">{evt.user || '—'}</td></tr>
                                        </tbody>
                                     </table>
                                  </div>
                                  <div style="flex:2;">
                                     <div style="color:#00ffe7; font-size:0.7rem; font-weight:700; margin-bottom:0.5rem; letter-spacing:0.1em;">PARSED RAW LOG</div>
                                     <Show when={parsedLog} fallback={
                                        <div style="background:#080f12; padding:0.8rem; border:1px solid #1e3040; border-radius:4px; font-family:'JetBrains Mono',monospace; color:#c8d8d8; white-space:pre-wrap; word-break:break-all;">
                                           {evt.raw_log}
                                        </div>
                                     }>
                                        <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                                           <tbody>
                                              <For each={Object.entries(parsedLog || {})}>
                                                 {([k, v]) => (
                                                    <tr style="border-bottom:1px solid transparent;" 
                                                        onMouseEnter={e=>(e.currentTarget as HTMLElement).style.borderBottomColor='#1e3040'}
                                                        onMouseLeave={e=>(e.currentTarget as HTMLElement).style.borderBottomColor='transparent'}>
                                                       <td style="padding:0.2rem 0; color:#ffaa00; width:30%; vertical-align:top;">{k}</td>
                                                       <td style="padding:0.2rem 0; color:#00ffe7; word-break:break-word;">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</td>
                                                    </tr>
                                                 )}
                                              </For>
                                           </tbody>
                                        </table>
                                     </Show>
                                  </div>
                               </div>
                            </td>
                         </tr>
                      </Show>
                      </>
                    );
                  }}
                </For>
              </tbody>
            </table>
          </div>
        </div>

      </div>
      
      {/* Global pulse animation for loading state */}
      <style>
         {`
           @keyframes pulse {
             0% { opacity: 0.5; }
             50% { opacity: 1; text-shadow: 0 0 10px #00ffe7; }
             100% { opacity: 0.5; }
           }
         `}
      </style>
    </div>
  );
}
