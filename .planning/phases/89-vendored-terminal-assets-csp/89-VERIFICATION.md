---
phase: 89-vendored-terminal-assets-csp
status: resolved
verified_date: 2026-04-22
human_verified_date: 2026-05-02
verifier: orchestrator-inline
human_verifier: Ken Scott
must_have_score: "12/12 automated + 3/3 HUMAN-UAT"
requirement_ids: [SEC-07, SEC-08]
human_verification_count: 3
---

# Phase 89 Verification — Vendored Terminal Assets + CSP

**Goal:** The interactive terminal page loads only from the embedded binary and is protected by a Content-Security-Policy that blocks inline/remote script injection.

**Requirements:** SEC-07 (no `https://cdn.jsdelivr.net` at runtime), SEC-08 (CSP with `script-src`/`style-src`/`connect-src` restricted to `self` plus WS origin).

## Status: human_needed

All automated verification checks pass. Three manual verification items remain in `89-HUMAN-UAT.md` and require operator attention.

## Automated Verification Results

### Plan-level must_haves (from frontmatter `must_haves.truths`)

| Plan | Claim | Evidence | Status |
|------|-------|----------|--------|
| 89-01 | xterm.js/css/addon-fit.js committed under `web/vendor/xterm/` | `ls web/vendor/xterm/` lists all 4 files | ✓ |
| 89-01 | Drift test fails on version mismatch vs pnpm-lock | `TestXtermVendorVersionsMatchPnpmLock` PASS | ✓ |
| 89-02 | All inline `<script>`/`<style>` blocks extracted | `TestSecurity_NoInlineScriptOrStyleInHTML` PASS; `web/assets/{terminal,dashboard,join}.{js,css}` present | ✓ |
| 89-02 | CDN URLs swapped for `/assets/xterm/*` | `TestSecurity_NoCDNReferencesInWebAssets` PASS; 5 CDN hosts checked | ✓ |
| 89-03 | `cspHeaders` middleware composes D-09 policy per request | 8/8 unit tests PASS (`TestCSPHeaders_*`) | ✓ |
| 89-03 | Fail-closed on empty BaseURL | `TestCSPHeaders_FailsClosedOnEmptyBaseURL` PASS | ✓ |
| 89-04 | `/assets/xterm/xterm.js` served from embed | `TestAssets_XtermJSServedFromEmbed` PASS | ✓ |
| 89-04 | `/assets/terminal.js` served from embed | `TestAssets_FirstPartyJS` PASS | ✓ |
| 89-04 | CSP header on `/dashboard`, `/join`, `/sessions/{id}` | 3/3 `TestCSPHeaderStrict_*` route tests PASS | ✓ |
| 89-04 | D-18 five-assertion suite per route | All passing (adjusted D-18.3 for style-src amendment) | ✓ |
| 89-04 | Source-grep regression test | `TestSecurity_No*` 3/3 PASS | ✓ |
| 89-05 | Browser-level e2e CSP test with `//go:build e2e` | 3/3 `TestBrowserCSP_*` PASS in Chromium | ✓ |

### Requirement Traceability

| Req | Description | Evidence | Verdict |
|-----|-------------|----------|---------|
| SEC-07 | No `cdn.jsdelivr.net` at runtime | xterm vendored; terminal.html uses `/assets/xterm/*`; `TestSecurity_NoCDNReferencesInWebAssets` enforces; e2e loads pages from server only | ✓ met (automated) |
| SEC-08 | CSP restricting script-src/style-src/connect-src to `self` + WS origin | `cspHeaders` middleware wired on three HTML routes; D-09 policy (amended to allow `'unsafe-inline'` on style-src only per xterm runtime finding); script-src stays strict `'self'`; connect-src is `'self' wss://<host>`; 13 unit+integration tests + 3 e2e tests PASS | ✓ met (automated Chromium) |

### Build & Test Gates

- `go build ./internal/... && go build -tags wailsassets ./` — OK
- `go test ./internal/webserver/ -count=1` — PASS (cached)
- `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP -count=1` — 3/3 PASS
- `go test ./internal/... ./` — ALL PASS (attach, capability, daemon, pty, relay, status, statusbar, tailnet, tui, updater, webserver, root)
- Schema drift check — no drift

## Known Deviation: D-09 Amendment

Original D-09 locked `style-src 'self'` with no `'unsafe-inline'`. Chromium e2e surfaced 12 × `style-src-elem 'inline'` violations on `/sessions/{id}` (xterm runtime style injection). Per user disposition during plan 89-05 execution, D-09 was amended to allow `'unsafe-inline'` on style-src only. `script-src` remains strict `'self'`, so Finding 4's CDN-injection class is still fully blocked. Change recorded in commit `9263a01` and annotated in `89-CONTEXT.md` D-09 block.

## Human Verification Required

Three items in `89-HUMAN-UAT.md` cannot be automated in CI:

1. **UAT-1 — Safari CSP compliance (Tailscale mode).** chromedp is Chromium-only; Safari (WebKit) has a distinct CSP engine. Operator verifies terminal/dashboard/join render without console CSP violations in Safari.

2. **UAT-2 — Local-network-fallback HTTPS mode.** Requires a second device on the same LAN, self-signed cert + Basic Auth flow. CI cannot simulate.

3. **UAT-3 — Live-session Network tab audit.** Verifies zero third-party origin requests during a multi-action terminal session with `jsdelivr|unpkg|cdnjs|cdn\.` filter.

## Next Steps

Operator completes `89-HUMAN-UAT.md` and reports results. Phase 89 formally closes after those three sign-offs are recorded.

Execute `/gsd-verify-work 89` or manually edit `89-HUMAN-UAT.md` with results.
