//go:build e2e

// Phase 94 Plan 94-05 — SRC-05 web parity e2e gate (T-94-05 mitigation).
//
// TestFindBar_Web launches headless Chromium via chromedp, navigates to the
// real served terminal page (testServerWithHub + capability token), and
// drives the find-bar lifecycle:
//   - Wait for window.term + window.searchAddonHandle (constructed by
//     applyPluginConfig when pluginConfig.search === true).
//   - Write deterministic content into the xterm buffer via window.term.write.
//   - Focus the terminal helper textarea (Pitfall #1 — Cmd-F only fires when
//     #terminal contains the active element).
//   - Synthesize a Cmd-F window keydown; assert #find-bar is visible.
//   - Type a query into #find-bar-input; wait past the 100ms debounce + the
//     addon's internal ~200ms highlight settle; assert the count label updates
//     (SRC-02 onDidChangeResults wiring; gated by `decorations: {}` per the
//     94-05 reconciliation — see SUMMARY).
//   - Synthesize Esc on the find-bar container (Pitfall #3); assert
//     #find-bar.hidden === true (SRC-01 dismiss path).
//
// Build-tagged e2e (//go:build e2e) so the default `go test ./...` lane skips
// it; matches the Phase 89 / 94-04 chromedp e2e suite convention. Self-skips
// when Chromium is unavailable.
//
// Closes ROADMAP Phase 94 SC-1 ("behavior is identical on desktop and on
// web-served Tailscale terminal pages") on the web surface.
package webserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestFindBar_Web — full open / type / count / dismiss round-trip on the
// served terminal page. T-94-05 mitigation gate.
func TestFindBar_Web(t *testing.T) {
	const sessionID = "sess-94-e2e-findbar"
	ws, _, _, _ := testServerWithHub(t, sessionID)
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, sessionID, "read,write")
	target := ws.BaseURL() + "/sessions/" + sessionID + "?cap=" + token

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var (
		findBarVisibleAfterCmdF bool
		matchCountText          string
		findBarHiddenAfterEsc   bool
	)

	err := chromedp.Run(ctx,
		chromedp.Navigate(target),
		// Wait for the IIFE to finish wiring + applyPluginConfig to construct
		// the SearchAddon. The handle is exposed at window.searchAddonHandle
		// inside the load arm of applyPluginConfig.
		chromedp.Poll(
			"typeof window.term !== 'undefined' && window.searchAddonHandle !== null && typeof window.searchAddonHandle !== 'undefined'",
			nil,
			chromedp.WithPollingTimeout(15*time.Second),
		),
		// Write deterministic content into the xterm buffer so the search has
		// real matches. term.write is async; the small Sleep below covers the
		// async settle.
		chromedp.Evaluate(`(function(){
			for (var i = 0; i < 20; i++) {
				window.term.write('hello world line ' + i + '\r\n');
			}
		})()`, nil),
		chromedp.Sleep(200*time.Millisecond),
		// Focus the terminal helper textarea so the Cmd-F focus gate
		// (Pitfall #1: termEl.contains(document.activeElement)) passes.
		chromedp.Evaluate(`(function(){
			var ta = document.querySelector('#terminal .xterm-helper-textarea');
			if (ta) ta.focus();
		})()`, nil),
		// Synthesize Cmd-F (mac) AND Ctrl-F (linux/windows) — the handler
		// gates by navigator.platform; setting both metaKey + ctrlKey makes
		// the test platform-agnostic.
		chromedp.Evaluate(`(function(){
			var e = new KeyboardEvent('keydown', {
				key: 'f', metaKey: true, ctrlKey: true,
				bubbles: true, cancelable: true
			});
			window.dispatchEvent(e);
		})()`, nil),
		// Find bar should now be visible.
		chromedp.Evaluate(`!document.getElementById('find-bar').hidden`, &findBarVisibleAfterCmdF),
		// Type "hello" into the input — fires both 'input' (debounced search)
		// and reaches the SearchAddon via runSearch + 100ms debounce.
		chromedp.Evaluate(`(function(){
			var input = document.getElementById('find-bar-input');
			input.value = 'hello';
			input.dispatchEvent(new Event('input', { bubbles: true }));
		})()`, nil),
		// Wait past the 100ms debounce + xterm-addon-search's internal ~200ms
		// highlight timeout + a small grace for onDidChangeResults to fire.
		chromedp.Sleep(700*time.Millisecond),
		chromedp.Evaluate(`document.getElementById('find-bar-count').textContent`, &matchCountText),
		// Synthesize Esc at the find-bar CONTAINER (Pitfall #3 — handler is
		// bound on the container, not the input, so Esc works regardless of
		// which child has focus).
		chromedp.Evaluate(`(function(){
			var bar = document.getElementById('find-bar');
			var input = document.getElementById('find-bar-input');
			if (input) input.focus();
			var e = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true });
			bar.dispatchEvent(e);
		})()`, nil),
		chromedp.Sleep(50*time.Millisecond),
		chromedp.Evaluate(`document.getElementById('find-bar').hidden`, &findBarHiddenAfterEsc),
	)
	if err != nil {
		// Self-skip when Chromium isn't installed locally (mirrors the Phase 89
		// + 94-04 e2e harness behavior). Treats "exec: chrome not found" as
		// an environment issue, not a regression.
		if strings.Contains(err.Error(), "exec") || strings.Contains(err.Error(), "chrome not found") {
			t.Skipf("chromedp: Chromium unavailable (%v) — manual UAT covers this path", err)
		}
		t.Fatalf("chromedp.Run: %v", err)
	}

	if !findBarVisibleAfterCmdF {
		t.Error("find bar did not appear after Cmd-F (SRC-05 / SC-1 — focus-conditioned Cmd-F open path broken)")
	}
	if matchCountText == "" {
		t.Errorf("find bar count text empty after typing 'hello' on a 20-line buffer — expected non-empty " +
			"(SRC-02 onDidChangeResults wiring broken; verify decorations:{} + Terminal allowProposedApi:true reconciliation — see 94-05 SUMMARY)")
	} else {
		t.Logf("find bar match count after typing 'hello': %q", matchCountText)
	}
	if !findBarHiddenAfterEsc {
		t.Error("find bar did not hide after Esc (SRC-01 / SC-1 — Esc dismiss broken)")
	}
}
