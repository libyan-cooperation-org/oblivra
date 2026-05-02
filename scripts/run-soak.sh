#!/usr/bin/env bash
# scripts/run-soak.sh — credibility-grade soak runner.
#
# Boots a clean OBLIVRA server, waits for /healthz, runs oblivra-soak
# at the configured EPS for the configured duration, archives the
# JSON + markdown reports under docs/operator/, then shuts the server
# down cleanly.
#
# Usage:
#   ./scripts/run-soak.sh [eps] [duration] [hardware-label]
#
# Defaults aim at "credibility check on dev hardware":
#   eps=1000  duration=60s  hardware=auto-detected
#
# For the published 10k-EPS number an operator runs:
#   ./scripts/run-soak.sh 10000 5m "AWS c6i.4xlarge / 16 vCPU / 32 GiB"
#
# Pass criteria (all gates must pass for exit 0):
#   - sustained EPS  >= 0.95 * target
#   - error rate     <= 0.01
#   - p99 latency    <= 500ms

set -euo pipefail

EPS="${1:-1000}"
DURATION="${2:-60s}"
HARDWARE="${3:-${OBLIVRA_SOAK_HARDWARE:-$(uname -m) on $(uname -s)}}"

REQUIRE_EPS="$(awk -v eps="$EPS" 'BEGIN { printf "%.0f\n", eps * 0.95 }')"
ERROR_RATE_MAX="0.01"
P99_MAX="500ms"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DATA_DIR="$(mktemp -d -t oblivra-soak-XXXXXX)"
DATE_TAG="$(date -u +%Y-%m-%dT%H%M%SZ)"
RESULTS_DIR="${ROOT}/docs/operator/soak-results"
mkdir -p "${RESULTS_DIR}"
JSON_OUT="${RESULTS_DIR}/${DATE_TAG}.json"
MD_OUT="${RESULTS_DIR}/${DATE_TAG}.md"
SLUG_LATEST="${ROOT}/docs/operator/soak-results-latest.md"

echo "==> building binaries"
go build -trimpath -ldflags "-w -s" -o "${DATA_DIR}/oblivra-server" ./cmd/server
go build -trimpath -ldflags "-w -s" -o "${DATA_DIR}/oblivra-soak"   ./cmd/soak

PORT=18081
SERVER_PID=""
cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    echo "==> stopping server (pid ${SERVER_PID})"
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  rm -rf "${DATA_DIR}"
}
trap cleanup EXIT INT TERM

echo "==> starting server (data dir: ${DATA_DIR})"
OBLIVRA_DATA_DIR="${DATA_DIR}/data" \
OBLIVRA_ADDR="127.0.0.1:${PORT}" \
OBLIVRA_DISABLE_SYSLOG=1 \
OBLIVRA_DISABLE_NETFLOW=1 \
"${DATA_DIR}/oblivra-server" >"${DATA_DIR}/server.log" 2>&1 &
SERVER_PID=$!

echo "==> waiting for /healthz"
for i in {1..30}; do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
  if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
    echo "server died during startup — log tail:"
    tail -n 30 "${DATA_DIR}/server.log" || true
    exit 1
  fi
done

echo "==> running soak: ${EPS} EPS for ${DURATION}"
"${DATA_DIR}/oblivra-soak" \
  --server "http://127.0.0.1:${PORT}" \
  --eps "${EPS}" \
  --duration "${DURATION}" \
  --warmup 5s \
  --report-json "${JSON_OUT}" \
  --report-md   "${MD_OUT}" \
  --require-eps "${REQUIRE_EPS}" \
  --max-error-rate "${ERROR_RATE_MAX}" \
  --max-p99 "${P99_MAX}" \
  --label-hardware "${HARDWARE}" \
  --label-comment "Run via scripts/run-soak.sh; clean-boot server with empty data dir."

EXIT_CODE=$?

cp "${MD_OUT}" "${SLUG_LATEST}"
echo
echo "==> archived:"
echo "    ${MD_OUT}"
echo "    ${JSON_OUT}"
echo "    ${SLUG_LATEST}  (always points at the most recent run)"

if [[ ${EXIT_CODE} -ne 0 ]]; then
  echo "==> FAIL — see ${MD_OUT} for the gate breakdown"
  exit ${EXIT_CODE}
fi
echo "==> PASS"
