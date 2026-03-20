# Phase 16: Auth Layer Removal - Research

**Researched:** 2026-03-20
**Domain:** Go HTTP middleware removal, React frontend cleanup, Wails binding cleanup
**Confidence:** HIGH

## Summary

Phase 16 removes all authentication infrastructure from AgentHub's web server. The codebase currently has a two-layer auth system: dashboard password auth (cookie-based, bcrypt) and per-session shareable tokens. With Tailscale providing network-level access control (only tailnet members can reach the server), these layers are redundant and should be deleted entirely.

The scope is well-bounded. Auth lives in two Go files (`internal/webserver/auth.go`, `internal/webserver/tokens.go`), three server methods/routes in `server.go`, three `App` methods in `app.go`, one persisted file path (`web_password`), one React component (`SettingsPanel.tsx` Security tab), one frontend helper in `App.tsx`, and one prop on `StatusBar.tsx`. The web `dashboard.html` also needs its login section removed, and `handleSessionQR` (currently behind dashboardAuth) needs to lose that guard.

The critical insight for the planner: `StartWebServer` in `app.go` currently gates on `IsWebPasswordSet()`. That gate must be removed as part of this phase — the server should start as long as Tailscale is healthy (already checked in Phase 15). There is also a pre-existing failing frontend test (`Security tab shows CA certificate path`) that expects a CA cert section in the Security tab that was removed in Phase 15. Phase 16 should delete that obsolete test along with the Security tab itself.

**Primary recommendation:** Delete auth files wholesale, strip auth fields from `WebServer` struct, reroute `sessionAuth` to only gate on web-enabled state (not cookie/token), update `StartWebServer` to not require a password, remove `SetWebPassword`/`IsWebPasswordSet`/`GenerateSessionToken` from `app.go` and Wails bindings, update the frontend and dashboard HTML, and delete all auth-covering tests.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTH-01 | Password authentication is removed from the web dashboard | Delete `AuthManager`, `POST /login`, `dashboardAuth` middleware, `SetWebPassword`/`IsWebPasswordSet` from app.go and Wails bindings, Security tab from SettingsPanel, login section from dashboard.html |
| AUTH-02 | Per-session shareable tokens and links are removed | Delete `TokenStore`, `POST /api/sessions/{id}/token` route, `handleCreateToken`, `GenerateSessionToken` from app.go and Wails bindings, "Copy Link" button from StatusBar, `copyTokenLink` logic from App.tsx |
| AUTH-03 | Web dashboard is accessible without authentication to any tailnet member | Remove `dashboardAuth` and `sessionAuth` cookie/token checks; `sessionAuth` replacement gates only on web-enabled state; `StartWebServer` no longer requires a password |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `net/http` | built-in | Route handler rewrite | All routing already uses this; no new dependency |
| React (existing) | existing | Frontend JSX cleanup | Already in use; just removing dead code |

### Supporting
None — this phase removes code, not adds it.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Full deletion | "No-op auth" (always allow) | No-op stubs leave dead code paths; full deletion is simpler and achieves AUTH-03 |

**Installation:**
No new packages to install. After auth removal, `golang.org/x/crypto` may become unused. Verify:
```bash
go mod tidy
```
If `golang.org/x/crypto` was only used for `bcrypt` (in `auth.go`), it will be dropped. Check `go.mod` after.

Note: At the time of research, `golang.org/x/crypto v0.46.0` is in `go.mod`. It may be pulled in transitively by `tailscale.com` — run `go mod tidy` and check if it stays or goes. Do not remove it manually; let `go mod tidy` decide.

## Architecture Patterns

### Recommended Project Structure (after deletion)

```
internal/webserver/
├── server.go           # routes + handlers (auth middleware removed)
├── server_test.go      # rewritten: no login/token tests
├── tailscale.go        # unchanged
├── tailscale_test.go   # unchanged
#   auth.go             DELETED
#   auth_test.go        DELETED
#   tokens.go           DELETED
#   tokens_test.go      DELETED
```

```
frontend/src/
├── App.tsx                         # remove GenerateSessionToken import + handleCopyTokenLink
├── components/
│   ├── SettingsPanel.tsx           # remove Security tab, remove isPasswordSet state
│   ├── StatusBar.tsx               # remove onCopyTokenLink prop + "Copy Link" button
│   └── __tests__/
│       ├── SettingsPanel.test.tsx  # update: remove Security tab tests, remove mock of SetWebPassword/IsWebPasswordSet
│       └── StatusBar.test.tsx      # update: remove "Copy Link" test cases
├── wailsjs/go/main/App.d.ts        # remove SetWebPassword, IsWebPasswordSet, GenerateSessionToken
├── wailsjs/go/main/App.js          # remove same three exports
```

