========================================================================
             GateKeeper - Zero-Trust USB Enforcement System
          Endpoint Hardening & Data Leakage Prevention (DLP)
========================================================================

This directory contains the installation files for the GateKeeper system.
The system runs as a Windows Service and automatically blocks unauthorized 
USB devices (Mass Storage & Network Adapters) during the Lock Screen 
or Pre-Login state.

------------------------------------------------------------------------
📂 Folder Contents
------------------------------------------------------------------------

1. gatekeeper.exe
   The Core Service binary.
   NOTE: This file must remain in the same folder as the installation 
   scripts during the setup process.

2. Setup_GUI.ps1  (Recommended for Manual Install) 🟢
   A user-friendly Graphical User Interface (GUI) wizard.
   Includes options for "Install & Harden" and "Uninstall".
   > Usage: Right-click -> "Run with PowerShell".

3. install_prod.ps1 (Enterprise Deployment - SCCM/GPO) 🔒
   A silent installation script designed for Production environments.
   * Hardening: This script applies strict Security Descriptors (SDDL).
   * WARNING: Once installed, even Local Administrators cannot stop 
     or delete the service (Anti-Tamper protection).

4. install_debug.ps1 (Development & Testing Only) 🛠️
   An installation script that grants full control to Administrators.
   Allows stopping/starting the service for debugging purposes.
   > DO NOT deploy this version to end-user workstations.

------------------------------------------------------------------------
🚀 Installation Instructions
------------------------------------------------------------------------

Method A - GUI Wizard (Easiest):
1. Ensure 'gatekeeper.exe' is located next to 'Setup_GUI.ps1'.
2. Right-click 'Setup_GUI.ps1' and select "Run with PowerShell".
3. Click the "Install & Harden" button.
4. Wait for the "SUCCESS" message.

Method B - Command Line (For SysAdmins):
1. Open PowerShell as Administrator.
2. Navigate to this directory.
3. Run: .\install_prod.ps1

------------------------------------------------------------------------
📝 Troubleshooting & Logs
------------------------------------------------------------------------
All system activities are logged to the Windows Event Viewer.
Log Path:
Event Viewer -> Windows Logs -> Application
Source: GateKeeper

Event IDs:
* 7000: Successful Block / Handover / Heal.
* 8000: Block Attempt (Audit).
* 9000: Error / Failure.

========================================================================