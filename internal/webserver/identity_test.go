package webserver

// Web-path identity, presence, and parity tests (Phase 152 / IDENT-01,
// IDENT-02, PRESENCE-01, PRESENCE-02, D-06).
//
// These tests live in package webserver (internal) so they can use the
// existing test helpers (testServerWithHub, dialWebServerWS, issueCapFor,
// capTestKey) without re-exporting them.
//
// Invariants exercised:
//   - WhoIs failure → non-"local" personKey (proves owner/web disambiguation,
//     criterion 5, without a live tailnet)
//   - MsgAliasSet over web path → propagates a MsgPresence roster with new alias
//   - Read-only web client can set alias and type (D-06)

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/relay"
)

// inMemAliasMap is a thread-safe in-memory alias provider for tests.
type inMemAliasMap struct {
	mu      sync.RWMutex
	aliases map[string]string
}

func newInMemAliasMap() *inMemAliasMap {
	return &inMemAliasMap{aliases: make(map[string]string)}
}

func (m *inMemAliasMap) get(personKey, def string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.aliases[personKey]; ok {
		return v
	}
	return def
}

func (m *inMemAliasMap) set(personKey, alias string) {
	m.mu.Lock()
	m.aliases[personKey] = alias
	m.mu.Unlock()
}

func (m *inMemAliasMap) getMap() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]string, len(m.aliases))
	for k, v := range m.aliases {
		out[k] = v
	}
	return out
}

// drainUntilPresence reads from conn until a MsgPresence frame arrives or the
// context is cancelled. Non-MsgPresence frames (MsgMeta, scrollback, etc.) are
// silently skipped. Fails the test if the context expires first.
func drainUntilPresence(t *testing.T, conn *websocket.Conn, timeout time.Duration) relay.PresencePayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("drainUntilPresence: conn.Read: %v", err)
		}
		msgType, payload, err := relay.ParseFrame(msg)
		if err != nil {
			continue
		}
		if msgType != relay.MsgPresence {
			continue
		}
		var p relay.PresencePayload
		if err := json.Unmarshal(payload, &p); err != nil {
			t.Fatalf("drainUntilPresence: json.Unmarshal: %v", err)
		}
		return p
	}
}

// dialIdentityWS is a test helper that creates a testServerWithHub, wires alias
// providers, issues a capability, and opens a WebSocket to the session relay
// endpoint. Returns the server, connection, and alias map for inspection.
func dialIdentityWS(
	t *testing.T,
	sessionID string,
	perms string,
	aliasMap *inMemAliasMap,
) (*WebServer, *websocket.Conn) {
	t.Helper()
	ws, client, _, _ := testServerWithHub(t, sessionID)
	ws.SetSigningKey(capTestKey)
	if aliasMap != nil {
		ws.SetAliasProviders(aliasMap.get, aliasMap.set)
	}

	token := issueCapFor(t, ws, sessionID, perms)
	headers := http.Header{}
	headers.Set("Origin", ws.BaseURL())

	path := "/sessions/" + sessionID + "/ws?cap=" + token
	conn := dialWebServerWS(t, client, ws.BaseURL(), path, headers)
	return ws, conn
}

// drainUntilSelf reads from conn until a MsgSelf frame arrives or the context
// is cancelled. Non-MsgSelf frames are silently skipped. Fails the test if the
// context expires first.
func drainUntilSelf(t *testing.T, conn *websocket.Conn, timeout time.Duration) relay.SelfPayload {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("drainUntilSelf: conn.Read: %v", err)
		}
		msgType, payload, err := relay.ParseFrame(msg)
		if err != nil {
			continue
		}
		if msgType != relay.MsgSelf {
			continue
		}
		var sp relay.SelfPayload
		if err := json.Unmarshal(payload, &sp); err != nil {
			t.Fatalf("drainUntilSelf: json.Unmarshal: %v", err)
		}
		return sp
	}
}

