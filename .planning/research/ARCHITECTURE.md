# Architecture Research

**Domain:** Local network fallback, auto-serve sessions, settings-as-tab, sidebar rename, Claude Code native path detection — AgentHub v1.11
**Researched:** 2026-04-08
**Confidence:** HIGH — based on direct code inspection of all affected files

---

## Existing Architecture (v1.10 baseline)

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        agenthub binary                               │
├──────────────────────┬──────────────────────┬───────────────────────┤
│  GUI mode (Wails)    │  CLI mode            │  Daemon mode          │
│  app.go (App struct) │  cmd_*.go            │  internal/daemon/     │
│  ↕ DaemonClient      │  ↕ DaemonClient      │  service.go           │
└──────────────────────┴──────────────────────┴───────────────────────┘
         │                       │                        │
         └───────────────────────┴────────────────────────┘
                                 │ Unix socket (named pipe on Windows)
                    ┌────────────┴───────────────┐
                    │     daemon API              │
                    │  internal/daemon/api.go     │
                    │  POST /webserver/start      │
                    │  POST /webserver/stop       │
                    │  POST /sessions/{id}/web-serve
                    │  GET  /sessions             │
                    │  POST /sessions             │
                    └────────────┬───────────────┘
                                 │
                    ┌────────────┴───────────────┐
                    │     SessionEngine           │
                    │  internal/daemon/engine.go  │
                    │  Registry + HubManager      │
                    └────────────┬───────────────┘
                                 │
         ┌───────────────────────┼─────────────────────┐
         │                       │                     │
┌────────┴───────┐  ┌────────────┴──────┐  ┌──────────┴──────┐
│ WebServer      │  │ Relay server      │  │ PTY backend     │
│ internal/      │  │ internal/relay/   │  │ internal/pty/   │
│ webserver/     │  │ (TCP random port) │  │ detect.go       │
│ server.go      │  │                   │  │ engine.go       │
│ Binds: TS IP   │  │ Binds: 127.0.0.1  │  │                 │
│ TLS: TS certs  │  │                   │  │                 │
└────────────────┘  └───────────────────┘  └─────────────────┘
```

### Frontend Architecture (React/Wails)

```
App.tsx (root state owner)
├── Sidebar.tsx          — navigation, collapsed state in localStorage
│     items: Home | Remote | Sessions | New Tab | Settings (bottom)
├── TabBar.tsx           — tab strip (session tabs only after v1.10)
└── terminal-container   — content area, one component per tab type
      ├── WelcomeTab               (type: 'welcome')
      ├── DaemonManagerPanel       (type: 'daemon-manager')
      ├── RemoteSessionsPanel      (type: 'remote-sessions')
      ├── TerminalPanel × N        (type: undefined / session tab)
      └── StatusBar × N            (paired with each TerminalPanel)

Overlays (rendered outside terminal-container):
├── SettingsPanel        — isOpen boolean, rendered as modal overlay
├── HealthModal          — triggered by Tailscale health state
├── NewSessionModal      — triggered by New Tab action
└── QRModal              — triggered by StatusBar QR button
```

### Key Boundaries

| Boundary | Protocol | Notes |
|----------|----------|-------|
| Frontend ↔ App.go | Wails bindings (JS→Go) + EventsEmit (Go→JS) | Defined in app.go, exposed as wailsjs TS types |
| App.go ↔ Daemon | HTTP/JSON over Unix socket | DaemonClient in internal/daemon/client.go |
| Daemon ↔ WebServer | In-process method calls | WebServer owned by API struct (a.webServer) |
| Browser ↔ WebServer | WSS + HTTP over TLS | Tailscale IP:port, Let's Encrypt certs |
| CLI ↔ Daemon | Same HTTP/JSON over Unix socket | Same DaemonClient used by GUI |

---

## v1.11 Feature Integration Analysis

### Feature 1: Local Network Fallback (Self-Signed TLS + Password)

**What needs to change:**

The current `StartWebServer` in `app.go` gates startup on three Tailscale conditions (Connected, IP available, HasCerts). When Tailscale is absent, the function returns an error and no server starts. The v1.11 fallback needs an alternative path that:

1. Generates a self-signed CA + leaf cert (the infrastructure removed in v1.2 must be re-added)
2. Binds to the LAN IP (not the Tailscale IP)
3. Adds password middleware (removed in v1.2 Phase 16)

**Integration points:**

```
app.go: StartWebServer()
  └── currently: gates on h.Connected && h.IP != "" && h.HasCerts
  └── new: if Tailscale healthy → Tailscale path (unchanged)
           else → fallback path: generate self-signed cert, bind LAN IP, add password

