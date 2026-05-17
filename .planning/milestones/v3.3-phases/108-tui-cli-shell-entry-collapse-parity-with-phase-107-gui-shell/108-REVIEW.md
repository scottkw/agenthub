---
phase: 108
status: addressed
depth: standard
critical_count: 0
warning_count: 2
warning_fixed: 2
info_count: 4
info_fixed: 0
files_reviewed: 7
files_reviewed_list:
  - internal/tui/modal.go
  - internal/tui/model.go
  - internal/tui/tui.go
  - internal/tui/update_test.go
  - cmd_cli.go
  - cmd_cli_test.go
  - README.md
---

# Phase 108: Code Review Report

**Reviewed:** 2026-05-16
**Depth:** standard
**Files Reviewed:** 7
**Status:** findings (2 warnings, 4 info — no blockers)

## Summary

Phase 108 collapses the TUI agent picker and CLI `agenthub new shell` to a
single "Shell" entry per surface, matching Phase 107's GUI shape. The eight
SPEC-locked requirements (PARITY-TUI-01..04, PARITY-CLI-01..03,
PARITY-DOCS-01) are satisfied. All grep gates in `108-SPEC.md` pass:

| Gate | Result |
|------|--------|
| `grep -nE 'Shell —\|sortShellsForPicker' internal/tui/modal.go` | 0 matches |
| `grep -nE 'detectedShells\|DiscoverShells' internal/tui/{model,modal,tui,update}.go` | 0 matches |
| `grep -nE '"shell".*flag\|shellFlag\|allowed.*shell\|unknown shell' cmd_cli.go` | 0 matches |
| `grep -nE '\-\-shell=' README.md` | 0 matches |
| `gofmt -l` on all 7 files | empty |
| `go vet ./internal/tui/... .` | clean |
| `go test -run 'AgentPicker\|TestModal_AgentCycle\|TestCmdNewShell\|TestUsage_IncludesNewShell' ./internal/tui/... .` | PASS |

Import cleanup verified:
- `internal/tui/modal.go` — `"sort"` and `internal/pty` imports gone, file
  no longer needs filesystem helpers.
- `cmd_cli.go` — `"errors"` import gone; was only used for the deleted
  `errors.New` call sites.

Cross-surface parity confirmed in the daemon resolution path
(`internal/daemon/engine.go:500-530`, branch (0)): all three surfaces pass
`cli = "shell"` and the daemon dispatches via `e.shellPath` → spec argv.
The CLI path (`cmdNewShell`) hard-codes `const cli = "shell"` so there is
no caller-controlled override left in the user-space layer.

Security review: the `--shell` flag deletion is a hard *reduction* of
attack surface — there is no path-traversal or arbitrary-binary regression.
The CLI never touches `e.shellPath` directly; spawn-time resolution stays
on the daemon side, behind `SetShellPath`'s executable-existence guard
(Phase 107 SHELL-11). No new shell-out, exec, or env-var consumption.

The two warnings below are about code that is now technically dead /
mis-commented after Phase 108's collapse — neither breaks correctness, but
both will mislead future readers and should be cleaned up in a follow-up.

## Warnings

### WR-01: `submitNewSession` still branches on per-shell variants that the new picker can never emit

**File:** `internal/tui/update.go:670-687` (also `update.go:692-698`,
`isShellCLI`)
**Issue:** After Phase 108-01, `agentEntries()` only ever emits
`cliKey = "shell"` for the shell row (plus AI CLI keys for the
`detectedCLIs` entries). `submitNewSession` reads `entries[idx].cliKey`
into `cli` and then checks `if !isShellCLI(cli) { … }`. `isShellCLI` still
returns true for `"shell" | "bash" | "zsh" | "pwsh" | "powershell"` — but
the picker has no way to produce the latter four. The `bash/zsh/pwsh/
powershell` branches in `isShellCLI` and the matching doc comment at
`update.go:644-645` (`"For shell entries (cli ∈ {shell, bash, zsh, pwsh,
powershell}) …"`) are now dead code paths that contradict the new
contract.

