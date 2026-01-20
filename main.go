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
	ServiceName = "GateKeeper"
	RegPath     = `SOFTWARE\Policies\GateKeeper`

	// קבועי אירועי Session (Windows Constants)
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
	// 1. אתחול הלוגים
	var err error
	elog, err = eventlog.Open(ServiceName)
	if err != nil {
		return
	}
	defer elog.Close()

	// 2. זיהוי האם אנחנו רצים כ-Service או ב-Console
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine session: %v", err)
	}

	// התקנה דרך שורת הפקודה
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
	fmt.Println("Simulating Lock Screen Protection...")

	// יצירת מופע זמני לבדיקה
	svc := &gateKeeperService{}
	svc.startProtection()

	// השארת התוכנה רצה לבדיקה
	select {}
}

func (m *gateKeeperService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	// חובה לקבל את svc.AcceptSessionChange כדי לזהות נעילה/לוגין
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptSessionChange
	changes <- svc.Status{State: svc.StartPending}

	// טעינת הגדרות
	config := loadConfig()
	elog.Info(100, fmt.Sprintf("GateKeeper Started. Config Loaded: Enabled=%v", config.Enabled))

	// הנחה התחלתית: ב-Boot אנחנו מגנים
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
				// המרה של EventType ל-uint32 כדי להשוות לקבועים שלנו
				m.handleSessionChange(uint32(c.EventType))
			default:
				elog.Warning(102, fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return false, 0
}

func (m *gateKeeperService) handleSessionChange(eventType uint32) {
	switch eventType {
	case WTS_SESSION_LOGON, WTS_SESSION_UNLOCK:
		elog.Info(200, "User Session Active -> Suspending Protection & Releasing Devices")
		m.stopProtection()
		
		// התוספת הקריטית: שחרור התקנים כדי שה-DLP יקח פיקוד
		go releaseAllBlockedDevices()

	case WTS_SESSION_LOGOFF, WTS_SESSION_LOCK:
		elog.Info(201, "User Locked/Away -> Enabling Protection")
		m.startProtection()
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

	// הפעלת ה-Sentry (הסורק) שנמצא בקובץ sentry.go
	go m.sentryLoop()
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
		DisplayName: "GateKeeper USB Security",
		Description: "Secures USB ports during pre-login state.",
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return err
	}
	defer s.Close()
	return nil
}