<div align="center">

# OBLIVRA

**Sovereign Log-Driven Security Platform**

*A self-hosted, air-gap-ready forensic SIEM. Log-driven detection, time-frozen
investigations, cryptographically verifiable evidence packages, single binary.*

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![Svelte](https://img.shields.io/badge/Svelte-5-FF3E00?logo=svelte)](https://svelte.dev)
[![Wails](https://img.shields.io/badge/Wails-v3-red?logo=wails)](https://wails.io)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-informational)](#building-from-source)

[Why](#why-oblivra) · [Quick Start](#quick-start) · [Agent](#agent) · [API](#api-surface) · [Building](#building-from-source) · [Backup verify](#backup-integrity-verification) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md)

</div>

---

## What it is

OBLIVRA is a **forensic SIEM** — a log-driven security platform optimised for
the question *"can we prove what happened?"* rather than *"can we react in
60 seconds?"*. It ships as one Go binary (Wails v3 desktop or headless server)
plus a separate forwarder binary. No external dependencies. No SaaS calls.

The core differentiators:

- **Time-frozen investigations** — opening a case captures the audit-chain
  root + receivedAt cutoff at that instant; every subsequent query goes
  through that snapshot lens, so the evidence you ship is byte-identical
  for the same case at the same audit-root.
- **Cryptographically verifiable evidence** — events carry a content hash,
  the audit log is SHA-256 Merkle-chained with optional HMAC root signature,
  and a daily anchor caps each UTC day. The verifier ships as a separate
  static binary an analyst can run on an air-gapped review box.
- **Per-event ed25519 signing at the agent edge** — even an MITM that
  decrypts TLS can't mutate events without invalidating the per-host
  signature, because the signing key never leaves the host.
- **Operator-grade audit trail** — every state-changing API call lands in
  the same tamper-evident chain as the events it observes; analyst actions
  *are* events.

## Why OBLIVRA?

| Concern | How OBLIVRA addresses it |
|---|---|
| **Data sovereignty** | Single binary, air-gap default. No SaaS callbacks. |
| **Auditable state of record** | Merkle-chained audit log; daily anchor; offline verifier. |
| **Forensic reconstruction** | Time-frozen cases; deterministic timeline; entity profiles; cross-protocol auth chains. |
| **Multi-tenant isolation** | Per-tenant BadgerDB key prefix; per-tenant Bleve index. |
| **Operator UX** | Hand-edited YAML config; CLI for everything; no in-app wizards required. |

## What it explicitly is *not*

- **Not a SOAR** — pair with Tines, Torq, n8n, or your own scripts. We emit
  webhooks (HMAC-signed); we don't run playbooks.
- **Not an EDR** — we ingest logs, including from EDR vendors. We don't
  isolate hosts or kill processes.
- **Not a copilot** — no LLM features. The audit chain is the story.
- **Not a compliance certifier** — pair with Drata / Vanta / Tugboat. We
  produce audit-grade evidence; they map it to control IDs.

---

## Quick start

### Prerequisites

| Tool | Minimum | Install |
|---|---|---|
| Go | 1.25 | [go.dev/dl](https://go.dev/dl/) |
| Bun | 1.1+ | [bun.sh](https://bun.sh) |
| Wails CLI (desktop only) | v3 alpha | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |

### Run the headless server

```bash
git clone https://github.com/libyan-cooperation-org/oblivra.git
cd oblivra
go run ./cmd/server
# → REST + WebSocket on http://localhost:8080
```

### Run the desktop shell (Wails)

```bash
cd frontend && bun install && cd ..
wails3 dev
```

### Send a test event

```bash
curl -X POST http://localhost:8080/api/v1/siem/ingest \
  -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"hostId":"web-01","eventType":"failed_login","message":"Failed password for root from 10.0.0.1"}'
```

### Run the agent against your local server

```bash
go build -o oblivra-agent ./cmd/agent
./oblivra-agent init > /etc/oblivra/agent.yml   # generates a commented sample
$EDITOR /etc/oblivra/agent.yml                  # set server URL + token + inputs
./oblivra-agent run --config /etc/oblivra/agent.yml
```

---

## Architecture (one screen)

```
┌──────────────────────────────────────────────────────────────────────┐
│                         OBLIVRA Server                                │
│                                                                       │
│  Listeners → Ingest pipeline → Hot store (BadgerDB)                   │
│   REST                          + Bleve full-text index               │
│   Syslog UDP/TCP                + WAL (crash safety)                  │
│   Splunk HEC                    + Periodic warm migration (Parquet)   │
│   OTLP/HTTP logs                + Optional cold tier (S3-compat)      │
│   Agent ingest (ed25519)                                              │
│                                                                       │
│  ↓ async fan-out                                                      │
│  Rules · UEBA · Forensics · Lineage · Reconstruction · Trust ·        │
│  Quality · Tamper · Audit                                             │
│                                                                       │
│  ↓                                                                    │
│  Operator UI (Svelte 5 + Tailwind)                                    │
│  Cases → Timeline → Hypotheses → Annotations → Evidence pack          │
└──────────────────────────────────────────────────────────────────────┘
```

The agent is a separate binary. It tails files, optionally extracts named
fields with regex, signs each event with ed25519, batches with adaptive
sizing, spills to encrypted disk under backpressure, and ships to one or
more receivers via gzip-compressed TLS.

---

## Agent

The forwarder is at `cmd/agent/`. It tries to be the "drop-in replacement
for an operator who's used to Splunk Universal Forwarder" — terminal driven,
config file first, restart safe — and adds five things UF doesn't do:

| Feature | UF | OBLIVRA agent |
|---|---|---|
| Per-event signing | TLS-only | ed25519 per-event, key never leaves host |
| Disk spill | plaintext | AES-256-GCM with Argon2id KDF |
| Priority routing | FIFO | local Sigma subset → priority queue |
| Dry-run config validation | partial | `agent test` opens every input + hits `/healthz` |
| Batch sizing | static | adaptive against observed flush time |
| Multi-egress | single | dual-egress fan-out (`secondaryServers:[]`) |

A minimal config:

```yaml
server:
  url: "https://oblivra.internal"
  tokenFile: "/etc/oblivra/agent.token"
  tls:
    caCertFile: "/etc/oblivra/ca.crt"

hostname: "web-01"
tenant: "prod"
batchSize: 100
flushEvery: 2s
compression: gzip
signEvents: true       # ed25519-sign every event
adaptiveBatch: true
localRules: true       # promote critical events to priority queue

inputs:
  - type: file
    path: "/var/log/auth.log"
    sourceType: "linux:auth"
    extract:
      - name: "sshd-fail"
        regex: 'Failed password for (?P<user>\S+) from (?P<srcIP>\S+) port (?P<srcPort>\d+)'
```

Subcommands: `init` · `run` · `test` · `status` · `reload` · `service install` · `version`.

---

## Detection

OBLIVRA loads Sigma rules natively. Drop any `.yml` into the `sigma/`
directory under your data dir; the watcher hot-reloads. Counter-forensic
rules ship under [sigma/counter_forensic/](sigma/counter_forensic) — auditd
flushed, eventlog cleared, timestomp, history purge, Defender disable,
shadow-copy delete, and OBLIVRA self-disable patterns.

For agent-side prioritisation under backpressure, the same critical
patterns live in [predetect.go](cmd/agent/predetect.go) so they bypass
FIFO on their way out the wire.

OQL is available for power users (`/api/v1/siem/oql`); regular search uses
Bleve query strings (`/api/v1/siem/search`).

---

## API surface

Full OpenAPI: [docs/openapi.yaml](docs/openapi.yaml). Highlights:

```
# Ingest
POST /api/v1/siem/ingest                # single event
POST /api/v1/siem/ingest/batch          # batch
POST /api/v1/siem/ingest/raw            # raw log line, auto-parse
POST /services/collector/event          # Splunk HEC (compat — Phase 41)
POST /v1/logs                           # OTLP/HTTP JSON logs (compat — Phase 41)

# Search + reconstruction
GET  /api/v1/siem/search?q=...
GET  /api/v1/siem/oql?q=...
GET  /api/v1/investigations/timeline?host=...
GET  /api/v1/investigations/pivot?host=&at=&delta=
GET  /api/v1/reconstruction/{sessions,state,entities,cmdline,auth}

# Cases
POST /api/v1/cases
GET  /api/v1/cases/{id}/timeline
GET  /api/v1/cases/{id}/confidence
GET  /api/v1/cases/{id}/report.html       # self-contained HTML evidence pack
POST /api/v1/cases/{id}/legal/{submit,approve,reject}
POST /api/v1/cases/{id}/seal

# Audit
GET  /api/v1/audit/log
GET  /api/v1/audit/verify
POST /api/v1/audit/packages/generate

# Operator
GET  /metrics                            # Prometheus exposition + Go-runtime metrics
GET  /debug/pprof/*                      # auth-gated profiling (Phase 47)
GET  /healthz · /readyz
```

Auth: `Authorization: Bearer <token>` or mTLS. Splunk HEC bridges
`Authorization: Splunk <token>` to the same pipeline.

---

## Backup integrity verification

`oblivra-cli backup verify <path>` runs offline against a restored data dir
and confirms:

- the audit Merkle chain replays cleanly entry-by-entry,
- every `*.parquet` file with a `.sha256` sidecar matches its hash,
- the vault file is parseable JSON.

`oblivra-cli backup diff <a> <b>` compares two snapshots — common-prefix
length, divergence point, both root hashes — answering "did one backup
quietly diverge from the other?".

Both subcommands emit JSON; exit code 1 on any failure, so they drop into
CI / backup-validation cron without parsing.

---

## Building from source

```bash
# Headless server
go build -tags production -trimpath -ldflags "-w -s" -o oblivra-server ./cmd/server

# Agent
go build -tags production -trimpath -ldflags "-w -s" -o oblivra-agent ./cmd/agent

# CLI
go build -o oblivra-cli ./cmd/cli

# Verifier (separate binary; ships standalone to air-gapped reviewers)
go build -o oblivra-verify ./cmd/verify

# Smoke harness (43-endpoint go-live gate)
go build -o oblivra-smoke ./cmd/smoke

# Desktop (Wails v3)
wails3 build
```

Docker: `docker compose up -d` brings up the server with a Caddyfile-driven
TLS reverse proxy on port 443. See [docs/operator/deployment.md](docs/operator/deployment.md).

---

## Storage

| Tier | Backing | Retention | Mutability |
|---|---|---|---|
| Hot | BadgerDB v4, key prefix `tenant:{id}:event:{nanoTs}:{evId}` | per-tenant `HotMaxAge` | mutable |
| Warm | Parquet files with v2 schema (carries content hash) | per-tenant `WarmMaxAge` | WORM-locked |
| Cold (optional) | S3-compatible (no SDK; we sign SigV4 directly) | operator policy | WORM |

Per-tenant retention is set via `PUT /api/v1/tenants/policies` and lives in
`tenant_policies.json`. Cross-tier verification recomputes content hashes
from the embedded Parquet column; reachable at `GET /api/v1/storage/verify-warm`.

---

## Self-observability

`/metrics` is the OBLIVRA Prometheus-format scrape target:

- ingest counters: `oblivra_events_total`, `oblivra_events_eps`, `oblivra_hot_events`, `oblivra_wal_*`
- alerts: `oblivra_alerts_total`
- fleet: `oblivra_agents_registered`
- runtime: `oblivra_runtime_sched_latency_p99_seconds`, `oblivra_runtime_gc_pause_p99_seconds`, heap classes, allocs/frees, goroutine count

For deeper investigation, `/debug/pprof/*` is wired in and gated behind the
same auth middleware as every other admin route.

---

## Roadmap & status

The full development log lives in [task.md](task.md). At a glance:

- **Beta-1 hardening pass complete** — durable audit chain, time-frozen
  cases, offline verifier, deployment guide, on-call runbook, security
  review, full-surface smoke harness all marked `[x]`.
- **Phase 40.x agent maturity** — ed25519 signing, encrypted spill, local
  pre-detection, test/service subcommands, adaptive batching, dual-egress.
- **Phase 41 UF compatibility** — Splunk HEC + OTLP/HTTP receivers wired.
- **Phase 44 counter-forensic** — 7-rule Sigma pack + agent-side echoes +
  self-disable / missing-anchor watchdogs.
- **Phase 47 pprof + runtime metrics** — auth-gated profiling and GC /
  scheduler latency on `/metrics`.
- **Phase 49 backup tooling** — `verify` and `diff` subcommands.
- **Phase 50 project hygiene** — Apache 2.0 LICENSE, CONTRIBUTING.md,
  SECURITY.md.

---

## License

Apache License 2.0. See [LICENSE](LICENSE).

## Security

Please do not open public issues for security vulnerabilities. See
[SECURITY.md](SECURITY.md) for the responsible-disclosure process.

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for
development quickstart, accepted change types, and coding standards.

---

<div align="center">
<sub>OBLIVRA — sovereign log-driven security platform · 2026</sub>
</div>
