package daemon

import (
	"encoding/json"
	"runtime"
	"testing"
)

// --- Phase 168 Plan 04: stay-on-hub-after-create route -----------------------

// TestAPIGetStayOnHubAfterCreate_Default verifies GET returns
// {"stayOnHubAfterCreate": false} on a fresh engine (default OFF per D-09).
func TestAPIGetStayOnHubAfterCreate_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/stay-on-hub-after-create")
	if status != 200 {
		t.Errorf("GET /settings/stay-on-hub-after-create: want 200, got %d (body=%s)", status, string(body))
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if resp["stayOnHubAfterCreate"] != false {
		t.Errorf("default value: got %v, want false (D-09 default OFF)", resp["stayOnHubAfterCreate"])
	}
}

// TestAPIPatchStayOnHubAfterCreate_FlipsTrue verifies that PATCH
// {"stayOnHubAfterCreate": true} returns 204 and a subsequent GET returns
// true.
func TestAPIPatchStayOnHubAfterCreate_FlipsTrue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawPatch(t, socketPath, "/settings/stay-on-hub-after-create", `{"stayOnHubAfterCreate":true}`)
	if status != 204 {
		t.Errorf("PATCH /settings/stay-on-hub-after-create: want 204, got %d (body=%s)", status, string(body))
	}

	status, body = rawGet(t, socketPath, "/settings/stay-on-hub-after-create")
	if status != 200 {
		t.Errorf("GET after PATCH: want 200, got %d", status)
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp["stayOnHubAfterCreate"] != true {
		t.Errorf("value after PATCH(true): got %v, want true", resp["stayOnHubAfterCreate"])
	}
}

// TestAPIPatchStayOnHubAfterCreate_BadBody verifies that a malformed body
// produces a 400 response and does not mutate state.
func TestAPIPatchStayOnHubAfterCreate_BadBody(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	api, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/stay-on-hub-after-create", `not-json`)
	if status != 400 {
		t.Errorf("PATCH /settings/stay-on-hub-after-create (bad body): want 400, got %d", status)
	}
	if got := api.engine.GetStayOnHubAfterCreate(); got != false {
		t.Errorf("state after malformed PATCH: got %v, want unchanged false", got)
	}
}

// TestDaemonClient_GetSetStayOnHubAfterCreate_RoundTrip verifies that the
// DaemonClient wrapper round-trips through the HTTP API.
func TestDaemonClient_GetSetStayOnHubAfterCreate_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, client, _ := testDaemon(t)

	// Default should be false (D-09).
	v, err := client.GetStayOnHubAfterCreate()
	if err != nil {
		t.Fatalf("initial GetStayOnHubAfterCreate: %v", err)
	}
	if v != false {
		t.Errorf("initial value: got %v, want false (D-09 default OFF)", v)
	}

	// Set true and round-trip.
	if err := client.SetStayOnHubAfterCreate(true); err != nil {
		t.Fatalf("SetStayOnHubAfterCreate(true): %v", err)
	}

	v, err = client.GetStayOnHubAfterCreate()
	if err != nil {
		t.Fatalf("post-set GetStayOnHubAfterCreate: %v", err)
	}
	if v != true {
		t.Errorf("post-set value: got %v, want true", v)
	}
}
