---
phase: 126-tui-write-parity-editor-shell-out
reviewed: 2026-06-14T00:00:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - internal/tui/files_edit.go
  - internal/tui/files_client.go
  - internal/tui/remote_files_client.go
  - internal/tui/files_cmds.go
  - internal/tui/files.go
  - internal/tui/update.go
  - internal/tui/modal.go
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 126: Code Review Report

**Reviewed:** 2026-06-14
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Reviewed the TUI write-parity phase: `$EDITOR` shell-out, `d`/`r`/`m` command dispatch, delete-confirm flow, and the `RemoteFilesClient` write methods. The command-injection surface is handled correctly — the editor is resolved via `exec.LookPath` and spawned with `exec.Command(editor, tmpPath)` so `tmpPath` is a distinct argv element and never enters a shell. Temp-file cleanup is correct on every error path, and `os.CreateTemp` produces 0600-mode files by default, so no sensitive-bytes-in-/tmp exposure. The cap-leak invariant holds in `remote_files_client.go`: every error string interpolates only `(statusCode, body)`, never the URL. All write I/O stays inside `tea.Cmd` closures — no synchronous filesystem calls leak into `Update`.

The one BLOCKER is a generation-staleness bug that silently swallows edit write-back **failures**, defeating the entire stated purpose of the unconditional write-back (TUIW-04: prevent silent data loss). There are also three quality WARNINGs worth fixing.

## Critical Issues

### CR-01: Edit write-back errors are silently discarded as stale — user believes a failed save succeeded

**File:** `internal/tui/update.go:1027-1039` (interacting with `applyFilesOpMsg` at `1047-1067`)
**Issue:**
`applyEditorExitMsg` batches the write-back cmd stamped with `msg.generation`, then bumps `m.files.generation++` **before** the write-back result returns, then batches `loadDirCmd` with the new generation:

```go
cmds = append(cmds, editWriteBackCmd(m.files.client, msg.sessionID, msg.relPath, msg.tmpPath, msg.generation))
m.files.generation++   // <-- bumped before write-back result lands
cmds = append(cmds, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation))
```

`editWriteBackCmd` returns `filesOpMsg{op:"edit", generation: msg.generation}`. By the time that message reaches `applyFilesOpMsg`, `m.files.generation` has already been incremented, so:

```go
if msg.generation < m.files.generation {
    return m, nil // stale — discard   <-- ALWAYS taken for the edit write-back
}
```

The edit write-back `filesOpMsg` is **always** discarded as stale. That is harmless for the success case (op=="edit" intentionally does nothing on success), but it also discards the **error toast** on `msg.err != nil`. If the write-back fails — daemon rejects the path, file exceeds the write cap, permission denied, protected-file block in `$HOME`, or a remote HTTPS/cap error — the user sees no error. Worse, the unconditional `loadDirCmd` refresh repaints the listing as if nothing happened, so the stale on-disk file looks normal. The user closes their editor believing the edit saved when it was rejected. This is precisely the silent-data-loss class TUIW-04 was written to prevent.

The existing tests (`files_edit_test.go`) only assert the write-back cmd is *dispatched*; none assert its *error* is surfaced, so the bug is uncaught.

**Fix:** Do not bump the generation between dispatching the write-back and its result, OR special-case `op=="edit"` so its error is surfaced regardless of staleness. Simplest correct fix — refresh the listing only after the write-back result, not eagerly:

```go
func (m Model) applyEditorExitMsg(msg editorExitMsg) (tea.Model, tea.Cmd) {
    cmds := []tea.Cmd{tea.ClearScreen}
    if msg.exitErr != nil {
        m.toast = fmt.Sprintf("Editor exited with error: %s", msg.exitErr)
        m.toastKind = toastError
        m.toastExp = time.Now().Add(3 * time.Second)
    }
    // Dispatch write-back stamped with the SAME generation the op msg will carry.
    // Do NOT bump generation here; let applyFilesOpMsg drive the refresh so the
    // write-back error is not discarded as stale.
    cmds = append(cmds, editWriteBackCmd(m.files.client, msg.sessionID, msg.relPath, msg.tmpPath, msg.generation))
    return m, tea.Batch(cmds...)
}
```

Then in `applyFilesOpMsg`, refresh the listing for `op=="edit"` too (it currently early-returns on success):

```go
// Success: refresh listing for ALL ops including edit write-back so the new
// mtime/size is reflected. (Remove the `op != "edit"` exclusion.)
m.files.generation++
return m, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation)
```

This keeps the write-back error visible (the `msg.err != nil` branch in `applyFilesOpMsg` now fires) while still refreshing on success.

## Warnings

### WR-01: Edited file content is fully discarded when the temp file cannot be read back

**File:** `internal/tui/files_edit.go:118-134`
**Issue:**
`editWriteBackCmd` does `defer os.Remove(tmpPath)` first, then `os.ReadFile(tmpPath)`. If `os.ReadFile` fails (e.g. the editor replaced the file via rename-and-swap and left a different inode, a transient I/O error, or the editor deleted it), the function returns a `filesOpMsg` error and the deferred `os.Remove` destroys whatever the user just edited. There is no recovery path and — compounded by CR-01 — the error itself is then discarded. The user's edits are unrecoverable. Cleanup-on-read-failure is the right default for the success path, but combined with the silent-discard bug it turns a transient read hiccup into total edit loss.
**Fix:** At minimum, only remove the temp file after a *successful* read, and on read failure leave the file in place and surface a message telling the user where their edited content is parked:

