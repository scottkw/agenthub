# AgentHub

<p align="center">
  <img src="docs/agenthub-title-logo.png" alt="AgentHub" width="400">
</p>

A cross-platform desktop app, CLI, and TUI for running AI coding CLIs — Claude Code, Codex, Gemini CLI, OpenCode — in persistent terminal sessions managed by a background daemon. Three access modes: GUI (Wails desktop app), CLI (`agenthub` subcommands), and TUI (`agenthub tui` — a full-screen Bubble Tea terminal UI). Sessions survive GUI restarts, are controllable from any interface, and can be shared over the web via Tailscale with browser-trusted TLS — or over the local network with self-signed TLS and password auth when Tailscale isn't available. Multiple clients can connect to the same session simultaneously with independent scrollback, read-only mode, and stable PTY resize arbitration. CLI attach displays a persistent tmux-style status bar with session context and live viewer count. The TUI provides near-GUI parity: two-pane sidebar+content layout with bordered session frames, per-agent colored badges, TokyoNight color palette, focus-aware navigation (Tab toggles panes, [/] cycles tabs), full session lifecycle (attach, create, kill, rename), unified local+remote session list with tailnet peer grouping, ASCII QR code overlay, and discoverable help. Sessions auto-close when the agent process exits — with a 5-second countdown, toast notification, and Keep Open cancel. Closing the GUI presents a quit confirmation modal showing active session count with the choice to quit the GUI only (daemon stays running, macOS notification sent) or quit everything. The web server starts automatically and new sessions are web-served by default. A collapsible sidebar with Heroicons provides quick access to all navigation — Home, Remote Sessions, Daemon Manager, New Session, and Settings (single scrollable page with section headers). System tray works on all platforms: macOS (native NSStatusBar), Linux (D-Bus StatusNotifierItem), and Windows (Shell_NotifyIcon). Agent CLIs are discovered automatically across common install locations (nvm, Volta, Homebrew, snap, flatpak, cargo, pipx, native installers). Terminal sessions support 138 curated color themes (WCAG-audited from the xterm-theme library) with live switching and persistence — including OpenCode, which honors the selected theme via managed config and SIGUSR2 broadcast. The terminal core is extended by a curated, vendored xterm.js plugin suite (v3.2) — GPU-accelerated rendering with software-fallback detection, scrollback search via a Cmd-F find bar (desktop + web), clickable web-links with strict scheme allowlist and IDN/typosquat click-confirmation, inline images via the sixel protocol, Unicode 11 width tables, "Save Terminal As…" via the serialize addon, OSC 52 system clipboard support, and OSC 9;4 progress reporting that surfaces as per-tab progress underlines plus an aggregate tray quartile glyph — all user-controlled from Settings → Plugins. All non-terminal GUI text meets WCAG AA 4.5:1 contrast ratio. Remote sessions on other tailnet machines are discoverable from the GUI, CLI, and TUI. Auto-update notifications keep you on the latest release. Built with Go/Wails and React.

## Features

### Terminal & Sessions
- **Collapsible sidebar** — Left sidebar with Heroicons SVG icons for all navigation: Home, Remote, Sessions, New Session (top); Settings (bottom). Toggle between collapsed (icons only, 48px) and expanded (icons + labels, 200px) via hamburger button; icons stay in fixed horizontal position during transitions; state persists in localStorage
- **Tabbed terminals** — Run multiple AI coding sessions side-by-side with full xterm.js terminals (ANSI 256-color, Unicode, emoji, 10K+ line scrollback, full-width viewport fill)
- **Background daemon** — Sessions live in a standalone daemon process; closing the GUI hides the window while sessions and the system tray remain active
- **CLI auto-detection** — Detects Claude Code, Codex, Gemini CLI, and OpenCode on startup — including when launched from Finder/Dock (augments PATH with `~/.local/bin`, Homebrew, nvm, Volta, snap, flatpak, cargo, pipx, and platform-specific install locations on macOS, Linux, and Windows); supports custom CLI path overrides
- **New session modal** — Select a CLI and pick a working directory with a native folder browser; remembers your last-used directory
- **CLI argument passing** — Pass extra arguments to CLIs with `--` separator syntax (e.g., `agenthub new claude ~/dir -- --arg1`); arguments are remembered per CLI
- **Terminal theming** — 138 curated color themes (WCAG-audited for readability across all 4 CLIs); select in Settings > Appearance, applies live to all open sessions including OpenCode (via SIGUSR2 broadcast), persists across restarts with localStorage fallback guard for removed themes
- **Terminal padding** — 8px inset around terminal content with dynamic background matching the active theme
- **Per-tab font size** — Zoom in/out per terminal with `Shift+=`/`Shift+-` (range 6–32px)
- **Tab management** — Rename tabs by double-clicking or right-click context menu
- **Session auto-close** — When an agent process exits, its tab shows a 5-second countdown with an inline banner and fixed-position toast notification; tab auto-closes after countdown unless "Keep Open" is clicked; non-zero exits skip auto-close and show error state; toggle auto-close behavior in Settings > Session Behavior
- **Live status indicators** — Colored dots per tab: running (green), waiting (yellow), idle (gray), errored (red)
- **Standard app menus** — File, Edit, Window, Help menus with keyboard shortcuts; Cmd+C/V clipboard in terminal tabs
- **Welcome tab** — Branded splash screen with version info, platform-specific installation instructions, and getting-started guide

