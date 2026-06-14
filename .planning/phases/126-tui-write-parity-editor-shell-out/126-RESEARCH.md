# Phase 126: TUI Write Parity (`$EDITOR` Shell-Out) - Research

**Researched:** 2026-06-14
**Domain:** Bubble Tea v2 TUI write operations — `$EDITOR` shell-out (suspend/resume), delete/rename/mkdir affordances, `FilesClient` interface extension, cross-surface parity with GUI write ops
**Confidence:** HIGH (all integration claims verified against live source; tea.Exec/ExecProcess API confirmed in `bubbletea/v2@v2.0.6`)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- `e` on a selected file → suspend TUI, spawn resolved `$EDITOR` (fallback chain `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi`) with the sandbox-absolute path, resume on exit. Terminal restored via `tea.ClearScreen`; directory listing refreshes unconditionally after every edit.
- No editor resolvable → inline error: ``"`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)."`` (not a crash / silent no-op).
- `d` → confirmation dialog (reuse kill-session pattern); recursive delete for dirs. `r` → inline rename. `m` → inline mkdir. Both refresh listing on completion.
- `u` (upload) → on-screen "Use desktop or web to upload files." (the one documented parity gap) + file a follow-up GitHub issue.
- `FilesClient` interface = exactly 8 methods (4 read + 4 write); both `*daemon.DaemonClient` and `*tui.RemoteFilesClient` satisfy it. `TestFiles_NoSyncFSCalls` static-grep gate passes with write commands — all write FS I/O via `tea.Cmd`, never synchronous in `Update`.

### Claude's Discretion
Command/keybinding wiring details, confirm-dialog reuse, inline-input component reuse — at Claude's discretion guided by existing TUI Files view patterns.

