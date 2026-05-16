# Phase 108: TUI + CLI shell-entry collapse — parity with Phase 107 GUI — Context

**Gathered:** 2026-05-16
**Status:** Ready for planning
**Source:** PRD Express Path (108-SPEC.md, ambiguity 0.125)

<domain>
## Phase Boundary

Collapse the TUI new-session agent picker and the `agenthub new shell` CLI surface to a single "Shell" entry — matching Phase 107's GUI SHELL-10 collapse — so all three surfaces (GUI, TUI, CLI) expose exactly one shell-session entry, with the spawned binary resolved exclusively from the daemon's Settings-stored `shellPath`.

**In scope:**
- TUI new-session agent picker collapse to a single static "Shell" entry (label = `"Shell"`).
- Removal of `--shell=X` flag from `agenthub new shell` (hard removal, no deprecation period).
- Rewriting TUI cycle-order tests + CLI `--shell` flag tests to lock the new contract.
- Updating CLI help text and README to remove references to per-surface shell-variant selection.
- Removing the `detectedShells` field and `pty.DiscoverShells()` call from the TUI startup path.

**Out of scope:**
- GUI changes (already done in Phase 107).
- Daemon API changes — `GetShellPath()` / `SetShellPath()` and resolution at `engine.go:500-530` are unchanged.
- TUI Settings UI for shell-binary selection (v3.4 candidate).
- CLI subcommand to view or set `shellPath` (v3.4 backlog).
- Hardening daemon behavior when persisted `shellPath` is invalid (separate phase).
- Stderr warnings or executable-existence checks added to the CLI — explicitly rejected.
- Deprecating `--shell` with a one-release warning window — explicitly rejected (hard removal).
- Removing `pty.DiscoverShells()` itself — only its TUI-consumption path is removed.

</domain>

<decisions>
## Implementation Decisions (locked from SPEC.md)

### PARITY-TUI-01 — Single TUI Shell entry
- `agentEntries()` (`internal/tui/modal.go:60-75`) appends exactly ONE shell entry with `cliKey = "shell"` and `displayLabel = "Shell"`.
- Per-shell discovery / `sortShellsForPicker` / iteration over `m.detectedShells` are removed from the picker code path.
- Acceptance: unit test returns `len(detectedCLIs) + 1` entries when shells discovered, trailing entry `cliKey == "shell"` && `displayLabel == "Shell"`. `grep -nE "Shell —|sortShellsForPicker" internal/tui/modal.go` → 0 in live code.

### PARITY-TUI-02 — Picker label is just "Shell"
- Label is the literal string `"Shell"` — no path, no basename, no detail line.
- No `GetShellPath()` round-trip on modal open.
- Acceptance: rendered picker contains `"< Shell >"`, does NOT contain `"Shell —"`, `"system default"`, or path-like substrings.

### PARITY-TUI-03 — Cycle-order test rewritten
- `internal/tui/update_test.go:1141-1172` (`TestAgentPicker_CycleOrder_AICLIsThenShells`) rewritten to: AI CLIs in `detectedCLIs` order → exactly one `"Shell"` entry → wraps to first AI CLI.
- Companion tests `TestAgentPicker_IncludesShellEntries` (`update_test.go:1095-1129`) and `TestAgentPicker_OnlyAICLIs_NoShells` (`update_test.go:1174-1192`) updated or replaced.
- Acceptance: `go test ./internal/tui/... -run 'AgentPicker'` passes; no remaining assertion expects `"Shell — bash"`, `"Shell — zsh"`, `"Shell — system default"`, `"Shell — PowerShell"`.

### PARITY-CLI-01 — `--shell` flag removed (hard removal)
- `cmd_cli.go:91-122` no longer declares `--shell`. Allowlist map, empty-value branch, and locked-stderr error strings deleted.
- Session always created with `cli = "shell"`.
- Passing `--shell=anything` produces Go `flag` package's default `flag provided but not defined: -shell` stderr and exit 1.
- Acceptance: `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` → 0 in live code. `agenthub new shell --shell=zsh` exits 1 with substring `flag provided but not defined: -shell`.

### PARITY-CLI-02 — Shell binary resolved from Settings
- CLI ONLY ever passes `cli = "shell"` to `daemon.CreateSession`.
- Daemon resolution at `engine.go:500-530` is unchanged — same path as GUI/TUI.
- Acceptance: CLI integration test creating a shell session against a daemon with `shellPath = /bin/bash` records `/bin/bash` as the spawned binary; CLI contains no per-shell binary selection logic.

### PARITY-CLI-03 — Invalid Settings shellPath falls back silently
- Daemon's existing fallback to `$SHELL` / platform default handles invalid paths.
- CLI adds NO stderr warning, executable-existence check, or non-zero exit.
- Acceptance: with `shellPath = /nonexistent/shell`, `agenthub new shell` exits 0 with no CLI stderr. No new validation code in `cmdNewShell`.

