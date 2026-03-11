# Zero Internet Dependency Audit

> **Status:** Complete  
> **Audited:** 2026-03-11  
> **Scope:** All runtime dependencies of the OBLIVRA sovereign stack  
> **Goal:** Confirm that OBLIVRA can operate in a fully air-gapped environment with zero outbound internet access after initial deployment.

---

## Audit Methodology

Each dependency was classified against the following criteria:

| Class | Meaning |
|---|---|
| ✅ **Offline-Capable** | Functions fully without any internet access |
| ⚠️ **Gracefully Degrades** | Falls back cleanly; feature is disabled but core platform stays healthy |
| ❌ **Internet-Required** | Fails or is non-functional without connectivity — mitigations documented |

---

## Core Runtime Components

### Storage & Indexing
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| BadgerDB | Embedded | ✅ Offline-Capable | Pure local disk, zero network |
| Bleve | Embedded | ✅ Offline-Capable | In-process full-text search |
| SQLite (via go-sqlite3) | Embedded | ✅ Offline-Capable | Local file-based database |
| Parquet archival | Embedded | ✅ Offline-Capable | Local file output |

### Networking & Protocol Handlers
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| SSH Client/Server | `golang.org/x/crypto/ssh` | ✅ Offline-Capable | Operates on LAN, no internet |
| Syslog ingestion | Built-in UDP/TCP | ✅ Offline-Capable | Local network only |
| gRPC agent transport | Embedded | ✅ Offline-Capable | mTLS over LAN |
| NetFlow/IPFIX collector | Built-in UDP | ✅ Offline-Capable | Local network tap |

### Security & Cryptography
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| AES-256 vault encryption | `crypto/aes` stdlib | ✅ Offline-Capable | |
| ed25519 signing | `crypto/ed25519` stdlib | ✅ Offline-Capable | Used for update bundle signing |
| FIDO2 / YubiKey | `github.com/google/go-tpm` | ✅ Offline-Capable | USB hardware key, no network |
| TOTP MFA | Built-in | ✅ Offline-Capable | Time-based, no server needed |
| TLS certificates | Built-in `crypto/tls` | ✅ Offline-Capable | Self-signed or private CA |

### Enrichment & Intelligence
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| GeoIP (MaxMind) | Local `.mmdb` file | ✅ Offline-Capable | Ship DB in offline bundle |
| DNS enrichment | Local DNS resolver | ✅ Offline-Capable | Uses system resolver |
| STIX/TAXII intel feed | External TAXII server | ⚠️ Gracefully Degrades | Falls back to local IOC cache; TAXII fetch fails cleanly |
| Threat intel IOC matching | Local BadgerDB cache | ✅ Offline-Capable | Populated from last sync or offline bundle |

### AI / ML Components
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| UEBA Isolation Forest | Embedded Go | ✅ Offline-Capable | Fully deterministic, in-process |
| Behavioral baseline storage | BadgerDB | ✅ Offline-Capable | |
| AI assistant (LLM) | External API | ⚠️ Gracefully Degrades | `ai_service.go` returns graceful error; UI shows "AI offline" |

### Update System
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| Online update check | GitHub API | ⚠️ Gracefully Degrades | `CheckUpdate()` returns error; UI shows "no network" |
| Offline update bundles | USB / local path | ✅ Offline-Capable | `ApplyVerifiedOfflineBundle()` — fully air-gap safe |
| Signature verification | `crypto/ed25519` stdlib | ✅ Offline-Capable | Sovereign key embedded at build time |

### Compliance & Reporting
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| PDF report generation | `jung-kurt/gofpdf` | ✅ Offline-Capable | Local generation, no CDN fonts |
| Compliance packs (YAML) | Local filesystem | ✅ Offline-Capable | Bundled in binary |
| Evidence Merkle tree | Embedded | ✅ Offline-Capable | |

### Authentication & Identity
| Component | Dependency | Air-Gap Status | Notes |
|---|---|---|---|
| Local user auth (bcrypt) | Stdlib | ✅ Offline-Capable | |
| OIDC / OAuth2 | External IdP | ⚠️ Gracefully Degrades | Disabled in air-gap mode; falls back to local auth |
| SAML 2.0 | External IdP | ⚠️ Gracefully Degrades | Same as OIDC |

---

## Offline Bundle Contents (Required for Full Air-Gap Deployment)

The following must be included in any offline deployment bundle to ensure full functionality:

```
oblivra_airgap_bundle/
├── oblivrashell_<version>_<os>_<arch>       # Signed binary
├── oblivrashell_<version>_<os>_<arch>.sig   # ed25519 signature
├── oblivrashell_<version>_<os>_<arch>.sha256 # SHA-256 sidecar
├── GeoLite2-City.mmdb                       # MaxMind GeoIP database
├── GeoLite2-ASN.mmdb                        # MaxMind ASN database
├── ioc_cache.json                           # Last-known IOC snapshot
├── detection_rules/                         # YAML detection rules
│   └── *.yaml
└── README_AIRGAP.md                         # Deployment instructions
```

---

## Graceful Degradation Matrix

All internet-dependent features implement a consistent degradation pattern:

1. **Attempt** the network operation with a short timeout (5–10s)
2. **Log** the failure at `WARN` level with context
3. **Return** a typed error that the calling service handles
4. **UI** shows an offline indicator — the rest of the platform continues operating

No internet-dependent feature will cause a panic, crash, or prevent platform startup.

---

## Verification Procedure

To verify air-gap safety before deployment:

```bash
# 1. Block all outbound traffic on the test machine
sudo iptables -P OUTPUT DROP
sudo iptables -A OUTPUT -d 0.0.0.0/8 -j ACCEPT   # loopback
sudo iptables -A OUTPUT -d 192.168.0.0/16 -j ACCEPT  # LAN only

# 2. Start OBLIVRA
./oblivrashell

# 3. Verify startup completes without fatal errors
# 4. Confirm all core features operational: SIEM, Vault, SSH, Agents, Compliance
# 5. Confirm graceful degradation: AI assistant, OIDC, TAXII show offline status in UI
```

---

## Residual Internet Dependencies (Accepted)

| Feature | Reason | Mitigation |
|---|---|---|
| NTP time sync | Accurate timestamps for SIEM events | Use local NTP server in air-gap environment |
| DNS resolution | Hostname lookups for enrichment | Point to internal DNS in air-gap environment |
| OIDC / SAML | Enterprise SSO | Disable in `config.yaml`; use local auth |
| AI LLM assistant | Command generation | Disable in `config.yaml`; shows offline state |
| TAXII intel feeds | Live IOC updates | Use periodic offline bundle sync |

---

*Audit performed by: OBLIVRA Engineering*  
*Next review: 2026-09-11*
