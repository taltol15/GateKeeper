package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	ServiceName = "GateKeeper-L" // שיניתי את השם כדי למנוע בלבול עם הגרסה הרגילה
	RegPath     = `SOFTWARE\Policies\GateKeeper`

	WTS_SESSION_LOGON  = 0x5
	WTS_SESSION_LOGOFF = 0x6
	WTS_SESSION_LOCK   = 0x7
	WTS_SESSION_UNLOCK = 0x8
)

var elog *eventlog.Log

type gateKeeperService struct {
	shutdownFast chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	isProtecting bool
}

func main() {
	var err error
	elog, err = eventlog.Open(ServiceName)
	if err != nil {
		return
	}
	defer elog.Close()

	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine session: %v", err)
	}

	if len(os.Args) > 1 && os.Args[1] == "install" {
		err := installService()
		if err != nil {
			fmt.Printf("Failed to install: %v\n", err)
		} else {
			fmt.Println("Service installed successfully.")
		}
		return
	}

	if isIntSess {
		runConsoleMode()
	} else {
		runServiceMode()
	}
}

func runServiceMode() {
	err := svc.Run(ServiceName, &gateKeeperService{
		shutdownFast: make(chan struct{}),
	})
	if err != nil {
		elog.Error(1, fmt.Sprintf("Service failed: %v", err))
	}
}

func runConsoleMode() {
	fmt.Println("Running in Console Mode (Debug)...")
	fmt.Println("GateKeeper-L: Simulating Pre-Login Protection (ignoring Lock)...")

	svc := &gateKeeperService{}
	svc.startProtection()

	select {}
}

func (m *gateKeeperService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptSessionChange
	changes <- svc.Status{State: svc.StartPending}

	config := loadConfig()
	elog.Info(100, fmt.Sprintf("GateKeeper-L Started. Config Loaded: Enabled=%v", config.Enabled))

	// חובה: מתחילים במצב חסימה (עבור Boot/Restart כשאין עדיין יוזר)
	m.startProtection()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				m.stopProtection()
				break loop
			case svc.SessionChange:
				m.handleSessionChange(uint32(c.EventType))
			default:
				elog.Warning(102, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return false, 0
}

// --- השינוי הגדול נמצא כאן ---
func (m *gateKeeperService) handleSessionChange(eventType uint32) {
	switch eventType {
	case WTS_SESSION_LOGON, WTS_SESSION_UNLOCK:
		elog.Info(200, "User Session Active -> Suspending Protection & Releasing Devices")
		m.stopProtection()
		
		go releaseAllBlockedDevices()

	case WTS_SESSION_LOGOFF:
		// רק כשהמשתמש מתנתק לגמרי (Sign out) אנחנו חוסמים
		elog.Info(201, "User Logged Off -> Enabling Protection")
		m.startProtection()

	case WTS_SESSION_LOCK:
		// כאן השינוי: בנעילת מסך (Win+L) אנחנו לא עושים כלום!
		// המכשיר יישאר במצב שהיה קודם (כלומר: פתוח, כי המשתמש מחובר)
		elog.Info(202, "Session Locked (Win+L) -> Ignoring (Policy: GateKeeper-L / Pre-Login Only)")
	}
}

func (m *gateKeeperService) startProtection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isProtecting {
		return
	}
	m.isProtecting = true
	m.wg.Add(1)

	go m.sentryLoop() // קורא לפונקציה מתוך sentry.go (ללא שינוי)
}

func (m *gateKeeperService) stopProtection() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isProtecting {
		return
	}
	m.isProtecting = false
}

// --- Helpers ---

type Config struct {
	Enabled bool
}

func loadConfig() Config {
	cfg := Config{Enabled: true}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.QUERY_VALUE)
	if err != nil {
		return cfg
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("Enabled")
	if err == nil && val == 0 {
		cfg.Enabled = false
	}
	return cfg
}

func installService() error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", ServiceName)
	}
	s, err = m.CreateService(ServiceName, exepath, mgr.Config{
		DisplayName: "GateKeeper L (Pre-Login Only)", // שם מעודכן לתצוגה
		Description: "Secures USB ports ONLY during Boot/Logoff (ignores Lock Screen).",
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return err
	}
	defer s.Close()
	return nil
}