<#
.SYNOPSIS
    Builds the OBLIVRA agent binary for Windows (amd64).
    Output: bin\oblivra-agent.exe

.DESCRIPTION
    Compiles cmd/agent/main.go with version metadata injected via ldflags.
    Produces a standalone binary that can be deployed to endpoints without
    any runtime dependencies.

.PARAMETER Release
    Strip debug symbols (-s -w) for a smaller production binary.

.PARAMETER Version
    Version string to embed (default: "dev").

.EXAMPLE
    # Quick dev build:
    .\build-agent.ps1

.EXAMPLE
    # Production release build:
    .\build-agent.ps1 -Release -Version "1.0.0"
#>

param(
    [switch]$Release,
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$StartTime = Get-Date
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

# ── Helpers ──────────────────────────────────────────────────────────────────

function Write-Step([string]$msg)  { Write-Host "`n[*] $msg" -ForegroundColor Cyan }
function Write-OK([string]$msg)    { Write-Host "    [+] $msg" -ForegroundColor Green }
function Write-Fail([string]$msg)  { Write-Host "`n[-] FAILED: $msg" -ForegroundColor Red; exit 1 }

# ── Banner ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔══════════════════════════════════════════════╗" -ForegroundColor DarkCyan
Write-Host "║    OBLIVRA Agent  —  Windows Build Script    ║" -ForegroundColor DarkCyan
Write-Host "╚══════════════════════════════════════════════╝" -ForegroundColor DarkCyan
Write-Host ""
Write-Host "  Target  : bin\oblivra-agent.exe" -ForegroundColor White
Write-Host "  Version : $Version" -ForegroundColor White
Write-Host "  Mode    : $(if ($Release) { 'RELEASE (stripped)' } else { 'DEVELOPMENT' })" -ForegroundColor White
Write-Host ""

# ── Prerequisites ─────────────────────────────────────────────────────────────

Write-Step "Checking prerequisites"

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Fail "go not found. Install from https://go.dev/dl/"
}
Write-OK "Go: $((go version 2>&1 | Select-Object -First 1).ToString().Trim())"

# ── Get git commit ─────────────────────────────────────────────────────────────

$BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
$GitCommit = "unknown"
try {
    $GitCommit = (git rev-parse --short HEAD 2>$null).Trim()
    if (-not $GitCommit) { $GitCommit = "unknown" }
} catch {}

Write-OK "Version : $Version  Commit : $GitCommit  Built : $BuildTime"

# ── Build ─────────────────────────────────────────────────────────────────────

Write-Step "Compiling agent"

$Module    = "github.com/kingknull/oblivrashell/cmd/agent"
$OutPath   = ".\bin\oblivra-agent.exe"
$LDFlags   = "-X main.version=$Version -X main.buildTime=$BuildTime"

if ($Release) {
    $LDFlags = "-s -w $LDFlags"
}

Write-Host "    Running: go build -ldflags `"$LDFlags`" -o $OutPath ./cmd/agent" -ForegroundColor DarkGray

$env:GOOS   = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

& go build -ldflags $LDFlags -o $OutPath ./cmd/agent

if ($LASTEXITCODE -ne 0) {
    Write-Fail "go build failed (exit $LASTEXITCODE)"
}

# ── Verify + checksum ─────────────────────────────────────────────────────────

Write-Step "Verifying output"

if (-not (Test-Path $OutPath)) {
    Write-Fail "Binary not found at $OutPath"
}

$Size    = (Get-Item $OutPath).Length
$SizeMB  = [math]::Round($Size / 1MB, 1)
$Hash    = (Get-FileHash $OutPath -Algorithm SHA256).Hash.ToLower()
$HashFile = "$OutPath.sha256"
"$Hash  oblivra-agent.exe" | Out-File -FilePath $HashFile -Encoding ascii -NoNewline

Write-OK "Binary : $OutPath ($SizeMB MB)"
Write-OK "SHA-256: $Hash"

# ── Summary ───────────────────────────────────────────────────────────────────

$Elapsed = [math]::Round(((Get-Date) - $StartTime).TotalSeconds, 1)

Write-Host ""
Write-Host "╔══════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║              BUILD SUCCESSFUL                ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "  Binary  : $OutPath" -ForegroundColor White
Write-Host "  Size    : $SizeMB MB" -ForegroundColor White
Write-Host "  SHA-256 : $Hash" -ForegroundColor White
Write-Host "  Time    : ${Elapsed}s" -ForegroundColor White
Write-Host ""
Write-Host "  Usage:" -ForegroundColor Cyan
Write-Host "    # Basic (connects to local server):" -ForegroundColor DarkGray
Write-Host "    .\bin\oblivra-agent.exe" -ForegroundColor White
Write-Host ""
Write-Host "    # With TLS CA verification:" -ForegroundColor DarkGray
Write-Host "    .\bin\oblivra-agent.exe -tls-ca `"C:\Users\Maverick\AppData\Roaming\sovereign-terminal\ca.pem`"" -ForegroundColor White
Write-Host ""
Write-Host "    # Full production deployment:" -ForegroundColor DarkGray
Write-Host "    .\bin\oblivra-agent.exe -server `"yourserver:8443`" -tls-ca ca.pem -fim -eventlog -version" -ForegroundColor White
Write-Host ""
