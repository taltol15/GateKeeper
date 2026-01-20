# 🛡️ GateKeeper: Zero-Trust USB Hardening

> **Pre-Login Endpoint Protection & Data Leakage Prevention (DLP)**

GateKeeper is a lightweight, hardened Windows Service written in Go. It enforces a strict **Zero-Trust policy** on USB ports whenever the workstation is locked (Pre-Login / Lock Screen), effectively closing the security gap before the OS or corporate DLP agents fully load.

## 🚀 Key Features

* **🔒 Aggressive Enforcement:** Automatically disables drivers for Mass Storage (Flash drives) and Network Adapters (Tethering) when the screen is locked.
* **🔌 Zero-Trust:** Whitelist-based approach. Only Keyboards, Mice, and authorized Hubs remain active.
* **🛡️ Hardened Service:** Protected by custom SDDL (Security Descriptor), preventing local admins from stopping or tampering with the service.
* **⚡ Native Performance:** Written in Go (Golang), zero dependencies, minimal footprint (<5MB).
* **🤝 Seamless Handover:** Instantly re-enables devices upon valid user login for corporate DLP inspection.

## 🛠️ Installation

### Prerequisites
* Windows 10 / 11 / Server 2016+
* Administrator Privileges

### Quick Install (Production)
1.  Download the latest release.
2.  Right-click `Setup_GUI.ps1` and select **Run with PowerShell**.
3.  Click **Install & Harden**.

### Manual / Enterprise Deployment (SCCM/Intune)
Run the following PowerShell command as SYSTEM/Admin:
```powershell
powershell.exe -ExecutionPolicy Bypass -File install_prod.ps1