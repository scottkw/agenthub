# Phase 87: Capability-Based Session Authorization - Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 17 new/modified files
**Analogs found:** 15 / 17 (2 files have no direct analog — see bottom)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/capability/capability.go` (NEW) | service (pure-Go crypto/tokens) | transform (claims ↔ token bytes) | `internal/webserver/auth.go` (middleware factory), `internal/relay/protocol.go` (stdlib-only bytes layer) | role-match |
| `internal/capability/keystore.go` (NEW) | utility (file persistence) | file-I/O (load/save 32-byte key) | `internal/daemon/engine.go` lines 86–133 (`loadSettingsFromDisk` / `saveSettingsToDisk`) | exact-pattern |
| `internal/capability/joincode.go` (NEW) | service (in-memory TTL map) | CRUD (issue/exchange/expire) | `internal/webserver/server.go` lines 82–115 (`webEnabled` map + mutex) | role-match |
| `internal/capability/capability_test.go` (NEW) | test (unit + fuzz) | — | `internal/webserver/auth_test.go`, `internal/daemon/engine_settings_test.go` | exact |
| `internal/capability/keystore_test.go` (NEW) | test (persistence round-trip) | file-I/O | `internal/daemon/engine_settings_test.go` lines 11–99 | exact |
| `internal/webserver/server.go` (MODIFIED) | controller + middleware | request-response + WS upgrade | self (existing `handleWSSRelay`, `setupRoutes`, `basicAuthMiddleware` wiring) | self |
| `internal/webserver/capability_mw.go` (NEW) | middleware | request-response | `internal/webserver/auth.go` (`BasicAuthMiddleware`) | exact-pattern |
| `internal/webserver/server_test.go` (MODIFIED) | test | HTTP + WS integration | self + `security-review/internal_webserver_server_test.go` helpers | self |
| `internal/daemon/api.go` (MODIFIED) | controller (Unix-socket IPC) | request-response (JSON) | self lines 501–524 (`handleWebServe`) for new capability IPC handlers | self |
| `internal/daemon/types.go` (MODIFIED) | model (JSON types) | transform | self lines 62–88 (`WebServerStartRequest/Response`) | self |
| `internal/daemon/client.go` (MODIFIED) | service (typed IPC client) | request-response | self lines 204–207 (`ToggleWebServing`) | self |
| `internal/daemon/engine.go` (MODIFIED) | service | file-I/O + state | self lines 136–154 (`NewSessionEngine`) — add KeyStore wiring | self |
| `app.go` (MODIFIED, root) | Wails binding | request-response | self lines 469–613 (`StartWebServer`, `GetSessionQRCode`, `GetLocalNetworkPassword`) | self |
| `frontend/src/components/SettingsTab.tsx` (MODIFIED) | component | UI state + IPC | self lines 289–310 (`handleToggleServer`), 451–478 (CT-disclosure pattern), 495–552 (URL/QR row) | self |
| `frontend/src/components/RegenerateKeyModal.tsx` (NEW) | component (confirmation modal) | event-driven | `frontend/src/components/QuitConfirmModal.tsx` (whole file) | exact |
| `frontend/src/components/SessionSharePanel.tsx` (NEW) | component (per-session share UI) | UI state + IPC | `frontend/src/components/SettingsTab.tsx` lines 495–552 (dashboard URL action row + QR) | role-match |
| `web/dashboard.html` (REPLACED) | view (server HTML) | static | self (structure + palette) — content rewrite | self |
| `web/join.html` (NEW) | view (server HTML) | form POST | `web/dashboard.html` (whole file; palette + inline CSS idiom) | exact-pattern |

---

## Pattern Assignments

### `internal/capability/capability.go` (NEW — service, transform)

**Analog:** no direct in-repo analog for HMAC token encoding. Closest style anchors:
- `internal/webserver/auth.go` — minimal single-purpose package layout
- `internal/relay/protocol.go` — stdlib-only byte layer (frame encode/decode)
- Research Pattern 1 in `87-RESEARCH.md` lines 214–281 is already exactly what to copy.

**Imports pattern** (copy from RESEARCH Pattern 1, matches stdlib-only convention in `internal/relay/protocol.go`):
```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "strings"
)
```

**Package-doc style** (mirror `internal/webserver/auth.go:7-15` terse doc comment):
```go
// Package capability issues and verifies HMAC-SHA256 capability tokens that
// gate web access to individual PTY sessions. Tokens encode a compact claim
// set as base64url(claimsJSON).base64url(sig). See SEC-01..SEC-05.
package capability
```

**Sign/Verify pattern** — use RESEARCH.md lines 238–281 verbatim. Key sub-patterns to carry:
- `hmac.Equal(sig, expected)` for constant-time comparison (NEVER `bytes.Equal`).
- `base64.RawURLEncoding` (no padding) for both segments.
- Marshal/Unmarshal JSON with the `Claims` struct in declaration order; `v` field is the forward-compat guard.

**Error surface** (matches `http.Error` short-message convention at `server.go:373`, `server.go:354`, etc.):
```go
var (
    ErrMalformedToken  = errors.New("capability: malformed token")
    ErrInvalidSignature = errors.New("capability: invalid signature")
    ErrMalformedClaims  = errors.New("capability: malformed claims")
)
```

**Context helper** (pattern used by `request.Context()` in `handleWSSRelay` at `server.go:409`):
```go
type ctxKey struct{}

