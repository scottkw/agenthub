---
phase: 110-linux-pty-natural-exit-detection
reviewed: 2026-05-18T00:00:00Z
depth: deep
files_reviewed: 5
files_reviewed_list:
  - internal/pty/exit_linux.go
  - internal/pty/exit_other.go
  - internal/pty/exit_linux_test.go
  - internal/pty/native.go
  - internal/daemon/engine_test.go
findings:
  critical: 0
  blocker: 0
  warning: 3
  info: 4
  total: 7
status: issues_found
---

# Phase 110: Code Review Report

**Reviewed:** 2026-05-18
**Depth:** deep
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Phase 110 implements a Linux-only Wait4-polling exit detector to fix Issue #57 (Linux PTY master `Read()` blocks indefinitely after child exit). The implementation is small (~120 LOC new), surgical, and well-aligned with the locked PLAN decisions (Q1 detector-owns-CancelContext, Q2 keep 100ms sleep, Q3 add unit tests). All review-focus items pass:

- **Wait4 + WNOHANG correctness:** Signature, return-value handling, and POSIX 128+signal exit-code derivation are correct.
- **PID-recycle race mitigation:** The locked sequence (SetExitCode -> CancelContext -> snapshot pty under mu -> Close -> return) is implemented exactly as specified in `decisions_locked Q1`.
- **No double-Wait:** Detector reads `IsKilled()` under mutex each tick before calling `Wait4`, and returns silently when kill path owns the session.
- **Build-tag split:** Both files carry the correct `//go:build` constraints on the first non-blank line; cross-compile clean on linux/darwin/windows amd64.
- **No `t.Skip` in `exit_linux_test.go`:** Confirmed zero matches.
- **engine_test.go skip flip:** Linux skip block deleted, Windows skip (lines 1297-1299) preserved, sh LookPath skip (1300-1303) preserved.
- **Cancel-after-reap idempotence:** `context.CancelFunc` is safe for repeat calls; the existing engine.go:337 100ms sleep adds defensive latency but does not hang.

No BLOCKER findings. Three WARNING findings flag genuine concerns about test fragility, an unchecked error return from `cmd.Wait()` after external reap (pre-existing but newly relevant), and a TOCTOU window between `IsKilled()` check and `Wait4`. Four INFO findings document minor code-quality observations.

## Warnings

### WR-01: TOCTOU race between IsKilled() check and syscall.Wait4

**File:** `internal/pty/exit_linux.go:61-66`
**Issue:** The detector checks `s.IsKilled()` (mutex-protected) at the top of each tick, then immediately calls `syscall.Wait4(pid, ...)`. If another goroutine (e.g., the caller of `b.Kill()`) calls `MarkKilled()` in the window between the IsKilled check and the Wait4 syscall, the detector still reaps the child. This is the classic check-then-act TOCTOU pattern.

Consequence on the kill path: detector reaps the child first; then `killSession` (cleanup.go:46) calls `s.cmd.Wait()`. Go's `os/exec.Cmd.Wait` internally tries to reap via the kernel's wait facility — since the child has already been reaped by our external Wait4, `Process.Wait` may return ECHILD or hang on a pidfd that has been consumed. Either way:
- killSession's `cmd.Wait` likely fails -> ProcessState stays nil -> killSession's `SetExitCode` (cleanup.go:48-50) is skipped -> the detector's already-cached exit code is preserved. Behaviour is correct.
- However, the goroutine spawned at cleanup.go:45-52 may **leak** if `cmd.Wait` blocks indefinitely on a pidfd that will never signal because the process is already reaped.

The threat model (T-110-01) acknowledges this race as "mitigated" by the IsKilled check, but the check is racy. The window is microseconds in practice, but `-race -shuffle=on` runs can stretch goroutine scheduling significantly.

**Fix:** Two options, in priority order:
1. (Preferred, low-risk) Document the race explicitly in `startExitDetector`'s doc comment and add a `linux operator sign-off` item in VERIFICATION.md confirming no goroutine leaks observed under `go test -race -count=10`.
2. (Stronger) Acquire `s.mu` for the IsKilled-check + Wait4 atomic block — but this requires adding a new mutex-protected method on Session (e.g., `tryReapIfNotKilled`) and is a larger refactor that exceeds the patch-release scope.

