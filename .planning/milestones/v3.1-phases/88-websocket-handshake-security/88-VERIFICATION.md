---
phase: 88-websocket-handshake-security
verified: 2026-04-21T00:00:00Z
status: human_needed
score: 3/4 must-haves verified (SC-2 local-HTTPS-fallback requires manual UAT)
overrides_applied: 0
human_verification:
  - test: "SC-2 local-HTTPS-fallback: open share link in browser on same LAN with self-signed cert, disable tailnet first"
    expected: "Terminal page loads and WebSocket upgrade completes (101); devtools shows Origin: https://<host-ip>:<port> accepted with no user-visible error"
    why_human: "Requires a live self-signed cert loaded in browser, LAN reachability, and browser cert trust dismissal — cannot be reproduced by httptest or an automated dial"
  - test: "SC-2 tailscale-mode UAT: open share link from another tailnet node browser"
    expected: "Terminal page attaches (WS 101); devtools confirms Origin: https://<host>.<tailnet>.ts.net:<port> accepted"
    why_human: "Requires live tailnet with magicsock active — httptest cannot reproduce a real Tailscale FQDN listener"
---

# Phase 88: WebSocket Handshake Security Verification Report

**Phase Goal:** Cross-site WebSocket hijacking is blocked at the handshake; only browsers whose `Origin` matches the server's own serving origin can complete the upgrade.
**Verified:** 2026-04-21
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC-1: Cross-site Origin -> 403 before capability check runs | VERIFIED | `TestSecurity_WebSocketRejectsCrossSiteOrigin` passes; `TestSecurity_OriginCheckShortCircuitsBeforeCapability` confirms 403 wins over 401 when cap is also invalid, proving short-circuit order |
| 2 | SC-2: Terminal page completes upgrade in tailscale and local-HTTPS modes | PARTIAL | Tailscale-mode automated proxy `TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode` passes (httptest TLS listener with matching Origin -> 101). Local-HTTPS-fallback with a real self-signed cert and browser trust requires manual UAT |
| 3 | SC-3: Missing-Origin -> documented policy (403) rather than accept-all | VERIFIED | `TestSecurity_WebSocketRejectsMissingOrigin` passes; `TestRequireAllowedOrigin_MissingOriginRejected` unit test passes; policy is explicit (D-05: non-browser clients use loopback relay) |
| 4 | SC-4: `OriginPatterns: ["*"]` / `InsecureSkipVerify: true` accept-all gone; regression test fails if reintroduced | VERIFIED | `grep -c 'OriginPatterns: []string{"*"}' internal/webserver/server.go` = 0; `grep -c 'InsecureSkipVerify: true' internal/relay/server.go` = 0; `TestSecurity_NoAcceptAllOriginInWebserver` GREEN; `TestSecurity_NoInsecureSkipVerifyInRelay` GREEN; `TestSecurity_WebserverOriginAllowlistWiredToBaseURL` GREEN |

