# Phase 3: Wails Desktop UI - Research

**Researched:** 2026-03-18
**Domain:** Wails v2 (Go desktop), xterm.js v5 (terminal rendering), React (UI), fyne.io/systray (system tray)
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TERM-01 | User can open multiple terminal tabs, each running an independent session | React tab state management + per-tab xterm.js Terminal instances; each mounts its own WebSocket connection to relay server |
| TERM-02 | User can name/rename terminal tabs for identification | Tab label stored in React state; Wails bound method persists name in-process (no disk needed for Phase 3) |
| TERM-03 | Terminal renders full ANSI color and Unicode/emoji output correctly | xterm.js @xterm/xterm v5 natively supports ANSI 256-color; @xterm/addon-unicode11 adds correct wide-char widths |
| TERM-04 | Terminal provides 10K+ line scrollback buffer | xterm.js `scrollback` option accepts any integer; 10000 is well-tested and low-overhead |
| TERM-05 | User can copy/paste text from terminal sessions | xterm.js built-in selection; @xterm/addon-clipboard for write-to-clipboard; paste via terminal.paste(text) on DataTransfer |
| CLI-03 | User can configure custom CLI paths in app settings | Settings panel with input fields; Go-bound `UpdateCLIPath(name, path)` method; validated at session launch time |
| SESS-02 | App runs in system tray when window is closed, keeping sessions alive | `HideWindowOnClose: true` in Wails options + `OnBeforeClose` returns true to prevent quit; fyne.io/systray for the tray icon via `RunWithExternalLoop` |
</phase_requirements>

---

## Summary

Phase 3 wires together the Go PTY/relay backend (Phases 1–2) with a Wails v2 desktop shell that renders terminal sessions in tabbed xterm.js panels. The architecture is straightforward: Wails embeds a React frontend via Go's `embed.FS`; the React app opens a WebSocket to the relay server that already exists inside the same process; xterm.js renders the stream with full ANSI support.

The largest surprise in this phase is **system tray**: Wails v2 (latest stable: v2.11.0) does NOT have built-in system tray support. System tray is a Wails v3 feature. The correct approach for v2 is to use `HideWindowOnClose: true` plus the `OnBeforeClose` callback to prevent the app from quitting, and separately run `fyne.io/systray` via `RunWithExternalLoop` to provide the tray icon and "Show Window / Quit" menu items.

The second nuance is the **WebSocket integration pattern**: xterm.js's `@xterm/addon-attach` speaks a raw text+binary WebSocket protocol, but the Phase 2 relay uses a custom framing protocol (1-byte type prefix). The frontend must implement its own thin WebSocket client that parses `MsgOutput` frames and sends `MsgInput`/`MsgResize` frames rather than using the attach addon directly.

**Primary recommendation:** Use Wails v2.11, React 18, @xterm/xterm v5.x, custom WebSocket client matching the Phase 2 framing protocol, and fyne.io/systray via RunWithExternalLoop for tray support.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/wailsapp/wails/v2 | v2.11.0 | Desktop shell: embeds React frontend, binds Go methods, manages window lifecycle | Stable since 2022; v3 is alpha — do not use v3 |
| @xterm/xterm | 5.x | Terminal emulator in the browser; full ANSI 256-color, Unicode, scrollback | Industry standard (used by VS Code, JetBrains); official scoped package (old `xterm` package deprecated) |
| @xterm/addon-fit | 5.x | Fits terminal to its container element on resize | Required for responsive terminal sizing; official addon |
| @xterm/addon-unicode11 | 5.x | Correct wide-char (CJK, emoji) widths for Unicode 11 | Needed for emoji and box-drawing accuracy in Claude Code output |
| @xterm/addon-webgl | 5.x | WebGL2 GPU-accelerated rendering | Required for 10K+ scrollback performance without jank |
| fyne.io/systray | latest | Cross-platform system tray icon + menu | Only stable Go systray library that supports macOS+Linux+Windows; used via RunWithExternalLoop to avoid main-loop conflict |
| react | 18.x | UI framework | Already chosen in project; Wails react template generates this |
| react-dom | 18.x | DOM rendering | Paired with React |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @xterm/addon-clipboard | 5.x | Copy selection to OS clipboard | Use for explicit copy button; standard selection-to-clipboard also works via browser's clipboard API |
| @xterm/addon-search | 5.x | Search within terminal buffer | Optional enhancement; not required for Phase 3 success criteria |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| fyne.io/systray | getlantern/systray | fyne-io fork removes GTK dependency; prefer it |
| fyne.io/systray | Wails v3 system tray | v3 is alpha; not production ready; do not migrate |
| @xterm/addon-webgl | Canvas renderer (default) | Canvas is fine for small buffers; webgl required for 10K+ scrollback without jank |
| Custom WS client | @xterm/addon-attach | addon-attach expects raw text/binary protocol; Phase 2 uses framed binary protocol with type bytes; custom client required |

