# Phase 75: CLI Status Bar - Research

**Researched:** 2026-04-14
**Domain:** Terminal ANSI escape sequences, DECSTBM scroll regions, Go status bar rendering, relay protocol extension
**Confidence:** HIGH

## Summary

Phase 75 adds a persistent bottom status bar to `agenthub attach` using standard VT100/ANSI
terminal techniques. The bar shows: session name, agent type, hostname, detach hint, elapsed
time, and viewer count. It refreshes on a timer without corrupting scrollable PTY output.

The canonical approach for a non-corrupting status bar is the **DECSTBM scroll region technique**:
set the terminal's scrolling region to rows 1 through (height-1), draw the status bar on the last
row, and save/restore the cursor across each draw cycle. This is the mechanism tmux uses for its
own status bar. All terminal output from the session scrolls within the restricted region, leaving
the last row permanently reserved for status. On detach, the scroll region is reset to the full
terminal size and the last row is cleared.

The implementation lives entirely in a new `internal/statusbar` package that `cmd_attach.go`
instantiates. The package must be self-contained: it must not know about the relay or daemon
packages — it only needs a `io.Writer` (stdout) and a callback to query current state.

Viewer count live-update is the one non-trivial piece: the status bar must receive viewer count
changes without polling the daemon API on every tick. The cleanest approach is a new relay frame
type (`MsgMeta`, `0x20`) that the relay server pushes to connected clients when subscriber count
changes. `wsOutputPump` in `cmd_attach.go` intercepts these frames and updates a shared variable
that the status bar reads on each tick. This avoids a second HTTP connection or goroutine and
keeps the protocol self-contained.

**Primary recommendation:** Implement `internal/statusbar` with a `Bar` type that uses DECSTBM
scroll-region technique; extend the relay protocol with a `MsgMeta` push frame for live viewer
count; hook the bar into `cmd_attach.go` and `cmdAttachRemote`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Status bar rendering (ANSI output) | `internal/statusbar` | — | Self-contained drawing; no coupling to relay or daemon |
| Scroll region management (DECSTBM) | `internal/statusbar` | — | Bar owns terminal region lifecycle: set on start, reset on stop |
| Viewer count update channel | relay.Server (push MsgMeta) | cmd_attach.go (intercept) | Server knows subscriber count; push avoids polling |
| TTY detection (SB-03) | cmd_attach.go | — | `term.IsTerminal` already called before raw mode |
| Elapsed time tracking | `internal/statusbar` | — | Time.Since(session.CreatedAt) computed on each tick |
| Status bar position flag (SB-06) | cmd_attach.go | `internal/statusbar` | CLI parses `--status-top`, passes to Bar constructor |
| Cleanup on detach (SB-07) | `internal/statusbar` (Bar.Stop()) | cmd_attach.go (defer) | Bar.Stop resets scroll region, clears bar line |
| SB-05 connection state display | cmd_attach.go → statusbar | relay.Server (MsgMeta) | WebSocket ping/pong latency measurable; reconnect state from dial errors |

## Standard Stack

### Core (all already in go.mod)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/term` | v0.41.0 | `GetSize(fd)` for terminal dimensions, `IsTerminal(fd)` for TTY check | [VERIFIED: go.mod, cmd_attach.go] Already used for raw mode |
| `fmt` (stdlib) | go1.26 | Write ANSI escape sequences as format strings | [VERIFIED: cmd_attach.go] Pattern established |
| `time` (stdlib) | go1.26 | `time.NewTicker` for refresh loop, `time.Since` for elapsed time | [VERIFIED: app.go] Ticker pattern used throughout codebase |
| `sync` (stdlib) | go1.26 | `sync.Mutex` for shared state (viewer count, connection state) | [VERIFIED: hub.go] Mutex pattern established |
| `os` (stdlib) | go1.26 | `os.Stdout.Fd()` for terminal fd | [VERIFIED: cmd_attach.go] |

### No New Dependencies Required

All functionality can be implemented with stdlib and existing go.mod dependencies.
[VERIFIED: codebase — `golang.org/x/term` already provides `GetSize` and `IsTerminal`]

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled ANSI escape strings | `github.com/charmbracelet/x/ansi` | External dep adds weight; the escape sequences needed (DECSTBM, cursor save/restore, cursor move) are 5-7 constants that fit in a single file. Not worth a dependency. |
| DECSTBM scroll region | Alternate-screen buffer (smcup) | Alternate screen hides session output entirely — wrong for a pass-through status bar |
| Timer-based viewer count polling | MsgMeta push from relay server | Polling requires a second goroutine calling daemon API on every tick; push keeps state accurate with zero polling overhead |

