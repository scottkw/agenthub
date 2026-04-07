# Stack Research

**Domain:** Remote Session Discovery, Auto-Update, Tailscale Install Assistance, App Menus — AgentHub v1.9
**Researched:** 2026-04-06
**Confidence:** HIGH (all libraries verified against pkg.go.dev and official docs as of research date)

---

## Scope

This file covers ONLY the new capabilities needed for v1.9. The existing stack (Go/Wails v2, React, xterm.js, nhooyr/websocket, go-pty, kardianos/service, skip2/go-qrcode, tailscale.com/client/local, native macOS cgo NSStatusBar) is validated and unchanged.

---

## New Dependencies Required

### Remote Session Discovery (Tailnet Peer Scanning)

**Verdict: No new dependencies.** `tailscale.com/client/local` — already in `go.mod` at `tailscale.com v1.96.3` — exposes everything needed.

The key API path:

```go
lc := &local.Client{}                          // already used for health checks
status, err := lc.Status(ctx)                  // returns *ipnstate.Status
for _, peer := range status.Peer {             // Peer is map[key.NodePublic]*ipnstate.PeerStatus
    // peer.HostName      — machine hostname
    // peer.DNSName       — FQDN with MagicDNS suffix (e.g. "machine.tail12345.ts.net.")
    // peer.TailscaleIPs  — []netip.Addr, use [0] for primary IPv4
    // peer.Online        — bool, connected to control plane
    // peer.Active        — bool, packet sent in past ~2 minutes
    // peer.OS            — operating system string
}
```

To probe whether a peer is running AgentHub, make an HTTP request to `http://<tailscale-ip>:<daemon-port>/health` or a new `/peers` endpoint on the daemon's Unix socket API. This is pure Go `net/http` — no new library needed.

**Integration:** The daemon's existing health check polling in `internal/daemon` is the right place to add a `ScanPeers()` function. Expose via the existing HTTP/JSON Unix socket protocol as a new endpoint. GUI consumes it like any other daemon endpoint.

---

### Update Checker — Version Check + Download Link

**Recommended: `github.com/creativeprojects/go-selfupdate` v1.5.2**

Use it in **version-check-only mode** (no in-place binary replacement on macOS). For a signed DMG-distributed app, in-place binary replacement breaks code signing. The right pattern for this app:

1. Call `DetectLatest()` to check current vs. latest version.
2. If newer: surface the new version + download URL in the UI.
3. User clicks "Download" → open the GitHub release page in browser (`pkg/browser` already in go.mod as an indirect dep via Wails).
4. User downloads the DMG and installs normally.

**Do NOT use `UpdateSelf()` or `UpdateTo()` on macOS.** Replacing a signed binary invalidates the signature. The Homebrew auto-update path (`brew upgrade --cask agenthub`) is the correct in-place update mechanism for Homebrew users.

```go
import "github.com/creativeprojects/go-selfupdate"

func CheckForUpdate(ctx context.Context, current string) (*selfupdate.Release, error) {
    latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("scottkw/agenthub"))
    if err != nil || !found {
        return nil, err
    }
    if latest.GreaterThan(current) {
        return latest, nil   // caller surfaces latest.Version() and latest.AssetURL in UI
    }
    return nil, nil  // already up to date
}
```

Asset naming must match what the action produces. go-selfupdate expects:
`agenthub_{darwin|linux|windows}_{amd64|arm64}{.tar.gz|.zip}` — which the existing release pipeline already produces for the CLI binary. The GUI DMG is separate; the update checker should point users to the GitHub releases page, not try to manage the DMG itself.

**Why not `google/go-github` (v84)?** go-github is a full GitHub API client (~50K LOC). For a single `GET /repos/scottkw/agenthub/releases/latest` call it's massive overkill. go-selfupdate wraps exactly this call with version comparison built in, is MIT-licensed, and is a negligible import size.

**Why not `rhysd/go-github-selfupdate`?** The creativeprojects fork is the actively maintained successor — supports Gitea, Gitlab, and HTTP sources, has universal binary support for macOS, and released v1.5.2 in December 2025. rhysd's original sees infrequent maintenance.

**Installation:**
```bash
go get github.com/creativeprojects/go-selfupdate@v1.5.2
```

---

### Tailscale Install Assistance

**Verdict: No new dependencies.** Platform detection and install command generation is pure Go stdlib.

Detection strategy per platform:

