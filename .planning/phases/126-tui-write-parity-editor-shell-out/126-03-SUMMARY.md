---
phase: 126-tui-write-parity-editor-shell-out
plan: 03
subsystem: tui
tags: [go, tui, bubbletea, delete, rename, mkdir, confirm-modal, inline-input, tdd]

# Dependency graph
requires:
  - phase: 126-02
    provides: "filesOpMsg, generation discipline, e key branch shape"
provides:
  - "deleteCmd — nil-guard + client.DeleteFile + filesOpMsg{op:'delete'}"
  - "renameCmd — nil-guard + client.RenameFile + filesOpMsg{op:'rename'}"
  - "mkdirCmd — nil-guard + client.MkdirFile + filesOpMsg{op:'mkdir'}"
  - "modalFileDeleteConfirm iota + fileDeleteTarget struct + fileDeleteFocusYes field"
  - "filesModel.nameInput + nameInputActive + nameInputMode + nameInputOriginal"
  - "handleFileDeleteConfirmKey — y/Enter-on-Yes→deleteCmd; n/esc→close (T-126-09)"
  - "handleFilesNameInputKey — Enter→renameCmd/mkdirCmd; esc→cancel; default→textinput"
  - "d/r/m key branches in handleFilesKey (files.go)"
  - "Priority 2.5/2.6 dispatch in handleKey — above handleFilesKey (T-126-10)"
  - "renderFileDeleteConfirmModal — 'Delete'+'cannot be undone' text (colorblind-safe)"
  - "applyFilesOpMsg success path — refresh listing for delete/rename/mkdir"
affects: [126-04, tui-write-parity-editor-shell-out]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Kill-confirm clone for file-delete: handleFileDeleteConfirmKey mirrors handleKillConfirmKey"
    - "Inline name-input: handleFilesNameInputKey mirrors handleRenameKey (session rename)"
    - "Dedicated filesModel.nameInput — NOT reusing editInput — avoids session-rename collision"
    - "Priority 2.5 (delete-confirm) + 2.6 (inline-input) above Priority 5.5 (handleFilesKey)"
    - "applyFilesOpMsg success refresh: op!='edit' unconditionally calls loadDirCmd"
    - "Colorblind-safe modal: explicit 'Delete'/'cannot be undone' TEXT; FgDanger is reinforcement"

key-files:
  created:
    - internal/tui/files_ops_test.go
  modified:
    - internal/tui/files_cmds.go
    - internal/tui/model.go
    - internal/tui/files.go
    - internal/tui/update.go
    - internal/tui/modal.go
    - internal/tui/view.go

key-decisions:
  - "fileDeleteFocusYes added to Model (not filesModel) to mirror killFocusYes pattern"
  - "fileDeleteTarget is *fileDeleteTarget on Model (not filesModel) for structural parity with killTarget"
  - "nameInput.Focus() returns a non-nil cmd from 'r'/'m' — tests updated to not assert nil"
  - "applyFilesOpMsg success path refreshes for op!='edit'; edit write-back already batched in applyEditorExitMsg"
  - "Priority 2.5 (delete-confirm) placed before Priority 3 (new-session) to sit at same tier as kill-confirm"
  - "renderFileDeleteConfirmModal renders '[ Delete ]' button label (not '[ Yes ]') for colorblind clarity"

requirements-completed: [TUIW-05]

# Metrics
duration: 40min
completed: 2026-06-14
---

# Phase 126 Plan 03: d Delete (confirm + recursive), r Rename, m Mkdir Summary

**deleteCmd/renameCmd/mkdirCmd + handleFileDeleteConfirmKey + handleFilesNameInputKey + d/r/m key branches at Priority 2.5/2.6 in handleKey; colorblind-safe renderFileDeleteConfirmModal; all tests GREEN (TUIW-05)**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-06-14T23:08:00Z
- **Completed:** 2026-06-14T23:48:00Z
- **Tasks:** 2 (TDD RED+GREEN each)
- **Files created:** 1 / **Files modified:** 6

## Accomplishments

### Task 1: deleteCmd/renameCmd/mkdirCmd + model state

- `internal/tui/files_cmds.go`: added three async tea.Cmd factories:
  - `deleteCmd` — nil-guard + `client.DeleteFile` + `filesOpMsg{op:"delete"}`
  - `renameCmd` — nil-guard + `client.RenameFile(oldRel, newRel)` + `filesOpMsg{op:"rename"}`
  - `mkdirCmd` — nil-guard + `client.MkdirFile` + `filesOpMsg{op:"mkdir"}`
  - All use `context.WithTimeout(10s)` and stamp sessionID + generation on result
- `internal/tui/model.go`:
  - Added `modalFileDeleteConfirm` iota to modalState
  - Added `fileDeleteTarget` struct with `relPath`, `isDir`, `name` fields
  - Added `fileDeleteTarget *fileDeleteTarget` and `fileDeleteFocusYes bool` to Model
