// Phase 93 PLUG-04 / WEB-03 — capability-gated /api/plugin-config endpoint.
//
// These tests pin the contract for the JSON read endpoint that the web
// terminal page fetches at load to gate addon instantiation:
//   - 401 without a cap (requireCapability rejects upstream)
//   - 200 + JSON body with the eight plugin keys when a valid cap is present
//   - 503 when the daemon never wired SetPluginSettingsProvider
//   - 503 when the provider returns nil (defensive — provider may marshal-fail)
//
// Plan 93-04 Task 1 (RED): These tests are authored BEFORE the production
// code exists. The first run MUST fail (compile error: SetPluginSettingsProvider
// undefined; or runtime 404 on the route). Task 2 (GREEN) adds the production
// surface and turns them green.
package webserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestPluginConfig_NoCap_Returns401 — requireCapability rejects requests
// without a ?cap= query param. Phase 93 PLUG-04.
func TestPluginConfig_NoCap_Returns401(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.SetPluginSettingsProvider(func() []byte { return []byte(`{"webgl":true}`) })

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config")
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without cap, got %d", resp.StatusCode)
	}
}

// TestPluginConfig_ValidCap_Returns200JSON — with a session-bound cap, the
// endpoint returns the daemon's pre-marshaled PluginSettings as JSON.
// Phase 93 PLUG-04 / WEB-03.
func TestPluginConfig_ValidCap_Returns200JSON(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pc")
	ws.SetPluginSettingsProvider(func() []byte {
		return []byte(`{"webgl":true,"unicode11":true,"clipboard":true,"search":true,"webLinks":true,"image":true,"serialize":true,"progress":false}`)
	})
	tok := issueCapFor(t, ws, "sess-pc", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config?cap=" + tok)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(body))
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("response not valid JSON: %v body=%s", err, string(body))
	}
	for _, k := range []string{"webgl", "unicode11", "clipboard", "search", "webLinks", "image", "serialize", "progress"} {
		if _, ok := got[k]; !ok {
			t.Errorf("response missing key %q; body=%s", k, string(body))
		}
	}
}

// TestPluginConfig_NoProvider_Returns503 — when the daemon never wired the
// provider, the endpoint returns 503 so the web client can fall back to its
// built-in defaults rather than hard-fail. Phase 93 PLUG-04.
func TestPluginConfig_NoProvider_Returns503(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pc-noprov")
	// SetPluginSettingsProvider intentionally NOT called.
	tok := issueCapFor(t, ws, "sess-pc-noprov", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config?cap=" + tok)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when provider unset, got %d", resp.StatusCode)
	}
}

// TestPluginConfig_NilProvider_Returns503 — provider returns nil (e.g.
// json.Marshal failed); endpoint must NOT serve an empty body as a "valid"
// settings object. Phase 93 PLUG-04.
func TestPluginConfig_NilProvider_Returns503(t *testing.T) {
	ws, client := testServer(t)
	ws.SetSigningKey(capTestKey)
	ws.EnableSession("sess-pc-nil")
	ws.SetPluginSettingsProvider(func() []byte { return nil })
	tok := issueCapFor(t, ws, "sess-pc-nil", "read,write")

	resp, err := client.Get(ws.BaseURL() + "/api/plugin-config?cap=" + tok)
	if err != nil {
		t.Fatalf("client.Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when provider returns nil, got %d", resp.StatusCode)
	}
}
