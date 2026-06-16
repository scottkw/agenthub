# Phase 130: Remote Browse GUI On-Ramp — Research

**Researched:** 2026-06-15
**Domain:** Go webserver / tailnet discovery / Wails RPC / React frontend
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**RESOLVED — #86 remote-browse architecture (user decision, 2026-06-15)**

**(a) Tailnet-trusted metadata-only discovery endpoint.**

- Add a discovery endpoint that returns **shareable-session metadata** (enough to list and pick a session) to **tailnet-trusted** callers.
- Content and capabilities stay **locked** — the endpoint exposes metadata only, never session content or a capability/cap token without the intended grant. This **preserves the Phase 87/88 no-enumeration security model** (RB-03).
- This directly satisfies RB-01 (a reachable peer's shareable sessions are visible — peers are no longer silently dropped because `/api/sessions` isn't enumerable without a session-scoped cap) and RB-04 (honest panel states — a reachable peer with shareable sessions is never shown as "No remote peers found"; genuinely empty/unreachable peers still surface a correct empty/error state).
- "Tailnet-trusted" = the caller is verified to be on the tailnet (the existing Phase 87/88 trust model for tailnet peers); a non-tailnet / unauthorized caller still cannot enumerate session content or obtain a cap (RB-03).

### Claude's Discretion (mechanics)

Endpoint path/shape, exactly what metadata fields are returned (must be the minimum needed to list+pick: e.g. session id, label, working-dir display name — NOT content), how the GUI renders the discovered list, and how the pick flow hands off to the File Browser are at Claude's discretion, guided by the existing remote-session and file-browser GUI patterns (Phases 52, 120, 122) and the resolved #86 decision. The session-scoped cap acquisition for actually opening a session reuses the existing web-share cap / join-code flow.

### Deferred Ideas (OUT OF SCOPE)

None for this phase. #82 (TUI Files upload parity) remains deferred to a later milestone.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RB-01 | User can see a discovered, reachable tailnet peer's shareable sessions in the Remote Sessions panel — a peer is no longer silently dropped because its `/api/sessions` list isn't enumerable without a session-scoped cap | New `/api/sessions/meta` endpoint on webserver returns shareable-session metadata without a cap; `FetchAllPeerSessionsMeta` in `internal/tailnet/sessions.go` calls it; `GetRemoteSessionsWithMeta` in `app.go` replaces `GetRemoteSessions` |
| RB-02 | User can select a remote peer session from the panel and open it in the File Browser, completing discover→list→pick end-to-end over the relay loopback the GUI uses | Existing `handleBrowseFilesRemote` → `RemoteJoinCodeModal` → `RegisterRemoteCap` → relay loopback path already works end-to-end; only the GUI on-ramp (surface the sessions to pick) is missing |
| RB-03 | The chosen approach preserves the Phase 87 no-enumeration security guarantee — an unauthorized / non-tailnet caller still cannot enumerate session content or obtain a cap without the intended grant | New endpoint is mounted on the webserver (Tailscale IP only) and returns metadata only — no cap, no content, no grant |
| RB-04 | Remote panel states are honest — a reachable peer with shareable sessions is never shown as "No remote peers found"; genuinely empty/unreachable peers still surface a correct empty/error state | `RemotePeerSessions` gains `reachable bool`; `FetchAllPeerSessionsMeta` always emits a group per probed peer; `RemoteSessionsPanel.tsx` renders per-peer states per UI-SPEC |
| RB-05 | A relay-surface regression test covers the discover→list→pick path (not just the webserver/fixture surface), guarding against the v3.5-class blind spot | New test in `internal/daemon/relay_remote_files_test.go` drives the new discovery endpoint through `api.RelayHandler()` if it is mounted there, or independently confirms the end-to-end relay path for file browse after discovery |
</phase_requirements>

---

## Summary

Phase 130 completes the "remote browse GUI on-ramp" — the discover→list→pick path the desktop GUI Remote Sessions panel needs to surface a tailnet peer's shareable sessions and hand off to the File Browser. The remote read/write data path is already proven live from Phase 128 (relay routes, proxy handlers, CORS). What is missing is a way for the Remote Sessions panel to get shareable-session metadata from a peer without holding a session-scoped cap.

The root cause of the current silent-drop (RB-01) is in `internal/tailnet/sessions.go` `FetchAllPeerSessions` (line 93): when `len(sessions) == 0` the peer is omitted from the result groups. `FetchPeerSessions` calls `doFetchSessions` which hits `/api/sessions` with no cap; the webserver's `requireCapability` middleware returns `401 "capability required"` (checked by `isAgentHubProbeResponse` for the probe, but `doFetchSessions` requires `200` at line 127 — so a 401 silently returns nil). The peer is probed as present (401 with the AgentHub marker passes `isAgentHubProbeResponse`) but then fetching its sessions returns empty (401 does not pass `doFetchSessions`'s `200` guard). With no sessions in the group the peer is not added to `groupMap`, and `GetRemoteSessions` in `app.go` never sees it.

The fix is a new webserver endpoint (e.g., `GET /api/sessions/meta`) that returns shareable-session metadata to callers without requiring a cap — "tailnet-trusted" by virtue of the webserver's Tailscale IP binding (only tailnet members can reach the Tailscale IP). The endpoint returns only the minimum fields needed to list and pick: session `id`, `name`, `cli_type`, `status`, and the session URL. It does NOT return a cap or any session content. A new `FetchPeerSessionsMeta` function in `internal/tailnet/sessions.go` calls this endpoint. `GetRemoteSessionsWithMeta` in `app.go` replaces `GetRemoteSessions`, now including all probed peers (even those with zero shareable sessions or that are unreachable), enabling honest per-peer states in the frontend.

