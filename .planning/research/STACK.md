# Stack Research — v4.2 Funnel Sharing & Polish

**Domain:** Go/Wails v2 desktop app — Tailscale Funnel public sharing + cross-platform native notifications
**Researched:** 2026-06-30
**Confidence:** MEDIUM (APIs verified from go module cache at v1.98.3 source + pkg.go.dev; beeep verified from GitHub source + pkg.go.dev; Wails v2 absence of notifications verified from pkg.go.dev/github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime)

This is a **subsequent-milestone STACK.md** — it covers ONLY new capabilities required for v4.2. The existing stack (Go 1.26.3 / Wails v2.10.2 / React 19 / TS / Vite / pnpm / xterm.js / coder/websocket v1.8.14 / tailscale.com v1.98.3 / capability tokens / react-markdown) is in-place. Do not re-survey or replace any of it.

---

## TL;DR

1. **Tailscale Funnel API**: Zero dependency changes. `tailscale.com v1.98.3` (already pinned) contains every required type and function. Need to persist `local.Client` as a field on `WebServer` (it is currently constructed-then-discarded inside `startTailscale`). All Funnel control goes through `lc.SetServeConfig` / `lc.GetServeConfig` on the same `tailscale.com/client/local` import already in `server.go`.

2. **Funnel prerequisite detection**: Call `ipn.CheckFunnelAccess(port, st.Self)` before `SetServeConfig`. The function checks: HTTPS enabled on the tailnet (`CapabilityHTTPS = "https"`) + node has the `funnel` nodeAttr (`NodeAttrFunnel = "funnel"`) + the requested port is allowed. Returns human-readable error strings the UI can surface verbatim.

3. **BaseURL / Origin landmine**: Funnel exposes `https://<hostname>.ts.net` (port 443, no port in URL). When Funnel is active, `BaseURL()` must return `https://<hostname>` (not `https://<hostname>:7443`). `requireAllowedOrigin` does a byte-for-byte match against `BaseURL()` — it will 403 every Funnel browser if not updated. Share URLs must emit the Funnel URL too.

