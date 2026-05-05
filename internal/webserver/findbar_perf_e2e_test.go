//go:build e2e

package webserver

import "testing"

// Phase 94 Wave 0 RED scaffold — Plan 94-04 implements (SRC-03 perf budget).
// See: .planning/phases/94-search-addon-find-bar-desktop-web/94-VALIDATION.md row 04-perf wave 3.
// RESEARCH §"Performance Envelope"; UI-SPEC §"Perf Budget — 10k-line scrollback".

func TestFindBar_10kPerf(t *testing.T) {
	t.Skip("RED scaffold — Plan 94-04 implements 10k-line scrollback perf harness via chromedp " +
		"using a dedicated internal/webserver/testdata/findbar_perf_harness.html fixture. " +
		"Asserts search-to-first-match completes within 1s frame budget. " +
		"Phase 89 chromedp e2e analog applies (//go:build e2e tag, testServer pattern).")
}
