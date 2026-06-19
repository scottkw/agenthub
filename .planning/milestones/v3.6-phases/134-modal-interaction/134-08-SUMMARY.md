---
phase: 134-modal-interaction
plan: "08"
subsystem: frontend/hub-modal-wiring
tags: [typescript, react, remote-modal, isRemote-discriminator, behavioral-tests, wr-fixes]
dependency_graph:
  requires:
    - "134-07: RelayClient remote seam + TerminalPanel/HubBriefingModal remote props"
  provides:
    - "isRemote threaded HubPanel → HubModal → HubInteractiveModal + HubBriefingModal (Plan 07 proxy seam activated)"
    - "WR-01: handleCapCancelled resets pendingModalSessionId/pendingSourceRectRef on join-modal dismiss"
    - "WR-02: onRequestRemoteCap guards against overwriting in-flight joinModalForSession"
    - "WR-03: terminalTheme required on HubPanelProps; unsafe ({} as ITheme) cast removed"
    - "WR-04: real per-session fontSize + onFontSizeChange threaded from App through HubModal"
    - "IN-01: relayPort > 0 guard on modal render (mirrors tab-grid guard)"
    - "WR-07: behavioral tests FE-ROUTE-01, CR-03-01 (a/b/c), TAIL-01 (a/b/c), WR-07 (a/b)"
    - "WR-05/WR-06 deferral comments in HubModal.tsx"
  affects:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
    - frontend/src/App.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/Hub/HubBriefingModal.test.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
tech_stack:
  added: []
  patterns:
    - "isRemote computed at render time (same rule as handleCardClick); threaded as remote prop"
    - "capCancelledRef pattern mirrors capAcquiredRef for symmetric cancel notification"
    - "vi.mock('../TerminalPanel') stub allows HubModal to mount in jsdom without xterm"
    - "MockRelayClient with triggerOpen/triggerOutput for deterministic send-path testing"
key_files:
  created: []
  modified:
    - frontend/src/components/Hub/HubPanel.tsx
    - frontend/src/components/Hub/HubModal.tsx
    - frontend/src/components/Hub/HubBriefingModal.tsx
    - frontend/src/App.tsx
    - frontend/src/components/Hub/HubPanel.test.tsx
    - frontend/src/components/Hub/HubBriefingModal.test.tsx
    - frontend/src/components/Hub/HubInteractiveModal.test.tsx
decisions:
  - "DEFAULT_FONT_SIZE defined module-locally in HubPanel.tsx (not imported from App) to avoid cross-file coupling; values stay in sync by convention"
  - "capCancelledRef mirrors capAcquiredRef pattern: both are stored in refs to avoid stale closure issues in handlers"
  - "TerminalPanel mocked in HubPanel behavioral tests (not full xterm mock stack) — single-file mock is simpler and sufficient since the proxy seam test only needs the remote prop to reach TerminalPanel"
  - "WR-01 cancel only fires for hub-modal intent: file-browse dismiss does not need HubPanel reset (no pending state set for that path)"
  - "WR-05/WR-06 explicitly not fixed here — deferred to Phase 135 a11y pass per plan objective and threat register T-134-08-DEFER"
metrics:
  duration: "~8 minutes"
  completed: "2026-06-17"
  tasks: 3
  files_created: 0
  files_modified: 7
---

# Phase 134 Plan 08: isRemote Discriminator + WR-Fixes + Behavioral Tests Summary

**One-liner:** Threaded the `isRemote` discriminator from HubPanel into the modal (activating the Plan 07 proxy seam), fixed WR-01 through WR-04 and IN-01 warning-level issues, added behavioral tests for the remote routing gate / briefing send ordering / remote tail snapshot (FE-ROUTE-01, CR-03-01, TAIL-01), and documented WR-05/WR-06 deferral to Phase 135.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Thread isRemote + fix WR-03/WR-04/IN-01 in HubPanel modal render | 135175ea | HubPanel.tsx, HubModal.tsx, App.tsx, HubPanel.test.tsx |
| 2 | Fix WR-01 (cancel reset) + WR-02 (intent overwrite guard) | 135175ea | HubPanel.tsx, App.tsx (in Task 1 commit — changes were interrelated) |
| 3 (behavioral) | Behavioral tests WR-07 (FE-ROUTE-01, CR-03-01, TAIL-01) | d61add83 | HubPanel.test.tsx, HubBriefingModal.test.tsx, HubInteractiveModal.test.tsx |
| 3 (deferral) | WR-05/WR-06 deferral comments + read-only silent-drop note | bdcf89c1 | HubModal.tsx, HubBriefingModal.tsx |

