---
phase: 89
slug: vendored-terminal-assets-csp
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-22
updated: 2026-04-22
---

# Phase 89 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib testing) + github.com/chromedp/chromedp (e2e tag) |
| **Config file** | none — Go test is discovery-based; chromedp gated via `//go:build e2e` |
| **Quick run command** | `go test ./internal/webserver/...` |
| **Full suite command** | `go test ./... && go test -tags=e2e ./internal/webserver/...` |
| **Estimated runtime** | ~15s quick, ~60s including e2e |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/webserver/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green; e2e run at least once (locally or CI with Chromium available)
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

Test names in this table are the GROUND TRUTH from the five plan files. Plans are authoritative; VALIDATION.md follows.

| Task Family | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| vendor-drift | 89-01 | 1 | SEC-07 | T-89-02/T-89-05 | xterm assets present under web/vendor/xterm (D-01/D-02); VERSION file matches pnpm-lock | unit | `go test ./internal/webserver/ -run TestXtermVendorVersionsMatchPnpmLock` | ❌ W0 | ⬜ pending |
| html-extraction | 89-02 | 1 | SEC-07/SEC-08 | T-89-01/T-89-02 | No inline script/style in the three HTML pages; /assets/xterm/* URL refs replace cdn.jsdelivr.net | unit | `go test ./internal/webserver/ -run TestSecurity_NoInlineScriptOrStyleInHTML` (see Plan 04 Task 5 — the source-grep regression validates Plan 02's output) | ❌ W0 | ⬜ pending |
| csp-middleware | 89-03 | 1 | SEC-08 | T-89-01/T-89-03/T-89-06 | cspHeaders middleware sets Content-Security-Policy + Cache-Control: no-store; fails closed on empty BaseURL; no unsafe-* tokens | unit | `go test ./internal/webserver/ -run TestCSPHeaders` (8 tests: HeaderSet, RequiredTokens, NoUnsafeTokens, WSSComposition, NoWildcardOutsideDataScheme, CacheControlNoStore, CallsNext, FailsClosedOnEmptyBaseURL) | ❌ W0 | ⬜ pending |
| assets-route | 89-04 | 2 | SEC-07 | T-89-02/T-89-04/T-89-07 | /assets/xterm/* serves from vendor/xterm fs.Sub; /assets/* serves from assets fs.Sub; public tier; Cache-Control: no-store on both mounts | integration | `go test ./internal/webserver/ -run TestAssets` (8 tests: XtermJSServedFromEmbed, XtermCSSServedFromEmbed, FirstPartyJS, FirstPartyCSS, CacheControlNoStore, PublicNoCapNeeded, NotFound, NoDirectoryListing) | ❌ W0 | ⬜ pending |
| csp-integration | 89-04 | 2 | SEC-08 | T-89-01/T-89-03 | CSP header present on /sessions/{id}, /dashboard, /join with all D-18 tokens incl. connect-src 'self' wss://host; present even on 401 | integration | `go test ./internal/webserver/ -run TestCSPHeaderStrict` (5 tests: TerminalPage, Dashboard, Join, CacheControl, OnAuthFailure) | ❌ W0 | ⬜ pending |
| no-cdn-regression | 89-04 | 2 | SEC-07 | T-89-02 | Source-grep regression guard: no cdn.jsdelivr, no unpkg, no ://cdn., no cross-origin http(s) in src/href; no inline script/style blocks | unit | `go test ./internal/webserver/ -run "TestSecurity_(NoCDN\|NoInlineScript)"` (2 tests: NoCDNReferencesInWebAssets, NoInlineScriptOrStyleInHTML) | ❌ W0 | ⬜ pending |
| browser-e2e | 89-05 | 3 | SEC-08 | T-89-01/T-89-03 | Real headless Chromium loads three pages with zero securitypolicyviolation events | e2e | `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP` (3 tests: TerminalNoViolations, DashboardNoViolations, JoinNoViolations) | ❌ W0 | ⬜ pending |
| human-uat | 89-05 | 3 | SEC-07/SEC-08 | T-89-08 | Safari + local-fallback + live-network audit (items UAT-1, UAT-2, UAT-3) signed off in 89-HUMAN-UAT.md | manual | `grep -q 'Phase 89 Manual UAT — COMPLETE' .planning/phases/89-vendored-terminal-assets-csp/89-HUMAN-UAT.md` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Wave mapping:** Plan 03 is Wave 1 (parallel with Plans 01 and 02 — files_modified disjoint from them). Plan 04 is Wave 2 (depends on all Wave 1 plans). Plan 05 is Wave 3 (depends on 04). See each plan's frontmatter `wave` field for the ground truth.

---

## Wave 0 Requirements

- [ ] `internal/webserver/csp_mw_test.go` — 8 TestCSPHeaders_* tests (Plan 03 Task 1)
- [ ] `internal/webserver/csp_integration_test.go` — 5 TestCSPHeaderStrict_* tests (Plan 04 Task 4)
- [ ] `internal/webserver/assets_test.go` — 8 TestAssets_* tests (Plan 04 Task 3)
- [ ] `internal/webserver/vendor_drift_test.go` — TestXtermVendorVersionsMatchPnpmLock (Plan 01 Task 2)
- [ ] `internal/webserver/no_cdn_regression_test.go` — TestSecurity_NoCDNReferencesInWebAssets + TestSecurity_NoInlineScriptOrStyleInHTML (Plan 04 Task 5)
- [ ] `internal/webserver/browser_csp_e2e_test.go` — `//go:build e2e`-gated 3 TestBrowserCSP_* tests (Plan 05 Task 1)
- [ ] `go.mod` + `go.sum` — add `github.com/chromedp/chromedp` at a concrete vX.Y.Z version (e2e-only; do not import in non-tagged code)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Safari (macOS + iOS) renders terminal page with zero CSP violations | SEC-08 SC-4 | chromedp is Chromium-only; Safari parity cannot be automated in-repo | See 89-HUMAN-UAT.md UAT-1 |
| Zero third-party origin requests during live session | SEC-07 SC-3 | Requires real network inspection, not just source text | See 89-HUMAN-UAT.md UAT-3 |
| Local-network-fallback HTTPS mode renders all three pages clean | SEC-08 SC-4 | Needs the fallback path and a second client; hard to CI | See 89-HUMAN-UAT.md UAT-2 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
