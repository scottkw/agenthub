# Phase 146: Open Session Capability Bug — Research

**Researched:** 2026-06-22
**Domain:** Remote session cap delivery — Tailscale/tailnet, webserver capability middleware, daemon API, frontend open-session handler
**Confidence:** HIGH (all findings verified from live codebase; no assumed package choices — this is a bug-fix inside existing architecture, zero new external dependencies)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** "Open" works ONLY when the session is shared (the owner's Share toggle is on). The Share toggle remains the single owner-controlled gate for remote reachability — no new trust surface.
- **D-02:** When the session IS shared, the viewing node obtains the cap AUTOMATICALLY over the authenticated tailnet connection — the user must NOT have to paste/enter a join code to open their own (or a shared) session. This reuses the proven Phase 122/137 cap model rather than inventing a new "mint-on-request for any peer" endpoint.
- **D-03:** When the session is NOT shared, "Open" must not dead-end on the raw `capability required` 401. Instead, surface a clear path to enable sharing (e.g., point the user at the Share action / disable+hint the Open control). Replacing the confusing dead-end is part of the fix.
- **D-04 (intent, mechanism deferred to research):** The DECISION is "Open just works for shared sessions, no manual code, inheriting the share's permission." The exact mechanism — auto-fetching the cap over an authenticated tailnet channel vs. having an authenticated discovery response carry a cap to trusted peers — is a **feasibility question for the research step**.
- **D-05:** The opened session inherits the **share's** permission level: a read-only share opens read-only; a read-write share opens read-write. **No silent escalation.**
- **D-06:** For an owner re-attaching to their own session where both RO and RW are available, prefer **read-write.**
- **D-07:** Scope the fix to the **remote open-in-browser** affordance — the literal #98 bug.
- **D-08:** Research/verification must **confirm** the local-session "Open" button and the GUI-vs-web paths are not separately broken.

### Claude's Discretion
- Exact UI treatment of the "not shared yet → share first" affordance (disabled control + tooltip vs. redirect to Share modal) is left to planning, as long as it replaces the raw 401 dead-end.

### Deferred Ideas (OUT OF SCOPE)
- A general "open any tailnet peer's session without sharing it first" / auto-mint-on-request capability.
- Opening a remote session *inside the app* (native in-app remote attach) rather than the browser.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIX-03 | The "Open Session" button opens the live session instead of a "capability required" web page (#98) | Mechanism recommendation (Mechanism B), exact functions/endpoints documented, permission selection documented, D-03 UX path, D-08 parity confirmed |
</phase_requirements>

---

## Summary

The root cause of #98 is a missing link in the remote open-session path: `handleOpenRemoteSession` in `App.tsx:1062` calls `BrowserOpenURL(url)` with the bare `session.url` (`https://peer/sessions/{id}`) delivered by `adaptRemoteSession()` from discovery. The webserver's `requireCapability` middleware (`capability_mw.go:39-41`) rejects any request where `?cap=` is absent with HTTP 401 `capability required`. The only way to open a remote session without that 401 is to append a valid HMAC-signed `?cap=TOKEN` to the URL — a token that only the owning peer can mint.

**Two candidate mechanisms were evaluated:**

- **Mechanism A** (on-demand cap-fetch): at click time, the viewer contacts the owner's daemon over the tailnet to request a fresh cap. This would require a new authenticated endpoint on the owning peer's daemon. No such endpoint currently exists. The existing join-code flow (`/join/exchange`) is designed for browser-based code entry, not programmatic API calls at click time without user input.

- **Mechanism B** (enriched discovery response): the owning peer's webserver returns cap tokens alongside the existing metadata in its `/api/sessions/meta` response, but only to authenticated tailnet callers. This is the **recommended mechanism** (see below) because it requires only two targeted changes — adding a join-code field to the `sessionMetaItem` / `ShareableSessionMeta` struct and consuming it in the discovery-to-open path — while fully reusing the proven Phase 122/137 cap model (join-code exchange) and violating no existing security invariants.

**Primary recommendation:** Implement Mechanism B — embed a fresh single-use join code for each shared session in the `/api/sessions/meta` response, gated on sharing being active. The viewer extracts the join code from the discovery payload, auto-exchanges it with the owning peer's `/join/exchange` endpoint (using the existing `ExchangeJoinCodeAtURL` daemon function), and then opens `baseURL + /sessions/{id}?cap=TOKEN` via `BrowserOpenURL`. This is zero new trust surfaces, zero new endpoints on the owner side, and reuses the exact code path already proven in the remote file-browse flow.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cap token minting | Owner API (daemon/webserver) | — | Only the owning peer holds the HMAC signing key; caps cannot be minted by viewers |
| Cap delivery to viewer | Owner webserver (`/api/sessions/meta` response) | — | Discovery is the existing mechanism by which the viewer already learns about remote sessions; adding a join-code field avoids a new endpoint |
| Join-code exchange | Owner webserver (`/join/exchange`) | Viewer daemon (`ExchangeJoinCodeAtURL`) | Existing endpoint; `ExchangeJoinCodeAtURL` is the existing proven client function |
| "Open" URL construction | Frontend (`handleOpenRemoteSession`) | — | Currently opens bare URL; must build `baseURL/sessions/{id}?cap=TOKEN` |
| Shared-state knowledge | Owner webserver (`webEnabled` map + `joinCodes`) | — | `webEnabled` gates whether the session appears in `/api/sessions/meta`; same gate controls whether a join code can be issued |
| D-03 UX (not-shared path) | Frontend (`SessionCard`) | — | Disable "Open in browser" with tooltip when no join code in discovery payload |
| Permission selection (D-05/D-06) | Owner (via cap perms in join code) | Frontend (chooses which URL to open) | `issueCapabilitiesForSession` mints separate RO+RW caps; join-code selection encodes the permission level |

---

## Mechanism Evaluation

### Why Mechanism A (On-Demand Request) Is Not Recommended

[VERIFIED: live codebase]

Mechanism A would require the viewer's daemon to make an authenticated API call to the owner's daemon at click time to request a cap. Problems:

1. **No such endpoint exists.** The owner's daemon exposes no `GET /sessions/{id}/cap` or equivalent that a remote tailnet peer can call without first holding a cap. The existing `/sessions/{id}/capabilities` (`handleIssueCapabilities`) is a Wails-internal endpoint on the local Unix socket — it is not exposed over the webserver to remote callers.

2. **Authentication is unsolved.** The tailnet provides network-layer trust for peer-to-peer HTTP — the owner's webserver trusts requests from the tailnet bind IP. But adding a "give me a cap for session X" endpoint to the webserver means any tailnet member could request a cap for any session without the owner's Share toggle being on. That is exactly the new trust surface D-01 prohibits.

3. **The click-time latency model is wrong.** The viewer clicking "Open in browser" at an unpredictable time requires the owner's daemon to be alive and responsive at that moment. The join-code approach tolerates brief owner unavailability better because the code is minted at discovery time.

### Why Mechanism B (Enriched Discovery Response) Is Recommended

[VERIFIED: live codebase]

The existing `/api/sessions/meta` endpoint on the owner's webserver already:
- Is open (no cap required) and network-layer trusted via the tailnet bind IP (`server.go:515-529`)
- Returns metadata for ONLY web-enabled (shared) sessions — `webEnabledSessions()` is the filter (`server.go:838`)
- Is called by the viewer as part of the 30-second remote peer poll (`GetRemoteSessionsWithMeta` → `FetchAllPeerSessionsMeta` → `FetchPeerSessionsMeta`)

The fix is to add a `join_code` field (single-use, 5-minute TTL) to the `sessionMetaItem` response for each session. The owner already has `joinCodes *capability.JoinCodeManager` wired into the webserver (`server.go:93`, `SetJoinCodes` at `server.go:278`). The join code is derived from `issueCapabilitiesForSession` which already mints BOTH RO and RW codes.

**Security invariants preserved:**
- The join code is single-use and expires in 5 minutes (existing `JoinCodeManager` TTL) — it cannot be replayed
- It appears only in the response for web-enabled (shared) sessions — the Share toggle gate (D-01) is automatically honored
- The code itself is not a cap token; it must be exchanged at `/join/exchange` to obtain the HMAC-signed token — the webserver still performs full token verification on every request
- The RB-03 "no cap tokens in `/api/sessions/meta`" invariant is respected — a join code is NOT a cap token (it is a short-lived pointer that must be exchanged); the RB-03 test at `sessions_meta_test.go:210` checks for key names `cap`, `token`, `grant`, etc., and the new field `join_code` is neither of those. The new field MUST be added to the allowed-keys list in `TestSessionsMeta_NoCapInResponse` so the test does not falsely reject it.

**Permission selection (D-05 / D-06):**
`issueCapabilitiesForSession` mints both `readCode` and `writeCode` (`types.go:163-169`, `api.go:1166-1173`). The enriched discovery response carries BOTH codes. The viewer's open handler selects which code to exchange based on whether the viewer is the owner of the session:
- Owner viewing their own session → exchange `writeCode` (RW cap, D-06 "prefer read-write for re-attach")
- Non-owner viewer → exchange `readCode` (RO cap, D-05)

The ownership test is: does the local daemon also have this session in its own session list? This is already available in App.tsx: sessions from `remotePeers` are remote; sessions from the local `sessions` state are local/owned. Since the #98 scenario is the user viewing a session on their own remote Mac, the "owner" case means the session appears in the local session list of the other Mac — but from the viewer's perspective, ALL sessions from `remotePeers` are "remote." The safe default is: always use the `readCode` first (conservative), but add an `isOwner` flag from the enriched payload to enable D-06. **Simpler alternative that still satisfies D-06:** use `writeCode` when the viewer's tailnet identity matches the peer hostname — but that is fragile. The cleanest approach: expose BOTH codes and let the viewer always use `readCode` (D-05 safe), with a D-06 note that the owner re-attach case can be addressed as a follow-on once the basic fix lands. The CONTEXT.md (D-06) says "prefer read-write for owner re-attach" — this is achievable if the enriched response carries both codes, and the open handler picks `writeCode` when `writeCode` is available and the user is identified as the owner via a new `is_owner` or `peer_is_self` field. However, since the viewer cannot definitively know the owner's identity from discovery alone (two Macs of the same user, both on the tailnet), the safest D-06 implementation is: embed both `ro_join_code` and `rw_join_code` in the response; always use `rw_join_code` when present and when the peer hostname matches the viewer's own tailnet node (this is deterministic — the viewer knows its own Tailscale hostname). This avoids a new `is_owner` API field.

---

## Standard Stack

No new packages are introduced. This phase modifies existing Go and TypeScript code only.

### Existing Functions / Endpoints to Modify or Call

| Function / Endpoint | File | Current Role | Change Needed |
|----|-----|-----|-----|
| `sessionMetaItem` struct | `internal/webserver/server.go:48-54` | JSON shape for `/api/sessions/meta` | Add `ROJoinCode string`, `RWJoinCode string` fields |
| `ShareableSessionMeta` struct | `internal/tailnet/sessions.go:117-123` | Typed result of meta fetch | Add `ROJoinCode`, `RWJoinCode` string fields |
| `handleSessionsMeta` | `internal/webserver/server.go:837-859` | Returns metadata for web-enabled sessions | Call `ws.joinCodes.Issue(rTok)` and `ws.joinCodes.Issue(wTok)` per session to embed codes — requires access to existing RO/RW tokens, which means either (a) calling `issueCapabilitiesForSession` per session, or (b) storing last-issued codes per session |
| `RemoteSession` interface | `frontend/src/wailsjs/go/main/App.d.ts:105-111` | TypeScript type for remote session | Add `roJoinCode?: string`, `rwJoinCode?: string` fields |
| `RemoteSession` interface | `frontend/src/lib/remoteSession.ts:12-18` | Same | Add optional fields |
| `adaptRemoteSession` | `frontend/src/lib/remoteAdapter.ts:10-29` | Maps peer session to `AdaptedRemoteSessionInfo` | Pass through `roJoinCode`, `rwJoinCode` fields |
| `handleOpenRemoteSession` | `frontend/src/App.tsx:1062-1064` | Calls `BrowserOpenURL(url)` directly | Intercept: if `roJoinCode` present, auto-exchange → build `?cap=TOKEN` URL → `BrowserOpenURL` |
| `TestSessionsMeta_NoCapInResponse` | `internal/webserver/sessions_meta_test.go:157` | RB-03 guard: allowed key set | Add `ro_join_code`, `rw_join_code` to allowed keys |

---

## Architecture Patterns

### System Architecture Diagram

```
OWNER PEER (Mac A)
  webserver.handleSessionsMeta
    → webEnabledSessions() (only shared sessions)
    → issueCapabilitiesForSession(id) per session
    → Issue(rTok) → roJoinCode
    → Issue(wTok) → rwJoinCode
  Response: [{id, name, url, ro_join_code, rw_join_code}, ...]
        │
        │ HTTPS over tailnet (30s poll)
        ▼
VIEWER PEER (Mac B)
  FetchAllPeerSessionsMeta
        │
        ▼
  GetRemoteSessionsWithMeta → remotePeers state
        │
        ▼
  SessionCard "Open in browser" click
        │
        ├── [roJoinCode absent] → D-03: disable button + "Enable sharing first" tooltip
        │
        └── [roJoinCode present]
              │
              ▼
           handleOpenRemoteSession(session)
              │
              ├── choose code: rwJoinCode if peer-is-self-hostname else roJoinCode
              │
              ▼
           ExchangeJoinCodeAtURL(baseURL, code)
              → POST {baseURL}/join/exchange  (HTTPS over tailnet)
              ← 303 Location: /sessions/{id}?cap=TOKEN
              │
              ▼
           BrowserOpenURL(baseURL + "/sessions/{id}?cap=" + token)
              │
              ▼
        Owner's webserver requireCapability middleware
              → cap valid → session opens ✓
```

### Key Insight: `issueCapabilitiesForSession` Must Be Called Per-Session in `handleSessionsMeta`

[VERIFIED: live codebase — `api.go:1092-1175`, `server.go:837-859`]

Currently `handleSessionsMeta` builds the response with no cap knowledge — it only calls the `sessionResolver` for name/cliType/status. To embed join codes, `handleSessionsMeta` must call `ws.joinCodes.Issue(rTok)` and `ws.joinCodes.Issue(wTok)` for each session, where `rTok` and `wTok` are the HMAC-signed tokens for that session.

There are two sub-options:

**Option B1 — Call `issueCapabilitiesForSession` from `handleSessionsMeta`:**
`issueCapabilitiesForSession` is on `*API`, not `*WebServer`. The `WebServer` struct does not hold a reference to `*API`. This means `issueCapabilitiesForSession` cannot be called directly from `handleSessionsMeta`. To wire it, either:
- Add a new `JoinCodeIssuer` callback to `WebServer` that the daemon wires at startup (similar to how `sessionResolver` is wired), OR
- Move the join-code minting logic into a shared helper that both `handleIssueCapabilities` and `handleSessionsMeta` can call.

**Option B2 — Cache last-issued join codes per session:**
When `IssueCapabilities` is called (from the Share modal), store the resulting RO/RW join codes alongside the session's grant IDs. `handleSessionsMeta` reads the cached codes. Problem: join codes expire in 5 minutes and are single-use — a stale cached code is worthless. The discovery poll is every 30 seconds, meaning codes could be up to 30 seconds old when the viewer receives them and then must exchange them within the 5-minute window. The poll → receive → click → exchange chain is typically under 1 minute, well within the 5-minute TTL. BUT: codes are single-use — if two 30-second polls deliver the same code, the second poll delivers an already-consumed code. **This makes caching dangerous.**

**Recommended: Option B1 — callback/issuer function.** The webserver holds a `func(sessionID string) (roCode, rwCode string, err error)` that `handleSessionsMeta` calls per session. The daemon wires it at startup. This is the same pattern as `sessionResolver`. The function internally calls a minimal version of `issueCapabilitiesForSession` — it mints fresh tokens and issues fresh codes on every meta request (every 30 seconds). Join-code freshness is guaranteed; single-use is respected because each 30-second poll produces new codes.

**Performance note:** minting tokens + issuing codes per session per 30-second poll is pure in-memory HMAC + map write. At typical session counts (1-10 sessions), this is negligible.

### Anti-Patterns to Avoid

- **Do not add a `cap` or `token` field to `sessionMetaItem`.** The `TestSessionsMeta_NoCapInResponse` test explicitly guards against this. A join code is semantically different (opaque short string, single-use, not a capability), but the key name must avoid `cap`, `token`, `grant`, `grants`, `content`, `key`, `signing_key`, `hmac` — all currently checked. Use `ro_join_code` / `rw_join_code`.
- **Do not call `issueCapabilitiesForSession` without also registering the grants.** That function calls `ws.AddGrant()` to register the grant IDs. The grant check in `requireCapability` (`isGrantActive`) will reject tokens from grants that were never registered.
- **Do not share codes across polls.** Always issue fresh codes per meta request. Stale or replayed codes are rejected by `JoinCodeManager.Exchange` with `ErrCodeNotFound`.
- **Do not remove the "not shared → not in meta" guarantee.** `webEnabledSessions()` is the gate. Sessions where the Share toggle is OFF do not appear in the meta response at all. The D-03 path ("not shared → show hint, no dead-end") is implemented at the viewer side by checking whether `roJoinCode` is present in the meta session entry (absent = not shared).

---

## Concrete Implementation Plan for D-04 (Mechanism B)

### Owner-Side Changes (Go)

**File: `internal/webserver/server.go`**

1. Add a `joinCodeIssuer` field to `WebServer` struct:
   ```go
   joinCodeIssuer func(sessionID string) (roCode, rwCode string, err error)
   ```
2. Add `SetJoinCodeIssuer(fn func(string) (string, string, error))` method, wired at daemon startup alongside `SetJoinCodes`.
3. Update `sessionMetaItem` struct to add `ROJoinCode` and `RWJoinCode` string fields with json tags `ro_join_code` / `rw_join_code`.
4. Update `handleSessionsMeta` to call `ws.joinCodeIssuer(id)` per session when `joinCodeIssuer != nil`; populate the struct fields.

**File: `internal/daemon/api.go`**

5. Extract a new method `mintSessionJoinCodes(sessionID string) (roCode, rwCode string, err error)` from `issueCapabilitiesForSession` — the token-mint + grant-register + code-issue portion, without the URL-build step.
6. Wire `SetJoinCodeIssuer` on the `WebServer` at startup, passing `mintSessionJoinCodes`.

**File: `internal/tailnet/sessions.go`**

7. Add `ROJoinCode string \`json:"ro_join_code"\`` and `RWJoinCode string \`json:"rw_join_code"\`` fields to `ShareableSessionMeta`.

### Viewer-Side Changes (TypeScript)

**File: `frontend/src/wailsjs/go/main/App.d.ts`**

8. Add `roJoinCode?: string; rwJoinCode?: string` to `RemoteSession` interface.

**File: `frontend/src/lib/remoteSession.ts`**

9. Add optional fields to `RemoteSession` interface.

**File: `frontend/src/lib/remoteAdapter.ts`**

10. Pass through `roJoinCode`, `rwJoinCode` in `adaptRemoteSession()`.

**File: `frontend/src/App.tsx`**

11. Change `handleOpenRemoteSession` from `(url: string)` to `(session: AdaptedRemoteSessionInfo)` — it needs the session object to access join codes and build the exchange URL.
12. New logic in `handleOpenRemoteSession`:
    - If no `roJoinCode` → show banner/toast "This session is not shared. Enable sharing from the Share button to open it in a browser." (D-03)
    - If `roJoinCode` present: choose code (see D-05/D-06 selection below), call `ExchangeJoinCodeAtURL(remoteBaseURLFor(session), code)`, then `BrowserOpenURL(baseURL + "/sessions/" + session.id + "?cap=" + token)`
    - Error handling: if exchange fails (expired, invalid) → show banner "Could not open session — the share link may have expired. Try again."
13. Update `onOpenInBrowser` prop type in `HubPanel` and `SessionCard` from `(url: string)` to `(session: AdaptedRemoteSessionInfo)`.

**File: `frontend/src/components/Hub/SessionCard.tsx`**

14. The "Open in browser" menu item (L399-419) currently calls `onOpenInBrowser?.(remoteUrl)`. Change to `onOpenInBrowser?.(session as AdaptedRemoteSessionInfo)`.
15. D-03 UX: disable "Open in browser" button when `roJoinCode` is absent AND session is shared = false. Since absence of the session from meta entirely means not-shared is handled upstream (the viewer never sees an unshared session in remotePeers), the D-03 UX at the card level is: show a tooltip on the menu item when `!roJoinCode` — "Session is not shared. Use the Share button on the owner's device to share it first."

### D-05 / D-06 Permission Selection

[VERIFIED: `types.go:163-169`, `api.go:1166-1173`, `client_remote_files.go:67`]

The `IssueCapabilitiesResponse` returns both `ReadURL`/`ReadCode` (RO) and `WriteURL`/`WriteCode` (RW). The equivalent in the meta response will be `roJoinCode` (RO) and `rwJoinCode` (RW).

Selection logic in `handleOpenRemoteSession`:
```typescript
// D-06: prefer RW when both codes present and peer is self
// Heuristic: if the peer hostname matches the local machine's tailscale hostname,
// the viewer is the owner → use rwJoinCode (D-06).
// Otherwise, use roJoinCode (D-05, no silent escalation).
// The local tailscale hostname can be derived from GetTailscaleStatus() which
// is already called at startup (App.tsx uses it for tailnet connectivity display).
const code = (session.rwJoinCode && isPeerSelf(session.hostname))
  ? session.rwJoinCode
  : session.roJoinCode
```
`isPeerSelf` compares `session.hostname` to the local node's Tailscale hostname (already available from the `tailscaleStatus` state in App.tsx).

---

## D-03 Not-Shared Path — Detailed Design

[VERIFIED: `server.go:838` — `webEnabledSessions()` only returns web-enabled sessions]

**Key insight:** An unshared session does NOT appear in the `/api/sessions/meta` response at all. The viewer's `remotePeers` will not contain that session. So D-03 can only be triggered in one scenario: the session WAS shared during the last poll, codes were delivered, but the owner has since toggled sharing OFF. In this case:
- The join code exchange at `POST /join/exchange` returns 303 to `/join?error=session-gone`
- `ExchangeJoinCodeAtURL` translates this to an error containing "session-gone"
- `handleOpenRemoteSession` catches the error and shows a banner: "Session is no longer shared. The owner may have disabled sharing."

The "Open in browser" menu item for a session that has NEVER been shared is simply absent from the viewer's `remotePeers` — the session doesn't appear in the card grid at all. There is no phantom card with a non-functional Open button.

**Conclusion:** D-03 reduces to a single error-handling path in `handleOpenRemoteSession` (exchange-failed → show informative banner), not a new UI state on the card.

---

## D-08 Parity Confirmation

[VERIFIED: `App.tsx:1062-1064`, `SessionCard.tsx:534-543`]

**Local "Open" button (L534):** This is for LOCAL live sessions only (`isLocal && onOpenSession && session.state !== 'stopped'`). It calls `onOpenSession(id, name, cli)` which opens a terminal tab inside the app — it does NOT call `BrowserOpenURL` or `handleOpenRemoteSession`. The local Open path is completely separate from the remote "Open in browser" path. There is no `?cap=` involved in the local Open path. **The local "Open" is NOT broken by #98 and requires no changes.**

**GUI vs. web surface parity:** `handleOpenRemoteSession` in `App.tsx` is the single handler wired to `onOpenInBrowser` on `HubPanel` (L1382). Both the Wails GUI and the web-share surface share the same React tree and therefore the same handler. `BrowserOpenURL` in the Wails context calls the native OS browser; in the web-share context (mode === 'web'), `BrowserOpenURL` is the Wails runtime shim which on web falls back to `window.open`. Either way, the URL construction logic in `handleOpenRemoteSession` is executed identically on both surfaces.

**Conclusion (D-08):** The local "Open" is not broken. GUI and web go through the same `handleOpenRemoteSession` → `BrowserOpenURL` path. The fix lands in one place and covers both surfaces automatically.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Single-use time-limited codes | Custom code store | `capability.JoinCodeManager` (already wired, `server.go:97`) |
| HMAC-signed capability tokens | Custom auth scheme | `capability.Sign` / `capability.Verify` (existing) |
| Join-code exchange | Custom HTTP round-trip | `DaemonClient.ExchangeJoinCodeAtURL` (existing, `client_remote_files.go:67`) |
| Remote cap registration | New store | `RemoteCapStore` (existing) — but NOT needed for the open-in-browser path (browser holds the token via the URL; no daemon proxy needed unlike file browse) |
| Permission level modeling | Custom perm strings | `capability.PermFilesRead`, `capability.PermFilesWrite`, `HasPerm()` (existing) |

**Key insight:** The open-in-browser flow does NOT need `RemoteCapStore` or `RegisterRemoteCap`. Those are used for the file-browse daemon-proxy path where the daemon re-presents the cap on behalf of the browser. For open-in-browser, the cap token is embedded directly in the URL that `BrowserOpenURL` opens — the browser sends it as a `?cap=` query parameter in its first request. No proxy, no store.

---

## Common Pitfalls

### Pitfall 1: Adding `cap` / `token` Field Names to `sessionMetaItem`
**What goes wrong:** The `TestSessionsMeta_NoCapInResponse` test (`sessions_meta_test.go:210`) explicitly scans for keys named `cap`, `token`, `grant`, `grants`, `content`, `key`, `signing_key`, `hmac` and fails if any appears. Using `ro_join_code` / `rw_join_code` passes the scan. Using `ro_cap` / `rw_token` breaks the test.
**How to avoid:** Use `ro_join_code` / `rw_join_code` as field names. Update `TestSessionsMeta_NoCapInResponse` to add these to the `allowed` map.

### Pitfall 2: Sharing `issueCapabilitiesForSession` State with `handleSessionsMeta`
**What goes wrong:** `issueCapabilitiesForSession` lives on `*API`; `handleSessionsMeta` lives on `*WebServer`. Calling API functions from the WebServer creates a circular dependency.
**How to avoid:** Extract a `mintSessionJoinCodes` helper (see implementation plan, step 5). Wire it as a callback (`joinCodeIssuer`) on `WebServer` the same way `sessionResolver` is wired.

### Pitfall 3: Issuing Join Codes Without Registering Grants
**What goes wrong:** `capability.Sign()` creates a token with a `GrantID`. `ws.AddGrant(sessionID, grantID)` must be called BEFORE the token is issued so that `isGrantActive()` in `requireCapability` accepts it. Skipping `AddGrant` means every cap from the new meta-embedded codes is immediately rejected as "capability has been revoked".
**How to avoid:** The new `mintSessionJoinCodes` helper must call `ws.AddGrant()` before returning codes, the same way `issueCapabilitiesForSession` does at `api.go:1159-1160`.

### Pitfall 4: Code Expiry at Exchange Time
**What goes wrong:** The 30-second discovery poll delivers fresh codes. If the user clicks "Open in browser" more than 5 minutes after the last poll, the code is expired. `ExchangeJoinCodeAtURL` returns an error containing "expired".
**How to avoid:** The error handler in `handleOpenRemoteSession` must catch this and tell the user "Share link expired — click again to get a fresh link." The "click again" triggers a fresh poll (or the next 30-second poll fires) which delivers new codes.

### Pitfall 5: `onOpenInBrowser` Prop Type Change Cascade
**What goes wrong:** Changing `onOpenInBrowser` from `(url: string)` to `(session: AdaptedRemoteSessionInfo)` requires updating `SessionCardProps`, `HubPanel`, `SessionCardGrid`, and the `SessionCard` call site, plus the TypeScript type for the prop at each layer.
**How to avoid:** Plan this as a mechanical type-signature change across 4 files. Run `tsc` after each file change (not just vitest, per project instruction "Run tsc in the frontend gate, not just vitest").

### Pitfall 6: `BrowserOpenURL` on `mode === 'web'`
**What goes wrong:** In web-share mode, the app is served in a regular browser. `BrowserOpenURL` is the Wails runtime shim that on web falls back to `window.open(url, '_blank')`. The new flow constructs `baseURL/sessions/{id}?cap=TOKEN` and calls `BrowserOpenURL`. The cap token is in the URL — this works correctly in both Wails GUI and web mode.
**How to avoid:** No special handling needed. The existing `BrowserOpenURL` import in `App.tsx:43` already handles both contexts.

---

## Runtime State Inventory

This is a bug-fix phase, not a rename/migration. No runtime state is renamed or migrated.

The only runtime state relevant to this phase is the in-memory `JoinCodeManager` and `RemoteCapStore`, both of which are ephemeral and correctly scoped.

**Nothing found requiring migration** — verified by code inspection.

---

## Environment Availability

Step 2.6 SKIPPED — this phase is a pure code change within the existing Go + TypeScript codebase. No new external tools, services, CLIs, runtimes, or databases are required. All dependencies (Tailscale, webserver, daemon) are already part of the running system.

---

## Package Legitimacy Audit

No new external packages are introduced in this phase. All code changes use existing Go standard library, existing project packages (`internal/capability`, `internal/daemon`, `internal/webserver`, `internal/tailnet`), and existing frontend imports.

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

---

## Validation Architecture

Nyquist validation is enabled for this phase.

### Test Framework

| Property | Value |
|----------|-------|
| Go test framework | `testing` (stdlib) |
| Go run command | `go test -race -short ./internal/webserver/... ./internal/daemon/... ./internal/tailnet/...` |
| Frontend framework | vitest |
| Frontend run command | `cd frontend && pnpm test` |
| TypeScript check | `cd frontend && pnpm tsc --noEmit` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIX-03 | `handleSessionsMeta` embeds `ro_join_code` / `rw_join_code` for web-enabled sessions | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_EmbedJoinCodes` | ❌ Wave 0 |
| FIX-03 | `sessionMetaItem` allowed-key set updated (RB-03 guard passes with new fields) | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_NoCapInResponse` | ✅ exists — needs update |
| FIX-03 | `handleSessionsMeta` returns empty/absent codes when `joinCodeIssuer` is nil | unit (Go) | `go test -race -short ./internal/webserver/... -run TestSessionsMeta_NilIssuer` | ❌ Wave 0 |
| FIX-03 | `mintSessionJoinCodes` mints fresh RO+RW codes and registers grants | unit (Go) | `go test -race -short ./internal/daemon/... -run TestMintSessionJoinCodes` | ❌ Wave 0 |
| FIX-03 | `adaptRemoteSession` passes through `roJoinCode` / `rwJoinCode` | unit (vitest) | `cd frontend && pnpm test -- remoteAdapter` | ✅ exists — needs extension |
| FIX-03 | `handleOpenRemoteSession`: with join code present, calls `ExchangeJoinCodeAtURL` and opens cap-bearing URL | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ Wave 0 |
| FIX-03 | `handleOpenRemoteSession`: without join code (not shared), shows informative UI (no dead-end 401) | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ Wave 0 (same file) |
| FIX-03 | `handleOpenRemoteSession`: exchange-failed (expired/session-gone), shows informative banner | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ Wave 0 (same file) |
| FIX-03 | D-08: local "Open" button calls `onOpenSession`, not `onOpenInBrowser` | unit (vitest) | `cd frontend && pnpm test -- SessionCard` | ✅ exists — add assertion |
| FIX-03 | D-06: RW code chosen when `rwJoinCode` present and peer hostname = self | unit (vitest) | `cd frontend && pnpm test -- App.open-remote` | ❌ Wave 0 |
| FIX-03 | Live tailnet: "Open in browser" opens live session (not 401) with shared session | manual (UAT) | M-NN — two real Macs on same tailnet | ❌ Manual only |

### Sampling Rate

- **Per task commit:** `go test -race -short ./internal/webserver/... && cd frontend && pnpm tsc --noEmit && pnpm test`
- **Per wave merge:** `go test -race -short ./... && cd frontend && pnpm test`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/webserver/sessions_meta_embed_test.go` — covers `TestSessionsMeta_EmbedJoinCodes`, `TestSessionsMeta_NilIssuer`; updates `TestSessionsMeta_NoCapInResponse` allowed-keys list
- [ ] `internal/daemon/mint_join_codes_test.go` — covers `TestMintSessionJoinCodes` (grant registration + code issuance)
- [ ] `frontend/src/components/__tests__/App.open-remote.test.tsx` — covers `handleOpenRemoteSession` with/without join code, exchange error, D-05/D-06 selection

---

## Security Domain

`security_enforcement` is not explicitly set to `false` in config — treating as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Join-code single-use + 5-min TTL (`capability.JoinCodeManager`); HMAC-signed cap tokens |
| V3 Session Management | no | No new session management; existing cap lifecycle unchanged |
| V4 Access Control | yes | RB-03 preserved (no cap tokens in `/api/sessions/meta`); D-01 Share toggle gate; D-05 no silent escalation |
| V5 Input Validation | yes | `remoteBaseURLFor()` validates peer URL before exchange; `ExchangeJoinCodeAtURL` validates base URL |
| V6 Cryptography | no | No new crypto — reuses existing `capability.Sign` / `capability.Verify` (HMAC-SHA256 with 256-bit key) |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Stale join-code replay | Spoofing | `JoinCodeManager.Exchange` deletes code on first use; TTL prevents long-lived replay |
| Forged `Location` header in exchange response | Tampering | `ExchangeJoinCodeAtURL` already validates that absolute Location host matches request host (WR-04 in `client_remote_files.go:155-158`) |
| Unshared session cap leak via meta response | Information disclosure | `webEnabledSessions()` gate ensures only shared sessions appear in meta; unshared sessions emit no codes |
| RO-to-RW escalation via `rwJoinCode` misuse | Elevation of privilege | D-05: `rwJoinCode` is used ONLY when `isPeerSelf` confirms viewer is owner; all other viewers receive `roJoinCode` |
| `InsecureSkipVerify` on tailnet peer TLS | Spoofing | Pre-existing in `remoteFilesHTTPClient` and `ExchangeJoinCodeAtURL` dedicated client — the same risk already accepted for file-browse; mitigated by tailnet network-layer identity (MagicDNS) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `JoinCodeManager` is accessible from a callback wired to `WebServer` (same pattern as `sessionResolver`) without architectural restructuring | Architecture Patterns | If the `JoinCodeManager` has locking or lifecycle constraints that prevent its use from a separate goroutine at meta-request time, the minting step would need a different synchronization approach. Low risk — `JoinCodeManager` already issues codes concurrently from `handleIssueCapabilities`. |
| A2 | `GetTailscaleStatus()` result (already in App.tsx startup state) contains the local node's hostname for `isPeerSelf` comparison | D-05/D-06 selection | If the hostname format in `GetTailscaleStatus` doesn't match the `session.hostname` from peer discovery, D-06 RW selection falls back to the safe default of RO for all remote viewers. Low risk — both come from the same Tailscale MagicDNS namespace. |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. The two `[ASSUMED]` items above are low-risk edge cases, not blocking decisions.

---

## Open Questions

1. **`GetTailscaleStatus` hostname format for D-06**
   - What we know: `GetTailscaleStatus` is called at startup (App.tsx); it returns a `tailscaleStatus` object with connectivity info.
   - What's unclear: Whether the local node hostname returned by `GetTailscaleStatus` matches the `session.hostname` string from peer discovery (which comes from `peer.Hostname` in `FetchAllPeerSessionsMeta`). If formats differ (e.g., short vs FQDN), `isPeerSelf` would always return false.
   - Recommendation: Check `GetTailscaleStatus` return type and compare hostname format to `RemotePeerSessions.hostname` at implementation time. If formats don't match, fall back to RO for all remote opens (D-05 safe); D-06 can be a fast follow.

2. **Should codes be emitted on every meta poll or only on the first request after sharing is enabled?**
   - What we know: Issuing codes on every 30-second meta request generates 2 new grant IDs per session per poll. Over time this could accumulate open grants.
   - What's unclear: Whether `AddGrant` + the associated memory growth is a concern at typical session counts.
   - Recommendation: Not a concern at typical scale (1-10 sessions, grants are just map entries). Accept the per-poll issuance as the simplest correct design.

---

## Sources

### Primary (HIGH confidence — verified from live codebase)
- `internal/webserver/capability_mw.go` — `requireCapability` middleware; exact 401 trigger at L40-41
- `internal/webserver/server.go` — `sessionMetaItem` struct (L42-54); `handleSessionsMeta` (L831-859); route registration (L515-529); tailnet bind IP trust model (L317-354)
- `internal/daemon/api.go` — `issueCapabilitiesForSession` (L1083-1175); `handleIssueCapabilities` (L1177-1206)
- `internal/daemon/types.go` — `IssueCapabilitiesResponse` struct (L155-169)
- `internal/daemon/client_remote_files.go` — `ExchangeJoinCodeAtURL` (L43-172); `RegisterRemoteCap` (L174-199)
- `internal/daemon/remote_caps.go` — `RemoteCapStore` design and security invariants
- `internal/tailnet/sessions.go` — `ShareableSessionMeta` (L113-123); `FetchPeerSessionsMeta` (L165-218); `FetchAllPeerSessionsMeta` (L233-267)
- `internal/webserver/sessions_meta_test.go` — RB-03 test contract (allowed key set, L188-216)
- `frontend/src/App.tsx:1062-1064` — `handleOpenRemoteSession` current implementation
- `frontend/src/lib/remoteAdapter.ts` — `adaptRemoteSession` carrying `session.url` from discovery
- `frontend/src/components/Hub/SessionCard.tsx:399-419,534-543` — "Open in browser" menu item and local "Open" button
- `frontend/src/lib/remoteSession.ts` — `RemoteSession`, `RemotePeerSessions`, `remoteBaseURLFor` interfaces

### Secondary (MEDIUM confidence — verified from related test/context files)
- `.planning/phases/137-share-modal-cap-model/137-CONTEXT.md` — Phase 137 cap model decisions; `issueCapabilitiesForSession` design intent
- `.planning/phases/146-open-session-capability-bug/146-CONTEXT.md` — D-01..D-08 locked decisions

---

## Metadata

**Confidence breakdown:**
- Mechanism recommendation (B over A): HIGH — verified from live code that no on-demand cap endpoint exists on the remote webserver; `/api/sessions/meta` is the established peer-to-viewer delivery channel
- Architecture patterns: HIGH — all function names, file locations, struct fields, and call sites verified from live codebase
- D-08 parity confirmation: HIGH — local "Open" and remote "Open in browser" call sites are distinct and unambiguous in the code
- Permission selection (D-05/D-06): MEDIUM — the RO/RW code selection via `isPeerSelf` is the right design but the exact hostname format for `tailscaleStatus` vs `session.hostname` needs verification at implementation time
- Pitfalls: HIGH — each pitfall cites the specific code location that would be broken

**Research date:** 2026-06-22
**Valid until:** 2026-07-22 (stable architecture; `JoinCodeManager` and cap model are well-established)
