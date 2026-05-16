---
phase: 108-tui-cli-shell-entry-collapse-parity-with-phase-107-gui-shell
plan: "108-02"
subsystem: cli
tags: [parity, cli, shell, flag-removal, settings-driven]
requires:
  - Phase 107 SHELL-11 (daemon GetShellPath / SetShellPath + Settings → Paths)
  - internal/daemon/engine.go:500-530 (cli=="shell" → e.shellPath routing)
provides:
  - PARITY-CLI-01 (—-shell flag hard-removed from `agenthub new shell`)
  - PARITY-CLI-02 (CLI always passes cli="shell"; daemon resolves binary)
  - PARITY-CLI-03 (CLI silent on invalid daemon shellPath — partial; harness limitation)
affects:
  - cmd_cli.go
  - cmd_cli_test.go
tech-stack:
  added: []
  patterns:
    - "Go flag-package default error path (`flag provided but not defined: -shell`) instead of custom-locked stderr"
    - "client.SetShellPath wrapper exercised in CLI test for end-to-end Settings-resolution check"
key-files:
  modified:
    - cmd_cli.go (cmdNewShell: --shell flag declaration deleted; allowlist + empty/unknown branches deleted; errors import removed; const cli = \"shell\"; doc comment rewritten for Phase 108)
    - cmd_cli_test.go (7 --shell=X tests deleted; 3 new contract tests added; TestUsage_IncludesNewShell needle updated)
decisions:
  - "PARITY-CLI-03 silent-fallback test skipped at the harness level — SetShellPath rejects nonexistent paths and engine field is unexported. Documented follow-up in deferred-items.md."
  - "Gofmt-applied tab-prefixed bullet formatting in the cmdNewShell doc comment — separate style commit (9a55f9e) rather than amending the Task 1 commit per protocol."
metrics:
  duration: ~25min
  completed: 2026-05-16
---

# Phase 108 Plan 02: CLI `--shell` flag hard-removal + Settings-resolution contract lock

## One-liner

Delete `--shell=bash|zsh|pwsh|powershell` from `agenthub new shell` and rewrite cmdNewShell tests to lock the new Settings-driven contract (SPEC PARITY-CLI-01/02/03; ROADMAP PARITY-02 + CLI half of PARITY-03).

## Goal

Collapse the CLI's per-binary shell-selection override so the daemon's `engine.go:500-530` shellPath resolution path (Phase 107 SHELL-11) is the single source of truth for which shell binary `agenthub new shell` spawns — matching Phase 107's GUI collapse and the parallel Phase 108-01 TUI collapse.

## Task-by-task results

### Task 1 — Delete `--shell` flag, allowlist, and locked stderr from cmdNewShell

**Files:** `cmd_cli.go`
**Commit:** `829144a` — `refactor(108-02): remove --shell flag from cmdNewShell (PARITY-CLI-01/02/03)`

Edits applied:
- Deleted `shellFlag := fs.String("shell", "", "")` (was line 94).
- Deleted the `fs.Visit` + `explicitlySet` + empty-value branch (lines 99–110) and its locked stderr `agenthub new shell: --shell flag requires a value (one of: bash, zsh, pwsh, powershell)`.
- Deleted the `allowed` map + unknown-value branch (lines 111–122) and its locked stderr `agenthub new shell: unknown shell %q (allowed: ...)`.
- Replaced `cli := "shell"` + `if *shellFlag != "" { cli = *shellFlag }` with `const cli = "shell"`.
- Removed `errors` from the import block (the only two `errors.New` call sites were in the deleted branches).
- Rewrote the cmdNewShell doc comment to reflect the new argv shape (`agenthub new shell [<path>]`) and embedded a Phase 108 PARITY-CLI-01 rationale block. Updated the "Locked stderr error strings" list to keep only the still-valid entries (extra args after `--`, daemon unreachable).
- Left the `usage()` help block on `cmd_cli.go:29-30` intentionally unchanged — that line + README "Shell sessions" section are Plan 108-03's scope (per the executor prompt's explicit instruction).
- The `fs.Parse` error path (line 100 `fmt.Fprintf(os.Stderr, "agenthub new shell: %v\n", err)`) was NOT modified; it now re-emits the Go flag package's default `flag provided but not defined: -shell` message wrapped with the `agenthub new shell:` prefix, satisfying PARITY-CLI-01 acceptance.

**Verification:**
- `go build .` → exit 0 (clean).
- `grep -nE 'shellFlag|allowed\[.shell|unknown shell|--shell flag requires' cmd_cli.go` → 0 matches.
- SPEC-mandated `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` → 0 matches.

### Task 2 — Rewrite cmdNewShell tests — delete `--shell` tests, add removed-flag + Settings-resolution tests

