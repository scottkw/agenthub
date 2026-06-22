package daemon

// Wave 0 RED tests for Phase 146 FIX-03 — mintSessionJoinCodes.
//
// These tests assert the contract for the new api.mintSessionJoinCodes helper
// that Wave 1 will extract from issueCapabilitiesForSession. They are
// intentionally RED until Wave 1 adds the method to api.go.
//
// Contract fixed here (before implementation):
//   - mintSessionJoinCodes(sessionID) returns (roCode, rwCode string, err error)
//   - Both codes are non-empty and distinct
//   - Grants are registered on the WebServer before the function returns
//   - Each code exchanges to a valid token (grant active check via exchange path)

import (
	"context"
	"testing"

	"github.com/scottkw/agenthub/internal/capability"
	"github.com/scottkw/agenthub/internal/webserver"
)

// mintCodesTestSetup mirrors issueCapsTestSetup from api_test.go.
// Creates a testDaemon, a real WebServer, and wires capability state.
// The WebServer listener is NOT started — mintSessionJoinCodes only needs
// ws.AddGrant + ws.BaseURL (same invariant as issueCapabilitiesForSession tests).
func mintCodesTestSetup(t *testing.T) (*API, *webserver.WebServer, []byte) {
	t.Helper()
	api, _, _ := testDaemon(t)
	ws, err := webserver.NewWebServer(webserver.Config{
		BindIP: "127.0.0.1",
		Port:   0,
		FQDN:   "test.local",
	}, api.engine.Manager())
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	api.SetWebServerForTest(ws)
	key := configureCapabilityStateForTest(t, api, ws)
	return api, ws, key
}

// TestMintSessionJoinCodes asserts the full contract of the new helper:
//  1. Both roCode and rwCode are non-empty.
//  2. roCode != rwCode (distinct join codes, distinct tokens).
//  3. Each code can be exchanged to a valid token (grants registered before return).
//
// RED: api.mintSessionJoinCodes does not exist until Wave 1.
func TestMintSessionJoinCodes(t *testing.T) {
	api, ws, key := mintCodesTestSetup(t)

	sid, err := api.engine.CreateSession(context.Background(), "cat", "mint-test-session", "", nil, 80, 24, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	t.Cleanup(func() { _ = api.engine.KillSession(sid) })

	ws.EnableSession(sid)

	// Call the not-yet-existing helper — RED until Wave 1 adds it.
	roCode, rwCode, err := api.mintSessionJoinCodes(sid)
	if err != nil {
		t.Fatalf("mintSessionJoinCodes: %v", err)
	}

	// 1. Both codes must be non-empty.
	if roCode == "" {
		t.Error("roCode must be non-empty")
	}
	if rwCode == "" {
		t.Error("rwCode must be non-empty")
	}

	// 2. Codes must be distinct (each wraps a different token with different grant_id).
	if roCode == rwCode {
		t.Error("roCode and rwCode must be distinct join codes")
	}

	// 3. Each code must exchange to a valid token via the join-code manager.
	// We probe by calling JoinCodeManager.Exchange directly (same manager
	// used by the HTTP exchange endpoint). If grants were registered before
	// mintSessionJoinCodes returned, Exchange succeeds; if not, the token
	// would fail requireCapability middleware checks.
	jc := api.joinCodes // directly accessible in package daemon
	if jc == nil {
		t.Fatal("api.joinCodes is nil after configureCapabilityStateForTest")
	}

	roToken, err := jc.Exchange(roCode)
	if err != nil {
		t.Fatalf("Exchange(roCode): %v — grant must be registered before mintSessionJoinCodes returns", err)
	}
	if roToken == "" {
		t.Error("Exchange(roCode) returned empty token")
	}

	rwToken, err := jc.Exchange(rwCode)
	if err != nil {
		t.Fatalf("Exchange(rwCode): %v — grant must be registered before mintSessionJoinCodes returns", err)
	}
	if rwToken == "" {
		t.Error("Exchange(rwCode) returned empty token")
	}

	// 4. Tokens must be distinct (different grant IDs).
	if roToken == rwToken {
		t.Error("roToken and rwToken must be distinct capability tokens")
	}

	// 5. Verify that the RO token carries "read" perms and RW carries "read,write".
	roClaims, err := capability.Verify(roToken, key)
	if err != nil {
		t.Fatalf("capability.Verify(roToken): %v", err)
	}
	if roClaims.Perms != "read" {
		t.Errorf("roToken Perms = %q; want \"read\"", roClaims.Perms)
	}
	if roClaims.SID != sid {
		t.Errorf("roToken SID = %q; want %q", roClaims.SID, sid)
	}

	rwClaims, err := capability.Verify(rwToken, key)
	if err != nil {
		t.Fatalf("capability.Verify(rwToken): %v", err)
	}
	if rwClaims.Perms != "read,write" {
		t.Errorf("rwToken Perms = %q; want \"read,write\"", rwClaims.Perms)
	}
	if rwClaims.SID != sid {
		t.Errorf("rwToken SID = %q; want %q", rwClaims.SID, sid)
	}

	// Suppress unused variable warning (key used in capability.Verify calls above).
	_ = key
}
