# GateKeeper-L Enterprise Installer (PRODUCTION HARDENED)
# Run as Administrator / SYSTEM (SCCM/Intune)

$ServiceName = "GateKeeper-L"
$ExeName = "gatekeeper-l.exe"
$InstallDir = "C:\Program Files\GateKeeper-L"
$SourceExe = "$PSScriptRoot\$ExeName"

Write-Host "Starting GateKeeper-L Production Installation..." -ForegroundColor Cyan

# 1. Cleanup: Try to stop/delete old versions (Best Effort)
try {
    sc.exe stop $ServiceName
    Start-Sleep -Seconds 2
    sc.exe delete $ServiceName
} catch {
    Write-Host "Note: Clean install or previous version locked." -ForegroundColor Yellow
}

# 2. Files: Create Directory & Copy
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}
Copy-Item -Path $SourceExe -Destination "$InstallDir\$ExeName" -Force

# 3. Create Service
$BinPath = "$InstallDir\$ExeName"
sc.exe create $ServiceName binPath= $BinPath start= auto DisplayName= "GateKeeper L (Pre-Login Only)"

# 4. Recovery Logic (Restart on crash)
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000

# 5. START SERVICE (לפני ההקשחה!)
Write-Host "Starting Service..."
sc.exe start $ServiceName

# תן לו רגע לעלות
Start-Sleep -Seconds 3

# 6. HARDENING (The Lock Down) 🔒
$HardenedSDDL = "D:(A;;GA;;;SY)(A;;CCLCSWLOCRRC;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"

Write-Host "Applying Security Hardening..."
$result = sc.exe sdset $ServiceName $HardenedSDDL

if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: GateKeeper-L is Installed, Running, and Locked." -ForegroundColor Green
} else {
    Write-Host "WARNING: Service installed but Hardening failed." -ForegroundColor Red
}