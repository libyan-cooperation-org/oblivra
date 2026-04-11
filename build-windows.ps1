<#
.SYNOPSIS
    OBLIVRA Windows Build Script
    Builds the Wails desktop application for Windows (amd64).

.DESCRIPTION
    Performs a full production build:
      1. Checks all prerequisites (Go, Wails v3, Bun/Node, CGO toolchain)
      2. Installs frontend dependencies
      3. Runs go mod tidy / verify
      4. Executes wails build with correct ldflags
      5. Verifies the output binary exists
      6. Generates SHA-256 checksum
      7. Prints a build summary

.PARAMETER Release
    If set, builds with -trimpath and symbol stripping (-s -w) for a smaller production binary.
    Default is a dev build (faster, includes debug symbols).

.PARAMETER LicensePubKey
    Optional Ed25519 public key (hex) to embed for commercial license enforcement.
    Leave empty for Community/dev builds.

.PARAMETER Clean
    If set, passes -clean to wails build (wipes build/bin before building).

.EXAMPLE
    # Dev build (fastest):
    .\build-windows.ps1

.EXAMPLE
    # Production release build:
    .\build-windows.ps1 -Release

.EXAMPLE
    # Production build with license key:
    .\build-windows.ps1 -Release -LicensePubKey "aabbcc..."

.EXAMPLE
    # Clean release build:
    .\build-windows.ps1 -Release -Clean
#>

param(
    [switch]$Release,
    [string]$LicensePubKey = "",
    [switch]$Clean
)

$ErrorActionPreference = "Stop"
$StartTime = Get-Date

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

function Write-Step([string]$msg) {
    Write-Host "`n[*] $msg" -ForegroundColor Cyan
}

function Write-OK([string]$msg) {
    Write-Host "    [+] $msg" -ForegroundColor Green
}

function Write-Warn([string]$msg) {
    Write-Host "    [!] $msg" -ForegroundColor Yellow
}

function Write-Fail([string]$msg) {
    Write-Host "`n[-] FAILED: $msg" -ForegroundColor Red
}

function Assert-Command([string]$cmd, [string]$hint) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Write-Fail "$cmd not found. $hint"
        exit 1
    }
    Write-OK "$cmd found: $((Get-Command $cmd).Source)"
}

function Get-CommandVersion([string]$cmd, [string[]]$versionArgs) {
    try {
        $out = & $cmd @versionArgs 2>&1 | Select-Object -First 1
        return $out.ToString().Trim()
    } catch {
        return "(version unavailable)"
    }
}

# ─────────────────────────────────────────────────────────────────────────────
# Banner
# ─────────────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════╗" -ForegroundColor DarkCyan
Write-Host "║         OBLIVRA  —  Windows Build Script             ║" -ForegroundColor DarkCyan
Write-Host "║         Wails v3 + SolidJS + Go 1.25                 ║" -ForegroundColor DarkCyan
Write-Host "╚══════════════════════════════════════════════════════╝" -ForegroundColor DarkCyan
Write-Host ""

if ($Release) {
    Write-Host "  Mode   : RELEASE (trimpath, stripped symbols)" -ForegroundColor Yellow
} else {
    Write-Host "  Mode   : DEVELOPMENT (debug symbols included)" -ForegroundColor Green
}
if ($Clean)          { Write-Host "  Clean  : YES (build/bin will be wiped)" -ForegroundColor Yellow }
if ($LicensePubKey)  { Write-Host "  License: EMBEDDED (commercial build)" -ForegroundColor Magenta }
else                 { Write-Host "  License: COMMUNITY (no key injected)" -ForegroundColor Gray }
Write-Host ""

# ─────────────────────────────────────────────────────────────────────────────
# Step 1 — Change to project root
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Changing to project root"
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -Path $ProjectRoot
Write-OK "Working directory: $ProjectRoot"

# ─────────────────────────────────────────────────────────────────────────────
# Step 2 — Prerequisite checks
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Checking prerequisites"

# Go
Assert-Command "go" "Install Go 1.21+ from https://go.dev/dl/"
$goVer = Get-CommandVersion "go" @("version")
Write-OK "Go: $goVer"

