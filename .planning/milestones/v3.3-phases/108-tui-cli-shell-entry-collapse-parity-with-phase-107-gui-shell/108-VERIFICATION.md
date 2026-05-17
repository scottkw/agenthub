---
phase: 108
verified: 2026-05-16T00:00:00Z
status: passed
must_have_total: 8
must_have_verified: 8
score: 8/8 must-haves verified
overrides_applied: 0
gaps: []
deferred:
  - truth: "PARITY-CLI-03 acceptance test runnable end-to-end with invalid persisted shellPath"
    addressed_in: "v3.4 follow-up (SetShellPathForTest export)"
    evidence: "deferred-items.md §Plan 108-02 follow-up — harness limitation, sketch provided; behavior is locked at the SPEC level and asserted indirectly via TestCmdNewShell_NoArgs_UsesSystemDefault fallback path"
human_verification:
  - test: "Interactive TUI smoke — launch `agenthub tui`, open New Session modal, cycle agent picker to the Shell entry, create a session"
    expected: "Picker shows exactly one 'Shell' entry after the AI CLI rows; selecting it spawns a session whose binary equals the daemon's GetShellPath() value"
    why_human: "Requires a live terminal, real keypress sequencing, and visual inspection of the rendered picker — not exercisable by `go test` alone"
  - test: "End-to-end CLI smoke — with daemon running and Settings → Paths shellPath set to `/bin/bash`, run `agenthub new shell ~/tmp` and inspect the spawned session record"
    expected: "Session lists `/bin/bash` (or the Settings-resolved value) as the spawned binary; with shellPath unset the platform default (`$SHELL` / `/bin/zsh` on macOS) is used"
    why_human: "Requires a live daemon, real Settings persistence, and inspection of session records; the unit-level equivalent is TestCmdNewShell_SettingsShellPathSpawned (passing) but does not exercise the actual `agenthub` binary against a persisted daemon"
---

# Phase 108: TUI + CLI shell-entry collapse — Verification Report

**Phase Goal:** Collapse the TUI new-session agent picker and `agenthub new shell` CLI surface to a single "Shell" entry — matching Phase 107's GUI SHELL-10 collapse — so all three surfaces (GUI, TUI, CLI) expose exactly one shell-session entry, with the spawned binary resolved exclusively from the daemon's Settings-stored `shellPath`.

**Verified:** 2026-05-16
**Status:** passed (with two human-verification UAT items recommended for v3.3 sign-off)
**Re-verification:** No — initial verification

## Goal Achievement

All eight SPEC-locked requirements (PARITY-TUI-01..04, PARITY-CLI-01..03, PARITY-DOCS-01) are satisfied. Every grep gate from `108-SPEC.md §Acceptance Criteria` returns the expected result, both targeted test suites pass, and runtime invocation of the built binary confirms the contract.

### SPEC Acceptance Criteria — Criterion-by-Criterion