| Platform | Detection | Install Command to Surface |
|----------|-----------|--------------------------|
| macOS | `exec.LookPath("tailscale")` | `brew install --cask tailscale-app` |
| macOS (no brew) | `stat /Applications/Tailscale.app` fails | Direct URL: `https://tailscale.com/download/mac` |
| Linux | `exec.LookPath("tailscale")` | `curl -fsSL https://tailscale.com/install.sh \| sh` |
| Windows | `exec.LookPath("tailscale.exe")` | `winget install tailscale.tailscale` |

For **auto-install on macOS**, run `brew install --cask tailscale-app` via `exec.Command` if Homebrew is detected (`exec.LookPath("brew")`). Otherwise, open the download URL via `runtime.BrowserOpenURL(ctx, "https://tailscale.com/download")` — Wails runtime already handles this.

**No subprocess shell needed.** `exec.Command("brew", "install", "--cask", "tailscale-app")` works directly. Pipe stdout/stderr to the UI via a channel for progress display.

The existing `tailscale.com/client/local` `Status()` call already surfaces whether Tailscale is installed and connected — if `Status()` returns a connection error, Tailscale is either not installed or tailscaled is not running.

---

### Standard App Menus (File, Edit, Window, Help)

**Verdict: No new dependencies.** Wails v2 (`github.com/wailsapp/wails/v2 v2.10.2`, already in go.mod) includes a full menu system at `github.com/wailsapp/wails/v2/pkg/menu`.

**Go API:**

```go
import (
    "github.com/wailsapp/wails/v2/pkg/menu"
    "github.com/wailsapp/wails/v2/pkg/menu/keys"
    rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

func buildAppMenu(ctx context.Context) *menu.Menu {
    appMenu := menu.NewMenu()

    if runtime.GOOS == "darwin" {
        appMenu.Append(menu.AppMenu())   // macOS: About, Services, Hide, Quit
    }

    // File menu
    fileMenu := appMenu.AddSubmenu("File")
    fileMenu.AddText("New Session", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
        rt.EventsEmit(ctx, "menu:new-session")
    })
    fileMenu.AddSeparator()
    fileMenu.AddText("Close Window", keys.CmdOrCtrl("w"), func(_ *menu.CallbackData) {
        rt.WindowHide(ctx)
    })

    if runtime.GOOS == "darwin" {
        appMenu.Append(menu.EditMenu())  // macOS: Undo, Redo, Cut, Copy, Paste, Select All
    }

    // Window menu
    appMenu.Append(menu.WindowMenu())    // Minimize, Zoom (macOS); Minimize, Maximize (Win/Lin)

    // Help menu
    helpMenu := appMenu.AddSubmenu("Help")
    helpMenu.AddText("AgentHub on GitHub", nil, func(_ *menu.CallbackData) {
        rt.BrowserOpenURL(ctx, "https://github.com/scottkw/agenthub")
    })
    helpMenu.AddText("Check for Updates...", nil, func(_ *menu.CallbackData) {
        rt.EventsEmit(ctx, "menu:check-updates")
    })

    return appMenu
}
```

Set at startup via `options.App{Menu: buildAppMenu(ctx)}`. Dynamic update (e.g. after session list changes) via `rt.MenuUpdateApplicationMenu(ctx)`.

**Platform behavior:**
- **macOS**: Menu bar appears at top of screen. `AppMenu()` and `EditMenu()` are macOS-only helpers — include them only on `darwin`. `EditMenu()` is required to enable Cmd+C/V/Z in text inputs.
- **Windows/Linux**: Menu bar appears at top of the application window. `AppMenu()` is not applicable. `EditMenu()` provides standard Edit shortcuts inline.

**Critical:** Do NOT call `AppMenu()` or `EditMenu()` unconditionally — they are macOS-specific menu structures that will produce unexpected behavior or blank entries on Windows/Linux.

---

## Summary: What Needs to Be Added

| New Dependency | Version | For What | Notes |
|----------------|---------|----------|-------|
| `github.com/creativeprojects/go-selfupdate` | v1.5.2 | Update checker | One new direct dep |

Everything else — peer discovery, Tailscale install detection, app menus — uses libraries already in `go.mod`.

