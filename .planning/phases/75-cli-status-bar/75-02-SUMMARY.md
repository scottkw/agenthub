---
phase: 75-cli-status-bar
plan: "02"
subsystem: statusbar
tags: [go, ansi, tui, terminal, statusbar, decstbm]
one_liner: "DECSTBM scroll-region status bar package with reverse-video rendering, 1s ticker goroutine, and terminal injection prevention"
dependency_graph:
  requires: []
  provides:
    - internal/statusbar.Bar
    - internal/statusbar.New
    - internal/statusbar.Options
    - internal/statusbar.Position
  affects:
    - cmd_attach.go (future: wire Bar.Start/Stop around PTY loop)
tech_stack:
  added: []
  patterns:
    - sync.Once teardown guard (idempotent Stop)
    - context.WithCancel ticker goroutine lifecycle
    - FallbackCols/FallbackRows for non-TTY test environments
    - utf8.RuneCountInString for terminal-width-safe string truncation
    - sanitize() stripping bytes < 0x20 for terminal injection prevention
key_files:
  created:
    - internal/statusbar/bar.go
    - internal/statusbar/bar_test.go
  modified: []
decisions:
  - "Added FallbackCols/FallbackRows to Options: term.GetSize fails in non-TTY CI environments; fallback dimensions let tests exercise the full render path without a real TTY"
  - "Extracted getSize() helper: centralises fallback logic so both Start() and draw() use consistent size resolution"
  - "Tests use testOpts() helper: reduces per-test boilerplate and ensures all tests supply fallback dimensions"
metrics:
  duration_minutes: 15
  completed_date: "2026-04-15"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 0
---

# Phase 75 Plan 02: statusbar Package Summary

DECSTBM scroll-region status bar package with reverse-video rendering, 1s ticker goroutine, and terminal injection prevention.

## What Was Built

`internal/statusbar` is a self-contained Go package that renders a persistent one-row ANSI status bar using the DECSTBM scroll region technique. It knows nothing about the relay or daemon packages — it only needs an `io.Writer` and configuration options.

**Bar type** (`internal/statusbar/bar.go`):
- `New(w io.Writer, opts Options) *Bar` — constructor
- `Start()` — sets DECSTBM scroll region and launches 1-second ticker goroutine
- `Stop()` — resets scroll region, clears bar line, idempotent via `sync.Once`
- `draw()` — saves/restores cursor, writes formatted bar to reserved row; self-heals on terminal resize by re-issuing DECSTBM when `GetSize` returns new dimensions
- `format(viewerCount, connState, cols)` — assembles all required fields with rune-safe truncation using `utf8.RuneCountInString`
- `SetViewerCount(n int)` / `SetConnectionState(state string)` — thread-safe setters called from any goroutine
- `sanitize(s string)` — strips bytes < 0x20 (including ESC 0x1B) to prevent terminal injection

**Tests** (`internal/statusbar/bar_test.go`): 9 tests covering SB-01/02/04/05/06/07 requirements and T-75-03 threat mitigation.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create internal/statusbar/bar.go | aceba20 | internal/statusbar/bar.go |
| 2 | Create internal/statusbar/bar_test.go | 6c14feb | internal/statusbar/bar.go (amended), internal/statusbar/bar_test.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added FallbackCols/FallbackRows to Options for non-TTY test environments**
- **Found during:** Task 2 — first test run
- **Issue:** `term.GetSize` returns an error when stdout is not a TTY (CI, piped test output). `Start()` hit the `return` early-exit, producing no draw output. All content-checking tests failed with only Stop() cleanup output.
- **Fix:** Added `FallbackCols int` and `FallbackRows int` fields to `Options`. Added `getSize()` helper that tries `term.GetSize` first and falls back to the configured dimensions when both are non-zero. Updated test file to use a `testOpts()` helper that sets `FallbackCols: 120, FallbackRows: 24` for all tests. The production path (`cmd_attach.go`) will always have a real TTY, so the fallback fields are zero-valued and ignored in production.
- **Files modified:** internal/statusbar/bar.go, internal/statusbar/bar_test.go
- **Commit:** 6c14feb

## Threat Model Coverage

| Threat ID | Mitigation | Verified By |
|-----------|-----------|-------------|
| T-75-03 | sanitize() strips bytes < 0x20 from SessionName and Hostname | TestBar_SanitizeSessionName |
| T-75-04 | sync.Once guard in Stop() | TestBar_StopIdempotent |

## Test Results

```
ok  github.com/scottkw/agenthub/internal/statusbar  4.513s  (9/9 tests pass)
go vet: no warnings
```

## Known Stubs

None. The package is fully functional — it renders real ANSI output with all required fields.

## Self-Check: PASSED

- [x] internal/statusbar/bar.go exists
- [x] internal/statusbar/bar_test.go exists
- [x] Commit aceba20 exists (feat: bar.go)
- [x] Commit 6c14feb exists (test: bar_test.go + getSize fix)
- [x] All 9 tests pass
- [x] go vet clean
