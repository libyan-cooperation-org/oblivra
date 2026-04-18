#!/bin/bash
set -e

echo "=========================================="
echo " OblivraShell Headless API Smoke Test"
echo "=========================================="

HOST=${1:-"https://localhost:8443"}
API_KEY=${API_KEY:-"test-key-123"} # Change this depending on deployed config

echo "Targeting Host: $HOST"
echo ""

# Wait for server to be responsive
echo "0. Waiting for Readiness Probe..."
for i in {1..5}; do
  if curl -k -s -f "$HOST/readyz" > /dev/null; then
    echo "   ✅ /readyz is UP"
    break
  fi
  echo "   ... waiting for server..."
  sleep 2
done

echo ""
echo "1. Testing Metrics Endpoint..."
if curl -k -s -f "$HOST/metrics" | grep "siem_" > /dev/null; then
    echo "   ✅ /metrics OK (Prometheus data found)"
else
    echo "   ❌ /metrics Failed"
    exit 1
fi

echo ""
echo "2. Testing Alerts Endpoint..."
STATUS=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$HOST/api/v1/alerts")
if [ "$STATUS" -eq 200 ]; then
    echo "   ✅ /api/v1/alerts OK"
else
    echo "   ❌ /api/v1/alerts Failed with status $STATUS"
    exit 1
fi

echo ""
echo "3. Testing SIEM Search Endpoint..."
STATUS=$(curl -k -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $API_KEY" "$HOST/api/v1/siem/search?q=failed")
if [ "$STATUS" -eq 200 ]; then
    echo "   ✅ /api/v1/siem/search OK"
else
    echo "   ❌ /api/v1/siem/search Failed with status $STATUS"
    exit 1
fi

echo ""
echo "=========================================="
echo " ✅ ALL SMOKE TESTS PASSED"
echo "=========================================="
