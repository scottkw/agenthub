---
phase: 100-shell-session-backend-discovery
plan: 03
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/daemon/path_windows.go
  - internal/daemon/path_windows_test.go
autonomous: true
requirements:
  - SHELL-04
user_setup: []

must_haves:
  truths:
    - "Windows daemon PATH augmentation includes C:\\Program Files\\PowerShell\\7 so pwsh.exe is discoverable in service-mode"
    - "Windows daemon PATH augmentation includes %LOCALAPPDATA%\\Microsoft\\WindowsApps so Microsoft Store pwsh installs are discoverable"
    - "platformExtraBins on Windows continues to include existing entries (APPDATA\\npm, LOCALAPPDATA\\pnpm, LOCALAPPDATA\\Programs\\nodejs, C:\\Program Files\\Tailscale)"
    - "platformExtraBins on non-Windows is unchanged (path_other.go untouched)"
  artifacts:
    - path: internal/daemon/path_windows.go
      provides: "Extended platformExtraBins with PowerShell install paths"
      contains: "PowerShell"
    - path: internal/daemon/path_windows_test.go
      provides: "Windows-tagged tests covering PowerShell PATH augmentation"
      contains: "TestPlatformExtraBins"
  key_links:
    - from: internal/daemon/path_windows.go
      to: internal/pty/shells.go
      via: "PATH augmentation enables exec.LookPath('pwsh') to find pwsh.exe in service-mode daemon"
      pattern: "PowerShell"
---

<objective>
Extend `internal/daemon/path_windows.go::platformExtraBins` to include the two canonical PowerShell 7 install locations: `C:\Program Files\PowerShell\7` and `%LOCALAPPDATA%\Microsoft\WindowsApps`. Without this, a service-mode daemon on Windows cannot discover `pwsh.exe` via `exec.LookPath` because the user's PATH is not inherited into the service process.

Purpose: Closes SHELL-04 reliability on Windows (RESEARCH.md Pitfall 1 + Assumption A5). Plan 01's `DiscoverShells()` calls `exec.LookPath("pwsh")` which respects the augmented PATH set up by `applyExtraBinsToPath` at daemon startup.

Output: One modified file (`internal/daemon/path_windows.go`) and one new test file (`internal/daemon/path_windows_test.go`). No changes to `path.go`, `path_other.go`, or any Plan 01/02/04 file.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md
@.planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md
@internal/daemon/path.go
@internal/daemon/path_windows.go
@internal/daemon/path_other.go

<interfaces>
Existing file internal/daemon/path_windows.go (verified 23 lines):

  //go:build windows
  package daemon
  import ( "os"; "path/filepath" )
  func platformExtraBins() []string {
      var paths []string
      if appdata := os.Getenv("APPDATA"); appdata != "" {
          paths = append(paths, filepath.Join(appdata, "npm"))
      }
      if local := os.Getenv("LOCALAPPDATA"); local != "" {
          paths = append(paths, filepath.Join(local, "pnpm"))
          paths = append(paths, filepath.Join(local, "Programs", "nodejs"))
      }
      paths = append(paths, raw-literal-backtick C:\Program Files\Tailscale raw-literal-backtick)
      return paths
  }

