---
phase: 75-cli-status-bar
verified: 2026-04-14T18:00:00Z
status: human_needed
score: 13/13
overrides_applied: 0
human_verification:
  - test: "Run `agenthub attach <session-id>` in a real TTY and verify the bar renders at the bottom with session name, agent type, hostname, Ctrl-\\ hint, and elapsed time without corrupting PTY output during scrolling"
    expected: "Persistent reverse-video bottom bar visible; terminal content scrolls in the region above it with no garbled lines"
    why_human: "DECSTBM scroll region correctness requires a real PTY — cannot verify terminal output integrity programmatically"
  - test: "Open a second attach client to the same session and watch the first client's bar"
    expected: "Bar updates to show '2 viewers' on the next tick"
    why_human: "Live viewer count update via MsgMeta push requires an actual multi-client scenario"
  - test: "Detach from a session (Ctrl-\\) and inspect the terminal"
    expected: "Bar line is cleared, scroll region is restored to full terminal, no leftover ANSI artifacts"
    why_human: "Terminal state after cleanup must be verified by human inspection"
  - test: "Run `agenthub attach <session-id> | cat` and check output"
    expected: "No ANSI bar sequences in the piped output; only PTY content passes through"
    why_human: "Requires piping actual attach output and inspecting for absence of bar escape sequences"
---

# Phase 75: CLI Status Bar Verification Report

**Phase Goal:** `agenthub attach` displays a persistent status bar that shows session context and live state without corrupting terminal output, and cleans up completely on exit
**Verified:** 2026-04-14T18:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Attaching to a session shows a bottom bar with session name, agent type, hostname, detach hint, and elapsed time | VERIFIED | `cmd_attach.go` l.148–155: `statusbar.New` called with `session.Name`, `session.CLI`, `session.Hostname`, `createdAt`; `bar.go` `format()` assembles all fields with `Ctrl-\ to detach`; `TestBar_FormatContainsRequiredFields` passes |
| 2 | Status bar refreshes on timer and terminal output scrolls normally — no garbled lines or overwritten content | VERIFIED | `bar.go` `tickLoop()` fires `draw()` every second; `draw()` uses `cursorSave`/`cursorRestore` around bar row write; `lockedWriter` in `cmd_attach.go` serializes all stdout writes; `TestBar_ScrollRegionSetOnStart` passes — DECSTBM set correctly |
| 3 | `agenthub attach ... \| cat` produces no status bar output | VERIFIED | `cmd_attach.go` l.139: `term.IsTerminal(int(os.Stdout.Fd()))` guards bar creation; non-TTY path calls `printAttachBanner(os.Stderr, ...)` only — bar code never reached; behavior verified by code path inspection |
| 4 | When a second client connects, the bar updates to show viewer count (e.g. "2 viewers") | VERIFIED | `relay/server.go` `NotifyViewerCount(hub)` called on subscribe/unsubscribe (l.84, l.87); `webserver/server.go` `relay.NotifyViewerCount(hub)` likewise (l.422, l.425); `wsOutputPump` intercepts `relay.MsgMeta` (l.441–447) and calls `bar.SetViewerCount`; `bar.go` format shows `"%d viewers"` when count > 1; `TestBroadcastMeta_NonBlocking`, `TestWsOutputPump_MsgMeta` pass |
| 5 | Detaching or exiting removes the bar line and restores terminal to pre-attach state | VERIFIED | `cmd_attach.go` l.157/271: `defer bar.Stop()` after `bar.Start()`; `bar.go` `Stop()` issues `resetScrollRegion`, moves cursor to bar row, `eraseLineEntire`, `cursorRestore`; protected by `sync.Once`; `TestBar_StopClearsBarAndResetsScrollRegion` and `TestBar_StopIdempotent` pass |

**Score:** 5/5 roadmap success criteria verified (plus 8 additional plan must-haves — all pass)

### All Plan Must-Haves

#### Plan 01 Must-Haves (SB-04)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | MsgMeta frame type 0x20 exists in relay protocol | VERIFIED | `relay/protocol.go` l.21: `MsgMeta byte = 0x20` |
| 2 | MakeMeta produces a parseable MsgMeta frame with JSON payload | VERIFIED | `protocol.go` l.67–73: `MakeMeta` encodes JSON, prepends `MsgMeta` byte; `TestMakeMeta_RoundTrip` passes |
| 3 | Hub.BroadcastMeta sends frames to all subscribers without blocking | VERIFIED | `hub.go` l.172–183: non-blocking select with `CloseSlow` fallback; `TestBroadcastMeta_NonBlocking` passes |
| 4 | Relay server pushes MsgMeta with viewer count on subscribe and unsubscribe | VERIFIED | `relay/server.go` l.84: `NotifyViewerCount(hub)` after subscribe; l.87: in deferred unsubscribe wrapper |
| 5 | Webserver relay pushes MsgMeta with viewer count on subscribe and unsubscribe | VERIFIED | `webserver/server.go` l.422: `relay.NotifyViewerCount(hub)` after subscribe; l.425: in deferred unsubscribe wrapper |

