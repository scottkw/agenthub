# Technology Stack

**Project:** AgentHub
**Researched:** 2026-03-17
**Confidence:** MEDIUM-HIGH (core Go/Wails/xterm.js verified; some integration patterns inferred from multiple sources)

---

## Recommended Stack

### Desktop Shell

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.22+ | Backend language | PTY management, net/http, crypto/tls, embed.FS — all stdlib. Single binary output. |
| Wails v2 | v2.11.0 | Desktop window + Go<->JS bridge | Stable, production-grade, React templates built-in. v3 is still alpha as of March 2026. |
| Wails CLI | latest | Build tooling | `wails build` produces signed distributable per platform. |

**Why Wails v2 not v3:** v3 is in active alpha (v3.0.0-alpha.74 as of March 2026). API is "reasonably stable" per the team but documentation and tooling are unfinished. v2.11.0 is stable, has a React+TypeScript+Vite template, and is battle-tested for cross-platform distribution. Migrate to v3 after it stabilizes — the API gap is manageable.

### Frontend Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| React | 19.x (^19.2) | UI framework | Wails ships a `react-ts` template out of the box. Component model maps cleanly to tabbed terminal sessions. |
| TypeScript | 5.6+ | Type safety | Catches xterm.js API misuse at compile time. Wails generates typed bindings for Go functions. |
| Vite | 5.x | Dev server + bundler | Default bundler in Wails react-ts template. HMR works with `wails dev` via `hmr: { host: "localhost", protocol: "ws" }` config. |

### Terminal Rendering

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| xterm.js | 5.x (`@xterm/xterm`) | Terminal emulator in browser/webview | Industry standard. Used by VS Code terminal. xterm.js 5.x is the stable API; 6.x exists but is a major breaking change — verify addon compatibility before adopting. |
| @xterm/addon-attach | 0.9.x | Attach terminal to WebSocket | Official addon. Maps WebSocket binary/text frames to terminal I/O bidirectionally. |
| @xterm/addon-fit | 0.10.x | Resize terminal to container | Required to track container size changes; must fire on every layout change. |
| @xterm/addon-web-links | 0.11.x | Clickable URLs in terminal output | Table stakes for AI CLI output which frequently prints URLs. |
| @xterm/addon-webgl | 0.18.x | GPU-accelerated rendering | Dramatically better perf for high-throughput AI CLI output. Falls back to canvas automatically on unsupported GPUs. |

**Note on xterm.js versioning:** The npm package was renamed from `xterm` to `@xterm/xterm` in v5. The `@xterm/addon-attach` replaces the old `xterm-addon-attach`. Use the `@xterm/` scoped packages exclusively — the un-scoped legacy packages are abandoned.

**Note on React wrappers:** All React wrapper libraries (`react-xtermjs`, `xterm-for-react`, `@pablo-lion/xterm-react`) are unmaintained or last updated 2+ years ago. Do NOT use them. Instantiate `Terminal` directly in a `useEffect` hook with a `ref` to a container div. This is the correct idiomatic pattern and avoids the stale-wrapper problem.

### PTY Management

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| github.com/aymanbagabas/go-pty | v0.2.2 | Cross-platform PTY spawning | Explicitly supports macOS, Linux (Unix PTYs) AND Windows via ConPty. `creack/pty` does NOT have merged Windows ConPty support — its PR #155 remains unmerged. go-pty is the only pure-Go option with real Windows support. |

**tmux mode:** For real tmux sessions, spawn tmux via `go-pty` (or `exec.Command` with PTY attached). The "attach from external terminal" requirement means tmux does the multiplexing — go-pty just handles the PTY wrapper for the Wails side. Detect tmux availability with `exec.LookPath("tmux")`.

**Go-native PTY mode:** For the built-in session persistence mode, go-pty spawns the AI CLI directly. Session state lives in Go (in-memory ring buffer or scrollback). No external dependencies required.

### WebSocket + Web Serving

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| net/http (stdlib) | Go stdlib | HTTP/HTTPS server | Zero deps. Handles both desktop RPC and the web-served terminal sessions on the same process. |
| github.com/coder/websocket | v1.8.x | WebSocket upgrade + I/O | `gorilla/websocket` is archived since 2022. `coder/websocket` (formerly `nhooyr/websocket`) is the actively maintained successor. Uses `context.Context`, handles concurrent writes safely, 2200 LOC vs 3500. |
| crypto/tls (stdlib) | Go stdlib | Self-signed TLS cert generation | Generate ECDSA P-256 cert + key in memory at startup. No cert files to manage, no user setup. |
| embed.FS (stdlib) | Go stdlib | Serve web terminal HTML/JS | Bundle the web terminal UI (xterm.js page) into the binary with `//go:embed`. |

**Architecture note:** The Go process runs two surfaces:
1. Wails IPC bridge — for the desktop UI (React in webview)
2. net/http server bound to VPN/local interface — for browser-accessible web terminals