Recommend Option 1 for v3.3.1; track Option 2 as a follow-up if Linux CI sees leaked goroutines under `-race`.

### WR-02: `TestStartExitDetector_SuppressedOnKill` is timing-sensitive

**File:** `internal/pty/exit_linux_test.go:71-105`
**Issue:** The test calls `b.Create()` (which spawns the detector goroutine) and then immediately calls `sess.MarkKilled()`. The detector uses `time.NewTicker(100ms)` whose first tick fires at ~100ms after the goroutine starts. The test relies on `MarkKilled()` being set BEFORE the first tick.

Under normal scheduling this is safe (test thread runs MarkKilled within microseconds of Create returning). But under `-race -shuffle=on -count=10` (the Linux CI gate per PTY-02), goroutine scheduling can be perturbed enough that the first tick fires before MarkKilled is set.

If the first tick fires first:
- `IsKilled()` returns false.
- `Wait4(pid, &status, WNOHANG)` is called on the still-running `sleep 5`.
- `wpid == 0, err == nil` -> `continue` to the next tick.
- By the next tick (200ms later), MarkKilled is set -> detector returns silently.

So the test still passes in this scenario, BUT only because `sleep 5` is long enough. The test's correctness invariant (detector did NOT call Wait4 on a killed session) is actually weaker than what the test asserts — the test merely verifies that ExitCode stays -1, which can also be true if Wait4 was called but returned wpid=0.