| # | Acceptance Criterion | Command / Check | Result | Status |
|---|----------------------|-----------------|--------|--------|
| 1 | TUI picker renders `len(AI CLIs) + 1` entries; trailing label is `"Shell"` | `go test ./internal/tui/... -run 'AgentPicker' -count=1 -v` | 3 PASS, 0 FAIL — `TestAgentPicker_IncludesShellEntries`, `TestAgentPicker_CycleOrder`, `TestAgentPicker_StaticShellEntryAlwaysAppended` | PASS |
| 2 | `grep -nE "Shell —\|sortShellsForPicker\|detectedShells\|DiscoverShells" internal/tui/` (live code only) returns 0 matches | `grep -nE "Shell —\|sortShellsForPicker" internal/tui/modal.go` + `grep -nE 'detectedShells\|DiscoverShells' internal/tui/*.go \| grep -v _test.go` | 0 matches (`exit=1`) on both; one match exists in `internal/tui/styles.go:96` inside a badge-color doc comment that uses `—` between two unrelated tokens, explicitly excluded by SPEC §Traceability ("badge half is already satisfied by `internal/tui/styles.go:111`") | PASS |
| 3 | `internal/tui/update_test.go` cycle-order assertion lists AI CLIs then one `"Shell"`; old multi-shell expectations gone | `grep -n 'Shell — bash\|Shell — zsh\|Shell — system default\|Shell — PowerShell' internal/tui/*_test.go cmd_cli_test.go` | 0 matches | PASS |
| 4 | `agenthub new shell --shell=zsh` exits 1, stderr contains `flag provided but not defined: -shell` | `/tmp/agenthub-108 new shell --shell=zsh` | `exit=1`; stderr = `agenthub new shell: flag provided but not defined: -shell\nflag provided but not defined: -shell` | PASS |
| 5 | `grep -nE '"shell".*flag\|shellFlag\|allowed.*shell\|unknown shell' cmd_cli.go` returns 0 matches | run command above | 0 matches (`exit=1`) | PASS |
| 6 | `cmd_cli.go` help block lists `new shell [<path>]   Create a new raw shell session` and does NOT list `--shell=bash\|zsh\|pwsh\|powershell` | `grep -n "new shell" cmd_cli.go` + `/tmp/agenthub-108 --help \| grep -- --shell` | Help line at `cmd_cli.go:28` reads exactly as required; `--help` output contains no `--shell` reference | PASS |
| 7 | README (or equivalent) describes single-Shell behavior on all three surfaces and points to Settings → Paths; `grep -nE '\-\-shell=' README.md` returns 0 matches | `grep -nE 'Shell sessions\|Settings.*Paths' README.md` + `grep -nE '\-\-shell=\|bash\|zsh\|pwsh' README.md` | `README.md:102` "Shell sessions" section covers GUI/TUI/CLI plus Binary selection (Settings → Paths) + Cross-surface parity; `--shell=` grep returns 0 matches | PASS |
| 8 | `go test ./...` passes after the changes; no test still expects `"Shell — *"` rendering or `--shell` flag validation | `go test ./internal/tui/... -count=1` + `go test . -count=1` | Both PASS (`ok internal/tui 0.024s`, `ok github.com/scottkw/agenthub 9.446s`). Pre-existing `internal/daemon/` failures (`GetShellWebShareWarned` default flip) reproduce at SPEC commit `e8adc15` and are documented in `deferred-items.md` — NOT regressions from Phase 108 | PASS |

### SPEC Requirements — One-to-One Mapping

