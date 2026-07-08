---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
plan: 06
subsystem: web-share
tags: [go, websocket, react, accessibility, colorblind-safe, tdd]

requires:
  - phase: 175
    provides: "175-02's skip-guarded RED scaffold internal/webserver/session_ended_test.go#TestSessionEnd_HubDone_CarriesCloseReason for BUG-02; 175-04's emulator-derived WS replay-site changes in server.go/relay/server.go (built on top of, not reverted)"
provides:
  - "Both WS write pumps (internal/webserver/server.go, internal/relay/server.go) call conn.Close(StatusNormalClosure, \"session ended\") on hub.Done() instead of a bare return — every connected viewer's socket now closes with a fixed, non-leaking code+reason"
  - "RelayClient.onClose widened to (code?, reason?) => void, capturing the raw WebSocket CloseEvent"
  - "SessionEndedBanner component — colorblind-safe (text + role=status + aria-live=polite), fixed generic copy, never renders the raw CloseEvent.reason"
  - "TerminalPanel renders SessionEndedBanner on the guest path only when the RelayClient closes; no auto-reconnect wired"
affects: [175-07]

tech-stack:
  added: []
  patterns:
    - "Fixed generic WS close reason (never raw Go error text, never user-set session name) mirrors the pre-existing IN-01 inject-NAK convention"
    - "Guest-only disconnect banner sourced from a widened onClose(code, reason) callback, reason accepted only for logging/branching — never rendered (dangerouslySetInnerHTML-avoidance discipline reused from HubModal.tsx:67)"

key-files:
  created:
    - frontend/src/components/SessionEndedBanner.tsx
    - frontend/src/components/__tests__/SessionEndedBanner.test.tsx
  modified:
    - internal/webserver/server.go
    - internal/relay/server.go
    - internal/webserver/session_ended_test.go
    - frontend/src/lib/relayClient.ts
    - frontend/src/lib/relayClient.test.ts
    - frontend/src/components/TerminalPanel.tsx
    - frontend/src/style.css

key-decisions:
  - "Both write pumps' hub.Done() branch calls conn.Close(websocket.StatusNormalClosure, \"session ended\") with an identical fixed literal reason at both sites — no session name, no error interpolation"
  - "SessionEndedBanner accepts an optional reason prop used ONLY for console.debug logging/branching; the visible copy is always the fixed string, verified by a dedicated hostile-string test (T-175-06-02)"
  - "Banner renders only on the guest path (isGuest = remote || !!wsURL); the host (session owner) never sees it, since it is the owner's own session ending, not a remote disconnect"
  - "No auto-reconnect wired anywhere — a \"session ended\" close means the session is genuinely gone (RESEARCH anti-pattern, phase-locked decision)"
  - "session_ended_test.go's single conn.Read() call was a latent test bug (Rule 1): the pre-buffered initial MsgMeta/MsgPresence frames could arrive before the close on the wire, so the test now drains ordinary data frames in a loop until it observes the CloseError"

patterns-established:
  - "SessionEndedBanner CSS colocated with .webgl-recovery-banner in style.css, reusing the existing destructive-red (#f7768e) left-accent decoration — meaning always carried by text + role=status, per [[user_colorblind]]"

requirements-completed: [BUG-02]

coverage:
  - id: D1
    description: "Both WS write pumps send a fixed generic close reason (StatusNormalClosure, \"session ended\") on hub.Done() instead of a bare return"
    requirement: BUG-02
    verification:
      - kind: unit
        ref: "internal/webserver/session_ended_test.go#TestSessionEnd_HubDone_CarriesCloseReason (unskipped, GREEN)"
        status: pass
      - kind: unit
        ref: "go build ./... && go vet ./... (clean); go test ./internal/webserver/... ./internal/relay/... -count=1 (GREEN, no regressions)"
        status: pass
    human_judgment: false
  - id: D2
    description: "RelayClient.onClose widened to (code?, reason?) => void, capturing evt.code/evt.reason from the WebSocket onclose handler"
    requirement: BUG-02
    verification:
      - kind: unit
        ref: "frontend/src/lib/relayClient.test.ts — 'RelayClient onClose callback dispatch (Phase 175-06 / BUG-02)' describe block (3 new tests, GREEN)"
        status: pass
    human_judgment: false
  - id: D3
    description: "SessionEndedBanner renders a fixed, colorblind-safe (text + role=status + aria-live=polite) disconnect notice and never renders the raw CloseEvent.reason, even when it is a hostile string"
    requirement: BUG-02
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SessionEndedBanner.test.tsx (6 tests incl. hostile-reason injection guard, GREEN)"
        status: pass
    human_judgment: false
  - id: D4
    description: "TerminalPanel shows SessionEndedBanner on the guest path only, on RelayClient close; no auto-reconnect"
    requirement: BUG-02
    verification:
      - kind: unit
        ref: "cd frontend && pnpm exec tsc --noEmit (clean); pnpm build (succeeds); pnpm exec vitest run (2411/2411 pass, full suite)"
        status: pass
    human_judgment: true
    rationale: "The guest-path gating and visual placement of the banner over a live terminal is best confirmed by a human seeing it render during an owner-ends-session UAT; unit/tsc/build coverage proves it compiles and the isolated component behaves correctly, but the live end-to-end \"owner stops session -> guest sees banner\" flow is deferred to 175-07's new M-NN manual UAT item per this plan's own <verification> section."

