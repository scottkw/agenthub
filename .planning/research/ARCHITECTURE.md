# Architecture Research

**Domain:** Multi-client WebSocket relay, CLI status bar, TUI mode for Go/Wails desktop app
**Researched:** 2026-04-14
**Confidence:** HIGH

## Standard Architecture

### System Overview (Current v1.14)

```
┌────────────────────────────────────────────────────────────────────┐
│                        agenthub binary                              │
│                                                                     │
│  dispatch: no args → GUI (Wails)                                   │
│            subcommand → CLI                                         │
│            "daemon" → daemon service                                │
└───────────────────────────┬────────────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
  ┌─────────────┐   ┌──────────────┐   ┌─────────────────┐
  │  GUI (Wails)│   │  CLI cmds    │   │  Daemon service  │
  │  React +    │   │  cmd_attach  │   │  internal/daemon │
  │  xterm.js   │   │  cmd_cli     │   │  SessionEngine   │
  └──────┬──────┘   └──────┬───────┘   └────────┬─────────┘
         │                  │                    │
         └──────────────────┼────────────────────┘
                           │  DaemonClient (Unix socket)
                           ▼
               ┌────────────────────────┐
               │    daemon HTTP API      │
               │    (Unix socket)        │
               │  /sessions, /health,    │
               │  /relay-port, etc.      │
               └────────────┬───────────┘
                            │
               ┌────────────┴───────────┐
               │    SessionEngine        │
               │  registry + backend +   │
               │  HubManager + statuses  │
               └────────────┬───────────┘
                            │
         ┌──────────────────┼───────────────────┐
         │                  │                   │
         ▼                  ▼                   ▼
  ┌─────────────┐  ┌─────────────────┐  ┌──────────────┐
  │ pty.Session  │  │  relay.Hub      │  │ status.Watch │
  │ (PTY proc)   │  │  (broadcast)    │  │ (heuristics) │
  │ aymanbagabas │  │  scrollback buf │  │              │
  │  /go-pty     │  │  N subscribers  │  │              │
  └─────────────┘  └─────────────────┘  └──────────────┘
                            │
               ┌────────────┴────────────┐
               │    relay WebSocket       │
               │    TCP :random           │
               │    GET /sessions/{id}/ws │
               └─────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Key File |
|-----------|----------------|----------|
| `main.go` | Binary dispatch (GUI/CLI/daemon) | `main.go` |
| `internal/daemon/SessionEngine` | All session state: registry, hub manager, tab names, statuses | `engine.go` |
| `internal/daemon/API` | HTTP JSON over Unix socket, relay startup | `api.go` |
| `internal/daemon/DaemonClient` | Typed Go client for CLI and GUI over Unix socket | `client.go` |
| `internal/relay/Hub` | Fan-out PTY output to N subscribers; owns scrollback | `hub.go` |
| `internal/relay/HubManager` | Lifecycle of Hub instances keyed by session ID | `manager.go` |
| `internal/relay/Server` | HTTP handler: upgrades to WebSocket, manages subscribe/replay cycle | `server.go` |
| `internal/pty/Session` | Single PTY process, implements `io.ReadWriter` | `session.go` |
| `cmd_attach.go` | Raw terminal mode, I/O pump, SIGWINCH watcher | `cmd_attach.go` |
| Frontend (React/xterm.js) | Tabbed terminal, sidebar nav, settings, status bars | `frontend/src/` |

---

## New Architecture: v2.0 Target

### System Overview (Post v2.0)

```
┌────────────────────────────────────────────────────────────────────┐
│                        agenthub binary                              │
│                                                                     │
│  dispatch: no args → GUI (Wails)                                   │
│            "tui" → TUI mode  ← NEW                                 │
│            subcommand → CLI                                         │
│            "daemon" → daemon service                                │
└───────────────────────────┬────────────────────────────────────────┘
                            │
    ┌───────────────────────┼───────────────────────┐
    │           NEW         │                       │
    ▼                       ▼                       ▼
