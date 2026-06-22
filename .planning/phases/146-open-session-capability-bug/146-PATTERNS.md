# Phase 146: Open Session Capability Bug — Pattern Map (OUT-OF-BAND REDESIGN)

**Mapped:** 2026-06-22
**Supersedes:** `superseded-broadcast/146-PATTERNS.md` (broadcast Mechanism B — rejected)
**Design basis:** CONTEXT.md D-02/D-04/D-09..D-12 + RESEARCH.md out-of-band flow
**Files analyzed:** 10 (6 modified source + 4 test files)
**Analogs found:** 10 / 10

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/webserver/server.go` | service (struct + handler) | request-response | self — `joinCodeIssuer` field + `SetJoinCodeIssuer` + `handleSessionsMeta` (remove broadcast wiring) | exact — surgical removal |
| `internal/daemon/api.go` | service (remove helper + wiring) | request-response | self — `mintSessionJoinCodes` extracted from `issueCapabilitiesForSession` | exact — remove wiring at lines 458 + 995 |
| `internal/tailnet/sessions.go` | model (struct field remove) | CRUD | self — `ShareableSessionMeta` struct (lines 122–130) | exact — remove `ROJoinCode`/`RWJoinCode` fields |
| `internal/webserver/sessions_meta_test.go` | test | — | self — `TestSessionsMeta_NoCapInResponse` (lines 150–221) | exact — revert allowed-key map |
| `internal/webserver/sessions_meta_embed_test.go` | test | — | `internal/webserver/sessions_meta_test.go` (RB-03 assertion pattern) | role-match — invert assertion |
| `frontend/src/lib/remoteSession.ts` | model (interface) | — | self — `RemoteSession` interface (lines 12–22) | exact — remove `roJoinCode`/`rwJoinCode` |
| `frontend/src/lib/remoteAdapter.ts` | utility (transform) | transform | self — `adaptRemoteSession` (lines 17–40) | exact — remove code pass-through |
| `frontend/src/App.tsx` | controller (handler + modal state) | request-response | self — `handleModalExchange` (lines 1158–1186), `handleBrowseFilesRemote` (lines 1116–1151) | exact — rewrite `handleOpenRemoteSession` + add open-session modal intent |
| `frontend/src/components/RemoteJoinCodeModal.tsx` | component (modal) | request-response | self — existing `intent` prop + `onExchange` pattern (lines 17–35, 51–94) | exact — add `'open-session'` intent value |
| `frontend/src/components/Hub/SessionCard.tsx` | component (menu item) | event-driven | self — `onOpenInBrowser` handler (lines 404–416) | exact — remove `roJoinCode` gate; open modal unconditionally |

---

## Pattern Assignments

### `internal/webserver/server.go` — remove broadcast, restore cap-free meta

**Work:** Remove `joinCodeIssuer` field (line 113), `SetJoinCodeIssuer` setter (lines 170–178), and the embed block in `handleSessionsMeta` (lines 879–890). Remove `ROJoinCode`/`RWJoinCode` fields from `sessionMetaItem` (lines 59–60). Restore `handleSessionsMeta` comment to cap-free contract.

**Analog:** self — the `handleSessionsMeta` function before Phase 146 wiring.

**Handler shape to restore** (`internal/webserver/server.go` lines 860–895, minus embed block):
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

**Struct to restore** (remove the two join-code fields, lines 53–61):
```go
type sessionMetaItem struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
    // ROJoinCode / RWJoinCode REMOVED — RB-03 cap-free discovery
}
```

**Setter to remove entirely** (`SetJoinCodeIssuer`, lines 170–178):
```go
// DELETE: func (ws *WebServer) SetJoinCodeIssuer(fn func(string) (string, string, error)) {
//     ws.joinCodeIssuer = fn
// }
```

---

### `internal/daemon/api.go` — remove mintSessionJoinCodes + wiring

**Work:** Remove `mintSessionJoinCodes` method (lines 1183–end of function), and remove two `SetJoinCodeIssuer` wiring calls (at lines 458 and 995).

**Analog:** `issueCapabilitiesForSession` (lines 1098–1181) — the proven pattern `mintSessionJoinCodes` was extracted from. It stays untouched; only the broadcast-specific extraction goes away.

**Wiring lines to delete** (lines 458, 995):
```go
// DELETE both occurrences:
ws.SetJoinCodeIssuer(a.mintSessionJoinCodes)
```

**Function to delete entirely** (lines 1183–~1255):
```go
// DELETE: func (a *API) mintSessionJoinCodes(sessionID string) (roCode, rwCode string, err error) { ... }
```

---

### `internal/tailnet/sessions.go` — restore cap-free ShareableSessionMeta

**Work:** Remove `ROJoinCode`/`RWJoinCode` fields from `ShareableSessionMeta` (lines 128–129) and update the struct comment to restore the RB-03 cap-free contract statement.

**Analog:** `PeerSession` struct (lines 19–27) — no credential fields, pure metadata.

**Struct to restore** (lines 122–130):
```go
type ShareableSessionMeta struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cli_type"`
    Status  string `json:"status"`
    URL     string `json:"url"`
    // ROJoinCode / RWJoinCode REMOVED — RB-03: never credentials in meta payload
}
```

---

### `internal/webserver/sessions_meta_test.go` — restore RB-03 cap-free assertion

**Work:** Revert the `allowed` map in `TestSessionsMeta_NoCapInResponse` (lines 192–200) to the original cap-free key set. The broadcast Phase 146 added `ro_join_code` and `rw_join_code` to `allowed`; they must be removed and must appear in `sensitiveKeys` or just absent from the allowed set.

**Analog:** The existing test structure — `allowed` map pattern at lines 192–205:
```go
// RESTORE to cap-free allowed set:
allowed := map[string]bool{
    "id":       true,
    "name":     true,
    "cli_type": true,
    "status":   true,
    "url":      true,
    // ro_join_code / rw_join_code NOT allowed — RB-03 restored
}
```

The test comment at line 188–191 must also be reverted (remove "Phase 146: ro_join_code and rw_join_code are intentionally allowed" caveat).

---

### `internal/webserver/sessions_meta_embed_test.go` — invert to cap-free assertion (RED→GREEN)

**Work:** Replace the embed test file with an inverted test asserting that `ro_join_code`/`rw_join_code` are ABSENT from the response (RB-03 restored). The file currently asserts their presence (superseded broadcast contract).

**Analog:** `TestSessionsMeta_NoCapInResponse` in `sessions_meta_test.go` (lines 150–221) — same `map[string]any` decode + key-set assertion pattern.

**Inverted test pattern to copy:**
```go
// Invert the embed test — assert codes are ABSENT:
func TestSessionsMeta_NoJoinCodesInResponse(t *testing.T) {
    ws, client := testServer(t)
    ws.SetSigningKey(ssExtTestKey)
    ws.EnableSession("sess-rb03")

    resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
    // ... standard decode ...
    item := items[0]

    // RB-03 RESTORED: join codes must NOT appear in the response.
    if _, ok := item["ro_join_code"]; ok {
        t.Error("RB-03 violation: ro_join_code must not appear in cap-free meta response")
    }
    if _, ok := item["rw_join_code"]; ok {
        t.Error("RB-03 violation: rw_join_code must not appear in cap-free meta response")
    }
}
```

**Test setup pattern** (`testServer` helper — in `server_test.go`):
```go
ws, client := testServer(t)
ws.SetSigningKey(ssExtTestKey)
ws.SetSessionResolver(func(id string) (string, string, string, string) { ... })
ws.EnableSession("sess-id")
resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
```

---

### `frontend/src/lib/remoteSession.ts` — remove join-code fields from RemoteSession

**Work:** Remove `roJoinCode` and `rwJoinCode` optional fields from the `RemoteSession` interface (lines 18–22).

**Analog:** self — the interface before Phase 146 fields were added. Target shape:
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
  // roJoinCode / rwJoinCode REMOVED — not broadcast; no field needed
}
```

