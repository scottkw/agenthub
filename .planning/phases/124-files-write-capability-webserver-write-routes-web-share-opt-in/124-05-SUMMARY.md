---
phase: 124-files-write-capability-webserver-write-routes-web-share-opt-in
plan: 05
subsystem: TUI files view — home-dir write warning (CAP-06 parity)
tags: [tui, colorblind-safe, warning, cap-06, parity, go, tdd]
requirements: [CAP-06]

dependency_graph:
  requires:
    - internal/daemon/types.go (SessionInfo.HomeDir + SessionInfo.FilesWrite — plan 124-02)
    - internal/tui/files.go (renderFilesTab, renderFilesStatusLine, truncate — plan 121)
    - internal/tui/styles.go (StatusWaiting amber token — plan 121)
  provides:
    - homeDirWriteWarning const (verbatim 124-UI-SPEC TUI copy)
    - renderFilesTab CAP-06 warning line (HomeDir && FilesWrite gate, StatusWaiting amber)
    - TestRenderFilesTab_HomeDirWarning_* (4 tests: positive + 3 negative cases)
  affects:
    - internal/daemon/types.go (HomeDir/FilesWrite fields aligned with 124-02)
    - internal/tui/files.go (warning line in renderFilesTab)
    - internal/tui/files_test.go (4 new warning tests)

tech_stack:
  added: []
  patterns:
    - TDD RED/GREEN: test written first, implementation second
    - lipgloss.NewStyle().Foreground(StatusWaiting).Width(cw).Render(truncate(..., cw)) — mirrors renderFilesStatusLine pattern
    - Iterate m.sessions by ID to find active session signal (same pattern as GUI banner reads daemon response)

key_files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/tui/files.go
    - internal/tui/files_test.go

decisions:
  - "SessionInfo.HomeDir and FilesWrite added to types.go initially as Rule 3 blocker (fields required to compile TUI code); later aligned to match canonical 124-02 version (no omitempty, canonical comments)"
  - "Warning line inserted as a variadic rows slice in renderFilesTab rather than a fixed JoinVertical — cleaner conditional insertion without a sentinel empty string"
  - "homeDirWriteWarning extracted as a package-level const so grep-verifiable and test-assertable without string duplication"

metrics:
  duration: "~20 minutes"
  completed: "2026-06-14"
  tasks_completed: 2
  files_changed: 3
---

# Phase 124 Plan 05: TUI Home-Dir Write Warning (CAP-06) Summary

**One-liner:** Colorblind-safe TUI warning line in renderFilesTab using the verbatim UI-SPEC copy, gated on SessionInfo.HomeDir && SessionInfo.FilesWrite for cross-surface parity with the GUI banner.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 0 | Write Wave-0 failing TUI warning render test (RED) | 3cd5e1a | internal/daemon/types.go, internal/tui/files_test.go |
| 1 | Render home-dir write warning line in renderFilesTab (GREEN) | c1e43fd | internal/tui/files.go |
| fix | Align types.go with 124-02 canonical version | 2a30fb3 | internal/daemon/types.go |

## What Was Built

### Task 0 (RED)

Added 4 tests to `internal/tui/files_test.go` under the `homeDirWarningModel` harness helper:

- `TestRenderFilesTab_HomeDirWarning_ShownWhenBothTrue` — asserts verbatim warning text, the glyph, and literal "Warning:" token all appear when HomeDir=true AND FilesWrite=true.
- `TestRenderFilesTab_HomeDirWarning_AbsentWhenFilesWriteFalse` — warning absent when FilesWrite=false.
- `TestRenderFilesTab_HomeDirWarning_AbsentWhenHomeDirFalse` — warning absent when HomeDir=false.
- `TestRenderFilesTab_HomeDirWarning_AbsentWhenBothFalse` — warning absent when both false.

Also added `HomeDir bool` and `FilesWrite bool` fields to `SessionInfo` in `internal/daemon/types.go` (Rule 3 blocker; later aligned to canonical 124-02 format).

### Task 1 (GREEN)

In `internal/tui/files.go`:

- Added `homeDirWriteWarning` const with the verbatim 124-UI-SPEC copy.
- Modified `renderFilesTab` to build the output as a slice of rows and conditionally insert the warning line between the body panes and status line when `HomeDir && FilesWrite`.
- The warning uses `lipgloss.NewStyle().Foreground(m.styles.StatusWaiting).Width(cw).Render(truncate(..., cw))` — same pattern as `renderFilesStatusLine`.
- Condition mirrors GUI banner: both gate on `SessionInfo.HomeDir && SessionInfo.FilesWrite` (cross-surface parity, T-124-18 mitigation).

## Verification

- `go test -race ./internal/tui/ -count=1` — green
- `grep -F 'Warning: cwd is $HOME' internal/tui/files.go` — matches
- `grep -F` for the warning glyph in `internal/tui/files.go` — matches
- `go vet ./internal/tui/` — clean
- `gofmt -l` — no diffs
- `go build ./...` — clean

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Added HomeDir/FilesWrite to SessionInfo in types.go**
- **Found during:** Task 0 (RED test writing)
- **Issue:** This worktree forked from main before the 124-02 merge. `SessionInfo` lacked `HomeDir` and `FilesWrite` fields.
- **Fix:** Added both fields; follow-up commit aligned to canonical 124-02 version (no `omitempty`, correct comments, added `IssueCapabilitiesResponse.HomeDir`). On merge, this worktree's types.go change will be identical to main — no conflict.
- **Files modified:** `internal/daemon/types.go`
- **Commits:** 3cd5e1a (initial), 2a30fb3 (alignment)

## TDD Gate Compliance

- RED gate: `test(124-05): add failing TUI home-dir write warning render test (RED)` (3cd5e1a)
- GREEN gate: `feat(124-05): render home-dir write warning line in TUI Files view (GREEN)` (c1e43fd)
- REFACTOR: not required

## Known Stubs

None. The warning line reads live server-computed signals from `m.sessions`.

## Threat Flags

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Warning is a pure render-side read of existing server signals.

## Self-Check: PASSED

Files created/modified:
- FOUND: internal/daemon/types.go
- FOUND: internal/tui/files.go
- FOUND: internal/tui/files_test.go

Commits:
- FOUND: 3cd5e1a (test(124-05))
- FOUND: c1e43fd (feat(124-05))
- FOUND: 2a30fb3 (fix(124-05))
