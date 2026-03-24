# Phase 23: Service Manager Integration - Research

**Researched:** 2026-03-24
**Domain:** Cross-platform service management (launchd / systemd user / Windows SCM) in Go
**Confidence:** HIGH (library API verified via pkg.go.dev; platform behavior verified via official docs)

---

## Summary

Phase 23 adds `agenthub daemon install/uninstall/start/stop` subcommands that register the daemon as a platform-native service. The daemon already has a clean `RunDaemon()` entry point in `internal/daemon/process.go` that blocks on SIGTERM/SIGINT — exactly the foreground contract service managers require. No double-fork or daemonization is needed; the service manager owns the process lifecycle.

The standard Go library for this is `github.com/kardianos/service` (v1.2.4, released 2025-07-14). It abstracts launchd (macOS), systemd user units (Linux), and Windows SCM behind one `service.Config` and one `service.Control()` call. On macOS, setting `Option["UserService"] = true` installs a plist in `~/Library/LaunchAgents/`. On Linux, setting `Option["UserService"] = true` installs a `.service` file in `~/.config/systemd/user/`. On Windows, the library registers with the SCM under the calling user's credentials.

The implementation adds a new `svc` subpackage (or file) in `internal/daemon/` with a `ServiceConfig()` factory, and adds `daemon install/uninstall/start/stop` dispatch in `cmd/agenthub-cli/main.go`. The existing `RunDaemon()` function is wrapped in the `service.Interface` so the service manager can call it cleanly.

**Primary recommendation:** Use `github.com/kardianos/service` v1.2.4 with `UserService: true` on all three platforms. Do not hand-roll plist, unit, or SCM registration code.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SVC-01 | Daemon can be installed as a platform service (launchd/systemd/Windows SCM) | `kardianos/service` v1.2.4 `service.Control(s, "install")` writes the platform artifact; `UserService: true` targets user-scope (LaunchAgents / systemd user / SCM user) |
| SVC-02 | Daemon auto-starts on login when installed as a service | macOS: `RunAtLoad: true` + `KeepAlive: true` in launchd plist; Linux: `systemctl --user enable` (kardianos does this automatically); Windows: StartType `"automatic"` (default) starts after login |
| SVC-03 | User can install/uninstall/start/stop the service from CLI (`agenthub daemon install/uninstall/start/stop`) | `service.Control(s, action)` with action = "install"/"uninstall"/"start"/"stop" dispatched from a `daemon` subcommand in main.go |
</phase_requirements>

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/kardianos/service` | v1.2.4 | Cross-platform service install/uninstall/start/stop | Only maintained library covering launchd + systemd + Windows SCM with a single API; used by k0s, Prometheus exporters, and similar Go system tools |

### Supporting

No additional libraries needed. The existing `internal/daemon` package provides `RunDaemon()`, `DefaultSocketPath()`, and `CleanupStaleSocket()` — all of which the service wrapper will call.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `kardianos/service` | Hand-rolled plist + unit file + SCM registration | Writing correct plist XML, systemd unit syntax, and Windows Service API is 600+ lines across 3 platforms with many edge cases; do not do this |
| `kardianos/service` | `distro/service` (niche) | Unmaintained, no Windows SCM support |

**Installation:**
```bash
go get github.com/kardianos/service@v1.2.4
```

**Version verification:** Confirmed v1.2.4 via `go list -m github.com/kardianos/service@latest` on 2026-03-24. Published 2025-07-14.

---

## Architecture Patterns

### Recommended Project Structure

```
cmd/agenthub-cli/
├── main.go               # add `daemon` subcommand dispatch
├── cmd_daemon.go         # NEW: install/uninstall/start/stop handlers
internal/daemon/
├── process.go            # existing RunDaemon() — wrap in service.Interface
├── service.go            # NEW: service.Config factory + service.Interface impl
├── service_test.go       # NEW: tests for config factory + svc wrapper compile
```

### Pattern 1: Service Interface Wrapper

The `kardianos/service` library requires your program to implement:

```go
type Interface interface {
    Start(s service.Service) error  // must return quickly; launch goroutines
    Stop(s service.Service) error   // must return quickly; signal goroutine to stop
}
```

**What:** Wrap the existing `RunDaemon()` logic behind this interface. `Start()` launches a goroutine that runs the daemon core; `Stop()` cancels a context to trigger graceful shutdown.

**When to use:** Always — this is the required entry point for `kardianos/service`.

**Example:**

```go
// Source: kardianos/service README + pkg.go.dev/github.com/kardianos/service
type daemonSvc struct {
    cancel context.CancelFunc
    done   chan struct{}
}