**Score:** 3/4 truths fully verified automated (SC-2 local-HTTPS half requires human)

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/webserver/origin_mw.go` | `requireAllowedOrigin` middleware + `allowedOrigins()` helper | VERIFIED | Both functions present; exact signatures match plan; `http.Error(w, "forbidden", http.StatusForbidden)` appears in both rejection paths |
| `internal/webserver/origin_mw_test.go` | 7 unit tests for requireAllowedOrigin | VERIFIED | All 7 literal test names present and passing: `TestRequireAllowedOrigin_MatchingOriginPasses`, `TestRequireAllowedOrigin_MismatchRejected`, `TestRequireAllowedOrigin_MissingOriginRejected`, `TestRequireAllowedOrigin_StrictCaseSensitive`, `TestRequireAllowedOrigin_OriginNullRejected`, `TestRequireAllowedOrigin_FailClosedWhenListenerNotReady`, `TestAllowedOrigins_ReturnsBaseURLSingleton` |
| `internal/webserver/origin_integration_test.go` | WSS handshake integration tests | VERIFIED | 6 tests present and passing (plan called for 5 required + optional 6th; all 6 delivered): cross-site reject, missing-origin reject, matching-origin accept, short-circuit proof, body check, library belt-and-suspenders |
| `internal/webserver/security_regression_test.go` | SC-4 source-grep guards (D-13 items 1 & 3) | VERIFIED | `TestSecurity_NoAcceptAllOriginInWebserver` and `TestSecurity_WebserverOriginAllowlistWiredToBaseURL` both present and GREEN |
| `internal/webserver/server.go` | Route wired with requireAllowedOrigin; OriginPatterns uses allowedOrigins() | VERIFIED | `ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay))` at line 400 (grep count=1); `OriginPatterns: ws.allowedOrigins()` at line 641 (grep count=1); `OriginPatterns: []string{"*"}` count=0 |
| `internal/relay/server.go` | loopbackOriginPatterns replacing InsecureSkipVerify | VERIFIED | `OriginPatterns: loopbackOriginPatterns(r.Host)` count=1; `func loopbackOriginPatterns` present; `InsecureSkipVerify: true` count=0; `"net"` imported |
| `internal/relay/origin_test.go` | Integration tests for loopback allowlist | VERIFIED | 4 tests present and passing: `TestServer_LoopbackOrigin127Accepted`, `TestServer_LoopbackOriginLocalhostAccepted`, `TestServer_CrossSiteOriginRejected`, `TestLoopbackOriginPatterns_DerivesPortFromHost` |
| `internal/relay/security_regression_test.go` | SC-4 regression guard for relay (D-13 item 2) | VERIFIED | `TestSecurity_NoInsecureSkipVerifyInRelay` present and GREEN |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/webserver/server.go setupRoutes` | `requireAllowedOrigin` | `mux.HandleFunc "GET /sessions/{id}/ws"` | WIRED | Pattern `ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay))` confirmed at line 400 |
| `internal/webserver/server.go handleWSSRelay` | `ws.allowedOrigins()` | `websocket.AcceptOptions.OriginPatterns` | WIRED | `OriginPatterns: ws.allowedOrigins()` confirmed at line 641 |
| `internal/webserver/origin_mw.go requireAllowedOrigin` | `ws.BaseURL()` | strict exact match on `r.Header.Get("Origin")` | WIRED | Pattern `origin != allowed` with `allowed = ws.BaseURL()` confirmed in middleware body |
| `internal/relay/server.go handleSession` | `loopbackOriginPatterns(r.Host)` | `websocket.AcceptOptions.OriginPatterns` | WIRED | `OriginPatterns: loopbackOriginPatterns(r.Host)` confirmed |

---

## Data-Flow Trace (Level 4)

The artifacts in this phase are security middleware and test infrastructure — they gate/reject/allow but do not render dynamic data to users. Level 4 data-flow trace is not applicable: no component fetches or renders state that could be hollow.

---

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Cross-site Origin rejected with 403 | `go test ./internal/webserver/... -run TestSecurity_WebSocketRejectsCrossSiteOrigin -count=1` | PASS | PASS |
| Missing Origin rejected with 403 | `go test ./internal/webserver/... -run TestSecurity_WebSocketRejectsMissingOrigin -count=1` | PASS | PASS |
| Matching Origin completes upgrade | `go test ./internal/webserver/... -run TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode -count=1` | PASS | PASS |
| Origin short-circuits before capability | `go test ./internal/webserver/... -run TestSecurity_OriginCheckShortCircuitsBeforeCapability -count=1` | PASS | PASS |
| Rejection body is generic "forbidden" | `go test ./internal/webserver/... -run TestSecurity_OriginRejectionBodyIsForbidden -count=1` | PASS | PASS |
| SC-4 regression guard (webserver) | `go test ./internal/webserver/... -run TestSecurity_NoAcceptAllOriginInWebserver -count=1` | PASS | PASS |
| SC-4 positive wiring guard (webserver) | `go test ./internal/webserver/... -run TestSecurity_WebserverOriginAllowlistWiredToBaseURL -count=1` | PASS | PASS |
| Relay loopback 127.0.0.1 accepted | `go test ./internal/relay/... -run TestServer_LoopbackOrigin127Accepted -count=1` | PASS | PASS |
| Relay loopback localhost accepted | `go test ./internal/relay/... -run TestServer_LoopbackOriginLocalhostAccepted -count=1` | PASS | PASS |
| Relay cross-site origin rejected | `go test ./internal/relay/... -run TestServer_CrossSiteOriginRejected -count=1` | PASS | PASS |
| SC-4 regression guard (relay) | `go test ./internal/relay/... -run TestSecurity_NoInsecureSkipVerifyInRelay -count=1` | PASS | PASS |
| Full webserver package (no Phase 87 regression) | `go test ./internal/webserver/... -count=1` | ok 1.099s | PASS |
| Full relay package (no regression) | `go test ./internal/relay/... -count=1` | ok 0.729s | PASS |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SEC-06 | 88-01-PLAN.md, 88-02-PLAN.md | WebSocket upgrade rejects requests whose Origin is not in server allowlist | SATISFIED | requireAllowedOrigin middleware wired on webserver WSS route; loopbackOriginPatterns wired on relay; both SC-4 regression guards GREEN; automated tests cover SC-1, SC-3, SC-4 fully; SC-2 automated (tailscale-mode half) passes, local-HTTPS half requires manual UAT |

