---
phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard
plan: 02
subsystem: auth
tags: [go, capability, daemon, funnel, rw-gate, wails-bindings]

requires:
  - phase: 171-01
    provides: "IssueSingleUseWithTTL, RemoveGrant, SetRWGate/isRWGated, gate-aware originAllowedForWrite — the enforcement primitives this plan wires into the daemon lifecycle"
provides:
  - "handleSetSessionFunnelWrite (POST /sessions/{id}/funnel-write): mints a terminal-only (Perms hardcoded \"read,write\", never browseEnabledFor-derived) public write capability with a single-use join code, registers ws.AddGrant + ws.SetRWGate(true), and clamps ExpiresIn unconditionally to (0, 3600]"
  - "revokeFunnelWriteLocked: the single shared teardown primitive — surgically revokes the write grant/code/gate/timer, never touches funnelSessions/funnelReadCode, never calls ws.DisableFunnel or ClearGrants"
  - "disableFunnelWriteForSession + DELETE /sessions/{id}/funnel-write: the RW-only disable path, delegating solely to revokeFunnelWriteLocked"
  - "One revokeFunnelWriteLocked call appended inside disableFunnelForSession's existing cascade (D-03): funnel-off / web-share-off / session-exit / auto-expiry now also revoke any live write cap"
  - "D-04 fix: issueCapabilitiesForSession's WriteURL (tailnet Full Access Link) never rebases to FunnelBaseURL, even for an active Funnel session — closes the accidental public-write exposure (T-171-07)"
  - "DaemonClient.SetSessionFunnelWrite/DisableSessionFunnelWrite + App.SetSessionFunnelWrite/DisableSessionFunnelWrite RPC surface"
  - "SessionInfo.FunnelWriteActive (no omitempty) + hand-synced Wails TS bindings (App.d.ts, models.ts, App.js) for 171-03/04's frontend wiring"
affects: [171-03, 171-04]

tech-stack:
  added: []
  patterns:
    - "revokeFunnelWriteLocked is the single shared sub-teardown, called from the RW-only HTTP disable path AND appended inside disableFunnelForSession's cascade — never the reverse; mirrors the FNL-08 funnelReadCode teardown-chokepoint pattern but scoped to the write cap only (never funnelSessions/ClearGrants/ws.DisableFunnel)."
    - "The mint handler re-invokes the shared teardown before installing fresh state, so a re-mint without an intervening disable cannot leak the previous grant/code or leave a stale timer that could later revoke the NEW mint out from under it (generalizes the read-Funnel T-165-13 double-fire guard)."
    - "A gate-minted public capability's ExpiresIn is clamped unconditionally to a bounded max (here 1h) — deliberately distinct from the existing read-Funnel handler's ExpiresIn==0-means-unbounded semantics, because a public WRITE cap must never be long-lived."

key-files:
  created: []
  modified:
    - internal/daemon/types.go
    - internal/daemon/api.go
    - internal/daemon/client.go
    - app.go
    - internal/daemon/funnel_test.go
    - frontend/src/wailsjs/go/main/App.d.ts
    - frontend/src/wailsjs/go/main/App.js
    - frontend/src/wailsjs/go/models.ts
    - TESTING.md
    - (12 frontend test/lib fixture files, adding `funnelWriteActive: false` for the new required SessionInfo field)

key-decisions:
  - "D-04: split issueCapabilitiesForSession's single `base` variable into `readBase`/`writeBase` — only readBase rebases to FunnelBaseURL; writeBase always stays on the tailnet BaseURL. The public write cap is minted ONLY by the new gate handler, never by ordinary capability issuance."
  - "The RW-disable HTTP path is a separate DELETE /sessions/{id}/funnel-write route (not an Enabled:false field on the POST body) — mirrors the existing DELETE /sessions/{id} pattern and keeps SetSessionFunnelWriteRequest exactly as specified (ExpiresIn int only)."
  - "App.js (the hand-authored runtime Call() wrapper file, distinct from the type-only App.d.ts) was updated even though the plan's files_modified list didn't name it — required for SetSessionFunnelWrite/DisableSessionFunnelWrite to actually invoke at runtime, per the project's own funnelBinding.contract.test.tsx precedent guarding this exact class of gap (Rule 2)."
  - "TESTING.md: added the FNL-09 Suite Manifest note + traceability rows for this plan's tests, and backfilled the missing FNL-09 rows for 171-01's test additions (joincode_test.go, the new rwgate_test.go, funnel_test.go's updated origin test) since 171-01 shipped without a TESTING.md update and 171-02 builds directly on those primitives for the same requirement."
  - "requirements-completed lists FNL-09 for traceability, but REQUIREMENTS.md's checkbox was NOT flipped — FNL-09 spans all 4 plans of this phase (frontend wiring is 171-03/04) and this project's established convention (per 170's precedent) is to keep ROADMAP/REQUIREMENTS pending until the whole phase is UAT-verified, not per-plan."

