---
phase: 126-tui-write-parity-editor-shell-out
plan: 02
subsystem: tui
tags: [go, tui, bubbletea, editor, shell-out, temp-file, tdd]

# Dependency graph
requires:
  - phase: 126-01
    provides: "FilesClient 8-method interface (WriteFile, DeleteFile, RenameFile, MkdirFile)"
provides:
  - "resolveEditor() — EDITOR→VISUAL→nano→vim→vi chain via exec.LookPath"
  - "editFetchCmd — ReadFile → os.CreateTemp → filesEditReadyMsg"
  - "editWriteBackCmd — os.ReadFile(tmp) + WriteFile → filesOpMsg; defer os.Remove"
  - "filesEditReadyMsg / editorExitMsg / filesOpMsg message types"
  - "'e' key branch in handleFilesKey — bounds-check, dir no-op, locked error, generation++, editFetchCmd"
  - "applyFilesEditReadyMsg — tea.ExecProcess(exec.Command(editor, tmpPath)) suspend"
  - "applyEditorExitMsg — tea.ClearScreen + UNCONDITIONAL editWriteBackCmd + loadDirCmd"
  - "applyFilesOpMsg — stale-discard + toast + refresh"
affects: [126-03, 126-04, tui-write-parity-editor-shell-out]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "temp-file round-trip edit: ReadFile→CreateTemp→ExecProcess→ReadFile→WriteFile (local+remote parity)"
    - "tea.ExecProcess(exec.Command(editor, tmpPath)) for suspend/resume — no hand-rolled raw mode"
    - "Unconditional post-edit refresh: editorExitMsg always batches ClearScreen+writeBack+loadDir regardless of exitErr"
    - "All os.* I/O inside tea.Cmd closures — gate-safe (TestFiles_NoSyncFSCalls passes)"
    - "fmt.Sprintf('%T', msg()) for inspecting unexported tea message types in tests"

key-files:
  created:
    - internal/tui/files_edit.go
    - internal/tui/files_edit_test.go
  modified:
    - internal/tui/files.go
    - internal/tui/update.go

key-decisions:
  - "clearScreenMsg is in 'tea' package (not 'tui') — test type assertion must use 'tea.clearScreenMsg'"
  - "Write-back is unconditional in editorExitMsg (NOT gated on exitErr==nil) per TUIW-04 spec"
  - "All editor file I/O in files_edit.go, not files_cmds.go — keeps files.go/files_cmds.go free of os.CreateTemp/ReadFile/Remove while staying gate-safe"
  - "exec.Command(editor, tmpPath) as separate argv elements (T-126-04, no shell string injection)"
  - "applyFilesEditReadyMsg/applyEditorExitMsg/applyFilesOpMsg added as methods at end of update.go"

requirements-completed: [TUIW-02, TUIW-03, TUIW-04]

# Metrics
duration: 25min
completed: 2026-06-14
---

# Phase 126 Plan 02: $EDITOR Shell-Out via Temp-File Round-Trip Summary

**resolveEditor() + editFetchCmd/editWriteBackCmd + e key branch + editorExitMsg with unconditional tea.ClearScreen + write-back + loadDirCmd; all tests GREEN including the write-back-on-exitErr!=nil assertion (TUIW-02/03/04)**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-14T00:00:00Z
- **Completed:** 2026-06-14T00:25:00Z
- **Tasks:** 2 (TDD RED+GREEN each)
- **Files created:** 2 / **Files modified:** 2

## Accomplishments

### Task 1: resolveEditor() + edit tea.Cmds + msg types (files_edit.go)

- Created `internal/tui/files_edit.go` with:
  - `resolveEditor()` — iterates `[$EDITOR, $VISUAL, nano, vim, vi]`, returns first `exec.LookPath` success or ""
  - `filesEditReadyMsg` / `editorExitMsg` / `filesOpMsg` message types
  - `editFetchCmd` — nil-guard + `client.ReadFile` + `os.CreateTemp("", "agenthub-edit-*"+ext)` + write bytes; returns `filesEditReadyMsg`
  - `editWriteBackCmd` — `defer os.Remove(tmpPath)` + `os.ReadFile` + nil-guard + `client.WriteFile(relPath, data)`; returns `filesOpMsg`
  - All `os.*` I/O inside `func() tea.Msg` closures (gate-safe, T-126-07)
