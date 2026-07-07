---
phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard
plan: 01
subsystem: auth
tags: [go, capability, websocket, funnel, csrf, rw-gate]

requires:
  - phase: 170-public-share-access-codes-reusable-share-lifetime-join-code-
    provides: JoinCodeManager (Issue/IssueReusable/Revoke/Rebind/Exchange atomic single-mutex primitive) and the grants map / isGrantActive enforcement point on WebServer
provides:
  - "JoinCodeManager.IssueSingleUseWithTTL(token, ttl) — single-use join code with a caller-supplied TTL, sibling to Issue/IssueReusable"
  - "WebServer.RemoveGrant(sessionID, grantID) — surgical single-grant revocation, siblings to AddGrant/ClearGrants"
  - "WebServer.SetRWGate/isRWGated + rwGated map — public-write consent gate state"
  - "Gate-aware originAllowedForWrite — Funnel-origin write requires isRWGated, tailnet-origin unaffected"
  - "TestHandleWSSRelay_WriteCap_RequiresGate — the real WS-upgrade proof that an unregistered write grant is rejected at the actual enforcement point (isGrantActive), not merely at an origin-check unit test"
affects: [171-02, 171-03, 171-04, join-exchange-http-boundary, daemon-funnel-toggle]

tech-stack:
  added: []
  patterns:
    - "Grant-registration gating (D-01) is the primary and sufficient terminal-write enforcement point; origin-based checks (D-02) are defense-in-depth for a narrower HTTP surface only — verify each layer at the boundary it actually reaches, not by proxy."
    - "Custom-TTL join-code primitive as a sibling function (IssueSingleUseWithTTL) rather than parameterizing the existing fixed-TTL Issue, to avoid touching a load-bearing atomic Exchange path."

key-files:
  created:
    - internal/webserver/rwgate_test.go
  modified:
    - internal/capability/joincode.go
    - internal/capability/joincode_test.go
    - internal/webserver/server.go
    - internal/webserver/capability_mw.go
    - internal/webserver/funnel_test.go

key-decisions:
  - "IssueSingleUseWithTTL reuses Issue's exact crypto/rand + joinCodeEncoding path and leaves Exchange untouched — the atomic lookup+expiry-check+delete under one mutex hold already handles both TTL classes and single-use semantics."
  - "RemoveGrant is a new surgical sibling to AddGrant/ClearGrants rather than reusing ClearGrants, which would delete every grant for a session (Pitfall 2 — would kill the session's ordinary tailnet read/write grants alongside the gate-minted write grant)."
  - "originAllowedForWrite's signature grew a sessionID parameter (extracted from claims.SID at its sole call site in requireFilesWrite) so the Funnel-origin branch can consult isRWGated; the tailnet-origin branch and the fail-closed empty-BaseURL behavior are unchanged."
  - "Updated the pre-existing TestOriginAllowedForWrite_FunnelOrigin (from Phase 165) to require SetRWGate before a Funnel-origin write passes — this is an intentional behavior change (Rule 1: the old assertion encoded the now-fixed accidental-write gap), not a regression."

patterns-established:
  - "R4 authorization-boundary proof: when a plan claims an enforcement point 'reaches' a code path, the required test drives the REAL request path (real handler, real upgrade) rather than a unit test of a helper function that may not actually be wired into that path (RESEARCH Pitfall 1 anti-pattern)."

requirements-completed: [FNL-09]

