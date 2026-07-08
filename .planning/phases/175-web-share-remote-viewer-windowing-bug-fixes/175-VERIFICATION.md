---
phase: 175-web-share-remote-viewer-windowing-bug-fixes
verified: 2026-07-08T22:55:55Z
status: human_needed
score: 6/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "M-48 (BUG-01, #128) — on a real narrow-phone browser viewport (or a resized desktop browser window simulating one), open a shared web session and confirm the 80-col terminal grid stops shrinking at the readability floor and switches to horizontal scroll instead of continuing to downscale."
    expected: "Text stays legible at the ~0.7 scale floor; below the floor, the container scrolls horizontally (touch pan-x works) instead of the glyphs continuing to shrink."
    why_human: "jsdom reports clientWidth/clientHeight as 0, so the actual pixel-legibility outcome on a real viewport cannot be asserted by vitest — this is unit-tested at the pure-function and CSS-class level only (frontend/src/lib/terminalScale.test.ts, frontend/src/components/__tests__/TerminalPanel.scale.test.tsx)."
  - test: "M-49 (BUG-02, #125) — with a real daemon, share a session, connect a second guest browser, then have the owner stop/end the session from the host."
    expected: "The guest sees the colorblind-safe SessionEndedBanner ('Session ended — the owner stopped this session.') instead of a frozen/silent terminal; no auto-reconnect occurs."
    why_human: "Requires a real WebSocket close event delivered end-to-end from a live daemon to a real browser guest; the close-reason wire format and the banner's accessibility/injection-safety contract are unit-tested (internal/webserver/session_ended_test.go, frontend/src/components/__tests__/SessionEndedBanner.test.tsx), but the live owner-ends-session -> guest-sees-banner integration is not."
  - test: "M-51 (BUG-04, #119 Problem 2) — run a real long-running full-screen TUI (e.g. Claude Code, vim) in a shared session long enough for the 256 KiB scrollback ring to wrap past the ESC[?1049h alt-screen-enter sequence, then connect a late/reconnecting guest."
    expected: "The guest lands on the current screen content in the correct buffer mode, not a blank/garbled window."
    why_human: "Requires real elapsed PTY output volume to force an actual ring wrap and a real late-joining guest connection; the emulator/RenderSnapshot reconstruction logic is unit-tested (internal/relay/scrollback_altscreen_test.go#TestScrollbackAltScreenReplay, TestLiveEmulatorFollowsResize), but the live end-to-end wrap-then-reconnect timing is not."
  - test: "M-50 (BUG-03, #126) — standing regression watch, not a fresh repro request. If a shared session is ever again observed NOT auto-closing its tab on exit, check for 175-05's new slog diagnostic lines (pollSessionStatus: stopping watch... / daemon ListSessions call unusually slow...) to localize the stall."
    expected: "175-01's live timed diagnosis (baseline / >5-min shared / <5-min shared) already confirmed the tab-close behavior works correctly today — this item exists only to catch a future recurrence, not to re-verify a currently-passing behavior."
    why_human: "Already satisfied by the 175-01-DIAGNOSIS.md live run (human-operated, real daemon); included here for completeness per TESTING.md's M-50 entry, not as an outstanding blocker."
---

# Phase 175: Web-share, Remote-viewer & Windowing Bug Fixes Verification Report

