# Phase 146: Open Session Capability Bug - Context

**Gathered:** 2026-06-22
**Revised:** 2026-06-22 — design reversed to out-of-band (broadcast approach superseded after first execution; see decisions D-02/D-04/D-09..D-12)
**Status:** Ready for re-planning

<domain>
## Phase Boundary

Fix FIX-03 / #98: clicking "Open" (the Hub session card "Open in browser" menu item) on a **remote** peer's session must open the live session in the browser instead of landing on the webserver's `capability required` (HTTP 401) page.

Root cause (confirmed by codebase scout): the remote "Open in browser" handler opens `session.url` verbatim — a bare `https://peer/sessions/{id}` with **no `?cap=` token** — because remote discovery (`/api/sessions/meta`, RB-03) intentionally omits cap tokens. The webserver's `requireCapability` middleware (`internal/webserver/capability_mw.go:41`) rejects any request lacking `?cap=`. The working path (pasting a read-only share link) succeeds only because the share modal mints a cap via `IssueCapabilities()` and embeds `?cap=TOKEN` (`internal/daemon/api.go:1162`).

The complication that drove this discussion: the opened session lives on a **different tailnet peer**, and a valid cap can only be minted by the **owning** peer — so the fix is not simply "append a cap locally."

**In scope:** the remote-session "Open in browser" affordance (GUI + web, same handler).
**Out of scope:** any new general-purpose "mint a cap for any tailnet peer on demand" capability; reworking the share/cap model itself; the in-app (local) attach flow beyond confirming it isn't separately broken.
</domain>

<decisions>
## Implementation Decisions

