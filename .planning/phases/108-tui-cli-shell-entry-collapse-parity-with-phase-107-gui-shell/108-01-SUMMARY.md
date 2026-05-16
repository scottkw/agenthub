---
phase: 108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell
plan: "108-01"
status: complete
deviation_notes: |
  - Plan called for one commit per task with TDD red→green cycles. In
    practice, removing m.detectedShells and pty.DiscoverShells() (Task 2)
    makes the existing tests fail to compile, and rewriting the tests
    (Task 3) before the source changes would leave the package in a
    non-green state mid-plan. Tasks 1–3 were therefore bundled into a
    single atomic commit (refactor(108-01): cb0fa04) so the package never
    transits a broken state on main. The plan's own note
    ("one commit per task, or one commit for the whole plan if cleaner")
    permits this. Task 4 is verification-only and added no new commit.
  - TestModal_AgentCycle (update_test.go:699) was not enumerated in
    Task 3 but assumes a 2-entry picker. Under the new contract the
    picker is 3 entries (claude → opencode → Shell → wrap) so the test
    was updated alongside the three named picker tests (Rule 1 auto-fix:
    pre-existing test contradicted by new contract).
  - Plan recommended "< Shell >" as the literal substring check on the
    rendered picker. lipgloss wraps the `<` and `>` arrows with ANSI
    foreground escapes, so the rendered string is
    `\x1b[…m<\x1b[m Shell \x1b[…m>\x1b[m`. Assertions match the
    uncolored " Shell " substring between the arrows; the "no em-dash /
    no path / no system default" forbidden-substring checks remain.
requirements_satisfied:
  spec: [PARITY-TUI-01, PARITY-TUI-02, PARITY-TUI-03, PARITY-TUI-04]
  roadmap: [PARITY-01, "PARITY-03 (TUI half)"]
files_modified:
  - internal/tui/modal.go
  - internal/tui/model.go
  - internal/tui/tui.go
  - internal/tui/update_test.go
commits:
  - cb0fa04 refactor(108-01): collapse TUI agent picker to single static "Shell" entry
---

# Phase 108 Plan 01: TUI agent-picker collapse Summary

One-liner: Collapsed the TUI new-session agent picker to one static
`Shell` row (cliKey=`"shell"`, displayLabel=`"Shell"`), removed the
`m.detectedShells` field and the `pty.DiscoverShells()` startup call, and
relocked four picker tests against the new contract — bringing the TUI
into parity with Phase 107's GUI SHELL-10 collapse.

## Task-by-task changes

### Task 1 — Collapse `agentEntries()` and delete `sortShellsForPicker` (PARITY-TUI-01, PARITY-TUI-02)

File: `internal/tui/modal.go`

- Deleted `sortShellsForPicker()` (formerly `modal.go:25-52`) entirely,
  including its doc-comment block. No remaining caller in the package.
- Rewrote `agentEntries()` (`modal.go:34-49`):
  - Returns `nil` when `len(m.detectedCLIs) == 0` (the static Shell row
    is appended only when there is at least one AI CLI to pair against).
  - Otherwise returns `len(m.detectedCLIs) + 1` entries: one
    `{cliKey: c.Name, displayLabel: c.DisplayName}` per AI CLI, then a
    single trailing `{cliKey: "shell", displayLabel: "Shell"}`.
- Updated the `agentEntry` struct doc comment (`modal.go:13-21`) so the
  example labels are `"Claude Code"` and `"Shell"` (dropped the
  `"Shell — system default"` / `"Shell — bash"` example).
- Updated the `renderAgentPicker` and `cycleAgent` doc comments
  (`modal.go:113-117`, `modal.go:131-134`) to point at Phase 108
  PARITY-TUI-01/02 and describe the single-Shell contract.
- Removed `"sort"` and `"github.com/scottkw/agenthub/internal/pty"`
  from the import block — both became unused with the helper deletion.

### Task 2 — Drop `m.detectedShells` and the `pty.DiscoverShells()` call (PARITY-TUI-04)

Files: `internal/tui/model.go`, `internal/tui/tui.go`

- `internal/tui/model.go`: deleted the `detectedShells []pty.DetectedShell`
  field and its Phase 101 SHELL-03 doc-comment block (formerly
  `model.go:130-135`). Left a brief Phase 108 PARITY-TUI-04 marker
  comment in its place noting that the TUI no longer probes the host
  filesystem at startup. The `pty` import stays — `DetectedCLI` still
  comes from `pty`.
- `internal/tui/tui.go`: deleted the
  `detectedShells: pty.DiscoverShells(),` line inside `newModel()`'s
  return literal. Rewrote the Phase 101 SHELL-03 doc comment above
  `newModel` so it now describes Phase 108 PARITY-TUI-04 (static Shell
  row; daemon resolves binary via `shellPath`). Kept the
  `pty.DetectCLIs()` call for `detectedCLIs`.

`pty.DiscoverShells()` itself is untouched — other callers (notably
`internal/daemon/engine.go`) still use it.

### Task 3 — Relock the three picker tests + one cascading test (PARITY-TUI-03)

File: `internal/tui/update_test.go`

- `testModel()` (`update_test.go:14-22`): removed the
  `m.detectedShells = nil` line and the preceding Phase 101 SHELL-03
  doc comment block. The helper is now a flat allocation of a
  `newModel()` with TUI dimensions set.
- `TestAgentPicker_IncludesShellEntries` (`update_test.go:1090-1148`):
  rewritten body-only (name unchanged). Now asserts:
  - `len(agentEntries()) == 3` with two AI CLIs.
  - `entries[2].cliKey == "shell"` and `entries[2].displayLabel == "Shell"`.
  - Rendered picker at `agentIdx=2` contains `" Shell "` and does NOT
    contain `"Shell —"`, `"system default"`, `"/"`, or `"\\"`.
  - Cycle right from idx=2 wraps to idx=0 (claude); cycle left from
    idx=0 wraps to idx=2 (Shell).