### PARITY-DOCS-01 — Help text and README updated
- CLI help block: `new shell [<path>]   Create a new raw shell session` (single line, no `--shell` modifier).
- README "Shell sessions" section describes single-Shell entry on all three surfaces, points to Settings → Paths.
- Acceptance: `grep -nE '\-\-shell=|bash\|zsh\|pwsh' cmd_cli.go README.md` → 0 matches in user-facing strings (test fixtures excluded).

### PARITY-TUI-04 — Discovery field removed
- `internal/tui/model.go:130-135` `detectedShells []pty.DetectedShell` field removed.
- `internal/tui/tui.go:34` `pty.DiscoverShells()` call removed.
- Test helper at `internal/tui/update_test.go:26` no longer needs reset.
- `pty.DiscoverShells()` itself stays — only TUI consumption removed.
- Acceptance: `grep -nE 'detectedShells|DiscoverShells' internal/tui/*.go` (excluding `*_test.go`) → 0 matches. Existing TUI tests still pass.

### Claude's Discretion
- Test layout — whether to rename `TestAgentPicker_CycleOrder_AICLIsThenShells` or keep the name with rewritten body.
- README section structure — heading hierarchy and ordering of GUI/TUI/CLI bullets within the "Shell sessions" section.
- Ordering of edits across the four file groups (TUI picker, CLI flag, tests, docs). The plan should choose an ordering that keeps the test suite green between commits where possible.
- Whether to bundle CLI flag removal + test rewrite into a single commit or split. Recommendation: split per requirement ID for clean atomic commits.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 108 SPEC
- `.planning/phases/108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell/108-SPEC.md` — 8 locked requirements, acceptance criteria, traceability matrix

### Phase 107 GUI reference (already-shipped pattern to mirror)
- `frontend/src/components/NewSessionModal.tsx:146-157` — GUI single-Shell row reference
- Phase 107 summaries — `.planning/phases/107-shell-ux-collapse-binary-path-picker-clean-exit-handling/107-03-SUMMARY.md` (NewSessionModal collapse + Settings → Paths plumbing)

### TUI code to modify
- `internal/tui/modal.go:31-75` — `sortShellsForPicker` (delete) and `agentEntries()` (rewrite)
- `internal/tui/model.go:130-135` — `detectedShells` field (delete)
- `internal/tui/tui.go:34` — `pty.DiscoverShells()` call (delete)
- `internal/tui/update_test.go:1095-1192` — three tests to rewrite/replace
- `internal/tui/update_test.go:26` — test helper reset (delete after field removed)
- `internal/tui/styles.go:111` — shell badge styling (already single-color per SPEC scout — no change)

### CLI code to modify
- `cmd_cli.go:29-30` — help block (update)
- `cmd_cli.go:91-145` — `--shell` flag registration, validation, allowlist, locked stderr (delete `--shell` parts; keep core `new shell [<path>]` handling)

### Daemon (read-only — do not modify)
- `internal/daemon/engine.go:500-530` — `cli == "shell"` resolution path through `e.shellPath` (already in place from Phase 107)
- `internal/daemon/engine.go:670-705` — `GetShellPath()` / `SetShellPath()` (already in place from Phase 107)

### Cross-surface UAT origin (release-blocker context)
- `.planning/phases/101-shell-session-surfaces-web-share-gating/101-UAT.md:79-92` — Test 3 / Test 4 that surfaced the gap

### Docs to update
- `cmd_cli.go` help string
- `README.md` "Shell sessions" section (or equivalent)

</canonical_refs>

<specifics>
## Specific Ideas

See SPEC.md "Requirements" section — 8 requirements with file:line refs, current/target states, and acceptance criteria for each. Plan should map 1:1 to those requirements (or group with explicit traceability).

Implementation ordering recommendation:
1. TUI picker collapse (PARITY-TUI-01, PARITY-TUI-02) — local change, doesn't break tests yet.
2. TUI cycle-order test rewrite (PARITY-TUI-03) — relock the contract.
3. TUI discovery field removal (PARITY-TUI-04) — finalize TUI side.
4. CLI flag removal (PARITY-CLI-01) — code change.
5. CLI tests update (PARITY-CLI-02 + acceptance for PARITY-CLI-01) — relock CLI contract.
6. CLI invalid-path test (PARITY-CLI-03).
7. Docs (PARITY-DOCS-01) — help string + README.

</specifics>

<deferred>
## Deferred Ideas

- TUI Settings UI for shell-binary selection — v3.4 candidate.
- CLI `agenthub config set shell.path` subcommand — v3.4 backlog.
- Daemon-side validation when persisted `shellPath` is invalid — separate phase.

</deferred>

---

*Phase: 108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell*
*Context gathered: 2026-05-16 via PRD Express Path (SPEC.md as the PRD)*
