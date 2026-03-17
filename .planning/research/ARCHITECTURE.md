# Architecture Patterns

**Domain:** Go/Wails Desktop App with Web Terminal Serving
**Researched:** 2026-03-17

---

## Recommended Architecture

AgentHub runs as a single Go process that hosts three distinct communication surfaces: the Wails webview (desktop UI), an embedded HTTPS web server (remote browser access), and a WebSocket relay layer shared by both. All three surfaces share the same in-memory session registry, making session state authoritative in Go with no sync or IPC needed.

```
┌─────────────────────────────────────────────────────────┐
│                    AgentHub Process                      │
│                                                          │
│  ┌──────────────┐    ┌──────────────────────────────┐   │
│  │  Wails Shell │    │    Go Core (shared state)     │   │
│  │              │    │                               │   │
│  │  webview     │◄───┤  SessionRegistry              │   │
│  │  bindings    │    │  PTYManager                   │   │
│  │  events      │    │  TmuxBridge                   │   │
│  └──────┬───────┘    │  TLSManager                   │   │
│         │            │  NetworkProbe                 │   │
│  React Frontend       │  AuthManager                 │   │
│  (embedded assets)   └────────────┬─────────────────┘   │
│                                   │                      │
│                       ┌───────────▼──────────────┐      │
│                       │  Embedded HTTPS Server   │      │
│                       │  (net/http + gorilla/ws) │      │
│                       │  Binds to VPN interface  │      │
│                       └───────────┬──────────────┘      │
└───────────────────────────────────┼─────────────────────┘
                                    │
              ┌─────────────────────┼──────────────────┐
              │                     │                   │
     ┌────────▼──────┐   ┌──────────▼────────┐   ┌────▼────────┐
     │  Local        │   │  Remote Browser   │   │  Dashboard  │
     │  xterm.js     │   │  xterm.js (web)   │   │  (session   │
     │  (Wails WS)   │   │  (HTTPS WS)       │   │   list)     │
     └───────────────┘   └───────────────────┘   └─────────────┘
```

---

## Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| **Wails Shell** | Native window, webview lifecycle, OS events | React Frontend via bindings+events; Go Core via bound methods |
| **React Frontend** | Tab management UI, xterm.js rendering, QR display | Wails bindings (Go calls), local WebSocket (terminal I/O) |
| **SessionRegistry** | Authoritative session state (id, name, PTY ref, connected clients) | PTYManager, TmuxBridge, all WS handlers |
| **PTYManager** | Create/resize/destroy PTY processes (creack/pty or go-pty) | SessionRegistry, WS relay goroutines |
| **TmuxBridge** | Wrap tmux exec calls, parse control-mode output, manage tmux sessions | PTYManager (for attach), SessionRegistry |
| **WebSocket Relay** | Fan-out PTY output to N connected clients; serialize client input to PTY | PTYManager (io.ReadWriter), all xterm.js clients |
| **Embedded HTTPS Server** | Serve web dashboard, per-session WS endpoints, static xterm.js bundle | SessionRegistry, AuthManager, WS Relay |
| **TLSManager** | Generate self-signed cert+key in memory at startup; provide tls.Config | Embedded HTTPS Server |
| **NetworkProbe** | Enumerate net.Interfaces, detect Tailscale 100.x range, present bind options | Embedded HTTPS Server (listen address) |
| **AuthManager** | Bcrypt hashed dashboard password; HMAC-signed session tokens; middleware | Embedded HTTPS Server routes |

---

## Data Flow: Terminal I/O

### Local Desktop Path (Wails webview → PTY)

```
User keystroke in xterm.js
  │
  ▼
xterm.js onData() callback
  │  (binary WebSocket message, type byte 0x00 = input)
  ▼
Local WebSocket ws://localhost:<ephemeral>/ws/sessions/{id}
  │  (served by Embedded HTTPS Server, loopback only for desktop)
  ▼
WS handler in Go: reads message, strips type byte
  │
  ▼
SessionRegistry.GetSession(id) → PTYManager.WriteInput([]byte)
  │
  ▼
PTY master fd (creack/pty or go-pty)
  │
  ▼
Shell process (claude, opencode, codex, etc.)
```

```
PTY output arrives on master fd
  │
  ▼
PTYManager read goroutine (one per session)
  │  loops: io.Read → fan-out
  ▼
SessionRegistry.BroadcastOutput(id, []byte)
  │  (type byte 0x01 = output prepended)
  ├──► Local WS connection(s) → xterm.js.write()
  └──► Remote WS connection(s) → xterm.js.write()
```