**Primary recommendation:** Add `GET /api/sessions/meta` on the webserver (open, no cap required, metadata-only), add `FetchPeerSessionsMeta` in tailnet package, add `GetRemoteSessionsWithMeta` Wails RPC in `app.go`, extend `RemotePeerSessions` with `reachable bool` and use it in `RemoteSessionsPanel.tsx` for honest per-peer states. The relay loopback test in `internal/daemon/relay_remote_files_test.go` gains a new test exercising the discover→list→pick relay surface end-to-end.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Shareable-session metadata (serve it) | Webserver (HTTPS on Tailscale IP) | — | Needs to be reachable by tailnet peers over TLS; daemon socket is not reachable cross-machine |
| Tailnet-peer discovery + metadata fetch | Backend daemon (via `internal/tailnet`) | App.go (Wails binding) | Discovery uses local tailscaled; fetching hits remote peers over TLS |
| Wails RPC binding | App.go (top-level `main` package) | — | Wails binds exported `*App` methods; daemon logic delegated via `DaemonClient` |
| Per-peer session listing (GUI) | Browser / Wails webview | — | React component renders the peer list with per-peer states |
| Pick flow → File Browser | Browser / Wails webview + App.go | relay loopback | Join-code modal + `RegisterRemoteCap` + `FileBrowserTab` via relay |
| Relay-surface test coverage | `internal/daemon` test package | `internal/relay` test package | `daemon_test` drives `api.RelayHandler()` — the actual GUI path |

---

## Standard Stack

### Core (no new packages needed)

| Library | Purpose | Already in Codebase |
|---------|---------|---------------------|
| Go stdlib (`net/http`, `encoding/json`) | New webserver endpoint, metadata serialization | Yes |
| `internal/tailnet` | Peer discovery + new metadata fetch function | Yes (`sessions.go`) |
| `internal/webserver` | Mount new open endpoint on webserver mux | Yes (`server.go`) |
| `internal/daemon` | `API.registerRoutes` wiring, `DaemonClient.FetchPeersMeta` (if needed) | Yes (`api.go`) |
| `app.go` (`main` package) | New `GetRemoteSessionsWithMeta` Wails RPC | Yes |
| React / TypeScript + vitest | Frontend component extension, new test cases | Yes |

**No new npm or Go packages are required for this phase.** All functionality uses existing dependencies.

---

## Package Legitimacy Audit

No new external packages are introduced in Phase 130. The phase uses only existing codebase dependencies. This section is not applicable.

---

## Architecture Patterns

### System Architecture Diagram

```
Wails GUI (webview, wails://wails origin)
  │
  │  polls GetRemoteSessionsWithMeta() every 30s
  ▼
app.go: App.GetRemoteSessionsWithMeta()
  │
  │  DaemonClient.ListTailnetPeers()  →  daemon unix socket  →  GET /tailnet/peers
  │  (uses tailnetCache with 30s TTL via tailnet.DiscoverAndProbe)
  │
  │  tailnet.FetchAllPeerSessionsMeta(ctx, peers)
  │     ├── goroutine per peer (max 5 concurrent, errgroup.SetLimit(5))
  │     │     GET https://{peer.DNSName}:7443/api/sessions/meta
  │     │     (NEW open endpoint — no cap required)
  │     │     ├── 200 → []ShareableSessionMeta (id, name, cli_type, status, url)
  │     │     ├── non-200 / timeout → peer marked reachable=false
  │     └──── returns []PeerSessionMetaGroup (ALL peers, even empty/unreachable)
  │
  ▼
[]RemotePeerSessions (with reachable bool, sessions []RemoteSession)
  │
  ▼
RemoteSessionsPanel.tsx
  ├── peer.reachable=true, len(sessions)>0  → session rows + Browse Files / Open Session buttons
  ├── peer.reachable=true, len(sessions)=0  → "No shareable sessions" inline text
  └── peer.reachable=false                  → "Unreachable" text badge

User clicks "Browse Files"
  ▼
App.tsx: handleBrowseFilesRemote(sessionId, sessionName)
  ├── remoteCapsCached.has(sessionId) → handleOpenFileBrowser(sessionId, sessionName) directly
  └── else → setJoinModalForSession → RemoteJoinCodeModal (existing Phase 122 flow)
              │
              │ user pastes join code
              ▼
        ExchangeJoinCodeAtURL(remoteBaseURL, code)
              │  daemon → POST https://{peer}:7443/join/exchange
        RegisterRemoteCap(sessionId, baseURL, capToken)
              │  deposit into daemon RemoteCapStore
        handleOpenFileBrowser(sessionId, sessionName)
              ▼
        FileBrowserTab(isRemote=true, pathPrefix=/api/files/remote/{sid})
              │
              │  fetches through relay loopback  127.0.0.1:<relayPort>
              │  /api/files/remote/{sessionID}/list  →  daemon.wrapRelayWithRemoteFiles
              │  →  handleRemoteFilesList  →  RemoteCapStore.Get()  →  outbound HTTPS
              ▼
        Remote peer's webserver: GET /api/files/list?session=...&cap=...
```

### Recommended Project Structure (changes only)

```
internal/webserver/
  server.go                      # add GET /api/sessions/meta route (open, no cap)
                                 # new handleSessionsMeta handler

internal/tailnet/
  sessions.go                    # add FetchPeerSessionsMeta, FetchAllPeerSessionsMeta
                                 # add ShareableSessionMeta type, PeerSessionMetaGroup type

app.go                           # add GetRemoteSessionsWithMeta() replacing GetRemoteSessions()
                                 # add RemotePeerSessionsWithMeta type OR extend RemotePeerSessions

frontend/src/components/
  RemoteSessionsPanel.tsx        # extend RemotePeerSessions with reachable bool
                                 # render per-peer states per UI-SPEC

frontend/src/components/__tests__/
  RemoteSessionsPanel.test.tsx   # extend with per-peer state tests

frontend/src/wailsjs/go/main/
  App.d.ts                       # add GetRemoteSessionsWithMeta binding

internal/daemon/
  relay_remote_files_test.go     # add RB-05 relay-surface test for discover→list→pick path

internal/webserver/
  server_test.go (or new file)   # unit test for handleSessionsMeta (open, metadata-only)
```

