# Phase 32: Daemon Startup Performance - Research

**Researched:** 2026-03-25
**Domain:** Go daemon performance, service-mode PATH resolution, polling loop timing
**Confidence:** HIGH

## Summary

This phase addresses three tightly scoped bugs in `app.go` and `internal/daemon/service.go`. All root causes are fully visible in the source code with no ambiguity.

**PERF-01/PERF-02** share a single root cause: `pollSessionStatus` in `app.go` (line 144) begins with `time.Sleep(2 * time.Second)` before making its first HTTP call. The fix is to restructure the loop so the HTTP call fires immediately, then sleep 500ms between subsequent polls. The deadline logic and status-change emission are otherwise correct.

**PERF-03** root cause: when the daemon runs as a launchd/systemd user service, `os.Environ()` (called in `NativePTYBackend.Create`) returns the service process's limited PATH. Tools installed via nvm (e.g., `~/.nvm/versions/node/<ver>/bin`), Volta (`~/.volta/bin`), and Homebrew (`/opt/homebrew/bin`) are absent from the service PATH because those entries are injected by shell initialization files (`~/.zshrc`, `~/.bashrc`), which launchd does not source. The fix is to augment the daemon process PATH at startup with well-known install directories before any PTY session is created.

**Primary recommendation:** Fix the sleep positioning in `pollSessionStatus` (one-line change) and add a `augmentPath()` function called at the top of `runDaemonCore()` that prepends well-known nvm/volta/Homebrew paths to `os.Environ`'s PATH via `os.Setenv("PATH", ...)`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PERF-01 | Session status appears immediately after session creation (no artificial delay) | Root cause is `time.Sleep(2s)` before first poll in `pollSessionStatus`; fix: move first HTTP call before any sleep |
| PERF-02 | `pollSessionStatus` first poll runs without 2-second sleep; subsequent polls at 500ms intervals | Same fix as PERF-01; restructure loop to poll-then-sleep pattern with 500ms interval |
| PERF-03 | Service-mode daemon resolves agent CLIs in user PATH (nvm, volta, Homebrew) | `runDaemonCore()` must call `os.Setenv("PATH", augmented)` before sessions are created; `exec.LookPath` uses the process env PATH |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- Go code: `go fmt`, `golangci-lint`, context-aware functions
- Testing: `go test` with `testing` package; 80%+ coverage in critical components
- All tests must pass: `go test ./...`
- Use `go fmt` on modified files
- Silent fallbacks (`or {}`) are forbidden — let failures surface

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` (stdlib) | Go 1.26 | PATH resolution via `exec.LookPath` | Already used in `detect.go` |
| `os` (stdlib) | Go 1.26 | `os.Setenv`, `os.Getenv`, `os.UserHomeDir` | Used throughout project |
| `time` (stdlib) | Go 1.26 | `time.Sleep`, `time.Ticker` | Already used in `app.go` |
| `kardianos/service` | v1.2.4 | launchd/systemd service management | Already in go.mod; `EnvVars map[string]string` field in `service.Config` |

### No New Dependencies

All fixes use stdlib only. No new imports are needed.

**Version verification:**
```bash
# kardianos/service v1.2.4 already in go.mod — verified
# All other packages are Go stdlib
```

## Architecture Patterns

### Current `pollSessionStatus` (broken)
```go
// app.go:140 — BUG: sleeps 2s BEFORE first call
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        time.Sleep(2 * time.Second)   // <-- fires before first HTTP call
        s, err := a.client.GetSessionStatus(sessionID)
        // ...
    }
}
```

### Fixed `pollSessionStatus` (poll-then-sleep)
```go
// Source: pattern from startHealthPoller in app.go (uses ticker correctly)
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        s, err := a.client.GetSessionStatus(sessionID)  // immediate first call
        if err != nil {
            return
        }
        if s != last {
            last = s
            if a.ctx != nil && a.ctx.Value("frontend") != nil {
                runtime.EventsEmit(a.ctx, "session:status", map[string]string{
                    "sessionId": sessionID,
                    "status":    s,
                })
            }
            if s == string(status.StatusErrored) {
                return
            }
        }
        time.Sleep(500 * time.Millisecond)  // sleep AFTER poll
    }
}
```

### PATH Augmentation Pattern
```go
// New function in internal/daemon/process.go (or a new path.go file)
// augmentServicePath prepends well-known user tool directories to the process
// PATH so that CLIs installed via nvm, volta, or Homebrew are found when the
// daemon runs as a launchd/systemd user service (which does not source shell
// init files). Called once at daemon startup before any session is created.
func augmentServicePath() {
    home, err := os.UserHomeDir()
    if err != nil {
        return // no HOME, nothing to augment
    }

    candidates := []string{
        // Volta — single well-known path, version-agnostic
        filepath.Join(home, ".volta", "bin"),
        // Homebrew on Apple Silicon
        "/opt/homebrew/bin",
        // Homebrew on Intel / Linux
        "/usr/local/bin",
        // nvm active version via alias file
        nvmActiveBin(home),
    }

    current := os.Getenv("PATH")
    var extra []string
    for _, dir := range candidates {
        if dir == "" {
            continue
        }
        if _, err := os.Stat(dir); err == nil {
            extra = append(extra, dir)
        }
    }
    if len(extra) > 0 {
        _ = os.Setenv("PATH", strings.Join(extra, string(os.PathListSeparator))+string(os.PathListSeparator)+current)
    }
}