coverage:
  - id: D1
    description: "Gate-minted write cap Perms is exactly \"read,write\" regardless of the session's browse toggle (terminal-only, never files.write) — the handler never calls browseEnabledFor"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/daemon/funnel_test.go#TestFunnelWriteGate_TerminalOnlyScope"
        status: pass
      - kind: other
        ref: "grep -n \"browseEnabledFor\" internal/daemon/api.go — no call inside handleSetSessionFunnelWrite"
        status: pass
    human_judgment: false
  - id: D2
    description: "ExpiresIn is clamped unconditionally to (0, 3600] server-side — 0 or >3600 becomes exactly 3600; in-range values pass through unchanged (NOT the read handler's 0==unbounded semantics)"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/daemon/funnel_test.go#TestHandleSetSessionFunnelWrite_ExpiryClamp"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-04: issueCapabilitiesForSession's WriteURL (tailnet Full Access Link) never rebases to FunnelBaseURL even for an active Funnel session; ReadURL's existing Funnel-rebase is unaffected"
    requirement: FNL-09
    verification:
      - kind: unit
        ref: "internal/daemon/funnel_test.go#TestIssueCapabilitiesForSession_WriteRebaseRemoved"
        status: pass
    human_judgment: false
  - id: D4
    description: "RW-only disable (DELETE /sessions/{id}/funnel-write) revokes exactly the write grant/code/gate/timer, leaving the reusable public read code resolving and Funnel/funnelSessions untouched"
    requirement: FNL-09
    verification:
      - kind: integration
        ref: "internal/daemon/funnel_test.go#TestDisableFunnelWrite_RevokesGrantOnly"
        status: pass
    human_judgment: false
  - id: D5
    description: "All teardown triggers (funnel-off, session natural end, the write cap's own auto-expiry timer) cascade into write-cap revocation; no orphaned write cap survives any trigger"
    requirement: FNL-09
    verification:
      - kind: integration
        ref: "internal/daemon/funnel_test.go#TestFunnelWriteTeardown_AllTriggers"
        status: pass
    human_judgment: false
  - id: D6
    description: "SessionInfo.FunnelWriteActive serializes false (no omitempty) so the frontend poll can detect a true->false teardown flip; DaemonClient/App RPC surface + hand-synced Wails TS bindings compile"
    requirement: FNL-09
    verification:
      - kind: other
        ref: "grep -n 'FunnelWriteActive' internal/daemon/types.go — no `,omitempty` json tag"
        status: pass
      - kind: other
        ref: "go build ./... && cd frontend && pnpm exec tsc --noEmit"
        status: pass
    human_judgment: false

duration: 26min
completed: 2026-07-07
status: complete
---

# Phase 171 Plan 02: Daemon RW-Gate Lifecycle Summary

**Gate-minted, terminal-only public write capability (single-use code, RW consent gate, unconditional 1h expiry clamp) with a surgical teardown that never disturbs the reusable public read share, plus the D-04 fix that stops the accidental tailnet-write-cap-to-Funnel-base rebasing.**

## Performance

- **Duration:** 26 min
- **Started:** 2026-07-07T16:05:00-05:00 (approx.)
- **Completed:** 2026-07-07T16:30:17-05:00
- **Tasks:** 3
- **Files modified:** 20 (0 new files; 12 of the 20 are frontend test/lib fixtures updated only to satisfy the new required `SessionInfo.funnelWriteActive` field)

