# Phase 1: PTY Foundation - Research

**Researched:** 2026-03-17
**Domain:** Cross-platform PTY process management in Go (macOS, Linux, Windows)
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLI-01 | App detects installed AI coding CLIs (Claude Code, Codex, Gemini CLI, OpenCode) via PATH | `exec.LookPath` pattern; binary names confirmed for all four CLIs |
| CLI-02 | User can launch a new session by selecting from detected CLIs | PTY spawn via go-pty `Cmd.Start()`; environment setup with `TERM=xterm-256color` |
| TERM-06 | Terminal resizes correctly when window is resized (SIGWINCH propagation) | `pty.Resize(width, height)` on POSIX; `ResizePseudoConsole` on Windows; debounce pattern |
| TERM-07 | User can close/kill a session cleanly (process group cleanup) | `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pgid, SIGKILL)` on POSIX; Job Object on Windows |
| SESS-01 | Sessions persist when the app window is closed (Go-native PTY backend) | In-memory SessionRegistry; PTY goroutine lives in Go process independent of any UI layer |
</phase_requirements>

---

## Summary

Phase 1 delivers a Go binary that can spawn AI coding CLIs in a real PTY, read/write I/O, resize correctly, detect installed CLIs, and kill sessions cleanly. This is a pure Go backend phase with no UI — the deliverable is a tested library package that later phases build on.

The critical technical choices for this phase are already locked from project research: use `github.com/aymanbagabas/go-pty` (not `creack/pty`) for cross-platform PTY including Windows ConPTY; implement a win32-input-mode state-machine parser for correct keyboard input on Windows; use `SysProcAttr{Setpgid: true}` + process-group kill on POSIX and Windows Job Objects for clean teardown; and detect CLIs via `exec.LookPath` at startup.

The four target CLIs have confirmed binary names: `claude` (Claude Code), `codex` (OpenAI Codex), `gemini` (Gemini CLI), and `opencode` (OpenCode). All four require a proper PTY (not a pipe) — they check `isatty()` via their TUI frameworks and degrade or refuse to run without one. Setting `TERM=xterm-256color` in the spawned environment is required for correct color rendering.

**Primary recommendation:** Build the PTY layer as a standalone `internal/pty` package with a `SessionBackend` interface from day one. The NativePTYBackend implementing that interface is the only backend needed for Phase 1. This separation prevents coupling and makes Phase 2 (session registry) trivial to wire in.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/aymanbagabas/go-pty` | v0.2.2 | Cross-platform PTY spawning | Only pure-Go option with real Windows ConPTY support; `creack/pty` Windows PR #155 remains unmerged |
| `os/exec` (stdlib) | Go stdlib | Command construction | go-pty's `Cmd` type wraps `exec.Cmd`; stdlib handles env, dir, args |
| `os/signal` (stdlib) | Go stdlib | Graceful shutdown signal handling | `signal.NotifyContext` (Go 1.16+) for SIGINT/SIGTERM cancellation |
| `syscall` (stdlib) | Go stdlib | Process group kill on POSIX | `syscall.Kill(-pgid, syscall.SIGKILL)` kills entire process tree |
| `golang.org/x/sys/windows` | latest | Windows Job Object for child process cleanup | Required for `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` on Windows |
| `sync` (stdlib) | Go stdlib | Session map protection | `sync.RWMutex` for the SessionRegistry map |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/mattn/go-isatty` | latest | isatty() detection for testing | Use in integration tests to verify PTY is real; not a runtime dep |
| `context` (stdlib) | Go stdlib | PTY command lifecycle via context | `pty.CommandContext(ctx, ...)` enables context-based cancellation |
| `io` (stdlib) | Go stdlib | PTY read/write | `io.Copy`, `io.ReadWriteCloser` — PTY implements these natively |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-pty | creack/pty | creack/pty has no merged Windows ConPTY support — wrong for this project |
| go-pty | `charmbracelet/x/conpty` | conpty is Windows-only; still need a Unix PTY library; go-pty is unified |
| `syscall.Kill(-pgid)` | `cmd.Process.Kill()` | Process.Kill() kills only the top-level process, not grandchildren |
| `golang.org/x/sys/windows` | unsafe + win32 direct | x/sys/windows provides type-safe bindings |

