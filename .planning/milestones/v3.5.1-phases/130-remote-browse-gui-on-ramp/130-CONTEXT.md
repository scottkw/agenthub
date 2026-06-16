# Phase 130: Remote Browse GUI On-Ramp - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss) + resolved open design decision (#86)

<domain>
## Phase Boundary

The desktop GUI Remote Sessions panel can discover, list, and open a tailnet peer's file browser — completing the umbrella #24 on-ramp and retiring the epic.

The remote read/write **data path is already proven live** (relay routes mounted `58af6d6`, discovery probe accepts cap-protected peers `3508bd7`, share-link relabel `e45ccba`). This phase fixes the GUI **on-ramp**: discover → list → pick a tailnet peer's shareable sessions and open one in the File Browser, end-to-end over the relay loopback the Wails GUI uses.

Requirements: RB-01, RB-02, RB-03, RB-04, RB-05.

</domain>

<decisions>
## Implementation Decisions

### RESOLVED — #86 remote-browse architecture (user decision, 2026-06-15)

**(a) Tailnet-trusted metadata-only discovery endpoint.**

- Add a discovery endpoint that returns **shareable-session metadata** (enough to list and pick a session) to **tailnet-trusted** callers.
- Content and capabilities stay **locked** — the endpoint exposes metadata only, never session content or a capability/cap token without the intended grant. This **preserves the Phase 87/88 no-enumeration security model** (RB-03).
- This directly satisfies RB-01 (a reachable peer's shareable sessions are visible — peers are no longer silently dropped because `/api/sessions` isn't enumerable without a session-scoped cap) and RB-04 (honest panel states — a reachable peer with shareable sessions is never shown as "No remote peers found"; genuinely empty/unreachable peers still surface a correct empty/error state).
- "Tailnet-trusted" = the caller is verified to be on the tailnet (the existing Phase 87/88 trust model for tailnet peers); a non-tailnet / unauthorized caller still cannot enumerate session content or obtain a cap (RB-03).

### Claude's Discretion (mechanics)

Endpoint path/shape, exactly what metadata fields are returned (must be the minimum needed to list+pick: e.g. session id, label, working-dir display name — NOT content), how the GUI renders the discovered list, and how the pick flow hands off to the File Browser are at Claude's discretion, guided by the existing remote-session and file-browser GUI patterns (Phases 52, 120, 122) and the resolved #86 decision. The session-scoped cap acquisition for actually opening a session reuses the existing web-share cap / join-code flow (see [[project_remote_browse_cap_reuse]], [[project_phase_122_design_locked]]).

</decisions>

<code_context>
## Existing Code Insights

Gathered during plan-phase research. Known landmarks from prior milestone work:
- Remote session discovery: `internal/tailnet/` (peer discovery), `app.go` `GetRemoteSessions` (discovers tailnet peers + fetches session lists).
- Relay surface (the GUI's actual path): `internal/relay/` + `internal/daemon/relay_remote_files.go` (+ `relay_remote_files_test.go`). RB-05 regression test MUST exercise this relay loopback — NOT just the webserver/fixture surface. The v3.5 audit scored 98/100 while blind to a 4-layer breakage precisely because tests only hit the webserver/fixture surface.
- File Browser GUI: `frontend/src/components/FileBrowserTab.tsx` (Phase 120); remote browse wiring (Phase 122, join-code modal for cap, CORS via local-daemon proxy).
- Phase 87/88 capability/no-enumeration model: the security guarantee RB-03 must preserve.
- DNS-03 banner (Phase 129) already warns when `accept-dns=false` before remote browse — the on-ramp this phase builds is where that banner contextually lives.

</code_context>

<specifics>
## Specific Ideas

- **Mandatory relay-surface coverage (RB-05, release-blocking per REQUIREMENTS.md):** the discover→list→pick path must be covered by a relay-surface regression test (`internal/relay/server_files_test.go` or `internal/daemon/relay_remote_files_test.go`), guarding against the v3.5-class blind spot.
- **Security (RB-03):** preserve the Phase 87/88 no-enumeration guarantee — an unauthorized / non-tailnet caller cannot enumerate session content or obtain a cap without the intended grant. The new discovery endpoint is metadata-only and tailnet-trusted.
- **Honest states (RB-04):** reachable peer with shareable sessions never shows "No remote peers found"; genuinely empty/unreachable peers surface a correct empty/error state.
- This phase retires umbrella epic **#24**.

</specifics>

<deferred>
## Deferred Ideas

None for this phase. #82 (TUI Files upload parity) remains deferred to a later milestone (out of scope per REQUIREMENTS.md).

</deferred>
