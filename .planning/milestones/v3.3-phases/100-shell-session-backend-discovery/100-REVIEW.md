---
phase: 100-shell-session-backend-discovery
reviewed: 2026-05-12T18:30:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - internal/pty/shells.go
  - internal/pty/shells_test.go
  - internal/daemon/engine.go
  - internal/daemon/engine_test.go
  - internal/daemon/path_windows.go
  - internal/daemon/path_windows_test.go
  - internal/daemon/api.go
  - internal/daemon/api_test.go
  - internal/daemon/types.go
  - internal/daemon/client.go
findings:
  critical: 0
  warning: 4
  info: 6
  total: 10
status: clean
fixed_at: 2026-05-12T19:15:00Z
fixed_findings:
  - WR-01: validate $HOME against "/" and "." (engine.go shell-workdir default)
  - WR-02: cache pty.DiscoverShells() in resolveShellSpawn (engine.go)
  - WR-03: allow /bin/sh as synthetic system-default shell (shells.go)
  - WR-04: inject /etc/shells path as parameter — removes mutable test hook (shells.go)
  - IN-01: document argvForShellBasename default branch as intentional
  - IN-02: drop redundant un-trimmed basename match in resolveShellSpawn
  - IN-03: route handleListShells through engine.DiscoverShells wrapper
  - IN-04: Windows shell tests skip explicitly on missing pwsh
  - IN-05: misleading testEtcShellsPath comment removed (variable deleted by WR-04 refactor)
  - IN-06: replace containsString with slices.Contains
fix_verification:
  build: pass (go build ./...)
  tests: pass (go test ./internal/pty/... ./internal/daemon/... -count=1 -race -skip TestOpenCodeANSICapture)
---

# Phase 100: Code Review Report

**Reviewed:** 2026-05-12T18:30:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

Phase 100 ships cross-platform shell discovery, an engine dispatch branch for shell-type sessions, a `GET /shells` HTTP route, Windows PATH augmentation for PowerShell, and a `status.Watch` bypass for shell sessions. The implementation closely mirrors the verified patterns in `internal/pty/detect.go` and `internal/daemon/engine.go` and the test suite is thorough.

No critical bugs, security vulnerabilities, or data-loss risks were found. The argv lists are compile-time constants, custom shell paths flow through the same `os.Stat`-gated `UpdateCLIPath` path used by AI CLIs, and no shell-style interpolation occurs anywhere in the dispatch chain.

Findings are concentrated in:
1. A behavioral gap where `os.UserHomeDir()` can return a useless value (e.g. `"/"`) in production but the workdir-default code does not validate it (the test code does).
2. A surprising re-run of `pty.DiscoverShells()` on the Windows safety-net path (filesystem I/O performed twice on every legacy-Windows `pwsh` spawn).
3. The synthetic `shell` entry silently drops `$SHELL=/bin/sh` despite `sh` being the most common system default on minimal Linux.
4. A package-level mutable test hook (`testEtcShellsPath`) that is read by production code and is not race-protected.
5. Several quality items: dead code in `argvForShellBasename`, a test that always passes regardless of outcome, and an inconsistency where the legacy-Windows pwsh fallback returns `sh.Path` (a `powershell.exe` path) under the requested `cli=pwsh`, which produces a `DetectedShell`-mismatch the caller may not expect.

## Warnings

### WR-01: `os.UserHomeDir()` "useless value" not validated in production code

**File:** `internal/daemon/engine.go:241-248`
**Issue:** When `workDir == ""` for a shell session, the code substitutes `os.UserHomeDir()` if it returns a non-empty string with no error:

```go
if workDir == "" {
    if home, err := os.UserHomeDir(); err == nil && home != "" {
        workDir = home
    }
}
```

