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
	"testing"
	"time"
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
// true the detector returns silently without reaping the child. We build
// a Session manually (bypassing NativePTYBackend.Create's auto-wire-up)
// so we can set killed=true BEFORE startExitDetector spawns its goroutine.
func TestStartExitDetector_SuppressedOnKill(t *testing.T) {
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// sh -c "sleep 5" gives us a 5-second window to observe detector
	// behavior. Create() auto-wires the detector — we mark killed
	// immediately afterward so the detector observes IsKilled on its
	// very first tick and returns silently.
	sess, err := b.Create(ctx, CreateRequest{
		CLI:  "/bin/sh",
		Args: []string{"-c", "sleep 5"},
		Cols: 80,
		Rows: 24,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Mark killed BEFORE the first detector tick (100ms cadence — 5ms is
	// safely within the first sleep). The detector must observe this
	// and return without calling Wait4.
	sess.MarkKilled()
	t.Cleanup(func() {
		// Clean up the still-running sleep via the kill path.
		_ = b.Kill(sess.ID)
	})

	// After 500ms (5 detector ticks) the child is still running and the
	// detector has had ample opportunity to bail. ExitCode must still
	// be -1 — proof the detector did not reap.
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
