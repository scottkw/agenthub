# Phase 117 Plan 01 — Summary

**Status:** Complete (2026-05-19)
**Closes:** v3.3.1 milestone audit "Phase-discovered deferrals" section
**Requirements satisfied:** PAPER-01, PAPER-02, PAPER-03

## What shipped

Three unrelated paper-cuts closed in one commit:

| ID | What | Where |
|----|------|-------|
| PAPER-01 | Defensive guard against zero/negative dimensions in `renderNewSessionModal` (no more `lipgloss.Place` panic) | `internal/tui/modal.go` + new `internal/tui/modal_test.go` |
| PAPER-02 | ANSI Erase-in-Display + Cursor-Home on `agenthub attach` entry (clean canvas, no scrollback bleed) | `cmd_attach.go` |
| PAPER-03 | Bounded-lifetime goroutine comment + regression test (was misleadingly comment-tagged as a "leak") | `internal/pty/cleanup.go` + new `internal/pty/cleanup_test.go` |

## Commits

- `<TBD-paper-cuts-fix>` fix(117): three paper-cut bugs (PAPER-01..03)

## Test results

- 6 new TUI tests (`TestRenderNewSessionModal_*`): GREEN
- 1 new PTY test (`TestKillSession_NoGoroutineLeak_NormalCase`): GREEN — `baseline=2 final=2 delta=+0` on 5 sessions
- Full TUI suite under `-race`: PASS (1.28 s)
- Full PTY suite under `-race`: PASS (2.67 s)
- `go build ./...`: clean (env'l `security-review/` mixed-package excluded per audit)

## Manual UAT

PAPER-02 manually verified on macOS with Warp terminal at HEAD with PAPER-02 fix applied. Pre-fix scrollback (3 PRE-ATTACH MARKERs + daemon log lines) cleared immediately on `attach` entry, leaving only the session's `bash-3.2$` prompt + status bar. Screenshot in `uat-evidence/attach-screen-cleared.png`.

## Open questions

None — three bugs, three fixes, three verifications.

## Surprises

- PAPER-03 turned out NOT to be a real leak. The Phase 110 REVIEW WR-03 finding called it "pre-existing leak, no new exposure" — investigation revealed it's actually a bounded-lifetime goroutine that completes when the OS reaps the killed process. The fix is mostly comment-rewrite + regression test, not behavior change. Original "Give up — process is truly stuck" comment was misleading; replaced with explicit bounded-lifetime contract.
- PAPER-01's defensive guard is technically defense-in-depth — Phase 109 scope addition #3 (`84e1387`) already eliminated the specific empty-agent repro that surfaced the panic. The guard hardens against future zero-dimension regressions (e.g. TUI rendering before WindowSizeMsg arrives, integration test code paths that construct partial Models).
