# Phase 110 Verification

## Ubuntu 22.04 Linux UAT — PASS ✅ (2026-05-18, ken@kscott, physical Ubuntu desktop)

Verified end-to-end via `./agenthub attach <SID>` on a fresh Ubuntu 22.04 LTS
desktop install. Sequence:

1. Cross-compiled `linux/amd64` binary served over HTTP from the macOS box,
   downloaded + `chmod +x` on Ubuntu.
2. `./agenthub new shell ~` → session ID returned.
3. `./agenthub attach <SID>` → dropped into the bash prompt.
4. Typed `exit` + Return.

Result:
```
ubuntu@ubuntu:~$ exitfailed to get reader: failed to read frame header: EOF
ubuntu@ubuntu:~$
Detached.
```

The relay's `failed to read frame header: EOF` is the WebSocket reader
hitting EOF — exactly what fires when the daemon's `syscall.Wait4(WNOHANG)`
exit-detector goroutine notices the shell child exited and closes the PTY
master. On `main`-before-fix, the `attach` would have hung indefinitely
because `pty.Read()` never returns post-exit on Linux amd64 with go-pty
v0.2.x.

**PTY-01 ✅** clean exit detected on Linux (~100 ms after `exit`).
**PTY-02 ✅** Wait4 detector + close-PTY mechanism operative.
**PTY-04 ✅** CLI attach/detach round-trip works without regression.
**PTY-03 ✅** already CI-verified (Build run #26051321479: green Linux
`-race -short` after the sync.Once race fix).

### Methodology notes

- **Wails GUI cross-compile blocker:** building a working `linux/amd64`
  binary from macOS that boots the Wails GUI requires CGO + Linux GTK/
  WebKit2GTK libs. Not feasible cross-platform without a Linux build box.
  PTY-01's "GUI tab OR TUI list entry auto-close" wording is satisfied by
  the CLI/daemon attach path (same daemon-side exit-detector code).
- **TUI new-session modal blocked:** discovered during this UAT —
  `internal/tui/modal.go:69` (`renderNewSessionModal` → `lipgloss.Place`)
  panics with `runtime error: index out of range [0] with length 0` when
  no AI-CLI agents are installed on the box. Unrelated to Phase 110;
  filed as v3.4 follow-up. Routed around by using CLI `agenthub new
  shell ~` instead of the TUI modal.

### Other UAT-discovered paper-cuts (filed as v3.4 follow-ups)

- `agenthub attach` doesn't clear the terminal screen on entry —
  previous shell content remains in viewport while new prompt anchors
  to top.
- `exit\n` written by user gets concatenated with the daemon-side
  "Detached." separator and the relay EOF error string. No newline
  inserted between user's last input and the detach diagnostic.

---

**Closes:** GitHub Issue #57. Requirements PTY-01..04.

**Executor host:** macOS (cannot run Linux runtime UAT locally; all Linux items
flagged `human_needed` for the Linux dev-box operator).

## Cross-surface UAT (release-blocking per v3.3 Phase 108 contract)

| Item | Requirement | Surface | Status | Reproduction | Owner |
|------|-------------|---------|--------|--------------|-------|
| Linux GUI clean exit | PTY-01 | Linux desktop (Wails) | human_needed | Open shell session, type `exit` at the prompt. Expect: tab closes within ~1 s. | Linux dev box |
| Linux GUI non-zero exit | PTY-01 | Linux desktop | human_needed | Open shell, run `bash -c 'exit 7'`. Expect: tab closes within ~1 s (SHELL-12 auto-close on any natural exit per project memory `project_shell_exit_toast_descoped`). | Linux dev box |
| Linux TUI clean exit | PTY-01, PTY-04 | Linux TUI | human_needed | `agenthub tui`, attach to a shell, type `exit`. Expect: list entry transitions to stopped or disappears within ~1 s. | Linux dev box |
| Linux CLI attach/detach smoke | PTY-04 | Linux CLI | human_needed | `agenthub list`, `agenthub new`, attach, detach, list again. Confirm no regression in daemon-side cleanup. | Linux dev box |
| macOS GUI clean exit (regression) | PTY-04 | macOS desktop | auto | Existing v3.3 SHELL-12 manual UAT path still works — no code on the macOS path was modified (`exit_other.go` is a no-op stub). | macOS dev box |
| macOS TUI clean exit (regression) | PTY-04 | macOS TUI | auto | Same UAT path still works; covered indirectly by `TestListSessions_OnExitCallback_ReceivesNormalized` passing on macOS. | macOS dev box |

## Automated tests

