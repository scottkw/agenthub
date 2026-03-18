---
phase: 05-qr-codes-status-indicators
plan: "02"
subsystem: status-detection
tags: [go, status, heuristics, wails, events, tdd]
dependency_graph:
  requires: []
  provides: [internal/status package, App.GetSessionStatus, session:status EventsEmit]
  affects: [app.go, frontend status badge (Plan 03)]
tech_stack:
  added: [internal/status package]
  patterns: [Hub subscriber goroutine, PatternSet pluggable patterns, EventsEmit push]
key_files:
  created:
    - internal/status/detector.go
    - internal/status/detector_test.go
  modified:
    - app.go
    - app_test.go
decisions:
  - "Detector initial state is empty-sentinel so first Feed always fires onTransit"
  - "Guard EventsEmit with ctx.Value(frontend) to prevent Wails runtime panic in tests"
  - "statusMu is separate from main mu to avoid contention between Hub drain and status updates"
metrics:
  duration: "8min"
  completed: "2026-03-18"
  tasks: 2
  files: 4
---

# Phase 05 Plan 02: Status Detection Engine Summary

Regex-based PTY output classifier with rolling 4KB tail, ANSI stripping, and Wails EventsEmit push for live per-tab status badges.

## What Was Built

### Task 1: internal/status package (TDD)

Created `internal/status/detector.go`:

- `SessionStatus` type with constants: `running`, `waiting`, `idle`, `errored`
- `PatternSet` struct with `Working`, `Idle`, `Waiting` regexp slices
- `DefaultClaudePatterns()` — empirically-known Claude Code patterns from RESEARCH.md
- `FallbackPatterns()` — empty set (conservative running-only for unknown CLIs)
- `PatternsForCLI(cliName)` — dispatcher: "claude" gets Claude patterns, else fallback
- `StripANSI(b []byte) []byte` — removes CSI escape sequences before pattern matching
- `appendTail(existing, new []byte, maxLen int) []byte` — rolling buffer with front-trim
- `Detector` struct with `Feed(raw []byte)` and `classify() SessionStatus`
- `HubLike` interface for testability (Subscribe/Unsubscribe/Done/ScrollbackSnapshot)
- `Watch(hub HubLike, sessionID, cli string, onTransit)` — goroutine that subscribes to hub, feeds scrollback snapshot first, processes frames, exits cleanly on `hub.Done()`

Key design: Detector initial `current` is `""` (empty sentinel) so the very first `Feed` always triggers `onTransit` — callers always get an initial status notification.

Classification priority: Waiting > Working (Running) > Idle > default Running.

### Task 2: App wiring (TDD)

Modified `app.go`:

- Added `sessionStatuses map[string]status.SessionStatus` + `statusMu sync.RWMutex` to `App` struct
- `NewApp()` initializes `sessionStatuses`
- `CreateSession`: captures `*Hub` return from `a.manager.Create`, starts `go status.Watch(hub, id, cli, onTransit)` goroutine where onTransit stores in map and calls `runtime.EventsEmit(ctx, "session:status", ...)`
- `KillSession`: emits `StatusErrored` before deleting from map
- `GetSessionStatus(sessionID string) string` — public accessor, returns "running" if not found

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Empty-sentinel initial state for Detector**
- **Found during:** Task 1 GREEN phase
- **Issue:** `TestDetector_ClaudeRunning` and `TestDetector_DefaultRunning` failed because Detector initialized `current = StatusRunning`, so first `Feed` with a "running" classification produced no transition (no callback fired), leaving `got = ""`
- **Fix:** Changed initial `current` to `""` (empty sentinel) so first classification always fires `onTransit` regardless of the result
- **Files modified:** internal/status/detector.go
- **Commit:** 50b2ed2

**2. [Rule 1 - Bug] Wails runtime panic with non-Wails context in tests**
- **Found during:** Task 2 full suite run with -race
- **Issue:** `TestKillSession` called `KillSession` which called `runtime.EventsEmit(context.Background(), ...)` — Wails panics with "invalid context" when not inside the Wails event loop
- **Fix:** Guard all `runtime.EventsEmit` calls with `a.ctx != nil && a.ctx.Value("frontend") != nil` — same pattern already used in `beforeClose`. In tests, `ctx = context.Background()` has no "frontend" key so EventsEmit is skipped safely
- **Files modified:** app.go
- **Commit:** d708ed1

## Self-Check: PASSED

- internal/status/detector.go: FOUND
- internal/status/detector_test.go: FOUND
- app.go: FOUND
- app_test.go: FOUND
- Commit cf4bb80 (test RED phase 1): FOUND
- Commit 50b2ed2 (feat task 1): FOUND
- Commit 5eb836d (test RED phase 2): FOUND
- Commit d708ed1 (feat task 2): FOUND