## Architecture Patterns

### System Architecture Diagram

```
cmd_attach.go                 internal/statusbar           relay.Server
─────────────                 ──────────────────           ────────────

agenthub attach
  │
  ├─ term.IsTerminal(stdout) ─── false ──► skip Bar creation (SB-03)
  │
  ├─ term.IsTerminal(stdout) ─── true ───► Bar.Start(stdout, cols, rows, opts)
  │                                           │
  │                                           ├─ Write DECSTBM: ESC[1;{rows-1}r    ← restrict scroll region
  │                                           ├─ Move cursor to row {rows}
  │                                           ├─ Write initial status line
  │                                           └─ Start ticker goroutine (1s interval)
  │                                                 │
  │                                        on tick: │
  │                                                 ├─ Save cursor: ESC[s
  │                                                 ├─ Move to row {rows}: ESC[{rows};1H
  │                                                 ├─ Clear line: ESC[2K
  │                                                 ├─ Write status text (truncated to cols)
  │                                                 └─ Restore cursor: ESC[u
  │
  ├─ wsOutputPump ◄─── WebSocket frames ─────────────────────────────────── relay.Server
  │     │                                                                         │
  │     ├─ MsgOutput(0x01) ──► w.Write(payload) [normal PTY output]              │
  │     │                                                                         │
  │     └─ MsgMeta(0x20) ───► parse JSON payload {viewerCount: N}                │
  │              │              Bar.SetViewerCount(N)                              │
  │              │                                                                 │
  │              └───────────────────────────────── pushed by relay.Server        │
  │                                                 on subscribe/unsubscribe       │
  │                                                                                │
  └─ on detach/exit:
       Bar.Stop()
         ├─ Stop ticker
         ├─ Write DECSTBM: ESC[r         ← reset scroll region (full screen)
         ├─ Move to row {rows}: ESC[{rows};1H
         ├─ Clear line: ESC[2K           ← erase status bar line
         └─ Restore cursor: ESC[u        ← leave cursor in scrollable area
```

### Recommended Project Structure

```
internal/statusbar/
├── bar.go           # Bar type: Start, Stop, SetViewerCount, SetConnectionState, draw loop
├── bar_test.go      # Unit tests: TTY suppression, draw output validation, cleanup
internal/relay/
├── protocol.go      # Add: MsgMeta = 0x20, MakeMeta(payload []byte), ParseMeta helpers
├── server.go        # Add: push MsgMeta to all subscribers on subscriber count change
cmd_attach.go        # Add: Bar instantiation, MsgMeta intercept in wsOutputPump
                     # Add: --status-top flag parsing
```

### Pattern 1: DECSTBM Scroll Region Status Bar

**What:** Reserve the last row of the terminal for the status bar by setting the scrolling
region to rows 1 through (height-1). PTY output scrolls within this region. Status bar is
drawn on row `height` and never scrolled away.

**When to use:** Any "persistent bottom bar" in a terminal pass-through attach session.

**ANSI constants (no external dep needed):**

```go
// Source: VT100 specification (vt100.net/docs/vt510-rm/DECSTBM.html)
// These are the only escape sequences needed for a bottom status bar.
const (
    // setScrollRegion sets the scrollable region to rows [top, bottom] (1-indexed).
    // After this, terminal output scrolls only within rows top..bottom.
    // Side effect: cursor moves to row 1, col 1.
    setScrollRegion = "\033[%d;%dr"

    // resetScrollRegion resets scrolling to the full terminal.
    resetScrollRegion = "\033[r"

    // cursorSave / cursorRestore (DECSC/DECRC — supported by xterm, iTerm2, macOS Terminal)
    cursorSave    = "\033[s"
    cursorRestore = "\033[u"

    // moveCursor moves cursor to row r, column 1 (1-indexed).
    moveCursor = "\033[%d;1H"

    // eraseLineEntire erases the entire current line.
    eraseLineEntire = "\033[2K"
)
```

**Draw cycle (called by ticker, safe to call from goroutine writing to stdout):**