**No new frontend dependencies.** The GUI surfaces new backend data (peer list, update status, install progress) through the existing Wails event system and daemon IPC pattern.

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **`google/go-github`** | Full GitHub API client (~50K LOC) for a single releases check call | `go-selfupdate` which wraps exactly that call |
| **`rhysd/go-github-selfupdate`** | Predecessor — infrequent maintenance since 2022; creativeprojects fork is the active successor | `creativeprojects/go-selfupdate@v1.5.2` |
| **`go-selfupdate` UpdateSelf() / UpdateTo() on macOS** | Replaces signed binary in-place, breaking notarized code signature | Check version only; open release page in browser for DMG download |
| **`fyne.io/systray` or any third-party tray library** | Duplicate symbol linker error with Wails AppDelegate — confirmed blocker from v1.7 | Native macOS cgo NSStatusBar (already implemented) |
| **`tsnet` (tailscale.com/tsnet)** | Creates a second Tailscale node — wrong for reading the host machine's existing tailnet state | `tailscale.com/client/local` (already in go.mod) |
| **Tailscale API key / control plane API** | Requires OAuth or API token setup, user-visible credentials, calls cloud API | `local.Client{}.Status()` queries the local tailscaled daemon directly — no auth, works offline |
| **Shell scripts for Tailscale detection** | `os/exec` with a shell subproc is fragile and slow | `exec.LookPath("tailscale")` + `exec.Command("brew", ...)` directly |

---

## Integration Points with Existing Stack

| Feature | Integration Point | Pattern |
|---------|------------------|---------|
| Peer discovery | `internal/daemon` — new `ScanPeers()` func → new `/api/peers` endpoint on Unix socket | Same HTTP/JSON IPC as existing session endpoints |
| Update check | Background goroutine in daemon or GUI; expose via Wails event `update:available` | Non-blocking; check at startup + periodic (e.g. daily) |
| Tailscale install | GUI `HealthModal` or new `SetupModal`; backend detects, frontend shows platform-specific instructions + optional auto-install button | Extend existing health check system |
| App menus | `main.go` (Wails startup) — pass `Menu:` in `options.App`; menu callbacks emit Wails events | No daemon involvement; pure GUI layer |
| "Check for Updates" menu item | Wails event `menu:check-updates` → frontend calls `CheckForUpdate` binding → backend returns version info | Standard Wails binding call pattern |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `creativeprojects/go-selfupdate@v1.5.2` | `go 1.21+` | Published Dec 19, 2025. No known conflicts with existing deps. |
| `wailsapp/wails/v2 v2.10.2` `pkg/menu` | All platforms | `AppMenu()` / `EditMenu()` are macOS-only — guard with `runtime.GOOS == "darwin"` |
| `tailscale.com/client/local` (via `tailscale.com v1.96.3`) | tailscaled 1.x | `Status()` returns `*ipnstate.PeerStatus` per peer with `HostName`, `DNSName`, `TailscaleIPs`, `Online`, `Active`, `OS` |

---

## Sources

- [pkg.go.dev/github.com/creativeprojects/go-selfupdate](https://pkg.go.dev/github.com/creativeprojects/go-selfupdate) — v1.5.2 confirmed, Dec 19 2025, `DetectLatest` API verified. HIGH confidence.
- [pkg.go.dev/tailscale.com/client/local](https://pkg.go.dev/tailscale.com/client/local) — `Status()`, `WhoIs()` methods verified. HIGH confidence.
- [pkg.go.dev/tailscale.com/ipn/ipnstate](https://pkg.go.dev/tailscale.com/ipn/ipnstate) — `PeerStatus` fields: `HostName`, `DNSName`, `TailscaleIPs`, `Online`, `Active`, `OS`. HIGH confidence.
- [pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/menu) — `AppMenu()`, `EditMenu()`, `WindowMenu()` helpers; `AddText`, `AddSeparator`, `AddSubmenu` builder API. HIGH confidence.
- [wails.io/docs/reference/menus/](https://wails.io/docs/reference/menus/) — `MenuSetApplicationMenu`, `MenuUpdateApplicationMenu`, options.App `Menu:` field. MEDIUM confidence (403 on direct fetch; content confirmed via search results).
- [wails.io/docs/reference/runtime/menu/](https://wails.io/docs/reference/runtime/menu/) — Runtime menu update API. HIGH confidence.
- [tailscale.com/docs/install](https://tailscale.com/docs/install) — Platform install commands verified (brew cask, winget, install.sh). HIGH confidence.
- [formulae.brew.sh/cask/tailscale-app](https://formulae.brew.sh/cask/tailscale-app) — `tailscale-app` cask name confirmed. HIGH confidence.
- [creativeprojects/go-selfupdate GitHub](https://github.com/creativeprojects/go-selfupdate) — DMG not supported (zip/tar only); asset naming convention `{name}_{goos}_{goarch}`; macOS universal binary via `UniversalArch`. HIGH confidence.

---

*Stack research for: Remote Sessions & App Polish — AgentHub v1.9*
*Researched: 2026-04-06*
