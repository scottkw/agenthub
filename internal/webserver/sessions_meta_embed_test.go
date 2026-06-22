package webserver_test

// Wave 0 RED tests for Phase 146 FIX-03 — out-of-band open redesign.
//
// This file encodes the RESTORED RB-03 cap-free contract for GET /api/sessions/meta.
//
// The superseded broadcast tests (TestSessionsMeta_EmbedJoinCodes and
// TestSessionsMeta_NilIssuer) asserted that ro_join_code / rw_join_code were
// embedded in the response. That design was REJECTED (see CONTEXT.md
// D-02/D-10/D-12 and superseded-broadcast/README.md).
//
// This inverted test asserts the ABSENCE of those keys — enforcing the RB-03
// invariant: /api/sessions/meta carries no credential material at all.
//
// RED state now: the current production code still embeds ro_join_code/rw_join_code
// (broadcast wiring from the superseded Wave 0). This test FAILS against current code.
// It goes GREEN when Plan 02 removes the broadcast (the join-code issuer wiring,
// the join-code fields on sessionMetaItem, and the enrichment in handleSessionsMeta).

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestSessionsMeta_NoJoinCodesInResponse encodes the RB-03 restored cap-free contract:
// GET /api/sessions/meta MUST NOT contain ro_join_code or rw_join_code in any item.
// Credentials are never broadcast to the discovery payload — the owner delivers
// them out of band when they choose to share.
//
// RED: current production code embeds codes via superseded broadcast wiring;
// this test FAILS until Plan 02 removes the broadcast.
func TestSessionsMeta_NoJoinCodesInResponse(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(ssExtTestKey)
	ws.SetSessionResolver(func(id string) (string, string, string, string) {
		if id == "sess-rb03" {
			return "RB03 Session", "claude", "running", "host.ts.net"
		}
		return "", "", "", ""
	})
	ws.EnableSession("sess-rb03")
	// No join-code issuer is wired — RB-03 requires cap-free discovery
	// regardless of any issuer configuration.

	resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
	if err != nil {
		t.Fatalf("GET /api/sessions/meta: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d: %v", len(items), items)
	}

	item := items[0]

	// RB-03 RESTORED: join codes MUST NOT appear in the cap-free discovery payload.
	if _, ok := item["ro_join_code"]; ok {
		t.Error("RB-03 violation: ro_join_code must not appear in cap-free meta response — codes are delivered out of band only")
	}
	if _, ok := item["rw_join_code"]; ok {
		t.Error("RB-03 violation: rw_join_code must not appear in cap-free meta response — codes are delivered out of band only")
	}

	// Assert the cap-free fields ARE present (RB-01 contract: id, name, cli_type, status, url).
	for _, key := range []string{"id", "name", "cli_type", "status", "url"} {
		if _, ok := item[key]; !ok {
			t.Errorf("expected cap-free field %q to be present in meta response, but it was absent", key)
		}
	}
}