Both share the same session state. The web terminal HTML page is a minimal xterm.js page embedded via `embed.FS`, served over TLS, with `@xterm/addon-attach` connecting over WSS.

### TLS Strategy

Use `crypto/tls` stdlib to generate self-signed ECDSA P-256 cert + key at first launch, persist to app data dir. On subsequent launches, load from disk. No external CA, no external deps. Browser will warn (expected for local/VPN use). For Tailscale, users can optionally accept the cert — the network-level security from Tailscale is the primary trust boundary.

**Do NOT use mkcert or certmagic** — mkcert requires user interaction and system trust store modification; certmagic is for Let's Encrypt which requires a domain name. Both are wrong for this use case.

### QR Code Generation

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| github.com/skip2/go-qrcode | v0.0.0-latest | QR code PNG generation | Most widely used Go QR library. Generates PNG to `[]byte`. Encode as base64 and return to frontend. Simple API: `qrcode.Encode(url, qrcode.Medium, 256)`. |

**Alternative:** `yeqown/go-qrcode` for styled QR codes with logo overlays. Not needed for this project — plain QR is sufficient.

### VPN / Network Interface Detection

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| github.com/Tailsecurity/tailutils | latest | Tailscale IP detection | `HasTailscaleIP()`, `GetTailscaleIP()`, `GetInterfaceName()`. Thin wrapper, does what we need without pulling in the full Tailscale SDK. |
| net (stdlib) | Go stdlib | Non-Tailscale VPN interface enumeration | `net.Interfaces()` for listing all network interfaces. User can select any VPN interface by name/IP as fallback when Tailscale not present. |

**Do NOT use `tailscale.com/tsnet`** — tsnet embeds a full Tailscale node in your process. We only need to detect the Tailscale interface IP, not run Tailscale ourselves. Pulling in tsnet would add massive binary size and require Tailscale auth in-process.

### Authentication

No external library needed. Implement in stdlib:
- Dashboard password: bcrypt hash stored in app config. Use `golang.org/x/crypto/bcrypt`.
- Session tokens: `crypto/rand` for 32-byte random tokens, hex-encoded. Store in memory map with session ID.

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| golang.org/x/crypto | latest | bcrypt for password hashing | The only non-stdlib dep for auth. Standard Go extended crypto library. |

### Frontend State Management

No Redux, no Zustand, no Jotai. Use React 19 built-in state (`useState`, `useReducer`, `useContext`) for:
- Session list (tab state)
- Per-session WebSocket connection state
- UI preferences

The session count will be small (single digits for typical use). Global state managers add complexity without benefit here.

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Desktop shell | Wails v2 | Electron | Electron adds ~150MB binary, Node.js runtime, separate npm security surface. Wails produces ~10-20MB binary with Go stdlib. |
| Desktop shell | Wails v2 | Wails v3 | v3 is alpha as of March 2026. Alpha.74 is not production-ready. Migrate later. |
| PTY library | go-pty | creack/pty | creack/pty has no merged Windows ConPty support. Windows is a first-class target. |
| WebSocket | coder/websocket | gorilla/websocket | gorilla is archived since 2022. Do not start new projects on archived deps. |
| React wrappers | Direct Terminal instantiation | react-xtermjs / xterm-for-react | All React xterm wrappers are unmaintained. Direct useEffect approach is idiomatic and future-proof. |
| TLS | stdlib crypto/tls | certmagic / Let's Encrypt | Requires domain name. This is local-first. |
| TLS | stdlib crypto/tls | mkcert | Requires user interaction + system trust store modification. |
| VPN detection | tailutils | tsnet | tsnet embeds a full Tailscale node. Overkill — we only need IP detection. |
| State management | React built-in | Zustand/Redux | Session count is small, no complex cross-component state flows warranting a store. |
| QR codes | skip2/go-qrcode | JS-side QR generation | Go side generates PNG → base64 → React. Keeps QR generation out of frontend bundle. |

---

## Installation

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Initialize project with React+TS template
wails init -n agenthub -t react-ts

# Go dependencies (add to go.mod)
go get github.com/wailsapp/wails/v2@v2.11.0
go get github.com/aymanbagabas/go-pty@v0.2.2
go get github.com/coder/websocket@latest
go get github.com/skip2/go-qrcode@latest
go get github.com/Tailsecurity/tailutils@latest
go get golang.org/x/crypto@latest

