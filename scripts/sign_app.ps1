$AppDir = Join-Path $env:USERPROFILE ".oblivrashell"
$CaCertPath = Join-Path $AppDir "ca.pem"
$PfxPath = Join-Path $AppDir "codesign.pfx"
$ExePath = "oblivrashell.exe"

if (-not (Test-Path $CaCertPath)) {
    Write-Host "CA certificate not found. Run certgen utility first." -ForegroundColor Red
    exit 1
}

# 1. Install CA to Trusted Root (Requires Admin)
Write-Host "Installing CA to Trusted Root Certification Authorities..." -ForegroundColor Cyan
try {
    Import-Certificate -FilePath $CaCertPath -CertStoreLocation Cert:\LocalMachine\Root -ErrorAction Stop
    Write-Host "CA installed successfully." -ForegroundColor Green
}
catch {
    Write-Host "Failed to install CA. Please run this script as Administrator." -ForegroundColor Yellow
}

# 2. Convert Code Signing Pem to PFX using .NET
if (Test-Path -Path (Join-Path $AppDir "codesign.pem")) {
    Write-Host "Converting PEM to PFX using .NET..." -ForegroundColor Cyan
    $certPem = Get-Content (Join-Path $AppDir "codesign.pem") -Raw
    $keyPem = Get-Content (Join-Path $AppDir "codesign.key") -Raw
    
    # Use .NET to create the certificate object
    $cert = [System.Security.Cryptography.X509Certificates.X509Certificate2]::CreateFromPem($certPem, $keyPem)
    
    # Export to PFX (PKCS12)
    $pfxBytes = $cert.Export([System.Security.Cryptography.X509Certificates.X509ContentType]::Pkcs12, "")
    [System.IO.File]::WriteAllBytes($PfxPath, $pfxBytes)
    Write-Host "PFX created at $PfxPath" -ForegroundColor Green
}

# 3. Sign the application
if (Test-Path $ExePath) {
    if (Test-Path $PfxPath) {
        Write-Host "Signing $ExePath with $PfxPath..." -ForegroundColor Cyan
        $signingCert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2($PfxPath, "")
        Set-AuthenticodeSignature -FilePath $ExePath -Certificate $signingCert
        Write-Host "Signing complete." -ForegroundColor Green
    }
    else {
        Write-Host "PFX file not found. Conversion might have failed." -ForegroundColor Red
    }
}
else {
    Write-Host "oblivrashell.exe not found in current directory." -ForegroundColor Yellow
}
