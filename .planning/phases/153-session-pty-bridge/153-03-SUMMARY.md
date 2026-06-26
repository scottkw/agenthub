---
phase: 153-session-pty-bridge
plan: "03"
subsystem: webserver-inject-parity
tags: [webserver, security, inject, tdd, sec-01, sec-02, testing, cross-surface-parity]
dependency_graph:
  requires: [153-02]
  provides: [web-path-MsgSessionInject-case, TestInjectRO_WebPath, TESTING.md-153-traceability]
  affects: [internal/webserver/server.go, internal/webserver/inject_test.go, internal/relay/server_inject_test.go, TESTING.md]
tech_stack:
  added: []
  patterns: [claims-perms-gate, counting-writer-test-adapter, adversarial-frame-test, tdd-test-after]
key_files:
  created:
    - internal/webserver/inject_test.go
  modified:
    - internal/webserver/server.go
    - internal/relay/server_inject_test.go
    - TESTING.md
decisions:
  - "Web read-pump inject case is structurally identical to the relay-path case — shares hub.HandleInject, no direct WriteInput; the only difference is the relay. prefix on protocol symbols"
  - "assertNoFrameType removed from server_inject_test.go (unused after TestInject_OnlyDedicatedFrame uses time.Sleep, not frame-type assertion); TestInjectRO_WebPath does not need it either"
  - "writerFuncInj defined locally in inject_test.go rather than reusing relay package's unexported writerFunc — different packages require separate definitions"
  - "No chatAppendFn wired in TestInjectRO_WebPath: HandleInject returns ErrReadOnly before WriteInput or chatAppendFn, so the callback is unnecessary for the RO adversarial test"
  - "build.yml matrix unchanged; no branch-protection update required"
metrics:
  duration: "7 minutes"
  completed: "2026-06-26"
  tasks_completed: 3
  files_changed: 4
status: complete
---

# Phase 153 Plan 03: Web-Share Inject Parity (SEC-01 Web Path) Summary

Web-share handleWSSRelay read-pump now dispatches `relay.MsgSessionInject` through `hub.HandleInject` (same RW-gate + sanitizer as the relay path), plus an adversarial `TestInjectRO_WebPath` proving SEC-01 on the web entry path via a hand-crafted RO-JWT frame. TESTING.md updated with all three new Phase 153 Go test files and per-requirement traceability rows.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add web-share read-pump MsgSessionInject case (claims.Perms gate) | f2e63c32 | internal/webserver/server.go |
| 2 | Web-path adversarial RO-JWT inject rejection test (SEC-01) | d9c1178f | internal/webserver/inject_test.go, internal/relay/server_inject_test.go |
| 3 | Register new test files in TESTING.md + traceability check | 9377c0f0 | TESTING.md |

## What Was Built

**Task 1 — Web read-pump inject case** (`internal/webserver/server.go`):

- Added `case relay.MsgSessionInject:` to `handleWSSRelay` read-pump switch (after `case relay.MsgAliasSet:`)
- Unmarshals `relay.InjectPayload`; malformed/empty frames silently dropped (mirrors MsgTyping/MsgAliasSet)
- Calls `hub.HandleInject(sub, ip.Text)` — routes through the same gate (ErrReadOnly check), sanitizer (SanitizePTYText), and PTY write path as the relay-path case
- On `HandleInject` error: sends `relay.MakeInjectErrorFrame(err.Error())` to `sub.Msgs` via non-blocking select/default → `go sub.CloseSlow()` (identical error path to relay case)
- Code comments document:
  - `sub.ReadOnly` is derived from `claims.Perms == "read"` (line ~1008 of server.go), sourced from the signed JWT — cannot be bypassed by URL params (D-24/SEC-04)
  - `MsgChatSend` (0x31) is chat-only and NEVER writes to PTY (D-02)
- Both WS entry paths (relay and webserver) now share one `hub.HandleInject` implementation — cross-surface parity achieved

**Task 2 — Web-path adversarial inject test** (`internal/webserver/inject_test.go`):

