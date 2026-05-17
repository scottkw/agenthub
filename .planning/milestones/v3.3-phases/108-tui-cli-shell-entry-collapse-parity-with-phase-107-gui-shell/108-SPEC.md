# Phase 108: TUI + CLI shell-entry collapse — parity with Phase 107 GUI — Specification

**Created:** 2026-05-16
**Ambiguity score:** 0.125
**Requirements:** 8 locked

## Goal

Collapse the TUI new-session agent picker and the `agenthub new shell` CLI surface to a single "Shell" entry — matching Phase 107's GUI SHELL-10 collapse — so all three surfaces (GUI, TUI, CLI) expose exactly one shell-session entry, with the spawned binary resolved exclusively from the daemon's Settings-stored `shellPath`.

## Background

Phase 107 collapsed the GUI new-session modal to a single static "Shell" row (`frontend/src/components/NewSessionModal.tsx:146-157`) that reads the daemon-resolved path via the `GetShellPath()` Wails binding (Phase 107 SHELL-10 + SHELL-11). The daemon side already exposes `engine.GetShellPath()` / `SetShellPath()` and persists the user's choice (`internal/daemon/engine.go:670-705`). The TUI and CLI were intentionally out of scope in Phase 107.

Today the TUI agent picker still iterates `m.detectedShells` — populated by `pty.DiscoverShells()` at TUI start (`internal/tui/tui.go:34`) — and renders one row per discovered binary as `"Shell — system default"`, `"Shell — bash"`, `"Shell — zsh"` (`internal/tui/modal.go:60-75`, `sortShellsForPicker` at `modal.go:31-52`). The cycle-order assertion at `internal/tui/update_test.go:1141-1172` locks that multi-row contract.

The CLI's `agenthub new shell` subcommand accepts an optional `--shell=bash|zsh|pwsh|powershell` modifier (`cmd_cli.go:91-145`) with allowlist-locked stderr error strings for unknown/empty values. The help block at `cmd_cli.go:29-30` advertises this flag.

During 101-UAT Test 3 (2026-05-16) the user declared this cross-surface inconsistency a release-blocking gap for v3.3 (see `.planning/phases/101-shell-session-surfaces-web-share-gating/101-UAT.md:79-92`). Test 4 (CLI `--shell` flag) was skipped because the contract is being rewritten by this phase.

## Requirements

1. **PARITY-TUI-01 — Single TUI Shell entry**: The TUI new-session agent picker exposes exactly one "Shell" entry.
   - Current: Picker renders N rows (one per detected shell) as `"Shell — <displayName>"`; `m.detectedShells` is populated by `pty.DiscoverShells()` and rendered by `agentEntries()` at `internal/tui/modal.go:60-75`.
   - Target: `agentEntries()` appends exactly ONE shell entry with `cliKey = "shell"` and `displayLabel = "Shell"` (no em-dash, no variant suffix). Per-shell discovery / `sortShellsForPicker` / iteration over `m.detectedShells` are removed from the picker code path.
   - Acceptance: A unit test on the rebuilt `agentEntries()` returns exactly `len(detectedCLIs) + 1` entries when one or more shells are discovered, the last entry's `cliKey == "shell"` and `displayLabel == "Shell"`. `grep -nE "Shell —|sortShellsForPicker" internal/tui/modal.go` returns 0 matches in live code.