### Pattern 1: Open Metadata-Only Endpoint on Webserver

**What:** Mount `GET /api/sessions/meta` on the webserver mux without any `requireCapability` wrapper. The endpoint returns shareable-session metadata for all web-enabled sessions.

**When to use:** When a tailnet peer needs to know which sessions are available to pick, before it has a cap for any of them.

**Security invariant preserved:** The endpoint is mounted on the webserver's TLS listener, which is bound to the Tailscale IP (`Config.BindIP`, typically `100.x.x.x`). Only tailnet members can reach that IP — this is the existing network-layer trust model (Phase 87/88). The endpoint returns metadata only: session ID, name, CLI type, status, and the session URL (which already appears in QR codes and is not secret). It does NOT return a cap token, session content, file listings, or any capability-granting token. An unauthorized/non-tailnet caller cannot reach the webserver at all; a tailnet caller who reaches it gets only enough to display a list.

**Contrast with existing `/api/sessions`:** The existing `GET /api/sessions` requires a cap token (D-18) and returns only the single session the cap is bound to. The new `/api/sessions/meta` requires no cap and returns metadata for ALL web-enabled sessions — this is intentional and safe because metadata (session name, status) is non-sensitive compared to session content.

**Example shape (handler pseudocode):**
```go
// Source: informed by internal/webserver/server.go handleListSessions pattern
func (ws *WebServer) handleSessionsMeta(w http.ResponseWriter, r *http.Request) {
    // No cap check — open, tailnet-trusted via network binding.
    ids := ws.webEnabledSessions() // existing method, server.go:200
    items := make([]sessionMetaItem, 0, len(ids))
    for _, id := range ids {
        name, cliType, st, _ := ws.sessionResolver(id) // existing resolver
        if name == "" { name = id }
        // Pre-build the session URL so the caller can use it for join-code modal.
        sessionURL := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), id)
        items = append(items, sessionMetaItem{
            ID:      id,
            Name:    name,
            CLIType: cliType,
            Status:  st,
            URL:     sessionURL,
        })
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}
```

### Pattern 2: Emit All Probed Peers (not just non-empty ones) — RB-04

**What:** In `FetchAllPeerSessionsMeta` (new function in `internal/tailnet/sessions.go`), always emit a group entry for every probed peer — even if the meta fetch fails (peer unreachable → `reachable=false`) or the meta list is empty (peer reachable, zero shareable sessions).

**Why:** `FetchAllPeerSessions` (existing) silently drops peers with empty session lists at line 93 (`if len(sessions) == 0 { return nil }`). This causes the silent-drop bug (RB-01). The new function includes all peers with a `reachable bool` discriminator so the frontend can render honest per-peer states.

**Existing type to extend (or parallel new type):**
```go
// internal/tailnet/sessions.go (existing)
type PeerSessionGroup struct {
    Hostname string        `json:"hostname"`
    Sessions []PeerSession `json:"sessions"`
}
```

New parallel type for the metadata path:
```go
// internal/tailnet/sessions.go (NEW)
type ShareableSessionMeta struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
}

type PeerSessionMetaGroup struct {
    Hostname  string                 `json:"hostname"`
    Reachable bool                   `json:"reachable"`
    Sessions  []ShareableSessionMeta `json:"sessions"`
}
```

### Pattern 3: Extend App.go RemotePeerSessions with Reachable Bool

The existing `RemotePeerSessions` type in `app.go` (line 58):
```go
type RemotePeerSessions struct {
    Hostname string          `json:"hostname"`
    Sessions []RemoteSession `json:"sessions"`
}
```

Must gain `Reachable bool` so the Wails binding exposes it to the frontend:
```go
type RemotePeerSessions struct {
    Hostname  string          `json:"hostname"`
    Reachable bool            `json:"reachable"`
    Sessions  []RemoteSession `json:"sessions"`
}
```

The new Wails RPC `GetRemoteSessionsWithMeta` replaces `GetRemoteSessions`. It may reuse the same return type `[]RemotePeerSessions` with the added `Reachable` field, or use a new named type — the planner should keep the type name consistent so the `wailsjs` binding stubs do not diverge unnecessarily.

**Note on naming:** The CONTEXT and UI-SPEC reference a new RPC `GetRemoteSessionsWithMeta`. The planner may choose to RENAME `GetRemoteSessions` (updating all callsites) or ADD `GetRemoteSessionsWithMeta` alongside it and deprecate the old one. Renaming is cleaner but requires updating `App.d.ts`, `App.tsx` (line 26 import, line 895 call), and the wailsjs stub.

### Pattern 4: Frontend Per-Peer State Rendering (per UI-SPEC)

The `RemoteSessionsPanel.tsx` currently maps `peers` and renders all sessions uniformly. After this phase it must branch on `peer.reachable` and `peer.sessions.length`:

```tsx
// Source: informed by 130-UI-SPEC.md and existing RemoteSessionsPanel.tsx
{peers.map((peer) => (
  <div key={peer.hostname} className="remote-panel__peer">
    <div className="remote-panel__peer-header">{peer.hostname}</div>
    {!peer.reachable ? (
      <div className="remote-panel__peer-unreachable">Unreachable</div>
    ) : peer.sessions.length === 0 ? (
      <>
        <div className="remote-panel__peer-meta">Shows shareable sessions</div>
        <div className="remote-panel__peer-empty-sessions">
          <div className="remote-panel__peer-empty-sessions-title">No shareable sessions</div>
          <div className="remote-panel__peer-empty-sessions-body">
            This peer has no sessions with web-sharing enabled.
          </div>
        </div>
      </>
    ) : (
      <>
        <div className="remote-panel__peer-meta">Shows shareable sessions</div>
        <div className="remote-panel__session-list">
          {peer.sessions.map((s) => (/* existing row markup */))}
        </div>
      </>
    )}
  </div>
))}
```

