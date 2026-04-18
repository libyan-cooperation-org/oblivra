$certPath = Join-Path $env:APPDATA 'sovereign-terminal\cert.pem'
$cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2($certPath)
Write-Host "Subject: $($cert.Subject)"
Write-Host "NotAfter: $($cert.NotAfter)"
foreach ($ext in $cert.Extensions) {
    if ($ext.Oid.Value -eq '2.5.29.17') {
        Write-Host "SAN:"
        Write-Host $ext.Format($true)
    }
}
