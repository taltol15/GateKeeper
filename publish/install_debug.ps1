# GateKeeper Installer (DEBUG VERSION - Admin Access Allowed)
# Run as Administrator

$ServiceName = "GateKeeper"
$ExeName = "gatekeeper.exe"
$InstallDir = "C:\Program Files\GateKeeper"
$SourceExe = "$PSScriptRoot\$ExeName"

Write-Host "Starting GateKeeper Installation (Debug Mode)..." -ForegroundColor Cyan

# 1. Stop and Clean previous versions
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
    Write-Host "Stopping existing service..."
    sc.exe stop $ServiceName
    Start-Sleep -Seconds 2
    sc.exe delete $ServiceName
    Start-Sleep -Seconds 2
}

# 2. Create Directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

# 3. Copy Executable
Copy-Item -Path $SourceExe -Destination "$InstallDir\$ExeName" -Force

# 4. Create Service
$BinPath = "$InstallDir\$ExeName"
sc.exe create $ServiceName binPath= $BinPath start= auto DisplayName= "GateKeeper Endpoint Protection"

# 5. HARDENING (MILD VERSION FOR DEBUGGING)
# כאן השינוי: נתנו ל-BA (אדמינים) הרשאות מלאות (GA = Generic All)
# SYSTEM (SY) = Full Access
# Admins (BA) = Full Access (כדי שתוכל לדבג)
$SDDL = "D:(A;;GA;;;SY)(A;;GA;;;BA)"
sc.exe sdset $ServiceName $SDDL

# 6. Recovery Logic
sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000

# 7. Start
sc.exe start $ServiceName

Write-Host "Installation Complete. Debug Access Enabled." -ForegroundColor Green