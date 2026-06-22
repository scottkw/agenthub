# Phase 146: Open Session Capability Bug — Pattern Map

**Mapped:** 2026-06-22
**Files analyzed:** 10 (6 modified source files + 4 new/modified test files)
**Analogs found:** 10 / 10

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/webserver/server.go` | service (struct + setter + handler) | request-response | `internal/webserver/server.go` (self — `SetJoinCodes`, `SetSessionResolver`, `handleSessionsMeta`) | exact |
| `internal/daemon/api.go` | service (extract helper) | request-response | `internal/daemon/api.go` (self — `issueCapabilitiesForSession`) | exact |
| `internal/tailnet/sessions.go` | model (struct field add) | CRUD | `internal/tailnet/sessions.go` (self — `ShareableSessionMeta`) | exact |
| `frontend/src/wailsjs/go/main/App.d.ts` | config (TS type) | — | `frontend/src/wailsjs/go/main/App.d.ts` (self — `RemoteSession` interface) | exact |
| `frontend/src/lib/remoteSession.ts` | model (interface field add) | — | `frontend/src/lib/remoteSession.ts` (self — `RemoteSession` interface) | exact |
| `frontend/src/lib/remoteAdapter.ts` | utility (pass-through mapping) | transform | `frontend/src/lib/remoteAdapter.ts` (self — `adaptRemoteSession`) | exact |
| `frontend/src/App.tsx` | controller (callback) | request-response | `frontend/src/App.tsx` (self — `handleModalExchange`, `handleBrowseFilesRemote`) | exact |
| `internal/webserver/sessions_meta_embed_test.go` | test | — | `internal/webserver/sessions_meta_test.go` | exact |
| `internal/daemon/mint_join_codes_test.go` | test | — | `internal/daemon/api_test.go` (§`issueCapsTestSetup`, `TestIssueCapabilities_*`) | exact |
| `frontend/src/components/__tests__/App.open-remote.test.tsx` | test | — | `frontend/src/components/__tests__/App.fileBrowserMode.test.tsx` + `SessionCard.share.test.tsx` | exact |

---

## Pattern Assignments

### `internal/webserver/server.go` — struct field + setter + handler changes

**Analog:** self (`internal/webserver/server.go`)

**Existing `SetJoinCodes` setter pattern** (lines 274–282) — template for the new `SetJoinCodeIssuer` setter:
```go
// SetJoinCodes installs the join-code manager used by handleJoinExchange
// (D-09/D-11). Plan 04 wires this at daemon startup; Plan 06 consumes
// ws.joinCodes in the exchange handler. The swap is race-free against
// concurrent readers via ws.mu.
func (ws *WebServer) SetJoinCodes(jc *capability.JoinCodeManager) {
    ws.mu.Lock()
    ws.joinCodes = jc
    ws.mu.Unlock()
}
```

**Existing `sessionResolver` field declaration pattern** (lines 99–100) — template for the new `joinCodeIssuer` field:
```go
// sessionResolver is set once before Start() and is not mutex-protected.
sessionResolver func(sessionID string) (name, cliType, status, hostname string)
```

**Existing `sessionMetaItem` struct** (lines 42–54) — add `ROJoinCode`/`RWJoinCode` fields here:
```go
type sessionMetaItem struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
    // Phase 146: add below
    // ROJoinCode string `json:"ro_join_code,omitempty"`
    // RWJoinCode string `json:"rw_join_code,omitempty"`
}
```

**Existing `handleSessionsMeta` handler** (lines 831–859) — the function to extend:
```go
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
**What to add:** after building `sessionURL`, call `ws.joinCodeIssuer(id)` if non-nil and populate `ROJoinCode`/`RWJoinCode` fields. Follow the nil-guard pattern used for `ws.sessionResolver`.

---

### `internal/daemon/api.go` — new `mintSessionJoinCodes` helper

**Analog:** `internal/daemon/api.go` — `issueCapabilitiesForSession` (lines 1083–1175)