But `os.UserHomeDir()` reads `$HOME` on POSIX (`%USERPROFILE%` on Windows) without semantic validation. On service-mode daemons or misconfigured environments, `$HOME` is frequently `/` or `.`. The corresponding test (`TestCreateSession_ShellEmptyWorkDirHome` at engine_test.go:677) explicitly skips when home is `""`, `"/"`, or `"."` — proving the author knew these values are unsafe — but the production code does not apply the same guard. RESEARCH.md Pitfall 4 explicitly warned about service-mode daemons landing in `/`.

**Impact:** A user on a service-mode daemon with `$HOME=/` opens a "shell" session with no folder picked and lands in `/`. Shell history files written at `/` will fail (read-only) or pollute the root.

**Fix:**
```go
if workDir == "" {
    if home, err := os.UserHomeDir(); err == nil && home != "" && home != "/" && home != "." {
        workDir = home
    }
    // If unreliable, fall through to the empty workdir (matches AI-CLI behavior).
}
```

Alternatively, validate via `filepath.IsAbs(home) && home != string(filepath.Separator)`.

### WR-02: Duplicate `pty.DiscoverShells()` call in Windows legacy-pwsh safety net

**File:** `internal/daemon/engine.go:489-506`
**Issue:** The function calls `pty.DiscoverShells()` once in the primary discovery loop, then again inside the `cli == "pwsh"` legacy-Windows fallback:

```go
// (2) Live discovery.
for _, sh := range pty.DiscoverShells() {       // call #1
    if sh.Name == cli { ... }
}

// (3) Legacy Windows safety net
if cli == "pwsh" {
    for _, sh := range pty.DiscoverShells() {   // call #2 — same data
        if sh.Name == "powershell" { ... }
    }
}
```

`DiscoverShells` performs filesystem I/O (`exec.LookPath` for every entry in `knownShellSpecs` + potentially reads `/etc/shells`). On every `cli="pwsh"` request, this work is done twice. Beyond performance, this is a correctness fragility: between the two calls PATH could mutate (unlikely but possible), so the two scans can disagree.

**Impact:** Wasted I/O per spawn; potential consistency issue under test fixtures that mutate `t.Setenv("PATH", ...)` between scans.

**Fix:** Cache the discovery result in a local variable:
```go
discovered := pty.DiscoverShells()
for _, sh := range discovered {
    if sh.Name == cli {
        return sh.Path, append([]string(nil), sh.Argv...), true
    }
}
if cli == "pwsh" {
    for _, sh := range discovered {
        if sh.Name == "powershell" {
            return sh.Path, append([]string(nil), sh.Argv...), true
        }
    }
}
```

### WR-03: Synthetic `shell` entry rejects `sh` — the most common minimal-container default

**File:** `internal/pty/shells.go:149-156` (`isEndorsedShellBasename`)
**Issue:** The allowlist for the synthetic POSIX system-default entry is:

```go
case "bash", "zsh", "pwsh", "powershell":
    return true
```

`/bin/sh` is omitted. On Alpine, distroless, and most minimal Linux containers — exactly the deployment targets noted in RESEARCH.md Pitfall 2 — `$SHELL=/bin/sh` (or `SHELL` is unset and `/bin/sh` is the only shell installed). With this allowlist, the daemon will report **zero** shells via `GET /shells` on such hosts, and `cli="shell"` (system default) will fail to resolve.

Note that `knownShellSpecs` intentionally excludes `sh` (out-of-scope per REQUIREMENTS.md), but for the *synthetic system-default* entry that derives its path from `$SHELL`, accepting `sh` would honor the actual user environment without expanding the "selectable shells" surface. This is also inconsistent with the requirements doc line 19 referenced in RESEARCH.md ("Linux: `$SHELL`, `/etc/shells` entries").

**Impact:** Slim-container deployments cannot use the "system default" shell session even when `/bin/sh` is the only viable interactive shell. Users see an empty shell picker.

