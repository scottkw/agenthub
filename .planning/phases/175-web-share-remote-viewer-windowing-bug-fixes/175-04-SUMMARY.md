---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 04
subsystem: relay
tags: [go, vt-emulator, websocket, charmbracelet-x-vt, alt-screen, reconnect]

requires:
  - phase: 175
    provides: "175-02's skip-guarded RED scaffold internal/relay/scrollback_altscreen_test.go#TestScrollbackAltScreenReplay for BUG-04"
provides:
  - "Hub.EnsureLiveEmulator() — lazily constructs a continuously-fed per-hub VT emulator (charmbracelet/x/vt), bootstrapped once from ScrollbackSnapshot, memory-bounded via SetScrollbackSize(50)"
  - "Hub.RenderSnapshot() — emulator-derived reconnect preamble (ESC[?1049h prefix when IsAltScreen(), current-screen content, MsgOutput-framed), replacing the raw ScrollbackSnapshot() write at both WS replay sites"
  - "Re-verification that BUG-04 Problem 1 (MDI-vs-tab) is already resolved by Phase 168-03 — no code change needed"
affects: [175-07]

tech-stack:
  added: []
  patterns:
    - "Continuously-fed live VT emulator per Hub (separate emuMu lock, never hub.mu) so reconnect state survives raw scrollback ring truncation — never re-bootstrap from a possibly-wrapped ring"
    - "Query-strip regex duplicated (not shared) between internal/daemon/engine.go and internal/relay/hub.go to avoid relay importing daemon (import cycle)"

key-files:
  created: []
  modified:
    - internal/relay/hub.go
    - internal/relay/scrollback_altscreen_test.go
    - internal/webserver/server.go
    - internal/relay/server.go

key-decisions:
  - "Live emulator bootstrap + continuous feed happens as soon as EnsureLiveEmulator is first called (in practice: the very first WS connection to the hub, typically the desktop owner's own loopback TerminalPanel/CLI-attach connection in relay/server.go) — NOT re-derived from scrollback on every subsequent connect, so later ring wraps cannot re-lose the alt-screen mode marker (RESEARCH Pitfall 3)"
  - "RenderSnapshot() trims Render()'s full-height trailing blank-row padding and returns nil when content is empty and not alt-screen — restores byte-identical behavior for ordinary (non-wrapped) sessions and preserves the pre-existing \"no frame before real PTY output\" contract several webserver/relay tests already depended on"
  - "Problem 1 (MDI-vs-tab) required no code change — confirmed via source-level grep that SessionCard.tsx's remote-card 'Open in tab' affordance (168-03/FIX-03) already exists"

requirements-completed: [BUG-04]

