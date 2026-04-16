# Phase 75: CLI Status Bar - Pattern Map

**Mapped:** 2026-04-15
**Files analyzed:** 5 new/modified files
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/statusbar/bar.go` | utility (renderer) | event-driven (ticker + push) | `internal/status/detector.go` | role-match |
| `internal/statusbar/bar_test.go` | test | — | `internal/status/detector_test.go` | exact |
| `internal/relay/protocol.go` | utility (protocol) | request-response | `internal/relay/protocol.go` (self) | exact — additive |
| `internal/relay/server.go` | service | pub-sub | `internal/relay/server.go` (self) | exact — additive |
| `cmd_attach.go` | utility (CLI command) | request-response | `cmd_attach.go` (self) | exact — additive |

## Pattern Assignments

---

### `internal/statusbar/bar.go` (utility, event-driven)

**Analog:** `internal/status/detector.go`

**Package declaration and imports pattern** (`internal/status/detector.go` lines 1-11):
```go
package status

import (
    "regexp"
    "sync"

    "github.com/scottkw/agenthub/internal/relay"
)
```

For `bar.go`, the imports will be:
```go
package statusbar

import (
    "context"
    "fmt"
    "strings"
    "sync"
    "time"

    "golang.org/x/term"
)
```

**Type definition with shared-state mutex pattern** (`internal/status/detector.go` lines 94-113):
```go
// Detector maintains a rolling tail of stripped PTY output for a single session
// and classifies its current state.
type Detector struct {
    sessionID string
    patterns  PatternSet
    tail      []byte
    current   SessionStatus
    onTransit func(string, SessionStatus)
}