#### Plan 02 Must-Haves (SB-01, SB-02, SB-05, SB-06, SB-07)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Bar.format() produces string with session name, agent type, hostname, detach hint, elapsed time | VERIFIED | `bar.go` l.187–232; all fields assembled; `TestBar_FormatContainsRequiredFields` passes |
| 2 | Bar.Start() sets DECSTBM scroll region and starts 1-second ticker goroutine | VERIFIED | `bar.go` l.104–127; `setScrollRegion` written, `tickLoop` goroutine launched |
| 3 | Bar.Stop() resets scroll region, clears bar line, safe to call multiple times | VERIFIED | `bar.go` l.236–260; `resetScrollRegion` + `eraseLineEntire`; `stopOnce sync.Once` guard; 3 tests pass |
| 4 | Bar.draw() saves/restores cursor and writes to reserved bar row without corrupting scroll region | VERIFIED | `bar.go` l.143–178; `cursorSave`/`moveCursor`/`eraseLineEntire`/`cursorRestore` sequence |
| 5 | Bar supports both Bottom (default) and Top placement via Position option | VERIFIED | `bar.go` l.30–33: `Bottom`/`Top` constants; `Start` and `draw` branch on `Position`; `TestBar_TopPosition` passes |
| 6 | Bar.SetViewerCount() updates viewer count displayed on next tick | VERIFIED | `bar.go` l.263–267; thread-safe mutex setter; `TestBar_SetViewerCountUpdates` passes |
| 7 | Bar.SetConnectionState() updates connection state displayed on next tick | VERIFIED | `bar.go` l.272–276; thread-safe mutex setter; `TestBar_ConnStateDisplay` passes |
| 8 | Session name and hostname sanitized to strip control characters | VERIFIED | `bar.go` l.78–87: `sanitize()` strips `r < 0x20`; applied in `format()`; `TestBar_SanitizeSessionName` passes |