// nvmActiveBin reads ~/.nvm/alias/default to find the active node version
// and returns the bin directory path, or "" if nvm is not installed.
func nvmActiveBin(home string) string {
    aliasFile := filepath.Join(home, ".nvm", "alias", "default")
    data, err := os.ReadFile(aliasFile)
    if err != nil {
        return ""
    }
    version := strings.TrimSpace(string(data))
    if version == "" {
        return ""
    }
    // nvm alias/default contains just a version number like "20" or "v20.19.3"
    // Check for full version first, then try with "v" prefix
    if !strings.HasPrefix(version, "v") {
        version = "v" + version
    }
    // Find the matching node version directory
    nvmDir := filepath.Join(home, ".nvm", "versions", "node")
    entries, err := os.ReadDir(nvmDir)
    if err != nil {
        return ""
    }
    for _, e := range entries {
        if strings.HasPrefix(e.Name(), version) {
            return filepath.Join(nvmDir, e.Name(), "bin")
        }
    }
    return ""
}
```

**Call site:** `runDaemonCore()` in `process.go`, before `NewSessionEngine()`:
```go
func runDaemonCore(ctx context.Context) {
    augmentServicePath()   // <-- add this line
    socketPath := DefaultSocketPath()
    // ...
}
```

### Anti-Patterns to Avoid

- **Sleep-first polling:** `time.Sleep` at the top of a polling loop creates guaranteed latency. Always poll first, sleep after.
- **time.Ticker for bounded polling:** `time.Ticker` is appropriate for indefinite background polling (`startHealthPoller`). For bounded session status polling with a deadline, the poll-then-sleep pattern is simpler and sufficient.
- **Capturing PATH at install time:** Setting `EnvVars["PATH"]` in `newServiceConfig()` captures the PATH at `agenthub daemon install` time. If the user later switches nvm node versions, the service gets a stale PATH. Runtime augmentation at daemon startup is more robust.
- **Shelling out to get PATH:** `exec.Command("sh", "-c", "echo $PATH")` from a launchd service would also have the limited PATH problem — this does not help.
- **Modifying PTY Env instead of process Env:** Setting PATH in `req.Env` passed to `NativePTYBackend.Create` only affects child process environment, not `exec.LookPath` resolution. `exec.Command("claude", ...)` calls `exec.LookPath("claude")` which reads the daemon process's own `PATH` env, not the child's.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Service management | Custom launchd plist writer | `kardianos/service` already in use | Already installed, already handles launchd/systemd/Windows |
| Path existence check | Custom path scanner | `os.Stat(dir)` | Stdlib, already used throughout project |
| nvm version resolution | Parse nvm shell scripts | Read `~/.nvm/alias/default` + `os.ReadDir` | nvm alias file is a simple text file with the version string |

## Common Pitfalls

### Pitfall 1: exec.LookPath Uses Process PATH, Not cmd.Env PATH
**What goes wrong:** Developer sets `req.Env` in `CreateRequest` to include a corrected PATH, but `exec.LookPath` (called inside `exec.Command`) still uses the daemon process's own `$PATH`.
**Why it happens:** Go's `os/exec` resolves the command name to a full path using `exec.LookPath` at `exec.Command()` call time, before `cmd.Env` is consulted. The child's environment is only applied after fork.
**How to avoid:** Use `os.Setenv("PATH", ...)` to modify the daemon process's own PATH before any `exec.LookPath` or `exec.Command` call.
**Warning signs:** Test with `exec.LookPath("claude")` before and after the fix — should return a path after augmentation, error before.

### Pitfall 2: nvm Default Alias Is a Version Number, Not a Full Path
**What goes wrong:** `~/.nvm/alias/default` may contain `"20"` (short alias) or `"v20.19.3"` (full version) — the format varies.
**Why it happens:** nvm allows both formats. `20` resolves to the latest v20.x installed.
**How to avoid:** Normalize the alias by adding `"v"` prefix if missing, then scan `~/.nvm/versions/node/` for a directory that starts with that prefix.
**Warning signs:** `nvmActiveBin` returns `""` on machines where it should return a path.

### Pitfall 3: pollSessionStatus Goroutine Leaks
**What goes wrong:** If `pollSessionStatus` polls immediately and `GetSessionStatus` panics or blocks, the goroutine can leak.
**Why it happens:** Removing the sleep doesn't change goroutine lifecycle — the 60-second deadline and `err != nil` return still handle termination.
**How to avoid:** The existing deadline (`time.Now().Add(60 * time.Second)`) and error-return guard are sufficient. No changes needed there.

### Pitfall 4: augmentServicePath Called After exec.LookPath
**What goes wrong:** If `augmentServicePath()` is called after `NewSessionEngine()` or `DetectCLIs()`, the PATH augmentation is too late for those calls.
**Why it happens:** PATH must be set before any `exec.LookPath` usage in the daemon.
**How to avoid:** Call `augmentServicePath()` as the very first line of `runDaemonCore()`.

### Pitfall 5: 500ms Poll Interval Test Timing
**What goes wrong:** Tests for `pollSessionStatus` timing may be flaky if they rely on wall-clock assertions.
**Why it happens:** CI runners can be slow; `500ms * N` assertions are unreliable.
**How to avoid:** Test the behavioral contract (first call is immediate, events are emitted) rather than timing assertions. Use a fake/stub client with a call counter.

## Code Examples

### Verified: Current broken pattern (app.go:140-162)
```go
// Source: /Users/ken/dev/agenthub/app.go lines 140-162
func (a *App) pollSessionStatus(sessionID string) {
    var last string
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        time.Sleep(2 * time.Second)   // BUG: fires before first GetSessionStatus
        s, err := a.client.GetSessionStatus(sessionID)
        if err != nil {
            return
        }
        // ...
    }
}
```

### Verified: kardianos/service EnvVars field (alternative approach, not recommended)
```go
// Source: /Users/ken/go/pkg/mod/github.com/kardianos/service@v1.2.4/service.go:141
// EnvVars map[string]string is available in service.Config
// The launchd template at service_darwin.go:299-307 renders it as EnvironmentVariables
// The systemd template at service_systemd_linux.go:329-330 renders it as Environment=K=V
// This approach captures PATH at install time — NOT used in this phase
cfg.EnvVars = map[string]string{"PATH": os.Getenv("PATH")}
```

### Verified: go-pty CommandContext PATH resolution chain
```
NativePTYBackend.Create()
  → p.CommandContext(ctx, req.CLI, req.Args...)  [stores req.CLI as c.Path verbatim]
  → c.start()                                    [cmd_unix.go:23]
  → exec.Command(c.Path, c.Args[1:]...)          [calls exec.LookPath if c.Path has no slash]
  → exec.LookPath("claude")                      [reads daemon process os.Getenv("PATH")]