---

### `frontend/src/lib/remoteAdapter.ts` — remove join-code pass-through

**Work:** Remove the `roJoinCode`/`rwJoinCode` comment block (lines 35–38) and the two field assignments from `adaptRemoteSession`. Remove the `AdaptedRemoteSessionInfo` join-code field declarations (lines 11–14).

**Analog:** self — the `adaptRemoteSession` function shape before Phase 146. The `url` pass-through (line 34) stays; only the join-code lines go:

```typescript
// REMOVE these from AdaptedRemoteSessionInfo:
// roJoinCode?: string
// rwJoinCode?: string

// REMOVE these from adaptRemoteSession return:
// roJoinCode: session.roJoinCode,
// rwJoinCode: session.rwJoinCode,
```

**Field removal is load-bearing for the downstream handler rewrite:** once `roJoinCode` is gone from `AdaptedRemoteSessionInfo`, `handleOpenRemoteSession` in App.tsx cannot check `session.roJoinCode` to gate on sharing — the handler must be rewritten to open the modal instead.

---

### `frontend/src/App.tsx` — rewrite handleOpenRemoteSession + add open-session modal intent

**Work (two sub-tasks):**

1. **Rewrite `handleOpenRemoteSession`** (lines 1082–1111): instead of reading `session.roJoinCode` and calling `ExchangeJoinCodeAtURL` directly, open `RemoteJoinCodeModal` with a new `'open-session'` intent. The join state must also store `sessionId` and `baseURL` (derived from `session.url` via `remoteBaseURLFor`) so the exchange completion handler can construct the cap-bearing URL.

