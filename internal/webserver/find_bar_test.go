package webserver

import (
	"net/http"
	"strings"
	"testing"
)

// Phase 94 Plan 94-01 Wave 0 — find-bar asset + web HTML/JS smoke tests.
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 01-vendor wave 0
// and row 05-web wave 4.

// TestAssets_AddonSearch is GREEN at Wave 0: Plan 94-01 already vendored
// addon-search.js into web/vendor/xterm/addons/ and added it to web/embed.go,
// so the existing FileServerFS routes (Phase 89) serve it at /assets/xterm/addons/.
// Pattern verbatim from internal/webserver/assets_test.go::TestAssets_VendoredAddons (lines 210-230).
func TestAssets_AddonSearch(t *testing.T) {
	ws, client := testServer(t)
	const path = "/assets/xterm/addons/addon-search.js"
	resp, err := client.Get(ws.BaseURL() + path)
	if err != nil {
		t.Fatalf("client.Get %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET %s: expected 200, got %d", path, resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "javascript") {
		t.Errorf("GET %s: expected javascript content-type, got %q", path, ct)
	}
}

// Phase 94 Wave 0 RED scaffold — Plan 94-05 will implement.
// See: 94-VALIDATION.md row 05-web wave 4.
func TestTerminalHTML_FindBar(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-05 injects <div id=\"find-bar\" hidden> + addon-search " +
		"<script> tag into web/terminal.html. Test will assert presence via httptest GET / " +
		"and substring matching (mirrors Phase 89 TestTerminalHTML_* style). " +
		"See 94-VALIDATION.md row 05-web wave 4.")
}

// Phase 94 Wave 0 RED scaffold — Plan 94-05 will implement.
// See: 94-VALIDATION.md row 05-web wave 4.
func TestTerminalJS_SearchAddon(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-05 wires UMD global `SearchAddon.SearchAddon` constructor " +
		"into web/assets/terminal.js (Pitfall #7 verification). Test asserts the constructor " +
		"expression appears via raw asset GET. See 94-VALIDATION.md row 05-web wave 4.")
}