**Installation:**
```bash
# Go side
go get github.com/wailsapp/wails/v2@latest
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go get fyne.io/systray

# Frontend (run inside frontend/ directory, using pnpm)
pnpm add @xterm/xterm @xterm/addon-fit @xterm/addon-unicode11 @xterm/addon-webgl @xterm/addon-clipboard
```

---

## Architecture Patterns

### Recommended Project Structure

The existing Go module at `github.com/agenthub/agenthub` already has `go.mod`. Wails is added to this module — not scaffolded into a new project. The `main.go` in `cmd/agenthub/` becomes the Wails entrypoint.

```
agenthub/                       # existing repo root
├── cmd/
│   └── agenthub/
│       ├── main.go             # wails.Run(options) — replaces smoke-test binary
│       └── app.go              # App struct — bound methods for frontend
├── internal/
│   ├── pty/                    # Phase 1 (unchanged)
│   └── relay/                  # Phase 2 (unchanged, gains Resize call)
├── frontend/                   # Wails-managed React app (NEW)
│   ├── src/
│   │   ├── App.tsx             # Root: tab bar + terminal panels
│   │   ├── components/
│   │   │   ├── TabBar.tsx      # Tab list, add/rename/close buttons
│   │   │   └── TerminalPanel.tsx  # Single xterm.js instance + WS connection
│   │   └── lib/
│   │       └── relayClient.ts  # WebSocket framing: send/receive Phase 2 frames
│   ├── package.json
│   └── vite.config.ts
├── build/                      # Wails build artifacts (NEW)
│   ├── appicon.png
│   ├── darwin/
│   ├── linux/
│   └── windows/
├── wails.json                  # Wails project config (NEW)
├── go.mod                      # gains wailsapp/wails/v2 and fyne.io/systray
└── go.sum
```

### Pattern 1: Wails App Bootstrap

**What:** `wails.Run` in `main.go` wires the embedded frontend to Go-bound methods. The `App` struct in `app.go` holds the `context.Context` from `OnStartup` and references to `SessionRegistry`, `NativePTYBackend`, `HubManager`, and the relay `Server`.

**When to use:** Every Wails app follows this pattern — main.go is thin, app.go does the work.

```go
// Source: wails.io docs + pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options
//go:embed all:frontend/dist
var assets embed.FS

func main() {
    app := NewApp()
    err := wails.Run(&options.App{
        Title:             "AgentHub",
        Width:             1200,
        Height:            800,
        MinWidth:          800,
        MinHeight:         600,
        AssetServer:       &assetserver.Options{Assets: assets},
        BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
        HideWindowOnClose: true,           // hide, not quit, on close button
        OnStartup:         app.startup,
        OnShutdown:        app.shutdown,
        OnBeforeClose:     app.beforeClose,
        Bind:              []interface{}{app},
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Pattern 2: App struct — bound methods

**What:** Public methods on `App` are auto-generated as async TypeScript functions in `frontend/wailsjs/go/main/App.js`. The frontend calls them as `await App.CreateSession(...)`.

**When to use:** Any data flow that is request/response (not streaming). Terminal output is NOT sent via bound methods — it goes over WebSocket. Bound methods handle: list sessions, create session, rename session, kill session, get detected CLIs, update settings.

```go
// Source: wails.io Application Development guide
type App struct {
    ctx       context.Context
    registry  *pty.SessionRegistry
    backend   pty.SessionBackend
    manager   *relay.HubManager
    server    *relay.Server
    httpSrv   *http.Server
}