// NewDetector creates a Detector for the given session.
// onTransit is called (from the same goroutine as Feed) whenever the classified
// status changes.  It is also called on the very first Feed to emit the initial status.
func NewDetector(sessionID string, patterns PatternSet, onTransit func(string, SessionStatus)) *Detector {
    return &Detector{
        sessionID: sessionID,
        patterns:  patterns,
        current:   "",
        onTransit: onTransit,
    }
}
```

Copy this constructor pattern for `Bar`: struct with all config fields + mutable state fields protected by `sync.Mutex`, and a `New()` constructor that returns a pointer.

**sync.Once lifecycle guard pattern** (`internal/relay/hub.go` lines 169-177):
```go
// Shutdown signals that the hub has stopped. Safe to call multiple times.
func (h *Hub) Shutdown() {
    h.closeOnce.Do(func() {
        h.mu.Lock()
        h.closed = true
        h.mu.Unlock()
        close(h.done)
    })
}
```

Copy `sync.Once` guard for `Bar.Stop()` — it must be safe to call multiple times (defer + panic paths).

**Mutex-protected shared state read/write pattern** (`internal/relay/hub.go` lines 100-126):
```go
func (h *Hub) ResizeClient(sub *Subscriber, cols, rows int) error {
    h.mu.Lock()
    sub.Cols = cols
    sub.Rows = rows
    // ... compute max ...
    needResize := ...
    h.mu.Unlock() // release BEFORE calling resizeFn

    if needResize && h.resizeFn != nil {
        return h.resizeFn(maxCols, maxRows)
    }
    return nil
}
```

Copy this lock/unlock-before-side-effect pattern for `Bar.SetViewerCount()` and `Bar.draw()`: lock, copy shared fields into locals, unlock, then use locals for I/O.

**Ticker goroutine lifecycle with context pattern** (`cmd_attach_unix.go` lines 18-37):
```go
func watchResize(ctx context.Context, conn *websocket.Conn) {
    winchCh := make(chan os.Signal, 1)
    signal.Notify(winchCh, syscall.SIGWINCH)
    go func() {
        defer signal.Stop(winchCh)
        for {
            select {
            case <-winchCh:
                // ... do work ...
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

For `bar.tickLoop()`, use the same `select` pattern with `<-ticker.C` instead of `<-winchCh` and `<-ctx.Done()` for shutdown. Use `context.WithCancel` created in `Bar.Start()` and cancelled in `Bar.Stop()`.

**No-fmt.Println rule (from ANSI drawing):** Use `fmt.Fprint(b.w, text)` only — never `fmt.Fprintf(..., "...\n")` inside draw. Trailing newlines scroll the reserved row.

---

### `internal/statusbar/bar_test.go` (test)

**Analog:** `internal/status/detector_test.go`

**External test package pattern** (`internal/status/detector_test.go` lines 1-9):
```go
package status_test

import (
    "sync"
    "testing"
    "time"

    "github.com/scottkw/agenthub/internal/relay"
    "github.com/scottkw/agenthub/internal/status"
)
```

Use `package statusbar_test` (external test package). Import `internal/statusbar` by path.

**Mock/stub for io.Writer capture pattern** (`cmd_attach_test.go` lines 168-184):
```go
// safeBuf is a bytes.Buffer protected by a mutex for concurrent test access.
type safeBuf struct {
    mu  sync.Mutex
    buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.buf.Write(p)
}

func (s *safeBuf) String() string {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.buf.String()
}
```

Use this `safeBuf` pattern to capture all `io.Writer` output from `bar.go` in tests. The draw goroutine writes concurrently, so the buffer must be mutex-protected.

**Test assertion for ANSI escape presence** (pattern from `cmd_attach_test.go` lines 280-305):
```go
func TestPrintAttachBanner(t *testing.T) {
    var buf bytes.Buffer
    printAttachBanner(&buf, "my-session", "claude", "macbook-pro.local")
    output := buf.String()

    if !strings.Contains(output, "my-session") {
        t.Errorf("banner missing session name; got: %s", output)
    }
    if !strings.Contains(output, `Ctrl-\`) {
        t.Errorf("banner missing detach key hint; got: %s", output)
    }
}
```

Use `strings.Contains(output, ...)` assertions for ANSI content tests. For verifying ANSI escape sequences themselves, assert against literal escape characters: `strings.Contains(output, "\033[7m")` for reverse video, `strings.Contains(output, "\033[r")` for scroll region reset.

**Goroutine timeout pattern** (`internal/relay/hub_test.go` lines 36-50):
```go
select {
case frame := <-sub.Msgs:
    // assert on frame
case <-time.After(2 * time.Second):
    t.Fatal("timeout waiting for frame")
}
```

Use `time.After` selects for any test that waits for a goroutine event (ticker firing, Stop completing).

---

### `internal/relay/protocol.go` (additive modification)

**Analog:** `internal/relay/protocol.go` (self — additive)

**Existing constant block pattern** (`internal/relay/protocol.go` lines 7-15):
```go
// Message type bytes — single-byte prefix for every framed message.
const (
    MsgOutput  byte = 0x01 // PTY stdout/stderr → client
    MsgResize  byte = 0x02 // Terminal resize (cols, rows as big-endian uint16)
    MsgTitle   byte = 0x03 // Window title update
    MsgInput   byte = 0x10 // Client keyboard input → PTY stdin
    MsgResize2 byte = 0x11 // Alternative resize format (reserved)
    MsgPing    byte = 0x12 // Keep-alive ping
)
```

Append `MsgMeta byte = 0x20` to this const block. Add comment noting it is the stable metadata push channel and listing reserved range for future use.

**Existing frame maker pattern** (`internal/relay/protocol.go` lines 18-31):
```go
// MakeOutputFrame prepends the MsgOutput type byte to data.
func MakeOutputFrame(data []byte) []byte {
    frame := make([]byte, 1+len(data))
    frame[0] = MsgOutput
    copy(frame[1:], data)
    return frame
}

// MakeInputFrame prepends the MsgInput type byte to data.
func MakeInputFrame(data []byte) []byte {
    frame := make([]byte, 1+len(data))
    frame[0] = MsgInput
    copy(frame[1:], data)
    return frame
}
```

Copy this exact `make([]byte, 1+len(data)) / frame[0] = MsgMeta / copy(frame[1:], b)` pattern for `MakeMeta`. Add a `MetaPayload` struct and `MakeMeta` function following the same style. The `encoding/json` import must be added to protocol.go.

---

### `internal/relay/server.go` (additive modification)

**Analog:** `internal/relay/server.go` (self — additive)

**Subscribe/Unsubscribe with deferred cleanup pattern** (`internal/relay/server.go` lines 83-85):
```go
hub.Subscribe(sub)
defer hub.Unsubscribe(sub)
defer conn.CloseNow()
```

The `broadcastMeta` call must happen AFTER `hub.Subscribe(sub)` returns (not under hub.mu) to avoid deadlock. Mirror the existing pattern: call hub methods, then release the lock before side effects.

**Non-blocking broadcast pattern** (`internal/relay/hub.go` lines 154-167):
```go
func (h *Hub) broadcast(frame []byte) {
    h.mu.Lock()
    defer h.mu.Unlock()

    for sub := range h.subscribers {
        select {
        case sub.Msgs <- frame:
        default:
            go sub.CloseSlow()
        }
    }
}
```

Add a `BroadcastMeta(frame []byte)` method on `Hub` using this identical non-blocking send pattern. `MsgMeta` frames must never block the PTY drain loop. This method belongs in `hub.go`, not `server.go`.

**handleSession switch-case extension pattern** (`internal/relay/server.go` lines 106-122):
```go
switch msgType {
case MsgInput:
    if !sub.ReadOnly {
        _ = hub.WriteInput(payload)
    }
case MsgResize2:
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.ResizeClient(sub, int(cols), int(rows))
    }
case MsgPing:
    // Keep-alive — no-op.
}
```

The server-side push of `MsgMeta` does not add a new case here (it is server-to-client, not client-to-server). Instead, the push hook is added at the Subscribe and Unsubscribe call sites in `handleSession`, calling `s.broadcastMeta(hub)` after each.

---

### `cmd_attach.go` (additive modification)

**Analog:** `cmd_attach.go` (self — additive)

**TTY detection pattern** (`cmd_attach.go` lines 58-61):
```go
// Must be run in an interactive terminal.
if !term.IsTerminal(int(os.Stdin.Fd())) {
    return fmt.Errorf("attach: stdin is not a terminal")
}
```

The status bar adds a second TTY check against `os.Stdout.Fd()` (not `os.Stdin.Fd()`). Pattern:
```go
var bar *statusbar.Bar
if term.IsTerminal(int(os.Stdout.Fd())) {
    bar = statusbar.New(os.Stdout, statusbar.Options{ ... })
    bar.Start()
    defer bar.Stop()
}
```

**Flag parsing pattern** (`cmd_attach.go` lines 32-50):
```go
for _, arg := range args[1:] {
    if arg == "--readonly" {
        readOnly = true
    } else if len(arg) > 9 && arg[:9] == "--client=" {
        clientName = arg[9:]
    } else if len(arg) > 13 && arg[:13] == "--detach-key=" {
        // ...
    }
}
```

Add `--status-top` flag using the same `if arg == "--status-top"` pattern. Pass the parsed position to `statusbar.Options.Position`.

**wsOutputPump switch extension pattern** (`cmd_attach.go` lines 319-334):
```go
func wsOutputPump(ctx context.Context, conn *websocket.Conn, w io.Writer) error {
    for {
        _, msg, err := conn.Read(ctx)
        if err != nil {
            return err
        }
        msgType, payload, ferr := relay.ParseFrame(msg)
        if ferr != nil {
            continue
        }
        if msgType == relay.MsgOutput {
            if _, werr := w.Write(payload); werr != nil {
                return werr
            }
        }
    }
}
```

Extend to a switch statement adding a `relay.MsgMeta` case that unmarshals JSON and calls `bar.SetViewerCount()`. The `w io.Writer` parameter becomes a mutex-wrapped writer (`lockedWriter`) passed to both `wsOutputPump` and `Bar.New()` to serialize all stdout writes.

**lockedWriter pattern** (new, no existing analog — use sync.Mutex wrapper):
```go
type lockedWriter struct {
    mu sync.Mutex
    w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (int, error) {
    lw.mu.Lock()
    defer lw.mu.Unlock()
    return lw.w.Write(p)
}
```

This is the mutex-wrapped `io.Writer` that serializes concurrent writes from `wsOutputPump` and `bar.draw()`. The `safeBuf` type in `cmd_attach_test.go` (lines 168-184) is the same pattern — use it as the reference.

---

## Shared Patterns

### sync.Mutex for shared mutable state
**Source:** `internal/relay/hub.go` lines 99-126 (`ResizeClient`)
**Apply to:** `internal/statusbar/bar.go` (`viewerCount`, `connState`, `cols`, `rows` fields), `lockedWriter` in `cmd_attach.go`

Pattern: Lock, copy to locals, unlock, then use locals. Never hold the lock during I/O.

### sync.Once for idempotent teardown
**Source:** `internal/relay/hub.go` lines 40-41, 169-177 (`closeOnce`, `Shutdown`)
**Apply to:** `internal/statusbar/bar.go` (`Bar.Stop()`)

```go
type Hub struct {
    closeOnce sync.Once
    // ...
}

func (h *Hub) Shutdown() {
    h.closeOnce.Do(func() {
        // ... teardown ...
    })
}
```

### context.WithCancel goroutine lifecycle
**Source:** `cmd_attach_unix.go` lines 18-37 (`watchResize`), `cmd_attach.go` lines 106-107
**Apply to:** `internal/statusbar/bar.go` (`tickLoop` goroutine started in `Bar.Start()`, cancelled in `Bar.Stop()`)

Pattern: `ctx, cancel = context.WithCancel(context.Background())` in Start; `cancel()` + `wg.Wait()` in Stop.

### Non-blocking channel send for broadcast
**Source:** `internal/relay/hub.go` lines 154-167 (`broadcast`)
**Apply to:** `internal/relay/hub.go` (new `BroadcastMeta` method)

Pattern: `select { case ch <- frame: default: go closeSlow() }` — slow clients are disconnected, never block the producer.

### External test package with mock io.Writer
**Source:** `internal/status/detector_test.go` line 1 (`package status_test`)
**Apply to:** `internal/statusbar/bar_test.go` (`package statusbar_test`)

### time.After timeout in tests
**Source:** `internal/relay/hub_test.go` lines 36-50
**Apply to:** `internal/statusbar/bar_test.go` — any test awaiting ticker or goroutine completion

### io.Pipe for test infrastructure
**Source:** `cmd_attach_test.go` lines 26-47 (`setupAttachTest`), `internal/relay/hub_test.go` lines 12-17 (`makeTestHub`)
**Apply to:** `internal/statusbar/bar_test.go` — use `io.Pipe()` or `bytes.Buffer` as the `io.Writer` substitute for stdout in tests

---

## No Analog Found

No files in this phase are entirely without analog. All files are either additive modifications to existing files (with themselves as analogs) or new internal packages with close role-match analogs.

---

## Metadata

**Analog search scope:** `/Users/ken/dev/agenthub` — all `.go` files (87 files scanned)
**Key files read:** `cmd_attach.go`, `cmd_attach_unix.go`, `cmd_attach_test.go`, `internal/relay/protocol.go`, `internal/relay/hub.go`, `internal/relay/server.go`, `internal/relay/protocol_test.go`, `internal/relay/hub_test.go`, `internal/status/detector.go`, `internal/status/detector_test.go`
**Pattern extraction date:** 2026-04-15
