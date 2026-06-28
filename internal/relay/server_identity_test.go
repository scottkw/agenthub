package relay

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// setupIdentityTestServer creates a test relay.Server wired with an in-memory
// alias store (no daemon import). The identity provider treats "host" as the
// owner default alias.
//
// Returns the httptest.Server, HubManager, and the test session ID.
// Callers must not close ptyOutputW — cleanup is registered via t.Cleanup.
func setupIdentityTestServer(t *testing.T) (*httptest.Server, *HubManager, string) {
	t.Helper()

	// In-memory alias store — stands in for daemon.AliasStore without importing daemon.
	var aliasMu sync.Mutex
	aliases := make(map[string]string)
	getAlias := func(personKey, def string) string {
		aliasMu.Lock()
		defer aliasMu.Unlock()
		if v, ok := aliases[personKey]; ok {
			return v
		}
		return def
	}
	setAlias := func(personKey, alias string) {
		aliasMu.Lock()
		defer aliasMu.Unlock()
		aliases[personKey] = alias
	}

	// PTY output pipe — not used by identity tests but required by Hub.
	ptyOutputR, ptyOutputW := io.Pipe()

	const sessionID = "identity-session"
	manager := NewHubManager()
	manager.Create(sessionID, ptyOutputR, io.Discard, nil)

	srv := NewServer(manager, nil, nil)
	srv.SetIdentityProviders("host", getAlias, setAlias)

	ts := httptest.NewServer(srv)
	t.Cleanup(func() {
		ts.Close()
		manager.Shutdown()
		ptyOutputW.Close()
	})
	return ts, manager, sessionID
}

// dialIdentityWS dials a WebSocket to the identity test server with optional
// query string (e.g. "readonly=1").
func dialIdentityWS(t *testing.T, ts *httptest.Server, sessionID, query string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/sessions/" + sessionID + "/ws"
	if query != "" {
		wsURL += "?" + query
	}
	conn, _, err := websocket.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dialIdentityWS: %v", err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

// waitForPresenceFrame reads frames from conn (skipping MsgMeta and other
// non-presence frames) and returns the decoded PresencePayload of the first
// MsgPresence frame seen. Fails the test on timeout or decode error.
// Timeout: 3 seconds (sub-second in practice — frames are pushed synchronously).
func waitForPresenceFrame(t *testing.T, conn *websocket.Conn, label string) PresencePayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("waitForPresenceFrame(%s): %v", label, err)
		}
		msgType, payload, pErr := ParseFrame(rawMsg)
		if pErr != nil {
			continue // malformed frame — ignore
		}
		if msgType != MsgPresence {
			continue // skip MsgMeta and any other non-presence frames
		}
		var pp PresencePayload
		if uErr := json.Unmarshal(payload, &pp); uErr != nil {
			t.Fatalf("waitForPresenceFrame(%s): unmarshal: %v", label, uErr)
		}
		return pp
	}
}

// waitForTypingFrame reads frames from conn (skipping everything except
// MsgTyping) and returns the decoded TypingPayload. Fails the test on timeout.
func waitForTypingFrame(t *testing.T, conn *websocket.Conn, label string) TypingPayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("waitForTypingFrame(%s): %v", label, err)
		}
		msgType, payload, pErr := ParseFrame(rawMsg)
		if pErr != nil {
			continue
		}
		if msgType != MsgTyping {
			continue
		}
		var tp TypingPayload
		if uErr := json.Unmarshal(payload, &tp); uErr != nil {
			t.Fatalf("waitForTypingFrame(%s): unmarshal: %v", label, uErr)
		}
		return tp
	}
}

// assertNoTypingFrame verifies that conn does NOT receive a MsgTyping frame
// within the given deadline. Non-typing frames received during the window
// are silently discarded (they are not a test failure).
func assertNoTypingFrame(t *testing.T, conn *websocket.Conn, within time.Duration, label string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), within)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			// Deadline exceeded or connection closed — no typing frame received; pass.
			return
		}
		msgType, _, pErr := ParseFrame(rawMsg)
		if pErr != nil {
			continue
		}
		if msgType == MsgTyping {
			t.Errorf("assertNoTypingFrame(%s): received unexpected MsgTyping (sender exclusion broken)", label)
			return
		}
		// Non-typing frame — keep draining until deadline.
	}
}

// presenceAliasFor returns the alias for the given personKey from the roster,
// or "" if the key is not present.
func presenceAliasFor(pp PresencePayload, personKey string) string {
	for _, e := range pp.Participants {
		if e.PersonKey == personKey {
			return e.Alias
		}
	}
	return ""
}

// waitForSelfFrame reads frames from conn (skipping MsgMeta, MsgPresence, and
// other non-self frames) and returns the decoded SelfPayload of the first
// MsgSelf frame seen. Fails the test on timeout or decode error.
func waitForSelfFrame(t *testing.T, conn *websocket.Conn, label string) SelfPayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for {
		_, rawMsg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("waitForSelfFrame(%s): %v", label, err)
		}
		msgType, payload, pErr := ParseFrame(rawMsg)
		if pErr != nil {
			continue
		}
		if msgType != MsgSelf {
			continue
		}
		var sp SelfPayload
		if uErr := json.Unmarshal(payload, &sp); uErr != nil {
			t.Fatalf("waitForSelfFrame(%s): unmarshal: %v", label, uErr)
		}
		return sp
	}
}

