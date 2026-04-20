# Phase 87: Capability-Based Session Authorization - Context

**Gathered:** 2026-04-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace tailnet-wide trust with explicit, server-issued, HMAC-signed capability tokens that gate:
- Session listing (`GET /api/sessions`)
- Per-session metadata (`GET /sessions/{id}`, `GET /api/sessions/{id}/info`)
- WebSocket relay (`GET /sessions/{id}/ws`)
- Write permission (server rejects `MsgInput` frames without a write-capability)

The signing key persists across daemon restarts so previously-shared URLs remain valid. Capability rotation/revocation UI, per-user identity, and audit logs remain out of scope for this phase (deferred to v3.2+).

Only the tailnet-facing `internal/webserver/` is in scope. The localhost-only `internal/relay/server.go` is NOT modified in this phase.

</domain>

<decisions>
## Implementation Decisions

### Token Format & Cryptography
- **D-01:** Signing algorithm is **HMAC-SHA256** (stdlib `crypto/hmac`). Symmetric — no public key distribution, signing and verifying share the secret, rotation is a single-value swap.
- **D-02:** Wire format is a compact two-segment `base64url(claims).base64url(sig)` (JWS-like, no JWT header). No `alg` field in the payload; algorithm is fixed and versioned via the `v` claim. Produces tokens ~120–160 chars.
- **D-03:** Claim set is `{ sid, perms, iat, grant_id, v }`:
  - `sid` — session ID the capability is bound to (enforces SEC-03 session-ID binding)
  - `perms` — `"read"` or `"read,write"` (enforces SEC-04)
  - `iat` — issued-at UNIX timestamp
  - `grant_id` — 128-bit random ID, enables future revocation and distinguishes grants for the same session
  - `v` — claim schema version (forward-compat)
  - **No `exp` claim** — capabilities live until session end, web-off, or signing-key rotation.
- **D-04:** Signing key (32 random bytes) is stored in a dedicated file `capability.key` (mode 0600) inside the daemon config directory, NOT inside `settings.json`. Generated via `crypto/rand` on first daemon start if missing; saved atomically alongside existing settings persistence.
- **D-05:** Key storage is abstracted behind a `KeyStore` interface with methods `Load() ([]byte, error)`, `Save(key []byte) error`, and `Location() string`. v3.1 ships only the `FileKeyStore` implementation. The interface exists now specifically so a keychain-backed implementation can land in a future phase without protocol or API changes.

### Grant UX Flow
- **D-06:** The existing **web-serving toggle** becomes the grant gesture. Flipping web-serving ON for a session issues capabilities. Flipping OFF revokes them (see D-15). The current auto-enable-on-session-create behavior in `internal/daemon/api.go:292` is **removed** to satisfy SEC-01.
- **D-07:** Each grant issues **two capabilities per session**: a read-only capability and a read-write capability. GUI/CLI/TUI expose these as separate "Copy Read-Only Link" and "Copy Full Access Link" actions. This gives users choice without requiring regeneration and gives each capability its own `grant_id` for future revocation granularity.
- **D-08:** Capabilities appear in URLs as a **query parameter**: `https://host/sessions/{id}?cap=<token>`. The WebSocket upgrade and HTML page load read the capability from `r.URL.Query().Get("cap")`. Fragment and path-segment alternatives were rejected as adding complexity for marginal benefit.
- **D-09:** QR codes encode a **join-code exchange URL**, not the capability directly. Format: `https://host/join?code=<base32-dashed>`. A scanner lands on a `/join` page with the code pre-filled, shows session name + permission level, and waits for the user to tap "Join." The tap sends the code to a server endpoint which exchanges it for the real capability (as a redirect to `/sessions/{id}?cap=<token>`). Rationale: prevents leaked photographs/screen recordings from granting access, and provides an explicit intent-confirmation step.
- **D-10:** Short join codes are **base32 with dashes**, 8 characters in two groups of 4 (e.g. `A7K-4P2N`). Base32 avoids `0/O` and `1/I/l` ambiguity. ~40 bits of entropy — sufficient for a 5-minute single-use window.
- **D-11:** Join codes are **single-use, 5-minute TTL**. Consumed on first successful exchange. Daemon keeps an in-memory map of `code → (capability, grant_id, expiry)`; entries are removed on use or TTL expiry. Codes are NOT persisted across daemon restart — if the daemon restarts during the 5-min window, the user regenerates. This is acceptable because short codes are an ephemeral sharing artifact, not a durable credential.

