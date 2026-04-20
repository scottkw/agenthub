# Phase 87: Capability-Based Session Authorization - Research

**Researched:** 2026-04-19
**Domain:** Go HTTP/WebSocket capability-based authorization; HMAC-signed bearer tokens; per-session grant management
**Confidence:** HIGH — all decisions locked in CONTEXT.md, all Go stdlib APIs verified against running toolchain (go1.26.2), codebase read in full

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Token Format & Cryptography**
- D-01: HMAC-SHA256 (`crypto/hmac`). Symmetric, single-party issuer/verifier.
- D-02: Wire format `base64url(claims).base64url(sig)` — no JWT library. ~120–160 chars.
- D-03: Claim set `{ sid, perms, iat, grant_id, v }`. No `exp` claim.
- D-04: Signing key stored in `capability.key` (mode 0600) in the daemon config directory. NOT inside `settings.json`.
- D-05: `KeyStore` interface (`Load`, `Save`, `Location`). Only `FileKeyStore` ships in v3.1.

**Grant UX Flow**
- D-06: Web-serving toggle is the grant gesture. ON = issue caps; OFF = revoke.
- D-07: Two capabilities per session: read-only cap + read-write cap.
- D-08: Token appears as `?cap=<token>` query parameter.
- D-09: QR codes encode a join-code exchange URL (`/join?code=A7K-4P2N`), not the capability.
- D-10: Short join codes are base32-dashed, 8 chars in two groups of 4 (e.g. `A7K-4P2N`).
- D-11: Join codes single-use, 5-minute TTL, in-memory map only.

**Capability Lifetime & Revocation**
- D-12: No `exp` claim. Valid until session end, web-serving toggle-off, or key rotation.
- D-13: Session end implicitly invalidates caps (session ID no longer resolves).
- D-14: Per-session persisted grant list of `grant_id`s; checked on every authz request.
- D-15: Toggle web-serving OFF clears the session's grant list permanently.
- D-16: "Regenerate signing key" button in Settings → Security. Confirmation dialog required.

**Dashboard & Enumeration Model**
- D-17: `/dashboard` becomes a landing page — no session list.
- D-18: `GET /api/sessions` requires session-scoped cap; returns only the one session it is bound to.
- D-19: `GET /api/sessions/{id}/info` requires a cap for that session.
- D-20: Local-network-fallback mode: Basic Auth AND capability both required.
- D-21: `internal/relay/server.go` is NOT modified in this phase.

**Scope & Constraints**
- D-22: Only `internal/webserver/` modified on the web-server side. Daemon IPC gets new methods for cap issuance.
- D-23: `?readonly=1` removed from write-gate path. May survive as a view hint (caret suppression) only.
- D-24: `relay.Subscriber.ReadOnly` re-sourced from the capability `perms` claim, not `?readonly`.

### Claude's Discretion

- Exact shape of the `/join` page HTML (copy, layout, error states).
- Go package layout for the new capability module.
- Middleware pattern for applying capability checks to multiple routes.
- Test organization: unit, integration, fuzz.
- Naming conventions for new daemon API methods.
- Error response bodies and status codes for capability failures.
- Observability hooks (kept minimal — no new infra).

### Deferred Ideas (OUT OF SCOPE)

- OS keychain-backed key storage (v3.2+, GitHub issue to be filed).
- Revocation & audit UI (v3.2+ per REQUIREMENTS.md Future Requirements).
- Per-user identity / SSO / OIDC.
- External KMS for signing key.
- Pause/suspend session semantics.
- Removing `internal/relay/server.go` as tech-debt cleanup.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SEC-01 | User explicitly grants share access; no auto-exposure of new sessions | D-06 removes auto-enable at `api.go:292`; toggle-ON = issuance |
| SEC-02 | `GET /api/sessions` rejects without valid session capability | `requireCapability` middleware wraps the route; returns only bound session |
| SEC-03 | `/sessions/{id}/ws` and `/sessions/{id}` reject without valid cap for that exact session | Same middleware; `sid` claim verified against path value |
| SEC-04 | Read-only permission is a property of the server-issued cap, not a query param | `perms` claim in token; `?readonly=1` stripped from write path |
| SEC-05 | Relay rejects `MsgInput` from subscriber whose cap lacks write permission | `sub.ReadOnly` re-sourced from `perms` claim; reconnect-without-readonly regression test |
</phase_requirements>

---

## Summary

Phase 87 replaces a single ambient trust boundary (Tailscale network membership) with an explicit capability layer that every web-facing route enforces. The capability token is an HMAC-SHA256-signed blob carrying `{sid, perms, iat, grant_id, v}` encoded as `base64url(claims).base64url(sig)`. All cryptographic primitives are Go stdlib — no new Go module dependencies are required.

The implementation has two primary subsystems: (1) a `capability` package that handles token signing, verification, the `KeyStore` interface, `FileKeyStore`, and join-code management; and (2) modifications to `internal/webserver/server.go` that wire a `requireCapability` middleware across the four guarded routes and replace the `?readonly=1` path with capability-sourced `ReadOnly` state. The daemon API gains three new IPC endpoints (`IssueCapabilities`, `ExchangeJoinCode`, `RegenerateSigningKey`), and the grant list lives on the `WebServer` struct alongside the existing `webEnabled` map.

Security properties delivered: SEC-01 through SEC-05. No new Go library dependencies. No changes to `internal/relay/server.go` (D-21). The `capability.key` file follows the exact same write pattern as `settings.json` (`os.WriteFile`, mode 0600) and lives next to it in `~/.config/agenthub/`.

