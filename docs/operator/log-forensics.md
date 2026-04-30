# Operator Guide: Log Forensics

OBLIVRA's primary positioning (post-Phase 36) is **a sovereign, log-driven security and forensic intelligence platform**. This guide walks operators through the forensic workflow: from collection to detection to evidence sealing.

> **What OBLIVRA is**: a high-integrity log SIEM that produces admissible-grade evidence packages.
>
> **What OBLIVRA is not**: a SOAR, an IR platform, an EDR, an AI assistant, or a disk/memory forensic acquisition tool. Pair OBLIVRA with dedicated tools (Tines/XSOAR for response, Velociraptor/FTK/Volatility for DFIR, Drata/Vanta for compliance attestation).

---

## 1. Trust boundaries

Every event flowing through OBLIVRA is implicitly tagged with one of three trust levels:

| Tier | Meaning | Example sources |
|---|---|---|
| **TE — Trusted Event** | Cryptographically signed at the ingestion source | OBLIVRA agent (mTLS + per-agent ed25519 signature) |
| **VE — Verified Event** | Validated through a chain-of-trust at ingest time | Syslog over TLS with mutual-auth client cert |
| **BE — Best-Effort Event** | Unauthenticated telemetry | Plain UDP syslog, JSON over plaintext HTTP |

**Hard rule**: Best-Effort Events are never used as **sole** evidence. They can corroborate TE/VE events but cannot stand alone in an evidence package. Phase 37 enforces this in the export pipeline.

---

## 2. Collection

### Agent (preferred — produces TE)

The OBLIVRA agent is the highest-trust source.

```bash
# Linux/macOS
oblivra-agent --tenant=acme --server=https://siem.acme.local:8443

# Windows (services manager)
oblivra-agent.exe install --tenant=acme --server=https://siem.acme.local:8443
oblivra-agent.exe start
```

Defaults that ship with the agent:
- File tailing: `/var/log/auth.log`, `/var/log/syslog`, `C:\Windows\System32\winevt\Logs\*.evtx`
- Windows Event Log: top 30 event IDs deep-parsed (logons, process creation, scheduled tasks, etc.)
- journald: top 20 unit types
- File Integrity Monitoring: `/etc/`, `/usr/bin/`, `C:\Windows\System32\`
- Metrics: CPU, memory, network, load
- Local WAL: 100 MB ring buffer for offline operation

### Agentless

For systems where you can't deploy the agent:

| Source | Configuration |
|---|---|
| WMI | Windows event log via WMI/WinRM (REST API: `POST /api/v1/agentless/collectors`) |
| SNMP | v2c/v3 trap listener on UDP 162 |
| REST poll | Declarative config for SaaS sources (Okta, GitHub, Cloudflare) |
| Syslog | RFC 5424/3164 on TCP/UDP 514 (use TLS for VE tier) |

---

## 3. Detection

Detection runs continuously over every ingested event:

1. **Sigma engine** — community Sigma `.yml` rules + 82+ built-in rules. Drop rules into `sigma/` and they hot-reload.
2. **OQL queries** — pipe-syntax query language. Save and schedule queries via the OQL Dashboard.
3. **Correlation engine** — multi-stage detection (threshold / frequency / sequence / temporal / graph rules).
4. **UEBA** — entity baselines + Isolation Forest anomaly scoring per tenant.
5. **NDR** — NetFlow/IPFIX, DNS tunneling heuristics, JA3/JA3S fingerprinting.

### Forensic search templates (Phase 37, in progress)

Once Phase 37 lands, the OQL Dashboard ships with operator-ready forensic templates:

```
# Failed-then-successful login (credential stuffing pivot)
event_type:auth.failed by user
| join event_type:auth.success on user within 5m
| where outcome="success"
| timeline by user

