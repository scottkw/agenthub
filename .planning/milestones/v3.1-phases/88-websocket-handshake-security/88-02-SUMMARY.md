---
phase: 88-websocket-handshake-security
plan: "02"
subsystem: relay
tags: [security, websocket, relay, origin-check, go, tdd]
dependency_graph:
  requires: []
  provides: [loopback-only-origin-allowlist-relay, insecure-skip-verify-removed-relay, sc4-relay-regression-guard]
  affects: [internal/relay/server.go]
tech_stack:
  added: []
  patterns: [source-grep-regression-guard, loopback-origin-allowlist, r-host-port-derivation]
key_files:
  modified:
    - internal/relay/server.go
  created:
    - internal/relay/origin_test.go
    - internal/relay/security_regression_test.go
decisions:
  - "Port derivation via r.Host (Pitfall 7 option a): reflects actual listener port without adding constructor parameter"
  - "Fail-closed on malformed Host: nil allowlist falls back to library same-Host check, still loopback-safe"
  - "4-element allowlist covers {http,https} x {localhost,127.0.0.1} for future TLS-fronted relay scenarios"
metrics:
  duration: "2m 7s"
  completed: "2026-04-22T02:21:54Z"
  tasks_completed: 1
  tasks_total: 1
  files_modified: 1
  files_created: 2
---

# Phase 88 Plan 02: Relay Loopback-Only OriginPatterns Summary

**One-liner:** Removed `InsecureSkipVerify: true` from relay WebSocket upgrade; replaced with 4-element loopback-only `OriginPatterns` list derived from `r.Host` port, plus source-grep regression guard.

## What Was Built

### Files Modified

**`internal/relay/server.go`**
- Replaced `InsecureSkipVerify: true` in `handleSession`'s `websocket.AcceptOptions` with `OriginPatterns: loopbackOriginPatterns(r.Host)`
- Added `loopbackOriginPatterns(host string) []string` helper: uses `net.SplitHostPort(r.Host)` to extract the listener port, returns 4 loopback strings (`http://localhost:<port>`, `http://127.0.0.1:<port>`, `https://localhost:<port>`, `https://127.0.0.1:<port>`), or nil if parsing fails (fail-closed)
- Added `"net"` to the import block

### Files Created

**`internal/relay/origin_test.go`**
- `TestServer_LoopbackOrigin127Accepted` — integration test: dial with `Origin: http://127.0.0.1:<port>` succeeds (WS 101)
- `TestServer_LoopbackOriginLocalhostAccepted` — integration test: dial with `Origin: http://localhost:<port>` succeeds (WS 101)
- `TestServer_CrossSiteOriginRejected` — integration test: raw HTTP with `Origin: https://evil.example` is rejected (not 101)
- `TestLoopbackOriginPatterns_DerivesPortFromHost` — unit test: verifies 4-element allowlist, nil on empty/malformed host

**`internal/relay/security_regression_test.go`**
- `TestSecurity_NoInsecureSkipVerifyInRelay` — source-grep guard reads `server.go` and fails if `InsecureSkipVerify: true` appears; mirrors Phase 87's `TestVerify_ConstantTimeComparison` pattern

## Success Criteria Satisfied

| SC | Status | Evidence |
|----|--------|---------|
| SC-4 (relay half): `InsecureSkipVerify: true` absent from code path | PASS | `grep -c "InsecureSkipVerify: true" internal/relay/server.go` = 0; `TestSecurity_NoInsecureSkipVerifyInRelay` passes |
| SC-4 (relay half): regression guard locks the literal out permanently | PASS | `TestSecurity_NoInsecureSkipVerifyInRelay` reads source and fails on reintroduction (D-13 item 2) |
| SC-1 (relay half): cross-site Origin rejected at loopback interface | PASS | `TestServer_CrossSiteOriginRejected` sends `Origin: https://evil.example`, asserts status != 101 |
| SC-1 (relay half): loopback Origin accepted | PASS | `TestServer_LoopbackOrigin127Accepted` and `TestServer_LoopbackOriginLocalhostAccepted` both PASS |

## Port Derivation Decision (Pitfall 7 Option A)

The `relay.Server` struct does not own its listener — the listener is created in `daemon/api.go:137` (`net.Listen("tcp", "127.0.0.1:0")`). To compose the loopback allowlist dynamically at handshake time, we derive the port from `r.Host` inside `handleSession`. This is the reflective approach (option a from RESEARCH Pitfall 7), matching the webserver's `ws.BaseURL()` pattern. No constructor parameter change was needed. `net.SplitHostPort(r.Host)` cleanly extracts the port; on failure, returns nil so the library falls back to its same-Host auto-pass, which is still loopback-safe given the daemon's bind to `127.0.0.1`.