// TestRelayIdentity_SelfFrameOnConnect verifies that the relay (desktop) path
// emits a MsgSelf (0x37) frame on connect carrying personKey "local:local" and
// the resolved owner alias. The self frame must arrive before or alongside the
// presence frame — the test drains until it sees a MsgSelf frame.
func TestRelayIdentity_SelfFrameOnConnect(t *testing.T) {
	ts, _, sessionID := setupIdentityTestServer(t)

	conn := dialIdentityWS(t, ts, sessionID, "")
	sp := waitForSelfFrame(t, conn, "relay self frame on connect")

	if sp.PersonKey != "local:local" {
		t.Errorf("relay self frame: PersonKey = %q, want 'local:local'", sp.PersonKey)
	}
	if sp.Alias != "host" {
		t.Errorf("relay self frame: Alias = %q, want 'host' (ownerDefaultAlias)", sp.Alias)
	}
}

// TestRelayIdentity_AliasPropagation verifies IDENT-02: a MsgAliasSet frame
// from one client propagates to all relay clients as a MsgPresence roster
// update within one round-trip.
func TestRelayIdentity_AliasPropagation(t *testing.T) {
	ts, _, sessionID := setupIdentityTestServer(t)

	connA := dialIdentityWS(t, ts, sessionID, "")
	// Drain A's initial MsgPresence (from first Subscribe+NotifyPresence).
	ppA := waitForPresenceFrame(t, connA, "A initial")
	// Owner entry must be present with default alias "host".
	if got := presenceAliasFor(ppA, "local:local"); got != "host" {
		t.Errorf("A initial: personKey 'local:local' alias = %q, want 'host'", got)
	}

	connB := dialIdentityWS(t, ts, sessionID, "")
	// When B joins: A receives a new MsgPresence (ConnCount 1→2); B receives its initial MsgPresence.
	_ = waitForPresenceFrame(t, connA, "A update when B joined")
	_ = waitForPresenceFrame(t, connB, "B initial")

	// Client A sends MsgAliasSet{"alias":"ken"}.
	aliasFrame := MakeAliasSetFrame(AliasPayload{Alias: "ken"})
	ctx := context.Background()
	if err := connA.Write(ctx, websocket.MessageBinary, aliasFrame); err != nil {
		t.Fatalf("connA write alias: %v", err)
	}

	// Both A and B must receive a MsgPresence with alias "ken" for "local:local".
	ppA2 := waitForPresenceFrame(t, connA, "alias propagation to A")
	ppB2 := waitForPresenceFrame(t, connB, "alias propagation to B")

	if got := presenceAliasFor(ppA2, "local:local"); got != "ken" {
		t.Errorf("A: alias after set = %q, want 'ken'; roster: %+v", got, ppA2.Participants)
	}
	if got := presenceAliasFor(ppB2, "local:local"); got != "ken" {
		t.Errorf("B: alias after set = %q, want 'ken'; roster: %+v", got, ppB2.Participants)
	}
}

// TestRelayIdentity_TypingExcludesSender verifies PRESENCE-02: a MsgTyping
// frame from client A is delivered to client B (the recipient) but NOT echoed
// back to client A (the sender).
func TestRelayIdentity_TypingExcludesSender(t *testing.T) {
	ts, _, sessionID := setupIdentityTestServer(t)

	connA := dialIdentityWS(t, ts, sessionID, "")
	_ = waitForPresenceFrame(t, connA, "A initial")

	connB := dialIdentityWS(t, ts, sessionID, "")
	_ = waitForPresenceFrame(t, connA, "A update when B joined")
	_ = waitForPresenceFrame(t, connB, "B initial")

	// A sends typing:true.
	typingFrame := MakeTypingFrame(TypingPayload{Typing: true})
	ctx := context.Background()
	if err := connA.Write(ctx, websocket.MessageBinary, typingFrame); err != nil {
		t.Fatalf("connA write typing: %v", err)
	}

	// B must receive MsgTyping with typing:true and the sender's personKey.
	tp := waitForTypingFrame(t, connB, "B receives typing from A")
	if !tp.Typing {
		t.Errorf("B: typing payload typing = %v, want true", tp.Typing)
	}
	if tp.PersonKey != "local:local" {
		t.Errorf("B: typing payload personKey = %q, want 'local:local'", tp.PersonKey)
	}

	// A must NOT receive its own typing frame (sender-exclusion / T-152-03).
	assertNoTypingFrame(t, connA, 200*time.Millisecond, "A should not receive own typing")
}

// TestRelayIdentity_ReadOnlyCanChat verifies D-06: a read-only client can
// send MsgAliasSet and have the alias accepted and propagated. Only MsgInput
// remains gated on sub.ReadOnly.
func TestRelayIdentity_ReadOnlyCanChat(t *testing.T) {
	ts, _, sessionID := setupIdentityTestServer(t)

	roConn := dialIdentityWS(t, ts, sessionID, "readonly=1")
	// Drain initial presence.
	ppInitial := waitForPresenceFrame(t, roConn, "RO initial")
	if presenceAliasFor(ppInitial, "local:local") == "" {
		t.Logf("RO initial presence: %+v", ppInitial)
	}

	// RO client sends MsgAliasSet (must NOT be blocked by ReadOnly guard).
	aliasFrame := MakeAliasSetFrame(AliasPayload{Alias: "rouser"})
	ctx := context.Background()
	if err := roConn.Write(ctx, websocket.MessageBinary, aliasFrame); err != nil {
		t.Fatalf("roConn write alias: %v", err)
	}

	// RO client must receive a MsgPresence confirming the alias was accepted.
	pp := waitForPresenceFrame(t, roConn, "RO alias propagation")
	if got := presenceAliasFor(pp, "local:local"); got != "rouser" {
		t.Errorf("RO client: alias after set = %q, want 'rouser'; roster: %+v", got, pp.Participants)
	}
}
