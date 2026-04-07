# Architecture Research

**Domain:** Remote Session Access, Auto-Update, App Polish — AgentHub v1.9 (Wails v2 Desktop App)
**Researched:** 2026-04-06
**Confidence:** HIGH — existing codebase inspected directly; Tailscale local.Client API and Wails menu API verified via official pkg.go.dev docs

---

## System Overview

### Existing Architecture (v1.8 baseline)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  agenthub binary (single executable per platform)                        │
│                                                                           │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────────────┐ │
│  │  GUI (Wails v2)  │  │  CLI commands    │  │  daemon subcommand     │ │
│  │  App{} thin shell│  │  runCLI()        │  │  RunDaemon()           │ │
│  │  DaemonClient    │  │  DaemonClient    │  │                        │ │
│  └────────┬─────────┘  └────────┬─────────┘  └────────────────────────┘ │
│           │ Unix socket          │ Unix socket                           │
│           └──────────────────────┘                                       │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │  daemon (internal/daemon) — independent OS process                 │  │
│  │                                                                     │  │
│  │  SessionEngine  ←→  API (HTTP/Unix socket)  ←→  DaemonClient      │  │
│  │       │                                                             │  │
│  │  relay.HubManager  ←→  relay.Server (TCP, internal)               │  │
│  │       │                                                             │  │
│  │  webserver.WebServer (HTTPS, Tailscale IP, Let's Encrypt TLS)      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  System tray (NSStatusBar cgo on macOS; stubs on Linux/Windows)   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                    │ TLS (100.x.x.x Tailscale IP, :7443)
                    ▼
           ┌──────────────────┐
           │  Remote browser  │  (any tailnet device)
           │  Web dashboard   │
           └──────────────────┘
```

### v1.9 Target Architecture (new components highlighted)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  agenthub binary                                                          │
│                                                                           │
│  ┌──────────────────┐  ┌──────────────────┐                             │
│  │  GUI (Wails v2)  │  │  CLI commands    │                             │
│  │  + App Menu ★    │  │  + remote ★      │                             │
│  │  + RemotePanel ★ │  │  + update ★      │                             │
│  │  + UpdateChecker ★│  │                  │                             │
│  └────────┬─────────┘  └──────────────────┘                             │
│           │ Unix socket                                                   │
│           ▼                                                               │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  daemon (internal/daemon)                                        │    │
│  │                                                                   │    │
│  │  + GET /tailnet/peers ★   → PeerDiscovery (internal/tailnet ★)  │    │
│  │  + GET /tailnet/sessions ★  ← HTTP probe to peer's AgentHub     │    │
│  │                                                                   │    │
│  │  existing: SessionEngine, relay.Server, webserver.WebServer      │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  internal/updater ★   — GitHub release checker + asset download  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                           │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  internal/tailscale ★  — install detection, platform-specific    │   │
│  │                          install instructions & auto-install      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

★ = new in v1.9

---

## Component Responsibilities

### Existing Components (Unchanged Interfaces)

| Component | Package | Responsibility |
|-----------|---------|----------------|
| `App{}` | root | Thin Wails shell; all session ops delegated via DaemonClient |
| `DaemonClient` | `internal/daemon` | HTTP client over Unix socket; typed Go methods |
| `API` | `internal/daemon` | HTTP/JSON server on Unix socket; routes to SessionEngine |
| `SessionEngine` | `internal/daemon` | Session lifecycle, PTY management, status tracking |
| `relay.Server` | `internal/relay` | WebSocket relay for terminal I/O (local TCP, 127.0.0.1) |
| `webserver.WebServer` | `internal/webserver` | HTTPS dashboard on Tailscale IP with Let's Encrypt TLS |
| `CheckHealth()` | `internal/webserver` | Tailscale health check via `local.Client{}` |

### New Components (v1.9)

| Component | Package | Responsibility |
|-----------|---------|----------------|
| `PeerDiscovery` | `internal/tailnet` | Query `local.Client{}.Status()` for peers; probe each peer for AgentHub |
| `PeerSessionList` | `internal/tailnet` | HTTP GET to remote peer's web server `/api/sessions` endpoint |
| `Updater` | `internal/updater` | Check GitHub releases API for newer version; download asset; apply |
| App Menu | root `main.go` | Wails `options.App.Menu` with AppMenu + EditMenu + File + Window + Help |
| `RemoteSessionsPanel` | `frontend/src/components` | React panel showing discovered remote peers and their sessions |
| `UpdateNotification` | `frontend/src/components` | React banner/modal for available update, download progress |

---

## Recommended Project Structure (New Files)

```
internal/
├── tailnet/                     # NEW — tailnet peer discovery
│   ├── discovery.go             # PeerDiscovery: Status() → filter AgentHub peers
│   ├── discovery_test.go
│   ├── probe.go                 # HTTP probe to /api/sessions on remote web server
│   └── probe_test.go
│
├── updater/                     # NEW — auto-update
│   ├── updater.go               # CheckLatest(), Download(), Apply()
│   └── updater_test.go
│
└── daemon/
    ├── api.go                   # MODIFY — add GET /tailnet/peers, GET /tailnet/sessions
    ├── client.go                # MODIFY — add GetTailnetPeers(), GetTailnetSessions()
    └── types.go                 # MODIFY — add PeerInfo, RemoteSessionInfo types

frontend/src/components/
├── RemoteSessionsPanel.tsx      # NEW — remote peer discovery UI
├── UpdateBanner.tsx             # NEW — update available notification
└── __tests__/
    ├── RemoteSessionsPanel.test.tsx
    └── UpdateBanner.test.tsx
```

---

## Feature Architecture: Remote Session Discovery

### Data Model

```go
// internal/tailnet/discovery.go

// PeerInfo represents a tailnet peer running AgentHub.
type PeerInfo struct {
    Hostname   string   // HostName from PeerStatus
    TailscaleIP string  // First TailscaleIPs entry
    DNSName    string   // FQDN from DNSName (trim trailing dot)
    Online     bool     // PeerStatus.Online
    AgentHubURL string  // Discovered HTTPS URL if AgentHub web server running
}

// RemoteSessionInfo is a session on a remote peer.
type RemoteSessionInfo struct {
    PeerHostname string
    PeerURL      string
    ID           string
    Name         string
    CLIType      string
    Status       string
    Hostname     string  // same as PeerHostname, from session metadata
}
```

### Discovery Flow

```
GUI polls GetTailnetPeers() every 30s (or on-demand button)
    │
    ▼
DaemonClient.GetTailnetPeers()
    │ HTTP GET /tailnet/peers (over Unix socket)
    ▼
API.handleTailnetPeers()
    │
    ▼
internal/tailnet.DiscoverPeers(ctx)
    │
    ├── local.Client{}.Status(ctx)    ← existing Tailscale daemon call
    │         returns *ipnstate.Status with Peer map
    │
    ├── filter: peer.Online == true
    │           peer is not self (compare against status.Self.HostName)
    │
    └── for each online peer:
            ProbePeer(ctx, peer.TailscaleIPs[0])
                │ HTTP GET https://<tailscale-ip>:<port>/api/sessions
                │ (try known ports: 7443, 443, 8443)
                │ TLS: skip cert verify for IP-addressed probe (cert is for FQDN)
                ├── success → peer is running AgentHub, has sessions
                └── timeout/error → peer not running AgentHub (skip)
```

**Key insight:** The existing web server's `/api/sessions` endpoint (already serving for the web dashboard) is the probe target. No new server-side protocol is needed. Remote discovery reuses the existing web serving infrastructure.

**TLS probe note:** When probing by Tailscale IP (100.x.x.x), the Let's Encrypt cert will not match the IP address — it's issued to the FQDN. Use `tls.Config{InsecureSkipVerify: true}` with a short 3s timeout for the probe. This is acceptable because: (1) we're probing only tailnet IPs (network-layer authenticated), (2) we only want to know if AgentHub is running, not establish a secure session yet.

### Attach to Remote Session (GUI)

Remote session attach opens the existing web terminal URL in the user's browser:

```
User clicks "Open" on remote session in RemoteSessionsPanel
    │
    ▼
Wails runtime.BrowserOpenURL(remoteSession.AgentHubURL + "/terminal/" + sessionID)
    │
    ▼
Browser opens → wss://peer-hostname.ts.net:7443/sessions/<id>/ws
    (uses existing web dashboard terminal — no new protocol)
```

**Rationale:** Browser-based attach reuses the entire existing web serving + xterm.js web terminal stack. No new WebSocket relay protocol needed. The remote peer's web server handles TLS (its own Let's Encrypt cert) and serves the terminal to the browser.

### Attach to Remote Session (CLI)

`agenthub remote list` and `agenthub remote attach <peer> <id>`:

```
agenthub remote attach hostname session-id
    │
    ├── Discover peer's AgentHub URL (same probe as GUI)
    ├── GET https://peer-fqdn:7443/api/sessions/<id>  (verify exists)
    └── Open ws relay: connect to peer's relay WebSocket
            wss://peer-fqdn:7443/sessions/<id>/ws
            Proxy stdin/stdout (same as local cmdAttach, but remote WebSocket source)
```

**CLI attach to remote uses the same existing WebSocket protocol** — the relay binary framing (MsgInput, MsgResize2, MsgOutput) already handles remote clients (this is what the web browser does). The CLI just becomes another WebSocket client.

### New Daemon API Routes

```
GET /tailnet/peers
    → []PeerInfo (all online tailnet peers, AgentHub probe results included)
    → 200 OK; empty array if Tailscale not running

GET /tailnet/sessions
    → []RemoteSessionInfo (all sessions across all AgentHub peers)
    → Aggregates probe results from all responding peers
    → 200 OK; empty array if no peers found
```

These are pure read operations. The daemon calls `internal/tailnet.DiscoverPeers()` which calls `local.Client{}.Status()` — same zero-value client pattern already used in `webserver.CheckHealth()`.

---

## Feature Architecture: Auto-Update

### Library Choice

**Use `creativeprojects/go-selfupdate`** (not `rhysd/go-github-selfupdate`).

Rationale:
- `creativeprojects/go-selfupdate` supports GitHub + has active maintenance (last commit 2025)
- Has checksum validation support (`ChecksumValidator` with `checksums.txt`) — AgentHub already publishes `checksums.txt` in releases (v1.8 Phase 46)
- `rhysd/go-github-selfupdate` is less actively maintained and lacks checksum validation

**This is a checker + downloader, not an in-process patcher.** For a macOS `.app` bundle, in-process binary replacement is unreliable (the running binary is inside an `.app` bundle, the bundle needs to be replaced atomically). The correct pattern for a GUI app:

```
Check → notify user → download to temp file → open Finder to downloaded location
```

Not: check → replace own binary → restart. That pattern fails on macOS due to Gatekeeper and bundle signing.

### Update Check Flow

```
App startup (app.startup) or user-initiated (Help > Check for Updates)
    │
    ▼
internal/updater.CheckLatest(ctx, "scottkw/agenthub", currentVersion)
    │ calls GitHub releases API: GET https://api.github.com/repos/scottkw/agenthub/releases/latest
    ├── latest.Version > currentVersion → returns UpdateInfo{Version, URL, ReleaseNotes}
    └── up to date → returns nil
    │
    ▼
Wails: runtime.EventsEmit(ctx, "update:available", UpdateInfo{...})
    │
    ▼
Frontend: UpdateBanner shows version + release notes + "Download" button
    │
    ▼
User clicks Download
    │
    ▼
App.DownloadUpdate(url) — Wails-bound method
    ├── Downloads to ~/Downloads/agenthub-vX.Y.Z-darwin-universal.zip
    ├── Emits "update:progress" events during download
    └── On complete: runtime.BrowserOpenURL(downloadPath) → opens Finder
        (or on Windows: exec.Command("explorer", downloadPath))
        (on Linux: exec.Command("xdg-open", downloadPath))
```

### Version Constant

`var Version string` already planned in main.go (v1.8 Phase 45 ldflags pattern). The updater reads this. In dev builds where `Version` is empty, skip the update check (or treat as "0.0.0-dev").

### New Daemon API Routes for Updater

The updater runs in the GUI process (App{}), not the daemon. Reason: the daemon is a long-running service that may run as a system service; update checks should be triggered by the user-facing GUI. No new daemon routes needed for the updater.

---

## Feature Architecture: Tailscale Install Assistance

### Current State

`internal/webserver/tailscale.go` already has `CheckHealth()` which returns `TailscaleHealth{Installed, Connected, HasCerts, IP, Domain}`. The `HealthModal.tsx` component displays platform-specific instructions.

### v1.9 Addition: Auto-Install

For macOS and Linux, offer a one-click install:

```go
// internal/tailscale/install.go (new, or add to webserver/tailscale.go)

func InstallTailscale(ctx context.Context) error {
    switch runtime.GOOS {
    case "darwin":
        // Option 1: Homebrew (if available)
        if _, err := exec.LookPath("brew"); err == nil {
            return exec.CommandContext(ctx, "brew", "install", "--cask", "tailscale").Run()
        }
        // Option 2: Open download page
        return openBrowser("https://tailscale.com/download/mac")
    case "linux":
        // Tailscale install script (curl | sh pattern — prompt user first)
        return openBrowser("https://tailscale.com/download/linux")
    case "windows":
        // Open download page (MSI installer)
        return openBrowser("https://tailscale.com/download/windows")
    }
}
```

**For auto-install via Homebrew on macOS:** This is viable — `brew install --cask tailscale` works silently. But it requires prompting the user first (security: running a command that installs software without consent is bad UX). Pattern: confirm dialog → execute brew install → show progress → re-run CheckHealth on completion.

**For Linux/Windows:** Direct auto-install is not safe to do without elevated permissions and user confirmation. Open the browser to the Tailscale download page — same as current behavior.

### Wails-bound Methods (additions to App{})

```go
func (a *App) InstallTailscale() error        // triggers install per platform
func (a *App) GetTailscaleInstallMethod() string  // returns "homebrew", "pkg", "browser"
```

---

## Feature Architecture: Standard App Menus

### Wails v2 Menu API

Wails v2 provides native application menus via `options.App.Menu`:

```go
// main.go — in runGUI(), set Menu field in options.App:

import (
    "github.com/wailsapp/wails/v2/pkg/menu"
    "github.com/wailsapp/wails/v2/pkg/menu/keys"
)

func buildAppMenu(app *App) *menu.Menu {
    appMenu := menu.NewMenu()

    // macOS: AppMenu (About, Services, Hide, Quit) — required for macOS conventions
    appMenu.Append(menu.AppMenu())

    // File menu
    fileMenu := appMenu.AddSubmenu("File")
    fileMenu.AddText("New Session", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
        runtime.EventsEmit(app.ctx, "menu:new-session")
    })
    fileMenu.AddSeparator()
    fileMenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
        runtime.WindowHide(app.ctx)
    })

    // Edit menu — macOS requires this for Cmd+C/V/Z to work in text fields
    appMenu.Append(menu.EditMenu())

    // Window menu
    appMenu.Append(menu.WindowMenu())  // Minimize, Zoom, etc.

    // Help menu
    helpMenu := appMenu.AddSubmenu("Help")
    helpMenu.AddText("Check for Updates...", nil, func(_ *menu.CallbackData) {
        runtime.EventsEmit(app.ctx, "menu:check-updates")
    })
    helpMenu.AddText("AgentHub on GitHub", nil, func(_ *menu.CallbackData) {
        runtime.BrowserOpenURL(app.ctx, "https://github.com/scottkw/agenthub")
    })

    return appMenu
}
```

Then in `runGUI()`:

```go
app := NewApp()
err := wails.Run(&options.App{
    // ... existing fields ...
    Menu: buildAppMenu(app),
})
```

### Menu Callbacks and Frontend Events

Menu callbacks use `runtime.EventsEmit` to send events to the React frontend. The frontend subscribes with `EventsOn`. This is the existing pattern for tray events (`tray:focus-session`) — apply the same pattern for menu events.

New frontend events to handle:
- `menu:new-session` → open NewSessionModal
- `menu:check-updates` → trigger update check and show UpdateBanner

### Platform Notes

**macOS:** `menu.AppMenu()` and `menu.EditMenu()` are mandatory. Without `AppMenu()`, the app has no About item and Cmd+Q is broken. Without `EditMenu()`, Cmd+C/V/Z do not work in any text input inside the WebView. This is a Wails-specific requirement verified in the Wails menu package docs.

**Windows/Linux:** The menu renders as a native application menu bar. `AppMenu()` is macOS-only (it renders nothing on other platforms — Wails handles this gracefully).

**LSUIElement conflict:** The app currently hides from the Dock using `LSUIElement = 1`. On macOS, `LSUIElement` apps conventionally do not have a menu bar. However, Wails renders the menu bar regardless of `LSUIElement`. This is acceptable for AgentHub — the menu bar is useful when the window is visible. No code change needed.

---

## Data Flow

### Remote Discovery Flow (GUI)

```
[RemoteSessionsPanel mounts or user clicks Refresh]
    │
    ▼