### Remote Browser Path (Browser → PTY)

```
Remote browser loads https://<tailscale-ip>:<port>/session/{token}
  │  (serves static HTML + xterm.js bundle)
  ▼
xterm.js connects WSS to /ws/sessions/{id}?token={token}
  │  AuthManager validates token in WS upgrade handler
  ▼
Same WS Relay goroutine as local path
  │  same fan-out, same PTY write path
  ▼
Shared PTY (tmux pane or native PTY)
```

### Terminal Resize Flow

```
Browser/Wails window resize event
  │
  ▼
xterm.js FitAddon.fit() → calculates cols/rows
  │
  ▼
WS message type 0x02 = resize, payload: {cols: uint16, rows: uint16}
  │
  ▼
WS handler: PTYManager.Resize(id, cols, rows)
  │
  ├── Native PTY: pty.Setsize(fd, &pty.Winsize{Rows, Cols})
  │     → kernel sends SIGWINCH to shell process
  └── tmux mode: exec("tmux", "resize-window", "-t", session, "-x", cols, "-y", rows)
```

---

## WebSocket Message Protocol

All WebSocket connections (local and remote) use binary frames with a 1-byte type prefix:

| Byte | Direction | Meaning | Payload |
|------|-----------|---------|---------|
| `0x00` | Client → Server | Terminal input | Raw bytes to write to PTY |
| `0x01` | Server → Client | Terminal output | Raw bytes from PTY to feed xterm.js |
| `0x02` | Client → Server | Resize | 4 bytes: uint16 cols (big-endian), uint16 rows |
| `0x03` | Server → Client | Control message | JSON: `{"type":"connected","sessionId":"..."}` |
| `0x04` | Client → Server | Ping | Empty (keepalive) |

This mirrors the approach used by ttyd/GoTTY (single type-byte prefix on binary frames) and is well-established in the Go+xterm.js ecosystem.

---

## Wails-Specific Integration Patterns

### Bindings (Go → JS RPC)

Wails v2 auto-generates TypeScript wrappers for all public methods on bound structs. The `App` struct exposes:

```go
// Bound to frontend - these become async TS functions
func (a *App) CreateSession(name string, cli string) (SessionInfo, error)
func (a *App) ListSessions() []SessionInfo
func (a *App) DestroySession(id string) error
func (a *App) EnableWebServing(id string) (string, error)  // returns URL
func (a *App) GetNetworkInterfaces() []InterfaceInfo
func (a *App) SetBindInterface(iface string) error
func (a *App) GetQRCode(sessionID string) string           // base64 PNG
func (a *App) GetDashboardURL() string
```

These are called from React as `window.go.main.App.CreateSession(name, cli)`.

### Events (Go → JS push)

Wails events enable push notifications without polling:

```go
// Go side (fire-and-forget push to frontend)
runtime.EventsEmit(ctx, "session:output", SessionOutputEvent{ID: id, Data: b})
runtime.EventsEmit(ctx, "session:created", session)
runtime.EventsEmit(ctx, "session:destroyed", id)
runtime.EventsEmit(ctx, "web:url-changed", url)
```

```typescript
// React side
useEffect(() => {
  EventsOn("session:created", (s: SessionInfo) => addTab(s))
  EventsOn("session:destroyed", (id: string) => removeTab(id))
  return () => { EventsOff("session:created"); EventsOff("session:destroyed") }
}, [])
```

### Local WebSocket for xterm.js

Wails webview has a restriction: it renders inside a webview that communicates via the Wails IPC bridge, not a raw browser. Binary WebSocket connections to external servers work correctly, but the local WebSocket endpoint for terminal I/O should be served on a localhost port separate from the Wails IPC port. The Embedded HTTPS Server listens on two addresses simultaneously:

1. `127.0.0.1:<local-port>` — for the Wails webview xterm.js connections (no TLS, loopback)
2. `<vpn-ip>:<remote-port>` — for remote browser connections (TLS required)

The React frontend calls a Wails binding to get the local WebSocket URL for each session, then connects directly.

---

## Patterns to Follow

### Pattern 1: Single Fan-Out Goroutine Per Session

Each session has exactly one read goroutine draining the PTY master. Output is broadcast to all registered WS connections via a registered client map protected by a mutex (or channel-based hub).

