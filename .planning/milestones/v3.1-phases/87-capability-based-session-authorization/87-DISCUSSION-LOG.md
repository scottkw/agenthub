# Phase 87: Capability-Based Session Authorization - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in 87-CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-19
**Phase:** 87-capability-based-session-authorization
**Areas discussed:** Token format & algorithm, Grant UX flow, Capability lifetime & revocation-for-now, Dashboard & enumeration model

---

## Gray Area Selection

| Option | Description | Selected |
|--------|-------------|----------|
| Token format & algorithm | HMAC vs ed25519, compact vs JWT, claim fields | ✓ |
| Grant UX flow | How sharing happens, URL location, read/write model, QR contents | ✓ |
| Capability lifetime & revocation-for-now | Expiry, toggle semantics, session-end, key rotation | ✓ |
| Dashboard & enumeration model | What /dashboard becomes, /api/sessions behavior, local-mode interaction | ✓ |

**User's choice:** All four.
**Notes:** User wanted to address every remaining decision before handing off to planning.

---

## Token format & algorithm

### Signing algorithm

| Option | Description | Selected |
|--------|-------------|----------|
| HMAC-SHA256 | Symmetric, 32-byte secret, stdlib crypto/hmac | ✓ |
| Ed25519 | Asymmetric; no benefit for single-daemon verifier | |
| HMAC-SHA512/256 | Truncated SHA-512; non-standard for no gain | |

### Wire format

| Option | Description | Selected |
|--------|-------------|----------|
| Compact: base64url(claims).base64url(sig) | Two-segment JWS-like, no jwt library | ✓ |
| Full JWT (JWS) | Standard but pulls library + alg-confusion CVE surface | |
| Opaque random token + server store | Breaks "survives daemon restart via key only" | |

### Claim set

| Option | Description | Selected |
|--------|-------------|----------|
| sid + perms + iat + grant_id + v | Session binding, permission, timestamp, revocation ID, schema version | ✓ |
| sid + perms only | No path to future revocation granularity | |
| sid + perms + exp + iss (JWT-style) | Forces finite expiry, conflicts with SC-5 | |

### Key storage

| Option | Description | Selected |
|--------|-------------|----------|
| New field on daemonSettings | Reuses existing persistence | (initially recommended, superseded by user discussion) |
| Separate file (capability.key) | Keeps secret out of settings.json | ✓ (via user-driven clarification) |
| OS Keychain | User asked for justification; deferred to future issue | |

