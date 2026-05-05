//go:build e2e

package webserver

import "testing"

// Phase 94 Wave 0 RED scaffold — Plan 94-05 implements (SRC-05 web parity).
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 05-web wave 4.
// RESEARCH §"Web Parity"; UI-SPEC §"Web FindBar Element + JS Wiring".

func TestFindBar_Web(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-05 implements web parity e2e via chromedp: " +
		"open Cmd-F on web-served terminal page, type query, assert match-count label updates, " +
		"navigate next/previous, dismiss with Esc, persist toggle round-trip via SSE. " +
		"Phase 89 chromedp e2e analog applies (//go:build e2e tag).")
}