```
app.go
# Remove: SetWebPassword, IsWebPasswordSet, GenerateSessionToken, webPasswordPath
# Remove: StartWebServer gate on IsWebPasswordSet
# Remove: LoadPasswordHash call in StartWebServer
```

```
web/dashboard.html
# Remove: login-section HTML, login-error div, doLogin() function
# Remove: 401-handling in refreshSessions()
# Remove: auto-check login on page load (just call refreshSessions() directly)
# Remove: copyTokenLink() function + "Copy Token Link" button from session card rendering
# Remove: CA Certificate Installation section (was already from v1.0, now fully obsolete with Tailscale TLS)
```

### Pattern 1: Simplified `sessionAuth` — web-enabled gate only

**What:** Replace the old `sessionAuth` middleware (which checked cookie OR token) with a direct check for web-enabled state only. No auth check at all.

**When to use:** For all `/sessions/{id}` and `/sessions/{id}/ws` routes.

**Example:**
```go
// BEFORE (auth.go + token store lookup)
func (ws *WebServer) sessionAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sessionID := r.PathValue("id")
        if !ws.isSessionEnabled(sessionID) {
            http.NotFound(w, r)
            return
        }
        // ... cookie/token check
        next(w, r)
    }
}

// AFTER (web-enabled gate only — no middleware struct needed)
// Just inline in setupRoutes or keep as a thin guard:
mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
    if !ws.isSessionEnabled(r.PathValue("id")) {
        http.NotFound(w, r)
        return
    }
    ws.handleTerminalPage(w, r)
})
```

### Pattern 2: `setupRoutes` after removal

**What:** Routes that were behind `dashboardAuth` become open. The `POST /login` and `POST /api/sessions/{id}/token` routes are deleted entirely.

```go
func (ws *WebServer) setupRoutes() {
    mux := ws.mux

    mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            http.Redirect(w, r, "/dashboard", http.StatusFound)
            return
        }
        http.NotFound(w, r)
    })

    mux.HandleFunc("GET /dashboard", ws.handleDashboard)

    // GET /api/sessions — open (no dashboardAuth)
    mux.HandleFunc("GET /api/sessions", ws.handleListSessions)

    // GET /sessions/{id} — gated only on web-enabled
    mux.HandleFunc("GET /sessions/{id}", ws.handleTerminalPage)  // with inline web-enabled check

    // GET /sessions/{id}/ws — gated only on web-enabled
    mux.HandleFunc("GET /sessions/{id}/ws", ws.handleWSSRelay)  // with inline web-enabled check

    // GET /api/sessions/{id}/qr — open (no dashboardAuth)
    mux.HandleFunc("GET /api/sessions/{id}/qr", ws.handleSessionQR)

    // POST /login                      DELETED
    // POST /api/sessions/{id}/token    DELETED
}
```

### Pattern 3: `WebServer` struct after removal

```go
type WebServer struct {
    config  Config
    // auth    *AuthManager   REMOVED
    // tokens  *TokenStore    REMOVED
    manager *relay.HubManager

    mu          sync.RWMutex
    webEnabled  map[string]bool
    listener    net.Listener
    mux         *http.ServeMux

    sessionResolver func(sessionID string) (name, cliType, status string)
}
```

### Pattern 4: `StartWebServer` in `app.go` after removal

The password gate and `LoadPasswordHash` call are deleted. The Tailscale health gate (added in Phase 15) is kept.

```go
func (a *App) StartWebServer(port int) error {
    // REMOVED: if !a.IsWebPasswordSet() { return error }

    h := a.GetTailscaleStatus()
    if !h.Connected { return fmt.Errorf("Tailscale is not connected") }
    if h.IP == "" { return fmt.Errorf("Tailscale IP not available") }

    // ... rest unchanged, except:
    // REMOVED: if hash, err := os.ReadFile(webPasswordPath()); err == nil { ws.LoadPasswordHash(hash) }
}
```

### Pattern 5: Wails bindings update

Wails auto-generates `frontend/src/wailsjs/go/main/App.js` and `App.d.ts` from the Go `App` struct's exported methods. After removing `SetWebPassword`, `IsWebPasswordSet`, and `GenerateSessionToken` from `app.go`, the Wails binding files must be updated to match. These files are checked into the repo as stubs (confirmed by reading `App.d.ts` which has `// AUTO-GENERATED by Wails`). They must be edited manually since `wails generate module` requires the full Wails toolchain.

