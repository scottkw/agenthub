---
phase: 150-shell-sharing-warning-toggle
plan: 03
subsystem: frontend share-warning
tags: [react, typescript, vitest, hub, share-modal, shell-sharing, warning, tdd]

# Dependency graph
requires:
  - phase: 150-01
    provides: GetShellWebShareWarningEnabled / GetShellWebShareWarned Wails bindings
  - phase: 150-02
    provides: SettingsTab onShellWarnEnabledChange? optional prop contract
provides:
  - warningEnabled gate in App.tsx handleToggleWeb (StatusBar path, D-02/D-04)
  - GetShellWebShareWarningEnabled mount hydration + re-arm re-sync (D-03)
  - SessionShareModal shell interception via SHELL_CLIS gate + ShellWebShareBanner (D-09)
  - Single shellWebShareWarned authority — no forked state (D-10)
  - HubPanel prop threading (shell-warn props App.tsx → HubPanel → SessionShareModal)
  - TESTING.md SET-01 §2/§4/§5 registration (M-16/M-17 manual entries)
affects:
  - Both share surfaces respect same warningEnabled && !warned gate

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cross-surface shared state: single shellWebShareWarned in App.tsx threaded via props/callbacks to two surfaces (StatusBar + Share modal) — no forked state"
    - "Banner reuse pattern: ShellWebShareBanner imported into SessionShareModal for parity without copy-paste (D-10)"
    - "pendingShellShare intercept: local modal state for 'banner shown' — cleared by confirm or cancel"
    - "Re-arm re-sync: onShellWarnEnabledChange callback in App.tsx re-fetches GetShellWebShareWarned when enabled=true"

key-files:
  modified:
    - frontend/src/App.tsx
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/SessionShareModal.tsx
    - frontend/src/components/__tests__/App.shellWebShare.test.tsx
    - frontend/src/components/__tests__/SessionShareModal.test.tsx
    - TESTING.md

key-decisions:
  - "Single warned authority: shellWebShareWarned lives only in App.tsx and is threaded as props to both surfaces — SessionShareModal uses props, not local state. Prevents the double-banner race (T-150-10 mitigation, pitfall 4)"
  - "Re-arm re-sync: onShellWarnEnabledChange callback (optional prop already on SettingsTab from Plan 02) re-fetches GetShellWebShareWarned from daemon when enabled=true, because SetShellWebShareWarningEnabled(true) atomically reset shellWebShareWarned=false on the daemon side (D-03 pitfall 2)"
  - "Banner reuse over inline JSX: import ShellWebShareBanner into SessionShareModal rather than copy-pasting copy, for D-10 parity and single point of truth for warning copy"
  - "Test window sizing: source-inspection tests for App.tsx use char-index slices — increased window to 1200 (onShellWarnEnabledChange) and 2500 (HubPanel block) to cover deeply-nested callback bodies"

# Metrics
duration: 8min
completed: 2026-06-23
---

# Phase 150 Plan 03: Share Modal Warning Interception + App.tsx Gate Summary

**Shell web-share warning wired to both share surfaces via single App.tsx `shellWebShareWarned` authority — StatusBar path gets `warningEnabled` gate; Hub Share modal gets full SHELL_CLIS interception + ShellWebShareBanner reuse; re-arm re-syncs frontend warned state.**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-06-23T14:31:17Z
- **Completed:** 2026-06-23T14:39:50Z
- **Tasks:** 3 (Task 1: App.tsx warningEnabled + threading; Task 2: SessionShareModal + HubPanel + tests; Task 3: TESTING.md)
- **Files modified:** 6

## Accomplishments

