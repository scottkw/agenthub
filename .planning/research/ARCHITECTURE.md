# Architecture Research

**Domain:** Tailscale Funnel public sharing + desktop notifications + web-share bug fixes integrated into existing AgentHub Go/Wails + React app
**Researched:** 2026-06-30
**Confidence:** HIGH — based on direct inspection of all named files and line ranges; no guesswork

---

## 1. Funnel Backend Lifecycle

### Where the LocalClient Lives Now

`internal/webserver/server.go:startTailscale()` (~line 406) constructs a `local.Client` as a stack-local variable solely to supply `GetCertificate` to the TLS config, then discards it:

```go
func (ws *WebServer) startTailscale() error {
    tlsCfg := ws.config.TLSConfig
    if tlsCfg == nil {
        var lc local.Client                         // ← stack-local, discarded after this block
        tlsCfg = &tls.Config{
            GetCertificate: lc.GetCertificate,
            MinVersion:     tls.VersionTLS12,
        }
    }
    ...
```

`local.Client` is zero-value usable (no constructor), so promoting it to a struct field costs nothing.

### Change: Store LocalClient on WebServer

**MODIFIED: `internal/webserver/server.go`**

Add to the `WebServer` struct:

```go
type WebServer struct {
    config  Config
    manager *relay.HubManager
    lc      local.Client   // NEW — promoted from startTailscale stack local; zero-value usable

    // Funnel state — guarded by ws.mu
    funnelActive  bool    // true when SetServeConfig/AllowFunnel is live
    funnelBaseURL string  // cached "https://<fqdn>" (no port) when Funnel active

    mu         sync.RWMutex
    ...
}
```

In `startTailscale()`, remove the stack-local `var lc local.Client` and use `ws.lc` instead:

```go
func (ws *WebServer) startTailscale() error {
    tlsCfg := ws.config.TLSConfig
    if tlsCfg == nil {
        tlsCfg = &tls.Config{
            GetCertificate: ws.lc.GetCertificate,   // ← use the struct field
            MinVersion:     tls.VersionTLS12,
        }
    }
    ...
```

The WhoIs calls in `handleWSSRelay` already create their own `var lc local.Client` separately (line ~1122); those remain unchanged.

### New WebServer Methods

```go
// EnableFunnel activates Tailscale Funnel for this webserver port.
// Caches the Funnel-facing base URL (https://<fqdn>, port 443) from lc.Status().
func (ws *WebServer) EnableFunnel() error

// DisableFunnel tears down the Tailscale Funnel serve config.
func (ws *WebServer) DisableFunnel() error

// FunnelBaseURL returns the https://<fqdn> URL (port 443, no port in string)
// when Funnel is active, or "" when inactive.
func (ws *WebServer) FunnelBaseURL() string
```

`EnableFunnel` calls `ws.lc.SetServeConfig(ctx, &ipn.ServeConfig{AllowFunnel: true, ...})`.
`DisableFunnel` calls `ws.lc.SetServeConfig(ctx, &ipn.ServeConfig{})` (empty config removes Funnel).
After `EnableFunnel` succeeds, cache the FQDN from `ws.lc.Status()` as `ws.funnelBaseURL = "https://" + st.Self.DNSName`.

### Funnel Toggle Data Flow

```
Frontend SessionShareModal.tsx
  ↓ new "Share over internet" toggle → handleFunnelToggle()
  ↓ calls App.SetSessionFunnel(sessionID, enabled)   [NEW Wails bound method]
  ↓
app.go: App.SetSessionFunnel(sessionID, enabled)
  ↓ a.client.SetSessionFunnel(sessionID, enabled)
  ↓
daemon/api.go: POST /sessions/{id}/funnel            [NEW endpoint]
  → handleSetSessionFunnel()
  ↓ updates api.funnelSessions[sessionID] = enabled  (under a.mu)
  ↓ if enabled: ws.EnableFunnel() → lc.SetServeConfig
  ↓ if disabled AND no other session has funnel: ws.DisableFunnel()
```

### Where Funnel State Lives in the Daemon