**Installation:**

```bash
go get github.com/aymanbagabas/go-pty@v0.2.2
go get golang.org/x/sys@latest
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── pty/                 # Phase 1: all PTY logic
│   ├── backend.go       # SessionBackend interface
│   ├── native.go        # NativePTYBackend implementation
│   ├── session.go       # Session struct (id, pty, process, state)
│   ├── registry.go      # SessionRegistry (in-memory map + mutex)
│   ├── detect.go        # CLI detection via exec.LookPath
│   ├── cleanup.go       # Graceful shutdown, process group kill
│   ├── win32input.go    # win32-input-mode parser (Windows build tag)
│   └── win32input_other.go  # no-op stub for non-Windows
cmd/
└── agenthub/
    └── main.go          # Entry point: init registry, signal handler, demo loop
```

### Pattern 1: SessionBackend Interface

**What:** All PTY management goes through a `SessionBackend` interface. Phase 1 implements `NativePTYBackend`. Future phases can add `TmuxBackend` without touching calling code.

**When to use:** From day one. tmux is unavailable on Windows; the interface enforces the right abstraction boundary.

```go
// Source: derived from architecture research + go-pty API
type SessionBackend interface {
    Create(ctx context.Context, req CreateRequest) (*Session, error)
    Resize(id string, cols, rows int) error
    Kill(id string) error
    List() []*Session
}

type CreateRequest struct {
    CLI  string   // e.g. "claude", "gemini"
    Args []string
    Env  []string // merged with base env; should include TERM=xterm-256color
    Cols int
    Rows int
}
```

### Pattern 2: PTY Spawn with go-pty

**What:** Use `go-pty` to open a PTY and attach a command to it, setting required environment variables.

**When to use:** Every new session creation.

```go
// Source: pkg.go.dev/github.com/aymanbagabas/go-pty
func spawnSession(ctx context.Context, req CreateRequest) (*Session, error) {
    p, err := gopty.New()
    if err != nil {
        return nil, fmt.Errorf("open pty: %w", err)
    }

    cmd := p.CommandContext(ctx, req.CLI, req.Args...)
    cmd.Env = mergeEnv(os.Environ(), req.Env, "TERM=xterm-256color", "COLORTERM=truecolor")

    // POSIX only: create new process group for clean teardown
    // Windows: go-pty handles ConPTY; Job Object is set separately after Start()
    if runtime.GOOS != "windows" {
        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    }

    if err := cmd.Start(); err != nil {
        p.Close()
        return nil, fmt.Errorf("start process: %w", err)
    }

    // Initial resize to match requested terminal dimensions
    if err := p.Resize(req.Cols, req.Rows); err != nil {
        // non-fatal on Windows (resize before process ready is ignored)
        _ = err
    }

    return &Session{id: newID(), pty: p, cmd: cmd}, nil
}
```

### Pattern 3: PTY Resize

**What:** Call `pty.Resize(width, height)` when the frontend reports a resize event. Debounce on the caller side.

**When to use:** Every time xterm.js fires `onResize`.

```go
// Source: pkg.go.dev/github.com/aymanbagabas/go-pty (Pty interface)
func (b *NativePTYBackend) Resize(id string, cols, rows int) error {
    s, ok := b.registry.Get(id)
    if !ok {
        return ErrSessionNotFound
    }
    return s.pty.Resize(cols, rows)
    // On POSIX: calls ioctl(TIOCSWINSZ) → kernel sends SIGWINCH to child
    // On Windows: calls ResizePseudoConsole()
}
```

