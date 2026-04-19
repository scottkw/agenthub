package daemon

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/scottkw/agenthub/internal/pty"
)

// TestExitEvent_AutoCloseSession_DefaultIsTrue verifies that GetAutoCloseSession
// returns true when the setting has never been set (nil pointer = default enabled per D-11).
func TestExitEvent_AutoCloseSession_DefaultIsTrue(t *testing.T) {
	e := &SessionEngine{}
	// autoCloseSession field is nil (zero value)
	got := e.GetAutoCloseSession()
	if !got {
		t.Errorf("GetAutoCloseSession with nil pointer: got false, want true (default enabled)")
	}
}

// TestExitEvent_AutoCloseSession_RoundTrip verifies that SetAutoCloseSession
// persists the value and GetAutoCloseSession returns it correctly.
func TestExitEvent_AutoCloseSession_RoundTrip(t *testing.T) {
	e := NewSessionEngine()
	e.configDir = t.TempDir() // isolate from real settings.json

	// Default is true.
	if !e.GetAutoCloseSession() {
		t.Error("initial GetAutoCloseSession: want true, got false")
	}

	// Set to false.
	e.SetAutoCloseSession(false)
	if got := e.GetAutoCloseSession(); got {
		t.Error("after SetAutoCloseSession(false): want false, got true")
	}

	// Set back to true.
	e.SetAutoCloseSession(true)
	if got := e.GetAutoCloseSession(); !got {
		t.Error("after SetAutoCloseSession(true): want true, got false")
	}
}

// TestExitEvent_AutoCloseSession_Persistence verifies that the auto-close-session
// setting survives a save/load cycle through settings.json.
func TestExitEvent_AutoCloseSession_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Write setting via engine 1, isolated to temp dir.
	e1 := NewSessionEngine()
	e1.configDir = dir
	e1.cliPaths = make(map[string]string) // isolate from real cliPaths

	e1.SetAutoCloseSession(false)

	// Reload via a fresh engine pointed at the same dir.
	e2 := NewSessionEngine()
	e2.configDir = dir
	e2.cliPaths = make(map[string]string)
	e2.loadSettingsFromDisk(dir)

	if got := e2.GetAutoCloseSession(); got {
		t.Error("after persist false + reload: want false, got true")
	}

	// Also verify JSON on disk directly.
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	var s daemonSettings
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal settings.json: %v", err)
	}
	if s.AutoCloseSession == nil {
		t.Fatal("settings.json AutoCloseSession: want *false, got nil")
	}
	if *s.AutoCloseSession != false {
		t.Errorf("settings.json AutoCloseSession: want false, got %v", *s.AutoCloseSession)
	}
}

// TestExitEvent_ExitWatcher_TransitionsToStopped verifies that when a session's
// process exits naturally (hub.Done() fires), the exit watcher goroutine transitions
// the session state to StateStopped and calls the onExit callback with the session ID
// and exit code.
//
// Uses the spy backend: Read returns EOF immediately, so hub.Done() fires quickly.
func TestExitEvent_ExitWatcher_TransitionsToStopped(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	var (
		mu           sync.Mutex
		callbackID   string
		callbackCode int
		called       = make(chan struct{})
	)

	onExit := func(id string, code int) {
		mu.Lock()
		callbackID = id
		callbackCode = code
		mu.Unlock()
		close(called)
	}

	id, err := e.CreateSession(context.Background(), "cat", "exit-test", "", nil, 80, 24, nil, onExit)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// hub.Done() fires when the hub's Run goroutine returns EOF from the spy session.
	// spy session Read returns error immediately (nil pty), so this is fast.
	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("onExit callback not called within 3s — exit watcher goroutine may not have fired")
	}

	// Verify callback received correct session ID.
	mu.Lock()
	gotID := callbackID
	gotCode := callbackCode
	mu.Unlock()

	if gotID != id {
		t.Errorf("onExit sessionID: got %q, want %q", gotID, id)
	}
	// spy cmd is nil, so WaitForExit returns 0.
	if gotCode != 0 {
		t.Errorf("onExit exitCode: got %d, want 0 (nil cmd)", gotCode)
	}

	// Verify session state was transitioned to StateStopped.
	sess, ok := e.registry.Get(id)
	if !ok {
		t.Fatal("session not found in registry after exit")
	}
	// Give a brief moment for the goroutine to complete SetState after onExit returns.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if sess.State == pty.StateStopped {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if sess.State != pty.StateStopped {
		t.Errorf("session state after exit: got %v, want StateStopped", sess.State)
	}
}

// TestExitEvent_ExitWatcher_NilOnExitNoIsPanic verifies that passing nil for onExit
// does not cause a panic when the exit watcher fires.
func TestExitEvent_ExitWatcher_NilOnExitNoIsPanic(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	id, err := e.CreateSession(context.Background(), "cat", "nil-exit-test", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Wait for hub to shut down — if it panics, the test will fail.
	hub, ok := e.manager.Get(id)
	if !ok {
		t.Fatal("hub not found")
	}
	select {
	case <-hub.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("hub.Done() not closed within 3s")
	}
	// Give exit watcher a moment to complete without panicking.
	time.Sleep(50 * time.Millisecond)
}

// TestExitEvent_ListSessions_ExitCodePopulatedForStopped verifies that ListSessions
// populates ExitCode and Duration for sessions that are in the StateStopped state,
// and leaves ExitCode nil for running sessions.
func TestExitEvent_ListSessions_ExitCodePopulatedForStopped(t *testing.T) {
	spy := &spyBackend{}
	e := NewSessionEngine()
	e.backend = spy

	// Create a session and manually transition it to stopped state.
	id, err := e.CreateSession(context.Background(), "cat", "stopped-session", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Initially the session should be running — ExitCode should be nil.
	sessions := e.ListSessions()
	var info SessionInfo
	for _, s := range sessions {
		if s.ID == id {
			info = s
			break
		}
	}
	if info.ID == "" {
		t.Fatal("session not found in ListSessions")
	}
	if info.ExitCode != nil {
		t.Errorf("ExitCode for running session: want nil, got %v", info.ExitCode)
	}
	if info.Duration != nil {
		t.Errorf("Duration for running session: want nil, got %v", info.Duration)
	}

	// Transition to stopped via SetState.
	sess, ok := e.registry.Get(id)
	if !ok {
		t.Fatal("session not found in registry")
	}
	sess.SetState(pty.StateStopped)

	// Now ListSessions must populate ExitCode and Duration.
	sessions = e.ListSessions()
	for _, s := range sessions {
		if s.ID == id {
			info = s
			break
		}
	}
	if info.ExitCode == nil {
		t.Error("ExitCode for stopped session: want non-nil *int, got nil")
	}
	if info.Duration == nil {
		t.Error("Duration for stopped session: want non-nil *int, got nil")
	}
	// ExitCode should be 0 (default cached value since spy session has no cmd).
	if info.ExitCode != nil && *info.ExitCode != 0 {
		t.Errorf("ExitCode value for stopped spy session: got %d, want 0 (default)", *info.ExitCode)
	}
}