> **DESIGN REVERSAL (2026-06-22, user decision).** The original D-02/D-04 ("Open just
> works, no manual code; cap auto-delivered over discovery") and the RESEARCH "Mechanism B"
> (broadcast RO+RW join codes in `/api/sessions/meta`) were **rejected** after execution.
> Broadcasting places credentials on an unauthenticated payload every tailnet peer can poll,
> and `/join/exchange` does no identity check. The new model is **out-of-band**: codes/links
> are never broadcast; the owner deliberately hands one out. See `superseded-broadcast/README.md`,
> `146-REVIEW.md`, `146-VERIFICATION.md`. D-02, D-04, D-06 below are rewritten; D-09..D-12 are new.

### Cap source — shared-gated, owner-delivered out of band
- **D-01:** "Open" works **only when the session is shared** (the owner's Share toggle is on). The Share toggle remains the single owner-controlled gate for remote reachability — no new trust surface is introduced.
- **D-02 (REWRITTEN):** Codes/links are **NOT broadcast**. To open a remote (cross-machine) session, the viewer must hold a credential the **owner deliberately delivered out of band** — a join code (pasted into the existing `RemoteJoinCodeModal`) or a share link (already cap-bearing). No code is auto-placed in the discovery payload. Access is an explicit grant, never access-by-default for tailnet peers.
- **D-03:** When the viewer has no credential for a session, "Open" must not dead-end on the raw `capability required` 401. Instead it surfaces the paste-a-code path (open `RemoteJoinCodeModal` with an "open session" intent) and/or a clear "ask the owner to share" hint. Replacing the confusing dead-end is part of the fix.
- **D-04 (REWRITTEN — mechanism LOCKED):** The mechanism is **reuse of the Phase 122 out-of-band flow**: owner generates a code/link in the Share modal → delivers it out of band → viewer pastes the code into `RemoteJoinCodeModal` → `ExchangeJoinCodeAtURL` → open `baseURL + /sessions/{id}?cap=TOKEN` via `BrowserOpenURL`. No new endpoint, no broadcast. The `/api/sessions/meta` payload stays cap-free (RB-03 fully restored).

### Permission level — owner-chosen, match the credential
- **D-05:** The opened session inherits the permission of **the code/link the owner handed out**: a read-only code opens read-only; a read-write code opens read-write. **No silent escalation.**
- **D-06 (REWRITTEN):** RO-vs-RW is the **owner's explicit choice at share time** (which code/link they generate and send), NOT a client-side guess. The rejected `isPeerSelf` hostname-matching selection (dead code per REVIEW WR-01) is removed.

### Scope & parity
- **D-07:** Scope the fix to the **remote open-in-browser** affordance — the literal #98 bug.
- **D-08:** Research/verification must **confirm** the local-session "Open" button and the GUI-vs-web paths are not separately broken (they share the same handler per the scout), so cross-surface parity is covered without expanding the build. See [[cross_surface_parity_release_blocker]].

### Out-of-band model — new decisions (2026-06-22)
- **D-09 (same-machine owner re-attach untouched):** Reopening your own session on the **same machine** already works one-click via the local loopback path (`handleOpenSessionTab` → relay WS, no cap). The fix MUST NOT regress or reroute this. Out-of-band applies only to **cross-machine** opens (your own session from another machine, or someone else's session).
- **D-10 (remove the broadcast):** The re-planned phase MUST remove the superseded broadcast: `mintSessionJoinCodes` wiring into `/api/sessions/meta` (`SetJoinCodeIssuer`), the `ROJoinCode`/`RWJoinCode` fields on `sessionMetaItem` (server.go) and `ShareableSessionMeta` (tailnet/sessions.go), the frontend code-threading, and the `ro_join_code`/`rw_join_code` allow-list entries — restoring RB-03 to cap-free discovery. `app.go` is NOT to be wired to carry codes (there are none).
- **D-11 (reuse, don't rebuild):** Reuse the existing Phase 122 `RemoteJoinCodeModal` + `ExchangeJoinCodeAtURL` + `RegisterRemoteCap`. The modal today serves a file-browse intent; extend it (or its caller) with an "open session in browser" intent that, after exchange, opens `baseURL + /sessions/{id}?cap=TOKEN` via `BrowserOpenURL` instead of (or in addition to) depositing the cap for file proxying.
- **D-12 (owner copy affordance):** The Share modal must surface a clear **copy-the-code / copy-the-link** affordance so the owner has something concrete to hand out of band. If `IssueCapabilities` already returns codes/links but the UI doesn't expose a copy control, adding it is in scope.

### Deferred (not this phase)
- **Identity-authenticated one-click cross-machine open:** making your *own* session open cross-machine with zero paste would require real tailnet-identity auth on `/join/exchange` (none today). Larger effort; out of scope. The code/link flow is acceptable for this fix.

### Claude's Discretion
- Exact UI treatment of the "not shared yet → share first" affordance (disabled control + tooltip vs. redirect to Share modal) is left to planning, as long as it replaces the raw 401 dead-end.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

No formal ADRs/specs exist for this bug-fix phase; the cap model is defined in code and prior-phase context. The authoritative sources:

### The bug & the cap model
- `internal/webserver/capability_mw.go` §`requireCapability` (~L37-75) — the `?cap=` check that emits `capability required` (L41). This is the gate the fix must satisfy.
- `internal/daemon/api.go` §`issueCapabilitiesForSession` (~L1092-1175) — how RO+RW caps are minted and cap-bearing URLs are built (L1162-1164). The pattern the Open flow must reuse.
- `internal/webserver/server.go:848` — where the current bare (cap-less) `sessionURL` is built.
- `frontend/src/components/Hub/SessionCard.tsx` — the "Open" button (L534, local) and remote "Open in browser" menu item (L399-419); `remoteUrl` source (L237).
- `frontend/src/App.tsx:1062-1064` — `handleOpenRemoteSession` → `BrowserOpenURL(url)` (shared GUI/web handler).
- `frontend/src/lib/remoteAdapter.ts` — `adaptRemoteSession()` carries `session.url` through from discovery.

### Cap/share/remote-trust prior decisions
- `.planning/phases/137-share-modal-cap-model/137-CONTEXT.md` — one share action mints RO+RW cap-tokens; file-browse derives from token.
- RB-03 (REQUIREMENTS.md) — remote discovery (`/api/sessions/meta`) deliberately excludes cap tokens (network-layer trust for listing only).
- Phase 122 design — remote browse uses a join-code modal for cap + local-daemon proxy for CORS; the established "viewer obtains a cap to reach a remote session's resources" pattern.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `IssueCapabilities(sessionId)` (daemon, via `SessionShareModal.tsx` L141/167/214) — already mints cap-bearing URLs; the Open flow should reuse this rather than build a parallel minting path.
- The remote-file-browse cap-reuse mechanism (memory: desktop GUI reuses the session's web-share cap, no separate cap flow) — the closest existing analog for "viewer reaches a remote session with a cap."

### Established Patterns
- `requireCapability` middleware gates session routes on a `?cap=` token; any opened URL MUST carry a valid cap.
- Remote sessions are discovered cap-free (RB-03); caps are obtained separately, gated by the owner's Share toggle.

### Integration Points
- The single shared handler `handleOpenRemoteSession` (App.tsx) means GUI + web fixes land in one place — good for parity.
- The owner-side mint (`issueCapabilitiesForSession`) and the viewer-side open must be bridged over the authenticated tailnet channel (mechanism TBD by research).

</code_context>

<specifics>
## Specific Ideas

- Reported environment (#98): two of the user's own Macs on the same tailnet (Tahoe 26.5.1), session running on the "main" Mac, opened from the other. The main session was shared (a working read-only link existed) — so in the real scenario the session IS shared, which fits the shared-gated decision.
- The user's expectation is to actually use the opened session ("Nothing ever opens inside the app itself"), reinforcing read-write for owner re-attach (D-06).

</specifics>

<deferred>
## Deferred Ideas

- A general "open any tailnet peer's session without sharing it first" / auto-mint-on-request capability — explicitly out of scope (new security surface; would need its own phase + security review). The shared-gated model (D-01) is the deliberate boundary.
- Opening a remote session *inside the app* (native in-app remote attach) rather than the browser — not part of #98; current remote affordance is browser-based.

None of the above blocks this phase.

</deferred>

---

*Phase: 146-open-session-capability-bug*
*Context gathered: 2026-06-22*
