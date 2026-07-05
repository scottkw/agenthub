# Phase 168: Bug Fix & Settings Polish - Pattern Map

**Mapped:** 2026-07-01
**Files analyzed:** 12 source (modify) + 7 test (new/extend)
**Analogs found:** 19 / 19 (every touched file has an in-repo, same-shape sibling)

> This phase is 100% edits to existing files — no new external deps, no new
> directories. Every fix has a same-shape sibling already merged (per
> 168-RESEARCH.md). The engineering risk is wiring, not new design. Excerpts
> below are the exact analogs to copy from, with file:line anchors.

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---------------|------|-----------|----------------|---------------|
| `internal/relay/hub.go` | service (relay hub) | event-driven / fan-out | same file: `SubscriberCount` (218), `broadcastResize` (292) | exact (same file, same pattern) |
| `internal/daemon/engine.go` | service (session engine) | CRUD / config-persist | same file: `Get/SetNotifyOnWaiting` (1107-1122) | exact |
| `internal/daemon/types.go` | model (DTO) | — (struct/comment) | same file: `ViewerCount` field (29) | exact |
| `internal/daemon/api.go` | controller (HTTP mux) | request-response | same file: `handleGet/SetNotifyOnWaiting` (874-888) | exact |
| `internal/daemon/client.go` | service (daemon RPC client) | request-response | same file: `Get/SetNotifyOnWaiting` (166-181) | exact |
| `app.go` | provider (Wails bound methods) | request-response | same file: `App.Get/SetNotifyOnWaiting` (727-753) | exact |
| `frontend/src/components/SettingsTab.tsx` | component | CRUD (toggle) | same file: `autoCloseSession` toggle (557-582) | exact |
| `frontend/src/components/StatusBar.tsx` | component | event-driven (button) | same file: existing "Enable Web" button (32-59) | exact (rename/rewire in place) |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | component | streaming (WS/SSE) | same file: `wsURL` construction (57-58) | role+flow match |
| `frontend/src/App.tsx` | provider (app shell) | event-driven / state | same file: `openWebSessionTab` (1058), `createTab` (768), `handleOpenRemoteSession` (1169) | exact |
| `frontend/src/components/Hub/SessionShareModal.tsx` | component (modal) | request-response (RPC) | same file: `ToggleWebServing` handler (214-233) | exact |
| `frontend/src/components/Hub/HubPanel.tsx` | component (container) | state (modal-open) | same file: `shareModalSession` state (281-300) | exact (lift to controlled prop) |

| New/Extended Test File | Role | Analog | Match Quality |
|------------------------|------|--------|---------------|
| `internal/relay/hub_test.go` (extend) | test (Go) | `TestHubTwoSubscribersBothReceive` (57), `TestHubSlowSubscriberGetsDisconnected` (93) | exact |
| `internal/daemon/engine_stayonhub_test.go` (new) | test (Go) | `internal/daemon/engine_notify_test.go` | exact |
| `internal/daemon/api_stayonhub_test.go` (new) | test (Go) | `internal/daemon/api_notify_test.go` | exact |
| `frontend/.../SettingsTab.stay-on-hub-toggle.test.tsx` (new) | test (vitest) | `SettingsTab.notify-toggle.test.tsx` | exact |
| `frontend/.../StatusBar.shareSession.test.tsx` (extend/new) | test (vitest) | `StatusBar.test.tsx` | exact |
| `frontend/.../App.handleOpenRemoteSession` (extend) | test (vitest) | `App.open-remote.test.tsx` | exact |
| `frontend/.../WebShareSessionView.plugin-config.test.tsx` (new) | test (vitest) | (no direct sibling — see No Analog Found) | partial |

---

## Pattern Assignments

### `internal/relay/hub.go` — `RemoteViewerCount()` + `DisconnectWebViewers()` (FIX-04, FIX-02)

**Analog:** same file — `SubscriberCount` (counting) + `broadcastResize` (origin-agnostic
fan-out with unlock-before-IO close semantics).

**Origin field already exists** (lines 44-60) — no new tagging, just filter on it:
```go
// Origin: "local" (relay loopback) or "web" (webserver Tailscale)   // hub.go:57
Origin string
// CloseSlow is called in its own goroutine when the Msgs channel is full.
// Implementations should close the WebSocket and call Unsubscribe.     // hub.go:40-42
CloseSlow func()
```

