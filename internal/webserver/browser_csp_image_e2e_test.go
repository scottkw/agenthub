//go:build e2e

// Package webserver Phase 96 IMG-03 — chromedp CSP-violation e2e test for
// addon-image.
//
// Verifies that loading the terminal page with addon-image enabled and
// emitting a sixel byte sequence into the relay's PTY-output pipe produces
// ZERO Content-Security-Policy violations in the browser. This is the
// load-bearing proof for Plan 96-03's CSP amendment (`script-src 'self'
// 'wasm-unsafe-eval'`): the SIXEL WASM decoder bootstraps via
// WebAssembly.instantiate / new WebAssembly.Module / new WebAssembly.Instance,
// which CSP3 §6.3 governs via the script-src directive. Without
// 'wasm-unsafe-eval', the browser would emit a securitypolicyviolation event
// for the WASM compile/instantiate step.
//
// This test asserts that no such event (or any other CSP violation) surfaces
// during a normal load + sixel-emission session.
//
// Mirrors browser_csp_e2e_test.go infrastructure (same DOM-level
// `securitypolicyviolation` listener; same self-skip on missing Chromium;
// same testServerWithHub harness from capability_test_helpers.go).
//
// Gated by //go:build e2e — default `go test ./...` does NOT run this.
// Run explicitly with: go test -tags=e2e ./internal/webserver/...
package webserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// TestBrowserCSP_TerminalImage_NoViolations — Phase 96 IMG-03.
//
// Drives a real headless Chromium, navigates to the terminal page with the
// addon-image vendored bundle in scope, injects a minimal-but-valid sixel
// stream into the session's PTY-output pipe via testServerWithHub's writer,
// gives the WASM SIXEL decoder ~3s to bootstrap, then asserts zero CSP
// violations were captured by the page's securitypolicyviolation listener.
func TestBrowserCSP_TerminalImage_NoViolations(t *testing.T) {
	const sessionID = "sess-96-e2e-image"

	ws, _, ptyWriter, _ := testServerWithHub(t, sessionID)
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, sessionID, "read,write")
	url := ws.BaseURL() + "/sessions/" + sessionID + "?cap=" + token

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

	// Same DOM-level securitypolicyviolation listener as
	// runChromedpAndCollectCSPViolations in browser_csp_e2e_test.go. Captures
	// every violation that fires after the listener attaches (added via
	// AddScriptToEvaluateOnNewDocument so it runs before any inline script in
	// terminal.html, including the WASM bootstrap inside addon-image).
	initScript := `
      window.__cspViolations = [];
      document.addEventListener('securitypolicyviolation', (e) => {
        window.__cspViolations.push({
          directive: e.violatedDirective,
          blockedURI: e.blockedURI,
          lineNumber: e.lineNumber,
          sourceFile: e.sourceFile,
        });
      });
    `

	// Synthetic minimal-valid sixel stream documented in 96-RESEARCH §"Code
	// Example 5". Two-color palette + a 10-pixel run on each row: enough to
	// exercise the WASM SIXEL decoder's compile/instantiate path; small
	// enough to fit comfortably under the 256 KiB scrollback cap. The DCS
	// terminator (\x1b\\) is required by the parser.
	sixel := []byte("\x1bPq#0;2;100;0;0#1;2;0;100;0!10A!10@-\x1b\\")

	var violations []map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(initScript).Do(ctx)
			return err
		}),
		chromedp.Navigate(url),
		// Wait for the xterm canvas to mount; this proves terminal.js ran
		// and (if image addon UMD loaded successfully) ImageAddon was
		// constructed alongside Unicode 11 inside initTerminal().
		chromedp.WaitVisible(".xterm", chromedp.ByQuery),
		// Inject sixel into the PTY-output pipe; the relay fan-out forwards
		// it over the WebSocket to the browser, where ImageAddon's parser
		// consumes the DCS sequence and triggers the WASM decode path.
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, werr := ptyWriter.Write(sixel)
			return werr
		}),
		// Give the WS round-trip + WASM compile/instantiate enough time to
		// surface any securitypolicyviolation events. 3s mirrors the
		// "WS handshake + late-asset" budget used in
		// browser_csp_e2e_test.go's 2s sleep, with extra headroom for the
		// WASM bootstrap.
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate("window.__cspViolations", &violations),
	)
	if err != nil {
		if strings.Contains(err.Error(), "exec") || strings.Contains(err.Error(), "chrome not found") {
			t.Skipf("chromedp: Chromium unavailable (%v) — run 96-HUMAN-UAT.md instead (Phase 96 IMG-03 fallback)", err)
		}
		t.Fatalf("chromedp.Run: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("Phase 96 IMG-03 CSP amendment regression: %d CSP violation(s) observed; expected 0", len(violations))
		for i, v := range violations {
			t.Errorf("  CSP violation #%d: %+v", i+1, v)
		}
	}
}
