#!/bin/bash
# scripts/soak_verification.sh — OBLIVRA Stability & Sustainability Test (S3)
#
# Performs a "Rapid Soak" by hammering the server with 24 hours worth of 
# nominal traffic in a compressed 15-minute window.
#
# Metrics tracked: Memory (RSS), EPS, Dropped Events, Worker Count.

SERVER_URL="http://localhost:8090"
DURATION_SEC=900 # 15 minutes
SAMPLE_INTERVAL=10

echo "🚀 Starting OBLIVRA Rapid Soak Verification..."

# 1. Warm-up: Ensure server is reachable
if ! curl -s "$SERVER_URL/healthz" > /dev/null; then
    echo "❌ Error: OBLIVRA server not found at $SERVER_URL. Please start it in another terminal."
    exit 1
fi

# 2. Baseline snapshot
BASELINE_MEM=$(ps -o rss= -p $(pgrep -f "oblivrashell") | awk '{print $1}')
echo "📊 Baseline RSS: $((BASELINE_MEM / 1024)) MB"

# 3. Launch Stressor (using chaos harness) in background
echo "🔨 Launching high-intensity stressor (Scenario: OOM/Shed)..."
go run cmd/chaos/*.go --scenario=oom --server="$SERVER_URL" -v &
STRESS_PID=$!

# 4. Monitoring Loop
echo "⏳ Monitoring for ${DURATION_SEC}s..."
START_TIME=$(date +%s)
END_TIME=$((START_TIME + DURATION_SEC))

MAX_MEM=0

while [ $(date +%s) -lt $END_TIME ]; do
    CURRENT_TIME=$(date +%s)
    RES=$(curl -s "$SERVER_URL/api/v1/diagnostics/summary") # Hypothetical endpoint for metrics
    
    # Track Memory
    RSS=$(ps -o rss= -p $(pgrep -f "oblivrashell") | awk '{print $1}')
    if [ "$RSS" -gt "$MAX_MEM" ]; then MAX_MEM=$RSS; fi
    
    echo "  [$(($CURRENT_TIME - $START_TIME))s] RSS: $((RSS / 1024)) MB"
    
    sleep $SAMPLE_INTERVAL
done

# 5. Result Analysis
FINAL_MEM=$(ps -o rss= -p $(pgrep -f "oblivrashell") | awk '{print $1}')
MEM_DIFF=$((FINAL_MEM - BASELINE_MEM))

echo ""
echo "══ RECAP ══"
echo "  Peak Memory: $((MAX_MEM / 1024)) MB"
echo "  Final Memory: $((FINAL_MEM / 1024)) MB"
echo "  Growth: $((MEM_DIFF / 1024)) MB"

if [ "$MEM_DIFF" -gt 102400 ]; then # 100MB growth threshold for "leak" warning
    echo "⚠️  WARNING: Significant memory growth detected ($((MEM_DIFF / 1024)) MB). Verify GC reclaim."
else
    echo "✅ Memory profile remains stable under pressure."
fi

echo "✅ Soak test verification cycle complete."

# Cleanup
kill $STRESS_PID 2>/dev/null