```go
data, rerr := os.ReadFile(tmpPath)
if rerr != nil {
    // Do NOT remove — preserve the user's edits and tell them where they are.
    return filesOpMsg{sessionID: sid, generation: gen, op: "edit",
        err: fmt.Errorf("could not read edited temp file (your edits are at %s): %w", tmpPath, rerr)}
}
defer os.Remove(tmpPath) // only after a successful read
```

### WR-02: Rename/mkdir name input accepts path separators and `..`, enabling unintended cross-directory operations

**File:** `internal/tui/update.go:783-822` (`handleFilesNameInputKey`), `internal/tui/files.go:434-439` (`joinDir`)
**Issue:**
The inline rename/mkdir input only rejects empty/whitespace names. A value like `../escape` or `sub/dir` is passed straight into `joinDir(m.files.cwd, name)` and then to `renameCmd`/`mkdirCmd`. `joinDir` uses `path.Join`, which collapses `..` segments — so a rename to `../foo` produces a target *outside* the current cwd (still inside the sandbox if the daemon re-validates, but not where the user is looking). The server is the real security boundary, so this is not a sandbox escape, but it is a correctness/UX defect: a user typing a name with a slash silently moves the entry to a different directory than the one displayed, and there is no inline validation feedback. The session-rename path has the same shape but only affects a label; here it affects real filesystem layout.
**Fix:** Reject names containing path separators or `..` before dispatch, with a toast:

```go
if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
    m.toast = "Name cannot contain path separators"
    m.toastKind = toastError
    m.toastExp = time.Now().Add(2 * time.Second)
    return m, nil
}
```

### WR-03: Delete/rename/mkdir dispatch reads cwd-relative path but does not re-validate selection against a possibly-changed listing

**File:** `internal/tui/files.go:576-593` (`d`), `internal/tui/update.go:762-775` (`executeFileDelete`)
**Issue:**
The delete target (`relPath`, `isDir`, `name`) is captured into `m.fileDeleteTarget` when `d` is pressed, but the actual `deleteCmd` is dispatched later from `executeFileDelete` after the confirm modal. Between capture and confirm, an in-flight `loadDirCmd` (e.g. the periodic refresh, or the `e`/edit refresh) can land via `applyFilesListMsg` and replace `m.files.entries` and reset `m.files.selected = 0`. The captured `m.fileDeleteTarget.relPath` is a stable string so the *correct path* is still deleted (good), but the confirm modal's displayed name can now point at a different entry than what is highlighted in the (refreshed) list — the user may confirm a delete believing it targets the now-highlighted row. Recursive directory delete (`isDir` → server `RemoveAll`) makes a mistaken confirm expensive.
**Fix:** This is acceptable if the capture-at-keypress semantics are intentional and documented, but the modal should make the target unambiguous. The modal already shows the captured `name` (modal.go:248-252) which is correct — verify the list selection is visually frozen or de-emphasized while the confirm modal is open, OR re-assert that the highlighted entry still matches `fileDeleteTarget.relPath` before executing, refusing if the listing changed:

```go
// In executeFileDelete, optionally guard against a listing that moved under us:
// (defensive — the captured relPath is still correct, but confirms the user's intent)
```
At minimum, document the capture-at-keypress contract so a future maintainer does not "fix" it by re-reading the selection at confirm time (which would reintroduce a TOCTOU against the refreshed list).

## Info

### IN-01: `redactCapFromURL` is defined but never called — dead defensive code

**File:** `internal/tui/remote_files_client.go:107-118`
**Issue:** `redactCapFromURL` is documented as defense-in-depth "for any future error path," but no current code path calls it (all error strings already interpolate only status+body). Unused code drifts out of sync with its intent and can give false confidence. The compile-time `var _` guard pattern is used elsewhere; an unused helper is weaker.
**Fix:** Either wire it into the error paths that could plausibly leak a URL (none currently do) or remove it and rely on the documented "(status, body) only" invariant. If kept intentionally for future use, add a test that exercises it so it cannot silently rot.

### IN-02: `filesErrMsg` type is declared but never emitted

**File:** `internal/tui/files_cmds.go:72-78`
**Issue:** Comment says "reserved for Plan 02 … Plan 01 does not emit it." Phase 126 is well past Plan 01/02; the type is still unused. Dead reserved type.
**Fix:** Remove `filesErrMsg` if no consumer materialized, or wire it in. Carrying speculative types adds maintenance surface. (Pre-existing, not introduced by this diff — flagged for cleanup, not as a phase-126 regression.)

### IN-03: Magic timeout constants duplicated across every Cmd factory

**File:** `internal/tui/files_cmds.go` (5s/10s literals at lines 92, 113, 135, 158, 172, 186) and `internal/tui/files_edit.go:78, 129` (10s/15s)
**Issue:** Per-operation timeouts are scattered inline literals (`5*time.Second`, `10*time.Second`, `15*time.Second`). The rationale for each value lives only in comments. A future change to the daemon's write cap or remote RTT budget would require hunting each literal.
**Fix:** Hoist to named constants (e.g. `filesListTimeout`, `filesReadTimeout`, `filesWriteTimeout`) so the budgets are centralized and self-documenting. Low priority — current values are individually justified in comments.

---

_Reviewed: 2026-06-14_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
