# Phase 88: WebSocket Handshake Security - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Close the cross-site WebSocket hijacking vector (security-review Finding 3 / SEC-06) by adding an Origin allowlist check at the WSS handshake. The Origin check layers on top of Phase 87's capability check — both must pass for `/sessions/{id}/ws` to complete the upgrade.

Only the WebSocket upgrade route is gated by the new Origin check. HTML/JSON routes remain protected by the capability token alone (capabilities are bearer tokens already resistant to the CSRF-adjacent attack the Origin check addresses, and plain navigation does not set `Origin` reliably).

Both the tailnet-facing `internal/webserver/` AND the localhost-only `internal/relay/server.go` are modified in this phase — `InsecureSkipVerify: true` is removed from the relay as well, per SC-4's literal reading ("accept-all is gone from the code path"). The relay's replacement policy is loopback-only origins, derived dynamically from its listener address.

</domain>

<decisions>
## Implementation Decisions

### Allowlist Composition
- **D-01:** Origin allowlist is sourced **dynamically from `ws.BaseURL()`** at upgrade time. The middleware reads `ws.listener.Addr()` under `ws.mu.RLock()` and composes the expected origin string (`scheme://host:port`) every request. Handles random-port fallback automatically — whatever port the listener holds is what the allowlist reflects.
- **D-02:** "Configured same-origin" in SEC-06 is interpreted as **reflective only**. The allowlist is strictly the server's own current serving origin — no new `settings.json` field for user-configurable extra origins, no Security-section UI for adding origins in v3.1. This can be added in a future phase if a reverse-proxy or alternate-hostname use case emerges.
- **D-03:** Matching is **strict byte-for-byte exact match**. No case-folding, no port-stripping, no normalization. Browsers emit a canonical Origin; the server's own `BaseURL()` already returns a canonical form. Any mismatch is a 403 and the user re-opens the link from the share panel. Simplest, most auditable, lowest attack surface.
- **D-04:** Compare the **origin tuple only** (`scheme://host:port`). `Origin` headers never carry a path; `BaseURL()` already returns the origin tuple without a path, so direct equality works.

### Missing-Origin Policy (resolves Phase 87 pending todo)
- **D-05:** An upgrade request with **no `Origin` header is always rejected with 403**. Browsers always set `Origin` on WebSocket upgrade. Non-browser clients (curl, native tools) have the localhost relay for programmatic access; the tailnet-facing WSS endpoint is designed for the browser terminal page. If a future non-browser use case emerges, a dedicated path will be added explicitly.
- **D-06:** The Origin check applies **only to the WebSocket upgrade route** (`GET /sessions/{id}/ws`), not to the HTML page route (`GET /sessions/{id}`), the info route (`GET /api/sessions/{id}/info`), the listing route (`GET /api/sessions`), or any other capability-gated route. Plain navigation does not set `Origin` reliably, and capability tokens already provide bearer-token CSRF resistance for those routes.
- **D-07:** Rejection body is a **generic `"forbidden"` with status 403** for both missing-Origin and wrong-Origin cases — no distinction in the response body between "Origin absent", "Origin mismatched", or "capability failure". Matches Phase 87's T-87-08 information-disclosure defense (all authz failures collapse to a single response shape).

### Relay Scope Inclusion
- **D-08:** Phase 88 **also removes `InsecureSkipVerify: true`** from `internal/relay/server.go:59`. Reading SC-4 literally: "OriginPatterns [\"*\"] / InsecureSkipVerify: true accept-all is gone from the code path." Leaving the flag in place is a landmine — any future change exposing the relay to a non-loopback listener would silently inherit accept-all. This supersedes Phase 87 D-21's "relay stays untouched" stance for this specific defensive cleanup.
- **D-09:** The relay's replacement Origin policy is **loopback-only**. Allowlist is `{http://localhost:<port>, http://127.0.0.1:<port>, https://localhost:<port>, https://127.0.0.1:<port>}` — schemes included because the relay may be fronted by TLS in the current deployment; ports derived dynamically from the relay's listener address. If the relay is ever rebound to a non-loopback address in a future change, the maintainer has to consciously add the new origin — which is exactly the landmine-avoidance property we want.