## Library Behavior Observed (coder/websocket v1.8.14)

During testing on macOS (darwin/arm64):

1. **`Origin: http://127.0.0.1:<port>`** — ACCEPTED. The `OriginPatterns` list contains `"http://127.0.0.1:<port>"` (scheme+host form). Library `authenticateOrigin` matches `u.Scheme+"://"+u.Host` = `"http://127.0.0.1:<port>"` against pattern. Match succeeds.

2. **`Origin: http://localhost:<port>`** — ACCEPTED. Pattern `"http://localhost:<port>"` (with `"://"`) causes library to match scheme+host. `http://localhost:<port>` matches. `r.Host` is `127.0.0.1:<port>` so library's auto-pass-when-Origin-equals-Host path does NOT fire — the explicit allowlist is what makes this work. This confirms the allowlist is load-bearing for the `localhost` case, not the library auto-pass.

3. **`Origin: https://evil.example`** — REJECTED. Library returns 403 (non-101). `TestServer_CrossSiteOriginRejected` confirms this via raw HTTP (not `websocket.Dial`, which would return an opaque error).

4. **Default `dialWS` calls (no explicit Origin header)** — ACCEPTED. The library short-circuits to accept when Origin header is absent (library `accept.go:230-232`). All existing relay tests using `dialWS` continue to pass because they are non-browser loopback clients that do not set Origin. This is intentional per CONTEXT D-05 ("the relay is the designated path for non-browser local clients").

## TDD Gate Compliance

- **RED commit** (`e880bb6`): `test(88-02)` — test files created; compile failed with `undefined: loopbackOriginPatterns`. Confirmed RED.
- **GREEN commit** (`7381001`): `feat(88-02)` — implementation added; all 5 new tests pass; full suite green (47/47).
- No REFACTOR commit needed — implementation was clean on first pass.

## Parallel Execution Note

Plan 01 (webserver) and Plan 02 (relay) were executed in the same wave (wave 1) with no shared files — `internal/webserver/` and `internal/relay/` are independent packages. Parallel execution is confirmed safe with zero file conflicts.

## Deviations from Plan

None — plan executed exactly as written. All three edits (server.go modification, origin_test.go creation, security_regression_test.go creation) were applied as specified. The library behavior for `localhost` origin (test point 2 above) confirmed the plan's prediction that the allowlist (not library auto-pass) is what makes localhost matching work.

## Threat Coverage

| Threat ID | Mitigation Delivered |
|-----------|---------------------|
| T-88-10 | `TestSecurity_NoInsecureSkipVerifyInRelay` source-grep guard blocks silent reintroduction of `InsecureSkipVerify: true` |
| T-88-11 | `OriginPatterns` allowlist means future listener rebind to non-loopback would require conscious origin addition |
| T-88-12 | `TestServer_CrossSiteOriginRejected` proves cross-site origin is rejected at loopback interface |
| T-88-13 | `loopbackOriginPatterns(r.Host)` derives port reflectively; `TestLoopbackOriginPatterns_DerivesPortFromHost` covers port parsing edge cases |
| T-88-14 | `net.SplitHostPort` error → nil → fail-closed; unit test covers empty and malformed host inputs |
| T-88-15 | Accepted residual — library's differentiated error body is acceptable for loopback-only service |

## Commits

| Hash | Type | Description |
|------|------|-------------|
| `e880bb6` | test | add failing tests for loopback-only OriginPatterns + regression guard (RED) |
| `7381001` | feat | replace InsecureSkipVerify: true with loopback-only OriginPatterns (GREEN) |

## Self-Check

- [x] `internal/relay/server.go` exists and contains `OriginPatterns: loopbackOriginPatterns(r.Host)` (1 occurrence)
- [x] `internal/relay/server.go` contains `func loopbackOriginPatterns` (1 occurrence)
- [x] `internal/relay/server.go` imports `"net"`
- [x] `internal/relay/server.go` does NOT contain `InsecureSkipVerify: true` (0 occurrences)
- [x] `internal/relay/origin_test.go` exists with 4 required test functions
- [x] `internal/relay/security_regression_test.go` exists with `TestSecurity_NoInsecureSkipVerifyInRelay`
- [x] `go vet ./internal/relay/` exits 0
- [x] `go test ./internal/relay/ -count=1` exits 0 (47/47 pass)
- [x] `gofmt -l` produces no output for all 3 files
- [x] Commits `e880bb6` and `7381001` exist in git log

## Self-Check: PASSED