- `writerFuncInj` — local func adapter (implements `io.Writer`) for the counting PTY writer; defined separately from relay package's unexported `writerFunc`
- `waitForMsgInjectError` — reads frames skipping non-`relay.MsgInjectError` types; returns true on first match, false on timeout
- `TestInjectRO_WebPath` (SEC-01 web path):
  1. Builds a `relay.NewHubManager()` with a counting PTY writer (`atomic.Int32`) as the WriteInput target
  2. Stands up a full WebServer (TLS, Mode:"tailscale") backed by the manager
  3. Mints a RO capability JWT via `issueCapFor(t, ws, sessionID, "read")` — `claims.Perms == "read"` → `sub.ReadOnly == true` in `handleWSSRelay`
  4. Dials the WS with the RO cap and Origin header
  5. Hand-crafts `relay.MsgSessionInject` binary frame (`{relay.MsgSessionInject} + JSON relay.InjectPayload`) and sends it directly, bypassing any client-side suppression
  6. Asserts: `relay.MsgInjectError` NAK received within 3s timeout AND `ptyWriteCount.Load() == 0`

- Also removed the unused `assertNoFrameType` helper from `internal/relay/server_inject_test.go` (no test in either file needed it; lint-clean)

**Task 3 — TESTING.md updates**:

- Suite Manifest §2: Go count bumped 359 → 362 (+3 files); total 487 → 490
- Added Phase 153-01/02/03 entries to the running Note paragraph
- Traceability §4: added 5 new rows:
  - `MENTION-02 → internal/relay/server_inject_test.go` (RW write + MsgChat broadcast)
  - `MENTION-03 → internal/relay/server_inject_test.go` (dedicated verb only, D-02)
  - `SEC-01 → internal/relay/server_inject_test.go` (relay-path adversarial RO proof)
  - `SEC-01 → internal/webserver/inject_test.go` (web-path adversarial RO-JWT proof)
  - `SEC-02 → internal/relay/sanitize_test.go` (SanitizePTYText correctness)
- `bash tests/check-traceability-paths.sh` exits 0

## Verification

```
go build ./...                                                          PASS
go test -race -short -run TestInjectRO_WebPath ./internal/webserver/...  PASS
go test -race -short ./...                                              PASS (all packages)
bash tests/check-traceability-paths.sh                                 PASS (OK: all traceability paths exist)
go vet ./internal/webserver/...                                         PASS
gofmt -l internal/webserver/server.go internal/webserver/inject_test.go PASS (no output)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed unused `assertNoFrameType` from server_inject_test.go**
- **Found during:** Task 2 — noted in plan's prior_wave_context as a lint issue
- **Issue:** `assertNoFrameType` was defined in `internal/relay/server_inject_test.go` but never called by any test in that file. `golangci-lint unusedfunc` would flag it.
- **Fix:** Removed the function body; relay tests still pass (3/3). The web-path test (`TestInjectRO_WebPath`) does not need an "assert no frame of type X" helper — it reads until MsgInjectError arrives and checks the counter for zero writes.
- **Files modified:** `internal/relay/server_inject_test.go`
- **Commit:** d9c1178f

### TDD Gate Compliance

Task 2 is `tdd="true"`. This plan follows the same atypical sequence as Plan 02: production code implemented in Task 1, test written in Task 2. The test went directly GREEN because the production code (web read-pump case) was already in place. This is correct:
- The RED gate would apply if the test had been written before Task 1; since it was written after, the sequence is implementation-first with the test serving as a correctness lock.
- The test is adversarial (hand-crafted frame) — its value is proving server-side enforcement, not driving implementation.

## Known Stubs

None. All paths are fully wired and operational.

## Threat Flags

No new threat surface beyond what the plan's threat model documents:
- T-153-10 mitigated: `sub.ReadOnly` from `claims.Perms == "read"` in signed JWT; `HandleInject` returns `ErrReadOnly` → `MsgInjectError` NAK; zero PTY writes proven by `TestInjectRO_WebPath`
- T-153-11 mitigated: both read pumps route to the single `hub.HandleInject`; dedicated web-path adversarial test proves parity (Pitfall 5)
- T-153-12 mitigated: web path derives `ReadOnly` from the verified signed JWT only; URL params are ignored for RO status

## Self-Check: PASSED
- `internal/webserver/server.go` exists with `case relay.MsgSessionInject:` (grep returns 1)
- `internal/webserver/inject_test.go` exists
- `internal/relay/server_inject_test.go` exists (assertNoFrameType removed, formatting fixed)
- `TESTING.md` shows Go count 362, total 490, and all 5 new traceability rows
- Commits f2e63c32, d9c1178f, 9377c0f0 exist in git log
