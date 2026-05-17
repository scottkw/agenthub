---
phase: 100-shell-session-backend-discovery
plan: 01
subsystem: internal/pty
tags: [shell-discovery, SHELL-04, foundation, tdd]
requires:
  - none (foundation plan, no upstream dependencies)
provides:
  - api: "DiscoverShells() []DetectedShell"
  - api: "KnownShellSpecs() []ShellSpec"
  - api: "DetectShell(name string) (*DetectedShell, error)"
  - api: "var ErrShellNotFound"
  - api: "var testEtcShellsPath (production-side test hook)"
  - types: "ShellSpec, DetectedShell"
  - contract: "H4 — empty $SHELL never produces a synthetic 'shell' entry"
  - contract: "Non-nil empty slice from DiscoverShells when nothing on PATH"
affects:
  - "Plan 02 (engine argv resolution): consumes KnownShellSpecs + DetectShell"
  - "Plan 04 (GET /shells HTTP route): consumes DiscoverShells; depends on H4 contract"
tech-stack:
  added: []
  patterns:
    - "Mirror of detect.go known-list discovery shape (CLISpec/DetectedCLI -> ShellSpec/DetectedShell)"
    - "Production-side test-hook variable (testEtcShellsPath) declared in shells.go, assigned by shells_test.go — H1 single-source-of-truth"
    - "Two-pass discovery: Pass 1 PATH-based known specs, Pass 2 POSIX synthetic system-default ($SHELL)"
    - "Defensive Argv copy in DetectedShell results (callers cannot mutate package-level knownShellSpecs)"
key-files:
  created:
    - internal/pty/shells.go
    - internal/pty/shells_test.go
  modified: []
decisions:
  - "powershell as first-class knownShellSpecs entry (M2): keeps the table platform-agnostic; Plan 02's override branch resolves uniformly via Pass 1 / exec.LookPath (stdlib honors PATHEXT on Windows). On POSIX hosts the powershell entry is present in knownShellSpecs but will not surface in DiscoverShells results (exec.LookPath fails) — intentional, avoids build-tag fragmentation."
  - "testEtcShellsPath declared in shells.go (production side), not shells_test.go: H1 single source of truth. Test file assigns directly; no var shadow declaration."
  - "Empty $SHELL contract (H4): DiscoverShells returns early before any /etc/shells read — locked by TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry. Guards Plan 04's empty-PATH test against future fall-back-to-first-/etc/shells-entry refactors."
  - "/etc/shells is consulted ONLY to cross-check $SHELL validity (T-100-03 mitigation): we never surface /etc/shells entries directly as DetectedShell results; only knownShellSpecs basenames can surface. Read errors are silently skipped (Pitfall 2 — missing /etc/shells on slim containers must not be fatal)."
  - "Argv slices in DetectedShell are defensive copies via append([]string(nil), spec.Argv...) so callers cannot mutate the package-level table."
metrics:
  duration: "2min 22s"
  completed: "2026-05-12T23:37:11Z"
  tasks: 2
  files: 2
  commits: 2
---

# Phase 100 Plan 01: Shell Discovery Library Summary

Cross-platform shell-discovery library (`internal/pty/shells.go`) that mirrors `detect.go`'s known-list pattern and surfaces installed interactive shells (bash, zsh, pwsh, powershell) via PATH lookup, with optional POSIX synthetic system-default entry derived from `$SHELL` + `/etc/shells` cross-check.

## Exported API Surface

```go
// Types
type ShellSpec struct { Name, DisplayName string; Argv []string }
type DetectedShell struct {
    Name        string   `json:"name"`
    DisplayName string   `json:"displayName"`
    Path        string   `json:"path"`
    Argv        []string `json:"argv"`
}

// Errors
var ErrShellNotFound = errors.New("shell not found")

// Functions
func DiscoverShells() []DetectedShell                  // always non-nil
func DetectShell(name string) (*DetectedShell, error)  // ErrShellNotFound if missing
func KnownShellSpecs() []ShellSpec                     // package-level slice; callers must not mutate

// Production-side test hook (declared in shells.go, assigned by shells_test.go)
var testEtcShellsPath = ""
```

