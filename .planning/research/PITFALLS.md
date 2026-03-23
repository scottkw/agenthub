# Pitfalls Research

**Domain:** CLI + Daemon — adding daemon mode, CLI commands, and terminal attach/detach to an existing Go/Wails desktop app
**Researched:** 2026-03-23
**Confidence:** HIGH for daemon/process/signal pitfalls (verified against official docs and GitHub issues); MEDIUM for Wails-specific coexistence (limited documented precedent, based on Wails source and community issues)

---

## Critical Pitfalls

### Pitfall 1: Wails Initialization Runs Before os.Args Dispatch

**What goes wrong:**
Wails v2's binding generation phase executes `main()` during the build process without the user's CLI arguments. If `main()` dispatches on `os.Args` before calling `wails.Run()`, any argument validation or subcommand routing that panics or calls `os.Exit` will break the build entirely. Even at runtime, the macOS `.app` bundle launch path does not pass arguments through the same way a terminal invocation does — double-clicking the app passes no arguments, but the binary must still route correctly to GUI mode.

**Why it happens:**
Developers assume `os.Args` dispatch is safe at the top of `main()`. In Wails v2, the framework calls the compiled binary during code generation, so `main()` runs twice: once during build (without args) and once for real (with args). Additionally, macOS `.app` bundles launched by the Finder do not pass CLI arguments — the binary receives only `os.Args[0]`.

**How to avoid:**
Dispatch on `os.Args` at the very top of `main()`, before any Wails imports are initialized, but use a safe default: if `len(os.Args) == 1` (no subcommand), fall through to `wails.Run()`. Never panic or exit on missing arguments in the dispatch path. Use `//go:build production` constraints to guard any argument validation that is only safe at runtime. Test the no-args path explicitly.

**Warning signs:**
- `wails build` fails with a panic or non-zero exit in code that "should never run"
- Double-clicking the `.app` bundle crashes immediately rather than showing the window
- Argument flags accepted by the CLI conflict with internal Wails flags (e.g. `-loglevel`)

**Phase to address:** Daemon binary architecture / single-binary mode dispatch (first phase of v1.3)

---

### Pitfall 2: Stale Unix Socket Blocks Daemon Startup

**What goes wrong:**
The daemon creates a Unix socket for IPC. If the daemon crashes, is killed with SIGKILL, or the machine reboots without a clean shutdown, the socket file remains on disk. The next daemon startup attempt calls `net.Listen("unix", socketPath)` and gets `address already in use`, refusing to start even though no daemon is actually running.

**Why it happens:**
Unix sockets are filesystem objects. Unlike TCP ports, the OS does not automatically reclaim a socket file when a process dies. The file persists until explicitly removed. This is one of the most common daemon startup failures and has thousands of hits in forums — there is no OS-level automatic cleanup.

**How to avoid:**
At startup, before calling `net.Listen`, attempt to connect to the socket. If the connect succeeds, another daemon is already running — exit with an "already running" error. If the connect fails (connection refused or no such file), the socket is stale — remove it with `os.Remove(socketPath)` and then listen. Use a lock file (via `flock` on POSIX) as the authoritative running-state indicator rather than the socket itself, because lock files are immune to PID reuse. On startup: check lock file → check PID liveness → remove stale socket → listen.

**Warning signs:**
- `agenthub daemon start` fails with "address already in use" after a crash
- Socket file exists at the expected path but no process holds the lock
- Users report having to manually `rm` the socket file to restart the daemon

**Phase to address:** Daemon process management (PID files, socket lifecycle)

---

### Pitfall 3: Terminal Left in Raw Mode After Crash or Panic

**What goes wrong:**
When the CLI attaches to a session, it calls `term.MakeRaw(os.Stdin.Fd())` to enter raw terminal mode. If the process panics, receives SIGKILL, or exits via `os.Exit` without running deferred cleanup, the terminal is left in raw mode. The user's shell prompt becomes invisible — keystrokes are not echoed, Enter does not submit commands, and the terminal appears frozen. The user must type `reset` blind to recover.