internal/webserver/server.go: WebServer struct + Config
  └── Config needs: Mode field (tailscale | local), Password string
  └── setupRoutes(): add auth middleware for local mode only

internal/webserver/: new file selfcert.go
  └── GenerateSelfSignedCert(ip string) (*tls.Config, error)
  └── Uses crypto/x509, crypto/ecdsa, crypto/rand — stdlib only, no new deps

internal/daemon/api.go: handleWebServerStart
  └── WebServerStartRequest: add Mode, Password fields
  └── Passes through to WebServer Config

internal/daemon/types.go:
  └── WebServerStartRequest: add Mode string, Password string fields

Password middleware:
  └── http.Handler wrapper checking Authorization header or cookie
  └── Single random password generated at server start (not stored persistently)
  └── Pass password back in WebServerStartResponse for display to user

Frontend: SettingsTab (after Feature 3 converts Settings to tab)
  └── When Tailscale absent: show "Local Network" mode info + generated password display
  └── Persistent nudge banner in App.tsx: shows when webServer is in local mode
```

**New components:**
- `internal/webserver/selfcert.go` — self-signed cert generation (stdlib crypto, no new deps)
- `internal/webserver/auth.go` — password middleware (wraps http.Handler, checks Authorization or session cookie)

**Modified components:**
- `internal/webserver/server.go` — Config gains Mode + Password; Start() branches on Mode
- `internal/daemon/types.go` — WebServerStartRequest adds Mode, Password; WebServerStatusResponse adds Mode
- `internal/daemon/api.go` — handleWebServerStart passes Mode/Password; handleWebServerStatus returns Mode
- `app.go` — StartWebServer detects fallback condition, generates random password, passes to daemon
- `frontend/src/App.tsx` — nudge banner state (`webServerMode: 'tailscale' | 'local' | null`)
- `frontend/src/components/SettingsPanel.tsx` — password display section for local mode

**Password generation:** `crypto/rand` (already a stdlib dep), 16 bytes → base64url → 22 chars. Generated in `app.go` at call time, passed to daemon, stored only in daemon's in-memory WebServer struct. Never written to disk.

**LAN IP resolution:** `net.InterfaceAddrs()` scanning for first non-loopback IPv4. This is the same approach removed in v1.2 (`network.go`). Re-add as a small helper function in `app.go` or a new `internal/webserver/localip.go`.

**Nudge banner:** A new string field `webServerMode` in App.tsx state. When mode is `'local'`, render a persistent banner above the tab bar showing the password and a note that Tailscale would provide a more secure alternative. Banner does not dismiss until Tailscale becomes available or user stops the server.

---

### Feature 2: Auto-Serve Sessions

**What "auto-serve" means:**
- When web server starts: automatically enable web serving for all existing sessions
- When a new session is created: automatically enable web serving if server is running

**Integration points:**

```
app.go: StartWebServer()
  └── After server starts successfully, iterate all sessions via ListSessions()
  └── Call client.ToggleWebServing(id, true) for each
  └── Emit session:web-enabled for each to sync frontend state

app.go: CreateSession()
  └── After daemon returns new session ID, check client.GetWebServerStatus()
  └── If running: call client.ToggleWebServing(newID, true)
  └── Emit session:web-enabled {sessionId, url} Wails event

Frontend App.tsx: useEffect event subscription block
  └── Add handler for session:web-enabled event
  └── Updates webEnabled[sessionId] = true and sessionURLs[sessionId] = url