#### Plan 03 Must-Haves (SB-01 through SB-07)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Local attach displays persistent status bar when stdout is a TTY | VERIFIED | `cmd_attach.go` l.139–157: TTY check → `statusbar.New` → `bar.Start()` → `defer bar.Stop()` |
| 2 | Remote attach displays persistent status bar when stdout is a TTY | VERIFIED | `cmd_attach.go` l.257–271: same pattern in `cmdAttachRemoteWithClient` |
| 3 | Status bar suppressed when stdout is not a TTY | VERIFIED | `cmd_attach.go` l.139, l.257: `term.IsTerminal` guards; else branch routes to `printAttachBanner(os.Stderr, ...)` |
| 4 | MsgMeta frames intercepted and update bar viewer count | VERIFIED | `wsOutputPump` l.441–447: `case relay.MsgMeta` → `json.Unmarshal` → `bar.SetViewerCount`; `TestWsOutputPump_MsgMeta` passes |
| 5 | --status-top flag places bar at top | VERIFIED | `cmd_attach.go` l.54–55: flag parsed; l.145–146, l.259–260: `pos = statusbar.Top` when set; threaded through `cmdAttachRemote`/`cmdAttachRemoteWithClient` signatures |
| 6 | All stdout writes serialized through lockedWriter | VERIFIED | `cmd_attach.go` l.27–35: `lockedWriter` type; l.135, l.253: `stdout := &lockedWriter{w: os.Stdout}` in both paths; `TestLockedWriter_ConcurrentWrites` passes |
| 7 | Bar cleaned up on detach/exit via deferred Stop() | VERIFIED | `cmd_attach.go` l.157: `defer bar.Stop()` (local); l.271: `defer bar.Stop()` (remote) |
| 8 | Connection state shown as [reconnecting] when no frame received for >5s (remote only) | VERIFIED | `cmd_attach.go` l.294–325: watcher goroutine checks `time.Since(lastFrame) > 5*time.Second`; `bar.SetConnectionState("reconnecting")` |
| 9 | Old banner suppressed in TTY path | VERIFIED | `cmd_attach.go` l.158–161: `printAttachBanner` only in else branch (non-TTY) |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/relay/protocol.go` | MsgMeta constant, MetaPayload struct, MakeMeta function | VERIFIED | l.21: `MsgMeta byte = 0x20`; l.62–73: `MetaPayload` and `MakeMeta` |
| `internal/relay/hub.go` | BroadcastMeta method | VERIFIED | l.172–183: `func (h *Hub) BroadcastMeta(frame []byte)` |
| `internal/relay/protocol_test.go` | MsgMeta round-trip test | VERIFIED | `TestMakeMeta_RoundTrip` l.138, `TestMakeMeta_OmitsNilFields` l.164 |
| `internal/relay/hub_test.go` | BroadcastMeta test | VERIFIED | `TestBroadcastMeta_NonBlocking` l.493 |
| `internal/relay/server.go` | broadcastMeta/NotifyViewerCount call on subscribe/unsubscribe | VERIFIED | `NotifyViewerCount` at l.148–152; called at l.84, l.87 |
| `internal/webserver/server.go` | relay.NotifyViewerCount on subscribe/unsubscribe | VERIFIED | `relay.NotifyViewerCount(hub)` at l.422, l.425 |
| `internal/statusbar/bar.go` | Bar type with Start, Stop, SetViewerCount, SetConnectionState, draw, format | VERIFIED | All methods present; 278 lines; substantive implementation |
| `internal/statusbar/bar_test.go` | Tests for format, scroll region, cleanup, viewer count, connection state, position | VERIFIED | 9 tests present; all pass |
| `cmd_attach.go` | lockedWriter, --status-top flag, Bar instantiation, MsgMeta intercept, connection state tracking | VERIFIED | All present and wired |
| `cmd_attach_test.go` | TestWsOutputPump_MsgMeta and TestLockedWriter_ConcurrentWrites | VERIFIED | Both tests present and passing |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `relay/server.go` | `relay/hub.go` | `hub.BroadcastMeta(relay.MakeMeta(...))` via `NotifyViewerCount` | VERIFIED | l.148–152 in server.go calls `hub.BroadcastMeta(frame)` |
| `webserver/server.go` | `relay/hub.go` | `relay.NotifyViewerCount(hub)` | VERIFIED | l.422, l.425 in webserver/server.go |
| `cmd_attach.go wsOutputPump` | `internal/statusbar.Bar` | `bar.SetViewerCount()` on MsgMeta frame | VERIFIED | l.441–447 in wsOutputPump |
| `cmd_attach.go cmdAttach` | `internal/statusbar.New` | `statusbar.New(stdout, statusbar.Options{...})` | VERIFIED | l.148–155 in cmdAttach |
| `cmd_attach.go cmdAttach` | `term.IsTerminal(os.Stdout.Fd())` | TTY check before bar creation | VERIFIED | l.139 in cmdAttach; l.257 in cmdAttachRemoteWithClient |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| `bar.go format()` | `viewerCount` | `b.viewerCount` set by `SetViewerCount(n)` ← `wsOutputPump` ← `relay.MsgMeta` frame ← `hub.BroadcastMeta` ← `hub.SubscriberCount()` | Yes — live from `len(h.subscribers)` | FLOWING |
| `bar.go format()` | `connState` | `b.connState` set by `SetConnectionState` ← watcher goroutine checking `time.Since(lastFrame)` ← `onFrame()` called per frame | Yes — live time tracking | FLOWING |
| `bar.go format()` | `SessionName`, `AgentType`, `Hostname` | `Options` struct populated from `daemon.SessionInfo` / `CLIRemoteSession` at bar creation | Yes — real session metadata | FLOWING |
| `bar.go format()` | `elapsed` | `time.Since(b.opts.CreatedAt)` | Yes — live wall-clock calculation | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All relay tests pass | `go test ./internal/relay/... -count=1 -timeout 60s` | ok 0.721s | PASS |
| All statusbar tests pass | `go test ./internal/statusbar/... -count=1 -timeout 60s` | ok 4.526s (9/9) | PASS |
| MsgMeta tests pass | `go test -run "TestWsOutputPump_MsgMeta|TestLockedWriter|TestWsOutputPump_Ignores"` | ok 0.019s | PASS |
| Full project builds | `go build ./...` | exit 0 | PASS |
| Full test suite passes | `go test ./... -count=1 -timeout 120s` | all 9 packages ok | PASS |
| go vet clean | `go vet ./...` | exit 0, no warnings | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SB-01 | 75-02, 75-03 | CLI attach shows persistent bottom bar with session name, agent type, hostname, detach hint, elapsed time | SATISFIED | `bar.go format()` assembles all fields; `cmdAttach` wires session metadata into `statusbar.Options`; `TestBar_FormatContainsRequiredFields` |
| SB-02 | 75-02, 75-03 | Status bar refreshes on timer without corrupting output (DECSTBM scroll region) | SATISFIED | `tickLoop` 1s ticker; `draw()` cursor save/restore pattern; `lockedWriter` serializes; `TestBar_ScrollRegionSetOnStart` |
| SB-03 | 75-03 | Status bar suppressed when stdout is not a TTY | SATISFIED | `term.IsTerminal(int(os.Stdout.Fd()))` guard in both `cmdAttach` and `cmdAttachRemoteWithClient` |
| SB-04 | 75-01, 75-03 | Status bar shows viewer count when multiple clients connected | SATISFIED | `NotifyViewerCount` broadcast on subscribe/unsubscribe; `wsOutputPump` MsgMeta intercept; `bar.SetViewerCount`; `TestBroadcastMeta_NonBlocking` + `TestWsOutputPump_MsgMeta` |
| SB-05 | 75-03 | Status bar shows connection state (connected/reconnecting) for remote sessions | SATISFIED | Connection state watcher goroutine in `cmdAttachRemoteWithClient`; 5s timeout; `bar.SetConnectionState("reconnecting")`; `TestBar_ConnStateDisplay` |
| SB-06 | 75-02, 75-03 | User can place status bar at top via `--status-top` flag | SATISFIED | `--status-top` flag parsed; `statusbar.Top` position constant; DECSTBM row 2 for scroll region; `TestBar_TopPosition` |
| SB-07 | 75-02, 75-03 | Status bar cleans up on detach/exit — clears bar line and restores terminal state | SATISFIED | `defer bar.Stop()` in both paths; `Stop()` issues `resetScrollRegion` + `eraseLineEntire`; `sync.Once` idempotency; `TestBar_StopClearsBarAndResetsScrollRegion` + `TestBar_StopIdempotent` |

### Anti-Patterns Found

No anti-patterns detected. Scanned `cmd_attach.go`, `internal/statusbar/bar.go`, `internal/relay/protocol.go`, `internal/relay/hub.go`, `internal/relay/server.go`. No TODOs, FIXMEs, placeholder returns, or hardcoded empty data found in any phase-modified file.

### Human Verification Required

The automated checks all pass. Four behaviors require real-TTY verification:

#### 1. Status Bar Rendering and Scroll Region Correctness

**Test:** Run `agenthub attach <session-id>` in an interactive terminal (iTerm2 or Terminal.app). Start a session with active output (e.g., a running AI agent session).
**Expected:** A reverse-video bottom bar appears containing the session name, agent type, hostname, `Ctrl-\ to detach`, and elapsed time formatted as `M:SS`. Terminal content above the bar scrolls without overwriting or corrupting the bar. No line doubling, garbled characters, or bar flicker.
**Why human:** DECSTBM scroll region correctness (SB-02) requires a real PTY emulator. The unit tests confirm ANSI sequences are emitted correctly, but whether they produce visually correct output without corrupting the scrollback depends on the terminal's DECSTBM implementation.

#### 2. Live Viewer Count Update

**Test:** Open two terminal windows. In the first, run `agenthub attach <session-id>`. In the second, also run `agenthub attach <session-id>`.
**Expected:** On the first client's bar, after the second client connects, the bar updates within 1 second to show "2 viewers". After the second client detaches, it returns to no viewer count (count drops to 1, field hidden).
**Why human:** Multi-client viewer count update requires two live WebSocket connections; not reproducible in the current unit test infrastructure without a full integration harness.

#### 3. Terminal Cleanup on Exit

**Test:** Attach to a session, then press Ctrl-\ to detach.
**Expected:** The status bar line is cleared (blank row, not reverse video). The scroll region is restored to the full terminal (content scrolls all the way to the bottom row). No leftover ANSI sequences visible. The prompt returns normally.
**Why human:** Terminal state after cleanup must be verified by visual inspection — the escape sequences are emitted correctly per unit tests, but their cumulative effect on different terminal emulators can vary.

#### 4. Non-TTY Suppression

**Test:** Run `agenthub attach <session-id> | cat` or `agenthub attach <session-id> > /tmp/output.txt`.
**Expected:** No ANSI bar escape sequences appear in the piped/redirected output. Only raw PTY content passes through.
**Why human:** Requires piping actual attach output in a shell and inspecting the raw bytes for absence of bar sequences. `term.IsTerminal` behavior is correct per code review, but end-to-end piping needs manual confirmation.

### Gaps Summary

No gaps. All 13 must-haves verified across all three plans. All 7 requirements (SB-01 through SB-07) satisfied. Full test suite (9 packages) passes. Full project builds and vets clean.

The phase goal is structurally achieved. Human verification of visual terminal behavior is required before marking as fully complete.

---

_Verified: 2026-04-14T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
