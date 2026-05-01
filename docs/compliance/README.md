# Compliance Adapter Manifest (Phase 46)

OBLIVRA does not produce SOC 2 / PCI-DSS / ISO 27001 / etc. attestation
PDFs. That cut was deliberate (Phase 36) — pair the platform with a
dedicated GRC tool (Drata, Vanta, Tugboat Logic, Hyperproof) and feed it
the audit-grade evidence we already produce.

The integration surface is a single read-only JSON-LD endpoint:

```
GET /api/v1/compliance/feed/{framework}
```

The compliance tool polls this on its own cadence. Each control returns:

- `controlId` — framework's identifier (e.g. `pci-dss-4:10.2.1`)
- `title` — short human label
- `evidenceType` — `audit-log` / `detection-alert` / `audit-verify` / `case-investigation`
- `sourceEndpoint` — which OBLIVRA endpoint provides the granular data
- `lastSeenAt` — RFC3339 timestamp of the most recent matching evidence
- `count24h` — count over the last 24h (freshness signal)
- `fresh` — boolean; `false` if `lastSeenAt` is stale (>7d) or missing

## Frameworks

| Framework path     | Standard                        | Controls covered                                  |
|--------------------|---------------------------------|---------------------------------------------------|
| `pci-dss-4`        | PCI-DSS v4.0                    | 10.2.1, 10.2.2, 10.5.5, 11.5.1                    |
| `soc2`             | SOC 2 Type II                   | CC6.1, CC6.6, CC7.2, CC7.3, CC8.1                 |
| `nist-800-53`      | NIST 800-53 Rev 5               | AU-2, AU-9, AC-2, IR-4, SI-4                      |
| `iso-27001`        | ISO/IEC 27001:2022              | A.5.27, A.8.15, A.8.16                            |
| `gdpr`             | GDPR                            | Art. 25, 30, 32, 33                               |
| `hipaa`            | HIPAA Security Rule             | 164.312(b), 164.312(c)(1), 164.312(d)             |

## Adding a framework

Append to `complianceFrameworks` in
[internal/httpserver/compliance.go](../../internal/httpserver/compliance.go).
Each entry is a `complianceControl` mapping a control ID to the audit-action
prefix (or arbitrary matcher func) that proves it. Submit a PR — the
mapping is intentionally code-not-config so changes go through review.

## What this does NOT do

- It does not stamp any compliance claim or generate a report PDF.
- It does not interpret control language — just maps actions to control IDs.
- It does not assess whether a control is "passing" — that's the GRC tool's call.

## Example

```bash
curl -s -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  http://localhost:8080/api/v1/compliance/feed/soc2 | jq .
```

```json
{
  "@context": "https://oblivra.dev/compliance/v1",
  "@type": "ComplianceFeed",
  "framework": "soc2",
  "generatedAt": "2026-05-01T12:34:56Z",
  "controls": [
    {
      "@type": "ComplianceEvidence",
      "controlId": "CC6.1",
      "title": "Logical access controls — auth events",
      "evidenceType": "audit-log",
      "sourceEndpoint": "/api/v1/audit/log",
      "lastSeenAt": "2026-05-01T12:33:14Z",
      "count24h": 487,
      "fresh": true
    }
  ]
}
```