```

**Recommended approach:** Handle auto-serve in `app.go` on the backend side for both cases. This keeps the frontend stateless about this policy and avoids a race where the frontend might call ToggleWebServing before the session relay hub is ready.

**New Wails event:**
```
session:web-enabled  { sessionId: string, url: string }
```

Frontend subscribes in the existing `useEffect` event subscription block alongside `session:status`. When received, updates both `webEnabled` and `sessionURLs` state maps.

---

### Feature 3: Settings as Sidebar Tab

**Current state:** Settings is a modal overlay (`SettingsPanel` with `isOpen` prop). Sidebar `onSettings` callback calls `setShowSettings(true)`. The component renders as `<div className="settings-overlay">` covering the full app.

**Target state:** Settings is a persistent tab in the tab system, like `WelcomeTab`, `DaemonManagerPanel`, and `RemoteSessionsPanel`.

**Integration points:**

```
frontend/src/App.tsx:
  └── Add SETTINGS_TAB constant: { id: '__settings__', name: 'Settings', type: 'settings' }
  └── Add handleOpenSettings: same pattern as handleOpenDaemonManager (find-or-add + focus)
  └── Sidebar: onSettings prop → handleOpenSettings (currently sets showSettings=true)
  └── terminal-container: add {activeId === SETTINGS_TAB.id && <SettingsPanel ... />}
  └── Remove showSettings state
  └── Remove bottom-of-JSX SettingsPanel overlay render
  └── Load webEnabled/serverRunning state when settings tab becomes active
      (poll pattern like daemon-manager, or just load on mount)

frontend/src/components/SettingsPanel.tsx:
  └── Remove settings-overlay and settings-panel wrapper divs
  └── Remove isOpen prop — rendered only when tab is active (JSX conditional)
  └── Remove onClose prop — closing is standard tab close (handleCloseTab)
  └── Remove settings-panel__header close button
  └── Remove settings-panel__footer Close button
  └── Load state on mount (useEffect with no isOpen guard)
  └── Content becomes full tab panel, not a modal

frontend/src/components/Sidebar.tsx:
  └── aria-label "Settings" already correct
  └── No prop interface changes needed
```

**Tab type expansion:** The `Tab` type in `TabBar.tsx` needs a new `type: 'settings'` variant. The tab-close behavior for settings tab removes it from the array (user can reopen via sidebar). This is the same pattern as all other singleton tabs — no special handling needed.

**SettingsPanel state loading:** Currently triggered by `isOpen` in a `useEffect`. After conversion, trigger on mount — the JSX conditional means the component mounts/unmounts on tab activation (same as WelcomeTab pattern). The `isOpen` guard on the `useEffect` can be removed entirely.

---

### Feature 4: Sidebar Label "New Tab" → "New Session"

**Minimal change:** Two strings in `Sidebar.tsx`.

```
frontend/src/components/Sidebar.tsx:
  aria-label="New Tab"  →  aria-label="New Session"
  <span className="sidebar__label">New Tab</span>
  →
  <span className="sidebar__label">New Session</span>
```

No other production files affected. The `onAdd` prop name is internal and does not need to change.

**Test impact:** Any vitest test in `frontend/src/components/__tests__/` asserting the "New Tab" label text needs updating.

---

### Feature 5: Claude Code Native Install Path Detection

**Current state:** `internal/pty/detect.go` uses `exec.LookPath("claude")` — finds the binary only if it is on the augmented PATH. Claude Code installed via the Anthropic native installer on macOS places the binary at `~/.claude/local/claude`, which is not a standard PATH location and not currently in `AugmentServicePath`.

**Known native install paths:**
- macOS/Linux: `~/.claude/local/claude` (Anthropic native installer)
- Windows: `%LOCALAPPDATA%\AnthropicClaude\claude.exe` or `%APPDATA%\Claude\claude.exe`

**Integration points:**

```
internal/pty/detect.go: DetectCLIs() and DetectCLI()
  └── After LookPath fails for "claude", probe claudeNativePaths() candidates
  └── New helper: claudeNativePaths() []string — platform-specific paths via runtime.GOOS
  └── If candidate exists and is executable: use that path

internal/daemon/path.go: AugmentServicePath()
  └── Add filepath.Join(home, ".claude", "local") to candidate prepend list
  └── Covers session creation (exec.LookPath at PTY spawn time) not just detection
```

**Implementation approach for detect.go:**

The `knownCLIs` loop in `DetectCLIs` gains a per-CLI fallback mechanism. Rather than coupling the fallback directly into the loop, add an optional `FallbackPaths func() []string` field to `CLISpec`, or simply special-case "claude" after the loop:

```go
// After the LookPath loop, check native paths for claude specifically.
// Only if claude was not found via PATH.
```

Either approach is valid. The special-case is simpler; the `FallbackPaths` field is more extensible if other CLIs add native installers later. Given only Claude Code has a known native installer issue today, the simpler special-case is preferred (avoid the 3-example abstraction rule).

**Confidence note on paths:** The path `~/.claude/local/claude` is based on community reports as of early 2026. Verify against the actual Anthropic installer output on a test machine before shipping. Structure the code to easily add more paths.

---

## Component Dependency Map

```
WebServer Config (server.go)
  ← handleWebServerStart (api.go)
    ← WebServerStartRequest (types.go)
      ← client.StartWebServer (client.go)
        ← App.StartWebServer (app.go)
          ← Frontend: handleToggleServer (SettingsTab)