The meta copy change: "Shows web-enabled sessions only" → "Shows shareable sessions" (per UI-SPEC Copywriting Contract). **This existing string literal in `RemoteSessionsPanel.test.tsx` at line 134 will need to be updated** in the test.

### Anti-Patterns to Avoid

- **Putting cap-gating on the metadata endpoint:** Defeats the purpose. The endpoint is intentionally open on the webserver (tailnet-trusted via network binding). Do not wrap it with `requireCapability`.
- **Returning session content or cap tokens from the metadata endpoint:** Violates RB-03 and the Phase 87/88 no-enumeration model. Only `id`, `name`, `cli_type`, `status`, `url` are allowed.
- **Mounting the metadata endpoint on the daemon socket instead of the webserver:** The daemon socket is not reachable by remote tailnet peers. The endpoint MUST be on the webserver (HTTPS, Tailscale IP).
- **Mounting the metadata endpoint on the relay loopback:** Same problem — the relay is `127.0.0.1:` only, not reachable cross-machine.
- **Dropping peers with zero shareable sessions:** The entire point of RB-04 is to show all peers honestly. Never filter out peers for having no sessions.
- **Dropping unreachable peers:** Same — `reachable=false` peers must appear in the list with the "Unreachable" badge.
- **Reusing `FetchAllPeerSessions` / `doFetchSessions` for the metadata path:** These functions target `/api/sessions` with cap-gated semantics. The metadata path needs a new function targeting `/api/sessions/meta` (the new open endpoint).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Concurrent peer probing with limit | Custom goroutine pool | `errgroup.SetLimit(5)` — already used in `FetchAllPeerSessions` (tailnet/sessions.go:88) |
| IP fallback when DNS fails | Custom retry logic | Copy `FetchPeerSessions` pattern (sessions.go:49-61) — try DNS, fall back to first TailscaleIP with ServerName set |
| JSON encoding | Custom serialization | `json.NewEncoder(w).Encode(items)` — existing pattern throughout |
| TLS client for outbound peer fetch | Skip-verify hack | `&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}}` — same as `FetchPeerSessions` (sessions.go:41-45) |
| Wails type stubs | Manual .d.ts edits | Wails auto-generates `wailsjs/go/main/App.d.ts` from exported `*App` methods — update `App.d.ts` manually as is done in the codebase (no wails codegen step in tests) |

---

## RB-01 Root Cause: The Exact Silent-Drop Location

**File:** `internal/tailnet/sessions.go`
**Function:** `FetchAllPeerSessions` (line 83)
**Drop location:** Line 93:

```go
g.Go(func() error {
    sessions := FetchPeerSessions(gctx, p)
    if len(sessions) == 0 {
        return nil  // <-- SILENT DROP. Peer never added to groupMap.
    }
    mu.Lock()
    groupMap[p.Hostname] = append(groupMap[p.Hostname], sessions...)
    mu.Unlock()
    return nil
})
```

**Why sessions is empty:** `FetchPeerSessions` calls `doFetchSessions` which sends `GET /api/sessions` with no cap. The webserver's `requireCapability` middleware returns `401 "capability required"`. `doFetchSessions` checks `resp.StatusCode != http.StatusOK` at line 127 and returns nil. Back in `FetchPeerSessions`, nil means zero sessions → `groupMap` never gets this peer.

**Note:** `probePeer` / `isAgentHubProbeResponse` DOES accept the 401+marker (the #84 fix, commit `3508bd7`) — so the peer IS discovered as an AgentHub peer. But the subsequent `FetchAllPeerSessions` call silently drops it because its session list comes back empty (401 ≠ 200).

**Fix:** New `FetchAllPeerSessionsMeta` function calls `GET /api/sessions/meta` (the new open endpoint) instead of `/api/sessions`. Always emits a group per peer. Uses `reachable=false` when the fetch fails (timeout, connection error) rather than silently dropping.

---

## Tailnet-Trust Mechanism (RB-03)

**How "tailnet-trusted" currently works in this codebase:** There is NO application-layer tailnet-peer authentication check. The webserver's trust model is pure network-layer:

1. `Config.BindIP` is the Tailscale IP (e.g., `100.x.x.x`) set by the daemon when starting the webserver.
2. The webserver listens on `{BindIP}:{Port}` (default 7443) with TLS.
3. Only tailnet members can route to a `100.x.x.x` address — Tailscale's WireGuard mesh enforces this.
4. No `tsnet.Server` whois or application-layer IP check is needed or present.

**For the new `/api/sessions/meta` endpoint:** The same trust model applies. Mounting it on the webserver mux (which is bound to the Tailscale IP) is sufficient for "tailnet-trusted." A non-tailnet caller cannot reach the Tailscale IP. The endpoint does NOT need a cap, but it also returns ONLY metadata — no content, no cap, no grant.

**What the metadata endpoint must NOT return (to preserve RB-03):**
- Cap tokens
- Session file content
- Join codes
- The `grants` map or any grant IDs
- The HMAC signing key
- Any data that would let an observer impersonate the session owner

**What the metadata endpoint returns (minimum to list+pick):**
- `id` (session ID — already public; appears in session URLs in QR codes)
- `name` (session display name — non-sensitive)
- `cli_type` (e.g., "claude", "codex" — non-sensitive, already in TUI listings)
- `status` (running/idle/waiting/errored — non-sensitive)
- `url` (the session URL — already baked into QR codes and share links)

---

## Common Pitfalls

### Pitfall 1: Registering the metadata route on the daemon socket instead of the webserver

**What goes wrong:** Remote tailnet peers probe `https://{peer.DNSName}:7443/api/sessions/meta`. The daemon socket is a Unix socket / Windows named pipe — it is not reachable cross-machine. If the route is only on the daemon socket, the metadata fetch returns a connection error from all remote peers.

**How to avoid:** Register `GET /api/sessions/meta` in `ws.setupRoutes()` (`internal/webserver/server.go`) — the same place as the existing `/api/sessions` route. Do NOT add it only to `registerRoutes()` in `internal/daemon/api.go`.

**Warning signs:** All peers appear `reachable=false` in the panel despite being probed as present.

### Pitfall 2: The relay loopback (127.0.0.1:<relayPort>) is not the metadata discovery path

**What goes wrong:** Developer thinks "the GUI goes through the relay, so mount the metadata endpoint there." The relay is a local TCP server bound to `127.0.0.1` — it is the path the Wails webview uses for LOCAL file ops and remote file proxy. The DISCOVERY of remote peer sessions happens from the local machine's backend (app.go / daemon) making outbound HTTPS requests to remote peers' webservers. The relay is not involved in discovery.

**How to avoid:** The metadata fetch path is: `app.go` → `tailnet.FetchAllPeerSessionsMeta` → outbound HTTPS to `https://{peer.DNSName}:7443/api/sessions/meta`. The relay is only involved AFTER a session is picked (for the file browse proxy).

### Pitfall 3: The metadata endpoint must NOT require a cap (defeats the purpose)

**What goes wrong:** Developer adds `ws.requireCapability(ws.handleSessionsMeta)` on the route. Remote peers still can't discover sessions, because they have no cap (that's exactly the problem being solved).