// All exported methods become TypeScript async functions.
func (a *App) ListSessions() []SessionInfo { ... }
func (a *App) CreateSession(cli string, name string) (string, error) { ... }
func (a *App) RenameSession(id string, name string) error { ... }
func (a *App) KillSession(id string) error { ... }
func (a *App) DetectCLIs() []pty.DetectedCLI { ... }
func (a *App) UpdateCLIPath(name, path string) error { ... }
```

### Pattern 3: System Tray via fyne.io/systray

**What:** Because Wails v2 has NO built-in system tray, use `fyne.io/systray` with `RunWithExternalLoop`. This is started in a goroutine during `OnStartup`. The tray menu provides "Show AgentHub" and "Quit" items. Window close is intercepted by `OnBeforeClose` to hide instead of quit.

**When to use:** Required for SESS-02. This pattern is the only stable cross-platform approach for Wails v2.

```go
// Source: pkg.go.dev/fyne.io/systray
func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    // Start relay HTTP server on loopback
    a.httpSrv = &http.Server{Addr: "127.0.0.1:0", Handler: a.server}
    go a.httpSrv.ListenAndServe()

    // System tray via fyne.io/systray RunWithExternalLoop
    start, end := systray.RunWithExternalLoop(a.onTrayReady, a.onTrayExit)
    start()
    a.trayEnd = end
}

func (a *App) onTrayReady() {
    systray.SetIcon(trayIconBytes) // embed PNG bytes
    systray.SetTooltip("AgentHub")
    mShow := systray.AddMenuItem("Show AgentHub", "")
    mQuit := systray.AddMenuItem("Quit", "Quit AgentHub")
    go func() {
        for {
            select {
            case <-mShow.ClickedCh:
                runtime.WindowShow(a.ctx)
            case <-mQuit.ClickedCh:
                systray.Quit()
                runtime.Quit(a.ctx)
            }
        }
    }()
}

// OnBeforeClose: return true to prevent quit; hide window instead.
func (a *App) beforeClose(ctx context.Context) bool {
    runtime.WindowHide(ctx)
    return true  // prevents Wails from quitting the app
}
```

### Pattern 4: xterm.js Terminal Component

**What:** Each tab renders a `TerminalPanel` that creates one `Terminal` instance, opens it into a `div` ref, connects a WebSocket to the relay, and cleans up on unmount. The `FitAddon` is used for resize; `onResize` sends a `MsgResize` frame over the WebSocket.

**When to use:** One `TerminalPanel` per tab. Hidden tabs use CSS `display: none` — the `Terminal` instance is NOT unmounted, preserving its buffer state. Only the visible terminal calls `fitAddon.fit()`.

```typescript
// Source: xtermjs.org docs + react-xtermjs pattern
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'

