# Phase 88: WebSocket Handshake Security - Research

**Researched:** 2026-04-21
**Domain:** WebSocket handshake Origin-allowlist enforcement in Go (`coder/websocket` v1.8.14 + Go 1.22+ `ServeMux`); defense-in-depth middleware layering; source-grep regression guards
**Confidence:** HIGH — library source inspected locally (`/Users/ken/go/pkg/mod/github.com/coder/websocket@v1.8.14/accept.go`); every locked decision in CONTEXT cross-referenced against production code; no unresolved open questions

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Allowlist Composition**
- D-01: Origin allowlist is sourced **dynamically from `ws.BaseURL()`** at upgrade time. The middleware reads `ws.listener.Addr()` under `ws.mu.RLock()` and composes `scheme://host:port` every request. Handles random-port fallback automatically.
- D-02: "Configured same-origin" in SEC-06 is interpreted as **reflective only**. Allowlist is strictly the server's own current serving origin — no new `settings.json` field, no UI for extra origins in v3.1.
- D-03: Matching is **strict byte-for-byte exact match**. No case-folding, no port-stripping, no normalization. Mismatch → 403; user re-opens the link from the share panel.
- D-04: Compare the **origin tuple only** (`scheme://host:port`). `Origin` never carries a path; `BaseURL()` returns the tuple without a path.

**Missing-Origin Policy (resolves Phase 87 pending todo)**
- D-05: Upgrade request with **no `Origin` header is always rejected with 403**. Browsers always set `Origin` on WS upgrade. Non-browser clients have the localhost relay for programmatic access.
- D-06: Origin check applies **only to the WebSocket upgrade route** (`GET /sessions/{id}/ws`), not HTML/API routes. Capability tokens are CSRF-resistant bearers already; plain navigation does not set `Origin` reliably.
- D-07: Rejection body is a **generic `"forbidden"` / 403** for missing-Origin and wrong-Origin — no distinction (T-87-08 info-disclosure defense).

**Relay Scope Inclusion**
- D-08: Phase 88 **also removes `InsecureSkipVerify: true`** from `internal/relay/server.go:59-62`. Reading SC-4 literally: "accept-all is gone from the code path." Supersedes Phase 87 D-21.
- D-09: Relay's replacement policy is **loopback-only**. Allowlist is `{http,https}://{localhost,127.0.0.1}:<port>` derived from the relay's listener address.

**Enforcement**
- D-10: New **`requireAllowedOrigin(next)` middleware** wraps only `/sessions/{id}/ws`, composed **outside** `requireCapability`. Composition order (outermost → innermost): Basic Auth (local only) → Origin → Capability → `handleWSSRelay`.
- D-11: Allowlist is **re-evaluated per request** (reads `ws.BaseURL()` under `ws.mu.RLock()`). Negligible cost; avoids snapshot-invalidation complexity.
- D-12: As **defense in depth**, `websocket.AcceptOptions.OriginPatterns` is ALSO set to the strict allowlist. The `OriginPatterns: []string{"*"}` at `server.go:632` is replaced.

**Anti-Regression Guard (SC-4)**
- D-13: **Source-grep regression test** asserts three things across two files:
  1. `internal/webserver/server.go` contains NO `"*"` literal inside an `OriginPatterns` slice.
  2. `internal/relay/server.go` contains NO `InsecureSkipVerify: true` literal.
  3. The expected `OriginPatterns` line in `handleWSSRelay` references `ws.BaseURL()`.

**Observability**
- D-14: **No logs, no metrics** on Origin rejection. Matches Phase 87's minimal-observability stance.

### Claude's Discretion
- Exact package location / filename for the new middleware (likely `internal/webserver/origin_mw.go`).
- Helper shape for composing the allowlist string(s) — method on `*WebServer` vs free function.
- Relay-side helper for deriving loopback origins from the listener address — placement and naming.
- Test organization: unit tests for origin matching, integration tests for the full upgrade flow, source-grep regression tests.
- Exact error strings in the regression test failure messages.
- Whether to inline the allowlist build at the `AcceptOptions` call site or pull it into a shared helper used by both the middleware and `AcceptOptions`.

### Deferred Ideas (OUT OF SCOPE)
- User-configurable additional origins (`extra_allowed_origins` in `settings.json` + Security-section UI) — future phase.
- CLI/native direct-to-WSS clients over tailnet — future phase with dedicated path.
- Observability for Origin rejections (debug logs / metrics) — future phase.
- Snapshot-based allowlist (only if profiling ever shows per-request `BaseURL()` is a bottleneck).
- Relay exposure hardening beyond Origin (capability checks etc.) — needed only if relay is ever bound to non-loopback.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SEC-06 | WebSocket upgrade rejects requests whose `Origin` is not in the server allowlist | D-01 dynamic allowlist from `BaseURL()`; D-03 strict exact match; D-05 missing-Origin → 403; D-10 middleware ordering; D-12 library-level belt-and-suspenders; D-13 source-grep regression guard covering SC-4 |

</phase_requirements>

---

## Summary

Phase 88 adds a single HTTP middleware — `requireAllowedOrigin` — that gates `GET /sessions/{id}/ws` on the browser's `Origin` header exactly matching `ws.BaseURL()`. It wraps around the existing `requireCapability` from Phase 87, so a request failing Origin is rejected with **403 `forbidden`** before any signature-verify work happens. Defense-in-depth: the `coder/websocket` library's own origin-authentication path is also switched from `OriginPatterns: []string{"*"}` to a strict allowlist built from `BaseURL()`. Separately, the loopback-only `internal/relay/server.go` drops `InsecureSkipVerify: true` in favor of a loopback `OriginPatterns` list derived from its listener address — a pure defense-in-depth cleanup (the relay is not exposed to tailnet in v3.1). A low-tech source-grep regression test locks the anti-regression contract in SC-4: any future maintainer that re-introduces `"*"` or `InsecureSkipVerify: true` fails the test.

