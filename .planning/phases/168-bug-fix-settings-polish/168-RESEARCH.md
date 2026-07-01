# Phase 168: Bug Fix & Settings Polish - Research

**Researched:** 2026-07-01
**Domain:** Go daemon/relay/webserver (session sharing, capability model) + React/Wails frontend (Hub, Settings, StatusBar)
**Confidence:** HIGH (all claims below are code-verified with exact file:line citations; no external libraries or unfamiliar frameworks are involved)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**FIX-04 — Hub viewer count (#121)**
- D-01: Count only subscribers whose `Subscriber.Origin == "web"`. Exclude `Origin == "local"`. A never-shared local session reads 0 viewers.
- D-02: Use a raw connection count over web-origin subscribers — NOT a PersonKey (presence-roster) collapse.
- D-03: Add a new Hub method (e.g. `RemoteViewerCount()`) that returns count of subscribers where `Origin == "web"`; wire `daemon.SessionInfo.ViewerCount` (`engine.go:538`) to it instead of `hub.SubscriberCount()`. Update the `types.go:29` comment. `SubscriberCount()` itself stays as-is (still used elsewhere).

**FIX-02 — Disconnect viewers (#117)**
- D-04: A single "Disconnect all viewers" button in the Share modal — NOT a per-viewer list.
- D-05: The button force-closes all `Origin == "web"` connections for the session (new backend method iterating web-origin subscribers and closing each; `Subscriber.CloseSlow`-style close is the closest analog). Local-origin connections are never touched.
- D-06: Drop connections only — do NOT revoke the capability.
- D-07 (multi-viewer support): The relay Hub already stores subscribers in a `map[*Subscriber]struct{}` with no eviction on new subscribe. RESEARCH ITEM: confirm whether Phase 165's dual-origin fix already resolved the multi-viewer-kick end-to-end before scoping any relay-buffer/disconnect work; if it still reproduces, locate it in the webserver Tailscale path / cap single-use, not the hub.

**UX-01 — Stay on Hub after creating session (#116)**
- D-08: Toggle lives in Settings → Session Behavior (`id="settings-session-behavior"`).
- D-09: Default OFF.
- D-10: When ON, the create flow skips `setActiveId(newSessionId)` (`App.tsx:791`) but still creates the tab.
- D-11 (scope): Gates only the GUI Hub "New session" create path — the only auto-switch in the app.
- D-12: Persist following the `NotifyOnWaiting` end-to-end pattern (engine.go → api.go → client.go → app.go → SettingsTab.tsx).

**UX-02 — Footer "Share Session" button (#115)**
- D-13: Rename the footer button (`StatusBar.tsx:40`) from "Enable Web" to "Share Session".
- D-14: The button always opens the Hub Share modal for the currently-active session. Remove the footer's direct `ToggleWebServing` call.
- D-15: The button is hidden when the active tab is not a shareable local session (Hub, Settings, Help, Welcome, remote/web-session tabs).

**FIX-01 — /app/ guest plugin-config + SSE (#112)**
- D-16: Backend already ships both capability-gated endpoints (`GET /api/plugin-config`, `GET /api/plugin-config/stream`). Gap is purely client-side: `WebShareSessionView` never self-fetches. Fix: when running as a web guest, self-fetch `/api/plugin-config?cap=<capToken>` for initial config and subscribe to the SSE stream for hot-swap. Reuse the existing `capToken`.

**FIX-03 — remote session opens in-app tab (#118)**
- D-17: An in-app `web-session` tab path already exists (`openWebSessionTab` → mounts `WebShareSessionView`). Today `handleOpenRemoteSession` wrongly routes through `OpenRemoteSessionURL(id)` → `BrowserOpenURL(url)`. Fix: reroute to open the session in the in-app `web-session` tab. Keep the existing join-code/cap-exchange (`RemoteJoinCodeModal`, `intent='open-session'`) flow.

### Claude's Discretion
- Exact new-method names (`RemoteViewerCount`, disconnect endpoint shape), button styling, and precise wiring of the disconnect action.
- Whether the "Disconnect all viewers" button is always visible in the Share modal or only when `viewerCount > 0` (prefer showing only when there are web viewers to disconnect).

### Deferred Ideas (OUT OF SCOPE)
- Per-viewer disconnect list / viewer roster UI in the Share modal.
- Disconnect + cap revocation as one action.
- FIX-05 / #120 Tailscale connection detection — split into Phase 169.
- "Help Guide — document Tailscale Funnel admin prerequisites" todo — belongs to Phase 166 territory, not this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-01 | Web-share guests on `/app/` receive live plugin-config + SSE hot-swap | Backend endpoints confirmed capability-gated and pre-existing (`plugin_config.go`, `plugin_config_stream.go`). `connect-src 'self'` CSP already permits same-origin fetch/EventSource — no CSP change needed. Existing (currently skipped) Playwright suite `web-plugin-hot-swap.spec.ts` targets this exact feature but needs adaptation, not a straight un-skip. |
| FIX-02 | Multiple simultaneous remote viewers; Disconnect-all-viewers control | Exhaustive code search (hub.go, server.go, capability_mw.go, joincode.go) found **zero** eviction/kick mechanisms anywhere in the current codebase. Verdict below: the "kick" bug likely does not reproduce today. Disconnect-all-viewers is 100% new plumbing (no existing analog endpoint). |
| FIX-03 | Opening a remote session from the Hub opens an in-app tab | `openWebSessionTab`/`WebShareSessionView` render path is currently **mount-stable** (single global `webParams`, not per-tab) and hardcodes `wss://${window.location.host}` — both must change to support multiple concurrent remote-peer tabs with distinct hosts/caps. This is a larger change than the CONTEXT.md summary implies. |
| FIX-04 | Hub viewer count excludes internal WebSocket subscribers | `engine.go:538` is confirmed the sole call site feeding `SessionInfo.ViewerCount`. A second caller of `SubscriberCount()` exists (`relay/server.go:506`, `NotifyViewerCount`) but feeds an MsgMeta WS frame the frontend never parses — safe to leave untouched. |
| UX-01 | Settings toggle to stay on Hub after creating a session | `NotifyOnWaiting` (bool, zero-value default false) is the exact template — confirmed end-to-end at engine.go/api.go/client.go/app.go/SettingsTab.tsx. `autoCloseSession` (Session Behavior section) is the sibling UI pattern to copy visually. |
| UX-02 | Footer "Share Session" button opens Share modal | `SessionShareModal` is currently rendered **only inside `HubPanel.tsx`** (local `shareModalSession` state) — StatusBar (rendered per active tab, outside HubPanel) has no existing path to open it. State must be lifted to `App.tsx` or a second modal instance must be introduced. This is a concrete integration gap CONTEXT.md's canonical refs don't call out. |
</phase_requirements>

## Summary

This phase is six well-scoped bug fixes/UX polish items against a codebase whose relevant seams (relay hub, capability grants, plugin-config endpoints, Settings toggle pattern) are already in good shape from Phases 152–167. The highest-value research finding is on **FIX-02/D-07**: an exhaustive read of `internal/relay/hub.go`, `internal/webserver/server.go` (WSS handler), `internal/webserver/capability_mw.go`, and `internal/capability/joincode.go` found **no code path anywhere that evicts, closes, or invalidates an existing viewer's connection when a second viewer subscribes**. Grants are additive (`ws.grants[sessionID]` is a set, never replaced), join codes are single-use only for the *code* (not the resulting capability token), and `Hub.Subscribe`/`Unsubscribe` never touch other subscribers. Phase 165's dual-origin fix (FNL-04) only *added* a secondary Funnel-origin branch to `requireAllowedOrigin` — it did not remove or add any single-viewer restriction, so it is very unlikely to be the actual fix for #117's Part A (if that fix happened at all, it must have happened in an earlier, unremarked phase, or the original bug report's root cause was never actually in the paths investigated). The pragmatic recommendation: treat FIX-02's backend eviction concern as **not reproducing in current code**, add a live/manual two-viewer smoke test to confirm empirically (the original issue explicitly recommends this over static analysis), and focus the phase's coding effort on the two things that are unambiguously still missing: the Disconnect-all-viewers button (D-04..D-06, entirely new) and FIX-04's viewer-count fix.

The second major finding is that **FIX-01 and FIX-03 are architecturally coupled** in a way CONTEXT.md's per-decision framing doesn't surface: `WebShareSessionView.tsx` hardcodes `wss://${window.location.host}` (line 57) and the surrounding `App.tsx` render branch (`activeId.startsWith('__websession__')`) reads a single mount-stable `webParams` object rather than per-tab state. FIX-01's client-side self-fetch and FIX-03's in-app remote tab both need a `baseURL`-style override so the component can talk to an arbitrary remote peer's origin instead of `window.location.host` — and FIX-03 additionally needs the tab/params resolution to become per-`sessionId` instead of global-singleton, since a GUI user could open two different remote sessions (different hosts, different caps) in two different in-app tabs. Sequencing FIX-01 before FIX-03 without this shared consideration risks a rework.

**Primary recommendation:** Ship FIX-04 first (smallest, most isolated — one Hub method + one call-site swap), then FIX-01 designed with a `baseURL` override from day one, then FIX-03 building on that override, then UX-01/UX-02 (independent, template-driven), and land FIX-02's Disconnect button + a live two-viewer regression test last (its "verify eviction still happens" research question is resolved as "does not reproduce" below, but empirical confirmation is cheap insurance).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Viewer counting (FIX-04) | API/Backend (Go daemon `engine.go`) | — | `SessionInfo.ViewerCount` is computed server-side from `relay.Hub` state; frontend only renders the number `SessionCard.tsx:500`. |
| Viewer eviction / disconnect (FIX-02) | API/Backend (Go relay `hub.go` + webserver) | Frontend (Share modal button) | Force-close must happen where the WS connections actually live (webserver process); the button is a thin RPC trigger. |
| Plugin-config hot-swap (FIX-01) | Browser/Client (React, `WebShareSessionView.tsx`) | API/Backend (existing capability-gated endpoints, no change) | The backend already serves the data; the gap is entirely in the client's failure to consume it. |
| Remote session open (FIX-03) | Browser/Client (React `App.tsx`, `WebShareSessionView.tsx`) | API/Backend (existing join-code/cap-exchange, no change) | Routing decision (in-app tab vs. external browser) and the WS/API base-URL override are pure frontend concerns; backend endpoints are reused unmodified. |
| Settings toggle persistence (UX-01) | API/Backend (`daemonSettings` in `engine.go`) | Frontend (Settings UI) | Mirrors `NotifyOnWaiting`: server is the source of truth; GUI is a thin Get/Set client. |
| Footer Share button (UX-02) | Browser/Client (`StatusBar.tsx`, `App.tsx` state lift) | — | Pure UI reorganization — no backend involvement; the existing `SessionShareModal` and `ToggleWebServing`/`SetSessionFunnel` RPCs are reused unchanged. |

## Standard Stack

No new external dependencies are required for this phase. All six fixes are implemented with the existing stack:

### Core (unchanged)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` + `nhooyr.io/websocket` (existing) | go 1.26.3 (`go.mod:3`) | WS relay/webserver transport | Already in use throughout `internal/relay`, `internal/webserver` — no reason to introduce anything new for a bug-fix phase. |
| React 18 + Wails v2 bindings (existing) | per `frontend/package.json` (unchanged) | Frontend/Settings/Hub UI | Already the whole frontend stack; this phase only edits existing components. |

### Alternatives Considered
Not applicable — no new library decision points in this phase.

**Installation:** none — no `npm install` / `go get` needed.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero external packages (Go or npm). All work is edits to existing files within `internal/relay`, `internal/webserver`, `internal/daemon`, `frontend/src/App.tsx`, `frontend/src/components/StatusBar.tsx`, `frontend/src/components/SettingsTab.tsx`, and `frontend/src/components/Hub/*`.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────────┐
                         │              Desktop GUI (Wails)             │
                         │                                               │
   Hub "New session" ──▶ │ createTab() ──▶ setActiveId() [UX-01: gated] │
                         │                                               │
   Footer "Share         │ StatusBar.onToggleWeb ──▶ [UX-02: now opens] │
    Session" button ───▶ │        openShareModalForActiveSession()      │──┐
                         │                                               │  │
   Hub card "Open" ────▶ │ handleOpenRemoteSession() [FIX-03: now opens] │  │
    (remote session)     │        openWebSessionTab(id, baseURL, cap)    │  │
                         └───────────────────────┬───────────────────────┘  │
                                                  │                          │
                                                  ▼                          ▼
                         ┌─────────────────────────────────────────────────────┐
                         │        SessionShareModal (single source of truth)   │
                         │  - ToggleWebServing / SetSessionFunnel (unchanged)   │
                         │  - viewerCount display (FIX-04, from SessionInfo)    │
                         │  - [NEW] "Disconnect all viewers" button (FIX-02)    │
                         └───────────────────────┬─────────────────────────────┘
                                                  │ RPC (daemon api.go)
                                                  ▼
   ┌───────────────────────────── Go daemon process ─────────────────────────────┐
   │                                                                               │
   │  engine.go:538 ListSessions()                                               │
   │     viewerCount = hub.RemoteViewerCount()  [FIX-04: was SubscriberCount()]   │
   │                          │                                                   │
   │                          ▼                                                   │
   │  internal/relay/hub.go   Hub.subscribers map[*Subscriber]struct{}            │
   │     Subscribe/Unsubscribe (no eviction — confirmed, see D-07 verdict)        │
   │     [NEW] DisconnectWebViewers(): iterate Origin=="web", call CloseSlow-     │
   │           style close on each (FIX-02)                                       │
   │                          ▲                                                   │
   │                          │ hub.Subscribe(sub) per WS connection              │
   │  internal/webserver/server.go                                               │
   │     GET /sessions/{id}/ws  (Origin allowlist → requireCapability → Subscribe)│
   │     GET /api/plugin-config (existing, capability-gated) ◀── [FIX-01: self-  │
   │     GET /api/plugin-config/stream (SSE, existing)          fetch added here]│
   │     GET /app/  (SPA, NOT wrapped in cspHeaders — see Pitfall below)         │
   └───────────────────────────────────────────────────────────────────────────┘
                                                  ▲
                                                  │ fetch + EventSource (same-origin)
                         ┌────────────────────────┴──────────────────────────┐
                         │   Browser guest: WebShareSessionView.tsx           │
                         │   [FIX-01] self-fetches /api/plugin-config?cap=…   │
                         │   [FIX-01] EventSource(/api/plugin-config/stream)  │
                         │   [FIX-03] needs baseURL override (not always      │
                         │            window.location.host — see Pitfall)     │
                         └─────────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new directories. Files touched:
```
internal/relay/hub.go              # RemoteViewerCount(), DisconnectWebViewers() (FIX-04, FIX-02)
internal/daemon/engine.go           # engine.go:538 call-site swap (FIX-04); new bool setting (UX-01)
internal/daemon/types.go            # ViewerCount comment update (FIX-04); Settings struct field (UX-01)
internal/daemon/api.go              # new disconnect route (FIX-02); Get/Set routes (UX-01)
internal/daemon/client.go           # DaemonClient wrapper (UX-01, FIX-02)
app.go                              # bound methods (UX-01, FIX-02)
frontend/src/components/Hub/SessionShareModal.tsx   # Disconnect button (FIX-02); viewerCount display
frontend/src/components/Hub/WebShareSessionView.tsx # self-fetch+SSE (FIX-01); baseURL prop (FIX-03)
frontend/src/App.tsx                # createTab gating (UX-01); handleOpenRemoteSession rewrite (FIX-03);
                                     # shareModalSession state lift (UX-02)
frontend/src/components/StatusBar.tsx   # button rename/behavior (UX-02)
frontend/src/components/SettingsTab.tsx # new toggle in Session Behavior section (UX-01)
frontend/src/components/Hub/HubPanel.tsx # shareModalSession becomes a controlled prop (UX-02)
```

### Pattern 1: Origin-filtered Hub counting/disconnect (FIX-04, FIX-02)

**What:** Add Hub methods that filter `h.subscribers` by `sub.Origin == "web"` rather than returning the raw map length.
**When to use:** Anywhere "real remote viewers" (as opposed to the app's own internal terminal/chat/status-watcher connections) needs to be counted or acted upon.
**Example (existing pattern this must mirror, `internal/relay/hub.go:218-223`):**
```go
// SubscriberCount returns the number of currently subscribed clients. (MC-04)
func (h *Hub) SubscriberCount() int {
    h.mu.Lock()
    defer h.mu.Unlock()
    return len(h.subscribers)
}
```
New `RemoteViewerCount()` follows the identical lock discipline, filtering on `Origin`:
```go
func (h *Hub) RemoteViewerCount() int {
    h.mu.Lock()
    defer h.mu.Unlock()
    n := 0
    for sub := range h.subscribers {
        if sub.Origin == "web" {
            n++
        }
    }
    return n
}
```
`DisconnectWebViewers()` needs the **unlock-before-IO discipline** already established by `ResizeClient`/`HandleInject` (`hub.go:274`, `hub.go:611-613`): collect the matching subscribers under `h.mu`, release the lock, then call each one's close mechanism outside the lock (calling a subscriber's close callback while holding `h.mu` risks deadlock if the callback's cleanup path calls back into the Hub, e.g. via `Unsubscribe`).

### Pattern 2: `NotifyOnWaiting`-style settings toggle (UX-01)

**What:** A `bool` field on `daemonSettings`, zero-value = OFF, no defaults-merge needed.
**When to use:** Any new Settings → Session Behavior toggle that defaults OFF.
**Example (exact template, confirmed live in the codebase):**
```go
// internal/daemon/engine.go:115
NotifyOnWaiting bool `json:"notifyOnWaiting,omitempty"` // Phase 167 NTF-04: default OFF; zero-value is the correct default (no defaults-merge needed)

// internal/daemon/engine.go:1107-1120 (GetNotifyOnWaiting/SetNotifyOnWaiting)
func (e *SessionEngine) GetNotifyOnWaiting() bool { ... }
func (e *SessionEngine) SetNotifyOnWaiting(val bool) { ... }
```
```go
// internal/daemon/api.go:125-126, 874-886
a.mux.HandleFunc("GET /settings/notify-on-waiting", a.handleGetNotifyOnWaiting)
a.mux.HandleFunc("PATCH /settings/notify-on-waiting", a.handleSetNotifyOnWaiting)
```
```go
// internal/daemon/client.go:166-178 — DaemonClient.GetNotifyOnWaiting / SetNotifyOnWaiting
```
```go
// app.go:727-752 — App.GetNotifyOnWaiting / App.SetNotifyOnWaiting (bound Wails methods)
```
```tsx
// frontend/src/components/SettingsTab.tsx:121-124, 395-409, 526-548
const [notifyOnWaiting, setNotifyOnWaiting] = useState(false)
// ... handleToggleNotifyOnWaiting() calls SetNotifyOnWaiting(next)
```
The new UX-01 setting (working name `stayOnHubAfterCreate` or similar) follows this exact chain, but is rendered under `id="settings-session-behavior"` (SettingsTab.tsx:557) alongside `autoCloseSession` (SettingsTab.tsx:558-582) — **not** under `id="settings-behavior"` where `notifyOnWaiting` itself lives (SettingsTab.tsx:496, per the Phase 167 locked correction). Do not conflate the two sections.

### Pattern 3: Same-origin self-fetch + SSE from a web-share component (FIX-01)

**What:** `WebShareSessionView` already builds a same-origin WS URL from `capToken`; the same technique applies to a plain fetch and an `EventSource`.
**Example (existing WS-URL construction to mirror, `WebShareSessionView.tsx:57-58`):**
```tsx
const wsURL = `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
const apiBaseURL = window.location.origin
```
The FIX-01 addition (illustrative, matches existing project conventions — `useEffect` + `fetch`/`EventSource`, cleanup on unmount):
```tsx
useEffect(() => {
  if (!isWebGuest) return // only self-fetch when there's no pluginConfig prop from Wails
  const ctrl = new AbortController()
  fetch(`${apiBaseURL}/api/plugin-config?cap=${encodeURIComponent(capToken)}`, { signal: ctrl.signal })
    .then((r) => (r.ok ? r.json() : null))
    .then((cfg) => cfg && setLivePluginConfig(cfg))
    .catch(() => {})
  const es = new EventSource(`${apiBaseURL}/api/plugin-config/stream?cap=${encodeURIComponent(capToken)}`)
  es.addEventListener('plugin-config', (ev) => {
    try { setLivePluginConfig(JSON.parse((ev as MessageEvent).data)) } catch {}
  })
  return () => { ctrl.abort(); es.close() }
}, [capToken, apiBaseURL])
```
**Backend endpoints are unchanged** — `internal/webserver/plugin_config.go:handleGetPluginConfig` and `internal/webserver/plugin_config_stream.go:handleStreamPluginConfig` are already capability-gated (`requireCapability`, registered at `server.go:862,867`) and were built exactly for this consumption pattern (Phase 93 PLUG-04).

### Pattern 4: Lifting modal-open state so two entry points share one modal instance (UX-02)

**What:** `SessionShareModal`'s open/closed state (`shareModalSession`) currently lives **only** inside `HubPanel.tsx` (`HubPanel.tsx:281-300`), triggered by `handleShare(session)` from a card click. `StatusBar.tsx` is rendered per-tab in `App.tsx` (line 1791), outside `HubPanel`'s subtree, so it has no way to reach that state today.
**Fix approach:** Lift `shareModalSession`/`setShareModalSession` to `App.tsx` (which already holds `sessions`, `webServerMode`, `webServerRunning`, `shellWebShareWarned`, etc. — everything `SessionShareModal` needs per its existing prop list, `HubPanel.tsx:586-596`), pass it into `HubPanel` as a controlled prop (replacing its local `useState`), and give `StatusBar`'s new handler a way to call `setShareModalSession(sessions.find(s => s.id === activeSessionId))`.
**Why this matters:** Without this lift, the naive implementation would either duplicate `<SessionShareModal>` (two mounted instances, double RPC polling / two sources of truth — the exact class of bug #115 exists to fix) or the footer button would silently do nothing outside the Hub tab.

### Anti-Patterns to Avoid
- **Re-deriving the multi-viewer-kick fix from scratch:** the code shows no eviction logic exists anywhere. Do not add new "single active viewer" enforcement — that would be introducing the bug the issue complains about, not fixing it.
- **Duplicating `<SessionShareModal>`:** see Pattern 4 above — lift state, don't clone the component.
- **Hardcoding `window.location.host` inside a component meant to serve both native web-share guests and in-app remote-peer tabs:** breaks FIX-03 the moment two different peers are opened.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Origin-filtered subscriber iteration | A parallel subscriber list keyed by origin | Loop over the existing `h.subscribers` map filtering `sub.Origin` (Pattern 1) | The map is already the single source of truth; a parallel structure risks drift on Subscribe/Unsubscribe. |
| Force-closing a WS connection | A new "kill switch" channel/flag on `Subscriber` | `Subscriber.CloseSlow` is already the exact "server closes this connection" mechanism (`hub.go:40-42`, invoked today only on slow-consumer backpressure) — reuse the same field, just invoke it from a new code path instead of only from the backpressure `select/default` branch. | Consistent close semantics (WS status code, cleanup via existing `defer hub.Unsubscribe`) without inventing a second termination mechanism. |
| Settings persistence | A new frontend-only localStorage setting | The `daemonSettings` Get/Set RPC chain (Pattern 2) | Every other Settings toggle in this codebase is server-truth-seeded; a frontend-only setting for UX-01 would be the first exception and would not survive a daemon restart or be visible to other GUI instances of the same daemon. |
| CSP allowance for the new fetch/SSE calls | A new CSP directive edit | Nothing — `connect-src 'self'` on `handleTerminalPage`'s CSP (`csp_mw.go:115`) already covers same-origin fetch/EventSource | See Common Pitfall below regarding `/app/` specifically not having this header at all currently. |

**Key insight:** Every one of these six fixes has a same-shape sibling already merged in this codebase (viewer-count filtering has no sibling, but the Origin field and lock discipline it needs already exist; settings toggles have two near-identical siblings; same-origin fetch/SSE has the WS-URL sibling; modal state-lifting has the `activeGroupId`/`groupDefs` POL-05 precedent already visible in `HubPanel.tsx`'s prop list). The phase's engineering risk is almost entirely in wiring, not new design.

## Common Pitfalls

### Pitfall 1: `/app/` has no CSP header at all today
**What goes wrong:** Verifying "no CSP errors in console" (FIX-01 success criterion 1) could pass trivially for the wrong reason.
**Why it happens:** `ws.cspHeaders(...)` middleware is wired on exactly three routes: `GET /dashboard`, `GET /join`, and `GET /sessions/{id}` (`handleTerminalPage`) — confirmed by grep, `server.go:820,825,930`. The `GET /app/` route (`server.go:985`, the actual SPA guests land on after the Phase 159 redirect) is a raw `mux.HandleFunc` with **no `cspHeaders` wrapper**. `/sessions/{id}` today only issues a redirect to `/app/?session=…&cap=…` (`server.go:1250`) and no longer serves page content directly.
**How to avoid:** Don't assume "no CSP violations observed" proves anything about the fetch/SSE additions — there is currently no CSP enforcement on the page guests actually load. This is a **pre-existing gap outside this phase's locked scope** (D-16 is explicitly client-only) — flag it, do not silently fix it (adding `cspHeaders` to `/app/` is a larger security-hardening change affecting the whole SPA bundle, untested against Vite's asset hashes, and not one of the six locked success criteria). Recommend filing a follow-up issue rather than expanding scope here.
**Warning signs:** A verifier who checks "0 console CSP errors" and calls it done without checking whether a `Content-Security-Policy` response header is present on `/app/` at all.

### Pitfall 2: FIX-01 and FIX-03 will collide on `WebShareSessionView`'s hardcoded origin
**What goes wrong:** FIX-01's self-fetch, if written using `window.location.origin`/`window.location.host` directly (mirroring the existing `wsURL` line), will silently point at the wrong host when the same component is reused for FIX-03's in-app remote-peer tab (where `window.location` is the Wails webview's own origin, not the remote peer's).
**Why it happens:** `WebShareSessionView.tsx:57-58` hardcodes `window.location.host` / `window.location.origin`. This is correct for the "guest opened the raw share URL in a browser" case but wrong for "desktop GUI user is viewing someone else's session in an in-app tab" case that FIX-03 introduces.
**How to avoid:** Add an optional `baseURL` prop to `WebShareSessionView` from the start (used by FIX-01's fetch/SSE construction and by the existing `wsURL`/`apiBaseURL` lines), defaulting to `window.location.origin` when absent (native web-share guest case unchanged) and supplied explicitly by FIX-03's remote-tab render path.
**Warning signs:** FIX-03 manual UAT shows a blank/error terminal because the plugin-config fetch (FIX-01) 404s or CORS-fails against the wrong origin, even though the WS terminal itself works (because `wsURL` was already parameterized correctly per-instance via the `TerminalPanel`/`ChatPanel` props).

### Pitfall 3: The remote-session tab/param resolution is currently a mount-stable singleton, not per-tab
**What goes wrong:** A straightforward "just call `openWebSessionTab(session.id)` instead of `BrowserOpenURL`" implementation will work for the *first* remote session opened, then silently reuse the *same* `capToken`/host for a second, different remote session.
**Why it happens:** `App.tsx:1581-1588`'s render branch (`activeId.startsWith('__websession__')`) resolves `sessionId`/`capToken` from a single mount-stable `webParams` object (populated once from the app's own web-mode URL params, `App.tsx:113`), not from per-tab state. This is correct today because there is exactly one `web-session` tab possible (the app's own web-share bootstrap, Phase 155-03). FIX-03 introduces the possibility of *multiple* `web-session` tabs (one per opened remote peer session), each needing its own `sessionId`/`capToken`/`baseURL`.
**How to avoid:** Extend the `Tab` type (or a side-map keyed by tab id) to carry `capToken`/`baseURL` per remote-session tab, and change the render branch to read from the tab object rather than the global `webParams`. `openWebSessionTab`'s signature will need to grow beyond `(sessionId: string)`.
**Warning signs:** Opening two different remote sessions from the Hub and having the second one silently render the first one's content, or fail auth entirely.

### Pitfall 4: `M-13` in TESTING.md documents the *current* (pre-fix) FIX-03 behavior and must be updated, not left stale
**What goes wrong:** After FIX-03 ships, `TESTING.md` Category G / M-13 ("Open in browser" opens `RemoteJoinCodeModal` → external browser at `baseURL/sessions/{id}?cap=TOKEN`) will describe behavior that no longer exists, silently becoming a wrong regression record.
**How to avoid:** Update M-13's expected behavior to "opens in an in-app tab" as part of this phase's TESTING.md wiring (per the project's standing convention in `./CLAUDE.md`), not just add new entries elsewhere.

### Pitfall 5: The existing `web-plugin-hot-swap.spec.ts` Playwright suite targets a retired UMD-global mechanism
**What goes wrong:** Naively removing the `test.describe.skip` wrapper will likely still fail, for reasons unrelated to whether FIX-01 is correctly implemented.
**Why it happens:** The skipped suite (`frontend/e2e/web-plugin-hot-swap.spec.ts`, header comment lines 1-16) was written against the old raw `/sessions/{id}` terminal.js viewer, hooking a UMD global (`window.WebglAddon =`) to detect addon instantiation. The current `/app/` SPA is Vite-bundled React — there is no guarantee the same UMD-global assignment pattern exists for how `TerminalPanel.tsx` loads xterm addons in the SPA build.
**How to avoid:** Treat this file as a **reference for what to test** (initial fetch → webgl addon load; SSE push → live disable without reload), not as a mechanical re-enable target. Verify how `TerminalPanel.tsx` actually loads/gates the WebGL addon in the current SPA before deciding whether the existing hook technique still applies or needs a rewrite.

## Code Examples

### Existing capability-gated plugin-config routes (FIX-01 backend, no changes needed)
```go
// internal/webserver/server.go:862,867
mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))
mux.HandleFunc("GET /api/plugin-config/stream", ws.requireCapability(ws.handleStreamPluginConfig))
```

### CSP header actually sent on `/sessions/{id}` (NOT on `/app/` — Pitfall 1)
```go
// internal/webserver/csp_mw.go:113-117
b.WriteString("script-src 'self' 'wasm-unsafe-eval'; ")
b.WriteString("style-src 'self' 'unsafe-inline'; ")
b.WriteString("connect-src 'self' ")
b.WriteString(wssOrigin)
b.WriteString("; ")
```

### Grant model confirming "no eviction" (FIX-02/D-07 evidence)
```go
// internal/webserver/server.go:303-332
func (ws *WebServer) AddGrant(sessionID, grantID string) {
    ws.mu.Lock()
    if ws.grants[sessionID] == nil {
        ws.grants[sessionID] = make(map[string]struct{})
    }
    ws.grants[sessionID][grantID] = struct{}{}   // ADDITIVE — never replaces an existing entry
    ws.mu.Unlock()
}
```

### Join-code single-use scope (FIX-02/D-07 evidence — the code, not the resulting cap, is single-use)
```go
// internal/capability/joincode.go:79-92 (Exchange)
entry, ok := m.codes[code]
if !ok { return "", ErrCodeNotFound }
if m.now().After(entry.expiry) { delete(m.codes, code); return "", ErrCodeExpired }
delete(m.codes, code)          // the CODE is consumed here
return entry.token, nil        // the returned TOKEN has no further single-use restriction
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Raw `/sessions/{id}` terminal.js viewer served plugin-config via its own fetch+SSE | `/sessions/{id}` redirects to `/app/` React SPA, which gets plugin-config only via Wails RPC (desktop) — web guests got no path | Phase 159 (WEBCHAT-01) | This IS Issue #112 / FIX-01 — the redirect fixed chat parity but silently dropped plugin-config parity for web guests, tracked as a "known, chat-orthogonal cross-surface gap" in the `web-plugin-hot-swap.spec.ts` header comment itself. |
| `handleOpenRemoteSession` opened remote sessions via `BrowserOpenURL` (system browser) | (this phase) routes through the in-app `web-session` tab | Phase 146 (D-02 out-of-band redesign) established `BrowserOpenURL`; Phase 168 changes it | FIX-03 is a deliberate reversal of the Phase 146 "out-of-band, external browser" design in favor of in-app viewing, now that `WebShareSessionView` exists (it didn't exist yet at Phase 146). |

**Deprecated/outdated:**
- The Phase 146 "open remote session in system browser" design (`BrowserOpenURL`) is being superseded by FIX-03's in-app tab — this is an intentional, locked (D-17) reversal, not a regression.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The FIX-02 multi-viewer-kick bug (#117 Part A) does not currently reproduce, based on exhaustive static analysis of hub.go/server.go/capability_mw.go/joincode.go finding no eviction code path | Summary, Common Pitfalls | If a live two-guest test still shows one viewer getting dropped, the root cause is somewhere not yet inspected (e.g. a websocket library-level connection limit, an OS/browser session-cookie collision, or a client-side `remoteCapsCached`-driven reconnect loop) and would need fresh live-debugging rather than a code-only fix. Mitigated by explicitly recommending a live two-viewer smoke test rather than declaring the bug closed from code-reading alone. |
| A2 | `NotifyViewerCount`'s MsgMeta broadcast (`relay/server.go:506`, using raw `SubscriberCount()`) is never consumed by the frontend, so it is safe to leave un-filtered when FIX-04 changes the Hub-card viewer count | Standard Stack / FIX-04 phase requirement row | If some frontend code path does parse MsgMeta viewer-count frames (not found via `grep -rn "MsgMeta\|MetaPayload"` across `frontend/src`), leaving it unfiltered would produce a visible inconsistency between the Hub card (fixed, web-only count) and any live in-terminal viewer-count indicator (still raw, includes local). Low risk — the grep found zero matches. |

**If this table is empty:** N/A — see above.

## Open Questions

1. **Is there a genuinely reproducing FIX-02 backend bug at all, or was #117 always primarily the frontend `remoteCapsCached` desync?**
   - What we know: The issue's own "Investigation notes" section says "root cause is NOT yet pinned" and lists three *candidate* causes, none of which are borne out by the current code (RemoteCapStore is a different, unrelated per-consumer-daemon store used for GUI-to-GUI remote session opening — FIX-03 territory — not a per-viewer eviction mechanism on the owner's webserver).
   - What's unclear: Whether the original bug reporter's live repro (two real browser clients) would still show the "second viewer kicks first" symptom today, given the amount of relay/webserver work landed since (Phases 152-165).
   - Recommendation: Treat this as empirically open. Add a live/manual UAT item (two browser tabs, same share link, same session) as this phase's acceptance gate for the "still reproduces?" question, rather than asserting it's fixed from code alone. If it does reproduce live, the next place to look (not yet inspected) is the websocket library's `Accept`/connection-handling layer and any OS/browser same-tab-reuse behavior — not the Hub or grant model.

2. **Should `/app/`'s missing CSP header be fixed in this phase?**
   - What we know: `cspHeaders` is wired to exactly 3 routes and NOT `/app/` — the page guests actually load has zero CSP enforcement today (Pitfall 1).
   - What's unclear: Whether this was an intentional Phase 159 decision (SPA static assets served via `http.FileServerFS`, harder to compute a per-request nonce/hash-friendly policy) or an oversight.
   - Recommendation: Out of scope for this phase (D-16 is explicitly client-only); flag as a follow-up issue candidate rather than silently expanding FIX-01.

## Environment Availability

Skipped — this phase has no new external tool/service dependencies beyond the existing Go toolchain, Node/pnpm, and the already-running daemon/relay/webserver processes used throughout local development in this repo.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Go framework | stdlib `testing`, `go test -race -short ./...` |
| vitest | `cd frontend && pnpm test` |
| Playwright e2e | `cd frontend && pnpm exec playwright test` |
| Config files | none new — existing `go.mod`, `frontend/vitest.config.ts` (or equivalent), `frontend/playwright.config.ts` |
| Quick run command (Go, scoped) | `go test ./internal/relay/... ./internal/daemon/... ./internal/webserver/... -run 'TestRemoteViewerCount|TestDisconnectWebViewers|TestHandleSetStayOnHub|TestHub.*MultiViewer' -v` |
| Full suite command | `go test -race -short ./...` && `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIX-04 | `RemoteViewerCount()` counts only `Origin=="web"` subscribers; a local-only session reads 0 | unit (Go) | `go test ./internal/relay/... -run TestRemoteViewerCount -v` | ❌ Wave 0 — new `internal/relay/hub_test.go` (or extend existing) case |
| FIX-04 | `engine.go` ListSessions wires `ViewerCount` from the new method, not `SubscriberCount()` | unit (Go) | `go test ./internal/daemon/... -run TestListSessions_ViewerCount -v` | ❌ Wave 0 |
| FIX-02 | Disconnect-all-viewers closes every `Origin=="web"` subscriber, leaves `Origin=="local"` untouched | unit (Go) | `go test ./internal/relay/... -run TestDisconnectWebViewers -v` | ❌ Wave 0 |
| FIX-02 | Two concurrent `Origin=="web"` subscribers both continue receiving frames (no eviction) | unit (Go), regression guard for A1 | `go test ./internal/relay/... -run TestHub_TwoWebOriginSubscribers_NoEviction -v` | ❌ Wave 0 — new regression test recommended even though code review suggests it already passes |
| FIX-02 | Two real browser viewers of the same live share link — does either get dropped? | manual/live | n/a (see TESTING.md Category G precedent for a similar two-Mac live test) | manual-only — new M-NN item |
| FIX-01 | Web guest self-fetches `/api/plugin-config` on load and applies it | unit (vitest) | `cd frontend && pnpm test -- WebShareSessionView.plugin-config` | ❌ Wave 0 — new `frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx` |
| FIX-01 | SSE push updates live plugin config without reload | e2e (Playwright) — adapt, don't blind-un-skip | see Pitfall 5 | ⚠️ exists but needs rewrite: `frontend/e2e/web-plugin-hot-swap.spec.ts` |
| FIX-03 | `handleOpenRemoteSession` opens an in-app `web-session` tab instead of `BrowserOpenURL` | unit (vitest) | `cd frontend && pnpm test -- App.handleOpenRemoteSession` | ❌ Wave 0 |
| FIX-03 | Two different remote sessions open in two independent in-app tabs with correct per-tab cap/host | unit (vitest) + manual (two-Mac live, replaces M-13) | see TESTING.md Category G | ⚠️ M-13 needs rewrite, not new addition |
| UX-01 | New Session-Behavior toggle persists via daemon Get/Set, default OFF | unit (Go + vitest) | mirrors `TestNotifyOnWaiting*` (`engine_notify_test.go`, `api_notify_test.go`) + `SettingsTab.notify-toggle.test.tsx` pattern | ❌ Wave 0 |
| UX-01 | Toggle ON skips `setActiveId` on Hub-created session; tab still created | unit (vitest) | `cd frontend && pnpm test -- App.createTab.stayOnHub` | ❌ Wave 0 |
| UX-02 | Footer button renamed, always opens Share modal, hidden on non-shareable tabs | unit (vitest) | `cd frontend && pnpm test -- StatusBar.shareSession` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** scoped `go test ./internal/relay/... ./internal/daemon/...` / `pnpm test -- <touched file>`
- **Per wave merge:** `go test -race -short ./...` && `cd frontend && pnpm test`
- **Phase gate:** Full suite green (`go test -race -short ./...`, `pnpm test`, `pnpm exec playwright test`) before `/gsd-verify-work`, plus `bash tests/check-traceability-paths.sh`.

### Wave 0 Gaps
- [ ] `internal/relay/hub_test.go` (or new file) — `TestRemoteViewerCount`, `TestDisconnectWebViewers`, `TestHub_TwoWebOriginSubscribers_NoEviction` — covers FIX-04, FIX-02
- [ ] `internal/daemon/engine_test.go` extension or new `engine_stayonhub_test.go` — mirrors `engine_notify_test.go` pattern — covers UX-01
- [ ] `internal/daemon/api_test.go` extension or new `api_stayonhub_test.go` — mirrors `api_notify_test.go` pattern — covers UX-01
- [ ] `frontend/src/components/Hub/__tests__/WebShareSessionView.plugin-config.test.tsx` — covers FIX-01
- [ ] `frontend/src/components/__tests__/App.handleOpenRemoteSession.test.tsx` (or extend existing App test file) — covers FIX-03
- [ ] `frontend/src/components/__tests__/StatusBar.shareSession.test.tsx` (rename/extend existing StatusBar tests if any) — covers UX-02
- [ ] `frontend/src/components/__tests__/SettingsTab.stay-on-hub-toggle.test.tsx` — mirrors `SettingsTab.notify-toggle.test.tsx` — covers UX-01
- [ ] `frontend/e2e/web-plugin-hot-swap.spec.ts` — needs a rewrite pass (verify addon-loading mechanism in the current SPA before deciding to reuse the UMD-hook technique) — covers FIX-01
- [ ] TESTING.md Category G / M-13 — needs rewording (not a new item) to reflect in-app-tab behavior — covers FIX-03
- [ ] New manual M-NN — live two-browser-viewer smoke test — covers FIX-02/D-07 empirical confirmation

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | No auth changes — capability-token model unchanged. |
| V3 Session Management | Yes (FIX-02) | Force-disconnect must only affect `Origin=="web"` subscribers and must never revoke the underlying capability grant (D-06, locked) — this is a session-termination control, not a session-authentication control. |
| V4 Access Control | Yes (FIX-01, FIX-02) | Both the new plugin-config self-fetch and the disconnect action reuse the existing `requireCapability` middleware / capability-token check — no new access-control surface is introduced; the disconnect RPC on the daemon side must itself be gated so only the session's owner (local Wails app / daemon-authenticated caller) can trigger it, never a web guest. |
| V5 Input Validation | N/A | No new user-supplied input parsing beyond what `requireCapability` already validates. |
| V6 Cryptography | No | No cap-token minting/signing changes in this phase — FIX-02 explicitly does NOT touch cap issuance (D-06). |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| A footer "Share Session" button accidentally exposed on a non-owner-controlled surface (e.g. a remote/web-session tab) | Elevation of Privilege | D-15 already locks the hide condition — verify the implementation checks tab `type` (not just `sessionId` presence) so it can't render on a `web-session` or `file-browser` tab. |
| Disconnect-all-viewers RPC callable by a web guest (not just the owning desktop app) | Elevation of Privilege | New disconnect route must sit behind the same trust boundary as `ToggleWebServing`/`SetSessionFunnel` (daemon-local Wails RPC, not an `/api/...` capability-gated HTTP route reachable by a guest browser) — confirm route registration mirrors `POST /sessions/{id}/funnel`'s daemon-only registration, not the guest-reachable `/sessions/{id}/ws` pattern. |
| Plugin-config self-fetch leaking the cap token via referrer/logging | Information Disclosure | Existing pattern already puts the cap token in the query string for the WS URL (`?cap=`) and accepts that risk at the project level (Cache-Control: no-store already set on both plugin-config routes, `plugin_config.go:34`, `plugin_config_stream.go:47`) — the new fetch/EventSource calls should follow the identical `?cap=` convention, no new exposure class. |

## Sources

### Primary (HIGH confidence — code read directly, this session)
- `internal/relay/hub.go` — Subscriber struct, Subscribe/Unsubscribe/SubscriberCount, lock discipline (ResizeClient/HandleInject unlock-before-IO pattern)
- `internal/relay/server.go` — NotifyViewerCount, loopback Origin stamping
- `internal/webserver/server.go` — handleWSSRelay (WSS subscribe path, two-phase subscribe), AddGrant/isGrantActive/ClearGrants, `/app/` route registration, cspHeaders wiring (3 routes only)
- `internal/webserver/origin_mw.go` — requireAllowedOrigin dual-origin logic (Phase 165 FNL-04)
- `internal/webserver/csp_mw.go` — exact CSP directive string, `connect-src 'self'`
- `internal/webserver/capability_mw.go` — requireCapability chain
- `internal/webserver/plugin_config.go`, `plugin_config_stream.go` — existing capability-gated endpoints
- `internal/capability/joincode.go` — JoinCodeManager single-use-per-code semantics
- `internal/daemon/engine.go`, `types.go`, `api.go`, `client.go` — ViewerCount call sites, NotifyOnWaiting/AutoCloseSession full chains
- `internal/daemon/remote_caps.go` — RemoteCapStore (confirmed unrelated per-consumer store, not an owner-side eviction mechanism)
- `app.go` — GetNotifyOnWaiting/SetNotifyOnWaiting bound methods
- `frontend/src/App.tsx` — handleOpenRemoteSession, openWebSessionTab, createTab/setActiveId, web-session render branch, handleToggleWeb
- `frontend/src/components/StatusBar.tsx` — footer button current implementation
- `frontend/src/components/SettingsTab.tsx` — Session Behavior / Behavior section layout, autoCloseSession pattern
- `frontend/src/components/Hub/HubPanel.tsx` — shareModalSession local state, SessionShareModal prop surface
- `frontend/src/components/Hub/SessionShareModal.tsx` — shareEnabled/ToggleWebServing/SetSessionFunnel wiring, no existing viewerCount/disconnect UI
- `frontend/src/components/Hub/WebShareSessionView.tsx` — wsURL/apiBaseURL construction, prop surface
- `frontend/src/lib/remoteSession.ts` — remoteBaseURLFor/findRemoteSession
- `TESTING.md` — Suite Manifest, Category G (M-13), the `web-plugin-hot-swap.spec.ts` skip note, traceability map format
- `frontend/e2e/web-plugin-hot-swap.spec.ts` — header comment explaining the skip and its target
- `.planning/phases/165-funnel-backend/165-UAT.md`, `165-VERIFICATION.md` — confirmed Phase 165 scope was 502/TLS/teardown/fallback, NOT multi-viewer eviction
- `gh issue view 117` — original bug report, investigation notes, candidate causes (none confirmed by code)

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` "Architecture Notes (v4.2)" section — anticipatory file-change list written at milestone-scoping time (2026-06-30); cross-checked against actual code and found **not yet implemented** for the Phase 168 items (`baseURL` prop, disconnect endpoint) — used as corroborating evidence for the FIX-01/FIX-03 coupling finding, not as an authoritative source on its own.

### Tertiary (LOW confidence)
- None — all findings in this document are grounded in direct code reads or the locked CONTEXT.md decisions.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, pure internal-code changes
- Architecture (FIX-04, UX-01, UX-02): HIGH — exact file:line citations for every claim, patterns already exist and were read directly
- Architecture (FIX-01/FIX-03 coupling): HIGH — confirmed by reading the actual component/render code, not inferred
- FIX-02/D-07 verdict ("does not reproduce in current code"): MEDIUM — exhaustive static analysis is strong evidence but the original issue explicitly calls for live two-client verification; this document recommends that live check rather than treating the static finding as fully conclusive
- Pitfalls: HIGH — each pitfall traces to a specific confirmed code location

**Research date:** 2026-07-01
**Valid until:** ~30 days (stable internal codebase; re-verify if Phase 169 or any interim hotfix touches `internal/relay/hub.go`, `internal/webserver/server.go`, or `frontend/src/App.tsx`'s web-session render branch before this phase is planned)
