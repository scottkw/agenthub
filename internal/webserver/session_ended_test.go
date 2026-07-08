package webserver

// Phase 175 Plan 02 Task 2 — Wave 0 RED scaffold for BUG-02 (#125): no
// disconnect notice on session end.
//
// TestSessionEnd_HubDone_CarriesCloseReason asserts the server-side contract
// 175-06 must implement: when a Hub shuts down (owner kills the session, or
// the session exits naturally), every connected WS client's write pump
// (internal/webserver/server.go:1590-1605, `case <-hub.Done(): return`)
// must close the connection with a code + reason instead of a bare `return`
// (which today falls through to the deferred conn.CloseNow() with no code
// or reason at all — server.go:1431).
//
// This test is SKIPPED until 175-06 lands the fix — see the RESEARCH.md
// "Bug 2 Testability" section, which specifies this exact harness pattern
// (inject_test.go:95-140: real WebServer + relay.HubManager + TLS WS dial,
// then hub.Shutdown() mid-test and assert the client's next conn.Read
// returns a *websocket.CloseError).
//
// The asserted reason is the fixed, generic string "session ended" — never
// a raw internal error — mirroring the IN-01 no-leak convention already
// established at server.go:1574-1579 for inject NAKs (T-175-02-02).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

func TestSessionEnd_HubDone_CarriesCloseReason(t *testing.T) {
	t.Skip("RED until 175-06 sends a close reason on hub.Done() — BUG-02")

	sessionID := "session-ended-test"

	// --- Set up hub manager with a synthetic PTY (no real process needed;
	// the test only cares about hub.Shutdown(), never writes/reads PTY data). ---
	manager := relay.NewHubManager()
	ptyOutputR, ptyOutputW := io.Pipe()
	manager.Create(sessionID, ptyOutputR, io.Discard, nil)

	// --- Build a WebServer backed by the manager (mirrors inject_test.go). ---
	tlsCfg, client := selfSignedTLSForTest(t)
	cfg := Config{
		BindIP:    "127.0.0.1",
		Port:      0,
		FQDN:      "127.0.0.1",
		Mode:      "tailscale",
		TLSConfig: tlsCfg,
	}
	ws, err := NewWebServer(cfg, manager)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	if err := ws.Start(); err != nil {
		t.Fatalf("ws.Start: %v", err)
	}
	ws.SetSigningKey(capTestKey)
	ws.EnableSession(sessionID)

	t.Cleanup(func() {
		_ = ws.Stop()
		manager.Shutdown()
		_ = ptyOutputW.Close()
	})

	// --- Mint a read-only capability JWT and dial a WS client. ---
	token := issueCapFor(t, ws, sessionID, "read")

	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/"+sessionID+"/ws?cap="+token, headers)

	// Allow the server time to process the WS upgrade, Subscribe, and send
	// the initial MsgMeta + MsgPresence frames.
	time.Sleep(50 * time.Millisecond)

	// --- Trigger the session-end path: shut down the hub mid-test. ---
	hub, ok := manager.Get(sessionID)
	if !ok {
		t.Fatalf("manager.Get(%q): session not found", sessionID)
	}
	hub.Shutdown()

	// --- Assert the client's next Read observes a CloseError carrying the
	// expected code + reason (not a bare abrupt close). ---
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, _, err = conn.Read(ctx)
	if err == nil {
		t.Fatalf("conn.Read: expected close error after hub.Shutdown(), got nil error")
	}

	var closeErr websocket.CloseError
	if !errors.As(err, &closeErr) {
		t.Fatalf("conn.Read error = %v (%T), want a *websocket.CloseError (BUG-02: session-end "+
			"close must carry a code + reason, not an abrupt/abnormal close)", err, err)
	}

	if closeErr.Code != websocket.StatusNormalClosure {
		t.Errorf("close code = %v, want %v (websocket.StatusNormalClosure)",
			closeErr.Code, websocket.StatusNormalClosure)
	}
	if closeErr.Reason != "session ended" {
		t.Errorf("close reason = %q, want %q (fixed generic reason — never a raw internal "+
			"error string, per IN-01/T-175-02-02)", closeErr.Reason, "session ended")
	}
}