// WithClaims attaches claims to the context so handlers downstream of
// requireCapability can read Perms, GrantID, etc. without re-verifying.
func WithClaims(ctx context.Context, c Claims) context.Context {
    return context.WithValue(ctx, ctxKey{}, c)
}

// ClaimsFromContext returns the claims attached by requireCapability.
// Returns zero-value Claims and false when the middleware did not run.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
    c, ok := ctx.Value(ctxKey{}).(Claims)
    return c, ok
}
```

---

### `internal/capability/keystore.go` (NEW — utility, file-I/O)

**Analog:** `internal/daemon/engine.go` lines 86–133 (`loadSettingsFromDisk`, `saveSettingsToDisk`) — this is the canonical persistence idiom in this codebase and MUST be copied exactly for:
- Missing file is NOT an error (first run)
- `os.WriteFile` with mode `0600` is the atomic-write primitive in use
- Config dir is provided via `daemonConfigDir()` at `engine.go:45-53` (creates `~/.config/agenthub/` with 0700)

**Imports pattern** (copy from `engine.go:1-16` filtered to file-only deps):
```go
import (
    "crypto/rand"
    "fmt"
    "os"
    "path/filepath"
)
```

**Atomic write pattern** (`engine.go:132`):
```go
// engine.go line 132 — the exact idiom to mirror in FileKeyStore.Save:
_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
```

Apply to keystore:
```go
func (s *FileKeyStore) Save(key []byte) error {
    // Mode 0600 + os.WriteFile — matches saveSettingsToDisk at engine.go:132.
    return os.WriteFile(s.path, key, 0600)
}
```

**Missing-file-is-not-error pattern** (`engine.go:88-92`):
```go
// engine.go lines 88-92 — the exact idiom to mirror in FileKeyStore.Load:
data, err := os.ReadFile(settingsPath(dir))
if err != nil {
    return // file not found or unreadable — not an error
}
```

Apply to keystore (matches RESEARCH Pattern 2 lines 325–337):
```go
func (s *FileKeyStore) Load() ([]byte, error) {
    data, err := os.ReadFile(s.path)
    if os.IsNotExist(err) {
        return nil, nil // first run — caller generates
    }
    if err != nil {
        return nil, fmt.Errorf("capability: read key file: %w", err)
    }
    if len(data) != 32 {
        return nil, fmt.Errorf("capability: key file corrupt (got %d bytes, want 32)", len(data))
    }
    return data, nil
}
```

---

### `internal/capability/joincode.go` (NEW — service, CRUD + TTL)

**Analog:** `internal/webserver/server.go` lines 82–115 (`EnableSession`/`DisableSession`/`webEnabledSessions`) — exact idiom for an in-memory map guarded by a mutex with short CRUD methods.

**Struct + mutex pattern** (`server.go:52-60` — the struct; `server.go:82-94` — the locking idiom):
```go
// server.go lines 82-94 pattern — copy this exactly for JoinCodeManager:
// ws.mu.Lock() / defer ws.mu.Unlock() on write; RLock on read-only.

type JoinCodeManager struct {
    mu    sync.Mutex                 // simple Mutex (NOT RWMutex) — see RESEARCH Pitfall 4
    codes map[string]joinEntry
    ttl   time.Duration              // 5 * time.Minute (D-11)
    now   func() time.Time           // injection seam for tests
}

type joinEntry struct {
    token  string
    expiry time.Time
}
```

**Base32 encode pattern** (no existing in-repo analog — copy from RESEARCH.md lines 460–485):
```go
// Base32 RFC 4648 alphabet A-Z 2-7 — matches D-10. No padding.
var joinCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func (m *JoinCodeManager) Issue(token string) (string, error) {
    var raw [5]byte
    if _, err := rand.Read(raw[:]); err != nil {
        return "", err
    }
    encoded := joinCodeEncoding.EncodeToString(raw[:])
    code := encoded[:4] + "-" + encoded[4:8]
    m.mu.Lock()
    m.codes[code] = joinEntry{token: token, expiry: m.now().Add(m.ttl)}
    m.mu.Unlock()
    return code, nil
}
```

**TOCTOU-safe exchange** (RESEARCH Pitfall 4 at line 589–594 — must hold mutex across lookup-then-delete):
```go
func (m *JoinCodeManager) Exchange(code string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    entry, ok := m.codes[code]
    if !ok {
        return "", ErrCodeNotFound
    }
    if m.now().After(entry.expiry) {
        delete(m.codes, code)
        return "", ErrCodeExpired
    }
    delete(m.codes, code)  // single-use: delete before returning success
    return entry.token, nil
}
```

---

### `internal/capability/keystore_test.go` (NEW — test, file-I/O)

**Analog:** `internal/daemon/engine_settings_test.go` lines 11–99 — exact template for `t.TempDir()` + write/read/round-trip tests.

**Template to copy** (from `engine_settings_test.go:11-56`):
```go
func TestFileKeyStoreRoundTrip(t *testing.T) {
    dir := t.TempDir()

    store := capability.NewFileKeyStore(dir)

    // First load: no file exists — returns (nil, nil).
    got, err := store.Load()
    if err != nil { t.Fatalf("Load on empty dir: %v", err) }
    if got != nil { t.Errorf("expected nil on missing file, got %x", got) }

    // Save a key.
    key := make([]byte, 32)
    for i := range key { key[i] = byte(i) }
    if err := store.Save(key); err != nil {
        t.Fatalf("Save: %v", err)
    }

    // Verify file mode is 0600 (mirrors saveSettingsToDisk behavior).
    info, err := os.Stat(store.Location())
    if err != nil { t.Fatal(err) }
    if info.Mode().Perm() != 0600 {
        t.Errorf("file mode = %v, want 0600", info.Mode().Perm())
    }

    // Second load: round-trips the saved bytes.
    got2, err := store.Load()
    if err != nil { t.Fatalf("Load after Save: %v", err) }
    if !bytes.Equal(got2, key) {
        t.Errorf("round-trip mismatch: got %x, want %x", got2, key)
    }
}

