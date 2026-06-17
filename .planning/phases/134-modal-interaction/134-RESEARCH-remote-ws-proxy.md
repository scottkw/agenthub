# Phase 134 (expansion): Cap-Gated Remote Terminal WebSocket Proxy — Research

**Researched:** 2026-06-17
**Domain:** Go reverse-WS-proxy on the daemon relay surface; React/xterm.js terminal client URL routing; tailnet capability auth
**Confidence:** HIGH (all findings verified against in-repo source; no external/training-data claims for the core design)

## Summary

Phase 134's modal correctly reuses `TerminalPanel`/`RelayClient` for LOCAL sessions but is structurally broken for REMOTE sessions (134-REVIEW CR-01/CR-02): both the interactive terminal WS and the briefing tail/send connect to the LOCAL relay (`ws://127.0.0.1:{relayPort}/sessions/{id}/ws`) and the LOCAL daemon (`e.manager.Get(id)`) using a session id that only exists on the remote peer. The Phase 122 join-code cap is consumed only by the HTTP file-browse proxy, never by the terminal WS.

The good news: **the peer end already exists and is fully cap-gated.** The remote peer's webserver serves the terminal WS at `GET /sessions/{id}/ws?cap=<token>` behind the middleware chain `requireAllowedOrigin → requireCapability → handleWSSRelay` (`internal/webserver/server.go:610-611`). It enforces HMAC verification, SID-match, grant-active, and web-enabled checks, and derives read/write from `claims.Perms`. The scrollback snapshot is replayed on connect (`server.go:997-998`) — which means **the tail and the live terminal are the same stream**: there is no separate HTTP tail endpoint to proxy.

So the minimal correct design is a single new daemon route that mirrors the existing remote-files proxy (`relay_remote_files.go` / `remote_files.go`) but for WebSocket: accept the client WS on the relay loopback surface, look up `(baseURL, capToken)` from the existing `RemoteCapStore`, dial the peer's `wss://{baseURL host}/sessions/{sid}/ws?cap=<token>` with the InsecureSkipVerify tailnet transport AND a synthesized `Origin: {baseURL}` header, then bidirectionally copy frames. The frontend selects this proxy URL vs the local-direct URL via one new optional param on `RelayClient`/`TerminalPanel`.

**Primary recommendation:** Add `GET /api/relay/remote/{sessionID}/ws` to the relay parent mux (in `relay_remote_files.go`), implemented as a new `*API` method `handleRemoteSessionWS` that upgrades the inbound WS, dials the peer WS using the cap, and runs two copy goroutines. Reuse the existing cap store and join-code exchange unchanged — investigation confirms the existing cap's `Perms` already gates the WS, so no scope addition is needed (the share grant's perms decide read vs read+write, exactly as for web-share viewers). Frontend: add a `wsURL?: string` (or `remoteBaseURL?: string`) seam to `RelayClient`/`TerminalPanel`; when set, build the daemon-proxy URL instead of the local one. Fix CR-03 (leak/race/unmount) in the same pass. Briefing tail for remote = read the scrollback snapshot frames off the proxied WS rather than calling `GetSessionTailLines`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Terminal WS upgrade (client side) | Frontend (RelayClient) | — | Browser WebSocket API; identical framing for local and remote |
| Local→peer WS proxy | Daemon (relay loopback surface) | — | Webview can only reach 127.0.0.1:<relayPort>, not the unix socket; must mirror file proxy placement |
| Cap lookup / storage | Daemon (`RemoteCapStore`) | — | Already owns per-session (baseURL, capToken); WS proxy reuses verbatim |
| Cap minting / join-code exchange | Peer webserver (`/join/exchange`) | Frontend (`ExchangeJoinCodeAtURL`) | Unchanged Phase 122 path; cap already carries the perms the WS needs |
| WS auth enforcement | Peer webserver (`requireCapability`) | — | HMAC verify + SID match + grant-active live on the peer; proxy is a dumb pipe |
| Origin enforcement (peer) | Peer webserver (`requireAllowedOrigin`) | Daemon proxy (synthesizes Origin) | Peer requires Origin == its BaseURL; proxy must inject it because a Go WS dialer sends none |
| Origin enforcement (local relay) | Daemon relay (`loopbackOriginPatterns`) | — | The inbound client WS is still loopback/Wails; same allowlist applies |
| Briefing tail (remote) | Peer webserver (scrollback snapshot on WS connect) | Frontend | No HTTP tail endpoint exists; tail == first frames of the WS stream |

## Standard Stack

