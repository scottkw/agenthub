# Phase 22: CLI Attach - Research

**Researched:** 2026-03-24
**Domain:** Terminal raw I/O, WebSocket PTY proxy, signal handling, terminal state management
**Confidence:** HIGH

## Summary

Phase 22 implements `agenthub attach <id>` — a command that connects the operator's local terminal to an existing daemon-managed PTY session via the already-built relay WebSocket server. The relay infrastructure is complete (Hub, HubManager, scrollback, binary framing protocol, WebSocket server). The attach command is the missing client side of that relay.

The primary technical challenge is not networking but **terminal lifecycle management**: putting the local terminal into raw mode, forwarding all bytes verbatim (including Ctrl-C as 0x03, not SIGINT), resizing the remote PTY when the local window changes (SIGWINCH), and reliably restoring terminal state on all exit paths — normal detach, SIGTERM, SIGHUP, panic. The scrollback replay requirement is already solved by the server (subscribe-before-snapshot ordering is in place).

There are no new dependencies beyond `golang.org/x/term` (already in go.sum as a transitive dep at v0.38.0) and the existing `github.com/coder/websocket` v1.8.14.

**Primary recommendation:** Implement `cmdAttach` in `cmd/agenthub-cli/` as a `websocket.Dial` loop with `golang.org/x/term.MakeRaw`, SIGWINCH-driven resize, and `os/signal.NotifyContext` for graceful shutdown; restore terminal state with a `defer` and dedicated signal handlers covering every exit path.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLI-05 | User can attach to a session with full interactive PTY proxy (`agenthub attach`) | WebSocket dial to relay server; relay server at `ws://127.0.0.1:<port>/sessions/<id>/ws` already handles multiplexing |
| CLI-06 | Attached session supports raw I/O, terminal resize propagation, and ctrl-c passthrough | `golang.org/x/term.MakeRaw` + `signal.Notify(SIGWINCH)` + raw stdin copy; ctrl-c is a byte (0x03) not a signal in raw mode |
| CLI-07 | User can detach from an attached session via configurable prefix key | Prefix-key interception in the stdin read pump (byte-level state machine); default `~.` or `Ctrl-\` |
| CLI-08 | Attaching to an existing session replays recent scrollback output | Server already sends scrollback snapshot before live frames; client just needs to receive and write to stdout |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/term` | v0.38.0 (in go.sum) | MakeRaw, Restore, GetSize, IsTerminal | The standard Go stdlib extension for terminal control; single cross-platform API |
| `github.com/coder/websocket` | v1.8.14 (already in go.mod) | WebSocket client dial to relay server | Already used by the relay server; consistent library |
| `os/signal` (stdlib) | Go stdlib | SIGWINCH, SIGTERM, SIGHUP, SIGINT interception | Standard signal delivery mechanism |
| `golang.org/x/sys/unix` | in go.mod as direct dep | SIGWINCH constant, IoctlGetWinsize | Already used in the pty package |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/relay` | project | Frame encoding/decoding (MakeInputFrame, MakeResizeFrame, ParseFrame, MsgOutput) | Reuse existing frame protocol — never build custom framing |
| `internal/daemon` | project | DaemonClient.GetRelayPort(), GetSession to validate session exists | Get relay port from daemon before dialing WebSocket |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `golang.org/x/term.MakeRaw` | `syscall.Syscall(SYS_IOCTL, ...)` manually | Manual ioctl is error-prone and non-portable; `x/term` wraps this correctly |
| `os/signal` for SIGWINCH | Polling terminal size | Polling adds latency and CPU waste; SIGWINCH is the correct mechanism |

**Installation:**
```bash
go get golang.org/x/term
```

(This moves it from indirect to direct in go.mod. The version v0.38.0 is already pinned in go.sum.)

## Architecture Patterns

### Recommended Project Structure

New file added to existing CLI package:
```
cmd/agenthub-cli/
├── main.go           # Add "attach" case to switch
├── main_test.go      # Existing tests
├── cmd_attach.go     # New: cmdAttach function + attachSession
└── cmd_attach_test.go # New: unit tests for attach logic
```

No new packages. `cmdAttach` follows the same `func cmdAttach(client *daemon.DaemonClient, args []string) error` signature pattern established in Phase 21.

### Pattern 1: Subscribe-Before-Snapshot (Already Implemented Server-Side)

**What:** The server subscribes the client to the Hub before sending the scrollback snapshot, ensuring no frames are lost between snapshot time and first live message.
**When to use:** Already done in `relay/server.go:handleSession`. The CLI client does not need to implement anything special — it just reads all incoming frames and writes payloads to stdout.
**Example:** See `relay/server.go` lines 73-83. Client receives a single binary message containing the full scrollback buffer (already MsgOutput-framed bytes), then receives live MsgOutput frames after.

### Pattern 2: Raw Terminal with Deferred Restore

**What:** Put terminal in raw mode immediately after validation, register restore on all exit paths before any goroutine is launched.
**When to use:** Any time `MakeRaw` is called.
**Example:**
```go
// Source: golang.org/x/term official docs
oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
if err != nil {
    return fmt.Errorf("attach: make raw: %w", err)
}
defer term.Restore(int(os.Stdin.Fd()), oldState)
```

The `defer` fires on normal return, panic, and runtime.Goexit. Signal handlers (SIGTERM, SIGHUP) must also call `term.Restore` explicitly because signals bypass `defer` when they terminate the process.

### Pattern 3: Signal Handler for Terminal Restore on SIGTERM/SIGHUP

**What:** Register SIGTERM and SIGHUP handlers that restore terminal state before exiting.
**Why:** `defer` does NOT run when a signal terminates the process via default signal disposition. With `signal.NotifyContext`, Go converts signals into context cancellation — `defer` does run when `<-ctx.Done()` unblocks. This is the correct pattern.
**Example:**
```go
// Source: os/signal package docs
ctx, stop := signal.NotifyContext(context.Background(),
    os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
defer stop()
// defer term.Restore runs because ctx cancellation causes clean return
```

### Pattern 4: SIGWINCH-Driven Resize

**What:** Listen for SIGWINCH signals, read new terminal size with `term.GetSize`, send a MsgResize frame to the relay server.
**When to use:** Whenever the user resizes their terminal window while attached.
**Example:**
```go
// Source: golang.org/x/sys/unix docs + golang.org/x/term docs
winchCh := make(chan os.Signal, 1)
signal.Notify(winchCh, syscall.SIGWINCH)
go func() {
    for range winchCh {
        cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
        if err != nil {
            continue
        }
        frame := relay.MakeResizeFrame(uint16(cols), uint16(rows))
        _ = conn.Write(ctx, websocket.MessageBinary, frame)
    }
}()
```

### Pattern 5: Detach Key Prefix State Machine

**What:** A single-pass byte scanner in the stdin read pump that detects the configured detach prefix sequence without buffering.
**Default detach key:** `Ctrl-\` (0x1C, one byte) is the tmux-style default — simple, single byte, unlikely to appear in normal AI CLI interaction. Alternative: `~.` (tilde-dot, the SSH detach sequence, two bytes). The config can be a single string field.
**Implementation approach:** Maintain a small state machine: if the next byte matches prefix[state], advance state; if state == len(prefix), detach; if the byte does NOT match, flush any buffered prefix bytes to the PTY and reset state.
**Example:**
```go
// Single-byte prefix example (Ctrl-\, 0x1C)
const detachKey = byte(0x1C)
for {
    n, err := os.Stdin.Read(buf)
    for _, b := range buf[:n] {
        if b == detachKey {
            return nil // clean detach
        }
        // write b as MsgInput frame
    }
}
```

For a two-byte prefix like `~.`, a small 2-state machine:
```go
// State: 0 = idle, 1 = saw ~
var prefixState int
for _, b := range buf[:n] {
    switch prefixState {
    case 0:
        if b == '~' { prefixState = 1; continue }
        sendByte(b)
    case 1:
        if b == '.' { return nil } // detach
        sendByte('~') // flush buffered ~
        sendByte(b)
        prefixState = 0
    }
}
```

### Pattern 6: Two-Goroutine Pump with errgroup-style Coordination

**What:** One goroutine reads from stdin and sends MsgInput frames; another reads from the WebSocket and writes MsgOutput payloads to stdout. Either goroutine can trigger detach/disconnect.
**When to use:** Standard bidirectional proxy pattern.
**Example:**
```go
readDone := make(chan error, 1)
writeDone := make(chan error, 1)
go func() { readDone <- stdinPump(ctx, conn) }()
go func() { writeDone <- wsOutputPump(ctx, conn, os.Stdout) }()
select {
case <-readDone:   // stdin closed or detach key
case <-writeDone:  // WS closed (session ended)
case <-ctx.Done(): // signal
}
```

### Anti-Patterns to Avoid

- **os.Exit in cmdAttach:** `os.Exit` bypasses `defer term.Restore`. Always return an error to `main()` and let `main()` call `os.Exit`.
- **Ctrl-C as SIGINT:** In raw mode, Ctrl-C is byte 0x03, not a signal. Do NOT call `signal.Notify` for `os.Interrupt` expecting to intercept Ctrl-C from the user; that would intercept it as a signal to the attach process itself. Let it pass through as a raw byte.
- **Goroutine context from HTTP request:** Not applicable here (no HTTP involved), but note the existing daemon pattern: PTY goroutines use `context.Background()`, not request context.
- **Reading terminal size before MakeRaw:** Read initial terminal size and send an initial resize frame immediately after connecting, before any I/O pump starts — ensures the remote PTY dimensions match the local terminal on first attach.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Raw terminal mode | Custom ioctl/termios calls | `golang.org/x/term.MakeRaw` + `Restore` | Platform differences (macOS/Linux/Windows); `x/term` handles all correctly |
| Terminal size query | Custom TIOCGWINSZ ioctl | `golang.org/x/term.GetSize` | Wraps platform differences; single call |
| Scrollback replay | Custom buffer/state in client | Server already sends snapshot on connect | Already implemented in `relay/server.go:handleSession` |
| WebSocket framing | Custom TCP framing | `github.com/coder/websocket` | Already in the project; server uses it |
| Frame encoding | Custom binary format | `relay.MakeInputFrame`, `relay.MakeResizeFrame`, `relay.ParseFrame` | Protocol already defined in `internal/relay` |

**Key insight:** The relay infrastructure (Hub, scrollback, fan-out, framing) is fully built and tested. The attach command is a thin WebSocket client with terminal lifecycle wrappers.

## Common Pitfalls

### Pitfall 1: Terminal Left in Raw Mode on Crash
**What goes wrong:** If the process exits without restoring terminal mode, the user's shell becomes unusable — no echo, no line discipline, no SIGINT from Ctrl-C.
**Why it happens:** `defer` does not fire when `os.Exit` is called directly or when a goroutine panics without recovery.
**How to avoid:** (1) Never call `os.Exit` inside `cmdAttach` — return errors to `main()`. (2) Use `signal.NotifyContext` so SIGTERM/SIGHUP become context cancellation (then `defer` does fire). (3) Add a recover in cmdAttach or at least test signal paths.
**Warning signs:** After a test run or crash, the terminal stops echoing keystrokes.

### Pitfall 2: MsgResize2 vs MsgResize Frame Type Mismatch
**What goes wrong:** The relay server's read pump handles `MsgResize2` (0x11) for resize frames sent by clients, but `MakeResizeFrame` uses `MsgResize` (0x02). Looking at `relay/server.go:101-106`, the input handler switches on `MsgResize2`, not `MsgResize`.
**Why it happens:** `MsgResize` (0x02) is the output direction (server→client in some designs); `MsgResize2` (0x11) is the client→server resize command.
**How to avoid:** The CLI attach command must send resize frames with `MsgResize2` (0x11), NOT `MakeResizeFrame()` which uses `MsgResize` (0x02). Either create a `MakeResize2Frame` helper in `internal/relay` or manually prepend `MsgResize2`.
**Evidence:** `relay/server.go` line 101: `case MsgResize2:` handles client-to-server resize.

### Pitfall 3: Scrollback Contains Framed Bytes, Not Raw Terminal Output
**What goes wrong:** The scrollback buffer stores `MakeOutputFrame(data)` bytes (with 0x01 prefix), not raw PTY bytes. When the client receives the scrollback snapshot it must strip the `MsgOutput` prefix byte before writing to stdout.
**Why it happens:** The Hub appends framed data: `frame := MakeOutputFrame(buf[:n]); h.scrollback.Append(frame)` (hub.go line 88-89). The snapshot sent as a single binary WebSocket message contains the raw scrollback bytes.
**How to avoid:** The snapshot message received by the client is a raw scrollback dump. Parse it as a sequence of frames: first byte is type, rest is payload. The `ParseFrame` function handles single-frame parsing; for the full snapshot the client must iterate, stripping each `MsgOutput` prefix.
**Note:** Looking at `server_test.go` line 149: `fullSnapshot := string(snapshotPayload)` — `snapshotPayload` is what comes after `ParseFrame`, meaning the outer `ParseFrame` strips one level of framing. The snapshot itself is the concatenated MakeOutputFrame bytes of all prior PTY output, so after the outer `ParseFrame` call strips the first `MsgOutput` byte, the remainder contains additional framed chunks. The safe approach: apply `ParseFrame` iteratively until the buffer is exhausted, writing each `MsgOutput` payload to stdout.

**Alternative (simpler):** Since the scrollback is raw PTY output wrapped in uniform MsgOutput frames, a simpler client approach is to receive ANY message from the WebSocket and, if the first byte is `MsgOutput` (0x01), write `msg[1:]` to stdout. This works for both scrollback (which is sent as one big binary message) and live frames (sent individually).

### Pitfall 4: Stdin is Not a Terminal in Test Environment
**What goes wrong:** `term.MakeRaw` returns an error if stdin is not a terminal (e.g., in CI, pipes, test harness).
**Why it happens:** `IsTerminal(int(os.Stdin.Fd()))` returns false in non-terminal environments.
**How to avoid:** Check `term.IsTerminal` before calling `MakeRaw`. In tests, pass an `io.Reader` and `io.Writer` instead of `os.Stdin`/`os.Stdout` — keep raw-mode setup in `main()`, not inside the testable function.

### Pitfall 5: SIGWINCH Not Available on Windows
**What goes wrong:** `syscall.SIGWINCH` does not exist on Windows; build fails.
**Why it happens:** SIGWINCH is a POSIX signal.
**How to avoid:** Use a build tag or platform-specific file for the SIGWINCH handler. Since this project already uses `_unix.go` / `_windows.go` suffix patterns (see `daemon/process_unix.go`, `daemon/process_windows.go`), follow the same convention: `cmd_attach_unix.go` for SIGWINCH, `cmd_attach_windows.go` stub that is a no-op.

### Pitfall 6: Missing Initial Resize Frame
**What goes wrong:** After attaching, the remote PTY dimensions don't match the local terminal because a resize frame was never sent. AI CLI output wraps at the wrong column width.
**Why it happens:** The remote PTY was created with whatever dimensions were specified at `CreateSession` time; the attach client never tells the server the current local terminal size.
**How to avoid:** After connecting and before starting the I/O pump, send one resize frame with `term.GetSize(int(os.Stdin.Fd()))`.

## Code Examples

### WebSocket Client Dial (using existing coder/websocket)
```go
// Source: github.com/coder/websocket dial pattern from relay/server_test.go
import "github.com/coder/websocket"

wsURL := fmt.Sprintf("ws://127.0.0.1:%d/sessions/%s/ws", relayPort, sessionID)
conn, _, err := websocket.Dial(ctx, wsURL, nil)
if err != nil {
    return fmt.Errorf("attach: dial relay: %w", err)
}
defer conn.CloseNow()
```

### Raw Terminal Mode Setup
```go
// Source: golang.org/x/term official documentation
import "golang.org/x/term"

if !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("attach: stdin is not a terminal")
}
oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
if err != nil {
    return fmt.Errorf("attach: make raw: %w", err)
}
defer term.Restore(int(os.Stdin.Fd()), oldState)
```

### Signal-Safe Shutdown with Terminal Restore
```go
// Source: os/signal.NotifyContext documentation
ctx, stop := signal.NotifyContext(context.Background(),
    os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
defer stop()
// defer term.Restore fires when ctx.Done() causes clean return
```

### Sending Correct Resize Frame (MsgResize2 = client-to-server)
```go
// Source: relay/server.go line 101 — server handles MsgResize2 from clients
// relay/protocol.go — MsgResize2 = 0x11
func makeClientResizeFrame(cols, rows uint16) []byte {
    return []byte{
        relay.MsgResize2,
        byte(cols >> 8), byte(cols),
        byte(rows >> 8), byte(rows),
    }
}
```

### Output Pump — Receiving and Printing Frames
```go
// Each WebSocket binary message is either:
// - Scrollback snapshot: raw scrollback bytes (sequence of framed chunks)
// - Live frame: single MsgOutput frame
// In both cases: parse type byte, if MsgOutput write payload to stdout.
for {
    _, msg, err := conn.Read(ctx)
    if err != nil {
        return // session ended or detached
    }
    // Iterate: the snapshot may contain multiple concatenated frames
    for len(msg) > 0 {
        msgType, payload, parseErr := relay.ParseFrame(msg)
        if parseErr != nil {
            break
        }
        if msgType == relay.MsgOutput {
            _, _ = os.Stdout.Write(payload)
        }
        // Advance past this frame: 1 (type) + len(payload)
        // ParseFrame returns payload = msg[1:], so advance by 1+len(payload)
        msg = msg[1+len(payload):]
    }
}
```

**Note on scrollback iteration:** `ParseFrame` returns `payload = frame[1:]` — i.e., everything after the type byte. For scrollback, the buffer is a flat byte sequence of concatenated MsgOutput frames: `[0x01, data..., 0x01, data..., ...]`. After stripping the outer WebSocket message boundary, iterate: `msg[0]` is type, `msg[1:]` is the rest. But since each frame's payload is variable-length (unbounded), and there's no length prefix, the only termination condition is exhausting the buffer. This works because the scrollback buffer contains complete MakeOutputFrame blobs appended sequentially; the outer binary WebSocket message contains the entire scrollback snapshot as one blob.

### SIGWINCH Handler (Unix only — in cmd_attach_unix.go)
```go
//go:build !windows

func watchResize(ctx context.Context, conn *websocket.Conn) {
    winchCh := make(chan os.Signal, 1)
    signal.Notify(winchCh, syscall.SIGWINCH)
    defer signal.Stop(winchCh)
    for {
        select {
        case <-winchCh:
            cols, rows, err := term.GetSize(int(os.Stdin.Fd()))
            if err != nil {
                continue
            }
            frame := makeClientResizeFrame(uint16(cols), uint16(rows))
            _ = conn.Write(ctx, websocket.MessageBinary, frame)
        case <-ctx.Done():
            return
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `syscall.RawConn` + manual termios | `golang.org/x/term.MakeRaw` | x/term added ~2019 | Single-function raw mode; handles all POSIX platforms + Windows |
| Manual signal channel for each signal | `signal.NotifyContext` | Go 1.16 | Clean context-driven shutdown; defer works normally |
| Custom WebSocket library per project | `github.com/coder/websocket` (nhooyr/websocket) | Forked ~2023 | Already in this project; context-aware; no goroutine leak on cancel |

**Deprecated/outdated:**
- `gorilla/websocket`: Present in go.sum as transitive dep but NOT used by this project's code. Do not use for new code — already using `coder/websocket`.

## Open Questions

1. **Detach key configuration source**
   - What we know: Requirements say "configurable prefix key" (CLI-07); no config file system exists yet in this project
   - What's unclear: Is the detach key hardcoded for Phase 22 with a future config hook, or does Phase 22 need a `--detach-key` flag?
   - Recommendation: Hardcode a sensible default (`Ctrl-\` = 0x1C) with a `--detach-key` flag for Phase 22. No persistent config file needed. This is the tmux default and well-understood.

2. **Windows support for `attach`**
   - What we know: The project builds for Windows (see `process_windows.go`, `win32input.go`); SIGWINCH doesn't exist on Windows
   - What's unclear: Should `attach` work on Windows in Phase 22, or is it Unix-only for now?
   - Recommendation: Build it Unix-first (`!windows` build tag) with a stub `cmd_attach_windows.go` that returns `fmt.Errorf("attach: not supported on Windows")`. Document as future work.

3. **Scrollback frame iteration termination**
   - What we know: The scrollback buffer is a flat byte sequence of concatenated MakeOutputFrame blobs
   - What's unclear: Each frame's payload is variable-length with no explicit length field — the only delimiter is exhausting the buffer
   - Recommendation: Iterate until `len(msg) == 0`. Each `ParseFrame` call consumes `msg[0]` (type) and `msg[1:]` (payload). Since there's no length prefix, treat the entire remainder as one payload per frame, which means the scrollback snapshot must be sent as a single WebSocket binary message (it is — see `server.go:79-83`). The outer message boundary is the frame length. Within the snapshot, consecutive MakeOutputFrame blobs are delimited only by the 0x01 type prefix. **This creates an ambiguity** if the payload itself contains 0x01 bytes. Resolution: The scrollback snapshot is sent as a single WebSocket message; receive it as one, strip the leading MsgOutput byte, write the rest directly to stdout (don't try to parse individual sub-frames). This matches what `server_test.go` does at line 149 with `fullSnapshot := string(snapshotPayload)`.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./cmd/agenthub-cli/ -run TestCmdAttach -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLI-05 | `cmdAttach` returns error for missing args | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_MissingArgs -v` | ❌ Wave 0 |
| CLI-05 | `cmdAttach` dials relay and receives output | integration | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ReceivesOutput -v` | ❌ Wave 0 |
| CLI-06 | Ctrl-C byte (0x03) is forwarded, not swallowed | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_CtrlCPassthrough -v` | ❌ Wave 0 |
| CLI-06 | Resize frame sent on terminal resize | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ResizeForwarded -v` | ❌ Wave 0 |
| CLI-07 | Detach key causes clean return from cmdAttach | unit | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_DetachKey -v` | ❌ Wave 0 |
| CLI-08 | Scrollback bytes are written to output before live frames | integration | `go test ./cmd/agenthub-cli/ -run TestCmdAttach_ScrollbackReplay -v` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./cmd/agenthub-cli/ -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `cmd/agenthub-cli/cmd_attach.go` — cmdAttach implementation
- [ ] `cmd/agenthub-cli/cmd_attach_unix.go` — SIGWINCH handler (build tag `!windows`)
- [ ] `cmd/agenthub-cli/cmd_attach_windows.go` — stub returning "not supported" (build tag `windows`)
- [ ] `cmd/agenthub-cli/cmd_attach_test.go` — unit/integration tests for attach
- [ ] `go get golang.org/x/term` — promote from indirect to direct dep in go.mod

## Sources

### Primary (HIGH confidence)

- `internal/relay/protocol.go` — Frame types: MsgOutput=0x01, MsgInput=0x10, MsgResize=0x02, MsgResize2=0x11, MsgPing=0x12
- `internal/relay/server.go` — Server input handler: uses `MsgResize2` (0x11) for client-to-server resize; uses `MsgInput` (0x10) for keyboard input; scrollback snapshot sent before live frames
- `internal/relay/hub.go` — scrollback stores framed output (MakeOutputFrame applied before Append)
- `internal/relay/server_test.go` — WebSocket dial pattern using `websocket.Dial`; snapshot handling
- `internal/daemon/api.go` — `GET /relay-port` returns TCP port; relay server on random port at `127.0.0.1`
- `internal/daemon/client.go` — `GetRelayPort()` method available on DaemonClient
- `golang.org/x/term` v0.38.0 — `MakeRaw`, `Restore`, `GetSize`, `IsTerminal` — confirmed via `go doc`
- `golang.org/x/sys/unix` — `SIGWINCH` constant confirmed via `go doc`
- `go.mod` + `go.sum` — Verified `coder/websocket v1.8.14` is direct dep; `golang.org/x/term v0.38.0` is in go.sum (indirect)

### Secondary (MEDIUM confidence)

- `os/signal.NotifyContext` — Go 1.16+ stdlib; confirmed available (project uses `go 1.26.1`)
- Build tag pattern `!windows` / `windows` — confirmed used in this project (`process_unix.go`, `process_windows.go`)

### Tertiary (LOW confidence)

- None — all claims verified against codebase or official Go stdlib

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified against go.mod, go.sum, and `go doc`
- Architecture: HIGH — derived directly from reading existing relay/server/hub code
- Pitfalls: HIGH — derived from direct code reading (MsgResize vs MsgResize2 discrepancy verified in relay/server.go line 101)

**Research date:** 2026-03-24
**Valid until:** 2026-04-24 (stable Go stdlib + project-internal code; no fast-moving deps)