## Accomplishments

- `handleSetSessionFunnelWrite` (`POST /sessions/{id}/funnel-write`) mints a terminal-only public write capability: `Perms` hardcoded `"read,write"` (never derived from `a.engine.browseEnabledFor` — T-171-06/D-05/Pitfall 5), `ExpiresIn` clamped unconditionally to `(0, 3600]` (R5/D-11/Pitfall 6 — deliberately not the read Funnel handler's `0`-means-unbounded semantics), and atomically applies `ws.AddGrant` + `ws.SetRWGate(true)` + `joinCodes.IssueSingleUseWithTTL` + an expiry timer.
- `revokeFunnelWriteLocked` is the single shared teardown primitive: surgically revokes the write grant/code/gate/timer, never touches `funnelSessions`/`funnelReadCode`, and never calls `ws.DisableFunnel` or `ClearGrants` (Pitfall 3). It is called from BOTH the new RW-only `DELETE /sessions/{id}/funnel-write` path (`disableFunnelWriteForSession`) and appended inside `disableFunnelForSession`'s existing cascade (D-03) — so funnel-off, web-share-off, session-exit, and auto-expiry all revoke a live write cap too, and it is also invoked at the top of the mint handler so a re-mint without an intervening disable cannot leak the prior grant/code or leave a stale timer.
- D-04: `issueCapabilitiesForSession` no longer shares one `base` variable between the read and write URLs — `readBase` still rebases to `FunnelBaseURL()` for a Funnel session (FNL-08, unchanged), but `writeBase` always stays on the tailnet `BaseURL()`. The owner's "Full Access Link" can never again be silently turned into a public URL by enabling Funnel (T-171-07, the accidental-write gap this whole phase exists to close).
- `DaemonClient`/`App` RPC surface (`SetSessionFunnelWrite`, `DisableSessionFunnelWrite`) and the hand-synced Wails TS binding trio (`App.d.ts` types, `models.ts` response class, `App.js` runtime `Call()` wrappers) are ready for 171-03/04's frontend wiring.
- `SessionInfo.FunnelWriteActive` (no `omitempty`) is populated in `handleListSessions` from the `funnelWriteGrant` map presence, mirroring the existing `FunnelActive`/`BrowseEnabled` poll-detection pattern.

## Task Commits

Each task was committed atomically:

1. **Task 1: types, struct fields, D-04 write-rebase removal, client RPC + Wails binding** - `af53dd7c` (feat)
2. **Task 2: handleSetSessionFunnelWrite mint handler + route (terminal-only, expiry clamp)** - `7e34a8fe` (feat, tdd)
3. **Task 3: revokeFunnelWriteLocked teardown + all-trigger wiring + RW-only disable path** - `26066de7` (feat, tdd)

**Plan metadata:** `7cb2f936` (docs: TESTING.md traceability — see Deviations)

_Note: Tasks 2 and 3 are `tdd="true"`; tests were authored alongside each implementation and verified passing in the scoped `go test -race` runs shown in each commit message before committing._

## Files Created/Modified

- `internal/daemon/types.go` - `SessionInfo.FunnelWriteActive`; `SetSessionFunnelWriteRequest`/`SetSessionFunnelWriteResponse`
- `internal/daemon/api.go` - `funnelWriteGrant`/`funnelWriteCode`/`funnelWriteExpiry` API fields; D-04 `readBase`/`writeBase` split; `handleSetSessionFunnelWrite`; `revokeFunnelWriteLocked`; `disableFunnelWriteForSession`; `handleDisableSessionFunnelWrite`; `POST`/`DELETE /sessions/{id}/funnel-write` routes; `disableFunnelForSession` cascade append; `handleListSessions` FunnelWriteActive population
- `internal/daemon/client.go` - `DaemonClient.SetSessionFunnelWrite`/`DisableSessionFunnelWrite`
- `app.go` - `App.SetSessionFunnelWrite`/`DisableSessionFunnelWrite`
- `internal/daemon/funnel_test.go` - `TestIssueCapabilitiesForSession_WriteRebaseRemoved`, `TestFunnelWriteGate_TerminalOnlyScope`, `TestHandleSetSessionFunnelWrite_ExpiryClamp`, `TestDisableFunnelWrite_RevokesGrantOnly`, `TestFunnelWriteTeardown_AllTriggers`
- `frontend/src/wailsjs/go/main/App.d.ts` - `SessionInfo.funnelWriteActive`; `SetSessionFunnelWrite`/`DisableSessionFunnelWrite` signatures
- `frontend/src/wailsjs/go/main/App.js` - real `Call()` wrappers for the two new bound methods (Rule 2 deviation, see below)
- `frontend/src/wailsjs/go/models.ts` - `daemon.SetSessionFunnelWriteResponse`
- `TESTING.md` - FNL-09 Suite Manifest note + traceability rows (this plan + 171-01 backfill)
- 12 frontend test/lib files - added `funnelWriteActive: false` to `SessionInfo` fixtures (tsc gate)

## Decisions Made

- D-04 fix implemented as a `readBase`/`writeBase` variable split rather than an `if isWriteCall` branch — keeps the existing read-Funnel-rebase logic byte-for-byte unchanged while making the write path's tailnet-only behavior structurally obvious at the call site.
- `revokeFunnelWriteLocked` takes `*webserver.WebServer` as a parameter (not reading `a.webServer` itself) so callers that already hold `a.mu` and have a snapshotted `ws` (like `disableFunnelForSession`) can pass it directly without a second lock acquisition.
- The RW-disable route is `DELETE /sessions/{id}/funnel-write` (separate route), not an `Enabled` field on the POST body — keeps `SetSessionFunnelWriteRequest` exactly as the plan specified and mirrors the existing `DELETE /sessions/{id}` pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `App.js` runtime `Call()` wrappers for the new bound methods**
- **Found during:** Task 1 (Wails TS binding sync)
- **Issue:** The plan's `files_modified` list only named `App.d.ts` and `models.ts`, but this project's actual binding architecture splits type declarations (`App.d.ts`) from the runtime `Call('main.App.X', [...])` wrapper (`App.js`) — confirmed by the project's own `funnelBinding.contract.test.tsx`, which exists specifically to guard against updating only the type stub and leaving the runtime wrapper missing (its own doc comment: "Pitfall 1: updating only the generated wailsjs/wailsjs/go tree (not imported by the app)"). Without `App.js`, `SetSessionFunnelWrite`/`DisableSessionFunnelWrite` would type-check but throw `undefined is not a function` the moment 171-03/04's frontend code called them.
- **Fix:** Added the two `Call()` wrapper exports to `App.js` alongside the `App.d.ts`/`models.ts` changes.
- **Files modified:** `frontend/src/wailsjs/go/main/App.js`
- **Verification:** `go build ./...` and `cd frontend && pnpm exec tsc --noEmit` both clean; manual read of the existing `App.js` confirmed the established `Call('main.App.X', [...])` pattern was followed exactly.
- **Committed in:** `af53dd7c` (Task 1 commit)

**2. [Rule 1 - Bug] Added `funnelWriteActive: false` to 12 frontend `SessionInfo` test/lib fixtures**
- **Found during:** Task 1 (frontend `tsc --noEmit` gate)
- **Issue:** Adding the new required (non-optional) `funnelWriteActive: boolean` field to the hand-authored `SessionInfo` interface in `App.d.ts` broke `tsc --noEmit` in 12 pre-existing test/lib files that construct `SessionInfo` object literals without it (`TS2741`/`TS2322`).
- **Fix:** Added `funnelWriteActive: false` next to the existing `funnelActive: false` line in each fixture/factory (mirroring the exact same default), or to the object literal for the two files without a shared factory.
- **Files modified:** `frontend/src/components/__tests__/App.open-remote.test.tsx`, `frontend/src/components/__tests__/SessionCard.share.test.tsx`, `frontend/src/components/Hub/HubBriefingModal.test.tsx`, `HubFilterBar.test.tsx`, `HubInteractiveModal.test.tsx`, `HubModal.test.tsx`, `HubPanel.test.tsx`, `SessionCard.test.tsx`, `SessionCardGrid.test.tsx`, `useChatUnreadListeners.test.tsx`, `frontend/src/lib/hubGroupCounts.test.ts`, `frontend/src/lib/remoteAdapter.ts`
- **Verification:** `cd frontend && pnpm exec tsc --noEmit` clean; `cd frontend && pnpm test -- --run` — 142 test files / 2337 tests pass.
- **Committed in:** `af53dd7c` (Task 1 commit)

**3. [Rule 2 - Missing Critical] Documentation: TESTING.md FNL-09 traceability (this plan + 171-01 backfill)**
- **Found during:** Overall verification (post-Task-3, before Summary)
- **Issue:** This repo's `CLAUDE.md` standing convention requires every phase that adds/extends tests to update `TESTING.md`'s Suite Manifest note and Requirement→Test Traceability Map. This plan extended `internal/daemon/funnel_test.go` with 5 new tests but had not yet updated `TESTING.md`. Additionally, 171-01 (the prior wave this plan builds directly on) added a new test file (`internal/webserver/rwgate_test.go`) and extended two others for the SAME requirement (FNL-09) without any `TESTING.md` update — leaving FNL-09 completely absent from the traceability table.
- **Fix:** Added a Section 2 Suite Manifest note for this plan's test additions (counts unchanged — no new files) and Section 4 traceability rows for FNL-09 covering both this plan's tests and 171-01's (joincode_test.go, rwgate_test.go, funnel_test.go's updated origin test).
- **Files modified:** `TESTING.md`
- **Verification:** `bash tests/check-traceability-paths.sh` exits 0 ("OK: all traceability paths exist").
- **Committed in:** `7cb2f936` (separate docs commit, per the standing convention's own guidance to track this as its own change)

---

**Total deviations:** 3 auto-fixed (2 Rule 2 - missing critical functionality/documentation, 1 Rule 1 - bug/build-breakage fix)
**Impact on plan:** All three are directly required for the plan's own stated build/test gates (Go build, frontend `tsc --noEmit`, this repo's standing TESTING.md convention) to actually pass, or for the shipped RPC surface to function at runtime. No scope creep beyond what was necessary to satisfy those gates.

