//go:build phase87_wave2

// Package webserver capability-enforcement tests (RED skeletons for Plan 03).
//
// These nine tests assert the boundary contract SEC-02..SEC-05 from
// REQUIREMENTS.md. Every test body calls t.Skip("implemented in plan 03")
// so the file compiles with the phase87_wave2 tag but executes no behavior
// until Plan 03 wires capability.Verify / requireCapability into
// internal/webserver/server.go and un-skips each test.
//
// These skeletons provide the automated verify target that every subsequent
// SEC-XX task in the phase points at — no "MISSING Wave 0" placeholders
// survive past this plan (Nyquist sampling requirement).
package webserver

import (
	"testing"
)

// TestSecurity_UnauthenticatedClientCannotEnumerateSessions is the inverted
// form of the security-review's TestSecurity_UnauthenticatedClientCanEnumerate
// scaffold. SEC-02: GET /api/sessions without a valid cap must return 401
// (or 403) rather than 200 with a session list. After Plan 03 wires
// requireCapability onto this route, the test asserts StatusUnauthorized.
func TestSecurity_UnauthenticatedClientCannotEnumerateSessions(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   ws, client := testServer(t)
	//   ws.EnableSession("sess-authz")
	//   resp, _ := client.Get(ws.BaseURL() + "/api/sessions")
	//   want resp.StatusCode == http.StatusUnauthorized
}

// TestSecurity_WrongSessionCapRejected covers SEC-03: a capability bound to
// session A must NOT grant access to session B's WebSocket upgrade. Plan 03
// checks claims.SID against r.PathValue("id") inside requireCapability and
// returns 403 on mismatch.
func TestSecurity_WrongSessionCapRejected(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   ws, client, _, _ := testServerWithHub(t, "sess-A")
	//   ws.EnableSession("sess-B")
	//   issue a cap for sess-A; attempt GET /sessions/sess-B/ws?cap=<A-cap>
	//   want StatusForbidden
}

// TestSecurity_ReadOnlyParamCannotGrantWrite covers SEC-04: the legacy
// ?readonly=1 query param is removed from the write path (D-23). A caller
// with a read-only capability cannot promote themselves to write by
// omitting or fabricating the readonly param — the perms claim is the only
// source of truth.
func TestSecurity_ReadOnlyParamCannotGrantWrite(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   dial WS with read-only cap (perms="read"); attempt to write MsgInput
	//   with and without ?readonly= variants; assert PTY pipe receives zero
	//   bytes via readPipeWithTimeout.
}

// TestSecurity_ReadOnlyCapabilityBlocksMsgInput covers SEC-05: the relay
// (via sub.ReadOnly sourced from claims.Perms) drops MsgInput frames from
// a subscriber whose capability lacks write permission. This is the
// straightforward read-only enforcement test.
func TestSecurity_ReadOnlyCapabilityBlocksMsgInput(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   dial WS with read-only cap; write MakeInputFrame([]byte("x\n"));
	//   readPipeWithTimeout(inputRead, 500ms) must timeout (no bytes reach PTY)
}

// TestSecurity_ReconnectWithoutReadonlyStillBlocked covers the SEC-05
// regression scenario that motivated the whole phase: a client with a
// read-only capability cannot upgrade themselves to write by disconnecting
// and reconnecting WITHOUT ?readonly=1. With client-asserted readonly
// removed (D-23/D-24), the perms claim binds readonly to the token
// server-side. The flow:
//
//   1. Dial /sessions/{id}/ws?cap=<read-only cap>  (no readonly= param)
//   2. Send a relay.MakeInputFrame([]byte("should-be-blocked\n"))
//   3. readPipeWithTimeout on the inputCaptureReader must TIME OUT —
//      the PTY pipe receives ZERO bytes. A timeout is the positive signal.
//
// This is the test that was green (buggy) against v3.0 and must be green
// (fixed) after Plan 03; that inversion is the heart of SEC-05.
func TestSecurity_ReconnectWithoutReadonlyStillBlocked(t *testing.T) {
	t.Skip("implemented in plan 03")
}

// TestCapability_MissingCapReturns401 covers the missing-?cap= path in
// requireCapability: no token at all returns 401 (not 403) to distinguish
// "you didn't send credentials" from "your credentials are wrong."
func TestCapability_MissingCapReturns401(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   ws, client := testServer(t); ws.EnableSession("s-1")
	//   resp, _ := client.Get(ws.BaseURL() + "/api/sessions")  // no ?cap=
	//   want StatusUnauthorized
}

// TestCapability_InvalidSignatureReturns401 covers the bad-signature path:
// a token that parses but fails HMAC verification also returns 401
// (same outward-facing status as missing cap — don't leak whether the
// cap format was recognized).
func TestCapability_InvalidSignatureReturns401(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   craft a syntactically-valid two-segment token with a random sig
	//   GET /api/sessions?cap=<bad token> → StatusUnauthorized
}

// TestCapability_RevokedGrantReturns403 covers the grant-list path: a token
// whose signature verifies and whose SID matches the path but whose GrantID
// is NOT in the session's active grant set returns 403 (revoked) rather
// than 401. This is the path exercised after ToggleWebServing(false)
// clears the grant list (D-15).
func TestCapability_RevokedGrantReturns403(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   issue a cap; EnableSession; verify GET works → 200
	//   DisableSession (clears grants); same cap → StatusForbidden
}

// TestCapability_ValidCapReturnsSession is the happy-path test: a cap with
// valid signature, matching SID, and an active grant_id returns 200 with
// the single-session response body shape mandated by D-18.
func TestCapability_ValidCapReturnsSession(t *testing.T) {
	t.Skip("implemented in plan 03")
	// Plan 03 will:
	//   issue cap for sess-ok; GET /api/sessions?cap=<cap> → 200
	//   decode JSON, assert exactly 1 item with id == sess-ok
}
