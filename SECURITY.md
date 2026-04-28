# Security Policy

OBLIVRA is a sovereign SIEM platform. Security findings are taken seriously and handled in accordance with this policy.

## Supported Versions

Security fixes are applied to the **two most recent minor releases**. Older versions are out of scope unless the finding is a critical Tier-1 issue (RCE, auth bypass, mass data exposure) and the operator has a paid support contract.

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | ✅ Active          |
| 0.x     | ⚠️ Critical-only   |
| < 0.x   | ❌ End of life     |

## Reporting a Vulnerability

**Do not file public GitHub issues for security findings.** Use one of:

1. **Email:** `security@oblivra.io` (PGP key fingerprint below)
2. **GitHub private security advisory:** https://github.com/kingknull/oblivrashell/security/advisories/new
3. **Signal / Wire (high-severity only):** request a contact handle via the email above

When reporting, please include:

- A description of the issue and its impact (RCE, auth bypass, info disclosure, DoS, etc.)
- A reproducible proof of concept (script, curl invocation, or step-by-step)
- The version + commit SHA you reproduced against
- Whether the issue is currently under embargo from any other party

## What to Expect

| Stage                               | Target time |
| ----------------------------------- | ----------- |
| Initial acknowledgement             | **48 hours** |
| Severity triage + scope assessment  | **5 business days** |
| Fix or mitigation in main branch    | **30 days** for critical · **60 days** for high · **90 days** for medium · **180 days** for low |
| Coordinated public disclosure       | After patched release lands; bare minimum **7 days** notice from disclosure to release |
| CVE assignment (if eligible)        | Requested via MITRE within 14 days of confirmed vulnerability |

These are commitments. Slippage gets reported back to the reporter with reasoning, not silently.

## Severity Rubric (CVSS-aligned)

| Severity | Examples |
| -------- | -------- |
| **Critical** | Pre-auth RCE, mass data exposure, auth bypass on `/api/v1/*`, fleet credential disclosure |
| **High**     | Authenticated RCE, privilege escalation, multi-tenant data leak, signed-binary forgery |
| **Medium**   | Authenticated info disclosure, log injection, CSRF on non-destructive endpoints, unbounded request body OOM |
| **Low**      | Reflected XSS in admin pages, missing security headers, weak default settings |

## What Is *Not* a Vulnerability

- Findings against `OBLIVRA_DEBUG=true` mode (debug mode loosens CORS + relaxes some checks; this is documented)
- The agent's `--insecure` TLS-skip flag (development convenience; warning prints on use)
- Known stub services flagged in `docs/AUDIT.md` (DBPanel, osquery integration, etc.)
- Issues requiring physical access or operator-level admin compromise
- Findings against pre-release / `dev` builds

## Scope

In scope:
- The `oblivra-server` binary (REST API, ingest pipeline, detection engine)
- The `oblivra-agent` binary (telemetry collector)
- The `oblivrashell` desktop app (Wails frontend, embedded services)
- The `sigma/` rules and `playbooks/` definitions
- The Helm charts under `deploy/` (when included in a release)

Out of scope:
- Third-party dependencies (file upstream)
- The compiled artifacts of competitors
- Social engineering against project maintainers
- DoS attacks against the public docs site

## Public Disclosure

After a fix has shipped, we publish:
- A GitHub Security Advisory describing the issue and the patched commit
- A CVE if one was assigned
- A line in `CHANGELOG.md` referencing both

Reporter credit is given by default unless the reporter requests anonymity.

## Hall of Fame

Contributors who report responsibly disclosed findings are listed at https://oblivra.io/security/credits unless they opt out.

## PGP Key

```
SECURITY contact: security@oblivra.io
PGP fingerprint:  TO BE PUBLISHED ON FIRST RELEASE
```

The fingerprint will be published in the first numbered release alongside a signing-key release artifact. Until then, encrypted reporting is best handled via GitHub's encrypted advisory channel.

---

**Last reviewed:** 2026-04-27 · See `docs/security/` for engineering-side runbook.
