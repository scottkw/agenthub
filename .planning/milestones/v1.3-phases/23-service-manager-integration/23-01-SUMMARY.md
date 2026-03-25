---
phase: 23-service-manager-integration
plan: 01
subsystem: daemon
tags: [service-manager, kardianos, launchd, systemd, refactor]
dependency_graph:
  requires: []
  provides: [ServiceControl, RunDaemonService, runDaemonCore]
  affects: [internal/daemon/process.go, internal/daemon/service.go]
tech_stack:
  added: [github.com/kardianos/service v1.2.4]
  patterns: [service.Interface adapter, context-driven daemon core, goroutine-based service wrapper]
key_files:
  created:
    - internal/daemon/service.go
    - internal/daemon/service_test.go
  modified:
    - internal/daemon/process.go
    - go.mod
    - go.sum
decisions:
  - "Use KeepAlive=false to allow manual stop without automatic restart (per RESEARCH.md Pitfall 4)"
  - "Use UserService=true for user-scope service: ~/Library/LaunchAgents on macOS, ~/.config/systemd/user on Linux"
  - "runDaemonCore uses fmt.Fprintf+return instead of os.Exit so service manager gets clean return"
metrics:
  duration: "104s"
  completed: "2026-03-24"
  tasks_completed: 2
  files_changed: 5
---

# Phase 23 Plan 01: Service Manager Integration — kardianos/service Wrapper Summary

**One-liner:** kardianos/service v1.2.4 integration with daemonSvc adapter, runDaemonCore extraction, and UserService/RunAtLoad/KeepAlive config for platform-native auto-start.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add kardianos/service dependency and create service.go with refactored runDaemonCore | 896301b | go.mod, go.sum, internal/daemon/service.go, internal/daemon/process.go |
| 2 | Unit tests for service config, daemonSvc lifecycle, and ServiceControl | a609559 | internal/daemon/service_test.go |

## What Was Built

### Task 1: kardianos/service Integration

Added `github.com/kardianos/service v1.2.4` as a direct dependency and created `internal/daemon/service.go` with:

- **`daemonSvc`** — implements `service.Interface` with non-blocking `Start` (launches goroutine) and context-cancelling `Stop` (waits for goroutine to finish)
- **`newServiceConfig()`** — builds `service.Config` with `UserService=true`, `RunAtLoad=true`, `KeepAlive=false`, absolute executable path, and `Arguments: ["daemon"]`
- **`ServiceControl(action string) error`** — exported function that creates a service instance and dispatches install/uninstall/start/stop to `kardianos/service.Control`

Refactored `internal/daemon/process.go`:
- Extracted `runDaemonCore(ctx context.Context)` from `RunDaemon()` — contains all daemon startup logic, blocks on `<-ctx.Done()`, uses `fmt.Fprintf+return` instead of `os.Exit` for clean service manager shutdown
- `RunDaemon()` simplified to signal context setup + `runDaemonCore(ctx)` delegation

### Task 2: Unit Tests

Created `internal/daemon/service_test.go` with 6 tests:
- `TestNewServiceConfig_Fields` — verifies all config field values
- `TestNewServiceConfig_AbsolutePath` — verifies executable is absolute
- `TestDaemonSvc_ImplementsInterface` — compile-time `service.Interface` check
- `TestDaemonSvc_StopNilCancel` — nil cancel safety check
- `TestServiceControl_Exported` — compile-time export check
- `TestRunDaemonCore_CancelledContext` — verifies clean return on cancelled context (5s timeout)

## Verification Results

```
go build ./internal/daemon/     -> OK
go build ./cmd/agenthub-cli/    -> OK
go vet ./internal/daemon/       -> OK
go test ./internal/daemon/ -timeout 60s -> PASS (all tests)
grep kardianos/service go.mod   -> 1 match
```

## Deviations from Plan

None - plan executed exactly as written.

The merge from main (87b1db1 -> 4231814) was required to bring the worktree branch up-to-date with the daemon package developed in phases 19-22. This was a necessary setup step, not a deviation.

## Self-Check: PASSED

Files created:
- internal/daemon/service.go: FOUND
- internal/daemon/service_test.go: FOUND

Commits:
- 896301b (feat): FOUND
- a609559 (test): FOUND
