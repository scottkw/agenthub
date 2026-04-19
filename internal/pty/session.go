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

	// exitCode caches the exit code once the process has exited.
	exitCode int
	exitOnce sync.Once
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

// WaitForExit blocks until the underlying process exits and returns the exit code.
// Returns 0 if cmd or ProcessState is nil (conservative default per D-10).
// IMPORTANT: Only call from the natural exit path (after hub.Done fires). The kill
// path must NOT call this — go-pty's cmd.Wait() is not safe to call concurrently
// with its internal waitOnContext goroutine triggered by context cancellation.
func (s *Session) WaitForExit() int {
	s.exitOnce.Do(func() {
		s.mu.Lock()
		cmd := s.cmd
		s.mu.Unlock()
		if cmd == nil {
			return
		}
		_ = cmd.Wait()
		if cmd.ProcessState != nil {
			s.exitCode = cmd.ProcessState.ExitCode()
		}
	})
	return s.exitCode
}

// ExitCode returns the exit code if the process has exited, or -1 if still running.
func (s *Session) ExitCode() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.ProcessState != nil {
		return s.cmd.ProcessState.ExitCode()
	}
	return -1
}