New in v1.11:
  selfcert.go → server.go (TLSConfig branch for local mode)
  auth.go → server.go (middleware wrap for local mode)
  localip helper → app.go (fallback IP resolution)
  App.StartWebServer → auto-toggle all existing sessions after start
  App.CreateSession → auto-toggle new session if server running
  session:web-enabled event → App.tsx state sync
```

---

## Build Order (Phase Dependencies)

**Phase ordering rationale:** Each feature is largely independent at the component level. Settings-as-tab affects the UI surface that will display the local-mode password, so it should land before the local network fallback UI work. The Claude Code detection fix and sidebar rename are purely isolated changes — safest to do first.

```
Phase A: Claude Code native path detection
  Files: internal/pty/detect.go, internal/daemon/path.go
  Risk: LOW — additive change, existing tests cover detect.go
  No frontend changes.
  Can ship alone.

Phase B: Sidebar label rename ("New Tab" → "New Session")
  Files: frontend/src/components/Sidebar.tsx, 1 test file (if asserting label text)
  Risk: VERY LOW — two string changes
  No backend changes.
  Can ship alone.

Phase C: Settings as sidebar tab
  Files: frontend/src/App.tsx, frontend/src/components/SettingsPanel.tsx,
         frontend/src/components/TabBar.tsx (type union), Sidebar.tsx (handler wiring)
  Risk: LOW — follows identical pattern to DaemonManagerPanel tab
  Recommended after Phase B so sidebar is fully consistent.
  No backend changes.

Phase D: Auto-serve sessions
  Files: app.go (StartWebServer + CreateSession), App.tsx (event handler)
  Risk: MEDIUM — changes behavior of session creation; must not break no-server case
  New Wails event: session:web-enabled
  No new backend packages.
  Recommend after Phase C so the result is visible in the Settings tab.

Phase E: Local network fallback (self-signed TLS + password)
  Files: internal/webserver/selfcert.go (NEW), internal/webserver/auth.go (NEW),
         internal/webserver/server.go, internal/daemon/types.go, internal/daemon/api.go,
         app.go (StartWebServer fallback branch), SettingsPanel.tsx (password display),
         App.tsx (nudge banner)
  Risk: HIGH — most complex; re-adds removed infrastructure; new middleware
  Depends on: Phase C (settings UI is a tab before adding password display to it)
              Phase D (auto-serve works before layering in local mode)
```

---

## Data Flow Changes

### Current: Web Server Start Flow
```
User clicks Start → SettingsPanel.handleToggleServer()
  → StartWebServer(port) [Wails binding]
    → app.go: check Tailscale health (must be Connected + HasCerts)
    → client.StartWebServer(h.IP, port, h.FQDN)
      → POST /webserver/start {ip, port, fqdn}
        → webserver.NewWebServer(Config{BindIP, Port, FQDN})
        → ws.Start() — tls.Listen on TS IP with lc.GetCertificate
```

### v1.11: Web Server Start Flow (with fallback + auto-serve)
```
User clicks Start (or future: auto-trigger on startup)
  → app.go: check Tailscale health
    → if healthy: existing Tailscale path (unchanged)
    → if not healthy:
        password = generatePassword()              // crypto/rand, 22 chars
        ip = resolveLocalIP()                      // net.InterfaceAddrs scan
        → client.StartWebServer(ip, port, ip, "local", password)
          → POST /webserver/start {ip, port, fqdn:ip, mode:"local", password}
            → webserver.NewWebServer(Config{BindIP, Port, FQDN, Mode:"local", Password})
            → ws.Start() — tls.Listen with selfcert TLSConfig + auth middleware wrap
  → After start (either mode): iterate ListSessions(), ToggleWebServing each
  → Emit session:web-enabled {sessionId, url} for each
  → Emit webserver:started {url, mode, password} to frontend
