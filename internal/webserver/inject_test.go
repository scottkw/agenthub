package webserver

// Phase 153 Plan 03 — web-share path adversarial inject test.
//
// TestInjectRO_WebPath (SEC-01 web path): proves that a client holding a
// read-only capability JWT (claims.Perms == "read") is rejected server-side
// when it sends a hand-crafted relay.MsgSessionInject frame directly over the
// WebSocket. The rejection must be a relay.MsgInjectError NAK frame AND the
// PTY write counter must read exactly zero.
//
// The RO status under test originates from claims.Perms == "read" on the
// signed JWT — the web-path derivation (server.go line ~1008). This is
// distinct from the relay-path adversarial test (TestInject_ROCap_RelayPath)
// which exercises the ?readonly=1 query param. Testing both entry points is
// required to close Pitfall 5: the relay path being gated while the web path
// is silently left open.
//
// No UI affordance is involved — the frame is sent directly over the WS,
// proving the gate is server-side and cannot be bypassed by client-side
// suppression (SEC-01 / D-04).

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

// writerFuncInj is a func([]byte)(int,error) adapter that implements io.Writer.
// Used to back the hub with a counting PTY writer without importing the relay
// package's unexported writerFunc (package relay, server_inject_test.go).
type writerFuncInj func(p []byte) (int, error)

func (f writerFuncInj) Write(p []byte) (int, error) { return f(p) }

// waitForMsgInjectError reads frames from conn, skipping all non-MsgInjectError
// frames, and returns true when a relay.MsgInjectError frame (first byte ==
// relay.MsgInjectError) arrives. Returns false if the context deadline passes
// without seeing the target frame type.
func waitForMsgInjectError(t *testing.T, conn *websocket.Conn, timeout time.Duration) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			// Deadline expired or connection closed before NAK arrived.
			return false
		}
		if len(rawMsg) > 0 && rawMsg[0] == relay.MsgInjectError {
			return true
		}
		// Skip non-MsgInjectError frames (e.g. MsgMeta, MsgPresence, scrollback).
	}
}

// TestInjectRO_WebPath verifies SEC-01 (web path): a client holding a RO JWT
// (claims.Perms == "read") that sends a hand-crafted relay.MsgSessionInject
// frame directly over the WebSocket:
//  1. receives a relay.MsgInjectError NAK frame within a short timeout, AND
//  2. causes exactly zero PTY writes.
//
// This proves the gate holds server-side on the web entry path regardless of
// any client-side suppression. The RO status originates from claims.Perms in
// the signed JWT (not a URL param — D-24/SEC-04), making this test distinct
// from TestInject_ROCap_RelayPath which uses ?readonly=1 (Pitfall 5).
func TestInjectRO_WebPath(t *testing.T) {
	const sessionID = "inject-ro-web"

	// --- Set up hub manager with a counting PTY writer ---
	var ptyWriteCount atomic.Int32
	countingWriter := writerFuncInj(func(p []byte) (int, error) {
		ptyWriteCount.Add(1)
		return len(p), nil
	})

	manager := relay.NewHubManager()
	ptyOutputR, ptyOutputW := io.Pipe()
	manager.Create(sessionID, ptyOutputR, countingWriter, nil)

	// --- Build a WebServer backed by the manager ---
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

	// --- Mint a read-only capability JWT (claims.Perms == "read") ---
	// issueCapFor is the standard test helper from capability_test_helpers.go.
	// Passing "read" yields claims.Perms == "read", which handleWSSRelay maps
	// to sub.ReadOnly == true (server.go: readonly := claims.Perms == "read").
	token := issueCapFor(t, ws, sessionID, "read")

	// --- Dial the WebSocket with the RO cap (Origin required by middleware) ---
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())
	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/"+sessionID+"/ws?cap="+token, headers)

	// Allow the server time to process the WS upgrade, Subscribe, and send
	// the initial MsgMeta + MsgPresence frames.
	time.Sleep(50 * time.Millisecond)

	// --- Hand-craft a relay.MsgSessionInject frame and send it directly ---
	// This bypasses any client-side suppression, proving server-side enforcement.
	injectPayload, _ := json.Marshal(relay.InjectPayload{Text: "evil command"})
	frame := append([]byte{relay.MsgSessionInject}, injectPayload...)

	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("write inject frame: %v", err)
	}

	// --- Assert (1): relay.MsgInjectError NAK received within timeout ---
	// hub.HandleInject returns ErrReadOnly for a RO subscriber; the web read
	// pump sends MakeInjectErrorFrame(err.Error()) to the subscriber's Msgs
	// channel and the write pump forwards it to the WS client.
	if !waitForMsgInjectError(t, conn, 3*time.Second) {
		t.Error("expected relay.MsgInjectError NAK from RO-JWT inject attempt; " +
			"none received within timeout (SEC-01 web path not enforced)")
	}

	// --- Assert (2): PTY write counter is exactly 0 ---
	// hub.HandleInject returns ErrReadOnly before reaching WriteInput, so the
	// counting PTY writer must record zero calls.
	if count := ptyWriteCount.Load(); count != 0 {
		t.Errorf("PTY write count = %d after RO-JWT inject attempt, want 0 "+
			"(SEC-01 gate failed on web path; HandleInject must return ErrReadOnly "+
			"before WriteInput)", count)
	}
}
