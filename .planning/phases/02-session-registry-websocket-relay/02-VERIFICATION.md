---
phase: 02-session-registry-websocket-relay
verified: 2026-03-18T14:10:00Z
status: passed
score: 7/7 must-haves verified
re_verification: false
---

# Phase 2: Session Registry and WebSocket Relay Verification Report

**Phase Goal:** Session registry and WebSocket relay
**Verified:** 2026-03-18T14:10:00Z
**Status:** passed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | PTY output is framed with a single-byte type prefix for unambiguous parsing | VERIFIED | `protocol.go` defines MsgOutput=0x01 through MsgPing=0x12; `ParseFrame` splits type byte from payload; 9 protocol tests pass under -race |
| 2 | Scrollback buffer stores bounded PTY output and replays it as a snapshot | VERIFIED | `scrollback.go` front-trims on overflow; `Snapshot()` returns isolated copy; 6 scrollback tests pass including concurrent safety |
| 3 | Hub drains PTY output from a single goroutine and fans out to N subscribers | VERIFIED | `hub.go` `Run()` is the sole reader; `broadcast()` iterates subscriber map under lock; 9 hub unit tests pass |
| 4 | Slow subscribers are disconnected without blocking fast subscribers | VERIFIED | Non-blocking select with `go sub.CloseSlow()` in `broadcast()`; `TestHub_SlowClientDisconnected` and `TestHub_SlowSubscriberGetsDisconnected` both pass |
| 5 | Two WebSocket clients connected to the same session both receive the same PTY output simultaneously | VERIFIED | `TestHub_TwoClientsFanOut` passes with real WS connections via httptest.Server |
| 6 | A disconnected client reconnects and receives scrollback replay then resumes live output | VERIFIED | `TestHub_ReconnectScrollback` passes; subscribe-before-snapshot anti-race pattern confirmed in server.go lines 69/74 |
| 7 | Input from any connected client reaches the PTY and produces output visible to all clients | VERIFIED | `TestHub_InputFanOut` passes; MsgInput dispatch calls `hub.WriteInput(payload)` at server.go line 95 |

**Score:** 7/7 truths verified

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/protocol.go` | Binary framing constants and encode/decode helpers | VERIFIED | Exports MsgOutput, MsgInput, MsgResize, MsgResize2, MsgTitle, MsgPing, MakeOutputFrame, MakeInputFrame, MakeResizeFrame, ParseFrame — all present and substantive |
| `internal/relay/scrollback.go` | Bounded scrollback buffer with Append and Snapshot | VERIFIED | Exports Scrollback, NewScrollback, DefaultScrollbackBytes=262144; mutex-protected; front-trim overflow; snapshot returns copy |
| `internal/relay/hub.go` | Per-session fan-out hub with drain goroutine | VERIFIED | Exports Hub, NewHub, Subscriber; Run reads in 32KB chunks, wraps MakeOutputFrame, appends to scrollback, broadcasts; Shutdown uses sync.Once |
| `internal/relay/manager.go` | HubManager: create/get/remove Hubs per session ID | VERIFIED | Exports HubManager, NewHubManager; Create is idempotent, starts `go hub.Run()`; Get, Remove, Shutdown, SessionIDs all implemented |
| `internal/relay/server.go` | HTTP server with WebSocket handler and REST endpoints | VERIFIED | Exports Server, NewServer; routes GET /sessions/{id}/ws and GET /sessions; subscribe-before-snapshot ordering enforced; bidirectional pumps present |
| `internal/relay/server_test.go` | Integration tests proving all 3 SESS-03 success criteria | VERIFIED | 4 integration tests using real WS connections (httptest.NewServer + coder/websocket.Dial); all pass under -race |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/relay/hub.go` | `internal/pty/session.go` | Hub reads from Session (io.Reader interface) | VERIFIED | Hub accepts `io.Reader` — Session implements io.ReadWriter from Phase 1; hub.go line 72: `h.reader.Read(buf)` |
| `internal/relay/hub.go` | `internal/relay/scrollback.go` | Hub appends every frame to scrollback | VERIFIED | hub.go line 77: `h.scrollback.Append(frame)` — called for every read in Run() |
| `internal/relay/hub.go` | `internal/relay/protocol.go` | Hub uses MakeOutputFrame to wrap PTY bytes | VERIFIED | hub.go line 76: `frame := MakeOutputFrame(buf[:n])` |
| `internal/relay/server.go` | `internal/relay/manager.go` | Server.handleSession looks up Hub from HubManager | VERIFIED | server.go line 41: `hub, ok := s.manager.Get(sessionID)` |
| `internal/relay/server.go` | `github.com/coder/websocket` | websocket.Accept upgrades HTTP to WS | VERIFIED | server.go line 47: `websocket.Accept(w, r, &websocket.AcceptOptions{...})`; go.mod confirms v1.8.14 |
| `internal/relay/manager.go` | `internal/relay/hub.go` | HubManager.Create instantiates Hub and starts Run goroutine | VERIFIED | manager.go lines 33-35: `hub := NewHub(...)`, `go hub.Run()` |
| `internal/relay/server.go` | `internal/relay/hub.go` | Handler subscribes before snapshot replay (anti-race pattern) | VERIFIED | server.go line 69 Subscribe, line 74 ScrollbackSnapshot — ordering enforced with comment |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SESS-03 | 02-01-PLAN.md, 02-02-PLAN.md | User can reattach to a running session after reopening the window | SATISFIED | Scrollback replay on reconnect proven by TestHub_ReconnectScrollback; marked [x] in REQUIREMENTS.md (line 30); Phase 2 mapping in traceability table (line 111) |

No orphaned requirements: REQUIREMENTS.md maps only SESS-03 to Phase 2, and both plans claim SESS-03.

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/relay/server.go` | 97-98 | MsgResize2 dispatch discards resize frames | Info | Intentional scope boundary — comment states "Phase 3 will call backend.Resize(cols, rows)"; does not block SESS-03 goal |
| `internal/relay/server.go` | 48-49 | InsecureSkipVerify on websocket.Accept | Info | Intentional scope boundary — comment states "Phase 4 will add proper CORS/origin policy"; does not block SESS-03 goal |

No blockers or warnings. Both info-level items are documented intentional deferrals to named future phases.

---

### Human Verification Required

None — all SESS-03 success criteria are proven by automated integration tests using real WebSocket connections. The test suite covers:

1. Fan-out to multiple simultaneous clients
2. Scrollback replay on reconnect plus subsequent live output
3. Input from a client reaching the PTY and output broadcasting to all clients
4. Slow-client disconnection without blocking other clients

---

### Gaps Summary

No gaps. All 7 observable truths verified. All 6 artifacts are substantive and wired. All 7 key links confirmed. SESS-03 satisfied. 29 tests pass under -race (25 unit + 4 integration).

---

_Verified: 2026-03-18T14:10:00Z_
_Verifier: Claude (gsd-verifier)_
