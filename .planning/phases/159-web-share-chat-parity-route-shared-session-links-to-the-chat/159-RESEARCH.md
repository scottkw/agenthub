# Phase 159: Web-Share Chat Parity — Research

**Researched:** 2026-06-27
**Domain:** Go HTTP routing / server-side redirect; React SPA URL-param wiring
**Confidence:** HIGH

---

## Summary

The real PARITY-01 gap is not in the React SPA — it's in the share-URL shape. The daemon mints
URLs in the form `https://{host}/sessions/{id}?cap=TOKEN`; every remote guest who clicks a shared
link lands on `handleTerminalPage`, which serves the vanilla-JS `terminal.html` viewer. That viewer
defines only frame types 0x01/0x02/0x10/0x11/0x12 — it has no handler for chat frames 0x30–0x36.
Chat frames ARE already relayed to web guests by `handleWSSRelay` / `BroadcastChat`; they are
simply silently dropped by the terminal.js receive loop.

The fix is a single-function change in `internal/webserver/server.go`: replace the 8-line
`handleTerminalPage` body (read + serve `terminal.html`) with an HTTP 302 redirect to
`/app/?session={sessionID}&cap={token}`. The React SPA at `/app/` is already fully wired:
`readWebModeParams` reads `?session=` and `?cap=`; `WebShareSessionView` mounts ChatPanel,
TerminalPanel, the chat toggle, and the unread badge — all the chat infrastructure Phase 155
built but that no remote guest was ever sent to.

The relay already relays chat. The SPA already has chat. The only missing piece is routing remote
guests to the SPA instead of the raw viewer.

**Primary recommendation:** Modify `handleTerminalPage` to issue an HTTP 302 to
`/app/?session={url.QueryEscape(sessionID)}&cap={url.QueryEscape(token)}`. This is the entire
server change. No frontend changes are needed. Update `TestWebServerToggle` (currently expects
200, must become 302) and add a dedicated redirect-mechanics test.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Routing remote guests to chat-capable surface | Backend (webserver) | — | The redirect is a server-side routing decision; the SPA is already complete |
| Chat UI (ChatPanel, toggle, badge) | Frontend SPA (/app/) | — | Already implemented in Phase 155; no changes needed |
| Chat frame relay | Backend (relay hub) | — | BroadcastChat already fans out to web subscribers; no changes needed |
| Cap validation before redirect | Backend (capability middleware) | — | requireCapability wraps handleTerminalPage; validation happens before redirect |
| Query-param parsing (session, cap) | Frontend SPA (webMode.ts) | — | readWebModeParams already handles ?session= and ?cap= |

---

## Standard Stack

No new libraries. This phase uses only the Go standard library and the already-deployed SPA.

### Core (existing — no changes)
| File | Purpose | Notes |
|------|---------|-------|
| `internal/webserver/server.go` | HTTP routing + handlers | `handleTerminalPage` is the change target |
| `internal/webserver/capability_mw.go` | `requireCapability` middleware | No change; runs before handleTerminalPage |
| `frontend/src/lib/webMode.ts` | `readWebModeParams` / `detectMode` | Already reads `?session=` and `?cap=` — no change |
| `frontend/src/components/Hub/WebShareSessionView.tsx` | Chat-capable web surface | Already complete — no change |

### Go packages in scope
| Package | Use | Version |
|---------|-----|---------|
| `net/http` | `http.Redirect`, `http.StatusFound` | stdlib |
| `net/url` | `url.QueryEscape` for cap token encoding | stdlib |
| `fmt` | target URL construction | stdlib |

**Installation:** No new packages.

---

## Package Legitimacy Audit

No external packages are introduced in this phase.

| Package | Registry | Verdict | Disposition |
|---------|----------|---------|-------------|
| (none) | — | — | — |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious SUS:** none

---

## Chesterton's Fence Analysis (Research Question 1)

**Why does `/sessions/{id}` exist as a separate route from `/app/`?**

The raw viewer (`terminal.html` + `terminal.js`) was the original and only browser surface before
Phase 120 added the React SPA at `/app/`. The raw viewer is a compact, dependency-free xterm.js
page that handles terminal output, resize, and all the addon plugins (search, web-links, image,
serialize, progress). It was designed before chat existed and has never had chat support.

