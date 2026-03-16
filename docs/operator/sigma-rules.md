# Sigma Rules Guide

Import and manage community Sigma detection rules — no rewriting required.

---

## What is Sigma?

[Sigma](https://sigmahq.io) is a generic signature format for SIEM systems, maintained by the security community. The SigmaHQ repository contains **2,500+ detection rules** covering Windows, Linux, cloud, and network threats.

Oblivra includes a built-in Sigma transpiler that converts `.yml` Sigma files directly into its native detection engine — no conversion tool required.

---

## How It Works

1. You drop `.yml` Sigma files into the `sigma/` directory.
2. The alerting service loads them at startup (or within 500ms via hot-reload).
3. The transpiler converts each Sigma rule to a native Oblivra rule in memory.
4. Rules participate in the normal detection pipeline — alerts, MITRE mapping, deduplication all work identically.

---

## Directory Location

| Platform | Path |
|---|---|
| Windows | `%LOCALAPPDATA%\sovereign-terminal\data\sigma\` |
| macOS | `~/Library/Application Support/sovereign-terminal/data/sigma/` |
| Linux | `~/.local/share/sovereign-terminal/data/sigma/` |

Create the directory if it doesn't exist:

```powershell
# Windows
mkdir $env:LOCALAPPDATA\sovereign-terminal\data\sigma
```

```bash
# Linux / macOS
mkdir -p ~/.local/share/sovereign-terminal/data/sigma
```

---

## Installing the SigmaHQ Community Ruleset

```bash
# Clone the Sigma community repo
git clone --depth=1 https://github.com/SigmaHQ/sigma.git /tmp/sigma

# Copy the rules you want — start with Windows and Linux
cp /tmp/sigma/rules/windows/*.yml ~/.local/share/sovereign-terminal/data/sigma/
cp /tmp/sigma/rules/linux/*.yml   ~/.local/share/sovereign-terminal/data/sigma/

# Or copy everything
cp -r /tmp/sigma/rules/**/*.yml   ~/.local/share/sovereign-terminal/data/sigma/
```

Rules load automatically on next startup or within 500ms if the app is already running.

---

## Supported Sigma Constructs

| Construct | Supported |
|---|---|
| `title`, `id`, `description`, `status` | ✅ |
| `level` → severity mapping | ✅ |
| `tags` → MITRE ATT&CK mapping | ✅ |
| `logsource` → EventType hint | ✅ |
| `detection.keywords` | ✅ |
| `detection.selection` with field matching | ✅ |
| Field modifiers: `contains`, `startswith`, `endswith` | ✅ |
| Field modifier: `re` (regular expression) | ✅ |
| Field modifier: `all` (all values must match) | ✅ (approximated) |
| `condition: selection` | ✅ |
| `condition: keywords` | ✅ |
| `condition: selection1 and selection2` | ✅ |
| `timeframe` → `window_sec` | ✅ |
| `falsepositives` | ✅ (appended to description) |
| Near/within operators | ⚠️ Not supported (skipped) |
| Aggregate functions (`count by`, `sum by`) | ⚠️ Not yet supported |
| Multi-document correlated rules (`related:`) | ⚠️ Not yet supported |
| `status: deprecated` | ✅ (skipped automatically) |

---

## Severity Mapping

| Sigma `level` | Oblivra severity |
|---|---|
| `critical` | `critical` |
| `high` | `high` |
| `medium` | `medium` |
| `low` | `low` |
| `informational` | `low` |

---

## Checking Which Rules Loaded

1. Go to **Ops Center → Dashboard** — the **Active Rules** tile shows the total.
2. Go to **MITRE Heatmap** — shows which tactics have coverage.
3. Open **Diagnostics** (click `● A` in status bar) — the diagnostics modal shows rule count.

From the log file:

```
[SIGMA] Community Sigma rules loaded from sigma (2543 total rules active)
```

---

## Hot Reload

The alerting service watches the `sigma/` directory with `fsnotify`. When you add, modify, or remove a `.yml` file:

- The engine debounces changes for 500ms.
- Then reloads the entire directory atomically.
- Emits a `sigma:rules_reloaded` event to the frontend.

**No restart required.**

You can also trigger a manual reload from the UI at any time:
**Ops Center → Alerts → ↺ Reload Rules**

---

## Filtering Which Rules Load

If you want a subset of the SigmaHQ ruleset (e.g., only Windows rules with `high` or `critical` severity), filter before copying:

```bash
# Copy only high/critical Windows rules using yq
find /tmp/sigma/rules/windows -name '*.yml' | while read f; do
  level=$(yq '.level' "$f")
  if [[ "$level" == "high" || "$level" == "critical" ]]; then
    cp "$f" ~/.local/share/sovereign-terminal/data/sigma/
  fi
done
```

Or use the SigmaHQ pipeline converter if you need custom field mapping for your specific log format.

---

## Writing Sigma-Compatible Rules

Oblivra fully accepts standard Sigma syntax. See the [Sigma specification](https://github.com/SigmaHQ/sigma/blob/master/specification/sigma-specification.md) for the full format.

Example Sigma rule that works directly in Oblivra:

```yaml
title: Suspicious PowerShell Invocation via RunDLL32
id: e6eb5a96-9e6f-4a18-9841-8f376cf63e2a
status: stable
description: Detects RunDLL32 invoking PowerShell via JScript or VBScript.
references:
  - https://attack.mitre.org/techniques/T1218/011/
author: Your Name
date: 2026-01-01
tags:
  - attack.defense_evasion
  - attack.t1218.011
logsource:
  category: process_creation
  product: windows
detection:
  selection:
    Image|endswith: '\rundll32.exe'
    CommandLine|contains:
      - 'powershell'
      - 'posh'
  condition: selection
falsepositives:
  - Legitimate administrative scripts
level: high
```

---

## Troubleshooting

**Rules not loading?**

Check the application log for lines starting with `[SIGMA]`:

```
[SIGMA] Failed to load rule some_rule.yml: sigma rule missing title
[SIGMA] Hot-reload failed: ...
```

**Rule fires too often?**

Add or increase `dedup_window_sec` in the Sigma rule using Oblivra's extended fields (Oblivra-specific fields are passed through transparently):

```yaml
# Add to any Sigma rule for Oblivra-specific dedup
x-oblivra:
  dedup_window_sec: 3600
  group_by:
    - source_ip
```

**Deprecated rules?**

Sigma rules with `status: deprecated` are automatically skipped — no action needed.
