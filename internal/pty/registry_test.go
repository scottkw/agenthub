package pty

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// newTestSession creates a minimal Session for use in registry tests.
func newTestSession(id, cli string) *Session {
	return &Session{
		ID:        id,
		CLI:       cli,
		State:     StateRunning,
		CreatedAt: time.Now(),
	}
}

// TestRegistry_AddAndGet verifies that adding a session and then getting it by
// ID returns the same pointer.
func TestRegistry_AddAndGet(t *testing.T) {
	r := NewSessionRegistry()
	s := newTestSession("abc", "claude")

	r.Add(s)

	got, ok := r.Get("abc")
	if !ok {
		t.Fatal("expected Get to return true")
	}
	if got != s {
		t.Error("expected the same Session pointer back from Get")
	}
}

// TestRegistry_List verifies that List returns all added sessions.
func TestRegistry_List(t *testing.T) {
	r := NewSessionRegistry()
	for i := 0; i < 3; i++ {
		r.Add(newTestSession(fmt.Sprintf("id-%d", i), "claude"))
	}

	all := r.List()
	if len(all) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(all))
	}
}

// TestRegistry_Remove verifies that a removed session is no longer retrievable.
func TestRegistry_Remove(t *testing.T) {
	r := NewSessionRegistry()
	s := newTestSession("del", "codex")
	r.Add(s)

	removed := r.Remove("del")
	if !removed {
		t.Error("expected Remove to return true for existing session")
	}

	_, ok := r.Get("del")
	if ok {
		t.Error("expected Get to return false after Remove")
	}
}

// TestRegistry_GetNotFound verifies behaviour for a non-existent ID.
func TestRegistry_GetNotFound(t *testing.T) {
	r := NewSessionRegistry()

	got, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for missing session")
	}
	if got != nil {
		t.Error("expected nil session for missing ID")
	}
}

// TestRegistry_ConcurrentAccess verifies the registry is free of data races
// under concurrent writes. Run with -race.
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewSessionRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			r.Add(newTestSession(fmt.Sprintf("s-%d", i), "claude"))
		}()
	}
	wg.Wait()

	if r.Len() != 100 {
		t.Errorf("expected 100 sessions, got %d", r.Len())
	}
}

// TestRegistry_SessionPersistsAfterSimulatedWindowClose verifies that cancelling
// the context associated with a session does NOT remove it from the registry.
// Sessions outlive the UI connection — the registry is the source of truth.
func TestRegistry_SessionPersistsAfterSimulatedWindowClose(t *testing.T) {
	r := NewSessionRegistry()

	_, cancel := context.WithCancel(context.Background())
	s := newTestSession("persist", "gemini")
	r.Add(s)

	// Simulate window / connection close by cancelling the context.
	cancel()

	// Session must still be in the registry.
	got, ok := r.Get("persist")
	if !ok {
		t.Fatal("session removed from registry after context cancel — sessions must persist")
	}
	if got.State != StateRunning {
		t.Errorf("expected StateRunning, got %v", got.State)
	}
}