```go
// Source: internal/statusbar/bar.go (proposed)
func (b *Bar) draw() {
    cols, rows, err := term.GetSize(int(b.fd))
    if err != nil {
        return
    }

    // Detect terminal resize: re-issue DECSTBM if dimensions changed.
    b.mu.Lock()
    if cols != b.cols || rows != b.rows {
        b.cols = cols
        b.rows = rows
        if b.pos == Bottom {
            fmt.Fprintf(b.w, setScrollRegion, 1, rows-1)
        } else {
            fmt.Fprintf(b.w, setScrollRegion, 2, rows)
        }
    }
    viewerCount := b.viewerCount
    connState := b.connState
    b.mu.Unlock()

    barRow := rows   // Bottom (default)
    if b.pos == Top {
        barRow = 1
    }

    text := b.format(viewerCount, connState, cols)

    fmt.Fprint(b.w, cursorSave)
    fmt.Fprintf(b.w, moveCursor, barRow)
    fmt.Fprint(b.w, eraseLineEntire)
    fmt.Fprint(b.w, text)
    fmt.Fprint(b.w, cursorRestore)
}
```

[VERIFIED: vt100.net/docs/vt510-rm/DECSTBM.html — DECSTBM parameter format confirmed]
[VERIFIED: golang.org/x/term — GetSize signature confirmed]

### Pattern 2: TTY Suppression (SB-03)

`cmd_attach.go` already calls `term.IsTerminal(int(os.Stdin.Fd()))` before entering raw mode.
The status bar must be skipped when stdout is not a TTY. stdout (not stdin) is the correct
fd to check for TTY status — piping `agenthub attach | cat` makes stdout non-TTY while
stdin remains a TTY.

```go
// Source: cmd_attach.go (proposed extension)
var bar *statusbar.Bar
if term.IsTerminal(int(os.Stdout.Fd())) {
    bar = statusbar.New(os.Stdout, statusbar.Options{
        SessionName: session.Name,
        AgentType:   session.CLI,
        Hostname:    session.Hostname,
        CreatedAt:   session.CreatedAt, // for elapsed time
        Position:    parseStatusPosition(args), // --status-top or default Bottom
    })
    bar.Start()
    defer bar.Stop()
}
```

[VERIFIED: cmd_attach.go line 59 — `term.IsTerminal(int(os.Stdin.Fd()))` is the current pattern; stdout fd check is the correct addition]

### Pattern 3: MsgMeta Push for Viewer Count (SB-04)

The relay server already knows the subscriber count on every subscribe/unsubscribe event.
A new frame type `MsgMeta` (`0x20`) carries a small JSON payload to connected clients.

**Protocol extension:**

```go
// Source: internal/relay/protocol.go (proposed addition)
const (
    MsgMeta byte = 0x20 // Server-to-client: session metadata update (JSON payload)
)

// MetaPayload is the JSON structure for MsgMeta frames.
type MetaPayload struct {
    ViewerCount *int `json:"viewerCount,omitempty"`
}

// MakeMeta serialises a MetaPayload into a MsgMeta frame.
func MakeMeta(p MetaPayload) []byte {
    b, _ := json.Marshal(p)
    frame := make([]byte, 1+len(b))
    frame[0] = MsgMeta
    copy(frame[1:], b)
    return frame
}
```

**Server-side push (relay/server.go):**

```go
// After hub.Subscribe or hub.Unsubscribe, broadcast a MsgMeta frame.
// This must be called OUTSIDE hub.mu to avoid deadlock with broadcast.
func (s *Server) broadcastMeta(hub *relay.Hub) {
    count := hub.SubscriberCount()
    frame := relay.MakeMeta(relay.MetaPayload{ViewerCount: &count})
    hub.BroadcastMeta(frame) // new Hub method that broadcasts to all subs
}
```

**Client-side intercept (cmd_attach.go — wsOutputPump extension):**

```go
// Source: cmd_attach.go wsOutputPump (proposed extension)
case relay.MsgMeta:
    var meta relay.MetaPayload
    if err := json.Unmarshal(payload, &meta); err == nil && meta.ViewerCount != nil {
        if bar != nil {
            bar.SetViewerCount(*meta.ViewerCount)
        }
    }
```

[ASSUMED: Hub.BroadcastMeta is a new method added alongside existing broadcast; it locks hub.mu internally but does NOT block on slow clients — it uses the non-blocking send pattern from broadcast()]

### Pattern 4: Connection State Display (SB-05)

For remote sessions, the WebSocket connection state is observable from the attach goroutine.
The relay.Server can push a `MsgMeta` frame with a `connState` field when the connection
is established. The CLI client tracks latency using WebSocket ping/pong round-trip time.

**Simple approach:** measure the time between `wsOutputPump` receiving consecutive frames.
If no frame arrives for N seconds, set state to "reconnecting". The status bar formats this
as `[OK]`, `[!]`, or a latency string.

**Minimal implementation for Phase 75:** track time of last received frame in wsOutputPump;
if `time.Since(lastFrame) > 5s`, display `[reconnecting]` in the bar. Otherwise display
nothing (connected is the default state). Latency display is a stretch goal.

