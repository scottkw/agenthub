---
phase: 126-tui-write-parity-editor-shell-out
verified: 2026-06-14T00:00:00Z
status: human_needed
score: 5/5
overrides_applied: 0
human_verification:
  - test: "Live $EDITOR suspend-resume terminal restore"
    expected: "Pressing e on a file suspends the TUI, spawns $EDITOR with the temp file, on save+exit the terminal redraws cleanly (tea.ClearScreen) and the directory listing refreshes"
    why_human: "tea.ExecProcess suspend/resume requires a real interactive terminal; cannot be driven by go test"
  - test: "Two-machine remote write edit (optional — deferred to Phase 128)"
    expected: "Editing a file in a remote session via RemoteFilesClient.WriteFile round-trips correctly end-to-end"
    why_human: "Requires two networked hosts with Tailscale; Phase 128 is the stated gate for this"
---

# Phase 126: TUI Write Parity — $EDITOR Shell-Out Verification Report

**Phase Goal:** TUI users edit files via $EDITOR shell-out, delete, rename, create dirs via keyboard shortcuts — full cross-surface parity with GUI write ops, minus upload (descoped: message + GitHub issue).
**Verified:** 2026-06-14
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `e` resolves $EDITOR ($EDITOR→$VISUAL→nano→vim→vi), suspends TUI, resumes with tea.ClearScreen + unconditional write-back + listing refresh | VERIFIED | `resolveEditor()` in `files_edit.go:20-30` iterates exact candidate order; `applyEditorExitMsg` in `update.go:1027-1039` prepends `tea.ClearScreen`, appends `editWriteBackCmd` UNCONDITIONALLY (not gated on exitErr), then `loadDirCmd`; `TestHandleFilesKey_Edit` and `TestEditorExit_RefreshesUnconditionally` both pass (including the exitErr!=nil write-back sub-case) |
| 2 | No resolvable editor → inline error EXACTLY "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." — no crash | VERIFIED | `files.go:634` sets `m.files.err = errors.New("`+"`"+`$EDITOR`+"`"+` is not set. Set it in your shell profile (e.g. `+"`"+`export EDITOR=nano`+"`"+`)."`+"`"+`)`; `TestHandleFilesKey_Edit/e_with_no_editor_sets_exact_locked_error_and_returns_nil_cmd` passes |
| 3 | `d` confirm-recursive-delete / `r` inline rename / `m` inline mkdir — all refresh listing on completion | VERIFIED | `deleteCmd/renameCmd/mkdirCmd` in `files_cmds.go:153-193` dispatch via `FilesClient`; `applyFilesOpMsg` at `update.go:1047-1067` calls `loadDirCmd` on success for op!="edit"; `TestFilesDelete`, `TestFilesRename`, `TestFilesMkdir` all pass; `TestFilesDelete_DispatchPriority` confirms modal keys cannot leak to tab-cycling |
| 4 | `u` → "Use desktop or web to upload files." (exact) + GitHub issue #82 filed | VERIFIED | `uploadDescoped` const at `files.go:705`; `case s == "u"` at `files.go:641-646` sets `m.files.err` with no write cmd; `TestFilesUpload_Descoped` passes; `gh issue view 82 --repo scottkw/agenthub` confirms issue exists with title "TUI Files: file upload parity gap (descoped in v3.5 / Phase 126)" and body referencing TUIW-06 |
| 5 | FilesClient = exactly 8 methods (4 read + 4 write, NO UploadFile); both `*daemon.DaemonClient` and `*RemoteFilesClient` satisfy via compile-time `var _` guards; `TestFiles_NoSyncFSCalls` passes with write commands | VERIFIED | `files_client.go` declares exactly 8 methods (confirmed by method count: `grep -c "context.Context" = 8`); UploadFile absent (`grep -c = 0`); both guards present (`files_client.go:45` and `remote_files_client.go:41`); `go build ./internal/tui/...` exits 0; `TestFiles_NoSyncFSCalls` passes with broadened regex covering `os.(ReadDir|Open|OpenFile|Stat|Create|Remove|ReadFile|WriteFile)` |