**[VERIFIED: codebase grep]** `web/assets/terminal.js:1-12` defines only:
```js
const MsgOutput  = 0x01;
const MsgResize  = 0x02;
const MsgInput   = 0x10;
const MsgResize2 = 0x11;
const MsgPing    = 0x12;
```
No 0x30–0x36 chat frames. The `switch(msgType)` on incoming frames silently drops them.

**Every caller/linker of the `/sessions/{id}` route:**

| Caller | Location | URL shape | After Phase 159 |
|--------|----------|-----------|----------------|
| `issueCapabilitiesForSession` | `internal/daemon/api.go:1288-1289` | `base + "/sessions/" + id + "?cap=" + token` | share modal URLs redirect to `/app/` |
| `handleJoinExchange` | `internal/webserver/server.go:854` | 303 to `/sessions/{id}?cap=token` | join-code flow double-redirects (303 → 302 → `/app/`) |
| `handleExchangeJoinCode` | `internal/daemon/api.go:1385` | returns URL string to frontend | frontend calls BrowserOpenURL, browser follows redirect |
| `handleSessionQR` | `internal/webserver/server.go:982` | `/sessions/{id}` WITHOUT `?cap=` | unchanged — no cap → 401 from requireCapability |
| `handleSessionsMeta` | `internal/webserver/server.go:911` | `/sessions/{id}` WITHOUT `?cap=` in response metadata | unchanged — no cap |

**[VERIFIED: grep /internal/daemon/api.go, /internal/webserver/server.go]**

**Verdict: Redirect unconditionally.** The `requireCapability` middleware (which wraps
`handleTerminalPage`) already gates on `?cap=` being present and valid. If there is no cap,
`requireCapability` returns 401 before `handleTerminalPage` even runs. The redirect only fires
for valid, authorized requests — which are all remote-browser share-flow hits. There is no
legitimate non-cap consumer of the HTML page that would be affected.

The QR code (`/api/sessions/{id}/qr`) and sessions-meta (`/api/sessions/meta`) produce URLs
without caps. These routes already 401 at `requireCapability`; Phase 159 does not change that.

**The raw viewer files (`terminal.html`, `terminal.js`) are NOT removed.** The `/sessions/{id}`
HTML route is redirected; the `/sessions/{id}/ws` WebSocket endpoint is untouched. Terminal.html
is still embedded in the binary. A future milestone could remove it, but Phase 159 must not.

---

## Redirect Mechanics (Research Question 2)

**[VERIFIED: codebase read — server.go:657-661, 962-971; capability_mw.go:37-74]**

### Current `handleTerminalPage` (lines 962-971)

```go
// handleTerminalPage serves the embedded terminal.html.
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
    data, err := webfs.WebFS.ReadFile("terminal.html")
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write(data) //nolint:errcheck
}
```

**Registration chain (in order, outermost → innermost):**
```go
mux.HandleFunc("GET /sessions/{id}",
    ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage)))
```

`requireCapability` runs before `handleTerminalPage` and enforces:
1. `?cap=` present
2. HMAC-SHA256 signature valid against current signing key
3. `claims.SID == path {id}` (SEC-03)
4. `claims.GrantID` is active in the grant list (SEC-04 / D-15)
5. Session is still web-enabled (defense-in-depth)
6. On success: attaches `capability.Claims` to `r.Context()`

**The redirect must happen inside `handleTerminalPage`, not before.** By the time the redirect
fires, all five validation steps above have passed. The `sessionID` and `token` are available
without re-verification.

### Phase 159 implementation

```go
// handleTerminalPage redirects the shared /sessions/{id}?cap=TOKEN URL to the
// chat-capable /app/ React SPA. The capability has already been validated by
// requireCapability; sessionID and token are safe to forward. URL-encoding is
// required for the cap token because JWTs contain base64 characters ('+', '/', '=').
// Phase 159 / WEBCHAT-01: remote guests land on the chat-capable surface.
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    token := r.URL.Query().Get("cap")
    target := fmt.Sprintf("/app/?session=%s&cap=%s",
        url.QueryEscape(sessionID),
        url.QueryEscape(token))
    http.Redirect(w, r, target, http.StatusFound) // 302
}
```

**Imports needed in server.go:** `net/url` (add to import block; `fmt` is already present).

**Why 302 (Found), not 307 (Temporary Redirect):** Both are GET→GET; 302 is the standard
choice for a "go here instead" server-side routing redirect. 302 is not cached by browsers by
default (unlike 301). The cap token is ephemeral and must not be permanently cached.