### Pattern 5: SIGWINCH Handling During Bar Active

When the terminal is resized, the status bar draw cycle must re-issue DECSTBM with new
dimensions to keep the scroll region correct. The existing `watchResize` goroutine in
`cmd_attach_unix.go` handles SIGWINCH to send `MsgResize2` to the relay. Phase 75 must
also call `bar.Resize()` (or the draw cycle itself calls `term.GetSize` on each tick, which
is simpler and correct for 1-second tick intervals).

**Recommendation:** On each tick, `bar.draw()` calls `term.GetSize` and detects dimension
changes by comparing against stored dimensions. This avoids a second SIGWINCH listener and
keeps the bar package self-contained.

### Pattern 6: Bar Format String

Status bar content for SB-01:

```
 session-name │ claude │ macbook-pro │ Ctrl-\ detach │ 2 viewers │ 0:05:23
```

With ANSI styling (dim/reverse video for the bar background):

```go
// Source: internal/statusbar/bar.go (proposed)
func (b *Bar) format(viewerCount int, connState string, cols int) string {
    elapsed := time.Since(b.opts.CreatedAt)
    h := int(elapsed.Hours())
    m := int(elapsed.Minutes()) % 60
    s := int(elapsed.Seconds()) % 60

    parts := []string{
        b.opts.SessionName,
        b.opts.AgentType,
        b.opts.Hostname,
        `Ctrl-\ to detach`,
    }
    if viewerCount > 1 {
        parts = append(parts, fmt.Sprintf("%d viewers", viewerCount))
    }
    if connState != "" {
        parts = append(parts, connState)
    }
    if h > 0 {
        parts = append(parts, fmt.Sprintf("%d:%02d:%02d", h, m, s))
    } else {
        parts = append(parts, fmt.Sprintf("%d:%02d", m, s))
    }

    text := " " + strings.Join(parts, " │ ") + " "

    // Truncate to terminal width to prevent wrapping.
    if len(text) > cols {
        text = text[:cols-1] + "…"
    }

    // Reverse video for the bar background (no external dep).
    return "\033[7m" + text + "\033[m"
}
```

### Anti-Patterns to Avoid

- **Writing status bar to stdout inside wsOutputPump:** The output pump goroutine writes
  raw PTY bytes directly. Interleaving status bar ANSI sequences in that same write path
  corrupts output. Use a separate goroutine (ticker) for bar draws and rely on the
  cursor save/restore pattern to be the only interleaver.

- **Not resetting DECSTBM on exit:** If `bar.Stop()` is not called (panic, signal, test),
  the terminal remains with a restricted scroll region. The `defer bar.Stop()` in
  `cmd_attach.go` is mandatory, and `bar.Stop()` must be safe to call multiple times via
  `sync.Once`.

- **Checking os.Stdin.Fd() for TTY (SB-03):** `agenthub attach | cat` keeps stdin as a
  TTY but redirects stdout. The status bar must check `os.Stdout.Fd()`, not `os.Stdin.Fd()`.

- **Using fmt.Println inside the bar draw:** `Println` appends `\n`, which causes the
  terminal to scroll even in the reserved bottom row. Use `fmt.Fprint` with explicit cursor
  positioning — no trailing newlines in bar draws.

- **One-byte ANSI escape form (`\033[s`):** `\033[s` (cursor save) and `\033[u` (restore)
  are DECSC/DECRC in the ANSI/SCO variant. xterm, iTerm2, and macOS Terminal all support
  them. The VT220 form is `\0337` / `\0338`. Both work in practice; the `\033[s/u` form is
  more readable and already used by other terminal tools.

- **Hub.BroadcastMeta blocking on slow clients:** The `MsgMeta` broadcast must use the same
  non-blocking send pattern as `hub.broadcast()`. A slow CLI client must never block the
  PTY drain loop.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal width detection | `os.Get_terminal_width()` custom | `term.GetSize(fd)` | Already in go.mod; handles platform differences |
| TTY detection | Custom `ioctl` call | `term.IsTerminal(fd)` | Already in go.mod; used in cmd_attach.go |
| ANSI color library | Charm Bracelet or similar | 3-4 constant escape strings | The bar needs only reverse video + reset; no color library needed |
| Viewer count polling | Periodic daemon API call | MsgMeta push frame | Push is zero-overhead and keeps count accurate on every subscribe/unsubscribe event |
| Independent tick goroutine per bar | — | Single `time.NewTicker` inside Bar.Start() | Bar owns its own goroutine lifecycle; cancels via context or done channel |