### Plugin Suite (v3.2)
- **8 curated xterm.js addons, vendored same-origin** — `addon-webgl`, `addon-unicode11`, `addon-search`, `addon-web-links`, `addon-image`, `addon-serialize`, `addon-clipboard`, `addon-progress`. All bundled under `web/vendor/xterm/addons/` with a CI-enforced version-parity gate (`vendor_drift_test.go`); no runtime CDN fetches; strict CSP-compatible
- **Settings → Plugins** — 8 enable/disable toggles in the order: WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress. Three plugins (Search, Web Links, Inline Images) include inline `<details>` disclosures for per-plugin sub-config (regex/case/word defaults; modifier-key + risk policy; image storage limit). Non-hot-swappable plugins (Unicode 11, Inline Images) show "Applies to new sessions you create" inline plus a one-shot toast after Save
- **WebGL renderer** — GPU-accelerated rendering with automatic DOM fallback on context loss (banner toast with 8-second auto-dismiss) and proactive software-rasterizer detection (SwiftShader / llvmpipe / ANGLE-software / iPad Safari fall back at startup)
- **Scrollback search (Cmd-F)** — Focus-conditioned find bar on desktop + web with regex / case-sensitive / whole-word toggles, persisted defaults, next-match (Enter / Cmd-G), previous-match (Shift-Enter / Cmd-Shift-G), match count display, 200ms slide animation matching the BannerStack vocabulary, theme-aware highlight, and a 10,000-line scrollback perf budget
- **Web Links** — Clickable URLs with a strict scheme allowlist (`https`, `http`, `mailto` only — never `file://`, `javascript:`, or custom protocols). Cmd-click activation on macOS / Ctrl-click on Linux + Windows by default; single-click never activates a link. OSC 8 hyperlink href is shown in the hover tooltip. A risk-aware confirmation popover catches IDN/Punycode spoofs and a 30-entry typosquat list before navigation. Desktop routes through Wails `BrowserOpenURL`; web opens in a new tab with `_blank` + `noopener,noreferrer` (current-tab navigation is never possible)
- **Inline images (sixel)** — Render sixel-encoded images directly in the terminal via `@xterm/addon-image`. CSP includes a minimal, audited `'wasm-unsafe-eval'` carve-out (the addon's sixel decoder uses inline WASM). `storageLimit` defaults to 16 MB per-tab (overrides the upstream 100 MB default to prevent OOM with many concurrent tabs); configurable in the Settings disclosure. Multi-client byte-fidelity is preserved across the relay
- **Unicode 11 width tables** — Correct wide-character and emoji handling. Buffer-interpretation plugin: applies to new sessions only (toggling on a live session would re-flow existing scrollback); web-served clients share the daemon-side setting so multi-client scrollback wrap stays identical across viewers
- **Save Terminal As…** — Right-click a tab to save its current scrollback as a `.txt` file via a native `SaveFileDialog`. Backed by `@xterm/addon-serialize`; explicit secrets-warning tooltip; no auto-save / no on-disk capture without an explicit gesture
- **OSC 52 clipboard** — System clipboard read/write driven by OSC 52 escape sequences emitted by the running CLI. On web-served sessions, write access honors the capability bound to the session token — read-only viewers cannot have OSC 52 writes affect their clipboard via the terminal channel
- **OSC 9;4 progress (default OFF)** — Terminals emitting OSC 9;4 progress sequences (long-running task percent) surface as a subtle per-tab progress underline plus an aggregate tray quartile glyph (debounced at 200ms, atomic). Optional in v3.2; defaults flip ON in a future release after field validation
- **Daemon source of truth + hot-swap** — Plugin state lives in the daemon's `settings.json` under `PluginSettings` (with `schemaVersion: 2` migration from v3.1). Desktop receives a Wails `settings:plugins` runtime event for hot-swappable plugins; web clients subscribe to a capability-gated `/api/plugin-config` REST + SSE stream. No app restart needed
- **Cross-browser e2e CSP audit** — Playwright runs Chromium + Firefox + WebKit with zero CSP violations enforced in CI (`.github/workflows/e2e.yml`)

### Multi-Client Sessions
- **Simultaneous connections** — Multiple WebSocket clients can connect to the same session and receive live output simultaneously
- **Independent scrollback** — Each connected client maintains its own scrollback position without affecting other viewers
- **Read-only mode** — Attach with `--readonly` flag to observe a session without sending input (`agenthub attach --readonly <id>`)
- **Viewer count** — Session metadata API and CLI `agenthub list` show the current viewer count per session
- **Client identity** — Clients can provide a name at connection (e.g., `agenthub attach --client=macbook <id>`)
- **Resize arbitration** — Max-wins strategy: PTY dimensions stabilize to the largest active client, preventing resize thrashing

### TUI Mode
- **Full-screen terminal UI** — `agenthub tui` launches an interactive Bubble Tea v2 interface as an alternative to the desktop GUI
- **Two-pane layout** — Left sidebar (Home, Sessions, Remote, Settings) mirrors GUI navigation; right content pane shows the active tab with bordered frames and section headers
- **Focus-aware navigation** — Tab key toggles between sidebar and content panes; Up/Down navigates sidebar items; Enter opens a tab; [ and ] cycle through open tabs
- **Session list** — Sessions displayed inside bordered lipgloss frames with labeled titles; each row shows a colored per-agent badge (6 CLIs with distinct TokyoNight-derived colors), status glyph, hostname, and viewer count
- **TokyoNight color palette** — 22+ adaptive color tokens using lipgloss LightDark for consistent styling across light and dark terminals; matches GUI theme
- **Attach** — Press Enter on a session to suspend TUI and enter raw PTY attach with status bar; Ctrl-\ detaches and resumes TUI
- **Create session** — Press `n` to open a modal with agent picker (Left/Right cycling), directory input, and argument field
- **Kill session** — Press `d` for a confirmation dialog with danger-styled overlay; default-No for safety
- **Rename session** — Press `r` for inline edit; Enter commits, Esc cancels
- **Remote sessions** — Unified local+remote session list with tailnet peer grouping and hostname divider rows
- **QR code overlay** — Press `q` to display an ASCII QR code for the selected session's web URL
- **Web server status** — Footer shows whether the web server is running and its URL
- **Help overlay** — Press `?` to see all keybindings for the current view (includes Tab, [/] navigation)
- **Auto-refresh** — Session list refreshes every 2 seconds with selection preserved by identity

### Remote Sessions
- **Tailscale peer discovery** — Automatically discovers AgentHub instances running on other machines in your tailnet
- **Remote Sessions panel** — GUI tab showing sessions grouped by peer hostname with loading states and 30-second auto-refresh
- **CLI remote list** — `agenthub list` shows local and remote sessions grouped by HOST column
- **CLI remote attach** — `agenthub attach hostname:session-id` connects to remote sessions via WSS relay over Tailscale HTTPS
- **TUI remote list** — `agenthub tui` shows remote sessions in a unified list with local sessions, grouped by peer hostname
- **One-click open** — Click any remote session to open it in your browser

### Auto-Update
- **Update checker** — Polls GitHub releases on startup and hourly for new versions
- **Notification banner** — In-app banner when an update is available with one-click download; multiple banners stack vertically with independent dismiss controls
- **Help menu trigger** — Manual check via Help > Check for Updates

### System Tray
- **Cross-platform tray** — macOS (native cgo NSStatusBar), Linux (D-Bus StatusNotifierItem for GNOME/KDE/XFCE), Windows (Shell_NotifyIcon Win32 API) — all sharing menu construction via common helpers
- **Menu bar icon** — Monochrome template icon adapts to light/dark mode (macOS); embedded PNG icon (Linux/Windows)
- **Session menu** — Dynamic menu listing all active sessions; click to activate
- **Session count tooltip** — Shows active session count (e.g., "AgentHub — 3 sessions")
- **Error state** — Tray icon switches to error state when daemon is unreachable
- **Quit confirmation** — Closing the window or selecting Quit from the tray menu presents a modal showing active session count with colored status dots and three options: Keep Running (dismiss), Quit GUI Only (hide window, send macOS notification, daemon stays running), or Quit Everything (stop daemon and exit)
- **Start minimized** — Optional "Start minimized to system tray" toggle in Settings > Behavior; when enabled, the app launches hidden with only the tray icon visible — preference persists across restarts
- **Dock hiding** — App hides from Dock and Cmd+Tab via LSUIElement (macOS)

### Daemon Management Panel
- **In-GUI session control** — "Sessions" tab showing all active sessions with status dots, CLI type, and hostname badges
- **Per-session actions** — Kill sessions and toggle web serving directly from the panel
- **Live polling** — Session list refreshes automatically every 3 seconds
- **Hostname identification** — Each session displays the machine hostname for multi-machine visibility

### CLI
- **Full CLI** — `agenthub new`, `list`, `kill`, `rename`, `attach`, `web`, `health`, `qr`, `settings`
- **Interactive attach** — `agenthub attach <id>` for full PTY proxy with raw I/O, resize propagation, Ctrl-C passthrough, scrollback replay, and configurable detach key (default Ctrl-\\, set with `--detach-key=`)
- **Status bar** — Persistent tmux-style bottom bar during attach showing session name, agent type, hostname, detach hint, elapsed time, and live viewer count; refreshes without corrupting output (DECSTBM scroll region); suppressed when stdout is not a TTY; `--status-top` flag for top placement; clean teardown on detach
- **Connection banner** — Attach displays session name, CLI type, and hostname before entering raw mode
- **Machine-readable output** — `--json` flag on list, web status, health, and daemon status commands
- **Daemon management** — `agenthub daemon install/uninstall/start/stop` registers with platform service managers (launchd, systemd, Windows SCM)

### Web Serving
- **Auto-serve** — Web server starts automatically on daemon launch; new sessions are web-served by default
- **Dual-mode networking** — Tailscale mode (Let's Encrypt TLS, zero-config security) when available; local network fallback (self-signed TLS + HTTP Basic Auth with generated password) when Tailscale is unavailable. Automatically upgrades from local to Tailscale mode when Tailscale connects after startup
- **Per-session toggle** — Enable/disable web access per session from GUI or CLI (`agenthub serve/unserve`)
- **Web dashboard** — Dark-themed dashboard with session cards, live status dots, CLI badges, QR code thumbnails, and direct connect links
- **Web terminal status bar** — Live session info with name, CLI type, hostname, and three-state connection indicator (connecting/connected/disconnected)
- **QR codes** — Every web-served session gets a scannable QR code in the desktop app and CLI
- **Health checks** — 4-state Tailscale health cascade (binary found → daemon running → connected → certs ready) across Homebrew, system package managers, Snap, Flatpak, and Windows default paths (`agenthub health` CLI command)
- **Nudge banner** — Context-aware in-app banner with 4-state detection: recommends Tailscale installation when binary not found; shows daemon-stopped instructions (platform-specific) when binary exists but daemon isn't running; shows "upgrading to Tailscale..." when Tailscale connects and the server is restarting; each banner is independently dismissible with fade-out animation

### Security
- **Capability-based session authorization** — Session listing, metadata, and WebSocket access require server-issued, HMAC-signed capability tokens bound to a specific session ID. Tailnet membership is no longer sufficient on its own; explicit grant is required to share each session. Cross-session capability use is rejected with 403.
- **Server-bound read-only enforcement** — Read-only is a property of the capability claims, not a client-supplied query string. Clients reconnecting without `?readonly` cannot bypass; the relay rejects `MsgInput` from read-only subscribers regardless of how the connection was opened.
- **No auto-expose** — Creating a new session while the web server is running does not automatically expose it. Sessions become reachable only after explicit grant via the GUI/CLI; the daemon then returns a signed capability-bearing URL.
- **Signing-key rotation panic button** — "Regenerate Signing Key" in Settings invalidates every outstanding capability across all sessions in one click; useful if a share link is suspected leaked.
- **Strict WebSocket Origin allowlist** — Cross-site WebSocket hijacking is blocked at the handshake. Allowlist covers Tailscale FQDN (`<host>.<tailnet>.ts.net:<port>`), local-mode bind host (`<lan-ip>:<port>`), and the Wails desktop webview (production `wails://wails`, dev `wails://wails.localhost:<port>`, Windows `http://wails.localhost`). All other origins return 403.
- **Vendored terminal assets + Content-Security-Policy** — xterm.js, addons, and themes are embedded in the binary at `web/vendor/xterm/`. Zero runtime CDN fetches. All three embedded HTML routes (`/dashboard`, `/join`, `/sessions/{id}`) enforce a strict CSP: `script-src 'self'`, `connect-src 'self' wss://<host>`, `style-src 'self' 'unsafe-inline'` (the last only to accommodate xterm's runtime style injection — see Phase 89 D-09).
- **Signed + notarized releases with SLSA L2 build provenance** — All third-party GitHub Actions are 40-character SHA-pinned (`scripts/grep-gate.sh` enforces this in CI). Build-tool versions (Wails, nfpm) pinned via `tools.go`. The release pipeline is split into `validate` → `build-{macos,windows,linux}` (no secrets) → `sign-macos` (gated by a required-reviewer rule, holds notarization credentials) → `publish`. Sigstore build-provenance attestation is verified BEFORE codesigning runs, so a compromised build job cannot inject a malicious binary into the signing flow.
- **Reproducibility** — `release.yml` and `distribute.yml` produce SHA256 checksums alongside artifacts; every published release includes a `checksums.txt`. Dependabot is configured for both `gomod` and `github-actions` ecosystems with no auto-merge — every dependency change goes through manual review.

### Settings
- **Settings as sidebar tab** — Persistent Settings tab accessible from the sidebar (not a modal), consistent with Home/Remote/Sessions panels
- **Single scrollable page** — All settings on one page organized by section headers (Plugins, Appearance, Web Server, Paths, Behavior, Session Behavior) with visual dividers — no sub-tabs
- **Plugins section** — 8 enable/disable toggles for the v3.2 plugin suite (WebGL, Unicode 11, Search, Web Links, Inline Images, Serialize, Clipboard, Progress) with per-plugin descriptions, "Applies to new sessions you create" inline captions for non-hot-swappable plugins, one-shot toast after Save, and inline `<details>` disclosures exposing per-plugin sub-config for Search (regex/case/word defaults), Web Links (modifier-key + risk-confirmation policy), and Inline Images (storage limit). Sub-key RPCs persist disclosure changes immediately — no "Save Plugins" required for sub-config
- **Appearance section** — Theme selector with 138 curated color schemes; selected theme applies live to all terminals and persists in localStorage
- **Web Server section** — Start/stop web server with mode-aware status display; URL actions (open in browser, copy to clipboard, inline QR code); local network password with click-to-copy
- **Behavior section** — "Start minimized to system tray" toggle with non-optimistic save, loading state, and error feedback
- **Session Behavior section** — "Auto-close tab on exit" toggle controls whether session tabs auto-close when the agent process exits; preference persists via daemon settings
- **Paths section** — Override auto-detected CLI paths per agent; each path has a native browse button that opens a file picker; save confirmation shows a green "Saved!" indicator for 1.5 seconds
- **Tailscale status indicator** — 4-state color-coded dot (Connected / Not Connected / Daemon Stopped / Not Installed) with collapsible diagnostics checklist showing binary detection, daemon status, connection state, and TLS readiness; platform-specific troubleshooting instructions for macOS, Linux, and Windows
- **Certificate Transparency disclosure** — Acknowledgment flow for CT log requirements

### Platform
- **Cross-platform** — macOS (universal, signed + notarized), Linux (Ubuntu 22.04 + 24.04), Windows (NSIS installer)
- **Single binary** — `agenthub` launches GUI; `agenthub <command>` runs CLI
- **Custom app icons** — Branded AgentHub logomark across all platforms
- **Build script** — `build.sh` for local cross-platform builds with optional macOS code signing

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         Clients                               │
│  ┌──────────────┐  ┌──────────────────┐  ┌────────────────┐ │
│  │  GUI (Wails)  │  │ CLI (agenthub    │  │ TUI (agenthub  │ │
│  │  React+xterm  │  │ <cmd>)           │  │ tui)           │ │
│  │               │  │ attach/list/new  │  │ Bubble Tea v2  │ │
│  └──────┬───────┘  └────────┬─────────┘  └──────┬─────────┘ │
│         │       DaemonClient │                    │           │
│         └────────────┬───────┴────────────────────┘           │
├──────────────────────┼────────────────────────────────────────┤
│              Unix Socket / Named Pipe                         │
├──────────────────────┼────────────────────────────────────────┤
│  ┌───────────────────┴────────────────────────────────────┐  │
│  │                Daemon (background process)              │  │
│  │  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │  │
│  │  │ Session  │  │ WebSocket │  │    Web Server        │  │  │
│  │  │ Engine   │  │ Relay Hub │  │  (Tailscale or      │  │  │
│  │  │ (go-pty) │  │ (fan-out, │  │   Local TLS)        │  │  │
│  │  │          │  │ multi-    │  │                     │  │  │
│  │  │          │  │ client)   │  │                     │  │  │
│  │  └──────────┘  └──────────┘  └─────────────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐  ┌─────────────────────┐  │  │
│  │  │  Status  │  │ QR Code  │  │   Service Manager   │  │  │
│  │  │ Detector │  │ Generator│  │                     │  │  │
│  │  └──────────┘  └──────────┘  └─────────────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐                            │  │
│  │  │ Tailnet  │  │ Update   │                            │  │
│  │  │ Peers    │  │ Checker  │                            │  │
│  │  └──────────┘  └──────────┘                            │  │
│  └────────────────────────────────────────────────────────┘  │
│                      HTTP/JSON API                            │
└──────────────────────────────────────────────────────────────┘
```

**Go packages:**

| Package | Purpose |
|---------|---------|
| `internal/daemon` | Session engine, HTTP/JSON API, Unix socket server, DaemonClient |
| `internal/pty` | PTY process management, CLI detection |
| `internal/relay` | Binary framing protocol, scrollback buffer, WebSocket fan-out hub with multi-client support, per-subscriber metadata, read-only enforcement, max-wins resize arbitration |
| `internal/status` | Heuristic status detection (running/waiting/idle/errored) |
| `internal/statusbar` | DECSTBM scroll-region status bar for CLI attach with rune-safe formatting, viewer count, connection state, terminal injection prevention |
| `internal/attach` | Shared attach logic for CLI and TUI — ANSI-safe border-title injection, allowlist attach-status guard, error-propagating AttachSession |
| `internal/tui` | Bubble Tea v2 terminal UI — session list, modals (create/kill/rename), QR overlay, remote sessions, help overlay, adaptive colors |
| `internal/tailnet` | Tailscale peer discovery, concurrent probe pool, cached peer list |
| `internal/updater` | GitHub release polling, semantic version comparison, update notifications |
| `internal/webserver` | HTTPS server (Tailscale or local self-signed TLS), dashboard, health checks, Basic Auth |
| `web/` | Embedded HTML assets (dashboard + terminal pages) |

**Frontend (`frontend/`):**

| Component | Purpose |
|-----------|---------|
| `App.tsx` | Root layout, daemon client, session management, sidebar + content flex layout |
| `Sidebar.tsx` | Collapsible navigation sidebar with Heroicons: Home, Remote, Sessions, New Session, Settings |
| `TabBar.tsx` | Tab strip with status dots, rename, close (session tabs only — no action buttons) |
| `TerminalPanel.tsx` | xterm.js terminal with WebSocket relay, per-tab font size, theme support |
| `NewSessionModal.tsx` | CLI selector, working directory picker, argument input |
| `DaemonManagerPanel.tsx` | Session list with kill, web toggle, hostname badges |
| `RemoteSessionsPanel.tsx` | Tailscale peer sessions with auto-refresh and browser open |
| `WelcomeTab.tsx` | Branded welcome screen with installation instructions |
| `StatusBar.tsx` | Per-tab web-serving controls |
| `ExitToast.tsx` | Fixed-position toast notification for session exits — clean/error variants, countdown display, Keep Open and dismiss buttons |
| `ExitCountdownBanner.tsx` | Inline countdown banner in terminal area — "Agent exited cleanly. Tab closes in Ns." with Keep Open button |
| `QuitConfirmModal.tsx` | Quit confirmation modal — session list with colored status dots, three exit options (Keep Running, Quit GUI Only, Quit Everything) |
| `SettingsTab.tsx` | Settings as sidebar tab: single scrollable page with section headers (Behavior, Session Behavior, Appearance, Web Server, Paths); start-minimized toggle, auto-close toggle, theme selector, web server controls with URL actions (open/copy/QR), CLI path overrides with native browse buttons, local network password |
| `LocalNetworkBanner.tsx` | 4-state context-aware nudge banner with independent dismiss: not-installed, daemon-stopped, not-connected, and upgrade-in-progress states with platform-specific instructions |
| `UpdateBanner.tsx` | Standalone update notification banner with version info, download button, and dismiss control |
| `QRModal.tsx` | QR code display for web-served sessions |

## Installation

### macOS (Homebrew)

```bash
brew tap scottkw/agenthub
brew install --cask agenthub
```

### Windows (WinGet)

```powershell
winget install scottkw.agenthub
```

> WinGet availability depends on the first submission being accepted by Microsoft. Check [Releases](https://github.com/scottkw/agenthub/releases) for direct download in the meantime.

### GitHub Releases

Download the latest release for your platform from [Releases](https://github.com/scottkw/agenthub/releases):

| Platform | File | Notes |
|----------|------|-------|
| macOS (universal) | `agenthub-v*-darwin-universal.dmg` | Signed and notarized |
| Windows | `agenthub-v*-windows-amd64-installer.exe` | NSIS installer |
| Windows | `agenthub-v*-windows-amd64.exe` | Standalone executable |
| Linux (deb) | `agenthub-v*-linux-amd64.deb` | Ubuntu/Debian package |
| Linux (tar.gz) | `agenthub-v*-linux-amd64.tar.gz` | Generic archive |

All releases include a `checksums.txt` with SHA256 hashes for verification.

## Prerequisites

- **Go** 1.22+ ([go.dev/dl](https://go.dev/dl/))
- **Node.js** 18+ and **pnpm** ([pnpm.io](https://pnpm.io/installation))
- **Wails CLI** v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Tailscale** (optional) — enables zero-config web serving with browser-trusted TLS; without it, local network mode uses self-signed TLS + password auth ([tailscale.com](https://tailscale.com))

### Platform-specific

**macOS:**
- Xcode Command Line Tools: `xcode-select --install`

**Linux (Ubuntu/Debian):**
```bash
# Ubuntu 24.04 / Debian 13+
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev

# Ubuntu 22.04
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev
```

**Windows:**
- [WebView2 runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (included in Windows 11; install manually on Windows 10)
- A C compiler — MSYS2 with MinGW-w64 or TDM-GCC

## Development

```bash
# Clone
git clone https://github.com/scottkw/agenthub.git
cd agenthub

# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Run in dev mode (hot-reload frontend, live Go rebuild)
wails dev
```

Dev mode opens the desktop app with Vite HMR for the frontend and automatic Go rebuild on save.

### Running tests

```bash
# Go tests (with race detector)
go test -race ./...

# Frontend tests
cd frontend && pnpm test
```

## Building

### Using `build.sh` (recommended)

```bash
# Build for the current platform
./build.sh --platform macos    # macOS universal binary (.app)
./build.sh --platform linux    # Linux amd64 via Docker
./build.sh --platform windows  # Windows amd64 via cross-compile

# Build all platforms
./build.sh --all

# Build + sign and notarize macOS (requires Apple Developer credentials)
./build.sh --platform macos --sign
```

### Manual builds

#### Local build (current platform)

```bash
wails build -tags wailsassets
```

Output: `build/bin/agenthub` (or `agenthub.exe` on Windows, `agenthub.app` on macOS)

#### macOS (universal binary)

```bash
wails build -platform darwin/universal -tags wailsassets
```

#### Linux

```bash
# Ubuntu 24.04 (WebKitGTK 4.1)
wails build -tags webkit2_41,wailsassets

# Ubuntu 22.04 (WebKitGTK 4.0)
wails build -tags wailsassets
```

#### Windows

```bash
# Standard build
wails build -tags wailsassets

# With NSIS installer
wails build -nsis -tags wailsassets
```

### CI/CD

GitHub Actions automates building, releasing, and distributing AgentHub:

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `build.yml` | Push/PR | 4-runner matrix build with race detector (no signing) |
| `release-please.yml` | Push to main | Auto-creates Release PRs with CHANGELOG and version bump |
| `release.yml` | Tag push (v*) | Multi-platform release builds with macOS signing/notarization |
| `distribute.yml` | Release published | Updates Homebrew tap + submits WinGet manifest PR |

**Build matrix:**

| Runner | Platform | Notes |
|--------|----------|-------|
| `macos-latest` | `darwin/universal` | Signing + notarization in release.yml |
| `ubuntu-latest` | `linux/amd64` | WebKitGTK 4.1 (`-tags webkit2_41`) |
| `ubuntu-22.04` | `linux/amd64` | WebKitGTK 4.0 |
| `windows-latest` | `windows/amd64` | NSIS installer + WebView2 embedded |

## Usage

### Desktop (GUI)

1. **Launch AgentHub** — run `agenthub` with no arguments to open the GUI
2. **Navigate via sidebar** — use the collapsible left sidebar for all navigation; toggle collapsed/expanded with the hamburger icon
3. **Create a session** — click New Session in the sidebar to open the new session modal; select a CLI, working directory, and optional arguments
4. **Use the terminal** — full interactive terminal with the selected CLI; new sessions are automatically web-served
5. **Manage sessions** — click Sessions in the sidebar to view all sessions, kill them, or toggle web access
6. **Remote sessions** — click Remote in the sidebar to discover and open sessions on other tailnet machines
7. **Web serve** — web server starts automatically; toggle web access per session as needed
8. **System tray** — close the window to hide; use the tray menu to switch sessions or quit

### CLI

```bash
# Session management
agenthub new claude-code ~/project           # Create a new session
agenthub new claude-code ~/project -- --arg  # Pass extra CLI arguments
agenthub list                                # List all sessions
agenthub list --json                         # Machine-readable output
agenthub attach <id>                         # Attach to session (Ctrl-\ to detach)
agenthub attach hostname:<id>                # Attach to remote session via Tailscale
agenthub kill <id>                           # Terminate a session
agenthub rename <id> "my session"            # Rename a session
agenthub attach --readonly <id>              # Read-only attach (observe without input)
agenthub attach --client=macbook <id>        # Attach with client identity name

# TUI mode
agenthub tui                                 # Launch full-screen terminal UI

# Web serving
agenthub web start                    # Start the Tailscale web server
agenthub web stop                     # Stop the web server
agenthub web status                   # Check web server state
agenthub serve <id>                   # Enable web access for a session
agenthub unserve <id>                 # Disable web access
agenthub health                       # Tailscale health check
agenthub qr <id>                      # Show session QR code in terminal

# Daemon management
agenthub daemon install               # Register as login service
agenthub daemon uninstall             # Remove service registration
agenthub daemon start                 # Start the daemon service
agenthub daemon stop                  # Stop the daemon service
agenthub daemon status                # Check daemon status

# Configuration
agenthub settings                     # Show current settings
```

### Status indicators

Each tab shows a colored status dot:

| Color | Status | Meaning |
|-------|--------|---------|
| Green | Running | CLI is actively producing output |
| Yellow | Waiting | CLI has shown a prompt and is waiting for input |
| Gray | Idle | No recent output activity |
| Red | Errored | CLI process has exited with a non-zero code |

Status detection uses heuristic output patterns for **Claude Code**. Other CLIs show "running" until their patterns are catalogued.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop framework | [Wails v2](https://wails.io) |
| Backend | Go 1.22+ |
| Frontend | React 19, TypeScript, Vite |
| Terminal | [xterm.js](https://xtermjs.org) v6 + vendored addons (`addon-webgl`, `addon-unicode11`, `addon-search`, `addon-web-links`, `addon-image`, `addon-serialize`, `addon-clipboard`, `addon-progress`) |
| Terminal themes | [xterm-theme](https://www.npmjs.com/package/xterm-theme) — 138 curated schemes (WCAG-audited from 157 candidates) |
| PTY | [go-pty](https://github.com/aymanbagabas/go-pty) (cross-platform) |
| WebSocket | [nhooyr/websocket](https://github.com/coder/websocket) |
| TUI framework | [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) + [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) + [Bubbles v2](https://github.com/charmbracelet/bubbles) |
| QR codes | [go-qrcode](https://github.com/skip2/go-qrcode) |
| TLS | Tailscale Let's Encrypt via `GetCertificate`; self-signed P256 for local network mode |
| Peer discovery | [tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) |
| Auto-update | [go-selfupdate](https://github.com/creativeprojects/go-selfupdate), [Masterminds/semver](https://github.com/Masterminds/semver) |
| Service manager | [kardianos/service](https://github.com/kardianos/service) |
| Cross-browser e2e | [Playwright](https://playwright.dev) (Chromium + Firefox + WebKit) — CSP zero-violation gate in CI |
| CI | GitHub Actions (4-runner matrix + Playwright e2e) |

## License

See [LICENSE](LICENSE) for details.