function TerminalPanel({ sessionId, isActive }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<Terminal | null>(null)

  useEffect(() => {
    const term = new Terminal({
      scrollback: 10000,
      allowProposedApi: true,  // required for unicode11 addon
      fontFamily: 'monospace',
      fontSize: 14,
      cursorBlink: true,
    })
    const fitAddon = new FitAddon()
    const unicode11 = new Unicode11Addon()
    const webgl = new WebglAddon()

    term.loadAddon(fitAddon)
    term.loadAddon(unicode11)
    term.loadAddon(webgl)
    term.unicode.activeVersion = '11'
    term.open(containerRef.current!)
    fitAddon.fit()
    termRef.current = term

    const ws = new WebSocket(`ws://127.0.0.1:${relayPort}/sessions/${sessionId}/ws`)
    ws.binaryType = 'arraybuffer'

    ws.onmessage = (evt) => {
      const data = new Uint8Array(evt.data as ArrayBuffer)
      const msgType = data[0]
      if (msgType === 0x01) { // MsgOutput
        term.write(data.slice(1))
      }
    }

    term.onData((input) => {
      // MsgInput frame: [0x10, ...utf8 bytes]
      const encoded = new TextEncoder().encode(input)
      const frame = new Uint8Array(1 + encoded.length)
      frame[0] = 0x10
      frame.set(encoded, 1)
      ws.send(frame)
    })

    term.onResize(({ cols, rows }) => {
      // MsgResize frame: [0x02, cols_hi, cols_lo, rows_hi, rows_lo]
      const frame = new Uint8Array(5)
      frame[0] = 0x02
      frame[1] = (cols >> 8) & 0xff
      frame[2] = cols & 0xff
      frame[3] = (rows >> 8) & 0xff
      frame[4] = rows & 0xff
      ws.send(frame)
    })

    const onWindowResize = () => isActive && fitAddon.fit()
    window.addEventListener('resize', onWindowResize)

    return () => {
      window.removeEventListener('resize', onWindowResize)
      ws.close()
      term.dispose()
    }
  }, [sessionId])

  // Hidden tabs: display:none preserves terminal buffer
  return <div ref={containerRef} style={{ display: isActive ? 'block' : 'none', height: '100%' }} />
}
```

### Pattern 5: Tab State Management

**What:** Tab state lives in root `App.tsx`. Each tab has `{ id, name, sessionId }`. The `sessionId` comes from a `CreateSession` Wails bound call. Renaming calls `RenameSession` which updates a name map on the Go side.

**When to use:** Simple `useState` array is sufficient for Phase 3 — no Redux/Zustand needed.

```typescript
interface Tab {
  id: string          // frontend-only UUID
  name: string        // display name, user-editable
  sessionId: string   // Go session ID from CreateSession
}

