# Phase 2: Session Registry + WebSocket Relay - Research

**Researched:** 2026-03-18
**Domain:** Go WebSocket fan-out hub, per-session broadcast, scrollback buffer, reconnect protocol
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SESS-03 | User can reattach to a running session after reopening the window | Scrollback buffer (bounded `bytes.Buffer`) holds recent PTY output; on reconnect the buffer is replayed before live streaming resumes; PTY process is never restarted |
</phase_requirements>

---

## Summary

Phase 2 wires the Phase 1 PTY backend into a WebSocket relay layer. The core deliverable is a per-session `Hub` that drains PTY output in a single goroutine, fans it out to N concurrently connected WebSocket clients, and replays a bounded scrollback buffer to reconnecting clients. No external message broker is needed — Go channels provide all the synchronization.

The WebSocket library decision is clear: use `github.com/coder/websocket` v1.8.14 (the successor to `nhooyr.io/websocket`). Gorilla/websocket is archived (no security patches); coder/websocket supports concurrent writes natively, uses `context.Context` throughout, and is actively maintained. This phase also defines the binary framing protocol that Phase 3 (Wails frontend) and Phase 4 (web server) will both consume — getting this right now avoids a rewrite later.

The three SESS-03 success criteria are all testable without a real PTY on CI: the tests create an `io.Pipe` in place of a PTY, a test WebSocket server, and two WebSocket client goroutines. The protocol is simple enough that integration tests are reliable within 30 seconds.

**Primary recommendation:** One `Hub` per session, owned by the `SessionHub` manager. The hub runs a single `drain` goroutine reading from `Session.Read`, writes to a bounded scrollback buffer, and fan-outs to registered client channels. HTTP routing uses the Go 1.22 `net/http` ServeMux with `GET /sessions/{id}/ws` pattern and `r.PathValue("id")`.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.14 | WebSocket upgrade, read/write | Concurrent writes built-in; context-aware; gorilla is archived; actively maintained by Coder |
| `net/http` (stdlib) | Go 1.22+ | HTTP server + ServeMux routing | Go 1.22 adds `{id}` path wildcards + `r.PathValue`; no third-party router needed |
| `sync` (stdlib) | Go stdlib | Hub client map protection | `sync.RWMutex` for the subscriber map; `sync.Mutex` for scrollback buffer |
| `bytes` (stdlib) | Go stdlib | Scrollback buffer | `bytes.Buffer` for the bounded output replay buffer |
| `context` (stdlib) | Go stdlib | Connection lifecycle, cancellation | Hub and per-connection contexts |
| `io` (stdlib) | Go stdlib | PTY read/write bridge | `Session` implements `io.ReadWriter`; already defined in Phase 1 |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` (stdlib) | Go stdlib | Control message encoding (resize, title) | Used for non-binary control frames only |
| `github.com/coder/websocket/wsjson` | bundled with v1.8.14 | JSON read/write helpers | Use for control messages; raw `Conn.Write` for binary PTY data |
| `testing` (stdlib) | Go stdlib | Test framework | Already used in Phase 1 |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `coder/websocket` | `gorilla/websocket` | gorilla archived late 2022; no security patches; gorilla requires manual mutex for concurrent writes |
| `coder/websocket` | `gobwas/ws` | gobwas has zero-alloc performance but a low-level API that requires much more code; overkill for this use case |
| Go 1.22 ServeMux `{id}` wildcard | `gorilla/mux` or `chi` | No extra dependency needed; Go 1.22 stdlib is sufficient |
| `bytes.Buffer` scrollback | ring buffer library | `bytes.Buffer` with a size cap is sufficient; ring buffer adds a dependency for a case that only matters with millions of sessions |
| In-process channels | Redis pub/sub | Redis adds an external dependency; this app is single-process, local-first |

**Installation:**

```bash
go get github.com/coder/websocket@v1.8.14
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── pty/                 # Phase 1 (complete)
│   ├── backend.go       # SessionBackend interface, Session struct
│   ├── registry.go      # SessionRegistry (in-memory map + mutex)
│   └── ...
├── relay/               # Phase 2: WebSocket relay layer
│   ├── hub.go           # Hub: per-session fan-out to N WebSocket clients
│   ├── manager.go       # HubManager: creates/looks up Hubs per session ID
│   ├── protocol.go      # Binary framing constants and helpers
│   ├── scrollback.go    # Bounded scrollback buffer
│   ├── server.go        # HTTP server with ServeMux routes
│   ├── hub_test.go      # Unit + integration tests for Hub
│   ├── protocol_test.go # Tests for framing encode/decode
│   └── server_test.go   # HTTP handler tests
cmd/
└── agenthub/
    └── main.go          # Updated: wire relay into existing PTY backend
