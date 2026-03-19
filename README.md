# AgentHub

A cross-platform desktop app for running AI coding CLIs — Claude Code, Codex, Gemini CLI, OpenCode — in tabbed terminal sessions. Any session can be shared over the web via HTTPS with QR codes, token links, and live status indicators. Built with Go/Wails and React.

## Features

- **Tabbed terminals** — Run multiple AI coding sessions side-by-side with full xterm.js terminals (ANSI 256-color, Unicode, emoji, 10K+ line scrollback)
- **CLI auto-detection** — Scans PATH for Claude Code, Codex, Gemini CLI, and OpenCode on startup; supports custom CLI paths
- **Session persistence** — Close the window to the system tray; sessions keep running. Reopen and reattach instantly with full scrollback replay
- **Web serving** — Toggle any session to be accessible from a remote browser over HTTPS. Self-signed TLS with a local CA cert pattern so browsers trust the connection
- **Authentication** — Password-protected dashboard lists all shared sessions. Per-session token links grant access without the dashboard password
- **QR codes** — Every web-served session gets a scannable QR code in the desktop app and on the web dashboard
- **Live status indicators** — Each tab shows a colored dot: running (green), waiting for input (yellow), idle (gray), or errored (red). Status detection uses heuristic output parsing
- **VPN binding** — Bind the web server to a specific network interface. Auto-detects Tailscale via CGNAT range; supports any VPN interface
- **Cross-platform** — Builds for macOS (universal, signed + notarized), Linux (Ubuntu 22.04 + 24.04), and Windows (NSIS installer)

## Architecture

```
┌─────────────────────────────────────────────────┐
│                  Wails Desktop App               │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  TabBar   │  │ Terminal  │  │  Settings /   │  │
│  │  + Status │  │  Panel    │  │  QR Modal     │  │
│  │  Badges   │  │ (xterm.js)│  │               │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
│         React Frontend (Vite + TypeScript)        │
├───────────────────────────────────────────────────┤
│         Wails v2 Bridge (bound Go methods)        │
├───────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ PTY      │  │ WebSocket │  │  Web Server   │  │
│  │ Backend  │  │ Relay Hub │  │  (TLS + Auth) │  │
│  │ (go-pty) │  │ (fan-out) │  │               │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Session  │  │  Status   │  │  QR Code Gen  │  │
│  │ Registry │  │ Detector  │  │ (go-qrcode)   │  │
│  └──────────┘  └──────────┘  └───────────────┘  │
│              Go Backend (single binary)           │
└───────────────────────────────────────────────────┘
```

**Go packages:**

| Package | Purpose |
|---------|---------|
| `internal/pty` | PTY process management, CLI detection, session registry |
| `internal/relay` | Binary framing protocol, scrollback buffer, WebSocket fan-out hub |
| `internal/status` | Heuristic status detection (running/waiting/idle/errored) |
| `internal/webserver` | HTTPS server, TLS cert generation, auth, dashboard, token links |
| `web/` | Embedded HTML assets (dashboard + terminal pages) |

**Frontend (`frontend/`):**

| Component | Purpose |
|-----------|---------|
| `App.tsx` | Root layout, session management, event wiring |
| `TabBar.tsx` | Tab strip with status dots, rename, close |
| `TerminalPanel.tsx` | xterm.js terminal with WebSocket relay client |
| `SettingsPanel.tsx` | CLI paths, web serving controls, network interface selection |
| `QRModal.tsx` | QR code display modal for web-served sessions |

## Prerequisites