- Added `shellWebShareWarningEnabled` state to App.tsx (default true, D-08); hydrated from `GetShellWebShareWarningEnabled()` on mount alongside existing `GetShellWebShareWarned()` hydration
- Updated `handleToggleWeb` gate: `SHELL_CLIS.has(tab.cli) && shellWebShareWarningEnabled && !shellWebShareWarned` (Success Criterion 2/4 — StatusBar path respects warningEnabled)
- Added `onShellWarnEnabledChange` callback to SettingsTab render (optional prop wired from Plan 02); callback re-fetches `GetShellWebShareWarned()` when enabled=true to sync the daemon's re-arm (D-03, pitfall 2)
- Threaded `shellWebShareWarned`, `shellWebShareWarningEnabled`, `handleShellWebShareConfirm`, `handleShellWebShareCancel` into HubPanel render (single authority, no fork — pitfall 4 mitigation)
- Added shell-warn props to `HubPanelProps`; destructured and forwarded into `SessionShareModal` render
- Added `cli: string` to `ShareSession` interface in SessionShareModal (T-150-11 mitigation — pitfall 3)
- Added module-level `SHELL_CLIS` Set to SessionShareModal matching App.tsx and engine.go
- Added `pendingShellShare` state and gate in `handleShareToggle`: `if (next && SHELL_CLIS.has(session.cli) && shellWebShareWarningEnabled && !shellWebShareWarned) { setPendingShellShare(true); return }` (D-09, Success Criterion 4)
- Added `ShellWebShareBanner` render in modal body when `pendingShellShare` (D-10 reuse, no copy-paste)
- 11 source-inspection tests in `App.shellWebShare.test.tsx` for warningEnabled gate, re-arm, HubPanel threading; 6 DOM-render tests in `SessionShareModal.test.tsx` for shell/non-shell/disabled/warned/cancel/confirm cases
- TESTING.md §2 note updated; §4 two new SET-01 rows; §5 two new M-16/M-17 manual entries

## Task Commits

Each task was committed atomically:

1. **TDD RED: App.shellWebShare — 11 failing tests** - `ddd028fd` (test)
2. **Task 1 GREEN: App.tsx warningEnabled state + gate + re-arm + prop threading** - `3e322db7` (feat)
3. **TDD RED: SessionShareModal — 2 failing interception tests** - `2860ca24` (test)
4. **Task 2 GREEN: SessionShareModal + HubPanel wiring** - `e5b12446` (feat)
5. **Task 3: TESTING.md SET-01 registration** - `d3eb81d3` (feat)

## Files Created/Modified

- `frontend/src/App.tsx` — imports `GetShellWebShareWarningEnabled`; `shellWebShareWarningEnabled` state (default true); mount hydration; updated `handleToggleWeb` gate with `&& shellWebShareWarningEnabled`; dep array updated; `onShellWarnEnabledChange` callback to SettingsTab with re-arm re-fetch; shell-warn props threaded to HubPanel
- `frontend/src/components/Hub/HubPanel.tsx` — shell-warn props added to `HubPanelProps` (4 props); destructured; forwarded to `SessionShareModal` render
- `frontend/src/components/Hub/SessionShareModal.tsx` — imports `ShellWebShareBanner`; `SHELL_CLIS` Set; `cli` added to `ShareSession`; shell-warn props in `SessionShareModalProps` and function signature; `pendingShellShare` state; gate in `handleShareToggle`; `ShellWebShareBanner` render in modal body
- `frontend/src/components/__tests__/App.shellWebShare.test.tsx` — 11 new SET-01 tests (warningEnabled state, hydration, gate, dep array, onShellWarnEnabledChange threading, HubPanel prop assertion)
- `frontend/src/components/__tests__/SessionShareModal.test.tsx` — 6 new SET-01 tests (shell+enabled+!warned→banner; disabled→no banner; non-shell→no banner; already-warned→no banner; cancel; confirm); `cli` added to `makeSession`; `renderModal` passes shell-warn props; `ToggleWebServing` imported for assertion
- `TESTING.md` — §2 Phase 150-03 note; §4 two new SET-01 rows for SessionShareModal.test.tsx and App.shellWebShare.test.tsx; §5 M-16 + M-17 live PTY + restart-persistence manual checklist entries

## Decisions Made

