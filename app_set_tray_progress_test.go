package main

import (
	"strings"
	"testing"
)

// newTestApp returns an App with lastTrayQuartile pre-stored to the given
// initial value. Centralizes the atomic.Int32 initialization so individual
// tests don't have to know the field's storage type.
func newTestApp(initialQuartile int32, refreshTrayStateFunc func()) *App {
	a := &App{
		trayInit:             true,
		refreshTrayStateFunc: refreshTrayStateFunc,
	}
	a.lastTrayQuartile.Store(initialQuartile)
	return a
}

// TestApp_SetTrayProgress_Idempotent verifies that calling SetTrayProgress
// with the same quartile twice invokes the tray update only once.
//
// Phase 98 PRG-03 / Pitfall #3 (transition guard) — idempotency check.
// Uses the refreshTrayStateFunc injection (Phase 98 / PROJECT.md
// "Function injection" pattern) to avoid cgo/D-Bus/Win32 calls in tests.
func TestApp_SetTrayProgress_Idempotent(t *testing.T) {
	calls := 0
	a := newTestApp(-1, func() { calls++ })

	// First call — quartile differs from -1, should invoke the refresh.
	if err := a.SetTrayProgress(2); err != nil {
		t.Fatalf("first SetTrayProgress(2) returned unexpected error: %v", err)
	}
	if got := a.lastTrayQuartile.Load(); got != 2 {
		t.Errorf("after first call: want lastTrayQuartile=2, got %d", got)
	}
	if calls != 1 {
		t.Errorf("after first call: want refreshTrayState called 1 time, got %d", calls)
	}

	// Second call — same quartile; must be a no-op (idempotency check).
	if err := a.SetTrayProgress(2); err != nil {
		t.Fatalf("second SetTrayProgress(2) returned unexpected error: %v", err)
	}
	if got := a.lastTrayQuartile.Load(); got != 2 {
		t.Errorf("after second call: want lastTrayQuartile=2, got %d", got)
	}
	if calls != 1 {
		t.Errorf("after second call: refreshTrayState should NOT have been called again; got calls=%d", calls)
	}
}

// TestApp_SetTrayProgress_BoundsCheck verifies that quartile values outside
// [0,4] return a non-nil error and do not mutate state.
//
// Phase 98 PRG-03 / T-98-02 (Tampering — bounds check mitigates frontend bug
// passing arbitrary quartile). Mirrors the error-wrapping convention of
// SaveTerminalSession (Phase 97 SER-01).
func TestApp_SetTrayProgress_BoundsCheck(t *testing.T) {
	a := newTestApp(-1, func() {})

	// Negative out-of-range.
	err := a.SetTrayProgress(-1)
	if err == nil {
		t.Fatal("SetTrayProgress(-1): expected non-nil error, got nil")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("SetTrayProgress(-1): error should contain 'out of range', got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "-1") {
		t.Errorf("SetTrayProgress(-1): error should contain '-1', got %q", err.Error())
	}
	if got := a.lastTrayQuartile.Load(); got != -1 {
		t.Errorf("SetTrayProgress(-1): state mutated — want lastTrayQuartile=-1, got %d", got)
	}

	// Positive out-of-range.
	err = a.SetTrayProgress(5)
	if err == nil {
		t.Fatal("SetTrayProgress(5): expected non-nil error, got nil")
	}
	if !strings.Contains(err.Error(), "5") {
		t.Errorf("SetTrayProgress(5): error should contain '5', got %q", err.Error())
	}
	if got := a.lastTrayQuartile.Load(); got != -1 {
		t.Errorf("SetTrayProgress(5): state mutated — want lastTrayQuartile=-1, got %d", got)
	}
}

// TestApp_SetTrayProgress_ErrorPrecedence verifies that trayIconBytesForState
// returns trayIconErrorBytes when connected=false, regardless of lastTrayQuartile.
//
// Phase 98 PRG-03 / T-98-08 (error-state precedence, Pitfall #8) — progress
// glyph must not mask daemon-disconnect state. Tests the helper directly to
// avoid any platform-tray API invocation. This test runs on the host GOOS;
// the helper is defined identically in all three platform files so the same
// assertion holds on any build target.
func TestApp_SetTrayProgress_ErrorPrecedence(t *testing.T) {
	// Apply quartile 3 (75% progress glyph).
	a := newTestApp(3, func() {})

	// When disconnected, must return error bytes regardless of quartile.
	got := a.trayIconBytesForState(false)
	if &got[0] != &trayIconErrorBytes[0] {
		t.Errorf("trayIconBytesForState(connected=false, quartile=3): expected trayIconErrorBytes, got different slice")
	}

	// When connected with quartile 3, must return progress-75 bytes.
	got = a.trayIconBytesForState(true)
	if &got[0] != &trayIconProgress75Bytes[0] {
		t.Errorf("trayIconBytesForState(connected=true, quartile=3): expected trayIconProgress75Bytes, got different slice")
	}

	// Reset to quartile 0 (no active progress) — connected path should return base icon.
	a.lastTrayQuartile.Store(0)
	got = a.trayIconBytesForState(true)
	if &got[0] != &trayIconBytes[0] {
		t.Errorf("trayIconBytesForState(connected=true, quartile=0): expected trayIconBytes, got different slice")
	}
}