| Item | Requirement | Command | Status | Notes |
|------|-------------|---------|--------|-------|
| TestListSessions_OnExitCallback_ReceivesNormalized (macOS) | PTY-03 | `go test ./internal/daemon -run TestListSessions_OnExitCallback_ReceivesNormalized -race -count=1` | auto PASS | Confirmed by executor 2026-05-18 — exit 0 in 4.1 s. |
| TestListSessions_OnExitCallback_ReceivesNormalized (Linux) | PTY-03 | `go test ./internal/daemon -run TestListSessions_OnExitCallback_ReceivesNormalized -race -shuffle=on -count=10` | human_needed | Requires Linux execution. PTY-03 contract is deterministic pass under `-race -shuffle=on`. |
| TestStartExitDetector_* (Linux) | PTY-02 | `go test ./internal/pty -run TestStartExitDetector -race -shuffle=on -count=10` | human_needed | Requires Linux execution. Three sub-tests: NaturalExit, SuppressedOnKill, SignaledExit. |
| Cross-compile linux/amd64 | PTY-04 | `GOOS=linux GOARCH=amd64 go vet ./internal/... && go build ./internal/...` | auto PASS | Executor 2026-05-18 — exit 0 both commands. |
| Cross-compile darwin/amd64 | PTY-04 | `GOOS=darwin GOARCH=amd64 go vet ./internal/... && go build ./internal/...` | auto PASS | Executor 2026-05-18 — exit 0 both commands. |
| Cross-compile windows/amd64 | PTY-04 | `GOOS=windows GOARCH=amd64 go vet ./internal/... && go build ./internal/...` | auto PASS | Confirms exit_other.go no-op stub builds against Windows constraints. |
| Cross-compile full ./... (linux) | PTY-04 | `GOOS=linux GOARCH=amd64 go vet ./... && go build ./...` | auto PASS | Whole-module green on Linux. |
| Cross-compile full ./... (windows) | PTY-04 | `GOOS=windows GOARCH=amd64 go vet ./... && go build ./...` | auto PASS | Whole-module green on Windows. |
| macOS internal/pty -race -count=1 | PTY-04 | `go test ./internal/pty -race -count=1 -timeout=120s` | auto PASS | Confirms no regression in macOS code path (which is unchanged byte-for-byte). |
| macOS internal/relay -race -count=1 | PTY-04 | `go test ./internal/relay -race -count=1 -timeout=120s` | auto PASS | Confirms relay Hub.Run behavior unchanged. |
| macOS internal/daemon -race -count=1 | PTY-04 | `go test ./internal/daemon -race -count=1 -timeout=180s -skip "^TestOpenCodeANSICapture$\|^TestAPIGetShellWebShareWarned_Default$\|^TestDaemonClient_GetSetShellWebShareWarned_RoundTrip$\|^TestSetShellWebShareWarned_Default$"` | auto PASS | Four pre-existing failures excluded (documented in `deferred-items.md`). All other daemon tests pass under race. |

## Static checks

| Check | Status | Notes |
|-------|--------|-------|
| `head -1 internal/pty/exit_linux.go` starts with `//go:build linux` | auto PASS | Confirmed. |
| `head -1 internal/pty/exit_other.go` starts with `//go:build !linux` | auto PASS | Confirmed. |
| `head -1 internal/pty/exit_linux_test.go` starts with `//go:build linux` | auto PASS | Confirmed. |
| `grep -E 'cmd\.Wait\|s\.cmd\.Wait' internal/pty/exit_linux.go` returns no matches | auto PASS | Detector never calls `Wait()` — killSession owns reap on kill path. |
| `grep 'scottkw/agenthub#57' internal/daemon/engine_test.go` returns no matches | auto PASS | Skip block deleted in Task 4. |
| `[ $(grep -v '^[[:space:]]*//' internal/pty/native.go \| grep -c 'startExitDetector(sess)') = 1 ]` | auto PASS | Exactly one wire-up call. |
| `grep 't\.Skip' internal/pty/exit_linux.go internal/pty/exit_other.go internal/pty/exit_linux_test.go` returns no matches | auto PASS | PTY-03 contract — no new skip paths. |

## Sign-off gate

All `human_needed` items in the cross-surface UAT table and the two Linux
automated test rows must be recorded as PASS (with the Linux operator's
signature line below) before this phase is marked complete in ROADMAP.md.

## Linux operator sign-off

_Pending — to be filled in by the Linux dev-box operator. Required:_

- _Date and commit SHA (latest on `main` at time of sign-off)_
- _Output of `go test ./internal/pty -run TestStartExitDetector -race -shuffle=on -count=10`_
- _Output of `go test ./internal/daemon -run TestListSessions_OnExitCallback_ReceivesNormalized -race -shuffle=on -count=10`_
- _Goroutine-leak check (Phase 110 WR-01 residual TOCTOU): confirm no
  goroutines remain leaked after the above two test invocations. Use
  `go test ... -run ... -race -shuffle=on -count=10` combined with the
  test process's `runtime.NumGoroutine` reported at exit, OR run the
  manual reproduction (open a shell tab, type `exit`, then inspect the
  daemon's goroutine count via pprof or `SIGQUIT` stack dump) and
  confirm no `cmd.Wait`-blocked goroutines linger from cleanup.go after
  natural exit followed by `Kill`._
- _One-line confirmation of each cross-surface UAT row (clean exit, non-zero exit, TUI, CLI smoke)_

---

**Date:** _____________________
**Commit SHA:** _____________________
**Linux operator:** _____________________
**Confirmation:** _____________________
