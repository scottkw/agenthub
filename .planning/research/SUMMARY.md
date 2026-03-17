# Project Research Summary

**Project:** AgentHub
**Domain:** Cross-platform desktop terminal multiplexer with remote web serving for AI coding CLIs
**Researched:** 2026-03-17
**Confidence:** MEDIUM-HIGH

## Executive Summary

AgentHub is a niche but well-defined product: a desktop app that hosts multiple AI coding CLI sessions (Claude Code, Codex, Gemini CLI, OpenCode) in a tabbed terminal UI and exposes each session to remote browsers over a self-hosted, VPN-scoped HTTPS connection. The closest prior art is ttyd/GoTTY for web terminal serving and agent-deck for AI CLI session management — AgentHub combines both with a better UX story (QR codes, session toggles, single app). The right approach is a single Go process using Wails v2 for the desktop shell, go-pty for cross-platform PTY management, xterm.js for terminal rendering, and a standard `net/http` server with self-signed TLS for the remote web serving surface. The stack is minimal, stdlib-heavy, and validated against production tools (VS Code uses xterm.js; Coder uses the same websocket library).

The architecture is naturally three layers: PTY management in Go (session lifecycle, process groups, fan-out broadcast), a Wails shell layer (React tabbed UI, Wails IPC for control-plane calls, local WebSocket for terminal I/O), and an embedded HTTPS server (VPN-bound, handles remote browser access and web dashboard). All three layers share one in-memory SessionRegistry — no database, no sync needed. This clean separation also defines the correct build order: PTY core first, session registry and WebSocket relay second, then the Wails frontend, then web serving and TLS, with tmux support and polish deferred.

