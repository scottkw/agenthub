---
phase: 108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell
plan: "108-03"
subsystem: docs
tags: [parity, docs, cli-help, readme, settings-driven]
requires:
  - Phase 108-01 (TUI single-Shell collapse)
  - Phase 108-02 (CLI --shell flag hard-removal)
provides:
  - PARITY-DOCS-01 (user-facing docs reflect single-Shell shape on GUI/TUI/CLI)
  - PARITY-04 (ROADMAP — docs sweep)
affects:
  - cmd_cli.go
  - README.md
tech-stack:
  added: []
  patterns:
    - "Single source of truth for shell-binary selection — Settings → Paths drives all three surfaces"
key-files:
  modified:
    - cmd_cli.go (usage() help block — removed --shell=bash|zsh|pwsh|powershell modifier line)
    - README.md (added new ### Shell sessions Features subsection between CLI and Web Serving)
decisions:
  - "Placed Shell sessions section between Features → CLI and Features → Web Serving — adjacent to surfaces that depend on the same Settings → Paths value (GUI/TUI/CLI documented inline within one bullet list)."
  - "Left the cmdNewShell doc-comment rationale block (cmd_cli.go:85, 88) untouched per plan constraint — Plan 108-03 must NOT modify cmdNewShell function body. SPEC grep gate exception clause covers this: 'user-facing strings' excludes source-level Go doc comments."
metrics:
  duration: ~5min
  completed: 2026-05-16
---

# Phase 108 Plan 03: PARITY-DOCS-01 docs sweep Summary

## One-liner

Dropped the `--shell=bash|zsh|pwsh|powershell` modifier line from `cmd_cli.go` usage() help block and added a new `### Shell sessions` README section documenting single-Shell parity on GUI / TUI / CLI with Settings → Paths as the single source of truth — closing PARITY-DOCS-01 and completing Phase 108.

## Goal

Align all user-facing documentation with the post-108-01/108-02 surface shape: `agenthub --help` lists `new shell [<path>]` with no flag modifier; README documents single-Shell behavior on all three surfaces and points users at Settings → Paths for binary selection.

## Task-by-task results

### Task 1 — Remove `--shell` modifier line from usage() help block

**File:** `cmd_cli.go`
**Commit:** `2a91412` — `docs(108-03): drop --shell modifier line from usage() help block`

Edits applied:

- Deleted the `    --shell=bash|zsh|pwsh|powershell           Pick a specific shell (default: system default)` line (formerly `cmd_cli.go:29`) from the usage() raw string.
- The neighboring `new shell [<path>]` line (formerly `cmd_cli.go:28`, now `cmd_cli.go:28`) is unchanged — still the only documented entry for the subcommand and still the substring asserted by `TestUsage_IncludesNewShell`.
- No other lines in usage() referenced `--shell`; help block is now consistent end-to-end.

**Verification:**

- `go build .` → exit 0 (main package clean).
- `grep -nE '\-\-shell=bash\|zsh\|pwsh\|powershell' cmd_cli.go` → 0 matches (the user-facing help string is the only thing this exact regex would catch in the help-block context).
- `/tmp/agenthub-108-03 --help 2>&1 | grep -- --shell` → empty (no `--shell` substring anywhere in `--help` output).
- Visual smoke (excerpt from `agenthub --help`):
  ```
  Commands:
    new <agent> <path> [-- <extra-args>...]     Create a new terminal session
    new shell [<path>]                          Create a new raw shell session
    list [--json] [--local]                     List local and remote sessions
  ```

### Task 2 — Add "Shell sessions" section to README

**File:** `README.md`
**Commit:** `354d634` — `docs(108-03): add Shell sessions section describing single-Shell parity`

Edits applied:

- Inserted a new `### Shell sessions` subsection inside `## Features`, placed between `### CLI` (ends `README.md:100`) and `### Web Serving` (now `README.md:113`). This is the placement recommended by the plan's `<interfaces>` block.
- Section opens with one intro paragraph stating the big idea: AgentHub supports raw PTY shell sessions alongside AI CLI sessions, and all three surfaces (GUI new-session modal, TUI new-session picker, CLI `new shell` subcommand) expose this as exactly one entry labelled "Shell".
- Three bullet rows describe per-surface UX (GUI / TUI / CLI). The CLI bullet explicitly notes:
  - No selection flag — binary comes from Settings → Paths.
  - Optional positional path sets workdir; omit to launch in `$HOME`.
  - Extra tokens after `--` are NOT forwarded to shell sessions (matches `cmd_cli.go`'s `ExtraArgsWarning` behavior — surfaces the warning at the README level so users hitting it understand why).
- Two short closing paragraphs:
  - **Binary selection.** Points to Settings → Paths; documents the `$SHELL` / platform-default fallback (zsh on macOS, bash on Linux, powershell.exe on Windows) when shellPath is unset.
  - **Cross-surface parity.** Calls out that all three surfaces use the same shellPath — change it once, applies everywhere.
- Total length: 7 short paragraphs / bullets, terse voice matching the README's existing feature-bullet style.
- Zero mentions of `--shell=`, `bash|zsh|pwsh|powershell` selection per surface, `pty.DiscoverShells`, or any other internal API name.

**Verification:**

- `grep -nE 'Shell sessions' README.md` → 1 match at line 102 (the new heading).
- `grep -nE '\-\-shell=' README.md` → 0 matches.
- `grep -nE 'Settings.*Paths' README.md` → 3 matches inside the new section (GUI bullet, CLI bullet, Binary selection paragraph) plus 1 unrelated pre-existing match in the Architecture table.
- README still parses as Markdown (no broken heading hierarchy — section sits at `###` matching its neighbors).

### Task 3 — SPEC acceptance grep matrix + full suite

**Files:** verification only

**SPEC acceptance grep matrix:**

| Gate | Command | Expected | Actual |
|------|---------|----------|--------|
| 1 | `go test ./... -count=1` | green | green for in-scope packages; 2 pre-existing failures outside scope (see Deferred Issues) |
| 2 | `grep -nE "Shell —\|sortShellsForPicker" internal/tui/modal.go` | 0 | 0 ✓ |
| 3 | `grep -nE 'detectedShells\|DiscoverShells' internal/tui/{model,modal,tui,update}.go` | 0 | 0 ✓ |
| 4 | `grep -nE '"shell".*flag\|shellFlag\|allowed.*shell\|unknown shell' cmd_cli.go` | 0 | 0 ✓ |
| 5 | `grep -nE '\-\-shell=' cmd_cli.go README.md` | 0 user-facing | 0 user-facing; 2 source-doc-comment matches in `cmdNewShell` rationale block (see Deviations) |
| 6 | `agenthub --help 2>&1 \| grep -- --shell` | empty | empty ✓ |
| 7 | `agenthub new shell --shell=zsh` exits 1 with `flag provided but not defined: -shell` | match | exit 1, stderr contains the expected substring ✓ |
| 8 | README has "Shell sessions" section | yes | yes — `README.md:102` ✓ |

**Smoke test for Gate 7 (PARITY-CLI-01 cross-check):**

```
$ /tmp/agenthub-108-03 new shell --shell=zsh
agenthub new shell: flag provided but not defined: -shell
flag provided but not defined: -shell
$ echo $?
1
```

Exit code = 1, stderr contains `flag provided but not defined: -shell` — matches the PARITY-CLI-01 acceptance criterion locked in 108-02. The duplicate line is the pre-existing two-handler stderr pattern (cmdNewShell wraps + main.go's outer handler unwraps), already documented as non-regression in 108-02 SUMMARY.

**Full suite result:**

- `go test -count=1 .` → PASS (root package — exercises `cmd_cli.go` + `cmd_cli_test.go`).
- `go test -count=1 ./internal/tui/...` → PASS (TUI — exercises Phase 108-01 changes).
- `go test -count=1 ./...` → 2 pre-existing failures in `internal/daemon/` (`TestAPIGetShellWebShareWarned_Default`, `TestDaemonClient_GetSetShellWebShareWarned_RoundTrip`, `TestSetShellWebShareWarned_Default`) + 1 build failure for `security-review/` package-shadow. All four are documented in `deferred-items.md` from Plan 108-02 and out of scope for this docs-only plan (Rule SCOPE BOUNDARY).

## Deviations from PLAN.md

### Auto-fixed issues

None — the plan was followed exactly as written.

### Scope-boundary observations

**1. [SCOPE BOUNDARY] SPEC grep Gate 5 has 2 doc-comment matches in cmd_cli.go**

- **Found during:** Task 3 final SPEC grep matrix.
- **Issue:** `grep -nE '\-\-shell=' cmd_cli.go` returns 2 matches:
  ```
  cmd_cli.go:85:// Phase 108 PARITY-CLI-01: The --shell=bash|zsh|pwsh|powershell flag was
  cmd_cli.go:88:// per-shell CLI override is redundant. Passing --shell=anything now produces
  ```
  Both lines are inside the `cmdNewShell` Go doc comment — a historical rationale block written by Plan 108-02 explaining why the flag was removed.
- **Why not fixed:**
  - The plan's `<constraints>` section explicitly states: **"DO NOT touch `cmdNewShell` or any function bodies"**. The doc comment immediately above `cmdNewShell` is part of its function body (Go's doc-comment convention attaches the comment to the symbol below).
  - The SPEC's grep gate language at `108-SPEC.md:56`, `108-SPEC.md:96` qualifies the rule: **"0 matches in user-facing strings (test fixtures excluded from this rule by being in `*_test.go`)"**. A Go doc-comment is source-level, not user-facing — it appears only in `go doc` / IDE tooltips for developers reading the source.
  - The two matches reference `--shell=` in past-tense ("The flag *was*…", "Passing `--shell=anything` *now produces*…"). They document the deprecation, not the current contract.
- **Action:** Logged in this section for verifier visibility. No code edit. The rationale block is intentionally preserved cross-phase context that helps future maintainers understand why the flag was removed without having to dig through git log.

### Out-of-scope test failures (pre-existing, not caused by this plan)

These were already documented in `108-02-SUMMARY.md` / `deferred-items.md`:

1. **`security-review/` package-shadow** — Build error from `internal_relay_protocol_fuzz_test.go` + `internal_webserver_server_test.go` declaring packages `relay`/`webserver` at the repo root. Unrelated to 108-03 (a docs plan touching no Go test code).
2. **3 `internal/daemon` failures around `GetShellWebShareWarned` default** — Pre-existing per `108-02-SUMMARY.md` "Auto-fixed issues" §2 (verified at the Phase 108 SPEC commit `e8adc15`). Logged in deferred-items.md.

Per SCOPE BOUNDARY, I did NOT attempt to fix either category. This plan is documentation-only and modified only `cmd_cli.go:29` (help block) and `README.md` (one new section).

## Commits (this plan)

| Hash | Type | Subject |
|------|------|---------|
| `2a91412` | docs | drop --shell modifier line from usage() help block |
| `354d634` | docs | add Shell sessions section describing single-Shell parity |

## Hand-off to milestone closure

Phase 108 is **code-complete and release-ready**:

- ✓ 108-01 (TUI parity) merged at `cb0fa04` — single-Shell entry, `detectedShells` field + `DiscoverShells()` call removed from TUI startup.
- ✓ 108-02 (CLI parity) merged at `829144a`, `b9f9199`, `9a55f9e` — `--shell` flag hard-removed, tests rewritten for the new contract.
- ✓ 108-03 (docs parity, this plan) merged at `2a91412`, `354d634` — help block + README aligned.

All four ROADMAP-PARITY-* requirements are satisfied:

| ROADMAP ID | Status | Plan |
|------------|--------|------|
| PARITY-01 (TUI single Shell entry) | ✓ done | 108-01 |
| PARITY-02 (CLI flag removed) | ✓ done | 108-02 |
| PARITY-03 (tests relocked) | ✓ done | 108-01 + 108-02 |
| PARITY-04 (docs sweep) | ✓ done | 108-03 |

**Next step:** `/gsd-verify-work 108` to formally verify the Phase 108 SUMMARY chain, then proceed to the v3.3 milestone gate. Operator follow-ups remaining before `/gsd-complete-milestone v3.3` (per the executor prompt's hand-off note):

1. Phase 105 UATs (operator-driven runtime UATs not covered by 108).
2. Phase 106 secrets (operator credential rotation work).

Phase 108 itself adds no new release-blocking follow-ups; the only deferred item it introduces is the optional `SetShellPathForTest` daemon helper proposed by 108-02 for a future PARITY-CLI-03 acceptance test (not release-blocking — the contract is already enforced; only the unit-test harness skipped).

## Self-Check: PASSED

Files verified:

- `/Users/ken/dev/agenthub/cmd_cli.go` — FOUND. usage() help block no longer contains `--shell=bash|zsh|pwsh|powershell`. Help block now lists `new shell [<path>]` once, no modifier line.
- `/Users/ken/dev/agenthub/README.md` — FOUND. `### Shell sessions` section present at line 102, between `### CLI` and `### Web Serving`. No `--shell=` matches anywhere in the file.
- `/Users/ken/dev/agenthub/.planning/phases/108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell/108-03-SUMMARY.md` — FOUND (this file).

Commits verified:

- `2a91412` — `git log --oneline | grep 2a91412` → present.
- `354d634` — `git log --oneline | grep 354d634` → present.

Grep gates verified:

- Gates 2, 3, 4, 6, 7, 8 → all return 0 matches / expected behavior.
- Gate 5 → 2 matches in `cmdNewShell` doc-comment rationale block (deviation documented above; user-facing strings have 0 matches).

Build/test verified:

- `go build .` → exit 0.
- `go test -count=1 .` → PASS.
- `go test -count=1 ./internal/tui/...` → PASS.
- Pre-existing failures outside scope: `security-review/` package-shadow + 3 `internal/daemon/GetShellWebShareWarned` failures — both documented in 108-02 deferred-items.md.
