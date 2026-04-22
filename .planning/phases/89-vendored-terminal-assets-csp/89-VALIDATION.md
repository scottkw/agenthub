---
phase: 89
slug: vendored-terminal-assets-csp
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-22
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 89-01-* | 01 | 1 | SEC-07 | D-02/D-04/D-20 | xterm assets present under web/vendor/xterm; VERSION file matches pnpm-lock | unit | `go test ./internal/webserver -run TestVendorVersionDrift` | ❌ W0 | ⬜ pending |
| 89-02-* | 02 | 1 | SEC-07/SEC-08 | D-06/D-07 | No inline script/style in the three HTML pages | unit | `go test ./internal/webserver -run TestNoInlineScriptOrStyle` | ❌ W0 | ⬜ pending |
| 89-03-* | 03 | 2 | SEC-07 | D-08/D-14 | /assets/* serves xterm.js, xterm.css, addon-fit.js, terminal.js etc. with 200+correct Content-Type | integration | `go test ./internal/webserver -run TestAssetsRoute` | ❌ W0 | ⬜ pending |
| 89-04-* | 04 | 2 | SEC-08 | D-09/D-10/D-13 | CSP header present on /sessions/{id}, /dashboard, /join with connect-src 'self' wss://host | integration | `go test ./internal/webserver -run TestCSPHeaderStrict` | ❌ W0 | ⬜ pending |
| 89-05-* | 05 | 2 | SEC-07 | D-17 | Source-grep regression guard: no cdn.jsdelivr, no unpkg, no ://cdn., no cross-origin http(s) in src/href | unit | `go test ./internal/webserver -run TestNoCDNReferences` | ❌ W0 | ⬜ pending |
| 89-06-* | 06 | 3 | SEC-08 | D-19 | Real browser loads three pages with zero CSP violations | e2e | `go test -tags=e2e ./internal/webserver -run TestBrowserNoCSPViolations` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/webserver/csp_test.go` — test stubs for TestCSPHeaderStrict + TestAssetsRoute
- [ ] `internal/webserver/vendor_drift_test.go` — stub for TestVendorVersionDrift
- [ ] `internal/webserver/regression_test.go` (or reuse existing) — stub for TestNoCDNReferences + TestNoInlineScriptOrStyle
- [ ] `internal/webserver/browser_csp_e2e_test.go` — `//go:build e2e` stub for chromedp test
- [ ] `go.mod` + `go.sum` — add `github.com/chromedp/chromedp` at a pinned version (e2e-only; do not import in non-tagged code)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Safari (macOS + iOS) renders terminal page with zero CSP violations | SEC-08 SC-4 | chromedp is Chromium-only; Safari parity cannot be automated in-repo | 1) Start daemon in Tailscale mode, 2) Open terminal page in Safari, 3) Open Web Inspector → Console, 4) Attach → resize → scroll → detach, 5) Confirm no `Refused to load ... CSP` messages |
| Zero third-party origin requests during live session | SEC-07 SC-3 | Requires real network inspection, not just source text | 1) DevTools → Network tab, 2) Filter `jsdelivr|unpkg|cdnjs`, 3) Exercise terminal through a full session, 4) Confirm zero matching rows |
| Local-network-fallback HTTPS mode renders all three pages clean | SEC-08 SC-4 | Needs the fallback path and a second client; hard to CI | 1) Disable Tailscale, 2) Trigger local-network fallback, 3) Open /dashboard, /join, /sessions/{id} from a second device, 4) Verify no console errors |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