### Capability Lifetime & Revocation-for-Now
- **D-12:** Capabilities **have no expiry claim**. They remain valid until one of three events:
  1. The session ends (capability's `sid` no longer resolves → 404)
  2. Web-serving is toggled off for the session (see D-15)
  3. The signing key is rotated (see D-16)
- **D-13:** Session end implicitly invalidates capabilities. No explicit revoke list is needed — the session ID binding already provides implicit revocation. The existing 10-second post-exit grace period (`onExit` callback in `internal/daemon/api.go`, D-12 from v1.11) still applies, then the session disappears from the registry and its capabilities match nothing.
- **D-14:** A **per-session persisted grant list** tracks which `grant_id`s are currently valid for each session. The list is consulted on every authz check alongside signature verification — signature must be valid AND `grant_id` must be in the session's current grant list AND `sid` must match the running session.
- **D-15:** Toggling web-serving OFF for a session **clears that session's grant list**. Re-enabling web-serving starts with an empty grant list — previously-issued capabilities are permanently invalid, and the user must run the Share flow again to produce fresh ones. This matches the intuitive "stop sharing = stop access" mental model.
- **D-16:** A **"Regenerate signing key"** button lives in Settings → Security. Clicking it generates a fresh 32-byte random key, overwrites `capability.key`, and every previously-issued capability fails signature verification globally across all sessions. The button is guarded by a confirmation dialog that explains the blast radius ("This invalidates all shared links across all sessions").

### Dashboard & Enumeration Model
- **D-17:** `/dashboard` becomes a **landing page**, not a session list. Content: AgentHub header, a "Join a Shared Session" prompt with the join-code form (reusing the `/join` flow from D-09), a QR-scan hint for mobile. The page does NOT list any sessions. The owner's management view stays in the native GUI/TUI/CLI, which talk to the daemon over Unix socket and are unaffected by SEC-02.
- **D-18:** `GET /api/sessions` requires a **session-scoped capability** and returns ONLY the session that capability is bound to (a single-item list). There is no listing-scoped capability. This collapses the endpoint from "enumeration" to "self-describe": no caller can ever receive a list longer than 1 via HTTPS.
- **D-19:** `GET /api/sessions/{id}/info` also requires a capability for that session (same validator as `/sessions/{id}/ws`). Used by the terminal HTML page to populate the status bar.
- **D-20:** In **local-network-fallback mode**, HTTP Basic Auth **and** capability are both required. Basic Auth gates reach to the server at all; capability gates session access within it. Defense in depth — the Basic Auth password is never a master key for session content.
- **D-21:** `internal/relay/server.go` (the localhost-only relay) is **unchanged**. It is not exposed on tailnet or LAN and was not flagged by the security review. If we later expose it, capability checks would be added at that time.

### Scope & Constraints
- **D-22:** `internal/webserver/` is the only server package modified. The daemon API surface (Unix socket IPC used by GUI/CLI/TUI) gets new methods for capability issuance (`IssueCapabilities(sessionID) → (readCap, writeCap, err)`, `RegenerateSigningKey()`, etc.) but no transport-level auth changes — the Unix socket boundary is already the trust boundary for those clients.
- **D-23:** The existing `?readonly=1` query parameter is **removed** from the write path. It may survive as a pure view hint (e.g., to tell the terminal HTML page to hide the input caret) but cannot grant or deny write access. Write access is determined exclusively by the capability's `perms` claim (SEC-04).
- **D-24:** `MsgInput` frames are rejected at the relay when the subscriber's capability lacks write permission. The existing `sub.ReadOnly` field at `internal/relay/hub.go:17` is repurposed: its value now comes from the capability's `perms` claim, not the `?readonly` query string. A regression test asserts that a reconnect without `?readonly` against a read-only capability still rejects `MsgInput` (SEC-05).

### Claude's Discretion
- Exact shape of the `/join` page HTML (copy, layout, error states for expired/wrong/used codes).
- Go package layout for the new capability module (likely `internal/capability/` or a sub-package of `internal/webserver/`).
- Middleware pattern for applying capability checks to multiple routes (single `requireCapability` wrapper vs route-level checks).
- Test organization: unit tests for token verify/sign, integration tests for the HTTP flow, fuzz tests for signature forgery resistance.
- Naming conventions for the new daemon API methods.
- Error response bodies (JSON shape, status codes) for capability failures — follow existing webserver error patterns.
- Metrics/logging hooks for successful/failed capability checks (keep minimal; don't add observability infra in this phase).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Threat Model
- `.planning/REQUIREMENTS.md` — SEC-01 through SEC-05 acceptance criteria
- `.planning/ROADMAP.md` §Phase 87 — Success criteria (5 must-be-TRUE conditions)
- `security-review/SECURITY_REVIEW.md` — Findings 1 & 2 (tailnet trust + read-only bypass), exploit scenarios, recommended-fix language, implementation order
- `security-review/SECURITY_DYNAMIC_VALIDATION.md` — Dynamic confirmation of findings against current code
- `security-review/internal_webserver_server_test.go` — Review-supplied test scaffolding for capability-based authz (reference only — tests should be added under `internal/webserver/`)
- `security-review/internal_relay_protocol_fuzz_test.go` — Review-supplied fuzz scaffolding (reference only — fuzz tests should be added under `internal/relay/`)

### Existing Code That Must Change
- `internal/webserver/server.go:46-48` — Comment saying "no application-layer authentication is required" — remove/update
- `internal/webserver/server.go:265-287` — Open `/api/sessions`, `/sessions/{id}`, `/sessions/{id}/ws` route wiring — add capability gates
- `internal/webserver/server.go:304-320` — `handleListSessions` — restrict to single-session response
- `internal/webserver/server.go:382-481` — `handleWSSRelay` — read capability, drop `OriginPatterns: ["*"]` (Origin handling moves to Phase 88 but the `*` pattern should be removed here in preparation or coordinated with Phase 88)
- `internal/webserver/server.go:385-391` — `?readonly` query param handling — replace with capability `perms` claim
- `internal/webserver/server.go:293-302` — `handleDashboard` — serve the new landing page HTML instead of the session-list dashboard
- `internal/daemon/api.go:257-295` — `handleCreateSession` — remove auto-`EnableSession(id)` at line 292 (SEC-01)
- `internal/daemon/api.go:515-525` — `handleToggleWebServing` — issue capabilities on enable, clear grant list on disable (D-15)
- `internal/daemon/engine.go:66-133` — `daemonSettings`, `loadSettingsFromDisk`, `saveSettingsToDisk` — do NOT add signing key here; use separate `capability.key` file (D-04)
- `internal/relay/hub.go:16-17` — `Subscriber.ReadOnly` — re-source from capability `perms` claim (D-24)
- `internal/relay/server.go` — Reviewed and confirmed out of scope for this phase (D-21)
- `web/dashboard.html` — Replaced / repurposed as the landing page (D-17)

### Settings Persistence Pattern
- `internal/daemon/engine.go:66-133` — Reference for how to add persisted state; the signing key does NOT follow this pattern (D-04)

### Frontend Touch-Points
- `frontend/src/App.tsx:15,498,547,561` — `ToggleWebServing` binding; where the grant gesture is plumbed
- `frontend/src/components/SettingsTab.tsx` — Existing web-server URL UX; gets a new "Security" section with the "Regenerate signing key" button (D-16)

### Phase Coordination
- **Phase 88 (WebSocket Handshake Security)** depends on Phase 87. Phase 88 adds the Origin allowlist check that runs BEFORE the capability check. Both must pass. The `OriginPatterns: ["*"]` at `internal/webserver/server.go:403` is removed during Phase 88; Phase 87 should not leave it more permissive than necessary but should not fully implement Origin allowlisting — that's Phase 88's scope.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/daemon/engine.go:66-133` — Existing `daemonSettings` pattern shows the persistence idiom (atomic write, mode 0600, missing-file-is-not-error). The new `FileKeyStore` follows the same shape with its own file.
- `internal/daemon/client.go:204` — `ToggleWebServing` already exists end-to-end (daemon → Wails binding → frontend). Phase 87 extends its server-side behavior but doesn't need to wire new transport.
- `internal/webserver/server.go:82-115` — `EnableSession` / `DisableSession` / `webEnabledSessions` — the per-session grant list (D-14) plugs into this same structure.
- `internal/relay/hub.go` — `Subscriber` struct already has `ReadOnly bool` (D-24 repurposes its source).
- `crypto/hmac` + `crypto/sha256` + `crypto/rand` + `encoding/base64` (base64url via `RawURLEncoding`) — all stdlib. No new Go deps required.
- `skip2/go-qrcode` — Already in use for QR encoding at `internal/webserver/server.go:371`. QR contents change (D-09) but the encoder stays.

### Established Patterns
- HTTP routing uses Go 1.22 `http.ServeMux` with path values (`r.PathValue("id")`). Capability middleware can wrap handlers the same way as `basicAuthMiddleware` at `internal/webserver/server.go:201`.
- Error responses use `http.Error(w, msg, code)` with short messages. Matches the existing style; no JSON error envelope.
- Settings-adjacent writes use mode 0600 and atomic-ish writes (`os.WriteFile` over existing file). Follow this for `capability.key`.
- Unix-socket daemon IPC uses JSON request/response with typed structs in `internal/daemon/types.go`. New methods (`IssueCapabilities`, `RegenerateSigningKey`, `ExchangeJoinCode`) follow this convention.

### Integration Points
- **Session lifecycle:** `CreateSession` at `internal/daemon/engine.go:160` → new behavior: no auto-enable. `ToggleWebServing` → issue caps on enable, clear grants on disable.
- **Session exit:** `onExit` callback at `internal/daemon/api.go:268-277` already runs after PTY EOF; no changes needed since capabilities are bound to the session ID, which goes away when the session is removed from the registry.
- **Frontend share UI:** SettingsTab currently shows a URL copy/open/QR block for the dashboard. After Phase 87, per-session share controls need a similar block inside the Daemon Manager Panel or each session's context menu, showing two URLs (read / read+write) and two QRs (one per capability).
- **Landing page:** new HTML file (`web/join.html` or replace `web/dashboard.html`). Served by the repurposed `handleDashboard`, with a sibling `handleJoin` that handles the POST code exchange.

</code_context>

<specifics>
## Specific Ideas

- User wants a **user-facing choice** of key storage (file vs OS keychain) as a future enhancement, not v3.1. Ship Phase 87 with file-only, use the `KeyStore` interface to make the addition clean. Create a GitHub issue for the keychain backend + settings toggle + cross-platform migration so this doesn't get lost.
- The join-code flow intentionally trades a small amount of convenience for meaningful security gains: **photograph/screenshot of a QR is worthless after the Join tap or 5-minute TTL**, and the explicit "Join" tap gives recipients a deliberate intent step.
- Read-only capability is a first-class concept, not a UI hint. The GUI's session share panel shows "Read-Only Link" and "Full Access Link" as two equal-weight actions; the user decides which to share.
- The "Regenerate signing key" button is the v3.1 panic button. Users who suspect a leak have a single in-product action that makes the blast radius obvious ("all shared links across all sessions become invalid").

</specifics>

<deferred>
## Deferred Ideas

### OS Keychain-backed key storage
User wants this as a future modification. Must be tracked as a GitHub issue.
- Platform-specific keychain backends (macOS Keychain, libsecret, Windows CredMan/DPAPI)
- Handle service-mode daemon limitations (system-domain launchd cannot access user login keychain; systemd system units lack D-Bus session; headless Linux)
- User-facing Settings toggle: `file | keychain | auto-detect`
- Migration path when switching stores (copy key → verify → delete from old store)
- Behavior when user-selected store is unavailable (fall back + banner vs. refuse to issue)

### Revocation & Audit UI (v3.2+ per REQUIREMENTS.md Future Requirements)
- Per-grant revocation (list outstanding `grant_id`s, revoke individual)
- Rotate individual session's capabilities without rotating daemon-wide signing key
- Audit log of grants issued, exchanges, revocations
- Rate limiting on the `/join` endpoint (hardening once capability model is stable)

### Per-user identity
Capabilities in v3.1 are bearer tokens — anyone with the cap can use it. Per-user identity (SSO/OIDC) is deferred until multi-user demand emerges, per REQUIREMENTS.md Out of Scope.

### Scope creep noted and deferred
- Pause/suspend semantics for sessions (not currently a feature; out of Phase 87 scope)
- Removing `internal/relay/server.go` as tech-debt cleanup (separate effort)
- External KMS for signing key — explicitly out of scope per REQUIREMENTS.md

</deferred>

---

*Phase: 87-capability-based-session-authorization*
*Context gathered: 2026-04-19*