## Issues Encountered

- `probeGrant` (an existing `api_test.go` helper) 404s unless `ws.SetSessionResolver` is set — the shared `makeFunnelTestWebServer` test helper doesn't set one by default. Resolved by following the exact precedent already established in this same file (`TestIssueCapabilities_BrowseToggleRebindsPublicCode`): call `ws.SetSessionResolver(...)` inline in each new test right after `makeFunnelTestWebServer` returns, before using `probeGrant`. Not a deviation — this is how the existing test suite already handles it.
- One pre-existing, unrelated test flake observed during a single `go test -race ./internal/daemon/...` full-package run: `TestExitEvent_ListSessions_ExitCodePopulatedForStopped` failed once, then passed 3/3 in isolation and on a full-package re-run. Confirmed unrelated (the file was not touched, and the failure mode — a timing-sensitive process-exit-code race — is orthogonal to the RW-gate logic this plan modifies). Not fixed (out of scope per the Scope Boundary rule).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The daemon-side RW-gate lifecycle (mint, gate, single-use code, unconditional 1h expiry clamp, surgical teardown across all four triggers, D-04 fix) is complete and proven at the real HTTP boundary — `go test -race ./internal/daemon/...` green, full repo `go test -race -short ./...` green, `go build ./...` clean, frontend `tsc --noEmit` clean, and `pnpm test -- --run` (142 files / 2337 tests) green.
- `DaemonClient`/`App` RPC surface and the full Wails TS binding trio (`App.d.ts`/`App.js`/`models.ts`) are ready for 171-03 (frontend Share modal RW toggle UI) and 171-04 to consume.
- No blockers for 171-03.

---
*Phase: 171-public-full-access-read-write-internet-sharing-behind-a-hard*
*Completed: 2026-07-07*

## Self-Check: PASSED

All created/modified files found on disk; all 4 task/summary commit hashes (af53dd7c, 7e34a8fe, 26066de7, 7cb2f936) found in git log.
