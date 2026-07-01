---
phase: 167-native-notifications
plan: 01
subsystem: daemon
tags: [go, settings-persistence, rest-api, unix-socket]

# Dependency graph
requires:
  - phase: 150-shell-web-share-polish
    provides: "The StartMinimized boolean-setting template (engine field + Get/Set + settings.json persistence) that this plan mirrors verbatim."
provides:
  - "daemonSettings.NotifyOnWaiting persisted boolean (json:notifyOnWaiting,omitempty), default OFF"
  - "SessionEngine.GetNotifyOnWaiting() / SetNotifyOnWaiting(bool)"
  - "GET/PATCH /settings/notify-on-waiting REST routes"
  - "DaemonClient.GetNotifyOnWaiting() / SetNotifyOnWaiting(bool)"
affects: [167-02, 167-03, 167-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Boolean daemon setting mirrors StartMinimized exactly: struct field (omitempty) + engine field + load/save wiring + Get(RLock)/Set(Lock+save) + GET/PATCH REST pair + DaemonClient wrapper"

key-files:
  created:
    - internal/daemon/engine_notify_test.go
    - internal/daemon/api_notify_test.go
  modified:
    - internal/daemon/engine.go
    - internal/daemon/api.go
    - internal/daemon/client.go

key-decisions:
  - "No CurrentSchemaVersion bump and no defaults-merge entry — zero-value false is the correct default (RESEARCH Pitfall 4)."
  - "NotifyOnWaiting placed immediately after StartMinimized in both the daemonSettings struct and the SessionEngine struct, matching the plan's LOCKED decision #6 (no separate Settings struct in types.go)."

patterns-established: []

requirements-completed: [NTF-04]

coverage:
  - id: D1
    description: "NotifyOnWaiting persists to settings.json (0600) and survives a daemon restart; defaults to false when absent (no schema bump)"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "internal/daemon/engine_notify_test.go#TestNotifyOnWaiting_Default"
        status: pass
      - kind: unit
        ref: "internal/daemon/engine_notify_test.go#TestNotifyOnWaiting_Persists"
        status: pass
      - kind: unit
        ref: "internal/daemon/engine_notify_test.go#TestNotifyOnWaiting_RoundTrip"
        status: pass
      - kind: unit
        ref: "internal/daemon/engine_notify_test.go#TestNotifyOnWaiting_NoSchemaBump"
        status: pass
    human_judgment: false
  - id: D2
    description: "GET/PATCH /settings/notify-on-waiting REST routes read/write the engine value; malformed PATCH body returns 400 without mutating state"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "internal/daemon/api_notify_test.go#TestAPIGetNotifyOnWaiting_Default"
        status: pass
      - kind: unit
        ref: "internal/daemon/api_notify_test.go#TestAPIPatchNotifyOnWaiting_FlipsTrue"
        status: pass
      - kind: unit
        ref: "internal/daemon/api_notify_test.go#TestAPIPatchNotifyOnWaiting_BadBody"
        status: pass
    human_judgment: false
  - id: D3
    description: "DaemonClient.GetNotifyOnWaiting / SetNotifyOnWaiting round-trip over the Unix socket"
    requirement: "NTF-04"
    verification:
      - kind: unit
        ref: "internal/daemon/api_notify_test.go#TestDaemonClient_GetSetNotifyOnWaiting_RoundTrip"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-07-01
status: complete
---

# Phase 167 Plan 01: NotifyOnWaiting Persistence Layer Summary

**Persisted, socket-reachable `NotifyOnWaiting` boolean setting (engine + REST + client) mirroring `StartMinimized` exactly, defaulting OFF with zero schema-version impact.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-07-01T06:30:00Z (approx)
- **Completed:** 2026-07-01T06:42:00Z (approx)
- **Tasks:** 2
- **Files modified:** 5 (3 modified, 2 created)

## Accomplishments
- `daemonSettings.NotifyOnWaiting` added to the on-disk settings schema (0600 `settings.json`), `omitempty` so existing files stay unchanged when the flag is false
- `SessionEngine.GetNotifyOnWaiting()` / `SetNotifyOnWaiting(bool)` wired through `loadSettingsFromDisk` / `saveSettingsToDisk`, copied verbatim from the `StartMinimized` pair (RLock/RUnlock read, Lock+save write)
- `GET /settings/notify-on-waiting` and `PATCH /settings/notify-on-waiting` routes registered and handled identically to the start-minimized pattern (400 on malformed PATCH body, 204 on success)
- `DaemonClient.GetNotifyOnWaiting()` / `SetNotifyOnWaiting(bool)` added to the client for the app layer and GUI to call over the Unix socket
- No `CurrentSchemaVersion` bump (still 4) and no defaults-merge rewrite triggered — verified with a dedicated no-op-rewrite test against a pre-existing up-to-date `settings.json`

## Task Commits

Each task was committed atomically:

1. **Task 1: NotifyOnWaiting field + Get/Set on the engine (mirror StartMinimized)** - `59851387` (feat)
2. **Task 2: REST routes + DaemonClient methods for notify-on-waiting** - `c703aece` (feat)

_Note: Both tasks were TDD-flavored (implementation + accompanying test file committed together per task, following this project's established single-commit-per-task convention rather than separate RED/GREEN commits — the plan's frontmatter is `type: execute`, not `type: tdd`, so the plan-level TDD gate does not apply)._

## Files Created/Modified
- `internal/daemon/engine.go` - `daemonSettings.NotifyOnWaiting` field, `SessionEngine.notifyOnWaiting` field, load/save wiring, `GetNotifyOnWaiting`/`SetNotifyOnWaiting`
- `internal/daemon/engine_notify_test.go` - default/persist/round-trip/no-schema-bump unit tests
- `internal/daemon/api.go` - `GET`/`PATCH /settings/notify-on-waiting` route registrations + handlers
- `internal/daemon/client.go` - `DaemonClient.GetNotifyOnWaiting` / `SetNotifyOnWaiting`
- `internal/daemon/api_notify_test.go` - GET default, PATCH persists + 204, malformed PATCH 400, client round-trip

## Decisions Made
- Mirrored `StartMinimized` exactly per the plan's LOCKED decision #6 — no new `Settings` struct in `types.go`, no schema bump, no defaults-merge entry (zero-value `false` is the correct default per RESEARCH Pitfall 4).
- Verified the no-schema-bump / no-unnecessary-rewrite behavior with an explicit test (`TestNotifyOnWaiting_NoSchemaBump`) that writes a pre-existing up-to-date `settings.json`, loads it, and asserts the file's mtime is unchanged — going beyond the plan's stated acceptance criteria (grep for `CurrentSchemaVersion` unchanged) to also prove the runtime load path doesn't silently rewrite disk.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The `NotifyOnWaiting` persistence layer (engine + REST + client) is complete and unit-tested; ready for Plan 03 (app-layer `atomic.Bool` cache + notification trigger) and Plan 04 (frontend Settings toggle) to consume it.
- No blockers. `go test ./internal/daemon/ -race -short` is green; `CurrentSchemaVersion` remains 4.

---
*Phase: 167-native-notifications*
*Completed: 2026-07-01*

## Self-Check: PASSED
