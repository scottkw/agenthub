---
phase: 98-progress-addon-p2-cuttable
plan: "02"
subsystem: progress-addon
tags: [progress, helpers, tray, wails-rpc, wave-1, go, typescript]
dependency_graph:
  requires:
    - "98-01 (wave-0 foundation: stub aggregateProgress.ts + RED test scaffolds + 4 tray PNGs)"
  provides:
    - "aggregateProgress mean-bucket implementation (all 9 vitest cases GREEN)"
    - "(*App).SetTrayProgress(quartile int) error Wails RPC"
    - "lastTrayQuartile int struct field (initialized to -1)"
    - "trayIconBytesForState helper in tray.go, tray_linux.go, tray_windows.go (3 verbatim copies)"
    - "4 //go:embed quartile PNG directives in all 3 platform tray files (12 total)"
    - "Windows: 4 pre-cached HICON handles + cleanup in cleanupTray"
    - "App.d.ts + App.js SetTrayProgress Wails RPC stub"
    - "app_set_tray_progress_test.go: 3 unit tests GREEN"
  affects:
    - "app.go (App struct + NewApp + SetTrayProgress method)"
    - "tray.go (darwin: embeds + helper + updateTray refactor)"
    - "tray_linux.go (linux: embeds + helper + updateTray refactor)"
    - "tray_windows.go (windows: embeds + struct fields + initTray + updateTray + cleanupTray)"
    - "frontend/src/wailsjs/go/main/App.d.ts"
    - "frontend/src/wailsjs/go/main/App.js"
tech_stack:
  added: []
  patterns:
    - "Function injection (refreshTrayStateFunc) mirroring saveFileDialogFunc and serviceControlFunc patterns"
    - "3-way verbatim helper duplication (trayIconBytesForState) mirrors per-platform updateTray duplication pattern"
    - "Pre-cached HICON handles at initTray time (Windows) to avoid runtime PNG decode per tray update"
    - "Wails binding hand-edit per Phase 92 PLUG-03 / Phase 97 SER-05 precedent"
key_files:
  created:
    - "app_set_tray_progress_test.go"
  modified:
    - "frontend/src/lib/aggregateProgress.ts (stub body replaced with mean-bucket implementation)"
    - "app.go (lastTrayQuartile field + refreshTrayStateFunc injection + SetTrayProgress method)"
    - "tray.go (4 //go:embed + trayIconBytesForState helper + updateTray refactor)"
    - "tray_linux.go (4 //go:embed + trayIconBytesForState helper + updateTray refactor)"
    - "tray_windows.go (4 //go:embed + 4 HICON fields + initTray pre-creation + updateTray refactor + cleanupTray disposal)"
    - "frontend/src/wailsjs/go/main/App.d.ts"
    - "frontend/src/wailsjs/go/main/App.js"
decisions:
  - "bounds check BEFORE idempotency: SetTrayProgress returns error for out-of-range values even when lastTrayQuartile == quartile would be false; this matches RESEARCH Pattern 6 — invalid input is rejected at the gate, not silently passed"
  - "refreshTrayStateFunc injection added to App (similar to saveFileDialogFunc) to enable unit tests without cgo/Win32/D-Bus invocation"
  - "trayIconBytesForState duplicated in all 3 platform files (not a shared tray_common.go) because each file's //go:build tag excludes the byte slices from other platforms"
  - "Windows: HICON handles pre-created at initTray time (not at updateTray time) for consistency with existing hIcon/hIconErr pattern and to avoid repeated PNG decode on every tray update"
  - "Windows HICON handles destroyed in cleanupTray alongside existing hIcon/hIconErr cleanup"
metrics:
  duration: "~25 minutes"
  completed_date: "2026-05-08"
  tasks: 2
  files_created: 1
  files_modified: 7
  commits: 2
---

# Phase 98 Plan 02: Wave 1 Helpers and Tray Plumbing Summary

**One-liner:** Implemented aggregateProgress mean-bucket helper (9 vitest GREEN), SetTrayProgress Wails RPC with idempotency + bounds + error-precedence, cross-platform tray byte selector in 3 verbatim platform-file copies, and Wails bindings — all compile gates green, cuttability invariant intact.

## Task 1: aggregateProgress Implementation

The Wave 0 stub (always returns 0) was replaced with the verbatim Pattern 3 mean-bucket implementation from RESEARCH. The function:

1. Collects `value` from all registry entries with `state === 1` (set state only)
2. If no active entries: returns `0` (no active progress)
3. Computes the arithmetic mean
4. Buckets into quartile [0..4]: `mean <= 0 → 0`, `mean <= 25 → 1`, `mean <= 50 → 2`, `mean <= 75 → 3`, else `→ 4`

