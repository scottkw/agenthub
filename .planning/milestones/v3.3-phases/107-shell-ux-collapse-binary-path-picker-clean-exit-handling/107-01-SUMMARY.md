---
phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling
plan: "107-01"
subsystem: daemon
tags: [go, daemon, settings, shell, http-api, wails]

# Dependency graph
requires:
  - phase: 101-shell-ux
    provides: shellWebShareWarned plumbing pattern mirrored verbatim
provides:
  - shellPath settings field with Get/SetShellPath engine methods
  - GET/PATCH /settings/shell-path HTTP routes with executable validation
  - DaemonClient.GetShellPath/SetShellPath methods
  - App.GetShellPath/SetShellPath Wails wrappers
  - TypeScript bindings for GetShellPath/SetShellPath in App.d.ts + App.js
  - resolveDefaultShellPath() fallback chain: $SHELL -> DiscoverShells[name=shell] -> platform hardcode
  - resolveShellSpawn branch (0): shellPath override for cli=shell before cliPaths
affects:
  - 107-03 (frontend Settings field consumes GetShellPath on mount, SetShellPath on Save Paths)
  - 107-02 (frontend modal) references shell-path for display

# Tech tracking
tech-stack:
  added: []
  patterns:
    - shellWebShareWarned mirror pattern: field + struct tag + load/save + Get/Set methods + HTTP route pair + DaemonClient pair + Wails wrapper pair + TS bindings
    - resolveDefaultShellPath() resolution order: env var -> DiscoverShells synthetic entry -> platform hardcode

key-files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - internal/daemon/engine_test.go
    - internal/daemon/api_test.go

key-decisions:
  - "Validation on PATCH is daemon-side (exist + executable bit); frontend receives 400 + plain-text body verbatim"
  - "Empty PATCH value clears override and restores platform default (not an error)"
  - "resolveShellSpawn branch (0) fires only for cli=shell AND cliPaths[shell] unset AND shellPath non-empty, preserving per-binary cliPaths overrides"
  - "resolveDefaultShellPath: $SHELL env -> DiscoverShells name=shell -> platform hardcode; NEVER returns empty string"

patterns-established:
  - "Shell settings plumbing: mirror shellWebShareWarned pattern verbatim (engine field + settings struct + load/save + Get/Set + HTTP pair + client pair + Wails wrapper + TS bindings)"

requirements-completed: [SHELL-11]

# Metrics
duration: 30min
completed: 2026-05-13
---

# Phase 107 Plan 01: Shell-Path Settings Backend Summary

**Persisted `shellPath` setting plumbed end-to-end: daemon engine + GET/PATCH /settings/shell-path (executable validation) + DaemonClient + Wails wrappers + TypeScript bindings, with resolveShellSpawn branch (0) honoring the user's configured shell binary**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-05-13T04:15:00Z
- **Completed:** 2026-05-13T04:47:41Z
- **Tasks:** 2 (both TDD: RED + GREEN)
- **Files modified:** 8

## Accomplishments
- `daemonSettings.ShellPath` field and `SessionEngine.shellPath` field added beside `shellWebShareWarned`; round-trips through settings.json via loadSettingsFromDisk/saveSettingsToDisk
- `GetShellPath()` never returns empty: resolves `$SHELL` -> `DiscoverShells()` `name="shell"` entry -> platform hardcode (`/bin/zsh` darwin, `/bin/bash` linux, `pwsh.exe` windows)
- `SetShellPath(path)` validates existence and executable bit (mode & 0111); empty path clears override cleanly
- `resolveShellSpawn` branch (0): when `cli=="shell"` and `cliPaths["shell"]` unset and `shellPath` non-empty, uses the configured binary with spec-derived argv; falls through to discovery if basename unknown
- HTTP layer: GET returns `{"value": "<path>"}` (200, never empty); PATCH returns 204 on success or 400 on validation failure
- DaemonClient and App Wails wrappers with nil-client guards, matching existing patterns exactly
- TypeScript bindings: `GetShellPath(): Promise<string>` and `SetShellPath(v: string): Promise<void>`
- 9 new tests (5 engine, 4 API); all pass under `-race -count=1`

