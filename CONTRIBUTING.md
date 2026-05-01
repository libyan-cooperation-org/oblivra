# Contributing to OBLIVRA

OBLIVRA is a sovereign log-driven security platform. Contributions that
strengthen the integrity, reproducibility, and operational story of the
platform are welcome.

## Quick start

```sh
# 1. Toolchain
go version             # >= 1.25
bun --version          # >= 1.1

# 2. Pull deps
go mod download
cd frontend && bun install && cd ..
cd frontend-web && bun install && cd ..

# 3. Run the headless server
go run ./cmd/server

# 4. Run the desktop app (Wails)
wails3 dev

# 5. Run the agent against the local server
go run ./cmd/agent run --config example/agent.yaml
```

## What kinds of changes we want

- **Detections**: new Sigma rules under `sigma/`, new local-rule patterns
  in `cmd/agent/predetect.go`, OQL example queries.
- **Parsers**: new sourcetypes in `internal/parsers/`. Each new parser
  must come with a corpus of golden inputs in `testdata/`.
- **Storage**: improvements to BadgerDB key sharding, parquet column
  encoding, S3 compatibility (we already speak SigV4 directly, no SDK).
- **Operator UX**: dashboard widgets, OQL composer, investigation tools.

## What we are cautious about

- Adding heavyweight dependencies — we ship as a single binary.
- Importing AWS / GCP / Azure SDKs — we hand-roll the API calls we need.
- Adding new endpoints that aren't audit-logged.
- Loosening auth defaults.

## Development workflow

1. Open an issue describing the change.
2. Fork → branch → implement → test → PR.
3. Each commit must build cleanly: `go build ./...` and `go vet ./...`
   must pass.
4. Add tests. We have unit, integration, soak, and chaos tiers; pick the
   tier that exercises your code.
5. If you change an HTTP endpoint, update `docs/openapi.yaml` and the
   smoke harness in `cmd/smoke`.

## Coding standards

- **Go**: standard `gofmt` formatting. Comments should explain *why*,
  not *what*.
- **Errors**: wrap with `fmt.Errorf("context: %w", err)`. Don't swallow.
- **Concurrency**: every shared struct documents which fields the mutex
  protects. Long-lived goroutines respond to `ctx.Done()`.
- **Audit**: any new state-changing endpoint goes through the audit
  middleware so the operator gets a tamper-evident record.

## Sigma rules

Place new rules under `sigma/<vendor>/<title>.yml`. The loader normalises
field names against the OBLIVRA schema; if your rule requires a field we
don't surface, add it to `internal/parsers/normalise.go` first.

## Reporting issues

For bugs that affect integrity guarantees (audit chain, signing,
tiering, vault), please use the security disclosure process described
in [SECURITY.md](SECURITY.md) instead of public issues.

For everything else, GitHub issues are fine.

## Code of conduct

Be kind. Engineering disagreements are fine; personal attacks are not.
We will close threads that turn hostile.