This is an in-tree feature; no new external dependencies are required or recommended.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.14 | WS accept (relay) + dial (proxy upstream) | Already the only WS lib in the repo (`relay/server.go`, `webserver/server.go`); has both `Accept` and `Dial` with `DialOptions.HTTPClient` for custom TLS [VERIFIED: go.mod + grep] |
| Go stdlib `net/http`, `crypto/tls`, `io` | go.mod toolchain | TLS transport, header injection, frame copy | Mirrors `remote_files.go`'s `newRemoteFilesHTTPClient()` shape exactly [VERIFIED: in-repo] |
| `@xterm/xterm` + existing `RelayClient` | in `frontend/package.json` | Terminal rendering + framing | Reused unchanged; only the WS URL changes [VERIFIED: relayClient.ts] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `internal/daemon.RemoteCapStore` | in-repo | (baseURL, capToken) per session | Looked up by the WS proxy exactly as `proxyRemoteFiles` does [VERIFIED] |
| `internal/capability` | in-repo | HMAC verify (peer side only) | No change; the proxy never inspects the token, just forwards it [VERIFIED] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New `/api/relay/remote/{sid}/ws` proxy | Restrict modal to local sessions; route remote to `BrowserOpenURL` (CR-01 fix option a) | Simpler, zero new Go code, but does NOT satisfy the user-approved expansion (interactive remote terminal in-modal). Keep as the documented fallback if the proxy slips. |
| Frame-level `io.Copy` of WS messages | Re-frame/parse each message in the proxy | Parsing buys nothing — the proxy is transport-agnostic; the existing binary framing (`MsgInput`/`MsgOutput`/etc.) is end-to-end between xterm and the peer hub. Copy opaque messages. |
| Synthesize `Origin` header in proxy | Relax the peer's `requireAllowedOrigin` to allow empty Origin on the proxied path | Relaxing peer auth is a security regression and touches the peer's locked Phase 88 origin contract. Inject the correct Origin instead — it is known (`baseURL`). |
| Reuse the file-browse cap (Perms-as-is) | Add a new `terminal`/`pty` perm scope | Investigation shows the WS gate already keys on `claims.Perms ∈ {"read","read,write"}`, the SAME perms a web-share viewer's cap carries (`server.go:945`). The join-code cap minted at share time already authorizes the WS. No scope addition needed. |

**Installation:** No new packages. (Package Legitimacy Audit therefore N/A — see below.)

## Package Legitimacy Audit

No external packages are installed by this expansion. All code uses `github.com/coder/websocket@v1.8.14` (already in `go.mod`, verified present) and Go stdlib. **Disposition: N/A — no new dependencies.**

## Architecture Patterns

### System Architecture Diagram

```
                            LOCAL HOST (this machine)                                 REMOTE PEER (tailnet)
  ┌───────────────┐   ws://127.0.0.1:relayPort      ┌──────────────────────┐   wss://peer.ts.net:7443     ┌────────────────────┐
  │ Wails webview │   /api/relay/remote/{sid}/ws     │  Daemon relay surface │   /sessions/{sid}/ws?cap=T   │  Peer webserver     │
  │  RelayClient  │ ───────────────────────────────► │  (RelayHandler /      │ ───────────────────────────► │  requireAllowedOrigin│
  │  (xterm.js)   │   binary frames (MsgInput/Ping)  │   wrapRelayWithRemote)│   + Origin: https://peer...  │  → requireCapability │
  │               │ ◄─────────────────────────────── │                       │ ◄─────────────────────────── │  → handleWSSRelay    │
  └───────────────┘   binary frames (MsgOutput/snap) │  handleRemoteSessionWS│   scrollback snapshot first  │   hub.Subscribe      │
        ▲                                            │   1. cap lookup (store)│   then live PTY frames       │   manager.Get(sid)   │
        │ select URL                                 │   2. websocket.Accept  │                              └─────────┬──────────┘
        │ (local vs remote)                          │      (loopback Origin) │                                        │
  ┌─────┴─────────┐                                  │   3. websocket.Dial    │                              ┌─────────▼──────────┐
  │ HubModal /    │                                  │      (InsecureSkipVfy, │                              │  PTY (agent CLI)   │
  │ HubPanel      │  cap deposited earlier via       │       inject Origin)   │                              └────────────────────┘
  │ discriminator │  Phase 122 join-code exchange →  │   4. 2× io copy goroutines                            
  └───────────────┘  RemoteCapStore.Put(sid,base,T)  └──────────────────────┘
```

Trace the primary use case: user clicks a remote card → cap already cached (else join-code modal first) → modal mounts `RelayClient` pointed at the daemon proxy URL → proxy looks up cap, dials peer with `?cap=T` + injected `Origin` → peer verifies cap, subscribes to hub, replays scrollback (this IS the tail), streams live PTY → frames copy back to xterm. Input frames flow the reverse direction; the peer's `claims.Perms == "read"` gate drops them if the cap is read-only.

### Recommended Project Structure
```
internal/daemon/
├── relay_remote_files.go      # ADD: mux.HandleFunc("GET /api/relay/remote/{sessionID}/ws", ...)
├── remote_ws_proxy.go         # NEW: handleRemoteSessionWS + dial/copy helpers (mirrors remote_files.go)
├── remote_ws_proxy_test.go    # NEW: cap-gated, frame-copy, origin, no-cap tests (mirrors relay_remote_files_test.go)
├── remote_caps.go             # UNCHANGED: reuse Get(sid) → (baseURL, capToken)
frontend/src/
├── lib/relayClient.ts         # EDIT: accept explicit ws URL (remote-proxy) vs derive local
├── components/TerminalPanel.tsx  # EDIT: thread remoteBaseURL/proxy flag → RelayClient
├── components/Hub/HubInteractiveModal.tsx  # EDIT: pass remote routing prop
├── components/Hub/HubBriefingModal.tsx     # EDIT: remote tail via WS snapshot; CR-03 cleanup
└── lib/remoteSession.ts       # UNCHANGED: remoteBaseURLFor / findRemoteSession reused
```