## Task Commits

Each task was committed atomically with TDD (RED then GREEN):

1. **Task 1 RED: engine_test.go failing tests** - `e42cf19` (test)
2. **Task 1 GREEN: engine.go implementation** - `840f9e3` (feat)
3. **Task 2 RED: api_test.go failing tests** - `1cd23b5` (test)
4. **Task 2 GREEN: api.go + client.go + app.go + TS bindings** - `3f52eb0` (feat)

## Files Created/Modified
- `internal/daemon/engine.go` - shellPath field + daemonSettings.ShellPath + load/save + resolveDefaultShellPath() + GetShellPath() + SetShellPath() + resolveShellSpawn branch (0)
- `internal/daemon/api.go` - GET/PATCH /settings/shell-path route registration + handlers
- `internal/daemon/client.go` - GetShellPath()/SetShellPath() DaemonClient methods
- `app.go` - GetShellPath()/SetShellPath() Wails wrappers (gofmt pre-existing alignment fix applied)
- `frontend/src/wailsjs/go/main/App.d.ts` - GetShellPath/SetShellPath TypeScript declarations
- `frontend/src/wailsjs/go/main/App.js` - GetShellPath/SetShellPath Call() bindings
- `internal/daemon/engine_test.go` - 5 new engine tests (TestGetShellPath_Default, TestSetShellPath_Rejects*, TestSetShellPath_Accepts*, TestSetShellPath_Empty*)
- `internal/daemon/api_test.go` - 4 new API tests (TestHandleGetShellPath_*, TestHandleUpdateShellPath_*)

## Decisions Made
- Validation runs daemon-side on PATCH; frontend (107-03) receives 400 + plain-text error body verbatim. Keeps validation in one place, avoids duplication.
- Empty value in PATCH clears override (returns 204), restoring platform default — not an error. Enables "reset to default" UX without a separate endpoint.
- resolveShellSpawn branch (0) ONLY fires when `cliPaths["shell"]` is unset, preserving per-binary `cliPaths["bash"/"zsh"]` overrides that take precedence.
- gofmt formatting applied to app.go (pre-existing struct alignment drift, not caused by this plan).

## Deviations from Plan

None - plan executed exactly as written. The gofmt struct alignment fix in app.go was pre-existing drift, not caused by this plan's changes (gofmt -l detected it as a side effect of opening the file for editing).

## Issues Encountered
- None.

## Known Stubs
- None. All plumbing is live; there are no hardcoded placeholder values in the files touched.

## Threat Flags
- None. No new network endpoints beyond the planned GET/PATCH /settings/shell-path routes. Both routes are on the Unix domain socket (daemon-local), consistent with the existing settings routes security model.

## Next Phase Readiness
- 107-03 (frontend Settings field) can now call `GetShellPath()` on mount and `SetShellPath(v)` on Save Paths click using the Wails bindings added here.
- `GetShellPath()` never returns empty, so the frontend can display the resolved default without a loading state.
- Validation errors from `SetShellPath` are returned as plain-text in the response body, ready to surface in the UI.

## Self-Check

- [x] internal/daemon/engine.go modified — confirmed
- [x] internal/daemon/api.go modified — confirmed
- [x] internal/daemon/client.go modified — confirmed
- [x] app.go modified — confirmed
- [x] frontend/src/wailsjs/go/main/App.d.ts modified — confirmed
- [x] frontend/src/wailsjs/go/main/App.js modified — confirmed
- [x] Commits e42cf19, 840f9e3, 1cd23b5, 3f52eb0 — confirmed in git log
- [x] All 9 new tests pass under -race -count=1
- [x] gofmt -l clean; go vet ./internal/daemon/ clean
- [x] go build . clean

## Self-Check: PASSED

---
*Phase: 107-shell-ux-collapse-binary-path-picker-clean-exit-handling*
*Completed: 2026-05-13*