The primary risks are Windows-specific: PTY library choice (creack/pty has no Windows ConPTY support — must use go-pty from day one), win32-input-mode keyboard encoding (requires a state-machine parser on the Go side), and self-signed TLS certificate trust (browsers silently reject wss:// connections to untrusted certs — requires a CA-signed cert flow, not just a leaf cert). These are not insurmountable, but they are hard to retrofit. Address all three in Phase 1 and Phase 2 before building any user-facing features.

## Key Findings

### Recommended Stack

The stack is deliberately minimal. Go handles everything on the backend — PTY management, WebSocket relay, TLS, VPN detection, QR code generation, and auth. The frontend is React 19 + TypeScript in a Wails v2 webview, with xterm.js 5.x for terminal rendering and `@xterm/addon-attach` connecting xterm.js to Go via WebSocket. No React state libraries, no CSS frameworks beyond what Wails ships — the session count is small and the UI is terminal-focused.

Wails v2 (not v3) is the correct choice: v3 is in active alpha as of March 2026 (alpha.74), with no stable release date and incomplete documentation. The entire stack avoids archived libraries: `coder/websocket` replaces the archived `gorilla/websocket`, direct `useEffect`-based xterm.js instantiation replaces unmaintained React wrappers, and `go-pty` replaces `creack/pty` (which has no merged Windows ConPTY support).

**Core technologies:**
- Go 1.22+ with Wails v2.11.0: desktop shell, PTY management, web server — single binary output
- `github.com/aymanbagabas/go-pty` v0.2.2: cross-platform PTY (macOS, Linux, Windows ConPTY) — the only option with real Windows support
- `github.com/coder/websocket` v1.8.x: WebSocket relay — actively maintained successor to archived gorilla/websocket
- xterm.js `@xterm/xterm` 5.x + `@xterm/addon-attach`, `@xterm/addon-fit`, `@xterm/addon-webgl`: terminal rendering — same library VS Code uses
- React 19 + TypeScript + Vite: frontend UI — ships in Wails react-ts template
- `golang.org/x/crypto` (bcrypt): password hashing — only non-stdlib auth dependency
- `github.com/skip2/go-qrcode`: QR code generation — simple API, PNG to base64 for IPC passthrough
- `crypto/tls` stdlib: self-signed ECDSA P-256 cert generation — no external CA tooling needed

### Expected Features

The must-have feature set for v1 is well-defined and modest. The app must validate one core proposition: "one app, multiple AI agents, accessible from anywhere over VPN." All v1 features are in service of that claim.

**Must have (table stakes):**
- Tabbed xterm.js terminal with session naming, ANSI/Unicode rendering, scrollback (10K+ lines)
- Launch sessions for Claude Code, Codex, Gemini CLI, OpenCode (PATH detection at startup)
- Go-native PTY backend with session persistence across window close
- SIGWINCH propagation (PTY resize when terminal pane is resized)
- Kill/close a session cleanly (SIGHUP + process group kill)
- System tray presence to keep sessions alive when main window is closed

**Should have (competitive differentiators):**
- Per-session web serving toggle with self-signed TLS over VPN interface
- Web dashboard with bcrypt password auth and per-session shareable token links
- QR code generation for web session URLs (validated pattern from predecessor project ccrs)
- VPN interface selection with Tailscale auto-detection (100.64.0.0/10 CGNAT heuristic)

**Defer to v2+:**
- Real tmux backend (Go-native PTY covers the use case; tmux adds complexity and is unavailable on Windows)
- Multi-CLI status indicators (heuristic output parsing is fragile — ship once core is stable)
- Per-session token expiry/revocation (long-lived tokens are acceptable for v1)
- Tab color coding and font/theme customization (polish layer)

**Never build (anti-features):**
- CLI installation/management, VPN/Tailscale setup, Let's Encrypt/ACME, user accounts/registration, cloud SaaS mode, plugin system, session replay/search, MCP server management, split panes.

### Architecture Approach

The architecture is a single Go process with three communication surfaces sharing one in-memory SessionRegistry. The Wails shell hosts the desktop React UI and exposes Go methods via auto-generated TypeScript bindings (Wails IPC) for control-plane operations only. Terminal I/O flows through a separate WebSocket relay on localhost (plain HTTP — no TLS needed for loopback) to avoid saturating the Wails IPC bridge with binary streaming data. An embedded HTTPS server bound to the VPN interface serves remote browser access using the same WebSocket relay endpoints and a static xterm.js bundle embedded via `embed.FS`. The critical architectural constraint is that PTY reads must happen in exactly one goroutine per session, with a fan-out hub broadcasting to all connected WebSocket clients — never multiple goroutines reading the same PTY fd.

**Major components:**
1. **PTYManager** — create/resize/destroy PTY processes, single read goroutine per session, fan-out broadcast via SessionHub
2. **SessionRegistry** — authoritative in-memory session state (id, name, PTY ref, connected clients, web serving toggle)
3. **Wails Shell + React Frontend** — tabbed UI, xterm.js rendering, Wails IPC for control-plane, local WS for terminal I/O
4. **WebSocket Relay / SessionHub** — fan-out PTY output to N clients; serialize client input to PTY; binary framing with 1-byte type prefix
5. **Embedded HTTPS Server** — VPN-bound TLS listener, web dashboard, per-session WS endpoints, static xterm.js bundle
6. **TLSManager** — ECDSA P-256 self-signed cert with VPN IP as SAN; generated at startup
7. **NetworkProbe** — enumerate `net.Interfaces()`, detect Tailscale via CGNAT heuristic, expose manual override
8. **AuthManager** — bcrypt dashboard password; HMAC-SHA256 session tokens (process-lifetime, no DB needed)

### Critical Pitfalls

1. **creack/pty has no merged Windows ConPTY support** — use `github.com/aymanbagabas/go-pty` from day one; test Windows PTY in Phase 1 before any other work.

2. **win32-input-mode corrupts all keyboard input on Windows** — Windows Terminal sends `\x1b[?9001h` mode sequences instead of raw ASCII; implement a state-machine parser in Go for this encoding. The disable sequence is unreliable — do not depend on it.

3. **Self-signed TLS certs are silently rejected for wss:// connections** — browsers show no "proceed anyway" dialog for WebSocket cert errors; must use a local CA pattern (generate CA cert, sign server cert with it, provide in-app cert installation flow) not just a bare self-signed leaf cert.

4. **PTY process orphans on app close** — AI coding CLIs spawn child processes; use `SysProcAttr{Setpgid: true}` + `syscall.Kill(-pgid, SIGKILL)` on POSIX, Job Objects on Windows; register a signal shutdown handler that iterates all sessions. Do this in Phase 1.

5. **AI CLIs require a proper PTY, not a pipe** — Claude Code, Gemini CLI, and OpenCode check `isatty()` and degrade or exit without a real PTY; always set `TERM=xterm-256color` in the spawned environment; test with actual CLI binaries in Phase 1.

## Implications for Roadmap

Based on combined research, the architecture's own build-order dependency graph maps cleanly to a 6-phase roadmap. The critical insight: PTY correctness on all three platforms (especially Windows) must be validated before any user-facing UI work begins, because retrofitting the PTY library or adding win32-input-mode support after the fact is high cost.

### Phase 1: PTY Foundation + Session Lifecycle

**Rationale:** Every other feature depends on a working PTY. Windows PTY correctness (go-pty + win32-input-mode parser) is the highest-risk item in the whole project and must be de-risked first. The session registry and process-group shutdown handler must also exist here — orphan processes are a Phase 1 bug, not a polish item.

**Delivers:** A Go binary that can spawn Claude Code / Gemini CLI in a proper PTY on macOS, Linux, and Windows; read/write PTY I/O; resize correctly; and kill cleanly on shutdown. No UI yet.

**Addresses features:** Session creation, session kill/close, session persistence foundation, CLI detection via PATH

**Avoids pitfalls:** Pitfall 1 (creack/pty Windows), Pitfall 2 (win32-input-mode), Pitfall 4 (PTY orphans), Pitfall 5 (AI CLIs need real PTY), Pitfall 4 (tmux abstraction layer — SessionBackend interface)

**Research flag:** None needed — go-pty is well-documented and the win32-input-mode parser approach is documented (DEV.to article, Dec 2025).

### Phase 2: Session Registry + WebSocket Relay

**Rationale:** The fan-out WebSocket hub is the architectural linchpin that both the local desktop UI and remote browser access share. Getting the protocol right (binary framing, flow control, connection lifecycle) here prevents rewrites later. CORS concerns between Wails WebView and the HTTP server must also be resolved here.

**Delivers:** SessionRegistry with in-memory session state; WebSocket hub with fan-out broadcast; binary message protocol (0x00 input, 0x01 output, 0x02 resize, 0x03 control, 0x04 ping); flow control via write callback or ACK protocol; heartbeat/ping-pong; `binaryType = 'arraybuffer'` set in WebSocket client.

**Uses stack:** `coder/websocket`, `net/http` on localhost, SessionHub pattern (single drainPTY goroutine per session)

**Avoids pitfalls:** Pitfall 6 (xterm.js flow control buffer overflow), Pitfall 7 (terminal resize race), Pitfall 8 (Wails CORS), Pitfall 15 (Blob message ordering), Pitfall 16 (tab visibility WebSocket freeze), Pitfall 17 (HTTP/2 WebSocket conflict)

**Research flag:** None needed — standard patterns with official documentation.

### Phase 3: Wails Shell + React Frontend (Local Desktop UI)

**Rationale:** First user-visible milestone. Wails IPC handles all control-plane calls; local WebSocket handles terminal I/O. Validates the full local stack. React state is kept minimal — Go is source of truth, React calls `ListSessions()` on mount and subscribes to events.

**Delivers:** Working desktop app with tabbed xterm.js terminal, session naming, ANSI/Unicode rendering, scrollback, copy/paste, resize, system tray presence. Connect xterm.js via `@xterm/addon-attach` to the local WebSocket. No web serving yet.

**Uses stack:** Wails v2.11.0 react-ts template, React 19, xterm.js 5.x with webgl/fit/web-links addons, direct `useEffect` Terminal instantiation (no React wrapper libraries)

**Addresses features:** All table stakes features (tabs, ANSI, Unicode, scrollback, resize, copy/paste, system tray)

**Avoids pitfalls:** Anti-Pattern 2 (Wails IPC for terminal I/O), Anti-Pattern 4 (React as session state source of truth)

**Research flag:** Linux WebKitGTK build tags (`webkit2_41` for Ubuntu 24.04) need explicit testing — flag for phase planning.

### Phase 4: Web Serving + TLS + Auth

**Rationale:** The remote access feature is the primary differentiator. TLS cert strategy must be planned holistically here — the CA + server cert pattern (not just a bare self-signed cert) is required for WSS to work in external browsers. Auth (bcrypt password + HMAC tokens) is also implemented here.

**Delivers:** Embedded HTTPS server bound to VPN interface; CA + server cert generation via `crypto/tls` stdlib; per-session web serving toggle; web dashboard with password auth; per-session shareable HMAC tokens; static xterm.js bundle served via `embed.FS`; remote browser connects to same WebSocket relay.

**Uses stack:** `net/http`, `crypto/tls`, `golang.org/x/crypto/bcrypt`, `embed.FS`, NetworkProbe (CGNAT heuristic for Tailscale detection)

**Addresses features:** Per-session web toggle, self-signed TLS, VPN interface binding, web dashboard, per-session tokens

**Avoids pitfalls:** Pitfall 3 (WSS with self-signed certs — CA pattern required), Anti-Pattern 3 (separate TLS port from loopback port), Pitfall 11 (VPN interface detection fragility — IP range heuristics + manual override + poll with timeout)

**Research flag:** The CA cert + in-app trust installation UX is underspecified in research — needs dedicated research during planning to determine per-OS trust store installation approach (macOS Keychain, Linux NSS, Windows cert store).

### Phase 5: QR Codes + VPN UI + Polish

**Rationale:** With the web serving infrastructure in place, QR code generation and VPN interface selection UI are low-complexity features that complete the "walk away" workflow. This phase is mostly wiring existing Go functionality into the React UI.

**Delivers:** QR code generation (`skip2/go-qrcode` → base64 PNG → React `<img>`); VPN interface selection dropdown; Tailscale auto-detection in UI; per-session URL display; basic session status indicators (running/waiting heuristic based on output patterns).

**Uses stack:** `github.com/skip2/go-qrcode`, NetworkProbe, Wails IPC (`GetQRCode`, `GetNetworkInterfaces`, `SetBindInterface`)

**Addresses features:** QR code generation, VPN interface selection, working directory context per tab, session status indicators (basic heuristic)

**Research flag:** Status indicator output pattern heuristics for each CLI (Claude Code, Codex, Gemini CLI, OpenCode) are not well-documented — needs targeted research during planning.

### Phase 6: Distribution + macOS Notarization + Cross-Platform Testing

**Rationale:** Cross-platform builds cannot be cross-compiled for macOS (requires macOS runner). Notarization requires specific tooling (`notarytool`, not `altool`) and a paid Apple Developer account. Linux requires explicit WebKitGTK version handling. This phase must be planned before Phase 1 begins (account setup, CI matrix) even though the work happens last.

**Delivers:** GitHub Actions CI matrix (macOS-latest, ubuntu-latest, windows-latest); macOS signing + notarization via `notarytool`; Linux WebKitGTK 4.0/4.1 build variants; Windows installer.

**Avoids pitfalls:** Pitfall 9 (macOS notarization — notarytool, `ditto` zip, paid account), Pitfall 14 (Linux WebKitGTK fragmentation), Pitfall 13 (WebView2 cookies on Windows — use localStorage)

**Research flag:** Windows code signing (Authenticode) is not covered in research — flag for planning if Windows Defender SmartScreen warnings are a distribution concern.

### Phase Ordering Rationale

- **PTY before UI:** Windows PTY correctness is the most technically risky item. It must be validated in isolation before investing in UI work that assumes it works.
- **Session Registry before everything:** The fan-out hub is shared by local and remote paths. Getting the protocol right once prevents two separate implementations.
- **Local UI before remote:** Remote access requires TLS and auth complexity; validate the core terminal UX first.
- **Web serving as a feature, not infrastructure:** The embedded HTTPS server is isolated from the core terminal loop — it can be added without touching Phase 1-3 code.
- **Notarization planning in Phase 1:** Apple Developer account registration and CI matrix setup takes time; start early even if distribution work is Phase 6.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3:** Linux WebKitGTK version fragmentation — which build tags are needed for Ubuntu 22.04 vs 24.04; Wayland/NVIDIA `WEBKIT_DISABLE_COMPOSITING_MODE` flag
- **Phase 4:** Per-OS TLS cert trust installation UX — macOS Keychain, Linux NSS/p11-kit, Windows certutil; how to surface this in the app's first-run setup without requiring admin rights
- **Phase 5:** Per-CLI status indicator output patterns — what Claude Code, Codex, Gemini CLI, and OpenCode actually print when waiting for user input vs. running vs. idle
- **Phase 6:** Windows Authenticode signing for SmartScreen warnings (if distribution at scale)

Phases with standard patterns (skip research-phase):
- **Phase 1:** PTY patterns with go-pty are documented; win32-input-mode parser has a Dec 2025 reference implementation
- **Phase 2:** WebSocket relay fan-out hub is a standard Go pattern; ttyd/GoTTY serve as reference implementations
- **Phase 4 (auth):** bcrypt + HMAC token pattern is well-established; no research needed

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Core choices (Wails v2, go-pty, coder/websocket, xterm.js 5.x) verified against official sources; alternatives explicitly ruled out with documented rationale |
| Features | HIGH | Table stakes validated against ttyd, GoTTY, agent-deck; differentiators validated against Claude Code Remote Control (Anthropic, Feb 2026); anti-features have concrete rationale |
| Architecture | HIGH | Component boundaries and data flow patterns derived from ttyd/GoTTY reference implementations and official Wails docs; WebSocket protocol is industry-standard |
| Pitfalls | HIGH | Most pitfalls traced to official GitHub issues (creack/pty #95, xterm.js #2077, Wails #1642, Microsoft Terminal #10400); win32-input-mode solution has working Dec 2025 reference |

**Overall confidence:** HIGH

### Gaps to Address

- **CA cert trust installation flow:** Research identifies that bare self-signed certs fail silently for WSS, and recommends a CA pattern, but does not specify the in-app UX or per-OS commands for installing the CA cert. This needs a dedicated design pass before Phase 4 implementation.

- **tailutils library confidence:** `github.com/Tailsecurity/tailutils` was identified via web search but not Context7-verified. The fallback (using `net.Interfaces()` with the CGNAT 100.64.0.0/10 heuristic directly) is well-validated and should be the primary approach. tailutils is a thin convenience wrapper.

- **win32-input-mode parser completeness:** The Dec 2025 reference article describes a working approach, but the exact state machine has not been code-reviewed. Plan a Phase 1 spike to validate before committing to the design.

- **Status indicator heuristics:** No documentation exists for reliable "waiting for input" output patterns across all four CLIs. This is intentionally deferred to v2 for the full feature, but even a basic heuristic for Claude Code (most common) needs empirical testing.

## Sources

### Primary (HIGH confidence)
- [Wails v2.11.0 + v3 alpha status](https://wails.io/) — framework version selection rationale
- [Wails — How does it work?](https://wails.io/docs/howdoesitwork/) — Wails IPC binding mechanism
- [xterm.js releases](https://github.com/xtermjs/xterm.js/releases) — v5.x API, addon compatibility
- [xterm.js Flow Control Guide](https://xtermjs.org/docs/guides/flowcontrol/) — buffer overflow prevention
- [go-pty cross-platform PTY](https://github.com/aymanbagabas/go-pty) — Windows ConPTY support
- [coder/websocket](https://github.com/coder/websocket) — gorilla/websocket successor
- [creack/pty Windows PR #155 unmerged](https://github.com/creack/pty/issues/95) — Windows limitation confirmed
- [Claude Code Remote Control docs](https://code.claude.com/docs/en/remote-control) — QR code + remote access UX validated
- [Go crypto/tls generate_cert.go](https://go.dev/src/crypto/tls/generate_cert.go) — self-signed cert generation
- [Tailscale IP Addresses](https://tailscale.com/docs/concepts/tailscale-ip-addresses) — CGNAT 100.64.0.0/10 confirmed
- [OpenCode orphan process issue #12913](https://github.com/anomalyco/opencode/issues/12913) — process leak documented
- [Gemini CLI PTY issue #20941](https://github.com/google-gemini/gemini-cli/issues/20941) — nested process tree not killed

### Secondary (MEDIUM confidence)
- [Win32-input-mode ConPTY Go parser (DEV.to, Dec 2025)](https://dev.to/andylbrummer/taming-windows-terminals-win32-input-mode-in-go-conpty-applications-7gg) — working parser approach
- [GoTTY](https://github.com/yudai/gotty) — WebSocket terminal architecture reference
- [ttyd](https://github.com/tsl0922/ttyd) — binary message framing reference
- [agent-deck README](https://github.com/asheshgoplani/agent-deck) — AI CLI session manager feature comparison
- [Wails Ubuntu 24.04 WebKitGTK issue #3581](https://github.com/wailsapp/wails/issues/3581) — webkit2gtk-4.1 required
- [Wails CORS issue #1642](https://github.com/wailsapp/wails/issues/1642) — WebView origin CORS behavior
- [Vite 5 + Wails HMR configuration](https://github.com/wailsapp/wails/issues/3845) — community issue thread
- [tailutils Tailscale IP detection](https://github.com/Tailsecurity/tailutils) — WebSearch only, not Context7 verified

### Tertiary (needs validation)
- [tailutils library](https://github.com/Tailsecurity/tailutils) — thin wrapper; fall back to stdlib `net.Interfaces()` if unresponsive

---
*Research completed: 2026-03-17*
*Ready for roadmap: yes*
