# Phase 119: WebServer Routes + `files.read` Capability Plumbing - Research

**Researched:** 2026-05-20
**Domain:** Go HTTP mux route mounting, capability middleware composition, cross-mux handler injection
**Confidence:** HIGH (all claims verified against actual source files in repo)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices at Claude's discretion. Use Phase 118 outputs:
- `files.NewHandler(resolver)` factory — accepts `func(sessionID string) (*Sandbox, error)`
- `capability.PermFilesRead` constant + `HasPerm` whole-token helper
- `requireFilesRead` middleware wrapper body in `internal/webserver/capability_mw.go` (Phase 118 wrote the body; Phase 119 MOUNTS it on webserver routes)
- `internal/daemon/engine.GetSessionWorkDir` for resolver implementation

### Claude's Discretion
Everything — discuss phase skipped. All architecture decisions for the webserver-side mounting, provider injection point, test layout, and CSP regression coverage are at Claude's discretion within the constraints of the Phase 118 outputs.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WEB-01 | Daemon's local-socket HTTP API exposes `/api/files/list`, `/stat`, `/read` (GET + HEAD) for in-process GUI/TUI/CLI; no cap-token middleware on this surface. | **Already shipped in Phase 118** (`internal/daemon/api.go:131-134`). Phase 119 must not regress this. Add a regression test pinning the daemon-socket surface stays auth-less. |
| WEB-02 | Webserver mux exposes the same three endpoints under `/api/files/...` wrapped by `requireFilesRead`; routes are mounted via `SetFilesHandlerProvider` (no direct coupling between `internal/webserver` and `internal/files/`). | Mounting pattern: §"Pattern 1: Provider Injection + Capability-Wrapped Route Registration". Note: `internal/webserver/capability_mw.go` already references `SetFilesHandlerProvider` in comments — the symbol does NOT yet exist and must be created in this phase. |
| WEB-03 | Read-only web-share viewer cannot use file browser endpoints: an explicit integration test asserts 403 with a viewer cap token across all three endpoints + both methods on `/read`. | Test pattern in `capability_test.go:TestRequireFilesRead` already exists for the wrapper standalone. Phase 119 adds the **mounted-route** version against the real webserver mux. See §"Test Patterns to Follow". |
| WEB-04 | Web-shared file browser works against tailnet-remote sessions via Tailscale HTTPS (not relay frames); frontend uses `fetch()` against remote peer's HTTPS base URL. | This is a **frontend** requirement satisfied transparently if WEB-02 mounts work — the routes exist on every webserver instance (one per agenthub daemon, including tailnet peers). No webserver-side code required for WEB-04 specifically; verify by integration test that all 4 endpoint shapes are reachable from a non-loopback Origin (Origin allowlist already permissive for `/api/*`). |
| WEB-05 | Zero new CSP amendments. Cross-browser Playwright e2e (Chromium + Firefox + WebKit) reports zero CSP violations from file browser flows. | CSP middleware (`csp_mw.go`) is mounted only on HTML-serving routes (`/dashboard`, `/join`, `/sessions/{id}`). `/api/files/*` returns JSON/octet-stream — CSP headers are irrelevant. The "zero new violations" gate is about not breaking *terminal/dashboard* CSP. Phase 119 should add a smoke test confirming no `/api/files/*` route accidentally serves HTML, and adds no `<script>`/`<style>` injection vector. See §"CSP Impact Analysis". |
</phase_requirements>

## Summary

Phase 118 left the webserver one step short of serving files: it shipped the `requireFilesRead` middleware wrapper body in `internal/webserver/capability_mw.go` and the `*files.Handler` stateless HTTP layer in `internal/files/`, but the webserver mux has no file routes mounted yet. Phase 119 is a small, mechanical phase: define a `SetFilesHandlerProvider` injection point on `WebServer`, mount three GET routes and one HEAD route under `requireFilesRead`, wire the provider from the daemon (`AutoStartWebServer` + `handleWebServerStart` — both webserver-construction sites), and write integration tests that exercise the full mounted stack with real cap tokens.

The architectural pattern is already proven twice in the codebase: `SetSessionResolver` (function injection for session metadata) and `SetPluginSettingsProvider` (function injection returning pre-marshaled JSON, deliberately `func() []byte` to avoid the `daemon→webserver→daemon` circular import). Phase 119 follows `SetSessionResolver` directly — the provider returns a `*files.Handler`, not raw bytes, because `internal/files` is already a leaf package importable by both `internal/daemon` and `internal/webserver`.

The five success criteria collapse to: (1) mount 4 routes with explicit method prefixes under `requireFilesRead`; (2) wire the provider at both construction sites; (3) integration test owner-200, viewer-403, missing-cap-401, POST-405; (4) regression test that daemon-socket file routes remain auth-less (WEB-01); (5) confirm no CSP regression by smoke-running the existing CSP e2e suite and asserting `/api/files/*` does not appear in any HTML page.