**URL encoding:** Cap tokens are HMAC-SHA256 JWTs encoded with standard base64, which can
contain `+`, `/`, and `=`. These MUST be percent-encoded in the query string. `url.QueryEscape`
handles this correctly. Session IDs are UUIDs (alphanumeric + hyphens) and are safe without
encoding, but `url.QueryEscape` is used for defense-in-depth.

**The SPA reads `?session=` and `?cap=` exactly as constructed:**

```typescript
// frontend/src/lib/webMode.ts:65-73
export function readWebModeParams(loc: Location = window.location): WebModeParams {
  const params = new URLSearchParams(loc.search)
  const session = (params.get('session') ?? '').trim()
  const cap = (params.get('cap') ?? '').trim()
  return {
    sessionId: session === '' ? null : session,
    capToken: cap === '' ? null : cap,
  }
}
```

`URLSearchParams` percent-decodes once on parse — no double-decoding issue. `detectMode`
returns `'web'` for any path starting with `/app/`, so `mode === 'web'` triggers in App.tsx.
`WebShareSessionView` then receives `webParams.sessionId` and `webParams.capToken`.

**[VERIFIED: codebase read — webMode.ts:51-73, App.tsx:1516-1531]**

---

## Cap Type: RO vs RW (Research Question 3)

**[VERIFIED: codebase read — server.go:1015-1016, capability_mw.go:37-74, relay/hub.go:559-569]**

Both read-only and read-write caps flow through identically:
- `requireCapability` validates both the same way (HMAC verify, grant check, web-enabled check)
- `handleTerminalPage` (after Phase 159) redirects both identically — just passes the token through
- On the WebSocket side, `handleWSSRelay` determines `readonly` from the signed JWT claims:

```go
claims, _ := capability.ClaimsFromContext(r.Context())
readonly := !capability.HasPerm(claims.Perms, "write")
```

RO caps have `claims.Perms = "read"` (or `"read,files.read"`). The server-enforced gate is in
`hub.HandleChatSend` (SEC-01): RO subscriber returns `ErrChatReadOnly`. The UI reflects this
(Send button disabled for RO), but the server enforces it regardless.

**D-06 confirmed: RO caps are full chat participants** — `MsgTyping` (0x33) and `MsgAliasSet`
(0x34) are NOT gated on `sub.ReadOnly` in `handleWSSRelay:1178-1202`. Only `MsgInput` (keyboard
input to PTY) and `MsgSessionInject` (inject into PTY) are read-only-gated.

```go
case relay.MsgTyping:
    // NOT gated on sub.ReadOnly — D-06: RO clients are full chat participants;
    // only MsgInput remains ReadOnly-gated.
```

---

## Frame Relay (Research Question 4)

**[VERIFIED: codebase read — relay/hub.go:556-570, server.go:1203-1243]**

`BroadcastChat` in `relay/hub.go:559`:
```go
func (h *Hub) BroadcastChat(frame []byte) {
    h.mu.Lock()
    defer h.mu.Unlock()
    for sub := range h.subscribers {
        select {
        case sub.Msgs <- frame:
        default:
            go sub.CloseSlow()
        }
    }
}
```

This fans out to ALL subscribers — including web guest subscribers registered via
`handleWSSRelay`. The write pump in `handleWSSRelay` (lines 1248-1262) simply forwards
`sub.Msgs` to the WebSocket connection:

```go
for {
    select {
    case frame := <-sub.Msgs:
        if err := conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
            return
        }
    ...
    }
}
```

**Confirmed: no server relay changes are needed for Phase 159.** Chat frames (0x30 MsgChat,
0x32 MsgPresence, 0x33 MsgTyping, 0x36 MsgInjectError) already reach the browser WebSocket.
The problem is entirely that `terminal.js` had no handler for them. The SPA's RelayClient
(`relayClient.ts`) already handles all these frames correctly, as proven by Phase 155 Playwright
tests.

---

## Verification Surface (Research Question 5)

**[VERIFIED: codebase read; cross-referenced with MEMORY.md notes on web-share UAT]**

WEBCHAT-02 requires parity be verified on the ACTUALLY-SHARED link, not by navigating to
`/app/` directly. The verification sequence:

1. **Produce a real share URL from a live daemon:**
   ```
   POST http://localhost:{daemon-port}/sessions/{id}/web-serve  (toggle on)
   POST http://localhost:{daemon-port}/sessions/{id}/capabilities  (issue caps)
   → returns: { "readURL": "https://host/sessions/{id}?cap=RO_TOKEN",
                 "writeURL": "https://host/sessions/{id}?cap=RW_TOKEN" }
   ```
   (Daemon socket path on macOS: `~/Library/Application Support/agenthub/daemon.sock`)