func TestFileKeyStoreMissingFile(t *testing.T) {
    // Mirrors TestSettingsLoadMissingFile at engine_settings_test.go:88-99.
    dir := t.TempDir()
    store := capability.NewFileKeyStore(dir)
    got, err := store.Load()
    if err != nil { t.Errorf("Load on missing file: %v", err) }
    if got != nil { t.Errorf("expected nil, got %x", got) }
}
```

---

### `internal/webserver/capability_mw.go` (NEW — middleware, request-response)

**Analog:** `internal/webserver/auth.go` (whole file, 34 lines). Exact template: middleware factory returning `func(http.Handler) http.Handler` AND an unexported alias delegate.

**Copy-structure pattern** (from `auth.go:7-28`):
```go
// auth.go is the exact shape to mirror. Key properties:
//  - Factory function returning func(http.Handler) http.Handler
//  - On failure: http.Error(w, msg, code); return immediately.
//  - On success: next.ServeHTTP(w, r) (or r.WithContext for injected claims).
```

**Adaptation for capability** (note the sig differs from `auth.go` because we wrap `http.HandlerFunc` per RESEARCH Pattern 3 — pick ONE shape and use consistently):
```go
// RECOMMENDED shape — mirrors auth.go signature but wraps HandlerFunc.
// This allows registration like:
//   mux.HandleFunc("GET /sessions/{id}/ws", ws.requireCapability(ws.handleWSSRelay))
// which mirrors how basicAuthMiddleware wraps ws.mux at server.go:201.

func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("cap")
        if token == "" {
            http.Error(w, "capability required", http.StatusUnauthorized)
            return
        }
        claims, err := capability.Verify(token, ws.signingKey())
        if err != nil {
            http.Error(w, "capability required", http.StatusUnauthorized)
            return
        }
        if pathID := r.PathValue("id"); pathID != "" && claims.SID != pathID {
            http.Error(w, "capability does not match session", http.StatusForbidden)
            return
        }
        if !ws.isGrantActive(claims.SID, claims.GrantID) {
            http.Error(w, "capability has been revoked", http.StatusForbidden)
            return
        }
        ctx := capability.WithClaims(r.Context(), claims)
        next(w, r.WithContext(ctx))
    }
}
```

**Ordering with Basic Auth** (D-20): in `startLocal`, the mux is wrapped in `basicAuthMiddleware` at `server.go:201`, which runs BEFORE any per-route middleware. That gives us defense-in-depth for free — Basic Auth first, then capability. No additional chaining code needed.

---

### `internal/webserver/server.go` (MODIFIED — controller + WS upgrade)

**Analog:** self. Specific edits and their patterns:

**Edit 1 — Struct field additions** (mirror existing field block at `server.go:52-60`):
```go
// After ws.webEnabled at server.go:54, add:
grants       map[string]map[string]struct{} // sessionID -> set of active grant_ids (D-14)
signingKeyFn func() []byte                  // loaded once, swapped on RegenerateSigningKey (D-16)
joinCodes    *capability.JoinCodeManager
```

**Edit 2 — setupRoutes wrapping** (existing pattern `server.go:263` `mux.HandleFunc("GET /dashboard", ws.handleDashboard)` → wrap with `requireCapability`):
```go
// BEFORE — server.go:266
mux.HandleFunc("GET /api/sessions", ws.handleListSessions)

// AFTER (SEC-02):
mux.HandleFunc("GET /api/sessions", ws.requireCapability(ws.handleListSessions))

// BEFORE — server.go:272-278
mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
    if !ws.IsSessionEnabled(r.PathValue("id")) {
        http.NotFound(w, r)
        return
    }
    ws.handleTerminalPage(w, r)
})

// AFTER (SEC-03) — drop the webEnabled pre-check; requireCapability already
// validates the grant is active, which implies web-enabled.
mux.HandleFunc("GET /sessions/{id}", ws.requireCapability(ws.handleTerminalPage))

