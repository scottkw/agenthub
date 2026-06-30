---
phase: 159-web-share-chat-parity-route-shared-session-links-to-the-chat
plan: "01"
subsystem: webserver
tags: [webserver, redirect, web-share, chat-parity, go, tdd]
dependency_graph:
  requires: [Phase 155 WebShareSessionView/ChatPanel, Phase 87 requireCapability]
  provides: [WEBCHAT-01 redirect, WEBCHAT-02 security invariant, PARITY-01 on shared link]
  affects: [internal/webserver/server.go, internal/webserver/server_test.go, TESTING.md]
tech_stack:
  added: [net/url (stdlib, no new dep)]
  patterns: [TDD RED/GREEN, HTTP 302 redirect, url.QueryEscape round-trip]
key_files:
  created: []
  modified:
    - internal/webserver/server.go
    - internal/webserver/server_test.go
    - internal/webserver/csp_integration_test.go
    - TESTING.md
decisions:
  - "302 issued inside handleTerminalPage AFTER requireCapability — route registration unchanged so cap validation always precedes redirect (T-159-02)"
  - "url.QueryEscape applied to both sessionID and cap token — handles any future token format including standard base64 with +/= chars"
  - "terminal.html, terminal.js, and /sessions/{id}/ws preserved (Chesterton's Fence — only the HTML page route redirects)"
  - "csp_integration_test.go updated with no-redirect clients so Phase 89 CSP/Cache-Control invariants are still tested on the redirect response"
metrics:
  duration: "~10 minutes"
  completed: "2026-06-27"
  tasks_completed: 2
  files_modified: 4
status: complete
---

# Phase 159 Plan 01: Web-Share Chat Parity (302 Redirect) Summary

HTTP 302 redirect from `/sessions/{id}?cap=` to `/app/?session=&cap=` so remote web guests land on the chat-capable React SPA instead of the vanilla-JS terminal viewer.

## What Was Built

**Modified handler** (`internal/webserver/server.go` → `handleTerminalPage`): The handler now reads `sessionID := r.PathValue("id")` and `token := r.URL.Query().Get("cap")`, builds `target := fmt.Sprintf("/app/?session=%s&cap=%s", url.QueryEscape(sessionID), url.QueryEscape(token))`, and calls `http.Redirect(w, r, target, http.StatusFound)`. `net/url` was added to the import block. Route registration is unchanged — `requireCapability` always runs before `handleTerminalPage`.

**New test** (`TestTerminalPageRedirect`): TDD RED/GREEN cycle. Tests: valid RW cap → 302 + Location `/app/?session=&cap=`; cap token round-trip via `url.Parse().Query().Get("cap")`; RO cap redirects identically to RW (D-06); standalone URL-encoding sub-test verifies `url.QueryEscape` percent-encodes `+`, `/`, `=` correctly.

**Updated test** (`TestWebServerToggle`): enabled-session case now uses a no-redirect client and expects 302 + Location containing `/app/?session=sess1&cap=`; disabled-session case (403) is unchanged.

**Updated CSP tests** (`csp_integration_test.go`): `TestCSPHeaderStrict_TerminalPage` and `TestCSPHeaderStrict_CacheControl` updated to use no-redirect clients — the `cspHeaders` middleware still sets `Content-Security-Policy` and `Cache-Control: no-store` on the 302 response (Phase 89 D-16/D-18 invariants preserved).

**TESTING.md**: Section 2 Phase 159 manifest delta note; Section 4 WEBCHAT-01 and WEBCHAT-02 traceability rows; Section 5 new Category R with manual item M-31 (6-step live-daemon UAT on the actually-shared `/sessions/{id}?cap=` link).

## Chesterton's Fence Re-Verification

Grep of `/sessions/` across `internal/`, `web/`, `frontend/`:
- All URL-minting code (`internal/tailnet/sessions.go`, `internal/daemon/remote_files.go`) produces `/sessions/{id}?cap=` URLs — every consumer arrives with a cap, so `requireCapability` always gates them.
- The `/sessions/{id}/ws` WebSocket route is separate (`mux.HandleFunc("GET /sessions/{id}/ws", ...)`) and is NOT touched.
- `terminal.html` and `terminal.js` are NOT removed or unembedded — the `web/assets` directory is preserved.
- No non-cap consumer depends on the terminal.html HTML being served at `/sessions/{id}`.

Verdict: redirect is safe; only the HTML page route is affected.

## TDD Gate Compliance

- RED commit `0422002d`: `test(159-01): add failing tests for handleTerminalPage 302 redirect` — tests failing (200 received, 302 expected).
- GREEN commit `99377a43`: `feat(159-01): redirect handleTerminalPage to chat-capable SPA (WEBCHAT-01)` — all tests pass, full package green.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] CSP integration tests broke after the 302 redirect**
- **Found during:** Task 1 full-package test run (after GREEN implementation)
- **Issue:** `TestCSPHeaderStrict_TerminalPage` expected HTTP 200 from `/sessions/{id}`. After Phase 159, the handler returns 302. The default `client` in tests follows redirects, landing on `/app/` which returns 503 (no `staticAppFS` in tests) with no CSP header — causing the test to fail at the 200 assertion and the CSP check.
- **Fix:** Updated both failing tests to use a no-redirect `http.Client` (same CA-trusting transport, `CheckRedirect: http.ErrUseLastResponse`). Updated status assertion from 200 → 302. Added explanatory comments noting that `cspHeaders` sets headers before the redirect fires.
- **Files modified:** `internal/webserver/csp_integration_test.go`
- **Commit:** `99377a43` (included in GREEN commit)

## Threat Surface Scan

No new threat surface introduced. The redirect target is a server-controlled relative path (`/app/?…`); sessionID and token are query params, never used as the redirect host — not an open redirect (T-159-01 mitigated). The 302 fires only after `requireCapability` validates the cap (T-159-02 mitigated). The cap token was already in the URL before this change; `url.QueryEscape` is a correctness measure, not a new exposure (T-159-03 accepted per threat register).

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| `internal/webserver/server.go` exists | FOUND |
| `internal/webserver/server_test.go` exists | FOUND |
| `internal/webserver/csp_integration_test.go` exists | FOUND |
| `TESTING.md` exists | FOUND |
| Commit `0422002d` (RED test) | FOUND |
| Commit `99377a43` (GREEN redirect) | FOUND |
| Commit `fac98a57` (TESTING.md) | FOUND |