**Full function to extract from** (lines 1092–1175):
```go
func (a *API) issueCapabilitiesForSession(sessionID string) (readURL, writeURL, readCode, writeCode string, err error) {
    a.signingKeyMu.RLock()
    key := a.signingKey
    a.signingKeyMu.RUnlock()
    if key == nil {
        return "", "", "", "", errors.New("capability: signing key not bootstrapped")
    }

    a.mu.RLock()
    ws := a.webServer
    a.mu.RUnlock()
    if ws == nil {
        return "", "", "", "", errors.New("web server not running")
    }
    if a.joinCodes == nil {
        return "", "", "", "", errors.New("capability: join-code manager not bootstrapped")
    }

    var rgid, wgid [16]byte
    if _, err := rand.Read(rgid[:]); err != nil {
        return "", "", "", "", err
    }
    if _, err := rand.Read(wgid[:]); err != nil {
        return "", "", "", "", err
    }

    now := time.Now().Unix()
    rPerms := "read"
    wPerms := "read,write"
    // (browse matrix omitted for brevity — copy it verbatim)

    rClaims := capability.Claims{SID: sessionID, Perms: rPerms, IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
    wClaims := capability.Claims{SID: sessionID, Perms: wPerms, IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}

    rTok, err := capability.Sign(rClaims, key)
    if err != nil {
        return "", "", "", "", err
    }
    wTok, err := capability.Sign(wClaims, key)
    if err != nil {
        return "", "", "", "", err
    }

    // CRITICAL: register grants BEFORE returning so cap is valid immediately.
    ws.AddGrant(sessionID, rClaims.GrantID)
    ws.AddGrant(sessionID, wClaims.GrantID)

    base := ws.BaseURL()
    readURL = base + "/sessions/" + sessionID + "?cap=" + rTok
    writeURL = base + "/sessions/" + sessionID + "?cap=" + wTok

    readCode, err = a.joinCodes.Issue(rTok)
    if err != nil {
        return "", "", "", "", err
    }
    writeCode, err = a.joinCodes.Issue(wTok)
    if err != nil {
        return "", "", "", "", err
    }
    return readURL, writeURL, readCode, writeCode, nil
}
```

**New `mintSessionJoinCodes` extracts the token-mint + grant-register + code-issue steps** (without URL construction). Signature:
```go
// mintSessionJoinCodes mints fresh RO+RW capability tokens for sessionID,
// registers both grant_ids on the WebServer, and issues a short-lived join
// code for each. Returns (roCode, rwCode, error). Called by the
// joinCodeIssuer callback wired to WebServer.handleSessionsMeta.
//
// Pitfall: AddGrant MUST be called before returning codes. The
// requireCapability middleware checks isGrantActive on every request;
// tokens from grants not yet registered are immediately rejected.
func (a *API) mintSessionJoinCodes(sessionID string) (roCode, rwCode string, err error)
```

**Wiring pattern** — at daemon startup (follow the existing `SetJoinCodes` wiring site):
```go
ws.SetJoinCodes(a.joinCodes)
ws.SetJoinCodeIssuer(a.mintSessionJoinCodes)  // Phase 146
```

---

### `internal/tailnet/sessions.go` — `ShareableSessionMeta` struct field add

**Analog:** self — `ShareableSessionMeta` (lines 117–123):
```go
type ShareableSessionMeta struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
    // Phase 146: add:
    // ROJoinCode string `json:"ro_join_code,omitempty"`
    // RWJoinCode string `json:"rw_join_code,omitempty"`
}
```
The new fields flow from the JSON decoder in `doFetchSessionsMeta` (line 156) automatically — no decoder changes needed (stdlib `json.Decode` maps matching keys into matching struct fields).

---

### `frontend/src/wailsjs/go/main/App.d.ts` — `RemoteSession` interface field add

**Analog:** self — `RemoteSession` interface (lines 105–111):
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
  // Phase 146: add:
  // roJoinCode?: string
  // rwJoinCode?: string
}
```
Use optional (`?`) fields — peers that have not been upgraded yet will not include these fields, so the viewer must treat absence as "not shared yet."

---

### `frontend/src/lib/remoteSession.ts` — `RemoteSession` interface field add

**Analog:** self — `RemoteSession` interface (lines 12–18):
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
  // Phase 146: add:
  // roJoinCode?: string
  // rwJoinCode?: string
}
```
Note: this interface is the canonical frontend type. `App.d.ts` is the Wails-auto-generated stub — both must stay in sync.

---

### `frontend/src/lib/remoteAdapter.ts` — `adaptRemoteSession` pass-through

**Analog:** self — full file (lines 1–35). The pattern for adding a new field is to add it in the returned object literal, mirroring how `url` was added for CR-01 (line 27):

