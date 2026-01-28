<#
    GateKeeper-L Installer GUI
    Professional Setup Wizard for Manual Installations
#>

# 1. Self-Elevation (Run as Admin check)
if (!([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Start-Process powershell.exe "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    Exit
}

# Load Windows Forms
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# --- Configuration ---
$ServiceName = "GateKeeper-L"
$ExeName = "gatekeeper-l.exe"
$InstallDir = "C:\Program Files\GateKeeper-L"
$SourceExe = "$PSScriptRoot\$ExeName"

# --- GUI Setup ---
$form = New-Object System.Windows.Forms.Form
$form.Text = "GateKeeper-L Setup"
$form.Size = New-Object System.Drawing.Size(500,400)
$form.StartPosition = "CenterScreen"
$form.FormBorderStyle = "FixedDialog"
$form.MaximizeBox = $false
$form.BackColor = "#ffffff"

# Title
$title = New-Object System.Windows.Forms.Label
$title.Text = "GateKeeper L (Pre-Login Only)"
$title.Font = New-Object System.Drawing.Font("Segoe UI", 16, [System.Drawing.FontStyle]::Bold)
$title.ForeColor = "#0078D7"
$title.AutoSize = $true
$title.Location = New-Object System.Drawing.Point(20, 20)
$form.Controls.Add($title)

# Subtitle
$subTitle = New-Object System.Windows.Forms.Label
$subTitle.Text = "Secures USB ports ONLY during Boot/Logoff"
$subTitle.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$subTitle.ForeColor = "#555555"
$subTitle.AutoSize = $true
$subTitle.Location = New-Object System.Drawing.Point(22, 50)
$form.Controls.Add($subTitle)

# Log Box
$logBox = New-Object System.Windows.Forms.TextBox
$logBox.Multiline = $true
$logBox.ScrollBars = "Vertical"
$logBox.ReadOnly = $true
$logBox.Location = New-Object System.Drawing.Point(20, 90)
$logBox.Size = New-Object System.Drawing.Size(440, 180)
$logBox.Font = New-Object System.Drawing.Font("Consolas", 9)
$logBox.BackColor = "#f0f0f0"
$logBox.Text = "Ready to install. Please ensure gatekeeper-l.exe is in the same folder.`r`n"
$form.Controls.Add($logBox)

# Helper Function for Logging
function Write-Log($message) {
    $logBox.AppendText("[$([DateTime]::Now.ToString('HH:mm:ss'))] $message`r`n")
    $logBox.ScrollToCaret()
    [System.Windows.Forms.Application]::DoEvents() # Keep UI responsive
}

# Install Button
$btnInstall = New-Object System.Windows.Forms.Button
$btnInstall.Text = "Install & Harden"
$btnInstall.Location = New-Object System.Drawing.Point(20, 290)
$btnInstall.Size = New-Object System.Drawing.Size(140, 40)
$btnInstall.Font = New-Object System.Drawing.Font("Segoe UI", 10, [System.Drawing.FontStyle]::Bold)
$btnInstall.BackColor = "#28a745" # Green
$btnInstall.ForeColor = "#ffffff"
$btnInstall.FlatStyle = "Flat"
$form.Controls.Add($btnInstall)

# Uninstall Button
$btnUninstall = New-Object System.Windows.Forms.Button
$btnUninstall.Text = "Uninstall"
$btnUninstall.Location = New-Object System.Drawing.Point(170, 290)
$btnUninstall.Size = New-Object System.Drawing.Size(100, 40)
$btnUninstall.Font = New-Object System.Drawing.Font("Segoe UI", 10)
$btnUninstall.BackColor = "#dc3545" # Red
$btnUninstall.ForeColor = "#ffffff"
$btnUninstall.FlatStyle = "Flat"
$form.Controls.Add($btnUninstall)

# Exit Button
$btnExit = New-Object System.Windows.Forms.Button
$btnExit.Text = "Exit"
$btnExit.Location = New-Object System.Drawing.Point(360, 290)
$btnExit.Size = New-Object System.Drawing.Size(100, 40)
$form.Controls.Add($btnExit)

# --- Logic Actions ---

$btnInstall.Add_Click({
    $btnInstall.Enabled = $false
    $btnUninstall.Enabled = $false
    Write-Log "Starting Installation..."

    # Check for EXE
    if (!(Test-Path $SourceExe)) {
        Write-Log "ERROR: gatekeeper-l.exe not found!"
        [System.Windows.Forms.MessageBox]::Show("gatekeeper-l.exe missing!", "Error", "OK", "Error")
        $btnInstall.Enabled = $true
        return
    }

    # Stop old service if exists
    Write-Log "Cleaning up old versions..."
    try {
        sc.exe stop $ServiceName | Out-Null
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    } catch {}

    # Copy Files
    Write-Log "Copying files to Program Files..."
    if (!(Test-Path $InstallDir)) { New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null }
    Copy-Item -Path $SourceExe -Destination "$InstallDir\$ExeName" -Force

    # Create Service
    Write-Log "Registering Windows Service..."
    $BinPath = "$InstallDir\$ExeName"
    sc.exe create $ServiceName binPath= $BinPath start= auto DisplayName= "GateKeeper L (Pre-Login Only)" | Out-Null

    # Recovery
    Write-Log "Setting recovery options..."
    sc.exe failure $ServiceName reset= 86400 actions= restart/60000/restart/60000/restart/60000 | Out-Null

    # Start
    Write-Log "Starting Service..."
    sc.exe start $ServiceName | Out-Null
    Start-Sleep -Seconds 2

    # Hardening
    Write-Log "Applying Security Hardening (SDDL)..."
    $HardenedSDDL = "D:(A;;GA;;;SY)(A;;CCLCSWLOCRRC;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"
    sc.exe sdset $ServiceName $HardenedSDDL | Out-Null

    Write-Log "SUCCESS: Installation Complete!"
    [System.Windows.Forms.MessageBox]::Show("GateKeeper-L Installed Successfully!", "Success", "OK", "Information")
    
    $btnInstall.Enabled = $true
    $btnUninstall.Enabled = $true
})

$btnUninstall.Add_Click({
    if ([System.Windows.Forms.MessageBox]::Show("Are you sure you want to remove GateKeeper-L?", "Confirm", "YesNo", "Warning") -eq "No") { return }
    
    $btnInstall.Enabled = $false
    $btnUninstall.Enabled = $false
    Write-Log "Starting Uninstallation..."

    # Nuke Hardening (Registry Hack)
    Write-Log "Removing Security Hardening..."
    Remove-Item -Path "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName" -Recurse -Force -ErrorAction SilentlyContinue
    
    # Try standard stop
    sc.exe stop $ServiceName | Out-Null
    
    # Remove Files
    Write-Log "Removing files..."
    Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue

    Write-Log "Uninstall logic complete. A RESTART is required to fully clear the service handle."
    [System.Windows.Forms.MessageBox]::Show("Uninstallation complete.`nPlease RESTART the computer to finish cleanup.", "Done", "OK", "Information")

    $btnInstall.Enabled = $true
    $btnUninstall.Enabled = $true
})

$btnExit.Add_Click({
    $form.Close()
})

# Show Form
$form.ShowDialog() | Out-Null