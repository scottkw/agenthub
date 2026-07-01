# Phase 168: Bug Fix & Settings Polish - Context

**Gathered:** 2026-07-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Five web-share/Hub bug repairs plus two Settings/Footer UX fixes, clearing Issues
#112, #115, #116, #117, #118, and #121. Requirements FIX-01..FIX-04, UX-01, UX-02.

Specifically:
- **FIX-01 (#112):** `/app/` web-share guests self-fetch live plugin-config + SSE hot-swap.
- **FIX-02 (#117):** multiple simultaneous remote viewers; a "Disconnect all viewers" control.
- **FIX-03 (#118):** opening a remote session from the Hub opens an in-app tab, not a browser.
- **FIX-04 (#121):** the Hub viewer count excludes the app's own internal WebSocket subscribers.
- **UX-01 (#116):** a "Stay on Hub after creating session" Settings toggle.
- **UX-02 (#115):** rename footer "Enable Web" → "Share Session" and open the Share modal.

Out of scope: Tailscale detection fix (#120 / FIX-05 — split into Phase 169), any Funnel
backend/frontend changes (Phases 165/166), notifications (Phase 167), and any new
capabilities beyond repairing these six issues.
</domain>

<decisions>
## Implementation Decisions

### FIX-04 — Hub viewer count (#121)
- **D-01:** Count **only subscribers whose `Subscriber.Origin == "web"`** (the webserver
  Tailscale/Funnel path). Exclude `Origin == "local"` (the loopback relay path) — that is
  where the app's own TerminalPanel, ChatPanel, status watcher, and Hub-card preview
  connect. A never-shared local session therefore reads **0 viewers** (success criterion 6).
- **D-02:** Use a **raw connection count** over web-origin subscribers — NOT a PersonKey
  (presence-roster) collapse. This is safe and matches the user's intent ("actual
  connections, not counting web/chat/Hub-preview as separate connections") because:
  a web guest's terminal **and** chat multiplex over a **single** WebSocket
  (`WebShareSessionView.tsx:57` builds one `wss://…/sessions/{id}/ws?cap=…` that both
  panels share → one `hub.Subscribe`); the plugin-config SSE stream is tracked in
  `pluginConfigSubscribers`, **not** in `hub.subscribers`, so it never counts; and the
  Hub preview is `Origin == "local"`, already excluded. Two browser tabs from one person
  = 2 actual connections, which is the desired reading.
- **D-03:** Add a new Hub method (e.g. `RemoteViewerCount()`) that returns
  `count of subscribers where Origin == "web"`; wire `daemon.SessionInfo.ViewerCount`
  (`engine.go:538`) to it instead of `hub.SubscriberCount()`. Update the
  `types.go:29` `ViewerCount` comment ("number of active WebSocket subscribers") to reflect
  the new remote-only semantics. `SubscriberCount()` itself stays as-is (still used elsewhere).

### FIX-02 — Disconnect viewers (#117)
- **D-04:** **A single "Disconnect all viewers" button** in the Share modal — NOT a
  per-viewer list with individual disconnect. No viewer-roster UI is needed for this phase.
- **D-05:** The button **force-closes all `Origin == "web"` connections** for the session
  (new backend method iterating web-origin subscribers and closing each; the existing
  `Subscriber.CloseSlow`-style close is the closest analog). Local-origin connections are
  never touched.
- **D-06:** **Drop connections only — do NOT revoke the capability.** A legitimately-stuck
  viewer can reconnect with the same join code. To fully cut access the user disables web
  sharing via the existing path. No cap-revocation plumbing in this button.
- **D-07 (multi-viewer support):** The relay `Hub` already stores subscribers in a
  `map[*Subscriber]struct{}` with no eviction on new subscribe — so a new viewer kicking an
  existing one (the other half of #117) is NOT in the relay hub. **RESEARCH ITEM:** confirm
  whether Phase 165's dual-origin fix already resolved the multi-viewer-kick end-to-end
  (per the ROADMAP `Depends on` note) before scoping any relay-buffer/disconnect work; if it
  still reproduces, locate it in the webserver Tailscale path / cap single-use, not the hub.

### UX-01 — Stay on Hub after creating session (#116)
- **D-08:** Toggle lives in **Settings → Session Behavior** (`id="settings-session-behavior"`
  in `SettingsTab.tsx`, the same section as the Phase 167 `notifyOnWaiting` toggle and the
  Phase 150 shell-warning toggle). Follow that established toggle pattern.
- **D-09:** **Default OFF** — preserves today's behavior (creating a session switches to it).
  The user opts in to staying on the Hub. No surprise behavior change on upgrade; matches the
  roadmap wording "when enabled … prevents auto-switching".
- **D-10:** When the toggle is **ON**, the create flow **skips `setActiveId(newSessionId)`**
  (`App.tsx:791`) but **still creates the tab** (so the session is openable from its card /
  tab strip). The active view stays on the Hub, where the new session's card appears.
- **D-11 (scope):** The toggle gates **only** the GUI Hub "New session" create path — which
  is the **only** auto-switch in the app. Investigation confirmed: the "New session" button
  exists only on `HubPanel` (`App.tsx:1534`) → `NewSessionModal` → `createTab`
  (`App.tsx:768`) → `setActiveId`. CLI-created sessions talk to the daemon directly, never run
  through `createTab`, and already never auto-switch the GUI. So no `fromHub` flag is
  required — gating the single `setActiveId` IS "Hub-created only".
- **D-12:** Persist the setting following the **`NotifyOnWaiting` end-to-end pattern**
  (add a `bool` to `daemonSettings` in `engine.go`, mirror `Get/Set` across
  engine.go → api.go → client.go → app.go → SettingsTab.tsx), consistent with Phase 167.

### UX-02 — Footer "Share Session" button (#115)
- **D-13:** Rename the footer button (`StatusBar.tsx:40`) from **"Enable Web"** to
  **"Share Session"**.
- **D-14:** The button **always opens the Hub Share modal** for the currently-active session.
  **Remove the footer's direct `ToggleWebServing` call** — enable/disable/links all happen
  inside the modal, making the modal the single source of truth and eliminating the
  button↔modal state drift that is the essence of #115.
- **D-15:** The button is **hidden** when the active tab is not a shareable local session
  (Hub, Settings, Help, Welcome, remote/web-session tabs) — no dead/greyed control on
  non-session surfaces.

### FIX-01 — /app/ guest plugin-config + SSE (#112) — determined by requirement
- **D-16:** The backend already ships both endpoints, **capability-gated**:
  `GET /api/plugin-config` (`handleGetPluginConfig`) and `GET /api/plugin-config/stream`
  (`handleStreamPluginConfig`, SSE — `plugin_config_stream.go`). The gap is purely
  client-side: `WebShareSessionView` receives `pluginConfig` as a prop from the Wails app and
  never self-fetches. **Fix:** when running as a web guest, `WebShareSessionView` self-fetches
  `/api/plugin-config?cap=<capToken>` for the initial config and subscribes to the SSE stream
  (`/api/plugin-config/stream?cap=<capToken>`, `EventSource` + `addEventListener('plugin-config')`)
  for hot-swap updates. Reuse the existing `capToken` it already uses for the WS URL. Verify no
  CSP errors in the guest browser console (success criterion 1).

### FIX-03 — remote session opens in-app tab (#118) — determined by requirement
- **D-17:** An in-app `web-session` tab path already exists (`openWebSessionTab` → mounts
  `WebShareSessionView`). Today `handleOpenRemoteSession` (`App.tsx:1169`) wrongly routes
  through `OpenRemoteSessionURL(id)` → `BrowserOpenURL(url)`. **Fix:** reroute the Hub
  remote-open action to open the session in the in-app `web-session` tab (connecting via the
  remote peer's host/baseURL + cap), not an external browser window. Keep the existing
  join-code / cap-exchange (`RemoteJoinCodeModal`, `intent='open-session'`) flow that supplies
  the cap.

### Claude's Discretion
- Exact new-method names (`RemoteViewerCount`, disconnect endpoint shape), button styling,
  and the precise wiring of the disconnect action are the planner/implementer's call, provided
  they honor the decisions above.
- Whether the "Disconnect all viewers" button is always visible in the Share modal or only
  when `viewerCount > 0` — implementer's discretion (prefer showing only when there are web
  viewers to disconnect).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & roadmap
- `.planning/REQUIREMENTS.md` §FIX (FIX-01..FIX-04), §UX (UX-01, UX-02) — the locked
  requirements this phase satisfies.
- `.planning/ROADMAP.md` "Phase 168" — goal + 6 success criteria; the `Depends on` note about
  verifying Phase 165's dual-origin fix against the #117 multi-viewer-kick.

### Viewer count / relay hub (FIX-04, FIX-02)
- `internal/relay/hub.go` — `Subscriber` struct (`Origin` field line 57: `"local"` loopback
  vs `"web"` webserver Tailscale), `Subscribe`/`Unsubscribe`, `SubscriberCount()` (line 219),
  `presenceRoster`/PersonKey collapse, `Subscriber.CloseSlow` (line 42/288).
- `internal/relay/server.go` — loopback relay path stamps `Origin = "local"` (lines ~264),
  `NotifyViewerCount`.
- `internal/webserver/server.go` — web WSS handler (`GET /sessions/{id}/ws`, line 938;
  `hub.Subscribe` at 1370) that stamps the `"web"` origin; the seam for the disconnect method.
- `internal/daemon/engine.go:538` — `viewerCount = hub.SubscriberCount()` → SessionInfo (the
  line to repoint at the new remote-only count).
- `internal/daemon/types.go:29` — `SessionInfo.ViewerCount` (comment to update).
- `frontend/src/components/Hub/SessionCard.tsx:500` — renders `viewerCount > 0`.

### Web-share plugin-config + SSE (FIX-01)
- `internal/webserver/plugin_config_stream.go` — SSE handler (`handleStreamPluginConfig`,
  `pluginConfigSubscribers`, `event: plugin-config` frames).
- `internal/webserver/plugin_config.go` — `handleGetPluginConfig`; routes registered in
  `internal/webserver/server.go` (`GET /api/plugin-config`, `GET /api/plugin-config/stream`,
  both `requireCapability`-wrapped).
- `frontend/src/components/Hub/WebShareSessionView.tsx` — web guest surface (line 57 wsURL;
  where the self-fetch + EventSource land).
- `frontend/src/components/TerminalPanel.tsx` / `ChatPanel.tsx` — consume `pluginConfig`.

### Remote session open (FIX-03)
- `frontend/src/App.tsx` — `handleOpenRemoteSession` (line 1169, the `BrowserOpenURL` to
  replace), `openWebSessionTab` (in-app `web-session` tab), `RemoteJoinCodeModal`
  (`intent='open-session'`), `OpenRemoteSessionURL` binding.
- `frontend/src/lib/remoteSession.ts` — `remoteBaseURLFor`, `findRemoteSession`.

### Footer button + Share modal (UX-02, FIX-02)
- `frontend/src/components/StatusBar.tsx:40` — the "Enable Web" footer button + `onToggleWeb`.
- `frontend/src/components/Hub/SessionShareModal.tsx` — the Share modal (Disconnect button
  D-04 + the modal the footer now opens); `shareEnabled` seeded from `session.webEnabled`.
- `internal/daemon/api.go` — `ToggleWebServing` RPC (the direct call the footer drops).

### Settings toggle (UX-01) — pattern to mirror
- `frontend/src/components/SettingsTab.tsx` — "Session Behavior" section
  (`id="settings-session-behavior"`); the Phase 167 `notifyOnWaiting` and Phase 150
  shell-warning toggles are the exact pattern.
- `.planning/phases/167-native-notifications/167-CONTEXT.md` — documents the
  `NotifyOnWaiting`/`StartMinimized` end-to-end daemon-setting wiring (engine.go → api.go →
  client.go → app.go → SettingsTab.tsx) that D-12 follows.

### Prior-phase seams
- `.planning/phases/165-funnel-backend/165-UAT.md` / `165-VERIFICATION.md` — Phase 165
  dual-origin/multi-viewer behavior (D-07 research item).
- `.planning/phases/166-funnel-frontend-help-guide/166-CONTEXT.md` — Share modal architecture
  (`SessionShareModal`, `SessionSharePanel`, server-truth-seeded state).

### Standing conventions
- `TESTING.md` — register new test files in the Suite Manifest (§2) + Traceability map (§4);
  run `bash tests/check-traceability-paths.sh` before committing; add M-NN manual items for
  behavior that can't be automated (live multi-viewer / browser-console CSP checks).
- Colorblind rule — verify any color-based UI at hex/source level, not by eye (user is
  colorblind). Relevant if the footer button or Disconnect control uses color for state.
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `Subscriber.Origin` (`internal/relay/hub.go:57`) — already distinguishes `"local"` vs
  `"web"`; FIX-04's classification rule is a field that already exists, no new tagging needed.
- `Subscriber.CloseSlow` close pattern — the analog for the FIX-02 force-disconnect.
- `openWebSessionTab` + `WebShareSessionView` — the in-app tab FIX-03 should reuse.
- Existing capability-gated `/api/plugin-config` + SSE endpoints — FIX-01 is client-only.
- Phase 167 `notifyOnWaiting` toggle + `NotifyOnWaiting` daemon-setting chain — the template
  for UX-01's toggle + persistence.
- `SessionShareModal` (server-truth-seeded, existing modal lifecycle) — hosts the Disconnect
  button and is what the footer opens.

### Established Patterns
- Web guests multiplex terminal + chat on a **single** WS per tab → one `hub.Subscribe` per
  guest tab (basis for FIX-04 D-02 raw-count safety).
- Settings persist server-side via the daemon (`daemonSettings` struct), surfaced through the
  Get/Set RPC chain — UX-01 follows this rather than a frontend-only setting.
- Share state is daemon-truth-seeded (`session.webEnabled`) — UX-02 removing the footer's
  direct toggle keeps the modal as the single writer.

### Integration Points
- `daemon.SessionInfo.ViewerCount` (engine.go:538) → the one line to repoint at the remote
  count; flows to the Hub card via `ListSessions`.
- New disconnect RPC (daemon `api.go`) → webserver/relay method → `SessionShareModal` button.
- UX-01 toggle → `daemonSettings` → gates `App.tsx:791 setActiveId` in `createTab`.
- UX-02 → `StatusBar` button → opens `SessionShareModal` for the active session.
</code_context>

<specifics>
## Specific Ideas

- User's FIX-04 guardrail (verbatim intent): raw connection count is fine "as long as they are
  actual connections and not counting web, chat, Hub preview, etc. all as separate
  connections." Verified satisfied by the single-WS-per-guest + local-origin-excluded design.
- FIX-02 "Disconnect all viewers" is a stuck-viewer escape hatch, not an access-revocation
  tool — revocation remains the existing "disable web sharing" path.
</specifics>

<deferred>
## Deferred Ideas

- Per-viewer disconnect list / viewer roster UI in the Share modal — heavier than a bug-fix
  phase needs; revisit only if a real need appears.
- Disconnect + cap revocation as one action — overlaps with "disable web sharing"; out for now.
- FIX-05 / #120 Tailscale connection detection — already split into **Phase 169** (orthogonal
  subsystem, needs a non-admin macOS test environment).

### Reviewed Todos (not folded)
- **"Help Guide — document Tailscale Funnel admin prerequisites"** (docs todo,
  `2026-06-30-help-guide-document-tailscale-funnel-admin-prerequisites.md`) — reviewed, NOT
  folded. It belongs to the Funnel Help guide (Phase 166 territory), not this web-share/Hub
  bug-fix phase.
</deferred>

---

*Phase: 168-bug-fix-settings-polish*
*Context gathered: 2026-07-01*
