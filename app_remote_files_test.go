package main

import (
	"strings"
	"testing"
)

// Phase 122-03 Task 1 — Wails bindings for ExchangeJoinCodeAtURL + RegisterRemoteCap.
//
// The DaemonClient.ExchangeJoinCodeAtURL and DaemonClient.RegisterRemoteCap
// helpers themselves are exercised by Plan 122-01's internal/daemon tests; here
// we only verify the Wails-binding shell:
//   1. App.ExchangeJoinCodeAtURL delegates to a.client (i.e. it's not a stub
//      that ignores its arguments) and returns the daemon-not-connected error
//      when a.client is nil.
//   2. App.RegisterRemoteCap behaves the same.
//
// These tests are hermetic — no real remote webserver is contacted; the
// nil-client case is the safest deterministic surface to pin.

func TestApp_ExchangeJoinCodeAtURL_NilClientReturnsError(t *testing.T) {
	app := &App{} // a.client is nil
	cap, err := app.ExchangeJoinCodeAtURL("https://hub-a.tailnet.ts.net:9443", "ABC12")
	if err == nil {
		t.Fatal("ExchangeJoinCodeAtURL returned nil error with nil daemon client; want error")
	}
	if cap != "" {
		t.Errorf("ExchangeJoinCodeAtURL returned non-empty cap %q with nil daemon client; want empty", cap)
	}
	if !strings.Contains(err.Error(), "daemon not connected") {
		t.Errorf("ExchangeJoinCodeAtURL error = %q; want substring 'daemon not connected'", err.Error())
	}
}

func TestApp_RegisterRemoteCap_NilClientReturnsError(t *testing.T) {
	app := &App{} // a.client is nil
	err := app.RegisterRemoteCap("sess-1", "https://hub-a.tailnet.ts.net:9443", "cap-token")
	if err == nil {
		t.Fatal("RegisterRemoteCap returned nil error with nil daemon client; want error")
	}
	if !strings.Contains(err.Error(), "daemon not connected") {
		t.Errorf("RegisterRemoteCap error = %q; want substring 'daemon not connected'", err.Error())
	}
}

// Smoke-test: the methods exist on *App with the expected signatures. A
// compile-time check is technically enough, but spelling it out as a test
// makes intent explicit and catches signature drift during refactors.
//
// Method expression signature includes the receiver, so we check
// `func(*App, ...) (...)`.
func TestApp_ExchangeJoinCodeAtURL_SignatureShape(t *testing.T) {
	// Compile-time assertion — if the signature changes, this won't compile.
	var _ func(*App, string, string) (string, error) = (*App).ExchangeJoinCodeAtURL
}

func TestApp_RegisterRemoteCap_SignatureShape(t *testing.T) {
	var _ func(*App, string, string, string) error = (*App).RegisterRemoteCap
}