```
Source: `/Users/ken/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/cmd_unix.go:23`

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| sleep-first polling | poll-first with shorter interval | This phase | Eliminates 2s blank status period |
| no PATH augmentation | runtime PATH augmentation at daemon start | This phase | Service-mode agents resolve correctly |

## Open Questions

1. **nvm default alias format variability**
   - What we know: `~/.nvm/alias/default` may contain `"20"`, `"v20.19.3"`, or other formats
   - What's unclear: Are there other nvm alias formats (e.g., `"lts/iron"`) that need handling?
   - Recommendation: Handle the common cases (`"20"` → `"v20"` prefix scan) and log a warning when the alias cannot be resolved. Do not block on edge cases — the function returns `""` gracefully.

2. **Linux PATH for service mode**
   - What we know: systemd user services also strip shell-init PATH additions; `~/.volta/bin` and nvm paths would be missing too
   - What's unclear: Whether Homebrew on Linux installs to `/home/linuxbrew/.linuxbrew/bin`
   - Recommendation: Include `/home/linuxbrew/.linuxbrew/bin` as a candidate path alongside the macOS paths. `os.Stat` skips non-existent dirs safely.

## Environment Availability

Step 2.6: SKIPPED — this phase is purely code changes within the existing Go binary. No new external tools or services are required. The nvm/volta/Homebrew directories probed at runtime are optional (if absent, `os.Stat` fails and they are skipped).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./... -run TestPoll -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| PERF-01 | Status event emitted within 1 second of `pollSessionStatus` start | unit | `go test . -run TestPollSessionStatus_ImmediateFirstCall -v` | ❌ Wave 0 |
| PERF-02 | First `GetSessionStatus` call happens before any sleep | unit | `go test . -run TestPollSessionStatus_ImmediateFirstCall -v` | ❌ Wave 0 |
| PERF-03 | `augmentServicePath` adds nvm/volta/Homebrew dirs to process PATH | unit | `go test ./internal/daemon/ -run TestAugmentServicePath -v` | ❌ Wave 0 |
| PERF-03 | `nvmActiveBin` returns correct bin path for installed nvm | unit | `go test ./internal/daemon/ -run TestNvmActiveBin -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `app_test.go` — add `TestPollSessionStatus_ImmediateFirstCall` covering PERF-01/PERF-02 using a stub client with a call counter
- [ ] `internal/daemon/path_test.go` — add `TestAugmentServicePath` and `TestNvmActiveBin` covering PERF-03; use `t.TempDir()` to simulate nvm directory structure