GetTailnetPeers() → Wails binding → App.GetTailnetPeers()
    │
    ▼
DaemonClient.GetTailnetPeers() → HTTP GET /tailnet/peers (Unix socket)
    │
    ▼
API.handleTailnetPeers() → tailnet.DiscoverPeers(ctx)
    │
    ├── local.Client{}.Status(ctx) → peer map (hostname, IPs, online)
    │
    └── for each online peer:
            tailnet.ProbePeer(ctx, ip, knownPorts)
                HTTP GET https://<ip>:<port>/api/sessions
                3s timeout, InsecureSkipVerify (IP-addressed probe)
                → success: PeerInfo{AgentHubURL, sessions}
    │
    ▼
[]PeerInfo serialized as JSON → DaemonClient → App → Wails → React state
    │
    ▼
RemoteSessionsPanel renders: peer cards with session lists
    │
    ▼
User clicks "Open Session"
    │
    ▼
runtime.BrowserOpenURL(peerURL + "/terminal/" + sessionID)
```

### Update Check Flow

```
App.startup() goroutine
    │
    ▼
updater.CheckLatest(ctx, "scottkw/agenthub", Version)
    (non-blocking, 5s timeout)
    │
    ├── no update → done
    └── update available
            │
            ▼
        runtime.EventsEmit(ctx, "update:available", UpdateInfo{
            Version: "v1.9.0",
            ReleaseNotesURL: "https://github.com/scottkw/agenthub/releases/tag/v1.9.0",
            DownloadURL: "https://github.com/scottkw/agenthub/releases/download/v1.9.0/agenthub-darwin-universal.zip",
        })
            │
            ▼
        Frontend: UpdateBanner appears at top of window
        User clicks "Download" → App.DownloadUpdate(url)
            │
            ▼
        Download to ~/Downloads/ with progress events
            │
            ▼
        Open Finder/Explorer to downloaded file
