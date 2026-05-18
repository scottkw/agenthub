package pty

import (
	"context"
	"fmt"
	"os"
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

	// mu protects concurrent access to mutable fields (State, pty, killed).
	mu sync.Mutex

	// killed is set by MarkKilled before killSession runs. The exit watcher
	// goroutine checks this to avoid calling cmd.Wait() on the kill path,
	// which would race with go-pty's internal waitOnContext goroutine.
	killed bool

	// exitCode caches the exit code once the process has exited (-1 = not set).
	exitCode int

	// ptyCloseOnce gates s.pty.Close() so the killSession path and the Linux
	// exit detector goroutine cannot race on go-pty's unsynchronised
	// unixPty.Close() field writes. Both paths must close through closePTY().
	ptyCloseOnce sync.Once
}

// closePTY closes the underlying PTY exactly once across all callers,
// even when killSession and the Linux exit detector race to clean up.
func (s *Session) closePTY() {
	s.mu.Lock()
	p := s.pty
	s.mu.Unlock()
	if p == nil {
		return
	}
	s.ptyCloseOnce.Do(func() {
		_ = p.Close()
	})
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

// Signal sends the given signal to the session's child process.
// Returns an error if the process has not been started or has already exited.
// The caller must hold no lock on s.mu — this method acquires it internally.
func (s *Session) Signal(sig os.Signal) error {
	s.mu.Lock()
	cmd := s.cmd
	s.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("session %s: process not running", s.ID)
	}
	return cmd.Process.Signal(sig)
}

// MarkKilled sets the killed flag so the exit watcher skips cmd.Wait().
// Must be called before killSession triggers PTY close or context cancel.
func (s *Session) MarkKilled() {
	s.mu.Lock()
	s.killed = true
	s.mu.Unlock()
}

// IsKilled returns whether the session was killed (vs natural exit).
func (s *Session) IsKilled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.killed
}

// GetState returns the current lifecycle state under the internal mutex.
// Safe to call from any goroutine.
func (s *Session) GetState() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.State
}

// SetState updates the session lifecycle state under the internal mutex.
// Safe to call from any goroutine.
func (s *Session) SetState(state SessionState) {
	s.mu.Lock()
	s.State = state
	s.mu.Unlock()
}

// CancelContext cancels the session's context, triggering go-pty's internal
// waitOnContext goroutine to reap the process.
func (s *Session) CancelContext() {
	if s.cancel != nil {
		s.cancel()
	}
}

// SetExitCode caches the exit code. Safe to call from any goroutine.
func (s *Session) SetExitCode(code int) {
	s.mu.Lock()
	s.exitCode = code
	s.mu.Unlock()
}

// ExitCode returns the cached exit code, or -1 if not yet set.
// Does NOT read cmd.ProcessState directly — that field is written by go-pty
// without our mutex and would race.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitCode
}
