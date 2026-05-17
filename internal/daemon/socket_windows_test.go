//go:build windows

package daemon

import (
	"fmt"
	"strings"
	"testing"
	"time"

	winio "github.com/tailscale/go-winio"
)

func uniqueWindowsPipePath(prefix string) string {
	return fmt.Sprintf(`\\.\pipe\agenthub-test-%s-%d`, prefix, time.Now().UnixNano())
}

func TestCleanupStaleSocket_WindowsPipe_NoServer(t *testing.T) {
	path := uniqueWindowsPipePath("nostale")
	// Nothing listening -- should return nil.
	if err := CleanupStaleSocket(path); err != nil {
		t.Errorf("CleanupStaleSocket on absent pipe: unexpected error: %v", err)
	}
}

func TestCleanupStaleSocket_WindowsPipe_Active(t *testing.T) {
	path := uniqueWindowsPipePath("active")
	ln, err := winio.ListenPipe(path, nil)
	if err != nil {
		t.Fatalf("ListenPipe: %v", err)
	}
	defer ln.Close()
	// Accept connections in background so DialPipe succeeds.
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	err = CleanupStaleSocket(path)
	if err == nil {
		t.Error("CleanupStaleSocket on active pipe: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error should mention 'already running', got: %v", err)
	}
}

func TestAPIStart_WindowsNamedPipeHealth(t *testing.T) {
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false

	api := NewAPI(engine)
	path := uniqueWindowsPipePath("api-health")
	if err := api.Start(path); err != nil {
		t.Fatalf("api.Start on named pipe: %v", err)
	}
	t.Cleanup(func() { _ = api.Stop() })

	client := NewDaemonClient(path)
	if err := client.Health(); err != nil {
		t.Fatalf("client.Health over named pipe: %v", err)
	}
}

func TestAPIStop_WindowsNamedPipe(t *testing.T) {
	engine := NewSessionEngine()
	engine.configDir = t.TempDir()
	engine.cliPaths = make(map[string]string)
	engine.startMinimized = false

	api := NewAPI(engine)
	path := uniqueWindowsPipePath("api-stop")
	if err := api.Start(path); err != nil {
		t.Fatalf("api.Start on named pipe: %v", err)
	}
	if err := api.Stop(); err != nil {
		t.Fatalf("api.Stop on named pipe: %v", err)
	}

	client := NewDaemonClient(path)
	if err := client.Health(); err == nil {
		t.Fatal("client.Health after api.Stop: expected error, got nil")
	}
}