┌──────────────┐   ┌──────────────┐        ┌─────────────────┐
│  TUI mode    │   │  GUI (Wails) │        │  CLI cmds        │
│  bubbletea   │   │  React +     │        │  cmd_attach      │
│  DaemonClient│   │  xterm.js    │        │  (+ status bar)  │
│  WebSocket   │   │              │        │  ← MODIFIED      │
└──────┬───────┘   └──────┬───────┘        └──────┬───────────┘
       │                  │                        │
       └──────────────────┼────────────────────────┘
                         │  DaemonClient (Unix socket)
                         ▼
             ┌────────────────────────┐
             │    daemon HTTP API      │
             │    (Unix socket)        │
             │  + GET /sessions/{id}   │
             │    (client-count)       │  ← NEW optional field
             └────────────┬───────────┘
                          │
             ┌────────────┴────────────┐
             │    SessionEngine         │
             │  (unchanged internally)  │
             └────────────┬────────────┘
                          │
      ┌───────────────────┼───────────────────┐
      │                   │                   │
      ▼                   ▼                   ▼
┌─────────────┐  ┌──────────────────┐  ┌──────────────┐
│ pty.Session  │  │  relay.Hub        │  │ status.Watch │
│ (unchanged)  │  │  N subscribers    │  │ (unchanged)  │
│              │  │  (was: max 1)     │  │              │
│              │  │  ← MULTI-CLIENT  │  │              │
└─────────────┘  └──────────────────┘  └──────────────┘
                          │
             ┌────────────┴────────────┐
             │    relay WebSocket       │
             │    GET /sessions/{id}/ws │
             │    Multiple conns OK     │  ← ALREADY WORKS
             └─────────────────────────┘
```

---

## Feature 1: Multi-Client WebSocket Sessions (GitHub #13)

### Current State Analysis

The relay architecture **already supports multiple clients** at the protocol level. `relay.Hub` has a `subscribers map[*Subscriber]struct{}` that fans out to N concurrent WebSocket connections. The `Subscribe`/`Unsubscribe` pattern is safe for concurrent use. `ScrollbackSnapshot` replays scrollback to each new joiner independently.

The issue is **resize arbitration**: when multiple clients send `MsgResize2`, the PTY gets resized to whichever client last sent a resize event. Each client operates at a different terminal size, and there is no negotiation.

### What Actually Needs to Change

**relay/hub.go — No changes needed.** Fan-out already works for N clients.

**relay/server.go — No changes needed.** Subscribe-before-snapshot pattern already prevents gaps for any number of joiners.

**Resize arbitration — New policy needed.** Three viable policies:

1. **First-wins:** Only the first subscriber can resize. Additional clients are read-only for resize. Simplest. Breaks experience for latecomers who want control.
2. **Last-wins (current implicit behavior):** Any client can resize. Breaks other clients. Do not use.
3. **Controlling client:** The client that sent the most recent `MsgInput` (i.e., the one typing) owns resize. Others are observers. Good default — matches tmux "active pane" semantics.

**Recommended: Controlling-client resize policy.** Hub tracks `controllingClient *Subscriber` (or nil if no input yet). Resize frames from non-controlling clients are silently dropped. When a subscriber sends MsgInput, they become the controlling client.

```go
// In Hub — add to existing struct:
controllingClient *Subscriber
controlMu         sync.Mutex

// In Hub.WriteInputFrom — called from server.go read pump:
func (h *Hub) WriteInputFrom(sub *Subscriber, data []byte) error {
    h.controlMu.Lock()
    h.controllingClient = sub
    h.controlMu.Unlock()
    _, err := h.writer.Write(data)
    return err
}

// In Hub.ResizeFrom — new method:
func (h *Hub) ResizeFrom(sub *Subscriber, cols, rows int) error {
    h.controlMu.Lock()
    cc := h.controllingClient
    h.controlMu.Unlock()
    if cc != nil && cc != sub {
        return nil // not the controlling client, ignore
    }
    return h.resizeFn(cols, rows)
}
```

**daemon/api.go — Optional client-count exposure.** Add `ConnectedClients int` field to `SessionInfo` so GUI and TUI can show "3 clients attached". Hub needs a `SubscriberCount() int` method (already has `subscribers` map under `mu`).

### Data Flow Change: Multi-Client Resize

```
Client A (controlling — last typed)         Client B (observer)
    MsgInput → WriteInputFrom(subA)              MsgInput → WriteInputFrom(subB) → subB becomes controlling
    MsgResize2 → ResizeFrom(subA) → PTY          MsgResize2 → ResizeFrom(subB) → dropped (not controlling)
    ← MsgOutput broadcast ←                  ← MsgOutput broadcast ←
