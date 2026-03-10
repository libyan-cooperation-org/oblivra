$ErrorActionPreference = "Stop"

$Version = "1.0.0"
$BuildDir = "build/bin"
$AppName = "oblivrashell"

Write-Host "OblivraShell Build Script" -ForegroundColor Cyan
Write-Host "===============================" -ForegroundColor Cyan
Write-Host "Version: $Version"

# Clean
Write-Host "`n[1/5] Cleaning build directory..." -ForegroundColor Yellow
if (Test-Path $BuildDir) {
    Remove-Item -Recurse -Force $BuildDir
}
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

# Frontend Build
Write-Host "`n[2/5] Building Frontend (SolidJS)..." -ForegroundColor Yellow
Set-Location frontend
bun install
bun run build
Set-Location ..

# Wails Build
Write-Host "`n[3/5] Building Wails Application..." -ForegroundColor Yellow
wails build -clean -upx -trimpath -ldflags="-s -w -X main.version=$Version" 

# CLI Build
Write-Host "`n[4/5] Building Headless CLI..." -ForegroundColor Yellow
$Env:CGO_ENABLED = 0
go build -trimpath -ldflags="-s -w -X main.version=$Version" -o "$BuildDir/${AppName}-cli.exe" ./cmd/cli
Remove-Item Env:\CGO_ENABLED

# Copy Assets
Write-Host "`n[5/5] Copying binaries to build/bin..." -ForegroundColor Yellow
Copy-Item "build/bin/oblivrashell.exe" "$BuildDir/oblivrashell-desktop.exe" -ErrorAction SilentlyContinue

Write-Host "`nBuild Complete!" -ForegroundColor Green
Write-Host "Binaries located in ./$BuildDir"
