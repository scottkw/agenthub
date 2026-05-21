# Phase 122: Remote-Session File Browse Wiring - Research

**Researched:** 2026-05-21
**Domain:** Cross-surface client wiring (React/Wails + Go/Bubble Tea) — pointing existing read-only file-browser surfaces at the remote-machine HTTPS webserver routes instead of (or in addition to) the local daemon socket
**Confidence:** HIGH for code surfaces; MEDIUM for the cap-token acquisition flow (see Open Questions)

## Summary

Phase 122 is a small wiring phase, not new component work. Every load-bearing piece already exists in main:

1. **Remote daemon side (already shipped):** `internal/files/` sandbox + handler (Phase 118) + webserver `/api/files/*` mount under `requireFilesRead` middleware (Phase 119). A `?cap=<token>` query param with `files.read` in `Claims.Perms` is sufficient to read.
2. **Desktop GUI consumer (already shipped):** `FileBrowserTab` accepts `(baseURL, capToken?)` per Phase 120-04 and is exercised end-to-end in web mode by Phase 120-06 (Playwright scenarios 13 + 14 mount the React shell against the webserver under `/app/?session=…&cap=…`).
3. **TUI consumer (already shipped):** `tabFiles` view, `filesModel`, three `tea.Cmd` factories (`loadDirCmd` / `readFileCmd` / `headFileCmd`) — Phase 121 — wired against `*daemon.DaemonClient` (a Unix-socket-bound HTTP client).

**Phase 122 closes exactly two gaps:**

- **GUI gap (App.tsx:1181-1206):** the desktop file-browser tab gate hard-codes `fbBaseURL = http://127.0.0.1:${relayPort}` for non-web mode regardless of whether the session is local or remote, and never supplies a `capToken`. The wiring is openly tagged a v3.5 follow-on in 120-04-SUMMARY.md and 120-06-SUMMARY.md.
- **TUI gap (update.go:381-411):** pressing `f` on a remote-session entry currently emits the toast `"File browser not available for remote sessions in v3.4"` instead of opening `tabFiles` with a remote-flavoured client.

**Primary recommendation:** Introduce a narrow `FilesClient` interface in `internal/tui` (matching the four methods the existing `*daemon.DaemonClient` already exposes) and inject either a Unix-socket client or a new HTTPS-cap client per session. On the desktop GUI side, route remote-session browse through the **local daemon** (not directly from the webview to the remote webserver) to avoid CORS preflight failure — exposing a new daemon endpoint that proxies / forwards to the remote webserver authenticated by a pre-acquired cap. **However, see Open Questions §OQ-1 — the cap-acquisition flow for cross-machine browse is not actually proven by existing code, and may require its own design decision before planning can complete.**

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Detect remote-vs-local session in desktop GUI | Frontend Server (App.tsx) | — | App.tsx is the only place with both `panelSessions` (local) and `remotePeers` (remote) data, and owns tab-opener routing |
| Mint / acquire owner cap with `files.read` | API / Backend (daemon) | — | Locked to the OWNING machine's daemon — `internal/daemon/api.go::issueCapabilitiesForSession` already injects `files.read` into the write-tier cap when `filesReadEnabled()` is true |
| Provide cap to remote viewer (cross-machine) | API / Backend (daemon) | — | Currently NO in-app mechanism transfers a cap from Machine A to Machine B — only the `/join?code=` browser flow does, which leaves the cap in browser cookies/URL params, not in any TUI/Wails accessible store. **See OQ-1.** |
| TUI cross-network file fetch | API / Backend (Go HTTP client) | — | Standard `net/http` over Tailscale TLS — no CORS, no browser involved |
| Desktop GUI cross-network file fetch | API / Backend (daemon proxy recommended) | Browser/Client (direct fetch — BLOCKED by CORS) | The webserver does NOT emit CORS headers; cross-origin browser fetches will fail preflight |
| File browser UI rendering | Browser/Client (FileBrowserTab) | — | Already implemented and tier-correct |
| TUI Files view rendering | Browser/Client (Bubble Tea sub-model) | — | Already implemented and tier-correct |

## Project Constraints (from CLAUDE.md and project memory)