**Key insight:** The entire status bar is ~150 lines of Go with zero new dependencies.
The hardest part is the DECSTBM scroll region lifecycle (set on start, update on resize,
reset on stop) and the `sync.Once` guard on Stop().

## Runtime State Inventory

Step 2.5: SKIPPED — this is a greenfield feature addition, not a rename/refactor/migration phase.

## Common Pitfalls

### Pitfall 1: Garbled Output When Status Bar Draws During PTY Burst
**What goes wrong:** The ticker fires during a high-throughput PTY output burst. The
cursor-save/draw/cursor-restore sequence is interleaved with `wsOutputPump`'s `w.Write(payload)`
calls, producing garbled terminal output.
**Why it happens:** Two goroutines write to the same `os.Stdout` without coordination.
**How to avoid:** Serialize all writes to stdout through a single `io.Writer` wrapped with
a mutex. `cmd_attach.go` passes a `lockedWriter` (mutex-wrapped `os.Stdout`) to both
`wsOutputPump` and `Bar.New()`. All writes — PTY output and status bar draw — go through
the same lock.
**Warning signs:** Occasional garbled lines during high output sessions.

### Pitfall 2: Status Bar Not Cleaned Up on Panic/Signal
**What goes wrong:** A panic or unhandled signal exits `cmdAttach` without calling
`bar.Stop()`, leaving the terminal with a restricted scroll region. The user's next terminal
session appears to have a missing last row.
**How to avoid:** Use `defer bar.Stop()` immediately after `bar.Start()`. Implement
`bar.Stop()` with a `sync.Once` so it is safe to call multiple times. Ensure the signal
context cancellation in `cmd_attach.go` causes `attachSession` to return, which triggers
deferred cleanup.
**Warning signs:** After detaching, the terminal prompt only appears in rows 1-23 even
though the terminal is 24 rows.

### Pitfall 3: Scroll Region Not Updated After Terminal Resize
**What goes wrong:** User resizes terminal during an attach session. DECSTBM was set to
rows 1-23 on a 24-row terminal. After resize to 48 rows, PTY output scrolls within rows
1-23, leaving rows 24-47 blank and row 48 as the status bar.
**Why it happens:** DECSTBM is sticky — it stays at 1-23 until explicitly reset.
**How to avoid:** On each tick, call `term.GetSize` and compare against stored dimensions.
If dimensions changed, re-issue DECSTBM with new values. The draw cycle naturally handles
this without a separate SIGWINCH listener in the statusbar package.
**Warning signs:** Large blank region appears in terminal after resize.

### Pitfall 4: vim/editor Output Corrupted by Status Bar
**What goes wrong:** The user runs a full-screen editor (vim, nano) inside the AI session.
The editor uses its own cursor positioning and may draw to the last row. The status bar's
DECSTBM scroll region prevents the editor from using the last row.
**Why it happens:** DECSTBM is a terminal-global setting — all programs writing to the PTY
see the restricted scroll region.
**How to avoid:** This is an inherent tradeoff of DECSTBM. It is the same tradeoff tmux
makes. Since agenthub attach is a pass-through (the user typically runs an AI agent, not
vim), this is acceptable for Phase 75. Document as a known limitation.
**Warning signs:** Full-screen editors lose their last row inside an attached session.

### Pitfall 5: MsgMeta Frame Confused With Future Protocol Extensions
**What goes wrong:** Phase 76 (TUI) or later phases introduce another server-to-client
push frame type and collide with `0x20`.
**How to avoid:** Add a comment in `relay/protocol.go` listing the reserved byte range
for future use and document that `MsgMeta = 0x20` is the stable metadata push channel.
Define the payload as extensible JSON so future fields can be added without new frame types.

### Pitfall 6: `CreatedAt` Is a String in SessionInfo, Not time.Time
**What goes wrong:** `daemon.SessionInfo.CreatedAt` is serialized as an RFC3339 string.
Computing elapsed time requires parsing it. If the parse fails, elapsed time shows garbage.
**Why it happens:** `SessionInfo` is the JSON API type; the underlying `pty.Session.CreatedAt`
is `time.Time`, but it's lost in the API layer.
**How to avoid:** Parse `session.CreatedAt` with `time.Parse(time.RFC3339, session.CreatedAt)`
in `cmdAttach` and pass the resulting `time.Time` to `statusbar.New()`. Add an error check:
if parse fails, use `time.Now()` as a safe fallback.

## Code Examples

Verified patterns from official sources:

### DECSTBM Setup and Teardown

