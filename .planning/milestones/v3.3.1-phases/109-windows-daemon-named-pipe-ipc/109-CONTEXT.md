# Phase 109: Windows daemon named-pipe IPC - Context

**Gathered:** 2026-05-18
**Status:** Ready for planning
**Mode:** Pre-authored from `.planning/milestones/v3.3.1-ROADMAP.md` + `.planning/REQUIREMENTS.md` (research complete — discuss skipped per user feedback)

<domain>
## Phase Boundary

AgentHub daemon, GUI, CLI, and TUI all work end-to-end on Windows 11 by listening on `\\.\pipe\agenthub-daemon` instead of failing to bind a Unix socket. Adapts the daemon's `internal/daemon` listener and `DaemonClient` dialer to use `winio` named pipes on Windows and Unix sockets on macOS/Linux. Closes GitHub Issue #52 and lands third-party PR #53 by `im-alexandre`.

</domain>

<decisions>
## Implementation Decisions

### IPC abstraction
- **Platform split via `ipc_windows.go` + `ipc_nonwindows.go` build tags** — per PR #53 structure, two new files plus threading through `api.go` + `client.go` + `tray_windows.go`. Avoids `runtime.GOOS` branching inside core code paths.
- **Windows listener:** `winio.ListenPipe(\\.\pipe\agenthub-daemon, &winio.PipeConfig{...})`. Use Microsoft's `github.com/Microsoft/go-winio` package (PR #53 already vendors this).
- **Unix listener (unchanged):** `net.Listen("unix", socketPath)` on macOS/Linux.
- **Windows client dial:** `winio.DialPipeContext(ctx, \\.\pipe\agenthub-daemon)` — context-aware for timeout handling consistent with existing `EnsureDaemon` retry semantics.

### PR #53 evaluation (MANDATORY, discrete task)
- **First task in the plan must be PR #53 evaluation** — fetch the PR, identify conflict files against `main`, decide rebase-then-merge vs. re-apply-from-scratch, document the decision in writing.
- **Predicted conflict surface:** `internal/daemon/api.go` and `internal/daemon/client.go` (five v3.3 commits touched these — handleListShells, handleUpdateShellPath, ShellWebShareWarned).
- **Two new IPC files drop in clean** (no upstream history on those paths).
- **Author attribution non-negotiable:** `Co-Authored-By: im-alexandre <email>` on the merge / cherry-picked commits, OR dedicated commit message line `Re-applies PR #53 by @im-alexandre` if re-applied. Either path acceptable; not both.

### Stop / cleanup
- **`API.Stop()` must NOT attempt filesystem removal on named-pipe paths.** Named pipes are kernel objects; closing the listener releases them. Add a build-tagged `cleanupSocketIfNeeded` helper or inline `runtime.GOOS` check at the cleanup site.
- **`CleanupStaleSocket` named-pipe probing remains functional** — used at daemon startup to detect a running daemon. PR #53 likely already addresses this; confirm during integration.

### Test surface
- **Windows regression test** required: exercises `API.Start` + `DaemonClient.Health()` over a real named pipe end-to-end, plus `API.Stop` on named-pipe path. Place under `internal/daemon/` with `//go:build windows` tag. Not just `CleanupStaleSocket` probing — full Health-check round-trip.
- **macOS/Linux unit tests** unchanged; confirm no regression.

### Cross-surface verification (release gate)
Windows 11 — GUI launch + daemon auto-start + create/list/attach session via GUI; `agenthub.exe list / new / daemon status / tui` via CLI; TUI session list + attach/detach. macOS + Linux — full smoke (daemon up, session create/list/attach, web-share toggle) confirming no regression.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/daemon/api.go` — existing `API.Start` / `API.Stop` lifecycle; needs platform branching at listener creation + cleanup.
- `internal/daemon/client.go` — `DaemonClient` with `Health`, `EnsureDaemon`, etc; needs platform branching at dial.
- `internal/daemon/socket.go` (or equivalent) — `CleanupStaleSocket` helper; likely already has Windows path detection per PR #53.
- `cmd/agenthub/tray_windows.go` — tray-icon driven daemon control; threaded through by PR #53.

### Established Patterns
- Build-tag splits already in use across the codebase (`*_windows.go` / `*_unix.go` or `*_nonwindows.go`). Follow that convention for IPC.
- Existing `EnsureDaemon` retry/timeout patterns — preserve when adding context-aware named-pipe dial.
- Daemon-side cleanup uses defer chains in `API.Start` — keep the same shape, just branch the cleanup primitive (rm vs. close-only).

### Integration Points
- PR #53 touches: `internal/daemon/ipc_windows.go` (new), `internal/daemon/ipc_nonwindows.go` (new), `internal/daemon/api.go`, `internal/daemon/client.go`, `cmd/agenthub/tray_windows.go`, plus `go.mod` / `go.sum` for `github.com/Microsoft/go-winio`.
- v3.3 commits since PR base (likely conflict zones): shell discovery routes (`handleListShells`), shell-path routes (`handleUpdateShellPath`), `ShellWebShareWarned` plumbing in `api.go` + `client.go`.
- Tests: `internal/daemon/*_test.go` — Windows regression test goes here.

</code_context>

<specifics>
## Specific Ideas

- **PR #53 reference:** https://github.com/scottkw/agenthub/pull/53 — base commit `032a6e9` (v3.2), 140 commits behind v3.3 tip at fetch time. 7 files, +214/-13.
- **PR #53 author handle:** `im-alexandre` — preserve full GitHub handle on attribution line; resolve email from `git log` if possible, otherwise use `<im-alexandre@users.noreply.github.com>`.
- **Issue #52 reproduction:** Fresh Windows 11, `agenthub.exe daemon run` → observe `bind: A socket operation encountered a dead network` on `main`. After fix: same command succeeds, session creation works end-to-end across GUI/CLI/TUI.
- **Cross-surface parity is release-blocking** per v3.3 Phase 108 contract (user memory `feedback_cross_surface_parity`). All three Windows surfaces must PASS before this phase verification can pass.

</specifics>

<deferred>
## Deferred Ideas

None — discuss skipped because spec is fully resolved upstream in REQUIREMENTS.md (IPC-01..06) and the v3.3.1 roadmap detail file. Any new questions surfaced during plan/execute should be raised back to the user as blockers, not silently absorbed.

</deferred>
