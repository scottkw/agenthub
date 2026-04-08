# Phase 52: Remote Sessions GUI Panel - Research

**Researched:** 2026-04-07
**Domain:** React/TypeScript frontend panel, Wails bindings, Go app.go method, remote session URL construction
**Confidence:** HIGH

## Summary

Phase 52 is a pure frontend-and-bindings phase. The backend is fully implemented: `internal/tailnet` (Phase 50 P01), `GET /tailnet/peers` daemon route with 30s cache (Phase 50 P02), and `DaemonClient.ListTailnetPeers()` (Phase 50 P02) are all in production. The daemon returns `[]tailnet.Peer` (hostname, DNSName, tailscaleIPs, os, online fields) from the Unix socket API.

The work splits into three concerns:
1. **Go: `app.go` Wails binding** — add `ListRemoteSessions()` method that calls `client.ListTailnetPeers()`, then for each peer fetches `/api/sessions` over HTTPS to get the session list, and returns a flat struct the frontend can consume. Alternatively (simpler): expose `ListTailnetPeers()` directly and let the frontend construct URLs; however, a Go-layer approach that fetches per-peer sessions is better because the frontend cannot make arbitrary HTTPS cross-origin requests without CORS issues. The correct architecture is: Go binding returns `[]RemotePeer` where each peer has its sessions embedded, and the session URL is computed in Go.
2. **Frontend: `RemoteSessionsPanel.tsx` component** — displays sessions grouped by peer hostname, with host, session name, agent type, status visible; loading spinner while probing; 30-second auto-refresh; "No remote peers found" empty state; "Open" button per session.
3. **Frontend: Tab bar / navigation wiring** — add a "Remote" button to the tab bar controls (mirrors the existing "Sessions" hamburger button pattern), with a dedicated `remote-sessions` tab type in the `Tab` union.

The existing `DaemonManagerPanel` is the direct architectural model: panel component receives data as props, `App.tsx` owns the polling interval (`setInterval` / cleanup on tab deactivation), and the tab type distinguishes the panel. The same `daemon-panel` BEM CSS naming convention should be extended with a `remote-panel` namespace.

**Primary recommendation:** Add `GetRemoteSessions()` Wails binding in `app.go` that returns `[]RemotePeerSessions` (peer hostname + pre-built session URLs array). Frontend component mirrors `DaemonManagerPanel` pattern exactly. Wails binding must be manually added to `App.js` + `App.d.ts` (established Phase 51 workaround — `wails dev` is not run between phases).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REM-02 | User can view a list of remote sessions with host, session name, agent type, and status in a dedicated GUI panel | `RemoteSessionsPanel.tsx` component consuming `GetRemoteSessions()` Wails binding; sessions grouped by peer hostname with host, name, cli_type (agent type), status fields from `/api/sessions` JSON |
| REM-03 | User can open a remote session in the web browser directly from the GUI remote panel | "Open" button in panel calls `BrowserOpenURL(sessionURL)` from Wails runtime; URL constructed as `https://{peer.DNSName_stripped}:{port}/sessions/{session.id}` |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 18.x (existing) | Component rendering | Project standard frontend framework |
| TypeScript | 5.x (existing) | Type safety for component props/state | Project convention (all components typed) |
| Vitest + jsdom | existing | Unit tests for component | Existing test framework for all frontend components |
| Wails runtime `BrowserOpenURL` | v2 (existing) | Open session URL in system browser | Already used in WelcomeTab for release URL opening |
| `net/http` + `crypto/tls` stdlib | Go stdlib | Fetch `/api/sessions` from each peer in `GetRemoteSessions()` | Already used in `internal/tailnet.probePeer` for peer probing |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `DaemonClient.ListTailnetPeers()` | existing Phase 50 output | Fetch discovered peers from daemon | Called inside `app.go` `GetRemoteSessions()` binding |
| `strings.TrimSuffix(peer.DNSName, ".")` | Go stdlib | Strip trailing dot from DNSName for URL construction | Every place a URL is built from `peer.DNSName` |
| `tailnet.DefaultProbePort` (7443) | existing constant | Default port for remote session URLs | URL construction in `GetRemoteSessions()` |

