# OBLIVRA — Production Deployment

This guide is the path from a clean OS install to a hardened production deployment with audit, backup, and verification baked in.

## Choose a topology

| Topology | When to use | Trade-offs |
|---|---|---|
| **Single-node** | <100 hosts, single-tenant, on-prem | No HA. WAL fsync still durable; restart resumes within seconds. |
| **Single-node + standby restore** | <500 hosts, RPO of "last hourly anchor" is acceptable | Failover is manual: copy data dir, start standby, point shippers at new IP. |
| **Multi-node behind a VIP** | High availability, no shared state required | Each node has its own data dir; queries are tenant-pinned. Cross-node aggregation is a roadmap item. |

OBLIVRA does **not** ship a Raft cluster, hot-standby replication, or a multi-master sync layer. These are out of scope for Beta-1.

## Pre-flight checklist

- [ ] Linux host or Windows Server 2022+
- [ ] At least 8 GB RAM (more for >5k EPS sustained)
- [ ] Filesystem with reliable fsync (`ext4`, `xfs`, NTFS — **not** `tmpfs`, **not** an SMB mount)
- [ ] `OBLIVRA_AUDIT_KEY` generated and stored in a vault tool (HashiCorp Vault, AWS Secrets Manager, sealed in your CI)
- [ ] Reverse proxy with TLS 1.3 (Caddy, nginx, or your equivalent)
- [ ] Backup destination reachable from the host

## 1. Build

```bash
git clone https://github.com/libyan-cooperation-org/oblivra.git
cd oblivra
task tools:build
```

Produces under `build/bin/`:
- `oblivra-server` — the platform
- `oblivra-cli` — analyst REST client
- `oblivra-agent` — file-tail forwarder
- `oblivra-verify` — offline integrity checker (ship this to auditors)
- `oblivra-migrate` — schema upgrades
- `oblivra-soak` — load tester
- `oblivra-smoke` — end-to-end endpoint check

## 2. System layout

```
/var/lib/oblivra/                 (OBLIVRA_DATA_DIR)
├── audit.log                     # Merkle audit chain — back up daily
├── cases.log                     # case journal — back up daily
├── lineage.log                   # process lineage journal
├── tenant_policies.json          # per-tenant retention overrides
├── wal/ingest.wal                # write-ahead log
├── siem_hot.badger/              # hot store (BadgerDB)
├── bleve.idx/                    # full-text indices (per tenant)
├── warm.parquet/                 # warm tier (WORM-locked)
└── oblivra.vault                 # secret store (if vault initialised)
```

## 3. systemd unit (Linux)

```ini
# /etc/systemd/system/oblivra.service
[Unit]
Description=OBLIVRA — sovereign log platform
After=network.target

[Service]
Type=simple
User=oblivra
Group=oblivra
WorkingDirectory=/opt/oblivra
EnvironmentFile=/etc/oblivra/oblivra.env
ExecStart=/opt/oblivra/oblivra-server
Restart=always
RestartSec=5
LimitNOFILE=65536
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
NoNewPrivileges=true
ReadWritePaths=/var/lib/oblivra

[Install]
WantedBy=multi-user.target
```

`/etc/oblivra/oblivra.env` (mode 0600, owned by `oblivra`):
```env
OBLIVRA_DATA_DIR=/var/lib/oblivra
OBLIVRA_ADDR=127.0.0.1:8080
OBLIVRA_API_KEYS=...:admin,...:analyst
OBLIVRA_AUDIT_KEY=...
OBLIVRA_SYSLOG_ADDR=:1514
OBLIVRA_NETFLOW_ADDR=:2055
```

The TLS-terminating reverse proxy fronts `127.0.0.1:8080` — the platform itself never speaks plaintext to anything except localhost.

## 4. Reverse proxy (Caddy example)

```caddy
oblivra.internal {
  tls /etc/ssl/certs/oblivra.crt /etc/ssl/private/oblivra.key
  encode gzip
  reverse_proxy 127.0.0.1:8080
  request_body {
    max_size 256MB
  }
  log {
    output file /var/log/caddy/oblivra.log
  }
}
```

## 5. First boot

```bash
sudo systemctl enable --now oblivra
journalctl -u oblivra -f
```

Expected log lines on a clean start:
```
audit journal opened path=/var/lib/oblivra/audit.log entries=0
investigations journal opened path=/var/lib/oblivra/cases.log cases=0
scheduler started jobs=3
sigma watcher started dir=sigma
platform ready dataDir=/var/lib/oblivra
oblivra-server listening addr=127.0.0.1:8080 syslog=:1514
```

