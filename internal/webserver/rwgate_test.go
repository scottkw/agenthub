// Package webserver tests for the FNL-09 public read-write consent gate
// primitives introduced in Phase 171 Plan 01:
//
//   - RemoveGrant (surgical single-grant removal, T-171-02 / Pitfall 2)
//   - SetRWGate / isRWGated (D-02 gate state, T-171-05)
//   - TestHandleWSSRelay_WriteCap_RequiresGate (T-171-01, THE critical R4
//     proof that a non-gated write cap is rejected at the REAL WS upgrade,
//     not merely at an originAllowedForWrite unit test — RESEARCH Pitfall 1)
//
// TestOriginAllowedForWrite_RWGate (Task 3, gate-aware originAllowedForWrite)
// lives in this same file per the plan's file manifest.
package webserver

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/relay"
)

// TestRemoveGrant_Surgical asserts RemoveGrant deletes only the named
// grantID, leaving the session's other grants intact (Pitfall 2) — unlike
// ClearGrants, which would wipe every grant for the session.
func TestRemoveGrant_Surgical(t *testing.T) {
	ws, _ := testServer(t)

	const sid = "sess-171-remove"
	ws.AddGrant(sid, "grant-a")
	ws.AddGrant(sid, "grant-b")

	ws.RemoveGrant(sid, "grant-a")

	if ws.isGrantActive(sid, "grant-a") {
		t.Error("expected grant-a removed after RemoveGrant")
	}
	if !ws.isGrantActive(sid, "grant-b") {
		t.Error("expected grant-b to remain active — RemoveGrant must not touch sibling grants")
	}
}

// TestRemoveGrant_AbsentGrantIsNoOp asserts RemoveGrant of a grant that was
// never added (or already removed) does not panic and leaves the manager
// usable.
func TestRemoveGrant_AbsentGrantIsNoOp(t *testing.T) {
	ws, _ := testServer(t)

	const sid = "sess-171-remove-noop"
	ws.RemoveGrant(sid, "never-added") // must not panic

	ws.AddGrant(sid, "grant-real")
	if !ws.isGrantActive(sid, "grant-real") {
		t.Error("expected manager to remain usable after no-op RemoveGrant")
	}
}

// TestRemoveGrant_AbsentSessionIsNoOp asserts RemoveGrant against a session
// that has no grants map entry at all (never AddGrant'd) is a safe no-op.
func TestRemoveGrant_AbsentSessionIsNoOp(t *testing.T) {
	ws, _ := testServer(t)
	ws.RemoveGrant("sess-never-seen", "grant-x") // must not panic
}

// TestSetRWGate_DefaultFalse asserts isRWGated defaults to false for a
// session that has never had SetRWGate called.
func TestSetRWGate_DefaultFalse(t *testing.T) {
	ws, _ := testServer(t)
	if ws.isRWGated("sess-171-gate-default") {
		t.Error("expected isRWGated to default to false")
	}
}

// TestSetRWGate_TrueThenFalse asserts SetRWGate(true) flips isRWGated to
// true, and SetRWGate(false) clears it back to false (delete, not a
// lingering true value).
func TestSetRWGate_TrueThenFalse(t *testing.T) {
	ws, _ := testServer(t)
	const sid = "sess-171-gate-toggle"

	ws.SetRWGate(sid, true)
	if !ws.isRWGated(sid) {
		t.Fatal("expected isRWGated true after SetRWGate(sid, true)")
	}

	ws.SetRWGate(sid, false)
	if ws.isRWGated(sid) {
		t.Error("expected isRWGated false after SetRWGate(sid, false)")
	}
}

// TestHandleWSSRelay_WriteCap_RequiresGate is the critical R4 proof
// (RESEARCH Pitfall 1, T-171-01): a write-permission capability whose
// grant_id has NOT been registered via AddGrant must be rejected at the
// REAL GET /sessions/{id}/ws upgrade handler — not merely by an
// originAllowedForWrite unit test, which does not reach handleWSSRelay at
// all. Once the grant is registered, the identical token completes the
// upgrade and its write permission actually reaches the PTY (asserted by
// observing an input frame land on the hub's input pipe).
func TestHandleWSSRelay_WriteCap_RequiresGate(t *testing.T) {
	ws, client, _, inputReader := testServerWithHub(t, "sess-171-wsgate")
	ws.SetSigningKey(capTestKey)

	const sid = "sess-171-wsgate"
	claims := capability.Claims{
		SID:     sid,
		Perms:   "read,write",
		IAT:     time.Now().Unix(),
		GrantID: "grant-171-wsgate-write",
		V:       1,
	}
	token, err := capability.Sign(claims, capTestKey)
	if err != nil {
		t.Fatalf("capability.Sign: %v", err)
	}

	// Deliberately do NOT call ws.AddGrant here — the grant is unregistered.
	wsURL := ws.BaseURL() + "/sessions/" + sid + "/ws?cap=" + token
	req, err := http.NewRequest("GET", wsURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Origin", ws.BaseURL())
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 403/401 for a write cap whose grant is not registered, got %d", resp.StatusCode)
	}

	// Register the grant and retry — the upgrade must now succeed AND the
	// write permission must actually reach the PTY (sub.ReadOnly == false).
	ws.AddGrant(sid, claims.GrantID)

	conn := dialWebServerWS(t, client, ws.BaseURL(),
		"/sessions/"+sid+"/ws?cap="+token, originHeader(ws.BaseURL()))
	if conn == nil {
		t.Fatal("expected successful upgrade once the grant is registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	want := []byte("gated-write\r")
	if err := conn.Write(ctx, websocket.MessageBinary, relay.MakeInputFrame(want)); err != nil {
		t.Fatalf("write input frame: %v", err)
	}
	got := readPipeWithTimeout(t, inputReader, 1*time.Second)
	if !bytes.Equal(got, want) {
		t.Errorf("expected write cap to reach PTY once gated (sub.ReadOnly==false): want %q got %q", want, got)
	}
}