**Installation:** No new dependencies. All libraries already present.

## Architecture Patterns

### Recommended Project Structure

New files:
```
frontend/src/components/
└── RemoteSessionsPanel.tsx         # new panel component
frontend/src/components/__tests__/
└── RemoteSessionsPanel.test.tsx    # new test file
```

Modified files:
```
app.go                              # add GetRemoteSessions() Wails binding
frontend/src/wailsjs/go/main/
├── App.d.ts                        # add RemotePeerSessions interface + GetRemoteSessions() signature
└── App.js                          # add GetRemoteSessions export
frontend/src/App.tsx                # wire panel: tab type, polling, REMOTE_SESSIONS_TAB constant
frontend/src/components/TabBar.tsx  # add 'remote-sessions' to Tab type union, add button
frontend/src/style.css              # add .remote-panel BEM CSS block
```

### Pattern 1: New Tab Type (mirrors daemon-manager)

**What:** Add `'remote-sessions'` to the `Tab.type` union in `TabBar.tsx`. Add a toolbar button in `TabBar` controls that calls `onOpenRemoteSessions` prop. In `App.tsx`, define `REMOTE_SESSIONS_TAB` constant and handle it in the `terminal-container` render block.

**When to use:** Every special panel tab (welcome, daemon-manager) follows this pattern.

**Example:**
```typescript
// TabBar.tsx — extend type union
export interface Tab {
  id: string
  name: string
  sessionId: string
  cli: string
  type?: 'terminal' | 'welcome' | 'daemon-manager' | 'remote-sessions'  // add 'remote-sessions'
}

// TabBar toolbar — add remote button
<button
  className="tab-bar__btn tab-bar__btn--remote"
  onClick={onOpenRemoteSessions}
  title="Remote sessions"
  aria-label="Remote sessions"
>
  &#127760;{/* globe icon or similar */}
</button>
```

```typescript
// App.tsx — constant and polling
const REMOTE_SESSIONS_TAB: Tab = { id: '__remote_sessions__', name: 'Remote', sessionId: '', cli: '', type: 'remote-sessions' }

// Poll when remote-sessions tab is active (mirrors daemon-manager polling pattern)
useEffect(() => {
  const isRemoteActive = activeId === REMOTE_SESSIONS_TAB.id
  if (!isRemoteActive) return

  let cancelled = false
  async function refresh() {
    setRemoteLoading(true)
    try {
      const peers = await GetRemoteSessions()
      if (!cancelled) {
        setRemotePeers(peers)
        setRemoteLoading(false)
      }
    } catch (err) {
      if (!cancelled) setRemoteLoading(false)
    }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 30_000)
  return () => {
    cancelled = true
    clearInterval(interval)
  }
}, [activeId])
```

**CRITICAL:** The polling interval is 30 seconds (matching the daemon-side cache TTL). The loading spinner is shown while `remoteLoading` is true. Setting `remoteLoading` back to false after each refresh (success or error) avoids a perpetual spinner.

### Pattern 2: Go Wails Binding for Remote Sessions

**What:** `GetRemoteSessions()` in `app.go` calls `client.ListTailnetPeers()`, then for each peer concurrently fetches `GET https://{fqdn}:{port}/api/sessions`, parses the JSON (same `sessionListItem` shape as webserver returns), and assembles a `[]RemotePeerSessions` response. Returns the array even if some peers fail — individual peer errors are silently omitted.

**Data shapes:**

```go
// app.go — new types
type RemoteSession struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cliType"`
    Status  string `json:"status"`
    URL     string `json:"url"`      // https://{fqdn}:{port}/sessions/{id}
}

