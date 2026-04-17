---
phase: 82-minimize-to-tray
verified: 2026-04-17T14:00:00Z
status: human_needed
score: 9/9
overrides_applied: 0
human_verification:
  - test: "Launch app with start-minimized enabled, verify only tray icon appears"
    expected: "Main window does not appear on launch; tray icon is present"
    why_human: "domReady branch requires live Wails runtime — cannot verify programmatically that runtime.WindowShow is skipped"
  - test: "Toggle off, restart, confirm window appears normally"
    expected: "After disabling start-minimized and restarting, the main window shows on launch"
    why_human: "Persistence across restarts requires physical app restart cycle with Wails runtime"
  - test: "Toggle shows loading state during save (opacity 0.6, pointer-events none)"
    expected: "Toggle dims and becomes unclickable while SetStartMinimized is in-flight"
    why_human: "Requires interactive UI testing to observe transient loading state"
  - test: "Toggle error state: Simulate SetStartMinimized failure, verify error message and state revert"
    expected: "Error message appears below description; toggle reverts to previous value"
    why_human: "Requires ability to inject a daemon error during toggle interaction"
---

# Phase 82: Minimize to Tray — Verification Report

**Phase Goal:** Users can configure the app to start hidden in the system tray, with the preference persisting across restarts
**Verified:** 2026-04-17T14:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Settings contains a clearly labeled toggle for "Start minimized to system tray" | VERIFIED | `SettingsTab.tsx` line 289: `<span className="settings-panel__toggle-label">Start minimized to system tray</span>` inside a `Behavior` section h3 that is the first section in `.settings-panel__body` |
| 2 | When the toggle is enabled and the app is launched, the main window is not shown and only the tray icon is visible | VERIFIED (code) / NEEDS HUMAN (runtime) | `app.go` lines 78-89: `domReady` reads `a.client.GetStartMinimized()`; if true, `runtime.WindowShow` and `setDockVisible` are skipped entirely. Code logic is correct; requires live Wails runtime to confirm visually. |
| 3 | The minimize-to-tray preference is saved and survives app restarts | VERIFIED (code) / NEEDS HUMAN (runtime) | `engine.go` `SetStartMinimized` → `saveSettingsToDisk` writes to `settings.json`; `loadSettingsFromDisk` reads it on `NewSessionEngine`. `TestStartMinimizedPersistence` and `TestStartMinimizedWithoutCLIPaths` pass. Physical restart cycle needs human confirmation. |

**Score:** 9/9 truths verified (3 roadmap SCs + 6 plan-level truths all pass code verification; 4 items require human runtime confirmation)

### Plan 01 Must-Have Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | When start-minimized is enabled and app launches, the window does not show — only the tray icon is visible | VERIFIED (code) | `domReady` gates `runtime.WindowShow` and `setDockVisible` on `!startMinimized` |
| 2 | When start-minimized is disabled (default), the app launches and shows the window normally | VERIFIED | `startMinimized` defaults to `false`; `domReady` calls `WindowShow` when daemon is unreachable or preference is false |
| 3 | The start-minimized preference survives app restarts — saved to settings.json and loaded on next launch | VERIFIED | `saveSettingsToDisk` / `loadSettingsFromDisk` tested by `TestStartMinimizedPersistence` — all 4 assertions pass |
| 4 | When the daemon is unreachable at startup, the window shows as a safe fallback | VERIFIED | `domReady`: `a.client == nil` → `startMinimized = false` → `WindowShow` called |