```typescript
export type AdaptedRemoteSessionInfo = SessionInfo & { url: string }

export function adaptRemoteSession(
  peer: RemotePeerSessions,
  session: RemoteSession,
): AdaptedRemoteSessionInfo {
  return {
    // ... existing fields ...
    url: session.url,          // CR-01: existing pass-through
    // Phase 146: add:
    // roJoinCode: session.roJoinCode,
    // rwJoinCode: session.rwJoinCode,
  }
}
```
The `AdaptedRemoteSessionInfo` type intersection must also be extended if `roJoinCode`/`rwJoinCode` need to be accessible via the typed `AdaptedRemoteSessionInfo` shape. The cleanest approach is to extend the intersection:
```typescript
export type AdaptedRemoteSessionInfo = SessionInfo & { url: string; roJoinCode?: string; rwJoinCode?: string }
```

---

### `frontend/src/App.tsx` — `handleOpenRemoteSession` replacement

**Analog:** `frontend/src/App.tsx` — `handleBrowseFilesRemote` (lines 1069–1103) and `handleModalExchange` (lines 1111–1138).

**Current implementation to replace** (lines 1062–1064):
```typescript
const handleOpenRemoteSession = useCallback((url: string) => {
  BrowserOpenURL(url)
}, [])
```

**Pattern from `handleBrowseFilesRemote`** — join-code modal path with `ExchangeJoinCodeAtURL` + banner on error:
```typescript
const handleBrowseFilesRemote = useCallback(
  async (sessionId: string, sessionName: string) => {
    // ... pre-flight check ...
    const baseURL = remoteBaseURLFor(remote)
    if (!baseURL) throw new Error('session-gone')
    const cap = await ExchangeJoinCodeAtURL(baseURL, code)
    // ... register cap + open tab ...
  },
  [remoteCapsCached, remotePeers, handleOpenFileBrowser, setRemotePeers, setSaveBanner],
)
```

**Pattern from `handleModalExchange`** — error handling with `setSaveBanner`:
```typescript
// Error → informative banner, not silent no-op or unhandled rejection
setSaveBanner({
  kind: 'error',
  text: 'Remote session is no longer available — refresh peers and try again.',
})
```

**Imports already present in App.tsx** (lines 35–53) — no new imports needed:
```typescript
import { ExchangeJoinCodeAtURL, RegisterRemoteCap } from './wailsjs/go/main/App'
import { BrowserOpenURL } from './wailsjs/wailsjs/runtime/runtime'
import { remoteBaseURLFor } from './lib/remoteSession'
```

**New `handleOpenRemoteSession` signature** (change from `(url: string)` to `(session: AdaptedRemoteSessionInfo)`):
```typescript
import type { AdaptedRemoteSessionInfo } from './lib/remoteAdapter'

const handleOpenRemoteSession = useCallback(
  async (session: AdaptedRemoteSessionInfo): Promise<void> => {
    if (!session.roJoinCode) {
      setSaveBanner({
        kind: 'error',
        text: 'This session is not shared. Enable sharing from the Share button to open it in a browser.',
      })
      return
    }
    const baseURL = remoteBaseURLFor(session)
    if (!baseURL) {
      setSaveBanner({ kind: 'error', text: 'Could not determine remote session URL.' })
      return
    }
    // D-05/D-06: prefer RW when viewer is the session owner (peer hostname = local node).
    const code = (session.rwJoinCode && isPeerSelf(session.hostname, tailscaleStatus))
      ? session.rwJoinCode
      : session.roJoinCode
    try {
      const token = await ExchangeJoinCodeAtURL(baseURL, code)
      BrowserOpenURL(baseURL + '/sessions/' + session.id + '?cap=' + token)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      if (msg.includes('expired') || msg.includes('session-gone')) {
        setSaveBanner({
          kind: 'error',
          text: 'Session share link expired — click Open again to get a fresh link.',
        })
      } else {
        setSaveBanner({ kind: 'error', text: 'Could not open session: ' + msg })
      }
    }
  },
  [setSaveBanner, tailscaleStatus],
)
```