```

### Pattern 1: Per-Session Hub with Single Drain Goroutine

**What:** Each session has exactly one `Hub`. A single `drain` goroutine reads from `Session.Read` (the PTY master) and pushes bytes to all registered client channels. The drain goroutine is the only reader of the PTY — PTY reads are destructive, so this is mandatory.

**When to use:** Created when a session is created (or lazily on first WebSocket connect). Destroyed when the session is killed.

**Key invariant from Phase 1 research:** "Reading PTY from multiple goroutines — PTY reads are destructive. One `drainPTY` goroutine per session, fan-out to registered clients via channels."

```go
// Source: derived from coder/websocket chat example + go fan-out patterns
type Hub struct {
    sessionID  string
    session    *pty.Session    // Phase 1 Session (io.ReadWriter)
    scrollback *Scrollback     // bounded replay buffer

    mu          sync.Mutex
    subscribers map[*subscriber]struct{}

    done   chan struct{}        // closed when hub is shut down
    doneMu sync.Mutex
    closed bool
}

type subscriber struct {
    msgs      chan []byte       // outbound PTY data frames
    closeSlow func()            // called when channel is full (client too slow)
}

func (h *Hub) run() {
    buf := make([]byte, 32*1024)
    for {
        n, err := h.session.Read(buf)
        if n > 0 {
            frame := makeOutputFrame(buf[:n])
            h.scrollback.Append(frame)
            h.broadcast(frame)
        }
        if err != nil {
            h.shutdown()
            return
        }
    }
}

func (h *Hub) broadcast(frame []byte) {
    h.mu.Lock()
    defer h.mu.Unlock()
    for sub := range h.subscribers {
        select {
        case sub.msgs <- frame:
        default:
            go sub.closeSlow()
        }
    }
}
```

### Pattern 2: WebSocket Client Handler

**What:** Each WebSocket connection is served by a handler that registers with the Hub, replays scrollback, then pumps messages until the connection closes.

**When to use:** Every incoming `GET /sessions/{id}/ws` request.

```go
// Source: coder/websocket v1.8.14 API (pkg.go.dev/github.com/coder/websocket)
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    hub, ok := s.manager.Get(sessionID)
    if !ok {
        http.Error(w, "session not found", http.StatusNotFound)
        return
    }

    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        // Allow connections from Wails WebView origin
        InsecureSkipVerify: true, // Phase 3 will tighten this
    })
    if err != nil {
        return
    }
    defer conn.CloseNow()

    ctx := r.Context()

    // Register subscriber
    sub := &subscriber{
        msgs: make(chan []byte, 256),
    }
    sub.closeSlow = func() {
        conn.Close(websocket.StatusPolicyViolation, "connection too slow")
    }
    hub.Subscribe(sub)
    defer hub.Unsubscribe(sub)

    // Replay scrollback before live stream
    for _, frame := range hub.scrollback.Snapshot() {
        if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
            return
        }
    }

    // Pump: forward incoming client messages to PTY, outgoing hub frames to client
    go func() {
        for {
            _, p, err := conn.Read(ctx)
            if err != nil {
                return
            }
            // First byte is message type (see Protocol section)
            if len(p) > 0 {
                handleClientMessage(hub, p)
            }
        }
    }()

    for {
        select {
        case frame := <-sub.msgs:
            if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
                return
            }
        case <-ctx.Done():
            return
        case <-hub.done:
            conn.Close(websocket.StatusGoingAway, "session ended")
            return
        }
    }
}
```

### Pattern 3: Binary Framing Protocol

**What:** A single-byte type prefix followed by payload. All frames are sent as `websocket.MessageBinary`. This matches the GoTTY/WebTTY convention and is simple enough to implement on both ends.

**When to use:** Every WebSocket message in both directions.

```go
// Source: derived from GoTTY webtty protocol analysis + project requirements
// protocol.go