coverage:
  - id: D1
    description: "IssueSingleUseWithTTL mints a single-use join code honoring a caller-supplied TTL (not the manager's fixed 5-minute field); first Exchange succeeds, second fails; concurrent redeem yields exactly one winner"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_IssueSingleUseWithTTL"
        status: pass
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_IssueSingleUseWithTTL_CustomTTLHonored"
        status: pass
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_IssueSingleUseWithTTL_ExpiresAfterCustomTTL"
        status: pass
      - kind: unit
        ref: "internal/capability/joincode_test.go#TestJoinCodeManager_IssueSingleUseWithTTL_ConcurrentExchangeIsAtomic"
        status: pass
    human_judgment: false
  - id: D2
    description: "RemoveGrant surgically removes one grant, leaving sibling grants for the same session active"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestRemoveGrant_Surgical"
        status: pass
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestRemoveGrant_AbsentGrantIsNoOp"
        status: pass
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestRemoveGrant_AbsentSessionIsNoOp"
        status: pass
    human_judgment: false
  - id: D3
    description: "SetRWGate/isRWGated track per-session public-write consent gate state"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestSetRWGate_DefaultFalse"
        status: pass
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestSetRWGate_TrueThenFalse"
        status: pass
    human_judgment: false
  - id: D4
    description: "A write-permission capability whose grant is not registered is rejected at the real GET /sessions/{id}/ws upgrade (403/401); once registered, the same token completes the upgrade and its write permission actually reaches the PTY"
    requirement: FNL-09
    verification:
      - kind: integration
        ref: "internal/webserver/rwgate_test.go#TestHandleWSSRelay_WriteCap_RequiresGate"
        status: pass
    human_judgment: false
  - id: D5
    description: "Gate-aware originAllowedForWrite: Funnel-origin write rejected when non-gated, permitted when gated; tailnet-origin write unaffected; fail-closed preserved on empty FunnelBaseURL"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/webserver/rwgate_test.go#TestOriginAllowedForWrite_RWGate"
        status: pass
      - kind: unit
        ref: "internal/webserver/funnel_test.go#TestOriginAllowedForWrite_FunnelOrigin"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-07-07
status: complete
---

# Phase 171 Plan 01: RW Enforcement Primitives Summary

**Two enforcement primitives (grant-registration gating at the real WS upgrade + Funnel-origin RW-gate check) and a custom-TTL single-use join code, closing the internet-write RCE surface at the layer that actually reaches the PTY-write path.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-07-07T15:52:00-05:00 (approx.)
- **Completed:** 2026-07-07T16:00:48-05:00
- **Tasks:** 3
- **Files modified:** 5 (1 new: `internal/webserver/rwgate_test.go`)

## Accomplishments

- `IssueSingleUseWithTTL(token, ttl)` mints a single-use join code with a caller-supplied TTL, siblings to `Issue`/`IssueReusable`, leaving the atomic `Exchange` path untouched.
- `RemoveGrant(sessionID, grantID)` surgically revokes one grant without disturbing a session's other active grants (unlike `ClearGrants`).
- `SetRWGate`/`isRWGated` + `rwGated map[string]bool` on `WebServer` track the public-write consent gate state.
- `TestHandleWSSRelay_WriteCap_RequiresGate` proves — through the REAL `GET /sessions/{id}/ws` upgrade handler, not a unit test of a helper — that a write-permission capability whose grant is unregistered is rejected (403/401) at `requireCapability`'s `isGrantActive` check, and that once `AddGrant` registers it, the identical token both completes the upgrade and reaches the PTY as a genuine writer.
- `originAllowedForWrite` gained a `sessionID` parameter and now requires `isRWGated(sessionID)` on the Funnel-origin branch — defense-in-depth for the `files.write` HTTP routes only. Its stale doc-comment (falsely claiming coverage of `MsgInput`/`MsgSessionInject`) was corrected to state the accurate boundary.

## Task Commits

Each task was committed atomically:

1. **Task 1: IssueSingleUseWithTTL join-code primitive** - `5d7ef982` (feat)
2. **Task 2: RemoveGrant + rwGated state on WebServer + WS-relay pre-gate rejection test** - `56049117` (feat)
3. **Task 3: gate-aware originAllowedForWrite (D-02 defense-in-depth)** - `e58483d2` (feat)

_Note: all three tasks are `tdd="true"`; tests were authored alongside each implementation and verified failing-then-passing in the scoped `go test -race` runs shown below before each commit._

## Files Created/Modified