```go
// Source: vt100.net/docs/vt510-rm/DECSTBM.html (specification)
// internal/statusbar/bar.go

// Start sets up the scroll region and draws the initial status bar.
func (b *Bar) Start() {
    cols, rows, err := term.GetSize(int(b.fd))
    if err != nil {
        return // graceful no-op if terminal size is unavailable
    }
    b.mu.Lock()
    b.cols = cols
    b.rows = rows
    b.mu.Unlock()

    if b.pos == Bottom {
        // Reserve the last row for the status bar.
        fmt.Fprintf(b.w, "\033[1;%dr", rows-1)  // DECSTBM: scroll rows 1..(rows-1)
    } else {
        // Reserve the first row for the status bar (--status-top).
        fmt.Fprintf(b.w, "\033[2;%dr", rows)    // DECSTBM: scroll rows 2..rows
    }
    b.draw() // draw initial bar

    b.ctx, b.cancel = context.WithCancel(context.Background())
    b.wg.Add(1)
    go b.tickLoop()
}

// Stop tears down the scroll region and clears the bar line.
func (b *Bar) Stop() {
    b.stopOnce.Do(func() {
        if b.cancel != nil {
            b.cancel()
        }
        b.wg.Wait()

        // Reset scroll region to full terminal.
        fmt.Fprint(b.w, "\033[r")         // DECSTBM reset

        // Clear the bar line.
        b.mu.Lock()
        rows := b.rows
        pos := b.pos
        b.mu.Unlock()

        barRow := rows
        if pos == Top {
            barRow = 1
        }
        fmt.Fprintf(b.w, "\033[%d;1H", barRow)  // move to bar row
        fmt.Fprint(b.w, "\033[2K")               // clear entire line
        fmt.Fprint(b.w, "\033[u")                // restore cursor
    })
}
```

### MsgMeta Protocol Frame

```go
// Source: internal/relay/protocol.go (proposed addition)
const (
    MsgMeta byte = 0x20 // Server-to-client metadata push (JSON payload)
)

// MetaPayload is the extensible JSON payload for MsgMeta frames.
// All fields are pointers so omitempty works correctly for partial updates.
type MetaPayload struct {
    ViewerCount *int `json:"viewerCount,omitempty"`
}

// MakeMeta encodes a MetaPayload as a MsgMeta frame.
func MakeMeta(p MetaPayload) []byte {
    b, _ := json.Marshal(p) // MetaPayload is always serialisable
    frame := make([]byte, 1+len(b))
    frame[0] = MsgMeta
    copy(frame[1:], b)
    return frame
}
```

### Locked Writer for Concurrent stdout Access

```go
// Source: cmd_attach.go (proposed addition)
// lockedWriter serialises concurrent writes to an underlying io.Writer.
// Used to prevent interleaving of PTY output and status bar draw sequences.
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

### Bar Instantiation in cmdAttach

```go
// Source: cmd_attach.go (proposed extension)
// After session metadata is available, before entering raw mode:

stdout := &lockedWriter{w: os.Stdout}

var bar *statusbar.Bar
if term.IsTerminal(int(os.Stdout.Fd())) {
    createdAt, _ := time.Parse(time.RFC3339, session.CreatedAt)
    if createdAt.IsZero() {
        createdAt = time.Now()
    }
    bar = statusbar.New(stdout, statusbar.Options{
        SessionName: session.Name,
        AgentType:   session.CLI,
        Hostname:    session.Hostname,
        CreatedAt:   createdAt,
        Position:    statusTopFlag, // bool from --status-top arg
    })
    bar.Start()
    defer bar.Stop()
}

// Pass lockedWriter to attachSession instead of os.Stdout:
err = attachSession(ctx, conn, os.Stdin, stdout, detachKey)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| One-shot banner on attach | Persistent bottom status bar (DECSTBM) | Phase 75 | User always sees session context; no hunting for session name |
| No live viewer count in CLI | MsgMeta push frame | Phase 75 | CLI client sees viewer count update in real time |
| Session info only at attach | Elapsed time + viewer count on 1s tick | Phase 75 | Session age visible without running `agenthub list` |

**Deprecated/outdated for this phase:**
- `printAttachBanner()` — the one-shot banner can remain for contexts where the status bar
  is suppressed (non-TTY), but the banner is redundant when the bar is active. Recommendation:
  keep banner for non-TTY path, suppress when bar starts.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `MsgMeta = 0x20` byte does not conflict with any existing or planned relay frame type | Protocol Extension | Collision with future phase breaks protocol; mitigate by reviewing protocol.go before implementing |