**Fix:** Decide explicitly. Either (a) add `"sh"` to `isEndorsedShellBasename`:
```go
case "sh", "bash", "zsh", "pwsh", "powershell":
    return true
```
…and add a corresponding `case "sh": return []string{"-i"}` to `argvForShellBasename` (already the default, so technically a no-op but worth being explicit), OR (b) document in `RESEARCH.md`/`REQUIREMENTS.md` that `/bin/sh` system defaults are intentionally unsupported in Phase 100 and add a test asserting this.

### WR-04: `testEtcShellsPath` is a package-level mutable variable read by production code without synchronization

**File:** `internal/pty/shells.go:58` (declaration) + `shells.go:98-100` (read site)
**Issue:** The variable is declared in production code (not under a `_test.go` build tag) so test code can assign to it:

```go
var testEtcShellsPath = ""
...
etcShellsPath := "/etc/shells"
if testEtcShellsPath != "" {
    etcShellsPath = testEtcShellsPath
}
```

This is read by `DiscoverShells`, which is called from the `GET /shells` HTTP handler and from `engine.resolveShellSpawn`. In production the value is always `""`, so behavior is correct. Under `-race`:
- No current test calls `t.Parallel()`, so package-level tests run sequentially.
- BUT three tests write `testEtcShellsPath` (`TestDiscoverShells_NoEtcShells`, `TestDiscoverShells_EtcShellsFixture`, `TestDiscoverShells_EmptySHELLEnv_NoSyntheticEntry`) without coordinating with any concurrent reader.
- If a future test (or someone running `go test -count=N -parallel=M`) introduces parallelism, the unsynchronized read/write is a data race.

The race is latent today but the design is fragile. Also: the comment on line 53-57 says "Empty string in production builds" — but the variable is *not* under a build tag. A user-visible production binary still has `testEtcShellsPath = ""` baked in; mostly harmless but the comment is misleading.

**Impact:** Latent race; misleading comment; production binary carries a test-only field.

**Fix:** Two options:
1. **Preferred:** Inject `/etc/shells` path as a parameter (or via a `discoverOpts` struct), making the test hook an explicit function argument:
   ```go
   func DiscoverShells() []DetectedShell { return discoverShells("/etc/shells") }
   func discoverShells(etcShellsPath string) []DetectedShell { ... }
   ```
   Tests call `discoverShells(tmpPath)` directly.

2. **Lighter touch:** Guard the variable with `sync/atomic` or a mutex, and update the comment to say "Read at every `DiscoverShells` call; mutated only by tests."

### Bonus — `TestDiscoverShells_Windows` and `TestDiscoverShells_WindowsPowerShell` never fail

**File:** `internal/pty/shells_test.go:244-261, 263-286`
**Note:** Combined into WR-04's family of test-quality issues but documented here for completeness.

Both Windows-only tests can complete without making any assertion. `TestDiscoverShells_Windows` only asserts `pwsh.Path` ends in `.exe` if a `pwsh` entry exists; if no pwsh is discovered, the test silently passes. `TestDiscoverShells_WindowsPowerShell` logs (`t.Log`) instead of `t.Skip` when neither is found, and makes no assertion afterward. The Windows CI runner has pwsh installed (per VERIFICATION.md A1), so in practice these tests *do* probe — but as written they are not failure-emitting under any condition. See IN-04.

## Info

### IN-01: `argvForShellBasename` carries a dead-code branch

**File:** `internal/pty/shells.go:161-169`
**Issue:** The function is only called from one site (`shells.go:111`) where the basename is guaranteed to be one of `bash`, `zsh`, `pwsh`, or `powershell` (because `isEndorsedShellBasename` was already checked at line 93). The `default` branch `return []string{"-i"}` is therefore unreachable in current code.

