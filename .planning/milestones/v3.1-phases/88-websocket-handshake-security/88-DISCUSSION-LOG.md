# Phase 88: WebSocket Handshake Security - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-21
**Phase:** 88-websocket-handshake-security
**Areas discussed:** Allowlist composition, Missing-Origin policy, Relay scope inclusion, Enforcement & regression guard

---

## Allowlist Composition

### Q1: How should the Origin allowlist be sourced?

| Option | Description | Selected |
|--------|-------------|----------|
| Dynamic from BaseURL() | Snapshot ws.BaseURL() at upgrade time; handles random-port fallback automatically | ✓ |
| Static from Config at Start() | Compute allowed origins once at listener open and cache; requires invalidation | |
| Both modes at once | Always allow both FQDN and BindIP regardless of current Mode | |

**User's choice:** Dynamic from BaseURL() (Recommended)

### Q2: What should SEC-06's 'configured same-origin' mean?

| Option | Description | Selected |
|--------|-------------|----------|
| Reflective only — no user config | Allowlist is strictly derived from server's current serving URL | ✓ |
| User-configurable extras | Add settings.json extra_origins field with UI | |
| Configurable but no UI yet | Add config field + persistence, no UI | |

**User's choice:** Reflective only — no user config (Recommended)

### Q3: How should the allowlist handle origin variations for the same server?

| Option | Description | Selected |
|--------|-------------|----------|
| Strict exact match | Scheme + host + port must equal BaseURL() byte-for-byte | ✓ |
| Normalize host + port | Lowercase FQDN, strip default ports, canonicalize before compare | |
| Suffix match on FQDN | Allow any :port on same host | |

**User's choice:** Strict exact match (Recommended)

### Q4: Allow BaseURL() as-is, or strip to origin tuple?

| Option | Description | Selected |
|--------|-------------|----------|
| Scheme://host:port only | Compare only origin tuple — matches Origin header semantics | ✓ |
| Full URL match | Match full URL including base path | |

**User's choice:** Scheme://host:port only (Recommended)

---

## Missing-Origin Policy

### Q1: What should happen when the WebSocket upgrade request arrives with NO Origin header?

| Option | Description | Selected |
|--------|-------------|----------|
| Reject all missing-Origin | Strictest SEC-06 reading; 403 for any non-Origin request | ✓ |
| Allow if capability valid | Skip Origin check when header absent; cap alone authorizes | |
| Allow unconditionally | Defer to capability entirely | |

**User's choice:** Reject all missing-Origin (Recommended)

### Q2: Non-browser client expectations?

| Option | Description | Selected |
|--------|-------------|----------|
| None expected | WSS endpoint is for terminal.html; non-browser uses localhost relay | ✓ |
| CLI clients may connect directly | Future CLI tools may want remote WSS | |
| Unknown — keep permissive | Don't commit to strict rejection yet | |

**User's choice:** None expected (Recommended)

### Q3: Does the Origin check apply to HTML/JSON routes or only the WSS upgrade?

| Option | Description | Selected |
|--------|-------------|----------|
| WebSocket upgrade only | SEC-06 is about CSWSH specifically; caps protect other routes | ✓ |
| All capability-gated routes | Defense in depth but breaks plain navigation | |
| All routes | Universal check, breaks /dashboard and /join too | |

**User's choice:** WebSocket upgrade only (Recommended)

### Q4: How much info should the rejection response disclose?

| Option | Description | Selected |
|--------|-------------|----------|
| Generic 403 'forbidden' | Matches Phase 87 T-87-08 info-disclosure defense | ✓ |
| 403 'origin not allowed' | Distinguish Origin rejection from cap rejection | |

**User's choice:** Generic 403 'forbidden' (Recommended)

---

## Relay Scope Inclusion

### Q1: Should Phase 88 remove InsecureSkipVerify: true at internal/relay/server.go:59?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — clean it up now | SC-4 says accept-all is gone from code path; leaving it is a landmine | ✓ |
| Defer to a follow-up phase | Stick with Phase 87 D-21's out-of-scope stance | |
| Just lock it behind a build tag/comment | Document stronger; no behavior change | |

**User's choice:** Yes — clean it up now (Recommended)

### Q2: What should the relay's replacement Origin policy be?

| Option | Description | Selected |
|--------|-------------|----------|
| Allow only loopback | Derive from listener address: localhost + 127.0.0.1 at current port | ✓ |
| Same pattern as webserver | Dynamic from relay's BaseURL — less protective if bind address changes | |
| Reject Origin entirely | Purely programmatic endpoint; browsers shouldn't talk to it | |

**User's choice:** Allow only loopback (Recommended)

---

## Enforcement & Regression Guard

### Q1: Where should the Origin check be enforced?

| Option | Description | Selected |
|--------|-------------|----------|
| HTTP middleware BEFORE capability check | requireAllowedOrigin wraps outside requireCapability; clean 403 response | ✓ |
| coder/websocket's OriginPatterns only | Library-native check; rejection shape is library-controlled | |
| Both | Middleware primary + OriginPatterns on Accept as redundant inner check | |

**User's choice:** HTTP middleware BEFORE capability check (Recommended)

*Note: The recommended option's description includes "OriginPatterns on websocket.Accept gets set to the empty slice or a strict allowlist too as defense in depth" — this is captured as D-12 (belt-and-suspenders).*

### Q2: Re-evaluate allowlist per request, or snapshot at Start()?

| Option | Description | Selected |
|--------|-------------|----------|
| Re-evaluate per request | Reads BaseURL() under RLock per upgrade; handles in-process port changes | ✓ |
| Snapshot at Start() | Cache at listener open; zero-cost but requires invalidation | |

**User's choice:** Re-evaluate per request (Recommended)

### Q3: How should SC-4's anti-regression guard be implemented?

| Option | Description | Selected |
|--------|-------------|----------|
| Source-grep test | Mirror Phase 87 constant-time guard pattern | ✓ |
| Behavioral integration test | Runtime test with evil.example Origin | |
| Both — grep + behavioral | Source-level + logic-level regression catches | |

**User's choice:** Source-grep test (Recommended)

### Q4: Should there be observability on Origin rejection?

| Option | Description | Selected |
|--------|-------------|----------|
| No logs, no metrics | Matches Phase 87 minimal-observability in security-layer code | ✓ |
| Structured debug log per rejection | Single debug line per reject; floodable | |
| Rate-limited warning log | Log + rate limit; adds state | |

**User's choice:** No logs, no metrics (Recommended)

---

## Claude's Discretion

- Exact filename/package location for the middleware (`origin_mw.go` alongside `capability_mw.go`, or inline)
- Allowlist-building helper shape (method on `*WebServer` vs free function)
- Relay-side loopback origin helper placement and naming
- Test organization: unit/integration/source-grep breakdown
- Exact error strings in regression test failure messages
- Whether to share allowlist code between the middleware and `AcceptOptions` call site

## Deferred Ideas

- User-configurable additional origins (settings.json + Security UI) — future phase if needed
- CLI/native direct-to-WSS clients — dedicated path if a use case emerges
- Observability for Origin rejections — add later if support burden emerges
- Snapshot-based allowlist — deferred because per-request read is effectively free
- Relay exposure hardening beyond Origin — not needed while relay stays localhost-only