const (
    // Server → Client message types
    MsgOutput  byte = 0x01  // PTY output data (raw bytes, ANSI escape sequences intact)
    MsgResize  byte = 0x02  // Terminal resize event: 4 bytes (uint16 cols + uint16 rows, big-endian)
    MsgTitle   byte = 0x03  // Session title change (UTF-8 string)

    // Client → Server message types
    MsgInput   byte = 0x10  // User keystrokes / paste data (raw bytes → PTY write)
    MsgResize2 byte = 0x11  // Client-reported resize: 4 bytes (uint16 cols + uint16 rows, big-endian)
    MsgPing    byte = 0x12  // Keepalive ping
)

func makeOutputFrame(data []byte) []byte {
    frame := make([]byte, 1+len(data))
    frame[0] = MsgOutput
    copy(frame[1:], data)
    return frame
}

func makeResizeFrame(cols, rows uint16) []byte {
    frame := make([]byte, 5) // 1 type + 2 cols + 2 rows
    frame[0] = MsgResize
    binary.BigEndian.PutUint16(frame[1:3], cols)
    binary.BigEndian.PutUint16(frame[3:5], rows)
    return frame
}
```

**Rationale for binary over JSON:** PTY output is a byte stream that contains arbitrary ANSI escape sequences. Base64-encoding it (as GoTTY does) adds 33% overhead. Raw binary with a type prefix is smaller and simpler to parse in xterm.js.

### Pattern 4: Bounded Scrollback Buffer

**What:** A `Scrollback` struct wraps a byte slice with a maximum size cap. When the cap is exceeded, old data is discarded from the front. On reconnect, `Snapshot()` returns a copy for replay.

**When to use:** Created with each Hub. Append on every PTY read. Snapshot on every new WebSocket client connect.

```go
// Source: derived from WebSocket reconnection patterns + project requirements
// scrollback.go

const DefaultScrollbackBytes = 256 * 1024 // 256 KB per session

type Scrollback struct {
    mu   sync.Mutex
    buf  []byte
    max  int
}

func NewScrollback(maxBytes int) *Scrollback {
    return &Scrollback{max: maxBytes}
}

func (s *Scrollback) Append(frame []byte) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.buf = append(s.buf, frame...)
    if len(s.buf) > s.max {
        // Discard oldest bytes to stay within cap
        s.buf = s.buf[len(s.buf)-s.max:]
    }
}

func (s *Scrollback) Snapshot() []byte {
    s.mu.Lock()
    defer s.mu.Unlock()
    out := make([]byte, len(s.buf))
    copy(out, s.buf)
    return out
}
```

**Note:** Scrollback stores complete raw frames (type byte + data). Replaying raw frames means xterm.js receives the same byte sequence a live client would. This correctly restores terminal state including cursor position and color for CLIs that redraw the screen (like Claude Code's TUI).

### Pattern 5: Go 1.22 ServeMux Routing

**What:** Use `net/http.NewServeMux()` with Go 1.22 wildcard patterns.

**When to use:** HTTP server setup in `server.go`.

```go
// Source: go.dev/blog/routing-enhancements (Go 1.22)
mux := http.NewServeMux()
mux.HandleFunc("GET /sessions/{id}/ws", s.handleSession)
mux.HandleFunc("GET /sessions", s.handleListSessions)
mux.HandleFunc("POST /sessions", s.handleCreateSession)