# Frontend dependencies (from frontend/ dir)
pnpm add @xterm/xterm @xterm/addon-attach @xterm/addon-fit @xterm/addon-web-links @xterm/addon-webgl
```

### Platform Build Requirements

| Platform | Build Machine Required | System Dependencies |
|----------|----------------------|---------------------|
| macOS | macOS runner (no cross-compile) | Xcode CLI tools |
| Linux | Linux runner | `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` (Ubuntu ≤23) or `libwebkit2gtk-4.1-dev` (Ubuntu 24+). Build with `-tags webkit2_41` for Ubuntu 24.04+. |
| Windows | Windows runner (or cross-compile from Linux with MSVC toolchain) | No CGO required (Wails v2 removed CGO for Windows) |

**Cross-platform CI:** Use GitHub Actions matrix with `macos-latest`, `ubuntu-latest`, `windows-latest` runners. macOS CANNOT be cross-compiled from Linux — requires native macOS runner. Use `dAppServer/wails-build-action` community action or write the matrix manually.

---

## Key Integration Points

### 1. Desktop Terminal: Wails IPC → WebSocket → xterm.js

The Wails frontend (React in a WebView) cannot use a raw OS socket. Instead:

1. Go process starts a local WebSocket endpoint per terminal session (e.g., `ws://localhost:{port}/session/{id}`)
2. React connects xterm.js to that WebSocket using `@xterm/addon-attach`
3. Go bridges the WebSocket to the PTY via `go-pty`

This pattern decouples the terminal protocol from Wails' IPC binding layer, which is designed for structured JSON calls — not binary streaming. The WebSocket bridge handles the streaming; Wails IPC handles session management commands (new tab, close tab, resize).

### 2. Web Terminal: Same Go WebSocket Endpoints, External TLS

The same WebSocket endpoints the desktop uses are exposed externally (on the VPN interface) over TLS. A thin HTML page with xterm.js (served via `embed.FS`) connects the same way a browser would. No separate web server process — same Go `net/http` server, same WebSocket handlers, different `http.ListenAndServeTLS` listener bound to the VPN IP.

### 3. PTY Resize: Frontend → Go → PTY

When xterm.js fires `onResize`, send the new columns/rows via Wails IPC (`window.go.App.ResizeSession(id, cols, rows)`). Go calls `go-pty`'s resize method. This keeps the PTY and terminal display in sync.

### 4. QR Code Flow

```
Go: url → qrcode.Encode() → []byte PNG → base64 string
Wails IPC: GetSessionQR(id) → base64 string
React: <img src={`data:image/png;base64,${qr}`} />
```

No frontend QR library needed. The PNG is small enough to pass over IPC without performance concern.

---

## What NOT to Add (Over-Engineering Risks)

| Temptation | Why to Resist |
|------------|--------------|
| Redis / SQLite for session state | Sessions live in process memory. Persistence across app restart is out of scope. In-memory map is sufficient. |
| gRPC instead of Wails IPC + WebSocket | Two protocols already (IPC + WS). Adding gRPC is a third with no benefit. |
| JWT for web session tokens | Random 32-byte hex tokens are simpler, equally secure for this use case. JWT adds parsing complexity with no benefit when there's no claims structure needed. |
| React Query / SWR | No REST API — data comes via IPC or WebSocket push. React Query targets REST/GraphQL; wrong tool here. |
| Tailwind CSS | No strong reason for or against — but Wails default templates don't include it and adding it mid-project increases config complexity. CSS Modules or plain CSS is sufficient for a terminal-focused UI. |
| Plugin system / dynamic loading | Out of scope per PROJECT.md. Initial CLI set is hardcoded. |
| tsnet (embedded Tailscale) | We detect the Tailscale interface, we don't run it. |

---

## Sources

- [Wails v2.11.0 release](https://github.com/wailsapp/wails/releases) — MEDIUM confidence (release page confirmed)
- [Wails v3 alpha status](https://v3alpha.wails.io/) — HIGH confidence (official site)
- [Wails cross-platform build guide](https://wails.io/docs/guides/crossplatform-build/) — HIGH confidence (official docs)
- [Wails Linux webkit2gtk issues](https://github.com/wailsapp/wails/issues/3513) — HIGH confidence (official issue tracker)
- [xterm.js releases](https://github.com/xtermjs/xterm.js/releases) — HIGH confidence (official releases)
- [coder/websocket (formerly nhooyr/websocket)](https://github.com/coder/websocket) — HIGH confidence (official repo)
- [go-pty cross-platform PTY](https://github.com/aymanbagabas/go-pty) — MEDIUM confidence (v0.2.2, last release Jan 2024)
- [creack/pty Windows ConPty PR unmerged](https://github.com/creack/pty/pull/155) — HIGH confidence (PR status verified)
- [tailutils Tailscale IP detection](https://github.com/Tailsecurity/tailutils) — MEDIUM confidence (WebSearch only, not Context7 verified)
- [skip2/go-qrcode](https://pkg.go.dev/github.com/skip2/go-qrcode) — HIGH confidence (well-established library)
- [React 19.2 stable](https://react.dev/versions) — HIGH confidence (official React site)
- [Vite 5 + Wails configuration](https://github.com/wailsapp/wails/issues/3845) — MEDIUM confidence (community issue thread)