**Why it happens:**
`defer term.Restore(...)` only runs on normal function returns and panics that unwind the stack. It does NOT run when: the process receives SIGKILL, `os.Exit` is called, `log.Fatal` is called (which calls `os.Exit`), or the program crashes due to a nil pointer dereference that goes unrecovered. Signal handlers for SIGTERM and SIGINT must also call restore before exiting.

**How to avoid:**
Save the original terminal state immediately: `oldState, _ := term.GetState(fd)`. Register signal handlers for SIGTERM, SIGINT, and SIGHUP that call `term.Restore(fd, oldState)` before `os.Exit`. Wrap the attach loop in a `recover()` that restores terminal state before re-panicking. Never call `log.Fatal` from within raw mode — capture the error, restore first, then log and exit. On Windows, `SetConsoleMode` must be similarly restored via a deferred call that is also wired through signal handlers.

**Warning signs:**
- Users report "broken terminal" or "invisible typing" after `agenthub attach` exits abnormally
- CI/CD attach tests leave the test runner's terminal in raw mode (causing subsequent test output to be garbled)
- Any `log.Fatal` call path reachable from attach mode

**Phase to address:** Terminal attach / raw mode (CLI attach command implementation)

---

### Pitfall 4: Daemon Does Not Daemonize on macOS (launchd Conflict)

**What goes wrong:**
Traditional Unix daemons double-fork to detach from the controlling terminal and become session leaders. On macOS, launchd expects to be the parent of the service process — if the daemon double-forks, launchd loses track of the PID, cannot monitor the process, and the service appears to crash immediately after "starting." Launchd then enters rapid-restart loops.

**Why it happens:**
Developers familiar with Linux daemon patterns (POSIX double-fork, `setsid()`, daemonize libraries) apply them to macOS launchd agents. The launchd contract is that the registered binary runs in the foreground; launchd handles backgrounding. The `go-daemon` library's fork approach also conflicts with Go's runtime, which does not support `fork()` without `exec()` — forked child processes in Go do not inherit goroutines.

**How to avoid:**
The daemon must NOT self-daemonize. Run in the foreground. Let launchd (macOS), systemd (Linux), and Windows SCM manage the process lifecycle. Use `kardianos/service` as an abstraction — it implements the correct Run/Stop pattern for each platform without requiring platform-specific fork/setsid code. The plist `RunAtLoad` key and `KeepAlive` key handle restarts. Never use `go-daemon`'s fork-based approach in a Wails project.

**Warning signs:**
- launchd shows the service as "crashed" immediately after `launchctl start`
- `launchctl list | grep agenthub` shows the service with a non-zero exit code
- Service appears to start and stop in rapid succession in Console.app

**Phase to address:** Service manager integration (launchd/systemd/Windows SCM)

---

### Pitfall 5: PTY Resize Events Not Propagated Through the Daemon Proxy

**What goes wrong:**
When a CLI client attaches to a session via the daemon, the daemon proxies PTY I/O over the Unix socket. Terminal resize events (SIGWINCH on POSIX) are delivered to the CLI process, not to the daemon process that owns the PTY. If the CLI attach handler does not forward resize events through the socket to the daemon, which then calls `pty.Resize()` on the backing PTY, the AI coding CLI's UI will misrender after any window resize — wrapped lines, broken layouts, overwritten output.

**Why it happens:**
PTY resize requires calling the `TIOCSWINSZ` ioctl on the master PTY file descriptor. The master FD lives in the daemon process. The CLI client receives SIGWINCH, but it does not own the master FD. Developers implement the I/O proxy but forget the resize channel, because it works fine during initial testing in a fixed-size terminal.

**How to avoid:**
The IPC protocol must include a resize message type (already partially present in the existing binary framing protocol). On the client side, install a SIGWINCH handler that reads the current terminal dimensions with `unix.IoctlGetWinsize` and sends a resize frame to the daemon. The daemon receives the resize frame and calls the PTY resize API. Test explicitly by resizing the terminal window during an active attach session. Note: Windows uses `ConPTY` resize via a separate call — handle with a build tag.

