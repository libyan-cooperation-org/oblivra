# scripts/run-soak.ps1 — credibility-grade soak runner for Windows.
#
# Boots a clean OBLIVRA server, waits for /healthz, runs oblivra-soak
# at the configured EPS for the configured duration, archives the
# JSON + markdown reports under docs/operator/, then shuts the server
# down cleanly.
#
# Usage:
#   .\scripts\run-soak.ps1 [-Eps 1000] [-Duration 60s] [-Hardware "label"]
#
# Defaults aim at "credibility check on dev hardware".
# For the published 10k-EPS number, run:
#   .\scripts\run-soak.ps1 -Eps 10000 -Duration 5m -Hardware "Win11 / i9-13900K / 64 GiB"

[CmdletBinding()]
param(
    [int]$Eps = 1000,
    [string]$Duration = "60s",
    [string]$Hardware = $env:OBLIVRA_SOAK_HARDWARE
)

$ErrorActionPreference = "Stop"
if (-not $Hardware) { $Hardware = "$env:PROCESSOR_ARCHITECTURE on Windows" }

$RequireEps    = [int]([math]::Floor($Eps * 0.95))
$ErrorRateMax  = "0.01"
$P99Max        = "500ms"

$Root = Split-Path -Parent (Split-Path -Parent $PSCommandPath)
Set-Location $Root

$DataDir = Join-Path ([System.IO.Path]::GetTempPath()) ("oblivra-soak-" + [Guid]::NewGuid().ToString("n").Substring(0, 8))
New-Item -ItemType Directory -Force -Path $DataDir | Out-Null

$DateTag    = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHHmmssZ")
$ResultsDir = Join-Path $Root "docs\operator\soak-results"
New-Item -ItemType Directory -Force -Path $ResultsDir | Out-Null
$JsonOut    = Join-Path $ResultsDir "$DateTag.json"
$MdOut      = Join-Path $ResultsDir "$DateTag.md"
$Latest     = Join-Path $Root "docs\operator\soak-results-latest.md"

$ServerExe = Join-Path $DataDir "oblivra-server.exe"
$SoakExe   = Join-Path $DataDir "oblivra-soak.exe"

Write-Host "==> building binaries"
go build -trimpath -ldflags "-w -s" -o $ServerExe ./cmd/server
go build -trimpath -ldflags "-w -s" -o $SoakExe   ./cmd/soak

$Port = 18081
$ServerLog = Join-Path $DataDir "server.log"

$serverProc = $null
try {
    Write-Host "==> starting server (data dir: $DataDir)"
    $env:OBLIVRA_DATA_DIR        = Join-Path $DataDir "data"
    $env:OBLIVRA_ADDR            = "127.0.0.1:$Port"
    $env:OBLIVRA_DISABLE_SYSLOG  = "1"
    $env:OBLIVRA_DISABLE_NETFLOW = "1"
    $serverProc = Start-Process -FilePath $ServerExe -PassThru -RedirectStandardOutput $ServerLog -RedirectStandardError $ServerLog -NoNewWindow

    Write-Host "==> waiting for /healthz"
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
        try {
            Invoke-RestMethod -Uri "http://127.0.0.1:$Port/healthz" -TimeoutSec 1 | Out-Null
            $ready = $true; break
        } catch {
            Start-Sleep -Milliseconds 500
        }
        if ($serverProc.HasExited) {
            Write-Host "server died during startup — log tail:"
            Get-Content $ServerLog -Tail 30
            exit 1
        }
    }
    if (-not $ready) { throw "server did not become healthy in 15s" }

    Write-Host "==> running soak: $Eps EPS for $Duration"
    $argList = @(
        "--server",          "http://127.0.0.1:$Port",
        "--eps",             "$Eps",
        "--duration",        $Duration,
        "--warmup",          "5s",
        "--report-json",     $JsonOut,
        "--report-md",       $MdOut,
        "--require-eps",     "$RequireEps",
        "--max-error-rate",  $ErrorRateMax,
        "--max-p99",         $P99Max,
        "--label-hardware",  $Hardware,
        "--label-comment",   "Run via scripts/run-soak.ps1; clean-boot server with empty data dir."
    )
    & $SoakExe @argList
    $exitCode = $LASTEXITCODE

    Copy-Item $MdOut $Latest -Force
    Write-Host ""
    Write-Host "==> archived:"
    Write-Host "    $MdOut"
    Write-Host "    $JsonOut"
    Write-Host "    $Latest  (always points at the most recent run)"

    if ($exitCode -ne 0) {
        Write-Host "==> FAIL — see $MdOut for the gate breakdown"
        exit $exitCode
    }
    Write-Host "==> PASS"
}
finally {
    if ($serverProc -and -not $serverProc.HasExited) {
        Write-Host "==> stopping server (pid $($serverProc.Id))"
        Stop-Process -Id $serverProc.Id -Force -ErrorAction SilentlyContinue
    }
    Remove-Item -Recurse -Force $DataDir -ErrorAction SilentlyContinue
}