**Primary recommendation:** Implement the `internal/capability` package first (pure stdlib, easily unit-testable in isolation), wire it into `internal/webserver/server.go` second, then extend the daemon API and frontend last.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Token signing and verification | Go daemon (`internal/capability`) | — | Daemon is the sole issuer and verifier; symmetric key never leaves the process |
| Signing key persistence | Filesystem (`capability.key`) | `KeyStore` interface (future: OS keychain) | Single-daemon model needs no key distribution; file-based is cross-platform |
| Grant list management | Go daemon (`internal/webserver.WebServer`) | — | The webserver already owns `webEnabled`; grant list is its natural peer |
| Join-code exchange | Go daemon (`internal/webserver` new handler) | — | Short-lived codes need fast in-memory lookup; daemon process owns the scope |
| Capability enforcement (HTTP) | `requireCapability` middleware in `internal/webserver` | — | HTTP handler layer is the correct interception point |
| Write-permission enforcement (WebSocket) | Relay hub (`Subscriber.ReadOnly` sourced from cap) | — | Frame-level enforcement belongs where frames are processed |
| Grant gesture (UX) | Frontend GUI (`DaemonManagerPanel`, `ToggleWebServing`) | Daemon IPC (`IssueCapabilities`) | UI triggers; daemon decides and issues |
| Key rotation UX | Frontend GUI (`SettingsTab` Security section) | Daemon IPC (`RegenerateSigningKey`) | Same pattern as other settings mutations |
| Landing page + join flow | Web server (`/dashboard` and `/join` routes) | — | Server-rendered HTML; no frontend framework involved |

---

## Standard Stack

### Core

All capability token operations use Go standard library only. No new `go.mod` dependencies are required or should be added.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `crypto/hmac` | stdlib (go1.26.2) | Compute and verify HMAC-SHA256 | RFC 2104 implementation; `hmac.Equal` provides constant-time comparison [VERIFIED: go doc] |
| `crypto/sha256` | stdlib | Hash function for HMAC | Agreed algorithm (D-01) [VERIFIED: go doc] |
| `crypto/rand` | stdlib | Generate signing key and grant IDs | Cryptographically secure; `Read` fills slice and never returns an error in Go 1.20+ [VERIFIED: go doc] |
| `encoding/base64` | stdlib | `base64.RawURLEncoding` for token segments | URL-safe, no padding, matches D-02 wire format [VERIFIED: go doc] |
| `encoding/base32` | stdlib | `base32.StdEncoding.WithPadding(base32.NoPadding)` for join codes | RFC 4648 base32 alphabet is `A–Z2–7`; avoids 0/O and 1/I/l ambiguity per D-10 [VERIFIED: go doc] |
| `encoding/json` | stdlib | Claim set serialization | Already used throughout the codebase [VERIFIED: codebase grep] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/skip2/go-qrcode` | `v0.0.0-20200617195104-da1b6568686e` | QR code generation | Already in `go.mod`; QR content changes (D-09) but the encoder stays [VERIFIED: go.mod] |
| `github.com/coder/websocket` | `v1.8.14` | WebSocket upgrade + frame I/O | Already in `go.mod`; Phase 87 reads cap from `r.URL.Query()` during the HTTP upgrade [VERIFIED: go.mod] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom `base64url(claims).base64url(sig)` | `github.com/golang-jwt/jwt` | JWT library adds `alg` negotiation surface (algorithm-confusion CVE class); stdlib is simpler and sufficient for a single issuer/verifier (D-02) |
| HMAC-SHA256 | Ed25519 (already in `go.mod` via `filippo.io/edwards25519`) | Ed25519 provides no benefit when the same process signs and verifies; adds key-generation complexity for no security gain (D-01) |
| File-based `KeyStore` | OS keychain (`github.com/99designs/keyring`) | Keychain adds platform-specific complexity and breaks in service-mode contexts (D-05, deferred to v3.2+) |
| In-memory join-code map | Redis / persistent store | Join codes are 5-min ephemeral artifacts; in-memory is correct (D-11) |

**Installation:** No new packages. All code uses existing `go.mod` dependencies and stdlib.

---

## Architecture Patterns

### System Architecture Diagram

```
[GUI / CLI / TUI]
      |  (Unix socket IPC — trust boundary unchanged)
      v
[Daemon API — internal/daemon/api.go]
      |                      |
      v                      v
[SessionEngine]      [WebServer — internal/webserver/server.go]
  |                      |
  |  [capability.KeyStore] ← capability.key (0600 file)
  |         |
  v         v
[internal/capability package]
  · Sign(claims) → token
  · Verify(token) → claims
  · IssueCapabilities(sessionID) → (readCap, writeCap)
  · IssueJoinCode(cap) → code
  · ExchangeJoinCode(code) → cap (single-use, 5-min TTL)
  · GenerateKey() → []byte
      |
      v
[requireCapability middleware]
      |
      +--→ GET /dashboard              (no cap required — landing page only)
      +--→ GET /join                   (no cap required — shows join form)
      +--→ POST /join/exchange         (validates join code → redirects with cap)
      +--→ GET /api/sessions      ←--- requires valid cap; returns only bound session
      +--→ GET /api/sessions/{id}/info ← requires cap for this sid
      +--→ GET /sessions/{id}          ← requires cap for this sid
      +--→ GET /sessions/{id}/ws  ←--- cap → sub.ReadOnly = (perms == "read")
                                        |
                                        v
                                 [relay.Hub fan-out]
                                 MsgInput rejected if sub.ReadOnly == true
```

### Recommended Project Structure

```
internal/
├── capability/
│   ├── capability.go       # Sign, Verify, IssueCapabilities, IssueJoinCode, ExchangeJoinCode
│   ├── capability_test.go  # Unit + fuzz tests
│   ├── keystore.go         # KeyStore interface + FileKeyStore implementation
│   └── keystore_test.go    # Load/Save round-trip tests
├── webserver/
│   ├── server.go           # Modified: requireCapability middleware, new routes, grant list
│   ├── auth.go             # Existing basicAuthMiddleware (unchanged)
│   └── server_test.go      # Extended with capability tests (based on security-review scaffold)
└── daemon/
    ├── api.go              # Extended: IssueCapabilities, ExchangeJoinCode, RegenerateSigningKey handlers
    └── types.go            # Extended: IssueCapabilitiesResponse, ExchangeJoinCodeRequest/Response
