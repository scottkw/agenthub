---
phase: 167-native-notifications
plan: 03
subsystem: infra
tags: [go, wails, notifications, tray-poller, edge-detection]

# Dependency graph
requires:
  - phase: 167-native-notifications (Plan 01)
    provides: "NotifyOnWaiting persisted daemon setting + DaemonClient.GetNotifyOnWaiting/SetNotifyOnWaiting"
  - phase: 167-native-notifications (Plan 02)
    provides: "Cross-platform sendNotification(identifier, title, body string) primitive on darwin/windows/linux"
provides:
  - "displayNameForCLI(cli string) string — static CLI->display-name mirror of internal/pty/detect.go's knownCLIs"
  - "(a *App) maybeNotifyWaiting(sessions []SessionInfo) — edge-detected, de-duped, cold-start-safe waiting-transition notifier"
  - "App.GetNotifyOnWaiting() / App.SetNotifyOnWaiting(bool) Wails-bound methods"
  - "refreshTrayState wired to call maybeNotifyWaiting every 5s tick, independent of window visibility"
  - "wailsjs bindings (4 files) for the two new bound methods"
affects: [167-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sendNotificationFunc injection field on App mirrors the saveFileDialogFunc pattern — lets unit tests assert on notification calls without touching real OS notification APIs"
    - "Edge-triggered de-dup over a polled slice: maybeNotifyWaiting keeps a map[string]string of last-observed Status per session ID, updated once per tick from the same a.ListSessions() call already used for tray icon/tooltip — no second ticker, no extra socket read"

key-files:
  created: []
  modified:
    - app.go
    - app_test.go
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/wailsjs/go/main/App.js
    - frontend/src/wailsjs/wailsjs/go/main/App.d.ts

key-decisions:
  - "maybeNotifyWaiting requires the session to have a KNOWN previous status (not just firstRun=false) before firing — a brand-new session that appears already in waiting status on the tick after cold-start fires nothing, matching the reference implementation in RESEARCH.md exactly (prev != waiting AND known)."
  - "SetNotifyOnWaiting stores the atomic cache BEFORE checking a.client != nil, so the in-process cache (read by the tray poller on the very next tick) always reflects the latest toggle state even if the daemon round-trip fails."
  - "GetNotifyOnWaiting reads only the cached atomic.Bool (no daemon round-trip) so the tray poller's edge detector and the Settings toggle read path can never disagree mid-tick."

patterns-established: []

requirements-completed: [NTF-01, NTF-02, NTF-03, NTF-04]

coverage:
  - id: D1
    description: "A non-waiting->waiting transition fires exactly one notification per transition; a session held in waiting across subsequent ticks fires nothing more"
    requirement: "NTF-02"
    verification:
      - kind: unit
        ref: "app_test.go#TestMaybeNotifyWaiting"
        status: pass
    human_judgment: false
  - id: D2
    description: "Cold-start baseline: the first poll tick after launch captures current statuses silently (no notification for sessions already waiting); a later tick's new transition does fire"
    requirement: "NTF-02"
    verification:
      - kind: unit
        ref: "app_test.go#TestMaybeNotifyWaiting_FirstTickNoNotify"
        status: pass
    human_judgment: false
  - id: D3
    description: "Notification body contains the session Name and the agent display name; title is AgentHub; per-session identifier agenthub.session-waiting.<id>"
    requirement: "NTF-03"
    verification:
      - kind: unit
        ref: "app_test.go#TestMaybeNotifyWaiting_BodyFormat"
        status: pass
    human_judgment: false
  - id: D4
    description: "With the NotifyOnWaiting toggle OFF, maybeNotifyWaiting fires nothing regardless of transitions"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "app_test.go#TestMaybeNotifyWaiting_DisabledNoop"
        status: pass
    human_judgment: false
  - id: D5
    description: "A session absent from the latest ListSessions() slice is pruned from lastWaitingStatus (bounded map growth)"
    requirement: "NTF-02"
    verification:
      - kind: unit
        ref: "app_test.go#TestMaybeNotifyWaiting_Pruning"
        status: pass
    human_judgment: false
  - id: D6
    description: "displayNameForCLI maps every known CLI to its human display name and falls back to the raw input for shells/unknowns"
    requirement: "NTF-03"
    verification:
      - kind: unit
        ref: "app_test.go#TestDisplayNameForCLI"
        status: pass
    human_judgment: false
  - id: D7
    description: "refreshTrayState (the always-on 5s tray poller, independent of window visibility) drives maybeNotifyWaiting; startup caches the persisted toggle; Get/SetNotifyOnWaiting are Wails-bound and reachable from all 4 wailsjs binding files; frontend tsc is clean"
    requirement: "NTF-01"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./..."
        status: pass
      - kind: other
        ref: "grep -q 'a.maybeNotifyWaiting(sessions)' app.go (refreshTrayState connected branch)"
        status: pass
      - kind: other
        ref: "grep -rl 'GetNotifyOnWaiting' frontend/src/wailsjs/ | wc -l == 4"
        status: pass
      - kind: other
        ref: "cd frontend && npx tsc --noEmit"
        status: pass
    human_judgment: false
  - id: D8
    description: "Real on-screen delivery + tray-hidden behavior confirmed live by a human"
    verification: []
    human_judgment: true
    rationale: "Requires a live GUI session observing the OS notification center actually render the banner while the window is hidden in the tray; this plan only proves the trigger logic and Wails binding surface through the injected sendNotificationFunc seam. Deferred to manual UAT M-41 (registered in Plan 04)."

# Metrics
duration: 12min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 03: GUI Notification Trigger + Toggle Binding Summary

**`maybeNotifyWaiting` edge-detects the non-waiting->waiting transition inside the existing 5s tray poller and fires through the injected `sendNotificationFunc` seam; `GetNotifyOnWaiting`/`SetNotifyOnWaiting` Wails-bound and wired end-to-end into all 4 wailsjs binding files.**

## Performance

- **Duration:** ~12 min
- **Tasks:** 2 completed
- **Files modified:** 6 (2 Go, 4 frontend binding files)

## Accomplishments
- `displayNameForCLI(cli string) string` added as a static switch mirroring `internal/pty/detect.go`'s `knownCLIs` table (no live PATH scan, per LOCKED decision #5) — maps `claude`/`codex`/`gemini`/`opencode`/`agy` to their human display names, falls back to the raw input for shells and unknowns.
- `(a *App) maybeNotifyWaiting(sessions []SessionInfo)` added: no-ops when `notifyOnWaiting` is off; on the first-ever call baselines all current statuses without firing (cold-start burst prevention); on subsequent calls fires exactly once per session whose previously-known status was non-waiting and current status is waiting, with body `"<Name> (<DisplayName>) is waiting for your input."` and per-session identifier `agenthub.session-waiting.<id>`; prunes session IDs no longer present in the latest slice.
- `App` struct gained `notifyOnWaiting atomic.Bool`, `lastWaitingStatus map[string]string`, and `sendNotificationFunc func(identifier, title, body string)` (defaults to the real `sendNotification` in `NewApp`) — the injection seam lets all five behaviors be unit-tested with zero real OS notifications.
- `refreshTrayState`'s connected branch now calls `a.maybeNotifyWaiting(sessions)` immediately after `sessions = a.ListSessions()` — reuses the slice the tray icon/tooltip already fetch every 5s tick, satisfying the tray-hidden requirement (NTF-01) with no new ticker and no extra socket round trip.
- `startup` reads `a.client.GetNotifyOnWaiting()` after the daemon client is available and caches the result into the atomic (silently leaves the zero-value `false` on error — the safe default).
- New Wails-bound methods: `GetNotifyOnWaiting() bool` (reads the cached atomic — no daemon round trip, so the tray poller and the Settings toggle never disagree mid-tick) and `SetNotifyOnWaiting(val bool) error` (stores the atomic immediately, then persists via `a.client.SetNotifyOnWaiting`, mirroring `SetStartMinimized`'s nil-client error contract).
- Hand-added the two new bindings to all four wailsjs files: the hand-maintained, git-tracked `frontend/src/wailsjs/go/main/{App.js,App.d.ts}` (grouped near the other Settings-toggle bound methods, matching existing section-comment style) and the git-ignored, Wails-auto-regenerated `frontend/src/wailsjs/wailsjs/go/main/{App.js,App.d.ts}` (inserted alphabetically, matching that tree's existing sort order) so `npx tsc --noEmit` passes locally today, ahead of the next `wails build` regenerating them anyway.

## Task Commits

Each task was committed atomically:

1. **Task 1: displayNameForCLI + maybeNotifyWaiting edge detector (test-first via injection seam)** - `27a2588f` (feat)
2. **Task 2: Wire refreshTrayState + startup load + Get/SetNotifyOnWaiting bound methods + wailsjs bindings** - `ea82ded5` (feat)

**Plan metadata:** (this commit, following SUMMARY)

## Files Created/Modified
- `app.go` - `displayNameForCLI`, `maybeNotifyWaiting`, `App` fields (`notifyOnWaiting`, `lastWaitingStatus`, `sendNotificationFunc`), `NewApp` default wiring, `refreshTrayState` call site, `startup` initial-value load, `GetNotifyOnWaiting`/`SetNotifyOnWaiting` bound methods
- `app_test.go` - `TestMaybeNotifyWaiting`, `TestMaybeNotifyWaiting_FirstTickNoNotify`, `TestMaybeNotifyWaiting_BodyFormat`, `TestMaybeNotifyWaiting_DisabledNoop`, `TestMaybeNotifyWaiting_Pruning`, `TestDisplayNameForCLI`
- `frontend/src/wailsjs/go/main/App.js` / `App.d.ts` - hand-added `GetNotifyOnWaiting`/`SetNotifyOnWaiting` bindings (tracked tree)
- `frontend/src/wailsjs/wailsjs/go/main/App.js` / `App.d.ts` - same two bindings, alphabetically placed (git-ignored, Wails-auto-regenerated tree — updated on disk for local `tsc`, not committed)

## Decisions Made
- Followed the RESEARCH.md reference implementation for `maybeNotifyWaiting` verbatim: the fire condition requires BOTH `known` (the session ID was present in the previous tick's map) AND `prev != waiting`, not just "not first run." This means a session that is brand-new to the poller (never seen before) and already reports `waiting` on its very first appearance fires nothing — consistent with the plan's cold-start intent, extended naturally to any session appearing mid-stream without a prior "running" observation.
- `SetNotifyOnWaiting` stores the atomic cache before attempting the daemon persist call, so a transient daemon-client failure never leaves the in-process toggle state stale for the next tray-poller tick.
- Went one test beyond the plan's five named tests: added `TestMaybeNotifyWaiting_Pruning` to directly assert the `lastWaitingStatus` map shrinks when a session disappears (the plan's behavior bullet list included pruning as a bullet but the acceptance criteria only named five tests) — covers threat T-167-06 (map growth DoS) explicitly rather than only implicitly through the other tests.

## Deviations from Plan

None - plan executed exactly as written. (One additional test, `TestMaybeNotifyWaiting_Pruning`, was added beyond the five explicitly named in the plan to give the pruning behavior bullet and the T-167-06 threat-model mitigation its own direct unit-test coverage; this is additive test coverage, not a deviation from any instruction.)

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The GUI-side trigger (`maybeNotifyWaiting`) and the `Get/SetNotifyOnWaiting` Wails binding are complete, unit-tested, and wired into the always-on tray poller. Plan 04 can now build the `SettingsTab.tsx` toggle against `GetNotifyOnWaiting`/`SetNotifyOnWaiting` and register the manual UAT (M-41) for real on-screen delivery + tray-hidden behavior across macOS/Windows/Linux.
- `go test . -race -short` is green; `go build ./...`, `go vet ./...`, and frontend `npx tsc --noEmit` are all clean.
- No blockers.

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED

All modified files confirmed present on disk (app.go, app_test.go, both tracked wailsjs App.js/App.d.ts, both git-ignored wailsjs App.js/App.d.ts, this SUMMARY.md), and both task commit hashes (27a2588f, ea82ded5) confirmed present in git log.