2. **Verify the redirect response itself (Go test):**
   ```go
   resp, err := client.Get(baseURL + "/sessions/sess1?cap=" + token)
   // assert resp.StatusCode == 302
   // assert resp.Header.Get("Location") matches /app/?session=sess1&cap=...
   ```
   The HTTP test client must NOT follow redirects for this check. Use:
   ```go
   client.CheckRedirect = func(*http.Request, []*http.Request) error {
       return http.ErrUseLastResponse
   }
   ```

3. **Verify the SPA loads with chat at the redirected URL (Playwright / dev-browser):**
   - Navigate to the share URL (the one minted by the daemon)
   - Assert browser follows redirect to `/app/?session=...&cap=...`
   - Assert `.hub-modal__chat-toggle` button is present in DOM
   - Assert `.hub-modal__body--interactive` is present (WebShareSessionView renders)

4. **UAT limitations (from project memory):**
   - Web-share WS blocks automated input — drive chat send/receive via dev-browser using
     the page's UI, not raw WebSocket frames
   - The Playwright chat-parity fixture uses a stub PTY; Phase 159 UAT needs a live daemon
     session so the redirect path is exercised end-to-end
   - UAT should be classified as a Category P manual item (like M-27/M-28/M-29/M-30)

---

## Risks / Landmines (Research Question 6)

**[VERIFIED: codebase read]**

### Risk 1: `TestWebServerToggle` must be updated (BLOCKER for CI)

`internal/webserver/server_test.go:324-354` currently expects `http.StatusOK` (200) for a
valid cap on `/sessions/sess1`. After Phase 159, `handleTerminalPage` returns 302. The test
uses Go's default `http.Client` which follows redirects, so it will follow the 302 to
`/app/?session=...&cap=...` and get 503 (because `ws.staticAppFS` is nil in the test setup).
**This test WILL fail if not updated.**

Fix: disable redirect following in the test client and assert 302 + Location header instead
of asserting 200.

### Risk 2: Cap token must be URL-encoded

Cap tokens are HMAC-SHA256 JWTs serialized as `header.payload.signature` where header and
payload are base64url-encoded. Standard base64url uses `+`, `/`, `=` (padding). These MUST be
percent-encoded in the redirect Location query string or the SPA will receive a malformed token.
`url.QueryEscape` handles this correctly.

### Risk 3: Double redirect via join-code exchange

The join-code exchange flow (`handleJoinExchange` in server.go:854) already 303-redirects to
`/sessions/{id}?cap=TOKEN`. After Phase 159, the browser sees two redirects:
1. POST `/join/exchange` → 303 → `GET /sessions/{id}?cap=TOKEN`
2. GET `/sessions/{id}?cap=TOKEN` → 302 → `GET /app/?session={id}&cap={token}`

Browsers follow this transparently. This is acceptable behavior. No change needed to
`handleJoinExchange`.

### Risk 4: `cspHeaders` middleware still runs, but does nothing harmful

The route is registered as:
```go
ws.cspHeaders(ws.requireCapability(ws.handleTerminalPage))
```
The `cspHeaders` middleware sets CSP headers on the response. A redirect response (302) WITH
CSP headers is unusual but harmless — the browser follows the redirect and ignores the headers
from the 302 response (they apply to the page being served, not to the redirect). No change
to the route registration is needed.

### Risk 5: `/app/` route lacks an explicit CSP header

The `/app/` route is served by a `GET /app/` handler without `cspHeaders`. This was true
before Phase 159 (the SPA has always served from `/app/` without an explicit CSP). Phase 159
doesn't change this. Do not add CSP headers to `/app/` in this phase — that's a separate
concern outside phase scope.

### Risk 6: Funnel forward-compat

The ROADMAP notes Phase 159 unblocks the Tailscale Funnel milestone. The redirect approach
(server-side 302) is fully compatible with Funnel since Funnel acts as a transparent HTTPS
reverse proxy. The cap token is preserved in the redirect Location and relayed to the browser.
No Funnel-specific handling is needed.

---

## Architecture Patterns

### System Architecture Diagram