**Fix:** Either (a) tighten the test to manually construct a Session and call `MarkKilled()` BEFORE invoking `startExitDetector` directly (the PLAN's task 3 explicitly permits this — "construct Session manually (or via Create then immediately MarkKilled before any natural exit could fire — sleep 5 buys us time)"); or (b) add a comment acknowledging the assertion is weaker than the test name implies and clarify what is actually proven.

Recommend (a) — it makes the test's invariant deterministic and removes timing fragility from a release-blocking PTY-02 verification path.

### WR-03: Test cleanup race — `b.Kill` after detector already reaped child

**File:** `internal/pty/exit_linux_test.go:40-42, 125-127`
**Issue:** Both `TestStartExitDetector_NaturalExit` and `TestStartExitDetector_SignaledExit` register `t.Cleanup(func() { _ = b.Kill(sess.ID) })`. By the time t.Cleanup runs, the detector has already:
1. Reaped the child via Wait4 (kernel-level reaped, PID released).
2. Called `s.cancel()`.
3. Closed the PTY master + slave.

`b.Kill` then runs:
1. `sess.MarkKilled()` — sets killed=true.
2. `sess.SetState(StateStopped)`.
3. `b.registry.Remove(id)`.
4. `killSession(sess)`:
   a. `syscall.Kill(-pgid, SIGHUP)` — returns ESRCH (no such process); ignored.
   b. `s.pty.Close()` — already closed; returns ErrClosed via errors.Join; ignored.
   c. Spawns goroutine that calls `s.cmd.Wait()` — this may hang or return ECHILD.
   d. Outer select waits up to 2s + 1s for `done` channel.

If `cmd.Wait()` hangs (pidfd already reaped externally), the inner goroutine leaks for the lifetime of the test process. This is **pre-existing behaviour** in cleanup.go and not Phase 110-introduced, but Phase 110's detector now triggers this code path on every natural-exit test run.

**Fix:** Out of scope for Phase 110 (cleanup.go is pre-existing, the bug is in killSession's lack of a hard timeout on the `cmd.Wait` goroutine). Document in `deferred-items.md` and file a follow-up bug-sweep ticket if `-race` runs surface the leak. The test still passes; the concern is goroutine accounting only.

## Info

### IN-01: `if s == nil` check at start of `startExitDetector` is unreachable

**File:** `internal/pty/exit_linux.go:46`
**Issue:** `startExitDetector(sess)` is called from exactly one site (`native.go:101`) which has just constructed `sess`. The nil check is defensive but adds dead code that can never trigger in normal flow.
**Fix:** Optional — remove the `s == nil` portion of the guard if you want to eliminate dead code, OR keep it as documented defensive programming. Either choice is reasonable; flagging only for completeness.

### IN-02: `pid := s.cmd.Process.Pid` captured outside the goroutine — clarify rationale

**File:** `internal/pty/exit_linux.go:52`
**Issue:** The code comment explains "Capture PID once, outside the goroutine — avoids repeated field access and ensures we always Wait4 on the original PID even if s.cmd is mutated elsewhere." But `s.cmd` is never mutated after Session construction (verified via grep across `internal/pty/*.go`). The comment is technically true but slightly misleading — the real reason is to avoid a data race on `s.cmd` (a write without mutex by Create, read by the detector goroutine) by synchronizing the read with the goroutine's happens-before-spawn edge.
**Fix:** Optional — tighten the comment to reflect the actual race-freedom rationale. Example: "Capture PID before spawning to ensure the read of s.cmd is synchronized with its construction in Create() via the goroutine-spawn happens-before edge."

### IN-03: `for range ticker.C` pattern delays first check by one tick (100ms)

**File:** `internal/pty/exit_linux.go:55-58`
**Issue:** Using `time.NewTicker` means the first iteration of the loop fires at `t=100ms`, not immediately. For a short-lived child that exits in <100ms (e.g., `exit 0` from a pre-spawned sh), the user-perceptible exit latency includes this initial delay plus the Wails 500ms poll. The PLAN's success criteria allow "within ~1 s", so this is fine.

If sub-100ms detection is ever required, the pattern `for { ... ; <-ticker.C }` (check immediately, then wait) would shave one tick. Out of scope for v3.3.1.

**Fix:** None required for Phase 110. Note for future tuning if user-perceptible latency becomes a complaint.

### IN-04: Magic number 128 for POSIX signal-exit offset is inlined

**File:** `internal/pty/exit_linux.go:84`
**Issue:** `exitCode = 128 + int(status.Signal())` uses the literal 128. POSIX convention is well-documented and the inline comment explains it, but a named constant (e.g., `posixSignalExitOffset = 128`) would improve discoverability if the convention is ever questioned in code review by someone unfamiliar with POSIX exit semantics.
**Fix:** Optional cosmetic — add a named constant. Not a defect.

---

## Verification Matrix

| Review focus item | Status | Notes |
|-------------------|--------|-------|
| Wait4 + WNOHANG correctness | PASS | Signature `Wait4(pid, &status, WNOHANG, nil)` correct; wpid==0/err==nil/err!=nil branches handled; POSIX 128+signal convention applied |
| PID-recycle race mitigation (Q1 sequence) | PASS | Implemented in exit_linux.go:93-101 exactly per locked decision: SetExitCode -> CancelContext -> mu.Lock/snapshot/Unlock -> Close |
| No double-Wait race (IsKilled gate) | PASS with caveat | IsKilled() is mutex-correct (session.go:115-119); WR-01 documents the residual TOCTOU |
| Build-tag split | PASS | exit_linux.go: `//go:build linux` line 1; exit_other.go: `//go:build !linux` line 1; signatures match |
| No t.Skip in exit_linux_test.go | PASS | `grep -E 't\.Skip' internal/pty/exit_linux_test.go` returns no matches |
| engine_test.go skip flip | PASS | Linux skip block (was at 1300-1307) gone; Windows skip (1297-1299) and sh LookPath skip (1300-1303 post-flip) preserved; `runtime.GOOS == "linux"` reference removed |
| Resource leak (detector goroutine termination) | PASS | Goroutine terminates on natural exit, kill, or Wait4 error; ticker.Stop deferred |
| Cancel-after-reap edge case | PASS | context.CancelFunc safe for repeat calls; engine.go:337 100ms sleep is defensive latency, not a hang risk |

## Cross-compile gate

```
GOOS=linux  GOARCH=amd64 go vet ./internal/pty/...  -> PASS
GOOS=darwin GOARCH=amd64 go vet ./internal/pty/...  -> PASS
GOOS=windows GOARCH=amd64 go vet ./internal/pty/... -> PASS
```

---

_Reviewed: 2026-05-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
