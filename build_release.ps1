# OblivraShell v1.0 Release Build Script
# This script builds a hardened, compressed executable using UPX to shrink the binary for USB/Air-Gap deployments.

Write-Host "[*] Beginning OblivraShell Release Build (v1.0.0)..." -ForegroundColor Cyan

# Ensure we are in the project root
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -Path $ProjectRoot

# Run Wails build with release flags
# -clean: wipes the `build/bin` directory
# -m: obfuscates binary symbols for reverse-engineering protection
# -upx: compresses the executable with Ultimate Packer for eXecutables
# -obfuscated: Disabled due to Go 1.26 incompatibility with garble linker patches
# -v 2: sets verbose output mode
Write-Host "[*] Executing Wails Compilation Process with Standard Minification..." -ForegroundColor Yellow
$WailsCommand = "wails build -clean -m -upx -v 2"
Invoke-Expression $WailsCommand

if ($LASTEXITCODE -eq 0) {
    Write-Host "[+] Build Successful! Binaries are located in build/bin/oblivrashell.exe" -ForegroundColor Green
} else {
    Write-Host "[-] Build Failed!" -ForegroundColor Red
    exit 1
}