All 9 Wave 0 RED vitest cases are now GREEN:
- Empty registry → 0
- All state:0 cleared → 0
- Single entry 5% → 1
- mean(50, 75) = 62.5 → 3
- value 100 → 4
- Boundary 25 → 1, 50 → 2, 75 → 3
- Ignores state:0 entries from mean (mixed state:1 + state:0 → only state:1 contributes)

## Task 2: SetTrayProgress + Cross-Platform Tray Selector

### aggregateProgress shape decisions

The implementation uses mean (not median, not max). This is the Locked Decision from RESEARCH. The bucket boundaries are inclusive of the upper bound: `mean <= 25` includes exactly 25, placing it in quartile 1 (25% glyph).

### SetTrayProgress shape: bounds check first, then idempotency

The order is:
```go
if !a.trayInit { return nil }          // 1. tray not yet initialized — silent no-op
if quartile < 0 || quartile > 4 { return fmt.Errorf(...) }  // 2. bounds check
if a.lastTrayQuartile == quartile { return nil }             // 3. idempotency
a.lastTrayQuartile = quartile          // 4. mutate state
a.refreshTrayState()                   // 5. propagate to platform tray
```

This matches RESEARCH Pattern 6 verbatim. Bounds check runs before idempotency because invalid input should always be rejected, even if by coincidence `lastTrayQuartile` equals the invalid value.

### Cross-platform tray byte selector placement decision

`trayIconBytesForState` is defined as identical verbatim copies in all three platform files:
- `tray.go` (`//go:build darwin`)
- `tray_linux.go` (`//go:build linux`)
- `tray_windows.go` (`//go:build windows`)

This mirrors the existing per-platform `updateTray` duplication. A single shared `tray_common.go` is not viable because the `trayIconProgress{25,50,75,100}Bytes` byte slices are declared via `//go:embed` in each platform file — when building for Linux, `tray.go` is excluded and its `trayIconProgress*` symbols don't exist. A shared helper referencing those symbols would fail to compile on any non-darwin target.

### trayIconBytesForState is a method on *App (reads a.lastTrayQuartile)

