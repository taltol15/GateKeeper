package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

const PnpUtilPath = `C:\Windows\System32\pnputil.exe`

const (
	EventInfo  = 7000
	EventWarn  = 8000
	EventError = 9000
)

var allowedGuids = map[string]bool{
	"{4d36e96b-e325-11ce-bfc1-08002be10318}": true, // Keyboard
	"{4d36e96f-e325-11ce-bfc1-08002be10318}": true, // Mouse
	"{745a17a0-74d3-11d0-b6fe-00a0c90f57da}": true, // HID Class
	"{50dd5230-ba8a-11d1-bf5d-0000f805f530}": true, // Smart Card
	"{36fc9e60-c465-11cf-8056-444553540000}": true, // USB Hubs
	"{e0cbf06c-cd8b-4647-bb8a-263b43f0f974}": true, // Bluetooth Wireless
}

var sentryMu sync.Mutex

func (m *gateKeeperService) sentryLoop() {
	defer m.wg.Done()
	
	elog.Info(EventInfo, "DEBUG: Sentry Loop Started. Running initial scan...")
	scanAndBlock()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		m.mu.Lock()
		active := m.isProtecting
		m.mu.Unlock()

		if !active {
			return
		}

		select {
		case <-ticker.C:
			scanAndBlock()
		}
	}
}

func scanPath(baseRegistryPath string, sessionCache map[string]bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, baseRegistryPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		elog.Error(EventError, fmt.Sprintf("CRITICAL: Failed to open Registry path [%s]. Error: %v", baseRegistryPath, err))
		return
	}
	defer k.Close()

	devices, err := k.ReadSubKeyNames(-1)
	if err != nil {
		elog.Error(EventError, fmt.Sprintf("CRITICAL: Failed to list subkeys in [%s]. Error: %v", baseRegistryPath, err))
		return
	}


	if len(devices) == 0 {
		// elog.Warning(EventWarn, fmt.Sprintf("DEBUG: No devices found in %s", baseRegistryPath))
	} 

	for _, deviceID := range devices {
		checkDevice(baseRegistryPath, deviceID, sessionCache)
	}
}

func scanAndBlock() {
	sessionCache := make(map[string]bool)
	scanPath(`SYSTEM\CurrentControlSet\Enum\USB`, sessionCache)
	scanPath(`SYSTEM\CurrentControlSet\Enum\USBSTOR`, sessionCache)
}

func checkDevice(basePath string, deviceID string, sessionCache map[string]bool) {
	fullKeyPath := basePath + `\` + deviceID
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fullKeyPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	defer k.Close()

	instances, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return
	}

	for _, instance := range instances {
		prefix := basePath[strings.LastIndex(basePath, `\`)+1:]
		fullDeviceID := prefix + "\\" + deviceID + "\\" + instance
		
		if sessionCache[fullDeviceID] {
			continue
		}

		instancePath := fullKeyPath + `\` + instance
		analyzeAndAct(instancePath, fullDeviceID, sessionCache)
	}
}

func analyzeAndAct(regPath string, fullDeviceID string, sessionCache map[string]bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, regPath, registry.QUERY_VALUE)
	if err != nil {
		elog.Error(EventError, fmt.Sprintf("DEBUG: Failed to read device properties for %s", fullDeviceID))
		return
	}
	defer k.Close()

	classGUID, _, err := k.GetStringValue("ClassGUID")
	
	if allowedGuids[strings.ToLower(classGUID)] {
		sessionCache[fullDeviceID] = true
		// elog.Info(EventInfo, fmt.Sprintf("DEBUG: Skipping Whitelisted Device: %s (Class: %s)", fullDeviceID, classGUID))
		return
	}

	configFlags, _, err := k.GetIntegerValue("ConfigFlags")
	isRegistrySaysDisabled := (err == nil && (configFlags&0x00000001) != 0)

	success, output := runPnpCommand("/disable-device", fullDeviceID)
	sessionCache[fullDeviceID] = true

	if success {
		if !isRegistrySaysDisabled {
			elog.Info(EventInfo, fmt.Sprintf("SUCCESS: Blocked %s (GUID: %s)", fullDeviceID, classGUID))
		}
	} else {
		if !strings.Contains(output, "device is not connected") {
			elog.Error(EventError, fmt.Sprintf("FAILED to block %s. Out: %s", fullDeviceID, output))
		} else {

		}
	}
}

func runPnpCommand(action string, deviceArg string) (bool, string) {

	if _, err := os.Stat(PnpUtilPath); os.IsNotExist(err) {
		return false, "CRITICAL: pnputil.exe NOT FOUND!"
	}
	
	cmd := exec.Command(PnpUtilPath, action, deviceArg)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	
	out, err := cmd.CombinedOutput()
	outputStr := string(out)

	if err != nil {
		return false, outputStr
	}
	if strings.Contains(outputStr, "Failed") {
		return false, outputStr
	}
	return true, outputStr
}

func releaseAllBlockedDevices() {
	elog.Info(EventInfo, "DEBUG: Handover started (Diagnostic Mode)...")
	
	
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Enum\USB`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		elog.Error(EventError, fmt.Sprintf("CRITICAL: Handover cannot read Registry! Error: %v", err))
		return
	}
	k.Close()

	
	pathsToScan := []string{
		`SYSTEM\CurrentControlSet\Enum\USB`,
		`SYSTEM\CurrentControlSet\Enum\USBSTOR`,
	}

	count := 0
	for _, basePath := range pathsToScan {
		prefix := basePath[strings.LastIndex(basePath, `\`)+1:]
		
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath, registry.ENUMERATE_SUB_KEYS)
		if err != nil { continue }
		defer k.Close()

		devices, _ := k.ReadSubKeyNames(-1)
		
		for _, deviceID := range devices {
			subPath := basePath + `\` + deviceID
			subK, err := registry.OpenKey(registry.LOCAL_MACHINE, subPath, registry.ENUMERATE_SUB_KEYS)
			if err != nil { continue }
			
			instances, _ := subK.ReadSubKeyNames(-1)
			subK.Close()

			for _, instance := range instances {
				fullDeviceID := prefix + "\\" + deviceID + "\\" + instance
				instancePath := subPath + `\` + instance
				
				dKey, err := registry.OpenKey(registry.LOCAL_MACHINE, instancePath, registry.QUERY_VALUE)
				if err != nil { continue }

				configFlags, _, err := dKey.GetIntegerValue("ConfigFlags")
				dKey.Close()

				if err == nil && (configFlags&0x00000001) != 0 {
					success, output := runPnpCommand("/enable-device", fullDeviceID)
					if success {
						elog.Info(EventInfo, fmt.Sprintf("Healing SUCCESS: %s", fullDeviceID))
						count++
					} else {
						if !strings.Contains(output, "device is not connected") {
							elog.Error(EventError, fmt.Sprintf("Healing FAILED: %s. Out: %s", fullDeviceID, output))
						}
					}
				}
			}
		}
	}
	elog.Info(EventInfo, fmt.Sprintf("Handover Complete. Healed %d devices.", count))
}