```

---

## Integration Points

### New vs. Modified Components

| Component | Action | What Changes |
|-----------|--------|-------------|
| `internal/daemon/api.go` | **MODIFY** | Add `GET /tailnet/peers`, `GET /tailnet/sessions` route handlers |
| `internal/daemon/client.go` | **MODIFY** | Add `GetTailnetPeers()`, `GetTailnetSessions()` typed client methods |
| `internal/daemon/types.go` | **MODIFY** | Add `PeerInfo`, `RemoteSessionInfo` types |
| `app.go` | **MODIFY** | Add `GetTailnetPeers()`, `GetTailnetSessions()`, `CheckForUpdate()`, `DownloadUpdate()`, `InstallTailscale()` Wails-bound methods |
| `main.go` (runGUI) | **MODIFY** | Add `Menu: buildAppMenu(app)` to `options.App{}` |
| `tray_objc.m` / `tray.go` | **NO CHANGE** | Tray menus remain NSMenuDelegate-based; standard app menus are separate |
| `internal/webserver/tailscale.go` | **MODIFY** | Add `InstallTailscale()` or extend health check |
| `frontend/src/App.tsx` | **MODIFY** | Add `EventsOn` handlers for `menu:*` and `update:*` events; add RemoteSessionsPanel tab |
| `frontend/src/components/DaemonManagerPanel.tsx` | **MODIFY** | May absorb RemoteSessionsPanel, or RemoteSessionsPanel is a separate tab |

### New Files

| File | Purpose |
|------|---------|
| `internal/tailnet/discovery.go` | PeerDiscovery using local.Client{}.Status() |
| `internal/tailnet/discovery_test.go` | Tests with injectable statusFunc |
| `internal/tailnet/probe.go` | HTTP probe to remote AgentHub web server |
| `internal/tailnet/probe_test.go` | Tests with httptest server |
| `internal/updater/updater.go` | GitHub releases check + download |
| `internal/updater/updater_test.go` | Tests with mock GitHub API |
| `frontend/src/components/RemoteSessionsPanel.tsx` | Remote peers + sessions UI |
| `frontend/src/components/UpdateBanner.tsx` | Update notification UI |

### Daemon IPC: New Routes

| Route | Method | Handler | Returns |
|-------|--------|---------|---------|
| `/tailnet/peers` | GET | `handleTailnetPeers` | `[]PeerInfo` |
| `/tailnet/sessions` | GET | `handleTailnetSessions` | `[]RemoteSessionInfo` |

These routes trigger active network probing. Each call to `GET /tailnet/peers` does:
1. One `local.Client{}.Status()` call (fast, local socket)
2. One HTTP probe per online peer (concurrent, 3s timeout each)

**Caching:** The daemon should cache peer discovery results for 30 seconds. Concurrent GUI polls and CLI calls within the window return cached results without re-probing. Implement as a simple `sync.Mutex`-protected struct with a `lastProbed time.Time` and `cachedPeers []PeerInfo`.

---

## Build Order (Phase Dependencies)

```
Phase A: Standard App Menus
  (pure Wails API, no new packages, no external deps)
  Modifies: main.go (buildAppMenu), app.go (menu event dispatch)
  Unblocks: nothing (self-contained)
      │
      ▼  (can start in parallel with Phase A)

