package webserver

import (
	"net/http"
	"strings"
	"testing"

	webfs "github.com/scottkw/agenthub/web"
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

// TestTerminalHTML_FindBar — Phase 94 Plan 94-05 SRC-05 web parity.
// terminal.html must contain the find-bar DOM and load the vendored
// addon-search UMD bundle. Mirrors TestAssets_VendoredAddons (same package,
// web-asset source-inspection style). T-94-05 mitigation gate: ensures the
// CSP-relevant script tag is registered same-origin (no third-party origin).
func TestTerminalHTML_FindBar(t *testing.T) {
	data, err := webfs.WebFS.ReadFile("terminal.html")
	if err != nil {
		t.Fatalf("ReadFile terminal.html: %v", err)
	}
	s := string(data)
	requiredSubstrings := []struct{ name, sub string }{
		{"find-bar container", `id="find-bar"`},
		{"find-bar hidden attribute", `id="find-bar" hidden`},
		{"find-bar role", `role="search"`},
		{"find-bar input", `id="find-bar-input"`},
		{"find-bar count", `id="find-bar-count"`},
		{"find-bar prev button", `id="find-bar-prev"`},
		{"find-bar next button", `id="find-bar-next"`},
		{"find-bar case toggle", `id="find-bar-toggle-case"`},
		{"find-bar regex toggle", `id="find-bar-toggle-regex"`},
		{"find-bar word toggle", `id="find-bar-toggle-word"`},
		{"find-bar close button", `id="find-bar-close"`},
		{"addon-search script tag", `/assets/xterm/addons/addon-search.js`},
		{"aria-label Find in terminal", `aria-label="Find in terminal"`},
		{"aria-label Search", `aria-label="Search"`},
		{"aria-label Previous match", `aria-label="Previous match"`},
		{"aria-label Next match", `aria-label="Next match"`},
		{"aria-label Case sensitive", `aria-label="Case sensitive"`},
		{"aria-label Regular expression", `aria-label="Regular expression"`},
		{"aria-label Whole word", `aria-label="Whole word"`},
		{"aria-label Close find bar", `aria-label="Close find bar"`},
	}
	for _, r := range requiredSubstrings {
		if !strings.Contains(s, r.sub) {
			t.Errorf("terminal.html missing %s: substring %q not found (T-94-05 / SRC-05 mitigation)", r.name, r.sub)
		}
	}
}

// Phase 94 Wave 0 RED scaffold — Plan 94-05 Task 2 will implement.
// See: 94-VALIDATION.md row 05-web wave 4.
func TestTerminalJS_SearchAddon(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-05 wires UMD global `SearchAddon.SearchAddon` constructor " +
		"into web/assets/terminal.js (Pitfall #7 verification). Test asserts the constructor " +
		"expression appears via raw asset GET. See 94-VALIDATION.md row 05-web wave 4.")
}
