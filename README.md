# AgentHub

A cross-platform desktop app and CLI for running AI coding CLIs — Claude Code, Codex, Gemini CLI, OpenCode — in persistent terminal sessions managed by a background daemon. Sessions survive GUI restarts, are controllable from the terminal, and can be shared over the web via Tailscale with browser-trusted TLS. Built with Go/Wails and React.

## Features

### Terminal & Sessions
- **Tabbed terminals** — Run multiple AI coding sessions side-by-side with full xterm.js terminals (ANSI 256-color, Unicode, emoji, 10K+ line scrollback, full-width viewport fill)
- **Background daemon** — Sessions live in a standalone daemon process; closing the GUI doesn't kill sessions
- **CLI auto-detection** — Scans PATH for Claude Code, Codex, Gemini CLI, and OpenCode on startup; supports custom CLI paths
- **New session modal** — Select a CLI and pick a working directory; remembers your last-used directory
- **Per-tab font size** — Zoom in/out per terminal with `Shift+=`/`Shift+-`
- **Tab management** — Rename tabs by double-clicking or right-click context menu
- **Live status indicators** — Colored dots per tab: running (green), waiting (yellow), idle (gray), errored (red)

### CLI
- **Full CLI** — `agenthub new`, `list`, `kill`, `rename`, `attach`, `web`, `health`, `qr`, `settings`
- **Interactive attach** — `agenthub attach <id>` for full PTY proxy with raw I/O, resize propagation, Ctrl-C passthrough, scrollback replay, and detach key (Ctrl-\\)
- **Machine-readable output** — `--json` flag on list, web status, health, and daemon status commands
- **Daemon management** — `agenthub daemon install/uninstall/start/stop` registers with platform service managers (launchd, systemd, Windows SCM)

### Web Serving
- **Tailscale networking** — Web server binds exclusively to Tailscale interface with Let's Encrypt TLS via `tsnet`
- **Zero-config security** — Tailscale network membership is the access control; no passwords or tokens needed
- **Per-session toggle** — Enable/disable web access per session from GUI or CLI (`agenthub serve/unserve`)
- **Web dashboard** — Dark-themed dashboard with session cards, live status dots, CLI badges, and direct connect links
- **QR codes** — Every web-served session gets a scannable QR code in the desktop app and CLI
- **Health checks** — Detects Tailscale installation, connection, and cert readiness with platform-specific setup guidance

### Platform
- **Cross-platform** — macOS (universal, signed + notarized), Linux (Ubuntu 22.04 + 24.04), Windows (NSIS installer)
- **Single binary** — `agenthub` launches GUI; `agenthub <command>` runs CLI
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
│  │  │ Engine   │  │ Relay Hub │  │ (Tailscale    │  │  │
│  │  │ (go-pty) │  │ (fan-out) │  │  TLS + FQDN) │  │  │
│  │  └──────────┘  └──────────┘  └───────────────┘  │  │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │  │
│  │  │  Status  │  │ QR Code  │  │   Service     │  │  │
│  │  │ Detector │  │ Generator│  │   Manager     │  │  │
│  │  └──────────┘  └──────────┘  └───────────────┘  │  │
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
| `internal/webserver` | HTTPS server via Tailscale, dashboard, health checks |
| `web/` | Embedded HTML assets (dashboard + terminal pages) |

**Frontend (`frontend/`):**

| Component | Purpose |
|-----------|---------|
| `App.tsx` | Root layout, daemon client, session management, event wiring |
| `TabBar.tsx` | Tab strip with status dots, rename, close |
| `TerminalPanel.tsx` | xterm.js terminal with WebSocket relay, per-tab font size |
| `NewSessionModal.tsx` | CLI selector + working directory picker |
| `StatusBar.tsx` | Per-tab web-serving controls |
| `SettingsPanel.tsx` | Tabbed settings with Tailscale status |
| `HealthModal.tsx` | Tailscale health check with platform-specific instructions |
| `QRModal.tsx` | QR code display for web-served sessions |

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
git clone https://gitea.eightabyte.com/scottkw/agenthub.git
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

The GitHub Actions workflow (`.github/workflows/build.yml`) builds for all platforms automatically on push. It runs a 4-job matrix:

| Runner | Platform | Notes |
|--------|----------|-------|
| `macos-latest` | `darwin/universal` | Signing + notarization when secrets are configured |
| `ubuntu-latest` | `linux/amd64` | WebKitGTK 4.1 (`-tags webkit2_41`) |
| `ubuntu-22.04` | `linux/amd64` | WebKitGTK 4.0 |
| `windows-latest` | `windows/amd64` | NSIS installer + WebView2 embedded |

## Usage

### Desktop (GUI)

1. **Launch AgentHub** — run `agenthub` with no arguments to open the GUI
2. **Create a session** — click `+` to open the new session modal; select a CLI and working directory
3. **Use the terminal** — full interactive terminal with the selected CLI
4. **Web serve** — toggle web access per session; Tailscale health check runs automatically

### CLI

```bash
# Session management
agenthub new claude-code ~/project    # Create a new session
agenthub list                         # List all sessions
agenthub list --json                  # Machine-readable output
agenthub attach <id>                  # Attach to session (Ctrl-\ to detach)
agenthub kill <id>                    # Terminate a session
agenthub rename <id> "my session"     # Rename a session

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
| TLS | Tailscale Let's Encrypt via `GetCertificate` |
| Service manager | [kardianos/service](https://github.com/kardianos/service) |
| CI | GitHub Actions (4-runner matrix) |

## License

See [LICENSE](LICENSE) for details.
