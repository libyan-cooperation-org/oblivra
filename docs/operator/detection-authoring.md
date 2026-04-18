# Detection Rule Authoring Guide

Write custom detection rules for Oblivra's native YAML engine.

---

## Rule Format

```yaml
# Required fields
id: "unique_rule_id"          # Unique string — no spaces
name: "Human-Readable Name"   # Shown in alerts and UI
description: "What this detects and why it matters."
severity: "critical"          # critical | high | medium | low
type: "threshold"             # threshold | sequence

# Triggering threshold
threshold: 5       # Number of matching events required
window_sec: 60     # Sliding time window in seconds

# Deduplication: suppress repeated alerts for the same entity
dedup_window_sec: 3600   # Seconds before the same group key can alert again

# MITRE ATT&CK mapping
mitre_tactics:
  - "TA0001"     # Initial Access
mitre_techniques:
  - "T1110.001"  # Brute Force: Password Guessing

# Grouping: track counts per unique entity
group_by:
  - "source_ip"  # Track per source IP
  - "user"       # or per user, or both

# Match conditions (all must be true)
conditions:
  EventType: "failed_login"
  source_ip: "cidr:10.0.0.0/8"
  output_contains: "regex:(?i)Failed password|Authentication failure"
```

---

## Rule Types

### `threshold` (most common)

Fires when `threshold` events matching the conditions occur within `window_sec` seconds.

```yaml
type: "threshold"
threshold: 10
window_sec: 60
```

### `sequence`

Fires when a specific causal chain of events occurs in order within the time window.
Each step has its own conditions.

```yaml
type: "sequence"
window_sec: 300
sequence:
  - name: "Port Scan"
    conditions:
      EventType: "network_connection"
      output_contains: "regex:(?i)SYN"
  - name: "Exploit Attempt"
    conditions:
      EventType: "network_connection"
      output_contains: "regex:(?i)exploit|shellcode"
  - name: "Shell Spawned"
    conditions:
      EventType: "linux_process_create"
      output_contains: "bash|sh"
```

---

## Condition Fields

| Field | Description | Example |
|---|---|---|
| `EventType` | Normalized event type | `failed_login` |
| `source_ip` | Source IP, exact or CIDR | `cidr:192.168.0.0/16` |
| `user` | Username | `root` |
| `host` | Host/entity ID | `web-01` |
| `output_contains` | Match against `RawLog` | `regex:(?i)error` |
| `location` | Geographic location | `regex:(?i)CN|RU` |

### Pattern types

**Exact match:**
```yaml
EventType: "failed_login"
user: "root"
```

**Regex match** (prefix with `regex:`):
```yaml
output_contains: "regex:(?i)Failed password|Authentication failure|Invalid user"
```

**CIDR range** (for `source_ip` only):
```yaml
source_ip: "cidr:10.0.0.0/8"
```

---

## EventType Reference

These are the normalized event types the engine understands:

| EventType | Source |
|---|---|
| `failed_login` | SSH, PAM, Windows Security |
| `successful_login` | SSH, PAM, Windows Security |
| `sudo_exec` | Linux sudo |
| `credential_dump` | LSASS access, /etc/shadow read |
| `process_creation` | Sysmon, auditd |
| `windows_process_create` | Sysmon Event ID 1 |
| `linux_process_create` | auditd EXECVE |
| `windows_registry_set` | Sysmon Event ID 13 |
| `windows_service_install` | Windows System log Event 7045 |
| `windows_logon` | Windows Security Event 4624/4625 |
| `windows_kerberos` | Windows Security Event 4769 |
| `windows_ad_replication` | Windows Security Event 4662 |
| `file_write` | auditd, Sysmon |
| `network_connection` | Zeek conn.log, Sysmon Event 3 |
| `network_dns` | Zeek dns.log |
| `network_smb` | Zeek smb_files.log |
| `network_modbus` | Zeek modbus.log |
| `aws_cloudtrail` | AWS CloudTrail |
| `azure_signin` | Azure AD Sign-in logs |
| `cron_modified` | Linux cron file changes |

---

## MITRE ATT&CK Reference

Use standard MITRE IDs. Key tactics:

| ID | Tactic |
|---|---|
| TA0001 | Initial Access |
| TA0002 | Execution |
| TA0003 | Persistence |
| TA0004 | Privilege Escalation |
| TA0005 | Defense Evasion |
| TA0006 | Credential Access |
| TA0007 | Discovery |
| TA0008 | Lateral Movement |
| TA0009 | Collection |
| TA0010 | Exfiltration |
| TA0011 | Command and Control |
| TA0040 | Impact |

---

## Examples

### SSH brute force (5 failures from same IP in 60s)

```yaml
id: "ssh_brute_force"
name: "SSH Brute Force Attack"
description: "5 or more failed SSH logins from the same IP within 60 seconds."
severity: "high"
type: "threshold"
threshold: 5
window_sec: 60
dedup_window_sec: 900
mitre_tactics: ["TA0001"]
mitre_techniques: ["T1110.001"]
group_by: ["source_ip"]
conditions:
  EventType: "failed_login"
```

### Ransomware precursor sequence

```yaml
id: "ransomware_precursor_chain"
name: "Ransomware Precursor: Shadow Delete + Defender Disable"
description: "Detects the two-step pattern seen in most ransomware: disable AV then delete backups."
severity: "critical"
type: "sequence"
window_sec: 300
dedup_window_sec: 3600
mitre_tactics: ["TA0040", "TA0005"]
mitre_techniques: ["T1490", "T1562.001"]
sequence:
  - name: "Defender Disabled"
    conditions:
      EventType: "windows_process_create"
      output_contains: "regex:(?i)Set-MpPreference.{0,30}DisableRealtimeMonitoring"
  - name: "Shadow Copies Deleted"
    conditions:
      EventType: "windows_process_create"
      output_contains: "regex:(?i)vssadmin.{0,20}delete|wbadmin.{0,20}delete"
```

### Off-hours admin login

```yaml
id: "offhours_admin_login"
name: "Admin Login Outside Business Hours"
description: "Privileged account login between 20:00 and 06:00."
severity: "medium"
type: "threshold"
threshold: 1
window_sec: 300
dedup_window_sec: 3600
mitre_tactics: ["TA0001"]
mitre_techniques: ["T1078"]
group_by: ["user"]
conditions:
  EventType: "successful_login"
  user: "regex:(?i)^(admin|root|administrator|svc-)"
```

---

## File Location

Place rule files in the `rules/` directory inside the data folder:

- **Windows**: `%LOCALAPPDATA%\sovereign-terminal\data\rules\`
- **macOS/Linux**: `~/.local/share/sovereign-terminal/data/rules/`

The engine loads all `.yaml` and `.yml` files from this directory on startup.

For **Sigma community rules**, use the `sigma/` directory instead — see the [Sigma Rules Guide](sigma-rules.md).

---

## Hot Reload

Drop a new `.yaml` file into `sigma/` and the alerting service reloads rules within 500ms — **no restart required**.

To trigger a manual reload from the UI: **Ops Center → Alerts → ↺ Reload Rules**.

---

## Testing a Rule

Use the **Purple Team Engine** (`/purple-team`) to replay attack scenarios against your rule set and see which rules fire.

For quick CLI testing, send a matching event via syslog:

```bash
echo '<34>1 2026-01-01T00:00:00Z myhost sshd - - Failed password for root from 10.0.0.1' | \
  nc -u 127.0.0.1 1514
```

Check **Ops Center → Alerts → History** for the resulting alert within a few seconds.