## Sources

### Primary (HIGH confidence)
- Source code analysis: `/Users/ken/dev/agenthub/app.go:140-162` — `pollSessionStatus` with `time.Sleep` before first poll
- Source code analysis: `/Users/ken/dev/agenthub/internal/pty/native.go:46` — `os.Environ()` used as base env; shows PATH comes from daemon process
- Source code analysis: `/Users/ken/go/pkg/mod/github.com/aymanbagabas/go-pty@v0.2.2/cmd_unix.go:23` — `exec.Command(c.Path, ...)` calls `exec.LookPath` using process PATH
- Source code analysis: `/Users/ken/go/pkg/mod/github.com/kardianos/service@v1.2.4/service.go:141` — `EnvVars map[string]string` field in `service.Config`
- Source code analysis: `/Users/ken/go/pkg/mod/github.com/kardianos/service@v1.2.4/service_darwin.go:299-307` — launchd plist template renders `EnvVars` as `EnvironmentVariables`
- Go stdlib docs: `go doc os/exec LookPath` — "searches for an executable in the directories named by the PATH environment variable"

### Secondary (MEDIUM confidence)
- Runtime probe: `launchctl getenv PATH` output confirms launchd PATH includes system dirs and Homebrew, but NOT nvm/volta entries (those come from shell init)
- Runtime probe: `ls ~/.nvm/versions/node/` confirms nvm installs per-version under `~/.nvm/versions/node/<ver>/bin/`

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib + existing dependency; verified in module cache
- Architecture: HIGH — root causes confirmed by reading actual source files, not assumed
- Pitfalls: HIGH — exec.LookPath behavior verified via `go doc`; nvm alias format verified by inspection

**Research date:** 2026-03-25
**Valid until:** 2026-06-25 (stable stdlib behavior; kardianos/service v1.2.4 pinned)
