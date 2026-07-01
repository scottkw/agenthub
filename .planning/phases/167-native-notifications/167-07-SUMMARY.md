---
phase: 167-native-notifications
plan: 07
subsystem: notifications
tags: [react, wails-events, settings-ui, gap-closure, m-41]

# Dependency graph
requires:
  - phase: 167-native-notifications (plan 06)
    provides: "onNotificationAuthResult exported callback emitting a notification:permission-denied Wails event on denial"
provides:
  - "SettingsTab subscription to the notification:permission-denied Wails event"
  - "User-facing remediation hint in the Settings Behavior section directing the user to System Settings > Notifications > AgentHub"
affects: [phase-167-verify-work (M-41 live re-test — signed-build denial path can now be visually confirmed)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "EventsOn(...) subscription in a useEffect with empty deps, returning the unsubscribe function directly (mirrors App.tsx's EventsOn subscriptions)"

key-files:
  created:
    - frontend/src/components/__tests__/SettingsTab.notify-permission-hint.test.tsx
  modified:
    - frontend/src/components/SettingsTab.tsx
    - frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx
    - frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx
    - frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx
    - TESTING.md

key-decisions:
  - "Hint reuses the existing settings-panel__error class (no new CSS) — plan explicitly forbade inventing new classes."
  - "Hint copy uses plain '>' separators (System Settings > Notifications > AgentHub) matching the plan's action-block wording verbatim, rather than a unicode arrow."
  - "The event carries no payload (T-167-12, accepted risk) — the handler just flips a boolean; a spurious event at worst shows a benign static hint."

requirements-completed: [NTF-04]

coverage:
  - id: D1
    description: "Hint hidden by default before any notification:permission-denied event fires"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "SettingsTab.notify-permission-hint.test.tsx — 'does NOT render the hint before any permission-denied event fires'"
        status: pass
    human_judgment: false
  - id: D2
    description: "Hint renders in the Behavior section (between settings-behavior and settings-session-behavior headings) once notification:permission-denied fires"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "SettingsTab.notify-permission-hint.test.tsx — 'renders the hint in the Behavior section once the event fires'"
        status: pass
    human_judgment: false
  - id: D3
    description: "EventsOn subscription registered on mount, unsubscribed on unmount (T-167-13 DoS mitigation)"
    verification:
      - kind: unit
        ref: "SettingsTab.notify-permission-hint.test.tsx — 'subscribes to the notification:permission-denied event on mount' + 'calls the EventsOn-returned unsubscribe function on unmount'"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live signed-build denial → hint appears in the running app"
    verification:
      - kind: manual
        ref: "TESTING.md Category U, M-41 (shared with 167-06)"
        status: pending
    human_judgment: true
    rationale: "Requires a real macOS denied-permission state on a signed production build; go/vitest cannot pump native UNUserNotificationCenter authorization or a real Wails runtime event bridge."

duration: 8min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 07: Notification-Permission-Denied Settings Hint Summary

**SettingsTab now subscribes to the backend's `notification:permission-denied` Wails event and renders a "System Settings > Notifications > AgentHub" remediation hint in the Behavior section — closing the frontend half of the M-41 gap where a proactively-requested, silently-denied permission had no way for the user to discover or fix it.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-07-01 (first commit 2004ffb8)
- **Completed:** 2026-07-01 (last commit 07d9d9a6)
- **Tasks:** 2
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `SettingsTab.tsx`: imports `EventsOn` from the Wails runtime module; adds `notifyPermissionDenied` state; a mount-only `useEffect` subscribes to `notification:permission-denied` and returns the unsubscribe function for cleanup; the existing notify-on-waiting field-group now conditionally renders a `settings-panel__error` hint reading "AgentHub is not allowed to send notifications. Enable it in System Settings > Notifications > AgentHub, then toggle this setting off and on again." when the event has fired.
- `SettingsTab.notify-permission-hint.test.tsx` (new): 4 tests — subscribes on mount, hidden by default, shown (in the correct section) once the event fires, unsubscribe called on unmount.
- Rule 3 (blocking) fix: 4 pre-existing SettingsTab DOM-render test files mocked the Wails runtime module without `EventsOn`. Once `SettingsTab.tsx` started importing `EventsOn`, mounting any of those tests threw `EventsOn is not a function`. Added a no-op `EventsOn: vi.fn().mockReturnValue(vi.fn())` stub to each mock — no test assertions changed.
- `TESTING.md`: Suite Manifest bumped 135→136 vitest / 517→518 total with a Phase 167-07 note; new NTF-04 traceability row for the hint test.

## Task Commits

Each task was committed atomically, following the plan's TDD instruction (RED then GREEN):

1. **Task 1 — RED:** `2004ffb8` (test) — added the failing `SettingsTab.notify-permission-hint.test.tsx` (3 of 4 assertions failed as expected; the hint element and subscription did not yet exist).
2. **Task 1 — GREEN:** `258195c0` (feat) — implemented the `EventsOn` subscription + hint render in `SettingsTab.tsx`, and fixed the 4 pre-existing test files whose runtime mocks were missing `EventsOn` (Rule 3, blocking issue this task's own import change caused).
3. **Task 2:** `07d9d9a6` (docs) — registered the new test in `TESTING.md` Section 2 (counts + note) and Section 4 (NTF-04 traceability row); `tests/check-traceability-paths.sh` passes.

## Files Created/Modified

- `frontend/src/components/__tests__/SettingsTab.notify-permission-hint.test.tsx` (new) — 4 tests covering subscription-on-mount, hidden-by-default, shown-on-event (with section-placement assertion), unsubscribe-on-unmount.
- `frontend/src/components/SettingsTab.tsx` — `EventsOn` import, `notifyPermissionDenied` state, mount effect, conditional hint paragraph in the notify-on-waiting field-group.
- `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` — added `EventsOn` no-op stub to the runtime mock.
- `frontend/src/components/__tests__/SettingsTab.shell-warn-toggle.test.tsx` — added `EventsOn` no-op stub to the runtime mock.
- `frontend/src/components/__tests__/SettingsTab.shellPath.test.tsx` — added `EventsOn` no-op stub to the runtime mock.
- `frontend/src/components/__tests__/SettingsTab.appearance-theme.test.tsx` — added `EventsOn` no-op stub to the runtime mock.
- `TESTING.md` — Suite Manifest counts/note (Section 2), NTF-04 traceability row (Section 4).

## Decisions Made

- Reused the existing `settings-panel__error` CSS class for the hint (plan explicitly forbade inventing new classes) — visually consistent with the adjacent `notifyOnWaitingError` paragraph.
- Copy uses plain `>` separators ("System Settings > Notifications > AgentHub") to match the plan's action-block wording verbatim rather than substituting a unicode arrow, minimizing drift from the spec text.
- No payload validation on the event (T-167-12, accepted per threat model) — the handler is a simple boolean flip; a spurious event at worst shows a benign, non-actionable hint.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added missing `EventsOn` stub to 4 pre-existing SettingsTab test mocks**

- **Found during:** Task 1 GREEN phase, after adding `EventsOn` to `SettingsTab.tsx`'s Wails runtime import.
- **Issue:** `SettingsTab.notify-toggle.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, `SettingsTab.shellPath.test.tsx`, and `SettingsTab.appearance-theme.test.tsx` each `vi.mock` the `wailsjs/wailsjs/runtime/runtime` module with only `BrowserOpenURL`/`ClipboardSetText`. Once `SettingsTab.tsx` called `EventsOn(...)` in a new mount effect, every DOM-render test that mounts `SettingsTab` threw `EventsOn is not a function`, failing 5 previously-passing tests in `notify-toggle.test.tsx` alone.
- **Fix:** Added `EventsOn: vi.fn().mockReturnValue(vi.fn())` to each of the 4 mocks — a no-op stub returning a no-op unsubscribe function, matching how the new test's own mock behaves for callers that don't need to inspect the subscription.
- **Files modified:** `SettingsTab.notify-toggle.test.tsx`, `SettingsTab.shell-warn-toggle.test.tsx`, `SettingsTab.shellPath.test.tsx`, `SettingsTab.appearance-theme.test.tsx`.
- **Verification:** `cd frontend && pnpm vitest run src/components/__tests__/SettingsTab.` — 10 files / 220 tests, all pass.
- **Committed in:** `258195c0` (Task 1 GREEN commit).

Other SettingsTab test files (`SettingsTab.persistence.test.tsx`, `SettingsTab.start-minimized.test.tsx`, `SettingsTab.web-link-ux.test.tsx`, `SettingsTab.test.tsx`, `SettingsTab.hyperlinked-index.test.tsx`) were checked and found to be source-inspection-only (`?raw` imports, no `createRoot`/DOM mount) or to mount a different sub-component — unaffected by this change, no fix needed.

---

**Total deviations:** 1 auto-fixed (1 blocking — pre-existing test mocks needed the new binding stubbed, no behavior or assertion change)
**Impact on plan:** No scope creep; a direct, necessary consequence of adding the `EventsOn` import that the plan itself specified.

## Issues Encountered

None beyond the mock-stub deviation above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Both scope items of the M-41 gap closure are now complete: 167-06 (backend instrumentation + proactive authorization + `notification:permission-denied` emission) and 167-07 (frontend subscription + remediation hint).
- The sole remaining open Phase 167 item is the M-41 live-delivery manual re-test on a SIGNED PRODUCTION BUILD (`wails build -tags wailsassets`) across macOS/Windows/Linux, per TESTING.md Category U. On macOS, if authorization is reported denied, the tester should now see the "System Settings > Notifications > AgentHub" hint in Settings → Behavior instead of a silent dead end.
- Run `/gsd-verify-work 167` next to execute the M-41 live re-test.

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED

All modified/created files confirmed present on disk (SettingsTab.notify-permission-hint.test.tsx, SettingsTab.tsx, TESTING.md, this SUMMARY.md); all 3 task commits (2004ffb8, 258195c0, 07d9d9a6) confirmed present in git history.