## What Was Built

### Task 1: isRemote + WR-03/WR-04/IN-01

**isRemote (CR-01 closure):**
- Computed at modal render time in HubPanel: `const isRemote = !!modalState.session.hostname && modalState.session.hostname !== ''`
- Passed as `remote={isRemote}` into HubModal, which forwards it to both `HubBriefingModal` and `HubInteractiveModal` (the seams added in Plan 07 are now activated).

**WR-03 (cast removal):**
- `terminalTheme` on `HubPanelProps` changed from optional (`terminalTheme?: ITheme`) to required (`terminalTheme: ITheme`).
- Unsafe `({} as ITheme)` fallback cast removed from the modal render; `theme={terminalTheme}` is used directly.
- App.tsx already supplies a non-null `terminalTheme` (falls back to a real named theme).

**WR-04 (real fontSize/onFontSizeChange):**
- Added `fontSizes?: Record<string, number>` and `onFontSizeChange?: (sessionId: string, delta: number) => void` to HubPanelProps.
- Modal render now uses `fontSizes?.[modalState.session.id] ?? DEFAULT_FONT_SIZE` (module-local `const DEFAULT_FONT_SIZE = 14`).
- `onFontSizeChange` threaded from HubPanel → HubModal → HubInteractiveModal (curried to `(delta) => onFontSizeChange(modalSession.id, delta)`).
- IN-02 dead-surface: `onFontSizeChange` is now wired; the `?? (() => {})` no-op default in HubInteractiveModal is still the prop-level fallback but is no longer the only path.
- App.tsx passes `fontSizes={fontSizes}` and `onFontSizeChange={handleFontSizeChange}` to HubPanel.

**IN-01 (relayPort > 0 guard):**
- Modal render guard changed from `relayPort !== undefined` to `relayPort !== undefined && relayPort > 0`.
- Mirrors the tab-grid guard at App.tsx:1535 — prevents building `ws://127.0.0.1:0/...` on a transient 0.

### Task 2: WR-01 + WR-02 (in same commit as Task 1)

**WR-01 (cancel reset):**
- Added `handleCapCancelled` callback in HubPanel: resets `pendingModalSessionId` and `pendingSourceRectRef`.
- Added `onRegisterCapCancelled` prop to HubPanelProps (mirrors `onRegisterCapAcquired`).
- Added `capCancelledRef` in App.tsx; registered via `onRegisterCapCancelled={(fn) => { capCancelledRef.current = fn }}`.
- RemoteJoinCodeModal `onClose` now checks `joinModalForSession?.intent === 'hub-modal'` and invokes `capCancelledRef.current?.()` before `setJoinModalForSession(null)`.

**WR-02 (intent overwrite guard):**
- `onRequestRemoteCap` in App.tsx now guards: `if (joinModalForSession) return` before `setJoinModalForSession({...intent: 'hub-modal'})`.
- Prevents a hub-modal cap request from silently overwriting an in-flight file-browse join modal.
- Comment added citing WR-02.

### Task 3: Behavioral Tests (WR-07)

**FE-ROUTE-01 (HubPanel.test.tsx):**
- Added `vi.mock('../TerminalPanel')` stub so HubModal can mount in jsdom without xterm.
- `renderPanel` helper extended with `remoteCapsCached` and `onRequestRemoteCap` overrides.
- `FE-ROUTE-01a`: renders a remote-without-cap card; click calls `onRequestRemoteCap` once; `.hub-modal-overlay` is NOT in DOM.
- `FE-ROUTE-01b`: renders a local card with `relayPort=51234`; click does NOT call `onRequestRemoteCap`; `.hub-modal-overlay` IS in DOM.