**Primary recommendation:** Mirror the `SetSessionResolver` pattern. Add `SetFilesHandler(*files.Handler)` (passing a constructed Handler rather than a constructor func — the Handler is stateless and shared between daemon and webserver mux). Mount four routes in `setupRoutes()` with explicit method prefixes. Wire the same `a.filesHandler` already constructed in `NewAPI` at both webserver construction sites (`AutoStartWebServer` and `handleWebServerStart`). Add a `files_routes_test.go` integration test mirroring `capability_test.go:TestRequireFilesRead` but against the real mounted routes.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Daemon-socket file routes (WEB-01) | Daemon API mux | — | Loopback socket is the trust boundary; already shipped Phase 118 |
| Tailscale-HTTPS file routes (WEB-02) | Webserver mux | — | Capability-gated public surface; this phase's main work |
| Cap token issuance with `files.read` bit | Daemon API (`issueCapabilitiesForSession`) | — | Already shipped Phase 118 — `engine.filesReadEnabled()` already feeds owner perms |
| `requireFilesRead` middleware body | Webserver (`capability_mw.go`) | — | Already shipped Phase 118 (body only) |
| `*files.Handler` HTTP handler logic | `internal/files/` (leaf package) | — | Already shipped Phase 118; reused unchanged |
| Sandbox/session resolver | Daemon (`engine.GetSessionWorkDir` + `files.NewSandbox`) | — | Already shipped Phase 118; same closure reused for webserver injection |
| CSP policy (WEB-05) | Webserver (`csp_mw.go`) | — | Unchanged — file routes do not serve HTML, so CSP middleware doesn't gate them |
| Provider injection point (`SetFilesHandlerProvider`) | Webserver (new in this phase) | — | Follows `SetSessionResolver` precedent |
| Playwright CSP e2e (WEB-05) | `frontend/e2e/` | webserver (test fixture) | Spec runs against real browser via `cmd/playwright-fixture` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` stdlib | Go 1.26.1 | HTTP mux + method-prefix routing | Already in use throughout webserver; Go 1.22+ method-prefix mux is the project standard (cf. `api.go:registerRoutes`, `server.go:setupRoutes`) [VERIFIED: source] |
| `internal/files` | local | `*Handler` providing `List`/`Stat`/`Read` methods | Already shipped Phase 118 [VERIFIED: source] |
| `internal/capability` | local | `requireFilesRead` + `HasPerm` + `PermFilesRead` | Already shipped Phase 118 [VERIFIED: source] |
| `github.com/coder/websocket` | (existing) | Unaffected by this phase | — |
| `github.com/chromedp/chromedp` | v0.15.1 | Optional in-tree browser CSP e2e (build-tagged `e2e`) | Already used by `browser_csp_e2e_test.go` [VERIFIED: go.mod] |
| `@playwright/test` | (existing in `frontend/`) | Cross-browser CSP e2e for WEB-05 | Already used by `frontend/e2e/web-csp.spec.ts` [VERIFIED: source] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `httptest` stdlib | Go 1.26.1 | In-process request/response testing | All `internal/webserver/*_test.go` use this; standard pattern |
| `selfSignedTLSForTest` (test helper) | local | TLS-aware test client | Use `testServer(t)` from `capability_test_helpers.go` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `SetFilesHandler(*files.Handler)` direct injection | `SetFilesHandlerProvider(func(sessionID string) *files.Handler)` | The provider-per-request form is over-flexible: `*files.Handler` is stateless, the same instance serves every session via the resolver closure inside the Handler. Passing one Handler at construction time is simpler and matches how `SetSessionResolver` injects one callback. **Recommendation: use the direct-handler form**, but the comment in `capability_mw.go:99` and `files/handler.go:6` mentions `SetFilesHandlerProvider` — those comments should be updated to match. |
| Mount routes via injected `http.Handler` | Mount with `ws.requireFilesRead(handler)` wrapping the Handler methods directly | The Handler exposes methods (`.List`, `.Stat`, `.Read`) that are `func(http.ResponseWriter, *http.Request)` — they slot straight into `mux.HandleFunc` after `requireFilesRead` wrapping. No `http.Handler`-interface adapter needed. |
| Per-request resolver | Resolver closure baked into the single `*files.Handler` | Already decided in Phase 118 — the daemon constructs one Handler with `files.NewHandler(closure-over-engine.GetSessionWorkDir)` and both muxes share it. Phase 119 reuses the same Handler, no new closure. |

**Installation:**
No new dependencies. All libraries are already in `go.mod` and `frontend/package.json`.

**Version verification:**
```bash
grep "^go " /Users/ken/dev/agenthub/go.mod  # → go 1.26.1 (supports method-prefix mux + os.OpenRoot)
grep "chromedp" /Users/ken/dev/agenthub/go.mod  # → v0.15.1 [VERIFIED: source]
```

## Package Legitimacy Audit

> Phase 119 installs **zero** new packages. Audit not required. All transitive deps already pinned in `go.mod` (Go) and `frontend/package-lock.json` (Playwright) and have shipped through prior phases (87, 89, 93, 96, 99, 118). Skipping slopcheck per "phases that install external packages" trigger condition.

## Architecture Patterns

### System Architecture Diagram

```
                       ┌──────────────────────────────────────────┐
                       │   browser (web-share viewer or owner)    │
                       │   - file browser tab (Phase 120)         │
                       │   - sends ?cap=<token> on every request  │
                       └────────────────┬─────────────────────────┘
                                        │ HTTPS over Tailscale
                                        │ (or Tailscale-HTTPS via FQDN)
                                        ▼
   ┌─────────────────────────────────────────────────────────────────┐
   │   WebServer (internal/webserver/server.go)                      │
   │                                                                 │
   │   GET /api/files/list  ───┐                                     │
   │   GET /api/files/stat  ───┼──► ws.requireFilesRead( ... )       │
   │   GET /api/files/read  ───┤        │                            │
   │   HEAD /api/files/read ───┘        │ (composes requireCapability│
   │                                    │  + HasPerm(PermFilesRead)) │
   │                                    ▼                            │
   │                          ws.filesHandler.List/Stat/Read         │
   │                          (= the *files.Handler injected via     │
   │                          SetFilesHandler at construction)       │
   └────────────────────────────────────┬────────────────────────────┘
                                        │ Handler.resolve(sessionID)
                                        ▼
   ┌─────────────────────────────────────────────────────────────────┐
   │   sandboxResolver closure (constructed in internal/daemon/api.go│
   │   NewAPI — same one already used by daemon mux)                 │
   │     1. engine.GetSessionWorkDir(sessionID) → resolved abs path  │
   │     2. files.NewSandbox(wd) → *files.Sandbox (wraps *os.Root)   │
   │     3. files.Handler routes the operation through the sandbox   │
   └─────────────────────────────────────────────────────────────────┘

   ── Parallel surface (unchanged in Phase 119) ──

   daemon-local Unix-socket / Named-pipe (internal/daemon/api.go)
     GET /api/files/list, /stat, /read, HEAD /api/files/read
        │
        │ NO auth — loopback is trust boundary (WEB-01)
        ▼
     a.filesHandler.List/Stat/Read  (same *files.Handler instance)
```

The diagram emphasizes that the same `*files.Handler` instance is shared between the daemon-local mux (auth-less, loopback-only) and the webserver mux (capability-gated, Tailscale-HTTPS). The Handler's `resolve` closure looks up sandbox roots from `engine.sessionWorkDirs` for both surfaces.

### Recommended Project Structure

No new files in `internal/files/` or `internal/capability/` — both packages were finalized in Phase 118. New/modified files:

```
internal/webserver/
├── server.go              # MODIFY: add filesHandler field, SetFilesHandler method, route mounts
├── files_routes_test.go   # NEW: integration tests for the 4 mounted routes (owner-200, viewer-403, no-cap-401, POST-405)
├── capability_mw.go       # OPTIONAL EDIT: update Phase 118 docstring referencing "SetFilesHandlerProvider" → "SetFilesHandler"

internal/daemon/
├── api.go                 # MODIFY: AutoStartWebServer + handleWebServerStart call ws.SetFilesHandler(a.filesHandler)
├── api_test.go            # OPTIONAL: add cross-cutting test that toggles webserver on, mints owner cap, hits /api/files/list

frontend/e2e/
├── web-files-csp.spec.ts  # NEW: WEB-05 cross-browser CSP smoke for file browser flow (or reuse existing web-csp.spec.ts by extending scope)
```

### Pattern 1: Provider Injection + Capability-Wrapped Route Registration

**What:** Mount handler methods on the webserver mux behind a stack of middlewares, with the handler instance injected via a `Set*` method called by the daemon before `ws.Start()`. The mounted middleware composes — outer-to-inner — `basicAuth (local mode) → requireFilesRead → handler method`.

**When to use:** Anytime the webserver needs to expose routes whose handler is constructed by the daemon (the daemon owns the `engine` and session state, the webserver does not).

**Example (closely follows existing `SetSessionResolver` + `setupRoutes` patterns at `server.go:119-121,383`):**
```go
// Source: internal/webserver/server.go (existing setupRoutes pattern, line 383)
// + internal/daemon/api.go (existing SetSessionResolver wiring, line 332)

// In internal/webserver/server.go (new field, new setter, new route mounts):

type WebServer struct {
    // ... existing fields ...
    // filesHandler is the *files.Handler injected by the daemon before Start()
    // via SetFilesHandler. Stateless; the same instance is reused across all
    // requests. Set once before Start(); not mutex-protected (matches
    // sessionResolver pattern).
    filesHandler *files.Handler
}

// SetFilesHandler installs the file handler used to serve the
// /api/files/{list,stat,read} routes on the webserver. Must be called
// before Start(). The handler must already be constructed with its
// sandboxResolver closure (the daemon's NewAPI does this).
//
// Mirrors SetSessionResolver — single setter, no mutex, set once.
func (ws *WebServer) SetFilesHandler(h *files.Handler) {
    ws.filesHandler = h
}

// In setupRoutes (after the existing plugin-config routes ~line 428):

// Phase 119 / WEB-02..WEB-05: capability-gated read-only file API. Mounted
// behind requireFilesRead which composes requireCapability (HMAC + grant +
// session-enabled) with a HasPerm(PermFilesRead) check. The route bodies
// delegate to the shared *files.Handler injected via SetFilesHandler.
//
// Method-prefixed registration per Go 1.22+ mux semantics — POST/PUT/DELETE
// on these URLs auto-return 405 (Pitfall 8 / WEB-02 success criterion 3).
// HEAD is registered explicitly because http.ServeContent dispatches HEAD
// correctly only if the route is mounted under HEAD as well as GET.
//
// Pre-Start() guard: if filesHandler is nil (daemon never wired it),
// requireFilesRead handler runs through and the Handler-method call panics
// on a nil deref. Better: wrap with an explicit "if nil → 503" branch — but
// the daemon always wires it at both NewAPI construction sites, so the nil
// path is theoretically unreachable. Add a one-line nil check for
// defense-in-depth (matches the joinCodes nil guard in handleJoinExchange).
filesH := ws.filesHandler
if filesH != nil {
    mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(filesH.List))
    mux.HandleFunc("GET /api/files/stat", ws.requireFilesRead(filesH.Stat))
    mux.HandleFunc("GET /api/files/read", ws.requireFilesRead(filesH.Read))
    mux.HandleFunc("HEAD /api/files/read", ws.requireFilesRead(filesH.Read))
}
```

**CRITICAL ordering note:** `setupRoutes()` runs inside `NewWebServer()` (line 112) BEFORE the daemon can call `SetFilesHandler`. So `ws.filesHandler` is nil at the moment `setupRoutes` runs. **Two valid resolutions:**

**Option A (recommended):** Defer route registration so the closure captures `ws.filesHandler` at request time instead of at registration time:
```go
mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(func(w http.ResponseWriter, r *http.Request) {
    h := ws.filesHandler
    if h == nil {
        http.Error(w, "files handler not configured", http.StatusServiceUnavailable)
        return
    }
    h.List(w, r)
}))
// (repeat for stat, read, HEAD read)
```
This matches the existing `pluginSettingsProvider` pattern (`server.go:91` field + `handleGetPluginConfig` reads it at request time → 503 if nil).

**Option B:** Change the daemon flow so `SetFilesHandler` is called BEFORE `NewWebServer`. Cannot — `NewWebServer` returns the `*WebServer` which is the receiver `SetFilesHandler` is called on. Rejected.

**Option C:** Move route registration to `Start()` instead of `NewWebServer`. Larger refactor; rejected for the same reason — `setupRoutes` is intentionally idempotent at construction time so the test harness can drive routes via `httptest` without calling `Start()`.

**Recommendation: Option A.** Matches the plugin-config precedent. Add a 503 response when the handler is nil, with a unit test that exercises the "daemon never wired it" path.

### Pattern 2: Daemon-Side Provider Wiring

**What:** Call `ws.SetFilesHandler(a.filesHandler)` after `NewWebServer` returns and before `ws.Start()` runs, at every webserver construction site.

**When to use:** Once per webserver lifecycle. Two construction sites exist today (verified via `grep "NewWebServer(" internal/daemon/api.go`):
- `AutoStartWebServer` (line 322) — startup-time auto-start
- `handleWebServerStart` (line 814) — REST API toggle-on

**Example:**
```go
// Both sites already have a block of "wire callbacks BEFORE Start()" lines.
// SetFilesHandler slots into that block alongside SetSessionResolver,
// SetPluginSettingsProvider, SetSigningKey, SetJoinCodes.
// Source: internal/daemon/api.go AutoStartWebServer line ~332-370

ws.SetSessionResolver(...)             // existing
ws.SetPluginSettingsProvider(...)      // existing
ws.SetFilesHandler(a.filesHandler)     // NEW (Phase 119)
ws.SetSigningKey(key)                  // existing
ws.SetJoinCodes(a.joinCodes)           // existing
if err := ws.Start(); err != nil { ... }
```

**The `a.filesHandler` field already exists in the daemon `API` struct** (`internal/daemon/api.go:54`) and is initialized inside `NewAPI` (line 68) with a resolver closing over `a.engine.GetSessionWorkDir`. **Zero new code in `NewAPI` — Phase 119 reuses the field verbatim.**

### Pattern 3: Integration Test Against Mounted Routes

**What:** Drive the full webserver mux via `testServer(t)` + a real cap token minted with `issueCapFor`, asserting status codes and response body substrings.

**When to use:** Every WEB-02 / WEB-03 success criterion requires a mounted-route assertion (the Phase 118 `TestRequireFilesRead` test mounted the wrapper on a one-off `http.ServeMux` — that does NOT exercise the production registration in `setupRoutes`).

**Example:**
```go
// Source: file pattern follows internal/webserver/capability_test.go:TestRequireFilesRead
// but uses real production routes via testServer(t).
// internal/webserver/files_routes_test.go (NEW)

func TestFilesRoutes_OwnerCapReturns200(t *testing.T) {
    ws, client := testServer(t)
    ws.SetSigningKey(capTestKey)
    ws.EnableSession("sess-files")

    // Wire a fake files.Handler that returns a known sentinel body for List.
    // (Real Handler requires engine.GetSessionWorkDir; use a stub resolver
    // pointing at t.TempDir() with a known file inside.)
    tmp := t.TempDir()
    if err := os.WriteFile(filepath.Join(tmp, "hello.txt"), []byte("hi\n"), 0o644); err != nil {
        t.Fatal(err)
    }
    h := files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
        if sessionID != "sess-files" {
            return nil, errors.New("unknown session")
        }
        return files.NewSandbox(tmp)
    })
    ws.SetFilesHandler(h)

    // Owner token includes files.read in Perms.
    token := issueCapFor(t, ws, "sess-files", "read,write,files.read")

    // GET /api/files/list → 200
    resp, err := client.Get(ws.BaseURL() + "/api/files/list?session=sess-files&path=.&cap=" + token)
    if err != nil { t.Fatal(err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("list owner: expected 200, got %d", resp.StatusCode)
    }
    // ... repeat for /stat, /read (GET + HEAD)
}

func TestFilesRoutes_ViewerCapReturns403(t *testing.T) {
    // Same harness, but token uses "read" only (no files.read).
    // Assert 403 + body contains "files.read".
}

func TestFilesRoutes_MissingCapReturns401(t *testing.T) {
    // Same harness, omit ?cap=. Assert 401 + body contains "capability required".
}

func TestFilesRoutes_PostReturns405(t *testing.T) {
    // Same harness, POST /api/files/list. Assert 405 (Go 1.22+ method-prefix mux behavior).
}

func TestFilesRoutes_NilHandlerReturns503(t *testing.T) {
    // testServer(t) + signing key but NEVER call SetFilesHandler.
    // Mint a token with files.read. Hit /api/files/list. Assert 503 (defense-in-depth path).
}
```

### Anti-Patterns to Avoid
- **Mounting `requireCapability` (not `requireFilesRead`) on file routes.** This passes the HMAC + session checks but skips the `HasPerm(PermFilesRead)` gate — every viewer with a valid cap would get 200 instead of 403. Verified by reading both wrappers; the `requireFilesRead` wrapper IS the gate.
- **Modifying `requireCapability` body to add the files.read check.** `TestRequireCapability_UnchangedByPhase118` (`capability_test.go:575-602`) is a source-inspection guard that fails compilation-equivalent if `requireCapability` body mentions `"files.read"`. Phase 119 must not touch this function.
- **Adding the file routes BEFORE the assets `mux.Handle("GET /assets/")` line.** Go 1.22 mux picks longest-prefix match, so route ordering doesn't matter for correctness — but a future maintainer reading top-to-bottom expects the API routes block to be grouped. Add immediately after the existing plugin-config routes (around line 428).
- **Forgetting to register `HEAD /api/files/read` as a separate route.** Go 1.22 method-prefix mux treats `HEAD` and `GET` as distinct methods. The Phase 118 daemon mux explicitly registers both (`api.go:133-134`); Phase 119 must do the same for the webserver mux. Otherwise `HEAD /api/files/read?cap=...` returns 405 instead of the headers-only 200.
- **Wiring `SetFilesHandler` AFTER `ws.Start()`.** Causes a race with the listener already accepting connections. Both `AutoStartWebServer` and `handleWebServerStart` order all `Set*` calls before `ws.Start()`; preserve that ordering.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Method dispatch (GET vs POST vs HEAD vs OPTIONS) | Custom switch on `r.Method` inside the handler | Go 1.22+ method-prefix mux (`GET /api/files/list`, `HEAD /api/files/read`) | Auto-returns 405 for unregistered methods — satisfies WEB-02 success criterion 3 with zero code. Pattern already used everywhere else in `setupRoutes` and `registerRoutes`. |
| HMAC + grant + session-enabled check | New per-route auth wrapper | Existing `ws.requireCapability` (already composed inside `ws.requireFilesRead`) | All five enforcement steps (no-cap → 401, no-key → 401, bad-sig → 401, wrong-SID → 403, revoked-grant → 403) are already implemented and tested (`capability_test.go`). Re-implementing risks bypassing one of them. |
| Whole-token comma-split for `Perms` | `strings.Contains(claims.Perms, "files.read")` | `capability.HasPerm(claims.Perms, capability.PermFilesRead)` | `strings.Contains` matches `"no-files.read"` as a substring — verified in `capability.go:33-44` and tested in `capability_test.go`. T-118-14 explicitly bans the substring path. |
| File serving with Range/HEAD/ETag | Custom `io.Copy` + Range parser | `http.ServeContent` (already used inside `files.Handler.Read`) | Already shipped Phase 118. The 0-byte short-circuit + 5 MiB cap + MIME cascade live in `files/handler.go:243-290`. Don't touch. |
| Cap token issuance with `files.read` | New Phase 119 code | `engine.filesReadEnabled()` + `issueCapabilitiesForSession` | Already shipped Phase 118 (`api.go:961-1029`). Owner tokens get `read,write,files.read`; viewer tokens get `read` only. **Phase 119 must not edit this path.** Verify by re-running `TestIssueCapabilities_*` tests after the phase. |
| TLS self-signed cert for tests | New keygen | `selfSignedTLSForTest(t)` in `capability_test_helpers.go:43` | Same helper backs every webserver test in the package. |
| CSP middleware on `/api/files/*` | New per-route CSP header setter | **Do not add CSP to JSON routes** | CSP `default-src 'none'` is correct for HTML pages only. JSON routes have no rendering surface; adding CSP headers would only widen request size and confuse the WEB-05 measurement. The existing `cspHeaders` middleware is mounted only on `/dashboard`, `/join`, `/sessions/{id}` — that's correct. |

**Key insight:** Phase 119 is a wiring phase, not a feature phase. Every component already exists; this phase connects four existing pieces (Handler, requireFilesRead wrapper, mux, daemon construction) and writes integration tests proving the connections are correct. The temptation to "improve" the Phase 118 outputs (rename `requireFilesRead`, refactor `*files.Handler`, change cap issuance) must be resisted — those changes belong in their own phase.

## Runtime State Inventory

> Phase 119 is route mounting + middleware wiring. No rename/refactor, no migration. **Skipping this section per template guidance.** Per-category nothing-found confirmation:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no schema changes, no new persisted state | None |
| Live service config | None — no external services touched | None |
| OS-registered state | None — no Windows Task Scheduler / launchd / systemd changes | None |
| Secrets/env vars | None — capability signing key already bootstrapped via `BootstrapCapabilityState` in Phase 87 | None |
| Build artifacts | None — no package rename, no `egg-info`/binary name change | None |

## Common Pitfalls

### Pitfall 1: Forgetting Method-Prefix on HEAD Registration
**What goes wrong:** Registering only `GET /api/files/read` causes `HEAD /api/files/read` to return 405, breaking WEB-02 success criterion 1 (HEAD must return 200 for owner cap) and FS-06 (HEAD preflight for inline-preview vs download decision).
**Why it happens:** `http.ServeContent` inside `files.Handler.Read` dispatches HEAD vs GET internally, so a developer might assume one route registration covers both. It doesn't — Go 1.22+ mux method-prefix matching treats HEAD and GET as distinct.
**How to avoid:** Register both `GET /api/files/read` and `HEAD /api/files/read` explicitly. Verify with the integration test `TestFilesRoutes_HeadReturns200`.
**Warning signs:** WEB-02 SC#1 assertion fails specifically on the HEAD line; daemon-mux tests pass but webserver-mux HEAD tests 405.

### Pitfall 2: Route Registration Ordering Captures Nil Handler
**What goes wrong:** `setupRoutes` runs inside `NewWebServer` (line 112) BEFORE the daemon calls `SetFilesHandler`. If routes are registered as `mux.HandleFunc("GET /api/files/list", ws.requireFilesRead(ws.filesHandler.List))`, the `ws.filesHandler.List` method value is evaluated AT REGISTRATION TIME — when `ws.filesHandler == nil`. Result: nil-pointer panic on first request.
**Why it happens:** Method values in Go bind to the receiver at the expression-evaluation site. `ws.filesHandler.List` evaluates `ws.filesHandler` immediately.
**How to avoid:** Use a closure that reads `ws.filesHandler` at request time (Option A in Pattern 1 above). Explicitly handle the nil case with 503. Same shape as `handleGetPluginConfig` (`plugin_config.go`) which reads `ws.pluginSettingsProvider` at request time and returns 503 if nil.
**Warning signs:** Test crashes with nil deref on first `client.Get` call against the mounted route. Or: works in production (daemon always wires it) but fails in any test that constructs the webserver standalone.

### Pitfall 3: Adding `?session=` to URL Path Instead of Query
**What goes wrong:** Path-style `/api/files/{id}/list` requires path parameter routing (`r.PathValue("id")`) and matches the daemon-socket URL shape from earlier ARCHITECTURE.md drafts. But the **Phase 118 daemon implementation** (`api.go:131-134`) uses **query-style** routing: `/api/files/list?session=<id>&path=<rel>`. The `files.Handler` reads `r.URL.Query().Get("session")` (`handler.go:71`) — it does NOT call `r.PathValue`.
**Why it happens:** ARCHITECTURE.md §1.3 still shows the path-style shape; Phase 118 deliberately changed to query-style to allow stateless Handler reuse across both muxes (no path-template parsing).
**How to avoid:** Mount routes as exactly `GET /api/files/list`, `GET /api/files/stat`, `GET /api/files/read`, `HEAD /api/files/read` — NO path parameters. Session ID arrives via `?session=`.
**Warning signs:** Tests that hit `/api/files/{sessionID}/list` get 404 (no route) or hit a different route entirely. Compare to the daemon's working URL shape in `internal/daemon/api_test.go:1903-1980`.

### Pitfall 4: Test Harness Doesn't Set Per-Session WorkDir
**What goes wrong:** Phase 119 integration tests need a `*files.Handler` whose resolver returns a valid `*Sandbox`. Real production uses `files.NewSandbox(engine.GetSessionWorkDir(sid))`. In a test, `engine` is not constructed — the test must supply a stand-in resolver.
**Why it happens:** `testServer(t)` returns a `*WebServer` that has a `relay.HubManager` but no `engine`. Tests that exercise files require their own sandbox setup.
**How to avoid:** Inside the test, construct a `*files.Handler` with `files.NewHandler(func(sid string) (*files.Sandbox, error) { ... })` where the closure reads from `t.TempDir()`. Then call `ws.SetFilesHandler(h)`. The `*files.Sandbox` type is part of `internal/files`; cross-package use from `internal/webserver` test code is fine because both are within the same module.
**Warning signs:** Tests return 404 "session not found" even with a valid cap token — the resolver closure is rejecting because it returns nil.

### Pitfall 5: WEB-04 (Tailnet Remote) Has No Webserver-Side Work
**What goes wrong:** A planner may believe WEB-04 ("file browser works against tailnet-remote sessions") requires server-side code in Phase 119. It does not. Each tailnet peer runs its own AgentHub daemon with its own webserver; once Phase 119 mounts the routes correctly on a single webserver, every peer exposes them automatically.
**Why it happens:** "Remote session" implies cross-machine RPC; on AgentHub it's just "fetch the remote peer's HTTPS URL with the cap token issued by that peer."
**How to avoid:** Treat WEB-04 as a frontend concern (Phase 120). The Phase 119 plan needs only a sentence acknowledging this — no code or test work.
**Warning signs:** Plan tasks include "wire tailnet peer discovery into file routes" or "add cross-peer proxying." Both wrong.

### Pitfall 6: CSP Smoke Misses the Real Question
**What goes wrong:** WEB-05 says "zero new CSP violations." A naive interpretation runs the existing CSP e2e tests, sees them pass, and ships. But none of the existing tests **navigate to a file browser page** — they navigate to `/dashboard`, `/join`, and `/sessions/{id}`. A CSP violation introduced by a future file-browser page would not be caught.
**Why it happens:** CSP middleware is mounted on HTML routes only. `/api/files/*` returns JSON/octet-stream — invoking those routes from a browser produces no CSP traffic at all.
**How to avoid:** WEB-05 is **really** about Phase 120 (the React file browser page that consumes these routes). Phase 119's WEB-05 coverage is: (a) run the existing CSP e2e suite and confirm it still passes (terminal + dashboard + join unchanged), and (b) add an assertion that no `/api/files/*` route serves `Content-Type: text/html`. The cross-browser Playwright smoke against the file-browser tab is a Phase 120 deliverable, not Phase 119.
**Warning signs:** Plan tasks include "test Playwright file browser CSP" with no actual file-browser UI to navigate to.

### Pitfall 7: Edge Cases in `requireFilesRead` Order Already Tested
**What goes wrong:** A planner adds tests for "401 takes priority over 403" (bad signature + missing files.read) or "claims missing from context" — these are already covered by `TestRequireFilesRead` subtests in `capability_test.go:445-568`. Duplicate coverage wastes time and adds maintenance overhead.
**Why it happens:** Phase 118 tested the wrapper standalone; Phase 119 tests need to cover only the **mounted-route delta** (URL routing, query parameter parsing, response body shape against the real Handler).
**How to avoid:** Read `capability_test.go:TestRequireFilesRead` before writing the Phase 119 test file. Only add tests that exercise the integration delta: real Handler + real mux + real URL parsing.

## Code Examples

Verified patterns from existing source (no Context7 lookups needed — all examples are from the local codebase, which is the authoritative source for AgentHub's conventions).

### Existing `SetSessionResolver` field + setter (pattern to mirror)
```go
// Source: internal/webserver/server.go:82-83, 119-121

// sessionResolver is set once before Start() and is not mutex-protected.
sessionResolver func(sessionID string) (name, cliType, status, hostname string)

// SetSessionResolver sets the callback used by handleListSessions and
// handleSessionInfo to resolve session metadata. Must be called before Start().
func (ws *WebServer) SetSessionResolver(fn func(string) (string, string, string, string)) {
    ws.sessionResolver = fn
}
```

### Existing nil-provider 503 pattern (template for the nil-filesHandler path)
```go
// Source: internal/webserver/plugin_config.go (handleGetPluginConfig reads
// ws.pluginSettingsProvider at request time and returns 503 if nil — the
// exact shape Phase 119 should use for the nil-filesHandler path).
//
// File at internal/webserver/plugin_config.go line ~20-30 — the actual lines
// are not shown above but the field declaration at server.go:91 confirms it
// is read at request time, not registration time.
```

### Existing route mount with method prefix + requireCapability wrapping
```go
// Source: internal/webserver/server.go:417, 423, 428

mux.HandleFunc("GET /api/sessions/{id}/info", ws.requireCapability(ws.handleSessionInfo))
mux.HandleFunc("GET /api/plugin-config", ws.requireCapability(ws.handleGetPluginConfig))
mux.HandleFunc("GET /api/plugin-config/stream", ws.requireCapability(ws.handleStreamPluginConfig))
```

### Daemon-side `Set*` wiring block (slot SetFilesHandler in here)
```go
// Source: internal/daemon/api.go:332-370 (AutoStartWebServer)
// and api.go:827-858 (handleWebServerStart)

ws.SetSessionResolver(func(sessionID string) (name, cliType, status, hostname string) { ... })
ws.SetPluginSettingsProvider(func() []byte { ... })
// NEW: ws.SetFilesHandler(a.filesHandler)   // Phase 119
a.engine.SetPluginSettingsListener(func() { ws.BroadcastPluginConfig(context.Background()) })
a.signingKeyMu.RLock()
key := a.signingKey
a.signingKeyMu.RUnlock()
ws.SetSigningKey(key)
ws.SetJoinCodes(a.joinCodes)
if err := ws.Start(); err != nil { ... }
```

### Daemon-side Handler construction (verbatim — already exists, do not duplicate)
```go
// Source: internal/daemon/api.go:68-77 (NewAPI). This block is already in
// production. Phase 119 reuses a.filesHandler — does NOT add a second
// NewHandler call.

a.filesHandler = files.NewHandler(func(sessionID string) (*files.Sandbox, error) {
    if sessionID == "" {
        return nil, errors.New("missing session parameter")
    }
    wd := a.engine.GetSessionWorkDir(sessionID)
    if wd == "" {
        return nil, errors.New("session not found or has no working directory")
    }
    return files.NewSandbox(wd)
})
```

### Daemon-mux file route registration (already shipped — mirror this for webserver)
```go
// Source: internal/daemon/api.go:131-134

a.mux.HandleFunc("GET /api/files/list", a.filesHandler.List)
a.mux.HandleFunc("GET /api/files/stat", a.filesHandler.Stat)
a.mux.HandleFunc("GET /api/files/read", a.filesHandler.Read)
a.mux.HandleFunc("HEAD /api/files/read", a.filesHandler.Read)
```

### Phase 118 wrapper-standalone test (template for Phase 119 mounted-route tests)
```go
// Source: internal/webserver/capability_test.go:445-568 — TestRequireFilesRead.
// Phase 119 tests follow the same structure but use the REAL webserver mux
// (via ws.BaseURL() + client.Get) instead of a one-off http.ServeMux.
```

## CSP Impact Analysis (WEB-05)

The Phase 118 daemon-mux file routes already serve `Content-Type: application/json` (for List/Stat) and the file's resolved MIME (for Read). **None of these are HTML.** The webserver mux Phase 119 mounts use the same handler — same content types.

**CSP middleware (`csp_mw.go`) is mounted only on HTML routes:**
- `/dashboard` (line 397)
- `/join` (line 402)
- `/sessions/{id}` (line 433)

File routes do NOT go through `cspHeaders`. This is correct: CSP is a browser-rendering policy with no effect on JSON consumed via `fetch()`.

**WEB-05 "zero new violations" verification path:**
1. **Existing tests pass:** Run the existing `browser_csp_e2e_test.go` suite (`-tags=e2e`) and confirm `TestBrowserCSP_TerminalNoViolations`, `TestBrowserCSP_DashboardNoViolations`, `TestBrowserCSP_JoinNoViolations` still report zero violations. Phase 119 does NOT modify HTML pages, so these should pass unchanged.
2. **Defense-in-depth assertion:** Add a unit test asserting that `GET /api/files/list?...` and `GET /api/files/read?...` responses have **no** `Content-Security-Policy` header and `Content-Type` does **not** start with `text/html`. This rules out a future regression where someone mistakenly wraps file routes in `cspHeaders` or sets HTML content type.
3. **Cross-browser Playwright (frontend/e2e):** Existing `web-csp.spec.ts` runs against `/sessions/{id}` (the terminal page) — does NOT yet navigate to a file browser tab (because none exists). This spec passes unchanged in Phase 119 and is **expected to be extended in Phase 120** when the file browser UI lands. The WEB-05 success criterion as written ("Zero new CSP violations... from a complete file browse flow against the webserver") cannot be fully exercised in Phase 119 because there is no file-browser browser-loadable page yet. **Recommendation: amend WEB-05 in plan-phase to read "no CSP regressions on existing HTML pages + no `/api/files/*` route serves HTML" for Phase 119, and defer the file-browse-flow Playwright suite to Phase 120.** This is consistent with the requirement's intent (the *frontend* must not introduce CSP violations).

## Test Patterns to Follow

### Integration tests (new file: `internal/webserver/files_routes_test.go`)
Mirror `internal/webserver/capability_test.go` structure:
- Use `testServer(t)` helper to get a real listening WebServer + TLS-aware HTTP client.
- Use `issueCapFor(t, ws, sid, perms)` to mint cap tokens. Pass `"read,write,files.read"` for owner, `"read"` for viewer.
- Use `t.TempDir()` to construct a real on-disk sandbox; populate with `os.WriteFile`.
- Construct `files.NewHandler(closure)` inside each test (or in a shared helper) and call `ws.SetFilesHandler(h)` before issuing requests.
- Assert `resp.StatusCode` against the WEB-02 success matrix. Assert body substring with `strings.Contains(string(body), "files.read")` for viewer-403 (WEB-03).

### Required tests (one per WEB-02/03 success criterion)
| Test name | Asserts |
|-----------|---------|
| `TestFilesRoutes_OwnerCapReturns200_List` | Owner cap → 200 on GET /api/files/list |
| `TestFilesRoutes_OwnerCapReturns200_Stat` | Owner cap → 200 on GET /api/files/stat |
| `TestFilesRoutes_OwnerCapReturns200_Read_Get` | Owner cap → 200 on GET /api/files/read |
| `TestFilesRoutes_OwnerCapReturns200_Read_Head` | Owner cap → 200 on HEAD /api/files/read + correct Content-Length |
| `TestFilesRoutes_ViewerCapReturns403_List` | Viewer cap (no files.read) → 403 + body contains "files.read" |
| `TestFilesRoutes_ViewerCapReturns403_Stat` | Same on /stat |
| `TestFilesRoutes_ViewerCapReturns403_Read_Get` | Same on GET /read |
| `TestFilesRoutes_ViewerCapReturns403_Read_Head` | Same on HEAD /read |
| `TestFilesRoutes_MissingCapReturns401` | No `?cap=` → 401 (not 404) — route-existence-leak guard |
| `TestFilesRoutes_PostReturns405` | POST /api/files/list → 405 (Go 1.22+ method-prefix mux) |
| `TestFilesRoutes_PutReturns405` | PUT /api/files/read → 405 |
| `TestFilesRoutes_DeleteReturns405` | DELETE /api/files/list → 405 |
| `TestFilesRoutes_NilHandlerReturns503` | SetFilesHandler never called → 503 (defense-in-depth) |
| `TestFilesRoutes_NoCSPHeader` | GET /api/files/list has no `Content-Security-Policy` response header (WEB-05 defense-in-depth) |
| `TestFilesRoutes_NoHTMLContentType` | GET /api/files/list `Content-Type` is `application/json`, never `text/html` (WEB-05) |
| `TestFilesRoutes_DaemonMuxStillNoAuth` | Regression guard for WEB-01: daemon-socket /api/files/list works WITHOUT any cap token |

### Optional but recommended: end-to-end test via daemon construction path
A test that constructs the full daemon (`NewAPI`) + bootstraps capability state + starts the webserver via `AutoStartWebServer` (or `SetWebServerForTest`) + issues a real owner cap via `issueCapabilitiesForSession` + hits the webserver mux. This catches **the wiring bug** (forgetting to call `SetFilesHandler` in `AutoStartWebServer` or `handleWebServerStart`). The unit-mocked tests above use `testServer(t)` which bypasses `AutoStartWebServer` entirely.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Mux's `http.HandleFunc("/", ...)` with method-switch inside handler | Method-prefix patterns: `mux.HandleFunc("GET /path", ...)` | Go 1.22 (Feb 2024) | 405 auto-returns for unregistered methods — no custom code needed |
| `filepath.EvalSymlinks` + `os.Open` (TOCTOU race) | `*os.Root` + `Root.Open` (atomic, kernel-level) | Go 1.24 (Feb 2025) | Already adopted in Phase 118 via `internal/files/sandbox.go` |
| `Claims.Perms == "read,write"` exact-match | `capability.HasPerm(claims.Perms, perm)` whole-token | Phase 118 (May 2026) | Already adopted; never use `strings.Contains` |

**Deprecated/outdated:**
- The path-style file routes shown in ARCHITECTURE.md §1.3 / Decision 2 (`/sessions/{id}/files/list`) were superseded by the query-style routes (`/api/files/list?session=<id>`) actually shipped in Phase 118. Phase 119 follows the shipped shape.
- `SetFilesHandlerProvider` (named in `internal/webserver/capability_mw.go:99` and `internal/files/handler.go:6` comments) was the originally proposed name. **The simpler `SetFilesHandler` form** (passing a `*Handler` directly, not a constructor func) is cleaner because the Handler is already stateless. Update both comments in this phase.

## Assumptions Log

> All technical claims in this research were verified against the local codebase via `Read` and `Bash grep`. No `[ASSUMED]` claims.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| — | (none) | — | — |

**Table empty:** All claims verified against `internal/webserver/`, `internal/daemon/api.go`, `internal/files/handler.go`, `internal/capability/capability.go`, `go.mod`, `frontend/e2e/`, and `.planning/REQUIREMENTS.md` / `ROADMAP.md`. No user confirmation needed before planning.

## Open Questions

1. **Rename in capability_mw.go docstring?**
   - What we know: `internal/webserver/capability_mw.go:99` and `internal/files/handler.go:6` reference `SetFilesHandlerProvider` (with "Provider" suffix).
   - What's unclear: Should Phase 119 use that name verbatim, or simplify to `SetFilesHandler`?
   - Recommendation: **Use `SetFilesHandler`** (no Provider suffix). Rationale: every other "provider" in the codebase (`SetPluginSettingsProvider`) injects a function; this one injects an already-constructed `*Handler` instance — the "Provider" suffix is misleading. Update both comments to match. The discuss step was skipped (auto-mode), so this is Claude's discretion per CONTEXT.md.

2. **Should `TestFilesRoutes_DaemonMuxStillNoAuth` live in `internal/daemon/api_test.go` or `internal/webserver/files_routes_test.go`?**
   - What we know: The daemon-mux WEB-01 surface is in `internal/daemon`; the webserver-mux WEB-02 surface is in `internal/webserver`.
   - What's unclear: Where does a cross-cutting "daemon mux still auth-less after Phase 119" regression test belong?
   - Recommendation: **Add it to `internal/daemon/api_test.go`** (the daemon already has `TestAPI_FilesRead_*` tests there per Phase 118). Phase 119 may not need to touch this file at all if those tests already cover the regression — verify in the plan-phase by reading `api_test.go:1900-2120`.

3. **Playwright fixture extension scope.**
   - What we know: `cmd/playwright-fixture/main.go` wires `SetSessionResolver` + `SetPluginSettingsProvider` but NOT `SetFilesHandler`.
   - What's unclear: Does Phase 119 add `SetFilesHandler` wiring to the fixture? Or defer to Phase 120 when the file-browser page tests actually need it?
   - Recommendation: **Defer to Phase 120.** Phase 119's Playwright work is only the WEB-05 regression check (existing tests still pass). The fixture extension belongs in Phase 120 alongside the new file-browser spec file. Phase 119 confirms `cmd/playwright-fixture/main.go` compiles unchanged.

## Environment Availability

> Phase 119 has zero new external dependencies. All tools already verified by prior phases. Quick re-confirmation:

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All Go work | ✓ | 1.26.1 (from `go.mod`) | — |
| `chromedp` (Go module) | `browser_csp_e2e_test.go` `-tags=e2e` runs | ✓ | v0.15.1 | Manual UAT per existing pattern |
| Chromium binary | chromedp e2e | (varies per dev box) | — | Tests self-skip via `chromedp not found` check (line 71-73 of `browser_csp_e2e_test.go`) |
| `@playwright/test` (npm) | `frontend/e2e/web-csp.spec.ts` | ✓ | (locked in `frontend/package-lock.json`) | — |
| Playwright browsers (Chromium/Firefox/WebKit) | Cross-browser CSP run | (varies — `npx playwright install`) | — | Skip cross-browser, run Chromium only |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** Chromium/Playwright browsers — graceful fallback to existing skip behavior.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `net/http/httptest` (in-process); `chromedp` v0.15.1 (in-tree browser); Playwright @latest (cross-browser e2e) |
| Config file | None for Go tests; `frontend/playwright.config.ts` for Playwright |
| Quick run command | `go test ./internal/webserver/... ./internal/daemon/...` |
| Full suite command | `go test ./... && cd frontend && npm run test:e2e` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WEB-01 | Daemon-socket file routes remain auth-less after Phase 119 | unit (Go) | `go test ./internal/daemon/... -run TestAPI_FilesRead -v` | ✅ (Phase 118) — `internal/daemon/api_test.go:1900-2000` already covers this; verify still passes |
| WEB-02 | Webserver mux exposes 4 file routes under requireFilesRead | integration (Go) | `go test ./internal/webserver/... -run TestFilesRoutes -v` | ❌ Wave 0 — `internal/webserver/files_routes_test.go` (new file) |
| WEB-03 | Viewer cap → 403 on all 4 endpoints with "files.read" in body | integration (Go) | `go test ./internal/webserver/... -run TestFilesRoutes_ViewerCap -v` | ❌ Wave 0 — same file |
| WEB-04 | Tailnet-remote file browsing | manual / Phase 120 frontend | — (no server-side code) | n/a |
| WEB-05 | Zero new CSP violations (existing pages unchanged) | e2e (chromedp + Playwright) | `go test -tags=e2e ./internal/webserver/... -run TestBrowserCSP` + `cd frontend && npx playwright test web-csp.spec.ts` | ✅ Already exists; verify still passes after Phase 119 |
| WEB-05 | No `/api/files/*` route serves HTML or sets CSP header | unit (Go) | `go test ./internal/webserver/... -run TestFilesRoutes_No(CSP|HTML)` | ❌ Wave 0 — same new file |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... -run TestFilesRoutes -v` (< 5s)
- **Per wave merge:** `go test ./internal/webserver/... ./internal/daemon/...` (< 30s for the targeted packages)
- **Phase gate:** Full suite green: `go test ./... && go test -tags=e2e ./internal/webserver/... && cd frontend && npm run test:e2e` (chromedp + Playwright may skip if browsers unavailable; that's acceptable as documented in `browser_csp_e2e_test.go:71-73`)

### Wave 0 Gaps
- [ ] `internal/webserver/files_routes_test.go` — new file covering 15 test cases listed in §"Test Patterns to Follow"
- [ ] (Optional) Daemon-mux regression-guard test in `internal/daemon/api_test.go` if not already covered — confirm by reading existing test names
- [ ] Update docstring in `internal/webserver/capability_mw.go:99` and `internal/files/handler.go:6` to refer to `SetFilesHandler` (no "Provider" suffix)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Capability tokens via HMAC-SHA256 — already implemented in `internal/capability`; Phase 119 mounts existing middleware |
| V3 Session Management | yes | Grant-list revocation (D-14/D-15) + signing-key rotation (D-16) — already implemented |
| V4 Access Control | yes | `requireFilesRead` enforces `files.read` capability bit; viewer-403 is the WEB-03 success criterion |
| V5 Input Validation | yes | Sandbox path validation done in `internal/files/sandbox.go` (Phase 118 — TOCTOU-free via `os.OpenRoot`); fuzz corpus passes |
| V6 Cryptography | yes | HMAC-SHA256 with constant-time `hmac.Equal`; never hand-rolled |

### Known Threat Patterns for Go HTTP + capability-gated file routes

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cap-token replay across sessions | Spoofing | `Claims.SID` matches route path/query session ID (already enforced by `requireCapability` SEC-03) |
| Path traversal via query parameter | Tampering | `os.OpenRoot` (Phase 118); Phase 119 does not touch the sandbox layer |
| Privilege escalation: viewer → file access | Elevation | `HasPerm(claims.Perms, PermFilesRead)` whole-token check in `requireFilesRead` — viewer never has the bit |
| Route-existence enumeration via 404 vs 401 | Information disclosure | `requireCapability` returns 401 for both "missing cap" and "invalid sig" — never 404 (WEB-02 success criterion 5) |
| Method-not-allowed bypass via OPTIONS/PUT/DELETE | Tampering | Go 1.22+ method-prefix mux returns 405 for unregistered methods (WEB-02 success criterion 3) |
| Cross-mux confusion (daemon-socket route exposed via Tailscale) | Spoofing | Separate `*http.ServeMux` instances per surface; daemon mux binds to loopback only via `listenDaemonSocket` |
| CSP regression introducing inline script | Tampering | WEB-05 — no CSP middleware mounted on JSON routes; unit test confirms no HTML content type or CSP header |

## Project Constraints (from CLAUDE.md)

Extracted from `./CLAUDE.md` (project instructions checked into the repo):

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions where applicable. Phase 119 handlers receive `r.Context()` via stdlib `*http.Request` — already context-aware.
- **Testing:** `pytest` / `go testing` / `vitest`-or-`jest`. New tests use Go stdlib `testing` package (consistent with rest of `internal/webserver/*_test.go`).
- **Make beliefs pay rent:** Phase 119 makes one explicit prediction — "every route registered with method prefix returns 405 for non-matching methods" — verified by test, not assumed.
- **Notice confusion:** If a `client.Get` returns 404 instead of 401 against a mounted route, that's the surprise marker for "the route isn't actually registered" — addressed in Pitfall 2.
- **Chesterton's Fence:** `requireCapability` is the fence. Phase 119 must not modify its body (T-118-14 / `TestRequireCapability_UnchangedByPhase118`).
- **Silent fallbacks forbidden:** Nil `filesHandler` returns 503 with an explicit error message, never silently 200 with empty body.
- **Premature abstraction:** Two webserver construction sites in `api.go` (`AutoStartWebServer`, `handleWebServerStart`) is the threshold for "three real examples before abstracting." Phase 119 does NOT abstract a `wireWebServerCallbacks` helper — two is not three.
- **Never `kill node.exe`:** N/A — Go test process, not Node-related.

## Sources

### Primary (HIGH confidence)
- `internal/webserver/server.go` — current setupRoutes pattern, `Set*` precedents, mux ordering
- `internal/webserver/capability_mw.go` — `requireCapability` + `requireFilesRead` middleware bodies (Phase 118 final form)
- `internal/webserver/capability_test.go` — existing test patterns; `TestRequireFilesRead` (wrapper standalone) is the template
- `internal/webserver/capability_test_helpers.go` — `testServer(t)`, `issueCapFor`, `selfSignedTLSForTest`
- `internal/webserver/csp_mw.go` — CSP policy verbatim; confirms file routes do not go through it
- `internal/webserver/browser_csp_e2e_test.go` — chromedp-based in-tree CSP e2e pattern + self-skip path
- `internal/daemon/api.go` — `NewAPI` constructs `a.filesHandler`; `AutoStartWebServer` + `handleWebServerStart` are the two `Set*` wiring sites
- `internal/files/handler.go` — `*Handler` exposes `List`, `Stat`, `Read` as `http.HandlerFunc`-shaped methods
- `internal/files/sandbox.go` — `*Sandbox` type used by tests
- `internal/capability/capability.go` — `PermFilesRead` constant, `HasPerm` helper, `Claims` struct
- `internal/daemon/engine.go` — `filesReadEnabled`, `GetSessionWorkDir`, `sessionWorkDirs` map
- `.planning/REQUIREMENTS.md` — WEB-01..WEB-05 verbatim; FS-10..FS-13 context
- `.planning/ROADMAP.md` — Phase 119 goal + success criteria
- `.planning/research/ARCHITECTURE.md` — overall decisions (note: §1.3 URL shape is superseded by Phase 118's query-style choice)
- `.planning/research/PITFALLS.md` — Pitfall 4 (capability-bit wire format), Pitfall 8 (mux conflict), Pitfall 9 (markdown CSP)
- `go.mod` — Go 1.26.1, chromedp v0.15.1, websocket dep already in tree
- `frontend/playwright.config.ts` + `frontend/e2e/web-csp.spec.ts` + `frontend/e2e/global-setup.ts` — Playwright harness
- `cmd/playwright-fixture/main.go` — fixture binary (not yet wiring `SetFilesHandler`)

### Secondary (MEDIUM confidence)
- None — every claim was verified against authoritative project source.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in `go.mod` / `frontend/package-lock.json`, verified by `grep`
- Architecture: HIGH — pattern follows existing `SetSessionResolver` + `setupRoutes` verbatim
- Pitfalls: HIGH — all 7 pitfalls grounded in either Phase 118 history (Pitfalls 1-5) or AgentHub-specific test patterns (Pitfalls 6-7)
- Test mapping: HIGH — every WEB-XX requirement has an explicit test command and file location
- Security domain: HIGH — relies entirely on Phase 118 / Phase 87 / Phase 89 already-shipped controls

**Research date:** 2026-05-20
**Valid until:** 2026-06-20 (30 days — stable Go 1.26 stdlib + frozen Phase 118 outputs)

## RESEARCH COMPLETE