4. **Native OS notifications (#110)**: Add `github.com/gen2brain/beeep v0.11.2`. Single-function API: `beeep.Notify(title, message, icon)`. No CGO, no dock icon required, cross-platform. One new `go get` required.

5. **Wails v2 has no notifications**: `github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime` has no `SendNotification` or `InitializeNotifications`. Notifications are Wails v3-only. Do not attempt a Wails upgrade — beeep is the correct path.

---

## New Go Dependencies

Only one new module:

| Package | Version | Purpose | Why |
|---------|---------|---------|-----|
| `github.com/gen2brain/beeep` | `v0.11.2` | Cross-platform native OS notifications | Only maintained cross-platform notification library with no CGO, no dock icon requirement, no Windows Store sandbox. Uses osascript/terminal-notifier on macOS, COM API + PowerShell on Windows, D-Bus + notify-send on Linux. |

All Funnel capabilities use modules already in go.mod:

| Capability | Module (already in go.mod) | Import path |
|------------|---------------------------|-------------|
| Serve/Funnel config read/write | `tailscale.com v1.98.3` | `tailscale.com/client/local` |
| ServeConfig struct + helpers | `tailscale.com v1.98.3` | `tailscale.com/ipn` |
| Funnel prereq constants | `tailscale.com v1.98.3` | `tailscale.com/tailcfg` |
| Status / DNSName / CapMap | `tailscale.com v1.98.3` | `tailscale.com/ipn/ipnstate` |

---

## Funnel API — Complete Reference

### LocalClient Methods (`tailscale.com/client/local`)

Server.go already imports this package as `"tailscale.com/client/local"` and creates `var lc local.Client` inside `startTailscale()`. The only change is persisting it as `ws.lc local.Client` on the `WebServer` struct.

```go
// Get the current serve configuration (nil, nil = not configured)
func (lc *Client) GetServeConfig(ctx context.Context) (*ipn.ServeConfig, error)

// Set or replace the serve configuration. nil config clears all serve settings.
func (lc *Client) SetServeConfig(ctx context.Context, config *ipn.ServeConfig) error

// Status including all peer info. For Funnel checks, Self is all that matters.
func (lc *Client) Status(ctx context.Context) (*ipnstate.Status, error)

// Status without peer table — use this for Funnel prereq checks (faster).
func (lc *Client) StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)

// QueryFeature returns a control-server URL for enabling a feature if needed.
// Valid feature strings: "serve", "funnel".
func (lc *Client) QueryFeature(ctx context.Context, feature string) (*tailcfg.QueryFeatureResponse, error)
```

### ServeConfig Struct (`tailscale.com/ipn`)

```go
type ServeConfig struct {
    // TCP maps listening port → handler. For HTTPS Funnel use HTTPS:true.
    TCP map[uint16]*TCPPortHandler `json:",omitempty"`

    // Web maps "hostname:port" → HTTP handlers keyed by mount point.
    Web map[HostPort]*WebServerConfig `json:",omitempty"`

    // Services maps service names → service configs (not used for node-level Funnel).
    Services map[tailcfg.ServiceName]*ServiceConfig `json:",omitempty"`

    // AllowFunnel is the set of "hostname:port" values for which Funnel is active.
    AllowFunnel map[HostPort]bool `json:",omitempty"`

    // Foreground holds ephemeral configs scoped to a WatchIPNBus session ID
    // (used by `tailscale serve` CLI without --bg; not needed for AgentHub).
    Foreground map[string]*ServeConfig `json:",omitempty"`

    // ETag for optimistic concurrency — send as If-Match on SetServeConfig.
    // Populated on GetServeConfig; must be re-sent on SetServeConfig to avoid races.
    ETag string `json:"-"`
}

type HostPort string // "hostname:port" — no implicit 443, must contain a colon
func (hp HostPort) Port() (uint16, error)

type TCPPortHandler struct {
    HTTPS bool   // let Tailscale daemon terminate TLS and route via Web handlers
    HTTP  bool   // plain HTTP (not needed for Funnel — always use HTTPS:true)
    TCPForward string   // "ip:port" for raw TCP forwarding (alternative to HTTPS)
    TerminateTLS string // SNI name to terminate TLS before TCPForward
}

type WebServerConfig struct {
    Handlers map[string]*HTTPHandler // mount point ("/", "/api/", etc.) => handler
}

type HTTPHandler struct {
    Proxy string // "http://localhost:PORT" or "https://localhost:PORT" — use for AgentHub
    Path  string // local filesystem path (not needed here)
    Text  string // static text response (not needed here)
}
```

### ServeConfig Helper Methods

```go
// SetFunnel sets AllowFunnel["hostname:port"] = setOn (or deletes the key if false).
// This is the correct way to toggle Funnel rather than manually editing AllowFunnel.
func (sc *ServeConfig) SetFunnel(host string, port uint16, setOn bool)

// IsFunnelOn reports whether any AllowFunnel entry is currently true.
func (sc *ServeConfig) IsFunnelOn() bool
```

### Funnel Prerequisite Functions (`tailscale.com/ipn`)

```go
// CheckFunnelAccess performs all three prerequisite checks before SetServeConfig.
// node = st.Self from StatusWithoutPeers. Returns nil on success.
func CheckFunnelAccess(port uint16, node *ipnstate.PeerStatus) error

// NodeCanFunnel checks HTTPS capability + "funnel" node attribute only (no port check).
func NodeCanFunnel(node *ipnstate.PeerStatus) error

// CheckFunnelPort verifies the port is allowed by CapabilityFunnelPorts policy.
func CheckFunnelPort(wantedPort uint16, node *ipnstate.PeerStatus) error
```

Exact error strings returned (surface these verbatim in the UI):
- `"Funnel not available; HTTPS must be enabled. See https://tailscale.com/s/https."`
- `"Funnel not available; \"funnel\" node attribute not set. See https://tailscale.com/s/no-funnel."`
- `"port N is not allowed for funnel; allowed ports are: 443,8443,10000"` (port list is policy-controlled)

### Capability Constants (`tailscale.com/tailcfg`)

```go
// CapabilityHTTPS — checked by NodeCanFunnel; means HTTPS/MagicDNS is on for the tailnet
tailcfg.CapabilityHTTPS    NodeCapability = "https"

// NodeAttrFunnel — checked by NodeCanFunnel; the "funnel" nodeAttr in tailnet policy
tailcfg.NodeAttrFunnel     NodeCapability = "funnel"

// CapabilityFunnelPorts — value includes ?ports=443,8443,10000 query param
tailcfg.CapabilityFunnelPorts NodeCapability = "https://tailscale.com/cap/funnel-ports"
```

Check with: `node.HasCap(tailcfg.NodeAttrFunnel)` where `node = st.Self`.

### Getting the Public Funnel Hostname

```go
st, err := lc.StatusWithoutPeers(ctx)
// st.Self.DNSName has a trailing dot: "myhostname.tail1234.ts.net."
// Trim it before using as the Funnel hostname:
hostname := strings.TrimSuffix(st.Self.DNSName, ".")
// OR use st.CertDomains[0] — same hostname but without trailing dot, already clean.
// CertDomains is what the existing Config.FQDN field is populated from.
```

### Minimal Funnel Enable/Disable Sequence

Funnel uses a SEPARATE external port (443, 8443, or 10000) — the Tailscale daemon terminates TLS on that port and proxies to AgentHub's local server. AgentHub's existing HTTPS listener on the Tailscale IP continues serving tailnet members on its own port unchanged.

```go
// ENABLE Funnel
func (ws *WebServer) EnableFunnel(ctx context.Context, funnelPort uint16) error {
    st, err := ws.lc.StatusWithoutPeers(ctx)
    if err != nil {
        return err
    }
    // Step 1: check prerequisites — surfaces human-readable errors
    if err := ipn.CheckFunnelAccess(funnelPort, st.Self); err != nil {
        return err
    }

    hostname := strings.TrimSuffix(st.Self.DNSName, ".")

    // Step 2: read-modify-write with ETag for optimistic concurrency
    sc, err := ws.lc.GetServeConfig(ctx)
    if err != nil {
        return err
    }
    if sc == nil {
        sc = new(ipn.ServeConfig)
    }

    // Step 3: TCP handler — Tailscale daemon terminates HTTPS on funnelPort
    if sc.TCP == nil {
        sc.TCP = make(map[uint16]*ipn.TCPPortHandler)
    }
    sc.TCP[funnelPort] = &ipn.TCPPortHandler{HTTPS: true}

    // Step 4: Web handler — proxy all traffic to AgentHub's local HTTPS server
    hp := ipn.HostPort(net.JoinHostPort(hostname, strconv.Itoa(int(funnelPort))))
    if sc.Web == nil {
        sc.Web = make(map[ipn.HostPort]*ipn.WebServerConfig)
    }
    // Use the local TLS server's address. AgentHub's listener is already on
    // ws.Addr() which is "100.x.x.x:PORT". Use localhost for the proxy target.
    _, localPort, _ := net.SplitHostPort(ws.Addr())
    sc.Web[hp] = &ipn.WebServerConfig{
        Handlers: map[string]*ipn.HTTPHandler{
            "/": {Proxy: "https://localhost:" + localPort},
        },
    }

    // Step 5: enable Funnel for this host:port
    sc.SetFunnel(hostname, funnelPort, true)

    // Step 6: apply (ETag is carried on sc from GetServeConfig for concurrency safety)
    return ws.lc.SetServeConfig(ctx, sc)
}

// DISABLE Funnel (call on toggle-off, web-share-off, and session end)
func (ws *WebServer) DisableFunnel(ctx context.Context, funnelPort uint16) error {
    sc, err := ws.lc.GetServeConfig(ctx)
    if err != nil || sc == nil {
        return err
    }
    hostname := strings.TrimSuffix(ws.funnelHostname, ".") // cached from enable
    hp := ipn.HostPort(net.JoinHostPort(hostname, strconv.Itoa(int(funnelPort))))

    sc.SetFunnel(hostname, funnelPort, false)
    delete(sc.TCP, funnelPort)
    delete(sc.Web, hp)

    // Nil out empty maps (SetFunnel does this for AllowFunnel; do the same for TCP/Web)
    if len(sc.TCP) == 0 { sc.TCP = nil }
    if len(sc.Web) == 0 { sc.Web = nil }

    return ws.lc.SetServeConfig(ctx, sc)
}
```

### BaseURL / Origin Fix When Funnel Active

The existing `requireAllowedOrigin` does a byte-for-byte match against `ws.BaseURL()`. When Funnel is enabled on port 443, the browser's Origin header will be `https://hostname.ts.net` (no port suffix — 443 is the default HTTPS port). `BaseURL()` must return this form.

```go
// WebServer needs a new field: funnelActive bool + funnelHostname string + funnelPort uint16
// BaseURL() must check these:
func (ws *WebServer) BaseURL() string {
    ws.mu.RLock()
    funnelActive := ws.funnelActive
    funnelHostname := ws.funnelHostname
    funnelPort := ws.funnelPort
    ws.mu.RUnlock()

    if funnelActive {
        if funnelPort == 443 {
            return fmt.Sprintf("https://%s", funnelHostname) // no port — 443 is default
        }
        return fmt.Sprintf("https://%s:%d", funnelHostname, funnelPort)
    }
    // ... existing tailscale/local logic unchanged
}
```

Join-code URLs emitted by `handleExchangeJoinCode` and `issueCapabilitiesForSession` already call `ws.BaseURL()` — they will automatically use the Funnel URL once `BaseURL()` is Funnel-aware. Capability tokens are session-scoped (not host-scoped), so no token changes are needed.

### Allowed Funnel Ports

Policy-controlled by the tailnet ACL via `CapabilityFunnelPorts`. Default allowed ports for most tailnets: **443**, **8443**, **10000**. Recommend defaulting to port **443** in AgentHub (standard HTTPS, no port in URL). Use `ipn.CheckFunnelPort` to validate before attempting to configure a specific port, and surface the error if the port isn't available. Do not hard-code port assumptions — the `CapabilityFunnelPorts` node attribute contains the actual allowed list.

---

## Notifications API — Complete Reference

### Package: `github.com/gen2brain/beeep`

**Version:** `v0.11.2` (published Dec 2025)
**Import:** `"github.com/gen2brain/beeep"`

```go
// Notify sends a native desktop notification.
// icon: string path to a PNG file, []byte PNG data, or "" for no icon.
// Returns non-nil error only if ALL platform notification methods fail.
func Notify(title, message string, icon any) error
```

Minimal call site:

```go
import "github.com/gen2brain/beeep"

func notifySessionAwaitingInput(sessionName string) {
    _ = beeep.Notify(
        "AgentHub",
        sessionName + " is awaiting input",
        "", // no custom icon — uses OS default
    )
}
```

### Platform Behavior

| Platform | Primary | Fallback | CGO? | Dock icon required? |
|----------|---------|---------|------|---------------------|
| macOS | `terminal-notifier` binary (if installed) | `osascript display notification` | No | No |
| Windows 10/11 | Windows Runtime COM API | PowerShell → Win32 | No | No |
| Linux | D-Bus `org.freedesktop.Notifications` | `notify-send` binary | No | No |

**macOS LSUIElement constraint:** AgentHub runs as `LSUIElement` (no Dock icon). `beeep` on macOS uses `osascript -e 'display notification ...'` as its fallback, which runs as a separate subprocess. This is fully compatible with LSUIElement apps — `osascript` has no dependency on the calling process's Dock presence. The notification appears attributed to "Script Editor" in macOS Notification Center (not "AgentHub") unless the user has installed `terminal-notifier`. This is an acceptable UX trade-off for v4.2. If branded attribution becomes important in a future milestone, the path is UNUserNotificationCenter via CGO — but that requires notification permission prompts, proper entitlements, and significantly more integration work.

**Windows constraint:** The COM API approach works for tray-resident apps with no taskbar window. No `AppUserModelId` registration is required for basic notifications via beeep. If Windows Store distribution is ever needed (unlikely for AgentHub), beeep's approach would need revisiting — not a concern here.

**Linux constraint:** The D-Bus path requires a running `org.freedesktop.Notifications` daemon (present in GNOME, KDE, XFCE; absent in minimal/server environments). The notify-send fallback handles most remaining cases. For headless Linux server installs (where notifications make no sense), both paths will fail silently — acceptable.

### Installation

```bash
go get github.com/gen2brain/beeep@v0.11.2
```

---

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Tailscale admin API OAuth client | Would require `policy:write` scope token, admin consent, and external HTTP calls to `api.tailscale.com`. Out of scope — the goal is node-local Funnel control via `LocalClient` only. | `local.Client.SetServeConfig` |
| `tailscale.com/tsnet` | AgentHub uses the system Tailscale daemon (not an embedded tsnet server). tsnet creates a second independent tailscale node — wrong architecture. | `tailscale.com/client/local` (already present) |
| Wails v3 upgrade | v3 is still alpha (v3.0.0-alpha); upgrading from v2.10.2 is a breaking rewrite, not a patch bump. The v3 notification API is the only thing v4.2 would gain, and beeep covers it adequately. v2.10.2 is the last stable v2 release. | `github.com/gen2brain/beeep` |
| `UNUserNotificationCenter` CGO | Would give branded macOS notifications ("AgentHub" as sender), but requires: entitlements config, macOS permission request dialog, CGO on macOS, and significantly more code. Disproportionate for v4.2. Revisit if user feedback calls for it. | `beeep.Notify` with osascript fallback |
| Any Funnel port other than 443 | Port 443 gives a clean `https://hostname.ts.net` URL with no port component — the cleanest shareable URL. Only fall back to 8443/10000 if `CheckFunnelPort(443, ...)` returns an error. | Default to 443 |
| ACL automation via admin API | Would require external HTTP calls, OAuth tokens, and admin-level tailnet access. The v4.2 Help article documents the manual ACL approach instead. | Help guide (Part 2 of #107) |

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `tailscale.com v1.98.3` | Go 1.26.3 | Already in go.mod. All Funnel APIs verified present: `SetServeConfig`, `GetServeConfig`, `CheckFunnelAccess`, `NodeCanFunnel`, `SetFunnel`, `IsFunnelOn`. No version bump needed. |
| `github.com/gen2brain/beeep v0.11.2` | Go 1.26.3 | Published Dec 2025. No CGO on any platform. Compatible with existing build matrix (macOS arm64/amd64, Windows amd64, Linux amd64). |
| `tailscale.com/ipn.ServeConfig` ETag field | `tailscale.com v1.98.3` | ETag is populated by `GetServeConfig` (via HTTP `ETag` response header translated to the struct field) and sent as `If-Match` by `SetServeConfig`. Use this to avoid concurrent Funnel config races. |

---

## Integration Points Against Existing Code

| Existing Code | Change Required |
|---------------|----------------|
| `internal/webserver/server.go: startTailscale()` | Persist `lc local.Client` as `ws.lc`; do not discard after use |
| `WebServer` struct | Add `lc local.Client`, `funnelActive bool`, `funnelHostname string`, `funnelPort uint16` (all guarded by `ws.mu`) |
| `WebServer.BaseURL()` | Return Funnel URL (`https://hostname` without port) when `ws.funnelActive` is true and funnelPort == 443 |
| `origin_mw.go: allowedOrigins()` | Already calls `ws.BaseURL()` — no change needed once BaseURL is Funnel-aware |
| `capability_mw.go` / join-code handlers | Already call `ws.BaseURL()` for URL construction — no change needed |
| Wails `App.go` | Add `EnableFunnel(port int)` and `DisableFunnel()` bound methods calling daemon's webserver methods |
| Daemon API (`internal/daemon/api.go`) | Add Funnel enable/disable IPC endpoints; store `funnelPort` in session/server state |
| `frontend/src/components/SessionShareModal.tsx` | Add Funnel toggle with risk acknowledgment dialog; show Funnel URL when active |

---

## Sources

- `/Users/ken/go/pkg/mod/tailscale.com@v1.98.3/ipn/serve.go` — `ServeConfig` struct (lines 51-81), `SetFunnel` (line 469), `CheckFunnelAccess`/`NodeCanFunnel`/`CheckFunnelPort` (lines 601-700), `IsFunnelOn` (line 576), exact error strings (lines 612-615). HIGH confidence — source of truth.
- `/Users/ken/go/pkg/mod/tailscale.com@v1.98.3/tailcfg/tailcfg.go` — `tailcfg.CapabilityHTTPS = "https"` (line 2473), `tailcfg.CapabilityFunnelPorts = "https://tailscale.com/cap/funnel-ports"` (line 2532), `tailcfg.NodeAttrFunnel = "funnel"` (line 2542). HIGH confidence — source of truth.
- `pkg.go.dev/tailscale.com@v1.98.3/client/local` — `SetServeConfig`, `GetServeConfig`, `Status`, `StatusWithoutPeers`, `QueryFeature` signatures confirmed. MEDIUM confidence.
- `pkg.go.dev/tailscale.com@v1.98.3/ipn/ipnstate#Status` — `Status.Self *PeerStatus`, `Status.CertDomains []string`, `PeerStatus.DNSName string` (trailing dot), `PeerStatus.CapMap`, `PeerStatus.HasCap()`. MEDIUM confidence.
- `pkg.go.dev/github.com/gen2brain/beeep` — `Notify(title, message string, icon any) error`, v0.11.2 (Dec 2025). MEDIUM confidence.
- `github.com/gen2brain/beeep/blob/master/notify_darwin.go` — Confirmed osascript subprocess approach; no CGO; no dock-icon dependency; falls back to terminal-notifier. MEDIUM confidence.
- `pkg.go.dev/github.com/wailsapp/wails/v2@v2.10.2/pkg/runtime` — Confirmed NO notification functions exist in Wails v2.10.2. HIGH confidence.
- `/Users/ken/dev/agenthub/go.mod` — confirmed `tailscale.com v1.98.3`, `wailsapp/wails/v2 v2.10.2`, Go 1.26.3.
- `/Users/ken/dev/agenthub/internal/webserver/server.go` — confirmed `tailscale.com/client/local` import as `"local"`, `var lc local.Client` created-and-discarded pattern, `Config.FQDN` populated from `CertDomains[0]`, `BaseURL()` shape, `allowedOrigins()` calls `BaseURL()`.
- `/Users/ken/dev/agenthub/internal/webserver/origin_mw.go` — confirmed `requireAllowedOrigin` does byte-for-byte match on `ws.BaseURL()`.

---
*Stack research for: v4.2 Funnel Sharing & Polish — new capability additions only*
*Researched: 2026-06-30*