**CR-03-01 (HubBriefingModal.test.tsx):**
- `vi.mock('../../lib/relayClient')` installs a `MockRelayClient` that exposes `triggerOpen()` and `triggerOutput()`.
- `CR-03-01a`: typing text + clicking Send → `triggerOpen()` → `sendInput` called once with `text\n` → `close` called → `onClose` called.
- `CR-03-01b`: typing text + clicking Send → advancing 5000ms (timeout) → `close` called, `sendInput` NOT called, `onClose` NOT called.
- `CR-03-01c`: typing text + clicking Send → advancing 5000ms → `triggerOpen()` → `sendInput` still NOT called (settled guard).

**TAIL-01 (HubBriefingModal.test.tsx):**
- `TAIL-01a`: `remote=true` → `triggerOpen()` + `triggerOutput(encoder.encode('line-one\nline-two\n'))` → advance 500ms → `<pre>` contains `line-one` and `line-two`; "No recent output available" absent.
- `TAIL-01b`: `remote=true` → `GetSessionTailLines` NOT called.
- `TAIL-01c`: `remote=false` → `GetSessionTailLines` called with `(id, 20)`; no RelayClient created for tail.

**WR-07 behavioral (HubInteractiveModal.test.tsx):**
- `WR-07a`: `remote=true` → mocked TerminalPanel receives `remote=true`.
- `WR-07b`: `remote=false` → mocked TerminalPanel receives falsy remote.

**Deferral comments:**
- `HubModal.tsx`: WR-05 (broad `stopImmediatePropagation`) and WR-06 (focus trap) explicitly documented as deferred to Phase 135 a11y pass.
- `HubBriefingModal.tsx`: read-only remote cap silent-drop documented at the Send button; non-color indicator deferred to Phase 135 (colorblind-safe is release-blocking per user constraint).

## Verification

| Check | Result |
|-------|--------|
| `pnpm exec tsc --noEmit` | Clean (no errors) |
| `pnpm exec vitest run HubPanel HubBriefingModal HubInteractiveModal` | 65/65 pass |
| `pnpm exec vitest run` (full suite) | 1699/1699 pass, 104/104 test files |
| `grep -c "({} as ITheme)" HubPanel.tsx` | 0 ✓ |
| `grep -c "fontSize={14}" HubPanel.tsx` | 0 ✓ |
| `grep -n "relayPort > 0" HubPanel.tsx` | line 480 ✓ |
| `grep -n "remote" HubModal.tsx` | lines 47-48, 185-203 ✓ |
| `grep -n "handleCapCancelled" HubPanel.tsx` | lines 167, 391-399 ✓ |
| `grep -c "?raw" HubBriefingModal.test.tsx` | 1 (import only; send/tail tests use MockRelayClient) ✓ |

## Deviations from Plan

**1. [Rule 3 - Interleaved] Task 1 and Task 2 committed together**
- **Found during:** Task 2 implementation.
- **Issue:** WR-01 (cap-cancel callback) requires new props on HubPanelProps (same file as Task 1 changes); WR-02 requires changes to App.tsx's `onRequestRemoteCap` prop on the HubPanel render (same location as Task 1's new props). Splitting into two atomic commits would have required partial editing of the same JSX attribute block.
- **Fix:** Both tasks committed in 135175ea. The commit message documents both tasks. No functionality is omitted.
- **Files modified:** HubPanel.tsx, App.tsx
- **Commit:** 135175ea

## Threat Model Adherence

- T-134-08-01 (WR-02 intent overwrite): mitigated — `if (joinModalForSession) return` guard in onRequestRemoteCap.
- T-134-08-02 (WR-01 stranded pending): mitigated — `handleCapCancelled` + `capCancelledRef` + RemoteJoinCodeModal onClose integration.
- T-134-08-03 (CR-01 regression / remote attaches local relay): mitigated — `isRemote` threaded to Plan 07 seam; FE-ROUTE-01b behavioral test guards the gate.
- T-134-08-04 (IN-04 read-only remote distinction): accepted — aria-label origin + colorblind-safe non-color badge deferred to Phase 135; deferral documented at call site.
- T-134-08-DEFER (WR-05/WR-06): transferred — in-code deferral comments added; no fix here.
- T-134-08-SC (npm installs): N/A — no new dependencies.

## Known Stubs

None — all WR-01 through WR-04 and IN-01 fixes are fully wired. The read-only remote cap silent-drop is documented at the call site with an explicit Phase 135 deferral (non-color indicator is release-blocking per user constraint).

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes introduced.

## Self-Check: PASSED
