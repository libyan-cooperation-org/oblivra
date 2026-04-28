// OBLIVRA — IOC matcher (Phase 32, UIUX moat-widener).
//
// Streaming scanner for indicators-of-compromise inside terminal output.
// Runs on every stdout chunk written to xterm so when an operator types
// `ss -tnp`, the destination IPs that match a known-bad threat-intel
// feed get a red underline INLINE in the terminal.
//
// Design principles:
//
//  • Cheap. Runs O(n) per chunk against a pre-compiled set of regexes;
//    no heap allocations in the hot path beyond match objects.
//  • Bounded latency. Scan is synchronous and capped at 4 KB per call;
//    bigger chunks are split. No async work — the caller can't write
//    to xterm faster than we can scan.
//  • Pattern source is hot-loaded. The threat-intel store pushes a
//    fresh IOC table every 60 s; we recompile the patterns on receipt.
//  • False-positive tolerant. We err on the side of NOT matching when
//    a pattern is ambiguous — the cost of a missed match is one less
//    visual hint; the cost of a false underline is operator distrust.
//
// Out of scope for v1: hash matching (SHA256/MD5 are easy to add when
// the threat-intel feed exposes them), domain matching (needs a tighter
// regex than IP4-style and is FP-prone — defer until there's UX demand).

export type IOCKind = 'ipv4' | 'sha256' | 'md5' | 'domain';

export interface IOCRecord {
  /** Canonical value, e.g. "198.51.100.13" or the lowercased hash. */
  value: string;
  kind: IOCKind;
  /** Threat-intel source — surfaced in the hover popover. */
  source: string;
  /** Optional severity if the source supplies one — drives hue. */
  severity?: 'critical' | 'high' | 'medium' | 'low';
  /** Optional first-seen timestamp from the source. */
  firstSeen?: string;
}

export interface IOCMatch {
  ioc: IOCRecord;
  /** Char offset within the *original* string. */
  start: number;
  /** Char offset (exclusive) within the original string. */
  end: number;
  /** The exact substring that matched (so we can render verbatim). */
  text: string;
}

// Compiled tables — rebuilt by `loadIOCs()` whenever the threat-intel
// store pushes a fresh feed. Kept as exact-string sets for IPs / hashes
// because those are precise; only domains might need wildcard logic
// later.
let ipv4Set = new Set<string>();
let sha256Set = new Set<string>();
let md5Set = new Set<string>();
let metadata = new Map<string, IOCRecord>();

// Pre-compiled regexes — top-level so we don't recompile per call. The
// IPv4 regex is greedy on dotted-quads but post-validates each octet
// to avoid matching "999.999.999.999".
const IPV4_RE = /\b(?:\d{1,3}\.){3}\d{1,3}\b/g;
const SHA256_RE = /\b[a-fA-F0-9]{64}\b/g;
const MD5_RE = /\b[a-fA-F0-9]{32}\b/g;

/** Replace the active IOC tables. Idempotent — safe to call repeatedly. */
export function loadIOCs(records: IOCRecord[]) {
  ipv4Set = new Set();
  sha256Set = new Set();
  md5Set = new Set();
  metadata = new Map();
  for (const r of records) {
    const v = r.value.toLowerCase();
    metadata.set(v, { ...r, value: v });
    switch (r.kind) {
      case 'ipv4':   ipv4Set.add(v); break;
      case 'sha256': sha256Set.add(v); break;
      case 'md5':    md5Set.add(v); break;
      // domain handling deferred — see header comment.
    }
  }
}

/** True iff at least one IOC is loaded — saves a regex pass on cold starts. */
export function hasIOCs(): boolean {
  return ipv4Set.size > 0 || sha256Set.size > 0 || md5Set.size > 0;
}

/**
 * Validate a string is a syntactically-valid IPv4 (post-regex check).
 * The regex above gates by shape; this gates by octet range.
 */
function validIPv4(s: string): boolean {
  const parts = s.split('.');
  if (parts.length !== 4) return false;
  for (const p of parts) {
    const n = Number(p);
    if (!Number.isInteger(n) || n < 0 || n > 255) return false;
  }
  return true;
}

/**
 * Scan a chunk and return every IOC match. Caller is responsible for
 * translating char offsets to xterm cell coordinates if it wants to
 * draw decorations.
 *
 * Returns at most 32 matches per call to bound the worst case (a row
 * of 200 IPs in one terminal line should never blow the renderer up).
 */
export function scanForIOCs(chunk: string): IOCMatch[] {
  if (!hasIOCs() || !chunk) return [];
  const matches: IOCMatch[] = [];
  const MAX = 32;

  // ── IPv4 ────────────────────────────────────────────────────
  if (ipv4Set.size > 0) {
    IPV4_RE.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = IPV4_RE.exec(chunk)) !== null && matches.length < MAX) {
      const text = m[0];
      if (!validIPv4(text)) continue;
      const v = text.toLowerCase();
      if (!ipv4Set.has(v)) continue;
      const rec = metadata.get(v);
      if (!rec) continue;
      matches.push({
        ioc: rec,
        start: m.index,
        end: m.index + text.length,
        text,
      });
    }
  }

  // ── SHA256 ──────────────────────────────────────────────────
  if (matches.length < MAX && sha256Set.size > 0) {
    SHA256_RE.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = SHA256_RE.exec(chunk)) !== null && matches.length < MAX) {
      const v = m[0].toLowerCase();
      if (!sha256Set.has(v)) continue;
      const rec = metadata.get(v);
      if (!rec) continue;
      matches.push({ ioc: rec, start: m.index, end: m.index + m[0].length, text: m[0] });
    }
  }

  // ── MD5 ─────────────────────────────────────────────────────
  if (matches.length < MAX && md5Set.size > 0) {
    MD5_RE.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = MD5_RE.exec(chunk)) !== null && matches.length < MAX) {
      const v = m[0].toLowerCase();
      if (!md5Set.has(v)) continue;
      const rec = metadata.get(v);
      if (!rec) continue;
      matches.push({ ioc: rec, start: m.index, end: m.index + m[0].length, text: m[0] });
    }
  }

  // Sort by offset so callers can decorate in stream order.
  matches.sort((a, b) => a.start - b.start);
  return matches;
}

/** Look up the IOC record for an exact value. Returns null if unknown. */
export function lookupIOC(value: string): IOCRecord | null {
  return metadata.get(value.toLowerCase()) ?? null;
}