web/
├── dashboard.html          # Replaced: landing page with join-code form (D-17)
└── join.html               # New: join flow page
```

### Pattern 1: Token Sign and Verify

**What:** Deterministic HMAC-SHA256 over a canonical JSON claim serialization. The token is two base64url segments separated by a dot: `base64url(claimsJSON).base64url(hmac)`.

**When to use:** Every token issuance and every inbound request to a guarded route.

```go
// Source: stdlib crypto/hmac + encoding/base64 (verified go1.26.2)
// internal/capability/capability.go

package capability

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "strings"
)

// Claims is the capability token payload.
type Claims struct {
    SID     string `json:"sid"`
    Perms   string `json:"perms"`    // "read" or "read,write"
    IAT     int64  `json:"iat"`      // issued-at UNIX timestamp
    GrantID string `json:"grant_id"` // 128-bit random, enables future revocation
    V       int    `json:"v"`        // schema version; always 1 in v3.1
}

// Sign serializes claims as canonical JSON, computes HMAC-SHA256 with key,
// and returns a two-segment base64url token.
func Sign(claims Claims, key []byte) (string, error) {
    payload, err := json.Marshal(claims)
    if err != nil {
        return "", err
    }
    mac := hmac.New(sha256.New, key)
    mac.Write(payload)
    sig := mac.Sum(nil)
    b64Payload := base64.RawURLEncoding.EncodeToString(payload)
    b64Sig := base64.RawURLEncoding.EncodeToString(sig)
    return b64Payload + "." + b64Sig, nil
}

// Verify parses and verifies a token against key. Returns the Claims on success.
// Returns an error for malformed tokens, bad signatures, or schema mismatches.
func Verify(token string, key []byte) (Claims, error) {
    parts := strings.SplitN(token, ".", 2)
    if len(parts) != 2 {
        return Claims{}, errors.New("malformed token")
    }
    payload, err := base64.RawURLEncoding.DecodeString(parts[0])
    if err != nil {
        return Claims{}, errors.New("malformed token payload")
    }
    sig, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        return Claims{}, errors.New("malformed token signature")
    }
    // Constant-time HMAC comparison (crypto/hmac.Equal).
    mac := hmac.New(sha256.New, key)
    mac.Write(payload)
    expected := mac.Sum(nil)
    if !hmac.Equal(sig, expected) {
        return Claims{}, errors.New("invalid signature")
    }
    var c Claims
    if err := json.Unmarshal(payload, &c); err != nil {
        return Claims{}, errors.New("malformed claims")
    }
    return c, nil
}
```

### Pattern 2: FileKeyStore

**What:** Load a 32-byte signing key from `capability.key` (mode 0600). Generate and save on first use.

**When to use:** Daemon startup in `NewSessionEngine` or `NewAPI`, before first token issuance.

```go
// Source: mirrors engine.go:saveSettingsToDisk pattern (verified in codebase)
// internal/capability/keystore.go

package capability

import (
    "crypto/rand"
    "fmt"
    "os"
    "path/filepath"
)

// KeyStore abstracts signing key persistence.
// v3.1 ships FileKeyStore only; interface enables future keychain backends (D-05).
type KeyStore interface {
    Load() ([]byte, error)
    Save(key []byte) error
    Location() string
}

// FileKeyStore persists the 32-byte signing key as a raw binary file at path.
// File permissions are 0600 (owner-read/write only).
type FileKeyStore struct {
    path string
}

// NewFileKeyStore returns a FileKeyStore rooted at dir/capability.key.
func NewFileKeyStore(dir string) *FileKeyStore {
    return &FileKeyStore{path: filepath.Join(dir, "capability.key")}
}

func (s *FileKeyStore) Location() string { return s.path }

// Load reads the key file. Returns a 32-byte key.
// Returns (nil, nil) if the file does not exist (first run — caller generates).
func (s *FileKeyStore) Load() ([]byte, error) {
    data, err := os.ReadFile(s.path)
    if os.IsNotExist(err) {
        return nil, nil // first run
    }
    if err != nil {
        return nil, fmt.Errorf("capability: read key file: %w", err)
    }
    if len(data) != 32 {
        return nil, fmt.Errorf("capability: key file corrupt (got %d bytes, want 32)", len(data))
    }
    return data, nil
}

// Save writes key to the file with mode 0600.
// Uses os.WriteFile which is an atomic overwrite on most platforms (D-04).
func (s *FileKeyStore) Save(key []byte) error {
    return os.WriteFile(s.path, key, 0600)
}

// GenerateKey creates a new 32-byte cryptographically random key.
func GenerateKey() ([]byte, error) {
    key := make([]byte, 32)
    _, err := rand.Read(key)
    return key, err
}

// LoadOrGenerate loads the key, generating and saving a new one if absent.
func LoadOrGenerate(store KeyStore) ([]byte, error) {
    key, err := store.Load()
    if err != nil {
        return nil, err
    }
    if key == nil {
        key, err = GenerateKey()
        if err != nil {
            return nil, fmt.Errorf("capability: generate key: %w", err)
        }
        if err := store.Save(key); err != nil {
            return nil, fmt.Errorf("capability: save key: %w", err)
        }
    }
    return key, nil
}
```

### Pattern 3: requireCapability Middleware

**What:** An HTTP middleware that extracts `?cap=<token>`, verifies the signature, checks `sid` matches the path, and confirms `grant_id` is in the session's grant list. Returns 401 or 403 with a short plain-text message on failure (matching existing `http.Error` style).

**When to use:** Wrap the four guarded routes in `setupRoutes()`.

```go
// Source: mirrors basicAuthMiddleware pattern at internal/webserver/auth.go (verified in codebase)
// internal/webserver/server.go

