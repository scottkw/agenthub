---
phase: 167-native-notifications
plan: 04
subsystem: ui
tags: [react, settings, notifications, testing-md, traceability]

# Dependency graph
requires:
  - phase: 167-native-notifications (Plan 03)
    provides: "App.GetNotifyOnWaiting()/SetNotifyOnWaiting(bool) Wails-bound methods + 4 wailsjs binding files"
provides:
  - "Notify-on-waiting toggle in SettingsTab.tsx Behavior section (default OFF, mirrors startMinimized end-to-end)"
  - "SettingsSearch.tsx index entry for the toggle, targeting settings-behavior"
  - "SettingsTab.notify-toggle.test.tsx — render/default-off/load/save/save-error + Behavior-section placement guard"
  - "TESTING.md: Suite Manifest note, NTF-01..04 traceability rows, Category U / M-41 manual checklist item"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "notifyOnWaiting toggle is an instant, no-confirm toggle mirroring handleToggleMinimized exactly (not the confirm-dialog pattern used by the shell web-share warning toggle) — appropriate because turning notifications on/off has no destructive consequence needing a confirm gate"

key-files:
  created:
    - frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx
  modified:
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/SettingsSearch.tsx
    - frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
    - TESTING.md

key-decisions:
  - "Toggle placed under <h3 id=\"settings-behavior\"> (Behavior section), NOT settings-session-behavior, per the LOCKED user correction recorded in 167-CONTEXT.md that overrides NTF-04's original 'Session Behavior' wording"
  - "handleToggleNotifyOnWaiting is instant (await Set -> setState on success, inline error on failure) with no confirm dialog, mirroring handleToggleMinimized rather than the shell-warn toggle's D-07 confirm-on-disable pattern — there is no destructive consequence to flipping this preference either direction"

requirements-completed: [NTF-04]

coverage:
  - id: D1
    description: "Notify-on-waiting toggle renders in the Behavior section, defaults off, loads via GetNotifyOnWaiting, and persists via SetNotifyOnWaiting"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx"
        status: pass
    human_judgment: false
  - id: D2
    description: "Toggle sits physically inside the Behavior section (between the Behavior and Session Behavior headings), not Session Behavior — source-level placement guard"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx#renders the toggle inside the Behavior section, not Session Behavior (LOCKED correction)"
        status: pass
    human_judgment: false
  - id: D3
    description: "SetNotifyOnWaiting rejection surfaces an inline error and does not flip the displayed checked state"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx#a SetNotifyOnWaiting rejection surfaces an inline error and does not flip the displayed state"
        status: pass
    human_judgment: false
  - id: D4
    description: "Settings search surfaces the notify toggle label and targets settings-behavior"
    requirement: "NTF-04"
    verification:
      - kind: other
        ref: "frontend/src/components/SettingsSearch.tsx SEARCH_INDEX entry: { label: 'Notify me when a session is awaiting input', target: 'settings-behavior' }"
        status: pass
    human_judgment: false
  - id: D5
    description: "M-41 manual item + Category U + traceability rows registered in TESTING.md; check-traceability-paths.sh exits 0"
    requirement: "NTF-04"
    verification:
      - kind: other
        ref: "bash tests/check-traceability-paths.sh"
        status: pass
    human_judgment: false
  - id: D6
    description: "Real cross-platform on-screen notification delivery + tray-hidden behavior, toggle-off no-op"
    verification: []
    human_judgment: true
    rationale: "Requires live desktop sessions on macOS/Windows/Linux observing the real OS notification center render the banner while the window is hidden in the tray. Tracked as M-41 (Category U) in TESTING.md — the trigger logic, de-dup, cold-start baseline, body format, and disabled-toggle no-op are all unit-proven via the injected sendNotificationFunc seam (Plan 03), but real OS delivery cannot be asserted in headless CI."

# Metrics
duration: 20min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 04: Settings Toggle + Regression Suite Registration Summary

**Notify-on-waiting toggle lands in the Behavior section (default OFF, mirrors startMinimized end-to-end), and TESTING.md gains the Phase-167 Suite Manifest note, four NTF traceability rows, and Category U's M-41 manual delivery item — closing the user-visible half of NTF-04.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2 completed
- **Files modified:** 7 (4 frontend source/test, 2 pre-existing test files fixed for a Rule-3 mock regression, 1 TESTING.md)

## Accomplishments