Phase B: Version from Build + Welcome Screen Polish
  (prerequisite: var Version string in main.go, already planned in v1.8)
  Modifies: main.go, WelcomeTab.tsx, wails.json
  Unblocks: Phase D (UpdateChecker needs Version)

Phase C: internal/tailnet package (PeerDiscovery + Probe)
  (no UI deps; builds and tests independently)
  Creates: internal/tailnet/discovery.go, probe.go, tests
  Unblocks: Phase E (daemon routes need this package)

Phase D: internal/updater package + UpdateBanner frontend
  (needs: Phase B for Version constant)
  Creates: internal/updater/updater.go, UpdateBanner.tsx
  Unblocks: Phase F (app.go needs updater)

Phase E: Daemon API routes for tailnet (/tailnet/peers, /tailnet/sessions)
  (needs: Phase C for internal/tailnet)
  Modifies: internal/daemon/api.go, client.go, types.go
  Unblocks: Phase F (GUI needs daemon routes)

Phase F: GUI Remote Sessions Panel + Update Integration + Tailscale Install
  (needs: Phase B, D, E)
  Creates: RemoteSessionsPanel.tsx
  Modifies: App.tsx, app.go, HealthModal.tsx
```

**Critical path:** C → E → F. Phases A and B are independent and can ship first.

---

## Architectural Patterns

### Pattern 1: Injectable statusFunc for Tailnet Discovery (Mirrors Existing Health Check)

**What:** `internal/tailnet/discovery.go` uses the same injectable `statusFunc` pattern as `internal/webserver/tailscale.go`. The production code calls `local.Client{}.Status()` via an injected function; tests pass a fake.

**Why:** `local.Client{}.Status()` requires a live tailscaled socket. All tests must be runnable without Tailscale installed (CI on GitHub Actions). The injectable function pattern is already proven in `tailscale_test.go`.

```go
type statusFunc func(ctx context.Context) (*ipnstate.Status, error)