- **Cross-surface parity is release-blocking.** GUI/TUI/CLI must stay in sync; never defer a parity gap without explicit user sign-off. REMOTE-04 + REMOTE-05 are the codified version of this principle for this phase.
- **User is colorblind** — verify color-based UAT at source level (hex constants in code), not by eye. Phase 122 has no new color contracts but TUI toast removal must be verified by source grep, not by terminal observation.
- **`pnpm` preferred** for Node packages. No new packages anticipated in this phase.
- **Wails build requires `-tags wailsassets`** for production — the Phase 120-06 mode-detection pattern depends on the relative-base build remaining intact.
- **Skip discuss when research is complete** — `skip_discuss: true` in config.json. This RESEARCH.md should resolve as many gray areas as possible.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REMOTE-01 | Desktop GUI's React `FileBrowserTab` opens against a remote tailnet session with `(baseURL=remote-URL, capToken=session's web-share cap)` | `FileBrowserTab` already accepts both props (120-04); App.tsx hardcodes loopback (lines 1189-1195). Pending: cap acquisition mechanism (OQ-1) |
| REMOTE-02 | Remote session NOT web-shared → "Enable web sharing to browse" message instead of broken tab | Webserver `/api/files/*` only mounts when `SetFilesHandler` was called — i.e., the remote session has a running webserver. Per Phase 119 `requireFilesRead` returns 401 when no cap is present. Without web-share toggled, the peer's `/api/sessions` listing OMITS the session (`internal/webserver/server.go` checks grants). Detection point: `RemoteSession` shows in panel ↔ session is web-shared on remote |
| REMOTE-03 | TUI `f` on remote → fetches from remote webserver HTTPS with cap, not local Unix socket; remove "File browser not available" toast | `update.go:381-411` is the dispatch site; `internal/tui/files_cmds.go` factories take `*daemon.DaemonClient` concrete type — must become an interface |
| REMOTE-04 | TUI behavior identical local vs remote (nav, preview, filter, status, glamour, refusals) | `filesModel` and `handleFilesKey` are client-agnostic; only the `tea.Cmd` factories touch the client. Identical surface preserved by interface-based injection |
| REMOTE-05 | Cross-surface parity: same observable behavior across GUI/web/TUI | Same `/api/files/*` wire contract on both daemon-loopback (no cap) and webserver-HTTPS (cap required), enforced by single `files.Handler` (Phase 118 stateless design — Phase 119 mounts the same instance on both muxes) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` | go 1.24+ stdlib | TUI's HTTPS-cap client | Already used pervasively (`internal/tailnet/sessions.go`, `internal/daemon/client.go`); zero new deps |
| `crypto/tls` | go 1.24+ stdlib | TLS 1.2+ for tailnet HTTPS | Same `tls.Config{MinVersion: tls.VersionTLS12}` as `internal/tailnet/sessions.go:43` |
| `golang.org/x/sync/errgroup` | already in go.mod | Concurrent peer fetch (already in use) | No change |

### Supporting (frontend)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `frontend/src/lib/filesApi.ts` (existing) | n/a | `FilesApiClient` already supports `capToken` | No new code; instantiation site changes only |
| `frontend/src/lib/webMode.ts` (existing, Phase 120-06) | n/a | Mode detection contract | Reference pattern for detecting remote sessions — do not inline `session.hostname !== localHostname`-style checks; centralize the predicate |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Daemon-side proxy of remote `/api/files/*` for the desktop GUI | Wails webview fetches remote URL directly | Direct fetch is blocked by browser CORS (webserver emits no `Access-Control-*` headers). Proxy requires new daemon route but reuses tailnet TLS path already in `internal/tailnet/` |
| New `FilesClient` interface in `internal/tui` | Extend `*daemon.DaemonClient` itself to switch between Unix and HTTPS modes | Interface is cleaner — keeps `DaemonClient` single-purpose (Unix socket only) and avoids per-call branching. Three sites (`loadDirCmd`, `readFileCmd`, `headFileCmd`) need parameterisation |
| Inject `(baseURL, capToken)` into `tea.Cmd` factories directly | Build a typed client wrapper struct | Type-safety wins. Per-Cmd injection means each call site repeats the same 3 args |

**Installation:** No new packages. All work uses existing stdlib + existing `internal/...` types.

## Package Legitimacy Audit

Not applicable — no external packages installed in this phase. All code uses Go stdlib + types already in the module (`internal/files`, `internal/daemon`, `internal/tailnet`, `internal/tui`).

## Architecture Patterns

### System Architecture Diagram

```
DESKTOP GUI (Wails webview, owner of Machine A's UI)
   │
   ├── Local session                          Remote session (Machine B's)
   │       │                                          │
   │       ▼                                          ▼
   │   FileBrowserTab                          FileBrowserTab
   │   (baseURL=loopback,                      (baseURL=??? + capToken=???)
   │    no cap)                                          │
   │       │                                  ┌──────────┴──────────┐
   │       │                                  │ OPTION A: direct    │
   │       │                                  │ browser fetch       │
   │       ▼                                  │ → BLOCKED by CORS   │
   │   localhost:relay/api/files/*            └─────────────────────┘
   │   (daemon mux, loopback trust)           ┌─────────────────────┐
   │                                          │ OPTION B (recommended)
   │                                          │ fetch via own daemon
   │                                          │ which proxies to    │
   │                                          │ remote webserver    │
   │                                          └──────────┬──────────┘
   │                                                     ▼
   │                                            local daemon socket
   │                                                     │
   │                                                     ▼
   │                                       (server-to-server HTTPS,
   │                                        TLS 1.2+, ?cap=<token>)
   │                                                     ▼
   │                                       Machine B webserver
   │                                       /api/files/* (requireFilesRead)
   │                                                     ▼
   │                                       Machine B daemon's filesHandler
   │                                                     ▼
   │                                       Sandbox + Machine B's session WorkDir
   │
   │
TUI (own process, no webview, no CORS)
   │
   ├── Local session                          Remote session
   │       │                                          │
   │       ▼                                          ▼
   │   m.client (Unix socket)                  remoteFilesClient
   │   ListFiles/StatFile/ReadFile/HeadFile    (HTTPS + cap, same 4 methods)
   │       │                                          │
   │       ▼                                          ▼
   │   loadDirCmd(m.client, sid, ...)          loadDirCmd(remoteClient, sid, ...)
   │       │                                          │
   │       └───────────┬──────────────────────────────┘
   │                   │
   │                   ▼
   │      Both honour same FilesClient interface
   │      (selected by `tabFiles` open dispatch)
```

### Component Responsibilities

| Component | File | Phase 122 Responsibility |
|-----------|------|--------------------------|
| `App.tsx` file-browser tab gate | `frontend/src/App.tsx:1181-1206` | Branch on local vs remote session; for remote, supply remote `baseURL` + `capToken`; for unshared remote, render "Enable web sharing" placeholder instead of `FileBrowserTab` |
| `handleOpenFileBrowser` | `frontend/src/App.tsx:924-940` | Accept (or detect) remote-session context; may need additional argument or look up from `remotePeers` |
| `FileBrowserTab` | `frontend/src/components/FileBrowserTab.tsx` | **No changes.** Already accepts `(baseURL, capToken?)` (lines 36-47) |
| `RemoteSessionsPanel` | `frontend/src/components/RemoteSessionsPanel.tsx` | Optional: add a "Browse files" action button alongside "Open Session" for web-shared remote sessions (UX trigger for Phase 122) |
| Daemon proxy route (NEW) | `internal/daemon/api.go` | NEW `GET /api/remote-files/*` that proxies a query to a remote tailnet peer's `/api/files/*`, injecting `?cap=` from a server-side store. **Only needed if OQ-1 resolves to "use daemon proxy"** |
| `internal/tui/files_cmds.go` | existing | Refactor: change first param from `*daemon.DaemonClient` to a new `FilesClient` interface |
| `internal/tui` new file `remote_files_client.go` | NEW | `RemoteFilesClient` struct implementing `FilesClient` using `net/http` + TLS 1.2+ + `?cap=` injection. Construction mirrors `internal/tailnet/sessions.go:38-72` (DNS-first, IP fallback) |
| `internal/tui/update.go` `FilesOpen` handler | lines 381-411 | Replace remote-toast branch with: if remote AND web-shared → build `RemoteFilesClient` from `entry.remote.URL` and a cap token, open `tabFiles`; if remote AND NOT web-shared → toast "Enable web sharing to browse"; if local → unchanged. **`m.client` field in Model becomes a per-tab selection** OR a new `m.filesClient FilesClient` field is set per-open |
| `Model.files` | `internal/tui/model.go:164` | Either extend with `filesClient FilesClient` or store baseURL+cap and have `files_cmds.go` construct the right client. Cleaner: store the interface on the model at `tabFiles` open time |

### Recommended Project Structure

```
internal/tui/
├── files.go                       # unchanged surface; just uses interface
├── files_cmds.go                  # signature change: takes FilesClient (interface)
├── remote_files_client.go         # NEW — HTTPS + cap implementation of FilesClient
├── files_client.go                # NEW — FilesClient interface declaration
└── update.go                      # FilesOpen handler updated

internal/daemon/
├── api.go                         # MAYBE — new proxy route for desktop GUI (see OQ-1)
└── client.go                      # if proxy: add ListRemoteFiles / etc. helpers

frontend/src/
├── App.tsx                        # branch tab gate on remote/local + cap source
├── components/
│   ├── RemoteSessionsPanel.tsx    # MAYBE — add "Browse files" button
│   └── FileBrowserTab.tsx         # NO CHANGES
└── lib/
    └── remoteSession.ts           # NEW — small helper deriving (baseURL, capToken) from a remote session entry
```

### Pattern 1: FilesClient interface (TUI side)

**What:** Narrow Go interface matching the four methods both `*daemon.DaemonClient` and a new `*RemoteFilesClient` need to expose.
**When to use:** Every `tea.Cmd` factory in `files_cmds.go`.
**Example:**
```go
// Source: derived from internal/daemon/client.go:381-484 signatures
package tui

import (
    "context"
    "time"

    "github.com/scottkw/agenthub/internal/daemon"
)

// FilesClient is the narrow interface used by the TUI Files view. Both the
// local *daemon.DaemonClient (Unix socket) and the new *RemoteFilesClient
// (HTTPS + cap) satisfy it. Phase 122.
type FilesClient interface {
    ListFiles(ctx context.Context, sessionID, relPath string) ([]daemon.FileEntry, bool, error)
    StatFile(ctx context.Context, sessionID, relPath string) (daemon.FileEntry, error)
    ReadFile(ctx context.Context, sessionID, relPath string) ([]byte, string, error)
    HeadFile(ctx context.Context, sessionID, relPath string) (int64, string, time.Time, error)
}

// Compile-time assertion *daemon.DaemonClient satisfies FilesClient.
var _ FilesClient = (*daemon.DaemonClient)(nil)
```

### Pattern 2: RemoteFilesClient (TUI side, HTTPS + cap)

**What:** Mirror `DaemonClient.filesURL` + the four methods, but with `https://{fqdn}:{port}` base + `?cap=` injected.
**When to use:** When `f` is pressed on a remote-session entry in the TUI.
**Example:**
```go
// Source: derived from internal/daemon/client.go:362-484 (DaemonClient.filesURL pattern)
// and internal/tailnet/sessions.go:38-72 (Tailscale TLS + IP fallback pattern)
package tui

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/scottkw/agenthub/internal/daemon"
    "github.com/scottkw/agenthub/internal/files"
)

type RemoteFilesClient struct {
    baseURL  string // e.g. https://hub-a.tailnet.ts.net:9443
    capToken string
    http     *http.Client
}

func NewRemoteFilesClient(baseURL, capToken string) *RemoteFilesClient {
    return &RemoteFilesClient{
        baseURL:  strings.TrimRight(baseURL, "/"),
        capToken: capToken,
        http: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
            },
        },
    }
}

func (c *RemoteFilesClient) filesURL(op, sessionID, relPath string) string {
    if relPath == "" {
        relPath = "."
    }
    q := url.Values{}
    q.Set("session", sessionID)
    q.Set("path", relPath)
    if c.capToken != "" {
        q.Set("cap", c.capToken)
    }
    return c.baseURL + "/api/files/" + op + "?" + q.Encode()
}

func (c *RemoteFilesClient) ListFiles(ctx context.Context, sid, rel string) ([]daemon.FileEntry, bool, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.filesURL("list", sid, rel), nil)
    if err != nil {
        return nil, false, fmt.Errorf("remote files list: %w", err)
    }
    resp, err := c.http.Do(req)
    if err != nil {
        return nil, false, fmt.Errorf("remote files list: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, false, fmt.Errorf("remote files list: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }
    var out files.FileListResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, false, fmt.Errorf("remote files list: decode: %w", err)
    }
    return out.Entries, out.Truncated, nil
}
// StatFile, ReadFile, HeadFile follow the exact same pattern — see
// internal/daemon/client.go:402-484 for the canonical shape.
```

### Pattern 3: App.tsx remote-vs-local branch

**What:** Branch on whether the active file-browser tab's session is local (in `panelSessions`) or remote (in `remotePeers`).
**When to use:** Single site at App.tsx:1181-1206.
**Example:**
```tsx
// Source: derived from frontend/src/App.tsx:1181-1206 (existing tab gate)
{activeId !== null && activeId.startsWith('__files__') && (() => {
  const fbSessionId = activeId.slice('__files__'.length)
  const fbLocalSession = panelSessions.find((s) => s.id === fbSessionId)
  const fbRemoteSession = !fbLocalSession
    ? remotePeers
        .flatMap((p) => p.sessions.map((s) => ({ ...s, hostname: p.hostname })))
        .find((s) => s.id === fbSessionId)
    : undefined

  if (fbRemoteSession) {
    // The remote session is in panelSessions ↔ web-shared. (See REMOTE-02.)
    // If we have no cap token at hand → render the "enable sharing" placeholder.
    // OQ-1: where does capToken come from? See Open Questions.
    const cap = lookupRemoteCap(fbRemoteSession.id) // TBD
    if (!cap) {
      return <EnableWebSharingPlaceholder hostname={fbRemoteSession.hostname} />
    }
    return (
      <FileBrowserTab
        sessionId={fbSessionId}
        sessionName={fbRemoteSession.name}
        isActive={true}
        isRemote={true}
        baseURL={remoteBaseURLFor(fbRemoteSession)} // e.g. strip /sessions/<id> from .url
        capToken={cap}
      />
    )
  }

  // Local-session path: unchanged (current behaviour)
  const isWeb = mode === 'web'
  const fbBaseURL = isWeb ? window.location.origin : `http://127.0.0.1:${relayPort ?? 0}`
  const fbCapToken: string | undefined = isWeb ? (webParams.capToken ?? undefined) : undefined
  return (
    <FileBrowserTab
      sessionId={fbSessionId}
      sessionName={fbLocalSession?.name || fbSessionId}
      isActive={true}
      isRemote={isWeb}
      baseURL={fbBaseURL}
      capToken={fbCapToken}
    />
  )
})()}
```

### Anti-Patterns to Avoid

- **Don't inline `session.hostname !== localHostname`-style remote detection.** Use the `panelSessions` (local) vs `remotePeers` (remote) distinction that already exists; if a session ID is in `remotePeers`, it's remote — period.
- **Don't have the Wails webview fetch the remote HTTPS URL directly without verifying CORS first.** The webserver emits NO `Access-Control-*` headers; cross-origin browser fetches will fail preflight. Either add CORS (security tradeoff — needs explicit decision) or proxy via the local daemon (recommended — server-to-server).
- **Don't extend `*daemon.DaemonClient` with a remote mode.** Single-responsibility: `DaemonClient` talks Unix socket. The TUI takes an interface.
- **Don't construct a fresh `RemoteFilesClient` per `tea.Cmd` invocation.** Construct once on `f` press, store on `Model.filesClient` (or `Model.files.client`), reuse across List/Stat/Read/Head dispatches.
- **Don't pull cap tokens from URLs into front-end state outside a single capture point.** The existing pattern (Phase 120-06 `webMode.readWebModeParams()`) reads once from URL params on mount; anything new should follow the same single-capture discipline.
- **Don't ignore the `generation` field in `filesListMsg`/`filesReadMsg`/`filesHeadMsg`.** Phase 121 introduced it as the WR-03 race guard; switching clients mid-flight (close tab, reopen against a different session) MUST bump `generation` so stale messages from the old client are discarded.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tailnet HTTPS client | New TLS / dial / cert handling | `internal/tailnet/sessions.go:38-72` pattern (TLS 1.2+, DNS-first, IP fallback w/ ServerName) | Already battle-tested; same code path used by `FetchPeerSessions` |
| URL query encoding | `fmt.Sprintf("?session=%s&path=%s&cap=%s", ...)` | `net/url.Values{}.Encode()` like `internal/daemon/client.go:370-373` | Caller-supplied paths may contain `#`, `?`, spaces — manual concat is a CVE waiting to happen |
| Cap-token plumbing in front-end | New global store / Redux-style slice | Single-pass URL-param capture (Phase 120-06 `webMode.ts` pattern) or per-tab prop | Two existing patterns already cover the in-app cap shape; introducing a third pattern fragments the contract |
| Browser CORS workaround | `mode: 'no-cors'` fetch (returns opaque response) or `Authorization: Bearer` cap | Daemon proxy or proper CORS allowlist | Opaque responses can't be read; `Authorization` header changes the security model and triggers preflight |
| Remote URL parsing | String surgery on `https://host:port/sessions/<id>` | `new URL(remoteSession.url).origin` | RemoteSession URL shape is `https://{fqdn}:{port}/sessions/{id}` — drop the path to get the base |

**Key insight:** every wire-level concern in this phase has a prior-art pattern. Phase 122 should bias hard toward "copy the pattern, don't invent a new one."

## Runtime State Inventory

Phase 122 is a wiring phase, not a rename/migration. The category below is included for completeness only — there is nothing to migrate.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — verified by grep for "remote.*cap" / "remote.*token" in `internal/` and `frontend/`; no persisted state introduced | None |
| Live service config | None — no n8n / Datadog / Tailscale ACL changes | None |
| OS-registered state | None | None |
| Secrets/env vars | None — cap tokens are runtime-only (in-memory `JoinCodeManager`); no env-var consumers | None |
| Build artifacts | Wails desktop build must include the new App.tsx render branch; tags `wailsassets` already required (per project memory) | Standard `pnpm build && go build -tags wailsassets ./...` |

**Nothing found in any category.** Phase 122 is pure additive wiring.

## Common Pitfalls

### Pitfall 1: Wails webview CORS preflight blocks direct cross-origin fetch

**What goes wrong:** Desktop GUI loaded under `wails://wails` (or `http://localhost:34115` in dev) issues `fetch('https://hub-a.tailnet.ts.net:9443/api/files/list?cap=...')`. Browser issues OPTIONS preflight; webserver returns no `Access-Control-Allow-Origin`; browser blocks the response.
**Why it happens:** `internal/webserver/server.go` emits no CORS headers. Only `/sessions/{id}/ws` has an Origin allowlist (`requireAllowedOrigin`, line 520). The web-share viewer DOM path works because the React shell is **served from the same origin** as the API (see Phase 120-06 `vite.config.ts` `base: './'` + `/app/` mount).
**How to avoid:** Proxy via local daemon. Add a new daemon route (e.g. `GET /api/remote-files/{op}?peer=<fqdn>&session=<id>&path=<rel>`) that injects the cap server-side and forwards to `https://{peer}:{port}/api/files/{op}`. Reuse the `internal/tailnet/` TLS client pattern (TLS 1.2+, DNS-first with IP fallback).
**Warning signs:** Browser DevTools shows "blocked by CORS policy" on the file list fetch; the fetch resolves with status 0 or `TypeError: Failed to fetch`.

### Pitfall 2: Remote session listed but NOT actually web-shared (Phase 119 WEB-04 nuance)

**What goes wrong:** `internal/tailnet/FetchPeerSessions` returns a session, but the peer's webserver has since toggled it off; `/api/files/*` returns 401 (no grant) or 404.
**Why it happens:** The peer's `/api/sessions` listing is cached/eventually-consistent and the user-facing distinction is "session is currently web-shared". The TUI's `remoteSessions` is refreshed on a poll cadence; race exists.
**How to avoid:** On `f` press for a remote session, **probe with a HEAD or GET on `/api/files/list?session=<id>&path=.`** before opening `tabFiles`. If 401 → "Enable web sharing to browse"; if 403 → "Missing files.read permission" (UI matches Phase 120-04 `PermissionDeniedTakeover`); if 200 → open the tab.
**Warning signs:** Tab opens, fires the first ListFiles, gets 401/403, dispatches the existing error states. The UX is "open then immediately fail" — better to gate at open time.

### Pitfall 3: `m.client` is concrete `*daemon.DaemonClient`, not interface — refactor must touch all sites

**What goes wrong:** Search-and-replace `m.client` → `m.filesClient` partially; some code paths still use `m.client` for Health / ListSessions / etc.
**Why it happens:** `Model.client` (`internal/tui/model.go:105`) is used for ALL daemon-socket operations, not just files. Files is the FIRST surface to need polymorphism.
**How to avoid:** Don't replace `m.client`. ADD `m.files.client FilesClient` (interface field on `filesModel`) that is set in `newFilesModel(...)` or in the `tabFiles` open dispatch. Cmd factories take the interface, not the model. `m.client` keeps doing all the non-files daemon work.
**Warning signs:** Compile errors in `tui_test.go` `testModel()` — many tests inject a nil `*daemon.DaemonClient`. Interface-typed nil is still nil; the existing `errNilClient` guard in `files_cmds.go:13-16` still works as long as the test path's filesModel has nil client.

### Pitfall 4: Cap-token cross-machine acquisition has no current in-app path

**What goes wrong:** Desktop GUI / TUI on Machine B wants to browse Machine A's session files. The only way Machine B currently obtains a Machine A cap is the join-code flow (`https://machine-a:9443/join?code=<code>`), which is a BROWSER flow — the cap ends up in the browser as a URL param, not in any Wails / TUI accessible store.
**Why it happens:** Phase 87 designed the join-code flow for browser sharing only. There's no `DaemonClient.RegisterRemoteCap(peerFQDN, sessionID, cap)` API.
**How to avoid:** This is the load-bearing OQ-1 below. Three plausible paths:
  - **(a)** Add a "Browse files" action button to RemoteSessionsPanel that prompts the user for a join code (or accepts a paste of the full `?cap=` URL), exchanges it, persists the cap in the daemon's grant store, and opens FileBrowserTab.
  - **(b)** Owner-machine GUI generates a remote-share invite that includes both code AND `peer-fqdn`, opening the cap directly into the receiving daemon's grant store via a tailnet message.
  - **(c)** Re-interpret "remote session" as "session displayed in the Remote panel that belongs to this same machine" — but that's empty by definition.
**Warning signs:** Planning hits "where does the cap come from?" with no obvious answer. Demand a CONTEXT/discuss-phase user decision before proceeding to plan tasks 1-2.

### Pitfall 5: `tabFiles` reset semantics with a remote-flavoured client

**What goes wrong:** User presses `f` on Local Session A → opens `tabFiles` with `DaemonClient`. User presses `f` again on Remote Session B → `filesModel` is reset (Pitfall TUI-PITFALL-6) but the in-flight Cmd from A may still resolve. The new Cmd uses the new client (`RemoteFilesClient`) but the response was generated by the old client.
**Why it happens:** Generation counter (Phase 121 WR-03) discards stale messages by sessionID + generation. As long as the new model resets generation AND captures the new sessionID, the existing guards hold.
**How to avoid:** When constructing the new `filesModel`, store the client choice **on the model** (so subsequent Cmd dispatches use the right client) AND bump `generation`. The existing `newFilesModel(...)` already resets everything to defaults; extend it to take a `FilesClient` parameter.
**Warning signs:** Test `TestFiles_OpenFromSessions_ResetsModel` (Phase 121-01) starts failing intermittently. Add a sibling test `TestFiles_OpenFromSessions_SwitchesClient` to lock the contract.

### Pitfall 6: 401 from missing cap vs 403 from missing perm — different UX needs

**What goes wrong:** Phase 119 returns 401 for missing/bad cap, 403 for present-but-missing-perm. Desktop GUI's existing `PermissionDeniedTakeover` (Phase 120-04) is designed for 403 with `"files.read"` in body. A 401 (no cap at all) renders as `NetworkErrorState` — wrong UX.
**Why it happens:** Phase 122 introduces a new failure mode (cap absent because user hasn't joined yet) that Phase 120 didn't anticipate.
**How to avoid:** In `FileBrowserTab`'s error switch (`frontend/src/components/FileBrowserTab.tsx` — uses `FilesApiError.isUnauthorized()`), add a new takeover variant `EnableWebSharingTakeover` for 401-when-isRemote, distinct from `PermissionDenied` (403) and `NetworkError` (other).
**Warning signs:** Vitest Playwright snapshot mismatch on `files-browser.spec.ts` cells if a new state is added without updating the test matrix. Update the test matrix in the same PR.

### Pitfall 7: TUI status line doesn't update when remote network is slow

**What goes wrong:** `loadDirCmd` uses 5s timeout (`files_cmds.go:71`); `readFileCmd` uses 10s (line 92). On a slow tailnet link these timeouts can fire BEFORE the user has any visual feedback (status line just says "Loading…").
**Why it happens:** Phase 121 sized the timeouts for daemon-socket latency, not tailnet RTT.
**How to avoid:** Either bump timeouts to 15s / 30s when using the remote client, OR add a separate "Connecting to {peer}…" status line state. Bias toward the simpler longer-timeout approach unless UAT shows otherwise.
**Warning signs:** User reports "Files view shows error immediately on remote" — verify by `TEMPSLOW=1 go test ./internal/tui/...` (if such a test util exists; Phase 121 docs don't suggest one).

## Code Examples

### Common Operation 1: Open remote session in TUI

```go
// Source: derived from internal/tui/update.go:381-411 + this phase's RemoteFilesClient
case key.Matches(msg, m.keys.FilesOpen):
    if len(m.unifiedList) == 0 {
        return m, nil
    }
    entry := m.unifiedList[m.selected]

    switch entry.kind {
    case entryLocal:
        // Existing path — unchanged.
        sid := entry.session.ID
        // ... dimension math elided (see existing code lines 397-409) ...
        m.files = newFilesModelWithClient(sid, m.client, listW, paneH-2, previewW, paneH-2)
        m.openTab(tabFiles)
        return m, loadDirCmd(m.files.client, sid, ".", m.files.generation)

    case entryRemote:
        if entry.remote == nil {
            return m, nil
        }
        // Derive base URL (drop /sessions/{id} suffix); cap comes from OQ-1 source.
        baseURL := remoteBaseURLFromSessionURL(entry.remote.URL) // helper in files_client.go
        cap := m.remoteCapStore[entry.remote.ID]                  // TBD by OQ-1
        if cap == "" {
            m.toast = "Enable web sharing on the remote machine to browse files"
            m.toastKind = toastInfo
            m.toastExp = time.Now().Add(3 * time.Second)
            return m, nil
        }
        sid := entry.remote.ID
        client := NewRemoteFilesClient(baseURL, cap)
        // ... same dimension math ...
        m.files = newFilesModelWithClient(sid, client, listW, paneH-2, previewW, paneH-2)
        m.openTab(tabFiles)
        return m, loadDirCmd(m.files.client, sid, ".", m.files.generation)
    }
    return m, nil
```

### Common Operation 2: Daemon-proxy route (if OQ-1 picks the proxy approach)

```go
// Source: derived from internal/daemon/api.go pattern (registerRoutes + a single handler)
// Phase 122 — proxies a query to a remote peer's /api/files/*.
//
// Trust model: trusts the caller (loopback Unix socket); injects cap from
// in-memory store keyed by (peerFQDN, sessionID); does NOT expose the cap
// to the caller.
a.mux.HandleFunc("GET /api/remote-files/list", a.handleRemoteFilesList)
a.mux.HandleFunc("GET /api/remote-files/stat", a.handleRemoteFilesStat)
a.mux.HandleFunc("GET /api/remote-files/read", a.handleRemoteFilesRead)
a.mux.HandleFunc("HEAD /api/remote-files/read", a.handleRemoteFilesRead)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Loopback-only file API | Webserver mount under `requireFilesRead` (cap-gated) | Phase 119 (2026-05-20) | Remote browse is now wire-possible; Phase 122 wires the consumers |
| TUI Files view = local only (toasts on remote) | TUI Files view = local + remote with shared interface | Phase 122 (this) | Cross-surface parity (REMOTE-04) |
| Desktop GUI file-browser hard-codes loopback baseURL | Per-tab `(baseURL, capToken)` from session metadata | Phase 122 (this) | REMOTE-01, REMOTE-02 |
| Web-share viewer parity gap (deferred to v3.5) | Already closed by Phase 120-06 (web-mode DOM scenarios) | Phase 120-06 (2026-05-20) | The "/app/?session=…&cap=…" URL pattern is already proven in Playwright; Phase 122's GUI work mirrors it |

**Deprecated/outdated:**
- The Phase 120-04 comment at App.tsx:1176-1180 stating "remote-on-desktop browse… is a v3.5 follow-on — out of scope for the v3.4 parity closure" — Phase 122 removes this scope deferral. The inline note should be cleaned up in this phase.
- The TUI toast at update.go:388 — `"File browser not available for remote sessions in v3.4"` — must be deleted.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Wails webview enforces CORS the same way Chrome does on tailnet-cross-origin fetches | Pitfall 1, Architecture diagram | If Wails has a CORS bypass (sometimes the case for `wails://` schemes), direct fetch may work and the daemon proxy is unnecessary. Verify with a probe before implementing the proxy |
| A2 | Cross-machine cap acquisition has no in-app path today (only browser-driven join-code flow) | OQ-1, Pitfall 4 | If a path exists that this research missed, the proxy / new UI is unnecessary. Confidence: HIGH based on grep of all `joinCode` / `JoinCode` / `ExchangeJoinCode` callers — but flagging as assumption |
| A3 | `RemoteSession.URL` shape is consistently `https://{fqdn}:{port}/sessions/{id}` across all peer-listing paths | App.tsx tab gate snippet, Pattern 2 | If older daemons emit a different shape, base-URL derivation breaks. Mitigation: parse with `new URL(...)` not regex |
| A4 | A remote session appearing in `remotePeers` ↔ it is web-shared on the remote machine | REMOTE-02 detection logic | Phase 119 WEB-04 architecture suggests this is guaranteed, but a stale poll could show a session that was toggled off seconds ago. Mitigation: probe-on-open per Pitfall 2 |
| A5 | Owner cap (`writeURL` from `IssueCapabilities`) ALWAYS includes `files.read` when `filesReadEnabled()` is true, which defaults to `nil`-means-`true` | Background, Pattern 3 | Verified by Phase 118-05 plan summary — `ownerPerms = "read,write," + capability.PermFilesRead` when filesReadEnabled. Settings opt-out is the only false-path. Confidence: HIGH |
| A6 | `internal/tui` test suite uses nil `*daemon.DaemonClient` injection (existing `errNilClient` pattern) | Pitfall 3 | If tests already inject a fake, refactor is more involved. Phase 121-01 summary line 159: `errNilClient` is returned by all three factories when client == nil. Confidence: HIGH |

## Open Questions

### OQ-1 (LOAD-BEARING): How does the desktop GUI / TUI on Machine B acquire a cap for Machine A's session?

**What we know:**
- The remote-session URL exposed in `RemoteSessionsPanel` and `RemoteSessionEntry.URL` is `https://{fqdn}:{port}/sessions/{id}` with **NO** `?cap=` query param.
- The webserver's `requireFilesRead` middleware rejects requests without a cap with 401.
- The owner machine's `IssueCapabilities` mints `readURL` (no `files.read`) and `writeURL` (has `files.read` when enabled). These caps are only readable on the owning machine.
- The only documented cross-machine cap-transfer path is the browser `https://machine-a:9443/join?code=<5-char-code>` flow — which leaves the cap in the BROWSER's URL bar after exchange (D-09 invariant).

**What's unclear:** The CONTEXT.md says "reuse the session's existing web-share cap" — but on the consuming machine (Machine B), no such cap exists in any in-app store today.

**Three plausible designs (planner needs user input or must make a call):**

1. **(P1)** Reuse the existing join-code browser flow but extend the daemon to accept a paste-or-prompt-entered code/URL, exchange it, and store the resulting cap in a new `remoteCapStore` field. UX: RemoteSessionsPanel grows a "Browse files…" action that opens a modal asking for the join code.
2. **(P2)** Tailnet trust: Machine A's daemon, upon detecting a tailnet peer probing `/api/sessions`, auto-issues a `files.read`-bearing cap and returns it as an additional field in the listing JSON. UX: zero-click — Browse files just works for any web-shared remote session. **Loosens the cap model significantly** — every tailnet peer becomes an implicit `files.read` holder.
3. **(P3)** Restrict scope: Phase 122's "remote" is interpreted as "owner browses their own web-shared session via the webserver path instead of the daemon socket path" — i.e., the Phase 120-04 stalled work, not actual cross-machine. Trivial to wire; matches the CONTEXT's "no new cap-minting flow" verbatim. **But** it doesn't deliver REMOTE-05 cross-surface parity for the cross-machine case.

**Recommendation:** Flag this for `discuss-phase` regeneration if `skip_discuss` is left on. P1 is the lowest-risk full-feature path; P3 is the smallest delivery that closes the verbatim CONTEXT decisions. P2 is a security-model change that warrants its own phase.

### OQ-2: Does the desktop GUI proxy via local daemon, or does it use Wails' webview-host bridge to bypass CORS?

**What we know:**
- The webserver emits no CORS headers (verified by grep).
- Wails' default scheme is `wails://wails` (or in dev `http://localhost:34115`). Cross-origin behavior is webview-engine-dependent (WebKit on macOS, WebView2/Chromium on Windows, WebKitGTK on Linux).
- Phase 120-06 explicitly states "remote-on-desktop browse remains a v3.5 follow-on" — meaning the desktop GUI never actually tried fetching a remote URL during Phase 120's verification.

**What's unclear:** Whether Wails webviews universally enforce CORS for `https://*.tailnet.ts.net/` cross-origin fetches, or whether one or more platforms permit it.

**Recommendation:** Plan a small Wave 0 probe task: spawn a remote-flavoured `FileBrowserTab` against a known web-shared session and observe whether `fetch('https://<peer>/api/files/list?...')` succeeds. If yes on all three platforms → no daemon proxy needed. If any one platform blocks → proxy required (and the proxy path should be the uniform implementation for portability).

### OQ-3: Does the TUI need a fallback path for IP-only access (when Tailscale MagicDNS is unavailable)?

**What we know:**
- `internal/tailnet/sessions.go:48-62` already implements a DNS-first → IP-fallback pattern with TLS ServerName override.
- The TUI's existing `fetchRemoteFn` (via `cmd_tui.go` → `tailnet.FetchAllPeerSessions`) inherits this fallback.

**What's unclear:** When `RemoteFilesClient` is constructed from `entry.remote.URL` (already DNS-resolved at listing time), does it need the same IP-fallback logic for the actual `/api/files/*` requests, or does the prior successful listing guarantee DNS will keep resolving?

**Recommendation:** Start with DNS-only `RemoteFilesClient` (simpler, matches `DaemonClient`'s shape). If UAT reports failures in DNS-flaky environments, add IP fallback as a follow-on. Track as a Phase 122-VERIFICATION watch item.

## Environment Availability

Skipped — Phase 122 has no external tool dependencies beyond the existing Go toolchain and Node/pnpm environments already required by prior phases.

## Validation Architecture

Nyquist validation is enabled (`workflow.nyquist_validation` is not explicitly false in config.json — treat as enabled).

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (TUI / daemon); Vitest 3.x (frontend unit); Playwright 1.59.1 (e2e) |
| Config file | `go test ./...` (root go.mod); `frontend/vitest.config.ts`; `frontend/playwright.config.ts` |
| Quick run command | `go test ./internal/tui/ ./internal/daemon/ -count=1 -short` + `pnpm --filter frontend test -- --run` |
| Full suite command | `go test ./... -count=1 && pnpm --filter frontend test -- --run && pnpm --filter frontend exec playwright test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REMOTE-01 | Desktop GUI opens FileBrowserTab against remote session with cap | unit + e2e | `pnpm test -- frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx --run` + `playwright test files-browser.spec.ts --grep "remote-session"` | ❌ Wave 0 |
| REMOTE-02 | NOT web-shared remote → "Enable web sharing" placeholder | unit | `pnpm test -- frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx --run` | ❌ Wave 0 |
| REMOTE-03 | TUI remote session opens via HTTPS + cap; toast removed | unit (Go) | `go test ./internal/tui/ -run 'TestFiles_OpenFromSessions_Remote' -count=1` | ❌ Wave 0 |
| REMOTE-04 | TUI behavior identical local vs remote | integration (Go) | `go test ./internal/tui/ -run 'TestFiles_Remote_BehaviorParity' -count=1` | ❌ Wave 0 |
| REMOTE-05 | Cross-surface parity (GUI/web/TUI) | integration | Combination of REMOTE-01..04 commands + new Playwright scenario | ❌ Wave 0 |
| REMOTE-03 (negative) | "File browser not available" toast NO LONGER appears | source-grep guard | `grep -c "File browser not available for remote sessions" internal/tui/update.go` == 0 | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/tui/ ./internal/daemon/ -short && pnpm --filter frontend test -- --run`
- **Per wave merge:** Full Go test suite + Vitest + the new Playwright remote-session scenario
- **Phase gate:** `go test ./... -race -count=1 && pnpm --filter frontend test -- --run && pnpm --filter frontend exec playwright test` all green

### Wave 0 Gaps

- [ ] `internal/tui/files_client.go` — new `FilesClient` interface declaration (no test; compile-time guard via `var _ FilesClient = ...`)
- [ ] `internal/tui/remote_files_client_test.go` — new tests for `RemoteFilesClient` against a `httptest.NewTLSServer` mocking the webserver routes
- [ ] `internal/tui/files_test.go` — new `TestFiles_OpenFromSessions_RemoteEntry_OpensTabWithRemoteClient` (replaces existing `_RemoteEntry_ShowsToast` test or supersedes it)
- [ ] `frontend/src/components/__tests__/App.remoteFileBrowser.test.tsx` — new unit suite for the App.tsx remote branch
- [ ] `frontend/e2e/files-browser.spec.ts` — new scenario "remote-session browse via owner cap" (scenarios 16+ in the existing matrix)
- [ ] Source-grep guard test: `grep -c "File browser not available for remote sessions"` must equal 0 in `internal/tui/update.go`

## Security Domain

`security_enforcement` is enabled (no explicit `false`). Phase 122 introduces no new HTTP routes if OQ-2 resolves to "no daemon proxy needed"; introduces ONE new daemon route (loopback-trusted) otherwise.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Cap-token signature verified by `requireFilesRead` middleware (unchanged) |
| V3 Session Management | yes | Cap tokens carry session SID; Phase 118 grant model enforces SID-match (unchanged) |
| V4 Access Control | yes | `HasPerm(claims.Perms, "files.read")` whole-token match (Phase 118, unchanged) |
| V5 Input Validation | yes | Sandbox path validation via `os.OpenInRoot` (Phase 118, unchanged); URL-encoded path via `url.Values{}.Encode()` (mirror existing client.go pattern) |
| V6 Cryptography | yes | TLS 1.2+ for tailnet HTTPS; `tls.Config{MinVersion: tls.VersionTLS12}` matches `internal/tailnet/sessions.go:43` |
| V13 API Security | yes | Same `/api/files/*` contract; no new public surface (the only NEW route, if added per OQ-2, is loopback-only) |

### Known Threat Patterns for Wails + Go HTTP cross-network file fetch

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cap-token leak via window.location.href / referrer | Information disclosure | Pattern from Phase 120-04 / 120-06: cap is in query param, but webserver pages emit no outbound links that would leak Referer. Maintain by NOT adding new `<a>` elements that point cross-origin from the file browser tab |
| Cap-token leak via in-process logs / panic stacks | Information disclosure | Phase 121 already redacts in error strings (`files list: %d %s` shows status + body, not URL). Match this discipline in `RemoteFilesClient` errors |
| Replay of expired cap | Spoofing | HMAC + TTL already enforced by `requireCapability`'s underlying verify. Phase 122 introduces no new token validation surface |
| MITM on tailnet TLS | Tampering | Tailscale MagicDNS uses pinned LetsEncrypt certs over WireGuard transport; TLS 1.2+ on top is defense-in-depth (unchanged) |
| Cap-token persisted to disk inadvertently | Information disclosure | If OQ-1 picks P1 (persist remote caps in daemon), the cap must live in-memory only (mirror `JoinCodeManager`). NO disk persistence; restart wipes the cache. Document this invariant explicitly in the planner |
| 401 response distinguishes route-existence | Information disclosure | Phase 119 already returns 401 (not 404) for missing cap on `/api/files/*`. Maintained by Phase 122 (no route shape change) |

## Sources

### Primary (HIGH confidence)

- `internal/files/types.go` — wire shape (FileEntry, FileListResponse) — confirmed by Phase 118-05 summary
- `internal/daemon/client.go:362-484` — DaemonClient files methods + filesURL helper — pattern source for RemoteFilesClient
- `internal/daemon/api.go:970-1038` — issueCapabilitiesForSession — confirms owner-cap shape includes files.read
- `internal/webserver/server.go:485-505` — file route mount + filesDispatch + nil-handler guard — confirms no CORS, no Origin allowlist on `/api/files/*`
- `internal/tailnet/sessions.go:38-72` — TLS 1.2+ HTTPS pattern with DNS→IP fallback — pattern source for tailnet HTTP client
- `internal/tui/files_cmds.go` (all) — current concrete-type signatures requiring interface refactor
- `internal/tui/update.go:381-411` — current `FilesOpen` dispatch with remote toast
- `internal/tui/model.go:30-101` — listEntry, RemoteSessionEntry shapes
- `frontend/src/App.tsx:1181-1206` — current file-browser tab gate (the load-bearing site to edit)
- `frontend/src/components/FileBrowserTab.tsx:36-115` — `FileBrowserTabProps` (already accepts baseURL + capToken)
- `frontend/src/lib/filesApi.ts` — FilesApiClient (already supports capToken via constructor)
- `frontend/src/lib/webMode.ts` (referenced via Phase 120-06 summary) — single-source-of-truth pattern for mode detection
- `frontend/src/components/RemoteSessionsPanel.tsx` — RemoteSession / RemotePeerSessions shapes
- `.planning/phases/120-filebrowsertab-tsx-desktop-web/120-04-SUMMARY.md` (decisions §, "Pragmatic capToken handling") — confirms the v3.5 deferral now being closed
- `.planning/phases/120-filebrowsertab-tsx-desktop-web/120-06-SUMMARY.md` (Architectural Notes for v3.5) — explicit recipe for "Remote-on-desktop browse is now isolated, not entangled"
- `.planning/phases/121-tui-files-view/121-01-SUMMARY.md` + `121-02-SUMMARY.md` — TUI internals, WR-03 generation guard, T-121-04 stale-msg pattern
- `.planning/phases/118-fs-sandbox-core-workdir-gap-daemon-routes-fuzz-corpus-capabi/118-05-SUMMARY.md` — owner-cap composition + filesReadEnabled gate
- `.planning/phases/119-webserver-routes-files-read-capability-plumbing/119-01-SUMMARY.md` — webserver mount + threat model
- `.planning/REQUIREMENTS.md` lines 181-194 — REMOTE-01..05 verbatim
- `.planning/ROADMAP.md` Phase 122 section (lines 412-432) — goal + success criteria

### Secondary (MEDIUM confidence)

- `frontend/src/components/SessionSharePanel.tsx:49-55` — joinURLFor pattern showing `?cap=<token>` query shape on `/sessions/<id>` URLs (mirror for derivation)
- `frontend/src/components/DaemonManagerPanel.tsx:78-135` — IssueCapabilities reconcile pattern (proves the owner-side flow that mints `files.read`-bearing caps)

### Tertiary (none required)

No web-only sources — every claim verifiable against the codebase or planning docs.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — all dependencies already in go.mod / package.json; patterns already used in adjacent files
- Architecture: HIGH for the local code shape; MEDIUM for the cross-machine cap-acquisition flow (load-bearing OQ-1)
- Pitfalls: HIGH — Pitfalls 1-3 verified against source; Pitfalls 4-7 derived from architecture analysis with at least one concrete reference each

**Research date:** 2026-05-21
**Valid until:** 2026-06-20 (30 days — stable area; only invalidator is if Phase 118/119 contracts change before plan begins)

---

## RESEARCH COMPLETE
