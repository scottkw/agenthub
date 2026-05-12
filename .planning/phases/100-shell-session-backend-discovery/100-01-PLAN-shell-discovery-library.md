---
phase: 100-shell-session-backend-discovery
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/pty/shells.go
  - internal/pty/shells_test.go
autonomous: true
requirements:
  - SHELL-04
user_setup: []

must_haves:
  truths:
    - "DiscoverShells() returns bash + zsh entries when both binaries are on PATH (POSIX)"
    - "DiscoverShells() returns a non-nil empty slice when no known shells are on PATH"
    - "DiscoverShells() resolves pwsh.exe via PATHEXT on Windows (without manual .exe suffixing)"
    - "DiscoverShells() silently skips missing /etc/shells on POSIX (no panic, no error leak)"
    - "Empty /etc/shells does not cause DiscoverShells to drop knownShellSpecs entries"
    - "knownShellSpecs contains exactly bash, zsh, pwsh, powershell (canonical order)"
    - "POSIX DiscoverShells() with empty $SHELL never appends a synthetic 'shell' entry, regardless of /etc/shells contents"
    - "Windows DiscoverShells() includes 'powershell' entry when powershell.exe is on PATH (cli='powershell' override resolvable)"
  artifacts:
    - path: internal/pty/shells.go
      provides: "ShellSpec/DetectedShell types, knownShellSpecs table, DiscoverShells(), DetectShell(name), ErrShellNotFound, KnownShellSpecs() exported accessor, testEtcShellsPath production-side test hook (empty in prod)"
      contains: "func DiscoverShells() []DetectedShell"
      min_lines: 80
    - path: internal/pty/shells_test.go
      provides: "Table-driven discovery tests with PATH + /etc/shells mocking"
      contains: "func TestDiscoverShells_FindsInstalledShells"
      min_lines: 100
  key_links:
    - from: internal/pty/shells.go
      to: os/exec
      via: "exec.LookPath for each ShellSpec.Name"
      pattern: "exec\\.LookPath\\("
    - from: internal/pty/shells.go
      to: runtime
      via: "GOOS branching for Windows fallback + POSIX /etc/shells"
      pattern: "runtime\\.GOOS"
    - from: internal/pty/shells_test.go
      to: internal/pty/shells.go
      via: "calls DiscoverShells() under mocked PATH; assigns to production-side testEtcShellsPath"
      pattern: "DiscoverShells\\(\\)"
---

<objective>
Create `internal/pty/shells.go` — a cross-platform shell-discovery library that mirrors the existing `internal/pty/detect.go` AI-CLI discovery shape. Add table-driven tests in `internal/pty/shells_test.go` covering PATH-based discovery, missing-binary skip, non-nil-empty-slice guarantee, `/etc/shells` supplementation (POSIX), and Windows `pwsh.exe`/`powershell.exe` fallback.

Purpose: This library is the foundation for SHELL-04 (daemon discovers available shells per platform). Plan 02 (engine argv resolution) and Plan 04 (`GET /shells` HTTP route) both consume `DiscoverShells()` and `KnownShellSpecs()` exports.

Output: Two new files (`shells.go`, `shells_test.go`) with no changes to any existing file. Discovery is pure: no side effects beyond filesystem reads via `exec.LookPath` and optional `/etc/shells` read.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md
@.planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md
@.planning/phases/100-shell-session-backend-discovery/100-VALIDATION.md
@internal/pty/detect.go
@internal/pty/detect_test.go

<interfaces>
<!-- The exact analog to mirror. Source: internal/pty/detect.go (verified 68 lines). -->
<!-- Executor must mirror this file's shape — package layout, error sentinel, non-nil empty slice -->
<!-- contract, GOOS branching idiom (from native.go:62). -->