// TestWebIdentity_SelfFrameOnConnect verifies that the web path emits a
// MsgSelf (0x37) frame on connect carrying a personKey ending in ":web" and
// the resolved alias (ComputedName or fallback) for the connecting client.
func TestWebIdentity_SelfFrameOnConnect(t *testing.T) {
	aliasMap := newInMemAliasMap()
	_, conn := dialIdentityWS(t, "sess-self-01", "read,write", aliasMap)

	sp := drainUntilSelf(t, conn, 5*time.Second)

	if !strings.HasSuffix(sp.PersonKey, ":web") {
		t.Errorf("web self frame: PersonKey = %q, want suffix ':web'", sp.PersonKey)
	}
	if sp.PersonKey == "local:local" {
		t.Errorf("web self frame: PersonKey must not be 'local:local' (D-04 / owner disambiguation)")
	}
	// Alias may be empty on WhoIs failure path (no live tailnet in tests); just
	// verify the field is present (not a required non-empty value in this path).
	t.Logf("web self frame: personKey=%q alias=%q", sp.PersonKey, sp.Alias)
}

// TestWebIdentity_ReadOnlySelfFrame verifies that a read-only (perms="read")
// web client also receives the MsgSelf frame on connect — the frame must NOT
// be gated on write permissions.
func TestWebIdentity_ReadOnlySelfFrame(t *testing.T) {
	aliasMap := newInMemAliasMap()
	_, conn := dialIdentityWS(t, "sess-self-ro-01", "read", aliasMap)

	sp := drainUntilSelf(t, conn, 5*time.Second)

	if !strings.HasSuffix(sp.PersonKey, ":web") {
		t.Errorf("RO web self frame: PersonKey = %q, want suffix ':web'", sp.PersonKey)
	}
	t.Logf("RO web self frame: personKey=%q alias=%q", sp.PersonKey, sp.Alias)
}

// TestWebIdentity_WhoIsFailureFallback verifies that in the absence of a live
// tailnet (WhoIs failure path, as in all automated tests), handleWSSRelay
// stamps the subscriber with a personKey that:
//   - is NOT "local:local" (the owner's composite key) — criterion 5 / D-04
//   - ends in ":web" (Origin stamping — IDENT-01)
//
// The MsgPresence frame broadcast on connect carries the full roster, which
// lets us inspect the stamped values without a live tailnet.
func TestWebIdentity_WhoIsFailureFallback(t *testing.T) {
	aliasMap := newInMemAliasMap()
	_, conn := dialIdentityWS(t, "sess-ident-01", "read,write", aliasMap)

	presence := drainUntilPresence(t, conn, 5*time.Second)

	if len(presence.Participants) == 0 {
		t.Fatal("expected at least one participant in presence roster after connect")
	}

	// Find our own entry. There should be exactly one — the connecting client.
	var our *relay.PresenceEntry
	for i := range presence.Participants {
		e := &presence.Participants[i]
		if strings.HasSuffix(e.PersonKey, ":web") {
			our = e
			break
		}
	}
	if our == nil {
		t.Fatalf("no :web entry in presence roster; participants=%+v", presence.Participants)
	}

	// Criterion 5: the web entry must not be "local:local".
	if our.PersonKey == "local:local" {
		t.Errorf("web client must not share personKey 'local:local' with the owner (criterion 5 / D-04); got %q", our.PersonKey)
	}

	// IDENT-01: Origin must be "web".
	if our.Origin != "web" {
		t.Errorf("Origin = %q; want %q", our.Origin, "web")
	}

	// IDENT-01: TailnetID must NOT be "local" (WhoIs fallback used "unknown").
	if our.TailnetID == "local" {
		t.Errorf("TailnetID = %q; must not be 'local' on WhoIs failure path", our.TailnetID)
	}
}