func (d *daemonSvc) Start(s service.Service) error {
    ctx, cancel := context.WithCancel(context.Background())
    d.cancel = cancel
    d.done = make(chan struct{})
    go func() {
        defer close(d.done)
        runDaemonWithContext(ctx)  // extracted from RunDaemon()
    }()
    return nil
}

func (d *daemonSvc) Stop(s service.Service) error {
    d.cancel()
    <-d.done
    return nil
}
```

### Pattern 2: Service Config with UserService

```go
// Source: pkg.go.dev/github.com/kardianos/service
func buildServiceConfig(exePath string) *service.Config {
    return &service.Config{
        Name:        "agenthub-daemon",
        DisplayName: "AgentHub Daemon",
        Description: "AgentHub session manager daemon",
        Executable:  exePath,   // full path to agenthub binary
        Arguments:   []string{"daemon"},
        Option: service.KeyValue{
            "UserService": true,   // LaunchAgents (macOS), systemd user (Linux), SCM user (Windows)
            "RunAtLoad":   true,   // macOS: start on login
            "KeepAlive":   true,   // macOS: restart if it crashes
        },
    }
}
```

**Key:** `Executable` must be set to the absolute path of the installed `agenthub` binary, not just `"agenthub"`. Use `os.Executable()` to resolve it at install time.

### Pattern 3: CLI Dispatch for Service Actions

```go
// cmd/agenthub-cli/main.go additions
case "daemon":
    err = cmdDaemon(os.Args[2:])