**Counting analog to mirror** (`SubscriberCount`, lines 218-223) — `RemoteViewerCount`
copies this lock discipline and adds an `Origin == "web"` filter:
```go
// SubscriberCount returns the number of currently subscribed clients. (MC-04)
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}
```
`SubscriberCount()` STAYS as-is (still called by `relay/server.go` NotifyViewerCount).
Add a new sibling `RemoteViewerCount()` that loops `h.subscribers` and counts
`sub.Origin == "web"`.

**Disconnect analog — unlock-before-IO close** (`broadcastResize`, lines 292-303) — the
close-on-full pattern (`go sub.CloseSlow()`) is the exact "server closes this connection"
mechanism to reuse for `DisconnectWebViewers`. CRITICAL: collect matching subscribers
under `h.mu`, then release the lock BEFORE calling `CloseSlow` (CloseSlow → Unsubscribe
re-enters `h.mu`; calling it under the lock self-deadlocks — the same T-157-04 hazard the
existing broadcast code documents at hub.go:288-291):
```go
func (h *Hub) broadcastResize(cols, rows uint16) {
	frame := MakeResizeFrame(cols, rows)
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subscribers {
		select {
		case sub.Msgs <- frame:
		default:
			go sub.CloseSlow()   // <-- the close mechanism DisconnectWebViewers reuses
		}
	}
}
```
`DisconnectWebViewers` shape: `h.mu.Lock()` → collect `[]*Subscriber` where
`Origin == "web"` into a local slice → `h.mu.Unlock()` → range the slice, `go sub.CloseSlow()`.

---

### `internal/daemon/engine.go` — viewerCount swap (FIX-04) + `stayOnHubAfterCreate` setting (UX-01)

**Analog:** same file — `daemonSettings` struct + `Get/SetNotifyOnWaiting`.

**FIX-04 call-site swap** (lines 538-541) — repoint the ONE line feeding `SessionInfo.ViewerCount`:
```go
// MC-04: populate viewer count from hub subscriber count.
viewerCount := 0
if hub, ok := e.manager.Get(s.ID); ok {
	viewerCount = hub.SubscriberCount()   // <-- FIX-04: change to hub.RemoteViewerCount()
}
```
This is the SOLE call site feeding the Hub card (`relay/server.go:506` NotifyViewerCount is
a separate MsgMeta frame the frontend never parses — leave it untouched, per RESEARCH A2).

**UX-01 settings field** — add a `bool` to `daemonSettings` (lines 112-121), copying the
`NotifyOnWaiting` line exactly (zero-value = OFF, `omitempty`, NO defaults-merge entry):
```go
NotifyOnWaiting  bool `json:"notifyOnWaiting,omitempty"` // Phase 167 NTF-04: default OFF; zero-value is the correct default (no defaults-merge needed)
```

**UX-01 Get/Set methods** (lines 1107-1122) — mirror exactly (guard `e.mu`, call `e.saveSettingsToDisk()`):
```go
func (e *SessionEngine) GetNotifyOnWaiting() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.notifyOnWaiting
}
func (e *SessionEngine) SetNotifyOnWaiting(val bool) {
	e.mu.Lock()
	e.notifyOnWaiting = val
	e.saveSettingsToDisk()
	e.mu.Unlock()
}
```
(Also add the backing `e.stayOnHubAfterCreate bool` engine field + load it from
`daemonSettings` where `notifyOnWaiting` is loaded.)

---

### `internal/daemon/types.go` — `ViewerCount` comment (FIX-04)

**Analog:** same file, line 29. Update the comment to reflect remote-only semantics (D-03).
No new field needed for UX-01 (the setting rides `daemonSettings`, not `SessionInfo`):
```go
ViewerCount int `json:"viewerCount"`  // MC-04: number of active WebSocket subscribers
//                                     ^ FIX-04: reword to "remote (web-origin) viewers only"
```

---

### `internal/daemon/api.go` — disconnect route (FIX-02) + stay-on-hub Get/Set routes (UX-01)

**Analog:** same file — `handleGet/SetNotifyOnWaiting` + route registration.

