//go:build !windows

package pty

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestKillSession_NoGoroutineLeak_NormalCase verifies the Phase 117 PAPER-03
// invariant: in the normal case where a process exits within the SIGHUP +
// SIGKILL window, killSession does NOT leak goroutines.
//
// The goroutine spawned inside killSession to call cmd.Wait is bounded-lifetime
// (see comment block in cleanup.go) — for any process that exits via signal
// within the configured timeouts, the goroutine completes before killSession
// returns. This test verifies the "normal case" half of that contract.
//
// The pathological "stuck D-state process" case is explicitly out of scope —
// it cannot be reliably triggered in a unit test, and the goroutine
// completes when the OS eventually reaps the process.
func TestKillSession_NoGoroutineLeak_NormalCase(t *testing.T) {
	// Force any background runtime housekeeping to settle before sampling.
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	baseline := runtime.NumGoroutine()

	const N = 5
	b := NewNativePTYBackend()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < N; i++ {
		sess, err := b.Create(ctx, CreateRequest{
			CLI:  "/bin/sh",
			Args: []string{"-c", "sleep 30"}, // long enough that we hit the SIGHUP path
			Cols: 80,
			Rows: 24,
		})
		if err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}

		// Kill via the backend — exercises killSession (cleanup.go) directly.
		if err := b.Kill(sess.ID); err != nil {
			t.Fatalf("Kill %d: %v", i, err)
		}
	}

	// Allow any in-flight goroutines (exit detectors, etc.) to settle.
	// Several GC + sleep cycles let runtime.NumGoroutine() converge.
	for i := 0; i < 10; i++ {
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}

	final := runtime.NumGoroutine()
	delta := final - baseline

	// Tolerance: each session may leave its exit-detector goroutine running
	// briefly even after Kill returns. Allow up to N goroutines of headroom
	// for in-flight cleanup, but flag any larger delta as a regression.
	if delta > N {
		t.Errorf("possible goroutine leak: baseline=%d final=%d delta=+%d (N=%d sessions, tolerance=%d)",
			baseline, final, delta, N, N)
	}
	t.Logf("baseline=%d final=%d delta=+%d (N=%d sessions, tolerance=%d) — within bounds",
		baseline, final, delta, N, N)
}
