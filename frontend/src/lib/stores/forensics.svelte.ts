/**
 * OBLIVRA — Forensics Store (Svelte 5 runes)
 *
 * Manages evidence lifecycle and chain-of-custody verification.
 */
import { IS_BROWSER } from '@lib/context';

export interface EvidenceItem {
  id: string;
  name: string;
  type: string;
  size: string;
  timestamp: string;
  collector: string;
  sealed: boolean;
  hash: string;
}

export interface ChainEntry {
  timestamp: string;
  actor: string;
  action: string;
  notes: string;
}

export class ForensicsStore {
  items = $state<EvidenceItem[]>([]);
  activeChain = $state<ChainEntry[]>([]);
  loading = $state(false);

  async loadIncidentEvidence(incidentID: string) {
    this.loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListEvidence } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      const list = await ListEvidence(incidentID);
      this.items = (list || []).map((e: any) => ({
        id: e.id,
        name: e.name,
        type: e.type,
        size: e.size_human || `${(e.size / 1024 / 1024).toFixed(2)} MB`,
        timestamp: e.timestamp,
        collector: e.collector,
        sealed: e.sealed,
        hash: e.hash
      }));
    } catch (e) {
      console.error('[ForensicsStore] Failed to load evidence', e);
    } finally {
      this.loading = false;
    }
  }

  async loadChain(itemID: string) {
    if (IS_BROWSER) return;
    try {
      const { GetChainOfCustody } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      const chain = await GetChainOfCustody(itemID);
      this.activeChain = (chain || []).map((c: any) => ({
        timestamp: c.timestamp,
        actor: c.actor,
        action: c.action,
        notes: c.notes
      }));
    } catch (e) {
      console.error('[ForensicsStore] Failed to load chain', e);
    }
  }
}

export const forensicsStore = new ForensicsStore();