```
Remote browser hits share URL
         |
         v
GET /sessions/{id}?cap=TOKEN
         |
         v
  cspHeaders middleware
         |
         v
  requireCapability middleware
    - validates HMAC signature
    - checks claims.SID == {id}
    - checks grant is active
    - checks session web-enabled
    - attaches Claims to context
         |
         v (on success only)
  handleTerminalPage  [CHANGED in Phase 159]
    - reads sessionID from r.PathValue("id")
    - reads token from r.URL.Query().Get("cap")
    - issues HTTP 302 to /app/?session={id}&cap={token}
         |
         v
  Browser follows 302
         |
         v
GET /app/?session={id}&cap={token}
         |
         v
  /app/ SPA handler (no cap gate — static bundle)
    - serves index.html (no-store)
         |
         v
  React SPA boots
    - detectMode() → 'web'
    - readWebModeParams() → {sessionId, capToken}
    - opens WebShareSessionView tab
         |
         v
  WebShareSessionView mounts
    - TerminalPanel (wsURL = wss://.../sessions/{id}/ws?cap={token})
    - ChatPanel (same wsURL, apiBaseURL, capToken)
    - chat toggle button
    - unread badge (ChatBadge)
         |
         v
  Both panels connect via GET /sessions/{id}/ws?cap=TOKEN
    - requireAllowedOrigin (passes — same origin)
    - requireCapability (passes — same token)
    - handleWSSRelay (relays all frames including 0x30-0x36 chat)
```

### Recommended Project Structure
No new files required beyond:
- Modified: `internal/webserver/server.go` (handleTerminalPage — redirect body)
- Modified: `internal/webserver/server_test.go` (TestWebServerToggle — expect 302 not 200)
- New: test function in `server_test.go` for redirect mechanics (or new file `server_redirect_test.go`)
- Modified: `TESTING.md` (add WEBCHAT-01/02/PARITY-01 rows, manual UAT item)

### Pattern: Server-Side SPA Redirect with Cap Passthrough

```go
// Source: Phase 159 implementation
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    token := r.URL.Query().Get("cap")
    target := fmt.Sprintf("/app/?session=%s&cap=%s",
        url.QueryEscape(sessionID),
        url.QueryEscape(token))
    http.Redirect(w, r, target, http.StatusFound) // 302
}
```

**Key invariant:** This handler ONLY runs after `requireCapability` has validated the token.
Never redirect before validation — that would send guests to the SPA with an invalid cap,
which the SPA cannot distinguish from a valid one at the HTML-load phase.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Chat UI for web guests | Rebuild ChatPanel in vanilla JS inside terminal.html | Redirect to `/app/` which already has ChatPanel | The SPA is complete; terminal.js lacks 11 chat dependencies |
| URL encoding of cap token | String concatenation `"?cap=" + token` | `url.QueryEscape(token)` | Base64 chars (+, /, =) break query parsing without encoding |
| Cap validation before redirect | Re-verify the JWT inside handleTerminalPage | Let requireCapability middleware handle it | The middleware already validated and attached claims |

---

## Common Pitfalls

### Pitfall 1: Redirecting Before Cap Validation
**What goes wrong:** If `handleTerminalPage` is changed to redirect without `requireCapability` wrapping it, the redirect happens for ANY request to `/sessions/{id}` — no cap, revoked cap, wrong session cap. The SPA would load and attempt to connect to the WebSocket, which WOULD correctly fail (requireCapability on `/sessions/{id}/ws` is unchanged), but it's a worse user experience and leaks that a session exists at that ID.
**How to avoid:** Do NOT change the route registration. The 302 must happen inside `handleTerminalPage`, which runs AFTER `requireCapability`.
**Warning signs:** A test that hits `/sessions/{id}` without a cap and gets 302 instead of 401.

### Pitfall 2: Forgetting to URL-Encode the Cap Token
**What goes wrong:** JWT cap tokens contain base64 characters including `=` (padding), `+`, and `/`. A redirect Location of `/app/?session=abc&cap=abc+def=` will be misinterpreted — `+` becomes a space, `=` may terminate the param. The SPA receives a malformed token, the WebSocket handshake fails with 401, no terminal or chat appears.
**How to avoid:** Always use `url.QueryEscape(token)` (or `url.Values.Encode()`).
**Warning signs:** UAT where the terminal is blank and the browser console shows a 401 on the WebSocket connection.

### Pitfall 3: Not Updating `TestWebServerToggle`
**What goes wrong:** `TestWebServerToggle` in `server_test.go:336` asserts `resp.StatusCode != http.StatusOK` fails if not 200. Go's default `http.Client` follows redirects. After Phase 159, the client follows the 302 to `/app/?...` and gets 503 (staticAppFS nil in test). CI fails.
**How to avoid:** Update the test to use a no-redirect client and assert 302 + Location header.
**Warning signs:** CI failure in `internal/webserver` package tests.