const [tabs, setTabs] = useState<Tab[]>([])
const [activeId, setActiveId] = useState<string | null>(null)
```

### Pattern 6: Relay Port Discovery

**What:** The relay HTTP server binds to `127.0.0.1:0` (OS-assigned port). The Go `App` struct exposes `GetRelayPort() int` as a bound method. The frontend calls this once on startup to construct WebSocket URLs.

**When to use:** Required — the frontend cannot hardcode a port since 0 is used to avoid conflicts.

```go
func (a *App) GetRelayPort() int {
    return a.httpSrv.Listener.Addr().(*net.TCPAddr).Port
}
```

### Anti-Patterns to Avoid

- **Unmounting hidden terminal tabs:** Never unmount a `TerminalPanel` when switching tabs. Use `display: none`. Disposing the `Terminal` loses the buffer; re-mounting creates a new instance that must replay scrollback from scratch.
- **Calling `fitAddon.fit()` on hidden terminal:** `fit()` reads the container's pixel dimensions. A `display: none` container has zero dimensions; calling fit() corrupts cols/rows. Only call `fit()` on the active, visible terminal.
- **Using `@xterm/addon-attach` directly:** The attach addon expects raw text (UTF-8) or binary data with no framing. Phase 2 uses a 1-byte type prefix. Use the custom WebSocket client in `relayClient.ts` instead.
- **Starting systray.Run() on the main goroutine in Wails:** Wails owns the main goroutine (OS main thread). Use `RunWithExternalLoop` and call `start()` from `OnStartup`. Never call `systray.Run()`.
- **Wails v3:** v3 is alpha. Do not use. Project is committed to Wails v2.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ANSI terminal emulation | Custom VT100/VT220 parser | @xterm/xterm | xterm.js is 100K+ lines of battle-tested VT sequence parsing; covers VT100/VT220/xterm extensions, OSC, DCS, SGR, etc. |
| Terminal resize-to-container | Custom ResizeObserver logic | @xterm/addon-fit | Handles edge cases: scrollbar width, fractional pixels, minimum cols/rows |
| Wide-char (emoji/CJK) width | Custom Unicode width tables | @xterm/addon-unicode11 | Unicode character width is a minefield; addon provides verified tables |
| System tray on macOS/Windows/Linux | NSStatusItem bindings or Win32 Shell_NotifyIcon | fyne.io/systray | Platform-specific tray APIs are deeply painful; fyne.io/systray handles all three |
| Frontend↔Go RPC | Custom HTTP polling or IPC | Wails bound methods | Wails generates TypeScript stubs and handles serialization |
| PTY output broadcast | Custom fan-out logic | Phase 2 HubManager | Already implemented and tested — wire to it, don't replace it |

**Key insight:** The entire value of Wails is that it handles the embedding, the RPC layer, the asset serving, and the webview lifecycle. The entire value of xterm.js is that it handles every VT sequence ever emitted by a terminal application. Resist the urge to build either from scratch.

---

## Common Pitfalls

### Pitfall 1: xterm.js Container Not Visible at open() Time
**What goes wrong:** If `term.open(el)` is called when the container div has zero pixel dimensions (hidden, or not yet rendered), xterm computes 0 cols × 0 rows. All output is discarded or renders incorrectly.
**Why it happens:** xterm.js reads `el.clientWidth`/`el.clientHeight` to compute the initial grid.
**How to avoid:** Only call `term.open()` after the container is rendered and visible. Use a `useEffect` with a DOM-ready check or `requestAnimationFrame`. Alternatively, set a fixed `cols`/`rows` in Terminal options and skip fit() until the first tab is shown.
**Warning signs:** Terminal shows as a tiny 80×24 box inside a large container; fit() does nothing on first call.

### Pitfall 2: WebGL Addon Context Loss on Hidden Tabs
**What goes wrong:** Browsers may reclaim WebGL contexts from canvases that are off-screen or hidden. On recovery, the WebGL addon may error silently.
**Why it happens:** GPU resource limits; browsers enforce max concurrent WebGL contexts.
**How to avoid:** Keep tabs visible via `display: none` rather than removing from DOM. If WebGL context loss occurs, fall back gracefully: catch the WebGL error in an `onContextLoss` handler and switch to canvas renderer.
**Warning signs:** Terminal goes blank when switching back to a tab; console shows "WebGL context lost".

### Pitfall 3: Wails v2 System Tray Does Not Exist
**What goes wrong:** Developer looks at `options.App` fields expecting a `SystemTray` option, doesn't find it, gets confused.
**Why it happens:** System tray was never added to Wails v2. All documentation and examples referencing system tray are for v3 (alpha) or third-party implementations.
**How to avoid:** Use `HideWindowOnClose: true` + `OnBeforeClose` returning `true` + `fyne.io/systray` via `RunWithExternalLoop`. Do not search Wails docs for tray — search fyne.io/systray.
**Warning signs:** `options.App` has no `Tray` or `SystemTray` field. This is expected and correct — use fyne.io/systray separately.

### Pitfall 4: Linux WebKitGTK Version Mismatch
**What goes wrong:** On Ubuntu 22.04, the default WebKitGTK is 4.0. On Ubuntu 24.04, it is 4.1. Building without the `webkit2_41` build tag on 24.04 fails; building with it on 22.04 also fails.
**Why it happens:** WebKitGTK changed API between 4.0 and 4.1, and the package was renamed.
**How to avoid:** Phase 3 targets macOS only for development validation (per STATE.md: cross-platform validation is incremental). Document the build tag for Phase 6. For any Linux CI, use: `wails build -tags webkit2_41` on 24.04+ and default on 22.04.
**Warning signs:** `pkg-config --libs webkit2gtk-4.0` fails on Ubuntu 24.04.

### Pitfall 5: Wails Bindings Not Regenerated After Adding Methods
**What goes wrong:** Developer adds a new exported method to `App` struct, tries to call it from TypeScript, gets "function not found".
**Why it happens:** Wails generates TypeScript bindings at `wails dev` or `wails build` time. Adding methods in Go doesn't automatically update the JS stubs.
**How to avoid:** After adding Go methods, run `wails dev` (which regenerates bindings) or `wails generate module`. The TypeScript files are in `frontend/wailsjs/go/main/App.js`.
**Warning signs:** TypeScript errors showing `App.NewMethod is not a function`.

### Pitfall 6: Relay Port Race Condition
**What goes wrong:** Frontend calls `GetRelayPort()` before the HTTP server has started and bound to a port.
**Why it happens:** `OnStartup` calls `go a.httpSrv.ListenAndServe()` asynchronously. If `OnDomReady` fires before the server goroutine has called `Accept`, the port is not yet assigned.
**How to avoid:** Use `net.Listen("tcp", "127.0.0.1:0")` synchronously in `OnStartup` to obtain the listener (and thus the port) before starting the server goroutine. Store the `net.Listener` in `App`, start the server with `http.Serve(listener, handler)`.
**Warning signs:** `GetRelayPort()` returns 0 intermittently.

---

## Code Examples

### wails.json minimal config
```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "agenthub",
  "outputfilename": "agenthub",
  "frontend:install": "pnpm install",
  "frontend:build": "pnpm run build",
  "frontend:dev:watcher": "pnpm run dev",
  "frontend:dev:serverUrl": "auto",
  "wailsjsdir": "./frontend/src/wailsjs",
  "assetdir": "./frontend/dist"
}
```

### xterm.js Terminal with all required addons
```typescript
// Source: xtermjs.org official addons list + pkg.go.dev npm packages
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { Unicode11Addon } from '@xterm/addon-unicode11'
import { WebglAddon } from '@xterm/addon-webgl'
import '@xterm/xterm/css/xterm.css'