**Primary recommendation:** Follow the exact Phase 87 `capability_mw.go` shape for `origin_mw.go`. Add the middleware at `setupRoutes()` by wrapping the WS route only: `mux.HandleFunc("GET /sessions/{id}/ws", ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay)))`. Build the allowlist once per request in the middleware AND pass it to `websocket.AcceptOptions.OriginPatterns` in `handleWSSRelay` — extract a private helper `ws.allowedOrigins()` that returns `[]string{ws.BaseURL()}` to share between the two call sites. Mirror Phase 87's `TestVerify_ConstantTimeComparison` pattern for the SC-4 source-grep guard (reads source files, asserts on string content, actionable failure messages).

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Origin header validation (handshake) | API / Backend (Go HTTP middleware) | — | `Origin` is an HTTP request header; only the server can enforce it because CORS response headers don't apply to WebSocket upgrades |
| Allowlist composition | API / Backend (`ws.BaseURL()` reflective) | — | Reflective policy: allowlist = server's own current serving URL; no config surface |
| Belt-and-suspenders origin check | API / Backend (library-internal `coder/websocket`) | — | Library-level check fires inside `websocket.Accept` if middleware is bypassed by a future routing bug |
| Regression test (SC-4) | API / Backend (Go `testing` source-grep) | — | Source-grep lives in-package with the files it guards; Go `os.ReadFile` is the established pattern (Phase 87 constant-time test) |
| Browser-side behavior (Origin emission) | Browser / Client | — | Browsers emit `Origin` automatically on `new WebSocket()` — no JS change needed; JS cannot forge/omit Origin from a browser context |

Note: no frontend changes. The terminal page at `/sessions/{id}` already loads from the same origin that hosts the WSS endpoint, so the browser emits `Origin: <ws.BaseURL()>` unconditionally on WebSocket upgrade. No tier below the Go backend is touched.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/coder/websocket` | v1.8.14 (pinned in go.mod) | WebSocket upgrade at `handleWSSRelay`; also owns the library-level `OriginPatterns` belt-and-suspenders | Already adopted project-wide; `AcceptOptions.OriginPatterns` exists natively; no replacement needed |
| `net/http` (stdlib) | Go 1.22+ ServeMux | Route registration and middleware chaining | Already used; Phase 87 uses the same `func(http.HandlerFunc) http.HandlerFunc` middleware shape |
| `testing` (stdlib) | Go 1.26.1 (per go.mod) | Unit + integration test framework | Only Go testing framework the codebase uses |

**No new dependencies.** Phase 88 is purely additive Go code using already-imported libraries.

**Version verification:** Confirmed `github.com/coder/websocket v1.8.14` is the current pinned version in `go.mod:go.sum`. Mod cache contains the exact source read during this research (`/Users/ken/go/pkg/mod/github.com/coder/websocket@v1.8.14/accept.go`). [VERIFIED: local mod cache inspection]

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/scottkw/agenthub/internal/capability` | in-tree | `ClaimsFromContext` (unused by Phase 88 middleware; Origin middleware sits outside capability) | Not needed for Origin middleware itself |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom middleware | `websocket.AcceptOptions.OriginPatterns` exclusively (no middleware) | REJECTED by CONTEXT D-10. Library rejection returns library-controlled body text; custom middleware returns clean `http.Error(w, "forbidden", 403)`. Middleware also runs before capability work. |
| Strict exact match | Case-insensitive / port-flexible matching | REJECTED by CONTEXT D-03. Strict match keeps attack surface minimal and the test matrix finite. |
| Static allowlist from config | Dynamic `BaseURL()` per request | REJECTED by CONTEXT D-01/D-02. Reflective allowlist handles random-port fallback and Tailscale FQDN changes without a config field to keep in sync. |

**Installation:** (none — no new packages)

---

## Architecture Patterns

### System Architecture Diagram

```
Browser                         WebServer (tailnet-facing)
───────                         ─────────────────────────
                                                                       
 (user loads /sessions/{id}?cap=TOKEN)                                  
  │                                                                    
  │ TLS handshake (Tailscale cert or self-signed)                       
  │ ──────────────────────────────────────────►                         
  │                                                                    
  │ GET /sessions/{id}/ws?cap=TOKEN                                     
  │   Origin: https://host.ts.net:7443 ◄── sent by browser              
  │   Sec-WebSocket-Key: …                                              
  │   Upgrade: websocket                                                
  │ ──────────────────────────────────────────►                         
  │                                             ┌─────────────────┐    
  │                                             │ basicAuth (local │    
  │                                             │   mode only)     │    
  │                                             └────────┬─────────┘    
  │                                                      ▼              
  │                                             ┌─────────────────┐    
  │                                             │ requireAllowed  │  ◄── NEW Phase 88
  │                                             │   Origin        │    
  │                                             │                  │    
  │                                             │ origin := hdr    │    
  │                                             │ allowed := ws.   │    
  │                                             │   BaseURL()      │    
  │                                             │                  │    
  │                                             │ if origin == ""  │    
  │                                             │   → 403 forbidden│    
  │                                             │ if origin !=     │    
  │                                             │   allowed        │    
  │                                             │   → 403 forbidden│    
  │                                             └────────┬─────────┘    
  │                                                      ▼              
  │                                             ┌─────────────────┐    
  │                                             │ require         │    
  │                                             │   Capability    │  ◄── Phase 87 (unchanged)
  │                                             │ (HMAC verify,   │    
  │                                             │  grant check,   │    
  │                                             │  SID match)      │    
  │                                             └────────┬─────────┘    
  │                                                      ▼              
  │                                             ┌─────────────────┐    
  │                                             │ handleWSSRelay  │    
  │                                             │                  │    
  │                                             │ websocket.Accept │    
  │                                             │   AcceptOptions{ │  ◄── NEW: OriginPatterns
  │                                             │     OriginPatterns:│     strict (was "*")
  │                                             │     ws.allowedOrig │    
  │                                             │   }              │    
  │                                             └──────────────────┘    
  │                                                                    
  │ 101 Switching Protocols                                             
  │ ◄──────────────────────────────────────────                         
  │                                                                    
  │ (WebSocket frames)                                                  
  │ ◄──────────────────────────────────────────►                         
```

Relay diagram (loopback-only, localhost):

```
Client (loopback)            Relay Server
──────────────               ────────────
                                                                      
  GET /sessions/{id}/ws                                               
    Origin: http://127.0.0.1:<port>                                   
  ──────────────────────────►                                         
                            ┌─────────────────────────────┐           
                            │ handleSession                │           
                            │                              │           
                            │ websocket.Accept (opts):     │           
                            │   was: InsecureSkipVerify=1  │  ◄── REMOVED
                            │   now: OriginPatterns =      │  ◄── NEW
                            │     {http,https}://          │           
                            │     {localhost,127.0.0.1}:   │           
                            │     <listener.port>          │           
                            └──────────────────────────────┘           
```