| A2 | Hub.BroadcastMeta is a new method that uses the same non-blocking send as Hub.broadcast() | Architecture Patterns | If blocking, slow CLI clients stall the PTY drain loop — critical performance regression |
| A3 | DECSTBM scroll region works correctly in all target terminals (macOS Terminal, iTerm2, Linux xterm) | Status bar rendering | If a terminal emulator doesn't support DECSTBM, the bar draws on the last line but output may overwrite it; all mainstream terminals support DECSTBM |
| A4 | The `\033[s` / `\033[u` cursor save/restore (SCO variant) works in all target terminals | ANSI constants | If only VT220 `\0337`/`\0338` works, substitute those forms; behavior is equivalent |
| A5 | Serializing all stdout writes through a `lockedWriter` is sufficient to prevent interleaving | Pitfall 1 | If other code paths write directly to `os.Stdout`, a locked wrapper does not help; audit all write sites in cmd_attach.go |
| A6 | `session.CreatedAt` is always a valid RFC3339 string returned by the daemon API | Code examples | If empty or malformed, elapsed time falls back to `time.Now()` — acceptable |

## Open Questions

1. **Should the old one-shot banner be suppressed when the status bar is active?**
   - What we know: `printAttachBanner` writes to `os.Stderr`, not `os.Stdout`; the status bar draws to `os.Stdout`
   - What's unclear: Whether the banner and the bar are visually redundant in the TTY path
   - Recommendation: Suppress the banner in the TTY path (bar shows the same info persistently); keep the banner for non-TTY path (`| cat`)

2. **Should `Hub.BroadcastMeta` be a new public method or inline in relay/server.go?**
   - What we know: Hub is the authoritative broadcast mechanism; adding a second broadcast path that bypasses Hub risks inconsistency
   - Recommendation: Add `Hub.BroadcastMeta(frame []byte)` as a public method using the same non-blocking send pattern

3. **Does the remote attach path (cmdAttachRemote) also need the status bar?**
   - What we know: SB-05 requires showing connection state for remote sessions; the bar makes most sense there
   - Recommendation: Yes — `cmdAttachRemoteWithClient` should also instantiate the bar; connection state display is the additional feature for remote sessions

## Environment Availability

Step 2.6: SKIPPED — Phase 75 is a pure Go code change. No new external tools, services,
CLIs, runtimes, or databases are required beyond what is already installed (Go 1.26.2 toolchain,
existing go.mod dependencies including `golang.org/x/term`).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test ./...` |
| Quick run command | `go test ./internal/statusbar/... -count=1 -timeout 30s` |
| Full suite command | `go test ./... -count=1 -timeout 60s` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SB-01 | Status bar line contains session name, agent type, hostname, detach hint, elapsed time | unit | `go test ./internal/statusbar/... -run TestBar_FormatContainsRequiredFields` | ❌ Wave 0 |
| SB-02 | Status bar draws without corrupting PTY output (scroll region) | unit | `go test ./internal/statusbar/... -run TestBar_ScrollRegionNotCorrupted` | ❌ Wave 0 |
| SB-03 | Bar suppressed when stdout is not a TTY | unit | `go test ./... -run TestCmdAttach_NoBarWhenNotTTY` | ❌ Wave 0 |
| SB-04 | Bar updates viewer count from MsgMeta frame | unit | `go test ./internal/statusbar/... -run TestBar_SetViewerCountUpdates` | ❌ Wave 0 |
| SB-05 | Bar shows connection state for remote sessions | unit | `go test ./internal/statusbar/... -run TestBar_ConnStateDisplay` | ❌ Wave 0 |
| SB-06 | `--status-top` flag places bar at top | unit | `go test ./internal/statusbar/... -run TestBar_TopPosition` | ❌ Wave 0 |
| SB-07 | Detach/exit clears bar line and resets terminal state | unit | `go test ./internal/statusbar/... -run TestBar_StopClearsBarAndResetsScrollRegion` | ❌ Wave 0 |

**Note on testing ANSI output:** Tests capture the bytes written to an `io.Writer` (using
`bytes.Buffer`) and assert that specific escape sequences appear in the correct order.
Testing that the terminal *looks* correct is a manual/UAT concern; automated tests verify
the escape sequence protocol.

**Note on MsgMeta:** Add one test to `relay/protocol_test.go` for `MakeMeta`/`ParseFrame`
round-trip, and one integration test to `cmd_attach_test.go` for viewer count update.

### Sampling Rate

- **Per task commit:** `go test ./internal/statusbar/... ./internal/relay/... -count=1 -timeout 30s`
- **Per wave merge:** `go test ./... -count=1 -timeout 60s`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/statusbar/bar.go` — create package (new)
- [ ] `internal/statusbar/bar_test.go` — all SB-01 through SB-07 tests
- [ ] `internal/relay/protocol.go` — add `MsgMeta = 0x20`, `MetaPayload`, `MakeMeta()`
- [ ] `internal/relay/protocol_test.go` — add `TestMakeMeta_RoundTrip`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Existing auth unchanged |
| V3 Session Management | no | Session lifecycle unchanged |
| V4 Access Control | no | No new access control decisions |
| V5 Input Validation | yes | Session name/hostname displayed in bar — must not allow terminal injection |
| V6 Cryptography | no | No new crypto |