**MODIFIED: `internal/daemon/api.go`** — Add to the `API` struct:

```go
type API struct {
    ...
    // funnelSessions tracks which session IDs have Funnel active.
    // When the map is empty, Funnel is torn down.
    // Guarded by a.mu.
    funnelSessions map[string]bool
}
```

Any session enabling Funnel calls `ws.EnableFunnel()`; the serve config is idempotent.
When `len(funnelSessions) == 0`, call `ws.DisableFunnel()`.

### Three Teardown Sites

| Trigger | Where | Action |
|---------|-------|--------|
| User disables Funnel toggle | `handleSetSessionFunnel(id, false)` | Remove from `funnelSessions`; if empty, `ws.DisableFunnel()` |
| User disables web-share | `handleWebServe(id, false)` | If `funnelSessions[id]`, remove it; if map now empty, `ws.DisableFunnel()` |
| Session ends naturally | `onExit` callback wired in `handleWebServerStart` / `AutoStartWebServer` | Same as web-share disable path |

### Per-Session vs Per-Listener Reconciliation

Funnel is a **node-level Tailscale serve config** — it exposes the single webserver port to the public internet. It is NOT configurable per-session. The "per-session" Funnel toggle in the UI means: **for this session's share URL, use the Funnel-facing base URL** (so the recipient can access it from outside the tailnet). All sessions on the node are technically reachable via Funnel, but only cap-token-bearing URLs are usable, preserving the access model.

Implication: when session A has Funnel enabled and session B does not, session B's share URL still uses the tailnet FQDN (port 7443). An internet user who somehow guesses or scans for session B would need a valid cap token. The risk acknowledgment dialog makes this clear.

---

## 2. The Origin / BaseURL Funnel-Awareness Change (Integration Landmine)

### Root Cause

Tailscale Funnel routes via port **443** (standard HTTPS, no port in URL). The webserver binds on **7443**. The serve config maps public 443 → local 7443 via TCP proxy. Consequence:

- Tailnet share URL (existing): `https://hostname.ts.net:7443/sessions/id?cap=TOKEN`
- Funnel share URL (new): `https://hostname.ts.net/sessions/id?cap=TOKEN` (no port)

When a browser WS-upgrades to `wss://hostname.ts.net/sessions/id/ws?cap=TOKEN`, it sends:
```
Origin: https://hostname.ts.net
```

But `ws.BaseURL()` returns `https://hostname.ts.net:7443`. The current `requireAllowedOrigin` does a byte-for-byte exact match — **403 before the cap token is ever checked**.

### Fix: Funnel-Aware BaseURL and Dual Origin Allowlist

**MODIFIED: `internal/webserver/server.go`** — `BaseURL()` stays unchanged (returns tailnet FQDN with port). Add `FunnelBaseURL()` (returns `https://hostname.ts.net`, no port). Expose it to the daemon.

**MODIFIED: `internal/webserver/origin_mw.go`** — `requireAllowedOrigin` checks both:

```go
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin == "" {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        tailnetURL := ws.BaseURL()
        funnelURL  := ws.FunnelBaseURL() // "" when Funnel inactive
        if tailnetURL == "" || (origin != tailnetURL && (funnelURL == "" || origin != funnelURL)) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        next(w, r)
    }
}
```

`allowedOrigins()` (used for websocket.AcceptOptions belt-and-suspenders) likewise returns both when Funnel is active.

`originAllowedForWrite()` in `capability_mw.go` needs the same dual check (write routes via web guests use the Funnel URL when Funnel is active).

### Fix: Share URLs Use Funnel Base URL

**MODIFIED: `internal/daemon/api.go`**

`issueCapabilitiesForSession` currently (~line 1287):
```go
base := ws.BaseURL()
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok
```

Change to:
```go
base := ws.BaseURL()
if funnelBase := ws.FunnelBaseURL(); funnelBase != "" && a.funnelSessions[sessionID] {
    base = funnelBase
}
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok
```

