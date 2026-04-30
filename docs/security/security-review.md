# OBLIVRA — Security Review

This document describes the security posture an external reviewer should expect to see, the threat model the platform is designed against, and the explicit non-goals. It is the artefact a CISO can hand to counsel before a procurement decision.

## Threat model

OBLIVRA's primary adversary is an actor who has already compromised a host whose logs the platform collects, and who wants to hide what they did. The platform is **not** primarily a network-perimeter defence — it's the **system of record** that a defender uses to reconstruct what happened *after* the perimeter was crossed.

### Adversary capabilities assumed

1. Can write arbitrary log lines from a compromised host
2. Can stop / disable / clear logs on the compromised host
3. Cannot modify the OBLIVRA platform binary, its on-disk state, or its TLS keys (it lives on a separate, trust-boundary-isolated host)
4. May be able to file a false report against an honest analyst

### Adversary capabilities **not** assumed

- Cannot break SHA-256 or compromise the HMAC signing key
- Cannot perform an undetected memory-corruption attack on the running platform process
- Cannot modify `audit.log` or `cases.log` after they've been fsynced

If any of those assumptions break, the platform is no longer trustworthy — that's why the **offline verifier** is a single static binary that re-derives every cryptographic claim independently.

## Defences that are real

### Tamper-evident audit chain

- Every analyst action lands in `audit.log` (line-delimited JSON, fsynced).
- Each entry hashes the previous entry's hash → forward-chain Merkle structure.
- HMAC-signed when `OBLIVRA_AUDIT_KEY` is configured.
- Replay-on-startup refuses to boot a tampered chain.
- Daily Merkle anchor: every UTC day the chain root is hashed into a new entry — partial-day cherry-picking is detectable across the boundary.
- `oblivra-verify` reproduces the chain check without needing the platform.

**Failure mode:** if the audit-key environment variable is logged elsewhere (shell history, container envs API, CI variables UI), an attacker who recovers it can forge HMAC signatures on entries they crafted *before* obtaining the key. The platform cannot defend against this — operators must treat `OBLIVRA_AUDIT_KEY` as critically as a TLS private key.

### Tamper-evident query log

- `internal/httpserver/auditmw.go` wraps every audited route.
- Every search, OQL query, evidence export, vault unlock, rule reload, etc. lands in the chain with `{actor, role, method, path, status, bytes, duration, query, uaHash}`.
- Cherry-picked evidence is therefore self-incriminating: the analyst's query history is on the same chain as the events they cherry-picked from.

### Per-event content hash + provenance

- Every event carries `Hash = sha256(canonical(event))` plus a structured `Provenance` block (`ingestPath`, `peer`, `agentId`, `parser`, `tlsFingerprint`).
- Mutating any field — including provenance — breaks the hash.
- Round-trip stable: marshal, store, replay, re-derive → same hash.

### Time-frozen investigation snapshots

- Opening a case captures `{auditRootAtOpen, receivedAtCutoff}`.
- Subsequent timeline / search calls *through the case* exclude any event whose `receivedAt > cutoff`.
- The case persistence file itself is replayed and chain-anchored.

### WORM warm tier

- Parquet files are read-only after fsync (`internal/storage/worm`).
- On Windows, `SetFileAttributes` adds the read-only bit; on Unix, write bits are stripped from mode.
- A privileged-root attacker can still delete the file — WORM here is "tamper-detectable", not "tamper-proof". Detection is via the cross-tier verifier and the daily Merkle anchor.

### Vault

- AES-256-GCM with Argon2id KDF.
- Atomic file writes (write to `<path>.tmp`, `os.Rename` to final).
- ErrInvalidKey returned on GCM auth failure.
- Vault contents are not hashed into the audit chain (so unlocking the vault doesn't leak the secret structure to the chain) — but the unlock action is.

## Defences that are deliberately out of scope

These are explicitly **not** Beta-1 features. Operators wanting them should pair OBLIVRA with the noted external systems.

- **Hardware-bound vault keys** (TPM PCR, FIDO2/YubiKey) — Argon2id passphrase only for now
- **eBPF kernel-level collectors** — file-tail agent only
- **Network perimeter / IDS / IPS** — pair with Suricata/Zeek/equivalent
- **Active response (kill process, quarantine, isolate)** — explicitly cut in Phase 36
- **Mutual TLS at the agent boundary** — agents authenticate via Bearer tokens; mTLS is on the post-Beta roadmap

## Cryptographic primitives — implementation notes

| Primitive | Library | Use |
|---|---|---|
| SHA-256 | `crypto/sha256` (stdlib) | event content hash, audit chain, Merkle anchor |
| HMAC-SHA256 | `crypto/hmac` | audit entry signature |
| AES-256-GCM | `crypto/aes` + `crypto/cipher` | vault encryption |
| Argon2id | `golang.org/x/crypto/argon2` | vault KDF |
| RFC 3339Nano | `time` | canonical timestamp encoding for hashes |

No custom crypto. No "just trust me" RNG — every random source is `crypto/rand`.

## Operational posture for a production deployment

### Network exposure

| Port | Default | Recommended |
|---|---|---|
| 8080/tcp | HTTP REST + WebSocket + Svelte UI | Behind a reverse proxy (nginx, Caddy) terminating TLS 1.3+ |
| 1514/udp | Syslog ingest | Firewalled to known shippers' subnets |
| 2055/udp | NetFlow v5 | Firewalled to network gear |

The platform speaks plaintext on the loopback by default. **Production deployments must terminate TLS at a reverse proxy or front it behind an authenticated VPN.**

### Required environment

```bash
# Auth — comma-separated keys with optional ":role" suffix
OBLIVRA_API_KEYS="ops-secret:admin,sec-team:analyst,siem-readonly:readonly,fleet-bot:agent"

# HMAC signing key for the audit chain. Treat this like a TLS private key.
OBLIVRA_AUDIT_KEY="$(openssl rand -hex 32)"

# Where data lives. Must be on a filesystem with reliable fsync.
OBLIVRA_DATA_DIR="/var/lib/oblivra"

# Tighten listeners to the recipients you trust.
OBLIVRA_SYSLOG_ADDR=":1514"          # or empty to disable
OBLIVRA_NETFLOW_ADDR=":2055"
```

### Audit verification cadence

- **Continuous (5min)**: scheduler runs `audit.health` — refuses to boot on tamper, alerts on mid-run break.
- **Hourly**: `audit.daily-anchor` — the previous UTC day is hashed into the chain.
- **Manual / forensic**: `oblivra-verify --hmac $OBLIVRA_AUDIT_KEY audit.log` recomputes the entire chain from scratch.

### Backup expectations

- `audit.log` and `cases.log` must be backed up *before* daily cleanup of the data dir.
- The Parquet warm-tier files are WORM-locked; backup can copy them but should not delete originals — the cross-tier verifier reads from the originals.
- The vault file (`oblivra.vault`) backed up separately, encrypted at rest by the backup system.

## Reporting vulnerabilities

Email `security@<your-domain>` with reproduction steps. We aim for a 48-hour acknowledgement and a 14-day patch for confirmed receipts. Never open a public GitHub issue for a security finding.

## Appendix: defensive features that are *not* security claims

These are useful but operational, not adversarial:

- DLP redaction on the search surface — masks credit cards / AWS keys / etc. on display, but does **not** scrub them from on-disk events. The chain still verifies because the on-disk event is unmodified.
- Source reliability scoring — analysts can see which log sources are noisy or delayed, but a determined attacker who knows the heuristic can shape their log production to score well. Don't treat this as a defence.
- Trust classification — same caveat. "Verified" means agent-signed at ingest, not "ground truth".