### Pitfall 4: Verifying WEBCHAT-02 on `/app/` Directly
**What goes wrong:** WEBCHAT-02 requires parity on the ACTUALLY-SHARED link (`/sessions/{id}?cap=`). Navigating directly to `/app/?session=...&cap=...` in UAT bypasses the redirect and does not prove the shared link works.
**How to avoid:** Always start verification from the URL returned by `handleIssueCapabilities` (`/sessions/{id}?cap=TOKEN`). Confirm the redirect happens before confirming chat works.
**Warning signs:** UAT that opens `/app/` directly and declares WEBCHAT-02 done — the same false-parity pattern that Phase 155 made with `/app/` bypassing the redirect path.

### Pitfall 5: Double-Redirect Through Join Code Exchange
**What goes wrong:** The join-code exchange flow (`handleJoinExchange`) 303-redirects to `/sessions/{id}?cap=TOKEN`, then Phase 159 adds a 302 redirect from there to `/app/`. If a test client is not configured to follow two redirects (or is configured to follow zero), it will not reach the SPA.
**How to avoid:** Understand that join-code UAT now involves two hops. Browser clients handle this transparently. Test clients need `CheckRedirect` set appropriately.

---

## Code Examples

### Current `handleTerminalPage` (to be replaced)
```go
// Source: internal/webserver/server.go:962-971 [VERIFIED]
func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
    data, err := webfs.WebFS.ReadFile("terminal.html")
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write(data) //nolint:errcheck
}
```

### Phase 159 `handleTerminalPage` (redirect implementation)
```go
// Source: Phase 159 pattern
import "net/url"  // add to existing imports

func (ws *WebServer) handleTerminalPage(w http.ResponseWriter, r *http.Request) {
    sessionID := r.PathValue("id")
    token := r.URL.Query().Get("cap")
    target := fmt.Sprintf("/app/?session=%s&cap=%s",
        url.QueryEscape(sessionID),
        url.QueryEscape(token))
    http.Redirect(w, r, target, http.StatusFound) // 302 / WEBCHAT-01
}
```

### `TestWebServerToggle` update pattern
```go
// Source: internal/webserver/server_test.go:324-354 — must be updated [VERIFIED]
// Current: asserts 200 OK
// Phase 159: assert 302 + Location header

// Use a no-redirect client:
noRedirectClient := &http.Client{
    CheckRedirect: func(*http.Request, []*http.Request) error {
        return http.ErrUseLastResponse
    },
}
resp, err := noRedirectClient.Get(baseURL + "/sessions/sess1?cap=" + token)
// assert resp.StatusCode == http.StatusFound (302)
// assert resp.Header.Get("Location") contains "/app/?session=sess1&cap="
```

### `readWebModeParams` in the SPA (no change required)
```typescript
// Source: frontend/src/lib/webMode.ts:65-73 [VERIFIED]
export function readWebModeParams(loc: Location = window.location): WebModeParams {
  const params = new URLSearchParams(loc.search)
  const session = (params.get('session') ?? '').trim()
  const cap = (params.get('cap') ?? '').trim()
  return {
    sessionId: session === '' ? null : session,
    capToken: cap === '' ? null : cap,
  }
}
```

### Share URL construction (unchanged — these will automatically route to SPA)
```go
// Source: internal/daemon/api.go:1287-1289 [VERIFIED]
base := ws.BaseURL()
readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok
```

---

## State of the Art

| Old Approach | Current Approach (Phase 159) | Rationale |
|--------------|------------------------------|-----------|
| `handleTerminalPage` serves `terminal.html` (vanilla JS, no chat) | `handleTerminalPage` redirects to `/app/?session=...&cap=...` (React SPA with chat) | SPA already has full chat; redirect is the minimal safe change |
| Remote guests hit the share URL and get chat-less viewer | Remote guests hit share URL, get 302, land on chat-capable SPA | Closes the real PARITY-01 gap |