**Warning signs:**
- AI CLI renders correctly at initial attach but breaks after window resize
- Line wrapping artifacts visible in the terminal output
- `stty size` inside the attached session returns wrong dimensions after resize

**Phase to address:** Terminal attach / PTY proxy (CLI attach command implementation)

---

### Pitfall 6: Session State Lives in Two Places After Extraction

**What goes wrong:**
Currently `App` in `app.go` holds session state (registry, tabNames, cliPaths, sessionStatuses) directly. When a daemon is extracted, session state must move to the daemon. But if the GUI still maintains a local copy — even a cache — the two diverge: the GUI shows stale session names, incorrect status, or phantom sessions that the daemon has already killed. Race conditions between the GUI's local state and the daemon's authoritative state cause confusing bugs that are hard to reproduce.

**Why it happens:**
The natural refactoring path is to "add a daemon" while keeping the existing App struct intact. Developers add RPC calls for new operations but leave old direct mutations for "performance" or "it works locally." The first divergence bug appears only after a multi-client scenario (GUI + CLI both attached) or after the daemon kills a session the GUI doesn't know about.

**How to avoid:**
The daemon is the single source of truth for ALL session state. The GUI is a client, not a peer. After extraction, `App` holds no session state — it only holds the IPC connection to the daemon and a local display cache that is invalidated and refreshed via daemon events (push notifications or periodic polling). The `tabNames` map, `sessionStatuses` map, and all other mutable session fields must live exclusively in the daemon. The GUI's `App.startup` connects to (or starts) the daemon and subscribes for state change events.

**Warning signs:**
- GUI shows a session as "running" that the CLI's `list` command shows as "stopped"
- Renaming a tab in the GUI doesn't appear in CLI's `list` output
- Creating a session via CLI doesn't appear in the GUI until restart

**Phase to address:** Daemon extraction / session state migration (first phase, architectural decision)

---

### Pitfall 7: macOS App Sandbox Blocks Unix Socket IPC

**What goes wrong:**
macOS App Sandbox (required for Mac App Store distribution and sometimes applied by notarization tooling) blocks Unix domain socket access between processes from different teams. The GUI app (sandboxed) cannot reach the daemon's socket at a path outside the app container. Additionally, `sun_path` in `sockaddr_un` is limited to 104 characters on macOS — paths under `XDG_RUNTIME_DIR` or long username paths can silently exceed this, causing `bind: invalid argument` with no clear error message.

**Why it happens:**
Unix socket entitlements in the App Sandbox require `com.apple.security.temporary-exception.files.absolute-path.read-write`, but this entitlement specifically does NOT cover Unix domain sockets — only regular files. Network framework restrictions and the 104-character limit are also not surfaced clearly by the compiler or runtime.

**How to avoid:**
For the v1.3 milestone (not targeting Mac App Store), sandbox is not active. But the socket path must be kept under 104 characters. Use `os.UserCacheDir()` or a path like `~/.config/agenthub/daemon.sock` — verify the length does not exceed 104 chars even for long usernames. On macOS 13+, consider the `ServiceManagement` framework for service registration to avoid launchd plist complexity. Add a path length assertion at startup that panics with a clear message rather than a cryptic `bind: invalid argument`.

**Warning signs:**
- `bind: invalid argument` on the socket listen call with no other error details
- Socket path length is over 100 characters when username is included
- App works for short usernames but fails for users with longer home directory paths

**Phase to address:** Daemon IPC design (socket path selection and validation)

---

### Pitfall 8: Windows Named Pipe vs Unix Socket IPC Split