Sanity-check from another shell:
```bash
oblivra-smoke --server https://oblivra.internal --token $ADMIN_KEY
```

Every line should print `✓`. If anything prints `✗`, fail the deployment.

## 6. Backups

### What to back up
- `/var/lib/oblivra/audit.log` (append-only — incremental rsync is fine)
- `/var/lib/oblivra/cases.log`
- `/var/lib/oblivra/lineage.log`
- `/var/lib/oblivra/tenant_policies.json`
- `/var/lib/oblivra/oblivra.vault`
- The newest 7 days of `/var/lib/oblivra/warm.parquet/*.parquet` (older files are already WORM and should already be on cold storage)

### Backup verification
After every backup window:
```bash
oblivra-verify --hmac "$OBLIVRA_AUDIT_KEY" /backups/oblivra/audit.log
```
The exit code must be 0 and the printed root hash must match what the live platform reports at `GET /api/v1/audit/verify`.

### What **not** to back up

- `siem_hot.badger/` — derived from WAL replay; restoring to a non-empty Badger dir corrupts state. Restore by replaying the WAL on a fresh node.
- `bleve.idx/` — also derived; rebuilds on first query.

## 7. Soak validation before go-live

```bash
# Terminal A — server
sudo systemctl start oblivra

# Terminal B — soak
oblivra-soak --server https://oblivra.internal --token $LOAD_KEY \
  --eps 5000 --duration 5m --hosts 200
```

Expected output should contain:
- `ok: ≥ 99.5%` of sent events
- `latency p99: < 200ms` on a 4-core VM
- `failed: 0` for the first run

Capture the report and archive under `docs/operator/soak-results-<YYYY-MM-DD>.md` so the next operator has a baseline.

## 8. Routine operations

| Task | Command | Cadence |
|---|---|---|
| Verify chain integrity | `oblivra-cli audit verify` | Daily |
| Check for tamper signals | `curl /api/v1/forensics/tamper` | Daily |
| Promote hot → warm | Auto (every 6h) — manual: `curl -X POST /api/v1/storage/promote` | n/a |
| Reload Sigma rules | Auto (fsnotify on `sigma/`) | n/a |
| Rotate API keys | Edit env file → `systemctl restart oblivra` | Quarterly |
| Anchor previous day | Auto (every hour) | n/a |
| Cross-tier verify | `curl /api/v1/storage/verify-warm` | Weekly |

## 9. Upgrades

```bash
# 1. Pull the new release
cd /opt/oblivra
sudo systemctl stop oblivra
git pull
task tools:build

# 2. Apply schema migrations (no-op at v1, framework ready for v2+)
sudo -u oblivra ./build/bin/oblivra-migrate run --all /var/lib/oblivra

# 3. Confirm the audit chain still verifies post-migration
sudo -u oblivra ./build/bin/oblivra-verify --hmac "$OBLIVRA_AUDIT_KEY" /var/lib/oblivra/audit.log

# 4. Restart
sudo systemctl start oblivra
```

If migration produces `.pre-migrate` files, keep them for at least one retention window.

## 10. When something goes wrong

- **`audit chain broken!` in logs** → Stop ingest, run `oblivra-verify` against the on-disk audit.log, isolate the host filesystem, and restore from the most recent verified backup.
- **`cases.log` won't replay** → A torn write at the last entry is recoverable: truncate the last line and restart. A mid-line corruption mid-file means you need to restore that file from backup.
- **WAL grows unbounded** → The warm-tier migrator should be running every 6h. Check the scheduler logs (`scheduled job ok`/`scheduled job failed`); the warm migrator's read-failure log line names the offending event.
- **Soak fails post-upgrade** → Roll back: `git checkout <previous-tag>`, rebuild, restart. Then use the migration framework's `.pre-migrate` files to confirm what changed.

## 11. Decommission

```bash
sudo systemctl stop oblivra
# 1. Final audit-chain anchor + export so future audits are conclusive
sudo -u oblivra ./build/bin/oblivra-cli audit log --limit 100000 > /backups/oblivra-final.log
sudo -u oblivra ./build/bin/oblivra-verify --hmac "$OBLIVRA_AUDIT_KEY" /var/lib/oblivra/audit.log
# 2. Crypto-wipe the vault
shred -u -z -n 7 /var/lib/oblivra/oblivra.vault
# 3. Move the rest to cold storage
tar czf /backups/oblivra-decommissioned-$(date +%F).tar.gz /var/lib/oblivra
shred -u /var/lib/oblivra/wal/*.wal
rm -rf /var/lib/oblivra
```