`handleExchangeJoinCode` (~line 1385) likewise:
```go
url := ws.BaseURL() + "/sessions/" + claims.SID + "?cap=" + token
```
becomes:
```go
base := ws.BaseURL()
if funnelBase := ws.FunnelBaseURL(); funnelBase != "" && a.funnelSessions[claims.SID] {
    base = funnelBase
}
url := base + "/sessions/" + claims.SID + "?cap=" + token
```

### Dependency Order: Funnel-Aware BaseURL Must Land With (or Before) Funnel Toggle

If `ws.EnableFunnel()` is called before the Origin allowlist is updated, the first web guest on the Funnel URL gets a 403. Both changes must ship in the same phase. The ordering is:

1. Store `lc` on `WebServer` struct
2. Add `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL` to `WebServer`
3. Update `requireAllowedOrigin` + `allowedOrigins` + `originAllowedForWrite` to use dual allowlist
4. Update share URL builders to use `funnelBase` when active
5. Add daemon API endpoint + `funnelSessions` map
6. Add `App.SetSessionFunnel` Wails bound method
7. Add risk-ack dialog + Funnel toggle to `SessionShareModal.tsx`

**Critical: steps 1-4 must be in the same phase as step 5-7.** A partial deployment where `EnableFunnel()` works but the Origin check is not updated causes silent 403s.

---

## 3. Notifications (#110) Wiring

### Detection: Where "Awaiting Input" Is Set

`internal/daemon/engine.go` calls `status.Watch(hub, id, cli, callback)` at session creation (~line 470). The callback is:

```go
func(sid string, s status.SessionStatus) {
    e.statusMu.Lock()
    e.sessionStatuses[sid] = s
    e.statusMu.Unlock()
    if onStatus != nil {
        onStatus(sid, s)
    }
}
```

The `onStatus` callback is passed from `api.go` → `engine.CreateSession()`. In `api.go`, this callback is wired at `handleCreateSession` time. In `app.go`, the GUI layer polls via `pollSessionStatus` (a goroutine started after `CreateSession`) every 500 ms, calling `a.client.GetSessionStatus(sessionID)` and emitting `session:status` Wails events when status changes (line ~303-309).

The notification trigger point is in `app.go:pollSessionStatus`:

```go
if s.Status != last {
    last = s.Status
    runtime.EventsEmit(a.ctx, "session:status", ...)   // existing
    if s.Status == "waiting" {
        a.maybeNotifyWaiting(sessionID, s.Name)          // NEW
    }
}
```

### New: App.maybeNotifyWaiting

**MODIFIED: `app.go`** — Add de-dup state and the notification call:

```go
// notifiedWaiting tracks when we last notified for each session (de-dup key).
// Guarded by notifiedMu.
notifiedMu      sync.Mutex
notifiedWaiting map[string]time.Time

func (a *App) maybeNotifyWaiting(sessionID, sessionName string) {
    if !a.settings().NotifyOnWaiting {    // user toggle
        return
    }
    a.notifiedMu.Lock()
    defer a.notifiedMu.Unlock()
    if last, ok := a.notifiedWaiting[sessionID]; ok && time.Since(last) < 60*time.Second {
        return // de-dup: no flood within 60s
    }
    a.notifiedWaiting[sessionID] = time.Now()
    sendNotification("AgentHub", sessionName+" is awaiting input")
}
```

### Platform Files

The existing pattern is `notification_darwin.go` (CGO/UNUserNotificationCenter) + `notification_other.go` (no-op). For v4.2, Windows and Linux need real implementations:

| File | Platform | Mechanism |
|------|----------|-----------|
| `notification_darwin.go` | `//go:build darwin` | Existing UNUserNotificationCenter (CGO) |
| `notification_windows.go` | `//go:build windows` | `github.com/go-toast/toast` or WinRT via `golang.org/x/sys` |
| `notification_linux.go` | `//go:build linux` | `exec.Command("notify-send", ...)` |
| `notification_other.go` | `//go:build !darwin && !windows && !linux` | No-op stub |

### User Toggle Wiring

**MODIFIED: `internal/daemon/types.go`** — add `NotifyOnWaiting bool` to `Settings` struct.

**MODIFIED: `internal/daemon/settings.go`** — add field to marshaling/unmarshaling.