- Single `shellWebShareWarned` authority in App.tsx — SessionShareModal receives warned state as props/callbacks, never via local state. This prevents the double-banner scenario (T-150-10) where confirming on one surface wouldn't suppress the other.
- Re-arm re-sync via `onShellWarnEnabledChange` callback: when the user re-enables the warning, App.tsx re-fetches `GetShellWebShareWarned()` because the daemon atomically reset `shellWebShareWarned=false` in `SetShellWebShareWarningEnabled(true)`. Without this, frontend state would still show `warned=true` and the re-armed banner would never fire.
- Banner reuse: imported `ShellWebShareBanner` into `SessionShareModal` rather than copying the JSX. Single point of truth for warning copy and behavior.
- Test window sizes increased: source-inspection tests use char-index slices over App.tsx — `onShellWarnEnabledChange` callback has large comments (needed 1200-char window); HubPanel block with new props appears ~1992 chars from `<HubPanel` (needed 2500-char window). Adjusted in the RED→GREEN transition to match actual file layout.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adjusted source-inspection test window sizes**
- **Found during:** Task 1 GREEN (App.shellWebShare tests passing GREEN after App.tsx changes)
- **Issue:** The RED tests used 600-char and 1500-char windows that cut off before the `setShellWebShareWarningEnabled` call (in the `onShellWarnEnabledChange` callback with large comments) and the HubPanel new props (which appear ~1992 chars into the `<HubPanel` block). Three tests remained red.
- **Fix:** Increased test window from 600→1200 (callback search) and 1500→2500 (HubPanel block search). Adjusted in the same commit as the GREEN implementation (test adjustments are part of the RED→GREEN TDD cycle per the protocol).
- **Files modified:** `frontend/src/components/__tests__/App.shellWebShare.test.tsx`
- **Committed in:** `3e322db7` (feat)

### Out-of-scope items

None discovered.

## Known Stubs

None — all state is fully wired:
- `shellWebShareWarningEnabled` loads from `GetShellWebShareWarningEnabled()` on mount (not a hardcoded constant)
- `shellWebShareWarned` threads from existing App.tsx authority (Plan 01)
- `ShellWebShareBanner` renders with live callbacks to App.tsx's `handleShellWebShareConfirm`

## Threat Flags

No new threat surface beyond what the plan's threat model documents:
- T-150-08 mitigated: `handleShareToggle` now gates shell ON-toggles with `shellWebShareWarningEnabled && !shellWebShareWarned`
- T-150-09 mitigated: `onShellWarnEnabledChange` re-fetches `GetShellWebShareWarned()` on re-arm so the re-armed daemon state is synced to frontend
- T-150-10 mitigated: single `shellWebShareWarned` authority in App.tsx (no fork)
- T-150-11 mitigated: `cli` added to `ShareSession`; `SHELL_CLIS.has(session.cli)` is the canonical shell test

## Self-Check

### Modified files contain required strings:
- [x] `App.tsx` contains `shellWebShareWarningEnabled` — confirmed
- [x] `App.tsx` contains `GetShellWebShareWarningEnabled` — confirmed
- [x] `App.tsx` contains `shellWebShareWarningEnabled &&` before `!shellWebShareWarned` in handleToggleWeb — confirmed
- [x] `App.tsx` contains `onShellWarnEnabledChange=` — confirmed
- [x] `SessionShareModal.tsx` contains `cli` in ShareSession interface — confirmed
- [x] `SessionShareModal.tsx` contains `SHELL_CLIS` — confirmed
- [x] `SessionShareModal.tsx` contains `ShellWebShareBanner` — confirmed
- [x] `SessionShareModal.tsx` contains `shellWebShareWarningEnabled && !shellWebShareWarned` — confirmed
- [x] `HubPanel.tsx` contains `shellWebShareWarningEnabled` — confirmed
- [x] `TESTING.md` contains `engine_shell_warn_test.go` — confirmed (pre-existing from Plan 01)
- [x] `TESTING.md` contains `SettingsTab.shell-warn-toggle.test.tsx` — confirmed (pre-existing from Plan 02)
- [x] `TESTING.md` contains `SessionShareModal.test.tsx` (SET-01 row) — confirmed (new)
- [x] `TESTING.md` contains `M-16` — confirmed (new)
- [x] `TESTING.md` contains `M-17` — confirmed (new)
- [x] `TESTING.md` contains `SET-01` — confirmed

### Commits exist:
- [x] ddd028fd — TDD RED App.shellWebShare
- [x] 3e322db7 — Task 1 GREEN
- [x] 2860ca24 — TDD RED SessionShareModal
- [x] e5b12446 — Task 2 GREEN
- [x] d3eb81d3 — Task 3 TESTING.md

### Verification results:
- [x] `pnpm tsc --noEmit` exits 0 — confirmed
- [x] `pnpm vitest run SessionShareModal App.shellWebShare` — 42 tests pass, 0 fail
- [x] `pnpm vitest run SettingsTab.shell-warn-toggle` — 19 tests pass, 0 fail
- [x] `bash tests/check-traceability-paths.sh` exits 0 — confirmed

## Self-Check: PASSED

---
*Phase: 150-shell-sharing-warning-toggle*
*Completed: 2026-06-23*
