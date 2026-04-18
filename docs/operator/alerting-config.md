# Alerting Configuration Guide

Configure multi-channel notifications so alerts reach the right people instantly.

---

## Overview

Oblivra sends alerts through multiple channels simultaneously. Credentials are stored locally in your encrypted vault — nothing is sent to a cloud configuration service.

Supported channels:

| Channel | Use case |
|---|---|
| Email (SMTP) | SOC team distribution lists, ticketing system integration |
| Telegram | Instant mobile alerts for on-call analysts |
| SMS (Twilio) | Critical alerts when internet is unreliable |
| WhatsApp (Twilio) | High-trust mobile channel for executive alerts |
| Webhook | Slack, Discord, Teams, PagerDuty, any HTTP endpoint |

---

## Accessing Alert Configuration

**Ops Center → Alerts → 📡 Notification Channels**

All channel configuration is in this single panel. Click **Save Configuration** after any change. Click **Test Connection** to verify delivery before saving.

---

## Email (SMTP)

### Configuration fields

| Field | Example |
|---|---|
| SMTP Host | `smtp.gmail.com` |
| Port | `587` (STARTTLS) or `465` (SSL) |
| Username | `alerts@yourcompany.com` |
| Password | App password (not account password for Google) |
| Send To | `soc-team@yourcompany.com` |

### Gmail setup

1. Enable 2-factor authentication on your Google account.
2. Go to Google Account → Security → App passwords.
3. Generate an app password for "Mail".
4. Use `smtp.gmail.com`, port `587`, your Gmail address as username, app password as password.

### Office 365 setup

```
Host:     smtp.office365.com
Port:     587
Username: your-email@domain.com
Password: your-password
```

### Self-hosted (Postfix / SendGrid)

```
Host:     mail.yourdomain.com
Port:     587
Username: noreply@yourdomain.com
Password: your-smtp-password
```

---

## Telegram

Best for instant mobile alerts. Delivers rich formatted messages.

### Setup steps

1. Open Telegram. Search for **@BotFather**.
2. Send `/newbot` and follow the prompts to create a bot.
3. Copy the **Bot Token** (format: `1234567890:ABCDEFGhijklmnop...`).
4. Start a conversation with your new bot (or add it to a group).
5. Search for **@userinfobot** and send it a message — it returns your **Chat ID**.
6. Enter both values in Oblivra's Telegram config.

For group alerts, add the bot to a Telegram group and use the group's Chat ID (starts with `-`).

---

## SMS & WhatsApp (Twilio)

### Twilio setup

1. Create a free account at [twilio.com](https://www.twilio.com).
2. Get your **Account SID** and **Auth Token** from the Console dashboard.
3. Buy a phone number ($1/month) — enable SMS and/or WhatsApp.
4. For WhatsApp: activate the Twilio WhatsApp sandbox or submit for production access.

### Configuration fields

| Field | Example |
|---|---|
| Account SID | `ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| Auth Token | `your-auth-token` |
| From Number | `+12025551234` |
| To Number | `+447700900123` |

Enable **SMS**, **WhatsApp**, or both independently.

---

## Webhook

Sends a JSON POST to any HTTP endpoint. Works with Slack, Discord, Teams, PagerDuty, and any custom integration.

### Slack

1. Go to your Slack workspace → Apps → Incoming Webhooks.
2. Click **Add to Slack** and choose a channel.
3. Copy the webhook URL (format: `https://hooks.slack.com/services/T.../B.../...`).
4. Paste into Oblivra's Webhook URL field.

### Discord

1. In Discord, right-click a channel → Edit Channel → Integrations → Webhooks.
2. Create a webhook and copy the URL.
3. Append `/slack` to make it Slack-compatible: `https://discord.com/api/webhooks/.../slack`

### Microsoft Teams

1. In Teams, click the `...` next to a channel → Connectors → Incoming Webhook.
2. Configure and copy the webhook URL.

### Webhook secret (optional)

If provided, Oblivra signs the webhook payload with HMAC-SHA256:

```
X-Oblivra-Signature: sha256=<hex>
```

Verify this in your receiver to authenticate the sender.

### Custom webhook payload

The JSON body sent to your webhook:

```json
{
  "alert": {
    "rule_id": "windows_shadow_copy_deletion",
    "rule_name": "Shadow Copy Deletion",
    "severity": "critical",
    "description": "Detects deletion of Volume Shadow Copies...",
    "triggered_at": "2026-03-16T14:32:00Z",
    "entity": "10.0.0.5",
    "event_count": 1,
    "mitre_tactics": ["TA0040"],
    "mitre_techniques": ["T1490"]
  }
}
```

---

## Alert Triggers (Regex Patterns)

Beyond the YAML detection rules, you can create simple pattern-match triggers directly in the UI.

**Ops Center → Alerts → ⚡ Alert Triggers**

| Field | Description |
|---|---|
| Name | Human-readable label |
| Regex Pattern | Matched against `raw_log` of every ingested event |
| Severity | critical / high / medium / low |

Example patterns:

```
# Detect any mention of "ransomware" in logs
ransomware

# Detect SQL injection attempts in web logs
(?i)union.{0,20}select|' or '1'='1

# Detect outbound connections to Tor exit nodes
\.onion\.

# Detect certificate errors that may indicate MITM
(?i)certificate.{0,30}(expired|invalid|untrusted)
```

These triggers fire immediately on each matching event (threshold = 1). For rate-limited detection, use YAML rules instead.

---

## Testing Alerts

### Test button

Click **Test Connection** in any channel config. This sends a synthetic test alert through all enabled channels immediately.

### Send a real test event

```bash
# SSH failure (triggers brute-force rules)
echo '<34>1 2026-01-01T00:00:00Z test-host sshd - - Failed password for root from 10.0.0.1' | \
  nc -u 127.0.0.1 1514

# Ransomware precursor (triggers shadow copy rule)
echo '<34>1 2026-01-01T00:00:00Z win-host powershell - - vssadmin.exe delete shadows /all' | \
  nc -u 127.0.0.1 1514
```

Check **Ops Center → Alerts → History** for alert delivery status.

---

## Alert History

**Ops Center → Alerts → History** shows:

- Timestamp
- Rule that fired
- Severity
- Entity (host/IP/user)
- Delivery status (NOTIFIED / LOGGED)

Alerts are retained for the duration configured in **Executive → Data Lifecycle → Alert Retention**.

---

## Suppression and Deduplication

Oblivra automatically deduplicates alerts per entity using the `dedup_window_sec` field in each rule. This prevents alert storms when a single attack triggers the same rule hundreds of times.

To permanently suppress a noisy rule without deleting it: set `dedup_window_sec: 86400` (24 hours) in the rule file and the watcher will hot-reload it.

To disable a rule entirely: rename the file with a `.disabled` extension or delete it from the `sigma/` or `rules/` directory.