Remove from `App.d.ts`:
```typescript
export function SetWebPassword(password: string): Promise<void>
export function IsWebPasswordSet(): Promise<boolean>
export function GenerateSessionToken(sessionID: string): Promise<string>
```

Remove the same three functions from `App.js`.

### Pattern 6: `WebSocket` origin patterns after removal

The `handleWSSRelay` currently has a comment about auth middleware. Update the comment:
```go
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    // The server is accessible only to tailnet members via Tailscale.
    // Accept connections from any origin.
    OriginPatterns: []string{"*"},
})
```

### Anti-Patterns to Avoid

- **Leaving dead code stubs:** Do not leave `SetWebPassword`, `IsWebPasswordSet`, or `GenerateSessionToken` as no-op functions in `app.go` — delete them entirely.
- **Leaving `simpleCookieJar` in `server.go`:** This unexported struct exists only to support test helpers that do cookie-based auth. Once auth tests are gone, so is the need for it.
- **Forgetting to update `testServer` helper:** `server_test.go`'s `testServer` function calls `ws.SetPassword("testpass")` — this call must be removed. Tests that called `login()` before API calls will need the `login()` call removed too, and the session API will now respond 200 without auth.
- **Leaving the CA section in `dashboard.html`:** The CA certificate installation section in the dashboard references `/ca.crt` which no longer exists (self-signed certs were removed in Phase 15). This should be deleted.
- **Forgetting `go mod tidy`:** After deleting `auth.go`, `golang.org/x/crypto` may become unused. Run `go mod tidy` before final test.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Replacing auth with "public" auth | Custom always-pass middleware | Just remove the middleware | No middleware needed when network provides access control |
| Cleaning unused Go deps | Manually editing go.mod | `go mod tidy` | Handles transitive dep analysis automatically |

**Key insight:** Tailscale's network layer IS the access control. No application-layer auth needed.

## Common Pitfalls

### Pitfall 1: Missed Import Cleanup
**What goes wrong:** After deleting `auth.go` and `tokens.go`, `server.go` still imports their types or calls their functions, causing compile errors.
**Why it happens:** `WebServer` struct fields `auth *AuthManager` and `tokens *TokenStore` are declared in `server.go`. Those field refs must be removed from the struct, `NewWebServer`, `SetPassword`, `LoadPasswordHash`, `IsPasswordSet`, `CreateToken`, and `setupRoutes`.
**How to avoid:** Remove the struct fields first, then let the compiler guide removal of all dependent call sites.
**Warning signs:** `undefined: AuthManager`, `undefined: TokenStore` errors.

### Pitfall 2: Stale Frontend Bindings
**What goes wrong:** `SettingsPanel.tsx` still imports `SetWebPassword`/`IsWebPasswordSet` from `wailsjs/go/main/App`, causing TypeScript errors after the Go methods are removed.
**Why it happens:** Frontend binding files are checked-in stubs, not auto-regenerated at edit time.
**How to avoid:** Update `App.d.ts` and `App.js` at the same time as `app.go`. Also update all import sites in `.tsx` files.
**Warning signs:** TypeScript error `Module '"../wailsjs/go/main/App"' has no exported member 'SetWebPassword'`.

### Pitfall 3: Test Helper Assumptions
**What goes wrong:** `server_test.go` has a `login()` helper and `testServer()` calls `ws.SetPassword()`. Tests that call `login()` before hitting `/api/sessions` will fail if `SetPassword` is removed but the call remains.
**Why it happens:** Multiple tests use the shared `testServer()` and `login()` pattern.
**How to avoid:** Remove `SetPassword` call from `testServer()`, delete the `login()` helper, rewrite each test that used `login()` to simply make the API call directly.
**Warning signs:** `ws.SetPassword undefined (type *WebServer has no field or method SetPassword)`.

### Pitfall 4: Pre-existing Failing Frontend Test
**What goes wrong:** `SettingsPanel.test.tsx` has a test `'Security tab shows CA certificate path'` that already fails in the current codebase (the CA cert section was removed in Phase 15 but the test was not updated).
**Why it happens:** Phase 15 removed the CA cert UI from the Security tab without cleaning up this test.
**How to avoid:** Delete or update this test in Phase 16 when the Security tab is removed entirely.
**Warning signs:** 1 failing test in the current suite before any Phase 16 changes.