### Pattern 1: Mirror the remote-files proxy placement (parent-mux fall-through)
**What:** Register the new WS route on the same parent mux that `wrapRelayWithRemoteFiles` builds, so it rides the relay loopback surface (the only one the webview can reach) and falls through to the relay server for everything else.
**When to use:** Always — this is the established, tested placement for daemon-owned routes that the webview hits.
**Example:**
```go
// internal/daemon/relay_remote_files.go — add to wrapRelayWithRemoteFiles's mux,
// alongside the 9 file routes. Source pattern: relay_remote_files.go:43-51.
mux.HandleFunc("GET /api/relay/remote/{sessionID}/ws", a.handleRemoteSessionWS)
// NOTE: no FilesCORS / FilesPreflight wrapper — a WebSocket upgrade is not a
// CORS-preflighted request; Origin is enforced at websocket.Accept (see Pattern 3).
```

### Pattern 2: Cap lookup + upstream WS dial with injected Origin (the core handler)
**What:** Accept the inbound WS (loopback Origin allowlist), look up the cap, dial the peer WS with the tailnet TLS transport and a synthesized Origin header, copy frames both ways.
**When to use:** The new `handleRemoteSessionWS` body.
**Example:**
```go
// internal/daemon/remote_ws_proxy.go (NEW). Mirrors remote_files.go cap lookup +
// newRemoteFilesHTTPClient() transport; mirrors relay/server.go Accept origin policy.
func (a *API) handleRemoteSessionWS(w http.ResponseWriter, r *http.Request) {
    sid := r.PathValue("sessionID")
    if sid == "" { http.Error(w, "missing sessionID", http.StatusBadRequest); return }
    if a.remoteCaps == nil { http.Error(w, "remote cap store not initialised", 500); return }

    baseURL, capToken, ok := a.remoteCaps.Get(sid)
    if !ok {
        // Same contract as proxyRemoteFiles: 404 → frontend re-prompts for join code.
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "no cap registered for session"})
        return
    }

    // 1. Accept the inbound (webview) WS — loopback/Wails Origin allowlist, mirrors
    //    relay.handleSession. Reuse relay.LoopbackOriginPatterns(r.Host) (export it).
    clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        OriginPatterns: relay.LoopbackOriginPatterns(r.Host), // export from relay pkg
    })
    if err != nil { return } // Accept already wrote the error
    defer clientConn.CloseNow()

    // 2. Build the upstream wss:// URL. baseURL is "https://peer.ts.net:7443" (RemoteCapStore
    //    enforces https at Put-time). Swap scheme to wss, append /sessions/{sid}/ws?cap=T.
    u, err := url.Parse(baseURL)
    if err != nil { clientConn.Close(websocket.StatusInternalError, "bad base url"); return }
    u.Scheme = "wss"
    u.Path = "/sessions/" + sid + "/ws"
    u.RawQuery = "cap=" + url.QueryEscape(capToken)

    // 3. Dial the peer. InsecureSkipVerify tailnet transport (same as remote_files.go).
    //    CRITICAL: inject Origin == baseURL or the peer's requireAllowedOrigin 403s
    //    (it rejects an EMPTY Origin — origin_mw.go:34). A Go WS dialer sends none.
    dialCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    hdr := http.Header{}
    hdr.Set("Origin", strings.TrimRight(baseURL, "/")) // must byte-match ws.BaseURL()
    upstream, _, err := websocket.Dial(dialCtx, u.String(), &websocket.DialOptions{
        HTTPClient: a.remoteFilesClient(), // reuse: 10s timeout + InsecureSkipVerify TLS
        HTTPHeader: hdr,
    })
    if err != nil {
        clientConn.Close(websocket.StatusTryAgainLater, "remote unreachable")
        return
    }
    defer upstream.CloseNow()

    // 4. Bidirectional opaque copy. Use the long-lived request context, NOT dialCtx
    //    (which carries the 10s dial deadline — see Pitfall 4).
    ctx := r.Context()
    errc := make(chan error, 2)
    go copyWS(ctx, upstream, clientConn, errc)  // peer → webview (output + scrollback)
    go copyWS(ctx, clientConn, upstream, errc)  // webview → peer (input + resize + ping)
    <-errc // first side to error/close tears down both via the defers
}

// copyWS reads whole messages from src and writes them verbatim to dst until
// either errors. Opaque pass-through — no framing knowledge required.
func copyWS(ctx context.Context, dst, src *websocket.Conn, errc chan<- error) {
    for {
        typ, data, err := src.Read(ctx)
        if err != nil { errc <- err; return }
        if err := dst.Write(ctx, typ, data); err != nil { errc <- err; return }
    }
}
```
**Source basis:** `internal/daemon/remote_files.go:38-58` (transport), `internal/relay/server.go:193-208` (Accept origin policy), `internal/webserver/origin_mw.go:31-51` (the Origin contract the proxy must satisfy), `coder/websocket@v1.8.14` `Dial`/`DialOptions.HTTPClient`+`HTTPHeader` [VERIFIED: in-repo usage + go.mod].