```

```go
// cmd_daemon.go
func cmdDaemon(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("usage: agenthub daemon <install|uninstall|start|stop|run>")
    }
    switch args[0] {
    case "run":
        // Called by service manager — RunDaemon() blocks until signal
        daemon.RunDaemon()
        return nil
    case "install", "uninstall", "start", "stop":
        return daemon.ServiceControl(args[0])
    default:
        return fmt.Errorf("unknown daemon subcommand %q", args[0])
    }
}
```

**Note on `run` vs direct execution:** When the service manager starts the binary, it passes `daemon run` (or `daemon` directly, depending on config). `Arguments: []string{"daemon"}` in service.Config means the binary is started as `agenthub daemon` — which already calls `daemon.RunDaemon()` in `main.go`. This is already handled by the existing dispatch. No change to `RunDaemon()` needed.

**Clarification:** With `Arguments: []string{"daemon"}`, the service manager will execute `<exe> daemon`, which hits the existing `cmd == "daemon"` branch in `main.go`. The `install/uninstall/start/stop` actions are separate subcommands invoked by the user, not by the service manager.

### Pattern 4: EnsureDaemon Interaction After Install

After a service is installed and `RunAtLoad`/`KeepAlive` are set, the daemon will already be running when the user runs any `agenthub` command. `EnsureDaemon()` in `process.go` first checks if the daemon is reachable before spawning. This means the service-managed daemon will be detected and reused — no double-start. No changes needed to `EnsureDaemon()`.

### Anti-Patterns to Avoid

- **Using `service.Run()` from within `RunDaemon()`:** `RunDaemon()` uses `signal.NotifyContext` which is incompatible with the service manager's lifecycle callbacks. When running as a service, use the `daemonSvc` wrapper with `service.Run(svc)` instead.
- **Not setting `Executable` to an absolute path:** If `Executable` is empty, `kardianos/service` falls back to `os.Executable()` at install time, which resolves correctly only if called from the installed binary location. Explicitly resolve with `os.Executable()` at install time to be safe.
- **Using `--user` flag with systemd manually:** `kardianos/service` handles the `--user` scope automatically when `UserService: true` is set on Linux. Do not call `systemctl --user enable` separately.
- **Setting `KeepAlive: true` for Windows:** The `KeepAlive` option is macOS-only (launchd). On Windows, use `OnFailure: "restart"` (default behavior with kardianos).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| macOS plist generation | `~/Library/LaunchAgents/com.agenthub.daemon.plist` template | `kardianos/service` install | Plist has 15+ relevant keys; incorrect `RunAtLoad`/`KeepAlive` interaction causes silent non-starts |
| Linux systemd unit | `~/.config/systemd/user/agenthub-daemon.service` | `kardianos/service` install | Requires `WantedBy=default.target`, correct `ExecStart` quoting, `systemctl --user enable` call |
| Windows SCM registration | Win32 API `CreateService` / `ChangeServiceConfig` | `kardianos/service` install | Windows SCM API is C-style, requires SE_SERVICE_LOGON_NAME privileges, error-prone |
| Cross-platform `start`/`stop` | `launchctl start`, `systemctl --user start`, `sc start` | `service.Control(s, "start/stop")` | Platform command syntax and error codes differ; kardianos normalizes them |

**Key insight:** Platform service APIs are semantically similar but syntactically incompatible. The abstraction cost of hand-rolling is high; edge cases (launchd `KeepAlive` overriding `RunAtLoad`, systemd requires `loginctl enable-linger` for non-GUI Linux sessions, Windows SCM requires `GENERIC_WRITE` access) make correctness difficult without the library.

---

## Common Pitfalls

### Pitfall 1: `RunDaemon()` Called from Within `service.Run()`

**What goes wrong:** The existing `RunDaemon()` calls `signal.NotifyContext` and blocks on `<-ctx.Done()`. If you call `RunDaemon()` directly inside `daemonSvc.Start()` (blocking the Start callback), `kardianos/service` will hang waiting for `Start()` to return. The service will appear to start but the OS service manager will time out.

**Why it happens:** `service.Interface.Start()` must return quickly (within a few seconds on Windows, immediately on macOS/Linux). The actual work must run in a goroutine.

**How to avoid:** Extract the daemon core (socket setup, API start, relay start) into a `runDaemonWithContext(ctx context.Context)` function. `RunDaemon()` retains its signal-based wrapper for non-service use; `daemonSvc.Start()` launches `runDaemonWithContext` in a goroutine.

**Warning signs:** Service manager logs "service did not respond to control function" or "Timeout (30000 milliseconds) waiting for a transaction response from the service."

### Pitfall 2: `agenthub daemon` Subcommand Conflict

**What goes wrong:** The existing `main.go` has `if cmd == "daemon" { daemon.RunDaemon(); return }` before any EnsureDaemon call. When `install/uninstall/start/stop` are added under `daemon`, they must be dispatched before `RunDaemon()` is called. If not, `agenthub daemon install` would call `RunDaemon()` and block forever.

**Why it happens:** The current code treats `daemon` as a single-action subcommand (run the daemon). Phase 23 turns it into a multi-action subcommand.

**How to avoid:** Change the `daemon` dispatch block to parse the next argument (`os.Args[2]`) before deciding whether to call `RunDaemon()` or `ServiceControl()`. Guard with a check: if no sub-subcommand, default to `run` for backward compatibility with `EnsureDaemon`'s `startDetachedDaemon` call.

**Warning signs:** `agenthub daemon install` hangs at the terminal instead of printing a success message.

### Pitfall 3: Windows SCM — Executable Must Be Absolute Path

**What goes wrong:** If `Config.Executable` resolves to a relative path or a symlink on Windows, the SCM will fail to find the binary after reboot (PATH may differ in Session 0).

**Why it happens:** Windows services run in a minimal environment. The `PATH` variable does not include user directories.

**How to avoid:** Use `filepath.Abs(exe)` where `exe = os.Executable()` at install time. Log the resolved path during install so users can verify.

**Warning signs:** Service installs successfully but silently fails to start on next reboot; Windows Event Log shows "The system cannot find the file specified."

### Pitfall 4: macOS `KeepAlive` Behavior After `stop`

**What goes wrong:** With `KeepAlive: true` in the launchd plist, `launchctl stop com.agenthub.daemon` (or `service.Control(s, "stop")`) will cause launchd to immediately restart the daemon. The user sees the service stop for ~1 second then restart.

**Why it happens:** `KeepAlive` tells launchd to always keep the process running. It overrides manual stops.

**How to avoid:** For `agenthub daemon stop`, do NOT use `KeepAlive: true` if you want stop to be respected. Use `KeepAlive: false` and `RunAtLoad: true` instead — this gives auto-start on login without automatic restart after crash. Alternatively, `unload` the job to stop it persistently, but that is effectively `uninstall`.

**Recommended config:** `RunAtLoad: true`, `KeepAlive: false`. The daemon will start at login. If it crashes, the user must restart manually (or run `agenthub daemon start`). This is appropriate for a user-facing tool, not a critical system service.

**Warning signs:** `agenthub daemon stop` appears to succeed but daemon is still running 2 seconds later.

### Pitfall 5: Linux — systemd User Services Require Linger for Headless Environments

**What goes wrong:** On Linux systems without a graphical session (headless CI, remote servers), systemd user instances do not start until a user logs in via PAM. Services enabled with `systemctl --user enable` will not auto-start at boot unless linger is enabled.

**Why it happens:** By design — systemd user instances are per-session, not per-boot.

**How to avoid:** For the typical developer workstation use case, this is acceptable — the daemon starts on GUI/SSH login. Document this behavior. Do not require `loginctl enable-linger` for basic functionality; it requires root if enabling for another user.

**Warning signs:** "Failed to connect to bus: No such file or directory" when running `systemctl --user` from a non-login shell.

### Pitfall 6: `EnsureDaemon` Double-Start Race

**What goes wrong:** If the daemon is installed as a service with auto-start, the daemon is already running when any CLI command is run. `EnsureDaemon` will detect it (health check succeeds) and skip spawning — correct behavior. However, if the daemon binary is updated and restarted while a CLI attach session is active, `EnsureDaemon` will see the old daemon is unreachable and spawn a second process, which will fail to bind the socket.

**Why it happens:** `EnsureDaemon` uses `CleanupStaleSocket` which removes the socket file if no process is listening. The new service-managed instance is already starting via launchd/systemd and may race to re-create the socket.

**How to avoid:** This is an existing edge case, not new to Phase 23. Document it as a known limitation. The service manager handles restart timing; `CleanupStaleSocket` will correctly detect and remove the stale socket file before the new instance starts.

---

## Code Examples

### Service Config Factory

```go
// Source: pkg.go.dev/github.com/kardianos/service
func newServiceConfig() (*service.Config, error) {
    exe, err := os.Executable()
    if err != nil {
        return nil, fmt.Errorf("resolve executable: %w", err)
    }
    exe, err = filepath.Abs(exe)
    if err != nil {
        return nil, fmt.Errorf("abs executable: %w", err)
    }
    return &service.Config{
        Name:        "agenthub-daemon",
        DisplayName: "AgentHub Daemon",
        Description: "AgentHub session manager daemon",
        Executable:  exe,
        Arguments:   []string{"daemon"},
        Option: service.KeyValue{
            "UserService": true,
            "RunAtLoad":   true,
            "KeepAlive":   false,
        },
    }, nil
}
```

### Install / Uninstall / Start / Stop

```go
// Source: pkg.go.dev/github.com/kardianos/service
func ServiceControl(action string) error {
    cfg, err := newServiceConfig()
    if err != nil {
        return err
    }
    prg := &daemonSvc{}
    s, err := service.New(prg, cfg)
    if err != nil {
        return fmt.Errorf("service.New: %w", err)
    }
    if err := service.Control(s, action); err != nil {
        return fmt.Errorf("daemon %s: %w", action, err)
    }
    return nil
}
```

### daemonSvc Interface Implementation

```go
// Source: kardianos/service README
type daemonSvc struct {
    cancel context.CancelFunc
    done   chan struct{}
}