# Wails v3 (binary name is 'wails3' in v3 alpha or 'wails' depending on install)
$wailsCmd = $null
foreach ($candidate in @("wails3", "wails")) {
    if (Get-Command $candidate -ErrorAction SilentlyContinue) {
        $wailsCmd = $candidate
        break
    }
}
if (-not $wailsCmd) {
    Write-Fail "Neither 'wails3' nor 'wails' found in PATH.`n  Install Wails v3: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
    exit 1
}
$wailsVer = Get-CommandVersion $wailsCmd @("version")
Write-OK "Wails CLI: $wailsCmd — $wailsVer"

# Bun (preferred) or Node/npm as fallback
$frontendInstaller = $null
$frontendInstallCmd = $null
$frontendBuildCmd = $null
if (Get-Command "bun" -ErrorAction SilentlyContinue) {
    $frontendInstaller = "bun"
    $frontendInstallCmd = @("install")
    $frontendBuildCmd   = @("run", "build")
    Write-OK "Frontend bundler: bun $(Get-CommandVersion 'bun' @('-v'))"
} elseif (Get-Command "node" -ErrorAction SilentlyContinue) {
    $frontendInstaller = "npm"
    $frontendInstallCmd = @("install")
    $frontendBuildCmd   = @("run", "build")
    Write-OK "Frontend bundler: npm $(Get-CommandVersion 'npm' @('-v')) (bun not found, using npm)"
} else {
    Write-Fail "Neither bun nor node/npm found in PATH.`n  Install bun: https://bun.sh  OR  Node.js: https://nodejs.org"
    exit 1
}

# GCC (required for CGO — sqlite3, sqlcipher)
if (Get-Command "gcc" -ErrorAction SilentlyContinue) {
    Write-OK "GCC: $(Get-CommandVersion 'gcc' @('--version'))"
} else {
    Write-Warn "GCC not found in PATH."
    Write-Warn "The project requires CGO (go-sqlite3, go-sqlcipher)."
    Write-Warn "Install MSYS2 + MinGW-w64: https://www.msys2.org"
    Write-Warn "Then run: pacman -S mingw-w64-x86_64-gcc"
    Write-Warn "And add C:\msys64\mingw64\bin to your PATH."
    Write-Warn "Continuing anyway — build will fail if CGO is unavailable."
}

# ─────────────────────────────────────────────────────────────────────────────
# Step 3 — Frontend dependencies
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Installing frontend dependencies (frontend/)"
Push-Location "$ProjectRoot\frontend"
try {
    & $frontendInstaller @frontendInstallCmd
    if ($LASTEXITCODE -ne 0) { throw "Frontend install failed" }
    Write-OK "frontend/ dependencies installed"
} finally {
    Pop-Location
}

# ─────────────────────────────────────────────────────────────────────────────
# Step 4 — Go module hygiene
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Running go mod tidy"
& go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Fail "go mod tidy failed"
    exit 1
}
Write-OK "go.mod / go.sum are clean"

Write-Step "Running go mod verify"
& go mod verify
if ($LASTEXITCODE -ne 0) {
    Write-Fail "go mod verify failed — dependency checksums do not match"
    exit 1
}
Write-OK "Module checksums verified"

# ─────────────────────────────────────────────────────────────────────────────
# Step 5 — Construct ldflags
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Constructing build flags"

# Get git short hash (non-fatal if git unavailable)
$gitCommit = "unknown"
try {
    $gitCommit = (git rev-parse --short HEAD 2>$null).Trim()
    if (-not $gitCommit) { $gitCommit = "unknown" }
} catch {}

$Version = "1.0.0"
$Module  = "github.com/kingknull/oblivrashell"

# Base ldflags: always inject version + commit
$ldflags = "-X ${Module}/internal/attestation.BuildVersion=${Version} -X ${Module}/internal/attestation.BuildCommit=${gitCommit}"

# Inject license public key if provided
if ($LicensePubKey) {
    $ldflags += " -X ${Module}/internal/core.licensePubKey=${LicensePubKey}"
    Write-OK "License public key will be embedded"
}