// TestWebAliasPropagation verifies that a MsgAliasSet frame over the web path
// (IDENT-02):
//  1. Causes a subsequent MsgPresence broadcast with the updated alias.
//  2. Persists the alias in the alias provider (in-memory map for tests).
func TestWebAliasPropagation(t *testing.T) {
	aliasMap := newInMemAliasMap()
	_, conn := dialIdentityWS(t, "sess-alias-01", "read,write", aliasMap)

	// Drain the initial presence broadcast (from the join event).
	initialPresence := drainUntilPresence(t, conn, 5*time.Second)
	if len(initialPresence.Participants) == 0 {
		t.Fatal("expected at least one participant in initial presence roster")
	}

	// Find our entry to get the personKey.
	var ourKey string
	for _, e := range initialPresence.Participants {
		if strings.HasSuffix(e.PersonKey, ":web") {
			ourKey = e.PersonKey
			break
		}
	}
	if ourKey == "" {
		t.Fatalf("no :web entry in initial roster; got %+v", initialPresence.Participants)
	}

	// Send a MsgAliasSet frame to update our alias.
	const wantAlias = "alice"
	frame := relay.MakeAliasSetFrame(relay.AliasPayload{Alias: wantAlias})
	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("conn.Write MsgAliasSet: %v", err)
	}

	// Drain until the next MsgPresence broadcast (alias update triggers one).
	updatedPresence := drainUntilPresence(t, conn, 5*time.Second)

	var updated *relay.PresenceEntry
	for i := range updatedPresence.Participants {
		e := &updatedPresence.Participants[i]
		if e.PersonKey == ourKey {
			updated = e
			break
		}
	}
	if updated == nil {
		t.Fatalf("personKey %q not found in updated presence roster; got %+v", ourKey, updatedPresence.Participants)
	}
	if updated.Alias != wantAlias {
		t.Errorf("Alias = %q; want %q", updated.Alias, wantAlias)
	}

	// Verify the alias was persisted to the in-memory alias provider.
	if got := aliasMap.getMap()[ourKey]; got != wantAlias {
		t.Errorf("alias provider for %q = %q; want %q", ourKey, got, wantAlias)
	}
}

// TestWebReadOnlyCanChat verifies that a read-only (perms="read") web client
// is a full chat participant (D-06): it can send MsgAliasSet (and by extension
// MsgTyping — covered by the same gate) and the alias change propagates.
//
// This test mirrors TestWebAliasPropagation but uses a read-only capability.
// The critical assertion is that the server does NOT discard the MsgAliasSet
// due to ReadOnly status — only MsgInput remains gated on ReadOnly.
func TestWebReadOnlyCanChat(t *testing.T) {
	aliasMap := newInMemAliasMap()
	_, conn := dialIdentityWS(t, "sess-ro-chat-01", "read", aliasMap)

	// Drain the initial join presence broadcast.
	initialPresence := drainUntilPresence(t, conn, 5*time.Second)

	var ourKey string
	for _, e := range initialPresence.Participants {
		if strings.HasSuffix(e.PersonKey, ":web") {
			ourKey = e.PersonKey
			break
		}
	}
	if ourKey == "" {
		t.Fatalf("no :web entry in initial roster (RO test); got %+v", initialPresence.Participants)
	}

	// As a read-only client, send MsgAliasSet. If D-06 is implemented
	// correctly the server processes it (outside the ReadOnly gate) and
	// broadcasts a MsgPresence with the new alias.
	const wantAlias = "rob"
	frame := relay.MakeAliasSetFrame(relay.AliasPayload{Alias: wantAlias})
	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
		t.Fatalf("conn.Write MsgAliasSet (read-only client): %v", err)
	}

	// Expect the updated presence roster.
	updatedPresence := drainUntilPresence(t, conn, 5*time.Second)

	var updated *relay.PresenceEntry
	for i := range updatedPresence.Participants {
		e := &updatedPresence.Participants[i]
		if e.PersonKey == ourKey {
			updated = e
			break
		}
	}
	if updated == nil {
		t.Fatalf("personKey %q not found in updated presence roster after RO alias-set; got %+v",
			ourKey, updatedPresence.Participants)
	}
	if updated.Alias != wantAlias {
		t.Errorf("RO client alias = %q; want %q (D-06: RO clients are full chat participants)", updated.Alias, wantAlias)
	}
}