func (d *daemonSvc) Start(s service.Service) error {
    ctx, cancel := context.WithCancel(context.Background())
    d.cancel = cancel
    d.done = make(chan struct{})
    go func() {
        defer close(d.done)
        runDaemonCore(ctx)
    }()
    return nil
}

func (d *daemonSvc) Stop(s service.Service) error {
    if d.cancel != nil {
        d.cancel()
    }
    if d.done != nil {
        <-d.done
    }
    return nil
}
```

### Refactored RunDaemon

```go
// RunDaemon is the non-service entry point (spawned by EnsureDaemon or run directly).
// It calls runDaemonCore with a signal-based context.
func RunDaemon() {
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGTERM, syscall.SIGINT)
    defer stop()
    runDaemonCore(ctx)
}

// runDaemonCore is the shared daemon logic used by both RunDaemon and daemonSvc.
func runDaemonCore(ctx context.Context) {
    socketPath := DefaultSocketPath()
    if err := CleanupStaleSocket(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
        return
    }
    engine := NewSessionEngine()
    api := NewAPI(engine)

    relayPort, err := api.StartRelay()
    if err != nil {
        fmt.Fprintf(os.Stderr, "daemon: start relay: %v\n", err)
        return
    }
    fmt.Fprintf(os.Stderr, "daemon: relay listening on port %d\n", relayPort)

    if err := api.Start(socketPath); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: start api: %v\n", err)
        return
    }
    fmt.Fprintf(os.Stderr, "daemon: listening on %s\n", socketPath)

    <-ctx.Done()
    fmt.Fprintf(os.Stderr, "daemon: shutting down\n")
    _ = api.Stop()
    engine.Manager().Shutdown()
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Double-fork daemonization (Unix) | Foreground process; service manager owns lifecycle | systemd ~2010, launchd ~2005 | Service managers expect foreground processes; double-fork breaks supervision |
| System-wide service (root required) | User-scoped service (LaunchAgents, systemd user, SCM user) | macOS 10.4+, systemd user units ~2013 | No root/sudo required for install; service runs with user's permissions |
| `launchctl load`/`unload` | `launchctl bootstrap`/`bootout` (macOS 10.10+) | macOS 10.10 Yosemite | `load`/`unload` deprecated but still works; kardianos may use old API |

