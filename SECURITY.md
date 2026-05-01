# Security policy

OBLIVRA is a security-critical product — operators rely on its
integrity guarantees to satisfy compliance regimes and forensic review.
Security reports are taken seriously and triaged ahead of feature work.

## Supported versions

Until OBLIVRA reaches v1.0, only the **current `main` branch** receives
security fixes. Tagged pre-1.0 releases are best-effort.

| Version | Status                |
|---------|-----------------------|
| `main`  | Active                |
| < v1.0  | Best-effort backports |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security bugs.**

Instead, email **security@oblivra.dev** with:

1. A clear description of the issue (component, attack vector).
2. Steps to reproduce — ideally a minimal proof of concept.
3. The version / commit you tested against.
4. Any suggested mitigation if you have one.

You should expect:

- Acknowledgement within **3 business days**.
- An initial triage and severity assessment within **7 business days**.
- A patch for confirmed high-severity issues within **30 days** of triage.
- A coordinated disclosure window of **60 days** by default — we are
  flexible if exploitation requires more lead time.

## Scope — in

- Authentication / authorization bypass on `/api/v1/...` endpoints.
- Audit log tampering that does not break the Merkle chain.
- Vault unlock bypass or key extraction.
- Agent-side signature forgery or signing-key disclosure.
- Tenant boundary violations (one tenant reading another's events).
- WORM enforcement bypass on parquet warm tier.
- TLS misconfiguration causing weakened transport.
- Denial-of-service that requires only a few well-formed requests.

## Scope — out

- DoS that requires saturating bandwidth or running thousands of
  concurrent ingest connections — that's an operations problem, not a
  protocol bug.
- Issues in dependencies that have already been patched upstream and
  are pending a routine bump.
- Findings on demo / example configuration files (`example/`) that ship
  with weakened defaults.
- Self-XSS or social engineering against the operator UI.

## Hardening guarantees

OBLIVRA's threat model assumes:

- The host is operator-controlled but the network is untrusted.
- An attacker may pause, replay, or drop network packets.
- An attacker may corrupt files at rest (the audit chain detects this).
- An attacker may submit malicious events through ingest.

It does **not** assume the host is fully compromised at the
hypervisor / firmware level — that scenario is out of scope for any
software-level security product.

## Cryptography

- Audit log: SHA-256 Merkle chain, optional HMAC-SHA256 root signature.
- Agent → server: TLS 1.3, optional mTLS, ed25519 per-event signing.
- Vault: AES-256-GCM with Argon2id KDF (m=64MiB, t=3, p=4).
- Webhook bodies: HMAC-SHA256 signed.

If you find a cryptographic primitive being misused (nonce reuse,
weak parameters, missing authentication), please report it.

## Hall of fame

Researchers credited with valid reports will be listed here once we
have any to credit. Credit is opt-in.
