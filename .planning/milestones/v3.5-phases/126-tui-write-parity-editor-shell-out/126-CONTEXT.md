# Phase 126: TUI Write Parity (`$EDITOR` Shell-Out) - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

TUI users can edit files via `$EDITOR` shell-out, delete, rename, and create directories using keyboard shortcuts within the Files view — with full cross-surface parity against the GUI write operations (minus upload, which is formally descoped with an on-screen message).

**Depends on:** Phase 123 (`DaemonClient` write methods available; `FilesClient` interface extension ready — Phase 123 deliberately left the interface at 4 methods, this phase extends it to 8). COMPLETE.

**Requirements:** TUIW-01, TUIW-02, TUIW-03, TUIW-04, TUIW-05, TUIW-06, TUIW-07

**Cross-surface parity (release-blocking):** TUI write ops must match the GUI (Phase 125) write ops, EXCEPT upload — which is FORMALLY DESCOPED per ROADMAP SC#4 (on-screen "Use desktop or web to upload files." + a follow-up GitHub issue). The ROADMAP is the user's sign-off for this sanctioned exception.
</domain>

<decisions>
## Implementation Decisions

### Locked (from ROADMAP success criteria)
- `e` on a selected file → suspend TUI, spawn resolved `$EDITOR` (fallback chain `$EDITOR` → `$VISUAL` → `nano` → `vim` → `vi`) with the sandbox-absolute path, resume on exit. Terminal restored via `tea.ClearScreen`; directory listing refreshes unconditionally after every edit.
- No editor resolvable → inline error: "`$EDITOR` is not set. Set it in your shell profile (e.g. `export EDITOR=nano`)." (not a crash / silent no-op).
- `d` → confirmation dialog (reuse kill-session pattern); recursive delete for dirs. `r` → inline rename. `m` → inline mkdir. Both refresh listing on completion.
- `u` (upload) → on-screen "Use desktop or web to upload files." (the one documented parity gap) + file a follow-up GitHub issue.
- `FilesClient` interface = exactly 8 methods (4 read + 4 write); both `*daemon.DaemonClient` and `*tui.RemoteFilesClient` satisfy it. `TestFiles_NoSyncFSCalls` static-grep gate passes with write commands — all write FS I/O via `tea.Cmd`, never synchronous in `Update`.

### Claude's Discretion
Command/keybinding wiring details, confirm-dialog reuse, inline-input component reuse — at Claude's discretion guided by existing TUI Files view patterns.
</decisions>

<code_context>
## Existing Code Insights

Gathered during plan-phase research. Anchors: the v3.4 TUI Files view (internal/tui/), the read-side FilesClient interface (4 methods), the DaemonClient write methods (Phase 123), the RemoteFilesClient, the existing kill-session confirm dialog pattern, the inline-input pattern, the Phase 124 TUI home-dir warning, and the `$EDITOR` shell-out / tea suspend-resume mechanism (tea.ExecProcess).

</code_context>

<specifics>
## Specific Ideas

Refer to ROADMAP Phase 126 success criteria (precise + testable). `$EDITOR` shell-out via Bubble Tea's process-suspension (tea.ExecProcess). All FS I/O through tea.Cmd (the TestFiles_NoSyncFSCalls gate enforces this).

</specifics>

<deferred>
## Deferred Ideas

TUI file upload — formally descoped this milestone (on-screen message + GitHub issue per SC#4).

</deferred>
