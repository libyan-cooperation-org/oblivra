# OBLIVRA — Operator Quickstart

This guide gets a fresh OBLIVRA install running, ingests its first events,
verifies the audit chain, and confirms the search and detection paths.

## Requirements

- Go 1.25+
- Node.js 20+ (for the frontend build)
- Wails v3 alpha CLI (only for the desktop shell): `go install github.com/wailsapp/wails/v3/cmd/wails@latest`
- A Linux/macOS box, Windows 11, or Windows Server 2022

## 1. Build

```bash
git clone https://github.com/libyan-cooperation-org/oblivra.git
cd oblivra
go mod tidy
cd frontend && npm install && npm run build && cd ..

# Headless server
go build -trimpath -ldflags "-w -s" -o oblivra-server ./cmd/server

# CLI
go build -o oblivra-cli ./cmd/cli

# Desktop shell (optional)
wails3 build
```

## 2. Run

The headless server serves the same Svelte UI at `http://localhost:8080`,
plus the REST API.

```bash
./oblivra-server
```

You should see:

```
level=INFO msg="platform ready" dataDir=...
level=INFO msg="oblivra-server listening" addr=:8080 syslog=:1514
```

Open <http://localhost:8080> and you'll see the **Overview** dashboard
populating live ingest stats.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `OBLIVRA_ADDR` | `:8080` | HTTP listen address |
| `OBLIVRA_SYSLOG_ADDR` | `:1514` | Syslog UDP listener |
| `OBLIVRA_NETFLOW_ADDR` | `:2055` | NetFlow v5 UDP listener |
| `OBLIVRA_DISABLE_SYSLOG` | — | Set to disable the syslog listener |
| `OBLIVRA_DISABLE_NETFLOW` | — | Set to disable the NetFlow listener |
| `OBLIVRA_DATA_DIR` | OS default | Override on-disk data dir |
| `OBLIVRA_API_KEYS` | — | Comma-separated keys with optional `:role` suffix |
| `OBLIVRA_AUDIT_KEY` | random | HMAC key used to sign audit-log entries |

### Default data layout (Windows)

```
%LOCALAPPDATA%\oblivra\
├── wal\ingest.wal              # write-ahead log (line-delimited JSON)
├── siem_hot.badger\            # hot store (BadgerDB v4)
├── bleve.idx\                  # full-text search indices (per-tenant)
└── warm.parquet\               # warm tier (Parquet files)
```

## 3. Ingest a first event

Without auth (default):

```bash
curl -X POST http://localhost:8080/api/v1/siem/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "source": "rest",
    "hostId": "web-01",
    "severity": "warning",
    "message": "sshd Failed password for root from 10.0.0.42"
  }'
```

This event triggers the **builtin-ssh-bruteforce** detection rule and raises
an alert tagged with MITRE T1110.001.

### Syslog

Point any RFC 5424 / RFC 3164 emitter at port 1514 UDP:

```bash
echo '<34>1 2026-04-30T12:00:00Z dc-01 sshd 1234 ID47 - Failed password for root' | nc -u -w1 127.0.0.1 1514
```

### NetFlow v5

Point any NetFlow v5 exporter at port 2055 UDP. Records appear under
`/api/v1/ndr/flows` and aggregated as top-talkers under
`/api/v1/ndr/top-talkers`.

### Raw line ingestion

```bash
curl -X POST "http://localhost:8080/api/v1/siem/ingest/raw?format=auto" \
  --data-binary @/var/log/auth.log
```

## 4. Search

### Bleve query syntax via REST

```bash
curl -s "http://localhost:8080/api/v1/siem/search?q=severity:warning&limit=20&newestFirst=true"
```

### CLI

```bash
./oblivra-cli search --q "message:sshd" --limit 25
```

## 5. Verify the audit chain

Every administrative action and evidence-seal lands in a Merkle-chained,
HMAC-signed audit log. To confirm integrity:

```bash
./oblivra-cli audit verify
# {"ok": true, "entries": 42, "rootHash": "ca2eb..."}

./oblivra-cli audit log --limit 5
```

## 6. Seal evidence

Bundle a host's recent events into a sealed evidence package
(SHA-256 + signed audit chain entry):

```bash
curl -X POST http://localhost:8080/api/v1/forensics/evidence \
  -H "Content-Type: application/json" \
  -d '{"hostId":"dc-01","title":"Suspicious LSASS access"}'
```

Each package is hash-pinned and recorded in the audit log.

## 7. Migrate hot → warm tier

Events older than 30 days (default) can be promoted into Parquet files in the
warm tier. Run on demand:

```bash
curl -X POST http://localhost:8080/api/v1/storage/promote
# {"moved": 12345}
```

## 8. Auth (production)

Set comma-separated keys with role assignments:

```bash
OBLIVRA_API_KEYS="ops-key:admin,sec-key:analyst,monitor:readonly,agent01:agent" ./oblivra-server
```

Roles enforce the permission catalogue in `internal/rbac`:

| Role | Reads SIEM | Ingests | Reads alerts | Writes rules | Audit export |
|---|:-:|:-:|:-:|:-:|:-:|
| admin | ✓ | ✓ | ✓ | ✓ | ✓ |
| analyst | ✓ | ✓ | ✓ | — | — |
| readonly | ✓ | — | ✓ | — | — |
| agent | — | ✓ | — | — | — |

CLI clients pass the key via `OBLIVRA_TOKEN`:

```bash
export OBLIVRA_TOKEN=ops-key
./oblivra-cli stats
```

## 9. Live tail

The browser UI's SIEM tab connects to `/api/v1/events` over WebSocket and
shows new events in real time. Any client can subscribe:

```bash
websocat ws://localhost:8080/api/v1/events
```

Each frame is `{"type":"event","event":{...}}`.

## Troubleshooting

| Symptom | Fix |
|---|---|
| "address already in use" on :1514 | Port-bind requires root on Linux for ports <1024 — use `sudo setcap cap_net_bind_service=+ep oblivra-server` or set `OBLIVRA_SYSLOG_ADDR=:5514`. |
| WAL grows unbounded | WAL is append-only by design — the warm-tier migrator does not yet truncate. Phase-22.4 will close that loop. |
| Bleve index missing after restart | Each tenant index lives at `bleve.idx/<tenant>.bleve` — they are lazily re-opened on first query. |
| Audit verify returns `ok:false` | Chain is broken (someone modified the in-memory or persisted entries). The chain is intentionally tamper-evident — restore from a backup if so. |

## Next steps

- Add detection rules: `internal/services/rules_service.go` for the builtin
  list; the engine accepts the same fields/AnyContain/AllContain shape from
  user-supplied YAML once the loader lands.
- Connect an agent: see `cmd/agent` (TBD) — for now any caller with the
  `agent` role can POST to `/api/v1/agent/ingest`.
- Wire your existing observability stack at `/metrics` once the Prometheus
  exporter ships in Phase 22.5.
