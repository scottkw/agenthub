---
phase: 89-vendored-terminal-assets-csp
plan: "05"
subsystem: webserver
tags: [go, csp, security, e2e, chromedp, uat, finding]
dependency_graph:
  requires: [89-01, 89-02, 89-03, 89-04]
  provides: [browser CSP e2e coverage, HUMAN-UAT checklist]
  affects: [phase-89-verification]
tech_stack:
  added: [github.com/chromedp/chromedp@v0.15.1, github.com/chromedp/cdproto]
  patterns: [build-tag-gated-e2e, chromedp-exec-allocator, AddScriptToEvaluateOnNewDocument, DOM-securitypolicyviolation-listener]
key_files:
  created:
    - internal/webserver/browser_csp_e2e_test.go
    - .planning/phases/89-vendored-terminal-assets-csp/89-HUMAN-UAT.md
  modified:
    - go.mod
    - go.sum
decisions:
  - "chromedp@v0.15.1 recorded as concrete version in go.mod (no @latest literal committed)."
  - "AddScriptToEvaluateOnNewDocument wrapped in chromedp.ActionFunc because cdproto returns (scriptID, error), not a plain Action."
  - "Test file first line is //go:build e2e; default `go test ./...` does not compile it."
  - "Skip-on-missing-Chromium pattern: any exec/chrome-not-found error triggers t.Skipf referencing 89-HUMAN-UAT.md; non-exec errors fail."
metrics:
  duration: "~15m"
  completed_date: "2026-04-22T23:30:00Z"
  tasks_completed: 2
  tasks_blocked: 1
  files_created: 2
  files_modified: 2
---

# Phase 89 Plan 05: Browser E2E + HUMAN-UAT

**One-liner:** chromedp-driven CSP violation test (e2e tag), HUMAN-UAT checklist for Safari + local-network-fallback + third-party-request audit. Terminal e2e surfaced a real SC-4 gap; Task 3 checkpoints to user for disposition.

## Tasks Completed

| # | Name | Type | Commit | Result |
|---|------|------|--------|--------|
| 1 | chromedp dep + browser_csp_e2e_test.go | feat | 38f6deb | Build OK, default suite unaffected, e2e: 2/3 PASS, 1 FAIL (see Finding) |
| 2 | 89-HUMAN-UAT.md with 3 manual items + finding disclosure | docs | 0df176f | All acceptance-criteria grep checks OK |
| 3 | Human UAT checkpoint | checkpoint | — | **Blocked on user: a real CSP/xterm incompatibility surfaced** |

## Finding — SC-4 Terminal Route Fails Under Strict D-09 Policy

`TestBrowserCSP_TerminalNoViolations` (run with `go test -tags=e2e`) reports **12 × `style-src-elem 'inline' lineNumber:1`** violations on the terminal route. Dashboard and join routes PASS.

**Root cause:** xterm.js creates `<style>` elements at runtime (cursor, selection, theming hooks) via `document.createElement('style')` + `appendChild`. These count as `style-src-elem` in CSP3 and are blocked by `style-src 'self'` (no `'unsafe-inline'`).

**Conflict:** D-06 and D-09 explicitly lock the policy to "no `'unsafe-inline'`, no nonces, no hashes — pure `'self'` for both script-src and style-src." xterm's runtime behavior cannot satisfy this without one of:
- Add `'unsafe-inline'` to `style-src` — simplest, but weakens policy (unlocks D-06/D-09).
- Switch to nonce or sha256-hash for style-src — requires per-request nonce injection into the CSP header AND into every xterm-generated `<style>` tag (difficult because xterm owns that code path).
- Patch/fork xterm to not inject runtime styles — out-of-proportion for the change.

Decision deferred to the user via the Task 3 checkpoint below.

## Artifacts

### `internal/webserver/browser_csp_e2e_test.go`
`//go:build e2e`-gated file with:
- `runChromedpAndCollectCSPViolations(t, url)` — helper launching headless Chromium, installing a DOM `securitypolicyviolation` listener, navigating, waiting, evaluating `window.__cspViolations`.
- `TestBrowserCSP_TerminalNoViolations` — uses `testServerWithHub` + `issueCapFor` for `/sessions/{id}?cap=...`.
- `TestBrowserCSP_DashboardNoViolations` — `/dashboard` on testServer.
- `TestBrowserCSP_JoinNoViolations` — `/join` on testServer.

### `.planning/phases/89-vendored-terminal-assets-csp/89-HUMAN-UAT.md`
Three-item operator checklist (UAT-1 Safari, UAT-2 local-network-fallback, UAT-3 live-session network audit) with a prominent KNOWN FINDING section that flags the terminal style-src-elem violations.

### `go.mod` / `go.sum`
Added `github.com/chromedp/chromedp v0.15.1` + transitives (`cdproto`, `sysutil`, `gobwas/*`). All test-only — no production import path.

## Test Results

- `go test ./internal/webserver/ -count=1` (default) → PASS (cached, count delta 0 for default suite).
- `go build ./internal/... && go build -tags wailsassets ./` → OK.
- `go test -tags=e2e ./internal/webserver/ -run TestBrowserCSP` on dev Mac:
  - `TestBrowserCSP_DashboardNoViolations` — PASS (2.65s)
  - `TestBrowserCSP_JoinNoViolations` — PASS (2.52s)
  - `TestBrowserCSP_TerminalNoViolations` — FAIL (8.37s) — 12 × style-src-elem inline

## Self-Check: CHECKPOINT

- [x] Task 1: chromedp added, tests written, build/test verifications pass
- [x] Task 2: 89-HUMAN-UAT.md created, all acceptance grep checks pass
- [ ] Task 3: **Awaiting user disposition on terminal SC-4 gap** (see Finding)
- [x] SUMMARY.md written and committed before returning
- [x] No modifications to STATE.md or ROADMAP.md

## Awaiting User

The phase cannot be marked SC-4-complete until the terminal style-src-elem gap is dispositioned. Likely paths:
1. **Accept** `'unsafe-inline'` for style-src → update csp_mw.go + csp_mw_test.go + csp_integration_test.go + CONTEXT D-09; re-run e2e → PASS.
2. **Gap-closure phase** (89.1) scoped to switching style-src to a hash-based policy (collect xterm's runtime style hashes at build time, or migrate to a nonce strategy).
3. **Accept as known limitation** in SECURITY.md and move on — D-09 stays intact, the terminal route is documented as not reaching SC-4 under strict policy, with the other 3/4 criteria verified.

The user should pick one path in the `/gsd-execute-phase` checkpoint response.