The helper signature is `func (a *App) trayIconBytesForState(connected bool) []byte`. It reads `a.lastTrayQuartile` to determine which progress glyph to serve. Error precedence (Pitfall #8): `if !connected { return trayIconErrorBytes }` — the daemon-disconnect error icon always takes priority over any progress quartile, regardless of `a.lastTrayQuartile`.

### Windows-specific HICON design

Windows uses pre-cached HICON handles (created at `initTray` time) rather than decoding PNG bytes at every tray update. This matches the existing `wt.hIcon` and `wt.hIconErr` pattern. The 4 quartile handles (`hIconProgress25/50/75/100`) are:
- Created in `initTray` via `createIconFromPNG(trayIconProgress*Bytes)`
- Used in `updateTray` with a 5-way switch on `a.lastTrayQuartile`
- Destroyed in `cleanupTray` alongside `hIcon` and `hIconErr`

**Note for Phase 99 / future maintenance:** The HICON handle disposal was added to `cleanupTray` in this wave. No OS-level handle leak should occur on normal shutdown. If `initTray` fails mid-sequence (e.g., progress-75 icon fails), the earlier handles (progress-25, progress-50) are explicitly destroyed before returning. This was implemented to avoid handle leaks on initialization failure.

The `trayIconBytesForState` helper is still defined in `tray_windows.go` — it is exercised by `TestApp_SetTrayProgress_ErrorPrecedence` and available for future callers. The production update path uses HICON handles for performance.

### refreshTrayStateFunc injection

A `refreshTrayStateFunc func()` field was added to `App` (parallel to `saveFileDialogFunc`). `SetTrayProgress` calls it when non-nil, otherwise calls `a.refreshTrayState()` directly. This allows unit tests to isolate the SetTrayProgress logic from the platform-specific tray APIs (cgo on darwin, D-Bus on Linux, Win32 on Windows) without requiring OS-level tray infrastructure.

## Unit Tests (3 GREEN)

| Test | Verifies |
|------|----------|
| TestApp_SetTrayProgress_Idempotent | Same quartile twice → refreshTrayState called only once; lastTrayQuartile updated correctly |
| TestApp_SetTrayProgress_BoundsCheck | -1 and 5 return non-nil error containing "out of range" and the invalid value; state not mutated |
| TestApp_SetTrayProgress_ErrorPrecedence | trayIconBytesForState(false, quartile=3) → trayIconErrorBytes (not trayIconProgress75Bytes) |

## Cross-Platform Compile Gate

| Target | Result |
|--------|--------|
| `go build ./...` (darwin host) | OK |
| `GOOS=linux go build ./...` | OK |
| `GOOS=windows go build ./...` | OK |
| `wails build -tags wailsassets` | OK |

## Wave 0 RED Test Flipped

`TestPRG_SetTrayProgressUsage` (in `internal/release/no_progress_when_off_test.go`) was RED at Wave 0 because `(*App).SetTrayProgress` did not exist. After Task 2, it is GREEN: the regex finds exactly one `func (a *App) SetTrayProgress` definition in `app.go`.

## Cuttability Invariant

This wave produces no UI side effects on its own:
- `SetTrayProgress` is a Wails RPC with no frontend caller yet (Wave 2 wires it)
- `aggregateProgress` is a pure function with no caller yet (Wave 2 wires it)
- No tab UI consumes `tabProgress` (Wave 2 adds the TabBar prop)
- Binary behavior is identical to Wave 0 (tests added, implementation added, but no invocation path exists from the UI)

`TestPRG_OffPath_NoProgressLogic` and `TestPRG_NewProgressAddonIsGated` remain GREEN.

## Deviations from Plan

### Auto-added: refreshTrayStateFunc injection field

- **Rule applied:** Rule 2 (missing critical functionality for correctness)
- **Found during:** Task 2, Sub-task F (unit tests)
- **Issue:** `(*App).SetTrayProgress` calls `a.refreshTrayState()` which calls `updateTray()` which invokes platform-specific tray APIs (cgo on darwin, D-Bus on Linux, Win32 on Windows). Unit tests with `trayInit: true` would have triggered cgo calls, which are not available in a standard `go test` environment.
- **Fix:** Added `refreshTrayStateFunc func()` field to `App` struct (nil in production; set in tests). `SetTrayProgress` calls the func when non-nil. This exactly mirrors the existing `saveFileDialogFunc` injection pattern.
- **Files modified:** `app.go`
- **Commits:** Included in Task 2 commit e8cde0b

### Auto-fixed: node_modules symlink for vitest

- **Rule applied:** Rule 3 (blocking issue)
- **Found during:** Task 1 verification
- **Issue:** The worktree has no `frontend/node_modules` (gitignored). The vitest binary is in the main repo's node_modules.
- **Fix:** Created a `frontend/node_modules` symlink to the main repo's node_modules during testing. The symlink is gitignored and not committed.

### Auto-fixed: @xterm/addon-progress missing from main repo node_modules

- **Rule applied:** Rule 3 (blocking issue)
- **Found during:** Task 2 wails build verification
- **Issue:** The main repo's `frontend/node_modules/@xterm/addon-progress` was not installed (package.json had it but node_modules was stale). The wails build TypeScript type check failed with "Cannot find module '@xterm/addon-progress'".
- **Fix:** Ran `pnpm install` in the main repo's `frontend/` directory to install the package. This is the expected state post-Wave 0 merge; the node_modules gap was a worktree artifact.

## Known Stubs

None — all Wave 2 features are deferred to Plan 03 (TerminalPanel hot-swap arm) and Plan 04 (TabBar underline). This wave's scope is fully implemented.

## Threat Flags

No new threat surface introduced beyond the plan's registered threats. All three STRIDE entries (T-98-02, T-98-03, T-98-08) are mitigated:

| Mitigation | Code | Test |
|------------|------|------|
| T-98-02 bounds check | `if quartile < 0 || quartile > 4 { return error }` | TestApp_SetTrayProgress_BoundsCheck |
| T-98-03 idempotency | `if a.lastTrayQuartile == quartile { return nil }` | TestApp_SetTrayProgress_Idempotent |
| T-98-08 error precedence | `if !connected { return trayIconErrorBytes }` | TestApp_SetTrayProgress_ErrorPrecedence |

## Self-Check

- [x] Task 1 commit c8eb1df exists: `feat(98-02): implement aggregateProgress mean-bucket function (turns 9 vitest cases GREEN)`
- [x] Task 2 commit e8cde0b exists: `feat(98-02): SetTrayProgress RPC + cross-platform tray byte selector + Wails bindings`
- [x] `frontend/src/lib/aggregateProgress.ts` — mean-bucket implementation present
- [x] `aggregateProgress.test.ts` — 9 tests pass
- [x] `app.go` — `lastTrayQuartile` field, `refreshTrayStateFunc` field, `SetTrayProgress` method
- [x] `tray.go` — 4 //go:embed directives + `trayIconBytesForState` helper + refactored `updateTray`
- [x] `tray_linux.go` — 4 //go:embed directives + `trayIconBytesForState` helper + refactored `updateTray`
- [x] `tray_windows.go` — 4 //go:embed + 4 HICON fields + initTray pre-creation + refactored `updateTray` + cleanupTray disposal
- [x] `app_set_tray_progress_test.go` — 3 unit tests present
- [x] `App.d.ts` — `SetTrayProgress` export present
- [x] `App.js` — `SetTrayProgress` Call present
- [x] 3 unit tests GREEN: Idempotent, BoundsCheck, ErrorPrecedence
- [x] TestPRG_SetTrayProgressUsage flipped GREEN
- [x] GOOS=linux build OK
- [x] GOOS=windows build OK
- [x] wails build -tags wailsassets OK

## Self-Check: PASSED