- TDD RED committed (`ac73b3f`), then GREEN (`58d19bc`)
- `TestResolveEditor` passes all three sub-tests (resolved binary, empty PATH, order assertion)

### Task 2: e key branch + editorExitMsg handler (files.go + update.go)

- `internal/tui/files.go`: added `case s == "e"` in `handleFilesKey` non-filter switch:
  - Bounds-check + `entry.IsDir` no-op guard
  - `resolveEditor()` → empty → set verbatim locked error, return nil cmd
  - `rel := joinDir(m.files.cwd, ansi.Strip(entry.Name))` + `m.files.generation++` + `editFetchCmd` dispatch
- `internal/tui/update.go`: added three handler methods:
  - `applyFilesEditReadyMsg` — on error: toast + refresh; on success: `exec.Command(editor, tmpPath)` → `tea.ExecProcess` with `editorExitMsg` callback
  - `applyEditorExitMsg` — UNCONDITIONAL `tea.Batch(tea.ClearScreen, editWriteBackCmd(...), loadDirCmd(...))` regardless of `exitErr`; toast on non-zero exit
  - `applyFilesOpMsg` — stale-discard (generation < current), toast on error, refresh on error
- TDD RED committed (`eba7810`), then GREEN (`7a087b3`)

## Task Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| Merge main | merge | chore | Merge main to get plan 01's 8-method FilesClient |
| Task 1 RED | `ac73b3f` | test | TestResolveEditor failing tests |
| Task 1 GREEN | `58d19bc` | feat | resolveEditor() + editFetchCmd/editWriteBackCmd + msg types |
| Task 2 RED | `eba7810` | test | TestHandleFilesKey_Edit + TestEditorExit_RefreshesUnconditionally failing |
| Task 2 GREEN | `7a087b3` | feat | e key branch + editorExitMsg handler |

## Acceptance Criteria Check

- [x] `go test ./internal/tui/ -run 'TestResolveEditor' -count=1` exits 0
- [x] `go test ./internal/tui/ -run 'TestHandleFilesKey_Edit|TestEditorExit_RefreshesUnconditionally' -count=1` exits 0
- [x] `go build ./internal/tui/...` exits 0
- [x] editorExitMsg test asserts write-back cmd present BOTH when exitErr==nil and exitErr!=nil
- [x] No-editor branch sets exact verbatim copy: `` "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." ``
- [x] `grep -v '^[[:space:]]*//' internal/tui/update.go | grep -c 'tea.ClearScreen'` returns 1
- [x] `gofmt -l internal/tui/files.go internal/tui/update.go` prints nothing
- [x] `go test ./internal/tui/ -run 'TestFiles_NoSyncFSCalls' -count=1` PASS
- [x] `go test -race ./internal/tui/...` green

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Worktree was behind main (missing plan 01's FilesClient changes)**
- **Found during:** Task 1 GREEN phase — `client.WriteFile undefined (type FilesClient has no field or method WriteFile)`
- **Fix:** `git merge main` — fast-forward with Plan 01 commits
- **Impact:** None to final code; merge was clean

**2. [Rule 1 - Bug] clearScreenMsg type name in test was wrong package**
- **Found during:** Task 2 GREEN phase — test looked for `tui.clearScreenMsg` but bubbletea's `ClearScreen` returns `tea.clearScreenMsg`
- **Fix:** Changed type assertion in `files_edit_test.go` to `"tea.clearScreenMsg"`
- **Impact:** None to production code; test assertion corrected

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes. Security requirements satisfied:
- T-126-04 (command injection): `exec.Command(editor, tmpPath)` passes tmpPath as a separate argv element, never via shell string
- T-126-05 (temp file cleanup): `defer os.Remove(tmpPath)` in `editWriteBackCmd` always runs
- T-126-06 (wrong path write-back): `relPath` carried through msg chain from `handleFilesKey`; never recomputed from tmpPath
- T-126-07 (sync I/O in Update): all `os.*` calls inside `tea.Cmd` closures in `files_edit.go`

## Known Stubs

None. All edit functionality fully wired: key → fetch → ExecProcess → write-back → refresh.

## Self-Check: PASSED
