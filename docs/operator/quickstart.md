# Operator Quick-Start Guide

Get Oblivra Sovereign Terminal running and ingesting real data in under 15 minutes.

---

## Prerequisites

| Requirement | Version | Notes |
|---|---|---|
| Go | ≥ 1.25 | `go version` |
| Wails CLI | ≥ 2.11 | `wails version` |
| Bun | latest | Frontend build |
| WebView2 | any | Windows only — usually pre-installed |

```powershell
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

---

## 1. Build

```powershell
# Windows
cd sovereign-terminal
go mod tidy
wails build
# → build/bin/oblivrashell.exe
```

```bash
# macOS / Linux
wails build
# → build/bin/oblivrashell
```

For development with hot-reload:

```bash
wails dev
```

---

## 2. First Launch

1. Run the binary. The Vault Gate appears — the login screen.
2. Click **Set up new vault** (first run only).
3. Choose a strong master passphrase. This encrypts all credentials and the database. **There is no recovery if you forget it.**
4. On subsequent launches, enter your passphrase to unlock.

The vault uses AES-256-GCM with Argon2id key derivation. The passphrase never leaves the device.

---

## 3. Data Locations

| Platform | Path |
|---|---|
| Windows | `%LOCALAPPDATA%\sovereign-terminal\` |
| macOS | `~/Library/Application Support/sovereign-terminal/` |
| Linux | `~/.local/share/sovereign-terminal/` |

Key files:

```
sovereign-terminal/
├── data/
│   ├── siem_hot.badger/    # BadgerDB hot SIEM store
│   ├── wal/ingest.wal      # Write-ahead log (crash recovery)
│   ├── bleve.idx/          # Full-text search index
│   ├── analytics.db        # SQLite analytics + alert history
│   ├── sigma/              # Drop Sigma rules here for hot-reload
│   └── rules/              # Native YAML detection rules
├── oblivra.vault           # Encrypted vault (AES-256-GCM)
└── oblivra.log             # Application log
```

---

## 4. Adding Your First Host

1. Click **Hosts** in the left navigation.
2. Click **+ New Host** (top right of sidebar, or `Cmd+N`).
3. Fill in: hostname/IP, port (default 22), username.
4. Choose authentication: password or SSH key (stored in vault).
5. Click **Add**. The host appears in the sidebar.
6. Click the host to open an SSH session.

---

## 5. Starting Log Ingestion

### Syslog (UDP/TCP port 1514)

Point your devices, firewalls, or log shippers at:

```
<your-machine-ip>:1514
```

Supported formats: RFC5424, RFC3164, CEF, raw lines.

Go to **Ops Center → Sources** and click **Start Syslog Server** to enable the listener.

### Agent-based (port 8443)

For Linux endpoints with the eBPF agent:

```bash
# On the endpoint
./oblivra-agent --server <your-ip>:8443 --token <api-token>
```

The agent provides: process telemetry, network connections, file events, login events.

### Manual event injection (API)

```bash
curl -X POST http://localhost:8080/api/v1/siem/ingest \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"host_id":"web-01","event_type":"failed_login","raw_log":"Failed password for root from 10.0.0.1"}'
```

---

## 6. Verifying Detection Rules Are Active

1. Go to **Ops Center → Dashboard**.
2. Check the **Active Rules** KPI tile — should show ≥50 on first launch.
3. Go to **MITRE Heatmap** to see tactic coverage across your loaded rules.

To verify a specific rule fires:

```bash
# Send a test SSH brute-force event via syslog
echo '<34>1 2026-01-01T00:00:00Z test-host sshd - - Failed password for root from 10.0.0.1 port 22 ssh2' | \
  nc -u 127.0.0.1 1514
```

Within a few seconds an alert should appear in **SIEM → Alerts**.

---

## 7. Configuring Alert Notifications

Go to **Ops Center → Alerts → Notification Channels**:

| Channel | What you need |
|---|---|
| Email (SMTP) | SMTP host, port, username, password |
| Telegram | Bot token (from @BotFather), Chat ID |
| SMS/WhatsApp | Twilio Account SID, Auth Token |
| Webhook | URL (Slack / Discord / Teams / any) |

Click **Test Connection** to verify before saving.

---

## 8. Observability Stack (Optional)

```bash
docker-compose up -d
```

- **Prometheus** → `http://localhost:9090`
- **Grafana** → `http://localhost:3000` (admin / oblivra)
- **Grafana Tempo** → `http://localhost:3200`

The pre-built Oblivra dashboard shows: goroutines, heap, ingest EPS, detection rate, SSH latency.

---

## 9. Platform Health Check

The **status bar** (bottom of window) shows a health grade (`● A` through `● F`).

- Click it to open the **Diagnostics Modal** — live goroutine count, heap usage, ingest throughput, query latency.
- Go to **Health** in the navigation for full service status.

---

## 10. Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Cmd/Ctrl + K` | Command palette |
| `Cmd/Ctrl + N` | Add new host |
| `Cmd/Ctrl + B` | Toggle sidebar |
| `Cmd/Ctrl + ,` | Settings |
| `Cmd/Ctrl + Shift + F` | Focus mode (hide all chrome) |
| `Ctrl+Click` nav item | Open as floating panel |

---

## Next Steps

- [Detection Authoring Guide](detection-authoring.md) — write custom YAML detection rules
- [Sigma Rules Guide](sigma-rules.md) — import community Sigma rules
- [Alerting Config](alerting-config.md) — configure multi-channel notifications
- [API Reference](api-reference.md) — integrate with external tools