**How to avoid:** Mount the handler WITHOUT any capability wrapper. The route is: `mux.HandleFunc("GET /api/sessions/meta", ws.handleSessionsMeta)` — no middleware.

### Pitfall 4: Updating only App.d.ts without updating App.tsx

**What goes wrong:** The new `GetRemoteSessionsWithMeta` RPC is added to `App.d.ts` but `App.tsx` still calls `GetRemoteSessions`. The panel never shows the new data.

**How to avoid:** Update both files. Also update the import at `App.tsx:26` and the call site at `App.tsx:895`.

### Pitfall 5: Existing RemoteSessionsPanel tests reference "Shows web-enabled sessions only"

**What goes wrong:** `RemoteSessionsPanel.test.tsx` line 134 asserts `'Shows web-enabled sessions only'`. After the UI-SPEC copy change to "Shows shareable sessions", this test will fail.

**How to avoid:** Update the test string at `RemoteSessionsPanel.test.tsx` line 89 (source inspection test for the literal) and line 134/135 (DOM test). Also update the source inspection test at line 89 (`expect(raw).toContain('Shows web-enabled sessions only')`).

### Pitfall 6: FetchPeerSessionsMeta must use a new HTTP client per call (not reuse one)

**What goes wrong:** Sharing a single `*http.Client` across concurrent goroutines is safe, but the IP-fallback path creates a NEW client with a custom TLS ServerName per peer. If the client is reused with a per-peer ServerName the TLS is incorrect for other peers.

**How to avoid:** Follow the existing `FetchPeerSessions` pattern (sessions.go:38-46): create a fresh `*http.Client` inside `FetchPeerSessionsMeta` per call, or use the same shared client for the DNS path and a new IP-fallback client only for the IP path.

### Pitfall 7: The RB-05 relay-surface test scope

**What goes wrong:** The new relay test only confirms the local file routes still work, or only drives the old `/api/sessions` probe. It doesn't cover the discover→list→pick relay path.

**The relay surface for RB-05:** The discover→list→pick path itself does NOT go through the relay (see Pitfall 2 — discovery is outbound HTTPS from app.go/tailnet). What the relay surface MUST cover for RB-05 is: **after discovery, the actual file browse** (the pick → cap deposit → relay proxy → remote file list). This is already partially covered by `TestRemoteFiles_MountedOnRelay`, but a new test can add an explicit end-to-end scenario: "given a metadata-discovered session, the file browse relay path works." Alternatively, RB-05 can be satisfied by a test that drives the new metadata endpoint through a fixture peer AND confirms the relay list path still works — tying the two together. The planner should create a test in `internal/daemon/relay_remote_files_test.go` named e.g. `TestRemoteFiles_DiscoverAndBrowse_RelaySurface` that uses `newFixtureRemotePeer` with a `/api/sessions/meta` handler added, and confirms the relay browse path resolves to a 200.

---

## Code Examples

### Adding the route to the webserver (server.go:setupRoutes)

```go
// Source: internal/webserver/server.go setupRoutes() pattern
// Add after the existing GET /api/sessions route (~line 458).
// Open — no capability wrapper. Tailnet-trusted via network binding (BindIP = Tailscale IP).
mux.HandleFunc("GET /api/sessions/meta", ws.handleSessionsMeta)
```

### New handler in server.go

```go
// Source: pattern from handleListSessions (server.go:733) + handleSessionsMeta
type sessionMetaItem struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
}

// handleSessionsMeta handles GET /api/sessions/meta.
// Returns metadata for all web-enabled sessions to tailnet-trusted callers.
// Open — no capability required. Trust boundary is the Tailscale IP binding.
// Returns only metadata (id, name, cli_type, status, url) — never content or caps.
func (ws *WebServer) handleSessionsMeta(w http.ResponseWriter, r *http.Request) {
    ids := ws.webEnabledSessions()
    items := make([]sessionMetaItem, 0, len(ids))
    for _, id := range ids {
        name, cliType, st := "", "", ""
        if ws.sessionResolver != nil {
            name, cliType, st, _ = ws.sessionResolver(id)
        }
        if name == "" {
            name = id
        }
        sessionURL := fmt.Sprintf("%s/sessions/%s", ws.BaseURL(), id)
        items = append(items, sessionMetaItem{
            ID:      id,
            Name:    name,
            CLIType: cliType,
            Status:  st,
            URL:     sessionURL,
        })
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items) //nolint:errcheck
}
```

### New tailnet functions (sessions.go)