**MODIFIED: `frontend/src/components/SettingsTab.tsx`** — add toggle in "Session Behavior" section (alongside the #116 "stay on Hub tab" option). Same `SaveSettings` / three-state button pattern as existing Settings toggles.

---

## 4. Bug Fixes — Root Causes Located in Code

### #112 — Web Guests Lost Plugin-Config + SSE After Phase 159 Redirect

**Root cause (confirmed):**

Phase 159 `handleTerminalPage` (server.go ~line 979) redirects `/sessions/{id}?cap=TOKEN` → `/app/?session={id}&cap=TOKEN`. The `/app/` SPA is `App.tsx`. In the Wails desktop app, `App.tsx` receives plugin settings via the `settings:plugins` Wails runtime event (`runtime.EventsEmit(a.ctx, "settings:plugins", s)` in `app.go` line ~681). This event never fires in a browser — it is a Wails-specific inter-process channel.

`WebShareSessionView.tsx` receives `pluginConfig` as a prop from `App.tsx`. In the browser (web guest), `App.tsx` never populates that state, so `pluginConfig` is always `null`. The SSE endpoint at `/api/plugin-config/stream?cap=TOKEN` exists and is capability-gated, but `WebShareSessionView` never subscribes to it.

**Fix — MODIFIED: `frontend/src/components/Hub/WebShareSessionView.tsx`**

Add a self-contained fetch + SSE hook inside `WebShareSessionView` that activates when the component runs in a browser context (no Wails runtime):

```tsx
// In WebShareSessionView, add local plugin config state:
const [localPluginConfig, setLocalPluginConfig] = useState<PluginSettings | null>(null)

useEffect(() => {
  // When pluginConfig prop is null (browser/web-share context), fetch it directly.
  if (pluginConfig != null) return
  const configURL = `${apiBaseURL}/api/plugin-config?cap=${encodeURIComponent(capToken)}`
  fetch(configURL)
    .then(r => r.ok ? r.json() : null)
    .then(data => { if (data) setLocalPluginConfig(data) })
    .catch(() => {})

  const es = new EventSource(`${apiBaseURL}/api/plugin-config/stream?cap=${encodeURIComponent(capToken)}`)
  es.onmessage = (e) => {
    try { setLocalPluginConfig(JSON.parse(e.data)) } catch {}
  }
  return () => es.close()
}, [apiBaseURL, capToken, pluginConfig])

const effectivePluginConfig = pluginConfig ?? localPluginConfig
```

Pass `effectivePluginConfig` to `TerminalPanel` instead of `pluginConfig`.

The `apiBaseURL` is already defined in `WebShareSessionView` as `window.location.origin`.

### #115 — Footer "Enable Web" Button State Drift

**Root cause (confirmed):**

`StatusBar.tsx` shows "Enable Web" / "Disable Web" buttons. These call `onToggleWeb` which wires to `ToggleWebServing(sessionID, !webEnabled)` in `App.tsx` (~line 891). This directly toggles web-serving without opening the Share modal, creating state drift: the session becomes web-enabled but no cap tokens are issued, so the user sees "WEB ON" with no shareable links.

**Fix — MODIFIED: `frontend/src/components/StatusBar.tsx`**

Rename prop `onToggleWeb` → `onOpenShareModal`. Remove both "Enable Web" and "Disable Web" buttons. Add a single "Share Session" button that always opens the Share modal regardless of current web-enabled state:

```tsx
{webServerRunning && (
  <>
    <span className={`tab-status-bar__state tab-status-bar__state--${webEnabled ? 'on' : 'off'}`}>
      {webEnabled ? 'WEB ON' : 'WEB OFF'}
    </span>
    <button className="tab-status-bar__btn" onClick={onOpenShareModal}>
      Share Session
    </button>
  </>
)}
```

**MODIFIED: `App.tsx`** — wire the StatusBar's `onOpenShareModal` to open `SessionShareModal` for the current session, same path as the Hub card Share button.

### #117 — Relay Allows Only One Remote Viewer (Second Kicks First)

**Root cause (confirmed by relay/hub.go inspection):**

The Hub does NOT have a single-viewer limit. Multiple subscribers are supported (the `subscribers map[*Subscriber]struct{}` has no size cap). The kick happens because `hub.broadcast` (line ~332) uses a non-blocking send with a 256-frame buffer:

```go
select {
case sub.Msgs <- frame:
default:
    go sub.CloseSlow()   // kicks the subscriber if buffer full
}
```

When a session has active PTY output (e.g. Claude Code streaming), a subscriber's 256-frame buffer fills quickly if their browser tab goes to background (browser throttles WebSocket reads). When a second viewer joins, `NotifyViewerCount` + `NotifyPresence` add 2 more frames to every existing subscriber's channel. If sub1's channel was at 255/256, those 2 frames overflow it → `CloseSlow` fires → sub1's WebSocket is closed by the server.

The "no way to disconnect" symptom: `CloseSlow` calls `conn.Close(StatusPolicyViolation, "too slow")`, which sends a WS close frame. The browser receives this but may not immediately trigger unload behavior. The Hub's viewer count is not updated until `hub.Unsubscribe(sub)` runs in the deferred cleanup, which requires the read pump to detect the error and signal `readDone`. This async gap leaves the Hub showing stale viewer count.

**Fix — two-part:**

**Part A — MODIFIED: `internal/relay/hub.go`** — increase subscriber buffer from 256 to 1024:
```go
func NewHub(...) *Hub {
    ...
    // Existing: make(chan []byte, 256)
    // Change to:  make(chan []byte, 1024)
```

Part A reduces but does not eliminate the kick risk.

**Part B — NEW component or MODIFIED: `frontend/src/components/Hub/SessionShareModal.tsx`** — add a viewer list showing active subscribers (from session info / viewer count API) with a "Disconnect" button for each. The disconnect button calls a new `App.KickSessionViewer(sessionID, personKey)` Wails bound method, which calls the daemon to forcibly unsubscribe that subscriber.

New daemon endpoint: `DELETE /sessions/{id}/viewers/{personKey}` → calls `hubManager.Get(id).KickPersonKey(personKey)`.

New Hub method: `KickPersonKey(personKey string)` — iterates subscribers, closes any with matching `sub.PersonKey`.

### #118 — Hub Remote-Open Launches External Browser

**Root cause (confirmed):**

`handleOpenRemoteSession` in `App.tsx` (line 1160-1186) ultimately calls `BrowserOpenURL(url)` after building the cap-bearing URL from the daemon's RemoteCapStore. `BrowserOpenURL` is the Wails runtime function that opens the system default browser. The fix is to instead create an in-app `__websession__` tab.

**Root constraint:** `WebShareSessionView.tsx` (line 57) constructs its WS URL using `window.location.host` — the Wails WebView host, not the remote peer's host. This must be changed to accept an explicit `baseURL` override.

**Fix — MODIFIED: `frontend/src/components/Hub/WebShareSessionView.tsx`**

Add `baseURL?: string` prop. When provided, use it instead of `window.location.host`:

```tsx
const wsURL = baseURL
  ? `wss://${baseURL.replace(/^https?:\/\//, '')}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
  : `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
const apiBaseURL = baseURL ?? window.location.origin
```

**MODIFIED: `App.tsx`** — `handleOpenRemoteSession` (line 1160): instead of `BrowserOpenURL(url)`, create an in-app tab:

```tsx
// Existing: BrowserOpenURL(url)
// Replace with:
setWebParams({ sessionId: pending.id, capToken: cap, baseURL: baseURL })
setActiveId(`__websession__${pending.id}`)
// Add new tab entry to tabs state with label = session.name
```

Pass `webParams.baseURL` through to `WebShareSessionView`:
```tsx
<WebShareSessionView
  sessionId={...}
  capToken={...}
  relayPort={...}
  baseURL={webParams.baseURL}     // NEW
  theme={...}
  pluginConfig={...}
/>
```

---

## 5. Help Guide Surface (#107 Part 2)

No new architecture required. Simple content addition:

**NEW: `frontend/src/content/help/sharing-guide.md`** — markdown article covering Funnel path (with risk acknowledgment link) and the contained alternative (device-share + `tag:agenthub` + `autogroup:shared`→`tcp:7443` ACL grant).

**MODIFIED: `frontend/src/components/HelpTab.tsx`** — add to `SECTION_META`:
```tsx
import sharingGuideMd from '../content/help/sharing-guide.md?raw'

const SECTION_META = [
  { id: 'help-getting-started', label: 'Getting Started', markdown: gettingStartedMd },
  { id: 'help-chat',            label: 'Chat',            markdown: chatMd },
  { id: 'help-sharing-guide',   label: 'Sharing Guide',   markdown: sharingGuideMd },  // NEW
  { id: 'help-faq',             label: 'FAQ',             markdown: faqMd },
]
```

**MODIFIED: `frontend/src/components/HelpSectionNav.tsx`** — add to `SECTIONS`:
```tsx
export const SECTIONS = [
  { id: 'help-getting-started', label: 'Getting Started' },
  { id: 'help-chat',            label: 'Chat' },
  { id: 'help-sharing-guide',   label: 'Sharing Guide' },  // NEW
  { id: 'help-faq',             label: 'Frequently Asked Questions' },
]
```

---

## Component Change Map

| Component | Status | Why |
|-----------|--------|-----|
| `internal/webserver/server.go` | MODIFIED | Promote `local.Client` to struct field; add `EnableFunnel`/`DisableFunnel`/`FunnelBaseURL` |
| `internal/webserver/origin_mw.go` | MODIFIED | Dual-URL allowlist for tailnet + Funnel origins |
| `internal/webserver/capability_mw.go` | MODIFIED | `originAllowedForWrite` needs same dual check |
| `internal/daemon/api.go` | MODIFIED | `funnelSessions` map; `handleSetSessionFunnel`; funnel-aware URL builders at lines 1287-1289 and 1385; `handleWebServe` teardown |
| `internal/daemon/types.go` | MODIFIED | `NotifyOnWaiting bool` in Settings |
| `internal/daemon/settings.go` | MODIFIED | Serialize/deserialize new field |
| `internal/relay/hub.go` | MODIFIED | Buffer 256→1024; new `KickPersonKey` method |
| `app.go` | MODIFIED | `SetSessionFunnel` bound method; `maybeNotifyWaiting` + de-dup state; `KickSessionViewer` bound method |
| `notification_windows.go` | NEW | Real Windows toast notification |
| `notification_linux.go` | NEW | notify-send via exec |
| `frontend/src/components/StatusBar.tsx` | MODIFIED | Replace Enable/Disable Web buttons with "Share Session" → opens modal |
| `frontend/src/components/Hub/SessionShareModal.tsx` | MODIFIED | Add Funnel toggle + risk-ack dialog; add viewer list + Disconnect button |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | MODIFIED | Self-contained plugin-config fetch + SSE; `baseURL` prop for remote-open |
| `frontend/src/components/SettingsTab.tsx` | MODIFIED | `NotifyOnWaiting` toggle in Session Behavior |
| `frontend/src/components/HelpTab.tsx` | MODIFIED | Add sharing-guide to SECTION_META |
| `frontend/src/components/HelpSectionNav.tsx` | MODIFIED | Add sharing-guide to SECTIONS |
| `frontend/src/content/help/sharing-guide.md` | NEW | Funnel + device-share help article |
| `App.tsx` | MODIFIED | `handleOpenRemoteSession` → in-app tab; StatusBar wiring; `webParams.baseURL` |

---

## Dependency-Ordered Build Sequence

```
Phase 165 (or first v4.2 phase):
  [A] Store lc on WebServer struct (server.go) — zero-risk refactor
  [B] Add EnableFunnel/DisableFunnel/FunnelBaseURL methods (server.go)
  [C] Update requireAllowedOrigin + allowedOrigins + originAllowedForWrite (origin_mw.go, capability_mw.go)
  [D] Update share URL builders (api.go lines 1287-1289, 1385)
  Note: A+B+C+D must all land together (see landmine ordering above)

Phase 166 (or same phase):
  [E] Add funnelSessions map + handleSetSessionFunnel endpoint (api.go)
  [F] Add handleWebServe teardown path + onExit teardown (api.go)
  [G] Add App.SetSessionFunnel Wails bound method (app.go)
  [H] Add Funnel toggle + risk-ack dialog to SessionShareModal.tsx
  Dep: [H] requires [G] requires [E]
  Dep: [E] requires [A]+[B]+[C]+[D] to already be in place

Phase 167 (or bundled with 166):
  [I] Help guide markdown + HelpTab + HelpSectionNav registration
  No deps on Funnel backend — can ship in any phase

Phase 168:
  [J] Notifications: NotifyOnWaiting in Settings struct/settings.go
  [K] notification_windows.go + notification_linux.go
  [L] maybeNotifyWaiting in app.go
  [M] SettingsTab.tsx: NotifyOnWaiting toggle

Phase 169 (bug fixes — independent of Funnel):
  [N] #117: Hub buffer 256→1024 + KickPersonKey + App.KickSessionViewer + viewer list in ShareModal
  [O] #115: StatusBar.tsx refactor + App.tsx wiring
  [P] #118: WebShareSessionView baseURL prop + App.tsx handleOpenRemoteSession in-app tab path
  [Q] #112: WebShareSessionView self-contained plugin-config fetch + SSE
```

Note: #112, #115, #117, #118 are independent of each other and of the Funnel feature. They can be split into separate phases or batched.

---

## Data Flow Summary

### Funnel Enable (Happy Path)

```
User clicks "Share over internet" toggle in SessionShareModal.tsx
  ↓ handleFunnelToggle() → App.SetSessionFunnel(sessionID, true)
  ↓ a.client.SetSessionFunnel(sessionID, true)      [Unix socket IPC]
  ↓ handleSetSessionFunnel() in api.go
  ↓ ws.EnableFunnel()
      ↓ ws.lc.SetServeConfig(ctx, &ipn.ServeConfig{AllowFunnel:true,...})
      ↓ st := ws.lc.Status()
      ↓ ws.funnelBaseURL = "https://" + st.Self.DNSName
  ↓ a.funnelSessions[sessionID] = true
  ↓ 200 OK
  ↓ Frontend re-calls IssueCapabilities(sessionID)
  ↓ issueCapabilitiesForSession: base = ws.FunnelBaseURL() (not BaseURL())
  ↓ readURL = "https://hostname.ts.net/sessions/id?cap=TOKEN"  ← no port
  ↓ Frontend shows Funnel URL in SessionShareModal + visual indicator
```

### External Viewer (Public Internet, Funnel Active)

```
Internet user opens: https://hostname.ts.net/sessions/id?cap=TOKEN
  ↓ Tailscale Funnel: public port 443 → local :7443 TCP proxy
  ↓ TLS via ws.lc.GetCertificate (already covered)
  ↓ requireCapability: token valid, grant active, session enabled → 302
  ↓ handleTerminalPage: redirect → /app/?session=id&cap=TOKEN
  ↓ /app/ → React SPA loads
  ↓ WS upgrade: wss://hostname.ts.net/sessions/id/ws?cap=TOKEN
      Origin: https://hostname.ts.net  (no port — port 443 is default)
  ↓ requireAllowedOrigin: origin == ws.FunnelBaseURL() → allowed (NEW dual check)
  ↓ requireCapability: valid → handleWSSRelay
  ↓ Session streams to browser
```

### Notification (Session Transitions to Waiting)

```
status.Watch goroutine detects PTY output pattern → "waiting"
  ↓ engine.sessionStatuses[sid] updated
  ↓ onStatus callback (api.go) propagates to app.go via session:status event
  ↓ app.go pollSessionStatus detects status == "waiting" && status changed
  ↓ maybeNotifyWaiting(sessionID, name)
      ↓ check settings.NotifyOnWaiting
      ↓ check de-dup: notifiedWaiting[id] < 60s ago → skip
      ↓ sendNotification("AgentHub", name + " is awaiting input")
          ↓ macOS: UNUserNotificationCenter (existing CGO)
          ↓ Windows: toast notification (NEW)
          ↓ Linux: notify-send exec (NEW)
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Funnel BaseURL Changing BaseURL() Return Value

**What people do:** Modify `BaseURL()` to return the Funnel URL when Funnel is active, so callers get the "right" URL automatically.

**Why it's wrong:** `BaseURL()` is used by the Origin allowlist, the local Wails frontend, the tray URL display, the Settings URL copy button, and the QR code generator. All of these should continue to show the tailnet URL. Only share links and the Origin allowlist need Funnel-awareness. Changing `BaseURL()` would break the local GUI display and require every caller to be audited.

**Do this instead:** Keep `BaseURL()` unchanged. Add `FunnelBaseURL()` and use it explicitly in the two URL-builder sites and the Origin allowlist.

### Anti-Pattern 2: Storing the LocalClient by Pointer

**What people do:** Add `lc *local.Client` to the WebServer struct and allocate it separately.

**Why it's wrong:** `local.Client` is zero-value usable — the Tailscale docs and source explicitly make no constructor necessary. Storing a pointer adds an indirection and requires allocation.

**Do this instead:** Store by value: `lc local.Client`. This is consistent with how the existing `startTailscale` and `handleWSSRelay` already use `var lc local.Client`.

### Anti-Pattern 3: Calling DisableFunnel in Every Web-Serve Toggle

**What people do:** Disable Funnel whenever any session turns off web-serving, without checking whether other sessions still have Funnel enabled.

**Why it's wrong:** If session A and session B both have Funnel active, disabling session B's web-share would prematurely tear down the Funnel serve config while session A still expects it to be active.

**Do this instead:** Reference-count via `funnelSessions` map. Only call `DisableFunnel()` when `len(funnelSessions) == 0`.

### Anti-Pattern 4: Firing Notifications from the Status Watch Goroutine

**What people do:** Add notification logic inside `status.Watch` callback in `engine.go`.

**Why it's wrong:** `engine.go` is intentionally free of Wails/GUI imports (confirmed in source: comment at top of engine.go). The `sendNotification` function lives in `app.go`'s package (main) and uses CGO/platform APIs. Importing it from engine would create a cross-package dependency that breaks the daemon-without-GUI pattern.

**Do this instead:** Fire from `app.go:pollSessionStatus`, which already owns the status polling and Wails event emission loop. The detection and notification both live in the same layer that already knows about the UI.

---

## Sources

- Direct code inspection:
  - `internal/webserver/server.go` (all sections: struct, startTailscale, BaseURL, setupRoutes, handleWSSRelay)
  - `internal/webserver/origin_mw.go` (requireAllowedOrigin, allowedOrigins)
  - `internal/webserver/capability_mw.go` (requireCapability, originAllowedForWrite)
  - `internal/daemon/api.go` (issueCapabilitiesForSession lines 1217-1299; handleExchangeJoinCode lines 1340-1387; handleWebServe lines 1181-1206)
  - `internal/relay/hub.go` (Subscriber struct, Subscribe, Unsubscribe, broadcast, CloseSlow pattern)
  - `internal/relay/server.go` (handleSession, loopback identity)
  - `app.go` (pollSessionStatus, sendNotification calls, BrowserOpenURL usage, SetSessionFunnel pattern for new method)
  - `frontend/src/components/Hub/SessionShareModal.tsx` (full — existing share toggle pattern)
  - `frontend/src/components/Hub/WebShareSessionView.tsx` (full — wsURL construction, pluginConfig prop)
  - `frontend/src/components/StatusBar.tsx` (full — Enable Web button)
  - `frontend/src/App.tsx` (handleOpenRemoteSession, __websession__ tab branch, webParams)
  - `frontend/src/components/HelpTab.tsx` (SECTION_META)
  - `frontend/src/components/HelpSectionNav.tsx` (SECTIONS)
  - `notification_darwin.go`, `notification_other.go`

---
*Architecture research for: AgentHub v4.2 Funnel Sharing & Polish — integration into existing Go/Wails/React/Tailscale stack*
*Researched: 2026-06-30*
