package daemon

import (
	"encoding/json"
	"runtime"
	"testing"
)

// --- Phase 150 Plan 01: shell-web-share-warning-enabled route ----------------

// TestAPIGetShellWebShareWarningEnabled_Default verifies GET returns
// {"value": true} on a fresh engine (default-ON per D-08).
func TestAPIGetShellWebShareWarningEnabled_Default(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawGet(t, socketPath, "/settings/shell-web-share-warning-enabled")
	if status != 200 {
		t.Errorf("GET /settings/shell-web-share-warning-enabled: want 200, got %d (body=%s)", status, string(body))
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, string(body))
	}
	if resp["value"] != true {
		t.Errorf("default value: got %v, want true (D-08 default-ON)", resp["value"])
	}
}

// TestAPIPatchShellWebShareWarningEnabled_FlipsFalse verifies that PATCH
// {"value": false} returns 204 and a subsequent GET returns false.
func TestAPIPatchShellWebShareWarningEnabled_FlipsFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, body := rawPatch(t, socketPath, "/settings/shell-web-share-warning-enabled", `{"value":false}`)
	if status != 204 {
		t.Errorf("PATCH /settings/shell-web-share-warning-enabled: want 204, got %d (body=%s)", status, string(body))
	}

	status, body = rawGet(t, socketPath, "/settings/shell-web-share-warning-enabled")
	if status != 200 {
		t.Errorf("GET after PATCH: want 200, got %d", status)
	}
	var resp map[string]bool
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp["value"] != false {
		t.Errorf("value after PATCH(false): got %v, want false", resp["value"])
	}
}

// TestAPIPatchShellWebShareWarningEnabled_BadBody verifies that a malformed
// body produces a 400 response.
func TestAPIPatchShellWebShareWarningEnabled_BadBody(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, _, socketPath := testDaemon(t)
	status, _ := rawPatch(t, socketPath, "/settings/shell-web-share-warning-enabled", `not-json`)
	if status != 400 {
		t.Errorf("PATCH /settings/shell-web-share-warning-enabled (bad body): want 400, got %d", status)
	}
}

// TestDaemonClient_GetSetShellWebShareWarningEnabled_RoundTrip verifies that
// the DaemonClient wrapper round-trips through the HTTP API.
func TestDaemonClient_GetSetShellWebShareWarningEnabled_RoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("testDaemon uses Unix domain sockets")
	}
	_, client, _ := testDaemon(t)

	// Default should be true (D-08).
	v, err := client.GetShellWebShareWarningEnabled()
	if err != nil {
		t.Fatalf("initial GetShellWebShareWarningEnabled: %v", err)
	}
	if v != true {
		t.Errorf("initial value: got %v, want true (D-08 default-ON)", v)
	}

	// Set false and round-trip.
	if err := client.SetShellWebShareWarningEnabled(false); err != nil {
		t.Fatalf("SetShellWebShareWarningEnabled(false): %v", err)
	}

	v, err = client.GetShellWebShareWarningEnabled()
	if err != nil {
		t.Fatalf("post-set GetShellWebShareWarningEnabled: %v", err)
	}
	if v != false {
		t.Errorf("post-set value: got %v, want false", v)
	}
}