```go
// Source: pattern from FetchPeerSessions / FetchAllPeerSessions (sessions.go)

type ShareableSessionMeta struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
}

type PeerSessionMetaGroup struct {
    Hostname  string                 `json:"hostname"`
    Reachable bool                   `json:"reachable"`
    Sessions  []ShareableSessionMeta `json:"sessions"`
}

// FetchPeerSessionsMeta fetches /api/sessions/meta from a single peer.
// Returns (sessions, true) on success; (nil, false) on any error (unreachable).
// Mirrors FetchPeerSessions pattern with DNS→IP fallback.
func FetchPeerSessionsMeta(ctx context.Context, peer Peer) ([]ShareableSessionMeta, bool) {
    fqdn := strings.TrimSuffix(peer.DNSName, ".")
    client := &http.Client{
        Timeout:   5 * time.Second,
        Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
    }
    url := fmt.Sprintf("https://%s:%d/api/sessions/meta", fqdn, DefaultProbePort)
    sessions := doFetchSessionsMeta(ctx, url, client, "")
    if sessions == nil && len(peer.TailscaleIPs) > 0 {
        // IP fallback — same pattern as FetchPeerSessions lines 48-61
        ipClient := &http.Client{
            Timeout:   5 * time.Second,
            Transport: &http.Transport{TLSClientConfig: &tls.Config{ServerName: fqdn, MinVersion: tls.VersionTLS12}},
        }
        ipURL := fmt.Sprintf("https://%s:%d/api/sessions/meta", peer.TailscaleIPs[0], DefaultProbePort)
        host := fmt.Sprintf("%s:%d", fqdn, DefaultProbePort)
        sessions = doFetchSessionsMeta(ctx, ipURL, ipClient, host)
    }
    if sessions == nil {
        return nil, false // unreachable
    }
    // Enrich URL if empty (defensive)
    for i := range sessions {
        if sessions[i].URL == "" {
            sessions[i].URL = fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, DefaultProbePort, sessions[i].ID)
        }
    }
    return sessions, true
}

// FetchAllPeerSessionsMeta discovers shareable-session metadata from ALL given peers
// concurrently (max 5 goroutines). Returns ALL peers in the result — even unreachable
// ones (reachable=false) and those with zero shareable sessions. This is the RB-04 fix:
// no peer is silently dropped.
func FetchAllPeerSessionsMeta(ctx context.Context, peers []Peer) []PeerSessionMetaGroup {
    var mu sync.Mutex
    groups := make([]PeerSessionMetaGroup, 0, len(peers))

    g, gctx := errgroup.WithContext(ctx)
    g.SetLimit(5)
    for _, p := range peers {
        p := p
        g.Go(func() error {
            sessions, reachable := FetchPeerSessionsMeta(gctx, p)
            if sessions == nil {
                sessions = []ShareableSessionMeta{}
            }
            mu.Lock()
            groups = append(groups, PeerSessionMetaGroup{
                Hostname:  p.Hostname,
                Reachable: reachable,
                Sessions:  sessions,
            })
            mu.Unlock()
            return nil
        })
    }
    _ = g.Wait()
    sort.Slice(groups, func(i, j int) bool { return groups[i].Hostname < groups[j].Hostname })
    return groups
}
```

### New Wails RPC in app.go

```go
// Source: pattern from GetRemoteSessions (app.go:1082)

// GetRemoteSessionsWithMeta discovers tailnet peers and fetches their shareable-session
// metadata via the new /api/sessions/meta endpoint. Returns ALL probed peers, including
// unreachable ones (Reachable=false) and peers with zero shareable sessions — enabling
// honest per-peer states in the Remote Sessions panel (RB-01, RB-04).
func (a *App) GetRemoteSessionsWithMeta() []RemotePeerSessions {
    if a.client == nil {
        return []RemotePeerSessions{}
    }
    peers, err := a.client.ListTailnetPeers()
    if err != nil || len(peers) == 0 {
        return []RemotePeerSessions{}
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    groups := tailnet.FetchAllPeerSessionsMeta(ctx, peers)

    results := make([]RemotePeerSessions, 0, len(groups))
    for _, g := range groups {
        sessions := make([]RemoteSession, 0, len(g.Sessions))
        for _, s := range g.Sessions {
            sessions = append(sessions, RemoteSession{
                ID:      s.ID,
                Name:    s.Name,
                CLIType: s.CLIType,
                Status:  s.Status,
                URL:     s.URL,
            })
        }
        results = append(results, RemotePeerSessions{
            Hostname:  g.Hostname,
            Reachable: g.Reachable,
            Sessions:  sessions,
        })
    }
    return results
}
```

(This requires adding `Reachable bool` to `RemotePeerSessions` in `app.go:58`.)

### doFetchSessionsMeta helper (sessions.go)

```go
// Source: pattern from doFetchSessions (sessions.go:114)
func doFetchSessionsMeta(ctx context.Context, url string, client *http.Client, host string) []ShareableSessionMeta {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil
    }
    if host != "" {
        req.Host = host
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil
    }
    var items []ShareableSessionMeta
    if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
        return nil
    }
    return items // may be empty slice (valid: peer has zero shareable sessions)
}
```

**Subtle difference from `doFetchSessions`:** When the server returns `200 []` (empty JSON array), this function returns `[]ShareableSessionMeta{}` (non-nil empty slice), signaling "reachable but zero sessions." `doFetchSessions` returned nil on decode failure only; here we must distinguish `nil` (unreachable, error) from `[]` (reachable, empty). The caller `FetchPeerSessionsMeta` uses the `nil` vs non-nil return to set `reachable`.

---

## State of the Art

