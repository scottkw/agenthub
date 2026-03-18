---
phase: 01-pty-foundation
plan: "01"
subsystem: pty
tags: [go, go-pty, pty, session, cli-detection, registry, concurrency]

# Dependency graph
requires: []
provides:
  - Go module (github.com/agenthub/agenthub) with go-pty v0.2.2 dependency
  - SessionBackend interface contract (Create, Resize, Kill, List)
  - Session struct with ID, CLI, State, PTY handle, io.ReadWriter
  - DetectCLIs / DetectCLI functions for PATH-based CLI discovery
  - Thread-safe SessionRegistry (Add, Get, List, Remove, Len, KillAll)
affects:
  - 01-02 (PTY backend implements SessionBackend against these contracts)
  - All subsequent phases (registry and session types are core data structures)

# Tech tracking
tech-stack:
  added:
    - github.com/aymanbagabas/go-pty v0.2.2
    - github.com/creack/pty v1.1.21 (transitive)
    - github.com/u-root/u-root v0.11.0 (transitive)
    - golang.org/x/sys, golang.org/x/crypto (transitive)
  patterns:
    - Interface-first design: SessionBackend interface defined before any implementation
    - TDD with RED/GREEN cycles for each behavioral component
    - sync.RWMutex for read-heavy concurrent map (registry uses RLock for reads)
    - exec.LookPath for platform-agnostic CLI binary discovery
    - make([]T, 0) instead of nil slice for always-non-nil return values

key-files:
  created:
    - go.mod
    - go.sum
    - internal/pty/backend.go
    - internal/pty/session.go
    - internal/pty/detect.go
    - internal/pty/detect_test.go
    - internal/pty/registry.go
    - internal/pty/registry_test.go
  modified: []

key-decisions:
  - "SessionBackend is an interface, not a concrete type — lets Plan 02 provide the platform implementation without touching Plan 01 types"
  - "Session.Read/Write delegate to gopty.Pty — prepares io.ReadWriter contract for Phase 2 fan-out without requiring a real PTY in this plan"
  - "KillAll marks StateStopped and clears map; actual process kill wired in Plan 02"
  - "DetectCLIs returns make([]DetectedCLI, 0) not nil — caller can range/len safely without nil check"

patterns-established:
  - "Interface contract first: define the interface in backend.go before any implementation exists"
  - "Registry owns session lifetime: context cancellation does NOT remove sessions from registry"
  - "Always-non-nil slice pattern: use make([]T, 0) as initializer for public return values"

requirements-completed:
  - CLI-01
  - SESS-01

# Metrics
duration: 2min
completed: "2026-03-18"
---

# Phase 1 Plan 1: PTY Foundation Scaffold Summary

**Go module with go-pty v0.2.2, SessionBackend interface, Session/Registry types, PATH-based CLI detection — 10 tests passing with race detector**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-18T00:07:19Z
- **Completed:** 2026-03-18T00:09:29Z
- **Tasks:** 3 of 3
- **Files modified:** 8

## Accomplishments

- Initialized Go module `github.com/agenthub/agenthub` with go-pty v0.2.2 as the cross-platform PTY library
- Defined `SessionBackend` interface (Create/Resize/Kill/List) and `Session` struct — Plan 02 implements against these contracts with no changes needed here
- Implemented `DetectCLIs` / `DetectCLI` with `exec.LookPath` covering claude, codex, gemini, opencode; 4 tests verify PATH-based discovery and empty-slice guarantee
- Implemented thread-safe `SessionRegistry` with `sync.RWMutex`; 6 tests pass including concurrent goroutine test and race detector

## Task Commits

Each task was committed atomically:

1. **Task 1: Initialize Go module, define interfaces and types** - `3a8817a` (feat)
2. **Task 2: Implement CLI detection with tests** - `0b167ec` (feat)
3. **Task 3: Implement session registry with tests** - `e6bcc1d` (feat)

## Files Created/Modified

- `go.mod` - Module declaration, go-pty v0.2.2 direct dependency
- `go.sum` - Dependency checksums
- `internal/pty/backend.go` - SessionBackend interface, CreateRequest struct, ErrSessionNotFound
- `internal/pty/session.go` - Session struct, SessionState enum (Running/Stopped), Read/Write/String methods
- `internal/pty/detect.go` - CLISpec, DetectedCLI, knownCLIs, DetectCLIs, DetectCLI, ErrCLINotFound
- `internal/pty/detect_test.go` - 4 unit tests for CLI detection
- `internal/pty/registry.go` - SessionRegistry with Add/Get/List/Remove/Len/KillAll
- `internal/pty/registry_test.go` - 6 unit tests including concurrent race test

## Decisions Made

- `SessionBackend` is an interface so Plan 02 provides the platform-specific PTY implementation without touching Plan 01 types
- `Session.Read/Write` delegate to `gopty.Pty` — establishes the `io.ReadWriter` contract that Phase 2 fan-out multiplexer will consume
- `KillAll` marks `StateStopped` and clears the map; actual process kill is wired in Plan 02 when the backend exists
- `DetectCLIs` returns `make([]DetectedCLI, 0)` (never nil) so callers can safely range without a nil check

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 02 can immediately implement `SessionBackend` against the defined interface in `backend.go`
- `Session` and `SessionRegistry` are ready for use; no API changes anticipated
- go-pty v0.2.2 is vendored in `go.sum`, so offline builds will work

---
*Phase: 01-pty-foundation*
*Completed: 2026-03-18*