### Pitfall 5: `StartWebServer` Gate Removal Requires Test Update
**What goes wrong:** `app_test.go` has `TestStartWebServerErrorsWhenPasswordNotSet` which specifically tests the password gate. After removing the gate, this test will fail because `StartWebServer` no longer errors on missing password.
**Why it happens:** Test was written to verify the password requirement.
**How to avoid:** Delete this test. `StartWebServer` still errors when Tailscale is not connected, so the Tailscale gate tests remain valid.
**Warning signs:** `TestStartWebServerErrorsWhenPasswordNotSet` passes unexpectedly or panics.

### Pitfall 6: `simpleCookieJar` in `server.go`
**What goes wrong:** After removing auth, the unexported `simpleCookieJar` struct in `server.go` becomes dead code that will fail linting.
**Why it happens:** It was added to support auth-related test helpers that could share the server file.
**How to avoid:** Delete `simpleCookieJar` from `server.go` when cleaning up server routes. Also delete `testCookieJar` in `server_test.go` if it's only used for cookie-based auth test flows.
**Warning signs:** `simpleCookieJar declared but not used` lint error (or unused type warning).

### Pitfall 7: `dashboard.html` CA Section
**What goes wrong:** The dashboard HTML still contains a CA certificate installation section that references `/ca.crt`, a route that no longer exists (self-signed cert infrastructure was removed in Phase 15). While this is UI dead code, it's an incorrect user-facing section.
**Why it happens:** Phase 15 removed the CA cert backend but did not update the dashboard HTML.
**How to avoid:** Delete the `<section id="ca-section">` block in `dashboard.html` during Phase 16.

## Code Examples

### Verifying Auth Removal Compiles Clean
```bash
# After removing auth.go, tokens.go, and updating server.go:
go build ./...
go vet ./...
go test ./internal/webserver/...
```

### Checking `golang.org/x/crypto` After Removal
```bash
go mod tidy
# If crypto was only used by auth.go (bcrypt), it will be removed from go.mod.
# If tailscale.com pulls it transitively, it stays as indirect.
grep "golang.org/x/crypto" go.mod
```