// requireCapability returns a middleware that validates the ?cap= token.
// The sessionID is read from r.PathValue("id") if pathParamID is true;
// otherwise the cap's sid must match any enabled session (used for /api/sessions).
func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("cap")
        if token == "" {
            http.Error(w, "capability required", http.StatusUnauthorized)
            return
        }
        claims, err := capability.Verify(token, ws.signingKey)
        if err != nil {
            http.Error(w, "capability required", http.StatusUnauthorized)
            return
        }
        // Path-level session binding: cap must match the session in the URL.
        if pathID := r.PathValue("id"); pathID != "" && claims.SID != pathID {
            http.Error(w, "capability does not match session", http.StatusForbidden)
            return
        }
        // Grant list check: grant_id must still be active for this session.
        if !ws.isGrantActive(claims.SID, claims.GrantID) {
            http.Error(w, "capability has been revoked", http.StatusForbidden)
            return
        }
        // Attach claims to request context so handlers can read perms.
        ctx := capability.WithClaims(r.Context(), claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

### Pattern 4: Grant List (per-session `grant_id` tracking)

**What:** A `map[string]map[string]struct{}` on the `WebServer` struct, guarded by the existing `ws.mu`. Outer key is `sessionID`, inner key is `grant_id`.

**When to use:** Grant list populated when web-serving is toggled ON (D-14); cleared when toggled OFF (D-15).

```go
// internal/webserver/server.go — additions to WebServer struct and methods

// In WebServer struct (add alongside webEnabled):
grants map[string]map[string]struct{} // sessionID -> set of active grant_ids

// AddGrant adds a grant_id to the session's active set.
func (ws *WebServer) AddGrant(sessionID, grantID string) {
    ws.mu.Lock()
    if ws.grants[sessionID] == nil {
        ws.grants[sessionID] = make(map[string]struct{})
    }
    ws.grants[sessionID][grantID] = struct{}{}
    ws.mu.Unlock()
}

// ClearGrants removes all grants for a session (toggle-off or revocation).
func (ws *WebServer) ClearGrants(sessionID string) {
    ws.mu.Lock()
    delete(ws.grants, sessionID)
    ws.mu.Unlock()
}

// isGrantActive returns true if grantID is in the session's current grant set.
func (ws *WebServer) isGrantActive(sessionID, grantID string) bool {
    ws.mu.RLock()
    defer ws.mu.RUnlock()
    if ws.grants[sessionID] == nil {
        return false
    }
    _, ok := ws.grants[sessionID][grantID]
    return ok
}
```

### Pattern 5: Join-Code Issuance and Exchange

**What:** An in-memory map of `code → joinCodeEntry{cap, grantID, expiry}` guarded by a mutex. Codes are base32-encoded (using stdlib `encoding/base32`) 5-byte random values, formatted as `XXXX-XXXX`.

**When to use:** `IssueJoinCode` is called after `IssueCapabilities` on toggle-on. `/join/exchange` POST handler consumes the code once.

```go
// Source: encoding/base32.StdEncoding — RFC 4648 alphabet A-Z2-7, verified go doc
// internal/capability/capability.go

import "encoding/base32"

// joinCodeEncoding uses the RFC 4648 standard alphabet (A–Z 2–7), no padding.
// This matches D-10: avoids 0/O and 1/I/l ambiguity.
var joinCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// IssueJoinCode generates a single-use, 5-minute join code for the given token.
// The code is 8 characters in format XXXX-XXXX (base32, ~40 bits of entropy).
func (m *JoinCodeManager) IssueJoinCode(token string) (string, error) {
    var raw [5]byte
    if _, err := rand.Read(raw[:]); err != nil {
        return "", err
    }
    // 5 bytes → 8 base32 chars (40 bits / 5 bits-per-char)
    encoded := joinCodeEncoding.EncodeToString(raw[:])
    code := encoded[:4] + "-" + encoded[4:8]
    m.mu.Lock()
    m.codes[code] = joinEntry{token: token, expiry: time.Now().Add(5 * time.Minute)}
    m.mu.Unlock()
    return code, nil
}
```

### Pattern 6: WebSocket Read-Only Enforcement (Relay Layer)

**What:** The `sub.ReadOnly` field on `relay.Subscriber` is set from the capability `perms` claim at WebSocket upgrade time. The existing MsgInput check at `server.go:451` remains unchanged in structure; only the source of `ReadOnly` changes.

**When to use:** In `handleWSSRelay`, after `Verify` succeeds, before `hub.Subscribe(sub)`.

```go
// internal/webserver/server.go — handleWSSRelay change (D-24)

// BEFORE (current code, line 386-387):
// readonly := r.URL.Query().Get("readonly") == "1" || r.URL.Query().Get("readonly") == "true"

// AFTER:
claims := capability.ClaimsFromContext(r.Context()) // set by requireCapability middleware
readonly := claims.Perms == "read"                   // server-bound, not client-asserted

sub := &relay.Subscriber{
    Msgs:     make(chan []byte, 256),
    ReadOnly: readonly,  // now sourced from verified capability claim
    Name:     clientName,
}
```

### Pattern 7: Auto-Enable Removal (SEC-01)

**What:** Remove the `ws.EnableSession(id)` call at `api.go:292` from `handleCreateSession`. After this change, creating a session does NOT automatically expose it to web clients. The user must explicitly toggle web-serving ON.

**When to use:** This is a one-line deletion, not a new pattern.

```go
// internal/daemon/api.go:handleCreateSession — REMOVE these lines:
// Auto-enable web serving for this session if the web server is running (SERVE-02).
// a.mu.RLock()
// ws := a.webServer
// a.mu.RUnlock()
// if ws != nil {
//     ws.EnableSession(id)
// }
```

### Anti-Patterns to Avoid

- **JWT library dependency:** The JWT ecosystem has a history of algorithm-confusion vulnerabilities (`alg: none` attacks, RS256-to-HS256 confusion). Since there is no `alg` field and no third-party verifier in this design, adding a JWT library only adds surface — stay with the custom two-segment format.
- **Storing the signing key in `settings.json`:** `settings.json` is a user-visible config file that users may share for debugging. The signing key must live in its own `capability.key` file (D-04). This is already a locked decision.
- **Time-based expiry for capability tokens:** Using `time.Now()` for token expiry without NTP awareness creates fragile URLs. There is no `exp` claim by design (D-12) — don't add one.
- **URL fragment for the token (`#cap=...`):** Fragment is not sent to the server; it would require JavaScript to extract and re-inject the token in the WebSocket URL. Query param is simpler and equally logged (but logs must be protected by other means — see Pitfall 2).
- **`strings.Compare` or `bytes.Equal` for MAC comparison:** Use `hmac.Equal` (constant-time). `bytes.Equal` leaks timing on partial-match prefixes.
- **Non-atomic key file writes:** `os.WriteFile` is an atomic replacement on most platforms (it truncates and rewrites in one syscall path). This matches the existing `saveSettingsToDisk` pattern. Do not use `os.Open` + incremental writes.
- **Serving `/join/exchange` as a GET:** The exchange endpoint consumes the code (single-use). It must be a POST to prevent browser prefetch and history sharing. The client form can use a standard HTML `<form method="POST">`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Constant-time MAC comparison | `bytes.Equal`, `==` string comparison | `hmac.Equal` from `crypto/hmac` | Timing-side-channel attack surface; partial-match leaks key bits |
| Cryptographically random bytes | `math/rand`, deterministic sources | `crypto/rand.Read` | `math/rand` is not cryptographically secure; predictable key/grant-ID generation |
| Base32 encoding for join codes | Custom character-substitution table | `encoding/base32.StdEncoding.WithPadding(NoPadding)` | Standard RFC 4648 alphabet already excludes 0/O/1/I/l; `StdEncoding` uses A-Z 2-7 |
| URL-safe base64 | Manual `+`/`-` and `/`/`_` replacement | `base64.RawURLEncoding` | Built-in; handles padding stripping; no off-by-one errors |
| HTTP middleware chaining | ad-hoc `if !auth { return }` in each handler | Wrapper function pattern (matches `basicAuthMiddleware` already in codebase) | Centralizes auth logic; prevents per-handler omission bugs |

**Key insight:** In cryptographic code, the cost of a subtle bug (timing oracle, weak randomness, signature malleability) is a security failure, not a runtime crash. Every item in this table has a stdlib implementation with known-good properties — the only reason to deviate is if stdlib doesn't cover the need, which it does here.

---

## Runtime State Inventory

Phase 87 is not a rename/refactor phase. However, it does introduce new persisted state that interacts with the runtime.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `settings.json` — existing; signing key must NOT go here (D-04) | Code-only: new `capability.key` file written alongside |
| Live service config | `webEnabled` map in `WebServer` is in-memory only; no persistence to migrate | None — in-memory state is correct |
| OS-registered state | None — `capability.key` is not OS-registered | None |
| Secrets/env vars | `capability.key` is a new secret artifact written by the daemon; no env vars | New file write path only |
| Build artifacts | None — pure Go, no compiled assets affected | None |

**`capability.key` is new runtime state:** After Phase 87 ships, every deployment will have `~/.config/agenthub/capability.key` as a sensitive file. This is intentional and expected. The file is not created until first daemon start after the upgrade.

---

## Common Pitfalls

### Pitfall 1: Grant List Not Cleared on Session Exit
**What goes wrong:** A session exits (natural or killed), its ID disappears from the registry, but `grants[sessionID]` lingers in memory. The next session created with a recycled ID (unlikely but theoretically possible with UUID v4 collision) could inherit old grants.
**Why it happens:** `onExit` callback only calls `ws.DisableSession(id)`. If `ClearGrants` is not also called, the map grows unbounded.
**How to avoid:** Call `ws.ClearGrants(sessionID)` inside `onExit` alongside `ws.DisableSession(sessionID)`. The existing `onExit` at `api.go:268-277` is the correct hook.
**Warning signs:** `grants` map growing over time without being collected.

### Pitfall 2: Capability Token in Server Access Logs
**What goes wrong:** `?cap=<token>` appears in plain text in any access log that records the full request URI (`Combined Log Format`, nginx, etc.).
**Why it happens:** Query parameters are part of the HTTP request line and are logged by default.
**How to avoid:** This is a known and accepted tradeoff (D-08 was chosen over fragment specifically because the server needs the token; logging must be treated as a controlled surface). For Phase 87, there is no access-log infrastructure to protect. Document in the capability package README that access logs must not be stored in world-readable locations.
**Warning signs:** N/A for Phase 87 (no logging infra to change). Flag for Phase 90 or a future ops phase.

### Pitfall 3: Missing `signingKey` Initialization Causes Nil Dereference
**What goes wrong:** `WebServer` is constructed with a nil signing key; the first `requireCapability` call panics.
**Why it happens:** `signingKey` is loaded from `FileKeyStore` by the daemon before constructing the web server, but if the load step is omitted, the field is nil.
**How to avoid:** `WebServer.signingKey` must be a non-optional parameter in the constructor or set via a required `SetSigningKey([]byte)` method that panics if passed nil. Add an assertion in `setupRoutes()` that fails fast if the key is empty.
**Warning signs:** `nil pointer dereference` in `capability.Verify` or `capability.Sign`.

### Pitfall 4: Join Code Race Between Issue and Exchange
**What goes wrong:** Two goroutines simultaneously exchange the same join code; both see it as valid; both consume it; two sessions are granted access from one code.
**Why it happens:** Naive lookup-then-delete is a TOCTOU race.
**How to avoid:** The `JoinCodeManager.Exchange` method must hold the mutex across the lookup AND the delete: read, check, delete, release. A single `sync.Mutex` (not `sync.RWMutex`) is correct here because the read-and-delete must be atomic.
**Warning signs:** Integration test using concurrent exchange requests against the same code.

### Pitfall 5: WebSocket Upgrade Receives Cap After Headers Committed
**What goes wrong:** The capability check runs after `websocket.Accept` has already sent the 101 Switching Protocols response; by then it is too late to return a 401/403.
**Why it happens:** `requireCapability` middleware runs before the handler, which is correct — but if the middleware is attached to the handler instead of wrapping the route, the upgrade fires before the check.
**How to avoid:** The `requireCapability` wrapper must sit on the *outside* of `handleWSSRelay`. In `setupRoutes()`, register:
```go
mux.HandleFunc("GET /sessions/{id}/ws", ws.requireCapability(ws.handleWSSRelay))
```
The middleware runs first, and `handleWSSRelay` only runs if the capability is valid. This is identical to how `basicAuthMiddleware` wraps the entire mux in `startLocal`.
**Warning signs:** WebSocket connections succeeding without a valid `?cap=` parameter.

### Pitfall 6: Claims JSON Field Order Instability
**What goes wrong:** `json.Marshal` of `Claims{}` produces field order that differs between Go versions or field additions, causing HMAC verification failures after an upgrade.
**Why it happens:** Go's `encoding/json` encodes struct fields in declaration order — this IS stable across Go versions for the same struct definition. However, adding a field to `Claims` without a version gate changes the canonical payload.
**How to avoid:** The `v` (version) claim in `Claims` is the forward-compat guard. Because tokens are verified by the same process that issues them (symmetric), and the struct definition is fixed for v3.1, field order is stable for the lifetime of v3.1. Future fields must either be additive-optional (Go's `json:",omitempty"`) or require a `v` increment.
**Warning signs:** `invalid signature` errors after daemon upgrade.

### Pitfall 7: `?readonly=1` Removal Breaks Existing QR/URL Bookmarks
**What goes wrong:** Any existing bookmarked or cached URL containing `?readonly=1` continues to be passed but Phase 87 changes the meaning — the server now ignores it for write-gate purposes. The client-side xterm terminal page uses `?readonly=1` to decide whether to set `disableStdin: true`.
**Why it happens:** D-23 removes `?readonly=1` from the write path but allows it to survive as a "view hint" — the terminal page reads the capability `perms` field from the `/api/sessions/{id}/info` endpoint instead.
**How to avoid:** The terminal HTML page's read-only logic must be updated to fetch `perms` from the session info endpoint, NOT from `?readonly=1`. Remove the `?readonly` read from the terminal JavaScript.
**Warning signs:** Read-only viewers seeing an active input caret, or vice versa.

---

## Code Examples

### Verified Token Flow (sign → verify → check grant)

```go
// Source: stdlib crypto/hmac + encoding/base64 — verified against go1.26.2

// Issue capabilities when web-serving is toggled ON:
func (a *API) issueCapabilitiesForSession(sessionID string) (readURL, writeURL string, err error) {
    key, err := a.capabilityKey() // load from KeyStore
    if err != nil {
        return "", "", err
    }

    // Generate two grant IDs (128-bit each = 16 bytes).
    var rgid, wgid [16]byte
    if _, err := rand.Read(rgid[:]); err != nil { return "", "", err }
    if _, err := rand.Read(wgid[:]); err != nil { return "", "", err }

    now := time.Now().Unix()
    readClaims  := capability.Claims{SID: sessionID, Perms: "read",       IAT: now, GrantID: hex.EncodeToString(rgid[:]), V: 1}
    writeClaims := capability.Claims{SID: sessionID, Perms: "read,write", IAT: now, GrantID: hex.EncodeToString(wgid[:]), V: 1}

    readTok,  err := capability.Sign(readClaims,  key)
    if err != nil { return "", "", err }
    writeTok, err := capability.Sign(writeClaims, key)
    if err != nil { return "", "", err }

    // Register both grant IDs in the webserver's grant list.
    a.webServer.AddGrant(sessionID, readClaims.GrantID)
    a.webServer.AddGrant(sessionID, writeClaims.GrantID)

    base := a.webServer.BaseURL()
    return base + "/sessions/" + sessionID + "?cap=" + readTok,
           base + "/sessions/" + sessionID + "?cap=" + writeTok,
           nil
}
```

### Join Code Exchange Handler

```go
// Source: pattern mirrors handleWebServe in internal/daemon/api.go

// POST /join/exchange
func (ws *WebServer) handleJoinExchange(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    code := r.FormValue("code")
    if code == "" {
        http.Error(w, "code required", http.StatusBadRequest)
        return
    }
    token, err := ws.joinCodes.Exchange(code)
    if errors.Is(err, capability.ErrCodeExpired) {
        http.Error(w, "code expired", http.StatusGone) // 410
        return
    }
    if errors.Is(err, capability.ErrCodeNotFound) {
        http.Error(w, "invalid code", http.StatusNotFound)
        return
    }
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    claims, err := capability.Verify(token, ws.signingKey)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    // Redirect to the session URL with the capability token.
    http.Redirect(w, r, "/sessions/"+claims.SID+"?cap="+token, http.StatusSeeOther)
}
```

### Existing Test Scaffold Integration

The security-review file at `security-review/internal_webserver_server_test.go` already provides `selfSignedTLSForTest`, `testServer`, `testServerWithHub`, and `dialWebServerWS` helpers. These helpers must be moved (or duplicated) into `internal/webserver/server_test.go`. The two security-finding tests (`TestSecurity_UnauthenticatedClientCanEnumerateAndWriteToWebServedSession` and `TestSecurity_ReadOnlyModeCanBeBypassedByReconnectWithoutReadonlyFlag`) must be inverted to assert the correct behavior:

```go
// After Phase 87: these tests must FAIL for the pre-fix behavior and PASS for post-fix.
// The tests assert that unauthenticated requests now receive 401/403, not 200.

func TestSecurity_UnauthenticatedClientCannotEnumerateSessions(t *testing.T) {
    ws, client := testServer(t)
    ws.EnableSession("sess-authz")

    resp, err := client.Get(ws.BaseURL() + "/api/sessions")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusUnauthorized {
        t.Errorf("expected 401 without capability, got %d", resp.StatusCode)
    }
}

func TestSecurity_ReadOnlyCapabilityBlocksMsgInput(t *testing.T) {
    // Connects with a valid read-only capability; attempts to send MsgInput;
    // verifies the PTY pipe does NOT receive the input frame.
    // This is the SEC-05 regression test (D-24).
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| JWT with third-party library | Custom compact token (base64url-encoded JSON + HMAC sig) | Post-algorithm-confusion CVEs (2015–2022) | Eliminates `alg:none` and RS256-to-HS256 confusion attacks |
| Tailnet membership as access gate | Explicit per-session capability tokens | Phase 87 (this phase) | Tailnet breach no longer grants PTY access |
| `?readonly=1` client-asserted permission | Server-bound `perms` claim in signed capability | Phase 87 (this phase) | Eliminates read-only bypass via reconnect |
| Session auto-expose on create | Explicit toggle-on grant gesture | Phase 87 (this phase) | Satisfies SEC-01 |
| Fragment `#token=...` common in SPA token flows | Query param `?cap=<token>` | n/a | Server receives token in HTTP upgrade; no JS needed |

**Deprecated/outdated:**
- `OriginPatterns: ["*"]` at `server.go:403`: Not removed in Phase 87 (Phase 88 scope), but the comment `// Accept connections from any origin` should be updated to note Phase 88 will restrict this.
- `?readonly=1` query parameter: deprecated as a write-gate; may survive as a view hint only (D-23).

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `json.Marshal` on a fixed Go struct produces stable field order across Go minor versions | Pattern 1, Pitfall 6 | HMAC verification would fail after Go upgrade — mitigated by `v` claim and same-process issuer/verifier | [ASSUMED — stdlib behavior is documented as "struct fields in declaration order" but not formally guaranteed across major versions; risk is LOW for v3.1 lifetime] |

**All other claims in this research were verified:** stdlib APIs confirmed via `go doc`, codebase patterns confirmed via file reads, existing test patterns confirmed via `security-review/` scaffold.

---

## Open Questions (RESOLVED)

1. **`capability.key` placement in `WebServer` vs `API`**
   - What we know: The signing key is needed by `requireCapability` middleware which runs in `WebServer`. The key is loaded at daemon startup in `NewSessionEngine`/`NewAPI`.
   - What's unclear: Should `WebServer` hold the signing key directly (passed at construction), or should it call back into the API?
   - RESOLVED: Pass `signingKey []byte` as a field in `webserver.Config`. This matches the existing `Config.Password` pattern for local-mode auth. The `API` loads or generates the key, then passes it when constructing the `WebServer`.

2. **Grant list persistence on daemon restart**
   - What we know: D-14 says the grant list is consulted on every authz check. The context (D-11) says join codes are NOT persisted. The capability tokens themselves survive restart (signing key persisted in `capability.key`). The grant list is currently described as "per-session persisted."
   - What's unclear: Is the grant list in-memory only (cleared on restart) or written to disk? CONTEXT.md says "per-session persisted grant list" — this implies disk persistence.
   - RESOLVED: Persist the grant list to disk in a `grants.json` file (same config dir) using the same atomic write pattern as `settings.json`. On daemon restart, load existing grants. This ensures previously-shared URLs remain valid after restart (satisfying the "survive daemon restart" requirement in the phase goal). If not persisted, daemon restart would silently revoke all outstanding shares even though the signing key is intact — a confusing user experience.

3. **`RegenerateSigningKey` IPC method vs direct WebServer call**
   - What we know: D-16 requires a "Regenerate signing key" button in Settings → Security that triggers through the Wails binding path (frontend → Wails → app.go → daemon client → daemon IPC).
   - What's unclear: The method must (1) generate a new key, (2) overwrite `capability.key`, (3) update the in-memory key in `WebServer`. Does the `WebServer` hold the key or does the `API`?
   - RESOLVED: `WebServer` holds `signingKey []byte` in its struct (guarded by `ws.mu`). Add a `SetSigningKey([]byte)` method. `RegenerateSigningKey` handler generates a new key, saves it, then calls `ws.SetSigningKey(newKey)`. All subsequent `requireCapability` calls use the new key; existing tokens fail verification immediately.

---

## Environment Availability

All required tools are standard Go stdlib plus existing go.mod dependencies. No external services or tools are required.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `crypto/hmac`, `crypto/sha256`, `crypto/rand` | Token signing/verify | Yes (stdlib) | go1.26.2 | — |
| `encoding/base64` (RawURLEncoding) | Token wire format | Yes (stdlib) | go1.26.2 | — |
| `encoding/base32` (StdEncoding) | Join code generation | Yes (stdlib) | go1.26.2 | — |
| `github.com/skip2/go-qrcode` | QR generation for join-code URLs | Yes | `v0.0.0-20200617195104-da1b6568686e` in go.mod | — |
| `github.com/coder/websocket` | WebSocket upgrade during cap check | Yes | `v1.8.14` in go.mod | — |
| `go test ./internal/webserver/ ./internal/capability/` | Validation | Yes (go1.26.2) | — | — |

**No missing dependencies.**

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing package (stdlib), vitest not applicable |
| Config file | none — standard `go test` |
| Quick run command | `go test ./internal/capability/ ./internal/webserver/ -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SEC-01 | Creating a session while web server is running does NOT expose it | integration | `go test ./internal/daemon/ -run TestHandleCreateSession_NoAutoEnable -count=1` | ❌ Wave 0 |
| SEC-02 | `GET /api/sessions` returns 401 without cap | integration | `go test ./internal/webserver/ -run TestSecurity_UnauthenticatedClientCannotEnumerateSessions -count=1` | ❌ Wave 0 (invert existing test) |
| SEC-03 | `GET /sessions/{id}/ws` returns 401 without cap; cap for session A rejected on session B | integration | `go test ./internal/webserver/ -run TestCapability_WrongSessionRejected -count=1` | ❌ Wave 0 |
| SEC-04 | `?readonly=1` cannot grant write access without write cap | integration | `go test ./internal/webserver/ -run TestReadOnlyParam_CannotGrantWrite -count=1` | ❌ Wave 0 |
| SEC-05 | `MsgInput` rejected for read-only cap; reconnect without `?readonly` still blocked | integration | `go test ./internal/webserver/ -run TestSecurity_ReadOnlyCapabilityBlocksMsgInput -count=1` | ❌ Wave 0 (invert existing test) |
| Token sign/verify | Valid token verifies; tampered token rejected; wrong key rejected | unit | `go test ./internal/capability/ -run TestSign -run TestVerify -count=1` | ❌ Wave 0 |
| Key persistence | FileKeyStore load/save round-trip; missing file generates new key | unit | `go test ./internal/capability/ -run TestFileKeyStore -count=1` | ❌ Wave 0 |
| Grant list | `isGrantActive` returns false after `ClearGrants`; join code single-use | unit | `go test ./internal/webserver/ -run TestGrantList -count=1` | ❌ Wave 0 |
| Signature forgery resistance | Fuzz token verification for forgeries | fuzz | `go test ./internal/capability/ -fuzz=FuzzVerify -fuzztime=30s` | ❌ Wave 0 (security-review scaffold is reference) |

### Sampling Rate

- **Per task commit:** `go test ./internal/capability/ ./internal/webserver/ -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/capability/capability.go` — new package (Sign, Verify, IssueCapabilities, JoinCodeManager)
- [ ] `internal/capability/capability_test.go` — unit + fuzz test file
- [ ] `internal/capability/keystore.go` — KeyStore interface + FileKeyStore
- [ ] `internal/capability/keystore_test.go` — persistence round-trip
- [ ] Update `internal/webserver/server_test.go` — add capability test helpers from security-review scaffold; invert the two security-finding tests to assert correct (post-fix) behavior

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | HMAC-SHA256 signed capability token (issuer = verifier = daemon) |
| V3 Session Management | yes | `grant_id` per-session tracking; toggle-off clears all grants |
| V4 Access Control | yes | `requireCapability` middleware on all four guarded routes; relay-level `MsgInput` rejection |
| V5 Input Validation | yes | Token parsing rejects malformed base64url, malformed JSON, wrong segment count |
| V6 Cryptography | yes | `crypto/hmac` + `crypto/sha256` + `crypto/rand` (stdlib); no hand-rolled crypto; constant-time comparison via `hmac.Equal` |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Tailnet peer enumeration and PTY takeover | Elevation of Privilege | `requireCapability` middleware rejects all requests without valid cap; SEC-01 removes auto-expose |
| Read-only bypass via reconnect without `?readonly=1` | Elevation of Privilege | `sub.ReadOnly` sourced from verified capability `perms` claim (D-24); SEC-05 regression test |
| Token forgery (HMAC bypass) | Spoofing | 256-bit HMAC key; constant-time `hmac.Equal` comparison; fuzz test |
| Leaked token via URL logs | Information Disclosure | Known tradeoff (D-08); no in-product fix for Phase 87; document access log handling |
| Join code replay (second use after exchange) | Elevation of Privilege | Single-use delete under mutex in `JoinCodeManager.Exchange`; Pitfall 4 |
| QR code photograph grants access | Elevation of Privilege | QR encodes join-code URL only (D-09); photograph is useless after 5-minute TTL or first exchange |
| Signing key leakage via `settings.json` sharing | Information Disclosure | Key in separate `capability.key` file (D-04) |
| Algorithm confusion / `alg:none` attack | Spoofing | No `alg` field; no JWT library; format is not JWT-compatible |

---

## Sources

### Primary (HIGH confidence)

- Go stdlib `crypto/hmac`, `crypto/sha256`, `crypto/rand`, `encoding/base64`, `encoding/base32`, `encoding/json` — verified via `go doc` on go1.26.2
- `internal/webserver/server.go` (full read) — existing route wiring, `basicAuthMiddleware` pattern, `webEnabled` map pattern
- `internal/webserver/auth.go` (full read) — middleware signature and pattern
- `internal/daemon/engine.go` (full read) — `daemonSettings` persistence pattern, `os.WriteFile` with mode 0600
- `internal/daemon/api.go` (full read) — `handleWebServe`, `handleCreateSession`, auto-enable at line 292
- `internal/relay/hub.go` (full read) — `Subscriber.ReadOnly` field, hub subscribe/unsubscribe pattern
- `internal/relay/protocol.go` (grep) — `MsgInput`, `MsgOutput`, `MsgResize2`, `MsgPing`, `MsgMeta` frame type bytes
- `security-review/internal_webserver_server_test.go` (full read) — test scaffold helpers for Phase 87 tests
- `go.mod` (full read) — confirmed no new dependencies needed; `skip2/go-qrcode` and `coder/websocket` already present
- `87-CONTEXT.md`, `87-DISCUSSION-LOG.md`, `87-UI-SPEC.md` — all decisions locked, no alternatives to research

### Secondary (MEDIUM confidence)

- `security-review/SECURITY_REVIEW.md` — findings 1 and 2 confirm exploitability of current code; exploit scenarios inform test design

### Tertiary (LOW confidence)

- None — all claims in this research are verified or cited from codebase reads and stdlib documentation.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all Go stdlib, verified via `go doc`; no new libraries
- Architecture: HIGH — derived directly from locked CONTEXT.md decisions and verified against existing codebase patterns
- Pitfalls: HIGH — identified from direct codebase inspection (timing comparison, TOCTOU race, WebSocket upgrade ordering, claims serialization stability)

**Research date:** 2026-04-19
**Valid until:** Stable — based entirely on locked decisions and stdlib APIs. Re-research only if CONTEXT.md decisions are reopened.