**Files:** `cmd_cli_test.go`
**Commit:** `b9f9199` — `test(108-02): rewrite cmdNewShell tests for PARITY-CLI-01/02/03 contract`

**Deleted (7 tests, all asserting the old `--shell=X` contract):**

| Test | Was at line | Notes |
|------|-------------|-------|
| `TestCmdNewShell_FlagBash` | 744–756 | locked old `--shell=bash` happy path |
| `TestCmdNewShell_FlagZsh` | 758–770 | locked old `--shell=zsh` happy path |
| `TestCmdNewShell_FlagPwsh` | 772–789 | locked old `--shell=pwsh` happy path |
| `TestCmdNewShell_FlagPowerShell` | 791–805 | locked old `--shell=powershell` Windows path |
| `TestCmdNewShell_FlagAndPath` | 807–819 | flag + positional combo |
| `TestCmdNewShell_UnknownShellFlag` | 821–842 | locked stderr `unknown shell %q` |
| `TestCmdNewShell_EmptyShellFlag` | 844–860 | locked stderr `--shell flag requires a value` |

**Kept unchanged (4 tests, the post-collapse contract):**

- `TestCmdNewShell_NoArgs_UsesSystemDefault` — `assertShellCLI(..., "shell")` still passes because the matcher accepts any endorsed shell basename when the daemon resolves via `$SHELL`.
- `TestCmdNewShell_PositionalPath_UsesShell` — positional path with `cli="shell"`.
- `TestCmdNewShell_ExtraArgsWarning` — the `extra arguments are not forwarded` stderr warning is still emitted.
- `TestCmdNewShell_DaemonError` — daemon-unreachable wrapping still in place.

**Updated:**

- `TestUsage_IncludesNewShell` — dropped the `--shell=bash|zsh|pwsh|powershell` needle from the assertion slice. Kept `new shell [<path>]` (substring still present in `usage()` line 28 of cmd_cli.go). Plan 108-03 will further rewrite the help-block line, but the substring is preserved end-to-end.

**Added (3 new tests):**

- `TestCmdNewShell_RejectsRemovedFlag` (PARITY-CLI-01 acceptance) — green. Pattern: pass `--shell=zsh`, assert `callErr != nil`, assert `stderr` contains `flag provided but not defined: -shell`, assert `len(ListSessions()) == 0` (daemon not called when fs.Parse fails).
- `TestCmdNewShell_SettingsShellPathSpawned` (PARITY-CLI-02 acceptance) — green. Pattern: `client.SetShellPath("/bin/bash")`, call `cmdNewShell(client, nil, nil, &buf)`, assert `filepath.Base(sessions[0].CLI) == "bash"`. Skipped on hosts without `/bin/bash` (always present on macOS and most Linux distros). Exercises the end-to-end Settings-driven binary-resolution path through `client.SetShellPath` → daemon `SetShellPath` → `engine.shellPath` → `resolveShellSpawn` branch (0).
- `TestCmdNewShell_InvalidShellPathSilentFallback` (PARITY-CLI-03 acceptance) — **skipped with documented follow-up**. See "Acceptance test skip" section below.

**Verification:**
- `go test -count=1 -run 'CmdNewShell|Usage_IncludesNewShell' .` →
  ```
  PASS: TestCmdNewShell_NoArgs_UsesSystemDefault
  PASS: TestCmdNewShell_PositionalPath_UsesShell
  PASS: TestCmdNewShell_RejectsRemovedFlag
  PASS: TestCmdNewShell_SettingsShellPathSpawned
  SKIP: TestCmdNewShell_InvalidShellPathSilentFallback (documented)
  PASS: TestCmdNewShell_ExtraArgsWarning
  PASS: TestCmdNewShell_DaemonError
  PASS: TestUsage_IncludesNewShell
  ```
- `grep -nE 'TestCmdNewShell_FlagBash|TestCmdNewShell_FlagZsh|TestCmdNewShell_FlagPwsh|TestCmdNewShell_FlagPowerShell|TestCmdNewShell_FlagAndPath|TestCmdNewShell_UnknownShellFlag|TestCmdNewShell_EmptyShellFlag' cmd_cli_test.go` → 0 matches.

### Task 3 — Full-suite verification and SPEC grep gates

**Files:** verification only
**Commits:** `9a55f9e` (gofmt fix discovered during Task 3) — `style(108-02): gofmt cmd_cli.go doc-comment indentation`

**Verification results:**