**What goes wrong:**
Windows does not support Unix domain sockets in Go's `net.Listen("unix", ...)` on older builds (pre-Windows 10 1803), and even on supported versions, permissions and path conventions differ from POSIX. Attempting to use a single Unix socket path on all platforms either breaks on Windows or produces confusing `\\.\pipe\` path handling. If the IPC layer is not abstracted from the start, adding Windows named pipe support later requires touching every call site.

**Why it happens:**
Development happens on macOS/Linux first. The Unix socket path works perfectly. Windows is "handled later." By then, the IPC protocol, connection logic, and socket path resolution are spread across the codebase.

**How to avoid:**
Abstract the transport behind an interface from day one: `type Transport interface { Listen() (net.Listener, error); Dial() (net.Conn, error) }`. On macOS/Linux, implement with `net.Listen("unix", ...)`. On Windows, implement with a named pipe (using `npipe` or `winio`). The IPC protocol over the transport is identical — only the transport layer differs. Use `//go:build` tags to select the correct implementation per platform. The socket path resolver must also be platform-aware: `~/.config/agenthub/daemon.sock` on POSIX, `\\.\pipe\agenthub-daemon` on Windows.

**Warning signs:**
- IPC connection works on macOS/Linux but fails silently on Windows
- Windows build shows `address family not supported` errors
- Hard-coded socket paths with `/` separators in Windows-targeting code

**Phase to address:** Daemon IPC design (transport abstraction), verified in Windows CI

---

### Pitfall 9: Signal Handling Does Not Account for Forwarded Signals in the Proxy Chain

**What goes wrong:**
When the CLI client is in raw attach mode, Ctrl-C (SIGINT) should be forwarded to the AI coding CLI process inside the daemon, not terminate the CLI client itself. But Go's `os/signal.Notify(sigChan, syscall.SIGINT)` intercepts Ctrl-C before it propagates. If the CLI client exits on SIGINT instead of forwarding it as a PTY input byte (0x03) to the session, the user cannot interrupt the AI CLI's current operation without detaching first.

**Why it happens:**
Go's default SIGINT handling terminates the process. Developers override this with `signal.Notify` for graceful shutdown, but in raw PTY proxy mode the correct behavior is to pass Ctrl-C through as a byte to the slave PTY — the receiving process should handle SIGINT. The distinction between "Ctrl-C to stop my attach client" vs "Ctrl-C to interrupt the AI CLI" is only possible via the detach prefix (e.g. Ctrl-B d to detach first).

**How to avoid:**
In raw PTY proxy mode, do NOT install SIGINT as a shutdown signal. Instead, forward all bytes from stdin directly to the PTY write. The detach sequence (e.g. Ctrl-B followed by `d`) must be intercepted in the byte stream BEFORE writing to the PTY — implement a small state machine that watches for the prefix key. Only after detach completes should normal signal handling resume. SIGTERM and SIGHUP (terminal close) should still trigger a clean detach + terminal restore.

**Warning signs:**
- Ctrl-C in attach mode exits the `agenthub attach` process instead of interrupting the AI CLI
- Users have no way to interrupt a running AI command without closing the terminal window
- The detach prefix key accidentally interrupts the AI CLI if the key byte passes through

**Phase to address:** Terminal attach / signal proxying (CLI attach command implementation)

---

### Pitfall 10: Daemon Startup Race — GUI Starts Before Daemon Is Ready

**What goes wrong:**
The GUI app starts the daemon as a subprocess (if not already running) and immediately tries to connect to the IPC socket. The daemon needs a non-trivial amount of time to create its socket, initialize the session registry, and start listening. The GUI's connect attempt fails because the socket does not exist yet. If the retry logic is naive (fixed sleep), it is either too short (race on slow machines) or too long (bad startup UX on fast machines).

**Why it happens:**
"Start subprocess then connect" patterns use `time.Sleep(500ms)` as the synchronization mechanism. This is brittle across machines and configurations. On a fast developer machine it always works; on a user's older hardware or at login time when disk I/O is contended, it fails intermittently.

