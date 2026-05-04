package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSetPluginSettingsRoundTrip validates the engine-level Set/Get round-trip
// AND the disk persistence reload-engine pattern (mirrors TestStartMinimizedPersistence
// at engine_settings_test.go:101-148).
func TestSetPluginSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: defaultPluginSettings(),
	}

	// Flip every plugin: defaults are 7-ON-1-OFF; flipped is 7-OFF-1-ON.
	flipped := PluginSettings{
		WebGL:     false,
		Unicode11: false,
		Search:    false,
		WebLinks:  false,
		Image:     false,
		Serialize: false,
		Clipboard: false,
		Progress:  true,
	}
	e.SetPluginSettings(flipped)

	got := e.GetPluginSettings()
	if got != flipped {
		t.Errorf("GetPluginSettings after Set: got %+v, want %+v", got, flipped)
	}

	// Reload-engine round-trip: a fresh engine constructed against the same
	// configDir must observe the persisted values via loadSettingsFromDisk.
	e2 := &SessionEngine{
		configDir: dir,
		cliPaths:  make(map[string]string),
	}
	e2.loadSettingsFromDisk(dir)

	got2 := e2.GetPluginSettings()
	if got2 != flipped {
		t.Errorf("reloaded GetPluginSettings: got %+v, want %+v", got2, flipped)
	}
}

// newPluginAPIForTest constructs an *API wired to a fresh SessionEngine whose
// configDir is the supplied temp directory. We intentionally bypass
// NewSessionEngine (which would load from the user's real config dir) and
// NewAPI's BootstrapCapabilityState path — the plugin-settings handlers do
// not consult capability state.
func newPluginAPIForTest(t *testing.T, dir string) *API {
	t.Helper()
	e := &SessionEngine{
		configDir:      dir,
		cliPaths:       make(map[string]string),
		pluginSettings: defaultPluginSettings(),
	}
	a := &API{
		engine: e,
		mux:    http.NewServeMux(),
	}
	a.registerRoutes()
	return a
}

// TestPluginSettingsHTTPRoundTrip exercises PATCH→engine→saveSettingsToDisk→GET
// at the handler level via httptest.NewRecorder against api.mux. End-to-end
// Unix-socket transport via DaemonClient is covered by production startup.
func TestPluginSettingsHTTPRoundTrip(t *testing.T) {
	dir := t.TempDir()
	a := newPluginAPIForTest(t, dir)

	// PATCH /settings/plugins with all-flipped settings.
	flipped := PluginSettings{Progress: true /* others false by zero-value */}
	body, err := json.Marshal(flipped)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/settings/plugins", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PATCH expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// GET /settings/plugins and assert the body matches what we PATCHed.
	req2 := httptest.NewRequest(http.MethodGet, "/settings/plugins", nil)
	rec2 := httptest.NewRecorder()
	a.mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}
	var got PluginSettings
	if err := json.Unmarshal(rec2.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal GET body: %v", err)
	}
	if got != flipped {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, flipped)
	}
}

// TestSetPluginSettingsRejectsUnknownFields verifies the V5 input-validation
// guard (T-92-03 mitigation): a PATCH body containing a key not present in
// PluginSettings is rejected with 400.
func TestSetPluginSettingsRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	a := newPluginAPIForTest(t, dir)

	body := []byte(`{"webgl": true, "evilPlugin": "x"}`)
	req := httptest.NewRequest(http.MethodPatch, "/settings/plugins", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown fields, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestSetPluginSettingsRejectsOversizedBody verifies the V5 input-validation
// guard (T-92-02 mitigation): a body larger than 8 KiB is rejected.
//
// We construct a body that is valid JSON but >8 KiB by prepending leading
// whitespace (json.Decoder tolerates whitespace; http.MaxBytesReader trips
// on total bytes).
func TestSetPluginSettingsRejectsOversizedBody(t *testing.T) {
	dir := t.TempDir()
	a := newPluginAPIForTest(t, dir)

	whitespace := bytes.Repeat([]byte(" "), 9000)
	body := append(whitespace, []byte(`{"webgl": true}`)...)
	req := httptest.NewRequest(http.MethodPatch, "/settings/plugins", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized body, got %d: %s", rec.Code, rec.Body.String())
	}
}