### Running Full Test Suite After Changes
```bash
# Go
go test ./... -race

# Frontend
cd frontend && pnpm test
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Password + session cookie for dashboard | No auth needed — Tailscale network layer | Phase 16 | Simpler, fewer moving parts |
| Per-session bearer tokens in URL | No tokens — direct `/sessions/{id}` URLs | Phase 16 | Clean URLs, no token lifecycle to manage |
| `/ca.crt` route + CA cert install instructions | Tailscale Let's Encrypt certs (trusted by browsers natively) | Phase 15 | Dashboard CA section obsolete |

**Deprecated/outdated after this phase:**
- `AuthManager` (auth.go): bcrypt password hashing and session cookie management
- `TokenStore` (tokens.go): per-session shareable token generation and lookup
- `POST /login` route: dashboard login endpoint
- `POST /api/sessions/{id}/token` route: token creation endpoint
- `dashboardAuth` middleware: session cookie validation
- `sessionAuth` middleware: token OR cookie validation (replaced by web-enabled-only gate)
- `SetWebPassword` / `IsWebPasswordSet` / `GenerateSessionToken` App methods
- `webPasswordPath()` helper and `~/.config/agenthub/web_password` persisted file
- Security tab in SettingsPanel
- "Copy Link" button in StatusBar + `onCopyTokenLink` prop
- Login section in dashboard.html
- CA certificate section in dashboard.html (from v1.0)

## Open Questions

1. **Is `golang.org/x/crypto` used transitively?**
   - What we know: Currently in `go.mod` as a direct dependency (`golang.org/x/crypto v0.46.0`). Used only in `auth.go` for `bcrypt`.
   - What's unclear: Whether `tailscale.com` or other deps pull it transitively.
   - Recommendation: Run `go mod tidy` after deletion and commit the updated `go.mod`/`go.sum`. Do not manually edit.

2. **Should `GetSessionQRCode` in app.go also change?**
   - What we know: `GetSessionQRCode` calls `handleSessionQR` logic. The route `GET /api/sessions/{id}/qr` was behind `dashboardAuth` but will be open after removal.
   - What's unclear: Whether the Wails binding `GetSessionQRCode` needs any change — it bypasses the HTTP layer entirely.
   - Recommendation: No change to `GetSessionQRCode` in `app.go`. It constructs QR codes directly, not via the web route. The QR endpoint in the dashboard will simply be unauthenticated after removing `dashboardAuth`.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib) + Vitest |
| Config file | `frontend/vitest.config.ts` (implied by `vitest run` in package.json) |
| Quick run command | `go test ./internal/webserver/... -race` |
| Full suite command | `go test ./... -race && cd frontend && pnpm test` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | Dashboard accessible without password | unit | `go test ./internal/webserver/... -run TestWebServerDashboardNoAuthRequired -race` | ✅ (test exists, needs update) |
| AUTH-01 | `/api/sessions` returns 200 without auth | unit | `go test ./internal/webserver/... -run TestWebServerSessionListAPI -race` | ✅ (test exists, needs update — remove login() call) |
| AUTH-01 | No `POST /login` route exists | unit | new test: `go test ./internal/webserver/... -run TestLoginRouteNotRegistered` | ❌ Wave 0 |
| AUTH-02 | No `POST /api/sessions/{id}/token` route exists | unit | new test: `go test ./internal/webserver/... -run TestTokenRouteNotRegistered` | ❌ Wave 0 |
| AUTH-02 | `GenerateSessionToken` removed from Wails bindings | source-inspection | `cd frontend && pnpm test -t "GenerateSessionToken not exported"` | ❌ Wave 0 (optional) |
| AUTH-03 | Sessions endpoint open without cookie | unit | `go test ./internal/webserver/... -run TestSessionAccessWithoutAuth -race` | ❌ Wave 0 |
| AUTH-03 | Web-enabled gate still returns 404 for non-enabled session | unit | `go test ./internal/webserver/... -run TestWebServerToggle -race` | ✅ (existing, no auth needed) |
| AUTH-03 | `StartWebServer` no longer requires password | unit | `go test . -run TestStartWebServerNoPasswordRequired -race` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver/... -race`
- **Per wave merge:** `go test ./... -race && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/webserver/server_test.go` — rewrite `testServer()` to not call `SetPassword`; delete `login()` helper; rewrite auth-dependent tests as open-access tests
- [ ] `internal/webserver/server_test.go` — add `TestLoginRouteNotRegistered`, `TestTokenRouteNotRegistered`, `TestSessionAccessWithoutAuth`
- [ ] `app_test.go` — delete `TestStartWebServerErrorsWhenPasswordNotSet`, `TestSetWebPasswordPersistsAndReloads`; add `TestStartWebServerNoPasswordRequired`
- [ ] `frontend/src/components/__tests__/SettingsPanel.test.tsx` — delete `Security tab shows CA certificate path` (already failing), delete Security tab test cases, update mock to remove `SetWebPassword`/`IsWebPasswordSet`
- [ ] `frontend/src/components/__tests__/StatusBar.test.tsx` — remove `onCopyTokenLink` prop references if prop is removed from `StatusBarProps`

*(Pre-existing failure: `SettingsPanel.test.tsx > Security tab shows CA certificate path` — 1 test failing before Phase 16 starts. This test must be deleted in Phase 16.)*

## Sources

### Primary (HIGH confidence)
- Direct code inspection of `/Users/ken/dev/agenthub/internal/webserver/auth.go` — full AuthManager implementation
- Direct code inspection of `/Users/ken/dev/agenthub/internal/webserver/tokens.go` — full TokenStore implementation
- Direct code inspection of `/Users/ken/dev/agenthub/internal/webserver/server.go` — route registration, middleware usage, struct fields
- Direct code inspection of `/Users/ken/dev/agenthub/app.go` — SetWebPassword, IsWebPasswordSet, GenerateSessionToken, StartWebServer gate
- Direct code inspection of `/Users/ken/dev/agenthub/frontend/src/components/SettingsPanel.tsx` — Security tab with password UI
- Direct code inspection of `/Users/ken/dev/agenthub/frontend/src/components/StatusBar.tsx` — Copy Link button, onCopyTokenLink prop
- Direct code inspection of `/Users/ken/dev/agenthub/web/dashboard.html` — login section, CA section, copyTokenLink JS
- Direct code inspection of `/Users/ken/dev/agenthub/frontend/src/wailsjs/go/main/App.d.ts` — Wails binding exports
- Live test run: `go test ./...` — all Go tests pass; `pnpm test` — 1 pre-existing frontend failure

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` — project decisions, dependency chain context
- `.planning/REQUIREMENTS.md` — AUTH-01/02/03 definitions, CLEAN-02/03 scope (Phase 17)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — pure deletion, no new libraries
- Architecture: HIGH — all implementation files read and fully understood
- Pitfalls: HIGH — discovered by reading actual test code and running the suite

**Research date:** 2026-03-20
**Valid until:** 60 days — stable Go stdlib patterns, no third-party deps to track