platformExtraBins is consumed by applyExtraBinsToPath in path.go at daemon startup
(per PROJECT.md "Runtime PATH augmentation at daemon startup" — v1.5 Phase 32).
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Add Windows-tagged tests for PowerShell PATH entries (RED)</name>
  <files>internal/daemon/path_windows_test.go</files>
  <read_first>
    - internal/daemon/path_windows.go (the file being extended)
    - internal/daemon/path_other.go (sibling for context)
    - internal/daemon/path.go (to see how platformExtraBins is consumed)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md final section
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md "Pitfall 1"
  </read_first>
  <behavior>
    Create internal/daemon/path_windows_test.go with build tag //go:build windows containing three test functions:

    1. TestPlatformExtraBins_WindowsIncludesPowerShell:
       - t.Setenv("LOCALAPPDATA", "C:\\Users\\test\\AppData\\Local")
       - Call platformExtraBins()
       - Assert result contains the literal string "C:\\Program Files\\PowerShell\\7"
       - Assert result contains the literal string "C:\\Users\\test\\AppData\\Local\\Microsoft\\WindowsApps"

    2. TestPlatformExtraBins_PreservesExistingEntries:
       - t.Setenv APPDATA="C:\\AppData" and LOCALAPPDATA="C:\\Local"
       - Call platformExtraBins()
       - Assert result contains "C:\\AppData\\npm"
       - Assert result contains "C:\\Local\\pnpm"
       - Assert result contains "C:\\Local\\Programs\\nodejs"
       - Assert result contains "C:\\Program Files\\Tailscale"
       - (Regression guard — proves the existing four entries survive.)

    3. TestPlatformExtraBins_LocalAppDataEmpty:
       - t.Setenv("LOCALAPPDATA", "")
       - Call platformExtraBins()
       - Assert result does NOT contain any entry ending in "Microsoft\\WindowsApps" (conditional skip when LOCALAPPDATA missing)
       - Assert result STILL contains "C:\\Program Files\\PowerShell\\7" (hardcoded absolute, not env-dependent)

    Membership tests use a small helper:
      func containsString(slice []string, want string) bool {
          for _, s := range slice { if s == want { return true } }
          return false
      }
  </behavior>
  <action>
    Create new file internal/daemon/path_windows_test.go beginning with build tag //go:build windows (one line followed by blank line, then package decl).

    Package: daemon
    Imports: "os", "strings", "testing"

    Implement the three test functions described in <behavior>. Use a containsString helper (in the same file) to avoid pulling in slices.Contains generics — keeps the test file flat and stdlib-only.

    Do NOT yet modify path_windows.go — these tests must fail on Windows because the new entries do not yet exist. RED phase.

    Verification fallback: Since the dev box is macOS, this file's tests cannot be executed locally. The verify command below performs a cross-compile vet + gofmt check that exercises the file under the Windows build tag.
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && GOOS=windows GOARCH=amd64 go vet ./internal/daemon/... && test -z "$(gofmt -l internal/daemon/path_windows_test.go)"</automated>
  </verify>
  <acceptance_criteria>
    - File internal/daemon/path_windows_test.go exists
    - File first non-blank line is `//go:build windows`
    - File contains `func TestPlatformExtraBins_WindowsIncludesPowerShell`
    - File contains `func TestPlatformExtraBins_PreservesExistingEntries`
    - File contains `func TestPlatformExtraBins_LocalAppDataEmpty`
    - `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` exits 0
    - `gofmt -l internal/daemon/path_windows_test.go` produces no output
  </acceptance_criteria>
  <done>path_windows_test.go committed; cross-compiles cleanly under GOOS=windows from the macOS dev box. Tests will fail on Windows CI until Task 2 lands the implementation.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Extend platformExtraBins with PowerShell install paths (GREEN)</name>
  <files>internal/daemon/path_windows.go</files>
  <read_first>
    - internal/daemon/path_windows.go (current 23-line file)
    - .planning/phases/100-shell-session-backend-discovery/100-PATTERNS.md final section (showing exact diff)
    - .planning/phases/100-shell-session-backend-discovery/100-RESEARCH.md "Pitfall 1"
    - internal/daemon/path_windows_test.go (Task 1 output — verify assertion shapes)
  </read_first>
  <behavior>
    After this task, the Windows-tagged tests from Task 1 pass when run on a Windows CI runner. Specifically:
    - platformExtraBins() result includes the literal hardcoded path "C:\\Program Files\\PowerShell\\7"
    - When %LOCALAPPDATA% is non-empty, result includes "<LOCALAPPDATA>\\Microsoft\\WindowsApps"
    - When %LOCALAPPDATA% is empty, the WindowsApps entry is silently skipped (no entry ending in "Microsoft\\WindowsApps" in result)
    - All four existing entries (npm, pnpm, nodejs, Tailscale) preserved in their original order
  </behavior>
  <action>
    Modify internal/daemon/path_windows.go::platformExtraBins to add two entries.

    Inside the existing `if local := os.Getenv("LOCALAPPDATA"); local != ""` block, after the nodejs append, add:
        paths = append(paths, filepath.Join(local, "Microsoft", "WindowsApps"))

    After the existing Tailscale append, add a new line:
        paths = append(paths, raw-literal-backtick C:\Program Files\PowerShell\7 raw-literal-backtick)

    (Use a real Go raw string literal — backtick-delimited — matching the existing C:\Program Files\Tailscale entry. The "raw-literal-backtick" text above is just placeholder syntax for this prose; the actual file must use literal backticks.)

    Critical preservation points:
    - Backtick raw-string literals for Windows hardcoded paths (matches existing convention).
    - WindowsApps entry inside the LOCALAPPDATA conditional (skipped when env var empty).
    - PowerShell\7 entry outside the conditional (hardcoded absolute).
    - Existing 4 entries retained in original order; new entries appended next to their conceptual neighbors.

    Run `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` and `gofmt -w internal/daemon/path_windows.go` after editing.

    The tests cannot be executed on the macOS dev box. Acceptance gates rely on cross-compile vet + grep-style content assertions. Windows CI run will exercise the actual test logic in a follow-up verification job (not blocking this plan's completion).
  </action>
  <verify>
    <automated>cd /Users/ken/dev/agenthub && grep -q 'PowerShell' internal/daemon/path_windows.go && grep -q 'WindowsApps' internal/daemon/path_windows.go && grep -q 'Tailscale' internal/daemon/path_windows.go && GOOS=windows GOARCH=amd64 go vet ./internal/daemon/... && test -z "$(gofmt -l internal/daemon/path_windows.go)"</automated>
  </verify>
  <acceptance_criteria>
    - File internal/daemon/path_windows.go contains the literal string `PowerShell` (verify: `grep -c PowerShell internal/daemon/path_windows.go` returns 1 or more)
    - File contains the literal string `WindowsApps` (Microsoft Store install location)
    - File still contains the literal string `Tailscale` (regression: existing entry preserved)
    - File still contains the literal string `Programs` (regression: nodejs entry preserved)
    - File still contains the literal string `pnpm` (regression: pnpm entry preserved)
    - File still contains the literal string `npm` (regression: npm entry preserved)
    - File still begins with `//go:build windows`
    - `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` exits 0
    - `gofmt -l internal/daemon/path_windows.go` produces no output
    - Diff against pre-edit version shows exactly two new `paths = append(...)` lines (no other modifications)
  </acceptance_criteria>
  <done>path_windows.go extended with PowerShell\7 and Microsoft\WindowsApps entries. Cross-compile vet clean. Tests from Task 1 will pass on Windows CI. SHELL-04 reliability on Windows service-mode satisfied (Pitfall 1 mitigated).</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Augmented PATH → exec.LookPath | The augmented PATH dirs become candidates for binary discovery. If an attacker can write to one of those dirs (e.g., `C:\Program Files\PowerShell\7`), they could shadow pwsh.exe. On standard Windows installs, Program Files is administrator-writable only. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-100-04 | Spoofing | Windows pwsh.exe path resolution | mitigate | Per RESEARCH.md T-100-04: `C:\Program Files\PowerShell\7` is administrator-writable only on stock Windows (NTFS default ACLs). `%LOCALAPPDATA%\Microsoft\WindowsApps` is user-writable but Microsoft Store install paths are gated by Store package signing. exec.LookPath honors first-match-wins ordering — user PATH entries earlier in PATH take precedence over our appended entries (appended, not prepended, by applyExtraBinsToPath). Document: daemon process privilege == discovered-binary privilege. |
| T-100-04b | Tampering | %LOCALAPPDATA% env var injection | accept | A user who can override their own LOCALAPPDATA can already control discovery against any service running as them. Out of scope — daemon trust model is host trust. |
</threat_model>

<verification>
After both tasks complete:

```
# Cross-compile gate (run from macOS dev box):
GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...
test -z "$(gofmt -l internal/daemon/path_windows.go internal/daemon/path_windows_test.go)"

# Content gates:
grep -q 'PowerShell' internal/daemon/path_windows.go
grep -q 'WindowsApps' internal/daemon/path_windows.go
grep -q 'Tailscale' internal/daemon/path_windows.go  # regression: existing entry preserved

# On Windows CI (post-merge):
go test ./internal/daemon -run TestPlatformExtraBins -race -count=1
```

Validation map: this plan does not appear in 100-VALIDATION.md's per-task table because the validation file focuses on the discovery/spawn/status surfaces. Add a footnote in the SUMMARY noting that path_windows.go changes are gated by Windows CI runner exec.LookPath behavior; they are validated empirically by Plan 04's `GET /shells` integration test on Windows CI returning a `pwsh` entry.
</verification>

<success_criteria>
- Two files modified/created: `internal/daemon/path_windows.go` (modified), `internal/daemon/path_windows_test.go` (new).
- Cross-compile gate green on macOS dev box: `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/...` exits 0.
- Existing four entries in `platformExtraBins` preserved verbatim (regression-guarded).
- Two new entries appended: `Microsoft\\WindowsApps` (conditional on LOCALAPPDATA) and `C:\\Program Files\\PowerShell\\7` (hardcoded).
- No changes to `path.go`, `path_other.go`, or any other file.
- SHELL-04 reliability on Windows service-mode: pwsh.exe discoverable via exec.LookPath regardless of user-shell-init-files availability.
</success_criteria>

<output>
After completion, create `.planning/phases/100-shell-session-backend-discovery/100-03-SUMMARY.md` documenting:
- Diff: which two lines were added to `platformExtraBins`.
- Confirmation that POSIX `path_other.go` is untouched.
- Note on test execution: Windows-only tests deferred to CI; macOS dev box gates are vet + grep.
- Any deviation from RESEARCH.md Pitfall 1 mitigation with rationale.
</output>