type RemotePeerSessions struct {
    Hostname string          `json:"hostname"`
    Sessions []RemoteSession `json:"sessions"`
}
```

**URL construction pattern:**
```go
// Strip trailing dot, build session URL using same port as probe
fqdn := strings.TrimSuffix(peer.DNSName, ".")
sessionURL := fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, tailnet.DefaultProbePort, session.ID)
```

**Concurrency:** Use `errgroup` with `SetLimit(5)` (same as `internal/tailnet.probeAll`) to fetch from multiple peers concurrently. Each peer fetch has a 5-second timeout.

```go
// app.go — GetRemoteSessions implementation sketch
func (a *App) GetRemoteSessions() []RemotePeerSessions {
    if a.client == nil {
        return []RemotePeerSessions{}
    }
    peers, err := a.client.ListTailnetPeers()
    if err != nil || len(peers) == 0 {
        return []RemotePeerSessions{}
    }

    var mu sync.Mutex
    var results []RemotePeerSessions

    g, gctx := errgroup.WithContext(context.Background())
    g.SetLimit(5)
    for _, p := range peers {
        p := p
        g.Go(func() error {
            ctx, cancel := context.WithTimeout(gctx, 5*time.Second)
            defer cancel()
            sessions := fetchPeerSessions(ctx, p)
            if len(sessions) > 0 {
                mu.Lock()
                results = append(results, RemotePeerSessions{Hostname: p.Hostname, Sessions: sessions})
                mu.Unlock()
            }
            return nil
        })
    }
    _ = g.Wait()
    return results
}
```

**Why fetch in Go, not frontend:** The Wails WebView cannot make arbitrary HTTPS requests to Tailscale peers (cross-origin restriction; no CORS headers on the peer's webserver). The Go layer has no such restriction.

### Pattern 3: RemoteSessionsPanel Component (mirrors DaemonManagerPanel)

**What:** Functional component receiving `peers: RemotePeerSessions[]`, `loading: boolean` as props. Renders a loading spinner when `loading` is true and peers is empty. Renders "No remote peers found" when `!loading && peers.length === 0`. Renders grouped sections per peer hostname with session rows. Each row has an "Open" button calling `onOpen(session.url)`.

**CSS namespace:** `.remote-panel` (BEM, mirroring `.daemon-panel` block).

**Example:**
```typescript
// RemoteSessionsPanel.tsx
export interface RemotePeerSessions {
  hostname: string
  sessions: RemoteSession[]
}

export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemoteSessionsPanelProps {
  peers: RemotePeerSessions[]
  loading: boolean
  onOpen: (url: string) => void
}

