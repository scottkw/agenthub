# Phase 109 — Verification

**Phase:** 109 — Windows daemon named-pipe IPC
**Plan:** 109-01 — cherry-pick PR #53
**Branch:** `phase-109-windows-named-pipe-ipc` (off `main` at `9cc1087`)
**Closes:** Issue [#52](https://github.com/scottkw/agenthub/issues/52)

## Commits Landed

| # | SHA       | Author                                          | Summary                                          |
| - | --------- | ----------------------------------------------- | ------------------------------------------------ |
| 1 | `68b2421` | Alexandre Castro `<im.alexandre07@gmail.com>` | docs: design Windows daemon named pipe IPC fix   |
| 2 | `2f25e63` | Alexandre Castro `<im.alexandre07@gmail.com>` | fix: use Windows named pipes for daemon IPC      |
| 3 | `fc50cd4` | Alexandre Castro `<im.alexandre07@gmail.com>` | fix: load Windows module handle from kernel32    |
| 4 | (pending Task 5 commit) | Ken Scott `<kscott@iprosystems.com>` | docs(109): PR #53 evaluation + VERIFICATION scaffold for Phase 109 |

Author audit:
```
$ git log --format='%an <%ae>' main..HEAD | sort -u
Alexandre Castro <im.alexandre07@gmail.com>
Ken Scott <kscott@iprosystems.com>   # (after Task 5 commit lands)
$ git log --format='%an' main..HEAD | grep -c "Alexandre Castro"
3
```

IPC-06 attribution mechanic: cherry-pick preserves `Author:` line on all three commits. See `109-PR53-EVALUATION.md` for full rationale.

## Automated Checks (PASSED — with one caveat)

### Cross-compile to Windows
```
$ GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...
# exit 0 — verified after each of Tasks 2 and 3
```
Result: PASS. Proves the build-tag split (`ipc_windows.go` + `ipc_nonwindows.go`) compiles cleanly on the Windows target without a Windows host.

### Build-tag wiring inspection
```
$ grep -l '//go:build windows' internal/daemon/ipc_windows.go
internal/daemon/ipc_windows.go
$ grep -l '//go:build !windows' internal/daemon/ipc_nonwindows.go
internal/daemon/ipc_nonwindows.go
$ grep -E 'listenDaemonSocket|removeDaemonSocket' internal/daemon/api.go | head -3
$ grep dialDaemonSocket internal/daemon/client.go | head -3
```
Result: PASS. All three helper symbols are wired through `api.go` and `client.go` as the plan's `<interfaces>` block predicts.

### macOS unit smoke (`./internal/daemon/`)
```
$ go test -race -short ./internal/daemon/
# After Task 2 and again after Task 3:
# FAIL — but only because of three pre-existing failures unrelated to Phase 109:
#   - TestAPIGetShellWebShareWarned_Default (api_test.go:1592)
#   - TestDaemonClient_GetSetShellWebShareWarned_RoundTrip (api_test.go:1638)
#   - TestSetShellWebShareWarned_Default (engine_test.go:905)
```

**These three failures pre-exist on plain `main`** — see `.planning/phases/109-windows-daemon-named-pipe-ipc/deferred-items.md` for reproduction and root-cause notes. They are in `ShellWebShareWarned` default-value logic introduced by Phase 101, not by Phase 109's IPC abstraction. Not a Phase 109 regression.

Tests skipped by build tag on macOS (correctly compile-clean, no-op execute):
```
$ go test -race -short -run TestAPIStart_WindowsNamedPipeHealth ./internal/daemon/
ok  github.com/scottkw/agenthub/internal/daemon  1.030s [no tests to run]
```
PASS — the Windows-only tests do not break the macOS build, they're simply skipped.

### Cross-platform regression (whole suite, `./...`)
```
$ go test -race -short ./...   # security-review/ dir excluded (stray local artifacts, .gitignored)
ok    github.com/scottkw/agenthub
ok    github.com/scottkw/agenthub/internal/attach
ok    github.com/scottkw/agenthub/internal/capability
FAIL  github.com/scottkw/agenthub/internal/daemon  (the same 3 pre-existing failures only)
ok    github.com/scottkw/agenthub/internal/pty
ok    github.com/scottkw/agenthub/internal/relay
ok    github.com/scottkw/agenthub/internal/release
ok    github.com/scottkw/agenthub/internal/status
ok    github.com/scottkw/agenthub/internal/statusbar
ok    github.com/scottkw/agenthub/internal/tailnet
ok    github.com/scottkw/agenthub/internal/tui
ok    github.com/scottkw/agenthub/internal/updater
ok    github.com/scottkw/agenthub/internal/webserver
```
Result: PASS for every package touched by Phase 109 (internal/daemon's IPC abstraction does not introduce a single new failure). The three pre-existing `internal/daemon` failures continue exactly as on `main` — see deferred-items.md.

## Human verification needed (IPC-05 — three discrete items)

Cross-surface parity is release-blocking per project memory `feedback_cross_surface_parity`. The macOS executor cannot run Windows binaries; the three items below require a Windows 11 host (physical or VM with full graphics — Wails GUI requires WebView2 runtime per project memory `feedback_verify_test_env_before_declaring_failure`).

### human_needed: WIN-GUI-01

**Surface:** GUI (Wails app)
**Host:** Fresh Windows 11 with WebView2 runtime installed.
**Steps:**
1. Build with `wails build -platform windows/amd64` (from macOS) or `wails build` (from the Windows host directly against `phase-109-windows-named-pipe-ipc`).
2. Transfer `agenthub.exe` to the Windows host (skip if built natively).
3. Double-click `agenthub.exe`. Observe the tray icon appears (proves Task 3 kernel32 GetModuleHandleW fix landed; previously `pGetModuleHandleW.Call(0)` would return 0 with `ERROR_PROC_NOT_FOUND`).
4. Click the tray icon → open the AgentHub window.
5. Create a new shell session via the GUI new-session modal (pick `claude` agent and a real folder).
6. Confirm the session tab opens with a working shell.
7. Toggle web-share ON; confirm the share modal displays a `https://...` capability URL.

**Pass criterion:** All of the above succeed without `bind: A socket operation encountered a dead network` in the daemon log AND tray icon visible AND web-share URL renders.
**Fail criterion:** Any of the listed failure modes.
**On pass:** Replace this entry's `human_needed: WIN-GUI-01` header with `human_verified: WIN-GUI-01 (YYYY-MM-DD by <name>)` and a one-line note.

### human_needed: WIN-CLI-01

**Surface:** CLI
**Host:** Same Windows 11 host as WIN-GUI-01 (fresh PowerShell session).
**Steps:**
1. `.\agenthub.exe daemon status` — expect "running" or auto-start then "running"; no bind error.
2. `.\agenthub.exe new claude C:\Users\$env:USERNAME\dev` (substitute a real path).
3. `.\agenthub.exe list` — expect the new session listed.
4. `.\agenthub.exe daemon status` again — still "running".

**Pass criterion:** All four commands connect over `\\.\pipe\agenthub-daemon` and return real data.
**Fail criterion:** Any `EnsureDaemon` timeout or named-pipe error.
**On pass:** Replace `human_needed: WIN-CLI-01` with `human_verified: WIN-CLI-01 (YYYY-MM-DD by <name>)`.

### human_needed: WIN-TUI-01

**Surface:** TUI (Bubble Tea)
**Host:** Same Windows 11 host (same PowerShell or new one).
**Steps:**
1. `.\agenthub.exe tui` — Bubble Tea UI renders with the session from WIN-CLI-01 listed.
2. Press the attach keybinding (Enter on the highlighted session); confirm a working shell inside the TUI.
3. Type `echo hello`; confirm output renders.
4. Press the detach keybinding (Ctrl-B then d, or whatever agenthub binds — `?` help overlay confirms).
5. Confirm return to the session list with the entry still present.

**Pass criterion:** TUI connects, attach round-trips, detach returns cleanly.
**Fail criterion:** TUI hangs on launch, empty session list, or attach fails.
**On pass:** Replace `human_needed: WIN-TUI-01` with `human_verified: WIN-TUI-01 (YYYY-MM-DD by <name>)`.

### Windows-only unit tests (operator must run on the Windows host)

Run on the Windows 11 test host (finalizes IPC-04):
```
go test -race -short -run 'TestAPI(Start|Stop)_WindowsNamedPipe|TestCleanupStaleSocket_WindowsPipe' -count=1 .\internal\daemon
```
Expected: all of `TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, `TestCleanupStaleSocket_WindowsPipe_NoServer`, `TestCleanupStaleSocket_WindowsPipe_Active` pass. Record the command and exit code under WIN-GUI-01 when flipping to human_verified.

## Cross-platform Regression (macOS / Linux)

### macOS
- macOS GUI/CLI/TUI smoke — exercised in-thread or during `/gsd-verify-work`: `wails dev` for GUI, `agenthub list/new/daemon status` for CLI, `agenthub tui` for TUI. Pass = no behavioral change from pre-cherry-pick `main`.
- IPC-03 regression check on Unix sockets: after stopping the daemon, `ls $TMPDIR/agenthub-*` or check the macOS socket path (`/Users/.../Library/Application Support/agenthub/daemon.sock`) — the socket file should be removed (`removeDaemonSocket` on macOS still calls `os.Remove`, unchanged behavior).

### Linux
- Linux smoke — typically delegated to CI: `go test -race -short ./...` on linux/amd64 runners. Pass = green CI on the phase branch's PR (the three pre-existing `ShellWebShareWarned` failures will also appear on Linux until that separate bug is fixed; reviewers must distinguish "pre-existing on main" from "introduced by this PR").

## Requirements Coverage Map

| Req     | Closed by                                                                                                                                  | Status                                                |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| IPC-01  | Cherry-pick `2f25e63` — `listenDaemonSocket` swap in `api.go::API.Start` (build-tagged: `winio.ListenPipe` on Windows, `net.Listen("unix")` elsewhere) | code-complete; pending Windows UAT (WIN-GUI-01 + WIN-CLI-01) |
| IPC-02  | Cherry-pick `2f25e63` — `dialDaemonSocket` swap in `client.go::NewDaemonClient` (`winio.DialPipeContext` on Windows)                            | code-complete; pending Windows UAT (WIN-CLI-01 + WIN-TUI-01) |
| IPC-03  | Cherry-pick `2f25e63` — `removeDaemonSocket` no-op-on-pipe in `api.go::API.Stop` (short-circuits on `isWindowsNamedPipe(path)`)                  | code-complete; pending Windows UAT (no error spam on daemon shutdown) |
| IPC-04  | Cherry-pick `2f25e63` — `TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, `uniqueWindowsPipePath` in `socket_windows_test.go` | tests compile-clean on macOS via cross-compile; pending Windows test run |
| IPC-05  | Three `human_needed` items above (WIN-GUI-01 + WIN-CLI-01 + WIN-TUI-01)                                                                       | pending Windows hardware                              |
| IPC-06  | `109-PR53-EVALUATION.md` + cherry-pick author preservation                                                                                  | **PASSED** (automated: `git log --format='%an' main..HEAD \| grep -c "Alexandre Castro"` = 3) |

## GitHub-issue cross-check

Per project memory `feedback_check_github_issues_during_uat`: the human operator who performs WIN-GUI-01 / WIN-CLI-01 / WIN-TUI-01 must also scan [scottkw/agenthub open issues](https://github.com/scottkw/agenthub/issues) before recording UAT pass. If any new bug filed citing "Discovered during Phase 109 UAT-NN" exists, it overrides a casual pass. Record the date of the issues-scan in the human_verified note (e.g., "verified 2026-05-19; issues scanned same day, no Phase-109 blockers filed").

## Deferred / Pre-existing Issues (do NOT block Phase 109)

See `.planning/phases/109-windows-daemon-named-pipe-ipc/deferred-items.md` for three pre-existing `ShellWebShareWarned` test failures on `main` (introduced by Phase 101, not Phase 109). These should be tracked as a separate follow-up bug, not blocking IPC-05 verification.