### Pattern 3: Why no CORS, but yes Origin — the two different boundaries
**What:** The WS proxy has TWO origin checks at TWO ends, and neither is HTTP CORS.
- **Inbound (webview → daemon relay):** `websocket.Accept` with `relay.LoopbackOriginPatterns(r.Host)` — same allowlist `relay.handleSession` already uses. WS upgrades are not CORS-preflighted, so the `FilesCORS`/`FilesPreflight` wrappers used for the file routes do NOT apply here.
- **Outbound (daemon → peer webserver):** the peer's `requireAllowedOrigin` (`origin_mw.go`) requires `Origin == ws.BaseURL()` exactly and **rejects an empty Origin** (D-05). The Go dialer sends no Origin by default, so the proxy MUST inject `Origin: <baseURL>`.
**When to use:** Both, always. Omitting the inbound allowlist weakens the loopback boundary; omitting the injected Origin makes every remote attach 403.

### Anti-Patterns to Avoid
- **Forwarding the cap as a header instead of the `?cap=` query param:** the peer reads `r.URL.Query().Get("cap")` (`capability_mw.go:39`). Use the query param.
- **Re-using `dialCtx` for the copy loop:** its 10s deadline would kill a healthy long-lived terminal after 10s. Use `r.Context()` for copies (Pitfall 4).
- **Parsing/re-framing messages in the proxy:** the binary protocol is end-to-end between xterm and the peer hub. Copy opaquely.
- **Adding a new perm scope for terminal access:** the existing cap perms already gate the WS (`server.go:945`). A new scope would diverge from the web-share viewer model and require touching the peer's locked Phase 87/88 cap contract.
- **Calling `GetSessionTailLines` for a remote session:** it does `e.manager.Get(id)` on the LOCAL engine and returns `[]string{}` for unknown ids (`engine.go:550`). Remote tail must come from the peer's scrollback snapshot replayed on WS connect.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WS framing for input/resize/ping | A new protocol for the proxy | Existing `relayClient.ts` encoders + opaque copy | Protocol is end-to-end; proxy is transport-only |
| Cap storage / lookup | A second cap cache for terminals | `RemoteCapStore` (reuse) | One cap authorizes both files and terminal |
| Cap minting / join exchange | A terminal-specific token flow | Phase 122 `ExchangeJoinCodeAtURL` + `RegisterRemoteCap` | Cap perms already cover WS; modal already runs this (MODAL-06) |
| TLS to a self-signed tailnet peer | A new http.Client | `a.remoteFilesClient()` / `newRemoteFilesHTTPClient()` | Identical InsecureSkipVerify + TLS1.2 shape, already reviewed |
| Inbound WS origin allowlist | A bespoke origin check | `relay.loopbackOriginPatterns` (export it) | Tested across Wails prod/dev + loopback (`origin_test.go`) |
| Remote tail fetch | An HTTP `/sessions/{id}/tail` endpoint | Read scrollback frames off the proxied WS | No such endpoint exists; snapshot is replayed on connect (`server.go:997`) |

**Key insight:** Every hard part of this feature already exists and is tested on one side or the other. The only genuinely new code is a ~60-line WS reverse proxy and a one-param frontend URL seam. Resist building parallel machinery.

## Runtime State Inventory

This is a code/feature addition, not a rename/migration. Inventory categories assessed:
- **Stored data:** None — `RemoteCapStore` is in-memory and already populated by the existing file-browse path; the WS proxy reads the same entries. No new persisted state.
- **Live service config:** None — no new ports, services, or peer-side config. The peer's `/sessions/{id}/ws` route already exists and is enabled by web-share toggle.
- **OS-registered state:** None.
- **Secrets/env vars:** None new. The cap token is the only secret and it already lives in `RemoteCapStore` (memory-only, never written to disk — `remote_caps.go` header invariant).
- **Build artifacts:** Standard Go rebuild + Wails frontend rebuild. Production build requires `-tags wailsassets` (per CLAUDE.md memory). No stale artifacts introduced.

**Nothing found in any category that requires migration** — verified by inspecting `remote_caps.go` (in-memory only) and confirming the peer route pre-exists.

## Common Pitfalls