2. **PARITY-TUI-02 — Picker label is just "Shell"**: The TUI single Shell entry shows the literal string `Shell` in the picker — no resolved path, no basename suffix, no detail line.
   - Current: Label is `"Shell — " + sh.DisplayName` (e.g. `"Shell — system default"`).
   - Target: Label is the literal string `"Shell"`. No `GetShellPath()` round-trip is needed when rendering the picker.
   - Acceptance: `renderAgentPicker()` rendered with `agentIdx` pointing at the shell entry contains the substring `"< Shell >"` and does NOT contain `"Shell —"`, `"system default"`, or any path-like substring (`/`, `\`).

3. **PARITY-TUI-03 — Cycle-order test rewritten**: The TUI cycle-order assertion locks the new single-Shell contract.
   - Current: `internal/tui/update_test.go:1141-1172` (`TestAgentPicker_CycleOrder_AICLIsThenShells`) walks `Claude Code → OpenCode → Shell — system default → Shell — bash → Shell — zsh → Shell — PowerShell → wrap`.
   - Target: The test (renamed if appropriate) asserts the new cycle: AI CLIs in `detectedCLIs` order, then exactly one `"Shell"` entry, then wraps to the first AI CLI. Companion tests `TestAgentPicker_IncludesShellEntries` (`update_test.go:1095-1129`) and `TestAgentPicker_OnlyAICLIs_NoShells` (`update_test.go:1174-1192`) are updated or replaced to reflect the collapsed contract.
   - Acceptance: `go test ./internal/tui/... -run 'AgentPicker'` passes; no assertion in the suite still expects `"Shell — bash"`, `"Shell — zsh"`, `"Shell — system default"`, `"Shell — PowerShell"`, or any multi-row shell rendering.

4. **PARITY-CLI-01 — `--shell` flag removed**: The CLI `--shell=bash|zsh|pwsh|powershell` modifier is deleted outright (no deprecation period).
   - Current: `cmd_cli.go:91-122` registers a `--shell` flag, validates against `{bash,zsh,pwsh,powershell,""}`, and emits locked-stderr errors for unknown/empty values.
   - Target: `flag.NewFlagSet("new shell", ...)` no longer declares `--shell`. The allowlist map, empty-value branch, and locked-stderr error strings for unknown/empty `--shell` values are deleted. The session is always created with `cli = "shell"`. Passing `--shell=anything` produces Go `flag` package's default `flag provided but not defined: -shell` stderr and exit 1.
   - Acceptance: `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` returns 0 matches. Running `agenthub new shell --shell=zsh` exits 1; stderr contains the substring `flag provided but not defined: -shell`.

5. **PARITY-CLI-02 — Shell binary resolved from Settings**: `agenthub new shell [<path>]` always creates a session whose spawned binary is the daemon-resolved `GetShellPath()` value.
   - Current: CLI passes `cli = "shell"` (or `cli = "bash"` etc.) to `daemon.CreateSession`; daemon resolution path at `internal/daemon/engine.go:500-530` already routes the bare `"shell"` cli through `e.shellPath` when set. Today's CLI is one of the callers that can bypass this with `--shell=X`.
   - Target: After PARITY-CLI-01, the CLI ONLY ever passes `cli = "shell"`. Daemon resolution path is unchanged; CLI inherits the same Settings-driven binary selection as the GUI and TUI.
   - Acceptance: A CLI integration test creating a shell session against a test daemon whose `shellPath` is set to a known executable (e.g. `/bin/bash`) records that exact path as the spawned binary. The CLI itself contains no per-shell binary selection logic.

6. **PARITY-CLI-03 — Invalid Settings shellPath falls back silently**: When `shellPath` is set to a missing/non-executable path, `agenthub new shell` succeeds using the platform default with no CLI-side warning or error.
   - Current: Daemon `GetShellPath()` returns the user-set value verbatim if set; falls back to `$SHELL` / platform default if unset. Validation of executable-ness lives in `SetShellPath()` (write path), not the read path.
   - Target: Behavior is unchanged from today's daemon logic. The CLI does NOT add a stderr warning, executable-existence check, or non-zero exit when the daemon-resolved path is missing. (Daemon-side hardening of invalid persisted paths, if needed, is a separate phase.)
   - Acceptance: With a test daemon configured to return a `shellPath` value pointing at `/nonexistent/shell`, running `agenthub new shell` produces exit code 0 and no stderr output from the CLI process itself (any error surfaces later via daemon session lifecycle, not the CLI command). No new validation code added to `cmdNewShell`.

7. **PARITY-DOCS-01 — Help text and README updated**: All user-facing documentation reflects the new shape.
   - Current: `cmd_cli.go:29-30` documents `--shell=bash|zsh|pwsh|powershell`; README and any user-facing docs may reference multi-shell selection on TUI/CLI.
   - Target: CLI help block reads `new shell [<path>]   Create a new raw shell session` (single line, no `--shell` modifier). README "Shell sessions" section (or equivalent) describes the single-Shell entry on all three surfaces and points to Settings → Paths for binary selection. No lingering references to per-surface `bash|zsh|pwsh|powershell` selection.
   - Acceptance: `grep -nE '\\-\\-shell=|bash\\|zsh\\|pwsh' cmd_cli.go README.md` returns 0 matches in user-facing strings (test-fixture matches are excluded from this rule by being in `*_test.go`).

8. **PARITY-TUI-04 — Discovery field removed**: `m.detectedShells` and `pty.DiscoverShells()` are no longer wired into the TUI start-up path (or their values are no longer consumed by the picker).
   - Current: `internal/tui/model.go:130-135` defines `detectedShells []pty.DetectedShell`; `internal/tui/tui.go:34` populates it via `pty.DiscoverShells()`. Test helper at `internal/tui/update_test.go:26` resets it to `nil`.
   - Target: The TUI no longer calls `pty.DiscoverShells()` and no longer carries the `detectedShells` field. (`pty.DiscoverShells()` itself stays — it may be used elsewhere or in tests; only the TUI's *consumption* is removed.) Test helper reset is no longer needed.
   - Acceptance: `grep -nE 'detectedShells|DiscoverShells' internal/tui/*.go` (excluding `*_test.go`) returns 0 matches. Existing TUI tests still pass.

## Boundaries

**In scope:**
- TUI new-session agent picker collapse to a single static "Shell" entry (label = `"Shell"`).
- Removal of `--shell=X` flag from `agenthub new shell` (hard removal, no deprecation period).
- Rewriting TUI cycle-order tests + CLI `--shell` flag tests to lock the new contract.
- Updating CLI help text and README to remove references to per-surface shell-variant selection.
- Removing the `detectedShells` field and `pty.DiscoverShells()` call from the TUI startup path.

**Out of scope:**
- GUI changes — Phase 107 already collapsed the GUI; no further GUI work in this phase.
- Daemon API changes — `GetShellPath()` / `SetShellPath()` and the resolution logic at `engine.go:500-530` are unchanged. (Phase 107 already shipped them.)
- TUI Settings UI for shell-binary selection — the TUI has no Settings surface today; users set `shellPath` via GUI Settings → Paths (Phase 107 SHELL-11) or direct config file edit. Adding a TUI Settings UI is a separate v3.4 candidate.
- A CLI subcommand to view or set `shellPath` (e.g. `agenthub config set shell.path …`) — out of scope; users use the GUI Settings tab. May be a v3.4 backlog item.
- Hardening daemon behavior when persisted `shellPath` is invalid — daemon currently returns the user-set value verbatim from `GetShellPath()` and only validates on `SetShellPath()`. If runtime validation is needed, it's a separate phase.
- Stderr warnings or executable-existence checks added to the CLI — explicitly rejected (PARITY-CLI-03: silent fallback).
- Deprecating `--shell` with a one-release warning window — explicitly rejected (PARITY-CLI-01: hard removal).
- Removing `pty.DiscoverShells()` itself — only its TUI-consumption path is removed.

## Constraints

- Cross-surface parity is release-blocking for v3.3 (see `STATE.md` and `.planning/phases/101-shell-session-surfaces-web-share-gating/101-UAT.md`). Any partial completion that leaves TUI or CLI showing per-shell variants must NOT be merged.
- Hard removal of `--shell` (PARITY-CLI-01) is a CLI break for any caller scripting `agenthub new shell --shell=…`. Acceptable because (a) v3.3 is the only release where the flag has existed, (b) the flag was introduced in Phase 101 (same milestone), and (c) the daemon-side resolution makes the flag value redundant in 100% of cases.
- All four PARITY-* requirement IDs from the ROADMAP (PARITY-01..04) must map onto requirements written above (see Traceability below). The ROADMAP IDs and the SPEC IDs differ in naming because the SPEC splits them across 8 finer-grained requirements.

## Acceptance Criteria

- [ ] TUI agent picker, with one or more shells discovered on the host, renders exactly `len(AI CLIs) + 1` entries when cycled through with the wrap-around test, with the trailing entry's label exactly `"Shell"`.
- [ ] `grep -nE "Shell —|sortShellsForPicker|detectedShells|DiscoverShells" internal/tui/` (live code only, not `*_test.go`) returns 0 matches.
- [ ] `internal/tui/update_test.go` cycle-order assertion lists AI CLIs followed by exactly one `"Shell"` entry; old `Shell — bash` / `Shell — zsh` / `Shell — system default` / `Shell — PowerShell` expectation strings are all gone from the suite.
- [ ] `agenthub new shell --shell=zsh` exits 1 with stderr containing `flag provided but not defined: -shell`.
- [ ] `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` returns 0 matches in live code.
- [ ] `cmd_cli.go` help block lists `new shell [<path>]   Create a new raw shell session` and does NOT list `--shell=bash|zsh|pwsh|powershell`.
- [ ] README (or equivalent user-facing docs) describes single-Shell behavior on all three surfaces and points to Settings → Paths; `grep -nE '\\-\\-shell=' README.md` returns 0 matches.
- [ ] `go test ./...` passes after the changes; no test still expects `"Shell — *"` rendering or `--shell` flag validation.
- [ ] Manual smoke: `agenthub new shell ~/tmp` against a daemon with `shellPath = /bin/bash` records `/bin/bash` as the spawned binary in the session record; against a daemon with `shellPath = ""` (unset), the platform default (`$SHELL` or `/bin/zsh` on macOS) is spawned.

## Traceability

ROADMAP-declared requirements → SPEC requirements:

| ROADMAP ID  | SPEC requirement(s)                          |
|-------------|----------------------------------------------|
| PARITY-01   | PARITY-TUI-01, PARITY-TUI-02, PARITY-TUI-04 (badge half is already satisfied by `internal/tui/styles.go:111` — verified in scout) |
| PARITY-02   | PARITY-CLI-01, PARITY-CLI-02, PARITY-CLI-03 |
| PARITY-03   | PARITY-TUI-03 (TUI tests) + acceptance criterion on `agenthub new shell --shell=zsh` (CLI tests) |
| PARITY-04   | PARITY-DOCS-01                              |

## Ambiguity Report

| Dimension          | Score | Min  | Status | Notes                                                  |
|--------------------|-------|------|--------|--------------------------------------------------------|
| Goal Clarity       | 0.95  | 0.75 | ✓      | Three surfaces, one entry each — outcome fully concrete |
| Boundary Clarity   | 0.85  | 0.70 | ✓      | 9 explicit out-of-scope items with reasoning           |
| Constraint Clarity | 0.80  | 0.65 | ✓      | Hard CLI break accepted; release-blocking rationale captured |
| Acceptance Criteria| 0.85  | 0.70 | ✓      | 9 pass/fail criteria, most grep-or-test-runnable        |
| **Ambiguity**      | 0.125 | ≤0.20| ✓      |                                                        |

## Interview Log

| Round | Perspective    | Question summary                                              | Decision locked                                                                 |
|-------|----------------|---------------------------------------------------------------|---------------------------------------------------------------------------------|
| 0     | Researcher (scout) | What exists today on TUI/CLI vs Phase 107 GUI?            | TUI iterates `detectedShells`; CLI accepts `--shell=X`; daemon `GetShellPath` already in place |
| 1     | Boundary Keeper| What happens to the `--shell=bash\|zsh\|pwsh\|powershell` flag? | **Remove entirely** — `agenthub new shell [<path>]` stays as the way to create a shell; flag deleted, no deprecation |
| 1     | Boundary Keeper| What does the TUI single Shell entry display?                 | **Just "Shell"** — no path, no basename, no detail line. Zero new daemon round-trips on modal open |
| 1     | Failure Analyst| What if Settings shellPath points to a missing/non-executable binary? | **Fall back silently** — daemon's existing fallback to `$SHELL` / platform default; no CLI-side warning or error |

---

*Phase: 108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell*
*Spec created: 2026-05-16*
*Next step: /gsd-discuss-phase 108 — implementation decisions (test layout, README section structure, ordering of edits)*