```

---

## Feature 2: tmux-Style CLI Status Bar (GitHub #8)

### Problem

`cmd_attach.go` puts the terminal in raw mode and directly proxies PTY output to stdout. There is no reserved screen region — PTY output scrolls the full terminal height. A persistent status bar requires reserving the bottom 1 line while scrolling content above it.

### Mechanism: ANSI Scroll Region + Status Bar

The standard mechanism used by tmux, vim, and htop: DECSTBM (`CSI r`) sets a scrolling region. By setting the scroll region to rows 1..N-1 (all but the last row), terminal output scrolls only within that region. The bottom row (row N) is outside the scroll region and persists.

```
Terminal: 80x24

Scroll region: rows 1–23 (CSI 1;23 r)
Bottom row 24: status bar — not scrolled

┌─────────────────────────────────────────────────┐
│ PTY output scrolls here (rows 1-23)              │
│                                                  │
│ $ claude "fix the bug"                           │
│ Analyzing...                                     │
│                                                  │
│                                                  │
├─────────────────────────────────────────────────┤
│ my-session │ claude │ hostname │ 0:03:42 │ Ctrl-\ │  ← row 24, persists
└─────────────────────────────────────────────────┘
```

### Implementation Plan

**New package: `internal/statusbar`**

Responsibilities:
- Draw initial status bar at terminal bottom (save cursor, move to row N, write content, restore cursor)
- Start `time.Ticker` to update elapsed time in place
- Set DECSTBM scroll region on attach; restore full scroll region on detach
- Handle SIGWINCH: recalculate row N, redraw bar at new position
- Restore terminal state on all exit paths (panic-safe defer)

```go
// internal/statusbar/bar.go
type Bar struct {
    w       io.Writer   // typically os.Stderr (out-of-band, does not go to PTY)
    session SessionMeta
    ticker  *time.Ticker
    start   time.Time
    rows    int
    cols    int
    done    chan struct{}
}

type SessionMeta struct {
    Name     string
    CLI      string
    Hostname string
}

