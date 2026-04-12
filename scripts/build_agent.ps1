$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path "$PSScriptRoot/.."
$Version = "1.0.0-sovereign"
$BuildTime = (Get-Date -Format "yyyy-MM-dd")
$BinDir = Join-Path $ProjectRoot "bin"

if (!(Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir
}

Write-Host "Building OBLIVRA Agents..." -ForegroundColor Cyan

# Windows Build
Write-Host "  -> Building Windows Agent (AMD64)..." -ForegroundColor Yellow
$Env:GOOS = "windows"
$Env:GOARCH = "amd64"
$Env:CGO_ENABLED = 0
go build -trimpath -ldflags="-s -w -X main.version=$Version -X main.buildTime=$BuildTime" -o "$BinDir/oblivra-agent.exe" "$ProjectRoot/cmd/agent"

# Linux Build
Write-Host "  -> Building Linux Agent (AMD64)..." -ForegroundColor Yellow
$Env:GOOS = "linux"
$Env:GOARCH = "amd64"
$Env:CGO_ENABLED = 0
go build -trimpath -ldflags="-s -w -X main.version=$Version -X main.buildTime=$BuildTime" -o "$BinDir/oblivra-agent" "$ProjectRoot/cmd/agent"

# Cleanup
Remove-Item Env:\GOOS
Remove-Item Env:\GOARCH
Remove-Item Env:\CGO_ENABLED

Write-Host "`nBuild Complete!" -ForegroundColor Green
Write-Host "Binaries located in ./$BinDir"