- `TestAgentPicker_CycleOrder` (`update_test.go:1150-1191`): rewritten
  body-only (name unchanged). `wantOrder` is now
  `[Claude Code, OpenCode, Shell, Claude Code (wraps)]`. The Shell-step
  assertion additionally checks the rendered string does NOT contain
  `"Shell —"`.
- `TestAgentPicker_OnlyAICLIs_NoShells` → replaced with
  `TestAgentPicker_StaticShellEntryAlwaysAppended`
  (`update_test.go:1193-1242`) as recommended in the plan (option b).
  The original premise ("no shells → no Shell entries") is no longer
  the contract. The replacement asserts: 1 AI CLI plus the static Shell
  yields `len=2`; trailing entry is `cliKey="shell"`/`displayLabel="Shell"`;
  no `"Shell —"` anywhere; cycle right toggles claude → Shell → claude.
- `TestModal_AgentCycle` (`update_test.go:699-741`) cascading update:
  this test was not in the plan's task list but assumed the
  pre-collapse 2-entry picker. Updated the four cycle assertions to
  match the new 3-entry cycle (claude(0) → opencode(1) → Shell(2) →
  wrap). Tracked under Rule 1 auto-fix (pre-existing test contradicted
  by new contract).

### Task 4 — Verification

Verification block in the plan ran clean:

```
go build ./internal/tui/...                                 exit 0
go vet ./internal/tui/...                                   exit 0
gofmt -l internal/tui/    (excluding pre-existing keys.go)  empty
grep -nE "Shell —|sortShellsForPicker" internal/tui/modal.go
                                                            0 matches
grep -nE "detectedShells|DiscoverShells" \
     internal/tui/{model,modal,tui,update}.go               0 matches
go test ./internal/tui/... -count=1                         PASS
go test ./internal/tui/... -run 'AgentPicker' -v -count=1   PASS (3/3)
```

`gofmt -l internal/tui/` independently flags `internal/tui/keys.go`,
which I did not touch this plan — pre-existing condition logged here
for the verifier and out-of-scope per Rule SCOPE BOUNDARY.

## Verification results (raw)

### AgentPicker tests

```
=== RUN   TestAgentPicker_IncludesShellEntries
--- PASS: TestAgentPicker_IncludesShellEntries (0.00s)
=== RUN   TestAgentPicker_CycleOrder
--- PASS: TestAgentPicker_CycleOrder (0.00s)
=== RUN   TestAgentPicker_StaticShellEntryAlwaysAppended
--- PASS: TestAgentPicker_StaticShellEntryAlwaysAppended (0.00s)
PASS
ok  	github.com/scottkw/agenthub/internal/tui	0.011s
```

### Full TUI suite

```
ok  	github.com/scottkw/agenthub/internal/tui	0.024s
```

### Grep gates (SPEC acceptance)

| Gate                                                                                            | Expected | Actual |
| ----------------------------------------------------------------------------------------------- | -------- | ------ |
| `grep -nE "Shell —\|sortShellsForPicker" internal/tui/modal.go`                                 | 0        | 0      |
| `grep -nE "detectedShells\|DiscoverShells" internal/tui/{model,modal,tui,update}.go`            | 0        | 0      |
| `grep -nE "Shell — bash\|Shell — zsh\|Shell — system default\|Shell — PowerShell" .../update_test.go` | 0    | 0      |

## Deviations from PLAN.md

Captured in the `deviation_notes` frontmatter field above. Summary:

1. **Single commit instead of per-task commits** — the package transits
   a non-compiling state if the source changes (Tasks 1–2) land before
   the test rewrite (Task 3). One atomic commit keeps `main` green at
   every point. Plan explicitly permits this option.
2. **TestModal_AgentCycle updated alongside the three named tests** —
   pre-existing test that assumed the 2-entry picker. Rule 1 auto-fix.
3. **`" Shell "` substring instead of `"< Shell >"` substring** in
   render assertions — lipgloss ANSI styling wraps the cycle arrows,
   so the literal `< Shell >` substring is not in the rendered output.
   The semantic check (label is `Shell` between arrows, no em-dash, no
   path) is fully preserved.

No deviations affect the SPEC's locked requirements or acceptance
criteria. No deferred items.

## Hand-off note for Plan 108-02

Plan 108-02 (`cmd_cli.go` `--shell` flag removal) is **already merged on
main** at commits `829144a` and `b9f9199` — these landed before this
plan's commit `cb0fa04`. The CLI's `cmdNewShell` now passes `cli="shell"`
through `daemon.CreateSession`, the TUI's `submitNewSession` will do the
same after this commit, and the daemon resolution path at
`internal/daemon/engine.go:500-530` (unchanged from Phase 107) routes
both through `shellPath`. The three surfaces (GUI, TUI, CLI) are now in
parity for shell-session creation.

Plan 108-03 (README + help-text doc sweep, PARITY-DOCS-01) remains
outstanding.

## Self-Check: PASSED

Files verified:
- `internal/tui/modal.go` — sortShellsForPicker absent, agentEntries
  returns the new shape (verified by passing tests)
- `internal/tui/model.go` — detectedShells field absent (verified by
  grep + clean build)
- `internal/tui/tui.go` — pty.DiscoverShells() call absent (verified
  by grep + clean build)
- `internal/tui/update_test.go` — all four affected tests pass
  (verified by `go test -v -run AgentPicker` + full suite green)

Commit verified:
- `cb0fa04` present on main: `git log --oneline -3` shows it at HEAD.