**How to avoid:**
Use exponential backoff with a deadline: try to connect every 50ms, doubling the interval, up to a 5-second total timeout. The connect attempt itself is cheap — if the socket doesn't exist yet, the error is immediate. This converges in ~100ms on fast machines and still works on slow ones. Alternatively, have the daemon write a sentinel file (distinct from the socket) when it is fully ready. The GUI watches for the sentinel before attempting the socket connection.

**Warning signs:**
- "Connection refused" errors logged at GUI startup on some machines but not others
- App initialization always works in development but occasionally fails on user machines
- Startup failures are more common immediately after login (system under load)

**Phase to address:** GUI-daemon connection bootstrap (GUI integration with daemon)

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Keep session state in App struct, duplicate to daemon | Avoid refactoring App | State divergence, dual-source bugs, GUI shows stale data | Never — migrate cleanly |
| Fixed `time.Sleep` for daemon startup sync | Simple code | Flaky on slow machines, bad UX at login | Never — use retry with deadline |
| Skip Windows named pipe abstraction, use TCP localhost | Works everywhere, fast | Security hole (any local process can connect), no auth on loopback | Only as temporary scaffold in CI |
| Single Unix socket path without length check | Simpler code | Cryptic `bind: invalid argument` for users with long usernames | Never — add assertion |
| `log.Fatal` inside raw-mode attach path | Simple error handling | Terminal left in raw mode, user sees broken shell | Never in raw mode path |
| Skip SIGWINCH handling for MVP attach | Faster to ship | All users with non-fixed terminal windows see rendering bugs | Never — too visible |
| Inline all CLI subcommands in main.go | Simple file structure | Untestable, giant file, hard to add subcommands | Never — use cobra or manual dispatch with clear boundaries |
| Use `os.Kill(pid)` to check if daemon is running | Simple PID check | PID reuse: kills unrelated process with recycled PID | Never — use lock file |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| launchd plist | Setting `RunAtLoad` without `KeepAlive` | Use `KeepAlive: true` for daemon resilience; `RunAtLoad` alone starts once but doesn't restart on crash |
| launchd plist | Hard-coding absolute binary path | Use `CFBundleIdentifier`-relative path or derive from `os.Executable()` at install time; app location may change |
| launchd plist | Calling `launchctl load` (deprecated) | Use `launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/plist` on macOS 10.11+ |
| systemd unit | Not setting `Type=simple` for foreground daemon | Without explicit Type, systemd may misidentify daemon state; `Type=simple` + `Restart=on-failure` is correct for a foreground Go binary |
| Windows SCM | Writing to stdout/stderr directly | Windows services have no console; use `kardianos/service` logger which routes to Event Log; direct fmt.Print output is lost |
| Windows SCM | Using `os.Signal` (SIGTERM) for shutdown | Windows SCM sends a stop command, not SIGTERM; `kardianos/service`'s `Stop()` method is the correct handler |
| go-pty (aymanbagabas) | Calling Resize on closed PTY | The PTY may close asynchronously when the child exits; always check for nil and recover from the resize call |
| IPC protocol | Sending string session IDs without length prefix | Stream corruption when session ID contains rare bytes that collide with frame delimiters; use the existing binary framing |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Full session list scan on every CLI command | CLI `list` is fast with 3 sessions | O(n) list operations become slow; registry already uses a map | At 50+ simultaneous sessions (unlikely but not impossible) |
| Synchronous IPC calls for every GUI refresh | No visible lag with 1 client | GUI stutter when daemon is under load | When CLI client is attached and hammering input simultaneously with GUI refresh |
| Scrollback replay on every new GUI attach | Fast with short sessions | Re-sending MB of scrollback over Unix socket blocks the attach path | Sessions running > 30 minutes with high output volume; already use bounded scrollback |
| Polling daemon status every 100ms from GUI | Unnoticeable at idle | Unnecessary CPU wake-ups prevent App Nap on macOS | Always — use push notification or 1s polling minimum |

---

## Security Mistakes

