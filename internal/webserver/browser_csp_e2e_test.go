//go:build e2e

// Package webserver browser-level CSP violation tests (Phase 89 D-19, SEC-08 SC-4).
//
// These tests launch headless Chromium via chromedp and navigate to each of
// the three embedded HTML pages with a JS listener attached for
// securitypolicyviolation events (DOM-level, per Research Q1 Option B).
// Any violation observed after the page has rendered and the WS handshake
// (terminal page) has completed marks the test as failed.
//
// Gated by //go:build e2e — default `go test ./...` does NOT run these.
// Run explicitly with: go test -tags=e2e ./internal/webserver/...
//
// Skipped when Chromium is unavailable: the test stack self-skips rather than
// failing, because chromedp reports an "exec: chrome not found" error that
// shouldn't be confused with an actual CSP regression. Manual UAT via
// 89-HUMAN-UAT.md is the fallback coverage path.
package webserver

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// runChromedpAndCollectCSPViolations navigates a fresh headless Chromium
// context to url, waits for the body to be visible, gives WS handshake and
// late asset loads ~2s to complete, then returns any collected CSP violations.
//
// Self-skips if Chromium is unavailable; fails the test only on non-exec errors.
func runChromedpAndCollectCSPViolations(t *testing.T, url string) []map[string]interface{} {
	t.Helper()

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

	initScript := `
      window.__cspViolations = [];
      document.addEventListener('securitypolicyviolation', (e) => {
        window.__cspViolations.push({
          directive: e.violatedDirective,
          blockedURI: e.blockedURI,
          lineNumber: e.lineNumber,
        });
      });
    `
	var violations []map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(initScript).Do(ctx)
			return err
		}),
		chromedp.Navigate(url),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate("window.__cspViolations", &violations),
	)
	if err != nil {
		if strings.Contains(err.Error(), "exec") || strings.Contains(err.Error(), "chrome not found") {
			t.Skipf("chromedp: Chromium unavailable (%v) — run 89-HUMAN-UAT.md instead (Phase 89 D-19 fallback)", err)
		}
		t.Fatalf("chromedp.Run: %v", err)
	}
	return violations
}

func TestBrowserCSP_TerminalNoViolations(t *testing.T) {
	ws, _, _, _ := testServerWithHub(t, "sess-89-e2e-terminal")
	ws.SetSigningKey(capTestKey)
	token := issueCapFor(t, ws, "sess-89-e2e-terminal", "read,write")
	url := ws.BaseURL() + "/sessions/sess-89-e2e-terminal?cap=" + token
	violations := runChromedpAndCollectCSPViolations(t, url)
	if len(violations) > 0 {
		t.Errorf("CSP violations on terminal page (Phase 89 D-19 / SC-4): %+v", violations)
	}
}

func TestBrowserCSP_DashboardNoViolations(t *testing.T) {
	ws, _ := testServer(t)
	url := ws.BaseURL() + "/dashboard"
	violations := runChromedpAndCollectCSPViolations(t, url)
	if len(violations) > 0 {
		t.Errorf("CSP violations on dashboard page (Phase 89 D-19 / SC-4): %+v", violations)
	}
}

func TestBrowserCSP_JoinNoViolations(t *testing.T) {
	ws, _ := testServer(t)
	url := ws.BaseURL() + "/join"
	violations := runChromedpAndCollectCSPViolations(t, url)
	if len(violations) > 0 {
		t.Errorf("CSP violations on join page (Phase 89 D-19 / SC-4): %+v", violations)
	}
}