---

## Anti-Patterns Found

| File | Pattern | Severity | Impact |
|------|---------|----------|--------|
| None | — | — | — |

Scan performed on all phase-created and phase-modified files:
- `internal/webserver/origin_mw.go` — no TODOs, no empty implementations, no hardcoded stubs
- `internal/webserver/origin_mw_test.go` — no TODOs, no placeholder tests
- `internal/webserver/origin_integration_test.go` — no TODOs, all 6 tests substantive with real assertions
- `internal/webserver/security_regression_test.go` — source-grep guards with actionable failure messages
- `internal/webserver/server.go` (modified lines) — `OriginPatterns: []string{"*"}` is absent; `ws.allowedOrigins()` present; composition order correct
- `internal/relay/server.go` (modified lines) — `InsecureSkipVerify: true` absent; `loopbackOriginPatterns(r.Host)` present
- `internal/relay/origin_test.go` — 4 substantive tests, no placeholders
- `internal/relay/security_regression_test.go` — source-grep guard with actionable failure message

---

## Human Verification Required

### 1. SC-2 Local-HTTPS-Fallback Terminal Upgrade

**Test:** Disable tailnet. Start agenthub in local mode. Open the share link in a browser on the same LAN. Dismiss the self-signed cert warning. Confirm the terminal page attaches.
**Expected:** WebSocket upgrade completes (101 Switching Protocols visible in devtools Network tab). No user-visible error or infinite spinner. devtools shows `Origin: https://<host-ip>:<port>` accepted.
**Why human:** Requires a live self-signed cert loaded in a real browser (cert trust dismissal), LAN reachability to the host, and a running agenthub process — none of which httptest or automated WebSocket dials can simulate.

### 2. SC-2 Tailscale-Mode Terminal Upgrade (full end-to-end)

**Test:** Start agenthub on a tailnet-joined node. Open the share link from a different tailnet node's browser. Confirm the terminal page attaches.
**Expected:** WebSocket upgrade completes (101). devtools Network tab shows `Origin: https://<host>.<tailnet>.ts.net:<port>` accepted with no rejection.
**Why human:** Requires an active Tailscale daemon (magicsock), a live FQDN-based TLS listener, and browser-side navigation to the share URL — cannot be simulated by the httptest TLS server used in automated tests.

---

## Gaps Summary

No automated gaps. All programmatically verifiable success criteria are met:

- SC-1 (cross-site rejected before capability) — VERIFIED by integration tests and unit tests
- SC-3 (missing Origin -> 403 per documented policy) — VERIFIED by integration tests and unit tests
- SC-4 (accept-all gone + regression guards) — VERIFIED by source-grep tests (both packages GREEN) and direct grep counts (0 occurrences of banned literals)
- SC-2 automated proxy (tailscale-mode httptest path) — VERIFIED by `TestSecurity_WebSocketAcceptsMatchingOriginTailscaleMode`

SC-2 local-HTTPS-fallback half and full live tailnet UAT are the only remaining items, and both are explicitly classified as manual-only in 88-VALIDATION.md.

---

_Verified: 2026-04-21_
_Verifier: Claude (gsd-verifier)_
