# GateKeeper Installer & Hardening Script
# Run as Administrator / SYSTEM

$ServiceName = "GateKeeper"
$ExeName = "gatekeeper.exe"
$InstallDir = "C:\Program Files\GateKeeper"
$SourceExe = "$PSScriptRoot\$ExeName"

Write-Host "Starting GateKeeper Installation..." -ForegroundColor Cyan

# 1. Stop and Clean previous versions
if (Get-Service $ServiceName -ErrorAction SilentlyContinue) {
    Write-Host "Stopping existing service..."
    sc.exe stop $ServiceName
    Start-Sleep -Seconds 2
    sc.exe delete $ServiceName
}

# 2. Create Directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

# 3. Copy Executable
Copy-Item -Path $SourceExe -Destination "$InstallDir\$ExeName" -Force

# 4. Create Service (Auto Start)
$BinPath = "$InstallDir\$ExeName"
sc.exe create $ServiceName binPath= $BinPath start= auto DisplayName= "GateKeeper Endpoint Protection"

$SDDL = "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCLCSWLOCRRC;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"
sc.exe sdset $ServiceName $SDDL

sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000


sc.exe start $ServiceName

Write-Host "Installation Complete. GateKeeper is Active and Hardened." -ForegroundColor Green