2. **Add open-session branch in `handleModalExchange`** (lines 1158–1186): when `pending.intent === 'open-session'`, after `ExchangeJoinCodeAtURL` succeeds, call `BrowserOpenURL(baseURL + '/sessions/' + sessionId + '?cap=' + token)` instead of `RegisterRemoteCap` + `handleOpenFileBrowser`.

3. **Remove `isPeerSelf`** (lines 1063–1076): dead code per CONTEXT WR-01. No RO/RW guess needed; owner's choice at share time drives the link they send.

**Analog for open-session intent:** `handleModalExchange` hub-modal branch (lines 1177–1183):
```typescript
// Current hub-modal branch (lines 1177–1183):
if (pending.intent === 'hub-modal') {
  capAcquiredRef.current?.(pending.id)
} else {
  handleOpenFileBrowser(pending.id, pending.name)
}

// Add open-session branch:
if (pending.intent === 'open-session') {
  const baseURL = pending.baseURL ?? remoteBaseURLFor(pending)
  BrowserOpenURL(baseURL + '/sessions/' + pending.id + '?cap=' + cap)
} else if (pending.intent === 'hub-modal') {
  capAcquiredRef.current?.(pending.id)
} else {
  handleOpenFileBrowser(pending.id, pending.name)
}
```

**Analog for modal-open on button click** — `handleBrowseFilesRemote` (lines 1116–1151):
```typescript
// Pattern: resolve session, then set joinModalForSession:
const handleBrowseFilesRemote = useCallback(
  async (sessionId: string, sessionName: string) => {
    if (remoteCapsCached.has(sessionId)) {
      handleOpenFileBrowser(sessionId, sessionName)
      return
    }
    let remote = findRemoteSession(sessionId, remotePeers)
    // ... re-poll on not-found ...
    if (!remote) {
      setSaveBanner({ kind: 'error', text: '...' })
      return
    }
    setJoinModalForSession({
      id: sessionId,
      name: sessionName,
      hostname: remote.hostname,
    })
  },
  [remoteCapsCached, remotePeers, handleOpenFileBrowser, setRemotePeers, setSaveBanner],
)
```

**joinModalForSession state shape** — must accommodate new intent + baseURL (lines 211–216):
```typescript
// Current shape (line 211–216):
const [joinModalForSession, setJoinModalForSession] = useState<{
  id: string
  name: string
  hostname: string
  intent?: 'files' | 'hub-modal'
} | null>(null)

// Extended shape for open-session:
const [joinModalForSession, setJoinModalForSession] = useState<{
  id: string
  name: string
  hostname: string
  intent?: 'files' | 'hub-modal' | 'open-session'
  baseURL?: string   // pre-computed for open-session intent; avoids re-deriving in handleModalExchange
} | null>(null)
```

**Analog for error-banner on not-shared** — existing setSaveBanner pattern (lines 1086–1087):
```typescript
setSaveBanner({
  kind: 'error',
  text: 'This session is not shared — enable sharing from the owner\'s Share menu first.',
})
```

**BrowserOpenURL cap-URL construction pattern** (already present at line 1100 — keep this shape in the new branch):
```typescript
BrowserOpenURL(baseURL + '/sessions/' + session.id + '?cap=' + token)
```

---

### `frontend/src/components/RemoteJoinCodeModal.tsx` — add 'open-session' intent value