### Plan 02 Must-Have Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Settings tab has a 'Behavior' section as the first section (before Appearance) | VERIFIED | `SettingsTab.tsx` line 278: `<h3>Behavior</h3>` appears before line 306: `<h3>Appearance</h3>` as the first child of `.settings-panel__body` |
| 2 | Behavior section contains a toggle labeled 'Start minimized to system tray' | VERIFIED | Line 289: label text confirmed; `htmlFor="startMinimized"` and `id="startMinimized"` wired |
| 3 | Toggle reflects the persisted value on mount (no flash from off to on) | VERIFIED (code) | `toggleLoaded` gate at line 280: label only renders after `GetStartMinimized()` resolves |
| 4 | Clicking the toggle calls SetStartMinimized, updates state only on success (non-optimistic) | VERIFIED | `handleToggleMinimized` lines 238-250: `await SetStartMinimized(next)` at line 243 before `setStartMinimized(next)` at line 244 |
| 5 | Toggle shows loading state (opacity 0.6, pointer-events none) while saving | VERIFIED (code) | Line 284: `style={toggleSaving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}` |
| 6 | Toggle shows error message below description if save fails | VERIFIED | Line 302: `{toggleError && <p className="settings-panel__error">{toggleError}</p>}` |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/daemon/engine.go` | `startMinimized bool` field, `daemonSettings.StartMinimized`, `GetStartMinimized`, `SetStartMinimized`, fixed `loadSettingsFromDisk` | VERIFIED | All 5 elements confirmed at lines 35, 67, 96, 100, 311, 318 |
| `internal/daemon/api.go` | `GET /settings/start-minimized` and `PATCH /settings/start-minimized` routes + handlers | VERIFIED | Routes at lines 53-54; handlers at lines 343-357 |
| `internal/daemon/client.go` | `GetStartMinimized() (bool, error)` and `SetStartMinimized(val bool) error` | VERIFIED | Lines 102-114 confirmed |
| `app.go` | `GetStartMinimized() bool`, `SetStartMinimized(val bool) error`, conditional `domReady` | VERIFIED | Lines 332-351 (bindings), lines 78-89 (domReady gate) |
| `internal/daemon/engine_settings_test.go` | `TestStartMinimizedPersistence`, `TestStartMinimizedWithoutCLIPaths` | VERIFIED | Lines 100-164; all tests pass |
| `frontend/src/components/SettingsTab.tsx` | Behavior section, toggle, state management, useEffect, handler | VERIFIED | Imports at lines 15-16; state at lines 84-87; useEffect at lines 139-144; handler at lines 238-250; JSX at lines 277-303 |
| `frontend/src/style.css` | 7 toggle CSS rules | VERIFIED | Lines 584-632; all rules present with spec-exact values |
| `frontend/src/wailsjs/go/main/App.d.ts` | `GetStartMinimized(): Promise<boolean>`, `SetStartMinimized(val: boolean): Promise<void>` | VERIFIED | Lines 110-111 |
| `frontend/src/wailsjs/go/main/App.js` | `GetStartMinimized` and `SetStartMinimized` exports | VERIFIED | Lines 68-69 confirmed |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `app.go domReady` | `internal/daemon/client.go GetStartMinimized` | `a.client.GetStartMinimized()` | WIRED | Line 81: `if val, err := a.client.GetStartMinimized(); err == nil {` |
| `internal/daemon/api.go` | `internal/daemon/engine.go` | `a.engine.GetStartMinimized()` / `a.engine.SetStartMinimized()` | WIRED | Line 344: `a.engine.GetStartMinimized()`, line 355: `a.engine.SetStartMinimized(req.StartMinimized)` |
| `internal/daemon/client.go` | `internal/daemon/api.go` | HTTP GET/PATCH `/settings/start-minimized` | WIRED | Client lines 103-113 call the exact paths registered in api.go lines 53-54 |
| `SettingsTab.tsx` | `wailsjs/go/main/App.ts` | `import { GetStartMinimized, SetStartMinimized }` | WIRED | Lines 15-16: both functions imported; called at lines 140 and 243 |
| `SettingsTab.tsx` | `style.css` | `className settings-panel__toggle-*` | WIRED | Lines 282, 286, 287, 289, 295 use toggle CSS classes defined in style.css |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `SettingsTab.tsx` toggle | `startMinimized` | `GetStartMinimized()` → daemon HTTP GET `/settings/start-minimized` → `engine.GetStartMinimized()` → `e.startMinimized` (loaded from `settings.json`) | Yes — engine field populated from disk by `loadSettingsFromDisk` at engine init | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Go tests pass (persistence round-trip) | `go test ./internal/daemon/... -run TestStartMinimized` | All 2 tests PASS | PASS |
| Go build succeeds | `go build ./...` | Exit 0, no errors | PASS |
| TypeScript bindings contain exported functions | `grep "GetStartMinimized" frontend/src/wailsjs/go/main/App.d.ts` | Found at line 110 | PASS |
| App.js exports wired | `grep "GetStartMinimized" frontend/src/wailsjs/go/main/App.js` | Found at line 68 | PASS |
| domReady window-hide path (requires runtime) | Cannot test without live Wails runtime | N/A | SKIP |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| TRAY-01 | Plan 02 | Settings includes a toggle for "Start minimized to system tray" | SATISFIED | `SettingsTab.tsx` Behavior section with labeled toggle, confirmed at line 278-303 |
| TRAY-02 | Plan 01 | When enabled, launching AgentHub opens with window hidden and only tray icon visible | SATISFIED (code) | `domReady` gates `WindowShow`; human runtime test required to confirm |
| TRAY-03 | Plan 01 | Minimize-to-tray preference persists across app restarts | SATISFIED (code) | `settings.json` persistence tested by `TestStartMinimizedPersistence`; physical restart requires human |

All 3 TRAY requirements mapped to Phase 82 in REQUIREMENTS.md are accounted for. No orphaned requirements.

### Anti-Patterns Found

No blockers or stubs detected in the modified files.

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | — |

Specific checks performed:
- `SettingsTab.tsx`: No TODO/FIXME comments in new code sections. `startMinimized` state is populated from real daemon call, not hardcoded. `handleToggleMinimized` has real await call, not console.log-only.
- `engine.go`: `GetStartMinimized` and `SetStartMinimized` operate on real field with proper lock patterns. `saveSettingsToDisk` writes real JSON.
- `app.go`: `domReady` gate uses a real conditional on the persisted value, not a placeholder.
- `style.css`: All 7 CSS rule blocks present with spec-exact values.

### Human Verification Required

**Note:** All automated checks pass. The 4 items below require a live Wails runtime — they cannot be verified by static analysis or unit tests.

#### 1. Start Minimized — Window Hidden on Launch

**Test:** Enable "Start minimized to system tray" in Settings, quit the app, relaunch it.
**Expected:** The main window does not appear; only the tray icon is visible in the menu bar/system tray.
**Why human:** `domReady` calls `runtime.WindowShow` conditionally — this branch requires Wails runtime execution and a physical display to observe.

#### 2. Start Minimized Off — Window Appears Normally

**Test:** With start-minimized disabled (toggle off), quit and relaunch the app.
**Expected:** The main window opens normally as before.
**Why human:** Same as above — requires live runtime to confirm the non-minimized path also works.

#### 3. Persistence Across Restarts

**Test:** Enable start-minimized, quit, relaunch (window hidden). Then open window via tray, disable toggle, quit, relaunch.
**Expected:** First relaunch: window hidden. Second relaunch: window shows.
**Why human:** Physical restart cycle needed to confirm `settings.json` round-trip works end-to-end with the full daemon startup sequence.

#### 4. Toggle Loading and Error States

**Test (loading):** Click the toggle and observe UI during the save in-flight.
**Expected:** Toggle dims to opacity 0.6 and becomes unclickable while `SetStartMinimized` is in-flight.

**Test (error):** With daemon stopped or via network fault, click the toggle.
**Expected:** Error message "Could not save preference — ..." appears below the description; toggle reverts to its previous state.
**Why human:** Transient states require interactive UI testing; error state requires injecting a daemon fault.

### Gaps Summary

No gaps. All code-verifiable must-haves are satisfied:
- Backend persistence layer: engine struct, settings.json serialization, load/save functions — all implemented and unit-tested.
- Daemon API: GET/PATCH `/settings/start-minimized` routes registered and handlers wired to engine.
- Client methods: `GetStartMinimized` and `SetStartMinimized` in `DaemonClient` with proper `doJSON` pattern.
- Wails bindings: `app.go` exposes both methods; TypeScript stubs (`App.d.ts`, `App.js`) export them.
- `domReady` gate: conditionally skips `WindowShow`/`setDockVisible` when preference is enabled; safe fallback when daemon unreachable.
- Frontend toggle: Behavior section first in Settings, non-optimistic save, flash-prevention gate, loading/error state management.
- CSS: All 7 toggle rules present with spec-exact values.
- Tests: `TestStartMinimizedPersistence` and `TestStartMinimizedWithoutCLIPaths` pass.
- Build: `go build ./...` succeeds.

Pending: 4 human verification items (runtime behavior — window visibility on launch, persistence across physical restarts, loading/error UI states).

---

_Verified: 2026-04-17T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