### Deferred Ideas (OUT OF SCOPE)
TUI file upload — formally descoped this milestone (on-screen message + GitHub issue per SC#4).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TUIW-01 | `FilesClient` grows 4→8 methods (`WriteFile`, `DeleteFile`, `RenameFile`, `MkdirFile`); both `*daemon.DaemonClient` and `*tui.RemoteFilesClient` satisfy all 8 | §"FilesClient Interface Extension" — exact signatures + the critical return-type mismatch with DaemonClient. RemoteFilesClient needs 4 new write methods. |
| TUIW-02 | TUI edit on `e` uses `tea.Exec` to suspend, spawn `$EDITOR` with the file path, resume; write-back/refresh via `tea.Cmd` (never sync I/O in `Update`) | §"Edit Path Design" — RESOLVED design question (read-to-temp + WriteFile-back, uniform for local+remote). §"tea.Exec / ExecProcess" — confirmed API + `attachCmd` reference. |
| TUIW-03 | `resolveEditor()` chain (`$EDITOR`→`$VISUAL`→`nano`→`vim`→`vi`); else clear error | §"resolveEditor()" — `exec.LookPath` pattern + no-editor error copy. |
| TUIW-04 | `tea.ClearScreen` on editor exit; `loadDirCmd` runs unconditionally post-exec | §"tea.Exec / ExecProcess" + §"Common Pitfalls" P3. `attachDoneMsg` handler at update.go:81 is the reference. |
| TUIW-05 | Delete (`d` + confirm reusing kill-session pattern), rename (`r` inline), mkdir (`m` inline) | §"Reusable UI Patterns" — kill-confirm modal + inline-rename textinput, both verified in source. |
| TUIW-06 | Upload formally descoped: on-screen "Use desktop or web to upload files." + GitHub issue | §"Upload Descope". Mirrors the existing binary-preview refusal copy convention. |
| TUIW-07 | `TestFiles_NoSyncFSCalls` extended to new write commands — all write FS I/O via `tea.Cmd` | §"The No-Sync FS Gate" — exact gate location, regex, and how the temp-file write must be placed to NOT trip it. |
</phase_requirements>

## Summary

Phase 126 is a low-risk, pattern-following phase: every mechanism it needs already exists in the live codebase. The `$EDITOR` shell-out reuses the exact `tea.ExecProcess`/`attachCmd` suspend-resume pattern proven in `internal/tui/attach.go` + `update.go:366`. Delete reuses the `modalKillConfirm` dialog (update.go:669 `handleKillConfirmKey`). Rename and mkdir reuse the inline-textinput pattern from `handleRenameKey` (update.go:639). The `DaemonClient` write methods (`WriteFile`/`DeleteFile`/`RenameFile`/`MkdirFile`) already exist from Phase 123. The `FilesClient` interface was deliberately left at 4 methods by Phase 123 (scope guard T-123-19) for this phase to extend.

**The one consequential design decision** — whether `e` edits the sandbox-absolute path on disk or round-trips through `ReadFile`→temp→`WriteFile` — is RESOLVED by the milestone architecture research (ARCHITECTURE.md §5.3, §7.3; SUMMARY.md §"Architecture Approach"): **use the read-to-temp + WriteFile-back path.** It works uniformly for local AND remote sessions (the temp file is always local; the editor always runs locally; only the byte transport differs), it keeps all FS write I/O routed through `tea.Cmd` (satisfying the no-sync gate), and it does NOT require the TUI to learn the session's absolute working directory — which it currently has no client method to obtain. This finding directly contradicts a literal reading of the CONTEXT/ROADMAP success-criterion phrase "spawn `$EDITOR` with the sandbox-absolute path"; see §"Edit Path Design" for why the temp-file design is the correct interpretation and the one open question it raises.

**Primary recommendation:** Implement edit as `ReadFile → write bytes to a host-local temp file → tea.ExecProcess(exec.Command(editor, tmpPath)) → on exit, read temp file via tea.Cmd → WriteFile back → tea.ClearScreen + loadDirCmd unconditionally`. Extend `FilesClient` to 8 methods using the DaemonClient signatures verbatim (note the response-struct return types, not `error`-only). Add the 4 write methods to `RemoteFilesClient` mirroring its existing read methods. Reuse kill-confirm for delete and inline-rename textinput for rename/mkdir.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `$EDITOR` process spawn + terminal suspend/resume | TUI (Bubble Tea) | — | `tea.ExecProcess` hands the terminal to a child process; this is purely a client-tier concern, identical to PTY attach. |
| Reading file bytes for edit | TUI → DaemonClient/RemoteFilesClient | Daemon/Remote webserver | Bytes come over the existing read transport; the editor never touches the sandbox directly. |
| Writing edited bytes back | TUI → DaemonClient/RemoteFilesClient | Daemon (`Handler.Write`, atomic temp+rename) | The sandbox boundary + atomic write live server-side (Phase 123); TUI is a transport client only. |
| Delete / rename / mkdir | TUI → DaemonClient/RemoteFilesClient | Daemon (`Handler.Delete/Rename/Mkdir`) | Same as write — TUI dispatches, server enforces sandbox + denylist. |
| Confirm dialog / inline input | TUI | — | Pure UI state in the `Model`; no server involvement until the action `tea.Cmd` fires. |
| Editor resolution (`$EDITOR` chain) | TUI (host process env) | — | `os.Getenv` + `exec.LookPath` against the local machine running the TUI — correct even for remote sessions (editor is always local). |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/bubbletea/v2` | v2.0.6 | TUI runtime; `tea.Exec`/`tea.ExecProcess` suspend-resume, `tea.ClearScreen`, `tea.Cmd` async dispatch | [VERIFIED: go.mod + GOMODCACHE inspection] Already in use; `attachCmd` proves the exact pattern. |
| `charm.land/bubbles/v2/textinput` | (transitive) | Inline rename/mkdir name input | [VERIFIED: codebase grep] Already used by `editInput`, `filterInput`, join-code prompt. |
| `os/exec` (stdlib) | Go 1.24+ | `exec.LookPath` (editor resolution), `exec.Command` (editor process) | [CITED: pkg.go.dev/os/exec] Stdlib; no new dependency. |
| `os` (stdlib) | Go 1.24+ | `os.Getenv("EDITOR"/"VISUAL")`, `os.CreateTemp`, `os.ReadFile`, `os.Remove` for the temp file | [CITED: pkg.go.dev/os] Stdlib. NOTE: these go in the `ExecCommand.Run()` body / a `tea.Cmd` closure, NOT in `Update` — see no-sync gate. |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/scottkw/agenthub/internal/files` | local | `FileEntry`, `FileWriteResponse`, `FileOpResponse` wire types referenced by the extended interface | The `FilesClient` interface signatures reference `files.FileWriteResponse` / `files.FileOpResponse`. |
| `github.com/scottkw/agenthub/internal/daemon` | local | `DaemonClient` (already implements the 4 write methods) | Local-session write transport. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Read-to-temp + WriteFile-back | Edit sandbox-absolute path in place (local sessions only) | Would require a new `DaemonClient.GetSessionWorkDir`-style method (does not exist today — work dir resolution is server-side only, api.go:85). Would also break remote parity (no absolute path on a remote host) and risk tripping the no-sync gate if the abs path is opened directly. REJECTED per ARCHITECTURE.md §5.3. |
| `tea.ExecProcess(exec.Command(...))` | Custom `tea.ExecCommand` impl (like `attachCmd`) | `attachCmd` exists because PTY attach needs custom stdin/stdout/raw-mode wiring. A plain editor spawn needs none of that — `tea.ExecProcess` wrapping `exec.Command(editor, tmpPath)` is simpler and sufficient. Use `ExecProcess`. |

**Installation:** No new packages. Zero `npm`/`go get`. (Confirmed by SUMMARY.md: "v3.5 adds zero new Go modules.")

## Package Legitimacy Audit

> Not applicable — this phase installs **no** external packages. All dependencies are Go stdlib (`os`, `os/exec`) or already-vendored modules (`bubbletea/v2`, `bubbles/v2`) present in `go.mod` and verified against the local module cache. `slopcheck` gate skipped (nothing to audit).

## Architecture Patterns

### System Architecture Diagram — TUI Edit Flow (local AND remote, uniform)

```
User presses 'e' on a file row in tabFiles
   │
   ▼
handleFilesKey: capture 'e' → resolveEditor()
   │                                   │
   │  editor == "" ──────────────────► set m.files.err = "$EDITOR is not set…"  (no exec)
   ▼
editFetchCmd (tea.Cmd): client.ReadFile(ctx, sid, relPath)   ◄── DaemonClient (local socket)
   │                                                          ◄── OR RemoteFilesClient (HTTPS+cap)
   ▼
filesEditReadyMsg{data, relPath, editor}
   │
   ▼  (in the ExecCommand.Run / a tea.Cmd-launched setup — NOT in Update)
write data → host-local temp file  (os.CreateTemp + ext from relPath)
   │
   ▼
tea.ExecProcess(exec.Command(editor, tmpPath), onExit)   ── TUI SUSPENDS, editor owns terminal
   │
   ▼  editor exits → onExit(err) → editorExitMsg{tmpPath, relPath, exitErr}
   │
   ├─ exitErr != nil ──► toast "Editor exited with error" ; STILL refresh listing
   ▼
editWriteBackCmd (tea.Cmd): read temp file → client.WriteFile(ctx, sid, relPath, data)
   │                                              (remove temp file regardless)
   ▼
filesWriteBackMsg{err}
   │
   ▼
Update: tea.Batch( tea.ClearScreen,  loadDirCmd(client, sid, cwd, ++generation) )  ── ALWAYS
```

Delete / rename / mkdir flows are simpler — no process suspend:

```
'd' → modalFileDeleteConfirm (reuse kill-confirm rendering) → 'y'/Enter
        → deleteCmd(tea.Cmd): client.DeleteFile(ctx, sid, relPath) → filesOpMsg → loadDirCmd

'r' → inline textinput (reuse editInput pattern) prefilled with current name → Enter
        → renameCmd(tea.Cmd): client.RenameFile(ctx, sid, oldRel, newRel) → filesOpMsg → loadDirCmd

'm' → inline textinput (empty) → Enter
        → mkdirCmd(tea.Cmd): client.MkdirFile(ctx, sid, joinDir(cwd,name)) → filesOpMsg → loadDirCmd
```

### Recommended File Layout (modify existing — no new files needed)
```
internal/tui/
├── files_client.go     # EXTEND FilesClient interface 4→8 methods (TUIW-01)
├── files.go            # ADD edit/delete/rename/mkdir key handling in handleFilesKey;
│                       #   ADD filesModel fields for confirm/inline-input sub-state
├── files_cmds.go       # ADD editFetchCmd, editWriteBackCmd, deleteCmd, renameCmd, mkdirCmd
│                       #   + new msg types (filesEditReadyMsg, editorExitMsg, filesOpMsg)
├── remote_files_client.go  # ADD WriteFile/DeleteFile/RenameFile/MkdirFile to RemoteFilesClient
├── update.go           # ADD editorExitMsg + filesOpMsg cases to Update;
│                       #   maybe a small filesEditCmd helper that builds tea.ExecProcess
├── files_test.go       # EXTEND TestFiles_NoSyncFSCalls file list if a new write file is added;
│                       #   add coverage tests for new requirements
└── modal.go / view.go  # ADD delete-confirm render + hint-bar e/d/r/m hints (parity)
```

### Pattern 1: `tea.ExecProcess` Suspend/Resume for `$EDITOR`
**What:** Hand the terminal to an external editor while Bubble Tea pauses its renderer/input, resume on exit.
**When to use:** The `e` keypress, after the file bytes are in a local temp file.
**Example:**
```go
// Source: charm.land/bubbletea/v2@v2.0.6/exec.go — func ExecProcess(c *exec.Cmd, fn ExecCallback) Cmd
// Reference impl in repo: internal/tui/update.go:366-369 (attachCmd via tea.Exec)
//
// In Update, after the temp file is written:
cmd := exec.Command(editorBin, tmpPath)            // [VERIFIED: API present in v2.0.6]
return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
    return editorExitMsg{tmpPath: tmpPath, relPath: relPath, sessionID: sid, exitErr: err}
})
```
The existing `attachDoneMsg` handler (update.go:81) is the resume-side template: it fires follow-up `tea.Cmd`s after the suspended process returns. Phase 126's `editorExitMsg` handler must `tea.Batch(tea.ClearScreen, editWriteBackCmd(...))` and ultimately `loadDirCmd`.

### Pattern 2: `resolveEditor()` Fallback Chain
**What:** Resolve a runnable editor binary from env + fallbacks.
**Example:**
```go
// resolveEditor returns the resolved editor binary path, or "" if none found.
// Order per TUIW-03: $EDITOR → $VISUAL → nano → vim → vi.
// NOTE: the CONTEXT copy lists $EDITOR before $VISUAL; honor that exact order
// even though many tools check $VISUAL first. The locked decision wins.
func resolveEditor() string {
    for _, cand := range []string{os.Getenv("EDITOR"), os.Getenv("VISUAL"), "nano", "vim", "vi"} {
        if cand == "" {
            continue
        }
        if p, err := exec.LookPath(cand); err == nil {  // [CITED: pkg.go.dev/os/exec#LookPath]
            return p
        }
    }
    return ""
}
```
No-editor error copy (verbatim, locked): `` "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." `` — surface via `m.files.err` (renders in the status line, see `renderFilesStatusLine`) or a toast; status-line err is consistent with existing Files error display.

### Pattern 3: Reuse the Kill-Confirm Modal for Delete
**What:** A centered confirm dialog with No/Yes focus toggle, `y`/`n`/`esc`/`enter`.
**When to use:** `d` on a file/dir.
**Example:** Model the new state after `modalKillConfirm`:
- `model.go`: add `modalFileDeleteConfirm modalState` iota + a `fileDeleteTarget` field (relPath + isDir).
- `update.go`: priority dispatch — add a `handleFileDeleteConfirmKey` mirroring `handleKillConfirmKey` (update.go:669-691). It must sit at a modal priority ABOVE `handleFilesKey` (which is Priority 5.5), exactly as kill-confirm (Priority 2) already does. The existing test `TestFiles_HandleKey_DispatchPriority_BelowKillConfirm` is the parity precedent.
- `modal.go`: a `renderFileDeleteConfirmModal` mirroring `renderKillConfirmModal` (modal.go:163). Use `FgDanger`. Copy: `Delete "name"?` + `This cannot be undone.` (dir variant: `Delete directory "name" and all contents?`).

### Pattern 4: Reuse Inline Textinput for Rename / Mkdir
**What:** A single-line text field captured at the modal/edit-state priority, `enter` commits, `esc` cancels.
**When to use:** `r` (prefill current name) and `m` (empty).
**Example:** `handleRenameKey` (update.go:639-667) is the exact template: `enter` → validate non-empty → dispatch the `tea.Cmd` → clear editing flag; `esc` → cancel; default → forward to `m.editInput.Update(msg)`. Reuse the existing `editInput textinput.Model` (model.go:154) OR add a dedicated `filesNameInput` to `filesModel` to avoid colliding with session-rename state. Recommend a dedicated field on `filesModel` for isolation (Files view is a distinct subsystem). Guard against empty names and (for rename) name-equals-original no-op, as `handleRenameKey` does.

### Anti-Patterns to Avoid
- **Editing the sandbox-absolute path in place:** No client method exposes the work dir; breaks remote parity; risks the no-sync gate. Use read-to-temp + WriteFile-back. (ARCHITECTURE.md §5.3)
- **Synchronous `client.WriteFile` / `os.ReadFile` in `Update`:** Trips `TestFiles_NoSyncFSCalls` and blocks the render loop. All write FS I/O via `tea.Cmd`. (PITFALLS.md §"Synchronous write in TUI update loop")
- **Marking a file "saved" on non-zero editor exit:** Still refresh the listing, but surface an error toast and do NOT suppress the write-back decision based on a happy assumption. (PITFALLS.md §13)
- **`d` firing on a single keypress without confirmation:** Cross-surface parity requires a confirmation step on every surface. (PITFALLS.md §14)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal suspend/resume around editor | Manual `term.MakeRaw`/restore + SIGCONT juggling | `tea.ExecProcess` | Bubble Tea owns the terminal state machine; `ExecProcess` does the suspend/resume correctly. `attachCmd` only hand-rolls raw mode because PTY attach needs byte-level I/O; an editor does not. |
| Editor discovery | Hardcoded `vi` | `resolveEditor()` chain via `exec.LookPath` | Cross-platform + respects user env. `LookPath` confirms the binary actually exists on PATH. |
| Atomic file write to disk | TUI-side temp+rename of the edited bytes | Server-side `Handler.Write` (Phase 123, atomic temp+`Sync`+rename) | The atomic-write + sandbox + denylist all live server-side already. The TUI just sends bytes via `WriteFile`. |
| Confirm dialog | New modal from scratch | Clone `renderKillConfirmModal` + `handleKillConfirmKey` | Established visual + keymap; guarantees cross-surface confirmation parity. |
| Inline name input | New input widget | Reuse `textinput.Model` + `handleRenameKey` shape | Already battle-tested for session rename and filter. |

**Key insight:** This phase is almost entirely composition of existing, tested mechanisms. The risk is in wiring (priority dispatch, generation bumping, unconditional refresh, no-sync gate placement), not in any novel component.

## Runtime State Inventory

> This is a code-only feature addition (new key handlers + interface methods), NOT a rename/refactor/migration. No stored data, live-service config, OS-registered state, secrets, or build artifacts are renamed or migrated.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by grep; no datastore keys/collections changed. | none |
| Live service config | None — no external service config references TUI key handlers. | none |
| OS-registered state | None — no Task Scheduler / launchd / pm2 entries involved. | none |
| Secrets/env vars | `$EDITOR` / `$VISUAL` are *read* (never written) by `resolveEditor()`. No new secret keys. | none — read-only env consumption |
| Build artifacts | None — no package rename; `go build` artifacts unaffected. | none |

## Common Pitfalls

### Pitfall 1: FilesClient return-type mismatch with DaemonClient (TUIW-01 blocker)
**What goes wrong:** The milestone ARCHITECTURE.md §4.1 sketched the interface with `error`-only returns (`DeleteFile(...) error`). But the **actual** Phase 123 `DaemonClient` methods return value structs:
```go
// VERIFIED from internal/daemon/client.go:513-661
func (c *DaemonClient) WriteFile(ctx, sid, relPath string, data []byte) (files.FileWriteResponse, error)
func (c *DaemonClient) DeleteFile(ctx, sid, relPath string)            (files.FileOpResponse, error)
func (c *DaemonClient) RenameFile(ctx, sid, oldRel, newRel string)     (files.FileOpResponse, error)
func (c *DaemonClient) MkdirFile(ctx, sid, relPath string)             (files.FileOpResponse, error)
```
**Why it happens:** The architecture doc was written before Phase 123 froze the signatures.
**How to avoid:** The `FilesClient` interface MUST declare these exact signatures (with the response structs) so `*daemon.DaemonClient` satisfies it via duck typing WITHOUT a wrapper. Then `RemoteFilesClient` must implement the SAME signatures. The interface is the contract; match the existing implementer, not the stale sketch.
**Recommended interface (8 methods):**
```go
type FilesClient interface {
    // Read (existing — DO NOT CHANGE)
    ListFiles(ctx context.Context, sessionID, relPath string) ([]files.FileEntry, bool, error)
    StatFile(ctx context.Context, sessionID, relPath string) (files.FileEntry, error)
    ReadFile(ctx context.Context, sessionID, relPath string) (data []byte, mime string, err error)
    HeadFile(ctx context.Context, sessionID, relPath string) (size int64, mime string, mtime time.Time, err error)
    // Write (new — match DaemonClient signatures exactly)
    WriteFile(ctx context.Context, sessionID, relPath string, data []byte) (files.FileWriteResponse, error)
    DeleteFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
    RenameFile(ctx context.Context, sessionID, oldRel, newRel string) (files.FileOpResponse, error)
    MkdirFile(ctx context.Context, sessionID, relPath string) (files.FileOpResponse, error)
}
```
**Warning sign:** Compile error `*daemon.DaemonClient does not implement FilesClient (wrong type for method DeleteFile)` — means the interface used `error`-only.
**Note on UploadFile:** `DaemonClient.UploadFile` exists but is NOT in the 8-method interface (upload is descoped for TUI — TUIW-06). Do not add it to `FilesClient`.

### Pitfall 2: `RemoteFilesClient` write methods + the `proxyRemoteFiles` body-forwarding question
**What goes wrong:** `RemoteFilesClient` (remote_files_client.go) talks DIRECTLY to the remote webserver over HTTPS+cap (no daemon proxy — confirmed by the type's doc comment, lines 22-26). Its existing read methods all use `nil` request bodies. The new `WriteFile` (PUT, octet-stream body) and `RenameFile` (POST, JSON body) need request bodies; `DeleteFile`/`MkdirFile` do not.
**Why it matters:** Because the TUI bypasses the daemon proxy, the Phase 124 `proxyRemoteFiles` body-forwarding fix (ARCHITECTURE.md §3.5) is NOT on the TUI path — the TUI builds the HTTPS request itself. So `RemoteFilesClient.WriteFile` must construct the body the same way `DaemonClient.WriteFile` does.
**How to avoid:** Mirror each `DaemonClient` write method's body construction in `RemoteFilesClient`, but point at `c.baseURL + "/api/files/<op>?...&cap=<token>"` via the existing `c.filesURL` helper. Preserve the CAP-LEAK invariant (T-122-04-01): never interpolate the full URL (with `cap=`) into an error string — interpolate only `(statusCode, body)` exactly like the existing read methods do.
**Open question:** Does the remote webserver accept the TUI's write requests at the `requireFilesWrite` gate, and does the join-code-exchanged cap carry `files.write`? That is Phase 124/128 territory (remote write parity is explicitly Phase 128 per ROADMAP). For Phase 126, implement the client methods + a `httptest.TLSServer` round-trip test; full two-machine remote write parity is Phase 128's gate. Flag in §Open Questions.

### Pitfall 3: Terminal corruption + stale listing after editor exit (TUIW-04)
**What goes wrong:** Some editors (`vim` on certain terminals) leave the alternate screen / cursor in a bad state; and the listing shows the pre-edit mtime.
**How to avoid:** In the `editorExitMsg` handler, ALWAYS `tea.Batch(tea.ClearScreen, …)` and ALWAYS dispatch `loadDirCmd(client, sid, cwd, ++generation)` — regardless of `exitErr` or write-back result. `tea.ClearScreen` is a `func() Msg` (verified screen.go:20) — pass it as a cmd in the batch. Bump `m.files.generation` before `loadDirCmd` so a stale in-flight request can't clobber the refresh (the WR-03 discipline already in `applyFilesListMsg`).

### Pitfall 4: The No-Sync FS Gate (TUIW-07) — exact placement rules
**What goes wrong:** `TestFiles_NoSyncFSCalls` (files_test.go:850) greps `files.go` and `files_cmds.go` for `\bos\.(ReadDir|Open|OpenFile|Stat)\b` on non-comment lines and fails on any match.
**Critical nuances:**
- The regex matches `os.Open`, `os.OpenFile`, `os.ReadDir`, `os.Stat`. It does NOT match `os.CreateTemp`, `os.ReadFile`, `os.Remove`, `os.Getenv`, `exec.LookPath`, or `exec.Command`. So the temp-file dance (`os.CreateTemp`/`os.ReadFile`/`os.Remove`) does NOT trip the current gate even if placed in `files_cmds.go`.
- BUT TUIW-07 says "extend the gate to cover the new write commands." The right move: keep all editor temp-file FS I/O inside `tea.Cmd` closures (so it's async, not in `Update`), AND consider broadening the regex to also forbid `os.Create`, `os.Remove`, `os.ReadFile`, `os.WriteFile` in `Update`-reachable code — OR (cleaner) put the temp-file I/O in a new file (e.g. `files_edit.go`) that the gate's file list does NOT include, while keeping `files.go`/`files_cmds.go` (the `Update`-path files) clean. The gate's `files []string{"files.go","files_cmds.go"}` is the surface that must stay pure.
- **Recommendation:** Place the editor temp-file read/write inside `tea.Cmd` closures in `files_cmds.go`. The gate's current regex permits `os.CreateTemp`/`os.ReadFile`/`os.Remove`. If a plan wants belt-and-suspenders, add those verbs to the regex AND ensure they only appear inside `func() tea.Msg {…}` closures (which are async by construction). Either way, `client.WriteFile`/`DeleteFile`/etc. are network calls, not `os.*`, so they never match the regex — but they STILL must be in `tea.Cmd`s to avoid blocking the loop.
**How to avoid:** Every write op is dispatched as a `tea.Cmd` returning a result `tea.Msg`. `Update` only sets flags and returns the cmd — never calls `client.WriteFile` or `os.*` directly.

### Pitfall 5: Priority dispatch ordering for new modals/inputs
**What goes wrong:** If delete-confirm or inline-input keys are handled inside `handleFilesKey` (Priority 5.5) instead of as a higher-priority modal, a stray key could leak to tab-cycling or be swallowed wrong.
**How to avoid:** Follow the established priority ladder in `handleKey` (update.go:207+): modals (kill-confirm = Priority 2, new-session = 3, join-code = 3.5) are checked BEFORE the Files handler (5.5). Add file-delete-confirm and Files-inline-input as their own modal/state checks ABOVE the `handleFilesKey` dispatch. Mirror `TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp` with new sub-tests.

## Code Examples

### Edit dispatch in `handleFilesKey` (the `e` branch)
```go
// In handleFilesKey, non-filter mode switch — add:
case s == "e":
    entries := m.files.filteredEntries()
    if len(entries) == 0 || m.files.selected < 0 || m.files.selected >= len(entries) {
        return m, nil
    }
    entry := entries[m.files.selected]
    if entry.IsDir {
        return m, nil // 'e' is a no-op on directories
    }
    editor := resolveEditor()
    if editor == "" {
        m.files.err = errors.New("`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`).")
        return m, nil
    }
    rel := joinDir(m.files.cwd, ansi.Strip(entry.Name))
    m.files.generation++ // supersede in-flight (WR-03 discipline)
    return m, editFetchCmd(m.files.client, m.files.sessionID, rel, editor, m.files.generation)
```

### Write-back tea.Cmd (in files_cmds.go — async, gate-safe)
```go
// Source pattern: mirrors loadDirCmd/readFileCmd nil-guard + context-timeout shape (files_cmds.go).
func editWriteBackCmd(client FilesClient, sid, relPath, tmpPath string, gen uint64) tea.Cmd {
    return func() tea.Msg {
        defer os.Remove(tmpPath) // [VERIFIED: os.Remove not matched by no-sync regex]
        data, rerr := os.ReadFile(tmpPath) // [VERIFIED: os.ReadFile not matched by regex]
        if rerr != nil {
            return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: rerr}
        }
        if isNilFilesClient(client) {
            return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: errNilClient}
        }
        ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer cancel()
        _, werr := client.WriteFile(ctx, sid, relPath, data)
        return filesOpMsg{sessionID: sid, generation: gen, op: "edit", err: werr}
    }
}
```

### Unconditional refresh after editor exit (update.go)
```go
case editorExitMsg:
    cmds := []tea.Cmd{tea.ClearScreen} // [VERIFIED: tea.ClearScreen is a Cmd-returning func, screen.go:20]
    if msg.exitErr != nil {
        m.toast = "Editor exited with error"
        m.toastKind = toastError
        m.toastExp = time.Now().Add(3 * time.Second)
    }
    // write the edited bytes back, then refresh — both async
    cmds = append(cmds, editWriteBackCmd(m.files.client, msg.sessionID, msg.relPath, msg.tmpPath, m.files.generation))
    m.files.generation++
    cmds = append(cmds, loadDirCmd(m.files.client, msg.sessionID, m.files.cwd, m.files.generation))
    return m, tea.Batch(cmds...)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Files view is read-only (preview/navigate) | Files view gains edit/delete/rename/mkdir | Phase 126 (this) | TUI reaches write parity with GUI (minus upload). |
| `FilesClient` = 4 read methods | `FilesClient` = 8 (4 read + 4 write) | Phase 126 (TUIW-01) | Phase 123 deliberately deferred this (scope guard T-123-19). |
| `DaemonClient` write methods unused by TUI | TUI calls them via the interface | Phase 126 | Phase 123 built+tested them (FSW-09); this phase consumes. |

**Deprecated/outdated:**
- The milestone ARCHITECTURE.md §4.1 interface sketch (`error`-only write returns) is superseded by the actual Phase 123 DaemonClient signatures (response-struct returns). Use the actual signatures.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The edit path should read-to-temp + WriteFile-back (NOT edit sandbox-abs path in place), per ARCHITECTURE.md §5.3. The CONTEXT phrase "sandbox-absolute path" is interpreted as "the file at its sandbox-relative path, resolved server-side" rather than a literal local absolute path the TUI computes. | Edit Path Design / Summary | If the user actually wants in-place local-disk editing (e.g. so the editor's own autosave/swap files land in the project dir, or to support editor features that need the real path), the temp-file design changes editor UX subtly (the editor shows a `/tmp/…` path, not the project path; relative includes won't resolve). HIGH-impact UX decision — flag for confirmation. |
| A2 | `os.CreateTemp`/`os.ReadFile`/`os.Remove` in a `tea.Cmd` closure satisfy TUIW-07 because the current gate regex only forbids `os.(ReadDir\|Open\|OpenFile\|Stat)`. | No-Sync Gate / Pitfall 4 | If a plan-checker reads TUIW-07 as "forbid ALL synchronous os.* including temp-file I/O," the gate regex must be broadened and the temp I/O moved to a non-gated file. Low risk (both readings are satisfiable). |
| A3 | Remote write via `RemoteFilesClient` is implemented + unit-tested in Phase 126, but full two-machine remote write parity UAT is Phase 128's gate (per ROADMAP). | Pitfall 2 / Open Questions | If Phase 126 is expected to fully prove remote write end-to-end, scope expands. ROADMAP clearly assigns remote parity to 128, so low risk. |
| A4 | The no-editor error and the upload-descope message are surfaced via `m.files.err` (status line) / status copy, consistent with existing Files error display. | resolveEditor / Upload Descope | If a toast is preferred, trivial change. Negligible risk. |

## Open Questions

1. **Literal "sandbox-absolute path" vs temp-file round-trip (the one real design fork).**
   - What we know: Milestone ARCHITECTURE.md §5.3 + §7.3 + SUMMARY.md all specify read-to-temp + WriteFile-back, working uniformly for local+remote. The TUI has NO client method to obtain a session's absolute work dir (verified: `DaemonClient` exposes no `GetSessionWorkDir`; resolution is server-side only, api.go:85).
   - What's unclear: The CONTEXT/ROADMAP success criterion literally says "spawn `$EDITOR` … with the sandbox-absolute path." Taken literally for local sessions, that would require either a new daemon client method or having the TUI compute the abs path — and it would break remote parity.
   - Recommendation: Implement the temp-file round-trip (A1). Surface this in discuss-phase / plan as the resolved interpretation. If the user insists on literal in-place editing for *local* sessions, that is a divergent design (local fast-path edits abs path; remote falls back to temp round-trip) — more code, breaks the "one pipeline" parity elegance, and needs a new `GetSessionWorkDir` client method. Flag before locking.

2. **Does the remote write path's cap carry `files.write`, and does the remote webserver accept TUI-originated writes?**
   - What we know: TUI uses `RemoteFilesClient` directly (no proxy). Phase 124 adds `requireFilesWrite` + cap issuance; Phase 128 owns remote write parity.
   - What's unclear: Whether the join-code-exchanged cap the TUI holds includes `files.write` by the time Phase 126 ships (Phase 124 controls issuance; build order has 124 before 126).
   - Recommendation: Implement `RemoteFilesClient` write methods + an `httptest.TLSServer` round-trip unit test now. Defer the live two-machine remote write UAT to Phase 128 (its stated gate). Don't block Phase 126 on remote infra.

3. **Recursive directory delete confirmation copy + behavior.**
   - What we know: Server-side `Handler.Delete` supports recursive dir delete (Phase 123); CONTEXT locks "recursive delete for dirs."
   - What's unclear: Whether to show the child count before deleting (PITFALLS.md §14 UX suggestion) — nice-to-have, not locked.
   - Recommendation: Distinct confirm copy for dirs (`Delete directory "x" and all contents?`). Child-count display is optional polish; defer unless trivial.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `nano`/`vim`/`vi` (any editor on PATH) | `e` edit when `$EDITOR`/`$VISUAL` unset | dev machine: likely ✓ | — | `resolveEditor()` returns "" → locked inline error message (TUIW-03). This IS the designed fallback. |
| Go toolchain (`os/exec`, `os` stdlib) | resolveEditor, temp file | ✓ | Go 1.24+ | — |
| `charm.land/bubbletea/v2` | tea.ExecProcess / ClearScreen | ✓ | v2.0.6 | — (verified in module cache) |

**Missing dependencies with no fallback:** None.
**Missing dependencies with fallback:** A bare environment with no `$EDITOR`/`$VISUAL`/`nano`/`vim`/`vi` is handled by design (TUIW-03 error path) — not a build blocker.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (table-driven, in-package `package tui`) |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./internal/tui/ -run 'TestFiles|TestHandleFilesKey|TestApplyFiles' -count=1` |
| Full suite command | `go test -race ./internal/tui/... ./internal/daemon/... ./internal/files/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TUIW-01 | `*daemon.DaemonClient` AND `*RemoteFilesClient` satisfy 8-method FilesClient | compile-time guard + unit | `go test ./internal/tui/ -run TestFilesClient_Interface8 -count=1` | ❌ Wave 0 (add `var _ FilesClient = (*daemon.DaemonClient)(nil)` + existing `var _ FilesClient = (*RemoteFilesClient)(nil)` will fail-compile if signatures wrong) |
| TUIW-02 | `e` dispatches edit via tea.Cmd; write-back is a tea.Cmd | unit | `go test ./internal/tui/ -run TestHandleFilesKey_Edit -count=1` | ❌ Wave 0 |
| TUIW-03 | resolveEditor chain + no-editor error | unit (env-override) | `go test ./internal/tui/ -run TestResolveEditor -count=1` | ❌ Wave 0 |
| TUIW-04 | editorExitMsg → tea.ClearScreen + loadDirCmd unconditionally | unit | `go test ./internal/tui/ -run TestEditorExit_RefreshesUnconditionally -count=1` | ❌ Wave 0 |
| TUIW-05 | delete-confirm + inline rename/mkdir dispatch tea.Cmds; priority correct | unit | `go test ./internal/tui/ -run 'TestFilesDelete|TestFilesRename|TestFilesMkdir' -count=1` | ❌ Wave 0 |
| TUIW-06 | `u` shows upload-descope message; no write attempted | unit | `go test ./internal/tui/ -run TestFilesUpload_Descoped -count=1` | ❌ Wave 0 |
| TUIW-07 | No-sync gate covers write commands | static-grep | `go test ./internal/tui/ -run TestFiles_NoSyncFSCalls -count=1` | ✅ exists (files_test.go:850) — extend file list/regex if needed |
| TUIW-01/02 (remote) | RemoteFilesClient write round-trip | integration | `go test ./internal/tui/ -run TestRemoteFilesClient_Write -count=1` | ❌ Wave 0 (httptest.TLSServer, mirror existing remote read tests in remote_files_client_test.go) |

### Sampling Rate
- **Per task commit:** `go test ./internal/tui/ -run '<targeted>' -count=1`
- **Per wave merge:** `go test -race ./internal/tui/...`
- **Phase gate:** `go test -race ./internal/tui/... ./internal/daemon/... ./internal/files/...` green + `gofmt -l internal/tui/` clean + `golangci-lint run ./internal/tui/...`

### Wave 0 Gaps
- [ ] Compile-time interface guards in `files_client.go`: add `var _ FilesClient = (*daemon.DaemonClient)(nil)` (NEW) alongside the existing RemoteFilesClient guard — fails to compile if signatures diverge (cheapest TUIW-01 enforcement).
- [ ] `resolveEditor` env-override test: set `EDITOR` to a known-present binary, assert resolution; unset all + PATH-strip fallbacks, assert "".
- [ ] `editorExitMsg` handler test: assert the returned `tea.Cmd` batch includes a ClearScreen-yielding cmd AND a loadDir-yielding cmd even when `exitErr != nil`.
- [ ] Delete-confirm priority sub-test mirroring `TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp`.
- [ ] `TestRemoteFilesClient_Write/Delete/Rename/Mkdir` round-trips against `httptest.NewTLSServer` (use `NewRemoteFilesClientForTest`, already exported, remote_files_client.go:82).
- [ ] Extend `TestFiles_NoSyncFSCalls` per the TUIW-07 reading chosen (broaden regex and/or add files to the list).
- [ ] Add a `TestFiles_Phase126_Requirements` traceability matrix mirroring `TestFiles_Phase121_Requirements` (files_test.go:1086) — this is the project's established per-phase verification convention.

## Security Domain

> `security_enforcement` not explicitly disabled in config — section included. Note: the heavy write-security work (sandbox boundary, atomic write, denylist, CSRF Origin check) lives server-side in Phases 123/124/127, NOT in this TUI phase. Phase 126's security surface is thin.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | TUI→local daemon is loopback-trust; remote uses existing cap token (issued elsewhere). |
| V3 Session Management | no | No new sessions/tokens minted here. |
| V4 Access Control | partial | Write authorization is enforced server-side (`requireFilesWrite`, Phase 124). TUI is a client; it cannot bypass server checks. |
| V5 Input Validation | yes | Rename/mkdir names + edit paths are validated SERVER-side (`validateAndClean`, both source+dest for rename — Phase 123). TUI must still send the sandbox-RELATIVE path (`joinDir(cwd, name)`), never an absolute or `../`-laden string. |
| V6 Cryptography | no | None hand-rolled; remote transport is existing TLS 1.2+ (`RemoteFilesClient` enforces `MinVersion: tls.VersionTLS12`). |
| V7 Error Handling/Logging | yes | CAP-LEAK invariant (T-122-04-01): RemoteFilesClient write methods MUST NOT interpolate `cap=` into error strings — interpolate only `(status, body)`, mirroring existing read methods. |

### Known Threat Patterns for {Go TUI + remote HTTPS write client}
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cap token leaking into error/toast text | Information Disclosure | Reuse `redactCapFromURL` / interpolate only (status, body) — established RemoteFilesClient convention. |
| Rename destination traversal (`to=../../.ssh/...`) | Tampering/Elevation | Server-side double-validation (Phase 123 `Sandbox.Rename`). TUI sends relative paths only; never constructs the destination beyond `joinDir(cwd, name)`. |
| Editing a file in `$HOME` session overwriting dotfiles | Tampering | Server-side denylist (Phase 127) + the existing TUI home-dir warning line (files.go:332, `homeDirWriteWarning`) already renders when `HomeDir && FilesWrite`. No new TUI work; warning is live. |
| Editor temp file leaking sensitive bytes to `/tmp` | Information Disclosure | `os.CreateTemp` in the user's temp dir (mode 0600 by default); `defer os.Remove(tmpPath)` always runs. Acceptable for a local single-operator tool; same trust level as the editor itself reading the file. |

## Sources

### Primary (HIGH confidence)
- `internal/tui/files.go` — filesModel struct, handleFilesKey dispatcher, home-dir warning const, preview pipeline [VERIFIED]
- `internal/tui/files_client.go` — current 4-method FilesClient interface [VERIFIED]
- `internal/tui/files_cmds.go` — loadDirCmd/readFileCmd/headFileCmd tea.Cmd patterns, nil-guard, generation discipline [VERIFIED]
- `internal/tui/files_test.go` — TestFiles_NoSyncFSCalls gate (line 850, exact regex), Phase 121 requirements matrix (line 1086) [VERIFIED]
- `internal/tui/attach.go` + `update.go:366` — tea.Exec/ExecCommand suspend-resume reference impl (attachCmd) [VERIFIED]
- `internal/tui/update.go:81` (attachDoneMsg), `:639` (handleRenameKey inline input), `:669` (handleKillConfirmKey) [VERIFIED]
- `internal/tui/modal.go:163` (renderKillConfirmModal), `internal/tui/model.go:13-22` (modalState iotas), `:148-156` (kill/edit state fields) [VERIFIED]
- `internal/tui/remote_files_client.go` — RemoteFilesClient read methods, filesURL helper, CAP-LEAK invariant, TLS 1.2+, NewRemoteFilesClientForTest [VERIFIED]
- `internal/daemon/client.go:513-661` — DaemonClient WriteFile/UploadFile/DeleteFile/RenameFile/MkdirFile EXACT signatures (response-struct returns) [VERIFIED]
- `internal/daemon/api.go:81-90` — server-side work dir resolution (GetSessionWorkDir); confirms no TUI-side abs-path access [VERIFIED]
- `internal/daemon/types.go` — SessionInfo (has WorkDir+HomeDir+FilesWrite), FileWriteResponse/FileOpResponse shapes [VERIFIED]
- `charm.land/bubbletea/v2@v2.0.6/exec.go` (Exec, ExecProcess, ExecCommand), `screen.go:20` (ClearScreen) — module cache inspection [VERIFIED]
- `.planning/phases/123-.../123-04-SUMMARY.md` — DaemonClient write methods provided; FilesClient scope-guard deferred to 126 [VERIFIED]
- `.planning/research/ARCHITECTURE.md` §5.3, §7.3 — TUI $EDITOR read-to-temp design [CITED]
- `.planning/research/PITFALLS.md` §1, §13, §14, "Synchronous write in TUI update loop" [CITED]
- `.planning/research/SUMMARY.md` — Phase 126 scope, "zero new Go modules", tea.Exec ratified [CITED]
- `.planning/REQUIREMENTS.md` TUIW-01..07 [VERIFIED]

### Secondary (MEDIUM confidence)
- [pkg.go.dev/os/exec#LookPath] — editor resolution [CITED]
- [pkg.go.dev/os#CreateTemp] — temp file creation [CITED]

### Tertiary (LOW confidence)
- None — all claims grounded in source or stdlib docs.

## Recommended Plan / Wave Decomposition

A clean four-wave decomposition (mirrors the requested ordering):

- **Wave 1 — Interface + transport (TUIW-01):** Extend `FilesClient` to 8 methods (exact DaemonClient signatures). Add `var _ FilesClient = (*daemon.DaemonClient)(nil)` guard. Implement the 4 write methods on `RemoteFilesClient` + `httptest.TLSServer` round-trip tests. *Gate: package compiles, both clients satisfy the interface, remote round-trips pass.* (Foundational — everything else depends on it.)
- **Wave 2 — `$EDITOR` shell-out (TUIW-02, TUIW-03, TUIW-04):** `resolveEditor()`, the `e` branch, `editFetchCmd`/`editWriteBackCmd`, `editorExitMsg` handler with `tea.ClearScreen` + unconditional `loadDirCmd`, no-editor error. *Gate: edit dispatch + refresh tests; manual UAT of suspend/resume.*
- **Wave 3 — d/r/m affordances (TUIW-05):** delete-confirm modal (clone kill-confirm), inline rename/mkdir (clone handleRenameKey), priority-dispatch wiring + tests, hint-bar/help parity. *Gate: dispatch + priority tests; cross-surface confirm parity.*
- **Wave 4 — Descope + gate (TUIW-06, TUIW-07):** `u` upload-descope message + file the GitHub issue (per SC#4 / memory: GitHub issues drive release planning — file in `scottkw/agenthub`). Extend `TestFiles_NoSyncFSCalls`. Add `TestFiles_Phase126_Requirements` traceability matrix. *Gate: full `-race` suite green, gofmt/lint clean, no-sync gate passes with write commands.*

Wave 1 must land first (interface is the contract). Waves 2 and 3 are independent of each other and could parallelize. Wave 4 is the closing/verification wave.

## Project Constraints (from CLAUDE.md)
- **Go conventions:** `gofmt`, `golangci-lint`, context-aware functions (`ctx context.Context` first param) — all new write tea.Cmds use `context.WithTimeout`.
- **Cross-surface parity is release-blocking:** TUI write ops must match GUI; the ONLY sanctioned gap is upload (TUIW-06, ROADMAP SC#4) — requires a filed GitHub issue.
- **GitHub issues drive release planning:** File the upload-descope follow-up issue in `scottkw/agenthub`, citing it as the documented parity gap.
- **Colorblind user:** Any color-coded state (error toast, danger button) must carry a non-color signal (glyph/text). The delete-confirm should use explicit "Delete"/"Cannot be undone" text, not just red — mirroring the existing `homeDirWriteWarning` ⚠+"Warning:" convention.
- **No browser UAT here** (TUI phase) — but `/gsd:verify-work` may want a terminal UAT of the suspend/resume + refresh cycle.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new deps; tea.Exec/ExecProcess/ClearScreen verified in module cache; DaemonClient write methods verified in source.
- Architecture: HIGH — edit-path design ratified by milestone research + cross-checked against the actual (no `GetSessionWorkDir`) client surface.
- Pitfalls: HIGH — the FilesClient return-type mismatch and no-sync gate placement are verified against live source, not assumed.

**Research date:** 2026-06-14
**Valid until:** 2026-07-14 (stable; internal codebase + stdlib + pinned bubbletea v2.0.6)