| Old Approach | Current Approach | Impact for Phase 130 |
|--------------|------------------|----------------------|
| `/api/sessions` enumeration (Phase 52) | Cap-gated `/api/sessions` (Phase 87, D-18) | This is WHY we need a new open metadata endpoint |
| Peer probe: 200-only (pre-#84) | Probe accepts 401+marker (commit 3508bd7) | Peers are discovered but sessions still silently dropped |
| FetchAllPeerSessions drops empty peers | FetchAllPeerSessionsMeta (Phase 130) includes all peers | RB-01 + RB-04 fix |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `wailsjs/go/main/App.d.ts` file is hand-maintained (not generated during the build/test cycle) — so updating it manually is correct | Standard Stack | If Wails codegen runs during `wails build` and overwrites it, the manual edit is lost — but this matches the existing codebase pattern; all other methods are manually documented there |
| A2 | `webEnabledSessions()` (webserver/server.go:200) is the correct source for which sessions appear in the metadata endpoint | Code Examples | If the set of "shareable" sessions diverges from "web-enabled" sessions in a future phase, the metadata endpoint would need a separate filter |

---

## Open Questions (RESOLVED)

1. **Old `GetRemoteSessions` — replace or keep alongside?**
   - What we know: `App.tsx:26` imports and `App.tsx:895` calls `GetRemoteSessions`. `App.d.ts:112` declares it.
   - What's unclear: Whether to rename it to `GetRemoteSessionsWithMeta` (single clean function, requires updating all callsites) or add the new one alongside and soft-deprecate the old one.
   - Recommendation: Rename to `GetRemoteSessionsWithMeta` — cleaner, no dead code. Update App.tsx import (line 26) and call (line 895). Update App.d.ts declaration. The planner may keep the old function as a no-op redirect for backward compatibility if the remoteSession lib tests need it, but that is unnecessary given all callsites are within this project.

2. **What happens to `findRemoteSession` (lib/remoteSession.ts) after the type change?**
   - What we know: `findRemoteSession` iterates `peer.sessions` on the existing `RemotePeerSessions[]` type. Adding `reachable` is a non-breaking extension in TypeScript.
   - Recommendation: No changes needed to `remoteSession.ts` — the `reachable` field is additive. `findRemoteSession` only reads `peer.sessions` which is still present.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Build + test | ✓ | go 1.22+ (project uses method-prefixed mux patterns) | — |
| pnpm | Frontend test | ✓ | (existing) | — |
| vitest | Frontend unit tests | ✓ | 4.1.0 | — |
| Tailscale daemon (live) | Production discovery | Manual UAT only | — | Tests use httptest + fake statusFunc |

**No missing dependencies.** All tooling is already installed and verified (87 test files, 1300 tests passing; Go packages pass).

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | `testing` (stdlib) |
| Go test run (fast) | `go test ./internal/tailnet/... ./internal/relay/... ./internal/daemon/... -count=1` |
| Go test run (full) | `go test ./...` |
| Frontend framework | vitest 4.1.0 |
| Frontend config | `frontend/vite.config.ts` |
| Frontend run (fast) | `cd frontend && pnpm test` (runs all 87 files) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| RB-01 | `/api/sessions/meta` returns shareable-session metadata without a cap | unit (Go) | `go test ./internal/webserver/... -run TestSessionsMeta` | ❌ Wave 0 |
| RB-01 | `FetchAllPeerSessionsMeta` includes all probed peers (no silent drop) | unit (Go) | `go test ./internal/tailnet/... -run TestFetchAllPeerSessionsMeta` | ❌ Wave 0 |
| RB-01 | `GetRemoteSessionsWithMeta` Wails RPC returns peers with `Reachable` field | unit (Go, app_test.go) | `go test -run TestGetRemoteSessionsWithMeta` | ❌ Wave 0 |
| RB-02 | "Browse Files" click triggers `onBrowseFiles(sessionId, sessionName)` | unit (vitest) | `cd frontend && pnpm test -- --reporter=verbose` | ✅ (existing, App.remoteFileBrowser.test.tsx) — may need extension |
| RB-03 | `/api/sessions/meta` does NOT return caps, grants, or content | unit (Go) | `go test ./internal/webserver/... -run TestSessionsMeta_NoCapInResponse` | ❌ Wave 0 |
| RB-03 | Non-tailnet IP cannot reach webserver metadata (network-layer, not unit-testable) | manual-only | — | N/A — trust is network-layer |
| RB-04 | Panel renders "Unreachable" badge for unreachable peers | unit (vitest) | `cd frontend && pnpm test -- --reporter=verbose -t "unreachable"` | ❌ Wave 0 |
| RB-04 | Panel renders "No shareable sessions" for reachable peers with empty list | unit (vitest) | `cd frontend && pnpm test -- --reporter=verbose -t "no shareable sessions"` | ❌ Wave 0 |
| RB-04 | Panel NEVER shows "No remote peers found" when at least one peer is probed | unit (vitest) | `cd frontend && pnpm test -- --reporter=verbose` | ❌ Wave 0 |
| RB-05 | Relay-surface test: discover→list→pick path reaches the proxy through api.RelayHandler() | unit (Go) | `go test ./internal/daemon/... -run TestRemoteFiles_DiscoverAndBrowse_RelaySurface` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/tailnet/... ./internal/relay/... ./internal/daemon/... -count=1 && cd frontend && pnpm test`
- **Per wave merge:** `go test ./...` + `cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps (test files to create or extend)

- [ ] `internal/webserver/server_test.go` or `internal/webserver/sessions_meta_test.go` — covers RB-01 (`TestSessionsMeta_ReturnsWebEnabledSessions`, `TestSessionsMeta_NoCap`, `TestSessionsMeta_EmptyWhenNoneEnabled`), RB-03 (`TestSessionsMeta_NoCapInResponse`, `TestSessionsMeta_NoContentOrGrants`)
- [ ] `internal/tailnet/tailnet_test.go` (extend) — covers RB-01 (`TestFetchAllPeerSessionsMeta_IncludesUnreachablePeers`, `TestFetchAllPeerSessionsMeta_EmptySessionsNotDropped`, `TestFetchPeerSessionsMeta_IPFallback`)
- [ ] `app_test.go` (extend) — covers RB-01 (`TestGetRemoteSessionsWithMeta_ReachableField`)
- [ ] `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` (extend) — covers RB-04 (per-peer state rendering: unreachable badge, zero-sessions text, existing tests updated for copy change "Shows shareable sessions")
- [ ] `internal/daemon/relay_remote_files_test.go` (extend) — covers RB-05 (`TestRemoteFiles_DiscoverAndBrowse_RelaySurface` — uses fixture peer with `/api/sessions/meta` handler, deposits cap, confirms relay browse path)

*(Existing tests in `internal/relay/server_files_test.go` and `internal/daemon/relay_remote_files_test.go` continue to pass unchanged.)*

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No (endpoint is intentionally open on tailnet-trusted surface) | — |
| V3 Session Management | No | — |
| V4 Access Control | Yes — metadata-only, no cap, no content | Network-layer trust (Tailscale IP binding); no application-layer cap on new endpoint |
| V5 Input Validation | Yes — session IDs used in URL construction | Existing `sessionResolver` validates session existence; URL built from known BaseURL + validated id |
| V6 Cryptography | No new crypto | — |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Metadata enumeration by non-tailnet attacker | Information Disclosure | Network-layer: webserver bound to Tailscale IP only; non-tailnet callers cannot reach it |
| Cap token in metadata response | Information Disclosure | Code contract: `handleSessionsMeta` MUST NOT include cap tokens, grants, or signing key in the response |
| Spoofed tailnet peer requesting metadata | Spoofing | Tailscale WireGuard mesh prevents IP spoofing on the tailnet; additional application-layer protection is out of scope per CONTEXT §Decisions |
| Stale session appearing in metadata after session exit | Tampering | `webEnabledSessions()` reads the live `webEnabled` map; `runSessionExitCleanup` calls `DisableSession` which removes from the map — metadata endpoint reflects live state |

---

## Project Constraints (from CLAUDE.md)

| Directive | Impact on Phase 130 |
|-----------|---------------------|
| Go: `go fmt`, `golangci-lint`, context-aware functions with `ctx context.Context` | All new Go functions must accept `ctx context.Context` as first param; run `go fmt` before commit |
| JS/TS: `camelCase`, `PascalCase` components, ESLint + Prettier | `RemoteSessionsPanel.tsx` changes must use camelCase props, PascalCase component name |
| pnpm preferred | Use `pnpm test`, not `npm test` |
| NEVER `kill node.exe` | N/A for this phase |
| Use LSP over grep for code navigation | Planner/executor should use LSP for cross-references |
| Make beliefs pay rent — explicit predictions before significant actions | Test assertions before claiming a fix works |
| Silent Fallbacks Forbidden — let it crash | No `|| {}` or `|| []` silent defaults in new Go code |

---

## Sources

### Primary (HIGH confidence)

- `internal/tailnet/sessions.go` — complete read; FetchAllPeerSessions drop at line 93 confirmed [VERIFIED: codebase]
- `internal/tailnet/tailnet.go` — complete read; isAgentHubProbeResponse (line 98), probePeer patterns [VERIFIED: codebase]
- `internal/webserver/server.go` — complete read; setupRoutes, handleListSessions, capability_mw [VERIFIED: codebase]
- `internal/webserver/capability_mw.go` — complete read; requireCapability, requireFilesRead, requireFilesWrite [VERIFIED: codebase]
- `internal/relay/server.go` — complete read; NewServer, wrapRelayWithRemoteFiles pattern [VERIFIED: codebase]
- `internal/relay/server_files_test.go` — complete read; existing relay-surface tests [VERIFIED: codebase]
- `internal/daemon/relay_remote_files.go` — complete read; wrapRelayWithRemoteFiles [VERIFIED: codebase]
- `internal/daemon/relay_remote_files_test.go` — complete read; newFixtureRemotePeer, newDaemonAPIWithUpstreamCert, newSandboxBackedRemotePeer, fixtureCap [VERIFIED: codebase]
- `internal/daemon/remote_files_parity_test.go` — complete read; test harness fixtures [VERIFIED: codebase]
- `internal/daemon/api.go` — partial read; registerRoutes, handleTailnetPeers, RelayHandler [VERIFIED: codebase]
- `internal/daemon/types.go` — complete read; SessionInfo, RemotePeerSessions (in app.go) [VERIFIED: codebase]
- `internal/daemon/tailnet_cache.go` — complete read; cacheTTL=30s [VERIFIED: codebase]
- `app.go` — partial read; GetRemoteSessions (line 1082), RemotePeerSessions type (line 58), RemoteSession type (line 49) [VERIFIED: codebase]
- `frontend/src/components/RemoteSessionsPanel.tsx` — complete read [VERIFIED: codebase]
- `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` — complete read; existing test strings [VERIFIED: codebase]
- `frontend/src/App.tsx` — partial read; GetRemoteSessions call (line 895), handleBrowseFilesRemote (line 984), state declarations (line 184-196) [VERIFIED: codebase]
- `frontend/src/lib/remoteSession.ts` — complete read [VERIFIED: codebase]
- `frontend/src/wailsjs/go/main/App.d.ts` — complete read; existing RPC declarations [VERIFIED: codebase]
- `130-CONTEXT.md`, `130-UI-SPEC.md`, `REQUIREMENTS.md`, `STATE.md` — complete read [VERIFIED: codebase]

### Secondary (MEDIUM confidence)

- Go test baseline: `go test ./internal/tailnet/... ./internal/relay/... ./internal/daemon/...` — all pass [VERIFIED: test run]
- Frontend baseline: `cd frontend && pnpm test` — 87 files, 1300 tests pass [VERIFIED: test run]

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing codebase dependencies; no new packages
- Architecture: HIGH — root cause confirmed by code read; trust model confirmed by webserver binding analysis
- Pitfalls: HIGH — derived from reading existing code, not training data assumptions
- Relay test strategy: HIGH — existing test harness (`newFixtureRemotePeer`, `newDaemonAPIWithUpstreamCert`, `depositCapOnSocket`) fully mapped

**Research date:** 2026-06-15
**Valid until:** 2026-07-15 (stable codebase; no fast-moving external dependencies)
