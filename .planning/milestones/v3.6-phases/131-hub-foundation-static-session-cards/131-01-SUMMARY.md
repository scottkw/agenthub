---
phase: 131
plan: "01"
subsystem: daemon-backend
tags: [hub, session-grid, workdir, wails-binding, tdd]
dependency_graph:
  requires: []
  provides:
    - daemon.SessionInfo.WorkDir field (internal/daemon/types.go)
    - engine.ListSessions() WorkDir population (internal/daemon/engine.go)
    - app.go SessionInfo with ViewerCount, ExitCode, Duration, WorkDir
    - App.d.ts workDir binding stub
  affects:
    - All ListSessions() callers (Hub card layer, DaemonManagerPanel)
    - TypeScript frontend consuming SessionInfo
tech_stack:
  added: []
  patterns:
    - TDD RED/GREEN cycle for daemon.SessionInfo.WorkDir
    - Lock-safe map read under existing e.mu.RLock (no new lock)
    - Silent-drop prevention pattern (mirrors HomeDir/FilesWrite UAT lesson)
key_files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
    - app.go
    - frontend/src/wailsjs/go/main/App.d.ts
decisions:
  - "WorkDir added after FilesWrite in daemon.SessionInfo (not CreateRequest); comment clarifies the confusion risk"
  - "Map read inside ListSessions uses held e.mu.RLock via defer — no second lock acquired"
  - "app.go propagates all four Hub fields with the same comment pattern as HomeDir/FilesWrite to prevent future silent drops"
metrics:
  duration: "~15 minutes"
  completed: "2026-06-16"
  tasks_completed: 2
  files_modified: 5
---

# Phase 131 Plan 01: Hub Backend Data Plumbing Summary

One-liner: Wired WorkDir, ViewerCount, ExitCode, and Duration from the engine's session maps through daemon.SessionInfo and app.go's Wails-bound SessionInfo to the TypeScript binding stub.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add WorkDir to daemon.SessionInfo and populate in ListSessions() | f7cd136 | internal/daemon/types.go, engine.go, engine_test.go |
| 2 | Propagate four fields through app.go and update App.d.ts | 6138011 | app.go, frontend/src/wailsjs/go/main/App.d.ts |

## What Was Built

**Task 1 (TDD):**
- Added `WorkDir string` field with `json:"workDir"` tag to `daemon.SessionInfo` in `internal/daemon/types.go`, after `FilesWrite`, with Phase 131 / GRID-02 comment
- Added `WorkDir: e.sessionWorkDirs[s.ID]` to the `SessionInfo{...}` literal in `engine.ListSessions()`. The read is safe because the function already holds `e.mu.RLock()` via `defer` — no new lock acquired. Added inline comment noting this.
- Added two Go tests in `engine_test.go` following the existing `newBareEngine` / `newExitedShell12Session` pattern:
  - `TestListSessions_WorkDir_Populated`: creates real session with tmpDir, asserts `SessionInfo.WorkDir` equals the EvalSymlinks-resolved path
  - `TestListSessions_WorkDir_EmptyForUnknown`: bare engine with no `sessionWorkDirs` entry, asserts `WorkDir == ""` without panic

**Task 2:**
- Added four fields to `app.go SessionInfo` struct after `FilesWrite`: `ViewerCount int`, `ExitCode *int` (omitempty), `Duration *int` (omitempty), `WorkDir string`, with Phase 131 / CARD-04..06, GRID-02 comment
- Propagated all four in `ListSessions()` result mapping: `ViewerCount: s.ViewerCount`, `ExitCode: s.ExitCode`, `Duration: s.Duration`, `WorkDir: s.WorkDir`
- Added `workDir: string` to `App.d.ts` `SessionInfo` interface with Phase 131 / GRID-02 comment

## Verification Results

- `go test ./internal/daemon/... -count=1`: PASS (all tests including new WorkDir assertions)
- `go build ./...`: SUCCESS
- `grep -q "workDir" App.d.ts`: PASS
- `gofmt -l` on all modified Go files: no output (fully formatted)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all fields are wired to real engine data (WorkDir from sessionWorkDirs map, ViewerCount from hub.SubscriberCount(), ExitCode and Duration from existing stopped-session path).

## Threat Flags

No new threat surface beyond what is documented in the plan's threat model. WorkDir path exposure via ListSessions() accepted per T-131-01 (same user, same single-user local RPC that already returns session names/hostnames).

## Self-Check: PASSED

- `internal/daemon/types.go` modified: FOUND (WorkDir field at line 34)
- `internal/daemon/engine.go` modified: FOUND (sessionWorkDirs[s.ID] at append site)
- `internal/daemon/engine_test.go` modified: FOUND (two new test functions)
- `app.go` modified: FOUND (four fields + propagation)
- `frontend/src/wailsjs/go/main/App.d.ts` modified: FOUND (workDir: string)
- Commit f7cd136: FOUND (`git log --oneline -5`)
- Commit 6138011: FOUND (`git log --oneline -5`)