# Release mode: strip debug info for smaller binary
if ($Release) {
    $ldflags = "-s -w $ldflags"
    Write-OK "Release flags: -s -w (symbol stripping + DWARF removal)"
}

Write-OK "ldflags: $ldflags"
Write-OK "Version: $Version  Commit: $gitCommit"

# ─────────────────────────────────────────────────────────────────────────────
# Step 6 — Wails build
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Building OBLIVRA Windows binary"

$wailsArgs = @("-ldflags", $ldflags)

if ($Clean)   { $wailsArgs += "-clean" }
if ($Release) { $wailsArgs += "-trimpath" }

Write-Host "    Running: $wailsCmd build $($wailsArgs -join ' ')" -ForegroundColor DarkGray
Write-Host ""

& $wailsCmd build @wailsArgs

if ($LASTEXITCODE -ne 0) {
    Write-Fail "Wails build failed (exit code $LASTEXITCODE)"
    Write-Host ""
    Write-Host "  Common fixes:" -ForegroundColor Yellow
    Write-Host "  - CGO error       → install GCC via MSYS2 (see Step 2 warning above)"
    Write-Host "  - Module error    → run: go mod tidy && go mod verify"
    Write-Host "  - Frontend error  → run: cd frontend && bun install && bun run build"
    Write-Host "  - Wails not found → run: go install github.com/wailsapp/wails/v3/cmd/wails3@latest"
    Write-Host "  - Port conflict   → close any running dev server on port 9245"
    exit 1
}

# ─────────────────────────────────────────────────────────────────────────────
# Step 7 — Verify output
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Verifying build output"

$BinPath = "$ProjectRoot\build\bin\oblivrashell.exe"

if (-not (Test-Path $BinPath)) {
    # Wails sometimes uses the app name from wails.json
    $AltPath = "$ProjectRoot\build\bin\OblivraShell.exe"
    if (Test-Path $AltPath) {
        $BinPath = $AltPath
    } else {
        Write-Fail "Binary not found at expected locations:`n  $BinPath`n  $AltPath"
        Write-Host "`n  Contents of build/bin/:" -ForegroundColor Yellow
        Get-ChildItem "$ProjectRoot\build\bin" -ErrorAction SilentlyContinue | ForEach-Object {
            Write-Host "    $_"
        }
        exit 1
    }
}

$BinSize = (Get-Item $BinPath).Length
$BinSizeMB = [math]::Round($BinSize / 1MB, 1)
Write-OK "Binary found: $BinPath"
Write-OK "Binary size : $BinSizeMB MB ($BinSize bytes)"

# ─────────────────────────────────────────────────────────────────────────────
# Step 8 — SHA-256 checksum
# ─────────────────────────────────────────────────────────────────────────────

Write-Step "Generating SHA-256 checksum"

$Hash        = (Get-FileHash $BinPath -Algorithm SHA256).Hash.ToLower()
$HashFile    = "$BinPath.sha256"
$HashContent = "$Hash  oblivrashell.exe"

$HashContent | Out-File -FilePath $HashFile -Encoding ascii -NoNewline
Write-OK "SHA-256 : $Hash"
Write-OK "Saved to: $HashFile"

# ─────────────────────────────────────────────────────────────────────────────
# Step 9 — Build summary
# ─────────────────────────────────────────────────────────────────────────────

$Elapsed = [math]::Round(((Get-Date) - $StartTime).TotalSeconds, 1)

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║                  BUILD SUCCESSFUL                   ║" -ForegroundColor Green
Write-Host "╚══════════════════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "  Binary  : $BinPath" -ForegroundColor White
Write-Host "  Size    : $BinSizeMB MB" -ForegroundColor White
Write-Host "  SHA-256 : $Hash" -ForegroundColor White
Write-Host "  Version : $Version  ($gitCommit)" -ForegroundColor White
Write-Host "  Mode    : $(if ($Release) { 'RELEASE' } else { 'DEVELOPMENT' })" -ForegroundColor White
Write-Host "  Time    : ${Elapsed}s" -ForegroundColor White
Write-Host ""
Write-Host "  Run it  : .\build\bin\oblivrashell.exe" -ForegroundColor Cyan
Write-Host ""
