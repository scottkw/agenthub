package webserver

import "testing"

// Phase 95 Plan 95-01 Task 2 — Wave 0 RED scaffolds for the webserver-side
// web-links assets + parity gates. Plan 95-06 implements all three.
//
// Mirrors Phase 94 find_bar_test.go pattern: t.Skip + explicit
// "Pending until Plan 95-06 implements ..." marker.
//
// Plan 95-06 owns:
//   - Vendoring web/vendor/xterm/addons/addon-web-links.js (UMD copy
//     pulled from frontend/node_modules/@xterm/addon-web-links/lib/).
//   - Extending web/embed.go to embed the new asset.
//   - Bumping vendor_drift_test.go min-count from 6 to 7.
//   - Wiring web/assets/terminal.js openLink() helper for the web client.

// TestAssets_AddonWebLinks — Plan 95-06 implements.
// GET /assets/xterm/addons/addon-web-links.js should return 200 with
// application/javascript content-type. Mirrors Phase 94
// TestAssets_AddonSearch in find_bar_test.go.
func TestAssets_AddonWebLinks(t *testing.T) {
	t.Skip("Pending until Plan 95-06 vendors web/vendor/xterm/addons/addon-web-links.js + extends web/embed.go (95-VALIDATION row 95-05-03).")
}

// TestSecurity_NoCurrentTabNavigation — Plan 95-06 implements.
// Source-inspect web/assets/terminal.js + frontend/src/lib/openLink.ts
// for any occurrence of `location.href = ` or `window.location =`. Must
// return ZERO matches. Defense against accidentally introducing a
// current-tab navigation regression in any future plan.
func TestSecurity_NoCurrentTabNavigation(t *testing.T) {
	t.Skip("Pending until Plan 95-06 implements regression grep test (95-RESEARCH §\"Pattern 5: Single-Helper Opener\").")
}

// TestTerminalJS_WebLinksOpener — Plan 95-06 implements.
// Source-inspect web/assets/terminal.js: must contain
//
//	window.open(<url>, '_blank', 'noopener,noreferrer')
//
// exactly once in the openLink helper, and must NOT contain BrowserOpenURL
// (web is never inside Wails). Mirrors Phase 94 find_bar_test.go pattern.
func TestTerminalJS_WebLinksOpener(t *testing.T) {
	t.Skip("Pending until Plan 95-06 implements web/assets/terminal.js openLink helper (95-VALIDATION row 95-05-02).")
}