### Pitfall 1: Empty Origin → peer 403
**What goes wrong:** Proxy dials the peer WS without an Origin header; peer's `requireAllowedOrigin` rejects empty Origin with 403 "forbidden" (`origin_mw.go:34-40`), so the upstream `websocket.Dial` returns an error and the modal shows a generic failure.
**Why it happens:** Go's WS dialer sends no Origin; browsers always do, so the peer assumes a browser.
**How to avoid:** Inject `Origin: <baseURL>` (byte-exact match to the peer's `ws.BaseURL()`). The `baseURL` in `RemoteCapStore` is the peer origin — use it directly.
**Warning signs:** Upstream dial returns 403; `websocket.Dial` error in proxy logs.

### Pitfall 2: Read-only cap silently swallows input
**What goes wrong:** If the share grant minted a read-only cap (`Perms == "read"`), the peer's `handleWSSRelay` sets `sub.ReadOnly = true` (`server.go:945`) and silently discards input frames (`relay/server.go:269`). The user types, nothing happens, no error.
**Why it happens:** The perms are decided at share time on the peer, not by the proxy. The interactive modal assumes write.
**How to avoid:** Surface read-only state in the modal. The cap's perms are not visible client-side, but the peer's `/api/sessions/{id}/info` endpoint returns `Perms` (`server.go:895`) — fetch it through a cap'd proxy to decide whether to show a read-only banner, OR accept that read-only sessions render but ignore input and document it. Recommend: at minimum, do not present the briefing Send button as guaranteed-delivered for read-only caps.
**Warning signs:** Live terminal renders output but ignores all keystrokes.

### Pitfall 3: Tail expectation mismatch (no HTTP tail endpoint)
**What goes wrong:** Planner assumes a remote `GetSessionTailLines` equivalent exists; there is none. Building one would require a new peer endpoint and a new cap-gated proxy route.
**Why it happens:** Local tail uses the in-process engine; remote has no symmetric HTTP surface.
**How to avoid:** For the remote briefing modal, derive the tail from the scrollback snapshot the peer replays as the first WS frame(s) on connect (`server.go:997-998`). The briefing modal opens a short-lived proxied WS, reads the `MsgOutput` snapshot frame, ANSI-strips/last-N-lines it client-side (mirror `engine.go:GetSessionTailLines` stripping), renders it, and either keeps the socket for the Send or closes after capturing the tail.
**Warning signs:** Remote briefing always shows "No recent output available."

### Pitfall 4: Dial-deadline context kills the live terminal
**What goes wrong:** Using the 10s `dialCtx` for the copy loops closes a healthy terminal after 10s.
**Why it happens:** Reusing the dial timeout context for steady-state I/O.
**How to avoid:** Dial with a bounded `dialCtx`; copy with the unbounded `r.Context()` (closed when the webview disconnects).
**Warning signs:** Remote terminal disconnects ~10s after opening regardless of activity.

### Pitfall 5 (CR-03): RelayClient/WS leak + post-abandon send race
**What goes wrong:** `HubBriefingModal.handleSend` (and any per-send client) leaks the WS on timeout, delivers untrusted input after the user abandoned, and leaks on unmount.
**Why it happens:** `client.close()` is only called inside `onOpen`; no `settled` guard; no unmount cleanup.
**How to avoid:** Adopt the REVIEW CR-03 fix verbatim — `clientRef` + `settled` flag + `clearTimeout` + `useEffect(() => () => clientRef.current?.close(), [])`. Applies identically to the remote path (the remote client is the same `RelayClient` with a different URL).
**Warning signs:** Lingering WS connections in devtools after closing the modal; "Failed to send" followed by the agent receiving the text anyway.

### Pitfall 6: `relayPort === 0` builds `ws://127.0.0.1:0/...` (IN-01)
**What goes wrong:** Modal renders with a transient `relayPort` of 0.
**How to avoid:** Guard `relayPort !== undefined && relayPort > 0` (mirror the tab grid). For the remote path, the port is still the LOCAL relay port (the proxy lives on the local relay surface) — the same guard applies.

## Code Examples

### Frontend: select local-direct vs remote-proxy WS URL
```typescript
// frontend/src/lib/relayClient.ts — change the constructor to accept an explicit
// URL builder input. Cleanest seam: an optional remoteBaseURL-derived proxy path.
// Local (unchanged):  ws://127.0.0.1:{port}/sessions/{id}/ws
// Remote (new):       ws://127.0.0.1:{port}/api/relay/remote/{id}/ws
constructor(
  port: number,
  sessionId: string,
  callbacks: RelayClientCallbacks,
  opts?: { remote?: boolean },   // ← new seam; default false preserves all existing callers
) {
  const path = opts?.remote
    ? `/api/relay/remote/${sessionId}/ws`   // daemon proxy → peer (cap looked up server-side)
    : `/sessions/${sessionId}/ws`           // local relay direct
  const url = `ws://127.0.0.1:${port}${path}`
  this.ws = new WebSocket(url)
  // ... rest unchanged
}
```
Thread `remote` from `HubPanel.handleCardClick`'s existing `isRemote` discriminator (`HubPanel.tsx:341`) → `HubModal` → `HubInteractiveModal`/`HubBriefingModal` → `TerminalPanel`/`RelayClient`. The cap is NOT passed client-side — it stays in the daemon's `RemoteCapStore`, looked up by sessionID. This keeps the token out of React state (preserves the Phase 122 T-122-03-01 invariant).

### Go: export the loopback origin helper for reuse
```go
// internal/relay/server.go — add an exported wrapper so the daemon proxy can
// reuse the exact same allowlist (do not duplicate the patterns).
func LoopbackOriginPatterns(host string) []string { return loopbackOriginPatterns(host) }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Modal attaches local relay with remote id (broken) | Daemon WS reverse proxy to peer, cap-gated | This expansion | Remote interactive terminal actually connects |
| Remote tail via `GetSessionTailLines` (returns empty) | Tail from peer scrollback snapshot over proxied WS | This expansion | Remote briefing shows real prompt |
| File-browse cap only | Same cap authorizes terminal WS (no scope change) | This expansion | One join-code exchange serves files + terminal |

