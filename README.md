# AgentHub

<p align="center">
  <img src="docs/agenthub-title-logo.png" alt="AgentHub" width="400">
</p>

A cross-platform desktop app and CLI for running AI coding CLIs — Claude Code, Codex, Gemini CLI, OpenCode — in persistent terminal sessions managed by a background daemon. Sessions survive GUI restarts, are controllable from the terminal, and can be shared over the web via Tailscale with browser-trusted TLS — or over the local network with self-signed TLS and password auth when Tailscale isn't available. The web server starts automatically and new sessions are web-served by default. A collapsible sidebar with Heroicons provides quick access to all navigation — Home, Remote Sessions, Daemon Manager, New Session, and Settings (as a full sidebar tab). Remote sessions on other tailnet machines are discoverable from both the GUI and CLI. Auto-update notifications keep you on the latest release. Built with Go/Wails and React.

## Features

### Terminal & Sessions
- **Collapsible sidebar** — Left sidebar with Heroicons SVG icons for all navigation: Home, Remote, Sessions, New Session (top); Settings (bottom). Toggle between collapsed (icons only, 48px) and expanded (icons + labels, 200px) via hamburger button; state persists in localStorage
- **Tabbed terminals** — Run multiple AI coding sessions side-by-side with full xterm.js terminals (ANSI 256-color, Unicode, emoji, 10K+ line scrollback, full-width viewport fill)
- **Background daemon** — Sessions live in a standalone daemon process; closing the GUI hides the window while sessions and the system tray remain active
- **CLI auto-detection** — Detects Claude Code, Codex, Gemini CLI, and OpenCode on startup — including when launched from Finder/Dock (augments PATH with `~/.local/bin`, Homebrew, nvm, volta, and other common install locations); supports custom CLI path overrides
- **New session modal** — Select a CLI and pick a working directory with a native folder browser; remembers your last-used directory
- **CLI argument passing** — Pass extra arguments to CLIs with `--` separator syntax (e.g., `agenthub new claude ~/dir -- --arg1`); arguments are remembered per CLI
- **Per-tab font size** — Zoom in/out per terminal with `Shift+=`/`Shift+-` (range 6–32px)
- **Tab management** — Rename tabs by double-clicking or right-click context menu
- **Live status indicators** — Colored dots per tab: running (green), waiting (yellow), idle (gray), errored (red)
- **Standard app menus** — File, Edit, Window, Help menus with keyboard shortcuts; Cmd+C/V clipboard in terminal tabs
- **Welcome tab** — Branded splash screen with version info, platform-specific installation instructions, and getting-started guide

### Remote Sessions
- **Tailscale peer discovery** — Automatically discovers AgentHub instances running on other machines in your tailnet
- **Remote Sessions panel** — GUI tab showing sessions grouped by peer hostname with loading states and 30-second auto-refresh
- **CLI remote list** — `agenthub list` shows local and remote sessions grouped by HOST column
- **CLI remote attach** — `agenthub attach hostname:session-id` connects to remote sessions via WSS relay over Tailscale HTTPS
- **One-click open** — Click any remote session to open it in your browser

### Auto-Update
- **Update checker** — Polls GitHub releases on startup and hourly for new versions
- **Notification banner** — In-app banner when an update is available with one-click download
- **Help menu trigger** — Manual check via Help > Check for Updates

### System Tray (macOS)
- **Menu bar icon** — Monochrome template icon adapts to light/dark mode
- **Session menu** — Dynamic menu listing all active sessions; click to activate
- **Session count tooltip** — Shows active session count (e.g., "AgentHub — 3 sessions")
- **Error state** — Tray icon switches to error state when daemon is unreachable
- **Hide-on-close** — Closing the window hides the GUI; quit from tray to fully exit
- **Dock hiding** — App hides from Dock and Cmd+Tab via LSUIElement

### Daemon Management Panel
- **In-GUI session control** — "Sessions" tab showing all active sessions with status dots, CLI type, and hostname badges
- **Per-session actions** — Kill sessions and toggle web serving directly from the panel
- **Live polling** — Session list refreshes automatically every 3 seconds
- **Hostname identification** — Each session displays the machine hostname for multi-machine visibility

### CLI
- **Full CLI** — `agenthub new`, `list`, `kill`, `rename`, `attach`, `web`, `health`, `qr`, `settings`
- **Interactive attach** — `agenthub attach <id>` for full PTY proxy with raw I/O, resize propagation, Ctrl-C passthrough, scrollback replay, and configurable detach key (default Ctrl-\\, set with `--detach-key=`)
- **Connection banner** — Attach displays session name, CLI type, and hostname before entering raw mode
- **Machine-readable output** — `--json` flag on list, web status, health, and daemon status commands
- **Daemon management** — `agenthub daemon install/uninstall/start/stop` registers with platform service managers (launchd, systemd, Windows SCM)