### Enforcement
- **D-10:** A new **`requireAllowedOrigin(next)` HTTP middleware** wraps the `/sessions/{id}/ws` route, composed **outside** `requireCapability`. Composition order (outermost → innermost): Basic Auth (local mode only) → Origin → Capability → `handleWSSRelay`. The middleware rejects with `http.Error(w, "forbidden", http.StatusForbidden)` before `websocket.Accept` is ever called. This keeps the rejection response shape clean (not library-controlled), makes the check testable in isolation, and ensures Origin rejection short-circuits the capability work.
- **D-11:** The middleware **re-evaluates the allowlist per request** (reads `ws.BaseURL()` under `ws.mu.RLock()` on every upgrade). Negligible cost relative to handshake overhead; avoids a snapshot/invalidation code path.
- **D-12:** As **defense in depth**, `websocket.AcceptOptions.OriginPatterns` is ALSO set to the strict allowlist (never `[]string{"*"}` again). If the middleware is ever bypassed by a future route wiring bug, the `coder/websocket` library's own check still rejects the handshake. The `OriginPatterns: []string{"*"}` line at `internal/webserver/server.go:632` is replaced with an explicit allowlist built from `ws.BaseURL()`.

### Anti-Regression Guard (SC-4)
- **D-13:** A **source-grep regression test** (mirroring Phase 87's constant-time regression guard pattern) asserts:
  1. `internal/webserver/server.go` contains NO `"*"` literal inside an `OriginPatterns` slice
  2. `internal/relay/server.go` contains NO `InsecureSkipVerify: true` literal
  3. The expected `OriginPatterns` line in `handleWSSRelay` references `ws.BaseURL()` (positive confirmation the allowlist is wired)
  The test reads the source files and asserts on string content — future maintainers cannot silently reintroduce accept-all without the test failing. Low-tech, high-signal, lives forever.

### Observability
- **D-14:** **No logs, no metrics** on Origin rejection. Matches Phase 87's minimal-observability stance in security-layer code (CONTEXT D-22 and Claude's Discretion). An attacker probing origins should not be able to fill disk; a legitimate rejection is a client error the user resolves by reopening the link from the share panel.

### Claude's Discretion
- Exact package location / filename for the new middleware (likely `internal/webserver/origin_mw.go` alongside `capability_mw.go`, or inline in `server.go` if trivial)
- Helper function shape for composing the allowlist string(s) — single function on `*WebServer`, or a free function taking `BaseURL` as input
- Relay-side helper for deriving loopback origins from the listener address — placement and naming
- Test organization: unit tests for origin matching, integration tests for the full upgrade flow, source-grep regression tests
- Exact error strings in the regression test failure messages (as long as they are actionable)
- Whether to inline the allowlist build at the `AcceptOptions` call site or pull it into a shared helper used by both the middleware and `AcceptOptions`

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Threat Model
- `.planning/REQUIREMENTS.md` — SEC-06 acceptance criteria
- `.planning/ROADMAP.md` §Phase 88 — Success criteria (4 must-be-TRUE conditions)
- `security-review/SECURITY_REVIEW.md` §Finding 3 — Cross-site WebSocket hijacking exploit scenario and recommended fix
- `security-review/SECURITY_DYNAMIC_VALIDATION.md` — Finding 3 dynamically confirmed against current code (`TestSecurity_WebSocketAcceptsCrossSiteOrigin` currently PASSES — Phase 88 inverts it)
- `security-review/internal_webserver_server_test.go` — Review-supplied scaffolding (reference only)

### Existing Code That Must Change
- `internal/webserver/server.go:627-633` — `handleWSSRelay`'s `websocket.Accept` call with `OriginPatterns: []string{"*"}` — replace with strict allowlist derived from `ws.BaseURL()`
- `internal/webserver/server.go` (route setup, currently `setupRoutes` circa line 352+) — wire `requireAllowedOrigin` around the `/sessions/{id}/ws` route
- `internal/webserver/server.go:321` — `BaseURL()` — the allowlist source (no change, but heavily referenced)
- `internal/relay/server.go:59-62` — `websocket.Accept` with `InsecureSkipVerify: true` — replace with loopback-only allowlist
- Existing test `TestSecurity_WebSocketAcceptsCrossSiteOrigin` (security-review scaffolding, or its in-tree equivalent under `internal/webserver/`) — **invert**: sending `Origin: https://evil.example` must now produce 403, not a successful upgrade

### Phase 87 Handoffs
- `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-CONTEXT.md` — Capability middleware composition pattern (Phase 88 layers on top of the same pattern)
- `.planning/milestones/v3.1-phases/87-capability-based-session-authorization/87-VERIFICATION.md` — Verified state of Phase 87 (Origin allowlist was explicitly deferred from Phase 87 to Phase 88 — see D-25 in that doc and the Plan 03 inline comment at server.go:628-632)