**Phase Goal:** Fix the outstanding web-share / remote-viewer / windowing bugs that degrade the guest and shared-session experience — a mobile guest can read the terminal (BUG-01/#128), a remote viewer learns when the owner ends the session (BUG-02/#125), an exited shared session cleans up its own tab (BUG-03/#126), and host/guest session-open never lands in a dead empty window (BUG-04/#119; re-verify against the Phase 168-03 in-app-tab fix first, scope only the residual gap).
**Verified:** 2026-07-08T22:55:55Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | BUG-01: guest terminal stops shrinking at a readability floor and offers horizontal scroll below it, host path unaffected | ✓ VERIFIED | `frontend/src/lib/terminalScale.ts` (`computeGuestViewport`, `DEFAULT_GUEST_MIN_SCALE=0.7`); wired in `frontend/src/components/TerminalPanel.tsx:191-200`; CSS `.terminal-guest--scroll-x` in `frontend/src/style.css:95`; 106 relevant vitest tests pass (`terminalScale.test.ts`, `TerminalPanel.scale.test.tsx`) |
| 2 | BUG-01 live: real narrow-phone viewport is actually legible | ⚠️ Human-only (Step 8) | jsdom cannot measure real pixel legibility — see Human Verification M-48 |
| 3 | BUG-02: both WS write pumps send a fixed close code + reason on `hub.Done()` instead of a silent bare `return` | ✓ VERIFIED | `internal/webserver/server.go:1614`, `internal/relay/server.go:450` — `conn.Close(websocket.StatusNormalClosure, "session ended")`; `TestSessionEnd_HubDone_CarriesCloseReason` PASS (single named run) |
| 4 | BUG-02: guest renders a colorblind-safe disconnect banner (text + `role=status` + `aria-live=polite`), never renders the raw `CloseEvent.reason`, no auto-reconnect | ✓ VERIFIED | `frontend/src/components/SessionEndedBanner.tsx`; wired guest-only in `TerminalPanel.tsx:965-968`; 106 vitest tests pass incl. hostile-string injection guard in `SessionEndedBanner.test.tsx` |
| 5 | BUG-02 live: owner ending session -> guest actually sees the banner end-to-end | ⚠️ Human-only (Step 8) | Requires a live daemon + real browser guest — see Human Verification M-49 |
| 6 | BUG-03: an exited shared session auto-closes its tab, matching unshared-session behavior | ✓ VERIFIED | `175-01-DIAGNOSIS.md` live timed run: baseline, >5-min shared, and <5-min shared cases ALL auto-closed normally — VERDICT DISPROVED (deadline was never the cause; behavior already works). Diagnostic instrumentation shipped for future recurrence: `app.go` `slog.Warn` at all 3 non-exit terminal branches (`daemon-gone`, `session-removed`, `deadline-expiry`) plus a >2s `ListSessions` round-trip warning. `TestShouldContinuePolling` unchanged/GREEN confirms no behavior drift. |
| 7 | BUG-04 Problem 1 (MDI-vs-tab): remote session cards already expose an in-app "Open in tab" affordance (168-03) — re-verified, no code change needed | ✓ VERIFIED | `frontend/src/components/Hub/SessionCard.tsx:447-450` — "Open in tab" wired to `onOpenInBrowser` |
| 8 | BUG-04 Problem 2: reconnecting/late-joining viewer reconstructs the correct buffer mode + current screen content after the raw 256 KiB scrollback ring wraps past `ESC[?1049h`, via a lazy, continuously-fed per-hub VT emulator that is correctly resized and does not double-count frames | ✓ VERIFIED | `internal/relay/hub.go` (`EnsureLiveEmulator`, `RenderSnapshot`, `recordFrame`, `resizeLiveEmulator`); `TestScrollbackAltScreenReplay` PASS, `TestLiveEmulatorFollowsResize` PASS (CR-01 regression guard), `TestHub_TwoClientsFanOut` PASS (CR-03 regression guard, 0/50 failures per code review) — all 3 run as single named tests, all GREEN |
| 9 | BUG-04 live: real two-client, real-TUI ring-wrap-then-reconnect end-to-end | ⚠️ Human-only (Step 8) | Requires a real long-running full-screen TUI + a real late-joining guest — see Human Verification M-51 |

**Score:** 6/9 truths verified (3 routed to human verification — all explicitly deferred by the plans' own `<verification>` sections as not jsdom/unit-testable; not gaps)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `frontend/src/lib/terminalScale.ts` | Floor-aware guest scale helper | ✓ VERIFIED | `computeGuestViewport` + `DEFAULT_GUEST_MIN_SCALE` present, exported, unit-tested |
| `frontend/src/components/__tests__/TerminalPanel.scale.test.tsx` | Component-level floor/scroll wiring tests | ✓ VERIFIED | Narrow/wide/host-path/CSS-gate tests present and passing |
| `app.go` (pollSessionStatus) | Diagnosis-gated BUG-03 fix | ✓ VERIFIED | `shouldContinuePolling`/`maxPollWindow` unchanged (correct DISPROVED-branch outcome); `slog.Warn` instrumentation at 4 sites |
| `app_poll_test.go` | Pure-helper deadline math tests | ✓ VERIFIED | `TestShouldContinuePolling` (4 subtests) + `TestShouldContinuePolling_ZeroWindow` all PASS |
| `internal/relay/hub.go` | Live per-hub VT emulator + RenderSnapshot | ✓ VERIFIED | `EnsureLiveEmulator`, `RenderSnapshot`, `feedLiveEmulator`/`recordFrame`, `resizeLiveEmulator` all present |
| `internal/relay/scrollback_altscreen_test.go` | Alt-screen reconnect + resize regression tests | ✓ VERIFIED | `TestScrollbackAltScreenReplay`, `TestLiveEmulatorFollowsResize` both present and GREEN |
| `internal/webserver/session_ended_test.go` | WS close-reason regression test | ✓ VERIFIED | `TestSessionEnd_HubDone_CarriesCloseReason` present and GREEN |
| `frontend/src/components/SessionEndedBanner.tsx` | Colorblind-safe disconnect banner | ✓ VERIFIED | `role=status`, `aria-live=polite`, fixed copy, `reason` prop never rendered |
| `frontend/src/components/__tests__/SessionEndedBanner.test.tsx` | Banner accessibility + injection-guard tests | ✓ VERIFIED | 6 tests present and passing, incl. hostile-string injection guard |
| `175-01-DIAGNOSIS.md` | BUG-03 live diagnosis with CONFIRMED/DISPROVED verdict | ✓ VERIFIED | Present, DISPROVED verdict recorded with 3 timed-run rows |
| `175-REVIEW.md` | Code review of the phase's changes | ✓ VERIFIED | `status: resolved`; CR-01/CR-03 fixed in code (confirmed below), CR-02/IN-01 fixed via corrected doc comment at `recordFrame` (partial — see Anti-Patterns) |
| `TESTING.md` | Suite Manifest + traceability + M-NN items reconciled | ✓ VERIFIED | M-48..M-51 present (`grep -n "M-4[7-9]\|M-5[0-1]"`); 6 new/extended traceability rows for BUG-01..04 present; Category P reconciled to `/app/` React SPA |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `TerminalPanel.recomputeScale` | `computeGuestViewport` | direct import + call | ✓ WIRED | `TerminalPanel.tsx:14,191` |
| `TerminalPanel` guest container | `.terminal-guest--scroll-x` CSS | `classList.toggle` | ✓ WIRED | `TerminalPanel.tsx:200` toggles class defined in `style.css:95` |
| `internal/webserver/server.go` write pump | WS close on `hub.Done()` | `conn.Close(StatusNormalClosure, "session ended")` | ✓ WIRED | `server.go:1607-1614` |
| `internal/relay/server.go` write pump | WS close on `hub.Done()` | `conn.Close(StatusNormalClosure, "session ended")` | ✓ WIRED | `server.go:443-450` |
| `frontend/src/lib/relayClient.ts` `ws.onclose` | `RelayClient.onClose(code, reason)` | passthrough of `CloseEvent` fields | ✓ WIRED | 3 new tests in `relayClient.test.ts` GREEN |
| `TerminalPanel` `onClose` callback | `SessionEndedBanner` render | `setSessionEnded` state, guest-only gate | ✓ WIRED | `TerminalPanel.tsx:360-363, 965-968` |
| `internal/relay/server.go` WS replay site | `Hub.EnsureLiveEmulator`/`RenderSnapshot` | replaces raw `ScrollbackSnapshot()` write | ✓ WIRED | confirmed in 175-04/175-REVIEW; both replay sites (`relay/server.go`, `webserver/server.go`) emit the emulator-derived preamble |
| `Hub.ResizeClient` | `Hub.resizeLiveEmulator` | called after `hub.mu` released, `emuMu`-only | ✓ WIRED | `hub.go:404-405`; regression-guarded by `TestLiveEmulatorFollowsResize` (CR-01 fix) |
| `Hub.Run()` drain loop | `Hub.recordFrame` (scrollback + emulator, atomic) | single call site per PTY read | ✓ WIRED | `hub.go:454`; regression-guarded by `TestHub_TwoClientsFanOut` (CR-03 fix, 0/50 flake) |
| `SessionCard.tsx` remote card kebab menu | `handleOpenRemoteSession` (in-app tab) | `onOpenInBrowser` prop | ✓ WIRED | confirmed pre-existing from Phase 168-03, re-verified not regressed |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Pure deadline-math helper (BUG-03) | `go test . -run TestShouldContinuePolling -v` | 4 subtests + zero-window test PASS | ✓ PASS |
| WS close-reason on session end (BUG-02) | `go test ./internal/webserver/... -run TestSessionEnd_HubDone_CarriesCloseReason -v` | PASS (2.05s) | ✓ PASS |
| Alt-screen reconnect + emulator resize + fan-out atomicity (BUG-04) | `go test ./internal/relay/... -run 'TestScrollbackAltScreenReplay\|TestLiveEmulatorFollowsResize\|TestHub_TwoClientsFanOut' -v` | all 3 PASS | ✓ PASS |
| Guest readability floor + banner + relayClient close (BUG-01/BUG-02) | `pnpm exec vitest run terminalScale.test.ts TerminalPanel.scale.test.tsx SessionEndedBanner.test.tsx relayClient.test.ts` | 106/106 PASS | ✓ PASS |
| Full build | `go build ./...` | clean | ✓ PASS |
| Debt-marker scan | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across all 12 phase-touched source files | 0 matches | ✓ PASS |

Orchestrator-supplied evidence (not independently re-run in full, spot-checked above): `go build ./...` clean, full `go test ./...` green, full relay+webserver `-race` clean, frontend `tsc` clean + vitest 2411/2411 pass, `TestHub_TwoClientsFanOut` 0/50 failures post-CR-03-fix.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| BUG-01 (#128) | 175-02 (test scaffold), 175-03 (fix), 175-07 (M-48) | Web-share terminal legible on mobile | ✓ SATISFIED (code); live confirmation → human_needed | `computeGuestViewport` + wiring + tests |
| BUG-02 (#125) | 175-02, 175-06, 175-07 (M-49) | Remote viewer sees a clear disconnect notice | ✓ SATISFIED (code); live confirmation → human_needed | WS close-reason + `SessionEndedBanner` + tests |
| BUG-03 (#126) | 175-01 (diagnosis), 175-02, 175-05, 175-07 (M-50) | Exiting from inside a shared session auto-closes its tab | ✓ SATISFIED | Live diagnosis DISPROVED any regression; behavior already correct; instrumentation shipped for future recurrence |
| BUG-04 (#119) | 175-02, 175-04, 175-07 (M-51) | Host card interaction + guest session-open produce a working session view | ✓ SATISFIED (code); live confirmation → human_needed | Problem 1 re-verified (168-03); Problem 2 fixed + regression-tested (incl. code-review CR-01/CR-03 fixes) |

**Note:** Phase 175's requirements are tracked as BUG-01..04 in ROADMAP.md (GitHub-issue-scoped), not as REQ-IDs in REQUIREMENTS.md — no entries expected there; this is expected, not a gap.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/relay/hub.go` | 115-117 | Struct-field doc comment for `emuMu` still states "a slow/stuck emulator write must never stall the PTY-drain/broadcast loop" — the code-review resolution note claims this overclaim was corrected, but only the `recordFrame` function-level comment (line ~478-483) was actually updated to name the real mitigation (`liveEmuQueryStripPattern`, not goroutine isolation); this earlier struct-field comment was not touched by the CR-02 fix commit (`d4c309db`) | ℹ️ Info | Documentation-only inconsistency, not a functional defect — the actual runtime protection (query-strip pattern) is real, present, and tested, matching the precedent already used by `internal/daemon/engine.go`'s `GetSessionStyledTailLines`. A future maintainer reading only the struct-field comment (not the `recordFrame` doc) could still be misled into believing there's an absolute stall-immunity guarantee. Does not block any BUG-0x truth. |
| `frontend/src/components/TerminalPanel.tsx` | 358-363 | WR-02 (code review, unresolved): `client.close()` in the unmount cleanup fires the WS close handshake asynchronously; the deferred `onClose` callback can run `setSessionEnded(...)` against an already-unmounted component. React 18 silently no-ops this (confirmed: no crash, no console warning) — same accepted pattern as the pre-existing Phase-94 WR-01 precedent in this same file. | ℹ️ Info | Not user-visible today per the code review's own analysis; explicitly left unaddressed by the review's `resolution` note (only CR-01/CR-02/CR-03/WR-01 are listed as fixed). Latent footgun for future changes that add real side effects to `onClose`, not a current defect. |

No debt markers (`TBD`/`FIXME`/`XXX`), no `TODO`/`HACK`/`PLACEHOLDER` comments, and no stub/empty-implementation patterns found in any of the 12 phase-touched source files (`app.go`, `app_poll_test.go`, `internal/relay/hub.go`, `internal/relay/scrollback_altscreen_test.go`, `internal/relay/server.go`, `internal/webserver/server.go`, `internal/webserver/session_ended_test.go`, `frontend/src/lib/terminalScale.ts`, `frontend/src/components/TerminalPanel.tsx`, `frontend/src/components/SessionEndedBanner.tsx`, `frontend/src/lib/relayClient.ts`, `frontend/src/style.css`).

### Human Verification Required

Three items require a live app + real daemon + real browser/device and were explicitly deferred by the plans' own `<verification>` sections (not automatable from vitest/jsdom or Go unit tests), plus one standing regression watch that is already satisfied:

### 1. M-48 — BUG-01 live mobile-viewport readability (#128)

**Test:** On a real narrow-phone browser viewport (or a resized desktop browser window simulating one), open a shared web session.
**Expected:** The 80-col terminal grid stops shrinking at the readability floor (~0.7 scale) and switches to horizontal scroll instead of continuing to shrink to unreadable text.
**Why human:** jsdom reports `clientWidth`/`clientHeight` as 0 — real pixel-legibility cannot be asserted in the unit-test harness.

### 2. M-49 — BUG-02 live owner-ends-session disconnect (#125)

**Test:** With a real daemon, share a session, connect a second guest browser, then have the owner stop/end the session from the host.
**Expected:** The guest sees the colorblind-safe `SessionEndedBanner` ("Session ended — the owner stopped this session.") instead of a frozen/silent terminal; no auto-reconnect occurs.
**Why human:** Requires a real WebSocket close event delivered end-to-end from a live daemon to a real browser guest.

### 3. M-51 — BUG-04 live two-client alt-screen reconnect (#119)

**Test:** Run a real long-running full-screen TUI in a shared session long enough for the raw scrollback ring to wrap past the alt-screen-enter sequence, then connect a late/reconnecting guest.
**Expected:** The guest lands on the current screen content in the correct buffer mode, not a blank/garbled window.
**Why human:** Requires real elapsed PTY output volume to force an actual ring wrap and a real late-joining guest connection.

### 4. M-50 — BUG-03 standing regression watch (#126) — already satisfied

**Test:** No action required now. If a shared session is ever again observed not auto-closing its tab on exit, check for 175-05's new `slog` diagnostic lines to localize the stall.
**Expected:** N/A — this is a forward-looking watch item, not a pending verification. 175-01's live timed diagnosis already confirmed the tab-close behavior works correctly today (baseline, >5-min shared, and <5-min shared cases all closed normally).
**Why human:** Already satisfied by the 175-01-DIAGNOSIS.md live run; listed here only because TESTING.md's M-50 entry references it.

### Gaps Summary

No blocking gaps. All four BUG-0x code deliverables are present, wired, and covered by passing automated tests (including three code-review-discovered-and-fixed defects: CR-01 emulator-never-resized, CR-03 bootstrap/feed double-count race, and BUG-03's diagnosis-gated DISPROVED-branch instrumentation). Two Info-level documentation/latent-footgun findings were noted (stale struct-doc-comment overclaim; WR-02 unmount-race, both non-functional and explicitly accepted/deferred by the code review) — neither blocks the phase goal. Three items require live human UAT that cannot be automated (real narrow viewport, real live daemon + two browsers, real long-running TUI) and were explicitly deferred to TESTING.md M-48/M-49/M-51 by the plans themselves; a fourth (M-50) is a forward-looking regression watch already satisfied by 175-01's completed live diagnosis.

---

_Verified: 2026-07-08T22:55:55Z_
_Verifier: Claude (gsd-verifier)_
