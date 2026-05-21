---
phase: 121-tui-files-view
plan: 03
subsystem: tui
tags: [tui, files, testing, integration-test, coverage-matrix, merge-gate, static-analysis]
requires: [phase-121/121-01, phase-121/121-02]
provides:
  - TestFiles_NoSyncFSCalls — TUI-07 static-grep guard against synchronous FS calls
  - TestFiles_PathTruncation_StatusLine — TUI-06 explicit "…/…/leaf" assertion
  - TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp — TUI-10 priority cascade (3 sub-tests)
  - TestFiles_Phase121_Requirements — TUI-XX → test-name traceability matrix (10 sub-tests, one per TUI-NN)
  - TestFiles_Integration_LocalSessionEndToEnd — end-to-end test against real DaemonClient + Unix-socket HTTP server
  - setupDaemonWithSession / drainCmd / drainAll — test-only helpers for future integration tests
affects:
  - internal/tui/files_test.go (4 new tests appended; +os/+regexp imports)
  - internal/tui/files_integration_test.go (NEW; `//go:build !windows`)
tech-stack:
  added: []
  patterns:
    - "Static-grep merge gate: TEST file reads PRODUCTION source via os.ReadFile (allowed in tests), strips comments, asserts ZERO regex matches for `os.(ReadDir|Open|OpenFile|Stat)`."
    - "Traceability meta-test: one sub-test per requirement ID, each sub-test logs the covering test set. Adding a new requirement without a covering test is an explicit error."
    - "In-process HTTP integration test via Unix-socket + closure-based files.Sandbox resolver — bypasses daemon's package-private test helpers (newFilesAPI/spyBackend) without compromising the TUI→DaemonClient→/api/files/* transport path."
    - "Bounded cmd-chain drainer (drainAll, 8-iteration cap) — drains head→read in chained tea.Cmd messages while surfacing accidental cmd-loop bugs as test failures."
key-files:
  modified:
    - internal/tui/files_test.go
  created:
    - internal/tui/files_integration_test.go
    - .planning/phases/121-tui-files-view/121-03-SUMMARY.md
decisions:
  - "Static-grep guard checks files.go + files_cmds.go ONLY (not files_test.go or files_integration_test.go) — test code is explicitly allowed to call os.ReadFile to inspect production source. Scope is correct: only Update-path production code freezes the TUI loop."
  - "Coverage matrix is a documentation test, not a re-runner: it asserts the TUI-XX → test-name mapping at a single source of truth. The individual covering tests run via the normal `go test ./internal/tui/...` invocation."
  - "Integration test uses /tmp/tuifN.sock (atomic counter) instead of t.TempDir()-relative paths — t.TempDir() under $TMPDIR on macOS routinely exceeds the 104-byte sun_path limit. Mirrors internal/daemon/socket_test.go::shortSocketPath."
  - "Integration test bypasses internal/daemon's mux and serves internal/files.NewHandler directly. internal/daemon's spyBackend + newFilesAPI helpers are package-private (lowercase types) — the plan explicitly authorises the alternative. The TUI → DaemonClient → /api/files/* transport remains the production path; only session-engine bookkeeping is short-circuited via a closure resolver."
  - "Build tag `//go:build !windows` for the integration test — DaemonClient transport on Windows is a named pipe, not a Unix socket; the parallel Windows test belongs in a separate file with a `windows` build tag. Matches the existing Phase 117 PAPER-01 integration test pattern."
  - "Pane width 80 (not 50 as the plan draft suggested) for the path-truncation assertion: at w=50 the pathBudget collapses to 10 and the leaf 'helper.ts' (9 runes) won't fully fit, breaking the `Contains('helper.ts')` assertion. w=80 gives pathBudget=40, which allows the snap-to-segment branch to produce '…/nested/path/structure/utils/helper.ts'. Documented as deviation Rule 1 below."