**Deprecated/outdated:**

- `launchctl load`/`unload`: Deprecated on macOS 10.10+, replaced by `bootstrap`/`bootout`. `kardianos/service` uses `launchctl load` internally as of v1.2.4 — verify this works on macOS 15 during testing.
- `SysV init scripts`: Still supported by `kardianos/service` but irrelevant for modern Ubuntu/Fedora/Arch (all use systemd).

---

## Open Questions

1. **`launchctl load` vs `bootstrap` on macOS 15 Sequoia**
   - What we know: `kardianos/service` v1.2.4 (2025-07-14) exists; macOS 14/15 still accepts `launchctl load` for user agents.
   - What's unclear: Whether any macOS 15-specific deprecation warnings or failures occur with `launchctl load`.
   - Recommendation: Run `agenthub daemon install` and `agenthub daemon uninstall` manually on the dev machine (macOS) as part of verification. Check for warnings in Console.app.

2. **Windows CI for SCM testing**
   - What we know: STATE.md notes "establish Windows CI during Phase 19 before Phase 23 makes it critical." The build includes `process_windows.go` — cross-compilation works but runtime testing requires a Windows environment.
   - What's unclear: Whether the project has a Windows test environment available.
   - Recommendation: Implement Windows SCM code using `kardianos/service`, compile-gate it with `go build -o /dev/null ./...` targeting `GOOS=windows`, and add a `//go:build windows` test that verifies the `ServiceControl` function calls compile. Runtime integration testing on Windows can be manual or deferred.

3. **`agenthub daemon` backward compatibility with `EnsureDaemon`'s `startDetachedDaemon`**
   - What we know: `startDetachedDaemon` spawns `exe daemon` (no subcommand). After Phase 23, `daemon` becomes a multi-action subcommand requiring `daemon run` or `daemon install` etc.
   - What's unclear: Whether to keep bare `daemon` as an alias for `daemon run`.
   - Recommendation: Keep `agenthub daemon` (no further args) as equivalent to `agenthub daemon run` — it calls `RunDaemon()` directly. This preserves backward compatibility with `EnsureDaemon`'s spawn call.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | `testing` (stdlib) + `go test` |