**`isPeerSelf` helper pattern** (inline or local utility):
```typescript
// D-06: true when the session's peer hostname matches the local tailscale node hostname.
// Both originate from the same Tailscale MagicDNS namespace; short hostname comparison.
function isPeerSelf(peerHostname: string, tsStatus: { domain: string } | null): boolean {
  if (!tsStatus?.domain || !peerHostname) return false
  // peerHostname from remotePeers is the short hostname (e.g. "mac-main")
  // tsStatus.domain is "mac-mini.ts.net" — take the part before the first dot
  const localShort = tsStatus.domain.split('.')[0]
  return peerHostname === localShort
}
```
**Note:** RESEARCH.md Open Question 1 — verify at implementation time that hostname format comparison is correct. If formats mismatch, fall back to `roJoinCode` (RO safe default for D-05).

**Prop type cascade** — `onOpenInBrowser` must change from `(url: string)` to `(session: AdaptedRemoteSessionInfo)` in:
- `HubPanel` props (call site at App.tsx line ~1382)
- `SessionCardGrid` props (HubPanel passes it through)
- `SessionCardProps` in `SessionCard.tsx`
- The call site at `SessionCard.tsx:399-419`

---

## Test Pattern Assignments

### `internal/webserver/sessions_meta_embed_test.go` (new file)

**Analog:** `internal/webserver/sessions_meta_test.go` — full file (217 lines)

**Package and import pattern** (lines 1–20):
```go
package webserver_test

import (
    "encoding/json"
    "net/http"
    "sort"
    "testing"
)
```

**Test server setup pattern** (lines 25–36 of `sessions_meta_test.go`):
```go
func TestSessionsMeta_EmbedJoinCodes(t *testing.T) {
    ws, client := testServer(t)
    ws.SetSigningKey(ssExtTestKey)
    ws.SetSessionResolver(func(id string) (string, string, string, string) {
        // return name, cliType, status, hostname
    })
    ws.EnableSession("sess-a")
    // Phase 146: wire a fake joinCodeIssuer:
    ws.SetJoinCodeIssuer(func(id string) (string, string, error) {
        return "ro-code-" + id, "rw-code-" + id, nil
    })
    // GET /api/sessions/meta and assert ro_join_code + rw_join_code present
}
```

**RB-03 allowed-keys update** — in `TestSessionsMeta_NoCapInResponse` (line 189 of existing file):
```go
// EXISTING (to update in sessions_meta_test.go, NOT in the new embed_test.go):
allowed := map[string]bool{
    "id":           true,
    "name":         true,
    "cli_type":     true,
    "status":       true,
    "url":          true,
    "ro_join_code": true,  // Phase 146 addition
    "rw_join_code": true,  // Phase 146 addition
}
```

**Nil-issuer test pattern** — follows `TestSessionsMeta_EmptyWhenNoneEnabled` pattern: wire no `joinCodeIssuer`, assert response items have no `ro_join_code`/`rw_join_code` keys (or empty strings).

---

### `internal/daemon/mint_join_codes_test.go` (new file)

**Analog:** `internal/daemon/api_test.go` — `issueCapsTestSetup` + `TestIssueCapabilities_BrowseOff_NoFilesPerms` (lines 1999–2057)

**Package declaration** (matches all daemon tests):
```go
package daemon
```
*(internal package — accesses unexported `api.mintSessionJoinCodes` directly)*

**Test setup pattern** (lines 1999–2013 of api_test.go):
```go
func mintCodesTestSetup(t *testing.T) (*API, *webserver.WebServer, []byte) {
    t.Helper()
    api, _, _ := testDaemon(t)
    ws, err := webserver.NewWebServer(webserver.Config{
        BindIP: "127.0.0.1",
        Port:   0,
        FQDN:   "test.local",
    }, api.engine.Manager())
    if err != nil {
        t.Fatalf("NewWebServer: %v", err)
    }
    api.SetWebServerForTest(ws)
    key := configureCapabilityStateForTest(t, api, ws)
    return api, ws, key
}
```

**Core test pattern** (follows `TestIssueCapabilities_BrowseOff_NoFilesPerms` at line 2034):
```go
func TestMintSessionJoinCodes(t *testing.T) {
    api, ws, key := mintCodesTestSetup(t)
    sid, err := api.engine.CreateSession(context.Background(), "cat", "test-session", "", nil, 80, 24, nil, nil)
    if err != nil {
        t.Fatalf("CreateSession: %v", err)
    }
    t.Cleanup(func() { _ = api.engine.KillSession(sid) })

    ws.EnableSession(sid)

    roCode, rwCode, err := api.mintSessionJoinCodes(sid)
    if err != nil {
        t.Fatalf("mintSessionJoinCodes: %v", err)
    }
    // Assert non-empty codes
    // Assert grants are registered (probe via isGrantActive or probeGrant)
    // Assert codes are distinct
    // Assert codes exchange for valid tokens
    _ = key  // used in extractClaimsFromURL
}
```

