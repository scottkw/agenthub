---
phase: 109-windows-daemon-named-pipe-ipc
plan: 01
subsystem: infra
tags: [windows, named-pipes, ipc, go, winio, cross-platform, build-tags, third-party-pr]

# Dependency graph
requires:
  - phase: 101 (ShellWebShareWarned routes/persistence)
    provides: existing api.go + client.go structure that PR #53 plugs into
  - phase: 100 (handleListShells route)
    provides: same — non-overlapping line ranges with PR #53 edits
  - phase: 107 (shell-path routes + MaxBytesReader hardening)
    provides: same
provides:
  - Windows daemon listens on named pipe (`\\.\pipe\agenthub-daemon`) via `winio.ListenPipe`
  - Build-tag-split IPC abstraction (`ipc_windows.go` + `ipc_nonwindows.go`) with three helpers — `listenDaemonSocket`, `dialDaemonSocket`, `removeDaemonSocket`
  - Stop-side pipe safety — `removeDaemonSocket` short-circuits on `isWindowsNamedPipe(path)` and returns nil, never calls `os.Remove` on a pipe path
  - Windows regression tests (`TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, `uniqueWindowsPipePath` helper)
  - Tray-icon kernel32 fix — `GetModuleHandleW` loaded from `kernel32.dll` (was incorrectly loaded from `user32.dll`)
  - Author attribution to Alexandre Castro `<im.alexandre07@gmail.com>` (PR #53 contributor) preserved on the three cherry-picked commits
affects: [Phase 110 (Windows release artifacts), Phase 114 (Windows SDK discovery), any future Windows-only work]

# Tech tracking
tech-stack:
  added: []  # No new direct deps — tailscale/go-winio was already a direct dep on main
  patterns:
    - "Build-tag-split platform helper pattern (`*_windows.go` + `*_nonwindows.go`) — joins existing precedents process_unix/process_windows, path/path_windows, notify_theme_unix/notify_theme_windows"
    - "Atomic addr capture before listener close (capture `ln.Addr().String()` BEFORE `ln.Close()`)"

key-files:
  created:
    - internal/daemon/ipc_windows.go
    - internal/daemon/ipc_nonwindows.go
    - docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md
    - .planning/phases/109-windows-daemon-named-pipe-ipc/109-PR53-EVALUATION.md
    - .planning/phases/109-windows-daemon-named-pipe-ipc/109-VERIFICATION.md
    - .planning/phases/109-windows-daemon-named-pipe-ipc/deferred-items.md
  modified:
    - internal/daemon/api.go (listener creation + cleanup swap)
    - internal/daemon/client.go (dial swap)
    - internal/daemon/socket_windows_test.go (new tests + uniqueWindowsPipePath helper + hardened existing tests)
    - tray_windows.go (kernel32 module handle fix)

key-decisions:
  - "Cherry-pick PR #53 (not re-apply): preserves Author: line, satisfies IPC-06 via author-field (not Co-Authored-By trailer)"
  - "Land PR #53 commit 3 (kernel32 fix) as separate logical commit (unrelated bug bundled in same PR)"
  - "Documented three pre-existing ShellWebShareWarned test failures as deferred (not Phase 109 regressions)"

patterns-established:
  - "Build-tag-split IPC helpers: `ipc_windows.go` (//go:build windows) calls `winio.ListenPipe`/`winio.DialPipeContext` while `ipc_nonwindows.go` (//go:build !windows) calls `net.Listen(\"unix\", ...)`/`net.Dialer.DialContext(\"unix\", ...)`"
  - "Author preservation via cherry-pick from third-party PRs: use `git fetch origin pull/N/head:pr-N-temp` + `git cherry-pick <sha>` to keep `Author:` field on each commit; no `Co-Authored-By:` trailer needed"

requirements-completed: [IPC-01, IPC-02, IPC-03, IPC-04, IPC-06]
# Note: IPC-05 is human_needed (Windows 11 UAT, Task 6) — not closed by this plan, deferred to Task 6 checkpoint

# Metrics
duration: 7min
completed: 2026-05-18
---

# Phase 109 Plan 01: Windows Daemon Named-Pipe IPC Summary

**Third-party PR #53 (Alexandre Castro) cherry-picked clean — Windows daemon now listens on `\\.\pipe\agenthub-daemon` via `winio.ListenPipe` instead of crashing on `net.Listen("unix", ...)`; build-tag-split (`ipc_windows.go` + `ipc_nonwindows.go`) keeps macOS/Linux Unix-socket behavior byte-identical.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-05-18T14:34:53Z
- **Completed:** 2026-05-18T14:42:10Z
- **Tasks:** 5 of 6 (Task 6 is a human-verify checkpoint deferred to Windows 11 hardware UAT)
- **Files modified:** 10 (4 created, 4 modified, plus 2 planning artifacts + 1 deferred-items log)

## Accomplishments

- **PR #53 landed clean** via cherry-pick on phase branch `phase-109-windows-named-pipe-ipc`. Three commits all attributed to `Alexandre Castro <im.alexandre07@gmail.com>` (IPC-06 PASS, automated check confirms count of 3).
- **Empirical clean-merge verified** before cherry-pick: `git merge-tree --write-tree main pr-53-temp` produced exit 0, tree `2b5c3f25…`, auto-merging `internal/daemon/api.go` and `internal/daemon/client.go` only — exactly as research predicted.
- **Build-tag IPC abstraction in place** — `listenDaemonSocket(path)`, `dialDaemonSocket(ctx, path)`, `removeDaemonSocket(path)` triplet wired through `api.go::API.Start`, `api.go::API.Stop`, and `client.go::NewDaemonClient`. Windows side uses `winio.ListenPipe`/`winio.DialPipeContext`; non-Windows side uses `net.Listen("unix", ...)`/`net.Dialer.DialContext`.
- **Stop-side pipe safety** — `removeDaemonSocket` on Windows short-circuits via `if isWindowsNamedPipe(path) { return nil }`. No `os.Remove` ever called on a `\\.\pipe\…` path.
- **Cross-compile to Windows passes** (`GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...` exit 0) — proves the build-tag split compiles on the Windows target without a Windows host.
- **Macos-side regression-free** — no new test failures introduced by the cherry-pick. (The three pre-existing `ShellWebShareWarned` failures continue exactly as on `main`; see Deviations / Issues.)
- **VERIFICATION.md scaffolded** with three discrete `human_needed` items (WIN-GUI-01 / WIN-CLI-01 / WIN-TUI-01) for Windows 11 hardware UAT (IPC-05).
- **Tray icon kernel32 fix** landed as separate logical commit (PR's third commit, unrelated to IPC but a real Windows bug — `GetModuleHandleW` was being loaded from `user32.dll`).

## Task Commits

Each task was committed atomically. Cherry-picks preserve original PR commit attributions:

1. **Task 1: Re-verify clean merge and write PR #53 evaluation note** — no immediate commit; planner doc deferred to Task 5 commit per plan design.
2. **Task 2 (a): Cherry-pick PR commit 1 (design doc)** — `68b2421` (docs, Author: Alexandre Castro)
3. **Task 2 (b): Cherry-pick PR commit 2 (IPC fix + tests)** — `2f25e63` (fix, Author: Alexandre Castro)
4. **Task 3: Cherry-pick PR commit 3 (kernel32 tray fix)** — `fc50cd4` (fix, Author: Alexandre Castro)
5. **Task 4: Clean up pr-53-temp + write VERIFICATION scaffold** — no commit yet (rolled into Task 5)
6. **Task 5: Commit planner-authored docs** — `84f4520` (docs, Committer: Ken Scott; includes deferred-items.md amended in)

**Plan metadata:** subsequent metadata commit will follow this SUMMARY.md write.

Full author audit:
```
$ git log --format='%an <%ae>' main..HEAD | sort -u
Alexandre Castro <im.alexandre07@gmail.com>
Ken Scott <kenscott@gmail.com>
$ git log --format='%an' main..HEAD | grep -c "Alexandre Castro"
3
```

## Files Created/Modified

**Created (by PR #53 cherry-picks):**
- `internal/daemon/ipc_windows.go` — `winio.ListenPipe` / `winio.DialPipeContext` / no-op-on-pipe `removeDaemonSocket` helpers; `//go:build windows`.
- `internal/daemon/ipc_nonwindows.go` — `net.Listen("unix", ...)` / `net.Dialer.DialContext` / `os.Remove` helpers; `//go:build !windows`.
- `docs/superpowers/specs/2026-05-17-windows-daemon-named-pipe-ipc-design.md` — PR-author-original design spec (retained verbatim).

**Created (by planner / executor):**
- `.planning/phases/109-windows-daemon-named-pipe-ipc/109-PR53-EVALUATION.md` — cherry-pick decision rationale + merge-tree evidence + IPC-06 attribution mechanic.
- `.planning/phases/109-windows-daemon-named-pipe-ipc/109-VERIFICATION.md` — automated checks + three `human_needed` UAT items + requirements coverage map.
- `.planning/phases/109-windows-daemon-named-pipe-ipc/deferred-items.md` — three pre-existing `ShellWebShareWarned` test failures (NOT Phase 109 regressions).

**Modified (by PR #53 cherry-picks):**
- `internal/daemon/api.go` — `net.Listen("unix", socketPath)` → `listenDaemonSocket(socketPath)`; `_ = os.Remove(a.ln.Addr().String())` → `addr := a.ln.Addr().String(); …; _ = removeDaemonSocket(addr)`.
- `internal/daemon/client.go` — dial closure now calls `dialDaemonSocket(ctx, socketPath)` instead of inline `net.Dialer.DialContext(ctx, "unix", socketPath)`.
- `internal/daemon/socket_windows_test.go` — added `TestAPIStart_WindowsNamedPipeHealth`, `TestAPIStop_WindowsNamedPipe`, `uniqueWindowsPipePath` helper; existing `TestCleanupStaleSocket_WindowsPipe_*` tests hardened to use unique per-run pipe names.
- `tray_windows.go` — added `kernel32 = windows.NewLazySystemDLL("kernel32.dll")` declaration; switched `pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")` (was wrongly `user32.NewProc(...)`).

## Decisions Made

- **Cherry-pick over re-apply.** Per PR-EVAL note: cherry-pick preserves `Author:` (satisfying IPC-06 via the first arm of its OR clause — "trailer on merged/cherry-picked commits, OR commit-message attribution if re-applied"). Re-apply would require manual trailer ceremony with no upside given the empirical clean-merge proof.
- **Keep the kernel32 tray fix as a separate logical commit.** It is unrelated to IPC but bundled in the same PR. Honest audit trail requires per-commit granularity, not a fused merge commit.
- **Do NOT add `Co-Authored-By: Alexandre Castro` trailer.** Trailer would muddle the audit (the `Author:` line already carries his attribution canonically; trailers signal "additional contributor when primary author is someone else" semantics).
- **Document pre-existing test failures rather than fix them.** Three `ShellWebShareWarned` failures pre-exist on `main` (Phase 101 territory); per executor scope-boundary rule, these are out-of-scope for Phase 109. Tracked in `deferred-items.md` for a future patch-release bug-sweep plan.

## Deviations from Plan

### Pre-existing test failures (NOT introduced by Phase 109 — scope-boundary call)

**1. [Scope-boundary call - NOT a Rule-1 auto-fix] Three pre-existing test failures in `internal/daemon`**
- **Found during:** Task 2 (running `go test -race -short ./internal/daemon/` after cherry-picks)
- **Issue:** Three failures —
  - `TestAPIGetShellWebShareWarned_Default` (api_test.go:1592: `default value: got true, want false`)
  - `TestDaemonClient_GetSetShellWebShareWarned_RoundTrip` (api_test.go:1638: `initial value: got true, want false`)
  - `TestSetShellWebShareWarned_Default` (engine_test.go:905: `default GetShellWebShareWarned: got true, want false`)
- **Investigation:** Stashed phase-branch work, checked out plain `main` (`9cc1087`), re-ran the same tests — all three FAIL on `main` BEFORE any Phase 109 commit lands. Root cause is `ShellWebShareWarned` default-value logic introduced by `dbd95a7 feat(101-01)`, not by Phase 109. Phase 109 does not touch the WebShareWarned default or its tests.
- **Decision (per executor scope-boundary rule):** Do NOT auto-fix. "Only auto-fix issues DIRECTLY caused by the current task's changes. Pre-existing failures in unrelated files are out of scope. Log out-of-scope discoveries to deferred-items.md."
- **Files documented (not modified):** `.planning/phases/109-windows-daemon-named-pipe-ipc/deferred-items.md`
- **Verification:** Plan's must_haves frontmatter still validates green (build-tag split present, helpers wired, tests compile, cross-compile passes, author audit shows 3 Alexandre commits). The plan does NOT depend on these three pre-existing tests passing.
- **Committed in:** `84f4520` (planning artifact in Task 5 commit).
- **Follow-up:** File a bug on `scottkw/agenthub` titled along the lines of "ShellWebShareWarned default is `true`, tests expect `false`" — determine intent (should default be `false` per tests, or `true` per live code) and fix in a dedicated patch-release bug-sweep plan.

### Local-environment pollution (security-review/ stray Go files)

**2. [Scope-boundary call - NOT a Rule-1 auto-fix] `go build ./...` failed due to stray `security-review/` workspace artifacts**
- **Found during:** Task 2 (running `GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...`)
- **Issue:** Local working tree has `/Users/ken/dev/agenthub/security-review/` (gitignored, dated April 19 — third-party security review artifacts) containing two stray Go files with conflicting package declarations (`relay` and `webserver` in the same dir). `go build ./...` walks into the dir despite `.gitignore` (Go's package discovery doesn't honor `.gitignore`) and fails with `found packages relay (…) and webserver (…) in /Users/ken/dev/agenthub/security-review`.
- **Investigation:** Confirmed `security-review/` is in `.gitignore` (rule: `security-review/  # Third-party security review artifacts (not committed)`). Confirmed it pre-dates Phase 109 (file mtimes April 19, before phase work started).
- **Decision:** Do NOT modify the gitignore'd local workspace. Worked around by temporarily renaming the directory during cross-compile and full-suite runs, then restoring. The repository state is unchanged.
- **Verification:** Cross-compile to Windows exits 0 with the directory moved aside.
- **Committed in:** N/A (no repo change required).
- **Follow-up:** None required for Phase 109. The user may want to relocate the security-review/ artifacts outside the Go module root in the future to avoid this snag.

---

**Total deviations:** 2 documented (both scope-boundary calls — neither is a Rule 1-3 auto-fix; both are out-of-scope-by-rule).
**Impact on plan:** Zero. All plan must_haves validate green; phase branch shape is canonical (3 Alexandre commits + 1 planner commit + this SUMMARY commit pending); IPC-01..04 + IPC-06 closed by this plan; IPC-05 deferred to Task 6 Windows-hardware UAT per plan design.

## Issues Encountered

None during the planned cherry-pick flow itself. Merge-tree prediction held: `git cherry-pick 6f312e1 410586d` produced "Auto-merging internal/daemon/api.go / client.go" with no conflict markers, exactly as research predicted. The third cherry-pick (`d1f0cdf`) applied to a single file (`tray_windows.go`) without incident.

The two deviations above (pre-existing `ShellWebShareWarned` failures and local-env `security-review/` pollution) are environmental artifacts, not Phase 109 issues. Both are documented for follow-up.

## User Setup Required

None — no external services configured. The phase is a transport substitution in a single Go module.

## Verification Status

### Automated (this plan)
- IPC-01 / IPC-02 / IPC-03 — code-complete on phase branch; structural verification passes (helper wiring inspection PASS, build-tag wiring inspection PASS).
- IPC-04 — Windows-only tests added in `socket_windows_test.go`; compile-clean on macOS via `GOOS=windows GOARCH=amd64 go build -tags wailsassets ./...`; runtime verification deferred to Windows host (Task 6 Step 1).
- IPC-06 — **CLOSED**. `109-PR53-EVALUATION.md` records the decision; cherry-pick preserves `Author: Alexandre Castro <im.alexandre07@gmail.com>` on all three PR commits (verified count of 3).

### Human (Task 6 — pending Windows 11 hardware)
- `human_needed: WIN-GUI-01` (tray icon + GUI new-session + web-share toggle on Windows 11)
- `human_needed: WIN-CLI-01` (`agenthub.exe daemon status / new / list` on Windows 11)
- `human_needed: WIN-TUI-01` (TUI session list + attach + detach on Windows 11)
- Windows-only unit test execution on the Windows host (finalizes IPC-04)
- macOS / Linux cross-platform regression smoke (after Windows passes)

Per project memory `feedback_cross_surface_parity`: Phase 109 is **complete-pending-human-UAT**. Release-eligibility requires WIN-GUI-01 + WIN-CLI-01 + WIN-TUI-01 all flip to `human_verified`.

## Next Phase Readiness

- **Phase 109 plan 01 deliverables on branch `phase-109-windows-named-pipe-ipc`** ready for review and human Windows UAT.
- **Phase 110** (Windows release artifacts — installer / signing / WinGet) can begin design work; it depends on a green Phase 109 IPC fix (this plan's deliverable) to produce a functional Windows binary.
- **Pre-existing `ShellWebShareWarned` failures** documented in `deferred-items.md` — should be addressed in a separate bug-sweep plan; does not block Phase 109 or Phase 110.

## Self-Check: PASSED

- Files created exist: `109-PR53-EVALUATION.md`, `109-VERIFICATION.md`, `deferred-items.md`, `ipc_windows.go`, `ipc_nonwindows.go`, `socket_windows_test.go` (modified), `api.go` (modified), `client.go` (modified), `tray_windows.go` (modified), design spec doc. **All present**.
- Commits exist:
  - `68b2421` (PR commit 1 cherry-pick) — confirmed via `git log --format='%H' main..HEAD`
  - `2f25e63` (PR commit 2 cherry-pick) — confirmed
  - `fc50cd4` (PR commit 3 cherry-pick) — confirmed
  - `84f4520` (planner commit incl. deferred-items.md) — confirmed
- Author count: `git log --format='%an' main..HEAD | grep -c "Alexandre Castro"` returns `3`. PASS.

---
*Phase: 109-windows-daemon-named-pipe-ipc*
*Plan: 01*
*Completed: 2026-05-18*