```go
type SessionHub struct {
    clients map[string]*websocket.Conn  // connID → conn
    mu      sync.RWMutex
    ptyCh   chan []byte
}

// One goroutine per session
func (h *SessionHub) drainPTY(pty io.Reader) {
    buf := make([]byte, 4096)
    for {
        n, err := pty.Read(buf)
        if err != nil { break }
        h.broadcast(buf[:n])
    }
}
```

**Why:** Avoids multiple concurrent reads from the same PTY fd. Central broadcast point makes attach/detach trivial (just add/remove from clients map).

### Pattern 2: Startup TLS Cert Generation

Generate the self-signed cert at process startup and hold it in memory. Do not write to disk unless caching for restart performance.

```go
func generateSelfSignedCert(hosts []string) (tls.Certificate, error) {
    key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    template := x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject:      pkix.Name{Organization: []string{"AgentHub"}},
        NotBefore:    time.Now(),
        NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
        KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        IPAddresses:  parseIPs(hosts),
    }
    certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
    // return as tls.Certificate
}
```

**Why:** stdlib-only, no external deps, certificate includes the VPN IP as a SAN so browsers show correct validation warning (not a generic error).

### Pattern 3: Interface Detection for VPN Binding

Use `net.Interfaces()` with CGNAT range heuristic for Tailscale detection:

```go
func detectTailscaleIP() (net.IP, bool) {
    ifaces, _ := net.Interfaces()
    tailscaleCIDR := "100.64.0.0/10"  // RFC6598 CGNAT, used by Tailscale
    _, tsNet, _ := net.ParseCIDR(tailscaleCIDR)
    for _, iface := range ifaces {
        addrs, _ := iface.Addrs()
        for _, addr := range addrs {
            ip, _, _ := net.ParseCIDR(addr.String())
            if tsNet.Contains(ip) { return ip, true }
        }
    }
    return nil, false
}
```

WireGuard interfaces are detected by interface name pattern (`wg0`, `wg1`, etc.) or by the 10.x.x.x/24 subnet they typically use — NetworkProbe exposes all candidates and lets the user select which to bind.

### Pattern 4: tmux Control Mode Bridge

When in tmux mode, AgentHub manages sessions via control mode (`tmux -C attach-session`) rather than raw exec, giving structured event parsing:

```
tmux -C attach-session -t agenthub-base
```

Control mode emits events like `%session-created`, `%window-renamed`, etc. over stdout, and accepts commands over stdin. This avoids polling and gives real-time session state updates to the SessionRegistry.

For xterm.js attachment in tmux mode, the PTYManager creates a new PTY fd and runs `tmux attach-session -t <window>` inside it — the xterm.js output is the live tmux pane content. This is the same approach used by ttyd when running `tmux attach`.

### Pattern 5: Session Tokens (HMAC-signed)

Per-session shareable tokens are HMAC-SHA256 signed with a random server secret generated at startup:

```go
// Signing
mac := hmac.New(sha256.New, serverSecret)
mac.Write([]byte(sessionID))
token := hex.EncodeToString(mac.Sum(nil))

// Verification (WS upgrade handler)
expected := computeToken(sessionID)
if !hmac.Equal([]byte(token), []byte(expected)) {
    http.Error(w, "forbidden", 403)
    return
}
```

**Why:** No database needed. Tokens are valid for the process lifetime. Restart invalidates all tokens (acceptable for a local-first app).

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Reading PTY from Multiple Goroutines

**What:** Having both the local WS handler and remote WS handler both read from the PTY fd directly.

**Why bad:** PTY reads are destructive — each Read consumes the bytes. Two readers get interleaved fragments. Classic race condition.

**Instead:** Single drainPTY goroutine per session, fan-out to registered client channels.

### Anti-Pattern 2: Wails IPC for Terminal I/O

**What:** Routing terminal output through `runtime.EventsEmit(ctx, "terminal:output", data)` for every keystroke response.

**Why bad:** Wails IPC serializes through the webview bridge. High-frequency binary data (terminal output can be 50KB/s+) will saturate the bridge, cause UI jank, and break flow control.

**Instead:** Use a direct WebSocket connection on localhost for all terminal data. Wails bindings are for control-plane calls (create session, list sessions, etc.) only.

### Anti-Pattern 3: Single HTTPS Port for Both Surfaces

