# Phase 117 Verification

**Closes:** v3.3.1 milestone audit "Phase-discovered deferrals" section.
**Requirements:** PAPER-01, PAPER-02, PAPER-03.
**Phase commit:** `<TBD>` (single fix commit covering all three paper-cuts).

## Automated tests

| Item | Requirement | Command | Status |
|------|-------------|---------|--------|
| TestRenderNewSessionModal_ZeroDimensions (5 sub-cases) | PAPER-01 | `go test ./internal/tui -run 'TestRenderNewSessionModal_ZeroDimensions' -count=1` | PASS — all 5 sub-cases (0,0 / 0,24 / 120,0 / -1,24 / 120,-1) green; would panic pre-fix. |
| TestRenderNewSessionModal_NoDetectedCLIs | PAPER-01 | `go test ./internal/tui -run 'TestRenderNewSessionModal_NoDetectedCLIs' -count=1` | PASS — defense-in-depth verifies Phase 109 `agentEntries` always-include-Shell contract. |
| TestKillSession_NoGoroutineLeak_NormalCase | PAPER-03 | `go test ./internal/pty -run 'TestKillSession_NoGoroutineLeak' -count=1 -v` | PASS — `baseline=2 final=2 delta=+0` (5 sessions, tolerance 5) — within bounds. |
| Full TUI suite under race | PAPER-01 | `go test ./internal/tui -race -count=1` | PASS — 1.28 s. |
| Full PTY suite under race | PAPER-03 | `go test ./internal/pty -race -count=1` | PASS — 2.67 s. |

## Manual UAT (PAPER-02 release gate)

**Operator:** ken@kscott
**Date:** 2026-05-19
**Build:** `go build -o bin/agenthub-v3.3.1-paper02 .` against HEAD with PAPER-02 fix applied.
**Pre-fix:** Local terminal scrollback (`agenthub attach <sid>` typed-command, daemon log lines, PRE-ATTACH MARKERs) remained visible above the attached session prompt.
**Post-fix:** Local terminal cleared immediately on `attach` entry; only the session's `bash-3.2$` prompt + status bar visible.

**Reproduction:**

```bash
/Users/ken/dev/agenthub/bin/agenthub-v3.3.1-paper02 daemon run &
sleep 1
/Users/ken/dev/agenthub/bin/agenthub-v3.3.1-paper02 new bash ~      # prints SID
echo "PRE-ATTACH MARKER 1"; echo "PRE-ATTACH MARKER 2"; echo "PRE-ATTACH MARKER 3"
/Users/ken/dev/agenthub/bin/agenthub-v3.3.1-paper02 attach <SID>    # screen clears
```

**Evidence:** `uat-evidence/attach-screen-cleared.png` — post-fix screenshot in Warp showing only the session's `bash-3.2$` prompt and the status bar at the bottom (`ken | /bin/bash | Kens-Personal-MacBook-Air.local | Ctrl-\ to detach | 1:40`). The PRE-ATTACH MARKERs and daemon log lines are gone.

**Detach key:** `Ctrl-\` (per status bar — Phase 117 itself unchanged from prior behavior).

## Static checks

| Check | Status | Notes |
|-------|--------|-------|
| `internal/tui/modal.go::renderNewSessionModal` includes `if m.width <= 0 \|\| m.height <= 0 { return "" }` guard | PASS | Lines 60-62 of post-fix file. |
| `internal/tui/modal_test.go` is a new file with `TestRenderNewSessionModal_*` functions | PASS | 2 test functions, 6 total test cases. |
| `cmd_attach.go` includes `term.IsTerminal(int(os.Stdout.Fd()))` guard before `stdout.Write([]byte("\x1b[2J\x1b[H"))` | PASS | Lines 124-128 of post-fix file. |
| `internal/pty/cleanup.go` goroutine comment no longer says "Give up — process is truly stuck" | PASS | Replaced with bounded-lifetime explanation that points to the comment block above the goroutine. |
| `internal/pty/cleanup_test.go` is a new file with `//go:build !windows` and `TestKillSession_NoGoroutineLeak_NormalCase` | PASS | 80 LoC, single regression test. |

## Sign-off

**PAPER-01** ✅ — `renderNewSessionModal` no longer panics with zero or negative dimensions. 6 test cases green; existing TUI suite unchanged.

**PAPER-02** ✅ — `agenthub attach` clears the local terminal on entry when stdout is a TTY. Manually verified in Warp; screenshot in `uat-evidence/attach-screen-cleared.png`. Non-TTY behavior (CI / pipes) unchanged.

**PAPER-03** ✅ — `killSession` goroutine is documented as bounded-lifetime (not a leak); regression test confirms `runtime.NumGoroutine()` returns to baseline after 5 session create-and-kill cycles.

**Resume signal:** `paper-cuts-cleared`.

---

**Date:** 2026-05-19
**Commit range:** Single commit covering all 3 paper-cuts.
**Tests:** 7 new automated tests (6 TUI + 1 PTY) + 1 manual UAT.
**Closes:** Last 3 deferred items from v3.3.1 milestone audit "Phase-discovered deferrals" section.