duration: 11min
completed: 2026-07-08
status: complete
---

# Phase 175 Plan 06: WS Session-End Close Reason + Colorblind-Safe Disconnect Banner Summary

**Both WS write pumps now emit a fixed `conn.Close(StatusNormalClosure, "session ended")` on `hub.Done()` (closing BUG-02/#125's silent-dead-terminal gap), and the guest client renders a colorblind-safe `SessionEndedBanner` (text + `role=status` + `aria-live=polite`, never the raw `CloseEvent.reason`) instead of a frozen terminal.**

## Performance

- **Duration:** ~11 min
- **Started:** 2026-07-08T19:27Z (approx)
- **Completed:** 2026-07-08T19:38Z (approx)
- **Tasks:** 2
- **Files modified:** 7 (+ 2 new)

## Accomplishments
- Both write pumps (`internal/webserver/server.go:1607`, `internal/relay/server.go:443`) replace the bare `return` in `case <-hub.Done():` with `conn.Close(websocket.StatusNormalClosure, "session ended")`, mirroring the existing `conn.Close(websocket.StatusPolicyViolation, "too slow")` idiom and the IN-01 no-leak convention — a fixed literal, never raw Go error text or a user-set session name.
- Unskipped 175-02's RED scaffold `internal/webserver/session_ended_test.go#TestSessionEnd_HubDone_CarriesCloseReason` and turned it GREEN. Along the way, fixed a latent bug in the test itself (Rule 1): a single `conn.Read()` call could observe one of the pre-buffered initial `MsgMeta`/`MsgPresence` frames instead of the close, since those frames are written to the wire before the test's only read. The test now drains ordinary data frames in a loop until it observes the `*websocket.CloseError`.
- `RelayClient.onClose` widened from `() => void` to `(code?: number, reason?: string) => void`; the `ws.onclose` handler now passes through `evt.code`/`evt.reason`. Only caller (`TerminalPanel.tsx`) updated; type widening is backward-compatible with any other `() => void`-shaped consumer.
- New `SessionEndedBanner` component mirrors `WebGLRecoveryBanner`'s accessible-banner pattern (`role="status"`, `aria-live="polite"`, dismiss button) with a single fixed generic message: "Session ended — the owner stopped this session." The `reason` prop (raw `CloseEvent.reason`) is accepted only for `console.debug` logging/branching and is **never** rendered into the DOM — verified by a dedicated test that passes a hostile `<script>`/`onerror` string and asserts no injection and no raw-reason leakage.
- `TerminalPanel` wires the widened `onClose(code, reason)` to `setSessionEnded({ ended: true, reason })`, gated to the guest path (`isGuest = remote || !!wsURL`) only — the session owner (host) never sees the banner about their own session ending. No auto-reconnect logic was added anywhere (explicit phase-locked decision / RESEARCH anti-pattern).
- New `.session-ended-banner` CSS colocated with `.webgl-recovery-banner` in `style.css`, positioned as an absolute overlay inside `.terminal-session-container` (same anchor pattern as `.find-bar`), reusing the existing destructive-red `#f7768e` left-accent as decoration only — the meaning is always carried by the text + `role="status"` contract, per the colorblind constraint.

## Task Commits

Each task was committed atomically:

1. **Task 1: Send a generic close code + reason on hub.Done() at BOTH write pumps** - `22c40b88` (fix)
2. **Task 2: Capture the CloseEvent in relayClient and render a colorblind-safe disconnect banner** - `4d47fffa` (feat)

## Files Created/Modified
- `internal/webserver/server.go` - `hub.Done()` write-pump branch now calls `conn.Close(StatusNormalClosure, "session ended")`
- `internal/relay/server.go` - identical fix at the loopback relay WS write pump
- `internal/webserver/session_ended_test.go` - unskipped `TestSessionEnd_HubDone_CarriesCloseReason`; fixed a latent single-Read test bug by draining pre-close data frames
- `frontend/src/lib/relayClient.ts` - `onClose` callback type widened to `(code?, reason?) => void`; `ws.onclose` passes through `evt.code`/`evt.reason`
- `frontend/src/lib/relayClient.test.ts` - 3 new tests for the widened `onClose` dispatch (fires once with code/reason, omittable, ping-clear still fires)
- `frontend/src/components/SessionEndedBanner.tsx` (new) - colorblind-safe fixed-copy disconnect banner
- `frontend/src/components/__tests__/SessionEndedBanner.test.tsx` (new) - 6 tests incl. accessibility contract + hostile-reason injection guard
- `frontend/src/components/TerminalPanel.tsx` - `sessionEnded` state; guest-path-only `onClose` wiring; `SessionEndedBanner` rendered in the JSX return
- `frontend/src/style.css` - `.session-ended-banner` + `__message`/`__dismiss` rules colocated with `.webgl-recovery-banner`

## Decisions Made
- Fixed literal `"session ended"` reason at both server sites — no session name, no error interpolation (mirrors IN-01/T-175-02-02).
- `SessionEndedBanner`'s `reason` prop is logging-only, never rendered; enforced by a dedicated hostile-string test rather than relying on code review alone.
- Banner gated strictly to the guest path via the pre-existing `isGuest = remote || !!wsURL` derivation already used elsewhere in `TerminalPanel` (Phase 157 VIEW-04/05 precedent) — no new guest-detection logic invented.
- No auto-reconnect: `sessionEnded` state is dismiss-only; closing the banner does not reopen the `RelayClient`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `session_ended_test.go`'s single `conn.Read()` could observe a pre-buffered data frame instead of the close**
- **Found during:** Task 1, first `go test` run against the unskipped scaffold — `conn.Read: expected close error after hub.Shutdown(), got nil error`.
- **Issue:** The 175-02 RED scaffold's `50ms` sleep after dial only allows the server to *send* the initial `MsgMeta`/`MsgPresence` frames onto the wire; it does not drain them client-side. A single subsequent `conn.Read()` after `hub.Shutdown()` could therefore return one of those already-buffered data frames (`err == nil`) instead of the close.
- **Fix:** Replaced the single `conn.Read()` assertion with a loop that discards ordinary data frames (`err == nil`) and only asserts once `err != nil`, matching against `*websocket.CloseError`. Bounded by the pre-existing 3s context timeout, so a genuinely missing close still fails the test.
- **Files modified:** `internal/webserver/session_ended_test.go`
- **Verification:** `go test ./internal/webserver/... -run TestSessionEnd -count=1 -v` GREEN; full `go test ./internal/webserver/... ./internal/relay/... -count=1` GREEN (no regressions).
- **Committed in:** `22c40b88` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug, self-caught during the task's own mandatory verification run)
**Impact on plan:** Necessary correctness fix confined to the test file this task already owned; no scope creep, no production code beyond what the plan specified.

## Issues Encountered
None beyond the deviation above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- BUG-02 is code-complete and unit-verified at both the Go WS layer and the React component/hook layer; `tsc --noEmit` clean, `pnpm build` succeeds, full frontend suite (2411/2411) and full Go webserver+relay suites GREEN.
- 175-07 owns the deferred live owner-ends-session -> guest-sees-banner UAT (new M-NN item, per this plan's `<verification>` section), alongside 175-04's deferred live two-client alt-screen reconnect UAT and the pre-existing TESTING.md Suite Manifest gap logged in `deferred-items.md`.

---
*Phase: 175-web-share-remote-viewer-windowing-bug-fixes*
*Completed: 2026-07-08*

## Self-Check: PASSED

All claimed files confirmed present on disk. Both task commit hashes
(`22c40b88`, `4d47fffa`) confirmed present in `git log --oneline --all`.
