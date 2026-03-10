# OBLIVRA Sovereign 72h Soak Test (10,000 EPS)
# This script executes the "Empirical Stability" phase required for DARPA-grade accreditation.
$ErrorActionPreference = "Stop"

$TARGET_EPS = 10000
$DURATION = "72h"
$LOG_SERVER = "soak_72h_server.log"
$LOG_GEN = "soak_72h_generator.log"

Write-Host "====================================================" -ForegroundColor Cyan
Write-Host " OBLIVRA SOVEREIGN 72H SOAK TEST " -ForegroundColor White -BackgroundColor Blue
Write-Host "====================================================" -ForegroundColor Cyan
Write-Host " Target: $TARGET_EPS EPS | Duration: $DURATION " -ForegroundColor Yellow
Write-Host "====================================================" -ForegroundColor Cyan

# 1. Cleanup
Write-Host "=> Cleaning up active sessions..." -ForegroundColor Gray
Stop-Process -Name "server" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "soak_test" -Force -ErrorAction SilentlyContinue

# 2. Build High-Performance Binaries
Write-Host "=> Rebuilding OBLIVRA with batch-optimizations..." -ForegroundColor Cyan
go build -o bin/server.exe ./cmd/server/main.go
go build -o bin/soak_test.exe ./cmd/soak_test/main.go

# 3. Start Server
Write-Host "=> Starting Sovereign Monitor (Headless)..." -ForegroundColor Cyan
$serverProc = Start-Process -FilePath "./bin/server.exe" -RedirectStandardOutput $LOG_SERVER -PassThru -NoNewWindow
Start-Sleep -Seconds 15

# 4. Start Generator
Write-Host "=> Initiating 72-hour adversarial data flood..." -ForegroundColor White -BackgroundColor Red
Start-Process -FilePath "./bin/soak_test.exe" -ArgumentList "-eps $TARGET_EPS -duration $DURATION -workers 12" -RedirectStandardOutput $LOG_GEN -NoNewWindow

Write-Host ""
Write-Host "SUCCESS: Soak test is now running in the background." -ForegroundColor Green
Write-Host "Monitor high-level stats with:" -ForegroundColor Gray
Write-Host "  Get-Content $LOG_SERVER -Tail 20 -Wait" -ForegroundColor White
Write-Host ""
Write-Host "Expectations for 'Unchallengeable' [x] status:" -ForegroundColor Yellow
Write-Host " 1. Zero 'Analytics buffer full' warnings in $LOG_SERVER."
Write-Host " 2. Memory (RSS) growth < 5MB per 24 hours."
Write-Host " 3. Total events processed should reach ~2.6 Billion."
Write-Host ""
Write-Host "To stop the test manually: Stop-Process -Id $($serverProc.Id)" -ForegroundColor Gray