**Grant registration check** — use `ws.isGrantActive` (unexported, must use `probeGrant` helper from api_test.go or check via exchange):
```go
// In tests, verify grants are registered by exchanging the code via the webserver.
// Pattern from api_test.go lines 1152-1169 (TestCapabilityIssueAndExchange).
```

---

### `frontend/src/components/__tests__/App.open-remote.test.tsx` (new file)

**Analog 1:** `frontend/src/components/__tests__/App.fileBrowserMode.test.tsx` — source-inspection via `App.tsx?raw`

**Analog 2:** `frontend/src/components/__tests__/SessionCard.share.test.tsx` — mock setup + render + prop assertion

**File header and mock pattern** (from `SessionCard.share.test.tsx` lines 19–28):
```typescript
import { describe, it, expect, vi } from 'vitest'
import raw from '../../App.tsx?raw'

// Source-inspection tests for handleOpenRemoteSession.
// Mounting the real <App /> requires stubbing ~30 wailsjs imports + xterm;
// source-inspection is the established pattern (see App.fileBrowserMode.test.tsx).
```

**Source-inspection assertions pattern** (from `App.fileBrowserMode.test.tsx`):
```typescript
import raw from '../../App.tsx?raw'

describe('App.tsx — Phase 146 handleOpenRemoteSession', () => {
  it('calls ExchangeJoinCodeAtURL (not BrowserOpenURL directly)', () => {
    // Find handleOpenRemoteSession body in source
    const idx = raw.indexOf('handleOpenRemoteSession')
    expect(idx).toBeGreaterThan(-1)
    const slice = raw.slice(idx, idx + 800)
    expect(slice).toContain('ExchangeJoinCodeAtURL')
  })

  it('selects rwJoinCode when isPeerSelf (D-06)', () => {
    expect(raw).toContain('rwJoinCode')
    expect(raw).toContain('isPeerSelf')
  })

  it('shows error banner when roJoinCode absent (D-03)', () => {
    // assert setSaveBanner call site without roJoinCode
    expect(raw).toMatch(/roJoinCode.*setSaveBanner|setSaveBanner.*roJoinCode/s)
  })

  it('handles expired/session-gone error with informative banner (Pitfall 4)', () => {
    expect(raw).toMatch(/expired|session-gone/)
    // The handler must call setSaveBanner on exchange failure
    const handlerSlice = raw.slice(raw.indexOf('handleOpenRemoteSession'), raw.indexOf('handleOpenRemoteSession') + 1200)
    expect(handlerSlice).toContain('setSaveBanner')
  })

  it('constructs cap-bearing URL: baseURL + /sessions/{id}?cap=TOKEN', () => {
    expect(raw).toMatch(/\/sessions\/.*\?cap=/)
  })
})
```

**For the `remoteAdapter` extension test** (add to existing `remoteAdapter.test.ts`):
```typescript
it('passes roJoinCode through from session (Phase 146)', () => {
  const session = makeSession({ roJoinCode: 'ro-abc', rwJoinCode: 'rw-xyz' })
  const adapted = adaptRemoteSession(makePeer(), session)
  expect((adapted as { roJoinCode?: string }).roJoinCode).toBe('ro-abc')
  expect((adapted as { rwJoinCode?: string }).rwJoinCode).toBe('rw-xyz')
})

it('roJoinCode absent when session has no join code (D-03 not-shared path)', () => {
  const session = makeSession()  // no roJoinCode field
  const adapted = adaptRemoteSession(makePeer(), session)
  expect((adapted as { roJoinCode?: string }).roJoinCode).toBeUndefined()
})
```

---

## Shared Patterns

### Setter/Callback Wiring (Go)
**Source:** `internal/webserver/server.go` — `SetSessionResolver` (lines 153–155), `SetJoinCodes` (lines 278–282), `SetFilesHandler` (lines 178–179)
**Apply to:** `SetJoinCodeIssuer` on `WebServer`; `mintSessionJoinCodes` wiring in `api.go`