### Known Threat Patterns for Phase 75 Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Terminal injection via session name | Tampering | Strip ANSI escape sequences from session name and hostname before displaying in bar. Session name is user-controlled input stored by the daemon. |
| Viewer count spoofing via crafted MsgMeta | Tampering | MsgMeta originates from the trusted relay server (same process as daemon); only display viewer count, no privileged action taken |

**Terminal injection note:** The session name and hostname in the status bar are read from
`daemon.SessionInfo` which was user-provided at session creation. A session name containing
`\033[2J` (clear screen) or similar would corrupt the terminal. Strip all bytes < 0x20
(control characters including ESC) from session name and hostname before embedding in the
bar format string.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: /Users/ken/dev/agenthub/cmd_attach.go] — attach flow, banner, raw mode, wsOutputPump
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/protocol.go] — existing frame types, byte assignments
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/hub.go] — SubscriberCount, broadcast non-blocking pattern
- [VERIFIED: /Users/ken/dev/agenthub/internal/relay/server.go] — handleSession subscribe/unsubscribe lifecycle
- [VERIFIED: /Users/ken/dev/agenthub/internal/daemon/types.go] — SessionInfo fields including CreatedAt string, ViewerCount
- [VERIFIED: /Users/ken/dev/agenthub/go.mod] — golang.org/x/term v0.41.0 already in deps
- [CITED: vt100.net/docs/vt510-rm/DECSTBM.html] — DECSTBM parameter format, side effects (cursor to 1,1)
- [CITED: pkg.go.dev/golang.org/x/term] — GetSize, IsTerminal signatures confirmed

### Secondary (MEDIUM confidence)
- [CITED: vt100.net/docs/vt100-ug/chapter3.html] — VT100 cursor save/restore escape sequences
- WebSearch: charmbracelet/x/ansi confirms DECSTBM = SetTopBottomMargins

### Tertiary (LOW confidence)
- [ASSUMED: A3] — DECSTBM support in all target terminal emulators (mainstream terminals universally support it)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing go.mod dependencies; no new deps needed
- Architecture: HIGH — DECSTBM technique is standard and well-documented; all extension points identified from direct code reading
- Protocol extension: MEDIUM — MsgMeta byte assignment (0x20) is not yet verified against future phase plans; flag A1
- Pitfalls: HIGH — terminal interleaving and DECSTBM lifecycle pitfalls are well-known; confirmed from codebase analysis

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (stable stdlib and terminal spec; no fast-moving deps)

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SB-01 | CLI attach displays a persistent tmux-style bottom bar with session name, agent type, hostname, detach hint, and elapsed time | New `internal/statusbar` package with DECSTBM scroll region; Bar.format() assembles all required fields |
| SB-02 | Status bar refreshes on a timer without corrupting terminal output (DECSTBM scroll region) | DECSTBM restricts scrolling to rows 1..N-1; cursor save/restore isolates draw cycle from PTY output; lockedWriter serializes all stdout writes |
| SB-03 | Status bar is suppressed when stdout is not a TTY | `term.IsTerminal(int(os.Stdout.Fd()))` check before creating Bar; suppress banner and bar in non-TTY path |
| SB-04 | Status bar shows viewer count when multiple clients are connected | MsgMeta (0x20) push frame from relay.Server on each subscribe/unsubscribe; wsOutputPump intercepts and calls Bar.SetViewerCount() |
| SB-05 | Status bar shows connection state (connected/reconnecting/latency) for remote sessions | Track time of last received frame in wsOutputPump; if > 5s, set connState to "reconnecting"; Bar.SetConnectionState() |
| SB-06 | User can place status bar at top via `--status-top` flag (bottom is default) | cmd_attach.go parses `--status-top` flag; passes Position=Top to statusbar.New(); Bar uses DECSTBM rows 2..N for top position |
| SB-07 | Status bar cleans up on detach/exit — clears bar line and restores terminal state | Bar.Stop() with sync.Once: reset DECSTBM (`ESC[r`), move to bar row, clear line (`ESC[2K`); defer bar.Stop() in cmdAttach |
</phase_requirements>