**What:** Serving the Wails webview and remote browsers on the same HTTPS listener at the same port.

**Why bad:** TLS on the local WebSocket causes certificate errors in the webview (self-signed, no system trust). Also requires the VPN IP to be reachable locally for the desktop UI to work.

**Instead:** Local (loopback) plain HTTP/WS for the desktop xterm.js connections. Separate TLS HTTPS/WSS listener on the VPN IP for remote access.

### Anti-Pattern 4: Storing Session State in React

**What:** Keeping the canonical session list (IDs, PTY refs, active connections) in React state.

**Why bad:** React state is ephemeral. If the webview refreshes or crashes, all session references are lost while PTYs continue running. Reconnection becomes impossible.

**Instead:** Go is the source of truth. React calls `ListSessions()` on mount and subscribes to events for updates.

### Anti-Pattern 5: One goroutine per WebSocket Client for PTY Read

**What:** Each WS client connection spawns its own goroutine to read from the PTY.

**Why bad:** Same as Anti-Pattern 1. Only one goroutine should own each PTY read loop.

**Instead:** WS client goroutines only write to PTY (input) and receive from a broadcast channel (output). The session hub goroutine owns the read loop.

---

## Component Integration Points

| Integration | Mechanism | Notes |
|-------------|-----------|-------|
| React ↔ Go (control) | Wails bindings (auto-generated TS) | Async, type-safe, for all non-terminal operations |
| React ↔ Go (push events) | Wails EventsEmit/EventsOn | Session lifecycle, URL changes, network status |
| xterm.js ↔ PTY (local) | WebSocket on 127.0.0.1 (plain HTTP) | Bypass TLS for loopback; high-frequency binary data |
| xterm.js ↔ PTY (remote) | WebSocket over TLS on VPN IP | Self-signed cert; remote browsers accept via one-time warning |
| PTYManager ↔ tmux | exec.Command + control mode stdin/stdout | Platform: macOS/Linux only; Windows uses native PTY only |
| PTYManager ↔ PTY | creack/pty (Unix) or go-pty (cross-platform with ConPTY) | go-pty preferred for Windows compatibility |
| TLSManager ↔ HTTPS server | tls.Config with in-memory Certificate | cert includes VPN IP SAN; regenerated on IP change |
| NetworkProbe ↔ OS | net.Interfaces() | Tailscale: CGNAT 100.64.0.0/10 heuristic; WireGuard: interface name |
| AuthManager ↔ HTTP routes | HTTP middleware chain | Dashboard: bcrypt password; WS upgrade: HMAC token |

---

## Build Order and Dependencies

The component dependency graph dictates this build order:

```
Phase 1: PTY Foundation
  PTYManager (creack/pty, basic PTY create/destroy/resize)
  → No other internal dependencies
  → Validates cross-platform PTY approach before investing in UI

Phase 2: Session Registry + WebSocket Relay
  SessionRegistry (in-memory session state)
  WebSocket Hub (fan-out broadcast, connect/disconnect lifecycle)
  → Depends on: PTYManager

Phase 3: Wails Shell + React Skeleton
  Wails app scaffold with bound App struct
  React tab UI with xterm.js
  Local WebSocket connect (loopback, no TLS yet)
  → Depends on: SessionRegistry, WebSocket Hub
  → First visible working demo: local terminal tabs

Phase 4: Web Server + TLS + Auth
  TLSManager (self-signed cert generation)
  NetworkProbe (interface enumeration)
  Embedded HTTPS Server with VPN binding
  AuthManager (password + HMAC tokens)
  Web dashboard + remote xterm.js page
  → Depends on: Session Registry, WebSocket Hub, TLS
  → Validates remote browser access end-to-end

Phase 5: tmux Integration
  TmuxBridge (detect tmux, control mode, session naming)
  tmux mode toggle in SessionRegistry
  → Depends on: PTYManager (supplements it)
  → Can be deferred; native PTY mode is fully functional without it

Phase 6: Polish — QR, VPN UI, Multi-CLI
  QR code generation (skip external dep: use pure-Go qrcode library)
  VPN interface selection UI in React
  Per-session web serving toggle
  → Depends on: all prior phases
```

---

## Scalability Considerations

This is a local-first desktop app. Scale concerns are bounded by what a single developer machine can handle.

