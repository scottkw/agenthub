package daemon

import (
	"encoding/json"
	"runtime"
	"testing"
)

// --- Phase 167 Plan 01: notify-on-waiting route -----------------------------

// TestAPIGetNotifyOnWaiting_Default verifies GET returns {"notifyOnWaiting":
// false} on a fresh engine (default OFF per NTF-04).
func TestAPIGetNotifyOnWaiting_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/notify-on-waiting")
	if status != 200 {
		t.Errorf("GET /settings/notify-on-waiting: want 200, got %d (body=%s)", status, string(body))
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if resp["notifyOnWaiting"] != false {
		t.Errorf("default value: got %v, want false (NTF-04 default OFF)", resp["notifyOnWaiting"])
	}
}

// TestAPIPatchNotifyOnWaiting_FlipsTrue verifies that PATCH
// {"notifyOnWaiting": true} returns 204 and a subsequent GET returns true.
func TestAPIPatchNotifyOnWaiting_FlipsTrue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawPatch(t, socketPath, "/settings/notify-on-waiting", `{"notifyOnWaiting":true}`)
	if status != 204 {
		t.Errorf("PATCH /settings/notify-on-waiting: want 204, got %d (body=%s)", status, string(body))
	}

	status, body = rawGet(t, socketPath, "/settings/notify-on-waiting")
	if status != 200 {
		t.Errorf("GET after PATCH: want 200, got %d", status)
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp["notifyOnWaiting"] != true {
		t.Errorf("value after PATCH(true): got %v, want true", resp["notifyOnWaiting"])
	}
}

// TestAPIPatchNotifyOnWaiting_BadBody verifies that a malformed body produces
// a 400 response and does not mutate state.
func TestAPIPatchNotifyOnWaiting_BadBody(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	api, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/notify-on-waiting", `not-json`)
	if status != 400 {
		t.Errorf("PATCH /settings/notify-on-waiting (bad body): want 400, got %d", status)
	}
	if got := api.engine.GetNotifyOnWaiting(); got != false {
		t.Errorf("state after malformed PATCH: got %v, want unchanged false", got)
	}
}

// TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip verifies that the
// DaemonClient wrapper round-trips through the HTTP API.
func TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, client, _ := testDaemon(t)

	// Default should be false (NTF-04).
	v, err := client.GetNotifyOnWaiting()
	if err != nil {
		t.Fatalf("initial GetNotifyOnWaiting: %v", err)
	}
	if v != false {
		t.Errorf("initial value: got %v, want false (NTF-04 default OFF)", v)
	}

	// Set true and round-trip.
	if err := client.SetNotifyOnWaiting(true); err != nil {
		t.Fatalf("SetNotifyOnWaiting(true): %v", err)
	}

	v, err = client.GetNotifyOnWaiting()
	if err != nil {
		t.Fatalf("post-set GetNotifyOnWaiting: %v", err)
	}
	if v != true {
		t.Errorf("post-set value: got %v, want true", v)
	}
}