func discoverPeers(ctx context.Context, fn statusFunc) ([]PeerInfo, error) { ... }

func DiscoverPeers(ctx context.Context) ([]PeerInfo, error) {
    var lc local.Client
    return discoverPeers(ctx, lc.Status)
}
```

### Pattern 2: Reuse Existing Web Server as Remote Discovery Target

**What:** Remote peers are discovered by probing their existing `/api/sessions` HTTP endpoint (already part of `webserver.WebServer`). No new server-side protocol needed.

**Why:** The web server already serves sessions over HTTPS on the Tailscale IP. The endpoint is already authenticated by tailnet membership (same rationale as the web dashboard). Adding a new discovery protocol would require versioning, backward compat, and additional server code.

**Trade-off:** Probe uses `InsecureSkipVerify` because certs are issued to the FQDN, not the IP. This is acceptable because the Tailscale network provides the transport-layer authentication. The data in the probe response (session list) is not sensitive.

### Pattern 3: Wails Menu Events for Menu→Frontend Communication

**What:** Menu item callbacks call `runtime.EventsEmit(ctx, "menu:event-name")`. Frontend subscribes with `EventsOn("menu:event-name", handler)`. This is the same pattern as `tray:focus-session`.

**Why:** Wails menu callbacks run on the main OS thread. Attempting to call JavaScript or React state directly from a menu callback is not safe. Emitting a Wails event is the correct cross-thread communication mechanism.

### Pattern 4: GUI-Only Update Checker (Not Daemon)

**What:** The update checker (`internal/updater`) is called from `App{}` (GUI process), not from the daemon.

**Why:** The daemon may run as a system service (launchd/systemd) with no user session. Update notifications require a user-visible UI. Performing downloads in a system service context creates permission complications (where to write files, how to notify the user). Keeping the updater in the GUI process is simpler and correct.

---

## Anti-Patterns

### Anti-Pattern 1: In-Process Binary Self-Replace on macOS

**What people do:** Use `go-selfupdate`'s `UpdateTo()` to replace the running executable in-place.

**Why it's wrong for AgentHub:** The Wails binary is embedded inside an `.app` bundle. Replacing the binary inside a running `.app` while it is open is unreliable. macOS Gatekeeper may reject the replacement if the code signature changes. The daemon is a separate OS process that holds the socket — restarting just the GUI is not sufficient.

**Do this instead:** Download the update to `~/Downloads/`, open Finder to the location, and instruct the user to quit AgentHub, drag the new `.app` to Applications, and relaunch. This is the standard macOS update UX for non-Sparkle apps.

### Anti-Pattern 2: Probing Peers via their FQDN TLS Connection

**What people do:** Probe `https://peer-hostname.ts.net:7443/api/sessions` using the FQDN for discovery.