### Recommended Project Structure

```
internal/
├── webserver/
│   ├── server.go            # setupRoutes wires new middleware (Phase 88 edit)
│   ├── capability_mw.go     # unchanged (Phase 87 sibling)
│   ├── origin_mw.go         # NEW: requireAllowedOrigin + ws.allowedOrigins helper
│   ├── origin_mw_test.go    # NEW: unit tests for origin matching logic
│   ├── capability_test.go   # Phase 87; minor integration-test additions possible
│   └── security_regression_test.go  # NEW: source-grep SC-4 guard (may co-locate with capability_test.go)
├── relay/
│   ├── server.go            # Phase 88 edit: OriginPatterns replaces InsecureSkipVerify
│   └── server_test.go       # Phase 88 additions: cross-origin rejected, loopback accepted
```

Placement of `ws.allowedOrigins()` helper is Claude's discretion per CONTEXT — recommended: method on `*WebServer` in `origin_mw.go`, returning `[]string{ws.BaseURL()}`, taking the same `ws.mu.RLock()` path `BaseURL()` already takes. Shared between the middleware and the `websocket.AcceptOptions` site in `handleWSSRelay`.

### Pattern 1: Middleware Composition (mirrors Phase 87 `capability_mw.go`)

**What:** Go 1.22 `http.ServeMux` accepts `func(w, r)` handlers. Phase 87 established the idiom `ws.requireX(next http.HandlerFunc) http.HandlerFunc`. Phase 88 adds `requireAllowedOrigin` with identical shape, composed *outside* `requireCapability`.

**When to use:** Any HTTP route needing to fail fast on a header check before downstream work.

**Example:**
```go
// Source: internal/webserver/capability_mw.go:37-75 (Phase 87 template)
// Phase 88 adaptation:
func (ws *WebServer) requireAllowedOrigin(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        if origin == "" {
            // D-05: missing Origin → 403, same body as mismatched
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        allowed := ws.BaseURL() // D-01 dynamic allowlist
        if allowed == "" || origin != allowed {
            // D-03 strict byte-for-byte exact match
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }
        next(w, r)
    }
}
```

### Pattern 2: Route Wiring with Two Wrappers

**What:** Phase 88 wraps the WS route with Origin outside, Capability inside. Basic Auth (local-mode only) wraps the entire mux at `startLocal()` — already outside both.

**Example:**
```go
// Source: internal/webserver/server.go:396 (current Phase 87 wiring)
// Phase 88 edit: add ws.requireAllowedOrigin outside ws.requireCapability
mux.HandleFunc("GET /sessions/{id}/ws",
    ws.requireAllowedOrigin(ws.requireCapability(ws.handleWSSRelay)))
```

No precedence conflict with the parent `GET /sessions/{id}` HTML route — Go 1.22 ServeMux treats `/sessions/{id}/ws` as a longer-path pattern with strict priority; the HTML route is at `mux.HandleFunc("GET /sessions/{id}", ...)` (server.go:391). [VERIFIED: `net/http` ServeMux pattern specificity rules, Go 1.22 release notes; existing Phase 87 test `TestCapability_ValidCapReturnsSession` succeeds against precisely this layout]

### Pattern 3: Belt-and-Suspenders with `AcceptOptions.OriginPatterns` (D-12)

**What:** After middleware rejection, `websocket.Accept` performs a second origin check if `OriginPatterns` is set. The library normalizes Origin against the pattern list via `path.Match` (case-insensitive).

**Key semantic from library source (`/Users/ken/go/pkg/mod/github.com/coder/websocket@v1.8.14/accept.go:228-264`):**

```go
func authenticateOrigin(r *http.Request, originHosts []string) error {
    origin := r.Header.Get("Origin")
    if origin == "" {
        return nil  // ← library ACCEPTS missing Origin when OriginPatterns is set
    }
    u, err := url.Parse(origin)
    if err != nil { return fmt.Errorf(...) }
    if strings.EqualFold(r.Host, u.Host) { return nil }  // ← host-match auto-pass
    for _, hostPattern := range originHosts {
        target := u.Host
        if strings.Contains(hostPattern, "://") {
            target = u.Scheme + "://" + u.Host  // ← if pattern has scheme, match scheme://host
        }
        matched, err := match(hostPattern, target)  // path.Match case-insensitive
        if matched { return nil }
    }
    return fmt.Errorf("request Origin %q is not authorized for Host %q", u.Host, r.Host)
}
```

**Critical implications:**

1. **Pattern semantics are host-based by default, scheme+host if pattern contains `"://"`.** `ws.BaseURL()` returns e.g. `https://host.ts.net:7443` — the pattern `"https://host.ts.net:7443"` will cause the library to match against `u.Scheme + "://" + u.Host` → `https://host.ts.net:7443`. Strict match behaves as we expect. [VERIFIED: library source `accept.go:244-248`]

2. **Library is case-INSENSITIVE** (`match` calls `strings.ToLower`). Middleware D-03 is byte-for-byte strict; library is case-folding. The strict middleware is ALWAYS tighter — the library only ever accepts what the middleware also accepts (modulo case). Safe composition.

3. **Library auto-passes when `Origin == Host`.** If a browser sends `Origin: https://foo` AND `Host: foo`, library accepts regardless of `OriginPatterns`. For our WS route the browser's `Host` is the same tailnet FQDN we're allowlisting, so this is identical behavior. No surprise.

4. **Library passes missing `Origin` silently.** ← ASYMMETRY. Our middleware rejects missing Origin (D-05); the library accepts it. This means the middleware-first ordering is load-bearing: if anyone ever removes the middleware, the library will silently allow non-browser direct-to-WSS clients. The D-13 source-grep guard explicitly checks for this. [VERIFIED: library `accept.go:230-232` — `if origin == "" { return nil }`]

5. **`OriginPatterns: []string{"*"}` is NOT a bypass** the way it looks. `"*"` is a valid `path.Match` pattern that matches any single path segment but not empty strings, and when the pattern contains no `"://"` it matches against `u.Host` only. However in practice it matches every non-empty host, so effectively accept-all. D-12 still treats this as a landmine and D-13's grep disallows the literal.