// AFTER (SEC-03 + D-22) for WS upgrade:
mux.HandleFunc("GET /sessions/{id}/ws", ws.requireCapability(ws.handleWSSRelay))
```

**Edit 3 — handleListSessions single-session response** (D-18) — modify `server.go:304-320`:
```go
// BEFORE — returns all enabled sessions.
// AFTER — reads claims from context, returns ONLY claims.SID.
func (ws *WebServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
    claims, _ := capability.ClaimsFromContext(r.Context())
    items := make([]sessionListItem, 0, 1)
    if ws.IsSessionEnabled(claims.SID) && ws.sessionResolver != nil {
        name, cliType, st, hostname := ws.sessionResolver(claims.SID)
        if name == "" { name = claims.SID }
        items = append(items, sessionListItem{
            ID: claims.SID, Name: name, CLIType: cliType, Status: st, Hostname: hostname,
        })
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items) //nolint:errcheck
}
```

**Edit 4 — handleWSSRelay readonly rewrite** (D-24) — modify `server.go:382-415`:
```go
// REMOVE — server.go:386-387 (`?readonly=1` read):
// readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"

// REPLACE WITH — source from verified claims:
claims, _ := capability.ClaimsFromContext(r.Context())
readonly := claims.Perms == "read"

// The sub assignment at server.go:411-415 stays structurally identical —
// only the source of ReadOnly changes.
sub := &relay.Subscriber{
    Msgs:     make(chan []byte, 256),
    ReadOnly: readonly, // now server-bound (SEC-04)
    Name:     clientName,
}
```

**Edit 5 — Grant list methods** (mirror `EnableSession`/`DisableSession`/`IsSessionEnabled` at `server.go:82-102` exactly):
```go
// Add alongside EnableSession/DisableSession — same lock pattern:
func (ws *WebServer) AddGrant(sessionID, grantID string) {
    ws.mu.Lock()
    if ws.grants[sessionID] == nil {
        ws.grants[sessionID] = make(map[string]struct{})
    }
    ws.grants[sessionID][grantID] = struct{}{}
    ws.mu.Unlock()
}

func (ws *WebServer) ClearGrants(sessionID string) {
    ws.mu.Lock()
    delete(ws.grants, sessionID)
    ws.mu.Unlock()
}

