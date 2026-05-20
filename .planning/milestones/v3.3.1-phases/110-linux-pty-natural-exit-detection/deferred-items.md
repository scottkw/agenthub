# Phase 110 — Deferred / Out-of-scope Discoveries

These items were observed during Phase 110 execution but are pre-existing,
out-of-scope by the executor scope-boundary rule. They are documented for
future bug-sweep / environment-cleanup attention.

## 1. Local workspace pollution: `security-review/` mixed-package directory

- **Severity:** Environmental (workstation only); not in git.
- **Symptom:** `go vet ./...` and `go build ./...` from the repo root fail
  with `found packages relay (...) and webserver (...) in .../security-review`
  because the directory contains two stray `*_test.go` files that declare
  different `package` lines in the same directory.
- **Status:** Pre-existing. Documented in Phase 109 SUMMARY under
  "Local-environment pollution". `security-review/` is in `.gitignore`.
- **Workaround used during Phase 110:** Temporarily renamed the directory
  during cross-compile verification, then restored. Repository state
  unchanged. No commit needed.
- **Suggested follow-up (user choice):** Relocate the security-review
  artifacts outside the Go module root, e.g. `../agenthub-security-review/`.

## 3. Pre-existing test failures inherited from main (NOT Phase 110 regressions)

Confirmed by running the affected tests against commit `57eb238`
(the commit immediately before any Phase 110 work) — all four fail
identically there. Documented in Phase 109 SUMMARY for the first three;
the fourth (OpenCodeANSICapture race) appears to be a separate
pre-existing issue.

- `TestAPIGetShellWebShareWarned_Default` — default value mismatch
  (Phase 101 territory).
- `TestDaemonClient_GetSetShellWebShareWarned_RoundTrip` — same root cause.
- `TestSetShellWebShareWarned_Default` — same root cause.
- `TestOpenCodeANSICapture` — race detected in goroutine 1112 (created
  at `opencode_ansi_test.go:66`). Not related to PTY/daemon-exit code.

**Phase 110 verification approach:** run `go test ./internal/daemon -race
-count=1 -skip "^TestOpenCodeANSICapture$|^TestAPIGetShellWebShareWarned_Default$|^TestDaemonClient_GetSetShellWebShareWarned_RoundTrip$|^TestSetShellWebShareWarned_Default$"`
— the rest of the suite passes cleanly on macOS with Phase 110 changes
applied.

**Follow-up:** File / track each in `scottkw/agenthub` as a separate
bug-sweep ticket. Out of scope for the v3.3.1 PTY-01..04 release-blocker.

## 4. cleanup.go `cmd.Wait` goroutine may leak after detector-reaped child (WR-03)

- **Severity:** Pre-existing in `internal/pty/cleanup.go`; surfaces more
  often now that Phase 110's detector externally reaps the child on natural
  exit before `t.Cleanup`'s `b.Kill` runs.
- **Symptom:** In `TestStartExitDetector_NaturalExit` and
  `TestStartExitDetector_SignaledExit`, the t.Cleanup-registered `b.Kill`
  fires after the detector has already reaped the child via Wait4. The
  goroutine spawned at cleanup.go:45-52 calls `s.cmd.Wait()`, which on
  Linux may hang on a pidfd that has been externally reaped (or return
  ECHILD). The outer select waits ~2-3s and proceeds, but the inner
  goroutine leaks for the lifetime of the test process.
- **Status:** Pre-existing in cleanup.go; bug is the lack of a hard
  timeout on the inner `cmd.Wait` goroutine. Tests still PASS — concern
  is goroutine accounting only. Phase 110 REVIEW WR-03 flagged this as
  out of scope for the patch release.
- **Suggested follow-up:** File a bug-sweep ticket in `scottkw/agenthub`
  for cleanup.go to add a context-based timeout / abandon path on the
  inner `cmd.Wait`. Confirm under Linux `-race` whether the leak
  surfaces (test goroutine counter > expected at exit).

## 2. `GOOS=darwin GOARCH=amd64 go vet ./...` from repo root prints a CGO error

- **Severity:** Environmental (executor host without Wails C toolchain).
- **Symptom:** Without `-tags wailsassets`, vet of the root package emits
  `vet: ./app.go:100:5: a.setDockVisible undefined` even though tray.go is
  build-tagged `//go:build darwin` and defines the method. With
  `-tags wailsassets`, exit code is 0 (production tag path is healthy).
- **Status:** Pre-existing on `main` before any Phase 110 commit. Verified
  by stashing Phase 110 changes and re-running — same error appears.
- **Workaround:** Phase 110 verifies cross-compile on the scoped
  `./internal/...` package set (which is what Phase 110 actually modifies);
  all three platforms exit 0. The root-package CGO snag is unrelated to
  PTY/daemon code.
- **Suggested follow-up:** Either add the missing Wails CGO toolchain to
  CI/dev docs or normalize on `-tags wailsassets` for all darwin vet runs.