metrics:
  completed: 2026-05-21T00:18:23Z
  duration: ~20 min wall-clock
  tasks_completed: 2
  files_changed: 2
  new_tests: 4 top-level (+13 sub-tests) — 1 grep guard, 1 path-truncation, 1 priority cascade (3 sub), 1 coverage matrix (10 sub), 1 integration test
requirements:
  - TUI-01
  - TUI-02
  - TUI-03
  - TUI-04
  - TUI-05
  - TUI-06
  - TUI-07
  - TUI-08
  - TUI-09
  - TUI-10
---

# Phase 121 Plan 03: TUI-XX coverage matrix + TestFiles_NoSyncFSCalls static guard + DaemonClient integration test Summary

Locked in the Phase 121 merge gate: a comprehensive test suite that covers TUI-01..TUI-10 (via a traceability matrix that fails fast if any requirement loses coverage), a static-grep guard against the reintroduction of synchronous filesystem calls inside the TUI Update path, and an end-to-end integration test that drives a real `*daemon.DaemonClient` against an in-process HTTP server bound to a Unix domain socket.

No production-source code was changed by this plan. The static-grep gate passes on first run because Plans 01 and 02 already shipped with zero synchronous FS calls; Plan 03's job is to harden that contract so a future refactor can't silently regress.

## What landed

### Task 1 — Coverage matrix + no-sync-FS-calls static guard (commit `36e2794`)

**File:** `internal/tui/files_test.go` (+204 lines, `+os` and `+regexp` imports)

Four new top-level tests, four new sub-test groups, thirteen total sub-tests:

- **`TestFiles_NoSyncFSCalls`** (TUI-07 merge gate): reads `files.go` and `files_cmds.go` via `os.ReadFile`, strips Go line comments (full-line `//` and trailing inline `//`), and asserts ZERO regex matches for `\bos\.(ReadDir|Open|OpenFile|Stat)\b`. Fails with a precise per-line message identifying the violating file, line number, and source snippet. The test file is explicitly allowed to call `os.ReadFile` — the gate is scoped to the production source files only.
- **`TestFiles_PathTruncation_StatusLine`** (TUI-06): renders `renderFilesStatusLine(80)` against a `cwd` of `very/deep/nested/path/structure/utils/helper.ts`, then `ansi.Strip`s and asserts `…/` is present, `helper.ts` is preserved, and `very/deep` is absent. (See Deviations below for the width-50→80 plan adjustment.)
- **`TestFiles_KeyDispatchPriority_AboveTabCycling_BelowHelp`** (TUI-10 explicit): three sub-tests confirming each Priority 5.5 boundary:
  - `help_beats_files`: with `showHelp=true` + `tabFiles` active, Esc closes help (Priority 5 wins).
  - `files_beats_tabcycling`: with `tabFiles` active and an unrecognised key `x`, the active tab stays at `tabFiles` (handleFilesKey swallows; Priority 5.5 returns before Priority 6 tab-cycling can run).
  - `killconfirm_beats_files`: with `modalKillConfirm` up and `tabFiles` active, `n` cancels the modal (Priority 2 wins).
- **`TestFiles_Phase121_Requirements`** (traceability matrix): one sub-test per `TUI-NN` requirement (TUI-01..TUI-10). Each sub-test logs the set of covering test names; failing fast if any requirement has zero coverage. `go test -run TestFiles_Phase121_Requirements -v` produces exactly 10 `PASS: TestFiles_Phase121_Requirements/TUI-NN` lines — verified.

### Task 2 — DaemonClient end-to-end integration test (commit `5bd8a8f`)

**File:** `internal/tui/files_integration_test.go` (NEW, 378 lines, `//go:build !windows`)

Drives a real `*daemon.DaemonClient` against an in-process HTTP server bound to a Unix domain socket, exercising the complete Files view flow:

1. Press `f` on a local session row → `tabFiles` opens, `m.files.sessionID` wired, `loadDirCmd` dispatched.
2. Drain → `m.files.entries` contains `a.txt`, `b.md`, `sub/`.
3. Cursor to `sub/`, Enter → `loadDirCmd("sub")` dispatched + drained; `cwd == "sub"`, entries contain `nested.txt`.
4. Backspace → `loadDirCmd(parent)` dispatched + drained; `cwd` back at root (`""` or `"."` accepted).
5. `/` activates filter; type `a` → `filteredEntries()` includes `a.txt`, excludes `b.md`.
6. Esc clears filter (value and active state both reset).
7. Cursor to `a.txt`, Enter → `headFileCmd` → drained → `applyFilesHeadMsg` dispatches `readFileCmd` → drained → `previewKind == previewText`, `preview.View()` contains `"alpha"`.

**Test helpers added:**

- `setupDaemonWithSession(t)` — builds a tempdir with three files (`a.txt`, `b.md`, `sub/nested.txt`), wraps it in a `files.Sandbox`, mounts `files.NewHandler` on a `http.ServeMux` at `/api/files/{list,stat,read}` (plus HEAD on `/read`), serves on a `/tmp/tuifN.sock` Unix socket, and returns a `*daemon.DaemonClient` bound to that socket. Returns a cleanup function that gracefully shuts down the server.
- `drainCmd(t, m, cmd)` — synchronously runs `cmd()`, feeds the resulting `tea.Msg` through `m.Update`, returns the updated `Model` and any follow-up cmd.
- `drainAll(t, m, cmd)` — repeatedly drains until the cmd chain terminates, capped at 8 iterations. The cap surfaces accidental cmd-loop bugs as test failures rather than CI timeouts.

## Verification

```
$ go build ./internal/tui/                                            # exit 0
$ go vet ./internal/tui/                                              # no diagnostics
$ go test -run '^TestFiles' ./internal/tui/ -count=1 -race -timeout 60s
ok  github.com/scottkw/agenthub/internal/tui  1.119s
$ go test -run '^TestFiles_Phase121_Requirements$' ./internal/tui/ -v -count=1 \
    | grep -c 'PASS: TestFiles_Phase121_Requirements/TUI-'
10
$ grep -E 'os\.(ReadDir|Open|OpenFile|Stat)\b' \
    internal/tui/files.go internal/tui/files_cmds.go | wc -l
0
$ go test ./... -count=1 -short -timeout 120s                         # all packages OK
```

Whole-plan grep gates:

| Check                                                                   | Expected | Actual |
| ----------------------------------------------------------------------- | -------- | ------ |
| Synchronous FS in `files.go` / `files_cmds.go` (grep gate)              | 0        | 0      |
| `TestFiles_NoSyncFSCalls` body lines                                    | ≥ 20     | 30     |
| `TestFiles_Phase121_Requirements` sub-test PASS lines                   | 10       | 10     |
| `TestFiles_Integration_LocalSessionEndToEnd` exists + passes            | yes      | yes    |
| `files_test.go` total tests added by Plan 03                            | 4        | 4      |
| `files_integration_test.go` lines                                       | ≥ 120    | 378    |
| `STATE.md` / `ROADMAP.md` modified (worktree-mode invariant)            | 0        | 0      |

Combined `--- PASS` count for `go test -run '^TestFiles' -v` across the package: **all green, race-clean**. No regressions in `go test ./... -short`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] `TestFiles_PathTruncation_StatusLine` pane width 50 → 80**