func (ws *WebServer) isGrantActive(sessionID, grantID string) bool {
    ws.mu.RLock()
    defer ws.mu.RUnlock()
    if ws.grants[sessionID] == nil { return false }
    _, ok := ws.grants[sessionID][grantID]
    return ok
}
```

**Edit 6 — OriginPatterns** (Phase 88 coordination — leave for Phase 88, but comment update):
```go
// server.go:400-404 — update comment only; do NOT remove OriginPatterns: ["*"]:
conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
    // Phase 87: capability check has already passed here. Origin allowlist
    // arrives in Phase 88 (WebSocket Handshake Security).
    OriginPatterns: []string{"*"},
})
```

---

### `internal/webserver/server_test.go` (MODIFIED — test, HTTP + WS integration)

**Analog:** self (existing `testServer`, `testServerWithHub` helpers) + `security-review/internal_webserver_server_test.go` scaffold.

**Copy helpers into `server_test.go`** (the `selfSignedTLSForTest`, `testServerWithHub`, `dialWebServerWS`, and `readPipeWithTimeout` helpers from `security-review/internal_webserver_server_test.go:29-198`). These are ALREADY written by the security reviewer and just need to be relocated into the test file.

**Invert the two exploit tests** (RESEARCH lines 701–724). The reviewer named them `TestSecurity_UnauthenticatedClientCanEnumerate…` — rename to `TestSecurity_UnauthenticatedClientCannotEnumerate…` and assert `401`/`403` instead of `200`.

**New test skeletons** (follow existing pattern at `server_test.go:115-125`):
```go
// Pattern to copy from server_test.go:115-125:
func TestCapability_MissingCapReturns401(t *testing.T) {
    ws, client := testServer(t)
    ws.EnableSession("s-1")

    resp, err := client.Get(ws.BaseURL() + "/api/sessions")
    if err != nil { t.Fatal(err) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusUnauthorized {
        t.Errorf("expected 401, got %d", resp.StatusCode)
    }
}
```

**Fuzz test** (new file `internal/capability/capability_fuzz_test.go` referencing the scaffold at `security-review/internal_relay_protocol_fuzz_test.go`):
```go
// Run with: go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s
func FuzzVerify(f *testing.F) {
    key := make([]byte, 32)
    for i := range key { key[i] = byte(i) }
    // Seed with a known-valid token.
    good, _ := capability.Sign(capability.Claims{SID: "s1", Perms: "read", V: 1}, key)
    f.Add(good)

    f.Fuzz(func(t *testing.T, token string) {
        // Must never panic; must return an error for all malformed/tampered input.
        _, _ = capability.Verify(token, key)
    })
}
```

---

### `internal/daemon/api.go` (MODIFIED — controller, request-response)

**Analog:** self. New IPC endpoints (`IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`) follow `handleWebServe` pattern at `api.go:501-524`.

**Route registration** (mirror `api.go:62`):
```go
// Add to registerRoutes at api.go:43-70:
a.mux.HandleFunc("POST /sessions/{id}/capabilities", a.handleIssueCapabilities)
a.mux.HandleFunc("POST /join/exchange", a.handleExchangeJoinCode)
a.mux.HandleFunc("POST /capability/regenerate-key", a.handleRegenerateSigningKey)
```

**Handler pattern** (copy shape from `api.go:501-524`):
```go
// handleWebServe at api.go:501-524 is the template. Replicate:
//  1. r.PathValue("id") for path param
//  2. json.NewDecoder(r.Body).Decode(&req) for JSON body
//  3. a.mu.RLock() to read a.webServer
//  4. http.Error for error responses, writeJSON (api.go:209-213) for success
```

**Auto-enable REMOVAL (SEC-01)** — delete `api.go:287-293`:
```go
// REMOVE THIS BLOCK from handleCreateSession:
// Auto-enable web serving for this session if the web server is running (SERVE-02).
// a.mu.RLock()
// ws := a.webServer
// a.mu.RUnlock()
// if ws != nil {
//     ws.EnableSession(id)
// }
```

**onExit grant cleanup (RESEARCH Pitfall 1)** — modify `api.go:268-277`:
```go
// CURRENT onExit closure only calls DisableSession. Add ClearGrants to avoid
// the leak described in RESEARCH Pitfall 1.
onExit := func(sessionID string, exitCode int) {
    time.AfterFunc(10*time.Second, func() {
        a.mu.RLock()
        ws := a.webServer
        a.mu.RUnlock()
        if ws != nil {
            ws.DisableSession(sessionID)
            ws.ClearGrants(sessionID)  // D-15 also applies on natural exit
        }
    })
}
```

**handleWebServe toggle behavior (D-06, D-15)** — modify `api.go:518-523`:
```go
// CURRENT: straight Enable/Disable pass-through.
// AFTER: on Enable → issue caps; on Disable → clear grants.
if req.Enabled {
    ws.EnableSession(id)
    // Issue two caps (D-07) and register grant_ids in the grant list.
    // Returns URLs via writeJSON.
    readURL, writeURL, err := a.issueCapabilitiesForSession(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, http.StatusOK, IssueCapabilitiesResponse{
        ReadURL: readURL, WriteURL: writeURL,
    })
    return
}
ws.DisableSession(id)
ws.ClearGrants(id)
w.WriteHeader(http.StatusNoContent)
```

---

### `internal/daemon/types.go` (MODIFIED — JSON types)

**Analog:** self lines 62–88. New types follow exactly the same struct-per-endpoint idiom.

**Copy template** (from `types.go:62-82`):
```go
// Mirror WebServerStartRequest/Response at types.go:62-82:

// IssueCapabilitiesResponse is the response body for POST /sessions/{id}/capabilities.
type IssueCapabilitiesResponse struct {
    ReadURL  string `json:"readUrl"`
    WriteURL string `json:"writeUrl"`
    ReadCode string `json:"readCode"`   // join code for read cap (D-09)
    WriteCode string `json:"writeCode"` // join code for write cap
}

// ExchangeJoinCodeRequest is the body for POST /join/exchange.
type ExchangeJoinCodeRequest struct {
    Code string `json:"code"`
}

// ExchangeJoinCodeResponse is the response for POST /join/exchange.
type ExchangeJoinCodeResponse struct {
    URL string `json:"url"` // redirect-worthy URL with ?cap=<token>
}

// RegenerateSigningKeyResponse is the response for POST /capability/regenerate-key.
type RegenerateSigningKeyResponse struct {
    // Intentionally empty — no data to return; 200 signals success.
}
```

---

### `internal/daemon/client.go` (MODIFIED — typed IPC client)

**Analog:** self lines 204–207 (`ToggleWebServing`). Every new client method follows the same three-line shape.

**Template** (`client.go:204-207`):
```go
// ToggleWebServing enables or disables web serving for a session.
func (c *DaemonClient) ToggleWebServing(sessionID string, enabled bool) error {
    return c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/web-serve", WebServeRequest{Enabled: enabled}, nil)
}
```

**New methods** (mirror the template):
```go
func (c *DaemonClient) IssueCapabilities(sessionID string) (IssueCapabilitiesResponse, error) {
    var resp IssueCapabilitiesResponse
    err := c.doJSON(http.MethodPost, "/sessions/"+sessionID+"/capabilities", nil, &resp)
    return resp, err
}

func (c *DaemonClient) ExchangeJoinCode(code string) (string, error) {
    var resp ExchangeJoinCodeResponse
    if err := c.doJSON(http.MethodPost, "/join/exchange", ExchangeJoinCodeRequest{Code: code}, &resp); err != nil {
        return "", err
    }
    return resp.URL, nil
}

func (c *DaemonClient) RegenerateSigningKey() error {
    return c.doJSON(http.MethodPost, "/capability/regenerate-key", nil, nil)
}
```

---

### `internal/daemon/engine.go` (MODIFIED — wire KeyStore at startup)

**Analog:** self lines 136–154 (`NewSessionEngine`).

**Startup wiring pattern** (mirror `engine.go:140-152` — how `loadSettingsFromDisk` is called after struct construction):
```go
// engine.go:140-152 pattern:
e := &SessionEngine{ /* fields */ }
e.loadSettingsFromDisk(cfgDir)
return e

// Apply the same idiom: add a signingKey field + key store load after
// loadSettingsFromDisk, so the key exists before any web server is started.
```

Alternative (cleaner per RESEARCH Open Question 1): keep signing key on `API`, not `SessionEngine`. The `API` then passes it to `WebServer` in `AutoStartWebServer` at `api.go:183-189`, mirroring the existing `Config.Password` plumbing.

---

### `app.go` (MODIFIED — Wails bindings at repo root)

**Analog:** self lines 469–613. Every Wails binding follows: check `a.client == nil`, delegate to `a.client.*`, return error or value.

**Copy template** (`app.go:520-525`):
```go
// ToggleWebServing enables or disables web serving for a specific session.
// Returns an error if the web server is not running.
func (a *App) ToggleWebServing(sessionID string, enabled bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.ToggleWebServing(sessionID, enabled)
}
```

**New bindings** (mirror the template):
```go
func (a *App) IssueCapabilities(sessionID string) (daemon.IssueCapabilitiesResponse, error) {
    if a.client == nil {
        return daemon.IssueCapabilitiesResponse{}, fmt.Errorf("daemon not connected")
    }
    return a.client.IssueCapabilities(sessionID)
}

func (a *App) RegenerateSigningKey() error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.RegenerateSigningKey()
}
```

**QR generation for capability URLs** — mirror `app.go:582-613` (`GetSessionQRCode` / `GetWebServerQRCode`). The encoder call `qrcode.Encode(url, qrcode.Medium, 256)` stays exactly the same; only the input URL changes to the join-code exchange URL (`https://host/join?code=A7K-4P2N`) per D-09.