- `SettingsTab.tsx` gained a full notify-on-waiting toggle: `notifyOnWaiting`/`notifyOnWaitingLoaded`/`notifyOnWaitingSaving`/`notifyOnWaitingError` state, a mount-time `useEffect` calling `GetNotifyOnWaiting()`, and `handleToggleNotifyOnWaiting()` — an instant (no confirm-dialog) save handler that calls `SetNotifyOnWaiting(next)`, updates state on success, and renders an inline error on rejection without flipping the displayed checked state. The toggle JSX sits under `<h3 id="settings-behavior">Behavior</h3>` — physically between the "Start minimized" toggle and the `settings-session-behavior` heading — per the LOCKED user correction that overrides NTF-04's original "Session Behavior" wording.
- `SettingsSearch.tsx` gained one `SEARCH_INDEX` entry: `{ label: 'Notify me when a session is awaiting input', target: 'settings-behavior' }`, so the autocomplete search jumps to the correct section.
- `SettingsTab.notify-toggle.test.tsx` (new, 15 tests): source-inspection tests for the Wails imports, state quartet, mount-time load call, handler existence, exact label text, and a source-level placement guard (toggle index falls after the Behavior heading and before the Session Behavior heading); DOM tests (createRoot + flushSync + act pattern) covering default-OFF render, flip-to-true persisting + reflecting checked state, a `SetNotifyOnWaiting` rejection surfacing `.settings-panel__error` without flipping the checkbox, load-resolves-true render, and flip-to-false.
- `TESTING.md` updated per the repo-root CLAUDE.md Standing Convention: a new Suite Manifest (§2) note summarizing all of Phase 167's test additions across Plans 01/03/04 (counts 368→370 Go / 134→135 vitest / 513→516 total); four new Traceability Map (§4) rows (NTF-01→`app_test.go` cold-start baseline, NTF-02→`app_test.go` de-dup+pruning, NTF-03→`app_test.go` body-format+display-name, NTF-04→`engine_notify_test.go`+`api_notify_test.go`+`app_test.go`+`SettingsTab.notify-toggle.test.tsx`); and a new `### Category U — Native Notifications (NTF)` under §5 containing **M-41** (per-platform macOS/Windows/Linux enable→hide-to-tray→drive-to-waiting→confirm exactly one attributed notification; toggle-OFF confirms none). `bash tests/check-traceability-paths.sh` exits 0.

## Task Commits

Each task was committed atomically:

1. **Task 1: Notify-on-waiting toggle in the Behavior section + search entry** - `32c7815c` (feat)
2. **Task 2: Register Phase-167 tests + M-41 in TESTING.md (Standing Convention)** - `11f64148` (docs)

**Plan metadata:** (this commit, following SUMMARY)

## Files Created/Modified

- `frontend/src/components/SettingsTab.tsx` — `GetNotifyOnWaiting`/`SetNotifyOnWaiting` import, `notifyOnWaiting` state quartet, load `useEffect`, `handleToggleNotifyOnWaiting`, toggle JSX in the Behavior section
- `frontend/src/components/SettingsSearch.tsx` — one `SEARCH_INDEX` entry targeting `settings-behavior`
- `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` — new, 15 tests (render/default-off/load/save/save-error + placement guard)
- `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` — Rule 3 fix: added `GetNotifyOnWaiting`/`SetNotifyOnWaiting` to the Wails mock (this test unconditionally renders `SettingsTab`, which now calls the new binding on mount)
- `frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx` — same Rule 3 fix
- `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx` — same Rule 3 fix
- `TESTING.md` — Suite Manifest note, NTF-01..04 traceability rows, Category U + M-41

## Decisions Made

- Followed `handleToggleMinimized`'s instant-save shape exactly for `handleToggleNotifyOnWaiting` (no confirm dialog) rather than the shell web-share warning toggle's D-07 confirm-on-disable pattern — flipping a notification preference has no destructive consequence in either direction, so a confirm gate would be unnecessary friction.
- Kept the toggle's default state (`useState(false)`) and description copy explicit about the default-off behavior, reinforcing NTF-04's "no surprise notifications on upgrade" intent from 167-CONTEXT.md.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Three pre-existing SettingsTab test files broke after the new unconditional `GetNotifyOnWaiting()` mount call**
- **Found during:** Task 1, running the full `pnpm test` suite after adding the toggle
- **Issue:** `SettingsTab.appearance-theme.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, and `SettingsTab.shellPath.test.tsx` each hand-roll a `vi.mock('../../wailsjs/go/main/App', ...)` factory listing every binding `SettingsTab` imports. None of the three included the new `GetNotifyOnWaiting`/`SetNotifyOnWaiting` exports, so mounting `SettingsTab` in those tests threw `[vitest] No "GetNotifyOnWaiting" export is defined on the mock` inside the new mount-time `useEffect`.
- **Fix:** Added `GetNotifyOnWaiting: vi.fn().mockResolvedValue(false)` and `SetNotifyOnWaiting: vi.fn().mockResolvedValue(undefined)` to each of the three mock factories, matching the existing style used for every other binding in those files.
- **Files modified:** `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx`, `frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx`, `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx`
- **Commit:** `32c7815c`

## Issues Encountered

None beyond the Rule 3 fix above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 167 (Native Notifications) is now feature-complete: Plan 01 (persisted setting), Plan 02 (beeep wrappers), Plan 03 (GUI trigger + Wails binding), and Plan 04 (Settings toggle + regression suite registration) are all done.
- `go test -race -short ./...` is green (370 Go test files); `cd frontend && pnpm test` is green (135 vitest files, 2268 tests); `npx tsc --noEmit` is clean; `bash tests/check-traceability-paths.sh` exits 0.
- Remaining phase-level work is the manual M-41 UAT (real OS notification delivery across macOS/Windows/Linux, including the tray-hidden case) — tracked in TESTING.md Category U, to be run before the next tagged release alongside the rest of the Manual Regression Checklist.
- No blockers. Phase 167 is ready for `/gsd-verify-work` / phase closeout.

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED

All modified/created files confirmed present on disk (SettingsTab.tsx, SettingsSearch.tsx, SettingsTab.notify-toggle.test.tsx, TESTING.md, this SUMMARY.md), and both task commit hashes (32c7815c, 11f64148) confirmed present in git log.