1. `go test ./... -count=1` — 3 pre-existing failures in `internal/daemon/` (unrelated to this plan, see "Deferred issues" below). All other packages pass:
   ```
   ok  github.com/scottkw/agenthub                      9.244s
   ok  github.com/scottkw/agenthub/internal/attach
   ok  github.com/scottkw/agenthub/internal/capability
   FAIL github.com/scottkw/agenthub/internal/daemon     (3 pre-existing failures — see deferred-items.md)
   ok  github.com/scottkw/agenthub/internal/pty
   ok  github.com/scottkw/agenthub/internal/relay
   ok  github.com/scottkw/agenthub/internal/release
   ok  github.com/scottkw/agenthub/internal/status
   ok  github.com/scottkw/agenthub/internal/statusbar
   ok  github.com/scottkw/agenthub/internal/tailnet
   ok  github.com/scottkw/agenthub/internal/tui
   ok  github.com/scottkw/agenthub/internal/updater
   ok  github.com/scottkw/agenthub/internal/webserver
   ```
   Scoped run on this plan's package (`go test -count=1 .`) → PASS.

2. `go vet ./...` → exit 0 (clean).

3. `gofmt -l cmd_cli.go cmd_cli_test.go` → empty (after the style commit `9a55f9e` re-aligned the doc-comment bullets).

4. `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` → **0 matches** in live code (SPEC PARITY-CLI-01 grep gate).