---

### `frontend/src/components/RegenerateKeyModal.tsx` (NEW — confirmation modal)

**Analog:** `frontend/src/components/QuitConfirmModal.tsx` (all 122 lines). Copy the entire file structure.

**Copy these exact patterns from QuitConfirmModal:**

**Imports + Escape handler** (`QuitConfirmModal.tsx:1-32`):
```tsx
import React, { useEffect, useRef, useState } from 'react'

interface RegenerateKeyModalProps {
  isOpen: boolean
  onConfirm: () => Promise<void>
  onCancel: () => void
}

// Escape-key pattern — copy from QuitConfirmModal.tsx:26-32:
useEffect(() => {
  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') onCancel()
  }
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [onCancel])
```

**Focus-on-open pattern** (`QuitConfirmModal.tsx:34-37`):
```tsx
const cancelBtnRef = useRef<HTMLButtonElement>(null)
useEffect(() => {
  if (isOpen) cancelBtnRef.current?.focus()
}, [isOpen])
```

**In-flight acting state** (`QuitConfirmModal.tsx:22, 104-115`):
```tsx
const [acting, setActing] = useState(false)
// Buttons disabled={acting}; label flips to "Invalidating…" while acting=true.
```

**JSX shell** (`QuitConfirmModal.tsx:56-120`) — copy `quit-modal-overlay`, `quit-modal`, `quit-modal__header`, `quit-modal__body`, `quit-modal__footer`, `quit-modal__close`, `quit-modal__btn--cancel`, `quit-modal__btn--quit-all` classes EXACTLY (UI-SPEC Surface 2 approves reuse of these classes; destructive variant already maps to `quit-modal__btn--quit-all`).

---

### `frontend/src/components/SessionSharePanel.tsx` (NEW — per-session share UI)

**Analog:** `frontend/src/components/SettingsTab.tsx` lines 495–552 (the dashboard URL action row + QR toggle block). The pattern is:
1. A row with truncated URL + three action buttons (Open/Copy/QR)
2. Inline QR image toggled by `showDashQR` state
3. Copy-with-timeout pattern → "Copied!" for 1500ms

**Imports pattern** (`SettingsTab.tsx:1-27`):
```tsx
import React, { useState } from 'react'
import { IssueCapabilities, GetCapabilityQRCode } from '../wailsjs/go/main/App'
import { BrowserOpenURL, ClipboardSetText } from '../wailsjs/wailsjs/runtime/runtime'
import {
  ArrowTopRightOnSquareIcon,
  ClipboardDocumentIcon,
  QrCodeIcon,
} from '@heroicons/react/24/outline'
```

**Copy-with-timeout pattern** (`SettingsTab.tsx:173-178`) — copy verbatim:
```tsx
async function handleCopyURL() {
  if (!serverURL) return
  await ClipboardSetText(serverURL)
  setUrlCopied(true)
  setTimeout(() => setUrlCopied(false), 1500)
}
```

**Open/Copy/QR row JSX** (`SettingsTab.tsx:514-540`) — copy the div structure; change labels from dashboard → "Read-Only Link" / "Full Access Link" per UI-SPEC Surface 1. Each row gets its own `urlCopied` / `showQR` state.

**QR toggle pattern** (`SettingsTab.tsx:180-196`):
```tsx
async function handleToggleDashQR() {
  if (showDashQR) { setShowDashQR(false); return }
  setQrError(null)
  if (!dashQRb64) {
    try {
      const b64 = await GetWebServerQRCode()
      setDashQRb64(b64)
    } catch {
      setQrError('QR unavailable — tap to retry')
      return
    }
  }
  setShowDashQR(true)
}
```

**QR image render** (`SettingsTab.tsx:542-550`):
```tsx
{showDashQR && dashQRb64 && (
  <img
    src={`data:image/png;base64,${dashQRb64}`}
    width={200}
    height={200}
    alt="QR code for dashboard URL"
    className="settings-web-server__qr"
  />
)}
```

