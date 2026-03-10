# OBLIVRA — Operational Runbook

> Version 1.0 · 2026-03-01
> Purpose: Incident response and operational procedures for OBLIVRA itself

---

## 1. Severity Levels

| Level | Description | Response Time | Escalation |
|---|---|---|---|
| **SEV-1** | OBLIVRA is down or data integrity compromised | 15 min | Immediate to engineering lead |
| **SEV-2** | Partial service degradation (ingestion backlog, slow search) | 1 hour | Engineering on-call |
| **SEV-3** | Non-critical issue (UI bug, cosmetic, single-source failure) | 24 hours | Next sprint |

---

## 2. Health Check Procedures

### 2.1 Quick Health Check (Daily)

```bash
# Check process is running
curl -k https://localhost:8443/api/health

# Check syslog is accepting connections
nc -z localhost 1514 && echo "Syslog OK" || echo "Syslog FAIL"

# Check Prometheus metrics
curl -s http://localhost:9090/metrics | head -5

# Check disk usage
df -h /var/lib/oblivra/
```

### 2.2 Deep Health Check (Weekly)

```bash
# Verify Merkle tree integrity
curl -k https://localhost:8443/api/integrity/verify

# Check audit log count growth (should be non-zero)
curl -k https://localhost:8443/api/audit/count

# Check BadgerDB LSM tree size
du -sh /var/lib/oblivra/badger/

# Check goroutine count (should be < 500 under normal load)
curl -s http://localhost:9090/metrics | grep go_goroutines

# Run compliance self-evaluation
curl -k -X POST https://localhost:8443/api/compliance/evaluate
```

---

## 3. Incident Procedures

### 3.1 OBLIVRA Process Crash (SEV-1)

**Symptoms:** API unreachable, syslog connection refused, frontend fails to load.

**Steps:**
1. Check process status: `systemctl status oblivra` or `docker ps`
2. Review last 100 lines of log: `journalctl -u oblivra -n 100` or `docker logs oblivra --tail 100`
3. Check for OOM kill: `dmesg | grep -i "out of memory"`
4. If OOM: increase memory limit in `docker-compose.yml` (`mem_limit`)
5. If panic: collect goroutine dump from logs, file bug
6. Restart: `systemctl restart oblivra` or `docker compose restart`
7. Verify recovery: `curl -k https://localhost:8443/api/health`
8. Log incident in audit: manual entry via CLI

**Post-Incident:**
- [ ] Root cause analysis within 48 hours
- [ ] Update memory limits if OOM
- [ ] Add goroutine watchdog alert if not yet enabled

### 3.2 Merkle Tree Integrity Failure (SEV-1)

**Symptoms:** Compliance check returns `merkle_integrity_valid: false`. Audit logs may have been tampered with.

**Steps:**
1. **DO NOT RESTART** — preserve process memory for forensic analysis
2. Immediately seal the evidence locker: `curl -k -X POST https://localhost:8443/api/forensics/seal-all`
3. Export current Merkle tree state: `curl -k https://localhost:8443/api/integrity/export > merkle_state.json`
4. Export audit logs: `curl -k https://localhost:8443/api/audit/export > audit_export.json`
5. Compare leaf hashes against exported audit entries
6. Identify the first divergent leaf — this is the tampering point
7. Check system access logs for unauthorized SSH/physical access around that timestamp
8. Contact security team — potential forensic investigation required

**Post-Incident:**
- [ ] Determine attack vector (insider, compromise, corruption)
- [ ] Re-initialize Merkle tree from verified audit data
- [ ] Rotate vault master key if compromise suspected
- [ ] File incident report

### 3.3 Syslog Ingestion Backlog (SEV-2)

**Symptoms:** Events arrive late in dashboards. Queue depth metric rising.

**Steps:**
1. Check EPS rate: `curl -s http://localhost:9090/metrics | grep siem_events_per_second`
2. Check queue depth: `curl -s http://localhost:9090/metrics | grep siem_queue_depth`
3. If EPS > capacity: identify top senders, consider rate limiting at source
4. If parser-bound: check for complex regex rules causing slowdown
5. Temporary relief: increase worker pool in config (`OBLIVRA_PARSER_WORKERS`)
6. If disk I/O bound: check BadgerDB compaction status, manually trigger GC

### 3.4 Vault Compromise Suspicion (SEV-1)

**Symptoms:** Unauthorized credential access detected in audit log, or physical breach suspected.

**Steps:**
1. **Immediately lock the vault:** close application or use kill-switch
2. Export audit logs for the suspected time window
3. Rotate vault master passphrase on a clean machine
4. Rotate ALL stored credentials (SSH keys, passwords, API keys)
5. If using OS Keychain: clear and re-initialize keychain entry
6. Review all SSH session recordings during the suspected window
7. Notify affected managed host administrators

### 3.5 Database Corruption (SEV-2)

**Symptoms:** SQL errors in logs, BadgerDB checksum failures, search returning incomplete results.

**Steps:**
1. Stop ingestion to prevent further writes
2. For SQLite: run `PRAGMA integrity_check;`
3. For BadgerDB: check logs for `checksum mismatch` errors
4. If repairable: BadgerDB has built-in recovery (`--truncate` on restart)
5. If not: restore from latest encrypted snapshot
6. Rebuild Bleve search index from BadgerDB events
7. Verify Merkle tree integrity after recovery

---

## 4. Maintenance Procedures

### 4.1 Scheduled Maintenance Window

1. Announce maintenance window (email/Slack)
2. Graceful shutdown: `kill -SIGTERM <pid>` (waits for in-flight operations)
3. Backup: `cp -r /var/lib/oblivra /backup/oblivra-$(date +%Y%m%d)`
4. Perform updates (binary swap, config changes, migration)
5. Start service: `systemctl start oblivra`
6. Verify health check (Section 2.1)
7. Announce maintenance complete

### 4.2 Vault Key Rotation

1. Unlock vault with current passphrase
2. Export all credentials to encrypted archive
3. Generate new passphrase (min 20 characters, high entropy)
4. Re-initialize vault with new passphrase
5. Import credentials
6. Update OS Keychain entry
7. Distribute new passphrase via secure channel
8. Audit: verify rotation logged

### 4.3 Log Archival

1. Identify retention window (default: 90 days hot, 365 days archive)
2. Export logs older than hot window to Parquet: `oblivra archive --before 90d`
3. Move Parquet files to cold storage (NAS, airgapped USB)
4. Purge archived data from BadgerDB: `oblivra purge --before 90d`
5. Trigger BadgerDB value log GC
6. Verify disk space recovery

---

## 5. Emergency Contacts

| Role | Contact | Escalation Path |
|---|---|---|
| Engineering Lead | [To be filled] | SEV-1/SEV-2 |
| Security Officer | [To be filled] | Vault compromise, Merkle failure |
| Operations | [To be filled] | Infrastructure issues |
| Legal | [To be filled] | Evidence handling, compliance breach |

---

## 6. Incident Report Template

```markdown
## Incident Report: [Title]
- **Date:** YYYY-MM-DD HH:MM
- **Severity:** SEV-1 / SEV-2 / SEV-3
- **Duration:** X hours X minutes
- **Impact:** [What was affected]
- **Root Cause:** [What went wrong]
- **Detection:** [How it was discovered]
- **Resolution:** [Steps taken to fix]
- **Follow-Up Actions:**
  - [ ] Action 1
  - [ ] Action 2
- **Lessons Learned:** [What to improve]
```
