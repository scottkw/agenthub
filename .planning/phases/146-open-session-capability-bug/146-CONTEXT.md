# Phase 146: Open Session Capability Bug - Context

**Gathered:** 2026-06-22
**Status:** Ready for planning

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

### Cap source — shared-gated, auto-delivered
- **D-01:** "Open" works **only when the session is shared** (the owner's Share toggle is on). The Share toggle remains the single owner-controlled gate for remote reachability — no new trust surface is introduced.
- **D-02:** When the session IS shared, the viewing node obtains the cap **automatically over the authenticated tailnet connection** — the user must NOT have to paste/enter a join code to open their own (or a shared) session. This reuses the proven Phase 122/137 cap model rather than inventing a new "mint-on-request for any peer" endpoint.
- **D-03:** When the session is NOT shared, "Open" must not dead-end on the raw `capability required` 401. Instead, surface a clear path to enable sharing (e.g., point the user at the Share action / disable+hint the Open control). Replacing the confusing dead-end is part of the fix.
- **D-04 (intent, mechanism deferred to research):** The DECISION is "Open just works for shared sessions, no manual code, inheriting the share's permission." The exact mechanism — auto-fetching the cap over an authenticated tailnet channel vs. having an authenticated discovery response carry a cap to trusted peers — is a **feasibility question for the research step**. Do not lock the mechanism here; research picks it.

### Permission level — match the source
- **D-05:** The opened session inherits the **share's** permission level: a read-only share opens read-only; a read-write share opens read-write. **No silent escalation** — opening a RO share must never grant write.
- **D-06:** For an owner re-attaching to their own session where both RO and RW are available, prefer **read-write** (the user expects to interact, per the #98 report: "Nothing ever opens inside the app itself").

### Scope & parity
- **D-07:** Scope the fix to the **remote open-in-browser** affordance — the literal #98 bug.
- **D-08:** Research/verification must **confirm** the local-session "Open" button and the GUI-vs-web paths are not separately broken (they share the same handler per the scout), so cross-surface parity is covered without expanding the build. See [[cross_surface_parity_release_blocker]].

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