### Existing Patterns to Mirror
- `internal/webserver/capability_mw.go` — Middleware shape (`func(http.HandlerFunc) http.HandlerFunc`) and route-wrapping pattern; `requireAllowedOrigin` follows the same shape
- Phase 87's constant-time regression guard (test that source-greps `internal/capability/capability.go` for `hmac.Equal` presence / `bytes.Equal` absence) — mirror for the Origin anti-regression test

### Library Reference
- `github.com/coder/websocket` — `AcceptOptions.OriginPatterns []string` semantics (exact-match on host, supports wildcards, empty slice means same-origin-only based on Host header). Downstream agents should verify semantics against the library version in `go.mod` before locking the allowlist format.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/webserver/capability_mw.go` — Middleware shape already established; `requireAllowedOrigin` is a direct sibling.
- `internal/webserver/server.go:321` (`BaseURL()`) — Already returns the canonical `scheme://host:port` form needed for matching. Reads `ws.listener.Addr()` under `ws.mu.RLock()`.
- `github.com/coder/websocket` — Already imported; `AcceptOptions.OriginPatterns` is the native way to express the allowlist on the library side.
- Existing `TestSecurity_WebSocketAcceptsCrossSiteOrigin` test (from security-review scaffolding or the in-tree version) — inversion is a single-line change plus a new counterpart that asserts the allowed-origin case still succeeds.

### Established Patterns
- HTTP routing uses Go 1.22 `http.ServeMux` with path values. Middleware wrappers compose via `mux.HandleFunc("GET /sessions/{id}/ws", requireAllowedOrigin(requireCapability(handleWSSRelay)))`.
- Source-grep regression tests (Phase 87 precedent). They read the source file, assert on string contents, and fail with actionable messages.
- Settings-adjacent code uses mode 0600 and atomic writes — irrelevant here since Phase 88 adds no persisted state.
- Error responses use `http.Error(w, msg, code)` with short messages — consistent with the Phase 87 collapsed-error-body policy.

### Integration Points
- **Route wiring:** `setupRoutes` in `internal/webserver/server.go` is where `requireAllowedOrigin` plugs into the `/sessions/{id}/ws` handler. Capability middleware is already the innermost wrapper; Origin is composed around it.
- **Listener mutex:** `ws.mu.RLock()` is already used by `BaseURL()`. The new middleware's per-request read is cheap under this existing lock.
- **Relay listener:** `internal/relay/server.go` derives its origin allowlist from its own `net.Listener` address (different from the webserver's; relay is localhost-only). No shared allowlist code between the two packages — they're independent defensive boundaries.

</code_context>

<specifics>
## Specific Ideas

- The middleware-first enforcement with a library-side belt-and-suspenders (`OriginPatterns` on `AcceptOptions`) is the pattern the user wants — rejection happens at the HTTP layer with a clean 403, and even if the middleware is ever bypassed by a routing bug, the library still rejects.
- The source-grep regression test is explicitly chosen to mirror Phase 87's proven pattern. The user has confidence in that style of guard and wants consistency.
- The relay cleanup is a deliberate supersession of Phase 87 D-21. The user read SC-4 literally and agreed that leaving `InsecureSkipVerify: true` in the code is a landmine.
- Missing-Origin rejection is strict by design — the v3.1 threat model is specifically about browser hijacking, and the localhost relay exists for non-browser use cases.

</specifics>

<deferred>
## Deferred Ideas

### User-configurable additional origins
If a reverse-proxy, alternate Tailscale node name, or other deployment topology requires a non-reflective origin entry, add a `settings.json` field (e.g., `extra_allowed_origins: []string`) + a Security-section UI input in a future phase. Not needed for v3.1 — no known user case.

### CLI/native direct-to-WSS clients
If a future feature needs non-browser tools to connect to `/sessions/{id}/ws` over tailnet, a dedicated path (or explicit opt-in to missing-Origin acceptance with a capability) would be added. v3.1 keeps the rejection strict.

### Observability for Origin rejections
Debug logs or metrics for rejected upgrades could be added later if a support burden emerges from users hitting the gate. v3.1 ships silent.

### Snapshot-based allowlist
If a profile reveals the per-request `BaseURL()` read becomes a bottleneck (it won't — handshakes are rare), snapshot at `Start()` with an invalidation hook. Deferred because the current read is effectively free.

### Relay exposure hardening beyond Origin
If `internal/relay/server.go` is ever exposed to tailnet or LAN in a future phase, additional hardening (capability checks, Basic Auth, etc.) would be required. v3.1 assumes it stays localhost-only and relies on the listener bind to enforce that.

</deferred>

---

*Phase: 88-websocket-handshake-security*
*Context gathered: 2026-04-21*
