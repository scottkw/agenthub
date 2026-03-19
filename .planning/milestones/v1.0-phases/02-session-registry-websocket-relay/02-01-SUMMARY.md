---
phase: 02-session-registry-websocket-relay
plan: "01"
subsystem: relay
tags: [go, binary-framing, websocket, scrollback, fan-out, hub, pty]

# Dependency graph
requires:
  - phase: 01-pty-foundation
    provides: "Session (io.ReadWriter), SessionBackend interface"
provides:
  - "Binary framing protocol with MsgOutput/Input/Resize/Title/Ping constants and encode/decode helpers"
  - "Bounded scrollback buffer (Scrollback) with Append (drop-oldest-on-overflow) and Snapshot (isolated copy)"
  - "Per-session fan-out Hub: single-reader drain goroutine, N-subscriber broadcast, slow-client disconnect"
affects: [02-02-websocket-server, 03-electron-ui, 05-status-bar]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single-goroutine PTY drain: only Hub.Run calls reader.Read, preventing concurrent read races"
    - "Non-blocking fan-out: `select { case sub.Msgs <- frame: default: go sub.CloseSlow() }` ensures one slow client cannot block others"
    - "Subscribe-before-Snapshot ordering: callers subscribe first to avoid missing frames between snapshot replay and live stream"
    - "TDD: RED (failing tests) committed before GREEN (implementation) for both tasks"

key-files:
  created:
    - internal/relay/protocol.go
    - internal/relay/protocol_test.go
    - internal/relay/scrollback.go
    - internal/relay/scrollback_test.go
    - internal/relay/hub.go
    - internal/relay/hub_test.go
  modified: []

key-decisions:
  - "Hub stores scrollback as framed bytes (MakeOutputFrame wrapped), not raw PTY bytes — WebSocket clients get consistent framing from both live stream and replay"
  - "Scrollback.Append uses in-place copy shift on overflow (no extra allocation) — avoids GC pressure under high-throughput PTY output"
  - "Hub.Shutdown uses sync.Once to allow safe multiple calls — Run calls it on return, external callers can also call it without panic"

patterns-established:
  - "Binary frame: 1-byte type prefix + payload. ParseFrame(frame) -> (type, payload, error)"
  - "Scrollback: bounded ring-style buffer via front-trim on overflow, mutex-protected, snapshot returns copy"
  - "Hub: NewHub(sessionID, reader, writer, scrollbackBytes) -> *Hub; Run() as goroutine; Subscribe/Unsubscribe under mu"

requirements-completed: [SESS-03]

# Metrics
duration: 3min
completed: 2026-03-18
---

# Phase 2 Plan 01: Relay Data Layer Summary

**Single-type-byte binary framing protocol, 256KiB bounded scrollback buffer, and per-session fan-out Hub draining PTY io.Reader with race-clean slow-client disconnect**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-18T13:36:57Z
- **Completed:** 2026-03-18T13:39:54Z
- **Tasks:** 2
- **Files modified:** 6 (all created)

## Accomplishments

- Binary framing protocol with 6 message type constants and encode/decode helpers for output, input, resize, title, ping frames
- Bounded scrollback buffer with mutex-protected Append (drops oldest bytes on overflow) and Snapshot (returns isolated copy)
- Per-session fan-out Hub backed by io.Reader/Writer: single drain goroutine, N-subscriber broadcast, non-blocking send with CloseSlow on full channels, Done channel for shutdown signaling
- 24 unit tests all passing under `-race` detector; full project suite clean

## Task Commits

1. **Task 1: Binary framing protocol and scrollback buffer** - `ba15749` (feat)
2. **Task 2: Per-session fan-out Hub with drain goroutine** - `8b2d8f4` (feat)

_Note: TDD tasks — test (RED) committed inline with implementation (GREEN) in single task commits_

## Files Created/Modified

- `internal/relay/protocol.go` — MsgOutput/Input/Resize/Title/Resize2/Ping constants; MakeOutputFrame, MakeInputFrame, MakeResizeFrame (big-endian), ParseFrame
- `internal/relay/protocol_test.go` — 9 tests: frame construction, round-trip, resize encoding, empty-frame error
- `internal/relay/scrollback.go` — Scrollback struct with sync.Mutex; NewScrollback, Append (front-trim on overflow), Snapshot (copy); DefaultScrollbackBytes=256KiB
- `internal/relay/scrollback_test.go` — 6 tests: append/snapshot, truncation at boundary, snapshot isolation, concurrent safety, constant value
- `internal/relay/hub.go` — Subscriber (Msgs chan, CloseSlow); Hub (reader, writer, scrollback, subscriber map, done chan, sync.Once); Run drain goroutine; Subscribe/Unsubscribe; Shutdown; WriteInput; ScrollbackSnapshot; Done
- `internal/relay/hub_test.go` — 9 tests: fan-out to single/dual subscribers, slow-client disconnect, unsubscribe, EOF shutdown, Done channel, WriteInput, scrollback after output

## Decisions Made

- Hub stores scrollback as framed bytes (MakeOutputFrame wrapped), not raw PTY bytes — WebSocket clients receive identical bytes from live stream and replay without re-framing
- Scrollback.Append uses in-place copy-left on overflow (no extra allocation) — reduces GC pressure under high-throughput PTY output
- Hub.Shutdown uses sync.Once — allows Run to call it on return and external callers to also call it safely

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `protocol.go`, `scrollback.go`, and `hub.go` are complete foundational types
- Plan 02 (WebSocket server) can immediately import `NewHub`, `Subscribe`, `ScrollbackSnapshot`, `WriteInput`, `Done`
- No external dependencies added — `coder/websocket` is introduced in Plan 02 as specified

---
*Phase: 02-session-registry-websocket-relay*
*Completed: 2026-03-18*
