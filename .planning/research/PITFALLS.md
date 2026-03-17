# Domain Pitfalls

**Domain:** Cross-platform desktop terminal multiplexer (Go/Wails/xterm.js)
**Researched:** 2026-03-17
**Overall confidence:** HIGH (most pitfalls verified against official issues and docs)

---

## Critical Pitfalls

Mistakes that cause rewrites or major architectural rethink.

---

### Pitfall 1: creack/pty Has No Stable Windows Support

**What goes wrong:** `creack/pty` — the most common Go PTY library — does not have merged, production-ready Windows support. Multiple PRs (#109, #155) attempted ConPTY integration but the Windows support issue remains open. Windows uses `ConPTY` (available since Windows 10 build 1903), which requires a fundamentally different API: user-provided I/O streams rather than a file descriptor pair. Code that compiles and works on macOS/Linux will fail or behave incorrectly on Windows at runtime.

**Why it happens:** POSIX PTY uses `/dev/pts` file descriptors — straightforward `io.ReadWriteCloser`. Windows ConPTY uses `CreatePseudoConsole()` with separate pipe handles and a special process attribute set via `UpdateProcThreadAttribute`. These APIs cannot be abstracted behind the same interface without platform-specific build tags.

**Consequences:** Windows users get no terminal, or a terminal that appears to work but drops input/output. Go's `os/exec` does not support setting the ConPTY process attribute (Go issues #62708, #6271), making the standard library insufficient for this on Windows.

**Prevention:** Use `github.com/aymanbagabas/go-pty` (Charm's cross-platform PTY library) or `github.com/UserExistsError/conpty` as the Windows ConPTY implementation. These libraries handle the `CreatePseudoConsole` / `os/exec` gap explicitly. Structure PTY code with build tags from day one (`//go:build !windows` vs `//go:build windows`) — do not add Windows support as an afterthought.

**Detection:** The app builds for Windows but terminal sessions produce no output, or `os/exec` panics on process start. Test Windows PTY in Phase 1 before any other feature.

**Phase:** Phase 1 (PTY core) — must be resolved before any further terminal work.

---

### Pitfall 2: win32-input-mode Escape Sequences Corrupt Input on Windows

**What goes wrong:** Windows Terminal enables `win32-input-mode` when output is written to the ConPTY's stdout. In this mode, keystrokes are sent as `\x1b[?9001h` VT sequences rather than raw ASCII or standard VT100. Applications that don't parse this extended format receive garbled input. The sequence to disable win32-input-mode (`\x1b[?9001l`) is unreliable — Windows Terminal continues sending win32-input-mode sequences in many scenarios even after the disable sequence is sent.

**Why it happens:** win32-input-mode was designed to give ConPTY applications richer keyboard input (modifiers, function keys). But it changes the encoding of ALL keystrokes, including simple characters. A terminal multiplexer sitting between Windows Terminal and a hosted process must either forward or translate these sequences.

**Consequences:** AI coding CLIs receive corrupted input. Ctrl+C doesn't interrupt. Arrow keys produce garbage. Claude Code, Gemini CLI, etc. become unusable or unstable.

**Prevention:** Implement a win32-input-mode parser on the Go side using a state machine (a Go 1.23+ iterator-based parser was documented as a working solution in December 2025). Do NOT rely on the disable sequence working. Test input handling with Windows Terminal specifically, not just PowerShell or cmd.exe.

**Detection:** On Windows, typing simple text in the terminal produces garbled output, or Ctrl+C fails to interrupt running processes.

**Phase:** Phase 1 (PTY core / Windows path) — address alongside ConPTY library selection.

---

### Pitfall 3: Self-Signed TLS Certs Are Silently Rejected by Browsers for WebSocket (wss://)

**What goes wrong:** HTTPS pages must use `wss://` — browsers block `ws://` from secure contexts as mixed content (enforced since Chrome 61, now universal across Chrome, Firefox, Safari, Edge). But browsers do NOT show a "click to proceed" dialog for untrusted certificates on WebSocket upgrades. The connection fails silently with a generic error. Users accessing the web terminal from an external browser will see a broken terminal with no actionable error message.

**Why it happens:** Browser security model treats WebSocket certificate rejection as a non-recoverable error. Unlike visiting an HTTPS page where users can click "Advanced > Proceed anyway", there is no equivalent for WebSocket connections.

**Consequences:** The entire web-served terminal feature is broken for any user who hasn't installed the certificate in their OS trust store first. Self-signed certs that work for the desktop app's embedded WebView will not work for external browser access without explicit trust installation.

**Prevention:** Two-pronged approach:
1. Generate certs using the CA pattern (create a local CA cert, sign the server cert with it). Ship the CA cert for user installation. Provide in-app instructions to install it per OS.
2. Consider using `mkcert` as a model — it automates local CA creation and trust store installation. Consider bundling this workflow in the app's first-run setup.

On the serving side: ensure the Go TLS server presents the full chain (server cert + CA cert) not just the leaf cert. Browsers reject incomplete chains.

**Detection:** External browser shows "WebSocket connection failed" or mixed content error. The terminal is blank. Adding the cert to the system trust store and restarting the browser resolves it.

**Phase:** Phase 2 (TLS + web serving) — design the cert generation and trust-installation flow here, not as a Phase 1 afterthought.

---

### Pitfall 4: tmux Does Not Exist on Windows — Feature Parity Gap Is Larger Than Expected

**What goes wrong:** tmux is a POSIX application. It does not run natively on Windows. Options for Windows include Cygwin/MSYS2 bundles (fragile, large), WSL (requires WSL enabled, adds complexity), or skipping tmux entirely. The project already plans a "Go-native PTY mode" as a fallback, but if tmux mode is treated as the primary feature and Go-native PTY as a secondary fallback, Windows ships as a degraded experience.

**Why it happens:** tmux uses POSIX APIs (`socketpair`, `fork/exec`, `sigwinch`, `/tmp` socket paths) that have no direct Windows equivalents.

**Consequences:** If the codebase assumes tmux is available for session management (e.g., `tmux new-session` in the session lifecycle), all of that code is dead on Windows. Session attach/detach from external terminal (`tmux attach`) is a documented feature that is entirely unavailable on Windows without WSL.

**Prevention:** Treat Go-native PTY mode as the primary implementation and tmux mode as an enhancement available when tmux is detected. Code the session lifecycle interface (create, list, attach, resize, kill) as an abstraction layer with two implementations: `TmuxBackend` and `NativePTYBackend`. Both satisfy the same `SessionBackend` interface. Windows uses `NativePTYBackend` unconditionally. macOS/Linux choose based on tmux availability and user preference.

**Detection:** On Windows, any code path that shells out to `tmux` will fail with "executable not found". The symptom is session creation failure at startup.

**Phase:** Phase 1 (session backend design) — the abstraction layer must exist from the start.

---

### Pitfall 5: Wails v3 Is Still Alpha — API Stability Is Not Guaranteed

**What goes wrong:** Wails v3 is in alpha as of 2026-03. The API is described as "reasonably stable" but there is no release date, no SLA, and known issues remain. The v2 to v3 migration path exists but is not trivial. Choosing v3 now means building on an alpha framework.

**Why it happens:** Wails v3 is a significant rewrite with a new multi-window architecture and plugin system. The team adopted a daily release strategy to push through to Beta.

**Consequences:** API changes mid-build can break the application. Debugging issues requires reading GitHub issues rather than stable documentation. Some features documented in v3 may change or be removed before release.

**Prevention:** Default to Wails v2 (latest stable release) unless a specific v3 feature is required. v2 is production-proven, has stable documentation, and the core Wails IPC + WebView embedding model is identical to what v3 will be. Re-evaluate v3 migration when it reaches stable release.

**Detection:** Build errors on framework upgrade, or Wails IPC method signatures changing unexpectedly.

**Phase:** Phase 1 (project scaffolding) — make the v2 vs v3 decision explicitly before writing any Wails-specific code.

---

## Moderate Pitfalls

---

### Pitfall 6: xterm.js Flow Control — Fast Producers Overflow Buffers

**What goes wrong:** xterm.js `write()` is non-blocking and buffers up to 50MB. AI coding CLIs can produce large output bursts (build logs, file scans, `find` results). Without flow control, fast PTY output fills the 50MB buffer, data is silently discarded, and the browser tab may become unresponsive.

**Why it happens:** The WebSocket transport between Go backend and xterm.js has no built-in back-pressure mechanism. The PTY can write faster than xterm.js can render.

**Consequences:** Missing output (silently dropped), browser tab freezing under heavy load, perceived as a broken terminal.

**Prevention:** Implement watermark-based flow control. Use xterm.js's write callback:
```javascript
pty.onData(chunk => {
  pty.pause();
  term.write(chunk, () => { pty.resume(); });
});
```
For WebSocket transport, implement an ACK-based flow control protocol: the client sends back an ACK message after each chunk is processed. The Go server waits for ACK before sending the next chunk. This spans the back-pressure accounting across the WebSocket boundary where direct pause/resume semantics are unavailable.

**Detection:** Under heavy output (e.g., `find /` or a large build), the terminal stops updating partway through, or output appears truncated.

**Phase:** Phase 2 (xterm.js WebSocket integration) — design the flow control protocol before wiring up the first terminal.

---

### Pitfall 7: Terminal Resize Race Condition (SIGWINCH / ConPTY Resize)

**What goes wrong:** When the user resizes the xterm.js terminal pane, the new dimensions must propagate: browser → WebSocket → Go backend → PTY resize (`ioctl(TIOCSWINSZ)` on POSIX, `ResizePseudoConsole()` on Windows). If resize events arrive before the PTY process has started, or are processed out of order, the process never gets the correct window size. On ConPTY, resizing near client attach/detach events can be silently ignored.

**Why it happens:** Browser resize events are frequent. The WebSocket introduces latency. PTY resize and process start are not synchronized.

**Consequences:** AI coding CLIs (Claude Code, Gemini CLI) use the terminal dimensions for their TUI layout. Incorrect dimensions cause wrapping artifacts, truncated lines, or UI elements outside the viewport.

**Prevention:** Debounce resize events in the browser (100-200ms). Send dimensions as part of the initial WebSocket handshake so the PTY is created with the correct size from the start. On Windows, test ConPTY resize specifically — there is a documented bug where resize calls near client attach are silently ignored.

**Detection:** TUI-based CLIs (Claude Code, OpenCode) show incorrect line wrapping immediately after launch, or after window resize.

**Phase:** Phase 2 (PTY + xterm.js wiring) — handle resize protocol in the same pass as initial terminal setup.

---

### Pitfall 8: Wails WebView + Custom HTTP Server CORS Conflicts

**What goes wrong:** The Wails frontend runs at the `wails://wails` or `http://wails.localhost` origin (platform-dependent). When the same Go process also serves an HTTP/WebSocket server on a real port for external browser access, cross-origin requests from the embedded WebView to the local server are blocked by CORS. In Wails v3 specifically, sending large payloads (2MB+) via service calls can fail with a CORS policy error.

**Why it happens:** The WebView origin and the Go HTTP server origin are different even though they're in the same process. The browser's same-origin policy applies regardless.

**Consequences:** The embedded desktop UI cannot talk to the same Go server that serves external browser sessions. The Wails IPC bridge and the web-serving HTTP server must be carefully isolated so they don't conflict.

**Prevention:** Route all desktop app communication through Wails IPC (Go method bindings), never via direct HTTP to the local server. The web-serving HTTP server is for external browsers only. Add proper CORS headers to the Go HTTP server for the cases where the WebView legitimately needs to call it. Explicitly add `Access-Control-Allow-Origin` for the Wails origin.

**Detection:** Console shows "has been blocked by CORS policy" when desktop UI calls the Go server directly. Or large IPC payloads fail silently.

**Phase:** Phase 2 (architecture of desktop IPC vs. web server paths) — separate the concerns before building any frontend features.

---

### Pitfall 9: macOS Notarization Requires Paid Developer Account and Correct Tooling

**What goes wrong:** The Wails documentation references `gon` for notarization, which uses the deprecated `altool` API. Apple has migrated to `notarytool`. Additionally, notarization requires a paid Apple Developer account ($99/year). Free accounts cannot notarize, and macOS Gatekeeper will show "unverified developer" warnings that most users will not know how to bypass.

**Why it happens:** Apple requires all distributed macOS apps to be signed and notarized to avoid Gatekeeper warnings. The toolchain for this has changed and documentation hasn't fully caught up.

**Consequences:** macOS users downloading the app see "Apple could not verify this app is free from malware" and many will refuse to open it. This significantly harms adoption.

**Prevention:** Use `notarytool` (not `altool`) in the CI/CD pipeline. Create the notarization zip with `ditto -c -k --keepParent` (not `zip -r` which causes invalid signature errors). Validate `notarytool` output in CI — it exits with status 0 even on failure, so parse stdout for errors. Budget for the Apple Developer account cost.

**Detection:** `notarytool` exits 0 but the output contains "Invalid" or "rejected". Or macOS shows Gatekeeper warning on first launch.

**Phase:** Phase 5 (distribution/packaging) — plan for this in advance, not as a last-minute issue.

---

### Pitfall 10: AI Coding CLIs Require a Proper PTY (Not Just a Pipe)

**What goes wrong:** Claude Code, Gemini CLI, and OpenCode are interactive TUI applications built with libraries like Bubble Tea or Ink. These libraries check `isatty()` on stdin/stdout and switch to a simplified non-interactive mode (or refuse to run) if PTY is not detected. Running them via a plain `exec.Cmd` with piped stdin/stdout will not produce a functional interactive terminal.

**Why it happens:** TUI libraries use ANSI escape codes for cursor positioning, color, and dynamic rendering. These rely on terminal capabilities reported via `TERM` environment variable and `isatty()`. Pipes don't support these.

**Consequences:** Claude Code launched without a PTY shows degraded output or exits immediately. Gemini CLI's PTY support for interactive subcommands (like `vim`) was explicitly added as a feature — it requires PTY to work at all.

**Prevention:** Always spawn AI coding CLIs inside a properly allocated PTY. Set the `TERM=xterm-256color` environment variable. Do not assume the inherited environment from the Wails process will have a correct `TERM` setting (the WebView process environment may not).

**Detection:** Claude Code starts but shows no TUI, or outputs "not a TTY" and exits. Gemini CLI's interactive commands fail silently.

**Phase:** Phase 1 (PTY session launch) — test with Claude Code and Gemini CLI as integration tests for the PTY implementation.

---

### Pitfall 11: VPN Interface Detection Is Platform-Specific and Fragile

**What goes wrong:** Binding the Go HTTP server to a specific VPN interface IP (e.g., Tailscale's `100.x.y.z` address) requires detecting that interface reliably. On macOS, Tailscale uses a `utun` interface. On Linux, it uses `tailscale0`. On Windows, it uses a WireGuard virtual adapter with a non-deterministic name. Interface names change. The interface may not exist yet when the app starts (if Tailscale isn't connected).

**Why it happens:** `net.Interfaces()` returns all interfaces, but none have stable, cross-platform names. Tailscale's IP range (`100.64.0.0/10`) is more reliable than interface names for detection, but address assignment is also not instant.

**Consequences:** If the app tries to bind to a VPN interface that doesn't exist yet, `net.Listen()` returns an error and web serving fails. If interface detection uses interface names, it fails on any platform other than the one it was tested on.

**Prevention:** Use IP range heuristics (`100.64.0.0/10` for Tailscale) rather than interface names for detection. Poll `net.Interfaces()` on a short interval at startup, with a timeout and graceful fallback to `0.0.0.0`. For Tailscale specifically, the `tailscale.com/net/netns` package provides `SetBindToInterfaceByRoute` — but be aware it currently only affects behavior on macOS and Windows. On Linux, binding is handled differently. Expose the detected VPN IP in the UI and allow manual override.

**Detection:** App starts but web serving reports "bind: cannot assign requested address". Tailscale indicator in UI shows "not connected" even when Tailscale is running.

**Phase:** Phase 3 (VPN binding feature) — prototype interface detection on all three platforms before committing to an architecture.

---

### Pitfall 12: PTY Process Orphans on App Close or Session Disconnect

**What goes wrong:** When the Wails app window is closed, the Go process receives a shutdown signal. If PTY child processes (Claude Code, etc.) are not explicitly terminated, they become orphans with PPID=1 and continue running, consuming memory and CPU indefinitely. This is a documented real-world issue in both OpenCode (GitHub issue #12913) and Gemini CLI (issue #20941 — nested background processes survive PTY abort).

**Why it happens:** Go's `os/exec` does not automatically kill child processes when the parent exits. PTY-based processes are not in the same process group by default on all platforms, so `cmd.Process.Kill()` may not kill child subprocesses spawned by the AI CLI.

**Consequences:** Multiple forgotten AI coding CLI sessions silently consume RAM. On resource-constrained machines, this can cause OOM within hours. Users report the app "leaking" processes.

**Prevention:**
- Use process groups: start PTY processes with `SysProcAttr{Setpgid: true}` on POSIX. Kill the entire group with `syscall.Kill(-pgid, syscall.SIGKILL)` on shutdown.
- On Windows, use a Job Object to ensure child process termination when the parent exits.
- Register a `signal.NotifyContext` shutdown handler in the Go process that iterates all active sessions and terminates them before exiting.
- Implement a session registry that tracks all live PTY processes.

**Detection:** `ps aux | grep claude` (or equivalent) shows multiple `claude` processes after closing the app. Memory usage grows across app restarts.

**Phase:** Phase 1 (session lifecycle) — build the session registry and shutdown handler before shipping any session management.

---

## Minor Pitfalls

---

### Pitfall 13: Wails WebView2 (Windows) Lacks Cookie Support

**What goes wrong:** Microsoft WebView2 on Windows has limited cookie support from the perspective of Wails-served content. This can affect session auth token storage if the frontend uses `document.cookie`. WebView2 uses Blink/Edge rendering which behaves differently from macOS WebKit and Linux WebKitGTK.

**Prevention:** Use `localStorage` or `sessionStorage` instead of cookies for auth token persistence in the embedded desktop UI. Do not rely on browser cookies for the Wails IPC layer. For the external web interface (served to external browsers), cookies work normally.

**Phase:** Phase 2 (auth implementation for web dashboard).

---

### Pitfall 14: Linux WebKitGTK Version Fragmentation

**What goes wrong:** Different Linux distributions ship different versions of WebKitGTK. Wails has had to update its dependency from `webkit2gtk-4.0` to `webkit2gtk-4.1` for Ubuntu 24.04 LTS. The NVIDIA/Wayland combination triggers a DMA-BUF renderer crash (Error 71, Protocol error) that requires auto-detection and fallback.

**Prevention:** Test on Ubuntu 22.04, Ubuntu 24.04, Fedora latest, and Arch. Set `WEBKIT_DISABLE_COMPOSITING_MODE=1` as an environment variable fallback for Wayland/NVIDIA. Wails v2 has added auto-detection for this — verify the installed version handles it. Document minimum WebKitGTK version in system requirements.

**Phase:** Phase 5 (packaging and distribution) — define minimum Linux requirements explicitly.

---

### Pitfall 15: xterm.js Blob Message Ordering (Attach Addon)

**What goes wrong:** The `@xterm/addon-attach` uses `FileReader.readAsArrayBuffer` for Blob messages, which is asynchronous. If the server sends multiple WebSocket frames in quick succession and some are Blobs, they can be processed out of order — later text messages render before earlier Blob messages complete.

**Prevention:** Use `ArrayBuffer` instead of `Blob` as the WebSocket binary type (`ws.binaryType = 'arraybuffer'`). This makes all binary messages synchronous in the message handler. Do not rely on the attach addon's default Blob handling.

**Phase:** Phase 2 (xterm.js WebSocket integration) — set the binary type explicitly in the WebSocket setup.

---

### Pitfall 16: WebSocket Heartbeat / Tab Visibility

**What goes wrong:** When the browser tab is in the background or the user switches windows, the browser throttles JavaScript execution. The xterm.js WebSocket connection can appear to hang — the backend process continues writing but the frontend stops processing messages. The connection doesn't technically drop, but it appears frozen.

**Prevention:** Implement WebSocket ping/pong at the Go server level (not just relying on the browser). Use the gorilla/websocket `SetPongHandler` and send periodic `WriteControl(PingMessage)`. Detect dead connections server-side and clean up. Client-side, the WebSocket reconnection logic should handle tab resume transparently without requiring a page reload.

**Phase:** Phase 2 (WebSocket infrastructure) — add heartbeat in the same pass as WebSocket setup.

---

### Pitfall 17: HTTP/2 and WebSocket Upgrade Conflict

**What goes wrong:** Go's `net/http` package automatically enables HTTP/2 when serving with TLS. HTTP/2 uses multiplexed streams over a single connection — WebSocket upgrades (`Connection: Upgrade`) are not compatible with HTTP/2 framing. Gorilla WebSocket explicitly does not support HTTP/2.

**Why it happens:** HTTP/2 spec (RFC 7540) does not define how to do a WebSocket upgrade over HTTP/2 (that's HTTP/3's territory via WebTransport). Gorilla WebSocket falls back gracefully, but only if the server properly negotiates HTTP/1.1 for WebSocket endpoints.

**Prevention:** Explicitly configure the TLS server to negotiate HTTP/1.1 for WebSocket paths. In Go, this means setting `TLSNextProto` to an empty map for the WebSocket handler's HTTP server, or using a different listener configuration for WebSocket vs non-WebSocket routes.

**Detection:** WebSocket connections fail with `bad handshake` or `protocol error` when TLS is enabled. Works over plain HTTP but not HTTPS.

**Phase:** Phase 2 (TLS + WebSocket serving) — test WebSocket over TLS explicitly, not just over plain HTTP.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1: PTY core | creack/pty has no Windows support | Use aymanbagabas/go-pty from day one |
| Phase 1: PTY core | AI CLIs need proper PTY, not a pipe | Test Claude Code launch in Phase 1 |
| Phase 1: Session lifecycle | Orphan processes on app close | Build session registry + shutdown handler before first demo |
| Phase 1: Windows PTY | win32-input-mode corrupts input | Write win32-input-mode parser or use existing solution |
| Phase 1: Session backend | tmux unavailable on Windows | Abstract SessionBackend interface from day one |
| Phase 2: xterm.js integration | Flow control / buffer overflow | Design ACK protocol before wiring WebSocket |
| Phase 2: xterm.js integration | Blob message ordering | Set `binaryType = 'arraybuffer'` immediately |
| Phase 2: xterm.js integration | Tab visibility / WebSocket freeze | Add gorilla/websocket ping/pong in same pass |
| Phase 2: Wails + HTTP server | CORS conflicts between WebView and HTTP server | Route desktop UI through Wails IPC only |
| Phase 2: TLS + WebSocket | HTTP/2 breaks WebSocket upgrade | Configure HTTP/1.1 for WebSocket endpoints |
| Phase 2: TLS design | Self-signed certs rejected silently for wss:// | Design CA + cert installation flow, not just cert generation |
| Phase 3: VPN binding | Interface detection fragile and platform-specific | Use IP range heuristics, poll with timeout, manual override |
| Phase 3: Terminal resize | SIGWINCH race on Windows ConPTY | Debounce + send dimensions in initial handshake |
| Phase 5: macOS distribution | notarytool vs altool, paid account required | Use notarytool, budget for Apple Developer account |
| Phase 5: Linux distribution | WebKitGTK version fragmentation | Test Ubuntu 22.04, 24.04, Fedora, Arch; document minimums |

---

## Sources

- [creack/pty Windows support issue #95](https://github.com/creack/pty/issues/95) — open since 2020, Windows support unmerged
- [go-pty cross-platform library (aymanbagabas)](https://github.com/aymanbagabas/go-pty) — Charm's cross-platform PTY with ConPTY support
- [Win32-input-mode ConPTY Go application (DEV, Dec 2025)](https://dev.to/andylbrummer/taming-windows-terminals-win32-input-mode-in-go-conpty-applications-7gg) — win32-input-mode parser implementation
- [ConPTY resize bug near client attach](https://github.com/microsoft/terminal/issues/10400) — ConPTY resize silently ignored
- [ConPTY modifies escape sequences](https://github.com/microsoft/terminal/issues/12166) — escape sequence passthrough issue
- [xterm.js Flow Control Guide](https://xtermjs.org/docs/guides/flowcontrol/) — official watermark strategy documentation
- [xterm.js 50MB buffer limit / flow control issue #2077](https://github.com/xtermjs/xterm.js/issues/2077) — buffer overflow documented
- [xterm.js Blob message ordering issue #1893](https://github.com/xtermjs/xterm.js/issues/1893) — async Blob reordering
- [Wails CORS issue #1642](https://github.com/wailsapp/wails/issues/1642) — wails:// origin blocked
- [Wails v3 large payload CORS issue #4428](https://github.com/wailsapp/wails/issues/4428) — 2MB+ CORS failure
- [Wails WebSocket message truncation fix PR #4215](https://github.com/wailsapp/wails/pull/4215) — websocket fragmentation bug
- [Wails v3 release status discussion #4447](https://github.com/wailsapp/wails/discussions/4447) — alpha, no release date
- [Wails macOS notarization issue #3290](https://github.com/wailsapp/wails/issues/3290) — notarization failures
- [Wails Ubuntu 24.04 WebKitGTK issue #3581](https://github.com/wailsapp/wails/issues/3581) — webkit2gtk-4.1 dependency
- [Wails Linux distribution support guide](https://wails.io/docs/guides/linux-distro-support/) — official distro support docs
- [Wails code signing guide](https://wails.io/docs/guides/signing/) — macOS signing documentation
- [OpenCode orphan process issue #12913](https://github.com/anomalyco/opencode/issues/12913) — process leak on disconnect
- [Gemini CLI PTY process group issue #20941](https://github.com/google-gemini/gemini-cli/issues/20941) — nested process tree not killed
- [Gemini CLI PTY support for interactive commands](https://developers.google.com/gemini-code-assist/docs/gemini-cli) — PTY required for vim/interactive subcommands
- [Chrome self-signed cert for WSS — CodeLessGenie](https://www.codelessgenie.com/blog/can-t-connect-to-local-node-js-secure-websocketserver/) — WSS with self-signed certs silently rejected
- [HTTP/2 WebSocket proposal golang/go #53209](https://github.com/golang/go/issues/53209) — WebSocket over HTTP/2 not supported
- [Gorilla WebSocket](https://pkg.go.dev/github.com/gorilla/websocket) — standard Go WebSocket library
