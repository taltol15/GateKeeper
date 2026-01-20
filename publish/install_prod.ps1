# GateKeeper Enterprise Installer (PRODUCTION HARDENED)
# Run as Administrator / SYSTEM (SCCM/Intune)

$ServiceName = "GateKeeper"
$ExeName = "gatekeeper.exe"
$InstallDir = "C:\Program Files\GateKeeper"
$SourceExe = "$PSScriptRoot\$ExeName"

Write-Host "Starting GateKeeper Production Installation..." -ForegroundColor Cyan

# 1. Cleanup: Try to stop/delete old versions (Best Effort)
# אם הגרסה הקודמת הייתה מוקשחת, הפקודות האלו ייכשלו וזה בסדר (הרישום מחדש יתקן)
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
sc.exe create $ServiceName binPath= $BinPath start= auto DisplayName= "GateKeeper Endpoint Protection"

# 4. Recovery Logic (Restart on crash)
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000

# 5. START SERVICE (לפני ההקשחה!)
# אנחנו מפעילים אותו כשהוא עדיין "פתוח" כדי לוודא שהוא עולה
Write-Host "Starting Service..."
sc.exe start $ServiceName

# תן לו רגע לעלות
Start-Sleep -Seconds 3

# 6. HARDENING (The Lock Down) 🔒
# D: = DACL
# (A;;GA;;;SY) = SYSTEM gets Full Control (Generic All).
# (A;;CCLCSWLOCRRC;;;BA) = Admins get Query/Read ONLY. 
# משמעות ל-BA: מותר לראות סטטוס, אסור לעצור (RP/WP) ואסור למחוק (SD).
$HardenedSDDL = "D:(A;;GA;;;SY)(A;;CCLCSWLOCRRC;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"

Write-Host "Applying Security Hardening..."
$result = sc.exe sdset $ServiceName $HardenedSDDL

if ($LASTEXITCODE -eq 0) {
    Write-Host "SUCCESS: GateKeeper is Installed, Running, and Locked." -ForegroundColor Green
} else {
    Write-Host "WARNING: Service installed but Hardening failed." -ForegroundColor Red
}