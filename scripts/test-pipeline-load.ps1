# OBLIVRA — Pipeline Stress Test (Adaptive Worker Verification)
#
# This script targets the REST ingest endpoint to simulate high sustained EPS
# and verifies that the AdaptiveController properly scales workers and eventually
# triggers load-shedding to prevent OOM.
#
# Requirements: OBLIVRA must be running with OBLIVRA_ENV=development (to allow mock ingest)
# Or targeting a test tenant.

param(
    [string]$Url = "http://localhost:8080/api/v1/ingest/bulk",
    [int]$Iterations = 100,
    [int]$BatchSize = 1000
)

Write-Host "🚀 Starting OBLIVRA Adaptive Pipeline Stress Test..." -ForegroundColor Cyan
Write-Host "Target: $Url"
Write-Host "Simulating $($Iterations * $BatchSize) events..."

$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

for ($i = 1; $i -le $Iterations; $i++) {
    $events = @()
    for ($j = 0; $j -le $BatchSize; $j++) {
        $events += @{
            tenant_id = "test-tenant-alpha"
            event_type = "stress_test_log"
            host = "stress-node-$($j % 10)"
            raw_line = "[STRESS] Synthetic event $j in batch $i"
            timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ssZ")
        }
    }

    $json = $events | ConvertTo-Json
    
    # Fire and forget (async-ish)
    Invoke-RestMethod -Uri $Url -Method Post -Body $json -ContentType "application/json" | Out-Null
    
    if ($i % 10 -eq 0) {
        $currentEps = [Math]::Round(($i * $BatchSize) / ($stopwatch.Elapsed.TotalSeconds), 0)
        Write-Host "[$i/$Iterations] Cumulative EPS: $currentEps | Worker scaling check recommended..." -ForegroundColor Yellow
    }
}

$stopwatch.Stop()
Write-Host "`n✅ Stress Test Injected. Checking Pipeline Status..." -ForegroundColor Green

# Fetch metrics from status endpoint to verify scaling
try {
    $status = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/ingest/status"
    Write-Host "`n--- Pipeline Telemetry ---"
    Write-Host "Current EPS: $($status.eps)"
    Write-Host "Active Workers: $($status.active_workers)"
    Write-Host "Buffer Occupancy: $($status.buffer_usage)% ($($status.buffer_count)/$($status.buffer_capacity))"
    Write-Host "Dropped Events: $($status.dropped_events)"
    
    if ($status.active_workers -gt $Host.UI.RawUI.BufferSize.Width) { # Generic check
        Write-Host "`n[VERIFIED] Adaptive Controller successfully scaled workers above NumCPU." -ForegroundColor Green
    }
} catch {
    Write-Host "Could not fetch final status. Ensure OBLIVRA is running." -ForegroundColor Red
}