### Anti-Patterns to Avoid

- **Running the check after `websocket.Accept`:** the library has already written 101 Switching Protocols. The handshake is committed. Must check BEFORE Accept. The middleware position satisfies this. [VERIFIED: `accept.go:151` `w.WriteHeader(http.StatusSwitchingProtocols)` runs after origin auth passes]
- **Stripping ports or normalizing case:** CONTEXT D-03 locks strict byte-for-byte match. Diverging from `ws.BaseURL()`'s canonical form introduces test matrix complexity without security benefit.
- **Allowlisting `"*"` as a placeholder:** fails D-13 source-grep guard. Never write `OriginPatterns: []string{"*"}` even in comments.
- **Reading `ws.listener.Addr()` without `ws.mu.RLock()`:** race with `Start()`/`Stop()`. `BaseURL()` already takes the lock — reuse it.
- **Building the allowlist from `ws.config.BindIP` directly:** in local mode the listener's resolved port differs from `ws.config.Port` (random-port fallback); `BaseURL()` returns the post-fallback port, `config.Port` does not.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL parsing for Origin | Custom `scheme://host:port` extraction | Exact string compare against `ws.BaseURL()` | D-03/D-04 lock strict byte-for-byte match. Parsing adds attack surface (IPv6, IDN, port-coalescing edge cases) for zero benefit. |
| WebSocket handshake validation | Custom `Sec-WebSocket-Key` check | `websocket.Accept()` (already used) | Library already does RFC 6455 handshake validation + Sec-WebSocket-Accept HMAC. |
| Origin allowlist matching for library layer | Custom loop, regex, or glob helper | `websocket.AcceptOptions.OriginPatterns` | Library supports this natively via `path.Match`; just hand it `[]string{ws.BaseURL()}`. |
| Regression-guard DSL | Abstract "source-must-contain" helper | Inline `os.ReadFile` + `strings.Contains` (Phase 87 pattern) | Three string assertions across two files do not justify abstraction. See Phase 87 `TestVerify_ConstantTimeComparison` (`internal/capability/capability_test.go:161-173`). |
| Per-request allowlist caching | Snapshot with invalidation hook | Read `ws.BaseURL()` under RLock every time | D-11 explicit: negligible cost, avoid complexity. WS handshakes are rare (one per browser page load). |
| Localhost loopback origin derivation | Resolve `localhost` → `::1`/`127.0.0.1` at runtime | Static list `{http,https}://{localhost,127.0.0.1}:<port>` | D-09 explicit: four literal strings. Avoids DNS resolver edge cases and IPv6 hostname-literal quoting. |

**Key insight:** Phase 88 is intentionally dumb. Strict string compare, no normalization, no parsing. Every layer added to "smarten" Origin matching is a bypass waiting to happen (proxy injects `Origin: https://EVIL.com` with different case; attacker adds trailing `.`; IPv6 with brackets vs without). Byte-for-byte equality is the only rule the middleware enforces.

---

## Runtime State Inventory

(This phase is additive: new middleware, new test, minor edits to two handler bodies. No persisted state changes, no database migrations, no file format changes.)

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — Phase 88 introduces no persisted state. | None |
| Live service config | None — reflective allowlist means no user-visible setting to migrate. | None |
| OS-registered state | None. | None |
| Secrets/env vars | None. Basic Auth password (local mode only) is unaffected — it sits OUTSIDE the new Origin middleware. | None |
| Build artifacts | None. Go binary rebuild only. | None |