If `isEndorsedShellBasename` ever grows (e.g. WR-03's `sh` addition), the default `["-i"]` is the right fallback — but the dead branch is currently misleading.

**Fix:** Either drop the `default` branch and let the function panic (or return nil) for unknown inputs, OR explicitly add a comment that the default is reserved for future basenames passed by tests / library callers.

### IN-02: `resolveShellSpawn` override-branch matches on three names — one is redundant

**File:** `internal/daemon/engine.go:478-482`
**Issue:**
```go
for _, spec := range pty.KnownShellSpecs() {
    if spec.Name == cli || spec.Name == baseNoExt || spec.Name == base {
        ...
    }
}
```
- `spec.Name == cli`: matches when the override key matches a known shell name (the documented `cliPaths["powershell"]=...` case).
- `spec.Name == baseNoExt`: matches when the override path's basename (minus `.exe`) matches a spec (e.g. override is `/usr/local/bin/zsh.exe` — wouldn't normally exist, but covered).
- `spec.Name == base`: redundant with `baseNoExt` for all current specs because none of `bash`/`zsh`/`pwsh`/`powershell` end in `.exe`. The only basenames that would match `base` and not `baseNoExt` are spec names ending in `.exe` — and there are none.

Not a bug, just unnecessary. Consider trimming to two conditions and documenting why.

### IN-03: `GET /shells` discovers shells via `pty.DiscoverShells()` directly, not via the engine

**File:** `internal/daemon/api.go:495-507`
**Issue:** Most handlers in `api.go` delegate to `a.engine.*` (so they can be tested with a mocked engine and so engine-level locking is consistent). `handleListShells` calls `pty.DiscoverShells()` directly, bypassing the engine. This is internally consistent with `pty.*` being a stateless discovery surface, but it means:
- Tests cannot inject a fake shell discovery into the API layer; they must mock at the `pty` package level (PATH manipulation), which they do — but this couples HTTP tests to filesystem state.
- Future engine-level caching or rate-limiting cannot be applied without restructuring.

PATTERNS.md anticipated this: line 508 noted "If the planner prefers engine-mediation (for testability with spyBackend / DI), add `engine.DiscoverShells()` as a thin wrapper that calls `pty.DiscoverShells()`." That wrapper was not added.

**Fix (optional):** Add `func (e *SessionEngine) DiscoverShells() []pty.DetectedShell { return pty.DiscoverShells() }` and route the handler through it. Cheap and future-proofs.

### IN-04: Windows test functions silently pass when their preconditions are not met

**File:** `internal/pty/shells_test.go:244-261, 263-286`
**Issue:** See WR-04 bonus. Both Windows tests use `t.Log` instead of `t.Skip` when their fixture preconditions (pwsh installed) are not met, and have no assertion paths that fire in the absence of pwsh. A buggy GitHub Actions Windows image where pwsh is suddenly absent would silently pass these tests.

**Fix:**
```go
if !hasPowerShell && !hasPwsh {
    t.Skip("neither powershell nor pwsh found on Windows runner")
}
// follow with explicit assertion
```

### IN-05: Misleading comment on `testEtcShellsPath`

**File:** `internal/pty/shells.go:53-57`
**Issue:** Comment claims "Empty string in production builds" but the variable is declared in a regular `.go` file with no build tag. Production binaries do compile this variable (it's just always `""`). The behavior described is correct; the wording is misleading.

**Fix:** Reword to "Variable is initialised to empty string; only mutated by tests in shells_test.go via direct assignment."

### IN-06: `containsString` (shells.go:198-205) is unexported and could use `slices.Contains`

**File:** `internal/pty/shells.go:198-205`
**Issue:** With Go 1.21+, `slices.Contains` is in the standard library and eliminates this 8-line helper:
```go
import "slices"
...
if !slices.Contains(shells, shellEnv) {
```

PATTERNS.md noted similar avoidance for `path_windows_test.go` for consistency, but the production code here can use the stdlib. Minor.

**Fix:** Replace `containsString(shells, shellEnv)` with `slices.Contains(shells, shellEnv)` and remove the helper.

---

_Reviewed: 2026-05-12T18:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