**Why it's wrong:** MagicDNS resolution may not work in all tailnet configurations (e.g., MagicDNS disabled, custom DNS). Probing by Tailscale IP (from `PeerStatus.TailscaleIPs[0]`) is more reliable.

**Do this instead:** Probe by IP with `InsecureSkipVerify`, then construct the FQDN-based URL (from `PeerStatus.DNSName`, stripping trailing dot) for the actual browser attach URL where TLS verification is needed.

### Anti-Pattern 3: Blocking GUI Startup on Peer Discovery

**What people do:** Run peer discovery synchronously during app startup.

**Why it's wrong:** Discovery requires a Tailscale daemon call + N concurrent HTTP probes with 3s timeouts each. With 10 tailnet peers, worst-case discovery takes 3 seconds. Blocking startup on this makes the GUI feel slow.

**Do this instead:** Run peer discovery in a goroutine. Update the RemoteSessionsPanel reactively when results arrive (use Wails events or React polling). Show a loading spinner while discovery runs.

### Anti-Pattern 4: Adding a New Discovery Protocol Instead of Reusing /api/sessions

**What people do:** Design a new UDP broadcast or mDNS protocol for AgentHub peer discovery.

**Why it's wrong:** Tailscale already provides peer enumeration via `local.Client{}.Status()`. No additional discovery protocol is needed. The existing `/api/sessions` HTTP endpoint already serves the data. A new protocol adds complexity, versioning concerns, and firewall port issues.