**User's choice:** HMAC-SHA256; compact format; full claim set; dedicated `capability.key` file.
**Notes:** User pushed back on "overkill" label for OS Keychain. Extended discussion covered defense-in-depth, Time Machine leak risk, and audit optics — weighed against cross-platform keychain complexity, service-mode constraints (macOS LaunchDaemon can't read user keychain; Linux systemd system units lack D-Bus; headless Linux has no keyring daemon), and test fragility. User asked if there was a way to give the user a choice. Agreed to ship file-backed `KeyStore` in v3.1, track keychain-backed implementation as a future GitHub issue + v3.2 deferred item. Resolution landed on dedicated `capability.key` file (not embedded in `settings.json`) to keep secret out of the file users might share when debugging.

---

## Grant UX flow

### Grant gesture

| Option | Description | Selected |
|--------|-------------|----------|
| Repurpose existing web-serving toggle | Toggle ON = issue caps; OFF = revoke | ✓ |
| New "Share" button separate from toggle | Adds two concepts users must distinguish | |
| Automatic on GUI attach, explicit for web | Splits permission model by client | |

### Token location in URL

| Option | Description | Selected |
|--------|-------------|----------|
| Query parameter: ?cap=<token> | Simplest, works for HTML + WS upgrade | ✓ |
| URL fragment: #cap=<token> | Extra JS plumbing for marginal hiding benefit | |
| Path segment: /sessions/{id}/{token} | Ambiguous vs future route params | |

### Read/write representation

| Option | Description | Selected |
|--------|-------------|----------|
| Two capabilities per session (read-only + read-write) | User picks which URL/QR to share | ✓ |
| Single capability with user-selected perms at issue time | Regenerating invalidates prior link | |
| Single read-write capability; viewer opts into read-only via client | Violates SEC-04 | |

### QR contents

| Option | Description | Selected |
|--------|-------------|----------|
| Full URL with capability | Preserves current scan-and-go UX | |
| Short code the user types on a landing page | User-driven choice — see driver question | ✓ |
| Session URL with no capability | Breaks scan-and-go flow | |

### Driver for short-code choice

| Option | Description | Selected |
|--------|-------------|----------|
| Security only (photograph risk) | | |
| User-intent proof only | | |
| Both — photograph risk + intent confirmation | | ✓ |
| Reconsider QR-with-URL | | |

### Short code semantics

| Option | Description | Selected |
|--------|-------------|----------|
| Single-use, short-TTL exchange for session-bound capability | First exchange consumes; TTL covers leak window | ✓ |
| Capability itself compressed | Not cryptographically possible | |
| Long-lived reusable code | Reintroduces server-side token store | |

### QR → session flow

| Option | Description | Selected |
|--------|-------------|----------|
| QR encodes join URL with code pre-filled; user taps Join | Intent = the tap; no typing friction | ✓ |
| QR encodes code only; user navigates to /join manually | Strongest intent but worst UX | |
| QR opens /join that requires code re-entry | Overengineered | |

### Short code format

| Option | Description | Selected |
|--------|-------------|----------|
| Base32 with dashes (A7K-4P2N) | Unambiguous, typable, dictatable | ✓ |
| 6-digit numeric | Only ~20 bits entropy | |
| Full base64url | Harder to dictate for no gain at 5-min TTL | |

### TTL

| Option | Description | Selected |
|--------|-------------|----------|
| 5 minutes | Standard OTP-exchange window | ✓ |
| 10 minutes | More slack, small attack-window increase | |
| 1 minute | Too tight for real handoff | |

**User's choice:** Web-serving toggle as grant; `?cap=` query param; two capabilities per session; short-code + landing page; both security and intent drive the QR choice; single-use 5-min exchange; pre-filled join URL with tap; base32-dashed code; 5 min TTL.
**Notes:** User went against the recommendation for QR contents, preferring the security-and-intent model over scan-and-go. User asked a clarifying question to confirm the short code remained one-time-use even with the pre-filled URL flow; confirmed yes.

---

## Capability lifetime & revocation-for-now

### Expiry claim

| Option | Description | Selected |
|--------|-------------|----------|
| No expiry — valid until session end / key rotate | Aligns with SC-5 and chosen claim set | ✓ |
| Finite TTL (e.g. 30 days) | Silent breakage of saved URLs | |
| User-configurable per-grant TTL | Defer to v3.2 revocation UI | |

### Web-off behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Revoke: discard grant_ids; re-enable requires fresh share | "Stop sharing = stop access" mental model | ✓ |
| Pause: caps remain valid if re-enabled | Confusing model | |
| Revoke only read-write | Asymmetric, confusing | |

### Session-end behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Implicit: session ID no longer exists, server returns 404 | No extra code path | ✓ |
| Explicit revoke list on session-end | Unneeded storage | |

### Key rotation UI

| Option | Description | Selected |
|--------|-------------|----------|
| Settings button: "Regenerate signing key (invalidates all shared links)" | In-product panic button | ✓ |
| No UI — users delete capability.key manually | No in-product recourse | |
| Automatic rotation on schedule | Contradicts SC-5 | |

**User's choice:** No expiry; web-off revokes; session-end is implicit; Settings button for key rotation.
**Notes:** User accepted all four recommendations.

---

## Dashboard & enumeration model

### /dashboard role

| Option | Description | Selected |
|--------|-------------|----------|
| Landing page: join-code form + QR scan, no list | Matches "no enumeration without grant" | ✓ |
| Keep dashboard, gate with admin capability | Adds second capability type + scope rules | |
| Remove /dashboard entirely | Bigger behavioral change than needed | |

### /api/sessions behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Requires session-scoped cap; returns only that session | Collapses enumeration to self-describe | ✓ |
| Remove endpoint entirely | Cleaner but breaks dashboard REST contract | |
| Keep public; return empty when un-capped | Confusing soft-fail | |

### Local-mode interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Basic Auth AND capability both required | Defense in depth | ✓ |
| Basic Auth sufficient; capability bypassed | Reintroduces master-key model | |
| Drop Basic Auth; capability alone | Out of Phase 87 scope | |

### internal/relay/server.go scope

| Option | Description | Selected |
|--------|-------------|----------|
| Only modify internal/webserver/server.go | Localhost relay not in review scope | ✓ |
| Apply capability to both | Unneeded code + tests | |
| Remove internal/relay/server.go if unused | Separate tech-debt cleanup | |

**User's choice:** Landing page; single-session cap returns one session; both auth mechanisms required in local mode; only webserver package in scope.
**Notes:** User accepted all four recommendations.

---

## Claude's Discretion

- Shape of the `/join` page HTML — copy, error states, layout.
- Go package layout for the new capability module.
- Middleware pattern for applying capability checks.
- Test organization (unit vs integration vs fuzz).
- Naming for new daemon API methods.
- Error response bodies / status codes for capability failures.
- Observability hooks (kept minimal).

## Deferred Ideas

- **OS Keychain-backed key storage** — user requested a future GitHub issue; covers keychain backends, service-mode handling, settings toggle, migration path. Targeted for v3.2+.
- **Revocation / audit UI** — per-grant revoke, audit log, rate limiting on /join, per-REQUIREMENTS.md Future Requirements.
- **Per-user identity** — SSO/OIDC deferred.
- **External KMS** — out of scope per REQUIREMENTS.md.
- **Pause/suspend session semantics** — not a current feature.
- **Removing internal/relay/server.go** — separate tech-debt effort.