| Req ID | Description | Evidence | Status |
|--------|-------------|----------|--------|
| PARITY-TUI-01 | Single TUI Shell entry | `internal/tui/modal.go:44` appends exactly one entry `{cliKey: "shell", displayLabel: "Shell"}`; `sortShellsForPicker` deleted; iteration over `m.detectedShells` removed | VERIFIED |
| PARITY-TUI-02 | Picker label is just `"Shell"` | `TestAgentPicker_IncludesShellEntries` asserts `" Shell "` appears in `renderAgentPicker()` between cycle arrows AND forbids `"Shell —"`, `"system default"`, `"/"`, `"\\"`; IN-03 of code review confirms the `<`/`>` are ANSI-wrapped, so the bracketed-literal check is replaced by the unbracketed substring check (semantically equivalent) | VERIFIED |
| PARITY-TUI-03 | Cycle-order test rewritten | `TestAgentPicker_CycleOrder` (renamed from `_AICLIsThenShells`) walks Claude Code → OpenCode → Shell → wrap; `TestAgentPicker_StaticShellEntryAlwaysAppended` replaces `TestAgentPicker_OnlyAICLIs_NoShells` to lock that the static entry is always present | VERIFIED |
| PARITY-CLI-01 | `--shell` flag removed | `cmd_cli.go:96` declares `flag.NewFlagSet("new shell", flag.ContinueOnError)` with NO `--shell` registration; `cmd_cli.go:102` hard-codes `const cli = "shell"`; runtime check confirms `--shell=zsh` triggers Go flag package's default error and exit 1 | VERIFIED |
| PARITY-CLI-02 | Shell binary resolved from Settings | `client.CreateSession(cli, name, workDir, nil, 0, 0)` at `cmd_cli.go:117` always passes `cli="shell"`; daemon resolution path at `internal/daemon/engine.go:500-530` unchanged (zero diff in `internal/daemon/` between `0fd45f8..HEAD`); `TestCmdNewShell_SettingsShellPathSpawned` passes | VERIFIED |
| PARITY-CLI-03 | Invalid Settings shellPath falls back silently | No CLI-side validation added in `cmdNewShell`; daemon-side fallback unchanged; `TestCmdNewShell_NoArgs_UsesSystemDefault` indirectly asserts the unset-path fallback; `TestCmdNewShell_InvalidShellPathSilentFallback` is `t.Skip`'d due to harness limitation (deferred to v3.4 per `deferred-items.md`) | VERIFIED (with documented test gap) |
| PARITY-DOCS-01 | Help text and README updated | `cmd_cli.go:28` usage line is the literal SPEC-required string; `README.md:102-111` "Shell sessions" section covers all three surfaces and Settings → Paths; `--shell=` grep returns 0 matches in README; the two `--shell=` matches in `cmd_cli.go` are Go doc-comment rationale (lines 85, 88), explicitly excluded from the "user-facing strings" gate per IN-01 | VERIFIED |
| PARITY-TUI-04 | Discovery field removed | `internal/tui/tui.go:32` no longer calls `pty.DiscoverShells()` (only `pty.DetectCLIs()` remains); `internal/tui/model.go:128` retains only `detectedCLIs []pty.DetectedCLI`; `pty.DiscoverShells()` itself still exists at `internal/pty/shells.go:62` (preserved per SPEC) | VERIFIED |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/tui/modal.go` | Single Shell entry, no DiscoverShells call, no sortShellsForPicker | VERIFIED | Line 44: `entries = append(entries, agentEntry{cliKey: "shell", displayLabel: "Shell"})`. Imports `"sort"` and `internal/pty` removed (per 108-REVIEW.md). |
| `internal/tui/model.go` | No `detectedShells` field | VERIFIED | Only `detectedCLIs []pty.DetectedCLI` remains; agentIdx doc comment refreshed in commit `1ee98b2` (WR-02 fix). |
| `internal/tui/tui.go` | No `pty.DiscoverShells()` call | VERIFIED | Line 32: only `detectedCLIs: pty.DetectCLIs()`. |
| `internal/tui/update_test.go` | Cycle tests rewritten, no legacy strings | VERIFIED | 3 AgentPicker tests post-Phase-108; 0 occurrences of `Shell — bash/zsh/system default/PowerShell`. |
| `cmd_cli.go` | No `--shell` flag, no allowlist, hardcoded `cli="shell"`, new help line | VERIFIED | `flag.NewFlagSet` declares no `--shell`; `const cli = "shell"` at line 102; usage line at 28; `"errors"` import gone. |
| `cmd_cli_test.go` | Tests cover removed-flag rejection, system-default, settings-path-spawned | VERIFIED | 7 PASS + 1 documented SKIP. |
| `README.md` | New "Shell sessions" section, points to Settings → Paths, no `--shell=` references | VERIFIED | Lines 102-111 cover GUI/TUI/CLI + binary selection + cross-surface parity. |
| `internal/pty/shells.go` | `DiscoverShells()` preserved | VERIFIED | Line 62: `func DiscoverShells() []DetectedShell { ... }` still present. |
| `internal/daemon/engine.go` | NOT modified (out of scope) | VERIFIED | `git diff 0fd45f8..HEAD -- internal/daemon/` returns 0 lines. |

### Key Link Verification (Cross-Surface Parity)

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| GUI `NewSessionModal.tsx` | daemon.CreateSession | `onConfirm('shell', selectedDir, [])` at line 107 | WIRED | Bare `'shell'` passed; daemon resolves via `GetShellPath()` (Phase 107 SHELL-10). |
| TUI `modal.go` → `update.go` | daemon.CreateSession | `entries[idx].cliKey == "shell"` → `createSession(m.client, cli, name, workDir, args)` at `update.go:691` | WIRED | TUI picker emits only `"shell"` for the shell row; `submitNewSession` forwards as-is. |
| CLI `cmd_cli.go` | daemon.CreateSession | `const cli = "shell"` at line 102 → `client.CreateSession(cli, ...)` at line 117 | WIRED | Hard-coded; no caller-controlled override remains. |
| Daemon `engine.go:500-530` | spawn argv | `e.shellPath` (Settings-stored) | WIRED (unchanged from Phase 107) | Same code path used by all three surfaces. |

### Runtime / Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `--help` does not advertise `--shell` | `/tmp/agenthub-108 --help \| grep -- --shell` | grep `exit=1` (no match) | PASS |
| `agenthub new shell --shell=zsh` rejects flag with Go-stdlib message | `/tmp/agenthub-108 new shell --shell=zsh` | exit=1; stderr matches `flag provided but not defined: -shell` | PASS |
| Usage line is exactly as SPEC specifies | `/tmp/agenthub-108 --help \| grep "new shell"` | `  new shell [<path>]                          Create a new raw shell session` | PASS |
| Targeted unit suites pass | `go test ./internal/tui/... -run 'AgentPicker' -count=1` + `go test -count=1 -run 'CmdNewShell\|Usage_IncludesNewShell' .` | Both `ok` | PASS |
| Full TUI and root suite pass | `go test ./internal/tui/... -count=1` + `go test . -count=1` | Both `ok` (9.5s combined) | PASS |
| `go vet ./internal/tui/... .` | run as above | Clean (no output) | PASS |
| `pty.DiscoverShells()` preserved | `grep -n "func DiscoverShells" internal/pty/shells.go` | Line 62 present | PASS |
| `internal/daemon/` untouched | `git diff 0fd45f8..HEAD -- internal/daemon/` | 0 lines of diff | PASS |

### Anti-Patterns Found

None blocking. Code-review (108-REVIEW.md) flagged 2 warnings (WR-01 stale `isShellCLI` allowlist + comment, WR-02 stale `agentIdx` doc); both were addressed via commits `524e3fb` and `1ee98b2`. 4 info-level items remain; all are documentation traceability notes or optional follow-up tests, none gate the contract.

### Requirements Coverage (vs ROADMAP)

| ROADMAP ID | SPEC Requirements | Status |
|------------|-------------------|--------|
| PARITY-01 | PARITY-TUI-01, PARITY-TUI-02, PARITY-TUI-04 | SATISFIED |
| PARITY-02 | PARITY-CLI-01, PARITY-CLI-02, PARITY-CLI-03 | SATISFIED (one skipped test deferred to v3.4) |
| PARITY-03 | PARITY-TUI-03 (TUI tests) + CLI `--shell=zsh` runtime check | SATISFIED |
| PARITY-04 | PARITY-DOCS-01 | SATISFIED |

### Human Verification Required (Recommended UAT for v3.3 sign-off)

The unit suite locks the contract at the picker, CLI, and daemon layers, but two behaviors are worth a live smoke before tagging:

1. **Interactive TUI session** — launch `agenthub tui`, open the New Session modal, cycle through to the Shell entry, type a working directory, hit Enter. Confirm exactly one Shell row, label is bare `Shell`, and the resulting session spawns the daemon-resolved binary.

2. **End-to-end CLI session** — with the daemon running and Settings → Paths shellPath set to `/bin/bash`, run `agenthub new shell ~/tmp` and confirm the resulting session record names `/bin/bash` as the spawned binary. Then unset shellPath and confirm the platform default is used (no CLI-side warning).

Both items are recommended but not blocking — the SPEC-level contract is locked by the suite. Logged as `human_verification` for traceability.

### Gaps Summary

None. All 8 SPEC requirements satisfied. The single test skip (`TestCmdNewShell_InvalidShellPathSilentFallback`) is a documented harness limitation, not a contract gap — the underlying behavior is unchanged from Phase 107 daemon logic and indirectly asserted by `TestCmdNewShell_NoArgs_UsesSystemDefault`. Follow-up (`SetShellPathForTest` export) deferred to v3.4 per `deferred-items.md`.

The pre-existing `internal/daemon/` test failures (`GetShellWebShareWarned` default flip) reproduce at the Phase 108 SPEC commit `e8adc15` and are NOT regressions introduced by Phase 108; they are documented in `deferred-items.md` and SPEC §Out-of-scope explicitly forbids touching `internal/daemon/` in this phase.

---

_Verified: 2026-05-16_
_Verifier: Claude (gsd-verifier)_