const term = new Terminal({
  scrollback: 10000,
  allowProposedApi: true,   // required by unicode11 addon
  cursorBlink: true,
  fontFamily: '"Cascadia Code", "MesloLGS NF", monospace',
  fontSize: 14,
  theme: {
    background: '#1a1b26',
  },
})
const fitAddon = new FitAddon()
const unicode11 = new Unicode11Addon()
const webgl = new WebglAddon()
term.loadAddon(fitAddon)
term.loadAddon(unicode11)
term.loadAddon(webgl)
term.unicode.activeVersion = '11'
term.open(containerElement)
fitAddon.fit()
```

### Phase 2 binary framing client (TypeScript)
```typescript
// Matches protocol.go constants exactly
const MSG_OUTPUT  = 0x01
const MSG_RESIZE  = 0x02
const MSG_INPUT   = 0x10
const MSG_PING    = 0x12

class RelayClient {
  private ws: WebSocket
  constructor(port: number, sessionId: string, term: Terminal) {
    this.ws = new WebSocket(`ws://127.0.0.1:${port}/sessions/${sessionId}/ws`)
    this.ws.binaryType = 'arraybuffer'
    this.ws.onmessage = (evt) => {
      const buf = new Uint8Array(evt.data as ArrayBuffer)
      if (buf[0] === MSG_OUTPUT) term.write(buf.slice(1))
    }
    term.onData((s) => this.sendInput(s))
    term.onResize(({ cols, rows }) => this.sendResize(cols, rows))
  }
  sendInput(text: string) {
    const enc = new TextEncoder().encode(text)
    const frame = new Uint8Array(1 + enc.length)
    frame[0] = MSG_INPUT
    frame.set(enc, 1)
    this.ws.send(frame)
  }
  sendResize(cols: number, rows: number) {
    const frame = new Uint8Array(5)
    frame[0] = MSG_RESIZE
    frame[1] = (cols >> 8) & 0xff; frame[2] = cols & 0xff
    frame[3] = (rows >> 8) & 0xff; frame[4] = rows & 0xff
    this.ws.send(frame)
  }
  close() { this.ws.close() }
}
```

### Wails OnBeforeClose hide-to-tray
```go
// Source: Wails options docs + community pattern
func (a *App) beforeClose(ctx context.Context) bool {
    // Hide window; do NOT quit the app.
    // Return true = prevent Wails from quitting.
    runtime.WindowHide(ctx)
    return true
}
```

### Relay server binding with synchronous port assignment
```go
// Source: standard library net.Listen pattern
ln, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil {
    log.Fatalf("relay listen: %v", err)
}
a.relayPort = ln.Addr().(*net.TCPAddr).Port
go http.Serve(ln, a.server)
```

### Resize forwarding — connecting the WebSocket resize frame to the PTY backend
```go
// In relay/server.go handleSession — Phase 3 completes the TODO left in Phase 2
case MsgResize2:
    // Phase 3: decode cols/rows and call backend.Resize
    if len(payload) >= 4 {
        cols := uint16(payload[0])<<8 | uint16(payload[1])
        rows := uint16(payload[2])<<8 | uint16(payload[3])
        _ = hub.Resize(int(cols), int(rows))
    }
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `xterm` npm package | `@xterm/xterm` scoped package | xterm.js v5 (2023) | Old package deprecated for security; all addons moved to `@xterm/*` scope |
| `term.setOption('scrollback', N)` | `term.options.scrollback = N` | xterm.js v5 | API breaking change — old `.setOption()` removed |
| `xterm-addon-fit` | `@xterm/addon-fit` | xterm.js v5 | Package rename only |
| Wails v2 system tray | No built-in support (use fyne.io/systray) | Never added | v2 users must bring their own systray library |
| Canvas renderer | WebGL renderer (addon) | xterm.js v4+ | WebGL provides significantly better performance for large scrollback |