**Route registration** (lines 123-126) — add sibling routes here:
```go
a.mux.HandleFunc("GET /settings/notify-on-waiting", a.handleGetNotifyOnWaiting)
a.mux.HandleFunc("PATCH /settings/notify-on-waiting", a.handleSetNotifyOnWaiting)
```

**Handler pair** (lines 874-888) — copy verbatim, rename the JSON key:
```go
func (a *API) handleGetNotifyOnWaiting(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"notifyOnWaiting": a.engine.GetNotifyOnWaiting()})
}
func (a *API) handleSetNotifyOnWaiting(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NotifyOnWaiting bool `json:"notifyOnWaiting"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	a.engine.SetNotifyOnWaiting(req.NotifyOnWaiting)
	w.WriteHeader(http.StatusNoContent)
}
```

**FIX-02 disconnect route — SECURITY (owner-only trust boundary):** the disconnect action
must NOT be a guest-reachable `/api/...` capability route. Register it on this daemon-local
mux (same trust boundary as `ToggleWebServing`), e.g. `POST /sessions/{id}/disconnect-viewers`,
so only the owning desktop app can trigger it (RESEARCH Security V4 / STRIDE-EoP). Follow the
`PATCH .../{id}/...` daemon-only registration style already in this mux, NOT the webserver's
`GET /sessions/{id}/ws` guest pattern.

---

### `internal/daemon/client.go` — `Get/SetStayOnHubAfterCreate` (UX-01) + disconnect wrapper (FIX-02)

**Analog:** same file — `Get/SetNotifyOnWaiting` (lines 166-181):
```go
func (c *DaemonClient) GetNotifyOnWaiting() (bool, error) {
	var resp map[string]bool
	if err := c.doJSON(http.MethodGet, "/settings/notify-on-waiting", nil, &resp); err != nil {
		return false, err
	}
	return resp["notifyOnWaiting"], nil
}
func (c *DaemonClient) SetNotifyOnWaiting(val bool) error {
	return c.doJSON(http.MethodPatch, "/settings/notify-on-waiting",
		map[string]bool{"notifyOnWaiting": val}, nil)
}
```
Disconnect wrapper: a single-line `c.doJSON(http.MethodPost, "/sessions/"+id+"/disconnect-viewers", nil, nil)`.

---

### `app.go` — bound Wails methods (UX-01 Get/Set, FIX-02 disconnect)

**Analog:** same file — `App.Get/SetNotifyOnWaiting` (lines 727-753):
```go
func (a *App) GetNotifyOnWaiting() bool {
	return a.notifyOnWaiting.Load()
}
func (a *App) SetNotifyOnWaiting(val bool) error {
	a.notifyOnWaiting.Store(val)
	if a.client == nil {
		return fmt.Errorf("daemon not connected")
	}
	return a.client.SetNotifyOnWaiting(val)
}
```
Note the `notifyOnWaiting` atomic cache exists because the tray poller reads it every tick.
UX-01's `stayOnHubAfterCreate` has NO such background reader — a plain client passthrough
(like `SetStartMinimized` at app.go:724, `return a.client.Set...`) is sufficient; the atomic
cache is not required. FIX-02 disconnect binds a thin `a.client.DisconnectViewers(id)`.

---

### `frontend/src/components/SettingsTab.tsx` — "Stay on Hub after creating a session" toggle (UX-01)

**Analog:** same file — the `autoCloseSession` toggle (lines 557-582), which lives in the
SAME `id="settings-session-behavior"` section this toggle belongs in (D-08). Copy the
label/track/thumb/input markup verbatim:
```tsx
<h3 id="settings-session-behavior">Session Behavior</h3>
<div className="settings-panel__field-group">
  {autoCloseLoaded && (
    <label
      className={`settings-panel__toggle-row${autoCloseSession ? ' settings-panel__toggle-row--checked' : ''}`}
      htmlFor="autoCloseSession"
      style={autoCloseSaving ? { pointerEvents: 'none', opacity: 0.6 } : undefined}
    >
      <span className="settings-panel__toggle-track"><span className="settings-panel__toggle-thumb" /></span>
      <span className="settings-panel__toggle-label">Auto-close tab on exit</span>
    </label>
  )}
  <input type="checkbox" id="autoCloseSession" className="settings-panel__toggle-input"
    checked={autoCloseSession} onChange={() => void handleToggleAutoClose()} />
  <p className="settings-panel__description">...</p>
  {autoCloseError && <p className="settings-panel__error">{autoCloseError}</p>}