`ShellSpec` is unexported-shape (no JSON tags) because it is the internal Go API consumed by Plan 02's `resolveShellSpawn` argv-resolver. `DetectedShell` carries JSON tags for the wire-exposed `GET /shells` route in Plan 04.

## knownShellSpecs Canonical Contents

Order is load-bearing (asserted by `TestKnownShellSpecs_HasExpectedEntries` and consumed by Plan 02's first-match walk):

| # | Name         | DisplayName          | Argv          |
|---|--------------|----------------------|---------------|
| 1 | bash         | bash                 | `["-i"]`      |
| 2 | zsh          | zsh                  | `["-i"]`      |
| 3 | pwsh         | PowerShell           | `["-NoLogo"]` |
| 4 | powershell   | Windows PowerShell   | `["-NoLogo"]` |

**M2 note:** `powershell` is a first-class spec, not a runtime fallback. This keeps the table platform-agnostic and gives Plan 02's override branch (`cliPaths["powershell"] = "..."`) a clean match path via Pass 1 / `exec.LookPath` (stdlib honors PATHEXT on Windows). On POSIX hosts the `powershell` entry is in the table but never surfaces in `DiscoverShells` results because `exec.LookPath("powershell")` fails.

## /etc/shells Parsing Strategy

`/etc/shells` (or `testEtcShellsPath` when non-empty) is read ONLY to validate the `$SHELL` environment variable points at a real interactive shell on POSIX. The parser:

1. Returns an empty slice on any read error (silent skip — Pitfall 2 from RESEARCH.md: missing `/etc/shells` is common on slim Linux containers and must not be fatal).
2. Skips comment (`#`-prefix) and blank lines via `bufio.NewScanner` + `strings.TrimSpace`.
3. Returns each remaining line verbatim — no further validation.

`DiscoverShells` then cross-checks whether `$SHELL` appears verbatim in the slice:

- If `/etc/shells` is unreadable (empty slice returned), `$SHELL` validation falls back to basename-allowlist only (`isEndorsedShellBasename`).
- If `/etc/shells` is readable AND does not list `$SHELL`, no synthetic entry is appended.
- If `/etc/shells` is readable AND lists `$SHELL`, the synthetic entry is appended.

**T-100-03 mitigation (in-plan threat register):** Non-knownShellSpecs basenames listed in `/etc/shells` (e.g., `fish`, `tcsh`, `dash`) are never surfaced as `DetectedShell` results — they are not in `knownShellSpecs` and only appear in `DiscoverShells` Pass 1 via `exec.LookPath` lookup of `knownShellSpecs` entries. The `/etc/shells` parser's output is consumed exclusively by the synthetic-default path, and that path checks `isEndorsedShellBasename` on `$SHELL` before any allocation.

## Test Hook Declaration Site

Per Plan 01 H1 (single source of truth):

- **Declaration:** `internal/pty/shells.go` line 56:
  ```go
  var testEtcShellsPath = ""
  ```
  Declared unconditionally (NOT under a `_test.go` build constraint) so the test file can assign to it without a shadow `var` declaration that would cause a duplicate-declaration build error.

- **Assignment:** `internal/pty/shells_test.go` — six call sites across three tests (`TestDiscoverShells_NoEtcShells`, `TestDiscoverShells_EtcShellsFixture`, `TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry`), each with a `t.Cleanup(func() { testEtcShellsPath = "" })` to restore production state after the test. No `var testEtcShellsPath` declaration anywhere in the test file (verified by grep gate in acceptance criteria).

## Empty-$SHELL Contract (H4)

`DiscoverShells` returns early before any `/etc/shells` read when `$SHELL` is empty. Locked by `TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry` which:

1. Writes a fixture `/etc/shells` containing `/bin/bash\n`.
2. Sets `$SHELL = ""` and `$PATH` to an empty temp dir (no known specs discoverable).
3. Asserts NO entry in the result has `Name == "shell"`.

**Why this matters:** Plan 04's `TestHandleListShells_EmptyPATH` test depends on this contract. Without the early return, a future refactor that "falls back to the first `/etc/shells` entry when `$SHELL` is unset" would silently break Plan 04's empty-PATH test in non-obvious ways.

## Deviations from Plan

None — plan executed exactly as written. Two minor execution notes:

1. **Task 1 RED verify (`go build`):** The plan's automated verify command was `go build ./internal/pty/...` expecting `undefined:` errors. `go build` does NOT compile `_test.go` files, so the build appeared clean. Substituted `go test ./internal/pty/... -run TestDiscoverShells -count=1` (which compiles test files) and verified the expected `undefined: DiscoverShells / knownShellSpecs / DetectedShell / testEtcShellsPath` symbol errors — RED state confirmed. This is a build-tooling fact, not a plan deviation; the spirit of the acceptance criterion ("build correctly fails with `undefined:` error for the not-yet-implemented API") is satisfied.

2. **`go fmt ./internal/pty/...` side-effect:** Running `go fmt` on the entire package reformatted an unrelated file (`win32input_parse.go`) due to a stale doc-comment indentation pre-dating this plan. Reverted that change (out of scope per scope-boundary rule in `<deviation_rules>`) and confirmed `gofmt -l internal/pty/shells.go internal/pty/shells_test.go` is clean on the new files. Logged as a pre-existing item rather than a Rule 1 fix.

## Verification Results

```
$ go test ./internal/pty -run Shell -race -count=1 -v
=== RUN   TestDiscoverShells_FindsInstalledShells   PASS (0.00s)
=== RUN   TestDiscoverShells_SkipsMissing           PASS (0.00s)
=== RUN   TestDiscoverShells_AllMissing             PASS (0.00s)
=== RUN   TestDiscoverShells_NoEtcShells            PASS (0.00s)
=== RUN   TestDiscoverShells_EtcShellsFixture       PASS (0.00s)
=== RUN   TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry PASS (0.00s)
=== RUN   TestDiscoverShells_Windows                SKIP (Windows-only test)
=== RUN   TestDiscoverShells_WindowsPowerShell      SKIP (Windows-only test)
=== RUN   TestKnownShellSpecs_HasExpectedEntries    PASS (0.00s)
PASS
ok      github.com/scottkw/agenthub/internal/pty        1.025s
```

7 POSIX tests pass under `-race`; 2 Windows tests correctly skip on the macOS dev box. `go vet ./internal/pty/...` and `gofmt -l internal/pty/shells.go internal/pty/shells_test.go` are both clean.

## Commits

| # | Hash    | Subject                                                                   |
|---|---------|---------------------------------------------------------------------------|
| 1 | af4c271 | test(100-01): add failing tests for DiscoverShells (RED)                  |
| 2 | 691b635 | feat(100-01): implement cross-platform shell discovery (GREEN)            |

## Self-Check: PASSED

- `internal/pty/shells.go`: FOUND
- `internal/pty/shells_test.go`: FOUND
- Commit af4c271: FOUND
- Commit 691b635: FOUND
- `go test ./internal/pty -run Shell -race -count=1`: exit 0
- `go vet ./internal/pty/...`: exit 0
- `gofmt -l` on both files: empty
- `var testEtcShellsPath = ""` in shells.go: 1 occurrence
- `var testEtcShellsPath` declarations in shells_test.go: 0 (no shadow)
- All 4 canonical knownShellSpecs names present: bash, zsh, pwsh, powershell
- All 9 test functions named verbatim per VALIDATION.md

## TDD Gate Compliance

Plan type: `execute` (not `tdd` at plan-level), but both tasks carry `tdd="true"`.

| Gate     | Commit  | Subject                                                          |
|----------|---------|------------------------------------------------------------------|
| RED      | af4c271 | `test(100-01): add failing tests for DiscoverShells (RED)`       |
| GREEN    | 691b635 | `feat(100-01): implement cross-platform shell discovery (GREEN)` |
| REFACTOR | —       | (skipped — implementation was clean on first pass, no rework)    |

RED + GREEN gate sequence verified in `git log --oneline -3`.