**Deprecated/outdated:**
- `xterm` (old npm package): Do not use. Install `@xterm/xterm`.
- `xterm-addon-*` (old addon packages): Do not use. Install `@xterm/addon-*`.
- `term.setOption()`: Removed in v5. Use `term.options.key = value`.

---

## Open Questions

1. **Resize frame format: MsgResize vs MsgResize2**
   - What we know: Phase 2 protocol.go defines both `MsgResize (0x02)` with 4-byte big-endian payload and `MsgResize2 (0x11)` as "alternative resize format (reserved)". The server.go `handleSession` accepts `MsgResize2` from clients and discards it with a Phase 3 TODO comment.
   - What's unclear: Which message type should the frontend send for resize? The server currently reads `MsgResize2` from the client side but `MsgResize` is what `MakeResizeFrame` produces. This is an asymmetry that needs to be resolved during planning.
   - Recommendation: Planner should standardize on one resize message type. Most natural: frontend sends `MsgResize2 (0x11)`, server decodes it and calls `backend.Resize`. The `MakeResizeFrame` function (used server→client) remains `MsgResize (0x02)` for terminal resize notifications.

2. **Hub.Resize method does not exist yet**
   - What we know: `relay/hub.go` has no `Resize` method. `server.go` has a TODO for Phase 3 to call `backend.Resize`. `pty.SessionBackend.Resize(id, cols, rows)` exists.
   - What's unclear: Does the Hub need to hold a reference to the backend, or does the server look up the session and call backend.Resize directly?
   - Recommendation: The planner should add `Hub.Resize(cols, rows int) error` that calls through to the underlying PTY via a stored reference to the backend.

3. **Tab name persistence scope**
   - What we know: TERM-02 requires tab names to persist "across session reattachment". Phase 3 is in-process only (no database).
   - What's unclear: "Persist across reattachment" — does this mean within one app session (window close + reopen without quitting), or across full app restarts?
   - Recommendation: Since SESS-02 keeps sessions alive when window is closed, persistence within one app lifecycle (in-memory Go map) satisfies the requirement. Full app restart persistence (disk) can be deferred. Planner should confirm this interpretation.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (existing tests in Phase 1/2) + Vitest for React frontend |