**Note on debounce:** Caller (not this layer) should debounce resize events at 100-200ms. On Windows, resize calls received before the ConPTY process is fully initialized are silently ignored — this is a known ConPTY bug (microsoft/terminal#10400). Sending dimensions in the initial spawn request avoids this.

### Pattern 4: Clean Session Kill (POSIX)

**What:** Kill the entire process group, not just the top-level process, to avoid orphaning AI CLI subprocesses.

**When to use:** On session close, app shutdown, or timeout.

```go
// Source: varunksaini.com/posts/kiling-processes-in-go/
//         bigkevmcd.github.io/go/pgrp/context/2019/02/19/terminating-processes-in-go.html
// Build tag: !windows

func killSession(s *Session) error {
    if s.cmd.Process == nil {
        return nil
    }
    pgid := s.cmd.Process.Pid // Setpgid: true means pgid == pid

    // SIGHUP first: signals the CLI to shut down gracefully
    _ = syscall.Kill(-pgid, syscall.SIGHUP)

    // Give the process 2 seconds to exit cleanly
    done := make(chan struct{})
    go func() {
        s.cmd.Wait()
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(2 * time.Second):
        // Force kill the entire process group
        _ = syscall.Kill(-pgid, syscall.SIGKILL)
    }

    s.pty.Close()
    return nil
}
```

### Pattern 5: Clean Session Kill (Windows)

**What:** Attach child process to a Windows Job Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. When the Job handle closes (app exit or explicit close), all attached processes terminate automatically.

**When to use:** On Windows, immediately after `cmd.Start()`.

```go
// Source: gist.github.com/hallazzang/76f3970bfc949831808bbebc8ca15209
// Build tag: windows

type jobObject struct {
    handle windows.Handle
}

func newJobObject() (*jobObject, error) {
    h, err := windows.CreateJobObject(nil, nil)
    if err != nil {
        return nil, err
    }
    info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
        BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
            LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
        },
    }
    if err := windows.SetInformationJobObject(h,
        windows.JobObjectExtendedLimitInformation,
        uintptr(unsafe.Pointer(&info)),
        uint32(unsafe.Sizeof(info)),
    ); err != nil {
        windows.CloseHandle(h)
        return nil, err
    }
    return &jobObject{handle: h}, nil
}

func (j *jobObject) Assign(p *os.Process) error {
    ph, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(p.Pid))
    if err != nil {
        return err
    }
    defer windows.CloseHandle(ph)
    return windows.AssignProcessToJobObject(j.handle, ph)
}
```

### Pattern 6: CLI Detection via exec.LookPath

**What:** At startup, scan PATH for each known CLI binary. Report which ones are found.

**When to use:** On application startup before any session can be created.

```go
// Source: pkg.go.dev/os/exec, go.dev/blog/path-security
var knownCLIs = []CLISpec{
    {Name: "claude",   DisplayName: "Claude Code"},
    {Name: "codex",    DisplayName: "OpenAI Codex"},
    {Name: "gemini",   DisplayName: "Gemini CLI"},
    {Name: "opencode", DisplayName: "OpenCode"},
}

type DetectedCLI struct {
    Name        string
    DisplayName string
    Path        string
}

func DetectCLIs() []DetectedCLI {
    var found []DetectedCLI
    for _, spec := range knownCLIs {
        path, err := exec.LookPath(spec.Name)
        if err == nil {
            found = append(found, DetectedCLI{
                Name:        spec.Name,
                DisplayName: spec.DisplayName,
                Path:        path,
            })
        }
    }
    return found
}
```

**Note on Windows:** On Windows, `exec.LookPath` automatically appends extensions from `PATHEXT` (`.com`, `.exe`, `.bat`, `.cmd`). Searching for `claude` finds `claude.exe` automatically.

### Pattern 7: Graceful Shutdown Handler

**What:** Use `signal.NotifyContext` to cancel all sessions on SIGINT/SIGTERM.

**When to use:** In `main()`, wrap the entire app lifecycle.

```go
// Source: pkg.go.dev/os/signal (NotifyContext added Go 1.16)
func main() {
    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    registry := pty.NewSessionRegistry()
    // ... start sessions ...

    <-ctx.Done()
    stop() // unregister signal handler

    // Graceful shutdown: kill all active sessions
    registry.KillAll()
}
```

### Pattern 8: win32-input-mode Parser (Windows)

**What:** A streaming state-machine parser that translates Windows Terminal's `win32-input-mode` escape sequences into the raw bytes the AI CLI expects. Runs between the ConPTY stdin pipe and the PTY write path.

**When to use:** On Windows only. Activated when reading from ConPTY input.

**Sequence format:** `ESC [ Vk ; Sc ; Uc ; Kd ; Cs ; Rc _`
- Uc (index 2): Unicode character value — emit this byte
- Kd (index 3): Key-down flag — emit only when `Kd == 1` (filter key-release events)

```go
// Source: dev.to/andylbrummer/taming-windows-terminals-win32-input-mode-in-go-conpty-applications-7gg
//         github.com/standardbeagle/agnt/blob/main/internal/overlay/input.go
// Build tag: windows

// ParseWin32Input reads from r and writes translated bytes to w.
// Incomplete sequences are buffered across Read calls.
func ParseWin32Input(r io.Reader, w io.Writer) error {
    var pending []byte
    buf := make([]byte, 4096)
    for {
        n, err := r.Read(buf)
        if err != nil {
            return err
        }
        data := append(pending, buf[:n]...)
        out, remainder := parseWin32Chunk(data)
        pending = remainder
        if len(out) > 0 {
            if _, werr := w.Write(out); werr != nil {
                return werr
            }
        }
    }
}

// parseWin32Chunk processes one buffer, returns (translated output, incomplete remainder).
func parseWin32Chunk(data []byte) (out []byte, remainder []byte) {
    i := 0
    for i < len(data) {
        if data[i] != 0x1b { // not ESC — pass through
            out = append(out, data[i])
            i++
            continue
        }
        // ESC found: need at least ESC [ to check for CSI
        if i+1 >= len(data) {
            remainder = data[i:]
            return
        }
        if data[i+1] != '[' { // not CSI — pass through ESC
            out = append(out, data[i])
            i++
            continue
        }
        // CSI sequence: find terminator
        end := findSequenceEnd(data, i+2)
        if end < 0 {
            // incomplete sequence — buffer it
            remainder = data[i:]
            return
        }
        // Parse the sequence
        if b, ok := parseSequence(data[i : end+1]); ok {
            out = append(out, b...)
        }
        i = end + 1
    }
    return
}
```

### Anti-Patterns to Avoid

- **Using `creack/pty` for new code:** Windows ConPTY support is not merged. Use `go-pty` exclusively.
- **`cmd.Process.Kill()` without process groups:** Kills only the top-level process. AI CLIs spawn subprocesses (node, python, etc.). Use `syscall.Kill(-pgid, SIGKILL)` on POSIX.
- **Running CLIs without a PTY:** Claude Code, Gemini CLI, and OpenCode check `isatty()`. A pipe produces degraded output or immediate exit.
- **Forgetting `TERM=xterm-256color`:** Gemini CLI PR #15828 confirmed that without this, tools get 8-color mode. Always set it explicitly — the Wails process environment may not inherit a correct `TERM`.
- **Reading PTY from multiple goroutines:** PTY reads are destructive. One `drainPTY` goroutine per session, fan-out to registered clients via channels.
- **Windows resize before process ready:** `ResizePseudoConsole()` near startup can be silently ignored. Send initial dimensions in the spawn call via `pty.Resize()` after `Start()`, but do not treat the error as fatal on Windows.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Windows ConPTY process spawning | Custom `CreatePseudoConsole` + `UpdateProcThreadAttribute` | `github.com/aymanbagabas/go-pty` | ConPTY API requires unsafe Go and complex process attribute chains; go-pty has battle-tested implementations |
| win32-input-mode parser | Custom byte scanner | Adapt from `github.com/standardbeagle/agnt/internal/overlay/input.go` | Sequence splitting across Read boundaries requires careful state management; reference implementation exists |
| Windows process-group kill | Custom process tree walker | Windows Job Object via `golang.org/x/sys/windows` | Process tree walking on Windows requires iterating all PIDs; Job Objects are the correct OS primitive |
| Path searching | Manual PATH split + file exist checks | `exec.LookPath` | Handles PATHEXT on Windows, symlinks, permission bits |

**Key insight:** Windows PTY and process management have enough edge cases that using existing libraries saves weeks of debugging. The three Windows-specific items above (ConPTY, win32-input-mode, Job Objects) are the most common sources of Windows terminal multiplexer bugs.

---

## Common Pitfalls

### Pitfall 1: Orphan Processes on Session Close

**What goes wrong:** AI CLIs spawn subprocesses (node for Claude Code, python for Codex). Killing only the top-level process via `cmd.Process.Kill()` leaves grandchild processes running with PPID=1.

**Why it happens:** `os/exec` does not automatically kill children of children. Documented real-world occurrences in OpenCode (issue #12913) and Gemini CLI (issue #20941).

**How to avoid:** Set `SysProcAttr{Setpgid: true}` before `cmd.Start()` on POSIX. After start, kill the group: `syscall.Kill(-pgid, syscall.SIGHUP)` then `syscall.Kill(-pgid, syscall.SIGKILL)`. On Windows, use Job Objects (see Pattern 5).

**Warning signs:** `ps aux | grep claude` shows processes still running after session close.

### Pitfall 2: win32-input-mode Corrupts Keyboard Input on Windows

**What goes wrong:** Windows Terminal enables win32-input-mode when a ConPTY application is launched. All keystrokes arrive as `ESC [ Vk ; Sc ; Uc ; Kd ; Cs ; Rc _` sequences instead of raw bytes. Without parsing, the AI CLI receives garbled input — Ctrl+C doesn't interrupt, arrow keys produce garbage.

**Why it happens:** Win32-input-mode is a Windows Terminal feature for richer keyboard support. It activates automatically and the disable sequence (`ESC[?9001l`) is unreliable.

**How to avoid:** Implement the state-machine parser from Pattern 8. Do not rely on the disable sequence. Test with Windows Terminal specifically.

**Warning signs:** On Windows, typing simple text shows garbage characters or nothing appears in the CLI.

### Pitfall 3: AI CLIs Detect Non-PTY Environments and Degrade

**What goes wrong:** Claude Code, Gemini CLI, and OpenCode use TUI frameworks (Bubble Tea / Ink) that call `isatty()`. Running them without a proper PTY produces non-interactive output or immediate exit.

**Why it happens:** TUI rendering requires terminal capabilities that pipes don't provide.

**How to avoid:** Always use go-pty to spawn these CLIs. Verify with a success criterion test: run the CLI, read a few bytes, check that `isatty()` would pass on the PTY slave fd.

**Warning signs:** Claude Code outputs plain text instead of its TUI, or exits with "not a TTY".

### Pitfall 4: Windows ConPTY Resize Race at Startup

**What goes wrong:** Calling `pty.Resize()` immediately after `cmd.Start()` on Windows is silently ignored. The ConPTY process needs a brief initialization window before it accepts resize calls.

**Why it happens:** Documented in microsoft/terminal issue #10400.

**How to avoid:** Do not treat the resize error as fatal. The initial dimensions should be passed at spawn time (go-pty `New()` accepts initial size on some paths). After startup, debounce resize events at 100-200ms before calling `Resize()`.

**Warning signs:** CLIs start with wrong dimensions even when initial resize was called.

### Pitfall 5: TERM Environment Variable Not Set

**What goes wrong:** The Wails desktop process environment may not have `TERM` set, or it may be set to a value the CLI doesn't expect. Without `TERM=xterm-256color`, AI CLIs fall back to 8-color or monochrome mode.

**Why it happens:** Wails wraps the app in a WebView process. The environment is not necessarily a shell environment. Gemini CLI PR #15828 demonstrates this exact bug.

**How to avoid:** Explicitly set `TERM=xterm-256color` and `COLORTERM=truecolor` in the `Env` field of every spawned command. Do not rely on inheriting the parent environment.

**Warning signs:** CLIs start but show no colors. `echo $TERM` inside the session shows `xterm` or nothing.

---

## Code Examples

Verified patterns from official sources:

### Opening a New PTY

```go
// Source: pkg.go.dev/github.com/aymanbagabas/go-pty
p, err := gopty.New()
if err != nil {
    return nil, fmt.Errorf("open pty: %w", err)
}
defer p.Close()

cmd := p.Command("claude")
cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
if err := cmd.Start(); err != nil {
    return nil, fmt.Errorf("start: %w", err)
}
```

### Resize

```go
// Source: pkg.go.dev/github.com/aymanbagabas/go-pty (Pty interface)
if err := p.Resize(cols, rows); err != nil {
    // non-fatal on Windows near startup
    log.Printf("resize warn: %v", err)
}
```

### Kill Process Group (POSIX)

```go
// Source: varunksaini.com/posts/kiling-processes-in-go/
// Build tag: !windows
pgid := cmd.Process.Pid
syscall.Kill(-pgid, syscall.SIGHUP)
time.Sleep(100 * time.Millisecond)
syscall.Kill(-pgid, syscall.SIGKILL)
```

### CLI Detection

```go
// Source: pkg.go.dev/os/exec
path, err := exec.LookPath("claude")
if err == nil {
    log.Printf("Claude Code found at %s", path)
}
```

### Graceful Shutdown

```go
// Source: pkg.go.dev/os/signal
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
<-ctx.Done()
// Kill all active sessions here
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `creack/pty` for all platforms | `go-pty` (aymanbagabas) for cross-platform | 2023-2024 | creack/pty Windows PR unmerged; go-pty is the current standard for Windows ConPTY in Go |
| Manual signal channel `signal.Notify` | `signal.NotifyContext` | Go 1.16 (2021) | Cleaner integration with context cancellation tree |
| Process.Kill() for child cleanup | Process groups (POSIX) + Job Objects (Windows) | Established practice | Kill only kills the leader; grandchild processes survive |
| Raw `xterm` TERM value | `xterm-256color` | Gemini CLI PR #15828 (2025) | Without explicit override, PTY-spawned processes may get 8-color mode |

**Deprecated/outdated:**

- `creack/pty`: Do not use for new code targeting Windows. Windows ConPTY support is not merged.
- `cmd.Process.Kill()` alone: Insufficient for processes that spawn children (all AI coding CLIs do this).

---

## Open Questions

1. **win32-input-mode parser completeness**
   - What we know: Reference implementation exists at `github.com/standardbeagle/agnt/internal/overlay/input.go`; sequence format is documented; key-up filtering and focus sequences are handled
   - What's unclear: Whether this implementation handles all edge cases for all four target CLIs (particularly function keys, modifier combos used by Bubble Tea UIs)
   - Recommendation: Phase 1 should include a Windows integration test that types input and verifies it appears correctly in the CLI output. Plan a spike to validate the parser against actual Claude Code input patterns.

2. **go-pty last release date**
   - What we know: v0.2.2 was released January 5, 2024; the library is maintained by the Charm team (active Go ecosystem contributors)
   - What's unclear: Whether v0.2.2 is compatible with the latest Windows Terminal and ConPTY API versions (Windows 11 24H2)
   - Recommendation: During Phase 1 implementation, test on Windows 11 current version. If issues arise, check go-pty issues and `charmbracelet/x/conpty` as a fallback for the Windows path.

3. **Initial PTY dimensions before xterm.js connects**
   - What we know: CLI dimensions must match terminal display or wrapping artifacts appear; resize after startup has a race on Windows
   - What's unclear: What sensible defaults to use when the frontend hasn't connected yet
   - Recommendation: Default to 80x24 (standard VT100) for initial spawn. Frontend sends actual dimensions immediately on WebSocket connect (Phase 2).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none — see Wave 0 |
| Quick run command | `go test ./internal/pty/... -v -timeout 30s` |
| Full suite command | `go test ./... -v -timeout 60s` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| CLI-01 | DetectCLIs() returns entries for installed CLIs | unit | `go test ./internal/pty/... -run TestDetectCLIs -v` | Wave 0 |
| CLI-02 | Spawning "claude" (or "echo" stub) via PTY passes isatty | integration | `go test ./internal/pty/... -run TestSpawnPTY -v` | Wave 0 |
| TERM-06 | Resize(80, 24) → Resize(120, 40) propagates without error | unit | `go test ./internal/pty/... -run TestResize -v` | Wave 0 |
| TERM-07 | Kill sends SIGHUP, process exits, no orphans remain | integration | `go test ./internal/pty/... -run TestKillClean -v` | Wave 0 |
| SESS-01 | Session struct persists in registry after simulated "window close" | unit | `go test ./internal/pty/... -run TestSessionPersist -v` | Wave 0 |

**Note on CLI-02 and TERM-07:** These require a real PTY-capable binary. Use `echo` or `cat` as a stub in CI (to avoid requiring Claude Code on CI runners). Gate the "real CLI" tests behind a build tag (`//go:build integration`) or environment variable check (`AGENTHUB_INTEGRATION_TEST=1`).

### Sampling Rate

- **Per task commit:** `go test ./internal/pty/... -v -timeout 30s`
- **Per wave merge:** `go test ./... -v -timeout 60s`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/pty/backend.go` — SessionBackend interface definition
- [ ] `internal/pty/native.go` — NativePTYBackend implementation
- [ ] `internal/pty/session.go` — Session struct
- [ ] `internal/pty/registry.go` — SessionRegistry with RWMutex
- [ ] `internal/pty/detect.go` — CLI detection via exec.LookPath
- [ ] `internal/pty/cleanup.go` — graceful shutdown + process group kill
- [ ] `internal/pty/win32input.go` — win32-input-mode parser (build tag: windows)
- [ ] `internal/pty/win32input_other.go` — no-op stub (build tag: !windows)
- [ ] `internal/pty/detect_test.go` — TestDetectCLIs
- [ ] `internal/pty/native_test.go` — TestSpawnPTY, TestResize, TestKillClean, TestSessionPersist
- [ ] `go.mod` — module definition (`go mod init`)
- [ ] `cmd/agenthub/main.go` — minimal main for manual smoke test
- [ ] Framework install: none needed — Go stdlib `testing` package

---

## Sources

### Primary (HIGH confidence)

- `pkg.go.dev/github.com/aymanbagabas/go-pty` — full API reference: Pty interface, Cmd type, Resize signature, Read/Write, SysProcAttr
- `go.dev/blog/path-security` — exec.LookPath PATH security model and Go 1.19+ behavior
- `pkg.go.dev/os/signal` — signal.NotifyContext pattern for graceful shutdown
- `pkg.go.dev/os/exec` — Command, LookPath, exec.Cmd lifecycle
- `github.com/google-gemini/gemini-cli/pull/15828` — TERM=xterm-256color requirement confirmed empirically
- `github.com/anthropics/claude-code` — binary name `claude` confirmed
- `github.com/google-gemini/gemini-cli` — binary name `gemini` confirmed
- `github.com/openai/codex` — binary name `codex` confirmed
- `opencode.ai/docs` — binary name `opencode` confirmed

### Secondary (MEDIUM confidence)

- `dev.to/andylbrummer/taming-windows-terminals-win32-input-mode-in-go-conpty-applications-7gg` — win32-input-mode sequence format and Go 1.23 iterator parser (Dec 2025 article, verified against agnt repo)
- `github.com/standardbeagle/agnt/internal/overlay/input.go` — reference implementation for win32-input-mode parser
- `gist.github.com/hallazzang/76f3970bfc949831808bbebc8ca15209` — Windows Job Object pattern for child process cleanup
- `varunksaini.com/posts/kiling-processes-in-go/` — process group kill pattern verified with multiple sources
- `github.com/microsoft/terminal/issues/10400` — ConPTY resize race near client attach documented
- `github.com/anomalyco/opencode/issues/12913` — orphan process behavior in OpenCode confirmed
- `github.com/google-gemini/gemini-cli/issues/20941` — nested process tree not killed on PTY abort confirmed

### Tertiary (LOW confidence)

- None for Phase 1 — all critical claims are verified from primary or secondary sources.

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — go-pty API verified from pkg.go.dev; stdlib packages are authoritative
- Architecture: HIGH — patterns derived from official docs + reference implementations; no speculation
- Pitfalls: HIGH — each pitfall traced to an official GitHub issue or verified PR
- CLI binary names: HIGH — confirmed from each CLI's official repository/docs

**Research date:** 2026-03-17
**Valid until:** 2026-09-17 (6 months — go-pty is stable; core Go APIs are not fast-moving)
