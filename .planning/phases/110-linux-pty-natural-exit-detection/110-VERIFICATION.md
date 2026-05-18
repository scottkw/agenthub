# Phase 110 Verification

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
- _One-line confirmation of each cross-surface UAT row (clean exit, non-zero exit, TUI, CLI smoke)_

---

**Date:** _____________________
**Commit SHA:** _____________________
**Linux operator:** _____________________
**Confirmation:** _____________________