- **Go** 1.22+ ([go.dev/dl](https://go.dev/dl/))
- **Node.js** 18+ and **pnpm** ([pnpm.io](https://pnpm.io/installation))
- **Wails CLI** v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

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

### Local build (current platform)

```bash
wails build
```

Output: `build/bin/agenthub` (or `agenthub.exe` on Windows, `agenthub.app` on macOS)

### macOS (universal binary)

```bash
wails build -platform darwin/universal
```

**Signing and notarization** (requires Apple Developer account):

```bash
# Sign
codesign --force --deep --sign "Developer ID Application: YOUR NAME (TEAM_ID)" \
  --entitlements build/entitlements.plist \
  --options runtime \
  build/bin/agenthub.app

# Notarize
ditto -c -k --keepParent build/bin/agenthub.app notarization.zip
xcrun notarytool submit notarization.zip \
  --apple-id "your@email.com" \
  --password "app-specific-password" \
  --team-id "TEAM_ID" \
  --wait
xcrun stapler staple build/bin/agenthub.app
```

### Linux

```bash
# Ubuntu 24.04 (WebKitGTK 4.1)
wails build -tags webkit2_41

# Ubuntu 22.04 (WebKitGTK 4.0)
wails build
```

### Windows

```bash
# Standard build
wails build

# With NSIS installer
wails build -nsis
```

The NSIS build produces both `agenthub.exe` and `agenthub-amd64-installer.exe`.

### CI/CD

The GitHub Actions workflow (`.github/workflows/build.yml`) builds for all platforms automatically on push. It runs a 4-job matrix:

| Runner | Platform | Notes |
|--------|----------|-------|
| `macos-latest` | `darwin/universal` | Signing + notarization when secrets are configured |
| `ubuntu-latest` | `linux/amd64` | WebKitGTK 4.1 (`-tags webkit2_41`) |
| `ubuntu-22.04` | `linux/amd64` | WebKitGTK 4.0 |
| `windows-latest` | `windows/amd64` | NSIS installer + WebView2 embedded |

Build artifacts are uploaded as GitHub Actions artifacts.

**Required secrets for macOS signing** (optional — builds work without them):

| Secret | Purpose |
|--------|---------|
| `MACOS_CERTIFICATE` | Base64-encoded .p12 certificate |
| `MACOS_CERTIFICATE_NAME` | Certificate common name |
| `MACOS_CERTIFICATE_PWD` | Certificate password |
| `MACOS_CI_KEYCHAIN_PWD` | Ephemeral CI keychain password |
| `MACOS_NOTARIZATION_APPLE_ID` | Apple ID for notarization |
| `MACOS_NOTARIZATION_PWD` | App-specific password |
| `MACOS_NOTARIZATION_TEAM_ID` | Apple Developer Team ID |

## Usage

### First launch

1. **Launch AgentHub** — the app scans your PATH for installed AI coding CLIs
2. **Create a session** — click the `+` button and select a detected CLI (or configure custom CLI paths in Settings)
3. **Use the terminal** — full interactive terminal with the selected CLI. Resize, scroll, copy/paste all work as expected

### Managing sessions

- **Multiple tabs** — open as many sessions as you need; each runs independently
- **Rename tabs** — double-click a tab name to rename it
- **Close sessions** — click the `×` on a tab to kill the session and its process
- **System tray** — close the window and sessions keep running in the background. Click the tray icon to reopen

### Web serving

1. **Set a password** — go to Settings and set a web dashboard password (required before starting the web server)
2. **Start the web server** — click "Start Web Server" in Settings. Choose a network interface (auto-detects Tailscale)
3. **Enable per-session** — toggle the web icon on any tab to make that session accessible remotely
4. **Share access:**
   - **Dashboard URL** — share the HTTPS URL; recipients enter the dashboard password to see all shared sessions
   - **Token link** — generate a per-session token link that grants direct access without the dashboard password
   - **QR code** — scan from the desktop app or web dashboard to open on a phone/tablet

### TLS certificate trust

AgentHub generates a local CA certificate on first run. To avoid browser warnings:

- **macOS:** Open Keychain Access, import `~/.config/agenthub/ca.crt`, set to "Always Trust"
- **Linux:** `sudo cp ~/.config/agenthub/ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates`
- **Windows:** `certutil -addstore "Root" %USERPROFILE%\.config\agenthub\ca.crt`

The app provides in-app guidance for this process in the Settings panel.

### Status indicators

Each tab shows a colored status dot:

| Color | Status | Meaning |
|-------|--------|---------|
| Green | Running | CLI is actively producing output |
| Yellow | Waiting | CLI has shown a prompt and is waiting for input |
| Gray | Idle | No recent output activity |
| Red | Errored | CLI process has exited with a non-zero code |

Status detection currently uses heuristic output patterns for **Claude Code**. Other CLIs (Codex, Gemini CLI, OpenCode) will show "running" until their output patterns are catalogued in a future release.

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
| TLS | Go `crypto/tls` + `crypto/x509` |
| Auth | bcrypt password hashing + cookie sessions |
| CI | GitHub Actions (4-runner matrix) |

## License

See [LICENSE](LICENSE) for details.
