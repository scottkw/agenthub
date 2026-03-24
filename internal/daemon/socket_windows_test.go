//go:build windows

package daemon

import (
	"strings"
	"testing"

	winio "github.com/tailscale/go-winio"
)

func TestCleanupStaleSocket_WindowsPipe_NoServer(t *testing.T) {
	path := `\\.\pipe\agenthub-test-nostale`
	// Nothing listening -- should return nil.
	if err := CleanupStaleSocket(path); err != nil {
		t.Errorf("CleanupStaleSocket on absent pipe: unexpected error: %v", err)
	}
}

func TestCleanupStaleSocket_WindowsPipe_Active(t *testing.T) {
	path := `\\.\pipe\agenthub-test-active`
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
