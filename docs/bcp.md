# OBLIVRA — Business Continuity Plan

> Version 1.0 · 2026-03-01
> Scope: Data protection, disaster recovery, and continuity of security monitoring

---

## 1. Objectives

| Metric | Target | Justification |
|---|---|---|
| **RPO** (Recovery Point Objective) | 1 hour | Maximum acceptable data loss |
| **RTO** (Recovery Time Objective) | 30 minutes | Maximum acceptable downtime |
| **MTTR** (Mean Time to Recover) | < 15 minutes | Typical recovery from prepared backup |

---

## 2. Data Classification

| Data | Volume | Criticality | Backup Frequency | Retention |
|---|---|---|---|---|
| Vault (credentials) | < 10 MB | **Critical** | Every unlock/change | Indefinite |
| Audit logs | ~100 MB/month | **Critical** | Continuous (Merkle-chained) | 6 years (HIPAA) |
| Security events | ~1 GB/month at 1k EPS | **High** | Daily Parquet archive | 365 days hot, 5 years cold |
| SQLite database | ~500 MB | **High** | Daily snapshot | 30 days |
| Bleve search index | ~2 GB | **Medium** | Not backed up (rebuildable) | Rebuilt on demand |
| Detection rules | ~1 MB | **High** | Version-controlled (Git) | Git history |
| Configuration | < 1 MB | **Medium** | Version-controlled | Git history |

---

## 3. Backup Strategy

### 3.1 Automated Backups

| Component | Method | Schedule | Storage |
|---|---|---|---|
| SQLite + Vault | `sqlite3 .backup` → encrypted tar | Every 6 hours | Local + offsite |
| BadgerDB events | Parquet archival pipeline | Daily at 02:00 | Cold storage |
| Merkle tree state | JSON export via API | Daily at 02:00 | With Parquet archives |
| Full data directory | Volume snapshot (LVM/ZFS) | Weekly | Offsite |
| Configuration | Git push to private repo | On every change | Git remote |

### 3.2 Air-Gap Backup (Libya/MENA Deployment)

For environments without reliable internet:

1. **USB Export:** Encrypted snapshot to removable media every 24 hours
2. **Format:** AES-256-GCM encrypted tarball with passphrase
3. **Procedure:**
   ```bash
   oblivra backup --encrypt --output /media/usb/oblivra-$(date +%Y%m%d).enc
   ```
4. **Rotation:** Keep 7 daily + 4 weekly + 12 monthly backups
5. **Verification:** Monthly restore test to isolated machine

### 3.3 Backup Verification

| Test | Frequency | Procedure |
|---|---|---|
| Restore to clean VM | Monthly | Full restore, verify data integrity |
| Merkle tree verification after restore | Monthly | Compare root hash |
| Partial restore (vault only) | Weekly | Decrypt and verify credential count |

---

## 4. Disaster Recovery Scenarios

### 4.1 Hardware Failure (Server Dies)

**Scenario:** Complete server loss — disk, memory, everything.

| Step | Action | Time |
|---|---|---|
| 1 | Provision new server (same OS, Docker installed) | 10 min |
| 2 | Restore encrypted backup from offsite: `oblivra restore --from backup.enc` | 5 min |
| 3 | Verify Merkle integrity: `oblivra verify --merkle` | 1 min |
| 4 | Start OBLIVRA: `docker compose up -d` | 2 min |
| 5 | Verify health + syslog ingestion resumes | 2 min |
| **Total RTO** | | **20 min** |
| **Data loss** | | **≤ 6 hours** (last backup interval) |

### 4.2 Database Corruption

**Scenario:** SQLite or BadgerDB corruption detected.

| Step | Action | Time |
|---|---|---|
| 1 | Stop OBLIVRA gracefully | 1 min |
| 2 | Attempt SQLite integrity check: `sqlite3 oblivra.db "PRAGMA integrity_check"` | 1 min |
| 3a | If repairable: repair in-place, restart | 5 min |
| 3b | If not: restore from last good snapshot | 5 min |
| 4 | Rebuild Bleve index: `oblivra reindex` | 5 min |
| 5 | Verify Merkle tree | 1 min |
| **Total RTO** | | **10–15 min** |

### 4.3 Ransomware on OBLIVRA Host

**Scenario:** Host running OBLIVRA is hit by ransomware.

| Step | Action | Time |
|---|---|---|
| 1 | Isolate host from network immediately | 1 min |
| 2 | Do NOT pay ransom | — |
| 3 | Image the disk for forensic analysis | 30 min |
| 4 | Provision clean server | 10 min |
| 5 | Restore from air-gapped USB backup | 5 min |
| 6 | Rotate all vault credentials (assume compromised) | 30 min |
| 7 | Verify Merkle integrity on restored data | 2 min |
| 8 | Reconnect to network, resume monitoring | 5 min |
| **Total RTO** | | **~1.5 hours** |

### 4.4 Complete Network Isolation (Sanctions / Infrastructure Failure)

**Scenario:** Libya-specific — internet goes down for extended period.

| Impact | Mitigation |
|---|---|
| No threat intel updates | Local threat intel cache (24-hour pre-fetch) |
| No vulnerability database updates | Offline `govulncheck` database |
| No Docker image pulls | Pre-cached images in local registry |
| No GitHub Actions | Pre-built binaries on USB |
| No time sync | Local NTP or GPS clock |
| No notifications | Local alerting dashboard only |

**OBLIVRA continues to function fully** in air-gap mode:
- Syslog ingestion ✅
- Detection rules ✅ (local YAML)
- Correlation engine ✅
- Compliance reports ✅
- Evidence collection ✅
- SSH to managed hosts ✅ (LAN only)

---

## 5. Communication Plan

| Scenario | Notify | Channel | Within |
|---|---|---|---|
| OBLIVRA down > 15 min | Operations team | Secure messenger / phone | 15 min |
| Data integrity breach | Security officer + legal | Phone (no email) | Immediate |
| Full compromise | Executive team | In-person / phone | 1 hour |
| Scheduled maintenance | All analysts | Email / dashboard banner | 24 hours prior |

---

## 6. Annual BCP Review

| Activity | Frequency | Owner |
|---|---|---|
| Full disaster recovery drill | Quarterly | Operations |
| Backup restore verification | Monthly | Operations |
| BCP document review | Annually | Security Officer |
| Contact list update | Quarterly | Operations |
| Air-gap deployment test | Bi-annually | Engineering |