Domain-specific security issues.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Unix socket world-readable (0777 permissions) | Any local user can send commands to the daemon, kill sessions, start new ones | Create socket with `0600` permissions; `os.Chmod(socketPath, 0600)` immediately after `net.Listen` |
| Not verifying daemon binary before connecting | Malicious process squats on socket path, GUI connects to it | In same-user single-daemon model, socket path under `~/.config/agenthub/` is sufficient; document the threat model |
| Daemon running as root for service installation | Root daemon accepts commands from non-root GUI with no auth | Daemon must run as the logged-in user (LaunchAgent not LaunchDaemon); user-space daemon only |
| Passing CLI command arguments through shell expansion | `agenthub new --cwd "$HOME"` injection via crafted directory names | Pass all arguments as `[]string` to `exec.Cmd`, never via shell; the existing `pty.NewNativePTYBackend` already does this |

---

## UX Pitfalls

Common user experience mistakes specific to this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Detach prefix key not shown anywhere in CLI output | Users don't know how to detach; resort to killing the terminal | Print `[Attached. Detach with Ctrl-B d]` on connect; configurable via settings |
| `attach` with no visual confirmation of session identity | User doesn't know which session they attached to | Print session ID, name, and CLI type as a banner before entering raw mode |
| Daemon errors surfaced as "connection refused" | Users see cryptic socket error, not actionable message | Translate socket errors into human messages: "Daemon is not running. Start it with: agenthub daemon start" |
| `agenthub kill` with no confirmation prompt | Accidentally killing the wrong session is irreversible | Require `--force` flag or print "Killed session 'my-session' (claude)" as confirmation |
| CLI exit codes not propagating daemon errors | Scripts using `agenthub` can't detect failures | Every CLI command must exit non-zero on daemon error; test exit codes in CI |
| Service install requiring manual plist editing | Users give up on auto-start | `agenthub service install` / `agenthub service uninstall` commands handle plist/unit generation |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Stale socket cleanup:** Looks done when `os.Remove` is called at startup — verify it also handles the case where the socket file does not exist (double-remove is an error; ignore `os.IsNotExist`)
- [ ] **Graceful shutdown:** Looks done when SIGTERM handler calls cancel() — verify active PTY sessions are drained (not killed mid-write) and the scrollback buffer is flushed before exit
- [ ] **Terminal raw mode restore:** Looks done when `defer term.Restore` is present — verify SIGTERM, SIGINT, and SIGHUP signal handlers ALSO call restore before exiting, not just defer
- [ ] **Daemon auto-start via service manager:** Looks done when plist/unit file is written — verify `agenthub service install` works for a fresh-install user, not just a developer who already has the binary on PATH
- [ ] **Cross-platform IPC:** Looks done on macOS/Linux — verify Windows named pipe implementation exists, is tested, and CI covers it; Unix socket code must not compile on Windows
- [ ] **Session state migration:** Looks done when daemon has a registry — verify GUI `App` struct no longer holds any authoritative session state (tabNames, sessionStatuses) and that all writes go through the daemon IPC, not a local map
- [ ] **PTY resize propagation:** Looks done when SIGWINCH handler fires — verify the resize frame is sent over the IPC socket, the daemon applies it to the correct session's PTY, and the AI CLI re-renders at the new size
- [ ] **Wails + CLI mode dispatch:** Looks done when `os.Args` check is present — verify a no-args invocation from Finder/desktop launcher still opens the GUI window, not a "missing command" error

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Session state divergence (GUI + daemon out of sync) | HIGH | Stop GUI, restart daemon, restart GUI; investigate which mutation path bypassed IPC and add the missing RPC call |
| Terminal left in raw mode | LOW | User types `reset` or `stty sane` blindly; add recovery note to CLI help text; fix signal handler in next patch |
| Stale socket prevents daemon start | LOW | `rm ~/.config/agenthub/daemon.sock && agenthub daemon start`; add `agenthub daemon restart --force` command that removes stale socket automatically |
| launchd rapid-restart loop | MEDIUM | `launchctl stop com.agenthub.daemon` then diagnose; check Console.app for exit reason; usually double-fork or stdout/stderr to terminal |
| PTY resize not working | MEDIUM | Detach and re-attach (inherits current terminal size on re-attach); fix SIGWINCH forwarding in next release |
| Windows IPC broken | HIGH | Fall back to TCP localhost for Windows-only builds as emergency scaffold; prioritize named pipe fix; Windows users cannot use CLI until fixed |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Wails init runs before os.Args dispatch | Phase: Binary mode dispatch | Build succeeds with `wails build`; `./agenthub` (no args) opens GUI; `./agenthub list` prints usage |
| Stale Unix socket blocks startup | Phase: Daemon process management | Kill daemon with SIGKILL, restart; daemon starts cleanly without manual socket removal |
| Terminal left in raw mode | Phase: CLI attach command | Send SIGKILL to attach process; parent shell still echoes input correctly |
| Daemon self-daemonizes (launchd conflict) | Phase: Service manager integration | `launchctl start com.agenthub.daemon` shows service stays running in `launchctl list` |
| PTY resize not propagated | Phase: CLI attach command | Resize terminal during active attach; AI CLI re-renders without artifacts |
| Session state in two places | Phase: Daemon extraction (architectural) | GUI and CLI `list` output is identical in all multi-client scenarios |
| macOS socket path too long | Phase: Daemon IPC design | Add path length assertion; test with a 30-char username |
| Windows named pipe abstraction missing | Phase: Cross-platform IPC scaffold | Windows CI build runs daemon start/stop/connect test |
| Signal forwarding (Ctrl-C) | Phase: CLI attach command | Ctrl-C inside attach interrupts AI CLI command; does NOT exit `agenthub attach` |
| Daemon startup race | Phase: GUI-daemon integration | Run startup 50 times on a loaded machine; zero "connection refused" errors |

