---
phase: 144-daemon-styled-tail-race-fix
plan: 01
subsystem: daemon
tags: [race-fix, concurrency, vt-emulator, styled-tail, ci-red]
dependency_graph:
  requires: []
  provides: [FIX-01]
  affects: [internal/daemon/engine.go, internal/daemon/engine_test.go, TESTING.md]
tech_stack:
  added: []
  patterns: [strip-then-synchronous-drive, package-level-regexp, subtractive-fix]
key_files:
  created: []
  modified:
    - internal/daemon/engine.go
    - internal/daemon/engine_test.go
    - TESTING.md
decisions:
  - "Strip query sequences pre-Write (Option 2) instead of bumping vt (Option 1 dead — newest vt still has unsynchronized closed field)"
  - "queryStripPattern covers 9 query verbs: DA1/DA2/DSR5/DSR6/DECXCPR/DECRQM-ANSI/DECRQM-DEC/OSC-color/mode-2048"
  - "Removed drain goroutine entirely — race is structurally impossible with no concurrent Read/Close"
  - "Removed io import — only use was io.Copy in the now-gone drain goroutine"
metrics:
  duration: 201s
  completed: 2026-06-22
  tasks_completed: 3
  files_modified: 3
---

# Phase 144 Plan 01: Daemon Styled-Tail Race Fix Summary

**One-liner:** Synchronous styled-tail drive via queryStripPattern pre-Write strip, eliminating the concurrent vt Read/Close data race (#100) without bumping the library.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Strip query sequences pre-Write, remove drain goroutine | 1376705e | internal/daemon/engine.go |
| 2 | Add AllQueriesNoHang broadened query fixture | 418bb93b | internal/daemon/engine_test.go |
| 3 | Add FIX-01 traceability row to TESTING.md | 848647fd | TESTING.md |

## What Was Built

**engine.go changes (subtractive fix):**
- Added package-level `queryStripPattern` regexp covering all 9 query verbs that elicit blocking writes into charmbracelet/x/vt's unbuffered response pipe: DA1 (`ESC[c`), DA2 (`ESC[>c`), DSR (`ESC[5n/6n`), DECXCPR (`ESC[?6n`), DECRQM ANSI (`ESC[...$p`), DECRQM DEC (`ESC[?...$p`), OSC color queries (`ESC]10|11|12;?`), and DEC mode 2048 set/reset (`ESC[?2048h/l`)
- Replaced 6-line goroutine block (drainDone channel + `go func() io.Copy(io.Discard, emu)` + Write + Close + wait) with 3-line synchronous block: `clean := queryStripPattern.ReplaceAll(stripped, nil)` + `emu.Write(clean)` + `_ = emu.Close()`
- Removed `"io"` from import block (only use was `io.Copy` in the removed goroutine)
- Net change: -1 goroutine, -1 channel, -1 import, +1 package-level regexp, +1 ReplaceAll call

**engine_test.go changes:**
- Added `TestGetSessionStyledTailLines_AllQueriesNoHang` — kitchen-sink fixture containing all 9 query verbs interleaved with 10 visible anchor words ("alpha"…"kappa"). Asserts (1) call completes within 5s timeout and (2) all anchor words survive in the rendered grid.

**TESTING.md changes:**
- Added Section 4 traceability row for FIX-01 pointing at `internal/daemon/engine_test.go`

## Verification Results

All 5 phase-gate checks passed:
1. `go build ./...` exits 0 — no unused-io-import compile error
2. `go test -race -short ./internal/daemon/ -count=1` exits 0 — no "DATA RACE" in output
3. `go test ./internal/daemon/ -run 'TestGetSessionStyledTailLines' -count=1` exits 0 — rendering preserved (color/bold + CR-overwrite collapse)
4. `bash tests/check-traceability-paths.sh` exits 0 — FIX-01 path valid
5. `gofmt -l internal/daemon/engine.go internal/daemon/engine_test.go` — empty output (formatted)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All logic is wired.

## Threat Flags

None. The change is strictly subtractive on emulator input — removes response-eliciting triggers and a goroutine. No new network surface, auth paths, file access patterns, or schema changes introduced.

## Self-Check: PASSED

Files exist:
- [x] internal/daemon/engine.go — contains `queryStripPattern` (line 574), `queryStripPattern.ReplaceAll` (line 678), no `drainDone`, no `io.Copy(io.Discard, emu)`
- [x] internal/daemon/engine_test.go — contains `TestGetSessionStyledTailLines_AllQueriesNoHang`
- [x] TESTING.md — contains `FIX-01` row at line 146

Commits exist:
- [x] 1376705e — Task 1 engine.go fix
- [x] 418bb93b — Task 2 test fixture
- [x] 848647fd — Task 3 TESTING.md traceability