- `internal/tui/files.go`:
  - Added `nameInputActive bool`, `nameInputMode string`, `nameInputOriginal string`, `nameInput textinput.Model` to filesModel
  - Initialized `nameInput` in `newFilesModelWithClient` (separate from filterInput)
- `internal/tui/modal.go`: added `renderFileDeleteConfirmModal` with colorblind-safe text

### Task 2: Key branches, handlers, priority dispatch

- `internal/tui/files.go` — added `d`/`r`/`m` cases in handleFilesKey:
  - `case s == "d"`: bounds-check, populate fileDeleteTarget, set modal=modalFileDeleteConfirm
  - `case s == "r"`: bounds-check, prefill nameInput with ansi.Strip(name), set nameInputMode="rename", focus
  - `case s == "m"`: empty nameInput, set nameInputMode="mkdir", focus
- `internal/tui/update.go`:
  - Added `handleFileDeleteConfirmKey` (clone of handleKillConfirmKey): y/Enter-on-Yes→executeFileDelete; n/esc→close modal; toggle focus
  - Added `executeFileDelete`: bumps generation, dispatches deleteCmd
  - Added `handleFilesNameInputKey` (clone of handleRenameKey): Enter→trim+empty-guard+no-op-guard+dispatch; esc→cancel; default→nameInput.Update
  - Priority 2.5 (`modalFileDeleteConfirm`) and 2.6 (`nameInputActive`) added in handleKey ABOVE handleFilesKey
  - `applyFilesOpMsg` success path updated: op!="edit" triggers unconditional `loadDirCmd` refresh
- `internal/tui/view.go`: added `modalFileDeleteConfirm` branch to renderFull

## Task Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| Task 1 RED | `bfe5db7` | test | Failing tests for deleteCmd/renameCmd/mkdirCmd + d/r/m handlers |
| Task 1 GREEN | `27a90d5` | feat | deleteCmd/renameCmd/mkdirCmd + model state + modal render |
| Task 2 GREEN | `99439e2` | feat | d/r/m branches + handlers + priority dispatch + view wiring |

## Acceptance Criteria Check

- [x] `go build ./internal/tui/...` exits 0
- [x] `grep -c 'func deleteCmd\|func renameCmd\|func mkdirCmd' internal/tui/files_cmds.go` returns 3
- [x] `grep -c 'modalFileDeleteConfirm' internal/tui/model.go` returns >0
- [x] Each new cmd guards with `isNilFilesClient` before the network call
- [x] `gofmt -l internal/tui/files_cmds.go internal/tui/model.go` prints nothing
- [x] `go test ./internal/tui/ -run 'TestFilesDelete|TestFilesRename|TestFilesMkdir' -count=1` exits 0
- [x] Delete-confirm priority sub-test asserts "y" under modalFileDeleteConfirm is consumed before tab-cycling
- [x] `grep -c 'This cannot be undone' internal/tui/modal.go` returns >0
- [x] Rename empty-guard and no-op-equals-original guard covered by tests
- [x] `gofmt -l internal/tui/files.go internal/tui/update.go internal/tui/modal.go` prints nothing
- [x] `go test -race ./internal/tui/...` green

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test assertions for `r` and `m` expected nil cmd but textinput.Focus() returns non-nil**
- **Found during:** Task 2 GREEN — tests for "r sets rename mode" and "m sets mkdir mode" asserted `cmd == nil`
- **Issue:** `textinput.Focus()` returns a non-nil cmd; this matches the existing session-rename pattern (update.go:449 also returns the Focus cmd)
- **Fix:** Updated test assertions to ignore the focus cmd with `_` (accepted behavior, matches existing patterns)
- **Files modified:** `internal/tui/files_ops_test.go`
- **Impact:** None to production code; test assertions corrected

### Deviations from Plan Structure

**1. `fileDeleteTarget` placed on `Model` (not `filesModel`)**
- Plan suggested fileDeleteTarget on filesModel but structural parity with `killTarget` (on Model, not sessions Model) placed it on Model
- This is the correct design: the delete-confirm modal is a top-level modal, not a files-pane sub-state

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes beyond those declared in the plan's threat model.

Security requirements satisfied:
- T-126-08 (path traversal): `joinDir(cwd, name)` constructs sandbox-relative paths; no absolute or `../` construction client-side
- T-126-09 (accidental delete): `d` always shows confirmation modal; single keypress never calls DeleteFile
- T-126-10 (stray key leaking): Priority 2.5/2.6 handlers sit above handleFilesKey (5.5); priority sub-tests enforce it
- T-126-12 (sync I/O): All delete/rename/mkdir I/O inside `tea.Cmd` closures; no sync calls in Update

## Known Stubs

None. All three operations fully wired: key → modal/input → confirm → cmd → applyFilesOpMsg → loadDirCmd refresh.

## Self-Check: PASSED
