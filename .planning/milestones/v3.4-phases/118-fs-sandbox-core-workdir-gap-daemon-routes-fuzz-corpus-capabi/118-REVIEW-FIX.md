---
phase: 118
fixed_at: 2026-05-20T18:51:00Z
review_path: .planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 118: Code Review Fix Report

**Fixed at:** 2026-05-20T18:51:00Z
**Source review:** 118-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (Warnings only; Info findings advisory and skipped per workflow scope)
- Fixed: 4
- Skipped: 0

All four Warning findings from Phase 118 were fixed and committed
atomically. `go test ./internal/files/... ./internal/daemon/...` passes
cleanly after each fix and at the end of the run. The fuzz target was
re-run for 5 s post-WR-04 (154 k executions, no crashes, baseline
coverage gathered).

## Fixed Issues

### WR-01: `Truncated` flag false-positives at exactly 10,000 entries

**Files modified:** `internal/files/handler.go`
**Commit:** 25f354c
**Applied fix:** Probe one entry past `maxListEntries` so the handler
can distinguish "exactly cap" from "more than cap." `ReadDir` now
requests `maxListEntries + 1`; `truncated` is true only when the
returned slice exceeds the cap, in which case the slice is trimmed to
`maxListEntries` before serialization. Existing `TestHandler_List_-
TruncatedAt10000` (which seeds 10,001 files) still asserts truncated
== true; the fix only changes behaviour at exactly 10,000 entries
(now correctly reports truncated == false).

### WR-02: `validateRelativePath` drive-letter check produces wrong error label for ADS-shaped paths

**Files modified:** `internal/files/sandbox.go`
**Commit:** 83590f0
**Applied fix:** Restricted the drive-letter prefix check to ASCII
A-Z/a-z by adding an `isASCIILetter` helper. Inputs like `a:foo` now
fall through to the colon-anywhere ban and get the accurate
`"colon (ADS) rejected"` message; real drive-letter inputs like
`C:\windows\system32` still hit the drive-letter branch and produce
`"drive letter rejected"`. Chose the ASCII-only check over importing
`unicode` because Windows drive letters are strictly ASCII — a Unicode-
aware test would be both over-broad and over-heavy for one byte's
work.

**Verification note:** Existing tests only check that rejection
happens, not the message text. The error-label routing was confirmed
by source inspection; recommend a follow-up test that asserts the
exact `Error()` string for both `C:\foo` (expect "drive letter
rejected") and `a:foo` (expect "colon (ADS) rejected") to lock in
the contract.

### WR-03: Symlink-escape test asserts via wrong path shape — coverage is shallower than it appears

**Files modified:** `internal/files/sandbox_test.go`
**Commit:** 09d6546
**Applied fix:** Added a positive control that reads `outside/secret`
directly via `os.ReadFile` before the symlink Open. The negative
assertion now can only pass if `os.Root` actively rejected the escape
— a broken implementation that returned ENOENT via the wrong route
would fail the positive control before the negative assertion ran.
Also closed the file handle on the unexpected-success branch
(`fp.Close()` before `t.Errorf`) so the test never leaves an open fd
on a tempdir file the cleanup is trying to remove.

### WR-04: Fuzz function contains tautological error check that triggers no real assertion

**Files modified:** `internal/files/sandbox_test.go`
**Commit:** 138835b
**Applied fix:** Removed the `if !errors.Is(err, err)` tautology and
its associated dead branch. The `err` variable is naturally consumed
by the `if err != nil` predicate, so no linter appeasement is needed.
Also removed the now-unused `errors` import. Verified with
`go test -fuzz FuzzSandboxPath -fuzztime 5s` (154 k execs, no
crashes, baseline coverage gathered cleanly).

## Skipped Issues

None.

## Info Findings (Not Applied)

Per workflow scope (`fix_scope: critical_warning`), Info findings IN-01
through IN-07 were not applied. They remain in 118-REVIEW.md as
advisory items for future maintenance:

- IN-01: `HasPerm` whitespace handling
- IN-02: `NewHandler` nil-resolver constructor check
- IN-03: `Stat` MIME-sniff Seek-back I/O
- IN-04: 0-byte `Read` Content-Length header
- IN-05: `extensionMIME` `.bat`/`.cmd` documentation
- IN-06: `daemonSettings.FilesRead` first-run nil
- IN-07: `requireFilesRead` unreachable claims-missing branch

## Tests

- `go test ./internal/files/...` — PASS after each fix and at end
- `go test ./internal/daemon/...` — PASS after each fix and at end
- `go test -fuzz FuzzSandboxPath -fuzztime 5s` — PASS (154 k execs,
  no crashes)
- No existing tests broke or had to be modified.

---

_Fixed: 2026-05-20T18:51:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