- **Found during:** Task 1 GREEN, before commit. Worked through the truncation arithmetic by hand.
- **Issue:** The plan's draft test used pane width 50. With `pathBudget = max(10, w-40)`, that resolves to `pathBudget = 10`. `truncateLeft` with `maxWidth=10` keeps the last 8 runes (after reserving 2 for `…/`). The last 8 chars of `./very/deep/nested/path/structure/utils/helper.ts` are `elper.ts` — which means the test assertion `strings.Contains(rendered, "helper.ts")` would FAIL, because only `elper.ts` survives the truncation. The plan's behavioural contract (preserve the leaf segment) requires `keep ≥ len("helper.ts") = 9`, so `pathBudget ≥ 11`, so `w ≥ 51`.
- **Fix:** Bumped pane width from 50 to 80 in the test. With `w=80`, `pathBudget=40`, `keep=38`, and the snap-to-segment branch in `truncateLeft` produces `…/nested/path/structure/utils/helper.ts` — which satisfies all three assertions (`…/` prefix, `helper.ts` leaf preserved, `very/deep` truncated).
- **Files modified:** `internal/tui/files_test.go` (the new test was authored with w=80 from the outset; the plan's draft never landed).
- **Commit:** `36e2794`.
- **No production-source change.** This is purely a test-arithmetic correction; the behaviour under test (truncateLeft snap-to-segment) is correct as implemented in Plan 01.

**2. [Rule 1 — Bug] Integration test socket path exceeded macOS 104-byte sun_path limit**

- **Found during:** First execution of `TestFiles_Integration_LocalSessionEndToEnd`. `net.Listen("unix", ...)` failed with `bind: invalid argument` — the classic sun_path overflow signature on macOS.
- **Issue:** The initial implementation used `filepath.Join(t.TempDir(), "d.sock")`. On macOS, `t.TempDir()` resolves under `$TMPDIR` (e.g. `/var/folders/lb/2__vrh3n2n155kwhrvz08lv80000gn/T/TestFiles_..._2321154356/002/`) — ~116 bytes before the socket filename even gets added. The Linux limit is 108 bytes; macOS is 104. Both exceed.
- **Fix:** Switched to `/tmp/tuif<N>.sock` with an `atomic.Uint64` counter (`socketCounter`) for uniqueness across parallel test runs. Added a `t.Cleanup(func() { _ = os.Remove(socketPath) })` so the socket file is reclaimed at test exit. Mirrors `internal/daemon/socket_test.go::shortSocketPath` exactly, which is the canonical pattern in this codebase.
- **Files modified:** `internal/tui/files_integration_test.go` (during initial authoring; the working version was the one committed).
- **Commit:** `5bd8a8f`.
- **No production-source change.** This is purely a test-environment portability fix.

**3. [Rule 1 — Bug] Integration test typed `'a'` but textinput consumed `Text`, not `Code`**

- **Found during:** Second execution of `TestFiles_Integration_LocalSessionEndToEnd`. The filter test step failed with `expected filterInput value='a', got ""`.
- **Issue:** `tea.KeyPressMsg{Code: 'a'}` carries `Code = 'a'` (rune) but `Text = ""`. The `charm.land/bubbles/v2/textinput.Update` default branch inserts `[]rune(msg.Text)` — not `msg.Code` — when no key binding matches. With `msg.Text == ""`, no rune is inserted and the value stays empty. This is correct Bubble Tea v2 semantics (`Text` is the human-typed text; `Code` is the canonical key identity, used for matching keybindings). The existing filter tests in `files_test.go` use `m.files.filterInput.SetValue("abc")` directly, which is why this only manifested in the integration test.
- **Fix:** Updated the synthetic key event to `tea.KeyPressMsg{Code: 'a', Text: "a"}`. Documented inline so a future contributor understands why both fields are set.
- **Files modified:** `internal/tui/files_integration_test.go`.
- **Commit:** `5bd8a8f`.
- **No production-source change.** This is a test-authoring correction — the production `handleFilesKey` filter cascade forwards `msg` unchanged to `m.files.filterInput.Update(msg)`, which is the correct behaviour.

### Plan compliance notes (NOT deviations)

- **Integration test transport choice:** The plan offered two options — (a) spin up a real `daemon.NewAPI` and use `daemon.NewDaemonClient`, or (b) serve `internal/files.NewHandler` directly via `httptest.NewServer` against a tempdir Sandbox. I chose a hybrid: option (b)'s closure-resolver pattern, served on a Unix domain socket via `net.Listen("unix", ...)` so the production `*daemon.DaemonClient` (which requires a socket path) can be used unchanged. This keeps the TUI → DaemonClient → HTTP transport path identical to production while bypassing the daemon's package-private session-engine bookkeeping. The plan explicitly authorises this trade-off.
- **`drainAll` iteration cap (8):** Not a deviation — the plan suggested "a small retry loop" with no cap. I added a hard cap because an uncapped drain loop in a future refactor that accidentally creates a cmd-loop would deadlock CI; surfacing it as `t.Fatalf` is strictly better.
- **TUI-09 / TUI-02 covered by existing tests only:** The traceability matrix lists Plan-01 / Plan-02 tests for these requirements without adding new ones. They already meet the "≥ 1 covering test" bar and the plan does not require Plan 03 to author additional coverage for them.

No architectural changes, no checkpoint hits, no package-install failures.

## Threat Model Coverage

| Threat ID | Disposition | Plan 03 status                                                                                                                                                                                                                              |
| --------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| T-121-11 (regression risk: synchronous FS reintroduced) | mitigate | Done — `TestFiles_NoSyncFSCalls` is the static guard. Any future PR that adds `os.ReadDir/Open/OpenFile/Stat` to `files.go` or `files_cmds.go` fails CI with a precise file:line:source error message.                                            |
| T-121-12 (verification gap: TUI-XX requirement ships unverified)         | mitigate | Done — `TestFiles_Phase121_Requirements` enumerates all 10 requirements as named sub-tests; each lists its covering test set. A future TUI-NN requirement added without a covering test entry fails the meta-test. Combined with the `-v` `PASS: ...TUI-NN` line-count gate in CI, this is a defence-in-depth check. |

Previous-plan threats (T-121-01..T-121-10) remain mitigated by the Plan 01 + Plan 02 implementations; the Plan 03 integration test exercises T-121-04 (stale-msg discard against a real session ID) and T-121-09 (preview reset on cwd change) along the happy path.

## Known Stubs

None. Plan 03 introduces only tests and test helpers; there are no production stubs to track.

## Self-Check: PASSED

Verified file existence (relative to worktree root `/Users/ken/dev/agenthub/.claude/worktrees/agent-a1bcc73b29dc5f9d1/`):

- `internal/tui/files_test.go` — FOUND (modified; +204 lines, 4 new top-level tests)
- `internal/tui/files_integration_test.go` — FOUND (NEW; 378 lines)
- `.planning/phases/121-tui-files-view/121-03-SUMMARY.md` — FOUND (this file)

Verified commits in branch history (`git log --oneline f58c512..HEAD`):

- `5bd8a8f` — test(121-03): add end-to-end DaemonClient integration test for Files view — FOUND
- `36e2794` — test(121-03): add TUI-XX coverage matrix + no-sync-FS-calls static guard — FOUND

Verified success criteria from execution prompt:

- [x] Tasks committed atomically with `test(121-03:)` prefixes (2 commits, both test-only)
- [x] SUMMARY.md at `.planning/phases/121-tui-files-view/121-03-SUMMARY.md`
- [x] `TestFiles_NoSyncFSCalls` static grep guard verifies production source (`files.go` + `files_cmds.go`) contains zero `os.ReadDir`/`os.Open`/`os.Stat`/`os.OpenFile` calls
- [x] TUI-XX coverage matrix test: each TUI-01..TUI-10 requirement maps to a verifiable test case (subtest name format `TUI-NN`) — verified 10/10 PASS lines
- [x] Integration test in `files_integration_test.go` (NEW, Unix-socket gated `//go:build !windows`) that:
  - [x] spins up a real daemon (in-process HTTP server serving `internal/files.NewHandler` at the daemon's exact routes via a Unix-socket listener)
  - [x] uses a real `*daemon.DaemonClient`
  - [x] exercises the loadDirCmd / readFileCmd / headFileCmd loop end-to-end
  - [x] confirms a text file preview round-trip works (`previewKind == previewText`, `preview.View()` contains the file body)
- [x] `go test ./internal/tui/... -race` passes (1.286s, all green)
- [x] No modifications to `STATE.md` / `ROADMAP.md` (worktree mode)
