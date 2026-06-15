//go:build e2e

// Phase 94 Plan 94-04 — SRC-03 perf budget + T-94-04 mitigation regression gate.
//
// TestFindBar_10kPerf launches headless Chromium via chromedp, serves a
// minimal HTML harness that loads the production-vendored xterm.js +
// @xterm/addon-search bundles, writes a 10,000-line scrollback fixture into
// the Terminal buffer, and times searchAddon.findNext('needle', ...) under
// performance.now(). Asserts the call completes within the 1000ms wall-clock
// frame budget defined by ROADMAP Phase 94 SC-3 and 94-RESEARCH §"Performance
// Envelope (SRC-03)".
//
// Mitigates T-94-04 (regex catastrophic backtracking on 10k-line scrollback):
// the test exercises plain-string findNext only (NOT pathological regex —
// pathological regex is documented as accepted limitation per RESEARCH
// Pitfall #5; detection is a research-grade problem).
//
// Build-tagged e2e (//go:build e2e) so the default `go test ./...` lane skips
// it; matches the Phase 89 chromedp e2e suite convention. Self-skips when
// Chromium is unavailable (mirrors browser_csp_e2e_test.go's exec-not-found
// path) — manual UAT remains the fallback in those environments.

package webserver

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	webfs "github.com/scottkw/agenthub/web"
)

// awaitPromiseOpt configures chromedp.Evaluate to wait for the returned
// Promise to resolve before continuing. Required for the term.write call
// because xterm batches writes asynchronously.
func awaitPromiseOpt(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

// startFindBarPerfHarness spins up a TLS httptest server that serves:
//   - GET /__test/findbar-perf-harness  -> internal/webserver/testdata/findbar_perf_harness.html
//   - GET /assets/xterm/*               -> webfs.WebFS sub-FS at vendor/xterm
//
// Mirrors the production /assets/xterm/ mount exactly (Plan 94-01 D-14) so the
// harness loads xterm.js + addon-search.js via the same URL paths the desktop
// and web entrypoints use. Returns the server (caller defers Close).
func startFindBarPerfHarness(t *testing.T) *httptest.Server {
	t.Helper()

	xtermFS, err := fs.Sub(webfs.WebFS, "vendor/xterm")
	if err != nil {
		t.Fatalf("fs.Sub(WebFS, vendor/xterm): %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/xterm/", http.StripPrefix("/assets/xterm/", http.FileServerFS(xtermFS)))
	mux.HandleFunc("GET /__test/findbar-perf-harness", func(w http.ResponseWriter, r *http.Request) {
		harnessPath := filepath.Join("testdata", "findbar_perf_harness.html")
		data, err := os.ReadFile(harnessPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("read harness: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestFindBar_10kPerf asserts that addon-search.findNext completes within
// 1000ms wall-clock against a 10,000-line scrollback buffer. Encodes ROADMAP
// Phase 94 SC-3 ("Search across a 10,000-line scrollback fixture completes
// without UI lockup; no >1s frame budget breach measured in DevTools
// Performance") as a regression gate.
//
// Mitigates T-94-04: regex catastrophic backtracking. Plain-string search
// only; pathological regex is an accepted limitation (RESEARCH Pitfall #5).
func TestFindBar_10kPerf(t *testing.T) {
	// --- Step 1: load + sanity-check the 10,000-line fixture. -----------
	fixturePath := filepath.Join("testdata", "findbar_perf_fixture.txt")
	fixtureBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	lineCount := bytes.Count(fixtureBytes, []byte("\n"))
	if lineCount != 10000 {
		t.Fatalf("fixture %s has %d lines, want 10000 (T-94-04 perf budget invariant)",
			fixturePath, lineCount)
	}
	needleCount := bytes.Count(fixtureBytes, []byte("needle"))
	if needleCount != 100 {
		t.Fatalf("fixture %s has %d 'needle' occurrences, want 100 (deterministic match-count invariant)",
			fixturePath, needleCount)
	}

	// --- Step 2: spin up the harness server. ----------------------------
	srv := startFindBarPerfHarness(t)
	target := srv.URL + "/__test/findbar-perf-harness"

	// --- Step 3: chromedp headless tab. ---------------------------------
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		// httptest.NewTLSServer uses a self-signed cert; the test client
		// accepts it via this flag (same approach as browser_csp_e2e_test.go).
		chromedp.Flag("ignore-certificate-errors", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	// --- Step 4: navigate, write the fixture, time findNext. ------------
	var elapsedMs float64
	var bufferLength int64
	err = chromedp.Run(ctx,
		chromedp.Navigate(target),
		// Wait for the harness to finish initializing the Terminal + SearchAddon.
		chromedp.Poll(
			"window.__findbarPerfHarnessReady === true && typeof window.term !== 'undefined' && typeof window.searchAddon !== 'undefined'",
			nil,
			chromedp.WithPollingTimeout(15*time.Second),
		),
		// Write the 10,000-line fixture into the xterm buffer. term.write is
		// async — the JS uses the callback form to resolve a Promise so the
		// poll afterwards has deterministic semantics.
		chromedp.Evaluate(fmt.Sprintf(`
			(function() {
				return new Promise(function(resolve) {
					window.term.write(%q, resolve);
				});
			})()
		`, string(fixtureBytes)), nil, awaitPromiseOpt),
		// Confirm the terminal buffer actually absorbed the content. Each line
		// in the fixture ends with \n so xterm should report length >= 10000.
		// (Lines may collapse if the terminal wraps; assert >= 10000 to allow
		// for a safety margin without over-constraining renderer internals.)
		chromedp.Evaluate(`window.term.buffer.active.length`, &bufferLength),
		// Run findNext under performance.now() and capture wall-clock duration
		// in milliseconds. Plain-string search; case-insensitive, no regex,
		// no whole-word — representative of normal user usage.
		chromedp.Evaluate(`
			(function() {
				const t0 = performance.now();
				window.searchAddon.findNext('needle', { regex: false, caseSensitive: false, wholeWord: false });
				return performance.now() - t0;
			})()
		`, &elapsedMs),
	)
	if err != nil {
		// Self-skip when Chromium isn't installed locally (mirrors the
		// Phase 89 e2e harness's behavior). Treats "exec: chrome not found"
		// as an environment issue, not a perf regression.
		if strings.Contains(err.Error(), "exec") || strings.Contains(err.Error(), "chrome not found") {
			t.Skipf("chromedp: Chromium unavailable (%v) — manual UAT covers this perf budget", err)
		}
		t.Fatalf("chromedp.Run: %v", err)
	}

	if bufferLength < 10000 {
		t.Fatalf("xterm buffer absorbed only %d lines, expected >= 10000 (fixture write incomplete; cannot trust perf measurement)",
			bufferLength)
	}

	// --- Step 5: assert the budget. -------------------------------------
	const budgetMs = 1000.0
	t.Logf("findNext over 10,000-line scrollback: %.2fms (budget %.0fms; SC-3 / T-94-04)", elapsedMs, budgetMs)
	if elapsedMs > budgetMs {
		t.Fatalf("findNext over 10,000-line scrollback took %.2fms; budget is %.0fms (T-94-04 mitigation regression; ROADMAP Phase 94 SC-3)",
			elapsedMs, budgetMs)
	}
}
