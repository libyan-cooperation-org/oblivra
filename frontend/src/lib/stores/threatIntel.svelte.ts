// OBLIVRA — Threat-Intel store (Phase 32).
//
// Loads IOC indicators from the backend and pushes them into the
// iocMatcher's compiled tables. Refreshes every 60 s by default —
// fast enough that a freshly-published IOC underlines in the
// operator's terminal within a minute, slow enough that the network
// chatter stays cheap.
//
// The store also exposes `loaded` so the UI can show a "no IOC
// matching active" indicator when the feed is cold or unreachable.

import { apiFetch } from '@lib/apiClient';
import { loadIOCs, type IOCRecord, type IOCKind } from '@lib/iocMatcher';

class ThreatIntelStore {
  loaded = $state(false);
  count = $state(0);
  lastRefresh = $state<string | null>(null);

  private refreshHandle: ReturnType<typeof setInterval> | null = null;

  /** Translate the backend indicator shape into the matcher's IOCRecord. */
  private toRecord(raw: any): IOCRecord | null {
    if (!raw || typeof raw.value !== 'string') return null;
    // Backend uses `type` for IOC type; map to the matcher's union.
    const t = String(raw.type ?? '').toLowerCase();
    let kind: IOCKind | null = null;
    if (t === 'ipv4' || t === 'ip')        kind = 'ipv4';
    else if (t === 'sha256')               kind = 'sha256';
    else if (t === 'md5')                  kind = 'md5';
    else if (t === 'domain' || t === 'fqdn') kind = 'domain';
    if (!kind) return null;
    return {
      value: raw.value,
      kind,
      source: String(raw.source ?? raw.feed ?? 'unknown'),
      severity: raw.severity ? String(raw.severity).toLowerCase() as any : undefined,
      firstSeen: raw.first_seen ?? undefined,
    };
  }

  /** Pull the current IOC table and push to the matcher. */
  async refresh(): Promise<number> {
    try {
      const res = await apiFetch('/api/v1/threatintel/indicators?limit=5000');
      if (!res.ok) {
        console.warn('[threatIntel] indicators GET failed', res.status);
        return 0;
      }
      const body = await res.json();
      const list: any[] = body.indicators ?? body ?? [];
      const records: IOCRecord[] = [];
      for (const r of list) {
        const rec = this.toRecord(r);
        if (rec) records.push(rec);
      }
      loadIOCs(records);
      this.count = records.length;
      this.loaded = true;
      this.lastRefresh = new Date().toISOString();
      return records.length;
    } catch (e) {
      console.warn('[threatIntel] refresh failed:', e);
      return 0;
    }
  }

  /** Start the 60 s refresh loop. Idempotent — safe to call repeatedly. */
  start(intervalMs = 60_000) {
    if (this.refreshHandle) return;
    void this.refresh();
    this.refreshHandle = setInterval(() => { void this.refresh(); }, intervalMs);
  }

  stop() {
    if (this.refreshHandle) clearInterval(this.refreshHandle);
    this.refreshHandle = null;
  }
}

export const threatIntelStore = new ThreatIntelStore();
