# Integration: SOC Prime DetectFlow

[DetectFlow](https://github.com/socprime/detectflow-main) is a pre-SIEM
streaming detection layer (Apache Flink + Sigma rules on Kafka topics).
It tags events with rule matches in-flight at sub-millisecond latency,
*before* a SIEM ever sees them.

OBLIVRA reads DetectFlow's output directly from Kafka, so the enriched
stream lands in the audit-grade store without an HTTP round trip.

## Pipeline

```
log sources
    │
    ▼
Apache Kafka  ◄───── DetectFlow (Flink + Sigma)
    │                    │
    │                    └─ writes back to <topic>.matched with
    │                       rule-match tags as Kafka headers
    ▼
OBLIVRA  (consumer-group: oblivra)
    │
    ├─ ingest pipeline → WAL → hot store → Bleve
    └─ rule-match headers surface as event Fields:
       kafka:rule.id, kafka:rule.severity, kafka:rule.tags
```

## OBLIVRA configuration

The Kafka listener is opt-in via environment variables. Set
`OBLIVRA_KAFKA_BROKERS` to enable it; everything else defaults sensibly.

| Variable | Default | Purpose |
|---|---|---|
| `OBLIVRA_KAFKA_BROKERS`         | _(unset = disabled)_ | Comma-separated `host:port` list |
| `OBLIVRA_KAFKA_TOPICS`          | _(required)_         | Comma-separated topic list; one consumer goroutine per topic |
| `OBLIVRA_KAFKA_GROUP`           | `oblivra`            | Consumer group ID — change if you run multiple OBLIVRA instances against the same broker |
| `OBLIVRA_KAFKA_SECURITY_PROTOCOL` | `PLAINTEXT`        | `PLAINTEXT` / `SSL` / `SASL_PLAINTEXT` / `SASL_SSL` |
| `OBLIVRA_KAFKA_SASL_MECHANISM`  | _(none)_             | `PLAIN` / `SCRAM-SHA-256` / `SCRAM-SHA-512` |
| `OBLIVRA_KAFKA_SASL_USERNAME`   | _(none)_             | SASL username |
| `OBLIVRA_KAFKA_SASL_PASSWORD`   | _(none)_             | SASL password |
| `OBLIVRA_KAFKA_TLS_CA_FILE`     | _(OS truststore)_    | PEM CA bundle for the broker cert |
| `OBLIVRA_KAFKA_TLS_CERT_FILE`   | _(none)_             | mTLS client cert |
| `OBLIVRA_KAFKA_TLS_KEY_FILE`    | _(none)_             | mTLS client key |
| `OBLIVRA_KAFKA_TLS_SKIP_VERIFY` | `0`                  | `1` skips broker cert verification (dev only) |
| `OBLIVRA_KAFKA_START_FROM_OLDEST` | `0`                | `1` reads the topic from offset 0 on first run; default reads only new messages |

### Example: VPN-attached lab cluster (no auth)

```env
# /etc/oblivra/oblivra.env
OBLIVRA_KAFKA_BROKERS=kafka.detectflow.svc.cluster.local:9092
OBLIVRA_KAFKA_TOPICS=detectflow.matched,detectflow.raw
OBLIVRA_KAFKA_START_FROM_OLDEST=1
```

### Example: production with SCRAM + TLS

```env
OBLIVRA_KAFKA_BROKERS=kafka-0:9094,kafka-1:9094,kafka-2:9094
OBLIVRA_KAFKA_TOPICS=detectflow.matched
OBLIVRA_KAFKA_SECURITY_PROTOCOL=SASL_SSL
OBLIVRA_KAFKA_SASL_MECHANISM=SCRAM-SHA-512
OBLIVRA_KAFKA_SASL_USERNAME=oblivra-consumer
OBLIVRA_KAFKA_SASL_PASSWORD=…
OBLIVRA_KAFKA_TLS_CA_FILE=/etc/oblivra/kafka-ca.pem
```

## What ends up in the event

Every Kafka record becomes one OBLIVRA event with:

- **Provenance** — `IngestPath: "kafka"`, `Format: <topic>`, `Peer: <broker list>`
- **Fields** — auto-extracted via the existing `parsers.FormatAuto` path
  (JSON / RFC5424 / RFC3164 / CEF / auditd) plus:
  - `kafkaTopic` — source topic
  - `kafkaPartition` / `kafkaOffset` — for replay correlation
  - `kafkaKey` — record key (often the host ID)
  - `kafka:<header>` — every Kafka record header surfaces as a field,
    so DetectFlow's `rule.id`, `rule.severity`, `rule.tags` headers
    become first-class search/OQL targets

- **Timestamp** — Kafka record timestamp (when DetectFlow wrote it),
  not the time OBLIVRA received it. This preserves the in-flight
  detection ordering exactly.

## Searching DetectFlow matches in OBLIVRA

```bash
# All events tagged by a specific Sigma rule
oblivra-cli search --q 'kafka\:rule.id:sigma-ssh-bruteforce'

# Cross-tier verify the warm-tier parquet has the kafka headers preserved
curl -s -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  http://localhost:8080/api/v1/storage/verify-warm | jq .
```

## Reliability behavior

- **At-least-once delivery** — OBLIVRA commits the Kafka offset only
  after the event has been accepted by the ingest pipeline (WAL write
  fsynced). On a crash mid-batch the broker re-delivers; the WAL's
  duplicate-detection layer handles the idempotency.
- **Backpressure** — if OBLIVRA's hot store is overwhelmed, the
  consumer stops fetching and the broker queues records server-side
  rather than the client buffering unbounded RAM.
- **Bad records** — payloads larger than `MaxBytesPerRecord` (1 MiB)
  are skipped and committed (so a single misbehaving producer can't
  wedge the pipeline forever); the offset is logged at WARN level.

## Co-deployment topology (single Linux host)

For a small DetectFlow + OBLIVRA lab, both can run on one VM:

```yaml
# docker-compose.yml fragment
services:
  kafka:
    image: bitnami/kafka:3.8
    ports: ["9092:9092"]
    environment:
      KAFKA_CFG_PROCESS_ROLES: controller,broker
      # ...
  detectflow:
    # see https://github.com/socprime/detectflow-one-click-local-deployment
    depends_on: [kafka]
  oblivra:
    image: ghcr.io/libyan-cooperation-org/oblivra:latest
    environment:
      OBLIVRA_KAFKA_BROKERS: kafka:9092
      OBLIVRA_KAFKA_TOPICS: detectflow.matched
      OBLIVRA_API_KEYS: ${OBLIVRA_API_KEYS}
      OBLIVRA_AUDIT_KEY: ${OBLIVRA_AUDIT_KEY}
    depends_on: [detectflow]
    ports: ["8080:8080"]
```

## License compatibility

- DetectFlow: EUPL v1.2 / SOC Prime Commercial
- OBLIVRA: Apache 2.0
- Integration is **wire-protocol only** (Kafka topics) — no DetectFlow
  code is statically linked into OBLIVRA, so the license boundary is
  clean.

## Ops runbook

| Symptom | Where to look |
|---|---|
| `kafka fetch` warnings every 2s | Broker unreachable. Check `OBLIVRA_KAFKA_BROKERS` resolves and the security protocol matches the listener config. |
| 0 records ingested but DetectFlow is producing | Check consumer group lag: `kafka-consumer-groups.sh --bootstrap-server kafka:9092 --describe --group oblivra`. If LAG is high, OBLIVRA fell behind; if `LAG=-` the group hasn't subscribed (likely an env var typo). |
| `kafka record exceeds max-bytes` | A producer is shipping payloads >1 MiB. Either fix the producer or bump `MaxBytesPerRecord` in `KafkaConfig`. |
| Restart re-reads days of history | You set `OBLIVRA_KAFKA_START_FROM_OLDEST=1`. Default is to read only new messages. |
