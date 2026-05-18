//go:build linux

// Phase 110 / PTY-02: Linux-only unit tests for startExitDetector.
//
// These tests cover the detector lifecycle, the kill-path suppression
// invariant (detector must NOT reap when IsKilled is true), and the
// POSIX 128+signal convention for signaled exits.
//
// macOS executors cannot run these tests locally — the build tag ensures
// they only compile (and only run) under GOOS=linux. The cross-compile
// gate (GOOS=linux go vet) catches type/import errors on the macOS dev
// machine; actual `-race -shuffle=on` execution is captured as
// human_needed in 110-VERIFICATION.md.
package pty

import (
	"context"
	"os/exec"
	"testing"
	"time"

	gopty "github.com/aymanbagabas/go-pty"
)

// TestStartExitDetector_NaturalExit verifies that on a clean `exit 0` the
// detector reaps the child, caches exitCode == 0, and closes the PTY master
// (subsequent Read returns a non-nil error — proof of close).
func TestStartExitDetector_NaturalExit(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sh -c "exit 0" exits immediately with code 0.
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "/bin/sh",
		Args: []string{"-c", "exit 0"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Kill(sess.ID)
	})

	// Poll for ExitCode flipping from -1 to 0 within 2 seconds (20 ticks
	// at 100ms cadence is a 1900ms safety margin over 1 expected tick).
	deadline := time.Now().Add(2 * time.Second)
	var got int
	for time.Now().Before(deadline) {
		got = sess.ExitCode()
		if got != -1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got != 0 {
		t.Fatalf("ExitCode after natural exit: got %d, want 0 (detector failed to reap)", got)
	}

	// PTY master should be closed after the detector fires — a Read here
	// must return a non-nil error (EBADF, os.ErrClosed, or io.EOF).
	buf := make([]byte, 64)
	if _, rerr := sess.Read(buf); rerr == nil {
		t.Errorf("expected non-nil Read error after detector close, got nil")
	}
}

// TestStartExitDetector_SuppressedOnKill verifies that when IsKilled() is
// true the detector returns silently without reaping the child.
//
// To eliminate timing fragility under `-race -shuffle=on -count=10` (Phase 110
// WR-02 fix), this test constructs a Session manually rather than going
// through NativePTYBackend.Create. By setting killed=true BEFORE invoking
// startExitDetector, the test guarantees the detector observes IsKilled on
// its very first tick regardless of scheduler perturbation. Going through
// b.Create() would call startExitDetector internally; even though the
// subsequent MarkKilled call finishes within microseconds in normal runs,
// the race detector's scheduler shuffling could let the first 100ms tick
// fire before MarkKilled lands, weakening the invariant the test asserts.
//
// We start `sleep 5` directly via os/exec (not a PTY) purely to obtain a
// real PID that startExitDetector's `s.cmd.Process == nil` guard accepts.
// The child is never read from and the PTY field stays nil — the detector
// is expected to bail on IsKilled before touching either.
func TestStartExitDetector_SuppressedOnKill(t *testing.T) {
	// Real child process for a real PID. Not wrapped in a PTY — the
	// detector returns before any PTY access.
	cmd := exec.Command("/bin/sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	// Manually constructed Session: killed=true is set BEFORE
	// startExitDetector spawns the goroutine. This is deterministic;
	// the goroutine's very first tick will observe IsKilled and return.
	_, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	sess := &Session{
		ID:        "test-suppressed-on-kill",
		CLI:       "/bin/sleep",
		State:     StateRunning,
		CreatedAt: time.Now(),
		cmd:       &gopty.Cmd{Process: cmd.Process},
		cancel:    cancelCtx,
		exitCode:  -1,
		killed:    true,
	}

	startExitDetector(sess)

	// After 500ms (5 detector ticks) the child is still running and the
	// detector has had ample opportunity to bail. ExitCode must still
	// be -1 — proof the detector returned without reaping.
	time.Sleep(500 * time.Millisecond)
	if got := sess.ExitCode(); got != -1 {
		t.Fatalf("ExitCode after MarkKilled (detector should have bailed): got %d, want -1", got)
	}
}

// TestStartExitDetector_SignaledExit verifies the POSIX 128+signal
// convention: a child killed by SIGKILL (9) gets exitCode == 137.
func TestStartExitDetector_SignaledExit(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sh -c "kill -9 $$" sends SIGKILL to the shell itself, producing
	// a signaled exit with WIFSIGNALED true and WTERMSIG == 9.
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "/bin/sh",
		Args: []string{"-c", "kill -9 $$"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Kill(sess.ID)
	})

	deadline := time.Now().Add(2 * time.Second)
	var got int
	for time.Now().Before(deadline) {
		got = sess.ExitCode()
		if got != -1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got != 137 {
		t.Fatalf("ExitCode after SIGKILL: got %d, want 137 (128+9 POSIX convention)", got)
	}
}
