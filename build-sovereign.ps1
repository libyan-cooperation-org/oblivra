<#
.SYNOPSIS
Builds the Sovereign (Air-Gapped) release of OBLIVRA.

.DESCRIPTION
Compiles the Wails application with reproducible build flags, trimming paths and enforcing build IDs
so that byte-for-byte binary attestation can be verified by government/enterprise contractors.
Generates an SHA256 checksum for the resulting executable.
#>

$ErrorActionPreference = "Stop"

$Version = "1.0.0"
$Commit = git rev-parse --short HEAD
if (-not $Commit) {
    $Commit = "unknown"
}

Write-Host "[*] Starting Reproducible Sovereign Build for v$Version (Commit: $Commit)..." -ForegroundColor Cyan

# Flags critical for reproducible builds (remove paths, static build ID)
$LDFlags = "-s -w -X main.version=$Version -X main.commit=$Commit"

Write-Host "[*] Executing wails build with strict attestation flags..."
# Provide the explicit ldflags and trimpath to normalize the binary outputs
wails build -clean -trimpath -ldflags $LDFlags

$BinPath = ".\build\bin\oblivrashell.exe"

if (Test-Path $BinPath) {
    Write-Host "[+] Build successful. Attesting binary..." -ForegroundColor Green
    $Hash = (Get-FileHash $BinPath -Algorithm SHA256).Hash.ToLower()
    $HashFilePath = ".\build\bin\oblivrashell.exe.sha256"
    
    $Hash | Out-File -FilePath $HashFilePath -Encoding ascii
    Write-Host "[+] SHA-256 Checksum generated: $Hash" -ForegroundColor Yellow
    Write-Host "[+] Artifacts are ready in build/bin/ for signing and deployment." -ForegroundColor Green
} else {
    Write-Host "[-] Build failed or executable not found at $BinPath" -ForegroundColor Red
    exit 1
}