**Do this instead:** Use `local.Client{}.Status()` for peer enumeration and HTTP probe to existing `/api/sessions` for AgentHub detection.

---

## Scaling Considerations

| Scale | Consideration |
|-------|--------------|
| 1-5 tailnet peers | No issues; discovery is fast |
| 10-50 peers | Concurrent probing with 3s timeout keeps total under 5s; cache results for 30s |
| 50+ peers | Most large tailnets have non-AgentHub devices; filter by port responsiveness quickly; cache is essential |

Peer count is bounded by the tailnet size — typical developer tailnets have 5-20 devices. This is not a scaling concern in practice.

---

## Sources

- `internal/daemon/api.go` direct inspection — confirmed existing routes, route registration pattern, `writeJSON` helper (HIGH confidence)
- `internal/daemon/client.go` direct inspection — confirmed `doJSON` pattern, typed method signatures (HIGH confidence)
- `internal/daemon/types.go` direct inspection — confirmed type shapes for new types to follow (HIGH confidence)
- `internal/webserver/tailscale.go` direct inspection — confirmed injectable `statusFunc` pattern and `local.Client{}` zero-value usage (HIGH confidence)
- `internal/webserver/server.go` direct inspection — confirmed `/api/sessions` endpoint exists as probe target (HIGH confidence)
- `app.go` direct inspection — confirmed Wails binding pattern, `pollSessionStatus` goroutine pattern for async ops (HIGH confidence)
- `main.go` direct inspection — confirmed `options.App{}` structure, confirmed no `Menu` field currently set (HIGH confidence)
- [tailscale.com/ipn/ipnstate#PeerStatus](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — `PeerStatus.HostName`, `TailscaleIPs`, `DNSName`, `Online` fields confirmed (HIGH confidence)
- [tailscale.com/client/local#Client.Status](https://pkg.go.dev/tailscale.com/client/local) — `Status()` vs `StatusWithoutPeers()` semantics confirmed (HIGH confidence)
- [Wails menu package](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu) — `AppMenu()`, `EditMenu()`, `WindowMenu()` predefined menus confirmed; `AddSubmenu()`, `AddText()` API confirmed (HIGH confidence)
- [creativeprojects/go-selfupdate](https://pkg.go.dev/github.com/creativeprojects/go-selfupdate) — `DetectLatest()`, `UpdateTo()`, `ChecksumValidator` API confirmed; GitHub source provider confirmed (MEDIUM confidence — not yet in go.mod, API shape verified)

---

*Architecture research for: AgentHub v1.9 — Remote Sessions, Auto-Update, App Polish*
*Researched: 2026-04-06*
