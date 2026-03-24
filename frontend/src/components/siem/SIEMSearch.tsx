// SIEMSearch.tsx — Phase 2 Desktop: Wails-based SIEM query interface
import { Component, createSignal, createResource, createMemo, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';

const SEVERITY_MAP: Record<string, { label: string; color: string }> = {
  failed_login:     { label: 'CRIT', color: '#ff3355' },
  security_alert:   { label: 'HIGH', color: '#ff6600' },
  sudo_exec:        { label: 'MED',  color: '#ffaa00' },
  successful_login: { label: 'INFO', color: '#00ff88' },
};

function getSeverity(eventType: string) {
  return SEVERITY_MAP[eventType] ?? { label: 'LOW', color: 'var(--text-muted)' };
}

export const SIEMSearch: Component = () => {
  const [query, setQuery]     = createSignal('');
  const [limit, setLimit]     = createSignal(100);
  const [trigger, setTrigger] = createSignal(0);
  const [lastQuery, setLastQuery] = createSignal('');
  const [expandedRows, setExpandedRows] = createSignal<Set<number>>(new Set());

  const [events] = createResource(
    () => ({ q: lastQuery(), l: limit(), t: trigger() }),
    async ({ q, l }) => {
      setExpandedRows(new Set());
      if (!q.trim()) return [];
      try {
        const r = await (SIEMService as any).SearchHostEvents(q, l);
        return r || [];
      } catch (e) {
        console.error("SIEM Search Error:", e);
        return [];
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
    // Use the native field name formatting
    const filter = `${field}:"${val}"`;
    const newQuery = current ? `${current} AND ${filter}` : filter;
    setQuery(newQuery);
    runSearch();
  }

  const evts = () => events() ?? [];

  // Helper to handle Go vs JSON casing safely
  const getField = (e: any, key: string, upperKey: string) => e[key] !== undefined ? e[key] : e[upperKey];

  // ==========================================
  // 1. Time-Series Histogram
  // ==========================================
  const histogram = createMemo(() => {
    const es = evts();
    if (es.length === 0) return [];
    
    const times = es.map((e: any) => {
       const ts = getField(e, 'timestamp', 'Timestamp');
       return ts ? new Date(ts).getTime() : 0;
    }).filter((t: number) => t > 0);
    
    if (times.length === 0) return [];

    const minT = Math.min(...times);
    const maxT = Math.max(...times);
    
    const binCount = 40;
    const bins = Array.from({ length: binCount }, () => 0);
    const timeSpan = maxT - minT || 1; 
    
    es.forEach((e: any) => {
      const ts = getField(e, 'timestamp', 'Timestamp');
      if (!ts) return;
      const t = new Date(ts).getTime();
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
    const es = evts();
    const fieldCounts: Record<string, Record<string, number>> = {};
    
    const fieldsToTrack = [
      { key: 'host_id', upper: 'HostID' },
      { key: 'event_type', upper: 'EventType' },
      { key: 'source_ip', upper: 'SourceIP' },
      { key: 'user', upper: 'User' }
    ];

    es.forEach((e: any) => {
       fieldsToTrack.forEach(f => {
          const val = String(getField(e, f.key, f.upper) || '');
          if (val && val !== 'undefined') {
             if (!fieldCounts[f.key]) fieldCounts[f.key] = {};
             fieldCounts[f.key][val] = (fieldCounts[f.key][val] || 0) + 1;
          }
       });
    });
    
    return Object.entries(fieldCounts).map(([field, valMap]) => {
      const vals = Object.entries(valMap)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5);
      return { field, vals };
    });
  });

  return (
    <div style="padding:1.5rem; color:var(--text-primary); font-family:var(--font-mono); height:100%; display:flex; flex-direction:column; background:var(--bg-primary); overflow:hidden;">
      {/* Header & Query Bar */}
      <div style="margin-bottom:1rem; flex-shrink:0;">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:1rem;">
          <div style="display:flex; align-items:center; gap:0.75rem;">
            <span style="font-size:1.5rem;">🔎</span>
            <div>
              <h2 style="font-size:1.1rem; letter-spacing:0.1em; font-weight:700; margin:0; text-transform:uppercase;">SIEM Search</h2>
              <p style="margin:0; font-size:0.7rem; color:var(--text-muted);">
                High-performance Splunk-style event analysis and field extraction
              </p>
            </div>
          </div>
          <div style="text-align:right;">
             <Show when={lastQuery()}>
                <span style="font-size:1.2rem; color:var(--accent-primary); font-weight:700;">{evts().length}</span> 
                <span style="font-size:0.8rem; color:var(--text-muted); margin-left:0.5rem;">Events matched</span>
             </Show>
          </div>
        </div>

        <div style="display:flex; gap:0.5rem; background:var(--bg-secondary); padding:0.5rem; border-radius:6px; border:1px solid var(--glass-border);">
          <div style="display:flex; align-items:center; padding:0 0.5rem; color:var(--accent-primary); font-weight:700;">&gt;</div>
          <input
            id="siem-query-input"
            type="text"
            value={query()}
            onInput={e => setQuery(e.currentTarget.value)}
            onKeyDown={handleKey}
            placeholder='e.g. event_type:failed_login OR source_ip:192.168.* | stats count by user'
            style="flex:1; background:transparent; border:none; color:var(--text-primary); padding:0.4rem; font-size:0.85rem; font-family:inherit; outline:none;"
          />
          <div style="width:1px; background:var(--glass-border); margin:0 0.5rem;"></div>
          <select
            value={limit()}
            onChange={e => setLimit(Number(e.currentTarget.value))}
            style="background:transparent; border:none; color:var(--accent-primary); font-family:inherit; font-size:0.8rem; outline:none; cursor:pointer;"
          >
            {[50, 100, 250, 500, 1000].map(n => <option style="background:var(--bg-secondary)" value={n}>{n} results</option>)}
          </select>
          <button
            onClick={runSearch}
            style="background:rgba(87,139,255,0.15); color:var(--accent-primary); border:1px solid rgba(87,139,255,0.4); padding:0.4rem 1.5rem; border-radius:3px; font-weight:800; cursor:pointer; font-size:0.8rem; letter-spacing:0.1em; transition:0.2s;"
            onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='rgba(87,139,255,0.25)'}
            onMouseLeave={e => (e.currentTarget as HTMLElement).style.background='rgba(87,139,255,0.15)'}
          >
            {events.loading ? 'SEARCHING...' : 'SEARCH'}
          </button>
        </div>
      </div>

      {/* Main Content Area (Sidebar + Results) */}
      <div style="display:flex; gap:1.5rem; flex:1; overflow:hidden;">
        
        {/* Left Sidebar: Extracted Fields */}
        <div style="width:260px; flex-shrink:0; overflow-y:auto; padding-right:0.5rem;">
          <div style="font-size:0.75rem; color:var(--text-muted); font-weight:700; letter-spacing:0.1em; margin-bottom:1rem; border-bottom:1px solid var(--glass-border); padding-bottom:0.5rem;">
            INTERESTING FIELDS
          </div>
          <Show when={evts().length === 0 && !events.loading}>
            <div style="font-size:0.75rem; color:var(--text-muted); font-style:italic;">No events to extract.</div>
          </Show>
          <For each={extractedFields()}>
             {({ field, vals }) => (
               <div style="margin-bottom:1.2rem;">
                 <div style="font-size:0.75rem; color:var(--accent-primary); margin-bottom:0.4rem; font-weight:700;">{field}</div>
                 <For each={vals}>
                   {([val, count]) => (
                     <div 
                       onClick={() => appendFilter(field, val)}
                       style="display:flex; justify-content:space-between; align-items:center; font-size:0.75rem; padding:0.2rem 0.4rem; cursor:pointer; border-radius:3px; transition:0.1s;"
                       onMouseEnter={e => {
                         const el = e.currentTarget as HTMLElement;
                         el.style.background = 'rgba(255,255,255,0.05)';
                         el.style.color = 'var(--text-primary)';
                       }}
                       onMouseLeave={e => {
                         const el = e.currentTarget as HTMLElement;
                         el.style.background = 'transparent';
                         el.style.color = 'var(--text-secondary)';
                       }}
                       title={`Filter: ${field}="${val}"`}
                     >
                       <span style="color:inherit; max-width:180px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">
                         {val}
                       </span>
                       <span style="color:var(--text-muted); font-size:0.7rem; background:var(--bg-secondary); padding:0.1rem 0.4rem; border-radius:8px;">
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
          <div style="height:100px; display:flex; align-items:flex-end; gap:2px; margin-bottom:1rem; padding-bottom:0.5rem; border-bottom:1px solid var(--glass-border); flex-shrink:0;">
            <Show when={evts().length === 0 && !events.loading}>
              <div style="width:100%; height:100%; display:flex; align-items:center; justify-content:center; color:var(--text-muted); font-size:0.8rem;">
                No timeline data available
              </div>
            </Show>
            <For each={histogram()}>
              {(bin) => (
                <div 
                   title={`Time: ${bin.timeLabel} | Count: ${bin.count}`}
                   style={`flex:1; background:${bin.count > 0 ? 'var(--accent-primary)' : 'transparent'}; opacity:0.8; transition:0.2s; height:${bin.heightPct}%; min-height:${bin.count > 0 ? '4px' : '0'}; cursor:pointer;`}
                   onMouseEnter={e => (e.currentTarget as HTMLElement).style.opacity='1'}
                   onMouseLeave={e => (e.currentTarget as HTMLElement).style.opacity='0.8'}
                ></div>
              )}
            </For>
          </div>

          {/* Event List */}
          <div style="flex:1; overflow-y:auto; background:var(--bg-secondary); border:1px solid var(--glass-border); border-radius:6px; position:relative;">
            
            <Show when={events.loading}>
              <div style="position:absolute; inset:0; background:rgba(0,0,0,0.5); display:flex; align-items:center; justify-content:center; z-index:10; backdrop-filter:blur(2px);">
                 <span style="color:var(--accent-primary); font-weight:700; font-size:1.2rem; letter-spacing:0.2em; animation: pulse 1.5s infinite;">SEARCHING...</span>
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
                <tr style="border-bottom:1px solid var(--glass-border); background:var(--bg-primary); position:sticky; top:0; z-index:5;">
                  <th style="padding:0.6rem;"></th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:var(--text-muted); font-weight:400; letter-spacing:0.1em;">TIME</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:var(--text-muted); font-weight:400; letter-spacing:0.1em;">HOST</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:var(--text-muted); font-weight:400; letter-spacing:0.1em;">EVENT TYPE</th>
                  <th style="padding:0.6rem 0.9rem; text-align:left; color:var(--text-muted); font-weight:400; letter-spacing:0.1em;">RAW EVENT</th>
                </tr>
              </thead>
              <tbody>
                <Show when={!events.loading && evts().length === 0}>
                  <tr>
                    <td colspan="5" style="padding:4rem; text-align:center; color:var(--text-muted);">
                      <div style="font-size:3rem; opacity:0.2; margin-bottom:1rem;">⬡</div>
                      {lastQuery() ? '0 EVENTS MATCHED THIS QUERY' : 'READY FOR SEARCH'}
                    </td>
                  </tr>
                </Show>
                
                <For each={evts()}>
                  {(evt: any) => {
                    const id = getField(evt, 'id', 'ID');
                    const isExpanded = expandedRows().has(id);
                    const eventType = getField(evt, 'event_type', 'EventType');
                    const sev = getSeverity(eventType);
                    const rawLog = getField(evt, 'raw_log', 'RawLog');
                    const hostId = getField(evt, 'host_id', 'HostID');
                    const ts = getField(evt, 'timestamp', 'Timestamp');
                    const tenant = getField(evt, 'tenant_id', 'TenantID');
                    const srcIp = getField(evt, 'source_ip', 'SourceIP');
                    const u = getField(evt, 'user', 'User');
                    
                    let parsedLog: any = null;
                    if (isExpanded) {
                       try {
                          parsedLog = JSON.parse(rawLog);
                       } catch { }
                    }

                    return (
                      <>
                      <tr 
                        onClick={() => toggleRow(id)}
                        style="border-bottom:1px solid rgba(255,255,255,0.03); cursor:pointer;"
                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background='rgba(255,255,255,0.02)'}
                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background='transparent'}
                      >
                        <td style="padding:0.6rem; text-align:center; color:var(--accent-primary); font-size:0.7rem;">
                           {isExpanded ? '▼' : '▶'}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:var(--text-muted); white-space:nowrap;">
                          {ts ? new Date(ts).toISOString().replace('T',' ').slice(0,19) : '—'}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:var(--text-primary); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">
                          {hostId}
                        </td>
                        <td style={`padding:0.6rem 0.9rem; color:${sev.color}; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;`}>
                          {eventType}
                        </td>
                        <td style="padding:0.6rem 0.9rem; color:var(--text-muted); overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title={rawLog}>
                          <span style="color:var(--text-secondary);">{rawLog}</span>
                        </td>
                      </tr>
                      {/* Expanded Row Content */}
                      <Show when={isExpanded}>
                         <tr style="background:var(--bg-primary); border-bottom:1px solid var(--glass-border);">
                            <td colspan="5" style="padding:1.5rem;">
                               <div style="display:flex; gap:2rem;">
                                  <div style="flex:1;">
                                     <div style="color:var(--accent-primary); font-size:0.7rem; font-weight:700; margin-bottom:0.5rem; letter-spacing:0.1em;">EVENT DETAIL</div>
                                     <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                                        <tbody>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:var(--text-muted);">ID</td><td style="color:var(--text-primary);">{id}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:var(--text-muted);">Tenant</td><td style="color:var(--accent-primary);">{tenant || 'GLOBAL'}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:var(--text-muted);">Source IP</td><td style="color:var(--text-primary);">{srcIp || '—'}</td></tr>
                                           <tr><td style="padding:0.3rem 0; width:120px; color:var(--text-muted);">User</td><td style="color:var(--text-primary);">{u || '—'}</td></tr>
                                        </tbody>
                                     </table>
                                  </div>
                                  <div style="flex:2;">
                                     <div style="color:var(--accent-primary); font-size:0.7rem; font-weight:700; margin-bottom:0.5rem; letter-spacing:0.1em;">PARSED RAW LOG</div>
                                     <Show when={parsedLog} fallback={
                                        <div style="background:var(--bg-secondary); padding:0.8rem; border:1px solid var(--border-primary); border-radius:4px; font-family:var(--font-mono); color:var(--text-primary); white-space:pre-wrap; word-break:break-all;">
                                           {rawLog}
                                        </div>
                                     }>
                                        <table style="width:100%; border-collapse:collapse; font-size:0.75rem;">
                                           <tbody>
                                              <For each={Object.entries(parsedLog || {})}>
                                                 {([k, v]) => (
                                                    <tr style="border-bottom:1px solid transparent;" 
                                                        onMouseEnter={e=>(e.currentTarget as HTMLElement).style.borderBottomColor='var(--glass-border)'}
                                                        onMouseLeave={e=>(e.currentTarget as HTMLElement).style.borderBottomColor='transparent'}>
                                                       <td style="padding:0.2rem 0; color:var(--alert-medium); width:30%; vertical-align:top;">{k}</td>
                                                       <td style="padding:0.2rem 0; color:var(--accent-primary); word-break:break-word;">{typeof v === 'object' ? JSON.stringify(v) : String(v)}</td>
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
      
      <style>
         {`
           @keyframes pulse {
             0% { opacity: 0.5; }
             50% { opacity: 1; text-shadow: 0 0 10px var(--accent-primary); }
             100% { opacity: 0.5; }
           }
         `}
      </style>
    </div>
  );
};
