// OBLIVRA — Next-Best-Action client (Phase 32).
//
// Thin wrapper around POST /api/v1/alerts/recommend. The recommender
// is server-side and stateless, so this module is a pure-function
// `recommend()` that returns a typed RecommendedAction.
//
// Why POST instead of caching client-side: the rule set will evolve
// (new MITRE-aware multipliers, new context fields). Centralising the
// logic on the server means a single binary upgrade rolls the change
// to every operator instantly.

import { apiPostJSON } from './apiClient';

export type NBAAction =
  | 'quarantine_host'
  | 'evidence_capture'
  | 'escalate_tier_3'
  | 'investigate_host'
  | 'watch_only'
  | 'suppress_as_fp';

export interface NBAFacts {
  alert_id: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info' | string;
  category?: string; // mitre tactic, e.g. "command-and-control"

  has_ioc_match?: boolean;
  ioc_source?: string;
  host_known?: boolean;
  host_is_critical?: boolean;
  is_repeat_offender?: boolean;
  has_outbound_c2_beacon?: boolean;
  is_first_time_binary?: boolean;
  user_is_service?: boolean;
  is_from_crown_jewel?: boolean;
}

export interface RecommendedAction {
  action: NBAAction;
  confidence: number;
  reason: string;
  alternatives: NBAAction[];
}

/**
 * Map an alert action to a human-readable label for the UI button.
 * Keep these short — they live inside a 32-px button and need to read
 * fast under stress.
 */
export const ACTION_LABEL: Record<NBAAction, string> = {
  quarantine_host:  'Quarantine host',
  evidence_capture: 'Capture evidence',
  escalate_tier_3:  'Escalate to T3',
  investigate_host: 'Investigate host',
  watch_only:       'Watch only',
  suppress_as_fp:   'Suppress as FP',
};

/**
 * Map an action to the variant the Button should render with. Matches
 * the colour rules in the design system: containment / destructive →
 * critical; investigative → primary; passive → secondary.
 */
export const ACTION_VARIANT: Record<NBAAction, 'critical' | 'primary' | 'secondary' | 'cta'> = {
  quarantine_host:  'critical',
  evidence_capture: 'primary',
  escalate_tier_3:  'cta',
  investigate_host: 'primary',
  watch_only:       'secondary',
  suppress_as_fp:   'secondary',
};

export async function recommend(facts: NBAFacts): Promise<RecommendedAction> {
  const res = await apiPostJSON('/api/v1/alerts/recommend', facts);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text().catch(() => '')}`);
  }
  return (await res.json()) as RecommendedAction;
}

/** Quick local fallback used when the network is down — keeps the UI
 *  responsive instead of locking up. Operator gets a generic "investigate"
 *  recommendation with confidence 0 so they know it's a fallback. */
export function fallbackRecommendation(severity: string): RecommendedAction {
  const sev = (severity || '').toLowerCase();
  if (sev === 'critical')               return { action: 'evidence_capture', confidence: 0,    reason: 'Offline fallback — capture and escalate', alternatives: ['escalate_tier_3'] };
  if (sev === 'high' || sev === 'med' || sev === 'medium') {
    return { action: 'investigate_host', confidence: 0, reason: 'Offline fallback', alternatives: ['watch_only'] };
  }
  return { action: 'watch_only', confidence: 0, reason: 'Offline fallback', alternatives: ['suppress_as_fp'] };
}