---

## Sources

- Wails GitHub Discussion #3098: Including a CLI with a Wails app — https://github.com/wailsapp/wails/discussions/3098
- Wails GitHub Discussion #4175: Getting CLI arguments in Wails — https://github.com/wailsapp/wails/discussions/4175
- Wails GitHub Issue #1533: `-appargs` flag conflict with argument flags — https://github.com/wailsapp/wails/issues/1533
- kardianos/service: Cross-platform service management for Go — https://github.com/kardianos/service
- Apple Developer: Creating Launch Daemons and Agents — https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html
- Apple Developer Forum: Unix socket from App Sandbox — https://developer.apple.com/forums/thread/788364
- launchd.info: Authoritative launchd tutorial — https://www.launchd.info/
- VictoriaMetrics: Graceful Shutdown in Go — https://victoriametrics.com/blog/go-graceful-shutdown/
- Go Blog: Graceful Shutdown, Signals, Contexts — https://rafalroppel.medium.com/graceful-shutdown-in-go-explained-signals-contexts-and-the-correct-shutdown-sequence-f24fd9ef8fac
- Go By Example: Signals — https://gobyexample.com/signals
- creack/pty GitHub: macOS tty kill issue after Start — https://github.com/creack/pty/issues/186
- go-tty Issue: raw mode to cooked mode failure on Linux — https://github.com/mattn/go-tty/issues/13
- XDG Base Directory Specification — https://specifications.freedesktop.org/basedir/latest/
- IPC Performance Comparison — https://www.baeldung.com/linux/ipc-performance-comparison
- Mastering Unix Domain Sockets in Go — https://dev.to/jones_charles_ad50858dbc0/mastering-unix-domain-sockets-in-go-fast-local-ipc-for-your-apps-o48
- Windows service graceful termination (SIGTERM alternatives) — https://www.codestudy.net/blog/gracefully-terminate-a-process-on-windows/

---
*Pitfalls research for: CLI + Daemon addition to Go/Wails desktop app (AgentHub v1.3)*
*Researched: 2026-03-23*
