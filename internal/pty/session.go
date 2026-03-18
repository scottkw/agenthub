package pty

import (
	"context"
	"fmt"
	"sync"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

// SessionState represents the current lifecycle state of a Session.
type SessionState int

const (
	// StateRunning means the CLI process is active.
	StateRunning SessionState = iota

	// StateStopped means the CLI process has exited or been killed.
	StateStopped
)

// Session represents a single running CLI session backed by a PTY.
// It implements io.ReadWriter so output fan-out can be added in Phase 2.
type Session struct {
	// ID is the unique identifier for this session.
	ID string

	// CLI is the name of the CLI binary that was launched (e.g. "claude").
	CLI string

	// State is the current lifecycle state of the session.
	State SessionState

	// CreatedAt is the time the session was created.
	CreatedAt time.Time

	// pty is the underlying pseudo-terminal handle.
	pty gopty.Pty

	// cmd is the PTY-attached command (holds the process reference).
	cmd *gopty.Cmd

	// cancel stops the context that was used to start the process.
	cancel context.CancelFunc

	// job holds a platform-specific cleanup handle (Windows: *jobObject, POSIX: nil).
	// Stored as any to avoid build-tag spread across files.
	job any

	// mu protects concurrent access to mutable fields (State, pty).
	mu sync.Mutex
}

// Read reads output from the underlying PTY.
// Implements io.Reader — Phase 2 will fan this out to connected WebSocket clients.
func (s *Session) Read(p []byte) (int, error) {
	s.mu.Lock()
	pty := s.pty
	s.mu.Unlock()
	if pty == nil {
		return 0, fmt.Errorf("session %s: PTY not initialised", s.ID)
	}
	return pty.Read(p)
}

// Write sends input to the underlying PTY (i.e. to the CLI process stdin).
// Implements io.Writer.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	pty := s.pty
	s.mu.Unlock()
	if pty == nil {
		return 0, fmt.Errorf("session %s: PTY not initialised", s.ID)
	}
	return pty.Write(p)
}

// String returns a human-readable description of the session.
func (s *Session) String() string {
	return fmt.Sprintf("Session{ID: %q, CLI: %q}", s.ID, s.CLI)
}