**Work:** Add `'open-session'` to the `intent` prop union type. The title for this intent is `'Join Remote Session — Open'` (or similar; planning's call per CONTEXT "Claude's Discretion"). No other behavior change needed in the component itself — the intent drives only the title and the body copy.

**Analog:** existing `intent` prop + title derivation (lines 23–24, 59):
```typescript
// Current:
intent?: 'files' | 'hub-modal'
const title = intent === 'files' ? 'Join Remote Session — Files' : 'Join Remote Session'

// Extended:
intent?: 'files' | 'hub-modal' | 'open-session'
const title =
  intent === 'files' ? 'Join Remote Session — Files' :
  intent === 'open-session' ? 'Open Remote Session' :
  'Join Remote Session'
```

Body copy hint: for `'open-session'`, the modal body should say the owner should send a share link/code; "Ask the owner to share the session and send you the join code" (Claude's Discretion on wording).

**The `onExchange: (code: string) => Promise<void>` contract stays unchanged** — the difference is in what `App.tsx`'s `handleModalExchange` does after resolution (opens browser URL instead of registering a remote cap).

---

### `frontend/src/components/Hub/SessionCard.tsx` — remove roJoinCode gate on Open in browser

**Work:** Remove the `roJoinCode` read from the session cast (lines 239–240) and the `disabled={!roJoinCode}` + `title` tooltip on the "Open in browser" menu item (lines 410–411). The button should open the modal unconditionally for all remote sessions (D-03: modal replaces the dead-end 401).

**Analog:** the "Browse files" button (lines 418–422) — no disabled gate:
```typescript
<button
  type="button"
  className="hub-card__menu-item"
  role="menuitem"
  onClick={(e) => { e.stopPropagation(); onBrowseFiles?.(id, name); setMenuOpen(false) }}
>
  Browse files
</button>
```

**Target shape for "Open in browser"** (remove disabled gate; pass session for modal routing):
```typescript
<button
  type="button"
  className="hub-card__menu-item"
  role="menuitem"
  // No disabled — modal will inform user if session is not yet shared
  onClick={(e) => {
    e.stopPropagation()
    onOpenInBrowser?.(session as AdaptedRemoteSessionInfo)
    setMenuOpen(false)
  }}
>
  <ArrowTopRightOnSquareIcon className="hub-card__conn-icon" aria-hidden="true" />
  Open in browser
</button>
```

**Lines to remove** (lines 239–240 — roJoinCode read):
```typescript
// DELETE:
const roJoinCode = (session as { roJoinCode?: string }).roJoinCode
```

---

### `frontend/src/components/Hub/SessionShareModal.tsx` — owner copy affordance (D-12)

**Work:** Verify whether `SessionSharePanel` already renders copyable codes/links. Based on reading `SessionSharePanel.tsx`, it already renders `CodeDisplay` components with "Copy" buttons for both `readCode` and `writeCode` (lines 204, 255), plus "Copy" buttons for the full-access and read-only links (lines 186, 242). D-12 is already satisfied by the existing `SessionSharePanel`. No change required unless the planner's review finds otherwise.

**Confirmation:** `SessionSharePanel` (lines 8–57, 176–268) — `CodeDisplay` inner component with `ClipboardSetText` + copy button at lines 16–24; used for both codes at lines 204 and 255.

---

### Frontend test: `frontend/src/components/__tests__/App.open-remote.test.tsx` — rewrite source-inspection tests

**Work:** Replace the current test file's assertions (which validate the broadcast-based `handleOpenRemoteSession`) with assertions that match the new out-of-band flow.

**Analog:** existing `App.open-remote.test.tsx` (lines 1–65) — source-inspection via `App.tsx?raw` pattern:
```typescript
import { describe, it, expect } from 'vitest'
import raw from '../../App.tsx?raw'

describe('App.tsx — handleOpenRemoteSession (FIX-03 out-of-band)', () => {
  it('opens RemoteJoinCodeModal (not BrowserOpenURL directly) when no cap held', () => {
    const slice = raw.slice(raw.indexOf('handleOpenRemoteSession'), raw.indexOf('handleOpenRemoteSession') + 600)
    expect(slice).toContain('setJoinModalForSession')
    // NOT: expect(slice).toContain('ExchangeJoinCodeAtURL') — that moves to handleModalExchange
  })
  it('handleModalExchange open-session branch calls BrowserOpenURL with ?cap= URL', () => {
    const slice = raw.slice(raw.indexOf('handleModalExchange'), raw.indexOf('handleModalExchange') + 1000)
    expect(slice).toContain('open-session')
    expect(slice).toMatch(/\/sessions\/.*\?cap=/)
  })
})
```

**Established pattern** for `App.tsx?raw` source-inspection tests — also used in `App.fileBrowserMode.test.tsx`:
```typescript
import raw from '../../App.tsx?raw'
// ...
const idx = raw.indexOf('handleOpenRemoteSession')
const slice = raw.slice(idx, idx + 900)
expect(slice).toContain('...')
```

---

### Frontend test: `internal/daemon/mint_join_codes_test.go` — remove (broadcast only)

**Work:** Delete `internal/daemon/mint_join_codes_test.go` entirely — it tests `mintSessionJoinCodes` which is being removed as part of the broadcast cleanup.

**Analog:** No replacement test needed for this file specifically. The equivalent behavior (cap issuance) is already tested by `api_test.go` `TestIssueCapabilities_*` tests.

---

## Shared Patterns

### Modal intent dispatch in App.tsx
**Source:** `App.tsx` `handleModalExchange` (lines 1158–1186) + `joinModalForSession` state (lines 211–216)
**Apply to:** open-session branch in `handleModalExchange`; the intent-discriminator pattern is already established:
```typescript
// Discriminator pattern (existing; extend don't replace):
if (pending.intent === 'hub-modal') {
  capAcquiredRef.current?.(pending.id)
} else {
  handleOpenFileBrowser(pending.id, pending.name)
}
```

### Cap-bearing URL construction
**Source:** `App.tsx` line 1100 (currently inside `handleOpenRemoteSession`)
**Apply to:** new `open-session` branch in `handleModalExchange`
```typescript
BrowserOpenURL(baseURL + '/sessions/' + pending.id + '?cap=' + cap)
```

### Error banner pattern
**Source:** `App.tsx` lines 1086–1087, 1101–1108 (setSaveBanner error reporting)
**Apply to:** `handleOpenRemoteSession` not-shared case + exchange failure case
```typescript
setSaveBanner({ kind: 'error', text: '...' })
```

### Go test helper: testServer + SetSigningKey + EnableSession
**Source:** `internal/webserver/sessions_meta_test.go` lines 26–40 + `server_test.go` (testServer helper)
**Apply to:** inverted RB-03 test in `sessions_meta_embed_test.go`
```go
ws, client := testServer(t)
ws.SetSigningKey(ssExtTestKey)
ws.EnableSession("sess-id")
resp, err := client.Get(ws.BaseURL() + "/api/sessions/meta")
```

### Go JSON key-set assertion
**Source:** `internal/webserver/sessions_meta_test.go` lines 157–220
**Apply to:** inverted embed test — decode to `[]map[string]any`, assert key absence
```go
var items []map[string]any
if err := json.NewDecoder(resp.Body).Decode(&items); err != nil { ... }
item := items[0]
if _, ok := item["ro_join_code"]; ok {
    t.Error("RB-03 violation: ...")
}
```

---

## No Analog Found

None. All files are modifications to existing code following established patterns.

---

## Notes for Planner

1. **App.d.ts is auto-generated** — RESEARCH.md flags the hand-edit as a violation. The planner must NOT hand-edit `App.d.ts` for the `roJoinCode`/`rwJoinCode` removal; those fields disappear when the Wails `RemoteSession` binding regenerates (which happens automatically when `RemoteSession` in the Go type is changed, or when there is no Go `RemoteSession` type carrying those fields). If no Go-side type surfaces them, no `App.d.ts` edit is needed.

2. **`mintSessionJoinCodes` removal is a pure delete** — the function was added entirely within the superseded Phase 146 commits. No partial retention.

3. **`isPeerSelf` removal** — dead code (WR-01 per CONTEXT). Removing it removes the `tailscaleHealth` dependency from `handleOpenRemoteSession`'s closure, which simplifies the deps array.

4. **D-12 copy affordance** — `SessionSharePanel` already provides copyable codes via `CodeDisplay` (verified in `SessionSharePanel.tsx` lines 8–57). If the planner's wave-0 source inspection confirms this, no `SessionShareModal` change is required for D-12.

5. **Wave 0 test contract (RESEARCH § Validation Architecture):** the re-planned Wave 0 must write RED tests for the inverted RB-03 assertion (no join codes in meta) AND a behavior-level assertion that `handleOpenRemoteSession` opens the modal — not just source-inspection. The prior mistake was source-inspecting only.

---

## Metadata

**Analog search scope:** `internal/webserver/`, `internal/daemon/`, `internal/tailnet/`, `frontend/src/`, `frontend/src/components/`, `frontend/src/lib/`
**Files read:** 14 (server.go, capability_mw.go, sessions_meta_test.go, sessions_meta_embed_test.go, tailnet/sessions.go, daemon/api.go, daemon/client_remote_files.go, daemon/mint_join_codes_test.go, remoteSession.ts, remoteAdapter.ts, App.tsx, RemoteJoinCodeModal.tsx, SessionShareModal.tsx, SessionSharePanel.tsx, SessionCard.tsx, RemoteJoinCodeModal.test.tsx, App.open-remote.test.tsx)
**Pattern extraction date:** 2026-06-22
