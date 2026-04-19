//go:build !windows

// Tests for Session.WaitForExit and Session.ExitCode methods.
// These run only on POSIX — process exit semantics are the same on macOS/Linux.
package pty

import (
	"context"
	"testing"
	"time"
)

// TestExitDetection_ExitCode_NilCmd verifies that ExitCode returns -1
// when the session has no underlying process (cmd == nil).
// Requirement: ExitCode returns -1 if still running / no process.
func TestExitDetection_ExitCode_NilCmd(t *testing.T) {
	s := &Session{ID: "test-no-cmd", cmd: nil}
	if got := s.ExitCode(); got != -1 {
		t.Errorf("ExitCode with nil cmd: got %d, want -1", got)
	}
}

// TestExitDetection_WaitForExit_NilCmd verifies that WaitForExit returns 0
// when the session has no underlying process (cmd == nil).
// Requirement: WaitForExit returns 0 for nil cmd (conservative default per D-10).
func TestExitDetection_WaitForExit_NilCmd(t *testing.T) {
	s := &Session{ID: "test-no-cmd-wait", cmd: nil}
	if got := s.WaitForExit(); got != 0 {
		t.Errorf("WaitForExit with nil cmd: got %d, want 0", got)
	}
}

// TestExitDetection_ExitCode_RunningProcess verifies that ExitCode returns -1
// while the process is still running (ProcessState is nil until the process exits).
func TestExitDetection_ExitCode_RunningProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}

	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// cat stays running until killed — ExitCode must return -1
	sess, err := b.Create(ctx, CreateRequest{CLI: "cat", Cols: 80, Rows: 24})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	got := sess.ExitCode()
	if got != -1 {
		t.Errorf("ExitCode while running: got %d, want -1", got)
	}
}

// TestExitDetection_WaitForExit_NaturalExit verifies that WaitForExit returns
// the exit code after a process exits naturally with exit code 0.
// Requirement: WaitForExit blocks until exit and returns the correct exit code.
func TestExitDetection_WaitForExit_NaturalExit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}

	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sh -c 'exit 0' exits immediately with code 0
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "sh",
		Args: []string{"-c", "exit 0"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	// Drain PTY output until EOF so the process finishes
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		for {
			_, err := sess.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process to exit")
	}

	// Give the OS a brief moment to populate ProcessState after PTY EOF
	time.Sleep(100 * time.Millisecond)

	got := sess.WaitForExit()
	if got != 0 {
		t.Errorf("WaitForExit after natural exit 0: got %d, want 0", got)
	}
}

// TestExitDetection_WaitForExit_NonZeroExit verifies that WaitForExit captures
// a non-zero exit code correctly.
func TestExitDetection_WaitForExit_NonZeroExit(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: skipped in -short mode")
	}

	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sh -c 'exit 42' exits with code 42
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "sh",
		Args: []string{"-c", "exit 42"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer b.Kill(sess.ID) //nolint:errcheck

	// Drain PTY output until EOF
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		for {
			_, err := sess.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process to exit")
	}

	time.Sleep(100 * time.Millisecond)

	got := sess.WaitForExit()
	if got != 42 {
		t.Errorf("WaitForExit after exit 42: got %d, want 42", got)
	}
}

// TestExitDetection_SetState verifies that SetState transitions session state
// under the mutex safely.
func TestExitDetection_SetState(t *testing.T) {
	s := &Session{ID: "test-set-state", State: StateRunning}

	if s.State != StateRunning {
		t.Fatalf("initial state: got %v, want StateRunning", s.State)
	}

	s.SetState(StateStopped)

	if s.State != StateStopped {
		t.Errorf("after SetState(StateStopped): got %v, want StateStopped", s.State)
	}
}