Pattern: single setter, set once before `Start()`, not mutex-protected when set from a single goroutine before serving begins. For swap-safe fields like `joinCodes`, use `ws.mu.Lock()`.

```go
func (ws *WebServer) SetJoinCodeIssuer(fn func(string) (string, string, error)) {
    ws.joinCodeIssuer = fn
}
```

### Error Handling (Go — webserver handler)
**Source:** `internal/daemon/api.go` — `handleIssueCapabilities` (lines 1181–1206)
**Apply to:** `handleSessionsMeta` when calling `ws.joinCodeIssuer`

Pattern: if `joinCodeIssuer` returns an error, omit codes (set fields to empty string) rather than returning a 500 — the meta endpoint must remain available even if code issuance fails (degraded mode: Open button shows D-03 hint).

### Error Handling (TypeScript — async callback)
**Source:** `frontend/src/App.tsx` — `handleBrowseFilesRemote` (lines 1085–1103) and `handleModalExchange` (lines 1111–1138)
**Apply to:** `handleOpenRemoteSession`

Pattern: `try/catch` around `await ExchangeJoinCodeAtURL(...)`, map error message substrings to user-facing banner text via `setSaveBanner({ kind: 'error', text: '...' })`.

### JSON Key Guard (Go — test)
**Source:** `internal/webserver/sessions_meta_test.go` — `TestSessionsMeta_NoCapInResponse` (lines 157–216)
**Apply to:** `TestSessionsMeta_EmbedJoinCodes` — update the `allowed` map to include `ro_join_code`/`rw_join_code`.

Sensitive key blacklist to keep in the updated test (line 210):
```go
sensitiveKeys := []string{"cap", "token", "grant", "grants", "content", "key", "signing_key", "hmac"}
```
`ro_join_code` and `rw_join_code` are NOT on this list — they are intentionally allowed. They must be in the `allowed` map so the key-set assertion passes.

### `testServer` + `ssExtTestKey` test infrastructure
**Source:** `internal/webserver/server_test.go` (lines 28–143)
**Apply to:** `internal/webserver/sessions_meta_embed_test.go`

Both helpers (`testServer` and `ssExtTestKey`) are in `server_test.go` in the same `webserver_test` package — the new test file is in the same package and gains access automatically with no additional imports.

### `configureCapabilityStateForTest` + `issueCapsTestSetup` test infrastructure
**Source:** `internal/daemon/api_test.go` (lines 833–847, 1999–2013)
**Apply to:** `internal/daemon/mint_join_codes_test.go`

Both helpers are in `package daemon` (internal test package) — the new test file is in the same package and accesses all unexported fields directly. Use `configureCapabilityStateForTest` verbatim.

---

## No Analog Found

All files have clear analogs in the codebase. No novel patterns are needed.

---

## Anti-Pattern Registry (from RESEARCH.md — copy to planning)

| Anti-Pattern | File Affected | Detection |
|---|---|---|
| Using field name `cap`, `token`, `grant` for the new join-code fields | `sessionMetaItem`, test allowed-map | `TestSessionsMeta_NoCapInResponse` explicit key scan |
| Calling `issueCapabilitiesForSession` from `handleSessionsMeta` directly | `server.go` / `api.go` | `*API` not accessible from `*WebServer` — compile error |
| Issuing join codes without registering grants (`ws.AddGrant`) | `api.go:mintSessionJoinCodes` | `requireCapability` rejects token; verified by `TestMintSessionJoinCodes` |
| Caching join codes across polls (Option B2) | `api.go` | Single-use codes consumed by first viewer poll — second poll gets an expired code |
| Calling `BrowserOpenURL(session.url)` directly in the new handler | `App.tsx` | Bypasses `ExchangeJoinCodeAtURL` — returns to 401 |
| Changing `onOpenInBrowser` type without updating all 4 call sites | `App.tsx`, `HubPanel`, `SessionCardGrid`, `SessionCard` | `pnpm tsc --noEmit` type error cascade (Pitfall 5) |

---

## Metadata

**Analog search scope:** `internal/webserver/`, `internal/daemon/`, `internal/tailnet/`, `frontend/src/`
**Files read:** 18 source + test files
**Pattern extraction date:** 2026-06-22