# Process-tree reconstruction from a suspicious parent
event_type:process.create
| where parent_pid={ROOT_PID}
| recurse on parent_pid -> pid
| timeline by ts
```

---

## 4. Evidence sealing

This is the differentiator. When you find something investigation-worthy:

### From the UI
1. Navigate to **Evidence Vault** (Govern domain)
2. Drag selected events into the active evidence locker
3. Click **Seal** — events are hashed, Merkle-chained, RFC 3161 timestamped, and locked

### From OQL
```
oql search "event_type:auth.success AND host:dc01"
| evidence-pack name="DC01-suspicious-auth-2026-04-30" timestamp=true sign=true
```

### From REST
```bash
curl -X POST https://siem.acme.local:8443/api/v1/audit/packages/generate \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"framework":"audit","from":"2026-04-30T00:00:00Z","to":"2026-04-30T23:59:59Z"}'
```

### What's in the sealed package

| Component | Purpose |
|---|---|
| `events.json` | The raw events, in canonical order |
| `manifest.json` | Per-event hash + global Merkle root |
| `chain.json` | Chain-of-custody log (who touched it, when) |
| `tsa-token.p7s` | RFC 3161 timestamp from FreeTSA (or your configured TSA) |
| `signature.bin` | TPM-bound or vault-keyed signature over the manifest |
| `verify.sh` / `verify.ps1` | Self-contained offline verifier |

> Phase 38 hardens this into a court-admissible bundle (PDF/HTML report + WORM mode + offline verification CLI). Until then, the sealed package is forensically sound but not yet legally vetted.

---

## 5. Verification (third-party)

Anyone can verify a sealed evidence package, **offline**, without connecting to your OBLIVRA instance:

```bash
unzip evidence-pack-DC01-suspicious-auth-2026-04-30.zip
cd evidence-pack-DC01-suspicious-auth-2026-04-30/
./verify.sh

# Output:
# ✓ Manifest hash matches Merkle root
# ✓ All event hashes verify against manifest
# ✓ RFC 3161 timestamp verifies (issued 2026-04-30T14:23:01Z by FreeTSA)
# ✓ TPM signature verifies (ed25519 pubkey: a3:f4:...:9c)
# Verdict: SEALED — unmodified since 2026-04-30T14:23:01Z
```

**Verification is the entire point.** If you cannot independently verify a sealed package on a machine that has never touched OBLIVRA, the evidence is worthless.

---

## 6. Gap markers (Phase 37 in progress)

Logs cannot prove what was not logged. OBLIVRA is honest about this:

- The forensic timeline view annotates **gap markers** wherever telemetry was unavailable, the agent was offline, or a heartbeat was missed.
- The evidence-pack manifest includes a `gaps` section: `{"agent": "host-fin-044", "from": "...", "to": "...", "reason": "agent_offline_clock_drift_42s"}`.
- An attacker who disabled logging during their dwell time leaves a **gap marker, not a clean record** — and that gap is itself signal.

---

## 7. What we don't do (and why)

| Operator request | Why we don't do it | Pair with |
|---|---|---|
| "Isolate this host." | Active response is out of scope (Phase 36). | Tines, XSOAR, Shuffle |
| "Image the disk and pull memory." | Forensic acquisition is its own discipline; we'd do it badly. | Velociraptor, FTK, Volatility |
| "Kill PID 4412 on host X." | Same — pair with SOAR. | (same as above) |
| "Generate the SOC 2 report." | We removed compliance YAML packs in Phase 36.x. | Drata, Vanta, Tugboat Logic |
| "Run an LLM over our incidents." | We removed the AI assistant. | Bring your own LLM |

---

## 8. Operator FAQ

**Q: Why is my evidence pack so small?**
A: Because we only seal what was queried. A search returning 47 events makes a pack of ~80 KB. Phase 38 will add expand-to-related-context.

**Q: Can I seal a continuous detection over a time range?**
A: Yes — schedule an OQL query, attach `evidence-pack` at the end of the pipe. The package includes both the query and the matching events at execution time.

**Q: What happens if my TSA is down?**
A: The seal proceeds without the RFC 3161 timestamp; the manifest includes `tsa_status: "unavailable"`. You can re-anchor later via `POST /api/v1/audit/packages/{id}/reanchor`. Until then, the package has only your TPM/vault signature.

**Q: Can I export to disk for air-gapped review?**
A: Yes — every sealed package is a self-contained zip. Copy it to a USB stick, open on any machine, run `verify.sh`. No network required.

---

## See also

- [`README.md`](../../README.md) — overall positioning
- [`FEATURES.md`](../FEATURES.md) — feature matrix with status tiers
- [`task.md`](../../task.md) — build phases (Phase 37/38/39 active)
- [`HARDENING.md`](../../HARDENING.md) — security ledger
- [`api-reference.md`](api-reference.md) — REST API reference
- [`detection-authoring.md`](detection-authoring.md) — writing Sigma + OQL rules
- [`sigma-rules.md`](sigma-rules.md) — Sigma rule pack reference