func New(w io.Writer, meta SessionMeta, cols, rows int) *Bar
func (b *Bar) Start()                // sets scroll region, draws bar, starts ticker
func (b *Bar) Stop()                 // restores scroll region, clears bar
func (b *Bar) Resize(cols, rows int) // called by SIGWINCH handler
```

**Modified: `cmd_attach.go`**

Current flow:
```
raw mode → printAttachBanner → attachSession → printDetachMessage → restore raw
```

New flow:
```
raw mode → statusbar.New(meta, cols, rows) → bar.Start() →
attachSession (proxies PTY output, bar.Resize on SIGWINCH) →
bar.Stop() → printDetachMessage → restore raw
```

Key constraint: the PTY output pump (`wsOutputPump`) writes to `os.Stdout`. The status bar writes to `os.Stderr`. Both file descriptors point to the same terminal device (`/dev/tty`). Writing the scroll region escape to either works. Using stderr for control sequences keeps them out of any stdout capture and avoids interleaving with PTY content.

**Modified: `cmd_attach_unix.go` (watchResize)**

SIGWINCH handler must call `bar.Resize(newCols, newRows)` in addition to sending the resize frame to the WebSocket relay.

**Terminal cleanup is critical.** The defer chain must ensure `bar.Stop()` runs on every exit path including panics. The scroll region must be reset to full terminal before `term.Restore`.

**DECSTBM sequence details (HIGH confidence — standard ANSI):**
- Set scroll region: `ESC [ top ; bottom r` where top=1, bottom=rows-1
- Reset scroll region: `ESC [ r` (no parameters = full terminal)
- Move cursor to row N, col 1: `ESC [ N ; 1 H`
- Save/restore cursor: `ESC 7` / `ESC 8` (DEC — more widely supported than CSI s/u)
- Clear to end of line: `ESC [ K`

### Status Bar Content

```
 session-name │ claude │ hostname │ elapsed │ Ctrl-\ to detach
```

Width-aware: truncate session name if terminal is narrow. Elapsed time ticks every second. Use `\r` (carriage return without newline) when redrawing in place at the bottom row.

---

## Feature 3: TUI Mode (GitHub #7)

### Binary Dispatch

Current dispatch in `main.go`:
```go
if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-") {
    runGUI()
    return
}
runCLI(os.Args[1:])
```

New dispatch:
```go
if len(os.Args) == 1 || strings.HasPrefix(os.Args[1], "-") {
    runGUI()
    return
}
if os.Args[1] == "tui" {
    runTUI(os.Args[2:])  // new
    return
}
runCLI(os.Args[1:])
```

This is a **non-breaking insertion** — "tui" was not a valid CLI subcommand before. No existing behavior changes.

### TUI Architecture

The TUI is a standalone Bubble Tea application that is a DaemonClient consumer, identical in role to the GUI. It communicates with the daemon over the Unix socket and the relay over WebSocket — the same paths the GUI and CLI use.

**Not** a wrapper around `cmd_attach.go`. TUI mode is a full interactive UI, not a single-session attach.

**Framework:** Bubbletea v2 + Lipgloss v2 + Bubbles (MEDIUM confidence — v2 is current as of 2026). The Charm stack is the dominant Go TUI ecosystem. No viable alternative for a feature-rich, layout-driven terminal UI in Go.

### TUI Component Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ [AgentHub TUI]                                    [status line] │
├──────────────┬──────────────────────────────────────────────────┤
│  Sidebar     │                                                  │
│  ─────────   │   Active pane: session terminal OR panel view   │
│  > Sessions  │                                                  │
│    s1: claude│   ┌──────────────────────────────────────────┐  │
│    s2: gemini│   │  raw PTY passthrough (NOT xterm.js)      │  │
│  > New       │   │  output from relay WebSocket             │  │
│  > Remote    │   │  input forwarded as MsgInput             │  │
│  > Settings  │   └──────────────────────────────────────────┘  │
│              │                                                  │
├──────────────┴──────────────────────────────────────────────────┤
│  session-name │ agent │ host │ elapsed │ keybindings            │
└─────────────────────────────────────────────────────────────────┘
```

### TUI Terminal Viewport Strategy

**Core problem:** xterm.js is a browser DOM component. The TUI runs in a real terminal. A different approach is needed for rendering PTY output.

**Two options:**

**Option A: Raw PTY passthrough within a constrained region.**
Subscribe to relay WebSocket, receive `MsgOutput` frames, write raw bytes directly to stdout within a constrained region (DECSTBM scroll region, same technique as the status bar). The TUI chrome (sidebar, status bar, header) occupies reserved rows. PTY output fills the remaining rows. When the active session pane is focused, Bubbletea suspends its renderer and PTY takes over the constrained region.

Pros: Perfect fidelity — all ANSI sequences from the AI agent render correctly. Same behavior as `cmd_attach`.
Cons: Requires careful coordination — Bubbletea's renderer and raw PTY output fight for stdout. Needs suspend/resume lifecycle.

**Option B: Soft-render PTY output via Bubbletea viewport.**
Receive `MsgOutput` frames, strip ANSI, display as plain text in `bubbles/viewport`.

Pros: Clean Bubbletea integration.
Cons: Loses all terminal formatting. AI agent UIs (claude-code, opencode) are ANSI-heavy. Unacceptable for this product.

**Recommended: Option A (raw passthrough with region isolation).**

The implementation pattern:
1. Bubbletea runs in alternate screen (`tea.WithAltScreen()`).
2. When a session pane is active, Bubbletea calls `tea.Suspend()` (bubbletea v2 API), sets DECSTBM scroll region to constrain output to the pane region, subscribes to relay, and starts raw PTY passthrough.
3. The status bar and sidebar chrome are rendered by Bubbletea before suspension via cursor positioning. The statusbar package handles the bottom row.
4. On Ctrl-W (switch pane) or Ctrl-\ (detach from session): PTY passthrough stops, scroll region resets, Bubbletea calls `tea.Resume()`.

This matches how `lazygit` and `k9s` handle embedded shell sessions — suspend the framework renderer while a subprocess/raw stream has terminal focus.

The `internal/statusbar` package (built for Feature 2) is directly reusable for the TUI's pane chrome.

### TUI Panels

Mirror the GUI sidebar panels:

| Panel | Content | Implementation |
|-------|---------|----------------|
| Sessions | List of running sessions with status dots | `bubbles/list` |
| New Session | Agent picker, folder path, args | `bubbles/textinput` + custom |
| Settings | Web server URL, Tailscale status, theme name (read-only display) | Static lipgloss view |
| Remote | Tailnet peers and their sessions | `bubbles/list` |

### TUI DaemonClient Usage

TUI calls `daemon.NewDaemonClient(socketPath)` identically to CLI. It calls:
- `ListSessions()` to populate sidebar
- `CreateSession()` for new session flow
- `KillSession()` for session termination
- `GetRelayPort()` to find where to connect WebSocket

No new daemon API endpoints are required for TUI functionality.

---

## Recommended Project Structure Changes

```
/
├── main.go                    # add tui dispatch branch before runCLI
├── cmd_attach.go              # integrate statusbar.Bar (Start/Stop)
├── cmd_attach_unix.go         # pass Bar.Resize to SIGWINCH handler
├── cmd_tui.go                 # NEW: runTUI() entry point
├── internal/
│   ├── relay/
│   │   ├── hub.go             # add WriteInputFrom, ResizeFrom, SubscriberCount
│   │   └── server.go          # use WriteInputFrom / ResizeFrom
│   ├── statusbar/             # NEW package
│   │   ├── bar.go             # Bar struct, Start, Stop, Resize
│   │   └── bar_test.go
│   ├── tui/                   # NEW package
│   │   ├── app.go             # tea.Program setup, root model
│   │   ├── session_pane.go    # suspend/resume + raw passthrough
│   │   ├── sessions_list.go   # sidebar sessions panel
│   │   ├── new_session.go     # new session flow
│   │   ├── settings_view.go   # settings read-only view
│   │   └── remote_view.go     # tailnet remote sessions
│   └── daemon/
│       └── engine.go          # add SubscriberCount to SessionInfo (optional)
```

---

## Data Flow Changes

### Multi-Client Attach

```
Client A (first, controlling)          Client B (joins later)
    WS connect                              WS connect
    Subscribe(subA)                         Subscribe(subB)
    ScrollbackSnapshot → send               ScrollbackSnapshot → send
    ← live MsgOutput broadcasts →       ← live MsgOutput broadcasts →
    MsgInput → WriteInputFrom(subA)         MsgInput → WriteInputFrom(subB) → subB becomes controlling
    MsgResize2 → ResizeFrom(subA) → PTY     MsgResize2 → ResizeFrom(subB) → dropped (not controlling)
```

### CLI Attach with Status Bar

```
cmdAttach()
  │
  ├── term.MakeRaw()
  ├── statusbar.New(meta, cols, rows)
  ├── bar.Start()
  │     ├── ESC[1;N-1 r    (set scroll region, reserve bottom row)
  │     ├── ESC[N;1H        (move to bottom row)
  │     └── write status content + start ticker goroutine
  │
  ├── go watchResize(ctx, conn, bar)   // SIGWINCH → bar.Resize + send resize frame
  │
  ├── attachSession(ctx, conn, stdin, stdout, detachKey)
  │     ├── stdinPump → relay MsgInput frames
  │     └── wsOutputPump → write MsgOutput bytes to stdout (within scroll region)
  │
  ├── bar.Stop()
  │     ├── cancel ticker
  │     ├── ESC[r            (reset scroll region to full terminal)
  │     └── ESC[N;1H ESC[K   (clear status bar row)
  │
  └── term.Restore()
```

### TUI Mode Data Flow

```
runTUI()
  │
  ├── daemon.EnsureDaemon(socketPath)
  ├── daemon.NewDaemonClient(socketPath)
  ├── tea.NewProgram(model, tea.WithAltScreen())
  └── p.Run()
       │
       ├── Init: fetch sessions list from daemon
       ├── Update loop:
       │     ├── KeyMsg → navigate sidebar / switch panes / pass input to relay
       │     ├── SessionListMsg → refresh from daemon poll
       │     └── RelayOutputMsg → (session pane active) write raw to constrained region
       └── View:
             ├── sidebar (lipgloss columns)
             ├── status bar (bottom row via statusbar package)
             └── session pane: tea.Suspend() → raw passthrough → tea.Resume()
```

---

## Architectural Patterns

### Pattern 1: Subscribe-Before-Snapshot (existing, unchanged)

**What:** New WebSocket clients subscribe to the Hub before taking a scrollback snapshot. Frames in-flight between snapshot and first live message arrive via the buffered `Msgs` channel — no gap.
**When to use:** Every new connection in `relay/server.go`. Already correct for N clients.
**Trade-offs:** Adds ~256 buffered frames per subscriber in memory. Acceptable.

### Pattern 2: Controlling-Client Resize Arbitration (new)

**What:** Track which subscriber last sent `MsgInput`. Only that subscriber's resize frames reach the PTY.
**When to use:** Whenever N > 1 clients are connected to a session.
**Trade-offs:** First client to type wins. Latecomers who want control must type first. Simple, no negotiation protocol needed.

### Pattern 3: DECSTBM Scroll Region for Status Bar (new)

**What:** Set terminal scroll region to rows 1..N-1, reserve row N for persistent status bar content that does not scroll.
**When to use:** CLI attach (default on) and TUI mode chrome.
**Trade-offs:** Requires tracking terminal rows and handling SIGWINCH. Must be restored on all exit paths or the terminal is left in a broken scroll region state.

### Pattern 4: Bubbletea Suspend/Resume for Raw PTY (new)

**What:** Call `tea.Suspend()` to temporarily yield terminal control to raw PTY passthrough, then `tea.Resume()` to return to Bubbletea rendering.
**When to use:** TUI session pane when user focuses a terminal session.
**Trade-offs:** Requires bubbletea v2 (`tea.Suspend`/`tea.Resume` API). Adds a mode-switching state machine to the TUI model. Each suspension must save/restore scroll region state.

---

## Anti-Patterns

### Anti-Pattern 1: Last-Wins Resize with Multiple Clients

**What people do:** Let any connected client send `MsgResize2` and resize the PTY unconditionally.
**Why it's wrong:** A client at 40x12 resizes the PTY, breaking the session for the 80x24 client who is actively typing.
**Do this instead:** Controlling-client resize policy — only the subscriber that last sent MsgInput can resize.

### Anti-Pattern 2: Writing Status Bar Escapes to Stdout

**What people do:** Interleave status bar ANSI escape sequences with PTY output on the same stdout stream.
**Why it's wrong:** PTY output is a raw byte stream. Status bar sequences can be split mid-ANSI-sequence by concurrent PTY output arriving from the WebSocket pump.
**Do this instead:** Write all status bar control sequences to `/dev/tty` directly (or `os.Stderr` which references the same terminal device). Status bar writes are independent of the stdout PTY stream.

### Anti-Pattern 3: Soft-Rendering PTY Output in Bubbletea Viewport

**What people do:** Strip ANSI from PTY output and display as plain text in `bubbles/viewport`.
**Why it's wrong:** AI coding CLIs (claude-code, opencode, gemini) use rich ANSI UIs — progress indicators, colored diff, interactive prompts. Stripping ANSI makes the TUI unusable for the product's core value.
**Do this instead:** Raw passthrough within a constrained scroll region (Pattern 4).

### Anti-Pattern 4: Forgetting Scroll Region Cleanup

**What people do:** Set DECSTBM, start status bar, let the program panic or receive SIGKILL — scroll region is never reset.
**Why it's wrong:** User's terminal is left with a truncated scroll region. All subsequent shell output is confined to rows 1..N-1 until the user manually resets with `reset` or `tput rmcup`.
**Do this instead:** `defer bar.Stop()` immediately after `bar.Start()`. `bar.Stop()` must be deferred before `term.Restore` so it runs while the terminal is still in raw mode.

---

## Build Order Rationale

The three features have a natural dependency ordering:

```
Phase 1: Multi-client resize arbitration
  Touches: relay/hub.go, relay/server.go
  No deps on other new features.
  Can ship independently — improves existing web client behavior.

Phase 2: CLI status bar
  Touches: internal/statusbar (new), cmd_attach.go, cmd_attach_unix.go
  No deps on Phase 1.
  Can run in parallel with Phase 1.

Phase 3: TUI mode
  Touches: internal/tui (new), cmd_tui.go, main.go
  Depends on Phase 2: reuses internal/statusbar for pane chrome.
  Benefits from Phase 1: multi-client behavior is correct when TUI + browser both connect.
  New go.mod deps: bubbletea v2, lipgloss v2, bubbles.
  Largest scope — build last.
```

Phases 1 and 2 are independent and can be built in parallel. Phase 3 depends on both.

---

## Integration Points

### Existing Components — No Change Required

| Component | Why Unchanged |
|-----------|--------------|
| `relay.Hub` broadcast | Already fans out to N subscribers correctly |
| `relay.Scrollback` | Already replays independently per subscriber |
| `relay/server.go` subscribe-before-snapshot | Already race-safe for N clients |
| `daemon/API` Unix socket | TUI uses same DaemonClient as CLI |
| `internal/pty/Session` | PTY is unaware of client count |
| `internal/tailnet` | TUI remote view calls same `ListTailnetPeers` |
| `internal/webserver` | Unchanged — TUI connects as relay client, not web server |

### Existing Components — Modified

| Component | Change | Risk |
|-----------|--------|------|
| `relay/hub.go` | Add `WriteInputFrom`, `ResizeFrom`, `SubscriberCount` | LOW — additive; existing `WriteInput` delegates |
| `relay/server.go` | Use `WriteInputFrom`/`ResizeFrom` in read pump | LOW — same logic, new signature |
| `cmd_attach.go` | Wrap `attachSession` with `statusbar.Bar` | LOW — wraps existing flow, clean defer chain |
| `cmd_attach_unix.go` | Pass Bar to SIGWINCH handler | LOW — one additional call |
| `main.go` | Add `tui` dispatch branch before `runCLI` | LOW — non-breaking insertion |

### New Components

| Component | Role |
|-----------|------|
| `internal/statusbar` | ANSI scroll region, status bar draw, elapsed ticker, cleanup |
| `internal/tui` | Bubbletea app: root model, sidebar views, raw passthrough pane |
| `cmd_tui.go` | `runTUI()` entry point, daemon ensure, tea.Program setup |

---

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1-3 clients/session | No changes needed beyond resize arbitration |
| 10+ clients/session | Hub's 256-frame per-subscriber buffer may need tuning; consider reducing to 64 per subscriber to limit memory |
| 100+ clients/session | Not a target use case for local desktop tool; if needed, add subscriber eviction policy |

---

## Sources

- Existing codebase: `internal/relay/hub.go`, `internal/relay/server.go`, `cmd_attach.go`, `main.go` — authoritative for current state (HIGH confidence, verified directly)
- [charmbracelet/bubbletea GitHub](https://github.com/charmbracelet/bubbletea) — Framework architecture, Suspend/Resume API (HIGH confidence)
- [charmbracelet/lipgloss GitHub](https://github.com/charmbracelet/lipgloss) — Layout styling (HIGH confidence)
- [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) — viewport, list, textinput components (HIGH confidence)
- [Charm v2 release announcement](https://lobste.rs/s/1to8sq/charm_v2_major_releases_for_bubble_tea_lip) — v2 release confirmation (MEDIUM confidence)
- [ANSI Escape Codes reference](https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797) — DECSTBM, cursor sequences (HIGH confidence — standard ANSI/VT100)
- [BubbleTea TUI patterns 2026](https://dasroot.net/posts/2026/03/build-tui-apps-go-bubbletea/) — Current ecosystem patterns (MEDIUM confidence)

---
*Architecture research for: AgentHub v2.0 multi-client, CLI status bar, TUI mode*
*Researched: 2026-04-14*