---

### `frontend/src/components/SettingsTab.tsx` (MODIFIED — add Security section)

**Analog:** self. Append a new `<h3>Security</h3>` section after the "Web Server" block at line 583 ("Paths").

**Section header pattern** (follow `SettingsTab.tsx:315`, `343`, `371`, `387`):
```tsx
<h3>Security</h3>
<p className="settings-panel__description">
  Rotating the signing key immediately invalidates all shared links across all sessions. Use this if you suspect a link has been leaked.
</p>
<div className="settings-panel__field-group">
  <button
    className="settings-panel__btn settings-panel__btn--destructive"
    onClick={() => setShowRegenModal(true)}
  >
    Regenerate Signing Key
  </button>
</div>
<RegenerateKeyModal
  isOpen={showRegenModal}
  onConfirm={handleRegenerate}
  onCancel={() => setShowRegenModal(false)}
/>
```

**Error handling pattern** (copy `SettingsTab.tsx:305-308`):
```tsx
} catch (err) {
  setRegenError(err instanceof Error ? err.message : String(err))
}
```

---

### `web/dashboard.html` (REPLACED — landing page)

**Analog:** self (existing file, replaced in whole). Copy the head/body shell + palette + inline CSS idiom. Remove the session-list JS entirely per D-17.

**Carry forward from current `dashboard.html`:**

**Shell** (`dashboard.html:1-8`):
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>AgentHub Dashboard</title>
  <style>