**Score:** 5/5 truths verified (automated)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/tui/files_client.go` | 8-method FilesClient interface + daemon compile-time guard | VERIFIED | 8 methods, `var _ FilesClient = (*daemon.DaemonClient)(nil)` present, UploadFile absent |
| `internal/tui/remote_files_client.go` | WriteFile/DeleteFile/RenameFile/MkdirFile on RemoteFilesClient + RemoteFilesClient guard | VERIFIED | All 4 write methods implemented with CAP-LEAK invariant; `var _ FilesClient = (*RemoteFilesClient)(nil)` at line 41 |
| `internal/tui/remote_files_client_test.go` | httptest.TLSServer round-trip tests for 4 write methods | VERIFIED | `TestRemoteFilesClient_Write/Delete/Rename/Mkdir/WriteCapLeak` all pass |
| `internal/tui/files_edit.go` | resolveEditor() chain + editFetchCmd + editWriteBackCmd + msg types | VERIFIED | All present and substantive; all FS I/O inside `tea.Cmd` closures |
| `internal/tui/files.go` | `e`/`d`/`r`/`m`/`u` key branches + uploadDescoped const | VERIFIED | All 5 branches present; `uploadDescoped = "Use desktop or web to upload files."` |
| `internal/tui/update.go` | editorExitMsg + filesOpMsg handlers with ClearScreen + unconditional loadDirCmd; priority ladder | VERIFIED | `applyEditorExitMsg` at line 1027; `handleFileDeleteConfirmKey` at line 738; inline-input at Priority 2.6 above handleFilesKey |
| `internal/tui/files_cmds.go` | deleteCmd/renameCmd/mkdirCmd | VERIFIED | All 3 present with nil-guard + context timeout |
| `internal/tui/modal.go` | renderFileDeleteConfirmModal with colorblind-safe text | VERIFIED | "This cannot be undone." at line 261; modal text carries danger signal without relying on color alone |
| `internal/tui/model.go` | modalFileDeleteConfirm iota + fileDeleteTarget + filesNameInput state | VERIFIED | `modalFileDeleteConfirm` present; `fileDeleteTarget` field; `nameInput`/`nameInputMode`/`nameInputOriginal`/`nameInputActive` on filesModel |
| `internal/tui/files_test.go` | TestFilesUpload_Descoped + extended TestFiles_NoSyncFSCalls + TestFiles_Phase126_Requirements | VERIFIED | All 3 tests present and passing; TUIW-01..07 all mapped in the matrix |

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `files.go` e-branch | `editFetchCmd` | `resolveEditor()` + `generation++` | WIRED | `files.go:619-639` calls `editFetchCmd(m.files.client, ..., editor, m.files.generation)` after resolveEditor() check |
| `update.go` editorExitMsg handler | `tea.ClearScreen` + `editWriteBackCmd` + `loadDirCmd` | `tea.Batch`, unconditionally | WIRED | `applyEditorExitMsg` at `update.go:1027-1039`; `editWriteBackCmd` appended before exitErr check; both nil and non-nil exitErr paths confirmed by test |
| `editWriteBackCmd` | `client.WriteFile` | os.ReadFile(tmpPath) then WriteFile inside tea.Cmd closure | WIRED | `files_edit.go:118-133`; `client.WriteFile` called at line 131 |
| `files_client.go` | `*daemon.DaemonClient` | compile-time `var _ FilesClient` assertion | WIRED | `files_client.go:45`; build succeeds |
| `files.go` d/r/m branches | `deleteCmd/renameCmd/mkdirCmd` | modal confirm / inline input | WIRED | `files.go:576-618`; cmds dispatched from `handleFileDeleteConfirmKey` (line 738) and `handleFilesNameInputKey` (line 783) |
| `deleteCmd/renameCmd/mkdirCmd` | `client.DeleteFile/RenameFile/MkdirFile` | tea.Cmd closure | WIRED | `files_cmds.go:153-193`; all 3 confirmed |
| `update.go` priority ladder | `handleFileDeleteConfirmKey` at Priority 2.5, inline-input at 2.6 | before handleFilesKey at 5.5 | WIRED | `update.go:227-239`; `TestFilesDelete_DispatchPriority` passes |
| `files.go` u-branch | `uploadDescoped` const / status line (no write) | `m.files.err = errors.New(uploadDescoped)` | WIRED | `files.go:641-646`; no write cmd dispatched |

---

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `editFetchCmd` | `data []byte` | `client.ReadFile(ctx, sid, relPath)` | Yes — reads from live FilesClient | FLOWING |
| `editWriteBackCmd` | `data []byte` | `os.ReadFile(tmpPath)` then `client.WriteFile` | Yes — writes back to FilesClient | FLOWING |
| `deleteCmd` | `filesOpMsg{err}` | `client.DeleteFile(ctx, sid, relPath)` | Yes | FLOWING |
| `renameCmd` | `filesOpMsg{err}` | `client.RenameFile(ctx, sid, oldRel, newRel)` | Yes | FLOWING |
| `mkdirCmd` | `filesOpMsg{err}` | `client.MkdirFile(ctx, sid, relPath)` | Yes | FLOWING |

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| TestResolveEditor | `go test ./internal/tui/ -run TestResolveEditor -count=1` | PASS (3 sub-tests) | PASS |
| TestHandleFilesKey_Edit + TestEditorExit_RefreshesUnconditionally | `go test ./internal/tui/ -run 'TestHandleFilesKey_Edit|TestEditorExit_RefreshesUnconditionally' -count=1` | PASS (5 sub-tests including exitErr!=nil write-back) | PASS |
| TestRemoteFilesClient write methods | `go test ./internal/tui/ -run 'TestRemoteFilesClient_(Write|Delete|Rename|Mkdir)' -count=1` | PASS (5 tests including WriteCapLeak) | PASS |
| TestFilesDelete + TestFilesRename + TestFilesMkdir | `go test ./internal/tui/ -run 'TestFilesDelete|TestFilesRename|TestFilesMkdir' -count=1` | PASS (all sub-tests including DispatchPriority and ColorblindSafeText) | PASS |
| TestFilesUpload_Descoped + TestFiles_NoSyncFSCalls + TestFiles_Phase126_Requirements | `go test ./internal/tui/ -run 'TestFilesUpload_Descoped|TestFiles_NoSyncFSCalls|TestFiles_Phase126_Requirements' -count=1` | PASS (all 7 TUIW rows pass in requirements matrix) | PASS |
| Full race suite | `go test -race ./internal/tui/... ./internal/daemon/... -count=1` | PASS | PASS |
| Build | `go build ./internal/tui/... ./internal/daemon/...` | exit 0 | PASS |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| TUIW-01 | Plan 01 | FilesClient 8 methods; both implementers satisfy at compile time | SATISFIED | files_client.go 8 methods, both var _ guards, go build passes |
| TUIW-02 | Plan 02 | e key tea.Exec shell-out; write-back/refresh via tea.Cmd | SATISFIED | editFetchCmd/editWriteBackCmd in files_edit.go; TestHandleFilesKey_Edit pass |
| TUIW-03 | Plan 02 | resolveEditor() chain $EDITOR→$VISUAL→nano→vim→vi; no-editor error | SATISFIED | files_edit.go:20-30; exact error at files.go:634; TestResolveEditor pass |
| TUIW-04 | Plan 02 | tea.ClearScreen on exit; loadDirCmd unconditional | SATISFIED | applyEditorExitMsg batches ClearScreen first, write-back always; TestEditorExit_RefreshesUnconditionally exitErr!=nil case pass |
| TUIW-05 | Plan 03 | d/r/m delete+confirm, inline rename, inline mkdir | SATISFIED | deleteCmd/renameCmd/mkdirCmd; modal at Priority 2.5; inline input at 2.6; TestFilesDelete/Rename/Mkdir pass |
| TUIW-06 | Plan 04 | TUI upload descoped with "Use desktop or web..." + GitHub issue | SATISFIED | uploadDescoped const; case "u"; issue #82 verified via gh |
| TUIW-07 | Plan 04 | TestFiles_NoSyncFSCalls extended to cover write verbs | SATISFIED | Broadened regex at files_test.go:860; both gated files pass with zero matches |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | No TBD/FIXME/XXX markers in any phase-modified file | — | — |

gofmt clean: all 7 phase-modified source files pass `gofmt -l` with no output.

---

### Human Verification Required

#### 1. Live $EDITOR Suspend-Resume Terminal Restore

**Test:** Run the TUI (`wails dev` or build), open the Files view, navigate to a text file, press `e`. Confirm the TUI suspends cleanly, your $EDITOR opens with the file content. Edit and save. Confirm the terminal redraws without corruption (tea.ClearScreen) and the directory listing refreshes to show the saved file.

**Expected:** Clean suspend → editor session → clean resume + refreshed listing. No terminal artifacts.

**Why human:** `tea.ExecProcess` suspend/resume requires a real interactive terminal. The unit tests cover command dispatch and unconditional write-back but cannot exercise the actual tty interaction (tea.ExecProcess is a runtime terminal suspend, not unit-testable).

#### 2. Two-Machine Remote Write Edit (Deferred to Phase 128)

**Test:** From a remote TUI session (different machine, RemoteFilesClient active), press `e` on a file, edit it, save. Confirm the write-back reaches the remote host.

**Expected:** Edited bytes appear on the remote machine's filesystem.

**Why human:** Requires two networked hosts. VALIDATION.md explicitly defers this to Phase 128 as its stated gate. RemoteFilesClient write methods are unit-tested via httptest.TLSServer; the end-to-end two-machine path is the manual residue.

---

### Gaps Summary

No automated gaps. All 5 success criteria verified by codebase inspection and passing tests. The two human verification items above are the expected manual residue documented in VALIDATION.md — live terminal suspend/resume (inherently untestable by unit tests) and two-machine remote write (deferred to Phase 128 by design).

---

_Verified: 2026-06-14_
_Verifier: Claude (gsd-verifier)_