export function RemoteSessionsPanel({ peers, loading, onOpen }: RemoteSessionsPanelProps): React.ReactElement {
  if (loading && peers.length === 0) {
    return (
      <div className="remote-panel">
        <div className="remote-panel__loading">Probing peers...</div>
      </div>
    )
  }
  if (!loading && peers.length === 0) {
    return (
      <div className="remote-panel">
        <div className="remote-panel__empty">No remote peers found</div>
      </div>
    )
  }
  return (
    <div className="remote-panel">
      {peers.map((peer) => (
        <div key={peer.hostname} className="remote-panel__peer">
          <div className="remote-panel__peer-header">{peer.hostname}</div>
          <div className="remote-panel__session-list">
            {peer.sessions.map((s) => (
              <div key={s.id} className="remote-panel__session-row">
                <span className={`remote-panel__status remote-panel__status--${s.status}`} title={s.status} />
                <span className="remote-panel__name">{s.name}</span>
                <span className="remote-panel__cli">{s.cliType}</span>
                <div className="remote-panel__actions">
                  <button
                    className="remote-panel__btn remote-panel__btn--open"
                    onClick={() => onOpen(s.url)}
                  >
                    Open
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
```

**In App.tsx:**
```typescript
// onOpen handler
const handleOpenRemoteSession = useCallback((url: string) => {
  BrowserOpenURL(url)
}, [])
```

### Pattern 4: Manual Wails Binding (Phase 51 established workaround)

**What:** Since `wails dev` rewrites `App.js` and `App.d.ts`, and that command is not run between plan phases, new Wails-bound methods must be manually added to both files.

**Confirmed from STATE.md:**
> [Phase 51]: Manually added Wails bindings (App.d.ts + App.js) for GetLastUpdateInfo/CheckForUpdates as parallel plan workaround

**New entries required in `App.d.ts`:**
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemotePeerSessions {
  hostname: string
  sessions: RemoteSession[]
}

export function GetRemoteSessions(): Promise<RemotePeerSessions[]>
```

**New entry required in `App.js`:**
```javascript
export const GetRemoteSessions = () => Call('main.App.GetRemoteSessions', [])
```

### Anti-Patterns to Avoid

- **Fetching `/api/sessions` from the frontend JS:** Cross-origin request to a Tailscale HTTPS peer from the Wails WebView fails. All remote HTTP fetches must go through the Go binding layer.
- **Blocking the polling while loading spinner is shown:** The 30-second interval should still fire even during a slow probe. The `loading` flag should be set to `true` for the first load only, not on every 30s refresh (or the panel flickers). Consider only setting `loading` during the initial empty-state load.
- **Ignoring per-peer failures:** If one peer's `/api/sessions` fetch fails (timeout, offline), the response should omit that peer silently rather than returning an error.
- **Polling when the tab is inactive:** The `useEffect` cleanup function must call `clearInterval` and set `cancelled = true`, exactly as `DaemonManagerPanel` does.
- **Constructing session URLs with the trailing dot:** `peer.DNSName` always has a trailing dot. Must strip it: `strings.TrimSuffix(peer.DNSName, ".")`.
- **Using `window.open()` instead of `BrowserOpenURL`:** On macOS with Wails, `window.open()` opens in the WebView, not the system browser. Always use the Wails `BrowserOpenURL` runtime function.
- **Race condition on `results` slice in Go goroutines:** Concurrent appends to `[]RemotePeerSessions` without a mutex will cause a data race. Use `sync.Mutex` around the append.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Browser URL opening | `window.open()` or custom IPC | `BrowserOpenURL` from Wails runtime | Wails intercepts `window.open` in WebView; `BrowserOpenURL` uses the OS default browser correctly |
| Concurrent peer fetching | Manual goroutine channels | `errgroup.SetLimit(5)` | Already in go.mod; same pattern as `internal/tailnet.probeAll` |
| Session URL construction | Any other approach | `fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, tailnet.DefaultProbePort, sessionID)` | Matches probe URL pattern; FQDN-only certs require exact format |
| CSS spinner | CSS animation from scratch | Single CSS class with `@keyframes rotate` (project convention) | Check existing style.css for any spinner classes; if none, add a minimal one |

**Key insight:** The entire backend is done. This phase is 90% frontend with a thin Go binding layer that reuses existing infrastructure (`ListTailnetPeers`, HTTPS client pattern from `internal/tailnet`).

## Common Pitfalls

### Pitfall 1: Loading Spinner Flicker on 30-Second Refresh
**What goes wrong:** Setting `remoteLoading = true` at the start of every 30s refresh causes the entire panel to flash "Probing peers..." every 30 seconds even when data is already loaded.
**Why it happens:** The loading state is meant to signal "no data yet", not "refreshing".
**How to avoid:** Only show the spinner when `loading && peers.length === 0`. On refresh cycles when `peers.length > 0`, update silently without showing spinner. Alternatively, use a separate `initialLoading` boolean vs `refreshing` boolean.
**Warning signs:** Panel visibly flickers between content and spinner every 30 seconds.

### Pitfall 2: `peer.DNSName` Trailing Dot in URLs
**What goes wrong:** `RemoteSession.url` contains `https://host.ts.net.:7443/sessions/id` — the dot breaks URL resolution.
**Why it happens:** `tailnet.Peer.DNSName` is documented as "FQDN with trailing dot (e.g. 'host.ts.net.')". The Go binding must strip it.
**How to avoid:** `fqdn := strings.TrimSuffix(peer.DNSName, ".")` before URL construction.
**Warning signs:** "Open" button results in browser showing "host not found" or connection error.

### Pitfall 3: `GetRemoteSessions` Returns Null in JS Instead of Empty Array
**What goes wrong:** Go `nil` slice serializes to JSON `null`; TypeScript receives `null` instead of `[]`.
**Why it happens:** `return []RemotePeerSessions{}` is required (not `return nil`). Same pattern as `ListSessions` in `app.go`.
**How to avoid:** Always return an initialized empty slice, never nil. `if peers == nil { peers = []RemotePeerSessions{} }` guard.
**Warning signs:** Frontend crashes on `.map()` with "Cannot read properties of null".

### Pitfall 4: webserver `/api/sessions` Only Returns Web-Enabled Sessions
**What goes wrong:** The remote `/api/sessions` endpoint only returns sessions where `web-serve` toggle is enabled. A peer with sessions but no web-enabled ones returns an empty array.
**Why it happens:** `handleListSessions` calls `webEnabledSessions()`, which filters by `webEnabled[id]`. This is by design.
**Implication for Phase 52:** REM-02 says "see sessions" — but only web-enabled sessions are visible remotely. This is architecturally correct and consistent with the phase goal (users share sessions via web serving). Document this constraint in the panel UI ("Shows web-enabled sessions only" or similar).
**Warning signs:** User reports "I can see my peer in the list but no sessions appear" — peer has sessions but none are web-enabled.

### Pitfall 5: Wails Binding Missing from App.js or App.d.ts
**What goes wrong:** Frontend calls `GetRemoteSessions()` but gets "undefined is not a function" at runtime; TypeScript compilation fails.
**Why it happens:** Phase 51 established that manual binding additions are required because `wails dev` is not run between plan phases.
**How to avoid:** Both `App.js` (Call export) and `App.d.ts` (type + function signature) must be updated in the same task that adds the Go method.
**Warning signs:** Build fails with "GetRemoteSessions is not exported" or runtime error.

### Pitfall 6: Tab Type Not Updated in TabBar Union
**What goes wrong:** `Tab.type` remains `'terminal' | 'welcome' | 'daemon-manager'`; assigning `type: 'remote-sessions'` causes TypeScript error.
**Why it happens:** `TabBar.tsx` defines the `Tab` interface with an explicit union type.
**How to avoid:** Update the union in `TabBar.tsx` to include `'remote-sessions'` in the same task that adds the tab constant in `App.tsx`.
**Warning signs:** TypeScript error "Type 'remote-sessions' is not assignable to type...".

## Code Examples

### Go: GetRemoteSessions binding (app.go)
```go
// Source: mirrors DaemonClient.ListTailnetPeers pattern (internal/daemon/client.go)
//         mirrors tailnet.probeAll goroutine pool (internal/tailnet/tailnet.go)

// RemoteSession is a session on a remote tailnet peer.
type RemoteSession struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    CLIType string `json:"cliType"`
    Status  string `json:"status"`
    URL     string `json:"url"`
}

// RemotePeerSessions groups sessions by peer hostname.
type RemotePeerSessions struct {
    Hostname string          `json:"hostname"`
    Sessions []RemoteSession `json:"sessions"`
}

// GetRemoteSessions discovers tailnet peers and fetches their session lists.
// Returns an empty slice if the daemon is unreachable or no peers are found.
// Individual peer fetch failures are silently omitted.
func (a *App) GetRemoteSessions() []RemotePeerSessions {
    if a.client == nil {
        return []RemotePeerSessions{}
    }
    peers, err := a.client.ListTailnetPeers()
    if err != nil || len(peers) == 0 {
        return []RemotePeerSessions{}
    }

    var mu sync.Mutex
    results := make([]RemotePeerSessions, 0)

    g, gctx := errgroup.WithContext(context.Background())
    g.SetLimit(5)
    for _, p := range peers {
        p := p
        g.Go(func() error {
            ctx, cancel := context.WithTimeout(gctx, 5*time.Second)
            defer cancel()
            fqdn := strings.TrimSuffix(p.DNSName, ".")
            sessionsURL := fmt.Sprintf("https://%s:%d/api/sessions", fqdn, tailnet.DefaultProbePort)
            sessions := fetchRemoteSessions(ctx, sessionsURL, fqdn, tailnet.DefaultProbePort)
            if len(sessions) > 0 {
                mu.Lock()
                results = append(results, RemotePeerSessions{Hostname: p.Hostname, Sessions: sessions})
                mu.Unlock()
            }
            return nil
        })
    }
    _ = g.Wait()
    return results
}

// fetchRemoteSessions fetches /api/sessions from a single peer and builds URLs.
func fetchRemoteSessions(ctx context.Context, apiURL, fqdn string, port int) []RemoteSession {
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
        },
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
    if err != nil {
        return nil
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil
    }
    var items []struct {
        ID      string `json:"id"`
        Name    string `json:"name"`
        CLIType string `json:"cli_type"`
        Status  string `json:"status"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
        return nil
    }
    sessions := make([]RemoteSession, 0, len(items))
    for _, item := range items {
        sessions = append(sessions, RemoteSession{
            ID:      item.ID,
            Name:    item.Name,
            CLIType: item.CLIType,
            Status:  item.Status,
            URL:     fmt.Sprintf("https://%s:%d/sessions/%s", fqdn, port, item.ID),
        })
    }
    return sessions
}
```

### Frontend: Wails binding stubs

**App.d.ts additions:**
```typescript
export interface RemoteSession {
  id: string
  name: string
  cliType: string
  status: string
  url: string
}

export interface RemotePeerSessions {
  hostname: string
  sessions: RemoteSession[]
}

export function GetRemoteSessions(): Promise<RemotePeerSessions[]>
```

**App.js addition:**
```javascript
export const GetRemoteSessions = () => Call('main.App.GetRemoteSessions', [])
```

### Frontend: App.tsx wiring (key additions only)
```typescript
// Constant
const REMOTE_SESSIONS_TAB: Tab = { id: '__remote_sessions__', name: 'Remote', sessionId: '', cli: '', type: 'remote-sessions' }

// State
const [remotePeers, setRemotePeers] = useState<RemotePeerSessions[]>([])
const [remoteLoading, setRemoteLoading] = useState(false)

// Polling effect
useEffect(() => {
  if (activeId !== REMOTE_SESSIONS_TAB.id) return
  let cancelled = false
  async function refresh() {
    if (!remotePeers.length) setRemoteLoading(true)  // spinner only on empty
    try {
      const peers = await GetRemoteSessions()
      if (!cancelled) {
        setRemotePeers(peers)
        setRemoteLoading(false)
      }
    } catch {
      if (!cancelled) setRemoteLoading(false)
    }
  }
  void refresh()
  const interval = setInterval(() => void refresh(), 30_000)
  return () => { cancelled = true; clearInterval(interval) }
}, [activeId])

// Handler
const handleOpenRemoteSession = useCallback((url: string) => {
  BrowserOpenURL(url)
}, [])

// handleOpenRemoteSessions tab opener (mirrors handleOpenDaemonManager)
const handleOpenRemoteSessions = useCallback(() => {
  const existing = tabs.find((t) => t.type === 'remote-sessions')
  if (existing) { setActiveId(existing.id); return }
  setTabs((prev) => [...prev, REMOTE_SESSIONS_TAB])
  setActiveId(REMOTE_SESSIONS_TAB.id)
}, [tabs])
```

### Frontend: Test pattern (RemoteSessionsPanel.test.tsx)
```typescript
// Source: mirrors DaemonManagerPanel.test.tsx conventions
// Uses source inspection (?raw import) + DOM tests
import raw from '../../components/RemoteSessionsPanel.tsx?raw'
import { RemoteSessionsPanel } from '../../components/RemoteSessionsPanel'

// Source inspection tests verify BEM class names, export, props
it('exports RemoteSessionsPanel function component', () => {
  expect(raw).toContain('export function RemoteSessionsPanel')
})
it('uses BEM class remote-panel__session-row', () => {
  expect(raw).toContain('remote-panel__session-row')
})
it('uses BrowserOpenURL for open action', () => {
  expect(raw).toContain('BrowserOpenURL')
})

// DOM tests
it('renders loading state when loading and no peers', () => { /* ... */ })
it('renders empty state when not loading and no peers', () => { /* ... */ })
it('renders peer group headers', () => { /* ... */ })
it('Open button calls onOpen with session url', () => { /* ... */ })
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Tailscale Services API for peer discovery | `local.Client{}.Status()` + HTTP probe | N/A — Services API never stable | Already resolved in Phase 50; Phase 52 consumes the result |
| Fetch remote sessions from frontend JS | Go binding fetches via HTTPS, returns structured data | Phase 52 design decision | Avoids cross-origin issues in WebView |

## Open Questions

1. **Show all peers or only peers with sessions?**
   - What we know: REM-02 says "see sessions with host, session name, agent type, and status". If a peer has no web-enabled sessions, its section would be empty.
   - Recommendation: Only include peers with at least one session in the returned array (filter in `GetRemoteSessions`). This matches the "No remote peers found" empty state design — if a peer exists but has no sessions, it's not useful to show it.

2. **Should "Open" URL use the web terminal or the dashboard?**
   - What we know: `URL` is constructed as `https://{fqdn}:{port}/sessions/{id}`. The webserver routes `GET /sessions/{id}` to the terminal page (only if web-enabled). The dashboard is at `/dashboard`.
   - Recommendation: Use `/sessions/{id}` — direct access to the terminal for that specific session, which is what the user expects when clicking "Open" on a specific session.

3. **Do we need to re-probe peers on each `GetRemoteSessions()` call?**
   - What we know: `client.ListTailnetPeers()` already returns the 30-second cached result from the daemon's `tailnetCache`. The Go binding doesn't need its own caching layer.
   - Recommendation: Call `ListTailnetPeers()` on each invocation. The 30s daemon cache already prevents excessive Tailscale probing. The per-peer `/api/sessions` fetch is fast (same network, 5s timeout).

## Environment Availability

Step 2.6: SKIPPED — Phase 52 is purely code changes with no new external dependencies. All required tools (Go, pnpm, Vitest, existing modules) confirmed present from prior phases.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest + jsdom (frontend), Go testing stdlib (backend) |
| Config file | `frontend/vite.config.ts` (test: { environment: 'jsdom', globals: true }) |
| Quick run command | `cd /Users/ken/dev/agenthub/frontend && pnpm test run` |
| Full suite command | `cd /Users/ken/dev/agenthub/frontend && pnpm test run && cd /Users/ken/dev/agenthub && go test ./... -race` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REM-02 | RemoteSessionsPanel renders peer groups with hostname, session name, cliType, status | unit (source + DOM) | `pnpm test run RemoteSessionsPanel` | ❌ Wave 0 |
| REM-02 | RemoteSessionsPanel renders loading state | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ Wave 0 |
| REM-02 | RemoteSessionsPanel renders empty state "No remote peers found" | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ Wave 0 |
| REM-02 | GetRemoteSessions Go binding returns structured array (not nil) | unit (Go) | `go test ./... -race -run TestGetRemoteSessions` | ❌ Wave 0 |
| REM-03 | Open button calls onOpen with session URL | unit (DOM) | `pnpm test run RemoteSessionsPanel` | ❌ Wave 0 |
| REM-03 | RemoteSessionsPanel uses BrowserOpenURL (source inspection) | unit (source) | `pnpm test run RemoteSessionsPanel` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `cd /Users/ken/dev/agenthub/frontend && pnpm test run`
- **Per wave merge:** `cd /Users/ken/dev/agenthub/frontend && pnpm test run && cd /Users/ken/dev/agenthub && go test ./... -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `frontend/src/components/RemoteSessionsPanel.tsx` — component does not exist yet; create in Wave 0
- [ ] `frontend/src/components/__tests__/RemoteSessionsPanel.test.tsx` — test file for source inspection + DOM tests
- [ ] `app_test.go` additions — test for `GetRemoteSessions` nil-safety (returns `[]RemotePeerSessions{}` not nil on daemon-nil path)
- [ ] No new framework install needed — Vitest and Go testing tooling confirmed present

## Project Constraints (from CLAUDE.md)

- **JS/TS:** `camelCase`, `PascalCase` components, ESLint + Prettier, TypeScript types — all prop interfaces must be explicitly typed
- **Node: pnpm preferred** — use `pnpm test run` not `npm test`
- **Testing: 80%+ coverage in critical components** — `RemoteSessionsPanel.tsx` must have source inspection + DOM tests
- **Core principles:** No silent fallbacks — `GetRemoteSessions` returns `[]RemotePeerSessions{}` (never nil) to prevent null crashes
- **Chesterton's Fence:** `TabBar.tsx` Tab type union is explicit by design — extend it, don't replace it

## Sources

### Primary (HIGH confidence)
- `internal/tailnet/tailnet.go` — Peer struct fields, DefaultProbePort constant, DNSName trailing dot documentation (direct code inspection)
- `internal/daemon/client.go` — `ListTailnetPeers()` method (direct code inspection, Phase 50 P02 output)
- `internal/daemon/tailnet_cache.go` — 30s cache TTL, thundering-herd pattern (direct code inspection)
- `internal/daemon/api.go` — `handleTailnetPeers` handler, `GET /tailnet/peers` route (direct code inspection)
- `internal/webserver/server.go` — `/api/sessions` response shape: `sessionListItem{ID, Name, CLIType, Status, Hostname}` (direct code inspection)
- `frontend/src/App.tsx` — polling pattern, tab type, daemon-manager panel wiring (direct code inspection)
- `frontend/src/components/DaemonManagerPanel.tsx` — panel component conventions, BEM naming, props interface (direct code inspection)
- `frontend/src/components/TabBar.tsx` — Tab type union, tab bar controls pattern (direct code inspection)
- `frontend/src/wailsjs/go/main/App.d.ts` and `App.js` — manual binding stub format (direct code inspection)
- `frontend/src/components/WelcomeTab.tsx` — `BrowserOpenURL` import and usage pattern (direct code inspection)
- `frontend/src/components/__tests__/DaemonManagerPanel.test.tsx` — test conventions: ?raw source inspection + DOM tests (direct code inspection)
- `.planning/STATE.md` — locked decisions, Phase 51 manual Wails binding workaround, Phase 50 injectable patterns
- `frontend/vite.config.ts` — Vitest configuration (direct code inspection)

### Secondary (MEDIUM confidence)
- Wails v2 `BrowserOpenURL` documentation (confirmed via `runtime.d.ts` generated file in project)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries in project already, no new deps
- Architecture: HIGH — all patterns directly observed from existing codebase (DaemonManagerPanel, WelcomeTab, Phase 50 output)
- Go binding design: HIGH — mirrors established patterns; `errgroup` already used in `internal/tailnet`
- Pitfalls: HIGH — trailing dot and nil-slice patterns confirmed from Phase 50 code; manual binding workaround confirmed from STATE.md
- Wails BrowserOpenURL behavior: HIGH — confirmed from `WelcomeTab.tsx` existing usage and runtime.d.ts

**Research date:** 2026-04-07
**Valid until:** 2026-05-07 (stable project; no external API dependency)
