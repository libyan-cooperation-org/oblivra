# OBLIVRA — On-call Runbook

Playbooks for the alerts a production OBLIVRA deployment can wake an
operator with. Each playbook follows the same shape:

> **Symptom** → **What it means** → **Verify** → **Mitigate** → **Resolve**

## Table of contents

1. [Audit chain reports broken](#audit-chain-reports-broken)
2. [Tamper signal raised](#tamper-signal-raised)
3. [Cases.log won't replay on startup](#caseslog-wont-replay-on-startup)
4. [WAL grows without bound](#wal-grows-without-bound)
5. [Sustained EPS dropped below SLA](#sustained-eps-dropped-below-sla)
6. [/healthz returns 5xx](#healthz-returns-5xx)
7. [/metrics scrape fails](#metrics-scrape-fails)
8. [Vault unlock fails on a known-good passphrase](#vault-unlock-fails-on-a-known-good-passphrase)
9. [Webhook delivery failing repeatedly](#webhook-delivery-failing-repeatedly)
10. [Disk usage warning](#disk-usage-warning)

---

## Audit chain reports broken

**Symptom:** Server logs include `audit chain broken!` (logged every 5 min by the scheduler), or the UI Evidence view shows red "broken at #N".

**What it means:** Either (a) the on-disk `audit.log` was modified outside the platform, or (b) a hardware failure produced silent corruption. The platform refuses to extend a broken chain — every subsequent `Append` will succeed but `Verify` will keep returning `ok:false`.

**Verify:**
```bash
oblivra-verify --hmac "$OBLIVRA_AUDIT_KEY" /var/lib/oblivra/audit.log
```
The output prints `BROKEN at entry N: <reason>`. Note that N.

**Mitigate (immediate):**
1. **Stop ingest from authoritative sources** — pause syslog/agent forwarders so further writes don't pollute the chain.
2. **Snapshot the data directory** for forensic analysis: `tar czf /backups/oblivra-broken-$(date +%s).tar.gz /var/lib/oblivra`.
3. Notify the security lead — chain breakage is a security event, not a bug.

**Resolve:**
- If entry N looks like a torn write at the end of the file (last entry only, no signed_at), truncate to N-1 and continue: `head -n $((N-1)) audit.log > audit.log.tmp && mv audit.log.tmp audit.log`.
- If N is mid-file, restore from the most-recent verified backup. The deployment guide mandates daily verified backups for exactly this case.
- After restore, re-run `oblivra-verify` on the restored file before unfreezing ingest.

**Don't:** delete the broken file. The chain itself is evidence.

---

## Tamper signal raised

**Symptom:** Alert with `ruleId: tamper-*` (auditd-disabled, journal-truncate, eventlog-clear, clock-rollback), or the **Trust & Quality** view's "Tamper findings" section is non-empty.

**What it means:** A log line on a monitored host matched a known anti-forensic pattern. This may be a real attacker action, a sysadmin doing routine maintenance, or a noisy log source.

**Verify:**
```bash
curl /api/v1/forensics/tamper | jq '.[] | select(.hostId == "<host>")'
```
Cross-reference with the host's recent activity:
```bash
curl '/api/v1/investigations/pivot?host=<host>&at=<unix-ts>&delta=900' | jq
```

**Mitigate:**
1. Open a case scoped to the host (`POST /api/v1/cases`) — this freezes the audit root *now* so the investigation snapshot is reproducible.
2. Add a hypothesis: "tamper signal X on host Y at time T is attacker activity" (or "sysadmin").
3. Annotate the matching event with whatever context you find.

**Resolve:**
- Confirmed maintenance → mark hypothesis `refuted`, seal the case.
- Confirmed attacker action → escalate to your IR process; the platform is *not* a SOAR — OBLIVRA produces evidence, your IR tool does isolation.

---

## Cases.log won't replay on startup

**Symptom:** Server logs `error: investigations: cases replay: <error>` and refuses to start.

**What it means:** A torn write or filesystem corruption left the cases journal in an unparseable state.

**Verify:** look at the last few lines of `/var/lib/oblivra/cases.log`. If the last line is truncated mid-JSON, that's a recoverable torn write.

**Mitigate:**
- For a torn last line: `head -n $(($(wc -l < cases.log) - 1)) cases.log > cases.log.tmp && mv cases.log.tmp cases.log`.
- For mid-file corruption: restore the file from backup. The cases journal has the same daily-backup requirement as `audit.log`.

**Resolve:** Restart `systemctl restart oblivra`. The cases-journal entries are append-only so restoring a slightly-older backup means a few recent case mutations are lost, but the structure is intact.

---

## WAL grows without bound

**Symptom:** `/var/lib/oblivra/wal/ingest.wal` is hundreds of MB or larger; `du` shows it's the dominant storage consumer.

**What it means:** The warm-tier migrator should be running every 6 hours and evicting events that were already promoted. If the WAL is growing it means migrations aren't completing, or the eviction step is failing.

**Verify:**
```bash
curl /api/v1/storage/stats | jq
```
Look at `lastRunAt` — if it's >12h old the scheduler isn't firing.

```bash
journalctl -u oblivra --since '6h ago' | grep -E 'tiering|warm'
```
Errors like "warm written but hot delete failed" are the smoking gun.

**Mitigate:**
- Manual promotion: `curl -X POST /api/v1/storage/promote`. Watch the response — `moved: N` for how many events shifted.
- If `Verify` returns OK but `lastRunAt` is stale, the scheduler goroutine likely died. Restart the platform: `systemctl restart oblivra`.

**Resolve:** Confirm the scheduler is running again (`scheduled job ok name=tiering.warm-migrate` in logs every 6h) and the WAL stops growing. Phase-22.4 will close the WAL-truncation loop; until then, manual rotation under sustained 10k+ EPS is operationally expected.

---

## Sustained EPS dropped below SLA

**Symptom:** `oblivra_events_eps` Prometheus metric is below the line documented in your `soak-results-<date>.md`, or `Pipeline.Stats().Latency.Total.P95` is >2× the baseline.

**What it means:** Backpressure is coming from one of: WAL fsync (disk slowed down), Bleve index commit (index growing too large), or BadgerDB compaction.

**Verify:**
```bash
curl /api/v1/siem/stats | jq '.latency'
```
Compare each stage's p95 against the baseline. The slowest stage is the bottleneck.

**Mitigate:**
- WAL p95 ↑ → check disk health (`smartctl`, `iostat`). Move data dir to faster storage if NVMe → SATA regression has happened.
- Index p95 ↑ → run a manual warm-tier promotion (offloads hot events, shrinks the working index).
- Hot p95 ↑ → BadgerDB GC may be queued; restart the platform to force a compaction.

**Resolve:** Re-run `oblivra-soak --duration 30s` against the live server; confirm the new latency is back inside SLA bounds.

---

## /healthz returns 5xx

**Symptom:** The reverse-proxy health check (or external uptime monitor) reports HTTP 5xx from `/healthz`.

**What it means:** The HTTP server is reachable but the platform's bootstrap failed. The endpoint is the loosest possible — if it's failing, things are very wrong.

**Verify:** Check the systemd journal: `journalctl -u oblivra -n 100`. Likely culprits: `audit journal: ...` (chain replay failed), `open hot store: ...` (Badger lock held), `bootstrap failed: ...`.

**Mitigate:**
1. If audit replay failure → see [Audit chain reports broken](#audit-chain-reports-broken).
2. If hot-store lock → kill the lock holder: `lsof | grep siem_hot.badger | head` (Linux); on Windows: `Get-Process | Where-Object MainModule -like '*oblivra*' | Stop-Process`.
3. If neither → restart: `systemctl restart oblivra`.

**Resolve:** `oblivra-cli ping` returns `{"status":"ok"}`.

---

## /metrics scrape fails

**Symptom:** Prometheus scrape errors for the OBLIVRA target.

**What it means:** Either the platform isn't running, or auth is enabled but the scraper isn't sending a token.

**Verify:** `/metrics` is **auth-exempt** — if the scrape fails with 401/403, double-check the proxy isn't injecting auth requirements.

**Resolve:**
- Caddyfile / nginx: confirm `/metrics` is *not* in any path-protected block.
- If you must require auth on metrics, point Prometheus at `/metrics?token=<api-key>` — the auth middleware honors `?token=`.

---

## Vault unlock fails on a known-good passphrase

**Symptom:** `POST /api/v1/vault/unlock` returns `{"error":"vault: invalid key"}` even though the passphrase is correct.

**What it means:** Either (a) the passphrase has whitespace / encoding drift, (b) the vault file was overwritten, or (c) Argon2 KDF parameters drifted.

**Verify:**
```bash
file /var/lib/oblivra/oblivra.vault
jq '.kdfParams' /var/lib/oblivra/oblivra.vault
```
The `kdfParams` block should match the `DefaultKDFParams()` shipped with the running binary version.

**Mitigate:**
- Check `passphrase` for trailing whitespace (a CR slipping into a Helm secret is the classic cause).
- Confirm the vault file hasn't been replaced by a backup from a *different* OBLIVRA install (it would have a different salt).

**Resolve:** If the passphrase is genuinely lost, **there is no recovery** — by design. Restore from a backup taken when the passphrase was known, or initialise a new vault and rotate every secret it contained.

---

## Webhook delivery failing repeatedly

**Symptom:** `/api/v1/webhooks/deliveries` shows status 4xx/5xx for every alert; Slack channel quiet.

**What it means:** The receiver is rejecting the body shape, the HMAC, or its own auth.

**Verify:**
```bash
curl /api/v1/webhooks/deliveries | jq '.[] | {webhookId, status, error}'
```
Look at the error column. Common shapes:
- `HTTP 401` → receiver expects an auth header we don't set; fix at the receiver
- `HTTP 400` → body shape mismatch; OBLIVRA's body is a flat JSON object documented in security-review.md
- `connection refused` → receiver is down

**Mitigate:** Disable the webhook (`PUT /api/v1/webhooks/{id}` with `disabled:true`) until you can fix the receiver — undelivered alerts still land in the audit chain via the alert-raised entry, so nothing is lost.

**Resolve:** Re-enable the hook; the next alert's delivery succeeds (visible in the deliveries list with `status:200`).

---

## Disk usage warning

**Symptom:** `/var/lib/oblivra` filling toward the partition's high-water mark.

**What it means:** Some combination of: WAL not rotating, warm tier accumulating, Bleve indices growing, audit chain naturally growing.

**Verify:**
```bash
du -sh /var/lib/oblivra/* | sort -h
```
Compare against the soak baseline + your retention policy.

**Mitigate:**
- WAL: see [WAL grows without bound](#wal-grows-without-bound).
- Warm tier: confirm cold-tier migration is configured (or that ageing files are being archived elsewhere).
- Bleve: indices grow forever today (Phase 22.x will add per-tenant index rotation). Workaround: set per-tenant `WarmMaxAge` more aggressively so the warm migrator evicts Bleve documents as well.

**Resolve:** disk usage stops climbing within one warm-migration window (default 6h).

---

## Calling for help

If a runbook step fails and you can't isolate the cause within an hour, stop poking and:

1. Snapshot the data dir.
2. `oblivra-verify` the audit log; capture the output verbatim.
3. Capture `journalctl -u oblivra -n 5000`.
4. Send all three to `security@<your-domain>` with a description of what happened just before the symptom.

The platform is designed so that even a hard incident is reconstructible after the fact — the audit chain *is* the post-mortem. Don't re-write history; report it.