- `internal/capability/joincode.go` - added `IssueSingleUseWithTTL(token, ttl)`
- `internal/capability/joincode_test.go` - 4 new tests: format/single-use, custom-TTL-honored, custom-TTL-expiry, concurrent-redeem atomicity
- `internal/webserver/server.go` - added `RemoveGrant`, `SetRWGate`, `isRWGated`, and the `rwGated map[string]bool` field
- `internal/webserver/capability_mw.go` - `originAllowedForWrite` gained `sessionID` param + `isRWGated` check on the Funnel-origin branch; doc-comment corrected
- `internal/webserver/rwgate_test.go` (NEW) - `TestRemoveGrant_*`, `TestSetRWGate_*`, `TestHandleWSSRelay_WriteCap_RequiresGate`, `TestOriginAllowedForWrite_RWGate`
- `internal/webserver/funnel_test.go` - updated `TestOriginAllowedForWrite_FunnelOrigin` to require `SetRWGate` before a Funnel-origin write passes (see Deviations)

## Decisions Made

- `IssueSingleUseWithTTL` mirrors `IssueReusable`'s exact RNG/encoding path; `reusable` stays at its zero value (`false`) so `Exchange`'s existing atomic delete-on-first-redeem is reused verbatim — no new concurrency surface introduced.
- `RemoveGrant` is a new, separate method rather than a `ClearGrants` reuse — `ClearGrants` wipes the entire per-session grant set, which would also revoke the session's unrelated tailnet read/write grants (Pitfall 2).
- `originAllowedForWrite`'s only call site (`requireFilesWrite`) now passes `claims.SID` (already in scope after `requireCapability` succeeds) — no new claims-extraction logic needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated the pre-existing `TestOriginAllowedForWrite_FunnelOrigin` test to reflect the new gate requirement**
- **Found during:** Task 3 (`go vet` failure — compile error from the changed `originAllowedForWrite` signature)
- **Issue:** The Phase 165 test asserted that a Funnel-origin write passes as soon as `EnableFunnel` runs, with no RW-gate concept. That assertion is precisely the pre-171 behavior this plan's objective calls a security gap (the plan's own preamble: "Supersedes today's ACCIDENTAL public write"). Task 3's intentional behavior change made the old assertion false.
- **Fix:** Updated the test to add an assertion that Funnel-origin still fails post-`EnableFunnel` when non-gated, then calls `ws.SetRWGate(sessionID, true)` before asserting the origin now passes. Signature call sites updated to pass `sessionID`.
- **Files modified:** `internal/webserver/funnel_test.go`
- **Verification:** `go test -race ./internal/webserver/... -run TestOriginAllowedForWrite` — both `TestOriginAllowedForWrite_FunnelOrigin` and the new `TestOriginAllowedForWrite_RWGate` pass.
- **Committed in:** `e58483d2` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — test update required by an intentional, plan-mandated behavior change)
**Impact on plan:** No scope creep; the fix is the direct consequence of Task 3's own objective (closing the accidental-write gap) and was required for the package to compile and for the pre-existing test to assert the now-correct behavior.

## Issues Encountered

- `TestRelay_MixedReplyAndKeystrokes` (in `oscabsorb_relay_test.go`, unrelated to this plan) failed once under `-race -count=1` with a pipe-read timeout, then passed on immediate re-run and on a full-suite re-run. Confirmed as pre-existing test flakiness, not caused by any change in this plan (the file was not touched and the failure mode — a timing-sensitive PTY-pipe read — is unrelated to the grant/gate/origin logic modified here).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The three primitives (`IssueSingleUseWithTTL`, `RemoveGrant`, `SetRWGate`/`isRWGated`, gate-aware `originAllowedForWrite`) are ready for Plan 02 to wire into the daemon-side consent-gate flow (minting the single-use write code, calling `SetRWGate` on redemption, and calling `RemoveGrant`/clearing the gate on teardown).
- `go test -race ./internal/capability/... ./internal/webserver/...` is green; `grep -n "originAllowedForWrite"` confirms no accidental new wiring into the relay path.
- No blockers for 171-02.

---
*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Completed: 2026-07-07*

## Self-Check: PASSED

All created/modified files found on disk; all 4 task/summary commit hashes (5d7ef982, 56049117, e58483d2, ee5313ab) found in git log.