// Extract path variable in handler:
sessionID := r.PathValue("id")
```

### Anti-Patterns to Avoid

- **Multiple PTY reader goroutines:** PTY reads are destructive — bytes read by one goroutine are gone. Only the Hub's drain goroutine may call `session.Read`. All clients receive data through the hub's subscriber channels.
- **Gorilla/websocket for new code:** The repository is archived. Do not introduce it.
- **Blocking broadcast:** If one slow subscriber blocks, all subscribers stall. Always use non-blocking `select` with `default: go sub.closeSlow()`.
- **Replaying scrollback after starting live stream:** Race condition — frames produced between snapshot and subscription are lost. Always subscribe before snapshotting. The Hub's Subscribe call must happen before Snapshot() is called.
- **JSON-encoding PTY output:** GoTTY base64-encodes output for text WebSocket frames. This is unnecessary overhead. Use `websocket.MessageBinary` with a type-prefix frame.
- **Storing scrollback as individual frames in a slice:** At high output rates, the slice grows unboundedly. Store raw bytes with a size cap, not a frame slice.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WebSocket upgrade + framing | Custom HTTP upgrade, custom frame parser | `coder/websocket` | RFC 6455 has edge cases (masking, fragmentation, ping/pong, close handshake); coder/websocket passes autobahn-testsuite |
| Concurrent WebSocket writes | Manual mutex around `conn.Write` | `coder/websocket` built-in | coder/websocket serializes concurrent writes internally — no user-level mutex needed |
| HTTP routing with path variables | Third-party router (gorilla/mux, chi) | Go 1.22 `net/http` ServeMux | Go 1.22 added `{id}` wildcards and `r.PathValue`; no external dependency needed |

**Key insight:** The fan-out hub itself must be hand-rolled because it is deeply coupled to this project's session model — but the WebSocket framing, upgrade, and write serialization must not be hand-rolled.

---

## Common Pitfalls

### Pitfall 1: Multiple PTY Readers

**What goes wrong:** If two goroutines call `session.Read` concurrently, each gets a different chunk of the byte stream. Both clients receive partial output. No errors are returned — the corruption is silent.

**Why it happens:** `session.Read` delegates to `gopty.Pty.Read`, which is a standard OS file descriptor read. Each read call consumes bytes.

**How to avoid:** Enforce exactly one drain goroutine per Hub. All client goroutines receive frames via channels, never by reading the PTY directly.

**Warning signs:** Clients connected to the same session see different output; output is randomly split between clients.

### Pitfall 2: Slow Client Blocks Fast Clients

**What goes wrong:** If a subscriber's `msgs` channel is full and the broadcast is blocking, all other subscribers stall until the slow client drains its channel.

**Why it happens:** Go channel sends block when the channel buffer is full.

**How to avoid:** Always use a non-blocking send in broadcast: `select { case sub.msgs <- frame: default: go sub.closeSlow() }`. The buffered channel (256 messages) gives the client time to catch up; if it overflows, the client is disconnected.

**Warning signs:** All clients freeze simultaneously when one tab is behind.

### Pitfall 3: Subscribe-After-Snapshot Race

**What goes wrong:** If the sequence is: (1) take scrollback snapshot, (2) subscribe to hub, then any output produced between (1) and (2) is never seen by the client.

**Why it happens:** The drain goroutine continues producing frames between the snapshot and subscription.

**How to avoid:** Subscribe to the hub FIRST, then snapshot. Since the hub appends to scrollback under the same mutex as broadcasting, any frame added after subscription will arrive via the channel. Frames in the snapshot are delivered first (they are older); then live frames arrive via channel.

**Warning signs:** Client appears to miss the last few lines of output visible to an already-connected client.

### Pitfall 4: Scrollback Replays Corrupted Terminal State

**What goes wrong:** If scrollback stores only partial output (e.g., cut in the middle of an ANSI escape sequence), xterm.js enters a corrupted parse state. Colors and cursor position look wrong.

**Why it happens:** When the scrollback buffer is truncated (oldest bytes discarded to fit within the cap), the truncation point may be mid-sequence.

**How to avoid:** This is an inherent limitation of raw byte buffers. For a 256 KB cap with typical AI CLI output rates, the buffer will contain many full screens and the chance of a visible corruption is low. Document the known limitation. Phase 3 may mitigate this by implementing an xterm.js `reset()` before replay if the terminal state is clearly wrong.

**Warning signs:** Reconnecting clients see garbled color output at the start of the session.

### Pitfall 5: gorilla/websocket Concurrent Write Panic

**What goes wrong:** If gorilla/websocket is used (not coder/websocket), calling `conn.WriteMessage` from multiple goroutines simultaneously causes a panic: `concurrent write to websocket connection`.

**Why it happens:** gorilla/websocket does not serialize concurrent writes.

**How to avoid:** Use `coder/websocket` exclusively. If gorilla is somehow introduced, add a `sync.Mutex` wrapping every write.

**Warning signs:** Random panics with `concurrent write to websocket connection` in the stack trace.

### Pitfall 6: Hub Not Cleaned Up After Session Kill

**What goes wrong:** A session is killed via `SessionBackend.Kill`, but the Hub continues running its drain goroutine, which blocks on `session.Read` forever. Memory leaks accumulate.

**Why it happens:** Phase 1 `Session.Read` will eventually return an error when the PTY is closed (after `killSession` calls `s.pty.Close()`). But the Hub must handle this error by calling `hub.shutdown()`.

**How to avoid:** The drain goroutine must handle EOF/error from `session.Read` by calling `hub.shutdown()`. `HubManager` must also expose a `Remove(sessionID)` method that the app calls alongside `backend.Kill`.

**Warning signs:** `go tool pprof` shows goroutine counts growing after session kills.

---

## Code Examples

### Accept WebSocket Connection (coder/websocket v1.8.14)

```go
// Source: pkg.go.dev/github.com/coder/websocket
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    InsecureSkipVerify: true, // loosen for local Wails WebView; Phase 4 will set OriginPatterns
})
if err != nil {
    return
}
defer conn.CloseNow()
```

### Write Binary Frame (concurrent-safe)

```go
// Source: pkg.go.dev/github.com/coder/websocket
// coder/websocket supports concurrent calls to Write without a mutex
if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
    return
}
```

### Read from Client

```go
// Source: pkg.go.dev/github.com/coder/websocket
msgType, p, err := conn.Read(ctx)
if err != nil {
    return
}
if msgType == websocket.MessageBinary && len(p) > 0 {
    // p[0] is message type byte; p[1:] is payload
}
```

### Go 1.22 Path Variable Extraction

```go
// Source: go.dev/blog/routing-enhancements
mux.HandleFunc("GET /sessions/{id}/ws", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    // ...
})
```

### Non-Blocking Hub Broadcast

```go
// Source: coder/websocket chat example (internal/examples/chat/chat.go)
for sub := range h.subscribers {
    select {
    case sub.msgs <- frame:
    default:
        go sub.closeSlow()
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `gorilla/websocket` | `coder/websocket` (formerly `nhooyr.io/websocket`) | gorilla archived 2022; Coder took over nhooyr 2024 | gorilla requires manual write mutex; coder/websocket handles concurrent writes; gorilla has no active security maintenance |
| Third-party HTTP routers (gorilla/mux, chi) for path variables | Go 1.22 stdlib `net/http` ServeMux with `{id}` wildcards | Go 1.22 (Feb 2024) | No third-party router needed for simple REST + WebSocket routing |
| Base64-encoded PTY output over text WebSocket frames (GoTTY approach) | Raw binary frames with type-byte prefix over `MessageBinary` | Established improvement | 33% smaller frames; simpler client-side parsing in xterm.js |
| Global hub broadcasting to all sessions | Per-session Hub with individual subscriber maps | Standard pattern | Correct isolation — clients only receive output from their own session |

**Deprecated/outdated:**
- `gorilla/websocket`: Archived. Do not use for new code.
- `nhooyr.io/websocket`: Module path changed to `github.com/coder/websocket`. Use the new path.

---

## Open Questions

1. **Scrollback truncation mid-ANSI-sequence**
   - What we know: Raw byte truncation may cut an ANSI escape sequence, corrupting xterm.js state on reconnect
   - What's unclear: How often this will occur in practice with Claude Code's TUI output at 256 KB cap
   - Recommendation: Accept the limitation for Phase 2. Phase 3 can call `xterm.reset()` before replay if the buffer was truncated. Document the boundary.

2. **Origin validation for Wails WebView**
   - What we know: coder/websocket blocks cross-origin requests by default; `InsecureSkipVerify: true` disables this
   - What's unclear: What origin header Wails' WebView sends for localhost connections; whether a Wails-specific `OriginPatterns` value can be set instead of skipping verification entirely
   - Recommendation: Use `InsecureSkipVerify: true` in Phase 2 (no browser exposure yet). Research Wails WebView origin in Phase 3 planning and tighten.

3. **Hub creation timing — eager vs lazy**
   - What we know: Hub can be created when the session is created (eager) or when the first WebSocket client connects (lazy)
   - What's unclear: Whether an always-running drain goroutine adds significant overhead for sessions with no connected clients
   - Recommendation: Create Hub eagerly at session creation time, matching the pattern that HubManager.Create is called alongside backend.Create. Drain goroutine cost is negligible (blocks on `session.Read`). This simplifies reconnect handling — the hub always exists.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib) |
| Config file | none — go test discovers by convention |
| Quick run command | `go test ./internal/relay/... -v -timeout 30s` |
| Full suite command | `go test ./... -v -timeout 60s -race` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| SESS-03 (criterion 1) | Two WebSocket clients receive the same PTY output simultaneously without loss | integration | `go test ./internal/relay/... -run TestHub_TwoClientsFanOut -v` | Wave 0 |
| SESS-03 (criterion 2) | Reconnecting client receives scrollback and resumes live output without PTY restart | integration | `go test ./internal/relay/... -run TestHub_ReconnectScrollback -v` | Wave 0 |
| SESS-03 (criterion 3) | Input from any client reaches the PTY and output is visible to all clients | integration | `go test ./internal/relay/... -run TestHub_InputFanOut -v` | Wave 0 |

**Test approach:** Use `io.Pipe` as the PTY substitute (writes to the pipe simulate PTY output; the Hub's drain goroutine reads from the read end). Use `net/http/httptest` to host a WebSocket server. Use `coder/websocket.Dial` for test clients. No real PTY or real CLI required on CI.

### Sampling Rate

- **Per task commit:** `go test ./internal/relay/... -v -timeout 30s -race`
- **Per wave merge:** `go test ./... -v -timeout 60s -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/relay/hub.go` — Hub struct, Subscribe/Unsubscribe, broadcast, drain goroutine
- [ ] `internal/relay/manager.go` — HubManager: Create/Get/Remove per session
- [ ] `internal/relay/protocol.go` — MsgOutput, MsgInput, MsgResize constants + frame helpers
- [ ] `internal/relay/scrollback.go` — Scrollback struct with Append/Snapshot/size cap
- [ ] `internal/relay/server.go` — HTTP server with ServeMux routes and WebSocket handler
- [ ] `internal/relay/hub_test.go` — TestHub_TwoClientsFanOut, TestHub_ReconnectScrollback, TestHub_InputFanOut, TestHub_SlowClientDisconnected
- [ ] `internal/relay/protocol_test.go` — Frame encode/decode round-trip tests
- [ ] `internal/relay/scrollback_test.go` — Append/Snapshot, truncation boundary test

*(Framework install: none — Go stdlib `testing`; `coder/websocket` added to go.mod)*

---

## Sources

### Primary (HIGH confidence)

- `pkg.go.dev/github.com/coder/websocket` — Full API: Accept, Conn.Read/Write, MessageBinary, AcceptOptions, StatusCode constants; v1.8.14
- `github.com/coder/websocket/blob/v1.8.14/internal/examples/chat/chat.go` — Official chat example: subscriber struct, buffered channel, non-blocking broadcast, closeSlow pattern
- `go.dev/blog/routing-enhancements` — Go 1.22 ServeMux wildcard patterns and `r.PathValue`
- Phase 1 RESEARCH.md + SUMMARY.md — Session.Read/Write interface, registry design, anti-patterns

### Secondary (MEDIUM confidence)

- `websocket.org/guides/languages/go/` — coder/websocket vs gorilla comparison; gorilla archive confirmation; concurrent write behavior
- `websocket.org/guides/reconnection/` — Sequence-based message replay pattern; server-side buffer with TTL tradeoffs
- GoTTY webtty.go analysis — Single-byte type prefix framing convention (MsgOutput, MsgInput, MsgResize); base64 encoding decision
- `forum.golangbridge.org/t/websocket-in-2025/38671` — Community confirmation of coder/websocket as current standard for new Go projects

### Tertiary (LOW confidence)

- None — all critical claims verified from primary or secondary sources.

---

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — coder/websocket API verified from pkg.go.dev; Go 1.22 ServeMux from official Go blog
- Architecture: HIGH — patterns derived from official coder/websocket chat example + established Go fan-out idioms
- Pitfalls: HIGH — each pitfall is either a PTY read property (destructive reads), a channel semantics property (blocking), or traced to library behavior (gorilla concurrent panic)
- Protocol framing: MEDIUM — single-byte prefix convention derived from GoTTY analysis; not an official standard, but widely used in web terminal tooling

**Research date:** 2026-03-18
**Valid until:** 2026-09-18 (coder/websocket is stable; Go stdlib routing is stable)