| Config file | none — standard Go test runner |
| Quick run command | `go test ./internal/daemon/ ./cmd/agenthub-cli/ -run TestSvc -timeout 30s` |
| Full suite command | `go test ./... -timeout 120s` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SVC-01 | `ServiceControl("install")` returns no error; platform artifact created | unit (mock) | `go test ./internal/daemon/ -run TestServiceControl_Install -timeout 30s` | ❌ Wave 0 |
| SVC-01 | `ServiceControl("uninstall")` returns no error after install | unit (mock) | `go test ./internal/daemon/ -run TestServiceControl_Uninstall -timeout 30s` | ❌ Wave 0 |
| SVC-02 | Service config sets RunAtLoad=true in all platforms | unit | `go test ./internal/daemon/ -run TestServiceConfig_RunAtLoad -timeout 30s` | ❌ Wave 0 |
| SVC-03 | `cmdDaemon(["install"])` dispatches to `ServiceControl("install")` | unit | `go test ./cmd/agenthub-cli/ -run TestCmdDaemon -timeout 30s` | ❌ Wave 0 |
| SVC-03 | `cmdDaemon(["start"])` dispatches to `ServiceControl("start")` | unit | `go test ./cmd/agenthub-cli/ -run TestCmdDaemon -timeout 30s` | ❌ Wave 0 |
| SVC-01/02/03 | End-to-end install/start/stop/uninstall on macOS | manual/integration | N/A — requires OS service manager | manual |

**Note on real service manager tests:** Installing/uninstalling a real launchd service in CI requires a macOS runner with full user context. Unit tests should mock `service.Control()` via dependency injection on a `ServiceController` interface. The integration tests (actual `launchctl` registration) are manual during verification.

### Sampling Rate

- **Per task commit:** `go test ./internal/daemon/ ./cmd/agenthub-cli/ -timeout 30s`
- **Per wave merge:** `go test ./... -timeout 120s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/daemon/service.go` — `runDaemonCore`, `daemonSvc`, `newServiceConfig`, `ServiceControl`
- [ ] `internal/daemon/service_test.go` — unit tests for config factory, svc wrapper Start/Stop
- [ ] `cmd/agenthub-cli/cmd_daemon.go` — `cmdDaemon` dispatcher
- [ ] `go.sum` update after `go get github.com/kardianos/service@v1.2.4`

---

## Sources

### Primary (HIGH confidence)

- `pkg.go.dev/github.com/kardianos/service` — Config struct, Interface, Control(), option keys, platform support
- `go list -m github.com/kardianos/service@latest` — confirmed v1.2.4, published 2025-07-14
- `launchd.info` — macOS launchd RunAtLoad, KeepAlive semantics
- `wiki.archlinux.org/title/Systemd/User` — systemd user unit file location, enable procedure
- `developer.apple.com` (Apple launchd documentation) — LaunchAgents plist location `~/Library/LaunchAgents/`

### Secondary (MEDIUM confidence)

- WebSearch cross-referenced with pkg.go.dev: Windows SCM behavior with `StartType: "automatic"`, Session 0 isolation, per-user service behavior

### Tertiary (LOW confidence)

- macOS 15 compatibility with `launchctl load` (deprecated API) — not verified against current macOS release; flag for manual testing
- Windows CI availability for runtime SCM testing — stated as pending in STATE.md, not confirmed

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — `kardianos/service` version confirmed via `go list`, pkg.go.dev checked for full API
- Architecture: HIGH — patterns follow official library examples; refactor of `RunDaemon` is straightforward given existing code
- Pitfalls: HIGH for KeepAlive/RunAtLoad interaction (verified in launchd docs); MEDIUM for Windows SCM path issues (standard practice, not project-specific); MEDIUM for macOS 15 `launchctl load` deprecation (needs empirical validation)

**Research date:** 2026-03-24
**Valid until:** 2026-04-24 (stable library, 30 days)
