# Sovereign Baseline Test (30m @ 10,000 EPS)
$ErrorActionPreference = "Stop"

Write-Host "=> Cleaning up old session..." -ForegroundColor Cyan
Stop-Process -Name "main" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "soak_test" -Force -ErrorAction SilentlyContinue

Write-Host "=> Building OBLIVRA Server..." -ForegroundColor Cyan
go build -o bin/server.exe ./cmd/server/main.go

Write-Host "=> Building Soak Test Generator..." -ForegroundColor Cyan
go build -o bin/soak_test.exe ./cmd/soak_test/main.go

Write-Host "=> Starting OBLIVRA Server (Headless)..." -ForegroundColor Cyan
Start-Process -FilePath "./bin/server.exe" -RedirectStandardOutput "soak_baseline_server.txt" -NoNewWindow
Start-Sleep -Seconds 10 # Wait for auto-unlock and warmup

Write-Host "=> Starting 30-minute Sovereign Baseline (10,000 EPS)..." -ForegroundColor Yellow
./bin/soak_test.exe -eps 10000 -duration 30m -workers 8 | Tee-Object -FilePath "soak_baseline_gen.txt"

Write-Host "=> Test Complete. Stopping Server..." -ForegroundColor Cyan
Stop-Process -Name "server" -Force -ErrorAction SilentlyContinue

Write-Host "=> Baseline results captured in 'soak_baseline_server.txt' and 'soak_baseline_gen.txt'." -ForegroundColor Green
