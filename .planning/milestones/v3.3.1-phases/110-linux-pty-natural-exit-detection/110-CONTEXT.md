# Phase 110: Linux PTY natural-exit detection - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning
**Mode:** Pre-authored from `.planning/milestones/v3.3.1-ROADMAP.md` + `.planning/REQUIREMENTS.md` (research complete — discuss skipped per user feedback)

<domain>
## Phase Boundary

On Linux, a shell session whose child process exits cleanly causes the GUI tab / TUI list entry to auto-close — matching macOS behavior shipped in v3.3 SHELL-12. The fix is platform-specific: Linux PTY EOF semantics differ from macOS, so a separate exit-detector goroutine must be added that polls `syscall.Wait4(pid, &status, WNOHANG)` and explicitly closes the PTY to unblock the read loop. Closes GitHub Issue #57.

</domain>

<decisions>
## Implementation Decisions

### Root cause
- **`pty.Read()` blocks indefinitely after a clean child exit on Linux amd64 with go-pty v0.2.2/v0.2.3** — confirmed by Issue #57 + v3.3 Phase 107 deferred notes. The frontend's `session:exit` listener never fires because the daemon-side read loop never returns.
- **macOS behaves correctly** — clean exit causes EOF on the PTY master, which terminates the read loop naturally. Do NOT regress macOS.

### Fix shape (per `.planning/milestones/v3.3.1-ROADMAP.md` Phase 110)
- **Separate exit-detector goroutine** (Linux-only via build tag) that polls `syscall.Wait4(pid, &status, syscall.WNOHANG)` at a low cadence (e.g. 100–250 ms), and when the child has exited, explicitly closes the PTY master FD to unblock the read loop.
- **Coordinate carefully with go-pty's internal `waitOnContext`** to avoid a double-`Wait` race (calling `Wait4` after go-pty has already reaped the child is undefined behavior — likely an ECHILD or wrong-status return).
  - Approach: use go-pty's existing exit callback / process handle, OR ensure our exit-detector is the only `Wait4` caller for that PID (suppress go-pty's internal waiter via API if available, or wrap the PID handle before passing it to go-pty).
- **Build-tag split:** `internal/pty/exit_linux.go` (with detector goroutine) vs. `internal/pty/exit_other.go` (no-op). Mirrors the convention used in Phase 109.

### Test requirements (PTY-03)
- **Re-enable `TestListSessions_OnExitCallback_ReceivesNormalized`** in `internal/daemon/engine_test.go` (or wherever it lives) — currently `t.Skip()`'d on Linux. After the fix, it must run and pass deterministically under `go test -race -shuffle=on` on `linux/amd64`.
- **Do NOT add `t.Skip()` paths** — fix the root cause, don't paper over.

### Cross-surface verification (release gate)
- **Linux GUI** — open a shell session, type `exit` at the prompt → tab auto-closes within ~1 s.
- **Linux TUI** — same test, list entry disappears within ~1 s.
- **Linux CLI attach/detach** — unaffected by this fix's primary path, but daemon-side cleanup is the same path so confirm CLI round-trip is not regressed.
- **macOS smoke** — same `exit` test still auto-closes; v3.3 SHELL-12 behavior holds for both exit-0 and non-zero exits per project memory `project_shell_exit_toast_descoped`.

### Out of scope
- **Windows PTY exit semantics** — separate ConPTY/winpty problem; not closed by this phase. If a regression test catches it, file separately.
- **Refactoring the existing PTY engine** — Chesterton's Fence; surgical addition only.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/pty/` — existing PTY package; go-pty v0.2.x dependency. Has macOS read-loop EOF path that works.
- `internal/daemon/engine.go` — likely the `session:exit` emitter; calls the OnExit callback that the frontend listens to.
- `internal/daemon/engine_test.go` — host of the skipped `TestListSessions_OnExitCallback_ReceivesNormalized`.

### Established Patterns
- Build-tag split conventions: `*_linux.go` / `*_other.go` or `*_unix.go` / `*_windows.go`. Choose the one in active use in `internal/pty`.
- Exit-code-0 normalization (SHELL-12 in Phase 107) — the frontend treats both exit code 0 and the natural-exit signal as "auto-close" per memory `project_shell_exit_toast_descoped`. Daemon must emit consistently for both.

### Integration Points
- Read loop in `internal/pty/` (whatever wraps `pty.Read()`).
- Exit callback chain: PTY exit → engine OnExit → daemon broadcasts `session:exit` → frontend `autoCloseRef`-gated tab close (Phase 107 SHELL-12) + TUI list entry removal.

</code_context>

<specifics>
## Specific Ideas

- Issue #57 reproduction: Linux desktop build, open shell session, type `exit 0`. On `main`, tab stays open (read loop blocked). After fix, tab auto-closes ~1 s after exit.
- Non-zero exit verification: `bash -c 'exit 7'` should also auto-close per v3.3 SHELL-12 "auto-close on any natural exit" (project memory `project_shell_exit_toast_descoped`). Verify both code paths.
- Wait4 cadence: 100–250 ms feels right (sub-second user-perceptible response, low daemon CPU overhead). Final value at executor's discretion; mention in commit message.

</specifics>

<deferred>
## Deferred Ideas

None — Phase 110 scope is tightly defined. Any related items (Windows PTY exit semantics, advisory log-clean-up, etc.) belong in separate issues / phases.

</deferred>