coverage:
  - id: D1
    description: "BUG-04 Problem 1 (MDI-vs-tab complaint) re-verified as already resolved by Phase 168-03 — remote session cards expose an in-app 'Open in tab' affordance; no code change"
    requirement: BUG-04
    verification:
      - kind: unit
        ref: "grep -n \"Open in tab\" frontend/src/components/Hub/SessionCard.tsx (present, wired to onOpenInBrowser)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Lazy live per-hub VT emulator (Hub.EnsureLiveEmulator) + RenderSnapshot() reconnect preamble reconstructs alt-screen mode + current screen content after the raw 256 KiB scrollback ring wraps past ESC[?1049h"
    requirement: BUG-04
    verification:
      - kind: unit
        ref: "internal/relay/scrollback_altscreen_test.go#TestScrollbackAltScreenReplay (unskipped, GREEN)"
        status: pass
      - kind: unit
        ref: "go test -race ./internal/relay/... -count=1 (full package, GREEN)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both WS replay sites (webserver/server.go, relay/server.go) emit the emulator-derived preamble on connect, preserving VIEW-03 ordering (resize -> self frame -> preamble), with no change to read-pump dispatch or ReadOnly gating"
    requirement: BUG-04
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... (clean); go test ./internal/webserver/... ./internal/relay/... -count=1 (GREEN, no regressions in pre-existing scrollback/ordering tests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live two-client alt-screen reconnect end-to-end (real TUI, real ring wrap, real late guest join)"
    requirement: BUG-04
    verification: []
    human_judgment: true
    rationale: "Deferred to 175-07's new M-NN manual UAT item per this plan's own <verification> section — needs a real alt-screen TUI running long enough to wrap the ring plus a late-joining guest; not automatable in this plan's unit-test harness."

duration: 25min
completed: 2026-07-08
status: complete
---

# Phase 175 Plan 04: Live Per-Hub VT Emulator + Reconnect Preamble Summary

**Lazy, continuously-fed VT emulator per Hub (charmbracelet/x/vt) whose `RenderSnapshot()` reconstructs alt-screen mode + current screen content for reconnecting/late-joining viewers, replacing the raw 256 KiB scrollback replay at both WS handler sites — closes BUG-04's residual blank-window bug (#119, Problem 2).**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-07-08T18:50Z (approx)
- **Completed:** 2026-07-08T19:15Z (approx)
- **Tasks:** 3
- **Files modified:** 4 (+ 1 new deferred-items.md doc)

## Accomplishments
- Re-verified BUG-04 Problem 1 (the "MDI window" complaint) is already resolved by Phase 168-03: `SessionCard.tsx`'s remote-card "Open in tab" affordance is wired to `handleOpenRemoteSession` (source-level grep confirmation, no code change).
- Added `Hub.liveEmu`/`Hub.emuMu` + `EnsureLiveEmulator()`/`RenderSnapshot()`/`feedLiveEmulator()` to `internal/relay/hub.go`: the emulator is constructed lazily on first call, bootstrapped once from the current `ScrollbackSnapshot()`, memory-bounded via `SetScrollbackSize(50)`, and continuously fed by `Run()`'s drain loop thereafter (own lock, never `hub.mu`, so a slow/stuck emulator write can never stall the PTY-drain/broadcast loop).
- Unskipped `internal/relay/scrollback_altscreen_test.go#TestScrollbackAltScreenReplay` (175-02's RED scaffold) and turned it GREEN: after the ring wraps past `ESC[?1049h`, `RenderSnapshot()` still reconstructs the alt-screen mode marker plus the current-screen content, because the emulator was bootstrapped and fed *before* the wrap, not re-derived from the (by-then-truncated) raw ring.
- Wired `hub.EnsureLiveEmulator()` + `hub.RenderSnapshot()` into both WS replay sites (`internal/webserver/server.go` and `internal/relay/server.go`), replacing the raw `hub.ScrollbackSnapshot()` write while preserving the exact VIEW-03 ordering (authoritative resize -> self frame -> reconnect preamble) and leaving read-pump dispatch / `sub.ReadOnly` gating untouched.

## Task Commits

Each task was committed atomically:

1. **Task 1: Re-verify BUG-04 Problem 1 (MDI-vs-tab) is already resolved by 168-03** - no code change, no commit (source-level verification only; recorded here).
2. **Task 2: Add a lazy live per-hub VT emulator + RenderSnapshot reconnect preamble** - `f65a0f79` (feat)
3. **Task 3: Emit the emulator-derived preamble on connect at BOTH WS handler sites** - `7b41a39d` (feat, includes a Rule-1 fix to `RenderSnapshot()` discovered while validating this task — see Deviations)

## Files Created/Modified
- `internal/relay/hub.go` - `liveEmu`/`emuMu` fields, `altScreenEnterSeq`/`liveEmulatorScrollbackLines`/`liveEmuQueryStripPattern` constants, `stripMsgOutputBytes`, `EnsureLiveEmulator`, `RenderSnapshot`, `feedLiveEmulator`; `Run()` now calls `feedLiveEmulator` per PTY read
- `internal/relay/scrollback_altscreen_test.go` - unskipped, retargeted at `EnsureLiveEmulator()`/`RenderSnapshot()`, GREEN
- `internal/webserver/server.go` - replaced raw scrollback write with emulator preamble at the browser WS replay site
- `internal/relay/server.go` - replaced raw scrollback write with emulator preamble at the loopback relay WS replay site (desktop owner's TerminalPanel / CLI attach)
- `.planning/phases/175-web-share-remote-viewer-windowing-bug-fixes/deferred-items.md` (new) - logs a pre-existing TESTING.md Suite Manifest gap from 175-02, out of this plan's scope

## Decisions Made
- The live emulator is bootstrapped and starts receiving continuous feed as soon as `EnsureLiveEmulator()` is first called for a hub — in production this is effectively at session-open time, since `relay/server.go`'s loopback handler (the desktop owner's own TerminalPanel / `agenthub attach`) is almost always the very first WS connection any hub ever receives. This makes the "lazy" construction the plan calls for functionally equivalent to "construct at session start" for real sessions, while still being genuinely free (zero emulator, zero memory) for any session that never gets a single WS viewer (e.g. a headless daemon-only session).
- `RenderSnapshot()` trims `Render()`'s full-height (default 50-row) trailing blank-line padding via `strings.TrimRight(..., "\n")` and returns `nil` when the trimmed content is empty and the emulator is not in alt-screen mode. Without this, every connection — even to a brand-new session with zero PTY output — would emit a spurious ~23-byte all-blank-lines `MsgOutput` preamble, which several pre-existing `internal/relay` tests treat as "the" first data frame (breaking `TestServer_ReadOnlyClientInputDiscarded`, `TestHub_TwoClientsFanOut`, `TestHub_InputFanOut`, `TestWebServerWSS`, `TestServer_ReadOnlyClientReceivesOutput`). This restores byte-identical behavior for ordinary sessions while still emitting the alt-screen mode marker whenever it's actually needed.
- `liveEmuQueryStripPattern` is a verbatim duplicate of `internal/daemon/engine.go`'s `queryStripPattern`, not a shared helper — `internal/relay` cannot import `internal/daemon` (daemon already imports relay; the reverse would be an import cycle), matching the plan's explicit instruction to mirror rather than share.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] RenderSnapshot's unconditional full-screen padding broke 5 pre-existing relay/webserver tests**
- **Found during:** Task 3 (wiring RenderSnapshot into both WS handler sites) — caught by the mandatory full `go test ./internal/webserver/... ./internal/relay/...` run before committing.
- **Issue:** The initial `RenderSnapshot()` implementation returned `MakeOutputFrame([]byte(content))` unconditionally, where `content` was the emulator's raw `Render()` output (default 50 rows, each blank row padded with a trailing `\n`). For a genuinely empty/fresh session this produced a non-nil ~23-byte frame instead of the prior behavior (no frame sent until real PTY output arrived), and for a single-line session it appended ~48 extra blank lines the tests' exact-payload assertions did not expect.
- **Fix:** `RenderSnapshot()` now (a) trims `Render()`'s trailing blank-line padding via `strings.TrimRight(content, "\n")`, matching `GetSessionStyledTailLines`'s existing "trim trailing blank rows" convention, and (b) returns `nil` when the trimmed content is empty AND the emulator is not in alt-screen mode, restoring the pre-existing `len(snapshot) > 0` guard's effective behavior for ordinary sessions.
- **Files modified:** `internal/relay/hub.go`
- **Verification:** `go test ./internal/webserver/... ./internal/relay/... -count=1` and `go test -race ./internal/relay/... ./internal/webserver/... -count=1` both GREEN; full-repo `go test ./... -count=1` GREEN.
- **Committed in:** `7b41a39d` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 bug, self-caught during the task's own verification run)
**Impact on plan:** Necessary correctness fix for the new code introduced in this plan; no scope creep, no pre-existing code touched beyond the one method this plan added.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- BUG-04 Problem 2 (the blank/garbled reconnect window) is code-complete and unit-verified (`TestScrollbackAltScreenReplay` GREEN, full relay+webserver suites GREEN under `-race`).
- 175-07 owns the deferred live two-client alt-screen reconnect UAT (new M-NN item, per this plan's `<verification>` section) — needs a real full-screen TUI running long enough to wrap the ring, plus a late-joining guest.
- 175-07 (or 175-06, which also unskips a RED scaffold) is also the natural place to reconcile the pre-existing TESTING.md Suite Manifest gap logged in `deferred-items.md` (175-02 created 3 new Go test files without registering them).

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*

## Self-Check: PASSED

All claimed files confirmed present on disk (`internal/relay/hub.go`,
`internal/relay/scrollback_altscreen_test.go`, `internal/webserver/server.go`,
`internal/relay/server.go`, this SUMMARY.md, `deferred-items.md`). Both task
commit hashes (`f65a0f79`, `7b41a39d`) confirmed present in `git log --oneline --all`.