```

**Palette + base styles** (`dashboard.html:9-19`) — KEEP these unchanged:
```css
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #1a1b26; color: #c0caf5; min-height: 100vh; padding: 2rem; }
h1 { color: #c0caf5; font-size: 1.5rem; margin-bottom: 1.5rem; }
h2 { color: #a9b1d6; font-size: 1.1rem; margin-bottom: 1rem; }
label { display: block; margin-bottom: 0.4rem; font-size: 0.9rem; color: #a9b1d6; }
button { padding: 0.6rem 1.2rem; background: #7aa2f7; border: none; border-radius: 4px; color: #1a1b26; font-size: 0.9rem; cursor: pointer; }
button:hover { background: #89b4fa; }
```

**Delete entirely:** `#session-list`, `.session-card`, `.qr-*`, the `refreshSessions()` and `renderSessions()` scripts at lines 103–186.

**Add:** form + input per UI-SPEC Surface 3.

---

### `web/join.html` (NEW — join flow page)

**Analog:** `web/dashboard.html` (full file) for the HTML shell and palette. Use the inline `<style>` idiom exactly — no external CSS file.

**Same head, same palette, same button styling**. The page has five state variants (A–E per UI-SPEC Surface 4); simplest implementation is five sibling `<div>` blocks keyed by server-rendered template variable OR a tiny vanilla-JS switch on query-param state.

**Form POST pattern** (new, not yet in codebase — uses RESEARCH Anti-Pattern guidance: must be POST to prevent browser prefetch):
```html
<form action="/join/exchange" method="POST">
  <input type="hidden" name="code" value="{{.Code}}">
  <button type="submit" class="join-btn">Join Session</button>
</form>
```

---

## Shared Patterns

### Pattern A — Mutex-guarded in-memory map

**Sources:**
- `internal/webserver/server.go:52-102` — `WebServer.mu` + `webEnabled` map
- `internal/relay/hub.go:37-91` — `Hub.mu` + `subscribers` map

**Apply to:**
- `WebServer.grants` (new in `server.go`)
- `JoinCodeManager.codes` (new in `capability/joincode.go`)

**Canonical excerpt** (`server.go:82-102`):
```go
func (ws *WebServer) EnableSession(sessionID string) {
    ws.mu.Lock()
    ws.webEnabled[sessionID] = true
    ws.mu.Unlock()
}
func (ws *WebServer) IsSessionEnabled(sessionID string) bool {
    ws.mu.RLock()
    ok := ws.webEnabled[sessionID]
    ws.mu.RUnlock()
    return ok
}
```

**Use `sync.Mutex` (not `RWMutex`) for `JoinCodeManager`** per RESEARCH Pitfall 4 — lookup-then-delete must be atomic.

---

### Pattern B — Atomic 0600 file write

**Source:** `internal/daemon/engine.go:132`

**Apply to:** `capability/keystore.go` `FileKeyStore.Save`

**Excerpt:**
```go
// engine.go line 132 — the canonical atomic-write idiom in this codebase.
_ = os.WriteFile(settingsPath(e.configDir), data, 0600)
```

---

### Pattern C — Missing-file-is-not-error persistence load

**Source:** `internal/daemon/engine.go:88-92`

**Apply to:** `capability/keystore.go` `FileKeyStore.Load`

**Excerpt:**
```go
data, err := os.ReadFile(settingsPath(dir))
if err != nil {
    return // file not found or unreadable — not an error
}
```

---

### Pattern D — HTTP error responses (plain-text, `http.Error`)

**Source:** `internal/webserver/server.go:354`, `373`, `388`, `296-297` and `internal/webserver/auth.go:23`

**Apply to:** all new middleware + handlers (`requireCapability`, `handleJoinExchange`, `handleIssueCapabilities`, etc.)

**Excerpt** (`auth.go:21-23`):
```go
w.Header().Set("WWW-Authenticate", `Basic realm="AgentHub"`)
http.Error(w, "Unauthorized", http.StatusUnauthorized)
return
```

**Do NOT introduce a JSON error envelope** — the existing webserver uses short plain-text messages; stay consistent (CONTEXT §Claude's Discretion on error bodies → "follow existing webserver error patterns").

---

### Pattern E — Middleware factory

**Source:** `internal/webserver/auth.go` (entire file)

**Apply to:** `internal/webserver/capability_mw.go`

Two variants coexist in this repo — pick the one matching your wrap target:
- **`func(http.Handler) http.Handler`** (used by `basicAuthMiddleware` to wrap `ws.mux` at `server.go:201`) — when wrapping the whole mux.
- **`func(http.HandlerFunc) http.HandlerFunc`** (NEW for `requireCapability`) — when wrapping individual `mux.HandleFunc` registrations per-route.

Use the HandlerFunc variant for capability checks so grant-list lookup runs AFTER path-value extraction (`r.PathValue("id")` is populated by `mux.HandleFunc` route matching).

---

### Pattern F — Unix-socket IPC handler

**Source:** `internal/daemon/api.go:501-524` (`handleWebServe`)

**Apply to:** all new capability IPC handlers in `api.go`

**Excerpt:**
```go
func (a *API) handleWebServe(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    var req WebServeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }
    a.mu.RLock()
    ws := a.webServer
    a.mu.RUnlock()
    if ws == nil {
        http.Error(w, "web server not running", http.StatusBadRequest)
        return
    }
    // ... action ...
    w.WriteHeader(http.StatusNoContent) // or writeJSON(w, http.StatusOK, resp)
}
```

---

### Pattern G — Wails binding shell

**Source:** `app.go:520-525`, `469-482`, `582-596`

**Apply to:** all new `App` methods (`IssueCapabilities`, `RegenerateSigningKey`, `GetCapabilityQRCode`, …)

**Excerpt:**
```go
func (a *App) ToggleWebServing(sessionID string, enabled bool) error {
    if a.client == nil {
        return fmt.Errorf("daemon not connected")
    }
    return a.client.ToggleWebServing(sessionID, enabled)
}
```

---

### Pattern H — React copy-to-clipboard with "Copied!" feedback

**Source:** `frontend/src/components/SettingsTab.tsx:162-178`

**Apply to:** `SessionSharePanel.tsx` (for each of read/write URLs)

**Excerpt:**
```tsx
const [urlCopied, setUrlCopied] = useState(false)

async function handleCopyURL() {
  if (!serverURL) return
  await ClipboardSetText(serverURL)
  setUrlCopied(true)
  setTimeout(() => setUrlCopied(false), 1500)
}
```

---

### Pattern I — React modal with Escape handler and focus trap

**Source:** `frontend/src/components/QuitConfirmModal.tsx:26-37`

**Apply to:** `RegenerateKeyModal.tsx`

Key excerpts already quoted above in the per-file Assignment.

---

### Pattern J — Table-driven Go test with `t.TempDir`

**Source:** `internal/daemon/engine_settings_test.go:11-99`

**Apply to:** `internal/capability/keystore_test.go`

Template quoted in the per-file Assignment.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/capability/capability.go` (Sign/Verify bodies) | crypto service | transform | No HMAC-token encoding exists in the codebase today. Use RESEARCH Pattern 1 verbatim; stylistic anchors (stdlib-only imports, terse package doc) come from `internal/relay/protocol.go` and `internal/webserver/auth.go`. |
| `web/join.html` form-POST flow (5 state variants) | view | form POST + server redirect | No existing page POSTs back to a server endpoint; `web/dashboard.html` is read-only JS-driven. Copy the HTML shell and inline-CSS pattern from `dashboard.html`, build state-variant structure from UI-SPEC Surface 4. |

Both gaps are addressed directly by the research document's code examples plus the UI-SPEC — planners should cite those sections rather than trying to find closer in-repo analogs.

---

## Metadata

**Analog search scope:**
- `/Users/ken/dev/agenthub/internal/webserver/` — full read
- `/Users/ken/dev/agenthub/internal/daemon/` — full read of `api.go`, `engine.go`, `client.go`, `types.go`, `engine_settings_test.go`
- `/Users/ken/dev/agenthub/internal/relay/hub.go` — full read
- `/Users/ken/dev/agenthub/frontend/src/components/` — `SettingsTab.tsx`, `DaemonManagerPanel.tsx`, `QuitConfirmModal.tsx`, `QRModal.tsx` — full read
- `/Users/ken/dev/agenthub/frontend/src/App.tsx` — grep for `ToggleWebServing` / `webEnabled`
- `/Users/ken/dev/agenthub/web/` — `dashboard.html` full read; `terminal.html` head + line count; `embed.go`
- `/Users/ken/dev/agenthub/app.go` — grep for all Web-related Wails bindings
- `/Users/ken/dev/agenthub/security-review/` — existing reviewer test scaffold

**Files scanned:** 17 Go files + 6 TSX files + 2 HTML files + 1 test scaffold

**Pattern extraction date:** 2026-04-20