</div>
```

**State + load + save wiring** — mirror `notifyOnWaiting` (state 121-124, load 215-218,
handler 395-409, import 20-21):
```tsx
const [notifyOnWaiting, setNotifyOnWaiting] = useState(false)         // default OFF (D-09)
// load:
GetNotifyOnWaiting().then(val => { setNotifyOnWaiting(val); setNotifyOnWaitingLoaded(true) })
              .catch(() => setNotifyOnWaitingLoaded(true))
// handler:
async function handleToggleNotifyOnWaiting() {
  const next = !notifyOnWaiting
  // ...saving guard...
  await SetNotifyOnWaiting(next); setNotifyOnWaiting(next)
}
```
IMPORTANT (RESEARCH Pattern 2): render under `id="settings-session-behavior"` (line 557,
same section as `autoCloseSession`) — NOT `id="settings-behavior"` (line 496) where
`notifyOnWaiting` itself lives. Do not conflate the two sections.

---

### `frontend/src/components/StatusBar.tsx` — "Share Session" button (UX-02)

**Analog:** same file — the current "Enable Web" button block (lines 32-59). Rewrite in place:
```tsx
{webServerRunning && !webEnabled && (
  <>
    <span className="tab-status-bar__state tab-status-bar__state--off">WEB OFF</span>
    <button className="tab-status-bar__btn" onClick={onToggleWeb} title="Enable web sharing for this session">
      Enable Web       {/* D-13: rename to "Share Session" */}
    </button>
  </>
)}
```
Changes per decisions:
- **D-13:** rename label `Enable Web` → `Share Session`.
- **D-14:** replace `onClick={onToggleWeb}` (direct `ToggleWebServing`) with a new
  `onShareSession` prop that opens the Share modal for the active session. Drop the
  footer's direct toggle entirely (modal becomes single source of truth). This collapses
  the current two-branch (WEB OFF "Enable Web" / WEB ON "Disable Web") logic into one button.
- **D-15 (SECURITY, STRIDE-EoP):** hide the button unless the active tab is a shareable
  LOCAL session. Gate on tab `type` (not just `sessionId` presence) so it never renders on
  `web-session` / `file-browser` / Hub / Settings / Help / Welcome tabs. The `webEnabled`/
  `webServerRunning` props already flow in (props 6-7, 21-22) — add the tab-type gate at the
  `App.tsx` render site (StatusBar is rendered per active tab, `App.tsx:1791`).

---

### `frontend/src/components/Hub/WebShareSessionView.tsx` — plugin-config self-fetch + SSE (FIX-01) + `baseURL` prop (FIX-03)

**Analog:** same file — the `wsURL`/`apiBaseURL` construction (lines 57-58):
```tsx
const wsURL = `wss://${window.location.host}/sessions/${encodeURIComponent(sessionId)}/ws?cap=${encodeURIComponent(capToken)}`
const apiBaseURL = window.location.origin
```
The component already receives `capToken` (prop, line 15) and `pluginConfig` (prop, line 21,
forwarded to `TerminalPanel`/`ChatPanel` at 73/85-87). FIX-01 adds a `useEffect` that, when
running as a web guest (no `pluginConfig` prop from Wails), self-fetches from `apiBaseURL`
and subscribes to the SSE stream — reusing the SAME `?cap=` convention as `wsURL`:
```tsx
useEffect(() => {
  if (!isWebGuest) return                       // only when no Wails-provided pluginConfig prop
  const ctrl = new AbortController()
  fetch(`${apiBaseURL}/api/plugin-config?cap=${encodeURIComponent(capToken)}`, { signal: ctrl.signal })
    .then(r => (r.ok ? r.json() : null)).then(cfg => cfg && setLivePluginConfig(cfg)).catch(() => {})
  const es = new EventSource(`${apiBaseURL}/api/plugin-config/stream?cap=${encodeURIComponent(capToken)}`)
  es.addEventListener('plugin-config', ev => { try { setLivePluginConfig(JSON.parse((ev as MessageEvent).data)) } catch {} })
  return () => { ctrl.abort(); es.close() }
}, [capToken, apiBaseURL])
```
Backend is UNCHANGED (endpoints already capability-gated — `plugin_config.go:handleGetPluginConfig`,
`plugin_config_stream.go:handleStreamPluginConfig`, registered `server.go:862,867`).

**CRITICAL — FIX-01/FIX-03 coupling (RESEARCH Pitfall 2):** do NOT hardcode
`window.location.origin` in the fetch/SSE. Add an optional `baseURL` prop, default it to
`window.location.origin`, and derive `apiBaseURL`/`wsURL` from it. FIX-03's in-app remote-peer
tab supplies a DIFFERENT peer's origin; hardcoding breaks it. Parameterize on day one.

---

### `frontend/src/App.tsx` — createTab gating (UX-01), remote-open rewrite (FIX-03), modal state lift (UX-02)

**Analog:** same file — three sibling spots.

**UX-01 gating** — `createTab` (lines 782-800). Gate the single `setActiveId(sessionId)`
(line 791) on the new setting; still create the tab (D-10, D-11):
```tsx
const sessionId = await CreateSession(cliName, defaultName, workDir, args, cols, rows)
const tab: Tab = { id: sessionId, name: defaultName, sessionId, cli: cliName }
setTabs((prev) => [...prev, tab])
setActiveId(sessionId)   // <-- UX-01: skip this ONE line when stayOnHubAfterCreate is ON
```
This is the ONLY auto-switch in the app (D-11 confirmed) — no `fromHub` flag needed.

**FIX-03 rewrite** — `handleOpenRemoteSession` (lines 1169-1188) currently routes through
`OpenRemoteSessionURL(id)` → `BrowserOpenURL(url)` (the external-browser path to replace):
```tsx
if (remoteCapsCached.has(session.id)) {
  const url = await OpenRemoteSessionURL(session.id)
  BrowserOpenURL(url)     // <-- FIX-03: replace with in-app openWebSessionTab(id, baseURL, cap)
  return
}
```
Reroute to the in-app tab via `openWebSessionTab` (lines 1058-1074) — the exact
find-or-focus tab pattern to reuse:
```tsx
const openWebSessionTab = useCallback((sessionId: string) => {
  const tabId = webSessionTabId(sessionId)
  const existing = tabs.find((t) => t.id === tabId)
  if (existing) { setActiveId(existing.id); return }
  const newTab: Tab = { id: tabId, name: 'Session', sessionId, cli: '', type: 'web-session' }
  setTabs((prev) => [...prev, newTab]); setActiveId(newTab.id)
}, [tabs])
```
**CRITICAL — per-tab params (RESEARCH Pitfall 3):** the web-session render branch
(lines 1581-1584) reads a single mount-stable `webParams` (line 113), correct today because
only ONE web-session tab exists. FIX-03 introduces multiple remote-peer tabs (distinct
host/cap). Extend `openWebSessionTab`'s signature to `(sessionId, baseURL, capToken)`, carry
`baseURL`/`capToken` per-tab (on the `Tab` object or a side-map keyed by tab id), and change
the render branch to read from the tab, not the global `webParams`:
```tsx
{activeId !== null && activeId.startsWith('__websession__') && (
  <WebShareSessionView
    sessionId={webParams.sessionId ?? activeId.slice('__websession__'.length)}
    capToken={webParams.capToken ?? ''}   // <-- FIX-03: read per-tab, pass baseURL
```
Keep the existing join-code/cap-exchange (`RemoteJoinCodeModal`, `intent='open-session'`,
lines 1245-1272) that supplies the cap.

**UX-02 modal state lift** — see HubPanel entry below; `shareModalSession`/`setShareModalSession`
move up to `App.tsx` (which already holds `sessions`, `webServerMode`, `webServerRunning`,
`shellWebShareWarned`) so StatusBar can call `setShareModalSession(sessions.find(s => s.id === activeId))`.

---

### `frontend/src/components/Hub/SessionShareModal.tsx` — Disconnect-all-viewers button + viewerCount (FIX-02, FIX-04 display)

**Analog:** same file — the `ToggleWebServing` action handler (lines 214-233):
```tsx
const next = !shareEnabled
// ...pending/optimistic guard...
await ToggleWebServing(session.id, next)
// on failure: revert (shareEnabled unchanged)
```
The Disconnect button is a new sibling action calling the new daemon RPC
(`DisconnectViewers(session.id)` via bound method), same try/optimistic/revert shape.
Decisions:
- **D-04/D-05:** ONE "Disconnect all viewers" button (no per-viewer roster). Force-closes
  all `Origin=="web"` connections; local untouched.
- **D-06:** drops connections only — does NOT call `ToggleWebServing(false)` / revoke the cap.
- **Discretion:** prefer showing the button only when `session.viewerCount > 0` (FIX-04's
  now-remote-only count, already available on `SessionInfo`, rendered at `SessionCard.tsx:500`).

---

### `frontend/src/components/Hub/HubPanel.tsx` — lift `shareModalSession` to controlled prop (UX-02)

**Analog:** same file — the local `shareModalSession` state + sync effect + modal render
(lines 281-300, 585-590):
```tsx
const [shareModalSession, setShareModalSession] = useState<SessionInfo | null>(null)
const handleShare = useCallback((session: SessionInfo) => { setShareModalSession(session) }, [])
// sync effect keeps the open modal's session fresh across the 3s Hub poll (keyed on .id):
useEffect(() => {
  if (!shareModalSession) return
  const updated = sessions.find((s) => s.id === shareModalSession.id)
  if (updated && updated !== shareModalSession) setShareModalSession(updated)
}, [sessions, shareModalSession?.id])
// render:
{shareModalSession && (<SessionShareModal session={shareModalSession} onClose={() => setShareModalSession(null)} ... />)}
```
**Fix (RESEARCH Pattern 4):** replace the local `useState` with a controlled prop pair
threaded from `App.tsx` (`shareModalSession` / `setShareModalSession`), so both the Hub card
click AND the new footer button drive ONE modal instance. Do NOT duplicate `<SessionShareModal>`
(a second instance = double RPC polling / two sources of truth — the exact class of bug #115
exists to fix). The `activeGroupId`/`groupDefs` controlled-prop threading already visible in
HubPanel's prop list is the local precedent for this lift.

---

## Shared Patterns

### Settings persistence chain (UX-01) — server-truth end-to-end
**Source:** the `NotifyOnWaiting` chain (Phase 167). **Apply to:** UX-01's `stayOnHubAfterCreate`.
Five hops, all sibling-exact:
`engine.go` (daemonSettings field + Get/Set) → `api.go` (2 routes + 2 handlers) →
`client.go` (Get/Set wrappers) → `app.go` (bound methods) → `SettingsTab.tsx` (state/load/save).
Default OFF via zero-value; NO defaults-merge entry. Do NOT introduce a frontend-only
localStorage setting (would be the codebase's first exception, won't survive daemon restart).

### Origin-filtered hub access (FIX-04, FIX-02)
**Source:** `Subscriber.Origin` (hub.go:57), `SubscriberCount` (218), `broadcastResize` (292).
**Apply to:** `RemoteViewerCount` and `DisconnectWebViewers`. Loop the existing
`h.subscribers` map filtering `sub.Origin == "web"` — do NOT build a parallel origin-keyed
list (drift risk on Subscribe/Unsubscribe). Reuse `Subscriber.CloseSlow` for termination;
never invent a second kill mechanism.

### Lock discipline — unlock-before-IO (FIX-02)
**Source:** `broadcastResize` (hub.go:288-303), `ResizeClient` (hub.go:274). **Apply to:**
`DisconnectWebViewers`. Collect targets under `h.mu`, release the lock, THEN call
`CloseSlow` per subscriber (CloseSlow → Unsubscribe re-enters `h.mu`; calling under the lock
self-deadlocks — T-157-04).

### Owner-only trust boundary for the disconnect RPC (FIX-02, SECURITY)
**Source:** daemon-local mux registration (api.go), contrasted with the guest-reachable
webserver `/sessions/{id}/ws` route. **Apply to:** the new disconnect endpoint. Register on
the daemon-local `api.go` mux (same boundary as `ToggleWebServing`), NOT as a
capability-gated `/api/...` route a guest browser can reach.

### Same-origin fetch + SSE with `?cap=` (FIX-01)
**Source:** `wsURL`/`apiBaseURL` (WebShareSessionView.tsx:57-58). **Apply to:** the
plugin-config self-fetch and EventSource. Reuse the existing `capToken` + `?cap=` query
convention; no new CSP directive needed (`connect-src 'self'` covers same-origin fetch/SSE).
Parameterize on `baseURL`, not `window.location`, for FIX-03 reuse.

### Regression-suite wiring (standing convention, ./CLAUDE.md + TESTING.md)
**Apply to:** every new test file below. Register in TESTING.md Suite Manifest (§2) +
Traceability map (§4, repo-relative path only); run `bash tests/check-traceability-paths.sh`
before committing. FIX-02 live two-viewer smoke + FIX-01 browser-console CSP check need new
M-NN manual items; FIX-03 requires REWORDING existing M-13 (Category G) to "opens in an
in-app tab" (RESEARCH Pitfall 4), not adding a new entry.

### Test-file analogs
| New/extended test | Copy from |
|-------------------|-----------|
| `internal/relay/hub_test.go` (RemoteViewerCount / DisconnectWebViewers / TwoWebOrigin-NoEviction) | `TestHubTwoSubscribersBothReceive` (hub_test.go:57), `TestHubSlowSubscriberGetsDisconnected` (hub_test.go:93) |
| `internal/daemon/engine_stayonhub_test.go` | `internal/daemon/engine_notify_test.go` |
| `internal/daemon/api_stayonhub_test.go` | `internal/daemon/api_notify_test.go` |
| `SettingsTab.stay-on-hub-toggle.test.tsx` | `frontend/src/components/__tests__/SettingsTab.notify-toggle.test.tsx` |
| `StatusBar.shareSession.test.tsx` | `frontend/src/components/__tests__/StatusBar.test.tsx` |
| `App.handleOpenRemoteSession` (extend) | `frontend/src/components/__tests__/App.open-remote.test.tsx` |

---

## No Analog Found

| File | Role | Data Flow | Reason / Guidance |
|------|------|-----------|-------------------|
| `frontend/.../WebShareSessionView.plugin-config.test.tsx` (new) | test (vitest) | streaming (SSE) | No existing test mocks `EventSource` + `?cap=` self-fetch for this component. Build fresh: mock `fetch` for the initial config, mock `EventSource`/`addEventListener('plugin-config')` for the hot-swap push. RESEARCH Pattern 3 is the shape. |
| `frontend/e2e/web-plugin-hot-swap.spec.ts` (rewrite) | test (Playwright) | streaming (SSE) | EXISTS but targets a RETIRED UMD-global mechanism (`window.WebglAddon =`) against the old raw `/sessions/{id}` viewer. RESEARCH Pitfall 5: do NOT blind-un-skip. Treat as a reference for WHAT to test; verify how the current `/app/` SPA `TerminalPanel.tsx` loads xterm addons before deciding whether the hook technique still applies. |
| FIX-02 backend eviction fix | — | — | No analog because RESEARCH (A1) found NO eviction code path exists anywhere (hub.go/server.go/capability_mw.go/joincode.go). The "second viewer kicks first" half of #117 likely does NOT reproduce in current code. Do NOT add "single active viewer" enforcement (that would introduce the bug). Confirm empirically via the new live two-viewer M-NN item; only if it reproduces, look at the websocket `Accept` layer / OS same-tab reuse — NOT the hub or grant model. |

**Out-of-scope flags surfaced during mapping (do NOT fix here — file follow-ups):**
- `/app/` route has NO CSP header at all today (`cspHeaders` wired to only 3 routes, not
  `/app/`) — RESEARCH Pitfall 1. FIX-01's "0 console CSP errors" success check could pass for
  the wrong reason. D-16 is client-only; flag as a follow-up issue, do not add `cspHeaders`
  to `/app/` (larger SPA-bundle security-hardening change, untested vs Vite asset hashes).

## Metadata

**Analog search scope:** `internal/relay`, `internal/daemon`, repo-root `app.go`,
`frontend/src/components` (+ `Hub/`, `__tests__/`), `frontend/src/App.tsx`.
**Files scanned:** 12 source analogs (all same-file siblings) + 7 test analogs — all confirmed present.
**Pattern extraction date:** 2026-07-01
**Upstream:** 168-CONTEXT.md (D-01..D-17), 168-RESEARCH.md (Patterns 1-4, Pitfalls 1-5, Security domain).