Why this matters: a future reader sees the multi-shell allowlist and
infers the picker might still emit those keys, undoing the conceptual
"single Shell entry" contract that Phase 108 just locked. The doc comment
at `update.go:643` ("Phase 101 SHELL-03: agent selection reads from the
unified agentEntries slice (AI CLIs + shells)") is also stale — it
predates the collapse.

Not a runtime bug: `cli == "shell"` still matches `isShellCLI`, so the
"drop args for shell sessions" branch fires correctly. This is correctness-
neutral but contract-divergent.

**Fix:** In a follow-up touching `internal/tui/update.go`, either:
1. Narrow `isShellCLI` to `case "shell": return true` (after confirming
   no other caller emits the legacy keys), or
2. Keep the broad allowlist as a defense-in-depth and add a comment
   stating that the TUI picker only emits `"shell"` post-Phase-108 — the
   other keys exist solely for back-compat with stored sessions.
   Update the `submitNewSession` doc comment to point at
   PARITY-TUI-01/02 instead of Phase 101 SHELL-03.

### WR-02: `m.agentIdx` doc comment is stale (claims "covers AI CLIs + shells")

**File:** `internal/tui/model.go:124`
**Issue:**
```go
agentIdx     int               // current agent picker index (covers AI CLIs + shells)
```
"shells" (plural) is the Phase 101 contract. Post-collapse, the index
covers AI CLIs + *one* static Shell entry. The plural-shells phrasing
will lead a future reader to expect multiple shell rows again.

**Fix:**
```go
agentIdx     int               // current agent picker index (AI CLIs + the single static Shell entry)
```

## Info

### IN-01: `Phase 108 PARITY-CLI-01` rationale block in `cmdNewShell` doc comment matches `--shell=` grep

**File:** `cmd_cli.go:85, 88`
**Issue:** `grep -nE '\-\-shell=' cmd_cli.go` returns 2 hits inside the
`cmdNewShell` doc comment (rationale block: "The `--shell=…` flag *was*
removed…"). These are deliberately preserved cross-phase context per
108-03 SUMMARY's scope-boundary note. The SPEC's grep gate language
("0 matches in user-facing strings") excludes Go doc comments. No action
required — flagging because the grep noise can mislead a future verifier
who copy-pastes the SPEC gate without reading the exception clause.

**Fix:** Optional — if the historical rationale is no longer needed once
Phase 108 is in git history, the two `--shell=` references in the doc
comment can be rephrased without the literal flag syntax (e.g. "the
removed shell-binary picker flag"). Not required.

### IN-02: PARITY-CLI-03 is locked by SPEC but not by a runnable test (documented skip)

**File:** `cmd_cli_test.go:789-813`
**Issue:** `TestCmdNewShell_InvalidShellPathSilentFallback` is `t.Skip`'d
because both `client.SetShellPath` and the unexported `engine.shellPath`
field reject / disallow installing a deliberately-broken path. The skip
message is clear, the follow-up (`SetShellPathForTest` export in
`internal/daemon`) is documented in `deferred-items.md` with a working
code sketch, and SPEC PARITY-CLI-03 acceptance is asserted at the
behavioral level by the existing `assertShellCLI` fallback path in
`TestCmdNewShell_NoArgs_UsesSystemDefault`. Treat this as a known
acceptance-test gap, not a contract gap. No action required for Phase 108
sign-off.

**Fix:** Open a v3.4 backlog item to add `SetShellPathForTest` and
unblock the skipped test (sketch already in deferred-items.md).

### IN-03: TUI test asserts `" Shell "` substring instead of literal `"< Shell >"`

**File:** `internal/tui/update_test.go:1129, 1186-1191, 1237`
**Issue:** Plan recommended `"< Shell >"` as the rendered-substring check,
but lipgloss wraps the `<` and `>` arrows in ANSI styling
(`\x1b[…m<\x1b[m Shell \x1b[…m>\x1b[m`), so the literal bracketed string
never appears in the rendered output. The tests check the uncolored
`" Shell "` substring between the arrows and separately forbid `"Shell —"`,
`"system default"`, `"/"`, `"\\"`. The semantic contract (label =
`Shell`, no em-dash, no path) is fully preserved. Documented as deviation
in 108-01 SUMMARY. No action required.

**Fix:** None — flagged for traceability only. Future ANSI-stripping
helper (e.g. `lipgloss.StripANSI(rendered)` or `regexp.MustCompile("\x1b\\[[0-9;]*m")`)
would allow asserting `"< Shell >"` literally if a stricter check is
ever wanted.

### IN-04: TUI submitNewSession test for an explicit "Shell" cycle landing missing

**File:** `internal/tui/update_test.go` (suite-level observation)
**Issue:** The picker tests cover `agentEntries()` shape, `renderAgentPicker`
output at the Shell idx, and `cycleAgent` wrap. There is no end-to-end
test that cycles the picker to the Shell entry and then verifies
`submitNewSession` dispatches with `cli = "shell"`. The existing
`TestModal_SubmitSuccess` (`update_test.go:779+`) hard-sets `agentIdx = 0`
and uses an AI CLI. The contract that "the static Shell row, when
selected, becomes a `cli="shell"` CreateSession call" is locked at the
CLI layer (`TestCmdNewShell_SettingsShellPathSpawned`) and at the picker-
shape layer, but not at the TUI submit layer.

**Fix:** Optional — add `TestModal_SubmitShell` mirroring
`TestModal_SubmitSuccess` but with `agentIdx = len(detectedCLIs)` (the
Shell entry) and assert the issued `createSession` cmd carries
`cli="shell"`. Low-priority because the daemon end-to-end test in
`cmd_cli_test.go` already covers the most important half of the
contract.

---

## Verdict

**APPROVE WITH MINOR FOLLOW-UPS.** All 8 SPEC requirements satisfied,
all grep gates green, all in-scope tests pass, no security regressions,
no blockers. The two warnings (stale Phase 101 comments / dead
`isShellCLI` branches in `internal/tui/update.go`) are correctness-
neutral cleanups that will keep future readers honest; they do not block
Phase 108 sign-off and do not require an in-phase commit. Both can be
folded into a v3.4 cleanup batch alongside the deferred
`SetShellPathForTest` export (IN-02) and the optional submit-layer test
(IN-04).

_Reviewed: 2026-05-16T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