**Deprecated/outdated:** Nothing deprecated. The `?readonly=` query hint on the local relay (`relay/server.go:181`) is NOT the auth mechanism for remote — the peer derives readonly from signed `claims.Perms`, ignoring any client-asserted readonly. Do not rely on `?readonly=` for the remote path.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| ~~A1~~ **RESOLVED → VERIFIED** | The join-code cap carries bare `Perms` (`"read"` for the read code, `"read,write,..."` for the write code) sufficient for the peer's WS `requireCapability` with NO scope addition | Stack / Alternatives | N/A — verified at `internal/daemon/api.go:1103-1145`: `issueCapabilitiesForSession` mints two tokens (rClaims `Perms:"read"`, wClaims `Perms:"read,write[,files.*]"`) and issues a join code for each. Whichever code the user pastes decides read vs read+write for the terminal. No `files.read`-only cap is ever issued for the WS path. |
| A2 | `websocket.Dial` in coder/websocket v1.8.14 honors `DialOptions.HTTPClient` for TLS (InsecureSkipVerify) and `HTTPHeader` for Origin | Pattern 2 | If the API differs, the dial would not use the tailnet transport. LOW risk — this is the documented coder/websocket API and the repo already uses `DialOptions.HTTPHeader` in `origin_test.go`. Confirm `HTTPClient` field name at implementation time. |
| A3 | The peer accepts a `wss://` upgrade on the same `/sessions/{id}/ws` route the browser uses (scheme is the only difference from the file proxy which uses https) | Pattern 2 | LOW — same TLS listener serves both HTTP and WS; the browser web-share path already uses wss to this exact route. |

**Note:** A1 was confirmed during research (`api.go:1103-1145`) — the "no new perm scope" decision is now VERIFIED, not assumed. A2/A3 are LOW-risk standard-library/route facts to confirm at implementation time. The Assumptions Log is otherwise effectively empty for design-blocking risks.

## Open Questions

1. **(RESOLVED) Does the share-time join code mint a cap with bare `read`/`read,write` perms?**
   - Answer: YES. `issueCapabilitiesForSession` (`internal/daemon/api.go:1103-1145`) mints a read token (`Perms:"read"`) and a write token (`Perms:"read,write"` plus optional `files.read`/`files.write`), and issues a join code for each (`joinCodes.Issue(rTok)` / `Issue(wTok)`). The WS gate (`server.go:945`) keys on bare `read`/`read,write`, which both tokens satisfy. **No new perm scope is needed.** The terminal's read-vs-write capability is determined by which join code (read or write) the user pasted — exactly like a web-share viewer.