| Concern | At 5 sessions | At 20 sessions | At 50 sessions |
|---------|---------------|----------------|----------------|
| PTY goroutines | 10 goroutines (1 read + 1 write per session) | 40 goroutines | 100 goroutines |
| WebSocket clients | 1-2 per session local, 1-3 remote | Fine | Fine |
| Memory per session | ~2MB PTY buffer + tmux overhead | ~40MB | ~100MB |
| CPU (terminal I/O) | Negligible | Negligible | Minor at high throughput |
| TLS cert | One cert, many SNI or wildcard | Same | Same |

No architectural changes needed for realistic usage. The fan-out hub pattern handles multiple concurrent remote viewers per session cleanly.

---

## Cross-Platform Concerns

| Concern | macOS | Linux | Windows |
|---------|-------|-------|---------|
| PTY library | creack/pty (stable) | creack/pty (stable) | go-pty with ConPTY API |
| tmux | Available, full support | Available, full support | Not available — native PTY only |
| VPN detection | net.Interfaces() CGNAT heuristic | Same | Same, plus utun/tun prefix check |
| TLS cert storage | In-memory, no disk write | Same | Same |
| Shell | /bin/zsh or /bin/bash | /bin/bash | cmd.exe or powershell.exe |
| Wails webview | WebKit (WKWebView) | WebKit (WebKitGTK) | WebView2 (Chromium-based) |
| Self-signed cert trust | Keychain (optional import) | NSS/system store (optional) | Windows cert store (optional) |

Windows-specific notes:
- ConPTY is available on Windows 10 build 1809+ (released 2018) — safe to require
- go-pty's `Cmd` type handles the Windows exec.Cmd limitation for PTY attachment
- tmux mode is silently disabled on Windows; the UI should reflect this
- Shell auto-detection: check `COMSPEC` env var, fall back to PowerShell

---

## Wails v2 vs v3 Decision

**Recommendation: Use Wails v2 (stable).**

Wails v3 is in alpha as of March 2026. The v3 service architecture and multi-window support are improvements, but the alpha status is a meaningful risk for a new project. v2 is production-stable and widely used. The AgentHub architecture (single window, bound struct, events) maps cleanly to v2's model.

Revisit v3 migration after v3 reaches stable release.

---

## Sources

- [Wails — How does it work?](https://wails.io/docs/howdoesitwork/) — Wails IPC binding mechanism, event system (HIGH confidence)
- [Wails — Application Development Guide](https://wails.io/docs/guides/application-development/) — goroutine and long-running task patterns (HIGH confidence)
- [Wails v3 What's New](https://v3alpha.wails.io/whats-new/) — v3 alpha status, service pattern differences (HIGH confidence)
- [go-pty — Cross-platform Go PTY](https://github.com/aymanbagabas/go-pty) — Cross-platform PTY with ConPTY for Windows (HIGH confidence)
- [charmbracelet/x/conpty](https://pkg.go.dev/github.com/charmbracelet/x/conpty) — Windows ConPTY package (HIGH confidence)
- [gorilla/websocket](https://pkg.go.dev/github.com/gorilla/websocket) — WebSocket library, connection lifecycle (HIGH confidence)
- [GoTTY](https://github.com/yudai/gotty) — Reference for PTY-over-WebSocket architecture patterns (MEDIUM confidence)
- [ttyd](https://github.com/tsl0922/ttyd) — Reference for message framing in WebSocket terminal apps (MEDIUM confidence)
- [xterm-addon-fit](https://www.npmjs.com/package/xterm-addon-fit) — Terminal resize handling, SIGWINCH propagation (HIGH confidence)
- [tailscale.com/net/interfaces](https://pkg.go.dev/tailscale.com/net/interfaces) — Tailscale IP detection functions (HIGH confidence)
- [Tailscale IP Addresses — 100.x.y.z](https://tailscale.com/docs/concepts/tailscale-ip-addresses) — CGNAT range confirmation (HIGH confidence)
- [Go crypto/tls generate_cert.go](https://go.dev/src/crypto/tls/generate_cert.go) — Self-signed cert generation in stdlib (HIGH confidence)
- [Master tmux: Go Session Manager](https://karandeepsingh.ca/posts/mastering-tmux-go-session-manager/) — tmux exec.Command patterns (MEDIUM confidence)
- [Go SSH to WebSocket with xterm.js](https://medium.com/@razikus/go-ssh-to-websocket-with-xterm-js-33af2e0c3bc7) — Go WebSocket + xterm.js integration pattern (MEDIUM confidence)