Frontend: receives webserver:started → updates webServerRunning, webServerMode, localPassword state
Frontend: receives session:web-enabled → updates webEnabled[id] and sessionURLs[id]
```

### v1.11: Session Creation Flow (with auto-serve)
```
createTab() → CreateSession() [Wails]
  → app.go: daemon.CreateSession(...)
  → check client.GetWebServerStatus()
  → if running: client.ToggleWebServing(newID, true)
  → Emit session:web-enabled {sessionId, url}
Frontend: receives event → webEnabled[sessionId] = true, sessionURLs[sessionId] = url
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Storing password on disk

**What people do:** Write the local-mode password to config dir for persistence across daemon restarts.
**Why it's wrong:** Self-signed TLS with a static persistent password becomes a weak permanent credential. The password should be regenerated each time the server starts.
**Do this instead:** Generate password in memory at `StartWebServer` call time; pass to daemon via the start request; stored only in the in-memory WebServer struct. If daemon restarts, server stops and user must re-enable (generating a new password).

### Anti-Pattern 2: Reusing CT disclosure gate for self-signed certs

**What people do:** Reuse the existing `ct_disclosed` sentinel file and `HasCTDisclosure` flow for local mode.
**Why it's wrong:** The CT disclosure is specifically about Let's Encrypt publishing hostnames to public Certificate Transparency logs. Self-signed certs have no CT log exposure. Gating local mode on the CT disclosure confuses users.
**Do this instead:** Local mode bypasses the CT check entirely. The SettingsTab Web Server section shows a different message for local mode: "Using self-signed certificate for local network access. Tailscale provides browser-trusted certificates without this limitation."

### Anti-Pattern 3: Making SettingsPanel a permanent tab entry

**What people do:** Add the Settings tab to the initial `tabs` state array so it always appears in the tab bar.
**Why it's wrong:** The pattern in this codebase (WelcomeTab, DaemonManagerPanel, RemoteSessionsPanel) is "singleton tab, find-or-add on demand." Making Settings permanent pollutes the initial tab bar.
**Do this instead:** Follow the existing singleton pattern: `handleOpenSettings` finds existing settings tab or adds it, then focuses. The tab can be closed and reopened via sidebar.

### Anti-Pattern 4: Frontend-side auto-serve decision

**What people do:** In `createTab()`, check the frontend `webServerRunning` state to decide whether to call `ToggleWebServing`.
**Why it's wrong:** Race condition — `webServerRunning` is stale React state. A new server might have just started, or the state might not reflect the daemon's current truth.
**Do this instead:** Have `app.go CreateSession` check `client.GetWebServerStatus()` on the backend side and auto-toggle there. Emit a `session:web-enabled` event to sync frontend state.

---

## Integration Summary Table

| Feature | New Files | Modified Files | New Wails Events |
|---------|-----------|----------------|-----------------|
| Claude Code paths | — | `internal/pty/detect.go`, `internal/daemon/path.go` | none |
| Sidebar rename | — | `Sidebar.tsx`, 1 test file | none |
| Settings as tab | — | `App.tsx`, `SettingsPanel.tsx`, `TabBar.tsx` | none |
| Auto-serve sessions | — | `app.go`, `App.tsx` | `session:web-enabled` |
| Local network fallback | `selfcert.go`, `auth.go` | `server.go`, `types.go`, `api.go`, `app.go`, `SettingsPanel.tsx`, `App.tsx` | `webserver:started` |

---

## Sources

- Direct code inspection: `/Users/ken/dev/agenthub/internal/webserver/server.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/webserver/tailscale.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/api.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/engine.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/types.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/pty/detect.go`
- Direct code inspection: `/Users/ken/dev/agenthub/internal/daemon/path.go`
- Direct code inspection: `/Users/ken/dev/agenthub/app.go`
- Direct code inspection: `/Users/ken/dev/agenthub/frontend/src/App.tsx`
- Direct code inspection: `/Users/ken/dev/agenthub/frontend/src/components/SettingsPanel.tsx`
- Direct code inspection: `/Users/ken/dev/agenthub/frontend/src/components/Sidebar.tsx`
- Project context: `/Users/ken/dev/agenthub/.planning/PROJECT.md`

---
*Architecture research for: AgentHub v1.11 — local network fallback, auto-serve, settings-as-tab, sidebar rename, Claude Code detection*
*Researched: 2026-04-08*