**Canonical question answer:** After the code lands, no runtime system has stale state referencing pre-Phase-88 behavior. The only observable change: `/sessions/{id}/ws` now 403s for non-matching `Origin` headers. Browsers that successfully loaded the terminal page at `/sessions/{id}` will always send the correct `Origin` on the WS upgrade (same-origin from the HTML document's URL).

---

## Common Pitfalls

### Pitfall 1: BaseURL() returns empty string before listener is up
**What goes wrong:** If `ws.requireAllowedOrigin` is invoked before `Start()` completes, `ws.listener` is nil and `BaseURL()` returns `""`. The middleware would then compare `Origin != ""` and 403 everything — acceptable fail-closed behavior, but means a race could break the very first upgrade.
**Why it happens:** `setupRoutes()` runs in `NewWebServer()` (server.go:95), BEFORE `Start()`. The mux is registered immediately; handlers only run on incoming requests. Handshakes can't arrive before the listener is up because there's nothing listening. So the race is theoretically impossible, but code readers may worry.
**How to avoid:** Document in the middleware: "relies on ws.BaseURL() returning canonical origin; fail-closed if listener not ready." Add a test that asserts an empty allowed origin fails closed (not opens).
**Warning signs:** `Origin` check returning 403 with allowed origin apparently matching — log/debug would reveal `allowed == ""`.

### Pitfall 2: `Origin: null` from sandboxed iframes, file:// pages, or data: URLs
**What goes wrong:** Some browsers emit `Origin: null` (4 literal characters) instead of omitting the header, for opaque-origin contexts. Our strict match `origin != ws.BaseURL()` correctly rejects this with 403.
**Why it happens:** HTML spec: sandboxed iframes, `file://` pages, and `data:`/`blob:` URLs produce opaque origins; the browser serializes these as the string `"null"`.
**How to avoid:** Do nothing — the strict byte-for-byte match inherently rejects `"null"`. Document in a comment that `"null"` is correctly rejected (don't special-case it). [VERIFIED: HTML Living Standard §Origin serialization; cross-referenced with coder/websocket library source which does NOT special-case "null"]
**Warning signs:** Users in exotic embedding scenarios (AgentHub terminal embedded in a sandboxed iframe in some other product) report 403 — expected, per CONTEXT D-05.

### Pitfall 3: Local-network-fallback self-signed cert does NOT affect Origin
**What goes wrong:** In local mode the server issues a self-signed cert for `BindIP` (e.g. `192.168.1.50`), and the user gets a TLS trust warning. Concerns: does the warning change what Origin the browser sends?
**Why it happens:** Origin is derived from the URL the user loaded (the document's URL), not from the cert chain. A self-signed cert with a TLS warning that the user dismisses still produces a document URL of `https://192.168.1.50:7443`, which is exactly what `BaseURL()` returns. No divergence.
**How to avoid:** Nothing needed. `BaseURL()` at server.go:332-334 emits `https://<BindIP>:<port>` in local mode, which the browser will match in `Origin`. [VERIFIED: RFC 6454 §4 — origin is derived from URL, not cert]
**Warning signs:** Regression test `TestWebServer_LocalMode_AllowedOriginIsBindIP` (Validation Architecture below) catches any drift.

### Pitfall 4: Case mismatch between middleware and library (D-12)
**What goes wrong:** The middleware does byte-for-byte strict match (`origin != allowed`), but `websocket.AcceptOptions.OriginPatterns` matching is case-INSENSITIVE via `strings.ToLower`. In an edge case where a proxy or browser emits `Origin: HTTPS://host.ts.net:7443` (upper-case scheme), the middleware rejects (403) but the library would have accepted. This asymmetry is safe — the tighter check wins — but the test matrix should confirm no regression.
**Why it happens:** Different layers implement different normalization. Documented pragmatically: the middleware is always tighter.
**How to avoid:** Write a unit test that asserts uppercase scheme in `Origin` is rejected by the middleware. Write an integration test that bypasses the middleware (direct Accept call) with uppercase scheme — library accepts it, so Middleware must be the enforcement primitive. [VERIFIED: library source, confirmed by inspection of `accept.go:262-263`]
**Warning signs:** None in practice — browsers emit lowercase canonical origins. But the asymmetry must be documented for future maintainers.

### Pitfall 5: Forgetting the library belt-and-suspenders (SC-4)
**What goes wrong:** Removing `OriginPatterns: []string{"*"}` from `server.go:632` and not replacing it with `ws.allowedOrigins()` would cause the library to fall back to its "Host header same-origin" default. In tailnet mode, `r.Host` is the tailnet FQDN + port and `Origin` from the browser is also that FQDN + port — they match, so library still passes. BUT: if anyone ever runs the server behind a proxy that rewrites `Host` but not `Origin`, the library's Host-based fallback silently accepts. The `OriginPatterns` explicit allowlist defends this case.
**Why it happens:** Library's defaulting is too permissive for a security-review-driven gate.
**How to avoid:** Always set `OriginPatterns` explicitly. Validate via the D-13 grep guard (point 3: "references `ws.BaseURL()`").
**Warning signs:** `server.go` has a `websocket.AcceptOptions{}` block with only `InsecureSkipVerify: false` and no `OriginPatterns` — regression. Grep guard catches this via the "line references `ws.BaseURL()`" check.

### Pitfall 6: Relay cleanup (D-08) accidentally exposes a loopback service
**What goes wrong:** Changing `InsecureSkipVerify: true` to `OriginPatterns: <loopback-list>` in `relay/server.go` does NOT harden the relay — the relay is still localhost-only via its listener bind in `daemon/api.go:137` (`net.Listen("tcp", "127.0.0.1:0")`). The Origin cleanup is a landmine-avoidance play, not a security gain.
**Why it happens:** Reading the diff, a reviewer might assume the relay has become "more accessible" via Origin-based policy. It hasn't.
**How to avoid:** Call this out in the commit message and the Plan. The relay is loopback-only via listener bind; Origin policy is belt-and-suspenders for a future mistake.
**Warning signs:** A reviewer asks "why is the relay doing Origin checks if it's loopback-only?" — answer: "future-proofing against accidental rebind."

### Pitfall 7: Relay port derivation (D-09)
**What goes wrong:** The relay Server struct in `relay/server.go` does NOT know its own listening port — the listener is owned by `daemon/api.go` (see `daemon/api.go:137-142`). To compose `http://127.0.0.1:<port>` at `websocket.Accept` time, we need the port from somewhere. Options:
  a. Derive from `r.Host` in the handler (cheapest — the request's Host header has it).
  b. Pass the port into `relay.NewServer` at construction time.
  c. Inspect `r.Context()` or `r.URL` (not reliable; URL may be path-only).
**Why it happens:** Relay is a pure http.Handler; it doesn't own a listener.
**How to avoid:** Recommended: option (a) — read `r.Host` inside `handleSession` and compose `{http,https}://localhost:<r.Host.port>` and `{http,https}://127.0.0.1:<r.Host.port>`. This is reflective (same D-01 pattern as the webserver). Alternatively (b) if simpler: pass port to `NewServer(manager, backend, port int)` and build the four-string list once at handler level. Both are acceptable per Claude's Discretion.
**Warning signs:** Relay integration test fails with "origin rejected" when dialing from loopback — means the port in the allowlist doesn't match the listener's actual port.

---

## Code Examples

Verified patterns from the existing codebase and library source:

### Phase 87 middleware shape (template to mirror for `requireAllowedOrigin`)
```go
// Source: internal/webserver/capability_mw.go:37-75
func (ws *WebServer) requireCapability(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("cap")
        if token == "" {
            http.Error(w, "capability required", http.StatusUnauthorized)
            return
        }
        // …verify + grant check + path-id check…
        ctx := capability.WithClaims(r.Context(), claims)
        next(w, r.WithContext(ctx))
    }
}
```

### Phase 87 constant-time source-grep regression (template for D-13)
```go
// Source: internal/capability/capability_test.go:161-173
func TestVerify_ConstantTimeComparison(t *testing.T) {
    data, err := os.ReadFile("capability.go")
    if err != nil {
        t.Fatalf("ReadFile capability.go: %v", err)
    }
    src := string(data)
    if !strings.Contains(src, "hmac.Equal") {
        t.Error("capability.go must call hmac.Equal for signature comparison")
    }
    if strings.Contains(src, "bytes.Equal") {
        t.Error("capability.go must not use bytes.Equal on signature bytes (timing side channel)")
    }
}
```

**Direct adaptation for Phase 88's D-13 three-condition guard:**

```go
// Phase 88 regression test (recommended placement: internal/webserver/origin_mw_test.go
// or a new internal/webserver/security_regression_test.go for clarity). Also add a
// sibling test in internal/relay/server_test.go for the InsecureSkipVerify literal.
func TestOriginAllowlist_NoAcceptAllRegression(t *testing.T) {
    src, err := os.ReadFile("server.go")
    if err != nil {
        t.Fatalf("ReadFile server.go: %v", err)
    }
    s := string(src)
    // 1. No "*" inside OriginPatterns
    if strings.Contains(s, `OriginPatterns: []string{"*"}`) {
        t.Error(`server.go must not contain OriginPatterns: []string{"*"} — Phase 88 SC-4 anti-regression`)
    }
    // 3. handleWSSRelay references ws.BaseURL() in its AcceptOptions site
    if !strings.Contains(s, "ws.BaseURL()") && !strings.Contains(s, "ws.allowedOrigins()") {
        t.Error(`server.go handleWSSRelay must reference ws.BaseURL() (or a helper that does) in AcceptOptions — Phase 88 SC-4`)
    }
}

// sibling test in internal/relay/server_test.go:
func TestRelayOrigin_NoInsecureSkipVerifyRegression(t *testing.T) {
    src, err := os.ReadFile("server.go")
    if err != nil {
        t.Fatalf("ReadFile server.go: %v", err)
    }
    if strings.Contains(string(src), "InsecureSkipVerify: true") {
        t.Error(`relay/server.go must not contain InsecureSkipVerify: true — Phase 88 SC-4 anti-regression`)
    }
}
```

### Phase 87 test helpers (template for WS integration tests)
```go
// Source: internal/webserver/capability_test_helpers.go:131-170
// testServerWithHub returns a running WebServer + live hub — ideal harness for
// Phase 88 integration tests that exercise the full upgrade flow. Use
// dialWebServerWS (line 205-221) with custom headers to inject Origin.
ws, client, _, _ := testServerWithHub(t, "sess-88")
ws.SetSigningKey(capTestKey)
token := issueCapFor(t, ws, "sess-88", "read,write")

// Good-origin case:
headers := http.Header{}
headers.Set("Origin", ws.BaseURL())
conn := dialWebServerWS(t, client, ws.BaseURL(), "/sessions/sess-88/ws?cap="+token, headers)
// conn should open successfully.

// Bad-origin case: use http.Client.Get against the WS URL to capture status
// code cleanly (websocket.Dial converts 403 into an opaque dial error).
wsURL := ws.BaseURL() + "/sessions/sess-88/ws?cap=" + token
req, _ := http.NewRequest("GET", wsURL, nil)
req.Header.Set("Origin", "https://evil.example")
req.Header.Set("Connection", "Upgrade")
req.Header.Set("Upgrade", "websocket")
req.Header.Set("Sec-WebSocket-Version", "13")
req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
resp, _ := client.Do(req)
// resp.StatusCode == 403
```

### coder/websocket library's authenticateOrigin (reference for D-12 belt-and-suspenders)
```go
// Source: /Users/ken/go/pkg/mod/github.com/coder/websocket@v1.8.14/accept.go:228-260
// Called from accept.go:116-125 when !opts.InsecureSkipVerify.
// Returns nil on: missing Origin, Origin-Host same-host match, or pattern match.
// Returns 403 on: unauthorized Origin.
// Patterns with "://" match scheme+host; without, match host only.
// Case-insensitive via strings.ToLower.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `OriginPatterns: []string{"*"}` accept-all | Strict allowlist from `ws.BaseURL()` | Phase 88 | Cross-site WS hijacking blocked; SEC-06 / SC-1 |
| `InsecureSkipVerify: true` on relay | Loopback `OriginPatterns` list | Phase 88 | Landmine removed — relay can't be accidentally exposed-then-hijacked |
| `r.URL.Query().Get("readonly")` as write gate | `claims.Perms == "read"` | Phase 87 | Client-asserted perms removed; server-signed only |
| Tailnet-wide trust = session access | Per-session HMAC capability | Phase 87 | Explicit grant model |

**Deprecated/outdated:**
- Client-declared Origin trust — completely abandoned by SEC-06; Origin is validated server-side.
- Query-parameter permission declaration — abandoned by Phase 87's capability `perms` claim.

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Browsers always emit `Origin` on WebSocket upgrades initiated by `new WebSocket()` from a page context | Pitfall 2, D-05 rationale | Low — RFC 6455 §4.1 mandates the client send Origin for browser WS clients. Verified against MDN. [CITED: RFC 6455 §4.1; HTML Living Standard §Fetch integration] |
| A2 | `Origin: null` only arises from sandboxed iframes, `file://`, `data:`/`blob:` contexts (not from regular tailnet-hosted HTTPS pages) | Pitfall 2 | Low — HTML spec. Normal deployment never produces null. |
| A3 | `path.Match` in Go stdlib does not match empty strings for `"*"` pattern — thus `OriginPatterns: []string{"*"}` does not accept truly-empty Origins | Code Examples, Pattern 3 #5 | Low — but also irrelevant because library short-circuits empty Origin before pattern match anyway. |

**Note:** All claims with direct code-reading evidence (library source, capability.go, capability_mw.go, server.go, accept.go) are `[VERIFIED]` not `[ASSUMED]`. The three assumptions above are well-established web standards.

---

## Open Questions

None remain. All design questions were resolved in CONTEXT D-01 through D-14 and confirmed against library source + existing code.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All Phase 88 code | ✓ | go1.26.2 darwin/arm64 | — |
| `github.com/coder/websocket` | Library-layer belt-and-suspenders | ✓ | v1.8.14 (in mod cache) | — |
| `net/http` (stdlib) | ServeMux + middleware | ✓ | stdlib | — |
| `os`, `strings` (stdlib) | D-13 source-grep guards | ✓ | stdlib | — |
| `testing` + `net/http/httptest` | Integration tests | ✓ | stdlib | — |

**Missing dependencies:** None. All Phase 88 code lives in existing packages with existing imports.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + `net/http/httptest` (stdlib); `go test` runner |
| Config file | `go.mod` (no separate test config) |
| Quick run command | `go test ./internal/webserver -run 'TestOrigin\|TestSecurity_.*Origin\|TestRequireAllowedOrigin' -count=1` |
| Full suite command | `GOCACHE=/tmp/go-build-cache go test ./... -count=1` |

### Phase Requirements → Test Map

SEC-06 decomposes into four Success Criteria (SC-1..SC-4). Each SC gets one or more sampling points:

| SC | Behavior | Test Type | Automated Command | File (Wave 0 gap) |
|----|----------|-----------|--------------------|-------------------|
| SC-1 | Upgrade request with `Origin: https://evil.example` → 403 at handshake, BEFORE any capability check | integration | `go test ./internal/webserver -run TestOrigin_RejectsCrossSiteOrigin -v` | ❌ new (Wave 0) |
| SC-1 | Upgrade request with no `Origin` header → 403 (D-05) | integration | `go test ./internal/webserver -run TestOrigin_RejectsMissingOrigin -v` | ❌ new (Wave 0) |
| SC-1 | Origin fails + valid cap → still 403 (short-circuits before cap) | integration | `go test ./internal/webserver -run TestOrigin_ShortCircuitsBeforeCapability -v` | ❌ new (Wave 0) |
| SC-1 | Unit: `requireAllowedOrigin` rejects `"https://EVIL.com"` against `"https://host.ts.net:7443"` | unit | `go test ./internal/webserver -run TestRequireAllowedOrigin_MismatchRejected -v` | ❌ new (Wave 0) |
| SC-1 | Unit: case-mismatch rejected (uppercase scheme) by strict middleware | unit | `go test ./internal/webserver -run TestRequireAllowedOrigin_StrictExactMatch -v` | ❌ new (Wave 0) |
| SC-2 | Upgrade with `Origin == ws.BaseURL()` in tailscale mode completes 101 | integration | `go test ./internal/webserver -run TestOrigin_TailscaleModeAllowedOriginUpgrades -v` | ❌ new (Wave 0) |
| SC-2 | Upgrade with `Origin == ws.BaseURL()` in local mode (self-signed + BindIP) completes 101 | integration | `go test ./internal/webserver -run TestOrigin_LocalModeAllowedOriginUpgrades -v` | ❌ new (Wave 0) |
| SC-3 | Missing `Origin` explicitly rejected with 403 body "forbidden" (contract test for D-05, D-07) | integration | `go test ./internal/webserver -run TestOrigin_MissingOriginReturnsForbiddenBody -v` | ❌ new (Wave 0) |
| SC-4 | `server.go` contains no `OriginPatterns: []string{"*"}` literal | regression (source-grep) | `go test ./internal/webserver -run TestOriginAllowlist_NoAcceptAllRegression -v` | ❌ new (Wave 0) |
| SC-4 | `relay/server.go` contains no `InsecureSkipVerify: true` literal | regression (source-grep) | `go test ./internal/relay -run TestRelayOrigin_NoInsecureSkipVerifyRegression -v` | ❌ new (Wave 0) |
| SC-4 | `handleWSSRelay`'s `AcceptOptions` line references `ws.BaseURL()` (or `ws.allowedOrigins()` helper) | regression (source-grep) | Same as line 1 above (combined test) | ❌ new (Wave 0) |
| SC-4 (belt-and-suspenders) | Bypass middleware (call `websocket.Accept` directly with `OriginPatterns: ws.allowedOrigins()`) → library rejects `Origin: https://evil.example` | integration | `go test ./internal/webserver -run TestAcceptOptions_OriginPatternsRejectsCrossSite -v` | ❌ new (Wave 0) |
| Inversion | Existing `TestSecurity_WebSocketAcceptsCrossSiteOrigin` (from security-review scaffolding) flipped to assert REJECTION | integration | `go test ./internal/webserver -run TestSecurity_WebSocketRejectsCrossSiteOrigin -v` | ❌ new (Wave 0) — **not in-tree yet** (see note below) |
| Relay SC-1 | Relay accepts loopback `Origin: http://127.0.0.1:<port>` | integration | `go test ./internal/relay -run TestServer_LoopbackOriginAccepted -v` | ❌ new (Wave 0) |
| Relay SC-1 | Relay accepts loopback `Origin: http://localhost:<port>` | integration | `go test ./internal/relay -run TestServer_LocalhostOriginAccepted -v` | ❌ new (Wave 0) |
| Relay SC-1 | Relay rejects `Origin: https://evil.example` | integration | `go test ./internal/relay -run TestServer_CrossSiteOriginRejected -v` | ❌ new (Wave 0) |

**Note on test-name inversion:** A grep of the in-tree code confirms `TestSecurity_WebSocketAcceptsCrossSiteOrigin` exists ONLY in `security-review/internal_webserver_server_test.go` (scaffolding that is gitignored / not compiled into any package per phase 87 verification note about the `security-review/` orphan). The in-tree `internal/webserver/capability_test.go` has `TestSecurity_` tests for SEC-02..SEC-05 but no Origin test. Phase 88 adds the Origin test fresh (as `TestSecurity_WebSocketRejectsCrossSiteOrigin`), not an inversion of an existing test. The original scaffold in `security-review/` can be left untouched — it's not compiled.

### Sampling Rate
- **Per task commit:** `go test ./internal/webserver -run 'TestOrigin\|TestRequireAllowedOrigin\|TestSecurity_.*Origin\|TestOriginAllowlist' -count=1` + `go test ./internal/relay -run 'TestServer_.*Origin\|TestRelayOrigin' -count=1` (≈3-5 seconds total).
- **Per wave merge:** `go test ./... -count=1` (full suite, ~15-30 seconds).
- **Phase gate:** Full suite green; `go vet ./...` clean; verification run includes `grep -c 'requireAllowedOrigin' internal/webserver/server.go` ≥ 1 and source-grep regression tests passing.

### Wave 0 Gaps
All Phase 88 tests are net-new. There is no existing Origin-focused test file under `internal/webserver/` or `internal/relay/` to extend. Propose:

- [ ] `internal/webserver/origin_mw_test.go` — unit tests for `requireAllowedOrigin` (7 tests covering mismatch, missing, exact match, case sensitivity, allowed-origin pass, fail-closed-on-nil-listener)
- [ ] `internal/webserver/origin_integration_test.go` — 6 integration tests (cross-site rejected, missing rejected, allowed in tailscale, allowed in local, short-circuit-before-cap, library belt-and-suspenders)
- [ ] `internal/webserver/security_regression_test.go` (or integrate into `capability_test.go` / `origin_mw_test.go`) — `TestOriginAllowlist_NoAcceptAllRegression` (two-condition source-grep for server.go)
- [ ] `internal/relay/origin_test.go` — 4 integration tests (loopback `127.0.0.1`, loopback `localhost`, cross-site rejected, `TestRelayOrigin_NoInsecureSkipVerifyRegression` source-grep)
- [ ] No new framework install; no new shared fixtures (existing `testServer`/`testServerWithHub`/`dialWebServerWS` from `internal/webserver/capability_test_helpers.go` sufficient; `setupTestServer` from `internal/relay/server_test.go` sufficient for relay).

**Expected counts after Phase 88:** 17 new test functions (10 webserver + 4 relay + 3 regression-guards spread across files). Approximate test count delta: +17 `Test*`, +0 `Benchmark*`, +0 `Fuzz*`.

---

## Security Domain

### Applicable ASVS Categories

Reference: OWASP ASVS v4.0.3. Only categories relevant to this phase are listed.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Capability (Phase 87) handles identity — Phase 88 is pre-authZ origin gate. |
| V3 Session Management | no | Sessions are server-owned PTY sessions — not web auth sessions in the ASVS sense. |
| V4 Access Control | yes | Origin gate is a pre-request access control at the transport boundary (V4.1.3 "trusted service layer"). `requireAllowedOrigin` enforces this. |
| V5 Input Validation | yes | `Origin` header is untrusted input. Strict byte-for-byte equality against a server-computed canonical value (V5.1.3 "positive validation"). No normalization, no parsing. |
| V6 Cryptography | no | Phase 88 touches no crypto (capability sig check is Phase 87). |
| V13 API & Web Service | yes | V13.2.3 "CSRF defense for state-changing endpoints" — `Origin` allowlist is a standard WebSocket-CSRF mitigation (RFC 6455 §10.2 recommends it). |

### Known Threat Patterns for Go HTTP/WebSocket

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Cross-site WebSocket hijacking (CSWSH) | Spoofing + Elevation of Privilege | `Origin` allowlist at handshake (this phase) + capability bearer token (Phase 87). Defense-in-depth: both must pass. |
| Accept-all via library defaults | Tampering | Explicit `OriginPatterns` allowlist; source-grep regression guard (D-13) blocks re-introduction of `"*"` or `InsecureSkipVerify: true`. |
| Bypass via middleware-removal code change | Tampering | Library-layer belt-and-suspenders (D-12) — even if middleware is removed by a future routing change, `AcceptOptions.OriginPatterns` still rejects. |
| Information disclosure via differentiated error bodies | Information Disclosure | Single generic `"forbidden"` body for all Origin-rejection cases (D-07; mirrors Phase 87 T-87-08). |
| Missing-Origin non-browser bypass | Spoofing | Missing Origin is explicitly 403'd at middleware (D-05). The library default (accept missing Origin) is defeated by middleware-first ordering. |

---

## Sources

### Primary (HIGH confidence)
- **`github.com/coder/websocket` v1.8.14 source** — `/Users/ken/go/pkg/mod/github.com/coder/websocket@v1.8.14/accept.go` lines 23-82 (`AcceptOptions`), 102-182 (`Accept`/`accept`), 228-264 (`authenticateOrigin`/`match`). Read in full during this research.
- **Phase 87 context, plan, verification** — `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/{87-CONTEXT.md,87-VERIFICATION.md}`. Locks composition pattern, middleware shape, regression-test style.
- **Production code being modified** — `internal/webserver/server.go` (especially lines 321 `BaseURL()`, 352-401 `setupRoutes`, 595-710 `handleWSSRelay`), `internal/webserver/capability_mw.go`, `internal/relay/server.go` (lines 43-143 `handleSession`), `internal/capability/capability_test.go` lines 157-173 (constant-time guard template).
- **CONTEXT.md locked decisions** — D-01 through D-14 explicitly cited throughout.
- **Test infrastructure already in tree** — `internal/webserver/capability_test_helpers.go` (lines 106-221 `testServer`, `testServerWithHub`, `dialWebServerWS`, `selfSignedTLSForTest`, `readPipeWithTimeout`); `internal/relay/server_test.go` lines 20-54 (`setupTestServer`, `dialWS`).

### Secondary (MEDIUM confidence)
- **RFC 6455 WebSocket Protocol** §4.1 (client Origin header), §10.2 (origin-based security model) — standard reference, cross-referenced against library behavior.
- **OWASP ASVS v4.0.3** — V4.1.3, V5.1.3, V13.2.3.

### Tertiary (LOW confidence)
- None. All load-bearing claims are verified from source.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new deps; existing `coder/websocket` pin verified against mod cache.
- Architecture: HIGH — wiring, composition order, and belt-and-suspenders all cross-referenced against library source and Phase 87 precedent.
- Pitfalls: HIGH — each pitfall traced to either a code line, library source line, or RFC/spec citation.
- Validation: HIGH — sampling points mapped to specific SC IDs with runnable commands.

**Research date:** 2026-04-21
**Valid until:** 2026-05-21 (30 days — coder/websocket is a stable, slow-moving library; library semantics unlikely to change within this window)

---

## Project Constraints (from CLAUDE.md)

The following directives from `/Users/ken/dev/CLAUDE.md` apply to Phase 88 and MUST be honored by every plan:

- **Go conventions:** `go fmt`, `golangci-lint`, context-aware functions (`ctx context.Context`). New test functions must follow `snake_case`-free `TestX_Y` naming (consistent with Phase 87).
- **Testing:** `pytest`/`testing`/`vitest` per language. Phase 88 is Go → `go test`. 80%+ coverage in critical components — Phase 88 code is security-critical, so aim for 100% branch coverage in `requireAllowedOrigin` and the allowlist helper.
- **Silent fallbacks forbidden:** CLAUDE.md "Silent Fallbacks" warns against `or {}`. Equivalent in Go: no silent nil-return. When `ws.listener == nil`, `BaseURL()` returns `""` — middleware MUST fail closed (403), not silently pass.
- **Chesterton's Fence:** Before removing `InsecureSkipVerify: true` from relay or `OriginPatterns: []string{"*"}` from webserver, articulate why they exist. Answer: review-era placeholder, never meant for production; Phase 88 replaces both.
- **Make beliefs pay rent:** Each plan should state an explicit prediction before a test run (e.g. "this commit makes TestOrigin_RejectsCrossSiteOrigin flip RED→GREEN").
- **Notice confusion:** If a test behaves unexpectedly, stop and identify why. Documented pitfalls (1-7) cover the known surprises.
- **Line of retreat:** "I don't know" is always available — prefer open question in PLAN over speculation.
- **LSP over Grep/Read:** Use `LSP.goToDefinition`/`findReferences`/`hover` for code navigation during implementation.

---

## RESEARCH COMPLETE
