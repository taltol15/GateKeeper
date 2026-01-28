# 🛡️ GateKeeper L (Lite)

> **Special Edition: Pre-Login Enforcement Only**

Current Branch: `gatekeeper-lite`
For the full Enterprise version (that blocks Lock Screen as well), switch to the [main branch](../../tree/main).

---

## ⚡ What is GateKeeper L?
This is a specific variant of the GateKeeper security tool designed for environments where **background tasks must continue running when the screen is locked**.

### Key Differences
| Feature | GateKeeper (Main) | GateKeeper L (Lite) |
| :--- | :--- | :--- |
| **Boot / Pre-Login** | 🔒 Blocked | 🔒 Blocked |
| **User Logged In** | ✅ Allowed | ✅ Allowed |
| **Screen Lock (Win+L)** | 🔒 **Blocked** | ✅ **Allowed (Active)** |
| **Logoff / Sign Out** | 🔒 Blocked | 🔒 Blocked |

## 🛠️ Installation
1. Download `gatekeeper-l.exe` and `Setup_GUI.ps1`.
2. Run `Setup_GUI.ps1` with PowerShell.
3. Click **Install & Harden**.

---