| Config file | `frontend/vite.config.ts` — `test` section |
| Quick run command | `go test ./... -short` (Go) or `pnpm test --run` (frontend) |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TERM-01 | Multiple independent tab sessions | integration | `go test ./internal/relay/... -run TestHubManager` | ✅ (Phase 2 hub tests) |
| TERM-02 | Tab rename persists | unit | `go test ./cmd/agenthub/... -run TestRenameSession` | ❌ Wave 0 |
| TERM-03 | ANSI color output renders | manual-only | N/A — visual verification; ANSI pass-through is architectural guarantee | manual |
| TERM-04 | 10K+ scrollback | unit | `go test ./internal/relay/... -run TestScrollback` | ✅ (Phase 2 scrollback tests cover capacity) |
| TERM-05 | Copy/paste | manual-only | N/A — clipboard API requires browser context | manual |
| CLI-03 | Custom CLI path config | unit | `go test ./cmd/agenthub/... -run TestUpdateCLIPath` | ❌ Wave 0 |
| SESS-02 | Sessions alive after window hide | integration | `go test ./cmd/agenthub/... -run TestHideWindowSessionsAlive` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./...`
- **Per wave merge:** `go test ./...` + `pnpm test --run` (frontend)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `cmd/agenthub/app_test.go` — unit tests for `CreateSession`, `RenameSession`, `KillSession`, `GetRelayPort`, `UpdateCLIPath`
- [ ] `cmd/agenthub/tray_test.go` — stub test confirming `HideWindowOnClose` behavior (session count unchanged after hide)
- [ ] `frontend/src/lib/relayClient.test.ts` — unit tests for binary framing encode/decode (MSG_OUTPUT, MSG_INPUT, MSG_RESIZE)
- [ ] Framework install: `pnpm add -D vitest @vitest/coverage-v8` in `frontend/` — if frontend dir is new

---

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options` — App struct fields, HideWindowOnClose, OnBeforeClose, lifecycle hooks
- `pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu` — TrayMenu struct, Menu/MenuItem definitions
- `pkg.go.dev/fyne.io/systray` — RunWithExternalLoop, SetIcon, AddMenuItem signatures
- `github.com/xtermjs/xterm.js` README — official addon list, @xterm/* scoped packages, scrollback config
- Phase 2 source code (`internal/relay/protocol.go`, `server.go`, `manager.go`) — framing protocol, server routing, hub API

### Secondary (MEDIUM confidence)
- `github.com/wailsapp/wails` README — v2.11.0 confirmed as latest stable; v3 is alpha
- Wails GitHub discussion #4514 "V2 SysTray" — confirms v2 has NO built-in systray; fyne.io/systray via RunWithExternalLoop recommended
- Wails GitHub issue #3581 / #3513 — confirms Ubuntu 24.04 requires `-tags webkit2_41`; Ubuntu 22.04 uses webkit2gtk-4.0
- `thedevelopercafe.com` Wails project structure article — wails.json schema, frontend directory layout
- `github.com/Qovery/react-xtermjs` — react-xtermjs v1.0.10 (April 2025), useXTerm hook API
- xterm.js GitHub issue discussions — v5 breaking change: setOption removed, options.scrollback property API

### Tertiary (LOW confidence)
- Various WebSearch results on resize propagation patterns — ANSI resize frame implementation details unverified against official xterm.js API docs
- fyne.io/systray + Wails integration pattern — no official guide exists; inferred from `RunWithExternalLoop` semantics and community discussion

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Wails v2.11.0 confirmed; xterm.js v5 official packages verified; fyne.io/systray confirmed via pkg.go.dev
- Architecture: HIGH — based on actual Phase 1/2 source code structure + Wails documented patterns
- Pitfalls: HIGH for systray/xterm pitfalls (documented in official issue trackers); MEDIUM for WebGL context loss (community reports, not official docs)

**Research date:** 2026-03-18
**Valid until:** 2026-06-18 (stable stack; Wails v2 in maintenance mode; xterm.js v5 stable)
