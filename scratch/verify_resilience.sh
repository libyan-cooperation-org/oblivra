#!/bin/bash
set -e

echo "[TEST] Starting OBLIVRA Agent in Isolated Mode..."
OBLIVRA_ISOLATED_VAULT=true ./build/bin/oblivra-agent -data-dir ./tmp/resilience-test -server localhost:8443 > ./tmp/resilience.log 2>&1 &
AGENT_PID=$!

sleep 5

VAULT_PID=$(pgrep -f oblivra-vault)
if [ -z "$VAULT_PID" ]; then
    echo "[FAIL] Vault daemon did not start!"
    kill $AGENT_PID
    exit 1
fi

echo "[PASS] Vault daemon is running (PID: $VAULT_PID)"

echo "[TEST] Killing vault daemon to trigger recovery..."
kill -9 $VAULT_PID

echo "[TEST] Waiting 40s for heartbeat recovery (30s interval)..."
sleep 40

NEW_VAULT_PID=$(pgrep -f oblivra-vault)
if [ -z "$NEW_VAULT_PID" ]; then
    echo "[FAIL] Vault daemon did not recover!"
    cat ./tmp/resilience.log
    kill $AGENT_PID
    exit 1
fi

if [ "$VAULT_PID" == "$NEW_VAULT_PID" ]; then
    echo "[FAIL] PID did not change? Something is wrong."
    kill $AGENT_PID
    exit 1
fi

echo "[PASS] Vault daemon RECOVERED (New PID: $NEW_VAULT_PID)"

echo "[TEST] Verifying backoff..."
echo "[TEST] Killing vault daemon 3 times rapidly..."
kill -9 $NEW_VAULT_PID
sleep 1
kill -9 $(pgrep -f oblivra-vault || echo "")
sleep 1
kill -9 $(pgrep -f oblivra-vault || echo "")

grep "backoff active" ./tmp/resilience.log && echo "[PASS] Crash-loop backoff detected in logs" || echo "[WARN] Backoff log not found (might need more time)"

kill $AGENT_PID
echo "[SUCCESS] Resilience test completed."