### Web Serving
- **Auto-serve** — Web server starts automatically on daemon launch; new sessions are web-served by default
- **Dual-mode networking** — Tailscale mode (Let's Encrypt TLS, zero-config security) when available; local network fallback (self-signed TLS + HTTP Basic Auth with generated password) when Tailscale is unavailable
- **Per-session toggle** — Enable/disable web access per session from GUI or CLI (`agenthub serve/unserve`)
- **Web dashboard** — Dark-themed dashboard with session cards, live status dots, CLI badges, QR code thumbnails, and direct connect links
- **Web terminal status bar** — Live session info with name, CLI type, hostname, and three-state connection indicator (connecting/connected/disconnected)
- **QR codes** — Every web-served session gets a scannable QR code in the desktop app and CLI
- **Health checks** — Detects Tailscale installation, connection, and cert readiness with platform-specific setup guidance
- **Nudge banner** — Persistent in-app banner recommending Tailscale installation when running in local network mode

### Tailscale Onboarding
- **Guided setup** — Platform-specific install commands with copy-to-clipboard buttons and download links
- **macOS auto-install** — One-click Tailscale installation via Homebrew directly from the health modal
- **Post-install guide** — Step-by-step HTTPS certificate configuration after Tailscale is installed

### Settings
- **Settings as sidebar tab** — Persistent Settings tab accessible from the sidebar (not a modal), consistent with Home/Remote/Sessions panels
- **Custom CLI paths** — Override auto-detected paths per CLI
- **Web server controls** — Start/stop web server with mode-aware status display
- **Local network password** — View the generated password with click-to-copy when running in local network mode
- **Tailscale health display** — Color-coded status indicators with platform-specific setup instructions
- **Certificate Transparency disclosure** — Acknowledgment flow for CT log requirements

### Platform
- **Cross-platform** — macOS (universal, signed + notarized), Linux (Ubuntu 22.04 + 24.04), Windows (NSIS installer)
- **Single binary** — `agenthub` launches GUI; `agenthub <command>` runs CLI
- **Custom app icons** — Branded AgentHub logomark across all platforms
- **Build script** — `build.sh` for local cross-platform builds with optional macOS code signing

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                     Clients                             │
│  ┌──────────────────┐    ┌───────────────────────────┐ │
│  │   GUI (Wails)     │    │   CLI (agenthub <cmd>)    │ │
│  │   React + xterm.js│    │   attach / list / new ... │ │
│  └────────┬─────────┘    └──────────┬────────────────┘ │
│           │     DaemonClient         │                  │
│           └──────────┬───────────────┘                  │
├──────────────────────┼──────────────────────────────────┤
│              Unix Socket / Named Pipe                   │
├──────────────────────┼──────────────────────────────────┤
│  ┌───────────────────┴──────────────────────────────┐  │
│  │              Daemon (background process)           │  │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │  │
│  │  │ Session  │  │ WebSocket │  │  Web Server   │  │  │
│  │  │ Engine   │  │ Relay Hub │  │ (Tailscale or │  │  │
│  │  │ (go-pty) │  │ (fan-out) │  │  Local TLS)  │  │  │
│  │  └──────────┘  └──────────┘  └───────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │  │
│  │  │  Status  │  │ QR Code  │  │   Service     │  │  │
│  │  │ Detector │  │ Generator│  │   Manager     │  │  │
│  │  └──────────┘  └──────────┘  └───────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐                     │  │
│  │  │ Tailnet  │  │ Update   │                     │  │
│  │  │ Peers    │  │ Checker  │                     │  │
│  │  └──────────┘  └──────────┘                     │  │
│  └──────────────────────────────────────────────────┘  │
│                   HTTP/JSON API                         │
└────────────────────────────────────────────────────────┘
```

**Go packages:**

| Package | Purpose |
|---------|---------|
| `internal/daemon` | Session engine, HTTP/JSON API, Unix socket server, DaemonClient |
| `internal/pty` | PTY process management, CLI detection |
| `internal/relay` | Binary framing protocol, scrollback buffer, WebSocket fan-out hub |
| `internal/status` | Heuristic status detection (running/waiting/idle/errored) |
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
| `TerminalPanel.tsx` | xterm.js terminal with WebSocket relay, per-tab font size |
| `NewSessionModal.tsx` | CLI selector, working directory picker, argument input |
| `DaemonManagerPanel.tsx` | Session list with kill, web toggle, hostname badges |
| `RemoteSessionsPanel.tsx` | Tailscale peer sessions with auto-refresh and browser open |
| `WelcomeTab.tsx` | Branded welcome screen with installation instructions |
| `StatusBar.tsx` | Per-tab web-serving controls |
| `SettingsTab.tsx` | Settings as sidebar tab: CLI paths, web server controls, local network password |
| `LocalNetworkBanner.tsx` | Persistent nudge banner recommending Tailscale when in local network mode |
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
- **Tailscale** — required for web serving features ([tailscale.com](https://tailscale.com))

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
| Terminal | [xterm.js](https://xtermjs.org) v6 |
| PTY | [go-pty](https://github.com/aymanbagabas/go-pty) (cross-platform) |
| WebSocket | [nhooyr/websocket](https://github.com/coder/websocket) |
| QR codes | [go-qrcode](https://github.com/skip2/go-qrcode) |
| TLS | Tailscale Let's Encrypt via `GetCertificate`; self-signed P256 for local network mode |
| Peer discovery | [tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) |
| Auto-update | [go-selfupdate](https://github.com/creativeprojects/go-selfupdate), [Masterminds/semver](https://github.com/Masterminds/semver) |
| Service manager | [kardianos/service](https://github.com/kardianos/service) |
| CI | GitHub Actions (4-runner matrix) |

## License

See [LICENSE](LICENSE) for details.