From internal/pty/detect.go:
```go
package pty

import (
    "errors"
    "os/exec"
)

type CLISpec struct { Name string; DisplayName string }
type DetectedCLI struct { Name string; DisplayName string; Path string }

var knownCLIs = []CLISpec{
    {Name: "claude", DisplayName: "Claude Code"},
    {Name: "codex",  DisplayName: "OpenAI Codex"},
    {Name: "gemini", DisplayName: "Gemini CLI"},
    {Name: "opencode", DisplayName: "OpenCode"},
}

func DetectCLIs() []DetectedCLI {
    result := make([]DetectedCLI, 0)  // CRITICAL: non-nil empty slice
    for _, spec := range knownCLIs {
        path, err := exec.LookPath(spec.Name)
        if err != nil { continue }
        result = append(result, DetectedCLI{Name: spec.Name, DisplayName: spec.DisplayName, Path: path})
    }
    return result
}

var ErrCLINotFound = errors.New("CLI not found")

func DetectCLI(name string) (*DetectedCLI, error) {
    for _, spec := range knownCLIs {
        if spec.Name != name { continue }
        path, err := exec.LookPath(spec.Name)
        if err != nil { return nil, ErrCLINotFound }
        return &DetectedCLI{Name: spec.Name, DisplayName: spec.DisplayName, Path: path}, nil
    }
    return nil, ErrCLINotFound
}
```

