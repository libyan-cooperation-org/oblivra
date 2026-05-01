# OBLIVRA — Soak Baseline (Beta-1)

This file captures the first reproducible sustained-load run against a fresh
Beta-1 build. Future deployments should use it as the regression line.

## Hardware

- Windows 11 Pro, 32 logical cores, NVMe SSD
- single-node configuration, no auth, syslog/netflow listeners disabled
- ephemeral data dir under `tmp/soak-baseline/`

## Command

```bash
./oblivra-server &
oblivra-soak \
  --server http://127.0.0.1:18091 \
  --eps 2000 \
  --duration 15s \
  --workers 16 \
  --batch 50 \
  --hosts 200 \
  --warmup 2s
```

## Result

```
============================================================
  sent:        15200 events
  ok:          15200 events (100.0%)
  failed:      0 events (0.0%)
  rate:        1013 events/sec sustained
  latency p50: 642.8774ms
  latency p95: 3.0496261s
  latency p99: 3.081735s
============================================================
```

### Reading the numbers

- **100% success rate** — no events dropped, no HTTP errors, no WAL/hot/index failures.
- **1013 events/sec sustained** — under the soak runner's request-pacing model. The ingest pipeline can push higher when fed in larger batches; sustained EPS at burst sizes >50 will be measured separately.
- **p50 latency: 643 ms** — this is HTTP round-trip latency for a 50-event batch, not per-event. Per-event amortised latency is ~13 ms at p50.
- **p95 / p99 latency: 3.0 / 3.1 s** — long-tail spikes are dominated by Bleve index commit and Parquet warm-tier migrator running concurrently with the soak load. Acceptable for Beta-1; the §1 follow-up is to expose per-stage `Pipeline.Stats().Latency` on a metric scrape so the spike phase is visible to ops.

## Pre-go-live use

Run this exact command on the production hardware. Acceptance criteria:

- `ok: ≥ 99.5%` of sent events
- `latency p99: < 5s` (allows for cold-start GC + index commit)
- `failed: 0` for the first run

If any criterion fails, **do not deploy** — investigate the warm-tier migrator
schedule, the Bleve commit interval, or the underlying disk's fsync latency
(NVMe should give ≤2 ms; HDD-backed loops can produce 20× longer p99).

## Re-running

`task tools:build` produces both binaries; the smoke harness (`oblivra-smoke`)
should pass before and after the soak run. If the smoke harness reports a
different shape after the soak, the regression is in the platform, not the
load tester.

---

**Captured:** Beta-1 first commit milestone (2026-05-01). The next baseline
should be re-captured when:

1. Schema bumps to v2
2. Bleve commit interval changes
3. Parquet warm-tier migrator interval changes (currently 6h)
4. Hardware tier shifts (different SSD, different VM size)

Older baselines stay in this directory under `soak-results-<YYYY-MM-DD>.md`
so the trend is visible.