2. **Read-only UX:** should the interactive modal detect a read-only cap and show a banner, or silently render output and drop input?
   - Recommendation: fetch `Perms` via the existing `/api/sessions/{id}/info` (cap'd) through a small proxy or include it in the meta the frontend already has, then render a non-color read-only badge (user is colorblind — use text/icon, not color). Defer full a11y to Phase 135 but do not present a Send button that silently fails.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `github.com/coder/websocket` | WS Accept + Dial | ✓ | v1.8.14 | — |
| Go toolchain | daemon build | ✓ | per go.mod | — |
| Wails (`-tags wailsassets`) | production frontend embed | ✓ | per project | dev: `wails dev` |
| Two tailnet peers (live UAT) | manual end-to-end test | ✗ at research time | — | Go httptest fixture peer (TLS) covers everything except a real second machine |

**Missing dependencies with no fallback:** A real second tailnet machine for live UAT — but this is a manual test, not a build dependency. The automated tests use an httptest TLS fixture peer (the pattern `newFixtureRemotePeer` / `newDaemonAPIWithUpstreamCert` already established in `relay_remote_files_test.go`).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `go test` (stdlib testing + `net/http/httptest`) |
| Framework (frontend) | Vitest |
| Config file | `go.mod` (Go); `frontend/vitest.config.*` |
| Quick run command | `go test ./internal/daemon/ -run RemoteSessionWS -race` |
| Full suite command | `go test ./... -tags wailsassets` then `cd frontend && pnpm test` |

### Phase Requirements → Test Map
| Req | Behavior | Test Type | Automated Command | File Exists? |
|-----|----------|-----------|-------------------|-------------|
| WS-PROXY-01 | `/api/relay/remote/{sid}/ws` mounted on relay surface (not 404) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_MountedOnRelay` | ❌ Wave 0 |
| WS-PROXY-02 | No cap deposited → handler reached, returns "no cap registered" (not bare route-miss) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_NoCap` | ❌ Wave 0 |
| WS-PROXY-03 | With cap, frames copy bidirectionally to fixture peer WS | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_FrameCopy -race` | ❌ Wave 0 |
| WS-PROXY-04 | Proxy injects `Origin: <baseURL>` on upstream dial (fixture peer asserts non-empty matching Origin) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_InjectsOrigin` | ❌ Wave 0 |
| WS-PROXY-05 | Cross-site inbound Origin rejected at Accept (mirror `origin_test.go`) | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_RejectsCrossSiteOrigin` | ❌ Wave 0 |
| WS-PROXY-06 | Copy loop uses request context, survives past the 10s dial timeout | integration (Go) | `go test ./internal/daemon -run RemoteSessionWS_LongLived` | ❌ Wave 0 |
| FE-URL-01 | RelayClient builds `/api/relay/remote/{id}/ws` when `remote` set, local path otherwise | source/unit (Vitest) | `pnpm test relayClient` | ❌ Wave 0 |
| FE-ROUTE-01 | HubInteractiveModal threads `remote` from `isRemote` discriminator | source/behavioral (Vitest) | `pnpm test HubInteractiveModal` | partial (source-string only — WR-07) |
| CR-03-01 | Briefing send: open→sendInput→close ordering; timeout cleanup; no post-abandon send | behavioral (Vitest, mock RelayClient) | `pnpm test HubBriefingModal` | ❌ (WR-07 gap) |
| TAIL-01 | Remote briefing tail rendered from WS scrollback snapshot (mock proxied WS) | behavioral (Vitest) | `pnpm test HubBriefingModal` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/daemon -run RemoteSessionWS -race` + `pnpm test relayClient HubBriefingModal`
- **Per wave merge:** `go test ./... -race` + `cd frontend && pnpm test`
- **Phase gate:** full suite green, then live two-peer UAT (manual) before `/gsd:verify-work`.

### Manual-only
- **Real two-machine tailnet interactive terminal** (type in the modal on machine A, see it execute on machine B's session; verify resize, scrollback, copy/paste). No automated substitute — requires two live peers with real Tailscale certs. Use `wails dev` web-share to a regular Chrome if DevTools needed (DevTools disabled in prod build, per project memory).

### Wave 0 Gaps
- [ ] `internal/daemon/remote_ws_proxy_test.go` — WS-PROXY-01..06 (model on `relay_remote_files_test.go`: `newFixtureRemotePeer` + `newDaemonAPIWithUpstreamCert`, but the fixture peer must serve a WS `/sessions/{id}/ws` that asserts `?cap=` and a non-empty Origin, then echoes frames).
- [ ] Fixture peer extension: a cap-guarded WS echo endpoint (new helper, e.g. `newFixtureRemotePeerWithWS`).
- [ ] `frontend/.../relayClient.test.ts` — assert URL construction for both modes.
- [ ] `frontend/.../HubBriefingModal.test.tsx` — behavioral mock-RelayClient tests (closes WR-07 for the briefing path).
- [ ] Export `relay.LoopbackOriginPatterns` (or add a `relay`-package test confirming the daemon proxy reuses it).

## Security Domain

`security_enforcement` not explicitly false in config → included.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | HMAC-signed capability token (`?cap=`), verified on the PEER (`requireCapability`); proxy never trusts the token, only forwards it |
| V3 Session Management | yes | Cap is bound to SID + GrantID; peer rejects on revoked grant / web-disabled session; cap lives in memory-only `RemoteCapStore` |
| V4 Access Control | yes | Read vs write derived from signed `claims.Perms` on the peer; client cannot escalate via query string |
| V5 Input Validation | yes | Untrusted PTY input bounded (`maxLength={4096}` briefing); WS frames opaque-copied (no eval); xterm renders as text, no `dangerouslySetInnerHTML` |
| V6 Cryptography | yes | TLS to peer (self-signed tailnet cert; `InsecureSkipVerify` justified by Tailscale transport authenticity — same as Phase 122). HMAC verify uses `internal/capability` — never hand-rolled |
| V9 Communication | yes | `wss://` to peer; inbound is loopback (`127.0.0.1`); Origin enforced both ends |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-site WS hijack (inbound) | Spoofing | `websocket.Accept` loopback/Wails Origin allowlist (reuse `loopbackOriginPatterns`); mirror `origin_test.go` |
| Origin-bypass on peer (empty Origin) | Spoofing | Proxy injects correct `Origin: <baseURL>`; peer rejects empty/mismatched (`origin_mw.go`) |
| Cap token disclosure in logs/errors | Information Disclosure | Never log the cap; redact like `redactCapTokenFromError` (`remote_files.go:322`); token stays server-side, never in WS URL visible to the webview (the webview hits `/api/relay/remote/{id}/ws` with NO cap — the daemon adds it) |
| MITM forging cap via InsecureSkipVerify | Tampering | Tailscale transport authenticity (WireGuard) substitutes for cert verification; documented caveat carried from `client_remote_files.go:39` |
| Privilege escalation read→write | Elevation | `claims.Perms` is signed; peer ignores client-asserted `?readonly=`; read-only caps drop input at the hub |
| Untrusted PTY input late-delivery (CR-03) | Tampering | `settled` guard + cleanup so abandoned text is never written post-timeout |
| Resource leak (WS not closed) | Denial of Service | `defer CloseNow()` both conns; `<-errc` single-tear-down; frontend `clientRef`+unmount cleanup |

### CR-03 fix (fold into this expansion — REQUIRED)
The CR-03 RelayClient/WS leak, settled-guard race, and unmount cleanup MUST be fixed here because the remote path uses the same `RelayClient` and the same per-send pattern. Apply the REVIEW's exact fix to `HubBriefingModal` and verify with the new behavioral test.

### Other 134-REVIEW warnings — fix recommendation
| ID | Issue | Recommendation for this expansion |
|----|-------|-----------------------------------|
| WR-01 | `pendingModalSessionId` stranded on join-modal cancel | **Fix** — directly in scope; the remote path makes the join modal central. Add `handleCapCancelled` reset. |
| WR-02 | `onRequestRemoteCap` overwrites in-flight `joinModalForSession` (file-browse intent) | **Fix** — the remote terminal path shares the single `joinModalForSession` slot with file-browse; collision is now more likely. Guard `if (joinModalForSession) return` or key by session id. |
| WR-03 | `({} as ITheme)` unsafe cast | **Fix (cheap)** — make `terminalTheme` required on `HubPanelProps`, drop the cast. Touches the same render block. |
| WR-04 | Hardcoded `fontSize={14}` ignores per-session size/zoom | **Fix** — thread real `fontSize`/`onFontSizeChange` from App (the modal already plumbs `relayPort`/`theme`/`pluginConfig` the same way). Low cost, high UX value for the now-functional terminal. At minimum replace `14` with `DEFAULT_FONT_SIZE`. |
| WR-05 | Broad `stopImmediatePropagation` on document Escape | **Defer/document** — not in the remote-proxy critical path. Document the suppression; full fix can wait for Phase 135 a11y unless trivially scoped to the dialog element. |
| WR-06 | No focus trap (Tab escapes modal) | **Defer to Phase 135 (a11y)** — CONTEXT marks full a11y as Phase 135. Note it; do not block this expansion. The colorblind/non-color-cue constraint IS release-blocking and applies to any read-only badge added here. |
| WR-07 | Source-string tests, no behavioral coverage | **Fix for the touched components** — the new Validation Architecture mandates behavioral tests for the remote WS routing, briefing send/timeout, and tail rendering. This directly closes WR-07 for the briefing + interactive modals. |

## Sources

### Primary (HIGH confidence — in-repo, verified this session)
- `internal/daemon/relay_remote_files.go` — parent-mux pattern, route placement, why daemon-owned routes live here
- `internal/daemon/remote_files.go` — cap lookup, InsecureSkipVerify transport, cap-redaction, MagicDNS handling
- `internal/daemon/remote_caps.go` — `RemoteCapStore.Get/Put`, https-only, memory-only invariant
- `internal/daemon/client_remote_files.go` — join-code exchange, InsecureSkipVerify caveat, Location/cap parsing
- `internal/daemon/api.go:241-256` — `RelayHandler()` wiring
- `internal/daemon/engine.go:548-560` — `GetSessionTailLines` (local-only; empty for unknown id)
- `internal/relay/server.go` — `handleSession`, `loopbackOriginPatterns`, Accept origin policy, framing
- `internal/relay/origin_test.go` — Origin acceptance/rejection test patterns to mirror
- `internal/webserver/server.go:604-611, 929-1010` — peer `/sessions/{id}/ws` route + `handleWSSRelay` + scrollback snapshot
- `internal/webserver/capability_mw.go` — `requireCapability`, perms model (`read`/`read,write` vs `files.read/write`)
- `internal/webserver/origin_mw.go` — strict Origin == BaseURL, empty-Origin rejection (the contract the proxy must satisfy)
- `internal/capability/capability.go` / `joincode.go` — Claims.Perms, `Verify`, `Exchange`
- `frontend/src/lib/relayClient.ts` — WS URL construction (the seam to extend)
- `frontend/src/lib/remoteSession.ts` — `remoteBaseURLFor`, `findRemoteSession`
- `frontend/src/App.tsx:1057-1135` — `handleModalExchange`, `RegisterRemoteCap`, MODAL-06 intent discriminator
- `frontend/src/components/Hub/{HubInteractiveModal,HubBriefingModal,HubPanel}.tsx` — consumers to rewire
- `.planning/phases/134-modal-interaction/134-REVIEW.md` — CR-01/02/03 + WR-01..07 (authoritative)
- `.planning/phases/134-modal-interaction/134-CONTEXT.md` — locked decisions
- `go.mod` — `coder/websocket v1.8.14`

### Secondary (MEDIUM)
- `internal/daemon/relay_remote_files_test.go` — fixture-peer + relay-surface test harness to clone for WS

### Tertiary (LOW)
- None — no external/training-data claims load-bearing in the core design.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new deps; all components verified in-repo
- Architecture (proxy placement + dial + Origin injection): HIGH — every piece traced to source on both ends
- Cap-perms-cover-WS (no new scope): MEDIUM — gate logic verified (`server.go:945`); the share-time mint perms (A1) not yet located — confirm during planning
- Tail-via-snapshot: HIGH — verified no HTTP tail endpoint; snapshot replay confirmed (`server.go:997`)
- CR-03 / warnings: HIGH — directly from the authoritative REVIEW

**Research date:** 2026-06-17
**Valid until:** 2026-07-17 (stable in-tree code; re-verify A1 mint-perms before locking)
</content>
</invoke>