**Deprecated/outdated (not to be removed in this phase):**
- `terminal.html` / `terminal.js` still exist and are still embedded; only the `/sessions/{id}` HTML route is redirected away from them. The WebSocket at `/sessions/{id}/ws` is unchanged.

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WEBCHAT-01 | Remote guest reaches a chat-capable surface via redirect | `handleTerminalPage` redirect to `/app/?session=...&cap=...`; `requireCapability` validates before redirect; SPA reads `?session=`/`?cap=` via `readWebModeParams` |
| WEBCHAT-02 | Cross-surface parity verified on the ACTUALLY-SHARED link (`/sessions/{id}?cap=`), not `/app/` directly | Verification must start from the daemon-issued readURL/writeURL; redirect response (302) must be asserted before confirming chat works |
| PARITY-01 (upstream) | Every Session Chat feature behaves identically on desktop GUI and web-share browser surface | The redirect routes guests to `WebShareSessionView` which already carries ChatPanel, toggle, unread badge, and presence — Phase 155's implementation applies to the real share surface for the first time |
</phase_requirements>

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + vitest + Playwright |
| Config file | `frontend/vitest.config.ts`, `frontend/playwright.config.ts` |
| Quick run command | `cd /Users/ken/dev/agenthub && go test ./internal/webserver/...` |
| Full suite command | `cd /Users/ken/dev/agenthub && go test ./... && cd frontend && pnpm exec vitest run && pnpm exec playwright test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WEBCHAT-01 | `GET /sessions/{id}?cap=TOKEN` returns 302 with Location `/app/?session=...&cap=...` | unit | `go test ./internal/webserver/ -run TestTerminalPageRedirect` | ❌ Wave 0 — new test |
| WEBCHAT-01 | `TestWebServerToggle` updated to expect 302 not 200 | unit | `go test ./internal/webserver/ -run TestWebServerToggle` | ✅ exists (needs update) |
| WEBCHAT-01 | Cap token URL-encoded in Location header (base64 chars survive round-trip) | unit | `go test ./internal/webserver/ -run TestTerminalPageRedirect` | ❌ Wave 0 — part of new test |
| WEBCHAT-01 | After redirect, SPA boots with chat at `/app/?session=...&cap=...` | e2e/UAT | Manual (M-31) or dev-browser | ❌ Manual UAT |
| WEBCHAT-02 | Parity verified starting from the share-flow URL, not `/app/` directly | UAT | Manual (M-31) | ❌ Manual UAT |
| WEBCHAT-02 | Join-code exchange flow: 303→302→`/app/` double redirect works in browser | integration | Part of M-31 | ❌ Manual UAT |
| PARITY-01 | ChatPanel, toggle, badge, presence all function for remote web guests on the shared URL | UAT | Manual (M-31) | ❌ Manual UAT |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/ -run TestTerminalPageRedirect -run TestWebServerToggle`
- **Per wave merge:** `go test ./internal/webserver/... && cd frontend && pnpm exec vitest run`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/webserver/server_test.go` (or `server_redirect_test.go`) — `TestTerminalPageRedirect`: covers WEBCHAT-01 redirect response (302 + Location + URL encoding)
- [ ] `TESTING.md` — add WEBCHAT-01 / WEBCHAT-02 / PARITY-01 rows per Section 4 convention
- [ ] `TESTING.md` — add M-31 manual UAT item in a new "Category R — Web-Share Chat Parity (Phase 159)" section

*(No new vitest or Playwright files required — the change is server-side only; existing Playwright chat-parity.spec.ts already proves chat works on the SPA surface, which is where guests now land)*

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Yes | `requireCapability` middleware — unchanged; validates HMAC-JWT before redirect |
| V3 Session Management | No | No session cookies; cap tokens are stateless |
| V4 Access Control | Yes | `requireCapability` + `isGrantActive` + `IsSessionEnabled` — all run before redirect |
| V5 Input Validation | Yes | `url.QueryEscape` for sessionID and token in redirect Location |
| V6 Cryptography | No | HMAC key handling is in existing `requireCapability` — no change |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Open redirect (redirect to attacker-controlled URL) | Elevation of privilege | NOT possible here: redirect target is a hardcoded relative path `/app/?...`; sessionID and token are embedded in the path/query, not used as a redirect destination themselves |
| Cap bypass (accessing `/app/` without valid cap) | Elevation of privilege | The redirect only fires after `requireCapability` validates the token; accessing `/app/` directly skips this gate but is equivalent to the current state (Phase 155 designed `/app/` as a no-gate static bundle) |
| Token leakage via Referer header | Information disclosure | The token is in the redirect Location (HTTP header), not in the body; browsers send the redirected URL as Referer for same-origin navigations; this is the same as the current state where the token is in the URL at `/sessions/{id}?cap=TOKEN` |
| Caching of redirect response | Replay attack | HTTP 302 responses are not cached by browsers by default; the daemon signs short-lived JWTs with expiry; no additional cache-busting needed |

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go stdlib `net/url` | URL encoding in handleTerminalPage | ✓ | already in go.mod | — |
| Go stdlib `net/http` | http.Redirect, http.StatusFound | ✓ | stdlib | — |
| Live daemon | WEBCHAT-02 UAT | ✓ (dev machine) | current | — |
| Tailscale (for WhoIs identity) | Presence identity on join | ✓ (dev machine) | current | Falls back to "unknown:web" sentinels — non-blocking |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `url.QueryEscape` is already importable from `net/url` without adding a new go.mod dependency | Redirect Mechanics | Low risk — `net/url` is standard library |
| A2 | The `/app/` SPA route serving a 503 when `staticAppFS == nil` in test environments is acceptable; the redirect test should not follow the redirect | Validation Architecture | If test infrastructure wires staticAppFS, test expectations change |

---

## Open Questions

1. **Should `handleIssueCapabilities` URL output be updated to mint `/app/` URLs directly?**
   - What we know: `api.go:1288-1289` still mints `/sessions/{id}?cap=` URLs for the GUI share modal. After Phase 159, these redirect correctly.
   - What's unclear: Is it cleaner to update the URL minting in this phase?
   - Recommendation: Do NOT update in Phase 159. The redirect approach works for all paths without touching URL-minting sites. Update URL minting in Phase 160 as tech-debt closeout if desired — but only after Phase 159 is proven stable.

2. **Should the Playwright chat-parity.spec.ts be extended to navigate via the share URL rather than directly to `/app/`?**
   - What we know: The existing spec opens `/app/?session=...&cap=...` directly (already past the redirect). Adding a test that starts from `/sessions/{id}?cap=` and asserts a redirect would improve WEBCHAT-02 automated coverage.
   - What's unclear: The Playwright fixture uses a stub server, not the real webserver. The redirect test belongs in Go (integration test), not Playwright.
   - Recommendation: The Go `TestTerminalPageRedirect` unit test (Wave 0 gap) is sufficient for automated WEBCHAT-02 redirect coverage. Playwright already proves chat works at `/app/`. Manual M-31 covers the end-to-end live-daemon path.

---

## Sources

### Primary (HIGH confidence)
- `internal/webserver/server.go` — `handleTerminalPage` (lines 962-971), route registration (657-661), `handleWSSRelay` (999-1263), `handleJoinExchange` (792-856), `handleSessionQR` (973-989) — read directly
- `internal/webserver/capability_mw.go` — `requireCapability` middleware body — read directly
- `internal/relay/hub.go` — `BroadcastChat` (556-570) — read directly
- `internal/relay/protocol.go` — frame constants 0x30-0x36 (76-86) — read directly
- `frontend/src/lib/webMode.ts` — `readWebModeParams`, `detectMode` (entire file) — read directly
- `frontend/src/components/Hub/WebShareSessionView.tsx` — entire file — read directly
- `frontend/src/App.tsx` — `webParams`, `openWebSessionTab`, `WebShareSessionView` render branch — read directly
- `internal/daemon/api.go` — `issueCapabilitiesForSession` URL minting (1287-1289), `handleExchangeJoinCode` (1340-1387) — read directly
- `internal/webserver/server_test.go` — `TestWebServerToggle` (318-354) — read directly
- `web/assets/terminal.js` — frame constant definitions (lines 1-12) — read directly

### Secondary (MEDIUM confidence)
- `.planning/ROADMAP.md` — Phase 159 root-cause evidence section
- `.planning/REQUIREMENTS.md` — PARITY-01, D-06 definitions
- `.planning/STATE.md` — Phase 159 approach decision

---

## Metadata

**Confidence breakdown:**
- Redirect mechanics: HIGH — implementation read directly from source; Go stdlib
- Chesterton's Fence: HIGH — exhaustive grep of all `/sessions/` callers in Go + TS files
- Frame relay (no change needed): HIGH — BroadcastChat and handleWSSRelay read directly
- Test update requirements: HIGH — TestWebServerToggle read directly; failure mode confirmed
- UAT approach: HIGH — consistent with existing manual UAT patterns in TESTING.md

**Research date:** 2026-06-27
**Valid until:** 2026-07-27 (stable codebase; no external dependencies)