From internal/pty/detect_test.go (the test pattern to mirror):
```go
func TestDetectCLIs_FindsInstalledCLIs(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("uses shell script stubs not executable on Windows")
    }
    dir := t.TempDir()
    stubPath := filepath.Join(dir, "claude")
    os.WriteFile(stubPath, []byte("#!/bin/sh\necho ok\n"), 0755)
    t.Setenv("PATH", dir)
    result := DetectCLIs()
    // assert claude is in result with non-empty Path
}
```
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Create shells_test.go (RED — tests fail because shells.go doesn't exist yet)</name>
  <files>internal/pty/shells_test.go</files>
  <read_first>
    - internal/pty/detect_test.go (MUST mirror exact shape: package, imports, fixture style, assertions)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md § "internal/pty/shells_test.go (NEW — test)"
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md § "Code Examples" → "Pattern: PATH-mocked discovery test"
    - .planning/phases/100-shell-session-backend-discovery/100-VALIDATION.md § "Per-Task Verification Map" (test names to use verbatim)
  </read_first>
  <behavior>
    Test functions to create (test names must match exactly — Plan 02/04 grep on them via -run filter):

    1. TestDiscoverShells_FindsInstalledShells (POSIX-only via runtime.GOOS skip):
       - t.TempDir() as PATH dir; write 0755 stubs named "bash" and "zsh" with content "#!/bin/sh\necho ok\n"
       - t.Setenv("PATH", dir)
       - Call DiscoverShells(); assert result contains entries with Name="bash" and Name="zsh", each with non-empty Path

    2. TestDiscoverShells_SkipsMissing:
       - t.Setenv("PATH", t.TempDir()) (empty dir)
       - On POSIX, also t.Setenv("SHELL", "") and override /etc/shells lookup by ensuring no fallback adds entries
       - Assert result has zero knownShellSpecs entries (bash/zsh/pwsh/powershell) — but allow synthetic "system default" entry if it cannot be suppressed; test should filter to known-spec names only

    3. TestDiscoverShells_AllMissing (mirrors detect_test.go:57-69):
       - Same setup as SkipsMissing
       - Assert result != nil (non-nil empty slice guarantee — per Pitfall 2 in RESEARCH.md, load-bearing for slim Linux containers)
       - Assert len(filtered to known specs) == 0

    4. TestKnownShellSpecs_HasExpectedEntries (mirrors detect_test.go:73-90):
       - Build expected := []string{"bash", "zsh", "pwsh", "powershell"}
       - Assert len(knownShellSpecs) == len(expected)
       - Assert every expected name is present in knownShellSpecs

    5. TestDiscoverShells_NoEtcShells (POSIX-only):
       - t.TempDir() PATH with stub bash only
       - Use the production-side test hook (testEtcShellsPath, declared in shells.go) to point /etc/shells reader at a non-existent path:
         ```go
         testEtcShellsPath = filepath.Join(t.TempDir(), "does-not-exist")
         t.Cleanup(func() { testEtcShellsPath = "" })
         ```
       - Assert DiscoverShells does not panic and returns at least the bash entry

    6. TestDiscoverShells_EtcShellsFixture (POSIX-only):
       - Write a fixture /etc/shells-style file at t.TempDir()/etc-shells with content:
         "# comments\n/bin/bash\n/bin/zsh\n/usr/local/bin/fish\n"
       - Assign the hook: `testEtcShellsPath = <fixture path>` with `t.Cleanup(func(){ testEtcShellsPath = "" })`
       - Setenv SHELL=/bin/zsh, stub /bin/zsh on PATH
       - Assert result includes a synthetic "shell" entry (Name="shell") with Path pointing to the zsh stub (system default resolution)
       - Assert result does NOT include "fish" as a selectable spec (out-of-scope per REQUIREMENTS.md line 87)

    7. TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry (POSIX-only — H4 contract lock-in):
       - This test locks the contract that Plan 04's TestHandleListShells_EmptyPATH depends on:
         when $SHELL is empty, no synthetic "shell" entry is ever appended, regardless of /etc/shells contents.
       - Skip on Windows.
       - Write a fixture /etc/shells containing "/bin/bash\n" at t.TempDir()/etc-shells.
       - `testEtcShellsPath = <fixture path>`; `t.Cleanup(func(){ testEtcShellsPath = "" })`.
       - `t.Setenv("SHELL", "")` (empty SHELL env).
       - `t.Setenv("PATH", t.TempDir())` (empty PATH dir — no knownShellSpecs binaries discoverable).
       - Call DiscoverShells().
       - Assert that NO entry in the result has Name == "shell" (regardless of whether real shells were discovered).
       - This guards against future refactors that might "fall back to first /etc/shells entry when SHELL is unset" — such a change would silently break Plan 04's empty-PATH test in non-obvious ways.

    8. TestDiscoverShells_Windows (Windows-only — gate with `if runtime.GOOS != "windows" { t.Skip(...) }`):
       - This test is allowed to be a smoke-level assertion only (relies on the Windows CI runner having pwsh.exe installed by default)
       - Assert DiscoverShells() result is non-nil; if a "pwsh" entry exists, Path must end with "pwsh.exe" (case-insensitive)

    9. TestDiscoverShells_WindowsPowerShell (Windows-only — M2 lock-in for `powershell` discoverability):
       - Skip on non-Windows.
       - Smoke-level: if `exec.LookPath("powershell.exe")` succeeds, assert DiscoverShells() result contains at least one entry whose Name is either "powershell" (canonical knownShellSpec name) or "pwsh" (acceptable when pwsh.exe is also present and the Windows fallback short-circuits).
       - The point is to lock the contract that on a Windows host with powershell.exe installed but pwsh.exe absent, the override branch in Plan 02 (`cliPaths["powershell"] = "..."`) can resolve to a valid spec via knownShellSpecs.

    Notes:
    - Use t.Setenv (auto-restore) for PATH and SHELL.
    - Stub files must be mode 0755 (executable bit honored by exec.LookPath).
    - Skip POSIX-stub tests on Windows with the message "uses shell script stubs not executable on Windows" (verbatim from detect_test.go:13).
    - `testEtcShellsPath` is declared in shells.go (Task 2). The test file assigns to it directly — NO `var testEtcShellsPath` declaration in shells_test.go.
  </behavior>
  <action>
    Create `internal/pty/shells_test.go` with the nine test functions described in <behavior>. Mirror `internal/pty/detect_test.go` exactly for package declaration, import order (os, path/filepath, runtime, testing), fixture pattern (t.TempDir + 0755 stub + t.Setenv), and non-nil-slice assertion idiom.

    Do NOT yet implement `shells.go` — these tests MUST fail with an "undefined" compilation error at this point (RED phase). The compilation failure IS the proof tests are wired correctly.

    For tests 5/6/7 (/etc/shells path injection), the test file ASSIGNS to the production-side `testEtcShellsPath` variable that will be declared in shells.go (Task 2):
    ```go
    // In each test that needs the hook:
    testEtcShellsPath = filepath.Join(dir, "etc-shells")
    t.Cleanup(func() { testEtcShellsPath = "" })
    ```
    Do NOT declare `var testEtcShellsPath` in shells_test.go — the production-side declaration in shells.go (Task 2) is the single source of truth. Declaring it here would cause a duplicate-declaration build error once Task 2 lands.

    During the RED phase (Task 1 only), the test file will fail to build because shells.go does not yet exist — so `testEtcShellsPath` is also undefined. That is expected and confirms the hook is wired correctly. Task 2 introduces the declaration and turns the build green.

    Filter helper in tests:
    ```go
    func filterKnownSpecs(in []DetectedShell) []DetectedShell {
        known := map[string]bool{"bash": true, "zsh": true, "pwsh": true, "powershell": true}
        out := []DetectedShell{}
        for _, s := range in { if known[s.Name] { out = append(out, s) } }
        return out
    }
    ```
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go build ./internal/pty/... 2>&1 | grep -qE 'undefined:[[:space:]]*(pty\.)?(DiscoverShells|knownShellSpecs|DetectedShell|ShellSpec|testEtcShellsPath)' && echo "RED-as-expected (build fails with undefined-symbol error for the not-yet-implemented API)" || (echo "FAIL: build did not fail with the expected undefined-symbol errors. Build output:"; go build ./internal/pty/... 2>&1; exit 1)</automated>
  </verify>
  <acceptance_criteria>
    - File internal/pty/shells_test.go exists
    - File contains func TestDiscoverShells_FindsInstalledShells
    - File contains func TestDiscoverShells_SkipsMissing
    - File contains func TestDiscoverShells_AllMissing
    - File contains func TestKnownShellSpecs_HasExpectedEntries
    - File contains func TestDiscoverShells_NoEtcShells
    - File contains func TestDiscoverShells_EtcShellsFixture
    - File contains func TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry
    - File contains func TestDiscoverShells_Windows
    - File contains func TestDiscoverShells_WindowsPowerShell
    - File contains the string `testEtcShellsPath = ` (assignment, NOT a declaration — production-side declaration lives in shells.go per H1)
    - File does NOT contain `var testEtcShellsPath` (no shadow declaration in the test file)
    - `go build ./internal/pty/...` fails with an error matching the regex `undefined:[[:space:]]*(pty\.)?(DiscoverShells|knownShellSpecs|DetectedShell|ShellSpec|testEtcShellsPath)` (proves tests reference the not-yet-implemented API via a real undefined-symbol error, not just any output containing those substrings)
  </acceptance_criteria>
  <done>shells_test.go committed with nine test functions; build correctly fails with an `undefined:` error for DiscoverShells / knownShellSpecs / testEtcShellsPath, proving the RED state is wired to actual symbol references rather than incidental string matches.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement shells.go (GREEN — all Plan 01 tests pass)</name>
  <files>internal/pty/shells.go</files>
  <read_first>
    - internal/pty/detect.go (the exact analog to mirror)
    - internal/pty/native.go lines 60-69 (runtime.GOOS branching pattern)
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md § "Pattern 1: Mirror detect.go's known-list pattern" + "Pitfall 2: /etc/shells missing"
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md § "internal/pty/shells.go (NEW — utility, discovery)"
    - internal/pty/shells_test.go (just-created, to see what symbols are required)
  </read_first>
  <behavior>
    After this task, every test from Task 1 passes. Specifically:
    - DiscoverShells returns non-nil empty slice when nothing is on PATH and /etc/shells is empty/missing
    - bash and zsh stubs on PATH are discovered (Name + Path populated, Argv = ["-i"])
    - knownShellSpecs has exactly 4 entries in canonical order: bash, zsh, pwsh, powershell
    - On Windows, when powershell.exe is on PATH, the "powershell" entry from knownShellSpecs surfaces with Argv = ["-NoLogo"]; on POSIX, the "powershell" spec is still present in the table but exec.LookPath("powershell") will typically fail so no entry surfaces in DetectedShell results
    - On POSIX: when SHELL env var points to a discovered shell binary, a synthetic entry with Name="shell" (DisplayName="system default") is appended with the SHELL value's argv inferred from its basename (zsh -> ["-i"], bash -> ["-i"], else ["-i"] as conservative default)
    - On POSIX: when SHELL env var is empty, NO synthetic "shell" entry is appended, regardless of /etc/shells contents (TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry locks this)
    - /etc/shells (or testEtcShellsPath if set) is read only to sanity-check that $SHELL is a known interactive shell; if /etc/shells is unreadable, system-default resolution falls back to checking $SHELL basename against knownShellSpecs directly
    - No /etc/shells parsing surfaces non-bash/non-zsh/non-pwsh/non-powershell entries (e.g., dash, fish, tcsh) as DetectedShell results — out-of-scope per REQUIREMENTS.md line 87
  </behavior>
  <action>
    Create `internal/pty/shells.go` implementing:

    Package & imports:
    ```go
    package pty

    import (
        "bufio"
        "errors"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strings"
    )
    ```

    Types:
    - `ShellSpec struct { Name string; DisplayName string; Argv []string }` — no JSON tags (internal Go API).
    - `DetectedShell struct { Name string; DisplayName string; Path string; Argv []string }` with JSON tags `name`, `displayName`, `path`, `argv` (wire-exposed via Plan 04).

    Package-level table (canonical order — TestKnownShellSpecs_HasExpectedEntries enforces this). Per M2: `powershell` is a first-class spec, not a runtime fallback. This gives Plan 02's `resolveShellSpawn` override branch a clean match path when callers set `cliPaths["powershell"]`:
    ```go
    var knownShellSpecs = []ShellSpec{
        {Name: "bash",       DisplayName: "bash",                Argv: []string{"-i"}},
        {Name: "zsh",        DisplayName: "zsh",                 Argv: []string{"-i"}},
        {Name: "pwsh",       DisplayName: "PowerShell",          Argv: []string{"-NoLogo"}},
        {Name: "powershell", DisplayName: "Windows PowerShell",  Argv: []string{"-NoLogo"}},
    }
    ```

    Note on `powershell` on POSIX: the spec is present in the table on all platforms (single source of truth), but exec.LookPath("powershell") will almost always fail on POSIX hosts, so no entry surfaces in DetectedShell results. This is intentional — keeping the table platform-agnostic avoids build-tag fragmentation. The Windows discovery test (TestDiscoverShells_WindowsPowerShell) validates the Windows-side behavior.

    Production-side test hook (per H1 — single source of truth, declared in shells.go, assigned by shells_test.go):
    ```go
    // testEtcShellsPath, if non-empty, overrides the /etc/shells path read by
    // DiscoverShells. Empty string in production builds (the var is declared
    // unconditionally in this file, NOT under a _test.go build constraint, so
    // shells_test.go can assign to it without a shadow declaration).
    // See Plan 01 H1 fix: declaration lives here, assignment lives in tests.
    var testEtcShellsPath = ""
    ```

    Exported accessors (Plan 02 consumes these):
    - `func KnownShellSpecs() []ShellSpec { return knownShellSpecs }` — returns the package-level slice directly (callers must not mutate; documented in doc-comment).
    - `var ErrShellNotFound = errors.New("shell not found")`
    - `func DetectShell(name string) (*DetectedShell, error)` — mirror DetectCLI shape from detect.go:50-68; loop knownShellSpecs, exec.LookPath, return constructed DetectedShell or ErrShellNotFound.

    Main function `func DiscoverShells() []DetectedShell` with two passes plus optional POSIX synthetic-default:
    1. Pass 1 — known specs via PATH (mirror detect.go:32-48):
       ```go
       result := make([]DetectedShell, 0)  // non-nil; load-bearing per Pitfall 2
       for _, spec := range knownShellSpecs {
           path, err := exec.LookPath(spec.Name)
           if err != nil { continue }
           result = append(result, DetectedShell{Name: spec.Name, DisplayName: spec.DisplayName, Path: path, Argv: append([]string(nil), spec.Argv...)})
       }
       ```
       Note: because `powershell` is now in knownShellSpecs (M2), Pass 1 handles it uniformly. No separate Windows fallback pass is needed — exec.LookPath honors PATHEXT on Windows and will resolve `powershell` -> `powershell.exe`.

    2. Pass 2 — POSIX system-default (gated by `runtime.GOOS != "windows"`):
       - Read SHELL env var. **If empty, skip entirely — do NOT append any synthetic entry** (this is the H4 contract: TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry locks this).
       - Check whether $SHELL value is "endorsed": its basename must be one of "bash", "zsh" (pwsh on POSIX is fine but unusual). Implement via `func isEndorsedShellBasename(p string) bool` — check `filepath.Base(p)` against allowlist.
       - Cross-check against /etc/shells (or testEtcShellsPath if non-empty) using `readEtcShells(path string) []string` helper that:
         * Opens the file; returns empty slice on error (silent skip per Pitfall 2)
         * Scans with bufio.NewScanner, trims whitespace, skips lines starting with "#" or empty
       - If $SHELL is endorsed AND (etc-shells slice is empty OR contains the exact $SHELL value), append synthetic entry: `{Name: "shell", DisplayName: "system default", Path: <SHELL>, Argv: argvForShellBasename(filepath.Base(SHELL))}` where `argvForShellBasename` returns ["-i"] for bash/zsh, ["-NoLogo"] for pwsh/powershell, ["-i"] default.

    Test hook integration:
    - `readEtcShells` checks the package-level variable `testEtcShellsPath` — if non-empty, use that path; otherwise use "/etc/shells".

    Argv defensiveness: `Argv` slices in DetectedShell results are copies (use `append([]string(nil), spec.Argv...)`) so callers cannot mutate the package-level table.

    Run `go fmt ./internal/pty/...` and `go vet ./internal/pty/...` before commit.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && go test ./internal/pty -run TestDiscoverShells -race -count=1 -v && go test ./internal/pty -run TestKnownShellSpecs -race -count=1 -v</automated>
  </verify>
  <acceptance_criteria>
    - File internal/pty/shells.go exists
    - File contains `func DiscoverShells() []DetectedShell`
    - File contains `func KnownShellSpecs() []ShellSpec`
    - File contains `func DetectShell(name string) (*DetectedShell, error)`
    - File contains `var ErrShellNotFound`
    - File contains `var knownShellSpecs` with exactly 4 entries (bash, zsh, pwsh, powershell) in canonical order — verify via grep that all four names appear in the var block (e.g., `grep -c 'Name: "bash"\|Name: "zsh"\|Name: "pwsh"\|Name: "powershell"' internal/pty/shells.go` returns >= 4 lines)
    - File contains `var testEtcShellsPath = ""` (production-side declaration, the single source of truth per H1)
    - File contains `Name: "powershell"` (M2: powershell is a first-class spec, not a runtime fallback)
    - `grep -v '^[[:space:]]*//' internal/pty/shells.go | grep -c "knownShellSpecs"` returns >= 3 (declaration + at least 2 usages, comment lines excluded)
    - `go test ./internal/pty -run TestDiscoverShells -race -count=1` exits 0 (all 7 POSIX-applicable Discover tests pass; Windows-tagged tests skip on macOS)
    - `go test ./internal/pty -run TestKnownShellSpecs -race -count=1` exits 0
    - `go test ./internal/pty -run TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry -race -count=1` exits 0 (H4 contract)
    - `go vet ./internal/pty/...` exits 0
    - `gofmt -l internal/pty/shells.go` produces no output (file is gofmt-clean)
  </acceptance_criteria>
  <done>shells.go and shells_test.go committed. All 9 test functions pass under -race (Windows-tagged tests skip on macOS dev box). `DiscoverShells`, `KnownShellSpecs`, `DetectShell`, `ErrShellNotFound`, `knownShellSpecs`, and `testEtcShellsPath` are accessible for Plan 02 and Plan 04. The H4 contract (empty $SHELL -> no synthetic entry) is locked by a dedicated test that Plan 04 depends on.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| PATH env var → exec.LookPath | An attacker who can write to a directory earlier on PATH than `/bin` can replace `bash`/`zsh`/`pwsh`/`powershell` with a malicious binary. This boundary exists for ALL discovery, not just shells (existing in detect.go). |
| /etc/shells → DiscoverShells | A user who can write to /etc/shells (root-only on stock POSIX) could attempt to list arbitrary binaries. Mitigation in this plan: /etc/shells is consulted ONLY to validate $SHELL is a real interactive shell; we never spawn a binary based on /etc/shells content alone — only knownShellSpecs (basename allowlist) are surfaced. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-100-01a | Tampering / Elevation | `exec.LookPath` in DiscoverShells | accept | Inherited PATH-based discovery risk; identical to existing detect.go discovery surface. Daemon process privilege == discovered-binary process privilege. Documented in PROJECT.md ("daemon API access implies host trust"). No new attack surface beyond existing CLI discovery. |
| T-100-03 | Tampering / Information Disclosure | `/etc/shells` parser | mitigate | Per RESEARCH.md T-100-03: `/etc/shells` cross-checked against `knownShellSpecs` basename allowlist (`bash`, `zsh`, `pwsh`, `powershell`). Unknown basenames listed in `/etc/shells` are silently skipped — never surfaced as `DetectedShell` entries and never spawned. `readEtcShells` returns empty slice on file read error (silent skip per Pitfall 2). |
| T-100-04a | Spoofing | Windows pwsh.exe / powershell.exe resolution via PATHEXT | mitigate | Per RESEARCH.md T-100-04: prefer absolute path resolution via `exec.LookPath` (stdlib handles PATHEXT correctly). With M2's change, both `pwsh` and `powershell` are first-class knownShellSpecs entries and resolve uniformly via Pass 1. Plan 03 adds well-known install dirs (`C:\Program Files\PowerShell\7`) to PATH augmentation — these dirs are administrator-writable only on standard Windows installs. User-writable PATH entries earlier in PATH could still spoof; this matches existing CLI-discovery threat posture. Documented assumption: daemon process privilege == discovered-binary privilege. |
| T-100-S05 | Denial of Service | DiscoverShells called per-request | accept | Discovery cost: ~4 `exec.LookPath` calls + 1 small file read. Bounded, no recursion, no network. Plan 04 may add a cache if profiling shows hot-path cost. |
</threat_model>

<verification>
After both tasks complete, the following must hold:

```bash
# Quick gate (per phase Validation map sampling rate):
go test ./internal/pty/... -run Shell -race -count=1

# Vet + format gates:
go vet ./internal/pty/...
gofmt -l internal/pty/shells.go internal/pty/shells_test.go  # must be empty

# Pattern preservation check (filter out comments — header prose contains the token):
grep -c "make(\[\]DetectedShell, 0)" internal/pty/shells.go               # >=1 (non-nil empty slice)
grep -v '^[[:space:]]*//' internal/pty/shells.go | grep -c "knownShellSpecs"  # >=3 (decl + usages)
grep -c 'Name: "powershell"' internal/pty/shells.go                      # >=1 (M2: powershell in knownShellSpecs)

# No spurious imports:
goimports -l internal/pty/shells.go  # must be empty
```

All nine test functions from Task 1 pass after Task 2's implementation. Validation map TBD-04-discover-found, TBD-04-discover-missing, TBD-04-etcshells-fixture, TBD-04-etcshells-missing, TBD-04-empty-shell-env, TBD-04-windows-pwsh, TBD-04-windows-powershell map to these tests (final task IDs assigned by this plan).
</verification>

<success_criteria>
- `internal/pty/shells.go` exists with `DiscoverShells`, `KnownShellSpecs`, `DetectShell`, `ErrShellNotFound` exports
- `internal/pty/shells.go` contains `var testEtcShellsPath = ""` (production-side declaration per H1)
- `internal/pty/shells.go` `knownShellSpecs` contains 4 entries (bash, zsh, pwsh, powershell) per M2
- `internal/pty/shells_test.go` exists with all nine test functions named verbatim per VALIDATION.md
- `internal/pty/shells_test.go` does NOT contain `var testEtcShellsPath` (no shadow declaration per H1)
- `go test ./internal/pty -run Shell -race -count=1` exits 0
- `go vet ./internal/pty/...` and `gofmt -l` are clean
- No changes to any file outside `internal/pty/` (zero file overlap with Plan 02/03)
- SHELL-04 requirement: daemon-side discovery library is callable; HTTP exposure ships in Plan 04
</success_criteria>

<output>
After completion, create `.planning/phases/100-shell-session-backend-discovery/100-01-SUMMARY.md` documenting:
- Exported API surface (signatures of DiscoverShells, KnownShellSpecs, DetectShell, ShellSpec, DetectedShell, ErrShellNotFound)
- knownShellSpecs canonical contents (bash, zsh, pwsh, powershell — note M2's promotion of powershell to first-class spec)
- /etc/shells parsing strategy (read-only for $SHELL validation; allowlist-filtered)
- Test hook (testEtcShellsPath) declaration site (shells.go — production-side, single source of truth per H1)
- Empty-$SHELL contract (no synthetic "shell" entry — locked by TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry per H4)
- Any deviations from RESEARCH.md (e.g., if /etc/shells handling differs) with rationale
</output>
</content>
</invoke>