5. **Manual smoke test:** `go build -o /tmp/agenthub-108 . && /tmp/agenthub-108 new shell --shell=zsh 2>&1; echo $?`
   ```
   agenthub new shell: flag provided but not defined: -shell
   flag provided but not defined: -shell
   1
   ```
   Exit code = 1, stderr contains `flag provided but not defined: -shell` ✓.

   (The duplicate line is pre-existing behavior: `cmdNewShell` line 100 prints the wrapped error, and `main.go` line 210's outer error handler `fmt.Fprintf(os.Stderr, "%v\n", err)` prints the unwrapped error a second time. This matches the pre-108-02 behavior — verified by checking out the parent commit `545dcc9` and running `agenthub new shell --shell=nope`, which produced the same two-line pattern with the old locked-stderr strings. Not a regression.)

## Acceptance test skip — PARITY-CLI-03

`TestCmdNewShell_InvalidShellPathSilentFallback` is skipped at the harness level because:

1. Both `client.SetShellPath(path)` (daemon-side validation at `internal/daemon/engine.go:689-707`) and a direct `engine.SetShellPath(path)` reject any path where `os.Stat` errors or the file is not executable. This is intentional hardening from Phase 107 SHELL-11.
2. The unexported `e.shellPath` field on `*SessionEngine` is in `internal/daemon` and is NOT reachable from the `main` package test in `cmd_cli_test.go`.
3. Plan 108-02 scope explicitly forbids modifying `internal/daemon/engine.go` to add a `SetShellPathForTest` export.

The plan explicitly allowed this skip pathway ("`t.Skip` the test with a clear comment, and note the limitation in SUMMARY.md"). The test body documents the proposed follow-up: add an unexported-field test setter in `internal/daemon/engine.go` mirroring the existing `(*API).SetWebServerForTest` pattern at `api.go:209-212`:

```go
// SetShellPathForTest directly assigns the shellPath override without
// validating executable-ness. Used only by tests that need to exercise
// daemon behavior when shellPath is deliberately broken.
func (e *SessionEngine) SetShellPathForTest(path string) {
    e.mu.Lock()
    e.shellPath = path
    e.mu.Unlock()
}
```

Filed under `.planning/phases/108-*/deferred-items.md`.

## Deviations from plan

### Auto-fixed issues

1. **[Rule 3 — Build-blocker] `security-review/` directory shadows `./...`**
   - Found during: Task 3 verification gates.
   - Issue: A gitignored `security-review/` directory at the repo root contains `internal_relay_protocol_fuzz_test.go` and `internal_webserver_server_test.go` (declaring packages `relay` and `webserver`). `go test ./...` and `go vet ./...` fail with "found packages relay (...) and webserver (...) in /Users/ken/dev/agenthub/security-review".
   - Fix: Temporarily moved the directory aside (`mv security-review /tmp/agenthub-security-review-108-02-backup`) for the verification runs, then restored it after Task 3 completed.
   - Files modified: none (local-only workaround).
   - Logged in `deferred-items.md` for follow-up.

2. **[Rule 1 — Bug, but pre-existing] 3 failures in `internal/daemon` about `GetShellWebShareWarned` default**
   - Found during: Task 3 `go test ./...`.
   - Issue: `TestAPIGetShellWebShareWarned_Default`, `TestDaemonClient_GetSetShellWebShareWarned_RoundTrip`, `TestSetShellWebShareWarned_Default` all fail with "default value: got true, want false".
   - Investigation: Confirmed pre-existing at the Phase 108 SPEC commit `e8adc15` (before any 108-* plan touched code), so NOT caused by 108-02 or 108-01. Failures are entirely in `internal/daemon/`, which 108-02 scope explicitly forbids modifying.
   - Action: Logged in `deferred-items.md` for separate-phase follow-up. Per SCOPE BOUNDARY, did NOT attempt to fix.

3. **[Style] gofmt re-indented the cmdNewShell doc-comment bullets**
   - Found during: Task 3 `gofmt -l cmd_cli.go cmd_cli_test.go`.
   - Issue: After Task 1, `gofmt -l cmd_cli.go` listed cmd_cli.go — the doc-comment bullets used space-prefix indentation; gofmt wants tab-prefix.
   - Fix: Ran `gofmt -w cmd_cli.go` and committed as `9a55f9e` (`style(108-02): gofmt cmd_cli.go doc-comment indentation`). Separate commit per protocol ("ALWAYS create NEW commits rather than amending").

4. **[Environment hygiene] Stale `git stash` entry from a prior unrelated session**
   - Found during: Task 3 verification (post `git stash pop` of an unrelated WIP-208ba5f stash that the executor inadvertently picked up).
   - Issue: An old stash from a prior session (`WIP on main: 208ba5f feat(77-03): create kill confirmation modal renderer`) was applied during my run, leaving conflict markers in `cmd_attach.go`, `internal/tui/update_test.go`, and `internal/tui/view_test.go`. `go vet` surfaced this as "missing import path" errors.
   - Fix: Reverted the three conflicted files to HEAD via `git checkout HEAD -- <file>`. Working tree returned to a clean state matching `cb0fa04`.
   - No files in 108-02's scope (`cmd_cli.go`, `cmd_cli_test.go`) were affected.

No deviations against the SPEC contract — `--shell` flag is fully removed, cmdNewShell unconditionally dispatches `cli="shell"`, no CLI-side validation of resolved shellPath, all SPEC grep gates return 0 matches.

## Hand-off note for Plan 108-03

Plan 108-03 (PARITY-DOCS-01) owns:

1. **`cmd_cli.go` `usage()` help block at lines 28-30:**
   - Current state (unchanged by 108-02):
     ```
     new shell [<path>]                          Create a new raw shell session
       --shell=bash|zsh|pwsh|powershell           Pick a specific shell (default: system default)
     ```
   - Target: remove the `--shell=bash|zsh|pwsh|powershell` modifier line entirely; collapse to a single `new shell [<path>]   Create a new raw shell session` line. The current `TestUsage_IncludesNewShell` only asserts the `new shell [<path>]` substring, which 108-03 should keep intact.

2. **`README.md` "Shell sessions" section (or equivalent):**
   - Add a paragraph that describes single-Shell entry on all three surfaces and points users to Settings → Paths for binary selection.
   - Remove any lingering references to per-surface `bash|zsh|pwsh|powershell` selection.

3. **Acceptance grep:** `grep -nE '\-\-shell=|bash\|zsh\|pwsh' cmd_cli.go README.md` should return 0 matches in user-facing strings (test fixtures excluded).

4. **Optional but recommended for 108-03:** lift the `TestCmdNewShell_InvalidShellPathSilentFallback` skip by adding `SetShellPathForTest` to `internal/daemon/engine.go` and updating `testSetup` to return the engine alongside the client. That's a small ~20-line change that closes the only currently-skipped SPEC acceptance gate. Out of 108-02 scope; suggested as a follow-up.

## Commits (this plan)

| Hash | Type | Subject |
|------|------|---------|
| `829144a` | refactor | remove --shell flag from cmdNewShell (PARITY-CLI-01/02/03) |
| `b9f9199` | test | rewrite cmdNewShell tests for PARITY-CLI-01/02/03 contract |
| `9a55f9e` | style | gofmt cmd_cli.go doc-comment indentation |

## Self-Check: PASSED

- File `/Users/ken/dev/agenthub/cmd_cli.go` — FOUND, contains const cli = "shell", no `shellFlag` or `allowed` map, no `errors` import.
- File `/Users/ken/dev/agenthub/cmd_cli_test.go` — FOUND, contains `TestCmdNewShell_RejectsRemovedFlag`, `TestCmdNewShell_SettingsShellPathSpawned`, `TestCmdNewShell_InvalidShellPathSilentFallback`; does NOT contain any of the 7 deleted test names.
- Commit `829144a` — FOUND in git log.
- Commit `b9f9199` — FOUND in git log.
- Commit `9a55f9e` — FOUND in git log.
- SPEC grep gate `grep -nE '"shell".*flag|shellFlag|allowed.*shell|unknown shell' cmd_cli.go` — 0 matches (exit 1).
- Manual smoke `agenthub new shell --shell=zsh` → exit 1 with stderr containing `flag provided but not defined: -shell`.
- `go test -count=1 -run 'CmdNewShell|Usage_IncludesNewShell' .` → all 7 PASS + 1 documented SKIP.
- `go vet .` → clean.
- `gofmt -l cmd_cli.go cmd_cli_test.go` → empty.
